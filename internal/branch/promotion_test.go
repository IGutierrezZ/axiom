package branch

import (
	"reflect"
	"strings"
	"testing"
)

// Las pruebas de la promoción de ramas son tablas puras, sin git ni disco: la resolución, el
// destino de `done` y la etapa de `status` son funciones sobre hechos.

func TestPromotionLabelsAreSeparateFromTheWorkModeLabels(t *testing.T) {
	// `flag` y `ask` ya están en `labels` con el sentido del modo de trabajo: la promoción
	// usa su propio mapa y nunca los comparte.
	if got, want := PromotionLabel("flag"), "bandera --promotion"; got != want {
		t.Errorf("PromotionLabel(flag) = %q, se esperaba %q (no la del modo de trabajo)", got, want)
	}
	if Label("flag") == PromotionLabel("flag") {
		t.Errorf("Label y PromotionLabel no deben compartir la etiqueta de flag")
	}
	for _, token := range []string{"branch", "release", "ask", "answer", "repository", "git", "default", "snapshot", "legacy", "integration", "main"} {
		if PromotionLabel(token) == token {
			t.Errorf("PromotionLabel(%q) debe tener etiqueta propia", token)
		}
	}
	if got := PromotionLabel("desconocido"); got != "desconocido" {
		t.Errorf("un token desconocido se devuelve tal cual, no %q", got)
	}
}

func TestKnownPromotionSource(t *testing.T) {
	for _, source := range []string{"", "answer", "flag", "repository", "git", "default", "snapshot", "legacy"} {
		if !KnownPromotionSource(source) {
			t.Errorf("%q debe ser un origen conocido", source)
		}
	}
	for _, source := range []string{"hub", "Answer", "azar"} {
		if KnownPromotionSource(source) {
			t.Errorf("%q no es un origen de la promoción", source)
		}
	}
}

// RF-BRA-29 (D10): los cinco errores de uso de `--promotion` y las peticiones válidas.
func TestParsePromotionRequests(t *testing.T) {
	valid := []struct {
		name   string
		values []string
		want   []PromotionRequest
	}{
		{"sin valores", nil, nil},
		{"sin cualificar", []string{"branch"}, []PromotionRequest{{Mode: PromotionBranch}}},
		{"cualificado", []string{"web=release"}, []PromotionRequest{{Repository: "web", Mode: PromotionRelease}}},
		{"varios repositorios", []string{"web=release", "api=branch"}, []PromotionRequest{{Repository: "web", Mode: PromotionRelease}, {Repository: "api", Mode: PromotionBranch}}},
		{"uno sin cualificar y otro cualificado", []string{"release", "api=branch"}, []PromotionRequest{{Mode: PromotionRelease}, {Repository: "api", Mode: PromotionBranch}}},
		{"con espacios alrededor del repositorio", []string{" web =branch"}, []PromotionRequest{{Repository: "web", Mode: PromotionBranch}}},
	}
	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParsePromotionRequests(tc.values)
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ParsePromotionRequests(%q) = %+v, %v; se esperaba %+v", tc.values, got, err, tc.want)
			}
		})
	}

	invalid := []struct {
		name   string
		values []string
		want   string
	}{
		{"valor vacío", []string{""}, "valor vacío"},
		{"valor en blanco", []string{"  "}, "valor vacío"},
		{"repositorio vacío", []string{"=branch"}, "no nombra el repositorio"},
		{"modo vacío", []string{"x="}, "no indica el modo"},
		{"ask no es respuesta", []string{"ask"}, "«ask» es una política"},
		{"ask cualificado", []string{"web=ask"}, "«ask» es una política"},
		{"modo desconocido", []string{"trunk"}, "no es una respuesta válida"},
		{"mayúscula", []string{"Branch"}, "no es una respuesta válida"},
		{"dos sin cualificar", []string{"branch", "release"}, "solo se admite una vez"},
		{"mismo repositorio dos veces", []string{"web=branch", "web=release"}, "repite el repositorio «web»"},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParsePromotionRequests(tc.values)
			if err == nil || got != nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("ParsePromotionRequests(%q) = %+v, %v; se esperaba un error con %q", tc.values, got, err, tc.want)
			}
		})
	}
}

