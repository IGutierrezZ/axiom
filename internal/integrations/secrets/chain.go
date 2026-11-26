package secrets

import (
	"context"
	"errors"
)

// Chain resuelve primero el entorno y, si no hay valor, un almacén persistente opcional (keyring).
// Nunca hay un tercer eslabón de fichero en texto plano.
type Chain struct {
	env    Store
	second WritableStore // nil: solo entorno
}

// NewChain compone el entorno con un segundo eslabón opcional (puede ser nil).
func NewChain(env Store, second WritableStore) *Chain {
	return &Chain{env: env, second: second}
}

// Get aplica la precedencia: entorno no vacío, segundo eslabón, error tipado.
func (c *Chain) Get(ctx context.Context, ref Ref) (Secret, Source, error) {
	if s, src, err := c.env.Get(ctx, ref); err == nil {
		return s, src, nil
	}
	if c.second == nil {
		return Secret{}, SourceMissing, &MissingError{Ref: ref}
	}
	s, _, err := c.second.Get(ctx, ref)
	switch {
	case err == nil:
		return s, SourceKeyring, nil
	case errors.Is(err, ErrNotFound):
		return Secret{}, SourceMissing, &MissingError{Ref: ref}
	default:
		return Secret{}, SourceKeyringUnavailable, &KeyringUnavailableError{Ref: ref, EnvVar: ref.EnvVar(), Cause: err}
	}
}

// Set guarda el secreto en el segundo eslabón; nunca en el entorno ni en un fichero.
func (c *Chain) Set(ctx context.Context, ref Ref, value Secret) error {
	if c.second == nil {
		return &KeyringUnavailableError{Ref: ref, EnvVar: ref.EnvVar(), Cause: ErrNoWritableStore}
	}
	if len(value.Reveal()) > MaxValueBytes {
		return &ValueTooLargeError{Ref: ref, Max: MaxValueBytes, EnvVar: ref.EnvVar()}
	}
	if err := c.second.Set(ctx, ref, value); err != nil {
		return &KeyringUnavailableError{Ref: ref, EnvVar: ref.EnvVar(), Cause: err}
	}
	return nil
}

// Delete borra el secreto del segundo eslabón. Es idempotente.
func (c *Chain) Delete(ctx context.Context, ref Ref) error {
	if c.second == nil {
		return &KeyringUnavailableError{Ref: ref, EnvVar: ref.EnvVar(), Cause: ErrNoWritableStore}
	}
	if err := c.second.Delete(ctx, ref); err != nil && !errors.Is(err, ErrNotFound) {
		return &KeyringUnavailableError{Ref: ref, EnvVar: ref.EnvVar(), Cause: err}
	}
	return nil
}
