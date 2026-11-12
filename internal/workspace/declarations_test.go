package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// declarationsBase es una configuración base válida a la que cada prueba añade
// los bloques `baseline:` y `technical_context:`.
const declarationsBase = `workspace:
  name: ejemplo
  topology: monorepo-embedded
  specs_repository: .
roles:
  core:
    name: Core
    repositories:
      - path: .
`

// parseDeclarations carga declarationsBase más el bloque dado con parseLayered.
func parseDeclarations(t *testing.T, blocks string) (*WorkspaceConfig, error, Declarations) {
	t.Helper()
	return parseLayered([]byte(declarationsBase + blocks))
}

// wantErrorContains comprueba que err existe y que su texto contiene cada
// fragmento; los errores de las declaraciones se acumulan, así que una misma
// violación puede convivir con otras.
func wantErrorContains(t *testing.T, err error, wants ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("se esperaba un error que contuviera %q, no hubo ninguno", wants)
	}
	for _, want := range wants {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no contiene %q:\n%v", want, err)
		}
	}
}

func TestDeclarationsAbsentAreTheZeroValue(t *testing.T) {
	cfg, baseErr, declarations := parseLayered([]byte(declarationsBase))
	if baseErr != nil {
		t.Fatalf("baseErr = %v, se esperaba nil", baseErr)
	}
	if declarations.ParseErr != nil || declarations.BaselineRetiredErr != nil ||
		declarations.TechnicalContext != nil || declarations.TechnicalContextErr != nil {
		t.Fatalf("sin bloques nuevos las declaraciones deben ser el valor cero, no %+v", declarations)
	}
	if cfg == nil || cfg.TechnicalContext != nil {
		t.Fatalf("la configuración base no debe llevar bloques nuevos: %+v", cfg)
	}
}

func TestDeclarationsValidBlocksAreLoaded(t *testing.T) {
	cfg, baseErr, declarations := parseDeclarations(t, `technical_context:
  maintained:
    entry: context/TECHNICAL_CONTEXT.md
    paths: [context/]
  reference:
    paths: [00_Glosario.md]
governance:
  context: docs/contexto.md
`)
	if baseErr != nil || declarations.TechnicalContextErr != nil || declarations.BaselineRetiredErr != nil {
		t.Fatalf("la declaración válida no debe fallar: base=%v contexto=%v retirada=%v", baseErr, declarations.TechnicalContextErr, declarations.BaselineRetiredErr)
	}
	if cfg.TechnicalContext == nil {
		t.Fatalf("la configuración debe incluir el bloque poblado: %+v", cfg)
	}
	if cfg.Governance.Context != "docs/contexto.md" {
		t.Errorf("governance.context debe convivir con la declaración sin interpretarse: %q", cfg.Governance.Context)
	}
	if got := cfg.TechnicalContext.Maintained; got == nil || got.Entry != "context/TECHNICAL_CONTEXT.md" || len(got.Paths) != 1 {
		t.Errorf("maintained mal cargado: %+v", got)
	}
	if got := cfg.TechnicalContext.Reference; got == nil || got.Entry != "" || len(got.Paths) != 1 {
		t.Errorf("reference mal cargado: %+v", got)
	}
}

// baselineDoc compone un bloque baseline residual con un único documento.
func baselineDoc(entries ...string) string {
	return "baseline:\n  documents:\n" + strings.Join(entries, "")
}

func proseDoc(domain, path string) string {
	return "    - domain: " + domain + "\n      path: " + path + "\n"
}

// REQ-30.21: la baseline retirada y un technical_context inválido se reportan
// cada uno por su canal.
func TestDeclarationsErrorsAccumulate(t *testing.T) {
	_, baseErr, declarations := parseDeclarations(t, baselineDoc(proseDoc("a", "../uno.md"))+"technical_context:\n  maintained:\n    paths: []\n")
	if baseErr != nil {
		t.Fatalf("baseErr = %v", baseErr)
	}
	if declarations.BaselineRetiredErr == nil {
		t.Fatal("un baseline residual debe dejar BaselineRetiredErr")
	}
	wantErrorContains(t, declarations.TechnicalContextErr, "requiere al menos una ruta en 'paths'")
}

func TestDeclarationsTechnicalContextValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  string // sustituye a la configuración base cuando no está vacío
		blocks  string
		valid   bool
		wantErr []string
	}{
		{"ambos modos", "", "technical_context:\n  maintained:\n    entry: context/TECHNICAL_CONTEXT.md\n    paths: [context/]\n  reference:\n    paths: [00_Glosario.md]\n", true, nil},
		{"solo referencia", "", "technical_context:\n  reference:\n    paths: [00_Glosario.md]\n", true, nil},
		{"bloque vacio", "", "technical_context: {}\n", false, []string{"requiere al menos el modo 'maintained' o 'reference'"}},
		{"bloque nulo", "", "technical_context:\n", false, []string{"requiere al menos el modo 'maintained' o 'reference'"}},
		{"bloque que no es un mapeo", "", "technical_context: [a]\n", false, []string{"debe ser un mapeo con los modos"}},
		{"modo desconocido", "", "technical_context:\n  readonly:\n    paths: [a.md]\n", false, []string{"clave desconocida 'readonly'", "technical_context", "claves admitidas: maintained, reference"}},
		{"clave desconocida en un modo", "", "technical_context:\n  reference:\n    paths: [a.md]\n    extra: 1\n", false, []string{"clave desconocida 'extra'", "technical_context.reference", "entry, paths"}},
		{"modo nulo", "", "technical_context:\n  maintained:\n", false, []string{"el modo 'technical_context.maintained' debe ser un mapeo"}},
		{"paths vacio", "", "technical_context:\n  maintained:\n    paths: []\n", false, []string{"el modo 'technical_context.maintained' requiere al menos una ruta"}},
		{"paths ausente", "", "technical_context:\n  reference:\n    entry: a.md\n", false, []string{"requiere al menos una ruta"}},
		{"entrada fuera de las rutas del modo", "", "technical_context:\n  maintained:\n    entry: README.md\n    paths: [context/]\n", false, []string{"'technical_context.maintained.entry' ('README.md') debe estar contenida"}},
		{"entrada igual a una ruta", "", "technical_context:\n  maintained:\n    entry: context/INDEX.md\n    paths: [context/INDEX.md]\n", true, nil},
		{"entrada contenida sin distinguir mayusculas", "", "technical_context:\n  maintained:\n    entry: Context/Index.md\n    paths: [context]\n", true, nil},
		{"entrada absoluta", "", "technical_context:\n  maintained:\n    entry: /abs/a.md\n    paths: [context/]\n", false, []string{"'technical_context.maintained.entry'", "relativa"}},
		{"ruta en ambos modos", "", "technical_context:\n  maintained:\n    paths: [context/a.md]\n  reference:\n    paths: [context/a.md]\n", false, []string{"'context/a.md'", "technical_context.maintained", "technical_context.reference", "no puede pertenecer a los dos modos"}},
		{"solapamiento por contencion", "", "technical_context:\n  maintained:\n    paths: [context/]\n  reference:\n    paths: [context/a.md]\n", false, []string{"'context/'", "'context/a.md'", "solapa"}},
		{"solapamiento sin distinguir mayusculas", "", "technical_context:\n  maintained:\n    paths: [Context/]\n  reference:\n    paths: [context/a.md]\n", false, []string{"solapa"}},
		{"prefijos que no son contencion", "", "technical_context:\n  maintained:\n    paths: [context]\n  reference:\n    paths: [context-extra]\n", true, nil},
		{"ruta absoluta y ruta fuera de la maestra", "", "technical_context:\n  maintained:\n    paths: [/abs/dir, ../../fuera/]\n", false, []string{"'/abs/dir'", "relativa", "'../../fuera/'", "sale de la carpeta maestra"}},
		{"ruta que sale de la maestra en embebido", "", "technical_context:\n  reference:\n    paths: [../hermano.md]\n", false, []string{"'../hermano.md'", "sale de la carpeta maestra"}},
		{
			name:    "multirepo permite subir al repositorio hermano",
			config:  "workspace:\n  name: e\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: repo-backend\n",
			blocks:  "technical_context:\n  maintained:\n    paths: [../repo-backend/docs/]\n",
			valid:   true,
			wantErr: nil,
		},
		{
			name:    "multirepo no permite salir de la carpeta maestra",
			config:  "workspace:\n  name: e\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: repo-backend\n",
			blocks:  "technical_context:\n  maintained:\n    paths: [../../fuera/]\n",
			wantErr: []string{"sale de la carpeta maestra"},
		},
		{
			name:    "specs_repository terminado en openspec desplaza a su padre",
			config:  "workspace:\n  name: e\n  topology: multirepo\n  specs_repository: repo-specs/openspec\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n",
			blocks:  "technical_context:\n  maintained:\n    paths: [../repo-backend/docs/]\n  reference:\n    paths: [../../fuera.md]\n",
			wantErr: []string{"'../../fuera.md'", "sale de la carpeta maestra"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := declarationsBase
			if tc.config != "" {
				config = tc.config
			}
			cfg, baseErr, declarations := parseLayered([]byte(config + tc.blocks))
			if baseErr != nil {
				t.Fatalf("la configuración base es válida y no debe fallar: %v", baseErr)
			}
			if tc.valid {
				if declarations.TechnicalContextErr != nil {
					t.Fatalf("la declaración debe ser válida: contexto=%v", declarations.TechnicalContextErr)
				}
				if declarations.TechnicalContext == nil || cfg.TechnicalContext == nil {
					t.Fatal("una declaración válida debe poblar el contexto técnico")
				}
				return
			}
			wantErrorContains(t, declarations.TechnicalContextErr, tc.wantErr...)
			if declarations.TechnicalContext != nil || cfg.TechnicalContext != nil {
				t.Error("un bloque inválido no debe poblar el contexto técnico ni parcial ni por defecto")
			}
		})
	}
}

