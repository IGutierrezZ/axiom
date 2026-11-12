package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/techcontext"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Las pruebas de `axiom archive context <check|prepare|verify|show>` (REQ-30.15 a
// REQ-30.20, REQ-30.29, REQ-30.30). Todos los textos son inventados.

const contextCLIChange = "inc-99-ejemplo"

const contextCLIConfig = `workspace:
  name: ejemplo
  topology: multirepo
  specs_repository: repo-specs
roles:
  backend:
    name: Backend
    repositories:
      - path: repo-backend
technical_context:
  maintained:
    paths:
      - context/
      - ../repo-backend/docs/
  reference:
    paths:
      - referencia/
`

// contextCLIBaseline es la declaración de una baseline de proyecto: mientras
// `axiom sdd integrate` exista, `archive context` se niega ante ella.
const contextCLIBaseline = `baseline:
  documents:
    - domain: modelo-de-datos
      path: 03_Modelo_Datos.md
`

const (
	contextCLIArquitectura = "# Arquitectura\n\n## Componentes\n\n### Servicio de catálogo\n\nGestiona los productos.\n\n### Servicio de pedidos\n\nGestiona los pedidos.\n"
	contextCLIAPI          = "# API\n\n## Pagos\n\nTexto de pagos.\n"
	contextCLIImpact       = `# Delta de contexto: inc-99-ejemplo

## Entrada: context/ARQUITECTURA.md

- Ancla: Componentes > Servicio de pedidos
- Acción: modify
- Motivo: se añade el reintento de pago

Texto nuevo para la sección.

## Entrada: ../repo-backend/docs/API.md

- Ancla: Errores
- Acción: add
- Motivo: documenta el error del cobro reintentado
`
)

type contextCLIFixture struct {
	ws         workspacetest.Workspace
	changeRoot string
}

func contextCLIIsolateGit(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("la prueba ejecuta git: se omite con -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git no está disponible: %v", err)
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Autor de Prueba")
	t.Setenv("GIT_AUTHOR_EMAIL", "autor@ejemplo.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "Autor de Prueba")
	t.Setenv("GIT_COMMITTER_EMAIL", "autor@ejemplo.invalid")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
}

func contextCLIGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s en %s: %v\n%s", strings.Join(args, " "), dir, err, output)
	}
	return strings.TrimRight(string(output), "\r\n")
}

func contextCLICommit(t *testing.T, dir, message string) {
	t.Helper()
	contextCLIGit(t, dir, "add", "-A")
	contextCLIGit(t, dir, "commit", "-q", "-m", message)
}

func newContextCLIFixture(t *testing.T, withGit, withBaseline bool) *contextCLIFixture {
	t.Helper()
	if withGit {
		contextCLIIsolateGit(t)
	}
	ws := workspacetest.Canonical(t)
	f := &contextCLIFixture{ws: ws, changeRoot: filepath.Join(ws.Home, "changes", contextCLIChange)}
	config := contextCLIConfig
	if withBaseline {
		config += contextCLIBaseline
		workspacetest.WriteFile(t, filepath.Join(ws.Specs, "03_Modelo_Datos.md"), "# Modelo\n\n## Entidades\n")
	}
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), config)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "context", "ARQUITECTURA.md"), contextCLIArquitectura)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "referencia", "legado", "VIEJO.md"), "# Viejo\n")
	workspacetest.WriteFile(t, filepath.Join(ws.Backend, "docs", "API.md"), contextCLIAPI)
	f.writeChange(t, "specs/pagos/spec.md", "## ADDED Requirements\n\n### Requirement: Reintento (REQ-1.1)\n\nEl sistema DEBE reintentar.\n")
	f.writeChange(t, techcontext.DeltaFileName, contextCLIImpact)
	if withGit {
		for _, dir := range []string{ws.Specs, ws.Backend} {
			contextCLIGit(t, dir, "init", "-q")
			contextCLIGit(t, dir, "symbolic-ref", "HEAD", "refs/heads/main")
			contextCLIGit(t, dir, "config", "core.autocrlf", "false")
			contextCLICommit(t, dir, "base")
		}
	}
	return f
}

func (f *contextCLIFixture) writeChange(t *testing.T, rel, content string) {
	t.Helper()
	workspacetest.WriteFile(t, filepath.Join(f.changeRoot, filepath.FromSlash(rel)), content)
}

