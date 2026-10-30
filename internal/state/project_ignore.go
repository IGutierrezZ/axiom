package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
)

// excludeHeader encabeza las líneas que Axiom añade a `.git/info/exclude`.
const excludeHeader = "# Axiom: estado local del proyecto (no versionar)"

// EnsureProjectStateIgnored excluye de git el estado de proyecto de root (el
// fichero y su cerrojo) escribiendo en `.git/info/exclude` del repositorio al que
// pertenece root. No hace nada si root no está en un repositorio git.
//
// Se usa `info/exclude` y no `.gitignore` porque el estado es personal y local: no
// ensucia el árbol de trabajo del usuario ni añade un cambio que haya que
// commitear en proyectos ajenos. El patrón excluye solo los dos ficheros y no el
// directorio `.axiom/`, que ya tiene otros usos que sí pueden versionarse
// (`.axiom/inbox/skills/`, la caché y el buzón de autoskills). En un worktree
// enlazado el fichero es el del git common dir, el que git lee en todos ellos.
//
// Es idempotente y trabaja solo sobre ficheros, sin lanzar procesos.
func EnsureProjectStateIgnored(root string) error {
	gitRoot, ok := gitcommon.GitRootFromFiles(root)
	if !ok {
		return nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolver la raíz del proyecto %q: %w", root, err)
	}
	rel, err := filepath.Rel(gitRoot, absRoot)
	if err != nil {
		return fmt.Errorf("situar %s dentro de %s: %w", absRoot, gitRoot, err)
	}
	commonDir, err := gitcommon.CommonDirFromFiles(gitRoot)
	if err != nil {
		return err
	}
	excludePath := filepath.Join(commonDir, "info", "exclude")

	existingBytes, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("leer %s: %w", excludePath, err)
	}
	existing := string(existingBytes)
	covered := excludeCoveredLines(existing)

	prefix := "/"
	if rel != "." {
		prefix += filepath.ToSlash(rel) + "/"
	}
	var missing []string
	for _, entry := range []string{prefix + ".axiom/" + stateFile, prefix + ".axiom/" + stateFile + ".lock"} {
		if !covered[entry] {
			missing = append(missing, entry)
		}
	}
	if len(missing) == 0 || coversWholeAxiomDir(covered, prefix) {
		return nil
	}

	var out strings.Builder
	out.WriteString(existing)
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		out.WriteString("\n")
	}
	if !strings.Contains(existing, excludeHeader) {
		out.WriteString(excludeHeader + "\n")
	}
	for _, entry := range missing {
		out.WriteString(entry + "\n")
	}
	if err := os.MkdirAll(filepath.Dir(excludePath), 0o755); err != nil {
		return fmt.Errorf("crear %s: %w", filepath.Dir(excludePath), err)
	}
	if _, err := filemerge.WriteFileAtomic(excludePath, []byte(out.String()), 0o644); err != nil {
		return fmt.Errorf("escribir %s: %w", excludePath, err)
	}
	return nil
}

func excludeCoveredLines(content string) map[string]bool {
	covered := make(map[string]bool)
	for _, line := range strings.Split(content, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			covered[trimmed] = true
		}
	}
	return covered
}

// coversWholeAxiomDir reconoce una exclusión previa de `.axiom/` entera para este
// proyecto, que ya cubre el estado.
func coversWholeAxiomDir(covered map[string]bool, prefix string) bool {
	return covered[prefix+".axiom/"] || covered[prefix+".axiom"] ||
		(prefix == "/" && (covered[".axiom/"] || covered[".axiom"]))
}
