package specbaseline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

const (
	// PlanSchema es el esquema de `integration.yaml` (D-13, §4.5).
	PlanSchema = "axiom.sdd-integration/v1"
	// PlanFileName es el nombre del plan dentro de la carpeta del cambio.
	PlanFileName = "integration.yaml"
	// MaxDirtyEntries es el máximo de entradas sucias que el plan registra por
	// repositorio: acota el tamaño de integration.yaml (D-13).
	MaxDirtyEntries = 1000
)

// Decisiones del delta de contexto que el plan registra.
const (
	// ContextEntries: el delta de contexto describe entradas.
	ContextEntries = "entries"
	// ContextNoImpact: el delta de contexto declara «Sin impacto».
	ContextNoImpact = "no_impact"
	// ContextNone: no hay contexto mantenido, o no hay delta.
	ContextNone = "none"
)

// Plan es el contenido de integration.yaml (axiom.sdd-integration/v1): lo que
// `prepare` fija y lo que `verify` comprueba (D-13, D-15).
type Plan struct {
	Schema       string           `yaml:"schema"`
	Change       string           `yaml:"change"`
	PreparedAt   time.Time        `yaml:"prepared_at"`
	VerifiedAt   *time.Time       `yaml:"verified_at,omitempty"`
	InputsSHA256 string           `yaml:"inputs_sha256"`
	Options      PlanOptions      `yaml:"options"`
	Repositories []PlanRepository `yaml:"repositories"`
	IDMap        []IDAssignment   `yaml:"id_map"`
	Documents    []PlanDocument   `yaml:"documents"`
	Context      ContextDecision  `yaml:"context"`
	Entries      []TraceEntry     `yaml:"entries"`
}

// PlanOptions son las opciones de supersede que el plan fija para el cambio.
type PlanOptions struct {
	Supersede              bool     `yaml:"supersede"`
	SupersededRequirements []string `yaml:"superseded_requirements"`
}

// PlanRepository es la línea base de un repositorio afectado: su HEAD y la foto
// de su suciedad previa.
type PlanRepository struct {
	Repo  string       `yaml:"repo"`  // relativo a la carpeta maestra, con '/'
	Head  string       `yaml:"head"`  // commit en prepare; "" si no hay commits
	Dirty []DirtyEntry `yaml:"dirty"` // como máximo MaxDirtyEntries
}

// DirtyEntry es un fichero con cambios sin confirmar en prepare.
type DirtyEntry struct {
	Path   string `yaml:"path"`   // relativa al repositorio, con '/'
	SHA256 string `yaml:"sha256"` // huella del árbol de trabajo; "" si el fichero estaba borrado
}

// PlanDocument es un documento de un dominio del cambio, con las huellas de su
// árbol de trabajo.
type PlanDocument struct {
	Domain       string                    `yaml:"domain"`
	Path         string                    `yaml:"path"` // relativo a workspaceRoot, con '/'
	Integration  workspace.IntegrationMode `yaml:"integration"`
	BeforeSHA256 string                    `yaml:"before_sha256"`
	AfterSHA256  string                    `yaml:"after_sha256,omitempty"` // solo compose
	State        TraceState                `yaml:"state,omitempty"`        // solo compose
}

// ContextDecision es la decisión del cambio sobre el contexto técnico.
type ContextDecision struct {
	Decision string `yaml:"decision"` // entries | no_impact | none
	Reason   string `yaml:"reason,omitempty"`
}

// sha256Hex devuelve el SHA-256 de data en hexadecimal.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// workingTreeFingerprint es la huella de un fichero del árbol de trabajo: el
// SHA-256 de su contenido, vacía si no existe (borrado). Un enlace simbólico o un
// directorio, como un repositorio anidado sin seguimiento, se identifican por su
// destino o por su naturaleza. Se calcula sobre el árbol de trabajo y nunca
// contra los bytes de un commit, que `core.autocrlf` puede normalizar (D-12).
func workingTreeFingerprint(file string) (string, error) {
	info, err := os.Lstat(file)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", nil
	case err != nil:
		return "", err
	case info.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(file)
		if err != nil {
			return "", err
		}
		return sha256Hex([]byte("symlink:" + target)), nil
	case info.IsDir():
		return sha256Hex([]byte("dir")), nil
	}
	handle, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer handle.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, handle); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// DeltaInput es el delta de un dominio tal como lo lee `prepare`.
type DeltaInput struct {
	Domain string
	Data   []byte
}

