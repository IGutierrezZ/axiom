package telemetryruntime

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/IGutierrezZ/axiom/v3/internal/components/mutationjournal"
)

const ownershipSchema = "gentle-ai.telemetry-runtime-ownership/v1"

// Axiom no longer ships or installs the OpenCode telemetry plugin. This package
// only retires an installed copy, so ownership is proven against a fixed,
// append-never allowlist of every plugin digest ever shipped, not against an
// embedded asset.
const (
	// Commit a9cab7dd embedded telemetry-runtime.ts (v1) with this digest.
	shippedPluginDigestA9cab7dd = "902fe09299c0bb04590f12ba993a196b838e1fbbd8a94e96100ea3d5f30182b7"
	// Upstream Gentle-AI v3.0.2 embedded telemetry-runtime.ts with this digest.
	shippedPluginDigestGentleAIV3 = "54150f7d76d684c39d2567312d22bd5a188390be5d98da16eb1e8833f9665bd4"
	// The last Axiom v1 plugin (OpenCode 1.x).
	shippedPluginDigestAxiomV1 = "ff05ed23208eb57264b2bcb74ca9c386a2ceedc399362df8b2199cfedc612479"
	// The last Axiom v2 plugin (OpenCode 2.x).
	shippedPluginDigestAxiomV2 = "8053fc82e69ee7a338e10797ebec00395f1cc3ae9d0b478422fccefaf520f803"
)

var shippedPluginDigests = map[string]struct{}{
	shippedPluginDigestA9cab7dd:   {},
	shippedPluginDigestGentleAIV3: {},
	shippedPluginDigestAxiomV1:    {},
	shippedPluginDigestAxiomV2:    {},
}

// Windows exposes writable regular files as 0666 regardless of the requested
// POSIX permission bits. Retain exact modes elsewhere and never equate a
// read-only file with a writable managed file.
func managedModeMatches(expected, observed os.FileMode) bool {
	return managedModeMatchesForOS(runtime.GOOS, expected, observed)
}

func managedModeMatchesForOS(goos string, expected, observed os.FileMode) bool {
	if expected == observed {
		return true
	}
	return goos == "windows" &&
		expected.Perm()&0222 != 0 &&
		observed.Perm() == 0666 &&
		expected&^os.ModePerm == observed&^os.ModePerm
}

type managedManifest struct {
	Schema string                    `json:"schema"`
	File   mutationjournal.OwnedFile `json:"file"`
}

// ManagedPaths accepts the adapter's resolved config directory, including XDG.
// The manifest lives outside plugins; no user JSON or exporter is modified.
func ManagedPaths(configDir string) []string {
	return []string{filepath.Join(configDir, "plugins", "telemetry-runtime.ts"), filepath.Join(configDir, ".gentle-ai-telemetry-runtime.json")}
}

// Refuse indirection strictly inside the caller-resolved configuration root,
// both a symlinked intermediate directory (e.g. a symlinked "plugins"
// directory) and a symlinked leaf managed file. The generic journal only
// refuses a leaf or a root escape.
//
// The root itself may be a symlink: an agent's whole configuration directory
// symlinked into a tracked dotfiles repository (stow/chezmoi pattern) is a
// valid managed root as long as it resolves to an existing directory. That
// symlink is not treated as indirection to refuse; only a dangling root
// symlink, or one resolving to a non-directory, is still refused, since sync
// cannot safely install files there.
func checkManagedPath(configDir, path string) error {
	// ManagedPaths builds cleaned paths, so the root only matches a cleaned
	// candidate; a caller-supplied trailing separator must not turn the
	// accepted root symlink back into a refusal.
	root := filepath.Clean(configDir)
	if info, err := os.Lstat(root); err != nil && !os.IsNotExist(err) {
		return err
	} else if err == nil && info.Mode()&os.ModeSymlink != 0 {
		if target, statErr := os.Stat(root); statErr != nil || !target.IsDir() {
			return fmt.Errorf("telemetry runtime symlink conflict: %s is a symlink that does not resolve to an existing directory; point it at a directory or replace it with one, then rerun 'axiom sync'", root)
		}
	}
	for _, candidate := range []string{filepath.Dir(path), path} {
		if candidate == root {
			continue
		}
		info, err := os.Lstat(candidate)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("telemetry runtime symlink conflict: %s is a symlink inside the managed root; replace it with a regular file or directory, then rerun 'axiom sync'", candidate)
		}
	}
	return nil
}

