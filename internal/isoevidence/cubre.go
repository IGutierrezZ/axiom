package isoevidence

import (
	"fmt"
	"regexp"
	"strings"
)

// cubreLine reconoce una línea `Cubre:` propia, con sangría o viñeta opcionales.
var cubreLine = regexp.MustCompile(`^[ \t]*(?:[-*+][ \t]+)?Cubre:[ \t]*(.*)$`)

// CubreLine es una línea `Cubre:` de un fichero de tareas.
type CubreLine struct {
	File string
	Line int
	// IDs son los identificadores tal como se escribieron, también los mal
	// formados.
	IDs []string
}

// ParseCubre extrae las líneas `Cubre: REQ-x[, REQ-y ...]` de un fichero de
// tareas, fuera de vallas. La cobertura es conjunta (unión de todos los
// ficheros), así que no se asocia cada línea a su tarea: se conservan el
// fichero y la línea para los motivos.
func ParseCubre(file, text string) []CubreLine {
	var out []CubreLine
	for _, line := range splitLines(text) {
		if line.Fenced {
			continue
		}
		m := cubreLine.FindStringSubmatch(line.Text)
		if m == nil {
			continue
		}
		ids := strings.FieldsFunc(m[1], func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t'
		})
		out = append(out, CubreLine{File: file, Line: line.Num, IDs: ids})
	}
	return out
}

// BaselineIndex indexa los requisitos de la baseline por su identificador
// `REQ-<n>.<m>` (REQ-29.14). Un requisito sin identificador en el nombre no
// entra: no puede citarse.
type BaselineIndex map[string][]Requirement

// NewBaselineIndex construye el índice. Varios requisitos con el mismo
// identificador quedan todos bajo él: la consulta los devuelve y el llamador
// decide si es ambiguo.
func NewBaselineIndex(baseline []Requirement) BaselineIndex {
	index := BaselineIndex{}
	for _, r := range baseline {
		valid, _ := RequirementIDs(r.Name)
		for _, id := range valid {
			index[id] = append(index[id], r)
		}
	}
	return index
}

// citation es la primera cita de un identificador en los `Cubre:`.
type citation struct {
	ID   string
	File string
	Line int
}

// citations devuelve los identificadores citados, sin repetidos y en el orden
// de la primera cita.
func citations(cubre []CubreLine) []citation {
	var out []citation
	seen := map[string]bool{}
	for _, c := range cubre {
		for _, id := range c.IDs {
			if seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, citation{ID: id, File: c.File, Line: c.Line})
		}
	}
	return out
}

// deltaIDs devuelve los identificadores válidos de los requisitos de la delta
// (en un RENAMED, el de su nombre nuevo), con el requisito que los declara.
func deltaIDs(delta []Requirement) (order []string, byID map[string]Requirement) {
	byID = map[string]Requirement{}
	for _, r := range delta {
		valid, _ := RequirementIDs(r.Name)
		for _, id := range valid {
			if _, seen := byID[id]; !seen {
				order = append(order, id)
				byID[id] = r
			}
		}
	}
	return order, byID
}

// CheckCoverage aplica REQ-29.4: cada requisito de la delta aparece en algún
// `Cubre:`, y todo identificador citado existe en la delta o en la baseline
// (también uno mal formado se reporta como desconocido, una vez por
// identificador). Con delta vacía solo se comprueban los desconocidos, que es
// lo que necesita el bug con requisito existente (REQ-29.5).
func CheckCoverage(delta []Requirement, index BaselineIndex, cubre []CubreLine) []Finding {
	var findings []Finding
	order, byID := deltaIDs(delta)
	cited := citations(cubre)
	citedSet := map[string]bool{}
	for _, c := range cited {
		citedSet[c.ID] = true
	}

	for _, id := range order {
		if citedSet[id] {
			continue
		}
		r := byID[id]
		findings = append(findings, Finding{
			Code:         CodeCoverage,
			Stage:        StageTasks,
			Subject:      id,
			Detail:       fmt.Sprintf("%s no está cubierto por ninguna tarea (%s)", id, describeRequirement(r)),
			Continuation: fmt.Sprintf("añade `Cubre: %s` en la tarea que lo implementa", id),
		})
	}
	for _, c := range cited {
		if _, inDelta := byID[c.ID]; inDelta {
			continue
		}
		if len(index[c.ID]) > 0 {
			continue
		}
		findings = append(findings, Finding{
			Code:         CodeCubreUnknown,
			Stage:        StageTasks,
			Subject:      c.ID,
			Detail:       fmt.Sprintf("%s:%d: `Cubre:` cita %s, que no existe en la delta ni en la baseline", c.File, c.Line, c.ID),
			Continuation: "corrige el identificador o retira la cita",
		})
	}
	return findings
}

