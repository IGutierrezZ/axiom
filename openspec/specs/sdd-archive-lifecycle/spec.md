<!-- Especificación Viva generada a partir de '2026-09-22-inc-21-upfront-flow-governance' -->

# Especificación de Requerimientos: Ciclo de Vida de Archive, Integración Formal y Sellado Inmutable (INC-21)

## Propósito

Definir de forma rigurosa, ejecutable y verificable los requerimientos funcionales y escenarios BDD para la fase terminal `archive` del ciclo de vida SDD: la precondición de integración o despliegue formal antes de ejecutarla, el contenido y sincronización de especificaciones vivas que debe efectuar, y la semántica de sellado inmutable post-archive que convierte cualquier corrección posterior en un ticket de bug o nuevo incremento.

---

## 1. Capacidad: `sdd-archive-lifecycle`

El archivado es la entrega formal del incremento a un entorno preproductivo o productivo, no un paso inmediato tras las pruebas locales; una vez ejecutado, el alcance del incremento queda congelado.

### Requirement: Precondición de Integración o Despliegue para Ejecutar `archive` (RF-ARC-01)

La fase `archive` NO DEBE ejecutarse como un paso inmediato tras la finalización de las pruebas locales de desarrollo, aunque `verify-report.md` (o el consolidado de RF-RCL-03) tenga veredicto favorable. `archive` SOLO DEBE ejecutarse cuando exista evidencia de que el incremento se ha integrado o desplegado hacia un entorno preproductivo o productivo, mediante un PR formal fusionado hacia la rama de integración (`git.integration_branch`) u otro evento de despliegue equivalente ya configurado para el proyecto. Si se solicita `archive` sin esa evidencia, el orquestador DEBE rechazar la operación de forma explícita, indicando la precondición concreta que falta, y el cambio DEBE permanecer sin mover a `openspec/changes/archive/`. En la topología `multirepo`, la evidencia `pr_merged` se verifica contra el grafo git del repositorio de especificaciones (`workspaceRoot`); la integración de un repositorio de código se acredita con las evidencias `deployment` o `attestation`.
(Previously: el escenario de rechazo citaba también los documentos de la baseline del proyecto; la baseline ajena se retira)

#### Scenario: PR fusionado habilita el archivado
- **DADO** un cambio con `verify-report.md` en veredicto favorable y un PR fusionado hacia la rama de integración que integra ese incremento
- **CUANDO** se solicita `archive` para ese cambio
- **ENTONCES** la fase `archive` procede

#### Scenario: Archive solicitado sin integración ni despliegue se rechaza explícitamente
- **DADO** un cambio con `verify-report.md` en veredicto favorable pero sin ningún PR fusionado ni evidencia de despliegue hacia preproducción o producción
- **CUANDO** se solicita `archive` para ese cambio
- **ENTONCES** el orquestador rechaza la operación de forma explícita, señalando la ausencia de integración o despliegue como motivo
- **Y** el cambio permanece en `openspec/changes/<cambio>/` sin moverse a `openspec/changes/archive/`
- **Y** no se actualiza ninguna especificación viva de `openspec/specs/`
- **Y** no se modifica ningún documento del contexto técnico

#### Scenario: La evidencia pr_merged en multirepo se verifica contra el repositorio de especificaciones
- **DADO** el workspace canónico, un commit fusionado en la rama de integración de `repo-specs` y un commit presente solo en `repo-backend`
- **CUANDO** se registra la evidencia `pr_merged` con el commit de `repo-specs`
- **ENTONCES** la evidencia se acepta
- **Y** registrada con el commit presente solo en `repo-backend`, se rechaza por no figurar en el grafo del repositorio de especificaciones

---

### Requirement: Contenido del Archivado (RF-ARC-02)

Al ejecutarse `archive` con su precondición satisfecha (RF-ARC-01), el sistema DEBE: (1) integrar los deltas del incremento en la spec viva `openspec/specs/`, sin crear nunca una segunda spec viva, asignando antes el número definitivo de los identificadores provisionales de todas las capacidades en un plan único, componiendo también las capacidades nuevas, reescribiendo el provisional en la spec viva, las líneas `Cubre:` de las tareas, las deltas del cambio y la matriz de los informes de verify, y escribiendo la traza del mapa de provisional a definitivo (RF-ARC-15); (2) con `technical_context.maintained` declarado, integrar el delta de contexto del cambio en las rutas mantenidas con el mismo mecanismo de edición dirigida con verificación nativa, registrando la traza por entrada (RF-ARC-04), sin modificar nunca las rutas del modo `reference` (RF-CTX-05); (3) transferir el incremento a `openspec/changes/archive/`; y (4) actualizar el inventario maestro (`openspec/INDEX.md`) y los roadmaps declarados que referencien el incremento. Las integraciones (1) y (2), incluida la asignación y la reescritura, DEBEN completarse y verificarse antes de la transferencia (3), porque un incremento archivado es inmutable (RF-ARC-03).
(Previously: la integración (1) solo componía los deltas por dominio; no asignaba identificadores provisionales, no reescribía las tareas ni las deltas, no dejaba traza y las capacidades nuevas se copiaban sin pasar por la composición nativa)

#### Scenario: Archivado exitoso actualiza incremento, especificación viva e inventario

- **DADO** un cambio cuya precondición de archivado ya está satisfecha
- **CUANDO** se ejecuta `archive`
- **ENTONCES** el incremento se transfiere a `openspec/changes/archive/`
- **Y** la especificación viva en `openspec/specs/` incorpora los requerimientos `ADDED`/`MODIFIED`/`REMOVED` del cambio
- **Y** `openspec/INDEX.md` y los roadmaps afectados reflejan el incremento como archivado

#### Scenario: Archivado con contexto mantenido integra el delta de contexto

- **DADO** un cambio con un delta de contexto con una entrada `modify` sobre un documento de `maintained.paths`
- **CUANDO** se ejecuta `archive`
- **ENTONCES** el documento se actualiza según el delta
- **Y** la traza por entrada consta en la carpeta del cambio y en el informe de archive
- **Y** ninguna ruta de `reference.paths` presenta cambios

#### Scenario: La integración se verifica antes de transferir el incremento

- **DADO** un cambio cuya integración falla en la composición o en la verificación nativa
- **CUANDO** se ejecuta `archive`
- **ENTONCES** el incremento permanece en `openspec/changes/<cambio>/` sin transferirse a `openspec/changes/archive/`

#### Scenario: La asignación y la traza se completan antes de transferir

- **DADO** un cambio con ADDED provisionales cuya precondición de archivado ya está satisfecha
- **CUANDO** se ejecuta `archive`
- **ENTONCES** la spec viva contiene identificadores definitivos y la traza del mapa consta en la carpeta archivada del cambio
- **Y** la carpeta se transfiere solo después de completarse la asignación y la reescritura

#### Scenario: Una asignación inválida deja el incremento sin transferir

- **DADO** un cambio con un MODIFIED cuyo identificador no existe en la spec viva
- **CUANDO** se ejecuta `archive`
- **ENTONCES** el incremento permanece en `openspec/changes/<cambio>/`
- **Y** ninguna spec viva ni ningún fichero del cambio cambia

---

### Requirement: Sellado Inmutable Post-Archive y Gestión Exclusiva vía Bug o Nuevo Incremento (RF-ARC-03)

Una vez archivado un incremento, su alcance queda formalmente congelado. El sistema NO DEBE reabrir un incremento ya archivado para aplicar un ajuste, un comportamiento anómalo o una regresión detectados con posterioridad. Cualquier corrección posterior DEBE gestionarse mediante un ticket de bug independiente o un nuevo incremento de evolución que referencie la especificación viva ya sellada, nunca modificando directamente el contenido versionado bajo `openspec/changes/archive/`. El estado local de la carpeta archivada, es decir, los ficheros cuyo nombre empieza por punto situados directamente en su raíz, que nunca se versionan, no forma parte del contenido del incremento y su escritura no lo reabre. El único escritor de Axiom autorizado a escribir estado local en la raíz de una carpeta archivada es la cola de salida de integraciones (`.integrations-outbox.yaml` y sus temporales de escritura atómica). Ningún otro verbo DEBE escribir bajo `openspec/changes/archive/`, e `integrations.yaml` archivado NO DEBE modificarse.
(Previously: prohibía modificar cualquier contenido bajo `openspec/changes/archive/`, sin distinguir el contenido versionado del estado local con punto de la raíz de la carpeta archivada.)

