package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/components/gga"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// stubGGAHomebrewEnv aísla el paso de GGA de la máquina real: gga nunca está
// disponible, el PATH solo contiene los binarios de onPath y los prefijos
// estándar de Homebrew solo los de atPrefix. Todo comando va a run en vez de a
// un proceso real.
func stubGGAHomebrewEnv(t *testing.T, onPath, atPrefix []string, run func(name string, args ...string) error) string {
	t.Helper()
	home := t.TempDir()

	origHome := osUserHomeDir
	origCmdLookPath := cmdLookPath
	origOsStat := osStat
	origRunCmd := runCommand
	origGGAAvailableCheck := ggaAvailableCheck
	origCleanup := cleanupGGAInstallDir
	t.Cleanup(func() {
		osUserHomeDir = origHome
		cmdLookPath = origCmdLookPath
		osStat = origOsStat
		runCommand = origRunCmd
		ggaAvailableCheck = origGGAAvailableCheck
		cleanupGGAInstallDir = origCleanup
	})

	executable := filepath.Join(t.TempDir(), "executable")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	osUserHomeDir = func() (string, error) { return home, nil }
	cmdLookPath = func(file string) (string, error) {
		if slices.Contains(onPath, file) {
			return "/usr/bin/" + file, nil
		}
		return "", &os.PathError{Op: "lookpath", Path: file, Err: errors.New("executable file not found in $PATH")}
	}
	osStat = func(name string) (os.FileInfo, error) {
		slashed := filepath.ToSlash(name)
		for _, prefix := range []string{"/opt/homebrew/bin/", "/usr/local/bin/", "/home/linuxbrew/.linuxbrew/bin/"} {
			if strings.HasPrefix(slashed, prefix) {
				if slices.Contains(atPrefix, strings.TrimPrefix(slashed, prefix)) {
					return os.Stat(executable)
				}
				return nil, fs.ErrNotExist
			}
		}
		return os.Stat(name)
	}
	ggaAvailableCheck = func(system.PlatformProfile) bool { return false }
	cleanupGGAInstallDir = func() error { return nil }
	runCommand = run
	return home
}

func darwinGGAStep(home string, state *runtimeState) componentApplyStep {
	return componentApplyStep{
		id:           "component:gga",
		component:    model.ComponentGGA,
		homeDir:      home,
		workspaceDir: home,
		agents:       []model.AgentID{model.AgentClaudeCode},
		selection:    model.Selection{},
		profile:      system.PlatformProfile{OS: "darwin", PackageManager: "brew", Supported: true},
		state:        state,
	}
}

// En macOS el perfil siempre dice PackageManager "brew": sin Homebrew, la
// instalación entera moría con `exec: "brew": executable file not found`. Un
// Homebrew ausente es un prerrequisito opcional: la instalación termina bien,
// se escribe la configuración de GGA y el usuario recibe una acción manual en
// lugar de la nota falsa «GGA is now installed globally».
func TestRunInstallMacOSWithoutHomebrewDefersGGA(t *testing.T) {
	var brewCalls []string
	home := stubGGAHomebrewEnv(t, nil, nil, func(name string, args ...string) error {
		if filepath.Base(name) == "brew" {
			brewCalls = append(brewCalls, strings.Join(append([]string{name}, args...), " "))
			return errors.New(`exec: "brew": executable file not found in $PATH`)
		}
		return nil
	})

	result, err := RunInstall([]string{"--agent", "claude-code", "--component", "gga"}, macOSDetectionResult())
	if err != nil {
		t.Fatalf("RunInstall() error = %v, want nil: la falta de Homebrew no debe abortar", err)
	}
	if len(brewCalls) != 0 {
		t.Fatalf("se invocó brew sin Homebrew instalado: %q", brewCalls)
	}
	if !result.Verify.Ready {
		t.Fatalf("Verify.Ready = false, want true: la configuración de GGA debe aplicarse igualmente")
	}
	if _, err := os.Stat(gga.ConfigPath(home)); err != nil {
		t.Fatalf("no se escribió la configuración de GGA: %v", err)
	}
	if !slices.Contains(result.Execution.ManualActions, GGAHomebrewDeferredAction) {
		t.Fatalf("ManualActions = %q, want la acción del prerrequisito Homebrew", result.Execution.ManualActions)
	}
	if rendered := RenderInstallManualActions(result); !strings.Contains(rendered, "axiom install --component gga") {
		t.Fatalf("RenderInstallManualActions() = %q, want el comando para reintentar", rendered)
	}
	if strings.Contains(result.Verify.FinalNote, "GGA is now installed globally") {
		t.Fatalf("FinalNote afirma que GGA se instaló: %q", result.Verify.FinalNote)
	}
}

// El camino de la TUI lee las acciones manuales de GGA del estado de ejecución.
func TestGGAStepWithoutHomebrewRecordsDeferredAction(t *testing.T) {
	home := stubGGAHomebrewEnv(t, nil, nil, func(string, ...string) error { return nil })
	state := &runtimeState{}

	if err := darwinGGAStep(home, state).Run(); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if !slices.Contains(state.ggaManualActions(), GGAHomebrewDeferredAction) {
		t.Fatalf("ggaManualActions() = %q, want la acción del prerrequisito Homebrew", state.ggaManualActions())
	}
}

// Un Homebrew hallado solo en su prefijo estándar se usa por ruta absoluta,
// igual que hace Engram.
func TestGGAStepUsesHomebrewOutsidePath(t *testing.T) {
	var commands []string
	home := stubGGAHomebrewEnv(t, nil, []string{"brew"}, func(name string, args ...string) error {
		commands = append(commands, strings.Join(append([]string{filepath.ToSlash(name)}, args...), " "))
		return nil
	})
	state := &runtimeState{}

	if err := darwinGGAStep(home, state).Run(); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	want := []string{
		"/opt/homebrew/bin/brew tap Gentleman-Programming/homebrew-tap",
		"/opt/homebrew/bin/brew reinstall gga",
	}
	if len(commands) < len(want) || !slices.Equal(commands[:len(want)], want) {
		t.Fatalf("commands = %q, want la secuencia de Homebrew que empiece por %q", commands, want)
	}
	if state.ggaInstallDeferred || state.ggaManualActions() != nil {
		t.Fatal("se pospuso la instalación de GGA aunque Homebrew está disponible")
	}
}

// Un Homebrew presente que falla por sus propios motivos sigue siendo un fallo
// real de instalación, no un prerrequisito pospuesto.
func TestGGAStepHomebrewFailureStillFails(t *testing.T) {
	home := stubGGAHomebrewEnv(t, []string{"brew"}, nil, func(name string, args ...string) error {
		if filepath.Base(name) == "brew" {
			return errors.New("simulated brew tap failure: exit status 1")
		}
		return nil
	})
	state := &runtimeState{}

	err := darwinGGAStep(home, state).Run()
	if err == nil || !strings.Contains(err.Error(), "simulated brew tap failure") {
		t.Fatalf("Run() error = %v, want el fallo de brew", err)
	}
	if state.ggaInstallDeferred {
		t.Fatal("un fallo real de Homebrew se notificó como instalación pospuesta")
	}
}
