package sddstatus

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const (
	tasksDone    = "- [x] 1.1 Trabajo hecho\n"
	tasksPending = "- [ ] 1.1 Trabajo pendiente\n"

	declInvalidBaseline = "baseline:\n  documents:\n    - domain: rf\n      path: rf.md\n      requirement:\n        heading: '^### (?P<id>RF-\\d+): (?P<title>.+)$'\n        level: 3\n        identity: nombre\n        render: '### {id}: {title}'\n"
	declValidBaseline   = "baseline:\n  documents:\n    - domain: modelo-de-datos\n      path: 03_Modelo_Datos.md\n"
	declMaintainedOnly  = "technical_context:\n  maintained:\n    entry: context/TECHNICAL_CONTEXT.md\n    paths: [context/]\n"
	declReferenceOnly   = "technical_context:\n  reference:\n    paths: [00_Glosario.md]\n"
)

// embeddedChange monta un proyecto embebido con los bloques dados, un
// openspec/config.yaml opcional y el cambio demo con las tareas dadas.
func embeddedChange(t *testing.T, blocks, openSpecConfig, tasks string) workspacetest.Workspace {
	t.Helper()
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+blocks)
	if openSpecConfig != "" {
		workspacetest.WriteFile(t, filepath.Join(ws.Home, "config.yaml"), openSpecConfig)
	}
	seedReadyChange(t, ws.Master, "demo", tasks)
	return ws
}

func resolveWithInstructions(t *testing.T, cwd, change string) Status {
	t.Helper()
	status, err := Resolve(ResolveOptions{CWD: cwd, ChangeName: change, IncludeInstructions: true})
	if err != nil {
		t.Fatalf("Resolve(%q): %v", cwd, err)
	}
	return status
}

func joined(lines []string) string { return strings.Join(lines, "\n") }

func wantContains(t *testing.T, what string, lines []string, wants ...string) {
	t.Helper()
	text := joined(lines)
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Errorf("%s no contiene %q:\n%s", what, want, text)
		}
	}
}

func wantNotContains(t *testing.T, what string, lines []string, unwanted string) {
	t.Helper()
	if strings.Contains(joined(lines), unwanted) {
		t.Errorf("%s no debe contener %q:\n%s", what, unwanted, joined(lines))
	}
}

