package specbaseline

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

const integrationConfig = `workspace:
  name: ejemplo
  topology: multirepo
  specs_repository: repo-specs
roles:
  backend:
    name: Backend
    repositories:
      - path: repo-backend
baseline:
  documents:
    - domain: requisitos-funcionales
      path: 01_Requisitos_Funcionales.md
      requirement:
        heading: '^#+ (?P<id>RF-\d+): (?P<title>.+)$'
        level: 3
        render: '### {id}: {title}'
    - domain: modelo-de-datos
      path: 03_Modelo_Datos.md
    - domain: flujos-de-negocio
      path: 04_Flujos_Negocio.md
      integration: register
technical_context:
  maintained:
    paths:
      - context/
      - ../repo-backend/docs/
  reference:
    paths:
      - referencia/
`

const (
	fixtureRequisitos = "# Requisitos funcionales\n\n## Catálogo\n\n### RF-001: Alta de producto\n\nEl sistema DEBE permitir dar de alta un producto.\n\n### RF-007: Consulta de producto\n\nEl sistema DEBE devolver los datos de un producto.\n"
	fixtureModelo     = "# Modelo de datos\n\n## Entidades\n\n### Cliente\n\nTiene nombre y correo.\n"
	fixtureFlujos     = "# Flujos de negocio\n\n## Pedidos\n\nEl pedido pasa por tres estados.\n"
	fixtureContext    = "# Arquitectura\n\n## Componentes\n\n### Servicio de catálogo\n\nGestiona los productos.\n\n### Servicio de pedidos\n\nGestiona los pedidos.\n"
	fixtureAPI        = "# API\n\n## Pagos\n\nTexto de pagos.\n"

	fixtureDeltaRequisitos = `## ADDED Requirements

### Requirement: RF-9001: Reintento de pago
(Placement: after RF-007)

El sistema DEBE reintentar el cobro fallido hasta tres veces.

#### Scenario: Primer reintento
- **CUANDO** falla el cobro
- **ENTONCES** se reintenta

## MODIFIED Requirements

### Requirement: RF-001: Alta de producto con imagen

El sistema DEBE permitir adjuntar una imagen. Véase RF-9001.
`
	fixtureDeltaModelo = "## ADDED Requirements\n\n### Requirement: Entidad Reintento\n\nCada reintento se asocia a un pedido (ver RF-9001).\n"
	fixtureDeltaFlujos = "## ADDED Requirements\n\n### Requirement: Flujo de devolución\n\nDescribe la devolución.\n"

	fixtureContextImpact = `# Delta de contexto: inc-99-ejemplo

## Entrada: context/ARQUITECTURA.md

- Ancla: Componentes > Servicio de pedidos
- Acción: modify
- Motivo: se añade el reintento de pago (RF-9001)

Texto nuevo para la sección (RF-9001).

## Entrada: ../repo-backend/docs/API.md

- Ancla: Errores
- Acción: add
- Motivo: documenta el error del cobro reintentado
`
)

// integrationFixture es un espacio canónico sintético: la carpeta maestra con el
// puntero, el repositorio de especificaciones con la baseline del proyecto y el
// contexto técnico, un repositorio de código hermano con documentación mantenida y
// el cambio inc-99-ejemplo con sus deltas.
type integrationFixture struct {
	master, specs, backend, changeRoot string
}

// integrationTemplate es el espacio canónico con git y todo confirmado: se monta
// una sola vez y se copia a la carpeta de cada prueba que lo necesita.
var integrationTemplate = &treeTemplate{name: "espacio-integracion"}

// newIntegrationFixtureAt describe el espacio canónico cuya carpeta maestra es
// master, sin crear nada en disco.
func newIntegrationFixtureAt(master string) *integrationFixture {
	return &integrationFixture{
		master:     master,
		specs:      filepath.Join(master, "repo-specs"),
		backend:    filepath.Join(master, "repo-backend"),
		changeRoot: filepath.Join(master, "repo-specs", "openspec", "changes", "inc-99-ejemplo"),
	}
}

func newIntegrationFixture(t *testing.T, withGit bool) *integrationFixture {
	t.Helper()
	if withGit {
		isolateGit(t)
	}
	master, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := newIntegrationFixtureAt(master)
	if withGit {
		integrationTemplate.materialize(t, master, func(t testing.TB, dir string) {
			buildIntegrationFixture(t, newIntegrationFixtureAt(dir), true)
		})
		refreshIndexes(t, f.specs, f.backend)
		return f
	}
	buildIntegrationFixture(t, f, false)
	return f
}

// buildIntegrationFixture escribe el espacio canónico en f.master y, con git, lo
// confirma en los dos repositorios.
func buildIntegrationFixture(t testing.TB, f *integrationFixture, withGit bool) {
	t.Helper()
	master := f.master
	if withGit {
		newRepo(t, f.specs)
		newRepo(t, f.backend)
	}
	writeRepoFile(t, master, ".axiom-workspace", "config: repo-specs/axiom.yaml\n")
	writeRepoFile(t, f.specs, "axiom.yaml", integrationConfig)
	writeRepoFile(t, f.specs, "01_Requisitos_Funcionales.md", fixtureRequisitos)
	writeRepoFile(t, f.specs, "03_Modelo_Datos.md", fixtureModelo)
	writeRepoFile(t, f.specs, "04_Flujos_Negocio.md", fixtureFlujos)
	writeRepoFile(t, f.specs, "context/ARQUITECTURA.md", fixtureContext)
	writeRepoFile(t, f.specs, "referencia/legado/VIEJO.md", "# Viejo\n")
	writeRepoFile(t, f.backend, "docs/API.md", fixtureAPI)
	f.writeChange(t, "specs/requisitos-funcionales/spec.md", fixtureDeltaRequisitos)
	f.writeChange(t, "specs/modelo-de-datos/spec.md", fixtureDeltaModelo)
	f.writeChange(t, "specs/flujos-de-negocio/spec.md", fixtureDeltaFlujos)
	f.writeChange(t, ContextDeltaFileName, fixtureContextImpact)
	if withGit {
		commitAll(t, f.specs, "baseline")
		commitAll(t, f.backend, "backend")
	}
}

