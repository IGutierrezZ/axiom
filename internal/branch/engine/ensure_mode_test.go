package engine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Modo de trabajo efectivo de `ensure` (REQ-25.8, REQ-25.29, design D-05 y D-08): la prioridad
// registro > `--mode` > hub > defecto, `ask` como `decision_required` sin efectos, la raíz de
// worktrees del hub y la identidad del proyecto. Con git real, entorno aislado y sin red; las
// preferencias del hub se inyectan con un doble salvo en las pruebas que ejercitan el hub
// real, que aíslan HOME. Las pruebas comparten repositorio y crean pocos worktrees: cada
// `git worktree add` cuesta segundos en Windows.

// fakePrefs es el doble de las preferencias del hub: devuelve lo que se le da y anota la
// carpeta maestra por la que se le preguntó.
type fakePrefs struct {
	mode    branch.Mode
	root    string
	source  string
	warning string
	asked   *[]string
}

func (f fakePrefs) WorktreePrefs(master string) (branch.Mode, string, string, string) {
	if f.asked != nil {
		*f.asked = append(*f.asked, master)
	}
	root := f.root
	if root == "" {
		root = branch.DefaultWorktreeRoot(master)
	}
	return f.mode, root, f.source, f.warning
}

var (
	hubWorktree = fakePrefs{mode: branch.ModeWorktree, source: branch.ModeSourceHub}
	hubAsk      = fakePrefs{mode: branch.ModeAsk, source: branch.ModeSourceHub}
	hubInPlace  = fakePrefs{mode: branch.ModeInPlace, source: branch.ModeSourceHub}
)

func ensureWith(t *testing.T, prefs branch.Prefs, request EnsureRequest) branch.Result {
	t.Helper()
	engine := Default()
	engine.prefs = prefs
	if request.Lane == "" {
		request.Lane = branch.LaneSDD
	}
	return engine.Ensure(context.Background(), request)
}

func wantMode(t *testing.T, result branch.Result, mode branch.Mode, source string) {
	t.Helper()
	if result.Mode != mode || result.ModeSource != source || result.ModeLabel == "" || result.ModeSourceLabel == "" {
		t.Errorf("modo %q (%q) con origen %q (%q); se esperaba %s con origen %s y las dos etiquetas", result.Mode, result.ModeLabel, result.ModeSource, result.ModeSourceLabel, mode, source)
	}
}

// REQ-25.8 y D-08: prioridad registro > `--mode` > hub > defecto, y el origen que se registra
// por cambio (`hub`, `flag`, `answer`). El origen `answer` es la respuesta a `ask`; `flag`, un
// `--mode` con un hub que no pregunta.
func TestEnsureModePriorityIsRegistryThenFlagThenHubThenDefault(t *testing.T) {
	repo := embeddedCoreProject(t)

	t.Run("el hub fija el modo y se registra con origen hub", func(t *testing.T) {
		result := ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "a"})
		wantReady(t, result)
		wantMode(t, result, branch.ModeWorktree, branch.ModeSourceHub)
		if record := loadRecord(t, repo, "a"); record.Mode != branch.ModeWorktree || record.ModeSource != branch.ModeSourceHub {
			t.Errorf("registro = %q con origen %q, se esperaba worktree con origen hub", record.Mode, record.ModeSource)
		}
	})

	t.Run("--mode con un hub que no pregunta tiene origen flag", func(t *testing.T) {
		result := ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "b", Mode: branch.ModeWorktree})
		wantReady(t, result)
		wantMode(t, result, branch.ModeWorktree, branch.ModeSourceFlag)
		if record := loadRecord(t, repo, "b"); record.ModeSource != branch.ModeSourceFlag {
			t.Errorf("origen registrado = %q, se esperaba flag", record.ModeSource)
		}
	})

	t.Run("--mode con un hub en ask es la respuesta y tiene origen answer", func(t *testing.T) {
		result := ensureWith(t, hubAsk, EnsureRequest{CWD: repo, Change: "c", Mode: branch.ModeWorktree})
		wantReady(t, result)
		wantMode(t, result, branch.ModeWorktree, branch.ModeSourceAnswer)
		if record := loadRecord(t, repo, "c"); record.ModeSource != branch.ModeSourceAnswer {
			t.Errorf("origen registrado = %q, se esperaba answer", record.ModeSource)
		}
	})

	t.Run("el registro manda sobre el hub: ni pregunta ni cambia de modo", func(t *testing.T) {
		for name, prefs := range map[string]fakePrefs{"ask": hubAsk, "in_place": hubInPlace} {
			result := ensureWith(t, prefs, EnsureRequest{CWD: repo, Change: "a"})
			wantReady(t, result)
			wantMode(t, result, branch.ModeWorktree, branch.ModeSourceHub)
			if record := loadRecord(t, repo, "a"); record.Mode != branch.ModeWorktree {
				t.Errorf("con el hub en %s el registro pasó a %q", name, record.Mode)
			}
		}
	})

	t.Run("el registro manda sobre --mode cuando coincide: conserva su origen", func(t *testing.T) {
		result := ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "c", Mode: branch.ModeWorktree})
		wantReady(t, result)
		wantMode(t, result, branch.ModeWorktree, branch.ModeSourceAnswer)
	})

	t.Run("un modo distinto del materializado es plan_conflict y no toca nada", func(t *testing.T) {
		state := repositoryState(t, repo)
		before, err := os.ReadFile(recordFile(commonDir(repo), "sdd", "a"))
		if err != nil {
			t.Fatal(err)
		}

		result := ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "a", Mode: branch.ModeInPlace})

		wantRefusal(t, result, branch.CodePlanConflict, branch.StatusRefused)
		for _, want := range []string{"a", "worktree", "in_place"} {
			if !strings.Contains(result.Message, want) {
				t.Errorf("el mensaje %q debe nombrar %q", result.Message, want)
			}
		}
		for _, want := range []string{"axiom branch ensure --change a --mode worktree --cwd", pathquote.Quote(repo)} {
			if !strings.Contains(result.Continuation, want) {
				t.Errorf("la continuación %q debe contener %q (repetir con el modo materializado)", result.Continuation, want)
			}
		}
		if got := repositoryState(t, repo); got != state {
			t.Errorf("el rechazo cambió el repositorio:\n%s\n---\n%s", state, got)
		}
		if after, _ := os.ReadFile(recordFile(commonDir(repo), "sdd", "a")); string(after) != string(before) {
			t.Error("el rechazo no debe reescribir el registro")
		}
	})
}

