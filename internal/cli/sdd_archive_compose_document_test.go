package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Las pruebas del modo documento de `axiom sdd archive-compose` (inc-24,
// REQ-24.38, REQ-24.47 y REQ-24.48). Todos los textos son inventados.

const composeBaselineConfig = `workspace:
  name: ejemplo-compose
  topology: monorepo-embedded
  specs_repository: ""
roles:
  core:
    name: Core
    repositories:
      - path: .
baseline:
  documents:
    - domain: requisitos-funcionales
      path: docs/01_Requisitos_Funcionales.md
      requirement:
        heading: '^#+ (?P<id>RF-\d+): (?P<title>.+)$'
        level: 3
        render: '### {id}: {title}'
    - domain: modelo-datos
      path: docs/03_Modelo_Datos.md
`

const composeProfileDocument = "# Requisitos\n\n## Catálogo\n\n" +
	"### RF-100: Alta\n\nCuerpo original de alta.\n\n" +
	"### Notas de implantación\n\nSección contigua.\n\n" +
	"### RF-101: Baja\n\nCuerpo de baja.\n"

const composeProfileDelta = "## MODIFIED Requirements\n\n" +
	"### Requirement: RF-100: Alta de cliente\n\nCuerpo nuevo de alta.\n\n(Previously: cuerpo anterior)\n"

const composeProfileComposed = "# Requisitos\n\n## Catálogo\n\n" +
	"### RF-100: Alta de cliente\n\nCuerpo nuevo de alta.\n\n" +
	"### Notas de implantación\n\nSección contigua.\n\n" +
	"### RF-101: Baja\n\nCuerpo de baja.\n"

// composeWorkspace monta un proyecto embebido con la baseline declarada y sus
// documentos, y devuelve su raíz.
func composeWorkspace(t *testing.T, config string, files map[string]string) string {
	t.Helper()
	ws := workspacetest.Embedded(t, "")
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), config)
	for name, content := range files {
		workspacetest.WriteFile(t, filepath.Join(ws.Specs, filepath.FromSlash(name)), content)
	}
	return ws.Specs
}

func runCompose(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	var out, errOut bytes.Buffer
	err = runSDDArchiveComposeWithStderr(args, &out, &errOut)
	return out.String(), errOut.String(), err
}

func TestRunSDDArchiveComposeDocumentWithProfileComposesTheDeclaredDocument(t *testing.T) {
	root := composeWorkspace(t, composeBaselineConfig, map[string]string{
		"docs/01_Requisitos_Funcionales.md":                            composeProfileDocument,
		"openspec/changes/inc-99/specs/requisitos-funcionales/spec.md": composeProfileDelta,
		"docs/03_Modelo_Datos.md":                                      "# Modelo\n",
	})
	delta := "openspec/changes/inc-99/specs/requisitos-funcionales/spec.md"

	stdout, stderr, err := runCompose(t, "--cwd", root, "--document", "requisitos-funcionales", "--delta", delta)
	if err != nil {
		t.Fatalf("runSDDArchiveComposeWithStderr() error = %v", err)
	}
	if stdout != composeProfileComposed {
		t.Errorf("salida = %q\nse esperaba %q", stdout, composeProfileComposed)
	}
	if !strings.Contains(stderr, "Aviso:") || !strings.Contains(stderr, "### Notas de implantación") {
		t.Errorf("la salida de error debe avisar de la sección contigua: %q", stderr)
	}
	original, err := os.ReadFile(filepath.Join(root, "docs", "01_Requisitos_Funcionales.md"))
	if err != nil || string(original) != composeProfileDocument {
		t.Errorf("el comando no debe modificar el documento (err = %v)", err)
	}
}

