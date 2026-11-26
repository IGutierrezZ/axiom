package model

// DefaultComponents returns the managed components Axiom installs when the
// person does not choose any: Engram, SDD, skills, Context7, permissions and the
// persona. It is always editable in the component tree.
//
// CodeGraph no forma parte del conjunto por defecto: es una herramienta de la
// comunidad que se elige aparte (Selection.CommunityTools) y nunca viene marcada
// por defecto.
func DefaultComponents() []ComponentID {
	return []ComponentID{
		ComponentEngram,
		ComponentSDD,
		ComponentSkills,
		ComponentContext7,
		ComponentPermission,
		ComponentPersona,
	}
}
