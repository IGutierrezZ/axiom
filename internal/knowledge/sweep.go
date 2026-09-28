package knowledge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/hub"
)

var ignoredDirs = map[string]bool{
	".git":         true,
	".github":      true,
	".axiom":       true,
	".openspec":    true,
	"openspec":     true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"bin":          true,
	".vscode":      true,
	".idea":        true,
	"testdata":     true,
}

// RunSweep ejecuta el barrido inicial rápido técnico y funcional de manera autónoma y no bloqueante.
func RunSweep(ctx context.Context, opts SweepOptions) (*SweepResult, error) {
	start := time.Now()
	if opts.WorkspaceRoot == "" {
		opts.WorkspaceRoot = "."
	}
	absWorkspace, err := filepath.Abs(opts.WorkspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("ruta workspace inválida: %w", err)
	}

	if opts.SpecsRoot == "" {
		opts.SpecsRoot = filepath.Join(absWorkspace, "openspec")
	}

	detector := hub.NewDetector()
	tech, _ := detector.Detect(absWorkspace)

	result := &SweepResult{
		PrimaryLanguage: "generic",
		Frameworks:      make([]string, 0),
		Entrypoints:     make([]string, 0),
		Modules:         make([]ModuleSummary, 0),
		Ambiguities:     make([]AmbiguityItem, 0),
		CreatedSpecs:    make([]string, 0),
	}

	if tech != nil {
		if tech.PrimaryLanguage != "" {
			result.PrimaryLanguage = tech.PrimaryLanguage
		}
		result.Frameworks = tech.Frameworks
	}

	// 1. Descubrir puntos de entrada
	result.Entrypoints = discoverEntrypoints(absWorkspace)

	// 2. Descubrir módulos funcionales preliminares
	modules := discoverModules(absWorkspace)
	result.Modules = modules

	// 3. Detectar ambigüedades / inconsistencias técnicas (de forma no bloqueante)
	ambiguities := detectAmbiguities(absWorkspace, modules)
	result.Ambiguities = ambiguities

	// 4. Sembrar borradores canónicos en openspec/specs/<modulo>/spec.md
	specsDir := filepath.Join(opts.SpecsRoot, "specs")
	_ = os.MkdirAll(specsDir, 0755)

	projectName := filepath.Base(absWorkspace)
	for _, mod := range modules {
		modDomain := slugify(mod.Name)
		modSpecDir := filepath.Join(specsDir, modDomain)
		modSpecFile := filepath.Join(modSpecDir, "spec.md")

		if _, err := os.Stat(modSpecFile); os.IsNotExist(err) {
			_ = os.MkdirAll(modSpecDir, 0755)
			specContent := generateDraftSpec(projectName, mod.Name, mod.Description, mod.Components)
			if err := os.WriteFile(modSpecFile, []byte(specContent), 0644); err == nil {
				result.CreatedSpecs = append(result.CreatedSpecs, modSpecFile)
			}
		}
	}

	// 5. Sincronizar catálogo maestro openspec/INDEX.md
	_ = SyncIndex(projectName, opts.SpecsRoot)

	result.Duration = time.Since(start)
	return result, nil
}

func discoverEntrypoints(root string) []string {
	var entries []string
	commonEntryFiles := []string{
		"main.go", "index.ts", "index.js", "app.py", "main.py",
		"Program.cs", "Startup.cs", "src/main.rs", "src/index.ts", "src/main.go",
	}

	for _, f := range commonEntryFiles {
		full := filepath.Join(root, f)
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			entries = append(entries, filepath.ToSlash(f))
		}
	}

	// Buscar en cmd/*/main.go
	cmdDir := filepath.Join(root, "cmd")
	if info, err := os.Stat(cmdDir); err == nil && info.IsDir() {
		subdirs, _ := os.ReadDir(cmdDir)
		for _, s := range subdirs {
			if s.IsDir() {
				mainFile := filepath.Join(cmdDir, s.Name(), "main.go")
				if _, err := os.Stat(mainFile); err == nil {
					rel, _ := filepath.Rel(root, mainFile)
					entries = append(entries, filepath.ToSlash(rel))
				}
			}
		}
	}

	return entries
}

