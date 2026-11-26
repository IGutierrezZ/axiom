package assets

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// skillsIndexRowPaths devuelve la ruta de cada fila de la tabla de skills de
// AGENTS.md cuya celda de nombre es `name` (con o sin comillas invertidas
// alrededor). La ruta se lee de la última celda y puede venir en texto plano, entre
// comillas invertidas o como enlace markdown; se devuelve siempre en texto plano.
// Se toman la primera y la última celda, no un número fijo de columnas, para no
// depender del texto de la descripción.
func skillsIndexRowPaths(agents, name string) []string {
	var paths []string
	for _, line := range strings.Split(agents, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if len(cells) < 2 || strings.Trim(strings.TrimSpace(cells[0]), "`") != name {
			continue
		}
		cell := strings.TrimSpace(cells[len(cells)-1])
		if strings.HasPrefix(cell, "[") {
			if text, _, ok := strings.Cut(cell[1:], "]("); ok {
				cell = text
			}
		}
		paths = append(paths, strings.Trim(strings.TrimSpace(cell), "`"))
	}
	return paths
}

func TestIssueCreationAuthorityBoundary(t *testing.T) {
	repositoryRoot := filepath.Join("..", "..")
	duplicatePath := filepath.Join(repositoryRoot, "skills", "issue-creation", "SKILL.md")
	if _, err := os.Stat(duplicatePath); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			t.Fatalf("duplicate project issue-creation authority still exists at %s", duplicatePath)
		}
		t.Fatalf("stat duplicate project issue-creation authority: %v", err)
	}

	agents, err := os.ReadFile(filepath.Join(repositoryRoot, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	// El bloque de skills de AGENTS.md lo genera `axiom skill index refresh`, y la
	// descripción de cada fila sale del frontmatter del SKILL.md. Por eso no se
	// compara la fila literal: se exige la identidad (celda de nombre) y la ruta.
	const canonicalIssueCreationPath = "internal/assets/skills/issue-creation/SKILL.md"
	paths := skillsIndexRowPaths(string(agents), "issue-creation")
	if len(paths) == 0 {
		t.Fatalf("AGENTS.md must route the canonical issue-creation identity directly to the embedded authority; missing a skills-table row with name `issue-creation` and path %q", canonicalIssueCreationPath)
	}
	for _, path := range paths {
		if path != canonicalIssueCreationPath {
			t.Fatalf("AGENTS.md row `issue-creation` points to %q, want the embedded authority %q", path, canonicalIssueCreationPath)
		}
	}

	canonical := MustRead("skills/issue-creation/SKILL.md")
	if !strings.Contains(canonical, "\nname: issue-creation\n") {
		t.Fatal("embedded issue-creation authority must retain canonical frontmatter identity name: issue-creation")
	}
}

// El parseo de la fila no puede depender de la descripción ni del formato de la
// celda de ruta, que ha cambiado entre versiones del generador.
func TestSkillsIndexRowPathsAcceptsEveryPathFormat(t *testing.T) {
	const want = "internal/assets/skills/issue-creation/SKILL.md"
	for name, row := range map[string]string{
		"texto plano":       "| `issue-creation` | Descripción cualquiera. | project | " + want + " |",
		"comillas":          "| `issue-creation` | Descripción cualquiera. | project | `" + want + "` |",
		"enlace":            "| `issue-creation` | Descripción cualquiera. | project | [`" + want + "`](" + want + ") |",
		"tres columnas":     "| `issue-creation` | Descripción cualquiera. | [`" + want + "`](" + want + ") |",
		"con barra en desc": "| `issue-creation` | Crea issues | y las triagea. | project | `" + want + "` |",
		"fin de línea CRLF": "| `issue-creation` | Descripción cualquiera. | project | `" + want + "` |\r",
	} {
		t.Run(name, func(t *testing.T) {
			agents := "# Doc\n\n| Skill | Trigger / description | Scope | Path |\n| --- | --- | --- | --- |\n" + row + "\n"
			got := skillsIndexRowPaths(agents, "issue-creation")
			if len(got) != 1 || got[0] != want {
				t.Fatalf("skillsIndexRowPaths() = %q, want [%q]", got, want)
			}
		})
	}

	otherRows := "| Skill | Trigger / description | Scope | Path |\n" +
		"| --- | --- | --- | --- |\n" +
		"| `go-testing` | Menciona `issue-creation` en la descripción. | project | `internal/assets/skills/go-testing/SKILL.md` |\n" +
		"| `issue-creation-extra` | Otra skill. | project | `skills/issue-creation-extra/SKILL.md` |\n"
	if got := skillsIndexRowPaths(otherRows, "issue-creation"); len(got) != 0 {
		t.Fatalf("skillsIndexRowPaths() = %q, want none: only the name cell identifies the row", got)
	}
}
