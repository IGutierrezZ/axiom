package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Pruebas de la guarda A en los escritores CLI de la carpeta del cambio (REQ-25.19, REQ-21.16,
// REQ-25.39, design D-11 y G-10, tabla de la §2.11). Cada escritor tiene su negativa (fuera de
// rama, sin escribir nada y con la continuación de `ensure`), su caso en la rama correcta y,
// cuando corresponde, su contexto `archive`. Con git real, entorno aislado y sin red; reutilizan
// los ayudantes de sdd_kickoff_branch_test.go. El escritor de `axiom handoff create` vive en
// `cmd/axiom` y tiene su prueba allí.

const guardArchiveBranch = "feat/x-archive"

// guardSealArgs sella los roles core y qa: con dos roles sobre el repositorio existen las ramas de
// rol `feat/x-core` y `feat/x-qa` (REQ-25.4).
var guardSealArgs = []string{"--execution-style", "checkpointed", "--handoff-policy", "none", "--role", "core", "--role", "qa"}

// captureWriterWarnings recoge los avisos de la guarda de los escritores.
func captureWriterWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	var warnings bytes.Buffer
	previous := writerBranchWarnings
	writerBranchWarnings = &warnings
	t.Cleanup(func() { writerBranchWarnings = previous })
	return &warnings
}

// guardProject es un monorepo embebido con el cambio `x` sellado desde `feat/x` (con instantánea
// y los roles core y qa) y el checkout devuelto a `main`: la forma de un proyecto cuya carpeta de
// cambio solo se puede escribir desde su rama.
func guardProject(t *testing.T) workspacetest.Workspace {
	t.Helper()
	return guardProjectSealedWith(t, guardSealArgs)
}

func guardProjectSealedWith(t *testing.T, sealArgs []string) workspacetest.Workspace {
	t.Helper()
	captureSealWarnings(t)
	captureWriterWarnings(t)
	ws := embeddedSealProject(t, twoRoleConfig)
	integrateGit(t, ws.Master, "switch", "-q", "-c", "feat/x")
	if _, err := runSeal(t, ws.Master, sealArgs...); err != nil {
		t.Fatalf("sellar en feat/x: %v", err)
	}
	integrateGit(t, ws.Master, "switch", "-q", "main")
	return ws
}

// retroSealForGuard escribe para el cambio un kickoff.yaml sin instantánea de política git, la forma
// que dejaba `kickoff seal` antes de la instantánea (un cambio preexistente): queda exento de la
// guarda A (REQ-25.37, R15). Los fixtures de otras pruebas que ejecutan un escritor guardado sobre un
// cambio de un repositorio git que está en `master` o `main` lo usan: un cambio sin ningún
// kickoff.yaml no está exento (supuesto S-7) y el escritor se negaría fuera de la rama del cambio.
// Se sella con `continuous` y un solo rol (plan `tasks.md`), sin pasar por la guarda del seal.
func retroSealForGuard(t *testing.T, cwd, change string) {
	t.Helper()
	parsed, err := kickoff.ParseSealArgs([]string{"--cwd", cwd, "--change", change, "--execution-style", "continuous", "--handoff-policy", "none"})
	if err != nil {
		t.Fatalf("argumentos del sellado de %s: %v", change, err)
	}
	_, changeRoot, err := resolveGovernanceChangeRoot(cwd, change)
	if err != nil {
		t.Fatalf("carpeta del cambio %s: %v", change, err)
	}
	if _, _, err := kickoff.Seal(changeRoot, parsed.ToKickoff(change)); err != nil {
		t.Fatalf("sellar %s sin instantánea: %v", change, err)
	}
}

func guardChangeRoot(ws workspacetest.Workspace) string {
	return filepath.Join(ws.Home, "changes", sealChange)
}

func gateRecord(t *testing.T, ws workspacetest.Workspace, extra ...string) (string, error) {
	t.Helper()
	args := append([]string{"record", "--cwd", ws.Master, "--change", sealChange}, extra...)
	var out bytes.Buffer
	err := RunSDDGate(args, &out)
	return out.String(), err
}

