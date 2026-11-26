# ODD: CI ágil (C-08-k)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/ci-agil.md`, en la rama `inc/ci-agil` (worktree `C:\repos\axiom-wt\ci-agil\axiom`, creado desde `origin/main` `3193c716` el 2026-11-26).
> Espejo de recuperación en Engram: topic `odd/ci-agil/tasks`, proyecto `axiom`.
> Fuente: plan del backlog 2026-11, revisión 10, sección 08, entrada C-08-k; P24 resuelta en el apartado 6.2.

## Objetivo

Que la CI no repita trabajo:

- ni la batería completa tras cada merge a `main`;
- ni los meta-tests del toolchain en cada PR que no los afecta;
- ni ejecuciones de `PR Validation` que no pueden cambiar el resultado.

## Decisiones (del plan; no se reabren)

- `ci.yml` deja `push` a `main` y gana una nocturna con la puerta de `windows-full-suite.yml`: salta si el último `headSha` en verde de `ci.yml` en `main` es el actual; si no puede leer el historial, ejecuta. Se conserva `workflow_dispatch`.
- `pr-check.yml` deja `edited`; `unlabeled` se queda.
- Los meta-tests que compilan o lanzan el toolchain salen de `Go Tests` (`-skip`) y van a un job propio (`-run`), sin tocar los ficheros de test. Corre en `schedule`, `workflow_dispatch` y en las PR que tocan sus entradas.
- Las guardas de la partición se amplían para que ningún meta-test quede fuera de todos los jobs.
- `Unit Tests` conserva el nombre y la semántica de agregador.

## Decisiones de implementación

- **Puerta nocturna en `changes`.** El job `changes` ya decide por evento; con `schedule` consulta `gh run list --workflow ci.yml --branch main --status success`. Si `main` no se ha movido, publica `up_to_date=true` y todos los jobs se omiten. Si se ha movido, el rango para Windows es `<último sha en verde>..<sha actual>`, igual que hacía el `push`. Sin historial legible, falla abierto.
- **Un único selector.** `TOOLCHAIN_META_TESTS` se define una vez en el `env` del workflow. `Go Tests` lo usa en `-skip` y el job nuevo en `-run`, así que no pueden divergir.
- **Paquetes del job nuevo.** Corre sobre `./internal/app ./internal/assets ./internal/update`. La guarda comprueba que los nombres del selector solo se declaran en esos paquetes; si no, un homónimo en otro paquete quedaría omitido en `Go Tests` sin correr en el job nuevo.
- **Entradas del job nuevo.** Las del plan (`.github/workflows/**`, `.goreleaser.yaml`, `internal/update/**`, `scripts/**`, la documentación de comandos y `cmd/axiom/**`) más los propios ficheros de los meta-tests, `go.mod` y `go.sum`: un cambio en el test o en la versión de Go cambia su resultado.
- **Entradas de documentación.** `docs/**/*.md` salvo `docs/audits/` y `internal/assets/**/*.md`: es lo que lee `TestDocumentedInvocationsRunAsDocumented`.
- **Agregador.** `Unit Tests` necesita el job nuevo y lo acepta omitido solo cuando `changes` terminó bien y dijo `toolchain=false`.
- **Familia de la política de release.** El plan nombra `TestReleaseDistributionPolicyAssertionFailsClosed`, pero la decisión es sacar «los meta-tests que compilan o lanzan el toolchain». Sus dos hermanos usan el mismo `newReleasePolicyFixture` (`go run ./internal/releasepolicycmd`): `TestReleaseDistributionPolicyAcceptsSemanticYAMLFormatting` y `TestModifiedReleaseVerifierCannotGainWriteAuthority`. Entran también; si no, la guarda nueva los señalaría.
- **Guarda nueva** (`TestToolchainMetaTestsRunInExactlyOneJob`, sin toolchain): el selector es único y lo usan los dos jobs; ningún nombre seleccionado se declara fuera de `META_PACKAGES`; todo test de esos paquetes que llama a `RequireGoToolchain`, a `newReleasePolicyFixture` o a `go build|run|test` está seleccionado. Comprobada con dos mutaciones (quitar un nombre; seleccionar uno de `internal/testenv`): ambas fallan.
- **Paso del job.** `-run` que no casa nada sale con 0, así que el paso exige al menos un `--- PASS: Test` y ningún `--- SKIP: Test` de primer nivel.
- **`TestBuiltBinaryClosedStdinRefusesBeforeBubbleTea`** (`internal/app/tty_boundary_test.go`) entra en el selector mientras exista. Si C-08-f lo borra, su nombre queda sin efecto en el selector: la guarda no exige que cada nombre exista, solo que alguno exista, para no obligar a C-08-f a tocar los workflows.

