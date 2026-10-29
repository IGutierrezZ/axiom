package branch

import (
	"errors"
	"fmt"
	pathpkg "path"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Policy es la política de ramas de un proyecto (REQ-25.3): la rama de integración
// y el prefijo del workspace, y los ajustes propios de cada repositorio que declara
// `repositories:`. La precedencia, de mayor a menor, es: excepción por cambio de
// `ensure` (se aplica en el planificador, sobre lo que devuelve For), entrada del
// repositorio, bloque `git:` y valores por defecto (`main` y `feat`).
type Policy struct {
	IntegrationBranch string
	BranchPrefix      string
	// PerRepository guarda lo declarado por cada repositorio, con la clave normalizada
	// (ruta relativa a la carpeta maestra, con «/»). Un campo vacío hereda el valor
	// del workspace; la herencia la resuelve For.
	PerRepository map[string]RepoPolicy
}

// RepoPolicy es la política efectiva de un repositorio (o lo declarado por él, en
// Policy.PerRepository).
type RepoPolicy struct{ IntegrationBranch, BranchPrefix string }

// ResolvePolicy resuelve la política a partir de la configuración cargada. Es total:
// con `cfg == nil` (sin axiom.yaml) o con `cfg.Git == nil` (una configuración
// construida a mano, o un bloque `git:` inválido) rigen los valores por defecto, y
// nunca falla. Que un valor sea inválido lo informa `ConfigRefusal` como
// `config_invalid`, a partir de lo que dejó la carga en Declarations.GitErr y
// Declarations.RepositoriesErr. El resultado no comparte estado con cfg. Si una ruta
// se declara dos veces, rige la primera entrada (la carga ya rechaza las repetidas).
func ResolvePolicy(cfg *workspace.WorkspaceConfig) Policy {
	policy := Policy{IntegrationBranch: workspace.DefaultIntegrationBranch, BranchPrefix: workspace.DefaultBranchPrefix}
	if cfg == nil {
		return policy
	}
	if git := cfg.Git; git != nil {
		if git.IntegrationBranch != "" {
			policy.IntegrationBranch = git.IntegrationBranch
		}
		if git.BranchPrefix != "" {
			policy.BranchPrefix = git.BranchPrefix
		}
	}
	for _, entry := range cfg.Repositories {
		key := repoKey(entry.Path)
		if _, declared := policy.PerRepository[key]; declared {
			continue
		}
		if policy.PerRepository == nil {
			policy.PerRepository = map[string]RepoPolicy{}
		}
		policy.PerRepository[key] = RepoPolicy{IntegrationBranch: entry.IntegrationBranch, BranchPrefix: entry.BranchPrefix}
	}
	return policy
}

// For devuelve la política efectiva del repositorio cuya ruta, relativa a la carpeta
// maestra y con «/», es repoRelPath (`.` para el repositorio de la propia carpeta
// maestra). La ruta se normaliza igual que las de `repositories:`, de modo que
// `./web/` y `web` son el mismo repositorio. Sin entrada propia, o en lo que la
// entrada no declara, rige el valor del workspace.
func (p Policy) For(repoRelPath string) RepoPolicy {
	effective := RepoPolicy{IntegrationBranch: p.IntegrationBranch, BranchPrefix: p.BranchPrefix}
	declared, ok := p.PerRepository[repoKey(repoRelPath)]
	if !ok {
		return effective
	}
	if declared.IntegrationBranch != "" {
		effective.IntegrationBranch = declared.IntegrationBranch
	}
	if declared.BranchPrefix != "" {
		effective.BranchPrefix = declared.BranchPrefix
	}
	return effective
}

// repoKey normaliza una ruta relativa a la carpeta maestra como lo hace la carga de
// `repositories:` (declarations.go, checkRelativeTo): sin espacios en los extremos,
// con «/» en lugar de «\» y limpia. La ruta vacía es la propia carpeta maestra.
func repoKey(path string) string {
	return pathpkg.Clean(strings.ReplaceAll(strings.TrimSpace(path), `\`, "/"))
}

// ConfigRefusal devuelve el rechazo `config_invalid` cuando la configuración del
// proyecto impide planificar las ramas (REQ-25.3): un axiom.yaml ilegible, o un
// bloque `git:` o `repositories:` inválido. Es nil cuando no hay nada que rechazar,
// también sin axiom.yaml, donde rigen los valores por defecto. Los errores de otras
// declaraciones (baseline, technical_context) no son asunto de la política de ramas.
//
// Distingue lo que viene de axiom.yaml o del hub (`config_invalid`) de un nombre
// inválido de la invocación (`invalid_name`): este último nace en el planificador.
func ConfigRefusal(loc workspace.Location) *Refusal {
	var causes []string
	for _, err := range []error{loc.ConfigErr, loc.Declarations.GitErr, loc.Declarations.RepositoriesErr} {
		if err != nil {
			causes = append(causes, singleLine(err))
		}
	}
	if len(causes) == 0 {
		return nil
	}
	return &Refusal{
		Code:         CodeConfigInvalid,
		Message:      fmt.Sprintf("la configuración de %s impide planificar las ramas: %s", loc.ConfigPath, strings.Join(causes, "; ")),
		Continuation: validateConfigContinuation(loc.Master),
		Details:      map[string]string{"path": loc.ConfigPath},
	}
}

// LocateRefusal convierte el fallo de la resolución del planning home (puntero roto,
// configuración que no se puede cargar, repositorio de especificaciones ausente) en
// `config_invalid`: sin ella no se sabe dónde planificar. Conserva la causa y la
// continuación tipadas de workspace.LocateError.
func LocateRefusal(err error) *Refusal {
	var located *workspace.LocateError
	if errors.As(err, &located) && located.Cause != "" {
		return &Refusal{
			Code:         CodeConfigInvalid,
			Message:      located.Cause,
			Continuation: located.Continuation,
			Details:      map[string]string{"path": located.Path},
		}
	}
	return &Refusal{Code: CodeConfigInvalid, Message: singleLine(err), Continuation: validateConfigContinuation("")}
}

// singleLine deja el texto de un error en una sola línea: la carga acumula varios
// errores separados por saltos de línea, y un mensaje del JSON es una frase.
func singleLine(err error) string {
	return strings.ReplaceAll(strings.TrimSpace(err.Error()), "\n", "; ")
}
