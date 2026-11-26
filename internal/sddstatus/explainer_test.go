package sddstatus

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/explainer"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// sealExplainerChange sella el cambio en continuous con la decisión del HTML explicativo dada
// ("" no escribe la clave) y, opcionalmente, la instantánea ISO.
func sealExplainerChange(t *testing.T, changeRoot, name string, value kickoff.ExplainerHTML, snapshot *kickoff.SDDSnapshot) {
	t.Helper()
	_, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: name, SealedBy: "test",
		Config: kickoff.FlowConfig{
			FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionContinuous, ExecutionStyleSource: "explicit",
			HandoffPolicy: kickoff.HandoffNone,
			Roles:         []kickoff.KickoffRole{{Role: "fullstack", GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"}},
			ExplainerHTML: value,
		},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
		SDD:       snapshot,
	})
	if err != nil || !ok {
		t.Fatalf("sealExplainerChange: kickoff.Seal() = (ok=%v, err=%v)", ok, err)
	}
}

// explainerChange crea un cambio con proposal, una spec, design y tasks, sellado con value.
func explainerChange(t *testing.T, value kickoff.ExplainerHTML) (root, changeRoot string) {
	t.Helper()
	root = t.TempDir()
	changeRoot = seedReadyChange(t, root, "demo", "- [ ] 1.1 Trabajo\n")
	sealExplainerChange(t, changeRoot, "demo", value, nil)
	return root, changeRoot
}

func explainerHTMLWith(digest string) string {
	return `<html><head><meta name="axiom-source-digest" content="` + digest + `"></head><body>x</body></html>`
}

func writeExplainerHTML(t *testing.T, changeRoot string, kind explainer.Kind, content string) {
	t.Helper()
	write(t, filepath.Join(changeRoot, explainer.FileName(kind)), content)
}

func expectState(t *testing.T, label string, got *ExplainerArtifactProjection, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s: proyección nil, se esperaba %q", label, want)
	}
	if got.State != want {
		t.Fatalf("%s: state = %q (error %q), se esperaba %q", label, got.State, got.Error, want)
	}
}

func TestExplainerFreshnessLifecycle(t *testing.T) {
	t.Parallel()
	_, changeRoot := explainerChange(t, kickoff.ExplainerHTMLYes)

	// Sin HTML: missing con el digest esperado para que la fase lo copie.
	result := ExplainerFreshness(changeRoot, ArtifactStoreOpenSpec)
	if !result.Enabled || result.Sealed != kickoff.ExplainerHTMLYes || result.Error != "" {
		t.Fatalf("resultado = %+v", result)
	}
	expectState(t, "spec sin html", result.Spec, "missing")
	expectState(t, "design sin html", result.Design, "missing")
	specDigest := result.Spec.ExpectedDigest
	designDigest := result.Design.ExpectedDigest
	if !strings.HasPrefix(specDigest, "sha256:") || len(specDigest) != len("sha256:")+64 || specDigest == designDigest {
		t.Fatalf("digests esperados inesperados: %q %q", specDigest, designDigest)
	}
	if result.Spec.Path != filepath.Join(changeRoot, "resumen-spec.html") || result.Design.Path != filepath.Join(changeRoot, "resumen-design.html") {
		t.Fatalf("rutas = %q %q", result.Spec.Path, result.Design.Path)
	}

	// Con el digest correcto: fresh.
	writeExplainerHTML(t, changeRoot, explainer.KindSpec, explainerHTMLWith(specDigest))
	writeExplainerHTML(t, changeRoot, explainer.KindDesign, explainerHTMLWith(strings.ToUpper(designDigest)))
	result = ExplainerFreshness(changeRoot, ArtifactStoreOpenSpec)
	expectState(t, "spec fresh", result.Spec, "fresh")
	// El valor del meta se normaliza a minúsculas, así que el digest en mayúsculas también es fresh.
	expectState(t, "design fresh", result.Design, "fresh")

	// Editar el md de la spec desactualiza solo el HTML de la spec.
	write(t, filepath.Join(changeRoot, "specs", "auth", "spec.md"), "### Requirement: Auth\n#### Scenario: Cambiado\n")
	result = ExplainerFreshness(changeRoot, ArtifactStoreOpenSpec)
	expectState(t, "spec stale", result.Spec, "stale")
	expectState(t, "design intacto", result.Design, "fresh")
	if result.Spec.ExpectedDigest == specDigest {
		t.Fatal("el digest esperado debe cambiar con el md")
	}

	// Borrar el HTML: missing.
	if err := os.Remove(filepath.Join(changeRoot, "resumen-spec.html")); err != nil {
		t.Fatal(err)
	}
	expectState(t, "spec missing", ExplainerFreshness(changeRoot, ArtifactStoreOpenSpec).Spec, "missing")
}

