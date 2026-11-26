package changestart

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscardExisting(t *testing.T) {
	env := newRepo(t)
	seedFullStaging(t, env, "2525")
	seedStaging(t, env, "otro-cambio", "3000") // otro staging: no se toca
	untouched := filepath.Join(env.WorkspaceRoot, "README.md")
	writeFile(t, untouched, "no tocar\n")

	res, err := Discard(env, adoptChange)

	if err != nil || res.Status != StatusDiscarded || res.Change != adoptChange {
		t.Fatalf("resultado = %+v, error %v", res, err)
	}
	if exists(env.StagingDir(adoptChange)) {
		t.Fatal("el staging debe desaparecer")
	}
	if !exists(env.StagingDir("otro-cambio")) || !exists(untouched) {
		t.Fatal("solo se retira el staging del cambio")
	}
	if got := stagings(t, env); len(got) != 1 || got[0] != "otro-cambio" {
		t.Fatalf("stagings = %v", got)
	}
}

func TestDiscardNothingWarns(t *testing.T) {
	env := newRepo(t)
	res, err := Discard(env, adoptChange)
	if err != nil || res.Status != StatusNothingToDiscard || len(res.Warnings) == 0 {
		t.Fatalf("resultado = %+v, error %v", res, err)
	}
	// Un staging sin work-item.json legible también se retira.
	writeFile(t, filepath.Join(env.StagingDir(adoptChange), "context", "work-item.md"), "a medias\n")
	if res, err := Discard(env, adoptChange); err != nil || res.Status != StatusDiscarded || exists(env.StagingDir(adoptChange)) {
		t.Fatalf("staging ilegible: %+v, %v", res, err)
	}
}

func TestDiscardRenameFailureWriteFailed(t *testing.T) {
	env := newRepo(t)
	seedFullStaging(t, env, "2525")
	env.FS.Rename = failRenameFrom(env.StagingDir(adoptChange))

	_, err := Discard(env, adoptChange)

	if r := refusalOf(t, err); r.Code != CodeWriteFailed {
		t.Fatalf("rechazo = %+v", r)
	}
	if !exists(filepath.Join(env.StagingDir(adoptChange), "context", "work-item.json")) {
		t.Fatal("el staging permanece intacto")
	}
}

func TestDiscardDeleteFailureWarns(t *testing.T) {
	env := newRepo(t)
	seedFullStaging(t, env, "2525")
	env.FS.RemoveAll = func(string) error { return errors.New("fallo simulado de borrado") }

	res, err := Discard(env, adoptChange)

	if err != nil || res.Status != StatusDiscarded || len(res.Warnings) == 0 {
		t.Fatalf("resultado = %+v, error %v", res, err)
	}
	if exists(env.StagingDir(adoptChange)) {
		t.Fatal("el staging ya no está en su sitio: se renombró antes de borrar")
	}
}

func TestDiscardSweepsTemps(t *testing.T) {
	env := newRepo(t)
	seedFullStaging(t, env, "2525")
	writeFile(t, filepath.Join(env.tmpDir(adoptChange), "abandonado", "context", "work-item.md"), "temporal\n")
	other := filepath.Join(env.tmpDir("otro-cambio"), "x", "f")
	writeFile(t, other, "de otro cambio\n")

	if res, err := Discard(env, adoptChange); err != nil || res.Status != StatusDiscarded {
		t.Fatalf("resultado = %+v, error %v", res, err)
	}
	if exists(env.tmpDir(adoptChange)) {
		t.Fatal("los temporales del cambio se barren")
	}
	if !exists(other) {
		t.Fatal("los temporales de otro cambio no se tocan")
	}
}

func TestDiscardRejectsInvalidChange(t *testing.T) {
	env := newRepo(t)
	seedFullStaging(t, env, "2525")
	for _, bad := range []string{"../../algo", "a/b", "..", "archive", ""} {
		_, err := Discard(env, bad)
		if r := refusalOf(t, err); r.Code != CodeUsageInvalid {
			t.Fatalf("%q: rechazo = %+v", bad, r)
		}
	}
	if !exists(env.StagingDir(adoptChange)) {
		t.Fatal("no se borra nada")
	}
	if _, err := os.Stat(env.v1()); err != nil {
		t.Fatal(err)
	}
}