#### Scenario: Intento de reabrir un incremento archivado se rechaza

- **DADO** un incremento ya presente en `openspec/changes/archive/`
- **CUANDO** se intenta reabrirlo o modificarlo directamente para aplicar un ajuste
- **ENTONCES** el sistema rechaza la operación
- **Y** señala que la vía correcta es un ticket de bug o un nuevo incremento

#### Scenario: Regresión post-archive se gestiona mediante bug o nuevo incremento

- **DADO** que se detecta una regresión en un comportamiento definido por un incremento ya archivado
- **CUANDO** se gestiona esa regresión
- **ENTONCES** se abre un ticket de bug o un nuevo incremento de evolución que referencia la especificación viva afectada
- **Y** el contenido del incremento archivado original permanece sin modificaciones

#### Scenario: La cola de integraciones escribe estado local en la carpeta archivada

- **DADO** un incremento archivado y un plugin configurado que escucha `archive`
- **CUANDO** `sdd publish --archive` termina sin rechazo y la entrega falla
- **ENTONCES** existe `.integrations-outbox.yaml` en la raíz de la carpeta archivada
- **Y** ningún fichero versionado de la carpeta archivada cambia, incluido `integrations.yaml`

#### Scenario: Ningún otro escritor abre la excepción

- **DADO** un incremento archivado
- **CUANDO** un verbo de Axiom intenta escribir en su carpeta archivada cualquier fichero distinto de la cola de integraciones y sus temporales, incluidos `integrations.yaml`, `gates.yaml` u otro fichero con punto
- **ENTONCES** se rechaza con el mensaje de RF-ARC-03
### Requirement: Integración del Delta de Contexto en las Rutas Mantenidas (RF-ARC-04)

Con `technical_context.maintained` declarado, archive DEBE integrar el delta de contexto del cambio en las rutas del modo `maintained` mediante el mecanismo de edición dirigida con verificación nativa y traza de la capacidad `project-technical-context` (RF-CTX-14 a RF-CTX-16): por cada entrada del delta, archive edita el documento y la sección ancla según la acción (`add`, `modify` o `remove`) y la traza registra el documento, el ancla, la acción y el estado. Si una entrada `add` apunta a un documento que aún no existe, archive PUEDE crearlo solo dentro de las rutas del modo `maintained` y DEBE registrarlo en la traza con la acción `created`. Archive NO DEBE modificar rutas del modo `reference`, ni documentos no nombrados en las entradas del delta, ni generar un contexto técnico inicial a partir del código. El delta de contexto de la carpeta del cambio se archiva sin alteración.
(Previously: remitía a REQ-24.50 a REQ-24.52 de `spec-baseline-integration` y exigía sustituir los identificadores provisionales por los definitivos en el texto integrado; los identificadores provisionales dejan de existir)

#### Scenario: Entrada `modify` integrada con traza
- **DADO** un delta con una entrada `modify` sobre `context/ARQUITECTURA.md`, ancla `Componentes > Servicio de pagos`
- **CUANDO** archive integra el cambio
- **ENTONCES** la sección ancla del documento refleja el cambio descrito en la entrada
- **Y** la traza registra el documento, el ancla, la acción `modified` y el estado `integrated`

#### Scenario: Documento mantenido inexistente se crea dentro de las rutas declaradas
- **DADO** `maintained.paths: [context/]` y una entrada `add` sobre `context/PAGOS.md`, que no existe
- **CUANDO** archive integra el cambio
- **ENTONCES** se crea `context/PAGOS.md`
- **Y** la traza registra la acción `created`

#### Scenario: Una entrada `add` fuera de las rutas mantenidas se rechaza
- **DADO** una entrada `add` sobre `docs/otro.md`, que no está bajo `maintained.paths`
- **CUANDO** archive valida el delta
- **ENTONCES** se rechaza nombrando el documento y las rutas mantenidas declaradas
- **Y** no se crea ningún fichero

#### Scenario: Archive no genera contexto a partir del código
- **DADO** `maintained.paths` con documentos aún inexistentes y un delta «Sin impacto»
- **CUANDO** archive integra el cambio
- **ENTONCES** no se crea ningún documento del contexto

---

### Requirement: Condiciones de Parada del Archivado con Baseline o Contexto del Proyecto (RF-ARC-05)

Archive DEBE detenerse sin mover el cambio a `openspec/changes/archive/`, nombrando la causa y la continuación, cuando: (a) con `technical_context.maintained` declarado, el cambio no lleva delta de contexto; (b) el delta de contexto es inválido (RF-CTX-06 y RF-CTX-07); (c) falla una precondición de la integración del contexto (RF-CTX-19); (d) falla la composición (RF-ARC-13) o la verificación nativa (RF-CTX-15); (e) con `technical_context.maintained` declarado, el almacén de artefactos es `engram` o `none` (RF-CTX-17), porque sin la carpeta del cambio en ficheros no hay delta de contexto ni traza que integrar; la continuación es declarar `artifact_store: openspec` o `hybrid` en `openspec/config.yaml`, o retirar la declaración; o (f) la declaración de `axiom.yaml` es inválida: un `technical_context:` que no supera la validación (RF-CTX-20) o un bloque `baseline:` residual (RF-TOP-12 de `workspace-topology`). Archive NO DEBE asumir «Sin impacto» ante la ausencia del delta. Cuando la parada ocurre tras ediciones dirigidas ya realizadas, estas DEBEN quedar como cambios sin confirmar en el árbol de trabajo, informadas y revertibles con git, y el cambio permanece en `openspec/changes/`.
(Previously: incluía como causas un dominio no declarado en la baseline, las precondiciones de integración de la baseline, la declaración `baseline:` inválida y el almacén `engram`/`none` con baseline del proyecto; ahora la causa (f) incluye el `baseline:` residual)

#### Scenario: Falta el delta de contexto
- **DADO** `technical_context.maintained` declarado y un cambio sin delta de contexto
- **CUANDO** se solicita el archivado
- **ENTONCES** archive se detiene pidiendo registrar el impacto o «Sin impacto»
- **Y** el cambio permanece en `openspec/changes/`

#### Scenario: Falla una precondición de la integración
- **DADO** un documento mantenido afectado con cambios sin confirmar
- **CUANDO** se solicita el archivado
- **ENTONCES** archive se detiene nombrando el documento y la continuación
- **Y** ningún documento se modifica

#### Scenario: Falla la verificación nativa tras una edición dirigida
- **DADO** una edición dirigida cuyo diff toca un documento no declarado
- **CUANDO** la verificación nativa falla
- **ENTONCES** archive se detiene nombrando el fichero
- **Y** las ediciones realizadas quedan como cambios sin confirmar, informadas y revertibles con git
- **Y** el cambio permanece en `openspec/changes/`

#### Scenario: Declaración de contexto inválida
- **DADO** un `axiom.yaml` con un bloque `technical_context:` inválido
- **CUANDO** se solicita el archivado
- **ENTONCES** archive se detiene nombrando la causa y `axiom workspace validate`

#### Scenario: Un `baseline:` residual detiene el archivado
- **DADO** un `axiom.yaml` que aún declara un bloque `baseline:`
- **CUANDO** se solicita el archivado
- **ENTONCES** archive se detiene con el motivo tipado de RF-TOP-12 y la continuación de retirar `baseline:` y migrar con `axiom archive adopt`
- **Y** no se escribe en `openspec/specs/`

#### Scenario: Almacén de artefactos sin ficheros
- **DADO** `technical_context.maintained` declarado y `artifact_store: engram` (o `none`) en `openspec/config.yaml`
- **CUANDO** se solicita el archivado
- **ENTONCES** archive se detiene nombrando el almacén declarado y la continuación (declarar `openspec` o `hybrid`, o retirar la declaración)
- **Y** no se modifica ningún documento ni se archiva el cambio

#### Scenario: Sin contexto mantenido el almacén Engram no detiene el archivado
- **DADO** un proyecto con `artifact_store: engram`, sin contexto mantenido y con la precondición de archivado satisfecha
- **CUANDO** se solicita el archivado
- **ENTONCES** archive no se detiene por el almacén y se comporta como antes de este cambio

---

### Requirement: «Sin Impacto» Explícito en el Archivado (RF-ARC-06)

