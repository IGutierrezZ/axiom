package gitcli

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
)

// Capacidad de `git merge-tree --write-tree` (Q1, design D-09): la capa 2 de la detección de
// fusión de `done` (`equivalent_tree`) la exige y solo existe desde git 2.38. `gitcli` lee `git
// version` una vez por proceso y la habilita con 2.38.0 o superior; si la versión no se puede leer,
// intenta la orden y trata la salida 129 (uso incorrecto) como «no admitida». Una capa no admitida
// no es un error: `done` la omite con un aviso.

// mergeTreeMinimum es la primera versión de git con `merge-tree --write-tree`.
const (
	mergeTreeMinimumMajor = 2
	mergeTreeMinimumMinor = 38
)

// usageExitCode es la salida con la que git rechaza un subcomando o una bandera que no conoce.
const usageExitCode = 129

// MergeTreeSupportedOverride fija, solo en las pruebas, si se da por admitido `merge-tree`: falso
// simula un git anterior a la 2.38 sin lanzar nada y cierto salta la lectura de la versión.
// Ningún código de producción lo cambia.
var MergeTreeSupportedOverride *bool

// gitVersion es la versión de git leída, o known = falso si no se pudo leer o no se entiende.
type gitVersion struct {
	major, minor int
	known        bool
}

// atLeast dice si la versión es igual o posterior a major.minor. Una versión desconocida nunca lo es.
func (v gitVersion) atLeast(major, minor int) bool {
	return v.known && (v.major > major || (v.major == major && v.minor >= minor))
}

var versionNumbers = regexp.MustCompile(`(\d+)\.(\d+)`)

// parseGitVersion interpreta la salida de `git version` (`git version 2.41.0`,
// `git version 2.55.0.windows.5`...). Una salida que no se entiende da una versión desconocida.
func parseGitVersion(output string) gitVersion {
	numbers := versionNumbers.FindStringSubmatch(output)
	if numbers == nil {
		return gitVersion{}
	}
	major, errMajor := strconv.Atoi(numbers[1])
	minor, errMinor := strconv.Atoi(numbers[2])
	if errMajor != nil || errMinor != nil {
		return gitVersion{}
	}
	return gitVersion{major: major, minor: minor, known: true}
}

// processVersion guarda la versión de git que se leyó la primera vez en este proceso.
type processVersion struct {
	once    sync.Once
	version gitVersion
}

var installedVersion processVersion

// read devuelve la versión de git, leyéndola con `git version` desde dir la primera vez (la orden
// no necesita repositorio, pero gitcommon.Run exige un directorio existente). Una lectura que falla
// deja la versión desconocida para todo el proceso: `git version` no cambia mientras corre.
func (p *processVersion) read(ctx context.Context, dir string) gitVersion {
	p.once.Do(func() {
		result, err := gitcommon.Run(ctx, dir, "version")
		if err != nil || result.ExitCode != 0 {
			return
		}
		p.version = parseGitVersion(string(result.Stdout))
	})
	return p.version
}

// mergeTreeKnownUnsupported dice si ya se sabe, sin lanzar `merge-tree`, que git no lo admite: la
// variable de prueba o una versión leída anterior a la 2.38. Con la versión desconocida no se sabe y
// se intenta la orden.
func mergeTreeKnownUnsupported(ctx context.Context, dir string) bool {
	if override := MergeTreeSupportedOverride; override != nil {
		return !*override
	}
	version := installedVersion.read(ctx, dir)
	return version.known && !version.atLeast(mergeTreeMinimumMajor, mergeTreeMinimumMinor)
}

// mergeTreeResult es lo que significa la salida de `git merge-tree --write-tree`.
type mergeTreeResult int

const (
	mergeTreeClean       mergeTreeResult = iota // salida 0: fusión limpia, la primera línea es el árbol
	mergeTreeConflicts                          // salida 1: la fusión tiene conflictos
	mergeTreeUnsupported                        // salida 129: git no conoce la orden o sus banderas
	mergeTreeFailed                             // cualquier otra: un fallo de git
)

// classifyMergeTree interpreta el código de salida de `git merge-tree --write-tree`: la salida 129
// es lo que da un git sin soporte (Q1) y no un fallo.
func classifyMergeTree(exitCode int) mergeTreeResult {
	switch exitCode {
	case 0:
		return mergeTreeClean
	case 1:
		return mergeTreeConflicts
	case usageExitCode:
		return mergeTreeUnsupported
	}
	return mergeTreeFailed
}

// MergeTreeEquivalent dice si la fusión simulada de branchRef en target produce el árbol de target
// (`git merge-tree --write-tree --no-messages <target> <branchRef>`, salida 0, y su primera línea
// igual a `git rev-parse <target>^{tree}`): la capa 2 de la detección de fusión de `done`
// (`equivalent_tree`, REQ-25.16), que detecta un squash o un rebase aunque target haya avanzado
// después. supported es falso, sin error, si git no admite la orden (versión anterior a la 2.38 o
// salida 129): la capa se omite. Una fusión con conflictos (salida 1) es admitida y no equivalente.
// Solo usa referencias locales; la orden escribe los objetos de árbol que calcula en el repositorio
// pero no toca ninguna referencia ni el árbol de trabajo. target y branchRef son referencias
// completas y dir, el checkout principal.
func (*Git) MergeTreeEquivalent(ctx context.Context, dir, target, branchRef string) (equivalent, supported bool, err error) {
	if err := plainArgument(target); err != nil {
		return false, false, err
	}
	if err := plainArgument(branchRef); err != nil {
		return false, false, err
	}
	if mergeTreeKnownUnsupported(ctx, dir) {
		return false, false, nil
	}
	args := []string{"merge-tree", "--write-tree", "--no-messages", target, branchRef}
	result, err := gitcommon.Run(ctx, dir, args...)
	if err != nil {
		return false, false, err
	}
	switch classifyMergeTree(result.ExitCode) {
	case mergeTreeConflicts:
		return false, true, nil
	case mergeTreeUnsupported:
		return false, false, nil
	case mergeTreeFailed:
		return false, false, commandFailed(dir, result, args...)
	}
	merged, _, _ := strings.Cut(strings.TrimSpace(string(result.Stdout)), "\n")

	treeArgs := []string{"rev-parse", "--verify", "--quiet", target + "^{tree}"}
	tree, err := gitcommon.Run(ctx, dir, treeArgs...)
	if err != nil {
		return false, false, err
	}
	if tree.ExitCode != 0 {
		return false, false, commandFailed(dir, tree, treeArgs...)
	}
	return strings.TrimSpace(merged) == strings.TrimSpace(string(tree.Stdout)), true, nil
}
