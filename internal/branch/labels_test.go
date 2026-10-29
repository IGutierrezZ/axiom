package branch

import "testing"

// REQ-25.2 (G-4): todo valor de estado o de clasificación del JSON es un token
// estable en inglés con su etiqueta en castellano en un campo aparte.
func TestLabelPairsEveryStableTokenWithItsSpanishLabel(t *testing.T) {
	tests := []struct {
		token string
		want  string
	}{
		{"ready", "listo"},
		{"decision_required", "decisión requerida"},
		{"refused", "rechazado"},
		{"partial", "parcial"},
		{"not_started", "no iniciado"},
		{"error", "error"},
		{"correct", "correcto"},
		{"wrong_branch", "rama equivocada"},
		{"not_applicable", "no aplicable"},
		{"ancestry", "ascendencia"},
		{"equivalent_tree", "árbol equivalente"},
		{"pr_merged", "PR fusionado"},
		{"forced", "forzado"},
	}
	for _, tc := range tests {
		t.Run(tc.token, func(t *testing.T) {
			if got := Label(tc.token); got != tc.want {
				t.Errorf("Label(%q) = %q, se esperaba %q", tc.token, got, tc.want)
			}
		})
	}
}

// Cada constante tipada de estado, veredicto y capa de fusión tiene etiqueta: añadir
// un token sin su etiqueta rompe esta prueba en lugar de publicar un campo hermano
// vacío o sin traducir.
func TestEveryStatusVerdictAndMergeLayerTokenHasALabel(t *testing.T) {
	var tokens []string
	for _, status := range []Status{StatusReady, StatusDecisionRequired, StatusRefused, StatusPartial, StatusNotStarted, StatusError} {
		tokens = append(tokens, string(status))
	}
	for _, verdict := range []VerdictKind{VerdictCorrect, VerdictWrongBranch, VerdictNotApplicable} {
		tokens = append(tokens, string(verdict))
	}
	for _, layer := range []MergeLayer{LayerAncestry, LayerEquivalentTree, LayerPRMerged, LayerForced} {
		tokens = append(tokens, string(layer))
	}
	for _, token := range tokens {
		if label, known := labels[token]; !known || label == "" {
			t.Errorf("el token %q no tiene etiqueta en castellano", token)
		}
	}
}

// El papel de cada repositorio y el tipo de cada rama son valores de clasificación del
// `status` (REQ-25.15): cada token lleva su etiqueta en castellano, con los términos
// de la spec.
func TestEveryRepositoryRoleAndBranchKindTokenHasASpanishLabel(t *testing.T) {
	tests := map[string]string{
		string(RepoChangeBranch): "repositorio con rama de cambio",
		string(RepoSpecsInPlace): "repositorio de specs in situ",
		string(RepoCode):         "repositorio de código",
		string(KindChange):       "rama del cambio",
		string(KindRole):         "rama de rol",
		string(KindArchive):      "rama de archivado",
	}
	for token, want := range tests {
		t.Run(token, func(t *testing.T) {
			if got := Label(token); got != want {
				t.Errorf("Label(%q) = %q, se esperaba %q", token, got, want)
			}
		})
	}
}

// El modo de trabajo materializado y su origen son valores de clasificación del
// `status` (REQ-25.15): cada token lleva su etiqueta en castellano.
func TestEveryModeAndModeSourceTokenHasASpanishLabel(t *testing.T) {
	tests := map[string]string{
		string(ModeWorktree): "un worktree por rama",
		string(ModeInPlace):  "en la carpeta del proyecto",
		string(ModeAsk):      "preguntar",
		ModeSourceAnswer:     "respuesta registrada",
		ModeSourceFlag:       "bandera --mode",
		ModeSourceHub:        "preferencias personales del hub",
		ModeSourceDefault:    "valor por defecto",
	}
	for token, want := range tests {
		t.Run(token, func(t *testing.T) {
			if got := Label(token); got != want {
				t.Errorf("Label(%q) = %q, se esperaba %q", token, got, want)
			}
		})
	}
}

// Un token que el dominio no conoce no produce una etiqueta vacía: el campo hermano
// repite el token, que al menos es legible y estable.
func TestLabelOfAnUnknownTokenIsTheTokenItself(t *testing.T) {
	if got := Label("token_inventado"); got != "token_inventado" {
		t.Errorf("Label de un token desconocido = %q, se esperaba el propio token", got)
	}
}

// Los tokens no se traducen ni cambian con el idioma (spec, «Tokens y etiquetas»):
// son ASCII en minúsculas con guion bajo.
func TestLabelTokensAreStableEnglishIdentifiers(t *testing.T) {
	for token := range labels {
		for _, char := range token {
			if (char < 'a' || char > 'z') && char != '_' {
				t.Errorf("el token %q debe ser un identificador estable en inglés (a-z y «_»)", token)
				break
			}
		}
	}
}
