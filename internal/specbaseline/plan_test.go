package specbaseline

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

func samplePlan() Plan {
	return Plan{
		Schema:       PlanSchema,
		Change:       "inc-99-ejemplo",
		PreparedAt:   time.Date(2026, 10, 4, 10, 0, 0, 0, time.UTC),
		InputsSHA256: sha256Hex([]byte("entradas")),
		Options:      PlanOptions{SupersededRequirements: []string{}},
		Repositories: []PlanRepository{
			{Repo: "repo-backend", Head: strings.Repeat("7a", 20), Dirty: []DirtyEntry{}},
			{Repo: "specs", Head: strings.Repeat("9c", 20), Dirty: []DirtyEntry{{Path: "notas/borrador.md", SHA256: sha256Hex([]byte("borrador"))}, {Path: "viejo.md"}}},
		},
		IDMap: []IDAssignment{{Provisional: "RF-9001", Definitive: "RF-008", Domain: "requisitos-funcionales"}},
		Documents: []PlanDocument{
			{Domain: "requisitos-funcionales", Path: "01_Requisitos_Funcionales.md", Integration: workspace.IntegrationCompose, BeforeSHA256: sha256Hex([]byte("antes")), AfterSHA256: sha256Hex([]byte("después")), State: StatePending},
			{Domain: "modelo-de-datos", Path: "03_Modelo_Datos.md", Integration: workspace.IntegrationDirected, BeforeSHA256: sha256Hex([]byte("modelo"))},
		},
		Context: ContextDecision{Decision: ContextEntries},
		Entries: []TraceEntry{
			{ID: "RF-008", Provisional: "RF-9001", Kind: EntryCompose, Domain: "requisitos-funcionales", Document: "01_Requisitos_Funcionales.md", Anchor: "RF-008", Action: ActionAdded, State: StatePending, ScenariosOmitted: true},
			{ID: "Reintento de pago", Kind: EntryDirected, Domain: "modelo-de-datos", Document: "03_Modelo_Datos.md", Anchor: "inc-99-ejemplo", Action: ActionAdded, State: StatePending},
			{ID: "C1", Kind: EntryContext, Document: "../repo-backend/docs/API.md", Anchor: "Errores > Reintentos", Action: ActionModified, Reason: "se añade el reintento", State: StatePending},
		},
	}
}

func TestPlanRoundTripsAndIsDeterministic(t *testing.T) {
	plan := samplePlan()
	first, err := MarshalPlan(plan)
	if err != nil {
		t.Fatalf("MarshalPlan: %v", err)
	}
	second, _ := MarshalPlan(plan)
	if string(first) != string(second) {
		t.Fatal("la misma entrada debe producir los mismos bytes")
	}
	text := string(first)
	for _, want := range []string{"schema: axiom.sdd-integration/v1", "change: inc-99-ejemplo", "inputs_sha256:", "before_sha256:", "after_sha256:", "head:", "dirty:", "provisional: RF-9001", "definitive: RF-008", "prepared_at: 2026-10-04T10:00:00Z"} {
		if !strings.Contains(text, want) {
			t.Errorf("integration.yaml debe contener %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, `\`) || strings.Contains(text, "C:/") {
		t.Errorf("el plan guarda rutas relativas con '/' y nunca rutas de la máquina:\n%s", text)
	}
	decoded, err := DecodePlan(first)
	if err != nil {
		t.Fatalf("DecodePlan: %v", err)
	}
	if !reflect.DeepEqual(decoded, plan) {
		t.Fatalf("la ida y vuelta cambia el plan:\n got %#v\nwant %#v", decoded, plan)
	}
}

func TestPlanFileIsWrittenAtomicallyAndReadBack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, PlanFileName)
	if _, err := ReadPlan(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("sin plan, ReadPlan debe devolver un error que envuelva fs.ErrNotExist: %v", err)
	}
	plan := samplePlan()
	if err := WritePlan(path, plan); err != nil {
		t.Fatalf("WritePlan: %v", err)
	}
	got, err := ReadPlan(path)
	if err != nil || !reflect.DeepEqual(got, plan) {
		t.Fatalf("ReadPlan = %v, %v", got, err)
	}
	plan.Change = "inc-99-otro"
	if err := WritePlan(path, plan); err != nil {
		t.Fatalf("reescribir: %v", err)
	}
	if got, _ := ReadPlan(path); got.Change != "inc-99-otro" {
		t.Errorf("la reescritura debe sustituir el plan: %q", got.Change)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("no deben quedar temporales junto al plan: %v", entries)
	}
}

