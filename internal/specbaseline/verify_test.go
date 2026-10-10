package specbaseline

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Las pruebas del verificador nativo E2 (inc-24, REQ-24.30, REQ-24.51,
// REQ-24.53, REQ-24.57, REQ-24.59 (e), D-15 y la matriz de amenazas del diseño §8).
// Todo se prueba sobre repositorios git reales y temporales, con el entorno de git
// aislado y sin red. Todos los textos son inventados.

const (
	docModelo   = "03_Modelo_Datos.md"
	docFlujos   = "04_Flujos_Negocio.md"
	docContexto = "context/ARQUITECTURA.md"
	docAPI      = "docs/API.md"

	// sectionModelo es la sección del cambio que la edición dirigida simulada añade
	// al documento de prosa: su título empieza por el nombre del cambio.
	sectionModelo = "\n## inc-99-ejemplo: Entidad Reintento\n\nCada reintento se asocia a un pedido (ver RF-008).\n"
	sectionAPI    = "\n## Errores\n\nEl cobro reintentado devuelve el error 402.\n"
)

func appendText(t *testing.T, path, text string) {
	t.Helper()
	handle, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	if _, err := handle.WriteString(text); err != nil {
		t.Fatal(err)
	}
}

func replaceText(t *testing.T, path, old, replacement string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(old)) {
		t.Fatalf("%s no contiene %q", path, old)
	}
	if err := os.WriteFile(path, bytes.Replace(data, []byte(old), []byte(replacement), 1), 0o644); err != nil {
		t.Fatal(err)
	}
}

// applyDirectedEdits simula la edición dirigida del agente: la sección del cambio
// en el documento de prosa y las dos entradas del delta de contexto.
func (f *integrationFixture) applyDirectedEdits(t *testing.T) {
	t.Helper()
	appendText(t, filepath.Join(f.specs, docModelo), sectionModelo)
	replaceText(t, filepath.Join(f.specs, filepath.FromSlash(docContexto)), "Gestiona los pedidos.", "Gestiona los pedidos y reintenta el cobro (RF-008).")
	appendText(t, filepath.Join(f.backend, filepath.FromSlash(docAPI)), sectionAPI)
}

// composeAll compone los documentos de composición del plan, como el paso 2 de D-13.
func composeAll(t *testing.T, target ChangeTarget, domains ...string) {
	t.Helper()
	if len(domains) == 0 {
		domains = []string{"requisitos-funcionales"}
	}
	report, err := ApplyCompose(context.Background(), target, domains)
	if err != nil {
		t.Fatalf("ApplyCompose: %v", err)
	}
	mustNoErrors(t, report)
}

// integratedFixture lleva el cambio hasta el punto de `verify`: prepare, la
// composición y la edición dirigida simulada, todo sin confirmar.
func integratedFixture(t *testing.T) (*integrationFixture, ChangeTarget, Plan) {
	t.Helper()
	f, target, plan := preparedFixture(t)
	composeAll(t, target)
	f.applyDirectedEdits(t)
	return f, target, plan
}

func verifyOf(t *testing.T, target ChangeTarget) VerifyReport {
	t.Helper()
	report, err := Verify(context.Background(), target)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.Violations == nil {
		t.Fatal("Violations nunca es nil: el JSON lista siempre las violaciones")
	}
	return report
}

func describeViolations(report VerifyReport) string {
	var lines []string
	for _, v := range report.Violations {
		lines = append(lines, "["+v.Check+"] entrada="+v.Entry+" fichero="+v.File+": "+v.Detail)
	}
	return strings.Join(lines, "\n")
}

func mustPass(t *testing.T, report VerifyReport) {
	t.Helper()
	if !report.Passed || len(report.Violations) != 0 {
		t.Fatalf("la verificación debía superarse:\n%s", describeViolations(report))
	}
}

// mustViolate exige una violación de la comprobación indicada cuya entrada,
// fichero y detalle juntos contengan todos los fragmentos.
func mustViolate(t *testing.T, report VerifyReport, check string, fragments ...string) Violation {
	t.Helper()
	if report.Passed {
		t.Fatalf("la verificación no debía superarse (%s %q)", check, fragments)
	}
	for _, v := range report.Violations {
		if v.Check != check {
			continue
		}
		whole := v.Entry + "\n" + v.File + "\n" + v.Detail
		matches := true
		for _, fragment := range fragments {
			if !strings.Contains(whole, fragment) {
				matches = false
				break
			}
		}
		if matches {
			return v
		}
	}
	t.Fatalf("falta una violación %q con %q; hay:\n%s", check, fragments, describeViolations(report))
	return Violation{}
}

func mustNotViolate(t *testing.T, report VerifyReport, check string) {
	t.Helper()
	for _, v := range report.Violations {
		if v.Check == check {
			t.Errorf("no debía haber violaciones %q:\n%s", check, describeViolations(report))
			return
		}
	}
}

func loadPlan(t *testing.T, f *integrationFixture) Plan {
	t.Helper()
	plan, err := ReadPlan(filepath.Join(f.changeRoot, PlanFileName))
	if err != nil {
		t.Fatalf("ReadPlan: %v", err)
	}
	return plan
}

// Éxito: marca integradas las entradas dirigidas y de contexto, las de solo
// registro siguen pendientes, escribe verified_at y solo toca integration.yaml.
func TestVerifyPassesAndMarksTheDirectedAndContextEntriesIntegrated(t *testing.T) {
	t.Parallel()
	f, target, prepared := integratedFixture(t)
	before := snapshot(t, f.master)

	report := verifyOf(t, target)
	mustPass(t, report)
	if VerifySchema != "axiom.sdd-integration-verify/v1" || report.Schema != VerifySchema || report.Change != "inc-99-ejemplo" {
		t.Errorf("informe = %+v", report)
	}

	changed := changedFiles(before, snapshot(t, f.master))
	if want := []string{"repo-specs/openspec/changes/inc-99-ejemplo/" + PlanFileName}; !reflect.DeepEqual(changed, want) {
		t.Fatalf("verify solo escribe el plan: %v", changed)
	}

	plan := loadPlan(t, f)
	if plan.VerifiedAt == nil || plan.VerifiedAt.IsZero() {
		t.Error("verify escribe verified_at")
	}
	if plan.InputsSHA256 != prepared.InputsSHA256 || !reflect.DeepEqual(plan.Repositories, prepared.Repositories) || !reflect.DeepEqual(plan.IDMap, prepared.IDMap) || !plan.PreparedAt.Equal(prepared.PreparedAt) {
		t.Errorf("verify no cambia la línea base del plan: antes %+v, después %+v", prepared, plan)
	}
	kinds := map[EntryKind]int{}
	for _, entry := range plan.Entries {
		kinds[entry.Kind]++
		wantState := StateIntegrated
		if entry.Kind == EntryRegister {
			wantState = StatePending
		}
		if entry.State != wantState {
			t.Errorf("entrada %s (%s) en estado %q, se esperaba %q", entry.ID, entry.Kind, entry.State, wantState)
		}
		if entry.Action == ActionCreated {
			t.Errorf("ningún documento se creó: %+v", entry)
		}
	}
	if kinds[EntryCompose] == 0 || kinds[EntryDirected] == 0 || kinds[EntryContext] != 2 || kinds[EntryRegister] != 1 {
		t.Errorf("entradas por tipo = %v", kinds)
	}
	for _, document := range plan.Documents {
		if document.Integration == workspace.IntegrationCompose && document.State != StateIntegrated {
			t.Errorf("el documento de composición %s debe seguir integrado: %+v", document.Domain, document)
		}
	}
}