// inputsDigest calcula `inputs_sha256`: la huella de los deltas, del delta de
// contexto y de la instantánea de la declaración (D-13). Es independiente del
// orden en que se reciben los deltas. Un delta de contexto nil (ausente) y uno
// vacío producen huellas distintas.
func inputsDigest(deltas []DeltaInput, contextDelta, declaration []byte) string {
	sorted := append([]DeltaInput(nil), deltas...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Domain < sorted[j].Domain })
	hash := sha256.New()
	frame := func(tag string, data []byte) {
		fmt.Fprintf(hash, "%s\x00%d\x00", tag, len(data))
		hash.Write(data)
	}
	for _, delta := range sorted {
		frame("delta-domain", []byte(delta.Domain))
		frame("delta", delta.Data)
	}
	if contextDelta == nil {
		frame("context-absent", nil)
	} else {
		frame("context", contextDelta)
	}
	frame("declaration", declaration)
	return hex.EncodeToString(hash.Sum(nil))
}

// declarationSnapshot es la instantánea determinista de las declaraciones de
// axiom.yaml que influyen en la integración: la baseline y el contexto técnico.
func declarationSnapshot(declarations workspace.Declarations) []byte {
	data, err := yaml.Marshal(struct {
		Baseline         *workspace.BaselineConfig         `yaml:"baseline"`
		TechnicalContext *workspace.TechnicalContextConfig `yaml:"technical_context"`
	}{declarations.Baseline, declarations.TechnicalContext})
	if err != nil {
		return []byte("declaración no serializable: " + err.Error())
	}
	return data
}

// AlreadyIntegratedError es el rechazo de preparar o aplicar de nuevo un cambio
// cuyas composiciones ya figuran como integradas (REQ-24.43, D-13). Nombra la
// correspondencia de IDs y propone verificar o consultar la traza.
type AlreadyIntegratedError struct {
	Change string
	IDs    []IDAssignment
}

func (e *AlreadyIntegratedError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "el cambio %q figura ya integrado: todas sus composiciones están marcadas como integradas en %s y no se vuelven a aplicar", e.Change, PlanFileName)
	if len(e.IDs) > 0 {
		pairs := make([]string, len(e.IDs))
		for i, id := range e.IDs {
			pairs[i] = fmt.Sprintf("%s -> %s (dominio %q)", id.Provisional, id.Definitive, id.Domain)
		}
		fmt.Fprintf(&b, "; correspondencia de IDs: %s", strings.Join(pairs, ", "))
	}
	b.WriteString(". Continuación: comprueba la integración con `axiom sdd integrate verify` o consulta la traza con `axiom sdd integrate show`")
	return b.String()
}

// PlanChangedError es el rechazo de reemitir un plan cuyas entradas ya no son las
// actuales (D-13, R35): los deltas, el delta de contexto, la declaración o un
// documento compuesto cambiaron desde `prepare`.
type PlanChangedError struct {
	Change    string
	Documents []string // documentos compuestos del plan, relativos a workspaceRoot
}

func (e *PlanChangedError) Error() string {
	documents := "<documentos>"
	if len(e.Documents) > 0 {
		documents = strings.Join(e.Documents, " ")
	}
	return fmt.Sprintf("el plan de integración del cambio %q (%s) ya no corresponde a las entradas actuales: cambiaron los deltas, el delta de contexto, la declaración o un documento de la baseline desde `prepare`. Continuación: devolver los documentos a su estado anterior con `git checkout -- %s`, retirar %s de la carpeta del cambio y volver a preparar con `axiom sdd integrate prepare`", e.Change, PlanFileName, documents, PlanFileName)
}

// reconcilePlan decide qué hace `prepare` cuando ya existe un plan (D-13). Si todas
// las composiciones del plan existente figuran como integradas, rechaza como «ya
// integrado», con precedencia sobre cualquier otro cambio. Si los deltas, el delta
// de contexto, la declaración o la huella `before` de un documento compuesto
// difieren de las actuales (fresh), rechaza con la continuación. En otro caso
// reemite el plan existente, con su `HEAD` y su foto previa: un commit posterior
// no lo invalida, y conservarlos mantiene la protección de `verify` (D-15).
func reconcilePlan(existing, fresh Plan) (Plan, error) {
	var composed []PlanDocument
	for _, document := range existing.Documents {
		if document.Integration == workspace.IntegrationCompose {
			composed = append(composed, document)
		}
	}
	integrated := len(composed) > 0
	for _, document := range composed {
		integrated = integrated && document.State == StateIntegrated
	}
	if integrated {
		return Plan{}, &AlreadyIntegratedError{Change: existing.Change, IDs: existing.IDMap}
	}

	changed := existing.InputsSHA256 != fresh.InputsSHA256
	current := map[string]PlanDocument{}
	for _, document := range fresh.Documents {
		current[document.Domain] = document
	}
	paths := make([]string, len(composed))
	for i, document := range composed {
		paths[i] = document.Path
		if now, ok := current[document.Domain]; !ok || now.BeforeSHA256 != document.BeforeSHA256 {
			changed = true
		}
	}
	if changed {
		return Plan{}, &PlanChangedError{Change: existing.Change, Documents: paths}
	}
	return existing, nil
}

