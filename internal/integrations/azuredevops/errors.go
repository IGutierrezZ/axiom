package azuredevops

import "errors"

// ErrConfig indica que integrations.azure-devops no trae un valor utilizable (clave ausente, vacía o
// que no es una cadena). La carga de axiom.yaml ya valida el bloque: este error cubre a quien llama
// al plugin con una configuración construida a mano.
var ErrConfig = errors.New("configuración de azure-devops inválida")

// ErrLifecycleUnsupported marca la entrega de eventos del ciclo de vida: este adaptador todavía no
// entrega ninguno; llegan con C-07-d y C-07-e.
var ErrLifecycleUnsupported = errors.New("el plugin azure-devops no entrega eventos del ciclo de vida en esta versión; llegan con C-07-d y C-07-e")
