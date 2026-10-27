# ODD: `ReindexCodeGraph` debe resolver `codegraph` a través del `Detector` inyectado (H2)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/semantic-detector-codegraph.md`, en el repositorio `axiom`. La rama y el worktree dedicados se crean al iniciar la ejecución, nunca en `C:\repos\axiom`.
> Espejo de recuperación en Engram: topic `odd/semantic-detector-codegraph/tasks`, proyecto `axiom` (creado el 2026-10-13 en estado propuesto; se actualiza al iniciar la ejecución).
> **Estado: En ejecución (2026-10-27).** Rama `fix/semantic-detector-codegraph`, worktree `C:\repos\axiom-wt\semantic-detector-codegraph`, desde `origin/main` `e4b4cc3d`.
> Origen: hallazgo H2 del ODD `windows-pruebas-entorno-y-ci` (`odd/tasks/windows-pruebas-entorno-y-ci.md`).

## Objetivo

Que `Service.ReindexCodeGraph` use el `Detector` inyectado para localizar el binario `codegraph`, en lugar de llamar a `exec.LookPath` directamente. Con ello, `TestServiceReindexCodeGraph` deja de depender del host, y el comportamiento para los usuarios reales no cambia.

## Problema y porqué

- `Service.ReindexCodeGraph` (`internal/semantic/service.go:247`) llama a `exec.LookPath("codegraph")` en `service.go:260`. Ignora el campo `detector` del servicio, que el resto del paquete sí usa.
- `TestServiceReindexCodeGraph` (`internal/semantic/semantic_test.go:253-273`) inyecta un `Detector` que responde «no encontrado» (`NewDetectorWithCustomHome(..., func(string) (string, error) { return "", os.ErrNotExist })`), pero esa inyección no tiene efecto sobre la reindexación. En cualquier host con `codegraph` en el PATH, la prueba ejecuta el `codegraph index` real sobre un directorio temporal.
- Es un defecto de diseño de producción (una dependencia sin inyectar), no solo de la prueba. Por eso es un cambio de producto, aunque pequeño, y no un arreglo de aislamiento de pruebas.
- El `Detector` ya tiene la función de búsqueda inyectable (`lookPathFunc`, `internal/semantic/detector.go:14`) y un método que la usa para `codegraph` (`CheckCodeGraphInPath`, `detector.go:184`). Ese método devuelve solo un booleano; la reindexación necesita la **ruta** del binario para ejecutarlo.

## Evidencia

- Llamada directa: `internal/semantic/service.go:260`.
- Constructor: `NewService(workspaceRoot, detector, engine)` (`service.go:26`); con `detector == nil` crea `NewDetector()`, que usa `exec.LookPath` (`detector.go:25`), de modo que la producción real ya pasa por el mismo mecanismo.
- Llamadores de `ReindexCodeGraph` en el árbol actual:
  - `cmd/axiom/main.go:1620` (`semantic.NewService(baseDir, nil, nil)`, detector por defecto);
  - `internal/dashboard/service.go:741`, que delega en `semanticService`, y su consumidor HTTP `internal/dashboard/server.go:633`.
- Un único llamador de pruebas directo: `semantic_test.go:260`.
- El diagnóstico de T1 del ODD de origen confirmó la ejecución del `codegraph index` real en una máquina con `codegraph` en el PATH.

## Alcance autorizado

El alcance previsto es el cambio mínimo siguiente. **El usuario debe autorizar la implementación al iniciar la sesión.**

- `internal/semantic/service.go`, y `internal/semantic/detector.go` si hace falta un método que devuelva la ruta del binario.
- `internal/semantic/semantic_test.go`, con la prueba que falla hoy.
- **Fuera de alcance:** cambiar el mensaje, el formato de salida o los tiempos de espera de la reindexación; el resto de los usos de `exec.LookPath` del repositorio; cualquier cambio en `dashboard` o en `cmd/axiom` salvo que T1 demuestre que hace falta.

## Restricciones

- Commits en castellano, Conventional Commits, **sin `Co-Authored-By` ni atribución de IA**.
- Trabajar en un worktree dedicado; `C:\repos\axiom` es el checkout de otra sesión y no se toca.
- En Windows, en local solo paquetes focalizados con `-run`, nunca la suite completa.
- Push, PR y merge solo con permiso del usuario.
- Sin reintentos ni omisiones que oculten un fallo.
- El comportamiento visible para el usuario no cambia: mismos mensajes, mismo `ReindexResult`, misma cancelación y mismo tiempo límite de 60 s.

