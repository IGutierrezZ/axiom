package settings

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

const monoDoc = `workspace:
  name: "Demo"
  topology: "monorepo-embedded"
  specs_repository: "openspec"
roles:
  fullstack:
    name: "Fullstack"
    repositories:
      - path: "."
`

const multiDoc = `workspace:
  name: "Demo"
  topology: "multirepo"
  specs_repository: "repo-specs"
roles:
  backend:
    name: "Backend"
    repositories:
      - path: "repo-backend"
  frontend:
    name: "Frontend"
    repositories:
      - path: "repo-frontend"
`

// writeConfig escribe content como axiom.yaml en un directorio temporal.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "axiom.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readSnapshot(t *testing.T, content string) *Snapshot {
	t.Helper()
	snap, err := Read(writeConfig(t, content))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return snap
}

func entryByKey(snap *Snapshot, key string) (Entry, bool) {
	i := slices.IndexFunc(snap.Entries, func(e Entry) bool { return e.Key == key })
	if i < 0 {
		return Entry{}, false
	}
	return snap.Entries[i], true
}

func mustEntry(t *testing.T, snap *Snapshot, key string) Entry {
	t.Helper()
	e, ok := entryByKey(snap, key)
	if !ok {
		t.Fatalf("falta la entrada %q del catálogo", key)
	}
	return e
}

// RF-TOP-21: los valores permitidos salen de los accesores del paquete dueño.
func TestCatalogAllowedComesFromTheSchemaAccessors(t *testing.T) {
	snap := readSnapshot(t, multiDoc+"repositories:\n  - path: repo-backend\n")
	cases := map[string][]string{
		"git.promotion":                        workspace.PromotionValues(),
		"workflow.lane":                        stringsOf(workspace.WorkflowLanes()),
		"sdd.explainer_html":                   stringsOf(workspace.ExplainerHTMLModes()),
		"roles[backend].gate_policy":           stringsOf(multirole.GatePolicies()),
		"repositories[repo-backend].promotion": workspace.PromotionValues(),
	}
	for key, want := range cases {
		if got := mustEntry(t, snap, key).Allowed; !slices.Equal(got, want) {
			t.Errorf("%s: Allowed = %v, quiero %v", key, got, want)
		}
	}
}

// RF-TOP-21: coherencia en los dos sentidos entre el catálogo y los decodificadores:
// cada valor permitido lo acepta ParseConfig y un valor fuera de vocabulario no.
func TestCatalogVocabularyIsCoherentWithParseConfig(t *testing.T) {
	cases := []struct {
		name  string
		build func(value string) string
	}{
		{"git.promotion", func(v string) string { return monoDoc + "git:\n  promotion: " + v + "\n" }},
		{"workflow.lane", func(v string) string { return monoDoc + "workflow:\n  lane: " + v + "\n" }},
		{"sdd.explainer_html", func(v string) string { return monoDoc + "sdd:\n  explainer_html: " + v + "\n" }},
		{"repositories[].promotion", func(v string) string {
			return multiDoc + "repositories:\n  - path: repo-backend\n    promotion: " + v + "\n"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec, ok := findSpec(tc.name)
			if !ok {
				t.Fatalf("la plantilla %q no está en el catálogo", tc.name)
			}
			for _, value := range spec.vocabulary() {
				if _, err := workspace.ParseConfig([]byte(tc.build(value))); err != nil {
					t.Errorf("el valor permitido %q lo rechaza ParseConfig: %v", value, err)
				}
			}
			if _, err := workspace.ParseConfig([]byte(tc.build("zzz"))); err == nil {
				t.Error("ParseConfig debería rechazar zzz")
			}
		})
	}
	// gate_policy lo valida el alta (hub), no ParseConfig: aquí solo se comprueba
	// que el catálogo coincide con el vocabulario de multirole.
	spec, _ := findSpec("roles[].gate_policy")
	if !slices.Equal(spec.vocabulary(), stringsOf(multirole.GatePolicies())) {
		t.Errorf("gate_policy: %v", spec.vocabulary())
	}
}

