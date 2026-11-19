//go:build !windows && !darwin

package secrets

// platformKeyring no existe fuera de Windows y macOS: sin Secret Service, el único almacén es la
// variable de entorno AXIOM_INTEGRATION_<PLUGIN>_<SECRETO> y `secret set` se niega con un error
// tipado (nunca se escribe un secreto en texto plano).
func platformKeyring() WritableStore { return nil }
