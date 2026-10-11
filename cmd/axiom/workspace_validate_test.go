package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const (
	validateEmbeddedBase = "workspace:\n  name: ejemplo-embebido\n  topology: monorepo-embedded\n  specs_repository: .\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n"
	validateBaseline     = "baseline:\n  documents:\n    - domain: modelo-de-datos\n      path: 03_Modelo_Datos.md\n"
	validateContext      = "technical_context:\n  maintained:\n    entry: context/TECHNICAL_CONTEXT.md\n    paths: [context/]\n  reference:\n    paths: [00_Glosario.md]\n"
)

// validateWorkspace monta un proyecto embebido con la declaración dada, el
// documento de la baseline y, si config no está vacío, openspec/config.yaml.
func validateWorkspace(t *testing.T, blocks, openSpecConfig string) string {
	t.Helper()
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), validateEmbeddedBase+blocks)
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "03_Modelo_Datos.md"), "# modelo\n")
	if openSpecConfig != "" {
		workspacetest.WriteFile(t, filepath.Join(ws.Home, "config.yaml"), openSpecConfig)
	}
	return ws.Master
}

func wantOutput(t *testing.T, output string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(output, want) {
			t.Errorf("la salida no contiene %q:\n%s", want, output)
		}
	}
}

// workspace-topology REQ-1.2 y REQ-24.17: una declaración válida y compatible
// con el almacén finaliza con COMPLIANT y código 0.
func TestWorkspaceValidateAcceptsAValidDeclaration(t *testing.T) {
	for name, openSpecConfig := range map[string]string{
		"sin almacen declarado": "",
		"almacen openspec":      "artifact_store: openspec\n",
		"almacen hybrid":        "artifact_store: hybrid\n",
	} {
		t.Run(name, func(t *testing.T) {
			master := validateWorkspace(t, validateBaseline+validateContext, openSpecConfig)
			workspacetest.WriteFile(t, filepath.Join(master, "context", "TECHNICAL_CONTEXT.md"), "# ctx\n")
			workspacetest.WriteFile(t, filepath.Join(master, "00_Glosario.md"), "# glosario\n")
			output, code := runAxiom(t, "workspace", "validate", "--path", master)
			if code != 0 {
				t.Fatalf("workspace validate salió con %d:\n%s", code, output)
			}
			wantOutput(t, output, "COMPLIANT")
			if strings.Contains(output, "Advertencias") {
				t.Errorf("sin rutas inexistentes no debe haber advertencias:\n%s", output)
			}
		})
	}
}

// REQ-24.27 y workspace-topology REQ-1.2: una ruta de contexto inexistente solo
// avisa, y las advertencias se imprimen también con COMPLIANT.
func TestWorkspaceValidatePrintsWarningsWithCompliant(t *testing.T) {
	master := validateWorkspace(t, validateContext, "")
	output, code := runAxiom(t, "workspace", "validate", "--path", master)
	if code != 0 {
		t.Fatalf("una ruta inexistente no altera el código de salida; salió con %d:\n%s", code, output)
	}
	wantOutput(t, output, "Advertencias:", "'context/' de 'technical_context.maintained.paths' no existe todavía", "'00_Glosario.md'", "COMPLIANT")
	if strings.Contains(output, "NON-COMPLIANT") {
		t.Errorf("el resultado debe ser conforme:\n%s", output)
	}
	if strings.Index(output, "Advertencias:") > strings.Index(output, "Resultado: COMPLIANT") {
		t.Errorf("las advertencias van antes del resultado:\n%s", output)
	}
}