func (f *integrationFixture) writeChange(t testing.TB, rel, content string) {
	t.Helper()
	writeRepoFile(t, f.changeRoot, rel, content)
}

func (f *integrationFixture) removeChange(t *testing.T, rel string) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(f.changeRoot, filepath.FromSlash(rel))); err != nil {
		t.Fatal(err)
	}
}

// target resuelve de nuevo la ubicación, de modo que un axiom.yaml modificado por
// la prueba se vea en la declaración.
func (f *integrationFixture) target(t *testing.T) ChangeTarget {
	t.Helper()
	loc, err := workspace.Locate(f.master)
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	return ChangeTarget{
		Location:      loc,
		Change:        "inc-99-ejemplo",
		ChangeRoot:    f.changeRoot,
		AllowedRoots:  []string{loc.WorkspaceRoot, f.backend},
		ArtifactStore: "openspec",
	}
}

// snapshot devuelve la huella de cada fichero bajo root, sin los directorios .git.
func snapshot(t *testing.T, root string) map[string]string {
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
		relative, _ := filepath.Rel(root, path)
		files[filepath.ToSlash(relative)] = sha256Hex(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func changedFiles(before, after map[string]string) []string {
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

func mustErrors(t *testing.T, report Report, fragments ...string) {
	t.Helper()
	joined := strings.Join(report.Errors, "\n")
	if len(report.Errors) == 0 {
		t.Fatalf("el informe debía tener errores con %q", fragments)
	}
	for _, fragment := range fragments {
		if !strings.Contains(joined, fragment) {
			t.Errorf("los errores deben contener %q:\n%s", fragment, joined)
		}
	}
}

func mustNoErrors(t *testing.T, report Report) {
	t.Helper()
	if len(report.Errors) != 0 {
		t.Fatalf("errores inesperados:\n%s", strings.Join(report.Errors, "\n"))
	}
}

func TestCheckReportsDocumentsOperationsIDsFamiliesAndContext(t *testing.T) {
	f := newIntegrationFixture(t, false)
	report, err := Check(context.Background(), f.target(t))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	mustNoErrors(t, report)
	if report.Schema != CheckSchema || report.Change != "inc-99-ejemplo" || report.Baseline != BaselineKindProject {
		t.Fatalf("informe = %+v", report)
	}
	if report.ComposeCommand != "" || len(report.Directed) != 0 || len(report.ContextItems) != 0 {
		t.Errorf("check no emite el comando ni los trabajos de edición: %+v", report)
	}

	domains := make([]string, len(report.Documents))
	for i, document := range report.Documents {
		domains[i] = document.Domain
	}
	if !reflect.DeepEqual(domains, []string{"requisitos-funcionales", "modelo-de-datos", "flujos-de-negocio"}) {
		t.Fatalf("dominios = %v, quería el orden de baseline.documents", domains)
	}
	requisitos := report.Documents[0]
	if requisitos.Integration != workspace.IntegrationCompose || requisitos.Path != "01_Requisitos_Funcionales.md" {
		t.Errorf("documento = %+v", requisitos)
	}
	wantOperations := []OperationReport{{Section: "ADDED", IDs: []string{"RF-9001"}}, {Section: "MODIFIED", IDs: []string{"RF-001"}}}
	if !reflect.DeepEqual(requisitos.Operations, wantOperations) {
		t.Errorf("operaciones = %+v, quería %+v", requisitos.Operations, wantOperations)
	}
	if report.Documents[1].Integration != workspace.IntegrationDirected || report.Documents[2].Integration != workspace.IntegrationRegister {
		t.Errorf("modos = %v, %v", report.Documents[1].Integration, report.Documents[2].Integration)
	}
	if !reflect.DeepEqual(report.IDMap, []IDAssignment{{Provisional: "RF-9001", Definitive: "RF-008", Domain: "requisitos-funcionales"}}) {
		t.Errorf("idMap = %+v", report.IDMap)
	}
	wantFamily := FamilyReport{Family: "RF-", Max: "RF-007", FirstDefinitive: "RF-008", LastDefinitive: "RF-008", SuggestedStart: "RF-9001"}
	if len(requisitos.Families) != 1 || requisitos.Families[0] != wantFamily {
		t.Errorf("familias = %+v, quería %+v", requisitos.Families, wantFamily)
	}

	if !report.Context.Declared || !report.Context.Present || report.Context.Decision != ContextEntries || len(report.Context.Entries) != 2 {
		t.Fatalf("contexto = %+v", report.Context)
	}
	first, second := report.Context.Entries[0], report.Context.Entries[1]
	if first.ID != "C1" || first.Document != "context/ARQUITECTURA.md" || first.Action != "modify" || first.Anchor != "Componentes > Servicio de pedidos" || first.Create {
		t.Errorf("primera entrada = %+v", first)
	}
	if second.ID != "C2" || second.Document != "../repo-backend/docs/API.md" || second.Action != "add" || second.Create {
		t.Errorf("segunda entrada (ancla inexistente con add sobre un documento que existe) = %+v", second)
	}
}

func TestCheckWritesNothingAndNeedsNeitherGitNorAuthorization(t *testing.T) {
	f := newIntegrationFixture(t, false) // ni git ni autorizaciones
	target := f.target(t)
	target.AllowedRoots = nil
	before := snapshot(t, f.master)

	report, err := Check(context.Background(), target)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	mustNoErrors(t, report)
	if changed := changedFiles(before, snapshot(t, f.master)); len(changed) != 0 {
		t.Fatalf("check no escribe nada, tampoco en la carpeta del cambio: %v", changed)
	}
	if _, err := os.Stat(filepath.Join(f.changeRoot, PlanFileName)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("check no crea %s: %v", PlanFileName, err)
	}
}

func TestCheckReportsEveryConflictingBodyWithItsLine(t *testing.T) {
	f := newIntegrationFixture(t, false)
	f.writeChange(t, "specs/requisitos-funcionales/spec.md", `## ADDED Requirements

### Requirement: RF-9001: Reintento de pago
(Placement: after RF-007)

Texto.

### Subtítulo que cerraría el bloque

Más texto.

## MODIFIED Requirements

### Requirement: RF-001: Alta de producto

Texto.
`)
	report, err := Check(context.Background(), f.target(t))
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, `dominio "requisitos-funcionales"`, "sección ADDED", `requisito "RF-9001: Reintento de pago"`, `"### Subtítulo que cerraría el bloque"`, "línea 8", "nivel 3", "cerraría el bloque")
	requisitos := report.Documents[0]
	if len(requisitos.Conflicts) != 1 || requisitos.Conflicts[0].Requirement != "RF-9001: Reintento de pago" || requisitos.Conflicts[0].Line != 8 || requisitos.Conflicts[0].Heading != "### Subtítulo que cerraría el bloque" {
		t.Errorf("conflictos = %+v", requisitos.Conflicts)
	}
}

func TestCheckListsAllTheConflictsOfSeveralRequirements(t *testing.T) {
	f := newIntegrationFixture(t, false)
	f.writeChange(t, "specs/requisitos-funcionales/spec.md", `## MODIFIED Requirements

### Requirement: RF-001: Alta de producto

Texto.

## Sección que no es de requisitos

### Requirement: RF-007: Consulta de producto

Texto.

#### Escenario válido

# Título nivel uno
`)
	report, err := Check(context.Background(), f.target(t))
	if err != nil {
		t.Fatal(err)
	}
	// Una sección `##` que no es ADDED, MODIFIED, REMOVED ni RENAMED no abre sección:
	// forma parte del cuerpo del requisito anterior, como en el motor nativo.
	want := []BodyConflict{
		{Section: "MODIFIED", Requirement: "RF-001: Alta de producto", Heading: "## Sección que no es de requisitos", Line: 7, Level: 2},
		{Section: "MODIFIED", Requirement: "RF-007: Consulta de producto", Heading: "# Título nivel uno", Line: 15, Level: 1},
	}
	if got := report.Documents[0].Conflicts; !reflect.DeepEqual(got, want) {
		t.Fatalf("conflictos = %+v, quería %+v (todos, no solo el primero)", got, want)
	}
	mustErrors(t, report, "RF-001: Alta de producto", "línea 7", "RF-007: Consulta de producto", "línea 15")
}

func TestCheckRejectsAnUndeclaredDomainWithoutCreatingIt(t *testing.T) {
	f := newIntegrationFixture(t, false)
	f.writeChange(t, "specs/nuevo-dominio/spec.md", fixtureDeltaFlujos)
	report, err := Check(context.Background(), f.target(t))
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, `"nuevo-dominio"`, "baseline.documents", "declara el dominio", "retira la delta")
	for _, created := range []string{filepath.Join(f.specs, "openspec", "specs"), filepath.Join(f.specs, "openspec", "INDEX.md")} {
		if _, err := os.Stat(created); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s no debe crearse: %v", created, err)
		}
	}
}

