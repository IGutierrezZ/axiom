package sddstatus

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Pruebas de la proyección `publication` del status (REQ-27.18, C-05-c): cuándo procede proyectar
// y cómo se traducen los hechos de git a un estado. Es aditiva: sin estado, el JSON no cambia.

const publicationChange = "x"

// dueStatus es un status de cambio activo con un kickoff de la política dada y los artefactos dados.
func dueStatus(style kickoff.ExecutionStyle, docs kickoff.DocsPublication, gates []kickoff.GateState, artifacts map[string]ArtifactState, completed int) *Status {
	change := publicationChange
	return &Status{
		ChangeName:    &change,
		ArtifactStore: ArtifactStoreOpenSpec,
		Artifacts:     artifacts,
		TaskProgress:  TaskProgress{Completed: completed},
		Governance: &Governance{
			Kickoff: kickoff.Kickoff{Config: kickoff.FlowConfig{ExecutionStyle: style, DocsPublication: docs}},
			Gates:   gates,
		},
	}
}

func approved(keys ...kickoff.GateKey) []kickoff.GateState {
	var gates []kickoff.GateState
	for _, key := range keys {
		gates = append(gates, kickoff.GateState{Key: key, Status: "approved"})
	}
	return gates
}

func gates(states ...kickoff.GateState) []kickoff.GateState { return states }

func passedGate(key kickoff.GateKey) kickoff.GateState {
	return kickoff.GateState{Key: key, Status: "approved"}
}

func openGate(key kickoff.GateKey, status string) kickoff.GateState {
	return kickoff.GateState{Key: key, Status: status}
}