Cuando el delta de contexto declara «Sin impacto», archive DEBE dejar intactas las rutas del modo `maintained` (diff vacío en ellas) y DEBE registrar la decisión y su motivo en la traza y en el informe de archive. Un cambio sin delta de contexto NUNCA DEBE tratarse como «Sin impacto».

#### Scenario: «Sin impacto» deja el contexto intacto y registra la decisión
- **DADO** un delta que solo declara «Sin impacto: el cambio solo corrige una errata en un mensaje de error»
- **CUANDO** archive integra el cambio
- **ENTONCES** el diff en las rutas de `maintained.paths` es vacío
- **Y** la traza y el informe de archive contienen la decisión y su motivo

#### Scenario: Una edición del contexto con «Sin impacto» se rechaza
- **DADO** un delta «Sin impacto» y una edición que modifica un documento de `maintained.paths`
- **CUANDO** se ejecuta la verificación nativa
- **ENTONCES** falla nombrando el fichero y la contradicción con la decisión registrada
- **Y** archive se detiene
### Requirement: En Monorepo el Archive se Ejecuta en la Rama de Archivado y se Entrega en un PR Documental Aparte (RF-ARC-07)

En los repositorios con rama de cambio (`monorepo-embedded`), `archive` DEBE ejecutarse en la rama de archivado `<prefijo>/<cambio>-archive` (RF-BRA-13 de `change-branch-lifecycle`), que se crea con `axiom branch ensure --change <c> --archive` **después** de fusionar en la rama de integración el PR del cambio, y su resultado DEBE entregarse en un PR documental aparte contra la rama de integración. La precondición de `archive` es la de RF-ARC-01, sin excepción: evidencia `pr_merged` del PR del cambio (verificada contra el grafo git del repositorio del planning home), veredicto favorable de verificación y compuertas aprobadas. Archivar en cualquier otra rama (la del cambio, la de integración o una ajena; en `monorepo-embedded` no existen ramas de rol porque el único rol es `fullstack`) DEBE rechazarse: `axiom sdd status` proyecta `blocked(wrong_branch)` en la fase archive con la continuación exacta `axiom branch ensure --cwd <ruta> --change <c> --archive` (RF-CLI-05 de `axiom-sdd-cli-integration`) y el escritor de archive se niega con `wrong_branch` (RF-BRA-20), nombrando la rama esperada y sin mover el cambio. El PR documental contiene solo documentación: el movimiento del cambio a `openspec/changes/archive/`, la integración en `openspec/specs/` y los índices. Mientras el cambio no se haya archivado, el cambio fusionado figura como pendiente de archivar (RF-BRA-21) sin bloquear nada. Tras fusionar el PR documental, `axiom branch done` elimina la rama de archivado (RF-BRA-17 y RF-BRA-18). En el repositorio de specs de un `multirepo` NO existe rama de archivado: `archive` se ejecuta in situ en la rama de integración, conserva la precondición de RF-ARC-01 (evidencia `pr_merged` contra el repositorio de especificaciones, `deployment` o `attestation`) y no crea ramas ni worktrees (RF-BRA-05). La skill `sdd-archive` DEBE indicar este orden y la rama.
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
- **ENTONCES** se rechaza explícitamente señalando la ausencia de la evidencia de integración, conforme a RF-ARC-01
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

### Requirement: Publicación de la Documentación Tras el Archivado (RF-ARC-08)

Una vez completado `archive`, la documentación que toca el archivado DEBE poder publicarse con `axiom sdd publish --change <x> --archive` (capacidad `sdd-docs-publication`) sobre la rama donde se archivó (el indicador `--archive` selecciona la rama de archivado como rama esperada de `publish`): en `monorepo-embedded`, la rama de archivado `<prefijo>/<cambio>-archive` que prepara `axiom branch ensure --change <x> --archive` (RF-BRA-13 de `change-branch-lifecycle`); en el repositorio de specs de un `multirepo`, su rama de integración, donde se archiva in situ. La publicación DEBE incluir únicamente rutas de la lista cerrada de documentación (RF-PUB-01): la carpeta archivada del cambio bajo `openspec/changes/archive/`, el borrado del directorio `openspec/changes/<x>/`, las especificaciones vivas de `openspec/specs/` que integró el archivado, `openspec/INDEX.md` y las rutas de contexto técnico mantenido a las que el archivado integró su delta (RF-ARC-04). NO DEBE incluir código ni rutas de `technical_context.reference`. En un cambio `continuous` con `docs_publication: auto` el orquestador DEBE ejecutar la publicación al cerrar el archivado, sin preguntar (RF-PUB-08); en un cambio sin ese campo DEBE ofrecer `axiom sdd publish --change <x> --archive` como siguiente paso, y en un cambio `checkpointed` DEBE ofrecerlo igualmente, sin parada adicional. `archive` NO DEBE ejecutar `git commit` ni `git push` por sí mismo: la publicación es una orden aparte. Una negativa de `publish` (rama equivocada, almacén sin ficheros, remoto inexistente) es informativa y NO revierte el archivado. La PR documental contra la integración y su merge siguen siendo decisión de la persona (RF-ARC-07).
(Previously: la lista incluía también las rutas de baseline a las que el archivado integraba su delta; la baseline ajena se retira)

#### Scenario: Monorepo publica la rama de archivado

- **DADO** un monorepo con el cambio `x` archivado en la rama `inc/x-archive`, con `openspec/changes/archive/<fecha>-x/` creado, `openspec/changes/x/` eliminado y las especificaciones vivas actualizadas
- **CUANDO** se ejecuta `axiom sdd publish --change x --archive`
- **ENTONCES** `origin/inc/x-archive` contiene un commit con esas rutas de documentación
- **Y** ningún fichero de código fuente aparece en el commit

#### Scenario: Multirepo publica la integración del repositorio de specs

- **DADO** un multirepo con el cambio archivado in situ en la rama de integración del repositorio de specs
- **CUANDO** se ejecuta `axiom sdd publish --change x --archive` en ese repositorio
- **ENTONCES** se sube su rama de integración a `origin` con los documentos archivados

#### Scenario: El contexto mantenido integrado se publica

- **DADO** un archivado que integró su delta de contexto en `docs/arquitectura.md` (`technical_context.maintained`)
- **CUANDO** se ejecuta `axiom sdd publish --change x --archive`
- **ENTONCES** esa ruta figura en el commit
- **Y** una ruta de `technical_context.reference` modificada en el mismo árbol no figura en el commit

#### Scenario: Cambio continuo autorizado publica al archivar

- **DADO** un cambio `continuous` con `docs_publication: auto` cuyo archivado acaba de completarse
- **CUANDO** el orquestador cierra el archivado
- **ENTONCES** ejecuta `axiom sdd publish --change x --archive` sin preguntar

#### Scenario: Cambio continuo sin autorización ofrece la publicación

- **DADO** un cambio `continuous` sin `docs_publication` cuyo archivado acaba de completarse
- **CUANDO** el orquestador cierra el archivado
- **ENTONCES** no ejecuta `publish` y ofrece `axiom sdd publish --change x --archive` como siguiente paso

#### Scenario: Publicar con `--archive` fuera de la rama de archivado se rechaza

- **DADO** un monorepo con el cambio `x` archivado y el repositorio desplegado en la rama del cambio `inc/x`
- **CUANDO** se ejecuta `axiom sdd publish --change x --archive`
- **ENTONCES** se rechaza con `wrong_branch` nombrando `inc/x-archive` como rama esperada
- **Y** no se confirma ni se sube nada

#### Scenario: Sin `--archive` desde la rama de archivado se rechaza nombrando la orden correcta

- **DADO** un monorepo con el cambio `x` archivado y el repositorio desplegado en `inc/x-archive`
- **CUANDO** se ejecuta `axiom sdd publish --change x` sin `--archive`
- **ENTONCES** se rechaza con `wrong_branch` nombrando `inc/x` como rama esperada
- **Y** el mensaje incluye la orden correcta `axiom sdd publish --change x --archive`
- **Y** no se confirma ni se sube nada

#### Scenario: `archive` no publica por sí mismo

- **DADO** un observador instalado en el único punto del producto que ejecuta git
- **CUANDO** se ejecuta `archive` con éxito
- **ENTONCES** el observador no registra `commit` ni `push`

#### Scenario: Una negativa de `publish` no revierte el archivado

- **DADO** un cambio archivado con un commit de código sin subir en la rama de archivado
- **CUANDO** `publish` se niega con `code_commits_unpublished`
- **ENTONCES** el cambio sigue archivado en `openspec/changes/archive/`
- **Y** el orquestador presenta la negativa como informativa con la orden de push
### Requirement: Al Menos un Escenario por Requisito en el Archivado de un Proyecto ISO (RF-ARC-09)

