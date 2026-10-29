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
// comentados. El verbo `axiom branch` que los aplica ya existe (inc-25, S12) y el
// manual remite a su sección.
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
		"Los bloques `git:` y `repositories:` los aplica el verbo `axiom branch`",
		"[Ramas y Worktrees (`axiom branch`)](#ramas-y-worktrees-axiom-branch)",
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
		"`axiom workspace validate` comprueba además en disco que cada `path` existe como directorio",
		"dos entradas que resuelven a la misma raíz git se rechazan nombrando ambas rutas y la raíz compartida",
		"Una ruta que no pertenece a ningún repositorio git solo produce una advertencia",
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

// readRepoFile lee un fichero versionado del repositorio con una ruta relativa a
// la raíz (los pins de scripts y documentos leen el texto que se publica).
func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("no se pudo leer %s: %v", rel, err)
	}
	return string(raw)
}

// repoMarkdownAndScripts lista los ficheros que forman la documentación y los
// scripts vigentes: la raíz, docs/ (sin docs/archive, que es histórico) y
// scripts/. openspec/ y odd/ quedan fuera: describen cambios, no instrucciones.
func repoMarkdownAndScripts(t *testing.T) []string {
	t.Helper()
	root := filepath.Join("..", "..")
	var files []string
	for _, name := range []string{"CLAUDE.md", "AGENTS.md", "GEMINI.md", "README.md", "axiom.example.yaml"} {
		files = append(files, name)
	}
	for _, dir := range []string{"docs", "scripts"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if filepath.ToSlash(path) == "../../docs/archive" {
					return filepath.SkipDir
				}
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			files = append(files, filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			t.Fatalf("no se pudo recorrer %s: %v", dir, err)
		}
	}
	return files
}

// REQ-25.24, D-21: cleanup-worktree.* se retira (solo lo citaban ODD-5.6 y el
// manual) y ningún documento ni script vigente lo cita; lo sustituye
// `axiom branch done`.
func TestScriptsCleanupWorktreeIsRetiredAndNoDocCitesIt(t *testing.T) {
	for _, name := range []string{"scripts/cleanup-worktree.ps1", "scripts/cleanup-worktree.sh"} {
		if _, err := os.Stat(filepath.Join("..", "..", filepath.FromSlash(name))); err == nil {
			t.Errorf("%s debería estar retirado", name)
		} else if !os.IsNotExist(err) {
			t.Fatalf("no se pudo comprobar %s: %v", name, err)
		}
	}

	files := repoMarkdownAndScripts(t)
	if len(files) < 10 {
		t.Fatalf("la comprobación solo ve %d ficheros; perdió cobertura", len(files))
	}
	for _, file := range files {
		if strings.Contains(readRepoFile(t, file), "cleanup-worktree") {
			t.Errorf("%s cita cleanup-worktree, retirado en favor de `axiom branch done`", file)
		}
	}
}

// REQ-25.24, D-21, G-12: axiom-worktree.* es un envoltorio fino de `axiom branch`
// (una sola implementación de ramas y worktrees), conserva `pr` y, si el binario
// no trae `axiom branch`, termina con el mensaje de instalación.
func TestScriptsAxiomWorktreeIsAThinWrapperOfAxiomBranch(t *testing.T) {
	for _, name := range []string{"scripts/axiom-worktree.ps1", "scripts/axiom-worktree.sh"} {
		script := readRepoFile(t, name)
		for _, want := range []string{
			"axiom branch ensure --cwd",
			"axiom branch done --cwd",
			"axiom branch status --cwd",
			"prTarget",
			"instala un `axiom` con `axiom branch` (`go install ./cmd/axiom`)",
			"axiom branch --help",
		} {
			if !strings.Contains(script, want) {
				t.Errorf("%s no contiene %q", name, want)
			}
		}
		// El verbo `pr` se conserva como entrega personal.
		if !strings.Contains(script, "gh pr create") || !strings.Contains(script, "push") {
			t.Errorf("%s perdió el verbo pr (push y gh pr create)", name)
		}
		// La implementación de ramas es la del producto: el script ya no ejecuta
		// `git worktree add`, ni borra ramas, ni desregistra proyectos del hub.
		for _, retired := range []string{
			"worktree add",
			"branch -d",
			"branch -D",
			"project remove",
			"cleanup-worktree",
		} {
			if strings.Contains(script, retired) {
				t.Errorf("%s conserva la implementación retirada %q", name, retired)
			}
		}
	}
}

