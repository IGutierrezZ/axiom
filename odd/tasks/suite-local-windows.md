# suite-local-windows

C-08-b del plan maestro (`docs/plan/backlog-2026-11/08-adelgazamiento.md`): suite local que termina, sin compilar con el toolchain.

## Objetivo

Que `go test` vuelva a servir de semáforo en Windows local: cada paquete termina y pasa sin lanzar el toolchain de Go, salvo los fallos que desaparecen con C-01-e (RDD) y C-08-e (GGA).

## Problema y porqué

- Cuatro tests lanzan el toolchain de Go (`go build`, `go run`, `go test -list` o `go version` por invocación) y suman unos 395 s en Windows local.
- Hay fallos locales en `main` que no son defectos del producto, sino del entorno de desarrollo, y tapan los fallos reales.

## Alcance

- Puerta para los cuatro tests del toolchain: siguen corriendo en la CI.
- Fallos locales de `backup`, `filemerge`, `cli` (doctor y state) y `app`.
- Orden local recomendada en `AGENTS.md`.

Fuera de alcance: paralelizar (C-08-g), retirar tests de prosa (C-08-h), fallos de RDD y GGA (C-01-e, C-08-e).

## Restricciones

- No compilar binarios sueltos de `axiom` en local (el EDR los borra): solo `go test` por paquetes o con `-run`.
- La CI debe seguir ejecutando los cuatro tests.

## Decisiones

- **Puerta del toolchain.** Nueva `testenv.RequireGoToolchain` sobre la misma variable `AXIOM_TEST_BUILD_BINARIES`, con su propio mensaje de omisión. Motivo: `ci.yml`, `windows-full-suite.yml`, `release.yml` y `promote-stable-rc.yml` ya fijan la variable, así que la CI sigue ejecutando los cuatro tests sin tocar los workflows (que C-01-e también edita). Mensaje propio porque `TestDocumentedInvocationsRunAsDocumented` no compila un binario: lanza `go version` en cada invocación documentada.

## TDD

Modo: desactivado (sin configuración de proyecto ni petición). Comprobaciones funcionales con `go test` por paquete.

## Tareas

- [x] T1. Puerta `testenv.RequireGoToolchain` y aplicarla a los cuatro tests del toolchain. Ruta: inline (unidad pequeña).
  - En `update`, la guarda va en `newReleasePolicyFixture`: cada ejecución de la política es `go run ./internal/releasepolicycmd`, así que cubre también `TestReleaseDistributionPolicyAcceptsSemanticYAMLFormatting` y `TestModifiedReleaseVerifierCannotGainWriteAuthority`, que compilaban por el mismo camino.
  - Evidencia: `go test ./internal/testenv/` ok; con `-v -run` los seis tests salen `SKIP` en local y `update`, `assets` y `app` terminan en `ok`.
- [x] T2. `backup` y `filemerge`: fallos del entorno en `%LOCALAPPDATA%\Temp`. Ruta: inline.
  - Diagnóstico: los tres tests pasan con `TEMP` en `C:\repos\...`, en la raíz del perfil y en `C:\Users\Public`; solo fallan con `TEMP` en `AppData\Local\Temp`, también con la ruta larga. No es el nombre corto 8.3 ni un defecto del producto: el software de seguridad del equipo protege esa carpeta.
  - `filemerge`: la creación exclusiva (`CREATE_NEW`, la de `os.CreateTemp`) a través de una junction devuelve «file already exists»; la escritura normal sí pasa. Nueva sonda `requireWritableJunction` con `O_EXCL`: si falla, `t.Skipf` con el motivo; donde funciona, la prueba corre entera.
  - `backup`: el socket AF_UNIX se crea, pero no admite `lstat` ni borrado, y la limpieza de `TempDir` falla aunque se omita la prueba. Sin sonda posible, en Windows solo corre en la CI (variable `CI`). Se corrige también el comentario de cabecera, que describía otro test.
  - Evidencia: `go test ./internal/backup/ ./internal/components/filemerge/ -count=1` en `ok`.
