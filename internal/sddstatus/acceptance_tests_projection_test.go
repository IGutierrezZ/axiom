package sddstatus

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const (
	acceptanceBlockOn  = "acceptance_tests:\n  enabled: true\n  environments: [dev, pre]\n"
	acceptanceBlockOff = "acceptance_tests:\n  enabled: false\n"
	acceptanceHeader   = "| ID | Cubre | Rol | Tipo | Prioridad | Entornos | Precondiciones | Pasos | Resultado esperado |\n|---|---|---|---|---|---|---|---|---|\n"
)

// acceptanceRow es una fila válida contra la delta de acceptanceChange.
func acceptanceRow(id, cubre, envs string) string {
	return "| " + id + " | " + cubre + " | backend | manual | alta | " + envs + " |  | abrir | ok |\n"
}

// acceptanceChange monta un cambio multirepo (canónico) con una delta que declara REQ-1.1 y
// REQ-1.2, sellado checkpointed con el rol backend, y el bloque dado en axiom.yaml.
func acceptanceChange(t *testing.T, blocks string) (ws workspacetest.Workspace, changeRoot string) {
	t.Helper()
	ws = workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), declCanonicalBase+blocks)
	changeRoot = seedReadyChange(t, ws.Specs, "inc-demo", planningHomeTasks)
	write(t, filepath.Join(changeRoot, "specs", "auth", "spec.md"),
		"## ADDED Requirements\n\n### Requirement: Uno REQ-1.1\n\nTexto.\n\n#### Scenario: A\n- **DADO** x\n\n#### Scenario: B\n- **DADO** x\n\n"+
			"### Requirement: Dos REQ-1.2\n\nTexto.\n\n#### Scenario: C\n- **DADO** x\n")
	_, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: "inc-demo", SealedBy: "test",
		Config: kickoff.FlowConfig{
			FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionCheckpointed, ExecutionStyleSource: "explicit",
			Roles: []kickoff.KickoffRole{{Role: "backend", GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"}},
		},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
	})
	if err != nil || !ok {
		t.Fatalf("kickoff.Seal() = (ok=%v, err=%v)", ok, err)
	}
	return ws, changeRoot
}

func acceptanceOf(t *testing.T, status Status) *AcceptanceTestsProjection {
	t.Helper()
	fields := statusJSON(t, status)
	if _, present := fields["acceptanceTests"]; !present {
		return nil
	}
	projection := decodeField[AcceptanceTestsProjection](t, fields, "acceptanceTests")
	return &projection
}

func locateFor(t *testing.T, ws workspacetest.Workspace) workspace.Location {
	t.Helper()
	location, err := workspace.Locate(ws.Master)
	if err != nil {
		t.Fatalf("Locate(%q): %v", ws.Master, err)
	}
	return location
}

func TestAcceptanceTestsProjectionOmittedWithoutBlockKeepsTheJSON(t *testing.T) {
	t.Parallel()
	bare := t.TempDir()
	seedReadyChange(t, bare, "demo", planningHomeTasks)
	reference := statusJSON(t, resolveFrom(t, bare, "demo"))

	ws := embeddedWithDeclaration(t, "")
	fields := statusJSON(t, resolveFrom(t, ws.Master, "demo"))
	if _, present := fields["acceptanceTests"]; present {
		t.Fatal("sin bloque ni pruebas.md la clave acceptanceTests debe omitirse")
	}
	if len(fields) != len(reference) {
		t.Fatalf("claves = %d, se esperaban las %d de un proyecto sin axiom.yaml", len(fields), len(reference))
	}
	if got := acceptanceOf(t, resolveFrom(t, ws.Master, "")); got != nil {
		t.Fatalf("sin cambio seleccionado ni bloque = %+v, se esperaba omitida", got)
	}
}

func TestAcceptanceTestsProjectionActiveWithBattery(t *testing.T) {
	t.Parallel()
	ws, changeRoot := acceptanceChange(t, acceptanceBlockOn)
	write(t, filepath.Join(changeRoot, "pruebas.md"), acceptanceHeader+
		acceptanceRow("P-01", "REQ-1.1#A; REQ-1.1#B", "dev")+
		acceptanceRow("P-02", "REQ-1.2#C", "pre; dev"))
	got := acceptanceOf(t, resolveFrom(t, ws.Master, "inc-demo"))
	if got == nil || !got.Active || !reflect.DeepEqual(got.Environments, []string{"dev", "pre"}) || got.Error != "" || got.Warning != "" {
		t.Fatalf("acceptanceTests = %+v", got)
	}
	b := got.Battery
	if b == nil || b.Path != filepath.Join(changeRoot, "pruebas.md") || b.Cases != 2 || b.Findings != 0 || b.Error != "" ||
		b.Scenarios.Covered != 3 || b.Scenarios.Total != 3 || b.Requirements.Covered != 2 || b.Requirements.Total != 2 {
		t.Fatalf("battery = %+v", b)
	}

	// Un caso con entorno fuera del dominio y escenarios sin cubrir dan hallazgos.
	write(t, filepath.Join(changeRoot, "pruebas.md"), acceptanceHeader+acceptanceRow("P-01", "REQ-1.1#A", "prod"))
	got = acceptanceOf(t, resolveFrom(t, ws.Master, "inc-demo"))
	if got == nil || got.Battery == nil || got.Battery.Findings < 2 || got.Battery.Scenarios.Covered != 1 {
		t.Fatalf("acceptanceTests = %+v, se esperaban hallazgos", got)
	}
}

