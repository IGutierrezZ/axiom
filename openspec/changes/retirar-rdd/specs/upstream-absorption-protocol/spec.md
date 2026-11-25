# Delta for upstream-absorption-protocol

## MODIFIED Requirements

### Requirement: Verificación sin filtrar (REQ-20.2)

Toda tanda de absorción DEBE presentar como evidencia de verificación `go build ./...`, `go vet ./...`, `go test ./...` sin la bandera `-run`, y `e2e/e2e_test.sh`, todos en verde. Ninguna ejecución filtrada por patrón (`-run <patrón>`) ni la omisión de `e2e/e2e_test.sh` DEBE aceptarse como evidencia de cierre de una tanda.
(Previously: la cláusula que exigía declarar en el registro que `bench/` queda fuera de la cobertura de la suite raíz; el módulo `bench/` se retira (C-01-e, P21).)

#### Scenario: Verificación completa aceptada
- **DADO** una tanda que completó su absorción
- **CUANDO** se registra su evidencia de verificación
- **ENTONCES** el registro cita `go build ./...`, `go vet ./...`, `go test ./...` sin `-run`, y `e2e/e2e_test.sh`, todos en verde

#### Scenario: Verificación filtrada rechazada
- **DADO** una tanda cuya evidencia de verificación cita `go test -run <patrón>` o no incluye `e2e/e2e_test.sh`
- **CUANDO** se evalúa el cierre de la tanda
- **ENTONCES** la tanda no se acepta como verificada
- **Y** no se marca `absorbido` en el registro para ninguno de sus commits

### Requirement: Rutas protegidas fuera de alcance (REQ-20.4)

Ninguna tanda de absorción DEBE contener ficheros bajo `internal/hub/`, `openspec/INDEX.md` u `openspec/config.yaml`. Un diff que toque cualquiera de esas rutas DEBE rechazarse y detener el incremento hasta el último estado verde.
(Previously: el segundo párrafo prohibía ficheros bajo `bench/` salvo una tanda de reconciliación del corpus; el módulo `bench/` se retira (C-01-e, P21).)

#### Scenario: Diff que toca una ruta protegida rechazado
- **DADO** una tanda cuyo diff incluye un fichero bajo `internal/hub/`
- **CUANDO** se evalúa la tanda para su cierre
- **ENTONCES** se rechaza
- **Y** se revierte hasta el último estado verde
