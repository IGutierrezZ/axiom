package agentguidance

import (
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
)

// stopExitsAsset is the single source of the text that defines a stop and its
// three exits (REQ-26.8, DD-12). It is SDD-only (REQ-27.17): ODD has no stops,
// so the routing doctrine no longer projects it and only the SDD orchestrator
// sections do, which lets a text pin find the very same bytes in every runtime
// instead of copies that can drift apart.
const stopExitsAsset = "generic/stop-exits.md"

// StopExits returns the canonical text of the three-exit stops: what a stop is,
// its three exits in order, what "push" means, and who runs the remote
// operations. The result is deterministic and carries no heading of its own, so
// each consumer decides where it sits.
func StopExits() string {
	return strings.TrimSpace(assets.MustRead(stopExitsAsset))
}
