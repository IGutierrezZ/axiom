# ODD: `done` rechaza el nombre de la rama de archivado como cambio

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/done-rechaza-archive.md`.
> Espejo de recuperación en Engram: topic `odd/done-rechaza-archive/tasks`, proyecto `axiom`.
> Rama `fix/done-rechaza-archive`, worktree `C:\repos\axiom-wt\done-rechaza-archive\axiom`.
> Carril ODD elegido por la persona (el proyecto tiene `workflow.lane: ask`).

Tipo: bug

## Objetivo

Que `axiom branch done --change <x>-archive` no pueda dejar huérfano el registro local de `<x>`: si el registro guarda el cambio `<x>` y no `<x>-archive`, `done` rechaza la orden sin tocar nada y propone `done --change <x>`.

## Problema y porqué

- El tablero de Incrementos mostraba once avisos «el worktree … figura en el registro local pero ya no existe o no es accesible» (`internal/branch/engine/listing.go`).
- Ocho de ellos eran cambios archivados cuyo registro (`<git common dir>/axiom/branch/v1/sdd/<x>.json`) solo guardaba la rama `inc/<x>-archive`. Junto a ellos había ficheros `<x>-archive.lock`: el archivo se había cerrado con `done --change <x>-archive`.
- Con ese nombre, `done` planifica un cambio ficticio `<x>-archive` cuya rama del cambio coincide con la de archivado de `<x>`: retira su worktree y su rama, pero actualiza el registro inexistente `<x>-archive.json` y deja `<x>.json` apuntando a un worktree que ya no está.
- El cierre correcto es `done --change <x>` tras fusionar la PR de archivo: `done` procesa la rama de archivado que guarda el registro (RF-BRA-18, escenario «La rama de archivado se elimina como las demás»).

## Alcance

1. `internal/branch/done_refusal.go`: rechazo `invalid_name` para el nombre de la rama de archivado, con la continuación `axiom branch done --change <x> --cwd <dir>`.
2. `internal/branch/engine/done.go`: tras reunir los hechos y antes de planificar, en el carril SDD y fuera de `multirepo`, si no hay registro de `<x>-archive` y sí de `<x>`, se rechaza.
3. Guía local del repositorio (`CLAUDE.md`, `AGENTS.md`, `GEMINI.md`): `done <slug>` también tras la PR de archivo, nunca `done <slug>-archive`.

## Decisiones

- **Rechazo acotado al caso dañino.** Solo se rechaza cuando el registro guarda `<x>` y no `<x>-archive`. Un cambio que se llame de verdad `<algo>-archive` y tenga su propio registro sigue funcionando; un nombre `-archive` sin registro de `<x>` no deja huérfano nada.
- **Código `invalid_name`.** Como el rol `archive` (S-15), el nombre colisiona con la rama de archivado; no se crea un código nuevo.
- **Solo carril SDD y fuera de `multirepo`.** ODD no tiene archivo y en `multirepo` no existe rama de archivado.
- **El aviso del tablero no cambia.** Sigue nombrando la entrada huérfana; los registros huérfanos existentes se borraron a mano (estado local derivado).

## Restricciones

- Pruebas localizadas: `go test ./internal/branch/engine -run 'TestDone'` y `go test ./internal/branch -run 'Done'`; la suite completa, en CI.
- TDD: desactivado (sin configuración de proyecto ni de sesión); se ejecutan las comprobaciones funcionales.
- Forecast: unas 150 líneas de autor. Entrega `single-pr`.

## Tareas

- [x] **T1 · Rechazo en `done` con su prueba y la guía local** — ruta: inline (unidad pequeña, sin disparadores de delegación).
  - `branch.ArchiveBranchNameAsChange` en `done_refusal.go` y `Engine.archiveBranchNameRefusal` en `done.go`, tras reunir los hechos y antes de planificar.
  - Pruebas en `done_test.go`: `TestDoneRefusesTheArchiveBranchNameOfARegisteredChange` (rechazo `invalid_name` sin escrituras de git, continuación `done --change x`, registro de `x` intacto, sin registro de `x-archive`; después `done --change x` elimina la rama y borra el registro) y `TestDoneProcessesARegisteredChangeWhoseNameEndsInArchive`.
  - Línea de guía en `CLAUDE.md`, `AGENTS.md` y `GEMINI.md`.
  - Evidencia: `go vet ./internal/branch/engine ./internal/branch` limpio; `go test ./internal/branch/engine -run 'TestDone|TestBranchDone'` ok (182 s); `go test ./internal/branch -run 'Done|Refusal|Archive'` ok; `gofmt -l internal/branch` vacío. Commit `ff985bbc7`.
- [x] **T2 · Integración de la spec viva y cierre** — ruta: inline.
  - `axiom archive integrate --odd done-rechaza-archive`: `prov:BRA:1` → `RF-BRA-47`, compuesto en `openspec/specs/change-branch-lifecycle/spec.md` y en `openspec/INDEX.md` (634 requisitos).
  - Las pruebas de T1 citan ya `RF-BRA-47`.

## Criterios de aceptación

- `done --change <x>-archive` con registro de `<x>` y sin registro de `<x>-archive` sale con 1, `invalid_name`, sin eliminar nada ni escribir el registro.
- La continuación es `axiom branch done --change <x> --cwd <dir>`.
- Los demás comportamientos de `done` no cambian (pruebas de `TestDone*` en verde).

## Spec viva

### Capacidad: change-branch-lifecycle

```openspec
## ADDED Requirements

