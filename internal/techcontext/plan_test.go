package techcontext

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCheckReportsTheContextEntries(t *testing.T) {
	f := newFixture(t, false)
	report := f.check(t)
	mustNoErrors(t, report)
	if report.Schema != CheckSchema || report.Change != "inc-99-ejemplo" {
		t.Errorf("informe = %+v", report)
	}
	if !report.Context.Declared || !report.Context.Present || report.Context.Decision != ContextEntries {
		t.Errorf("contexto = %+v", report.Context)
	}
	if len(report.Context.Entries) != 2 {
		t.Fatalf("entradas = %+v", report.Context.Entries)
	}
	first, second := report.Context.Entries[0], report.Context.Entries[1]
	if first.ID != "C1" || first.Document != "context/ARQUITECTURA.md" || first.Anchor != "Componentes > Servicio de pedidos" || first.Action != "modify" || first.Create {
		t.Errorf("primera entrada = %+v", first)
	}
	if second.ID != "C2" || second.Document != "../repo-backend/docs/API.md" || second.Action != "add" || second.Create {
		t.Errorf("segunda entrada = %+v (la sección Errores se añade a un documento existente)", second)
	}
}

// REQ-30.15 y REQ-30.29: `check` no escribe nada, tampoco en la carpeta del cambio.
func TestCheckDoesNotWrite(t *testing.T) {
	f := newFixture(t, false)
	before := snapshot(t, f.master)
	f.check(t)
	if changed := changedFiles(before, snapshot(t, f.master)); len(changed) != 0 {
		t.Fatalf("check no debe escribir, pero cambió %v", changed)
	}
}

func TestCheckMarksTheDocumentsThatArchiveCreates(t *testing.T) {
	f := newFixture(t, false)
	f.writeChange(t, DeltaFileName, "## Entrada: context/NUEVO.md\n\n- Ancla: Pagos\n- Acción: add\n- Motivo: documento nuevo\n")
	report := f.check(t)
	mustNoErrors(t, report)
	if len(report.Context.Entries) != 1 || !report.Context.Entries[0].Create {
		t.Fatalf("la entrada `add` de un documento inexistente lo crea: %+v", report.Context.Entries)
	}
}

func TestCheckWarnsWhenTheDeltaIsMissingAndNeverAssumesNoImpact(t *testing.T) {
	f := newFixture(t, false)
	f.removeChange(t, DeltaFileName)
	report := f.check(t)
	mustNoErrors(t, report)
	if report.Context.Present || report.Context.Decision != ContextNone {
		t.Errorf("sin delta la decisión es none, no «Sin impacto»: %+v", report.Context)
	}
	if len(report.Warnings) == 0 || !strings.Contains(report.Warnings[0], DeltaFileName) {
		t.Errorf("la ausencia del delta se avisa: %v", report.Warnings)
	}
}

func TestCheckAcceptsNoImpact(t *testing.T) {
	f := newFixture(t, false)
	f.writeChange(t, DeltaFileName, "# Delta\n\nSin impacto: solo cambia código interno\n")
	report := f.check(t)
	mustNoErrors(t, report)
	if report.Context.Decision != ContextNoImpact || report.Context.Reason != "solo cambia código interno" || len(report.Context.Entries) != 0 {
		t.Errorf("contexto = %+v", report.Context)
	}
}

func TestCheckWithoutTechnicalContextHasNothingToIntegrate(t *testing.T) {
	f := newFixture(t, false)
	writeTestFile(t, filepath.Join(f.specs, "axiom.yaml"), configWithoutContext)
	report := f.check(t)
	mustNoErrors(t, report)
	if report.Context.Declared || report.Context.Decision != ContextNone || len(report.Warnings) != 0 {
		t.Errorf("sin technical_context no hay nada que integrar ni que avisar: %+v %v", report.Context, report.Warnings)
	}
}

func TestCheckRejections(t *testing.T) {
	tests := []struct {
		name   string
		delta  string
		doc    map[string]string // ficheros del repositorio de especificaciones que se escriben antes
		errors []string
	}{
		{
			name:   "formato inválido",
			delta:  "## Entrada: context/ARQUITECTURA.md\n\n- Ancla: Componentes\n- Acción: borrar\n- Motivo: x\n",
			errors: []string{DeltaFileName, "REQ-24.31", "add, modify o remove", "Continuación"},
		},
		{
			name:   "ruta de referencia",
			delta:  "## Entrada: referencia/legado/VIEJO.md\n\n- Ancla: Viejo\n- Acción: modify\n- Motivo: x\n",
			errors: []string{"ruta de referencia", "nunca se modifican"},
		},
		{
			name:   "fuera de las rutas mantenidas",
			delta:  "## Entrada: otro/DOC.md\n\n- Ancla: Algo\n- Acción: add\n- Motivo: x\n",
			errors: []string{"no está bajo ninguna ruta mantenida"},
		},
		{
			name:   "ancla ambigua",
			delta:  "## Entrada: context/ARQUITECTURA.md\n\n- Ancla: Servicio de pedidos\n- Acción: modify\n- Motivo: x\n",
			doc:    map[string]string{"context/ARQUITECTURA.md": fixtureContext + "\n## Otro\n\n### Servicio de pedidos\n\nRepetido.\n"},
			errors: []string{"ambigua", "resuelve a 2 secciones"},
		},
		{
			name:   "modify sin ancla",
			delta:  "## Entrada: context/ARQUITECTURA.md\n\n- Ancla: Inexistente\n- Acción: modify\n- Motivo: x\n",
			errors: []string{"no resuelve a ninguna sección", "exige una sección existente"},
		},
		{
			name:   "modify de un documento que no existe",
			delta:  "## Entrada: context/NO-EXISTE.md\n\n- Ancla: Algo\n- Acción: modify\n- Motivo: x\n",
			errors: []string{"solo la acción 'add' puede crear un documento mantenido"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, false)
			f.writeChange(t, DeltaFileName, tt.delta)
			for rel, content := range tt.doc {
				writeTestFile(t, filepath.Join(f.specs, filepath.FromSlash(rel)), content)
			}
			mustErrors(t, f.check(t), tt.errors...)
		})
	}
}

