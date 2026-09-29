package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SpecEntry resume la información de una especificación viva para el catálogo.
type SpecEntry struct {
	Domain      string
	Title       string
	ReqCount    int
	ScenarioCnt int
	RelativeURL string
}

// SyncIndex recalcula y sincroniza el catálogo maestro openspec/INDEX.md con las specs en openspec/specs/.
func SyncIndex(projectName, specsRoot string) error {
	specsDir := filepath.Join(specsRoot, "specs")
	indexPath := filepath.Join(specsRoot, "INDEX.md")

	entries := make([]SpecEntry, 0)

	if info, err := os.Stat(specsDir); err == nil && info.IsDir() {
		domainDirs, err := os.ReadDir(specsDir)
		if err == nil {
			for _, d := range domainDirs {
				if !d.IsDir() {
					continue
				}
				domain := d.Name()
				specFile := filepath.Join(specsDir, domain, "spec.md")
				content, err := os.ReadFile(specFile)
				if err != nil {
					continue
				}

				entry := parseSpecMetadata(domain, string(content))
				entry.RelativeURL = fmt.Sprintf("specs/%s/spec.md", domain)
				entries = append(entries, entry)
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Catálogo Maestro de Especificaciones Vivas — %s\n\n", projectName))
	sb.WriteString(fmt.Sprintf("> **Proyecto:** %s\n> **Perfil:** Knowledge\n> **Total Dominios:** %d\n\n", projectName, len(entries)))
	sb.WriteString("---\n\n## Resumen de Especificaciones Vivas\n\n")
	sb.WriteString("| Dominio | Título de la Especificación | Reqs | Escenarios | Enlace |\n")
	sb.WriteString("| :--- | :--- | :---: | :---: | :--- |\n")

	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("| `%s` | %s | %d | %d | [Ver Spec](%s) |\n", e.Domain, e.Title, e.ReqCount, e.ScenarioCnt, e.RelativeURL))
	}

	return os.WriteFile(indexPath, []byte(sb.String()), 0644)
}

var reqRegex = regexp.MustCompile(`(?i)#{2,4}\s+Requirement:`)
var scenarioRegex = regexp.MustCompile(`(?i)#{3,5}\s+Scenario:`)

func parseSpecMetadata(domain, content string) SpecEntry {
	entry := SpecEntry{
		Domain:      domain,
		Title:       "Especificación Viva: " + strings.Title(domain),
		ReqCount:    len(reqRegex.FindAllString(content, -1)),
		ScenarioCnt: len(scenarioRegex.FindAllString(content, -1)),
	}

	lines := strings.Split(content, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimPrefix(trimmed, "# ")
			if title != "" {
				entry.Title = title
				break
			}
		}
	}

	return entry
}
