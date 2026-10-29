package filemerge

import (
	"encoding/json"
	"reflect"
	"testing"
)

func decodeStrings(t *testing.T, raw []byte, path ...string) []any {
	t.Helper()
	root, err := unmarshalJSONObject(raw)
	if err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	array, ok := jsonArrayAt(root, path)
	if !ok {
		t.Fatalf("no array at %v in %s", path, raw)
	}
	return array
}

func TestUnionJSONArraysKeepsBaseFirstThenAppendsMissing(t *testing.T) {
	base := []byte(`{"permissions":{"deny":["b","a","b"]}}`)
	overlay := []byte(`{"permissions":{"deny":["a","c","d","c"]}}`)

	got, err := UnionJSONArrays(base, overlay, []string{"permissions", "deny"})
	if err != nil {
		t.Fatalf("UnionJSONArrays() error = %v", err)
	}

	// The base keeps its own order and entries verbatim; only entries the base
	// lacks are appended, once each.
	want := []any{"b", "a", "b", "c", "d"}
	if diff := decodeStrings(t, got, "permissions", "deny"); !reflect.DeepEqual(diff, want) {
		t.Fatalf("union = %v, want %v", diff, want)
	}
}

func TestUnionJSONArraysLeavesOverlayWhenBaseHasNoArray(t *testing.T) {
	overlay := []byte(`{"permissions":{"deny":["a","b"]}}`)
	for name, base := range map[string]string{
		"empty":       ``,
		"no key":      `{"permissions":{"allow":["x"]}}`,
		"not array":   `{"permissions":{"deny":"x"}}`,
		"not object":  `{"permissions":"x"}`,
		"unreadable":  `{not json`,
		"other array": `{"deny":["x"]}`,
	} {
		t.Run(name, func(t *testing.T) {
			got, err := UnionJSONArrays([]byte(base), overlay, []string{"permissions", "deny"})
			if err != nil {
				t.Fatalf("UnionJSONArrays() error = %v", err)
			}
			if want := []any{"a", "b"}; !reflect.DeepEqual(decodeStrings(t, got, "permissions", "deny"), want) {
				t.Fatalf("overlay changed: %s", got)
			}
		})
	}
}

func TestUnionJSONArraysOnlyTouchesDeclaredPaths(t *testing.T) {
	base := []byte(`{"permissions":{"deny":["u"],"allow":["x"]}}`)
	overlay := []byte(`{"permissions":{"deny":["a"],"allow":["y"]}}`)

	got, err := UnionJSONArrays(base, overlay, []string{"permissions", "deny"})
	if err != nil {
		t.Fatalf("UnionJSONArrays() error = %v", err)
	}

	if want := []any{"u", "a"}; !reflect.DeepEqual(decodeStrings(t, got, "permissions", "deny"), want) {
		t.Fatalf("deny = %s", got)
	}
	// The undeclared array keeps the merge's ordinary replace semantics: the
	// overlay value is passed through untouched.
	if want := []any{"y"}; !reflect.DeepEqual(decodeStrings(t, got, "permissions", "allow"), want) {
		t.Fatalf("allow = %s", got)
	}
}

func TestUnionJSONArraysRejectsMalformedOverlay(t *testing.T) {
	if _, err := UnionJSONArrays([]byte(`{}`), []byte(`{broken`), []string{"a"}); err == nil {
		t.Fatal("expected an error for a malformed overlay")
	}
}

func TestUnionJSONArraysThroughMergeIsIdempotent(t *testing.T) {
	path := []string{"permissions", "deny"}
	overlay := []byte(`{"permissions":{"deny":["a","b"]}}`)
	base := []byte(`{"permissions":{"defaultMode":"plan","deny":["z","a"]}}`)

	merge := func(base []byte) []byte {
		union, err := UnionJSONArrays(base, overlay, path)
		if err != nil {
			t.Fatal(err)
		}
		merged, err := MergeJSONObjects(base, union)
		if err != nil {
			t.Fatal(err)
		}
		return merged
	}

	first := merge(base)
	second := merge(first)
	if string(first) != string(second) {
		t.Fatalf("second merge changed the output:\nfirst=%s\nsecond=%s", first, second)
	}

	var settings struct {
		Permissions struct {
			Mode string   `json:"defaultMode"`
			Deny []string `json:"deny"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(first, &settings); err != nil {
		t.Fatal(err)
	}
	if settings.Permissions.Mode != "plan" || !reflect.DeepEqual(settings.Permissions.Deny, []string{"z", "a", "b"}) {
		t.Fatalf("merged = %+v", settings.Permissions)
	}
}
