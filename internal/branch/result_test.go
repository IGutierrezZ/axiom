package branch

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
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

// statusPlan es un plan con un repositorio de dos roles, como el que entrega el
// planificador al motor.
func statusPlan() RepoPlan {
	return RepoPlan{
		Name:              "axiom",
		GitRoot:           `C:\repos\axiom`,
		CommonDir:         `C:\repos\axiom\.git`,
		RelPath:           ".",
		Role:              RepoChangeBranch,
		Roles:             []string{"core", "qa"},
		IntegrationBranch: "main",
		Branches: []BranchPlan{
			{Name: "feat/x", Kind: KindChange, PRTarget: "main", Worktree: `C:\repos\axiom-wt\x\axiom`},
			{Name: "feat/x-core", Kind: KindRole, RoleID: "core", From: "feat/x", PRTarget: "feat/x"},
		},
	}
}

// REQ-25.15 y REQ-25.2 (G-4): el resultado de `status` lleva el papel de cada
// repositorio y el tipo de cada rama como token con su etiqueta en un campo hermano.
func TestNewRepositoryResultPairsRoleAndKindWithTheirLabels(t *testing.T) {
	repository := NewRepositoryResult(statusPlan())

	if repository.Name != "axiom" || repository.GitRoot != `C:\repos\axiom` || repository.CommonDir != `C:\repos\axiom\.git` {
		t.Errorf("identidad del repositorio = %+v", repository)
	}
	if repository.Role != RepoChangeBranch || repository.RoleLabel != "repositorio con rama de cambio" {
		t.Errorf("role %q / roleLabel %q; se esperaba change_branch con su etiqueta", repository.Role, repository.RoleLabel)
	}
	if repository.IntegrationBranch != "main" || !reflect.DeepEqual(repository.Roles, []string{"core", "qa"}) {
		t.Errorf("integración %q, roles %v", repository.IntegrationBranch, repository.Roles)
	}
	if len(repository.Branches) != 2 {
		t.Fatalf("ramas = %+v, se esperaban dos", repository.Branches)
	}
	change, role := repository.Branches[0], repository.Branches[1]
	if change.Name != "feat/x" || change.Kind != KindChange || change.KindLabel != "rama del cambio" || change.PRTarget != "main" {
		t.Errorf("rama del cambio = %+v", change)
	}
	if role.Name != "feat/x-core" || role.Kind != KindRole || role.KindLabel != "rama de rol" || role.Role != "core" || role.PRTarget != "feat/x" {
		t.Errorf("rama de rol = %+v", role)
	}
	if change.Exists || change.Worktree != "" || change.Dirty || change.Ahead != nil || change.Behind != nil {
		t.Errorf("el estado de git lo rellena el motor: una rama recién planificada no existe ni tiene worktree: %+v", change)
	}
}

// REQ-25.15: el estado de una rama distingue «cero commits» de «no se calculó»:
// adelanto y atraso salen como `0` cuando se calcularon y no salen cuando no.
func TestBranchResultDistinguishesZeroFromNotComputed(t *testing.T) {
	zero, two := 0, 2
	result := NewResult("status")
	result.Repositories = []RepositoryResult{NewRepositoryResult(statusPlan())}
	first := &result.Repositories[0].Branches[0]
	first.Exists, first.Ahead, first.Behind, first.ComparedTo = true, &zero, &two, "origin/main"

	_, decoded := resultJSON(t, result)
	branches := decoded["repositories"].([]any)[0].(map[string]any)["branches"].([]any)
	computed, notComputed := branches[0].(map[string]any), branches[1].(map[string]any)

	if computed["ahead"] != float64(0) || computed["behind"] != float64(2) || computed["comparedTo"] != "origin/main" || computed["exists"] != true {
		t.Errorf("rama con adelanto y atraso calculados = %v", computed)
	}
	for _, absent := range []string{"ahead", "behind", "comparedTo", "worktree"} {
		if _, present := notComputed[absent]; present {
			t.Errorf("una rama sin cálculo no lleva %q: %v", absent, notComputed)
		}
	}
	if notComputed["exists"] != false || notComputed["dirty"] != false {
		t.Errorf("exists y dirty se imprimen siempre: %v", notComputed)
	}
}

