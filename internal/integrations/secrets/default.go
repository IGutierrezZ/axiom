package secrets

import (
	"context"
	"sync"
)

var (
	defaultMu    sync.RWMutex
	defaultStore WritableStore
)

// Default devuelve el almacén de producción. De momento compone solo el entorno; la unidad del
// keyring le añade el segundo eslabón sin cambiar esta firma.
func Default() WritableStore {
	defaultMu.RLock()
	override := defaultStore
	defaultMu.RUnlock()
	if override != nil {
		return override
	}
	return NewChain(Env{}, nil)
}

// SetDefaultForTest instala store como almacén por defecto y lo restaura con Cleanup. Es estado
// de paquete: las pruebas que lo usan no llaman a t.Parallel().
func SetDefaultForTest(t interface {
	Helper()
	Cleanup(func())
}, store WritableStore) {
	t.Helper()
	defaultMu.Lock()
	previous := defaultStore
	defaultStore = store
	defaultMu.Unlock()
	t.Cleanup(func() {
		defaultMu.Lock()
		defaultStore = previous
		defaultMu.Unlock()
	})
}

// Memory es un almacén en memoria para pruebas. Se comporta como el segundo eslabón (keyring) de
// la cadena: ErrNotFound cuando falta el secreto.
type Memory struct {
	mu     sync.Mutex
	values map[Ref]string
}

// NewMemory crea un almacén en memoria vacío.
func NewMemory() *Memory { return &Memory{values: map[Ref]string{}} }

// Get devuelve el valor guardado o ErrNotFound.
func (m *Memory) Get(_ context.Context, ref Ref) (Secret, Source, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if v, ok := m.values[ref]; ok {
		return NewSecret(v), SourceKeyring, nil
	}
	return Secret{}, SourceMissing, ErrNotFound
}

// Set guarda el valor.
func (m *Memory) Set(_ context.Context, ref Ref, value Secret) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.values == nil {
		m.values = map[Ref]string{}
	}
	m.values[ref] = value.Reveal()
	return nil
}

// Delete borra el valor; borrar uno inexistente no es un error.
func (m *Memory) Delete(_ context.Context, ref Ref) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.values, ref)
	return nil
}