func TestCheckRejectsAnInvalidOperationWithDomainSectionAndRequirement(t *testing.T) {
	f := newIntegrationFixture(t, false)
	f.writeChange(t, "specs/requisitos-funcionales/spec.md", "## MODIFIED Requirements\n\n### Requirement: RF-777: Inexistente\n\nTexto.\n")
	f.writeChange(t, "specs/modelo-de-datos/spec.md", "# Sin requisitos\n")
	report, err := Check(context.Background(), f.target(t))
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, `dominio "requisitos-funcionales"`, "sección MODIFIED", `"RF-777: Inexistente"`, `dominio "modelo-de-datos"`, "no declara requisitos")
}

func TestCheckFamilySuggestionOfTheProvisionalStart(t *testing.T) {
	heading := func(pattern string) Profile {
		return profileFor(t, pattern, 3, workspace.IdentityID, "### {id}: {title}", workspace.ScenariosInclude)
	}
	added := func(count int, prefix string, base int) string {
		var b strings.Builder
		b.WriteString("## ADDED Requirements\n\n")
		for i := 1; i <= count; i++ {
			fmt.Fprintf(&b, "### Requirement: %s%d: Nuevo %d\n(Placement: after %s)\n\nTexto.\n\n", prefix, base+i, i, "RF-001")
		}
		return b.String()
	}
	tests := []struct {
		name    string
		pattern string
		doc     string
		count   int
		base    int // base de los IDs provisionales del delta
		want    FamilyReport
	}{
		{
			name: "máximo corto", pattern: `^#+ (?P<id>RF-\d+): (?P<title>.+)$`, count: 1, base: 9000,
			doc:  "# D\n\n### RF-001: a\n\n### RF-007: b\n",
			want: FamilyReport{Family: "RF-", Max: "RF-007", FirstDefinitive: "RF-008", LastDefinitive: "RF-008", SuggestedStart: "RF-9001"},
		},
		{
			name: "el máximo con K supera los dígitos mínimos", pattern: `^#+ (?P<id>RF-\d+): (?P<title>.+)$`, count: 2, base: 9000,
			doc:  "# D\n\n### RF-001: a\n\n### RF-9500: b\n",
			want: FamilyReport{Family: "RF-", Max: "RF-9500", FirstDefinitive: "RF-9501", LastDefinitive: "RF-9502", SuggestedStart: "RF-90001"},
		},
		{
			name: "relleno de ceros del máximo", pattern: `^#+ (?P<id>RF-\d+): (?P<title>.+)$`, count: 1, base: 9000,
			doc:  "# D\n\n### RF-0099: a\n",
			want: FamilyReport{Family: "RF-", Max: "RF-0099", FirstDefinitive: "RF-0100", LastDefinitive: "RF-0100", SuggestedStart: "RF-9001"},
		},
		{
			name: "un patrón que fija el ancho prueba 9·10^(ancho-1)", pattern: `^#+ (?P<id>RF-\d{3}): (?P<title>.+)$`, count: 1, base: 900,
			doc:  "# D\n\n### RF-001: a\n\n### RF-007: b\n",
			want: FamilyReport{Family: "RF-", Max: "RF-007", FirstDefinitive: "RF-008", LastDefinitive: "RF-008", SuggestedStart: "RF-900"},
		},
		{
			name: "ningún candidato cumple el patrón", pattern: `^#+ (?P<id>RF-\d{3}): (?P<title>.+)$`, count: 10, base: 900,
			doc:  "# D\n\n### RF-001: a\n\n### RF-995: b\n",
			want: FamilyReport{Family: "RF-", Max: "RF-995", FirstDefinitive: "RF-996", LastDefinitive: "RF-1005"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			document := PlannedDocument{Domain: "requisitos", RelPath: "01.md", Profile: heading(tt.pattern), Original: []byte(tt.doc)}
			families := planFamilies(document, parseDelta([]byte(added(tt.count, "RF-", tt.base))))
			if len(families) != 1 {
				t.Fatalf("familias = %+v", families)
			}
			note := families[0].Note
			families[0].Note = ""
			if families[0] != tt.want {
				t.Fatalf("familia = %+v, quería %+v", families[0], tt.want)
			}
			if tt.want.SuggestedStart == "" && (!strings.Contains(note, "ningún candidato") || !strings.Contains(note, "heading")) {
				t.Errorf("sin candidato válido debe explicarse: %q", note)
			}
			if tt.want.SuggestedStart != "" && note != "" {
				t.Errorf("con inicio sugerido no hay nota: %q", note)
			}
		})
	}

	t.Run("identidad por texto y familias sin IDs no tienen sugerencia", func(t *testing.T) {
		text := PlannedDocument{Domain: "t", Profile: profileFor(t, `^#+ (?P<title>.+)$`, 3, workspace.IdentityText, "### {title}", workspace.ScenariosInclude), Original: []byte("### Uno\n")}
		if families := planFamilies(text, parseDelta([]byte(added(1, "RF-", 9000)))); len(families) != 0 {
			t.Errorf("la identidad `text` no tiene familias: %+v", families)
		}
		other := PlannedDocument{Domain: "r", Profile: heading(`^#+ (?P<id>(?:RF|NFR)-\d+): (?P<title>.+)$`), Original: []byte("### RF-001: a\n")}
		if families := planFamilies(other, parseDelta([]byte(added(1, "NFR-", 9000)))); len(families) != 0 {
			t.Errorf("una familia sin ningún ID existente no se sugiere (PlanIDs la rechaza): %+v", families)
		}
	})
}

