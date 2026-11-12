package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/adoption"
	"github.com/IGutierrezZ/axiom/v3/internal/adoption/adoptcheck"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Las pruebas de `axiom archive adopt <scan|check>` (REQ-30.9 a REQ-30.14). Todos los
// textos son inventados.

const (
	adoptCLIChange = "adopcion-requisitos"

	adoptCLIRequisitos = "# Requisitos\n\nIntroducción.\n\n## RF-001 Alta de usuario\n\nEl sistema permite el alta.\n\n## RF-002 Baja de usuario\n\nEl sistema permite la baja.\n"

	adoptCLIDelta = `# Delta para proyecto

## ADDED Requirements

### Requirement: Alta de usuario (REQ-40.1)

El sistema DEBE permitir el alta de un usuario.

Origen: docs/requisitos.md#rf-001-alta-de-usuario (RF-001)

#### Scenario: Alta correcta
- **DADO** un usuario nuevo
- **CUANDO** se da de alta
- **ENTONCES** queda registrado

### Requirement: Baja de usuario (REQ-40.2)

El sistema DEBE permitir la baja de un usuario.

Origen: docs/requisitos.md#rf-002-baja-de-usuario (RF-002)

#### Scenario: Baja correcta
- **DADO** un usuario existente
- **CUANDO** se da de baja
- **ENTONCES** deja de existir
`
	adoptCLIDiscards = "# Adopción\n\n## Descartado\n\n- docs/requisitos.md#requisitos: es el título del documento\n"
)

func runArchiveAdoptCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	captureWriterWarnings(t)
	var out bytes.Buffer
	err := RunArchiveAdopt(args, &out)
	return out.String(), err
}

func adoptSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
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
		relative, _ := filepath.Rel(root, path)
		files[filepath.ToSlash(relative)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func decodeAdoptInventory(t *testing.T, stdout string) adoption.Inventory {
	t.Helper()
	var inv adoption.Inventory
	if err := json.Unmarshal([]byte(stdout), &inv); err != nil {
		t.Fatalf("la salida no es un inventario JSON: %v\n%s", err, stdout)
	}
	return inv
}

func adoptChangeRoot(ws workspacetest.Workspace) string {
	return filepath.Join(ws.Home, "changes", adoptCLIChange)
}

func TestArchiveAdoptScanWithoutChangeWritesNothing(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "docs", "requisitos.md"), adoptCLIRequisitos)
	before := adoptSnapshot(t, ws.Master)

	stdout, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master, "--json")
	if err != nil {
		t.Fatalf("archive adopt scan: %v\n%s", err, stdout)
	}
	inv := decodeAdoptInventory(t, stdout)
	if inv.Schema != adoption.InventorySchema || len(inv.Sources) != 1 || inv.Sources[0].Path != "docs/requisitos.md" || len(inv.Sources[0].Units) != 3 {
		t.Fatalf("inventario = %+v", inv)
	}
	if after := adoptSnapshot(t, ws.Master); !reflect.DeepEqual(before, after) {
		t.Fatalf("scan sin --change no escribe: %v", contextCLIChanged(before, after))
	}

	again, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master, "--json")
	if err != nil || again != stdout {
		t.Fatalf("dos pasadas distintas (err %v):\n%s\n%s", err, stdout, again)
	}

	text, err := runArchiveAdoptCLI(t, "scan", "--cwd", ws.Master)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"docs/requisitos.md", "RF-001", "RF-002", "rf-001-alta-de-usuario", "No se ha escrito ningún fichero"} {
		if !strings.Contains(text, want) {
			t.Errorf("la salida legible debe contener %q:\n%s", want, text)
		}
	}
}

