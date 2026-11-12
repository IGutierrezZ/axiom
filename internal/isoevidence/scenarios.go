package isoevidence

import (
	"fmt"
	"regexp"
)

var scenarioHeading = regexp.MustCompile(`^#### Scenario:[ \t]*(.*?)[ \t]*$`)

// Scenarios devuelve los nombres de los escenarios `#### Scenario:` de un
// bloque de requisito, en orden y sin contar los que quedan dentro de una valla.
func Scenarios(blockText string) []string {
	var names []string
	for _, line := range splitLines(blockText) {
		if line.Fenced {
			continue
		}
		if m := scenarioHeading.FindStringSubmatch(line.Text); m != nil {
			names = append(names, m[1])
		}
	}
	return names
}

// CheckScenarios aplica REQ-29.10 y la regla de nombres repetidos de REQ-29.6:
// un requisito ADDED o MODIFIED sin escenarios, o con dos escenarios del mismo
// nombre, produce un hallazgo. REMOVED y RENAMED no llevan escenarios y quedan
// exentos.
func CheckScenarios(delta []Requirement) []Finding {
	var findings []Finding
	for _, r := range delta {
		if r.Section != SectionAdded && r.Section != SectionModified {
			continue
		}
		subject := describeRequirement(r)
		names := Scenarios(r.Text)
		if len(names) == 0 {
			findings = append(findings, Finding{
				Code:         CodeRequirementScenarios,
				Stage:        StageSpecs,
				Subject:      r.Name,
				Detail:       subject + ": no tiene ningún escenario `#### Scenario:`",
				Continuation: "añade al menos un `#### Scenario:` al requisito",
			})
			continue
		}
		seen := map[string]int{}
		var order []string
		for _, n := range names {
			if seen[n] == 0 {
				order = append(order, n)
			}
			seen[n]++
		}
		for _, n := range order {
			if seen[n] > 1 {
				findings = append(findings, Finding{
					Code:         CodeRequirementScenarios,
					Stage:        StageSpecs,
					Subject:      r.Name,
					Detail:       fmt.Sprintf("%s: el escenario %q se repite %d veces", subject, n, seen[n]),
					Continuation: "renombra el escenario repetido para que cada nombre sea único dentro del requisito",
				})
			}
		}
	}
	return findings
}
