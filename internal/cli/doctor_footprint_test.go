package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/components/gga"
	"github.com/IGutierrezZ/axiom/v3/internal/mcpexclusion"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// stubDoctorUserPath fija las entradas del PATH de usuario que lee el doctor, para
// que el apartado de huella no dependa del PATH real de la máquina.
func stubDoctorUserPath(t *testing.T, entries []string) {
	t.Helper()
	orig := doctorUserPathEntriesFn
	t.Cleanup(func() { doctorUserPathEntriesFn = orig })
	doctorUserPathEntriesFn = func() []string { return entries }
}

// runDoctorOutput ejecuta RunDoctor con HOME aislado y todos los binarios
// resueltos salvo los que indique missing.
func runDoctorOutput(t *testing.T, home string, missing ...string) string {
	t.Helper()
	origLook, origAvail, origHTTP := lookPathFn, availableBytesFn, httpGetFn
	origPath, origHome, origExe := pathDirsFn, osUserHomeDirDoctor, osExecutableDoctor
	t.Cleanup(func() {
		lookPathFn, availableBytesFn, httpGetFn = origLook, origAvail, origHTTP
		pathDirsFn, osUserHomeDirDoctor, osExecutableDoctor = origPath, origHome, origExe
	})
	lookPathFn = func(name string) (string, error) {
		for _, m := range missing {
			if name == m || name == m+".ps1" {
				return "", errors.New("not found")
			}
		}
		return "/usr/local/bin/" + name, nil
	}
	availableBytesFn = func(string) (int64, error) { return 1024 * 1024 * 1024, nil }
	httpGetFn = func(string, time.Duration) (int, error) { return 200, nil }
	pathDirsFn = func() []string { return []string{"/usr/local/bin"} }
	osUserHomeDirDoctor = func() (string, error) { return home, nil }
	osExecutableDoctor = func() (string, error) { return "/usr/local/bin/axiom", nil }
	t.Setenv(engramHealthEnvVar, "https://engram.example.test")

	var buf bytes.Buffer
	if err := RunDoctor(context.Background(), &buf); err != nil {
		t.Fatalf("RunDoctor devolvió un error: %v", err)
	}
	return buf.String()
}

func writeDoctorState(t *testing.T, home, payload string) {
	t.Helper()
	dir := filepath.Join(home, ".axiom")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeFootprintFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// El doctor no marca GGA ausente como fallo: ya no forma parte de ningún preset.
func TestRunDoctor_GGAAbsentIsNotAFailure(t *testing.T) {
	// El proyecto de la carpeta actual aporta sus propias comprobaciones (project:skills): se aísla.
	t.Chdir(t.TempDir())
	stubDoctorUserPath(t, nil)
	home := t.TempDir()
	writeDoctorState(t, home, `{"installed_agents":["claude-code"]}`)

	out := runDoctorOutput(t, home, "gga")

	if strings.Contains(out, "tool:gga") {
		t.Errorf("GGA no elegido no debe aparecer entre las herramientas exigidas:\n%s", out)
	}
	if !strings.Contains(out, "Status:  healthy") {
		t.Errorf("el doctor debe ser healthy sin GGA:\n%s", out)
	}
}

// Si el usuario eligió GGA de forma explícita, su ausencia sí es un fallo.
func TestRunDoctor_GGARequiredWhenSelected(t *testing.T) {
	stubDoctorUserPath(t, nil)
	home := t.TempDir()
	writeDoctorState(t, home, `{"installed_agents":["claude-code"],"components":["gga"]}`)

	out := runDoctorOutput(t, home, "gga")

	if !strings.Contains(out, "[xx]  tool:gga") {
		t.Errorf("GGA elegido y ausente debe fallar:\n%s", out)
	}
}

// Sin nada a nivel de usuario salvo los binarios y el estado, el apartado solo
// lista lo declarado.
func TestRunDoctor_FootprintListsOnlyDeclaredOnCleanHome(t *testing.T) {
	stubDoctorUserPath(t, nil)
	home := t.TempDir()
	writeDoctorState(t, home, `{"installed_agents":["claude-code"]}`)

	out := runDoctorOutput(t, home)

	if !strings.Contains(out, "Huella de máquina (nivel de usuario)") {
		t.Fatalf("falta el apartado de huella de máquina:\n%s", out)
	}
	for _, want := range []string{"[declarado]", "binario axiom", "binario engram", "hub y estado de Axiom"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en el apartado:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"[resto global]", "[opcional]"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("no debe haber %s en un HOME limpio:\n%s", unwanted, out)
		}
	}
}

