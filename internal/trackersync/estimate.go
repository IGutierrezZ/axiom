package trackersync

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// MaxEstimateHours es el tope de una estimación: lo comparten la bandera `--estimate` y las líneas
// de la sección `## Estimación por rol` de design.md (D14).
const MaxEstimateHours = 1000

// EstimateSectionTitle es el título exacto de la sección de estimaciones de design.md.
const EstimateSectionTitle = "## Estimación por rol"

// CodeEstimateInvalid es el código estable del rechazo de una estimación inválida.
const CodeEstimateInvalid = "estimate_invalid"

// EstimateError es el rechazo `estimate_invalid`: Line es la línea (o la bandera) culpable y Reason
// el motivo.
type EstimateError struct {
	Line   string
	Reason string
}

func (e *EstimateError) Error() string {
	return fmt.Sprintf("%s: %q: %s", CodeEstimateInvalid, e.Line, e.Reason)
}

var (
	estimateLineRe = regexp.MustCompile(`^- ([a-z][a-z0-9-]*): ([0-9]+(\.[0-9]{1,2})?) h$`)
	estimateFlagRe = regexp.MustCompile(`^([a-z][a-z0-9-]*)=([0-9]+(\.[0-9]{1,2})?)$`)
)

// checkEstimateHours valida el dominio común de una estimación: positiva y de 1000 h como mucho. La
// forma (un decimal con punto y dos decimales como mucho) la fijan las expresiones regulares.
func checkEstimateHours(raw string) (float64, string) {
	hours, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, "la estimación no es un número"
	}
	switch {
	case hours <= 0:
		return 0, "la estimación debe ser positiva"
	case hours > MaxEstimateHours:
		return 0, fmt.Sprintf("la estimación no puede superar %d h", MaxEstimateHours)
	}
	return hours, ""
}

// ParseEstimateFlag analiza un valor de `--estimate`: `<rol>=<horas>`, con horas decimales con punto,
// positivas, de dos decimales como mucho y de 1000 como máximo. El rol se canoniza en minúsculas.
func ParseEstimateFlag(value string) (role string, hours float64, err error) {
	trimmed := strings.TrimSpace(value)
	groups := estimateFlagRe.FindStringSubmatch(strings.ToLower(trimmed))
	if groups == nil {
		return "", 0, &EstimateError{Line: value, Reason: "la forma es <rol>=<horas>, con horas decimales con punto y dos decimales como mucho"}
	}
	hours, reason := checkEstimateHours(groups[2])
	if reason != "" {
		return "", 0, &EstimateError{Line: value, Reason: reason}
	}
	return groups[1], hours, nil
}

// ParseEstimates lee la sección `## Estimación por rol` de design.md: el título exacto, fuera de
// vallas de código, hasta el siguiente `## `. Toda línea de la sección que empieza por `- ` es de
// datos y debe casar con `- <rol>: <horas> h`; el resto del texto (la justificación) se ignora.
// Rechaza con *EstimateError, nombrando la línea, un valor no positivo o por encima del tope, un rol
// repetido o un rol fuera de roster. Sin sección no es un error: el resultado es un mapa vacío.
func ParseEstimates(design string, roster []string) (map[string]float64, error) {
	inRoster := make(map[string]bool, len(roster))
	for _, role := range roster {
		inRoster[role] = true
	}
	estimates := map[string]float64{}

	inFence, inSection := false, false
	var fenceMarker string
	for _, raw := range strings.Split(design, "\n") {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if marker, ok := fenceOpen(trimmed); ok {
			switch {
			case !inFence:
				inFence, fenceMarker = true, marker
			case strings.HasPrefix(trimmed, fenceMarker):
				inFence = false
			}
			continue
		}
		if inFence {
			continue
		}
		if strings.HasPrefix(line, "## ") || line == "##" {
			inSection = line == EstimateSectionTitle
			continue
		}
		if !inSection || !strings.HasPrefix(line, "- ") {
			continue
		}
		groups := estimateLineRe.FindStringSubmatch(line)
		if groups == nil {
			return nil, &EstimateError{Line: line, Reason: "la línea debe ser `- <rol>: <horas> h` con horas decimales con punto y dos decimales como mucho"}
		}
		hours, reason := checkEstimateHours(groups[2])
		if reason != "" {
			return nil, &EstimateError{Line: line, Reason: reason}
		}
		role := groups[1]
		if _, dup := estimates[role]; dup {
			return nil, &EstimateError{Line: line, Reason: fmt.Sprintf("el rol %s está repetido", role)}
		}
		if !inRoster[role] {
			return nil, &EstimateError{Line: line, Reason: fmt.Sprintf("el rol %s no pertenece al roster sellado (%s)", role, strings.Join(roster, ", "))}
		}
		estimates[role] = hours
	}
	return estimates, nil
}

// fenceOpen reconoce una valla de código (``` o ~~~) y devuelve su marcador.
func fenceOpen(trimmed string) (string, bool) {
	for _, marker := range []string{"```", "~~~"} {
		if strings.HasPrefix(trimmed, marker) {
			return marker, true
		}
	}
	return "", false
}
