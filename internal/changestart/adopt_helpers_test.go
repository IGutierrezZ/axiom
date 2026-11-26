package changestart

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

const adoptChange = "US-2525-alta-de-usuarios"

// adoptTracker es un Tracker en memoria cuyo enlace de vuelta imita al adaptador: no escribe si la
// etiqueta ya figura, consulta Allow con las etiquetas actuales y cuenta las escrituras efectivas.
type adoptTracker struct {
	*stubTracker
	tags    []string
	linkErr error
	calls   int
	writes  int
	lastTag string
}

func newAdoptTracker(tags ...string) *adoptTracker {
	return &adoptTracker{stubTracker: newStub(), tags: tags}
}

func (a *adoptTracker) LinkBack(_ context.Context, _ string, tag string, allow func([]string) error) (integrations.BackLink, error) {
	a.calls++
	a.lastTag = tag
	if a.linkErr != nil {
		return integrations.BackLink{}, a.linkErr
	}
	for _, t := range a.tags {
		if strings.EqualFold(t, tag) {
			return integrations.BackLink{Outcome: integrations.BackLinkPresent}, nil
		}
	}
	if allow != nil {
		if err := allow(a.tags); err != nil {
			return integrations.BackLink{Outcome: integrations.BackLinkConflict, Conflicting: a.tags}, nil
		}
	}
	a.tags = append(a.tags, tag)
	a.writes++
	return integrations.BackLink{Outcome: integrations.BackLinkAdded}, nil
}

// setHead escribe el HEAD del checkout: la rama dada.
func setHead(t *testing.T, env Env, branchName string) {
	t.Helper()
	writeFile(t, filepath.Join(env.CommonDir, "HEAD"), "ref: refs/heads/"+branchName+"\n")
}

// preparedEnv deja un cambio preparado en el carril: registro con su rama, HEAD en esa rama y, en SDD,
// la carpeta del cambio.
func preparedEnv(t *testing.T, lane branch.Lane) Env {
	t.Helper()
	env := newRepo(t)
	setHead(t, env, "inc/"+adoptChange)
	seedRecord(t, env, lane, adoptChange, "")
	if lane == branch.LaneSDD {
		if err := os.MkdirAll(env.changeRoot(adoptChange), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return env
}

// seedFullStaging deja un staging completo del work item 2525: Markdown, un adjunto y work-item.json.
func seedFullStaging(t *testing.T, env Env, id string) {
	t.Helper()
	dir := filepath.Join(env.StagingDir(adoptChange), "context")
	writeFile(t, filepath.Join(dir, "work-item.md"), "# Alta de usuarios\n")
	writeFile(t, filepath.Join(dir, "attachments", "tasks.md"), "adjunto con nombre de artefacto\n")
	doc := ContextDoc{
		Schema: ContextSchema, Change: adoptChange, Plugin: "azure-devops", ID: id,
		URL:         "https://dev.azure.com/contoso/Fabrikam/_workitems/edit/" + id,
		ExternalRef: "ado:" + id, Profile: string(ProfileIncrement), TypeKey: "User Story",
		StagedAt: "2026-11-26T10:00:00Z", Attachments: Attachments{Downloaded: 1},
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "work-item.json"), string(data)+"\n")
}

func mustAdopt(t *testing.T, env Env, tr Tracker, lane Lane) AdoptResult {
	t.Helper()
	res, err := Adopt(context.Background(), env, tr, adoptChange, lane)
	if err != nil {
		t.Fatalf("Adopt: %v", err)
	}
	return res
}

func adoptRefusal(t *testing.T, env Env, tr Tracker, lane Lane) *Refusal {
	t.Helper()
	_, err := Adopt(context.Background(), env, tr, adoptChange, lane)
	if err == nil {
		t.Fatal("Adopt debía rechazar")
	}
	return refusalOf(t, err)
}

// destDir es el context/ del destino del carril.
func destDir(env Env, lane Lane) string {
	dest, _ := env.contextPaths(lane, adoptChange)
	return dest
}

func readDest(t *testing.T, env Env, lane Lane) ContextDoc {
	t.Helper()
	doc, err := readContextDoc(filepath.Join(destDir(env, lane), "work-item.json"))
	if err != nil {
		t.Fatalf("work-item.json del destino: %v", err)
	}
	return doc
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// snapshotTree devuelve ruta relativa → contenido de los ficheros de root, sin las rutas bajo skip.
func snapshotTree(t *testing.T, root string, skip ...string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		for _, s := range skip {
			if rel == s || strings.HasPrefix(rel, s+"/") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func sameSnapshot(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}

// failOn devuelve una costura de escritura que falla en una ruta exacta y delega en la real en el resto.
func failWriteAt(target string) func(string, []byte, os.FileMode) error {
	real := fsSeams{}.withDefaults().WriteFile
	return func(path string, data []byte, perm os.FileMode) error {
		if path == target {
			return errors.New("fallo simulado de escritura")
		}
		return real(path, data, perm)
	}
}

// failRenameFrom devuelve una costura de renombrado que falla cuando el origen es from.
func failRenameFrom(from string) func(string, string) error {
	return func(oldPath, newPath string) error {
		if oldPath == from {
			return errors.New("fallo simulado de renombrado")
		}
		return os.Rename(oldPath, newPath)
	}
}

// seedSpecsInPlaceRecord siembra el registro de un cambio SDD cuyo repositorio de specs está en el sitio
// (multirepo): solo exige registro y carpeta.
func seedSpecsInPlaceRecord(t *testing.T, env Env, change string) {
	t.Helper()
	repo := branch.RecordRepo{Name: "specs", GitRoot: env.GitRoot, CommonDir: env.CommonDir, Role: branch.RepoSpecsInPlace}
	rec := branch.Record{
		Schema: branch.RegistrySchema, Lane: branch.LaneSDD, Change: change, Mode: branch.ModeInPlace,
		ModeSource: branch.ModeSourceDefault, PlanningRepo: env.GitRoot, Repositories: []branch.RecordRepo{repo},
	}
	if err := registry.New().Save(env.CommonDir, rec); err != nil {
		t.Fatalf("sembrar registro: %v", err)
	}
}
