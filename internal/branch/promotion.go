package branch

// Promotion es el modo de promoción de las ramas de un repositorio de dos etapas
// (`git.main_branch` distinta de la rama de integración): `branch` crea la rama del
// cambio desde la rama principal y su segunda PR va a ella; `release` la crea desde
// la integración y el paso a la rama principal es la release. `ask` es valor de
// política y, en el plan, la marca de «pendiente de preguntar»: nunca se registra.
type Promotion string

const (
	PromotionBranch  Promotion = "branch"
	PromotionRelease Promotion = "release"
	PromotionAsk     Promotion = "ask"
)

// Orígenes de la política declarada que ya conoce `Policy.For`; el resto de orígenes
// de la promoción los añade la resolución (promotion_resolve.go).
const (
	PromotionSourceRepository = "repository"
	PromotionSourceGit        = "git"
)