func TestCheckWithoutContextDeltaWarnsWhilePrepareFails(t *testing.T) {
	t.Parallel()
	f := newIntegrationFixture(t, true)
	f.removeChange(t, ContextDeltaFileName)
	commitAll(t, f.specs, "sin delta de contexto")

	check, err := Check(context.Background(), f.target(t))
	if err != nil {
		t.Fatal(err)
	}
	mustNoErrors(t, check)
	if joined := strings.Join(check.Warnings, "\n"); !strings.Contains(joined, ContextDeltaFileName) || !strings.Contains(joined, "Sin impacto") {
		t.Errorf("la ausencia del delta es un aviso que pide registrar el impacto: %v", check.Warnings)
	}
	if !check.Context.Declared || check.Context.Present || check.Context.Decision != ContextNone {
		t.Errorf("contexto = %+v", check.Context)
	}

	plan, report, err := Prepare(context.Background(), f.target(t), PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, ContextDeltaFileName, "«Sin impacto: <motivo>»", "nunca se asume")
	if plan.Schema != "" {
		t.Errorf("sin delta de contexto no hay plan: %+v", plan)
	}
	if _, err := os.Stat(filepath.Join(f.changeRoot, PlanFileName)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("prepare no escribe el plan con errores: %v", err)
	}
}

func TestCheckValidatesTheContextDelta(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{name: "documento de referencia", content: "## Entrada: referencia/legado/VIEJO.md\n\n- Ancla: Viejo\n- Acción: modify\n- Motivo: m\n", want: []string{"C1", "referencia/legado/VIEJO.md", "ruta de referencia", "modo reference"}},
		{name: "documento de la baseline", content: "## Entrada: 03_Modelo_Datos.md\n\n- Ancla: Entidades\n- Acción: modify\n- Motivo: m\n", want: []string{"'baseline.documents'"}},
		{name: "fuera de las rutas mantenidas", content: "## Entrada: docs/otro.md\n\n- Ancla: Uno\n- Acción: add\n- Motivo: m\n", want: []string{"docs/otro.md", "no está bajo ninguna ruta mantenida", `"context/"`, `"../repo-backend/docs/"`}},
		{name: "modify sobre un ancla inexistente", content: "## Entrada: context/ARQUITECTURA.md\n\n- Ancla: Componentes > Inexistente\n- Acción: modify\n- Motivo: m\n", want: []string{"no resuelve a ninguna sección", "solo 'add'"}},
		{name: "documento inexistente con modify", content: "## Entrada: context/NUEVO.md\n\n- Ancla: Uno\n- Acción: modify\n- Motivo: m\n", want: []string{"no existe", "solo la acción 'add'"}},
		{name: "formato inválido", content: "## Entrada: context/ARQUITECTURA.md\n\n- Ancla: Uno\n- Acción: replace\n- Motivo: m\n", want: []string{ContextDeltaFileName, "no es válido", "replace"}},
		{name: "«Sin impacto» sin motivo", content: "Sin impacto:\n", want: []string{"requiere un motivo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newIntegrationFixture(t, false)
			f.writeChange(t, ContextDeltaFileName, tt.content)
			report, err := Check(context.Background(), f.target(t))
			if err != nil {
				t.Fatal(err)
			}
			mustErrors(t, report, tt.want...)
		})
	}

	t.Run("«Sin impacto» con su motivo es una decisión válida", func(t *testing.T) {
		f := newIntegrationFixture(t, false)
		f.writeChange(t, ContextDeltaFileName, "Sin impacto: el cambio solo corrige una errata\n")
		report, err := Check(context.Background(), f.target(t))
		if err != nil {
			t.Fatal(err)
		}
		mustNoErrors(t, report)
		if report.Context.Decision != ContextNoImpact || report.Context.Reason != "el cambio solo corrige una errata" || len(report.Context.Entries) != 0 {
			t.Errorf("contexto = %+v", report.Context)
		}
	})
}