// Verificar dos veces es idempotente: el plan ya marcado vuelve a verificarse.
func TestVerifyCanBeRepeatedOverAnAlreadyVerifiedPlan(t *testing.T) {
	t.Parallel()
	f, target, _ := integratedFixture(t)
	mustPass(t, verifyOf(t, target))
	first := loadPlan(t, f)
	mustPass(t, verifyOf(t, target))
	second := loadPlan(t, f)
	if !reflect.DeepEqual(first.Entries, second.Entries) || !reflect.DeepEqual(first.Documents, second.Documents) {
		t.Errorf("la segunda verificación no cambia entradas ni documentos:\n%+v\n%+v", first.Entries, second.Entries)
	}
}

// REQ-24.55: una entrada `add` que crea el documento queda como `created`.
func TestVerifyRecordsCreatedForTheAddEntryThatCreatedTheDocument(t *testing.T) {
	t.Parallel()
	f := newIntegrationFixture(t, true)
	f.writeChange(t, ContextDeltaFileName, "## Entrada: context/PAGOS.md\n\n- Ancla: Reintentos\n- Acción: add\n- Motivo: documenta los pagos (RF-9001)\n\n"+
		"## Entrada: ../repo-backend/docs/API.md\n\n- Ancla: Errores\n- Acción: add\n- Motivo: documenta el error del cobro\n")
	commitAll(t, f.specs, "contexto nuevo")
	target := f.target(t)
	if _, report, err := Prepare(context.Background(), target, PlanOptions{}); err != nil {
		t.Fatal(err)
	} else {
		mustNoErrors(t, report)
	}
	composeAll(t, target)
	appendText(t, filepath.Join(f.specs, docModelo), sectionModelo)
	appendText(t, filepath.Join(f.backend, filepath.FromSlash(docAPI)), sectionAPI)
	writeRepoFile(t, f.specs, "context/PAGOS.md", "# Pagos\n\n## Reintentos\n\nEl cobro se reintenta (RF-008).\n")

	mustPass(t, verifyOf(t, target))
	var created, added int
	for _, entry := range loadPlan(t, f).Entries {
		if entry.Kind != EntryContext {
			continue
		}
		switch {
		case entry.Document == "context/PAGOS.md" && entry.Action == ActionCreated:
			created++
		case entry.Document == "../repo-backend/docs/API.md" && entry.Action == ActionAdded:
			added++
		default:
			t.Errorf("entrada inesperada %+v", entry)
		}
	}
	if created != 1 || added != 1 {
		t.Errorf("created=%d added=%d", created, added)
	}
	// Repetir la verificación reconoce la acción `created` como la entrada `add` del delta.
	mustPass(t, verifyOf(t, target))
}

// Si falla, lista todas las violaciones y no modifica documentos ni plan.
func TestVerifyFailureListsEveryViolationAndModifiesNeitherDocumentsNorPlan(t *testing.T) {
	t.Parallel()
	f, target, _ := integratedFixture(t)
	writeRepoFile(t, f.specs, "referencia/legado/README.sh", "#!/bin/sh\n")
	writeRepoFile(t, f.specs, "03_Modelo_Datos.mdx", "# Otro\n")
	appendText(t, filepath.Join(f.specs, docFlujos), "\nTexto añadido.\n")
	replaceText(t, filepath.Join(f.specs, docModelo), "## inc-99-ejemplo: Entidad Reintento", "## Entidad Reintento")
	before := snapshot(t, f.master)
	gitBefore := map[string]string{f.specs: gitIndexDigest(t, f.specs), f.backend: gitIndexDigest(t, f.backend)}

	report := verifyOf(t, target)
	mustViolate(t, report, ViolationScope, "repo-specs/referencia/legado/README.sh")
	mustViolate(t, report, ViolationScope, "repo-specs/03_Modelo_Datos.mdx")
	mustViolate(t, report, ViolationScope, "repo-specs/"+docFlujos, "register")
	mustViolate(t, report, ViolationAnchor, "Entidad Reintento", docModelo)
	if len(report.Violations) < 4 {
		t.Errorf("deben listarse todas las violaciones:\n%s", describeViolations(report))
	}
	if changed := changedFiles(before, snapshot(t, f.master)); len(changed) != 0 {
		t.Errorf("tras un fallo no cambia ningún documento ni el plan: %v", changed)
	}
	plan := loadPlan(t, f)
	if plan.VerifiedAt != nil {
		t.Error("un fallo no escribe verified_at")
	}
	for _, entry := range plan.Entries {
		if entry.Kind != EntryCompose && entry.State != StatePending {
			t.Errorf("un fallo no marca la entrada %+v", entry)
		}
	}
	for dir, digest := range gitBefore {
		if got := gitIndexDigest(t, dir); got != digest {
			t.Errorf("verify no toca el índice de git de %s", dir)
		}
	}
}

