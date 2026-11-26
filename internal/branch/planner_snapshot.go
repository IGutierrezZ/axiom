package branch

import (
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// La instantánea de política git del kickoff (RF-KOF-06, design D-12) fija los nombres de rama
// con los que se selló el cambio: con ella el plan los conserva aunque la política de
// `axiom.yaml` cambie después (decisión T2 y D-10: instantánea, registro y política, en ese
// orden). Sin instantánea, o sin entrada para el repositorio, rigen la política y las
// excepciones, como siempre.

// snapshotRepo devuelve la entrada de la instantánea del repositorio de facts, que se reconoce
// por su ruta relativa a la carpeta maestra. Un repositorio fuera de ella (RelPath vacío) no
// tiene entrada propia: rige la política del workspace.
func snapshotRepo(in PlanInput, facts RepoFacts) (kickoff.GitSnapshotRepo, bool) {
	if in.Snapshot == nil || facts.RelPath == "" {
		return kickoff.GitSnapshotRepo{}, false
	}
	for _, repo := range in.Snapshot.Repositories {
		if repoKey(repo.Path) == repoKey(facts.RelPath) {
			return repo, true
		}
	}
	return kickoff.GitSnapshotRepo{}, false
}

// snapshotBranch devuelve el nombre de la rama de la entrada de ese tipo (y, en las de rol, de
// ese rol), o la cadena vacía si la instantánea no la nombra.
func snapshotBranch(repo kickoff.GitSnapshotRepo, kind BranchKind, role string) string {
	for _, named := range repo.Branches {
		if named.Kind == string(kind) && named.Role == role {
			return named.Name
		}
	}
	return ""
}

// SnapshotPromotionFor devuelve lo que la instantánea del plan sabe de la promoción del
// repositorio de facts, o nil si no hay instantánea o no lista el repositorio.
func SnapshotPromotionFor(in PlanInput, facts RepoFacts) *SnapshotPromotion {
	repo, ok := snapshotRepo(in, facts)
	if !ok {
		return nil
	}
	return &SnapshotPromotion{Mode: Promotion(repo.Promotion), SealedWithMain: in.Snapshot.MainBranch != ""}
}

// SnapshotChangeBranch devuelve la rama del cambio que la instantánea del plan nombra para el
// repositorio de facts, o la cadena vacía si no hay instantánea o no la nombra. El motor la usa
// para derivar la rama de archivado (D-25, paso 3) cuando el registro no guarda la del cambio.
func SnapshotChangeBranch(in PlanInput, facts RepoFacts) string {
	repo, ok := snapshotRepo(in, facts)
	if !ok {
		return ""
	}
	return snapshotBranch(repo, KindChange, "")
}

// snapshotBranchTemplate devuelve la plantilla informativa con la que se nombró la rama de ese tipo
// (y rol) en la instantánea, o la cadena vacía si no la lleva.
func snapshotBranchTemplate(repo kickoff.GitSnapshotRepo, kind BranchKind, role string) string {
	for _, named := range repo.Branches {
		if named.Kind == string(kind) && named.Role == role {
			return named.Template
		}
	}
	return ""
}

// snapshotPending devuelve la entrada pendiente de la instantánea de ese tipo y, en las de rol, de
// ese rol (sin distinguir mayúsculas). Las entradas pendientes no llevan nombre (design D-06).
func snapshotPending(repo kickoff.GitSnapshotRepo, kind BranchKind, role string) (kickoff.GitSnapshotPendingBranch, bool) {
	for _, pending := range repo.PendingBranches {
		if pending.Kind == string(kind) && (kind == KindChange || strings.EqualFold(pending.Role, role)) {
			return pending, true
		}
	}
	return kickoff.GitSnapshotPendingBranch{}, false
}