// ~/.agents/skills es del usuario y de otras herramientas: sin ninguna skill de
// Axiom dentro no es un resto de Axiom.
func TestRunDoctor_FootprintIgnoresAnAgentsSkillsDirWithoutAxiomSkills(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	writeDoctorState(t, home, `{"installed_agents":["claude-code"]}`)
	writeFootprintFile(t, filepath.Join(home, ".agents", "skills", "mi-skill", "SKILL.md"))

	out := runDoctorOutput(t, home)

	if strings.Contains(out, "~/.agents/skills") {
		t.Errorf("una carpeta de skills sin skills de Axiom no es un resto:\n%s", out)
	}
}

// Restos de una instalación global: se listan y se marcan como tales sin afectar
// al estado de salud.
func TestRunDoctor_FootprintMarksGlobalLeftovers(t *testing.T) {
	// El proyecto de la carpeta actual aporta sus propias comprobaciones (project:skills): se aísla.
	t.Chdir(t.TempDir())
	home := t.TempDir()
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	writeDoctorState(t, home, `{"installed_agents":["claude-code"]}`)

	skills := filepath.Join(home, ".agents", "skills", "sdd-init")
	if err := os.MkdirAll(skills, 0o755); err != nil {
		t.Fatal(err)
	}
	ggaRuntime := gga.RuntimePRModePath(home)
	writeFootprintFile(t, ggaRuntime)
	plugin := filepath.Join(home, ".config", "opencode", "plugins", "skill-registry.ts")
	writeFootprintFile(t, plugin)
	if err := os.MkdirAll(filepath.Join(home, ".gentle-ai"), 0o755); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(home, ".axiom", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	engramDir := t.TempDir()
	engramName := "engram"
	if doctorGOOS == "windows" {
		engramName = "engram.exe"
	}
	writeFootprintFile(t, filepath.Join(engramDir, engramName))
	stubDoctorUserPath(t, []string{engramDir, binDir})

	out := runDoctorOutput(t, home)

	for _, want := range []string{
		"[resto global] GGA",
		"[resto global] ~/.agents/skills",
		"[resto global] plugins de OpenCode en HOME",
		"[resto global] ~/.gentle-ai",
		"[declarado]    PATH: binario de Engram",
		"[opcional]     PATH: lanzadores de Axiom",
		"propios de una instalación global",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en la salida:\n%s", want, out)
		}
	}
	if !strings.Contains(out, ggaRuntime) && !strings.Contains(out, filepath.Dir(filepath.Dir(ggaRuntime))) {
		t.Errorf("GGA debe nombrar sus rutas de HOME:\n%s", out)
	}
	if !strings.Contains(out, "Status:  healthy") {
		t.Errorf("los restos globales no deben cambiar el estado de salud:\n%s", out)
	}
}

// GGA elegido de forma explícita figura como opcional, no como resto.
func TestRunDoctor_FootprintShowsSelectedGGAAsOptional(t *testing.T) {
	stubDoctorUserPath(t, nil)
	home := t.TempDir()
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	writeDoctorState(t, home, `{"installed_agents":["claude-code"],"components":["gga"]}`)
	writeFootprintFile(t, gga.RuntimePRModePath(home))

	out := runDoctorOutput(t, home)

	if !strings.Contains(out, "[opcional]     GGA") {
		t.Errorf("GGA elegido debe figurar como opcional:\n%s", out)
	}
	if strings.Contains(out, "[resto global] GGA") {
		t.Errorf("GGA elegido no es un resto:\n%s", out)
	}
}

func TestCollectMachineFootprintEmptyHomeIsEmpty(t *testing.T) {
	stubDoctorUserPath(t, nil)
	origLook := lookPathFn
	t.Cleanup(func() { lookPathFn = origLook })
	lookPathFn = func(string) (string, error) { return "", errors.New("not found") }

	if got := collectMachineFootprint(t.TempDir(), "", state.InstallState{}); len(got) != 0 {
		t.Fatalf("un HOME vacío sin binarios no debe tener huella, got %+v", got)
	}

	var buf bytes.Buffer
	renderMachineFootprint(&buf, nil)
	if !strings.Contains(buf.String(), "Axiom no tiene nada a nivel de usuario") {
		t.Errorf("salida inesperada: %q", buf.String())
	}
}

