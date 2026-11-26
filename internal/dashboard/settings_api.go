package dashboard

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/settings"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Ajustes del proyecto (C-06-e, RF-WEB-18, RF-WEB-19). Es el único camino del servidor que
// escribe axiom.yaml, y lo hace siempre mediante internal/settings: este fichero solo traduce
// entre HTTP y ese paquete.

// maxSettingsBody acota el cuerpo del PUT: un parche de ajustes son unos pocos cientos de bytes.
const maxSettingsBody = 1 << 20

// rejectNonLocal responde 403 y devuelve true cuando la petición no viene del propio dashboard
// (CSRF y DNS rebinding). Toda escritura lo llama la primera, antes de leer el cuerpo.
func (s *Server) rejectNonLocal(w http.ResponseWriter, r *http.Request) bool {
	if s.isLocalRequest(r) {
		return false
	}
	s.respondJSON(w, http.StatusForbidden, map[string]string{"error": "Petición rechazada: solo se admiten peticiones del propio dashboard"})
	return true
}

// settingsStatus es la correspondencia única de un código tipado de settings con su estado HTTP.
func settingsStatus(code settings.Code) int {
	switch code {
	case settings.CodeInvalidRequest:
		return http.StatusBadRequest
	case settings.CodeConfigMissing:
		return http.StatusNotFound
	case settings.CodeRevisionMismatch:
		return http.StatusConflict
	case settings.CodeUnsupportedDocument, settings.CodeUnknownKey, settings.CodeKeyNotEditable,
		settings.CodeKeyNotApplicable, settings.CodeInvalidValue, settings.CodeValidationFailed:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

// respondSettingsError escribe el cuerpo `{code, error, violations?}` de un error de ajustes.
func (s *Server) respondSettingsError(w http.ResponseWriter, err error) {
	var typed *settings.Error
	if errors.As(err, &typed) {
		s.respondJSON(w, settingsStatus(typed.Code), settingsErrorBody{Code: string(typed.Code), Error: typed.Message, Violations: typed.Violations})
		return
	}
	s.respondJSON(w, http.StatusInternalServerError, settingsErrorBody{Error: err.Error()})
}

// handleSettings sirve `GET|PUT /api/settings`. El GET devuelve el catálogo con los valores y la
// revisión; el PUT aplica un lote de cambios sobre esa revisión. POST, PATCH y DELETE responden 405.
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		dto, err := s.service.GetSettings()
		if err != nil {
			s.respondSettingsError(w, err)
			return
		}
		s.respondJSON(w, http.StatusOK, dto)
	case http.MethodPut:
		if s.rejectNonLocal(w, r) {
			return
		}
		var req SettingsPutRequest
		r.Body = http.MaxBytesReader(w, r.Body, maxSettingsBody)
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.respondJSON(w, http.StatusBadRequest, settingsErrorBody{Code: string(settings.CodeInvalidRequest), Error: "JSON inválido: el cuerpo debe ser {\"revision\": \"…\", \"changes\": {…}}"})
			return
		}
		resp, err := s.service.PutSettings(req)
		if err != nil {
			s.respondSettingsError(w, err)
			return
		}
		s.respondJSON(w, http.StatusOK, resp)
	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

// settingsLocation resuelve el axiom.yaml efectivo del proyecto con workspace.Locate: en
// multirepo es el del repositorio de specs apuntado, nunca uno inventado en la raíz del código.
func (s *Service) settingsLocation() (workspace.Location, error) {
	loc, err := workspace.Locate(s.getRootPath())
	if err != nil || loc.ConfigPath == "" {
		message := "el proyecto no tiene un axiom.yaml efectivo: crea uno en la carpeta del proyecto"
		if err != nil {
			message += " (" + err.Error() + ")"
		}
		return workspace.Location{}, &settings.Error{Code: settings.CodeConfigMissing, Message: message}
	}
	return loc, nil
}

// GetSettings devuelve el catálogo de ajustes del proyecto con los valores tal como están escritos.
func (s *Service) GetSettings() (*SettingsDTO, error) {
	loc, err := s.settingsLocation()
	if err != nil {
		return nil, err
	}
	return readSettingsDTO(loc.ConfigPath)
}

func readSettingsDTO(configPath string) (*SettingsDTO, error) {
	snap, err := settings.Read(configPath)
	if err != nil {
		return nil, err
	}
	dto := &SettingsDTO{
		ConfigPath:         snap.ConfigPath,
		Revision:           snap.Revision,
		Topology:           string(snap.Topology),
		ISO:                snap.ISO,
		Writable:           snap.Writable,
		WriteBlockedReason: snap.WriteBlockedReason,
		DocumentErrors:     append([]string{}, snap.DocumentErrors...),
		Entries:            make([]SettingDTO, 0, len(snap.Entries)),
	}
	for _, e := range snap.Entries {
		dto.Entries = append(dto.Entries, SettingDTO{
			Key: e.Key, Section: e.Section, Label: e.Label, Description: e.Description, Effect: e.Effect,
			Type: string(e.Type), Allowed: e.Allowed, Topologies: e.Topologies, Editable: e.Editable,
			ReadOnlyReason: e.ReadOnlyReason, Required: e.Required, Default: e.Default, Declared: e.Declared, Value: e.Value,
			Plugin: e.Plugin, Format: e.Format, Min: e.Min, Max: e.Max,
		})
	}
	return dto, nil
}

// PutSettings aplica un lote de cambios sobre el axiom.yaml del proyecto, bajo settingsMu.
func (s *Service) PutSettings(req SettingsPutRequest) (*SettingsPutResponse, error) {
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	loc, err := s.settingsLocation()
	if err != nil {
		return nil, err
	}
	result, err := settings.Write(settings.WriteRequest{
		ConfigPath:       loc.ConfigPath,
		Master:           loc.Master,
		ExpectedRevision: req.Revision,
		Patch:            settings.Patch(req.Changes),
	})
	if err != nil {
		return nil, err
	}
	dto, err := readSettingsDTO(loc.ConfigPath)
	if err != nil {
		return nil, err
	}
	return &SettingsPutResponse{Settings: dto, Changed: result.Changed, Warnings: append([]string{}, result.Warnings...)}, nil
}

// handleSettingsPersonal sirve `GET|PUT /api/settings/personal` (RF-WEB-20): la preferencia
// personal de autofetch del proyecto activo, guardada en el hub. El PUT exige una petición del
// propio dashboard y un cuerpo `{"autofetch": true|false}`; un JSON inválido, un valor no booleano
// o la clave ausente responden 400 sin guardar nada. Un proyecto sin registrar responde 404.
func (s *Server) handleSettingsPersonal(w http.ResponseWriter, r *http.Request) {
	var (
		dto *PersonalSettingsDTO
		err error
	)
	switch r.Method {
	case http.MethodGet:
		dto, err = s.service.GetPersonalSettings()
	case http.MethodPut:
		if s.rejectNonLocal(w, r) {
			return
		}
		var req struct {
			Autofetch *bool `json:"autofetch"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxSettingsBody)
		if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
			s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido: el cuerpo debe ser {\"autofetch\": true} o {\"autofetch\": false}"})
			return
		}
		if req.Autofetch == nil {
			s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "falta la clave 'autofetch': envía true o false"})
			return
		}
		dto, err = s.service.SetPersonalAutofetch(*req.Autofetch)
	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var notRegistered *projectNotRegisteredError
	switch {
	case errors.As(err, &notRegistered) || errors.Is(err, hub.ErrProjectNotRegistered):
		s.respondJSON(w, http.StatusNotFound, map[string]string{"error": (&projectNotRegisteredError{path: s.service.getRootPath()}).Error()})
	case err != nil:
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	default:
		s.respondJSON(w, http.StatusOK, dto)
	}
}

// GetPersonalSettings devuelve la preferencia de autofetch del proyecto activo.
func (s *Service) GetPersonalSettings() (*PersonalSettingsDTO, error) {
	mgr, err := s.hub()
	if err != nil {
		return nil, err
	}
	root := s.getRootPath()
	enabled, registered, err := mgr.LookupAutofetch(root)
	if err != nil {
		return nil, err
	}
	if !registered {
		return nil, &projectNotRegisteredError{path: root}
	}
	return &PersonalSettingsDTO{Autofetch: enabled, Path: root}, nil
}

// SetPersonalAutofetch guarda la preferencia de autofetch del proyecto activo en el hub. Solo
// escribe el hub: ningún fichero del proyecto cambia.
func (s *Service) SetPersonalAutofetch(enabled bool) (*PersonalSettingsDTO, error) {
	mgr, err := s.hub()
	if err != nil {
		return nil, err
	}
	root := s.getRootPath()
	if _, _, err := mgr.SetAutofetch(root, enabled); err != nil {
		return nil, err
	}
	return &PersonalSettingsDTO{Autofetch: enabled, Path: root}, nil
}
