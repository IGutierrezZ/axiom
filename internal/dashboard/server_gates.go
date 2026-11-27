package dashboard

import "net/http"

// handleGateStop es `GET /api/increments/gate/stop?change=`: la parada de puerta del cambio, o su
// ausencia con el motivo. Solo lee.
func (s *Server) handleGateStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	stop, refusal := s.service.GateStop(r.Context(), r.URL.Query().Get("change"))
	if refusal != nil {
		s.respondMarksRefusal(w, refusal)
		return
	}
	s.respondJSON(w, http.StatusOK, stop)
}

// handleGateRecord es `POST /api/increments/gate`: registra la decisión de la puerta proyectada.
func (s *Server) handleGateRecord(w http.ResponseWriter, r *http.Request) {
	var req GateRecordRequest
	if !s.localJSONPreflight(w, r, &req, gatePreflightCodes) {
		return
	}
	recorded, refusal := s.service.RecordGate(r.Context(), req)
	if refusal != nil {
		s.respondMarksRefusal(w, refusal)
		return
	}
	s.respondJSON(w, http.StatusCreated, recorded)
}

// handleGatePublish es `POST /api/increments/gate/publish`: publica los documentos del cambio con
// confirmación explícita.
func (s *Server) handleGatePublish(w http.ResponseWriter, r *http.Request) {
	var req PublishGateRequest
	if !s.localJSONPreflight(w, r, &req, gatePreflightCodes) {
		return
	}
	published, refusal := s.service.PublishGate(r.Context(), req)
	if refusal != nil {
		s.respondMarksRefusal(w, refusal)
		return
	}
	s.respondJSON(w, http.StatusOK, published)
}
