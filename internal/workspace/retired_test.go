package workspace

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// REQ-30.21: un bloque `baseline:` residual nunca se ignora ni se degrada a «sin
// baseline»: cada canal de carga lo recibe como *BaselineRetiredError.

func TestRetiredBaselineIsATypedErrorOnParse(t *testing.T) {
	for name, block := range map[string]string{
		"con documentos":   "baseline:\n  documents:\n    - domain: a\n      path: a.md\n",
		"sin documentos":   "baseline:\n  documents: []\n",
		"mapa vacío":       "baseline: {}\n",
		"nulo":             "baseline:\n",
		"no es un mapeo":   "baseline: [a]\n",
		"con clave futura": "baseline:\n  clave_futura: 1\n",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseConfig([]byte(declarationsBase + block))
			var retired *BaselineRetiredError
			if !errors.As(err, &retired) {
				t.Fatalf("ParseConfig = %v, se esperaba un *BaselineRetiredError", err)
			}
			if retired.Code() != "baseline_retired" || retired.Line != 10 {
				t.Errorf("código %q, línea %d: se esperaba baseline_retired en la línea 10", retired.Code(), retired.Line)
			}
			wantErrorContains(t, err, "baseline_retired", "línea 10", "`baseline:`", "axiom archive adopt scan --path <ruta> --change <cambio>")
		})
	}
}

func TestRetiredBaselineDoesNotAffectAConfigWithoutIt(t *testing.T) {
	for name, data := range map[string]string{
		"sin bloques":                declarationsBase,
		"clave anidada baseline":     declarationsBase + "governance:\n  baseline: x\n",
		"clave con otro prefijo":     declarationsBase + "baseline_futura: 1\n",
		"technical_context sano":     declarationsBase + "technical_context:\n  reference:\n    paths: [a.md]\n",
		"governance sin interpretar": declarationsBase + "governance:\n  context: docs/c.md\n",
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := ParseConfig([]byte(data))
			if err != nil || cfg == nil {
				t.Fatalf("ParseConfig = %v, %v: un axiom.yaml sin `baseline:` no se ve afectado", cfg, err)
			}
			if _, _, declarations := parseLayered([]byte(data)); declarations.BaselineRetiredErr != nil {
				t.Errorf("BaselineRetiredErr = %v, se esperaba nil", declarations.BaselineRetiredErr)
			}
		})
	}
}

func TestRetiredBaselineLoadConfigNamesThePathAndTheMigrationTarget(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "axiom.yaml")
	workspacetest.WriteFile(t, file, declarationsBase+declarationsRetiredBlock)

	_, err := LoadConfig(file)
	var retired *BaselineRetiredError
	if !errors.As(err, &retired) {
		t.Fatalf("LoadConfig = %v, se esperaba un *BaselineRetiredError", err)
	}
	if retired.Path != file {
		t.Errorf("Path = %q, se esperaba %q", retired.Path, file)
	}
	wantErrorContains(t, err, file, "axiom archive adopt scan --path", dir)
}

// declarationsRetiredBlock es el bloque residual que usan estas pruebas.
const declarationsRetiredBlock = "baseline:\n  documents:\n    - domain: modelo-de-datos\n      path: 03_Modelo_Datos.md\n"

func TestRetiredBaselineLocateDoesNotRejectAndKeepsTheCause(t *testing.T) {
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), locateCanonicalConfig+declarationsRetiredBlock)

	loc := mustLocate(t, ws.Master)
	retired := loc.Declarations.BaselineRetiredErr
	if retired == nil {
		t.Fatal("Locate no rechaza (D-03) y deja el bloque retirado en Declarations.BaselineRetiredErr")
	}
	if want := filepath.Join(ws.Specs, "axiom.yaml"); retired.Path != want {
		t.Errorf("Path = %q, se esperaba %q", retired.Path, want)
	}
	if loc.PlanningHome != ws.Home || loc.Config == nil {
		t.Errorf("el planning home y la configuración base se resuelven igualmente: %+v", loc)
	}
}

func TestRetiredBaselineGuardSpecsWrite(t *testing.T) {
	t.Run("sin baseline no se rechaza", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), locateEmbeddedConfig)
		if err := mustLocate(t, ws.Master).GuardSpecsWrite("sync"); err != nil {
			t.Fatalf("GuardSpecsWrite = %v, se esperaba nil", err)
		}
	})
	t.Run("con baseline residual se rechaza con la continuación", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), locateEmbeddedConfig+declarationsRetiredBlock)
		err := mustLocate(t, ws.Master).GuardSpecsWrite("sync")
		var retired *BaselineRetiredError
		if !errors.As(err, &retired) || !strings.Contains(err.Error(), "axiom archive adopt scan") {
			t.Fatalf("GuardSpecsWrite = %v, se esperaba un *BaselineRetiredError con la migración", err)
		}
	})
}