// gitIndexDigest es la huella del índice: el verificador solo lee git.
func gitIndexDigest(t *testing.T, repo string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repo, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// (c) Alcance por identidad (REQ-24.51, REQ-24.30, REQ-24.57, design §8).
func TestVerifyScopeRejectsEveryPathOutsideTheAllowedSet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		mutate    func(t *testing.T, f *integrationFixture)
		file      string
		fragments []string
	}{
		{
			name: "un README.sh bajo una ruta de referencia",
			mutate: func(t *testing.T, f *integrationFixture) {
				writeRepoFile(t, f.specs, "referencia/legado/README.sh", "#!/bin/sh\n")
			},
			file:      "repo-specs/referencia/legado/README.sh",
			fragments: []string{"referencia", "modo reference"},
		},
		{
			name: "un README.sh en una ruta no declarada",
			mutate: func(t *testing.T, f *integrationFixture) {
				writeRepoFile(t, f.specs, "notas/README.sh", "#!/bin/sh\n")
			},
			file:      "repo-specs/notas/README.sh",
			fragments: []string{"conjunto permitido"},
		},
		{
			name: "un fichero .mdx junto a un documento declarado",
			mutate: func(t *testing.T, f *integrationFixture) {
				writeRepoFile(t, f.specs, "03_Modelo_Datos.mdx", "# Modelo\n")
			},
			file:      "repo-specs/03_Modelo_Datos.mdx",
			fragments: []string{"conjunto permitido"},
		},
		{
			name: "un temporal huérfano de una escritura interrumpida",
			mutate: func(t *testing.T, f *integrationFixture) {
				writeRepoFile(t, f.specs, ".03_Modelo_Datos.md.tmp-123456", "# a medias\n")
			},
			file:      "repo-specs/.03_Modelo_Datos.md.tmp-123456",
			fragments: []string{"conjunto permitido"},
		},
		{
			name: "un documento de contexto que el delta no nombra",
			mutate: func(t *testing.T, f *integrationFixture) {
				writeRepoFile(t, f.specs, "context/OTRO.md", "# Otro\n")
			},
			file:      "repo-specs/context/OTRO.md",
			fragments: []string{"conjunto permitido"},
		},
		{
			name: "un documento mantenido de un repositorio hermano que el delta no nombra",
			mutate: func(t *testing.T, f *integrationFixture) {
				writeRepoFile(t, f.backend, "docs/OTRO.md", "# Otro\n")
			},
			file:      "repo-backend/docs/OTRO.md",
			fragments: []string{"conjunto permitido"},
		},
		{
			name: "la edición de un fichero de referencia",
			mutate: func(t *testing.T, f *integrationFixture) {
				appendText(t, filepath.Join(f.specs, "referencia", "legado", "VIEJO.md"), "\nretoque\n")
			},
			file:      "repo-specs/referencia/legado/VIEJO.md",
			fragments: []string{"referencia", "modo reference"},
		},
		{
			name: "un documento con integración register",
			mutate: func(t *testing.T, f *integrationFixture) {
				appendText(t, filepath.Join(f.specs, docFlujos), "\nretoque\n")
			},
			file:      "repo-specs/" + docFlujos,
			fragments: []string{"register", "flujos-de-negocio"},
		},
		{
			name: "un fichero borrado fuera de la carpeta del cambio",
			mutate: func(t *testing.T, f *integrationFixture) {
				if err := os.Remove(filepath.Join(f.specs, "referencia", "legado", "VIEJO.md")); err != nil {
					t.Fatal(err)
				}
			},
			file:      "repo-specs/referencia/legado/VIEJO.md",
			fragments: []string{"borrado", "nunca borra"},
		},
		{
			name: "un documento permitido borrado",
			mutate: func(t *testing.T, f *integrationFixture) {
				if err := os.Remove(filepath.Join(f.specs, docFlujos)); err != nil {
					t.Fatal(err)
				}
			},
			file:      "repo-specs/" + docFlujos,
			fragments: []string{"borrado"},
		},
		{
			name: "el índice de OpenSpec con la baseline del proyecto",
			mutate: func(t *testing.T, f *integrationFixture) {
				writeRepoFile(t, f.specs, "openspec/INDEX.md", "# Índice\n")
			},
			file:      "repo-specs/openspec/INDEX.md",
			fragments: []string{"conjunto permitido"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, target, _ := integratedFixture(t)
			tt.mutate(t, f)
			report := verifyOf(t, target)
			mustViolate(t, report, ViolationScope, append([]string{tt.file}, tt.fragments...)...)
		})
	}
}

// El control de las pruebas anteriores: lo permitido no genera violaciones de alcance.
func TestVerifyScopeAllowsTheChangeFolderAndTheNamedDocuments(t *testing.T) {
	t.Parallel()
	f, target, _ := integratedFixture(t)
	f.writeChange(t, "notas.md", "# Notas del cambio\n")
	f.writeChange(t, "specs/modelo-de-datos/borrador.md", "# Borrador\n")
	if err := os.Remove(filepath.Join(f.changeRoot, "specs", "flujos-de-negocio", "spec.md")); err != nil {
		t.Fatal(err)
	}
	// Los ficheros de la carpeta del cambio quedan fuera de la verificación de IDs
	// aunque nombren provisionales: se archivan sin alteración.
	f.writeChange(t, "notas.md", "# Notas del cambio\n\nRF-9001 pasa a RF-008.\n")
	report := verifyOf(t, target)
	mustNotViolate(t, report, ViolationScope)
	mustNotViolate(t, report, ViolationIDs)
}

