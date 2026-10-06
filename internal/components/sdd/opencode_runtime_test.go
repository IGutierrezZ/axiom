package sdd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	agent "github.com/IGutierrezZ/axiom/v3/internal/agents/opencode"
	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/opencode"
)

// Existing config tests model V1, never an ambient executable.
func init() {
	opencode.VersionRunnerOverride = func(context.Context, opencode.Command) (opencode.CommandOutput, error) {
		return opencode.CommandOutput{Stdout: []byte("1.18.30")}, nil
	}
}
func TestOpenCodePluginMajorSelection(t *testing.T) {
	old := opencode.VersionRunnerOverride
	t.Cleanup(func() { opencode.VersionRunnerOverride = old })
	for _, version := range []string{"2.0.4", "unknown"} {
		t.Run(version, func(t *testing.T) {
			opencode.VersionRunnerOverride = func(context.Context, opencode.Command) (opencode.CommandOutput, error) {
				return opencode.CommandOutput{Stdout: []byte(version)}, nil
			}
			home := t.TempDir()
			a := agent.NewAdapter()
			assetDir, err := openCodePluginAssetDirectory(a.Agent())
			var result InjectionResult
			if err == nil {
				result, err = installOpenCodePluginsDirectory(home, a, assetDir)
			}
			if version == "unknown" {
				if err == nil || len(result.Files) != 0 {
					t.Fatal("unknown runtime mutated plugins")
				}
				if _, err := os.Stat(a.GlobalConfigDir(home)); !os.IsNotExist(err) {
					t.Fatal("unknown runtime created directory")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			p := filepath.Join(a.GlobalConfigDir(home), "plugins", "skill-registry.ts")
			b, _ := os.ReadFile(p)
			if string(b) != assets.MustRead("opencode/plugins-v2/skill-registry.ts") {
				t.Fatal("V2 selected legacy plugin")
			}
			os.WriteFile(p, []byte("user-owned"), 0600)
			if _, err := RefreshInstalledOpenCodePlugins(home, a); err == nil || !strings.Contains(err.Error(), "preserved") {
				t.Fatal("custom V2 plugin overwritten")
			}
			b, _ = os.ReadFile(p)
			if string(b) != "user-owned" {
				t.Fatal("custom bytes lost")
			}
		})
	}
}

func withOpenCodeV2Runtime(t *testing.T) {
	t.Helper()
	old := opencode.VersionRunnerOverride
	t.Cleanup(func() { opencode.VersionRunnerOverride = old })
	opencode.VersionRunnerOverride = func(context.Context, opencode.Command) (opencode.CommandOutput, error) {
		return opencode.CommandOutput{Stdout: []byte("2.0.4")}, nil
	}
}

func TestOpenCodeManagedPluginAssetsSpawnAxiom(t *testing.T) {
	for _, path := range []string{
		"opencode/plugins/skill-registry.ts", "opencode/plugins-v2/skill-registry.ts",
		"opencode/plugins/opencode-review-transport.ts", "opencode/plugins-v2/opencode-review-transport.ts",
	} {
		content := assets.MustRead(path)
		if !strings.Contains(content, `"axiom"`) {
			t.Errorf("%s does not spawn axiom", path)
		}
		if strings.Contains(content, `"gentle-ai"`) {
			t.Errorf("%s still spawns the retired gentle-ai binary", path)
		}
		sum := sha256.Sum256([]byte(content))
		if isHistoricalOpenCodePlugin(filepath.Base(path), []byte(content)) {
			t.Errorf("%s digest %x is listed as historical", path, sum)
		}
	}
}

func TestHistoricalOpenCodePluginDigestTable(t *testing.T) {
	want := map[string]map[string]int{
		"skill-registry.ts":            {"v1": 5, "v2": 1},
		"opencode-review-transport.ts": {"v1": 7, "v2": 1},
	}
	seen := map[string]bool{}
	for name, generations := range want {
		for generation, count := range generations {
			digests := historicalOpenCodePluginDigests[name][generation]
			if len(digests) != count {
				t.Errorf("%s %s: %d digests, want %d", name, generation, len(digests), count)
			}
			for _, digest := range digests {
				if raw, err := hex.DecodeString(digest); err != nil || len(raw) != sha256.Size {
					t.Errorf("%s %s: malformed digest %q", name, generation, digest)
				}
				if seen[digest] {
					t.Errorf("%s %s: duplicate digest %s", name, generation, digest)
				}
				seen[digest] = true
			}
		}
	}
	if len(historicalOpenCodePluginDigests) != len(want) {
		t.Errorf("unexpected plugins in the digest table: %v", historicalOpenCodePluginDigests)
	}
}

func TestOpenCodeV2ReplacesKnownHistoricalPluginCopies(t *testing.T) {
	withOpenCodeV2Runtime(t)
	const historical = "// unmodified earlier managed copy that spawned gentle-ai\n"
	sum := sha256.Sum256([]byte(historical))
	digests := historicalOpenCodePluginDigests["skill-registry.ts"]
	digests["v2"] = append(digests["v2"], hex.EncodeToString(sum[:]))
	t.Cleanup(func() { digests["v2"] = digests["v2"][:len(digests["v2"])-1] })

	home := t.TempDir()
	a := agent.NewAdapter()
	assetDir, err := openCodePluginAssetDirectory(a.Agent())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := installOpenCodePluginsDirectory(home, a, assetDir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(a.GlobalConfigDir(home), "plugins", "skill-registry.ts")
	if err := os.WriteFile(path, []byte(historical), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RefreshInstalledOpenCodePlugins(home, a); err != nil {
		t.Fatalf("known historical copy rejected: %v", err)
	}
	if got, _ := os.ReadFile(path); string(got) != assets.MustRead("opencode/plugins-v2/skill-registry.ts") {
		t.Fatal("known historical copy was not replaced with the current asset")
	}

	// The same bytes with one edit are no longer a known managed copy.
	if err := os.WriteFile(path, []byte(historical+"// edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = RefreshInstalledOpenCodePlugins(home, a)
	if err == nil || !strings.Contains(err.Error(), "unverified ownership; custom bytes preserved") {
		t.Fatalf("edited copy error = %v", err)
	}
	if got, _ := os.ReadFile(path); string(got) != historical+"// edited\n" {
		t.Fatal("edited copy was overwritten")
	}
	if _, err := installOpenCodePluginsDirectory(home, a, assetDir); err == nil {
		t.Fatal("install replaced an edited copy")
	}
}
