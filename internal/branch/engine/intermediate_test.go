package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Pruebas de la fusión intermedia de planificación (REQ-26.11, REQ-25.20, REQ-25.45, DD-09, supuestos A-7
// y A-8): un cambio cuya PR de una salida 2 ya está en la integración, con la marca de cesión en su
// `gates.yaml` y algún rol sin su `role-apply` aprobado, no figura como pendiente de archivar y
// `ensure --archive` lo rechaza. Con git real y sin red.

// approval es un registro `approved` de la puerta dada, con o sin la marca de cesión.
func approval(gate kickoff.GateKey, handedOff bool) kickoff.GateRecord {
	return kickoff.GateRecord{Gate: gate, Decision: kickoff.DecisionApproved, Reason: "conforme", HandedOff: handedOff, Actor: "prueba"}
}

// integrateChange confirma en la rama actual de repo la carpeta del cambio con su kickoff sellado con
// los roles dados y, si hay registros, su `gates.yaml`: es como si la PR de una salida 2 ya se
// hubiera fusionado en la integración.
func integrateChange(t *testing.T, repo, change string, roles []string, records ...kickoff.GateRecord) {
	t.Helper()
	sealChangeAt(t, repo, change, roles...)
	changeRoot := filepath.Join(repo, "openspec", "changes", change)
	for _, record := range records {
		if err := kickoff.AppendGate(changeRoot, record); err != nil {
			t.Fatalf("anexar la puerta %s de %s: %v", record.Gate, change, err)
		}
	}
	// El cerrojo del registro es estado local que nunca se versiona.
	_ = os.Remove(filepath.Join(changeRoot, ".gates.lock"))
	runGit(t, repo, "add", "-A")
	commit(t, repo, "integra "+change)
}

func pendingContains(t *testing.T, result branch.Result, change string) bool {
	t.Helper()
	for _, name := range pendingNames(t, result) {
		if name == change {
			return true
		}
	}
	return false
}

// REQ-26.11, escenarios 1 a 5: la fusión intermedia no figura; el cambio terminado, sí.
func TestPendingArchiveSkipsAnIntermediateMergeAndListsAFinishedChange(t *testing.T) {
	t.Parallel()
	roles := []string{"core", "web"}
	tests := []struct {
		name    string
		records []kickoff.GateRecord
		pending bool
	}{
		{"solo spec cedida", []kickoff.GateRecord{approval(kickoff.GateSpec, true)}, false},
		{"design cedida con los roles sin aprobar", []kickoff.GateRecord{
			approval(kickoff.GateSpec, false), approval(kickoff.GateDesign, true),
		}, false},
		{"un role-apply cedido con otro rol pendiente", []kickoff.GateRecord{
			approval(kickoff.GateSpec, false), approval(kickoff.GateDesign, false), approval(kickoff.GateTasks, false),
			approval(kickoff.RoleApplyGate("core"), true),
		}, false},
		{"todos los role-apply aprobados, el último cedido (A-7)", []kickoff.GateRecord{
			approval(kickoff.RoleApplyGate("core"), false), approval(kickoff.RoleApplyGate("web"), true),
		}, true},
		{"todos los role-apply aprobados y design cedida (A-7)", []kickoff.GateRecord{
			approval(kickoff.GateDesign, true),
			approval(kickoff.RoleApplyGate("core"), false), approval(kickoff.RoleApplyGate("web"), false),
		}, true},
		{"aprobaciones sin ninguna marca (A-8)", []kickoff.GateRecord{
			approval(kickoff.GateSpec, false), approval(kickoff.GateDesign, false),
		}, true},
		{"sin gates.yaml (A-8)", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := embeddedProject(t)
			integrateChange(t, repo, "x", roles, tt.records...)
			addChanges(t, repo, "", "y")

			result := status(t, repo, "", branch.LaneSDD)

			wantReady(t, result)
			if got := pendingContains(t, result, "x"); got != tt.pending {
				t.Errorf("x pendiente = %v, se esperaba %v (lista %q)", got, tt.pending, pendingNames(t, result))
			}
			if !pendingContains(t, result, "y") {
				t.Errorf("y, sin gates.yaml ni kickoff, figura como antes: %q", pendingNames(t, result))
			}
			if len(result.Warnings) != 0 {
				t.Errorf("avisos = %v, se esperaba ninguno", result.Warnings)
			}
		})
	}
}

