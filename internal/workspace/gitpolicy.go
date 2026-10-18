package workspace

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
)

const (
	// DefaultIntegrationBranch es la rama de integración cuando axiom.yaml no
	// declara `git.integration_branch` (REQ-25.26).
	DefaultIntegrationBranch = "main"
	// DefaultBranchPrefix es el prefijo de las ramas de cambio cuando axiom.yaml no
	// declara `git.branch_prefix` (REQ-25.26).
	DefaultBranchPrefix = "feat"
)

// GitConfig es el bloque `git:` de axiom.yaml (REQ-25.26): la política de ramas
// del proyecto. Es política versionada y compartida por el equipo; el modo de
// trabajo y la raíz de los worktrees son preferencias personales y no se
// declaran aquí.
//
// En Declarations.Git cada campo conserva lo escrito en el fichero (cadena vacía
// si la clave no se declaró). En WorkspaceConfig.Git los dos campos están
// siempre rellenos: lo no declarado toma su valor por defecto.
type GitConfig struct {
	IntegrationBranch string `yaml:"integration_branch,omitempty"`
	BranchPrefix      string `yaml:"branch_prefix,omitempty"`
}

// gitKeys es la lista cerrada de claves del bloque `git:` (D-03).
var gitKeys = []string{"integration_branch", "branch_prefix"}

// resolveGit devuelve la política de ramas efectiva a partir de lo declarado, que
// puede ser nil (sin bloque): completa con los valores por defecto cada clave que
// falte. Nunca devuelve nil ni modifica declared.
func resolveGit(declared *GitConfig) *GitConfig {
	resolved := GitConfig{IntegrationBranch: DefaultIntegrationBranch, BranchPrefix: DefaultBranchPrefix}
	if declared != nil {
		if declared.IntegrationBranch != "" {
			resolved.IntegrationBranch = declared.IntegrationBranch
		}
		if declared.BranchPrefix != "" {
			resolved.BranchPrefix = declared.BranchPrefix
		}
	}
	return &resolved
}

// decodeGit decodifica y valida el bloque `git:` recorriendo su yaml.Node contra
// la lista cerrada de claves (yaml.v3 no ofrece KnownFields en Node.Decode), con
// el mismo patrón que `baseline:` (declarations.go). Acumula todos los errores:
// claves desconocidas o repetidas y valores que no son nombres de referencia git
// válidos (gitref.ValidRefName y gitref.ValidPrefix). Cada error nombra la clave
// y, si procede, el valor y la regla incumplida.
//
// Un bloque que no es un mapeo, incluido el nulo (`git:` sin contenido), se
// rechaza: para usar los valores por defecto se elimina el bloque. Cualquier
// valor escalar se valida por su texto, de modo que `branch_prefix: 2024` es el
// prefijo «2024»; un valor nulo, una lista o un mapeo no son cadenas.
func decodeGit(node *yaml.Node) (*GitConfig, error) {
	node = resolveAlias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil, errors.New("el bloque 'git' debe ser un mapeo con las claves opcionales 'integration_branch' y 'branch_prefix'; para usar los valores por defecto, elimina el bloque")
	}

	var errs []error
	errs = append(errs, unknownKeys(node, "git", gitKeys)...)

	var declared GitConfig
	seen := map[string]bool{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		if !containsString(gitKeys, key.Value) {
			continue // ya reportada por unknownKeys
		}
		if seen[key.Value] {
			errs = append(errs, fmt.Errorf("clave duplicada '%s' en 'git' (línea %d)", key.Value, key.Line))
			continue
		}
		seen[key.Value] = true

		isPrefix := key.Value == "branch_prefix"
		validate := gitref.ValidRefName
		if isPrefix {
			validate = gitref.ValidPrefix
		}
		value, err := gitValue("git."+key.Value, node.Content[i+1], validate)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if isPrefix {
			declared.BranchPrefix = value
		} else {
			declared.IntegrationBranch = value
		}
	}

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return &declared, nil
}

// gitValue lee el valor de una clave de `git:` y lo valida con validate. label
// es la clave completa (`git.branch_prefix`) que nombran los errores.
func gitValue(label string, node *yaml.Node, validate func(string) error) (string, error) {
	node = resolveAlias(node)
	switch {
	case isNull(node):
		return "", fmt.Errorf("%s no puede ser un valor nulo: debe ser una cadena no vacía", label)
	case node.Kind == yaml.SequenceNode:
		return "", fmt.Errorf("%s debe ser una cadena, no una lista", label)
	case node.Kind != yaml.ScalarNode:
		return "", fmt.Errorf("%s debe ser una cadena, no un mapeo", label)
	}
	if err := validate(node.Value); err != nil {
		return "", fmt.Errorf("%s: el valor '%s' no es válido: %w", label, node.Value, err)
	}
	return node.Value, nil
}
