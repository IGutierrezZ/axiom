package telemetryruntime

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/components/mutationjournal"
)

const testOwnershipMarker = "// gentle-ai:managed telemetry-runtime/v1\n"

// Every plugin digest ever shipped, with the file that proves it. The upstream
// v3.0.2 plugin is not vendored; its digest is pinned from the release.
var shippedFixtures = map[string]string{
	shippedPluginDigestA9cab7dd: "testdata/telemetry-runtime-a9cab7dd.ts",
	shippedPluginDigestAxiomV1:  "testdata/telemetry-runtime-axiom-v1.ts",
	shippedPluginDigestAxiomV2:  "testdata/telemetry-runtime-axiom-v2.ts",
}

func readShippedFixture(t *testing.T, digest string) []byte {
	t.Helper()
	data, err := os.ReadFile(shippedFixtures[digest])
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != digest {
		t.Fatalf("fixture %s digest = %s, want %s", shippedFixtures[digest], got, digest)
	}
	return data
}

func TestShippedPluginDigestAllowlistIsComplete(t *testing.T) {
	for digest := range shippedFixtures {
		readShippedFixture(t, digest)
		if _, ok := shippedPluginDigests[digest]; !ok {
			t.Errorf("digest %s is not in the shipped allowlist", digest)
		}
	}
	if _, ok := shippedPluginDigests[shippedPluginDigestGentleAIV3]; !ok {
		t.Error("upstream v3.0.2 digest is not in the shipped allowlist")
	}
	if len(shippedPluginDigests) != 4 {
		t.Errorf("allowlist has %d digests, want 4 (it is append-never)", len(shippedPluginDigests))
	}
}

func TestRemoveManagedRetiresEveryShippedPlugin(t *testing.T) {
	for digest := range shippedFixtures {
		t.Run(digest[:8], func(t *testing.T) {
			dir := t.TempDir()
			writeManagedTelemetryFixture(t, dir, readShippedFixture(t, digest))
			if err := CheckManaged(dir); err != nil {
				t.Fatalf("shipped plugin is not recognised as owned: %v", err)
			}
			customPath := filepath.Join(dir, "plugins", "custom.ts")
			if err := os.WriteFile(customPath, []byte("custom"), 0o600); err != nil {
				t.Fatal(err)
			}
			removed, err := RemoveManaged(dir)
			if err != nil || len(removed) != 2 {
				t.Fatalf("remove shipped plugin: %v, paths=%v", err, removed)
			}
			for _, path := range ManagedPaths(dir) {
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					t.Fatalf("owned path remains after removal: %s: %v", path, err)
				}
			}
			if custom, err := os.ReadFile(customPath); err != nil || string(custom) != "custom" {
				t.Fatalf("removal changed an unrelated file: %q, %v", custom, err)
			}
			if again, err := RemoveManaged(dir); err != nil || len(again) != 0 {
				t.Fatalf("second removal is not a no-op: %v, %v", again, err)
			}
		})
	}
}

func TestRemoveManagedUnapprovedAssetConflicts(t *testing.T) {
	dir := t.TempDir()
	unapproved := []byte(testOwnershipMarker + "// unapproved historical content\n")
	writeManagedTelemetryFixture(t, dir, unapproved)
	paths := ManagedPaths(dir)

	if err := CheckManaged(dir); err == nil {
		t.Fatal("unapproved asset passed ownership validation")
	}
	if _, err := RemoveManaged(dir); err == nil {
		t.Fatal("unapproved asset was removed")
	}
	if plugin, err := os.ReadFile(paths[0]); err != nil || !bytes.Equal(plugin, unapproved) {
		t.Fatalf("unapproved plugin was not preserved: %v", err)
	}
}

