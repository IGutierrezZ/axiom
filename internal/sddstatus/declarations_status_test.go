package sddstatus

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const (
	declEmbeddedBase  = "workspace:\n  name: ejemplo-embebido\n  topology: monorepo-embedded\n  specs_repository: .\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n"
	declCanonicalBase = "workspace:\n  name: ejemplo-canonico\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  backend:\n    name: Backend\n    repositories:\n      - path: repo-backend\n  frontend:\n    name: Frontend\n    repositories:\n      - path: repo-frontend\n"

	// Tres documentos: uno con perfil de requisito, uno de prosa por defecto
	// (directed) y uno de prosa en modo solo registro.
	declThreeDocuments = `baseline:
  documents:
    - domain: requisitos-funcionales
      path: 01_Requisitos_Funcionales.md
      requirement:
        heading: '^### (?P<id>RF-\d+): (?P<title>.+)$'
        level: 3
        render: '### {id}: {title}'
    - domain: modelo-de-datos
      path: docs\03_Modelo_Datos.md
    - domain: flujos-de-negocio
      path: 04_Flujos_Negocio.md
      integration: register
`
	declBothModes = "technical_context:\n  maintained:\n    entry: context/TECHNICAL_CONTEXT.md\n    paths: [context/]\n  reference:\n    paths: [00_Glosario.md]\n"
)

// statusJSON devuelve el status v2 proyectado como mapa de claves de primer nivel.
func statusJSON(t *testing.T, status Status) map[string]json.RawMessage {
	t.Helper()
	projected, err := ProjectStatusV2(status)
	if err != nil {
		t.Fatalf("ProjectStatusV2: %v", err)
	}
	raw, err := json.Marshal(projected)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	return fields
}

func decodeField[T any](t *testing.T, fields map[string]json.RawMessage, key string) T {
	t.Helper()
	raw, ok := fields[key]
	if !ok {
		t.Fatalf("el status no contiene la clave %q", key)
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("clave %q: %v", key, err)
	}
	return value
}

// embeddedWithDeclaration monta un proyecto embebido con un cambio listo y los
// bloques dados añadidos a su axiom.yaml.
func embeddedWithDeclaration(t *testing.T, blocks string) workspacetest.Workspace {
	t.Helper()
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+blocks)
	seedReadyChange(t, ws.Master, "demo", planningHomeTasks)
	return ws
}

// REQ-24.19 y R1: sin declaración no aparece ninguna clave nueva, ni con un
// proyecto solo de openspec/ ni con un axiom.yaml que no declara los bloques.
func TestDeclarationsStatusWithoutDeclarationHasNoNewKeys(t *testing.T) {
	bare := t.TempDir()
	seedReadyChange(t, bare, "demo", planningHomeTasks)
	reference := statusJSON(t, resolveFrom(t, bare, "demo"))

	cases := map[string]string{
		"axiom.yaml sin bloques":                   declEmbeddedBase,
		"clave de primer nivel desconocida":        declEmbeddedBase + "clave_futura: 1\n",
		"governance.context sin technical_context": declEmbeddedBase + "governance:\n  context: docs/contexto.md\n",
	}
	for name, config := range cases {
		t.Run(name, func(t *testing.T) {
			ws := workspacetest.Embedded(t, ".")
			workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), config)
			seedReadyChange(t, ws.Master, "demo", planningHomeTasks)
			status := resolveFrom(t, ws.Master, "demo")
			fields := statusJSON(t, status)
			for _, key := range []string{"baseline", "technicalContext"} {
				if _, present := fields[key]; present {
					t.Errorf("sin declaración no debe aparecer %q", key)
				}
			}
			if len(fields) != len(reference) {
				t.Errorf("claves = %d, se esperaban las %d de un proyecto sin axiom.yaml", len(fields), len(reference))
			}
			if len(status.Notes) != 0 {
				t.Errorf("notes = %v, sin declaración no hay avisos", status.Notes)
			}
		})
	}

	t.Run("proyecto sin axiom.yaml", func(t *testing.T) {
		for _, key := range []string{"baseline", "technicalContext"} {
			if _, present := reference[key]; present {
				t.Errorf("un proyecto solo de openspec/ no debe proyectar %q", key)
			}
		}
	})
}

