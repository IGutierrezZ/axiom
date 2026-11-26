package sddstatus

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeLegacyRuntimeFile(t *testing.T, base, content string) string {
	t.Helper()
	path := filepath.Join(base, "sdd-runtime", "v1", "cambio", "HEAD")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOpenRuntimeStoreMigratesLegacyContainer(t *testing.T) {
	repo := initRuntimeLedgerRepo(t)
	commonDir := commonDirOracle(t, repo)
	writeLegacyRuntimeFile(t, filepath.Join(commonDir, "gentle-ai"), "viejo")

	if _, err := OpenRuntimeStore(context.Background(), repo, "common-dir-change"); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(commonDir, "axiom", "sdd-runtime", "v1", "cambio", "HEAD")
	if got, err := os.ReadFile(moved); err != nil || string(got) != "viejo" {
		t.Fatalf("el almacén heredado no se movió a la ruta nueva: %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(commonDir, "gentle-ai", "sdd-runtime")); !os.IsNotExist(err) {
		t.Fatal("el almacén heredado debía desaparecer tras moverse")
	}

	// Idempotente.
	if _, err := OpenRuntimeStore(context.Background(), repo, "common-dir-change"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(moved); string(got) != "viejo" {
		t.Fatalf("repetir la apertura alteró el almacén: %q", got)
	}
}

func TestOpenRuntimeStoreKeepsExistingAxiomContainer(t *testing.T) {
	repo := initRuntimeLedgerRepo(t)
	commonDir := commonDirOracle(t, repo)
	legacy := writeLegacyRuntimeFile(t, filepath.Join(commonDir, "gentle-ai"), "viejo")
	current := writeLegacyRuntimeFile(t, filepath.Join(commonDir, "axiom"), "nuevo")

	if _, err := OpenRuntimeStore(context.Background(), repo, "common-dir-change"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(current); string(got) != "nuevo" {
		t.Fatalf("el almacén nuevo se pisó: %q", got)
	}
	if got, _ := os.ReadFile(legacy); string(got) != "viejo" {
		t.Fatalf("el almacén heredado debía quedar intacto: %q", got)
	}
}
