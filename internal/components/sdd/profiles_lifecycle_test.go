package sdd

import ()

func profileSDDKeysForTest(name string) []string {
	suffix := ""
	if name != "" {
		suffix = "-" + name
	}
	keys := []string{"sdd-orchestrator" + suffix}
	for _, phase := range profilePhaseOrder {
		keys = append(keys, phase+suffix)
	}
	return keys
}
