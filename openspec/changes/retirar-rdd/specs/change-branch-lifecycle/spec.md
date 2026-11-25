# Delta for change-branch-lifecycle

## MODIFIED Requirements

### Requirement: Registro Local en el Git Common Dir (REQ-25.14)

`ensure` DEBE mantener un registro local por cambio, fuera de cualquier árbol versionado, en el git common dir del repositorio del planning home, obtenido con `git rev-parse --git-common-dir` (nunca asumiendo `.git`). El registro DEBE contener las rutas absolutas de todos los repositorios, ramas (incluida la de archivado) y worktrees del cambio, la respuesta a `ask` del modo de trabajo (REQ-25.8) y, por cada repositorio con dos etapas, el modo de promoción resuelto y su origen (REQ-25.46). Los campos del modo de promoción DEBEN ser opcionales: un registro sin ellos sigue siendo válido y un binario anterior los ignora. El registro es la fuente operativa del modo de promoción en todos los carriles. Debe ser accesible con el mismo contenido desde el checkout principal y desde cualquier worktree enlazado del mismo repositorio. Si el registro se borra, el repositorio NO DEBE verse afectado y `ensure` DEBE reconstruirlo de forma idempotente a partir del estado de git (ramas y worktrees existentes) y, para el modo de promoción, de la instantánea del `kickoff.yaml` si la hay (REQ-25.46), mientras `status` DEBE informar que el cambio no tiene registro.
(Previously: el registro no guardaba el modo de promoción ni su origen y la reconstrucción no consultaba la instantánea)
(Previously: la cláusula sobre la coexistencia con el estado local de RDD (`<git common dir>/gentle-ai/`) y sus linajes; RDD se retira (C-01-e) y `ensure` ya no tiene nada que preservar de él.)

#### Scenario: El registro se resuelve con el git common dir
- **DADO** un repositorio cuyo directorio git no se llama `.git` (git dir enlazado)
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el registro se crea bajo el directorio devuelto por `git rev-parse --git-common-dir`

#### Scenario: El registro es visible desde un worktree enlazado
- **DADO** un cambio preparado por `ensure` en modo `worktree`
- **CUANDO** se ejecuta `axiom branch status --change x --cwd <ruta del worktree>`
- **ENTONCES** el resultado incluye las rutas absolutas de todos los repositorios y worktrees del cambio registrados desde el checkout principal

#### Scenario: Borrar el registro no daña el repositorio
- **DADO** un cambio preparado y su registro borrado
- **CUANDO** se ejecuta `ensure --change x` de nuevo
- **ENTONCES** las ramas y worktrees existentes se reutilizan sin modificarlos
- **Y** el registro se reconstruye con las mismas rutas absolutas

#### Scenario: Registro ausente informado por `status`
- **DADO** un cambio sin registro local
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON indica que el cambio no tiene registro y no falla

#### Scenario: El registro guarda el modo de promoción y su origen por repositorio
- **DADO** un multirepo con dos repositorios de código con dos etapas, uno resuelto por respuesta y otro por la política
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el registro contiene, por cada repositorio, su modo y su origen

#### Scenario: Un registro sin modo de promoción sigue siendo válido
- **DADO** un registro escrito antes de esta capacidad, sin los campos del modo
- **CUANDO** se ejecutan `status` y `ensure --change x`
- **ENTONCES** ambas órdenes lo leen sin error y no lo consideran corrupto

#### Scenario: `ensure` conserva el modo registrado al reescribir el registro
- **DADO** un cambio con el modo `branch` registrado para `repo-backend`
- **CUANDO** se ejecuta `ensure --change x --archive` o de nuevo `ensure --change x`
- **ENTONCES** el modo de `repo-backend` sigue siendo `branch` en el registro

### Requirement: `done`: Seguridad y Eliminación de Worktree y Rama Local (REQ-25.17)

Para cada rama cuya fusión esté confirmada por la detección por capas de REQ-25.16, `done` DEBE eliminar su worktree (`git worktree remove`), después eliminar la rama local (también cuando git no la considere fusionada, como ocurre tras un squash, porque la fusión ya está confirmada por las capas) y actualizar el registro local. Si la eliminación del worktree falla, NO DEBE eliminarse la rama. `done` DEBE rechazar la operación sobre una rama cuya fusión no esté confirmada contra el destino que corresponde a su modo (`merge_not_confirmed`, REQ-25.49, nombrando ese destino) y sobre un worktree con cambios sin confirmar (`dirty_worktree`), salvo `--force`. El cerrojo de la compuerta que deja `axiom sdd gate record` en la carpeta del cambio o en la archivada (`.gates.lock`, un fichero vacío y sin seguimiento, REQ-27.5) NO DEBE contar como cambio sin confirmar ni aparecer en el rechazo; si es lo único sin confirmar, `done` DEBE retirar el worktree descartándolo. Un `.gates.lock` con contenido sí cuenta como cambio sin confirmar. Con `--force`, DEBE saltarse la exigencia de fusión confirmada y la de árbol limpio, y DEBE informar en el JSON de los cambios sin confirmar descartados y, si la rama no estaba fusionada, de la punta (SHA) que tenía la rama eliminada para poder recuperarla. `done` NUNCA DEBE tocar ramas remotas ni ejecutar operaciones remotas (REQ-25.1). Cada rama del cambio, incluida la de archivado, DEBE procesarse por separado, sin selector de rol en esta versión: las confirmadas se eliminan y las no confirmadas se conservan y se reportan, y el código de salida DEBE ser distinto de cero si alguna rama no pudo procesarse.

En modo `in_place` no hay worktree que eliminar: `done` DEBE devolver el checkout a la rama de integración (también en modo de promoción `branch`) y, después, eliminar la rama local, y solo si el árbol está limpio. En `in_place`, `--force` solo salta la exigencia de fusión confirmada: NUNCA salta la de árbol limpio, de modo que con cambios sin confirmar `done` rechaza con `dirty_worktree` sin cambiar de rama ni eliminar nada.
(Previously: el `.gates.lock` vacío que deja `gate record` contaba como cambio sin confirmar y `done` se negaba con `dirty_worktree` tras archivar)
(Previously: si quedaba abierto un linaje RDD asociado al worktree, `done` emitía un aviso que nombraba el linaje y `axiom review abandon`; ese aviso desaparece con RDD (C-01-e) y ningún comando `axiom review` existe ya.)

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

#### Scenario: `done` no emite ningún aviso de linajes
- **DADO** un worktree de una rama fusionada en cuyo git common dir sobrevive un directorio `gentle-ai/` con restos de revisiones antiguas
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la salida no incluye ningún aviso sobre linajes ni cita `axiom review`
- **Y** la eliminación se realiza igual que sin ese directorio

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