func TestRunSDDArchiveComposeDocumentKeepsCRLFAndBOM(t *testing.T) {
	crlf := func(text string) string { return "\ufeff" + strings.ReplaceAll(text, "\n", "\r\n") }
	root := composeWorkspace(t, composeBaselineConfig, map[string]string{
		"docs/01_Requisitos_Funcionales.md": crlf(composeProfileDocument),
		"cambio/delta.md":                   composeProfileDelta,
	})
	stdout, _, err := runCompose(t, "--cwd", root, "--document", "requisitos-funcionales", "--delta", "cambio/delta.md")
	if err != nil {
		t.Fatalf("runSDDArchiveComposeWithStderr() error = %v", err)
	}
	if stdout != crlf(composeProfileComposed) {
		t.Errorf("la salida debe conservar BOM y CRLF:\n%q", stdout)
	}
}

func TestRunSDDArchiveComposeDocumentResolvesRelativePathsAgainstTheWorkspaceRoot(t *testing.T) {
	root := composeWorkspace(t, composeBaselineConfig, map[string]string{
		"docs/01_Requisitos_Funcionales.md": composeProfileDocument,
		"cambio/delta.md":                   composeProfileDelta,
		"src/.gitkeep":                      "",
	})

	// --cwd apunta a un subdirectorio: el workspaceRoot es el del marcador más
	// cercano, y la ruta relativa del delta y la de salida se resuelven contra él.
	workspacetest.MkdirAll(t, filepath.Join(root, "salida"))
	if _, _, err := runCompose(t, "--cwd", filepath.Join(root, "src"), "--document", "requisitos-funcionales", "--delta", "cambio/delta.md", "--output", "salida/compuesto.md"); err != nil {
		t.Fatalf("runSDDArchiveComposeWithStderr() error = %v", err)
	}
	written, err := os.ReadFile(filepath.Join(root, "salida", "compuesto.md"))
	if err != nil {
		t.Fatalf("la salida relativa debe escribirse bajo el workspaceRoot: %v", err)
	}
	if string(written) != composeProfileComposed {
		t.Errorf("fichero de salida = %q", written)
	}
}

func TestRunSDDArchiveComposeDocumentWithSupersedeOptions(t *testing.T) {
	root := composeWorkspace(t, composeBaselineConfig, map[string]string{
		"docs/01_Requisitos_Funcionales.md": composeProfileDocument,
		"cambio/delta.md":                   "## REMOVED Requirements\n\n### Requirement: RF-100: Alta\n\n(Reason: sustituido por el nuevo alta)\n",
	})
	stdout, _, err := runCompose(t, "--cwd", root, "--document", "requisitos-funcionales", "--delta", "cambio/delta.md", "--supersede", "--superseded-requirements", "RF-101")
	if err != nil {
		t.Fatalf("runSDDArchiveComposeWithStderr() error = %v", err)
	}
	for _, want := range []string{
		"### RF-100: Alta [SUPERSEDED / DEPRECADO]\n> [!CAUTION]\n",
		"> Motivo: sustituido por el nuevo alta\n",
		"### RF-101: Baja [SUPERSEDED / DEPRECADO]\n> [!CAUTION]\n",
		"> Motivo: Superado por consolidación de arquitectura\n",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("falta %q en:\n%s", want, stdout)
		}
	}
}

func TestRunSDDArchiveComposeDocumentRefusesWithoutWriting(t *testing.T) {
	root := composeWorkspace(t, composeBaselineConfig, map[string]string{
		"docs/01_Requisitos_Funcionales.md": composeProfileDocument,
		"cambio/delta.md":                   "## MODIFIED Requirements\n\n### Requirement: RF-999: No existe\n\nCuerpo.\n",
	})
	output := filepath.Join(root, "salida.md")
	_, _, err := runCompose(t, "--cwd", root, "--document", "requisitos-funcionales", "--delta", "cambio/delta.md", "--output", output)
	if err == nil {
		t.Fatal("se esperaba un rechazo por el requisito inexistente")
	}
	for _, want := range []string{"requisitos-funcionales", "MODIFIED", "RF-999", "Continuación"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no menciona %q: %v", want, err)
		}
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Errorf("no debe escribirse la salida ante un rechazo: statErr = %v", statErr)
	}
}