// REQ-24.14 y REQ-24.17: un error por cada ruta inválida, con código 1.
func TestWorkspaceValidateReportsOneErrorPerInvalidPath(t *testing.T) {
	blocks := "baseline:\n  documents:\n    - domain: uno\n      path: ../fuera.md\n    - domain: dos\n      path: /abs/dos.md\n    - domain: tres\n      path: tres.md\n"
	master := validateWorkspace(t, blocks, "")
	output, code := runAxiom(t, "workspace", "validate", "--path", master)
	if code != 1 {
		t.Fatalf("salió con %d, se esperaba 1:\n%s", code, output)
	}
	wantOutput(t, output,
		"No se pudo cargar la configuración de Axiom",
		"'../fuera.md' sale de workspaceRoot",
		"'/abs/dos.md'",
		"debe ser relativa a workspaceRoot",
	)
	lines := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "el dominio 'uno'") || strings.Contains(line, "el dominio 'dos'") {
			lines++
		}
	}
	if lines != 2 {
		t.Errorf("se esperaba una línea por ruta inválida, hubo %d:\n%s", lines, output)
	}
}

// REQ-24.17: una baseline sin documentos o con un perfil sin grupos finaliza con 1.
func TestWorkspaceValidateRejectsAnInvalidBaseline(t *testing.T) {
	for name, tc := range map[string]struct {
		blocks string
		wants  []string
	}{
		"sin documentos":         {"baseline: {}\n", []string{"la baseline declarada requiere al menos un documento"}},
		"clave desconocida":      {"baseline:\n  documentss: []\n", []string{"clave desconocida 'documentss'", "claves admitidas: documents"}},
		"perfil sin los grupos":  {"baseline:\n  documents:\n    - domain: rf\n      path: rf.md\n      requirement:\n        heading: '^### (RF-\\d+): (.+)$'\n        level: 3\n        render: '### {id}: {title}'\n", []string{"el dominio 'rf'", "los grupos con nombre 'id' y 'title'"}},
		"identidad desconocida":  {"baseline:\n  documents:\n    - domain: rf\n      path: rf.md\n      requirement:\n        heading: '^### (?P<id>RF-\\d+): (?P<title>.+)$'\n        level: 3\n        identity: nombre\n        render: '### {id}: {title}'\n", []string{"el dominio 'rf'", "'id' o 'text', no 'nombre'"}},
		"contexto sin rutas":     {"technical_context:\n  maintained:\n    paths: []\n", []string{"requiere al menos una ruta"}},
		"modo desconocido":       {"technical_context:\n  readonly:\n    paths: [a.md]\n", []string{"clave desconocida 'readonly'", "claves admitidas: maintained, reference"}},
		"nivel con include":      {"baseline:\n  documents:\n    - domain: rf\n      path: rf.md\n      requirement:\n        heading: '^#### (?P<id>RF-\\d+): (?P<title>.+)$'\n        level: 4\n        render: '#### {id}: {title}'\n", []string{"el dominio 'rf'", "'scenarios: omit' o un 'level' menor o igual que 3"}},
		"baseline y contexto":    {"baseline: {}\ntechnical_context: {}\n", []string{"la baseline declarada requiere", "requiere al menos el modo"}},
		"ruta compartida (modo)": {"baseline:\n  documents:\n    - domain: a\n      path: a.md\ntechnical_context:\n  reference:\n    paths: [a.md]\n", []string{"solapa con el documento 'a.md'"}},
	} {
		t.Run(name, func(t *testing.T) {
			master := validateWorkspace(t, tc.blocks, "")
			output, code := runAxiom(t, "workspace", "validate", "--path", master)
			if code != 1 {
				t.Fatalf("salió con %d, se esperaba 1:\n%s", code, output)
			}
			wantOutput(t, output, tc.wants...)
		})
	}
}

// REQ-24.14: un documento declarado que no existe es un error nombrando el
// dominio y la ruta; los demás documentos válidos no lo ocultan.
func TestWorkspaceValidateReportsAMissingDocument(t *testing.T) {
	blocks := "baseline:\n  documents:\n    - domain: modelo-de-datos\n      path: 03_Modelo_Datos.md\n    - domain: ausente\n      path: docs/ausente.md\n"
	master := validateWorkspace(t, blocks, "")
	output, code := runAxiom(t, "workspace", "validate", "--path", master)
	if code != 1 {
		t.Fatalf("salió con %d, se esperaba 1:\n%s", code, output)
	}
	wantOutput(t, output, "NO CONFORME", "el documento 'docs/ausente.md' del dominio 'ausente' no existe", "NON-COMPLIANT")
	if strings.Contains(output, "dominio 'modelo-de-datos'") {
		t.Errorf("el documento existente no debe reportarse:\n%s", output)
	}
}