func TestAcceptanceTestsProjectionDisabledAndInvalid(t *testing.T) {
	t.Parallel()
	t.Run("enabled false se proyecta no activa sin aviso", func(t *testing.T) {
		ws, _ := acceptanceChange(t, acceptanceBlockOff)
		got := acceptanceOf(t, resolveFrom(t, ws.Master, "inc-demo"))
		if got == nil || got.Active || got.Error != "" || got.Warning != "" || got.Battery != nil || got.Environments != nil {
			t.Fatalf("acceptanceTests = %+v", got)
		}
	})
	t.Run("bloque inválido proyecta el error y no asume activación", func(t *testing.T) {
		ws, _ := acceptanceChange(t, "acceptance_tests:\n  enabled: true\n  environments: [dev]\n  modo: x\n")
		got := acceptanceOf(t, resolveFrom(t, ws.Master, "inc-demo"))
		if got == nil || got.Active || got.Warning != "" {
			t.Fatalf("acceptanceTests = %+v", got)
		}
		for _, want := range []string{"modo", "axiom workspace validate --path"} {
			if !strings.Contains(got.Error, want) {
				t.Errorf("error %q debe contener %q", got.Error, want)
			}
		}
	})
	t.Run("regla de topología: enabled true en monorepo-embedded", func(t *testing.T) {
		ws := embeddedWithDeclaration(t, acceptanceBlockOn)
		got := acceptanceOf(t, resolveFrom(t, ws.Master, "demo"))
		if got == nil || got.Active || !strings.Contains(got.Error, "axiom workspace validate --path") {
			t.Fatalf("acceptanceTests = %+v, se esperaba el error de topología", got)
		}
	})
}

// El aviso por ausencia vive dentro de la proyección y no cambia el enrutado.
func TestAcceptanceTestsMissingWarningNeverChangesRouting(t *testing.T) {
	t.Parallel()
	active, _ := acceptanceChange(t, acceptanceBlockOn)
	inactive, _ := acceptanceChange(t, acceptanceBlockOff)
	on := resolveWithInstructions(t, active.Master, "inc-demo")
	off := resolveWithInstructions(t, inactive.Master, "inc-demo")

	got := acceptanceOf(t, on)
	if got == nil || !got.Active || !strings.HasPrefix(got.Warning, "pruebas_ausente:") || got.Battery != nil {
		t.Fatalf("acceptanceTests = %+v, se esperaba el aviso", got)
	}
	if on.NextRecommended != off.NextRecommended || !reflect.DeepEqual(on.Dependencies, off.Dependencies) ||
		!reflect.DeepEqual(on.BlockedReasons, off.BlockedReasons) || !reflect.DeepEqual(on.Notes, off.Notes) {
		t.Fatalf("el aviso cambió el enrutado: %q/%v/%v/%v frente a %q/%v/%v/%v",
			on.NextRecommended, on.Dependencies, on.BlockedReasons, on.Notes,
			off.NextRecommended, off.Dependencies, off.BlockedReasons, off.Notes)
	}
	for _, line := range append(append([]string{}, on.Notes...), on.BlockedReasons...) {
		if strings.Contains(line, "pruebas") {
			t.Errorf("el aviso no va a notes ni blockedReasons: %q", line)
		}
	}

	// Sin tareas no hay aviso.
	noTasks, changeRoot := acceptanceChange(t, acceptanceBlockOn)
	if err := os.Remove(filepath.Join(changeRoot, "tasks.md")); err != nil {
		t.Fatal(err)
	}
	if got := acceptanceOf(t, resolveFrom(t, noTasks.Master, "inc-demo")); got == nil || got.Warning != "" {
		t.Fatalf("sin tareas = %+v, no debe haber aviso", got)
	}
}

// Con la función no activa y pruebas.md presente se ejecuta Check sin validar Entornos.
func TestAcceptanceTestsFromAnInactiveFunctionSkipsEnvironments(t *testing.T) {
	t.Parallel()
	for name, block := range map[string]string{"enabled false": acceptanceBlockOff, "sin bloque": ""} {
		t.Run(name, func(t *testing.T) {
			ws, changeRoot := acceptanceChange(t, block)
			write(t, filepath.Join(changeRoot, "pruebas.md"), acceptanceHeader+
				acceptanceRow("P-01", "REQ-1.1#A; REQ-1.1#B", "inventado")+
				acceptanceRow("P-02", "REQ-9.9", "dev"))
			result := AcceptanceTestsFor(locateFor(t, ws), changeRoot)
			if result.Active || !result.Present || result.Error != "" {
				t.Fatalf("result = %+v", result)
			}
			var codes []string
			for _, finding := range result.Findings {
				codes = append(codes, string(finding.Code))
			}
			joined := strings.Join(codes, ",")
			if strings.Contains(joined, "entorno_invalido") {
				t.Fatalf("hallazgos = %s: no debe validar Entornos", joined)
			}
			if !strings.Contains(joined, "id_desconocido") || !strings.Contains(joined, "requisito_sin_cubrir") {
				t.Fatalf("hallazgos = %s: el resto de comprobaciones sí se ejecutan", joined)
			}
			if got := acceptanceOf(t, resolveFrom(t, ws.Master, "inc-demo")); got == nil || got.Active || got.Battery == nil || got.Battery.Findings == 0 {
				t.Fatalf("acceptanceTests = %+v", got)
			}
		})
	}
}

