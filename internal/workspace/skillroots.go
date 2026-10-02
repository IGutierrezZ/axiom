package workspace

import (
	"path"
	"path/filepath"
	"strings"
)

// CleanSkillRoot valida una entrada de 'workspace.skill_roots' y la devuelve
// normalizada con el separador del sistema, lista para unirla a la raíz del
// proyecto. Es válida la ruta relativa que, una vez limpia, queda dentro del
// proyecto y no es el propio proyecto: "." haría que se escaneara todo el
// repositorio como si fuera un directorio de skills.
//
// La comprobación es la misma en todos los sistemas operativos: axiom.yaml se
// versiona y lo comparten clones en Windows y Linux, así que una ruta como
// "C:\skills" o "/opt/skills" se rechaza también donde no sería absoluta.
func CleanSkillRoot(root string) (string, bool) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", false
	}
	slashed := strings.ReplaceAll(root, "\\", "/")
	if strings.HasPrefix(slashed, "/") || hasDriveLetter(slashed) || filepath.IsAbs(root) {
		return "", false
	}
	clean := path.Clean(slashed)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	return filepath.FromSlash(clean), true
}

// hasDriveLetter detecta "C:" y "C:ruta": no son absolutas en Windows, pero
// tampoco son relativas al proyecto.
func hasDriveLetter(p string) bool {
	if len(p) < 2 || p[1] != ':' {
		return false
	}
	c := p[0]
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// ValidSkillRoots devuelve las raíces de skills declaradas que superan
// CleanSkillRoot, en el orden de declaración. Las inválidas se ignoran; Validate
// es quien las reporta.
func (s WorkspaceSection) ValidSkillRoots() []string {
	var roots []string
	for _, root := range s.SkillRoots {
		if clean, ok := CleanSkillRoot(root); ok {
			roots = append(roots, clean)
		}
	}
	return roots
}
