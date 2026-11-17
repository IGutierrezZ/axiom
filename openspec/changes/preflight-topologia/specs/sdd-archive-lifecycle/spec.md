# Delta para sdd-archive-lifecycle

> Parte del estado vigente: el resto de requisitos NO cambia. En `monorepo-embedded` el único rol es `fullstack` (REQ-2.2 de `workspace-topology`), así que el escenario de archivado en una rama de rol se expresa con la rama del cambio.

## MODIFIED Requirements

### Requirement: En Monorepo el Archive se Ejecuta en la Rama de Archivado y se Entrega en un PR Documental Aparte (REQ-25.39)

En los repositorios con rama de cambio (`monorepo-embedded`), `archive` DEBE ejecutarse en la rama de archivado `<prefijo>/<cambio>-archive` (REQ-25.45 de `change-branch-lifecycle`), que se crea con `axiom branch ensure --change <c> --archive` **después** de fusionar en la rama de integración el PR del cambio, y su resultado DEBE entregarse en un PR documental aparte contra la rama de integración. La precondición de `archive` es la de REQ-21.16, sin excepción: evidencia `pr_merged` del PR del cambio (verificada contra el grafo git del repositorio del planning home), veredicto favorable de verificación y compuertas aprobadas. Archivar en cualquier otra rama (la del cambio, la de integración o una ajena; en `monorepo-embedded` no existen ramas de rol porque el único rol es `fullstack`) DEBE rechazarse: `axiom sdd status` proyecta `blocked(wrong_branch)` en la fase archive con la continuación exacta `axiom branch ensure --cwd <ruta> --change <c> --archive` (REQ-25.42 de `axiom-sdd-cli-integration`) y el escritor de archive se niega con `wrong_branch` (REQ-25.19), nombrando la rama esperada y sin mover el cambio. El PR documental contiene solo documentación: el movimiento del cambio a `openspec/changes/archive/`, la integración en `openspec/specs/` y los índices. Mientras el cambio no se haya archivado, el cambio fusionado figura como pendiente de archivar (REQ-25.20) sin bloquear nada. Tras fusionar el PR documental, `axiom branch done` elimina la rama de archivado (REQ-25.16 y REQ-25.17). En el repositorio de specs de un `multirepo` NO existe rama de archivado: `archive` se ejecuta in situ en la rama de integración, conserva la precondición de REQ-21.16 (evidencia `pr_merged` contra el repositorio de especificaciones, `deployment` o `attestation`) y no crea ramas ni worktrees (REQ-25.5). La skill `sdd-archive` DEBE indicar este orden y la rama.
(Previously: el PR documental contenía la integración «en la baseline declarada»; la baseline ajena se retira; la rama de rol deja de ser un caso de monorepo porque el único rol es `fullstack`, y el escenario de archivado en una rama de rol pasa a una rama ajena)

#### Scenario: Archive en la rama de archivado con la evidencia del PR del cambio
- **DADO** un monorepo con el cambio `x` fusionado en la rama de integración, evidencia `pr_merged` del PR del cambio, `verify-report.md` favorable, compuertas aprobadas y el repositorio desplegado en `feat/x-archive`
- **CUANDO** se ejecuta `archive`
- **ENTONCES** `openspec/changes/archive/<fecha>-x/` existe en la rama `feat/x-archive` y `openspec/changes/x/` ya no existe en ella
- **Y** la integración en `openspec/specs/` y los índices se actualizan en esa rama
- **Y** ningún fichero de código fuente cambia respecto de la rama de integración

#### Scenario: Sin la evidencia del PR del cambio el archive se rechaza aunque la rama sea la de archivado
- **DADO** un monorepo con el repositorio desplegado en `feat/x-archive`, verificación favorable y compuertas aprobadas, pero sin evidencia `pr_merged` del PR del cambio
- **CUANDO** se solicita `archive`
- **ENTONCES** se rechaza explícitamente señalando la ausencia de la evidencia de integración, conforme a REQ-21.16
- **Y** el cambio permanece en `openspec/changes/x/`

#### Scenario: Archivar en la rama del cambio se rechaza
- **DADO** un monorepo con el cambio `x` fusionado, evidencia `pr_merged`, verificación favorable y compuertas aprobadas, y el repositorio desplegado en `feat/x`
- **CUANDO** se solicita `archive`
- **ENTONCES** se rechaza con `blocked(wrong_branch)` y la continuación exacta `axiom branch ensure --cwd <ruta> --change x --archive`
- **Y** el cambio permanece en `openspec/changes/x/`

#### Scenario: Archivar en la rama de integración se rechaza
- **DADO** el mismo cambio y el repositorio desplegado en la rama de integración
- **CUANDO** se solicita `archive`
- **ENTONCES** se rechaza con `blocked(wrong_branch)` y la continuación exacta `axiom branch ensure --cwd <ruta> --change x --archive`
- **Y** el cambio no se mueve

#### Scenario: Archivar en una rama ajena se rechaza
- **DADO** un monorepo con el cambio `x` y el rol único `fullstack`, fusionado, y el repositorio desplegado en una rama ajena `feat/otra`
- **CUANDO** se solicita `archive`
- **ENTONCES** se rechaza con `blocked(wrong_branch)` nombrando `feat/x-archive` como rama esperada y la continuación exacta de `ensure --archive`

#### Scenario: Status proyecta el bloqueo de rama en la fase archive
- **DADO** un monorepo con el cambio `x` en la fase archive y el repositorio desplegado en `feat/x`
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** se proyecta `blocked(wrong_branch)` con la continuación exacta de `axiom branch ensure --change x --archive`

#### Scenario: Mientras no se archiva, el cambio fusionado figura como pendiente
- **DADO** un monorepo con el cambio `x` fusionado en la rama de integración y sin archivar
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** `x` figura entre los cambios pendientes de archivar
- **Y** ninguna orden se bloquea por ello

#### Scenario: Un PR documental rechazado deja el cambio pendiente de archivar
- **DADO** un monorepo con el cambio `x` archivado en `feat/x-archive` y su PR documental rechazado, sin fusionar
- **CUANDO** la persona descarta la rama de archivado
- **ENTONCES** la rama de integración sigue conteniendo `openspec/changes/x/` sin archivar
- **Y** `x` sigue figurando como pendiente de archivar y puede volver a archivarse con `axiom branch ensure --change x --archive`

#### Scenario: Multirepo archiva in situ y conserva la precondición
- **DADO** un multirepo con el repositorio de specs en su rama de integración y evidencia `pr_merged`, `deployment` o `attestation` aceptada
- **CUANDO** se ejecuta `archive`
- **ENTONCES** el archive se completa en la rama de integración del repositorio de specs
- **Y** no se crea ninguna rama ni worktree

#### Scenario: Multirepo sin evidencia sigue rechazando el archive
- **DADO** un multirepo con verificación favorable y compuertas aprobadas, sin evidencia `pr_merged`, `deployment` ni `attestation`
- **CUANDO** se solicita `archive`
- **ENTONCES** se rechaza explícitamente señalando la evidencia que falta
- **Y** el cambio permanece en `openspec/changes/<cambio>/`

#### Scenario: Multirepo no tiene rama de archivado
- **DADO** un multirepo con el repositorio de specs en su rama de integración
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se rechaza con `archive_branch_not_applicable`

#### Scenario: La skill de archive indica rama y orden
- **DADO** el texto de `sdd-archive/SKILL.md`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica crear la rama de archivado con `axiom branch ensure --change <c> --archive` tras fusionar el PR del cambio, ejecutar allí el archive y entregarlo en un PR documental aparte