// REQ-24.17: las claves desconocidas fuera de los bloques nuevos se siguen ignorando.
func TestDeclarationsUnknownKeysOutsideTheNewBlocksAreIgnored(t *testing.T) {
	data := declarationsBase + "clave_futura: 1\ngovernance:\n  language: es\n  otra: x\n"
	cfg, baseErr, declarations := parseLayered([]byte(data))
	if baseErr != nil || declarations.BaselineRetiredErr != nil || declarations.TechnicalContextErr != nil || declarations.ParseErr != nil {
		t.Fatalf("las claves desconocidas ajenas a los bloques nuevos no deben fallar: %v %+v", baseErr, declarations)
	}
	if cfg == nil || cfg.Governance.Language != "es" {
		t.Fatalf("la configuración debe cargarse: %+v", cfg)
	}
	if _, err := ParseConfig([]byte(data)); err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
}

// D-03: ParseErr solo es no nulo cuando existe un axiom.yaml que no se puede
// leer como YAML; sin contenido útil conserva el valor cero.
func TestDeclarationsParseErrOnlyForUnreadableYAML(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		parseErr bool
	}{
		{"yaml ilegible", "workspace: [sin cerrar\n", true},
		{"vacio", "", false},
		{"solo espacios", "  \n\n", false},
		{"solo comentarios", "# nada\n", false},
		{"base valida", declarationsBase, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, baseErr, declarations := parseLayered([]byte(tc.data))
			if (declarations.ParseErr != nil) != tc.parseErr {
				t.Fatalf("ParseErr = %v, se esperaba no nulo = %v", declarations.ParseErr, tc.parseErr)
			}
			if tc.parseErr && baseErr == nil {
				t.Error("un YAML ilegible también es un error de la configuración base")
			}
			if declarations.BaselineRetiredErr != nil || declarations.TechnicalContext != nil {
				t.Errorf("sin bloques legibles no hay declaraciones: %+v", declarations)
			}
		})
	}
}

