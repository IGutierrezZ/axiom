package hub

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// Preferencias personales de worktree de un proyecto (INC-25, REQ-25.29, design D-08): el
// modo de trabajo y la raíz de worktrees. Viven en la entrada del proyecto de
// `~/.axiom/workspaces.json` y nunca en `axiom.yaml`. Este fichero trae la validación, la
// lectura (`LookupWorktreePrefs`) y la escritura (`SetWorktreePrefs`); el registro con
// preferencias (`RegisterWithWorktreePrefs`) está en manager.go.

// Los modos de trabajo que admite la preferencia (REQ-25.8): un worktree por rama, la propia
// carpeta del proyecto, o preguntar en cada cambio.
const (
	WorktreeModeWorktree = "worktree"
	WorktreeModeInPlace  = "in_place"
	WorktreeModeAsk      = "ask"
)

// WorktreePrefs son las preferencias de worktree de un proyecto. Un campo vacío es «sin
// preferencia»: rigen entonces los valores por defecto.
type WorktreePrefs struct {
	Mode string
	Root string
}

// ValidateWorktreePrefs comprueba unas preferencias: el modo, si lo hay, es uno de los tres
// valores (se distinguen mayúsculas) y la raíz, si la hay, es una ruta absoluta. Los mensajes
// están en castellano y dicen qué es válido.
func ValidateWorktreePrefs(prefs WorktreePrefs) error {
	switch prefs.Mode {
	case "", WorktreeModeWorktree, WorktreeModeInPlace, WorktreeModeAsk:
	default:
		return fmt.Errorf("el modo de trabajo «%s» no es válido: usa %s, %s o %s", prefs.Mode, WorktreeModeWorktree, WorktreeModeInPlace, WorktreeModeAsk)
	}
	if prefs.Root != "" && !filepath.IsAbs(prefs.Root) {
		return fmt.Errorf("la raíz de worktrees «%s» no es válida: debe ser una ruta absoluta", prefs.Root)
	}
	return nil
}

// LookupWorktreePrefs devuelve las preferencias efectivas del proyecto cuya carpeta maestra es
// projectPath: las guardadas en su entrada del hub y, para lo que no tiene, los valores por
// defecto (modo `worktree` y la raíz `<padre de la carpeta maestra>/<nombre>-wt`). Un proyecto
// que no está registrado, o un hub que todavía no existe, recibe los valores por defecto sin
// error (REQ-25.29).
//
// El proyecto se busca por identidad de directorio (pathidentity.SameDirectory), no por
// texto. stored dice si el hub guarda el modo del proyecto: el origen de ese modo es entonces
// el hub y, si no, el valor por defecto. Un hub que no se puede leer o un valor guardado
// inválido (REQ-25.29) es un error que nombra el fichero; no modifica nada.
func (m *Manager) LookupWorktreePrefs(projectPath string) (WorktreePrefs, bool, error) {
	cfg, err := m.Load()
	if err != nil {
		return WorktreePrefs{}, false, err
	}
	var saved WorktreePrefs
	for _, record := range cfg.Workspaces {
		if sameProject(record.Path, projectPath) {
			saved = WorktreePrefs{Mode: record.WorktreeMode, Root: record.WorktreeRoot}
			break
		}
	}
	if err := ValidateWorktreePrefs(saved); err != nil {
		return WorktreePrefs{}, false, fmt.Errorf("las preferencias de worktree guardadas en %s para %s: %w", m.configPath, projectPath, err)
	}

	effective := saved
	if effective.Mode == "" {
		effective.Mode = WorktreeModeWorktree
	}
	if effective.Root == "" {
		effective.Root = branch.DefaultWorktreeRoot(projectPath)
	}
	return effective, saved.Mode != "", nil
}

// SetWorktreePrefs guarda las preferencias de worktree del proyecto registrado que idOrPath
// nombra, por id o por ruta (sameProject), y devuelve su entrada actualizada. Sustituye ambos
// campos: un campo vacío es «sin preferencia» y se retira del JSON. Valida antes de leer el
// hub, de modo que un valor inválido no lo modifica (REQ-25.29); un proyecto que no está
// registrado es un error que lo dice y tampoco lo modifica.
func (m *Manager) SetWorktreePrefs(idOrPath string, p WorktreePrefs) (*WorkspaceRecord, error) {
	if err := ValidateWorktreePrefs(p); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, err := m.loadUnlocked()
	if err != nil {
		return nil, err
	}
	for i := range cfg.Workspaces {
		if cfg.Workspaces[i].ID != idOrPath && !sameProject(cfg.Workspaces[i].Path, idOrPath) {
			continue
		}
		cfg.Workspaces[i].WorktreeMode, cfg.Workspaces[i].WorktreeRoot = p.Mode, p.Root
		if err := m.saveUnlocked(cfg); err != nil {
			return nil, err
		}
		return &cfg.Workspaces[i], nil
	}
	return nil, fmt.Errorf("el proyecto «%s» no está registrado en %s", idOrPath, m.configPath)
}

// sameProject dice si dos rutas son la misma carpeta de proyecto: la misma carpeta existente
// por identidad (que cubre los nombres cortos 8.3 y los enlaces) o, si alguna no existe, la
// misma ruta limpia, sin distinguir mayúsculas en Windows.
func sameProject(a, b string) bool {
	if pathidentity.SameDirectory(a, b) {
		return true
	}
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
