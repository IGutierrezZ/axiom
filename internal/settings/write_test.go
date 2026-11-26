package settings

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// project es un proyecto temporal con su axiom.yaml.
type project struct {
	dir  string
	path string
}

// newProject escribe content como axiom.yaml en un directorio temporal y crea las
// carpetas que el motor comprueba contra el disco.
func newProject(t *testing.T, content string, dirs ...string) project {
	t.Helper()
	dir := t.TempDir()
	for _, d := range append([]string{"openspec"}, dirs...) {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	p := project{dir: dir, path: filepath.Join(dir, "axiom.yaml")}
	if err := os.WriteFile(p.path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func (p project) bytes(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(p.path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func (p project) revision(t *testing.T) string { return revisionOf(p.bytes(t)) }

func (p project) listing(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(p.dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

func (p project) write(t *testing.T, patch Patch) (*WriteResult, error) {
	t.Helper()
	return Write(WriteRequest{ConfigPath: p.path, Master: p.dir, ExpectedRevision: p.revision(t), Patch: patch})
}

// rejects comprueba que la escritura se rechaza con ese código, que los bytes del
// fichero no cambian y que el directorio no gana ningún temporal.
func (p project) rejects(t *testing.T, patch Patch, code Code) *Error {
	t.Helper()
	before, listing := p.bytes(t), p.listing(t)
	_, err := p.write(t, patch)
	var typed *Error
	if !errors.As(err, &typed) {
		t.Fatalf("quería un *Error %s y obtuve %v", code, err)
	}
	if typed.Code != code {
		t.Fatalf("código = %s (%s), quiero %s", typed.Code, typed, code)
	}
	if !bytes.Equal(p.bytes(t), before) {
		t.Fatal("un rechazo no debe modificar el fichero")
	}
	if got := strings.Join(p.listing(t), ","); got != strings.Join(listing, ",") {
		t.Fatalf("el directorio cambió: %s frente a %s", got, strings.Join(listing, ","))
	}
	return typed
}

// RF-TOP-22: el round-trip conserva comentarios, orden y líneas en blanco: solo
// cambia la línea editada (golden).
func TestWriteGoldenOnlyTheEditedLineChanges(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("testdata", "commented.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	golden, err := os.ReadFile(filepath.Join("testdata", "commented.lane-sdd.golden"))
	if err != nil {
		t.Fatal(err)
	}
	p := newProject(t, string(source))
	before := p.revision(t)
	result, err := p.write(t, Patch{"workflow.lane": "sdd"})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !result.Changed || result.Revision == before || result.Revision != p.revision(t) {
		t.Fatalf("resultado = %+v (revisión previa %s)", result, before)
	}
	if got := p.bytes(t); !bytes.Equal(got, golden) {
		t.Fatalf("no coincide con el golden:\n%s", got)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("no debería haber avisos: %v", result.Warnings)
	}
	for _, name := range p.listing(t) {
		if strings.Contains(name, "tmp") {
			t.Errorf("queda un temporal: %s", name)
		}
	}
}

// RF-TOP-23: un parche sin efecto no escribe y deja el fichero byte a byte idéntico.
func TestWriteNoOpPatchDoesNotWrite(t *testing.T) {
	source, _ := os.ReadFile(filepath.Join("testdata", "commented.yaml"))
	p := newProject(t, string(source))
	info, _ := os.Stat(p.path)
	rev := p.revision(t)
	result, err := p.write(t, Patch{"workflow.lane": "ask", "git.promotion": nil})
	if err != nil {
		t.Fatal(err)
	}
	if result.Changed || result.Revision != rev || !bytes.Equal(p.bytes(t), source) {
		t.Fatalf("resultado = %+v", result)
	}
	if after, _ := os.Stat(p.path); !after.ModTime().Equal(info.ModTime()) {
		t.Error("un parche sin efecto no debe tocar el fichero")
	}
}

// RF-TOP-22: CRLF y BOM se conservan al escribir un cambio de una sola línea.
func TestWriteKeepsCRLFAndBOM(t *testing.T) {
	source, _ := os.ReadFile(filepath.Join("testdata", "commented.yaml"))
	crlf := append(append([]byte{}, utf8BOM...), strings.ReplaceAll(string(source), "\n", "\r\n")...)
	p := newProject(t, string(crlf))
	if _, err := p.write(t, Patch{"git.branch_prefix": "feat"}); err != nil {
		t.Fatal(err)
	}
	want := bytes.Replace(crlf, []byte(`branch_prefix: "inc"`), []byte(`branch_prefix: "feat"`), 1)
	if got := p.bytes(t); !bytes.Equal(got, want) {
		t.Fatalf("CRLF/BOM:\n%q\n%q", got, want)
	}
}

// RF-TOP-23: revisión ausente, obsoleta y coincidente.
func TestWriteRevisionRules(t *testing.T) {
	source, _ := os.ReadFile(filepath.Join("testdata", "commented.yaml"))
	p := newProject(t, string(source))

	before := p.bytes(t)
	_, err := Write(WriteRequest{ConfigPath: p.path, Master: p.dir, Patch: Patch{"workflow.lane": "sdd"}})
	var typed *Error
	if !errors.As(err, &typed) || typed.Code != CodeInvalidRequest {
		t.Fatalf("sin revisión: %v", err)
	}
	_, err = Write(WriteRequest{ConfigPath: p.path, Master: p.dir, ExpectedRevision: p.revision(t)})
	if !errors.As(err, &typed) || typed.Code != CodeInvalidRequest {
		t.Fatalf("sin parche: %v", err)
	}

	stale := p.revision(t)
	external := append([]byte("# edición externa\n"), before...)
	if err := os.WriteFile(p.path, external, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Write(WriteRequest{ConfigPath: p.path, Master: p.dir, ExpectedRevision: stale, Patch: Patch{"workflow.lane": "sdd"}})
	if !errors.As(err, &typed) || typed.Code != CodeRevisionMismatch {
		t.Fatalf("revisión obsoleta: %v", err)
	}
	if !bytes.Equal(p.bytes(t), external) {
		t.Fatal("el fichero conserva la edición externa")
	}

	result, err := p.write(t, Patch{"workflow.lane": "sdd"})
	if err != nil || !result.Changed || result.Revision == stale {
		t.Fatalf("revisión coincidente: %+v %v", result, err)
	}

	missing := filepath.Join(t.TempDir(), "axiom.yaml")
	_, err = Write(WriteRequest{ConfigPath: missing, ExpectedRevision: "x", Patch: Patch{"workflow.lane": "sdd"}})
	if !errors.As(err, &typed) || typed.Code != CodeConfigMissing {
		t.Fatalf("sin fichero: %v", err)
	}
}

// RF-TOP-23: un cambio externo entre la validación y el reemplazo se detecta y no
// deja temporales. No usa t.Parallel: modifica una variable de paquete.
func TestWriteDetectsExternalChangeBeforeReplace(t *testing.T) {
	source, _ := os.ReadFile(filepath.Join("testdata", "commented.yaml"))
	p := newProject(t, string(source))
	external := append([]byte("# editado mientras se validaba\n"), source...)
	t.Cleanup(func() { beforeReplace = func(string) {} })
	beforeReplace = func(path string) {
		if err := os.WriteFile(path, external, 0o644); err != nil {
			t.Error(err)
		}
	}
	listing := p.listing(t)
	_, err := p.write(t, Patch{"workflow.lane": "sdd"})
	var typed *Error
	if !errors.As(err, &typed) || typed.Code != CodeRevisionMismatch {
		t.Fatalf("err = %v, quiero revision_mismatch", err)
	}
	if !bytes.Equal(p.bytes(t), external) {
		t.Fatal("el fichero debe conservar el cambio externo")
	}
	if got := strings.Join(p.listing(t), ","); got != strings.Join(listing, ",") {
		t.Fatalf("quedan temporales: %s", got)
	}
}

// RF-TOP-22: un valor fuera de vocabulario llega con el error del validador del motor.
func TestWriteInvalidPromotionReturnsTheValidatorError(t *testing.T) {
	source, _ := os.ReadFile(filepath.Join("testdata", "commented.yaml"))
	p := newProject(t, string(source))
	typed := p.rejects(t, Patch{"git.promotion": "siempre"}, CodeValidationFailed)
	joined := strings.Join(typed.Violations, "\n")
	if !strings.Contains(joined, "debe ser 'branch', 'release' o 'ask'") {
		t.Fatalf("falta el mensaje del validador: %v", typed.Violations)
	}
	for _, bad := range []Patch{{"delivery.split_hint_lines": 0}, {"git.branch_prefix": ""}, {"sdd.explainer_html": "nunca-jamas"}, {"sdd.id_pattern": "sin-marcadores"}} {
		p.rejects(t, bad, CodeValidationFailed)
	}
}

// RF-TOP-22: ISO en las dos direcciones y en un mismo lote válido.
func TestWriteISOContradictionBothWaysAndValidBatch(t *testing.T) {
	isoDoc := monoDoc + "workflow:\n  lane: \"sdd\"\nsdd:\n  iso: true\n"
	p := newProject(t, isoDoc)
	typed := p.rejects(t, Patch{"workflow.lane": "odd"}, CodeValidationFailed)
	if len(typed.Violations) == 0 {
		t.Fatal("debe nombrar la contradicción ISO")
	}

	oddDoc := monoDoc + "workflow:\n  lane: \"odd\"\n"
	q := newProject(t, oddDoc)
	q.rejects(t, Patch{"sdd.iso": true}, CodeValidationFailed)

	// Desactivar ISO y elegir odd en el mismo lote es válido.
	if _, err := p.write(t, Patch{"workflow.lane": "odd", "sdd.iso": false}); err != nil {
		t.Fatalf("lote válido: %v", err)
	}
	// Y activar ISO con un carril compatible en el mismo lote también.
	if _, err := q.write(t, Patch{"workflow.lane": "sdd", "sdd.iso": true}); err != nil {
		t.Fatalf("lote válido: %v", err)
	}
	if got := string(q.bytes(t)); !strings.Contains(got, "iso: true") || !strings.Contains(got, `lane: "sdd"`) {
		t.Fatalf("resultado:\n%s", got)
	}
}

// RF-TOP-22: claves desconocidas, de solo lectura y no aplicables no escriben nada.
func TestWriteRejectsUnknownReadOnlyAndNotApplicableKeys(t *testing.T) {
	source, _ := os.ReadFile(filepath.Join("testdata", "commented.yaml"))
	p := newProject(t, string(source))
	p.rejects(t, Patch{"git.inventada": "x"}, CodeUnknownKey)
	p.rejects(t, Patch{"integrations.token": "x"}, CodeUnknownKey)
	p.rejects(t, Patch{"integrations": nil}, CodeUnknownKey)
	p.rejects(t, Patch{"workspace.topology": "multirepo"}, CodeKeyNotEditable)
	p.rejects(t, Patch{"governance.language": "en"}, CodeKeyNotEditable)
	p.rejects(t, Patch{"acceptance_tests.enabled": true}, CodeKeyNotApplicable)
	p.rejects(t, Patch{"workflow.lane": "sdd", "workspace.root": "."}, CodeKeyNotEditable)
	p.rejects(t, Patch{"sdd.iso": "true"}, CodeInvalidValue)
}

// RF-TOP-22: anclas, alias, `<<`, multidocumento y clave repetida no se reescriben.
func TestWriteRejectsUnsupportedDocuments(t *testing.T) {
	cases := map[string]string{
		"ancla":          monoDoc + "git:\n  integration_branch: &b main\n",
		"alias":          monoDoc + "git:\n  integration_branch: &b main\n  main_branch: *b\n",
		"mezcla":         "base: &b\n  a: 1\n" + monoDoc + "extra:\n  <<: *b\n",
		"multidocumento": monoDoc + "---\nworkspace:\n  name: \"otro\"\n",
		"clave repetida": monoDoc + "git:\n  branch_prefix: a\n  branch_prefix: b\n",
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			p := newProject(t, doc)
			typed := p.rejects(t, Patch{"workflow.lane": "sdd"}, CodeUnsupportedDocument)
			if !strings.Contains(typed.Message, "no se puede editar") {
				t.Errorf("mensaje: %q", typed.Message)
			}
		})
	}
}

// RF-TOP-22: el lote es atómico y un fichero ilegible no se reescribe.
func TestWriteBatchIsAtomic(t *testing.T) {
	source, _ := os.ReadFile(filepath.Join("testdata", "commented.yaml"))
	p := newProject(t, string(source))
	p.rejects(t, Patch{"workflow.lane": "sdd", "git.promotion": "siempre"}, CodeValidationFailed)
	p.rejects(t, Patch{"workflow.lane": "sdd", "sdd.iso": "x"}, CodeInvalidValue)
}

// RF-TOP-22: un error previo del motor no bloquea y vuelve como aviso.
func TestWriteEnginePreexistingErrorIsAWarning(t *testing.T) {
	// El rol `core` es especializado: en monorepo-embedded el motor lo rechaza.
	doc := strings.Replace(monoDoc, "fullstack:", "core:", 1)
	p := newProject(t, doc)
	result, err := p.write(t, Patch{"workflow.lane": "sdd"})
	if err != nil {
		t.Fatalf("un error previo no debe bloquear: %v", err)
	}
	if !result.Changed || len(result.Warnings) == 0 {
		t.Fatalf("el error previo debe volver como aviso: %+v", result)
	}
	if !strings.Contains(string(p.bytes(t)), `lane: "sdd"`) {
		t.Fatal("el parche debía escribirse")
	}
}

// RF-TOP-22: un error nuevo del motor (entrada de repositories hacia una ruta
// inexistente) bloquea.
func TestWriteEngineNewErrorBlocks(t *testing.T) {
	p := newProject(t, multiDoc, "repo-specs", "repo-backend")
	typed := p.rejects(t, Patch{"repositories[repo-frontend].promotion": "release"}, CodeValidationFailed)
	if !strings.Contains(strings.Join(typed.Violations, "\n"), "repo-frontend") {
		t.Fatalf("la violación debe nombrar la ruta: %v", typed.Violations)
	}
	// Con la carpeta creada, la misma escritura es válida.
	q := newProject(t, multiDoc, "repo-specs", "repo-backend", "repo-frontend")
	if _, err := q.write(t, Patch{"repositories[repo-frontend].promotion": "release"}); err != nil {
		t.Fatalf("con la carpeta presente debe escribirse: %v", err)
	}
}

// RF-TOP-22: el campo de un documento inválido que se corrige desde la UI se acepta
// si el documento resultante es válido.
func TestWriteFixesAnInvalidDocument(t *testing.T) {
	p := newProject(t, monoDoc+"workflow:\n  lane: zzz\n")
	if _, err := p.write(t, Patch{"workflow.lane": "sdd"}); err != nil {
		t.Fatalf("corregir el valor inválido: %v", err)
	}
	if snap := readSnapshot(t, string(p.bytes(t))); len(snap.DocumentErrors) != 0 {
		t.Fatalf("el documento debe quedar válido: %v", snap.DocumentErrors)
	}
}

// Error implementa error con las violaciones, una por línea.
func TestErrorMessageListsViolations(t *testing.T) {
	err := &Error{Code: CodeValidationFailed, Message: "no es válido", Violations: []string{"a", "b"}}
	if got := err.Error(); got != "no es válido:\na\nb" {
		t.Fatalf("Error() = %q", got)
	}
	if got := (&Error{Message: "solo"}).Error(); got != "solo" {
		t.Fatalf("Error() = %q", got)
	}
}
