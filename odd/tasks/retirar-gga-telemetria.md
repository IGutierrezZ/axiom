# ODD: Retirar GGA y las lápidas de telemetría (retirar-gga-telemetria)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/retirar-gga-telemetria.md`. Espejo Engram: `odd/retirar-gga-telemetria/tasks`.
> Rama: `inc/retirar-gga-telemetria` · worktree `C:\repos\axiom-wt\retirar-gga-telemetria\axiom`. Cambio C-08-e del plan maestro (`docs/plan/backlog-2026-11/08-adelgazamiento.md`), ola A4; decisión P20 (coordinación).

## Objetivo

Quitar GGA (Gentleman Guardian Angel, el revisor IA previo al commit heredado de gentle-ai) y los restos de la telemetría ya retirada.

## Problema y porqué

Decisión del usuario: «No quiero telemetría, ni GGA». La telemetría no envía nada desde el 2026-10-07, pero quedan lápidas de retirada (`telemetryruntime`, `components/sdd/telemetry_retire.go`, sus llamadas y tests). GGA son unas 975 líneas de producción y 2.140 de test.

## Decisiones

- **P20 (usuario: «Sí, borrarlas ya»).** Sin lápidas: ni no-ops en `uninstall` ni comandos tumba. `install`, `sync`, `uninstall`, `doctor` y `upgrade` dejan de conocer GGA y la telemetría. Lo que Axiom escribió en equipos con instalaciones antiguas lo limpia la persona a mano: lista en `docs/releases/notas-retirada-gga-telemetria.md`.
- **Lector tolerante.** Un estado con `gga` en `components` sigue cargando: el componente se ignora con un aviso en `sync` y la siguiente escritura del estado ya no lo incluye (mismo patrón que los agentes retirados en C-08-d).
- **Spec viva.** Se retira la capacidad `gga` entera. La spec `axiom-distribution-identity` (REQ-20.12, ya marcado RETIRADO) no se toca.
- **Históricos.** No se tocan `docs/archive`, `openspec/changes/archive`, `docs/audits`, `docs/plan`, `docs/releases` existentes, `docs/upstream-absorption-ledger.md`, `CONTRIBUTORS.md` ni los `odd/tasks/` cerrados.
- **CI.** No se tocan `ci.yml` ni `windows-full-suite.yml` (C-08-i va en paralelo).

## Alcance autorizado

Lo que lista C-08-e en la sección 08: `internal/components/gga`, `internal/assets/gga`, tests `cli/gga_*` y `sync_gga_provider_test.go`; registro (`catalog`, `planner`, `model`); CLI (`run`, `sync`, `doctor`, `doctor_footprint`, `machine_footprint`); `installcmd`; `uninstall`; `versions`, `update` y `update/upgrade`; TUI (`preset`, `complete`, `upgrade_sync`, `model.go`); `app`; e2e; docs; y las lápidas de telemetría (`telemetryruntime`, `sdd/telemetry_retire.go`, `cli/opencode_telemetry_retirement_test.go` y sus llamadas).

Fuera de alcance: una instalación de GGA hecha fuera de Axiom; persona, presets y temas (C-08-f); workflows de CI (C-08-i).

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y no hay elección del usuario. Comprobaciones con `go vet` y `go test` de los paquetes tocados (`-run` en `cli`), sin la batería completa ni binarios sueltos de `axiom` (el EDR los borra). El lector tolerante se prueba con un test nuevo.

## Entrega

`single-pr`: una PR con un commit por tarea. Etiqueta `type:refactor`. Previsión: unas 5.000 líneas, casi todo borrado.

## Tareas

- [x] T1 · Borrar las lápidas de telemetría: `telemetryruntime`, `sdd/telemetry_retire.go`, el paso de OpenCode en `run`/`sync`, la limpieza de hooks y plugin en `uninstall`, y sus tests.
- [x] T2 · Retirar GGA del código y sus tests: componente, assets, registro, `installcmd`, `cli`, `uninstall`, TUI, `versions`, `update`, `update/upgrade` y `app`, con el lector tolerante del estado. (Se funde con la antigua T3: `update/upgrade` importaba el paquete `gga` y el commit no compilaría por separado.)
- [ ] T3 · Docs, e2e, notas de limpieza manual, `## Spec viva` integrada con `axiom archive integrate --odd retirar-gga-telemetria` y criterio `rg`.

Ruta: T1, en línea. T2, código de producción en línea y limpieza de unos 30 ficheros de test delegada en un escritor (disparador: proteger el contexto del padre; ~450 menciones mecánicas). T3, en línea salvo `PRD.md`, delegado en un segundo escritor en paralelo (disparador: paralelismo, superficie disjunta).

## Criterio de terminado

- `rg -il 'gga|guardian'` sin resultados fuera de los históricos y del lector tolerante.
- La instalación no ofrece GGA y `axiom doctor` no lo menciona.
- Un estado guardado con GGA carga con un aviso.
- La spec `gga` queda retirada.
- `go vet` y `go test` de los paquetes tocados en verde; la batería completa, en la CI de la PR.

## Spec viva

Retira la capacidad `gga` entera: Axiom deja de instalar, sincronizar, diagnosticar, desinstalar y actualizar GGA (P20, sin lápidas).

### Capacidad: gga

```openspec
## REMOVED Requirements

### Requirement: PowerShell Shim Asset

(Reason: C-08-e retira GGA; Axiom ya no escribe el shim `gga.ps1`.)

### Requirement: Windows Install Step

(Reason: C-08-e retira GGA; la instalación ya no tiene paso de GGA en ningún sistema.)

### Requirement: Non-Windows Systems Unaffected

(Reason: C-08-e retira GGA; sin componente GGA no hay flujo de Linux o macOS que preservar.)
```

## Progreso

- 2026-11-25: worktree creado desde `origin/main` `9ed8e03c`; documento creado.
- 2026-11-25: T1 hecha en `ce2fdf4e` (`go test` de `sdd`, `uninstall` y `cli -run OpenCode|Retire|Legacy|Sync` en verde).
- 2026-11-26: T2 hecha. `go vet ./...` limpio; `go test` en verde de `app`, `model`, `versions`, `installcmd`, `components/uninstall`, `components/mcp`, `update`, `update/upgrade`, `state`, `catalog`, `tui/screens`, `tui -run Preset|Complete|Uninstall|Model` y `cli -run Doctor|Footprint|OptionalTools|NormalizeInstallFlags|RunNotes|PostInstallNotes|PathGuidance|GoInstallPathNote|WorkspaceAgents|Sync|RunInstall|CheckOneTool|CheckToolBinaries`. Goldens `preset-custom-*` regenerados (solo pierden la fila de GGA). Tests nuevos: `state.TestReadIgnoresComponentsAxiomNoLongerSupports` y `catalog.TestModelKnownComponentsCoverCatalog`. El método genérico `InstallScript` de `update` queda sin herramientas que lo usen; se conserva con sus tests.

## Siguiente paso

T3: docs, e2e, notas y Spec viva.
