package marks

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/acceptancetests"
)

var caseIDPattern = regexp.MustCompile(`^P-[0-9]{2,}$`)

// EnvSummary es el recuento de un entorno. Las categorías son disjuntas y cumplen
// OK+KO+Blocked+NA+Stale+Unmarked == Total.
type EnvSummary struct {
	Environment string `json:"environment"`
	Total       int    `json:"total"`
	OK          int    `json:"ok"`
	KO          int    `json:"ko"`
	Blocked     int    `json:"blocked"`
	NA          int    `json:"na"`
	Unmarked    int    `json:"unmarked"`
	Stale       int    `json:"stale"`
}

// Summary es el resumen por entorno de las marcas contra la batería y el digest actuales.
type Summary struct {
	Environments []EnvSummary
	// Orphans cuenta las entradas cuyo caso ya no existe o cuyo entorno no está declarado ni
	// aplica al caso. No cuentan en ninguna categoría.
	Orphans int
	// Invalid cuenta las entradas fuera de forma del fichero (File.Invalid); Summarize no la
	// calcula porque solo recibe entradas válidas: quien lee el fichero la rellena.
	Invalid int
}

type pair struct{ caseID, env string }

// Summarize calcula el resumen (D4). Entornos: los declarados, en su orden. Casos: filas con
// ID P-NN; un ID repetido cuenta una vez con la unión de sus entornos. La marca vigente de un
// par es la de mayor `at` entre las de digest igual al actual; a igualdad, la última del
// fichero. Un par sin vigente pero con alguna entrada es caducado; sin ninguna, sin marcar.
func Summarize(battery acceptancetests.Battery, environments []string, entries []Entry, digest string) Summary {
	declared := make(map[string]bool, len(environments))
	for _, env := range environments {
		declared[env] = true
	}
	applicable := map[string]map[string]bool{}
	var caseOrder []string
	for _, c := range battery.Cases {
		if !caseIDPattern.MatchString(c.ID) {
			continue
		}
		if _, seen := applicable[c.ID]; !seen {
			applicable[c.ID] = map[string]bool{}
			caseOrder = append(caseOrder, c.ID)
		}
		for _, env := range c.Environments {
			if declared[env] {
				applicable[c.ID][env] = true
			}
		}
	}

	current := map[pair]Entry{}
	currentAt := map[pair]time.Time{}
	hasEntry := map[pair]bool{}
	var out Summary
	for _, e := range entries {
		if !applicable[e.Case][e.Environment] {
			out.Orphans++
			continue
		}
		key := pair{e.Case, e.Environment}
		hasEntry[key] = true
		if e.PlanDigest != digest {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, e.At)
		if err != nil {
			continue
		}
		if prev, ok := currentAt[key]; !ok || !at.Before(prev) {
			current[key] = e
			currentAt[key] = at
		}
	}

	for _, env := range environments {
		s := EnvSummary{Environment: env}
		for _, id := range caseOrder {
			if !applicable[id][env] {
				continue
			}
			s.Total++
			key := pair{id, env}
			if e, ok := current[key]; ok {
				switch e.Result {
				case ResultOK:
					s.OK++
				case ResultKO:
					s.KO++
				case ResultBlocked:
					s.Blocked++
				case ResultNA:
					s.NA++
				}
			} else if hasEntry[key] {
				s.Stale++
			} else {
				s.Unmarked++
			}
		}
		out.Environments = append(out.Environments, s)
	}
	return out
}

// Text es el texto canónico que copian las tres presentaciones: por entorno
// `E: <ok>/<total> ok` seguido de las categorías no nulas, separadas por `, `; los entornos
// se separan por `; `. Un entorno sin casos es `E: sin casos`.
func (s Summary) Text() string {
	parts := make([]string, 0, len(s.Environments))
	for _, e := range s.Environments {
		if e.Total == 0 {
			parts = append(parts, e.Environment+": sin casos")
			continue
		}
		items := []string{fmt.Sprintf("%d/%d ok", e.OK, e.Total)}
		add := func(n int, text string) {
			if n > 0 {
				items = append(items, fmt.Sprintf("%d %s", n, text))
			}
		}
		add(e.KO, "ko")
		add(e.Blocked, plural(e.Blocked, "bloqueado", "bloqueados"))
		add(e.NA, "n/a")
		add(e.Unmarked, "sin marcar")
		add(e.Stale, plural(e.Stale, "caducada", "caducadas"))
		parts = append(parts, e.Environment+": "+strings.Join(items, ", "))
	}
	return strings.Join(parts, "; ")
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
