package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/settings"
)

// Rutas /api/integrations* (D-05, D-06). Todas pasan por la comprobación local de Host y Origin de
// withHeaders y responden `Cache-Control: no-store`. Las POST exigen `application/json`, cuerpo
// acotado, campos conocidos y ningún parámetro en la query (el PAT nunca viaja en la URL); los
// errores son `{code, error, …}` en castellano y nunca derivan de un secreto.

const (
	// maxIntegrationsBody acota el cuerpo de las POST de integraciones: un PAT y unos nombres.
	maxIntegrationsBody = 8 << 10

	// defaultIntegrationBudget es el plazo de connect y discover; la escritura se amplía 5 s más.
	defaultIntegrationBudget = 45 * time.Second
	writeDeadlineSlack       = 5 * time.Second
)

// respondIntegrations escribe una respuesta de integraciones, siempre `no-store`: también cuando el
// handler se ejecuta sin la cadena de withHeaders.
func (s *Server) respondIntegrations(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Cache-Control", "no-store")
	s.respondJSON(w, status, data)
}

func (s *Server) respondIntegrationsError(w http.ResponseWriter, err *integrationsError) {
	s.respondIntegrations(w, err.Status, integrationsErrorBody{Code: err.Code, Error: err.Message, EnvVar: err.EnvVar, Candidates: err.Candidates})
}

func (s *Server) respondIntegrationsFailure(w http.ResponseWriter, status int, code, message string) {
	s.respondIntegrationsError(w, integrationsFail(status, code, message))
}

// requireIntegrationsMethod responde 405 y devuelve false cuando el método no es el admitido.
func (s *Server) requireIntegrationsMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	s.respondIntegrationsFailure(w, http.StatusMethodNotAllowed, "method_not_allowed", "método no permitido: esta ruta solo admite "+method)
	return false
}

// decodeIntegrationsBody valida una POST de integraciones y decodifica su cuerpo en dst. Responde
// el 400 que corresponda y devuelve false si algo falla; los mensajes son fijos y no repiten el
// contenido recibido.
func (s *Server) decodeIntegrationsBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	invalid := func(message string) bool {
		s.respondIntegrationsFailure(w, http.StatusBadRequest, "invalid_request", message)
		return false
	}
	if r.URL.RawQuery != "" {
		return invalid("la petición no admite parámetros en la URL: envía los datos en el cuerpo JSON")
	}
	if mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mediaType != "application/json" {
		return invalid("el cuerpo debe ser JSON: envía Content-Type: application/json")
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxIntegrationsBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return invalid("el cuerpo supera el límite de 8 KiB")
		}
		return invalid("JSON inválido: revisa los campos del cuerpo")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return invalid("JSON inválido: el cuerpo solo puede contener un objeto")
	}
	return true
}

// handleIntegrations sirve `GET /api/integrations`: el estado de los plugins sin revelar secretos.
func (s *Server) handleIntegrations(w http.ResponseWriter, r *http.Request) {
	if !s.requireIntegrationsMethod(w, r, http.MethodGet) {
		return
	}
	dto, err := s.service.GetIntegrations(r.Context())
	if err != nil {
		s.respondIntegrationsSettingsError(w, err)
		return
	}
	s.respondIntegrations(w, http.StatusOK, dto)
}

// respondIntegrationsSettingsError traduce un error de lectura de ajustes al contrato de errores de
// integraciones: `config_missing` es 404 y cualquier otro fallo 500 con su texto.
func (s *Server) respondIntegrationsSettingsError(w http.ResponseWriter, err error) {
	status, code, message := http.StatusInternalServerError, "settings_unreadable", err.Error()
	var typed *settings.Error
	if errors.As(err, &typed) {
		status, code, message = settingsStatus(typed.Code), string(typed.Code), typed.Message
	}
	s.respondIntegrationsFailure(w, status, code, message)
}

type secretRequest struct {
	Plugin string `json:"plugin"`
	Name   string `json:"name"`
	Value  string `json:"value"`
}

// secretDeleteRequest no tiene campo de valor: DisallowUnknownFields rechaza un PAT enviado por error.
type secretDeleteRequest struct {
	Plugin  string `json:"plugin"`
	Name    string `json:"name"`
	Confirm bool   `json:"confirm"`
}