// RF-BRA-28 (D4): una fila por regla de la precedencia.
func TestResolvePromotion(t *testing.T) {
	twoStage := func(f PromotionFacts) PromotionFacts { f.TwoStage = true; return f }
	tests := []struct {
		name         string
		facts        PromotionFacts
		want         ResolvedPromotion
		materialized Promotion
	}{
		// 1. Exento o de una etapa.
		{"repositorio de una etapa", PromotionFacts{Requested: PromotionBranch, Declared: PromotionRelease}, ResolvedPromotion{}, ""},
		{"repositorio exento", PromotionFacts{Exempt: true, TwoStage: true, Declared: PromotionAsk}, ResolvedPromotion{}, ""},
		{"una etapa ignora el registro", PromotionFacts{Recorded: &RecordedPromotion{Mode: PromotionBranch}}, ResolvedPromotion{}, ""},

		// 2. Registro.
		{"registro con modo", twoStage(PromotionFacts{Recorded: &RecordedPromotion{Mode: PromotionBranch, Source: PromotionSourceAnswer}}),
			ResolvedPromotion{Mode: PromotionBranch, Source: PromotionSourceAnswer, Recorded: true}, ""},
		{"registro prevalece sobre la política", twoStage(PromotionFacts{Recorded: &RecordedPromotion{Mode: PromotionRelease, Source: PromotionSourceGit}, Declared: PromotionBranch, DeclaredSource: PromotionSourceGit}),
			ResolvedPromotion{Mode: PromotionRelease, Source: PromotionSourceGit, Recorded: true}, ""},
		{"registro sin modo y con rama del cambio es legacy", twoStage(PromotionFacts{Recorded: &RecordedPromotion{HasChangeBranch: true}, Declared: PromotionBranch}),
			ResolvedPromotion{Mode: PromotionRelease, Source: PromotionSourceLegacy, Recorded: true}, ""},
		{"registro sin modo y sin rama del cambio sigue", twoStage(PromotionFacts{Recorded: &RecordedPromotion{}, Declared: PromotionRelease, DeclaredSource: PromotionSourceRepository}),
			ResolvedPromotion{Mode: PromotionRelease, Source: PromotionSourceRepository}, ""},
		{"mismo modo pedido es idempotente", twoStage(PromotionFacts{Recorded: &RecordedPromotion{Mode: PromotionBranch, Source: PromotionSourceFlag}, Requested: PromotionBranch}),
			ResolvedPromotion{Mode: PromotionBranch, Source: PromotionSourceFlag, Recorded: true}, ""},
		{"otro modo pedido es conflicto", twoStage(PromotionFacts{Recorded: &RecordedPromotion{Mode: PromotionBranch, Source: PromotionSourceFlag}, Requested: PromotionRelease}),
			ResolvedPromotion{Mode: PromotionBranch, Source: PromotionSourceFlag, Recorded: true}, PromotionBranch},
		{"conflicto con legacy por registro", twoStage(PromotionFacts{Recorded: &RecordedPromotion{HasChangeBranch: true}, Requested: PromotionBranch}),
			ResolvedPromotion{Mode: PromotionRelease, Source: PromotionSourceLegacy, Recorded: true}, PromotionRelease},

		// 3. Instantánea.
		{"instantánea con modo", twoStage(PromotionFacts{Snapshot: &SnapshotPromotion{Mode: PromotionBranch, SealedWithMain: true}, Declared: PromotionRelease}),
			ResolvedPromotion{Mode: PromotionBranch, Source: PromotionSourceSnapshot}, ""},
		{"instantánea con modo y conflicto", twoStage(PromotionFacts{Snapshot: &SnapshotPromotion{Mode: PromotionBranch}, Requested: PromotionRelease}),
			ResolvedPromotion{Mode: PromotionBranch, Source: PromotionSourceSnapshot}, PromotionBranch},
		{"registro prevalece sobre la instantánea", twoStage(PromotionFacts{Recorded: &RecordedPromotion{Mode: PromotionRelease, Source: PromotionSourceFlag}, Snapshot: &SnapshotPromotion{Mode: PromotionBranch}}),
			ResolvedPromotion{Mode: PromotionRelease, Source: PromotionSourceFlag, Recorded: true}, ""},
		{"instantánea sin marcador y con rama es legacy", twoStage(PromotionFacts{Snapshot: &SnapshotPromotion{SealedWithMain: false}, ChangeBranchExists: true, Declared: PromotionBranch}),
			ResolvedPromotion{Mode: PromotionRelease, Source: PromotionSourceLegacy}, ""},
		{"instantánea sin marcador y sin rama sigue", twoStage(PromotionFacts{Snapshot: &SnapshotPromotion{SealedWithMain: false}, Declared: PromotionRelease, DeclaredSource: PromotionSourceGit}),
			withSource(PromotionRelease, PromotionSourceGit), ""},
		{"instantánea con marcador y con rama sigue", twoStage(PromotionFacts{Snapshot: &SnapshotPromotion{SealedWithMain: true}, ChangeBranchExists: true, Declared: PromotionRelease, DeclaredSource: PromotionSourceGit}),
			withSource(PromotionRelease, PromotionSourceGit), ""},
		{"conflicto con legacy por instantánea", twoStage(PromotionFacts{Snapshot: &SnapshotPromotion{}, ChangeBranchExists: true, Requested: PromotionBranch}),
			ResolvedPromotion{Mode: PromotionRelease, Source: PromotionSourceLegacy}, PromotionRelease},

		// 4. Bandera.
		{"bandera sin política ask es flag", twoStage(PromotionFacts{Requested: PromotionRelease, Declared: PromotionBranch, DeclaredSource: PromotionSourceGit}),
			withSource(PromotionRelease, PromotionSourceFlag), ""},
		{"bandera sin política es flag", twoStage(PromotionFacts{Requested: PromotionBranch}), withSource(PromotionBranch, PromotionSourceFlag), ""},
		{"bandera con política ask es answer", twoStage(PromotionFacts{Requested: PromotionBranch, Declared: PromotionAsk, DeclaredSource: PromotionSourceRepository}),
			withSource(PromotionBranch, PromotionSourceAnswer), ""},

		// 5. Política.
		{"política del repositorio", twoStage(PromotionFacts{Declared: PromotionRelease, DeclaredSource: PromotionSourceRepository}), withSource(PromotionRelease, PromotionSourceRepository), ""},
		{"política de git", twoStage(PromotionFacts{Declared: PromotionBranch, DeclaredSource: PromotionSourceGit}), withSource(PromotionBranch, PromotionSourceGit), ""},

		// 6. Ask sin respuesta.
		{"ask pendiente", twoStage(PromotionFacts{Declared: PromotionAsk, DeclaredSource: PromotionSourceGit}), ResolvedPromotion{Mode: PromotionAsk}, ""},

		// 7. Nada declarado.
		{"defecto branch", twoStage(PromotionFacts{}), withSource(PromotionBranch, PromotionSourceDefault), ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolvePromotion(tc.facts)
			if got.Resolved != tc.want || got.Materialized != tc.materialized {
				t.Errorf("ResolvePromotion = %+v, se esperaba {%+v, materializado %q}", got, tc.want, tc.materialized)
			}
		})
	}
}