// REQ-24.18: una declaración inválida bloquea solo el archivado.
func TestArchiveBlockedByAnInvalidDeclaration(t *testing.T) {
	t.Run("ruta archive: bloqueada, con la causa en notes y blockedReasons", func(t *testing.T) {
		ws := embeddedChange(t, declInvalidBaseline, "", tasksDone)
		status := resolveWithInstructions(t, ws.Master, "demo")
		if status.Dependencies.Archive != DependencyBlocked {
			t.Fatalf("dependencies.archive = %q, se esperaba blocked", status.Dependencies.Archive)
		}
		if status.NextRecommended != "resolve-blockers" {
			t.Fatalf("nextRecommended = %q, se esperaba resolve-blockers", status.NextRecommended)
		}
		for name, lines := range map[string][]string{"blockedReasons": status.BlockedReasons, "notes": status.Notes} {
			wantContains(t, name, lines,
				"La declaración de axiom.yaml no es válida",
				"'id' o 'text', no 'nombre'",
				"axiom workspace validate --path",
				ws.Master,
				"antes de archivar",
			)
		}
		if status.Baseline != nil {
			t.Error("una baseline inválida no se proyecta")
		}
	})

	t.Run("ruta apply: la causa va solo a notes y apply no se bloquea", func(t *testing.T) {
		ws := embeddedChange(t, declInvalidBaseline, "", tasksPending)
		status := resolveWithInstructions(t, ws.Master, "demo")
		if status.Dependencies.Archive != DependencyBlocked {
			t.Fatalf("dependencies.archive = %q, se esperaba blocked", status.Dependencies.Archive)
		}
		if status.Dependencies.Apply != DependencyReady || status.NextRecommended != "apply" || status.ApplyState != ApplyReady {
			t.Fatalf("apply = %q next = %q applyState = %q: la declaración no puede bloquear la implementación", status.Dependencies.Apply, status.NextRecommended, status.ApplyState)
		}
		if len(status.BlockedReasons) != 0 {
			t.Fatalf("blockedReasons = %v, el contrato prohíbe apply con blockedReasons no vacío", status.BlockedReasons)
		}
		wantContains(t, "notes", status.Notes, "La declaración de axiom.yaml no es válida", "axiom workspace validate")
	})

	t.Run("ruta de planificación: las dependencias de planificación no se bloquean", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+declInvalidBaseline)
		workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "demo", "proposal.md"), "# Propuesta\n")
		status := resolveWithInstructions(t, ws.Master, "demo")
		if status.NextRecommended != "spec" {
			t.Fatalf("nextRecommended = %q, se esperaba spec", status.NextRecommended)
		}
		if status.Dependencies.Proposal != DependencyAllDone {
			t.Fatalf("dependencies.proposal = %q, no debe cambiar", status.Dependencies.Proposal)
		}
		if len(status.BlockedReasons) != 0 {
			t.Fatalf("blockedReasons = %v, la planificación no se bloquea por esta causa", status.BlockedReasons)
		}
		wantContains(t, "notes", status.Notes, "La declaración de axiom.yaml no es válida")
	})

	t.Run("technical_context inválido sin afectar a la baseline", func(t *testing.T) {
		ws := embeddedChange(t, declValidBaseline+"technical_context:\n  maintained:\n    paths: []\n", "", tasksDone)
		status := resolveWithInstructions(t, ws.Master, "demo")
		if status.Dependencies.Archive != DependencyBlocked || status.NextRecommended != "resolve-blockers" {
			t.Fatalf("archive = %q next = %q", status.Dependencies.Archive, status.NextRecommended)
		}
		wantContains(t, "blockedReasons", status.BlockedReasons, "requiere al menos una ruta", "axiom workspace validate")
		if status.Baseline == nil || status.TechnicalContext != nil {
			t.Errorf("la baseline válida se proyecta y el contexto inválido no: baseline=%v contexto=%v", status.Baseline, status.TechnicalContext)
		}
	})

	t.Run("axiom.yaml ilegible sin puntero deja el status operativo", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), "workspace: [sin cerrar\n")
		seedReadyChange(t, ws.Master, "demo", tasksDone)
		status := resolveWithInstructions(t, ws.Master, "demo")
		if status.Dependencies.Archive != DependencyBlocked || status.NextRecommended != "resolve-blockers" {
			t.Fatalf("archive = %q next = %q", status.Dependencies.Archive, status.NextRecommended)
		}
		wantContains(t, "blockedReasons", status.BlockedReasons, "La declaración de axiom.yaml no es válida", "error al parsear axiom.yaml", "axiom workspace validate")
		// El aviso de resolución del caso 3 (REQ-24.7) sigue presente.
		wantContains(t, "notes", status.Notes, "se conserva la resolución embebida")
		if status.Dependencies.Proposal != DependencyAllDone || status.Dependencies.Apply == DependencyBlocked {
			t.Errorf("la planificación y apply no se bloquean: %+v", status.Dependencies)
		}
	})

	t.Run("axiom.yaml ilegible con puntero rechaza la resolución", func(t *testing.T) {
		ws := workspacetest.Canonical(t, workspacetest.WithUnreadableConfig())
		_, err := Resolve(ResolveOptions{CWD: ws.Master})
		var locateErr *workspace.LocateError
		if !errors.As(err, &locateErr) || locateErr.Kind != workspace.LocateConfigUnloadable {
			t.Fatalf("Resolve = %v, se esperaba un rechazo config_unloadable (no hay status que reportar)", err)
		}
	})

	t.Run("base inválida y declaración inválida sin puntero", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), "workspace: {name: x, topology: monorepo-embedded}\nbaseline: {}\n")
		seedReadyChange(t, ws.Master, "demo", tasksDone)
		status := resolveWithInstructions(t, ws.Master, "demo")
		if status.Dependencies.Archive != DependencyBlocked {
			t.Fatalf("archive = %q, la declaración inválida bloquea aunque la base falle", status.Dependencies.Archive)
		}
		wantContains(t, "notes", status.Notes, "la baseline declarada requiere al menos un documento")
	})

	t.Run("base inválida sin bloques nuevos no bloquea archive", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), "workspace: {name: x, topology: monorepo-embedded}\n")
		seedReadyChange(t, ws.Master, "demo", tasksDone)
		status := resolveWithInstructions(t, ws.Master, "demo")
		if status.Dependencies.Archive != DependencyReady || status.NextRecommended != "archive" {
			t.Fatalf("archive = %q next = %q: sin bloques nuevos la causa no existe", status.Dependencies.Archive, status.NextRecommended)
		}
		wantNotContains(t, "notes", status.Notes, "La declaración de axiom.yaml no es válida")
	})
}