// sealArchivedChange sella un kickoff en la carpeta archivada del cambio bajo root.
func sealArchivedChange(t *testing.T, root string, style kickoff.ExecutionStyle, docs kickoff.DocsPublication) {
	t.Helper()
	changeRoot := filepath.Join(root, "openspec", "changes", "archive", "2026-01-02-"+publicationChange)
	if err := os.MkdirAll(changeRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: publicationChange, SealedBy: "prueba",
		Config: kickoff.FlowConfig{
			FlowMode: kickoff.FlowSDD, ExecutionStyle: style, ExecutionStyleSource: "explicit", DocsPublication: docs,
			HandoffPolicy: kickoff.HandoffNone,
			Roles:         []kickoff.KickoffRole{{Role: "fullstack", GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"}},
		},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
	}); err != nil || !ok {
		t.Fatalf("sellar el kickoff archivado: %v (publicado: %v)", err, ok)
	}
}

// REQ-27.13: solo se proyecta en el momento en que `axiom sdd publish` se espera.
func TestPublicationDue(t *testing.T) {
	planned := map[string]ArtifactState{"specs": ArtifactDone, "design": ArtifactMissing, "tasks": ArtifactMissing}
	reviewed := map[string]ArtifactState{"specs": ArtifactDone, "design": ArtifactDone, "tasks": ArtifactDone, "verifyReport": ArtifactDone}
	none := map[string]ArtifactState{"specs": ArtifactPartial, "design": ArtifactMissing, "tasks": ArtifactMissing}
	designing := map[string]ArtifactState{"specs": ArtifactDone, "design": ArtifactDone, "tasks": ArtifactMissing}
	designPartial := map[string]ArtifactState{"specs": ArtifactDone, "design": ArtifactPartial, "tasks": ArtifactMissing}
	reviewedTasks := map[string]ArtifactState{"specs": ArtifactDone, "design": ArtifactDone, "tasks": ArtifactDone, "applyProgress": ArtifactMissing}
	applying := map[string]ArtifactState{"specs": ArtifactDone, "design": ArtifactDone, "tasks": ArtifactDone, "applyProgress": ArtifactDone}
	checkpointed, continuous := kickoff.ExecutionCheckpointed, kickoff.ExecutionContinuous

	archivedRoot := t.TempDir()
	sealArchivedChange(t, archivedRoot, continuous, kickoff.DocsPublicationAuto)
	archivedCheckpointed := t.TempDir()
	sealArchivedChange(t, archivedCheckpointed, checkpointed, "")
	archivedWithoutAuto := t.TempDir()
	sealArchivedChange(t, archivedWithoutAuto, continuous, "")
	archivedWithoutKickoff := t.TempDir()
	if err := os.MkdirAll(filepath.Join(archivedWithoutKickoff, "openspec", "changes", "archive", "2026-01-02-"+publicationChange), 0o755); err != nil {
		t.Fatal(err)
	}
	archived := func() *Status {
		status := dueStatus(continuous, kickoff.DocsPublicationAuto, nil, nil, 0)
		status.Governance, status.Archived = nil, &ArchivedProjection{Path: "openspec/changes/archive/2026-01-02-x"}
		return status
	}

	cases := []struct {
		name        string
		status      *Status
		root        string
		wantDue     bool
		wantArchive bool
	}{
		{"checkpointed con spec aprobada", dueStatus(checkpointed, "", approved(kickoff.GateSpec), planned, 0), "", true, false},
		{"checkpointed con tasks y apply por rol aprobadas", dueStatus(checkpointed, "", approved(kickoff.GateTasks, kickoff.GateKey("role-apply:fullstack")), reviewed, 3), "", true, false},
		{"checkpointed solo con integration aprobada", dueStatus(checkpointed, "", approved(kickoff.GateIntegration), reviewed, 3), "", false, false},
		// W1 de C-05-c: con la fase siguiente en curso no se proyecta; iterar no publica nada.
		{"checkpointed con spec aprobada y design por empezar", dueStatus(checkpointed, "", gates(passedGate(kickoff.GateSpec), openGate(kickoff.GateDesign, "pending")), planned, 0), "", true, false},
		{"checkpointed con spec aprobada y design en iteración", dueStatus(checkpointed, "", gates(passedGate(kickoff.GateSpec), openGate(kickoff.GateDesign, "pending")), designing, 0), "", false, false},
		{"checkpointed con spec aprobada y design a medio escribir", dueStatus(checkpointed, "", gates(passedGate(kickoff.GateSpec), openGate(kickoff.GateDesign, "pending")), designPartial, 0), "", false, false},
		{"checkpointed con spec aprobada y design rechazado", dueStatus(checkpointed, "", gates(passedGate(kickoff.GateSpec), openGate(kickoff.GateDesign, "rejected")), designing, 0), "", false, false},
		{"checkpointed con design aprobado y tasks en iteración", dueStatus(checkpointed, "", gates(passedGate(kickoff.GateSpec), passedGate(kickoff.GateDesign), openGate(kickoff.GateTasks, "pending")), reviewedTasks, 0), "", false, false},
		{"checkpointed con tasks aprobada y apply sin empezar", dueStatus(checkpointed, "", gates(passedGate(kickoff.GateTasks), openGate("role-apply:fullstack", "pending")), reviewedTasks, 0), "", true, false},
		{"checkpointed con tasks aprobada y apply en curso", dueStatus(checkpointed, "", gates(passedGate(kickoff.GateTasks), openGate("role-apply:fullstack", "pending")), reviewedTasks, 1), "", false, false},
		{"checkpointed con tasks aprobada y progreso de apply", dueStatus(checkpointed, "", gates(passedGate(kickoff.GateTasks), openGate("role-apply:fullstack", "pending")), applying, 0), "", false, false},
		{"checkpointed con solo integration abierta", dueStatus(checkpointed, "", gates(passedGate(kickoff.GateTasks), passedGate("role-apply:fullstack"), openGate(kickoff.GateIntegration, "pending")), reviewed, 3), "", true, false},
		{"continuous con auto y design parcial", dueStatus(continuous, kickoff.DocsPublicationAuto, nil, designPartial, 0), "", false, false},
		{"checkpointed sin ninguna aprobada", dueStatus(checkpointed, "", nil, planned, 0), "", false, false},
		{"continuous sin auto", dueStatus(continuous, "", nil, planned, 0), "", false, false},
		{"continuous con auto y specs hechas", dueStatus(continuous, kickoff.DocsPublicationAuto, nil, planned, 0), "", true, false},
		{"continuous con auto sin ningún artefacto hecho", dueStatus(continuous, kickoff.DocsPublicationAuto, nil, none, 0), "", false, false},
		{"continuous con auto y tareas completadas", dueStatus(continuous, kickoff.DocsPublicationAuto, nil, planned, 1), "", false, false},
		{"continuous con auto y verify hecho", dueStatus(continuous, kickoff.DocsPublicationAuto, nil, reviewed, 0), "", false, false},
		{"archivado continuous con auto", archived(), archivedRoot, true, true},
		{"archivado checkpointed", archived(), archivedCheckpointed, false, true},
		{"archivado continuous sin auto", archived(), archivedWithoutAuto, false, true},
		{"archivado sin kickoff", archived(), archivedWithoutKickoff, false, false},
		{"almacén engram", withStore(dueStatus(checkpointed, "", approved(kickoff.GateSpec), planned, 0), ArtifactStoreEngram), "", false, false},
		{"almacén none", withStore(dueStatus(checkpointed, "", approved(kickoff.GateSpec), planned, 0), ArtifactStoreNone), "", false, false},
		{"sin cambio seleccionado", withoutChange(dueStatus(checkpointed, "", approved(kickoff.GateSpec), planned, 0)), "", false, false},
		{"sin kickoff", withoutGovernance(dueStatus(checkpointed, "", approved(kickoff.GateSpec), planned, 0)), "", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := tc.root
			if root == "" {
				root = t.TempDir()
			}
			archive, due := publicationDue(tc.status, workspace.Location{WorkspaceRoot: root})
			if due != tc.wantDue || (due && archive != tc.wantArchive) {
				t.Errorf("publicationDue = (archive %v, due %v), se esperaba (archive %v, due %v)", archive, due, tc.wantArchive, tc.wantDue)
			}
		})
	}
}

