package hub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

func multirepoRoles() []RoleInput {
	return []RoleInput{
		{Key: "backend", Name: "Backend", Repositories: []string{"repo-backend"}},
		{Key: "frontend", Name: "Frontend", Repositories: []string{"repo-frontend"}},
	}
}

// G4.2: un alta multirepo deja el axiom.yaml en el repositorio de specs (creado
// sin git init), el puntero en la carpeta maestra y workspace.LocateAt lo resuelve.
func TestInitMultirepoLeavesConfigInSpecsRepoAndPointerInMaster(t *testing.T) {
	master := filepath.Join(t.TempDir(), "maestra")
	res, err := NewInitializer(nil, nil).Init(InitOptions{
		Path: master, Name: "Multi", Topology: "multirepo", SpecsRepository: "repo-specs", Roles: multirepoRoles(),
	})
	if err != nil {
		t.Fatalf("Init multirepo falló: %v", err)
	}

	specs := filepath.Join(master, "repo-specs")
	if want := filepath.Join(specs, "axiom.yaml"); res.ConfigPath != want {
		t.Errorf("ConfigPath = %s, quiero %s", res.ConfigPath, want)
	}
	if _, err := os.Stat(filepath.Join(master, "axiom.yaml")); !os.IsNotExist(err) {
		t.Errorf("la carpeta maestra no debe llevar axiom.yaml (err=%v)", err)
	}
	pointer, err := os.ReadFile(filepath.Join(master, ".axiom-workspace"))
	if err != nil || strings.TrimSpace(string(pointer)) != "config: repo-specs/axiom.yaml" {
		t.Errorf("puntero = %q (err=%v)", pointer, err)
	}
	if _, err := os.Stat(filepath.Join(specs, ".git")); !os.IsNotExist(err) {
		t.Errorf("el alta no debe hacer git init en el repositorio de specs (err=%v)", err)
	}
	for _, dir := range []string{"openspec/specs", "openspec/changes", ".axiom/inbox/skills"} {
		if info, err := os.Stat(filepath.Join(specs, filepath.FromSlash(dir))); err != nil || !info.IsDir() {
			t.Errorf("falta %s en el repositorio de specs (err=%v)", dir, err)
		}
	}
	if _, err := os.Stat(filepath.Join(master, "openspec")); !os.IsNotExist(err) {
		t.Errorf("la carpeta maestra no debe tener un openspec propio (err=%v)", err)
	}

	loc, err := workspace.LocateAt(master)
	if err != nil {
		t.Fatalf("LocateAt no resuelve el alta: %v", err)
	}
	if loc.Origin != workspace.OriginPointer || loc.WorkspaceRoot != specs || loc.PlanningHome != filepath.Join(specs, "openspec") {
		t.Errorf("localización inesperada: origen=%v root=%s home=%s", loc.Origin, loc.WorkspaceRoot, loc.PlanningHome)
	}
	if loc.Config.Workspace.Topology != workspace.TopologyMultirepo || loc.Config.Workspace.SpecsRepository != "repo-specs" {
		t.Errorf("configuración inesperada: %+v", loc.Config.Workspace)
	}
	yamlBytes, _ := os.ReadFile(res.ConfigPath)
	if strings.Contains(string(yamlBytes), "root:") {
		t.Errorf("el multirepo con repositorio propio no declara root:\n%s", yamlBytes)
	}

	// Repetir el alta sobre la misma carpeta es idempotente y no choca con la protección de planning home ajeno.
	if _, err := NewInitializer(nil, nil).Init(InitOptions{
		Path: master, Name: "Multi", Topology: "multirepo", SpecsRepository: "repo-specs",
	}); err != nil {
		t.Errorf("repetir el alta multirepo debe ser idempotente: %v", err)
	}
}