func TestRemoveManagedKeepsEditedShippedPlugin(t *testing.T) {
	dir := t.TempDir()
	writeManagedTelemetryFixture(t, dir, readShippedFixture(t, shippedPluginDigestAxiomV1))
	paths := ManagedPaths(dir)
	if err := os.WriteFile(paths[0], []byte("// user edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RemoveManaged(dir); err == nil {
		t.Fatal("edited plugin was removed")
	}
	if plugin, err := os.ReadFile(paths[0]); err != nil || string(plugin) != "// user edit\n" {
		t.Fatalf("edited plugin was not preserved: %q, %v", plugin, err)
	}
	if _, err := os.Stat(paths[1]); err != nil {
		t.Fatalf("manifest of a kept plugin was removed: %v", err)
	}
}

// writeManagedTelemetryFixture reproduces the pair the retired installer wrote:
// the plugin with mode 0644 and its ownership manifest with mode 0600.
func writeManagedTelemetryFixture(t *testing.T, dir string, content []byte) {
	t.Helper()
	paths := ManagedPaths(dir)
	if err := os.MkdirAll(filepath.Dir(paths[0]), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths[0], content, 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := managedManifest{Schema: ownershipSchema, File: mutationjournal.OwnedFile{
		After: string(content), AfterHash: fmt.Sprintf("%x", sha256.Sum256(content)), Overlay: false, Mode: 0o644,
	}}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths[1], append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestOpenCodeTelemetryManagedModeEquivalence(t *testing.T) {
	for _, tt := range []struct {
		name     string
		goos     string
		expected os.FileMode
		observed os.FileMode
		want     bool
	}{
		{"exact writable on Unix", "linux", 0644, 0644, true},
		{"read-only drift on Unix", "linux", 0644, 0444, false},
		{"different writable mode on Unix", "linux", 0644, 0640, false},
		{"widened writable plugin on Windows", "windows", 0644, 0666, true},
		{"widened writable manifest on Windows", "windows", 0600, 0666, true},
		{"read-only expected file remains distinct on Windows", "windows", 0444, 0666, false},
		{"other writable modes remain distinct on Windows", "windows", 0644, 0664, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := managedModeMatchesForOS(tt.goos, tt.expected, tt.observed); got != tt.want {
				t.Errorf("managedModeMatchesForOS(%q, %#o, %#o) = %t, want %t", tt.goos, tt.expected, tt.observed, got, tt.want)
			}
		})
	}
}

func TestOpenCodeTelemetryManifestStrictness(t *testing.T) {
	for _, kind := range []string{"alias", "nested-alias", "duplicate", "nested-duplicate", "missing", "null", "schema-null", "mode-null", "mode-type", "mode-value", "mode-drift", "metadata-mode", "unknown-pair", "oversized", "trailing"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			writeManagedTelemetryFixture(t, dir, readShippedFixture(t, shippedPluginDigestAxiomV1))
			paths := ManagedPaths(dir)
			raw, _ := os.ReadFile(paths[1])
			text := string(raw)
			switch kind {
			case "alias":
				text = strings.Replace(text, `"schema"`, `"Schema"`, 1)
			case "nested-alias":
				text = strings.Replace(text, `"afterHash"`, `"AfterHash"`, 1)
			case "nested-duplicate":
				text = strings.Replace(text, `"overlay":`, `"overlay":true,"overlay":`, 1)
			case "schema-null":
				text = strings.Replace(text, `"schema": "`+ownershipSchema+`"`, `"schema":null`, 1)
			case "mode-null":
				text = string(replaceManagedManifestMode(t, raw, json.RawMessage("null")))
			case "trailing":
				text += `{}`
			case "duplicate":
				text = strings.Replace(text, `"schema":`, `"schema":"discarded","schema":`, 1)
			case "missing":
				text = strings.Replace(text, `"overlay": false,`, "", 1)
			case "null":
				text = strings.Replace(text, `"overlay": false`, `"overlay": null`, 1)
			case "mode-type":
				text = string(replaceManagedManifestMode(t, raw, json.RawMessage(`"420"`)))
			case "mode-value":
				text = string(replaceManagedManifestMode(t, raw, json.RawMessage("511")))
			case "mode-drift":
				if runtime.GOOS == "windows" {
					t.Skip("Windows does not preserve POSIX chmod mode drift")
				}
				if err := os.Chmod(paths[0], 0600); err != nil {
					t.Fatal(err)
				}
			case "metadata-mode":
				if runtime.GOOS == "windows" {
					t.Skip("Windows does not support writable metadata after POSIX chmod drift")
				}
				if err := os.Chmod(paths[1], 0644); err != nil {
					t.Fatal(err)
				}
			case "unknown-pair":
				custom := testOwnershipMarker + "// custom data, not a package asset\n"
				var m managedManifest
				if err := json.Unmarshal(raw, &m); err != nil {
					t.Fatal(err)
				}
				m.File.After = custom
				m.File.AfterHash = fmt.Sprintf("%x", sha256.Sum256([]byte(custom)))
				raw, _ = json.Marshal(m)
				text = string(raw)
				if err := os.WriteFile(paths[0], []byte(custom), 0644); err != nil {
					t.Fatal(err)
				}
			case "oversized":
				text += strings.Repeat(" ", 65537)
			}
			if err := os.WriteFile(paths[1], []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			before, _ := os.ReadFile(paths[0])
			if err := CheckManaged(dir); err == nil {
				t.Error("manifest accepted")
			}
			if _, err := RemoveManaged(dir); err == nil {
				t.Error("unsafe uninstall accepted")
			}
			if after, err := os.ReadFile(paths[0]); err != nil || !bytes.Equal(before, after) {
				t.Error("custom pair not preserved")
			}
		})
	}
}

// replaceManagedManifestMode targets the parsed field instead of assuming the
// platform's serialized writable mode (0600 on POSIX, 0666 on Windows).
func replaceManagedManifestMode(t *testing.T, raw []byte, mode json.RawMessage) []byte {
	t.Helper()
	var manifest map[string]json.RawMessage
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	var file map[string]json.RawMessage
	if err := json.Unmarshal(manifest["file"], &file); err != nil {
		t.Fatal(err)
	}
	file["mode"] = mode
	encodedFile, err := json.Marshal(file)
	if err != nil {
		t.Fatal(err)
	}
	manifest["file"] = encodedFile
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestOpenCodeTelemetryManagedLifecycle(t *testing.T) {
	dir := t.TempDir()
	writeManagedTelemetryFixture(t, dir, readShippedFixture(t, shippedPluginDigestAxiomV1))
	paths := ManagedPaths(dir)
	raw, _ := os.ReadFile(paths[1])
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths[1], compact.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if err := CheckManaged(dir); err != nil {
		t.Fatalf("compact manifest is not owned: %v", err)
	}
	// A missing plugin with an intact manifest is still owned and removable.
	if err := os.Remove(paths[0]); err != nil {
		t.Fatal(err)
	}
	if removed, err := RemoveManaged(dir); err != nil || len(removed) != 1 {
		t.Fatal("manifest-only removal", removed, err)
	}
	for _, path := range paths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("owned file remains", path, err)
		}
	}
}