func withStore(status *Status, store ArtifactStore) *Status {
	status.ArtifactStore = store
	return status
}

func withoutChange(status *Status) *Status {
	status.ChangeName = nil
	return status
}

func withoutGovernance(status *Status) *Status {
	status.Governance = nil
	return status
}

// Sin motor (el de los fixtures antiguos devuelve nulo) no se proyecta nada, aunque proceda.
func TestDecoratePublicationWithoutEngineProjectsNothing(t *testing.T) {
	status := dueStatus(kickoff.ExecutionCheckpointed, "", approved(kickoff.GateSpec), nil, 0)
	decoratePublication(status, workspace.Location{WorkspaceRoot: t.TempDir()})
	if status.Publication != nil {
		t.Errorf("Publication = %+v, se esperaba nil sin motor", status.Publication)
	}
}

// Un fallo de lectura de git (la configuración del remoto, por ejemplo) no se disfraza de estado:
// se proyecta `unavailable` con el código `git_failed`.
func TestDecoratePublicationProjectsUnavailableWhenTheReadFails(t *testing.T) {
	cases := []struct {
		name    string
		refusal *branch.Refusal
		want    string
	}{
		{"fallo de git al leer el remoto", branch.GitFailed("status", "dir", "leer la configuración del remoto origin", os.ErrInvalid), "git_failed"},
		{"rama equivocada", &branch.Refusal{Code: branch.CodeWrongBranch, Message: "m", Continuation: "c"}, "wrong_branch"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := dueStatus(kickoff.ExecutionCheckpointed, "", approved(kickoff.GateSpec), nil, 0)
			read := func(context.Context, branch.PublicationRequest) (branch.PublicationState, *branch.Refusal) {
				return branch.PublicationState{}, tc.refusal
			}
			decoratePublicationWith(status, workspace.Location{WorkspaceRoot: t.TempDir()}, read)
			if status.Publication == nil || status.Publication.State != "unavailable" || status.Publication.Unavailable != tc.want {
				t.Fatalf("Publication = %+v, se esperaba unavailable con el código %q", status.Publication, tc.want)
			}
			if status.Publication.Invocation != "" || status.Publication.PushCommand != "" {
				t.Errorf("un estado unavailable no lleva órdenes: %+v", status.Publication)
			}
		})
	}
}

