package acceptancetests

import (
	"fmt"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// requirement es un requisito de la delta con ID único, en la base de cobertura.
type requirement struct {
	id        string
	domain    string
	scenarios []string // nombres distintos, en orden
	citable   map[string]bool
}

// Check parsea `pruebas.md` y lo cruza con los requisitos ADDED y MODIFIED de la delta
// (RF-KOF-20). Es de solo lectura y determinista. No emite `pruebas_ausente`: lo añade el
// adaptador con MissingFinding.
func Check(text string, in Input) Report {
	b := Parse(text)
	var findings []Finding
	findings = append(findings, b.Issues...)
	findings = append(findings, checkCases(b.Cases, in)...)

	reqs, deltaFindings, noID := walkDelta(in)
	findings = append(findings, deltaFindings...)

	byID := map[string]*requirement{}
	for i := range reqs {
		byID[reqs[i].id] = &reqs[i]
	}
	coveredReq := map[string]bool{}
	coveredScn := map[[2]string]bool{}
	for _, c := range b.Cases {
		seen := map[CubreEntry]bool{}
		for _, e := range c.Cubre {
			if e.ID == "" || seen[e] {
				continue // entrada vacía (ya informada) o cita repetida: no es hallazgo
			}
			seen[e] = true
			if !recognized(e.ID, in.Pattern) {
				findings = append(findings, caseFinding(c, CodeUnknownID, e.ID,
					fmt.Sprintf("el identificador %q no lo reconoce el patrón de identificadores del proyecto", e.ID)))
				continue
			}
			r, ok := byID[e.ID]
			if !ok {
				findings = append(findings, caseFinding(c, CodeUnknownID, e.ID,
					fmt.Sprintf("el identificador %q no figura entre los requisitos ADDED o MODIFIED de la delta", e.ID)))
				continue
			}
			coveredReq[e.ID] = true
			if e.Scenario == "" {
				continue
			}
			if !r.citable[e.Scenario] {
				findings = append(findings, caseFinding(c, CodeUnknownScenario, e.ID+"#"+e.Scenario,
					fmt.Sprintf("el requisito %s no tiene el escenario %q", e.ID, e.Scenario)))
				continue
			}
			coveredScn[[2]string{e.ID, e.Scenario}] = true
		}
	}

	rep := Report{Battery: b, Cases: len(b.Cases)}
	rep.Requirements.Total = len(reqs) + noID
	for _, r := range reqs {
		file := deltaFile(r.domain)
		if coveredReq[r.id] {
			rep.Requirements.Covered++
		} else {
			findings = append(findings, Finding{Code: CodeUncoveredReq, File: file, Subject: r.id,
				Message: fmt.Sprintf("ningún caso cita el requisito %s", r.id)})
		}
		for _, name := range r.scenarios {
			rep.Scenarios.Total++
			switch {
			case coveredScn[[2]string{r.id, name}]:
				rep.Scenarios.Covered++
			case !r.citable[name]:
				// no citable: ya informado como escenario_ambiguo
			default:
				findings = append(findings, Finding{Code: CodeUncoveredScenario, File: file, Subject: r.id + "#" + name,
					Message: fmt.Sprintf("ningún caso cita el escenario %q del requisito %s con %s#nombre", name, r.id, r.id)})
			}
		}
	}
	sortFindings(findings)
	rep.Findings = findings
	return rep
}

func deltaFile(domain string) string { return "specs/" + domain + "/spec.md" }

func caseFinding(c Case, code Code, subject, msg string) Finding {
	return Finding{Code: code, File: FileName, Line: c.Line, Subject: subject, Message: msg}
}

// recognized dice si id lo reconoce el puerto reqid: un ID del patrón o un provisional que
// el patrón admite.
func recognized(id string, p reqid.Pattern) bool {
	if p.Valid(id) {
		return true
	}
	spans := reqid.ScanProvisional(id)
	if len(spans) != 1 || spans[0].Token != id {
		return false
	}
	_, err := p.ParseProvisional(id)
	return err == nil
}

// walkDelta recorre la delta con la misma selección que isoevidence.AffectedFromDelta
// (ADDED con válidos más provisionales, MODIFIED, exactamente un ID), pero informa de lo que
// esa función omite en silencio: requisitos sin ID, IDs duplicados y escenarios ambiguos.
// Devuelve los requisitos con ID único, los hallazgos de delta y el número de requisitos sin
// ID citable (que cuentan en el total y nunca quedan cubiertos).
func walkDelta(in Input) (reqs []requirement, findings []Finding, noID int) {
	seen := map[string]bool{}
	for _, r := range in.Delta {
		if r.Section != isoevidence.SectionAdded && r.Section != isoevidence.SectionModified {
			continue
		}
		file := deltaFile(r.Domain)
		valid, provisional, _ := isoevidence.RequirementKeys(r.Name, in.Pattern)
		ids := valid
		if r.Section == isoevidence.SectionAdded {
			ids = append(append([]string(nil), valid...), provisional...)
		}
		if len(ids) != 1 {
			noID++
			msg := fmt.Sprintf("el requisito %q no tiene ningún identificador citable", r.Name)
			if len(ids) > 1 {
				msg = fmt.Sprintf("el requisito %q tiene más de un identificador (%s) y no se puede citar", r.Name, strings.Join(ids, ", "))
			}
			findings = append(findings, Finding{Code: CodeRequirementNoID, File: file, Subject: r.Name, Message: msg})
			continue
		}
		id := ids[0]
		if seen[id] {
			findings = append(findings, Finding{Code: CodeDuplicateDeltaID, File: file, Subject: id,
				Message: fmt.Sprintf("el identificador %s figura en más de un requisito de la delta", id)})
			continue
		}
		seen[id] = true
		req := requirement{id: id, domain: r.Domain, citable: map[string]bool{}}
		names := map[string]bool{}
		for _, name := range isoevidence.Scenarios(r.Text) {
			if names[name] {
				findings = append(findings, Finding{Code: CodeAmbiguousScenario, File: file, Subject: id + "#" + name,
					Message: fmt.Sprintf("el requisito %s repite el escenario %q", id, name)})
				continue
			}
			names[name] = true
			req.scenarios = append(req.scenarios, name)
			if strings.Contains(name, ";") {
				findings = append(findings, Finding{Code: CodeAmbiguousScenario, File: file, Subject: id + "#" + name,
					Message: fmt.Sprintf("el escenario %q del requisito %s contiene ';' y no se puede citar", name, id)})
				continue
			}
			req.citable[name] = true
		}
		reqs = append(reqs, req)
	}
	return reqs, findings, noID
}

// checkCases valida los dominios de cada celda (RF-KOF-18) y la sintaxis de `Cubre`.
func checkCases(cases []Case, in Input) []Finding {
	var out []Finding
	seenIDs := map[string]bool{}
	for _, c := range cases {
		switch {
		case !caseIDPattern.MatchString(c.ID):
			out = append(out, caseFinding(c, CodeInvalidCaseID, c.ID,
				fmt.Sprintf("el ID %q no cumple P-NN (P- seguido de al menos dos dígitos)", c.ID)))
		case seenIDs[c.ID]:
			out = append(out, caseFinding(c, CodeDuplicateCaseID, c.ID, fmt.Sprintf("el ID %s está repetido", c.ID)))
		}
		if c.ID != "" {
			seenIDs[c.ID] = true
		}
		if !contains([]string{"manual", "automatizable"}, c.Type) {
			out = append(out, caseFinding(c, CodeInvalidCell, "Tipo",
				fmt.Sprintf("Tipo %q fuera de dominio; admitidos: manual, automatizable", c.Type)))
		}
		if !contains([]string{"alta", "media", "baja"}, c.Priority) {
			out = append(out, caseFinding(c, CodeInvalidCell, "Prioridad",
				fmt.Sprintf("Prioridad %q fuera de dominio; admitidos: alta, media, baja", c.Priority)))
		}
		if strings.TrimSpace(c.Steps) == "" {
			out = append(out, caseFinding(c, CodeInvalidCell, "Pasos", "la celda Pasos está vacía"))
		}
		if strings.TrimSpace(c.Expected) == "" {
			out = append(out, caseFinding(c, CodeInvalidCell, "Resultado esperado", "la celda Resultado esperado está vacía"))
		}
		if !contains(in.Roles, c.Role) {
			out = append(out, caseFinding(c, CodeInvalidRole, c.Role,
				fmt.Sprintf("el rol %q no es un rol sellado del cambio; roles sellados: %s", c.Role, strings.Join(in.Roles, ", "))))
		}
		if !in.SkipEnvironments {
			out = append(out, checkEnvironments(c, in.Environments)...)
		}
		empty := len(c.Cubre) == 0
		for _, e := range c.Cubre {
			if e.ID == "" {
				empty = true
			}
		}
		if empty {
			out = append(out, caseFinding(c, CodeNoCubre, c.ID,
				fmt.Sprintf("la celda Cubre %q está vacía o tiene una entrada vacía", c.CubreRaw)))
		}
	}
	return out
}

func checkEnvironments(c Case, declared []string) []Finding {
	if len(c.Environments) == 0 {
		return []Finding{caseFinding(c, CodeInvalidEnvironment, "Entornos", "la celda Entornos está vacía")}
	}
	var out []Finding
	seen := map[string]bool{}
	for _, env := range c.Environments {
		switch {
		case env == "":
			out = append(out, caseFinding(c, CodeInvalidEnvironment, "Entornos", "Entornos tiene un elemento vacío"))
		case seen[env]:
			out = append(out, caseFinding(c, CodeInvalidEnvironment, env, fmt.Sprintf("el entorno %q está repetido", env)))
		case !contains(declared, env):
			out = append(out, caseFinding(c, CodeInvalidEnvironment, env,
				fmt.Sprintf("el entorno %q no está declarado; declarados: %s", env, strings.Join(declared, ", "))))
		}
		seen[env] = true
	}
	return out
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// sortFindings ordena por (fichero, línea, código, asunto, mensaje) para un informe estable.
func sortFindings(f []Finding) {
	sort.SliceStable(f, func(i, j int) bool {
		a, b := f[i], f[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.Subject != b.Subject {
			return a.Subject < b.Subject
		}
		return a.Message < b.Message
	})
}
