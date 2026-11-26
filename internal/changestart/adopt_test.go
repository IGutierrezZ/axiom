package changestart

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

func excludeContent(env Env) string {
	data, _ := os.ReadFile(filepath.Join(env.CommonDir, "info", "exclude"))
	return string(data)
}

func TestAdoptSDD(t *testing.T) {
	env := preparedEnv(t, branch.LaneSDD)
	seedFullStaging(t, env, "2525")
	tr := newAdoptTracker()

	res := mustAdopt(t, env, tr, LaneSDD)

	if res.Status != StatusAdopted || res.Link != LinkCreated || res.Backlink != BacklinkAdded || res.Lane != LaneSDD || res.ExternalRef != "ado:2525" {
		t.Fatalf("resultado = %+v", res)
	}
	dest := destDir(env, LaneSDD)
	if want := filepath.Join(env.changeRoot(adoptChange), "context"); dest != want || res.Context != want {
		t.Fatalf("destino = %s (resultado %s), quiero %s", dest, res.Context, want)
	}
	if got, _ := os.ReadFile(filepath.Join(dest, "attachments", "tasks.md")); string(got) != "adjunto con nombre de artefacto\n" {
		t.Errorf("el adjunto no se copió bajo attachments/: %q", got)
	}
	if exists(env.StagingDir(adoptChange)) {
		t.Error("el staging debe retirarse")
	}
	links, err := integrations.WorkItemLinks(integrations.Folder{WorkspaceRoot: env.WorkspaceRoot, Root: env.changeRoot(adoptChange)})
	if err != nil || len(links) != 1 || links[0].Plugin != "azure-devops" || links[0].ID != "2525" || links[0].Source != "change.start" || links[0].URL == "" {
		t.Fatalf("vínculos = %+v, %v", links, err)
	}
	doc := readDest(t, env, LaneSDD)
	if doc.Lane != "sdd" || doc.Backlink != "done" || doc.AdoptedAt == "" {
		t.Fatalf("work-item.json = %+v", doc)
	}
	if tr.calls != 1 || tr.lastTag != "axiom:"+adoptChange || tr.writes != 1 {
		t.Fatalf("enlace de vuelta: llamadas %d, etiqueta %q, escrituras %d", tr.calls, tr.lastTag, tr.writes)
	}
	exclude := excludeContent(env)
	for _, entry := range []string{"/openspec/changes/*/context/", "/openspec/changes/*/.context-tmp/", "/odd/context/"} {
		if strings.Count(exclude, entry+"\n") != 1 {
			t.Errorf("info/exclude no contiene %q una vez:\n%s", entry, exclude)
		}
	}
	if exists(filepath.Join(env.changeRoot(adoptChange), ".context-tmp")) {
		t.Error("no deben quedar temporales de la copia")
	}
}

func TestAdoptODDNoIntegrationsYAML(t *testing.T) {
	env := preparedEnv(t, branch.LaneODD)
	seedFullStaging(t, env, "2525")

	res := mustAdopt(t, env, newAdoptTracker(), LaneODD)

	if res.Status != StatusAdopted || res.Link != LinkNotApplicable || res.Lane != LaneODD {
		t.Fatalf("resultado = %+v", res)
	}
	dest := filepath.Join(env.WorkspaceRoot, "odd", "context", adoptChange)
	if !exists(filepath.Join(dest, "work-item.md")) || !exists(filepath.Join(dest, "attachments", "tasks.md")) {
		t.Fatalf("el contexto debe quedar directamente bajo %s", dest)
	}
	if exists(filepath.Join(env.WorkspaceRoot, "openspec")) {
		t.Error("el carril ODD no crea nada bajo openspec/")
	}
	if doc := readDest(t, env, LaneODD); doc.Lane != "odd" || doc.Backlink != "done" {
		t.Fatalf("work-item.json = %+v", doc)
	}
	if exists(filepath.Join(env.WorkspaceRoot, "odd", "context", ".tmp")) {
		t.Error("no deben quedar temporales de la copia")
	}
}

