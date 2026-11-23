package azuredevops

import (
	"errors"
	"testing"
)

// newPatchBody es la única forma de construir el cuerpo de una modificación (REQ-45.7).
func TestNewPatchBodyPrependsTestRev(t *testing.T) {
	body, err := newPatchBody(7, []PatchOp{{Op: "replace", Path: "/fields/System.Title", Value: "nuevo"}})
	if err != nil {
		t.Fatalf("newPatchBody: %v", err)
	}
	const want = `[{"op":"test","path":"/rev","value":7},{"op":"replace","path":"/fields/System.Title","value":"nuevo"}]`
	if string(body) != want {
		t.Fatalf("cuerpo = %s\nesperaba %s", body, want)
	}
}

func TestNewPatchBodyKeepsFalsyValues(t *testing.T) {
	body, err := newPatchBody(3, []PatchOp{{Op: "add", Path: "/fields/Microsoft.VSTS.Scheduling.RemainingWork", Value: 0}})
	if err != nil {
		t.Fatal(err)
	}
	const want = `[{"op":"test","path":"/rev","value":3},{"op":"add","path":"/fields/Microsoft.VSTS.Scheduling.RemainingWork","value":0}]`
	if string(body) != want {
		t.Fatalf("un valor 0 no puede perderse: %s", body)
	}
}

func TestNewPatchBodyRejections(t *testing.T) {
	ok := []PatchOp{{Op: "replace", Path: "/fields/System.Title", Value: "x"}}
	tests := []struct {
		name string
		rev  int
		ops  []PatchOp
		want error
	}{
		{"rev cero", 0, ok, ErrMissingRev},
		{"rev negativa", -1, ok, ErrMissingRev},
		{"sin operaciones", 7, nil, ErrBadRequest},
		{"operaciones vacías", 7, []PatchOp{}, ErrBadRequest},
		{"test sobre /rev aportado", 7, append([]PatchOp{{Op: "test", Path: "/rev", Value: 7}}, ok...), ErrBadRequest},
		{"test sobre /rev en mayúsculas", 7, []PatchOp{{Op: "TEST", Path: "/rev", Value: 99}}, ErrBadRequest},
		{"op vacía", 7, []PatchOp{{Path: "/fields/System.Title"}}, ErrBadRequest},
		{"path vacío", 7, []PatchOp{{Op: "replace"}}, ErrBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := newPatchBody(tt.rev, tt.ops)
			if !errors.Is(err, tt.want) || body != nil {
				t.Fatalf("newPatchBody = %s, %v; esperaba %v", body, err, tt.want)
			}
		})
	}
}

func TestNewPatchBodyAllowsATestOnAnotherPath(t *testing.T) {
	body, err := newPatchBody(7, []PatchOp{{Op: "test", Path: "/fields/System.State", Value: "Active"}})
	if err != nil {
		t.Fatalf("una prueba sobre otro campo es legítima: %v", err)
	}
	if got := string(body); got != `[{"op":"test","path":"/rev","value":7},{"op":"test","path":"/fields/System.State","value":"Active"}]` {
		t.Fatalf("cuerpo = %s", got)
	}
}

func TestMissingIDFromMessage(t *testing.T) {
	requested := []int{1, 999, 5}
	tests := []struct {
		message string
		want    int
	}{
		{"TF401232: Work item 999 does not exist, or you do not have permissions to read it.", 999},
		{"Work item 5 no existe", 5},
		{"sin números", 0},
		{"TF401232: otro", 0}, // 401232 no es un id pedido
		{"el 9990 no es el 999", 999},
	}
	for _, tt := range tests {
		if got := missingIDFromMessage(tt.message, requested); got != tt.want {
			t.Errorf("missingIDFromMessage(%q) = %d, esperaba %d", tt.message, got, tt.want)
		}
	}
}
