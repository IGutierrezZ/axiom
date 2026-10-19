package branch

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// Pruebas del resultado de `status` sin `--change` (los cambios registrados y los
// pendientes de archivar, REQ-25.15 y REQ-25.20) y del campo `pendingArchive` de
// `status --change`.

func boolPointer(value bool) *bool { return &value }

// REQ-25.15 y design §4.3: `pendingArchive` es una lista de cambios pendientes sin
// `--change` y un booleano con él; la lista evaluada y vacía se imprime como `[]` y la que
// no se evaluó (sin referencia de integración) no se imprime.
func TestPendingArchiveSerialisesAsAListOrAsABoolean(t *testing.T) {
	pending := []PendingChange{{Change: "y", Continuation: "axiom branch ensure --change y --archive"}}
	tests := []struct {
		name   string
		field  *PendingArchive
		want   []string
		absent bool
	}{
		{"lista con un pendiente", &PendingArchive{Changes: pending}, []string{`"pendingArchive": [`, `"change": "y"`, `"continuation": "axiom branch ensure --change y --archive"`}, false},
		{"lista evaluada y vacía", &PendingArchive{Changes: []PendingChange{}}, []string{`"pendingArchive": []`}, false},
		{"lista sin elementos se imprime vacía, no null", &PendingArchive{}, []string{`"pendingArchive": []`}, false},
		{"booleano verdadero", &PendingArchive{Requested: boolPointer(true)}, []string{`"pendingArchive": true`}, false},
		{"booleano falso", &PendingArchive{Requested: boolPointer(false)}, []string{`"pendingArchive": false`}, false},
		{"sin evaluar", nil, nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := NewResult("status")
			result.PendingArchive = tc.field

			raw, decoded := resultJSON(t, result)

			if _, present := decoded["pendingArchive"]; present == tc.absent {
				t.Fatalf("pendingArchive presente = %v, se esperaba %v:\n%s", present, !tc.absent, raw)
			}
			for _, want := range tc.want {
				if !strings.Contains(raw, want) {
					t.Errorf("la salida debe contener %q:\n%s", want, raw)
				}
			}
		})
	}
}

// REQ-25.20: la continuación de un cambio pendiente es una orden ejecutable `axiom
// branch ensure --change <c> --archive`; un nombre con caracteres que el intérprete
// leería de otro modo va citado.
func TestPendingChangeNamesTheArchiveContinuation(t *testing.T) {
	for _, tc := range []struct{ change, want string }{
		{"inc-99-ejemplo", "axiom branch ensure --change inc-99-ejemplo --archive"},
		{"mi cambio", "axiom branch ensure --change 'mi cambio' --archive"},
	} {
		if got := NewPendingChange(tc.change); got.Change != tc.change || got.Continuation != tc.want {
			t.Errorf("NewPendingChange(%q) = %+v, se esperaba la continuación %q", tc.change, got, tc.want)
		}
	}
}

