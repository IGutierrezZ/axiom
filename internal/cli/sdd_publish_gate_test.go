package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Pruebas de integración de `axiom sdd publish` encadenado tras `axiom sdd gate record` (REQ-27.8,
// REQ-27.11, REQ-27.12, REQ-27.15; DD-11, DD-14): en proceso, con git real, un remoto desnudo
// temporal local y sin red. `gate record` solo registra y calcula la orden; `publish` la ejecuta y
// es el único que confirma y sube.

// publishGateProject es el proyecto de una parada con un remoto desnudo `origin` que ya tiene `main`:
// la rama del cambio `feat/x` aún no está publicada.
func publishGateProject(t *testing.T) (ws workspacetest.Workspace, bare string) {
	t.Helper()
	ws = handOffProject(t, guardSealArgs)
	bare = t.TempDir()
	integrateGit(t, bare, "init", "-q", "--bare", "-b", "main")
	integrateGit(t, ws.Master, "remote", "add", "origin", bare)
	integrateGit(t, ws.Master, "push", "-q", "origin", "main")
	return ws, bare
}

// publishInvocationArgs convierte la invocaciÃ³n `axiom sdd publish ...` que devuelve `gate record` en los
// argumentos de RunSDDPublish.
func publishInvocationArgs(t *testing.T, invocation string) []string {
	t.Helper()
	// La ruta se cita con comillas dobles y, en Windows, lleva barras invertidas que no son escapes:
	// se reparte a mano, sin interpretar escapes.
	const prefix = "axiom sdd publish "
	if !strings.HasPrefix(invocation, prefix) {
		t.Fatalf("la invocaciÃ³n %q no es un `axiom sdd publish`", invocation)
	}
	var args []string
	var word strings.Builder
	quoted, started := false, false
	for _, r := range strings.TrimPrefix(invocation, prefix) {
		switch {
		case r == '"':
			quoted, started = !quoted, true
		case r == ' ' && !quoted:
			if started {
				args = append(args, word.String())
				word.Reset()
				started = false
			}
		default:
			word.WriteRune(r)
			started = true
		}
	}
	if started {
		args = append(args, word.String())
	}
	return args
}

type gateRecordPublishJSON struct {
	Publish *struct {
		Remote     string `json:"remote"`
		Branch     string `json:"branch"`
		PRTarget   string `json:"prTarget"`
		Invocation string `json:"invocation"`
	} `json:"publish"`
	PublishUnavailable string `json:"publishUnavailable"`
}

func recordForPublish(t *testing.T, ws workspacetest.Workspace, extra ...string) gateRecordPublishJSON {
	t.Helper()
	out, err := gateRecord(t, ws, append(extra, "--json")...)
	if err != nil {
		t.Fatalf("gate record %v: %v", extra, err)
	}
	var document gateRecordPublishJSON
	if err := json.Unmarshal([]byte(out), &document); err != nil {
		t.Fatalf("gate record --json: %v\n%s", err, out)
	}
	if document.Publish == nil {
		t.Fatalf("la aprobación debe devolver publish (publishUnavailable=%q):\n%s", document.PublishUnavailable, out)
	}
	return document
}

// REQ-27.12 y REQ-27.11: la salida 2 registra con `gate record` y la invocación que devuelve publica
// `gates.yaml` y los documentos de la carpeta del cambio, sin subir el código que hay en el árbol.
func TestSDDPublishGateChainsAfterHandOffAndPublishesOnlyDocuments(t *testing.T) {
	ws, bare := publishGateProject(t)
	// Código sin confirmar y sin seguimiento: no es documentación y no viaja en ningún push.
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "internal", "core.go"), "package core\n")

	document := recordForPublish(t, ws, "--gate", "spec", "--decision", "approved", "--hand-off")
	if document.Publish.Branch != "feat/x" || document.Publish.PRTarget != "main" || document.Publish.Remote != "origin" {
		t.Fatalf("publish = %+v", document.Publish)
	}

	var out bytes.Buffer
	if err := RunSDDPublish(append(publishInvocationArgs(t, document.Publish.Invocation), "--json"), &out); err != nil {
		t.Fatalf("publish: %v\n%s", err, out.String())
	}
	result := decodePublish(t, out.String())
	if result.Status != "published" || result.Branch != "feat/x" || result.Push == nil || !result.Push.Created {
		t.Fatalf("resultado = %+v", result)
	}
	if result.Commit == nil || !slices.Contains(result.Commit.Paths, "openspec/changes/x/gates.yaml") {
		t.Fatalf("el commit debe llevar gates.yaml: %+v", result.Commit)
	}
	for _, path := range result.Commit.Paths {
		if !strings.HasPrefix(path, "openspec/changes/x/") {
			t.Errorf("el commit solo lleva documentos del cambio, y lleva %q", path)
		}
	}

	published := publishGitOutput(t, bare, "show", "feat/x:openspec/changes/x/gates.yaml")
	if !strings.Contains(published, "handed_off: true") {
		t.Errorf("el remoto debe tener gates.yaml con la cesión:\n%s", published)
	}
	if out, err := publishGitOutputIn(bare, "cat-file", "-e", "feat/x:internal/core.go"); err == nil {
		t.Errorf("el código no se sube: %s", out)
	}
	if _, err := os.Stat(filepath.Join(ws.Master, "internal", "core.go")); err != nil {
		t.Errorf("el código local debe seguir intacto: %v", err)
	}
	if status := strings.TrimSpace(publishGitOutput(t, ws.Master, "status", "--porcelain", "--", "internal")); !strings.HasPrefix(status, "??") {
		t.Errorf("el código sigue sin confirmar y sin preparar: %q", status)
	}
}