// REQ-24.57: con «Sin impacto», cualquier edición de las rutas mantenidas contradice
// la decisión registrada.
func TestVerifyNoImpactDecisionRejectsAnEditOfTheMaintainedPaths(t *testing.T) {
	t.Parallel()
	f := newIntegrationFixture(t, true)
	f.writeChange(t, ContextDeltaFileName, "Sin impacto: el cambio solo corrige una errata en un mensaje de error\n")
	commitAll(t, f.specs, "sin impacto")
	target := f.target(t)
	plan, report, err := Prepare(context.Background(), target, PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mustNoErrors(t, report)
	if plan.Context.Decision != ContextNoImpact {
		t.Fatalf("decisión = %+v", plan.Context)
	}
	composeAll(t, target)
	appendText(t, filepath.Join(f.specs, docModelo), sectionModelo)

	mustPass(t, verifyOf(t, target))
	if loadPlan(t, f).VerifiedAt == nil {
		t.Error("con «Sin impacto» y sin ediciones de contexto la verificación escribe verified_at")
	}

	replaceText(t, filepath.Join(f.specs, filepath.FromSlash(docContexto)), "Gestiona los pedidos.", "Gestiona los pedidos y más.")
	failed := verifyOf(t, target)
	mustViolate(t, failed, ViolationScope, "repo-specs/"+docContexto, "Sin impacto", "el cambio solo corrige una errata")
}

// El alcance con la baseline por defecto incluye las specs del cambio e INDEX.md.
func TestVerifyScopeWithTheDefaultBaselineIncludesTheChangeSpecsAndTheIndex(t *testing.T) {
	t.Parallel()
	const defaultConfig = `workspace:
  name: ejemplo
  topology: multirepo
  specs_repository: repo-specs
roles:
  backend:
    name: Backend
    repositories:
      - path: repo-backend
technical_context:
  maintained:
    paths:
      - context/
      - ../repo-backend/docs/
  reference:
    paths:
      - referencia/
`
	setup := func(t *testing.T) (*integrationFixture, ChangeTarget) {
		f := newIntegrationFixture(t, true)
		writeRepoFile(t, f.specs, "axiom.yaml", defaultConfig)
		commitAll(t, f.specs, "baseline por defecto")
		target := f.target(t)
		plan, report, err := Prepare(context.Background(), target, PlanOptions{})
		if err != nil {
			t.Fatal(err)
		}
		mustNoErrors(t, report)
		if len(plan.Documents) != 0 {
			t.Fatalf("sin baseline del proyecto no hay documentos de composición: %+v", plan.Documents)
		}
		// Los bloques fijados del Step 2 componen openspec/specs con el motor nativo
		// y `axiom archive sync` regenera INDEX.md.
		for _, domain := range []string{"requisitos-funcionales", "modelo-de-datos", "flujos-de-negocio"} {
			writeRepoFile(t, f.specs, "openspec/specs/"+domain+"/spec.md", "# "+domain+"\n")
		}
		writeRepoFile(t, f.specs, "openspec/INDEX.md", "# Índice\n")
		replaceText(t, filepath.Join(f.specs, filepath.FromSlash(docContexto)), "Gestiona los pedidos.", "Gestiona los pedidos y reintenta el cobro.")
		appendText(t, filepath.Join(f.backend, filepath.FromSlash(docAPI)), sectionAPI)
		return f, target
	}

	t.Run("las specs de los dominios del cambio y el índice pasan", func(t *testing.T) {
		t.Parallel()
		_, target := setup(t)
		mustPass(t, verifyOf(t, target))
	})
	t.Run("la spec de un dominio que el cambio no toca es una violación", func(t *testing.T) {
		t.Parallel()
		f, target := setup(t)
		writeRepoFile(t, f.specs, "openspec/specs/otro-dominio/spec.md", "# otro\n")
		mustViolate(t, verifyOf(t, target), ViolationScope, "repo-specs/openspec/specs/otro-dominio/spec.md", "conjunto permitido")
	})
	t.Run("un fichero distinto de spec.md bajo el dominio es una violación", func(t *testing.T) {
		t.Parallel()
		f, target := setup(t)
		writeRepoFile(t, f.specs, "openspec/specs/modelo-de-datos/README.sh", "#!/bin/sh\n")
		mustViolate(t, verifyOf(t, target), ViolationScope, "repo-specs/openspec/specs/modelo-de-datos/README.sh")
	})
}

// Un commit entre prepare y verify no oculta una edición fuera de alcance (H6).
func TestVerifyACommitBetweenPrepareAndVerifyDoesNotHideAnOutOfScopeEdit(t *testing.T) {
	t.Parallel()
	t.Run("lo confirmado dentro de alcance sigue verificándose y pasa", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		commitAll(t, f.specs, "edición dirigida")
		commitAll(t, f.backend, "edición del contexto")
		mustPass(t, verifyOf(t, target))
	})
	t.Run("lo confirmado fuera de alcance es una violación", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		writeRepoFile(t, f.specs, "notas/README.sh", "#!/bin/sh\n")
		commitAll(t, f.specs, "edición dirigida y un script")
		report := verifyOf(t, target)
		mustViolate(t, report, ViolationScope, "repo-specs/notas/README.sh")
	})
	t.Run("lo confirmado y borrado fuera de alcance también", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		if err := os.Remove(filepath.Join(f.specs, "referencia", "legado", "VIEJO.md")); err != nil {
			t.Fatal(err)
		}
		commitAll(t, f.specs, "borrado ajeno")
		mustViolate(t, verifyOf(t, target), ViolationScope, "referencia/legado/VIEJO.md", "borrado")
	})
}

