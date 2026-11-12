package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/livespec"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Las pruebas de `axiom archive integrate --odd <feature>` (REQ-30.1 a REQ-30.4, REQ-30.8).
// Todos los textos son inventados.

const oddFeature = "mi-feature"

const oddFacturacionSpec = `# Especificación: facturacion

## Requirements

### Requirement: Alta de factura (REQ-31.1)

El sistema DEBE dar de alta una factura.
`

const oddAddedDelta = `## ADDED Requirements

### Requirement: Nota de abono (REQ-31.5)

El sistema DEBE emitir notas de abono.

#### Scenario: Abono correcto

- **DADO** una factura emitida
- **CUANDO** se solicita el abono
- **ENTONCES** se emite la nota
`

func oddFence(delta string) string { return "```openspec\n" + delta + "```\n" }

func oddDoc(header, body string) string {
	return "# ODD · " + oddFeature + "\n\n" + header + "## Objetivo\n\nTexto.\n\n## Tareas\n\n- [x] 1.1 Hecho\n\n" + body
}

func oddSpecSection(caps ...[2]string) string {
	text := "## Spec viva\n\n"
	for _, c := range caps {
		text += "### Capacidad: " + c[0] + "\n\n" + oddFence(c[1]) + "\n"
	}
	return text
}

type oddFixture struct {
	ws   workspacetest.Workspace
	root string // workspaceRoot: el repositorio de specs
}

func (f *oddFixture) docPath(feature string) string {
	return filepath.Join(f.root, "odd", "tasks", feature+".md")
}

func (f *oddFixture) specPath(capability string) string {
	return filepath.Join(f.ws.Home, "specs", capability, "spec.md")
}

func (f *oddFixture) writeDoc(t *testing.T, content string) {
	t.Helper()
	workspacetest.WriteFile(t, f.docPath(oddFeature), content)
}

// newODDFixture monta un proyecto embebido con un repositorio git en `main`
// y la especificación de facturación; con onFeatureBranch pasa a la rama que la guarda
// espera para la feature.
func newODDFixture(t *testing.T, onFeatureBranch bool) *oddFixture {
	t.Helper()
	return newODDFixtureWith(t, onFeatureBranch, nil)
}

// newODDFixtureWith añade al primer commit los ficheros dados (rutas relativas al
// workspaceRoot, con `/`); uno de ellos puede ser axiom.yaml para declarar `technical_context`.
func newODDFixtureWith(t *testing.T, onFeatureBranch bool, files map[string]string) *oddFixture {
	t.Helper()
	contextCLIIsolateGit(t)
	captureWriterWarnings(t)
	ws := workspacetest.Embedded(t, ".")
	f := &oddFixture{ws: ws, root: ws.Master}
	workspacetest.WriteFile(t, f.specPath("facturacion"), oddFacturacionSpec)
	for rel, content := range files {
		workspacetest.WriteFile(t, filepath.Join(ws.Master, filepath.FromSlash(rel)), content)
	}
	contextCLIGit(t, ws.Master, "init", "-q")
	contextCLIGit(t, ws.Master, "symbolic-ref", "HEAD", "refs/heads/main")
	contextCLIGit(t, ws.Master, "config", "core.autocrlf", "false")
	contextCLICommit(t, ws.Master, "base")
	if onFeatureBranch {
		f.switchToFeatureBranch(t)
	}
	return f
}

// switchToFeatureBranch pregunta a la guarda cuál es la rama esperada (desde main se
// niega con wrong_branch) y crea esa rama.
func (f *oddFixture) switchToFeatureBranch(t *testing.T) {
	t.Helper()
	contextCLIGit(t, f.root, "switch", "-q", "main")
	f.writeDoc(t, oddDoc("", ""))
	_, err := runODDIntegrate(t, "--odd", oddFeature, "--path", f.root)
	var refusal *branch.RefusalError
	if !errors.As(err, &refusal) || refusal.Code != branch.CodeWrongBranch {
		t.Fatalf("desde main la guarda debe negarse con wrong_branch: %v", err)
	}
	expected := strings.Split(refusal.Details["expected"], ", ")[0]
	if expected == "" {
		t.Fatalf("el rechazo debe nombrar la rama esperada: %+v", refusal)
	}
	contextCLIGit(t, f.root, "switch", "-q", "-c", expected)
	if err := os.Remove(f.docPath(oddFeature)); err != nil {
		t.Fatal(err)
	}
}