func TestDecodePlanIsStrict(t *testing.T) {
	good, err := MarshalPlan(samplePlan())
	if err != nil {
		t.Fatal(err)
	}
	raw := []struct {
		name string
		data string
		want string
	}{
		{name: "vacío", data: "", want: "vacío"},
		{name: "clave desconocida", data: string(good) + "extra: 1\n", want: "extra"},
		{name: "dos documentos YAML", data: string(good) + "---\nschema: otro\n", want: "un solo documento"},
		{name: "no es YAML", data: "schema: [sin cerrar\n", want: "integration.yaml"},
	}
	for _, tt := range raw {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodePlan([]byte(tt.data))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("DecodePlan error = %v, quería que contuviera %q", err, tt.want)
			}
		})
	}

	mutated := []struct {
		name   string
		mutate func(p *Plan)
		want   string
	}{
		{name: "esquema distinto", mutate: func(p *Plan) { p.Schema = "axiom.sdd-integration/v2" }, want: "axiom.sdd-integration/v1"},
		{name: "sin esquema", mutate: func(p *Plan) { p.Schema = "" }, want: "schema"},
		{name: "sin cambio", mutate: func(p *Plan) { p.Change = "" }, want: "change"},
		{name: "sin fecha de preparación", mutate: func(p *Plan) { p.PreparedAt = time.Time{} }, want: "prepared_at"},
		{name: "huella de entradas inválida", mutate: func(p *Plan) { p.InputsSHA256 = "xyz" }, want: "inputs_sha256"},
		{name: "documento con barra invertida", mutate: func(p *Plan) { p.Documents[1].Path = `carpeta\03_Modelo_Datos.md` }, want: "'/'"},
		{name: "documento absoluto", mutate: func(p *Plan) { p.Documents[1].Path = "/etc/03_Modelo_Datos.md" }, want: "relativa"},
		{name: "documento con unidad de disco", mutate: func(p *Plan) { p.Documents[1].Path = "C:/03_Modelo_Datos.md" }, want: "relativa"},
		{name: "documento que sale de workspaceRoot", mutate: func(p *Plan) { p.Documents[1].Path = "../03_Modelo_Datos.md" }, want: "no puede salir"},
		{name: "repositorio absoluto", mutate: func(p *Plan) { p.Repositories[0].Repo = "/srv/repo-backend" }, want: "relativa"},
		{name: "repositorio con barra invertida", mutate: func(p *Plan) { p.Repositories[0].Repo = `..\repo-backend` }, want: "'/'"},
		{name: "repositorio repetido", mutate: func(p *Plan) { p.Repositories[1].Repo = p.Repositories[0].Repo }, want: "repetido"},
		{name: "ruta sucia que sale del repositorio", mutate: func(p *Plan) { p.Repositories[1].Dirty[1].Path = "../viejo.md" }, want: "no puede salir"},
		{name: "huella sucia corta", mutate: func(p *Plan) { p.Repositories[1].Dirty[0].SHA256 = "abc" }, want: "sha256"},
		{name: "huella before inválida", mutate: func(p *Plan) { p.Documents[1].BeforeSHA256 = "nada" }, want: "before_sha256"},
		{name: "huella after inválida", mutate: func(p *Plan) { p.Documents[0].AfterSHA256 = "nada" }, want: "after_sha256"},
		{name: "HEAD inválido", mutate: func(p *Plan) { p.Repositories[0].Head = "no-es-un-commit" }, want: "head"},
		{name: "tipo de entrada inválido", mutate: func(p *Plan) { p.Entries[1].Kind = "manual" }, want: "kind"},
		{name: "acción inválida", mutate: func(p *Plan) { p.Entries[1].Action = "borrado" }, want: "action"},
		{name: "estado de entrada inválido", mutate: func(p *Plan) { p.Entries[0].State = "hecho" }, want: "state"},
		{name: "estado de documento inválido", mutate: func(p *Plan) { p.Documents[0].State = "hecho" }, want: "state"},
		{name: "decisión de contexto inválida", mutate: func(p *Plan) { p.Context.Decision = "quizá" }, want: "decision"},
		{name: "modo de integración inválido", mutate: func(p *Plan) { p.Documents[1].Integration = "manual" }, want: "integration"},
		{name: "entrada con barra invertida", mutate: func(p *Plan) { p.Entries[2].Document = `..\repo-backend\docs\API.md` }, want: "'/'"},
		{name: "entrada absoluta", mutate: func(p *Plan) { p.Entries[2].Document = "/srv/API.md" }, want: "relativa"},
		{name: "entrada sin documento", mutate: func(p *Plan) { p.Entries[2].Document = "" }, want: "document"},
	}
	for _, tt := range mutated {
		t.Run(tt.name, func(t *testing.T) {
			plan := samplePlan()
			tt.mutate(&plan)
			data, err := MarshalPlan(plan)
			if err != nil {
				t.Fatalf("MarshalPlan no valida: %v", err)
			}
			if _, err := DecodePlan(data); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("DecodePlan error = %v, quería que contuviera %q", err, tt.want)
			}
		})
	}
}