// REQ-30.19: con contexto mantenido, un almacén de artefactos sin ficheros no tiene
// delta ni traza que integrar.
func TestCheckRequiresAFileArtifactStoreWithMaintainedContext(t *testing.T) {
	for _, store := range []string{"engram", "none"} {
		t.Run(store, func(t *testing.T) {
			f := newFixture(t, false)
			target := f.target(t)
			target.ArtifactStore = store
			report, err := Check(context.Background(), target)
			if err != nil {
				t.Fatal(err)
			}
			mustErrors(t, report, "technical_context.maintained", store, "artifact_store: openspec")
		})
	}
	t.Run("hybrid", func(t *testing.T) {
		f := newFixture(t, false)
		target := f.target(t)
		target.ArtifactStore = "hybrid"
		report, _ := Check(context.Background(), target)
		mustNoErrors(t, report)
	})
}

// REQ-30.30: una declaración `technical_context:` inválida falla cerrada.
func TestCheckFailsClosedOnAnInvalidTechnicalContext(t *testing.T) {
	f := newFixture(t, false)
	writeTestFile(t, filepath.Join(f.specs, "axiom.yaml"), strings.Replace(fixtureConfig, "paths:\n      - context/\n      - ../repo-backend/docs/\n", "paths: []\n", 1))
	mustErrors(t, f.check(t), "no es válida", "axiom workspace validate")
}

func fixedClock(t *testing.T) {
	t.Helper()
	original := now
	now = func() time.Time { return time.Date(2026, 3, 4, 5, 6, 7, 123, time.UTC) }
	t.Cleanup(func() { now = original })
}

func prepare(t *testing.T, f *fixture) (Trace, Report) {
	t.Helper()
	trace, report, err := Prepare(context.Background(), f.target(t))
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	return trace, report
}

func TestPrepareWritesOnlyTheTraceAndRegistersTheRepositories(t *testing.T) {
	f := newFixture(t, true)
	fixedClock(t)
	specsHead := strings.TrimSpace(runGit(t, f.specs, "rev-parse", "HEAD"))
	before := snapshot(t, f.master)

	trace, report := prepare(t, f)
	mustNoErrors(t, report)

	want := []string{"repo-specs/openspec/changes/inc-99-ejemplo/integration.yaml"}
	if changed := changedFiles(before, snapshot(t, f.master)); !reflect.DeepEqual(changed, want) {
		t.Fatalf("prepare solo escribe integration.yaml; cambió %v", changed)
	}
	if report.Schema != PrepareSchema || len(report.ContextItems) != 2 {
		t.Errorf("informe = %+v", report)
	}
	if got := report.ContextItems[0]; got.ID != "C1" || got.Action != "modify" || got.Text != "Texto nuevo para la sección." {
		t.Errorf("trabajo de edición = %+v", got)
	}
	if trace.Schema != PlanSchema || trace.Change != "inc-99-ejemplo" || trace.Context.Decision != ContextEntries || len(trace.Entries) != 2 {
		t.Fatalf("traza = %+v", trace)
	}
	if !trace.PreparedAt.Equal(time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)) {
		t.Errorf("prepared_at = %v, quería la hora truncada al segundo", trace.PreparedAt)
	}
	for i, entry := range trace.Entries {
		if entry.Kind != EntryContext || entry.State != StatePending {
			t.Errorf("entrada %d = %+v", i, entry)
		}
	}
	if trace.Entries[0].Action != ActionModified || trace.Entries[1].Action != ActionAdded {
		t.Errorf("acciones = %v, %v", trace.Entries[0].Action, trace.Entries[1].Action)
	}

	repos := map[string]string{}
	for _, repo := range trace.Repositories {
		repos[repo.Repo] = repo.Head
	}
	if repos["repo-specs"] != specsHead || repos["repo-backend"] == "" {
		t.Errorf("repositorios = %v, quería el HEAD de cada uno", repos)
	}
	read, err := ReadTrace(filepath.Join(f.changeRoot, TraceFileName))
	if err != nil || read.InputsSHA256 != trace.InputsSHA256 {
		t.Fatalf("el plan escrito no se lee: %v", err)
	}
}