func TestExplainerFreshnessEmptyOrMalformedMetaIsStale(t *testing.T) {
	t.Parallel()
	_, changeRoot := explainerChange(t, kickoff.ExplainerHTMLYes)
	for name, html := range map[string]string{
		"meta vacía":       explainerHTMLWith(""),
		"meta mal formada": explainerHTMLWith("sha256:xyz"),
		"sin meta":         "<html><body>x</body></html>",
		"otro digest":      explainerHTMLWith("sha256:" + strings.Repeat("0", 64)),
	} {
		writeExplainerHTML(t, changeRoot, explainer.KindSpec, html)
		expectState(t, name, ExplainerFreshness(changeRoot, ArtifactStoreOpenSpec).Spec, "stale")
	}
}

// Varios spec.md por dominio cuentan todos, y el alta o la baja de un dominio cambia el digest.
func TestExplainerFreshnessCoversEverySpecDomain(t *testing.T) {
	t.Parallel()
	_, changeRoot := explainerChange(t, kickoff.ExplainerHTMLYes)
	base := ExplainerFreshness(changeRoot, ArtifactStoreOpenSpec).Spec.ExpectedDigest

	billing := filepath.Join(changeRoot, "specs", "billing", "spec.md")
	write(t, billing, "### Requirement: Cobro\n")
	withBilling := ExplainerFreshness(changeRoot, ArtifactStoreOpenSpec).Spec.ExpectedDigest
	if withBilling == base {
		t.Fatal("el alta de un dominio debe cambiar el digest")
	}
	writeExplainerHTML(t, changeRoot, explainer.KindSpec, explainerHTMLWith(withBilling))
	expectState(t, "con dos dominios", ExplainerFreshness(changeRoot, ArtifactStoreOpenSpec).Spec, "fresh")

	write(t, billing, "### Requirement: Cobro cambiado\n")
	expectState(t, "edición del segundo dominio", ExplainerFreshness(changeRoot, ArtifactStoreOpenSpec).Spec, "stale")

	if err := os.RemoveAll(filepath.Dir(billing)); err != nil {
		t.Fatal(err)
	}
	if got := ExplainerFreshness(changeRoot, ArtifactStoreOpenSpec).Spec.ExpectedDigest; got != base {
		t.Fatalf("la baja del dominio vuelve al digest base: %q != %q", got, base)
	}
}

func TestExplainerFreshnessOmitsKindsWithoutSourceMD(t *testing.T) {
	t.Parallel()
	_, changeRoot := explainerChange(t, kickoff.ExplainerHTMLYes)
	if err := os.Remove(filepath.Join(changeRoot, "design.md")); err != nil {
		t.Fatal(err)
	}
	// Un HTML huérfano no hace aparecer el tipo.
	writeExplainerHTML(t, changeRoot, explainer.KindDesign, explainerHTMLWith("sha256:"+strings.Repeat("a", 64)))
	result := ExplainerFreshness(changeRoot, ArtifactStoreOpenSpec)
	if result.Design != nil {
		t.Fatalf("design sin md de origen debe omitirse: %+v", result.Design)
	}
	expectState(t, "spec", result.Spec, "missing")

	if err := os.RemoveAll(filepath.Join(changeRoot, "specs")); err != nil {
		t.Fatal(err)
	}
	result = ExplainerFreshness(changeRoot, ArtifactStoreOpenSpec)
	if result.Spec != nil || result.Design != nil || !result.Enabled {
		t.Fatalf("sin ningún md no hay tipos: %+v", result)
	}
}

func TestExplainerFreshnessReportsUnreadableHTMLWithoutState(t *testing.T) {
	t.Parallel()
	_, changeRoot := explainerChange(t, kickoff.ExplainerHTMLYes)

	write(t, filepath.Join(changeRoot, explainer.FileName(explainer.KindSpec)), strings.Repeat("x", explainer.MaxBytes+1))
	spec := ExplainerFreshness(changeRoot, ArtifactStoreOpenSpec).Spec
	if spec == nil || spec.State != "" || !strings.Contains(spec.Error, "5 MiB") || spec.ExpectedDigest == "" {
		t.Fatalf("HTML de MaxBytes+1: %+v", spec)
	}

	// Un directorio con el nombre del HTML no es un fichero regular.
	if err := os.MkdirAll(filepath.Join(changeRoot, explainer.FileName(explainer.KindDesign)), 0o755); err != nil {
		t.Fatal(err)
	}
	design := ExplainerFreshness(changeRoot, ArtifactStoreOpenSpec).Design
	if design == nil || design.State != "" || design.Error == "" {
		t.Fatalf("directorio con el nombre del HTML: %+v", design)
	}
}

