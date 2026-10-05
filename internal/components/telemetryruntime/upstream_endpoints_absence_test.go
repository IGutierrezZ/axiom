package telemetryruntime

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Axiom sends no telemetry. The upstream collector endpoints must not come
// back into any production file, so a regression cannot start talking to the
// upstream servers again without this test failing.
var retiredUpstreamEndpoints = []string{
	"telemetry.gentlemanprogramming.com",
	"gentlemanprogramming.com/v1/events",
	"/v1/runtime-events",
}

// Historical records keep the strings on purpose: they describe what was
// removed. Matching is by repository-relative slash path; a trailing slash
// matches a whole directory.
var upstreamEndpointAllowlist = []string{
	"docs/audits/",
	"odd/tasks/",
	"openspec/changes/archive/",
	"docs/upstream-absorption-ledger.md",
}

const upstreamEndpointMaxFileBytes = 4 << 20

func TestUpstreamTelemetryEndpointsAbsentFromProductionFiles(t *testing.T) {
	root := upstreamEndpointRepositoryRoot(t)

	hits, err := upstreamEndpointHits(root)
	if err != nil {
		t.Fatalf("scan repository: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("retired upstream telemetry endpoint found in production files: %v", hits)
	}
}

func TestUpstreamTelemetryEndpointScannerDetectsAndSkips(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("internal/prod/client.go", `const u = "https://telemetry.gentlemanprogramming.com/v1/events"`)
	write("scripts/send.sh", "curl https://example.invalid/v1/runtime-events")
	write("internal/prod/client_test.go", `const u = "telemetry.gentlemanprogramming.com"`)
	write("odd/tasks/old.md", "telemetry.gentlemanprogramming.com")
	write("docs/audits/a.md", "telemetry.gentlemanprogramming.com")
	write("docs/upstream-absorption-ledger.md", "gentlemanprogramming.com/v1/events")
	write("openspec/changes/archive/x/spec.md", "/v1/runtime-events")
	write(".git/config", "telemetry.gentlemanprogramming.com")
	write("node_modules/p/index.js", "telemetry.gentlemanprogramming.com")
	write("internal/prod/blob.bin", "telemetry.gentlemanprogramming.com\x00")
	write("internal/prod/clean.go", "package prod\n")

	hits, err := upstreamEndpointHits(root)
	if err != nil {
		t.Fatalf("scan synthetic repository: %v", err)
	}
	want := []string{"internal/prod/client.go", "scripts/send.sh"}
	if strings.Join(hits, ",") != strings.Join(want, ",") {
		t.Fatalf("hits = %v, want %v", hits, want)
	}
}

func upstreamEndpointRepositoryRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get test working directory: %v", err)
	}
	for {
		info, err := os.Stat(filepath.Join(dir, "go.mod"))
		switch {
		case err == nil && !info.IsDir():
			return dir
		case err == nil:
			t.Fatalf("go.mod under %q is not a file", dir)
		case !errors.Is(err, fs.ErrNotExist):
			t.Fatalf("stat go.mod under %q: %v", dir, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("locate repository root from test working directory: go.mod not found")
		}
		dir = parent
	}
}

func upstreamEndpointAllowed(rel string) bool {
	for _, entry := range upstreamEndpointAllowlist {
		if strings.HasSuffix(entry, "/") {
			if strings.HasPrefix(rel, entry) {
				return true
			}
			continue
		}
		if rel == entry {
			return true
		}
	}
	return false
}

// upstreamEndpointHits returns the repository-relative paths of non-test,
// non-allowlisted text files that contain a retired endpoint.
func upstreamEndpointHits(root string) ([]string, error) {
	var hits []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := entry.Name()
		if entry.IsDir() {
			if name == ".git" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if upstreamEndpointAllowed(rel) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() > upstreamEndpointMaxFileBytes {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.IndexByte(data, 0) >= 0 {
			return nil
		}
		for _, endpoint := range retiredUpstreamEndpoints {
			if bytes.Contains(data, []byte(endpoint)) {
				hits = append(hits, rel)
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(hits)
	return hits, nil
}