func readManaged(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 65536 {
		return nil, fmt.Errorf("telemetry runtime ownership conflict: %s", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 65537))
	if len(data) > 65536 {
		return nil, fmt.Errorf("telemetry runtime manifest size conflict")
	}
	return data, err
}

// inspect refuses adoption, even when an unowned file matches this release.
// A missing owned asset is repairable; altered bytes or metadata are not.
func inspect(configDir string) ([][]byte, error) {
	paths := ManagedPaths(configDir)
	current := make([][]byte, 2)
	journal := mutationjournal.New(configDir)
	for i, path := range paths {
		if err := checkManagedPath(configDir, path); err != nil {
			return nil, err
		}
		if err := journal.Validate(path); err != nil {
			return nil, err
		}
		data, err := readManaged(path)
		if err != nil {
			return nil, err
		}
		current[i] = data
	}
	conflict := fmt.Errorf("telemetry runtime ownership conflict in %s; custom files preserved", configDir)
	if current[1] == nil {
		if current[0] != nil {
			return nil, conflict
		}
		return current, nil
	}
	var manifest managedManifest
	root, err := manifestObject(current[1], "schema", "file")
	if err != nil {
		return nil, conflict
	}
	file, err := manifestObject(root["file"], "after", "afterHash", "overlay", "mode")
	if err != nil || !bytes.Equal(bytes.TrimSpace(file["overlay"]), []byte("false")) || json.Unmarshal(current[1], &manifest) != nil || manifest.Schema != ownershipSchema ||
		(!managedModeMatches(0600, os.FileMode(manifest.File.Mode)) && !managedModeMatches(0644, os.FileMode(manifest.File.Mode))) || manifest.File.AfterHash != fmt.Sprintf("%x", sha256.Sum256([]byte(manifest.File.After))) {
		return nil, conflict
	}
	if _, shipped := shippedPluginDigests[manifest.File.AfterHash]; !shipped {
		return nil, conflict
	}
	for i, path := range paths {
		if current[i] == nil {
			continue
		}
		info, err := os.Lstat(path)
		expected := os.FileMode(0600)
		if i == 0 {
			expected = os.FileMode(manifest.File.Mode)
		}
		if err != nil || !managedModeMatches(expected, info.Mode()) {
			return nil, conflict
		}
	}
	if current[0] != nil && string(current[0]) != manifest.File.After {
		return nil, conflict
	}
	return current, nil
}

// Decode exact required keys before struct decoding: Go structs alone accept
// aliases, duplicates and omitted/null scalar fields. Each object is bounded.
func manifestObject(raw []byte, keys ...string) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return nil, errors.New("invalid ownership object")
	}
	fields := map[string]json.RawMessage{}
	for d.More() {
		token, err = d.Token()
		key, ok := token.(string)
		known := false
		for _, expected := range keys {
			known = known || key == expected
		}
		if err != nil || !ok || !known || fields[key] != nil {
			return nil, errors.New("invalid ownership key")
		}
		var value json.RawMessage
		if err := d.Decode(&value); err != nil {
			return nil, err
		}
		fields[key] = value
	}
	if _, err = d.Token(); err != nil {
		return nil, err
	}
	if _, err = d.Token(); err != io.EOF || len(fields) != len(keys) {
		return nil, errors.New("incomplete ownership object")
	}
	return fields, nil
}

// CheckManaged refuses unsafe/conflicting targets before a lifecycle snapshot.
// Removal also rechecks at execution time.
func CheckManaged(configDir string) error {
	_, err := inspect(configDir)
	return err
}

// RemoveManaged validates the pair immediately before transactional removal.
// Drift is an explicit conflict, not permission to remove a user-modified asset.
func RemoveManaged(configDir string) (removed []string, err error) {
	if _, err = inspect(configDir); err != nil {
		return nil, err
	}
	journal := mutationjournal.New(configDir)
	defer func() {
		if err != nil {
			err = errors.Join(err, journal.Restore())
		}
	}()
	paths := ManagedPaths(configDir)
	for _, path := range paths {
		if err = journal.Capture(path); err != nil {
			return nil, err
		}
	}
	if _, err = inspect(configDir); err != nil {
		return nil, err
	}
	for _, path := range paths {
		var changed bool
		changed, err = journal.Remove(path)
		if err != nil {
			return nil, err
		}
		if changed {
			removed = append(removed, path)
		}
	}
	return removed, nil
}