func runODDIntegrate(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := RunArchiveIntegrate(args, &out)
	return out.String(), err
}

func (f *oddFixture) run(t *testing.T, extra ...string) (string, error) {
	t.Helper()
	return runODDIntegrate(t, append([]string{"--odd", oddFeature, "--path", f.root}, extra...)...)
}

func decodeODDReport(t *testing.T, stdout string) livespec.Report {
	t.Helper()
	var report livespec.Report
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("la salida no es un informe JSON: %v\n%s", err, stdout)
	}
	return report
}

func readODDFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func oddSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func oddSameSnapshot(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if other, ok := b[k]; !ok || other != v {
			return false
		}
	}
	return true
}

func TestArchiveIntegrateComposesTheDeltaOntoTheLivingSpec(t *testing.T) {
	f := newODDFixture(t, true)
	doc := oddDoc("", oddSpecSection([2]string{"facturacion", oddAddedDelta}))
	f.writeDoc(t, doc)

	out, err := f.run(t, "--json")
	if err != nil {
		t.Fatalf("RunArchiveIntegrate: %v", err)
	}

	spec := readODDFile(t, f.specPath("facturacion"))
	if !strings.HasPrefix(spec, oddFacturacionSpec) || !strings.Contains(spec, "Nota de abono (REQ-31.5)") {
		t.Errorf("la spec viva debe conservar lo existente y añadir el requisito:\n%s", spec)
	}
	if _, err := os.Stat(filepath.Join(f.ws.Home, "changes", oddFeature)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("no se crea ninguna carpeta de cambio SDD (%v)", err)
	}
	for _, name := range []string{"kickoff.yaml", "gates.yaml"} {
		if _, err := os.Stat(filepath.Join(f.ws.Home, "changes", oddFeature, name)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("no se crea %s (%v)", name, err)
		}
	}
	if got := readODDFile(t, f.docPath(oddFeature)); got != doc {
		t.Errorf("el documento ODD no se modifica:\n%s", got)
	}
	report := decodeODDReport(t, out)
	if report.Feature != oddFeature || report.Impact != "delta" || report.NothingToIntegrate ||
		len(report.Capabilities) != 1 || report.Capabilities[0].Outcome != livespec.OutcomeComposed ||
		len(report.Written) != 1 || report.Written[0] != "openspec/specs/facturacion/spec.md" {
		t.Errorf("informe = %+v", report)
	}
}

func TestArchiveIntegrateCreatesANewCapability(t *testing.T) {
	f := newODDFixture(t, true)
	f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"pagos", strings.Replace(oddAddedDelta, "REQ-31.5", "REQ-32.1", 1)})))

	if _, err := f.run(t); err != nil {
		t.Fatalf("RunArchiveIntegrate: %v", err)
	}
	spec := readODDFile(t, f.specPath("pagos"))
	for _, want := range []string{"# Especificación: pagos", "Integrada desde el documento ODD `mi-feature`.", "Nota de abono (REQ-32.1)"} {
		if !strings.Contains(spec, want) {
			t.Errorf("la capacidad nueva debe contener %q:\n%s", want, spec)
		}
	}
}

func TestArchiveIntegrateTwiceLeavesIdenticalBytesAndSaysThereWasNothingToIntegrate(t *testing.T) {
	f := newODDFixture(t, true)
	f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"facturacion", oddAddedDelta})))
	if _, err := f.run(t); err != nil {
		t.Fatalf("primera integración: %v", err)
	}
	before := oddSnapshot(t, f.root)

	out, err := f.run(t)
	if err != nil {
		t.Fatalf("segunda integración: %v", err)
	}
	if !oddSameSnapshot(before, oddSnapshot(t, f.root)) {
		t.Error("repetir la integración no cambia ningún byte")
	}
	if !strings.Contains(out, "No había nada que integrar") {
		t.Errorf("la salida debe decir que no había nada que integrar:\n%s", out)
	}
	jsonOut, err := f.run(t, "--json")
	if err != nil || !decodeODDReport(t, jsonOut).NothingToIntegrate {
		t.Errorf("--json debe informar nothingToIntegrate (err %v):\n%s", err, jsonOut)
	}
}

