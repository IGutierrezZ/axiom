package manifest

import (
	"fmt"
	"sync"
)

// builtinCatalog es la lista cerrada de manifiestos de esta versión. Está vacía hasta que C-07-c
// añada el primer adaptador real: cualquier plugin declarado en axiom.yaml es un error de carga.
var builtinCatalog = []Manifest{}

var (
	catalogMu sync.RWMutex
	catalog   = builtinCatalog
)

// Catalog devuelve una copia profunda del catálogo cerrado de manifiestos.
func Catalog() []Manifest {
	catalogMu.RLock()
	defer catalogMu.RUnlock()
	out := make([]Manifest, len(catalog))
	for i, m := range catalog {
		out[i] = m.Clone()
	}
	return out
}

// Lookup devuelve una copia del manifiesto del plugin id.
func Lookup(id string) (Manifest, bool) {
	catalogMu.RLock()
	defer catalogMu.RUnlock()
	for _, m := range catalog {
		if m.ID == id {
			return m.Clone(), true
		}
	}
	return Manifest{}, false
}

// ValidateCatalog comprueba cada manifiesto y las reglas que solo se ven en conjunto: identificadores
// únicos y una variable de entorno única para todo par (plugin, secreto).
func ValidateCatalog(manifests []Manifest) error {
	var problems []string
	ids := map[string]bool{}
	envs := map[string]string{}
	for _, m := range manifests {
		if err := m.Validate(); err != nil {
			if ve, ok := err.(*ValidationError); ok {
				for _, p := range ve.Problems {
					problems = append(problems, fmt.Sprintf("plugin %q: %s", m.ID, p))
				}
			} else {
				problems = append(problems, fmt.Sprintf("plugin %q: %v", m.ID, err))
			}
		}
		if ids[m.ID] {
			problems = append(problems, fmt.Sprintf("identificador de plugin %q repetido en el catálogo", m.ID))
		}
		ids[m.ID] = true
		for _, s := range m.Secrets {
			env := EnvVarName(m.ID, s.Name)
			pair := m.ID + "/" + s.Name
			if other, dup := envs[env]; dup && other != pair {
				problems = append(problems, fmt.Sprintf("los secretos %s y %s dan la misma variable de entorno %s", other, pair, env))
				continue
			}
			envs[env] = pair
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return &ValidationError{Problems: problems}
}

// SetCatalogForTest instala manifests como catálogo y lo restaura con Cleanup. Es una costura de
// prueba (estado de paquete): las pruebas que la usan no llaman a t.Parallel(). Un catálogo
// inválido entra en pánico para que la prueba falle en el acto.
func SetCatalogForTest(t interface {
	Helper()
	Cleanup(func())
}, manifests ...Manifest) {
	t.Helper()
	if err := ValidateCatalog(manifests); err != nil {
		panic(fmt.Sprintf("SetCatalogForTest: %v", err))
	}
	installed := make([]Manifest, len(manifests))
	for i, m := range manifests {
		installed[i] = m.Clone()
	}
	catalogMu.Lock()
	previous := catalog
	catalog = installed
	catalogMu.Unlock()
	t.Cleanup(func() {
		catalogMu.Lock()
		catalog = previous
		catalogMu.Unlock()
	})
}
