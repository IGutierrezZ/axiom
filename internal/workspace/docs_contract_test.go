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
		"`technical_context` lives only in `axiom.yaml`",
		"`baseline_retired`",
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

// REQ-30.21 (design §9): el manual documenta la baseline retirada, el error tipado
// que produce, los verbos retirados y la migración por adopción, con el aviso de
// usar el binario nuevo en todo el equipo.
func TestManualDeInicioDocumentsTheRetiredBaselineAndItsMigration(t *testing.T) {
	manual := readManualDeInicio(t)
	assertDocContains(t, manual,
		"### Baseline Retirada y Migración",
		"El bloque `baseline:` de `axiom.yaml` ya no existe: la única spec viva es OpenSpec (`openspec/specs/`)",
		"el error tipado `baseline_retired`, que nombra la ruta, la línea y la continuación",
		"`axiom workspace validate --path <maestra>` sale con código `1`",
		"bloquea solo el archivado",
		"`axiom init` se rechaza antes de escribir ni registrar nada",
		"`axiom sdd integrate` ya no existe: remite a `axiom archive context check|prepare|verify|show`",
		"`axiom sdd archive-compose` solo admite `--canonical --delta`",
		"#### Migración en tres pasos",
		"1. Retira el bloque `baseline:` de `axiom.yaml`.",
		"2. Migra sus requisitos a `openspec/specs/` con `axiom archive adopt scan --path <maestra> --change <cambio>`",
		"3. Ejecuta `axiom workspace validate --path <maestra>` hasta que sea conforme.",
		"todo el equipo debe usar el binario nuevo antes de migrar",
	)
	for _, retired := range []string{"baseline:\n  documents:", "integration: directed", "integration: register", "axiom sdd integrate check"} {
		if strings.Contains(manual, retired) {
			t.Errorf("el manual aún describe la baseline retirada: %q", retired)
		}
	}
}