func TestAdoptTwiceAlreadyAdoptedNoNetwork(t *testing.T) {
	for _, lane := range []branch.Lane{branch.LaneSDD, branch.LaneODD} {
		t.Run(string(lane), func(t *testing.T) {
			env := preparedEnv(t, lane)
			seedFullStaging(t, env, "2525")
			tr := newAdoptTracker()
			mustAdopt(t, env, tr, Lane(lane))
			before := snapshotTree(t, env.WorkspaceRoot, ".git")
			calls := tr.calls

			res := mustAdopt(t, env, tr, Lane(lane))

			if res.Status != StatusAlreadyAdopted || res.ID != "2525" || res.Context == "" {
				t.Fatalf("resultado = %+v", res)
			}
			if tr.calls != calls || tr.reads != 0 {
				t.Fatalf("la segunda adopción no usa la red: %d llamadas más", tr.calls-calls)
			}
			if after := snapshotTree(t, env.WorkspaceRoot, ".git"); !sameSnapshot(before, after) {
				t.Fatalf("la segunda adopción no cambia el árbol:\nantes %v\ndespués %v", before, after)
			}
			if lane == branch.LaneSDD {
				links, _ := integrations.WorkItemLinks(integrations.Folder{WorkspaceRoot: env.WorkspaceRoot, Root: env.changeRoot(adoptChange)})
				if len(links) != 1 {
					t.Fatalf("un único vínculo del work item: %+v", links)
				}
			}
			if tr.writes != 1 {
				t.Fatalf("una sola etiqueta: %d escrituras", tr.writes)
			}
		})
	}
}

func TestAdoptInterruptedRetry(t *testing.T) {
	env := preparedEnv(t, branch.LaneSDD)
	seedFullStaging(t, env, "2525")
	tr := newAdoptTracker()
	// Interrupción tras el movimiento: el contexto está en el destino con el mismo work item y el
	// staging sigue presente; faltan el vínculo, la exclusión y el resto.
	if _, err := copyContextForTest(env); err != nil {
		t.Fatal(err)
	}

	res := mustAdopt(t, env, tr, LaneSDD)

	if res.Status != StatusAdopted || res.Link != LinkCreated {
		t.Fatalf("resultado = %+v", res)
	}
	if exists(env.StagingDir(adoptChange)) {
		t.Error("el reintento retira el staging")
	}
	links, _ := integrations.WorkItemLinks(integrations.Folder{WorkspaceRoot: env.WorkspaceRoot, Root: env.changeRoot(adoptChange)})
	if len(links) != 1 || tr.writes != 1 {
		t.Fatalf("sin duplicados: %d vínculos, %d etiquetas", len(links), tr.writes)
	}
	if strings.Count(excludeContent(env), "/odd/context/\n") != 1 {
		t.Errorf("la exclusión se completa una sola vez:\n%s", excludeContent(env))
	}
}

// copyContextForTest simula un paso 2 ya hecho: copia el staging al destino SDD.
func copyContextForTest(env Env) (string, error) {
	env = env.withDefaults()
	dest, tmpBase := env.contextPaths(LaneSDD, adoptChange)
	return env.copyContext(filepath.Join(env.StagingDir(adoptChange), "context"), dest, tmpBase)
}

func TestAdoptDestinationOtherItemConflict(t *testing.T) {
	env := preparedEnv(t, branch.LaneSDD)
	seedFullStaging(t, env, "2525")
	writeFile(t, filepath.Join(destDir(env, LaneSDD), "work-item.json"), contextJSON(adoptChange, "azure-devops", "3000", ""))
	writeFile(t, filepath.Join(destDir(env, LaneSDD), "work-item.md"), "otro work item\n")
	before := snapshotTree(t, env.WorkspaceRoot, ".git/axiom/change-start/v1/.locks")

	r := adoptRefusal(t, env, newAdoptTracker(), LaneSDD)

	if r.Code != CodeStagingConflict {
		t.Fatalf("código = %s", r.Code)
	}
	if after := snapshotTree(t, env.WorkspaceRoot, ".git/axiom/change-start/v1/.locks"); !sameSnapshot(before, after) {
		t.Fatal("el destino y el staging deben quedar intactos")
	}
}

