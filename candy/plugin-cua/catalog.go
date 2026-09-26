package cua

// catalog.go — the method catalog: one thin, typed wrapper per authored `cua:` method
// over the `cua-driver` CLI. Every method here is either observation of the live
// desktop or an input/app action; the mutating ones additionally pass the
// allow_control gate in provider.go.

import (
	"context"
	"fmt"
	"strings"

	"github.com/opencharly/plugin-cua/candy/plugin-cua/params"
	"github.com/opencharly/sdk"
	"github.com/opencharly/spec/spec"
)

// mutatingMethods is the allowlist of methods that change desktop state. It is an
// ALLOWLIST so a method added later is read-only-by-default and cannot actuate a
// desktop until it is deliberately classified — the same shape plugin-jetkvm uses.
var mutatingMethods = map[string]bool{
	"click":            true,
	"double-click":     true,
	"right-click":      true,
	"drag":             true,
	"type":             true,
	"press-key":        true,
	"hotkey":           true,
	"scroll":           true,
	"move-cursor":      true,
	"launch-app":       true,
	"kill-app":         true,
	"bring-to-front":   true,
	"set-window-frame": true,
	"set-value":        true,
	"recording-start":  true,
	"recording-stop":   true,
	"recording-render": true,
	"call":             true,
}

// isMutating reports whether a method requires allow_control.
func isMutating(method string) bool { return mutatingMethods[method] }

// deliveryMode returns the authored delivery mode, or "" (the driver default). The
// value is passed through to the input tools' `delivery_mode` field — foreground is the
// documented escalation when a background action is refused.
func deliveryMode(in *params.CuaInput) any {
	if in.Delivery == "" {
		return nil
	}
	return in.Delivery
}

