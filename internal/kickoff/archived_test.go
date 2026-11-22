package kickoff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestArchivedChangeRoot fija la regla de búsqueda de cambios archivados
// (REQ-24.9, escenario 3): nombre exacto (forma heredada) y entrada con prefijo
// de fecha, de la que gana la más reciente; un sufijo parcial no coincide.
func TestArchivedChangeRoot(t *testing.T) {
	tests := []struct {
		name    string
		entries []string // directorios bajo openspec/changes/archive
		files   []string // ficheros (no directorios) bajo archive
		change  string
		want    string // entrada esperada; "" si no se resuelve
	}{
		{name: "nombre exacto", entries: []string{"inc-1"}, change: "inc-1", want: "inc-1"},
		{name: "entrada con fecha", entries: []string{"2026-10-04-inc-1"}, change: "inc-1", want: "2026-10-04-inc-1"},
		{name: "gana la fecha más reciente", entries: []string{"2026-01-02-inc-1", "2026-10-04-inc-1", "2026-03-09-inc-1"}, change: "inc-1", want: "2026-10-04-inc-1"},
		{name: "el nombre exacto gana a las entradas con fecha", entries: []string{"inc-1", "2026-10-04-inc-1"}, change: "inc-1", want: "inc-1"},
		{name: "sufijo parcial no coincide", entries: []string{"2026-10-04-inc-1-extra", "2026-10-04-otro-inc-1"}, change: "inc-1"},
		{name: "prefijo de fecha incompleto no coincide", entries: []string{"2026-10-inc-1", "26-10-04-inc-1"}, change: "inc-1"},
		{name: "un fichero no es un cambio archivado", files: []string{"2026-10-04-inc-1"}, change: "inc-1"},
		{name: "sin archivo", change: "inc-1"},
		{name: "nombre con separador no se resuelve", entries: []string{"inc-1"}, change: "../inc-1"},
		{name: "nombre vacío no se resuelve", entries: []string{"inc-1"}, change: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			workspaceRoot := t.TempDir()
			archiveDir := filepath.Join(workspaceRoot, "openspec", "changes", "archive")
			for _, entry := range tc.entries {
				if err := os.MkdirAll(filepath.Join(archiveDir, entry), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			for _, file := range tc.files {
				if err := os.MkdirAll(archiveDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(archiveDir, file), []byte("x"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, ok := ArchivedChangeRoot(workspaceRoot, tc.change)
			if tc.want == "" {
				if ok || got != "" {
					t.Fatalf("ArchivedChangeRoot() = (%q, %v), se esperaba (\"\", false)", got, ok)
				}
				return
			}
			if want := filepath.Join(archiveDir, tc.want); !ok || got != want {
				t.Fatalf("ArchivedChangeRoot() = (%q, %v), se esperaba (%q, true)", got, ok, want)
			}
			if err := RefuseArchivedRoot(workspaceRoot, got); err == nil {
				t.Fatalf("RefuseArchivedRoot(%q) = nil: la carpeta archivada debe rechazarse por cambio archivado", got)
			}
		})
	}
}

// TestRefuseArchivedRootRejectsArchivedChange is D-14's core rule: no
// write path this package owns may target a change root already living
// under openspec/changes/archive/.
func TestRefuseArchivedRootRejectsArchivedChange(t *testing.T) {
	workspaceRoot := t.TempDir()
	changeRoot := filepath.Join(workspaceRoot, "openspec", "changes", "archive", "2026-01-01-x")
	if err := RefuseArchivedRoot(workspaceRoot, changeRoot); err == nil {
		t.Fatal("RefuseArchivedRoot() = nil, se esperaba rechazo para una raiz bajo archive/")
	}
}

// TestRefuseArchivedRootAcceptsActiveChange is the mirror case: an active
// (non-archived) change root must not be rejected.
func TestRefuseArchivedRootAcceptsActiveChange(t *testing.T) {
	workspaceRoot := t.TempDir()
	changeRoot := filepath.Join(workspaceRoot, "openspec", "changes", "x")
	if err := RefuseArchivedRoot(workspaceRoot, changeRoot); err != nil {
		t.Fatalf("RefuseArchivedRoot() error = %v, se esperaba nil para un cambio activo", err)
	}
}

// TestRefuseArchivedRootRejectsEscapeAboveWorkspace covers T-7's "raiz de
// cambio" vector: a changeRoot that resolves above workspaceRoot via ".."
// must never be accepted, regardless of whether it happens to land under
// an archive/-looking path or not.
func TestRefuseArchivedRootRejectsEscapeAboveWorkspace(t *testing.T) {
	workspaceRoot := t.TempDir()
	changeRoot := filepath.Join(workspaceRoot, "..", "otro")
	if err := RefuseArchivedRoot(workspaceRoot, changeRoot); err == nil {
		t.Fatal("RefuseArchivedRoot() = nil, se esperaba rechazo por contencion (.. escapa del workspace)")
	}
}

// TestRefuseArchivedRootRejectsAbsolutePathOutsideWorkspace covers the
// second T-7 containment vector: an absolute path naming a location that
// is not inside workspaceRoot at all.
func TestRefuseArchivedRootRejectsAbsolutePathOutsideWorkspace(t *testing.T) {
	workspaceRoot := t.TempDir()
	unrelatedRoot := t.TempDir()
	changeRoot := filepath.Join(unrelatedRoot, "openspec", "changes", "x")
	if err := RefuseArchivedRoot(workspaceRoot, changeRoot); err == nil {
		t.Fatal("RefuseArchivedRoot() = nil, se esperaba rechazo por contencion (ruta absoluta fuera del workspace)")
	}
}

// MODIFIED REQ-21.18 y REQ-43.28: la guarda hermana autoriza bajo archive/ solo la cola de
// integraciones (y sus temporales) en la raíz exacta de una carpeta archivada.
func TestRefuseArchivedWriteExceptLocalState(t *testing.T) {
	workspaceRoot := t.TempDir()
	changeRoot := filepath.Join(workspaceRoot, "openspec", "changes", "x")
	archiveDir := filepath.Join(workspaceRoot, "openspec", "changes", "archive")
	archived := filepath.Join(archiveDir, "2026-11-19-x")

	tests := []struct {
		name       string
		changeRoot string
		file       string
		wantErr    bool
	}{
		{"carpeta ordinaria delega y acepta cualquier nombre simple", changeRoot, "integrations.yaml", false},
		{"carpeta ordinaria acepta la cola", changeRoot, IntegrationsOutboxFile, false},
		{"raíz archivada con la cola", archived, ".integrations-outbox.yaml", false},
		{"raíz archivada con un temporal de la cola", archived, ".integrations-outbox-123456.yaml", false},
		{"raíz archivada con integrations.yaml", archived, "integrations.yaml", true},
		{"raíz archivada con gates.yaml", archived, "gates.yaml", true},
		{"raíz archivada con .gates.lock", archived, ".gates.lock", true},
		{"raíz archivada con otro fichero con punto", archived, ".otro", true},
		{"raíz archivada con un temporal de integrations.yaml", archived, ".integrations-123.yaml", true},
		{"temporal sin sufijo yaml", archived, ".integrations-outbox-123.tmp", true},
		{"temporal vacío", archived, ".integrations-outbox-.yaml", true},
		{"subcarpeta de la archivada", filepath.Join(archived, "specs"), ".integrations-outbox.yaml", true},
		{"subcarpeta anidada profunda", filepath.Join(archived, "specs", "a"), ".integrations-outbox.yaml", true},
		{"archive sin carpeta de cambio", archiveDir, ".integrations-outbox.yaml", true},
		{"carpeta archivada sin fecha", filepath.Join(archiveDir, "x"), ".integrations-outbox.yaml", true},
		{"carpeta archivada con fecha incompleta", filepath.Join(archiveDir, "2026-11-x"), ".integrations-outbox.yaml", true},
		{"nombre con separador", archived, "sub/.integrations-outbox.yaml", true},
		{"nombre con separador de Windows", archived, `sub\.integrations-outbox.yaml`, true},
		{"nombre con ..", archived, "..", true},
		{"nombre con ../", archived, "../.integrations-outbox.yaml", true},
		{"nombre vacío", archived, "", true},
		{"nombre con separador en carpeta ordinaria", changeRoot, "../integrations.yaml", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := RefuseArchivedWriteExceptLocalState(workspaceRoot, tc.changeRoot, tc.file)
			if (err != nil) != tc.wantErr {
				t.Fatalf("RefuseArchivedWriteExceptLocalState(%q, %q) = %v, wantErr %v", tc.changeRoot, tc.file, err, tc.wantErr)
			}
		})
	}

	t.Run("el rechazo en la archivada lleva el mensaje de REQ-21.18", func(t *testing.T) {
		want := RefuseArchivedRoot(workspaceRoot, archived)
		if want == nil {
			t.Fatal("RefuseArchivedRoot debe rechazar la carpeta archivada")
		}
		for _, file := range []string{"integrations.yaml", "gates.yaml", ".gates.lock"} {
			got := RefuseArchivedWriteExceptLocalState(workspaceRoot, archived, file)
			if got == nil || got.Error() != want.Error() {
				t.Errorf("%s: error = %v, se esperaba el mensaje de REQ-21.18 %q", file, got, want)
			}
		}
		if err := RefuseArchivedWriteExceptLocalState(workspaceRoot, archiveDir, IntegrationsOutboxFile); err == nil || !strings.Contains(err.Error(), "REQ-21.18") {
			t.Errorf("archive/ sin carpeta de cambio debe llevar REQ-21.18: %v", err)
		}
	})

	t.Run("una raíz fuera del workspace se rechaza", func(t *testing.T) {
		if err := RefuseArchivedWriteExceptLocalState(workspaceRoot, t.TempDir(), IntegrationsOutboxFile); err == nil {
			t.Fatal("una carpeta fuera del workspace se rechaza como en RefuseArchivedRoot")
		}
	})

	t.Run("RefuseArchivedRoot queda intacta", func(t *testing.T) {
		if err := RefuseArchivedRoot(workspaceRoot, archived); err == nil {
			t.Fatal("RefuseArchivedRoot sigue rechazando toda carpeta archivada")
		}
		if err := RefuseArchivedRoot(workspaceRoot, changeRoot); err != nil {
			t.Fatalf("RefuseArchivedRoot sigue aceptando la ordinaria: %v", err)
		}
	})

	t.Run("IntegrationsOutboxFile es el único literal", func(t *testing.T) {
		if IntegrationsOutboxFile != ".integrations-outbox.yaml" {
			t.Fatalf("IntegrationsOutboxFile = %q", IntegrationsOutboxFile)
		}
	})
}