// R1: un proyecto solo de openspec/, o con axiom.yaml sin bloques nuevos, no
// recibe causas ni notas aunque declare un almacén sin ficheros.
func TestArchiveNotBlockedWithoutDeclarations(t *testing.T) {
	bare := t.TempDir()
	seedReadyChange(t, bare, "demo", tasksDone)
	workspacetest.WriteFile(t, filepath.Join(bare, "openspec", "config.yaml"), "artifact_store: none\n")
	withConfig := embeddedChange(t, "", "artifact_store: none\n", tasksDone)

	for name, root := range map[string]string{"sin axiom.yaml": bare, "axiom.yaml sin bloques": withConfig.Master} {
		t.Run(name, func(t *testing.T) {
			status := resolveWithInstructions(t, root, "demo")
			if status.Dependencies.Archive != DependencyReady || status.NextRecommended != "archive" {
				t.Fatalf("archive = %q next = %q, no debe bloquearse", status.Dependencies.Archive, status.NextRecommended)
			}
			if len(status.BlockedReasons) != 0 || len(status.Notes) != 0 {
				t.Fatalf("blockedReasons = %v notes = %v, se esperaban vacíos", status.BlockedReasons, status.Notes)
			}
			wantNotContains(t, "phaseInstructions.archive", status.PhaseInstructions.Archive, "integrate prepare")
		})
	}
}

// REQ-24.36: la ausencia del delta de contexto nunca se asume, también en
// modalidad continua.
func TestArchiveBlockedWithoutTheContextDelta(t *testing.T) {
	t.Run("sin delta: bloquea archive y nombra el fichero", func(t *testing.T) {
		ws := embeddedChange(t, declMaintainedOnly, "", tasksDone)
		status := resolveWithInstructions(t, ws.Master, "demo")
		if status.Dependencies.Archive != DependencyBlocked || status.NextRecommended != "resolve-blockers" {
			t.Fatalf("archive = %q next = %q", status.Dependencies.Archive, status.NextRecommended)
		}
		want := filepath.Join(ws.Home, "changes", "demo", "context-impact.md")
		for name, lines := range map[string][]string{"blockedReasons": status.BlockedReasons, "notes": status.Notes} {
			wantContains(t, name, lines, "Falta el delta de contexto", want, "Sin impacto")
		}
		if status.TechnicalContext == nil || status.TechnicalContext.Maintained == nil {
			t.Error("el contexto sigue proyectado aunque falte el delta")
		}
	})

	t.Run("con delta no bloquea", func(t *testing.T) {
		ws := embeddedChange(t, declMaintainedOnly, "", tasksDone)
		workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "demo", "context-impact.md"), "# Delta de contexto: demo\n\nSin impacto: no toca el contexto.\n")
		status := resolveWithInstructions(t, ws.Master, "demo")
		if status.Dependencies.Archive != DependencyReady || status.NextRecommended != "archive" {
			t.Fatalf("archive = %q next = %q", status.Dependencies.Archive, status.NextRecommended)
		}
		wantNotContains(t, "notes", status.Notes, "Falta el delta de contexto")
	})

	t.Run("el delta de contexto no altera los localizadores", func(t *testing.T) {
		ws := embeddedChange(t, declMaintainedOnly, "", tasksDone)
		without := resolveWithInstructions(t, ws.Master, "demo")
		workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "demo", "context-impact.md"), "# Delta\n")
		with := resolveWithInstructions(t, ws.Master, "demo")
		if joined(with.ArtifactPaths.Proposal) != joined(without.ArtifactPaths.Proposal) || len(with.ArtifactPaths.Specs) != len(without.ArtifactPaths.Specs) ||
			len(with.ContextFiles.Tasks) != len(without.ContextFiles.Tasks) || len(with.Artifacts) != len(without.Artifacts) {
			t.Errorf("artifacts y localizadores no cambian con el delta: %+v frente a %+v", with.ArtifactPaths, without.ArtifactPaths)
		}
	})

	t.Run("almacén hybrid declarado también lo exige", func(t *testing.T) {
		ws := embeddedChange(t, declMaintainedOnly, "artifact_store: hybrid\n", tasksDone)
		defer stubEngramExport(t, nil)()
		status := resolveWithInstructions(t, ws.Master, "demo")
		if status.Dependencies.Archive != DependencyBlocked {
			t.Fatalf("archive = %q", status.Dependencies.Archive)
		}
		wantContains(t, "notes", status.Notes, "Falta el delta de contexto")
	})

	t.Run("solo contexto de referencia no lo exige", func(t *testing.T) {
		ws := embeddedChange(t, declReferenceOnly, "", tasksDone)
		status := resolveWithInstructions(t, ws.Master, "demo")
		if status.Dependencies.Archive != DependencyReady || len(status.Notes) != 0 {
			t.Fatalf("archive = %q notes = %v", status.Dependencies.Archive, status.Notes)
		}
	})

	t.Run("modalidad continua", func(t *testing.T) {
		ws := embeddedChange(t, declMaintainedOnly, "", tasksDone)
		sealFullstackKickoff(t, filepath.Join(ws.Home, "changes", "demo"), "demo", kickoff.ExecutionContinuous)
		status := resolveWithInstructions(t, ws.Master, "demo")
		if status.Governance != nil {
			t.Fatalf("una modalidad continua no tiene compuertas: %+v", status.Governance)
		}
		if status.Dependencies.Archive != DependencyBlocked || status.NextRecommended != "resolve-blockers" {
			t.Fatalf("archive = %q next = %q: la ausencia del delta bloquea también en modalidad continua", status.Dependencies.Archive, status.NextRecommended)
		}
		wantContains(t, "blockedReasons", status.BlockedReasons, "Falta el delta de contexto")
	})
}