func TestDecodePlanRejectsMoreDirtyEntriesThanTheLimit(t *testing.T) {
	plan := samplePlan()
	plan.Repositories[0].Dirty = make([]DirtyEntry, MaxDirtyEntries+1)
	for i := range plan.Repositories[0].Dirty {
		plan.Repositories[0].Dirty[i] = DirtyEntry{Path: fmt.Sprintf("f%04d.md", i)}
	}
	data, err := MarshalPlan(plan)
	if err == nil {
		_, err = DecodePlan(data)
	}
	if err == nil || !strings.Contains(err.Error(), "1000") {
		t.Fatalf("más de %d entradas sucias deben rechazarse nombrando el límite: %v", MaxDirtyEntries, err)
	}
}

func TestInputsDigest(t *testing.T) {
	deltas := []DeltaInput{{Domain: "b", Data: []byte("delta b")}, {Domain: "a", Data: []byte("delta a")}}
	declaration := []byte("baseline: x")
	base := inputsDigest(deltas, []byte("contexto"), declaration)
	if len(base) != 64 {
		t.Fatalf("la huella es un SHA-256 en hexadecimal: %q", base)
	}
	if again := inputsDigest(deltas, []byte("contexto"), declaration); again != base {
		t.Error("la huella es determinista")
	}
	reordered := []DeltaInput{deltas[1], deltas[0]}
	if inputsDigest(reordered, []byte("contexto"), declaration) != base {
		t.Error("el orden en que se reciben los deltas no cambia la huella")
	}
	differs := map[string]string{
		"un byte del delta":          inputsDigest([]DeltaInput{{Domain: "b", Data: []byte("delta B")}, deltas[1]}, []byte("contexto"), declaration),
		"el dominio del delta":       inputsDigest([]DeltaInput{{Domain: "c", Data: []byte("delta b")}, deltas[1]}, []byte("contexto"), declaration),
		"un delta menos":             inputsDigest(deltas[:1], []byte("contexto"), declaration),
		"el delta de contexto":       inputsDigest(deltas, []byte("contexto 2"), declaration),
		"sin delta de contexto":      inputsDigest(deltas, nil, declaration),
		"delta de contexto vacío":    inputsDigest(deltas, []byte{}, declaration),
		"la instantánea declarada":   inputsDigest(deltas, []byte("contexto"), []byte("baseline: y")),
		"el reparto entre segmentos": inputsDigest([]DeltaInput{{Domain: "b", Data: []byte("delta ")}, {Domain: "a", Data: []byte("bdelta a")}}, []byte("contexto"), declaration),
	}
	seen := map[string]string{base: "la base"}
	for name, digest := range differs {
		if digest == base {
			t.Errorf("cambiar %s debe cambiar la huella", name)
		}
		if other, dup := seen[digest]; dup {
			t.Errorf("%s y %s producen la misma huella", name, other)
		}
		seen[digest] = name
	}
}

