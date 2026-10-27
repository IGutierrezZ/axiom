# ODD: el descubrimiento del catálogo de OpenCode debe meter en el Job a todos los descendientes (H6-D)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/catalogo-opencode-job-object.md`, en el repositorio `axiom`. La rama y el worktree de ejecución se crean al iniciarla, nunca en `C:\repos\axiom`.
> Espejo de recuperación en Engram: topic `odd/catalogo-opencode-job-object/tasks`, proyecto `axiom`.
> **Estado: Propuesto (2026-10-27); no iniciado.**
> Origen: hallazgo H6-D del ODD `pruebas-aislamiento-host` (`odd/tasks/pruebas-aislamiento-host.md`, sección «Hallazgos»; cerrado con #181 y #183).

## Objetivo

Que en Windows, `runCatalogCommand` (`internal/opencode/catalog.go:226`) mate siempre todo el árbol de procesos del comando de catálogo, incluidos los descendientes que el hijo lanza nada más arrancar. Y que su prueba de Windows lo demuestre de forma determinista, sin compilar ejecutables propios.

## Problema y porqué

- **Defecto de producción (H6-D).** `configureProcessGroup` (`internal/opencode/catalog_process_windows.go:31`) asigna el hijo al Job Object en `afterStart`, **después** de `cmd.Start()`. Entre el arranque y la asignación, el hijo ya ejecuta código. Si lanza un descendiente en ese intervalo, el descendiente no hereda el Job y sobrevive a `TerminateJobObject` y a `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`. Si ese descendiente hereda stdout o stderr, puede quedar vivo después del descubrimiento.
- **Síntoma observado (H6).** `TestRunCatalogCommandDeadlineNotBlockedByInheritingDescendantWindows` falla con carga (`go test -p 4`) al limpiar `t.TempDir()`: `unlinkat ...\descendant-helper.exe: The process cannot access the file because it is being used by another process`. `t.TempDir()` ya reintenta 2 s ante `ERROR_SHARING_VIOLATION` (`testing.go:1649`, `testing_windows.go:30`), así que el bloqueo dura más de 2 s: encaja con un nieto fuera del Job durmiendo 30 s.
- **La prueba no cubre lo que dice.** El hijo del ayudante sale a los ~25 ms, antes del plazo de 300 ms. `exec.Cmd.watchCtx` entrega el resultado al terminar el proceso y **no llama a `cmd.Cancel`**, así que `TerminateJobObject` no se ejecuta. El plazo lo acota `cmd.WaitDelay` (5 s), no la terminación del Job. El comentario de la prueba afirma lo contrario.
- **La prueba choca con el EDR.** Compila un ejecutable con `go build -o` y lo ejecuta. El Cortex XDR corporativo veta eso en la máquina de desarrollo, así que hoy no se puede verificar en local.

## Evidencia y patrón de referencia

- El repositorio ya resuelve esta carrera en dos sitios, con el mismo patrón:
  1. crear el proceso con `CREATE_SUSPENDED`;
  2. asignarlo al Job;
  3. reanudarlo con `NtResumeProcess`;
  4. si algo falla, matarlo (falla cerrado).
  - `internal/reviewtransaction/git_process_windows.go` (`startGitProcessTree`), con la asignación inyectable (`assignGitProcessToJob`) para probar el fallo.
  - `internal/components/engram/healthprobe_process_windows.go` (`startProbeProcessTree`).
- Patrón de prueba compatible con el EDR: volver a lanzar el binario de la prueba como ayudante (`exec.Command(os.Args[0], "-test.run=^...Helper$")` con una variable de entorno de modo), como `internal/reviewtransaction/git_process_windows_test.go:91` y `:107`. `go test` está permitido; `go build -o`, no.
- `syscall.SysProcAttr` de Go 1.27 no permite crear el proceso directamente dentro de un Job (no expone `PROC_THREAD_ATTRIBUTE_JOB_LIST`). Por eso hace falta el patrón de proceso suspendido.

## Alcance autorizado

El alcance previsto es el de las tareas siguientes. **El usuario debe autorizar la implementación al iniciar la ejecución.**

- `internal/opencode/catalog_process_windows.go` (producción).
- `internal/opencode/catalog_process_unix.go`, solo si cambia la firma compartida de `configureProcessGroup`.
- `internal/opencode/catalog.go`, solo en `runCatalogCommand` y solo si cambia esa firma.
- `internal/opencode/catalog_process_windows_test.go`.
- **Fuera de alcance:**
  - extraer un ayudante común de Job Object para los tres usos (`opencode`, `reviewtransaction`, `engram`); se anota como posible seguimiento;
  - `internal/opencode/catalog_test.go:130`, que también compila un ayudante con `go build -o`; se anota como hallazgo;
  - cambiar el comportamiento en Unix.

## Restricciones

- Commits en castellano, Conventional Commits, **sin `Co-Authored-By` ni atribución de IA**.
- Worktree dedicado (`scripts/axiom-worktree.ps1 new <slug> -Prefix fix`). `C:\repos\axiom` queda en `main`, sin ediciones.
- **EDR:** prohibido compilar o ejecutar ejecutables propios (`go build -o`, `go run ./cmd/axiom`). Solo `go test`, `go build ./...`, `go vet` y `go run` de las herramientas del repositorio (`./internal/gofmtcheck`).
- En local, solo paquetes focalizados con `-run`. Nunca `go test ./internal/opencode/...` entero, porque incluye `catalog_test.go:130` (`go build -o`).
- Sin `t.Skip` para esquivar el problema y sin bucles de `sleep` y reintento. Las esperas de la prueba se acotan con esperas sobre manejadores de proceso (`WaitForSingleObject` con plazo), no con sondeo.
- Push, PR y merge solo con permiso del usuario.

## Decisiones

- **D1 (por confirmar al iniciar): qué hacer si no se puede asignar el proceso suspendido.** Hoy la asignación es «best effort»: si falla, la cancelación se degrada a matar al hijo directo. Con el hijo suspendido, no se puede dejar así.
  - Recomendación: fallar cerrado, como `reviewtransaction` y `engram`. Se mata el hijo suspendido y `runCatalogCommand` devuelve el error, de modo que el descubrimiento informa del fallo en lugar de dejar procesos fuera de control.
  - Alternativa: reanudarlo sin Job y mantener la degradación actual. Conserva el descubrimiento en hosts que denieguen Jobs anidados, pero mantiene el riesgo de descendientes huérfanos.
  - La documentación de Microsoft indica que los Jobs anidados están permitidos desde Windows 8 y Windows Server 2012.

## Modo de trabajo

Re-comprobar al iniciar.

- **TDD:** modo estándar. Fuente: `openspec/config.yaml` (`strict_tdd: false`). Runner: `go test`. Aun así, se busca un RED observado antes de la corrección de producción (T1), porque el defecto es una carrera y sin RED no hay prueba de que la corrección la cierre.
- **RDD:** releer el estado con `axiom review mode status`. Es un cambio de producción que gestiona procesos y manejadores en Windows: cabe esperar riesgo medio o alto en `axiom review assess`.
- **Entrega:** estrategia `ask-on-risk`, un solo PR. Previsión orientativa: de 120 a 220 líneas de autoría.

## Tareas

- [ ] **T1 · Prueba determinista y compatible con el EDR (RED)**
  - Ruta: en línea (un fichero).
  - Reescribir el ayudante de `catalog_process_windows_test.go` como relanzamiento del binario de la prueba (`os.Args[0]`, `-test.run`, variable de modo). El hijo lanza un nieto que hereda stdout y stderr, y publica su PID por un canal sin carreras (por ejemplo, una línea en stdout antes de dormir).
  - Afirmar lo que importa: cuando `runCatalogCommand` vuelve, **el nieto ya no está vivo**. Comprobarlo con `OpenProcess` + `WaitForSingleObject` y un plazo acotado.
  - Cubrir los dos caminos:
    - el hijo sale antes del plazo (lo acota `WaitDelay`; al nieto lo mata el cierre del Job);
    - el hijo sigue vivo al vencer el plazo (`cmd.Cancel` → `TerminateJobObject`).
  - RED determinista: la carrera depende del reparto de CPU. Para reproducirla sin depender de la carga, hacer inyectable el paso de asignación (como `assignGitProcessToJob`) y, en la prueba, retrasar la asignación hasta que el hijo haya publicado el PID del nieto. Con el código actual, el nieto escapa y la prueba falla. Anotar la salida.
  - Corregir el comentario de la prueba: el límite de tiempo lo da `WaitDelay` cuando el hijo sale antes del plazo.
  - Comprobación: `go test ./internal/opencode -run '^TestRunCatalogCommand.*Windows$' -count=1 -v` (RED esperado).
  - Commit: `test(opencode): demostrar que un descendiente temprano escapa del Job en el descubrimiento de catálogo`.
    - Si el repositorio exige no dejar `main` en rojo entre commits, T1 y T2 se entregan en el mismo commit y el RED queda anotado en este documento.
- [ ] **T2 · Corrección de producción: crear suspendido, asignar y reanudar**
  - Ruta: en línea si basta con `catalog_process_windows.go`; si cambia la firma compartida y toca dos o más ficheros de forma no trivial, escritor delegado.
  - `configureProcessGroup` (Windows) fija `SysProcAttr.CreationFlags |= CREATE_SUSPENDED`. El paso posterior al arranque abre el proceso con `PROCESS_SET_QUOTA|PROCESS_TERMINATE|PROCESS_SUSPEND_RESUME`, lo asigna al Job y lo reanuda con `NtResumeProcess`.
  - Fallos según D1. Si se falla cerrado, el paso posterior al arranque devuelve error, `runCatalogCommand` mata el hijo, espera con `Wait` y propaga el error. La firma `afterStart func()` pasa a devolver `error` también en Unix (allí, `nil`).
  - Mantener `runtime.KeepAlive` de la estructura de límites tras pasar su puntero, como en `reviewtransaction`.
  - Actualizar el comentario de `configureProcessGroup`: la asignación ya no es «best effort» y la carrera queda cerrada.
  - Comprobación: la misma de T1, ya en verde; y la prueba de T1 con `-count=20` y con `-p 4 -count=10` sobre el mismo filtro, sin fallos de limpieza.
  - Commit: `fix(opencode): crear suspendido el comando de catálogo para que sus descendientes hereden el Job`.
- [ ] **T3 · Prueba del fallo de asignación** (si D1 = fallar cerrado)
  - Con la asignación inyectada para fallar, `runCatalogCommand` devuelve error y el hijo suspendido no queda vivo. Mismo enfoque que `TestStartGitProcessTreeKillsUnboundChildOnFailure`.
  - Commit: `test(opencode): el descubrimiento de catálogo mata el hijo si no puede asignarlo al Job`.
- [ ] **T4 · Cierre**
  - `go build ./...`, `go vet ./internal/opencode/...`, `go run ./internal/gofmtcheck`, `bash scripts/deadcode-ratchet.sh`.
  - Las pruebas de T1 a T3 con `-count=1`; T1 también con carga, como en T2.
  - Comprobar con `GOOS=linux go vet ./internal/opencode/...` que la variante Unix compila si cambió la firma.
  - `git status --porcelain` vacío. Actualizar este documento y el espejo.
  - Preparar el PR. Push, PR y merge los decide el usuario. Valorar la etiqueta `ci:windows` para que el CI de Windows ejecute la prueba.

## Criterios de aceptación

- En Windows, cuando `runCatalogCommand` vuelve, ningún descendiente del comando de catálogo sigue vivo, tanto si el hijo sale antes del plazo como si se cancela por plazo.
- La prueba reproduce la carrera de forma determinista (RED con el código anterior) y pasa con la corrección, también con `-p 4` y repeticiones.
- Ninguna prueba del paquete en este alcance compila ejecutables propios.
- El fallo de asignación tiene el comportamiento acordado en D1, cubierto por una prueba.
- Sin `t.Skip`, sin reintentos y sin sondeo con `sleep`.

## Hallazgos de partida

- `internal/opencode/catalog_test.go:130` compila un ayudante con `go build -o`: mismo choque con el EDR, fuera de este alcance.
- Tres implementaciones casi iguales del patrón de Job Object (`opencode`, `reviewtransaction`, `engram`): posible seguimiento para unificarlas.

## Progreso

- 2026-10-27: documento abierto a partir del hallazgo H6-D, sobre `main` `f077bf1b`. Líneas verificadas en ese árbol. No iniciado.

## Siguiente paso

El usuario autoriza la ejecución y decide D1. Empezar por la conciliación: `mem_search` del topic, este fichero, `git fetch --prune origin` y comprobar que las líneas citadas siguen siendo válidas. Después, T1.
