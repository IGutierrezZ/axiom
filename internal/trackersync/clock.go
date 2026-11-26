package trackersync

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Reloj de jornada de `axiom integrations time propose` (eventos-ado, D7) y dominio de las horas de
// `time record` (D8). Son funciones puras: la zona horaria y el instante actual los inyecta quien
// llama, de modo que ni leen el reloj ni dependen de la zona del proceso.

// MaxHours es el tope de las horas de `time record --hours`.
const MaxHours = 1000

var hoursFlagRe = regexp.MustCompile(`^[0-9]+(\.[0-9]{1,2})?$`)

// ParseHours analiza `--hours` de `time record` (D8): un decimal con punto y como mucho dos
// decimales, con 0 < h <= 1000. minutes = round(h × 60), media hacia arriba, y debe ser >= 1.
func ParseHours(raw string) (hours float64, minutes int, err error) {
	trimmed := strings.TrimSpace(raw)
	if !hoursFlagRe.MatchString(trimmed) {
		return 0, 0, fmt.Errorf("--hours %q no es un decimal con punto y como mucho dos decimales", raw)
	}
	hours, err = strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("--hours %q no es un número", raw)
	}
	switch {
	case hours <= 0:
		return 0, 0, fmt.Errorf("--hours debe ser positivo y recibió %q", raw)
	case hours > MaxHours:
		return 0, 0, fmt.Errorf("--hours no puede superar %d h y recibió %q", MaxHours, raw)
	}
	minutes = int(math.Floor(hours*60 + 0.5))
	if minutes < 1 {
		return 0, 0, fmt.Errorf("--hours %q es menos de un minuto", raw)
	}
	return hours, minutes, nil
}

// ClockInput son los datos de ProposeHours.
type ClockInput struct {
	// StartedAt es el inicio registrado del rol; su hora local es el ancla de la ventana diaria.
	StartedAt time.Time
	// LastClose es el instante de la última imputación role.close/<rol>/* del rol; nil si no hay.
	LastClose *time.Time
	// Now es el instante de la consulta.
	Now time.Time
	// Loc es la zona horaria local (time.Local en producción).
	Loc *time.Location
	// WorkdayHours es la duración de la jornada y Workdays los días laborables ("mon".."sun").
	WorkdayHours int
	Workdays     []string
}

// Proposal es la propuesta de horas: la suma exacta, los minutos truncados y las horas redondeadas al
// cuarto de hora más próximo (media hacia arriba).
type Proposal struct {
	Since   time.Time
	Seconds int64
	Minutes int
	Hours   float64
}

var weekdayNames = map[string]time.Weekday{
	"sun": time.Sunday, "mon": time.Monday, "tue": time.Tuesday, "wed": time.Wednesday,
	"thu": time.Thursday, "fri": time.Friday, "sat": time.Saturday,
}

// ProposeHours calcula las horas imputables según D7:
//
//	since = max(started_at, última imputación role.close/<rol>/*)
//
// La ventana de cada día local D (de fecha(since)-1 a fecha(now)) empieza a la hora local de
// started_at, nunca a la de since, y dura WorkdayHours absolutas; cuenta solo si el día de la semana
// de su inicio es laborable, y aporta su intersección con [since, now). Una jornada que cruza la
// medianoche se imputa al día en que empieza.
func ProposeHours(in ClockInput) (Proposal, error) {
	if in.Loc == nil {
		return Proposal{}, errors.New("ProposeHours exige la zona horaria")
	}
	if in.WorkdayHours <= 0 {
		return Proposal{}, fmt.Errorf("workday_hours debe ser positivo y es %d", in.WorkdayHours)
	}
	working := map[time.Weekday]bool{}
	for _, name := range in.Workdays {
		day, ok := weekdayNames[strings.ToLower(strings.TrimSpace(name))]
		if !ok {
			return Proposal{}, fmt.Errorf("día laborable %q desconocido", name)
		}
		working[day] = true
	}

	since := in.StartedAt
	if in.LastClose != nil && in.LastClose.After(since) {
		since = *in.LastClose
	}
	proposal := Proposal{Since: since}
	if !in.Now.After(since) {
		return proposal, nil
	}

	anchor := in.StartedAt.In(in.Loc)
	hour, min, sec := anchor.Clock()
	first := since.In(in.Loc)
	last := in.Now.In(in.Loc)
	start := time.Date(first.Year(), first.Month(), first.Day()-1, 0, 0, 0, 0, in.Loc)
	end := time.Date(last.Year(), last.Month(), last.Day(), 0, 0, 0, 0, in.Loc)
	length := time.Duration(in.WorkdayHours) * time.Hour

	var total time.Duration
	for day := start; !day.After(end); day = time.Date(day.Year(), day.Month(), day.Day()+1, 0, 0, 0, 0, in.Loc) {
		windowStart := time.Date(day.Year(), day.Month(), day.Day(), hour, min, sec, anchor.Nanosecond(), in.Loc)
		if !working[windowStart.Weekday()] {
			continue
		}
		windowEnd := windowStart.Add(length)
		from, to := windowStart, windowEnd
		if since.After(from) {
			from = since
		}
		if in.Now.Before(to) {
			to = in.Now
		}
		if to.After(from) {
			total += to.Sub(from)
		}
	}

	proposal.Seconds = int64(total / time.Second)
	proposal.Minutes = int(proposal.Seconds / 60)
	proposal.Hours = float64((2*proposal.Minutes+15)/30) * 15 / 60
	return proposal, nil
}
