# ODD: RDD apagado de fábrica y sin posibilidad de activarlo (rdd-apagado)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/rdd-apagado.md`. Espejo Engram: `odd/rdd-apagado/tasks`.
> Rama: `inc/rdd-apagado` · worktree `C:\repos\axiom-wt\rdd-apagado\axiom`. Cambio C-01-c del plan maestro (`docs/plan/backlog-2026-11/01-poda-friccion.md`), ola 3; apartados 6.2 y 6.3 del README.

## Objetivo

Que RDD quede apagado para todo el mundo y no se pueda activar: el modo efectivo es siempre `off`, `axiom review mode enable` falla en todos los alcances, y Axiom deja de proyectar prosa de RDD, de instalar los agentes `review-*` y de registrar el hook `axiom review stop-hook`. El código Go queda latente hasta su retirada (C-01-e, ola 16).

## Problema

RDD es opt-in, pero su prosa ocupa los orquestadores y la guía de agentes de todos los runtimes, sus agentes se instalan siempre y Claude registra el hook de parada en `Stop` y `SessionStart`. El selector aparece en la instalación, la TUI y el dashboard. Todo ello es fricción y contexto para una función que no se va a usar.

## Por qué

Poda de fricción (sección 01 del plan): apagar primero, de forma reversible y sin tocar el motor, y retirar el código después en C-01-e.

## Decisiones

- **Alcance (brief de la sesión de coordinación, no se reabre):** puntos 1-5 de la sección C-01-c. `routing.go:77`, `:129`, `:131` y `:152-166` son de este cambio (README 6.3).
- **Etiqueta `rdd_harness` (usuario, 2026-11-18).** Solo un binario compilado con `-tags rdd_harness` admite `enable` y respeta las preferencias guardadas. Bench, e2e y la CI compilan su binario con la etiqueta (una línea en `bench/main_test.go`, `e2e/organicruntime/organic_runtime_test.go` y `.github/workflows/ci.yml`). Los tests Go en proceso usan un punto de inyección sin bandera de CLI, activado en el `TestMain` de cada paquete que ejercita RDD; los tests de este cambio lo desactivan para comprobar el apagado. Motivo: 91 journeys de bench y los e2e activan RDD con `review mode enable` contra el binario, y la CI los ejecuta.
- **`Delegated Verification Gate` (delegada en la sesión):** se conservan las dos secciones (completa y reducida) y sus marcadores, reescritas sin `axiom review assess` ni mención a RDD. El padre deduce el nivel desde el diff: documentación o contenido pasivo → relectura estructural; código ordinario → autoverificación del escritor, más un verificador si el escritor corrió en un modelo pequeño; seguridad, ejecutables o contenido dudoso → además un verificador independiente. Se mantienen los comandos de `## Verification` con `<command>: <observed result>`, la comprobación puntual del padre, `## Known environmental failures`, la regla de exploración y la línea de SDD sin la mención a RDD.
- **`Axiom Provider Defect Handoff` (delegada en la sesión):** se retira entera (sección compartida, marcadores y cabeceras de los 12 orquestadores). No tiene código Go y solo la usa el sobre de consentimiento v3 de RDD; `SDD Edit-Authority Consent Relay` es independiente y se conserva.
- **Compatibilidad:** `axiom review stop-hook` sigue existiendo y, sin la etiqueta, no hace nada y termina con 0 (proyectos antiguos con el hook registrado). Los campos `rdd_mode` y `rdd_mode_recorded_at` del estado se conservan con `omitempty` para que el JSON antiguo siga cargando; se leen pero no se aplican. Los ficheros embebidos de los agentes `review-*` se conservan porque `uninstall` los usa para borrar.
- **Retirada en `sync`:** los agentes `review-*` gestionados ya instalados se borran por nombre exacto, y el hook de parada se retira de `Stop` y `SessionStart` de Claude con el patrón de `telemetry_retire.go`.
- **Fuera de alcance (C-01-e):** borrar el código Go de RDD, los journeys de bench de RDD, los pasos de CI salvo la etiqueta, los ficheros embebidos de los agentes, el plugin de transporte de revisión de OpenCode y las pantallas de la TUI (quedan sin cablear, no borradas).

## Alcance autorizado

