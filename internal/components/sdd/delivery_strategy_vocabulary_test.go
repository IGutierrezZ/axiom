package sdd

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// SDD delivery strategy has a producer and a consumer. The producer is the
// session-preflight label->canonical mapping; the consumer is every phase skill
// and orchestrator branch that reads `delivery_strategy`. Claude and OpenCode
// receive the same canonical producer at render time; raw assets are templates.
// Preserved OpenCode prompts receive it through marker-owned migration.
//
// Nothing asserted the halves agreed, so the preflight emitted `ask-always`,
// `single-pr-default`, `force-chained`, and `auto-forecast` into a consumer
// whose whole declared domain was `ask-on-risk | auto-chain | single-pr |
// exception-ok` (community report by @MarcosArispe). Every consumer branch list
// is a bare four-way match, so an out-of-domain value matched nothing and the
// orchestrator model picked a delivery strategy freely.
//
// Both halves below are derived from shipped artifacts, never from a list
// restated here: the domain from the phase skills that declare it as their
// input contract, the producer from the preflight UI label list plus the
// mapping block in the installed document. Walking the embedded assets rejects
// a second producer. A renamed label, a fifth canonical value, or a typo on
// either side fails here instead of drifting silently.

// deliveryDomainDeclaration matches a phase skill's declared input domain for
// delivery strategy: one backticked, pipe-separated list led by the default
// value. Keeping the pattern anchored on that shape is what lets the guard read
// the domain out of the shipped skill instead of hardcoding it.
var deliveryDomainDeclaration = regexp.MustCompile("`(ask-on-risk(?: \\| [a-z][a-z0-9-]*)+)`")

// preflightPRGroupLabels matches the user-facing PR option list the preflight
// renders, e.g. "- **PR strategy** (`pr-strategy`): Ask me, Single PR, or Auto."
var preflightPRGroupLabels = regexp.MustCompile(`(?m)^\s*- \*\*PR strategy\*\* \(` + "`pr-strategy`" + `\): (.+?)\.[ \t]*\r?$`)

var backtickSpan = regexp.MustCompile("`([^`]+)`")

// canonicalValueShape matches the shape every delivery strategy value has:
// lowercase, hyphen-separated, nothing else. It exists to skip the prompt
// variables (`delivery_strategy`) and qualified labels (`size:exception`) that
// legitimately share those sentences, while still catching a retired or
// mistyped strategy such as `single-pr-default`.
var canonicalValueShape = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

// deliveryStrategySkillContracts are the phase skills that declare the domain as
// their own input contract. They are the consumers that reject an unknown value,
// which makes them the authority on what the producer is allowed to emit.
var deliveryStrategySkillContracts = []string{
	"skills/sdd-apply/SKILL.md",
	"skills/sdd-tasks/SKILL.md",
}

// canonicalDeliveryStrategyDomain reads the domain out of the shipped phase
// skills and fails when they disagree with each other.
func canonicalDeliveryStrategyDomain(t *testing.T) []string {
	t.Helper()

	var domain []string
	var source string
	for _, path := range deliveryStrategySkillContracts {
		matches := deliveryDomainDeclaration.FindAllStringSubmatch(assets.MustRead(path), -1)
		if len(matches) != 1 {
			t.Fatalf(
				"%s must declare the delivery strategy domain exactly once as a backticked `a | b | ...` list; found %d declarations",
				path,
				len(matches),
			)
		}

		values := splitDeclaredDomain(matches[0][1])
		if domain == nil {
			domain, source = values, path
			continue
		}
		if strings.Join(values, "|") != strings.Join(domain, "|") {
			t.Fatalf(
				"delivery strategy domain disagrees between shipped skills: %s declares %v, %s declares %v",
				source,
				domain,
				path,
				values,
			)
		}
	}

	if len(domain) == 0 {
		t.Fatal("no delivery strategy domain could be derived from the shipped phase skills")
	}
	return domain
}

