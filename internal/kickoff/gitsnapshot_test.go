package kickoff

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// Pruebas de la instantánea de política git de `kickoff.yaml` (RF-KOF-06, RF-KOF-07, design D-12
// y §4.5): ida y vuelta YAML, ausencia tolerada, forma validada con gitref, sin SHAs ni rutas
// absolutas, lectura no estricta (un binario anterior la ignora) y escritura única con el
// sellado.

// sampleSnapshot es la instantánea de un multirepo (único lugar con ramas de rol: en un monorepo
// el único rol es `fullstack`, RF-TOP-04 de `workspace-topology`) con los roles core y qa sobre el
// mismo repositorio de código, la integración `main` y el prefijo `feat`; el repositorio de specs
// se trabaja in situ y no lleva ramas.
func sampleSnapshot() *GitSnapshot {
	return &GitSnapshot{
		IntegrationBranch: "main",
		BranchPrefix:      "feat",
		Exceptions:        &GitExceptions{Branch: "inc/x", Base: "develop"},
		ChangeBranch:      "inc/x",
		Repositories: []GitSnapshotRepo{
			{
				Path:              "repo-backend",
				Name:              "repo-backend",
				Role:              "code",
				IntegrationBranch: "main",
				Base:              "develop",
				Branches: []GitSnapshotBranch{
					{Name: "inc/x", Kind: "change"},
					{Name: "inc/x-core", Kind: "role", Role: "core"},
					{Name: "inc/x-qa", Kind: "role", Role: "qa"},
				},
			},
			{Path: "repo-specs", Name: "repo-specs", Role: "specs_in_place", IntegrationBranch: "main", Base: "main"},
		},
	}
}

func snapshotKickoff(change string) Kickoff {
	k := sampleKickoff(change)
	k.Git = sampleSnapshot()
	return k
}

// RF-KOF-06: la instantánea sobrevive a escribir y leer `kickoff.yaml`, campo a campo.
func TestGitSnapshotRoundTripsThroughKickoffYAML(t *testing.T) {
	changeRoot := t.TempDir()

	sealed, wrote, err := Seal(changeRoot, snapshotKickoff("x"))
	if err != nil || !wrote {
		t.Fatalf("Seal() = (%v, %v)", wrote, err)
	}
	if !reflect.DeepEqual(sealed.Git, sampleSnapshot()) {
		t.Errorf("instantánea sellada = %+v, se esperaba %+v", sealed.Git, sampleSnapshot())
	}
	loaded, err := Load(changeRoot)
	if err != nil || loaded == nil {
		t.Fatalf("Load() = (%v, %v)", loaded, err)
	}
	if !reflect.DeepEqual(loaded.Git, sampleSnapshot()) {
		t.Errorf("instantánea releída = %+v, se esperaba %+v", loaded.Git, sampleSnapshot())
	}

	data, err := os.ReadFile(filepath.Join(changeRoot, KickoffFileName))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"git:", "integration_branch: main", "branch_prefix: feat", "change_branch: inc/x", "kind: role", "role: core"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("kickoff.yaml debe contener %q:\n%s", want, data)
		}
	}
}

// RF-KOF-06: sin SHAs ni rutas absolutas ni de worktree en lo que se escribe.
func TestGitSnapshotWritesNeitherShasNorAbsolutePaths(t *testing.T) {
	changeRoot := t.TempDir()
	if _, _, err := Seal(changeRoot, snapshotKickoff("x")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(changeRoot, KickoffFileName))
	if err != nil {
		t.Fatal(err)
	}
	if regexp.MustCompile(`\b[0-9a-f]{40}\b|\b[0-9a-f]{64}\b`).Match(data) {
		t.Errorf("kickoff.yaml no debe contener SHAs:\n%s", data)
	}
	if strings.Contains(string(data), changeRoot) || strings.Contains(string(data), filepath.ToSlash(changeRoot)) {
		t.Errorf("kickoff.yaml no debe contener rutas absolutas:\n%s", data)
	}
}

