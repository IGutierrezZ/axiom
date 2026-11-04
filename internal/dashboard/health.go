package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	// DefaultPort es el puerto fijo de `axiom ui`. Una PWA instalada queda ligada
	// a su origen exacto, así que el puerto no puede variar entre ejecuciones.
	// Está en el rango registrado (1024-49151), fuera del dinámico de Windows.
	DefaultPort = 47700

	// ServiceID identifica al dashboard de Axiom en `GET /api/health`.
	ServiceID = "axiom-dashboard"

	// probeTimeout acota el sondeo de un puerto local: si nada responde, falla rápido.
	probeTimeout = 800 * time.Millisecond
)

// ErrPortInUse indica que el puerto solicitado ya está ocupado por otro proceso.
var ErrPortInUse = errors.New("el puerto está ocupado")

// healthResponse es el cuerpo de `GET /api/health`.
type healthResponse struct {
	Service string `json:"service"`
	Version string `json:"version"`
}

// SetVersion fija la versión de Axiom que publica `GET /api/health`.
func (s *Server) SetVersion(version string) {
	s.version = version
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	s.respondJSON(w, http.StatusOK, healthResponse{Service: ServiceID, Version: s.version})
}

// ProbeAxiom informa de si en 127.0.0.1:port ya atiende un dashboard de Axiom,
// consultando `GET /api/health`. Un puerto sin nada escuchando, o con otra
// aplicación, devuelve (false, nil): solo los errores de construcción de la
// petición se propagan.
func ProbeAxiom(ctx context.Context, port int) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	url := fmt.Sprintf("http://127.0.0.1:%d/api/health", port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, nil
	}
	var body healthResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false, nil
	}
	return body.Service == ServiceID, nil
}