// CheckExistingRequirementBug aplica REQ-29.5 a un bug con requisito existente:
// debe citar al menos un requisito, el requisito citado no puede ser ambiguo en
// la baseline y debe tener escenarios. Un identificador que no existe no se
// reporta aquí: lo reporta CheckCoverage como `iso_cubre_unknown`.
func CheckExistingRequirementBug(index BaselineIndex, cubre []CubreLine) []Finding {
	cited := citations(cubre)
	if len(cited) == 0 {
		return []Finding{{
			Code:         CodeBugRequirement,
			Stage:        StageTasks,
			Subject:      "",
			Detail:       "el bug sin delta no cita ningún requisito existente: ninguna tarea tiene `Cubre:`",
			Continuation: "cita el requisito que el bug restaura con `Cubre: REQ-<n>.<m>` en alguna tarea",
		}}
	}
	var findings []Finding
	for _, c := range cited {
		matches := index[c.ID]
		switch {
		case len(matches) == 0:
			continue
		case len(matches) > 1:
			findings = append(findings, Finding{
				Code:         CodeBugRequirement,
				Stage:        StageTasks,
				Subject:      c.ID,
				Detail:       fmt.Sprintf("%s:%d: %s es ambiguo en la baseline: lo declaran %d requisitos", c.File, c.Line, c.ID, len(matches)),
				Continuation: "corrige los identificadores de la baseline con un cambio normal con delta",
			})
		case len(Scenarios(matches[0].Text)) == 0:
			findings = append(findings, Finding{
				Code:         CodeBugRequirement,
				Stage:        StageTasks,
				Subject:      c.ID,
				Detail:       fmt.Sprintf("%s existe en la baseline (dominio %q, requisito %q) pero no tiene ningún escenario: no hay nada que verificar", c.ID, matches[0].Domain, matches[0].Name),
				Continuation: "tramita el trabajo como un cambio normal con delta que añada los escenarios, porque el requisito estaba mal especificado",
			})
		}
	}
	return findings
}

// CheckNoImpactBug aplica REQ-29.19: un bug sin impacto en la spec no puede
// tener `Cubre:`. Produce un hallazgo por cada identificador citado, y uno por
// la línea si no lleva ninguno.
func CheckNoImpactBug(cubre []CubreLine) []Finding {
	const continuation = "retira el `Cubre:` (el bug no toca ningún requisito) o retira la marca `" + NoImpactMark + "` y tramítalo como bug con requisito existente"
	var findings []Finding
	for _, c := range cubre {
		ids := c.IDs
		if len(ids) == 0 {
			findings = append(findings, Finding{
				Code:         CodeBugContradiction,
				Stage:        StageTasks,
				Subject:      c.File,
				Detail:       fmt.Sprintf("%s:%d: `Cubre:` junto a la marca `%s`", c.File, c.Line, NoImpactMark),
				Continuation: continuation,
			})
			continue
		}
		for _, id := range ids {
			findings = append(findings, Finding{
				Code:         CodeBugContradiction,
				Stage:        StageTasks,
				Subject:      id,
				Detail:       fmt.Sprintf("%s:%d: `Cubre:` cita %s junto a la marca `%s`", c.File, c.Line, id, NoImpactMark),
				Continuation: continuation,
			})
		}
	}
	return findings
}

// ScenarioKey identifica un escenario afectado: el par (requisito, nombre).
type ScenarioKey struct{ RequirementID, Scenario string }

// AffectedFromDelta devuelve los escenarios afectados de una delta: cada
// escenario de un bloque ADDED o MODIFIED, identificado por el `REQ-<n>.<m>` de
// su nombre. Sin repetidos y en orden.
func AffectedFromDelta(delta []Requirement) []ScenarioKey {
	var out []ScenarioKey
	seen := map[ScenarioKey]bool{}
	for _, r := range delta {
		if r.Section != SectionAdded && r.Section != SectionModified {
			continue
		}
		valid, _ := RequirementIDs(r.Name)
		if len(valid) != 1 {
			continue
		}
		for _, name := range Scenarios(r.Text) {
			k := ScenarioKey{RequirementID: valid[0], Scenario: name}
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	return out
}

// AffectedFromBaseline devuelve los escenarios de los requisitos existentes
// que cita un bug con requisito existente. Un identificador que no existe no
// aporta ninguno.
func AffectedFromBaseline(index BaselineIndex, ids []string) []ScenarioKey {
	var out []ScenarioKey
	seen := map[ScenarioKey]bool{}
	for _, id := range ids {
		for _, r := range index[id] {
			for _, name := range Scenarios(r.Text) {
				k := ScenarioKey{RequirementID: id, Scenario: name}
				if !seen[k] {
					seen[k] = true
					out = append(out, k)
				}
			}
		}
	}
	return out
}
