package engine

import (
	"slices"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// guardRepository son los hechos del repositorio consultado con los que la guarda calcula sus
// nombres esperados.
type guardRepository struct {
	root, common string
	isPlanning   bool
	planning     branch.RepoFacts // repositorio del planning home (identidad y ruta relativa)
}

// expectedNames compone los nombres esperados del repositorio consultado (design D-10, «Nombres
// esperados»). La rama del cambio sale, en este orden, de la `--branch` registrada, la instantánea
// del kickoff y el nombre registrado; si ninguna la nombra, del mismo planificador que `ensure`
// (plantilla efectiva con los valores de variables de D-13, o `<prefijo>/<cambio>`), de modo que
// la guarda y el plan nunca discrepan, también sin registro. Las ramas de rol salen de la
// instantánea o del registro y, con plantilla de rol, de su render; un rol cuya Task aún no existe
// pasa a PendingRoles. Para la de archivado: la registrada; con `--prefix` registrado, solo sin
// plantilla; si no, la de la instantánea; si no, `<rama del cambio>-archive`. El carril ODD no
// tiene rama de archivado. Un nombre que no se puede resolver devuelve el rechazo tipado: la
// guarda falla cerrado (D-10).
func expectedNames(q VerdictQuery, loc workspace.Location, record *branch.Record, snapshot *kickoff.GitSnapshot, repo guardRepository) (branch.ExpectedNames, *branch.Refusal) {
	policy := branch.ResolvePolicy(loc.Config)
	effective := branch.RepoPolicy{IntegrationBranch: policy.IntegrationBranch, BranchPrefix: policy.BranchPrefix}
	relative, inside := relativeWithin(loc.Master, repo.root)
	if inside {
		effective = policy.For(relative)
	}

	recordedChange, recordedRoles, recordedArchive := recordedBranches(record, repo.common)
	var recordedException, rawPrefix string
	if record != nil {
		recordedException, rawPrefix = record.Exceptions.Branch, record.Exceptions.Prefix
	}
	recordedPrefix := rawPrefix
	if recordedPrefix == "" {
		recordedPrefix = effective.BranchPrefix
	}
	// La instantánea manda sobre el registro y la política en las ramas del cambio y de rol; un
	// repositorio al que no da rama del cambio (uno fuera de la carpeta maestra, o un repositorio de
	// código que no tocan los roles sellados) sigue con los nombres de abajo.
	entry, found := snapshotEntry(snapshot, relative, inside)
	snapshotChange := snapshotBranch(entry, "change")
	fromSnapshot := found && (snapshotChange != "" || len(entry.PendingBranches) > 0)
	named := branch.ResolveChangeBranch(q.Change, "",
		branch.NamedBranch("--branch registrada", recordedException),
		branch.NamedBranch("instantánea", snapshotChange),
		branch.NamedBranch("registro", recordedChange))

	// Entrada del planificador: solo lo que necesita para nombrar las ramas, sin tocar git.
	in := branch.PlanInput{
		Operation:  "verdict",
		Lane:       q.Lane,
		Change:     q.Change,
		Topology:   topologyOf(loc),
		Master:     loc.Master,
		HasKickoff: snapshot != nil,
		Snapshot:   snapshot,
		Policy:     policy,
		PlanningRepo: branch.RepoFacts{
			GitRoot: repo.planning.GitRoot, CommonDir: repo.planning.CommonDir, RelPath: repo.planning.RelPath,
		},
		Exceptions:    branch.Exceptions{Prefix: rawPrefix, Branch: recordedException},
		WorkspaceRoot: loc.WorkspaceRoot,
	}
	if named != "" {
		in.Exceptions.Branch = named
	}
	in.TemplateValues = resolveTemplateValues(in, loc)

	touching := rolesTouching(loc, repo.root, q.SealRoles)
	var roleNames []string
	switch {
	case fromSnapshot:
		for _, pending := range entry.PendingBranches {
			if pending.Kind == "role" && !slices.Contains(roleNames, pending.Role) {
				roleNames = append(roleNames, pending.Role)
			}
		}
	case len(touching) >= 2:
		roleNames = touching
	}
	rendered, refusal := branch.RenderGuardBranches(in, branch.GuardRepository{
		Facts: branch.RepoFacts{
			GitRoot: repo.root, CommonDir: repo.common, RelPath: relativeOrEmpty(relative, inside),
			Frozen: frozenFor(record, repo.common),
		},
		Planning:  repo.isPlanning,
		Roles:     touching,
		Effective: effective,
		Prefix:    recordedPrefix,
	}, roleNames)
	if refusal != nil {
		return branch.ExpectedNames{}, refusal
	}
	changeBranch := rendered.Change
	names := branch.ExpectedNames{Change: changeBranch, Integration: effective.IntegrationBranch, PendingRoles: rendered.PendingRoles}

	// Con dos o más roles que tocan el repositorio hay ramas de rol (RF-BRA-04); las registradas
	// valen aunque el kickoff aún no las nombre.
	roles := make(map[string]bool, len(recordedRoles))
	if fromSnapshot {
		for _, entryBranch := range entry.Branches {
			if entryBranch.Kind == "role" {
				roles[entryBranch.Name] = true
			}
		}
	} else {
		for _, name := range recordedRoles {
			roles[name] = true
		}
	}
	for _, name := range rendered.Roles {
		roles[name] = true
	}
	for name := range roles {
		names.Roles = append(names.Roles, name)
	}
	slices.Sort(names.Roles)

	if q.Lane != branch.LaneODD && changeBranch != "" {
		templated := branch.ChangeBranchTemplated(in)
		switch snapshotArchive := snapshotBranch(entry, "archive"); {
		case recordedArchive != "":
			names.Archive = recordedArchive
		case record != nil && record.ArchivePrefix != "" && !templated:
			names.Archive = branch.ArchiveBranchFor(branch.ResolveChangeBranch(q.Change, record.ArchivePrefix))
		case fromSnapshot && snapshotArchive != "":
			names.Archive = snapshotArchive
		default:
			names.Archive = branch.ArchiveBranchFor(changeBranch)
		}
	}
	return names, nil
}

// relativeOrEmpty es la ruta relativa a la carpeta maestra, o vacía si el repositorio queda fuera
// de ella (entonces rige la política del workspace).
func relativeOrEmpty(relative string, inside bool) string {
	if !inside {
		return ""
	}
	return relative
}