// withSource es un atajo de las tablas: un modo con su origen, sin registro.
func withSource(mode Promotion, source string) ResolvedPromotion {
	return ResolvedPromotion{Mode: mode, Source: source}
}

func twoStageRepo(promotion Promotion) RepoPlan {
	repo := RepoPlan{Name: "api", GitRoot: "/r/api", IntegrationBranch: "develop", MainBranch: "master", Promotion: promotion}
	return repo
}

// RF-BRA-31 (D7): destino de la confirmación y aplicabilidad de la capa 3.
func TestMergeDestination(t *testing.T) {
	change := BranchPlan{Name: "feat/x", Kind: KindChange, PRTarget: "develop"}
	role := BranchPlan{Name: "feat/x-core", Kind: KindRole, From: "feat/x", PRTarget: "feat/x"}
	archive := BranchPlan{Name: "feat/x-archive", Kind: KindArchive, PRTarget: "develop"}
	oneStage := RepoPlan{IntegrationBranch: "main"}
	tests := []struct {
		name        string
		repo        RepoPlan
		planned     BranchPlan
		destination string
		prMerged    bool
	}{
		{"cambio de una etapa", oneStage, change, "main", true},
		{"cambio en modo release", twoStageRepo(PromotionRelease), change, "develop", true},
		{"cambio en modo branch", twoStageRepo(PromotionBranch), change, "master", false},
		{"cambio con ask pendiente toma el destino más exigente", twoStageRepo(PromotionAsk), change, "master", false},
		{"cambio de dos etapas sin modo resuelto", twoStageRepo(""), change, "develop", true},
		{"rama de rol sigue a la de seguimiento", twoStageRepo(PromotionBranch), role, "feat/x", true},
		{"rama de archivado va a la integración", twoStageRepo(PromotionBranch), archive, "develop", false},
		{"rama de archivado de una etapa", oneStage, archive, "main", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			destination, prMerged := MergeDestination(tc.repo, tc.planned)
			if destination != tc.destination || prMerged != tc.prMerged {
				t.Errorf("MergeDestination = %q, %v; se esperaba %q, %v", destination, prMerged, tc.destination, tc.prMerged)
			}
		})
	}
}

