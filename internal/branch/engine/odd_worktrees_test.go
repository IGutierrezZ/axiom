package engine

import (
	"reflect"
	"testing"
)

// SelectODDDocs elige la feature de un worktree: el documento con el nombre del slug; si no, los
// añadidos; si no, los modificados.
func TestSelectODDDocs(t *testing.T) {
	modified := ODDDocChange{Name: "previo"}
	added := ODDDocChange{Name: "nuevo", Added: true}
	slug := ODDDocChange{Name: "mi-slug"}

	cases := []struct {
		name    string
		changes []ODDDocChange
		want    []ODDDocChange
	}{
		{"sin cambios", nil, nil},
		{"el del slug gana a todo", []ODDDocChange{added, slug, modified}, []ODDDocChange{slug}},
		{"sin slug, los añadidos", []ODDDocChange{modified, added}, []ODDDocChange{added}},
		{"sin añadidos, los modificados", []ODDDocChange{modified}, []ODDDocChange{modified}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SelectODDDocs("mi-slug", tc.changes)
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("SelectODDDocs = %+v, se esperaba %+v", got, tc.want)
			}
		})
	}
}