// wantNoWrite comprueba que el rechazo no dejó rastro en la carpeta del cambio.
func wantNoWrite(t *testing.T, ws workspacetest.Workspace, before map[string]string) {
	t.Helper()
	if after := consentCLISnapshot(t, guardChangeRoot(ws)); !reflect.DeepEqual(before, after) {
		t.Errorf("el rechazo no debe escribir en la carpeta del cambio:\nantes: %v\ndespués: %v", before, after)
	}
}

// wantNotRefusedByTheGuard comprueba que err no es el rechazo `wrong_branch` de la guarda: el
// escritor llegó a ejecutarse (puede fallar después por motivos propios).
func wantNotRefusedByTheGuard(t *testing.T, err error) {
	t.Helper()
	var refusal *branch.RefusalError
	if errors.As(err, &refusal) {
		t.Errorf("la guarda no debe rechazar este escritor: %v", err)
	}
}

func wantContinuation(t *testing.T, err error, want ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("se esperaba el rechazo de la guarda y no hubo error")
	}
	for _, fragment := range want {
		if !strings.Contains(err.Error(), fragment) {
			t.Errorf("el rechazo %q debe contener %q", err.Error(), fragment)
		}
	}
}

// REQ-25.19, escenario «Cada escritor se niega fuera de rama»: `gate record` desde `main` se
// niega con `wrong_branch` y la continuación de `ensure`, sin tocar la carpeta del cambio.
func TestGuardGateRecordRefusesOutsideTheChangeBranch(t *testing.T) {
	ws := guardProject(t)
	before := consentCLISnapshot(t, guardChangeRoot(ws))

	out, err := gateRecord(t, ws, "--gate", "spec", "--decision", "approved", "--reason", "ok")

	wantWrongBranch(t, err, "main", "feat/x", ws.Master)
	t.Logf("rechazo de gate record desde main: %v", err)
	wantNoWrite(t, ws, before)
	if out != "" {
		t.Errorf("un rechazo no imprime la confirmación de la compuerta: %q", out)
	}
}

// REQ-25.19, escenario «Cada escritor escribe en la rama correcta» (contexto `ordinary`): la
// rama del cambio y la de un rol sellado escriben la compuerta con normalidad.
func TestGuardGateRecordWritesOnTheChangeAndRoleBranches(t *testing.T) {
	ws := guardProject(t)
	for _, name := range []string{"feat/x", "feat/x-core"} {
		integrateGit(t, ws.Master, "switch", "-q", "-C", name)
		if _, err := gateRecord(t, ws, "--gate", "spec", "--decision", "approved", "--reason", "ok en "+name); err != nil {
			t.Fatalf("gate record en %s: %v", name, err)
		}
	}
	ledger, err := kickoff.LoadGates(guardChangeRoot(ws))
	if err != nil || len(ledger.Records) != 2 {
		t.Fatalf("compuertas = %+v, %v; se esperaban 2 registros", ledger.Records, err)
	}
}

// G-10: `gate record --gate integration` es del contexto `archive`: en `feat/x` se niega
// esperando la rama de archivado y con la continuación `--archive`; en `feat/x-archive` procede.
func TestGuardGateRecordIntegrationUsesTheArchiveContext(t *testing.T) {
	ws := guardProject(t)
	integrateGit(t, ws.Master, "switch", "-q", "feat/x")
	before := consentCLISnapshot(t, guardChangeRoot(ws))
	integration := []string{"--gate", "integration", "--decision", "approved", "--evidence-kind", "attestation", "--evidence", "validado a mano"}

	_, err := gateRecord(t, ws, integration...)

	wantWrongBranch(t, err, "feat/x", guardArchiveBranch, ws.Master)
	wantContinuation(t, err, " --archive")
	wantNoWrite(t, ws, before)

	integrateGit(t, ws.Master, "switch", "-q", "-c", guardArchiveBranch)
	if _, err := gateRecord(t, ws, integration...); err != nil {
		t.Fatalf("gate record --gate integration en %s: %v", guardArchiveBranch, err)
	}
}