// REQ-25.15 (escenario «Cambio no iniciado»): `not_started` es un resultado alcanzado
// (código de salida 0), con su etiqueta, y conserva el resto del resultado.
func TestNotStartedKeepsTheResultAndExitsWithZero(t *testing.T) {
	result := NewResult("status")
	result.Change = "x"
	started := result.NotStarted()

	if started.Status != StatusNotStarted || started.StatusLabel != "no iniciado" || started.ExitCode() != 0 {
		t.Errorf("status %q / %q, salida %d; se esperaba not_started / no iniciado / 0", started.Status, started.StatusLabel, started.ExitCode())
	}
	if started.Change != "x" || started.Operation != "status" || started.Code != "" {
		t.Errorf("not_started conserva el resultado y no lleva código: %+v", started)
	}
	if result.Status != StatusReady {
		t.Errorf("NotStarted no altera el resultado original: %q", result.Status)
	}
}

// REQ-25.2: el resultado de `status` lleva el cambio, el carril, la topología y las
// rutas del plan; las claves salen en camelCase y sin marcas de tiempo.
func TestStatusResultSerialisesTheChangeAndItsPlan(t *testing.T) {
	result := NewResult("status")
	result.Change, result.Lane, result.Topology = "x", LaneSDD, workspace.TopologyMonorepoEmbedded
	result.WorkspaceRoot, result.PlanningHome = `C:\wt\x\axiom`, `C:\wt\x\axiom\openspec`
	result.OddDocument = `C:\wt\x\axiom\odd\tasks\x.md`
	result.Repositories = []RepositoryResult{NewRepositoryResult(statusPlan())}
	result.Repositories[0].Base = &BaseResult{Ref: "origin/main", SHA: "abc123", Source: "remote-tracking"}
	result.Skipped = []SkippedPath{{Path: `C:\p\scripts`, Role: "tooling", Code: CodeNotAGitRepository, Message: "no es un repositorio"}}

	_, decoded := resultJSON(t, result)
	for key, want := range map[string]any{
		"change": "x", "lane": "sdd", "topology": "monorepo-embedded",
		"workspaceRoot": `C:\wt\x\axiom`, "planningHome": `C:\wt\x\axiom\openspec`,
		"oddDocument": `C:\wt\x\axiom\odd\tasks\x.md`,
	} {
		if decoded[key] != want {
			t.Errorf("%s = %v, se esperaba %v", key, decoded[key], want)
		}
	}
	base := decoded["repositories"].([]any)[0].(map[string]any)["base"].(map[string]any)
	if base["ref"] != "origin/main" || base["sha"] != "abc123" || base["source"] != "remote-tracking" || base["fetched"] != false {
		t.Errorf("base = %v; se esperaba ref, sha, source y fetched=false", base)
	}
	skipped := decoded["skipped"].([]any)[0].(map[string]any)
	if skipped["path"] != `C:\p\scripts` || skipped["role"] != "tooling" || skipped["code"] != "not_a_git_repository" || skipped["message"] == "" {
		t.Errorf("skipped = %v", skipped)
	}
	// Los resultados que no son de un cambio no llevan estas claves.
	_, bare := resultJSON(t, NewResult("status"))
	for _, absent := range []string{"change", "lane", "topology", "workspaceRoot", "planningHome", "oddDocument", "skipped", "mode", "modeLabel", "modeSource", "modeSourceLabel", "registry"} {
		if _, present := bare[absent]; present {
			t.Errorf("un resultado vacío no lleva la clave %q", absent)
		}
	}
}

