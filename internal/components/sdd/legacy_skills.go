package sdd

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/components/skills"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// Migración de las skills de proyecto de las instalaciones workspace anteriores.
//
// Hasta ahora, con --scope workspace cada agente recibía su propia copia de las
// skills (.opencode/skills, .kilo/skills, .gemini/skills, .cursor/skills,
// .github/skills, .codex/skills). Esos agentes leen además .agents/skills, donde
// Axiom escribe ahora una única copia compartida, así que las carpetas antiguas
// harían aparecer cada skill dos veces (en OpenCode, avisos «duplicate skill name»).
//
// Una carpeta de skill antigua solo se retira cuando se prueba que es de Axiom:
//   - su nombre es el de una skill que Axiom embebe (de las ordinarias, de SDD o la
//     carpeta de soporte _shared);
//   - el conjunto de ficheros que contiene es exactamente el que Axiom genera para
//     esa skill, sin ficheros de más ni subcarpetas ajenas, y cada uno es byte a byte
//     igual a lo que Axiom genera hoy (con cualquiera de las variantes que admite:
//     capacidad capable o small, e identidad de runtime del agente o genérica);
//   - la copia que la sustituye ya existe en la carpeta compartida.
//
// Una carpeta que difiere, que tiene ficheros extra o que no es un directorio normal
// se conserva con un aviso. Las skills ajenas (cualquier otro nombre) no se tocan, y la
// carpeta contenedora solo se borra si queda vacía.

// LegacySkillsResult es el resultado de retirar las skills de una carpeta antigua.
type LegacySkillsResult struct {
	// Removed lista los ficheros borrados.
	Removed []string
	// Notes lista los avisos de las carpetas conservadas.
	Notes []string
}

// legacySkillsPlan es lo que la retirada borraría: la lectura es previa y sin
// escribir nada, para que la copia de seguridad previa pueda incluir esos ficheros.
type legacySkillsPlan struct {
	files []string
	dirs  []string
	notes []string
}

// managedSkillBytes es, por ruta relativa a la carpeta de skills, el conjunto de
// contenidos que Axiom genera hoy para ese fichero.
type managedSkillBytes map[string][][]byte

// capturingWriter recoge lo que escribiría la inyección en lugar de escribirlo, para
// obtener los bytes exactos que Axiom genera sin duplicar su lógica de generación.
type capturingWriter struct {
	root    string
	content managedSkillBytes
}

func (c *capturingWriter) write(path string, data []byte, _ fs.FileMode) (filemerge.WriteResult, error) {
	rel, err := filepath.Rel(c.root, path)
	if err != nil {
		return filemerge.WriteResult{}, err
	}
	rel = filepath.ToSlash(rel)
	for _, existing := range c.content[rel] {
		if bytes.Equal(existing, data) {
			return filemerge.WriteResult{}, nil
		}
	}
	c.content[rel] = append(c.content[rel], append([]byte(nil), data...))
	return filemerge.WriteResult{}, nil
}

func noLegacyMarker(string) (bool, error) { return false, nil }

// generatedManagedSkills devuelve todo lo que Axiom genera hoy en una carpeta de
// skills para el agente dado: las skills ordinarias, y las de SDD con _shared en cada
// variante (capable y small; identidad del agente y genérica).
func generatedManagedSkills(agent model.AgentID) (managedSkillBytes, error) {
	const virtualRoot = "skills-root"
	root := filepath.FromSlash(virtualRoot)
	capture := &capturingWriter{root: root, content: managedSkillBytes{}}

	entries, err := fs.ReadDir(assets.FS, "skills")
	if err != nil {
		return nil, fmt.Errorf("enumerar las skills embebidas: %w", err)
	}
	var ordinary []model.SkillID
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "_shared" || strings.HasPrefix(entry.Name(), "sdd-") {
			continue
		}
		ordinary = append(ordinary, model.SkillID(entry.Name()))
	}
	if _, err := skills.InjectDirectoryWithWriter(root, ordinary, capture.write); err != nil {
		return nil, fmt.Errorf("generar las skills ordinarias: %w", err)
	}
	for _, capability := range []string{"capable", "small"} {
		for _, identity := range []model.AgentID{agent, ""} {
			if _, err := injectSkillDirectoryWithWriter(root, capability, identity, capture.write, noLegacyMarker); err != nil {
				return nil, fmt.Errorf("generar las skills de SDD: %w", err)
			}
		}
	}
	return capture.content, nil
}