func TestExplainerFreshnessCorruptKickoffReportsError(t *testing.T) {
	t.Parallel()
	_, changeRoot := explainerChange(t, kickoff.ExplainerHTMLYes)
	write(t, filepath.Join(changeRoot, "kickoff.yaml"), "schema: [roto\n")
	result := ExplainerFreshness(changeRoot, ArtifactStoreOpenSpec)
	if result.Error == "" || result.Enabled || result.Spec != nil || result.Design != nil {
		t.Fatalf("un kickoff corrupto es un error informativo: %+v", result)
	}
}

// Sin activación (no, sin clave, sin kickoff) o con un almacén sin ficheros no hay frescura aunque
// el HTML exista.
func TestExplainerFreshnessRequiresSealedYesAndAFileStore(t *testing.T) {
	t.Parallel()
	html := explainerHTMLWith("sha256:" + strings.Repeat("a", 64))
	for _, tc := range []struct {
		name   string
		value  kickoff.ExplainerHTML
		seal   bool
		store  ArtifactStore
		sealed kickoff.ExplainerHTML
	}{
		{"no", kickoff.ExplainerHTMLNo, true, ArtifactStoreOpenSpec, kickoff.ExplainerHTMLNo},
		{"sin campo", "", true, ArtifactStoreOpenSpec, kickoff.ExplainerHTMLNo},
		{"sin kickoff", "", false, ArtifactStoreOpenSpec, kickoff.ExplainerHTMLNo},
		{"yes con engram", kickoff.ExplainerHTMLYes, true, ArtifactStoreEngram, kickoff.ExplainerHTMLYes},
		{"yes con none", kickoff.ExplainerHTMLYes, true, ArtifactStoreNone, kickoff.ExplainerHTMLYes},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			changeRoot := seedReadyChange(t, root, "demo", "- [ ] 1.1 Trabajo\n")
			if tc.seal {
				sealExplainerChange(t, changeRoot, "demo", tc.value, nil)
			}
			writeExplainerHTML(t, changeRoot, explainer.KindSpec, html)
			result := ExplainerFreshness(changeRoot, tc.store)
			if result.Enabled || result.Spec != nil || result.Design != nil || result.Error != "" || result.Sealed != tc.sealed {
				t.Fatalf("resultado = %+v", result)
			}
		})
	}
	for _, store := range []ArtifactStore{ArtifactStoreOpenSpec, ArtifactStoreHybrid} {
		_, changeRoot := explainerChange(t, kickoff.ExplainerHTMLYes)
		if !ExplainerFreshness(changeRoot, store).Enabled {
			t.Errorf("el almacén %s debe activar la frescura", store)
		}
	}
}

func TestDeclaredArtifactStoreWrapper(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if _, ok := DeclaredArtifactStore(root); ok {
		t.Fatal("sin openspec/config.yaml no hay almacén declarado")
	}
	declareArtifactStore(t, root, "hybrid")
	if got, ok := DeclaredArtifactStore(root); !ok || got != ArtifactStoreHybrid {
		t.Fatalf("DeclaredArtifactStore = (%q, %v)", got, ok)
	}
}

func explainerOf(t *testing.T, status Status) *ExplainerProjection {
	t.Helper()
	fields := statusJSON(t, status)
	if _, present := fields["explainer"]; !present {
		return nil
	}
	projection := decodeField[ExplainerProjection](t, fields, "explainer")
	return &projection
}