### Requirement: `done` no acepta el nombre de la rama de archivado como cambio (RF-BRA-47)

En el carril SDD y fuera de la topología `multirepo`, `axiom branch done --change <x>-archive` DEBE rechazarse con `invalid_name` cuando el registro local guarda el cambio `<x>` y no guarda ningún cambio `<x>-archive`: ese nombre es el de la rama de archivado de `<x>`, que `done --change <x>` procesa junto a las demás ramas del cambio (RF-BRA-18). El rechazo NO DEBE eliminar ningún worktree ni ninguna rama, NO DEBE escribir el registro local y su continuación DEBE ser la orden `axiom branch done --change <x>` con la misma carpeta de partida. Un cambio cuyo nombre termina en `-archive` y tiene su propio registro DEBE procesarse como cualquier otro.

#### Scenario: El nombre de la rama de archivado se rechaza y se indica el del cambio
- **DADO** un registro local del cambio `x` que guarda la rama de archivado `feat/x-archive`, sin registro de `x-archive`
- **CUANDO** se ejecuta `axiom branch done --change x-archive`
- **ENTONCES** se rechaza con `invalid_name` y código de salida 1
- **Y** la continuación es `axiom branch done --change x` con la misma carpeta de partida
- **Y** la rama `feat/x-archive`, su worktree y el registro de `x` permanecen intactos

#### Scenario: Un cambio con nombre terminado en `-archive` y registro propio se procesa
- **DADO** un cambio `y-archive` con su propio registro local y su rama fusionada
- **CUANDO** se ejecuta `axiom branch done --change y-archive`
- **ENTONCES** `done` procesa sus ramas como las de cualquier otro cambio
```

## Progreso

- 2026-11-27: documento creado; worktree preparado con `scripts/axiom-worktree.ps1 new done-rechaza-archive -Lane odd -Prefix fix`.

- 2026-11-27: T1 hecha y verificada (`ff985bbc7`).
- 2026-11-27: T2 hecha: spec viva integrada como RF-BRA-47.

## Siguiente paso

PR con `scripts/axiom-worktree.ps1 pr done-rechaza-archive -Lane odd -Label bug`, a decisión de la persona; tras el merge, `scripts/axiom-worktree.ps1 done done-rechaza-archive -Lane odd`.

## Identificadores asignados

```yaml
schema: axiom.requirement-ids/v1
pattern: RF-<DOM>-<NN>
assignments:
  - provisional: prov:BRA:1
    definitive: RF-BRA-47
    capability: change-branch-lifecycle
    requirement: '`done` no acepta el nombre de la rama de archivado como cambio (RF-BRA-47)'
```
