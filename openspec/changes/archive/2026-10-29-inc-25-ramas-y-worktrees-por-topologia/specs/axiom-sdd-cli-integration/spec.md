# Delta para Integración de Comandos SDD en la CLI axiom (INC-25)

> **Incremento:** `inc-25-ramas-y-worktrees-por-topologia` · **Tipo:** delta sobre la capacidad viva `axiom-sdd-cli-integration`
> **Idioma:** Español (castellano peninsular)

Cambios de este delta (solo requerimientos nuevos; REQ-13.1 a REQ-13.4 no cambian):

- REQ-25.42: `axiom sdd status` proyecta `blocked(wrong_branch)` como segundo motivo tipado, con la continuación exacta de `axiom branch ensure` (guarda B).
- REQ-25.43: las raíces que registra `axiom branch ensure` se proyectan en `allowedEditRoots` y las rutas de las tareas se resuelven respecto del worktree que corresponde.
- REQ-25.44: `axiom sdd status` y `axiom sdd continue` resuelven los cambios que viven en worktrees registrados.

> La semántica de `actionContext.allowedEditRoots` y de `planningHome` está definida en la capacidad `planning-home-resolution` (REQ-24.9), modificada también en este incremento. El contrato del status v2 sigue siendo solo aditivo (REQ-24.20): ninguna clave existente cambia.

## ADDED Requirements

### Requirement: `blocked(wrong_branch)` en `axiom sdd status` (REQ-25.42)

Cuando el veredicto de rama (REQ-25.18 de `change-branch-lifecycle`) para el cambio es `wrong_branch`, `axiom sdd status` DEBE proyectar el estado de aplicación como bloqueado con el motivo tipado `blocked(wrong_branch)`, el segundo motivo tipado junto a `blocked(edit_authority_missing)`. El motivo DEBE nombrar el repositorio, la rama actual y la rama esperada, e incluir la continuación exacta: la orden `axiom branch ensure` con `--cwd`, `--change` y, si procede, `--lane`. En un repositorio con rama de cambio (`monorepo-embedded`) se evalúa en toda fase: en la fase archive la rama esperada es la rama de archivado `<prefijo>/<cambio>-archive` y la continuación exacta es `axiom branch ensure --cwd <ruta> --change <c> --archive`, de modo que estar en la rama del cambio, en la de integración o en una de rol bloquea; en el resto de fases la rama esperada es la del cambio (o una de rol). En un repositorio de código de un `multirepo` solo se evalúa cuando el estado de aplicación evalúa unidades de trabajo que lo tocan (apply); en el repositorio de specs de un `multirepo` exige la rama de integración, también en la fase archive. La evaluación NO consulta si la rama de integración contiene cambios activos (REQ-25.20 de `change-branch-lifecycle`). NO se proyecta para cambios exentos (REQ-25.37 de `sdd-lane-kickoff`). Cuando concurren `wrong_branch` y `edit_authority_missing`, `wrong_branch` DEBE proyectarse primero, porque las raíces autorizadas dependen de la rama correcta (supuesto S-9). `axiom sdd continue` DEBE reflejar el mismo bloqueo sin autorizar ninguna fase siguiente. La evaluación DEBE ser de solo lectura y, si git no está disponible o la ruta no es un repositorio git, NO DEBE bloquear y DEBE emitir un aviso. El motivo NO lleva envoltorio de consentimiento: la continuación es una orden que el orquestador ejecuta, no una concesión.

#### Scenario: Status en la rama de integración de un monorepo
- **DADO** un cambio `x` con instantánea en el `kickoff.yaml` y el repositorio del planning home desplegado en `main`
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** el estado de aplicación es `blocked` con el motivo `blocked(wrong_branch)`
- **Y** el motivo nombra la rama actual `main`, la esperada `feat/x` y el repositorio
- **Y** incluye la continuación `axiom branch ensure --cwd <ruta absoluta> --change x`

#### Scenario: Status en la rama del cambio no bloquea
- **DADO** el mismo cambio con el repositorio desplegado en `feat/x`
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** no se proyecta `blocked(wrong_branch)`

#### Scenario: Se evalúa en toda fase en un monorepo
- **DADO** un cambio en fase `spec` con instantánea y el repositorio desplegado en `main`
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** se proyecta `blocked(wrong_branch)`

