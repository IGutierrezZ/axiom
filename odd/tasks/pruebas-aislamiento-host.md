# ODD: aislamiento del host en tres pruebas de `cli`, `tui` y `opencode` (H1, H3 y H6)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/pruebas-aislamiento-host.md`, en el repositorio `axiom`. La rama y el worktree dedicados se crean al iniciar la ejecución, nunca en `C:\repos\axiom`.
> Espejo de recuperación en Engram: topic `odd/pruebas-aislamiento-host/tasks`, proyecto `axiom` (creado el 2026-10-13 en estado propuesto; se actualiza al iniciar la ejecución).
> **Estado: Propuesto; no iniciado. Se ejecuta en una sesión aparte.**
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

- [ ] **T1 · H1: aislar `cmdLookPath` en la prueba de Pi CodeGraph**
  - Ruta: en línea (un fichero, cambio acotado). Si el aislamiento obliga a tocar más de un fichero de forma no trivial, escritor delegado.
  - RED: con `codegraph` en el PATH, ejecutar la prueba y comprobar que lanza el `codegraph install` real (o falla por ello). Anotar la salida.
  - Corrección: sustituir `cmdLookPath` con restauración en `t.Cleanup`, como hace la prueba hermana, o inyectar el ejecutor del paso (`s.runner`). Decidir el valor que conserve el sentido de la prueba (la hermana devuelve «encontrado»; aquí lo que se afirma es la reconciliación de la selección de Pi, y un valor «no encontrado» evita el `ReconcileOpenCodeCodeGraph`).
  - Comprobación: `go test ./internal/cli -run '^TestRunSyncMigratesLegacyManaged(Pi)?CodeGraphSelection$' -count=1`, **con `codegraph` en el PATH y sin él**.
  - Commit: `test(cli): aislar la detección de codegraph en la migración de la selección de Pi`.
- [ ] **T2 · H3: aislar la configuración real de OpenCode**
  - Ruta: en línea (un fichero, cambio acotado).
  - RED: con un `opencode.json` real presente (o un `opencode.json` de prueba con un proveedor ajeno en el directorio de inicio real), comprobar que el proveedor ajeno entra en el catálogo. Anotarlo.
  - Corrección: al principio de la prueba, `t.Setenv` de `USERPROFILE`, `HOME` y `OPENCODE_CONFIG_DIR` apuntando a directorios temporales (la última, una ruta absoluta, porque `config.go:49` ignora las relativas).
  - Comprobación: `go test ./internal/tui -run '^TestRuntimeCatalogDiscoveryIgnoresStaleProjectResults$' -count=1`, con y sin un `opencode.json` real del usuario presente.
  - Commit: `test(tui): aislar el directorio de inicio y la configuración de OpenCode en el descubrimiento de catálogo`.
- [ ] **T3 · H6: limpieza determinista del ayudante en Windows**
  - Ruta: en línea o escritor delegado según el tamaño.
  - Diagnóstico previo, obligatorio: determinar por qué el nieto sigue vivo en la limpieza.
    - Si es el orden de cierre de la prueba (la limpieza se ejecuta antes de que termine la terminación del árbol), se corrige en la prueba.
    - **Si el nieto sigue vivo porque la producción no termina el árbol de procesos tras el plazo, es un defecto real**: se detiene el trabajo, se anota como hallazgo y se pregunta al usuario. No se arregla en este ODD y no se oculta con una limpieza más tolerante.
  - Corrección, solo en la prueba: limpieza determinista. Matar el árbol del ayudante o esperar a que termine, y registrar la limpieza en el orden correcto (`t.Cleanup` se ejecuta en orden inverso al registro; la terminación debe ejecutarse antes del borrado del directorio). Prohibido un bucle de reintentos con `sleep`.
  - Comprobación:
    - `go test ./internal/opencode -run '^TestRunCatalogCommandDeadlineNotBlockedByInheritingDescendantWindows$' -count=1`;
    - bajo carga: `go test -p 4 ./internal/opencode/... -count=1`, varias veces, y comprobar que el directorio temporal se limpia sin error.
  - Commit: `test(opencode): limpieza determinista del ayudante descendiente en la prueba de plazo de Windows`.
- [ ] **T4 · Cierre**
  - `go build ./...`, `go vet ./...`, `go run ./internal/gofmtcheck`, `bash scripts/deadcode-ratchet.sh`, y las tres pruebas con `-count=1`.
  - `git diff --stat origin/main...HEAD` debe mostrar solo ficheros `_test.go`.
  - `git status --porcelain` vacío. Actualizar este documento y el espejo.
  - Preparar el PR (un solo PR). Push, PR y merge los decide el usuario.

## Criterios de aceptación

- H1 y H3 pasan con `codegraph` en el PATH y con un `opencode.json` real presente en la máquina.
- H6 pasa bajo `go test -p 4` del paquete `internal/opencode`, sin errores de limpieza del directorio temporal.
- Ninguna de las tres tareas cambia código de producción: el diff contra `main` contiene solo ficheros `_test.go`.
- Ninguna prueba se omite ni se reintenta para pasar.

## Progreso

- 2026-10-13: documento abierto desde el cierre del ODD `windows-pruebas-entorno-y-ci`. Líneas verificadas en el árbol de `main` `eea58801`. No iniciado.

## Siguiente paso

El usuario lanza esta ejecución en una sesión aparte (segundo ODD de la serie: H8, aislamiento de pruebas, H2). Empezar por la conciliación: `mem_context`, `mem_search` del topic, este fichero, `git fetch --prune origin`, y comprobar que las líneas citadas siguen siendo válidas. Después T1.