func TestRunSDDArchiveComposeDocumentWithoutBaselineUsesTheNativeEngine(t *testing.T) {
	nativeConfig := "workspace:\n  name: ejemplo-nativo\n  topology: monorepo-embedded\n  specs_repository: \"\"\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n"
	canonical := "## Requirements\n\n" +
		"### Requirement: Unrelated Listing\n\nThe system MUST list widgets.\n\n" +
		"### Requirement: Widget Expiration\n\nThe system MUST expire widgets after 30 days.\n\n" +
		"### Requirement: Old Feature\n\nOld body.\n\n" +
		"### Requirement: Active Feature\n\nActive body.\n\n" +
		"### Requirement: Legacy Name\n\nLegacy body.\n"
	deltaText := "## ADDED Requirements\n\n### Requirement: Widget Tagging\n\nThe system MUST support tagging widgets.\n\n" +
		"## MODIFIED Requirements\n\n### Requirement: Widget Expiration\n\nThe system MUST expire widgets after 90 days.\n\n" +
		"## REMOVED Requirements\n\n### Requirement: Old Feature\n\n(Reason: replaced by the new flow)\n\n" +
		"## RENAMED Requirements\n\n### Requirement: Legacy Name → Modern Name\n\n(Reason: clearer name)\n"
	root := composeWorkspace(t, nativeConfig, map[string]string{
		"openspec/specs/widgets/spec.md":                canonical,
		"openspec/changes/inc-99/specs/widgets/spec.md": deltaText,
	})
	canonicalPath := filepath.Join(root, "openspec", "specs", "widgets", "spec.md")
	deltaPath := filepath.Join(root, "openspec", "changes", "inc-99", "specs", "widgets", "spec.md")

	for name, extra := range map[string][]string{
		"sin opciones":        nil,
		"con supersede":       {"--supersede"},
		"con superseded-reqs": {"--supersede", "--superseded-requirements", "Active Feature"},
	} {
		t.Run(name, func(t *testing.T) {
			native, _, err := runCompose(t, append([]string{"--canonical", canonicalPath, "--delta", deltaPath}, extra...)...)
			if err != nil {
				t.Fatalf("modo nativo: %v", err)
			}
			viaDocument, _, err := runCompose(t, append([]string{"--cwd", root, "--document", "widgets", "--delta", "openspec/changes/inc-99/specs/widgets/spec.md"}, extra...)...)
			if err != nil {
				t.Fatalf("modo documento: %v", err)
			}
			if viaDocument != native {
				t.Errorf("el modo documento sin baseline debe producir los mismos bytes que --canonical (R1)\n--- documento ---\n%s\n--- nativo ---\n%s", viaDocument, native)
			}
			if !strings.Contains(native, "### Requirement: Widget Tagging") || !strings.Contains(native, "Modern Name") {
				t.Errorf("la fixture debe ejercitar ADDED y RENAMED:\n%s", native)
			}
		})
	}
}

func TestRunSDDArchiveComposeDocumentRejectsProseDomains(t *testing.T) {
	root := composeWorkspace(t, composeBaselineConfig, map[string]string{
		"docs/03_Modelo_Datos.md": "# Modelo\n",
		"cambio/delta.md":         "## MODIFIED Requirements\n\n### Requirement: Entidad\n\nCuerpo.\n",
	})
	output := filepath.Join(root, "salida.md")
	_, _, err := runCompose(t, "--cwd", root, "--document", "modelo-datos", "--delta", "cambio/delta.md", "--output", output)
	if err == nil {
		t.Fatal("un dominio de prosa debe rechazarse")
	}
	for _, want := range []string{"modelo-datos", "prosa", "edición dirigida", "solo registro"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no menciona %q: %v", want, err)
		}
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Error("no debe escribirse nada")
	}
}

