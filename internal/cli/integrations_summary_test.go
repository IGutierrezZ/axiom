package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

const (
	summaryPatternBlock = "sdd:\n  id_pattern: \"RF-<DOM>-<NN>\"\n"

	summaryAlfaDelta = "## ADDED Requirements\n\n" +
		"### Requirement: Alta de cuenta (prov:ADO:1)\nEl sistema DEBE dar de alta cuentas.\n\n#### Scenario: alta\n- **WHEN** se pide\n\n" +
		"### Requirement: Baja de cuenta (prov:ADO:2)\nEl sistema DEBE dar de baja cuentas.\n"

	summaryBetaDelta = "## MODIFIED Requirements\n\n### Requirement: Cambio con <marcas> (prov:ADO:3)\nTexto con `codigo`.\n"

	summaryIDsYAML = "schema: axiom.requirement-ids/v1\npattern: RF-<DOM>-<NN>\nassignments:\n" +
		"  - provisional: prov:ADO:1\n    definitive: RF-ADO-01\n    capability: alfa\n    requirement: Alta de cuenta\n" +
		"  - provisional: prov:ADO:2\n    definitive: RF-ADO-02\n    capability: alfa\n    requirement: Baja de cuenta\n"
)

// summaryEnv es el escenario de `summary push`: el cambio x sellado, el work item 100 vinculado y dos
// capacidades con deltas provisionales.
func newSummaryEnv(t *testing.T, style kickoff.ExecutionStyle, extraBlock string) *tasksEnv {
	t.Helper()
	e := newTasksEnvWith(t, adoBlock+extraBlock+summaryPatternBlock, style, "")
	writeCLIFixture(t, filepath.Join(e.changeRoot, "specs", "alfa", "spec.md"), summaryAlfaDelta)
	writeCLIFixture(t, filepath.Join(e.changeRoot, "specs", "beta", "spec.md"), summaryBetaDelta)
	return e
}

func (e *tasksEnv) summaryCLI(t *testing.T, args ...string) (int, summaryJSON, string, string) {
	t.Helper()
	full := append([]string{"summary", "push"}, args...)
	full = append(full, "--cwd", e.root, "--json")
	code, out, errOut := runIntegrationsCLI(t, "", full...)
	var got summaryJSON
	if strings.TrimSpace(out) != "" {
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("la salida --json no es JSON válido: %v\n%s", err, out)
		}
	}
	return code, got, out, errOut
}

func (e *tasksEnv) recordSpecGate(t *testing.T) {
	t.Helper()
	var out bytes.Buffer
	if err := RunSDDGate([]string{"record", "--cwd", e.root, "--change", "x", "--gate", "spec", "--decision", "approved", "--reason", "ok", "--actor", "Ana"}, &out); err != nil {
		t.Fatalf("aprobar la compuerta spec: %v", err)
	}
}

func (e *tasksEnv) workItemField(t *testing.T, field string) string {
	t.Helper()
	wi, ok := e.srv.WorkItem(tasksParentID)
	if !ok {
		t.Fatalf("el work item %d no existe", tasksParentID)
	}
	s, _ := wi.Fields[field].(string)
	return s
}

func tagSet(raw string) map[string]bool {
	set := map[string]bool{}
	for _, tag := range strings.Split(raw, ";") {
		if tag = strings.TrimSpace(tag); tag != "" {
			set[tag] = true
		}
	}
	return set
}

// prov:INT:4 y prov:ADO:4: la primera ejecución escribe descripción y etiquetas con un solo PATCH.
func TestIntegrationsSummaryPushWritesDescriptionAndTagsWithOnePatch(t *testing.T) {
	e := newSummaryEnv(t, kickoff.ExecutionContinuous, "")

	code, got, _, errOut := e.summaryCLI(t, "--change", "x")

	if code != 0 || got.Status != trackerStatusDone || !got.Wrote || got.Schema != integrationsSummarySchema || got.Folder != summaryFolderOrdinary {
		t.Fatalf("código %d, salida %+v, stderr %q", code, got, errOut)
	}
	if got.Description == nil || !got.Description.Changed || got.Description.Bytes == 0 || len(got.Description.SHA256) != 64 {
		t.Fatalf("descripción: %+v", got.Description)
	}
	if got.Tags == nil || got.Tags.Prefix != "req:" || !got.Tags.Sent || len(got.Tags.Added) != 3 || len(got.Tags.Removed) != 0 {
		t.Fatalf("etiquetas: %+v", got.Tags)
	}
	if e.requests("PATCH") != 1 {
		t.Errorf("una sola escritura: %d PATCH", e.requests("PATCH"))
	}
	html := e.workItemField(t, "System.Description")
	for _, want := range []string{"<h2>alfa</h2>", "<h3>Alta de cuenta (prov:ADO:1)</h3>", "<h2>beta</h2>", "Cambio con &lt;marcas&gt;", "<code>codigo</code>"} {
		if !strings.Contains(html, want) {
			t.Errorf("la descripción debe contener %q:\n%s", want, html)
		}
	}
	tags := tagSet(e.workItemField(t, "System.Tags"))
	for _, want := range []string{"req:prov:ADO:1", "req:prov:ADO:2", "req:prov:ADO:3"} {
		if !tags[want] {
			t.Errorf("falta la etiqueta %s en %v", want, tags)
		}
	}
}

