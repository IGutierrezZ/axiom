package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/doctor"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// Comprobaciones del proyecto: solo leen ficheros bajo la raíz indicada (nada de red,
// de subprocesos nuevos ni del directorio actual del proceso).

const projectSyncRemedy = "Run 'axiom sync --scope workspace' to refresh the project's managed assets"

// Overridable for testing.
var doctorManagedDigestFn = managedAssetDigest

// projectDoctorChecks devuelve las comprobaciones del estado instalado del proyecto.
// Sin raíz o sin `.axiom/state.json` no hay instalación de proyecto que mirar y no se
// añade ninguna (así `axiom doctor` no cambia para quien solo usa la instalación global).
func projectDoctorChecks(opts doctor.Options, store state.Store) []doctor.Check {
	if strings.TrimSpace(opts.Root) == "" || !store.IsProject() {
		return nil
	}
	return []doctor.Check{
		{ID: doctor.CheckProjectAssets, Scope: doctor.ScopeProject, Run: func(context.Context) doctor.Result {
			return checkProjectAssets(store)
		}},
		{ID: doctor.CheckProjectAdapters, Scope: doctor.ScopeProject, Run: func(context.Context) doctor.Result {
			return checkProjectAdapters(opts.Root, store)
		}},
	}
}

// checkProjectAssets compara el digest de los assets gestionados que escribió la
// última instalación del proyecto con el del binario en ejecución. Sin estado de
// proyecto o sin digest registrado no hay nada instalado que pueda estar desfasado.
func checkProjectAssets(store state.Store) doctor.Result {
	if !store.IsProject() {
		return doctor.Result{Status: doctor.StatusPass, Detail: "project has no managed install (no .axiom/state.json) — asset check skipped"}
	}
	installed, err := store.Read()
	if err != nil {
		return doctor.Result{Status: doctor.StatusWarn, Detail: "project state could not be read — asset check skipped (see state:json)"}
	}
	if installed.ManagedAssetDigest == "" {
		return doctor.Result{Status: doctor.StatusPass, Detail: "no managed asset digest recorded in the project state — check skipped"}
	}
	expected, err := doctorManagedDigestFn()
	if err != nil {
		return doctor.Result{Status: doctor.StatusWarn, Detail: "could not derive the managed asset digest of this binary: " + err.Error()}
	}
	if installed.ManagedAssetDigest != expected {
		return doctor.Result{
			Status: doctor.StatusWarn,
			Detail: fmt.Sprintf("project managed assets were written by a different asset set (installed %s, this binary %s)", shortDigest(installed.ManagedAssetDigest), shortDigest(expected)),
			Remedy: doctor.NewRemedy(doctor.RemedySync, projectSyncRemedy),
		}
	}
	return doctor.Result{Status: doctor.StatusPass, Detail: "project managed assets match this binary (" + shortDigest(expected) + ")"}
}

func shortDigest(digest string) string {
	if len(digest) > 12 {
		return digest[:12]
	}
	return digest
}

// checkProjectAdapters verifica que cada agente registrado en el estado del proyecto
// tiene su configuración en la raíz (el adaptador mira la raíz como si fuese el HOME
// del ámbito, igual que hace la instalación con ámbito workspace).
func checkProjectAdapters(root string, store state.Store) doctor.Result {
	if !store.IsProject() {
		return doctor.Result{Status: doctor.StatusPass, Detail: "project has no managed install — adapter check skipped"}
	}
	installed, err := store.Read()
	if err != nil {
		return doctor.Result{Status: doctor.StatusWarn, Detail: "project state could not be read — adapter check skipped (see state:json)"}
	}
	if len(installed.InstalledAgents) == 0 {
		return doctor.Result{Status: doctor.StatusPass, Detail: "no agents registered in the project state"}
	}
	registry, err := agents.NewDefaultRegistry()
	if err != nil {
		return doctor.Result{Status: doctor.StatusWarn, Detail: "agent adapters are unavailable: " + err.Error()}
	}

	var absent, unknown, unreadable []string
	for _, id := range installed.InstalledAgents {
		adapter, ok := registry.Get(model.AgentID(id))
		if !ok {
			unknown = append(unknown, id)
			continue
		}
		_, _, _, configFound, detectErr := adapter.Detect(context.Background(), root)
		switch {
		case detectErr != nil && !os.IsNotExist(detectErr):
			unreadable = append(unreadable, id)
		case !configFound:
			absent = append(absent, id)
		}
	}

	var problems []string
	if len(absent) > 0 {
		problems = append(problems, "registered but without configuration in the project root: "+strings.Join(absent, ", "))
	}
	if len(unknown) > 0 {
		problems = append(problems, "unknown to this binary: "+strings.Join(unknown, ", "))
	}
	if len(unreadable) > 0 {
		problems = append(problems, "configuration could not be inspected: "+strings.Join(unreadable, ", "))
	}
	if len(problems) > 0 {
		return doctor.Result{
			Status: doctor.StatusWarn,
			Detail: "agents " + strings.Join(problems, "; "),
			Remedy: doctor.NewRemedy(doctor.RemedySync, projectSyncRemedy),
		}
	}
	return doctor.Result{Status: doctor.StatusPass, Detail: fmt.Sprintf("%d agent(s) configured in the project: %s", len(installed.InstalledAgents), strings.Join(installed.InstalledAgents, ", "))}
}