// RF-KOF-06 y RF-KOF-07: la lectura tolera la ausencia de instantánea y el documento no emite la
// clave `git:` si no la hay, de modo que un kickoff antiguo no cambia ni un byte.
func TestKickoffWithoutSnapshotReadsAndWritesNoGitBlock(t *testing.T) {
	changeRoot := t.TempDir()
	if _, _, err := Seal(changeRoot, sampleKickoff("x")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(changeRoot, KickoffFileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "git:") {
		t.Errorf("sin instantánea no se emite `git:`:\n%s", data)
	}
	loaded, err := Load(changeRoot)
	if err != nil || loaded == nil || loaded.Git != nil {
		t.Fatalf("Load() = (%+v, %v), se esperaba un kickoff sin instantánea", loaded, err)
	}
}

// RF-KOF-06: un binario anterior —que decodifica sin estricto y no conoce `git:`— ignora el bloque
// sin error, y ParseKickoff tolera claves desconocidas dentro de él (versiones futuras).
func TestGitSnapshotIsIgnoredByAnOlderDecoderAndToleratesUnknownKeys(t *testing.T) {
	data, err := yaml.Marshal(snapshotKickoff("x"))
	if err != nil {
		t.Fatal(err)
	}
	var older struct {
		Schema string `yaml:"schema"`
		Change string `yaml:"change"`
	}
	if err := yaml.Unmarshal(data, &older); err != nil || older.Change != "x" {
		t.Fatalf("un decodificador anterior debe ignorar git: (%+v, %v)", older, err)
	}

	future := strings.Replace(string(data), "change_branch: inc/x", "change_branch: inc/x\n    clave_futura: 1", 1)
	parsed, err := ParseKickoff([]byte(future))
	if err != nil || parsed.Git == nil || parsed.Git.ChangeBranch != "inc/x" {
		t.Fatalf("ParseKickoff con una clave desconocida = (%+v, %v)", parsed.Git, err)
	}
}

// RF-KOF-06 y D-12: Validate comprueba la forma de la instantánea con gitref si el bloque
// existe, y nunca la coacciona.
func TestGitSnapshotValidateChecksTheShape(t *testing.T) {
	mutations := map[string]func(g *GitSnapshot){
		"integración inválida":                  func(g *GitSnapshot) { g.IntegrationBranch = "ma in" },
		"integración vacía":                     func(g *GitSnapshot) { g.IntegrationBranch = "" },
		"prefijo inválido":                      func(g *GitSnapshot) { g.BranchPrefix = "mi prefijo//" },
		"rama del cambio inválida":              func(g *GitSnapshot) { g.ChangeBranch = "a..b" },
		"excepción de prefijo inválida":         func(g *GitSnapshot) { g.Exceptions = &GitExceptions{Prefix: "a b"} },
		"excepción de rama inválida":            func(g *GitSnapshot) { g.Exceptions = &GitExceptions{Branch: "-x"} },
		"repositorio sin ruta":                  func(g *GitSnapshot) { g.Repositories[0].Path = "" },
		"repositorio sin nombre":                func(g *GitSnapshot) { g.Repositories[0].Name = "" },
		"papel desconocido":                     func(g *GitSnapshot) { g.Repositories[0].Role = "otro" },
		"integración del repositorio":           func(g *GitSnapshot) { g.Repositories[0].IntegrationBranch = "x:y" },
		"rama principal inválida":               func(g *GitSnapshot) { g.MainBranch = "ma in" },
		"promoción desconocida":                 func(g *GitSnapshot) { g.Repositories[0].Promotion = "otra" },
		"promoción ask no es un modo congelado": func(g *GitSnapshot) { g.Repositories[0].Promotion = "ask" },
		"base vacía":                            func(g *GitSnapshot) { g.Repositories[0].Base = "" },
		"tipo de rama desconocido":              func(g *GitSnapshot) { g.Repositories[0].Branches[0].Kind = "otra" },
		"nombre de rama inválido":               func(g *GitSnapshot) { g.Repositories[0].Branches[1].Name = "feat/x y" },
		"rol en una rama que no es de rol":      func(g *GitSnapshot) { g.Repositories[0].Branches[0].Role = "core" },
		"rama de rol sin rol":                   func(g *GitSnapshot) { g.Repositories[0].Branches[1].Role = "" },
		"ruta absoluta de POSIX":                func(g *GitSnapshot) { g.Repositories[0].Path = "/home/yo/axiom" },
		"ruta absoluta de Windows":              func(g *GitSnapshot) { g.Repositories[0].Path = `C:\repos\axiom` },
		"ruta absoluta de Windows con «/»":      func(g *GitSnapshot) { g.Repositories[0].Path = "C:/repos/axiom" },
		"ruta de worktree como base":            func(g *GitSnapshot) { g.Repositories[0].Base = "/tmp/wt/x" },
		"SHA como base del repositorio":         func(g *GitSnapshot) { g.Repositories[0].Base = strings.Repeat("a1", 20) },
		"SHA como base de la excepción":         func(g *GitSnapshot) { g.Exceptions = &GitExceptions{Base: strings.Repeat("0f", 20)} },
		"SHA-256 como base de la excepción":     func(g *GitSnapshot) { g.Exceptions = &GitExceptions{Base: strings.Repeat("ab", 32)} },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			k := snapshotKickoff("x")
			mutate(k.Git)
			if err := k.Validate(); err == nil {
				t.Fatalf("Validate() = nil, se esperaba rechazar la instantánea: %+v", k.Git)
			}
			if _, _, err := Seal(t.TempDir(), k); err == nil {
				t.Fatal("Seal() debe rechazar una instantánea inválida sin escribir nada")
			}
		})
	}

	if err := snapshotKickoff("x").Validate(); err != nil {
		t.Errorf("la instantánea de ejemplo es válida: %v", err)
	}
	// Una instantánea de un multirepo sin código puede no tener rama del cambio.
	k := snapshotKickoff("x")
	k.Git.ChangeBranch = ""
	k.Git.Repositories = k.Git.Repositories[1:]
	if err := k.Validate(); err != nil {
		t.Errorf("sin rama del cambio (solo el repositorio de specs) es válida: %v", err)
	}
}

