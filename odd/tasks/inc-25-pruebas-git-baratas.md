# ODD: Pruebas git baratas y deterministas en `internal/branch`

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/inc-25-pruebas-git-baratas.md`.
> Espejo de recuperación en Engram: topic `odd/inc-25-pruebas-git-baratas/tasks`, proyecto `axiom`.

## Objetivo

Que las pruebas con git real de `internal/branch/**`, `internal/gitcommon` y `internal/cli/branch*_test.go` sean baratas en Windows y deterministas en la CI, antes de que S5 de `inc-25-ramas-y-worktrees-por-topologia` amplíe `internal/branch/engine`.

## Problema y motivo

- `internal/branch/engine` (114 pruebas, unos 2.000 a 3.000 procesos git) tarda 341 s aislado en Windows y 568 s dentro de `go test ./...`. El límite por paquete es de 600 s. En la CI de Linux tarda 29 s: el coste es el arranque de procesos git en Windows, no la lógica.
- En la CI de Linux, `TestStatusWithoutChangeListsMergedAndUnarchivedChangesAsPending` falló en el PR #169 al limpiar el directorio temporal (`unlinkat …/.git/objects/pack: directory not empty`). La causa probable es el `gc`/`maintenance` automático que git lanza en segundo plano tras un `commit`.
- El entorno de prueba apunta `GIT_CONFIG_GLOBAL` a un fichero que nunca se escribe, así que git usa sus valores por defecto: `gc.auto`, `maintenance.auto`, `fsmonitor`, `fsync` y la plantilla de hooks.

## Alcance

- Solo código de prueba (`*_test.go` y helpers de prueba) de `internal/branch/**`, `internal/gitcommon` e `internal/cli/branch*_test.go`.
- Sin cambios de comportamiento de producción, de aserciones ni de cobertura.
- Fuera de alcance:
  - Memoizar llamadas git en producción.
  - Cambiar `.github/workflows/**`. Solo se tocarían en la T3, y solo si la T1 y la T2 no bastan, con decisión previa del usuario.
  - `internal/reviewtransaction/**`.
  - `.deadcode-baseline.txt`.

## Restricciones

- Rama `inc/inc-25-pruebas-git-baratas`, que sale de la rama de seguimiento `inc/inc-25-ramas-y-worktrees-por-topologia` (56ccc3df). PR contra la rama de seguimiento.
- TDD estricto no declarado (modo estándar): se ejecutan las comprobaciones funcionales ordinarias. RDD desactivado (global off).
- Un commit Conventional por tarea, sin atribución de IA.

## Tareas

- [x] **T1 · Configuración git aislada y sin trabajo en segundo plano** (ruta: delegada; disparador: escritor, 5 o más ficheros)
  - En el `TestMain` de `engine`, `gitcli`, `registry` y `gitcommon`, y en `isolateBranchGit` de `internal/cli`, escribir el gitconfig al que ya apunta `GIT_CONFIG_GLOBAL`. Contenido: `gc.auto=0`, `gc.autoDetach=false`, `maintenance.auto=false`, `core.fsmonitor=false`, `core.untrackedCache=false` y `core.fsync=none`.
  - Fijar `GIT_TEMPLATE_DIR` a un directorio vacío. Antes, comprobar que ninguna prueba depende de `.git/hooks`.
  - Medir `go test ./internal/branch/engine -count=1` antes y después.
  - Criterio: los mismos tests en verde, sin `gc` en segundo plano, y tiempo medido y registrado.
- [x] **T2 · `t.Parallel()` en las pruebas del motor sin estado global** (ruta: delegada)
  - Marcar `t.Parallel()` en las pruebas de `internal/branch/engine` que no usan `observe(t)`, `t.Setenv`, `t.Chdir` ni globales de paquete.
  - Las que sí los usan siguen en serie.
  - Criterio: `go test ./internal/branch/engine -count=1` y `-race` (si está disponible) en verde, con el tiempo medido y registrado.
- [x] **T3 · Timeout propio (solo si hace falta)**: no necesaria, decisión del usuario del 2026-10-20.
  - Si, tras la T1 y la T2, `engine` sigue por encima de unos 300 s dentro de `go test ./...` en Windows, proponer al usuario cómo darle un timeout propio. Go no permite un timeout por paquete dentro de un único `go test ./...`, así que habría que separar la invocación en scripts o en la CI.
  - Sin decisión del usuario no se toca nada.

## Comprobaciones aplicables

- `go build ./...`, `go vet ./...`, `go run ./internal/gofmtcheck`, `bash scripts/deadcode-ratchet.sh`.
- `go test ./internal/branch/... ./internal/gitcommon -count=1 -timeout 20m`.
- `go test ./internal/cli -run 'TestBranch' -count=1 -timeout 20m`.

## Progreso y evidencia

- Línea base: `internal/branch/engine` tarda 341 s aislado en Windows (S4-c3, 2026-10-19) y 568 s dentro de `./...`. Pendiente de volver a medir en la T1.
- T1 (hecha): los cuatro `TestMain` (`engine`, `gitcli`, `registry`, `gitcommon`) y `isolateBranchGit` de `internal/cli` escriben ahora el gitconfig al que apuntaba `GIT_CONFIG_GLOBAL` (`gc.auto=0`, `gc.autoDetach=false`, `maintenance.auto=false`, `core.fsmonitor=false`, `core.untrackedCache=false`, `core.fsync=none`) y fijan `GIT_TEMPLATE_DIR` a un directorio vacío. Ninguna prueba de estos paquetes depende de `.git/hooks` ni de plantillas (búsqueda sin coincidencias), y no hay más paquetes con git real en `internal/branch/**` ni en `internal/gitcommon`. Solo código de prueba, sin cambios de aserciones.
  - Tiempos (Windows, git 2.41, máquina con carga variable, una sola ejecución por medida):
    - `go test ./internal/branch/engine -count=1`: 561 s antes, 467 s después (aislado). Dentro de `go test ./internal/branch/... ./internal/gitcommon`: 322 s. Mejora real pero modesta y con mucho ruido entre ejecuciones; la sobrecarga sigue siendo el arranque de procesos git.
    - `gitcli` + `registry` + `gitcommon` (una ejecución conjunta): antes 46,1 s, 15,8 s y 25,5 s; después 46,4 s, 17,3 s y 25,6 s (sin diferencia apreciable).
  - Comprobaciones: `go build ./...`, `go vet ./...`, `go run ./internal/gofmtcheck`, `bash scripts/deadcode-ratchet.sh` (sin funciones inalcanzables nuevas), `go test ./internal/branch/... ./internal/gitcommon -count=1 -timeout 20m` y `go test ./internal/cli -run 'TestBranch' -count=1 -timeout 20m`, todo en verde.
  - Commit: `d27945bd` (`test(branch): aislar la configuración git de las pruebas y desactivar el mantenimiento automático`).
- T2 (hecha): 84 de las 113 pruebas de `internal/branch/engine` (más `TestMain`) llaman a `t.Parallel()` al inicio, y 29 siguen en serie. Se auditó con un análisis del AST (grafo de llamadas transitivo por nombre) que ninguna prueba paralela alcance, directamente o por un ayudante, `observe(t)` (que cambia el global `gitcommon.Observe`), `useRealHub`, `t.Setenv`, `t.Chdir`, `os.Setenv`/`os.Unsetenv` ni asignaciones a globales de paquete. Las 29 en serie son exactamente las que los usan: 25 por `observe(t)` (una de ellas, `TestStatusRefusals`, también por `t.Setenv`), 2 por `useRealHub` (`t.Setenv` de `HOME`) y 2 solo por `t.Setenv`/`t.Chdir` (`TestEnsureSelectsTheRepositoryFromTheDirectoryNotFromTheEnvironment` y `TestEnsureReportsGitFailedNamingTheOperationAndRepeatingCompletes`). Ninguna prueba serial tiene subpruebas paralelas. Go ejecuta las paralelas solo cuando terminan todas las seriales, así que los cambios de globales nunca se solapan con ellas. Las paralelas solo leen el hub del `HOME` aislado de `TestMain` y cada una trabaja en su propio `t.TempDir()`. Sin correcciones sobre el diff parcial anterior: las 84 líneas eran correctas y no había pruebas seguras sin marcar.
  - Tiempos (Windows, 12 CPU lógicas, git 2.41, go 1.26.3; una ejecución por medida, con ruido): `go test ./internal/branch/engine -count=1 -parallel 1` (equivale a serie) 327 s; con el `-parallel` por defecto 139 s, 136 s y 135 s en tres ejecuciones, y 151 s dentro de `go test ./internal/branch/... ./internal/gitcommon`.
  - `-race` no disponible: `CGO_ENABLED=0` y no hay `gcc` en el `PATH`. No se ejecutó; la ausencia de carreras se apoya en la auditoría y en las tres ejecuciones paralelas sin fallos.
  - Comprobaciones: `go build ./...`, `go vet ./...`, `go run ./internal/gofmtcheck`, `bash scripts/deadcode-ratchet.sh` (sin funciones inalcanzables nuevas), `go test ./internal/branch/... ./internal/gitcommon -count=1 -timeout 20m` y `go test ./internal/cli -run 'TestBranch' -count=1 -timeout 20m`, todo en verde.
  - Commit: `f677a858` (`test(engine): ejecutar en paralelo las pruebas del motor sin estado global`).
- T3 (no necesaria, decisión del usuario del 2026-10-20):
  - Medición de `go test ./... -count=1` sobre `f677a858`, de 16:36 a 16:52, con la máquina muy cargada. `internal/branch/engine` pasa en 445 s, frente a 568 s antes y a un límite de 600 s. `gitcli`, `registry`, `gitcommon` y `hub` también pasan.
  - Los fallos son todos de la línea base o de carga y no tocan este cambio. Expiran organicruntime, app, cli, communitytool, components/sdd y reviewtransaction. Fallan backup, filemerge, semantic, tui y update.
  - Motivo para no hacerla: el límite de 10 min solo afecta al `go test ./...` local en Windows, donde ya expiran otros paquetes. La CI de Linux tarda 29 s en `engine`, y la suite completa de Windows (`windows-full-suite.yml`) usa `-timeout=30m`.
  - No se toca código, scripts ni la CI.

## Siguiente paso

PR contra la rama de seguimiento `inc/inc-25-ramas-y-worktrees-por-topologia`. Tras la fusión, borrar la rama y el worktree.
