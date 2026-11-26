package dashboard

import (
	"context"

	"github.com/IGutierrezZ/axiom/v3/internal/cli"
)

// MarksPublisher es la frontera con git de las marcas de pruebas (C-06-d, prov:WEB:3): el
// dashboard no ejecuta git ni importa sddpublish; la identidad del autor y la publicación acotada
// de `pruebas-estado.yaml` las da quien se inyecta (cli.MarksPublisher en producción).
type MarksPublisher interface {
	// MarksIdentity devuelve el autor `Nombre <email>` de la configuración git de dir o, si
	// falta alguna parte, las claves de git que faltan.
	MarksIdentity(ctx context.Context, dir string) (actor string, missing []string)
	// PublishMarks confirma y sube solo `pruebas-estado.yaml` del cambio.
	PublishMarks(ctx context.Context, req cli.MarksPublishRequest) cli.MarksPublishResult
}

// SetMarksPublisher inyecta el publicador de marcas. Sin él, las marcas responden
// `marks_publisher_missing`.
func (s *Service) SetMarksPublisher(p MarksPublisher) {
	s.marksMu.Lock()
	defer s.marksMu.Unlock()
	s.marksPub = p
}

func (s *Service) marksPublisher() MarksPublisher {
	s.marksMu.Lock()
	defer s.marksMu.Unlock()
	return s.marksPub
}
