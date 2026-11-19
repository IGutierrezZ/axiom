package integrations

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

// ErrDuplicatePlugin indica dos plugins con el mismo identificador.
var ErrDuplicatePlugin = errors.New("plugin de integración duplicado")

// Registry es el conjunto cerrado de plugins instalados, por identificador.
type Registry struct {
	plugins map[string]Plugin
	ids     []string // en orden de registro
}

// NewRegistry construye un registro. Rechaza identificadores duplicados, manifiestos inválidos,
// identificadores ausentes del catálogo cerrado y manifiestos distintos (reflect.DeepEqual) del
// publicado en manifest.Catalog(): el código del plugin y el manifiesto que valida axiom.yaml no
// pueden divergir.
func NewRegistry(plugins ...Plugin) (*Registry, error) {
	r := &Registry{plugins: make(map[string]Plugin, len(plugins))}
	for _, p := range plugins {
		m := p.Manifest()
		if _, dup := r.plugins[m.ID]; dup {
			return nil, fmt.Errorf("%w: %q", ErrDuplicatePlugin, m.ID)
		}
		if err := m.Validate(); err != nil {
			return nil, fmt.Errorf("plugin %q: %w", m.ID, err)
		}
		cataloged, ok := manifest.Lookup(m.ID)
		if !ok {
			return nil, fmt.Errorf("el plugin %q no está en el catálogo cerrado de manifiestos", m.ID)
		}
		if !reflect.DeepEqual(m, cataloged) {
			return nil, fmt.Errorf("el manifiesto del plugin %q difiere del publicado en el catálogo cerrado", m.ID)
		}
		r.plugins[m.ID] = p
		r.ids = append(r.ids, m.ID)
	}
	return r, nil
}

// Get devuelve el plugin con ese identificador.
func (r *Registry) Get(id string) (Plugin, bool) {
	p, ok := r.plugins[id]
	return p, ok
}

// IDs devuelve los identificadores en orden de registro.
func (r *Registry) IDs() []string { return append([]string(nil), r.ids...) }

// builtinPlugins es la lista cerrada de plugins compilados en esta versión. Está vacía hasta
// C-07-c; añadir uno exige también añadir su manifiesto a manifest.builtinCatalog.
var builtinPlugins = []Plugin{}

var (
	defaultRegistryMu sync.RWMutex
	defaultRegistry   *Registry
)

// DefaultRegistry devuelve el registro de producción: la lista cerrada de esta versión.
func DefaultRegistry() (*Registry, error) {
	defaultRegistryMu.RLock()
	override := defaultRegistry
	defaultRegistryMu.RUnlock()
	if override != nil {
		return override, nil
	}
	return NewRegistry(builtinPlugins...)
}

// SetDefaultRegistryForTest instala un registro con plugins y lo restaura con Cleanup. Es estado de
// paquete: las pruebas que lo usan no llaman a t.Parallel(). Un registro inválido entra en
// pánico para que la prueba falle en el acto.
func SetDefaultRegistryForTest(t interface {
	Helper()
	Cleanup(func())
}, plugins ...Plugin) {
	t.Helper()
	reg, err := NewRegistry(plugins...)
	if err != nil {
		panic(fmt.Sprintf("SetDefaultRegistryForTest: %v", err))
	}
	defaultRegistryMu.Lock()
	previous := defaultRegistry
	defaultRegistry = reg
	defaultRegistryMu.Unlock()
	t.Cleanup(func() {
		defaultRegistryMu.Lock()
		defaultRegistry = previous
		defaultRegistryMu.Unlock()
	})
}