#### Scenario: En la fase archive solo la rama de archivado es correcta
- **DADO** un monorepo con el cambio `x` fusionado en la fase archive y el repositorio desplegado, por separado, en `feat/x`, en `main` y en `feat/x-core`
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** en los tres casos se proyecta `blocked(wrong_branch)` con la rama esperada `feat/x-archive`
- **Y** la continuación es `axiom branch ensure --cwd <ruta absoluta> --change x --archive`
- **Y** con el repositorio desplegado en `feat/x-archive` no se proyecta `blocked(wrong_branch)`

#### Scenario: En la fase archive del repositorio de specs de un multirepo se exige la integración
- **DADO** un multirepo en la fase archive con el repositorio de specs desplegado en `feat/x-core`
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** se proyecta `blocked(wrong_branch)` con la rama esperada `main` y sin continuación de `--archive`

#### Scenario: Los cambios activos en la rama de integración no bloquean
- **DADO** un cambio `y` fusionado y presente en la rama de integración, y un cambio `x` con instantánea y el repositorio desplegado en `feat/x`
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** no se proyecta `blocked(wrong_branch)`

#### Scenario: Un repositorio de código de un multirepo solo se evalúa en apply
- **DADO** un multirepo, un cambio con instantánea y `repo-backend` desplegado en `main`
- **CUANDO** se ejecuta `axiom sdd status x --json` en fase `spec`
- **ENTONCES** no se proyecta `blocked(wrong_branch)`
- **Y** en fase de aplicación con una unidad de trabajo que toca `repo-backend` se proyecta `blocked(wrong_branch)` nombrando `repo-backend`

#### Scenario: Un cambio exento no se bloquea
- **DADO** un cambio con `kickoff.yaml` sin instantánea y el repositorio desplegado en `main`
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** no se proyecta `blocked(wrong_branch)`

#### Scenario: `wrong_branch` precede a `edit_authority_missing`
- **DADO** un cambio con instantánea, el repositorio desplegado en `main` y una tarea que toca un repositorio sin autorizar
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** el motivo proyectado es `blocked(wrong_branch)`
- **Y** tras corregir la rama se proyecta `blocked(edit_authority_missing)`

#### Scenario: `axiom sdd continue` refleja el bloqueo
- **DADO** el mismo cambio bloqueado por rama
- **CUANDO** se ejecuta `axiom sdd continue x`
- **ENTONCES** no emite ninguna fase siguiente autorizada
- **Y** reporta `blocked(wrong_branch)` con la misma continuación

#### Scenario: Sin git no se bloquea
- **DADO** un proyecto fuera de un repositorio git
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** no se proyecta `blocked(wrong_branch)`
- **Y** el status incluye un aviso no bloqueante

#### Scenario: El contrato del status solo crece
- **DADO** el status de un cambio sin bloqueo de rama
- **CUANDO** se compara con el de antes de este incremento
- **ENTONCES** ninguna clave existente cambia de nombre, tipo ni valor

---

### Requirement: Las Raíces Registradas por `ensure` Entran en `allowedEditRoots` (REQ-25.43)

`actionContext.allowedEditRoots` DEBE incluir, además de `workspaceRoot` y las concesiones por cambio, las raíces absolutas de los repositorios y worktrees que `axiom branch ensure` registró para el cambio (en modo `in_place`, la raíz del propio repositorio). Las rutas de las unidades de trabajo de `tasks.md` y `tasks.<rol>.md` DEBEN resolverse respecto del worktree que corresponde al rol (el de la rama de rol para `tasks.<rol>.md` cuando existe, y el del cambio en otro caso) y reducirse a su raíz git. Un cambio cuyas tareas tocan solo sus propios worktrees NO DEBE quedar en `blocked(edit_authority_missing)`. Una raíz no registrada por `ensure` sigue exigiendo la concesión por cambio (`axiom sdd attempt grant`). Sin registro local para el cambio, el comportamiento DEBE ser el anterior a este incremento.

#### Scenario: Las tareas de un cambio en worktree no exigen autorización
- **DADO** un monorepo con el cambio `x` en un worktree registrado y un `tasks.md` cuyas unidades nombran ficheros del repositorio
- **CUANDO** se consulta `axiom sdd status x --json`
- **ENTONCES** `actionContext.allowedEditRoots` incluye la raíz absoluta del worktree
- **Y** `applyState` no es `blocked(edit_authority_missing)`