// D-14 (design §9): el manual indica, para los runtimes con el modelo fijado por
// agente, que se asigne a sdd-archive un modelo de nivel de implementación.
func TestManualDeInicioDocumentsTheArchiveModelForMaintainedContext(t *testing.T) {
	assertDocContains(t, readManualDeInicio(t),
		"#### Modelo de `sdd-archive` con contexto mantenido",
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

// workspace-topology REQ-27.1 y organic-agent-trigger-rules REQ-27.17: el manual
// documenta el bloque `workflow:` (valores, defecto `sdd`, cambio de comportamiento
// para proyectos sin la clave, proyección en `axiom sdd status`) y que ODD no tiene
// paradas propias; la plantilla muestra el bloque comentado y ninguna documentación
// vigente conserva la parada única del plan de ODD.
func TestDocsDocumentTheWorkflowLaneBlock(t *testing.T) {
	doc := readManualDeInicio(t)
	assertDocContains(t, doc,
		"[Carril de Trabajo (`workflow:`)](#carril-de-trabajo-workflow)",
		"### Carril de Trabajo (`workflow:`)",
		"workflow:\n  lane: ask",
		"`odd`: carril ágil",
		"`sdd`: carril formal",
		"`ask`: conserva la pregunta bloqueante",
		"Este repositorio declara `ask`",
		"Sin el bloque, o sin la clave `lane`, rige `sdd`",
		"**Cambio de comportamiento por defecto:**",
		"`workflow: {lane: ask}`",
		"El valor distingue mayúsculas (`SDD` es inválido)",
		"`axiom workspace validate` rechaza un bloque inválido nombrando `workflow.lane`",
		"`axiom sdd status --json` proyecta el carril efectivo en el campo `workflow`",
		"Sin `axiom.yaml`, o con uno inválido, proyecta `ask`",
		"ODD no tiene paradas propias",
		"Los binarios anteriores ignoran el bloque, como el de `git:`",
	)
	if strings.Contains(doc, "parada única del plan de ODD") {
		t.Error("docs/manual-de-inicio.md conserva la parada única del plan de ODD, retirada por REQ-27.17")
	}

	example := readRepoFile(t, "axiom.example.yaml")
	for _, want := range []string{"# workflow:", "#   lane: ask"} {
		if !strings.Contains(example, want) {
			t.Errorf("axiom.example.yaml no contiene %q", want)
		}
	}
	for _, file := range []string{"AGENTS.md", "GEMINI.md"} {
		text := readRepoFile(t, file)
		if !strings.Contains(text, "`workflow.lane`") {
			t.Errorf("%s no menciona el carril del proyecto (`workflow.lane`)", file)
		}
		if strings.Contains(text, "parada de plan") || strings.Contains(text, "Plan stop") {
			t.Errorf("%s conserva la parada de plan de ODD", file)
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

	// Codex no carga .codex/AGENTS.md en el proyecto y `axiom sync` lo retira como
	// fichero heredado (LegacySystemPromptFiles de internal/agents/codex): una copia
	// versionada no la regenera nada y se desfasa sin que ningún test lo note. La
	// doctrina de ramas y worktrees que proyecta Codex se fija en su origen:
	// internal/components/agentguidance (todos los agentes) e internal/assets (kickoff).
	if _, err := os.Stat(filepath.Join("..", "..", ".codex", "AGENTS.md")); !os.IsNotExist(err) {
		t.Errorf(".codex/AGENTS.md no debe versionarse: Codex no lo lee y axiom sync lo retira (err = %v)", err)
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

// sdd-block-review-gates REQ-27.12, change-branch-lifecycle REQ-27.15 y sdd-lane-kickoff REQ-27.4: el
// manual documenta que aprobar una parada publica los documentos con `axiom sdd publish`, que quien
// continúa adopta la rama con `git fetch` y `ensure` sin PR fusionada, la autorización
// `docs_publication: auto` del kickoff y la nota del CI ajeno; ni el manual ni AGENTS.md ni GEMINI.md
// conservan la exigencia retirada de una PR fusionada.
func TestDocsDocumentThePublicationOfTheChangeDocuments(t *testing.T) {
	doc := readManualDeInicio(t)
	assertDocContains(t, doc,
		"«aprobar, publicar y parar» y «aprobar, publicar y seguir»",
		"la aprobación es la autorización explícita de subir los documentos del cambio a `origin/<rama>`",
		"el push de código, la PR y el merge siguen siendo de la persona",
		"`git fetch origin` (o `axiom branch ensure --fetch`, con autorización) y `axiom branch ensure --change <cambio>`, que adopta la rama publicada",
		"`gate record` solo registra",
		"`publish.invocation`",
		"`publishUnavailable`",
		"Una publicación rechazada no deshace la aprobación ya registrada",
		"`--docs-publication auto`",
		"`docs_publication: auto` en `kickoff.yaml`",
		"el push de la rama puede disparar su CI, que queda fuera del control de Axiom",
	)
	for _, retired := range []string{
		"solo empieza cuando la PR esté fusionada",
		"no recupera trabajo sin fusionar",
		"aprobar, subir y parar",
		"se rechaza sin escribir nada si la carpeta del cambio",
	} {
		if strings.Contains(doc, retired) {
			t.Errorf("docs/manual-de-inicio.md conserva la redacción retirada %q", retired)
		}
	}
	for _, file := range []string{"AGENTS.md", "GEMINI.md"} {
		text := readRepoFile(t, file)
		if !strings.Contains(text, "aprobar-publicar-y-parar") || !strings.Contains(text, "`axiom sdd publish`") {
			t.Errorf("%s no describe la publicación de los documentos en las salidas de la parada", file)
		}
		if strings.Contains(text, "«Subir» significa una PR fusionada") {
			t.Errorf("%s conserva la exigencia retirada de una PR fusionada", file)
		}
	}
}