// RF-BRA-30 (D8): las reglas de la etapa de `status`.
func TestPromotionStage(t *testing.T) {
	repo := twoStageRepo(PromotionBranch)
	zero, three := 0, 3
	tests := []struct {
		name         string
		ahead        *int
		evidence     MergeEvidence
		stage        string
		target       string
		inconclusive bool
	}{
		{"sin adelanto calculado", nil, MergeEvidence{Ancestry: true}, PromotionStageIntegration, "develop", false},
		{"rama recién creada desde master no parece fusionada", &zero, MergeEvidence{Ancestry: true}, PromotionStageIntegration, "develop", false},
		{"fusionada por ascendencia", &three, MergeEvidence{Ancestry: true}, PromotionStageMain, "master", false},
		{"fusionada por árbol equivalente con soporte", &three, MergeEvidence{TreeLayerSupported: true, TreeEquivalent: true}, PromotionStageMain, "master", false},
		{"capa 2 sin soporte y sin ascendencia no concluye", &three, MergeEvidence{}, PromotionStageIntegration, "develop", true},
		{"capa 2 con soporte y sin fusión", &three, MergeEvidence{TreeLayerSupported: true}, PromotionStageIntegration, "develop", false},
		{"capa 2 sin soporte con equivalencia ignorada", &three, MergeEvidence{TreeEquivalent: true}, PromotionStageIntegration, "develop", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stage, target, inconclusive := PromotionStage(repo, tc.ahead, tc.evidence)
			if stage != tc.stage || target != tc.target || inconclusive != tc.inconclusive {
				t.Errorf("PromotionStage = %q, %q, %v; se esperaba %q, %q, %v", stage, target, inconclusive, tc.stage, tc.target, tc.inconclusive)
			}
		})
	}
}

func TestRepoPlanTwoStageAndBase(t *testing.T) {
	tests := []struct {
		name     string
		repo     RepoPlan
		twoStage bool
		base     string
	}{
		{"sin rama principal", RepoPlan{IntegrationBranch: "main"}, false, "main"},
		{"rama principal igual a la integración", RepoPlan{IntegrationBranch: "main", MainBranch: "main"}, false, "main"},
		{"dos etapas sin base calculada", RepoPlan{IntegrationBranch: "develop", MainBranch: "master"}, true, "develop"},
		{"dos etapas con base", RepoPlan{IntegrationBranch: "develop", MainBranch: "master", BaseBranch: "master"}, true, "master"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.repo.TwoStage(); got != tc.twoStage {
				t.Errorf("TwoStage() = %v, se esperaba %v", got, tc.twoStage)
			}
			if got := tc.repo.Base(); got != tc.base {
				t.Errorf("Base() = %q, se esperaba %q", got, tc.base)
			}
		})
	}
}