En un cambio ISO (sellado con `iso`, RF-ISO-01 de `sdd-iso-evidence` y RF-KOF-12 de `sdd-lane-kickoff`), la composición del archivado (`archive` y `axiom sdd archive-compose` cuando resuelve el dominio en la baseline del proyecto (`openspec/specs/`) y puede determinar el cambio, por `--change` o porque la delta está en la carpeta de un cambio; supuesto visible) DEBE rechazar, sin escribir ningún documento y con un motivo tipado que nombre el dominio, la sección y el requisito, todo requisito ADDED o MODIFIED de la delta que no contenga al menos un escenario (un encabezado `#### Scenario:`). NO se aplica a los requisitos REMOVED ni RENAMED. `axiom sdd status` DEBE adelantar el rechazo con el mismo motivo tipado en cuanto la delta existe (RF-CLI-11 de `axiom-sdd-cli-integration`). Fuera de un cambio ISO la composición NO cambia. Un cambio sellado sin `iso`, o sin instantánea, es un cambio no ISO y la regla NO se aplica. Cuando `axiom sdd archive-compose` se invoca con `--canonical` y `--delta` sin contexto de proyecto ni de cambio, no hay instantánea que consultar y la regla NO se aplica.

#### Scenario: Un ADDED sin escenarios se rechaza en ISO

- **DADO** un cambio ISO y una delta con un requisito ADDED sin ningún `#### Scenario:`
- **CUANDO** se ejecuta el archivado
- **ENTONCES** se rechaza con un motivo tipado que nombra el dominio, la sección ADDED y el requisito
- **Y** ningún documento de la baseline se modifica
- **Y** el cambio permanece en `openspec/changes/<cambio>/`

#### Scenario: Un MODIFIED sin escenarios se rechaza en ISO

- **DADO** un cambio ISO y una delta con un MODIFIED cuyo bloque no contiene escenarios
- **CUANDO** se ejecuta `axiom sdd archive-compose --cwd <raíz> --document <dominio> --delta <ruta>`
- **ENTONCES** se rechaza con el motivo tipado nombrando el requisito
- **Y** no se escribe el fichero de salida

#### Scenario: REMOVED y RENAMED no exigen escenarios

- **DADO** un cambio ISO y una delta con un REMOVED y un RENAMED, ambos sin escenarios
- **CUANDO** se ejecuta el archivado
- **ENTONCES** la regla de escenarios no los rechaza

#### Scenario: Status adelanta el rechazo

- **DADO** un cambio ISO con una delta con un ADDED sin escenarios, todavía sin archivar
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se proyecta el mismo motivo tipado que nombra el dominio, la sección y el requisito

#### Scenario: Fuera de ISO el archivado no cambia

- **DADO** un cambio no ISO y una delta con un ADDED sin escenarios
- **CUANDO** se ejecuta el archivado
- **ENTONCES** la composición procede como antes de este cambio

#### Scenario: Sin contexto de proyecto la regla no se aplica

- **DADO** una invocación `axiom sdd archive-compose --canonical <ruta> --delta <ruta>` con un ADDED sin escenarios
- **CUANDO** se ejecuta
- **ENTONCES** la composición procede como antes de este cambio

---

### Requirement: En un Cambio ISO el Archivado Exige la Matriz de Verify Válida y con Éxito (RF-ARC-10)

En un cambio ISO, `archive` DEBE bloquearse con un motivo tipado mientras la matriz de verify del `verify-report.md` esté ausente (salvo en un bug sin impacto en la spec, que no tiene escenarios afectados y cuya matriz ausente o vacía es válida, RF-ISO-10 de `sdd-iso-evidence`), sea inválida (RF-ISO-06 a RF-ISO-08 de `sdd-iso-evidence`) o contenga alguna fila con resultado `fail`, con independencia del ritmo sellado (`continuous` o `checkpointed`). `axiom sdd status` DEBE reflejar el bloqueo en la dependencia `archive` nombrando la fila o el escenario afectado, `axiom sdd continue` NO DEBE autorizarla, y el escritor de `archive` DEBE negarse sin mover el cambio ni modificar la baseline. El bloqueo se **suma** a las demás precondiciones del archivado (integración o despliegue, rama y compuertas) y no sustituye a ninguna de ellas. Esta regla es una **excepción declarada** a la doctrina «Missing or failed verification never gates archive» (`persistence-contract.md`) y a RF-REV-03 de `sdd-independent-review`, que siguen vigentes íntegras para los cambios no ISO y para los hallazgos de revisión: en ISO, lo que bloquea es la validez de la matriz (evidencia declarada y contrastada con el árbol), nunca un hallazgo de verify, un juicio `ESCALATED` ni una revisión de cobertura. Fuera de un cambio ISO, un verify ausente, libre o fallido NO DEBE bloquear el archivado por sí mismo.

#### Scenario: Una matriz con una fila fallida bloquea el archivado en ISO

