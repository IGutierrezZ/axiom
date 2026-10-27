package branch

// MergeEvidence es lo que el motor ha podido comprobar de una rama para confirmar que ya está
// fusionada en su destino (REQ-25.16, design D-09). Cada campo es el resultado de una capa; el
// motor solo rellena las que ha evaluado y deja en falso las demás.
type MergeEvidence struct {
	// Ancestry es la capa 1: la rama es ancestro del destino (`git merge-base --is-ancestor`).
	Ancestry bool
	// TreeEquivalent es la capa 2: la fusión simulada de la rama en el destino produce el árbol
	// del destino (`git merge-tree --write-tree`), de modo que detecta un squash o un rebase aunque
	// el destino haya avanzado después. Solo cuenta si TreeLayerSupported.
	TreeEquivalent bool
	// TreeLayerSupported dice si el git instalado admite la orden de la capa 2 (desde la 2.38,
	// Q1). Sin soporte la capa se omite sin error y su resultado se ignora.
	TreeLayerSupported bool
	// PRMerged es la capa 3: el `gates.yaml` del cambio tiene una evidencia `pr_merged` verificada.
	// Acredita el PR del cambio, no el PR documental del archivado, de modo que nunca confirma una
	// rama de archivado.
	PRMerged bool
}

// ConfirmMerge decide qué capa confirma la fusión de una rama del tipo dado (REQ-25.16, design
// D-09). Es pura y se detiene en la primera capa que confirma: ascendencia, árbol equivalente (solo
// si git lo admite) y PR fusionado (nunca para la rama de archivado). Si ninguna confirma, la
// fusión no está confirmada, salvo que force sea cierto: `--force` salta la exigencia de fusión
// confirmada (REQ-25.17) y la confirmación es la capa `forced`. force nunca oculta una capa que sí
// confirma. Sin confirmación devuelve la capa vacía y false.
func ConfirmMerge(kind BranchKind, evidence MergeEvidence, force bool) (layer MergeLayer, confirmed bool) {
	switch {
	case evidence.Ancestry:
		return LayerAncestry, true
	case evidence.TreeLayerSupported && evidence.TreeEquivalent:
		return LayerEquivalentTree, true
	case evidence.PRMerged && kind != KindArchive:
		return LayerPRMerged, true
	case force:
		return LayerForced, true
	}
	return "", false
}