// RF-TOP-21: filtros por topología.
func TestCatalogFiltersByTopology(t *testing.T) {
	isElementKey := func(k string) bool {
		return strings.HasPrefix(k, "repositories[") || strings.HasPrefix(k, "acceptance_tests.") || strings.HasPrefix(k, "roles[")
	}
	embedded := []struct{ name, doc string }{
		{"monorepo", monoDoc + "repositories:\n  - path: .\n"},
		{"topología ilegible", "workspace:\n  name: x\n  topology: [no, es, texto]\nroles:\n  a:\n    repositories:\n      - path: .\n"},
		{"sin topología", "workspace:\n  name: x\n"},
	}
	for _, tc := range embedded {
		t.Run(tc.name, func(t *testing.T) {
			snap := readSnapshot(t, tc.doc)
			for _, e := range snap.Entries {
				if isElementKey(e.Key) {
					t.Errorf("la topología embebida no debe incluir %q", e.Key)
				}
			}
			for _, key := range []string{"workspace.name", "git.promotion", "workflow.lane", "delivery.split_hint_lines", "sdd.iso", "sdd.id_pattern", "sdd.explainer_html"} {
				mustEntry(t, snap, key)
			}
		})
	}
	snap := readSnapshot(t, multiDoc+"repositories:\n  - path: repo-backend\n")
	for _, key := range []string{
		"repositories[repo-backend].integration_branch", "repositories[repo-backend].branch_prefix", "repositories[repo-backend].promotion",
		"repositories[repo-frontend].promotion", "acceptance_tests.enabled", "acceptance_tests.environments",
		"roles[backend].gate_policy", "roles[frontend].gate_policy",
	} {
		mustEntry(t, snap, key)
	}
	if !slices.Contains(mustEntry(t, snap, "roles[backend].gate_policy").Allowed, "optional") {
		t.Error("gate_policy debe admitir optional")
	}
}

// RF-TOP-21: un proyecto ISO no ofrece `odd`.
func TestCatalogISOHidesODDLane(t *testing.T) {
	snap := readSnapshot(t, monoDoc+"workflow:\n  lane: sdd\nsdd:\n  iso: true\n")
	if !snap.ISO {
		t.Fatal("ISO debería ser true")
	}
	if got, want := mustEntry(t, snap, "workflow.lane").Allowed, []string{"sdd", "ask"}; !slices.Equal(got, want) {
		t.Errorf("lane ISO = %v, quiero %v", got, want)
	}
	plain := readSnapshot(t, monoDoc)
	if got := mustEntry(t, plain, "workflow.lane").Allowed; !slices.Contains(got, "odd") {
		t.Errorf("sin ISO, lane = %v debe incluir odd", got)
	}
}

// RF-TOP-21: claves de solo lectura y exclusiones.
func TestCatalogReadOnlyAndExclusions(t *testing.T) {
	doc := `workspace:
  name: "Demo"
  topology: "monorepo-embedded"
  specs_repository: "openspec"
  root: "."
  skill_roots:
    - "skills"
roles:
  fullstack:
    name: "Fullstack"
    repositories:
      - path: "."
technical_context:
  entry: docs/ctx.md
governance:
  language: es
  shared_memory: engram
integrations:
  token: nunca-visible
`
	snap := readSnapshot(t, doc)
	for _, key := range []string{"workspace.topology", "workspace.specs_repository", "workspace.root", "workspace.skill_roots", "technical_context", "governance.language", "governance.shared_memory"} {
		e := mustEntry(t, snap, key)
		if e.Editable || e.ReadOnlyReason == "" {
			t.Errorf("%s debe ser de solo lectura con motivo: %+v", key, e)
		}
	}
	for _, e := range snap.Entries {
		for _, banned := range []string{"artifact_store", "rdd", "commit_review", "docs_publication"} {
			if strings.Contains(e.Key, banned) {
				t.Errorf("el catálogo no debe contener %q (%s)", banned, e.Key)
			}
		}
		if v, ok := e.Value.(string); ok && strings.Contains(v, "nunca-visible") {
			t.Errorf("el valor de %q expone el bloque integrations", e.Key)
		}
	}
	if v := mustEntry(t, snap, "technical_context").Value; !strings.Contains(v.(string), "docs/ctx.md") {
		t.Errorf("technical_context debe serializarse entero: %v", v)
	}
	if !mustEntry(t, snap, "git.promotion").Editable {
		t.Error("git.promotion debe ser editable")
	}
}

