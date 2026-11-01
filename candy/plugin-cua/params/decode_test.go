package params

// decode_test.go — the ONE runtime path these generated structs participate in, pinned against
// the SOURCE OF TRUTH.
//
// charly hands a plugin its parameters as JSON (`spec.Op`'s params_json) and the plugin decodes
// that JSON into this projection; the sdk's kit.DecodeInput re-marshals to JSON before it
// unmarshals too. No YAML decoder ever sees these types — which is why the projection's `yaml:`
// tags are inert, and why this test pins the JSON side rather than asserting the claim in prose.
//
// The expected wire keys below are copied from `candy/plugin-cua/schema/*.cue` — the schema that
// GENERATES this file. They are hardcoded on purpose: deriving them from the struct's own `json`
// tags (the obvious thing to do) makes the test vacuous, because a tag that drifts takes its
// derived expectation with it and the decode still lines up. Anchored to the schema instead, a
// drifted, emptied or removed tag leaves the field unpopulated and fails here.

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// schemaKeys is the wire-key set declared by candy/plugin-cua/schema/*.cue.
var schemaKeys = map[string]bool{
	"allow_control":           true,
	"amount":                  true,
	"app":                     true,
	"args":                    true,
	"artifact":                true,
	"artifact_contains_text":  true,
	"artifact_dir":            true,
	"artifact_min_bytes":      true,
	"artifact_min_dimensions": true,
	"artifact_not_uniform":    true,
	"button":                  true,
	"capability_manifest":     true,
	"delivery":                true,
	"description":             true,
	"device":                  true,
	"direction":               true,
	"driver":                  true,
	"element_index":           true,
	"env":                     true,
	"from_x":                  true,
	"from_y":                  true,
	"height":                  true,
	"host":                    true,
	"key":                     true,
	"keys":                    true,
	"method":                  true,
	"permission_mode":         true,
	"pid":                     true,
	"text":                    true,
	"tool":                    true,
	"value":                   true,
	"width":                   true,
	"window_id":               true,
	"x":                       true,
	"y":                       true,
}

func TestProjectionMatchesSchemaWireKeys(t *testing.T) {
	rt := reflect.TypeOf(CuaInput{})
	ptr := reflect.New(rt)
	wire := map[string]any{}

	exported := 0
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.PkgPath != "" {
			continue // unexported
		}
		exported++
		key := strings.Split(f.Tag.Get("json"), ",")[0]
		if key == "" || key == "-" {
			t.Fatalf("field %s carries no usable json tag (%q) — the schema's key cannot reach it", f.Name, key)
		}
		if !schemaKeys[key] {
			t.Fatalf("field %s is tagged json:%q, which the schema candy/plugin-cua/schema does NOT declare — the projection has drifted from its source of truth", f.Name, key)
		}
		switch f.Type.Kind() {
		case reflect.String:
			wire[key] = "probe"
		case reflect.Bool:
			wire[key] = true
		case reflect.Int, reflect.Int64:
			wire[key] = 7
		case reflect.Float64:
			wire[key] = 1.5
		case reflect.Slice:
			wire[key] = []any{"probe"}
		case reflect.Map:
			wire[key] = map[string]any{"k": "v"}
		default:
			// nested structs/pointers/interfaces are not filled by a flat wire map
			continue
		}
	}
	if exported == 0 {
		t.Fatal("the projection has no exported fields — the test would prove nothing")
	}

	b, err := json.Marshal(wire)
	if err != nil {
		t.Fatalf("marshal the wire map: %v", err)
	}
	// THE PATH: json -> struct, exactly `json.Unmarshal(req.GetParamsJson(), &op)`.
	if err := json.Unmarshal(b, ptr.Interface()); err != nil {
		t.Fatalf("decode the wire map into the projection: %v", err)
	}

	elem := ptr.Elem()
	missing := []string{}
	for key := range wire {
		found := false
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if f.PkgPath != "" {
				continue
			}
			if strings.Split(f.Tag.Get("json"), ",")[0] != key {
				continue
			}
			found = true
			if elem.Field(i).IsZero() {
				missing = append(missing, fmt.Sprintf("%s (field %s)", key, f.Name))
			}
		}
		if !found {
			missing = append(missing, key+" (no field carries it)")
		}
	}
	if len(missing) > 0 {
		t.Fatalf("%d wire key(s) were not delivered into the projection: %v", len(missing), missing)
	}
	t.Logf("verified %d exported field(s) against the schema's wire keys and decoded %d of them", exported, len(wire))
}
