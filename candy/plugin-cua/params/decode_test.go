package params

// decode_test.go — pins each generated projection against its SOURCE OF TRUTH, in both directions.
//
// charly hands a plugin its parameters as JSON (`spec.Op`'s params_json) and the plugin decodes
// that JSON into these projections; the sdk's kit.DecodeInput re-marshals to JSON before it
// unmarshals too. No YAML decoder ever sees these types — which is why the projections' `yaml:`
// tags are inert, and why this test pins the JSON side rather than asserting that claim in prose.
//
// The expected keys are copied from the `#…` definitions in `candy/plugin-cua/schema/*.cue` — the
// schema that GENERATES this file. They are hardcoded on purpose: deriving them from a struct's own
// `json` tags (the obvious thing) makes the test vacuous, because a tag that drifts takes its
// derived expectation with it and the decode still lines up.

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// cueInputKeys is the field set of #CuaInput in candy/plugin-cua/schema.
var cueInputKeys = []string{
	"allow_control", "amount", "app", "args", "artifact", "artifact_contains_text",
	"artifact_dir", "artifact_min_bytes", "artifact_min_dimensions", "artifact_not_uniform",
	"button", "delivery", "device", "direction", "driver", "element_index", "env", "from_x",
	"from_y", "height", "host", "key", "keys", "method", "pid", "text", "tool", "value",
	"width", "window_id", "x", "y",
}

// cueDeviceKeys is the field set of #CuaDeviceInput — the OTHER generated struct this diff
// touches, so it carries its own coverage rather than riding on CuaInput's.
var cueDeviceKeys = []string{
	"capability_manifest", "description", "driver", "env", "host", "permission_mode",
}

func TestCuaInputMatchesItsSchema(t *testing.T) {
	checkProjection(t, reflect.TypeOf(CuaInput{}), cueInputKeys, "#CuaInput")
}

func TestCuaDeviceInputMatchesItsSchema(t *testing.T) {
	checkProjection(t, reflect.TypeOf(CuaDeviceInput{}), cueDeviceKeys, "#CuaDeviceInput")
}

// checkProjection asserts BOTH directions between one generated struct and the schema definition
// that produced it, then decodes a wire map through it.
func checkProjection(t *testing.T, rt reflect.Type, schemaKeys []string, defName string) {
	t.Helper()
	ptr := reflect.New(rt)

	// Direction 1: STRUCT -> SCHEMA. Every exported field must carry a json key the schema
	// declares; a drifted, emptied or removed tag is caught here.
	wire := map[string]any{}
	exported := 0
	tagOf := map[string]string{} // json key -> Go field name
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.PkgPath != "" {
			continue // unexported
		}
		exported++
		key := strings.Split(f.Tag.Get("json"), ",")[0]
		if key == "" || key == "-" {
			t.Fatalf("%s: field %s carries no usable json tag (%q) — no schema key can reach it", defName, f.Name, key)
		}
		if !containsKey(schemaKeys, key) {
			t.Fatalf("%s: field %s is tagged json:%q, which candy/plugin-cua/schema does NOT declare — the projection has drifted from its source of truth", defName, f.Name, key)
		}
		tagOf[key] = f.Name
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
		t.Fatalf("%s: the projection has no exported fields — the test would prove nothing", defName)
	}

	// Direction 2: SCHEMA -> STRUCT. Every key the schema declares must be carried by a field; a
	// key the projection dropped is caught here, checked against schemaKeys directly rather than
	// against anything derived from the tags.
	for _, key := range schemaKeys {
		if _, ok := tagOf[key]; !ok {
			t.Fatalf("%s: schema key %q is declared in candy/plugin-cua/schema but NO exported field carries it — the projection is missing part of its schema", defName, key)
		}
	}

	b, err := json.Marshal(wire)
	if err != nil {
		t.Fatalf("%s: marshal the wire map: %v", defName, err)
	}
	// THE PATH: json -> struct, exactly `json.Unmarshal(req.GetParamsJson(), &op)`.
	if err := json.Unmarshal(b, ptr.Interface()); err != nil {
		t.Fatalf("%s: decode the wire map into the projection: %v", defName, err)
	}
	elem := ptr.Elem()
	missing := []string{}
	for key := range wire {
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if f.PkgPath == "" && strings.Split(f.Tag.Get("json"), ",")[0] == key && elem.Field(i).IsZero() {
				missing = append(missing, fmt.Sprintf("%s (field %s)", key, f.Name))
			}
		}
	}
	if len(missing) > 0 {
		t.Fatalf("%s: %d wire key(s) were not delivered into the projection: %v", defName, len(missing), missing)
	}
	t.Logf("%s: both directions hold — %d exported field(s), %d schema key(s), %d decoded", defName, exported, len(schemaKeys), len(wire))
}

func containsKey(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}