// REQ-24.62: una baseline del proyecto o un contexto mantenido exigen un almacén de ficheros.
func TestArchiveBlockedByAFilelessStore(t *testing.T) {
	t.Run("none declarado con el cambio en disco", func(t *testing.T) {
		ws := embeddedChange(t, declValidBaseline, "artifact_store: none\n", tasksPending)
		status := resolveWithInstructions(t, ws.Master, "demo")
		if status.Dependencies.Archive != DependencyBlocked {
			t.Fatalf("archive = %q, el almacén declarado prevalece sobre el hallado en disco", status.Dependencies.Archive)
		}
		wantContains(t, "notes", status.Notes, "La baseline del proyecto (`baseline`) exige un almacén de ficheros", "`none`", "`artifact_store: openspec` o `hybrid` en `openspec/config.yaml`")
		if status.Baseline == nil {
			t.Error("la baseline sigue proyectada: la declaración no es inválida")
		}
		if status.Dependencies.Apply != DependencyReady || status.NextRecommended != "apply" || len(status.BlockedReasons) != 0 {
			t.Errorf("apply = %q next = %q blockedReasons = %v: la causa solo bloquea archive", status.Dependencies.Apply, status.NextRecommended, status.BlockedReasons)
		}
	})

	t.Run("none declarado en la ruta archive va a blockedReasons", func(t *testing.T) {
		ws := embeddedChange(t, declValidBaseline+declMaintainedOnly, "artifact_store: none\n", tasksDone)
		status := resolveWithInstructions(t, ws.Master, "demo")
		if status.Dependencies.Archive != DependencyBlocked || status.NextRecommended != "resolve-blockers" {
			t.Fatalf("archive = %q next = %q", status.Dependencies.Archive, status.NextRecommended)
		}
		wantContains(t, "blockedReasons", status.BlockedReasons,
			"La baseline del proyecto (`baseline`) y el contexto técnico mantenido (`technical_context.maintained`) exigen un almacén de ficheros", "`none`")
	})

	engramObservations := func() []engramObservation {
		return []engramObservation{
			{Title: "sdd/add-auth/proposal", Content: "## Proposal\n", Project: "demo", Scope: "project"},
			{Title: "sdd/add-auth/spec", Content: "## Requirements\n- SHALL work\n", Project: "demo", Scope: "project"},
			{Title: "sdd/add-auth/design", Content: "## Design\n", Project: "demo", Scope: "project"},
			{Title: "sdd/add-auth/tasks", Content: "- [ ] 1.1 Wire routes\n", Project: "demo", Scope: "project"},
		}
	}

	t.Run("engram declarado con el cambio en Engram", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+declValidBaseline)
		workspacetest.WriteFile(t, filepath.Join(ws.Home, "config.yaml"), "artifact_store: engram\n")
		t.Setenv("ENGRAM_PROJECT", "demo")
		defer stubEngramExport(t, engramObservations())()
		status := resolveWithInstructions(t, ws.Master, "")
		if status.ArtifactStore != ArtifactStoreEngram || status.ChangeName == nil || *status.ChangeName != "add-auth" {
			t.Fatalf("store = %q change = %v, el cambio debe resolverse en Engram", status.ArtifactStore, status.ChangeName)
		}
		if status.Dependencies.Archive != DependencyBlocked {
			t.Fatalf("archive = %q", status.Dependencies.Archive)
		}
		wantContains(t, "notes", status.Notes, "La baseline del proyecto (`baseline`) exige un almacén de ficheros", "`engram`")
		if status.Baseline == nil {
			t.Error("la baseline sigue proyectada")
		}
		if status.Dependencies.Apply != DependencyReady || status.NextRecommended != "apply" || len(status.BlockedReasons) != 0 {
			t.Errorf("apply = %q next = %q blockedReasons = %v", status.Dependencies.Apply, status.NextRecommended, status.BlockedReasons)
		}
	})

	t.Run("sin declaración de almacén, resuelto en Engram", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+declMaintainedOnly)
		workspacetest.MkdirAll(t, filepath.Join(ws.Master, ".engram"))
		t.Setenv("ENGRAM_PROJECT", "demo")
		defer stubEngramExport(t, engramObservations())()
		status := resolveWithInstructions(t, ws.Master, "")
		if status.ArtifactStore != ArtifactStoreEngram {
			t.Fatalf("store = %q, se esperaba engram resuelto", status.ArtifactStore)
		}
		if status.Dependencies.Archive != DependencyBlocked {
			t.Fatalf("archive = %q: el almacén resuelto cuenta sin declaración", status.Dependencies.Archive)
		}
		wantContains(t, "notes", status.Notes, "El contexto técnico mantenido (`technical_context.maintained`)", "`engram`")
	})

	t.Run("solo contexto de referencia con engram no bloquea", func(t *testing.T) {
		ws := embeddedChange(t, declReferenceOnly, "artifact_store: none\n", tasksDone)
		status := resolveWithInstructions(t, ws.Master, "demo")
		if status.Dependencies.Archive != DependencyReady || len(status.Notes) != 0 {
			t.Fatalf("archive = %q notes = %v", status.Dependencies.Archive, status.Notes)
		}
	})

	t.Run("almacenes de ficheros no bloquean", func(t *testing.T) {
		for _, store := range []string{"openspec", "hybrid"} {
			ws := embeddedChange(t, declValidBaseline, "artifact_store: "+store+"\n", tasksDone)
			func() {
				defer stubEngramExport(t, nil)()
				status := resolveWithInstructions(t, ws.Master, "demo")
				if status.Dependencies.Archive != DependencyReady || len(status.Notes) != 0 {
					t.Errorf("%s: archive = %q notes = %v", store, status.Dependencies.Archive, status.Notes)
				}
			}()
		}
	})
}

