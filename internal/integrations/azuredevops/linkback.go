package azuredevops

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

// linkBackAttempts es el presupuesto de intentos del enlace de vuelta: lectura, decisión y escritura
// con test /rev, repetido ante un conflicto de revisión.
const linkBackAttempts = 3

// LinkBack implementa integrations.WorkItemSource: añade req.Tag a System.Tags con una sola operación
// add sobre ese campo, protegida con test /rev.
//
//   - Si la etiqueta ya figura (sin distinguir mayúsculas) devuelve present sin escribir.
//   - Si req.Allow devuelve un error devuelve conflict con las etiquetas axiom:* ajenas, sin escribir.
//   - Ante un conflicto de revisión relee y repite, hasta 3 intentos en total; después devuelve
//     integrations.ErrItemConflict. Un PATCH aplicado cuya respuesta se perdió termina como present en
//     la relectura.
func (p *Plugin) LinkBack(ctx context.Context, req integrations.BackLinkRequest) (integrations.BackLink, error) {
	client, err := p.NewClient(req.Config, req.Secrets)
	if err != nil {
		return integrations.BackLink{}, err
	}
	id, err := strconv.Atoi(req.ID)
	if err != nil || id <= 0 {
		return integrations.BackLink{}, invalidWorkItemID(id)
	}

	for attempt := 0; attempt < linkBackAttempts; attempt++ {
		wi, err := client.GetWorkItem(ctx, id)
		if err != nil {
			var notFound *NotFoundError
			if errors.As(err, &notFound) {
				return integrations.BackLink{}, fmt.Errorf("%w: %w", integrations.ErrItemNotFound, err)
			}
			return integrations.BackLink{}, err
		}
		tags := splitTags(fieldString(wi.Fields, fieldTags))
		for _, t := range tags {
			if strings.EqualFold(t, req.Tag) {
				return integrations.BackLink{Outcome: integrations.BackLinkPresent}, nil
			}
		}
		if req.Allow != nil {
			if err := req.Allow(tags); err != nil {
				return integrations.BackLink{Outcome: integrations.BackLinkConflict, Conflicting: foreignAxiomTags(tags, req.Tag)}, nil
			}
		}
		_, err = client.UpdateWorkItem(ctx, id, wi.Rev, []PatchOp{{
			Op: "add", Path: "/fields/" + fieldTags, Value: strings.Join(append(tags, req.Tag), "; "),
		}})
		if err == nil {
			return integrations.BackLink{Outcome: integrations.BackLinkAdded}, nil
		}
		var conflict *RevConflictError
		if !errors.As(err, &conflict) {
			return integrations.BackLink{}, err
		}
	}
	return integrations.BackLink{}, fmt.Errorf("%w (tras %d intentos)", integrations.ErrItemConflict, linkBackAttempts)
}

// foreignAxiomTags devuelve las etiquetas axiom:* distintas de own, en su orden y grafía.
func foreignAxiomTags(tags []string, own string) []string {
	var out []string
	for _, t := range tags {
		if strings.HasPrefix(strings.ToLower(t), "axiom:") && !strings.EqualFold(t, own) {
			out = append(out, t)
		}
	}
	return out
}
