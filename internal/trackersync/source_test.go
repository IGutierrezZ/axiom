package trackersync

import (
	"testing"
	"time"
)

func TestNextCloseSourceIsCorrelativePerRole(t *testing.T) {
	entries := []CloseEntry{
		{Source: "role.close/fullstack/1"},
		{Source: "role.close/fullstack/3"},
		{Source: "role.close/web/7"},
		{Source: "role.close/fullstack/x"},
		{Source: "role.start/fullstack"},
	}
	for role, want := range map[string]string{
		"fullstack": "role.close/fullstack/4",
		"web":       "role.close/web/8",
		"core":      "role.close/core/1",
	} {
		if got := NextCloseSource(role, entries); got != want {
			t.Errorf("%s: %s, quiero %s", role, got, want)
		}
	}
	if got := NextCloseSource("fullstack", nil); got != "role.close/fullstack/1" {
		t.Errorf("sin imputaciones empieza en 1: %s", got)
	}
}

func TestLastCloseAtIgnoresOtherRolesAndSources(t *testing.T) {
	base := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	entries := []CloseEntry{
		{Source: "role.close/fullstack/1", RecordedAt: base},
		{Source: "role.close/fullstack/2", RecordedAt: base.Add(48 * time.Hour)},
		{Source: "role.close/web/1", RecordedAt: base.Add(96 * time.Hour)},
		{Source: "otra", RecordedAt: base.Add(200 * time.Hour)},
	}
	got := LastCloseAt("fullstack", entries)
	if got == nil || !got.Equal(base.Add(48*time.Hour)) {
		t.Errorf("última imputación de fullstack: %v", got)
	}
	if LastCloseAt("core", entries) != nil {
		t.Errorf("un rol sin cierres no tiene última imputación")
	}
}