// REQ-25.19: una sola guarda cubre `AppendGate` y el handoff de integración que escribe la última
// compuerta de rol: fuera de rama no se escribe ni la compuerta ni `handoff.md`; en la rama del
// cambio se escriben las dos.
func TestGuardGateRecordOfTheLastRoleGuardsTheIntegrationHandoffToo(t *testing.T) {
	ws := guardProject(t)
	handoffPath := filepath.Join(guardChangeRoot(ws), "handoff.md")
	firstRole := []string{"--gate", "role-apply:core", "--decision", "approved", "--reason", "ok"}
	lastRole := []string{"--gate", "role-apply:qa", "--decision", "approved", "--reason", "ok"}

	integrateGit(t, ws.Master, "switch", "-q", "-c", "feat/x-core")
	if _, err := gateRecord(t, ws, firstRole...); err != nil {
		t.Fatalf("gate record del primer rol en feat/x-core: %v", err)
	}
	integrateGit(t, ws.Master, "switch", "-q", "main")
	before := consentCLISnapshot(t, guardChangeRoot(ws))

	_, err := gateRecord(t, ws, lastRole...)

	wantWrongBranch(t, err, "main", "feat/x", ws.Master)
	wantNoWrite(t, ws, before)
	if _, statErr := os.Stat(handoffPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("fuera de rama no se escribe el handoff de integración (%v)", statErr)
	}

	integrateGit(t, ws.Master, "switch", "-q", "feat/x")
	out, err := gateRecord(t, ws, lastRole...)
	if err != nil {
		t.Fatalf("gate record del último rol en feat/x: %v", err)
	}
	if !strings.Contains(out, lastRoleNoticeMarker) {
		t.Errorf("la salida debe avisar del último rol:\n%s", out)
	}
	if _, statErr := os.Stat(filepath.Join(guardChangeRoot(ws), "handoff.md")); statErr != nil {
		t.Errorf("en la rama del cambio se escribe el handoff de integración: %v", statErr)
	}
}

// `gate show` solo lee: no se guarda.
func TestGuardGateShowIsNotGuarded(t *testing.T) {
	ws := guardProject(t)

	var out bytes.Buffer
	err := RunSDDGate([]string{"show", "--cwd", ws.Master, "--change", sealChange}, &out)

	if err != nil {
		t.Fatalf("gate show desde main: %v", err)
	}
}

// REQ-25.18, REQ-25.37: las exenciones. Un sumidero acumulativo por nombre y un cambio retro-sellado
// con `--infer` (kickoff.yaml sin instantánea) escriben desde `main`.
func TestGuardGateRecordExemptChangesWriteFromMain(t *testing.T) {
	t.Run("sumidero acumulativo", func(t *testing.T) {
		captureWriterWarnings(t)
		ws := embeddedSealProject(t, "")
		workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "cumulative-core", "proposal.md"), "# sumidero\n")
		var out bytes.Buffer
		err := RunSDDGate([]string{"record", "--cwd", ws.Master, "--change", "cumulative-core", "--gate", "design", "--decision", "approved", "--reason", "ok"}, &out)
		if err != nil {
			t.Fatalf("un sumidero acumulativo está exento por nombre: %v", err)
		}
	})
	t.Run("retro-sellado sin instantánea", func(t *testing.T) {
		captureSealWarnings(t)
		captureWriterWarnings(t)
		ws := embeddedSealProject(t, "")
		if _, err := runSeal(t, ws.Master, "--infer"); err != nil {
			t.Fatalf("retro-sellar: %v", err)
		}
		if _, err := gateRecord(t, ws, "--gate", "spec", "--decision", "approved", "--reason", "ok"); err != nil {
			t.Fatalf("un kickoff sin instantánea exime al cambio (R15): %v", err)
		}
	})
}

// REQ-25.37 no añade más exenciones (supuesto S-7): un cambio sin ningún kickoff.yaml no está
// exento y se rechaza fuera de la rama que le correspondería.
func TestGuardGateRecordOfAChangeWithoutKickoffIsNotExempt(t *testing.T) {
	captureWriterWarnings(t)
	ws := embeddedSealProject(t, "")

	_, err := gateRecord(t, ws, "--gate", "spec", "--decision", "approved", "--reason", "ok")

	wantWrongBranch(t, err, "main", "feat/x", ws.Master)
}

