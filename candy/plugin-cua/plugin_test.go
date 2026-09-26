package cua

// plugin_test.go — the unit gate for the cua plugin's real logic: the mutating-method
// allowlist (the desktop-safety gate), the kind-entity default merge precedence, the
// session-env merge, and the shell/JSON argument rendering. All pure — no live desktop,
// no registry — so they run everywhere. The live driver contract is exercised by the
// check-cua-* beds in distro-omarchy.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/opencharly/plugin-cua/candy/plugin-cua/params"
)

// The mutating allowlist is the safety gate: a method added later must be read-only by
// default. This pins both directions — the classified mutators and the read-only ones.
func TestMutatingMethodsAllowlist(t *testing.T) {
	mutating := []string{
		"click", "double-click", "right-click", "drag", "type", "press-key", "hotkey",
		"scroll", "move-cursor", "launch-app", "kill-app", "bring-to-front",
		"set-window-frame", "set-value", "recording-start", "recording-stop",
		"recording-render", "call",
	}
	readOnly := []string{
		"status", "doctor", "version", "list-tools", "list-apps", "list-windows",
		"get-window-state", "get-desktop-state", "get-screen-size", "screenshot",
		"recording-status",
	}
	for _, m := range mutating {
		if !isMutating(m) {
			t.Errorf("%q must be classified MUTATING (requires allow_control)", m)
		}
	}
	for _, m := range readOnly {
		if isMutating(m) {
			t.Errorf("%q must be read-only (a new method is read-only by default)", m)
		}
	}
	// An unknown method must be read-only (allowlist ⇒ safe default).
	if isMutating("some-future-method") {
		t.Error("an unclassified method must default to read-only")
	}
}

// The entity default merge: authored step fields WIN, and env merges per key with the
// step winning.
func TestApplyDeviceDefaults_AuthoredWins(t *testing.T) {
	dev := &params.CuaDeviceInput{
		Driver: "/opt/cua-driver",
		Host:   "entity-host",
		Env:    map[string]string{"WAYLAND_DISPLAY": "wayland-1", "XDG_RUNTIME_DIR": "/run/user/1000"},
	}
	in := &params.CuaInput{
		Driver: "/custom/cua-driver", // authored: must win
		Env:    map[string]string{"WAYLAND_DISPLAY": "wayland-9"},
	}
	applyDeviceDefaults(in, dev)

	if in.Driver != "/custom/cua-driver" {
		t.Errorf("authored driver must win, got %q", in.Driver)
	}
	if in.Host != "entity-host" {
		t.Errorf("entity host must fill an empty step host, got %q", in.Host)
	}
	if in.Env["WAYLAND_DISPLAY"] != "wayland-9" {
		t.Errorf("authored env value must win, got %q", in.Env["WAYLAND_DISPLAY"])
	}
	if in.Env["XDG_RUNTIME_DIR"] != "/run/user/1000" {
		t.Errorf("entity env key must be inherited, got %q", in.Env["XDG_RUNTIME_DIR"])
	}
}

// jsonArgs drops nil values and empty maps (a zero-arg tool sends nothing).
func TestJSONArgs(t *testing.T) {
	if got := jsonArgs(nil); got != "" {
		t.Errorf("nil map must render empty, got %q", got)
	}
	if got := jsonArgs(map[string]any{"pid": nil, "x": nil}); got != "" {
		t.Errorf("all-nil map must render empty, got %q", got)
	}
	got := jsonArgs(map[string]any{"pid": 42, "x": 10, "y": nil})
	var m map[string]any
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("jsonArgs produced invalid JSON %q: %v", got, err)
	}
	if m["pid"] != float64(42) || m["x"] != float64(10) {
		t.Errorf("jsonArgs lost values: %v", m)
	}
	if _, present := m["y"]; present {
		t.Errorf("jsonArgs must drop nil keys: %v", m)
	}
}

// shellSingleQuote must make an arbitrary value safe to interpolate.
func TestShellSingleQuote(t *testing.T) {
	cases := map[string]string{
		"hello":       "'hello'",
		"it's":        `'it'\''s'`,
		"a b":         "'a b'",
		`{"x":"y"} `:  `'{"x":"y"} '`,
		"$(rm -rf /)": "'$(rm -rf /)'",
		"`whoami`":    "'`whoami`'",
	}
	for in, want := range cases {
		if got := shellSingleQuote(in); got != want {
			t.Errorf("shellSingleQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

// The session env prefix exports every merged variable, shell-quoted.
func TestSessionEnvPrefix(t *testing.T) {
	in := &params.CuaInput{Env: map[string]string{"WAYLAND_DISPLAY": "wayland-1"}}
	env := cuaEnv{Env: map[string]string{"XDG_RUNTIME_DIR": "/run/user/1000"}}
	p := sessionEnvPrefix(in, env)
	if !strings.Contains(p, "export WAYLAND_DISPLAY='wayland-1';") {
		t.Errorf("prefix must export the step env, got %q", p)
	}
	if !strings.Contains(p, "export XDG_RUNTIME_DIR='/run/user/1000';") {
		t.Errorf("prefix must export the venue env, got %q", p)
	}
	// No env at all ⇒ no prefix (the driver inherits the session).
	if got := sessionEnvPrefix(&params.CuaInput{}, cuaEnv{}); got != "" {
		t.Errorf("no env must produce no prefix, got %q", got)
	}
}

// driverPath resolves authored step > built-in default.
func TestDriverPath(t *testing.T) {
	if got := driverPath(&params.CuaInput{}); got != defaultDriverPath {
		t.Errorf("default driver path %q, want %q", got, defaultDriverPath)
	}
	if got := driverPath(&params.CuaInput{Driver: "/x/cua-driver"}); got != "/x/cua-driver" {
		t.Errorf("authored driver path must win, got %q", got)
	}
}

// A method with a missing required modifier must produce a clear error, not a bad call.
func TestDispatchRequiredModifiers(t *testing.T) {
	cases := []struct {
		name string
		in   params.CuaInput
		want string
	}{
		{"type without text", params.CuaInput{Method: "type"}, "requires text"},
		{"press-key without key", params.CuaInput{Method: "press-key"}, "requires key"},
		{"hotkey without keys", params.CuaInput{Method: "hotkey"}, "requires keys"},
		{"get-window-state without pid", params.CuaInput{Method: "get-window-state"}, "requires pid and window_id"},
		{"launch-app without app", params.CuaInput{Method: "launch-app"}, "requires app"},
		{"call without tool", params.CuaInput{Method: "call"}, "requires tool"},
		{"unknown method", params.CuaInput{Method: "nope"}, "unknown method"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// ex is nil, but the required-modifier checks run BEFORE any exec, so we
			// must reach them — assert the error names the missing field.
			_, err := dispatch(t.Context(), nil, &c.in, cuaEnv{}, nil)
			if err == nil {
				t.Fatalf("%s: expected an error", c.name)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("%s: error %q must mention %q", c.name, err.Error(), c.want)
			}
		})
	}
}
