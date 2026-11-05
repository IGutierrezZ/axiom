package cli

import (
	"os"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/state"
	"github.com/IGutierrezZ/axiom/v3/internal/statecoord"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

func withInstallStateLock(homeDir string, operation func() error) error {
	return statecoord.WithLock(homeDir, operation)
}

// withStateStoreLock serializa un ciclo leer-modificar-escribir sobre el almacén de
// estado: el global comparte cerrojo con withInstallStateLock y cada proyecto tiene
// el suyo.
func withStateStoreLock(store state.Store, operation func() error) error {
	return statecoord.WithStoreLock(store, operation)
}

// StateStoreFor devuelve el almacén del estado de instalación que corresponde al
// ámbito (decisión D2): con --scope workspace, el del proyecto en
// `<raíz>/.axiom/state.json`; con --scope global, el de usuario de siempre. Sin raíz
// de workspace resuelta cae al global, porque no hay dónde escribir el del proyecto.
func StateStoreFor(scope InstallScope, homeDir, workspaceDir string) state.Store {
	if scope == ScopeWorkspace && strings.TrimSpace(workspaceDir) != "" {
		return state.ProjectStore(homeDir, workspaceDir)
	}
	return state.GlobalStore(homeDir)
}

// EffectiveStateStore elige el almacén para quien lee estado sin conocer el ámbito
// (doctor, revisión): el del proyecto de la carpeta actual si ya tiene `.axiom/state.json`
// y, si no, el global. Nunca crea nada.
func EffectiveStateStore(homeDir string) state.Store {
	store, _ := effectiveStateStoreWithRoot(homeDir)
	return store
}

// effectiveStateStoreWithRoot es EffectiveStateStore y, además, la raíz del proyecto
// cuando el almacén efectivo es el suyo ("" si es el global): quien necesita leer
// también la configuración del proyecto (doctor) no repite la resolución.
func effectiveStateStoreWithRoot(homeDir string) (state.Store, string) {
	cwd, err := os.Getwd()
	if err != nil {
		return state.GlobalStore(homeDir), ""
	}
	root, _ := workspace.ResolveRoot(cwd, homeDir)
	return effectiveStateStoreAt(homeDir, root)
}

// effectiveStateStoreAt es effectiveStateStoreWithRoot con la raíz del proyecto ya
// resuelta por quien llama, sin tocar el directorio actual del proceso: el del
// proyecto si `<raíz>/.axiom/state.json` existe y, si no (o con la raíz vacía), el
// global.
func effectiveStateStoreAt(homeDir, root string) (state.Store, string) {
	if strings.TrimSpace(root) == "" {
		return state.GlobalStore(homeDir), ""
	}
	if info, statErr := os.Stat(state.ProjectPath(root)); statErr == nil && info.Mode().IsRegular() {
		return state.ProjectStore(homeDir, root), root
	}
	return state.GlobalStore(homeDir), ""
}
