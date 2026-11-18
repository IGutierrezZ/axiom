# Delta para axiom-sdd-cli-integration

> Parte del estado vigente: el resto de requisitos NO cambia. En `monorepo-embedded` el único rol es `fullstack` (REQ-2.2 de `workspace-topology`): el escenario de archivado desplegado en una rama de rol pasa a una rama ajena, sin cambiar lo que prueba.

## MODIFIED Requirements

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
- **DADO** un monorepo con el cambio `x` fusionado en la fase archive y el repositorio desplegado, por separado, en `feat/x`, en `main` y en una rama ajena `feat/otra`
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