func discoverModules(root string) []ModuleSummary {
	var modules []ModuleSummary
	candidates := []string{root}

	for _, sub := range []string{"internal", "src", "packages", "services", "apps", "pkg"} {
		subPath := filepath.Join(root, sub)
		if info, err := os.Stat(subPath); err == nil && info.IsDir() {
			candidates = append(candidates, subPath)
		}
	}

	seen := make(map[string]bool)
	for _, baseDir := range candidates {
		dirs, err := os.ReadDir(baseDir)
		if err != nil {
			continue
		}

		for _, d := range dirs {
			if !d.IsDir() {
				continue
			}
			name := d.Name()
			if ignoredDirs[name] || strings.HasPrefix(name, ".") {
				continue
			}

			fullPath := filepath.Join(baseDir, name)
			relPath, _ := filepath.Rel(root, fullPath)
			relSlash := filepath.ToSlash(relPath)

			if seen[name] {
				continue
			}
			seen[name] = true

			// Inspeccionar componentes dentro del directorio
			var components []string
			subEntries, _ := os.ReadDir(fullPath)
			for _, se := range subEntries {
				if !se.IsDir() && !strings.HasPrefix(se.Name(), ".") {
					components = append(components, se.Name())
				}
			}

			modules = append(modules, ModuleSummary{
				Name:        name,
				Path:        relSlash,
				Description: fmt.Sprintf("Módulo funcional identificado en '%s'", relSlash),
				Components:  components,
			})
		}
	}

	return modules
}

func detectAmbiguities(root string, modules []ModuleSummary) []AmbiguityItem {
	var ambiguities []AmbiguityItem

	// Verificar conflicto de gestores de paquetes en frontend
	hasNpm := fileExists(filepath.Join(root, "package-lock.json"))
	hasYarn := fileExists(filepath.Join(root, "yarn.lock"))
	hasPnpm := fileExists(filepath.Join(root, "pnpm-lock.yaml"))

	lockCount := 0
	if hasNpm {
		lockCount++
	}
	if hasYarn {
		lockCount++
	}
	if hasPnpm {
		lockCount++
	}

	if lockCount > 1 {
		ambiguities = append(ambiguities, AmbiguityItem{
			Category:    "dependency",
			Severity:    "medium",
			Path:        "package.json",
			Description: "Se detectaron múltiples lockfiles en el proyecto (npm, yarn o pnpm en conflicto)",
			Remediation: "Consolidar las dependencias en un único gestor de paquetes.",
		})
	}

	// Verificar módulos sin componentes detectados
	for _, m := range modules {
		if len(m.Components) == 0 {
			ambiguities = append(ambiguities, AmbiguityItem{
				Category:    "module",
				Severity:    "low",
				Path:        m.Path,
				Description: fmt.Sprintf("El módulo '%s' no contiene archivos de código directos", m.Name),
				Remediation: "Verificar si es un contenedor de subpaquetes o una carpeta vacía.",
			})
		}
	}

	return ambiguities
}

func generateDraftSpec(project, moduleName, description string, components []string) string {
	var sb strings.Builder
	titleModule := strings.Title(moduleName)
	sb.WriteString(fmt.Sprintf("# Especificación Viva: %s\n\n", titleModule))
	sb.WriteString(fmt.Sprintf("> **Dominio:** `%s`\n", slugify(moduleName)))
	sb.WriteString(fmt.Sprintf("> **Proyecto:** %s\n", project))
	sb.WriteString("> **Estado:** Borrador preliminar generado por barrido autónomo\n\n---\n\n")

	sb.WriteString("## Propósito y Alcance\n\n")
	sb.WriteString(fmt.Sprintf("%s. Esta especificación define las reglas funcionales y técnicas del dominio.\n\n", description))

	if len(components) > 0 {
		sb.WriteString("### Componentes Clave Detectados\n\n")
		for _, c := range components {
			sb.WriteString(fmt.Sprintf("- `%s`\n", c))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("### Requirement: Definición Base del Módulo (REQ-01)\n\n")
	sb.WriteString(fmt.Sprintf("El módulo `%s` DEBE encapsular la lógica correspondiente a su dominio funcional.\n\n", moduleName))
	sb.WriteString("#### Scenario: Inicialización canónica del dominio\n")
	sb.WriteString(fmt.Sprintf("- **DADO** el subsistema `%s`\n", moduleName))
	sb.WriteString("- **CUANDO** se invoca su funcionalidad principal\n")
	sb.WriteString("- **ENTONCES** opera conforme a las invariantes de negocio definidas\n")

	return sb.String()
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		} else if r == ' ' || r == '/' || r == '\\' {
			result.WriteRune('-')
		}
	}
	return strings.Trim(result.String(), "-")
}
