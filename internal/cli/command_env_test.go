package cli

import (
	"reflect"
	"testing"
)

func TestCommandEnvLeavesNonBrewCommandsUnchanged(t *testing.T) {
	base := []string{"HOME=/home/dev", "PATH=/usr/bin"}
	got := commandEnv("npm", []string{"install", "-g", "pkg"}, base)
	if !reflect.DeepEqual(got, base) {
		t.Fatalf("commandEnv(npm, ...) = %v, want unchanged %v", got, base)
	}
}

func TestCommandEnvAddsHomebrewNoAutoUpdateForBrewInstall(t *testing.T) {
	base := []string{"HOME=/home/dev"}
	got := commandEnv("brew", []string{"install", "engram"}, base)
	want := []string{"HOME=/home/dev", "HOMEBREW_NO_AUTO_UPDATE=1", "HOMEBREW_NO_INSTALL_CLEANUP=1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("commandEnv(brew install, ...) = %v, want %v", got, want)
	}
	if len(base) != 1 {
		t.Fatalf("commandEnv modificó base: %v", base)
	}
}

func TestCommandEnvAddsHomebrewNoAutoUpdateForResolvedBrewPath(t *testing.T) {
	base := []string{"HOME=/home/dev"}
	got := commandEnv("/opt/homebrew/bin/brew", []string{"tap", "Gentleman-Programming/homebrew-tap"}, base)
	want := []string{"HOME=/home/dev", "HOMEBREW_NO_AUTO_UPDATE=1", "HOMEBREW_NO_INSTALL_CLEANUP=1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("commandEnv(ruta de brew resuelta, ...) = %v, want %v", got, want)
	}
}

func TestCommandEnvExcludesBrewUpdateAndUpgrade(t *testing.T) {
	base := []string{"HOME=/home/dev"}
	for _, args := range [][]string{{"update"}, {"upgrade", "--formula", "engram"}} {
		got := commandEnv("brew", args, base)
		if !reflect.DeepEqual(got, base) {
			t.Fatalf("commandEnv(brew %v) = %v, want unchanged %v", args, got, base)
		}
	}
}

func TestCommandEnvNeverOverridesExplicitUserValue(t *testing.T) {
	base := []string{"HOME=/home/dev", "HOMEBREW_NO_AUTO_UPDATE=0"}
	got := commandEnv("brew", []string{"install", "engram"}, base)
	want := []string{"HOME=/home/dev", "HOMEBREW_NO_AUTO_UPDATE=0", "HOMEBREW_NO_INSTALL_CLEANUP=1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("commandEnv() = %v, want %v (HOMEBREW_NO_AUTO_UPDATE explícito preservado)", got, want)
	}
}
