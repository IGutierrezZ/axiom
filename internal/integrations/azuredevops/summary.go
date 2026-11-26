package azuredevops

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

const (
	fieldDescription = "System.Description"

	// maxSummaryAttempts es el número de lecturas y escrituras del resumen ante conflictos de /rev.
	maxSummaryAttempts = 3
	// maxTagsPerItem y maxTagLength son los topes de Azure DevOps para las etiquetas de un work item.
	maxTagsPerItem = 100
	maxTagLength   = 400
)

// UpdateSummary implementa integrations.TaskTracker (D12): lee el work item, calcula la descripción y
// el conjunto de etiquetas y envía un único PATCH con test /rev solo con las operaciones necesarias.
// Sin diferencias no escribe. Ante un conflicto relee y repite hasta tres veces.
func (p *Plugin) UpdateSummary(ctx context.Context, req integrations.SummaryRequest) (integrations.SummaryResult, error) {
	client, err := p.NewClient(req.Config, req.Secrets)
	if err != nil {
		return integrations.SummaryResult{}, err
	}
	id, err := strconv.Atoi(strings.TrimSpace(req.ItemID))
	if err != nil || id <= 0 {
		return integrations.SummaryResult{}, invalidWorkItemID(id)
	}
	var lastRev int
	for attempt := 1; attempt <= maxSummaryAttempts; attempt++ {
		wi, err := client.GetWorkItem(ctx, id)
		if err != nil {
			return integrations.SummaryResult{}, mapReadError(err)
		}
		lastRev = wi.Rev
		result, ops := planSummary(wi, req)
		if len(ops) == 0 {
			return result, nil
		}
		if _, err := client.UpdateWorkItem(ctx, id, wi.Rev, ops); err != nil {
			var conflict *RevConflictError
			if errors.As(err, &conflict) {
				continue
			}
			return integrations.SummaryResult{}, mapWriteError(err)
		}
		result.Wrote = true
		return result, nil
	}
	return integrations.SummaryResult{}, &integrations.RevConflictError{Rev: lastRev}
}

// planSummary calcula, para la lectura wi, las operaciones del PATCH y el resultado esperado.
// Las etiquetas propias son las que empiezan por req.TagPrefix (con distinción de mayúsculas): las
// ajenas se conservan. Por encima de los topes, o con una etiqueta no válida, el conjunto final son
// solo las ajenas y se avisa; la descripción se actualiza igualmente.
func planSummary(wi WorkItem, req integrations.SummaryRequest) (integrations.SummaryResult, []PatchOp) {
	var result integrations.SummaryResult
	var ops []PatchOp

	if fieldString(wi.Fields, fieldDescription) != req.HTML {
		ops = append(ops, PatchOp{Op: "add", Path: "/fields/" + fieldDescription, Value: req.HTML})
		result.DescriptionChanged = true
	}

	current := splitTags(fieldString(wi.Fields, fieldTags))
	var own, foreign []string
	for _, tag := range current {
		if req.TagPrefix != "" && strings.HasPrefix(tag, req.TagPrefix) {
			own = append(own, tag)
		} else {
			foreign = append(foreign, tag)
		}
	}
	final, warnings := finalTags(foreign, uniqueStrings(req.Tags), req.TagMax)
	result.Warnings = warnings

	if !sameStringSet(current, final) {
		ops = append(ops, PatchOp{Op: "add", Path: "/fields/" + fieldTags, Value: strings.Join(final, "; ")})
		result.TagsSent = true
		result.Added, result.Removed = diffOwnTags(own, final, req.TagPrefix)
	}
	return result, ops
}

// finalTags devuelve el conjunto final de etiquetas (las ajenas y, si caben, las deseadas) y los avisos.
func finalTags(foreign, desired []string, limit int) ([]string, []integrations.Warning) {
	for _, tag := range desired {
		if utf8.RuneCountInString(tag) > maxTagLength || strings.ContainsAny(tag, ",;") {
			return foreign, []integrations.Warning{{
				Code:    "req_tag_invalid",
				Message: fmt.Sprintf("la etiqueta %q no es válida (más de %d caracteres o contiene «,» o «;»); no se escribe ninguna etiqueta de requisito", tag, maxTagLength),
			}}
		}
	}
	if len(desired) > limit || len(foreign)+len(desired) > maxTagsPerItem {
		return foreign, []integrations.Warning{{
			Code: "req_tags_over_limit",
			Message: fmt.Sprintf("hay %d etiquetas de requisito y el tope es %d (y %d en total por work item); no se escribe ninguna etiqueta de requisito",
				len(desired), limit, maxTagsPerItem),
		}}
	}
	final := append([]string(nil), foreign...)
	return append(final, desired...), nil
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func sameStringSet(a, b []string) bool {
	set := make(map[string]bool, len(a))
	for _, s := range a {
		set[s] = true
	}
	other := uniqueStrings(b)
	if len(set) != len(other) {
		return false
	}
	for _, s := range other {
		if !set[s] {
			return false
		}
	}
	return true
}

// diffOwnTags devuelve las etiquetas propias nuevas de final y las propias actuales que desaparecen.
func diffOwnTags(own, final []string, prefix string) (added, removed []string) {
	was := make(map[string]bool, len(own))
	for _, s := range own {
		was[s] = true
	}
	now := make(map[string]bool, len(final))
	for _, s := range final {
		if prefix != "" && strings.HasPrefix(s, prefix) {
			now[s] = true
			if !was[s] {
				added = append(added, s)
			}
		}
	}
	for _, s := range own {
		if !now[s] {
			removed = append(removed, s)
		}
	}
	return added, removed
}
