package branch

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// allCodes son los dieciséis códigos tipados de la spec (design §2.2).
var allCodes = []Code{
	CodeDecisionRequired, CodeDirtyWorktree, CodeWrongBranch, CodeSpecsRepoNotOnIntegration,
	CodeBaseNotFound, CodeFetchFailed, CodePathTooLong, CodeInvalidName, CodeConfigInvalid,
	CodeGitFailed, CodePlanConflict, CodeBranchCheckedOutElsewhere, CodeNotAGitRepository,
	CodeMergeNotConfirmed, CodeChangeNotIntegrated, CodeArchiveBranchNotApplicable,
}

func resultJSON(t *testing.T, result Result) (raw string, decoded map[string]any) {
	t.Helper()
	var buffer bytes.Buffer
	if err := result.WriteJSON(&buffer); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if err := json.Unmarshal(buffer.Bytes(), &decoded); err != nil {
		t.Fatalf("la salida no es JSON: %v\n%s", err, buffer.String())
	}
	return buffer.String(), decoded
}

func TestNewResultIsReadyAndNamesItsSchemaAndOperation(t *testing.T) {
	_, decoded := resultJSON(t, NewResult("status"))

	want := map[string]any{
		"schema":      "axiom.branch-result/v1",
		"operation":   "status",
		"status":      "ready",
		"statusLabel": "listo",
	}
	for key, value := range want {
		if decoded[key] != value {
			t.Errorf("%s = %v, se esperaba %v", key, decoded[key], value)
		}
	}
	for _, absent := range []string{"code", "message", "continuation", "details", "repositories", "warnings"} {
		if _, present := decoded[absent]; present {
			t.Errorf("un resultado listo y vacío no lleva la clave %q: %v", absent, decoded)
		}
	}
}

// REQ-25.2: `code` aparece cuando el estado es `refused`, `decision_required`,
// `partial` o `error`, y es uno de los dieciséis códigos. Cada estado lleva su token
// con su etiqueta.
func TestRefusedMapsEveryCodeToItsStatusTokenAndLabel(t *testing.T) {
	for _, code := range allCodes {
		t.Run(string(code), func(t *testing.T) {
			wantStatus, wantLabel := "refused", "rechazado"
			switch code {
			case CodeDecisionRequired:
				wantStatus, wantLabel = "decision_required", "decisión requerida"
			case CodeGitFailed:
				wantStatus, wantLabel = "error", "error"
			}
			refusal := &Refusal{Code: code, Message: "mensaje", Continuation: "axiom branch status --help"}
			_, decoded := resultJSON(t, NewResult("status").Refused(refusal))

			if decoded["status"] != wantStatus || decoded["statusLabel"] != wantLabel {
				t.Errorf("status %v / statusLabel %v, se esperaba %s / %s", decoded["status"], decoded["statusLabel"], wantStatus, wantLabel)
			}
			if decoded["code"] != string(code) {
				t.Errorf("code = %v, se esperaba %s", decoded["code"], code)
			}
			if decoded["message"] != "mensaje" || decoded["continuation"] != "axiom branch status --help" {
				t.Errorf("message %v, continuation %v; se esperaba el mensaje y la continuación del rechazo", decoded["message"], decoded["continuation"])
			}
		})
	}
}

// REQ-25.2: «0 únicamente cuando la operación ha dejado el estado pedido»: `ready` y
// `not_started` salen con 0; todo rechazo, `decision_required`, `partial` y `error`
// con 1. El 2 es de los errores de uso, que no producen un resultado.
func TestResultExitCodeIsZeroOnlyWhenTheRequestedStateWasReached(t *testing.T) {
	tests := []struct {
		status Status
		want   int
	}{
		{StatusReady, 0},
		{StatusNotStarted, 0},
		{StatusRefused, 1},
		{StatusDecisionRequired, 1},
		{StatusPartial, 1},
		{StatusError, 1},
	}
	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			if got := (Result{Status: tc.status}).ExitCode(); got != tc.want {
				t.Errorf("ExitCode de %s = %d, se esperaba %d", tc.status, got, tc.want)
			}
		})
	}
	if ExitReady != 0 || ExitRefused != 1 || ExitUsage != 2 {
		t.Errorf("códigos de salida %d, %d, %d; se esperaban 0, 1 y 2", ExitReady, ExitRefused, ExitUsage)
	}
}

