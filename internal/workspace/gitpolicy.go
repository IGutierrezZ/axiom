package workspace

import (
	"errors"
	"fmt"
	"strings"

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

// RepositoryPolicy es una entrada del bloque `repositories:` de axiom.yaml
// (REQ-25.27): los ajustes de rama de un repositorio, que sustituyen a los del
// bloque `git:` solo en lo que la entrada declara.
//
// Path es la ruta tal como se escribió, relativa a la carpeta maestra. Los otros
// dos campos conservan lo escrito y quedan vacíos si la clave no se declaró: un
// valor vacío significa «hereda el del bloque `git:`». La herencia no se aplica
// aquí; la resuelve quien consume la política de ramas.
type RepositoryPolicy struct {
	Path              string `yaml:"path"`
	IntegrationBranch string `yaml:"integration_branch,omitempty"`
	BranchPrefix      string `yaml:"branch_prefix,omitempty"`
}

// Listas cerradas de claves de los bloques de política de ramas (D-03).
var (
	gitKeys        = []string{"integration_branch", "branch_prefix"}
	repositoryKeys = []string{"path", "integration_branch", "branch_prefix"}
)

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

	values, errs := decodeStringKeys(node, "git", gitKeys, branchValidator)
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return &GitConfig{IntegrationBranch: values["integration_branch"], BranchPrefix: values["branch_prefix"]}, nil
}

// decodeRepositories decodifica y valida el bloque `repositories:` con el mismo
// patrón que decodeGit (REQ-25.27): una lista de mapeos, cada uno recorrido contra
// la lista cerrada de claves `path`, `integration_branch` y `branch_prefix`, con
// los errores acumulados. Cada error nombra la entrada (`repositories[0]`), la
// clave y, si procede, el valor y la regla incumplida.
//
// Un bloque que no es una lista, incluido el nulo, se rechaza, igual que un
// `git:` que no es un mapeo; una lista vacía es válida y equivale a no declarar
// entradas. `path` es obligatoria, relativa a la carpeta maestra y no puede
// salir de ella (checkRepositoryPath), y dos entradas no pueden declarar la misma
// ruta. Que la ruta exista y cuántas entradas comparten raíz git son reglas del
// sistema de ficheros que aplica `workspace validate` (ValidateRepositoryPolicies).
func decodeRepositories(node *yaml.Node) ([]RepositoryPolicy, error) {
	node = resolveAlias(node)
	if node == nil || node.Kind != yaml.SequenceNode {
		return nil, errors.New("el bloque 'repositories' debe ser una lista de entradas con 'path' y, opcionalmente, 'integration_branch' y 'branch_prefix'; para usar los valores del bloque 'git', elimina el bloque")
	}

	type declaredPath struct {
		index int
		raw   string
	}
	var errs []error
	entries := make([]RepositoryPolicy, 0, len(node.Content))
	firstByPath := map[string]declaredPath{}
	for i, item := range node.Content {
		label := fmt.Sprintf("repositories[%d]", i)
		item = resolveAlias(item)
		if item == nil || item.Kind != yaml.MappingNode {
			errs = append(errs, fmt.Errorf("%s debe ser un mapeo con 'path' y, opcionalmente, 'integration_branch' y 'branch_prefix'", label))
			continue
		}

		values, entryErrs := decodeStringKeys(item, label, repositoryKeys, repositoryValidator)
		errs = append(errs, entryErrs...)
		if _, declared := mappingValue(item, "path"); !declared {
			errs = append(errs, fmt.Errorf("%s no declara 'path'", label))
			continue
		}
		raw, hasPath := values["path"]
		if !hasPath {
			continue // su valor inválido ya está reportado
		}

		// La ruta es válida: decodeStringKeys ya la comprobó. Se compara sin
		// distinguir mayúsculas, como el resto de rutas de axiom.yaml, que se
		// comparte entre Windows y Linux.
		clean, _ := checkRepositoryPath(raw)
		key := strings.ToLower(clean)
		if first, seen := firstByPath[key]; seen {
			errs = append(errs, fmt.Errorf("las entradas repositories[%d] ('%s') y %s ('%s') declaran la misma ruta '%s': declara cada repositorio una sola vez",
				first.index, first.raw, label, raw, clean))
		} else {
			firstByPath[key] = declaredPath{index: i, raw: raw}
		}
		entries = append(entries, RepositoryPolicy{Path: raw, IntegrationBranch: values["integration_branch"], BranchPrefix: values["branch_prefix"]})
	}

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}
	return entries, nil
}

// branchValidator devuelve la validación del valor de una clave de rama:
// `branch_prefix` sigue gitref.ValidPrefix y `integration_branch`,
// gitref.ValidRefName.
func branchValidator(key string) func(string) error {
	if key == "branch_prefix" {
		return gitref.ValidPrefix
	}
	return gitref.ValidRefName
}

// repositoryValidator es branchValidator más la regla de `path`.
func repositoryValidator(key string) func(string) error {
	if key == "path" {
		return func(raw string) error {
			_, err := checkRepositoryPath(raw)
			return err
		}
	}
	return branchValidator(key)
}

// checkRepositoryPath valida la ruta de una entrada de `repositories:` y la
// devuelve normalizada con «/»: relativa a la carpeta maestra y contenida en ella.
// Que exista como directorio lo comprueba workspace validate.
func checkRepositoryPath(raw string) (string, error) {
	clean, err := checkRelativeTo(raw, "la carpeta maestra")
	if err != nil {
		return "", err
	}
	if escapes(clean) {
		return "", errors.New("sale de la carpeta maestra: un repositorio debe permanecer dentro de ella")
	}
	return clean, nil
}

// decodeStringKeys recorre los pares clave-valor de un mapeo contra la lista
// cerrada keys y devuelve el valor de cada clave que se declaró y es válida, junto
// con todos los errores: claves desconocidas, claves repetidas y valores que no
// son una cadena o no superan la validación que devuelve validatorFor. label
// nombra el mapeo en los errores (`git`, `repositories[0]`) y prefija cada clave
// en los de sus valores (`git.branch_prefix`).
func decodeStringKeys(node *yaml.Node, label string, keys []string, validatorFor func(key string) func(string) error) (map[string]string, []error) {
	errs := unknownKeys(node, label, keys)
	values := map[string]string{}
	seen := map[string]bool{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		if !containsString(keys, key.Value) {
			continue // ya reportada por unknownKeys
		}
		if seen[key.Value] {
			errs = append(errs, fmt.Errorf("clave duplicada '%s' en '%s' (línea %d)", key.Value, label, key.Line))
			continue
		}
		seen[key.Value] = true

		value, err := gitValue(label+"."+key.Value, node.Content[i+1], validatorFor(key.Value))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		values[key.Value] = value
	}
	return values, errs
}

// gitValue lee el valor de una clave de `git:` o de una entrada de
// `repositories:` y lo valida con validate. label es la clave completa
// (`git.branch_prefix`) que nombran los errores.
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
