package adotest

import "net/http"

// Blob es un recurso binario (un adjunto o una imagen incrustada) del servidor simulado.
type Blob struct {
	Data []byte
	// ContentType vacío con NoContentType es la respuesta sin cabecera Content-Type; vacío sin
	// NoContentType responde application/octet-stream, como Azure DevOps con casi todos los adjuntos.
	ContentType   string
	NoContentType bool
	// Status distinto de cero fija el estado de la respuesta (por ejemplo 500).
	Status int
	// RedirectTo responde 302 con esa cabecera Location y sin cuerpo.
	RedirectTo string
}

func (b Blob) response() Response {
	if b.RedirectTo != "" {
		return Response{Status: http.StatusFound, Header: http.Header{"Location": {b.RedirectTo}}}
	}
	resp := Response{Status: b.Status, Body: string(b.Data), ContentType: b.ContentType, NoContentType: b.NoContentType,
		Chunked: true} // nunca lleva Content-Length: el límite del cliente no puede depender de él
	if resp.ContentType == "" && !resp.NoContentType {
		resp.ContentType = "application/octet-stream"
	}
	return resp
}

// SeedAttachment sirve b en la ruta decodificada path con GET. La respuesta nunca lleva
// Content-Length. Las rutas registradas con Handle tienen prioridad. Count(GET, path) da el número
// de lecturas.
func (s *Server) SeedAttachment(path string, b Blob) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blobs == nil {
		s.blobs = map[string]Blob{}
	}
	s.blobs[path] = b
}