// REQ-25.8, escenario «Modo `ask` devuelve `decision_required` sin crear nada»: estado
// tipado, código de salida distinto de cero, ni rama, ni worktree, ni registro, y ninguna
// operación de escritura ni de red (tampoco con `--fetch`: la decisión va antes que el fetch).
func TestEnsureAskReturnsDecisionRequiredWithoutCreatingAnything(t *testing.T) {
	repo := embeddedCoreProject(t)
	state := repositoryState(t, repo)
	seen := observe(t)

	result := ensureWith(t, hubAsk, EnsureRequest{CWD: repo, Change: "x", Fetch: true})

	wantRefusal(t, result, branch.CodeDecisionRequired, branch.StatusDecisionRequired)
	if result.StatusLabel != "decisión requerida" || result.ExitCode() == branch.ExitReady {
		t.Errorf("estado %q (%q), salida %d; se esperaba decisión requerida y una salida distinta de cero", result.Status, result.StatusLabel, result.ExitCode())
	}
	decision := result.Decision
	if decision == nil || decision.Question == "" || len(decision.Choices) != 2 {
		t.Fatalf("decision = %+v, se esperaba la pregunta y dos opciones", decision)
	}
	for i, want := range []branch.Mode{branch.ModeWorktree, branch.ModeInPlace} {
		choice := decision.Choices[i]
		if choice.Answer != string(want) || choice.Label != branch.Label(string(want)) || choice.Effect == "" {
			t.Errorf("opción %d = %+v, se esperaba la respuesta %s con su etiqueta y su efecto", i, choice, want)
		}
		for _, wantPart := range []string{"axiom branch ensure --change x --mode " + string(want) + " --cwd", pathquote.Quote(repo)} {
			if !strings.Contains(choice.Invocation, wantPart) {
				t.Errorf("la invocación %q de %s debe contener %q", choice.Invocation, want, wantPart)
			}
		}
	}
	if !strings.Contains(decision.Question, "cambio x") || strings.Contains(result.Continuation, "--fetch") {
		t.Errorf("la pregunta %q debe nombrar el cambio y la continuación %q no arrastra --fetch", decision.Question, result.Continuation)
	}

	wantNothingCreated(t, repo, state, "x")
	wantNoRegistryFile(t, repo, "x")
	for _, invocation := range *seen {
		if isWriteInvocation(invocation) || remoteOperation(invocation[1]) {
			t.Errorf("decision_required no puede lanzar %v", invocation)
		}
	}

	t.Run("el JSON lleva la decisión con las dos opciones y sin acciones", func(t *testing.T) {
		var decoded struct {
			Status   string `json:"status"`
			Code     string `json:"code"`
			Decision struct {
				Question string `json:"question"`
				Choices  []struct {
					Answer, Label, Effect, Invocation string
				} `json:"choices"`
			} `json:"decision"`
		}
		if err := json.Unmarshal([]byte(resultJSON(t, result)), &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.Status != "decision_required" || decoded.Code != "decision_required" || decoded.Decision.Question == "" ||
			len(decoded.Decision.Choices) != 2 || decoded.Decision.Choices[0].Answer != "worktree" || decoded.Decision.Choices[1].Answer != "in_place" {
			t.Errorf("JSON inesperado: %+v", decoded)
		}
	})

	t.Run("el carril odd se conserva en las invocaciones", func(t *testing.T) {
		odd := ensureWith(t, hubAsk, EnsureRequest{CWD: repo, Change: "mi-feature", Lane: branch.LaneODD})
		wantRefusal(t, odd, branch.CodeDecisionRequired, branch.StatusDecisionRequired)
		for _, choice := range odd.Decision.Choices {
			if !strings.Contains(choice.Invocation, "--lane odd") {
				t.Errorf("la invocación %q debe repetir --lane odd", choice.Invocation)
			}
		}
	})
}

// REQ-25.8, escenario «La respuesta a `ask` se registra por cambio»: tras `--mode worktree`, el
// mismo cambio reutiliza el modo sin preguntar (y sin lanzar ninguna escritura), otro cambio
// sin respuesta vuelve a preguntar y la preferencia del hub no se toca.
func TestEnsureAskAnswerIsRecordedPerChangeAndReused(t *testing.T) {
	repo := embeddedCoreProject(t)
	var asked []string
	prefs := hubAsk
	prefs.asked = &asked

	wantReady(t, ensureWith(t, prefs, EnsureRequest{CWD: repo, Change: "x", Mode: branch.ModeWorktree}))
	seen := observe(t)

	again := ensureWith(t, prefs, EnsureRequest{CWD: repo, Change: "x"})
	wantReady(t, again)
	wantMode(t, again, branch.ModeWorktree, branch.ModeSourceAnswer)
	for _, invocation := range *seen {
		if isWriteInvocation(invocation) {
			t.Errorf("reutilizar la respuesta no tiene nada que crear y lanzó %v", invocation)
		}
	}

	wantRefusal(t, ensureWith(t, prefs, EnsureRequest{CWD: repo, Change: "y"}), branch.CodeDecisionRequired, branch.StatusDecisionRequired)
	wantNoRegistryFile(t, repo, "y")
	if len(asked) == 0 {
		t.Fatal("el doble del hub no recibió ninguna consulta: la prueba sería vacua")
	}
	// La respuesta no modifica la preferencia: el hub sigue diciendo `ask` (el doble no tiene
	// forma de escribir) y la consulta siempre va por la misma carpeta maestra.
	for _, master := range asked {
		wantSameDirectory(t, "carpeta maestra consultada", master, repo)
	}
}

// design D-08: la raíz de worktrees de las preferencias sustituye a la de por defecto.
func TestEnsureUsesTheWorktreeRootOfThePrefs(t *testing.T) {
	repo := embeddedCoreProject(t)
	root := filepath.Join(t.TempDir(), "mis-worktrees")
	prefs := hubWorktree
	prefs.root = root

	result := ensureWith(t, prefs, EnsureRequest{CWD: repo, Change: "x"})

	wantReady(t, result)
	wantSameDirectory(t, "worktreeRoot", result.WorktreeRoot, root)
	feature := branchByName(t, onlyRepository(t, result), "feat/x")
	wantSameDirectory(t, "worktree de feat/x", feature.Worktree, filepath.Join(root, "x", filepath.Base(repo)))
}

// design D-08 y REQ-25.8: una petición con un modo que no es una respuesta (`ask`, o un
// valor desconocido) es `config_invalid` y no toca nada.
func TestEnsureRejectsAModeThatIsNotAnAnswer(t *testing.T) {
	repo := embeddedCoreProject(t)
	state := repositoryState(t, repo)

	for _, mode := range []branch.Mode{branch.ModeAsk, "siempre"} {
		result := ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x", Mode: mode})

		wantRefusal(t, result, branch.CodeConfigInvalid, branch.StatusRefused)
		for _, want := range []string{string(mode), "worktree", "in_place"} {
			if !strings.Contains(result.Message, want) {
				t.Errorf("el mensaje %q debe nombrar %q", result.Message, want)
			}
		}
		if !strings.Contains(result.Continuation, "axiom branch ensure --change x --mode worktree --cwd") {
			t.Errorf("la continuación %q debe ofrecer una orden con un modo válido", result.Continuation)
		}
	}
	wantNothingCreated(t, repo, state, "x")
}

// useRealHub aísla HOME y USERPROFILE y devuelve el gestor del hub que usa `Default()`.
func useRealHub(t *testing.T) *hub.Manager {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	manager, err := hub.NewManager("")
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

// setHubPrefs registra el proyecto en el hub y le guarda las preferencias.
func setHubPrefs(t *testing.T, manager *hub.Manager, project, mode string) {
	t.Helper()
	if _, err := manager.Register(project, "ejemplo", ""); err != nil {
		t.Fatal(err)
	}
	cfg, err := manager.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Workspaces[0].WorktreeMode = mode
	if err := manager.Save(cfg); err != nil {
		t.Fatal(err)
	}
}

// REQ-25.29 y design D-08, con el hub real: un `workspaces.json` sin leer da el modo y la raíz
// por defecto con un aviso, y repetir da el mismo resultado byte a byte.
func TestEnsureFallsBackToTheDefaultModeWithAWarningWhenTheHubCannotBeRead(t *testing.T) {
	repo := embeddedCoreProject(t)
	manager := useRealHub(t)
	if err := os.MkdirAll(filepath.Dir(manager.GetConfigPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manager.GetConfigPath(), []byte("{esto no es json"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := Default().Ensure(context.Background(), EnsureRequest{CWD: repo, Change: "x", Lane: branch.LaneSDD})

	wantReady(t, result)
	wantMode(t, result, branch.ModeWorktree, branch.ModeSourceDefault)
	wantSameDirectory(t, "worktreeRoot", result.WorktreeRoot, branch.DefaultWorktreeRoot(canonicalDirectory(t, repo)))
	warned := false
	for _, warning := range result.Warnings {
		warned = warned || (strings.Contains(warning, "hub") && strings.Contains(warning, "valores por defecto"))
	}
	if !warned {
		t.Errorf("avisos %q: debe avisar de que el hub no se pudo leer y se usan los valores por defecto", result.Warnings)
	}
	first := resultJSON(t, result)
	if second := resultJSON(t, Default().Ensure(context.Background(), EnsureRequest{CWD: repo, Change: "x", Lane: branch.LaneSDD})); second != first {
		t.Errorf("repetir cambió el resultado:\n%s\n---\n%s", first, second)
	}
}

// REQ-25.29 y design D-08, con el hub real: la identidad del proyecto es la carpeta maestra
// del checkout principal, también cuando `ensure` se ejecuta desde un worktree enlazado, y
// la raíz de worktrees por defecto sale de ella, no del worktree.
func TestEnsureReadsTheHubPrefsOfTheMainCheckoutFromALinkedWorktree(t *testing.T) {
	repo := embeddedCoreProject(t)
	manager := useRealHub(t)
	wantReady(t, Default().Ensure(context.Background(), EnsureRequest{CWD: repo, Change: "x", Lane: branch.LaneSDD, Mode: branch.ModeWorktree}))
	linked := plannedWorktree(t, repo, "x", filepath.Base(repo))
	setHubPrefs(t, manager, repo, string(branch.ModeAsk))

	for name, cwd := range map[string]string{"desde el checkout principal": repo, "desde un worktree enlazado": linked} {
		t.Run(name, func(t *testing.T) {
			result := Default().Ensure(context.Background(), EnsureRequest{CWD: cwd, Change: "y", Lane: branch.LaneSDD})
			wantRefusal(t, result, branch.CodeDecisionRequired, branch.StatusDecisionRequired)
		})
	}

	t.Run("la respuesta desde el worktree enlazado crea el worktree junto a la maestra", func(t *testing.T) {
		result := Default().Ensure(context.Background(), EnsureRequest{CWD: linked, Change: "y", Lane: branch.LaneSDD, Mode: branch.ModeWorktree})
		wantReady(t, result)
		wantMode(t, result, branch.ModeWorktree, branch.ModeSourceAnswer)
		wantSameDirectory(t, "worktreeRoot", result.WorktreeRoot, branch.DefaultWorktreeRoot(canonicalDirectory(t, repo)))
	})
}
