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

## Modo de trabajo

- **TDD:** modo estándar. Fuente: `openspec/config.yaml` (`strict_tdd: false`). Runner: `go test`.
- **RDD:** `off (decided by global)` el 2026-10-12.
- **Entrega:** previsión de 400–700 líneas de autoría: el helper, la edición de unas 15 pruebas, los workflows y la documentación. Estrategia `ask-on-risk`: al pasar de unas 400 líneas se pregunta una vez si se encadenan PRs. Corte natural: PR 1 con T1 a T3 (pruebas) y PR 2 con T4 y T5 (CI y documentación).

## Tareas

- [ ] **T1 · Inventario y diagnóstico**
  - Ruta: delegada (mapeo de más de 4 ficheros).
  - Confirmar la causa de cada fila de las tablas, sobre todo las marcadas «a diagnosticar», y cruzarlas con los 18 fallos genuinos del comentario del workflow.
  - Clasificar cada fallo como **Capacidad**, **Tiempo** o **Defecto real**. Los defectos reales se anotan aquí como hallazgos y no se arreglan en este ODD.
  - Sin commit de código; se actualiza este documento.
- [ ] **T2 · Helper de capacidades y omisión motivada**
  - Ruta: delegada (writer único; varios ficheros).
  - Helper compartido de pruebas (ubicación a decidir según los helpers existentes del repo) con sondas cacheadas por proceso: `RequireSymlink(t)`, `RequireDirJunction(t)` si aplica y `RequireUnixSocket(t)`.
  - Aplicarlo solo a las pruebas clasificadas como Capacidad en T1.
  - Probar el propio helper: debe omitir solo cuando la sonda falla.
  - Comprobación: en esta máquina, cada una de esas pruebas pasa a SKIP con un motivo claro; en Linux (WSL, `GOOS=linux go test -c`) se ejecutan y pasan.
  - Commit: `test: omitir con motivo las pruebas que requieren capacidades del sistema no disponibles`.
- [ ] **T3 · Presupuestos de tiempo de las pruebas**
  - Ruta: delegada.
  - Localizar los presupuestos de las pruebas de clase Tiempo, empezando por el de git de `internal/sddstatus` y el de `operation_timeout` de `e2e/organicruntime`.
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

## Siguiente paso

T1.
