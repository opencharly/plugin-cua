// The `cua` plugin's OWN CUE schema — the typed plugin_input for the `cua`
// computer-use check/control verb AND the `kind: cua` entity body.
//
// It is the SINGLE SOURCE for this plugin's params, used two ways (the same contract
// core `spec` and every other plugin use):
//
//  1. GENERATE the Go param struct — `cue exp gengotypes` (driven by the cue:gen
//     pipeline) emits ../params/cue_types_gen.go, so the provider decodes plugin_input
//     into a TYPED struct, never a hand-parsed map.
//  2. VALIDATE authored input AT RUNTIME — the plugin serves this source over the
//     Describe channel; the host splices it onto the base (base ++ plugin) and
//     validates every authored `cua:` step's plugin_input against #CuaInput.
//
// An authored `cua: <method>` step (scalar sugar) or `cua: {method: …, …}` (map form)
// desugars to the INTERNAL plugin/plugin_input envelope, and every cua-exclusive
// modifier lives HERE. The shared assertion matchers (exit_status/stdout/stderr) and
// the general `timeout` stay on core #Op, read off the step Op by the provider.
//
// SELF-CONTAINED: it references NO base def, so it compiles standalone (gengotypes +
// the load-gate compile) AND splices onto the base (base ++ plugin is a def-name
// collision check, not a base-reference resolver).
#CuaInput: {
	// method — the cua method to dispatch (also the scalar-sugar primary:
	// `cua: <method>`).
	method: #CuaMethod

	// --- connection -------------------------------------------------------
	// host — the deployment/venue address hosting the Cua Driver. Omitted means
	// "the deploy I am running against": the plugin drives it over the host
	// reverse channel (the venue executor), which is the normal case.
	host?: string & !=""
	// driver — the `cua-driver` binary path inside the venue (default
	// /usr/local/bin/cua-driver).
	driver?: string @go(Driver)
	// device — the name of a `kind: cua` DEVICE entity whose connection/permission
	// defaults this step inherits. Authored step fields WIN over the entity's.
	device?: string @go(Device)
	// env — extra session environment the driver needs inside the venue (the
	// desktop-session variables). Merged over the entity's env.
	env?: {[string]: string}
	// allow_control — REQUIRED to be true for every MUTATING method (input,
	// app control, config writes). Read-only methods ignore it. This is the
	// bed-safety gate: a live desktop is not a throwaway, and an unattended plan
	// must not type into it by accident.
	allow_control?: bool @go(AllowControl)

	// --- input (keyboard/pointer) ----------------------------------------
	// text — the text `type` inserts.
	text?: string
	// key — the named key `press-key` presses (return, escape, tab, f5, …).
	key?: string @go(KeyName)
	// keys — the chord `hotkey` presses, e.g. ["ctrl","c"].
	keys?: [...string]
	// x / y — desktop-absolute pixel coordinates for click/move/scroll/drag.
	x?: int @go(,type=int)
	y?: int @go(,type=int)
	// from_x / from_y — drag start coordinates.
	from_x?: int @go(FromX,type=int)
	from_y?: int @go(FromY,type=int)
	// button — pointer button (left/right/middle; default left).
	button?: string
	// direction / amount — scroll direction (up/down/left/right) and notches.
	direction?: string
	amount?:    int @go(,type=int)
	// pid / window_id — target a specific window for element/typed actions.
	pid?:       int @go(PID,type=int)
	window_id?: int @go(WindowID,type=int)
	// element_index — the AX element index from a prior get-window-state.
	element_index?: int @go(ElementIndex,type=int)
	// delivery — the Cua delivery mode for an input action: background (default)
	// or foreground. Foreground briefly fronts the window; it is the documented
	// escalation when a background action is refused or unverifiable.
	delivery?: "background" | "foreground"

	// --- app / window -----------------------------------------------------
	// app — the app name/bundle/launch-path for launch-app.
	app?: string
	// window_id / value — the window a set-window-frame targets, and its value.
	value?: string
	// width / height — rectangle size for set-window-frame.
	width?:  int @go(,type=int)
	height?: int @go(,type=int)

	// --- raw escape hatch -------------------------------------------------
	// tool — the Cua Driver tool `call` invokes (any tool the plugin has not typed
	// yet). Mutating by nature, so `call` requires allow_control.
	tool?: string
	// args — the JSON object of arguments for `call`.
	args?: string

	// --- artifact ---------------------------------------------------------
	// artifact — the host path `screenshot` writes the PNG to.
	artifact?: string
	// artifact_dir — the runner-injected generic evidence-artifact dir.
	artifact_dir?: string @go(ArtifactDir)

	// artifact_min_bytes / artifact_min_dimensions / artifact_not_uniform /
	// artifact_contains_text — the post-run artifact-reality assertions
	// (sdk.RunArtifactValidators) this verb can produce. artifact_contains_text is
	// the OCR wait-for-screen primitive in `screenshot` form (tesseract on the HOST).
	artifact_min_bytes?:      int & >=0                    @go(ArtifactMinBytes,type=int)
	artifact_min_dimensions?: string & =~"^[0-9]+x[0-9]+$" @go(ArtifactMinDimensions)
	artifact_not_uniform?:    bool                         @go(ArtifactNotUniform)
	artifact_contains_text?:  string                       @go(ArtifactContainsText)
}