func TestAcceptanceTestsUnreadableKickoffGoesToError(t *testing.T) {
	t.Parallel()
	ws, changeRoot := acceptanceChange(t, acceptanceBlockOn)
	write(t, filepath.Join(changeRoot, "pruebas.md"), acceptanceHeader+acceptanceRow("P-01", "REQ-1.1#A", "dev"))
	write(t, filepath.Join(changeRoot, "kickoff.yaml"), "esto: [no es un kickoff\n")
	result := AcceptanceTestsFor(locateFor(t, ws), changeRoot)
	if result.Error == "" || len(result.Findings) != 0 || result.Report.Cases != 0 {
		t.Fatalf("result = %+v, se esperaba Error sin informe parcial", result)
	}
}

func TestAcceptanceTestsWithoutChangeRootReturnsOnlyTheDeclaration(t *testing.T) {
	t.Parallel()
	ws, _ := acceptanceChange(t, acceptanceBlockOn)
	result := AcceptanceTestsFor(locateFor(t, ws), "")
	if !result.Active || !reflect.DeepEqual(result.Environments, []string{"dev", "pre"}) || result.Present || result.Missing || result.Error != "" {
		t.Fatalf("result = %+v", result)
	}
}

// RF-TOP-20: un acceptance_tests inválido no bloquea el archivado.
func TestAcceptanceTestsInvalidBlockDoesNotBlockArchive(t *testing.T) {
	t.Parallel()
	reference := embeddedChange(t, "", "", tasksDone)
	invalid := embeddedChange(t, "acceptance_tests:\n  enabled: true\n  modo: x\n", "", tasksDone)
	want := resolveWithInstructions(t, reference.Master, "demo")
	got := resolveWithInstructions(t, invalid.Master, "demo")
	if got.Dependencies.Archive == DependencyBlocked || got.NextRecommended != want.NextRecommended ||
		!reflect.DeepEqual(got.Dependencies, want.Dependencies) || !reflect.DeepEqual(got.BlockedReasons, want.BlockedReasons) {
		t.Fatalf("archive = %q next = %q blocked = %v: el bloque inválido no debe bloquear (referencia %q/%q)",
			got.Dependencies.Archive, got.NextRecommended, got.BlockedReasons, want.Dependencies.Archive, want.NextRecommended)
	}
	if acceptanceOf(t, got) == nil {
		t.Fatal("el bloque inválido sí debe proyectarse")
	}
}

// Un cambio archivado no proyecta su pruebas.md: solo la declaración, si la hay.
func TestAcceptanceTestsOmittedForArchivedChanges(t *testing.T) {
	t.Parallel()
	ws, changeRoot := acceptanceChange(t, "")
	write(t, filepath.Join(changeRoot, "pruebas.md"), acceptanceHeader+acceptanceRow("P-01", "REQ-1.1#A", "dev"))
	location := locateFor(t, ws)
	name := "inc-demo"

	archived := Status{ChangeName: &name, ChangeRoot: &changeRoot, Archived: &ArchivedProjection{Path: "x"}}
	decorateAcceptanceTests(&archived, location)
	if archived.AcceptanceTests != nil {
		t.Fatalf("archivado sin bloque = %+v, se esperaba omitida", archived.AcceptanceTests)
	}
	live := Status{ChangeName: &name, ChangeRoot: &changeRoot}
	decorateAcceptanceTests(&live, location)
	if live.AcceptanceTests == nil || live.AcceptanceTests.Battery == nil {
		t.Fatalf("no archivado con pruebas.md = %+v, se esperaba la batería", live.AcceptanceTests)
	}

	ws2, changeRoot2 := acceptanceChange(t, acceptanceBlockOn)
	write(t, filepath.Join(changeRoot2, "pruebas.md"), acceptanceHeader+acceptanceRow("P-01", "REQ-1.1#A", "dev"))
	archived2 := Status{ChangeName: &name, ChangeRoot: &changeRoot2, Archived: &ArchivedProjection{Path: "x"}}
	decorateAcceptanceTests(&archived2, locateFor(t, ws2))
	if archived2.AcceptanceTests == nil || !archived2.AcceptanceTests.Active || archived2.AcceptanceTests.Battery != nil {
		t.Fatalf("archivado con bloque = %+v, se esperaba solo la declaración", archived2.AcceptanceTests)
	}
}
