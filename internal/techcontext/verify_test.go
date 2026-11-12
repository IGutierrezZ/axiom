package techcontext

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	editedContext = "# Arquitectura\n\n## Componentes\n\n### Servicio de catálogo\n\nGestiona los productos.\n\n### Servicio de pedidos\n\nGestiona los pedidos y reintenta el cobro fallido.\n"
	editedAPI     = fixtureAPI + "\n## Errores\n\nEl cobro reintentado devuelve un error 402.\n"
)

// preparedFixture deja el espacio con git, el plan escrito y las ediciones de las
// dos entradas del delta aún sin hacer.
func preparedFixture(t *testing.T) *fixture {
	t.Helper()
	f := newFixture(t, true)
	_, report, err := Prepare(context.Background(), f.target(t))
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	mustNoErrors(t, report)
	return f
}

// applyEdits hace las dos ediciones que el delta de contexto describe.
func (f *fixture) applyEdits(t *testing.T) {
	t.Helper()
	writeTestFile(t, filepath.Join(f.specs, "context", "ARQUITECTURA.md"), editedContext)
	writeTestFile(t, filepath.Join(f.backend, "docs", "API.md"), editedAPI)
}

func (f *fixture) verify(t *testing.T) VerifyReport {
	t.Helper()
	report, err := Verify(context.Background(), f.target(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	return report
}

func mustViolate(t *testing.T, report VerifyReport, check string, fragments ...string) {
	t.Helper()
	if report.Passed {
		t.Fatalf("la verificación debía fallar con %q", check)
	}
	for _, violation := range report.Violations {
		if violation.Check != check {
			continue
		}
		matches := true
		for _, fragment := range fragments {
			if !strings.Contains(violation.Detail+violation.File+violation.Entry, fragment) {
				matches = false
			}
		}
		if matches {
			return
		}
	}
	t.Fatalf("ninguna violación %q contiene %q; violaciones: %+v", check, fragments, report.Violations)
}

func mustNotViolate(t *testing.T, report VerifyReport, check string) {
	t.Helper()
	for _, violation := range report.Violations {
		if violation.Check == check {
			t.Fatalf("violación %q inesperada: %+v", check, violation)
		}
	}
}

func TestVerifyPassesAndMarksTheEntriesAsIntegrated(t *testing.T) {
	f := preparedFixture(t)
	fixedClock(t)
	f.applyEdits(t)
	report := f.verify(t)
	if !report.Passed || len(report.Violations) != 0 {
		t.Fatalf("la verificación debía pasar: %+v", report.Violations)
	}
	if report.Schema != VerifySchema || report.Change != "inc-99-ejemplo" {
		t.Errorf("informe = %+v", report)
	}
	trace, err := ReadTrace(filepath.Join(f.changeRoot, TraceFileName))
	if err != nil {
		t.Fatal(err)
	}
	if trace.VerifiedAt == nil || len(trace.Entries) != 2 {
		t.Fatalf("traza = %+v", trace)
	}
	for _, entry := range trace.Entries {
		if entry.State != StateIntegrated {
			t.Errorf("entrada %s = %s, quería integrated", entry.ID, entry.State)
		}
	}
}

func TestVerifyRecordsCreatedDocuments(t *testing.T) {
	f := newFixture(t, true)
	f.writeChange(t, DeltaFileName, "## Entrada: context/NUEVO.md\n\n- Ancla: Pagos\n- Acción: add\n- Motivo: documento nuevo\n")
	commitAll(t, f.specs, "delta nuevo")
	if _, report, err := Prepare(context.Background(), f.target(t)); err != nil || len(report.Errors) > 0 {
		t.Fatalf("Prepare: %v %v", err, report.Errors)
	}
	writeTestFile(t, filepath.Join(f.specs, "context", "NUEVO.md"), "# Nuevo\n\n## Pagos\n\nTexto.\n")
	if report := f.verify(t); !report.Passed {
		t.Fatalf("violaciones: %+v", report.Violations)
	}
	trace, _ := ReadTrace(filepath.Join(f.changeRoot, TraceFileName))
	if trace.Entries[0].Action != ActionCreated {
		t.Errorf("acción = %s, quería created: el documento no existía en el commit de prepare", trace.Entries[0].Action)
	}
}

func TestVerifyDoesNotHideACommitMadeBetweenPrepareAndVerify(t *testing.T) {
	f := preparedFixture(t)
	f.applyEdits(t)
	commitAll(t, f.specs, "contexto")
	commitAll(t, f.backend, "api")
	if report := f.verify(t); !report.Passed {
		t.Fatalf("las ediciones confirmadas también cuentan: %+v", report.Violations)
	}
}

func TestVerifyTrace(t *testing.T) {
	t.Run("sin plan", func(t *testing.T) {
		f := newFixture(t, false)
		report := f.verify(t)
		mustViolate(t, report, ViolationTrace, TraceFileName, "axiom archive context prepare")
	})
	t.Run("plan de otro cambio", func(t *testing.T) {
		f := preparedFixture(t)
		trace, _ := ReadTrace(filepath.Join(f.changeRoot, TraceFileName))
		trace.Change = "otro-cambio"
		if err := WriteTrace(filepath.Join(f.changeRoot, TraceFileName), trace); err != nil {
			t.Fatal(err)
		}
		mustViolate(t, f.verify(t), ViolationTrace, "otro-cambio", "inc-99-ejemplo")
	})
	t.Run("plan ilegible", func(t *testing.T) {
		f := preparedFixture(t)
		writeTestFile(t, filepath.Join(f.changeRoot, TraceFileName), "schema: [\n")
		mustViolate(t, f.verify(t), ViolationTrace, "no se puede leer")
	})
	t.Run("plan ajeno de una baseline", func(t *testing.T) {
		f := newFixture(t, false)
		foreign := strings.Replace(planV1, "documents: []", "documents:\n  - domain: modelo\n    path: 03.md\n    integration: compose\n    before_sha256: "+testDigest, 1)
		writeTestFile(t, filepath.Join(f.changeRoot, TraceFileName), strings.Replace(foreign, "inc-99-ejemplo", "inc-99-ejemplo", 1))
		mustViolate(t, f.verify(t), ViolationTrace, "no es un plan de contexto técnico", "axiom sdd integrate")
	})
}

// REQ-30.19 y REQ-30.30: la verificación falla cerrada ante una declaración
// inválida o un almacén sin ficheros.
func TestVerifyFailsClosedOnTheDeclarationAndTheArtifactStore(t *testing.T) {
	t.Run("almacén sin ficheros", func(t *testing.T) {
		f := preparedFixture(t)
		f.applyEdits(t)
		target := f.target(t)
		target.ArtifactStore = "engram"
		report, err := Verify(context.Background(), target)
		if err != nil {
			t.Fatal(err)
		}
		mustViolate(t, report, ViolationPresence, "artifact_store: openspec", "engram")
	})
	t.Run("technical_context inválido", func(t *testing.T) {
		f := preparedFixture(t)
		f.applyEdits(t)
		writeTestFile(t, filepath.Join(f.specs, "axiom.yaml"), strings.Replace(fixtureConfig, "paths:\n      - context/\n      - ../repo-backend/docs/\n", "paths: []\n", 1))
		report := f.verify(t)
		mustViolate(t, report, ViolationPresence, "no es válida")
		trace, _ := ReadTrace(filepath.Join(f.changeRoot, TraceFileName))
		if trace.VerifiedAt != nil {
			t.Error("una verificación fallida no toca el plan")
		}
	})
}

func TestVerifyPresence(t *testing.T) {
	t.Run("las ediciones no se hicieron", func(t *testing.T) {
		f := preparedFixture(t)
		report := f.verify(t)
		mustViolate(t, report, ViolationPresence, "C1", "no aparece en el diff de archive")
		mustViolate(t, report, ViolationPresence, "C2", "no aparece en el diff de archive")
	})
	t.Run("una entrada nueva en el delta que el plan no conoce", func(t *testing.T) {
		f := preparedFixture(t)
		f.applyEdits(t)
		f.writeChange(t, DeltaFileName, fixtureContextImpact+"\n## Entrada: context/OTRO.md\n\n- Ancla: X\n- Acción: add\n- Motivo: y\n")
		mustViolate(t, f.verify(t), ViolationPresence, "figura en los ficheros del cambio y no en el plan")
	})
	t.Run("los deltas cambiaron desde prepare", func(t *testing.T) {
		f := preparedFixture(t)
		f.applyEdits(t)
		f.writeChange(t, DeltaFileName, strings.Replace(fixtureContextImpact, "se añade el reintento de pago", "otro motivo", 1))
		report := f.verify(t)
		mustViolate(t, report, ViolationPresence, TraceFileName, "cambiaron desde `prepare`")
	})
	t.Run("Sin impacto no exige ediciones", func(t *testing.T) {
		f := newFixture(t, true)
		f.writeChange(t, DeltaFileName, "Sin impacto: solo cambia código interno\n")
		commitAll(t, f.specs, "sin impacto")
		if _, report, err := Prepare(context.Background(), f.target(t)); err != nil || len(report.Errors) > 0 {
			t.Fatalf("Prepare: %v %v", err, report.Errors)
		}
		if report := f.verify(t); !report.Passed {
			t.Fatalf("violaciones: %+v", report.Violations)
		}
	})
}

func TestVerifyAnchors(t *testing.T) {
	t.Run("la sección no existe tras la edición", func(t *testing.T) {
		f := preparedFixture(t)
		f.applyEdits(t)
		writeTestFile(t, filepath.Join(f.backend, "docs", "API.md"), fixtureAPI+"\nTexto sin sección.\n")
		mustViolate(t, f.verify(t), ViolationAnchor, "C2", "no resuelve a ninguna sección")
	})
	t.Run("la sección está duplicada", func(t *testing.T) {
		f := preparedFixture(t)
		f.applyEdits(t)
		writeTestFile(t, filepath.Join(f.backend, "docs", "API.md"), editedAPI+"\n## Errores\n\nOtra vez.\n")
		mustViolate(t, f.verify(t), ViolationAnchor, "C2", "resuelve a 2 secciones")
	})
	t.Run("remove exige que la sección ya no exista", func(t *testing.T) {
		f := newFixture(t, true)
		f.writeChange(t, DeltaFileName, "## Entrada: context/ARQUITECTURA.md\n\n- Ancla: Componentes > Servicio de pedidos\n- Acción: remove\n- Motivo: el servicio desaparece\n")
		commitAll(t, f.specs, "remove")
		if _, report, err := Prepare(context.Background(), f.target(t)); err != nil || len(report.Errors) > 0 {
			t.Fatalf("Prepare: %v %v", err, report.Errors)
		}
		writeTestFile(t, filepath.Join(f.specs, "context", "ARQUITECTURA.md"), fixtureContext+"\nCambio.\n")
		mustViolate(t, f.verify(t), ViolationAnchor, "C1", "sigue existiendo")
		writeTestFile(t, filepath.Join(f.specs, "context", "ARQUITECTURA.md"), "# Arquitectura\n\n## Componentes\n\n### Servicio de catálogo\n\nGestiona los productos.\n")
		if report := f.verify(t); !report.Passed {
			t.Fatalf("violaciones: %+v", report.Violations)
		}
	})
}

func TestVerifyScope(t *testing.T) {
	t.Run("ruta de referencia", func(t *testing.T) {
		f := preparedFixture(t)
		f.applyEdits(t)
		writeTestFile(t, filepath.Join(f.specs, "referencia", "legado", "VIEJO.md"), "# Viejo editado\n")
		mustViolate(t, f.verify(t), ViolationScope, "referencia/legado/VIEJO.md", "ruta de referencia", "REQ-24.30")
	})
	t.Run("fichero ajeno", func(t *testing.T) {
		f := preparedFixture(t)
		f.applyEdits(t)
		writeTestFile(t, filepath.Join(f.specs, "README.md"), "# ajeno\n")
		mustViolate(t, f.verify(t), ViolationScope, "README.md", "no pertenece al conjunto permitido")
	})
	t.Run("fichero borrado fuera de la carpeta del cambio", func(t *testing.T) {
		f := preparedFixture(t)
		f.applyEdits(t)
		if err := os.Remove(filepath.Join(f.specs, "referencia", "legado", "VIEJO.md")); err != nil {
			t.Fatal(err)
		}
		mustViolate(t, f.verify(t), ViolationScope, "se ha borrado", "archive nunca borra documentos")
	})
	t.Run("la spec canónica del dominio del cambio y el índice están permitidos", func(t *testing.T) {
		f := preparedFixture(t)
		f.applyEdits(t)
		writeTestFile(t, filepath.Join(f.specs, "openspec", "specs", "pagos", "spec.md"), "# Especificación: pagos\n")
		writeTestFile(t, filepath.Join(f.specs, "openspec", "INDEX.md"), "# Índice\n")
		if report := f.verify(t); !report.Passed {
			t.Fatalf("violaciones: %+v", report.Violations)
		}
	})
	t.Run("la spec de un dominio que el cambio no toca no está permitida", func(t *testing.T) {
		f := preparedFixture(t)
		f.applyEdits(t)
		writeTestFile(t, filepath.Join(f.specs, "openspec", "specs", "otro", "spec.md"), "# Especificación: otro\n")
		mustViolate(t, f.verify(t), ViolationScope, "openspec/specs/otro/spec.md")
	})
	t.Run("ruta mantenida con Sin impacto", func(t *testing.T) {
		f := newFixture(t, true)
		f.writeChange(t, DeltaFileName, "Sin impacto: solo cambia código interno\n")
		commitAll(t, f.specs, "sin impacto")
		if _, report, err := Prepare(context.Background(), f.target(t)); err != nil || len(report.Errors) > 0 {
			t.Fatalf("Prepare: %v %v", err, report.Errors)
		}
		writeTestFile(t, filepath.Join(f.specs, "context", "ARQUITECTURA.md"), editedContext)
		mustViolate(t, f.verify(t), ViolationScope, "context/ARQUITECTURA.md", "Sin impacto", "REQ-24.57")
	})
	t.Run("la suciedad previa que no cambia no es del archivado", func(t *testing.T) {
		f := newFixture(t, true)
		writeTestFile(t, filepath.Join(f.specs, "NOTAS.md"), "borrador previo\n")
		if _, report, err := Prepare(context.Background(), f.target(t)); err != nil || len(report.Errors) > 0 {
			t.Fatalf("Prepare: %v %v", err, report.Errors)
		}
		f.applyEdits(t)
		if report := f.verify(t); !report.Passed {
			t.Fatalf("la foto previa no cuenta: %+v", report.Violations)
		}
		writeTestFile(t, filepath.Join(f.specs, "NOTAS.md"), "borrador previo, retocado\n")
		mustViolate(t, f.verify(t), ViolationScope, "NOTAS.md")
	})
	t.Run("el repositorio dejó de ser git", func(t *testing.T) {
		f := preparedFixture(t)
		f.applyEdits(t)
		if err := os.RemoveAll(filepath.Join(f.backend, ".git")); err != nil {
			t.Fatal(err)
		}
		mustViolate(t, f.verify(t), ViolationScope, "repo-backend")
	})
}

func TestVerifyFailureLeavesTheTraceUntouched(t *testing.T) {
	f := preparedFixture(t)
	planFile := filepath.Join(f.changeRoot, TraceFileName)
	before, _ := os.ReadFile(planFile)
	report := f.verify(t)
	if report.Passed || len(report.Violations) == 0 {
		t.Fatal("sin ediciones la verificación debe fallar")
	}
	after, _ := os.ReadFile(planFile)
	if string(before) != string(after) {
		t.Error("una verificación fallida no modifica el plan")
	}
}