// ReadDeclarations es la carga por capas aplicada al axiom.yaml propio de un
// destino: coincide con la de Locate para el mismo fichero y nunca necesita
// resolver el planning home.
func TestReadDeclarations(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		path := filepath.Join(dir, name)
		workspacetest.WriteFile(t, path, content)
		return path
	}

	t.Run("sin fichero es el valor cero", func(t *testing.T) {
		got := ReadDeclarations(filepath.Join(dir, "no-existe.yaml"))
		if got.ParseErr != nil || got.BaselineRetiredErr != nil || got.TechnicalContext != nil || got.TechnicalContextErr != nil {
			t.Fatalf("ReadDeclarations sin fichero = %+v, se esperaba el valor cero", got)
		}
	})
	t.Run("legible sin bloques nuevos", func(t *testing.T) {
		got := ReadDeclarations(write("sin-bloques.yaml", locateEmbeddedConfig))
		if got.ParseErr != nil || got.BaselineRetiredErr != nil || got.TechnicalContext != nil {
			t.Fatalf("ReadDeclarations = %+v, se esperaba el valor cero", got)
		}
	})
	t.Run("baseline residual y technical_context valido", func(t *testing.T) {
		path := write("residual.yaml", locateEmbeddedConfig+declarationsRetiredBlock+"technical_context:\n  reference:\n    paths: [00_Glosario.md]\n")
		got := ReadDeclarations(path)
		if got.BaselineRetiredErr == nil || got.BaselineRetiredErr.Path != path || got.TechnicalContextErr != nil || got.TechnicalContext == nil {
			t.Fatalf("ReadDeclarations = %+v, se esperaban la baseline retirada con su ruta y el contexto válido", got)
		}
	})
	t.Run("cada bloque tiene su canal de error", func(t *testing.T) {
		got := ReadDeclarations(write("invalidas.yaml", locateEmbeddedConfig+"baseline:\n  documents: []\ntechnical_context:\n  maintained:\n    paths: []\n"))
		if got.BaselineRetiredErr == nil || got.TechnicalContextErr == nil || got.ParseErr != nil {
			t.Fatalf("ReadDeclarations = %+v, se esperaban los dos errores de bloque sin ParseErr", got)
		}
	})
	t.Run("una base invalida no oculta las declaraciones", func(t *testing.T) {
		got := ReadDeclarations(write("base-invalida.yaml", locateInvalidBase+declarationsRetiredBlock))
		if got.ParseErr != nil || got.BaselineRetiredErr == nil {
			t.Fatalf("ReadDeclarations = %+v, se esperaba la baseline retirada pese a la base inválida", got)
		}
	})
	t.Run("yaml ilegible deja la causa en ParseErr", func(t *testing.T) {
		got := ReadDeclarations(write("ilegible.yaml", "workspace: [sin cerrar\n"))
		if got.ParseErr == nil || got.BaselineRetiredErr != nil {
			t.Fatalf("ReadDeclarations = %+v, se esperaba solo ParseErr", got)
		}
	})
	t.Run("un fichero que no se puede leer deja la causa en ParseErr", func(t *testing.T) {
		// Un directorio con el nombre del fichero existe pero no se puede leer.
		workspacetest.MkdirAll(t, filepath.Join(dir, "directorio.yaml"))
		if got := ReadDeclarations(filepath.Join(dir, "directorio.yaml")); got.ParseErr == nil {
			t.Fatal("ParseErr debe nombrar la causa de un fichero que no se puede leer")
		}
	})
	t.Run("coincide con Locate para el mismo fichero", func(t *testing.T) {
		for name, content := range map[string]string{
			"sin baseline": locateEmbeddedConfig,
			"residual":     locateEmbeddedConfig + declarationsRetiredBlock,
			"ilegible":     "workspace: [sin cerrar\n",
		} {
			ws := workspacetest.Embedded(t, ".")
			workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), content)
			fromLocate := mustLocate(t, ws.Master).Declarations
			fromRead := ReadDeclarations(filepath.Join(ws.Master, "axiom.yaml"))
			if (fromLocate.BaselineRetiredErr == nil) != (fromRead.BaselineRetiredErr == nil) || (fromLocate.ParseErr == nil) != (fromRead.ParseErr == nil) {
				t.Errorf("%s: Locate responde %+v y ReadDeclarations %+v", name, fromLocate, fromRead)
			}
		}
	})
}
