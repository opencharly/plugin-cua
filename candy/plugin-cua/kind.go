package cua

// kind.go implements the plugin's `kind: cua` entity: its OpLoad decode, and the
// resolution of a named entity's connection/permission defaults over the reverse channel.
//
// WHY THE VERB, NOT A SEPARATE COMMAND, RESOLVES THE ENTITY (RDD, measured elsewhere in
// the plugin corpus): a `check:`/`run:` plugin-verb step carries a CheckEnv + a
// reverse-channel broker id, so the out-of-process provider can self-load the project
// (`loaderkit.LoadUnifiedViaExecutor`) and read `uf.PluginKinds["cua"]`.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opencharly/plugin-cua/candy/plugin-cua/params"
	"github.com/opencharly/sdk"
	"github.com/opencharly/sdk/loaderkit"
	"github.com/opencharly/spec/spec"
)

// kindWord is the `kind:` discriminator this plugin serves.
const kindWord = "cua"

// projectDir resolves the project directory over the reverse channel — the canonical
// "deploy-plugins-connect" host seam (os.Getwd host-side).
func projectDir(ctx context.Context, ex *sdk.Executor, name string) (string, error) {
	reqJSON, err := json.Marshal(spec.DeployPluginsConnectRequest{Path: name})
	if err != nil {
		return "", err
	}
	out, err := ex.HostBuild(ctx, "deploy-plugins-connect", reqJSON)
	if err != nil {
		return "", err
	}
	var reply spec.DeployPluginsConnectReply
	if err := json.Unmarshal(out, &reply); err != nil {
		return "", fmt.Errorf("cua: decode deploy-plugins-connect reply: %w", err)
	}
	return reply.Dir, nil
}

// resolveDeviceEntity loads the project out-of-process and returns the decoded
// `kind: cua` entity named name. An absent entity is a clear error, never a zero value.
func resolveDeviceEntity(ctx context.Context, ex *sdk.Executor, name string) (*params.CuaDeviceInput, error) {
	if ex == nil {
		return nil, fmt.Errorf("cua: resolving device entity %q needs a host reverse channel (run it inside a deploy/check step)", name)
	}
	dir, err := projectDir(ctx, ex, name)
	if err != nil {
		return nil, fmt.Errorf("cua: resolving device entity %q: %w", name, err)
	}
	uf, ok, err := loaderkit.LoadUnifiedViaExecutor(ctx, ex, dir)
	if err != nil {
		return nil, fmt.Errorf("cua: loading project for device entity %q: %w", name, err)
	}
	if !ok || uf == nil {
		return nil, fmt.Errorf("cua: resolving device entity %q: no charly.yml loaded from %s", name, dir)
	}
	body, found := loaderkit.ResolveKindEntityBody(uf, kindWord, name)
	if !found {
		return nil, fmt.Errorf("cua: no kind:cua entity named %q in %s", name, dir)
	}
	var dev params.CuaDeviceInput
	if err := json.Unmarshal(body, &dev); err != nil {
		return nil, fmt.Errorf("cua: decoding kind:cua entity %q: %w", name, err)
	}
	return &dev, nil
}

// applyDeviceEntity fills a step's driver/host/env from a referenced `kind: cua` entity.
// Authored inline fields WIN — the entity is the default, so a bed can override one
// field without restating the whole device.
func applyDeviceEntity(ctx context.Context, ex *sdk.Executor, brokerID uint32, in *params.CuaInput) error {
	if in.Device == "" {
		return nil
	}
	dev, err := resolveDeviceEntity(ctx, ex, in.Device)
	if err != nil {
		return err
	}
	applyDeviceDefaults(in, dev)
	return nil
}

// applyDeviceDefaults merges an entity's defaults into `in`, authored step fields
// winning. It is PURE so the precedence contract is unit-testable without a reverse
// channel — the test calls THIS function, not a re-implementation.
func applyDeviceDefaults(in *params.CuaInput, dev *params.CuaDeviceInput) {
	if in.Driver == "" {
		in.Driver = dev.Driver
	}
	if in.Host == "" {
		in.Host = dev.Host
	}
	// Env merges: the entity's env is the base, the step's env wins per key.
	if len(dev.Env) > 0 {
		merged := map[string]string{}
		for k, v := range dev.Env {
			merged[k] = v
		}
		for k, v := range in.Env {
			merged[k] = v
		}
		in.Env = merged
	}
}

// deviceCanonicalJSON is the OpLoad body decode: the host validates the authored entity
// against #CuaDeviceInput, then this re-marshals it canonically.
func deviceCanonicalJSON(paramsJSON []byte) (json.RawMessage, error) {
	var dev params.CuaDeviceInput
	if len(paramsJSON) > 0 {
		if err := json.Unmarshal(paramsJSON, &dev); err != nil {
			return nil, fmt.Errorf("cua: decode device entity: %w", err)
		}
	}
	out, err := json.Marshal(dev)
	if err != nil {
		return nil, fmt.Errorf("cua: marshal device entity: %w", err)
	}
	return out, nil
}