// affectedPath es un documento que la integración edita. Creates indica que
// archive lo crea (REQ-24.55): queda excluido de la precondición de cambios sin
// confirmar (REQ-24.49).
type affectedPath struct {
	Path    string // absoluta
	Creates bool
}

// PreconditionError es la parada de REQ-24.49: un documento afectado no está en
// un repositorio git, o tiene cambios sin confirmar o sin seguimiento, o hay
// demasiada suciedad ajena. Cada problema nombra los ficheros y la continuación.
type PreconditionError struct {
	Problems []string
}

func (e *PreconditionError) Error() string { return strings.Join(e.Problems, "; ") }

// repositoryCapture es la línea base de los repositorios afectados y los avisos
// no bloqueantes de su captura.
type repositoryCapture struct {
	Repositories []PlanRepository
	Warnings     []string
}

const preparePreconditionContinuation = "confirma, guarda (`git stash -u`) o descarta los cambios y vuelve a ejecutar `axiom sdd integrate prepare`"

// repoWork reúne un repositorio con los documentos afectados que contiene.
type repoWork struct {
	repo  gitRepository
	files []affectedPath // con la ruta resuelta
}

// captureRepositories registra la línea base de cada repositorio afectado
// (REQ-24.49, D-15): su HEAD (vacío si no tiene commits) y la foto acotada de su
// suciedad previa, con la huella del árbol de trabajo de cada fichero. Los
// repositorios son los que contienen los documentos de affected, que además deben
// estar dentro de la carpeta maestra una vez resueltos los enlaces simbólicos, y
// los de also, que solo se vigilan: uno que no existe o no es git se avisa y nunca
// detiene. Devuelve *PreconditionError si un documento afectado está en un
// directorio que no es git, o con cambios en el índice, sin stage o sin
// seguimiento, o si un repositorio supera MaxDirtyEntries. Solo lee: nunca
// modifica el repositorio.
func captureRepositories(ctx context.Context, master string, affected []affectedPath, also []string) (repositoryCapture, error) {
	master = resolveDeepest(filepath.Clean(master))
	var works []*repoWork
	var problems, warnings []string

	workOf := func(repo gitRepository) *repoWork {
		for _, work := range works {
			if pathidentity.SameDirectory(work.repo.root, repo.root) {
				return work
			}
		}
		work := &repoWork{repo: repo}
		works = append(works, work)
		return work
	}

	for _, document := range affected {
		resolved := resolveDeepest(filepath.Clean(document.Path))
		if !pathidentity.Contains(master, resolved) {
			return repositoryCapture{}, fmt.Errorf("la ruta %s resuelve a %s, fuera de la carpeta maestra %s: un enlace simbólico no puede sacar la integración de ella", document.Path, resolved, master)
		}
		repo, err := openRepository(ctx, resolved)
		var notGit *NotGitRepositoryError
		if errors.As(err, &notGit) {
			problems = append(problems, fmt.Sprintf("el documento %s no está dentro de un repositorio git: la verificación por diff de la integración requiere un repositorio git; versiona su directorio con `git init` y confirma su estado actual, o declara el documento en un repositorio versionado", displayPath(master, resolved)))
			continue
		}
		if err != nil {
			return repositoryCapture{}, err
		}
		work := workOf(repo)
		work.files = append(work.files, affectedPath{Path: resolved, Creates: document.Creates})
	}

	for _, dir := range also {
		resolved := resolveDeepest(filepath.Clean(dir))
		if info, err := os.Stat(resolved); err != nil || !info.IsDir() {
			warnings = append(warnings, fmt.Sprintf("el repositorio %s que se vigila no existe; se omite", displayPath(master, resolved)))
			continue
		}
		repo, err := openRepository(ctx, resolved)
		var notGit *NotGitRepositoryError
		if errors.As(err, &notGit) {
			warnings = append(warnings, fmt.Sprintf("el directorio %s que se vigila no es un repositorio git; se omite", displayPath(master, resolved)))
			continue
		}
		if err != nil {
			return repositoryCapture{}, err
		}
		workOf(repo)
	}

	type captured struct {
		repository PlanRepository
		warnings   []string
	}
	var captures []captured
	for _, work := range works {
		name := repositoryName(master, work.repo.root)
		head, err := work.repo.head(ctx)
		if err != nil {
			return repositoryCapture{}, err
		}
		entries, err := work.repo.status(ctx)
		if err != nil {
			return repositoryCapture{}, err
		}
		if len(entries) > MaxDirtyEntries {
			problems = append(problems, fmt.Sprintf("el repositorio %s tiene %d entradas sin confirmar, por encima del límite de %d que el plan registra: %s", name, len(entries), MaxDirtyEntries, preparePreconditionContinuation))
			continue
		}
		if dirty := dirtyAffected(work, entries); len(dirty) > 0 {
			problems = append(problems, fmt.Sprintf("el repositorio %s tiene cambios sin confirmar en documentos que la integración modifica: %s; %s", name, strings.Join(dirty, ", "), preparePreconditionContinuation))
		}

		photo := make([]DirtyEntry, 0, len(entries))
		for _, entry := range entries {
			path := strings.TrimSuffix(entry.Path, "/")
			fingerprint, err := workingTreeFingerprint(filepath.Join(work.repo.root, filepath.FromSlash(path)))
			if err != nil {
				return repositoryCapture{}, fmt.Errorf("calcular la huella de %s en el repositorio %s: %w", path, name, err)
			}
			photo = append(photo, DirtyEntry{Path: path, SHA256: fingerprint})
		}
		state := work.repo.upstream(ctx)
		var notes []string
		if state.Behind > 0 {
			notes = append(notes, fmt.Sprintf("el repositorio %s va %s por detrás de su upstream (el dato procede de las referencias locales: Axiom nunca ejecuta fetch); si otros han integrado cambios, actualiza el repositorio antes de asignar IDs definitivos", name, pluralCommits(state.Behind)))
		}
		for _, note := range state.Notes {
			notes = append(notes, fmt.Sprintf("repositorio %s: %s", name, note))
		}
		captures = append(captures, captured{
			repository: PlanRepository{Repo: name, Head: head, Dirty: photo},
			warnings:   notes,
		})
	}

	if len(problems) > 0 {
		return repositoryCapture{}, &PreconditionError{Problems: problems}
	}
	sort.SliceStable(captures, func(i, j int) bool { return captures[i].repository.Repo < captures[j].repository.Repo })
	result := repositoryCapture{Warnings: warnings}
	for _, capture := range captures {
		result.Repositories = append(result.Repositories, capture.repository)
		result.Warnings = append(result.Warnings, capture.warnings...)
	}
	return result, nil
}