// D-03, los casos de la tabla: cada canal de error es independiente.
func TestLayeredErrorChannels(t *testing.T) {
	const invalidContext = "technical_context: {}\n"
	const validContext = "technical_context:\n  reference:\n    paths: [a.md]\n"
	const noName = "workspace:\n  topology: monorepo-embedded\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n"

	t.Run("YAML ilegible", func(t *testing.T) {
		cfg, baseErr, declarations := parseLayered([]byte("workspace: [sin cerrar\n"))
		if cfg != nil || baseErr == nil || declarations.ParseErr == nil {
			t.Fatalf("cfg=%v baseErr=%v ParseErr=%v", cfg, baseErr, declarations.ParseErr)
		}
		if _, err := ParseConfig([]byte("workspace: [sin cerrar\n")); err == nil {
			t.Error("ParseConfig debe fallar, como hasta ahora")
		}
	})

	t.Run("base invalida con declaracion valida", func(t *testing.T) {
		cfg, baseErr, declarations := parseLayered([]byte(noName + validContext))
		wantErrorContains(t, baseErr, "'workspace.name' es obligatorio")
		if cfg != nil {
			t.Error("una base inválida no devuelve configuración")
		}
		if declarations.TechnicalContextErr != nil || declarations.TechnicalContext == nil {
			t.Errorf("la declaración válida se evalúa aunque la base falle: %+v", declarations)
		}
		if declarations.ParseErr != nil {
			t.Errorf("ParseErr solo se refiere al YAML ilegible: %v", declarations.ParseErr)
		}
		if _, err := ParseConfig([]byte(noName + validContext)); err == nil || !strings.Contains(err.Error(), "'workspace.name'") {
			t.Errorf("ParseConfig debe fallar con el error de la base: %v", err)
		}
	})

	t.Run("declaracion invalida con base valida", func(t *testing.T) {
		cfg, baseErr, declarations := parseLayered([]byte(declarationsBase + invalidContext))
		if baseErr != nil {
			t.Fatalf("la base es válida: %v", baseErr)
		}
		if cfg == nil || cfg.Workspace.Name != "ejemplo" || cfg.TechnicalContext != nil {
			t.Fatalf("la configuración base se lee sin el bloque inválido: %+v", cfg)
		}
		wantErrorContains(t, declarations.TechnicalContextErr, "requiere al menos el modo 'maintained' o 'reference'")
		_, err := ParseConfig([]byte(declarationsBase + invalidContext))
		wantErrorContains(t, err, "requiere al menos el modo 'maintained' o 'reference'")
	})

	t.Run("declaracion invalida y base invalida", func(t *testing.T) {
		cfg, baseErr, declarations := parseLayered([]byte(noName + invalidContext))
		if cfg != nil || baseErr == nil || declarations.TechnicalContextErr == nil {
			t.Fatalf("cfg=%v baseErr=%v TechnicalContextErr=%v", cfg, baseErr, declarations.TechnicalContextErr)
		}
		_, err := ParseConfig([]byte(noName + invalidContext))
		wantErrorContains(t, err, "'workspace.name'", "requiere al menos el modo 'maintained' o 'reference'")
	})

	t.Run("ParseConfig acumula la baseline retirada y el contexto invalido", func(t *testing.T) {
		_, err := ParseConfig([]byte(declarationsBase + "baseline:\n  documents: []\n" + invalidContext))
		wantErrorContains(t, err, BaselineRetiredCode, "requiere al menos el modo 'maintained' o 'reference'")
	})
}

func TestDeclarationsLoadConfigRejectsAnInvalidDeclaration(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "axiom.yaml")
	if err := os.WriteFile(file, []byte(declarationsBase+"technical_context: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(file)
	wantErrorContains(t, err, "requiere al menos el modo 'maintained' o 'reference'")

	if err := os.WriteFile(file, []byte(declarationsBase+"technical_context:\n  reference:\n    paths: [a.md]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(file)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.TechnicalContext == nil || cfg.TechnicalContext.Reference == nil {
		t.Fatalf("LoadConfig debe incluir el bloque válido: %+v", cfg)
	}
}