// Con la marca y el roster vacío, la fusión sigue siendo intermedia (como LastBlockingRoleClosed, que nunca
// cierra con roster vacío).
func TestPendingArchiveTreatsAHandedOffChangeWithAnEmptyRosterAsIntermediate(t *testing.T) {
	t.Parallel()
	repo := embeddedProject(t)
	integrateChange(t, repo, "x", nil, approval(kickoff.GateSpec, true))

	wantPendingNames(t, status(t, repo, "", branch.LaneSDD))
}

// REQ-26.11, escenario 6 y A-8: un `gates.yaml` o un `kickoff.yaml` ilegibles, o la marca sin kickoff, se
// tratan como hasta ahora, sin error y sin aviso.
func TestPendingArchiveTreatsUnreadableFilesAsBefore(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		break_ func(t *testing.T, changeRoot string)
	}{
		{"gates.yaml ilegible", func(t *testing.T, changeRoot string) {
			writeFile(t, filepath.Join(changeRoot, "gates.yaml"), "{ esto: no: es yaml")
		}},
		{"kickoff.yaml ilegible con la marca presente", func(t *testing.T, changeRoot string) {
			writeFile(t, filepath.Join(changeRoot, "kickoff.yaml"), "{ esto: no: es yaml")
		}},
		{"kickoff.yaml ausente con la marca presente", func(t *testing.T, changeRoot string) {
			if err := os.Remove(filepath.Join(changeRoot, "kickoff.yaml")); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := embeddedProject(t)
			integrateChange(t, repo, "x", []string{"core", "web"}, approval(kickoff.GateSpec, true))
			tt.break_(t, filepath.Join(repo, "openspec", "changes", "x"))
			runGit(t, repo, "add", "-A")
			commit(t, repo, "estropea "+tt.name)

			result := status(t, repo, "", branch.LaneSDD)

			wantReady(t, result)
			wantPendingNames(t, result, "x")
			if len(result.Warnings) != 0 {
				t.Errorf("un contenido ilegible no es un aviso: %v", result.Warnings)
			}
		})
	}
}

// REQ-26.11, escenarios 7 y 8: la evaluación no usa la red y no bloquea `status --change`.
func TestIntermediateMergeUsesNoNetworkAndDoesNotBlockStatus(t *testing.T) {
	repo := embeddedProject(t)
	runGit(t, repo, "remote", "add", "origin", "https://ejemplo.invalid/repo.git")
	integrateChange(t, repo, "x", []string{"core", "web"}, approval(kickoff.GateSpec, true))
	seen := observe(t)

	listing := status(t, repo, "", branch.LaneSDD)
	forX := status(t, repo, "x", branch.LaneSDD)

	wantReady(t, listing)
	wantPendingNames(t, listing)
	wantReached(t, forX)
	if forX.PendingArchive == nil || forX.PendingArchive.Requested == nil || *forX.PendingArchive.Requested {
		t.Errorf("x es una fusión intermedia, no está pendiente de archivar: %+v", forX.PendingArchive)
	}
	if forX.PendingArchiveContinuation != "" {
		t.Errorf("sin pendiente no hay continuación de archivado: %q", forX.PendingArchiveContinuation)
	}
	for _, invocation := range *seen {
		if remoteOperation(invocation[1]) {
			t.Errorf("la evaluación no usa la red, pero se lanzó %v", invocation)
		}
	}
}

// Un fallo de git al leer los ficheros (no de contenido) trata el cambio como hasta ahora y avisa.
func TestPendingArchiveWarnsWhenGitFailsReadingTheFiles(t *testing.T) {
	t.Parallel()
	master := t.TempDir()
	location := workspace.Location{Master: master, WorkspaceRoot: master, PlanningHome: filepath.Join(master, "openspec")}
	fake := &fakeGit{
		topLevel: func(string) (string, error) { return master, nil },
		resolve:  func(string) (string, bool, error) { return "abc", true, nil },
		treeDirs: func(string, string) ([]string, bool, error) { return []string{"x", "y"}, true, nil },
		fileAt: func(_, relPath string) ([]byte, bool, error) {
			if strings.Contains(relPath, "/x/") {
				return nil, false, errors.New("fallo simulado de git")
			}
			return nil, false, nil
		},
	}

	pending, warnings := (&Engine{git: fake}).PendingArchive(context.Background(), location)

	if len(pending) != 2 {
		t.Errorf("pendientes = %v, se esperaba x e y como hasta ahora", pending)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "fallo simulado") || !strings.Contains(warnings[0], "x") {
		t.Errorf("avisos = %q, se esperaba uno que nombre x y el fallo", warnings)
	}
}