// La guarda no bloquea lo que no puede comprobar: fuera de un repositorio git la escritura
// procede y avisa.
func TestGuardWarnsAndDoesNotBlockOutsideGit(t *testing.T) {
	warnings := captureWriterWarnings(t)
	root, _ := newGovernanceWorkspace(t, "inc-99-example")

	err := RunSDDGate([]string{"record", "--cwd", root, "--change", "inc-99-example", "--gate", "design", "--decision", "approved", "--reason", "ok"}, &bytes.Buffer{})

	if err != nil {
		t.Fatalf("fuera de git no se bloquea: %v", err)
	}
	if !strings.Contains(warnings.String(), "Aviso:") {
		t.Errorf("fuera de git debe avisar de que no se pudo comprobar la rama: %q", warnings.String())
	}
}

// G-10: `integrate prepare` y `verify` escriben `integration.yaml` y son del contexto `archive`;
// `check` y `show` solo leen y no se guardan.
func TestGuardIntegrateWritersUseTheArchiveContext(t *testing.T) {
	ws := guardProject(t)
	integrateGit(t, ws.Master, "switch", "-q", "feat/x")
	before := consentCLISnapshot(t, guardChangeRoot(ws))
	for _, verb := range []string{"prepare", "verify"} {
		var out bytes.Buffer
		err := RunSDDIntegrate([]string{verb, "--cwd", ws.Master, "--change", sealChange}, &out)

		wantWrongBranch(t, err, "feat/x", guardArchiveBranch, ws.Master)
		wantContinuation(t, err, " --archive")
		wantNoWrite(t, ws, before)
	}

	integrateGit(t, ws.Master, "switch", "-q", "-c", guardArchiveBranch)
	for _, verb := range []string{"prepare", "verify"} {
		err := RunSDDIntegrate([]string{verb, "--cwd", ws.Master, "--change", sealChange}, &bytes.Buffer{})
		wantNotRefusedByTheGuard(t, err)
	}
}

func TestGuardIntegrateCheckAndShowAreNotGuarded(t *testing.T) {
	ws := guardProject(t)
	for _, verb := range []string{"check", "show"} {
		err := RunSDDIntegrate([]string{verb, "--cwd", ws.Master, "--change", sealChange}, &bytes.Buffer{})
		wantNotRefusedByTheGuard(t, err)
	}
}

const guardCanonical = "## Requirements\n\n### Requirement: Listado\n\nEl sistema DEBE listar los elementos.\n"
const guardDelta = "## ADDED Requirements\n\n### Requirement: Etiquetado\n\nEl sistema DEBE etiquetar los elementos.\n"

func guardComposeFiles(t *testing.T, ws workspacetest.Workspace) (canonical, delta string) {
	t.Helper()
	canonical = filepath.Join(ws.Home, "specs", "widgets", "spec.md")
	delta = filepath.Join(guardChangeRoot(ws), "specs", "widgets", "spec.md")
	workspacetest.WriteFile(t, canonical, guardCanonical)
	workspacetest.WriteFile(t, delta, guardDelta)
	return canonical, delta
}

// REQ-25.19, escenario «`archive-compose` solo escribe en la rama de archivado»: con un delta bajo
// `changes/<c>/` y un fichero de salida, en `feat/x` se niega con la continuación `--archive` y en
// `feat/x-archive` escribe. Cubre los modos `--canonical --delta` y `--document --delta`.
func TestGuardArchiveComposeWithAChangeDeltaWritesOnlyOnTheArchiveBranch(t *testing.T) {
	for name, modeArgs := range map[string]func(canonical, delta string) []string{
		"canonical": func(canonical, delta string) []string { return []string{"--canonical", canonical, "--delta", delta} },
		"document": func(_, delta string) []string {
			return []string{"--document", "widgets", "--delta", delta}
		},
	} {
		t.Run(name, func(t *testing.T) {
			ws := guardProject(t)
			canonical, delta := guardComposeFiles(t, ws)
			output := filepath.Join(t.TempDir(), "spec-compuesta.md")
			args := append(modeArgs(canonical, delta), "--cwd", ws.Master, "--output", output)
			integrateGit(t, ws.Master, "switch", "-q", "feat/x")
			before := consentCLISnapshot(t, guardChangeRoot(ws))

			err := RunSDDArchiveCompose(args, &bytes.Buffer{})

			wantWrongBranch(t, err, "feat/x", guardArchiveBranch, ws.Master)
			wantContinuation(t, err, " --archive")
			t.Logf("rechazo de archive-compose --output <fichero> desde feat/x: %v", err)
			wantNoWrite(t, ws, before)
			if _, statErr := os.Stat(output); !errors.Is(statErr, os.ErrNotExist) {
				t.Errorf("el rechazo no debe escribir la salida (%v)", statErr)
			}

			integrateGit(t, ws.Master, "switch", "-q", "-c", guardArchiveBranch)
			if err := RunSDDArchiveCompose(args, &bytes.Buffer{}); err != nil {
				t.Fatalf("archive-compose en %s: %v", guardArchiveBranch, err)
			}
			if composed, readErr := os.ReadFile(output); readErr != nil || !strings.Contains(string(composed), "Etiquetado") {
				t.Errorf("la salida compuesta no se escribió: %q, %v", composed, readErr)
			}
		})
	}
}

