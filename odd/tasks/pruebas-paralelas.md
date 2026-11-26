# ODD: Pruebas en paralelo y repositorios git de plantilla (pruebas-paralelas)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/pruebas-paralelas.md`. Espejo Engram: `odd/pruebas-paralelas/tasks`.
> Rama: `inc/pruebas-paralelas` · worktree `C:\repos\axiom-wt\pruebas-paralelas\axiom`. Cambio C-08-g del plan maestro (`docs/plan/backlog-2026-11/08-adelgazamiento.md`), ola A7 (revisión 10).

## Objetivo

Bajar el tiempo de pared de las pruebas en Windows local: fijar el entorno de git una sola vez por binario de pruebas en `TestMain`, abaratar la creación de repositorios con plantillas que se copian en lugar de reconstruirse con procesos git, y marcar `t.Parallel()` en los paquetes que hoy corren en serie.

## Problema y porqué

En Windows lo caro es arrancar procesos git: `sddstatus` tarda 33 veces más que en Linux, `sddpublish` 26, `techcontext` 24 y `dashboard` 11; `internal/cli` pasa de 52 s en Linux a más de 10 min en local. Ninguno de los cuatro primeros tiene un solo `t.Parallel()`. El patrón de `internal/branch/engine` (inc-25: entorno fijado en `TestMain` y `t.Parallel()`) dio ×2,4.

## Decisiones

- **Entorno común en `internal/testenv` (plan).** `IsolateGit` lo llama el `TestMain` de cada paquete: HOME vacío, `gitconfig` global sin `gc`, mantenimiento, `fsmonitor`, caché de no seguidos ni `fsync`, plantilla de `git init` vacía, autor fijo, sin terminal ni sistema, y sin `GIT_DIR`, `GIT_WORK_TREE` ni `GIT_INDEX_FILE` heredados. Así los helpers `isolate*` dejan de llamar a `t.Setenv`, que impide `t.Parallel()`.
- **Plantillas de repositorio (plan).** `testenv.RepoTemplate` construye el repositorio una vez por paquete y cada prueba recibe una copia del directorio. Las rutas absolutas de la plantilla en los ficheros `config`, `gitdir` y `.git` se reescriben a la copia, para que un `origin` de la plantilla nunca apunte al remoto compartido.
- **Observador de git global (consecuencia).** `gitcommon.Observe` es global, así que las pruebas de `sddpublish` que lo instalan (`observe`) siguen en serie. Se descartó repartir las invocaciones por directorio: git devuelve rutas largas con barras y `t.TempDir` cortas en Windows, y un fallo de normalización dejaría pasar en vacío las aserciones de «no se ejecutó nada». Son 7 de 51 pruebas.
- **Pruebas que siguen en serie.** Las que fijan entorno propio (`t.Setenv`), cambian de directorio (`t.Chdir`) o sustituyen variables globales de producción. Se anotan por paquete en el registro de abajo.
- **Duplicados de `runGit` y `requireGit` (plan).** En los paquetes tocados, los helpers locales pasan a delegar en `testenv`; no se reescriben las llamadas.
- **Fuera de alcance.** Comportamiento de producción. `components/sdd`, `app` y `update` (el plan los deja para «si compensa»). `TestScriptsAxiomWorktreePowerShellDrivesAxiomBranch` (fallo previo de `internal/workspace`).

## Alcance autorizado

`internal/testenv`; los `TestMain` y helpers de prueba de `internal/sddpublish`, `internal/sddstatus`, `internal/dashboard`, `internal/techcontext` e `internal/cli`; las líneas `t.Parallel()` de sus pruebas.

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y no hay elección del usuario. Las piezas nuevas de `testenv` llevan sus pruebas; cada paquete se comprueba con `go vet` y `go test` del propio paquete (en `cli`, con `-run`). Sin la batería completa ni binarios sueltos de `axiom`. `-race -shuffle=on` se comprueba en la CI de Linux (en Windows local no hay gcc).

## Entrega

`single-pr`: una PR con un commit por tarea, etiqueta `type:refactor`. Previsión: unas 1.200 líneas cambiadas, la mayoría líneas `t.Parallel()`.

## Tareas

- [x] T1 · `internal/testenv`: `IsolateGit`, `RequireGit`, `Git`, `TryGit`, `RepoTemplate` y sus pruebas.
- [x] T2 · `sddpublish`: `TestMain` con `IsolateGit`, plantillas para `newRepo`, `newBare` y `newMonorepo`, observador por repositorio y `t.Parallel()`.
- [x] T3 · `sddstatus`: el `TestMain` aísla git y `t.Parallel()` donde no hay estado global.
- [ ] T4 · `techcontext`: `TestMain` sobre `testenv.IsolateGit` y `t.Parallel()`.
- [ ] T5 · `dashboard`: `t.Parallel()` donde no hay estado global.
- [ ] T6 · `cli`: `isolateIntegrateGit`, `isolateBranchGit` e `isolateGateGitEnv` sin `t.Setenv`, entorno de git en `TestMain` y `t.Parallel()` en sus pruebas.
- [ ] T7 · Cierre: tiempos antes y después por paquete, CI en verde.

Ruta: todas en línea (cambios mecánicos y ya entendidos; ninguna unidad requiere un mapeo que exceda el presupuesto de evidencia ni hay motivo nombrado para delegar un escritor).

## Registro de tiempos (Windows local, `go test -count=1 -json`)

| Paquete | Antes | Después |
|---|---|---|
| `sddpublish` | 106,4 s | 28,6 s |
| `techcontext` | 62,8 s | pendiente |
| `sddstatus` | 79,4 s | 21,5 s |
| `dashboard` | 69,8 s | pendiente |

## Pruebas que siguen en serie

- `sddpublish`: las 7 que instalan `observe`, las 3 con `t.Setenv` (`TestGitPortIdentityConfigured`, `TestPublishIgnoresGitDirEnvironment`, `TestPublishIdentityMissingWritesNothing`) y `TestGitPortPushOutcomes` (sustituye `pushTimeout`).
- `sddstatus`: 45 de 348 (las que usan `t.Setenv` de `ENGRAM_PROJECT` o `HOME`, `t.Chdir`, `useBranchGuard` y sus fixtures `branchMonorepo`, `checkpointedMonorepo` e `isoBranchMonorepo`, `stubEngramExport`, `publishChangeInstanceMarker` o `gitcommon.Observe`).

## Criterio de terminado

- Tiempos antes y después registrados arriba.
- `go vet` y `go test` de cada paquete tocado en verde en Windows local; la CI de Linux (con `-race`) en verde en la PR.

## Progreso

- Worktree creado; documento registrado.

## Siguiente paso

T1.
