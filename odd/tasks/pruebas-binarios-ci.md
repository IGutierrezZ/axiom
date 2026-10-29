# ODD: las pruebas que compilan binarios solo corren cuando se piden (CI)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/pruebas-binarios-ci.md`, en el repositorio `axiom`.
> Espejo de recuperación en Engram: topic `odd/pruebas-binarios-ci/tasks`, proyecto `axiom`.
> **Estado: En curso (2026-10-29), PR abierto.** Rama `fix/pruebas-binarios-ci`, worktree `C:\repos\axiom-wt\pruebas-binarios-ci`.

## Objetivo

Que `go test ./...` en un equipo de desarrollo no compile ni ejecute binarios propios (`axiom.exe` ni ayudantes), y que la CI los siga ejecutando todos.

## Problema y porqué

- El EDR corporativo (Cortex XDR) bloquea y borra los binarios Go sin firmar que las pruebas compilan en `%TEMP%`. Regla YARA `pe.sliver_implant_32bit`, falso positivo. Cada ejecución local genera una alerta de seguridad.
- Caso observado el 2026-10-27 a las 14:16: `TestTUIExecuteWithBackgroundPreservesConcurrentCLIStateMutation` (`internal/app/app_test.go`), vía `buildAppCandidateBinary`, ejecutando `axiom.exe review mode enable --scope global`.
- La norma «solo `go test`» no basta: el propio `go test` de estos paquetes compila y lanza ejecutables.

## Puntos que compilan binarios (exploración, 2026-10-27)

- `internal/app/app_test.go` → `buildAppCandidateBinary` (`go build ../../cmd/axiom`).
- `internal/app/tty_boundary_test.go` → `go build ./cmd/axiom`.
- `internal/update/cooldown_concurrency_test.go` → `go build ./cmd/axiom`.
- `internal/opencode/catalog_test.go` y `catalog_process_windows_test.go` → ayudante compilado.
- `internal/components/gga/runtime_test.go` → ayudante (*shim*) compilado.
- `bench/main_test.go` → binario de prueba compilado.
- `e2e/organicruntime/organic_runtime_test.go` → `go build ./cmd/axiom` (vía `organicCommandContext`) y otro *fixture*.
- No compilan: `internal/app/testmain_test.go` (`go env`) e `internal/assets/formatter_ordering_test.go` (`go test -list`).

## Decisión

- Una única guarda en `internal/testenv`: la prueba que compila un binario llama a la guarda antes de compilar y hace `t.Skip` salvo que `AXIOM_TEST_BUILD_BINARIES=1`. El mensaje de omisión nombra la variable.
- Sin activación implícita por `CI`: la CI la declara de forma explícita y visible.
- La CI fija `AXIOM_TEST_BUILD_BINARIES: "1"` a nivel de *workflow* en los flujos que ejecutan `go test`.
- Coste aceptado: en local esas pruebas dejan de cubrirse; la CI las sigue ejecutando.

## Alcance autorizado

- `internal/testenv` (guarda y su prueba).
- Los ficheros de prueba de la lista anterior: solo llamar a la guarda.
- `.github/workflows/*.yml` que ejecutan `go test`: solo la variable de entorno.
- Documentación de desarrollo que describa cómo ejecutar las pruebas, si existe.
- Este documento.
- **Fuera de alcance:** cambiar lo que comprueban las pruebas o convertirlas a arneses en proceso.

## Restricciones

- Commits en castellano, Conventional Commits, sin `Co-Authored-By` ni atribución de IA.
- Worktree dedicado; `C:\repos\axiom` siempre en `main` y sin escrituras.
- Veto del EDR: no compilar ni ejecutar binarios propios en local; no ejecutar las pruebas con `AXIOM_TEST_BUILD_BINARIES=1` en local.
- No pasar `go test ./...` completo ni e2e en local.
- Push, PR y merge (squash) solo con permiso del usuario; el repositorio no admite auto-merge.

## Modo de trabajo

- **TDD:** desactivado. Fuente: `openspec/config.yaml` (`strict_tdd: false`). Runner: `go test`.
- **RDD:** apagado salvo que `axiom review mode status` diga lo contrario.
- **Entrega:** `ask-on-risk`. Previsión: menos de 150 líneas de autoría, un solo PR.

## Tareas

