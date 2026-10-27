# ODD: aislamiento del host en tres pruebas de `cli`, `tui` y `opencode` (H1, H3 y H6)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/pruebas-aislamiento-host.md`, en el repositorio `axiom`. La rama y el worktree dedicados se crean al iniciar la ejecución, nunca en `C:\repos\axiom`.
> Espejo de recuperación en Engram: topic `odd/pruebas-aislamiento-host/tasks`, proyecto `axiom` (creado el 2026-10-13 en estado propuesto; se actualiza al iniciar la ejecución).
> **Estado: Listo para PR (2026-10-27). T1, T2 y T4 hechas; T3 fuera del PR por el defecto de producción H6-D (ver «Hallazgos»).** Rama `fix/pruebas-aislamiento-host`, worktree `C:\repos\axiom-wt\pruebas-aislamiento-host`.
> Origen: hallazgos H1, H3 y H6 del ODD `windows-pruebas-entorno-y-ci` (`odd/tasks/windows-pruebas-entorno-y-ci.md`).

## Objetivo

Que tres pruebas no dependan de lo que haya instalado o configurado en la máquina que las ejecuta, ni de la carga de esa máquina. Son correcciones **solo de pruebas**, en un único PR. Ninguna toca código de producción.

## Problema y porqué

Las tres fallan, o pueden fallar, según el host. Un rojo que depende del host se confunde con un rojo real, y vuelve a crear la lista de «fallos conocidos de entorno» que el ODD de origen quería eliminar.

- **H1.** `TestRunSyncMigratesLegacyManagedPiCodeGraphSelection` (`internal/cli/sync_test.go:2777`) ejecuta el `codegraph install` real si `codegraph` está en el PATH.
  - No sustituye `cmdLookPath`. El paso `codeGraphGuidanceSyncStep.Run` (`internal/cli/sync.go:1114` y `:1125`) lo consulta con `communitytool.DetectorFunc(cmdLookPath)`.
  - La prueba hermana `TestRunSyncMigratesLegacyManagedCodeGraphSelection` (`sync_test.go:2749`) sí lo sustituye. Esta prueba solo sustituye `refreshPiCodeGraphIfConfigured`.
  - Pasa sin `codegraph` en el PATH y falla donde está instalado.
- **H3.** `TestRuntimeCatalogDiscoveryIgnoresStaleProjectResults` (`internal/tui/model_test.go:8307`) lee el `opencode.json` real del usuario.
  - `model.go:5397` llama a `opencode.ResolveEffectiveConfig(projectDir)`. La prueba solo sustituye `modelPickerSettingsPath` (`model_test.go:8326`).
  - `ResolveEffectiveConfig` resuelve el directorio de inicio con `os.UserHomeDir()` (`internal/opencode/config.go:37`) y acepta `OPENCODE_CONFIG_DIR` si es una ruta absoluta (`config.go:49`).
  - En la máquina de diagnóstico, el proveedor `litellm` del `opencode.json` real contaminaba el resultado.
- **H6.** `TestRunCatalogCommandDeadlineNotBlockedByInheritingDescendantWindows` (`internal/opencode/catalog_process_windows_test.go:19`) falla con carga al limpiar el directorio temporal.
  - Error: `unlinkat ...\descendant-helper.exe: The process cannot access the file because it is being used by another process`.
  - Un nieto duerme 30 s y mantiene bloqueado el ejecutable del ayudante, mientras `t.TempDir()` intenta borrarlo.

## Evidencia

- H1: ver los números de línea de arriba y el diagnóstico de T1 del ODD de origen (`TestRunSyncMigratesLegacyManagedPiCodeGraphSelection`, tabla de diagnóstico). Pasa sin `codegraph` en el PATH.
- H3: ejecución local con el `opencode.json` real del usuario; el proveedor `litellm` entra en el catálogo.
- H6: fallo observado en la segunda ejecución con carga (`go test -p 4`) del paquete `internal/opencode`, en T3 del ODD de origen.
- La cancelación por *Job Object* en producción (`internal/opencode/catalog_process_windows.go`, `configureProcessGroup`, `TerminateJobObject`) está pensada para matar el árbol entero. Que el nieto siga vivo en la limpieza podría deberse al orden de cierre en la prueba o a que la terminación del árbol aún no ha concluido. Hay que averiguar cuál antes de corregir (ver T3).

