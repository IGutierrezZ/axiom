package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/gitidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Pruebas de la capacidad change-actor-identity (REQ-42.1 a REQ-42.6 y REQ-42.9): `gate record`
// registra la identidad git como `actor` y `kickoff seal` como `sealed_by`. El TestMain fija un
// resolvedor determinista; estas pruebas lo sustituyen localmente.

const (
	actorChange   = "inc-99-example"
	anaIdentity   = "Ana Pérez <ana@example.com>"
	anaName       = "Ana Pérez"
	anaEmail      = "ana@example.com"
	gateActorWord = "actor"
)

// stubActorIdentity sustituye el resolvedor por uno que devuelve identity y anota los
// directorios desde los que se le llamó.
func stubActorIdentity(t *testing.T, identity gitidentity.Identity) *[]string {
	t.Helper()
	var dirs []string
	previous := resolveActorIdentity
	resolveActorIdentity = func(_ context.Context, dir string) gitidentity.Identity {
		dirs = append(dirs, dir)
		return identity
	}
	t.Cleanup(func() { resolveActorIdentity = previous })
	return &dirs
}

// forbidActorIdentity hace fallar la prueba si algo consulta la identidad git.
func forbidActorIdentity(t *testing.T) {
	t.Helper()
	previous := resolveActorIdentity
	resolveActorIdentity = func(_ context.Context, dir string) gitidentity.Identity {
		t.Errorf("no se debía resolver la identidad git (dir %q)", dir)
		return gitidentity.Identity{}
	}
	t.Cleanup(func() { resolveActorIdentity = previous })
}

// useRealActorIdentity restaura la resolución real de producción para las pruebas de extremo a
// extremo, que aíslan git por su cuenta.
func useRealActorIdentity(t *testing.T) {
	t.Helper()
	previous := resolveActorIdentity
	resolveActorIdentity = gitidentity.Resolve
	t.Cleanup(func() { resolveActorIdentity = previous })
}

// captureActorIdentityWarnings recoge el aviso de identidad incompleta.
func captureActorIdentityWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	var warnings bytes.Buffer
	previous := actorIdentityWarnings
	actorIdentityWarnings = &warnings
	t.Cleanup(func() { actorIdentityWarnings = previous })
	return &warnings
}

// actorWorkspace es un workspace temporal con la carpeta del cambio y los avisos de rama
// recogidos para que no ensucien la salida.
func actorWorkspace(t *testing.T) (root, changeRoot string) {
	t.Helper()
	captureSealWarnings(t)
	captureWriterWarnings(t)
	return newGovernanceWorkspace(t, actorChange)
}

func recordActorGate(t *testing.T, root string, extra ...string) (string, error) {
	t.Helper()
	args := append([]string{"record", "--cwd", root, "--change", actorChange, "--gate", "spec", "--decision", "approved"}, extra...)
	var out bytes.Buffer
	err := RunSDDGate(args, &out)
	return out.String(), err
}

func sealActorChange(t *testing.T, root string, extra ...string) error {
	t.Helper()
	args := append([]string{"seal", "--cwd", root, "--change", actorChange}, extra...)
	return RunSDDKickoff(args, &bytes.Buffer{})
}

var sealContinuousArgs = []string{"--execution-style", "continuous", "--handoff-policy", "none"}

func ledgerActor(t *testing.T, changeRoot string) string {
	t.Helper()
	ledger, err := kickoff.LoadGates(changeRoot)
	if err != nil {
		t.Fatalf("LoadGates() error = %v", err)
	}
	if len(ledger.Records) != 1 {
		t.Fatalf("Records = %+v, se esperaba exactamente 1 registro", ledger.Records)
	}
	return ledger.Records[0].Actor
}

func sealedBy(t *testing.T, changeRoot string) string {
	t.Helper()
	sealed, err := kickoff.Load(changeRoot)
	if err != nil || sealed == nil {
		t.Fatalf("Load() = %v, error %v", sealed, err)
	}
	return sealed.SealedBy
}

// partialIdentities son las tres identidades incompletas de REQ-42.5.
var partialIdentities = []struct {
	name     string
	identity gitidentity.Identity
	missing  string
}{
	{"ninguna", gitidentity.New("", ""), "faltan user.name y user.email"},
	{"solo nombre", gitidentity.New(anaName, ""), "falta user.email"},
	{"solo email", gitidentity.New("", anaEmail), "falta user.name"},
}

