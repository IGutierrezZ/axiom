# ODD: `done` tolera el cerrojo local de la compuerta (done-cerrojo-local)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/done-cerrojo-local.md`. Espejo Engram: `odd/done-cerrojo-local/tasks`.
> Rama: `fix/done-cerrojo-local` · worktree `C:\repos\axiom-wt\done-cerrojo-local\axiom`. Seguimiento de C-07-b (PR #285 y #287).

Tipo: bug

## Objetivo

Que `axiom branch done` no se niegue con `dirty_worktree` por el `.gates.lock` vacío y sin versionar que deja `axiom sdd gate record` en la carpeta del cambio o en la archivada.

## Problema

`gate record` toma un cerrojo consultivo en `<carpeta del cambio>/.gates.lock` (`internal/kickoff/ledger.go:96`) y el fichero persiste: es un cerrojo de fichero y borrarlo tras liberarlo abriría carreras entre procesos. Es estado local con punto y nunca se publica (REQ-27.5), pero `done` cuenta todo fichero sin seguimiento como árbol sucio (`internal/branch/engine/done.go:338`). Tras registrar la compuerta `integration` en la rama de archivado (paso obligatorio del archivado en `monorepo-embedded`), `done` se niega siempre: ocurrió al cerrar C-07-b y hubo que borrar el fichero a mano.

## Por qué

El archivado documentado (skill `sdd-archive`, sección `monorepo-embedded`) termina siempre en ese rechazo. El cerrojo no contiene datos: descartarlo no pierde nada.

## Decisiones

- `done` excluye del árbol sucio los ficheros `.gates.lock` sin datos. Con solo cerrojos, el worktree se retira con `git worktree remove --force`. El resto de la regla de árbol limpio no cambia: la cola `.integrations-outbox.yaml` pendiente sigue bloqueando (REQ-43.31) y cualquier otro fichero sin confirmar también.
- El nombre sale de `kickoff.GatesLockFileName` (exportado), sin duplicar la constante.
- En `in_place`, el cerrojo no bloquea el cambio de rama: un fichero sin seguimiento sobrevive al checkout y no se borra.
- No se cambia `gate record`: borrar el cerrojo tras liberarlo rompería la exclusión entre procesos.

## Alcance autorizado

- `internal/branch/engine/done.go` y su test.
- `internal/kickoff/ledger.go` (exportar la constante).
- Delta MODIFIED de REQ-25.17 en `## Spec viva` e integración con `axiom archive integrate --odd done-cerrojo-local`.

## Modo TDD

Desactivado: `axiom.yaml` no declara TDD. Aun así, se observa el test en rojo antes del arreglo. Runner: `go test` (sin binarios sueltos).

## Tareas

- [x] **T1 · `done` tolera el cerrojo local** (ruta: en línea; unidad de menos de 100 líneas)
  - Test en `internal/branch/engine`: rama de archivado fusionada con solo `.gates.lock` sin seguimiento en la carpeta archivada → `done` la elimina. Con `.gates.lock` y otro fichero sin confirmar → `dirty_worktree` que nombra el otro fichero.
  - Arreglo en `done.go` y exportación de `GatesLockFileName`.
  - Comprobación: `go test ./internal/branch/... ./internal/kickoff/...`.
- [x] **T2 · Spec viva de REQ-25.17** (ruta: en línea)
  - Delta MODIFIED en `## Spec viva` e integración con `axiom archive integrate --odd done-cerrojo-local`.
  - Comprobación: la integración termina sin rechazo y `openspec/specs/change-branch-lifecycle/spec.md` refleja el texto.

## Criterio de terminado

`done` elimina la rama de archivado cuando el único fichero sin confirmar es `.gates.lock`, y sigue negándose con cualquier otro fichero, incluida la cola pendiente. `go test ./internal/branch/... ./internal/kickoff/...` en verde.

## Entrega

Una sola PR (`single-pr`); previsión de unas 150 líneas. Push, PR y fusión los decide el usuario.

## Progreso

- T1: test `TestDoneOnTheArchiveBranchIgnoresTheLocalGatesLock` en rojo antes del arreglo (el rechazo nombraba `.gates.lock` y `done` se negaba con solo el cerrojo); en verde después, junto con todos los `TestDone` de `internal/branch/engine` (incluido el de la cola pendiente, REQ-43.31).
- T2: `axiom archive integrate --odd done-cerrojo-local` compuso `change-branch-lifecycle` sin rechazo y regeneró `openspec/INDEX.md`.

## Spec viva

### Capacidad: change-branch-lifecycle

```openspec
## MODIFIED Requirements

### Requirement: `done`: Seguridad y Eliminación de Worktree y Rama Local (REQ-25.17)

Para cada rama cuya fusión esté confirmada por la detección por capas de REQ-25.16, `done` DEBE eliminar su worktree (`git worktree remove`), después eliminar la rama local (también cuando git no la considere fusionada, como ocurre tras un squash, porque la fusión ya está confirmada por las capas) y actualizar el registro local. Si la eliminación del worktree falla, NO DEBE eliminarse la rama. `done` DEBE rechazar la operación sobre una rama cuya fusión no esté confirmada contra el destino que corresponde a su modo (`merge_not_confirmed`, REQ-25.49, nombrando ese destino) y sobre un worktree con cambios sin confirmar (`dirty_worktree`), salvo `--force`. El cerrojo de la compuerta que deja `axiom sdd gate record` en la carpeta del cambio o en la archivada (`.gates.lock`, un fichero vacío y sin seguimiento, REQ-27.5) NO DEBE contar como cambio sin confirmar ni aparecer en el rechazo; si es lo único sin confirmar, `done` DEBE retirar el worktree descartándolo. Un `.gates.lock` con contenido sí cuenta como cambio sin confirmar. Con `--force`, DEBE saltarse la exigencia de fusión confirmada y la de árbol limpio, y DEBE informar en el JSON de los cambios sin confirmar descartados y, si la rama no estaba fusionada, de la punta (SHA) que tenía la rama eliminada para poder recuperarla. `done` NUNCA DEBE tocar ramas remotas ni ejecutar operaciones remotas (REQ-25.1). Si queda abierto un linaje RDD asociado al worktree, `done` DEBE emitir un aviso que nombre el linaje y la acción (`axiom review abandon`), sin impedir la operación. Cada rama del cambio, incluida la de archivado, DEBE procesarse por separado, sin selector de rol en esta versión: las confirmadas se eliminan y las no confirmadas se conservan y se reportan, y el código de salida DEBE ser distinto de cero si alguna rama no pudo procesarse.

En modo `in_place` no hay worktree que eliminar: `done` DEBE devolver el checkout a la rama de integración (también en modo de promoción `branch`) y, después, eliminar la rama local, y solo si el árbol está limpio. En `in_place`, `--force` solo salta la exigencia de fusión confirmada: NUNCA salta la de árbol limpio, de modo que con cambios sin confirmar `done` rechaza con `dirty_worktree` sin cambiar de rama ni eliminar nada.
(Previously: el `.gates.lock` vacío que deja `gate record` contaba como cambio sin confirmar y `done` se negaba con `dirty_worktree` tras archivar)

#### Scenario: Eliminación tras una fusión confirmada
- **DADO** una rama `feat/x` fusionada, su worktree con el árbol limpio y una rama remota `origin/feat/x`
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** el worktree se retira del disco y de `git worktree list`
- **Y** la rama local `feat/x` se elimina
- **Y** la rama remota `origin/feat/x` permanece intacta y no consta ninguna operación remota

#### Scenario: Una rama integrada con squash se elimina tras confirmarse por `equivalent_tree`
- **DADO** `feat/x` integrada en `main` mediante un squash, confirmada por la capa `equivalent_tree` (REQ-25.16)
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se eliminan su worktree y la rama local aunque git no la considere fusionada

#### Scenario: La rama de archivado se elimina como las demás
- **DADO** la rama `feat/x-archive` fusionada en la rama de integración y su worktree con el árbol limpio
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se eliminan el worktree y la rama local `feat/x-archive`
- **Y** no se toca ninguna rama remota

#### Scenario: Árbol sucio impide la eliminación
- **DADO** una rama fusionada cuyo worktree tiene un fichero modificado sin confirmar
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `dirty_worktree`
- **Y** el worktree y la rama local permanecen intactos

#### Scenario: El cerrojo vacío de la compuerta no ensucia el árbol
- **DADO** la rama de archivado fusionada cuyo worktree solo tiene sin confirmar el `.gates.lock` vacío de la carpeta archivada
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se eliminan el worktree y la rama local `feat/x-archive`
- **Y** con otro fichero sin confirmar junto al cerrojo, el rechazo `dirty_worktree` nombra ese fichero y no el cerrojo

#### Scenario: Rama sin fusionar impide la eliminación
- **DADO** una rama con commits propios sin fusionar y sin evidencia, con su worktree limpio
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `merge_not_confirmed`
- **Y** el worktree y la rama local permanecen intactos

#### Scenario: `--force` elimina con fusión no confirmada y avisa de lo descartado
- **DADO** una rama sin fusionar y su worktree con cambios sin confirmar
- **CUANDO** se ejecuta `axiom branch done --change x --force`
- **ENTONCES** se eliminan el worktree y la rama local
- **Y** el JSON informa de que se descartaron cambios sin confirmar, indica la punta (SHA) de la rama eliminada y la capa de confirmación es el token `forced`
- **Y** no se toca ninguna rama remota

#### Scenario: Linaje RDD abierto genera un aviso
- **DADO** un worktree con un linaje RDD abierto
- **CUANDO** se ejecuta `axiom branch done --change x` sobre una rama fusionada
- **ENTONCES** la salida incluye un aviso que nombra el linaje y `axiom review abandon`
- **Y** la eliminación se realiza

#### Scenario: Ramas de rol y de seguimiento se procesan por separado
- **DADO** `feat/x-core` fusionada en `feat/x` y `feat/x` aún sin fusionar en `main`
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** el worktree y la rama local `feat/x-core` se eliminan
- **Y** el worktree y la rama local `feat/x` se conservan con `merge_not_confirmed`
- **Y** el código de salida es distinto de cero

#### Scenario: `in_place` con árbol limpio vuelve a la integración y elimina la rama local
- **DADO** un cambio materializado en modo `in_place`, desplegado en `feat/x`, fusionado y con el árbol limpio
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** el repositorio queda desplegado en la rama de integración
- **Y** la rama local `feat/x` se elimina
- **Y** no se toca ninguna rama remota

#### Scenario: `in_place` con árbol sucio se rechaza incluso con `--force`
- **DADO** un cambio en modo `in_place`, desplegado en `feat/x`, con un fichero modificado sin confirmar
- **CUANDO** se ejecuta `axiom branch done --change x` y después `axiom branch done --change x --force`
- **ENTONCES** ambas ejecuciones se rechazan con `dirty_worktree`
- **Y** el repositorio sigue desplegado en `feat/x` con la rama y el fichero intactos

#### Scenario: `in_place` sin fusión confirmada se rechaza
- **DADO** un cambio en modo `in_place`, con el árbol limpio y la rama `feat/x` sin fusionar
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `merge_not_confirmed` sin cambiar de rama

#### Scenario: El rechazo en modo `branch` nombra la rama principal
- **DADO** un repositorio con `main_branch: master`, modo `branch` y `feat/x` fusionada solo en la integración
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** el rechazo `merge_not_confirmed` nombra `master` como destino y no `develop`
- **Y** el worktree y la rama local permanecen intactos

#### Scenario: `in_place` en modo `branch` vuelve a la integración al borrar
- **DADO** un cambio en `in_place`, modo `branch`, con `feat/x` fusionada en `master` y el árbol limpio
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** el repositorio queda desplegado en la rama de integración y se elimina la rama local `feat/x`
```