// D-05: con baseline del proyecto o contexto mantenido, phaseInstructions.archive
// nombra el verbo que prepara la integración.
func TestArchiveInstructionsNameIntegratePrepare(t *testing.T) {
	for name, blocks := range map[string]string{"baseline": declValidBaseline, "contexto mantenido": declMaintainedOnly} {
		t.Run(name, func(t *testing.T) {
			ws := embeddedChange(t, blocks, "", tasksPending)
			status := resolveWithInstructions(t, ws.Master, "demo")
			wantContains(t, "phaseInstructions.archive", status.PhaseInstructions.Archive,
				"axiom sdd integrate prepare --cwd", ws.Master, "--change demo")
		})
	}
	t.Run("sin baseline del proyecto ni contexto mantenido", func(t *testing.T) {
		ws := embeddedChange(t, declReferenceOnly, "", tasksPending)
		status := resolveWithInstructions(t, ws.Master, "demo")
		wantNotContains(t, "phaseInstructions.archive", status.PhaseInstructions.Archive, "integrate prepare")
	})
	t.Run("sin instrucciones pedidas no se generan", func(t *testing.T) {
		ws := embeddedChange(t, declValidBaseline, "", tasksPending)
		status, err := Resolve(ResolveOptions{CWD: ws.Master, ChangeName: "demo"})
		if err != nil {
			t.Fatal(err)
		}
		if status.PhaseInstructions != nil {
			t.Errorf("phaseInstructions = %+v, solo se generan si se piden", status.PhaseInstructions)
		}
	})
}