## Modo de trabajo

Re-comprobar al iniciar.

- **TDD:** modo estándar. Fuente: `openspec/config.yaml` (`strict_tdd: false`). Runner: `go test`. Aun así, T2 pide la prueba que falla (RED) antes del cambio, porque el defecto solo se demuestra así.
- **RDD:** releer el estado con `axiom review mode status`. Si está activo, el candidato toca código de producción: evaluar su riesgo con `axiom review assess`.
- **Entrega:** estrategia `ask-on-risk`. Previsión orientativa: de 40 a 100 líneas de autoría, en un solo PR.

## Tareas

- [x] **T1 · Mapear**
  - Ruta: en línea (pocos ficheros) o mapeador delegado si el árbol de llamadas resulta mayor de lo previsto.
  - Confirmar el listado de llamadores de `ReindexCodeGraph` y de `Detector`, y cómo se inyectan en `cmd/axiom`, `dashboard` y las pruebas. Decidir qué API añadir al `Detector`, por ejemplo un método que devuelva la ruta de `codegraph` resuelta con `lookPathFunc` (y un error si no está), sin romper `CheckCodeGraphInPath`.
  - Comprobar si alguna otra prueba (por ejemplo, de `dashboard`) depende hoy de que `ReindexCodeGraph` resuelva por el PATH real.
  - Sin commit; el mapa va a este documento.
  - **Resultado (2026-10-27, en línea, 3 ficheros leídos):** las líneas citadas siguen válidas en `e4b4cc3d` (`service.go:260`). Llamadores de producción: `cmd/axiom/main.go:1618` y `internal/dashboard/service.go:64,92,775`, todos con `semantic.NewService(..., nil, nil)`, es decir, `NewDetector()` con `exec.LookPath`: la conducta de producción no cambia. API elegida: `Detector.ResolveCodeGraph() (string, error)`, que delega en `lookPathFunc`; `CheckCodeGraphInPath` pasa a usarla. Las pruebas de `dashboard` (`TestSemanticEndpoints`, `TestSemanticReindexWithoutCodeGraph`, `TestSemanticReindexSurfacesCodeGraphFailure`) ponen un `codegraph` falso en el PATH y usan el detector por defecto, así que no dependen del cambio. `cmd/axiom` no tiene pruebas de la reindexación.
  - **Hallazgo:** esta máquina Windows **sí** tiene `codegraph` en el PATH (`%APPDATA%\npm\codegraph`). `TestServiceReindexCodeGraph` fallaba porque ejecutaba el `codegraph index` real sobre un directorio temporal, que sale con `exit status 1`.
- [x] **T2 · Resolver mediante el `Detector`, con RED primero**
  - Ruta: escritor delegado si se tocan dos o más ficheros no triviales; en línea si el cambio queda acotado.
  - RED: una prueba que falla hoy en cualquier host con `codegraph` en el PATH, o una versión hermética que no necesite `codegraph` real:
    - un `codegraph` falso en el PATH (un script o ejecutable de prueba que deja una marca) y un `Detector` que responde «no encontrado»; la prueba exige que **no** se ejecute el falso y que el resultado sea el mensaje de «no se encuentra en el PATH»;
    - y la contraria: un `Detector` que devuelve una ruta ficticia hacia un ejecutable de prueba y la prueba exige que se ejecute ese, con `index` como argumento y el directorio de trabajo del workspace.
  - Cambio: `ReindexCodeGraph` obtiene el binario del `Detector` (`s.detector`) y no de `exec.LookPath`. El resto del flujo queda igual.
  - Comprobación: `go test ./internal/semantic -count=1` (el paquete es pequeño), con y sin `codegraph` en el PATH.
  - Commit: `fix(semantic): resolver codegraph a través del detector inyectado en la reindexación`.
  - **Resultado (2026-10-27, en línea: dos ediciones de una línea en producción y un fichero de prueba nuevo; no dispara el escritor delegado):** pruebas nuevas en `internal/semantic/reindex_test.go` con un `codegraph` falso en script (`.cmd` en Windows, `sh` en Unix), sin compilar ejecutables: `TestServiceReindexCodeGraphIgnoresPathWhenDetectorMisses` (falso en el PATH y `Detector` que no lo encuentra) y `TestServiceReindexCodeGraphRunsDetectorBinary` (PATH vacío y `Detector` que devuelve la ruta del falso; exige `index` y el directorio del workspace). `TestServiceReindexCodeGraph` exige además el conector `native-ast`.
  - RED observado antes del cambio: las tres fallan (conector `codegraph` y ejecución del falso del PATH; conector `native-ast` sin ejecutar el falso del `Detector`; `exit status 1` del `codegraph` real).
  - GREEN tras el cambio: `go test ./internal/semantic/... -count=1` → `ok` (7 pruebas), con el `codegraph` real en el PATH de esta máquina.
