# Delta para Ciclo de Vida de Archive, Integración Formal y Sellado Inmutable (INC-25)

> **Incremento:** `inc-25-ramas-y-worktrees-por-topologia` · **Tipo:** delta sobre la capacidad viva `sdd-archive-lifecycle`
> **Idioma:** Español (castellano peninsular)

Cambios de este delta:

- REQ-21.16: conserva su texto original con solo dos cambios: la rama de destino pasa de «la rama principal» a la rama de integración configurada (`git.integration_branch`) y se retira `monorepo-decoupled` de la verificación de `pr_merged`. No lleva ninguna excepción.
- REQ-25.39 (nuevo): en monorepo el archive se ejecuta en la rama de archivado `<prefijo>/<cambio>-archive` y se entrega en un PR documental aparte, después de fusionar el PR del cambio.

> Decisión D16 (2026-10-17): sustituye a la decisión anterior de archivar en la rama del cambio sin exigir un PR fusionado, que queda retirada. En monorepo el archivado va en un PR documental aparte y la precondición de REQ-21.16 se mantiene íntegra, con la evidencia `pr_merged` del PR del cambio. La rama de archivado es la decisión D18 (REQ-25.45 de `change-branch-lifecycle`).

## MODIFIED Requirements

### Requirement: Precondición de Integración o Despliegue para Ejecutar `archive` (REQ-21.16)

La fase `archive` NO DEBE ejecutarse como un paso inmediato tras la finalización de las pruebas locales de desarrollo, aunque `verify-report.md` (o el consolidado de REQ-21.15) tenga veredicto favorable. `archive` SOLO DEBE ejecutarse cuando exista evidencia de que el incremento se ha integrado o desplegado hacia un entorno preproductivo o productivo, mediante un PR formal fusionado hacia la rama de integración (`git.integration_branch`) u otro evento de despliegue equivalente ya configurado para el proyecto. Si se solicita `archive` sin esa evidencia, el orquestador DEBE rechazar la operación de forma explícita, indicando la precondición concreta que falta, y el cambio DEBE permanecer sin mover a `openspec/changes/archive/`. En la topología `multirepo`, la evidencia `pr_merged` se verifica contra el grafo git del repositorio de especificaciones (`workspaceRoot`); la integración de un repositorio de código se acredita con las evidencias `deployment` o `attestation`.
(Previously: el destino del PR era «la rama principal» y la verificación de `pr_merged` se enunciaba también para `monorepo-decoupled`, topología retirada)

#### Scenario: PR fusionado habilita el archivado
- **DADO** un cambio con `verify-report.md` en veredicto favorable y un PR fusionado hacia la rama de integración que integra ese incremento
- **CUANDO** se solicita `archive` para ese cambio
- **ENTONCES** la fase `archive` procede

#### Scenario: Archive solicitado sin integración ni despliegue se rechaza explícitamente
- **DADO** un cambio con `verify-report.md` en veredicto favorable pero sin ningún PR fusionado ni evidencia de despliegue hacia preproducción o producción
- **CUANDO** se solicita `archive` para ese cambio
- **ENTONCES** el orquestador rechaza la operación de forma explícita, señalando la ausencia de integración o despliegue como motivo
- **Y** el cambio permanece en `openspec/changes/<cambio>/` sin moverse a `openspec/changes/archive/`
- **Y** no se actualiza ninguna especificación viva de la baseline declarada (ni `openspec/specs/` ni los documentos de la baseline del proyecto)
- **Y** no se modifica ningún documento del contexto técnico

#### Scenario: La evidencia pr_merged en multirepo se verifica contra el repositorio de especificaciones
- **DADO** el workspace canónico, un commit fusionado en la rama de integración de `repo-specs` y un commit presente solo en `repo-backend`
- **CUANDO** se registra la evidencia `pr_merged` con el commit de `repo-specs`
- **ENTONCES** la evidencia se acepta
- **Y** registrada con el commit presente solo en `repo-backend`, se rechaza por no figurar en el grafo del repositorio de especificaciones

## ADDED Requirements

### Requirement: En Monorepo el Archive se Ejecuta en la Rama de Archivado y se Entrega en un PR Documental Aparte (REQ-25.39)

En los repositorios con rama de cambio (`monorepo-embedded`), `archive` DEBE ejecutarse en la rama de archivado `<prefijo>/<cambio>-archive` (REQ-25.45 de `change-branch-lifecycle`), que se crea con `axiom branch ensure --change <c> --archive` **después** de fusionar en la rama de integración el PR del cambio, y su resultado DEBE entregarse en un PR documental aparte contra la rama de integración. La precondición de `archive` es la de REQ-21.16, sin excepción: evidencia `pr_merged` del PR del cambio (verificada contra el grafo git del repositorio del planning home), veredicto favorable de verificación y compuertas aprobadas. Archivar en cualquier otra rama (la del cambio, la de integración, una de rol o una ajena) DEBE rechazarse: `axiom sdd status` proyecta `blocked(wrong_branch)` en la fase archive con la continuación exacta `axiom branch ensure --cwd <ruta> --change <c> --archive` (REQ-25.42 de `axiom-sdd-cli-integration`) y el escritor de archive se niega con `wrong_branch` (REQ-25.19), nombrando la rama esperada y sin mover el cambio. El PR documental contiene solo documentación: el movimiento del cambio a `openspec/changes/archive/`, la integración en la baseline declarada y los índices. Mientras el cambio no se haya archivado, el cambio fusionado figura como pendiente de archivar (REQ-25.20) sin bloquear nada. Tras fusionar el PR documental, `axiom branch done` elimina la rama de archivado (REQ-25.16 y REQ-25.17). En el repositorio de specs de un `multirepo` NO existe rama de archivado: `archive` se ejecuta in situ en la rama de integración, conserva la precondición de REQ-21.16 (evidencia `pr_merged` contra el repositorio de especificaciones, `deployment` o `attestation`) y no crea ramas ni worktrees (REQ-25.5). La skill `sdd-archive` DEBE indicar este orden y la rama.

#### Scenario: Archive en la rama de archivado con la evidencia del PR del cambio
- **DADO** un monorepo con el cambio `x` fusionado en la rama de integración, evidencia `pr_merged` del PR del cambio, `verify-report.md` favorable, compuertas aprobadas y el repositorio desplegado en `feat/x-archive`
- **CUANDO** se ejecuta `archive`
- **ENTONCES** `openspec/changes/archive/<fecha>-x/` existe en la rama `feat/x-archive` y `openspec/changes/x/` ya no existe en ella
- **Y** la integración en la baseline declarada y los índices se actualizan en esa rama
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

#### Scenario: Archivar en una rama de rol se rechaza
- **DADO** un cambio con los roles `core` y `qa`, fusionado, y el repositorio desplegado en `feat/x-core`
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