func TestDeclarationSnapshotFollowsTheDeclaration(t *testing.T) {
	baseline := &workspace.BaselineConfig{Documents: []workspace.BaselineDocument{{Domain: "requisitos", Path: "01.md", Requirement: &workspace.RequirementProfile{Heading: "^### (?P<id>RF-\\d+): (?P<title>.+)$", Level: 3, Render: "### {id}: {title}"}}}}
	context := &workspace.TechnicalContextConfig{Maintained: &workspace.ContextMode{Paths: []string{"context/"}}}
	one := declarationSnapshot(workspace.Declarations{Baseline: baseline, TechnicalContext: context})
	if string(one) != string(declarationSnapshot(workspace.Declarations{Baseline: baseline, TechnicalContext: context})) {
		t.Fatal("la instantánea es determinista")
	}
	changed := *baseline
	changed.Documents = []workspace.BaselineDocument{{Domain: "requisitos", Path: "02.md"}}
	if string(one) == string(declarationSnapshot(workspace.Declarations{Baseline: &changed, TechnicalContext: context})) {
		t.Error("cambiar un documento declarado debe cambiar la instantánea")
	}
	if string(one) == string(declarationSnapshot(workspace.Declarations{Baseline: baseline})) {
		t.Error("retirar el contexto técnico debe cambiar la instantánea")
	}
}

func TestWorkingTreeFingerprint(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.md")
	if err := os.WriteFile(file, []byte("contenido"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := workingTreeFingerprint(file)
	if err != nil || got != sha256Hex([]byte("contenido")) {
		t.Fatalf("un fichero regular se identifica por el SHA-256 de su contenido: %q %v", got, err)
	}
	if got, err := workingTreeFingerprint(filepath.Join(dir, "no-existe.md")); err != nil || got != "" {
		t.Fatalf("un fichero borrado tiene huella vacía: %q %v", got, err)
	}
	if got, err := workingTreeFingerprint(dir); err != nil || got == "" {
		t.Fatalf("un directorio (un repositorio anidado sin seguimiento) tiene huella: %q %v", got, err)
	}
}

// Reentrada y reaplicación (D-13, R35).
func TestReconcilePlan(t *testing.T) {
	existing := samplePlan()
	fresh := samplePlan()
	fresh.PreparedAt = existing.PreparedAt.Add(time.Hour)
	fresh.Repositories[0].Head = strings.Repeat("ab", 20) // un commit posterior no invalida el plan

	t.Run("las mismas entradas reemiten el mismo plan", func(t *testing.T) {
		got, err := reconcilePlan(existing, fresh)
		if err != nil {
			t.Fatalf("reconcilePlan: %v", err)
		}
		if !reflect.DeepEqual(got, existing) {
			t.Error("debe devolver el plan existente, con su prepared_at y su HEAD registrados")
		}
	})

	t.Run("entradas cambiadas", func(t *testing.T) {
		changed := fresh
		changed.InputsSHA256 = sha256Hex([]byte("otras entradas"))
		_, err := reconcilePlan(existing, changed)
		var stale *PlanChangedError
		if !errors.As(err, &stale) {
			t.Fatalf("error = %v, quería *PlanChangedError", err)
		}
		for _, want := range []string{"git checkout -- ", "01_Requisitos_Funcionales.md", PlanFileName, "volver a preparar", "axiom sdd integrate prepare"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("el rechazo debe contener %q: %v", want, err)
			}
		}
	})

	t.Run("un documento cambió desde prepare", func(t *testing.T) {
		changed := samplePlan()
		changed.Documents[0].BeforeSHA256 = sha256Hex([]byte("editado a mano"))
		_, err := reconcilePlan(existing, changed)
		var stale *PlanChangedError
		if !errors.As(err, &stale) || len(stale.Documents) != 1 || stale.Documents[0] != "01_Requisitos_Funcionales.md" {
			t.Fatalf("error = %v, quería *PlanChangedError nombrando el documento", err)
		}
	})

	t.Run("todas las composiciones ya integradas", func(t *testing.T) {
		integrated := samplePlan()
		integrated.Documents[0].State = StateIntegrated
		other := fresh
		other.InputsSHA256 = sha256Hex([]byte("otras entradas")) // «ya integrado» tiene precedencia
		_, err := reconcilePlan(integrated, other)
		var done *AlreadyIntegratedError
		if !errors.As(err, &done) {
			t.Fatalf("error = %v, quería *AlreadyIntegratedError", err)
		}
		for _, want := range []string{"ya integrado", "inc-99-ejemplo", "RF-9001 -> RF-008", "axiom sdd integrate verify", "axiom sdd integrate show"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("el rechazo debe contener %q: %v", want, err)
			}
		}
	})

	t.Run("sin composiciones y con las mismas entradas se reemite", func(t *testing.T) {
		directed := samplePlan()
		directed.Documents = directed.Documents[1:]
		got, err := reconcilePlan(directed, directed)
		if err != nil || !reflect.DeepEqual(got, directed) {
			t.Fatalf("un plan sin composiciones no está «ya integrado»: %v", err)
		}
	})
}