func TestArchiveAdoptScanWithoutOriginsSucceedsAndSaysSo(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	before := adoptSnapshot(t, ws.Master)
	for _, args := range [][]string{
		{"scan", "--path", ws.Master},
		{"scan", "--path", ws.Master, "--change", adoptCLIChange},
	} {
		stdout, err := runArchiveAdoptCLI(t, args...)
		if err != nil {
			t.Fatalf("sin orígenes la orden sale con 0: %v", err)
		}
		if !strings.Contains(stdout, "No hay orígenes candidatos") {
			t.Errorf("debe informar de que no hay orígenes:\n%s", stdout)
		}
	}
	if after := adoptSnapshot(t, ws.Master); !reflect.DeepEqual(before, after) {
		t.Fatalf("sin orígenes no se escribe nada: %v", contextCLIChanged(before, after))
	}
}

func TestArchiveAdoptScanWithChangeWritesOnlyTheInventory(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "docs", "requisitos.md"), adoptCLIRequisitos)
	before := adoptSnapshot(t, ws.Master)

	stdout, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master, "--change", adoptCLIChange)
	if err != nil {
		t.Fatalf("archive adopt scan --change: %v\n%s", err, stdout)
	}
	changed := contextCLIChanged(before, adoptSnapshot(t, ws.Master))
	want := []string{"openspec/changes/" + adoptCLIChange + "/" + adoption.InventoryFileName}
	if !reflect.DeepEqual(changed, want) {
		t.Fatalf("ficheros escritos = %v, quería solo %v", changed, want)
	}
	inv, err := adoption.ReadInventory(adoptChangeRoot(ws))
	if err != nil || len(inv.Sources) != 1 {
		t.Fatalf("inventario escrito = %+v, err = %v", inv, err)
	}
	if !strings.Contains(stdout, adoption.InventoryFileName) {
		t.Errorf("la salida debe citar el inventario escrito:\n%s", stdout)
	}

	// Idempotente: repetirlo deja el mismo inventario byte a byte.
	first := adoptSnapshot(t, ws.Master)
	if _, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master, "--change", adoptCLIChange); err != nil {
		t.Fatal(err)
	}
	if second := adoptSnapshot(t, ws.Master); !reflect.DeepEqual(first, second) {
		t.Fatalf("la segunda pasada cambia el inventario: %v", contextCLIChanged(first, second))
	}
}

// REQ-25.19: `adopt scan --change` es un escritor guardado.
func TestArchiveAdoptScanChangeRefusesOutsideTheChangeBranch(t *testing.T) {
	ws := guardProject(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "docs", "requisitos.md"), adoptCLIRequisitos)
	before := consentCLISnapshot(t, guardChangeRoot(ws))

	_, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master, "--change", sealChange)

	wantWrongBranch(t, err, "main", "feat/x", ws.Master)
	wantNoWrite(t, ws, before)
	if _, statErr := os.Stat(filepath.Join(guardChangeRoot(ws), adoption.InventoryFileName)); !os.IsNotExist(statErr) {
		t.Errorf("la negativa no debe dejar inventario (%v)", statErr)
	}

	t.Run("sin --change no se guarda", func(t *testing.T) {
		if _, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master); err != nil {
			t.Fatalf("el escaneo de solo lectura no se niega: %v", err)
		}
	})
	t.Run("en la rama del cambio escribe", func(t *testing.T) {
		integrateGit(t, ws.Master, "switch", "-q", "feat/x")
		if _, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master, "--change", sealChange); err != nil {
			t.Fatalf("en la rama correcta escribe con normalidad: %v", err)
		}
		if _, err := adoption.ReadInventory(guardChangeRoot(ws)); err != nil {
			t.Errorf("falta el inventario: %v", err)
		}
	})
}

