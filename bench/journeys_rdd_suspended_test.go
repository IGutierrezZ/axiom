package main

import "testing"

func TestCorpusDoesNotRunSuspendedRDDTUIJourneys(t *testing.T) {
	for _, journey := range allDeclaredJourneys() {
		if reason, suspended := suspendedRDDTUIJourneys[journey.ID]; suspended {
			t.Errorf("journey %q is still registered although %s", journey.ID, reason)
		}
	}
}

func TestSuspendedRDDTUIJourneysStillExist(t *testing.T) {
	defined := map[string]bool{}
	for _, journey := range append(issue3766Journeys(), issue4377Journeys()...) {
		defined[journey.ID] = true
	}
	for id := range suspendedRDDTUIJourneys {
		if !defined[id] {
			t.Errorf("suspended journey %q is no longer defined; drop it from suspendedRDDTUIJourneys", id)
		}
	}
}