func TestCheckAndPrepareReportTheArtifactStoreCause(t *testing.T) {
	t.Parallel()
	for _, store := range []string{"engram", "none"} {
		t.Run("baseline y contexto con "+store, func(t *testing.T) {
			f := newIntegrationFixture(t, true)
			target := f.target(t)
			target.ArtifactStore = store
			check, err := Check(context.Background(), target)
			if err != nil {
				t.Fatal(err)
			}
			mustErrors(t, check, "baseline", "technical_context.maintained", fmt.Sprintf("%q", store), "artifact_store: openspec", "hybrid", "retira la declaración")
			_, prepare, err := Prepare(context.Background(), target, PlanOptions{})
			if err != nil {
				t.Fatal(err)
			}
			mustErrors(t, prepare, fmt.Sprintf("%q", store), "artifact_store: openspec")
			if _, err := os.Stat(filepath.Join(f.changeRoot, PlanFileName)); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("no se escribe el plan: %v", err)
			}
		})
	}

	t.Run("solo contexto mantenido", func(t *testing.T) {
		f := newIntegrationFixture(t, false)
		writeRepoFile(t, f.specs, "axiom.yaml", "workspace:\n  name: e\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  backend:\n    name: Backend\n    repositories:\n      - path: repo-backend\ntechnical_context:\n  maintained:\n    paths:\n      - context/\n")
		target := f.target(t)
		target.ArtifactStore = "none"
		report, err := Check(context.Background(), target)
		if err != nil {
			t.Fatal(err)
		}
		mustErrors(t, report, "technical_context.maintained", `"none"`)
		if strings.Contains(strings.Join(report.Errors, ""), "la baseline del proyecto") {
			t.Errorf("no declara baseline: %v", report.Errors)
		}
	})

	for _, store := range []string{"openspec", "hybrid", ""} {
		t.Run("los almacenes de ficheros o sin resolver no se reportan: "+store, func(t *testing.T) {
			f := newIntegrationFixture(t, false)
			target := f.target(t)
			target.ArtifactStore = store
			report, err := Check(context.Background(), target)
			if err != nil {
				t.Fatal(err)
			}
			mustNoErrors(t, report)
		})
	}

	t.Run("sin baseline del proyecto ni contexto mantenido el almacén engram no se reporta", func(t *testing.T) {
		f := newIntegrationFixture(t, false)
		writeRepoFile(t, f.specs, "axiom.yaml", "workspace:\n  name: e\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  backend:\n    name: Backend\n    repositories:\n      - path: repo-backend\ntechnical_context:\n  reference:\n    paths:\n      - referencia/\n")
		target := f.target(t)
		target.ArtifactStore = "engram"
		report, err := Check(context.Background(), target)
		if err != nil {
			t.Fatal(err)
		}
		mustNoErrors(t, report)
		if report.Baseline != BaselineKindDefault || len(report.Documents) != 0 || report.Context.Declared {
			t.Errorf("informe = %+v", report)
		}
	})
}

func TestCheckRejectsAnInvalidOrUnreadableDeclaration(t *testing.T) {
	t.Run("baseline inválida", func(t *testing.T) {
		f := newIntegrationFixture(t, false)
		writeRepoFile(t, f.specs, "axiom.yaml", strings.Replace(integrationConfig, "path: 03_Modelo_Datos.md", "path: /absoluta/03_Modelo_Datos.md", 1))
		report, err := Check(context.Background(), f.target(t))
		if err != nil {
			t.Fatal(err)
		}
		mustErrors(t, report, "no es válida", "axiom workspace validate")
		if len(report.Documents) != 0 {
			t.Errorf("con la declaración inválida no se evalúa nada: %+v", report.Documents)
		}
	})
	t.Run("YAML ilegible", func(t *testing.T) {
		f := newIntegrationFixture(t, false)
		loc := workspace.Location{
			ConfigPath:    filepath.Join(f.specs, "axiom.yaml"),
			Master:        f.master,
			WorkspaceRoot: f.specs,
			Declarations:  workspace.Declarations{ParseErr: errors.New("yaml: line 1: did not find expected ',' or ']'")},
		}
		report, err := Check(context.Background(), ChangeTarget{Location: loc, Change: "inc-99-ejemplo", ChangeRoot: f.changeRoot, ArtifactStore: "openspec"})
		if err != nil {
			t.Fatal(err)
		}
		mustErrors(t, report, "no se puede leer como YAML", "axiom workspace validate")
	})
}