// La segunda ejecución con los mismos datos es idéntica byte a byte y no escribe.
func TestIntegrationsSummaryPushSecondRunIsIdenticalAndWritesNothing(t *testing.T) {
	e := newSummaryEnv(t, kickoff.ExecutionContinuous, "")
	_, first, _, _ := e.summaryCLI(t, "--change", "x")
	patches := e.requests("PATCH")
	htmlBefore := e.workItemField(t, "System.Description")

	code, second, _, errOut := e.summaryCLI(t, "--change", "x")

	if code != 0 || second.Wrote || second.Description.Changed || second.Tags.Sent || len(second.Tags.Added) != 0 {
		t.Fatalf("código %d, salida %+v, stderr %q", code, second, errOut)
	}
	if e.requests("PATCH") != patches {
		t.Errorf("la segunda ejecución no envía PATCH: %d antes, %d después", patches, e.requests("PATCH"))
	}
	if first.Description.SHA256 != second.Description.SHA256 || first.Description.Bytes != second.Description.Bytes {
		t.Errorf("el HTML debe ser idéntico byte a byte: %+v frente a %+v", first.Description, second.Description)
	}
	if e.workItemField(t, "System.Description") != htmlBefore {
		t.Errorf("la descripción no cambia")
	}
}

// Etiquetas añadidas, retiradas y ajenas conservadas.
func TestIntegrationsSummaryPushAddsRemovesAndKeepsForeignTags(t *testing.T) {
	e := newSummaryEnv(t, kickoff.ExecutionContinuous, "")
	e.srv.SeedWorkItem(tasksParentID, 3, map[string]any{
		"System.WorkItemType": "User Story", "System.Title": "Historia de origen",
		"System.Tags": "equipo-a; req:prov:ADO:1; req:prov:ADO:9; REQ:otra",
	})

	code, got, _, errOut := e.summaryCLI(t, "--change", "x")

	if code != 0 || got.Tags == nil {
		t.Fatalf("código %d, salida %+v, stderr %q", code, got, errOut)
	}
	if strings.Join(got.Tags.Added, ",") != "req:prov:ADO:2,req:prov:ADO:3" || strings.Join(got.Tags.Removed, ",") != "req:prov:ADO:9" {
		t.Errorf("añadidas %v, retiradas %v", got.Tags.Added, got.Tags.Removed)
	}
	tags := tagSet(e.workItemField(t, "System.Tags"))
	if !tags["equipo-a"] || !tags["REQ:otra"] || tags["req:prov:ADO:9"] || len(tags) != 5 {
		t.Errorf("las ajenas se conservan y la obsoleta se retira: %v", tags)
	}
}

// Por encima de req_tag_max no se envía ninguna etiqueta de ID y se retiran las provisionales
// que ya estaban; la descripción se actualiza igualmente.
func TestIntegrationsSummaryPushOverTheTagCapRemovesAllOwnTags(t *testing.T) {
	e := newSummaryEnv(t, kickoff.ExecutionContinuous, "    req_tag_max: 2\n")
	e.srv.SeedWorkItem(tasksParentID, 3, map[string]any{
		"System.WorkItemType": "User Story", "System.Title": "Historia de origen",
		"System.Tags": "equipo-a; req:prov:ADO:1",
	})

	code, got, _, errOut := e.summaryCLI(t, "--change", "x")

	if code != 0 || got.Status != trackerStatusDone || !got.Wrote {
		t.Fatalf("código %d, salida %+v, stderr %q", code, got, errOut)
	}
	if len(got.Warnings) != 1 || got.Warnings[0].Code != "req_tags_over_limit" {
		t.Errorf("avisos: %+v", got.Warnings)
	}
	if len(got.Tags.Added) != 0 || strings.Join(got.Tags.Removed, ",") != "req:prov:ADO:1" || len(got.Tags.Desired) != 3 {
		t.Errorf("etiquetas: %+v", got.Tags)
	}
	if tags := tagSet(e.workItemField(t, "System.Tags")); len(tags) != 1 || !tags["equipo-a"] {
		t.Errorf("solo queda la ajena: %v", tags)
	}
	if !got.Description.Changed || e.workItemField(t, "System.Description") == "" {
		t.Errorf("la descripción se actualiza igualmente")
	}
}

