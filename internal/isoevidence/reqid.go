package isoevidence

import (
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// RequirementKeys extrae los identificadores del nombre de un requisito. Es válido el que
// encaja exactamente con el patrón p del proyecto (`REQ-<n>.<m>` por defecto); es
// provisional el token `prov:…` que el patrón admite (REQ-47.1); el resto de candidatos
// (`REQ-30`, `REQ-x.1`, un provisional que el patrón no admite) son mal formados. Cada
// lista va sin repetidos y en el orden de aparición. El análisis es del puerto reqid
// (REQ-44.5).
func RequirementKeys(name string, p reqid.Pattern) (valid, provisional, malformed []string) {
	valid, malformed = p.Scan(name)
	seen := map[string]bool{}
	for _, span := range reqid.ScanProvisional(name) {
		if seen[span.Token] {
			continue
		}
		seen[span.Token] = true
		if _, err := p.ParseProvisional(span.Token); err != nil {
			malformed = append(malformed, span.Token)
			continue
		}
		provisional = append(provisional, span.Token)
	}
	return valid, provisional, malformed
}

// isProvisionalToken dice si s es, entero, un token con la forma de un provisional.
func isProvisionalToken(s string) bool {
	spans := reqid.ScanProvisional(s)
	return len(spans) == 1 && spans[0].Start == 0 && spans[0].End == len(s)
}

// ensureTrailingNewline reproduce la única normalización que aplica el
// compositor nativo al insertar un bloque ADDED (D17).
func ensureTrailingNewline(text string) string {
	if strings.HasSuffix(text, "\n") {
		return text
	}
	return text + "\n"
}

// describeRequirement nombra el dominio, la sección y el requisito.
func describeRequirement(r Requirement) string {
	return fmt.Sprintf("dominio %q, sección %s, requisito %q", r.Domain, r.Section, r.Name)
}

// continuations son las continuaciones de REQ-29.3; nombran el patrón efectivo con
// su plantilla, de modo que con el defecto el texto es el de siempre.
type continuations struct {
	addID, renamed, unique, oneID, sameID, provisionalAdded, uniqueProvisional string
}

func continuationsFor(p reqid.Pattern) continuations {
	t := p.Template()
	return continuations{
		addID:             "añade o corrige en el nombre un identificador " + t,
		renamed:           "añade el identificador " + t + " al nombre con un RENAMED de la delta",
		unique:            "elige un identificador " + t + " que no exista ya en la baseline ni en otro requisito ADDED",
		oneID:             "deja un único identificador " + t + " en el nombre",
		sameID:            "conserva el mismo identificador al renombrar, o retira el del nombre antiguo",
		provisionalAdded:  "usa el identificador definitivo del requisito existente: un provisional solo va en el nombre de un ADDED",
		uniqueProvisional: "usa un provisional distinto en cada requisito ADDED del cambio",
	}
}

// CheckRequirementIDs aplica la regla REQ-29.3 a los requisitos de la delta con el
// patrón p del proyecto. Solo se comprueban los de la delta, nunca el resto de la
// baseline. ADDED, la colisión y la tolerancia D17 siguen por título e identificador
// exactos; un MODIFIED o REMOVED lleva el identificador de su nombre, que es el del
// requisito existente, así que un título cambiado con el mismo identificador pasa igual
// (REQ-30.22).
func CheckRequirementIDs(delta, baseline []Requirement, p reqid.Pattern) []Finding {
	c := continuationsFor(p)
	var findings []Finding
	add := func(r Requirement, detail, continuation string) {
		findings = append(findings, Finding{
			Code:         CodeRequirementID,
			Stage:        StageSpecs,
			Subject:      r.Name,
			Detail:       detail,
			Continuation: continuation,
		})
	}

	baselineByID := map[string][]Requirement{}
	for _, b := range baseline {
		valid, _, _ := RequirementKeys(b.Name, p)
		for _, id := range valid {
			baselineByID[id] = append(baselineByID[id], b)
		}
	}

	// adders recoge, por identificador, los requisitos de la delta que
	// introducen ese identificador (ADDED y RENAMED que lo añade).
	adders := map[string][]Requirement{}
	var adderOrder []string
	noteAdder := func(id string, r Requirement) {
		if _, seen := adders[id]; !seen {
			adderOrder = append(adderOrder, id)
		}
		adders[id] = append(adders[id], r)
	}

	// provisionals recoge, por token, los ADDED que lo declaran: un provisional es único en el
	// cambio, en el mismo dominio o en otro (REQ-47.1).
	provisionals := map[string][]Requirement{}
	var provisionalOrder []string

	for _, r := range delta {
		valid, provisional, malformed := RequirementKeys(r.Name, p)
		subject := describeRequirement(r)
		if r.Section == SectionRenamed {
			if _, oldProvisional, _ := RequirementKeys(r.OldName, p); len(oldProvisional) > 0 {
				add(r, fmt.Sprintf("%s: el nombre antiguo del RENAMED contiene el provisional %s, que solo es válido en un ADDED", subject, oldProvisional[0]), c.provisionalAdded)
				continue
			}
		}
		all := append(append([]string(nil), valid...), provisional...)
		switch {
		case len(all) == 0:
			detail := subject + ": el nombre no contiene un identificador " + p.Template()
			if len(malformed) > 0 {
				detail += fmt.Sprintf(" (forma no válida: %s)", strings.Join(malformed, ", "))
			}
			continuation := c.addID
			if r.Section == SectionModified || r.Section == SectionRemoved {
				continuation = c.renamed
			}
			add(r, detail, continuation)
			continue
		case len(all) > 1:
			add(r, fmt.Sprintf("%s: el nombre es ambiguo, contiene los identificadores %s", subject, strings.Join(all, ", ")), c.oneID)
			continue
		case len(provisional) > 0 && r.Section != SectionAdded:
			add(r, fmt.Sprintf("%s: el nombre contiene el provisional %s, que solo es válido en un ADDED", subject, provisional[0]), c.provisionalAdded)
			continue
		case len(provisional) > 0:
			if _, seen := provisionals[provisional[0]]; !seen {
				provisionalOrder = append(provisionalOrder, provisional[0])
			}
			provisionals[provisional[0]] = append(provisionals[provisional[0]], r)
			continue
		}
		id := valid[0]

		switch r.Section {
		case SectionAdded:
			noteAdder(id, r)
			for _, b := range baselineByID[id] {
				if tolerated(r, b) {
					continue
				}
				add(r, fmt.Sprintf("%s: el identificador %s ya existe en la baseline (dominio %q, requisito %q)", subject, id, b.Domain, b.Name), c.unique)
				break
			}
		case SectionRenamed:
			oldValid, _, _ := RequirementKeys(r.OldName, p)
			if len(oldValid) > 0 {
				if len(oldValid) != 1 || oldValid[0] != id {
					add(r, fmt.Sprintf("%s: el RENAMED cambia el identificador de %s a %s", subject, strings.Join(oldValid, ", "), id), c.sameID)
				}
				continue
			}
			noteAdder(id, r)
			if existing := baselineByID[id]; len(existing) > 0 {
				b := existing[0]
				add(r, fmt.Sprintf("%s: el identificador %s ya existe en la baseline (dominio %q, requisito %q)", subject, id, b.Domain, b.Name), c.unique)
			}
		}
	}

	for _, token := range provisionalOrder {
		group := provisionals[token]
		if len(group) < 2 {
			continue
		}
		names := make([]string, 0, len(group))
		for _, r := range group {
			names = append(names, fmt.Sprintf("%q (dominio %q)", r.Name, r.Domain))
		}
		findings = append(findings, Finding{
			Code:         CodeRequirementID,
			Stage:        StageSpecs,
			Subject:      token,
			Detail:       fmt.Sprintf("el provisional %s se repite en %d requisitos ADDED de la delta: %s", token, len(group), strings.Join(names, " y ")),
			Continuation: c.uniqueProvisional,
		})
	}

	for _, id := range adderOrder {
		group := adders[id]
		if len(group) < 2 {
			continue
		}
		names := make([]string, 0, len(group))
		for _, r := range group {
			names = append(names, fmt.Sprintf("%q (dominio %q, sección %s)", r.Name, r.Domain, r.Section))
		}
		findings = append(findings, Finding{
			Code:         CodeRequirementID,
			Stage:        StageSpecs,
			Subject:      id,
			Detail:       fmt.Sprintf("el identificador %s se repite en %d requisitos de la delta: %s", id, len(group), strings.Join(names, " y ")),
			Continuation: c.unique,
		})
	}
	return findings
}

// tolerated aplica la excepción del archivado interrumpido (D17): el ADDED no
// bloquea si la baseline del mismo dominio contiene un requisito con el mismo
// nombre y el mismo bloque, byte a byte, tras la normalización del compositor.
func tolerated(added, baseline Requirement) bool {
	return added.Domain == baseline.Domain &&
		added.Name == baseline.Name &&
		ensureTrailingNewline(added.Text) == baseline.Text
}
