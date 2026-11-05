package dashboard

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// El proceso de `axiom ui` ya no tiene el workspace como directorio de trabajo
// (ServerWorkDir). La sincronización posterior a la actualización debe seguir
// apuntando a la raíz de --path: RunSync entra en ella con un Chdir temporal y
// restaura el directorio anterior.
func TestPostUpdateSync_TargetsThePathRootWhileTheProcessCwdIsElsewhere(t *testing.T) {
	neutral := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(neutral); err != nil {
		t.Fatal(err)
	}

	svc := postUpdateService(t)
	root := svc.getRootPath()

	var cwdDuringSync string
	stubRunAppArgs(t, func(_ []string, _ io.Writer) error {
		cwdDuringSync, _ = os.Getwd()
		return nil
	})

	svc.StartPostUpdateSync()
	st := waitUpdateFinished(t, svc)
	if st.Phase != "done" {
		t.Fatalf("estado = %+v", st)
	}

	same := func(a, b string) bool {
		ea, _ := filepath.EvalSymlinks(a)
		eb, _ := filepath.EvalSymlinks(b)
		return ea == eb
	}
	if !same(cwdDuringSync, root) {
		t.Fatalf("la sincronización corrió en %q, quiero la raíz %q", cwdDuringSync, root)
	}
	if after, _ := os.Getwd(); !same(after, neutral) {
		t.Fatalf("el cwd tras sincronizar es %q, quiero el neutro %q restaurado", after, neutral)
	}
}

func TestRelaunchedChildRunsOutsideTheWorkspace(t *testing.T) {
	root := t.TempDir()
	dir := ServerWorkDir(root)
	if isWithinDir(root, dir) {
		t.Fatalf("el cwd del hijo relanzado (%q) está dentro del workspace %q", dir, root)
	}
	if !reflect.DeepEqual(relaunchArgs(root, 47700)[:3], []string{"ui", "--path", root}) {
		t.Fatal("el workspace debe seguir llegando por --path")
	}
}
