package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseConfig deserializa y valida la estructura sintáctica de axiom.yaml desde
// bytes: la configuración base y, si existen, los bloques `baseline:` y
// `technical_context:`. Si la base o alguna declaración es inválida devuelve un
// error que acumula todas las violaciones (REQ-24.17).
func ParseConfig(data []byte) (*WorkspaceConfig, error) {
	cfg, baseErr, declarations := parseLayered(data)
	if err := joinLoadErrors(baseErr, declarations); err != nil {
		return nil, err
	}
	return cfg, nil
}

// joinLoadErrors reúne en un solo error el de la configuración base y los de las
// declaraciones. Con un único error lo devuelve tal cual, para conservar los
// mensajes de la configuración base de siempre.
func joinLoadErrors(baseErr error, declarations Declarations) error {
	var errs []error
	for _, err := range []error{baseErr, declarations.BaselineErr, declarations.TechnicalContextErr} {
		if err != nil {
			errs = append(errs, err)
		}
	}
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return errors.Join(errs...)
	}
}

// parseLayered es la carga por capas de axiom.yaml (D-03). Devuelve la
// configuración base (nil si es inválida), su error y las declaraciones, cada una
// con su propio canal de error: una declaración inválida no impide leer la base
// y una base inválida no oculta los errores de las declaraciones.
//
// ParseErr solo es no nulo cuando el contenido no se puede leer como YAML. Sin
// fichero, las declaraciones conservan su valor cero, que equivale a «sin
// declaraciones»: un proyecto sin axiom.yaml nunca bloquea el archivado (R1).
// La base se decodifica desde un yaml.Node, con las claves `baseline` y
// `technical_context` fuera de la decodificación base; el resto de claves
// desconocidas se sigue ignorando.
//
// El orden del tipo de retorno (configuración, error de la base, declaraciones)
// es el que fija el diseño (§4.1, D-03).
func parseLayered(data []byte) (*WorkspaceConfig, error, Declarations) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, fmt.Errorf("el archivo de configuración está vacío"), Declarations{}
	}

	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		parseErr := fmt.Errorf("error al parsear axiom.yaml: %w", err)
		return nil, parseErr, Declarations{ParseErr: parseErr}
	}

	var cfg WorkspaceConfig
	var baseErr error
	// Un documento que solo contiene comentarios no tiene nodo que decodificar:
	// equivale a una configuración vacía, como cuando se decodificaba directo.
	if document.Kind != 0 {
		if err := document.Decode(&cfg); err != nil {
			baseErr = fmt.Errorf("error al parsear axiom.yaml: %w", err)
		}
	}
	// Las declaraciones se evalúan aunque la base falle: su validación solo
	// necesita la topología y specs_repository, tal como se hayan leído.
	declarations := readDeclarationBlocks(&document, lexicalDisplacement(&cfg))

	if baseErr == nil {
		baseErr = validateBaseConfig(&cfg)
	}
	if baseErr != nil {
		return nil, baseErr, declarations
	}
	if declarations.BaselineErr == nil {
		cfg.Baseline = declarations.Baseline
	}
	if declarations.TechnicalContextErr == nil {
		cfg.TechnicalContext = declarations.TechnicalContext
	}
	return &cfg, nil, declarations
}

// ReadDeclarations lee el axiom.yaml de configPath con la misma carga por capas
// que LoadConfig, Locate y workspace validate (parseLayered) y devuelve solo las
// declaraciones, cada una con su canal de error. Lo usa axiom init sobre el
// axiom.yaml propio del destino, donde no puede llamar a Locate (D-06).
//
// Sin fichero devuelve el valor cero, que equivale a «sin declaraciones». Un
// fichero que existe y no se puede leer, o que no es YAML, deja la causa en
// ParseErr, como hace Locate con un axiom.yaml directo.
func ReadDeclarations(configPath string) Declarations {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Declarations{}
		}
		return Declarations{ParseErr: fmt.Errorf("no se pudo leer el archivo '%s': %w", configPath, err)}
	}
	_, _, declarations := parseLayered(data)
	return declarations
}

// validateBaseConfig aplica las validaciones estructurales de la configuración
// base (nombre, topología y roles), con sus mensajes y su orden de siempre.
func validateBaseConfig(cfg *WorkspaceConfig) error {
	if strings.TrimSpace(cfg.Workspace.Name) == "" {
		return fmt.Errorf("el campo 'workspace.name' es obligatorio y no puede estar vacío")
	}

	switch cfg.Workspace.Topology {
	case TopologyMonorepoEmbedded, TopologyMonorepoDecoupled, TopologyMultirepo:
		// Topología válida
	default:
		return fmt.Errorf("topología desconocida o ausente '%s'. Debe ser: %s, %s o %s",
			cfg.Workspace.Topology, TopologyMonorepoEmbedded, TopologyMonorepoDecoupled, TopologyMultirepo)
	}

	if len(cfg.Roles) == 0 {
		return fmt.Errorf("debe definirse al menos un rol de desarrollo en la sección 'roles'")
	}

	for roleKey, role := range cfg.Roles {
		if strings.TrimSpace(role.Name) == "" {
			return fmt.Errorf("el rol '%s' debe tener un nombre descriptivo en 'name'", roleKey)
		}
		if len(role.Repositories) == 0 {
			return fmt.Errorf("el rol '%s' debe contener al menos un repositorio en 'repositories'", roleKey)
		}
		for i, repo := range role.Repositories {
			if strings.TrimSpace(repo.Path) == "" {
				return fmt.Errorf("el repositorio #%d en el rol '%s' no puede tener una ruta vacía", i+1, roleKey)
			}
		}
	}
	return nil
}

