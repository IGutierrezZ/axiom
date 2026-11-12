package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockFS simula un sistema de archivos en memoria para pruebas deterministas
type mockFS struct {
	existingDirs map[string]bool
}

func newMockFS() *mockFS {
	return &mockFS{
		existingDirs: make(map[string]bool),
	}
}

func (m *mockFS) addDir(path string) {
	m.existingDirs[filepath.Clean(path)] = true
}

func (m *mockFS) Stat(name string) (os.FileInfo, error) {
	if m.existingDirs[filepath.Clean(name)] {
		return nil, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFS) IsDir(path string) (bool, error) {
	clean := filepath.Clean(path)
	if m.existingDirs[clean] {
		return true, nil
	}
	return false, os.ErrNotExist
}

func TestValidateTopology(t *testing.T) {
	baseDir := filepath.Clean("/workspace")

	t.Run("monorepo embebido valido en raiz", func(t *testing.T) {
		fs := newMockFS()
		fs.addDir(baseDir)

		cfg := &WorkspaceConfig{
			Workspace: WorkspaceSection{
				Name:            "SingleApp",
				Topology:        TopologyMonorepoEmbedded,
				SpecsRepository: ".",
			},
			Roles: map[string]RoleConfig{
				"fullstack": {
					Name: "Fullstack",
					Repositories: []RepositoryEntry{
						{Path: "."},
					},
				},
			},
		}

		report, err := Validate(fs, cfg, baseDir)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if !report.Valid {
			t.Errorf("se esperaba que el reporte fuera valido, errores: %v", report.Errors)
		}
	})

	t.Run("monorepo embebido con subcarpeta openspec valida", func(t *testing.T) {
		fs := newMockFS()
		fs.addDir(baseDir)
		fs.addDir(filepath.Join(baseDir, "openspec"))

		cfg := &WorkspaceConfig{
			Workspace: WorkspaceSection{
				Name:            "SingleApp",
				Topology:        TopologyMonorepoEmbedded,
				SpecsRepository: "openspec",
			},
			Roles: map[string]RoleConfig{
				"fullstack": {
					Name: "Fullstack",
					Repositories: []RepositoryEntry{
						{Path: "."},
					},
				},
			},
		}

		report, err := Validate(fs, cfg, baseDir)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if !report.Valid {
			t.Errorf("se esperaba valido, errores: %v", report.Errors)
		}
	})

	t.Run("topologia retirada rechazada aunque exista el repositorio de specs", func(t *testing.T) {
		fs := newMockFS()
		fs.addDir(baseDir)
		fs.addDir(filepath.Join(baseDir, "especificacion"))
		fs.addDir(filepath.Join(baseDir, "monorepo-app"))

		// Un valor retirado solo llega aquí en una configuración construida a mano:
		// la carga ya lo rechaza (REQ-25.25), pero Validate tampoco lo acepta.
		cfg := &WorkspaceConfig{
			Workspace: WorkspaceSection{
				Name:            "RetiredProject",
				Topology:        TopologyType("monorepo-decoupled"),
				SpecsRepository: "especificacion",
			},
			Roles: map[string]RoleConfig{
				"team": {
					Name: "Team",
					Repositories: []RepositoryEntry{
						{Path: "monorepo-app"},
					},
				},
			},
		}

		report, err := Validate(fs, cfg, baseDir)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if report.Valid {
			t.Fatalf("se esperaba reporte invalido para la topologia retirada")
		}
		assertContains(t, report.Errors, "la topología `monorepo-decoupled` fue retirada: sustitúyela por `multirepo`")
	})

	t.Run("topologia desconocida rechazada", func(t *testing.T) {
		fs := newMockFS()
		fs.addDir(baseDir)
		cfg := &WorkspaceConfig{
			Workspace: WorkspaceSection{Name: "Otra", Topology: TopologyType("otra"), SpecsRepository: "."},
			Roles:     map[string]RoleConfig{"team": {Name: "Team", Repositories: []RepositoryEntry{{Path: "."}}}},
		}
		report, err := Validate(fs, cfg, baseDir)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if report.Valid {
			t.Fatalf("se esperaba reporte invalido para una topologia desconocida")
		}
		assertContains(t, report.Errors, "topología desconocida o ausente 'otra'")
	})

	t.Run("multirepo valido con roles y repositorios completos", func(t *testing.T) {
		fs := newMockFS()
		fs.addDir(baseDir)
		fs.addDir(filepath.Join(baseDir, "especificacion"))
		fs.addDir(filepath.Join(baseDir, "frontend-app"))
		fs.addDir(filepath.Join(baseDir, "backend-api"))
		fs.addDir(filepath.Join(baseDir, "infra"))

		cfg := &WorkspaceConfig{
			Workspace: WorkspaceSection{
				Name:            "FederatedPlatform",
				Topology:        TopologyMultirepo,
				SpecsRepository: "especificacion",
			},
			Roles: map[string]RoleConfig{
				"frontend": {
					Name: "Frontend",
					Repositories: []RepositoryEntry{
						{Path: "frontend-app"},
					},
				},
				"backend": {
					Name: "Backend",
					Repositories: []RepositoryEntry{
						{Path: "backend-api"},
					},
				},
				"devops": {
					Name: "DevOps",
					Repositories: []RepositoryEntry{
						{Path: "infra"},
					},
				},
			},
		}

		report, err := Validate(fs, cfg, baseDir)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if !report.Valid {
			t.Errorf("se esperaba valido, errores: %v", report.Errors)
		}
	})

	t.Run("multirepo rechazado si no tiene repositorio canonico de specs", func(t *testing.T) {
		fs := newMockFS()
		fs.addDir(baseDir)
		fs.addDir(filepath.Join(baseDir, "frontend-app"))

		cfg := &WorkspaceConfig{
			Workspace: WorkspaceSection{
				Name:            "FederatedPlatform",
				Topology:        TopologyMultirepo,
				SpecsRepository: "", // Vacio
			},
			Roles: map[string]RoleConfig{
				"frontend": {
					Name: "Frontend",
					Repositories: []RepositoryEntry{
						{Path: "frontend-app"},
					},
				},
			},
		}

		report, err := Validate(fs, cfg, baseDir)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if report.Valid {
			t.Errorf("se esperaba rechazo por falta de specs_repository")
		}
		assertContains(t, report.Errors, "requieren obligatoriamente un repositorio canónico de especificaciones")
	})

	t.Run("multirepo rechazado si repositorio de rol no existe", func(t *testing.T) {
		fs := newMockFS()
		fs.addDir(baseDir)
		fs.addDir(filepath.Join(baseDir, "especificacion"))
		// Falta frontend-app

		cfg := &WorkspaceConfig{
			Workspace: WorkspaceSection{
				Name:            "FederatedPlatform",
				Topology:        TopologyMultirepo,
				SpecsRepository: "especificacion",
			},
			Roles: map[string]RoleConfig{
				"frontend": {
					Name: "Frontend",
					Repositories: []RepositoryEntry{
						{Path: "frontend-app"},
					},
				},
			},
		}

		report, err := Validate(fs, cfg, baseDir)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if report.Valid {
			t.Errorf("se esperaba invalido por repo de rol faltante")
		}
		assertContains(t, report.Errors, "el repositorio 'frontend-app' asignado al rol 'frontend' no existe")
	})
}

func TestValidateSkillRoots(t *testing.T) {
	baseDir := filepath.Clean("/workspace")

	newConfig := func(roots ...string) *WorkspaceConfig {
		return &WorkspaceConfig{
			Workspace: WorkspaceSection{
				Name:            "SingleApp",
				Topology:        TopologyMonorepoEmbedded,
				SpecsRepository: ".",
				SkillRoots:      roots,
			},
			Roles: map[string]RoleConfig{
				"fullstack": {Name: "Fullstack", Repositories: []RepositoryEntry{{Path: "."}}},
			},
		}
	}

	t.Run("raices relativas validas no generan errores aunque no existan en disco", func(t *testing.T) {
		fs := newMockFS()
		fs.addDir(baseDir)

		report, err := Validate(fs, newConfig("internal/assets/skills", "docs/skills"), baseDir)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if !report.Valid || len(report.Errors) != 0 {
			t.Errorf("se esperaba valido sin errores, errores: %v", report.Errors)
		}
	})

	t.Run("cada raiz invalida se reporta como error", func(t *testing.T) {
		fs := newMockFS()
		fs.addDir(baseDir)

		report, err := Validate(fs, newConfig("internal/assets/skills", "/opt/skills", "../fuera", ".", ""), baseDir)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if report.Valid {
			t.Fatalf("se esperaba invalido por raices de skills no validas")
		}
		if len(report.Errors) != 4 {
			t.Fatalf("se esperaban 4 errores (uno por raiz invalida), obtenidos %d: %v", len(report.Errors), report.Errors)
		}
		assertContains(t, report.Errors, "la raíz de skills '/opt/skills' declarada en 'workspace.skill_roots' no es válida")
		assertContains(t, report.Errors, "la raíz de skills '../fuera'")
		assertContains(t, report.Errors, "la raíz de skills '.'")
		assertContains(t, report.Errors, "la raíz de skills ''")
		for _, e := range report.Errors {
			if strings.Contains(e, "internal/assets/skills") {
				t.Errorf("la raiz valida no debe reportarse: %s", e)
			}
		}
	})
}

func assertContains(t *testing.T, list []string, substr string) {
	t.Helper()
	for _, item := range list {
		if strings.Contains(item, substr) {
			return
		}
	}
	t.Errorf("se esperaba encontrar la subcadena '%s' en la lista: %v", substr, list)
}

// declarationsConfig carga una configuración válida con los bloques dados.
func declarationsConfig(t *testing.T, config, blocks string) *WorkspaceConfig {
	t.Helper()
	cfg, err := ParseConfig([]byte(config + blocks))
	if err != nil {
		t.Fatalf("la configuración de la prueba debe ser válida: %v", err)
	}
	return cfg
}

// linkDirOrSkip enlaza el directorio link con target o omite la prueba si el
// sistema no permite crear enlaces simbólicos (en Windows exige un privilegio o
// el modo de desarrollador). No recurre a una unión de directorio: desde Go 1.23
// filepath.EvalSymlinks no evalúa los puntos de montaje de Windows, así que una
// unión no ejercitaría la regla que estas pruebas comprueban.
func linkDirOrSkip(t *testing.T, target, link string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("no se pueden crear enlaces simbólicos en este sistema: %v", err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Sin declaraciones, o sin configuración, no hay nada que comprobar.
func TestValidateDeclarationPathsWithoutDeclarations(t *testing.T) {
	errs, warnings := ValidateDeclarationPaths(declarationsConfig(t, locateEmbeddedConfig, ""), t.TempDir())
	if errs != nil || warnings != nil {
		t.Fatalf("errores=%v advertencias=%v", errs, warnings)
	}
	if errs, warnings = ValidateDeclarationPaths(nil, t.TempDir()); errs != nil || warnings != nil {
		t.Fatalf("una configuración nula no produce nada: %v %v", errs, warnings)
	}
}

// REQ-24.27: una ruta de contexto inexistente solo avisa.
func TestValidateDeclarationPathsContextPaths(t *testing.T) {
	const block = "technical_context:\n  maintained:\n    entry: context/TECHNICAL_CONTEXT.md\n    paths: [context/]\n  reference:\n    paths: [00_Glosario.md]\n"

	t.Run("rutas inexistentes avisan sin error", func(t *testing.T) {
		errs, warnings := ValidateDeclarationPaths(declarationsConfig(t, locateEmbeddedConfig, block), t.TempDir())
		if len(errs) != 0 {
			t.Fatalf("una ruta inexistente no es un error: %v", errs)
		}
		if len(warnings) != 3 {
			t.Fatalf("advertencias = %v, se esperaban 3 (dos rutas y la entrada)", warnings)
		}
		assertContains(t, warnings, "'context/' de 'technical_context.maintained.paths' no existe todavía")
		assertContains(t, warnings, "'context/TECHNICAL_CONTEXT.md' de 'technical_context.maintained.entry'")
		assertContains(t, warnings, "'00_Glosario.md' de 'technical_context.reference.paths'")
	})

	t.Run("rutas existentes no producen nada", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, filepath.Join(root, "context", "TECHNICAL_CONTEXT.md"), "# ctx\n")
		writeTestFile(t, filepath.Join(root, "00_Glosario.md"), "# glosario\n")
		errs, warnings := ValidateDeclarationPaths(declarationsConfig(t, locateEmbeddedConfig, block), root)
		if len(errs) != 0 || len(warnings) != 0 {
			t.Fatalf("errores=%v advertencias=%v", errs, warnings)
		}
	})

	t.Run("multirepo admite una ruta mantenida en el repositorio de codigo hermano", func(t *testing.T) {
		master := t.TempDir()
		writeTestFile(t, filepath.Join(master, "repo-backend", "docs", "arquitectura.md"), "# arq\n")
		cfg := declarationsConfig(t, locateCanonicalConfig, "technical_context:\n  maintained:\n    paths: [../repo-backend/docs/]\n")
		if err := os.MkdirAll(filepath.Join(master, "repo-specs"), 0o755); err != nil {
			t.Fatal(err)
		}
		errs, warnings := ValidateDeclarationPaths(cfg, master)
		if len(errs) != 0 || len(warnings) != 0 {
			t.Fatalf("errores=%v advertencias=%v", errs, warnings)
		}
	})
}

// REQ-24.27: un enlace simbólico que sale del límite declarado es un error.
func TestValidateDeclarationPathsSymlinks(t *testing.T) {
	t.Run("ruta de contexto que resuelve fuera de la carpeta maestra", func(t *testing.T) {
		outside := t.TempDir()
		writeTestFile(t, filepath.Join(outside, "ajeno.md"), "# ajeno\n")
		root := t.TempDir()
		linkDirOrSkip(t, outside, filepath.Join(root, "context"))
		cfg := declarationsConfig(t, locateEmbeddedConfig, "technical_context:\n  reference:\n    paths: [context/]\n")
		errs, _ := ValidateDeclarationPaths(cfg, root)
		if len(errs) != 1 {
			t.Fatalf("errores = %v, se esperaba 1", errs)
		}
		assertContains(t, errs, "la ruta 'context/' de 'technical_context.reference.paths' resuelve a")
		assertContains(t, errs, "fuera de la carpeta maestra")
	})

	t.Run("entrada que resuelve fuera de la carpeta maestra", func(t *testing.T) {
		outside := t.TempDir()
		writeTestFile(t, filepath.Join(outside, "ajeno.md"), "# ajeno\n")
		root := t.TempDir()
		writeTestFile(t, filepath.Join(root, "context", "otro.md"), "# otro\n")
		writeTestFile(t, filepath.Join(outside, "ENTRADA.md"), "# entrada\n")
		// context/externo/ es un enlace a un directorio ajeno que contiene la entrada.
		linkDirOrSkip(t, outside, filepath.Join(root, "context", "externo"))
		cfg := declarationsConfig(t, locateEmbeddedConfig, "technical_context:\n  maintained:\n    entry: context/externo/ENTRADA.md\n    paths: [context/otro.md, context/externo/]\n")
		errs, _ := ValidateDeclarationPaths(cfg, root)
		assertContains(t, errs, "'context/externo/ENTRADA.md' de 'technical_context.maintained.entry' resuelve a")
	})

	t.Run("solape por identidad entre grafias distintas del mismo directorio", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, filepath.Join(root, "context", "a.md"), "# a\n")
		linkDirOrSkip(t, filepath.Join(root, "context"), filepath.Join(root, "alias"))
		cfg := declarationsConfig(t, locateEmbeddedConfig, "technical_context:\n  maintained:\n    paths: [context/]\n  reference:\n    paths: [alias/]\n")
		errs, _ := ValidateDeclarationPaths(cfg, root)
		if len(errs) != 1 {
			t.Fatalf("errores = %v, se esperaba 1", errs)
		}
		assertContains(t, errs, "'context/' (technical_context.maintained) y la ruta 'alias/' (technical_context.reference) solapan por identidad")
	})

	t.Run("rutas distintas no solapan", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, filepath.Join(root, "context", "a.md"), "# a\n")
		writeTestFile(t, filepath.Join(root, "context-extra", "b.md"), "# b\n")
		cfg := declarationsConfig(t, locateEmbeddedConfig, "technical_context:\n  maintained:\n    paths: [context]\n  reference:\n    paths: [context-extra]\n")
		if errs, _ := ValidateDeclarationPaths(cfg, root); len(errs) != 0 {
			t.Fatalf("errores = %v, se esperaba ninguno", errs)
		}
	})
}