func TestAdoptDestinationWithoutJSONConflict(t *testing.T) {
	env := preparedEnv(t, branch.LaneSDD)
	seedFullStaging(t, env, "2525")
	writeFile(t, filepath.Join(destDir(env, LaneSDD), "work-item.md"), "sin json\n")
	if r := adoptRefusal(t, env, newAdoptTracker(), LaneSDD); r.Code != CodeStagingConflict {
		t.Fatalf("código = %s", r.Code)
	}
	if !exists(filepath.Join(env.StagingDir(adoptChange), "context", "work-item.json")) {
		t.Fatal("el staging queda intacto")
	}
}

func TestAdoptWithoutStaging(t *testing.T) {
	for _, lane := range []branch.Lane{branch.LaneSDD, branch.LaneODD} {
		env := preparedEnv(t, lane)
		if r := adoptRefusal(t, env, newAdoptTracker(), Lane(lane)); r.Code != CodeStagingNotFound {
			t.Fatalf("%s: código = %s", lane, r.Code)
		}
		// Un context/ sin work-item.json legible tampoco es qué adoptar.
		writeFile(t, filepath.Join(destDir(env, Lane(lane)), "work-item.md"), "sin json\n")
		if r := adoptRefusal(t, env, newAdoptTracker(), Lane(lane)); r.Code != CodeStagingNotFound {
			t.Fatalf("%s sin json: código = %s", lane, r.Code)
		}
	}
}

func TestAdoptChangeNotPrepared(t *testing.T) {
	t.Run("sin_registro", func(t *testing.T) {
		env := newRepo(t)
		setHead(t, env, "inc/"+adoptChange)
		seedFullStaging(t, env, "2525")
		before := snapshotTree(t, env.WorkspaceRoot, ".git/axiom/change-start/v1/.locks")
		r := adoptRefusal(t, env, newAdoptTracker(), LaneSDD)
		if r.Code != CodeChangeNotPrepared || r.Fields["reason"] != "no_record" {
			t.Fatalf("rechazo = %+v", r)
		}
		if after := snapshotTree(t, env.WorkspaceRoot, ".git/axiom/change-start/v1/.locks"); !sameSnapshot(before, after) {
			t.Fatal("el staging queda intacto y no se escribe nada")
		}
	})
	t.Run("sin_carpeta", func(t *testing.T) {
		env := preparedEnv(t, branch.LaneSDD)
		if err := os.RemoveAll(env.changeRoot(adoptChange)); err != nil {
			t.Fatal(err)
		}
		seedFullStaging(t, env, "2525")
		if r := adoptRefusal(t, env, newAdoptTracker(), LaneSDD); r.Code != CodeChangeNotPrepared || r.Fields["reason"] != "folder_missing" {
			t.Fatalf("rechazo = %+v", r)
		}
		if exists(env.changeRoot(adoptChange)) {
			t.Fatal("la adopción SDD no crea la carpeta del cambio")
		}
	})
	t.Run("registro_ilegible", func(t *testing.T) {
		env := preparedEnv(t, branch.LaneSDD)
		seedFullStaging(t, env, "2525")
		writeFile(t, filepath.Join(env.CommonDir, "axiom", "branch", "v1", "sdd", adoptChange+".json"), "{no es json")
		r := adoptRefusal(t, env, newAdoptTracker(), LaneSDD)
		if r.Code != CodeChangeNotPrepared || r.Fields["reason"] != "record_unreadable" || r.Fields["detail"] == "" {
			t.Fatalf("rechazo = %+v", r)
		}
	})
	t.Run("odd_no_exige_carpeta", func(t *testing.T) {
		env := preparedEnv(t, branch.LaneODD)
		seedFullStaging(t, env, "2525")
		if res := mustAdopt(t, env, newAdoptTracker(), LaneODD); res.Status != StatusAdopted {
			t.Fatalf("resultado = %+v", res)
		}
	})
}

