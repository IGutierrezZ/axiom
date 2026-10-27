package branch

import "testing"

// REQ-25.16 y REQ-25.17, design D-09: la confirmación de la fusión por capas es una función pura
// de la evidencia que reúne el motor. Se detiene en la primera capa que confirma, la capa del árbol
// equivalente solo cuenta si git la admite, la evidencia `pr_merged` nunca confirma la rama de
// archivado y `--force` solo se aplica cuando ninguna capa confirma.
func TestConfirmMergeStopsAtTheFirstLayerThatConfirms(t *testing.T) {
	tests := []struct {
		name          string
		kind          BranchKind
		evidence      MergeEvidence
		force         bool
		want          MergeLayer
		wantConfirmed bool
	}{
		{
			name:          "la ascendencia confirma aunque las demás capas también lo harían",
			kind:          KindChange,
			evidence:      MergeEvidence{Ancestry: true, TreeEquivalent: true, TreeLayerSupported: true, PRMerged: true},
			want:          LayerAncestry,
			wantConfirmed: true,
		},
		{
			name:          "sin ascendencia, el árbol equivalente confirma antes que el PR fusionado",
			kind:          KindChange,
			evidence:      MergeEvidence{TreeEquivalent: true, TreeLayerSupported: true, PRMerged: true},
			want:          LayerEquivalentTree,
			wantConfirmed: true,
		},
		{
			name:          "sin ascendencia ni árbol equivalente, confirma el PR fusionado",
			kind:          KindChange,
			evidence:      MergeEvidence{TreeLayerSupported: true, PRMerged: true},
			want:          LayerPRMerged,
			wantConfirmed: true,
		},
		{
			name:          "una rama de rol se confirma con las mismas capas",
			kind:          KindRole,
			evidence:      MergeEvidence{PRMerged: true},
			want:          LayerPRMerged,
			wantConfirmed: true,
		},
		{
			name:     "el árbol equivalente solo cuenta si git admite la capa",
			kind:     KindChange,
			evidence: MergeEvidence{TreeEquivalent: true, TreeLayerSupported: false},
		},
		{
			name:          "sin soporte de la capa 2, la evidencia pr_merged sigue confirmando",
			kind:          KindChange,
			evidence:      MergeEvidence{TreeEquivalent: true, TreeLayerSupported: false, PRMerged: true},
			want:          LayerPRMerged,
			wantConfirmed: true,
		},
		{
			name:          "la rama de archivado se confirma por ascendencia",
			kind:          KindArchive,
			evidence:      MergeEvidence{Ancestry: true},
			want:          LayerAncestry,
			wantConfirmed: true,
		},
		{
			name:          "la rama de archivado se confirma por árbol equivalente tras un squash",
			kind:          KindArchive,
			evidence:      MergeEvidence{TreeEquivalent: true, TreeLayerSupported: true},
			want:          LayerEquivalentTree,
			wantConfirmed: true,
		},
		{
			name:     "pr_merged nunca confirma la rama de archivado",
			kind:     KindArchive,
			evidence: MergeEvidence{TreeLayerSupported: true, PRMerged: true},
		},
		{
			name:     "ninguna capa confirma: la fusión no está confirmada",
			kind:     KindChange,
			evidence: MergeEvidence{TreeLayerSupported: true},
		},
		{
			name:     "sin evidencia alguna tampoco",
			kind:     KindRole,
			evidence: MergeEvidence{},
		},
		{
			name:          "con force y ninguna capa, la confirmación es forced",
			kind:          KindChange,
			evidence:      MergeEvidence{},
			force:         true,
			want:          LayerForced,
			wantConfirmed: true,
		},
		{
			name:          "con force la rama de archivado sin árbol equivalente es forced aunque conste pr_merged",
			kind:          KindArchive,
			evidence:      MergeEvidence{PRMerged: true},
			force:         true,
			want:          LayerForced,
			wantConfirmed: true,
		},
		{
			name:          "force no oculta la capa que sí confirma",
			kind:          KindChange,
			evidence:      MergeEvidence{Ancestry: true},
			force:         true,
			want:          LayerAncestry,
			wantConfirmed: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, confirmed := ConfirmMerge(tc.kind, tc.evidence, tc.force)

			if got != tc.want || confirmed != tc.wantConfirmed {
				t.Errorf("ConfirmMerge(%s, %+v, force=%v) = (%q, %v), se esperaba (%q, %v)", tc.kind, tc.evidence, tc.force, got, confirmed, tc.want, tc.wantConfirmed)
			}
		})
	}
}

// Una capa sin confirmar no deja ningún token: el JSON de `done` nunca lleva un `confirmedBy`
// vacío con su etiqueta, y cada capa que sí confirma tiene la suya (REQ-25.2).
func TestConfirmMergeNeverReturnsALayerWithoutALabel(t *testing.T) {
	for _, evidence := range []MergeEvidence{
		{Ancestry: true},
		{TreeEquivalent: true, TreeLayerSupported: true},
		{PRMerged: true},
		{},
	} {
		layer, confirmed := ConfirmMerge(KindChange, evidence, true)
		if !confirmed || layer == "" {
			t.Fatalf("con force siempre hay confirmación: (%q, %v) para %+v", layer, confirmed, evidence)
		}
		if label := Label(string(layer)); label == "" || label == string(layer) {
			t.Errorf("la capa %q debe tener etiqueta en castellano, tiene %q", layer, label)
		}
	}
}