// REQ-25.24, G-12: la guarda local de rutas se conserva, con su política por ruta
// y su fallo en abierto, y sus mensajes remiten a `axiom branch ensure` en lugar
// de al envoltorio `new`.
func TestScriptsGuardPrimaryCheckoutPointsToAxiomBranchEnsure(t *testing.T) {
	script := readRepoFile(t, "scripts/claude-hooks/guard-primary-checkout.ps1")
	for _, want := range []string{
		"axiom branch ensure",
		// La política por ruta y el fallo en abierto no cambian.
		"Test-InsidePath $targetPath $primary",
		"exit 0",
		"exit 2",
		"git -C $primary symbolic-ref",
		"rev-list --count origin/main..main",
		"La guarda falla en abierto",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("guard-primary-checkout.ps1 no contiene %q", want)
		}
	}
	for _, retired := range []string{
		"axiom-worktree.ps1 new",
		"worktree add ..\\axiom-wt\\<slug>",
	} {
		if strings.Contains(script, retired) {
			t.Errorf("guard-primary-checkout.ps1 sigue remitiendo a la creación manual %q", retired)
		}
	}
}

// REQ-25.24 (escenario «La documentación remite al verbo»): CLAUDE.md, AGENTS.md y
// GEMINI.md remiten a `axiom branch` para crear ramas y worktrees, al envoltorio
// local y a `done`, y ninguno afirma que el worktree se crea o se decide en apply.
// La prohibición local de las primitivas de cliente se conserva.
func TestDocsRemitToAxiomBranchEnsure(t *testing.T) {
	for _, file := range []string{"CLAUDE.md", "AGENTS.md", "GEMINI.md"} {
		doc := readRepoFile(t, file)
		for _, want := range []string{
			"axiom branch ensure",
			"axiom branch done",
			"scripts/axiom-worktree",
			"`EnterWorktree`",
			"`ExitWorktree`",
		} {
			if !strings.Contains(doc, want) {
				t.Errorf("%s no contiene %q", file, want)
			}
		}
		for _, retired := range []string{
			"lo crea en `apply`",
			"se crea en apply",
			"se crea en `apply`",
			"se decide en apply",
			"se decide en `apply`",
			"desde `origin/main` actualizado",
			"crea el worktree en `axiom-wt/<slug>`",
			"cleanup-worktree",
		} {
			if strings.Contains(doc, retired) {
				t.Errorf("%s conserva la afirmación retirada %q", file, retired)
			}
		}
	}

	// La instrucción gestionada de Codex que se versiona en el repositorio lleva la
	// doctrina nueva (paso 6, subsección de ramas y sección de kickoff) y no la regla
	// retirada. Se regenera con `axiom sync` (HOME aislado); aquí solo se fijan los
	// tres textos de ramas y worktrees.
	codex := readRepoFile(t, ".codex/AGENTS.md")
	for _, want := range []string{
		"### Branches and worktrees",
		"run `axiom branch ensure --lane odd --change <feature-name> --json`",
		"on the branch that `axiom branch ensure --lane odd --change <feature-name>` prepared",
		"run `axiom branch ensure --change <name> --json` and use the absolute paths of its JSON as `--cwd` of every later command",
		"Never use `EnterWorktree` or `ExitWorktree`; work in another worktree only through the absolute paths that `axiom branch ensure` returns",
	} {
		if !strings.Contains(codex, want) {
			t.Errorf(".codex/AGENTS.md no contiene %q", want)
		}
	}
	if strings.Contains(codex, "branch first when on the default branch") {
		t.Error(".codex/AGENTS.md conserva la regla retirada «branch first when on the default branch»")
	}

	claude := readRepoFile(t, "CLAUDE.md")
	for _, want := range []string{
		"`EnterWorktree` / `ExitWorktree` están prohibidos",
		"`axiom branch ensure --cwd <checkout principal> --change <slug> --mode worktree --json`",
		"`C:\\repos\\axiom-wt\\<slug>\\axiom`",
		"nunca ramificando en el directorio actual",
	} {
		if !strings.Contains(claude, want) {
			t.Errorf("CLAUDE.md no contiene %q", want)
		}
	}
}

