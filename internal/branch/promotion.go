package branch

import (
	"errors"
	"fmt"
	"strings"
)

// La promoción de ramas (change promocion-ramas, C-02-a): un repositorio con dos ramas
// permanentes (`git.main_branch` distinta de la rama de integración) promueve el
// trabajo en dos etapas. Este fichero es el dominio puro: los tokens, la resolución del
// modo de cada repositorio, el destino que confirma la fusión en `done` y la etapa de
// `status`. No lee el disco ni ejecuta git: el motor reúne los hechos y traduce los
// resultados, de modo que todo se prueba con tablas.

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

// Orígenes del modo de promoción (RecordRepo.PromotionSource y RepoPlan.PromotionSource).
const (
	// PromotionSourceAnswer es `--promotion` cuando la política del repositorio es `ask`.
	PromotionSourceAnswer = "answer"
	// PromotionSourceFlag es `--promotion` cuando la política no es `ask`.
	PromotionSourceFlag = "flag"
	// PromotionSourceRepository es `repositories[].promotion`.
	PromotionSourceRepository = "repository"
	// PromotionSourceGit es `git.promotion`.
	PromotionSourceGit = "git"
	// PromotionSourceDefault es el defecto `branch`: nada declarado.
	PromotionSourceDefault = "default"
	// PromotionSourceSnapshot es el modo tomado de la instantánea del kickoff porque no
	// había registro.
	PromotionSourceSnapshot = "snapshot"
	// PromotionSourceLegacy es un registro o una instantánea sin el campo y con la rama del
	// cambio ya creada: una sola etapa, que equivale a `release`.
	PromotionSourceLegacy = "legacy"
)

// Etapas de `status` de la rama del cambio de un repositorio en modo `branch`: adónde va
// su siguiente PR.
const (
	PromotionStageIntegration = "integration"
	PromotionStageMain        = "main"
)

// promotionLabels son las etiquetas en castellano de los tokens de la promoción. Van en un
// mapa propio y no en `labels`: allí `flag` ya significa «bandera --mode» y `ask`,
// «preguntar» del modo de trabajo, y mezclarlos daría una etiqueta falsa.
var promotionLabels = map[string]string{
	string(PromotionBranch):  "rama a rama",
	string(PromotionRelease): "por release",
	string(PromotionAsk):     "preguntar",

	PromotionSourceAnswer:     "respuesta registrada",
	PromotionSourceFlag:       "bandera --promotion",
	PromotionSourceRepository: "entrada de repositories",
	PromotionSourceGit:        "bloque git",
	PromotionSourceDefault:    "valor por defecto",
	PromotionSourceSnapshot:   "instantánea del kickoff",
	PromotionSourceLegacy:     "cambio anterior (una sola etapa)",

	PromotionStageIntegration: "hacia la integración",
	PromotionStageMain:        "hacia la rama principal",
}

// PromotionLabel devuelve la etiqueta en castellano de un token de la promoción (modo,
// origen o etapa), o el propio token si no lo conoce.
func PromotionLabel(token string) string {
	if label, known := promotionLabels[token]; known {
		return label
	}
	return token
}

// KnownPromotionSource dice si source es un origen del vocabulario cerrado; la cadena vacía
// (sin origen) también lo es.
func KnownPromotionSource(source string) bool {
	switch source {
	case "", PromotionSourceAnswer, PromotionSourceFlag, PromotionSourceRepository, PromotionSourceGit,
		PromotionSourceDefault, PromotionSourceSnapshot, PromotionSourceLegacy:
		return true
	}
	return false
}

// PromotionRequest es una petición de `--promotion`: Repository vacío la aplica a todos los
// repositorios del plan; con nombre, solo a ese. Mode es `branch` o `release`.
type PromotionRequest struct {
	Repository string
	Mode       Promotion
}