// wantSingleWarning comprueba que hay exactamente una línea de aviso con los fragmentos dados.
func wantSingleWarning(t *testing.T, warnings *bytes.Buffer, fragments ...string) {
	t.Helper()
	text := warnings.String()
	if strings.Count(text, "\n") != 1 || !strings.HasSuffix(text, "\n") || !strings.HasPrefix(text, "Aviso: ") {
		t.Fatalf("se esperaba exactamente una línea que empiece por %q, hay %q", "Aviso: ", text)
	}
	for _, fragment := range fragments {
		if !strings.Contains(text, fragment) {
			t.Errorf("el aviso %q debe contener %q", text, fragment)
		}
	}
}

func TestGateRecordUsesGitIdentityAsActor(t *testing.T) {
	root, changeRoot := actorWorkspace(t)
	stubActorIdentity(t, gitidentity.New(anaName, anaEmail))
	warnings := captureActorIdentityWarnings(t)

	if _, err := recordActorGate(t, root); err != nil {
		t.Fatalf("RunSDDGate() error = %v", err)
	}
	if got := ledgerActor(t, changeRoot); got != anaIdentity {
		t.Errorf("actor = %q, se esperaba %q", got, anaIdentity)
	}
	if warnings.Len() != 0 {
		t.Errorf("con identidad completa no hay aviso: %q", warnings.String())
	}
}

func TestGateRecordExplicitActorSkipsResolution(t *testing.T) {
	for _, name := range []string{"con identidad", "sin identidad"} {
		t.Run(name, func(t *testing.T) {
			root, changeRoot := actorWorkspace(t)
			forbidActorIdentity(t)
			warnings := captureActorIdentityWarnings(t)

			if _, err := recordActorGate(t, root, "--actor", "maintainer"); err != nil {
				t.Fatalf("RunSDDGate() error = %v", err)
			}
			if got := ledgerActor(t, changeRoot); got != "maintainer" {
				t.Errorf("actor = %q, --actor explícito debe prevalecer", got)
			}
			if warnings.Len() != 0 {
				t.Errorf("con --actor no hay aviso: %q", warnings.String())
			}
		})
	}
}

func TestGateRecordWithoutIdentityFallsBackToCli(t *testing.T) {
	for _, tc := range partialIdentities {
		t.Run(tc.name, func(t *testing.T) {
			root, changeRoot := actorWorkspace(t)
			stubActorIdentity(t, tc.identity)
			warnings := captureActorIdentityWarnings(t)

			if _, err := recordActorGate(t, root); err != nil {
				t.Fatalf("la identidad incompleta no debe bloquear: %v", err)
			}
			if got := ledgerActor(t, changeRoot); got != kickoff.FallbackActor {
				t.Errorf("actor = %q, se esperaba %q", got, kickoff.FallbackActor)
			}
			wantSingleWarning(t, warnings, tc.missing, "--actor", `actor "cli"`)
		})
	}

	t.Run("texto exacto sin identidad", func(t *testing.T) {
		root, _ := actorWorkspace(t)
		stubActorIdentity(t, gitidentity.Identity{})
		warnings := captureActorIdentityWarnings(t)
		if _, err := recordActorGate(t, root); err != nil {
			t.Fatal(err)
		}
		// La raíz que ve el adaptador es la resuelta por el workspace: el texto se compara tras ella.
		const tail = `): se registra actor "cli". Configúrala con git config --global user.name "<nombre>" y git config --global user.email "<email>", o indica --actor.` + "\n"
		const head = "Aviso: la identidad git no está completa en "
		text := warnings.String()
		if !strings.HasPrefix(text, head) || !strings.HasSuffix(text, ` (faltan user.name y user.email`+tail) {
			t.Errorf("texto del aviso inesperado: %q", text)
		}
	})
}

