package dashboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Features ODD en el tablero de Incrementos: se detectan por el contenido del worktree (el documento
// `odd/tasks/*.md` difiere de la base), no por el carril del registro. Pruebas con git real, entorno
// aislado y sin red; los worktrees se crean con `git worktree add` y se registran a mano porque el
// registro de ramas los guarda igual que `scripts/axiom-worktree.ps1`.

func TestParseODDDoc(t *testing.T) {
	cases := []struct {
		name            string
		content         string
		title           string
		total, complete int
	}{
		{"título con dos puntos y casillas en negrita", "# ODD: Mi feature\n\n> **Estado: Cerrado (2026-10-27).**\n\n- [x] **T1 · a**\n- [ ] **T2 · b**\n- [X] **T3 · c**\n", "Mi feature", 3, 2},
		{"título con punto medio", "# ODD · distribucion-un-comando (P1)\n\n- [ ] T1\n", "distribucion-un-comando (P1)", 1, 0},
		{"título sin prefijo ODD", "# Patrón Canónico de Workspace\n- [x] a\n", "Patrón Canónico de Workspace", 1, 1},
		{"encabezados con casilla", "# ODD: x\n\n### [x] T1 · a\n\n### [ ] T2 · b\n\n## [x] T3 · c\n", "x", 3, 2},
		{"las casillas con sangría son subpasos si hay de primer nivel", "# ODD: x\n- [x] T1\n  - [ ] subpaso\n  - [x] otro\n- [ ] T2\n", "x", 2, 1},
		{"solo casillas con sangría", "# ODD: x\n  - [x] a\n  - [ ] b\n", "x", 2, 1},
		{"sin tareas", "# ODD: vacío\n\nTexto.\n", "vacío", 0, 0},
		{"finales de línea CRLF", "# ODD: crlf\r\n- [x] a\r\n- [ ] b\r\n", "crlf", 2, 1},
		{"solo cuenta el primer título", "# ODD: uno\n# Otro\n- [ ] a\n", "uno", 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := parseODDDoc(tc.content)
			if doc.Title != tc.title || doc.Total != tc.total || doc.Completed != tc.complete {
				t.Errorf("parseODDDoc = %+v, se esperaba título %q y %d/%d", doc, tc.title, tc.complete, tc.total)
			}
		})
	}
}

func TestODDPhase(t *testing.T) {
	for doc, want := range map[oddDoc]string{
		{}:                       "explore",
		{Total: 3, Completed: 1}: "apply",
		{Total: 3, Completed: 3}: "verify",
	} {
		if got := oddPhase(doc); got != want {
			t.Errorf("oddPhase(%+v) = %s, se esperaba %s", doc, got, want)
		}
	}
}

// oddProject es un monorepo embebido con git en `main` y un commit base. Los ficheros dados se
// escriben antes del commit (rutas relativas a la raíz).
func oddProject(t *testing.T, files map[string]string) workspacetest.Workspace {
	t.Helper()
	ws := workspacetest.Embedded(t, ".")
	for rel, content := range files {
		workspacetest.WriteFile(t, filepath.Join(ws.Master, filepath.FromSlash(rel)), content)
	}
	commitGitRepo(t, ws.Master)
	return ws
}

// oddWorktree crea el worktree de la rama `inc/<slug>` desde `main` y lo registra en el carril dado,
// como lo haría `axiom branch ensure`. Devuelve su ruta.
func oddWorktree(t *testing.T, ws workspacetest.Workspace, lane branch.Lane, slug string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), slug)
	runGit(t, ws.Master, "worktree", "add", "-q", "-b", "inc/"+slug, dir, "main")
	common, err := gitcommon.CommonDirFromFiles(ws.Master)
	if err != nil {
		t.Fatalf("git common dir: %v", err)
	}
	record := branch.Record{
		Schema: branch.RegistrySchema, Lane: lane, Change: slug, Mode: branch.ModeWorktree,
		ModeSource: branch.ModeSourceDefault, PlanningRepo: ws.Master,
		Repositories: []branch.RecordRepo{{
			Name: filepath.Base(ws.Master), GitRoot: ws.Master, CommonDir: common, Role: branch.RepoChangeBranch,
			Branches: []branch.RecordBranch{{Name: "inc/" + slug, Kind: branch.KindChange, Worktree: dir}},
		}},
	}
	if err := registry.New().Save(common, record); err != nil {
		t.Fatalf("registrar el worktree %s: %v", slug, err)
	}
	return dir
}