// Decisión del usuario (2026-10-27): la previsualización `--output -` (stdout, por defecto) no
// escribe nada y no se guarda: procede desde `feat/x` y desde `main`.
func TestGuardArchiveComposePreviewToStdoutIsNotGuarded(t *testing.T) {
	ws := guardProject(t)
	canonical, delta := guardComposeFiles(t, ws)
	for _, name := range []string{"main", "feat/x"} {
		integrateGit(t, ws.Master, "switch", "-q", name)
		var out bytes.Buffer
		err := RunSDDArchiveCompose([]string{"--cwd", ws.Master, "--canonical", canonical, "--delta", delta}, &out)
		if err != nil {
			t.Fatalf("la previsualización desde %s no se guarda: %v", name, err)
		}
		if !strings.Contains(out.String(), "Etiquetado") {
			t.Errorf("la previsualización desde %s no compuso el delta:\n%s", name, out.String())
		}
	}
}

// Un delta fuera de la carpeta de un cambio no escribe en ninguna: no se guarda.
func TestGuardArchiveComposeOfALooseDeltaIsNotGuarded(t *testing.T) {
	ws := guardProject(t)
	canonical, _ := guardComposeFiles(t, ws)
	loose := filepath.Join(t.TempDir(), "widgets", "spec.md")
	workspacetest.WriteFile(t, loose, guardDelta)
	output := filepath.Join(t.TempDir(), "spec-compuesta.md")

	err := RunSDDArchiveCompose([]string{"--canonical", canonical, "--delta", loose, "--output", output}, &bytes.Buffer{})

	if err != nil {
		t.Fatalf("un delta suelto no se guarda: %v", err)
	}
}

// REQ-25.19: el modo cambio (`--change <c> --in-place`) es del contexto `archive`.
func TestGuardArchiveComposeInPlaceUsesTheArchiveContext(t *testing.T) {
	ws := guardProject(t)
	integrateGit(t, ws.Master, "switch", "-q", "feat/x")
	before := consentCLISnapshot(t, guardChangeRoot(ws))
	args := []string{"--cwd", ws.Master, "--change", sealChange, "--document", "widgets", "--in-place"}

	err := RunSDDArchiveCompose(args, &bytes.Buffer{})

	wantWrongBranch(t, err, "feat/x", guardArchiveBranch, ws.Master)
	wantContinuation(t, err, " --archive")
	wantNoWrite(t, ws, before)

	integrateGit(t, ws.Master, "switch", "-q", "-c", guardArchiveBranch)
	wantNotRefusedByTheGuard(t, RunSDDArchiveCompose(args, &bytes.Buffer{}))
}

// guardContinueProject es guardProject con un cambio listo cuya tarea toca una ruta fuera del
// planning home, de modo que `sdd continue` prepara el marcador de instancia.
func guardContinueProject(t *testing.T) workspacetest.Workspace {
	t.Helper()
	// Un solo rol (fullstack, el predeterminado): su plan de tareas es `tasks.md`.
	ws := guardProjectSealedWith(t, sealBaseArgs)
	root := guardChangeRoot(ws)
	workspacetest.WriteFile(t, filepath.Join(root, "specs", "feature", "spec.md"), "# Spec\n")
	workspacetest.WriteFile(t, filepath.Join(root, "design.md"), "# Design\n")
	workspacetest.WriteFile(t, filepath.Join(root, "tasks.md"), "- [ ] 1.1 Update `../source/main.go`\n")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "source", "main.go"), "package source\n")
	return ws
}