// applyEdits hace las dos ediciones que el delta de contexto describe.
func (f *contextCLIFixture) applyEdits(t *testing.T) {
	t.Helper()
	workspacetest.WriteFile(t, filepath.Join(f.ws.Specs, "context", "ARQUITECTURA.md"), strings.Replace(contextCLIArquitectura, "Gestiona los pedidos.", "Gestiona los pedidos y reintenta el cobro.", 1))
	workspacetest.WriteFile(t, filepath.Join(f.ws.Backend, "docs", "API.md"), contextCLIAPI+"\n## Errores\n\nError 402.\n")
}

func (f *contextCLIFixture) snapshot(t *testing.T) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(f.ws.Master, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		sum := sha256.Sum256(data)
		relative, _ := filepath.Rel(f.ws.Master, path)
		files[filepath.ToSlash(relative)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func contextCLIChanged(before, after map[string]string) []string {
	var changed []string
	for path, sum := range after {
		if before[path] != sum {
			changed = append(changed, path)
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			changed = append(changed, path+" (borrado)")
		}
	}
	return changed
}

func runArchiveContextCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := RunArchiveContext(args, &out)
	return out.String(), err
}

// withContextStatus parte del status real y lo ajusta, como la costura de las
// demás pruebas de integración.
func withContextStatus(t *testing.T, adjust func(sddstatus.Status) sddstatus.Status) {
	t.Helper()
	previous := resolveContextStatus
	t.Cleanup(func() { resolveContextStatus = previous })
	resolveContextStatus = func(options sddstatus.ResolveOptions) (sddstatus.Status, error) {
		status, err := previous(options)
		if err != nil {
			return status, err
		}
		return adjust(status), nil
	}
}

func authorizeBackend(t *testing.T, f *contextCLIFixture) {
	t.Helper()
	withContextStatus(t, func(status sddstatus.Status) sddstatus.Status {
		status.ActionContext.AllowedEditRoots = append(append([]string{}, status.ActionContext.AllowedEditRoots...), f.ws.Backend)
		return status
	})
}

func decodeContextReport(t *testing.T, stdout string) techcontext.Report {
	t.Helper()
	var report techcontext.Report
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("la salida no es un informe JSON: %v\n%s", err, stdout)
	}
	return report
}

func TestArchiveContextCheckReportsAndWritesNothing(t *testing.T) {
	f := newContextCLIFixture(t, false, false)
	before := f.snapshot(t)
	for name, start := range map[string]string{
		"carpeta maestra":                 f.ws.Master,
		"repositorio de especificaciones": f.ws.Specs,
		"repositorio de código":           filepath.Join(f.ws.Backend, "src"),
	} {
		t.Run(name, func(t *testing.T) {
			stdout, err := runArchiveContextCLI(t, "check", "--path", start, "--change", contextCLIChange, "--json")
			if err != nil {
				t.Fatalf("archive context check: %v\n%s", err, stdout)
			}
			report := decodeContextReport(t, stdout)
			if report.Schema != techcontext.CheckSchema || report.Change != contextCLIChange || len(report.Errors) != 0 {
				t.Fatalf("informe = %+v", report)
			}
			if !report.Context.Declared || report.Context.Decision != techcontext.ContextEntries || len(report.Context.Entries) != 2 {
				t.Errorf("contexto = %+v", report.Context)
			}
			if report.Warnings == nil || report.Errors == nil {
				t.Errorf("warnings y errors son siempre listas: %+v", report)
			}
		})
	}
	if changed := contextCLIChanged(before, f.snapshot(t)); len(changed) != 0 {
		t.Fatalf("check no escribe nada: %v", changed)
	}

	text, err := runArchiveContextCLI(t, "check", "--path", f.ws.Specs, "--change", contextCLIChange)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Contexto técnico del cambio inc-99-ejemplo: check",
		"Contexto técnico mantenido: presente=true, decisión=entries",
		"C1 modify: context/ARQUITECTURA.md > Componentes > Servicio de pedidos",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("falta %q en el informe:\n%s", want, text)
		}
	}
}

