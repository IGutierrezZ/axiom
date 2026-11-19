package manifest

// Capability describe qué sabe hacer un plugin contra su tracker. La lista es cerrada.
type Capability string

const (
	CapItemsRead       Capability = "items.read"
	CapItemsTransition Capability = "items.transition"
	CapItemsCreate     Capability = "items.create"
	CapRelations       Capability = "relations"
	CapAssignment      Capability = "assignment"
	CapEstimation      Capability = "estimation"
	CapTime            Capability = "time"
	CapTags            Capability = "tags"
	CapAttachments     Capability = "attachments"
	CapVCSLinks        Capability = "vcs_links"
	CapDiscovery       Capability = "discovery"
)

// Capabilities devuelve la lista cerrada de capacidades en orden estable.
func Capabilities() []Capability {
	return []Capability{
		CapItemsRead,
		CapItemsTransition,
		CapItemsCreate,
		CapRelations,
		CapAssignment,
		CapEstimation,
		CapTime,
		CapTags,
		CapAttachments,
		CapVCSLinks,
		CapDiscovery,
	}
}

// ValidCapability indica si c pertenece a la lista cerrada.
func ValidCapability(c Capability) bool {
	for _, k := range Capabilities() {
		if k == c {
			return true
		}
	}
	return false
}