// Una fila por estado de la tabla de precedencia de la design.
func TestProjectPublication(t *testing.T) {
	const root = `C:\Users\dev name\axiom-wt\x\axiom`
	target := branch.PublicationTarget{GitRoot: root, Branch: "feat/x", IntegrationBranch: "main", PRTarget: "main", PlanningRel: "openspec"}
	invocation := "axiom sdd publish --cwd " + pathquote.Quote(`C:\ws`) + " --change x --branch feat/x"
	push := "git -C " + pathquote.Quote(root) + " push origin feat/x:feat/x"
	state := func(mutate func(*branch.PublicationState)) branch.PublicationState {
		facts := branch.PublicationState{Target: target, RemoteBranch: true}
		mutate(&facts)
		return facts
	}
	docs := []string{"openspec/changes/x/design.md"}

	cases := []struct {
		name  string
		facts branch.PublicationState
		want  *PublicationProjection
	}{
		{"sin origin no hay publicación", state(func(s *branch.PublicationState) { s.RemoteMissing, s.DirtyDocs = true, docs }), nil},
		{"rango desconocido", state(func(s *branch.PublicationState) { s.RangeUnknown = true }),
			&PublicationProjection{State: "unavailable", Remote: "origin", Branch: "feat/x", PRTarget: "main", Unavailable: "unpublished_range_unknown"}},
		{"documentación sin confirmar", state(func(s *branch.PublicationState) { s.DirtyDocs = docs }),
			&PublicationProjection{State: "pending", Reasons: []string{"docs_uncommitted"}, Remote: "origin", Branch: "feat/x", PRTarget: "main", Invocation: invocation}},
		{"documentación sin confirmar con todo lo demás", state(func(s *branch.PublicationState) {
			s.DirtyDocs, s.Unpushed, s.RemoteBranch, s.CodeCommits = docs, 2, false, []string{"abc1234"}
		}),
			&PublicationProjection{State: "pending", Reasons: []string{"docs_uncommitted", "branch_ahead", "remote_branch_missing", "code_commits"}, Remote: "origin", Branch: "feat/x", PRTarget: "main", Invocation: invocation}},
		{"rango demasiado grande", state(func(s *branch.PublicationState) { s.Unpushed, s.RangeTooBig = 501, true }),
			&PublicationProjection{State: "code_unpublished", Reasons: []string{"range_too_big"}, Remote: "origin", Branch: "feat/x", PRTarget: "main", PushCommand: push}},
		{"commits de código sin subir", state(func(s *branch.PublicationState) { s.Unpushed, s.CodeCommits = 2, []string{"abc1234", "def5678"} }),
			&PublicationProjection{State: "code_unpublished", Reasons: []string{"code_commits"}, Remote: "origin", Branch: "feat/x", PRTarget: "main", PushCommand: push, CodeCommits: []string{"abc1234", "def5678"}}},
		{"commits de código sin rama remota", state(func(s *branch.PublicationState) {
			s.Unpushed, s.CodeCommits, s.RemoteBranch = 1, []string{"abc1234"}, false
		}),
			&PublicationProjection{State: "code_unpublished", Reasons: []string{"code_commits", "remote_branch_missing"}, Remote: "origin", Branch: "feat/x", PRTarget: "main", PushCommand: push, CodeCommits: []string{"abc1234"}}},
		{"solo commits de documentación sin subir", state(func(s *branch.PublicationState) { s.Unpushed = 1 }),
			&PublicationProjection{State: "pending", Reasons: []string{"branch_ahead"}, Remote: "origin", Branch: "feat/x", PRTarget: "main", Invocation: invocation}},
		{"documentación sin subir en una rama sin remota", state(func(s *branch.PublicationState) { s.Unpushed, s.RemoteBranch = 1, false }),
			&PublicationProjection{State: "pending", Reasons: []string{"branch_ahead", "remote_branch_missing"}, Remote: "origin", Branch: "feat/x", PRTarget: "main", Invocation: invocation}},
		{"nada que confirmar ni subir", state(func(*branch.PublicationState) {}), nil},
		{"sin rama remota pero sin nada que subir", state(func(s *branch.PublicationState) { s.RemoteBranch = false }), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := projectPublication(tc.facts, `C:\ws`, publicationChange, false)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("projectPublication = %+v, se esperaba %+v", got, tc.want)
			}
		})
	}

	t.Run("el archivado usa la invocación con --archive", func(t *testing.T) {
		got := projectPublication(state(func(s *branch.PublicationState) { s.DirtyDocs = docs }), `C:\ws`, publicationChange, true)
		want := invocation + " --archive"
		if got == nil || got.Invocation != want || !got.Archive {
			t.Errorf("projectPublication = %+v, se esperaba archive con la invocación %q", got, want)
		}
	})
	t.Run("code_unpublished nunca lleva invocación", func(t *testing.T) {
		got := projectPublication(state(func(s *branch.PublicationState) { s.Unpushed, s.CodeCommits = 1, []string{"abc1234"} }), `C:\ws`, publicationChange, false)
		if got == nil || got.Invocation != "" {
			t.Errorf("projectPublication = %+v, un estado informativo no lleva invocation", got)
		}
	})
}