// RF-TOP-21: el patrón por defecto sale de reqid y los efectos son explícitos.
func TestCatalogDefaultsAndEffects(t *testing.T) {
	snap := readSnapshot(t, monoDoc)
	if got, want := mustEntry(t, snap, "sdd.id_pattern").Default, reqid.Default().Template(); got != want {
		t.Errorf("id_pattern por defecto = %q, quiero %q", got, want)
	}
	if got := mustEntry(t, snap, "git.integration_branch").Default; got != workspace.DefaultIntegrationBranch {
		t.Errorf("integration_branch por defecto = %q", got)
	}
	if got := mustEntry(t, snap, "workflow.lane").Default; got != string(workspace.DefaultWorkflowLane) {
		t.Errorf("lane por defecto = %q", got)
	}
	if e := mustEntry(t, snap, "sdd.iso").Effect; !strings.Contains(e, "verify es obligatorio") {
		t.Errorf("efecto de sdd.iso = %q", e)
	}
	if e := mustEntry(t, snap, "sdd.id_pattern").Effect; !strings.Contains(e, "no se renombran") {
		t.Errorf("efecto de sdd.id_pattern = %q", e)
	}
	for _, e := range snap.Entries {
		if e.Editable && !strings.Contains(e.Effect, onlyNewChanges) {
			t.Errorf("%s no lleva el aviso «solo afecta a cambios nuevos»", e.Key)
		}
	}
}

// RF-TOP-21: el selector va hasta el último `].`, de modo que un `]` en la ruta no
// rompe el análisis.
func TestCatalogParseKeyKeepsBracketsInSelector(t *testing.T) {
	got := parseKey("repositories[dir[1]/repo].promotion")
	if got.selector != "dir[1]/repo" || got.field != "promotion" || got.template != "repositories[].promotion" || got.scope != scopeRepo {
		t.Errorf("parseKey = %+v", got)
	}
	if got := parseKey("roles[web].gate_policy"); got.selector != "web" || got.template != "roles[].gate_policy" {
		t.Errorf("parseKey roles = %+v", got)
	}
	if got := parseKey("git.promotion"); got.scope != scopeBlock || got.template != "git.promotion" {
		t.Errorf("parseKey bloque = %+v", got)
	}
	if got := normalizeRepoPath(`.\repo-a\..\repo-b`); got != "repo-b" {
		t.Errorf("normalizeRepoPath = %q", got)
	}
}

// RF-TOP-21: las filas branch_template se filtran por topología.
func TestCatalogBranchTemplateRowsFilterByTopology(t *testing.T) {
	mono := readSnapshot(t, monoDoc)
	mustEntry(t, mono, "git.branch_template")
	for _, e := range mono.Entries {
		if strings.HasSuffix(e.Key, "].branch_template") {
			t.Errorf("la topología embebida no debe incluir %q", e.Key)
		}
	}
	multi := readSnapshot(t, multiDoc+"repositories:\n  - path: repo-backend\n")
	for _, key := range []string{"git.branch_template", "repositories[repo-backend].branch_template", "repositories[repo-frontend].branch_template", "roles[backend].branch_template", "roles[frontend].branch_template"} {
		if e := mustEntry(t, multi, key); e.Type != TypeString || len(e.Allowed) != 0 {
			t.Errorf("%s debe ser de tipo cadena sin vocabulario: %+v", key, e)
		}
	}
}
