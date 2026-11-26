package trackersync

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CloseEntry es lo que el cálculo de cierres necesita de una imputación ya registrada.
type CloseEntry struct {
	Source     string
	RecordedAt time.Time
}

// CloseSourcePrefix es el prefijo del `source` de las imputaciones de cierre del rol: `role.close/<rol>/`.
func CloseSourcePrefix(role string) string { return "role.close/" + role + "/" }

// closeNumber devuelve el correlativo <n> de un `source` de cierre del rol, o 0 si no lo es.
func closeNumber(role, source string) int {
	rest, ok := strings.CutPrefix(source, CloseSourcePrefix(role))
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(rest)
	if err != nil || n < 1 {
		return 0
	}
	return n
}

// NextCloseSource calcula el `source` único del siguiente cierre del rol, `role.close/<rol>/<n>`, con
// <n> correlativo desde 1: uno más que el mayor ya registrado.
func NextCloseSource(role string, entries []CloseEntry) string {
	highest := 0
	for _, e := range entries {
		if n := closeNumber(role, e.Source); n > highest {
			highest = n
		}
	}
	return fmt.Sprintf("%s%d", CloseSourcePrefix(role), highest+1)
}

// LastCloseAt devuelve el instante de la última imputación de cierre del rol, si la hay.
func LastCloseAt(role string, entries []CloseEntry) *time.Time {
	var last *time.Time
	for _, e := range entries {
		if closeNumber(role, e.Source) == 0 {
			continue
		}
		at := e.RecordedAt
		if last == nil || at.After(*last) {
			last = &at
		}
	}
	return last
}