func TestAdoptLaneMismatch(t *testing.T) {
	t.Run("sdd_default_odd_record", func(t *testing.T) {
		env := newRepo(t)
		setHead(t, env, "inc/"+adoptChange)
		worktree := t.TempDir()
		seedRecord(t, env, branch.LaneODD, adoptChange, worktree)
		seedFullStaging(t, env, "2525")
		r := adoptRefusal(t, env, newAdoptTracker(), LaneSDD)
		if r.Code != CodeChangeNotPrepared || r.Fields["reason"] != "lane_mismatch" || r.Fields["record_lane"] != "odd" || r.Fields["worktree"] != worktree {
			t.Fatalf("rechazo = %+v", r)
		}
		if !exists(filepath.Join(env.StagingDir(adoptChange), "context", "work-item.json")) {
			t.Fatal("el staging queda intacto")
		}
	})
	t.Run("odd_on_sdd_record", func(t *testing.T) {
		env := preparedEnv(t, branch.LaneSDD)
		seedFullStaging(t, env, "2525")
		r := adoptRefusal(t, env, newAdoptTracker(), LaneODD)
		if r.Code != CodeChangeNotPrepared || r.Fields["reason"] != "lane_mismatch" || r.Fields["record_lane"] != "sdd" {
			t.Fatalf("rechazo = %+v", r)
		}
	})
	t.Run("odd_head_other_branch", func(t *testing.T) {
		env := preparedEnv(t, branch.LaneODD)
		setHead(t, env, "main")
		seedFullStaging(t, env, "2525")
		r := adoptRefusal(t, env, newAdoptTracker(), LaneODD)
		if r.Code != CodeChangeNotPrepared || r.Fields["reason"] != "head_other_branch" {
			t.Fatalf("rechazo = %+v", r)
		}
	})
}

func TestAdoptHeadOtherBranch(t *testing.T) {
	env := preparedEnv(t, branch.LaneSDD)
	setHead(t, env, "main")
	seedFullStaging(t, env, "2525")
	before := snapshotTree(t, env.WorkspaceRoot, ".git/axiom/change-start/v1/.locks")
	r := adoptRefusal(t, env, newAdoptTracker(), LaneSDD)
	if r.Code != CodeChangeNotPrepared || r.Fields["reason"] != "head_other_branch" {
		t.Fatalf("rechazo = %+v", r)
	}
	if after := snapshotTree(t, env.WorkspaceRoot, ".git/axiom/change-start/v1/.locks"); !sameSnapshot(before, after) {
		t.Fatal("no se escribe nada")
	}
	// HEAD desacoplado: tampoco es la rama del cambio.
	writeFile(t, filepath.Join(env.CommonDir, "HEAD"), strings.Repeat("a", 40)+"\n")
	if r := adoptRefusal(t, env, newAdoptTracker(), LaneSDD); r.Fields["reason"] != "head_other_branch" {
		t.Fatalf("HEAD desacoplado: %+v", r)
	}
}

func TestAdoptRefusesMainCheckout(t *testing.T) {
	// En el checkout principal (HEAD en main) la carpeta del cambio no existe: el motivo es el HEAD.
	env := preparedEnv(t, branch.LaneSDD)
	setHead(t, env, "main")
	if err := os.RemoveAll(env.changeRoot(adoptChange)); err != nil {
		t.Fatal(err)
	}
	seedFullStaging(t, env, "2525")
	if r := adoptRefusal(t, env, newAdoptTracker(), LaneSDD); r.Fields["reason"] != "head_other_branch" {
		t.Fatalf("rechazo = %+v", r)
	}
}

func TestAdoptUnknownHeadWarns(t *testing.T) {
	env := preparedEnv(t, branch.LaneSDD)
	writeFile(t, filepath.Join(env.CommonDir, "HEAD"), "ref: refs/heads/.invalid\n")
	seedFullStaging(t, env, "2525")
	res := mustAdopt(t, env, newAdoptTracker(), LaneSDD)
	if res.Status != StatusAdopted || len(res.Warnings) == 0 || !strings.Contains(res.Warnings[0], "HEAD") {
		t.Fatalf("resultado = %+v", res)
	}
}