func TestExplainerProjectionPresence(t *testing.T) {
	t.Parallel()
	html := explainerHTMLWith("sha256:" + strings.Repeat("a", 64))
	tests := []struct {
		name       string
		blocks     string
		value      kickoff.ExplainerHTML
		seal       bool
		html       bool
		change     string
		wantAbsent bool
		wantFor    string // project esperado
		wantSpec   string // state esperado de la spec; vacío: sin tipo
	}{
		{name: "sin bloque, sin sellar y sin html", seal: false, change: "demo", wantAbsent: true},
		{name: "el html sin activar no hace aparecer el bloque", seal: true, value: kickoff.ExplainerHTMLNo, html: true, change: "demo", wantAbsent: true},
		{name: "sellado sin campo y con html no lo proyecta", seal: true, html: true, change: "demo", wantAbsent: true},
		{name: "sellado yes sin bloque sdd lo proyecta como ask", seal: true, value: kickoff.ExplainerHTMLYes, change: "demo", wantFor: "ask", wantSpec: "missing"},
		{name: "yes con html obsoleto", seal: true, value: kickoff.ExplainerHTMLYes, html: true, change: "demo", wantFor: "ask", wantSpec: "stale"},
		{name: "bloque sdd sin clave es ask", blocks: "sdd:\n  iso: false\n", seal: true, value: kickoff.ExplainerHTMLNo, change: "demo", wantFor: "ask"},
		{name: "always", blocks: "sdd:\n  explainer_html: always\n", seal: true, value: kickoff.ExplainerHTMLYes, change: "demo", wantFor: "always", wantSpec: "missing"},
		{name: "never", blocks: "sdd:\n  explainer_html: never\n", seal: true, value: kickoff.ExplainerHTMLNo, html: true, change: "demo", wantFor: "never"},
		{name: "sin cambio seleccionado se proyecta el valor del proyecto", blocks: "sdd:\n  explainer_html: always\n", wantFor: "always"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := embeddedWithDeclaration(t, tt.blocks)
			if tt.change == "" {
				// Sin ningún cambio activo, el status no selecciona ninguno.
				ws = workspacetest.Embedded(t, ".")
				workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+tt.blocks)
			}
			changeRoot := filepath.Join(ws.Master, "openspec", "changes", "demo")
			if tt.seal {
				sealExplainerChange(t, changeRoot, "demo", tt.value, nil)
			}
			if tt.html {
				writeExplainerHTML(t, changeRoot, explainer.KindSpec, html)
			}
			status := resolveFrom(t, ws.Master, tt.change)
			got := explainerOf(t, status)
			if tt.wantAbsent {
				if got != nil {
					t.Fatalf("explainer = %+v, se esperaba la clave ausente", got)
				}
				return
			}
			if got == nil {
				t.Fatal("explainer ausente, se esperaba la proyección")
			}
			if got.Project != tt.wantFor || got.Reason != "" || got.Error != "" {
				t.Fatalf("explainer = %+v, se esperaba project %q", got, tt.wantFor)
			}
			if tt.wantSpec == "" {
				if got.Spec != nil || got.Design != nil {
					t.Fatalf("sin activación no hay tipos: %+v", got)
				}
				return
			}
			if got.Spec == nil || got.Spec.State != tt.wantSpec || got.Design == nil {
				t.Fatalf("spec = %+v, se esperaba %q", got.Spec, tt.wantSpec)
			}
		})
	}
}

// Una declaración inválida se proyecta invalid con el motivo y la continuación; nunca se asume ask.
func TestExplainerProjectionInvalidDeclaration(t *testing.T) {
	t.Parallel()
	for _, change := range []string{"demo", ""} {
		ws := embeddedWithDeclaration(t, "sdd:\n  explainer_html: quizás\n")
		got := explainerOf(t, resolveFrom(t, ws.Master, change))
		if got == nil || got.Project != "invalid" {
			t.Fatalf("explainer = %+v, se esperaba invalid", got)
		}
		for _, want := range []string{"sdd.explainer_html", `"quizás"`, "usa ask, always o never", "axiom workspace validate --path"} {
			if !strings.Contains(got.Reason, want) {
				t.Errorf("reason %q debe contener %q", got.Reason, want)
			}
		}
	}
}

// Governance.Kickoff.ExplainerHTML se proyecta en v2 y la copia de explainer coincide con v1.
func TestExplainerProjectionInV2(t *testing.T) {
	t.Parallel()
	ws := embeddedWithDeclaration(t, "")
	changeRoot := filepath.Join(ws.Master, "openspec", "changes", "demo")
	sealExplainerChange(t, changeRoot, "demo", kickoff.ExplainerHTMLYes, nil)
	status := resolveFrom(t, ws.Master, "demo")

	projected, err := ProjectStatusV2(status)
	if err != nil {
		t.Fatal(err)
	}
	if projected.Governance == nil || projected.Governance.Kickoff.ExplainerHTML != "yes" {
		t.Fatalf("governance.kickoff.explainerHtml = %+v", projected.Governance)
	}
	if !reflect.DeepEqual(projected.Explainer, status.Explainer) || projected.Explainer == nil {
		t.Fatalf("v2 explainer = %+v, v1 = %+v", projected.Explainer, status.Explainer)
	}

	// Un kickoff sin la clave no proyecta explainerHtml.
	other := embeddedWithDeclaration(t, "")
	sealExplainerChange(t, filepath.Join(other.Master, "openspec", "changes", "demo"), "demo", "", nil)
	projected, err = ProjectStatusV2(resolveFrom(t, other.Master, "demo"))
	if err != nil {
		t.Fatal(err)
	}
	if projected.Governance == nil || projected.Governance.Kickoff.ExplainerHTML != "" {
		t.Fatalf("sin clave no se proyecta: %+v", projected.Governance)
	}
}