// REQ-24.62: el contexto mantenido exige un almacén de ficheros.
func TestValidateDeclarationPathsArtifactStore(t *testing.T) {
	const both = "technical_context:\n  maintained:\n    paths: [context/]\n"
	const referenceOnly = "technical_context:\n  reference:\n    paths: [00_Glosario.md]\n"

	setup := func(t *testing.T, configFile, content string) string {
		t.Helper()
		root := t.TempDir()
		writeTestFile(t, filepath.Join(root, "03_Modelo_Datos.md"), "# modelo\n")
		writeTestFile(t, filepath.Join(root, "context", "a.md"), "# ctx\n")
		writeTestFile(t, filepath.Join(root, "00_Glosario.md"), "# glosario\n")
		if content != "" {
			writeTestFile(t, filepath.Join(root, "openspec", configFile), content)
		}
		return root
	}

	for _, store := range []string{"engram", "none"} {
		t.Run("almacen "+store+" con contexto mantenido", func(t *testing.T) {
			root := setup(t, "config.yaml", "artifact_store: "+store+"\n")
			errs, _ := ValidateDeclarationPaths(declarationsConfig(t, locateEmbeddedConfig, both), root)
			if len(errs) != 1 {
				t.Fatalf("errores = %v, se esperaba uno", errs)
			}
			assertContains(t, errs, "la declaración 'technical_context.maintained' exige un almacén de artefactos de ficheros")
			assertContains(t, errs, "declara artifact_store '"+store+"'")
			assertContains(t, errs, "Declara 'artifact_store: openspec' o 'hybrid' en openspec/config.yaml, o retira la declaración 'technical_context.maintained'")
		})
	}

	t.Run("solo contexto mantenido con none", func(t *testing.T) {
		root := setup(t, "config.yaml", "artifact_store: none\n")
		errs, _ := ValidateDeclarationPaths(declarationsConfig(t, locateEmbeddedConfig, "technical_context:\n  maintained:\n    paths: [context/]\n"), root)
		if len(errs) != 1 {
			t.Fatalf("errores = %v", errs)
		}
		assertContains(t, errs, "'technical_context.maintained'")
		assertContains(t, errs, "'none'")
	})

	t.Run("config.yml tambien se lee", func(t *testing.T) {
		root := setup(t, "config.yml", "artifactStore: \"Engram\"  # almacén\n")
		errs, _ := ValidateDeclarationPaths(declarationsConfig(t, locateEmbeddedConfig, both), root)
		if len(errs) != 1 {
			t.Fatalf("errores = %v", errs)
		}
	})

	for _, tc := range []struct {
		name    string
		content string
		blocks  string
	}{
		{"almacen openspec", "artifact_store: openspec\n", both},
		{"almacen hybrid", "artifact_store: hybrid\n", both},
		{"sin artifact_store declarado", "rules:\n  apply: []\n", both},
		{"sin openspec/config.yaml", "", both},
		{"solo contexto de referencia con engram", "artifact_store: engram\n", referenceOnly},
		{"sin declaraciones con none", "artifact_store: none\n", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := setup(t, "config.yaml", tc.content)
			errs, _ := ValidateDeclarationPaths(declarationsConfig(t, locateEmbeddedConfig, tc.blocks), root)
			if len(errs) != 0 {
				t.Fatalf("errores = %v, no se esperaba ninguno por el almacén", errs)
			}
		})
	}

	t.Run("multirepo lee el almacen del repositorio de especificaciones", func(t *testing.T) {
		master := t.TempDir()
		writeTestFile(t, filepath.Join(master, "repo-specs", "context", "a.md"), "# ctx\n")
		writeTestFile(t, filepath.Join(master, "repo-specs", "openspec", "config.yaml"), "artifact_store: none\n")
		cfg := declarationsConfig(t, locateCanonicalConfig, both)
		errs, _ := ValidateDeclarationPaths(cfg, master)
		if len(errs) != 1 {
			t.Fatalf("errores = %v", errs)
		}
		assertContains(t, errs, "la declaración 'technical_context.maintained' exige")
	})
}
