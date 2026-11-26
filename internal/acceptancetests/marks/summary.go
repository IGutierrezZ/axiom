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
	c := classify(battery, environments, entries, digest)
	out := Summary{Orphans: c.orphans}
	for _, env := range environments {
		s := EnvSummary{Environment: env}
		for _, id := range c.caseOrder {
			if !c.applicable[id][env] {
				continue
			}
			s.Total++
			key := pair{id, env}
			if e, ok := c.current[key]; ok {
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
			} else if c.hasEntry[key] {
				s.Stale++
			} else {
				s.Unmarked++
			}
		}
		out.Environments = append(out.Environments, s)
	}
	return out
}

// Estados de una celda sin marca vigente.
const (
	StateStale    = "stale"
	StateUnmarked = "unmarked"
)

// Cell es el estado de un par aplicable (caso, entorno). State es el resultado de la marca
// vigente (`ok`, `ko`, `blocked`, `n/a`), `stale` o `unmarked`. Current es la marca vigente;
// Stale es la última marca caducada del par (la última del fichero), solo sin vigente.
type Cell struct {
	Case        string
	Environment string
	State       string
	Current     *Entry
	Stale       *Entry
}

// Cells devuelve las celdas aplicables con la misma clasificación que Summarize (D4): casos en
// el orden de pruebas.md y, dentro de cada caso, entornos en el orden declarado.
func Cells(battery acceptancetests.Battery, environments []string, entries []Entry, digest string) []Cell {
	c := classify(battery, environments, entries, digest)
	var out []Cell
	for _, id := range c.caseOrder {
		for _, env := range environments {
			if !c.applicable[id][env] {
				continue
			}
			key := pair{id, env}
			cell := Cell{Case: id, Environment: env, State: StateUnmarked}
			if e, ok := c.current[key]; ok {
				entry := e
				cell.Current, cell.State = &entry, string(e.Result)
			} else if stale, ok := c.lastStale[key]; ok {
				entry := stale
				cell.Stale, cell.State = &entry, StateStale
			}
			out = append(out, cell)
		}
	}
	return out
}

type classification struct {
	applicable map[string]map[string]bool
	caseOrder  []string
	current    map[pair]Entry
	lastStale  map[pair]Entry
	hasEntry   map[pair]bool
	orphans    int
}

// classify es el cálculo único de D4: entornos aplicables por caso, marca vigente por par y
// entradas huérfanas.
func classify(battery acceptancetests.Battery, environments []string, entries []Entry, digest string) classification {
	declared := make(map[string]bool, len(environments))
	for _, env := range environments {
		declared[env] = true
	}
	c := classification{
		applicable: map[string]map[string]bool{},
		current:    map[pair]Entry{},
		lastStale:  map[pair]Entry{},
		hasEntry:   map[pair]bool{},
	}
	for _, cs := range battery.Cases {
		if !caseIDPattern.MatchString(cs.ID) {
			continue
		}
		if _, seen := c.applicable[cs.ID]; !seen {
			c.applicable[cs.ID] = map[string]bool{}
			c.caseOrder = append(c.caseOrder, cs.ID)
		}
		for _, env := range cs.Environments {
			if declared[env] {
				c.applicable[cs.ID][env] = true
			}
		}
	}
	currentAt := map[pair]time.Time{}
	for _, e := range entries {
		if !c.applicable[e.Case][e.Environment] {
			c.orphans++
			continue
		}
		key := pair{e.Case, e.Environment}
		c.hasEntry[key] = true
		if e.PlanDigest != digest {
			c.lastStale[key] = e
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, e.At)
		if err != nil {
			continue
		}
		if prev, ok := currentAt[key]; !ok || !at.Before(prev) {
			c.current[key] = e
			currentAt[key] = at
		}
	}
	return c
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