func TestArchiveIntegrateDocumentWithoutSectionsIntegratesNothing(t *testing.T) {
	f := newODDFixture(t, true)
	f.writeDoc(t, oddDoc("", ""))
	before := oddSnapshot(t, f.root)

	out, err := f.run(t)
	if err != nil {
		t.Fatalf("RunArchiveIntegrate: %v", err)
	}
	if !oddSameSnapshot(before, oddSnapshot(t, f.root)) || !strings.Contains(out, "No había nada que integrar") {
		t.Errorf("un documento sin la sección no integra nada:\n%s", out)
	}
}

func TestArchiveIntegrateRefusesWithoutWriting(t *testing.T) {
	tests := []struct {
		name    string
		doc     string
		want    []string
		noDoc   bool
		feature string
	}{
		{name: "documento inexistente", noDoc: true, want: []string{filepath.Join("odd", "tasks", oddFeature+".md"), "axiom branch ensure"}},
		{name: "capacidad sin bloque vallado", doc: oddDoc("", "## Spec viva\n\n### Capacidad: facturacion\n\nSolo texto.\n"), want: []string{"facturacion", "no_fence", "axiom archive integrate --odd"}},
		{
			name: "todo o nada entre capacidades",
			doc: oddDoc("", oddSpecSection(
				[2]string{"facturacion", oddAddedDelta},
				[2]string{"pagos", "## MODIFIED Requirements\n\n### Requirement: Inexistente (REQ-32.9)\n\nCuerpo.\n"},
			)),
			want: []string{"pagos", "MODIFIED"},
		},
		{
			name: "colisión de identificador",
			doc:  oddDoc("", oddSpecSection([2]string{"pagos", strings.Replace(oddAddedDelta, "REQ-31.5", "REQ-31.1", 1)})),
			want: []string{"REQ-31.1", "facturacion", "REQ-31.2"},
		},
		{
			name: "requisito ya presente con otro cuerpo",
			doc:  oddDoc("", oddSpecSection([2]string{"facturacion", "## ADDED Requirements\n\n### Requirement: Alta de factura (REQ-31.1)\n\nOtro cuerpo.\n"})),
			want: []string{"Alta de factura (REQ-31.1)", "facturacion"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newODDFixture(t, true)
			if !tt.noDoc {
				f.writeDoc(t, tt.doc)
			}
			before := oddSnapshot(t, f.root)

			out, err := f.run(t)
			if err == nil {
				t.Fatalf("se esperaba un rechazo; salida:\n%s", out)
			}
			for _, want := range tt.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("el rechazo %q debe contener %q", err, want)
				}
			}
			if !oddSameSnapshot(before, oddSnapshot(t, f.root)) {
				t.Error("un rechazo no escribe nada: ni en openspec/specs ni en el índice")
			}
		})
	}
}

func TestArchiveIntegrateRegeneratesTheIndexOnlyWhenItExists(t *testing.T) {
	f := newODDFixture(t, true)
	workspacetest.WriteFile(t, filepath.Join(f.ws.Home, "INDEX.md"), "# Índice viejo\n")
	f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"facturacion", oddAddedDelta})))

	out, err := f.run(t, "--json")
	if err != nil {
		t.Fatalf("RunArchiveIntegrate: %v", err)
	}
	if index := readODDFile(t, filepath.Join(f.ws.Home, "INDEX.md")); strings.Contains(index, "Índice viejo") || !strings.Contains(index, "facturacion") {
		t.Errorf("el índice debe regenerarse:\n%s", index)
	}
	report := decodeODDReport(t, out)
	if last := report.Written[len(report.Written)-1]; last != "openspec/INDEX.md" {
		t.Errorf("written = %v", report.Written)
	}
}