// #CuaMethod — the method catalog, grouped by intent so the read-only surface is
// separable from the mutating one (the allow_control gate).
//
// NOTE: written as a bare trailing-pipe disjunction, NOT `(...)`-wrapped: CUE does
// not terminate a parenthesised expression at a newline, so the grouped form requires
// commas and breaks `cue exp gengotypes`.
#CuaMethod: string &
	// observation (read-only)
	"status" |
	"doctor" |
	"version" |
	"list-tools" |
	"list-apps" |
	"list-windows" |
	"get-window-state" |
	"get-desktop-state" |
	"get-screen-size" |
	"screenshot" |
	// input (mutating)
	"click" |
	"double-click" |
	"right-click" |
	"drag" |
	"type" |
	"press-key" |
	"hotkey" |
	"scroll" |
	"move-cursor" |
	// app / window control (mutating)
	"launch-app" |
	"kill-app" |
	"bring-to-front" |
	"set-window-frame" |
	"set-value" |
	// recording
	"recording-start" |
	"recording-stop" |
	"recording-status" |
	"recording-render" |
	// raw escape hatch (mutating)
	"call"

// #CuaDeviceInput — the authored `kind: cua` DEVICE entity body: the connection and
// permission defaults a `cua:` step inherits when it names the entity with `device:`.
// One place for the driver path, the permission mode, and a default session env, so a
// bed states them once.
#CuaDeviceInput: {
	// description — human label for the deployment/device.
	description?: string @go(Description)
	// driver — the `cua-driver` binary path inside the venue.
	driver?: string @go(Driver)
	// host — the deployment/venue address (omit for "the deploy I run against").
	host?: string & !=""
	// permission_mode — Cua Driver's own authorization mode. `standard` is the
	// promptless default; `bounded` admits only a reviewed manifest; `unrestricted`
	// requires an explicit danger acknowledgement. The plugin passes the mode to the
	// driver's launch, never overrides a running daemon's policy.
	permission_mode?: "standard" | "bounded" | "unrestricted" @go(PermissionMode)
	// capability_manifest — the reviewed manifest path for `bounded` mode.
	capability_manifest?: string @go(CapabilityManifest)
	// env — extra session environment the driver needs inside the venue (the
	// desktop-session variables: XDG_RUNTIME_DIR, WAYLAND_DISPLAY,
	// HYPRLAND_INSTANCE_SIGNATURE, DBUS_SESSION_BUS_ADDRESS, YDOTOOL_SOCKET).
	env?: {[string]: string}
}