func writeODD(t *testing.T, root, name, content string) {
	t.Helper()
	workspacetest.WriteFile(t, filepath.Join(root, "odd", "tasks", name+".md"), content)
}

func commitAll(t *testing.T, dir, message string) {
	t.Helper()
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-q", "-m", message)
}

// oddItems son los incrementos del listado con `change_type: "odd"`, por tipo y nombre.
func (r listingResponse) oddItems() map[string]IncrementSummaryDTO {
	items := map[string]IncrementSummaryDTO{}
	for _, increment := range r.list {
		if increment.ChangeType == "odd" {
			items[increment.Type+":"+increment.Name] = increment
		}
	}
	return items
}

func (r listingResponse) wantODD(t *testing.T, keys ...string) {
	t.Helper()
	items := r.oddItems()
	if len(items) != len(keys) {
		t.Errorf("features ODD = %v, se esperaban %v", items, keys)
	}
	for _, key := range keys {
		if _, ok := items[key]; !ok {
			t.Errorf("falta %s entre las features ODD: %v", key, items)
		}
	}
}

// Una feature ODD sin commit (documento sin seguimiento) figura como activa, con su rama, su ruta y su
// progreso, aunque el worktree esté en el carril `sdd` del registro.
func TestIncrementsListsAnODDFeatureOfAWorktree(t *testing.T) {
	ws := oddProject(t, nil)
	worktree := oddWorktree(t, ws, branch.LaneSDD, "mi-feature")
	writeODD(t, worktree, "mi-feature", "# ODD: Mi feature\n\n- [x] **T1 · a**\n- [ ] **T2 · b**\n")

	listing := getListing(t, NewService(ws.Master))

	listing.wantODD(t, "active:mi-feature")
	item := listing.oddItems()["active:mi-feature"]
	if item.Phase != "apply" || item.TasksTotal != 2 || item.TasksCompleted != 1 || item.ProgressPct != 50 {
		t.Errorf("fase o progreso erróneos: %+v", item)
	}
	if item.Title != "Mi feature" || item.Branch != "inc/mi-feature" {
		t.Errorf("título o rama erróneos: %+v", item)
	}
	wantSameDirectory(t, "worktreePath", item.WorktreePath, worktree)
	if raw := listing.rawSummary(t, "mi-feature"); raw["change_type"] != "odd" || raw["type"] != "active" {
		t.Errorf("change_type y type deben viajar como odd y active: %v", raw)
	}
	for _, note := range listing.notes {
		if strings.Contains(note, "integrado") {
			t.Errorf("un worktree con trabajo en curso no debía avisarse como integrado: %q", note)
		}
	}
}

// El carril `odd` del registro se enumera igual que el `sdd`.
func TestIncrementsListsAnODDFeatureOfAnODDLaneWorktree(t *testing.T) {
	ws := oddProject(t, nil)
	worktree := oddWorktree(t, ws, branch.LaneODD, "otra")
	writeODD(t, worktree, "otra", "# ODD: Otra\n- [ ] T1\n")

	getListing(t, NewService(ws.Master)).wantODD(t, "active:otra")
}