- `internal/reviewtransaction/` (punto único `rddModeStatus` y resolutores), paquete o ficheros con la etiqueta `rdd_harness` y el punto de inyección.
- `internal/cli/review_mode.go`, `internal/cli/review_stop_hook.go`, `internal/hub/rddpref.go`.
- `internal/assets/skills/_shared/sdd-orchestrator-sections.md`, `internal/assets/*/sdd-orchestrator.md`, `internal/components/sdd/{boundedreview,orchestrator,inject,telemetry_retire}.go`, `internal/components/agentguidance/{routing,delegation,approval}.go`, overlays de OpenCode y lo que los conecte.
- TUI (`internal/tui/`) y dashboard (`internal/dashboard/`, Go y frontend).
- Tests y goldens afectados en esos paquetes; `TestMain` con el punto de inyección en los paquetes que ejercitan RDD.
- `bench/main_test.go`, `e2e/organicruntime/organic_runtime_test.go`, `.github/workflows/ci.yml`: solo la etiqueta de compilación.
- `## Spec viva` de este documento e integración con `axiom archive integrate --odd rdd-apagado`.

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y no hay elección del usuario. Comprobaciones funcionales ordinarias con `go test` troceado por paquetes o `-run`, sin binarios sueltos de `axiom` (el EDR los borra).

## Tareas

- [x] **T1 · Modo siempre apagado y etiqueta `rdd_harness`** (ruta: delegada, un escritor; disparador: 4+ ficheros no triviales en varios paquetes)
  - Constante de compilación por etiqueta y punto de inyección para tests; `rddModeStatus`, `ResolveRDDMode` y `ResolveRDDModeStrict` devuelven `off`/`default` sin la etiqueta, aunque haya preferencias `on` guardadas (global, proyecto del hub, clon).
  - `TestMain` con el punto de inyección en los paquetes que ejercitan RDD en proceso.
  - Comprobación: `go test ./internal/reviewtransaction/ -run 'RDDMode'` y tests nuevos de apagado con preferencias `on` guardadas.
- [x] **T2 · CLI, hub y stop-hook** (ruta: delegada, mismo escritor)
  - `review mode enable --scope global|project|clone` sale con error «RDD no está disponible» sin escribir nada; `disable` y `status` siguen funcionando; `status` informa `off`.
  - `SetProjectRDD` rechaza `on`; `axiom review stop-hook` es un no-op con código 0.
  - Comprobación: `go test ./internal/cli/ -run 'ReviewMode|StopHook'` y `go test ./internal/hub/`.
- [ ] **T3 · Prosa: orquestadores y guía de agentes** (ruta: delegada, un escritor; disparador: 15+ ficheros)
  - Secciones compartidas: reescritura de `Delegated Verification Gate` (completa y reducida), retirada de `Axiom Provider Defect Handoff`, de la prosa RDD de `Native Checking Contract` y de la mención en `SDD Change Kickoff and Block Gates`; `Review Execution Contract` deja de renderizarse.
  - Los 12 `sdd-orchestrator.md`: sin marcador ni cabecera del handoff; `codex:341`.
  - `routing.go` (`:77`, `:120`, `:122`, `:129`, `:131`, `:152-166`), `delegation.go:48,65`, `approval.go:17`.
  - Tests que fijan la prosa y goldens regenerados con `-update` y revisados por diff.
  - Comprobación: `go test ./internal/components/... ./internal/assets/...`.
- [ ] **T4 · Instalación: agentes `review-*` y hook de parada** (ruta: delegada, un escritor)
  - `writeSubAgents` omite `review-*` (Claude, Cursor, Kiro, Kimi); el overlay de OpenCode deja de declararlos y de expandirlos; `sync` borra los gestionados ya instalados.
  - Claude: no se registra el hook de parada y se retira de `Stop` y `SessionStart` (patrón `telemetry_retire.go`).
  - Test en proceso de `sync --scope workspace` sobre un proyecto limpio: sin `review-*`, sin stop-hook y sin coincidencias de `receipt-driven|Review Execution Contract|Provider Defect Handoff` (sin mayúsculas).
  - Comprobación: `go test ./internal/components/sdd/ -run '<tests nuevos>'` y el paquete.
- [ ] **T5 · TUI y dashboard sin selector** (ruta: delegada, un escritor)
  - TUI: sin pantalla de modo de revisión en la instalación, sin entrada «Revisión formal RDD» en el menú ni en el resumen; goldens regenerados.
  - Dashboard: sin selector en la creación de proyecto ni tarjeta del modo; el endpoint queda latente y rechaza `on`.
  - Comprobación: `go test ./internal/tui/... ./internal/dashboard/...`.
- [ ] **T6 · Bench, e2e y CI con la etiqueta** (ruta: en línea; tres líneas mecánicas)
  - `-tags rdd_harness` en la compilación del binario de `bench/main_test.go`, `e2e/organicruntime/organic_runtime_test.go` y el paso de `ci.yml` que lo compila.
  - Comprobación: relectura del diff; `go vet -tags rdd_harness ./internal/reviewtransaction/`.
- [ ] **T7 · Spec viva y verificación final** (ruta: en línea)
  - MODIFIED REQ-18.2 en `## Spec viva`; `axiom archive integrate --odd rdd-apagado` (en proceso, sin binario suelto).
  - Batería del criterio de terminado.

## Criterio de terminado