- **DADO** un cambio ISO con una matriz válida en la que una fila tiene resultado `fail`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json` en la fase `archive`
- **ENTONCES** el archivado queda bloqueado con el motivo tipado de matriz
- **Y** el motivo nombra la fila fallida

#### Scenario: Una matriz inválida bloquea el archivado en ISO aunque el cambio sea `continuous`

- **DADO** un cambio ISO sellado `continuous` y una matriz con un escenario sin fila
- **CUANDO** se solicita el archivado
- **ENTONCES** se bloquea con el motivo tipado de matriz
- **Y** el cambio permanece en `openspec/changes/<cambio>/`

#### Scenario: Una matriz ausente bloquea el archivado en ISO

- **DADO** un cambio ISO cuyo `verify-report.md` no contiene matriz
- **CUANDO** se solicita el archivado
- **ENTONCES** se bloquea con el motivo tipado de matriz ausente

#### Scenario: Un bug sin impacto en la spec archiva sin matriz

- **DADO** un cambio ISO declarado como bug sin impacto en la spec, con `Motivo:` no vacío, sin `verify-report.md` o con la matriz vacía, y el resto de precondiciones de archivado satisfechas
- **CUANDO** se solicita el archivado
- **ENTONCES** el archivado no se bloquea por la matriz
- **Y** el archivado no integra ningún requisito en la baseline

#### Scenario: Una referencia de prueba que no resuelve bloquea el archivado

- **DADO** un cambio ISO con una fila automática cuya referencia no resuelve a ningún símbolo del árbol de trabajo de la rama de archivado
- **CUANDO** se solicita el archivado
- **ENTONCES** se bloquea con el motivo tipado de matriz, que nombra la fila y la referencia

#### Scenario: Una matriz válida y con éxito permite el archivado en ISO

- **DADO** un cambio ISO con una matriz válida, todas las filas con resultado `pass`, y el resto de precondiciones de archivado satisfechas
- **CUANDO** se solicita el archivado
- **ENTONCES** el archivado no se bloquea por la matriz

#### Scenario: Una matriz válida no sustituye las demás precondiciones

- **DADO** un cambio ISO con una matriz válida y con éxito, sin evidencia de integración o despliegue
- **CUANDO** se solicita el archivado
- **ENTONCES** se rechaza por la ausencia de integración, conforme a RF-ARC-01

#### Scenario: Un hallazgo de verify no bloquea el archivado en ISO

- **DADO** un cambio ISO con una matriz válida y con éxito y un verify con un hallazgo de severidad crítica y un juicio `ESCALATED`
- **CUANDO** se solicita el archivado
- **ENTONCES** el archivado no se bloquea por esos hallazgos

#### Scenario: El escritor de `archive` se niega sin mover el cambio

- **DADO** un cambio ISO con una matriz inválida
- **CUANDO** se invoca el escritor de `archive`
- **ENTONCES** se niega con el motivo tipado de matriz
- **Y** no se mueve el cambio ni se modifica ningún documento de la baseline

#### Scenario: Fuera de ISO un verify fallido no bloquea el archivado

- **DADO** un cambio no ISO y un `verify-report.md` con veredicto fallido o ausente, y el resto de precondiciones satisfechas
- **CUANDO** se solicita el archivado
- **ENTONCES** el archivado no se bloquea por la verificación, como antes de este cambio
### Requirement: Composición Nativa de OpenSpec con `axiom sdd archive-compose --canonical --delta` (RF-ARC-11)

La composición de una delta sobre una especificación canónica y `axiom sdd archive-compose --canonical <ruta> --delta <ruta>` DEBEN producir la salida con identidad del requisito por su identificador cuando el nombre de la delta contiene un identificador válido según el patrón del proyecto (RF-RID-04 de `requirement-identifiers`; sin contexto de proyecto rige el patrón por defecto) y por el texto exacto del nombre si no lo contiene, orden de aplicación RENAMED, MODIFIED, REMOVED y ADDED, rechazo sin escribir ante una delta no aplicable y opciones `--supersede` y `--superseded-requirements`. El identificador de un nombre es el último identificador válido entre paréntesis.

**Resolución por identificador.** Se aplica a MODIFIED, REMOVED y al nombre antiguo de un RENAMED. Con un identificador en el nombre, se buscan los requisitos de la canónica que NO estén superados (los que no llevan el sufijo ` [SUPERSEDED / DEPRECADO]`) con ese identificador: si hay exactamente uno, es ese; si no hay ninguno, solo vale el requisito cuyo título coincide exactamente con el nombre (así se conserva la composición de hoy cuando la delta cita tal cual un requisito superado o un título con dos identificadores, RF-ARC-12), y sin él la composición se rechaza sin escribir con el error tipado de identificador inexistente, que nombra la sección y el identificador y conserva el nombre del requisito (RF-RID-11); si hay varios, rige la coincidencia por título exacto cuando exactamente uno lo tiene, y si no la composición DEBE rechazarse sin escribir con un motivo tipado que nombre el identificador y los requisitos afectados (no se resuelve en silencio hacia uno de ellos). Sin identificador en el nombre, se busca el título exacto (primera coincidencia), como hasta ahora, y un nombre sin identificador que no coincide se rechaza como hoy (`no canonical requirement named "<nombre>"`). Un MODIFIED cuyo título cambió pero conserva el identificador se aplica sobre el requisito que lleva ese identificador y lo sustituye por el bloque de la delta, que pasa a ser el nombre del requisito para las operaciones siguientes de la misma delta. Un REMOVED por identificador elimina (o, con `--supersede`, marca como superado) el requisito de la canónica resuelto, y la anotación usa el nombre que ese requisito tiene en la canónica. Un RENAMED sigue exigiendo ambos nombres (`Antiguo → Nuevo`) y la nota `(Reason: …)`. Los requisitos superados quedan excluidos de la resolución por identificador (su identificador está retirado y un MODIFIED no los reabre), pero conservan su `(ID)` en el título y siguen contando como identificador usado en el registro de la spec viva.

**Por título exacto, sin cambios.** ADDED (existencia previa), la colisión del nombre nuevo de un RENAMED y `--superseded-requirements` siguen emparejándose por título exacto: el compositor no rechaza un ADDED ni un nuevo nombre de RENAMED cuyo identificador ya figura bajo otro título; esa unicidad la imponen antes de componer RF-ISO-03 (ISO) y RF-OLS-04 (ODD).

**Provisionales.** `axiom sdd archive-compose` NO DEBE aceptar un ADDED cuyo nombre lleve un identificador provisional (RF-RID-09): DEBE rechazarlo sin escribir con un error tipado que remita al paso de archivado del cambio (RF-ARC-15), que es el que asigna el número. Tampoco DEBE aceptar un provisional en MODIFIED, REMOVED o RENAMED, ni un token con la forma de un provisional (bien o mal formado) en el cuerpo de cualquier requisito de la delta, porque el cuerpo llega a la spec viva: la composición rechaza la delta entera con el mismo error tipado, que nombra la sección, el requisito y el token. La composición de una delta sin provisionales no cambia.

**Archivado interrumpido (D17).** `axiom sdd archive-check` calcula «ya compuesto» por nombre para ADDED, MODIFIED y RENAMED, y por la misma resolución por identificador para REMOVED (un REMOVED se da por compuesto cuando ya no se resuelve en la canónica); con provisionales, reconoce la asignación ya hecha por el mapa guardado (RF-ARC-15). La clasificación idempotente de la integración ODD (RF-OLS-02) resuelve MODIFIED y REMOVED con esa misma resolución: un MODIFIED integrado deja la canónica con el nombre nuevo y una segunda ejecución informa `already`. Compositor, `archive-check` y clasificación ODD DEBEN coincidir siempre.

Toda composición aceptada antes de este cambio produce la misma salida byte a byte (RF-ARC-12). `archive-compose` NO DEBE aceptar los modos `--document` ni `--in-place`: su uso DEBE rechazarse con un motivo tipado que indique que la baseline ajena se retiró y que la composición es `--canonical --delta`. (Es REQ-24.38 de `spec-baseline-integration`, aplicable a OpenSpec, más la retirada de los modos de documento.) Con un bloque `sdd:` inválido la composición en el contexto de un proyecto se niega sin escribir (RF-RID-08 de `requirement-identifiers`).
(Previously: un MODIFIED, REMOVED o RENAMED con un identificador que no existe se rechazaba solo con el mensaje genérico `no canonical requirement named`, sin error tipado propio, y un ADDED provisional no estaba contemplado; ahora el identificador inexistente tiene un error tipado propio y el ADDED provisional se rechaza remitiendo al paso de archivado del cambio. La identidad por identificador y la salida de toda composición aceptada no cambian.)

Supuesto: se conservan sin cambios las pruebas nativas de composición de `--canonical --delta`.

#### Scenario: La salida nativa no cambia
- **DADO** una especificación canónica y un delta con ADDED, MODIFIED, REMOVED y RENAMED
- **CUANDO** se ejecuta `axiom sdd archive-compose --canonical <ruta> --delta <ruta>`
- **ENTONCES** la salida es idéntica byte a byte a la que producía la composición antes de este cambio

#### Scenario: Rechazo nativo sin escribir
- **DADO** un delta con un MODIFIED sobre un nombre de requisito inexistente
- **CUANDO** se ejecuta la composición con `--output <ruta>`
- **ENTONCES** finaliza con código distinto de cero nombrando la sección y el requisito
- **Y** no se escribe el fichero de salida

#### Scenario: `--supersede` conserva su comportamiento
- **DADO** un delta con un REMOVED y la opción `--supersede`
- **CUANDO** se compone
- **ENTONCES** el requisito se conserva marcado como superado, con su nota de motivo

#### Scenario: Los modos de documento se rechazan
- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom sdd archive-compose --document <dominio> --delta <ruta>` o con `--in-place`
- **ENTONCES** finaliza con código distinto de cero con un motivo tipado que remite a `--canonical --delta`
- **Y** no se escribe ningún fichero

#### Scenario: Un MODIFIED con el título cambiado conserva el identificador
- **DADO** una canónica con `### Requirement: Cobro (REQ-30.1)` y una delta con un MODIFIED `### Requirement: Cobro con IVA (REQ-30.1)`
- **CUANDO** se compone
- **ENTONCES** la salida contiene el bloque de la delta con el título `Cobro con IVA (REQ-30.1)` en la posición del requisito original
- **Y** no queda ningún requisito `Cobro (REQ-30.1)`
- **Y** no se rechaza la composición

#### Scenario: Sin identificador en el título se empareja por el texto exacto
- **DADO** una canónica con `### Requirement: Cobro` y una delta con un MODIFIED `### Requirement: Cobro`
- **CUANDO** se compone
- **ENTONCES** el MODIFIED se aplica por coincidencia exacta del nombre, como antes de este cambio

#### Scenario: Un MODIFIED sin identificador cuyo nombre no coincide se rechaza
- **DADO** una canónica con `### Requirement: Cobro` y una delta con un MODIFIED `### Requirement: Cobro con IVA`
- **CUANDO** se compone
- **ENTONCES** finaliza con código distinto de cero nombrando la sección MODIFIED y el requisito
- **Y** no se escribe el fichero de salida

#### Scenario: Con un patrón declarado se empareja por el identificador de ese patrón
- **DADO** un proyecto con `sdd: {id_pattern: "RF-<DOM>-<NN>"}`, una canónica con `### Requirement: Alta de usuario (RF-USU-01)` y una delta con un MODIFIED `### Requirement: Alta de usuario con correo (RF-USU-01)`
- **CUANDO** se compone en el contexto de ese proyecto
- **ENTONCES** el MODIFIED sustituye al requisito `RF-USU-01`