// REQ-24.19 y 24.28: proyección con rutas absolutas y los tres modos de integración.
func TestDeclarationsStatusProjectsBothBlocksWithAbsolutePaths(t *testing.T) {
	ws := embeddedWithDeclaration(t, declThreeDocuments+declBothModes)
	for name, resolve := range map[string]func() Status{
		"desde el directorio": func() Status { return resolveFrom(t, ws.Master, "demo") },
		"raíz explícita": func() Status {
			status, err := Resolve(ResolveOptions{WorkspaceRoot: ws.Master, ChangeName: "demo"})
			if err != nil {
				t.Fatal(err)
			}
			return status
		},
	} {
		t.Run(name, func(t *testing.T) {
			fields := statusJSON(t, resolve())

			baseline := decodeField[BaselineProjection](t, fields, "baseline")
			want := []BaselineDocumentProjection{
				{Domain: "requisitos-funcionales", Path: filepath.Join(ws.Master, "01_Requisitos_Funcionales.md"), Integration: "compose"},
				{Domain: "modelo-de-datos", Path: filepath.Join(ws.Master, "docs", "03_Modelo_Datos.md"), Integration: "directed"},
				{Domain: "flujos-de-negocio", Path: filepath.Join(ws.Master, "04_Flujos_Negocio.md"), Integration: "register"},
			}
			if !reflect.DeepEqual(baseline.Documents, want) {
				t.Fatalf("baseline.documents = %+v\nse esperaba %+v", baseline.Documents, want)
			}

			context := decodeField[TechnicalContextProjection](t, fields, "technicalContext")
			if context.Maintained == nil || context.Reference == nil {
				t.Fatalf("technicalContext = %+v, se esperaban los dos modos", context)
			}
			if want := []string{filepath.Join(ws.Master, "context")}; !reflect.DeepEqual(context.Maintained.Paths, want) {
				t.Errorf("maintained.paths = %v, se esperaba %v", context.Maintained.Paths, want)
			}
			if want := filepath.Join(ws.Master, "context", "TECHNICAL_CONTEXT.md"); context.Maintained.Entry != want {
				t.Errorf("maintained.entry = %q, se esperaba %q", context.Maintained.Entry, want)
			}
			if want := []string{filepath.Join(ws.Master, "00_Glosario.md")}; !reflect.DeepEqual(context.Reference.Paths, want) || context.Reference.Entry != "" {
				t.Errorf("reference = %+v, se esperaba %v sin entry", context.Reference, want)
			}
		})
	}
}

// Un modo no declarado no aparece, y `entry` solo aparece si se declaró.
func TestDeclarationsStatusProjectsOnlyTheDeclaredMode(t *testing.T) {
	ws := embeddedWithDeclaration(t, "technical_context:\n  reference:\n    paths: [00_Glosario.md]\n")
	fields := statusJSON(t, resolveFrom(t, ws.Master, "demo"))
	if _, present := fields["baseline"]; present {
		t.Error("sin baseline declarada no debe aparecer la clave baseline")
	}
	var context map[string]json.RawMessage
	if err := json.Unmarshal(fields["technicalContext"], &context); err != nil {
		t.Fatal(err)
	}
	if _, present := context["maintained"]; present {
		t.Error("un modo no declarado no debe aparecer")
	}
	var reference map[string]json.RawMessage
	if err := json.Unmarshal(context["reference"], &reference); err != nil {
		t.Fatal(err)
	}
	if _, present := reference["entry"]; present {
		t.Error("entry no declarada no debe aparecer")
	}
}