#### Scenario: La tarea de un rol se resuelve contra el worktree del rol
- **DADO** un cambio con los roles `core` y `qa` y un `tasks.core.md` cuyas unidades nombran `internal/core/a.go`
- **CUANDO** se consulta el status
- **ENTONCES** la ruta se resuelve contra el worktree de `feat/x-core`
- **Y** `allowedEditRoots` incluye la raíz de ese worktree

#### Scenario: Un repositorio no registrado exige concesión
- **DADO** un multirepo y una unidad de trabajo que toca un repositorio de código que `ensure` no registró para el cambio
- **CUANDO** se consulta el status
- **ENTONCES** se proyecta `blocked(edit_authority_missing)` nombrando ese repositorio
- **Y** el repositorio entra en `allowedEditRoots` solo tras `axiom sdd attempt grant`

#### Scenario: Un repositorio de código registrado por `ensure` entra en las raíces
- **DADO** un multirepo con el cambio `x` y `ensure` ejecutado, que registró el worktree de `repo-backend`
- **CUANDO** se consulta el status
- **ENTONCES** `allowedEditRoots` incluye la raíz del worktree de `repo-backend`

#### Scenario: Sin registro el comportamiento es el anterior
- **DADO** un cambio sin registro local
- **CUANDO** se consulta el status
- **ENTONCES** `allowedEditRoots` es `workspaceRoot` más las concesiones por cambio

---

### Requirement: `status` y `continue` Resuelven los Cambios de los Worktrees Registrados (REQ-25.44)

`axiom sdd status` y `axiom sdd continue` DEBEN resolver los cambios activos que viven en los worktrees registrados en el registro local de `axiom branch` (REQ-25.14), de modo que, al ejecutarse desde el checkout principal u otro directorio del proyecto, `axiom sdd status <cambio>` resuelva el cambio en su worktree. En ese caso `workspaceRoot`, `planningHome`, `changeRoot` y `artifactPaths` DEBEN referirse al worktree. Sin nombre de cambio, los candidatos DEBEN incluir los cambios activos de los worktrees registrados con el orden de preferencia vigente. Si un mismo cambio figura en el checkout principal y en un worktree, prevalece el del worktree; si figura en el worktree de la rama del cambio y en el de la rama de archivado, prevalece el de la rama de archivado, porque el cambio ya está fusionado y solo falta archivarlo (REQ-25.45 de `change-branch-lifecycle`). Una entrada del registro cuyo worktree ya no existe DEBE omitirse con un aviso. Con `--cwd` dentro de un worktree, el cambio se resuelve en su propio árbol (REQ-25.28 de `planning-home-resolution`).

#### Scenario: Status desde el checkout principal resuelve el worktree
- **DADO** un proyecto cuyo checkout principal solo contiene `archive/` y un cambio `x` activo en un worktree registrado
- **CUANDO** se ejecuta `axiom sdd status x --json --cwd <checkout principal>`
- **ENTONCES** `changeRoot` es `<worktree>/openspec/changes/x`
- **Y** `workspaceRoot` es la raíz del worktree

#### Scenario: Sin nombre de cambio se consideran los worktrees registrados
- **DADO** un proyecto con un único cambio activo `x`, que vive en un worktree registrado
- **CUANDO** se ejecuta `axiom sdd status --json --cwd <checkout principal>`
- **ENTONCES** el cambio activo resuelto es `x`

#### Scenario: El worktree prevalece sobre el checkout principal
- **DADO** un cambio presente en el checkout principal y en un worktree registrado
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el status corresponde al del worktree

#### Scenario: El worktree de la rama de archivado prevalece sobre el de la rama del cambio
- **DADO** el cambio `x` presente en el worktree de `feat/x` y en el de `feat/x-archive`, ambos registrados
- **CUANDO** se ejecuta `axiom sdd status x --json --cwd <checkout principal>`
- **ENTONCES** `changeRoot` es `<worktree de feat/x-archive>/openspec/changes/x`

#### Scenario: Un worktree desaparecido se omite con aviso
- **DADO** una entrada del registro local cuyo worktree no existe
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el status se devuelve sin ese cambio
- **Y** incluye un aviso que nombra la ruta ausente

#### Scenario: `continue` resuelve igual que `status`
- **DADO** el cambio `x` en un worktree registrado
- **CUANDO** se ejecuta `axiom sdd continue x --cwd <checkout principal>`
- **ENTONCES** la siguiente fase autorizada se calcula sobre el cambio del worktree