func TestAdoptStagingIntactOnFailure(t *testing.T) {
	steps := map[string]func(env *Env, lane Lane){
		"copy": func(env *Env, _ Lane) {
			env.FS.CopyTree = func(src, dst string) error { return os.ErrPermission }
		},
		"verify": func(env *Env, _ Lane) {
			env.FS.VerifyTree = func(src, dst string) error { return errTreeMismatch }
		},
		"rename": func(env *Env, lane Lane) {
			real := fsSeams{}.withDefaults().Rename
			env.FS.Rename = func(oldPath, newPath string) error {
				if dest, _ := env.contextPaths(lane, adoptChange); newPath == dest {
					return os.ErrPermission
				}
				return real(oldPath, newPath)
			}
		},
		"link": func(env *Env, _ Lane) {
			env.FS.LinkWorkItem = func(integrations.Folder, string, integrations.Link, time.Time) (bool, error) {
				return false, os.ErrPermission
			}
		},
		"exclude": func(env *Env, _ Lane) {
			env.FS.EnsureIgnored = func(string) error { return os.ErrPermission }
		},
		"workitem_json": func(env *Env, lane Lane) {
			dest, _ := env.contextPaths(lane, adoptChange)
			env.FS.WriteFile = failWriteAt(filepath.Join(dest, "work-item.json"))
		},
	}
	for step, inject := range steps {
		for _, lane := range []branch.Lane{branch.LaneSDD, branch.LaneODD} {
			if lane == branch.LaneODD && step == "link" {
				continue // el carril ODD no vincula
			}
			t.Run(step+"_"+string(lane), func(t *testing.T) {
				env := preparedEnv(t, lane)
				seedFullStaging(t, env, "2525")
				stagingBefore := snapshotTree(t, env.StagingDir(adoptChange))
				inject(&env, Lane(lane))

				r := adoptRefusal(t, env, newAdoptTracker(), Lane(lane))

				if r.Code != CodeWriteFailed || r.Fields["step"] != step {
					t.Fatalf("rechazo = %+v", r)
				}
				if !sameSnapshot(stagingBefore, snapshotTree(t, env.StagingDir(adoptChange))) {
					t.Fatal("el staging permanece intacto")
				}
				dest := destDir(env, Lane(lane))
				if step == "workitem_json" {
					if !exists(dest) {
						t.Error("tras el paso 5 el context/ queda (ya excluido) para el reintento")
					}
				} else if exists(dest) {
					t.Errorf("tras un fallo del paso %s no debe quedar context/ en el destino", step)
				}
				if _, tmpBase := env.contextPaths(Lane(lane), adoptChange); exists(tmpBase) {
					t.Errorf("no debe quedar el temporal %s", tmpBase)
				}

				// Un --adopt posterior, sin el fallo, completa la adopción.
				env.FS = fsSeams{}
				tr := newAdoptTracker()
				res := mustAdopt(t, env, tr, Lane(lane))
				if res.Status != StatusAdopted || tr.writes != 1 || exists(env.StagingDir(adoptChange)) {
					t.Fatalf("el reintento no completó: %+v", res)
				}
				if doc := readDest(t, env, Lane(lane)); doc.Backlink != "done" || doc.Lane != string(lane) {
					t.Fatalf("work-item.json = %+v", doc)
				}
			})
		}
	}
}

func TestAdoptStagingRetireFailureWarns(t *testing.T) {
	env := preparedEnv(t, branch.LaneSDD)
	seedFullStaging(t, env, "2525")
	staging := env.StagingDir(adoptChange)
	env.FS.Rename = failRenameFrom(staging)
	tr := newAdoptTracker()

	res := mustAdopt(t, env, tr, LaneSDD)

	if res.Status != StatusAdopted || res.Backlink != BacklinkAdded {
		t.Fatalf("la adopción continúa al paso 7: %+v", res)
	}
	joined := strings.Join(res.Warnings, "\n")
	if !strings.Contains(joined, "staging_not_retired") || !strings.Contains(joined, "axiom change start --adopt --change "+adoptChange) {
		t.Fatalf("avisos = %v", res.Warnings)
	}
	if !exists(staging) {
		t.Fatal("el staging sigue ahí tras el fallo")
	}

	// Un --adopt posterior retira el staging sin más peticiones: el enlace ya estaba hecho.
	env.FS = fsSeams{}
	again := mustAdopt(t, env, tr, LaneSDD)
	if again.Status != StatusAdopted || exists(staging) || tr.calls != 1 {
		t.Fatalf("segundo intento: %+v, llamadas %d", again, tr.calls)
	}
}