func TestArchiveContextCwdIsAnAliasOfPath(t *testing.T) {
	f := newContextCLIFixture(t, false, false)
	for _, args := range [][]string{
		{"check", "--cwd", f.ws.Specs, "--change", contextCLIChange, "--json"},
		{"check", "--path", f.ws.Specs, "--cwd", f.ws.Specs, "--change", contextCLIChange, "--json"},
	} {
		if stdout, err := runArchiveContextCLI(t, args...); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, stdout)
		}
	}
	_, err := runArchiveContextCLI(t, "check", "--path", f.ws.Specs, "--cwd", f.ws.Backend, "--change", contextCLIChange)
	if err == nil {
		t.Fatal("--path y --cwd con valores distintos deben fallar")
	}
	for _, want := range []string{"--path", "--cwd", "axiom archive context check"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error %q debe contener %q", err, want)
		}
	}
}

func TestArchiveContextArgumentErrorsNameTheContinuation(t *testing.T) {
	f := newContextCLIFixture(t, false, false)
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"sin verbo", nil, []string{"check, prepare, verify o show", "axiom archive context --help"}},
		{"verbo desconocido", []string{"borrar"}, []string{`"borrar"`, "axiom archive context --help"}},
		{"sin --change", []string{"check", "--path", f.ws.Specs}, []string{"--change", "axiom archive context check --path"}},
		{"cambio con ruta", []string{"check", "--path", f.ws.Specs, "--change", "../x"}, []string{"no es un nombre de cambio válido", "axiom archive context check"}},
		{"argumento posicional", []string{"check", "--change", contextCLIChange, "sobra"}, []string{`"sobra"`, "axiom archive context check"}},
		{"formato de show", []string{"show", "--path", f.ws.Specs, "--change", contextCLIChange, "--format", "xml"}, []string{"--format", "markdown o json", "axiom archive context show"}},
		{"cambio inexistente", []string{"check", "--path", f.ws.Specs, "--change", "no-existe"}, []string{"no-existe", "axiom sdd status"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runArchiveContextCLI(t, tt.args...)
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

func TestArchiveContextHelp(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		stdout, err := runArchiveContextCLI(t, "check", flag)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"Uso: axiom archive context <check|prepare|verify|show>", "--path <ruta>", "--cwd", "Sale con código 1"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("la ayuda no contiene %q:\n%s", want, stdout)
			}
		}
	}
}

func TestArchiveContextCheckExitsNonZeroWithErrors(t *testing.T) {
	f := newContextCLIFixture(t, false, false)
	f.writeChange(t, techcontext.DeltaFileName, "## Entrada: referencia/legado/VIEJO.md\n\n- Ancla: Viejo\n- Acción: modify\n- Motivo: x\n")

	stdout, err := runArchiveContextCLI(t, "check", "--path", f.ws.Specs, "--change", contextCLIChange, "--json")
	if err == nil {
		t.Fatal("se esperaba un código distinto de cero con una ruta de referencia")
	}
	for _, want := range []string{"archive context check", "1 error(es)", "axiom archive context check --path"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no contiene %q: %v", want, err)
		}
	}
	report := decodeContextReport(t, stdout)
	if len(report.Errors) != 1 || !strings.Contains(report.Errors[0], "ruta de referencia") {
		t.Fatalf("errores = %v", report.Errors)
	}
	text, err := runArchiveContextCLI(t, "check", "--path", f.ws.Specs, "--change", contextCLIChange)
	if err == nil || !strings.Contains(text, "Errores:") {
		t.Errorf("el informe de texto lista los errores (err = %v):\n%s", err, text)
	}
}

func TestArchiveContextPrepareWritesOnlyTheTrace(t *testing.T) {
	f := newContextCLIFixture(t, true, false)
	authorizeBackend(t, f)
	before := f.snapshot(t)

	stdout, err := runArchiveContextCLI(t, "prepare", "--path", f.ws.Master, "--change", contextCLIChange, "--json")
	if err != nil {
		t.Fatalf("archive context prepare: %v\n%s", err, stdout)
	}
	report := decodeContextReport(t, stdout)
	if report.Schema != techcontext.PrepareSchema || len(report.ContextItems) != 2 {
		t.Fatalf("informe = %+v", report)
	}
	if item := report.ContextItems[0]; item.ID != "C1" || item.Text != "Texto nuevo para la sección." {
		t.Errorf("trabajo de edición = %+v", item)
	}
	want := []string{"repo-specs/openspec/changes/" + contextCLIChange + "/" + techcontext.TraceFileName}
	if changed := contextCLIChanged(before, f.snapshot(t)); !reflect.DeepEqual(changed, want) {
		t.Fatalf("prepare solo escribe integration.yaml: cambió %v", changed)
	}

	again, err := runArchiveContextCLI(t, "prepare", "--path", f.ws.Specs, "--change", contextCLIChange)
	if err != nil {
		t.Fatalf("segunda ejecución: %v\n%s", err, again)
	}
	for _, want := range []string{"Edición del contexto:", "C2 add: ../repo-backend/docs/API.md > Errores", "Plan de integración: " + filepath.Join(f.changeRoot, techcontext.TraceFileName)} {
		if !strings.Contains(again, want) {
			t.Errorf("falta %q en el informe:\n%s", want, again)
		}
	}
}