func splitDeclaredDomain(declaration string) []string {
	values := make([]string, 0, 4)
	for _, value := range strings.Split(declaration, "|") {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

// preflightMappingSources reads the actual installed Claude authority and rejects
// any raw asset that reintroduces a separately authored mapping producer.
func preflightMappingSources(t *testing.T) map[string]string {
	t.Helper()

	home := t.TempDir()
	if _, err := writeClaudeLazySDDWorkflow(home, claudeAdapter()); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".claude", "skills", "_shared", "sdd-orchestrator-workflow.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if err := validateSDDSessionPreflightProjection(content, testSDDSessionPreflightEntryAnchor, "AskUserQuestion"); err != nil {
		t.Fatal(err)
	}
	sources := map[string]string{"installed Claude": content}
	err = fs.WalkDir(assets.FS, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || !strings.HasSuffix(path, ".md") {
			return walkErr
		}
		content := assets.MustRead(path)
		for _, producer := range []string{"Map answers to canonical values", "Canonical mappings:", "Ask me ->", sddSessionPreflightMarker} {
			if strings.Contains(content, producer) {
				t.Errorf("raw asset %s owns a second preflight producer: %q", path, producer)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded assets: %v", err)
	}
	return sources
}

// mappedCanonicalValue reads the canonical value the mapping block assigns to
// one user-facing label.
func mappedCanonicalValue(mapping, label string) (string, bool) {
	marker := label + " -> `"
	start := strings.Index(mapping, marker)
	if start < 0 {
		return "", false
	}
	rest := mapping[start+len(marker):]
	end := strings.Index(rest, "`")
	if end < 0 {
		return "", false
	}
	return rest[:end], true
}

// preflightBlockOf isolates the marker-owned session preflight from the rest of
// the orchestrator document it is installed in.
func preflightBlockOf(t *testing.T, path, content string) string {
	t.Helper()

	start := strings.Index(content, sddSessionPreflightMarker)
	if start < 0 {
		t.Fatalf("%s lost its session preflight block", path)
	}
	block := content[start:]
	if end := strings.Index(block, sddSessionPreflightEnd); end >= 0 {
		block = block[:end]
	}
	return block
}

// canonicalMappingBlock isolates the mapping from the surrounding prose so a
// label that merely appears in the UI section is never mistaken for a mapping.
func canonicalMappingBlock(t *testing.T, path, content string) string {
	t.Helper()

	start := strings.Index(content, "Canonical mappings:")
	if start < 0 {
		t.Fatalf("%s lost its canonical mapping block", path)
	}
	block := content[start:]
	if end := strings.Index(block, sddSessionPreflightEnd); end >= 0 {
		block = block[:end]
	}
	return block
}

// The preflight no longer produces a delivery strategy: it offers no PR group, no
// option list and no label -> value mapping, and it never states a review
// threshold. The only producer left is the hook (and the OpenCode plugin), whose
// block always says `Delivery strategy: single-pr`.
func TestSDDPreflightOffersNoDeliveryStrategyGroupOrMapping(t *testing.T) {
	for path, content := range preflightMappingSources(t) {
		if match := preflightPRGroupLabels.FindStringSubmatch(content); match != nil {
			t.Errorf("%s still offers a PR strategy group: %q", path, match[0])
		}
		mapping := canonicalMappingBlock(t, path, content)
		for _, label := range []string{"Ask me", "Single PR", "Auto", "Chained"} {
			if value, mapped := mappedCanonicalValue(mapping, label); mapped {
				t.Errorf("%s still maps the PR option %q to %q", path, label, value)
			}
		}
		block := preflightBlockOf(t, path, content)
		for _, residue := range []string{"pr-strategy", "PR strategy", "ask-on-risk", "auto-chain", "exception-ok", "Review policy", "400"} {
			if strings.Contains(block, residue) {
				t.Errorf("%s still carries the retired preflight text %q", path, residue)
			}
		}
	}
}

// The delivery the preflight fixes must stay inside the domain the phase skills
// declare, or the consumer branch list would have no arm for it.
func TestSDDPreflightFixedDeliveryStaysInsideConsumerDomain(t *testing.T) {
	domain := canonicalDeliveryStrategyDomain(t)
	allowed := map[string]bool{}
	for _, value := range domain {
		allowed[value] = true
	}

	fixed := regexp.MustCompile("Delivery is never asked: it is always `([a-z][a-z0-9-]*)`")
	checked := 0
	for path, content := range preflightMappingSources(t) {
		match := fixed.FindStringSubmatch(preflightBlockOf(t, path, content))
		if match == nil {
			t.Errorf("%s does not state the delivery strategy it fixes", path)
			continue
		}
		checked++
		if !allowed[match[1]] {
			t.Errorf("%s fixes the delivery strategy %q, which is outside the domain %v declared by %v", path, match[1], domain, deliveryStrategySkillContracts)
		}
	}
	if checked == 0 {
		t.Fatal("no preflight states its fixed delivery strategy; the guard has lost its subject")
	}
}

// reviewWorkloadGuardSections returns the Review Workload Guard body of every
// shipped orchestrator, discovered by walking the embedded tree.
func reviewWorkloadGuardSections(t *testing.T) map[string]string {
	t.Helper()

	const heading = "### Review Workload Guard"
	sections := map[string]string{}
	err := fs.WalkDir(assets.FS, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || !strings.HasSuffix(path, ".md") {
			return walkErr
		}
		content := assets.MustRead(path)
		if strings.HasSuffix(path, "/sdd-orchestrator.md") {
			content = substituteSharedOrchestratorSections(content)
		}
		start := strings.Index(content, heading)
		if start < 0 {
			return nil
		}
		body := content[start+len(heading):]
		for _, next := range []string{"\n### ", "\n## ", "\n# ", "\n<!-- gentle-ai:"} {
			if end := strings.Index(body, next); end >= 0 {
				body = body[:end]
			}
		}
		sections[path] = body
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded assets: %v", err)
	}
	if len(sections) == 0 {
		t.Fatal("no shipped orchestrator carries a Review Workload Guard; the guard has lost its subject")
	}
	return sections
}

// A branch list that silently drops an arm is the same defect seen from the
// other side: a value the producer can emit that the consumer cannot match. The
// domain is derived, so a typo in any arm shows up here as a missing arm.
func TestSDDReviewWorkloadGuardsCoverTheWholeDeliveryStrategyDomain(t *testing.T) {
	domain := canonicalDeliveryStrategyDomain(t)

	for path, section := range reviewWorkloadGuardSections(t) {
		var missing []string
		for _, value := range domain {
			if !strings.Contains(section, "`"+value+"`") {
				missing = append(missing, value)
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			t.Errorf(
				"%s Review Workload Guard has no arm for delivery strategy %v; an unmatched value lets the orchestrator choose freely",
				path,
				missing,
			)
		}
	}
}

// The vocabularies agreeing today does not stop a future typo from reintroducing
// silent fallthrough. Every guard must say what happens to a value it does not
// recognise instead of leaving the branch list open.
func TestSDDReviewWorkloadGuardsRejectUnrecognisedDeliveryStrategy(t *testing.T) {
	for path, section := range reviewWorkloadGuardSections(t) {
		if !strings.Contains(section, "Any other `delivery_strategy` value is invalid") {
			t.Errorf(
				"%s Review Workload Guard does not say what to do with an unrecognised `delivery_strategy`; "+
					"a bare branch list falls through silently",
				path,
			)
		}
	}
}

// A preserved OpenCode prompt with a legacy owned block must replace that block,
// so retired delivery mappings cannot survive a sync.
func TestInjectOpenCodeMigratesRetiredDeliveryStrategyMapping(t *testing.T) {
	home := t.TempDir()
	mockNoPackageManager(t)

	settingsPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(settings dir) error = %v", err)
	}

	stalePrompt := legacySDDSessionPreflightMarker + "\n" +
		"3. PRs: Ask me, Single PR, Auto.\n" +
		"Ask me -> `ask-always`; Single PR -> `single-pr-default`; Auto -> `auto-forecast`\n" +
		legacySDDSessionPreflightEnd
	if !strings.Contains(stalePrompt, "Ask me -> `ask-always`") {
		t.Fatal("test seed did not reproduce the retired mapping; the literal shape changed")
	}

	seed := `{
  "agent": {
    "gentle-orchestrator": {
      "mode": "primary",
      "prompt": ` + strconv.Quote("# Custom prompt\n"+stalePrompt) + `
    }
  }
}`
	if err := os.WriteFile(settingsPath, []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(opencode.json) error = %v", err)
	}

	if _, err := Inject(home, opencodeAdapter(), model.SDDModeMulti, InjectOptions{
		PreserveOpenCodeOrchestratorPrompt: true,
	}); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	text := preservedOrchestratorPrompt(t, settingsPath)

	for _, retired := range []string{"ask-always", "single-pr-default", "force-chained", "auto-forecast"} {
		if strings.Contains(text, retired) {
			t.Errorf("opencode.json kept retired delivery strategy value %q after sync", retired)
		}
	}
	if strings.Contains(text, "Ask me ->") || !strings.Contains(text, "Delivery is never asked: it is always `single-pr`") {
		t.Error("opencode.json did not receive the preflight without a delivery strategy mapping")
	}
	if !strings.Contains(text, "# Custom prompt") {
		t.Error("migration discarded the user's own prompt content")
	}
}

// The four-option preflight shipped alongside the already-corrected
// `ask-on-risk` mapping, so a prompt that still offers `Chained` satisfies every
// freshness clause that existed before this change — including the one added
// when the retired mapping was fixed. Without a clause that fails on the retired
// option, an installed config would keep the four-option menu forever and the
// asset-only removal would be reverted on the operator's next sync.
func TestInjectOpenCodeMigratesRetiredChainedPRPreflightOption(t *testing.T) {
	home := t.TempDir()
	mockNoPackageManager(t)

	settingsPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(settings dir) error = %v", err)
	}

	// Rebuild the exact prompt the previous release injected by reversing this
	// change on the current literal, so the seed tracks the literal instead of
	// freezing a copy of it that could drift.
	stalePrompt := legacySDDSessionPreflightMarker + "\n" +
		"3. PRs: Ask me, Single PR, Chained, Auto.\n" +
		"Chained -> `auto-chain`\n" +
		"Chained and Auto both resolve to `auto-chain` because `delivery_strategy` is only consulted once the tasks forecast flags review-budget risk.\n" +
		legacySDDSessionPreflightEnd

	for _, seeded := range []string{
		"3. PRs: Ask me, Single PR, Chained, Auto.",
		"Chained -> `auto-chain`",
		"Chained and Auto both resolve to `auto-chain`",
	} {
		if !strings.Contains(stalePrompt, seeded) {
			t.Fatalf("test seed did not reproduce the retired four-option preflight fragment %q; the literal shape changed", seeded)
		}
	}

	seed := `{
  "agent": {
    "gentle-orchestrator": {
      "mode": "primary",
      "prompt": ` + strconv.Quote("# Custom prompt\n"+stalePrompt) + `
    }
  }
}`
	if err := os.WriteFile(settingsPath, []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(opencode.json) error = %v", err)
	}

	if _, err := Inject(home, opencodeAdapter(), model.SDDModeMulti, InjectOptions{
		PreserveOpenCodeOrchestratorPrompt: true,
	}); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	text := preservedOrchestratorPrompt(t, settingsPath)

	for _, residue := range []string{
		"3. PRs: Ask me, Single PR, Chained, Auto.",
		"Chained -> `auto-chain`",
		"Chained and Auto both resolve to `auto-chain`",
	} {
		if strings.Contains(text, residue) {
			t.Errorf("opencode.json kept retired preflight fragment %q after sync", residue)
		}
	}
	if strings.Contains(text, "PR strategy") || strings.Contains(text, "auto-chain") || strings.Contains(text, "Auto ->") {
		t.Error("opencode.json kept the PR strategy group after the migration; the preflight no longer offers it")
	}
	if !strings.Contains(text, "Delivery is never asked: it is always `single-pr`") {
		t.Error("opencode.json did not receive the fixed delivery text")
	}
	if !strings.Contains(text, "# Custom prompt") {
		t.Error("migration discarded the user's own prompt content")
	}
	if count := strings.Count(text, sddSessionPreflightMarker); count != 1 {
		t.Errorf("migrated prompt carries %d canonical preflight blocks; the retired menu must be replaced, not appended to", count)
	}

	// The freshness clause that fires this migration must also stop firing once
	// it has run, or every later sync would rewrite the operator's prompt again.
	if _, err := Inject(home, opencodeAdapter(), model.SDDModeMulti, InjectOptions{
		PreserveOpenCodeOrchestratorPrompt: true,
	}); err != nil {
		t.Fatalf("second Inject() error = %v", err)
	}
	if resynced := preservedOrchestratorPrompt(t, settingsPath); resynced != text {
		t.Error("re-syncing the migrated prompt changed it again; the migration is not idempotent")
	}
}

// preservedOrchestratorPrompt reads the preserved prompt back through JSON
// rather than matching raw bytes: encoding/json escapes the ">" in "->", so a
// substring match on the file would silently never fire.
func preservedOrchestratorPrompt(t *testing.T, settingsPath string) string {
	t.Helper()

	settingsBytes, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(opencode.json) error = %v", err)
	}
	var settings struct {
		Agent map[string]struct {
			Prompt string `json:"prompt"`
		} `json:"agent"`
	}
	if err := json.Unmarshal(settingsBytes, &settings); err != nil {
		t.Fatalf("Unmarshal(opencode.json) error = %v", err)
	}
	prompt := settings.Agent["axiom-orchestrator"].Prompt
	if prompt == "" {
		prompt = settings.Agent["gentle-orchestrator"].Prompt
	}
	if prompt == "" {
		t.Fatal("opencode.json lost the preserved orchestrator prompt")
	}
	return prompt
}
