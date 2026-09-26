package cua

// client.go — the `cua-driver` client: thin wrappers over the CLI the plugin runs in
// the venue over the executor reverse channel.
//
// Cua Driver's modern interface is MCP-over-stdio AND a CLI (`cua-driver call <tool>
// '<json-args>'`, `cua-driver status`, `cua-driver list-tools`, …). The plugin uses the
// CLI, not the MCP transport: the CLI is one process per call with no long-lived client
// state, which is exactly what an out-of-process check step wants. (The driver's HTTP
// endpoint remains on legacy MCP; stdio is the modern path, and the CLI wraps stdio.)
//
// EVERY invocation is routed through the venue executor's RunCapture, so the driver is
// driven WHERE IT LIVES (the desktop VM/pod), never on the host running charly.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/opencharly/plugin-cua/candy/plugin-cua/params"
	"github.com/opencharly/sdk"
)

// defaultDriverPath is where the layer-cua driver candy installs the wrapper.
const defaultDriverPath = "/usr/local/bin/cua-driver"

// driverPath resolves the driver binary: authored step > entity default > built-in.
func driverPath(in *params.CuaInput) string {
	if in.Driver != "" {
		return in.Driver
	}
	return defaultDriverPath
}

// sessionEnvPrefix builds the `export K=V; …` prefix that places the driver in the
// desktop session. The desktop-session variables are load-bearing: without
// WAYLAND_DISPLAY / XDG_RUNTIME_DIR / DBUS_SESSION_BUS_ADDRESS the driver cannot reach
// the compositor or the accessibility bus, and without HYPRLAND_INSTANCE_SIGNATURE the
// Hyprland routes refuse. Values are shell-quoted; the entity's env map (and any
// authored env) supplies them, and the layer's own session units set the rest.
func sessionEnvPrefix(in *params.CuaInput, env cuaEnv) string {
	merged := map[string]string{}
	for k, v := range env.Env {
		merged[k] = v
	}
	for k, v := range in.Env {
		merged[k] = v
	}
	if len(merged) == 0 {
		return ""
	}
	var b strings.Builder
	for k, v := range merged {
		fmt.Fprintf(&b, "export %s=%s; ", k, shellSingleQuote(v))
	}
	return b.String()
}

// callTool runs `cua-driver call <tool> '<json>'` and returns stdout. argsJSON may be
// empty for a zero-argument tool.
func callTool(ctx context.Context, ex *sdk.Executor, in *params.CuaInput, env cuaEnv, tool, argsJSON string) (string, error) {
	cmd := sessionEnvPrefix(in, env) + driverPath(in) + " call " + shellSingleQuote(tool)
	if strings.TrimSpace(argsJSON) != "" {
		cmd += " " + shellSingleQuote(argsJSON)
	}
	stdout, stderr, code, err := ex.RunCapture(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("cua-driver call %s: %w", tool, err)
	}
	if code != 0 {
		return "", fmt.Errorf("cua-driver call %s: exit %d: %s", tool, code, sdk.Preview(stderr))
	}
	return stdout, nil
}

// runDriver runs a bare `cua-driver <args>` and returns stdout.
func runDriver(ctx context.Context, ex *sdk.Executor, in *params.CuaInput, env cuaEnv, args string) (string, error) {
	cmd := sessionEnvPrefix(in, env) + driverPath(in) + " " + args
	stdout, stderr, code, err := ex.RunCapture(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("cua-driver %s: %w", args, err)
	}
	if code != 0 {
		return "", fmt.Errorf("cua-driver %s: exit %d: %s", args, code, sdk.Preview(stderr))
	}
	return stdout, nil
}

// jsonArgs marshals an argument map, dropping nil so a zero-arg tool sends nothing.
func jsonArgs(m map[string]any) string {
	if len(m) == 0 {
		return ""
	}
	// strip nil values
	clean := map[string]any{}
	for k, v := range m {
		if v != nil {
			clean[k] = v
		}
	}
	if len(clean) == 0 {
		return ""
	}
	b, _ := json.Marshal(clean)
	return string(b)
}

// shellSingleQuote wraps s in single quotes for safe shell interpolation.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