- [x] **T1 · Explorar** — Ruta: en línea. Evidencia arriba. Sin commit.
- [x] **T2 · Guarda y pruebas condicionadas** — Ruta: delegada (escritor; disparador: 2+ ficheros no triviales).
  - Guarda en `internal/testenv` con su prueba unitaria; llamarla en cada punto de la lista.
  - Comprobaciones: `gofmt -l`, `go vet` y `go test` de los paquetes tocados **sin** la variable (las pruebas condicionadas deben salir como `SKIP` y sin compilar binarios); `go test ./internal/testenv`.
  - **Resultado (2026-10-27):** `testenv.RequireBinaryBuilds` y `testenv.BinaryBuildsEnabled` (valores `1` o `true`). `bench/` es un módulo Go aparte y no puede importar `internal/testenv`: lleva una copia local de la guarda. En `e2e/organicruntime`, `TestMain` ya no compila sin la variable y cada prueba que necesita el binario se omite por separado; si se pasa un binario ya hecho, siguen corriendo.
  - Evidencia (sin la variable): `gofmt -l` vacío; `go vet` limpio; `go test ./internal/testenv` → ok; las pruebas condicionadas de app, update, opencode, gga, bench y e2e → `SKIP`; `go test ./e2e/organicruntime` → ok; `go test ./internal/update ./internal/opencode ./internal/components/gga` → ok. Comprobación del padre: `go test ./internal/app -run TestTUIExecuteWithBackgroundPreservesConcurrentCLIStateMutation -v` → `SKIP`.
  - `go test ./internal/app` → `FAIL` solo en `TestTuiInstallOnThenSyncPreservesAndRefreshesOpenCodeActivation` («OpenCode runtime version unavailable or unsupported»). El escritor lo reproduce también sin los cambios (`git stash`), así que ya existía antes y no forma parte de este ODD.
- [x] **T3 · Activar en CI y documentar** — Ruta: delegada (mismo escritor).
  - `AXIOM_TEST_BUILD_BINARIES: "1"` en los *workflows* con `go test`; nota en la documentación de pruebas.
  - **Resultado (2026-10-27):** la variable se fija a nivel de *workflow* en `ci.yml` y `windows-full-suite.yml`. La nota va en `docs/testing/windows-tests.md`, en inglés como el resto del documento.
  - **Seguimiento (decisión del usuario, 2026-10-29):** `release.yml` y `promote-stable-rc.yml` no se han tocado. `internal/releasepolicy/policy.go` (`expectedReleaseWorkflowYAML`) compara el *workflow* de release con un YAML fijo, y añadir `env:` rompe `internal/update` («release workflow key set changed»). Mientras tanto, el `go test ./...` previo a release y a promote omite estas pruebas. Riesgo bajo: `ci.yml` corre en `pull_request` y en `push` a `main`, así que el commit etiquetado ya pasó la suite completa. Este PR se entrega sin tocarlos y la ampliación queda como trabajo aparte.
  - RDD: `off (decided by default)`, sin evaluación de riesgo.
- [ ] **T4 · Entregar** — push y PR con permiso del usuario; CI en verde, con las pruebas condicionadas ejecutadas (no omitidas).
  - 2026-10-29: rama rebasada sobre `origin/main` (`c780061a`) sin conflictos. Sobre ese árbol y sin la variable: `gofmt -l` vacío; `go vet` limpio; `go test` de `internal/testenv`, `internal/update`, `internal/opencode`, `internal/components/gga` y `e2e/organicruntime` → ok; las pruebas de `internal/app` que compilan binarios → `SKIP`. `main` no ha añadido nuevos puntos que compilen binarios.
  - Push y PR autorizados por el usuario (2026-10-29). Falta: CI en verde con las pruebas condicionadas ejecutadas.

## Criterios de aceptación

- `go test` local de los paquetes afectados no deja ningún `.exe` compilado en `%TEMP%` y marca esas pruebas como `SKIP`.
- En la CI esas pruebas se ejecutan y pasan.

## Progreso

- 2026-10-27: T1 hecha; documento abierto.
- 2026-10-27: T2 y T3 hechas, salvo los *workflows* de release.
- 2026-10-29: release y promote pasan a seguimiento; rebase sobre `main`, push y PR (T4 en curso).

## Siguiente paso

Confirmar en la CI del PR que las pruebas condicionadas se ejecutan (no `SKIP`) y pasan; cerrar T4 tras el merge. Seguimiento aparte: fijar la variable en `release.yml` y `promote-stable-rc.yml` junto con `internal/releasepolicy/policy.go`.