// RF-KOF-06 y D-12: un `kickoff.yaml` con la instantánea mal formada no se acepta al leer.
func TestParseKickoffRejectsAMalformedSnapshot(t *testing.T) {
	k := snapshotKickoff("x")
	k.Git.Repositories[0].Role = "otro"
	data, err := yaml.Marshal(k)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseKickoff(data); err == nil {
		t.Fatal("ParseKickoff() debe rechazar una instantánea con un papel desconocido")
	}
}

// RF-KOF-06: la instantánea se escribe una sola vez, con el resto de la configuración sellada
// y en la misma escritura: un segundo sellado devuelve al ganador sin reescribir los bytes, y
// una instantánea distinta del segundo no sustituye a la primera.
func TestSealWritesTheSnapshotOnceAndReturnsTheWinnerWithoutRewriting(t *testing.T) {
	changeRoot := t.TempDir()
	first, wrote, err := Seal(changeRoot, snapshotKickoff("x"))
	if err != nil || !wrote {
		t.Fatalf("primer Seal() = (%v, %v)", wrote, err)
	}
	path := filepath.Join(changeRoot, KickoffFileName)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	second := snapshotKickoff("x")
	second.Git.ChangeBranch = "otra/x"
	second.Git.Repositories[0].Branches[0].Name = "otra/x"
	winner, wrote, err := Seal(changeRoot, second)
	if err != nil || wrote {
		t.Fatalf("segundo Seal() = (%v, %v), se esperaba devolver al ganador sin escribir", wrote, err)
	}
	if !reflect.DeepEqual(winner.Git, first.Git) {
		t.Errorf("el ganador conserva la primera instantánea: %+v", winner.Git)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("el segundo sellado reescribió kickoff.yaml:\n%s\n---\n%s", before, after)
	}
}

// RF-KOF-06: los ficheros se escriben con `\n`, también en Windows.
func TestSealedSnapshotUsesUnixLineEndings(t *testing.T) {
	changeRoot := t.TempDir()
	if _, _, err := Seal(changeRoot, snapshotKickoff("x")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(changeRoot, KickoffFileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "\r") {
		t.Errorf("kickoff.yaml debe escribirse con \\n:\n%q", data)
	}
}

// RF-KOF-06 y RF-BRA-28: la rama principal sellada y el modo de promoción ya materializado
// sobreviven a escribir y leer `kickoff.yaml`; ambos campos son válidos con `branch` y `release`.
func TestGitSnapshotRoundTripsTheMainBranchAndThePromotion(t *testing.T) {
	for _, mode := range []string{"branch", "release"} {
		t.Run(mode, func(t *testing.T) {
			changeRoot := t.TempDir()
			k := snapshotKickoff("x")
			k.Git.MainBranch = "master"
			k.Git.Repositories[0].Promotion = mode

			sealed, wrote, err := Seal(changeRoot, k)
			if err != nil || !wrote {
				t.Fatalf("Seal() = (%v, %v)", wrote, err)
			}
			if sealed.Git.MainBranch != "master" || sealed.Git.Repositories[0].Promotion != mode {
				t.Errorf("instantánea sellada = %+v", sealed.Git)
			}
			data, err := os.ReadFile(filepath.Join(changeRoot, KickoffFileName))
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"main_branch: master", "promotion: " + mode} {
				if !strings.Contains(string(data), want) {
					t.Errorf("kickoff.yaml debe contener %q:\n%s", want, data)
				}
			}
			loaded, err := Load(changeRoot)
			if err != nil || loaded == nil || loaded.Git.MainBranch != "master" || loaded.Git.Repositories[0].Promotion != mode {
				t.Fatalf("Load() = (%+v, %v)", loaded, err)
			}
		})
	}
}

// RF-KOF-06: sin los campos nuevos no se emite ninguna clave y un kickoff anterior se lee sin ellos.
func TestGitSnapshotWithoutPromotionFieldsEmitsNoNewKeysAndReadsOldDocuments(t *testing.T) {
	changeRoot := t.TempDir()
	if _, _, err := Seal(changeRoot, snapshotKickoff("x")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(changeRoot, KickoffFileName))
	if err != nil {
		t.Fatal(err)
	}
	for _, unwanted := range []string{"main_branch", "promotion"} {
		if strings.Contains(string(data), unwanted) {
			t.Errorf("sin los campos no se emite %q (los bytes de un kickoff anterior no cambian):\n%s", unwanted, data)
		}
	}
	loaded, err := Load(changeRoot)
	if err != nil || loaded == nil || loaded.Git.MainBranch != "" || loaded.Git.Repositories[0].Promotion != "" {
		t.Fatalf("Load() = (%+v, %v), se esperaba una instantánea sin los campos", loaded, err)
	}
}