## Alcance autorizado

El alcance previsto es el de las tres tareas siguientes. **El usuario debe autorizar la implementación al iniciar la sesión.**

- Solo ficheros `_test.go`:
  - `internal/cli/sync_test.go` (H1);
  - `internal/tui/model_test.go` (H3);
  - `internal/opencode/catalog_process_windows_test.go` (H6).
- **Fuera de alcance:** cualquier cambio en código de producción; el resto de pruebas con el mismo defecto de aislamiento, que se anotan como hallazgo y no se arreglan aquí; paralelizar `internal/cli`.

## Restricciones

- Commits en castellano, Conventional Commits, **sin `Co-Authored-By` ni atribución de IA**.
- Trabajar en un worktree dedicado; `C:\repos\axiom` es el checkout de otra sesión y no se toca.
- En Windows, en local solo paquetes focalizados con `-run`, nunca la suite completa. Nunca `go test ./internal/cli/` completo.
- Push, PR y merge solo con permiso del usuario.
- **Sin reintentos** ni bucles de `sleep` y reintento que oculten un fallo. La limpieza de H6 debe ser determinista.
- **Ninguna omisión** (`t.Skip`) para esquivar el problema: las pruebas se aíslan, no se saltan.

## Modo de trabajo

Re-comprobar al iniciar.

- **TDD:** modo estándar. Fuente: `openspec/config.yaml` (`strict_tdd: false`). Runner: `go test`. Aunque el modo sea estándar, cada tarea intenta reproducir el fallo (RED) antes de corregirlo cuando es viable, porque el defecto es justo la dependencia del host.
- **RDD:** releer el estado con `axiom review mode status`. Si está activo, el candidato son cambios solo de pruebas: lo esperable es riesgo pasivo o medio, y se evalúa con `axiom review assess`.
- **Entrega:** estrategia `ask-on-risk`. Previsión orientativa: de 60 a 150 líneas de autoría en total, en un solo PR. No se espera pasar de unas 400.

## Tareas

- [x] **T1 · H1: aislar `cmdLookPath` en la prueba de Pi CodeGraph** — commit `f3b2a9a4`.
  - Ruta: en línea (un fichero, cambio acotado). Si el aislamiento obliga a tocar más de un fichero de forma no trivial, escritor delegado.
  - RED: con `codegraph` en el PATH, ejecutar la prueba y comprobar que lanza el `codegraph install` real (o falla por ello). Anotar la salida.
  - Corrección: sustituir `cmdLookPath` con restauración en `t.Cleanup`, como hace la prueba hermana, o inyectar el ejecutor del paso (`s.runner`). Decidir el valor que conserve el sentido de la prueba (la hermana devuelve «encontrado»; aquí lo que se afirma es la reconciliación de la selección de Pi, y un valor «no encontrado» evita el `ReconcileOpenCodeCodeGraph`).
  - Comprobación: `go test ./internal/cli -run '^TestRunSyncMigratesLegacyManaged(Pi)?CodeGraphSelection$' -count=1`, **con `codegraph` en el PATH y sin él**.
  - Commit: `test(cli): aislar la detección de codegraph en la migración de la selección de Pi`.
  - Evidencia (2026-10-27, Windows):
    - RED: con un `codegraph.cmd` falso en el PATH que no escribe el cableado, la prueba de Pi falla con `sync OpenCode CodeGraph wiring: CodeGraph installer did not create effective OpenCode MCP wiring`. Con el `codegraph` real instalado (npm global) pasaba porque se ejecutaba el `codegraph install` real.
    - Corrección: `cmdLookPath` devuelve `os.ErrNotExist`, restaurado en `t.Cleanup`. «No encontrado» conserva el sentido de la prueba (migración de la selección de Pi) y evita el reconcile de OpenCode.
    - GREEN: las dos pruebas de `^TestRunSyncMigratesLegacyManaged(Pi)?CodeGraphSelection$` pasan con el `codegraph` falso, con el real y sin `codegraph` en el PATH.
