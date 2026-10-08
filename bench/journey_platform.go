package main

// journeyUnsupportedReason says why a journey cannot run on goos, or "" when it
// can. The declaration is Journey.ExecutesPOSIXShims; Windows is the only host
// that cannot execute a `#!/bin/sh` shim.
func journeyUnsupportedReason(journey Journey, goos string) string {
	if journey.ExecutesPOSIXShims && goos == "windows" {
		return "the journey executes POSIX shell shims (#!/bin/sh) that Windows cannot run; without them the product would resolve a real agent instead of the sandbox's"
	}
	return ""
}

// unsupportedJourneyResult is the explicit `unsupported` outcome: no step ran,
// the reason is in the report, and aggregate excludes it from every total.
func unsupportedJourneyResult(journey Journey, reason string) JourneyResult {
	return JourneyResult{
		ID:               journey.ID,
		Title:            journey.Title,
		Source:           journey.Source,
		Status:           StatusUnsupported,
		UnsupportedSteps: []string{"journey (" + reason + ")"},
		Metrics:          newAccumulator().metrics(""),
		Commands:         []CommandRecord{},
	}
}