- [x] **T3 · Comprobar que la conducta pública no cambia**
  - Ruta: en línea.
  - Con `codegraph` real en el PATH y con `NewService(dir, nil, nil)` (el camino de `cmd/axiom/main.go:1618`), la reindexación sigue ejecutando `codegraph index` en la raíz del workspace y devolviendo el mismo `ReindexResult`. Sin `codegraph`, el mismo mensaje de antes.
  - Comprobar también las pruebas de `internal/dashboard` y de `cmd/axiom` que usen la reindexación, con `-run` focalizado.
  - Sin commit si no hay hallazgos; si aparece una divergencia, se anota y se decide con el usuario.
  - **Resultado (2026-10-27, en línea):** sin divergencias. `go test ./internal/dashboard -run TestSemantic -count=1` → `ok`: `TestSemanticEndpoints` (detector por defecto con `codegraph` falso en el PATH: conector `codegraph`, `index` en la raíz del proyecto), `TestSemanticReindexWithoutCodeGraph` (sin `codegraph`: `native-ast`) y `TestSemanticReindexSurfacesCodeGraphFailure` (fallo visible con 500). No se ejecutó el `codegraph index` real sobre un workspace para no escribir su índice; el camino `NewService(dir, nil, nil)` queda cubierto por esas pruebas con el falso. `cmd/axiom` no tiene pruebas de la reindexación.
- [x] **T4 · Cierre**
  - `go build ./...`, `go vet ./...`, `go run ./internal/gofmtcheck`, `bash scripts/deadcode-ratchet.sh`, los paquetes tocados con `-count=1` y `git status --porcelain` vacío.
  - Si RDD está activo, evaluar el candidato con `axiom review assess` antes de declararlo terminado.
  - Actualizar este documento y el espejo. Preparar el PR; push, PR y merge los decide el usuario.
  - **Resultado (2026-10-27):** `go build ./...` → OK; `go vet ./internal/semantic/... ./internal/dashboard/...` → OK; `go run ./internal/gofmtcheck` → OK; `go test ./internal/semantic/... -count=1` → `ok`. `go vet ./...` completo no se pasó (límite de tiempo en esta máquina; se acotó a los paquetes tocados y su consumidor). **No disponible:** `bash scripts/deadcode-ratchet.sh` entra en pánico dentro de `golang.org/x/tools@v0.30.0/go/ssa` con Go 1.27 (`panic: unexpected expr: *ast.KeyValueExpr`), ajeno al cambio; queda para la CI. RDD apagado, sin evaluación de riesgo. Commit de la unidad: `57679e91`.

## Criterios de aceptación

- `TestServiceReindexCodeGraph` es hermética: pasa igual con `codegraph` en el PATH y sin él, y nunca ejecuta el binario real.
- Con `codegraph` presente, el comportamiento de producción no cambia (T3).
- Existe una prueba que falla antes del cambio y pasa después.

## Progreso

- 2026-10-13: documento abierto desde el cierre del ODD `windows-pruebas-entorno-y-ci`. Líneas verificadas en el árbol de `main` `eea58801`. No iniciado.
- 2026-10-27: conciliado con el espejo de Engram (#628, mismo contenido). Implementación autorizada por la sesión de origen. T1 y T2 hechas. TDD: estándar (`openspec/config.yaml`, `strict_tdd: false`), runner `go test`. RDD: `off (decided by default)`.
- 2026-10-27: T3 y T4 hechas. Pendiente: push y PR con permiso del usuario (`axiom-worktree.ps1 pr semantic-detector-codegraph`, etiqueta `type:bug`).

## Siguiente paso

Tareas T1 a T4 hechas en la rama `fix/semantic-detector-codegraph`. Falta el push y el PR contra `main` con `scripts/axiom-worktree.ps1 pr semantic-detector-codegraph` (etiqueta `type:bug`), que decide el usuario; tras el merge, `scripts/axiom-worktree.ps1 done semantic-detector-codegraph`.