func pluralCommits(n int) string {
	if n == 1 {
		return "1 commit"
	}
	return fmt.Sprintf("%d commits", n)
}

// dirtyAffected devuelve, en orden y con su estado, los documentos afectados que
// tienen cualquier estado en `git status`: índice, árbol de trabajo o sin
// seguimiento. Los documentos que archive crea quedan excluidos.
func dirtyAffected(work *repoWork, entries []statusEntry) []string {
	var dirty []string
	for _, file := range work.files {
		if file.Creates {
			continue
		}
		relative, err := filepath.Rel(work.repo.root, file.Path)
		if err != nil {
			continue
		}
		relative = filepath.ToSlash(relative)
		for _, entry := range entries {
			if samePath(entry.Path, relative) {
				dirty = append(dirty, fmt.Sprintf("%s (%s)", relative, describeStatus(entry.Code)))
			}
		}
	}
	sort.Strings(dirty)
	return dirty
}

// samePath compara dos rutas relativas de git, sin distinguir mayúsculas donde el
// sistema de ficheros no las distingue.
func samePath(a, b string) bool {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// describeStatus describe en castellano un código XY de `git status`.
func describeStatus(code string) string {
	if code == "??" {
		return "sin seguimiento"
	}
	staged := code[0] != ' ' && code[0] != '?'
	worktree := code[1] != ' ' && code[1] != '?'
	switch {
	case staged && worktree:
		return "cambios en el índice y en el árbol de trabajo"
	case staged:
		return "cambios en el índice"
	default:
		return "cambios en el árbol de trabajo"
	}
}

// repositoryName es el nombre de un repositorio en el plan: su ruta relativa a la
// carpeta maestra, con `/`, y `.` si es la propia carpeta maestra.
func repositoryName(master, root string) string {
	relative, err := filepath.Rel(master, root)
	if err != nil {
		return filepath.ToSlash(root)
	}
	return filepath.ToSlash(relative)
}

// displayPath muestra una ruta relativa a la carpeta maestra si está dentro de ella.
func displayPath(master, p string) string {
	if relative, err := filepath.Rel(master, p); err == nil && !strings.HasPrefix(filepath.ToSlash(relative), "../") && relative != ".." {
		return filepath.ToSlash(relative)
	}
	return p
}