func TestArchiveContextPrepareRefusesWithoutEditAuthorization(t *testing.T) {
	f := newContextCLIFixture(t, true, false)
	before := f.snapshot(t)
	stdout, err := runArchiveContextCLI(t, "prepare", "--path", f.ws.Specs, "--change", contextCLIChange, "--json")
	if err == nil {
		t.Fatal("el repositorio de código no está autorizado")
	}
	report := decodeContextReport(t, stdout)
	if len(report.Errors) != 1 || !strings.Contains(report.Errors[0], "axiom sdd-attempt grant") {
		t.Fatalf("errores = %v", report.Errors)
	}
	if len(report.ContextItems) != 0 {
		t.Errorf("con errores no hay trabajos de edición: %+v", report.ContextItems)
	}
	if changed := contextCLIChanged(before, f.snapshot(t)); len(changed) != 0 {
		t.Fatalf("un rechazo no escribe nada: %v", changed)
	}
}

func TestArchiveContextVerifyAndShow(t *testing.T) {
	f := newContextCLIFixture(t, true, false)
	authorizeBackend(t, f)
	if stdout, err := runArchiveContextCLI(t, "prepare", "--path", f.ws.Specs, "--change", contextCLIChange); err != nil {
		t.Fatalf("prepare: %v\n%s", err, stdout)
	}

	stdout, err := runArchiveContextCLI(t, "verify", "--path", f.ws.Specs, "--change", contextCLIChange, "--json")
	if err == nil {
		t.Fatal("sin las ediciones la verificación debe fallar")
	}
	var failed techcontext.VerifyReport
	if jsonErr := json.Unmarshal([]byte(stdout), &failed); jsonErr != nil || failed.Passed || len(failed.Violations) == 0 {
		t.Fatalf("informe = %s (%v)", stdout, jsonErr)
	}
	if !strings.Contains(err.Error(), "axiom archive context verify --path") {
		t.Errorf("el error nombra la continuación: %v", err)
	}
	text, err := runArchiveContextCLI(t, "verify", "--path", f.ws.Specs, "--change", contextCLIChange)
	if err == nil || !strings.Contains(text, "no superada") || !strings.Contains(text, "[presence]") {
		t.Errorf("el texto lista las violaciones (err = %v):\n%s", err, text)
	}

	f.applyEdits(t)
	text, err = runArchiveContextCLI(t, "verify", "--path", f.ws.Specs, "--change", contextCLIChange)
	if err != nil {
		t.Fatalf("verify: %v\n%s", err, text)
	}
	for _, want := range []string{"superada", "axiom archive context show --path"} {
		if !strings.Contains(text, want) {
			t.Errorf("falta %q:\n%s", want, text)
		}
	}

	markdown, err := runArchiveContextCLI(t, "show", "--path", f.ws.Specs, "--change", contextCLIChange)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## Traza de integración", "| C1 | context/ARQUITECTURA.md | Componentes > Servicio de pedidos | modified | integrated |", "| C2 | ../repo-backend/docs/API.md | Errores | added | integrated |"} {
		if !strings.Contains(markdown, want) {
			t.Errorf("falta %q en la traza:\n%s", want, markdown)
		}
	}
	asJSON, err := runArchiveContextCLI(t, "show", "--path", f.ws.Specs, "--change", contextCLIChange, "--json")
	if err != nil || !strings.Contains(asJSON, `"schema": "axiom.sdd-integration-trace/v1"`) {
		t.Errorf("show --json = %v\n%s", err, asJSON)
	}

	// La traza de un cambio ya archivado con prefijo de fecha se consulta igual.
	archived := filepath.Join(f.ws.Home, "changes", "archive", "2026-03-04-"+contextCLIChange)
	if err := os.MkdirAll(filepath.Dir(archived), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(f.changeRoot, archived); err != nil {
		t.Fatal(err)
	}
	if again, err := runArchiveContextCLI(t, "show", "--path", f.ws.Specs, "--change", contextCLIChange); err != nil || !strings.Contains(again, "Traza de integración") {
		t.Errorf("show de un cambio archivado: %v\n%s", err, again)
	}
}

