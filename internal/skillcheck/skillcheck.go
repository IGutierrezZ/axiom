// Package skillcheck es la única fuente de reglas para validar skills
// (`SKILL.md`). La usan el test de assets embebidos, las puertas de entrada
// (`axiom skill approve` y el minero) y las comprobaciones bajo
// demanda (`axiom doctor`, `axiom skill check`).
//
// El paquete no depende de la CLI ni escribe nada: informa de hallazgos con
// severidad y deja que cada llamador decida qué bloquea.
package skillcheck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Severity gradúa un hallazgo.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Profile decide la severidad de las reglas canónicas.
type Profile string

const (
	// Canonical: assets embebidos, skills minadas; las reglas canónicas son errores.
	Canonical Profile = "canonical"
	// Portable: skills de terceros y carpetas de usuario/proyecto; las reglas
	// canónicas son avisos.
	Portable Profile = "portable"
)

// Identificadores estables de regla (parte del contrato de `--json`).
const (
	RuleFrontmatterStart = "frontmatter-start"
	RuleFrontmatterClose = "frontmatter-close"
	RuleFrontmatterLine  = "frontmatter-line"
	RuleNameMissing      = "name-missing"
	RuleNameMismatch     = "name-mismatch"
	RuleNameReserved     = "name-reserved"
	RuleDescMissing      = "description-missing"
	RuleDescQuoted       = "description-quoted"
	RuleDescBlock        = "description-block-scalar"
	RuleDescMultiline    = "description-multiline"
	RuleDescTrigger      = "description-trigger"
	RuleDescBudget       = "description-budget"
	RuleKeysAllowed      = "keys-allowed"
	RuleReference        = "reference-missing"
	RuleUnreadable       = "unreadable"
)

// DefaultDescriptionBudget es el máximo de runas de `description`.
const DefaultDescriptionBudget = 160

var allowedKeys = map[string]bool{
	"name": true, "description": true, "license": true, "metadata": true,
	"version": true, "user-invocable": true, "disable-model-invocation": true,
}

// Finding es un hallazgo de una regla sobre una skill.
type Finding struct {
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
}

// Options ajusta una comprobación.
type Options struct {
	// Profile vacío equivale a Canonical.
	Profile Profile
	// DescriptionBudget en runas; 0 usa DefaultDescriptionBudget.
	DescriptionBudget int
	// SkipReservedName omite la regla de nombres reservados: los assets
	// embebidos contienen legítimamente `sdd-*` y `skill-registry`.
	SkipReservedName bool
	// Exists indica si una ruta relativa a la carpeta de la skill existe. Con
	// nil no se comprueban las referencias.
	Exists func(rel string) bool
}

func (o Options) profile() Profile {
	if o.Profile == "" {
		return Canonical
	}
	return o.Profile
}

// Report es el resultado de comprobar una skill.
type Report struct {
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	Profile  Profile   `json:"profile"`
	Findings []Finding `json:"findings"`
}

// HasErrors indica si algún hallazgo es un error.
func (r Report) HasErrors() bool { return count(r.Findings, SeverityError) > 0 }

// Errors devuelve solo los hallazgos de severidad error.
func (r Report) Errors() []Finding { return filter(r.Findings, SeverityError) }

// Count devuelve cuántos hallazgos tienen la severidad dada.
func (r Report) Count(s Severity) int { return count(r.Findings, s) }

func count(fs []Finding, s Severity) int { return len(filter(fs, s)) }

func filter(fs []Finding, s Severity) []Finding {
	var out []Finding
	for _, f := range fs {
		if f.Severity == s {
			out = append(out, f)
		}
	}
	return out
}

// IsReservedName replica las exclusiones silenciosas del registro de skills.
func IsReservedName(name string) bool {
	return name == "_shared" || name == "skill-registry" || strings.HasPrefix(name, "sdd-")
}

