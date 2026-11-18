package engine

import (
	"context"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// mergeReference es una referencia contra la que se busca la fusión de una rama: shown es como se
// muestra (`origin/develop`) y full la referencia completa que se resuelve.
type mergeReference struct{ shown, full string }

// integrationReferences son las referencias de una rama de integración o principal que pueden
// existir en local, por orden de preferencia: la remota ya traída y la local (design D-09).
func integrationReferences(branchName string) []mergeReference {
	return []mergeReference{
		{originRemote + "/" + branchName, "refs/remotes/" + originRemote + "/" + branchName},
		{branchName, "refs/heads/" + branchName},
	}
}

// mergeLayers evalúa las capas 1 (ascendencia) y 2 (árbol equivalente) de la confirmación de
// fusión de branchRef contra las referencias de targets que existen en local (design D7), y se
// detiene en la primera que confirma. Es lo que comparten `done` y `status`. Devuelve las
// referencias que existen (para el mensaje) y la evidencia. treeUnsupported es verdadero cuando
// git no admite `merge-tree --write-tree` (requiere 2.38) y la capa se omitió: cada consumidor
// emite su propio aviso. Usa solo referencias locales.
func (e *Engine) mergeLayers(ctx context.Context, dir, branchRef string, targets []mergeReference) (existing []mergeReference, evidence branch.MergeEvidence, treeUnsupported bool, err error) {
	for _, candidate := range targets {
		if candidate.shown == "" {
			continue
		}
		_, found, err := e.git.ResolveCommit(ctx, dir, candidate.full)
		if err != nil {
			return nil, evidence, false, err
		}
		if found {
			existing = append(existing, candidate)
		}
	}

	// Capa 1: ascendencia.
	for _, target := range existing {
		if evidence.Ancestry, err = e.git.IsAncestor(ctx, dir, branchRef, target.full); err != nil {
			return nil, evidence, false, err
		}
		if evidence.Ancestry {
			return existing, evidence, false, nil
		}
	}

	// Capa 2: árbol equivalente. Sin soporte se omite sin error (Q1).
	for _, target := range existing {
		equivalent, supported, err := e.git.MergeTreeEquivalent(ctx, dir, target.full, branchRef)
		if err != nil {
			return nil, evidence, false, err
		}
		if !supported {
			return existing, evidence, true, nil
		}
		evidence.TreeLayerSupported = true
		if evidence.TreeEquivalent = equivalent; equivalent {
			return existing, evidence, false, nil
		}
	}
	return existing, evidence, false, nil
}

// shownReferences son los nombres con los que se muestran las referencias.
func shownReferences(references []mergeReference) []string {
	shown := make([]string, 0, len(references))
	for _, reference := range references {
		shown = append(shown, reference.shown)
	}
	return shown
}
