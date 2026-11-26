package trackersync

import (
	"testing"
	"time"
)

var weekdays = []string{"mon", "tue", "wed", "thu", "fri"}

func at(t *testing.T, loc *time.Location, layout, value string) time.Time {
	t.Helper()
	parsed, err := time.ParseInLocation(layout, value, loc)
	if err != nil {
		t.Fatalf("fecha %q: %v", value, err)
	}
	return parsed
}

const layout = "2006-01-02 15:04"

func madrid(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Skipf("sin base de zonas horarias: %v", err)
	}
	return loc
}

// prov:INT:7: los tres escenarios de la spec.
func TestProposeHoursSpecScenarios(t *testing.T) {
	loc := time.FixedZone("UTC+2", 2*3600)
	cases := []struct {
		name          string
		start, now    string
		wantH         float64
		wantMin       int
		workdayHours  int
		workdaysUsing []string
	}{
		{"lunes 09:00 a martes 12:00 da 11 h", "2026-11-23 09:00", "2026-11-24 12:00", 11, 660, 8, weekdays},
		{"00:00 a 23:59 del lunes da 8 h", "2026-11-23 00:00", "2026-11-23 23:59", 8, 480, 8, weekdays},
		{"sábado a domingo da 0 h", "2026-11-28 10:00", "2026-11-29 18:00", 0, 0, 8, weekdays},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ProposeHours(ClockInput{StartedAt: at(t, loc, layout, tc.start), Now: at(t, loc, layout, tc.now), Loc: loc,
				WorkdayHours: tc.workdayHours, Workdays: tc.workdaysUsing})
			if err != nil {
				t.Fatal(err)
			}
			if got.Hours != tc.wantH || got.Minutes != tc.wantMin {
				t.Errorf("propuesta %+v, quiero %v h y %d min", got, tc.wantH, tc.wantMin)
			}
		})
	}
}

// El segundo cierre cuenta desde la última imputación, con la ventana anclada a la hora de inicio.
func TestProposeHoursSecondCloseCountsFromTheLastEntry(t *testing.T) {
	loc := time.FixedZone("UTC+1", 3600)
	start := at(t, loc, layout, "2026-11-23 09:00")
	last := at(t, loc, layout, "2026-11-24 12:00")

	got, err := ProposeHours(ClockInput{StartedAt: start, LastClose: &last, Now: at(t, loc, layout, "2026-11-25 11:00"), Loc: loc, WorkdayHours: 8, Workdays: weekdays})
	if err != nil {
		t.Fatal(err)
	}
	// Martes 12:00-17:00 (5 h) más miércoles 09:00-11:00 (2 h).
	if got.Hours != 7 || got.Minutes != 420 || !got.Since.Equal(last) {
		t.Errorf("propuesta %+v, quiero 7 h desde la última imputación", got)
	}

	// Una imputación anterior al inicio no adelanta el intervalo.
	old := at(t, loc, layout, "2026-11-20 10:00")
	got, err = ProposeHours(ClockInput{StartedAt: start, LastClose: &old, Now: at(t, loc, layout, "2026-11-23 11:00"), Loc: loc, WorkdayHours: 8, Workdays: weekdays})
	if err != nil || !got.Since.Equal(start) || got.Hours != 2 {
		t.Errorf("propuesta %+v (%v)", got, err)
	}
}

