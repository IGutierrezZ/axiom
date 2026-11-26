package dashboard

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
)

// maxMarksBody es el tamaño máximo del cuerpo de las rutas de marcas (D13).
const maxMarksBody = 16 << 10

// marksPreflight aplica el orden de D13 hasta el cuerpo: método (405), petición local (403),
// Content-Type JSON (415) y cuerpo de hasta 16 KiB con campos conocidos (400). Devuelve false si
// ya respondió.
func (s *Server) marksPreflight(w http.ResponseWriter, r *http.Request, into any) bool {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return false
	}
	if !s.isLocalRequest(r) {
		s.respondMarksRefusal(w, marksRefuse(http.StatusForbidden, codeRequestNotLocal,
			"la petición no procede del propio dashboard",
			"abre el dashboard en http://127.0.0.1 o http://localhost y repite la acción"))
		return false
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		s.respondMarksRefusal(w, marksRefuse(http.StatusUnsupportedMediaType, codeMarksContentType,
			"el cuerpo debe enviarse como application/json",
			"envía un JSON con Content-Type: application/json"))
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxMarksBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(into); err != nil {
		s.respondMarksRefusal(w, marksBodyInvalid("JSON inválido o demasiado grande: "+err.Error()))
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		s.respondMarksRefusal(w, marksBodyInvalid("el cuerpo debe ser un único objeto JSON"))
		return false
	}
	return true
}

func (s *Server) respondMarksRefusal(w http.ResponseWriter, refusal *MarksRefusal) {
	s.respondJSON(w, refusal.HTTPStatus, refusal.body())
}

// handleIncrementMark es `POST /api/increments/marks`: añade una marca de ejecución.
func (s *Server) handleIncrementMark(w http.ResponseWriter, r *http.Request) {
	var req MarkRequest
	if !s.marksPreflight(w, r, &req) {
		return
	}
	recorded, refusal := s.service.RecordMark(r.Context(), req)
	if refusal != nil {
		s.respondMarksRefusal(w, refusal)
		return
	}
	s.respondJSON(w, http.StatusCreated, recorded)
}

// handleIncrementMarksPublish es `POST /api/increments/marks/publish`: publica las marcas
// pendientes con confirmación explícita.
func (s *Server) handleIncrementMarksPublish(w http.ResponseWriter, r *http.Request) {
	var req PublishMarksRequest
	if !s.marksPreflight(w, r, &req) {
		return
	}
	published, refusal := s.service.PublishMarks(r.Context(), req)
	if refusal != nil {
		s.respondMarksRefusal(w, refusal)
		return
	}
	s.respondJSON(w, http.StatusOK, published)
}