// pushCommand es el formato exacto que `axiom sdd publish` nombra al negarse, con la raíz citada.
func TestPublicationPushCommandQuotesTheRootAndPinsTheRefspec(t *testing.T) {
	got := publicationPushCommand(`C:\Users\dev name\axiom-wt\x\axiom`, "feat/x")
	want := "git -C " + pathquote.Quote(`C:\Users\dev name\axiom-wt\x\axiom`) + " push origin feat/x:feat/x"
	if got != want {
		t.Errorf("publicationPushCommand = %q, se esperaba %q", got, want)
	}
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := append([]string{"-C", dir, "-c", "user.name=Prueba", "-c", "user.email=prueba@ejemplo.invalid", "-c", "commit.gpgsign=false"}, args...)
	if output, err := exec.Command("git", full...).CombinedOutput(); err != nil {
		t.Fatalf("git %v en %s: %v\n%s", args, dir, err, output)
	}
}

func rootKeys(t *testing.T, status any) map[string]json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &keys); err != nil {
		t.Fatal(err)
	}
	return keys
}

// Con el motor real: una compuerta aprobada y un documento sucio proyectan `pending` en la raíz del
// status, con la misma invocación que la pregunta de la compuerta; no cambian la ruta ni los
// bloqueos; y tras publicar la clave desaparece del JSON.
func TestStatusProjectsPendingPublicationWithTheRealEngine(t *testing.T) {
	root := checkpointedMonorepo(t, "main")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "base")
	gitIn(t, root, "switch", "-q", "-c", "feat/x")
	bare := t.TempDir()
	gitIn(t, bare, "init", "-q", "--bare", "-b", "main")
	gitIn(t, root, "remote", "add", "origin", bare)
	gitIn(t, root, "push", "-q", "origin", "main")
	changeRoot := filepath.Join(root, "openspec", "changes", publicationChange)

	// Sin compuertas aprobadas no es el momento: no se proyecta nada, aunque haya algo sucio.
	write(t, filepath.Join(changeRoot, "design.md"), "# Design\n\nCambiado.\n")
	if status := resolveFrom(t, root, publicationChange); status.Publication != nil {
		t.Fatalf("Publication = %+v, sin compuertas aprobadas no procede", status.Publication)
	}

	// Con `spec` aprobada y `design.md` en iteración (compuerta `design` abierta), iterar no
	// publica nada (W1 de C-05-c).
	approveGates(t, changeRoot, kickoff.GateSpec)
	if status := resolveFrom(t, root, publicationChange); status.Publication != nil {
		t.Fatalf("Publication = %+v, con el design en iteración no procede", status.Publication)
	}

	// Aprobados design y tasks, y con apply sin empezar, es el momento de publicar.
	approveGates(t, changeRoot, kickoff.GateDesign, kickoff.GateTasks)
	pending := resolveFrom(t, root, publicationChange)
	if pending.Publication == nil || pending.Publication.State != "pending" {
		t.Fatalf("Publication = %+v, se esperaba pending", pending.Publication)
	}
	if want := []string{"docs_uncommitted", "remote_branch_missing"}; !slices.Equal(pending.Publication.Reasons, want) {
		t.Errorf("reasons = %v, se esperaba %v", pending.Publication.Reasons, want)
	}
	gate, code := resolveGatePublication(publicationChange, root)
	if gate == nil {
		t.Fatalf("resolveGatePublication: %s", code)
	}
	if pending.Publication.Invocation != gate.Invocation {
		t.Errorf("invocation = %q, la de la pregunta de compuerta es %q", pending.Publication.Invocation, gate.Invocation)
	}
	if pending.Publication.Branch != "feat/x" || pending.Publication.Remote != "origin" || pending.Publication.PRTarget != "main" {
		t.Errorf("destino = %+v, se esperaba origin feat/x hacia main", pending.Publication)
	}

	// La clave va en la raíz y no dentro de governance; el status v2 la proyecta igual.
	keys := rootKeys(t, pending)
	if _, ok := keys["publication"]; !ok {
		t.Error("el JSON del status no lleva `publication` en la raíz")
	}
	var governance map[string]json.RawMessage
	if err := json.Unmarshal(keys["governance"], &governance); err != nil {
		t.Fatal(err)
	}
	if _, ok := governance["publication"]; ok {
		t.Error("`publication` no debe ir dentro de `governance`")
	}
	v2, err := ProjectStatusV2(pending)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(v2.Publication, pending.Publication) {
		t.Errorf("el v2 proyecta %+v, se esperaba %+v", v2.Publication, pending.Publication)
	}
	if _, ok := rootKeys(t, v2)["publication"]; !ok {
		t.Error("el JSON del status v2 no lleva `publication` en la raíz")
	}

	// Lo publica quien corresponda; aquí basta confirmar y subir como lo haría `publish`.
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "docs(sdd): publicar los documentos de x")
	gitIn(t, root, "push", "-q", "origin", "feat/x")
	published := resolveFrom(t, root, publicationChange)
	if published.Publication != nil {
		t.Fatalf("Publication = %+v, tras publicar no hay nada pendiente", published.Publication)
	}
	if _, ok := rootKeys(t, published)["publication"]; ok {
		t.Error("sin estado, la clave `publication` no debe aparecer en la raíz del JSON")
	}

	// La proyección es aditiva: no toca la ruta, las dependencias ni los motivos de bloqueo.
	if pending.NextRecommended != published.NextRecommended || !reflect.DeepEqual(pending.Dependencies, published.Dependencies) ||
		!reflect.DeepEqual(pending.BlockedReasons, published.BlockedReasons) || !reflect.DeepEqual(pending.Notes, published.Notes) {
		t.Errorf("la publicación cambió la ruta:\ncon: %s %+v %v\nsin: %s %+v %v",
			pending.NextRecommended, pending.Dependencies, pending.BlockedReasons, published.NextRecommended, published.Dependencies, published.BlockedReasons)
	}
}