// Un cambio en el índice de un fichero ajeno anterior a prepare no se atribuye al
// archivado; si el fichero cambia después, sí.
func TestVerifyDoesNotAttributeAForeignChangeThatPrepareAlreadySaw(t *testing.T) {
	t.Parallel()
	f := newIntegrationFixture(t, true)
	writeRepoFile(t, f.specs, "notas/borrador.md", "# Borrador ajeno\n")
	runGit(t, f.specs, "add", "notas/borrador.md")
	writeRepoFile(t, f.specs, "notas/sin-seguimiento.md", "# Ajeno\n")
	target := f.target(t)
	plan, report, err := Prepare(context.Background(), target, PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mustNoErrors(t, report)
	if len(repoByName(t, plan.Repositories, "repo-specs").Dirty) != 2 {
		t.Fatalf("la foto previa debe recoger los dos ficheros ajenos: %+v", plan.Repositories)
	}
	composeAll(t, target)
	f.applyDirectedEdits(t)

	mustPass(t, verifyOf(t, target))

	appendText(t, filepath.Join(f.specs, "notas", "borrador.md"), "\ncambiado después de prepare\n")
	report2 := verifyOf(t, target)
	mustViolate(t, report2, ViolationScope, "repo-specs/notas/borrador.md")
	for _, v := range report2.Violations {
		if strings.Contains(v.File, "sin-seguimiento") {
			t.Errorf("el fichero ajeno sin cambios no se atribuye: %+v", v)
		}
	}
}

// core.autocrlf=true: git normaliza los finales de línea y el verificador no
// atribuye al archivado un fichero que solo difiere en ellos (D-12).
func TestVerifyWorksWithAutocrlfAndCRLFDocuments(t *testing.T) {
	t.Parallel()
	f := newIntegrationFixture(t, true)
	runGit(t, f.specs, "config", "core.autocrlf", "true")
	runGit(t, f.backend, "config", "core.autocrlf", "true")
	toCRLF := func(repo, rel string) {
		path := filepath.Join(repo, filepath.FromSlash(rel))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
		if err := os.WriteFile(path, bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n")), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, rel := range []string{docRequisitos, docModelo, docFlujos, docContexto, "referencia/legado/VIEJO.md"} {
		toCRLF(f.specs, rel)
	}
	toCRLF(f.backend, docAPI)
	// Como tras un checkout con autocrlf: el árbol de trabajo en CRLF y el índice en LF,
	// con el índice refrescado (git add no confirma nada porque el contenido normalizado
	// es idéntico).
	runGit(t, f.specs, "add", "-A")
	runGit(t, f.backend, "add", "-A")
	for _, repo := range []string{f.specs, f.backend} {
		if status := runGit(t, repo, "status", "--porcelain"); status != "" {
			t.Skipf("este git no normaliza los finales de línea con core.autocrlf=true: %q", status)
		}
	}

	target := f.target(t)
	plan, report, err := Prepare(context.Background(), target, PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mustNoErrors(t, report)
	if dirty := repoByName(t, plan.Repositories, "repo-specs").Dirty; len(dirty) != 0 {
		t.Fatalf("con autocrlf el árbol limpio no tiene suciedad previa: %+v", dirty)
	}
	composeAll(t, target)
	appendText(t, filepath.Join(f.specs, docModelo), strings.ReplaceAll(sectionModelo, "\n", "\r\n"))
	replaceText(t, filepath.Join(f.specs, filepath.FromSlash(docContexto)), "Gestiona los pedidos.", "Gestiona los pedidos y reintenta el cobro (RF-008).")
	appendText(t, filepath.Join(f.backend, filepath.FromSlash(docAPI)), strings.ReplaceAll(sectionAPI, "\n", "\r\n"))

	report2 := verifyOf(t, target)
	mustPass(t, report2)

	// Un documento CRLF sin tocar no es un cambio, aunque su índice esté en LF; uno
	// ajeno retocado sí.
	appendText(t, filepath.Join(f.specs, docFlujos), "retoque\r\n")
	mustViolate(t, verifyOf(t, target), ViolationScope, "repo-specs/"+docFlujos, "register")
}

// El alcance se decide por identidad, no por texto: un directorio de contexto
// declarado como enlace simbólico se alcanza por su destino real.
func TestVerifyScopeIsDecidedByIdentityThroughASymlink(t *testing.T) {
	t.Parallel()
	f := newIntegrationFixture(t, true)
	realDir := filepath.Join(f.specs, "contexto-real")
	if err := os.Rename(filepath.Join(f.specs, "context"), realDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("contexto-real", filepath.Join(f.specs, "context")); err != nil {
		t.Skipf("no se pueden crear enlaces simbólicos: %v", err)
	}
	commitAll(t, f.specs, "context como enlace simbólico")
	target := f.target(t)
	if _, report, err := Prepare(context.Background(), target, PlanOptions{}); err != nil {
		t.Fatal(err)
	} else {
		mustNoErrors(t, report)
	}
	composeAll(t, target)
	appendText(t, filepath.Join(f.specs, docModelo), sectionModelo)
	appendText(t, filepath.Join(f.backend, filepath.FromSlash(docAPI)), sectionAPI)
	// git informa de la ruta real; el delta nombra la del enlace: son el mismo fichero.
	replaceText(t, filepath.Join(realDir, "ARQUITECTURA.md"), "Gestiona los pedidos.", "Gestiona los pedidos y reintenta el cobro (RF-008).")
	mustPass(t, verifyOf(t, target))

	writeRepoFile(t, realDir, "OTRO.md", "# Otro\n")
	mustViolate(t, verifyOf(t, target), ViolationScope, "repo-specs/contexto-real/OTRO.md", "conjunto permitido")
}

// Un repositorio sin commits se compara con el árbol vacío (design §8).
func TestVerifyARepositoryWithoutCommitsIsComparedWithTheEmptyTree(t *testing.T) {
	t.Parallel()
	const config = `workspace:
  name: ejemplo
  topology: multirepo
  specs_repository: repo-specs
roles:
  backend:
    name: Backend
    repositories:
      - path: repo-backend
      - path: repo-nuevo
baseline:
  documents:
    - domain: requisitos-funcionales
      path: 01_Requisitos_Funcionales.md
      requirement:
        heading: '^#+ (?P<id>RF-\d+): (?P<title>.+)$'
        level: 3
        render: '### {id}: {title}'
    - domain: modelo-de-datos
      path: 03_Modelo_Datos.md
    - domain: flujos-de-negocio
      path: 04_Flujos_Negocio.md
      integration: register
technical_context:
  maintained:
    paths:
      - context/
      - ../repo-backend/docs/
  reference:
    paths:
      - referencia/
`
	setup := func(t *testing.T) (*integrationFixture, ChangeTarget, string) {
		f := newIntegrationFixture(t, true)
		writeRepoFile(t, f.specs, "axiom.yaml", config)
		commitAll(t, f.specs, "con un repositorio nuevo")
		fresh := newRepo(t, filepath.Join(f.master, "repo-nuevo"))
		writeRepoFile(t, fresh, "LEEME-previo.md", "# Previo, anterior a prepare\n")
		target := f.target(t)
		plan, report, err := Prepare(context.Background(), target, PlanOptions{})
		if err != nil {
			t.Fatal(err)
		}
		mustNoErrors(t, report)
		recorded := repoByName(t, plan.Repositories, "repo-nuevo")
		if recorded.Head != "" || len(recorded.Dirty) != 1 {
			t.Fatalf("repositorio sin commits = %+v", recorded)
		}
		composeAll(t, target)
		f.applyDirectedEdits(t)
		return f, target, fresh
	}

	t.Run("lo que ya había antes de prepare no se atribuye", func(t *testing.T) {
		t.Parallel()
		_, target, _ := setup(t)
		mustPass(t, verifyOf(t, target))
	})
	t.Run("un fichero nuevo fuera de alcance es una violación", func(t *testing.T) {
		t.Parallel()
		_, target, fresh := setup(t)
		writeRepoFile(t, fresh, "README.sh", "#!/bin/sh\n")
		mustViolate(t, verifyOf(t, target), ViolationScope, "repo-nuevo/README.sh", "conjunto permitido")
	})
	t.Run("el fichero previo, retocado, sí se atribuye", func(t *testing.T) {
		t.Parallel()
		_, target, fresh := setup(t)
		appendText(t, filepath.Join(fresh, "LEEME-previo.md"), "\nretoque\n")
		mustViolate(t, verifyOf(t, target), ViolationScope, "repo-nuevo/LEEME-previo.md")
	})
}

// El HEAD que registró prepare debe seguir existiendo: sin él no hay comparación.
func TestVerifyNeedsTheRecordedHeadAndTheRecordedRepositories(t *testing.T) {
	t.Parallel()
	t.Run("un commit que ya no existe", func(t *testing.T) {
		t.Parallel()
		f, target, plan := integratedFixture(t)
		plan = loadPlan(t, f)
		for i := range plan.Repositories {
			if plan.Repositories[i].Repo == "repo-specs" {
				plan.Repositories[i].Head = strings.Repeat("a", 40)
			}
		}
		if err := WritePlan(filepath.Join(f.changeRoot, PlanFileName), plan); err != nil {
			t.Fatal(err)
		}
		mustViolate(t, verifyOf(t, target), ViolationScope, "repo-specs", strings.Repeat("a", 40), "axiom sdd integrate prepare")
	})
	t.Run("un repositorio que ya no es git", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		if err := os.RemoveAll(filepath.Join(f.backend, ".git")); err != nil {
			t.Fatal(err)
		}
		mustViolate(t, verifyOf(t, target), ViolationScope, "repo-backend", "repositorio git")
	})
}

// (a) Presencia (REQ-24.51 a, D-15).
func TestVerifyPresenceRequiresTheEditedDocuments(t *testing.T) {
	t.Parallel()
	t.Run("un documento dirigido sin editar", func(t *testing.T) {
		t.Parallel()
		f, target, _ := preparedFixture(t)
		composeAll(t, target)
		replaceText(t, filepath.Join(f.specs, filepath.FromSlash(docContexto)), "Gestiona los pedidos.", "Gestiona los pedidos y reintenta (RF-008).")
		appendText(t, filepath.Join(f.backend, filepath.FromSlash(docAPI)), sectionAPI)
		report := verifyOf(t, target)
		mustViolate(t, report, ViolationPresence, "Entidad Reintento", docModelo)
	})
	t.Run("un documento de contexto modificado sin editar", func(t *testing.T) {
		t.Parallel()
		f, target, _ := preparedFixture(t)
		composeAll(t, target)
		appendText(t, filepath.Join(f.specs, docModelo), sectionModelo)
		appendText(t, filepath.Join(f.backend, filepath.FromSlash(docAPI)), sectionAPI)
		mustViolate(t, verifyOf(t, target), ViolationPresence, "C1", docContexto)
	})
	t.Run("un documento de contexto con add sin editar", func(t *testing.T) {
		t.Parallel()
		f, target, _ := preparedFixture(t)
		composeAll(t, target)
		appendText(t, filepath.Join(f.specs, docModelo), sectionModelo)
		replaceText(t, filepath.Join(f.specs, filepath.FromSlash(docContexto)), "Gestiona los pedidos.", "Gestiona los pedidos y reintenta (RF-008).")
		mustViolate(t, verifyOf(t, target), ViolationPresence, "C2", "API.md")
	})
	t.Run("una entrada remove no exige que el documento aparezca en el diff", func(t *testing.T) {
		t.Parallel()
		f := newIntegrationFixture(t, true)
		f.writeChange(t, ContextDeltaFileName, "## Entrada: context/ARQUITECTURA.md\n\n- Ancla: Componentes > Servicio de catálogo\n- Acción: remove\n- Motivo: el servicio de catálogo desaparece\n")
		commitAll(t, f.specs, "contexto con remove")
		target := f.target(t)
		if _, report, err := Prepare(context.Background(), target, PlanOptions{}); err != nil {
			t.Fatal(err)
		} else {
			mustNoErrors(t, report)
		}
		composeAll(t, target)
		appendText(t, filepath.Join(f.specs, docModelo), sectionModelo)

		// Sin editar el contexto: la presencia no se exige, pero el ancla sigue existiendo.
		report := verifyOf(t, target)
		mustNotViolate(t, report, ViolationPresence)
		mustViolate(t, report, ViolationAnchor, "C1", "Servicio de catálogo")

		replaceText(t, filepath.Join(f.specs, filepath.FromSlash(docContexto)), "### Servicio de catálogo\n\nGestiona los productos.\n\n", "")
		mustPass(t, verifyOf(t, target))
	})
	t.Run("las entradas esperadas se recalculan desde los ficheros actuales del cambio", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		replaceText(t, filepath.Join(f.changeRoot, ContextDeltaFileName), "se añade el reintento de pago", "se añade el reintento con retroceso")
		report := verifyOf(t, target)
		mustViolate(t, report, ViolationPresence, "cambiaron desde `prepare`", "axiom sdd integrate prepare")
		mustViolate(t, report, ViolationPresence, "C1")
	})
	t.Run("un delta cambiado después de prepare", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		f.writeChange(t, "specs/modelo-de-datos/spec.md", fixtureDeltaModelo+"\n### Requirement: Entidad Aviso\n\nTexto.\n")
		report := verifyOf(t, target)
		mustViolate(t, report, ViolationPresence, "cambiaron desde `prepare`")
		mustViolate(t, report, ViolationPresence, "Entidad Aviso")
	})
	t.Run("una entrada retirada de los ficheros del cambio", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		f.removeChange(t, "specs/modelo-de-datos")
		mustViolate(t, verifyOf(t, target), ViolationPresence, "Entidad Reintento", "ya no está")
	})
}

