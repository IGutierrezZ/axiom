package idmigration_test

import (
	"regexp"
	"sort"
	"testing"
)

// domains es la tabla aprobada de códigos de dominio (Engram sdd/migrar-ids-rf/dominios):
// capacidad de openspec/specs -> código <DOM> del patrón RF-<DOM>-<NN>.
var domains = map[string]string{
	"antigravity-support":           "AGV",
	"autoskills":                    "ASK",
	"axiom-binary-ci-coverage":      "CIB",
	"axiom-distribution-identity":   "DIS",
	"axiom-sdd-cli-integration":     "CLI",
	"axiom-skills-index-governance": "SKI",
	"axiom-tui-branding":            "TUI",
	"axiom-updater-resilience":      "UPD",
	"axiom-user-state-and-env":      "UST",
	"azure-devops-adapter":          "ADO",
	"change-actor-identity":         "ACT",
	"change-branch-lifecycle":       "BRA",
	"dashboard-sdd-orchestration":   "DSO",
	"engram-protocol-injection":     "ENG",
	"integration-plugins":           "INT",
	"knowledge-profile-init":        "KNW",
	"living-documentation":          "LDO",
	"local-web-dashboard":           "WEB",
	"multi-project-hub":             "HUB",
	"multi-role-fan-out":            "FAN",
	"odd-cli-commands":              "OCL",
	"odd-living-document":           "ODO",
	"odd-living-spec-integration":   "OLS",
	"odd-sdd-promotion":             "OPR",
	"odd-ui-integration":            "OUI",
	"openspec-adoption":             "ADP",
	"orchestrator-commands-markers": "MRK",
	"organic-agent-trigger-rules":   "RUT",
	"persona-behavior-contract":     "PER",
	"planning-home-resolution":      "PLH",
	"project-technical-context":     "CTX",
	"requirement-identifiers":       "RID",
	"requirement-trace":             "TRC",
	"sdd-archive-lifecycle":         "ARC",
	"sdd-block-review-gates":        "GAT",
	"sdd-docs-publication":          "PUB",
	"sdd-independent-review":        "REV",
	"sdd-iso-evidence":              "ISO",
	"sdd-lane-kickoff":              "KOF",
	"sdd-orchestrator-assets":       "ORQ",
	"sdd-preflight-configuration":   "PRE",
	"sdd-research":                  "RES",
	"sdd-role-closure-handoff":      "RCL",
	"spec-baseline-integration":     "BAS",
	"structured-handoffs":           "HND",
	"tui-ui-parity":                 "UIP",
	"upstream-absorption-protocol":  "UPS",
	"visual-decoupling":             "VIS",
	"workspace-topology":            "TOP",
}

func TestDomainsTable(t *testing.T) {
	if len(domains) != 49 {
		t.Fatalf("la tabla aprobada tiene 49 códigos; hay %d", len(domains))
	}
	code := regexp.MustCompile(`^[A-Z]{3}$`)
	seen := map[string]string{}
	caps := make([]string, 0, len(domains))
	for c := range domains {
		caps = append(caps, c)
	}
	sort.Strings(caps)
	for _, capability := range caps {
		dom := domains[capability]
		if !code.MatchString(dom) {
			t.Errorf("%s: el código %q no son tres mayúsculas", capability, dom)
		}
		if other, dup := seen[dom]; dup {
			t.Errorf("el código %s está repetido en %s y %s", dom, other, capability)
		}
		seen[dom] = capability
		if id := "RF-" + dom + "-01"; !rfPattern.Valid(id) {
			t.Errorf("%s: %s no es válido bajo %s", capability, id, rfTemplate)
		}
	}
}

func TestDomainsSeriesNumbering(t *testing.T) {
	s, ok := rfPattern.Series("RF-ASK-01")
	if !ok {
		t.Fatal("RF-ASK-01 no forma serie")
	}
	for n, want := range map[int]string{1: "RF-ASK-01", 9: "RF-ASK-09", 10: "RF-ASK-10", 123: "RF-ASK-123"} {
		if got := rfPattern.Format(s, n); got != want {
			t.Errorf("Format(%d) = %s, quiero %s", n, got, want)
		}
	}
}