// Un commit de código sin subir se informa, con la orden de push para la persona y sin invocación.
func TestStatusProjectsCodeUnpublishedWithTheRealEngine(t *testing.T) {
	root := checkpointedMonorepo(t, "main")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "base")
	gitIn(t, root, "switch", "-q", "-c", "feat/x")
	bare := t.TempDir()
	gitIn(t, bare, "init", "-q", "--bare", "-b", "main")
	gitIn(t, root, "remote", "add", "origin", bare)
	gitIn(t, root, "push", "-q", "origin", "main")
	approveGates(t, filepath.Join(root, "openspec", "changes", publicationChange), kickoff.GateSpec, kickoff.GateDesign, kickoff.GateTasks)
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "docs: compuertas de planificación")
	write(t, filepath.Join(root, "requirements.txt"), "x\n")
	gitIn(t, root, "add", "requirements.txt")
	gitIn(t, root, "commit", "-q", "-m", "dependencias")

	status := resolveFrom(t, root, publicationChange)

	got := status.Publication
	if got == nil || got.State != "code_unpublished" || got.Invocation != "" || len(got.CodeCommits) != 1 {
		t.Fatalf("Publication = %+v, se esperaba code_unpublished con un commit y sin invocación", got)
	}
	if want := "git -C " + pathquote.Quote(root) + " push origin feat/x:feat/x"; got.PushCommand != want {
		t.Errorf("pushCommand = %q, se esperaba %q", got.PushCommand, want)
	}
}