// Las composiciones se comprueban por huella after, no por ancla (D-15 b).
func TestVerifyChecksTheCompositionByItsAfterFingerprint(t *testing.T) {
	t.Parallel()
	t.Run("una composición sin aplicar", func(t *testing.T) {
		t.Parallel()
		f, target, _ := preparedFixture(t)
		f.applyDirectedEdits(t)
		report := verifyOf(t, target)
		mustViolate(t, report, ViolationCompose, "requisitos-funcionales", docRequisitos, "axiom sdd archive-compose")
	})
	t.Run("un documento retocado después de componerse", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		appendText(t, filepath.Join(f.specs, docRequisitos), "\nretoque a mano\n")
		mustViolate(t, verifyOf(t, target), ViolationCompose, "requisitos-funcionales", "after")
	})
	t.Run("una interrupción tras renombrar: el plan sigue pendiente y el documento ya está en after", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		plan := loadPlan(t, f)
		for i := range plan.Documents {
			if plan.Documents[i].Integration == workspace.IntegrationCompose {
				plan.Documents[i].State = StatePending
			}
		}
		for i := range plan.Entries {
			if plan.Entries[i].Kind == EntryCompose {
				plan.Entries[i].State = StatePending
			}
		}
		if err := WritePlan(filepath.Join(f.changeRoot, PlanFileName), plan); err != nil {
			t.Fatal(err)
		}
		mustPass(t, verifyOf(t, target))
		after := loadPlan(t, f)
		for _, document := range after.Documents {
			if document.Integration == workspace.IntegrationCompose && document.State != StateIntegrated {
				t.Errorf("verify marca como integrado el documento ya compuesto: %+v", document)
			}
		}
		for _, entry := range after.Entries {
			if entry.Kind == EntryCompose && entry.State != StateIntegrated {
				t.Errorf("verify marca como integrada la entrada compuesta: %+v", entry)
			}
		}
	})
	t.Run("pendiente con after igual a before solo se reconoce por el estado del plan", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		plan := loadPlan(t, f)
		for i := range plan.Documents {
			if plan.Documents[i].Integration == workspace.IntegrationCompose {
				plan.Documents[i].State = StatePending
				plan.Documents[i].AfterSHA256 = plan.Documents[i].BeforeSHA256
			}
		}
		if err := WritePlan(filepath.Join(f.changeRoot, PlanFileName), plan); err != nil {
			t.Fatal(err)
		}
		mustViolate(t, verifyOf(t, target), ViolationCompose, "requisitos-funcionales")
	})
}

// (b) Anclas (REQ-24.51 b, D-15).
func TestVerifyAnchorsOfTheDirectedSection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		section string // texto que sustituye a la sección del cambio
		ok      bool
		detail  string
	}{
		{name: "el título es el nombre del cambio", section: "\n## inc-99-ejemplo\n\nTexto (RF-008).\n", ok: true},
		{name: "el título empieza por el nombre del cambio", section: "\n## inc-99-ejemplo — Entidad Reintento\n\nTexto (RF-008).\n", ok: true},
		{name: "el nombre del cambio es prefijo de otro token", section: "\n## inc-99-ejemplo-bis: Entidad\n\nTexto (RF-008).\n", detail: "ninguna sección"},
		{name: "el nombre del cambio no abre el título", section: "\n## Entidad de inc-99-ejemplo\n\nTexto (RF-008).\n", detail: "ninguna sección"},
		{name: "dos secciones del cambio", section: "\n## inc-99-ejemplo: Uno\n\nTexto (RF-008).\n\n## inc-99-ejemplo: Dos\n\nTexto.\n", detail: "2 secciones"},
		{name: "un encabezado dentro de una valla no cuenta", section: "\n## Entidad\n\n```\n## inc-99-ejemplo: falso\n```\n", detail: "ninguna sección"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, target, _ := preparedFixture(t)
			composeAll(t, target)
			appendText(t, filepath.Join(f.specs, docModelo), tt.section)
			replaceText(t, filepath.Join(f.specs, filepath.FromSlash(docContexto)), "Gestiona los pedidos.", "Gestiona los pedidos y reintenta (RF-008).")
			appendText(t, filepath.Join(f.backend, filepath.FromSlash(docAPI)), sectionAPI)
			report := verifyOf(t, target)
			if tt.ok {
				mustPass(t, report)
				return
			}
			v := mustViolate(t, report, ViolationAnchor, "Entidad Reintento", docModelo, tt.detail)
			if !strings.Contains(v.Detail, "inc-99-ejemplo") {
				t.Errorf("el detalle nombra el título esperado: %+v", v)
			}
		})
	}
}

