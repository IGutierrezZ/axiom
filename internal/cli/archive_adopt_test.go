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
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Las pruebas de `axiom archive adopt <scan|check>` (RF-ADP-01 a RF-ADP-06). Todos los
// textos son inventados.

const (
	adoptCLIChange = "adopcion-requisitos"

	adoptCLIRequisitos = "# Requisitos\n\nIntroducción.\n\n## RF-001 Alta de usuario\n\nEl sistema permite el alta.\n\n## RF-002 Baja de usuario\n\nEl sistema permite la baja.\n"

	adoptCLIDelta = `# Delta para proyecto

## ADDED Requirements

### Requirement: Alta de usuario (RF-001)

El sistema DEBE permitir el alta de un usuario.

Origen: docs/requisitos.md#rf-001-alta-de-usuario (RF-001)

#### Scenario: Alta correcta
- **DADO** un usuario nuevo
- **CUANDO** se da de alta
- **ENTONCES** queda registrado

### Requirement: Baja de usuario (RF-002)

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

// RF-BRA-20: `adopt scan --change` es un escritor guardado.
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
	declareAdoptPattern(t, f.ws, "sdd:\n  id_pattern: \"RF-<NNN>\"\n")
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

// declareAdoptPattern añade el bloque `sdd` dado al axiom.yaml del fixture.
func declareAdoptPattern(t *testing.T, ws workspacetest.Workspace, block string) {
	t.Helper()
	path := filepath.Join(ws.Master, "axiom.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	workspacetest.WriteFile(t, path, string(data)+block)
}

// RF-ADP-03 y RF-RID-04: `check` mide con el patrón declarado y no escribe nada, axiom.yaml incluido.
func TestArchiveAdoptCheckMeasuresWithTheDeclaredPattern(t *testing.T) {
	f := newAdoptCheckFixture(t)
	declareAdoptPattern(t, f.ws, "sdd:\n  id_pattern: \"RF-<NNN>\"\n")
	before := adoptSnapshot(t, f.ws.Master)

	stdout, err := runArchiveAdoptCLI(t, "check", "--path", f.ws.Master, "--change", adoptCLIChange, "--json")
	if err != nil {
		t.Fatalf("con (RF-001) y RF-<NNN> declarado el cambio pasa: %v\n%s", err, stdout)
	}
	var report adoptcheck.CheckReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	if report.IDPattern != "RF-<NNN>" || !report.IDPatternDeclared || !report.Passed {
		t.Fatalf("informe = %+v", report)
	}
	text, err := runArchiveAdoptCLI(t, "check", "--path", f.ws.Master, "--change", adoptCLIChange)
	if err != nil || !strings.Contains(text, "Patrón de identificadores: RF-<NNN> (declarado en axiom.yaml)") {
		t.Fatalf("el texto abre con el patrón efectivo: %q (%v)", text, err)
	}
	if after := adoptSnapshot(t, f.ws.Master); !reflect.DeepEqual(before, after) {
		t.Fatalf("check no escribe nada, axiom.yaml incluido: %v", contextCLIChanged(before, after))
	}

	// El patrón declarado no cae al defecto: los REQ-n.m de la delta original ya no valen.
	delta := strings.NewReplacer("(RF-001)", "(REQ-40.1)", "(RF-002)", "(REQ-40.2)").Replace(adoptCLIDelta)
	workspacetest.WriteFile(t, filepath.Join(adoptChangeRoot(f.ws), "specs", "proyecto", "spec.md"), delta)
	if _, err := runArchiveAdoptCLI(t, "check", "--path", f.ws.Master, "--change", adoptCLIChange, "--json"); err == nil {
		t.Fatal("con RF-<NNN> declarado, un (REQ-40.1) no es un identificador")
	}
}

func TestArchiveAdoptCheckWithoutDeclarationReportsTheDefaultPattern(t *testing.T) {
	f := newAdoptCheckFixture(t)
	// Sin declaración el cambio no pasa (pattern_undeclared), pero el informe se mide con el defecto.
	stdout, _ := runArchiveAdoptCLI(t, "check", "--path", f.ws.Master, "--change", adoptCLIChange, "--json")
	var report adoptcheck.CheckReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	if report.IDPatternDeclared || report.IDPattern != reqid.Default().Template() {
		t.Fatalf("sin declaración rige el defecto: %+v", report)
	}
}

// RF-RID-08: con el patrón inválido, `check` se niega con el mismo error y la misma
// continuación que el resto de verbos, sin escribir.
func TestArchiveAdoptCheckRefusesAnInvalidDeclaredPattern(t *testing.T) {
	f := newAdoptCheckFixture(t)
	declareAdoptPattern(t, f.ws, "sdd:\n  id_pattern: \"<m><m>\"\n")
	before := adoptSnapshot(t, f.ws.Master)
	_, err := runArchiveAdoptCLI(t, "check", "--path", f.ws.Master, "--change", adoptCLIChange)
	wantValidateContinuation(t, err, "sdd.id_pattern")
	if after := adoptSnapshot(t, f.ws.Master); !reflect.DeepEqual(before, after) {
		t.Fatalf("la negativa no escribe: %v", contextCLIChanged(before, after))
	}
}

// RF-ADP-01: el escaneo propone el patrón de identificadores del cliente (texto y JSON).
func TestArchiveAdoptScanProposesTheIDPattern(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "docs", "requisitos.md"), adoptCLIRequisitos)

	text, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Patrón de identificadores propuesto (sdd.id_pattern; axiom.yaml no se modifica):",
		"plantilla: RF-<NNN>",
		"evidencia: 2 identificador(es)", "prefijo RF ×2", "separador «-» ×2", "sin dominio ×2", "ancho 3 ×2",
		"sdd:", `id_pattern: "RF-<NNN>"`, "conflictos: ninguno",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("el bloque del patrón debe contener %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Patrón declarado en axiom.yaml") {
		t.Errorf("sin declaración no hay línea de patrón declarado:\n%s", text)
	}

	stdout, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	proposed, _ := report["proposed_id_pattern"].(map[string]any)
	if proposed["template"] != "RF-<NNN>" {
		t.Fatalf("proposed_id_pattern = %v", report["proposed_id_pattern"])
	}
	if _, ok := report["declared_id_pattern"]; ok {
		t.Errorf("sin declaración no hay declared_id_pattern: %v", report)
	}
}

func TestArchiveAdoptScanWithConflictsHasNoTemplate(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "docs", "requisitos.md"), "# R\n\n## RF-001 Alta\n\nA.\n\n## RNF-001 Rendimiento\n\nB.\n")
	text, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"sin plantilla: normaliza a mano los identificadores del origen; Axiom no los reescribe", "[mixed_prefixes]"} {
		if !strings.Contains(text, want) {
			t.Errorf("debe contener %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "id_pattern:") || strings.Contains(text, "\n  plantilla: ") {
		t.Errorf("con conflicto de esquema no hay plantilla ni líneas de copia:\n%s", text)
	}
}

func TestArchiveAdoptScanWithoutIDsHasNoPatternBlock(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "docs", "requisitos.md"), "# R\n\n## Alta de usuario\n\nA.\n")
	text, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(text, "Patrón de identificadores propuesto") {
		t.Errorf("sin identificadores ajenos no hay bloque:\n%s", text)
	}
}

func TestArchiveAdoptScanShowsTheDeclaredPattern(t *testing.T) {
	cases := []struct {
		name, template, wantText string
		matches                  bool
		uncovered                int
	}{
		{"coincide", "RF-<NNN>", "Patrón declarado en axiom.yaml: RF-<NNN> (coincide con la propuesta)", true, 0},
		{"difiere", "RF-<m>", "Patrón declarado en axiom.yaml: RF-<m> (difiere de la propuesta)", false, 0},
		{"no reconoce", "HU-<NNN>", "Patrón declarado en axiom.yaml: HU-<NNN> (no reconoce 2 identificador(es): RF-001, RF-002)", false, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ws := workspacetest.Embedded(t, ".")
			workspacetest.WriteFile(t, filepath.Join(ws.Master, "docs", "requisitos.md"), adoptCLIRequisitos)
			declareAdoptPattern(t, ws, "sdd:\n  id_pattern: \""+tc.template+"\"\n")
			text, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master)
			if err != nil || !strings.Contains(text, tc.wantText) {
				t.Fatalf("falta %q (err %v):\n%s", tc.wantText, err, text)
			}
			stdout, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master, "--json")
			if err != nil {
				t.Fatal(err)
			}
			var report adoptScanReport
			if err := json.Unmarshal([]byte(stdout), &report); err != nil {
				t.Fatal(err)
			}
			declared := report.DeclaredIDPattern
			if declared == nil || declared.Template != tc.template || declared.MatchesProposal != tc.matches || len(declared.Uncovered) != tc.uncovered {
				t.Fatalf("declared_id_pattern = %+v", declared)
			}
		})
	}
}

func TestArchiveAdoptScanReportsAnInvalidDeclaredPatternWithoutRefusing(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "docs", "requisitos.md"), adoptCLIRequisitos)
	declareAdoptPattern(t, ws, "sdd:\n  id_pattern: \"<m><m>\"\n")
	text, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master)
	if err != nil {
		t.Fatalf("scan no depende del patrón y no se niega: %v", err)
	}
	if !strings.Contains(text, "axiom workspace validate") || !strings.Contains(text, "plantilla: RF-<NNN>") {
		t.Errorf("debe informar del error y seguir proponiendo:\n%s", text)
	}
	stdout, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var report adoptScanReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	if report.DeclaredIDPatternError == "" || report.DeclaredIDPattern != nil {
		t.Fatalf("informe = %+v", report)
	}
}

// RF-ADP-01: `scan --change` no toca axiom.yaml y no hay opción para escribir el patrón.
func TestArchiveAdoptScanChangeLeavesAxiomYamlUntouched(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "docs", "requisitos.md"), adoptCLIRequisitos)
	axiomYAML := filepath.Join(ws.Master, "axiom.yaml")
	before, err := os.ReadFile(axiomYAML)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master, "--change", adoptCLIChange); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(axiomYAML)
	if err != nil || string(before) != string(after) {
		t.Fatalf("axiom.yaml debe quedar byte a byte igual (err %v)", err)
	}
	if _, err := runArchiveAdoptCLI(t, "scan", "--path", ws.Master, "--write-pattern"); err == nil {
		t.Fatal("no existe --write-pattern")
	}
}