// REQ-24.18: un bloque inválido no se proyecta, ni parcial ni por defecto, y no
// arrastra al otro bloque.
func TestDeclarationsStatusNeverProjectsAnInvalidBlock(t *testing.T) {
	t.Run("baseline inválida con contexto válido", func(t *testing.T) {
		ws := embeddedWithDeclaration(t, "baseline:\n  documents:\n    - domain: a\n      path: ../fuera.md\n    - domain: b\n      path: b.md\n"+declBothModes)
		fields := statusJSON(t, resolveFrom(t, ws.Master, "demo"))
		if _, present := fields["baseline"]; present {
			t.Error("una baseline inválida no se proyecta, ni siquiera sus documentos válidos")
		}
		if _, present := fields["technicalContext"]; !present {
			t.Error("el contexto válido se proyecta aunque la baseline sea inválida")
		}
	})
	t.Run("contexto inválido con baseline válida", func(t *testing.T) {
		ws := embeddedWithDeclaration(t, declThreeDocuments+"technical_context:\n  maintained:\n    paths: []\n")
		fields := statusJSON(t, resolveFrom(t, ws.Master, "demo"))
		if _, present := fields["technicalContext"]; present {
			t.Error("un contexto inválido no se proyecta")
		}
		if _, present := fields["baseline"]; !present {
			t.Error("la baseline válida se proyecta aunque el contexto sea inválido")
		}
	})
	t.Run("axiom.yaml ilegible", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), "workspace: [sin cerrar\n")
		seedReadyChange(t, ws.Master, "demo", planningHomeTasks)
		fields := statusJSON(t, resolveFrom(t, ws.Master, "demo"))
		for _, key := range []string{"baseline", "technicalContext"} {
			if _, present := fields[key]; present {
				t.Errorf("un axiom.yaml ilegible no proyecta %q", key)
			}
		}
	})
	t.Run("base inválida con declaración válida sin puntero", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), "workspace: {name: x, topology: monorepo-embedded}\n"+declThreeDocuments)
		seedReadyChange(t, ws.Master, "demo", planningHomeTasks)
		fields := statusJSON(t, resolveFrom(t, ws.Master, "demo"))
		if _, present := fields["baseline"]; !present {
			t.Error("las declaraciones se evalúan aunque la base falle (D-03): la baseline válida se proyecta")
		}
	})
}

// REQ-24.19: la proyección existe con o sin cambio seleccionado.
func TestDeclarationsStatusProjectsWithoutASelectedChange(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase+declThreeDocuments+declBothModes)
	status := resolveFrom(t, ws.Master, "")
	if status.ChangeName != nil {
		t.Fatalf("changeName = %v, se esperaba nulo", *status.ChangeName)
	}
	fields := statusJSON(t, status)
	if string(fields["changeName"]) != "null" {
		t.Errorf("changeName = %s, se esperaba null", fields["changeName"])
	}
	for _, key := range []string{"baseline", "technicalContext"} {
		if _, present := fields[key]; !present {
			t.Errorf("la proyección debe existir sin cambio seleccionado: falta %q", key)
		}
	}
}

