package skillcheck

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/skillregistry"
)

// Identificadores de las reglas de conjunto (parte del contrato de `--json`).
const (
	RuleCollision      = "collision"
	RuleReservedDrop   = "reserved-dropped"
	RuleFolderScope    = "folder-scope"
	RulePiNotice       = "pi-notice"
	RuleRegistryState  = "registry-freshness"
	PiRuntime          = "pi"
	piNoticeMessage    = "Pi: las skills llegan por gentle-pi, fuera de Axiom (Axiom no instala skills en Pi)"
	registryRefreshCmd = "`axiom skill index refresh`"
)

// SetOptions configura el escaneo de las carpetas de skills de proyecto y usuario.
type SetOptions struct {
	// FolderRuntimes dice qué runtimes cargan cada carpeta de skills (clave: la
	// ruta de la carpeta; valor: ids de runtime). Lo construye el llamador a
	// partir de los adaptadores, razonando por carpeta porque varios runtimes
	// comparten una (p. ej. `.agents/skills`). Con nil no se emiten avisos de
	// alcance.
	FolderRuntimes map[string][]string
	// PiRelevant fuerza el aviso informativo de Pi (p. ej. Pi está instalado).
	// También se emite cuando alguna skill vive en una carpeta de Pi.
	PiRelevant bool
}

// SetReport es el resultado de escanear el conjunto de skills visibles.
type SetReport struct {
	Skills      []Report  `json:"skills"`
	SetFindings []Finding `json:"set_findings"`
}

// Count suma los hallazgos de severidad s, por skill y de conjunto.
func (s SetReport) Count(sev Severity) int {
	n := count(s.SetFindings, sev)
	for _, r := range s.Skills {
		n += r.Count(sev)
	}
	return n
}

// HasErrors indica si hay algún error, por skill o de conjunto.
func (s SetReport) HasErrors() bool { return s.Count(SeverityError) > 0 }

type scanned struct {
	dir    string // carpeta de la skill
	file   string // SKILL.md
	root   string // carpeta de skills que la contiene
	report Report
}

// ScanSet comprueba, con el perfil portable, cada skill de las carpetas que el
// registro escanea (proyecto y usuario, un nivel de profundidad como el
// registro) y añade los hallazgos de conjunto: colisiones, nombres reservados
// que el registro descarta, alcance por carpeta, aviso de Pi y frescura del
// registro. Nunca escribe nada y los hallazgos de conjunto nunca son errores.
func ScanSet(cwd, home string, opts SetOptions) SetReport {
	var items []scanned
	for _, root := range skillregistry.ScanDirs(cwd, home) {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			dir := filepath.Join(root, e.Name())
			file := filepath.Join(dir, "SKILL.md")
			if !isDir(dir) || !isFile(file) {
				continue
			}
			// El nombre reservado no es un error aquí: es un descarte silencioso
			// del registro y se informa como hallazgo de conjunto.
			rep := CheckDir(dir, Options{Profile: Portable, SkipReservedName: true})
			items = append(items, scanned{dir: dir, file: file, root: root, report: rep})
		}
	}

	out := SetReport{Skills: make([]Report, 0, len(items)), SetFindings: []Finding{}}
	for _, it := range items {
		out.Skills = append(out.Skills, it.report)
	}
	out.SetFindings = append(out.SetFindings, collisionFindings(cwd, items)...)
	out.SetFindings = append(out.SetFindings, reservedFindings(items)...)
	out.SetFindings = append(out.SetFindings, scopeFindings(items, opts)...)
	out.SetFindings = append(out.SetFindings, registryFindings(cwd, home, len(items))...)
	return out
}

func collisionFindings(cwd string, items []scanned) []Finding {
	byName := map[string][]scanned{}
	var order []string
	for _, it := range items {
		name := it.report.Name
		if skillregistry.IsReservedName(name) {
			continue
		}
		if _, ok := byName[name]; !ok {
			order = append(order, name)
		}
		byName[name] = append(byName[name], it)
	}
	var out []Finding
	for _, name := range order {
		group := byName[name]
		if len(group) < 2 {
			continue
		}
		copies := make([]string, len(group))
		for i, it := range group {
			copies[i] = it.file
		}
		chosen := skillregistry.ChosenCopy(cwd, copies)

		first, _ := os.ReadFile(group[0].file)
		identical := true
		for _, it := range group[1:] {
			data, _ := os.ReadFile(it.file)
			if !bytes.Equal(first, data) {
				identical = false
				break
			}
		}
		sev, kind := SeverityWarning, "con contenido distinto"
		if identical {
			sev, kind = SeverityInfo, "copias idénticas"
		}
		out = append(out, Finding{Rule: RuleCollision, Severity: sev,
			Message: fmt.Sprintf("la skill %q existe en %d carpetas (%s): %s; el registro elige %s",
				name, len(group), kind, strings.Join(copies, ", "), chosen)})
	}
	return out
}

func reservedFindings(items []scanned) []Finding {
	var out []Finding
	for _, it := range items {
		if skillregistry.IsReservedName(it.report.Name) || skillregistry.IsReservedName(filepath.Base(it.dir)) {
			out = append(out, Finding{Rule: RuleReservedDrop, Severity: SeverityInfo,
				Message: fmt.Sprintf("%s: el nombre %q está reservado y el registro de skills la descarta en silencio", it.file, it.report.Name)})
		}
	}
	return out
}

func scopeFindings(items []scanned, opts SetOptions) []Finding {
	if opts.FolderRuntimes == nil && !opts.PiRelevant {
		return nil
	}
	perFolder := map[string]int{}
	var folders []string
	for _, it := range items {
		if perFolder[it.root] == 0 {
			folders = append(folders, it.root)
		}
		perFolder[it.root]++
	}
	sort.Strings(folders)

	var out []Finding
	pi := opts.PiRelevant
	for _, folder := range folders {
		runtimes := lookupRuntimes(opts.FolderRuntimes, folder)
		for _, r := range runtimes {
			if r == PiRuntime {
				pi = true
			}
		}
		if opts.FolderRuntimes == nil {
			continue
		}
		msg := fmt.Sprintf("%s (%d skills): ", folder, perFolder[folder])
		if len(runtimes) == 0 {
			msg += "ningún runtime la carga directamente; solo la indexa el registro de skills"
		} else {
			msg += "la cargan " + strings.Join(runtimes, ", ")
		}
		out = append(out, Finding{Rule: RuleFolderScope, Severity: SeverityInfo, Message: msg})
	}
	if pi {
		out = append(out, Finding{Rule: RulePiNotice, Severity: SeverityInfo, Message: piNoticeMessage})
	}
	return out
}

func registryFindings(cwd, home string, skills int) []Finding {
	switch skillregistry.RegistryFreshness(cwd, home) {
	case skillregistry.FreshnessStale:
		return []Finding{{Rule: RuleRegistryState, Severity: SeverityWarning,
			Message: "el registro de skills está desactualizado respecto al disco; ejecuta " + registryRefreshCmd}}
	case skillregistry.FreshnessAbsent:
		if skills == 0 {
			return nil
		}
		return []Finding{{Rule: RuleRegistryState, Severity: SeverityInfo,
			Message: "no hay registro de skills generado; ejecuta " + registryRefreshCmd}}
	}
	return nil
}

func lookupRuntimes(m map[string][]string, folder string) []string {
	if m == nil {
		return nil
	}
	want := folderKey(folder)
	for k, v := range m {
		if folderKey(k) == want {
			out := append([]string(nil), v...)
			sort.Strings(out)
			return out
		}
	}
	return nil
}

func folderKey(p string) string {
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	return p
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func isFile(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