func TestGateRecordRejectedEmitsNoIdentityWarning(t *testing.T) {
	cases := map[string][]string{
		"evidencia no confirmada": {"--gate", "integration", "--evidence-kind", "pr_merged", "--commit", "0123456789abcdef0123456789abcdef01234567"},
		"hand-off sin artefactos": {"--gate", "spec", "--hand-off"},
	}
	for name, extra := range cases {
		t.Run(name, func(t *testing.T) {
			root, changeRoot := actorWorkspace(t)
			dirs := stubActorIdentity(t, gitidentity.Identity{})
			warnings := captureActorIdentityWarnings(t)

			args := append([]string{"record", "--cwd", root, "--change", actorChange, "--decision", "approved", "--reason", "ok"}, extra...)
			if err := RunSDDGate(args, &bytes.Buffer{}); err == nil {
				t.Fatal("la compuerta debía rechazarse")
			}
			if name == "evidencia no confirmada" && len(*dirs) != 1 {
				t.Errorf("la evidencia se verifica tras resolver la identidad: se esperaba 1 resolución y hubo %d", len(*dirs))
			}
			ledger, err := kickoff.LoadGates(changeRoot)
			if err != nil || len(ledger.Records) != 0 {
				t.Fatalf("un rechazo no registra nada: %+v, error %v", ledger.Records, err)
			}
			if warnings.Len() != 0 {
				t.Errorf("un rechazo no emite aviso de identidad: %q", warnings.String())
			}
		})
	}
}

