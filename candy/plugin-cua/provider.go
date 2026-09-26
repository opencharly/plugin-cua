package cua

// provider.go is the out-of-process cua verb+kind provider — charly's host dispatches
// a `cua:` check step to it through the registry (ResolveVerb("cua") -> this
// grpcProvider -> Provider.Invoke) with the FULL #Op marshaled as params_json and a
// CheckEnv snapshot as env.
//
// Because the out-of-process path runs NO host-side matcher pipeline, this Invoke OWNS
// the whole verdict: it decodes the typed input, resolves the venue executor (the live
// DeployExecutor over the E3b reverse channel — the plugin-record/plugin-wl/plugin-jetkvm
// precedent), runs the method, and evaluates the stdout/stderr/exit_status matchers and
// the artifact validators itself — via the SHARED sdk implementation (R3), never a copy.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opencharly/plugin-cua/candy/plugin-cua/params"
	"github.com/opencharly/sdk"
	"github.com/opencharly/sdk/kit"
	pb "github.com/opencharly/spec/proto"
	"github.com/opencharly/spec/spec"
)

// cuaEnv is the plugin-side decode of the CheckEnv the host ships as Operation.Env for a
// `cua:` check step. Mode distinguishes a live deployment from `charly check box` (where
// no desktop/driver exists).
type cuaEnv struct {
	Box  string            `json:"box"`
	Mode string            `json:"mode"` // "live" | "box"
	Host string            `json:"host"`
	Env  map[string]string `json:"env"`
}

type provider struct{ pb.UnimplementedProviderServer }

// Invoke runs one `cua:` operation (or the OpLoad of a `kind: cua` entity).
func (provider) Invoke(ctx context.Context, req *pb.InvokeRequest) (*pb.InvokeReply, error) {
	// OpLoad is the KIND leg: the host validates the authored `kind: cua` entity against
	// #CuaDeviceInput, then dispatches OpLoad with the raw canonical body.
	if req.GetOp() == sdk.OpLoad {
		out, err := deviceCanonicalJSON(req.GetParamsJson())
		if err != nil {
			return nil, err
		}
		return &pb.InvokeReply{ResultJson: out}, nil
	}

	var op spec.Op
	if len(req.GetParamsJson()) > 0 {
		if err := json.Unmarshal(req.GetParamsJson(), &op); err != nil {
			return sdk.ResultJSON("fail", "cua: decode op: "+err.Error())
		}
	}
	var in params.CuaInput
	kit.DecodeInput(op.PluginInput, &in)
	var env cuaEnv
	if len(req.GetEnvJson()) > 0 {
		_ = json.Unmarshal(req.GetEnvJson(), &env)
	}
	method := string(in.Method)

	// A cua method needs a live desktop. `charly check box` runs in a disposable
	// container with no session — mirror every other live-verb plugin's box-mode skip so
	// a plan carrying `cua:` steps still passes the build check.
	if env.Mode == "box" {
		return sdk.ResultJSON("skip", fmt.Sprintf("cua: %s requires a live desktop session (skip under charly check box)", method))
	}

	// Resolve the venue executor over the reverse channel; apply the `kind: cua` entity
	// defaults when the step names one (authored step fields win).
	ex, exErr := sdk.ExecutorFromInvoke(req.GetExecutorBrokerId())
	if exErr != nil || ex == nil {
		return sdk.ResultJSON("fail", fmt.Sprintf("cua: %s needs a live deployment executor (run it inside a deploy/check step): %v", method, exErr))
	}
	if in.Device != "" {
		if err := applyDeviceEntity(ctx, ex, req.GetExecutorBrokerId(), &in); err != nil {
			return sdk.ResultJSON("fail", err.Error())
		}
	}

	// The mutating-method safety gate: a live desktop is not a throwaway, so every
	// mutating method requires allow_control, reporting the documented skip otherwise.
	if isMutating(method) && !in.AllowControl {
		return sdk.ResultJSON("skip", fmt.Sprintf("cua: %s mutates a live desktop; set allow_control: true to authorize it", method))
	}

	out, runErr := dispatch(ctx, ex, &in, env, &op)
	// The shared exit/stdout/stderr verdict pipeline (R3); screenshot is the
	// artifact-producing method.
	return sdk.VerbVerdict("cua", method, out, runErr, &op, method == "screenshot" || method == "get-desktop-state" || method == "get-window-state")
}
