//go:build windows || darwin

package secrets

import (
	"context"
	"errors"

	"github.com/zalando/go-keyring"
)

// keyringService es el servicio bajo el que Axiom guarda sus secretos de integración; la cuenta es
// <plugin>/<secreto>.
const keyringService = "axiom"

// Keyring guarda los secretos en el almacén del sistema (Windows Credential Manager o llavero de
// macOS). Solo se compila en Windows y macOS: en el resto de sistemas el único almacén es la
// variable de entorno.
type Keyring struct{}

func (Keyring) Get(_ context.Context, ref Ref) (Secret, Source, error) {
	value, err := keyring.Get(keyringService, ref.String())
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return Secret{}, SourceMissing, ErrNotFound
		}
		return Secret{}, SourceKeyringUnavailable, err
	}
	return NewSecret(value), SourceKeyring, nil
}

func (Keyring) Set(_ context.Context, ref Ref, value Secret) error {
	return keyring.Set(keyringService, ref.String(), value.Reveal())
}

// Delete es idempotente: borrar un secreto inexistente no es un error.
func (Keyring) Delete(_ context.Context, ref Ref) error {
	if err := keyring.Delete(keyringService, ref.String()); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	return nil
}

// platformKeyring devuelve el almacén del sistema de esta plataforma.
func platformKeyring() WritableStore { return Keyring{} }
