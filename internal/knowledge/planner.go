package knowledge

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/semantic"
)

// CrawlPlanOptions define los parámetros para la planificación determinista del crawl.
type CrawlPlanOptions struct {
	WorkspaceRoot string
	SpecsRoot     string
	JobPath       string
	Force         bool
}

var sourceExtensions = map[string]bool{
	".go":   true,
	".ts":   true,
	".tsx":  true,
	".js":   true,
	".jsx":  true,
	".py":   true,
	".rs":   true,
	".java": true,
	".cs":   true,
}

// GenerateCrawlPlan analiza el AST y árbol de directorios para generar un plan de unidades discretas.
func GenerateCrawlPlan(ctx context.Context, opts CrawlPlanOptions) (*CrawlJob, error) {
	if opts.WorkspaceRoot == "" {
		opts.WorkspaceRoot = "."
	}
	absWorkspace, err := filepath.Abs(opts.WorkspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("ruta de workspace inválida: %w", err)
	}

	if opts.SpecsRoot == "" {
		opts.SpecsRoot = filepath.Join(absWorkspace, "openspec")
	}

	jobPath := opts.JobPath
	if jobPath == "" {
		jobPath = DefaultJobPath(absWorkspace)
	}

	// Si ya existe un job y no se fuerza uno nuevo, devolver el existente para reanudación
	if !opts.Force {
		if existing, err := LoadJob(jobPath); err == nil && existing != nil {
			return existing, nil
		}
	}

	units, err := discoverCrawlUnits(absWorkspace, opts.SpecsRoot)
	if err != nil {
		return nil, fmt.Errorf("error descubriendo unidades de crawl: %w", err)
	}

	jobID := fmt.Sprintf("crawl-%d", time.Now().Unix())
	job := NewCrawlJob(jobID, absWorkspace, opts.SpecsRoot, units)

	if err := SaveJob(job, jobPath); err != nil {
		return nil, fmt.Errorf("error persistiendo manifiesto inicial del job: %w", err)
	}

	return job, nil
}

// discoverCrawlUnits explora directorios candidatos y extrae unidades jerárquicas con AST.
func discoverCrawlUnits(root, specsRoot string) ([]CrawlUnit, error) {
	var units []CrawlUnit
	semanticEngine := semantic.NewEngine()

	// Candidatos base para módulos
	candidates := []string{root}
	for _, sub := range []string{"internal", "pkg", "src", "services", "cmd", "packages", "apps"} {
		subPath := filepath.Join(root, sub)
		if info, err := os.Stat(subPath); err == nil && info.IsDir() {
			candidates = append(candidates, subPath)
		}
	}

	seenDirs := make(map[string]bool)

	for _, baseDir := range candidates {
		entries, err := os.ReadDir(baseDir)
		if err != nil {
			continue
		}

		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			dirName := e.Name()
			if ignoredDirs[dirName] || strings.HasPrefix(dirName, ".") {
				continue
			}

			moduleDir := filepath.Join(baseDir, dirName)
			if seenDirs[moduleDir] {
				continue
			}
			seenDirs[moduleDir] = true

			// Inspeccionar si este módulo tiene submódulos (subdirectorios con código)
			subDirs, _ := os.ReadDir(moduleDir)
			hasSubmodules := false

			for _, sub := range subDirs {
				if sub.IsDir() && !ignoredDirs[sub.Name()] && !strings.HasPrefix(sub.Name(), ".") {
					subPath := filepath.Join(moduleDir, sub.Name())
					if dirContainsSourceFiles(subPath) {
						hasSubmodules = true
						unit := buildUnitFromDir(root, specsRoot, dirName, sub.Name(), subPath, semanticEngine)
						if len(unit.Files) > 0 {
							units = append(units, unit)
						}
					}
				}
			}

			// Si no tiene submódulos o contiene archivos fuente en la raíz del módulo
			rootSourceFiles := listSourceFilesInDir(moduleDir)
			if !hasSubmodules && len(rootSourceFiles) > 0 {
				unit := buildUnitFromDir(root, specsRoot, dirName, "core", moduleDir, semanticEngine)
				if len(unit.Files) > 0 {
					units = append(units, unit)
				}
			} else if hasSubmodules && len(rootSourceFiles) > 0 {
				// Módulo tiene submódulos pero además código en su raíz -> unidad raíz/coordinadora
				unit := buildUnitFromDir(root, specsRoot, dirName, "root", moduleDir, semanticEngine)
				// Filtrar solo archivos directos de moduleDir
				var directFiles []string
				for _, f := range unit.Files {
					if filepath.Dir(filepath.Join(root, f)) == moduleDir {
						directFiles = append(directFiles, f)
					}
				}
				unit.Files = directFiles
				if len(unit.Files) > 0 {
					units = append(units, unit)
				}
			}
		}
	}

	// Ordenar unidades por ID para determinismo absoluto
	sort.Slice(units, func(i, j int) bool {
		return units[i].ID < units[j].ID
	})

	return units, nil
}

