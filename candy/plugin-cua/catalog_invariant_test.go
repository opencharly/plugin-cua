package cua

// catalog_invariant_test.go pins the invariant the jetkvm sibling keeps and this
// plugin needs for the same reason: every method the SCHEMA advertises must be
// reachable — dispatched by dispatch() — and every method dispatch() routes must
// be in the schema. Without it a method can sit in the catalog as a phantom
// (advertised but unimplemented): the schema validates the authored step, and the
// step then fails only at dispatch with `unknown method`, so the catalog and the
// skill that documents it actively mislead the agent that dispatches on it.
//
// The tests read the REAL sources — the SERVED #CuaMethod enum (through the CUE
// API, schema_served_test.go) and catalog.go's dispatch switch — rather than
// restating either, so they cannot drift alongside the code they guard.

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// schemaMethods returns the sorted method names the SERVED #CuaMethod enum
// accepts. It reads the schema through the CUE API rather than its text: the
// enum is now §10.1 S3 (a vocabulary table plus a derived word list), and a text
// parse of the union would find ZERO methods and leave every invariant below
// passing vacuously.
func schemaMethods(t *testing.T) []string {
	t.Helper()
	return schemaWordList(t, servedSchema(t), "#CuaMethod")
}

// dispatchedMethods returns the sorted method names dispatch() routes. Cases may
// be grouped (`case "a", "b":`), so each case line is split on commas.
func dispatchedMethods(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile("catalog.go")
	if err != nil {
		t.Fatalf("read catalog: %v", err)
	}
	fn := regexp.MustCompile(`(?s)func dispatch\(.*?\n\}`).FindString(string(b))
	if fn == "" {
		t.Fatal("could not locate dispatch in catalog.go")
	}
	seen := map[string]bool{}
	for _, m := range regexp.MustCompile(`case ([^:]+):`).FindAllStringSubmatch(fn, -1) {
		for _, part := range strings.Split(m[1], ",") {
			name := strings.Trim(strings.TrimSpace(part), `"`)
			if name != "" {
				seen[name] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// TestServedCatalogCoversTheWholeEnum guards the READER and the served catalog in
// one step: the enum's first and last entries must be present, so neither a
// truncated read nor a table whose tail never reaches the enum can exempt the
// catalog. `status` is the first method and `call` the last.
func TestServedCatalogCoversTheWholeEnum(t *testing.T) {
	got := schemaMethods(t)
	has := func(name string) bool {
		for _, m := range got {
			if m == name {
				return true
			}
		}
		return false
	}
	if !has("status") {
		t.Errorf("served enum is missing its HEAD (status) — got %d methods: %v", len(got), got)
	}
	if !has("call") {
		t.Errorf("served enum is missing its TAIL (call) — the derivation is losing rows; got %d methods: %v", len(got), got)
	}
	if len(got) < 28 {
		t.Errorf("read only %d methods — the served #CuaMethod derivation looks truncated", len(got))
	}
}

// TestCatalogHasNoPhantomMethods asserts every schema method is dispatched — no
// method is advertised that the plugin cannot act on.
func TestCatalogHasNoPhantomMethods(t *testing.T) {
	reachable := map[string]bool{}
	for _, m := range dispatchedMethods(t) {
		reachable[m] = true
	}
	for _, m := range schemaMethods(t) {
		if !reachable[m] {
			t.Errorf("schema method %q is a PHANTOM: not routed by dispatch — remove it from the schema or implement it", m)
		}
	}
}

// TestDispatchHasNoPhantomArms is the other direction: a `case` arm for a word the
// schema no longer accepts is dead code, and it hides a rename that only half
// landed — the arm would keep an unreachable path compiling.
func TestDispatchHasNoPhantomArms(t *testing.T) {
	advertised := map[string]bool{}
	for _, m := range schemaMethods(t) {
		advertised[m] = true
	}
	for _, m := range dispatchedMethods(t) {
		if !advertised[m] {
			t.Errorf("dispatch routes %q, which the served #CuaMethod enum does NOT accept — the arm is unreachable", m)
		}
	}
}

// TestMutatingAllowlistIsASubsetOfTheCatalog keeps the desktop-safety allowlist
// honest: a word classified MUTATING that the catalog no longer carries can never
// be dispatched, so the classification is stale and would silently stop gating
// anything. The read-only/mutating split ITSELF is pinned by
// TestMutatingMethodsAllowlist (plugin_test.go).
func TestMutatingAllowlistIsASubsetOfTheCatalog(t *testing.T) {
	advertised := map[string]bool{}
	for _, m := range schemaMethods(t) {
		advertised[m] = true
	}
	if len(mutatingMethods) == 0 {
		t.Fatal("mutatingMethods is empty — the allowlist gate would pass over nothing")
	}
	for m := range mutatingMethods {
		if !advertised[m] {
			t.Errorf("%q is classified MUTATING but the served #CuaMethod enum does not accept it — the classification is stale", m)
		}
	}
}
