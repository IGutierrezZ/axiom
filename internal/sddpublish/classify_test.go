package sddpublish

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// Clasificación de las negativas de `git push` y `git commit` con entradas `--porcelain` y `stderr`
// fijas (`LC_ALL=C`), sin ejecutar git (DD-09, paso 10).
func TestClassifyPush(t *testing.T) {
	const b = "inc/x"
	rejected := func(reason string) string {
		return "To /remoto\n!\trefs/heads/inc/x:refs/heads/inc/x\t[rejected] (" + reason + ")\nDone\n"
	}
	cases := []struct {
		name    string
		outcome PushOutcome
		hook    bool
		want    Code
	}{
		{"non-fast-forward", PushOutcome{ExitCode: 1, Stdout: rejected("non-fast-forward")}, false, CodePushNonFastForward},
		{"fetch first", PushOutcome{ExitCode: 1, Stdout: rejected("fetch first")}, false, CodePushNonFastForward},
		{"non-fast-forward con hook pre-push presente", PushOutcome{ExitCode: 1, Stdout: rejected("non-fast-forward")}, true, CodePushNonFastForward},
		{"rechazo remoto", PushOutcome{ExitCode: 1, Stdout: "To /remoto\n!\trefs/heads/inc/x:refs/heads/inc/x\t[remote rejected] (pre-receive hook declined)\nDone\n",
			Stderr: "remote: error: no se admite\nerror: failed to push some refs"}, false, CodePushHookRejected},
		{"rama protegida", PushOutcome{ExitCode: 1, Stdout: "!\trefs/heads/inc/x:refs/heads/inc/x\t[remote rejected] (protected branch hook declined)\n"}, false, CodePushHookRejected},
		{"hook pre-push local", PushOutcome{ExitCode: 1, Stderr: "error: failed to push some refs to '/remoto'"}, true, CodePushHookRejected},
		{"sin lectura del remoto", PushOutcome{ExitCode: 128, Stderr: "fatal: Could not read from remote repository.\n\nPlease make sure you have the correct access rights"}, false, CodeRemoteUnavailable},
		{"no es un repositorio", PushOutcome{ExitCode: 128, Stderr: "fatal: '/x' does not appear to be a git repository\nfatal: Could not read from remote repository."}, true, CodeRemoteUnavailable},
		{"unable to access", PushOutcome{ExitCode: 128, Stderr: "fatal: unable to access 'https://h/r.git/': Could not resolve host: h"}, false, CodeRemoteUnavailable},
		{"autenticación", PushOutcome{ExitCode: 128, Stderr: "remote: Invalid username or password.\nfatal: Authentication failed for 'https://h/r.git/'"}, false, CodeRemoteUnavailable},
		{"sin credenciales interactivas", PushOutcome{ExitCode: 128, Stderr: "fatal: could not read Username for 'https://h': terminal prompts disabled"}, false, CodeRemoteUnavailable},
		{"tiempo máximo", PushOutcome{TimedOut: true}, true, CodeRemoteUnavailable},
		{"otro rechazo", PushOutcome{ExitCode: 1, Stdout: "!\trefs/heads/inc/x:refs/heads/inc/x\t[rejected] (already exists)\n"}, false, CodePushFailed},
		{"cualquier otro fallo", PushOutcome{ExitCode: 1, Stderr: "error: src refspec does not match any"}, false, CodePushFailed},
		{"128 sin señal conocida", PushOutcome{ExitCode: 128, Stderr: "fatal: algo raro"}, false, CodePushFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			refusal := classifyPush(tc.outcome, tc.hook, 2*time.Minute, b)
			if refusal.Code != tc.want {
				t.Fatalf("código %q (%s), se esperaba %q", refusal.Code, refusal.Message, tc.want)
			}
			if refusal.Label == "" || refusal.Message == "" || refusal.Continuation == "" {
				t.Errorf("la negativa debe llevar etiqueta, mensaje y continuación: %+v", refusal)
			}
			if tc.want == CodeRemoteUnavailable && tc.outcome.TimedOut && refusal.Details["timeout"] != "2m0s" {
				t.Errorf("details.timeout = %v, se esperaba 2m0s", refusal.Details["timeout"])
			}
			for _, forbidden := range []string{"--force", "--no-verify"} {
				if strings.Contains(refusal.Continuation, forbidden) {
					t.Errorf("la continuación %q no debe proponer %s", refusal.Continuation, forbidden)
				}
			}
		})
	}
}