// La frescura es solo informativa: nextRecommended, dependencias, blockedReasons y notes no
// dependen del valor sellado ni de la existencia del HTML.
func TestExplainerNeverChangesRoutingOrGates(t *testing.T) {
	t.Parallel()
	type routing struct {
		Next         string
		Dependencies Dependencies
		Blocked      []string
		Notes        []string
		Apply        ApplyState
	}
	capture := func(value kickoff.ExplainerHTML, html string) routing {
		ws := embeddedWithDeclaration(t, "")
		changeRoot := filepath.Join(ws.Master, "openspec", "changes", "demo")
		sealExplainerChange(t, changeRoot, "demo", value, nil)
		if html != "" {
			writeExplainerHTML(t, changeRoot, explainer.KindSpec, html)
			writeExplainerHTML(t, changeRoot, explainer.KindDesign, html)
		}
		status := resolveFrom(t, ws.Master, "demo")
		return routing{status.NextRecommended, status.Dependencies, status.BlockedReasons, status.Notes, status.ApplyState}
	}
	stale := explainerHTMLWith("sha256:" + strings.Repeat("0", 64))
	reference := capture(kickoff.ExplainerHTMLNo, "")
	for name, got := range map[string]routing{
		"yes sin html":      capture(kickoff.ExplainerHTMLYes, ""),
		"yes con html":      capture(kickoff.ExplainerHTMLYes, stale),
		"no con html":       capture(kickoff.ExplainerHTMLNo, stale),
		"sin campo":         capture("", ""),
		"sin campo y html":  capture("", stale),
		"yes con html roto": capture(kickoff.ExplainerHTMLYes, strings.Repeat("x", explainer.MaxBytes+1)),
	} {
		if !reflect.DeepEqual(got, reference) {
			t.Errorf("%s: el enrutado cambia:\n%+v\nreferencia:\n%+v", name, got, reference)
		}
	}
}

// En un cambio ISO el HTML desactualizado o ausente no bloquea ni altera las dependencias de
// archivado ni la proyección iso.
func TestExplainerISO(t *testing.T) {
	t.Parallel()
	capture := func(html string) (Status, map[string]any) {
		ws := embeddedWithDeclaration(t, "sdd:\n  iso: true\n")
		changeRoot := filepath.Join(ws.Master, "openspec", "changes", "demo")
		sealExplainerChange(t, changeRoot, "demo", kickoff.ExplainerHTMLYes, &kickoff.SDDSnapshot{ISO: true})
		switch html {
		case "fresh":
			digest := ExplainerFreshness(changeRoot, ArtifactStoreOpenSpec).Spec.ExpectedDigest
			writeExplainerHTML(t, changeRoot, explainer.KindSpec, explainerHTMLWith(digest))
			designDigest := ExplainerFreshness(changeRoot, ArtifactStoreOpenSpec).Design.ExpectedDigest
			writeExplainerHTML(t, changeRoot, explainer.KindDesign, explainerHTMLWith(designDigest))
		case "stale":
			writeExplainerHTML(t, changeRoot, explainer.KindSpec, explainerHTMLWith("sha256:"+strings.Repeat("0", 64)))
			writeExplainerHTML(t, changeRoot, explainer.KindDesign, explainerHTMLWith("sha256:"+strings.Repeat("0", 64)))
		}
		status := resolveFrom(t, ws.Master, "demo")
		return status, map[string]any{
			"next": status.NextRecommended, "dependencies": status.Dependencies, "iso": status.ISO,
			"blocked": status.BlockedReasons, "notes": status.Notes,
		}
	}
	freshStatus, fresh := capture("fresh")
	expectState(t, "referencia fresh", freshStatus.Explainer.Spec, "fresh")
	for _, variant := range []string{"stale", "missing"} {
		status, got := capture(variant)
		expectState(t, variant, status.Explainer.Spec, variant)
		if !reflect.DeepEqual(got, fresh) {
			t.Errorf("%s: el status ISO cambia:\n%+v\nreferencia:\n%+v", variant, got, fresh)
		}
		for _, reason := range status.BlockedReasons {
			if strings.Contains(strings.ToLower(reason), "resumen") || strings.Contains(strings.ToLower(reason), "explainer") {
				t.Errorf("%s: un motivo de bloqueo cita el HTML: %q", variant, reason)
			}
		}
	}
}