// ParsePromotionRequests analiza los valores de `--promotion` (`<modo>` o `<repo>=<modo>`).
// Rechaza, con un error de uso: un valor vacío, un `<repo>` vacío, un modo vacío, un modo
// distinto de `branch` y `release` (`ask` es política, no una respuesta), dos valores sin
// cualificar y el mismo `<repo>` dos veces. Conserva el orden dado.
func ParsePromotionRequests(values []string) ([]PromotionRequest, error) {
	var requests []PromotionRequest
	unqualified := false
	seen := map[string]bool{}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return nil, errors.New("--promotion no admite un valor vacío: usa branch, release, <repo>=branch o <repo>=release")
		}
		repository, mode := "", value
		if before, after, qualified := strings.Cut(value, "="); qualified {
			repository, mode = strings.TrimSpace(before), after
			if repository == "" {
				return nil, fmt.Errorf("--promotion %q no nombra el repositorio antes de «=»: usa <repo>=branch o <repo>=release", value)
			}
		}
		mode = strings.TrimSpace(mode)
		switch Promotion(mode) {
		case PromotionBranch, PromotionRelease:
		case PromotionAsk:
			return nil, fmt.Errorf("--promotion %q no es una respuesta válida: «ask» es una política; usa %s o %s", value, PromotionBranch, PromotionRelease)
		case "":
			return nil, fmt.Errorf("--promotion %q no indica el modo: usa %s o %s", value, PromotionBranch, PromotionRelease)
		default:
			return nil, fmt.Errorf("--promotion %q: el modo «%s» no es una respuesta válida: usa %s o %s", value, mode, PromotionBranch, PromotionRelease)
		}
		if repository == "" {
			if unqualified {
				return nil, errors.New("--promotion sin <repo> solo se admite una vez: cualifica cada valor con <repo>=<modo>")
			}
			unqualified = true
		} else {
			if seen[repository] {
				return nil, fmt.Errorf("--promotion repite el repositorio «%s»: indica un solo modo por repositorio", repository)
			}
			seen[repository] = true
		}
		requests = append(requests, PromotionRequest{Repository: repository, Mode: Promotion(mode)})
	}
	return requests, nil
}

// PromotionFacts son los hechos de un repositorio que entran a ResolvePromotion.
type PromotionFacts struct {
	// Exempt es el repositorio de specs de un multirepo; TwoStage dice que el repositorio
	// tiene `main_branch` distinta de su integración. Un repositorio exento o de una etapa
	// no tiene modo.
	Exempt, TwoStage bool
	// Recorded es la entrada del registro local del repositorio; nil, sin entrada.
	Recorded *RecordedPromotion
	// Snapshot es la entrada de la instantánea del kickoff; nil, sin entrada.
	Snapshot *SnapshotPromotion
	// ChangeBranchExists solo cuenta para la regla `legacy` de la instantánea.
	ChangeBranchExists bool
	// Requested es el `--promotion` aplicable (el cualificado prevalece sobre el que no lo
	// está); vacío si no hay.
	Requested Promotion
	// Declared es RepoPolicy.Promotion y DeclaredSource su capa (`repository` o `git`).
	Declared       Promotion
	DeclaredSource string
}

// RecordedPromotion es lo que el registro local sabe de la promoción de un repositorio.
type RecordedPromotion struct {
	Mode   Promotion
	Source string
	// HasChangeBranch dice que el registro lista una rama `change` para el repositorio.
	HasChangeBranch bool
}

// SnapshotPromotion es lo que la instantánea del kickoff sabe de la promoción de un
// repositorio.
type SnapshotPromotion struct {
	Mode Promotion
	// SealedWithMain dice que la instantánea se selló con `main_branch` declarada
	// (GitSnapshot.MainBranch != ""): es el marcador de la regla `legacy`.
	SealedWithMain bool
}

// ResolvedPromotion es el modo resuelto de un repositorio: lo que entra al plan
// (PlanInput.Promotions, por raíz git).
type ResolvedPromotion struct {
	// Mode es "" (una etapa o exento), `branch`, `release` o `ask` (pendiente de preguntar).
	Mode   Promotion
	Source string
	// Recorded dice que sale del registro: la instantánea lo copia solo entonces.
	Recorded bool
}

// PromotionResolution es el resultado de ResolvePromotion. Un Materialized no vacío es el
// modo con el que el cambio ya está materializado y que contradice Requested: conflicto.
type PromotionResolution struct {
	Resolved     ResolvedPromotion
	Materialized Promotion
}