// Las fases salen de las tareas del documento, y un documento ya confirmado en la rama (diferencia
// con la base) cuenta igual que uno sin seguimiento.
func TestIncrementsODDPhaseFollowsTheTasksOfACommittedDocument(t *testing.T) {
	ws := oddProject(t, nil)
	explore := oddWorktree(t, ws, branch.LaneSDD, "sin-tareas")
	writeODD(t, explore, "sin-tareas", "# ODD: Sin tareas\n\nSolo texto.\n")
	commitAll(t, explore, "docs: sin tareas")
	verify := oddWorktree(t, ws, branch.LaneSDD, "todo-hecho")
	writeODD(t, verify, "todo-hecho", "# ODD: Todo hecho\n- [x] a\n- [x] b\n")
	commitAll(t, verify, "docs: todo hecho")

	items := getListing(t, NewService(ws.Master)).oddItems()

	if got := items["active:sin-tareas"].Phase; got != "explore" {
		t.Errorf("sin tareas: fase %q, se esperaba explore", got)
	}
	if got := items["active:todo-hecho"]; got.Phase != "verify" || got.ProgressPct != 100 {
		t.Errorf("todas marcadas: %+v, se esperaba verify al 100%%", got)
	}
}

// Si un documento se llama como el slug del worktree, solo ese es la feature; los demás documentos que
// difieren de la base se ignoran.
func TestIncrementsODDPrefersTheDocumentNamedAsTheSlug(t *testing.T) {
	ws := oddProject(t, nil)
	worktree := oddWorktree(t, ws, branch.LaneSDD, "propio")
	writeODD(t, worktree, "propio", "# ODD: Propio\n- [ ] T1\n")
	writeODD(t, worktree, "ajeno", "# ODD: Ajeno\n- [ ] T1\n")

	getListing(t, NewService(ws.Master)).wantODD(t, "active:propio")
}

// Sin documento con el slug, cuentan los añadidos en la rama o sin seguimiento; los modificados solo
// si no hay ninguno añadido.
func TestIncrementsODDFallsBackToAddedAndThenModifiedDocuments(t *testing.T) {
	ws := oddProject(t, map[string]string{"odd/tasks/previo.md": "# ODD: Previo\n- [ ] T1\n"})
	added := oddWorktree(t, ws, branch.LaneSDD, "con-nuevo")
	writeODD(t, added, "nuevo", "# ODD: Nuevo\n- [ ] T1\n")
	writeODD(t, added, "previo", "# ODD: Previo\n- [x] T1\n")
	modified := oddWorktree(t, ws, branch.LaneSDD, "solo-modifica")
	writeODD(t, modified, "previo", "# ODD: Previo\n- [x] T1\n")

	items := getListing(t, NewService(ws.Master)).oddItems()

	if _, ok := items["active:nuevo"]; !ok {
		t.Errorf("el documento añadido debía listarse: %v", items)
	}
	if item, ok := items["active:previo"]; !ok || item.Branch != "inc/solo-modifica" {
		t.Errorf("previo solo debía listarse como activo desde el worktree que lo modifica: %v", items)
	}
	if len(items) != 2 { // nuevo y previo activos; el archivado de previo cede ante el worktree
		t.Errorf("features ODD = %v, se esperaban nuevo y previo activos", items)
	}
}

// Un worktree integrado (su rama es ancestro de la base, sin cambios sin confirmar y con la punta
// distinta de la base) no aporta feature activa, genera un aviso y su documento queda como archivado
// en el checkout principal.
func TestIncrementsODDIntegratedWorktreeIsNotActiveAndIsNoted(t *testing.T) {
	ws := oddProject(t, nil)
	worktree := oddWorktree(t, ws, branch.LaneSDD, "ya-fusionada")
	writeODD(t, worktree, "ya-fusionada", "# ODD: Ya fusionada\n- [x] T1\n")
	commitAll(t, worktree, "feat: ya fusionada")
	runGit(t, ws.Master, "merge", "-q", "--no-ff", "-m", "Merge inc/ya-fusionada", "inc/ya-fusionada")

	listing := getListing(t, NewService(ws.Master))

	listing.wantODD(t, "archived:ya-fusionada")
	note := listing.noteMentioning(t, "ya-fusionada", "inc/ya-fusionada", "ya está integrado en main", "axiom branch done --change ya-fusionada")
	if !strings.HasPrefix(note, "El worktree ya-fusionada") {
		t.Errorf("aviso inesperado: %q", note)
	}
}