func TestPrepareWritesOnlyTheIntegrationPlan(t *testing.T) {
	// Sin t.Parallel: sustituye now, una variable de paquete.
	fixed := time.Date(2026, 10, 10, 9, 30, 15, 987, time.UTC)
	previous := now
	now = func() time.Time { return fixed }
	t.Cleanup(func() { now = previous })

	f := newIntegrationFixture(t, true)
	// Suciedad ajena: un fichero sin seguimiento en el repositorio de
	// especificaciones y uno en el índice del de código.
	writeRepoFile(t, f.specs, "notas/borrador.md", "borrador\n")
	writeRepoFile(t, f.backend, "ajeno.md", "ajeno\n")
	runGit(t, f.backend, "add", "ajeno.md")
	before := snapshot(t, f.master)

	plan, report, err := Prepare(context.Background(), f.target(t), PlanOptions{})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	mustNoErrors(t, report)

	changed := changedFiles(before, snapshot(t, f.master))
	wantChanged := []string{"repo-specs/openspec/changes/inc-99-ejemplo/" + PlanFileName}
	if !reflect.DeepEqual(changed, wantChanged) {
		t.Fatalf("prepare solo escribe integration.yaml: cambiaron %v", changed)
	}

	written, err := ReadPlan(filepath.Join(f.changeRoot, PlanFileName))
	if err != nil {
		t.Fatalf("ReadPlan: %v", err)
	}
	if !reflect.DeepEqual(written, plan) {
		t.Fatalf("el plan devuelto no es el escrito:\n got %#v\nwant %#v", plan, written)
	}
	if plan.Schema != PlanSchema || plan.Change != "inc-99-ejemplo" || !plan.PreparedAt.Equal(fixed.Truncate(time.Second)) {
		t.Errorf("plan = %+v", plan)
	}
	if plan.InputsSHA256 == "" || plan.Options.Supersede || plan.Options.SupersededRequirements == nil {
		t.Errorf("opciones = %+v", plan.Options)
	}

	if len(plan.Repositories) != 2 || plan.Repositories[0].Repo != "repo-backend" || plan.Repositories[1].Repo != "repo-specs" {
		t.Fatalf("repositorios = %+v", plan.Repositories)
	}
	for repo, dir := range map[string]string{"repo-backend": f.backend, "repo-specs": f.specs} {
		captured := plan.Repositories[0]
		if repo == "repo-specs" {
			captured = plan.Repositories[1]
		}
		if captured.Head != runGit(t, dir, "rev-parse", "HEAD") {
			t.Errorf("%s: head = %q", repo, captured.Head)
		}
	}
	if dirty := plan.Repositories[1].Dirty; len(dirty) != 1 || dirty[0].Path != "notas/borrador.md" || dirty[0].SHA256 != sha256Hex([]byte("borrador\n")) {
		t.Errorf("foto previa de repo-specs = %+v", dirty)
	}
	if dirty := plan.Repositories[0].Dirty; len(dirty) != 1 || dirty[0].Path != "ajeno.md" {
		t.Errorf("el cambio en el índice de un fichero ajeno no detiene y queda en la foto: %+v", dirty)
	}

	if !reflect.DeepEqual(plan.IDMap, []IDAssignment{{Provisional: "RF-9001", Definitive: "RF-008", Domain: "requisitos-funcionales"}}) {
		t.Errorf("id_map = %+v", plan.IDMap)
	}
	documents := map[string]PlanDocument{}
	for _, document := range plan.Documents {
		documents[document.Domain] = document
	}
	requisitos := documents["requisitos-funcionales"]
	if requisitos.Integration != workspace.IntegrationCompose || requisitos.BeforeSHA256 != sha256Hex([]byte(fixtureRequisitos)) || requisitos.AfterSHA256 == "" || requisitos.AfterSHA256 == requisitos.BeforeSHA256 || requisitos.State != StatePending {
		t.Errorf("documento compuesto = %+v", requisitos)
	}
	if documents["modelo-de-datos"].AfterSHA256 != "" || documents["modelo-de-datos"].BeforeSHA256 != sha256Hex([]byte(fixtureModelo)) || documents["flujos-de-negocio"].Integration != workspace.IntegrationRegister {
		t.Errorf("documentos de prosa = %+v", documents)
	}
	if plan.Context != (ContextDecision{Decision: ContextEntries}) {
		t.Errorf("contexto = %+v", plan.Context)
	}

	kinds := map[EntryKind][]TraceEntry{}
	for _, entry := range plan.Entries {
		kinds[entry.Kind] = append(kinds[entry.Kind], entry)
		if entry.State != StatePending {
			t.Errorf("toda entrada nace pendiente: %+v", entry)
		}
	}
	if len(kinds[EntryCompose]) != 2 || len(kinds[EntryDirected]) != 1 || len(kinds[EntryRegister]) != 1 || len(kinds[EntryContext]) != 2 {
		t.Fatalf("entradas = %+v", plan.Entries)
	}
	var added TraceEntry
	for _, entry := range kinds[EntryCompose] {
		if entry.Action == ActionAdded {
			added = entry
		}
	}
	if added.ID != "RF-008" || added.Provisional != "RF-9001" || added.Anchor != "### RF-008: Reintento de pago" {
		t.Errorf("entrada ADDED = %+v", added)
	}
	if directed := kinds[EntryDirected][0]; directed.ID != "Entidad Reintento" || directed.Anchor != "inc-99-ejemplo" || directed.Document != "03_Modelo_Datos.md" || directed.Action != ActionAdded {
		t.Errorf("entrada dirigida = %+v", directed)
	}
	if register := kinds[EntryRegister][0]; register.ID != "Flujo de devolución" || register.Document != "04_Flujos_Negocio.md" || register.Anchor != "inc-99-ejemplo" {
		t.Errorf("entrada de solo registro = %+v", register)
	}
	if first := kinds[EntryContext][0]; first.ID != "C1" || first.Reason != "se añade el reintento de pago (RF-008)" || first.Action != ActionModified {
		t.Errorf("la entrada de contexto lleva el motivo con el ID definitivo: %+v", first)
	}
}

