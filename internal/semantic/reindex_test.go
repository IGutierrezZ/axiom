package semantic

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeFakeCodeGraph writes a stand-in `codegraph` script into binDir and
// returns its path and the file it records to. Each invocation writes its
// working directory and arguments (one per line) to that file and exits 0, so
// a test can prove whether, and where, a reindex ran it.
func writeFakeCodeGraph(t *testing.T, binDir string) (binPath, recordPath string) {
	t.Helper()
	recordPath = filepath.Join(t.TempDir(), "codegraph-invocation.txt")

	var name, script string
	if runtime.GOOS == "windows" {
		name = "codegraph.cmd"
		script = "@echo off\r\n" +
			">\"" + recordPath + "\" echo %cd%\r\n" +
			">>\"" + recordPath + "\" echo %*\r\n" +
			"exit /b 0\r\n"
	} else {
		name = "codegraph"
		script = "#!/bin/sh\n" +
			"{ pwd; echo \"$*\"; } > '" + recordPath + "'\n" +
			"exit 0\n"
	}
	binPath = filepath.Join(binDir, name)
	if err := os.WriteFile(binPath, []byte(script), 0o755); err != nil {
		t.Fatalf("escribir el codegraph de prueba: %v", err)
	}
	return binPath, recordPath
}

// Con un `codegraph` en el PATH y un Detector que no lo encuentra, la
// reindexación obedece al Detector: no ejecuta el binario del PATH y responde
// con el motor nativo.
func TestServiceReindexCodeGraphIgnoresPathWhenDetectorMisses(t *testing.T) {
	binDir := t.TempDir()
	_, recordPath := writeFakeCodeGraph(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	detector := NewDetectorWithCustomHome(t.TempDir(), func(string) (string, error) {
		return "", os.ErrNotExist
	})
	srv := NewService(t.TempDir(), detector, NewEngine())

	res, err := srv.ReindexCodeGraph(context.Background())
	if err != nil {
		t.Fatalf("error inesperado en ReindexCodeGraph: %v", err)
	}
	if res.Connector != string(ConnectorNativeAST) {
		t.Errorf("connector = %q, se esperaba %q", res.Connector, ConnectorNativeAST)
	}
	if _, err := os.Stat(recordPath); err == nil {
		t.Errorf("se ejecutó el codegraph del PATH aunque el Detector no lo encuentra")
	}
}

// codeGraphServiceFor devuelve un Service sobre un workspace nuevo cuyo
// Detector resuelve `codegraph` al binario de prueba. Con indexed crea el
// directorio `.codegraph`, como lo dejaría una inicialización previa.
func codeGraphServiceFor(t *testing.T, indexed bool) (srv *Service, workspaceRoot, recordPath string) {
	t.Helper()
	binPath, recordPath := writeFakeCodeGraph(t, t.TempDir())
	t.Setenv("PATH", t.TempDir())

	detector := NewDetectorWithCustomHome(t.TempDir(), func(file string) (string, error) {
		if file != "codegraph" {
			return "", os.ErrNotExist
		}
		return binPath, nil
	})
	workspaceRoot = t.TempDir()
	if indexed {
		if err := os.MkdirAll(filepath.Join(workspaceRoot, ".codegraph"), 0o755); err != nil {
			t.Fatalf("crear .codegraph: %v", err)
		}
	}
	return NewService(workspaceRoot, detector, NewEngine()), workspaceRoot, recordPath
}

// Sin `.codegraph`, la reindexación inicializa el índice con `init --yes <raíz>`
// en lugar de lanzar `index`, que fallaría.
func TestServiceReindexCodeGraphInitializesMissingIndex(t *testing.T) {
	srv, workspaceRoot, recordPath := codeGraphServiceFor(t, false)

	res, err := srv.ReindexCodeGraph(context.Background())
	if err != nil {
		t.Fatalf("error inesperado en ReindexCodeGraph: %v", err)
	}
	if !res.Success || res.Connector != string(ConnectorCodeGraph) || res.Operation != "init" {
		t.Errorf("resultado = %+v, se esperaba éxito con operación init", res)
	}
	if !strings.Contains(res.Message, "Inicialización") {
		t.Errorf("mensaje = %q, se esperaba que nombrara la inicialización", res.Message)
	}

	raw, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatalf("no se ejecutó el codegraph de prueba: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 {
		t.Fatalf("registro de codegraph inesperado: %q", raw)
	}
	args := strings.Fields(lines[1])
	if len(args) != 3 || args[0] != "init" || args[1] != "--yes" {
		t.Fatalf("argumentos de codegraph = %q, se esperaba \"init --yes <raíz>\"", lines[1])
	}
	gotArg, err := os.Stat(args[2])
	if err != nil {
		t.Fatalf("ruta pasada a init %q ilegible: %v", args[2], err)
	}
	wantDir, _ := os.Stat(workspaceRoot)
	if !os.SameFile(gotArg, wantDir) {
		t.Errorf("init recibió %s, se esperaba el workspace %s", args[2], workspaceRoot)
	}
}

// El estado informa codegraph_indexed según exista `.codegraph` en el workspace.
func TestServiceStatusReportsCodeGraphIndexed(t *testing.T) {
	for _, indexed := range []bool{false, true} {
		srv, _, _ := codeGraphServiceFor(t, indexed)
		status, err := srv.GetStatus(context.Background())
		if err != nil {
			t.Fatalf("GetStatus: %v", err)
		}
		if status.CodeGraphIndexed != indexed {
			t.Errorf("CodeGraphIndexed = %v, se esperaba %v", status.CodeGraphIndexed, indexed)
		}
	}
}

// Con un Detector que resuelve `codegraph` a una ruta fuera del PATH y un
// índice existente, la reindexación ejecuta esa ruta con `index` en la raíz del
// workspace.
func TestServiceReindexCodeGraphRunsDetectorBinary(t *testing.T) {
	srv, workspaceRoot, recordPath := codeGraphServiceFor(t, true)

	res, err := srv.ReindexCodeGraph(context.Background())
	if err != nil {
		t.Fatalf("error inesperado en ReindexCodeGraph: %v", err)
	}
	if !res.Success || res.Connector != string(ConnectorCodeGraph) || res.Operation != "index" {
		t.Errorf("resultado = %+v, se esperaba éxito con el conector %q y operación index", res, ConnectorCodeGraph)
	}

	raw, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatalf("no se ejecutó el codegraph que resolvió el Detector: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 {
		t.Fatalf("registro de codegraph inesperado: %q", raw)
	}
	if args := strings.TrimSpace(lines[1]); args != "index" {
		t.Errorf("argumentos de codegraph = %q, se esperaba \"index\"", args)
	}
	gotDir, err := os.Stat(strings.TrimSpace(lines[0]))
	if err != nil {
		t.Fatalf("directorio de ejecución %q ilegible: %v", lines[0], err)
	}
	wantDir, err := os.Stat(workspaceRoot)
	if err != nil {
		t.Fatalf("workspace %q ilegible: %v", workspaceRoot, err)
	}
	if !os.SameFile(gotDir, wantDir) {
		t.Errorf("codegraph se ejecutó en %s, se esperaba el workspace %s", lines[0], workspaceRoot)
	}
}