// REQ-24.62: con almacén declarado engram o none, un error por declaración
// afectada con la continuación, y código 1.
func TestWorkspaceValidateRejectsAFileDeclarationWithAFilelessStore(t *testing.T) {
	for _, store := range []string{"engram", "none"} {
		t.Run(store, func(t *testing.T) {
			master := validateWorkspace(t, validateBaseline+"technical_context:\n  maintained:\n    paths: [context/]\n", "artifact_store: "+store+"\n")
			workspacetest.WriteFile(t, filepath.Join(master, "context", "a.md"), "# ctx\n")
			output, code := runAxiom(t, "workspace", "validate", "--path", master)
			if code != 1 {
				t.Fatalf("salió con %d, se esperaba 1:\n%s", code, output)
			}
			wantOutput(t, output,
				"la declaración 'baseline' exige un almacén de artefactos de ficheros",
				"la declaración 'technical_context.maintained' exige un almacén de artefactos de ficheros",
				"artifact_store '"+store+"'",
				"Declara 'artifact_store: openspec' o 'hybrid' en openspec/config.yaml",
			)
		})
	}

	t.Run("solo contexto de referencia no exige almacen de ficheros", func(t *testing.T) {
		master := validateWorkspace(t, "technical_context:\n  reference:\n    paths: [00_Glosario.md]\n", "artifact_store: engram\n")
		workspacetest.WriteFile(t, filepath.Join(master, "00_Glosario.md"), "# glosario\n")
		output, code := runAxiom(t, "workspace", "validate", "--path", master)
		if code != 0 {
			t.Fatalf("salió con %d, se esperaba 0:\n%s", code, output)
		}
	})
}

// El espacio canónico: --path es la carpeta maestra, los documentos se resuelven
// contra el repositorio de especificaciones que apunta el puntero.
func TestWorkspaceValidateResolvesDeclarationsAgainstTheSpecsRepository(t *testing.T) {
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"),
		"workspace:\n  name: ejemplo-canonico\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  backend:\n    name: Backend\n    repositories:\n      - path: repo-backend\n"+
			validateBaseline+"technical_context:\n  maintained:\n    paths: [../repo-backend/docs/]\n")
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "03_Modelo_Datos.md"), "# modelo\n")

	output, code := runAxiom(t, "workspace", "validate", "--path", ws.Master)
	if code != 0 {
		t.Fatalf("salió con %d:\n%s", code, output)
	}
	wantOutput(t, output, "COMPLIANT", "Advertencias:", "'../repo-backend/docs/' de 'technical_context.maintained.paths'")

	workspacetest.WriteFile(t, filepath.Join(ws.Backend, "docs", "arquitectura.md"), "# arq\n")
	output, code = runAxiom(t, "workspace", "validate", "--path", ws.Master)
	if code != 0 || strings.Contains(output, "Advertencias") {
		t.Fatalf("con la ruta mantenida existente no debe haber advertencias (salió con %d):\n%s", code, output)
	}
}

// Sin bloques nuevos el comando se comporta como antes.
func TestWorkspaceValidateWithoutDeclarationsIsUnchanged(t *testing.T) {
	master := validateWorkspace(t, "", "artifact_store: none\n")
	output, code := runAxiom(t, "workspace", "validate", "--path", master)
	if code != 0 {
		t.Fatalf("salió con %d:\n%s", code, output)
	}
	wantOutput(t, output, "COMPLIANT")
	if strings.Contains(output, "Advertencias") {
		t.Errorf("sin declaraciones no hay advertencias:\n%s", output)
	}
}