// REQ-25.2: serialización estable sin marcas de tiempo. Las claves de `details` salen
// ordenadas y dos escrituras del mismo resultado dan los mismos bytes.
func TestResultSerialisationIsStableAndCarriesNoTimestamps(t *testing.T) {
	result := NewResult("status").Refused(&Refusal{
		Code:         CodeGitFailed,
		Message:      "git falló",
		Continuation: "axiom branch status --help",
		Details:      map[string]string{"summary": "s", "operation": "o", "alfa": "a"},
	})

	first, _ := resultJSON(t, result)
	second, _ := resultJSON(t, result)
	if first != second {
		t.Errorf("dos escrituras difieren:\n%s\n---\n%s", first, second)
	}
	if !strings.HasSuffix(first, "\n") {
		t.Errorf("la salida JSON debe terminar en un salto de línea: %q", first)
	}
	if alfa, operation, summary := strings.Index(first, `"alfa"`), strings.Index(first, `"operation": "o"`), strings.Index(first, `"summary"`); !(alfa < operation && operation < summary) {
		t.Errorf("las claves de details deben salir ordenadas:\n%s", first)
	}
	for _, timeKey := range []string{"time", "date", "timestamp", "createdAt", "updatedAt"} {
		if strings.Contains(strings.ToLower(first), strings.ToLower(timeKey)) {
			t.Errorf("la salida no lleva marcas de tiempo (%q):\n%s", timeKey, first)
		}
	}
}

// REQ-25.2: claves en camelCase, también en los objetos anidados.
func TestResultKeysAreCamelCase(t *testing.T) {
	result := NewResult("status").Refused(&Refusal{Code: CodeGitFailed, Message: "m", Continuation: "axiom branch status --help", Details: map[string]string{"operation": "o"}})
	result.Repositories = []RepositoryResult{{GitRoot: "C:\\repo", CommonDir: "C:\\repo\\.git", IntegrationBranch: "main", CurrentBranch: "main", Detached: true}}
	result.Warnings = []string{"aviso"}

	_, decoded := resultJSON(t, result)
	var check func(path string, value any)
	check = func(path string, value any) {
		switch typed := value.(type) {
		case map[string]any:
			for key, nested := range typed {
				if path != "details" && (key == "" || !unicode.IsLower(rune(key[0])) || strings.ContainsAny(key, "_- ")) {
					t.Errorf("la clave %q de %q no es camelCase", key, path)
				}
				check(key, nested)
			}
		case []any:
			for _, nested := range typed {
				check(path, nested)
			}
		}
	}
	check("", decoded)
	for _, want := range []string{"gitRoot", "commonDir", "integrationBranch", "currentBranch", "detached"} {
		if !strings.Contains(mustMarshal(t, result), `"`+want+`"`) {
			t.Errorf("la salida debe incluir la clave %q", want)
		}
	}
}

func mustMarshal(t *testing.T, result Result) string {
	t.Helper()
	raw, _ := resultJSON(t, result)
	return raw
}

// La salida JSON no escapa HTML: una ruta o un mensaje con `<`, `>` o `&` sale tal cual.
func TestResultWriteJSONDoesNotEscapeHTML(t *testing.T) {
	result := NewResult("status").Refused(&Refusal{Code: CodeInvalidName, Message: `a<b>&"c"`, Continuation: "axiom branch status --help"})
	raw, decoded := resultJSON(t, result)
	if strings.Contains(raw, `\u003c`) || strings.Contains(raw, `\u0026`) {
		t.Errorf("la salida escapó HTML: %s", raw)
	}
	if decoded["message"] != `a<b>&"c"` {
		t.Errorf("message = %v", decoded["message"])
	}
}

func TestWriteTextPrintsReadyResultsToStdoutInSpanish(t *testing.T) {
	result := NewResult("status")
	result.Repositories = []RepositoryResult{{GitRoot: "C:\\repo", CommonDir: "C:\\repo\\.git", IntegrationBranch: "develop", CurrentBranch: "main"}}
	result.Warnings = []string{"algo que revisar"}

	var stdout, stderr bytes.Buffer
	result.WriteText(&stdout, &stderr)

	if stderr.Len() != 0 {
		t.Errorf("un resultado listo no escribe en stderr: %q", stderr.String())
	}
	for _, want := range []string{
		"Repositorio git: C:\\repo\n",
		"Directorio común de git: C:\\repo\\.git\n",
		"Rama actual: main\n",
		"Rama de integración: develop\n",
		"Aviso: algo que revisar\n",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("la salida de texto debe contener %q:\n%s", want, stdout.String())
		}
	}
}

func TestWriteTextNamesADetachedHeadAndOmitsAnUnknownBranch(t *testing.T) {
	detached := NewResult("status")
	detached.Repositories = []RepositoryResult{{GitRoot: "r", CommonDir: "c", IntegrationBranch: "main", Detached: true}}
	unknown := NewResult("status")
	unknown.Repositories = []RepositoryResult{{GitRoot: "r", CommonDir: "c", IntegrationBranch: "main"}}

	var out bytes.Buffer
	detached.WriteText(&out, &bytes.Buffer{})
	if !strings.Contains(out.String(), "Rama actual: ninguna (HEAD desacoplado)") {
		t.Errorf("un HEAD desacoplado se nombra:\n%s", out.String())
	}
	out.Reset()
	unknown.WriteText(&out, &bytes.Buffer{})
	if strings.Contains(out.String(), "Rama actual") {
		t.Errorf("sin rama conocida no se imprime «Rama actual»:\n%s", out.String())
	}
}