func TestGateRecordJSONShapeUnchanged(t *testing.T) {
	keysOf := func(t *testing.T, identity gitidentity.Identity) ([]string, string, string) {
		t.Helper()
		root, _ := actorWorkspace(t)
		stubActorIdentity(t, identity)
		warnings := captureActorIdentityWarnings(t)
		out, err := recordActorGate(t, root, "--json")
		if err != nil {
			t.Fatalf("RunSDDGate() error = %v", err)
		}
		var document map[string]any
		if err := json.Unmarshal([]byte(out), &document); err != nil {
			t.Fatalf("stdout no es JSON válido: %v\n%s", err, out)
		}
		keys := make([]string, 0, len(document))
		for key := range document {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return keys, out, warnings.String()
	}

	withKeys, withOut, withWarn := keysOf(t, gitidentity.New(anaName, anaEmail))
	withoutKeys, withoutOut, withoutWarn := keysOf(t, gitidentity.Identity{})
	if !reflect.DeepEqual(withKeys, withoutKeys) {
		t.Errorf("las claves del JSON cambian con la identidad: %v frente a %v", withKeys, withoutKeys)
	}
	for _, key := range withKeys {
		if key == gateActorWord || key == "warnings" {
			t.Errorf("el JSON no debe llevar la clave %q", key)
		}
	}
	if withWarn != "" {
		t.Errorf("con identidad no hay aviso: %q", withWarn)
	}
	if !strings.HasPrefix(withoutWarn, "Aviso: ") {
		t.Errorf("sin identidad el aviso va por su writer: %q", withoutWarn)
	}
	if strings.Contains(withOut, "Aviso") || strings.Contains(withoutOut, "Aviso") {
		t.Errorf("el aviso no debe ir en stdout:\n%s\n%s", withOut, withoutOut)
	}
}

func TestGateRecordHelpMentionsGitIdentity(t *testing.T) {
	var out bytes.Buffer
	if err := RunSDDGate([]string{"--help"}, &out); err != nil {
		t.Fatalf("RunSDDGate(--help) error = %v", err)
	}
	if !strings.Contains(out.String(), "--actor") || !strings.Contains(out.String(), "identidad git") {
		t.Errorf("la ayuda de record debe mencionar la identidad git como valor por defecto de --actor:\n%s", out.String())
	}
}

func TestKickoffSealUsesGitIdentityAsSealedBy(t *testing.T) {
	root, changeRoot := actorWorkspace(t)
	stubActorIdentity(t, gitidentity.New(anaName, anaEmail))
	warnings := captureActorIdentityWarnings(t)

	if err := sealActorChange(t, root, sealContinuousArgs...); err != nil {
		t.Fatalf("seal error = %v", err)
	}
	if got := sealedBy(t, changeRoot); got != anaIdentity {
		t.Errorf("sealed_by = %q, se esperaba %q", got, anaIdentity)
	}
	if warnings.Len() != 0 {
		t.Errorf("con identidad completa no hay aviso: %q", warnings.String())
	}
}

func TestKickoffSealWithoutIdentityFallsBackToCli(t *testing.T) {
	for _, tc := range partialIdentities {
		t.Run(tc.name, func(t *testing.T) {
			root, changeRoot := actorWorkspace(t)
			stubActorIdentity(t, tc.identity)
			warnings := captureActorIdentityWarnings(t)

			if err := sealActorChange(t, root, sealContinuousArgs...); err != nil {
				t.Fatalf("la identidad incompleta no debe bloquear el sellado: %v", err)
			}
			if got := sealedBy(t, changeRoot); got != kickoff.FallbackActor {
				t.Errorf("sealed_by = %q, se esperaba %q", got, kickoff.FallbackActor)
			}
			wantSingleWarning(t, warnings, tc.missing, `sealed_by "cli"`)
			if strings.Contains(warnings.String(), "--actor") {
				t.Errorf("seal no admite --actor: el aviso no debe sugerirlo: %q", warnings.String())
			}
		})
	}
}

func TestKickoffSealInferKeepsInferredAndSkipsGit(t *testing.T) {
	for _, name := range []string{"con identidad", "sin identidad"} {
		t.Run(name, func(t *testing.T) {
			root, changeRoot := actorWorkspace(t)
			forbidActorIdentity(t)
			warnings := captureActorIdentityWarnings(t)

			if err := sealActorChange(t, root, "--infer"); err != nil {
				t.Fatalf("seal --infer error = %v", err)
			}
			if got := sealedBy(t, changeRoot); got != kickoff.SealedByInferred {
				t.Errorf("sealed_by = %q, --infer conserva %q", got, kickoff.SealedByInferred)
			}
			if warnings.Len() != 0 {
				t.Errorf("--infer no avisa: %q", warnings.String())
			}
		})
	}
}

func TestKickoffSealExistingKickoffEmitsNoWarning(t *testing.T) {
	root, changeRoot := actorWorkspace(t)
	stubActorIdentity(t, gitidentity.New(anaName, anaEmail))
	if err := sealActorChange(t, root, sealContinuousArgs...); err != nil {
		t.Fatalf("primer seal error = %v", err)
	}

	stubActorIdentity(t, gitidentity.Identity{})
	warnings := captureActorIdentityWarnings(t)
	if err := sealActorChange(t, root, sealContinuousArgs...); err != nil {
		t.Fatalf("segundo seal error = %v", err)
	}
	if warnings.Len() != 0 {
		t.Errorf("un kickoff ya sellado no escribe nada y no avisa: %q", warnings.String())
	}
	if got := sealedBy(t, changeRoot); got != anaIdentity {
		t.Errorf("sealed_by = %q, el kickoff original no debe cambiar", got)
	}
}

func TestKickoffSealRejectsActorFlags(t *testing.T) {
	for _, flag := range []string{"--actor", "--sealed-by"} {
		t.Run(flag, func(t *testing.T) {
			root, changeRoot := actorWorkspace(t)
			stubActorIdentity(t, gitidentity.New(anaName, anaEmail))

			err := sealActorChange(t, root, append([]string{flag, "x"}, sealContinuousArgs...)...)
			if err == nil || !strings.Contains(err.Error(), "desconocid") {
				t.Fatalf("error = %v, se esperaba rechazo como bandera desconocida", err)
			}
			if _, statErr := os.Stat(filepath.Join(changeRoot, kickoff.KickoffFileName)); !os.IsNotExist(statErr) {
				t.Errorf("el rechazo no debe escribir kickoff.yaml: %v", statErr)
			}
		})
	}
}

func TestGateAndSealShareIdentity(t *testing.T) {
	root, changeRoot := actorWorkspace(t)
	stubActorIdentity(t, gitidentity.New("Íñigo Gutiérrez", "inigo@example.com"))
	captureActorIdentityWarnings(t)

	if err := sealActorChange(t, root, sealContinuousArgs...); err != nil {
		t.Fatal(err)
	}
	if _, err := recordActorGate(t, root); err != nil {
		t.Fatal(err)
	}
	if sealer, actor := sealedBy(t, changeRoot), ledgerActor(t, changeRoot); sealer != actor || sealer != "Íñigo Gutiérrez <inigo@example.com>" {
		t.Errorf("sealed_by %q y actor %q deben ser idénticos", sealer, actor)
	}
}

func TestBranchWarningsStayEmptyWithoutIdentity(t *testing.T) {
	sealWarnings := captureSealWarnings(t)
	writerWarnings := captureWriterWarnings(t)
	captureActorIdentityWarnings(t)
	stubActorIdentity(t, gitidentity.Identity{})

	ws := embeddedSealProject(t, "")
	integrateGit(t, ws.Master, "switch", "-q", "-c", "feat/x")
	if _, err := runSeal(t, ws.Master, sealBaseArgs...); err != nil {
		t.Fatalf("seal error = %v", err)
	}
	if _, err := gateRecord(t, ws, "--gate", "spec", "--decision", "approved"); err != nil {
		t.Fatalf("gate record error = %v", err)
	}
	if sealWarnings.Len() != 0 || writerWarnings.Len() != 0 {
		t.Errorf("los avisos de rama no deben recibir el aviso de identidad: seal=%q escritores=%q", sealWarnings.String(), writerWarnings.String())
	}
}

func TestActorIdentityMultirepoUsesSpecsRoot(t *testing.T) {
	captureSealWarnings(t)
	captureWriterWarnings(t)
	captureActorIdentityWarnings(t)
	dirs := stubActorIdentity(t, gitidentity.New(anaName, anaEmail))
	ws := workspacetest.Canonical(t)

	const change = "inc-99-ejemplo"
	sealArgs := []string{"seal", "--cwd", ws.Master, "--change", change, "--role", "backend", "--role", "frontend"}
	if err := RunSDDKickoff(sealArgs, &bytes.Buffer{}); err != nil {
		t.Fatalf("seal multirepo error = %v", err)
	}
	gateArgs := []string{"record", "--cwd", ws.Master, "--change", change, "--gate", "spec", "--decision", "approved"}
	if err := RunSDDGate(gateArgs, &bytes.Buffer{}); err != nil {
		t.Fatalf("gate multirepo error = %v", err)
	}

	if len(*dirs) != 2 {
		t.Fatalf("el resolvedor se llamó con %v, se esperaban dos llamadas (seal y gate)", *dirs)
	}
	for _, dir := range *dirs {
		if filepath.Clean(dir) != filepath.Clean(ws.Specs) {
			t.Errorf("el resolvedor recibió %q, se esperaba la raíz del repositorio de specs %q", dir, ws.Specs)
		}
	}
	changeRoot := filepath.Join(ws.Home, "changes", change)
	if sealer, actor := sealedBy(t, changeRoot), ledgerActor(t, changeRoot); sealer != anaIdentity || actor != anaIdentity {
		t.Errorf("sealed_by %q y actor %q deben ser la identidad del repositorio de specs", sealer, actor)
	}
}

func TestKickoffSealRealGitIdentity(t *testing.T) {
	ws := embeddedSealProject(t, "")
	integrateGit(t, ws.Master, "config", "--local", "user.name", anaName)
	integrateGit(t, ws.Master, "config", "--local", "user.email", anaEmail)
	integrateGit(t, ws.Master, "switch", "-q", "-c", "feat/x")
	captureSealWarnings(t)
	warnings := captureActorIdentityWarnings(t)
	useRealActorIdentity(t)

	if _, err := runSeal(t, ws.Master, sealBaseArgs...); err != nil {
		t.Fatalf("seal error = %v", err)
	}
	if got := sealedBy(t, filepath.Join(ws.Home, "changes", sealChange)); got != anaIdentity {
		t.Errorf("sealed_by = %q, se esperaba la identidad local %q", got, anaIdentity)
	}
	if warnings.Len() != 0 {
		t.Errorf("con identidad local completa no hay aviso: %q", warnings.String())
	}
}

func TestGateRecordRealGitIdentityNonRepository(t *testing.T) {
	isolateIntegrateGit(t)
	global := os.Getenv("GIT_CONFIG_GLOBAL")
	if err := os.WriteFile(global, []byte("[user]\n\tname = Global Pérez\n\temail = global@example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, changeRoot := actorWorkspace(t)
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(root))
	warnings := captureActorIdentityWarnings(t)
	useRealActorIdentity(t)

	if _, err := recordActorGate(t, root); err != nil {
		t.Fatalf("RunSDDGate() error = %v", err)
	}
	if got := ledgerActor(t, changeRoot); got != "Global Pérez <global@example.com>" {
		t.Errorf("actor = %q, se esperaba la identidad global", got)
	}
	if warnings.Len() != 0 {
		t.Errorf("con identidad global completa no hay aviso: %q", warnings.String())
	}
}