func TestArchiveAdoptScanMultirepoInventoriesTheSpecsRepository(t *testing.T) {
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "docs", "requisitos.md"), adoptCLIRequisitos)
	workspacetest.WriteFile(t, filepath.Join(ws.Backend, "docs", "prd-backend.md"), adoptCLIRequisitos)

	for name, start := range map[string]string{"carpeta maestra": ws.Master, "repositorio de código": ws.Backend} {
		t.Run(name, func(t *testing.T) {
			stdout, err := runArchiveAdoptCLI(t, "scan", "--path", start, "--json")
			if err != nil {
				t.Fatalf("archive adopt scan: %v\n%s", err, stdout)
			}
			inv := decodeAdoptInventory(t, stdout)
			if len(inv.Sources) != 1 || inv.Sources[0].Path != "docs/requisitos.md" {
				t.Fatalf("orígenes = %+v, quería solo los del repositorio de especificaciones", inv.Sources)
			}
		})
	}
}

func TestArchiveAdoptScanMonorepoEmbeddedInventoriesTheCurrentRepository(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "docs", "requisitos.md"), adoptCLIRequisitos)
	stdout, err := runArchiveAdoptCLI(t, "scan", "--path", filepath.Join(ws.Master, "docs"), "--json")
	if err != nil {
		t.Fatal(err)
	}
	if inv := decodeAdoptInventory(t, stdout); len(inv.Sources) != 1 {
		t.Fatalf("el inventario cubre el repositorio actual: %+v", inv.Sources)
	}
}

type adoptCheckFixture struct{ ws workspacetest.Workspace }

// newAdoptCheckFixture es un monorepo embebido con un origen, su inventario escrito con la
// propia CLI y un cambio de adopción válido.
func newAdoptCheckFixture(t *testing.T) adoptCheckFixture {
	t.Helper()
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "docs", "requisitos.md"), adoptCLIRequisitos)
	if _, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master, "--change", adoptCLIChange); err != nil {
		t.Fatalf("scan --change: %v", err)
	}
	workspacetest.WriteFile(t, filepath.Join(adoptChangeRoot(ws), "specs", "proyecto", "spec.md"), adoptCLIDelta)
	workspacetest.WriteFile(t, filepath.Join(adoptChangeRoot(ws), adoption.DiscardsFileName), adoptCLIDiscards)
	return adoptCheckFixture{ws: ws}
}

func TestArchiveAdoptCheckValidChangePasses(t *testing.T) {
	f := newAdoptCheckFixture(t)
	before := adoptSnapshot(t, f.ws.Master)
	stdout, err := runArchiveAdoptCLI(t, "check", "--path", f.ws.Master, "--change", adoptCLIChange, "--json")
	if err != nil {
		t.Fatalf("archive adopt check: %v\n%s", err, stdout)
	}
	var report adoptcheck.CheckReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	if !report.Passed || len(report.Failures) != 0 {
		t.Fatalf("informe = %+v", report)
	}
	if after := adoptSnapshot(t, f.ws.Master); !reflect.DeepEqual(before, after) {
		t.Fatalf("check no escribe nada (tampoco registra compuertas): %v", contextCLIChanged(before, after))
	}
	text, err := runArchiveAdoptCLI(t, "check", "--cwd", f.ws.Master, "--change", adoptCLIChange)
	if err != nil || !strings.Contains(text, "superada") {
		t.Fatalf("salida legible = %q, err = %v", text, err)
	}
}