- [x] T3. `cli`: `TestCheckInstalledAssetVersion_*` y `TestCheckStateJSON_*`. Ruta: inline.
  - Causa: `EffectiveStateStore` parte del directorio actual. El checkout principal tiene `.axiom/state.json` de proyecto (ignorado por git), así que los tests leían ese estado real en vez del de su `homeDir`. En un worktree recién creado pasan; con una copia de ese fichero fallan cinco (RED observado).
  - Arreglo: `isolatedStateStore` hace `t.Chdir` a un directorio vacío antes de resolver el almacén; sustituye las 14 llamadas de `doctor_test.go`.
  - Evidencia: con `.axiom/state.json` copiado al worktree, `-run 'TestCheckInstalledAssetVersion_|TestCheckStateJSON_|TestDoctor|TestRunDoctor'` solo falla en `TestRunDoctor_GGARequiredWhenSelected` y `TestRunDoctor_FootprintShowsSelectedGGAAsOptional`.
  - Esos dos de GGA fallan por la misma fuga (sin el fichero pasan), pero el plan los deja fuera: desaparecen con C-08-e.
- [x] T4. `app`: `TestTuiInstallOnThenSyncPreservesAndRefreshesOpenCodeActivation` y `TestDeferredSyncIncludesCodexPermissionsArgs`. Ruta: inline.
  - `TestTuiInstall...`: **defecto real** (commit `7ef7f55a`). El lanzador gestionado `~/.axiom/bin/opencode.cmd` escribía la ruta larga del runtime (`C:\Users\IñigoGutiérrezZufia\...`) en UTF-8; `cmd.exe` lee el `.cmd` con la página de códigos OEM y devolvía «El sistema no puede encontrar la ruta especificada» (exit 3), así que la detección de versión fallaba. Lo mismo pasa con el `.ps1` en Windows PowerShell 5.1. Afecta a cualquier usuario con un perfil no ASCII. Arreglo: `windowsLauncherTarget` escribe la forma corta 8.3 (ASCII) si la larga no es ASCII y el volumen la tiene; si no, conserva la larga. Pruebas: `TestWindowsLauncherTargetUsesASCIIShortPathForNonASCIITargets` (con `shortPathName` simulado) y `TestPlatformShortPathNamePointsAtTheSameFile` (Windows real; se omite si el volumen no tiene nombres 8.3).
  - `TestDeferredSync...`: la misma fuga de T3 (`syncArgsForDiscoveredAgents` usa `EffectiveStateStore` desde el directorio actual). Arreglo: `t.Chdir(t.TempDir())`.
  - Evidencia: con `.axiom/state.json` copiado al worktree, los dos tests en `PASS`; `go test ./internal/opencode/` en `ok`; `GOOS=linux go vet ./internal/opencode/` limpio.
  - Sin impacto en la spec viva: el lanzador sigue invocando el mismo runtime; cambia solo la forma de la ruta escrita.
- [x] T6. `mcp`: `TestInjectClaudeWorkspaceIsDiscoveredByNativeClaudeMCPList` (aparecido al lanzar la suite desde una sesión de Claude Code). Ruta: inline.
  - Causa: el test ejecuta el `claude` real y hereda `BUN_INSPECT` de la sesión padre; el hijo intenta abrir el mismo puerto del depurador (9226) y sale con `EADDRINUSE`. Arreglo: quitar `BUN_INSPECT*` del entorno del subproceso.
  - Evidencia: `go test ./internal/components/mcp/ -count=1` en `ok` desde la misma sesión.
- [ ] T5. Orden local en `AGENTS.md` y verificación del criterio de terminado. Ruta: inline.

## Criterio de terminado

- En Windows local, `go test ./... -count=1` sin `internal/cli` ni `internal/reviewtransaction` termina sin fallos.
- En `internal/cli` solo fallan los tests atribuidos a RDD o GGA, listados aquí.
- El log del job `unit-tests` de la CI muestra que los cuatro tests del toolchain se ejecutan.

## Progreso y evidencias

(pendiente)