func TestVerifyAnchorsOfTheContextEntries(t *testing.T) {
	t.Parallel()
	t.Run("modify cuya ancla ya no existe", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		replaceText(t, filepath.Join(f.specs, filepath.FromSlash(docContexto)), "### Servicio de pedidos", "### Servicio de reintentos")
		mustViolate(t, verifyOf(t, target), ViolationAnchor, "C1", "Componentes > Servicio de pedidos")
	})
	t.Run("add cuya ancla queda repetida", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		appendText(t, filepath.Join(f.backend, filepath.FromSlash(docAPI)), "\n## Errores\n\nOtra vez.\n")
		mustViolate(t, verifyOf(t, target), ViolationAnchor, "C2", "Errores", "2 secciones")
	})
	t.Run("add cuya ancla no se creó", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		replaceText(t, filepath.Join(f.backend, filepath.FromSlash(docAPI)), "## Errores", "## Fallos")
		mustViolate(t, verifyOf(t, target), ViolationAnchor, "C2", "Errores", "ninguna sección")
	})
}

// (d) Traza (REQ-24.51 d).
func TestVerifyRequiresAReadableTraceWithTheExpectedSchema(t *testing.T) {
	t.Parallel()
	t.Run("sin plan", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		if err := os.Remove(filepath.Join(f.changeRoot, PlanFileName)); err != nil {
			t.Fatal(err)
		}
		report := verifyOf(t, target)
		mustViolate(t, report, ViolationTrace, PlanFileName, "axiom sdd integrate prepare")
		if len(report.Violations) != 1 {
			t.Errorf("sin plan no hay nada más que comprobar:\n%s", describeViolations(report))
		}
	})
	t.Run("YAML ilegible", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		if err := os.WriteFile(filepath.Join(f.changeRoot, PlanFileName), []byte("schema: [\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		mustViolate(t, verifyOf(t, target), ViolationTrace, PlanFileName)
	})
	t.Run("esquema distinto", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		path := filepath.Join(f.changeRoot, PlanFileName)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, bytes.Replace(data, []byte("axiom.sdd-integration/v1"), []byte("axiom.sdd-integration/v2"), 1), 0o644); err != nil {
			t.Fatal(err)
		}
		mustViolate(t, verifyOf(t, target), ViolationTrace, "axiom.sdd-integration/v1")
	})
	t.Run("clave desconocida", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		appendText(t, filepath.Join(f.changeRoot, PlanFileName), "extra: 1\n")
		mustViolate(t, verifyOf(t, target), ViolationTrace, PlanFileName)
	})
	t.Run("el plan es de otro cambio", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		plan := loadPlan(t, f)
		plan.Change = "inc-98-otro"
		if err := WritePlan(filepath.Join(f.changeRoot, PlanFileName), plan); err != nil {
			t.Fatal(err)
		}
		mustViolate(t, verifyOf(t, target), ViolationTrace, "inc-98-otro", "inc-99-ejemplo")
	})
}

// (e) Sustitución de IDs (REQ-24.51 e, REQ-24.59, D-15).
func TestVerifyRejectsAProvisionalIDLeftInTheAddedText(t *testing.T) {
	t.Parallel()
	t.Run("en una sección dirigida", func(t *testing.T) {
		t.Parallel()
		f, target, _ := preparedFixture(t)
		composeAll(t, target)
		appendText(t, filepath.Join(f.specs, docModelo), strings.Replace(sectionModelo, "RF-008", "RF-9001", 1))
		replaceText(t, filepath.Join(f.specs, filepath.FromSlash(docContexto)), "Gestiona los pedidos.", "Gestiona los pedidos y reintenta (RF-008).")
		appendText(t, filepath.Join(f.backend, filepath.FromSlash(docAPI)), sectionAPI)
		v := mustViolate(t, verifyOf(t, target), ViolationIDs, "RF-9001", "RF-008", docModelo)
		if v.Entry != "RF-9001" || !strings.Contains(v.File, docModelo) {
			t.Errorf("la violación nombra el provisional y el fichero: %+v", v)
		}
	})
	t.Run("en una entrada de contexto", func(t *testing.T) {
		t.Parallel()
		f, target, _ := preparedFixture(t)
		composeAll(t, target)
		appendText(t, filepath.Join(f.specs, docModelo), sectionModelo)
		replaceText(t, filepath.Join(f.specs, filepath.FromSlash(docContexto)), "Gestiona los pedidos.", "Gestiona los pedidos y reintenta (RF-9001).")
		appendText(t, filepath.Join(f.backend, filepath.FromSlash(docAPI)), sectionAPI)
		mustViolate(t, verifyOf(t, target), ViolationIDs, "RF-9001", "RF-008", docContexto)
	})
	t.Run("en el documento compuesto", func(t *testing.T) {
		t.Parallel()
		f, target, _ := integratedFixture(t)
		// El retoque rompe además la huella after (compose), y deja un provisional.
		appendText(t, filepath.Join(f.specs, docRequisitos), "\nVéase RF-9001.\n")
		report := verifyOf(t, target)
		mustViolate(t, report, ViolationIDs, "RF-9001", "RF-008", docRequisitos)
		mustViolate(t, report, ViolationCompose, "requisitos-funcionales")
	})
}