// TestOpenCodeTelemetryManagedAcceptsSymlinkedConfigRoot covers the
// stow/chezmoi dotfiles pattern where the agent's whole configuration
// directory is a symlink into a tracked repository (issue #4559). The
// managed pair must validate and remove cleanly through that symlinked root,
// resolved to the real target rather than being refused outright.
func TestOpenCodeTelemetryManagedAcceptsSymlinkedConfigRoot(t *testing.T) {
	target := filepath.Join(t.TempDir(), "opencode-real")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "opencode")
	if err := os.Symlink(target, link); err != nil {
		t.Skip(err)
	}

	writeManagedTelemetryFixture(t, target, readShippedFixture(t, shippedPluginDigestAxiomV2))
	for _, path := range ManagedPaths(target) {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("managed file missing under resolved root: %s: %v", path, err)
		}
	}
	if err := CheckManaged(link); err != nil {
		t.Fatalf("check managed through symlinked root: %v", err)
	}

	removed, err := RemoveManaged(link)
	if err != nil || len(removed) != 2 {
		t.Fatalf("remove managed through symlinked root: removed=%v err=%v", removed, err)
	}
	for _, path := range ManagedPaths(target) {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("managed file remains under resolved root after removal: %s: %v", path, err)
		}
	}
	if info, err := os.Lstat(target); err != nil || !info.IsDir() {
		t.Fatalf("resolved root directory was removed or altered: %v", err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("config root symlink was removed or altered: %v", err)
	}
}

// TestOpenCodeTelemetryManagedRefusesSymlinkedPluginsDirectory covers real
// indirection strictly inside a regular (non-symlinked) managed root: a
// symlinked "plugins" directory must still be refused, unlike the root
// symlink itself.
func TestOpenCodeTelemetryManagedRefusesSymlinkedPluginsDirectory(t *testing.T) {
	dir := t.TempDir()
	realPlugins := filepath.Join(t.TempDir(), "plugins-real")
	if err := os.MkdirAll(realPlugins, 0o700); err != nil {
		t.Fatal(err)
	}
	pluginsLink := filepath.Join(dir, "plugins")
	if err := os.Symlink(realPlugins, pluginsLink); err != nil {
		t.Skip(err)
	}

	err := CheckManaged(dir)
	if err == nil {
		t.Fatal("symlinked plugins directory was accepted")
	}
	if !strings.Contains(err.Error(), "telemetry runtime symlink conflict:") {
		t.Fatalf("unexpected error missing conflict prefix: %v", err)
	}
	if !strings.Contains(err.Error(), "rerun 'axiom sync'") {
		t.Fatalf("error does not name an executable exit: %v", err)
	}
	if _, err := RemoveManaged(dir); err == nil {
		t.Fatal("symlinked plugins directory passed removal check")
	}
}

