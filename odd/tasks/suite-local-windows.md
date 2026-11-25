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
- [ ] T2. `backup` y `filemerge`: aislar del `TEMP` con nombres cortos 8.3. Ruta: inline.
- [ ] T3. `cli`: `TestCheckInstalledAssetVersion_*` y `TestCheckStateJSON_*`. Ruta: inline.
- [ ] T4. `app`: `TestTuiInstallOnThenSyncPreservesAndRefreshesOpenCodeActivation` y `TestDeferredSyncIncludesCodexPermissionsArgs`. Ruta: inline.
- [ ] T5. Orden local en `AGENTS.md` y verificación del criterio de terminado. Ruta: inline.

## Criterio de terminado

- En Windows local, `go test ./... -count=1` sin `internal/cli` ni `internal/reviewtransaction` termina sin fallos.
- En `internal/cli` solo fallan los tests atribuidos a RDD o GGA, listados aquí.
- El log del job `unit-tests` de la CI muestra que los cuatro tests del toolchain se ejecutan.

## Progreso y evidencias

(pendiente)