func TestArchiveIntegrateGuardRefusesOutsideTheFeatureBranch(t *testing.T) {
	f := newODDFixture(t, false)
	f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"facturacion", oddAddedDelta})))
	before := oddSnapshot(t, f.root)

	_, err := f.run(t)

	var refusal *branch.RefusalError
	if !errors.As(err, &refusal) || refusal.Code != branch.CodeWrongBranch {
		t.Fatalf("error = %v, se esperaba el rechazo tipado wrong_branch", err)
	}
	for _, want := range []string{"main", "axiom branch ensure", "--lane odd"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el rechazo %q debe contener %q", err, want)
		}
	}
	if !oddSameSnapshot(before, oddSnapshot(t, f.root)) {
		t.Error("la guarda no deja escribir")
	}
}

func TestArchiveIntegrateArgumentErrorsNameTheContinuation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"sin --odd", nil, []string{"--odd", "axiom archive integrate --odd <feature>"}},
		{"--odd vacío", []string{"--odd", " "}, []string{"--odd", "axiom archive integrate --odd <feature>"}},
		{"feature con separador", []string{"--odd", "../otra"}, []string{"no es un nombre de feature válido", "axiom archive integrate --odd <feature>"}},
		{"argumento posicional", []string{"--odd", oddFeature, "sobra"}, []string{"sobra", "axiom archive integrate --odd <feature>"}},
		{"bandera desconocida", []string{"--odd", oddFeature, "--nada"}, []string{"axiom archive integrate --help"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runODDIntegrate(t, tt.args...)
			if err == nil {
				t.Fatal("se esperaba un error")
			}
			for _, want := range tt.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("el error %q debe contener %q", err, want)
				}
			}
		})
	}
}

func TestArchiveIntegrateCwdIsAnAliasOfPath(t *testing.T) {
	f := newODDFixture(t, true)
	f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"facturacion", oddAddedDelta})))

	if _, err := runODDIntegrate(t, "--odd", oddFeature, "--cwd", f.root); err != nil {
		t.Fatalf("--cwd como alias: %v", err)
	}
	other := t.TempDir()
	_, err := runODDIntegrate(t, "--odd", oddFeature, "--path", f.root, "--cwd", other)
	if err == nil || !strings.Contains(err.Error(), "--cwd es un alias de --path") {
		t.Errorf("--path y --cwd distintos deben fallar: %v", err)
	}
}

func TestArchiveIntegrateHelp(t *testing.T) {
	out, err := runODDIntegrate(t, "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Uso: axiom archive integrate --odd <feature>", "--path", "--json", "odd/tasks/<feature>.md"} {
		if !strings.Contains(out, want) {
			t.Errorf("la ayuda debe contener %q:\n%s", want, out)
		}
	}
}

// REQ-30.21: un `baseline:` residual se rechaza sin escribir con el error tipado de
// su retirada y la migración `axiom archive adopt`.
func TestArchiveIntegrateRefusesWhenABaselineIsRetired(t *testing.T) {
	f := newODDFixture(t, true)
	config := readODDFile(t, filepath.Join(f.root, "axiom.yaml")) + contextCLIBaseline
	workspacetest.WriteFile(t, filepath.Join(f.root, "03_Modelo_Datos.md"), "# Modelo\n\n## Entidades\n")
	workspacetest.WriteFile(t, filepath.Join(f.root, "axiom.yaml"), config)
	f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"facturacion", oddAddedDelta})))
	before := oddSnapshot(t, f.root)

	_, err := f.run(t)
	var retired *workspace.BaselineRetiredError
	if !errors.As(err, &retired) || !strings.Contains(err.Error(), "axiom archive adopt scan") {
		t.Fatalf("error = %v, se esperaba un *workspace.BaselineRetiredError con la migración", err)
	}
	if !oddSameSnapshot(before, oddSnapshot(t, f.root)) {
		t.Error("el rechazo no escribe nada")
	}
}

// REQ-30.7: un proyecto que declara `sdd.iso: true` rechaza la integración ODD con
// un motivo tipado que remite al carril SDD, antes de leer el documento ODD y sin
// escribir nada; la spec viva no cambia.
func TestArchiveIntegrateRefusesAnISOProject(t *testing.T) {
	f := newODDFixture(t, true)
	config := readODDFile(t, filepath.Join(f.root, "axiom.yaml")) + "sdd:\n  iso: true\n"
	workspacetest.WriteFile(t, filepath.Join(f.root, "axiom.yaml"), config)
	f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"facturacion", oddAddedDelta})))
	before := oddSnapshot(t, f.root)

	_, err := f.run(t)
	var iso *livespec.ISOProjectError
	if !errors.As(err, &iso) {
		t.Fatalf("error = %v, se esperaba un *livespec.ISOProjectError", err)
	}
	for _, want := range []string{"sdd.iso: true", "perfil de bug", "axiom sdd kickoff seal --cwd"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el rechazo %q debe contener %q", err, want)
		}
	}
	if !oddSameSnapshot(before, oddSnapshot(t, f.root)) {
		t.Error("el rechazo no escribe nada: openspec/specs/ no cambia")
	}
}