// REQ-25.15: un cambio registrado se resume con su carril, su modo y su origen (cada
// token con su etiqueta) y las ramas del registro con su worktree, con rutas absolutas.
func TestNewRegisteredChangeSummarisesTheRecord(t *testing.T) {
	record := Record{
		Schema: RegistrySchema, Lane: LaneSDD, Change: "x", Mode: ModeWorktree, ModeSource: ModeSourceHub,
		Repositories: []RecordRepo{
			{Name: "api", Branches: []RecordBranch{
				{Name: "feat/x", Kind: KindChange, Worktree: `C:\wt\x\api`},
				{Name: "feat/x-core", Kind: KindRole, RoleID: "core", Worktree: `C:\wt\x-core\api`},
			}},
			{Name: "web", Branches: []RecordBranch{{Name: "feat/x", Kind: KindChange}}},
		},
	}

	got := NewRegisteredChange(record)

	want := RegisteredChange{
		Change: "x", Lane: LaneSDD,
		Mode: ModeWorktree, ModeLabel: "un worktree por rama",
		ModeSource: ModeSourceHub, ModeSourceLabel: "preferencias personales del hub",
		Branches: []RegisteredBranch{
			{Repository: "api", Name: "feat/x", Kind: KindChange, KindLabel: "rama del cambio", Worktree: `C:\wt\x\api`},
			{Repository: "api", Name: "feat/x-core", Kind: KindRole, KindLabel: "rama de rol", Role: "core", Worktree: `C:\wt\x-core\api`},
			{Repository: "web", Name: "feat/x", Kind: KindChange, KindLabel: "rama del cambio"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewRegisteredChange = %+v\nse esperaba %+v", got, want)
	}
}

// `registered` se imprime como `[]` cuando se listó y no hay ninguno, y no se imprime
// cuando no se listó (un resultado con `--change`).
func TestRegisteredSerialisesEmptyOnlyWhenItWasListed(t *testing.T) {
	listed := NewResult("status")
	listed.Registered = []RegisteredChange{}
	if raw, _ := resultJSON(t, listed); !strings.Contains(raw, `"registered": []`) {
		t.Errorf("una lista vacía se imprime como []:\n%s", raw)
	}
	notListed := NewResult("status")
	if raw, decoded := resultJSON(t, notListed); decoded["registered"] != nil {
		t.Errorf("sin listar no hay clave registered:\n%s", raw)
	}

	populated := NewResult("status")
	populated.Registered = []RegisteredChange{NewRegisteredChange(Record{Lane: LaneSDD, Change: "x", Mode: ModeInPlace, ModeSource: ModeSourceFlag})}
	_, decoded := resultJSON(t, populated)
	entry := decoded["registered"].([]any)[0].(map[string]any)
	for key, want := range map[string]any{"change": "x", "lane": "sdd", "mode": "in_place", "modeLabel": "en la carpeta del proyecto", "modeSource": "flag", "modeSourceLabel": "bandera --mode"} {
		if entry[key] != want {
			t.Errorf("registered[0].%s = %v, se esperaba %v", key, entry[key], want)
		}
	}
	if _, present := entry["branches"]; present {
		t.Errorf("un registro sin ramas no imprime branches: %v", entry)
	}
}

// El texto de `status` sin `--change` lista los registrados y los pendientes en
// castellano, con «ninguno» cuando no hay; con `--change`, el aviso de que está pendiente.
func TestWriteTextPrintsTheRegisteredAndThePendingChanges(t *testing.T) {
	overview := NewResult("status")
	overview.Lane, overview.WorkspaceRoot = LaneSDD, `C:\proyecto`
	overview.Registered = []RegisteredChange{NewRegisteredChange(Record{
		Lane: LaneSDD, Change: "a", Mode: ModeWorktree, ModeSource: ModeSourceAnswer,
		Repositories: []RecordRepo{{Name: "api", Branches: []RecordBranch{{Name: "feat/a", Kind: KindChange, Worktree: `C:\wt\a\api`}}}},
	})}
	overview.PendingArchive = &PendingArchive{Changes: []PendingChange{NewPendingChange("y")}}

	var stdout, stderr bytes.Buffer
	overview.WriteText(&stdout, &stderr)

	for _, want := range []string{
		"Proyecto: C:\\proyecto (carril sdd)\n",
		"Cambios registrados: 1\n",
		"  - a: un worktree por rama (respuesta registrada)\n",
		"    - api: feat/a (rama del cambio), worktree C:\\wt\\a\\api\n",
		"Cambios pendientes de archivar: 1\n",
		"  - y: axiom branch ensure --change y --archive\n",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("la salida de texto debe contener %q:\n%s", want, stdout.String())
		}
	}
	if stderr.Len() != 0 {
		t.Errorf("un resultado listo no escribe en stderr: %q", stderr.String())
	}

	t.Run("sin registrados ni pendientes", func(t *testing.T) {
		empty := NewResult("status")
		empty.Lane = LaneSDD
		empty.Registered = []RegisteredChange{}
		empty.PendingArchive = &PendingArchive{Changes: []PendingChange{}}
		var out bytes.Buffer
		empty.WriteText(&out, &stderr)
		for _, want := range []string{"Cambios registrados: ninguno\n", "Cambios pendientes de archivar: ninguno\n"} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("la salida de texto debe contener %q:\n%s", want, out.String())
			}
		}
	})
	t.Run("pendientes sin evaluar no se mencionan", func(t *testing.T) {
		notEvaluated := NewResult("status")
		notEvaluated.Lane = LaneSDD
		notEvaluated.Registered = []RegisteredChange{}
		var out bytes.Buffer
		notEvaluated.WriteText(&out, &stderr)
		if strings.Contains(out.String(), "pendientes de archivar") {
			t.Errorf("sin evaluar no se informa de pendientes (el aviso lo explica):\n%s", out.String())
		}
	})
	t.Run("con --change", func(t *testing.T) {
		pending := NewResult("status")
		pending.Change, pending.Lane = "x", LaneSDD
		pending.PendingArchive = &PendingArchive{Requested: boolPointer(true)}
		pending.PendingArchiveContinuation = NewPendingChange("x").Continuation
		var out bytes.Buffer
		pending.WriteText(&out, &stderr)
		if want := "Pendiente de archivar: sí (axiom branch ensure --change x --archive)\n"; !strings.Contains(out.String(), want) {
			t.Errorf("la salida de texto debe contener %q:\n%s", want, out.String())
		}

		notPending := NewResult("status")
		notPending.Change, notPending.Lane = "x", LaneSDD
		notPending.PendingArchive = &PendingArchive{Requested: boolPointer(false)}
		out.Reset()
		notPending.WriteText(&out, &stderr)
		if strings.Contains(out.String(), "Pendiente de archivar") {
			t.Errorf("un cambio que no está pendiente no lo menciona:\n%s", out.String())
		}
	})
}