func TestResultWriteTextPrintsRefusalsToStderrWithTheContinuation(t *testing.T) {
	result := NewResult("status").Refused(NotAGitRepository("status", `C:\sin git`))

	var stdout, stderr bytes.Buffer
	result.WriteText(&stdout, &stderr)

	if stdout.Len() != 0 {
		t.Errorf("un rechazo no escribe en stdout: %q", stdout.String())
	}
	for _, want := range []string{"Error [not_a_git_repository]:", `C:\sin git`, "Continuación: ", "axiom branch status --cwd"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("el rechazo en texto debe contener %q:\n%s", want, stderr.String())
		}
	}
}

// REQ-25.2: `git_failed` nombra la operación y resume el error.
func TestResultGitFailedNamesTheOperationAndSummarisesTheError(t *testing.T) {
	refusal := GitFailed("status", `C:\repo`, "localizar la raíz del repositorio", errors.New("git rev-parse terminó con el código 128: fatal: dubious ownership"))

	if refusal.Code != CodeGitFailed {
		t.Errorf("code = %s, se esperaba git_failed", refusal.Code)
	}
	for _, want := range []string{"localizar la raíz del repositorio", "dubious ownership"} {
		if !strings.Contains(refusal.Message, want) {
			t.Errorf("el mensaje %q debe contener %q", refusal.Message, want)
		}
	}
	if refusal.Details["operation"] != "localizar la raíz del repositorio" || !strings.Contains(refusal.Details["summary"], "dubious ownership") {
		t.Errorf("details = %v; se esperaba la operación y el resumen", refusal.Details)
	}
	for _, want := range []string{"axiom branch status --cwd", pathquote.Quote(`C:\repo`)} {
		if !strings.Contains(refusal.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", refusal.Continuation, want)
		}
	}
}

func TestResultNotAGitRepositoryQuotesThePathInItsContinuation(t *testing.T) {
	dir := `C:\Users\Ana Pérez\sin git`
	refusal := NotAGitRepository("status", dir)

	if refusal.Code != CodeNotAGitRepository || !strings.Contains(refusal.Message, dir) {
		t.Errorf("code %s, mensaje %q; se esperaba not_a_git_repository y un mensaje que nombre %s", refusal.Code, refusal.Message, dir)
	}
	for _, want := range []string{"axiom branch status --cwd <ruta>", "git -C " + pathquote.Quote(dir) + " init"} {
		if !strings.Contains(refusal.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", refusal.Continuation, want)
		}
	}
}

func TestResultUnusableDirectoryKeepsItsReasonAndPointsToCwd(t *testing.T) {
	refusal := UnusableDirectory("status", "el directorio X no existe")

	if refusal.Code != CodeNotAGitRepository || refusal.Message != "el directorio X no existe" {
		t.Errorf("code %s, mensaje %q", refusal.Code, refusal.Message)
	}
	if !strings.Contains(refusal.Continuation, "axiom branch status --cwd <ruta>") {
		t.Errorf("la continuación %q debe nombrar `axiom branch status --cwd <ruta>`", refusal.Continuation)
	}
}

// Regla 11: toda negativa nombra una continuación ejecutable (una orden `axiom ...`).
func TestResultEveryRefusalNamesAnExecutableContinuation(t *testing.T) {
	executable := regexp.MustCompile("(?:gentle-ai|axiom) [a-z][a-z-]*")
	refusals := map[string]*Refusal{
		"NotAGitRepository": NotAGitRepository("status", `C:\x`),
		"UnusableDirectory": UnusableDirectory("status", "razón"),
		"GitFailed":         GitFailed("status", `C:\x`, "operación", errors.New("fallo")),
		"ConfigRefusal":     ConfigRefusal(configLocation(errors.New("git inválido"), nil, nil)),
		"LocateRefusal":     LocateRefusal(errors.New("no se pudo resolver")),
	}
	for name, refusal := range refusals {
		if refusal == nil {
			t.Errorf("%s no devolvió un rechazo", name)
			continue
		}
		if !executable.MatchString(refusal.Continuation) {
			t.Errorf("%s: la continuación %q no nombra una orden `axiom ...`", name, refusal.Continuation)
		}
		if refusal.Message == "" {
			t.Errorf("%s: el mensaje está vacío", name)
		}
	}
}