// R30: en multirepo, una ruta de tasks.md que nombra un repositorio de código
// hermano sin `../` ni ruta absoluta se avisa.
func TestStatusWarnsAboutSiblingRepositoryPathsWithoutDotDot(t *testing.T) {
	canonical := func(t *testing.T, tasks string) workspacetest.Workspace {
		t.Helper()
		ws := workspacetest.Canonical(t)
		seedReadyChange(t, ws.Specs, "inc-demo", tasks)
		workspacetest.WriteFile(t, filepath.Join(ws.Specs, "docs", "notas.md"), "# notas\n")
		return ws
	}

	t.Run("ruta relativa al repositorio hermano", func(t *testing.T) {
		ws := canonical(t, "- [ ] 1.1 Modificar `repo-backend/src/servicio.go` y `repo-frontend/app/vista.ts`\n- [ ] 1.2 Otra vez `repo-backend/src/servicio.go`\n")
		status := resolveWithInstructions(t, ws.Master, "inc-demo")
		wantContains(t, "notes", status.Notes, "tasks.md cita 2 ruta(s) relativa(s)", "`repo-backend/src/servicio.go`", "`repo-frontend/app/vista.ts`", "../<repo>/...", "ruta absoluta")
		if len(status.BlockedReasons) != 0 || status.NextRecommended != "apply" {
			t.Errorf("el aviso es informativo: blockedReasons = %v next = %q", status.BlockedReasons, status.NextRecommended)
		}
	})

	t.Run("con ../ o ruta absoluta no avisa", func(t *testing.T) {
		abs := filepath.ToSlash(filepath.Join(workspacetest.Canonical(t).Backend, "src", "x.go"))
		ws := canonical(t, "- [ ] 1.1 Modificar `../repo-backend/src/servicio.go` y `"+abs+"`\n")
		status := resolveWithInstructions(t, ws.Master, "inc-demo")
		wantNotContains(t, "notes", status.Notes, "tasks.md cita")
	})

	t.Run("rutas que existen bajo el repositorio de especificaciones o no son de un repositorio hermano", func(t *testing.T) {
		ws := canonical(t, "- [ ] 1.1 Leer `docs/notas.md`, `src/otro.go`, `repo-backend` y `repo-inexistente/a.go`\n")
		status := resolveWithInstructions(t, ws.Master, "inc-demo")
		wantNotContains(t, "notes", status.Notes, "tasks.md cita")
	})

	t.Run("una ruta que existe bajo workspaceRoot no avisa", func(t *testing.T) {
		ws := canonical(t, "- [ ] 1.1 Leer `repo-backend/doc.md`\n")
		workspacetest.WriteFile(t, filepath.Join(ws.Specs, "repo-backend", "doc.md"), "# existe\n")
		status := resolveWithInstructions(t, ws.Master, "inc-demo")
		wantNotContains(t, "notes", status.Notes, "tasks.md cita")
	})

	t.Run("monorepo embebido no avisa", func(t *testing.T) {
		ws := embeddedChange(t, "", "", "- [ ] 1.1 Modificar `core/src/a.go`\n")
		status := resolveWithInstructions(t, ws.Master, "demo")
		wantNotContains(t, "notes", status.Notes, "tasks.md cita")
	})
}