- [x] **T2 · H3: aislar la configuración real de OpenCode** — commit `72cba681`.
  - Ruta: en línea (un fichero, cambio acotado).
  - RED: con un `opencode.json` real presente (o un `opencode.json` de prueba con un proveedor ajeno en el directorio de inicio real), comprobar que el proveedor ajeno entra en el catálogo. Anotarlo.
  - Corrección: al principio de la prueba, `t.Setenv` de `USERPROFILE`, `HOME` y `OPENCODE_CONFIG_DIR` apuntando a directorios temporales (la última, una ruta absoluta, porque `config.go:49` ignora las relativas).
  - Comprobación: `go test ./internal/tui -run '^TestRuntimeCatalogDiscoveryIgnoresStaleProjectResults$' -count=1`, con y sin un `opencode.json` real del usuario presente.
  - Commit: `test(tui): aislar el directorio de inicio y la configuración de OpenCode en el descubrimiento de catálogo`.
  - Evidencia (2026-10-27, Windows; esta máquina no tiene `opencode.json` real):
    - RED: con `USERPROFILE` y `HOME` apuntando a un inicio falso con un `opencode.json` que declara `litellm`, la prueba falla con `stale result replaced active catalog: [litellm poison project-b]`.
    - Corrección: `t.Setenv` de `HOME`, `USERPROFILE` y `OPENCODE_CONFIG_DIR` (ruta absoluta temporal), y además `XDG_CONFIG_HOME` vacío, porque `DefaultSettingsPathForHome` (`internal/opencode/models.go:24`) también lo lee.
    - GREEN: pasa con el inicio, `OPENCODE_CONFIG_DIR` y `XDG_CONFIG_HOME` del host contaminados con ese `opencode.json`, y pasa en el entorno limpio.
- [ ] **T3 · H6: limpieza determinista del ayudante en Windows** — **FUERA DE ESTE PR** (decisión del usuario, 2026-10-27): H6-D se acepta como defecto de producción y queda como hallazgo (ver «Hallazgos»). No se cambia la prueba.
  - Ruta: en línea o escritor delegado según el tamaño.
  - Diagnóstico previo, obligatorio: determinar por qué el nieto sigue vivo en la limpieza.
    - Si es el orden de cierre de la prueba (la limpieza se ejecuta antes de que termine la terminación del árbol), se corrige en la prueba.
    - **Si el nieto sigue vivo porque la producción no termina el árbol de procesos tras el plazo, es un defecto real**: se detiene el trabajo, se anota como hallazgo y se pregunta al usuario. No se arregla en este ODD y no se oculta con una limpieza más tolerante.
  - Corrección, solo en la prueba: limpieza determinista. Matar el árbol del ayudante o esperar a que termine, y registrar la limpieza en el orden correcto (`t.Cleanup` se ejecuta en orden inverso al registro; la terminación debe ejecutarse antes del borrado del directorio). Prohibido un bucle de reintentos con `sleep`.
  - Comprobación:
    - `go test ./internal/opencode -run '^TestRunCatalogCommandDeadlineNotBlockedByInheritingDescendantWindows$' -count=1`;
    - bajo carga: `go test -p 4 ./internal/opencode/... -count=1`, varias veces, y comprobar que el directorio temporal se limpia sin error.
  - Commit: `test(opencode): limpieza determinista del ayudante descendiente en la prueba de plazo de Windows`.
- [x] **T4 · Cierre** (2026-10-27, Windows, go1.27.0)
  - Evidencia: `go build ./...`, `go vet ./...` y `go run ./internal/gofmtcheck` en verde. Las pruebas de T1 y T2 pasan con `-count=1`. La de H6 no se ejecuta: T3 sale del PR y su prueba compila un `.exe` vetado por el EDR.
  - `bash scripts/deadcode-ratchet.sh` **no verificable en esta máquina**: `panic: unexpected expr: *ast.KeyValueExpr` dentro de `golang.org/x/tools@v0.30.0/go/ssa`. Falla igual en `main` `e4b4cc3d`, así que es del entorno (herramienta frente a Go 1.27), no de este cambio, que solo toca pruebas.
  - `git diff --stat origin/main...HEAD`: `internal/cli/sync_test.go`, `internal/tui/model_test.go` y este documento. Ningún fichero de producción.
  - `go build ./...`, `go vet ./...`, `go run ./internal/gofmtcheck`, `bash scripts/deadcode-ratchet.sh`, y las tres pruebas con `-count=1`.
  - `git diff --stat origin/main...HEAD` debe mostrar solo ficheros `_test.go`.
  - `git status --porcelain` vacío. Actualizar este documento y el espejo.
  - Preparar el PR (un solo PR). Push, PR y merge los decide el usuario.