func plainRepo(t *testing.T, master, name string) string {
	t.Helper()
	dir := filepath.Join(master, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func repoByName(t *testing.T, repos []PlanRepository, name string) PlanRepository {
	t.Helper()
	for _, repo := range repos {
		if repo.Repo == name {
			return repo
		}
	}
	t.Fatalf("no hay repositorio %q en %v", name, repos)
	return PlanRepository{}
}

func requirePrecondition(t *testing.T, err error, fragments ...string) {
	t.Helper()
	var stop *PreconditionError
	if !errors.As(err, &stop) {
		t.Fatalf("error = %v, quería *PreconditionError", err)
	}
	for _, fragment := range fragments {
		if !strings.Contains(err.Error(), fragment) {
			t.Errorf("la parada debe contener %q: %v", fragment, err)
		}
	}
}

func TestCaptureRepositoriesRecordsHeadAndAnEmptyPhotoOfACleanTree(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	master, specs, backend := siblings(t)
	affected := []affectedPath{
		{Path: filepath.Join(specs, "01_Requisitos.md")},
		{Path: filepath.Join(specs, "..", "repo-backend", "docs", "API.md")},
		{Path: filepath.Join(backend, "docs", "API.md")}, // el mismo repositorio: una sola entrada
	}
	got, err := captureRepositories(ctx, master, affected, []string{specs})
	if err != nil {
		t.Fatalf("captureRepositories: %v", err)
	}
	if len(got.Repositories) != 2 || got.Repositories[0].Repo != "repo-backend" || got.Repositories[1].Repo != "specs" {
		t.Fatalf("repositorios = %+v, quería repo-backend y specs ordenados y sin duplicados", got.Repositories)
	}
	for name, dir := range map[string]string{"specs": specs, "repo-backend": backend} {
		repo := repoByName(t, got.Repositories, name)
		if repo.Head != runGit(t, dir, "rev-parse", "HEAD") || len(repo.Dirty) != 0 {
			t.Errorf("%s = %+v", name, repo)
		}
	}
}

func TestCaptureRepositoriesStopsOnAnyStateOfAnAffectedDocument(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		name  string
		setup func(t *testing.T, specs string)
		want  []string
	}{
		{name: "cambio en el índice", setup: func(t *testing.T, specs string) {
			writeRepoFile(t, specs, "01_Requisitos.md", "# Requisitos\n\nen el índice\n")
			runGit(t, specs, "add", "01_Requisitos.md")
		}, want: []string{"01_Requisitos.md", "índice"}},
		{name: "cambio sin stage", setup: func(t *testing.T, specs string) {
			writeRepoFile(t, specs, "01_Requisitos.md", "# Requisitos\n\nsin stage\n")
		}, want: []string{"01_Requisitos.md", "árbol de trabajo"}},
		{name: "cambio en el índice y en el árbol", setup: func(t *testing.T, specs string) {
			writeRepoFile(t, specs, "01_Requisitos.md", "uno\n")
			runGit(t, specs, "add", "01_Requisitos.md")
			writeRepoFile(t, specs, "01_Requisitos.md", "dos\n")
		}, want: []string{"índice y en el árbol de trabajo"}},
		{name: "documento borrado", setup: func(t *testing.T, specs string) {
			if err := os.Remove(filepath.Join(specs, "01_Requisitos.md")); err != nil {
				t.Fatal(err)
			}
		}, want: []string{"01_Requisitos.md"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			master, specs, _ := siblings(t)
			tt.setup(t, specs)
			_, err := captureRepositories(ctx, master, []affectedPath{{Path: filepath.Join(specs, "01_Requisitos.md")}}, nil)
			requirePrecondition(t, err, append(tt.want, "confirma", "git stash -u", "descarta", "axiom sdd integrate prepare")...)
		})
	}

	t.Run("documento afectado sin seguimiento", func(t *testing.T) {
		t.Parallel()
		master, _, backend := siblings(t)
		writeRepoFile(t, backend, "docs/NUEVO.md", "# Nuevo\n")
		_, err := captureRepositories(ctx, master, []affectedPath{{Path: filepath.Join(backend, "docs", "NUEVO.md")}}, nil)
		requirePrecondition(t, err, "docs/NUEVO.md", "sin seguimiento", "repo-backend")
	})
}

func TestCaptureRepositoriesExemptsADocumentThatWillBeCreated(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	master, _, backend := siblings(t)
	creates := filepath.Join(backend, "docs", "PAGOS.md")

	got, err := captureRepositories(ctx, master, []affectedPath{{Path: creates, Creates: true}}, nil)
	if err != nil {
		t.Fatalf("un documento que aún no existe no detiene la integración: %v", err)
	}
	if repo := repoByName(t, got.Repositories, "repo-backend"); len(repo.Dirty) != 0 {
		t.Errorf("foto = %+v", repo.Dirty)
	}

	writeRepoFile(t, backend, "docs/PAGOS.md", "# Pagos\n") // una ejecución anterior lo dejó sin confirmar
	if _, err := captureRepositories(ctx, master, []affectedPath{{Path: creates}}, nil); err == nil {
		t.Fatal("sin la exención, un documento sin seguimiento detiene la integración")
	}
	got, err = captureRepositories(ctx, master, []affectedPath{{Path: creates, Creates: true}}, nil)
	if err != nil {
		t.Fatalf("el documento que archive crearía queda excluido de la precondición (REQ-24.49): %v", err)
	}
	if repo := repoByName(t, got.Repositories, "repo-backend"); len(repo.Dirty) != 1 || repo.Dirty[0].Path != "docs/PAGOS.md" {
		t.Errorf("queda en la foto previa: %+v", repo.Dirty)
	}
}

func TestCaptureRepositoriesKeepsForeignChangesInThePhoto(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	master, specs, _ := siblings(t)
	writeRepoFile(t, specs, "viejo.md", "viejo\n")
	commitAll(t, specs, "viejo")
	if err := os.Remove(filepath.Join(specs, "viejo.md")); err != nil { // borrado
		t.Fatal(err)
	}
	writeRepoFile(t, specs, "ajeno.md", "ajeno\n")
	runGit(t, specs, "add", "ajeno.md")                        // en el índice
	writeRepoFile(t, specs, "notas/borrador.md", "borrador\n") // sin seguimiento

	got, err := captureRepositories(ctx, master, []affectedPath{{Path: filepath.Join(specs, "01_Requisitos.md")}}, nil)
	if err != nil {
		t.Fatalf("un cambio ajeno, también en el índice, no detiene: %v", err)
	}
	dirty := map[string]string{}
	for _, entry := range repoByName(t, got.Repositories, "specs").Dirty {
		dirty[entry.Path] = entry.SHA256
	}
	want := map[string]string{
		"ajeno.md":          sha256Hex([]byte("ajeno\n")),
		"notas/borrador.md": sha256Hex([]byte("borrador\n")),
		"viejo.md":          "",
	}
	if !reflect.DeepEqual(dirty, want) {
		t.Fatalf("foto previa = %v, quería %v (con huella del árbol de trabajo, vacía si se borró)", dirty, want)
	}
}

func TestCaptureRepositoriesStopsAboveTheDirtyLimit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	master, specs, _ := siblings(t)
	for i := 0; i <= MaxDirtyEntries; i++ {
		writeRepoFile(t, specs, fmt.Sprintf("ruido/f%04d.md", i), "x\n")
	}
	_, err := captureRepositories(ctx, master, []affectedPath{{Path: filepath.Join(specs, "01_Requisitos.md")}}, nil)
	requirePrecondition(t, err, "1001", "1000", "git stash -u", "specs", "axiom sdd integrate prepare")

	// Con exactamente el límite no se detiene.
	if err := os.Remove(filepath.Join(specs, "ruido", "f0000.md")); err != nil {
		t.Fatal(err)
	}
	if got, err := captureRepositories(ctx, master, []affectedPath{{Path: filepath.Join(specs, "01_Requisitos.md")}}, nil); err != nil || len(repoByName(t, got.Repositories, "specs").Dirty) != MaxDirtyEntries {
		t.Fatalf("con %d entradas sucias debe capturarse: %v", MaxDirtyEntries, err)
	}
}