#### Scenario: RENAMED sigue exigiendo ambos nombres
- **DADO** una delta con un RENAMED `Cobro (REQ-30.1) → Cobro con IVA (REQ-30.1)` sobre una canónica que contiene `Cobro (REQ-30.1)`
- **CUANDO** se compone
- **ENTONCES** el requisito pasa a llamarse `Cobro con IVA (REQ-30.1)`
- **Y** un RENAMED al que le falte uno de los dos nombres se rechaza sin escribir

#### Scenario: Un MODIFIED sobre el nombre nuevo de un RENAMED de la misma delta se aplica
- **DADO** una delta con un RENAMED `Cobro (REQ-30.1) → Cobro con IVA (REQ-30.1)` y un MODIFIED `Cobro con IVA (REQ-30.1)`
- **CUANDO** se compone
- **ENTONCES** se aplican primero el RENAMED y después el MODIFIED, con el cuerpo del MODIFIED en el requisito renombrado

#### Scenario: Un identificador repetido en la canónica no se resuelve en silencio
- **DADO** una canónica con dos requisitos `Cobro (REQ-30.1)` y `Devolución (REQ-30.1)` y una delta con un MODIFIED `Reembolso (REQ-30.1)`, cuyo nombre no coincide con ninguno de los dos
- **CUANDO** se compone
- **ENTONCES** finaliza con código distinto de cero con un motivo tipado que nombra `REQ-30.1` y los dos requisitos
- **Y** no se escribe el fichero de salida

#### Scenario: Con un identificador repetido, la coincidencia exacta de nombre sigue valiendo
- **DADO** una canónica con dos requisitos `Cobro (REQ-30.1)` y `Devolución (REQ-30.1)` y una delta con un MODIFIED `Cobro (REQ-30.1)`, que coincide por nombre exacto con exactamente uno
- **CUANDO** se compone
- **ENTONCES** el MODIFIED se aplica sobre `Cobro (REQ-30.1)`, como antes de este cambio

#### Scenario: Una composición hoy aceptada produce la misma salida
- **DADO** el conjunto de composiciones que el compositor aceptaba antes de este cambio
- **CUANDO** se componen con el compositor nuevo
- **ENTONCES** cada salida es idéntica byte a byte a la anterior (RF-ARC-12)

#### Scenario: Un REMOVED con el título obsoleto se resuelve por identificador
- **DADO** una canónica con `### Requirement: Cobro con IVA (REQ-30.1)` y una delta con un REMOVED `### Requirement: Cobro (REQ-30.1)`
- **CUANDO** se compone, con y sin `--supersede`
- **ENTONCES** se elimina (o, con `--supersede`, se marca como superado) el requisito `Cobro con IVA (REQ-30.1)`
- **Y** con `--supersede` la anotación de superado usa el nombre `Cobro con IVA (REQ-30.1)` de la canónica

#### Scenario: Un RENAMED resuelve su nombre antiguo por identificador
- **DADO** una canónica con `### Requirement: Cobro (REQ-30.1)` y una delta con un RENAMED `Cobro antiguo (REQ-30.1) → Cobro con IVA (REQ-30.1)` con su nota `(Reason: …)`
- **CUANDO** se compone
- **ENTONCES** el requisito `Cobro (REQ-30.1)` pasa a llamarse `Cobro con IVA (REQ-30.1)`

#### Scenario: Un MODIFIED que cambia el identificador no se resuelve
- **DADO** una canónica con `### Requirement: Cobro (REQ-30.1)` y una delta con un MODIFIED `### Requirement: Cobro (REQ-30.2)`, sin ningún requisito con `REQ-30.2`
- **CUANDO** se compone
- **ENTONCES** finaliza con código distinto de cero con el error tipado de identificador inexistente, que nombra la sección MODIFIED, `REQ-30.2` y el requisito
- **Y** no se escribe el fichero de salida

#### Scenario: Los requisitos superados quedan fuera de la resolución por identificador
- **DADO** una canónica con `### Requirement: Cobro (REQ-30.1) [SUPERSEDED / DEPRECADO]` y una delta con un MODIFIED `### Requirement: Cobro con IVA (REQ-30.1)`
- **CUANDO** se compone
- **ENTONCES** se rechaza sin escribir con el error tipado de identificador inexistente, porque ningún requisito vivo lleva `REQ-30.1`
- **Y** si la canónica tuviera además un requisito vivo `Cobro nuevo (REQ-30.1)`, el MODIFIED se aplicaría sobre ese

#### Scenario: ADDED y el nombre nuevo de un RENAMED se emparejan por título
- **DADO** una canónica con `### Requirement: Cobro (REQ-30.1)` y una delta con un ADDED `### Requirement: Devolución (REQ-30.1)`
- **CUANDO** se compone
- **ENTONCES** el ADDED se añade, porque no existe ningún requisito con ese título exacto
- **Y** el compositor no valida la unicidad del identificador (la imponen RF-ISO-03 y RF-OLS-04 antes de componer)

#### Scenario: Con un patrón declarado, un requisito con identificador de otro patrón se empareja por título
- **DADO** un proyecto con `sdd: {id_pattern: "RF-<DOM>-<NN>"}`, una canónica con `### Requirement: Cobro (REQ-30.1)` y una delta con un MODIFIED `### Requirement: Cobro (REQ-30.1)`
- **CUANDO** se compone en el contexto de ese proyecto
- **ENTONCES** como `REQ-30.1` no es un identificador válido, el MODIFIED se empareja por título exacto y se aplica

#### Scenario: `archive-check` con un REMOVED por identificador y título obsoleto
- **DADO** una delta con un REMOVED `Cobro (REQ-30.1)` y una canónica que aún contiene `Cobro con IVA (REQ-30.1)`
- **CUANDO** se ejecuta `axiom sdd archive-check` antes de componer
- **ENTONCES** el dominio no figura como ya compuesto
- **Y** tras componer y volver a consultar, el dominio figura como ya compuesto

#### Scenario: Una segunda integración ODD de un MODIFIED por identificador es idempotente
- **DADO** una delta ODD con un MODIFIED `Cobro con IVA (REQ-30.1)` ya integrado sobre `Cobro (REQ-30.1)`
- **CUANDO** se repite la integración
- **ENTONCES** el MODIFIED se clasifica como ya aplicado y no se escribe nada

#### Scenario: Con un bloque `sdd:` inválido la composición se niega
- **DADO** un proyecto cuyo `axiom.yaml` tiene un bloque `sdd:` inválido y una delta de un cambio de ese proyecto
- **CUANDO** se ejecuta `axiom sdd archive-compose` en su contexto
- **ENTONCES** finaliza con código distinto de cero con la continuación `axiom workspace validate`
- **Y** no se escribe ningún fichero

#### Scenario: Un ADDED provisional se rechaza remitiendo al paso de archivado del cambio
- **DADO** una delta con un ADDED cuyo nombre lleva `«prov-a»`
- **CUANDO** se ejecuta `axiom sdd archive-compose --canonical <ruta> --delta <ruta> --output <ruta>`
- **ENTONCES** finaliza con código distinto de cero con un error tipado que nombra `«prov-a»` y remite al paso de archivado del cambio
- **Y** no se escribe el fichero de salida

#### Scenario: Un provisional en MODIFIED, REMOVED o RENAMED se rechaza
- **DADO** una delta con un MODIFIED cuyo nombre lleva `«prov-a»`
- **CUANDO** se compone
- **ENTONCES** finaliza con código distinto de cero con un error tipado que nombra la sección MODIFIED y `«prov-a»`
- **Y** no se escribe el fichero de salida

#### Scenario: Un provisional solo en el cuerpo se rechaza
- **DADO** una delta con un ADDED `Cobro (REQ-30.5)` cuyo cuerpo cita `«prov-a»`
- **CUANDO** se ejecuta `axiom sdd archive-compose --canonical <ruta> --delta <ruta> --output <ruta>`
- **ENTONCES** finaliza con código distinto de cero con el error tipado de provisional, que nombra la sección ADDED, el requisito y `«prov-a»`
- **Y** no se escribe el fichero de salida

#### Scenario: Un REMOVED con un identificador inexistente tiene su error tipado
- **DADO** una canónica sin `REQ-30.9` y una delta con un REMOVED `Cobro (REQ-30.9)`
- **CUANDO** se compone
- **ENTONCES** finaliza con código distinto de cero con el error tipado de identificador inexistente que nombra la sección REMOVED y `REQ-30.9`
- **Y** no se escribe el fichero de salida
### Requirement: Fidelidad de Bytes al Componer (RF-ARC-12)

