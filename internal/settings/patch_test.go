package settings

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// patched aplica el parche a doc como lo hace Write (sin validar) y devuelve el texto.
func patched(t *testing.T, doc string, patch Patch) (string, *Error) {
	t.Helper()
	parsed := parseDocument([]byte(doc))
	if parsed.blocked != "" {
		return "", newError(CodeUnsupportedDocument, parsed.blocked)
	}
	if err := parsed.applyPatch(patch); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(parsed.top); err != nil {
		t.Fatal(err)
	}
	_ = enc.Close()
	return string(restoreLayout([]byte(doc), buf.Bytes())), nil
}

func mustPatch(t *testing.T, doc string, patch Patch) string {
	t.Helper()
	out, err := patched(t, doc, patch)
	if err != nil {
		t.Fatalf("patch %v: %v", patch, err)
	}
	return out
}

func wantCode(t *testing.T, err *Error, code Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("quería el error %s y no hubo ninguno", code)
	}
	if err.Code != code {
		t.Fatalf("código = %s (%s), quiero %s", err.Code, err.Message, code)
	}
}

// RF-TOP-22: cambiar un escalar existente conserva su estilo de comillas y su
// comentario de línea.
func TestPatchChangesAScalarKeepingItsStyle(t *testing.T) {
	doc := monoDoc + "workflow:\n  lane: \"ask\" # carril\ngit:\n  branch_prefix: \"inc\"  # dos espacios\n"
	got := mustPatch(t, doc, Patch{"workflow.lane": "sdd"})
	if want := strings.Replace(doc, `"ask"`, `"sdd"`, 1); got != want {
		t.Fatalf("resultado:\n%s", got)
	}
	plain := monoDoc + "workflow:\n  lane: ask\n"
	if got := mustPatch(t, plain, Patch{"workflow.lane": "odd"}); got != strings.Replace(plain, "ask", "odd", 1) {
		t.Fatalf("un escalar sin comillas sigue sin comillas:\n%s", got)
	}
}

// RF-TOP-22: una clave nueva va al final del mapeo, con la cadena entre comillas
// dobles, y el booleano y el entero sin comillas.
func TestPatchCreatesKeysAtTheEndOfTheMappingWithTypedStyles(t *testing.T) {
	doc := monoDoc + "git:\n  integration_branch: main\n"
	got := mustPatch(t, doc, Patch{"git.branch_prefix": "inc", "delivery.split_hint_lines": 400, "sdd.iso": true})
	for _, want := range []string{"git:\n  integration_branch: main\n  branch_prefix: \"inc\"\n", "delivery:\n  split_hint_lines: 400\n", "sdd:\n  iso: true\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("falta %q en:\n%s", want, got)
		}
	}
	if strings.Contains(got, `"400"`) || strings.Contains(got, `"true"`) {
		t.Errorf("el entero y el booleano no llevan comillas:\n%s", got)
	}
	numeric := mustPatch(t, monoDoc, Patch{"workspace.name": "123"})
	if !strings.Contains(numeric, `name: "123"`) {
		t.Errorf("una cadena numérica conserva su tipo con comillas:\n%s", numeric)
	}
}

// RF-TOP-22: un bloque nuevo se inserta en el orden canónico, antes de roles.
func TestPatchInsertsNewBlocksInCanonicalOrder(t *testing.T) {
	got := mustPatch(t, monoDoc, Patch{"delivery.split_hint_lines": 400, "git.branch_prefix": "inc", "sdd.iso": false, "workflow.lane": "sdd"})
	order := []string{"workspace:", "\ngit:", "\nworkflow:", "\nsdd:", "\ndelivery:", "\nroles:"}
	last := -1
	for _, marker := range order {
		i := strings.Index(got, marker)
		if i < 0 || i < last {
			t.Fatalf("orden canónico roto en %q:\n%s", marker, got)
		}
		last = i
	}
	if !strings.Contains(got, "\n\ngit:") || !strings.Contains(got, "\n\nroles:") {
		t.Errorf("los bloques nuevos van separados por una línea en blanco:\n%s", got)
	}
}