func TestPrepareReportCarriesTheComposeCommandAndTheSubstitutedText(t *testing.T) {
	t.Parallel()
	f := newIntegrationFixture(t, true)
	target := f.target(t)
	_, report, err := Prepare(context.Background(), target, PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mustNoErrors(t, report)
	if report.Schema != PrepareSchema {
		t.Errorf("esquema = %q", report.Schema)
	}
	want := fmt.Sprintf(`axiom sdd archive-compose --cwd "%s" --change inc-99-ejemplo --document requisitos-funcionales --in-place`, target.Location.WorkspaceRoot)
	if report.ComposeCommand != want {
		t.Errorf("composeCommand = %q, quería %q", report.ComposeCommand, want)
	}
	if strings.Contains(report.ComposeCommand, "openspec/specs") {
		t.Errorf("el comando no compone rutas de OpenSpec: %q", report.ComposeCommand)
	}

	if len(report.Directed) != 1 {
		t.Fatalf("directed = %+v", report.Directed)
	}
	directed := report.Directed[0]
	if directed.Domain != "modelo-de-datos" || directed.Document != "03_Modelo_Datos.md" || directed.Section != "inc-99-ejemplo" || len(directed.Entries) != 1 {
		t.Errorf("directed = %+v", directed)
	}
	if entry := directed.Entries[0]; entry.Section != "ADDED" || entry.Name != "Entidad Reintento" || entry.Text != "Cada reintento se asocia a un pedido (ver RF-008)." {
		t.Errorf("el texto dirigido lleva el ID definitivo: %+v", entry)
	}

	if len(report.ContextItems) != 2 {
		t.Fatalf("contextItems = %+v", report.ContextItems)
	}
	first, second := report.ContextItems[0], report.ContextItems[1]
	if first.Reason != "se añade el reintento de pago (RF-008)" || first.Text != "Texto nuevo para la sección (RF-008)." || first.Create {
		t.Errorf("primer elemento de contexto = %+v", first)
	}
	if second.Create || second.Document != "../repo-backend/docs/API.md" || second.Action != "add" {
		t.Errorf("segundo elemento de contexto = %+v", second)
	}
	if len(report.Register) != 1 || report.Register[0].Kind != EntryRegister || report.Register[0].State != StatePending {
		t.Errorf("register = %+v", report.Register)
	}
	if !reflect.DeepEqual(report.IDMap, []IDAssignment{{Provisional: "RF-9001", Definitive: "RF-008", Domain: "requisitos-funcionales"}}) {
		t.Errorf("idMap = %+v", report.IDMap)
	}
	// El delta de contexto de la carpeta del cambio se archiva sin alteración (REQ-24.59).
	if data, _ := os.ReadFile(filepath.Join(f.changeRoot, ContextDeltaFileName)); string(data) != fixtureContextImpact {
		t.Error("prepare no altera el delta de contexto")
	}
}

func TestPrepareReemitsTheSamePlanAndRejectsChangedInputs(t *testing.T) {
	// Sin t.Parallel: sustituye now, una variable de paquete.
	first := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)
	previous := now
	t.Cleanup(func() { now = previous })
	now = func() time.Time { return first }

	f := newIntegrationFixture(t, true)
	plan, report, err := Prepare(context.Background(), f.target(t), PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mustNoErrors(t, report)
	planPath := filepath.Join(f.changeRoot, PlanFileName)
	written, _ := os.ReadFile(planPath)

	// Ahora el plan es el que existe: la misma preparación lo reemite sin tocarlo.
	now = func() time.Time { return first.Add(time.Hour) }
	again, report, err := Prepare(context.Background(), f.target(t), PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mustNoErrors(t, report)
	if !reflect.DeepEqual(again, plan) || !again.PreparedAt.Equal(first) {
		t.Errorf("la reemisión debe devolver el mismo plan, con su prepared_at: %+v", again)
	}
	if current, _ := os.ReadFile(planPath); string(current) != string(written) {
		t.Error("reemitir no reescribe el plan")
	}
	if report.ComposeCommand == "" || len(report.Directed) != 1 {
		t.Errorf("la reemisión también informa del comando y de los trabajos: %+v", report)
	}

	// Un delta cambiado desde prepare.
	f.writeChange(t, "specs/modelo-de-datos/spec.md", fixtureDeltaModelo+"\n### Requirement: Entidad Extra\n\nTexto.\n")
	_, report, err = Prepare(context.Background(), f.target(t), PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, "ya no corresponde a las entradas actuales", "git checkout -- 01_Requisitos_Funcionales.md", PlanFileName, "volver a preparar", "axiom sdd integrate prepare")
	if current, _ := os.ReadFile(planPath); string(current) != string(written) {
		t.Error("rechazar no modifica el plan")
	}

	// Un documento de la baseline cambió desde prepare.
	f.writeChange(t, "specs/modelo-de-datos/spec.md", fixtureDeltaModelo)
	writeRepoFile(t, f.specs, "01_Requisitos_Funcionales.md", fixtureRequisitos+"\n### RF-008: Colado a mano\n\nTexto.\n")
	_, report, err = Prepare(context.Background(), f.target(t), PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, "ya no corresponde", "git checkout -- 01_Requisitos_Funcionales.md")
}

func TestPrepareRejectsAChangeWhoseCompositionsAreAlreadyIntegrated(t *testing.T) {
	t.Parallel()
	f := newIntegrationFixture(t, true)
	plan, report, err := Prepare(context.Background(), f.target(t), PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mustNoErrors(t, report)
	for i := range plan.Documents {
		if plan.Documents[i].Integration == workspace.IntegrationCompose {
			plan.Documents[i].State = StateIntegrated
		}
	}
	if err := WritePlan(filepath.Join(f.changeRoot, PlanFileName), plan); err != nil {
		t.Fatal(err)
	}
	// Los documentos ya están compuestos, y por tanto sucios: «ya integrado» tiene
	// precedencia sobre las precondiciones de git.
	writeRepoFile(t, f.specs, "01_Requisitos_Funcionales.md", "# compuesto\n")

	_, report, err = Prepare(context.Background(), f.target(t), PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, "ya integrado", "RF-9001 -> RF-008", "axiom sdd integrate verify", "axiom sdd integrate show")
}

func TestPrepareStopsOnGitPreconditionsWithoutWritingAnything(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(t *testing.T, f *integrationFixture)
		want  []string
	}{
		{name: "documento de la baseline con cambios sin stage", setup: func(t *testing.T, f *integrationFixture) {
			writeRepoFile(t, f.specs, "01_Requisitos_Funcionales.md", fixtureRequisitos+"\nsucio\n")
		}, want: []string{"repo-specs", "01_Requisitos_Funcionales.md", "árbol de trabajo", "confirma", "git stash -u", "axiom sdd integrate prepare"}},
		{name: "documento de prosa en el índice", setup: func(t *testing.T, f *integrationFixture) {
			writeRepoFile(t, f.specs, "03_Modelo_Datos.md", fixtureModelo+"\nsucio\n")
			runGit(t, f.specs, "add", "03_Modelo_Datos.md")
		}, want: []string{"03_Modelo_Datos.md", "índice"}},
		{name: "documento de contexto sucio en el repositorio hermano", setup: func(t *testing.T, f *integrationFixture) {
			writeRepoFile(t, f.backend, "docs/API.md", fixtureAPI+"\nsucio\n")
		}, want: []string{"repo-backend", "docs/API.md"}},
		{name: "documento de contexto sin seguimiento", setup: func(t *testing.T, f *integrationFixture) {
			f.writeChange(t, ContextDeltaFileName, "## Entrada: context/NUEVO.md\n\n- Ancla: Uno\n- Acción: add\n- Motivo: m\n")
			writeRepoFile(t, f.specs, "context/NUEVO.md", "# Nuevo\n")
		}, want: []string{"context/NUEVO.md", "sin seguimiento"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newIntegrationFixture(t, true)
			tt.setup(t, f)
			before := snapshot(t, f.master)
			plan, report, err := Prepare(context.Background(), f.target(t), PlanOptions{})
			if err != nil {
				t.Fatal(err)
			}
			mustErrors(t, report, tt.want...)
			if plan.Schema != "" {
				t.Errorf("con la precondición fallida no hay plan: %+v", plan)
			}
			if changed := changedFiles(before, snapshot(t, f.master)); len(changed) != 0 {
				t.Errorf("la parada no modifica nada: %v", changed)
			}
		})
	}

	t.Run("un documento afectado en un directorio que no es git", func(t *testing.T) {
		t.Parallel()
		f := newIntegrationFixture(t, true)
		if err := os.RemoveAll(filepath.Join(f.backend, ".git")); err != nil {
			t.Fatal(err)
		}
		_, report, err := Prepare(context.Background(), f.target(t), PlanOptions{})
		if err != nil {
			t.Fatal(err)
		}
		mustErrors(t, report, "repo-backend/docs/API.md", "repositorio git", "git init")
	})

	t.Run("un documento por crear y la suciedad ajena no detienen", func(t *testing.T) {
		t.Parallel()
		f := newIntegrationFixture(t, true)
		f.writeChange(t, ContextDeltaFileName, "## Entrada: context/PAGOS.md\n\n- Ancla: Pagos\n- Acción: add\n- Motivo: documenta los pagos\n")
		writeRepoFile(t, f.specs, "04_Flujos_Negocio.md", fixtureFlujos+"\nsucio de solo registro\n") // un documento de solo registro no se edita
		writeRepoFile(t, f.specs, "notas/ajeno.md", "ajeno\n")
		_, report, err := Prepare(context.Background(), f.target(t), PlanOptions{})
		if err != nil {
			t.Fatal(err)
		}
		mustNoErrors(t, report)
		if len(report.ContextItems) != 1 || !report.ContextItems[0].Create {
			t.Errorf("contextItems = %+v", report.ContextItems)
		}
	})
}

func TestPrepareRequiresEditAuthorizationOutsideTheSpecsRepository(t *testing.T) {
	t.Parallel()
	f := newIntegrationFixture(t, true)
	target := f.target(t)
	target.AllowedRoots = []string{target.Location.WorkspaceRoot} // sin repo-backend
	before := snapshot(t, f.master)

	plan, report, err := Prepare(context.Background(), target, PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, "C2", "../repo-backend/docs/API.md", "autorización de edición", f.backend, "axiom sdd-attempt grant", "--change inc-99-ejemplo", "nunca se concede de forma implícita")
	if plan.Schema != "" {
		t.Error("sin autorización no hay plan")
	}
	if changed := changedFiles(before, snapshot(t, f.master)); len(changed) != 0 {
		t.Errorf("sin autorización no se escribe nada: %v", changed)
	}

	check, err := Check(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	mustNoErrors(t, check) // check no exige autorización

	target.AllowedRoots = append(target.AllowedRoots, f.backend)
	if _, report, err = Prepare(context.Background(), target, PlanOptions{}); err != nil || len(report.Errors) != 0 {
		t.Fatalf("con la autorización concedida prepare tiene éxito: %v %v", err, report.Errors)
	}
}

func TestPrepareRecordsTheNoImpactDecisionWithoutNeedingAuthorization(t *testing.T) {
	t.Parallel()
	f := newIntegrationFixture(t, true)
	f.writeChange(t, ContextDeltaFileName, "Sin impacto: el cambio solo corrige una errata en un mensaje de error\n")
	commitAll(t, f.specs, "sin impacto")
	target := f.target(t)
	target.AllowedRoots = []string{target.Location.WorkspaceRoot}

	plan, report, err := Prepare(context.Background(), target, PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mustNoErrors(t, report)
	if plan.Context != (ContextDecision{Decision: ContextNoImpact, Reason: "el cambio solo corrige una errata en un mensaje de error"}) {
		t.Errorf("contexto = %+v", plan.Context)
	}
	for _, entry := range plan.Entries {
		if entry.Kind == EntryContext {
			t.Errorf("«Sin impacto» no tiene entradas de contexto: %+v", entry)
		}
	}
}

func TestPrepareIsAllOrNothingWhenOneCompositionFails(t *testing.T) {
	t.Parallel()
	f := newIntegrationFixture(t, true)
	f.writeChange(t, "specs/requisitos-funcionales/spec.md", fixtureDeltaRequisitos+"\n### Requirement: RF-777: Inexistente\n\nTexto.\n")
	commitAll(t, f.specs, "delta con un MODIFIED sobre un ID inexistente")
	before := snapshot(t, f.master)

	plan, report, err := Prepare(context.Background(), f.target(t), PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, `dominio "requisitos-funcionales"`, "sección MODIFIED", `"RF-777: Inexistente"`)
	if plan.Schema != "" || len(plan.Entries) != 0 {
		t.Errorf("no hay plan parcial: %+v", plan)
	}
	if changed := changedFiles(before, snapshot(t, f.master)); len(changed) != 0 {
		t.Errorf("ningún documento ni el plan cambian: %v", changed)
	}
}

func TestPrepareAddsContextEntriesForANewMaintainedDocumentAndRecordsTheOptions(t *testing.T) {
	t.Parallel()
	f := newIntegrationFixture(t, true)
	f.writeChange(t, ContextDeltaFileName, "## Entrada: context/PAGOS.md\n\n- Ancla: Reintentos\n- Acción: add\n- Motivo: documenta los pagos (RF-9001)\n\nTexto (RF-9001).\n")
	f.writeChange(t, "specs/requisitos-funcionales/spec.md", "## REMOVED Requirements\n\n### Requirement: RF-007: Consulta de producto\n(Reason: sustituido por RF-9001)\n")
	commitAll(t, f.specs, "contexto nuevo y supersede")

	plan, report, err := Prepare(context.Background(), f.target(t), PlanOptions{Supersede: true, SupersededRequirements: []string{" RF-001 ", ""}})
	if err != nil {
		t.Fatal(err)
	}
	mustNoErrors(t, report)
	if !plan.Options.Supersede || !reflect.DeepEqual(plan.Options.SupersededRequirements, []string{"RF-001"}) {
		t.Errorf("opciones = %+v", plan.Options)
	}
	var contextEntry TraceEntry
	for _, entry := range plan.Entries {
		if entry.Kind == EntryContext {
			contextEntry = entry
		}
	}
	if contextEntry.Document != "context/PAGOS.md" || contextEntry.Action != ActionAdded || contextEntry.Reason != "documenta los pagos (RF-9001)" {
		t.Errorf("entrada de contexto = %+v (sin ADDED en el cambio, el provisional no tiene correspondencia y se conserva)", contextEntry)
	}
}
