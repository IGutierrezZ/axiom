package main

// suspendedRDDTUIJourneys lists the journeys that drive the Receipt-Driven
// Development selector in the TUI. C-01-c removed that selector from every
// build (the `rdd_harness` tag keeps the CLI switch, not the screens), so
// these journeys can no longer reach their first expectation. They stay
// defined, latent like the screens they drive, until C-01-e retires both.
var suspendedRDDTUIJourneys = map[string]string{
	"j121-rdd-tui-controls-global-mode":    "the TUI menu no longer offers the RDD entry (C-01-c)",
	"j127-customizable-install-rdd-choice": "the customizable installer no longer asks for RDD (C-01-c)",
}

func removeSuspendedRDDTUIJourneys(journeys []Journey) []Journey {
	active := make([]Journey, 0, len(journeys))
	for _, journey := range journeys {
		if _, suspended := suspendedRDDTUIJourneys[journey.ID]; !suspended {
			active = append(active, journey)
		}
	}
	return active
}