func TestRunSDDArchiveComposeDocumentRejectsUndeclaredDomains(t *testing.T) {
	root := composeWorkspace(t, composeBaselineConfig, map[string]string{
		"docs/01_Requisitos_Funcionales.md": composeProfileDocument,
		"cambio/delta.md":                   "## ADDED Requirements\n\n### Requirement: RF-9001: Nuevo\n\nCuerpo.\n",
	})
	_, _, err := runCompose(t, "--cwd", root, "--document", "nuevo-dominio", "--delta", "cambio/delta.md")
	if err == nil {
		t.Fatal("un dominio no declarado debe rechazarse")
	}
	for _, want := range []string{"nuevo-dominio", "baseline.documents", "declara el dominio", "retira la delta"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no menciona %q: %v", want, err)
		}
	}
	if _, statErr := os.Stat(filepath.Join(root, "openspec", "specs", "nuevo-dominio")); !os.IsNotExist(statErr) {
		t.Errorf("nunca debe crearse openspec/specs/<dominio>: statErr = %v", statErr)
	}
}

func TestRunSDDArchiveComposeDocumentAndCanonicalAreMutuallyExclusive(t *testing.T) {
	root := composeWorkspace(t, composeBaselineConfig, map[string]string{
		"docs/01_Requisitos_Funcionales.md": composeProfileDocument,
		"cambio/delta.md":                   composeProfileDelta,
	})
	output := filepath.Join(root, "salida.md")
	_, _, err := runCompose(t, "--cwd", root, "--document", "requisitos-funcionales", "--canonical", filepath.Join(root, "docs", "01_Requisitos_Funcionales.md"), "--delta", "cambio/delta.md", "--output", output)
	if err == nil || !strings.Contains(err.Error(), "excluyentes") {
		t.Fatalf("error = %v, se esperaba el rechazo por excluyentes", err)
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Error("no debe escribirse nada")
	}
}

func TestRunSDDArchiveComposeDocumentFlagIsRepeatableButComposesOneDomain(t *testing.T) {
	root := composeWorkspace(t, composeBaselineConfig, map[string]string{
		"docs/01_Requisitos_Funcionales.md": composeProfileDocument,
		"cambio/delta.md":                   composeProfileDelta,
	})
	_, _, err := runCompose(t, "--cwd", root, "--document", "requisitos-funcionales", "--document", "modelo-datos", "--delta", "cambio/delta.md")
	if err == nil {
		t.Fatal("varios --document con --delta deben rechazarse")
	}
	for _, want := range []string{"un solo dominio", "requisitos-funcionales", "modelo-datos"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no menciona %q: %v", want, err)
		}
	}
}

func TestRunSDDArchiveComposeDocumentRequiresADelta(t *testing.T) {
	root := composeWorkspace(t, composeBaselineConfig, nil)
	_, _, err := runCompose(t, "--cwd", root, "--document", "requisitos-funcionales")
	if err == nil || !strings.Contains(err.Error(), "--delta") {
		t.Fatalf("error = %v, se esperaba que pidiera --delta", err)
	}
}

func TestRunSDDArchiveComposeRequiresCanonicalOrDocument(t *testing.T) {
	_, _, err := runCompose(t, "--delta", "delta.md")
	if err == nil || !strings.Contains(err.Error(), "--canonical or --document") {
		t.Fatalf("error = %v, se esperaba que pidiera --canonical o --document", err)
	}
}

func TestRunSDDArchiveComposeDocumentRejectsDomainsThatAreNotNames(t *testing.T) {
	root := composeWorkspace(t, composeBaselineConfig, map[string]string{"cambio/delta.md": composeProfileDelta})
	for _, domain := range []string{"../fuera", "a/b", `a\b`, "..", "."} {
		t.Run(domain, func(t *testing.T) {
			_, _, err := runCompose(t, "--cwd", root, "--document", domain, "--delta", "cambio/delta.md")
			if err == nil || !strings.Contains(err.Error(), "no es un nombre de dominio válido") {
				t.Fatalf("error = %v, se esperaba el rechazo del dominio %q", err, domain)
			}
		})
	}
}

