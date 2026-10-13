# ODD: pruebas de Windows sin ruido de entorno y con ejecución selectiva en CI

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/windows-pruebas-entorno-y-ci.md`, en la rama `fix/windows-pruebas-entorno-y-ci` (worktree `C:\repos\axiom-wt\windows-pruebas-entorno-y-ci`, creado desde `origin/main` `bd82ed31` el 2026-10-12).
> Espejo de recuperación en Engram: topic `odd/windows-pruebas-entorno-y-ci/tasks`, proyecto `axiom`.
> Abierto en la sesión de cierre de inc-24. **Se ejecuta desde otra sesión.**

## Objetivo

1. **Fallos deterministas de entorno.** Las pruebas que necesitan una capacidad del sistema que la máquina no tiene (enlaces simbólicos sin privilegio, *junctions*, sockets Unix) se omiten con un motivo claro en lugar de fallar. Siguen ejecutándose donde la capacidad existe, como Linux o un runner de Windows con privilegio.
2. **Fallos intermitentes por tiempo.** Los presupuestos de tiempo que fallan con la máquina cargada pasan a ser holgados y configurables, sin reintentos.
3. **Ejecución selectiva de Windows en CI.** Política decidida por el usuario el 2026-10-12: las pruebas de Windows no se ejecutan con cada merge. Se ejecutan solo cuando un cambio toca algo que lo exige, o cuando llevan tiempo sin ejecutarse. Se mantiene la ejecución manual.

## Problema y porqué

- **Ruido local.** Durante inc-24 cada lote arrastró la misma lista de «fallos conocidos de entorno». Mientras exista esa lista, un rojo en Windows no significa nada y un fallo real puede pasar por ambiental.
- **CI sin señal de Windows.** Desde el PR #55 (`1c8951ff`, «detener consumo masivo de minutos»), todos los jobs de Windows solo se lanzan a mano:
  - en `ci.yml`, los jobs `windows-runtime`, `organic-runtime-e2e`, `darwin-runtime` y `e2e-tests` llevan `if: github.event_name == 'workflow_dispatch'`;
  - `windows-full-suite.yml` solo tiene `workflow_dispatch`.
  - La última ejecución de la suite completa fue el 2026-09-26 y falló.
  - El riesgo actual no es el coste, sino que Windows se rompa sin que nadie lo vea. Ya pasó antes del 2026-07-29, según el comentario de cabecera de `windows-full-suite.yml`.
- **Coste.** La suite completa se reparte en unos 10 shards de hasta unos 30 minutos cada uno. Lanzarla en cada merge con muchos merges al día no es asumible.

## Evidencia de partida (sesión de inc-24, Windows 11, git 2.41.0.windows.2)

Estas pruebas fallan siempre en esta máquina, también en la base:

| Paquete | Prueba(s) | Causa observada | Clase |
|---|---|---|---|
| `internal/pathidentity` | 4 pruebas de enlace simbólico | Sin privilegio para crear enlaces simbólicos | Capacidad |
| `internal/releasepolicy` | `TestValidateRunMarkerRejectsAMarkerInsideASymlinkedDist` | Ídem | Capacidad |
| `internal/reviewtransaction` | `TestMaintenanceLockRejectsSymlinksAndStaleBytesAreNotOwnership`, `TestMaintenanceLockRejectsSymlinkedAuthorityComponent` | Ídem | Capacidad |
| `internal/cli` | `TestCodeGraphGuidanceSyncStepRestoresSymlinkAfterInstallerFailure`, `TestCodeGraphGuidanceSyncStepPreservesBrokenSymlinkChain`, `TestRunSyncMigratesLegacyManagedPiCodeGraphSelection` | Enlaces simbólicos (a confirmar el tercero) | Capacidad, a confirmar |
| `internal/components/filemerge` | `TestWriteFileAtomicFollowsDirectoryJunction`, `TestWriteFileAtomicFollowsRelativeDirectoryJunction` | `createtemp ... file already exists` a través de una *junction* | A diagnosticar: puede ser un defecto real y no de entorno |
| `internal/backup` | `TestSnapshotterSkipsUnixSockets` | Sockets Unix en Windows | Capacidad |
| `internal/semantic` | `TestServiceReindexCodeGraph` | Sin diagnosticar | A diagnosticar |
| `internal/opencode` | `CancelsOverflowingChild` | Sin diagnosticar | A diagnosticar |
| `internal/tui` | `TestRuntimeCatalogDiscoveryIgnoresStaleProjectResults` | Sin diagnosticar | A diagnosticar |
| `e2e` | `TestNativeProviderCaptureResultCLIUsesCompiledAdapters` | Sin diagnosticar | A diagnosticar |
| `bench/` | 19 pruebas | git fuera del PATH del sandbox, TTY, limpieza con «Access is denied» | Entorno del arnés |

Estas fallan solo con carga:

| Paquete | Síntoma | Clase |
|---|---|---|
| `internal/sddstatus` | Procesos git que superan el presupuesto de 2 minutos con paquetes en paralelo | Tiempo |
| `e2e/organicruntime` | `TestNativeProviderCaptureFailureReoffersTheSameBinding` con `operation_timeout`; ejecutada sola pasa | Tiempo |

`windows-full-suite.yml` menciona además «dieciocho fallos genuinos de Windows». El inventario de T1 debe cruzarlos con esta tabla.

## Alcance autorizado

- Helper de pruebas para detectar capacidades y su uso en las pruebas de clase «Capacidad».
- Presupuestos de tiempo de las pruebas de clase «Tiempo», configurables y más holgados, sin reintentos.
- Cambios de disparadores en `.github/workflows/ci.yml` (job `windows-runtime`) y en `.github/workflows/windows-full-suite.yml`.
- Documentación de la política: qué se ejecuta en local en Windows y qué decide el CI.
- **Fuera de alcance:**
  - paralelizar `internal/cli`, que tiene 89 llamadas a `os.Setenv`/`t.Setenv` y es una refactorización aparte;
  - arreglar los defectos reales que salgan del diagnóstico: cada uno se anota como hallazgo y se decide aparte;
  - los jobs de macOS y e2e en Docker, salvo que la política nueva lo pida explícitamente.

## Restricciones

- **Nunca ocultar un fallo real.** Una prueba solo se omite si la sonda de capacidad falla en esa máquina. Donde la capacidad existe, la prueba se ejecuta y debe pasar. El motivo de la omisión nombra la capacidad y cómo habilitarla (por ejemplo, el modo desarrollador de Windows).
- **Sin reintentos automáticos.** Los presupuestos de tiempo se ajustan; no se repiten ejecuciones.
- Commits en castellano, Conventional Commits y **sin `Co-Authored-By` ni atribución de IA**.
- `C:\repos\axiom` es el checkout de otra sesión y no se toca.
- En local no se ejecuta la suite completa en Windows: solo paquetes focalizados con `-run`, más el ratchet de código muerto y el de refusals.

## Diagnóstico de T1 (2026-10-12)

Datos de la máquina: Windows 11 Enterprise 10.0.26200, Go 1.26.3 y modo desarrollador desactivado.

- `os.Symlink` falla con el error 1314 (`ERROR_PRIVILEGE_NOT_HELD`).
- `mklink /J` funciona.
- `net.Listen("unix")` funciona.
- `codegraph` está en el PATH.

| Paquete | Prueba(s) | Resultado y causa confirmada | Clase | Destino |
|---|---|---|---|---|
| `internal/pathidentity` | `TestSameDirectoryAcceptsTwoSpellingsOfOneDirectory`, `TestContainsAcceptsAPathAddressedThroughAnAliasedAncestor`, `TestContainsAcceptsTheRootItself`, `TestContainsAcceptsADescendantThatDoesNotExistYet` | Error 1314 en el helper compartido `aliasedRoots` (`identity_test.go:25`) | Capacidad | T2 |
| `internal/releasepolicy` | `TestValidateRunMarkerRejectsAMarkerInsideASymlinkedDist` | Error 1314 (`run_marker_identity_test.go:24`) | Capacidad | T2 |
| `internal/reviewtransaction` | `TestMaintenanceLockRejectsSymlinksAndStaleBytesAreNotOwnership`, `TestMaintenanceLockRejectsSymlinkedAuthorityComponent` | Error 1314 (`maintenance_lock_test.go:126`, `:330`) | Capacidad | T2 |
| `internal/cli` | `TestCodeGraphGuidanceSyncStepRestoresSymlinkAfterInstallerFailure`, `TestCodeGraphGuidanceSyncStepPreservesBrokenSymlinkChain` | Error 1314 (`sync_test.go:2265`, `:2329`) | Capacidad | T2 |
| `internal/cli` | `TestRunSyncMigratesLegacyManagedPiCodeGraphSelection` | No usa enlaces simbólicos. Ejecuta el `codegraph install` real porque no sustituye `cmdLookPath` (`sync_test.go:2774`; la prueba hermana de `:2755` sí lo hace). Pasa sin `codegraph` en el PATH. | Defecto de aislamiento de la prueba | Hallazgo H1 |
| `internal/components/filemerge` | `TestWriteFileAtomicFollowsDirectoryJunction`, `TestWriteFileAtomicFollowsRelativeDirectoryJunction` | La *junction* se crea bien. Falla `os.CreateTemp` (`O_CREATE\|O_EXCL`) a través de ella, y solo bajo `...\AppData\Local\...`. Con `TMP=TEMP=C:\jntmp`, pasan. | Entorno del arnés (TEMP) | Documentar en T5; hallazgo H4 |
| `internal/backup` | `TestSnapshotterSkipsUnixSockets` | `net.Listen("unix")` funciona, pero `os.Lstat` del socket falla con `ERROR_CANT_ACCESS_FILE`, solo bajo `AppData\Local`. Pasa con TEMP fuera de esa ruta. | Entorno del arnés (TEMP) | Documentar en T5 |
| `internal/semantic` | `TestServiceReindexCodeGraph` | `service.go:260` llama a `exec.LookPath("codegraph")` sin pasar por el `Detector` inyectado y ejecuta el `codegraph index` real | Defecto de aislamiento | Hallazgo H2 |
| `internal/opencode` | `TestRunCatalogCommandCancelsOverflowingChild` | Falla 4 de 4 veces: 1,1–1,4 s frente a un presupuesto de 1 s. El reloj incluye el primer `exec` del binario recién compilado (`catalog_test.go:118-141`). | Tiempo | T3 |
| `internal/tui` | `TestRuntimeCatalogDiscoveryIgnoresStaleProjectResults` | Lee el `opencode.json` real del usuario, que trae el proveedor `litellm` (`model.go:5397`). La prueba solo aísla `modelPickerSettingsPath`. | Defecto de aislamiento | Hallazgo H3 |
| `e2e/organicruntime` | `TestNativeProviderCaptureResultCLIUsesCompiledAdapters` | Pasa sola (97 s); con el paquete entero, agota el tiempo | Tiempo | T3 |
| `e2e/organicruntime` | `TestNativeProviderCaptureFailureReoffersTheSameBinding` | Falla incluso sola: el subtest codex muere a los 118 s con la salida vacía, señal del `organicLocalTimeout` de 90 s (inferido) | Tiempo | T3 |
| `internal/sddstatus` | Procesos git con carga | Sin carga, el paquete pasa en 169 s. El presupuesto es `reviewtransaction.LocalGitCommandTimeout = 120s` (`snapshot.go:1899`). | Tiempo | T3 |
| `bench/` (módulo aparte) | 19 pruebas | Cuatro causas: falsos binarios `#!/bin/sh` que Windows no ejecuta; TTY/ConPTY; «Access is denied» al limpiar un `.exe` bloqueado; y git o PATH. El CI solo ejecuta `bench` en Ubuntu (`ci.yml:99-103`). | Entorno del arnés y capacidad POSIX | Fuera de alcance: se documenta en T5 que `bench` no se ejecuta en Windows |

