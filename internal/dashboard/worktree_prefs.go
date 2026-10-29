package dashboard

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Preferencias personales de worktree en el dashboard (INC-25, REQ-3.1, REQ-25.29,
// REQ-25.33, design D-20). Viven en el hub (`~/.axiom/workspaces.json`) y nunca en `axiom.yaml`:
// ninguna ruta de este fichero escribe un fichero del proyecto.

// projectNotRegisteredError dice que el proyecto que el dashboard tiene abierto no está en el
// hub, de modo que no hay entrada donde leer ni guardar sus preferencias. El servidor lo
// responde con 404 y el mensaje ya trae la continuación ejecutable.
type projectNotRegisteredError struct{ path string }

func (e *projectNotRegisteredError) Error() string {
	return fmt.Sprintf("el proyecto activo %s no está registrado en el Hub, así que no tiene preferencias de worktree: regístralo con `axiom project add %s`", pathquote.Quote(e.path), pathquote.Quote(e.path))
}

// worktreePrefsOf devuelve las preferencias guardadas en la entrada del hub y las efectivas
// (REQ-25.29): para lo que no está guardado, el modo `worktree` y la raíz
// `<padre>/<nombre>-wt`. Se calcula sobre la propia entrada, sin releer el hub. Una entrada con
// un valor guardado inválido (editada a mano) no rompe el listado: rigen los valores por
// defecto, y las guardadas se devuelven tal cual para que el usuario vea qué hay en el fichero.
func worktreePrefsOf(rec hub.WorkspaceRecord) (saved, effective WorktreePrefsDTO) {
	saved = WorktreePrefsDTO{Mode: rec.WorktreeMode, Root: rec.WorktreeRoot}
	effective = saved
	if hub.ValidateWorktreePrefs(hub.WorktreePrefs{Mode: saved.Mode, Root: saved.Root}) != nil {
		effective = WorktreePrefsDTO{}
	}
	if effective.Mode == "" {
		effective.Mode = hub.WorktreeModeWorktree
	}
	if effective.Root == "" {
		effective.Root = branch.DefaultWorktreeRoot(rec.Path)
	}
	return saved, effective
}

func newProjectDTO(rec hub.WorkspaceRecord) ProjectDTO {
	saved, effective := worktreePrefsOf(rec)
	return ProjectDTO{WorkspaceRecord: rec, WorktreePrefs: saved, EffectiveWorktreePrefs: effective}
}

func newProjectWorktreePrefsDTO(rec hub.WorkspaceRecord) ProjectWorktreePrefsDTO {
	saved, effective := worktreePrefsOf(rec)
	return ProjectWorktreePrefsDTO{ID: rec.ID, Path: rec.Path, WorktreePrefs: saved, EffectiveWorktreePrefs: effective}
}

// activeProjectRecord busca en el hub la entrada del proyecto que el dashboard tiene abierto
// (la raíz del servicio), por identidad de directorio. Es la raíz del servicio y no el
// `active_workspace` del hub: `axiom ui` sirve la carpeta donde se lanzó aunque el hub
// tenga marcado otro proyecto, y las preferencias que se editan son las del proyecto que se ve.
func (s *Service) activeProjectRecord() (*hub.WorkspaceRecord, error) {
	if s.hubManager == nil {
		mgr, err := hub.NewManager("")
		if err != nil {
			return nil, err
		}
		s.hubManager = mgr
	}
	records, err := s.hubManager.List()
	if err != nil {
		return nil, err
	}
	root := s.getRootPath()
	for i := range records {
		if pathidentity.SameDirectory(records[i].Path, root) || strings.EqualFold(filepath.Clean(records[i].Path), filepath.Clean(root)) {
			return &records[i], nil
		}
	}
	return nil, &projectNotRegisteredError{path: root}
}

// GetActiveWorktreePrefs devuelve las preferencias de worktree, guardadas y efectivas, del
// proyecto activo.
func (s *Service) GetActiveWorktreePrefs() (*ProjectWorktreePrefsDTO, error) {
	rec, err := s.activeProjectRecord()
	if err != nil {
		return nil, err
	}
	dto := newProjectWorktreePrefsDTO(*rec)
	return &dto, nil
}

// SetActiveWorktreePrefs sustituye las preferencias de worktree del proyecto activo: un campo
// vacío retira la preferencia. Solo escribe el hub.
func (s *Service) SetActiveWorktreePrefs(p hub.WorktreePrefs) (*ProjectWorktreePrefsDTO, error) {
	rec, err := s.activeProjectRecord()
	if err != nil {
		return nil, err
	}
	updated, err := s.hubManager.SetWorktreePrefs(rec.ID, p)
	if err != nil {
		return nil, err
	}
	dto := newProjectWorktreePrefsDTO(*updated)
	return &dto, nil
}

// validateWorktreePrefs comprueba unas preferencias antes de que ninguna operación toque el
// hub, con los mensajes en castellano del hub (REQ-25.29).
func validateWorktreePrefs(mode, root string) error {
	return hub.ValidateWorktreePrefs(hub.WorktreePrefs{Mode: mode, Root: root})
}

// handleActiveWorktreePrefs sirve `GET|PUT /api/projects/active/worktree-prefs` (REQ-25.33).
// El PUT sustituye ambas preferencias; un valor inválido responde 400 sin cambiar el hub y un
// proyecto activo sin registrar responde 404.
func (s *Server) handleActiveWorktreePrefs(w http.ResponseWriter, r *http.Request) {
	var (
		dto *ProjectWorktreePrefsDTO
		err error
	)
	switch r.Method {
	case http.MethodGet:
		dto, err = s.service.GetActiveWorktreePrefs()
	case http.MethodPut:
		var req WorktreePrefsDTO
		if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
			s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
			return
		}
		if invalid := validateWorktreePrefs(req.Mode, req.Root); invalid != nil {
			s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": invalid.Error()})
			return
		}
		dto, err = s.service.SetActiveWorktreePrefs(hub.WorktreePrefs{Mode: req.Mode, Root: req.Root})
	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var notRegistered *projectNotRegisteredError
	switch {
	case errors.As(err, &notRegistered):
		s.respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	case err != nil:
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	default:
		s.respondJSON(w, http.StatusOK, dto)
	}
}