func TestRunSDDArchiveComposeDocumentReportsAnInvalidBaseline(t *testing.T) {
	invalid := strings.Replace(composeBaselineConfig, "level: 3", "level: 9", 1)
	root := composeWorkspace(t, invalid, map[string]string{"cambio/delta.md": composeProfileDelta})
	_, _, err := runCompose(t, "--cwd", root, "--document", "requisitos-funcionales", "--delta", "cambio/delta.md")
	if err == nil {
		t.Fatal("una baseline inválida debe rechazarse")
	}
	for _, want := range []string{"baseline", "no es válida", "level", "workspace validate"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no menciona %q: %v", want, err)
		}
	}
}

func TestRunSDDArchiveComposeCanonicalWithCwdResolvesRelativePaths(t *testing.T) {
	root := composeWorkspace(t, composeBaselineConfig, map[string]string{
		"canonico/spec.md": "## Requirements\n\n### Requirement: Existing\n\nBody.\n",
		"cambio/delta.md":  "## MODIFIED Requirements\n\n### Requirement: Existing\n\nNew body.\n",
	})
	stdout, _, err := runCompose(t, "--cwd", root, "--canonical", "canonico/spec.md", "--delta", "cambio/delta.md")
	if err != nil {
		t.Fatalf("runSDDArchiveComposeWithStderr() error = %v", err)
	}
	if !strings.Contains(stdout, "New body.") {
		t.Errorf("salida = %q", stdout)
	}
}

func TestRunSDDArchiveComposeHelpDocumentsTheDocumentMode(t *testing.T) {
	stdout, _, err := runCompose(t, "--help")
	if err != nil {
		t.Fatalf("--help error = %v", err)
	}
	for _, want := range []string{"--document", "--cwd", "excluyente con --canonical", "--supersede"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la ayuda no menciona %q:\n%s", want, stdout)
		}
	}
}

// composeAddedConfig declara dos documentos con perfil de requisito: el segundo
// omite los escenarios y comparte la familia `RF-` con el primero.
const composeAddedConfig = `workspace:
  name: ejemplo-compose
  topology: monorepo-embedded
  specs_repository: ""
roles:
  core:
    name: Core
    repositories:
      - path: .
baseline:
  documents:
    - domain: requisitos-funcionales
      path: docs/01_Requisitos_Funcionales.md
      requirement:
        heading: '^#+ (?P<id>RF-\d+): (?P<title>.+)$'
        level: 3
        render: '### {id}: {title}'
    - domain: requisitos-secundarios
      path: docs/02_Requisitos_Secundarios.md
      scenarios: omit
      requirement:
        heading: '^#+ (?P<id>RF-\d+): (?P<title>.+)$'
        level: 3
        render: '### {id}: {title}'
`

const composeSecondaryDocument = "# Secundarios\n\n### RF-050: Auditoría\n\nCuerpo de auditoría.\n"

const composeAddedDelta = "## ADDED Requirements\n\n" +
	"### Requirement: RF-9001: Reintento de cobro\n(Placement: after RF-100)\n\nEl sistema DEBE reintentar el cobro; ver RF-9001.\n"

const composeAddedComposed = "# Requisitos\n\n## Catálogo\n\n" +
	"### RF-100: Alta\n\nCuerpo original de alta.\n\n" +
	"### Notas de implantación\n\nSección contigua.\n\n" +
	"### RF-102: Reintento de cobro\n\nEl sistema DEBE reintentar el cobro; ver RF-102.\n\n" +
	"### RF-101: Baja\n\nCuerpo de baja.\n"