// handleIntegrationsSecret sirve `POST /api/integrations/secret`: guarda el PAT en el almacén del
// sistema y responde solo la procedencia resultante.
func (s *Server) handleIntegrationsSecret(w http.ResponseWriter, r *http.Request) {
	if !s.requireIntegrationsMethod(w, r, http.MethodPost) {
		return
	}
	var req secretRequest
	if !s.decodeIntegrationsBody(w, r, &req) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), secretLookupBudget)
	defer cancel()
	resp, bad := s.service.SaveIntegrationSecret(ctx, req.Plugin, req.Name, req.Value)
	if bad != nil {
		s.respondIntegrationsError(w, bad)
		return
	}
	s.respondIntegrations(w, http.StatusOK, resp)
}

// handleIntegrationsSecretDelete sirve `POST /api/integrations/secret/delete`: exige `confirm: true`
// y es idempotente.
func (s *Server) handleIntegrationsSecretDelete(w http.ResponseWriter, r *http.Request) {
	if !s.requireIntegrationsMethod(w, r, http.MethodPost) {
		return
	}
	var req secretDeleteRequest
	if !s.decodeIntegrationsBody(w, r, &req) {
		return
	}
	if !req.Confirm {
		s.respondIntegrationsFailure(w, http.StatusBadRequest, "invalid_request",
			"falta la confirmación: el PAT es tuyo y lo comparten todos tus proyectos; envía confirm: true para eliminarlo")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), secretLookupBudget)
	defer cancel()
	resp, bad := s.service.DeleteIntegrationSecret(ctx, req.Plugin, req.Name)
	if bad != nil {
		s.respondIntegrationsError(w, bad)
		return
	}
	s.respondIntegrations(w, http.StatusOK, resp)
}

// integrationsContext acota la llamada al tracker con su presupuesto, ligado a la petición (si el
// cliente cierra, se cancela), y amplía el plazo de escritura del servidor por encima del presupuesto.
func (s *Server) integrationsContext(w http.ResponseWriter, r *http.Request, budget time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(r.Context(), budget)
	// httptest.ResponseRecorder no admite plazos (ErrNotSupported): se ignora.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(budget + writeDeadlineSlack))
	return ctx, cancel
}

func (s *Server) integrationsBusy(w http.ResponseWriter) {
	s.respondIntegrationsFailure(w, http.StatusConflict, "integration_busy",
		"ya hay una operación de integración en curso: espera a que termine")
}

// handleIntegrationsConnect sirve `POST /api/integrations/connect`.
func (s *Server) handleIntegrationsConnect(w http.ResponseWriter, r *http.Request) {
	if !s.requireIntegrationsMethod(w, r, http.MethodPost) {
		return
	}
	var req struct {
		Plugin string `json:"plugin"`
		User   string `json:"user"`
	}
	if !s.decodeIntegrationsBody(w, r, &req) {
		return
	}
	if !s.integrationsLock.TryLock() {
		s.integrationsBusy(w)
		return
	}
	defer s.integrationsLock.Unlock()
	ctx, cancel := s.integrationsContext(w, r, s.integrationBudget)
	defer cancel()
	resp, bad := s.service.ConnectIntegration(ctx, req.Plugin, req.User)
	if bad != nil {
		s.respondIntegrationsError(w, bad)
		return
	}
	s.respondIntegrations(w, http.StatusOK, resp)
}

// handleIntegrationsDiscover sirve `POST /api/integrations/discover`: no escribe axiom.yaml.
func (s *Server) handleIntegrationsDiscover(w http.ResponseWriter, r *http.Request) {
	if !s.requireIntegrationsMethod(w, r, http.MethodPost) {
		return
	}
	var req struct {
		Plugin string `json:"plugin"`
	}
	if !s.decodeIntegrationsBody(w, r, &req) {
		return
	}
	if !s.integrationsLock.TryLock() {
		s.integrationsBusy(w)
		return
	}
	defer s.integrationsLock.Unlock()
	ctx, cancel := s.integrationsContext(w, r, s.integrationBudget)
	defer cancel()
	resp, bad := s.service.DiscoverIntegration(ctx, req.Plugin)
	if bad != nil {
		s.respondIntegrationsError(w, bad)
		return
	}
	s.respondIntegrations(w, http.StatusOK, resp)
}