// El contexto se copia y se verifica siempre: el staging nunca se renombra hacia el destino, así que el
// resultado es el mismo en uno o dos volúmenes (D-15).
func TestAdoptCopyAcrossVolumesEquivalent(t *testing.T) {
	env := preparedEnv(t, branch.LaneSDD)
	seedFullStaging(t, env, "2525")
	staging := env.StagingDir(adoptChange)
	stagingBefore := snapshotTree(t, staging)
	var copies, verifies int
	var renamedFromStaging []string
	real := fsSeams{}.withDefaults()
	env.FS.CopyTree = func(src, dst string) error { copies++; return real.CopyTree(src, dst) }
	env.FS.VerifyTree = func(src, dst string) error { verifies++; return real.VerifyTree(src, dst) }
	env.FS.Rename = func(oldPath, newPath string) error {
		if strings.HasPrefix(oldPath, filepath.Join(staging, "context")) {
			renamedFromStaging = append(renamedFromStaging, oldPath)
		}
		return os.Rename(oldPath, newPath)
	}

	mustAdopt(t, env, newAdoptTracker(), LaneSDD)

	if copies != 1 || verifies != 1 || len(renamedFromStaging) != 0 {
		t.Fatalf("copias %d, verificaciones %d, renombrados desde context/ del staging %v", copies, verifies, renamedFromStaging)
	}
	dest := destDir(env, LaneSDD)
	for rel, want := range stagingBefore {
		if rel == "context/work-item.json" {
			continue // el destino lleva además el carril y el enlace
		}
		if rel2 := strings.TrimPrefix(rel, "context/"); snapshotTree(t, dest)[rel2] != want {
			t.Errorf("%s difiere entre el staging y el destino", rel)
		}
	}
}

func TestAdoptMultirepoSpecsInPlace(t *testing.T) {
	// Repositorio de specs en el sitio: basta con registro y carpeta, sea cual sea el HEAD.
	env := newRepo(t)
	setHead(t, env, "main")
	seedSpecsInPlaceRecord(t, env, adoptChange)
	if err := os.MkdirAll(env.changeRoot(adoptChange), 0o755); err != nil {
		t.Fatal(err)
	}
	seedFullStaging(t, env, "2525")

	res := mustAdopt(t, env, newAdoptTracker(), LaneSDD)

	if res.Status != StatusAdopted || !exists(filepath.Join(destDir(env, LaneSDD), "work-item.md")) {
		t.Fatalf("resultado = %+v", res)
	}
	// Sin carpeta sigue sin estar preparado.
	if err := os.RemoveAll(env.changeRoot(adoptChange)); err != nil {
		t.Fatal(err)
	}
	seedFullStaging(t, env, "2525")
	if r := adoptRefusal(t, env, newAdoptTracker(), LaneSDD); r.Fields["reason"] != "folder_missing" {
		t.Fatalf("rechazo = %+v", r)
	}
}

// El vínculo se escribe bajo el cerrojo de integraciones y conserva lo que otra escritura (una entrega)
// haya dejado en integrations.yaml, también cuando ambas ocurren a la vez.
func TestAdoptConcurrentWithDelivery(t *testing.T) {
	env := preparedEnv(t, branch.LaneSDD)
	seedFullStaging(t, env, "2525")
	folder := integrations.Folder{WorkspaceRoot: env.WorkspaceRoot, Root: env.changeRoot(adoptChange)}
	if _, err := integrations.LinkWorkItem(folder, "azure-devops", integrations.Link{Kind: "work_item", ID: "9001"}, fixedNow); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := Adopt(context.Background(), env, newAdoptTracker(), adoptChange, LaneSDD)
		errs <- err
	}()
	go func() {
		defer wg.Done()
		_, err := integrations.LinkWorkItem(folder, "azure-devops", integrations.Link{Kind: "work_item", ID: "9002"}, fixedNow)
		errs <- err
	}()
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("escritura concurrente: %v", err)
		}
	}
	links, err := integrations.WorkItemLinks(folder)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for _, l := range links {
		seen[l.ID]++
	}
	if seen["2525"] != 1 || seen["9001"] < 1 || seen["9002"] < 1 {
		t.Fatalf("vínculos tras la escritura concurrente: %v (%+v)", seen, links)
	}
}