func TestArchiveAdoptCheckReportsEveryFailureAndExitsNonZero(t *testing.T) {
	f := newAdoptCheckFixture(t)
	delta := strings.Replace(adoptCLIDelta, "#### Scenario: Alta correcta", "Sin escenario", 1)
	workspacetest.WriteFile(t, filepath.Join(adoptChangeRoot(f.ws), "specs", "proyecto", "spec.md"), delta)
	workspacetest.WriteFile(t, filepath.Join(adoptChangeRoot(f.ws), adoption.DiscardsFileName), "## Descartado\n")
	before := adoptSnapshot(t, f.ws.Master)

	stdout, err := runArchiveAdoptCLI(t, "check", "--path", f.ws.Master, "--change", adoptCLIChange, "--json")
	if err == nil {
		t.Fatal("con fallos la orden sale distinta de cero")
	}
	if !strings.Contains(err.Error(), "axiom archive adopt check --path") {
		t.Errorf("el error debe nombrar la continuación: %v", err)
	}
	var report adoptcheck.CheckReport
	if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil {
		t.Fatalf("el informe se imprime entero aunque haya fallos: %v\n%s", jsonErr, stdout)
	}
	rules := map[string]bool{}
	for _, failure := range report.Failures {
		rules[failure.Rule] = true
	}
	if report.Passed || !rules[adoptcheck.RuleScenarios] || !rules[adoptcheck.RuleCoverage] {
		t.Fatalf("informe = %+v", report)
	}
	if after := adoptSnapshot(t, f.ws.Master); !reflect.DeepEqual(before, after) {
		t.Fatalf("check no escribe: %v", contextCLIChanged(before, after))
	}

	text, _ := runArchiveAdoptCLI(t, "check", "--path", f.ws.Master, "--change", adoptCLIChange)
	for _, want := range []string{"no superada", "[scenarios]", "[coverage]", "Alta de usuario"} {
		if !strings.Contains(text, want) {
			t.Errorf("la salida legible debe contener %q:\n%s", want, text)
		}
	}
}

func TestArchiveAdoptCheckWithoutInventoryPointsToScan(t *testing.T) {
	f := newAdoptCheckFixture(t)
	if err := os.Remove(filepath.Join(adoptChangeRoot(f.ws), adoption.InventoryFileName)); err != nil {
		t.Fatal(err)
	}
	stdout, err := runArchiveAdoptCLI(t, "check", "--path", f.ws.Master, "--change", adoptCLIChange)
	if err == nil {
		t.Fatal("sin inventario la orden falla")
	}
	if !strings.Contains(stdout, "axiom archive adopt scan --change "+adoptCLIChange) {
		t.Errorf("el informe remite a scan:\n%s", stdout)
	}
}

func TestArchiveAdoptCheckUnknownChangeIsRefusedWithContinuation(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	_, err := runArchiveAdoptCLI(t, "check", "--path", ws.Master, "--change", "no-existe")
	if err == nil || !strings.Contains(err.Error(), "no-existe") || !strings.Contains(err.Error(), "axiom ") {
		t.Fatalf("error = %v", err)
	}
}

func TestArchiveAdoptRejectsBadArgumentsNamingTheContinuation(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{"sin verbo", nil, []string{"scan o check", "axiom archive adopt --help"}},
		{"verbo desconocido", []string{"borrar"}, []string{`"borrar"`, "axiom archive adopt --help"}},
		{"check sin --change", []string{"check", "--path", ws.Master}, []string{"--change", "axiom archive adopt check --change"}},
		{"cambio con ruta", []string{"scan", "--path", ws.Master, "--change", "../x"}, []string{"no es un nombre de cambio válido", "axiom archive adopt scan"}},
		{"argumento posicional", []string{"scan", "sobra"}, []string{`"sobra"`, "axiom archive adopt scan"}},
		{"--path y --cwd distintos", []string{"scan", "--path", ws.Master, "--cwd", ws.Specs + "-otro"}, []string{"--path", "--cwd", "axiom archive adopt scan"}},
		{"ruta inexistente", []string{"scan", "--path", filepath.Join(ws.Master, "no-existe")}, []string{"axiom "}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runArchiveAdoptCLI(t, c.args...)
			if err == nil {
				t.Fatal("se esperaba un error")
			}
			for _, want := range c.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("el error %q debe contener %q", err.Error(), want)
				}
			}
		})
	}
}

func TestArchiveAdoptHelp(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"scan", "--help"}, {"check", "-h"}} {
		stdout, err := runArchiveAdoptCLI(t, args...)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		for _, want := range []string{"Uso: axiom archive adopt <scan|check>", "adoption-inventory.yaml", "--path", "--cwd", "--change"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("la ayuda %v debe contener %q:\n%s", args, want, stdout)
			}
		}
	}
}