La composición DEBE preservar byte a byte todo lo que no pertenece a un bloque editado: finales de línea (LF y CRLF), marca de orden de bytes (BOM) UTF-8, contenido entre bloques y final del fichero. Los bloques insertados o sustituidos DEBEN emitirse con el final de línea del documento y NUNCA DEBEN introducir un BOM; si el fichero tiene BOM, se conserva. Si el documento no termina en salto de línea y el último bloque no se edita, el final del fichero se conserva. (Es REQ-24.45 de `spec-baseline-integration`.)

#### Scenario: Round-trip en LF, CRLF y CRLF con BOM
- **DADO** una especificación fixture en variantes LF, CRLF y CRLF con BOM
- **CUANDO** se compone un delta con ADDED, MODIFIED y REMOVED
- **ENTONCES** la salida es idéntica byte a byte a la original salvo en los bloques editados, en las tres variantes

#### Scenario: Los bloques editados usan el final de línea del documento
- **DADO** un documento CRLF y un delta en LF
- **CUANDO** se compone
- **ENTONCES** los bloques insertados o sustituidos usan CRLF y el fichero no contiene saltos LF sueltos

#### Scenario: El BOM se conserva y no se añade
- **DADO** un documento con BOM y otro sin BOM
- **CUANDO** se componen sendos deltas
- **ENTONCES** el primero conserva su BOM y el segundo no gana BOM

#### Scenario: Final de fichero sin salto de línea
- **DADO** un documento cuyo último bloque no termina en salto de línea y un delta que no lo edita
- **CUANDO** se compone
- **ENTONCES** el final del fichero conserva sus bytes

---

### Requirement: Validación Completa antes de Escribir (RF-ARC-13)

Todas las composiciones de todas las capacidades del cambio DEBEN validarse y componerse en memoria antes de escribir ninguna especificación ni el contexto: o se aplican todas las composiciones o ninguna. Un rechazo NO DEBE dejar la spec viva parcialmente modificada y DEBE nombrar la capacidad, la sección, el requisito y el motivo. La integración del contexto solo comienza cuando todas las composiciones han tenido éxito. (Es REQ-24.46 de `spec-baseline-integration`.)

#### Scenario: Una capacidad falla y ninguna se modifica
- **DADO** un cambio con deltas para dos capacidades, la segunda con un MODIFIED sobre un requisito inexistente
- **CUANDO** se ejecuta el archivado
- **ENTONCES** se rechaza nombrando la capacidad, la sección y el requisito
- **Y** ninguna especificación cambia

#### Scenario: Todas las operaciones válidas se aplican
- **DADO** un cambio con deltas válidos para dos capacidades
- **CUANDO** se ejecuta el archivado
- **ENTONCES** ambas especificaciones se sustituyen por su composición

#### Scenario: La integración del contexto no empieza si una composición falla
- **DADO** un cambio con una composición inválida y un delta de contexto válido
- **CUANDO** se ejecuta el archivado
- **ENTONCES** se rechaza nombrando la composición inválida
- **Y** no se edita ningún documento del contexto

---

### Requirement: Retirada de `axiom sdd integrate` (RF-ARC-14)

El subcomando `axiom sdd integrate` NO DEBE existir. Su invocación DEBE rechazarse con un motivo tipado que remita a `axiom archive context check|prepare|verify|show` para el contexto técnico (RF-CTX-13 de `project-technical-context`) y a `axiom archive adopt` para la migración (RF-ADP-01 y RF-ADP-03 de `openspec-adoption`), sin escribir nada. Las skills y la doctrina gestionadas NO DEBEN citar `axiom sdd integrate`.