func TestArchiveContextShowWithoutATraceNamesPrepare(t *testing.T) {
	f := newContextCLIFixture(t, false, false)
	_, err := runArchiveContextCLI(t, "show", "--path", f.ws.Specs, "--change", contextCLIChange)
	if err == nil {
		t.Fatal("un cambio sin traza no se muestra")
	}
	for _, want := range []string{"no tiene traza de integración", "axiom archive context prepare --path"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error %q debe contener %q", err, want)
		}
	}
}

// REQ-30.15 (solo U1–U5): con una baseline declarada, `archive context` se niega
// sin escribir y remite a `axiom sdd integrate`.
func TestArchiveContextRefusesWhenABaselineIsDeclared(t *testing.T) {
	f := newContextCLIFixture(t, false, true)
	before := f.snapshot(t)
	for _, verb := range []string{"check", "prepare", "verify", "show"} {
		t.Run(verb, func(t *testing.T) {
			_, err := runArchiveContextCLI(t, verb, "--path", f.ws.Specs, "--change", contextCLIChange)
			if err == nil {
				t.Fatal("se esperaba la negativa")
			}
			for _, want := range []string{"baseline", "axiom sdd integrate " + verb, "--change " + contextCLIChange} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("la negativa %q debe contener %q", err, want)
				}
			}
		})
	}
	if changed := contextCLIChanged(before, f.snapshot(t)); len(changed) != 0 {
		t.Fatalf("la negativa no escribe nada: %v", changed)
	}
}

// REQ-25.19 (guarda A): `prepare` y `verify` escriben `integration.yaml` y se
// consultan en el contexto `archive`: fuera de la rama de archivado se niegan con
// `wrong_branch`, sin escribir, y con la continuación de `ensure`.
func TestArchiveContextWritersRefuseOutsideTheArchiveBranch(t *testing.T) {
	ws := guardProject(t)
	for _, verb := range []string{"prepare", "verify"} {
		t.Run(verb, func(t *testing.T) {
			integrateGit(t, ws.Master, "switch", "-q", "main")
			before := consentCLISnapshot(t, guardChangeRoot(ws))
			_, err := runArchiveContextCLI(t, verb, "--path", ws.Master, "--change", sealChange)
			wantWrongBranch(t, err, "main", guardArchiveBranch, ws.Master)
			wantContinuation(t, err, " --archive")
			wantNoWrite(t, ws, before)
		})
	}
	t.Run("desde la rama del cambio, que no es la de archivado", func(t *testing.T) {
		integrateGit(t, ws.Master, "switch", "-q", "feat/x")
		_, err := runArchiveContextCLI(t, "prepare", "--path", ws.Master, "--change", sealChange)
		wantWrongBranch(t, err, "feat/x", guardArchiveBranch, ws.Master)
	})
	t.Run("check y show no se guardan", func(t *testing.T) {
		integrateGit(t, ws.Master, "switch", "-q", "main")
		if _, err := runArchiveContextCLI(t, "check", "--path", ws.Master, "--change", sealChange); err != nil {
			wantNotRefusedByTheGuard(t, err)
		}
	})
	t.Run("en la rama de archivado procede", func(t *testing.T) {
		integrateGit(t, ws.Master, "switch", "-q", "-c", guardArchiveBranch)
		_, err := runArchiveContextCLI(t, "prepare", "--path", ws.Master, "--change", sealChange)
		wantNotRefusedByTheGuard(t, err)
		if _, statErr := os.Stat(filepath.Join(guardChangeRoot(ws), techcontext.TraceFileName)); errors.Is(statErr, os.ErrNotExist) {
			t.Errorf("en la rama de archivado prepare escribe el plan (error %v)", err)
		}
	})
}
