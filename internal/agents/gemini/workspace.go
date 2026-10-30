package gemini

import (
	"path/filepath"
)

// Distribución de proyecto de Gemini CLI (documentación oficial,
// google-gemini/gemini-cli docs/cli/gemini-md.md y docs/reference/configuration.md):
//
//   - GEMINI.md en la raíz del proyecto y en sus carpetas padre (y el global
//     ~/.gemini/GEMINI.md): es el contexto de proyecto que carga Gemini. Es un
//     fichero del equipo, normalmente versionado, y Axiom no lo toca.
//   - .gemini/{settings.json,skills,commands,agents}: configuración y recursos del
//     proyecto. Coinciden con lo que ya escribía Axiom.
//   - context.fileName en .gemini/settings.json: nombre o nombres de los ficheros
//     de contexto. Gemini une cada nombre a cada carpeta de búsqueda con path.join
//     (memoryDiscovery.ts, findUpwardGeminiFiles), de modo que un nombre con
//     separador (".gemini/AXIOM.md") carga ese fichero.
//
// Gemini no lee <proyecto>/.gemini/GEMINI.md como contexto de proyecto, que era donde
// Axiom escribía las instrucciones (orquestador ODD/SDD y persona) con --scope
// workspace.
const (
	// projectInstructionsFileName es el fichero propio de Axiom con las
	// instrucciones. Se evita GEMINI.md dentro de .gemini para no confundirlo con el
	// de la raíz.
	projectInstructionsFileName = "AXIOM.md"
	// contextFileEntry es el valor que se añade a context.fileName: relativo a la
	// carpeta de búsqueda, con barras normales.
	contextFileEntry = ".gemini/" + projectInstructionsFileName
	// defaultContextFileName es el nombre que Gemini carga cuando context.fileName no
	// está declarado. context.fileName del proyecto sustituye a la lista del usuario
	// (no se fusionan), así que hay que conservarlo para que el GEMINI.md de la raíz y
	// el global sigan cargándose.
	defaultContextFileName = "GEMINI.md"
)

// WorkspaceAdapter es el adaptador de Gemini CLI ligado a la raíz de un proyecto.
// Para esa raíz sitúa las instrucciones en un fichero propio de .gemini; para
// cualquier otra carpeta (la de usuario) da las rutas globales de siempre. Es un
// tipo aparte del adaptador global para que las capacidades que solo tienen
// sentido en el proyecto (registro en context.fileName y retirada del
// .gemini/GEMINI.md antiguo) no existan en la instalación global.
type WorkspaceAdapter struct {
	*Adapter
	root string
}

// WithWorkspace devuelve el adaptador ligado a la raíz de un proyecto.
func (a *Adapter) WithWorkspace(root string) *WorkspaceAdapter {
	return &WorkspaceAdapter{Adapter: a, root: root}
}

// inWorkspace indica si dir es la raíz del proyecto a la que está ligado el
// adaptador.
func (a *WorkspaceAdapter) inWorkspace(dir string) bool {
	return a.root != "" && dir != "" && filepath.Clean(dir) == filepath.Clean(a.root)
}

// SystemPromptFile es ~/.gemini/GEMINI.md en la carpeta de usuario. En el
// proyecto es el fichero propio de Axiom (.gemini/AXIOM.md), que se registra en
// context.fileName: el GEMINI.md de la raíz es del equipo.
func (a *WorkspaceAdapter) SystemPromptFile(dir string) string {
	if a.inWorkspace(dir) {
		return filepath.Join(dir, ".gemini", projectInstructionsFileName)
	}
	return a.Adapter.SystemPromptFile(dir)
}

// LegacySystemPromptFiles lista el .gemini/GEMINI.md que escribían las
// instalaciones workspace anteriores. Solo se retira si se prueba que es de Axiom.
func (a *WorkspaceAdapter) LegacySystemPromptFiles(dir string) []string {
	if !a.inWorkspace(dir) {
		return nil
	}
	return []string{filepath.Join(dir, ".gemini", defaultContextFileName)}
}

// WorkspaceContextFile indica dónde se registra el fichero de instrucciones de
// Axiom: el fichero de configuración que lo declara (.gemini/settings.json), la
// entrada que se añade a context.fileName y los nombres que se conservan cuando el
// proyecto no declara ninguno. ok es false fuera del proyecto.
func (a *WorkspaceAdapter) WorkspaceContextFile(dir string) (settingsPath, entry string, defaults []string, ok bool) {
	if !a.inWorkspace(dir) {
		return "", "", nil, false
	}
	return a.Adapter.SettingsPath(dir), contextFileEntry, []string{defaultContextFileName}, true
}
