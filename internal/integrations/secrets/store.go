package secrets

import (
	"context"
	"errors"
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

// Ref identifica un secreto: el plugin que lo declara y su nombre en el manifiesto.
type Ref struct {
	Plugin string
	Name   string
}

func (r Ref) String() string { return r.Plugin + "/" + r.Name }

// EnvVar devuelve la variable de entorno que alimenta este secreto.
func (r Ref) EnvVar() string { return manifest.EnvVarName(r.Plugin, r.Name) }

// Source indica de dónde salió (o por qué no salió) el valor de un secreto.
type Source string

const (
	SourceEnv                Source = "env"
	SourceKeyring            Source = "keyring"
	SourceMissing            Source = "missing"
	SourceKeyringUnavailable Source = "keyring_unavailable"
)

// ErrNotFound es lo que devuelve un eslabón cuando no tiene el secreto. La cadena lo convierte
// en *MissingError.
var ErrNotFound = errors.New("secreto no encontrado")

// ErrNoWritableStore indica que no hay almacén en el que guardar o borrar un secreto.
var ErrNoWritableStore = errors.New("este sistema no tiene almacén de secretos propio (solo Windows y macOS): usa la variable de entorno, Axiom nunca escribe secretos en texto plano")

// MaxValueBytes es el tamaño máximo del valor de un secreto. Se fija en 2 KiB, por debajo de lo que
// admite el almacén de credenciales de Windows (unos 2,5 KB), para que un valor aceptado quepa en
// todos los almacenes en lugar de fallar en el sistema con un error opaco.
const MaxValueBytes = 2048

// ValueTooLargeError indica que el valor supera MaxValueBytes. Nunca lleva el valor.
type ValueTooLargeError struct {
	Ref    Ref
	Max    int
	EnvVar string
}

func (e *ValueTooLargeError) Error() string {
	return fmt.Sprintf(
		"el valor del secreto %s supera el máximo de %d bytes que admiten los almacenes del sistema: reduce su longitud o define la variable de entorno %s",
		e.Ref, e.Max, e.EnvVar)
}

// Store resuelve secretos. La fuente se devuelve siempre, también con error.
type Store interface {
	Get(ctx context.Context, ref Ref) (Secret, Source, error)
}

// WritableStore añade el ciclo de vida del secreto. Set y Delete solo van al almacén persistente
// (keyring), nunca al entorno ni a un fichero.
type WritableStore interface {
	Store
	Set(ctx context.Context, ref Ref, value Secret) error
	// Delete es idempotente: borrar un secreto inexistente no es un error.
	Delete(ctx context.Context, ref Ref) error
}

// MissingError indica que el secreto no está en ninguna fuente. Nunca lleva valor.
type MissingError struct {
	Ref Ref
}

func (e *MissingError) Error() string {
	return fmt.Sprintf(
		"falta el secreto %s: guárdalo con `axiom integrations secret set --plugin %s --name %s` (valor por stdin) o define la variable de entorno %s",
		e.Ref, e.Ref.Plugin, e.Ref.Name, e.Ref.EnvVar())
}

// Is permite errors.Is(err, ErrNotFound) sobre un *MissingError.
func (e *MissingError) Is(target error) bool { return target == ErrNotFound }

// KeyringUnavailableError indica que el almacén persistente falló por una causa distinta de
// «no existe» (sin servicio de secretos, plataforma no soportada...). Nunca se cae a un fichero
// en texto plano: la salida es la variable de entorno.
type KeyringUnavailableError struct {
	Ref    Ref
	EnvVar string
	Cause  error
}

func (e *KeyringUnavailableError) Error() string {
	return fmt.Sprintf(
		"el almacén de secretos del sistema no está disponible para %s (%v): define la variable de entorno %s",
		e.Ref, e.Cause, e.EnvVar)
}

func (e *KeyringUnavailableError) Unwrap() error { return e.Cause }