// ResolvePromotion decide el modo de promoción de un repositorio con esta precedencia:
// repositorio exento o de una etapa (sin modo), registro, instantánea, `--promotion`,
// `repositories[].promotion` o `git.promotion`, `ask` pendiente y el defecto `branch`. Un
// registro o una instantánea con modo materializan el cambio: si `--promotion` pide otro,
// es un conflicto; si pide el mismo, es idempotente. Es pura.
func ResolvePromotion(f PromotionFacts) PromotionResolution {
	if f.Exempt || !f.TwoStage {
		return PromotionResolution{}
	}

	if recorded := f.Recorded; recorded != nil {
		switch {
		case recorded.Mode != "":
			return materialized(f, ResolvedPromotion{Mode: recorded.Mode, Source: recorded.Source, Recorded: true})
		case recorded.HasChangeBranch:
			return materialized(f, ResolvedPromotion{Mode: PromotionRelease, Source: PromotionSourceLegacy, Recorded: true})
		}
	}

	if snapshot := f.Snapshot; snapshot != nil {
		switch {
		case snapshot.Mode != "":
			return materialized(f, ResolvedPromotion{Mode: snapshot.Mode, Source: PromotionSourceSnapshot})
		case !snapshot.SealedWithMain && f.ChangeBranchExists:
			return materialized(f, ResolvedPromotion{Mode: PromotionRelease, Source: PromotionSourceLegacy})
		}
	}

	if f.Requested != "" {
		source := PromotionSourceFlag
		if f.Declared == PromotionAsk {
			source = PromotionSourceAnswer
		}
		return PromotionResolution{Resolved: ResolvedPromotion{Mode: f.Requested, Source: source}}
	}

	switch f.Declared {
	case PromotionBranch, PromotionRelease:
		return PromotionResolution{Resolved: ResolvedPromotion{Mode: f.Declared, Source: f.DeclaredSource}}
	case PromotionAsk:
		return PromotionResolution{Resolved: ResolvedPromotion{Mode: PromotionAsk}}
	}
	return PromotionResolution{Resolved: ResolvedPromotion{Mode: PromotionBranch, Source: PromotionSourceDefault}}
}

// materialized completa un modo ya materializado con el conflicto frente a `--promotion`.
func materialized(f PromotionFacts, resolved ResolvedPromotion) PromotionResolution {
	resolution := PromotionResolution{Resolved: resolved}
	if f.Requested != "" && f.Requested != resolved.Mode {
		resolution.Materialized = resolved.Mode
	}
	return resolution
}

// MergeDestination es la referencia contra la que `done` busca la fusión de una rama del
// plan, y si la capa 3 (evidencia `pr_merged` de `gates.yaml`) puede acreditarla:
//   - rama de rol: la de seguimiento (planned.From, local), con capa 3, como siempre;
//   - rama de archivado: la integración, sin capa 3, como siempre;
//   - rama del cambio de un repositorio de dos etapas en modo `branch`, o con el modo aún
//     pendiente (`ask`): la rama principal y sin capa 3, porque `pr_merged` acredita la PR
//     del cambio hacia la integración y nunca la segunda; con el modo pendiente se toma el
//     destino más exigente, porque borrar una rama es irreversible y `--force` sigue siendo
//     la salida;
//   - cualquier otro caso: la integración, con capa 3.
func MergeDestination(repo RepoPlan, planned BranchPlan) (destination string, prMergedApplies bool) {
	switch planned.Kind {
	case KindRole:
		return planned.From, true
	case KindArchive:
		return repo.IntegrationBranch, false
	}
	if repo.TwoStage() && (repo.Promotion == PromotionBranch || repo.Promotion == PromotionAsk) {
		return repo.MainBranch, false
	}
	return repo.IntegrationBranch, true
}

// PromotionStage decide adónde va la siguiente PR de la rama del cambio de un repositorio de
// dos etapas en modo `branch` (RF-BRA-30), sin fingir certeza. aheadOfMain son los commits
// que la rama lleva por delante de la rama principal (nil si no se calculó) y evidence, las
// capas 1 y 2 evaluadas contra la integración:
//   - nil o 0 por delante: `integration`; la rama no lleva nada que la principal no tenga y
//     no puede parecer ya fusionada porque la principal suela ser ascendiente de la
//     integración;
//   - fusionada en la integración por la capa 1, o por la capa 2 con soporte: `main`;
//   - sin ascendencia y sin soporte de la capa 2: `integration` e inconclusive, para que
//     quien llama avise;
//   - en cualquier otro caso: `integration`.
//
// target es la rama destino de esa PR.
func PromotionStage(repo RepoPlan, aheadOfMain *int, evidence MergeEvidence) (stage, target string, inconclusive bool) {
	switch {
	case aheadOfMain == nil || *aheadOfMain == 0:
		return PromotionStageIntegration, repo.IntegrationBranch, false
	case evidence.Ancestry, evidence.TreeLayerSupported && evidence.TreeEquivalent:
		return PromotionStageMain, repo.MainBranch, false
	case !evidence.TreeLayerSupported:
		return PromotionStageIntegration, repo.IntegrationBranch, true
	}
	return PromotionStageIntegration, repo.IntegrationBranch, false
}