// REQ-30.7: la negativa ISO precede a la lectura del documento ODD: un proyecto ISO
// sin documento se rechaza por ISO, no por el documento ausente.
func TestArchiveIntegrateISORefusalPrecedesReadingTheDocument(t *testing.T) {
	f := newODDFixture(t, true)
	config := readODDFile(t, filepath.Join(f.root, "axiom.yaml")) + "sdd:\n  iso: true\n"
	workspacetest.WriteFile(t, filepath.Join(f.root, "axiom.yaml"), config)

	_, err := f.run(t)
	var iso *livespec.ISOProjectError
	if !errors.As(err, &iso) {
		t.Fatalf("error = %v, se esperaba un *livespec.ISOProjectError antes de leer el documento", err)
	}
}

// REQ-30.8: en multirepo opera sobre el repositorio de specs, desde la carpeta maestra, y
// no modifica ningún repositorio de código.
func TestArchiveIntegrateMultirepoWritesTheSpecsRepository(t *testing.T) {
	contextCLIIsolateGit(t)
	captureWriterWarnings(t)
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "odd", "tasks", oddFeature+".md"),
		oddDoc("", oddSpecSection([2]string{"pagos", strings.Replace(oddAddedDelta, "REQ-31.5", "REQ-32.1", 1)})))
	workspacetest.WriteFile(t, filepath.Join(ws.Backend, "src", "main.txt"), "código\n")
	workspacetest.WriteFile(t, filepath.Join(ws.Frontend, "app.txt"), "interfaz\n")
	for _, dir := range []string{ws.Specs, ws.Backend, ws.Frontend} {
		contextCLIGit(t, dir, "init", "-q")
		contextCLIGit(t, dir, "symbolic-ref", "HEAD", "refs/heads/main")
		contextCLIGit(t, dir, "config", "core.autocrlf", "false")
		contextCLICommit(t, dir, "base")
	}
	backendBefore, frontendBefore := oddSnapshot(t, ws.Backend), oddSnapshot(t, ws.Frontend)

	out, err := runODDIntegrate(t, "--odd", oddFeature, "--path", ws.Master, "--json")
	if err != nil {
		t.Fatalf("RunArchiveIntegrate desde la carpeta maestra: %v", err)
	}

	if spec := readODDFile(t, filepath.Join(ws.Home, "specs", "pagos", "spec.md")); !strings.Contains(spec, "REQ-32.1") {
		t.Errorf("la spec viva del repositorio de specs debe contener el requisito:\n%s", spec)
	}
	if !oddSameSnapshot(backendBefore, oddSnapshot(t, ws.Backend)) || !oddSameSnapshot(frontendBefore, oddSnapshot(t, ws.Frontend)) {
		t.Error("ningún repositorio de código cambia")
	}
	if report := decodeODDReport(t, out); len(report.Written) != 1 || report.Written[0] != "openspec/specs/pagos/spec.md" {
		t.Errorf("informe = %+v", report)
	}
}

func TestArchiveIntegrateOutsideGitIsNotRefusedByTheGuard(t *testing.T) {
	captureWriterWarnings(t)
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "specs", "facturacion", "spec.md"), oddFacturacionSpec)
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "odd", "tasks", oddFeature+".md"),
		oddDoc("", oddSpecSection([2]string{"facturacion", oddAddedDelta})))

	if _, err := runODDIntegrate(t, "--odd", oddFeature, "--path", ws.Master); err != nil {
		t.Fatalf("fuera de git la guarda no comprueba la rama: %v", err)
	}
}

// Impacto de los bugs y contexto técnico (REQ-30.5, REQ-30.6).

