package autoskill

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeInboxRaw(t *testing.T, workspace, name, skillMD, metaJSON string) string {
	t.Helper()
	folder := filepath.Join(workspace, ".axiom", "skills", "inbox", name)
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
		t.Fatal(err)
	}
	if metaJSON != "" {
		if err := os.WriteFile(filepath.Join(folder, "metadata.json"), []byte(metaJSON), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return folder
}

func newGateManager(workspace string) *Manager {
	m := NewManager(workspace, nil, nil, nil)
	m.RegenerateIndex = func(cwd, home string) error { return nil }
	return m
}

func TestApproveRechazaSkillSinFrontmatter(t *testing.T) {
	ws := t.TempDir()
	folder := writeInboxRaw(t, ws, "sin-front", "# Sin frontmatter\n", `{"name":"sin-front","origin":"midudev"}`)

	outcome, err := newGateManager(ws).Approve("sin-front")
	if err == nil || outcome.Promoted {
		t.Fatalf("Approve() debe rechazar: outcome=%+v err=%v", outcome, err)
	}
	var verr *SkillValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("error = %T, want *SkillValidationError", err)
	}
	if !strings.Contains(err.Error(), "frontmatter-start") {
		t.Errorf("el mensaje debe nombrar la regla: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(folder, "SKILL.md")); statErr != nil {
		t.Errorf("el buzón debe quedar intacto: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(ws, "skills", "sin-front")); !os.IsNotExist(statErr) {
		t.Errorf("no debe copiarse nada a skills/ (stat err = %v)", statErr)
	}
}

func TestApproveAceptaMidudevSinTriggerSoloConAviso(t *testing.T) {
	ws := t.TempDir()
	writeInboxRaw(t, ws, "tercero", "---\nname: tercero\ndescription: Descripción sin comillas ni trigger\n---\n\nBody.\n",
		`{"name":"tercero","origin":"midudev"}`)

	outcome, err := newGateManager(ws).Approve("tercero")
	if err != nil || !outcome.Promoted {
		t.Fatalf("Approve() = %+v, %v; los avisos no bloquean en perfil portable", outcome, err)
	}
}

func TestApproveMinadaExigeCanonico(t *testing.T) {
	ws := t.TempDir()
	skill := "---\nname: minada\ndescription: Sin comillas ni trigger\n---\n\nBody.\n"
	writeInboxRaw(t, ws, "minada", skill, `{"name":"minada","origin":"mined"}`)

	_, err := newGateManager(ws).Approve("minada")
	var verr *SkillValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("error = %v, want *SkillValidationError (perfil canónico)", err)
	}
	if !strings.Contains(err.Error(), "description-quoted") || !strings.Contains(err.Error(), "description-trigger") {
		t.Errorf("el mensaje debe listar cada regla fallida: %v", err)
	}
}