// Una jornada anclada de noche cruza la medianoche y se imputa al día en que empieza.
func TestProposeHoursNightAnchorCrossesMidnight(t *testing.T) {
	loc := time.FixedZone("UTC", 0)
	start := at(t, loc, layout, "2026-11-23 22:00") // lunes

	got, err := ProposeHours(ClockInput{StartedAt: start, Now: at(t, loc, layout, "2026-11-24 07:00"), Loc: loc, WorkdayHours: 8, Workdays: []string{"mon"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Hours != 8 {
		t.Errorf("lunes 22:00 a martes 06:00 se imputa al lunes: %+v", got)
	}

	// El martes no es laborable aunque la ventana del lunes llegue a él.
	got, _ = ProposeHours(ClockInput{StartedAt: start, Now: at(t, loc, layout, "2026-11-24 03:00"), Loc: loc, WorkdayHours: 8, Workdays: []string{"tue"}})
	if got.Hours != 0 {
		t.Errorf("solo cuenta el día en que empieza la ventana: %+v", got)
	}
}

// Los días del cambio de hora de Europe/Madrid siguen sumando las horas absolutas de la jornada.
func TestProposeHoursAcrossDaylightSavingChanges(t *testing.T) {
	loc := madrid(t)
	cases := []struct {
		name       string
		start, now string
		want       float64
	}{
		// Último domingo de marzo de 2026 (29): 02:00 pasa a las 03:00.
		{"marzo: ventana de día entero", "2026-03-28 09:00", "2026-03-30 00:00", 16},
		{"marzo: la ventana nocturna sigue durando 8 h absolutas", "2026-03-28 22:00", "2026-03-29 07:00", 8},
		{"marzo: una hora menos de reloj de pared", "2026-03-28 22:00", "2026-03-29 06:00", 7},
		// Último domingo de octubre de 2026 (25): 03:00 pasa a las 02:00.
		{"octubre: ventana de día entero", "2026-10-24 09:00", "2026-10-26 00:00", 16},
		{"octubre: la ventana nocturna termina una hora antes de reloj", "2026-10-24 22:00", "2026-10-25 12:00", 8},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ProposeHours(ClockInput{StartedAt: at(t, loc, layout, tc.start), Now: at(t, loc, layout, tc.now), Loc: loc,
				WorkdayHours: 8, Workdays: []string{"sat", "sun", "mon"}})
			if err != nil {
				t.Fatal(err)
			}
			if got.Hours != tc.want {
				t.Errorf("propuesta %+v, quiero %v h", got, tc.want)
			}
		})
	}
}

// Los minutos se truncan y las horas se redondean al cuarto de hora más próximo (media hacia arriba).
func TestProposeHoursRoundsToQuarterHours(t *testing.T) {
	loc := time.FixedZone("UTC", 0)
	start := at(t, loc, layout, "2026-11-23 09:00")
	cases := []struct {
		offset  time.Duration
		minutes int
		hours   float64
	}{
		{7*time.Minute + 29*time.Second, 7, 0},
		{8 * time.Minute, 8, 0.25},
		{22*time.Minute + 59*time.Second, 22, 0.25},
		{37*time.Minute + 59*time.Second, 37, 0.5},
		{23 * time.Minute, 23, 0.5},
		{59*time.Minute + 59*time.Second, 59, 1},
	}
	for _, tc := range cases {
		got, err := ProposeHours(ClockInput{StartedAt: start, Now: start.Add(tc.offset), Loc: loc, WorkdayHours: 8, Workdays: weekdays})
		if err != nil {
			t.Fatal(err)
		}
		if got.Minutes != tc.minutes || got.Hours != tc.hours {
			t.Errorf("%v: %d min y %v h, quiero %d min y %v h", tc.offset, got.Minutes, got.Hours, tc.minutes, tc.hours)
		}
	}
}

func TestProposeHoursRejectsBadInput(t *testing.T) {
	loc := time.UTC
	now := time.Now()
	for name, in := range map[string]ClockInput{
		"sin zona":        {StartedAt: now, Now: now, WorkdayHours: 8, Workdays: weekdays},
		"jornada cero":    {StartedAt: now, Now: now, Loc: loc, Workdays: weekdays},
		"día desconocido": {StartedAt: now, Now: now, Loc: loc, WorkdayHours: 8, Workdays: []string{"lunes"}},
	} {
		if _, err := ProposeHours(in); err == nil {
			t.Errorf("%s: debe fallar", name)
		}
	}
	if got, err := ProposeHours(ClockInput{StartedAt: now, Now: now.Add(-time.Hour), Loc: loc, WorkdayHours: 8, Workdays: weekdays}); err != nil || got.Hours != 0 {
		t.Errorf("un ahora anterior al inicio no suma: %+v (%v)", got, err)
	}
}

func TestParseHoursDomain(t *testing.T) {
	good := map[string]struct {
		hours   float64
		minutes int
	}{"2": {2, 120}, "0.5": {0.5, 30}, "1.25": {1.25, 75}, "1000": {1000, 60000}, "0.01": {0.01, 1}, " 3 ": {3, 180}}
	for raw, want := range good {
		hours, minutes, err := ParseHours(raw)
		if err != nil || hours != want.hours || minutes != want.minutes {
			t.Errorf("%q: %v, %d, %v; quiero %+v", raw, hours, minutes, err, want)
		}
	}
	for _, raw := range []string{"0", "0.00", "-1", "mucho", "", "1.234", "1,5", "1000.01", "1001", "+2", "1e2"} {
		if _, _, err := ParseHours(raw); err == nil {
			t.Errorf("%q debe rechazarse", raw)
		}
	}
}