func TestRunSDDArchiveComposeDocumentAddsARequirementWithItsDefinitiveID(t *testing.T) {
	root := composeWorkspace(t, composeAddedConfig, map[string]string{
		"docs/01_Requisitos_Funcionales.md": composeProfileDocument,
		"docs/02_Requisitos_Secundarios.md": composeSecondaryDocument,
		"cambio/delta.md":                   composeAddedDelta,
	})
	stdout, stderr, err := runCompose(t, "--cwd", root, "--document", "requisitos-funcionales", "--delta", "cambio/delta.md")
	if err != nil {
		t.Fatalf("runSDDArchiveComposeWithStderr() error = %v", err)
	}
	if stdout != composeAddedComposed {
		t.Errorf("salida = %q\nse esperaba %q", stdout, composeAddedComposed)
	}
	if want := `Correspondencia de IDs: RF-9001 -> RF-102 (dominio "requisitos-funcionales")`; !strings.Contains(stderr, want) {
		t.Errorf("la salida de error debe imprimir la correspondencia %q: %q", want, stderr)
	}
	if strings.Contains(stdout, "RF-9001") {
		t.Error("el documento no debe conservar el ID provisional")
	}
	original, err := os.ReadFile(filepath.Join(root, "docs", "01_Requisitos_Funcionales.md"))
	if err != nil || string(original) != composeProfileDocument {
		t.Errorf("el comando no debe modificar el documento (err = %v)", err)
	}
}

func TestRunSDDArchiveComposeDocumentCalculatesTheIDMapOfTheWholeChange(t *testing.T) {
	changeDelta := func(domain string) string { return "openspec/changes/inc-99/specs/" + domain + "/spec.md" }
	secondaryDelta := "## ADDED Requirements\n\n### Requirement: RF-9002: Registro de accesos\n(Placement: after RF-050)\n\n" +
		"Depende de RF-9001.\n\n#### Scenario: acceso correcto\n\n- **CUANDO** entra\n"

	t.Run("la correspondencia incluye los demás dominios del cambio", func(t *testing.T) {
		root := composeWorkspace(t, composeAddedConfig, map[string]string{
			"docs/01_Requisitos_Funcionales.md":   composeProfileDocument,
			"docs/02_Requisitos_Secundarios.md":   composeSecondaryDocument,
			changeDelta("requisitos-funcionales"): composeAddedDelta,
			changeDelta("requisitos-secundarios"): secondaryDelta,
		})
		stdout, stderr, err := runCompose(t, "--cwd", root, "--document", "requisitos-funcionales", "--delta", changeDelta("requisitos-funcionales"))
		if err != nil {
			t.Fatalf("runSDDArchiveComposeWithStderr() error = %v", err)
		}
		if stdout != composeAddedComposed {
			t.Errorf("salida = %q", stdout)
		}
		for _, want := range []string{
			`Correspondencia de IDs: RF-9001 -> RF-102 (dominio "requisitos-funcionales")`,
			`Correspondencia de IDs: RF-9002 -> RF-051 (dominio "requisitos-secundarios")`,
		} {
			if !strings.Contains(stderr, want) {
				t.Errorf("falta %q en la salida de error: %q", want, stderr)
			}
		}

		// El otro dominio, con scenarios: omit, sustituye la referencia a un ID de
		// su cambio calculado sobre el cambio entero.
		secondary, _, err := runCompose(t, "--cwd", root, "--document", "requisitos-secundarios", "--delta", changeDelta("requisitos-secundarios"))
		if err != nil {
			t.Fatalf("runSDDArchiveComposeWithStderr() error = %v", err)
		}
		want := "# Secundarios\n\n### RF-050: Auditoría\n\nCuerpo de auditoría.\n### RF-051: Registro de accesos\n\nDepende de RF-102.\n"
		if secondary != want {
			t.Errorf("salida = %q\nse esperaba %q (referencia sustituida y escenarios omitidos)", secondary, want)
		}
	})

	t.Run("un ID provisional repetido entre dominios solo se detecta con el cambio entero", func(t *testing.T) {
		duplicated := strings.Replace(secondaryDelta, "RF-9002", "RF-9001", 1)
		files := map[string]string{
			"docs/01_Requisitos_Funcionales.md":   composeProfileDocument,
			"docs/02_Requisitos_Secundarios.md":   composeSecondaryDocument,
			changeDelta("requisitos-funcionales"): composeAddedDelta,
			changeDelta("requisitos-secundarios"): duplicated,
			"cambio/delta.md":                     composeAddedDelta,
		}
		root := composeWorkspace(t, composeAddedConfig, files)

		output := filepath.Join(root, "salida.md")
		_, _, err := runCompose(t, "--cwd", root, "--document", "requisitos-funcionales", "--delta", changeDelta("requisitos-funcionales"), "--output", output)
		if err == nil {
			t.Fatal("un ID provisional declarado en dos dominios del cambio debe rechazarse")
		}
		for _, want := range []string{"RF-9001", "dos requisitos ADDED", "requisitos-funcionales", "requisitos-secundarios", "Continuación"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("el error no menciona %q: %v", want, err)
			}
		}
		if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
			t.Error("no debe escribirse nada ante un rechazo")
		}

		// El mismo delta fuera de la carpeta del cambio se compone solo.
		if _, _, err := runCompose(t, "--cwd", root, "--document", "requisitos-funcionales", "--delta", "cambio/delta.md"); err != nil {
			t.Fatalf("el delta suelto solo calcula su propia correspondencia: %v", err)
		}
	})
}

