package changestart

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

func TestAdoptTagAddedOnce(t *testing.T) {
	env := preparedEnv(t, branch.LaneSDD)
	seedFullStaging(t, env, "2525")
	tr := newAdoptTracker("a", "b")

	mustAdopt(t, env, tr, LaneSDD)
	mustAdopt(t, env, tr, LaneSDD)

	if want := []string{"a", "b", "axiom:" + adoptChange}; strings.Join(tr.tags, ";") != strings.Join(want, ";") {
		t.Fatalf("etiquetas = %v, quiero %v", tr.tags, want)
	}
	if tr.writes != 1 {
		t.Fatalf("una sola escritura efectiva, hubo %d", tr.writes)
	}
}

func TestAdoptTrackerDownWarnsExitZero(t *testing.T) {
	env := preparedEnv(t, branch.LaneSDD)
	seedFullStaging(t, env, "2525")
	tr := newAdoptTracker()
	tr.linkErr = os.ErrDeadlineExceeded

	res := mustAdopt(t, env, tr, LaneSDD)

	if res.Status != StatusAdopted || res.Backlink != BacklinkPending {
		t.Fatalf("resultado = %+v", res)
	}
	if joined := strings.Join(res.Warnings, "\n"); !strings.Contains(joined, "axiom change start --adopt --change "+adoptChange) {
		t.Fatalf("el aviso no nombra el reintento: %v", res.Warnings)
	}
	if doc := readDest(t, env, LaneSDD); doc.Backlink != "pending" {
		t.Fatalf("backlink = %q", doc.Backlink)
	}
	if !exists(filepath.Join(destDir(env, LaneSDD), "work-item.md")) || res.Link != LinkCreated {
		t.Fatal("el contexto y el vínculo quedan aplicados")
	}
	if exists(env.StagingDir(adoptChange)) {
		t.Fatal("el staging ya se retiró antes del enlace")
	}
	// ODD avisa con --lane odd.
	odd := preparedEnv(t, branch.LaneODD)
	seedFullStaging(t, odd, "2525")
	tr = newAdoptTracker()
	tr.linkErr = os.ErrDeadlineExceeded
	oddRes := mustAdopt(t, odd, tr, LaneODD)
	if joined := strings.Join(oddRes.Warnings, "\n"); !strings.Contains(joined, "--adopt --change "+adoptChange+" --lane odd") {
		t.Fatalf("aviso ODD: %v", oddRes.Warnings)
	}
}

func TestAdoptRetryCompletesPendingBacklink(t *testing.T) {
	env := preparedEnv(t, branch.LaneSDD)
	seedFullStaging(t, env, "2525")
	tr := newAdoptTracker()
	tr.linkErr = os.ErrDeadlineExceeded
	mustAdopt(t, env, tr, LaneSDD)
	tr.linkErr = nil

	res := mustAdopt(t, env, tr, LaneSDD)

	if res.Status != StatusAdopted || res.Backlink != BacklinkAdded || res.Link != LinkPresent && res.Link != LinkNotApplicable {
		t.Fatalf("resultado = %+v", res)
	}
	if doc := readDest(t, env, LaneSDD); doc.Backlink != "done" {
		t.Fatalf("backlink = %q", doc.Backlink)
	}
	links, _ := integrations.WorkItemLinks(integrations.Folder{WorkspaceRoot: env.WorkspaceRoot, Root: env.changeRoot(adoptChange)})
	if len(links) != 1 || tr.writes != 1 {
		t.Fatalf("sin duplicados: %d vínculos, %d etiquetas", len(links), tr.writes)
	}
	// Una vez completo, un tercer intento es already_adopted.
	if third := mustAdopt(t, env, tr, LaneSDD); third.Status != StatusAlreadyAdopted {
		t.Fatalf("tercer intento = %+v", third)
	}
}

func TestAdoptOtherTagAppearedBetweenPhases(t *testing.T) {
	env := preparedEnv(t, branch.LaneSDD)
	seedFullStaging(t, env, "2525")
	tr := newAdoptTracker("axiom:otro")

	res := mustAdopt(t, env, tr, LaneSDD)

	if res.Status != StatusAdopted || res.Backlink != BacklinkConflict || tr.writes != 0 {
		t.Fatalf("resultado = %+v, escrituras %d", res, tr.writes)
	}
	if joined := strings.Join(res.Warnings, "\n"); !strings.Contains(joined, "axiom:otro") {
		t.Fatalf("el aviso debe nombrar el conflicto: %v", res.Warnings)
	}
	if doc := readDest(t, env, LaneSDD); doc.Backlink != "conflict" {
		t.Fatalf("backlink = %q", doc.Backlink)
	}
	// Una etiqueta de nombre inválido no es de Axiom: no bloquea y nunca se nombra.
	other := preparedEnv(t, branch.LaneSDD)
	seedFullStaging(t, other, "2525")
	tr = newAdoptTracker("axiom:../x", "axiom:")
	if res := mustAdopt(t, other, tr, LaneSDD); res.Backlink != BacklinkAdded || len(res.Warnings) != 0 {
		t.Fatalf("etiquetas ajenas: %+v", res)
	}
}

func TestAdoptArchivedTagToleratedInPhase2(t *testing.T) {
	env := preparedEnv(t, branch.LaneSDD)
	seedFullStaging(t, env, "2525")
	if err := os.MkdirAll(filepath.Join(env.PlanningHome, "changes", "archive", "2026-10-01-viejo"), 0o755); err != nil {
		t.Fatal(err)
	}
	tr := newAdoptTracker("axiom:viejo")

	res := mustAdopt(t, env, tr, LaneSDD)

	if res.Backlink != BacklinkAdded || len(res.Warnings) != 0 {
		t.Fatalf("resultado = %+v", res)
	}
	if want := "axiom:viejo;axiom:" + adoptChange; strings.Join(tr.tags, ";") != want {
		t.Fatalf("etiquetas = %v, quiero %s", tr.tags, want)
	}
}

// Una escritura protegida por revisión que pierde la carrera no sobrescribe lo ajeno: el adaptador la
// reintenta y, si persiste, el verbo avisa y deja las etiquetas como estaban.
func TestAdoptRevRaceKeepsTags(t *testing.T) {
	env := preparedEnv(t, branch.LaneSDD)
	seedFullStaging(t, env, "2525")
	tr := newAdoptTracker("a", "b")
	tr.linkErr = integrations.ErrItemConflict

	res, err := Adopt(context.Background(), env, tr, adoptChange, LaneSDD)

	if err != nil || res.Backlink != BacklinkPending {
		t.Fatalf("resultado = %+v, error %v", res, err)
	}
	if strings.Join(tr.tags, ";") != "a;b" || tr.writes != 0 {
		t.Fatalf("las etiquetas no deben cambiar: %v", tr.tags)
	}
	if len(res.Warnings) == 0 {
		t.Fatal("debe avisar")
	}
	tr.linkErr = nil
	if again := mustAdopt(t, env, tr, LaneSDD); again.Backlink != BacklinkAdded || strings.Join(tr.tags, ";") != "a;b;axiom:"+adoptChange {
		t.Fatalf("reintento = %+v, etiquetas %v", again, tr.tags)
	}
}