// El texto de `status` nombra el cambio, el estado con su etiqueta y, por repositorio,
// sus ramas con su worktree, su limpieza, su adelanto y atraso y el destino del PR.
func TestWriteTextPrintsTheStatusOfAChange(t *testing.T) {
	zero, three := 0, 3
	result := NewResult("status")
	result.Change, result.Lane = "x", LaneSDD
	result.Repositories = []RepositoryResult{NewRepositoryResult(statusPlan())}
	repository := &result.Repositories[0]
	repository.CurrentBranch = "main"
	repository.Base = &BaseResult{Ref: "origin/main", SHA: "0123456789abcdef", Source: "remote-tracking"}
	repository.Branches[0].Exists, repository.Branches[0].Dirty = true, true
	repository.Branches[0].Worktree = `C:\repos\axiom-wt\x\axiom`
	repository.Branches[0].ComparedTo = "origin/main"
	repository.Branches[0].Ahead, repository.Branches[0].Behind = &zero, &three
	result.Warnings = []string{"algo que revisar"}

	var stdout, stderr bytes.Buffer
	result.WriteText(&stdout, &stderr)

	if stderr.Len() != 0 {
		t.Errorf("un resultado listo no escribe en stderr: %q", stderr.String())
	}
	for _, want := range []string{
		"Cambio: x (carril sdd)",
		"Estado: listo",
		"axiom (repositorio con rama de cambio)",
		"Rama actual: main\n",
		"Base: origin/main (0123456789ab)",
		"feat/x (rama del cambio)",
		`worktree C:\repos\axiom-wt\x\axiom`,
		"con cambios sin confirmar",
		"adelanto 0, atraso 3 respecto de origin/main",
		"PR hacia main",
		"feat/x-core (rama de rol)",
		"sin crear",
		"PR hacia feat/x",
		"Aviso: algo que revisar\n",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("la salida de texto debe contener %q:\n%s", want, stdout.String())
		}
	}

	stdout.Reset()
	result.NotStarted().WriteText(&stdout, &stderr)
	if !strings.Contains(stdout.String(), "Estado: no iniciado") {
		t.Errorf("un cambio no iniciado se nombra con su etiqueta:\n%s", stdout.String())
	}
}

// REQ-25.15 y REQ-25.14: el resultado de `status` informa del modo de trabajo
// materializado y de su origen, cada uno con su etiqueta, y de si el cambio tiene
// registro local (`registry.present`, siempre impreso: `false` también es información).
func TestStatusResultCarriesTheModeAndTheRegistry(t *testing.T) {
	t.Run("con registro", func(t *testing.T) {
		result := NewResult("status")
		result.SetMode(ModeWorktree, ModeSourceHub)
		result.Registry = &RegistryResult{Present: true}

		_, decoded := resultJSON(t, result)
		for key, want := range map[string]any{
			"mode": "worktree", "modeLabel": "un worktree por rama",
			"modeSource": "hub", "modeSourceLabel": "preferencias personales del hub",
		} {
			if decoded[key] != want {
				t.Errorf("%s = %v, se esperaba %v", key, decoded[key], want)
			}
		}
		if registry, _ := decoded["registry"].(map[string]any); registry["present"] != true {
			t.Errorf("registry = %v, se esperaba present=true", decoded["registry"])
		}
	})
	t.Run("sin registro no hay modo y present es falso", func(t *testing.T) {
		result := NewResult("status")
		result.Registry = &RegistryResult{}

		_, decoded := resultJSON(t, result)
		registry, _ := decoded["registry"].(map[string]any)
		if present, ok := registry["present"]; !ok || present != false {
			t.Errorf("registry = %v, se esperaba present=false impreso", decoded["registry"])
		}
		for _, absent := range []string{"mode", "modeLabel", "modeSource", "modeSourceLabel"} {
			if _, present := decoded[absent]; present {
				t.Errorf("sin registro no se conoce el modo: no debe llevar %q", absent)
			}
		}
	})
	t.Run("el modo in_place y el origen por defecto", func(t *testing.T) {
		result := NewResult("status")
		result.SetMode(ModeInPlace, ModeSourceDefault)
		if result.ModeLabel != "en la carpeta del proyecto" || result.ModeSourceLabel != "valor por defecto" {
			t.Errorf("etiquetas = %q / %q", result.ModeLabel, result.ModeSourceLabel)
		}
	})
}