#### Scenario: `axiom sdd integrate` remite a los verbos nuevos
- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom sdd integrate check --change <cambio>`
- **ENTONCES** finaliza con código distinto de cero con un motivo tipado que nombra `axiom archive context` y `axiom archive adopt`
- **Y** no se escribe ningún fichero

#### Scenario: Las skills no citan el verbo retirado
- **DADO** los textos de las skills y la doctrina gestionadas
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** ninguno contiene `axiom sdd integrate`
### Requirement: Paso de Archivado del Cambio Entero Dirigido por Go, Capacidades Nuevas Incluidas (RF-ARC-15)

El archivado SDD DEBE disponer de un paso nativo (verbo o modo; el nombre lo fija el diseño) que cubra todo el cambio y que, antes de la instantánea y del movimiento de la carpeta del cambio a `openspec/changes/archive/`, DEBE:

1. Planificar las asignaciones de todos los identificadores provisionales de todas las capacidades del cambio en un plan único (RF-RID-11, RF-RID-10), validado antes de escribir.
2. Componer todas las capacidades del cambio sobre la spec viva, tanto las que modifican una especificación existente como las capacidades nuevas (sin spec canónica), que hoy se copian sin pasar por Go y dejarían los provisionales en `openspec/specs/`. Una capacidad nueva con ADDED provisionales se archiva con identificadores definitivos, incluida la apertura de una serie nueva (`REQ-<n>.<m>` y `RF-<DOM>-<NN>`). Una capacidad nueva cuya delta trae MODIFIED, REMOVED o RENAMED se rechaza con el error tipado de capacidad inexistente (RF-RID-11), en lugar de copiarse.
3. Reescribir las líneas `Cubre:` de los ficheros de tareas del cambio, las deltas del cambio y la columna del requisito de la matriz de los informes de verify (RF-RID-12).
4. Escribir la traza del mapa de provisional a definitivo con la capacidad de cada entrada en la carpeta del cambio (RF-RID-14).

Las opciones de superación (`--supersede`, `--superseded-requirements`) de RF-ARC-11 se aplican a todo el cambio: `--supersede` actúa sobre los REMOVED de todas las capacidades, y cada nombre de `--superseded-requirements` DEBE casar por título exacto en la especificación canónica de exactamente una capacidad del cambio, o llevar el prefijo `<capacidad>/` que la fija; un nombre que no casa en ninguna, o que casa en varias sin prefijo, DEBE rechazarse sin escribir con un error tipado que lo nombre.

Una capacidad cuyos requisitos ya están todos integrados (RF-OLS-03, con la misma clasificación que la integración ODD) se informa como ya integrada y no se escribe; una capacidad parcialmente integrada (por ejemplo, por un archivado anterior interrumpido o por una composición manual de parte del cambio) se completa componiendo solo lo pendiente, de modo que el archivado siempre es reanudable. Una capacidad sin nada integrado se compone con la delta entera, igual que `axiom sdd archive-compose`.

El paso DEBE emitir un informe, legible y en JSON, con: por capacidad, su resultado (creada, compuesta, reanudada o ya integrada); el mapa de provisional a definitivo con la capacidad, el requisito y el estado de cada entrada (`assigned`, `kept` o `reassigned`); los duplicados detectados (RF-RID-13), con su carácter bloqueante o informativo; y los ficheros escritos. El informe de la fase de archivado (`archive-report.md`) DEBE incluir la tabla del mapa. El paso DEBE ofrecer una opción de solo lectura que ejecute las mismas comprobaciones y emita el mismo informe sin escribir nada (equivalente a RF-OLS-10) y que, sobre un cambio ya archivado, compruebe que cada definitivo de su mapa aparece una sola vez en la spec viva y en su capacidad.

Todo el paso DEBE ser atómico (RF-RID-12): un fallo en cualquier punto no deja a medias la spec viva, las tareas, las deltas, los informes de verify ni la traza. Las reescrituras DEBEN ocurrir antes de la instantánea previa al movimiento de la carpeta, de modo que los ficheros del cambio archivado conservan, byte a byte, el contenido de esa instantánea (`diff -r` de la skill de archivado). Un cambio sin provisionales produce la misma spec viva, las mismas tareas y las mismas deltas que producía el archivado por dominio hasta hoy.

El archivado interrumpido DEBE ser reanudable: `axiom sdd archive-check` DEBE reconocer la asignación ya hecha usando el mapa guardado (RF-RID-14) y no volver a asignar números, de modo que tras una interrupción la reejecución del paso termina el archivado sin duplicar requisitos ni cambiar los identificadores ya asignados. Si el estado del cambio reevalúa puertas por el digest de las tareas o de la delta, la reescritura NO DEBE dejar la reejecución bloqueada por puertas en `pending`.

Dos cambios SDD que se archivan uno tras otro con ADDED provisionales en la misma capacidad DEBEN recibir números distintos sin que el autor renumere a mano: el segundo calcula el siguiente libre sobre la spec viva que ya contiene el primero.

Si dos ramas de archivado asignaron el mismo número y la segunda se rebasa sobre la rama de integración que ya contiene la primera, la opción de solo lectura del paso sobre el cambio archivado DEBE detectar el duplicado (RF-RID-13) con un error tipado cuya continuación es descartar la rama de archivado y rehacer el archivado desde la rama de integración. El cambio archivado es inmutable (RF-ARC-03) y NO se reasigna en sitio; en la rama de integración la carpeta del cambio conserva los provisionales, y la reejecución del paso asigna el siguiente libre.

#### Scenario: Se archiva un cambio con ADDED provisionales en una capacidad existente

- **DADO** el patrón por defecto, una spec viva con `REQ-30.1` a `REQ-30.4` y un cambio con un ADDED provisional con la serie `30` fijada en una capacidad existente
- **CUANDO** se ejecuta el paso de archivado del cambio
- **ENTONCES** la spec viva contiene el requisito con `REQ-30.5`
- **Y** ningún provisional queda en `openspec/specs/`

#### Scenario: Una capacidad nueva se archiva con identificadores definitivos

- **DADO** un cambio con una capacidad nueva con dos ADDED cuyos provisionales comparten una etiqueta de serie nueva y una spec viva cuyo mayor número de serie es `46`
- **CUANDO** se ejecuta el paso de archivado del cambio
- **ENTONCES** `openspec/specs/<capacidad>/spec.md` existe con los requisitos `REQ-47.1` y `REQ-47.2`
- **Y** ningún provisional queda en la spec viva

#### Scenario: Una capacidad nueva con el patrón de dominio abre su serie

- **DADO** el patrón `RF-<DOM>-<NN>` y una capacidad nueva con un ADDED con un provisional con el dominio `FAC` fijado, sin ningún `RF-FAC-` en la spec viva
- **CUANDO** se ejecuta el paso de archivado del cambio
- **ENTONCES** la capacidad nueva se escribe en la spec viva con `RF-FAC-01`

#### Scenario: Todas las capacidades se planifican a la vez

- **DADO** un cambio con dos capacidades con ADDED provisionales de la misma serie
- **CUANDO** se ejecuta el paso de archivado del cambio
- **ENTONCES** los números asignados son consecutivos entre las capacidades, sin repetirse

#### Scenario: Se reescriben las tareas y las deltas y queda la traza

- **DADO** un cambio con una línea `Cubre: «prov-a»` en `tasks.md` y una delta con `«prov-a»` en un ADDED
- **CUANDO** se ejecuta el paso de archivado del cambio
- **ENTONCES** `tasks.md` y la delta citan el identificador definitivo en lugar del provisional
- **Y** la carpeta del cambio contiene la traza con el provisional, el definitivo y la capacidad

#### Scenario: Las reescrituras ocurren antes de la instantánea

- **DADO** un cambio con provisionales
- **CUANDO** se ejecuta el paso de archivado del cambio y luego la instantánea y el movimiento de la carpeta
- **ENTONCES** los ficheros del cambio archivado son idénticos byte a byte a los de la instantánea previa al movimiento

#### Scenario: Un fallo en cualquier punto no deja el archivado a medias

- **DADO** un cambio con provisionales en dos capacidades y un fallo inyectado al escribir la segunda
- **CUANDO** se ejecuta el paso de archivado del cambio
- **ENTONCES** la spec viva, `tasks.md`, las deltas y la traza conservan sus bytes originales
- **Y** el cambio permanece en `openspec/changes/<cambio>/`

#### Scenario: Un archivado interrumpido se reanuda sin reasignar

- **DADO** un cambio con provisionales cuyo archivado se interrumpió después de asignar y componer y antes de mover la carpeta
- **CUANDO** se ejecuta `axiom sdd archive-check` y se reanuda el paso de archivado
- **ENTONCES** `archive-check` reconoce la asignación ya hecha por el mapa guardado
- **Y** la reejecución completa el archivado con los mismos identificadores y sin duplicar requisitos

#### Scenario: La reejecución no queda bloqueada por las puertas

- **DADO** un cambio `checkpointed` con puertas aprobadas, cuyo archivado se interrumpió tras reescribir `tasks.md` y la delta
- **CUANDO** se reanuda el archivado
- **ENTONCES** el estado no devuelve el cambio a puertas en `pending` ni bloquea la reejecución

#### Scenario: Dos cambios archivados uno tras otro reciben números distintos

- **DADO** dos cambios SDD con un ADDED provisional cada uno en la misma capacidad, con el mismo máximo previo `REQ-31.1`
- **CUANDO** se archivan en sucesión, el segundo tras rebasar sobre la spec viva que ya contiene el primero
- **ENTONCES** el primero recibe `REQ-31.2` y el segundo `REQ-31.3`
- **Y** ninguna de las dos trazas se reescribe a mano

#### Scenario: Un cambio sin provisionales no cambia

- **DADO** un cambio cuyos ADDED llevan identificadores definitivos
- **CUANDO** se ejecuta el paso de archivado del cambio
- **ENTONCES** la spec viva resultante es idéntica byte a byte a la que producía el archivado por dominio
- **Y** las tareas y las deltas conservan sus bytes y no se escribe ninguna traza de asignación

#### Scenario: Una capacidad nueva con un MODIFIED se rechaza en lugar de copiarse

- **DADO** un cambio con una capacidad nueva `pagos`, sin spec canónica, cuya delta trae un MODIFIED
- **CUANDO** se ejecuta el paso de archivado del cambio
- **ENTONCES** se rechaza con el error tipado de capacidad inexistente que nombra `pagos`, la sección MODIFIED y el requisito
- **Y** no se crea `openspec/specs/pagos/spec.md` ni cambia ningún otro fichero

#### Scenario: Una capacidad parcialmente integrada se completa

- **DADO** un cambio con dos ADDED en una capacidad existente, uno de los cuales ya está en la spec viva con el mismo cuerpo por un archivado anterior interrumpido, y una capacidad nueva ya escrita con el mismo contenido
- **CUANDO** se ejecuta el paso de archivado del cambio
- **ENTONCES** la capacidad existente se completa con el ADDED pendiente y se informa como reanudada
- **Y** la capacidad nueva se informa como ya integrada y no se reescribe
- **Y** ningún requisito queda duplicado

#### Scenario: El informe del paso lista capacidades, mapa, duplicados y ficheros

- **DADO** un cambio con un ADDED provisional en una capacidad existente y otro en una capacidad nueva
- **CUANDO** se ejecuta el paso de archivado del cambio con salida JSON
- **ENTONCES** el informe contiene el resultado de cada capacidad, las dos entradas del mapa con su provisional, su definitivo, su capacidad, su requisito y el estado `assigned`, y la lista de ficheros escritos
- **Y** `archive-report.md` incluye la tabla del mapa

#### Scenario: La opción de solo lectura no escribe

- **DADO** un cambio con provisionales
- **CUANDO** se ejecuta el paso de archivado del cambio con su opción de solo lectura
- **ENTONCES** el informe es el mismo que el de la ejecución real, salvo los ficheros escritos
- **Y** ningún fichero del árbol cambia

#### Scenario: Una segunda rama de archivado rebasada detecta el duplicado y se rehace

- **DADO** dos cambios archivados en ramas de archivado distintas que asignaron `REQ-31.2` en la misma capacidad, la primera ya fusionada en la rama de integración
- **CUANDO** la segunda rama de archivado se rebasa y se ejecuta la opción de solo lectura del paso sobre su cambio archivado
- **ENTONCES** finaliza con el error tipado de duplicado que nombra `REQ-31.2` y la continuación de rehacer el archivado desde la rama de integración
- **Y** al rehacerlo, el cambio recibe `REQ-31.3` sin que el autor renumere a mano

#### Scenario: Un nombre superado que no casa se rechaza

- **DADO** un cambio con dos capacidades existentes y `--superseded-requirements "Cobro"`, cuando `Cobro` existe en las dos canónicas o en ninguna
- **CUANDO** se ejecuta el paso de archivado del cambio
- **ENTONCES** se rechaza con un error tipado que nombra `Cobro` y la continuación de fijar la capacidad con el prefijo `<capacidad>/`
- **Y** no se escribe nada