// RF-TOP-22: `null` retira la clave y conserva las demás; el bloque vacío se borra.
func TestPatchNullRemovesTheKeyAndEmptyBlocks(t *testing.T) {
	doc := "workspace:\n  name: \"x\"\n\ngit:\n  main_branch: trunk\n  branch_prefix: inc\n\ndelivery:\n  split_hint_lines: 400\n\nroles:\n  a:\n    name: A\n"
	got := mustPatch(t, doc, Patch{"git.main_branch": nil})
	if strings.Contains(got, "main_branch") || !strings.Contains(got, "branch_prefix: inc") {
		t.Fatalf("retirar una clave conserva las demás:\n%s", got)
	}
	got = mustPatch(t, doc, Patch{"delivery.split_hint_lines": nil})
	if strings.Contains(got, "delivery") || strings.Contains(got, "\n\n\n") {
		t.Fatalf("el bloque vacío se borra sin dejar blancas dobles:\n%s", got)
	}
	if want := "workspace:\n  name: \"x\"\n\ngit:\n  main_branch: trunk\n  branch_prefix: inc\n\nroles:\n  a:\n    name: A\n"; got != want {
		t.Fatalf("resultado:\n%s", got)
	}
	// Retirar lo que no está es un parche sin efecto.
	if got := mustPatch(t, doc, Patch{"sdd.iso": nil}); got != doc {
		t.Fatalf("retirar una clave ausente no cambia nada:\n%s", got)
	}
	_, err := patched(t, doc, Patch{"workspace.name": nil})
	wantCode(t, err, CodeInvalidValue)
}

// RF-TOP-22: repositories[<ruta>].<campo> se crea (entrada y bloque) y se borra.
func TestPatchCreatesAndRemovesRepositoryEntries(t *testing.T) {
	created := mustPatch(t, multiDoc, Patch{"repositories[repo-backend].promotion": "release"})
	if want := "repositories:\n  - path: \"repo-backend\"\n    promotion: \"release\"\n"; !strings.Contains(created, want) {
		t.Fatalf("entrada creada:\n%s", created)
	}
	// El selector se normaliza.
	again := mustPatch(t, created, Patch{"repositories[.\\repo-backend\\].branch_prefix": "fix"})
	if !strings.Contains(again, "branch_prefix: \"fix\"") || strings.Count(again, "path:") != 3 {
		t.Fatalf("el selector se compara normalizado:\n%s", again)
	}
	partial := mustPatch(t, again, Patch{"repositories[repo-backend].promotion": nil})
	if !strings.Contains(partial, "branch_prefix: \"fix\"") || strings.Contains(partial, "promotion") {
		t.Fatalf("retirar un campo conserva la entrada con otros campos:\n%s", partial)
	}
	removed := mustPatch(t, partial, Patch{"repositories[repo-backend].branch_prefix": nil})
	if strings.Contains(removed, "\nrepositories:") {
		t.Fatalf("al quedar sin campos de ajuste se borran la entrada y el bloque:\n%s", removed)
	}
	if strings.ReplaceAll(removed, "\n\n", "\n") != multiDoc {
		t.Fatalf("debe volver al original:\n%s", removed)
	}
}

// RF-TOP-22: roles[x].gate_policy en minúsculas, y retirarlo no borra el rol.
func TestPatchSetsGatePolicyLowercaseAndNeverDeletesTheRole(t *testing.T) {
	got := mustPatch(t, multiDoc, Patch{"roles[backend].gate_policy": " Deferred "})
	if !strings.Contains(got, "gate_policy: \"deferred\"") {
		t.Fatalf("gate_policy en minúsculas:\n%s", got)
	}
	back := mustPatch(t, got, Patch{"roles[backend].gate_policy": nil})
	if back != multiDoc {
		t.Fatalf("retirar gate_policy deja el rol intacto:\n%s", back)
	}
	_, err := patched(t, multiDoc, Patch{"roles[backend].gate_policy": "siempre"})
	wantCode(t, err, CodeInvalidValue)
}

