# ODD: Adelgazar los e2e de la CI

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/e2e-adelgazar.md`.
> Espejo de recuperación en Engram: topic `odd/e2e-adelgazar/tasks`, proyecto `axiom`.
> Rama `refactor/e2e-adelgazar`, worktree `C:\repos\axiom-wt\e2e-adelgazar\axiom`.

## Objetivo

Retirar la suite e2e en contenedores Docker, que prueba la superficie del instalador heredado de gentle-ai, y dejar Organic Runtime E2E en verde y con señal periódica en la ejecución nocturna.

## Problema

- Los jobs `E2E Tests (ubuntu|arch|fedora)`, `Organic Runtime E2E` y `Darwin Runtime` solo corrían a mano (`workflow_dispatch`, y desde #327 solo con `full=true`). Las tres ejecuciones manuales del historial (10-nov ×2 y 26-nov) fallaron y nadie lo veía.
- La suite Docker (`e2e/e2e_test.sh`, `e2e/lib.sh`, `e2e/docker-test.sh`, tres Dockerfiles) pide componentes y agentes retirados: `unsupported component "theme"`, preset `full-gentleman` con `claude-theme`, Qwen, Antigravity y el plugin `model-variants`. El 10-nov ni llegó a ejecutarse: la imagen de Ubuntu no se construyó (apt 404).
- Organic Runtime falla en un solo subtest, `TestOrganicConfiguredAgentReceivesRoutingGuidanceRealAgents/orchestrator_prompt`: espera la guía de OpenCode en `~/.config/opencode/opencode.json`, pero con `--scope workspace` el producto la escribe ahora en el `opencode.json` del proyecto y deja la carpeta de usuario intacta (contrato de `TestInstallRoutingGuidanceWorkspaceScopeDeliversOpenCodeToHome` en `internal/cli/run_component_paths_test.go`).

## Por qué

Axiom se está adelgazando. La suite Docker prueba una superficie que se retira, nadie la actualizó en los retiros de agentes y componentes, y probar un binario estático de Go en tres distribuciones aporta poco. Organic Runtime sí prueba lo central (la guía de enrutado llega a agentes reales y el ciclo de consentimiento de SDD), pero sin ejecución periódica se pudre.

## Alcance autorizado

- Borrar `e2e/e2e_test.sh`, `e2e/lib.sh`, `e2e/docker-test.sh`, `e2e/Dockerfile.{ubuntu,arch,fedora}` y `docs/docker-e2e-testing.md`.
- Quitar el job `e2e-tests` de `.github/workflows/ci.yml` y las pruebas de `internal/assets` que lo describen o leen sus Dockerfiles.
- Actualizar la documentación viva que los cita: `.github/PULL_REQUEST_TEMPLATE.md`, `docs/architecture.md`, `docs/testing-agents-deterministically.md`, `docs/release-signing.md`.
- Corregir la expectativa obsoleta del subtest de OpenCode en `e2e/organicruntime`.
- Ejecutar `organic-runtime-e2e` también en la nocturna (`schedule`), además de a mano con `full=true`.

Fuera de alcance: `Darwin Runtime` (fallo de red, no de prueba), `internal/cli/*_e2e_test.go` (corren en `Go Tests`), documentos históricos (`docs/audits/`, `docs/archive/`, `docs/plan/`, `docs/releases/`, `docs/upstream-absorption-ledger.md`, `odd/`, `openspec/changes/archive/`).

## Restricciones

- Sin compilar binarios sueltos en local (Cortex XDR): Organic Runtime construye `axiom` y necesita OpenCode real, así que se valida en CI.
- Pruebas locales localizadas: solo paquetes tocados y `-run`.
- TDD: desactivado (ni `axiom.yaml` ni la sesión lo configuran); checks funcionales con `go test`.

## Tareas

- [ ] **T1 · Retirar la suite Docker e2e** — ruta: inline (borrado mecánico y ediciones pequeñas; sin disparadores de delegación).
  - Borrar ficheros, el job `e2e-tests`, `TestArchOpenCodeProvisioningRunsPublishedRepairBeforeVersionCheck` y la entrada `e2e-tests` de la tabla de jobs de `formatter_ordering_test.go`; actualizar la documentación viva.
  - Checks: `go test ./internal/assets -run 'TestFormatter|TestOrganicRuntimeE2EUsesInstalledOpenCodePin|TestCI'`; búsqueda de referencias vivas a los ficheros borrados.
- [ ] **T2 · Corregir el subtest de OpenCode en Organic Runtime** — ruta: inline (una prueba de unas 20 líneas).
  - Esperar la guía en `<proyecto>/opencode.json` y comprobar que `~/.config/opencode/opencode.json` no existe.
  - Checks: `go vet -tags real_agent_e2e ./e2e/organicruntime`; la ejecución real, en CI.
- [ ] **T3 · Organic Runtime en la nocturna** — ruta: inline (cambio de `if:` y su prueba).
  - `if:` acepta `schedule` además de `workflow_dispatch && inputs.full`; actualizar `formatter_ordering_test.go` y la documentación (`docs/release-signing.md`, `docs/testing-agents-deterministically.md`).
  - Checks: `go test ./internal/assets -run TestFormatter`.

## Criterios de aceptación

- No queda ninguna referencia viva a la suite Docker fuera de documentos históricos.
- `go test` de `internal/assets` pasa con el job retirado y el nuevo `if:`.
- `go vet -tags real_agent_e2e ./e2e/organicruntime` compila el subtest corregido.
- `organic-runtime-e2e` corre en la nocturna cuando `main` tiene commits sin cubrir y a mano con `full=true`.

## Spec viva

### Capacidad: upstream-absorption-protocol

```openspec
## MODIFIED Requirements

### Requirement: Verificación sin filtrar (REQ-20.2)

Toda tanda de absorción DEBE presentar como evidencia de verificación `go build ./...`, `go vet ./...` y `go test ./...` sin la bandera `-run`, todos en verde. Ninguna ejecución filtrada por patrón (`-run <patrón>`) DEBE aceptarse como evidencia de cierre de una tanda.
(Previously: exigía además `e2e/e2e_test.sh` en verde, que se retira con la suite e2e en contenedores; antes, la cláusula que exigía declarar que `bench/` quedaba fuera de la suite raíz, retirada con el módulo `bench/` (C-01-e, P21).)

#### Scenario: Verificación completa aceptada
- **DADO** una tanda que completó su absorción
- **CUANDO** se registra su evidencia de verificación
- **ENTONCES** el registro cita `go build ./...`, `go vet ./...` y `go test ./...` sin `-run`, todos en verde

#### Scenario: Verificación filtrada rechazada
- **DADO** una tanda cuya evidencia de verificación cita `go test -run <patrón>`
- **CUANDO** se evalúa el cierre de la tanda
- **ENTONCES** la tanda no se acepta como verificada
- **Y** no se marca `absorbido` en el registro para ninguno de sus commits
```

## Progreso y evidencias

(pendiente)

## Siguiente paso

T1.