// REQ-25.24 y REQ-25.1: el manual documenta los cuatro verbos de `axiom branch`,
// el flujo de la rama de archivado (PR del cambio, `ensure --archive`, PR
// documental), el hook de la capa D y su instalación por `axiom sync`, las
// preferencias personales del hub y la política local de este repositorio.
func TestManualDeInicioDocumentsTheBranchVerbsAndTheArchiveFlow(t *testing.T) {
	doc := readManualDeInicio(t)
	assertDocContains(t, doc,
		"[Ramas y Worktrees (`axiom branch`)](#ramas-y-worktrees-axiom-branch)",
		"### Ramas y Worktrees (`axiom branch`)",
		"- `axiom branch ensure --change <cambio>",
		"- `axiom branch status",
		"- `axiom branch done --change <cambio>",
		"- `axiom branch guard --agent <runtime>`",
		"`decision_required`",
		"`prTarget`",
		"La persona publica (`push`) y abre los PR; `axiom branch` nunca lo hace.",
		"`axiom project add --worktree-mode in_place --worktree-root <ruta absoluta>",
		"#### Flujo de la rama de archivado (monorepo)",
		"1. El PR del cambio se fusiona en la rama de integración.",
		"`axiom branch ensure --change <cambio> --archive`",
		"`<prefijo>/<cambio>-archive`",
		"`change_not_integrated`",
		"`--fetch`, que solo se usa con tu autorización",
		"un PR documental aparte hacia la rama de integración",
		"`axiom branch done --change <cambio>`",
		"el cambio sigue pendiente de archivar",
		"#### Hook de guarda (capa D) e instalación",
		"`axiom sync` instala `axiom branch guard` como hook en Claude Code, Codex y OpenCode (no en Pi)",
		"no instala hooks de git",
		"falla en abierto",
		"#### En este repositorio (política local)",
		"`git: {integration_branch: main, branch_prefix: inc}`",
		"`scripts/axiom-worktree.ps1`",
		"`scripts/claude-hooks/guard-primary-checkout.ps1`",
	)

	// La sección retirada ya no afirma que el repositorio de specs opera siempre en main
	// ni que el worktree se decida con un cuestionario previo.
	for _, retired := range []string{
		"Política Git Diferencial y Gestión de Worktrees",
		"opera **siempre en la rama principal (`main` o `master`)**",
		"cuestionario interactivo de pre-vuelo",
		"el verbo `axiom branch` que los aplicará llega en una entrega posterior",
	} {
		if strings.Contains(doc, retired) {
			t.Errorf("docs/manual-de-inicio.md conserva el texto retirado %q", retired)
		}
	}
}

// REQ-25.24: la visión de arquitectura remite a `axiom branch` para las ramas y
// los worktrees de cada topología.
func TestDocsEnterpriseVisionRemitsToAxiomBranch(t *testing.T) {
	doc := readRepoFile(t, "docs/architecture/AXIOM_ENTERPRISE_VISION.md")
	for _, want := range []string{
		"**Ramas y worktrees.**",
		"`axiom branch`",
		"`ensure`, `status`, `done` y `guard`",
		"rama `-archive`",
		"PR documental aparte",
		"`git:` y `repositories:`",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("AXIOM_ENTERPRISE_VISION.md no contiene %q", want)
		}
	}
}
