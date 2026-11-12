package isoevidence

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// reqCandidate encuentra un token que empieza por `REQ-` en límite de
	// palabra. El último carácter es alfanumérico, de modo que la puntuación
	// final (`REQ-30.1,` o `REQ-30.1.`) no forma parte del token.
	reqCandidate = regexp.MustCompile(`\bREQ-[0-9A-Za-z.]*[0-9A-Za-z]`)
	reqValid     = regexp.MustCompile(`^REQ-\d+\.\d+$`)
)

// RequirementIDs extrae los identificadores del nombre de un requisito. Es
// válido el que encaja exactamente con `REQ-<n>.<m>`; el resto de candidatos
// (`REQ-30`, `REQ-x.1`) son mal formados. Cada lista va sin repetidos y en el
// orden de aparición.
func RequirementIDs(name string) (valid []string, malformed []string) {
	seenValid := map[string]bool{}
	seenMalformed := map[string]bool{}
	for _, token := range reqCandidate.FindAllString(name, -1) {
		if reqValid.MatchString(token) {
			if !seenValid[token] {
				seenValid[token] = true
				valid = append(valid, token)
			}
			continue
		}
		if !seenMalformed[token] {
			seenMalformed[token] = true
			malformed = append(malformed, token)
		}
	}
	return valid, malformed
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

const (
	continuationAddID   = "añade o corrige en el nombre un identificador REQ-<n>.<m>"
	continuationRenamed = "añade el identificador REQ-<n>.<m> al nombre con un RENAMED de la delta"
	continuationUnique  = "elige un identificador REQ-<n>.<m> que no exista ya en la baseline ni en otro requisito ADDED"
	continuationOneID   = "deja un único identificador REQ-<n>.<m> en el nombre"
	continuationSameID  = "conserva el mismo identificador al renombrar, o retira el del nombre antiguo"
)

// CheckRequirementIDs aplica la regla REQ-29.3 a los requisitos de la delta.
// Solo se comprueban los de la delta, nunca el resto de la baseline.
func CheckRequirementIDs(delta, baseline []Requirement) []Finding {
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
		valid, _ := RequirementIDs(b.Name)
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

	for _, r := range delta {
		valid, malformed := RequirementIDs(r.Name)
		subject := describeRequirement(r)
		switch {
		case len(valid) == 0:
			detail := subject + ": el nombre no contiene un identificador REQ-<n>.<m>"
			if len(malformed) > 0 {
				detail += fmt.Sprintf(" (forma no válida: %s)", strings.Join(malformed, ", "))
			}
			continuation := continuationAddID
			if r.Section == SectionModified || r.Section == SectionRemoved {
				continuation = continuationRenamed
			}
			add(r, detail, continuation)
			continue
		case len(valid) > 1:
			add(r, fmt.Sprintf("%s: el nombre es ambiguo, contiene los identificadores %s", subject, strings.Join(valid, ", ")), continuationOneID)
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
				add(r, fmt.Sprintf("%s: el identificador %s ya existe en la baseline (dominio %q, requisito %q)", subject, id, b.Domain, b.Name), continuationUnique)
				break
			}
		case SectionRenamed:
			oldValid, _ := RequirementIDs(r.OldName)
			if len(oldValid) > 0 {
				if len(oldValid) != 1 || oldValid[0] != id {
					add(r, fmt.Sprintf("%s: el RENAMED cambia el identificador de %s a %s", subject, strings.Join(oldValid, ", "), id), continuationSameID)
				}
				continue
			}
			noteAdder(id, r)
			if existing := baselineByID[id]; len(existing) > 0 {
				b := existing[0]
				add(r, fmt.Sprintf("%s: el identificador %s ya existe en la baseline (dominio %q, requisito %q)", subject, id, b.Domain, b.Name), continuationUnique)
			}
		}
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
			Continuation: continuationUnique,
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
