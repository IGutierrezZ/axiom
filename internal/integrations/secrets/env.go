package secrets

import (
	"context"
	"os"
)

// Env lee los secretos de las variables de entorno AXIOM_INTEGRATION_<PLUGIN>_<CLAVE>. Es solo
// lectura: una variable de entorno no se escribe ni se borra desde Axiom.
type Env struct {
	// Getenv sustituye a os.Getenv en las pruebas; nil usa os.Getenv.
	Getenv func(string) string
}

// Get devuelve el valor de la variable del secreto. Una variable vacía cuenta como ausente.
func (e Env) Get(_ context.Context, ref Ref) (Secret, Source, error) {
	getenv := e.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	if v := getenv(ref.EnvVar()); v != "" {
		return NewSecret(v), SourceEnv, nil
	}
	return Secret{}, SourceMissing, ErrNotFound
}