- `review mode enable --scope global|project|clone` sale con error que dice que RDD no está disponible.
- `review mode status` informa `off` aunque haya preferencias `on` guardadas.
- `sync --scope workspace` sobre un proyecto limpio no escribe agentes `review-*`, no registra el stop-hook y no deja secciones RDD (`rg -i "receipt-driven|Review Execution Contract|Provider Defect Handoff"` sin coincidencias).
- La TUI y el dashboard no muestran el selector.
- Goldens regenerados y revisados por diff.
- `go test ./internal/components/... ./internal/assets/... ./internal/cli/... ./internal/tui/... ./internal/dashboard/...` en verde (troceado si no cabe).

## Entrega

Una sola PR con etiqueta `type:*` y `size:exception` (estrategia `exception-ok`): la previsión supera las 400 líneas escritas (unas 900-1.300 sin goldens). Push, PR y fusión los decide el usuario. Apertura: `powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/axiom-worktree.ps1 pr rdd-apagado -Lane odd`.

## Progreso

Plan stop: approved 2026-11-18; implementation continues

- **T1** `57a743c0`, **T2** `dc693b24` (escritor delegado). Paquete nuevo `internal/rddharness` (`Enabled`, `SetForTesting`, `UnavailableMessage`); `ResolveRDDMode`/`ResolveRDDModeStrict` cortocircuitan a off/default sin leer la anulación del clon. `TestMain` con el punto de inyección en `reviewtransaction`, `hub`, `cli`, `app`, `dashboard`, `tui`, `sddstatus` y `cmd/axiom`.
  - Observado: `go test ./internal/reviewtransaction/` ok; `./internal/hub/` ok; `./internal/cli/ -run 'ReviewMode|StopHook|WhileSuspended'` ok; lo mismo con `-tags rdd_harness` ok; `go vet -tags rdd_harness` limpio; `dashboard`, `tui`, `sddstatus`, `cmd/axiom`, `components/sdd` ok. Comprobación del padre: `-run 'Suspended'` ok.
  - `go test ./internal/cli/ -run 'Review'` completo agota el tiempo (sin `--- FAIL`); se trocea en la verificación final.
  - Fallo ambiental preexistente: `internal/app` `TestTuiInstallOnThenSyncPreservesAndRefreshesOpenCodeActivation` («OpenCode runtime version unavailable or unsupported») falla igual en la base `bd36e5c2`.

## Spec viva

### Capacidad: rdd-decoupling-v3-stability

```openspec
## MODIFIED Requirements

### Requirement: Independencia Operativa de las Herramientas RDD (REQ-18.2)
RDD está suspendido (C-01-c) hasta su retirada (C-01-e). Las herramientas de revisión (`axiom review ...`) NO DEBEN condicionar la máquina de estados SDD. El modo efectivo de RDD DEBE ser siempre `off`: `axiom review mode enable` DEBE fallar en todos los alcances con un mensaje que indique que RDD no está disponible, y `axiom review mode status` DEBE informar `off` aunque haya preferencias `on` guardadas. Axiom NO DEBE proyectar prosa de RDD en orquestadores ni en la guía de agentes, NO DEBE instalar los agentes `review-*` y NO DEBE registrar el hook `axiom review stop-hook`; `axiom review stop-hook` DEBE terminar sin error y sin efecto para los proyectos que aún lo tengan registrado. Solo un binario compilado con la etiqueta `rdd_harness` (bench, e2e y CI) PUEDE activar RDD.
(Previously: las herramientas `axiom review ...` eran comandos opt-in gobernados por el usuario)

#### Scenario: Activar RDD falla en todos los alcances
- **DADO** el binario `axiom` sin la etiqueta `rdd_harness`
- **CUANDO** el usuario ejecuta `axiom review mode enable --scope global`, `--scope project` o `--scope clone`
- **ENTONCES** la orden termina con error indicando que RDD no está disponible
- **Y** no guarda ninguna preferencia

#### Scenario: Las preferencias guardadas no activan RDD
- **DADO** una preferencia `on` guardada en el ámbito global, en el proyecto del hub o en el clon
- **CUANDO** se ejecuta `axiom review mode status`
- **ENTONCES** el modo efectivo es `off`

#### Scenario: La sincronización no instala RDD
- **DADO** un proyecto limpio
- **CUANDO** se ejecuta `axiom sync --scope workspace`
- **ENTONCES** no se escriben agentes `review-*`, no se registra el hook de parada de revisión y los ficheros gestionados no contienen prosa de RDD

#### Scenario: El hook de parada antiguo no falla
- **DADO** un proyecto que aún registra `axiom review stop-hook`
- **CUANDO** el runtime lo invoca
- **ENTONCES** termina con código 0 sin efecto

#### Scenario: Las herramientas de revisión no alteran las fases SDD
- **DADO** el binario `axiom`
- **CUANDO** el usuario ejecuta comandos bajo `axiom review`
- **ENTONCES** las fases SDD del workspace no cambian
```