const oddContextConfig = `workspace:
  name: ejemplo-embebido
  topology: monorepo-embedded
  specs_repository: "."
roles:
  core:
    name: Core
    repositories:
      - path: .
technical_context:
  maintained:
    paths:
      - context/
  reference:
    paths:
      - referencia/
`

func newODDContextFixture(t *testing.T) *oddFixture {
	t.Helper()
	return newODDFixtureWith(t, true, map[string]string{
		"axiom.yaml":              oddContextConfig,
		"context/ARQUITECTURA.md": "# Arquitectura\n\nTexto.\n",
		"referencia/ADR.md":       "# ADR\n\nTexto.\n",
	})
}

const oddValidMarker = "## Impacto en la spec\n\nSin impacto en la spec\nMotivo: el fallo era un error tipográfico en un mensaje\n"

func TestArchiveIntegrateBugImpactContract(t *testing.T) {
	rejected := []struct {
		name string
		body string
		want []string
	}{
		{"sin delta ni marcador", "", []string{"## Impacto en la spec", "Sin impacto en la spec", "Motivo:"}},
		{"Motivo vacío", "## Impacto en la spec\n\nSin impacto en la spec\nMotivo:\n", []string{"## Impacto en la spec", "impact_reason_missing"}},
		{"punto final", "## Impacto en la spec\n\nSin impacto en la spec.\nMotivo: razón\n", []string{"## Impacto en la spec", "impact_marker_invalid"}},
	}
	for _, tt := range rejected {
		t.Run("rechaza "+tt.name, func(t *testing.T) {
			f := newODDFixture(t, true)
			f.writeDoc(t, oddDoc("Tipo: bug\n\n", tt.body))
			before := oddSnapshot(t, f.root)

			out, err := f.run(t)
			if err == nil {
				t.Fatalf("se esperaba un rechazo; salida:\n%s", out)
			}
			for _, want := range append(tt.want, "axiom archive integrate --odd") {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("el rechazo %q debe contener %q", err, want)
				}
			}
			if !oddSameSnapshot(before, oddSnapshot(t, f.root)) {
				t.Error("un rechazo no escribe nada")
			}
		})
	}

	t.Run("acepta el marcador válido sin cambiar la spec", func(t *testing.T) {
		f := newODDFixture(t, true)
		f.writeDoc(t, oddDoc("Tipo: bug\n\n", oddValidMarker))
		before := oddSnapshot(t, f.root)

		out, err := f.run(t, "--json")
		if err != nil {
			t.Fatalf("RunArchiveIntegrate: %v", err)
		}
		report := decodeODDReport(t, out)
		if report.Kind != livespec.KindBug || report.Impact != livespec.ImpactNone || !report.NothingToIntegrate || report.Reason == "" || len(report.Written) != 0 {
			t.Errorf("informe = %+v", report)
		}
		if !oddSameSnapshot(before, oddSnapshot(t, f.root)) {
			t.Error("con el marcador la spec viva no cambia")
		}
		text, err := f.run(t)
		if err != nil || !strings.Contains(text, "motivo: el fallo era un error tipográfico") {
			t.Errorf("la salida legible debe informar del motivo (err %v):\n%s", err, text)
		}
	})

	t.Run("el marcador junto con una delta se ignora y se integra la delta", func(t *testing.T) {
		f := newODDFixture(t, true)
		f.writeDoc(t, oddDoc("Tipo: bug\n\n", oddSpecSection([2]string{"facturacion", oddAddedDelta})+oddValidMarker))

		out, err := f.run(t, "--json")
		if err != nil {
			t.Fatalf("RunArchiveIntegrate: %v", err)
		}
		if report := decodeODDReport(t, out); report.Impact != livespec.ImpactDelta || report.NothingToIntegrate {
			t.Errorf("informe = %+v: el marcador se ignora y el documento se evalúa como delta", report)
		}
		if spec := readODDFile(t, f.specPath("facturacion")); !strings.Contains(spec, "Nota de abono (REQ-31.5)") {
			t.Errorf("se integra la delta:\n%s", spec)
		}
	})

	t.Run("un incremento sin impacto declarado se acepta", func(t *testing.T) {
		f := newODDFixture(t, true)
		f.writeDoc(t, oddDoc("", ""))
		if _, err := f.run(t); err != nil {
			t.Fatalf("RunArchiveIntegrate: %v", err)
		}
	})
}