const (
	// DefaultConfigFilename es el nombre canónico del archivo de configuración de Axiom.
	DefaultConfigFilename = "axiom.yaml"

	// WorkspacePointerFilename es el nombre del archivo puntero opcional en la raíz del workspace.
	WorkspacePointerFilename = ".axiom-workspace"
)

// WorkspacePointer modela el contenido del archivo .axiom-workspace para el Patrón Canónico.
type WorkspacePointer struct {
	Config string `yaml:"config,omitempty"` // Ruta al archivo axiom.yaml (ej: "repo-specs/axiom.yaml")
	Specs  string `yaml:"specs,omitempty"`  // Ruta a la carpeta de especificaciones (ej: "repo-specs")
}

// ParsePointer deserializa el archivo puntero .axiom-workspace.
func ParsePointer(data []byte) (*WorkspacePointer, error) {
	var ptr WorkspacePointer
	if err := yaml.Unmarshal(data, &ptr); err != nil {
		return nil, fmt.Errorf("error al parsear %s: %w", WorkspacePointerFilename, err)
	}
	return &ptr, nil
}

// ResolveConfigFile resuelve la ruta efectiva al archivo axiom.yaml aplicando la cascada canónica:
// 1. Puntero .axiom-workspace en baseDir.
// 2. Archivo axiom.yaml directo en baseDir.
// 3. Fallback de auto-descubrimiento en subdirectorios típicos (repo-specs, specs, openspec, especificacion).
func ResolveConfigFile(target string) (string, error) {
	cleanTarget := filepath.Clean(target)
	info, err := os.Stat(cleanTarget)

	// Si target es un fichero regular existente:
	if err == nil && info.Mode().IsRegular() {
		if filepath.Base(cleanTarget) == WorkspacePointerFilename {
			return resolveFromPointerFile(cleanTarget)
		}
		return cleanTarget, nil
	}

	// Determinar el directorio base para buscar
	baseDir := cleanTarget
	if err != nil || !info.IsDir() {
		baseName := filepath.Base(cleanTarget)
		if baseName != DefaultConfigFilename && baseName != WorkspacePointerFilename && strings.Contains(baseName, ".") {
			return "", fmt.Errorf("archivo de configuración no encontrado en '%s'", cleanTarget)
		}
		baseDir = filepath.Dir(cleanTarget)
	}

	// 1. Comprobar si existe .axiom-workspace en baseDir
	pointerPath := filepath.Join(baseDir, WorkspacePointerFilename)
	if fileExists(pointerPath) {
		return resolveFromPointerFile(pointerPath)
	}

	// 2. Comprobar si existe axiom.yaml en baseDir
	directPath := filepath.Join(baseDir, DefaultConfigFilename)
	if fileExists(directPath) {
		return directPath, nil
	}

	// 3. Fallback de auto-descubrimiento en subdirectorios canónicos
	candidates := []string{"repo-specs", "specs", "openspec", ".openspec", "especificacion"}
	for _, c := range candidates {
		subPath := filepath.Join(baseDir, c, DefaultConfigFilename)
		if fileExists(subPath) {
			return subPath, nil
		}
	}

	return "", fmt.Errorf("archivo de configuración no encontrado en '%s' (ni %s ni %s)", baseDir, WorkspacePointerFilename, DefaultConfigFilename)
}

func resolveFromPointerFile(pointerPath string) (string, error) {
	data, err := os.ReadFile(pointerPath)
	if err != nil {
		return "", fmt.Errorf("no se pudo leer el archivo puntero '%s': %w", pointerPath, err)
	}

	ptr, err := ParsePointer(data)
	if err != nil {
		return "", err
	}

	baseDir := filepath.Dir(pointerPath)
	if strings.TrimSpace(ptr.Config) != "" {
		targetPath := strings.TrimSpace(ptr.Config)
		if !filepath.IsAbs(targetPath) {
			targetPath = filepath.Join(baseDir, targetPath)
		}
		if !fileExists(targetPath) {
			return "", fmt.Errorf("el puntero '%s' apunta a un archivo inexistente: '%s'", pointerPath, targetPath)
		}
		return filepath.Clean(targetPath), nil
	}

	if strings.TrimSpace(ptr.Specs) != "" {
		specsDir := strings.TrimSpace(ptr.Specs)
		targetPath := filepath.Join(baseDir, specsDir, DefaultConfigFilename)
		if !fileExists(targetPath) {
			return "", fmt.Errorf("el puntero '%s' indica repositorio de specs '%s', pero no contiene '%s'", pointerPath, specsDir, DefaultConfigFilename)
		}
		return filepath.Clean(targetPath), nil
	}

	return "", fmt.Errorf("el puntero '%s' debe definir 'config' o 'specs'", pointerPath)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// LoadConfig carga el archivo axiom.yaml resolviendo punteros (.axiom-workspace) o rutas directas.
func LoadConfig(filePath string) (*WorkspaceConfig, error) {
	resolvedPath, err := ResolveConfigFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("archivo de configuración no encontrado en '%s'", filePath)
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("archivo de configuración no encontrado en '%s'", filePath)
		}
		return nil, fmt.Errorf("no se pudo leer el archivo '%s': %w", resolvedPath, err)
	}

	return ParseConfig(data)
}
