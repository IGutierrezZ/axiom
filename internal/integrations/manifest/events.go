package manifest

// EventType identifica un evento del ciclo de vida de un cambio que un plugin de integración
// puede escuchar. La lista es cerrada: ningún otro nombre es válido en un manifiesto.
type EventType string

const (
	// EventChangeStart solo existe en el catálogo: lo dispara C-07-d, no este cambio.
	EventChangeStart        EventType = "change.start"
	EventGateSpecApproved   EventType = "gate.spec.approved"
	EventGateDesignApproved EventType = "gate.design.approved"
	EventRoleStart          EventType = "role.start"
	EventRoleApplyApproved  EventType = "role.apply.approved"
	EventArchive            EventType = "archive"
)

// Events devuelve la lista cerrada de eventos en orden estable.
func Events() []EventType {
	return []EventType{
		EventChangeStart,
		EventGateSpecApproved,
		EventGateDesignApproved,
		EventRoleStart,
		EventRoleApplyApproved,
		EventArchive,
	}
}

// ValidEvent indica si t pertenece a la lista cerrada.
func ValidEvent(t EventType) bool {
	for _, e := range Events() {
		if e == t {
			return true
		}
	}
	return false
}
