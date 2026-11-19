package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeCheckSkill(t *testing.T, project, rel, body string) {
	t.Helper()
	p := filepath.Join(project, filepath.FromSlash(rel), "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSkillCheckJSONConErroresSaleConUno(t *testing.T) {
	project, _ := makeSkillIndexProject(t)
	writeCheckSkill(t, project, ".agents/skills/rota", "# sin frontmatter\n")

	var stdout, stderr bytes.Buffer
	code := runSkillGroup([]string{"check", "--json", "--path", project}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit = %d, want 1 (stderr=%q)", code, stderr.String())
	}
	var rep skillCheckReport
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatalf("JSON inválido: %v\n%s", err, stdout.String())
	}
	if len(rep.Skills) != 2 || rep.Errors != 1 {
		t.Fatalf("skills=%d errors=%d, want 2 y 1: %+v", len(rep.Skills), rep.Errors, rep)
	}
	var found bool
	for _, s := range rep.Skills {
		if s.Name == "rota" {
			found = len(s.Findings) == 1 && s.Findings[0].Rule == "frontmatter-start" && s.Profile == "portable" && s.Path != ""
		}
	}
	if !found {
		t.Errorf("falta el informe de la skill rota: %s", stdout.String())
	}
	for _, key := range []string{`"skills"`, `"set_findings"`, `"errors"`, `"warnings"`} {
		if !strings.Contains(stdout.String(), key) {
			t.Errorf("el JSON debe incluir %s", key)
		}
	}
}

func TestSkillCheckLimpioSaleConCeroYEscribeTexto(t *testing.T) {
	project, _ := makeSkillIndexProject(t)

	var stdout, stderr bytes.Buffer
	code := runSkillGroup([]string{"check", "--path", project}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stdout=%q stderr=%q)", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"go-testing", "perfil: portable", "sin hallazgos", "Resumen: 1 skills, 0 error(es), 0 aviso(s)"} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida debe contener %q:\n%s", want, out)
		}
	}
}

func TestSkillCheckJSONSinSkillsDevuelveListasVacias(t *testing.T) {
	_, home := makeSkillIndexProject(t)
	empty := t.TempDir()
	_ = home

	var stdout, stderr bytes.Buffer
	if code := runSkillGroup([]string{"check", "--json", "--path", empty}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"skills": []`) || !strings.Contains(stdout.String(), `"set_findings": []`) {
		t.Errorf("las listas vacías deben serializarse como []:\n%s", stdout.String())
	}
}

func TestSkillCheckRechazaArgumentosInesperados(t *testing.T) {
	makeSkillIndexProject(t)
	var stdout, stderr bytes.Buffer
	if code := runSkillGroup([]string{"check", "sobra"}, &stdout, &stderr); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "Uso: axiom skill check") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func TestSkillCheckRechazaRutaInexistente(t *testing.T) {
	var stdout, stderr bytes.Buffer
	missing := filepath.Join(t.TempDir(), "no-existe")
	if code := runSkillGroup([]string{"check", "--path", missing}, &stdout, &stderr); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "no existe") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func TestSkillGroupHelpListsCheck(t *testing.T) {
	if !strings.Contains(skillGroupHelp(), "check") || !strings.Contains(axiomHelpText(), "skill check") {
		t.Fatal("la ayuda debe documentar `skill check`")
	}
}