func TestClassifyCommit(t *testing.T) {
	cases := []struct {
		name string
		fail CommandError
		hook bool
		want Code
	}{
		{"hook que rechaza", CommandError{Step: "commit", ExitCode: 1, Stderr: "lint falló"}, true, CodeCommitHookRejected},
		{"fallo sin hook", CommandError{Step: "commit", ExitCode: 128, Stderr: "fatal: no se pudo escribir el índice"}, false, CodeCommitFailed},
		{"fallo del add", CommandError{Step: "add", ExitCode: 128, Stderr: "fatal: Unable to create index.lock"}, false, CodeCommitFailed},
		{"identidad desconocida", CommandError{Step: "commit", ExitCode: 128, Stderr: "Author identity unknown\n\n*** Please tell me who you are."}, true, CodeIdentityMissing},
		{"sin detectar el correo", CommandError{Step: "commit", ExitCode: 128, Stderr: "fatal: unable to auto-detect email address (got 'x@y.(none)')"}, false, CodeIdentityMissing},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			refusal := classifyCommit(&tc.fail, tc.hook)
			if refusal.Code != tc.want {
				t.Fatalf("código %q (%s), se esperaba %q", refusal.Code, refusal.Message, tc.want)
			}
			if tc.want != CodeIdentityMissing && !strings.Contains(refusal.Message, "preparadas") {
				t.Errorf("el mensaje %q debe avisar de que las rutas pueden haber quedado preparadas", refusal.Message)
			}
			if refusal.Continuation == "" || refusal.Label == "" {
				t.Errorf("falta la continuación o la etiqueta: %+v", refusal)
			}
		})
	}
}

func TestClassifyKeepsOnlyTheTailOfStderr(t *testing.T) {
	var lines []string
	for i := 1; i <= 30; i++ {
		lines = append(lines, "línea "+string(rune('a'+i%26)))
	}
	tail := stderrTail(strings.Join(lines, "\n"))
	if got := strings.Count(tail, "|") + 1; got != 8 {
		t.Errorf("la cola conserva 8 líneas, hay %d: %q", got, tail)
	}
}

// Los códigos propios tienen etiqueta en castellano y los reenviados de `branch` conservan la suya.
func TestResultLabels(t *testing.T) {
	for code := range labels {
		if Label(code) == "" || Label(code) == string(code) {
			t.Errorf("el código %s no tiene etiqueta en castellano", code)
		}
	}
	if got, want := Label(Code(branch.CodeDetachedHead)), "HEAD desacoplado"; got != want {
		t.Errorf("Label(detached_head) = %q, se esperaba %q", got, want)
	}
	if got, want := Label(Code(branch.CodeIntegrationBranch)), "rama de integración no publicable"; got != want {
		t.Errorf("Label(integration_branch) = %q, se esperaba %q", got, want)
	}
	forwarded := fromBranch(&branch.Refusal{Code: branch.CodeWrongBranch, Message: "m", Continuation: "c", Details: map[string]string{"k": "v"}})
	if forwarded.Code != "wrong_branch" || forwarded.Label == "" || forwarded.Details["k"] != "v" {
		t.Errorf("fromBranch = %+v", forwarded)
	}
}

// El JSON es estable: `commit` y `push` se omiten cuando no ocurrieron y `refusal` es null sin negativa.
func TestResultJSONIsStable(t *testing.T) {
	published := Result{
		Schema: Schema, Change: "x", Context: ContextChange, Status: StatusPublished,
		Repository: &RepositoryRef{Name: "axiom", GitRoot: "/r"}, Remote: Remote, Branch: "inc/x",
		Commit:             &CommitResult{SHA: "abc", Message: "docs(sdd): publicar los documentos de x", Paths: []string{"openspec/changes/x/design.md"}},
		Push:               &PushResult{Refspec: "refs/heads/inc/x:refs/heads/inc/x", SHA: "abc", Created: true},
		UnpublishedCommits: []string{"abc"}, Resume: ResumeSteps("x"),
	}
	data, err := json.Marshal(published)
	if err != nil {
		t.Fatal(err)
	}
	const wantPublished = `{"schema":"axiom.sdd-publish/v1","change":"x","context":"change","status":"published","repository":{"name":"axiom","gitRoot":"/r"},"remote":"origin","branch":"inc/x","inPlace":false,` +
		`"commit":{"sha":"abc","message":"docs(sdd): publicar los documentos de x","paths":["openspec/changes/x/design.md"]},` +
		`"push":{"refspec":"refs/heads/inc/x:refs/heads/inc/x","sha":"abc","created":true},"unpublishedCommits":["abc"],` +
		`"resume":["git fetch origin","axiom branch ensure --change x --json"],"refusal":null}`
	if string(data) != wantPublished {
		t.Errorf("JSON publicado:\n got %s\nwant %s", data, wantPublished)
	}

	refused := Result{Schema: Schema, Change: "x", Context: ContextArchive, Status: StatusRefused, Remote: Remote,
		Refusal: refuse(CodeRemoteMissing, "m", "c", nil)}
	data, err = json.Marshal(refused)
	if err != nil {
		t.Fatal(err)
	}
	const wantRefused = `{"schema":"axiom.sdd-publish/v1","change":"x","context":"archive","status":"refused","remote":"origin","inPlace":false,` +
		`"refusal":{"code":"remote_missing","label":"no existe el remoto origin","message":"m","continuation":"c"}}`
	if string(data) != wantRefused {
		t.Errorf("JSON rechazado:\n got %s\nwant %s", data, wantRefused)
	}
	if published.ExitCode() != 0 || refused.ExitCode() != 1 || (Result{Status: StatusUpToDate}).ExitCode() != 0 {
		t.Error("el código de salida es 0 con published y up_to_date y 1 con refused")
	}
}