// El worktree que el registro guarda para una rama se informa aparte del que git
// conoce (`worktree`): así se ve cuando divergen. Sale solo cuando el registro lo tiene.
func TestBranchResultReportsTheRegisteredWorktreeApartFromTheRealOne(t *testing.T) {
	result := NewResult("status")
	result.Repositories = []RepositoryResult{NewRepositoryResult(statusPlan())}
	first := &result.Repositories[0].Branches[0]
	first.Exists, first.Worktree = true, `C:\repos\axiom-wt\x\axiom`
	first.RegisteredWorktree = `C:\registrado\x\axiom`

	_, decoded := resultJSON(t, result)
	branches := decoded["repositories"].([]any)[0].(map[string]any)["branches"].([]any)
	registered, plain := branches[0].(map[string]any), branches[1].(map[string]any)
	if registered["registeredWorktree"] != `C:\registrado\x\axiom` || registered["worktree"] != `C:\repos\axiom-wt\x\axiom` {
		t.Errorf("rama con worktree registrado = %v", registered)
	}
	if _, present := plain["registeredWorktree"]; present {
		t.Errorf("una rama sin worktree registrado no lleva la clave: %v", plain)
	}

	var stdout, stderr bytes.Buffer
	result.WriteText(&stdout, &stderr)
	if !strings.Contains(stdout.String(), `worktree registrado C:\registrado\x\axiom`) {
		t.Errorf("el texto debe nombrar el worktree registrado cuando difiere del real:\n%s", stdout.String())
	}
	stdout.Reset()
	first.RegisteredWorktree = first.Worktree
	result.WriteText(&stdout, &stderr)
	if strings.Contains(stdout.String(), "worktree registrado") {
		t.Errorf("si el registrado coincide con el real no se repite:\n%s", stdout.String())
	}
}

// El texto de `status` nombra el modo con su origen y si hay registro local.
func TestWriteTextPrintsTheModeAndTheRegistry(t *testing.T) {
	result := NewResult("status")
	result.Change, result.Lane = "x", LaneSDD
	result.SetMode(ModeWorktree, ModeSourceAnswer)
	result.Registry = &RegistryResult{Present: true}

	var stdout, stderr bytes.Buffer
	result.WriteText(&stdout, &stderr)
	for _, want := range []string{"Modo de trabajo: un worktree por rama (respuesta registrada)\n", "Registro local: presente\n"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("la salida de texto debe contener %q:\n%s", want, stdout.String())
		}
	}

	stdout.Reset()
	absent := NewResult("status")
	absent.Change, absent.Lane = "x", LaneSDD
	absent.Registry = &RegistryResult{}
	absent.WriteText(&stdout, &stderr)
	if !strings.Contains(stdout.String(), "Registro local: ausente\n") || strings.Contains(stdout.String(), "Modo de trabajo") {
		t.Errorf("sin registro se informa de su ausencia y no del modo:\n%s", stdout.String())
	}
}

// Regla 11: toda negativa nombra una continuación ejecutable (una orden `axiom ...`).
func TestResultEveryRefusalNamesAnExecutableContinuation(t *testing.T) {
	executable := regexp.MustCompile("(?:gentle-ai|axiom) [a-z][a-z-]*")
	refusals := map[string]*Refusal{
		"NotAGitRepository":  NotAGitRepository("status", `C:\x`),
		"UnusableDirectory":  UnusableDirectory("status", "razón"),
		"GitFailed":          GitFailed("status", `C:\x`, "operación", errors.New("fallo")),
		"ConfigRefusal":      ConfigRefusal(configLocation(errors.New("git inválido"), nil, nil)),
		"LocateRefusal":      LocateRefusal(errors.New("no se pudo resolver")),
		"ValidateChangeName": ValidateChangeName("status", `C:\x`, "a b:c"),
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
