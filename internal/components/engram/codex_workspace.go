package engram

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
)

// CodexWorkspaceNoticeWriter recibe los avisos de la instalación de Codex en un
// proyecto. Es una variable para que las pruebas los capturen.
var CodexWorkspaceNoticeWriter io.Writer = os.Stderr

const (
	// codexWorkspaceCompactFile es el fichero del prompt de compactación en la carpeta
	// .codex del proyecto. En la configuración de proyecto de Codex las rutas
	// relativas se resuelven respecto a esa carpeta, así que el valor de la clave es
	// el nombre del fichero.
	codexWorkspaceCompactFile = "engram-compact-prompt.md"
	// codexCompactPromptKey es la clave de config.toml que apunta a ese fichero.
	codexCompactPromptKey = "experimental_compact_prompt_file"
)

// codexWorkspaceConfigPath devuelve el fichero de configuración de proyecto de
// Codex cuando adapter es el adaptador ligado a la raíz dir. ok es false con la
// instalación global, que sigue su propio camino.
func codexWorkspaceConfigPath(adapter agents.Adapter, dir string) (configPath string, ok bool) {
	provider, isInline := adapter.(agents.WorkspaceInlineInstructionsProvider)
	if !isInline {
		return "", false
	}
	configPath, _, ok = provider.WorkspaceInlineInstructions(dir)
	return configPath, ok && configPath != ""
}

// injectCodexWorkspaceConfig escribe la configuración de Engram de Codex en el
// proyecto: el MCP, las claves de multiagente, el prompt de compactación y, si el
// usuario eligió un modelo para el orquestador, ese modelo. handled es false con la
// instalación global.
//
// Respecto a la instalación global:
//   - No se escribe model_instructions_file. Esa clave sustituye las instrucciones
//     base de Codex en lugar de añadirse a ellas, y el protocolo de memoria va en
//     developer_instructions, con el resto de instrucciones de Axiom.
//   - No se escriben los ficheros de perfil <carril>.config.toml: Codex ignora
//     profiles en la configuración de proyecto.
//   - No se valida el ejecutable de Codex: solo servía para decidir si había que
//     escribir esos perfiles.
//
// Un config.toml que no es TOML válido se rechaza antes de escribir: es un fichero
// del usuario, normalmente versionado, y no hay vuelta atrás si se reescribe.
// Si el usuario ya declara su propio experimental_compact_prompt_file, se respeta.
func injectCodexWorkspaceConfig(root string, adapter agents.Adapter, opts InjectOptions) (changed bool, files []string, handled bool, err error) {
	configPath, ok := codexWorkspaceConfigPath(adapter, root)
	if !ok {
		return false, nil, false, nil
	}
	existing, err := readFileOrEmpty(configPath)
	if err != nil {
		return false, nil, true, err
	}
	if parseErr := filemerge.ValidateTOML(existing); parseErr != nil {
		return false, nil, true, fmt.Errorf("Axiom no modifica %s: no es TOML válido: %w", configPath, parseErr)
	}

	compactPath := filepath.Join(filepath.Dir(configPath), codexWorkspaceCompactFile)
	compactWrite, err := filemerge.WriteFileAtomic(compactPath, []byte(codexCompact()), 0o644)
	if err != nil {
		return false, nil, true, fmt.Errorf("write codex %s: %w", codexWorkspaceCompactFile, err)
	}
	changed = compactWrite.Changed
	files = append(files, compactPath)

	content := filemerge.UpsertTOMLTableKey(existing, "features", "multi_agent", "true")
	content = filemerge.UpsertTOMLTableKey(content, "agents", "max_threads", "4")
	content = filemerge.UpsertTOMLTableKey(content, "agents", "max_depth", "2")

	value, declared, isString, err := filemerge.TopLevelTOMLString(content, codexCompactPromptKey)
	if err != nil {
		return false, nil, true, err
	}
	switch {
	case !declared || (isString && value == codexWorkspaceCompactFile):
		content = filemerge.UpsertTopLevelTOMLValue(content, codexCompactPromptKey, filemerge.FormatTOMLString(codexWorkspaceCompactFile))
	default:
		fmt.Fprintf(CodexWorkspaceNoticeWriter, "Aviso: %s ya declara %s con otro valor; Axiom no lo modifica y el prompt de compactación de Engram no se usa en este proyecto.\n", configPath, codexCompactPromptKey)
	}
	if opts.CodexOrchestratorAssignment != nil {
		content = filemerge.UpsertTopLevelTOMLValue(content, "model", filemerge.FormatTOMLString(opts.CodexOrchestratorAssignment.Model))
		content = filemerge.UpsertTopLevelTOMLValue(content, "model_reasoning_effort", filemerge.FormatTOMLString(string(opts.CodexOrchestratorAssignment.Effort)))
	}
	content = filemerge.UpsertCodexEngramBlock(content, stableEngramCommandForMergedConfig(configPath, adapter.Agent()))

	tomlWrite, err := filemerge.WriteFileAtomic(configPath, []byte(content), 0o644)
	if err != nil {
		return false, nil, true, err
	}
	changed = changed || tomlWrite.Changed
	files = append(files, configPath)
	return changed, files, true, nil
}

// injectCodexWorkspacePassiveCapture añade al fichero de instrucciones de Axiom la
// sección de captura pasiva. En la instalación global Codex la recibe dentro de
// engram-instructions.md (model_instructions_file); en el proyecto no hay ese
// fichero, así que viaja con el resto de instrucciones.
func injectCodexWorkspacePassiveCapture(promptPath string) (bool, error) {
	existing, err := readFileOrEmpty(promptPath)
	if err != nil {
		return false, err
	}
	updated := filemerge.InjectMarkdownSection(existing, filemerge.EngramPassiveCaptureSectionID, protocolPassiveCapture())
	write, err := filemerge.WriteFileAtomic(promptPath, []byte(updated), 0o644)
	if err != nil {
		return false, err
	}
	return write.Changed, nil
}