// Un worktree recién creado (rama en la punta de la base, sin cambios) también es ancestro de la base
// y está limpio, pero no está integrado: ni feature ni aviso.
func TestIncrementsODDFreshWorktreeHasNoItemAndNoNote(t *testing.T) {
	ws := oddProject(t, nil)
	oddWorktree(t, ws, branch.LaneSDD, "recien-creado")

	listing := getListing(t, NewService(ws.Master))

	listing.wantODD(t)
	for _, note := range listing.notes {
		if strings.Contains(note, "recien-creado") {
			t.Errorf("un worktree recién creado no debía avisarse: %q", note)
		}
	}
}

// Un worktree integrado con cambios sin confirmar vuelve a ser trabajo en curso.
func TestIncrementsODDIntegratedWorktreeWithUncommittedChangesIsActive(t *testing.T) {
	ws := oddProject(t, nil)
	worktree := oddWorktree(t, ws, branch.LaneSDD, "retomada")
	writeODD(t, worktree, "retomada", "# ODD: Retomada\n- [x] T1\n")
	commitAll(t, worktree, "feat: retomada")
	runGit(t, ws.Master, "merge", "-q", "--no-ff", "-m", "Merge inc/retomada", "inc/retomada")
	writeODD(t, worktree, "retomada", "# ODD: Retomada\n- [x] T1\n- [ ] T2\n")

	listing := getListing(t, NewService(ws.Master))

	if item, ok := listing.oddItems()["active:retomada"]; !ok || item.TasksTotal != 2 {
		t.Errorf("retomada debía listarse como activa con 2 tareas: %v", listing.oddItems())
	}
	for _, note := range listing.notes {
		if strings.Contains(note, "ya está integrado") {
			t.Errorf("no debía avisarse como integrado: %q", note)
		}
	}
}

// Los documentos ODD del checkout principal se listan como archivados, salvo los que están activos en
// un worktree: el worktree prevalece y el documento figura una sola vez.
func TestIncrementsODDMainCheckoutDocumentsAreArchivedAndTheWorktreePrevails(t *testing.T) {
	ws := oddProject(t, map[string]string{
		"odd/tasks/viejo.md": "# ODD: Viejo\n- [x] T1\n- [x] T2\n",
		"odd/tasks/doble.md": "# ODD: Doble\n- [ ] T1\n",
		"odd/tasks/nota.txt": "no es un documento ODD\n",
	})
	worktree := oddWorktree(t, ws, branch.LaneSDD, "doble")
	writeODD(t, worktree, "doble", "# ODD: Doble\n- [x] T1\n- [ ] T2\n")

	listing := getListing(t, NewService(ws.Master))

	listing.wantODD(t, "archived:viejo", "active:doble")
	viejo := listing.oddItems()["archived:viejo"]
	if viejo.Phase != "archive" || viejo.ProgressPct != 100 || viejo.WorktreePath != "" {
		t.Errorf("viejo debía ser un archivado completo del checkout principal: %+v", viejo)
	}
	doble := listing.oddItems()["active:doble"]
	if doble.TasksTotal != 2 || doble.TasksCompleted != 1 {
		t.Errorf("doble debía leerse del worktree (1/2): %+v", doble)
	}
	wantSameDirectory(t, "worktreePath", doble.WorktreePath, worktree)
}

// El worktree ODD desaparecido se omite con el aviso del motor, sin error.
func TestIncrementsODDMissingWorktreeIsOmittedWithANote(t *testing.T) {
	ws := oddProject(t, nil)
	worktree := oddWorktree(t, ws, branch.LaneODD, "perdida")
	if err := os.RemoveAll(worktree); err != nil {
		t.Fatal(err)
	}

	listing := getListing(t, NewService(ws.Master))

	listing.wantODD(t)
	listing.noteMentioning(t, "ya no existe", "perdida")
}