**Cruce con CI.** El comentario de `windows-full-suite.yml` no enumera los dieciocho fallos. La única ejecución disponible, la 36278046574 del 2026-09-26, muestra 107 FAIL en 9 paquetes, la mayoría por la deriva del renombrado gentle-ai→axiom. Muchos ya se corrigieron en `main` (#60, #63, #70).

- Ninguna prueba de la tabla local falló en ese runner. Se infiere que el runner tiene privilegio de enlaces simbólicos y un TEMP sin el problema de `AppData\Local`.
- Para tener una línea base fiable hace falta una ejecución nueva de `windows-full-suite`, que lanza el usuario. De esa ejecución salen fallos que podrían ser defectos reales: H5.

**Hallazgos (no se arreglan en este ODD; se deciden aparte):**

- **H1** `internal/cli`: `TestRunSyncMigratesLegacyManagedPiCodeGraphSelection` depende de que no haya `codegraph` en el PATH. Corrección sugerida: sustituir `cmdLookPath` como hace la prueba hermana.
- **H2** `internal/semantic`: `Service.ReindexCodeGraph` ignora el `Detector` inyectado (`service.go:260`). Es un defecto de diseño, y por eso la prueba depende del host.
- **H3** `internal/tui`: `TestRuntimeCatalogDiscoveryIgnoresStaleProjectResults` lee la configuración real de OpenCode. Corrección sugerida: aislar `USERPROFILE`, `HOME` y `OPENCODE_CONFIG_DIR`.
- **H4** `internal/components/filemerge`: `ensureAtomicParentDir` (`writer.go:300-345`) resuelve la *junction*, pero descarta la ruta resuelta, y `replaceDurably` crea el temporal a través de la *junction*. No es la causa del fallo, pero crear el temporal en la ruta resuelta evitaría `O_EXCL` sobre un punto de reanálisis. Es un endurecimiento opcional del producto.
- **H5** CI (2026-09-26): `edit_authority_missing` con rutas `C:\Users\runneradmin\...` en tres pruebas de `e2e/organicruntime` (`TestSDDEditAuthorityConsentGrantLoop`, `TestSDDPreparationInvocationRoundTripsSpacedChange`, `TestSDDSameParentRepositoryConsentGrantLoop`), en `internal/sddstatus` (`TestCharacterization_EditAuthorityMissingBlocksApplyAndArchive`) y en `internal/cli` (`TestSDDStatusDoesNotPrepareConsentMarker`). Puede ser un defecto real de rutas Windows; hay que confirmarlo con una ejecución nueva.
- **H6** `internal/opencode`: `TestRunCatalogCommandDeadlineNotBlockedByInheritingDescendantWindows` falla con carga al limpiar el directorio temporal. Un nieto sigue dormido 30 s y mantiene bloqueado `descendant-helper.exe`. Es la misma clase de limpieza de `.exe` bloqueado que en `bench`; hace falta una limpieza tolerante, no un presupuesto.
- **H7** `e2e/organicruntime`: con carga, el presupuesto de producción de 25 s del binario hijo (`reviewFacadeOperationTimeout`, `internal/cli/review_facade.go:493`) agota `review status` con `operation_timeout`. Las pruebas no pueden escalarlo. Arreglarlo exige una decisión de producción: una costura por entorno o un presupuesto más holgado.

## Política de CI propuesta (T4) — confirmar con el usuario antes de implementarla

Se aplica a `windows-runtime` (`ci.yml`) y a `windows-full-suite.yml`:

1. **Manual:** se mantiene `workflow_dispatch`.
2. **Por cambio relevante:** en `pull_request` y en `push` a `main`, solo cuando el diff toca ficheros que existen únicamente para Windows o que deciden su comportamiento:
   - `**/*_windows.go`, `**/*_windows_test.go`, y sus pares `**/*_other.go` y `**/*_unix.go`;
   - `scripts/*.ps1`;
   - los propios ficheros de workflow;
   - `go.mod` y `go.sum`.
   - No se filtra por paquete: 46 paquetes tienen código específico de Windows, entre ellos `internal/cli/run.go` y `sync.go`, que cambian a menudo, así que filtrar por paquete equivaldría a ejecutar casi siempre.
3. **Forzado manual en un PR:** una etiqueta (por ejemplo `ci:windows`) que lanza las pruebas de Windows aunque el diff no toque esas rutas.
4. **Por antigüedad:** un `schedule` semanal (propuesta: lunes 03:20 UTC) que solo ejecuta si `main` tiene commits nuevos desde la última ejecución correcta. Si no hay nada nuevo, sale enseguida.

Coste asumido: un cambio que rompa Windows sin tocar las rutas del punto 2 se detecta, como mucho, en la ejecución semanal o al forzarla. Linux sigue ejecutando la suite completa en cada PR, y eso cubre los ratchets y contratos entre paquetes que preocupaban en el comentario de `windows-full-suite.yml`.

**Decisiones para el usuario:**

- (a) La lista de rutas del punto 2.
- (b) La periodicidad del punto 4.
- (c) Si `windows-full-suite` también se dispara por cambio relevante o solo por antigüedad y a mano. Por coste, se propone que solo `windows-runtime` reaccione a los cambios.

**Decididas por el usuario el 2026-10-12:**

- (a) La lista propuesta, sin filtrar por paquete.
- (b) Semanal, los lunes a las 03:20 UTC, solo si hay commits nuevos.
- (c) `windows-full-suite` se lanza solo por antigüedad, a mano o con la etiqueta. Solo `windows-runtime` reacciona a los cambios.

## Modo de trabajo

- **TDD:** modo estándar. Fuente: `openspec/config.yaml` (`strict_tdd: false`). Runner: `go test`.
- **RDD:** `off (decided by global)` el 2026-10-12.
- **Entrega:** previsión de 400–700 líneas de autoría: el helper, la edición de unas 15 pruebas, los workflows y la documentación. Estrategia `ask-on-risk`: al pasar de unas 400 líneas se pregunta una vez si se encadenan PRs. Corte natural: PR 1 con T1 a T3 (pruebas) y PR 2 con T4 y T5 (CI y documentación).

## Tareas

- [x] **T1 · Inventario y diagnóstico**
  - Ruta: delegada (dos mapeos en paralelo: diagnóstico local, y cruce con CI y presupuestos de tiempo). Disparador: mapeo de más de 4 ficheros.
  - Resultado en la sección «Diagnóstico de T1». Sin commit de código; solo se actualiza este documento.
- [x] **T2 · Helper de capacidades y omisión motivada** — `87fd84ff`, 190 líneas.
  - Evidencia:
    - `go test ./internal/testenv`: PASS.
    - Las 9 pruebas: SKIP en esta máquina, con el motivo «symlink creation requires privilege on Windows (error 1314); enable Windows Developer Mode or run as administrator…».
    - En WSL (Ubuntu, binarios compilados para Linux con `TMPDIR=/tmp`) las 9 pruebas y `testenv` pasan sin omitirse.
    - Ratchet de código muerto: sin cambios, porque `testenv` solo lo importan pruebas.
  - Verificación: riesgo `high` (señal `process_boundary` en `timeout.go`). El verificador independiente no encontró bloqueos y la comprobación puntual del padre coincide.
  - Ruta: delegada (writer único; varios ficheros).
  - Alcance ajustado por T1: solo `RequireSymlink(t)`. Las *junctions* y los sockets Unix funcionan en esta máquina; sus fallos son del arnés (ver T1), así que no se crean sondas para ellos.
  - Ubicación: paquete hoja nuevo `internal/testenv` (no hay helpers de pruebas compartidos en el repo). Se reutiliza la detección del error 1314 de `isSymlinkPrivilegeError` (`internal/components/filemerge/writer_test.go`).
  - Aplicarlo solo a las 9 pruebas clasificadas como Capacidad en T1.
  - Probar el propio helper: debe omitir solo cuando la sonda falla.
  - Comprobación: en esta máquina, cada una de esas pruebas pasa a SKIP con un motivo claro; en Linux (WSL, `GOOS=linux go test -c`) se ejecutan y pasan.
  - Commit: `test: omitir con motivo las pruebas que requieren capacidades del sistema no disponibles`.
- [ ] **T3 · Presupuestos de tiempo de las pruebas** — PARCIAL. `b245fc3a`, 221 líneas.
  - Hecho:
    - `AXIOM_TEST_TIMEOUT_SCALE`: número mayor o igual que 1. Un valor inválido aborta con un mensaje; nunca se reduce un presupuesto. Por defecto vale 2 en Windows y 1 en el resto.
    - Se aplica en tres sitios: el `TestMain` de `internal/sddstatus`, las variables de tiempo de `e2e/organicruntime` y `TestRunCatalogCommandCancelsOverflowingChild`. En esta última se añade además un `exec` de calentamiento antes del reloj, porque la medición era coste de arranque en frío. La prueba sigue demostrando la cancelación.
  - Evidencia por separado:
    - opencode: 3/3 PASS.
    - Las dos pruebas de organicruntime: PASS en 233 s.
    - sddstatus: ok en 75 s.
  - Con carga (`go test -p 4`, dos ejecuciones): sddstatus pasa siempre. Hubo un fallo distinto en cada ejecución, ambos fuera del alcance de los presupuestos de las pruebas: H7 en la 1 y H6 en la 2.
  - Fuera del alcance de las pruebas: `StagingCommandTimeoutFloor` se inicializa al cargar el paquete desde `LocalGitCommandTimeout` y no se escala. Se añadirá solo si aparece un agotamiento de *staging* en sddstatus.
  - Ruta: delegada.
  - Presupuestos localizados en T1:
    - `reviewtransaction.LocalGitCommandTimeout` y `RemoteGitCommandTimeout`: variables de producción que ya son una costura de pruebas. Se escalan en el `TestMain` de `internal/sddstatus`.
    - `organicLocalTimeout` y sus hermanas en `e2e/organicruntime`.
    - El presupuesto de 1 s de `TestRunCatalogCommandCancelsOverflowingChild` en `internal/opencode`.
  - Hacerlos configurables con una variable de entorno de escala, por ejemplo `AXIOM_TEST_TIMEOUT_SCALE`, y con un valor por defecto más holgado en Windows si la medición lo justifica.
  - El presupuesto de producción no cambia.
  - Comprobación: los paquetes afectados pasan con la máquina cargada (`go test -p 4`).
  - Commit: `test: presupuestos de tiempo de las pruebas escalables y holgados con carga`.
- [ ] **T4 · Ejecución selectiva de Windows en CI**
  - Ruta: en línea o delegada según tamaño. **Requiere antes las decisiones (a) a (c) del usuario.**
  - Implementar la política en `ci.yml` (`windows-runtime`) y en `windows-full-suite.yml`.
  - Los filtros de rutas a nivel de job necesitan detectar los cambios dentro del workflow, sin depender de acciones de terceros no fijadas.
  - El `schedule` compara el SHA de `main` con el de la última ejecución correcta (`gh run list`) y termina sin ejecutar si no hay commits nuevos.
  - Actualizar el comentario de cabecera de `windows-full-suite.yml`.
  - Comprobación: validar la sintaxis (por ejemplo, `actionlint` si está disponible) y una ejecución manual de prueba con `workflow_dispatch`. El usuario autoriza el push.
  - Commit: `ci: ejecutar las pruebas de Windows solo por cambio relevante, etiqueta o antigüedad`.
- [ ] **T5 · Documentación de la política**
  - Ruta: en línea.
  - En `docs/platforms.md`, o en la guía de contribución si existe, qué se ejecuta en local en Windows (paquetes focalizados), cómo forzar Windows en un PR y cada cuánto se ejecuta solo.
  - Commit: `docs: política de pruebas en Windows`.
- [ ] **T6 · Cierre**
  - `go build ./...`, `go vet ./...`, `go run ./internal/gofmtcheck`, `bash scripts/deadcode-ratchet.sh`, los paquetes tocados con `-count=1` y `git status --porcelain` vacío.
  - Actualizar este documento y el espejo.
  - Preparar los PRs. Push, PR y merge los decide el usuario.

## Criterios de aceptación

- En esta máquina, la lista de fallos de entorno conocidos queda vacía. Cada prueba de clase Capacidad aparece como SKIP con un motivo, y cada fallo restante está clasificado como Tiempo resuelto o como Defecto real anotado.
- En Linux y en un runner con la capacidad, esas pruebas se ejecutan y pasan.
- Ninguna prueba se omite sin una sonda que lo justifique.
- Las pruebas de Windows en CI no se lanzan en un PR o merge que no toque las rutas acordadas, se lanzan cuando sí las toca o con la etiqueta, y hay una ejecución periódica que solo trabaja si hubo cambios.

## Progreso

- 2026-10-12: documento abierto con 6 tareas. T4 espera las decisiones (a) a (c) del usuario. Pendiente de ejecución desde otra sesión.
- 2026-10-12: T1 cerrada. Resultado: 9 pruebas de Capacidad (todas por enlaces simbólicos), 4 de Tiempo, 3 de entorno por la ruta de TEMP, 3 con defecto de aislamiento (H1–H3) y `bench` fuera de alcance. Hallazgos H1–H5 anotados. El alcance de T2 se reduce a `RequireSymlink`.
- 2026-10-12: T2 hecha (`87fd84ff`). T3 parcial (`b245fc3a`): con carga quedan H6 y H7, que necesitan una decisión aparte. Líneas acumuladas: unas 410 (umbral de `ask-on-risk` alcanzado; corte natural previsto: PR 1 con T1–T3).

- 2026-10-12: el usuario decide (a) a (c) y la estrategia de cadena `stacked-to-main`.
  - PR 1: rama `fix/windows-pruebas-entorno-y-ci`, con T1–T3 y el documento ODD. Base: `main`.
  - PR 2: rama `ci/windows-ejecucion-selectiva`, apilada sobre el PR 1, con T4–T5. Se rebasa sobre `main` cuando se fusione el PR 1.
  - H6 y H7 quedan anotados como defectos (instrucción del usuario: no arreglar defectos reales). T3 se mantiene parcial por ellos.

## Siguiente paso

T4 en la rama `ci/windows-ejecucion-selectiva`.