func TestPrepareWithNoImpactRegistersTheDecisionWithoutEntries(t *testing.T) {
	f := newFixture(t, true)
	f.writeChange(t, DeltaFileName, "Sin impacto: solo cambia código interno\n")
	commitAll(t, f.specs, "sin impacto")
	trace, report := prepare(t, f)
	mustNoErrors(t, report)
	if trace.Context.Decision != ContextNoImpact || trace.Context.Reason != "solo cambia código interno" || len(trace.Entries) != 0 {
		t.Errorf("traza = %+v", trace.Context)
	}
}

func TestPrepareStopsWhenTheDeltaIsMissingWithMaintainedContext(t *testing.T) {
	f := newFixture(t, true)
	f.removeChange(t, DeltaFileName)
	_, report, err := Prepare(context.Background(), f.target(t))
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, DeltaFileName, "nunca se asume como «Sin impacto»", "Continuación")
	if _, statErr := os.Stat(filepath.Join(f.changeRoot, TraceFileName)); !errors.Is(statErr, os.ErrNotExist) {
		t.Error("con errores no se escribe el plan")
	}
}

func TestPrepareStopsOnUncommittedChangesInTheEditedDocuments(t *testing.T) {
	f := newFixture(t, true)
	writeTestFile(t, filepath.Join(f.specs, "context", "ARQUITECTURA.md"), fixtureContext+"\nCambio sin confirmar.\n")
	_, report, err := Prepare(context.Background(), f.target(t))
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, "context/ARQUITECTURA.md", "cambios en el árbol de trabajo", "axiom archive context prepare")
	if _, statErr := os.Stat(filepath.Join(f.changeRoot, TraceFileName)); !errors.Is(statErr, os.ErrNotExist) {
		t.Error("con la precondición sin cumplir no se escribe el plan")
	}
}

func TestPrepareRequiresEditAuthorizationForSiblingRepositories(t *testing.T) {
	f := newFixture(t, true)
	target := f.target(t)
	target.AllowedRoots = []string{target.Location.WorkspaceRoot}
	_, report, err := Prepare(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, "C2", "allowedEditRoots", "axiom sdd-attempt grant", "REQ-24.37")
}

func TestPrepareIsIdempotentAndRejectsChangedInputs(t *testing.T) {
	f := newFixture(t, true)
	fixedClock(t)
	first, _ := prepare(t, f)
	planFile := filepath.Join(f.changeRoot, TraceFileName)
	firstBytes, _ := os.ReadFile(planFile)

	second, report := prepare(t, f)
	mustNoErrors(t, report)
	if second.InputsSHA256 != first.InputsSHA256 {
		t.Error("volver a preparar reemite el plan existente")
	}
	if again, _ := os.ReadFile(planFile); string(again) != string(firstBytes) {
		t.Error("el plan reemitido debe conservar sus bytes")
	}

	f.writeChange(t, DeltaFileName, fixtureContextImpact+"\n## Entrada: context/OTRO.md\n\n- Ancla: X\n- Acción: add\n- Motivo: y\n")
	_, report, err := Prepare(context.Background(), f.target(t))
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, "ya no corresponde a las entradas actuales", "axiom archive context prepare")
}

func TestPrepareRejectsAnAlreadyIntegratedChange(t *testing.T) {
	f := newFixture(t, true)
	trace, _ := prepare(t, f)
	for i := range trace.Entries {
		trace.Entries[i].State = StateIntegrated
	}
	if err := WriteTrace(filepath.Join(f.changeRoot, TraceFileName), trace); err != nil {
		t.Fatal(err)
	}
	_, report, err := Prepare(context.Background(), f.target(t))
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, "ya integrado", "axiom archive context verify", "axiom archive context show")
}

func TestPrepareRejectsAForeignPlan(t *testing.T) {
	f := newFixture(t, true)
	foreign := strings.Replace(planV1, "documents: []", "documents:\n  - domain: modelo\n    path: 03.md\n    integration: compose\n    before_sha256: "+testDigest, 1)
	writeTestFile(t, filepath.Join(f.changeRoot, TraceFileName), foreign)
	_, report, err := Prepare(context.Background(), f.target(t))
	if err != nil {
		t.Fatal(err)
	}
	mustErrors(t, report, "no es un plan de contexto técnico", "axiom sdd integrate")
}

func TestPrepareWithoutTechnicalContextWritesAnEmptyPlan(t *testing.T) {
	f := newFixture(t, true)
	writeTestFile(t, filepath.Join(f.specs, "axiom.yaml"), configWithoutContext)
	commitAll(t, f.specs, "sin contexto")
	trace, report := prepare(t, f)
	mustNoErrors(t, report)
	if trace.Context.Decision != ContextNone || len(trace.Entries) != 0 {
		t.Errorf("traza = %+v", trace)
	}
}
