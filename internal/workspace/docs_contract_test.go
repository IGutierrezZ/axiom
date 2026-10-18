package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readManualDeInicio lee el manual real de docs/ con una ruta relativa al
// paquete (mismo patrón que internal/absorptionledger/ledger_test.go): los pins
// de documentación leen el texto que se publica, no una copia.
func readManualDeInicio(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "manual-de-inicio.md"))
	if err != nil {
		t.Fatalf("no se pudo leer docs/manual-de-inicio.md: %v", err)
	}
	return string(raw)
}

func assertDocContains(t *testing.T, doc string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(doc, want) {
			t.Errorf("docs/manual-de-inicio.md no contiene %q", want)
		}
	}
}

// REQ-24.12: la receta del espacio canónico multirepo se documenta, con el
// puntero, la configuración a la que apunta y el montaje manual.
func TestManualDeInicioDocumentsTheCanonicalWorkspaceRecipe(t *testing.T) {
	assertDocContains(t, readManualDeInicio(t),
		"### Espacio Canónico Multirepo (Montaje Manual)",
		"se monta a mano",
		"carpeta maestra no versionada",
		"`.axiom-workspace`",
		"`<specs_repository>/axiom.yaml`",
		"config: repo-specs/axiom.yaml",
		"`<maestra>/repo-specs/openspec`",
		"`axiom workspace validate --path <maestra>`",
		"`axiom sdd-attempt grant`",
	)
}

// REQ-24.12 y REQ-21.16: la evidencia pr_merged se verifica contra el
// repositorio de especificaciones y la de los repositorios de código es otra.
func TestManualDeInicioDocumentsPRMergedEvidence(t *testing.T) {
	assertDocContains(t, readManualDeInicio(t),
		"`axiom sdd gate record --evidence-kind pr_merged`",
		"se verifica contra el grafo git del repositorio de especificaciones (`workspaceRoot`)",
		"con las evidencias `deployment` o `attestation`",
	)
}

// REQ-24.21: la frontera entre axiom.yaml y openspec/config.yaml se documenta.
func TestOpenSpecConfigDocumentsTheConfigurationBoundary(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "openspec-config.md"))
	if err != nil {
		t.Fatalf("no se pudo leer docs/openspec-config.md: %v", err)
	}
	doc := string(raw)
	for _, want := range []string{
		"## Configuration boundary with `axiom.yaml`",
		"`baseline` and `technical_context` live only in `axiom.yaml`",
		"A key with that name in `openspec/config.yaml` has no effect",
		"The behavioral rules (`rules.<phase>`, `strict_tdd`, `artifact_store`) stay in `openspec/config.yaml`",
		"`governance.context` is unchanged",
		"it must be `openspec` or `hybrid`",
		"`axiom workspace validate --path <master folder>`",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("docs/openspec-config.md no contiene %q", want)
		}
	}
}

// m6 de design §13: el cambio de resolución desde un subdirectorio sin
// marcadores propios y el aviso de usar el binario nuevo en todo el equipo.
func TestManualDeInicioDocumentsTheResolutionChangeFromASubdirectory(t *testing.T) {
	assertDocContains(t, readManualDeInicio(t),
		"desde un subdirectorio sin marcadores propios resolvían `<subdirectorio>/openspec`",
		"Ahora resuelven el planning home del marcador más cercano",
		"todo el equipo debe usar el binario nuevo",
	)
}

// REQ-24.13, REQ-24.17 y REQ-24.62 (design §9): el manual documenta la
// declaración de la baseline del proyecto, su validación y la adopción en cuatro
// pasos, con el aviso de usar el binario nuevo en todo el equipo.
func TestManualDeInicioDocumentsTheProjectBaselineAndItsAdoption(t *testing.T) {
	assertDocContains(t, readManualDeInicio(t),
		"### Baseline del Proyecto y Contexto Técnico",
		"Se declara solo en `axiom.yaml` (nunca en `openspec/config.yaml`",
		"baseline:\n  documents:",
		"integration: directed",
		"integration: register",
		"technical_context:\n  maintained:",
		"`axiom workspace validate --path <maestra>` comprueba la declaración contra el sistema de ficheros y el almacén de artefactos",
		"bloquea solo el archivado",
		"exigen `artifact_store` `openspec` o `hybrid`",
		"ninguna operación de Axiom crea `openspec/specs/` ni `openspec/INDEX.md`",
		"`axiom sdd integrate check --cwd <maestra> --change <cambio>`",
		"#### Adopción en cuatro pasos",
		"1. Monta a mano la topología canónica",
		"2. Declara `baseline:` y `technical_context:` en `axiom.yaml`",
		"3. Ejecuta `axiom workspace validate --path <maestra>` hasta que sea conforme.",
		"4. Para los cambios con baseline con perfil de requisito, ejecuta `axiom sdd integrate check` desde `sdd-spec`.",
		"todo el equipo debe usar el binario nuevo antes de declarar la baseline",
	)
}

