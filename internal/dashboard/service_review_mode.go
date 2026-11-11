package dashboard

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
)

// Valores del cuerpo de POST /api/project/review-mode y de `review_mode` en el alta.
// «inherit» retira la preferencia del proyecto y deja que mande el interruptor global.
const (
	reviewModeOn      = "on"
	reviewModeOff     = "off"
	reviewModeInherit = "inherit"
)

// ErrProjectNotRegistered indica que el proyecto abierto no está en el registro del hub,
// donde vive la preferencia de revisión.
var ErrProjectNotRegistered = errors.New("el proyecto abierto no está registrado en el hub: la preferencia de revisión solo se guarda para proyectos registrados")

// ReviewModeDTO es el modo de revisión (RDD) efectivo del proyecto abierto y su origen:
// qué decide («clone_local», «project», «global» o «default») y cada fuente por separado.
// Registered dice si hay entrada del hub donde guardar la preferencia del proyecto.
type ReviewModeDTO struct {
	Effective  string `json:"effective"`
	Source     string `json:"source"`
	Global     string `json:"global"`
	Project    string `json:"project"`
	CloneLocal string `json:"clone_local"`
	Registered bool   `json:"registered"`
}

// normalizeReviewModeInput traduce el valor de la petición a la preferencia guardada
// («on», «off» o «» para heredar). El vacío también hereda.
func normalizeReviewModeInput(mode string) (string, error) {
	switch strings.TrimSpace(mode) {
	case "", reviewModeInherit:
		return "", nil
	case reviewModeOn:
		return hub.RDDModeOn, nil
	case reviewModeOff:
		return hub.RDDModeOff, nil
	default:
		return "", fmt.Errorf("el modo de revisión %q no es válido: usa %s, %s o %s", mode, reviewModeOn, reviewModeOff, reviewModeInherit)
	}
}

// reviewModeHub devuelve el registro del hub del servicio, creando el de la máquina si
// todavía no hay uno (como el resto de lecturas de proyectos).
func (s *Service) reviewModeHub() (*hub.Manager, error) {
	if s.hubManager == nil {
		mgr, err := hub.NewManager("")
		if err != nil {
			return nil, err
		}
		s.hubManager = mgr
	}
	return s.hubManager, nil
}

// GetReviewMode resuelve el modo de revisión del proyecto abierto sin tocar el
// directorio actual del proceso: la raíz es la del proyecto registrado.
func (s *Service) GetReviewMode() (*ReviewModeDTO, error) {
	mgr, err := s.reviewModeHub()
	if err != nil {
		return nil, err
	}
	root := s.getRootPath()
	status, err := cli.ReviewModeStatusFor(context.Background(), root, mgr)
	if err != nil {
		return nil, err
	}
	record, err := mgr.ProjectForRepo(root)
	if err != nil {
		return nil, err
	}
	return &ReviewModeDTO{
		Effective:  string(status.Effective),
		Source:     string(status.Source),
		Global:     string(status.Global),
		Project:    string(status.Project),
		CloneLocal: string(status.CloneLocal),
		Registered: record != nil,
	}, nil
}

// SetReviewMode guarda la preferencia de revisión del usuario para el proyecto abierto
// («on», «off» o «inherit») en el registro del hub, nunca en el repositorio, y devuelve
// el estado resuelto. Un proyecto sin registrar devuelve ErrProjectNotRegistered.
func (s *Service) SetReviewMode(mode string) (*ReviewModeDTO, error) {
	stored, err := normalizeReviewModeInput(mode)
	if err != nil {
		return nil, err
	}
	mgr, err := s.reviewModeHub()
	if err != nil {
		return nil, err
	}
	root := s.getRootPath()
	record, err := mgr.ProjectForRepo(root)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrProjectNotRegistered
	}
	if _, err := cli.SetProjectReviewMode(context.Background(), root, stored, mgr); err != nil {
		return nil, err
	}
	return s.GetReviewMode()
}