func TestArchiveIntegrateRejectsAReferenceEditWithoutWriting(t *testing.T) {
	f := newODDContextFixture(t)
	f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"facturacion", oddAddedDelta})))
	workspacetest.WriteFile(t, filepath.Join(f.root, "referencia", "ADR.md"), "# ADR\n\nEditado.\n")
	before := oddSnapshot(t, f.root)

	out, err := f.run(t)
	if err == nil {
		t.Fatalf("se esperaba un rechazo; salida:\n%s", out)
	}
	var refErr *livespec.ReferencePathError
	if !errors.As(err, &refErr) {
		t.Fatalf("error = %v, se esperaba *livespec.ReferencePathError", err)
	}
	for _, want := range []string{"referencia/ADR.md", "reference", "axiom archive integrate --odd"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el rechazo %q debe contener %q", err, want)
		}
	}
	if !oddSameSnapshot(before, oddSnapshot(t, f.root)) {
		t.Error("un rechazo del contexto no escribe nada en openspec/specs")
	}
}

func TestArchiveIntegrateReportsTheMaintainedContextTheBranchTouched(t *testing.T) {
	f := newODDContextFixture(t)
	f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"facturacion", oddAddedDelta})))
	workspacetest.WriteFile(t, filepath.Join(f.root, "context", "ARQUITECTURA.md"), "# Arquitectura\n\nEditada.\n")

	out, err := f.run(t, "--json")
	if err != nil {
		t.Fatalf("RunArchiveIntegrate: %v", err)
	}
	report := decodeODDReport(t, out)
	if report.Context == nil || !report.Context.Declared ||
		!reflect.DeepEqual(report.Context.MaintainedTouched, []string{"context/ARQUITECTURA.md"}) || len(report.Context.ReferenceTouched) != 0 {
		t.Errorf("context = %+v", report.Context)
	}
	if !strings.Contains(out, `"referenceTouched": []`) && !strings.Contains(out, `"referenceTouched":[]`) {
		t.Errorf("referenceTouched debe ser una lista vacía en el JSON:\n%s", out)
	}
	if spec := readODDFile(t, f.specPath("facturacion")); !strings.Contains(spec, "Nota de abono (REQ-31.5)") {
		t.Errorf("el informe del contexto no bloquea la integración:\n%s", spec)
	}
}

func TestArchiveIntegrateTextOutputSaysWhichContextWasTouched(t *testing.T) {
	f := newODDContextFixture(t)
	f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"facturacion", oddAddedDelta})))

	untouched, err := f.run(t)
	if err != nil {
		t.Fatalf("RunArchiveIntegrate: %v", err)
	}
	if !strings.Contains(untouched, "no tocó ninguna ruta mantenida") {
		t.Errorf("la salida debe decir que no se tocó ninguna ruta mantenida:\n%s", untouched)
	}

	f2 := newODDContextFixture(t)
	f2.writeDoc(t, oddDoc("", oddSpecSection([2]string{"facturacion", oddAddedDelta})))
	workspacetest.WriteFile(t, filepath.Join(f2.root, "context", "ARQUITECTURA.md"), "# Arquitectura\n\nEditada.\n")
	touched, err := f2.run(t)
	if err != nil {
		t.Fatalf("RunArchiveIntegrate: %v", err)
	}
	if !strings.Contains(touched, "context/ARQUITECTURA.md") {
		t.Errorf("la salida debe nombrar la ruta mantenida tocada:\n%s", touched)
	}
}

func TestArchiveIntegrateWithoutTechnicalContextHasNoContextReport(t *testing.T) {
	f := newODDFixture(t, true)
	f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"facturacion", oddAddedDelta})))

	out, err := f.run(t, "--json")
	if err != nil {
		t.Fatalf("RunArchiveIntegrate: %v", err)
	}
	if strings.Contains(out, `"context"`) {
		t.Errorf("sin technical_context el informe no lleva `context`:\n%s", out)
	}
	text, err := f.run(t)
	if err != nil || strings.Contains(text, "ontexto") {
		t.Errorf("sin technical_context la salida no habla del contexto (err %v):\n%s", err, text)
	}
}
