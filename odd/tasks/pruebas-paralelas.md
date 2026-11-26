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
- [x] T2 · `sddpublish`: `TestMain` con `IsolateGit`, plantillas para `newRepo`, `newBare`, `newRepoWithRemote` y `newMonorepo`, y `t.Parallel()` (las que observan siguen en serie).
- [x] T3 · `sddstatus`: el `TestMain` aísla git y `t.Parallel()` donde no hay estado global.
- [x] T4 · `techcontext`: `TestMain` sobre `testenv.IsolateGit` y `t.Parallel()`.
- [x] T5 · `dashboard`: `t.Parallel()` donde no hay estado global.
- [x] T6 · `cli`: `isolateIntegrateGit`, `isolateBranchGit` e `isolateGateGitEnv` sin `t.Setenv`, entorno de git en `TestMain` y `t.Parallel()` en sus pruebas.
- [ ] T7 · Cierre: tiempos antes y después por paquete, CI en verde.

Ruta: todas en línea (cambios mecánicos y ya entendidos; ninguna unidad requiere un mapeo que exceda el presupuesto de evidencia ni hay motivo nombrado para delegar un escritor).

## Registro de tiempos (Windows local, `go test -count=1 -json`)

| Paquete | Antes | Después |
|---|---|---|
| `sddpublish` | 106,4 s | 28,6 s |
| `techcontext` | 62,8 s | 24,4 s |
| `sddstatus` | 79,4 s | 21,5 s |
| `dashboard` | 69,8 s | 50,5 s |
| `cli`, las 190 pruebas que usan los helpers `isolate*Git` (con `-run`) | 305,3 s | 213,0 s |

## Pruebas que siguen en serie

- `sddpublish`: las 7 que instalan `observe`, las 3 con `t.Setenv` (`TestGitPortIdentityConfigured`, `TestPublishIgnoresGitDirEnvironment`, `TestPublishIdentityMissingWritesNothing`), `TestGitPortPushOutcomes` (sustituye `pushTimeout`) y `TestPublishUnpublishedRangeTooLarge` (sustituye `maxUnpublishedCommits`; la detectó `-race` en la CI).
- `sddstatus`: 47 de 348 (además de `runtimeGrantClock`, `runtimeSyncDirectory` y `runtimeReplaceHead`, detectadas por `-race` en la CI; las que usan `t.Setenv` de `ENGRAM_PROJECT` o `HOME`, `t.Chdir`, `useBranchGuard` y sus fixtures `branchMonorepo`, `checkpointedMonorepo` e `isoBranchMonorepo`, `stubEngramExport`, `publishChangeInstanceMarker` o `gitcommon.Observe`).
- `techcontext`: 6 de 69 (las de `t.Chdir`, `fixedClock` y `renameFile`).
- `dashboard`: 75 de 253. Además de `t.Setenv`, `t.Chdir` y los seams del paquete (`runAppArgsFn`, `upgradeSequence*Fn`, los de `relaunch`, `installAdapters`, `discoverOpenCodeCatalog`), las que escriben preferencias de worktree en el hub del HOME compartido (`setWorktreePrefs` y sus fixtures `gitProject` y `embeddedProject`): `hub.Manager` reescribe `workspaces.json` con un renombrado sin bloqueo entre instancias, que en Windows choca con las escrituras de las pruebas paralelas, y las que reservan un puerto con `freePort` (lo liberan y lo vuelven a abrir: en paralelo, otra prueba puede ocuparlo entre medias; lo vio la CI). Son las de git más caras; inyectar el hub en el servicio sería un cambio de producción (fuera de alcance).
- `cli`: de las 190 pruebas con git, 125 siguen en serie: además de `t.Setenv` y `t.Chdir`, sustituyen alguna de las 58 variables del paquete que las pruebas cambian (`resolveActorIdentity`, los avisos `*Warnings`, `cmdLookPath`, etc.), `backup.UserHomeDirFn`, los `LookPathOverride` o `gitcommon.Observe`, o escriben el hub compartido (`askModeHub`). El resto de `cli` (unas 1.000 pruebas sin git) queda como estaba: 209 `t.Setenv` y decenas de variables sustituidas impiden marcarlas sin reescribirlas.
- Efecto colateral: los helpers `isolate*Git` daban a cada prueba un HOME propio; ahora comparten el de `TestMain`. `TestBranchEnsureAskAnswerWithTheModeFlagIsRecordedAndReused` suponía un hub vacío (`Workspaces[0]`) y ahora busca el proyecto por id. `TestGateRecordRealGitIdentityNonRepository` reescribía el fichero de `GIT_CONFIG_GLOBAL`, que ahora es el compartido, y dejaba a las pruebas siguientes sin la configuración que apaga el mantenimiento (la guarda de `TestBranchGuardIsInertInAForeignRepositoryWithABareOpenSpec` vio un `maintenance.lock` en la CI): ahora fija su propio `gitconfig`.

## Criterio de terminado

- Tiempos antes y después registrados arriba.
- `go vet` y `go test` de cada paquete tocado en verde en Windows local; la CI de Linux (con `-race`) en verde en la PR.

## Progreso

- T1 a T6 cerradas con su commit: `79126437`, `a8890ec1`, `59d4d19d`, `1bf86564`, `c12ec9d8` y `954e0329`.
- `-race -shuffle=on`: paso temporal en `ci.yml` (grupos `rest` y `cli` de `Go Tests`), que se retira antes del merge. En Windows local se pasó `-shuffle=on` de cada paquete en verde; `dashboard`, además, con `-count=2`. La primera ejecución detectó tres pruebas paralelas que sustituían variables del paquete en bloques `var (...)` que el primer inventario no leía; el inventario se rehízo en los cinco paquetes y esas tres volvieron a serie.
- Comprobado en local: `go vet` y `go test` de `testenv`, `sddpublish`, `sddstatus`, `techcontext` y `dashboard`; en `cli`, las 190 pruebas con git y `-run 'Identity|Actor|Gitconfig|GitConfig|Home'`; en `assets`, las guardas de los workflows.

## Siguiente paso

T7: PR, CI con `-race` en verde, retirar el paso temporal y merge.