func TestVerifyCountsProvisionalIDsAgainstTheRecordedCommit(t *testing.T) {
	t.Parallel()
	// El texto preexistente puede mencionar el provisional: contar tokens, no líneas,
	// tolera ese texto y el reajuste del texto preexistente.
	setup := func(t *testing.T, after string) VerifyReport {
		f := newIntegrationFixture(t, true)
		writeRepoFile(t, f.specs, docModelo, fixtureModelo+"\nNota histórica: el borrador citaba RF-9001 dos veces, RF-9001.\n")
		commitAll(t, f.specs, "texto previo con el provisional")
		target := f.target(t)
		if _, report, err := Prepare(context.Background(), target, PlanOptions{}); err != nil {
			t.Fatal(err)
		} else {
			mustNoErrors(t, report)
		}
		composeAll(t, target)
		replaceText(t, filepath.Join(f.specs, filepath.FromSlash(docContexto)), "Gestiona los pedidos.", "Gestiona los pedidos y reintenta (RF-008).")
		appendText(t, filepath.Join(f.backend, filepath.FromSlash(docAPI)), sectionAPI)
		appendText(t, filepath.Join(f.specs, docModelo), after)
		return verifyOf(t, target)
	}

	t.Run("el texto previo no cuenta", func(t *testing.T) {
		t.Parallel()
		mustPass(t, setup(t, sectionModelo))
	})
	t.Run("una aparición más que en el commit registrado sí", func(t *testing.T) {
		t.Parallel()
		report := setup(t, strings.Replace(sectionModelo, "RF-008", "RF-9001", 1))
		mustViolate(t, report, ViolationIDs, "RF-9001", "RF-008", docModelo)
	})
	t.Run("reajustar el texto previo sin añadir apariciones se tolera", func(t *testing.T) {
		t.Parallel()
		f := newIntegrationFixture(t, true)
		writeRepoFile(t, f.specs, docModelo, fixtureModelo+"\nNota histórica: el borrador citaba RF-9001.\n")
		commitAll(t, f.specs, "texto previo con el provisional")
		target := f.target(t)
		if _, report, err := Prepare(context.Background(), target, PlanOptions{}); err != nil {
			t.Fatal(err)
		} else {
			mustNoErrors(t, report)
		}
		composeAll(t, target)
		replaceText(t, filepath.Join(f.specs, filepath.FromSlash(docContexto)), "Gestiona los pedidos.", "Gestiona los pedidos y reintenta (RF-008).")
		appendText(t, filepath.Join(f.backend, filepath.FromSlash(docAPI)), sectionAPI)
		replaceText(t, filepath.Join(f.specs, docModelo), "Nota histórica: el borrador citaba RF-9001.", "Nota histórica, reajustada: el borrador citaba RF-9001 en otra línea.")
		appendText(t, filepath.Join(f.specs, docModelo), sectionModelo)
		mustPass(t, verifyOf(t, target))
	})
}

// Un documento que no existía en el commit registrado se compara con el vacío.
func TestVerifyComparesANewDocumentWithAnEmptyCommittedText(t *testing.T) {
	t.Parallel()
	f := newIntegrationFixture(t, true)
	f.writeChange(t, ContextDeltaFileName, "## Entrada: context/PAGOS.md\n\n- Ancla: Reintentos\n- Acción: add\n- Motivo: documenta los pagos\n")
	commitAll(t, f.specs, "contexto nuevo")
	target := f.target(t)
	if _, report, err := Prepare(context.Background(), target, PlanOptions{}); err != nil {
		t.Fatal(err)
	} else {
		mustNoErrors(t, report)
	}
	composeAll(t, target)
	appendText(t, filepath.Join(f.specs, docModelo), sectionModelo)
	writeRepoFile(t, f.specs, "context/PAGOS.md", "# Pagos\n\n## Reintentos\n\nSe reintenta como RF-9001.\n")
	mustViolate(t, verifyOf(t, target), ViolationIDs, "RF-9001", "RF-008", "context/PAGOS.md")
}

// El informe es JSON estable: las violaciones nunca son null.
func TestVerifyReportJSONShape(t *testing.T) {
	t.Parallel()
	_, target, _ := integratedFixture(t)
	report := verifyOf(t, target)
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"schema":"axiom.sdd-integration-verify/v1","change":"inc-99-ejemplo","passed":true,"violations":[]}`; string(data) != want {
		t.Errorf("JSON = %s", data)
	}
	var violation Violation
	if err := json.Unmarshal([]byte(`{"check":"scope","entry":"C1","file":"a.md","detail":"d"}`), &violation); err != nil || violation != (Violation{Check: "scope", Entry: "C1", File: "a.md", Detail: "d"}) {
		t.Errorf("Violation = %+v, %v", violation, err)
	}
}

// Verify solo lee: ningún documento ni el índice de git cambian, y el repositorio
// elegido es el de la ruta, no el del proceso (design §8).
func TestVerifyNeverChangesTheWorkingTreeAndIgnoresTheProcessDirectory(t *testing.T) {
	// Sin t.Parallel: cambia el directorio del proceso (t.Chdir).
	f, target, _ := integratedFixture(t)
	other := t.TempDir()
	t.Chdir(other)
	before := snapshot(t, f.master)
	indexes := []string{gitIndexDigest(t, f.specs), gitIndexDigest(t, f.backend)}
	mustPass(t, verifyOf(t, target))
	changed := changedFiles(before, snapshot(t, f.master))
	if want := []string{"repo-specs/openspec/changes/inc-99-ejemplo/" + PlanFileName}; !reflect.DeepEqual(changed, want) {
		t.Errorf("solo cambia el plan: %v", changed)
	}
	if got := []string{gitIndexDigest(t, f.specs), gitIndexDigest(t, f.backend)}; !reflect.DeepEqual(got, indexes) {
		t.Error("verify nunca escribe el índice de git")
	}
}

// Con las marcas de tiempo del índice desfasadas (un `touch` de un fichero seguido
// tras el commit), `git diff` reescribiría `.git/index`: Verify promete solo leer,
// de modo que el índice de cada repositorio queda con los mismos bytes y el informe
// es el mismo que con el índice al día.
func TestVerifyDoesNotRewriteAStaleGitIndexAndReportsTheSame(t *testing.T) {
	t.Parallel()
	f, target, _ := integratedFixture(t)
	repos := []string{f.specs, f.backend}
	for _, repo := range repos {
		runGit(t, repo, "update-index", "-q", "--refresh")
	}
	fresh := verifyOf(t, target)
	mustPass(t, fresh)

	for _, repo := range repos {
		makeIndexStale(t, repo)
	}
	before := []string{gitIndexState(t, f.specs), gitIndexState(t, f.backend)}
	stale := verifyOf(t, target)
	if got := []string{gitIndexState(t, f.specs), gitIndexState(t, f.backend)}; !reflect.DeepEqual(got, before) {
		t.Error("verify reescribió el índice de git con las marcas de tiempo desfasadas")
	}
	if !reflect.DeepEqual(stale, fresh) {
		t.Errorf("el informe con el índice desfasado difiere del del índice al día:\n%s\n---\n%s", describeViolations(stale), describeViolations(fresh))
	}
}

// El verificador ignora las variables de git que redirigen el repositorio.
func TestVerifyIgnoresGitVariablesThatRedirectTheRepository(t *testing.T) {
	// Sin t.Parallel: fija variables de entorno de git (t.Setenv).
	f, target, _ := integratedFixture(t)
	foreign := newRepo(t, filepath.Join(t.TempDir(), "ajeno"))
	t.Setenv("GIT_DIR", filepath.Join(foreign, ".git"))
	t.Setenv("GIT_WORK_TREE", foreign)
	_ = f
	mustPass(t, verifyOf(t, target))
}