## Hallazgos

- **H6-D (2026-10-27): el nieto puede escapar del Job Object.** Diagnóstico estático, sin ejecutar la prueba (ver «Restricción del EDR»):
  - En `runCatalogCommand` (`internal/opencode/catalog.go:226`), el hijo se asigna al Job en `afterStart`, **después** de `cmd.Start()` (`catalog_process_windows.go:56-69`). El ayudante lanza al nieto como primera acción. Si el nieto nace antes de la asignación, no hereda el Job.
  - En la prueba, el hijo sale a los ~25 ms, antes del plazo de 300 ms. `exec.Cmd.watchCtx` entrega el resultado al terminar el proceso y **no llama a `cmd.Cancel`**: `TerminateJobObject` no se ejecuta. El plazo lo acota `WaitDelay` (5 s), no la cancelación. El comentario de la prueba atribuye el límite a la terminación del Job, y en este camino no es así.
  - Al nieto solo lo mata `release()` (`JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`), y solo si está en el Job.
  - `t.TempDir()` ya reintenta 2 s ante `ERROR_SHARING_VIOLATION` (`testing.go:1649`, `testing_windows.go:30`). Que la limpieza falle implica que el ejecutable sigue bloqueado más de 2 s: encaja con un nieto fuera del Job durmiendo 30 s, no con el orden de cierre de la prueba.
  - Conclusión: según la regla de T3, es un defecto real (carrera entre `Start` y `AssignProcessToJobObject`). No se arregla en este ODD ni se oculta con una limpieza más tolerante. Una corrección de producción típica sería crear el proceso suspendido, asignarlo al Job y reanudarlo.
  - Decisión del usuario (2026-10-27): se acepta como defecto de producción; T3 sale de este PR. La corrección de producción, y con ella la de la prueba H6 (incluido su comentario), queda para un trabajo aparte.
  - Hipótesis alternativa no descartada: que el EDR mantenga abierto el `.exe` recién compilado más de 2 s.
- **Restricción del EDR:** la prueba de H6 compila un ejecutable Go con `go build -o` y lo ejecuta, igual que `internal/opencode/catalog_test.go:130`. En esta máquina, el Cortex XDR corporativo veta compilar y ejecutar ejecutables propios, así que la verificación de T3 y `go test -p 4 ./internal/opencode/...` no se ejecutan aquí sin permiso expreso.
- **Entorno de la máquina (2026-10-27):** tiene `node`, `npm` y `codegraph` (npm global) en el PATH; no tiene `opencode` ni `opencode.json` real.
- **Fuera de alcance:** `internal/app` `TestTuiInstallOnThenSyncPreservesAndRefreshesOpenCodeActivation` no figura en este ODD; no se toca.

## Criterios de aceptación

- H1 y H3 pasan con `codegraph` en el PATH y con un `opencode.json` real presente en la máquina.
- H6 pasa bajo `go test -p 4` del paquete `internal/opencode`, sin errores de limpieza del directorio temporal.
- Ninguna de las tres tareas cambia código de producción: el diff contra `main` contiene solo ficheros `_test.go`.
- Ninguna prueba se omite ni se reintenta para pasar.

## Progreso

- 2026-10-13: documento abierto desde el cierre del ODD `windows-pruebas-entorno-y-ci`. Líneas verificadas en el árbol de `main` `eea58801`. No iniciado.
- 2026-10-27: ejecución iniciada desde `main` `e4b4cc3d`; las líneas citadas siguen siendo válidas. T1 (`f3b2a9a4`) y T2 (`72cba681`) hechas con RED y GREEN observados. T3 bloqueada por el hallazgo H6-D.
- 2026-10-27: el usuario acepta H6-D como defecto de producción; T3 sale del PR. T4 hecha. Falta el push y el PR, con permiso del usuario.

## Siguiente paso

Push de `fix/pruebas-aislamiento-host` y PR contra `main` con `scripts/axiom-worktree.ps1 pr pruebas-aislamiento-host`, etiqueta `type:bug`, con permiso del usuario. Aparte: corregir H6-D en producción (asignación al Job antes de que el hijo pueda lanzar descendientes) y actualizar el ratchet de código muerto para Go 1.27.