// REQ-24.20: planningHome y los localizadores de artefactos no cambian con la declaración.
func TestDeclarationsStatusDoesNotAlterPlanningHomeOrArtifacts(t *testing.T) {
	plain := workspacetest.Embedded(t, ".")
	seedReadyChange(t, plain.Master, "demo", planningHomeTasks)
	declared := embeddedWithDeclaration(t, declThreeDocuments+declBothModes)
	// Un delta de contexto presente no se proyecta ni cambia los localizadores.
	workspacetest.WriteFile(t, filepath.Join(declared.Home, "changes", "demo", "context-impact.md"), "# Delta de contexto: demo\n")

	before := resolveFrom(t, plain.Master, "demo")
	after := resolveFrom(t, declared.Master, "demo")
	if after.PlanningHome.Mode != ActionModeRepoLocal || after.PlanningHome.Path != filepath.Join(declared.Master, "openspec") {
		t.Fatalf("planningHome = %+v, no debe cambiar con la declaración", after.PlanningHome)
	}
	relative := func(root string, paths ArtifactPaths) ArtifactPaths {
		rebase := func(list []string) []string {
			out := make([]string, 0, len(list))
			for _, item := range list {
				out = append(out, strings.TrimPrefix(item, root))
			}
			return out
		}
		return ArtifactPaths{Proposal: rebase(paths.Proposal), Specs: rebase(paths.Specs), Design: rebase(paths.Design), Tasks: rebase(paths.Tasks), ApplyProgress: rebase(paths.ApplyProgress), VerifyReport: rebase(paths.VerifyReport)}
	}
	if !reflect.DeepEqual(relative(plain.Master, before.ArtifactPaths), relative(declared.Master, after.ArtifactPaths)) {
		t.Errorf("artifactPaths cambió con la declaración: %+v frente a %+v", before.ArtifactPaths, after.ArtifactPaths)
	}
	if !reflect.DeepEqual(relative(plain.Master, before.ContextFiles), relative(declared.Master, after.ContextFiles)) {
		t.Errorf("contextFiles cambió con la declaración: %+v frente a %+v", before.ContextFiles, after.ContextFiles)
	}
	if !reflect.DeepEqual(before.Artifacts, after.Artifacts) {
		t.Errorf("artifacts = %+v, se esperaba %+v", after.Artifacts, before.Artifacts)
	}
	if !reflect.DeepEqual(before.Dependencies, after.Dependencies) || before.NextRecommended != after.NextRecommended {
		t.Errorf("dependencies/nextRecommended cambiaron con una declaración válida y compatible")
	}
}

// REQ-24.21: una clave baseline en openspec/config.yaml no tiene efecto.
func TestDeclarationsStatusIgnoresADeclarationInOpenSpecConfig(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), declEmbeddedBase)
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "config.yaml"), "baseline:\n  documents:\n    - domain: a\n      path: a.md\ntechnical_context:\n  reference:\n    paths: [a.md]\n")
	seedReadyChange(t, ws.Master, "demo", planningHomeTasks)
	fields := statusJSON(t, resolveFrom(t, ws.Master, "demo"))
	for _, key := range []string{"baseline", "technicalContext"} {
		if _, present := fields[key]; present {
			t.Errorf("openspec/config.yaml no declara %q", key)
		}
	}
}

// Espacio canónico: las rutas son absolutas y relativas al repositorio de
// especificaciones, y las del contexto pueden subir a un repositorio hermano.
func TestDeclarationsStatusInTheCanonicalWorkspace(t *testing.T) {
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), declCanonicalBase+declThreeDocuments+"technical_context:\n  maintained:\n    paths: [../repo-backend/docs/]\n")
	seedReadyChange(t, ws.Specs, "inc-demo", planningHomeTasks)
	for name, start := range map[string]string{
		"carpeta maestra":                 ws.Master,
		"repositorio de especificaciones": ws.Specs,
		"repositorio de código":           filepath.Join(ws.Backend, "src"),
	} {
		t.Run(name, func(t *testing.T) {
			fields := statusJSON(t, resolveFrom(t, start, "inc-demo"))
			baseline := decodeField[BaselineProjection](t, fields, "baseline")
			if len(baseline.Documents) != 3 || baseline.Documents[0].Path != filepath.Join(ws.Specs, "01_Requisitos_Funcionales.md") {
				t.Fatalf("baseline = %+v, los documentos cuelgan del repositorio de especificaciones", baseline)
			}
			context := decodeField[TechnicalContextProjection](t, fields, "technicalContext")
			if want := []string{filepath.Join(ws.Master, "repo-backend", "docs")}; context.Maintained == nil || !reflect.DeepEqual(context.Maintained.Paths, want) {
				t.Fatalf("technicalContext = %+v, se esperaba %v", context, want)
			}
		})
	}
}