// REQ-26.11 y REQ-25.45 (S-16 enmendado): `ensure --archive` sobre una fusión intermedia rechaza con
// `change_not_integrated`, dice que es intermedia, no sugiere `--fetch` y no crea nada.
func TestEnsureArchiveRefusesAnIntermediateMergeWithoutSuggestingFetch(t *testing.T) {
	repo := embeddedCoreProject(t)
	integrateChange(t, repo, "x", []string{"core", "web"}, approval(kickoff.GateSpec, true))
	state := repositoryState(t, repo)
	seen := observe(t)

	result := ensureArchive(t, archiveRequest(repo))

	wantRefusal(t, result, branch.CodeChangeNotIntegrated, branch.StatusRefused)
	if result.Details["intermediate"] != "true" || result.Details["path"] != "openspec/changes/x/" {
		t.Errorf("details = %v, se esperaba intermediate=true y la carpeta del cambio", result.Details)
	}
	if !strings.Contains(result.Message, "intermedia") {
		t.Errorf("el mensaje %q debe decir que es una fusión intermedia", result.Message)
	}
	if strings.Contains(result.Continuation, "--fetch") {
		t.Errorf("la continuación %q no debe sugerir --fetch: la base ya tiene la carpeta", result.Continuation)
	}
	for _, want := range []string{"role-apply", "--change x --archive"} {
		if !strings.Contains(result.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
		}
	}
	wantNothingCreated(t, repo, state, "x")
	wantNoRegistryFile(t, repo, "x")
	for _, invocation := range *seen {
		if isWriteInvocation(invocation) || remoteOperation(invocation[1]) {
			t.Errorf("el rechazo no escribe ni toca la red, pero se lanzó %v", invocation)
		}
	}
}

// Sin marca, o con todos los role-apply aprobados, `--archive` se comporta como antes; con la carpeta
// ausente, rechaza con la sugerencia de `--fetch`.
func TestEnsureArchiveBehavesAsBeforeWithoutAnIntermediateMerge(t *testing.T) {
	t.Run("sin marca crea la rama de archivado", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		integrateChange(t, repo, "x", []string{"core", "web"}, approval(kickoff.GateSpec, false))
		request := archiveRequest(repo)
		request.Mode = branch.ModeInPlace

		wantReady(t, ensureArchive(t, request))
	})
	t.Run("todos los role-apply aprobados crea la rama de archivado", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		integrateChange(t, repo, "x", []string{"core", "web"},
			approval(kickoff.RoleApplyGate("core"), true), approval(kickoff.RoleApplyGate("web"), true))
		request := archiveRequest(repo)
		request.Mode = branch.ModeInPlace

		wantReady(t, ensureArchive(t, request))
	})
	t.Run("con la carpeta ausente sugiere --fetch", func(t *testing.T) {
		repo := embeddedCoreProject(t)

		result := ensureArchive(t, archiveRequest(repo))

		wantRefusal(t, result, branch.CodeChangeNotIntegrated, branch.StatusRefused)
		if !strings.Contains(result.Continuation, "--fetch") || result.Details["intermediate"] != "" {
			t.Errorf("continuación %q y detalles %v: se esperaba la sugerencia de --fetch y ningún detalle intermediate", result.Continuation, result.Details)
		}
	})
}

// REQ-26.11, último escenario de recogida: quien recoge la cesión obtiene la rama del cambio desde la
// base fusionada con `ensure`, sin operaciones remotas.
func TestEnsurePicksUpAnIntermediateMergeFromTheMergedBaseWithoutNetwork(t *testing.T) {
	repo := embeddedCoreProject(t)
	integrateChange(t, repo, "x", []string{"core"}, approval(kickoff.GateSpec, true))
	integration := runGit(t, repo, "rev-parse", "HEAD")
	seen := observe(t)

	result := ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x", Lane: branch.LaneSDD, Mode: branch.ModeInPlace})

	wantReady(t, result)
	if got := runGit(t, repo, "rev-parse", "refs/heads/feat/x"); got != integration {
		t.Errorf("la rama del cambio apunta a %s, se esperaba la base fusionada %s", got, integration)
	}
	for _, invocation := range *seen {
		if remoteOperation(invocation[1]) {
			t.Errorf("recoger la cesión no usa la red, pero se lanzó %v", invocation)
		}
	}
}