// D-14 (design §9): el manual indica, para los runtimes con el modelo fijado por
// agente, que se asigne a sdd-archive un modelo de nivel de implementación.
func TestManualDeInicioDocumentsTheArchiveModelForDirectedIntegration(t *testing.T) {
	assertDocContains(t, readManualDeInicio(t),
		"#### Modelo de `sdd-archive` con integración dirigida",
		"el orquestador lanza `sdd-archive` con el modelo asignado a `sdd-apply`",
		"En los runtimes con el modelo fijado por agente (OpenCode, Kiro y otros)",
		"asigna a `sdd-archive` un modelo de nivel de implementación con el selector de modelos existente",
		"Nunca bloquea el archivado",
	)
}

// workspace-topology REQ-25.25 (escenario «Ninguna documentación presenta el
// valor como válido»): el manual de inicio, la plantilla axiom.example.yaml y la
// visión de arquitectura no ofrecen `monorepo-decoupled` como valor de
// `workspace.topology`. Una mención solo se admite si la propia línea la declara
// retirada y remite a `multirepo`. docs/archive/ROADMAP.md queda fuera: es el
// registro histórico de INC-01, no documentación vigente. Las specs vivas las
// actualiza sdd-archive con los deltas del cambio.
func TestDocsDoNotOfferTheRetiredTopologyAsValid(t *testing.T) {
	for _, file := range []string{
		filepath.Join("docs", "manual-de-inicio.md"),
		"axiom.example.yaml",
		filepath.Join("docs", "architecture", "AXIOM_ENTERPRISE_VISION.md"),
	} {
		raw, err := os.ReadFile(filepath.Join("..", "..", file))
		if err != nil {
			t.Fatalf("no se pudo leer %s: %v", file, err)
		}
		for number, line := range strings.Split(string(raw), "\n") {
			if !strings.Contains(line, "monorepo-decoupled") {
				continue
			}
			if !strings.Contains(line, "retirad") || !strings.Contains(line, "multirepo") {
				t.Errorf("%s:%d presenta `monorepo-decoupled` sin declararlo retirado y remitir a `multirepo`: %s", file, number+1, strings.TrimSpace(line))
			}
		}
	}
}

// workspace-topology REQ-25.26 y REQ-25.27: el manual documenta los bloques
// `git:` y `repositories:` de axiom.yaml (esquema, valores por defecto,
// herencia, validación y lo que no se declara ahí) y la plantilla los muestra
// comentados. El verbo `axiom branch` que los consume llega después, y el manual
// lo dice.
func TestDocsDocumentTheGitBranchPolicyBlock(t *testing.T) {
	doc := readManualDeInicio(t)
	assertDocContains(t, doc,
		"[Política de Ramas (`git:`)](#política-de-ramas-git)",
		"### Política de Ramas (`git:`)",
		"git:\n  integration_branch: main",
		"  branch_prefix: feat",
		"rigen `integration_branch: main` y `branch_prefix: feat`",
		"nombres de referencia git válidos",
		"`branch_prefix` además no puede empezar ni terminar en `/`",
		"Una clave desconocida dentro de `git:` es un error",
		"El modo de trabajo y la raíz de los worktrees **no** se declaran en `axiom.yaml`",
		"`axiom workspace validate` rechaza un bloque `git:` inválido nombrando la clave, el valor y la regla incumplida",
		"Los binarios anteriores ignoran el bloque",
		"el verbo `axiom branch` que los aplicará llega en una entrega posterior",
		// El bloque `repositories:` (REQ-25.27) se documenta en la misma sección.
		"[Ajustes por Repositorio (`repositories:`)](#ajustes-por-repositorio-repositories)",
		"### Ajustes por Repositorio (`repositories:`)",
		"repositories:\n  - path: repo-backend",
		"    integration_branch: release",
		"  - path: repo-frontend\n    branch_prefix: ui",
		"`path` es obligatoria y relativa a la carpeta maestra",
		"no puede ser absoluta ni salir de la carpeta maestra",
		"Dos entradas con la misma ruta se rechazan",
		"Un repositorio sin entrada, o una entrada sin una de las claves, hereda el valor del bloque `git:`",
		"Una clave desconocida dentro de una entrada es un error",
	)

	example, err := os.ReadFile(filepath.Join("..", "..", "axiom.example.yaml"))
	if err != nil {
		t.Fatalf("no se pudo leer axiom.example.yaml: %v", err)
	}
	for _, want := range []string{
		"# git:", "#   integration_branch: main", "#   branch_prefix: feat",
		"# repositories:", "#   - path: backend-api", "#     integration_branch: release",
	} {
		if !strings.Contains(string(example), want) {
			t.Errorf("axiom.example.yaml no contiene %q", want)
		}
	}
}

// REQ-21.17 (4): `INDEX.md` lo produce `axiom archive sync`, no `archive-compose`.
func TestManualDeInicioAttributesIndexMdToArchiveSync(t *testing.T) {
	doc := readManualDeInicio(t)
	assertDocContains(t, doc,
		"Lo genera `axiom archive sync`, no `axiom sdd archive-compose`.",
		"`axiom archive sync` regenera `openspec/INDEX.md` (`archive-compose` no lo produce)",
	)
	// La línea antigua atribuía la actualización de INDEX.md a archive-compose.
	for _, retired := range []string{
		"consolida las especificaciones en `openspec/specs/`, actualiza `openspec/INDEX.md`",
	} {
		if strings.Contains(doc, retired) {
			t.Errorf("docs/manual-de-inicio.md conserva la atribución retirada %q", retired)
		}
	}
}