// planLegacyWorkspaceSkills decide qué se retiraría de legacyRoot. sharedRoot es la
// carpeta que sustituye a la antigua.
func planLegacyWorkspaceSkills(legacyRoot, sharedRoot string, agent model.AgentID, requireReplacement bool) legacySkillsPlan {
	var plan legacySkillsPlan
	info, err := os.Lstat(legacyRoot)
	if err != nil {
		if !os.IsNotExist(err) {
			plan.notes = append(plan.notes, fmt.Sprintf("No se pudo leer la carpeta de skills anterior %s (%v): revísala a mano, porque las skills pueden aparecer duplicadas con .agents/skills.", legacyRoot, err))
		}
		return plan
	}
	if !info.IsDir() || info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
		plan.notes = append(plan.notes, fmt.Sprintf("La carpeta de skills anterior %s no es un directorio normal y se conserva: revísala a mano, porque las skills pueden aparecer duplicadas con .agents/skills.", legacyRoot))
		return plan
	}

	generated, err := generatedManagedSkills(agent)
	if err != nil {
		plan.notes = append(plan.notes, fmt.Sprintf("No se pudo comprobar qué skills de %s son de Axiom (%v): se conservan.", legacyRoot, err))
		return plan
	}
	// Ficheros que Axiom genera, agrupados por la carpeta de skill (primer componente).
	byName := map[string][]string{}
	for rel := range generated {
		name := rel
		if i := strings.IndexByte(rel, '/'); i >= 0 {
			name = rel[:i]
		}
		byName[name] = append(byName[name], rel)
	}

	children, err := os.ReadDir(legacyRoot)
	if err != nil {
		plan.notes = append(plan.notes, fmt.Sprintf("No se pudo leer la carpeta de skills anterior %s (%v): revísala a mano.", legacyRoot, err))
		return plan
	}
	names := make([]string, 0, len(children))
	for _, child := range children {
		names = append(names, child.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		want, managed := byName[name]
		if !managed {
			continue // skill ajena: no se toca
		}
		dir := filepath.Join(legacyRoot, name)
		files, dirs, reason := proveManagedSkillDir(dir, want, generated)
		if reason != "" {
			plan.notes = append(plan.notes, fmt.Sprintf("La skill %s se conservó: %s. Axiom ahora escribe las skills en %s, así que puede aparecer duplicada: revísala y bórrala a mano si ya no la necesitas.", dir, reason, sharedRoot))
			continue
		}
		if requireReplacement && !sharedCopyExists(sharedRoot, name, want) {
			continue // la copia nueva aún no está: no se retira lo único que hay
		}
		plan.files = append(plan.files, files...)
		plan.dirs = append(plan.dirs, dirs...)
		plan.dirs = append(plan.dirs, dir)
	}
	return plan
}