// La fase 2 y el descarte nunca piden el cerrojo del work item (D-12, D-24): con él retenido por una
// fase 1, ambos terminan sin esperar.
func TestLockOrderPhase2NeverTakesItemLock(t *testing.T) {
	env := preparedEnv(t, branch.LaneSDD)
	seedFullStaging(t, env, "2525")
	unlock, rf := env.lockItem("azure-devops", "2525")
	if rf != nil {
		t.Fatalf("lockItem: %v", rf)
	}
	defer unlock()

	if res := mustAdopt(t, env, newAdoptTracker(), LaneSDD); res.Status != StatusAdopted {
		t.Fatalf("Adopt con el cerrojo del work item retenido: %+v", res)
	}
	if res, err := Discard(env, adoptChange); err != nil || res.Status != StatusNothingToDiscard {
		t.Fatalf("Discard con el cerrojo del work item retenido: %+v, %v", res, err)
	}
}

// Con git real: tras cada fallo inyectado y tras el éxito, git status no lista nada bajo context/.
func TestAdoptContextNeverUntracked(t *testing.T) {
	injectors := map[string]func(env *Env){
		"copy":   func(env *Env) { env.FS.CopyTree = func(src, dst string) error { return os.ErrPermission } },
		"verify": func(env *Env) { env.FS.VerifyTree = func(src, dst string) error { return errTreeMismatch } },
		"link": func(env *Env) {
			env.FS.LinkWorkItem = func(integrations.Folder, string, integrations.Link, time.Time) (bool, error) {
				return false, os.ErrPermission
			}
		},
		"exclude": func(env *Env) { env.FS.EnsureIgnored = func(string) error { return os.ErrPermission } },
		"ok":      func(env *Env) {},
	}
	for name, inject := range injectors {
		t.Run(name, func(t *testing.T) {
			env := newRepo(t)
			if err := os.RemoveAll(env.CommonDir); err != nil {
				t.Fatal(err)
			}
			if out, err := gitInTemp(t, env.WorkspaceRoot, "init", "-q"); err != nil {
				t.Skipf("git no disponible: %v\n%s", err, out)
			}
			if out, err := gitInTemp(t, env.WorkspaceRoot, "symbolic-ref", "HEAD", "refs/heads/inc/"+adoptChange); err != nil {
				t.Fatalf("symbolic-ref: %v\n%s", err, out)
			}
			seedRecord(t, env, branch.LaneSDD, adoptChange, "")
			writeFile(t, filepath.Join(env.changeRoot(adoptChange), "proposal.md"), "# propuesta\n")
			if out, err := gitInTemp(t, env.WorkspaceRoot, "add", "openspec/changes/"+adoptChange+"/proposal.md"); err != nil {
				t.Fatalf("git add: %v\n%s", err, out)
			}
			seedFullStaging(t, env, "2525")
			inject(&env)

			_, adoptErr := Adopt(context.Background(), env, newAdoptTracker(), adoptChange, LaneSDD)
			if (name == "ok") != (adoptErr == nil) {
				t.Fatalf("error de Adopt = %v", adoptErr)
			}
			status, err := gitInTemp(t, env.WorkspaceRoot, "status", "--porcelain", "--untracked-files=all")
			if err != nil {
				t.Fatalf("git status: %v\n%s", err, status)
			}
			if strings.Contains(status, "context") {
				t.Fatalf("git ve context/:\n%s", status)
			}
		})
	}
}
