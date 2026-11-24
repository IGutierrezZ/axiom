package branch

// labels asocia cada token estable en inglés con su etiqueta en castellano (G-4,
// REQ-25.2). Los tokens no se traducen ni cambian con el idioma; lo que se traduce
// es la etiqueta, que va en un campo hermano `<campo>Label`.
//
// Los tokens de estado, de veredicto y de capa de fusión son los de la spec
// (change-branch-lifecycle, «Tokens y etiquetas del JSON»). `wrong_branch` y
// `decision_required` son además códigos tipados: un código no lleva campo
// hermano, pero cuando su token es también un estado o un veredicto, la etiqueta
// es la misma.
var labels = map[string]string{
	string(StatusReady):            "listo",
	string(StatusDecisionRequired): "decisión requerida",
	string(StatusRefused):          "rechazado",
	string(StatusPartial):          "parcial",
	string(StatusNotStarted):       "no iniciado",
	string(StatusError):            "error",

	string(VerdictCorrect):       "correcto",
	string(VerdictWrongBranch):   "rama equivocada",
	string(VerdictNotApplicable): "no aplicable",

	string(LayerAncestry):       "ascendencia",
	string(LayerEquivalentTree): "árbol equivalente",
	string(LayerPRMerged):       "PR fusionado",
	string(LayerForced):         "forzado",

	// Lo que `done` hizo con cada rama (REQ-25.17).
	string(OutcomeRemoved): "eliminada",
	string(OutcomeKept):    "conservada",
	string(OutcomeAbsent):  "ya no existía",

	// Papel del repositorio y tipo de rama del `status` (REQ-25.15): los términos de la
	// spec («repositorio con rama de cambio», «rama de rol»…).
	string(RepoChangeBranch): "repositorio con rama de cambio",
	string(RepoSpecsInPlace): "repositorio de specs in situ",
	string(RepoCode):         "repositorio de código",
	string(KindChange):       "rama del cambio",
	string(KindRole):         "rama de rol",
	string(KindArchive):      "rama de archivado",

	// Códigos de la publicación de documentos (REQ-27.7).
	string(CodeDetachedHead):      "HEAD desacoplado",
	string(CodeIntegrationBranch): "rama de integración no publicable",

	// Rechazos de `ensure --role` (REQ-43.24).
	string(CodeRoleNotApplicable): "rol no aplicable",
	string(CodeKickoffNotSealed):  "kickoff sin sellar",
	string(CodeRoleNotInRoster):   "rol fuera del roster",

	// Modo de trabajo materializado y su origen (REQ-25.15, design §4.4).
	string(ModeWorktree): "un worktree por rama",
	string(ModeInPlace):  "en la carpeta del proyecto",
	string(ModeAsk):      "preguntar",
	ModeSourceAnswer:     "respuesta registrada",
	ModeSourceFlag:       "bandera --mode",
	ModeSourceHub:        "preferencias personales del hub",
	ModeSourceDefault:    "valor por defecto",

	// `axiom branch scope` (REQ-25.51 a REQ-25.54): código del rechazo, resultado de cada
	// repositorio, motivos de `unverifiable`, oferta, fuente del punto de partida, de la rama y aviso.
	string(CodeScopeNotApplicable):     "alcance no aplicable",
	ScopeOutcomeOK:                     "alcance calculado",
	ScopeOutcomeEmpty:                  "sin commits propios",
	ScopeOutcomeUnverifiable:           "no verificable",
	ScopeOutcomeExempt:                 "exento",
	ScopeReasonBranchMissing:           "rama ausente",
	ScopeReasonBaseMissing:             "base o integración ausente",
	ScopeReasonForkPointUndetermined:   "punto de bifurcación indeterminado",
	ScopeReasonNotMerged:               "sin fusionar en la integración",
	ScopeOfferOfferable:                "verificación conjunta ofrecible",
	ScopeOfferPendingMerge:             "pendiente de fusionar",
	ScopeOfferNothingToVerify:          "nada que verificar",
	ScopeStartFirstParent:              "cadena de primeros padres",
	ScopeStartReflog:                   "reflog de la rama",
	ScopeWarningForeignCommitsPossible: "posibles commits ajenos",
}

// Label devuelve la etiqueta en castellano de un token estable. Un token que el
// dominio no conoce se devuelve tal cual: el campo hermano nunca queda vacío y, al
// menos, es estable y legible.
func Label(token string) string {
	if label, known := labels[token]; known {
		return label
	}
	return token
}
