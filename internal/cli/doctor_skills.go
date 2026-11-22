package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/doctor"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/skillcheck"
)

const skillsFixRemedy = "Run 'axiom skill check' for the full report; run 'axiom skill index refresh' if the registry is stale"

// SkillFolderRuntimes dice qué runtimes usan cada carpeta de skills del proyecto
// root y de la carpeta de usuario home, a partir de los adaptadores (la carpeta
// donde Axiom instala las skills de cada runtime). Razona por carpeta porque
// varios runtimes comparten una (p. ej. `.agents/skills`). Pi no tiene carpeta de
// instalación de Axiom, pero lee `.pi/skills` y `~/.pi/agent/skills`.
func SkillFolderRuntimes(home, root string) map[string][]string {
	out := map[string][]string{}
	add := func(dir string, id model.AgentID) {
		dir = strings.TrimSpace(dir)
		if dir == "" || dir == "." {
			return
		}
		key := filepath.Clean(dir)
		for _, existing := range out[key] {
			if existing == string(id) {
				return
			}
		}
		out[key] = append(out[key], string(id))
	}

	reg, err := agents.NewDefaultRegistry()
	if err != nil {
		return out
	}
	for _, id := range reg.SupportedAgents() {
		adapter, ok := reg.Get(id)
		if !ok {
			continue
		}
		if id == model.AgentPi {
			if root != "" {
				add(filepath.Join(root, ".pi", "skills"), id)
			}
			if home != "" {
				add(filepath.Join(home, ".pi", "agent", "skills"), id)
			}
			continue
		}
		if !adapter.SupportsSkills() {
			continue
		}
		if root != "" {
			add(agents.BindWorkspace(adapter, root).SkillsDir(root), id)
		}
		if home != "" {
			add(adapter.SkillsDir(home), id)
		}
	}
	return out
}

// skillsDoctorChecks devuelve la comprobación `project:skills`. Sin raíz no hay
// proyecto que mirar y no se añade ninguna.
func skillsDoctorChecks(opts doctor.Options, installedAgents []string) []doctor.Check {
	root := strings.TrimSpace(opts.Root)
	if root == "" {
		return nil
	}
	piInstalled := false
	for _, a := range installedAgents {
		if a == string(model.AgentPi) {
			piInstalled = true
		}
	}
	return []doctor.Check{{ID: doctor.CheckProjectSkills, Scope: doctor.ScopeProject, Run: func(context.Context) doctor.Result {
		return checkProjectSkills(root, opts.Home, piInstalled)
	}}}
}

// checkProjectSkills pasa skillcheck por las skills de proyecto y usuario. Nunca es
// un fallo: el registro las tolera. Es un aviso si hay errores o avisos de skill o
// de conjunto; los hallazgos informativos (alcance, Pi) no lo cambian.
func checkProjectSkills(root, home string, piInstalled bool) doctor.Result {
	rep := skillcheck.ScanSet(root, home, skillcheck.SetOptions{
		FolderRuntimes: SkillFolderRuntimes(home, root),
		PiRelevant:     piInstalled,
	})
	errs, warns := rep.Count(skillcheck.SeverityError), rep.Count(skillcheck.SeverityWarning)

	var notes []string
	for _, f := range rep.SetFindings {
		if f.Rule == skillcheck.RulePiNotice {
			notes = append(notes, f.Message)
		}
	}
	suffix := ""
	if len(notes) > 0 {
		suffix = "; " + strings.Join(notes, "; ")
	}
	if errs+warns == 0 {
		return doctor.Result{Status: doctor.StatusPass, Detail: fmt.Sprintf("%d skills checked, no issues%s", len(rep.Skills), suffix)}
	}

	var top []string
	collect := func(sev skillcheck.Severity) {
		for _, r := range rep.Skills {
			for _, f := range r.Findings {
				if f.Severity == sev && len(top) < 3 {
					top = append(top, fmt.Sprintf("%s: %s", r.Name, f.Rule))
				}
			}
		}
		for _, f := range rep.SetFindings {
			if f.Severity == sev && len(top) < 3 {
				top = append(top, f.Rule+": "+f.Message)
			}
		}
	}
	collect(skillcheck.SeverityError)
	collect(skillcheck.SeverityWarning)
	return doctor.Result{
		Status: doctor.StatusWarn,
		Detail: fmt.Sprintf("%d error(s), %d warning(s) across %d skills; e.g. %s%s", errs, warns, len(rep.Skills), strings.Join(top, "; "), suffix),
		Remedy: doctor.NewRemedy(doctor.RemedyFixSkills, skillsFixRemedy),
	}
}
