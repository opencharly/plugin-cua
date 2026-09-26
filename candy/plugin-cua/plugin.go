// Package cua is the charly plugin serving the `cua` computer-use check/control verb
// and the `kind: cua` entity — an importable root package + its own go.mod, served
// OUT-OF-PROCESS over go-plugin gRPC via the charly plugin SDK
// (github.com/opencharly/sdk).
//
// The `cua:` verb drives a live desktop through Cua Driver (trycua/cua) — the
// background computer-use driver that speaks MCP over stdio and a CLI (`cua-driver
// call <tool> '<json>'`). charly's loader fetches this candy's repo, go-builds the
// provider binary on the HOST, and connects it via LocalTransport, so the Cua wire
// lives HERE, out of charly's core check surface.
//
// Every method is a thin wrapper over `cua-driver call`; the plugin owns the verdict
// (the shared sdk matcher pipeline + artifact validators), the mutating-method safety
// gate (`allow_control`), and the `kind: cua` entity resolution. It needs no podman /
// venue machinery — the driver runs INSIDE the deployment (a desktop VM or pod), and
// the plugin drives it over the host's DeployExecutor reverse channel exactly like
// plugin-record / plugin-wl / plugin-jetkvm do.
//
// Dual-placement by construction: the SAME NewProvider()/NewMeta() compile INTO charly
// in-process when listed in compiled_plugins, or cmd/serve serves them OUT-OF-PROCESS
// over go-plugin gRPC when they are not — placement is invisible above the registry.
package cua

import (
	"embed"

	"github.com/opencharly/sdk"
	pb "github.com/opencharly/spec/proto"
)

//go:embed schema/*.cue
var schemaFS embed.FS

// NewProvider returns the cua verb+kind provider.
func NewProvider() pb.ProviderServer { return &provider{} }

// NewMeta advertises verb:cua AND kind:cua with the plugin's self-contained CUE schema
// (via sdk.NewMeta → BuildCapabilities). The verb's entire authoring contract — the
// method enum + every cua-exclusive modifier — lives in the served #CuaInput
// (schema/cua.cue), which the host splices onto the base and validates every authored
// `cua:` step's plugin_input against.
func NewMeta() pb.PluginMetaServer {
	return sdk.NewMeta("2026.269.1200",
		[]sdk.ProvidedCapability{
			{Class: "verb", Word: "cua", InputDef: "#CuaInput", Primary: "method"},
			{Class: "kind", Word: "cua", InputDef: "#CuaDeviceInput"},
		},
		schemaFS)
}