func buildUnitFromDir(root, specsRoot, domain, submodule, dir string, engine *semantic.Engine) CrawlUnit {
	domainSlug := slugify(domain)
	submoduleSlug := slugify(submodule)
	unitID := fmt.Sprintf("%s/%s", domainSlug, submoduleSlug)
	if submoduleSlug == "core" {
		unitID = domainSlug
	}

	relPath, _ := filepath.Rel(root, dir)
	relSlash := filepath.ToSlash(relPath)

	files := listSourceFilesInDir(dir)
	var relFiles []string
	for _, f := range files {
		rf, _ := filepath.Rel(root, f)
		relFiles = append(relFiles, filepath.ToSlash(rf))
	}

	var exposedInterfaces []string
	var dependencies []string

	// 1. Extraer AST semántico si hay archivos Go
	hasGo := false
	for _, f := range files {
		if strings.HasSuffix(f, ".go") {
			hasGo = true
			break
		}
	}

	if hasGo && engine != nil {
		if res, err := engine.AnalyzeDirectory(dir, domain); err == nil && res != nil {
			depMap := make(map[string]bool)
			for _, dep := range res.Dependencies {
				if dep.TargetPackage != "" && !depMap[dep.TargetPackage] {
					depMap[dep.TargetPackage] = true
					dependencies = append(dependencies, dep.TargetPackage)
				}
			}

			for _, sym := range res.Symbols {
				if isExportedSymbol(sym.Name) {
					exposedInterfaces = append(exposedInterfaces, fmt.Sprintf("%s %s (%s)", sym.Kind, sym.Name, sym.Signature))
				}
			}
		}
	} else {
		// 2. Extracción genérica basada en patrones (TypeScript, Python, etc.)
		dependencies, exposedInterfaces = scanGenericExportsAndImports(files)
	}

	sort.Strings(dependencies)
	sort.Strings(exposedInterfaces)

	specPath := filepath.Join(specsRoot, "specs", domainSlug, "spec.md")
	relSpec, _ := filepath.Rel(root, specPath)
	if relSpec == "" {
		relSpec = specPath
	}

	return CrawlUnit{
		ID:                unitID,
		Domain:            domainSlug,
		Submodule:         submoduleSlug,
		Path:              relSlash,
		Files:             relFiles,
		Dependencies:      dependencies,
		ExposedInterfaces: exposedInterfaces,
		Status:            UnitStatusPending,
		SpecPath:          filepath.ToSlash(relSpec),
	}
}

func isExportedSymbol(name string) bool {
	if len(name) == 0 {
		return false
	}
	first := rune(name[0])
	return first >= 'A' && first <= 'Z'
}

func dirContainsSourceFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() {
			ext := filepath.Ext(e.Name())
			if sourceExtensions[ext] {
				return true
			}
		}
	}
	return false
}

func listSourceFilesInDir(dir string) []string {
	var files []string
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != dir && (ignoredDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(d.Name())
		if sourceExtensions[ext] && !strings.HasSuffix(d.Name(), "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	return files
}

var (
	importRegexTS = regexp.MustCompile(`(?:import|from)\s+['"]([^'"]+)['"]`)
	exportRegexTS = regexp.MustCompile(`export\s+(?:default\s+)?(?:class|function|interface|type|const)\s+([A-Za-z0-9_]+)`)
)

func scanGenericExportsAndImports(files []string) ([]string, []string) {
	depMap := make(map[string]bool)
	expMap := make(map[string]bool)

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if match := importRegexTS.FindStringSubmatch(line); len(match) > 1 {
				depMap[match[1]] = true
			}
			if match := exportRegexTS.FindStringSubmatch(line); len(match) > 1 {
				expMap[match[1]] = true
			}
		}
		_ = f.Close()
	}

	var deps []string
	for d := range depMap {
		deps = append(deps, d)
	}

	var exps []string
	for e := range expMap {
		exps = append(exps, fmt.Sprintf("export %s", e))
	}

	return deps, exps
}
