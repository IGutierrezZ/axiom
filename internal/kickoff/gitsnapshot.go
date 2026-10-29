package kickoff

import (
	"fmt"
	"path"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
)

// La instantánea de política git (REQ-25.36, design D-12 y §4.5) es el bloque `git:` de
// `kickoff.yaml`: lo que `kickoff seal` escribe, una sola vez y en la misma escritura que el
// resto de la configuración sellada, con la política que regía al sellar. Contiene entradas
// de política —la integración y el prefijo resueltos, las excepciones por cambio de `ensure`
// y, por repositorio, la base (un nombre de rama) y los nombres de rama derivados del cambio—,
// nunca resultados: ni SHAs, ni rutas absolutas, ni rutas de worktree, que son personales o
// cambian.
//
// El bloque es aditivo: la lectura de `kickoff.yaml` no es estricta, así que un binario
// anterior lo ignora sin error, y su ausencia (un cambio preexistente, un retro-sellado con
// `--infer`) deja el cambio exento de las guardas de rama (REQ-25.37).

// GitSnapshot es la instantánea de política git de un cambio.
type GitSnapshot struct {
	// IntegrationBranch y BranchPrefix son los del workspace, resueltos al sellar.
	IntegrationBranch string `yaml:"integration_branch"`
	BranchPrefix      string `yaml:"branch_prefix"`
	// Exceptions son las excepciones por cambio de `ensure` que regían al sellar; nil si no había.
	Exceptions *GitExceptions `yaml:"exceptions,omitempty"`
	// ChangeBranch es la rama del cambio. Vacía si ningún repositorio la tiene (un multirepo
	// cuyo único repositorio sellado es el de specs, que se trabaja in situ).
	ChangeBranch string            `yaml:"change_branch"`
	Repositories []GitSnapshotRepo `yaml:"repositories"`
}

// GitExceptions son las excepciones por cambio de `ensure` (`--prefix`, `--branch`, `--base`).
type GitExceptions struct {
	Prefix string `yaml:"prefix,omitempty"`
	Branch string `yaml:"branch,omitempty"`
	Base   string `yaml:"base,omitempty"`
}

// GitSnapshotRepo es la política de un repositorio del cambio.
type GitSnapshotRepo struct {
	// Path es la ruta del repositorio relativa a la carpeta maestra, con «/».
	Path string `yaml:"path"`
	Name string `yaml:"name"`
	// Role es el papel del repositorio: change_branch, specs_in_place o code.
	Role              string `yaml:"role"`
	IntegrationBranch string `yaml:"integration_branch"`
	// Base es el nombre de la referencia de la que salen las ramas, nunca un SHA.
	Base     string              `yaml:"base"`
	Branches []GitSnapshotBranch `yaml:"branches,omitempty"`
}

// GitSnapshotBranch es una rama derivada del cambio.
type GitSnapshotBranch struct {
	Name string `yaml:"name"`
	// Kind es el tipo de rama: change, role o archive.
	Kind string `yaml:"kind"`
	// Role es el rol de una rama de rol; vacío en las demás.
	Role string `yaml:"role,omitempty"`
}

// Papeles de repositorio y tipos de rama de la instantánea: los tokens de internal/branch
// (RepoRole y BranchKind), repetidos aquí porque `kickoff` no puede importar `branch`.
var (
	validSnapshotRoles = map[string]bool{"change_branch": true, "specs_in_place": true, "code": true}
	validSnapshotKinds = map[string]bool{"change": true, "role": true, "archive": true}
)

