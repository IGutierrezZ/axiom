package filemerge

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// UnionJSONArrays returns overlayJSON with the arrays at the declared key paths
// replaced by the union of the array already present in baseJSON and the
// overlay's own array. The base entries come first and keep their order; only
// the overlay entries that the base does not carry are appended, so the result
// has no duplicates and a second merge over its own output changes nothing.
//
// The union is limited to the declared paths: array semantics elsewhere in the
// merge (mergeObjects replaces arrays) are untouched. A path whose base or
// overlay value is not an array is left as the overlay declares it. A base that
// cannot be decoded leaves the overlay unchanged, because the merge that
// follows already decides how to treat an unreadable base.
func UnionJSONArrays(baseJSON, overlayJSON []byte, paths ...[]string) ([]byte, error) {
	overlay, err := unmarshalJSONObject(overlayJSON)
	if err != nil {
		return nil, fmt.Errorf("unmarshal overlay json: %w", err)
	}
	base, err := unmarshalJSONObject(baseJSON)
	if err != nil {
		return overlayJSON, nil
	}

	for _, path := range paths {
		baseArray, ok := jsonArrayAt(base, path)
		if !ok {
			continue
		}
		parent, ok := jsonParentAt(overlay, path)
		if !ok {
			continue
		}
		key := path[len(path)-1]
		overlayArray, ok := parent[key].([]any)
		if !ok {
			continue
		}
		parent[key] = unionValues(baseArray, overlayArray)
	}

	encoded, err := json.Marshal(overlay)
	if err != nil {
		return nil, fmt.Errorf("marshal overlay json: %w", err)
	}
	return encoded, nil
}

func jsonArrayAt(root map[string]any, path []string) ([]any, bool) {
	if len(path) == 0 {
		return nil, false
	}
	parent, ok := jsonParentAt(root, path)
	if !ok {
		return nil, false
	}
	array, ok := parent[path[len(path)-1]].([]any)
	return array, ok
}

func jsonParentAt(root map[string]any, path []string) (map[string]any, bool) {
	if len(path) == 0 {
		return nil, false
	}
	current := root
	for _, segment := range path[:len(path)-1] {
		next, ok := current[segment].(map[string]any)
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func unionValues(first, second []any) []any {
	union := make([]any, 0, len(first)+len(second))
	union = append(union, first...)
	for _, candidate := range second {
		present := false
		for _, existing := range union {
			if reflect.DeepEqual(existing, candidate) {
				present = true
				break
			}
		}
		if !present {
			union = append(union, candidate)
		}
	}
	return union
}
