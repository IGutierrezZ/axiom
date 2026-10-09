package workspace

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// ownershipOf evalúa la consulta sobre el contenido dado de axiom.yaml con la
// misma carga por capas que usan LoadConfig, Locate y workspace validate.
func ownershipOf(t *testing.T, content string) (BaselineOwnership, error) {
	t.Helper()
	_, _, declarations := parseLayered([]byte(content))
	return declarations.Ownership()
}

// REQ-24.22: las tres respuestas de la consulta única y los casos límite.
func TestOwnershipAnswers(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    BaselineOwnership
		cause   string // fragmento esperado de la causa con indeterminate
	}{
		{name: "sin baseline", content: locateEmbeddedConfig, want: BaselineDefault},
		{name: "baseline valida", content: locateEmbeddedConfig + locateValidBaseline, want: BaselineProject},
		{name: "baseline invalida", content: locateEmbeddedConfig + locateInvalidBaseline, want: BaselineIndeterminate, cause: "al menos un documento"},
		{name: "yaml ilegible", content: "workspace: [sin cerrar\n", want: BaselineIndeterminate, cause: "error al parsear axiom.yaml"},
		{
			name:    "error ajeno al bloque (sin roles) con baseline por defecto",
			content: "workspace:\n  name: x\n  topology: monorepo-embedded\n",
			want:    BaselineDefault,
		},
		{
			name:    "error ajeno al bloque (sin roles) con baseline valida",
			content: "workspace:\n  name: x\n  topology: monorepo-embedded\n" + locateValidBaseline,
			want:    BaselineProject,
		},
		{
			name:    "technical_context invalido no altera la baseline por defecto",
			content: locateEmbeddedConfig + "technical_context:\n  maintained:\n    paths: []\n",
			want:    BaselineDefault,
		},
		{
			name:    "technical_context invalido no altera la baseline valida",
			content: locateEmbeddedConfig + locateValidBaseline + "technical_context:\n  maintained:\n    paths: []\n",
			want:    BaselineProject,
		},
		{
			name:    "baseline invalida con technical_context valido",
			content: locateEmbeddedConfig + locateInvalidBaseline + "technical_context:\n  reference:\n    paths: [00_Glosario.md]\n",
			want:    BaselineIndeterminate,
			cause:   "al menos un documento",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, cause := ownershipOf(t, tt.content)
			if got != tt.want {
				t.Fatalf("Ownership = %q, se esperaba %q", got, tt.want)
			}
			if (tt.want == BaselineIndeterminate) != (cause != nil) {
				t.Fatalf("la causa debe existir solo con indeterminate: respuesta %q, causa %v", got, cause)
			}
			if tt.cause != "" && !strings.Contains(cause.Error(), tt.cause) {
				t.Errorf("la causa %q no contiene %q", cause, tt.cause)
			}
		})
	}

	t.Run("el valor cero es por defecto", func(t *testing.T) {
		if got, cause := (Declarations{}).Ownership(); got != BaselineDefault || cause != nil {
			t.Fatalf("Ownership() del valor cero = (%q, %v), se esperaba (por defecto, nil)", got, cause)
		}
	})
}

// REQ-24.22: para un mismo axiom.yaml, la respuesta no depende del punto de
// partida de la resolución ni de que el valor viva en un repositorio de
// especificaciones distinto.
func TestLocationOwnershipAgreesFromEveryStart(t *testing.T) {
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), locateCanonicalConfig+locateValidBaseline)
	for name, start := range map[string]string{
		"carpeta maestra":                 ws.Master,
		"repositorio de especificaciones": ws.Specs,
		"repositorio de codigo":           filepath.Join(ws.Backend, "src"),
	} {
		got, err := mustLocate(t, start).BaselineOwnership()
		if got != BaselineProject || err != nil {
			t.Errorf("%s: BaselineOwnership = (%q, %v), se esperaba (del proyecto, nil)", name, got, err)
		}
	}
}

// REQ-24.22 y REQ-24.62: una causa de almacén no altera la respuesta.
func TestOwnershipIgnoresTheArtifactStore(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), locateEmbeddedConfig+locateValidBaseline)
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "config.yaml"), "artifact_store: none\n")
	if got, err := mustLocate(t, ws.Master).BaselineOwnership(); got != BaselineProject || err != nil {
		t.Fatalf("BaselineOwnership con almacén none = (%q, %v), se esperaba (del proyecto, nil)", got, err)
	}
}

// REQ-24.22: con un puntero cuyo destino no se puede leer, la resolución se
// rechaza antes y no hay respuesta de baseline.
func TestOwnershipHasNoAnswerWhenLocateRejects(t *testing.T) {
	ws := workspacetest.Canonical(t, workspacetest.WithUnreadableConfig())
	loc, err := Locate(ws.Master)
	var locateErr *LocateError
	if !errors.As(err, &locateErr) || locateErr.Kind != LocateConfigUnloadable {
		t.Fatalf("Locate = (%+v, %v), se esperaba un *LocateError config_unloadable", loc, err)
	}
}