// Validate comprueba la forma de la instantánea con las reglas de gitref (ramas y prefijos),
// los vocabularios cerrados de papeles y tipos de rama y que no lleve SHAs ni rutas absolutas.
// Nunca coacciona un valor: lo que no encaja se rechaza nombrando el campo.
func (g GitSnapshot) Validate() error {
	if err := gitref.ValidRefName(g.IntegrationBranch); err != nil {
		return fmt.Errorf("kickoff: git.integration_branch %q no es válido: %w", g.IntegrationBranch, err)
	}
	if err := gitref.ValidPrefix(g.BranchPrefix); err != nil {
		return fmt.Errorf("kickoff: git.branch_prefix %q no es válido: %w", g.BranchPrefix, err)
	}
	if g.ChangeBranch != "" {
		if err := gitref.ValidRefName(g.ChangeBranch); err != nil {
			return fmt.Errorf("kickoff: git.change_branch %q no es válido: %w", g.ChangeBranch, err)
		}
	}
	if e := g.Exceptions; e != nil {
		if e.Prefix != "" {
			if err := gitref.ValidPrefix(e.Prefix); err != nil {
				return fmt.Errorf("kickoff: git.exceptions.prefix %q no es válido: %w", e.Prefix, err)
			}
		}
		if e.Branch != "" {
			if err := gitref.ValidRefName(e.Branch); err != nil {
				return fmt.Errorf("kickoff: git.exceptions.branch %q no es válido: %w", e.Branch, err)
			}
		}
		if e.Base != "" {
			if err := validBase(e.Base); err != nil {
				return fmt.Errorf("kickoff: git.exceptions.base %q no es válido: %w", e.Base, err)
			}
		}
	}
	for i, repo := range g.Repositories {
		if err := repo.validate(); err != nil {
			return fmt.Errorf("kickoff: git.repositories[%d]: %w", i, err)
		}
	}
	return nil
}

func (r GitSnapshotRepo) validate() error {
	if err := validRelativePath(r.Path); err != nil {
		return fmt.Errorf("path %q no es válido: %w", r.Path, err)
	}
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("el repositorio %q no tiene nombre", r.Path)
	}
	if !validSnapshotRoles[r.Role] {
		return fmt.Errorf("role desconocido %q para el repositorio %q", r.Role, r.Path)
	}
	if err := gitref.ValidRefName(r.IntegrationBranch); err != nil {
		return fmt.Errorf("integration_branch %q no es válido: %w", r.IntegrationBranch, err)
	}
	if err := validBase(r.Base); err != nil {
		return fmt.Errorf("base %q no es válida: %w", r.Base, err)
	}
	for _, branch := range r.Branches {
		if !validSnapshotKinds[branch.Kind] {
			return fmt.Errorf("kind desconocido %q para la rama %q", branch.Kind, branch.Name)
		}
		if err := gitref.ValidRefName(branch.Name); err != nil {
			return fmt.Errorf("la rama %q no es válida: %w", branch.Name, err)
		}
		if (branch.Kind == "role") != (branch.Role != "") {
			return fmt.Errorf("la rama %q de tipo %s debe llevar role si y solo si es de rol", branch.Name, branch.Kind)
		}
	}
	return nil
}

// validRelativePath rechaza una ruta vacía o absoluta, en la grafía de POSIX (`/x`) o de Windows
// (`C:\x`, `C:/x`, `\\servidor`): la instantánea no lleva rutas absolutas.
func validRelativePath(value string) error {
	switch {
	case strings.TrimSpace(value) == "":
		return fmt.Errorf("no puede estar vacía")
	case isAbsolutePath(value):
		return fmt.Errorf("no puede ser absoluta: la instantánea lleva rutas relativas a la carpeta maestra")
	}
	return nil
}

func isAbsolutePath(value string) bool {
	value = strings.ReplaceAll(value, `\`, "/")
	if path.IsAbs(value) {
		return true
	}
	return len(value) >= 2 && value[1] == ':' && (value[0]|0x20 >= 'a' && value[0]|0x20 <= 'z')
}

// validBase comprueba que una base es el nombre de una referencia: no vacía, sin espacios ni
// caracteres de control, y ni un SHA ni una ruta absoluta (la instantánea es política, no
// resultados). Es más laxa que un nombre de rama porque `--base` admite cualquier referencia
// que git resuelva (`origin/main`, `HEAD~1`).
func validBase(value string) error {
	switch {
	case value == "":
		return fmt.Errorf("no puede estar vacía")
	case isObjectID(value):
		return fmt.Errorf("es un identificador de objeto: la base se guarda como nombre de referencia, nunca como SHA")
	case isAbsolutePath(value):
		return fmt.Errorf("es una ruta absoluta: la base se guarda como nombre de referencia")
	}
	for _, char := range value {
		if char <= ' ' || char == 0x7f {
			return fmt.Errorf("no puede contener espacios ni caracteres de control")
		}
	}
	return nil
}

// isObjectID dice si value es un identificador de objeto completo de git: 40 hexadecimales
// (SHA-1) o 64 (SHA-256).
func isObjectID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, char := range value {
		switch {
		case char >= '0' && char <= '9', char >= 'a' && char <= 'f', char >= 'A' && char <= 'F':
		default:
			return false
		}
	}
	return true
}