// CheckContent valida el contenido de un SKILL.md cuyo directorio se llama
// dirName. Es puro: no toca el disco salvo a través de opts.Exists.
func CheckContent(content, dirName string, opts Options) []Finding {
	prof := opts.profile()
	canon := SeverityError
	if prof == Portable {
		canon = SeverityWarning
	}

	fm, findings := parseFrontmatter(content)
	if hasRule(findings, RuleFrontmatterStart) || hasRule(findings, RuleFrontmatterClose) {
		// Frontmatter ausente o sin cerrar: no hay nada más que evaluar.
		return findings
	}

	// Estructurales (error en ambos perfiles).
	switch {
	case fm.name == "":
		findings = append(findings, Finding{RuleNameMissing, SeverityError, "falta el campo `name:`"})
	case fm.name != dirName:
		findings = append(findings, Finding{RuleNameMismatch, SeverityError,
			fmt.Sprintf("`name` es %q y debe coincidir con el directorio %q", fm.name, dirName)})
	}
	if !opts.SkipReservedName {
		for _, n := range uniqueNonEmpty(fm.name, dirName) {
			if IsReservedName(n) {
				findings = append(findings, Finding{RuleNameReserved, SeverityError,
					fmt.Sprintf("el nombre %q está reservado: el registro de skills lo descarta en silencio", n)})
				break
			}
		}
	}
	if !fm.descPresent || strings.TrimSpace(fm.description) == "" {
		findings = append(findings, Finding{RuleDescMissing, SeverityError, "falta el campo `description:` o está vacío"})
	} else {
		findings = append(findings, descriptionFindings(fm, opts, canon)...)
	}

	for _, key := range fm.keys {
		if !allowedKeys[key] {
			findings = append(findings, Finding{RuleKeysAllowed, canon,
				fmt.Sprintf("clave de primer nivel no permitida %q (permitidas: name, description, license, metadata, version, user-invocable, disable-model-invocation)", key)})
		}
	}

	if opts.Exists != nil {
		for _, ref := range missingReferences(fm.body, opts.Exists) {
			findings = append(findings, Finding{RuleReference, SeverityWarning,
				fmt.Sprintf("la referencia %q no existe dentro de la carpeta de la skill", ref)})
		}
	}
	return findings
}

func descriptionFindings(fm frontmatter, opts Options, canon Severity) []Finding {
	var out []Finding
	switch {
	case fm.descBlock:
		out = append(out, Finding{RuleDescBlock, canon,
			fmt.Sprintf("`description` usa un escalar de bloque (`%s`); debe ser un escalar entrecomillado en una línea", fm.descRaw[:1])})
	case fm.descMulti:
		out = append(out, Finding{RuleDescMultiline, canon, "`description` ocupa varias líneas; debe ir en una sola"})
	case !isQuoted(fm.descRaw):
		out = append(out, Finding{RuleDescQuoted, canon,
			fmt.Sprintf("`description` debe ir entrecomillada para ser YAML seguro; valor crudo: %q", fm.descRaw)})
	}
	if !strings.Contains(fm.description, "Trigger:") {
		out = append(out, Finding{RuleDescTrigger, canon, "`description` debe contener `Trigger:`"})
	}
	budget := opts.DescriptionBudget
	if budget <= 0 {
		budget = DefaultDescriptionBudget
	}
	if n := len([]rune(fm.description)); n > budget {
		out = append(out, Finding{RuleDescBudget, canon,
			fmt.Sprintf("`description` mide %d runas y el máximo es %d", n, budget)})
	}
	return out
}

// CheckDir valida la skill de la carpeta dir (lee dir/SKILL.md) e incluye la
// comprobación de referencias contra el disco.
func CheckDir(dir string, opts Options) Report {
	rep := Report{Path: filepath.Join(dir, "SKILL.md"), Name: filepath.Base(dir), Profile: opts.profile()}
	data, err := os.ReadFile(rep.Path)
	if err != nil {
		rep.Findings = []Finding{{RuleUnreadable, SeverityError, "no se puede leer SKILL.md: " + err.Error()}}
		return rep
	}
	if opts.Exists == nil {
		opts.Exists = func(rel string) bool {
			_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
			return err == nil
		}
	}
	rep.Findings = CheckContent(string(data), filepath.Base(dir), opts)
	if fm, _ := parseFrontmatter(string(data)); fm.name != "" {
		rep.Name = fm.name
	}
	return rep
}

func hasRule(fs []Finding, rule string) bool {
	for _, f := range fs {
		if f.Rule == rule {
			return true
		}
	}
	return false
}

func uniqueNonEmpty(vals ...string) []string {
	var out []string
	seen := map[string]bool{}
	for _, v := range vals {
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
