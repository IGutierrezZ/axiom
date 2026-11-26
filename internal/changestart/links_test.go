package changestart

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

const linkedName = "US-2525-alta-de-usuarios"

func scanTags(t *testing.T, env Env, name string, tags ...string) (linkScan, *Refusal) {
	t.Helper()
	item := integrations.ExternalItem{Tags: tags}
	return newLocalState(env).scanLinks(item, "azure-devops", "2525", name)
}

// linkedWorktree crea un worktree con su carpeta de cambio SDD y su integrations.yaml.
func linkedWorktree(t *testing.T, change, content string) string {
	t.Helper()
	wt := t.TempDir()
	writeFile(t, filepath.Join(wt, "openspec", "changes", change, "integrations.yaml"), content)
	return wt
}

func TestStartLinked(t *testing.T) {
	t.Run("tag_other_open", func(t *testing.T) {
		env := newRepo(t)
		seedRecord(t, env, branch.LaneSDD, "otro-cambio", t.TempDir())
		_, rf := scanTags(t, env, linkedName, "x", "axiom:otro-cambio")
		if rf == nil || rf.Code != CodeWorkItemLinked || rf.Fields["tag"] != "axiom:otro-cambio" || rf.Fields["local"] != "true" {
			t.Fatalf("rechazo = %+v", rf)
		}
	})
	t.Run("local_link_no_tag", func(t *testing.T) {
		env := newRepo(t)
		wt := linkedWorktree(t, "otro", sddLinkYAML("otro"))
		seedRecord(t, env, branch.LaneSDD, "otro", wt)
		_, rf := scanTags(t, env, linkedName)
		if rf == nil || rf.Code != CodeWorkItemLinked || rf.Fields["change"] != "otro" || rf.Fields["tag"] != "" {
			t.Fatalf("rechazo = %+v", rf)
		}
	})
	t.Run("odd_local_adoption", func(t *testing.T) {
		env := newRepo(t)
		wt := t.TempDir()
		writeFile(t, filepath.Join(wt, "odd", "context", "otro", "work-item.json"), contextJSON("otro", "azure-devops", "2525", "done"))
		seedRecord(t, env, branch.LaneODD, "otro", wt)
		_, rf := scanTags(t, env, linkedName)
		if rf == nil || rf.Code != CodeWorkItemLinked || rf.Fields["change"] != "otro" {
			t.Fatalf("rechazo = %+v", rf)
		}
	})
	t.Run("tag_archived_accepted", func(t *testing.T) {
		env := newRepo(t)
		writeFile(t, filepath.Join(env.PlanningHome, "changes", "archive", "2026-01-01-viejo", "tasks.md"), "x")
		seedRecord(t, env, branch.LaneSDD, "viejo", t.TempDir()) // el archivado manda aunque el registro siga
		if _, rf := scanTags(t, env, linkedName, "axiom:viejo"); rf != nil {
			t.Fatalf("un cambio archivado cuenta como cerrado: %+v", rf)
		}
	})
	t.Run("tag_archived_on_main_worktree", func(t *testing.T) {
		main := t.TempDir()
		writeFile(t, filepath.Join(main, ".git", "HEAD"), "ref: refs/heads/main\n")
		writeFile(t, filepath.Join(main, "openspec", "changes", "archive", "2026-02-02-viejo", "tasks.md"), "x")
		linked := t.TempDir()
		gitDir := filepath.Join(main, ".git", "worktrees", "wt")
		writeFile(t, filepath.Join(linked, ".git"), "gitdir: "+gitDir+"\n")
		writeFile(t, filepath.Join(gitDir, "commondir"), "../..\n")
		env := Env{
			WorkspaceRoot: linked, PlanningHome: filepath.Join(linked, "openspec"), GitRoot: linked,
			CommonDir: filepath.Join(main, ".git"),
		}
		if _, rf := scanTags(t, env, linkedName, "axiom:viejo"); rf != nil {
			t.Fatalf("archivado en el worktree principal: %+v", rf)
		}
	})
	t.Run("odd_finished_accepted", func(t *testing.T) {
		env := newRepo(t)
		writeFile(t, filepath.Join(env.WorkspaceRoot, "odd", "tasks", "hecho.md"), "# hecho")
		if _, rf := scanTags(t, env, linkedName, "axiom:hecho"); rf != nil {
			t.Fatalf("ODD terminado sin registro: %+v", rf)
		}
	})
	t.Run("odd_registry_open", func(t *testing.T) {
		env := newRepo(t)
		writeFile(t, filepath.Join(env.WorkspaceRoot, "odd", "tasks", "abierto.md"), "# abierto")
		seedRecord(t, env, branch.LaneODD, "abierto", t.TempDir())
		_, rf := scanTags(t, env, linkedName, "axiom:abierto")
		if rf == nil || rf.Code != CodeWorkItemLinked {
			t.Fatalf("con registro el cambio ODD sigue abierto: %+v", rf)
		}
	})
	t.Run("tag_unknown_locally_refused", func(t *testing.T) {
		env := newRepo(t)
		_, rf := scanTags(t, env, linkedName, "axiom:remoto")
		if rf == nil || rf.Code != CodeWorkItemLinked || rf.Fields["local"] != "false" || rf.Fields["tag"] != "axiom:remoto" {
			t.Fatalf("sin rastro local cuenta como abierto: %+v", rf)
		}
	})
	t.Run("same_name_tag", func(t *testing.T) {
		env := newRepo(t)
		if _, rf := scanTags(t, env, linkedName, "axiom:"+linkedName); rf != nil {
			t.Fatalf("la misma etiqueta no es conflicto: %+v", rf)
		}
	})
	t.Run("same_name_tag_case_insensitive", func(t *testing.T) {
		env := newRepo(t)
		if _, rf := scanTags(t, env, linkedName, "AXIOM: "+strings.ToUpper(linkedName)+" "); rf != nil {
			t.Fatalf("la comparación no distingue mayúsculas: %+v", rf)
		}
	})
	t.Run("tag_invalid_component_ignored", func(t *testing.T) {
		env := newRepo(t)
		if _, rf := scanTags(t, env, linkedName, "axiom:../x", "axiom:a b", "axiom:"); rf != nil {
			t.Fatalf("una etiqueta ajena no bloquea: %+v", rf)
		}
	})
	t.Run("tag_archive_reserved_ignored", func(t *testing.T) {
		env := newRepo(t)
		if _, rf := scanTags(t, env, linkedName, "axiom:archive"); rf != nil {
			t.Fatalf("archive es reservado y se ignora: %+v", rf)
		}
	})
	t.Run("unreadable_registry_warns", func(t *testing.T) {
		env := newRepo(t)
		writeFile(t, filepath.Join(env.CommonDir, "axiom", "branch", "v1", "sdd", "roto.json"), "{no es json")
		scan, rf := scanTags(t, env, linkedName)
		if rf != nil {
			t.Fatalf("un registro ilegible no bloquea: %+v", rf)
		}
		if len(scan.Warnings) != 1 || !strings.Contains(scan.Warnings[0], "registro local") {
			t.Fatalf("avisos = %v", scan.Warnings)
		}
	})
	t.Run("self_adoption_sdd", func(t *testing.T) {
		env := newRepo(t)
		wt := linkedWorktree(t, linkedName, sddLinkYAML(linkedName))
		seedRecord(t, env, branch.LaneSDD, linkedName, wt)
		scan, rf := scanTags(t, env, linkedName)
		if rf != nil || scan.Self == nil || scan.Self.Lane != LaneSDD {
			t.Fatalf("adopción propia no detectada: %+v %+v", scan, rf)
		}
	})
}

func TestStartLinkedTagWinsOverLocal(t *testing.T) {
	env := newRepo(t)
	wt := linkedWorktree(t, "local", sddLinkYAML("local"))
	seedRecord(t, env, branch.LaneSDD, "local", wt)
	seedRecord(t, env, branch.LaneSDD, "etiquetado", t.TempDir())
	_, rf := scanTags(t, env, linkedName, "axiom:etiquetado")
	if rf == nil || rf.Fields["tag"] != "axiom:etiquetado" || rf.Fields["change"] != "etiquetado" {
		t.Fatalf("manda el WI: %+v", rf)
	}
	if _, err := os.Stat(wt); err != nil {
		t.Fatal(err)
	}
}