// Una descripción editada a mano se sustituye y la salida lo indica.
func TestIntegrationsSummaryPushReplacesAHandEditedDescription(t *testing.T) {
	e := newSummaryEnv(t, kickoff.ExecutionContinuous, "")
	e.summaryCLI(t, "--change", "x")
	e.srv.SeedWorkItem(tasksParentID, 9, map[string]any{
		"System.WorkItemType": "User Story", "System.Title": "Historia de origen",
		"System.Description": "<p>escrita a mano</p>", "System.Tags": "req:prov:ADO:1; req:prov:ADO:2; req:prov:ADO:3",
	})

	code, got, _, errOut := e.summaryCLI(t, "--change", "x")

	if code != 0 || !got.Wrote || !got.Description.Changed || got.Tags.Sent {
		t.Fatalf("código %d, salida %+v, stderr %q", code, got, errOut)
	}
	if strings.Contains(e.workItemField(t, "System.Description"), "a mano") {
		t.Errorf("la descripción manual se sustituye por diseño")
	}
}

// prov:ARC:1: tras archivar, el resumen se reenvía con los IDs definitivos sobre la carpeta archivada.
func TestIntegrationsSummaryPushArchivedFolderResendsDefinitiveIDs(t *testing.T) {
	e := newSummaryEnv(t, kickoff.ExecutionContinuous, "")
	e.summaryCLI(t, "--change", "x")
	archived := filepath.Join(e.root, "openspec", "changes", "archive", "2026-11-27-x")
	if err := os.MkdirAll(filepath.Dir(archived), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(e.changeRoot, archived); err != nil {
		t.Fatal(err)
	}
	writeCLIFixture(t, filepath.Join(archived, "ids.yaml"), summaryIDsYAML)
	// La delta archivada conserva un provisional residual que ids.yaml traduce, y otra ya reescrita.
	writeCLIFixture(t, filepath.Join(archived, "specs", "alfa", "spec.md"), strings.Replace(summaryAlfaDelta, "Baja de cuenta (prov:ADO:2)", "Baja de cuenta (RF-ADO-02)", 1))
	writeCLIFixture(t, filepath.Join(archived, "specs", "beta", "spec.md"), strings.Replace(summaryBetaDelta, "prov:ADO:3", "RF-ADO-03", 1))

	code, got, _, errOut := e.summaryCLI(t, "--change", "x")

	if code != 0 || got.Folder != summaryFolderArchived || !got.Wrote || !got.Description.Changed {
		t.Fatalf("código %d, salida %+v, stderr %q", code, got, errOut)
	}
	if strings.Join(got.Tags.Desired, ",") != "req:RF-ADO-01,req:RF-ADO-02,req:RF-ADO-03" {
		t.Errorf("IDs definitivos: %v", got.Tags.Desired)
	}
	if len(got.Tags.Removed) != 3 {
		t.Errorf("se retiran las tres provisionales: %v", got.Tags.Removed)
	}
	html := e.workItemField(t, "System.Description")
	if strings.Contains(html, "prov:") || !strings.Contains(html, "Alta de cuenta (RF-ADO-01)") {
		t.Errorf("la descripción reenviada lleva IDs definitivos:\n%s", html)
	}
	tags := tagSet(e.workItemField(t, "System.Tags"))
	if len(tags) != 3 || !tags["req:RF-ADO-01"] {
		t.Errorf("etiquetas finales: %v", tags)
	}
	// El diario traído de la carpeta ordinaria (el primer envío) no crece: la archivada no se escribe.
	notes, err := integrations.TrackerNotes(integrations.Folder{WorkspaceRoot: e.root, Root: archived})
	if err != nil || len(notes) != 1 {
		t.Errorf("la carpeta archivada no se escribe: %d anotaciones (%v)", len(notes), err)
	}
}

// Un provisional residual sin entrada en ids.yaml se rechaza sin ninguna escritura y nombra archive-check.
func TestIntegrationsSummaryPushArchivedUnresolvedProvisionalIsRefused(t *testing.T) {
	e := newSummaryEnv(t, kickoff.ExecutionContinuous, "")
	archived := filepath.Join(e.root, "openspec", "changes", "archive", "2026-11-27-x")
	if err := os.MkdirAll(filepath.Dir(archived), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(e.changeRoot, archived); err != nil {
		t.Fatal(err)
	}
	writeCLIFixture(t, filepath.Join(archived, "ids.yaml"), strings.Replace(summaryIDsYAML, "prov:ADO:2", "prov:ADO:7", 1))

	code, got, _, _ := e.summaryCLI(t, "--change", "x")

	if code != 1 || got.Status != trackerStatusRefused || got.Refusal == nil || got.Refusal.Code != "provisional_ids_unresolved" ||
		!strings.Contains(got.Refusal.Next, "axiom sdd archive-check --cwd") || !strings.Contains(got.Refusal.Next, "--change x --json") {
		t.Fatalf("código %d, salida %+v", code, got)
	}
	if e.requests("PATCH") != 0 {
		t.Errorf("ninguna escritura ante un provisional sin traducir")
	}
}

// El WI que ya no admite edición da tracker_rejected con el reintento.
func TestIntegrationsSummaryPushRejectedByTheTrackerNamesTheRetry(t *testing.T) {
	e := newSummaryEnv(t, kickoff.ExecutionContinuous, "")
	e.srv.Reply("PATCH", "/contoso/Fabrikam/_apis/wit/workitems/100", adotest.Response{Status: 403})

	code, got, _, _ := e.summaryCLI(t, "--change", "x")

	if code != 1 || got.Refusal == nil || got.Refusal.Code != "tracker_rejected" ||
		got.Refusal.Next != "axiom integrations summary push --change x" {
		t.Fatalf("código %d, salida %+v", code, got)
	}
	notes := notesOf(t, e)
	if len(notes) != 1 || notes[0].Verb != "summary push" || notes[0].Code != "tracker_rejected" {
		t.Errorf("el rechazo se anota en continuous: %+v", notes)
	}
}

func TestIntegrationsSummaryPushCheckpointedNeedsTheApprovedSpecGate(t *testing.T) {
	e := newSummaryEnv(t, kickoff.ExecutionCheckpointed, "")

	code, got, _, _ := e.summaryCLI(t, "--change", "x")

	if code != 1 || got.Refusal == nil || got.Refusal.Code != "gate_not_approved" ||
		!strings.Contains(got.Refusal.Next, "axiom sdd gate show --change x") || !strings.Contains(got.Refusal.Text, "spec") {
		t.Fatalf("código %d, salida %+v", code, got)
	}
	if len(e.srv.Requests()) != 0 || fileExistsCLI(filepath.Join(e.changeRoot, integrations.TrackerNotesFile)) {
		t.Errorf("ninguna petición ni diario en checkpointed")
	}

	e.recordSpecGate(t)
	if code, got, _, errOut := e.summaryCLI(t, "--change", "x"); code != 0 || !got.Wrote {
		t.Fatalf("con la compuerta aprobada procede: código %d, salida %+v, stderr %q", code, got, errOut)
	}

	// Una edición de la spec posterior a la aprobación caduca el digest.
	writeCLIFixture(t, filepath.Join(e.changeRoot, "specs", "beta", "spec.md"), summaryBetaDelta+"\nNota nueva.\n")
	if code, got, _, _ := e.summaryCLI(t, "--change", "x"); code != 1 || got.Refusal == nil || got.Refusal.Code != "gate_not_approved" {
		t.Fatalf("el digest caducado se rechaza: código %d, salida %+v", code, got)
	}
}

func TestIntegrationsSummaryPushContinuousOnlyNeedsTheDeltas(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, "")

	code, got, _, _ := e.summaryCLI(t, "--change", "x")

	if code != 1 || got.Refusal == nil || got.Refusal.Code != "artifact_missing" || len(e.srv.Requests()) != 0 {
		t.Fatalf("sin deltas: código %d, salida %+v", code, got)
	}

	writeCLIFixture(t, filepath.Join(e.changeRoot, "specs", "alfa", "spec.md"), summaryAlfaDelta)
	if code, got, _, errOut := e.summaryCLI(t, "--change", "x"); code != 0 || !got.Wrote {
		t.Fatalf("basta con que exista el artefacto: código %d, salida %+v, stderr %q", code, got, errOut)
	}
	notes := notesOf(t, e)
	if len(notes) != 2 || notes[0].Code != "artifact_missing" || notes[1].Code != "done" || notes[1].Verb != "summary push" {
		t.Errorf("el diario recoge el rechazo y el éxito: %+v", notes)
	}
}

func TestIntegrationsSummaryPushODDLaneRejectedWithoutFiles(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	writeCLIFixture(t, filepath.Join(e.root, "odd", "tasks", "f.md"), "# f\n")

	code, out, _ := runIntegrationsCLI(t, "", "summary", "push", "--change", "f", "--cwd", e.root, "--json")

	var got summaryJSON
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
	if code != 1 || got.Refusal == nil || got.Refusal.Code != "odd_lane_not_supported" {
		t.Fatalf("código %d, salida %+v", code, got)
	}
	if n := len(e.srv.Requests()); n != 0 || fileExistsCLI(filepath.Join(e.root, "openspec", "changes", "f")) {
		t.Errorf("ninguna petición ni carpeta: %d peticiones", n)
	}
}

func TestIntegrationsSummaryPushNotApplicableWithoutActivePlugin(t *testing.T) {
	e := newTasksEnvWith(t, "", kickoff.ExecutionContinuous, "")

	code, got, _, errOut := e.summaryCLI(t, "--change", "x")

	if code != 0 || got.Status != trackerStatusNotApplicable || errOut != "" || len(e.srv.Requests()) != 0 {
		t.Fatalf("código %d, salida %+v, stderr %q", code, got, errOut)
	}
	if fileExistsCLI(filepath.Join(e.changeRoot, integrations.TrackerNotesFile)) {
		t.Errorf("no se crea el diario")
	}
}

func TestIntegrationsSummaryPushWorkItemNotLinkedAndChangeNotFound(t *testing.T) {
	e := newSummaryEnv(t, kickoff.ExecutionContinuous, "")
	if err := os.Remove(filepath.Join(e.changeRoot, integrations.LinksFile)); err != nil {
		t.Fatal(err)
	}
	if code, got, _, _ := e.summaryCLI(t, "--change", "x"); code != 1 || got.Refusal == nil || got.Refusal.Code != "work_item_not_linked" {
		t.Fatalf("sin vínculo: código %d, salida %+v", code, got)
	}
	if code, got, _, _ := e.summaryCLI(t, "--change", "nada"); code != 1 || got.Refusal == nil || got.Refusal.Code != "change_not_found" {
		t.Fatalf("sin cambio: código %d, salida %+v", code, got)
	}
	if len(e.srv.Requests()) != 0 {
		t.Errorf("ninguna petición")
	}
}

func TestIntegrationsSummaryPushUsageErrorsMakeNoRequests(t *testing.T) {
	e := newSummaryEnv(t, kickoff.ExecutionContinuous, "")
	for _, args := range [][]string{{}, {"--change", "../x"}, {"--change", "x", "sobra"}} {
		full := append([]string{"summary", "push"}, args...)
		full = append(full, "--cwd", e.root)
		if code, _, _ := runIntegrationsCLI(t, "", full...); code != 2 {
			t.Errorf("%v: código %d, quería 2", args, code)
		}
	}
	if code, _, _ := runIntegrationsCLI(t, "", "summary", "inspect"); code != 2 {
		t.Errorf("un verbo desconocido de summary es error de uso: %d", code)
	}
	if len(e.srv.Requests()) != 0 {
		t.Errorf("ninguna petición ante un error de uso")
	}
}

func TestIntegrationsSummaryHelp(t *testing.T) {
	code, out, _ := runIntegrationsCLI(t, "", "summary", "--help")
	if code != 0 || !strings.Contains(out, "push") || !strings.Contains(out, "--json") {
		t.Fatalf("ayuda de summary: código %d\n%s", code, out)
	}
	code, out, _ = runIntegrationsCLI(t, "", "summary", "push", "--help")
	if code != 0 {
		t.Fatalf("summary push --help: código %d", code)
	}
	for _, flag := range []string{"--change", "--cwd", "--json", "axiom sdd publish --archive"} {
		if !strings.Contains(out, flag) {
			t.Errorf("la ayuda de summary push debe mencionar %s", flag)
		}
	}
}