// RF-TOP-22: claves desconocidas, de solo lectura y no aplicables.
func TestPatchRejectsUnknownReadOnlyAndNotApplicableKeys(t *testing.T) {
	cases := []struct {
		name  string
		doc   string
		patch Patch
		code  Code
	}{
		{"desconocida", monoDoc, Patch{"git.inventada": "x"}, CodeUnknownKey},
		{"integrations", monoDoc, Patch{"integrations.token": "x"}, CodeUnknownKey},
		{"solo lectura topology", monoDoc, Patch{"workspace.topology": "multirepo"}, CodeKeyNotEditable},
		{"solo lectura governance", monoDoc, Patch{"governance.language": "en"}, CodeKeyNotEditable},
		{"solo lectura technical_context", monoDoc, Patch{"technical_context": "x"}, CodeKeyNotEditable},
		{"multirepo en monorepo", monoDoc, Patch{"acceptance_tests.enabled": true}, CodeKeyNotApplicable},
		{"rol en monorepo", monoDoc, Patch{"roles[fullstack].gate_policy": "optional"}, CodeKeyNotApplicable},
		{"rol inexistente", multiDoc, Patch{"roles[ghost].gate_policy": "optional"}, CodeKeyNotApplicable},
		{"repositorio inexistente", multiDoc, Patch{"repositories[ghost].promotion": "release"}, CodeKeyNotApplicable},
		{"tipo incorrecto", monoDoc, Patch{"sdd.iso": "true"}, CodeInvalidValue},
		{"entero con decimales", monoDoc, Patch{"delivery.split_hint_lines": 4.5}, CodeInvalidValue},
		{"lista con números", multiDoc, Patch{"acceptance_tests.environments": []any{"dev", 3}}, CodeInvalidValue},
		{"texto donde va una lista", multiDoc, Patch{"acceptance_tests.environments": "dev"}, CodeInvalidValue},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := patched(t, tc.doc, tc.patch)
			wantCode(t, err, tc.code)
			if err.Message == "" {
				t.Error("el mensaje debe estar en castellano")
			}
		})
	}
	// Un lote con una clave válida y otra inválida se rechaza entero.
	_, err := patched(t, monoDoc, Patch{"workflow.lane": "sdd", "git.inventada": "x"})
	wantCode(t, err, CodeUnknownKey)
}

// RF-TOP-22: una lista nueva usa comillas dobles y una existente conserva su estilo.
func TestPatchWritesStringLists(t *testing.T) {
	created := mustPatch(t, multiDoc, Patch{"acceptance_tests.enabled": true, "acceptance_tests.environments": []any{"dev", "pro"}})
	if want := "acceptance_tests:\n  enabled: true\n  environments:\n    - \"dev\"\n    - \"pro\"\n"; !strings.Contains(created, want) {
		t.Fatalf("lista nueva:\n%s", created)
	}
	flow := multiDoc + "acceptance_tests:\n  enabled: true\n  environments: [dev, pro]\n"
	got := mustPatch(t, flow, Patch{"acceptance_tests.environments": []string{"dev", "pre", "pro"}})
	if !strings.Contains(got, "environments: [dev, pre, pro]") {
		t.Fatalf("la lista en línea conserva su estilo:\n%s", got)
	}
}

// RF-TOP-22: una clave repetida en el mapeo que se edita es unsupported_document.
func TestPatchRejectsRepeatedKeys(t *testing.T) {
	_, err := patched(t, monoDoc+"git:\n  branch_prefix: a\n  branch_prefix: b\n", Patch{"git.branch_prefix": "c"})
	wantCode(t, err, CodeUnsupportedDocument)
	if !strings.Contains(err.Message, "branch_prefix") || !strings.Contains(err.Message, "línea") {
		t.Errorf("el mensaje nombra la clave y la línea: %q", err.Message)
	}
	var typed *Error
	if !errors.As(error(err), &typed) {
		t.Error("Error debe ser un error tipado")
	}
}