func TestCaptureRepositoriesOfARepositoryWithoutCommits(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	isolateGit(t)
	master := t.TempDir()
	repo := newRepo(t, filepath.Join(master, "nuevo"))
	got, err := captureRepositories(ctx, master, []affectedPath{{Path: filepath.Join(repo, "docs", "A.md"), Creates: true}}, nil)
	if err != nil {
		t.Fatalf("captureRepositories: %v", err)
	}
	if captured := repoByName(t, got.Repositories, "nuevo"); captured.Head != "" || len(captured.Dirty) != 0 {
		t.Fatalf("un repositorio sin commits registra HEAD vacío (se compara con el árbol vacío): %+v", captured)
	}
}

func TestCaptureRepositoriesStopsWhenAnAffectedDocumentIsNotInAGitRepository(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	master, specs, _ := siblings(t)
	plain := plainRepo(t, master, "sin-git")
	if err := os.WriteFile(filepath.Join(plain, "doc.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := captureRepositories(ctx, master, []affectedPath{{Path: filepath.Join(specs, "01_Requisitos.md")}, {Path: filepath.Join(plain, "doc.md")}}, nil)
	requirePrecondition(t, err, "sin-git", "repositorio git", "git init")
}

func TestCaptureRepositoriesWatchesTheOtherRepositoriesWithoutStoppingOnThem(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	master, specs, backend := siblings(t)
	frontend := newRepo(t, filepath.Join(master, "repo-frontend"))
	writeRepoFile(t, frontend, "App.md", "# App\n")
	commitAll(t, frontend, "frontend")
	writeRepoFile(t, frontend, "App.md", "# App\n\nsucio\n")
	plain := plainRepo(t, master, "repo-sin-git")

	got, err := captureRepositories(ctx, master, []affectedPath{{Path: filepath.Join(specs, "01_Requisitos.md")}}, []string{specs, backend, frontend, plain, filepath.Join(master, "no-existe")})
	if err != nil {
		t.Fatalf("los repositorios que solo se vigilan nunca detienen: %v", err)
	}
	if len(got.Repositories) != 3 {
		t.Fatalf("repositorios = %+v, quería specs, repo-backend y repo-frontend", got.Repositories)
	}
	if front := repoByName(t, got.Repositories, "repo-frontend"); len(front.Dirty) != 1 || front.Dirty[0].Path != "App.md" {
		t.Errorf("el repositorio vigilado registra su suciedad previa: %+v", front)
	}
	joined := strings.Join(got.Warnings, "\n")
	for _, want := range []string{"repo-sin-git", "no-existe"} {
		if !strings.Contains(joined, want) {
			t.Errorf("un repositorio vigilado que no es git, o no existe, se avisa nombrándolo (%q): %v", want, got.Warnings)
		}
	}
}

func TestCaptureRepositoriesNamesTheMasterRepositoryAsDot(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	isolateGit(t)
	master := newRepo(t, t.TempDir())
	writeRepoFile(t, master, "specs/doc.md", "# doc\n")
	commitAll(t, master, "doc")
	got, err := captureRepositories(ctx, master, []affectedPath{{Path: filepath.Join(master, "specs", "doc.md")}}, nil)
	if err != nil || len(got.Repositories) != 1 || got.Repositories[0].Repo != "." {
		t.Fatalf("repositorios = %+v, %v; quería el repositorio «.»", got.Repositories, err)
	}
}

func TestCaptureRepositoriesRejectsASymlinkThatLeavesTheMasterFolder(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	master, specs, _ := siblings(t)
	outside := t.TempDir()
	newRepo(t, filepath.Join(outside, "externo"))
	link := filepath.Join(master, "enlace")
	if err := os.Symlink(filepath.Join(outside, "externo"), link); err != nil {
		t.Skipf("no se pueden crear enlaces simbólicos en este sistema: %v", err)
	}
	_, err := captureRepositories(ctx, master, []affectedPath{{Path: filepath.Join(specs, "01_Requisitos.md")}, {Path: filepath.Join(link, "doc.md"), Creates: true}}, nil)
	if err == nil || !strings.Contains(err.Error(), "carpeta maestra") || !strings.Contains(err.Error(), "enlace") {
		t.Fatalf("un enlace simbólico que sale de la carpeta maestra es un error de validación: %v", err)
	}
	if errors.As(err, new(*PreconditionError)) {
		t.Error("no es una precondición de git: es un error de validación de la ruta")
	}
}

// Estado del push: aviso con el número y la procedencia del dato, nota sin
// upstream o con HEAD desacoplado, y nunca la red.
func TestCaptureRepositoriesWarnsOfABranchBehindItsUpstream(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	isolateGit(t)
	master := t.TempDir()
	origin := newRepo(t, filepath.Join(master, "origen"))
	writeRepoFile(t, origin, "doc.md", "# doc\n")
	commitAll(t, origin, "uno")
	clone := filepath.Join(master, "specs")
	runGit(t, master, "clone", "-q", origin, clone)
	runGit(t, clone, "config", "core.autocrlf", "false")
	writeRepoFile(t, origin, "otro.md", "otro\n")
	commitAll(t, origin, "dos")
	runGit(t, clone, "fetch", "-q", "origin")

	affected := []affectedPath{{Path: filepath.Join(clone, "doc.md")}}
	got, err := captureRepositories(ctx, master, affected, nil)
	if err != nil {
		t.Fatalf("estar por detrás no detiene: %v", err)
	}
	joined := strings.Join(got.Warnings, "\n")
	for _, want := range []string{"specs", "1 commit", "por detrás", "referencias locales", "fetch"} {
		if !strings.Contains(joined, want) {
			t.Errorf("el aviso debe contener %q: %v", want, got.Warnings)
		}
	}

	runGit(t, clone, "checkout", "-q", "--detach")
	got, err = captureRepositories(ctx, master, affected, nil)
	if err != nil || !strings.Contains(strings.Join(got.Warnings, "\n"), "desacoplado") {
		t.Fatalf("HEAD desacoplado = nota, sin detener: %v %v", got.Warnings, err)
	}
}