## Alcance autorizado

`.github/workflows/ci.yml`, `.github/workflows/pr-check.yml`, `internal/assets/workflow_shards_test.go`, `internal/assets/formatter_ordering_test.go`, `scripts/require-ci-success.sh`, `docs/testing/windows-tests.md`, `docs/release-signing.md` y este documento.

Fuera: `windows-full-suite.yml` y los jobs manuales.

## Tareas

Modo TDD: desactivado (`openspec/config.yaml`, `strict_tdd: false`). Ruta: todas inline (unidades pequeñas, sin disparador de delegación).

- [x] **T1.** `ci.yml`: sin `push`, con nocturna y puerta en `changes`; documentación de Windows actualizada.
- [x] **T2.** Job `Toolchain Meta Tests` con `-skip`/`-run`, agregador y guardas ampliadas.
- [x] **T3.** `pr-check.yml` sin `edited`.
- [x] **T4.** Release: mensaje de `require-ci-success.sh` y guía en `docs/release-signing.md`.

## Comprobaciones

- Localizadas: `go test ./internal/assets -run 'TestUnitTestShards|TestUnitTestsAggregator|TestRequiredChecksFailClosed|TestToolchainMeta' -count=1` y `go test ./internal/update -run TestRequireCISuccess -count=1`.
- La CI de la PR valida los workflows: debe aparecer `Toolchain Meta Tests` (la PR toca `.github/workflows/**`) y `Unit Tests` en verde.

## Tiempos

| Medida | Antes (ejecución 45725054482, PR) | Después (ejecución 45730104048, esta PR) |
|---|---|---|
| CI de PR hasta `Unit Tests` | 1:59 | 1:45 |
| `Go Tests (app-assets-update)` | 1:04 | 0:48 |
| `Go Tests (rest)` | 1:40 | 1:08 |
| `Toolchain Meta Tests` | (dentro de `Go Tests`) | 1:15, en paralelo; solo en PR que tocan sus entradas |
| Ejecuciones de `ci.yml` por merge a `main` | 1 (≈2:00 cada una) | 0; una nocturna si hay commits |

Esta PR toca sus entradas (`.github/workflows/**`), así que el job nuevo corrió: los 7 meta-tests pasaron (`7 toolchain meta-test(s) passed`). El camino crítico pasa a ser `Go Tests (cli)` (1:30). La diferencia de `rest` incluye la variación del runner.

## Evidencia

- Commits: `c4b92170` (T1), `e41c7e00` (T2), `192975f4` (T3), `621c5c07` (T4).
- Comprobaciones locales en verde: guardas de `internal/assets` (partición, agregador, formato, meta-tests) y `TestRequireCISuccessSelectsNewestExactCommitRun`, `TestReleaseDocumentationStatesArchiveDownloadCeiling`, `TestReleaseSecurityScriptsAreSyntacticallyValidAndFailClosed` de `internal/update`.
- Mutaciones de la guarda nueva: quitar `TestModifiedReleaseVerifierCannotGainWriteAuthority` del selector y seleccionar `TestRequireGoToolchainSkipsUnlessEnabled` (`internal/testenv`) fallan con el mensaje esperado.
- CI de la PR #322 en verde, con `Go Format` y `Unit Tests` presentes.
- Pendiente de observar tras el merge: la primera nocturna (corre si `main` se movió; salta si no).

## Progreso

- 2026-11-26: worktree creado; T1-T4 hechas; PR #322 abierta y en verde.

## Siguiente paso

Merge de la PR #322 y `done`.