// REQ-24.23: GuardSpecsWrite deja pasar la baseline por defecto y rechaza las
// otras dos con una continuación.
func TestGuardSpecsWrite(t *testing.T) {
	t.Run("por defecto no rechaza", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		if err := mustLocate(t, ws.Master).GuardSpecsWrite("axiom archive sync"); err != nil {
			t.Fatalf("GuardSpecsWrite con la baseline por defecto = %v, se esperaba nil", err)
		}
	})
	t.Run("sin axiom.yaml no rechaza", func(t *testing.T) {
		if err := (Location{}).GuardSpecsWrite("axiom archive sync"); err != nil {
			t.Fatalf("GuardSpecsWrite sin declaraciones = %v, se esperaba nil", err)
		}
	})

	t.Run("del proyecto nombra la baseline y la continuacion", func(t *testing.T) {
		ws := workspacetest.Canonical(t)
		configPath := filepath.Join(ws.Specs, "axiom.yaml")
		workspacetest.WriteFile(t, configPath, locateCanonicalConfig+locateValidBaseline+"    - domain: flujos-de-negocio\n      path: 04_Flujos_Negocio.md\n")
		err := mustLocate(t, ws.Master).GuardSpecsWrite("axiom archive coldstart")
		var second *SecondBaselineError
		if !errors.As(err, &second) {
			t.Fatalf("GuardSpecsWrite = %v, se esperaba un *SecondBaselineError", err)
		}
		if second.Ownership != BaselineProject || second.Operation != "axiom archive coldstart" || second.Cause != nil {
			t.Errorf("campos = %+v", second)
		}
		if second.ConfigPath != configPath || second.Master != ws.Master {
			t.Errorf("ConfigPath/Master = %q / %q, se esperaban %q / %q", second.ConfigPath, second.Master, configPath, ws.Master)
		}
		if got := strings.Join(second.Domains, ","); got != "modelo-de-datos,flujos-de-negocio" {
			t.Errorf("Domains = %q, se esperaban los dominios en orden de declaración", got)
		}
		for _, want := range []string{"axiom archive coldstart", "modelo-de-datos, flujos-de-negocio", "retira el bloque `baseline:`", "flujo de archive", configPath} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("el mensaje no contiene %q:\n%v", want, err)
			}
		}
	})

	t.Run("indeterminada nombra la causa y workspace validate", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), locateEmbeddedConfig+locateInvalidBaseline)
		err := mustLocate(t, ws.Master).GuardSpecsWrite("axiom archive sync")
		var second *SecondBaselineError
		if !errors.As(err, &second) {
			t.Fatalf("GuardSpecsWrite = %v, se esperaba un *SecondBaselineError", err)
		}
		if second.Ownership != BaselineIndeterminate || second.Cause == nil || len(second.Domains) != 0 {
			t.Errorf("campos = %+v", second)
		}
		if !errors.Is(err, second.Cause) {
			t.Errorf("Unwrap debe exponer la causa %v", second.Cause)
		}
		for _, want := range []string{"axiom archive sync", "no se puede determinar", "al menos un documento", "axiom workspace validate --path"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("el mensaje no contiene %q:\n%v", want, err)
			}
		}
	})

	t.Run("yaml ilegible sin puntero es indeterminada", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), "workspace: [sin cerrar\n")
		err := mustLocate(t, ws.Master).GuardSpecsWrite("axiom archive sync")
		var second *SecondBaselineError
		if !errors.As(err, &second) || second.Ownership != BaselineIndeterminate {
			t.Fatalf("GuardSpecsWrite = %v, se esperaba un *SecondBaselineError indeterminado", err)
		}
	})
}

// REQ-2.2: la comparación de planning homes es por ruta y no exige que existan.
func TestSamePlanningHome(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "uno")
	other := filepath.Join(root, "dos")
	workspacetest.MkdirAll(t, existing)
	workspacetest.MkdirAll(t, other)

	t.Run("misma ruta existente", func(t *testing.T) {
		if !SamePlanningHome(existing, filepath.Join(existing, "..", "uno")) {
			t.Fatal("dos grafias del mismo directorio existente deben ser el mismo planning home")
		}
	})
	t.Run("directorios existentes distintos", func(t *testing.T) {
		if SamePlanningHome(existing, other) {
			t.Fatal("dos directorios existentes distintos no son el mismo planning home")
		}
	})
	t.Run("ninguno existe, mismo texto", func(t *testing.T) {
		missing := filepath.Join(root, "M", "openspec")
		if !SamePlanningHome(missing, filepath.Join(root, "M", "x", "..", "openspec")) {
			t.Fatal("el respaldo léxico debe igualar rutas limpias idénticas aunque no existan")
		}
	})
	t.Run("ninguno existe, distintos", func(t *testing.T) {
		if SamePlanningHome(filepath.Join(root, "M", "openspec"), filepath.Join(root, "M", "repo-specs", "openspec")) {
			t.Fatal("rutas distintas que no existen no son el mismo planning home")
		}
	})
	t.Run("solo uno existe", func(t *testing.T) {
		if SamePlanningHome(existing, filepath.Join(root, "inexistente")) {
			t.Fatal("un directorio existente y una ruta distinta que no existe no son el mismo planning home")
		}
		if !SamePlanningHome(filepath.Join(root, "inexistente", "openspec"), filepath.Join(root, "inexistente", ".", "openspec")) {
			t.Fatal("el respaldo léxico debe cubrir el caso en que alguna ruta no existe")
		}
	})
	t.Run("mayusculas", func(t *testing.T) {
		missingLower := filepath.Join(root, "m", "openspec")
		missingUpper := filepath.Join(root, "M", "OpenSpec")
		got := SamePlanningHome(missingLower, missingUpper)
		if want := runtime.GOOS == "windows"; got != want {
			t.Fatalf("SamePlanningHome sin distinguir mayúsculas = %v en %s, se esperaba %v", got, runtime.GOOS, want)
		}
	})
}