// dispatch routes one method to its cua-driver invocation.
func dispatch(ctx context.Context, ex *sdk.Executor, in *params.CuaInput, env cuaEnv, op *spec.Op) (string, error) {
	method := string(in.Method)

	switch method {
	// --- observation ------------------------------------------------------
	case "status":
		return runDriver(ctx, ex, in, env, "status")
	case "doctor":
		return runDriver(ctx, ex, in, env, "doctor")
	case "version":
		return runDriver(ctx, ex, in, env, "version")
	case "list-tools":
		return runDriver(ctx, ex, in, env, "list-tools")
	case "list-apps":
		return callTool(ctx, ex, in, env, "list_apps", "")
	case "list-windows":
		return callTool(ctx, ex, in, env, "list_windows", jsonArgs(map[string]any{
			"pid": in.PID,
		}))
	case "get-window-state":
		if in.PID == 0 || in.WindowID == 0 {
			return "", fmt.Errorf("cua: get-window-state requires pid and window_id")
		}
		return callTool(ctx, ex, in, env, "get_window_state", jsonArgs(map[string]any{
			"pid": in.PID, "window_id": in.WindowID,
		}))
	case "get-desktop-state":
		return screenshotCall(ctx, ex, in, env, op, "get_desktop_state", in.Artifact)
	case "screenshot":
		// The portable screenshot tool: the desktop capture. Cua's get_desktop_state
		// is the desktop-scope capture; use it for a full-frame screenshot.
		return screenshotCall(ctx, ex, in, env, op, "get_desktop_state", in.Artifact)
	case "get-screen-size":
		return callTool(ctx, ex, in, env, "get_screen_size", "")

	// --- input (mutating) -------------------------------------------------
	case "click", "double-click", "right-click":
		tool := map[string]string{"click": "click", "double-click": "double_click", "right-click": "right_click"}[method]
		return callTool(ctx, ex, in, env, tool, jsonArgs(map[string]any{
			"pid": in.PID, "window_id": in.WindowID, "x": in.X, "y": in.Y,
			"button": in.Button, "delivery_mode": deliveryMode(in),
		}))
	case "drag":
		return callTool(ctx, ex, in, env, "drag", jsonArgs(map[string]any{
			"pid": in.PID, "window_id": in.WindowID,
			"from_x": in.FromX, "from_y": in.FromY, "to_x": in.X, "to_y": in.Y,
			"button": in.Button, "delivery_mode": deliveryMode(in),
		}))
	case "type":
		if in.Text == "" {
			return "", fmt.Errorf("cua: type requires text")
		}
		return callTool(ctx, ex, in, env, "type_text", jsonArgs(map[string]any{
			"pid": in.PID, "window_id": in.WindowID, "text": in.Text,
			"delivery_mode": deliveryMode(in),
		}))
	case "press-key":
		if in.KeyName == "" {
			return "", fmt.Errorf("cua: press-key requires key")
		}
		return callTool(ctx, ex, in, env, "press_key", jsonArgs(map[string]any{
			"pid": in.PID, "window_id": in.WindowID, "key": in.KeyName,
			"delivery_mode": deliveryMode(in),
		}))
	case "hotkey":
		if len(in.Keys) == 0 {
			return "", fmt.Errorf("cua: hotkey requires keys (e.g. [ctrl, c])")
		}
		return callTool(ctx, ex, in, env, "hotkey", jsonArgs(map[string]any{
			"pid": in.PID, "window_id": in.WindowID, "keys": in.Keys,
			"delivery_mode": deliveryMode(in),
		}))
	case "scroll":
		return callTool(ctx, ex, in, env, "scroll", jsonArgs(map[string]any{
			"pid": in.PID, "window_id": in.WindowID, "x": in.X, "y": in.Y,
			"direction": in.Direction, "amount": in.Amount,
			"delivery_mode": deliveryMode(in),
		}))
	case "move-cursor":
		return callTool(ctx, ex, in, env, "move_cursor", jsonArgs(map[string]any{
			"pid": in.PID, "window_id": in.WindowID, "x": in.X, "y": in.Y,
		}))

	// --- app / window control (mutating) ----------------------------------
	case "launch-app":
		if in.App == "" {
			return "", fmt.Errorf("cua: launch-app requires app")
		}
		return callTool(ctx, ex, in, env, "launch_app", jsonArgs(map[string]any{"name": in.App}))
	case "kill-app":
		if in.PID == 0 {
			return "", fmt.Errorf("cua: kill-app requires pid")
		}
		return callTool(ctx, ex, in, env, "kill_app", jsonArgs(map[string]any{"pid": in.PID}))
	case "bring-to-front":
		return callTool(ctx, ex, in, env, "bring_to_front", jsonArgs(map[string]any{
			"pid": in.PID, "window_id": in.WindowID,
		}))
	case "set-window-frame":
		if in.PID == 0 || in.WindowID == 0 {
			return "", fmt.Errorf("cua: set-window-frame requires pid and window_id")
		}
		return callTool(ctx, ex, in, env, "set_window_frame", jsonArgs(map[string]any{
			"pid": in.PID, "window_id": in.WindowID,
			"x": in.X, "y": in.Y, "width": in.Width, "height": in.Height,
		}))
	case "set-value":
		if in.Value == "" {
			return "", fmt.Errorf("cua: set-value requires value")
		}
		return callTool(ctx, ex, in, env, "set_value", jsonArgs(map[string]any{
			"pid": in.PID, "window_id": in.WindowID, "value": in.Value,
			"element_index": in.ElementIndex,
		}))

	// --- recording --------------------------------------------------------
	case "recording-start":
		return runDriver(ctx, ex, in, env, "recording start "+shellSingleQuote(in.Artifact))
	case "recording-stop":
		return runDriver(ctx, ex, in, env, "recording stop")
	case "recording-status":
		return runDriver(ctx, ex, in, env, "recording status")
	case "recording-render":
		if in.Artifact == "" {
			return "", fmt.Errorf("cua: recording-render requires artifact (output mp4 path)")
		}
		return runDriver(ctx, ex, in, env, "recording render "+shellSingleQuote(in.Value)+" "+shellSingleQuote(in.Artifact))

	// --- raw escape hatch (mutating) --------------------------------------
	case "call":
		if strings.TrimSpace(in.Tool) == "" {
			return "", fmt.Errorf("cua: call requires tool")
		}
		return callTool(ctx, ex, in, env, in.Tool, in.Args)
	}
	return "", fmt.Errorf("cua: unknown method %q", method)
}

// screenshotCall runs a screenshot-producing tool with the artifact written INSIDE the
// venue, then pulls the PNG back to the host artifact path via sdk.LandArtifact — the
// shared pull+validate entry point every capture plugin uses (R3): the venue file is
// GetFile-pulled over the reverse channel to the host artifact path, and the op's
// artifact validators (artifact_min_bytes / artifact_contains_text / …) run there.
func screenshotCall(ctx context.Context, ex *sdk.Executor, in *params.CuaInput, env cuaEnv, op *spec.Op, tool, hostArtifact string) (string, error) {
	venuePath := "/tmp/charly-cua-shot.png"
	out, err := callTool(ctx, ex, in, env, tool, jsonArgs(map[string]any{"screenshot_out_file": venuePath}))
	if err != nil {
		return "", err
	}
	if hostArtifact != "" {
		if err := sdk.LandArtifact(ctx, ex, venuePath, hostArtifact, op); err != nil {
			return "", fmt.Errorf("cua: pulling screenshot artifact %s: %w", hostArtifact, err)
		}
	}
	return out, nil
}