// REQ-25.19, REQ-25.39, REQ-25.42: `sdd continue` guarda el marcador de instancia
// (`.axiom-instance`), el único llamador de `PrepareChangeInstanceConsent`. Fuera de rama el status
// ya proyecta blocked(wrong_branch) (S9-c2): continue no prepara el marcador y muestra el bloqueo en
// lugar de rechazarse; en la rama del cambio lo escribe. La guarda A de continue solo cubre la
// ventana entre la resolución y la escritura.
func TestGuardContinueDoesNotPrepareTheInstanceMarkerOutsideTheChangeBranch(t *testing.T) {
	ws := guardContinueProject(t)
	marker := filepath.Join(guardChangeRoot(ws), ".axiom-instance")
	args := []string{sealChange, "--cwd", ws.Master, "--json"}

	var out bytes.Buffer
	err := RunSDDContinue(args, &out)

	if err != nil {
		t.Fatalf("fuera de rama continue refleja el bloqueo del status y no se rechaza: %v", err)
	}
	if _, statErr := os.Stat(marker); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("fuera de rama no se escribe el marcador (%v)", statErr)
	}
	for _, want := range []string{"blocked(wrong_branch)", "«main»", "«feat/x»", `\" --change x`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("la salida debe reflejar el bloqueo con %q: %s", want, out.String())
		}
	}

	integrateGit(t, ws.Master, "switch", "-q", "feat/x")
	if err := RunSDDContinue(args, &bytes.Buffer{}); err != nil {
		t.Fatalf("sdd continue en feat/x: %v", err)
	}
	if content, readErr := os.ReadFile(marker); readErr != nil || !strings.HasPrefix(strings.TrimSpace(string(content)), "sdd-") {
		t.Errorf("en la rama del cambio se escribe el marcador: %q, %v", content, readErr)
	}
}

// Una continuación que no va a escribir el marcador no se guarda: sin raíces sin autorizar,
// `sdd continue` solo lee y procede desde `main`.
func TestGuardContinueWithoutAMarkerToWriteIsNotGuarded(t *testing.T) {
	ws := guardProjectSealedWith(t, sealBaseArgs)
	root := guardChangeRoot(ws)
	workspacetest.WriteFile(t, filepath.Join(root, "specs", "feature", "spec.md"), "# Spec\n")
	workspacetest.WriteFile(t, filepath.Join(root, "design.md"), "# Design\n")
	workspacetest.WriteFile(t, filepath.Join(root, "tasks.md"), "- [ ] 1.1 Sin rutas fuera del cambio\n")

	if err := RunSDDContinue([]string{sealChange, "--cwd", ws.Master, "--json"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("sdd continue sin marcador que escribir desde main: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".axiom-instance")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("no debía escribirse el marcador (%v)", err)
	}
}

// design §2.11: `sdd-attempt grant` no escribe el marcador de instancia (lo exige ya escrito) y
// por eso no se guarda: sin marcador se niega por su propia causa y no lo crea, desde cualquier
// rama.
func TestGuardSDDAttemptGrantDoesNotWriteTheInstanceMarker(t *testing.T) {
	ws := guardContinueProject(t)
	marker := filepath.Join(guardChangeRoot(ws), ".axiom-instance")

	err := RunSDDAttempt([]string{
		"grant", "--cwd", ws.Master, "--change", sealChange, "--change-instance", "sdd-inexistente",
		"--request-id", "req-1", "--actor", "mantenedor", "--reason", "prueba", "--root", ws.Master,
	}, &bytes.Buffer{})

	if err == nil {
		t.Fatal("el grant sin marcador debía rechazarse")
	}
	wantNotRefusedByTheGuard(t, err)
	if _, statErr := os.Stat(marker); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("grant no debe escribir el marcador (%v)", statErr)
	}
}