// proveManagedSkillDir comprueba que dir contiene exactamente los ficheros de want,
// todos regulares y byte a byte iguales a alguna de las variantes que Axiom genera.
// Devuelve los ficheros y las subcarpetas (de la más profunda a la menos) que se
// borrarían, o la razón por la que la carpeta no se puede dar por de Axiom.
func proveManagedSkillDir(dir string, want []string, generated managedSkillBytes) (files, dirs []string, reason string) {
	wantSet := make(map[string]struct{}, len(want))
	for _, rel := range want {
		wantSet[rel] = struct{}{}
	}
	base := filepath.Dir(dir)
	found := map[string]struct{}{}
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
			return fmt.Errorf("contiene un enlace o un fichero especial (%s)", path)
		}
		if entry.IsDir() {
			if path != dir {
				dirs = append([]string{path}, dirs...)
			}
			return nil
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if _, ok := wantSet[rel]; !ok {
			return fmt.Errorf("tiene el fichero %s, que Axiom no genera", rel)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		matches := false
		for _, candidate := range generated[rel] {
			if bytes.Equal(candidate, data) {
				matches = true
				break
			}
		}
		if !matches {
			return fmt.Errorf("el fichero %s difiere de lo que Axiom genera hoy", rel)
		}
		found[rel] = struct{}{}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, nil, err.Error()
	}
	for rel := range wantSet {
		if _, ok := found[rel]; !ok {
			return nil, nil, fmt.Sprintf("le falta el fichero %s: no es una copia completa de Axiom", rel)
		}
	}
	return files, dirs, ""
}

// sharedCopyExists indica si la carpeta compartida ya tiene la skill name completa.
func sharedCopyExists(sharedRoot, name string, want []string) bool {
	for _, rel := range want {
		info, err := os.Lstat(filepath.Join(sharedRoot, filepath.FromSlash(rel)))
		if err != nil || !info.Mode().IsRegular() {
			return false
		}
	}
	return true
}

// LegacyWorkspaceSkillFiles lista los ficheros que RetireLegacyWorkspaceSkills
// borraría de legacyRoot, para que la copia de seguridad previa los guarde y una
// vuelta atrás los restaure. No escribe nada. Como la copia de seguridad se hace antes
// de instalar, no exige que la copia compartida exista ya: es el máximo que se podría borrar.
func LegacyWorkspaceSkillFiles(legacyRoot, sharedRoot string, agent model.AgentID) []string {
	plan := planLegacyWorkspaceSkills(legacyRoot, sharedRoot, agent, false)
	files := append([]string(nil), plan.files...)
	files = append(files, skills.InspectRetired(legacyRoot).Owned...)
	return files
}

// RetireLegacyWorkspaceSkills retira de legacyRoot (la carpeta de skills que usaba
// antes el agente en el proyecto) las skills que se prueba que son de Axiom, siempre
// que su copia ya esté en sharedRoot. Retira además las skills que Axiom ya no
// distribuye. Lo demás se conserva: devuelve un aviso por cada carpeta de Axiom que
// difiere, para que el usuario la revise. Nunca falla ni toca skills ajenas; la
// carpeta contenedora se borra solo si queda vacía.
func RetireLegacyWorkspaceSkills(legacyRoot, sharedRoot string, agent model.AgentID) LegacySkillsResult {
	var result LegacySkillsResult
	// La prueba se repite justo antes de borrar: no se fía de una inspección anterior.
	plan := planLegacyWorkspaceSkills(legacyRoot, sharedRoot, agent, true)
	result.Notes = append(result.Notes, plan.notes...)
	for _, path := range plan.files {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			result.Notes = append(result.Notes, fmt.Sprintf("No se pudo retirar %s (%v): bórralo a mano, porque la skill puede aparecer duplicada con %s.", path, err, sharedRoot))
			continue
		}
		result.Removed = append(result.Removed, path)
	}
	// Remove, no RemoveAll: falla en una carpeta con contenido y así conserva lo que
	// el usuario haya añadido.
	for _, dir := range plan.dirs {
		_ = os.Remove(dir)
	}
	result.Removed = append(result.Removed, skills.RetireInstalled(legacyRoot).Owned...)
	// La carpeta contenedora solo se borra si queda vacía.
	if entries, err := os.ReadDir(legacyRoot); err == nil && len(entries) == 0 {
		_ = os.Remove(legacyRoot)
	}
	return result
}
