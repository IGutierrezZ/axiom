package adotest

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// State es un estado de un tipo de work item del servidor simulado.
type State struct {
	Name     string
	Category string // Proposed | InProgress | Resolved | Completed | Removed
}

type stateSeed struct {
	states     []State
	failStatus int
}

var statesPath = regexp.MustCompile(`/_apis/wit/workitemtypes/([^/]+)/states$`)

// SeedStates da de alta los estados del tipo typ (la grafía con la que llega la petición,
// decodificada, sin distinguir mayúsculas). Activa el simulacro de GET
// .../_apis/wit/workitemtypes/{tipo}/states con cualquier prefijo de organización y proyecto.
func (s *Server) SeedStates(typ string, states ...State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.states == nil {
		s.states = map[string]stateSeed{}
	}
	s.states[strings.ToLower(typ)] = stateSeed{states: append([]State(nil), states...)}
}

// FailStates hace que la consulta de estados del tipo typ responda siempre con status.
func (s *Server) FailStates(typ string, status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.states == nil {
		s.states = map[string]stateSeed{}
	}
	s.states[strings.ToLower(typ)] = stateSeed{failStatus: status}
}

// matchExtras devuelve el manejador de los estados y los adjuntos sembrados. Se llama con s.mu
// tomado.
func (s *Server) matchExtras(req *Request) Handler {
	if req.Method != http.MethodGet {
		return nil
	}
	if m := statesPath.FindStringSubmatch(req.Path); m != nil && s.states != nil {
		seed, ok := s.states[strings.ToLower(m[1])]
		if !ok {
			return nil
		}
		return func(*Request) Response { return statesResponse(m[1], seed) }
	}
	if b, ok := s.blobs[req.Path]; ok {
		return func(*Request) Response { return b.response() }
	}
	return s.extraGet(req)
}

func statesResponse(typ string, seed stateSeed) Response {
	if seed.failStatus != 0 {
		return Response{Status: seed.failStatus, JSON: map[string]any{"message": fmt.Sprintf("adotest: fallo programado al leer los estados de %s", typ)}}
	}
	value := make([]map[string]any, 0, len(seed.states))
	for _, st := range seed.states {
		value = append(value, map[string]any{"name": st.Name, "category": st.Category, "color": "ffffff"})
	}
	return Response{JSON: map[string]any{"count": len(value), "value": value}}
}