// REQ-27.8 y REQ-26.4: un role-apply con un commit de código sin subir se registra igualmente (la
// cesión no inspecciona el trabajo del rol) y `publish` informa `code_commits_unpublished` sin
// deshacer la aprobación: el commit de documentación se conserva en local.
func TestSDDPublishGateRefusesUnpublishedCodeAndKeepsTheRecordedApproval(t *testing.T) {
	ws, bare := publishGateProject(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "internal", "core.go"), "package core\n")
	integrateCommit(t, ws.Master, "feat(core): trabajo del rol")

	document := recordForPublish(t, ws, "--gate", "role-apply:core", "--decision", "approved", "--hand-off")

	var out bytes.Buffer
	err := RunSDDPublish(append(publishInvocationArgs(t, document.Publish.Invocation), "--json"), &out)
	if err == nil {
		t.Fatalf("publish debía negarse por el código sin subir:\n%s", out.String())
	}
	result := decodePublish(t, out.String())
	if result.Status != "refused" || result.Refusal == nil || result.Refusal.Code != "code_commits_unpublished" {
		t.Fatalf("resultado = %+v", result)
	}
	if !strings.Contains(result.Refusal.Continuation, "git -C") || !strings.Contains(result.Refusal.Continuation, "push origin feat/x:feat/x") {
		t.Errorf("la continuación debe ser el push de código para la persona: %q", result.Refusal.Continuation)
	}
	if result.Commit == nil {
		t.Error("el commit de documentación se conserva e informa")
	}

	ledger := handOffLedger(t, ws)
	approval, approved := ledger.Approval(kickoff.RoleApplyGate("core"))
	if !approved || !approval.HandedOff {
		t.Fatalf("la aprobación registrada se conserva con la marca: %+v", ledger.Records)
	}
	if out, err := publishGitOutputIn(bare, "rev-parse", "--verify", "--quiet", "refs/heads/feat/x"); err == nil {
		t.Errorf("la negativa previa al push no sube nada: %s", out)
	}
}

// REQ-27.15: quien sigue hace `git fetch origin` en otra copia y `axiom branch ensure` adopta la rama
// publicada (`createdFrom: remote`), sin PR fusionada.
func TestSDDPublishGateWhoFollowsAdoptsThePublishedBranchWithEnsure(t *testing.T) {
	ws, bare := publishGateProject(t)
	document := recordForPublish(t, ws, "--gate", "spec", "--decision", "approved", "--hand-off")
	if err := RunSDDPublish(publishInvocationArgs(t, document.Publish.Invocation), io.Discard); err != nil {
		t.Fatalf("publish: %v", err)
	}

	second := filepath.Join(t.TempDir(), "segunda")
	integrateGit(t, filepath.Dir(second), "clone", "-q", bare, second)
	integrateGit(t, second, "fetch", "origin")
	if _, err := publishGitOutputIn(second, "rev-parse", "--verify", "--quiet", "refs/heads/feat/x"); err == nil {
		t.Fatal("la segunda copia aún no tiene la rama local feat/x")
	}

	var stdout, stderr bytes.Buffer
	if code := RunBranch([]string{"ensure", "--change", "x", "--mode", "in_place", "--cwd", second, "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("ensure en la segunda copia: código %d\n%s\n%s", code, stdout.String(), stderr.String())
	}
	var result any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("ensure --json: %v\n%s", err, stdout.String())
	}
	if !containsCreatedFromRemote(result) {
		t.Errorf("ensure debe adoptar la rama publicada con createdFrom remote:\n%s", stdout.String())
	}
	if got := strings.TrimSpace(publishGitOutput(t, second, "rev-parse", "refs/heads/feat/x")); got != strings.TrimSpace(publishGitOutput(t, bare, "rev-parse", "refs/heads/feat/x")) {
		t.Errorf("la rama local adoptada debe apuntar al SHA publicado: %s", got)
	}
}

func containsCreatedFromRemote(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		if typed["createdFrom"] == "remote" {
			return true
		}
		for _, inner := range typed {
			if containsCreatedFromRemote(inner) {
				return true
			}
		}
	case []any:
		for _, inner := range typed {
			if containsCreatedFromRemote(inner) {
				return true
			}
		}
	}
	return false
}

// REQ-27.11: ni `gate record` ni los verbos de `axiom branch` ejecutan `push` o `commit`: solo
// `publish` lo hace. Observador dinámico de la cadena completa.
func TestSDDPublishGateOnlyPublishRunsCommitAndPush(t *testing.T) {
	ws, _ := publishGateProject(t)

	var recorded [][]string
	gitcommon.Observe = func(dir string, args []string) { recorded = append(recorded, append([]string{dir}, args...)) }
	t.Cleanup(func() { gitcommon.Observe = nil })

	document := recordForPublish(t, ws, "--gate", "spec", "--decision", "approved", "--hand-off")
	for _, invocation := range recorded {
		if slices.Contains([]string{"push", "commit", "fetch", "pull", "ls-remote"}, invocation[1]) {
			t.Fatalf("gate record no puede ejecutar %s: %v", invocation[1], invocation)
		}
	}
	recorded = nil

	if err := RunSDDPublish(publishInvocationArgs(t, document.Publish.Invocation), io.Discard); err != nil {
		t.Fatalf("publish: %v", err)
	}
	var pushed, committed bool
	for _, invocation := range recorded {
		switch invocation[1] {
		case "push":
			pushed = true
		case "commit":
			committed = true
		case "fetch", "pull", "ls-remote":
			t.Errorf("publish no hace operaciones remotas de lectura: %v", invocation)
		}
	}
	if !pushed || !committed {
		t.Errorf("publish debe confirmar y subir (commit=%v push=%v)", committed, pushed)
	}
}
