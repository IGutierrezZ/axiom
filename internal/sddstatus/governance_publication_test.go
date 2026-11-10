package sddstatus

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Pruebas de la publicación que lleva la pregunta de una compuerta de planificación (REQ-27.7,
// REQ-27.12, DD-11): el destino se calcula con el motor de ramas, solo con lecturas.

// checkpointedMonorepo monta un monorepo embebido con un kickoff checkpointed y la instantánea de
// política git, con el HEAD en la rama dada y la compuerta spec abierta.
func checkpointedMonorepo(t *testing.T, head string) string {
	t.Helper()
	useBranchGuard(t)
	ws := workspacetest.Embedded(t, ".", workspacetest.WithGit())
	root := ws.Master
	switchTo(t, root, head)
	changeRoot := seedReadyChange(t, root, branchChange, branchPlan)
	_, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: branchChange, SealedBy: "prueba",
		Config: kickoff.FlowConfig{
			FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionCheckpointed, ExecutionStyleSource: "explicit",
			HandoffPolicy: kickoff.HandoffNone,
			Roles:         []kickoff.KickoffRole{{Role: "fullstack", GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"}},
		},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
		Git:       monorepoSnapshot(workspaceName(root)),
	})
	if err != nil || !ok {
		t.Fatalf("sellar %s: %v (publicado: %v)", branchChange, err, ok)
	}
	return root
}

func workspaceName(root string) string {
	parts := strings.Split(strings.ReplaceAll(root, "\\", "/"), "/")
	return parts[len(parts)-1]
}

func TestGateQuestionCarriesThePublicationOfTheCurrentBranch(t *testing.T) {
	root := checkpointedMonorepo(t, "feat/x")

	status := resolveFrom(t, root, branchChange)

	question := status.GateQuestion
	if question == nil {
		t.Fatal("GateQuestion = nil, se esperaba la pregunta de la compuerta abierta")
	}
	if question.PublicationUnavailable != "" || question.Publication == nil {
		t.Fatalf("publication = %+v / %q, se esperaba el destino resuelto", question.Publication, question.PublicationUnavailable)
	}
	want := GatePublication{
		Remote:     "origin",
		Branch:     "feat/x",
		PRTarget:   "main",
		Invocation: "axiom sdd publish --cwd " + pathquote.Quote(root) + " --change x --branch feat/x",
	}
	if *question.Publication != want {
		t.Errorf("publication = %+v, se esperaba %+v", *question.Publication, want)
	}
	labels := map[string]string{}
	for _, choice := range question.Choices {
		labels[choice.Answer] = choice.Label
		if !strings.HasPrefix(choice.Invocation, gateRecordInvocationPrefix) {
			t.Errorf("la invocación de %s debe seguir siendo `gate record`: %q", choice.Answer, choice.Invocation)
		}
		if choice.Answer == gateAnswerRejected {
			continue
		}
		if !strings.Contains(choice.Effect, "feat/x") {
			t.Errorf("el efecto de %s debe nombrar la rama: %q", choice.Answer, choice.Effect)
		}
		for _, fragment := range []string{"A refused publication keeps the approval.", "Code pushes, pull requests and merges remain yours."} {
			if !strings.Contains(choice.Effect, fragment) {
				t.Errorf("el efecto de %s debe contener %q: %q", choice.Answer, fragment, choice.Effect)
			}
		}
	}
	if labels[gateAnswerHandOff] != "Approve, publish and stop" || labels[gateAnswerApproved] != "Approve, publish and continue" || labels[gateAnswerRejected] != "Iterate on this artifact" {
		t.Errorf("etiquetas = %v", labels)
	}
	if !strings.Contains(question.Value, "origin/feat/x") || !strings.Contains(question.Value, "remain yours") {
		t.Errorf("el valor debe nombrar el remoto y la rama: %q", question.Value)
	}

	encoded, err := json.Marshal(question)
	if err != nil || !strings.Contains(string(encoded), `"publication":{"remote":"origin","branch":"feat/x"`) {
		t.Fatalf("serialización = %s (err %v), se esperaba publication", encoded, err)
	}
}

// REQ-27.7: con la rama equivocada la pregunta lleva `publicationUnavailable` con el código y las
// opciones siguen siendo las de `gate record`.
func TestGateQuestionReportsPublicationUnavailableOnTheWrongBranch(t *testing.T) {
	root := checkpointedMonorepo(t, "feat/x")
	spec := kickoff.GateState{Key: kickoff.GateSpec}

	question, err := newGovernanceGateQuestion(branchChange, root, spec, ArtifactPaths{}, ArtifactStoreOpenSpec)
	if err != nil {
		t.Fatal(err)
	}
	if question.Publication == nil {
		t.Fatal("en feat/x la publicación debe resolverse")
	}

	switchTo(t, root, "main")
	question, err = newGovernanceGateQuestion(branchChange, root, spec, ArtifactPaths{}, ArtifactStoreOpenSpec)
	if err != nil {
		t.Fatal(err)
	}
	if question.Publication != nil || question.PublicationUnavailable == "" {
		t.Fatalf("publication = %+v / %q, se esperaba publicationUnavailable", question.Publication, question.PublicationUnavailable)
	}
	if got := strings.Join(gateAnswers(question), ","); got != "rejected,hand_off,approved" {
		t.Errorf("respuestas = %s, las opciones siguen siendo las de gate record", got)
	}
	for _, choice := range question.Choices {
		if !strings.HasPrefix(choice.Invocation, gateRecordInvocationPrefix) {
			t.Errorf("la invocación de %s debe ser `gate record`: %q", choice.Answer, choice.Invocation)
		}
	}
}

// REQ-26.6 e integración: con un almacén sin ficheros y en la compuerta de integración no hay
// publicación ni etiquetas que la prometan.
func TestGateQuestionCarriesNoPublicationForFilelessStoresOrIntegration(t *testing.T) {
	root := checkpointedMonorepo(t, "feat/x")
	for _, tc := range []struct {
		name  string
		gate  kickoff.GateKey
		store ArtifactStore
	}{
		{"engram", kickoff.GateSpec, ArtifactStoreEngram},
		{"none", kickoff.GateDesign, ArtifactStoreNone},
		{"integration", kickoff.GateIntegration, ArtifactStoreOpenSpec},
	} {
		t.Run(tc.name, func(t *testing.T) {
			question, err := newGovernanceGateQuestion(branchChange, root, kickoff.GateState{Key: tc.gate}, ArtifactPaths{}, tc.store)
			if err != nil {
				t.Fatal(err)
			}
			if question.Publication != nil || question.PublicationUnavailable != "" {
				t.Errorf("publication = %+v / %q, se esperaba ninguna", question.Publication, question.PublicationUnavailable)
			}
			for _, choice := range question.Choices {
				if strings.Contains(choice.Label, "publish") {
					t.Errorf("la etiqueta %q no puede prometer publicación", choice.Label)
				}
			}
		})
	}
}

func TestGateQuestionValidateRejectsAPublicationNotBoundToItsBranch(t *testing.T) {
	root := checkpointedMonorepo(t, "feat/x")
	question, err := newGovernanceGateQuestion(branchChange, root, kickoff.GateState{Key: kickoff.GateSpec}, ArtifactPaths{}, ArtifactStoreOpenSpec)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*SDDGovernanceGateResult){
		"sin --branch": func(r *SDDGovernanceGateResult) { r.Publication.Invocation = "axiom sdd publish --cwd /r --change x" },
		"otra rama": func(r *SDDGovernanceGateResult) {
			r.Publication.Invocation = "axiom sdd publish --cwd /r --change x --branch otra"
		},
		"prefijo ajeno":       func(r *SDDGovernanceGateResult) { r.Publication.Invocation = "axiom sdd gate record --branch feat/x" },
		"ambas informaciones": func(r *SDDGovernanceGateResult) { r.PublicationUnavailable = "wrong_branch" },
	} {
		t.Run(name, func(t *testing.T) {
			copyOf := question
			published := *question.Publication
			copyOf.Publication = &published
			mutate(&copyOf)
			if err := copyOf.Validate(); err == nil {
				t.Fatal("Validate() = nil, se esperaba rechazo")
			}
		})
	}
}