func TestInitMultirepoReusesAnExistingSpecsRepoAndNestedOpenspec(t *testing.T) {
	master := t.TempDir()
	if err := os.MkdirAll(filepath.Join(master, "repos", "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := NewInitializer(nil, nil).Init(InitOptions{Path: master, Topology: "multirepo", SpecsRepository: "repos/specs"}); err != nil {
		t.Fatalf("Init con repositorio de specs existente y anidado falló: %v", err)
	}
	loc, err := workspace.LocateAt(master)
	if err != nil || loc.PlanningHome != filepath.Join(master, "repos", "specs", "openspec") {
		t.Errorf("LocateAt = %+v (err=%v)", loc, err)
	}

	// Si la ruta termina en openspec, ese directorio es el planning home y su padre el repositorio.
	other := t.TempDir()
	if _, err := NewInitializer(nil, nil).Init(InitOptions{Path: other, Topology: "multirepo", SpecsRepository: "repo-specs/openspec"}); err != nil {
		t.Fatalf("Init con specs terminado en openspec falló: %v", err)
	}
	if _, err := os.Stat(filepath.Join(other, "repo-specs", "axiom.yaml")); err != nil {
		t.Errorf("el axiom.yaml debe estar en el padre de openspec: %v", err)
	}
	if loc, err := workspace.LocateAt(other); err != nil || loc.PlanningHome != filepath.Join(other, "repo-specs", "openspec") {
		t.Errorf("LocateAt = %+v (err=%v)", loc, err)
	}
}

func TestInitMonorepoAndLegacyMultirepoAreUnchanged(t *testing.T) {
	embedded := t.TempDir()
	res, err := NewInitializer(nil, nil).Init(InitOptions{Path: embedded, Name: "Emb"})
	if err != nil {
		t.Fatal(err)
	}
	yamlBytes, _ := os.ReadFile(res.ConfigPath)
	if res.ConfigPath != filepath.Join(embedded, "axiom.yaml") || !strings.Contains(string(yamlBytes), `root: "."`) || !strings.Contains(string(yamlBytes), `specs_repository: "openspec"`) {
		t.Errorf("monorepo-embedded cambió:\n%s", yamlBytes)
	}
	if _, err := os.Stat(filepath.Join(embedded, ".axiom-workspace")); !os.IsNotExist(err) {
		t.Errorf("monorepo-embedded no lleva puntero (err=%v)", err)
	}

	// multirepo sin repositorio indicado: disposición de siempre.
	legacy := t.TempDir()
	res, err = NewInitializer(nil, nil).Init(InitOptions{Path: legacy, Name: "Leg", Topology: "multirepo"})
	if err != nil {
		t.Fatal(err)
	}
	if res.ConfigPath != filepath.Join(legacy, "axiom.yaml") {
		t.Errorf("multirepo sin SpecsRepository debe escribir en la carpeta maestra: %s", res.ConfigPath)
	}
}

func TestInitMultirepoRejectsBadSpecsRepositoryWithoutWriting(t *testing.T) {
	cases := []struct {
		name, specs, topology, want string
	}{
		{"propia carpeta maestra", ".", "multirepo", "no puede ser la propia carpeta maestra"},
		{"ruta absoluta", absoluteSample(), "multirepo", "es una ruta absoluta"},
		{"fuera de la maestra", "../specs", "multirepo", "sale de la carpeta del proyecto"},
		{"en monorepo-embedded", "repo-specs", "monorepo-embedded", "solo se admite en la topología multirepo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			master := filepath.Join(t.TempDir(), "maestra")
			opts := InitOptions{Path: master, Topology: tc.topology, SpecsRepository: tc.specs}
			requireViolation(t, NewInitializer(nil, nil).ValidateInit(opts), tc.want)
			_, err := NewInitializer(nil, nil).Init(opts)
			requireViolation(t, err, tc.want)
			requireNothingWritten(t, master)
		})
	}
}

func TestInitMultirepoRejectsAddingSpecsRepoToAnExistingDirectConfig(t *testing.T) {
	master := t.TempDir()
	if _, err := NewInitializer(nil, nil).Init(InitOptions{Path: master, Name: "Previo"}); err != nil {
		t.Fatal(err)
	}
	opts := InitOptions{Path: master, Topology: "multirepo", SpecsRepository: "repo-specs"}
	requireViolation(t, NewInitializer(nil, nil).ValidateInit(opts), "solo se puede indicar en un alta nueva")
	if _, err := os.Stat(filepath.Join(master, "repo-specs")); !os.IsNotExist(err) {
		t.Errorf("el rechazo no debe crear el repositorio de specs (err=%v)", err)
	}
}

// La protección de planning home ajeno sigue en pie: un puntero hacia otro
// repositorio de specs que el pedido se rechaza.
func TestInitMultirepoStillRejectsAForeignPlanningHome(t *testing.T) {
	master := t.TempDir()
	if _, err := NewInitializer(nil, nil).Init(InitOptions{Path: master, Topology: "multirepo", SpecsRepository: "repo-specs"}); err != nil {
		t.Fatal(err)
	}
	_, err := NewInitializer(nil, nil).Init(InitOptions{Path: master, Topology: "multirepo", SpecsRepository: "otro-specs"})
	if _, ok := err.(*ForeignPlanningHomeError); !ok {
		t.Fatalf("se esperaba *ForeignPlanningHomeError, se obtuvo %v", err)
	}
	// Sin SpecsRepository, el planning home esperado es <ruta>/openspec y el del puntero es ajeno.
	_, err = NewInitializer(nil, nil).Init(InitOptions{Path: master})
	if _, ok := err.(*ForeignPlanningHomeError); !ok {
		t.Fatalf("se esperaba *ForeignPlanningHomeError sin SpecsRepository, se obtuvo %v", err)
	}
}