func TestOpenCodeTelemetryManagedAcceptsSymlinkedConfigRootWithTrailingSeparator(t *testing.T) {
	target := filepath.Join(t.TempDir(), "opencode-real")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "opencode")
	if err := os.Symlink(target, link); err != nil {
		t.Skip(err)
	}
	uncleaned := link + string(filepath.Separator)

	writeManagedTelemetryFixture(t, target, readShippedFixture(t, shippedPluginDigestAxiomV1))
	for _, path := range ManagedPaths(target) {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("managed file missing under resolved target: %v", err)
		}
	}
	if err := CheckManaged(uncleaned); err != nil {
		t.Fatalf("check through uncleaned symlinked config root: %v", err)
	}
}

func TestOpenCodeTelemetryManagedRefusesDanglingConfigRootSymlink(t *testing.T) {
	link := filepath.Join(t.TempDir(), "opencode")
	if err := os.Symlink(filepath.Join(t.TempDir(), "missing-target"), link); err != nil {
		t.Skip(err)
	}

	err := CheckManaged(link)
	if err == nil {
		t.Fatal("dangling config root symlink was accepted")
	}
	if !strings.Contains(err.Error(), "telemetry runtime symlink conflict:") {
		t.Fatalf("unexpected error missing conflict prefix: %v", err)
	}
	if !strings.Contains(err.Error(), "does not resolve to an existing directory") {
		t.Fatalf("error does not explain the dangling root: %v", err)
	}
	if !strings.Contains(err.Error(), "rerun 'axiom sync'") {
		t.Fatalf("error does not name an executable exit: %v", err)
	}
	if _, err := RemoveManaged(link); err == nil {
		t.Fatal("dangling config root symlink passed removal check")
	}
}

func TestOpenCodeTelemetryManagedPreservesConflicts(t *testing.T) {
	for _, kind := range []string{"unowned", "modified", "symlink", "metadata"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			paths := ManagedPaths(dir)
			if kind == "modified" || kind == "metadata" {
				writeManagedTelemetryFixture(t, dir, readShippedFixture(t, shippedPluginDigestAxiomV1))
			}
			if err := os.MkdirAll(filepath.Dir(paths[0]), 0700); err != nil {
				t.Fatal(err)
			}
			target := paths[0]
			if kind == "metadata" {
				target = paths[1]
			}
			if kind == "symlink" {
				target = filepath.Join(dir, "custom.ts")
				if err := os.WriteFile(target, []byte("personal"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, paths[0]); err != nil {
					t.Skip(err)
				}
			} else if err := os.WriteFile(target, []byte("personal"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := RemoveManaged(dir); err == nil {
				t.Fatal("conflict not reported during uninstall")
			}
			if data, err := os.ReadFile(target); err != nil || string(data) != "personal" {
				t.Fatal("custom content changed", err)
			}
		})
	}
}

// A manifest may claim an allowlisted digest while recording different bytes.
// Ownership requires the recorded bytes to hash to the claimed digest, so a
// forged manifest cannot get an arbitrary plugin removed as Axiom-owned.
func TestRemoveManagedRejectsManifestWhoseAfterDoesNotMatchItsHash(t *testing.T) {
	dir := t.TempDir()
	custom := []byte("// user plugin passed off as a shipped one\n")
	paths := ManagedPaths(dir)
	if err := os.MkdirAll(filepath.Dir(paths[0]), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths[0], custom, 0o644); err != nil {
		t.Fatal(err)
	}
	forged := managedManifest{Schema: ownershipSchema, File: mutationjournal.OwnedFile{
		After: string(custom), AfterHash: shippedPluginDigestAxiomV1, Overlay: false, Mode: 0o644,
	}}
	raw, err := json.MarshalIndent(forged, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths[1], append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := shippedPluginDigests[forged.File.AfterHash]; !ok {
		t.Fatal("fixture must claim an allowlisted digest")
	}
	if err := CheckManaged(dir); err == nil {
		t.Fatal("manifest with a mismatched afterHash passed ownership validation")
	}
	if removed, err := RemoveManaged(dir); err == nil || len(removed) != 0 {
		t.Fatalf("forged manifest removed files: %v, %v", removed, err)
	}
	if plugin, err := os.ReadFile(paths[0]); err != nil || !bytes.Equal(plugin, custom) {
		t.Fatalf("plugin was not preserved: %v", err)
	}
	if _, err := os.Stat(paths[1]); err != nil {
		t.Fatalf("manifest was removed: %v", err)
	}
}
