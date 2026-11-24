package livespec

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// RefKind dice de dónde sale una referencia del registro.
type RefKind string

const (
	// RefSpec es un requisito de <planning home>/specs/<capacidad>/spec.md.
	RefSpec RefKind = "spec"
	// RefChange es un requisito ADDED, MODIFIED o RENAMED de la delta de un cambio activo.
	RefChange RefKind = "change"
)

// IDRef es una aparición de un identificador de requisito.
type IDRef struct {
	ID          string
	Kind        RefKind
	Capability  string
	Change      string // solo con RefChange
	Path        string
	Requirement string
}

// Where nombra el lugar de la aparición para los motivos de rechazo.
func (r IDRef) Where() string {
	if r.Kind == RefChange {
		return fmt.Sprintf("el cambio activo %s (capacidad %s)", r.Change, r.Capability)
	}
	return "la capacidad " + r.Capability
}

// RegistryOptions acota el registro: ExcludeChange deja fuera la delta de un cambio (el
// propio de `adopt check`) y SpecsOnly indexa solo openspec/specs/. Pattern es el patrón de
// identificadores del proyecto (REQ-44.1); el valor cero es el patrón por defecto.
type RegistryOptions struct {
	ExcludeChange string
	SpecsOnly     bool
	Pattern       reqid.Pattern
}

// Registry es el índice único de identificadores de requisito (D6) de las
// especificaciones del planning home y de las deltas de sus cambios activos, con el patrón
// del proyecto. No indexa `changes/archive/`.
type Registry struct {
	pattern reqid.Pattern
	byID    map[string][]IDRef
	maxFor  map[reqid.Series]int
}

// Pattern devuelve el patrón de identificadores con el que se construyó el registro.
func (r *Registry) Pattern() reqid.Pattern { return r.pattern }

// LoadRegistry construye el registro de <planningHome>/specs/*/spec.md y, salvo con
// SpecsOnly, de <planningHome>/changes/*/specs/*/spec.md. Una carpeta que no existe no
// es un error: el registro queda vacío en esa parte.
func LoadRegistry(planningHome string, opts RegistryOptions) (*Registry, error) {
	reg := &Registry{pattern: opts.Pattern, byID: map[string][]IDRef{}, maxFor: map[reqid.Series]int{}}

	specsDir := filepath.Join(planningHome, "specs")
	capabilities, err := subdirectories(specsDir)
	if err != nil {
		return nil, err
	}
	for _, capability := range capabilities {
		path := filepath.Join(specsDir, capability, "spec.md")
		data, err := readOptional(path)
		if err != nil {
			return nil, err
		}
		for _, r := range openspecdoc.ParseCanonical(string(data)) {
			reg.add(IDRef{ID: reg.pattern.InTitle(r.Name), Kind: RefSpec, Capability: capability, Path: path, Requirement: r.Name})
		}
	}
	if opts.SpecsOnly {
		return reg, nil
	}

	changesDir := filepath.Join(planningHome, "changes")
	changes, err := subdirectories(changesDir)
	if err != nil {
		return nil, err
	}
	for _, change := range changes {
		if change == "archive" || change == opts.ExcludeChange {
			continue
		}
		deltaDir := filepath.Join(changesDir, change, "specs")
		deltaCapabilities, err := subdirectories(deltaDir)
		if err != nil {
			return nil, err
		}
		for _, capability := range deltaCapabilities {
			path := filepath.Join(deltaDir, capability, "spec.md")
			data, err := readOptional(path)
			if err != nil {
				return nil, err
			}
			delta := openspecdoc.ParseDelta(string(data))
			for _, group := range [][]openspecdoc.Requirement{delta.Added, delta.Modified, delta.Renamed} {
				for _, r := range group {
					reg.add(IDRef{ID: reg.pattern.InTitle(r.Name), Kind: RefChange, Capability: capability, Change: change, Path: path, Requirement: r.Name})
				}
			}
		}
	}
	return reg, nil
}

func (r *Registry) add(ref IDRef) {
	if ref.ID == "" {
		return
	}
	r.byID[ref.ID] = append(r.byID[ref.ID], ref)
	if series, ok := r.pattern.Series(ref.ID); ok {
		if number, _ := r.pattern.Number(ref.ID); number > r.maxFor[series] {
			r.maxFor[series] = number
		}
	}
}

// Lookup devuelve todas las apariciones de un identificador, en un orden estable.
func (r *Registry) Lookup(id string) []IDRef {
	refs := append([]IDRef(nil), r.byID[id]...)
	sort.SliceStable(refs, func(i, j int) bool { return refs[i].Path < refs[j].Path })
	return refs
}

// NextFree devuelve el siguiente identificador libre de una serie del patrón del registro:
// el número máximo de la serie en todo el registro, más uno, con el relleno que declara la
// plantilla (`RF-USU-09` da `RF-USU-10`, sin truncar). Los huecos no se rellenan y otra
// serie no cuenta. Solo parte de una serie existente: la asignación de una serie vacía no
// es de este paquete (C-03-b).
func (r *Registry) NextFree(series reqid.Series) string {
	return r.pattern.Format(series, r.maxFor[series]+1)
}

// SeriesRefs devuelve todas las apariciones de identificadores de una serie, en un orden
// estable (por ruta y después por identificador).
func (r *Registry) SeriesRefs(series reqid.Series) []IDRef {
	var refs []IDRef
	for id, list := range r.byID {
		if s, ok := r.pattern.Series(id); ok && s == series {
			refs = append(refs, list...)
		}
	}
	sort.SliceStable(refs, func(i, j int) bool {
		if refs[i].Path != refs[j].Path {
			return refs[i].Path < refs[j].Path
		}
		return refs[i].ID < refs[j].ID
	})
	return refs
}

// subdirectories lista los directorios de dir, ordenados; un dir que no existe da una
// lista vacía.
func subdirectories(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("no se puede leer %s: %w; comprueba los permisos de la carpeta", dir, err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// readOptional lee un fichero; uno que no existe da contenido vacío.
func readOptional(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("no se puede leer %s: %w; comprueba los permisos del fichero", path, err)
	}
	return data, nil
}