func TestRunSDDArchiveComposeDocumentRejectsAnAddedWithoutPlacementOrWithAnUnknownAnchor(t *testing.T) {
	root := composeWorkspace(t, composeAddedConfig, map[string]string{
		"docs/01_Requisitos_Funcionales.md": composeProfileDocument,
		"docs/02_Requisitos_Secundarios.md": composeSecondaryDocument,
		"sin-ubicacion.md":                  "## ADDED Requirements\n\n### Requirement: RF-9001: Sin ubicación\n\nCuerpo.\n",
		"ancla-inexistente.md":              "## ADDED Requirements\n\n### Requirement: RF-9001: Sin ancla\n(Placement: after RF-999)\n\nCuerpo.\n",
	})
	for name, wants := range map[string][]string{
		"sin-ubicacion.md":     {"ADDED", "RF-9001", "ubicación", "explícita"},
		"ancla-inexistente.md": {"ADDED", "RF-999", "no contiene"},
	} {
		t.Run(name, func(t *testing.T) {
			output := filepath.Join(root, "salida.md")
			_, _, err := runCompose(t, "--cwd", root, "--document", "requisitos-funcionales", "--delta", name, "--output", output)
			if err == nil {
				t.Fatal("se esperaba un rechazo")
			}
			for _, want := range wants {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("el error no menciona %q: %v", want, err)
				}
			}
			if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
				t.Error("no debe escribirse nada ante un rechazo")
			}
		})
	}
}

func TestRunSDDArchiveComposeDocumentStopsOnAFamilyWithoutIDs(t *testing.T) {
	config := strings.Replace(composeAddedConfig, `'^#+ (?P<id>RF-\d+): (?P<title>.+)$'`, `'^#+ (?P<id>(?:RF|NFR)-\d+): (?P<title>.+)$'`, 1)
	root := composeWorkspace(t, config, map[string]string{
		"docs/01_Requisitos_Funcionales.md": composeProfileDocument,
		"docs/02_Requisitos_Secundarios.md": composeSecondaryDocument,
		"cambio/delta.md":                   "## ADDED Requirements\n\n### Requirement: NFR-9001: Rendimiento\n(Placement: after RF-100)\n\nCuerpo.\n",
	})
	_, _, err := runCompose(t, "--cwd", root, "--document", "requisitos-funcionales", "--delta", "cambio/delta.md")
	if err == nil {
		t.Fatal("una familia sin ningún ID debe detener la composición")
	}
	for _, want := range []string{"NFR-9001", `familia de IDs "NFR-"`, "siembra la numeración", "axiom sdd integrate check"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no menciona %q: %v", want, err)
		}
	}
}