// El contexto de Axiom que existe a la vez en HOME y en el proyecto sale como
// duplicado, con las dos rutas y el aviso de divergencia, y es informativo.
func TestCollectMachineFootprintReportsDuplicateContext(t *testing.T) {
	stubDoctorUserPath(t, nil)
	origLook := lookPathFn
	t.Cleanup(func() { lookPathFn = origLook })
	lookPathFn = func(string) (string, error) { return "", errors.New("not found") }

	home, root := t.TempDir(), t.TempDir()
	userFile := filepath.Join(home, ".claude", "CLAUDE.md")
	projectFile := filepath.Join(root, ".claude", "CLAUDE.md")
	for path, version := range map[string]string{userFile: "v1", projectFile: "v2"} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		block := "<!-- axiom:sdd-orchestrator -->\nreglas " + version + "\n<!-- /axiom:sdd-orchestrator -->\n"
		if err := os.WriteFile(path, []byte(block), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Sin raíz de proyecto no se comprueba nada.
	if got := collectMachineFootprint(home, "", state.InstallState{}); len(got) != 0 {
		t.Fatalf("sin raíz no debe haber duplicados, got %+v", got)
	}

	entries := collectMachineFootprint(home, root, state.InstallState{})
	var duplicates []footprintEntry
	for _, entry := range entries {
		if entry.Kind == footprintDuplicate {
			duplicates = append(duplicates, entry)
		}
	}
	if len(duplicates) != 1 {
		t.Fatalf("quiero 1 duplicado, got %+v", entries)
	}
	detail := duplicates[0].Detail
	for _, want := range []string{userFile, projectFile, "divergen"} {
		if !strings.Contains(detail, want) {
			t.Errorf("el detalle %q no contiene %q", detail, want)
		}
	}

	var buf bytes.Buffer
	renderMachineFootprint(&buf, entries)
	out := buf.String()
	if !strings.Contains(out, "[duplicado]") || !strings.Contains(out, "1 contexto(s) de Axiom duplicado(s)") {
		t.Errorf("falta la línea de duplicados:\n%s", out)
	}
	if strings.Contains(out, "propios de una instalación global") {
		t.Errorf("un duplicado no es un resto global:\n%s", out)
	}
}

// Un servidor MCP de usuario cargado en el proyecto y sin llamadas en 30 días sale como
// «sin uso» (informativo); uno usado, no; y sin transcripciones del proyecto no sale nada.
func TestCollectMachineFootprintReportsUnusedMCPServers(t *testing.T) {
	stubDoctorUserPath(t, nil)
	origLook := lookPathFn
	t.Cleanup(func() { lookPathFn = origLook })
	lookPathFn = func(string) (string, error) { return "", errors.New("not found") }

	home, root := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("AXIOM_STATE_DIR", filepath.Join(home, ".axiom"))
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"engram":{},"ocioso":{}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	unusedEntries := func(entries []footprintEntry) []footprintEntry {
		var found []footprintEntry
		for _, entry := range entries {
			if entry.Kind == footprintUnused {
				found = append(found, entry)
			}
		}
		return found
	}

	// Sin transcripciones: «sin datos», no «sin uso».
	if got := unusedEntries(collectMachineFootprint(home, root, state.InstallState{})); len(got) != 0 {
		t.Fatalf("sin transcripciones no debe haber entradas, got %+v", got)
	}

	folder := filepath.Join(home, ".claude", "projects", mcpexclusion.EncodeProjectDir(root))
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"assistant","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `","message":{"content":[{"type":"tool_use","name":"mcp__engram__mem_save","input":{}}]}}` + "\n"
	if err := os.WriteFile(filepath.Join(folder, "s.jsonl"), []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}

	// Con raíz vacía no se comprueba nada.
	if got := unusedEntries(collectMachineFootprint(home, "", state.InstallState{})); len(got) != 0 {
		t.Fatalf("sin raíz no debe haber entradas, got %+v", got)
	}

	entries := collectMachineFootprint(home, root, state.InstallState{})
	got := unusedEntries(entries)
	if len(got) != 1 || got[0].Name != "MCP: ocioso" {
		t.Fatalf("quiero solo ocioso sin uso, got %+v", got)
	}

	var buf bytes.Buffer
	renderMachineFootprint(&buf, entries)
	out := buf.String()
	if !strings.Contains(out, "[sin uso]") || !strings.Contains(out, "1 servidor(es) MCP cargado(s) sin uso") {
		t.Errorf("falta la línea de sin uso:\n%s", out)
	}
}
