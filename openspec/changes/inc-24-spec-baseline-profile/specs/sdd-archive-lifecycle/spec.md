# Delta para Ciclo de Vida de Archive, Integración Formal y Sellado Inmutable (INC-24)

> **Incremento:** `inc-24-spec-baseline-profile` · **Tipo:** delta sobre la capacidad viva `sdd-archive-lifecycle`

Cambios de este delta:

- REQ-21.16 (modificado): el rechazo por falta de integración se refiere a la baseline declarada y se documenta la semántica de la evidencia `pr_merged` en multirepo.
- REQ-21.17 (modificado): archive integra en la baseline declarada (la del proyecto si existe; `openspec/specs/` por defecto), nunca crea una segunda, e integra además el delta de contexto en las rutas mantenidas sin tocar nunca las de referencia.
- REQ-24.55, REQ-24.56 y REQ-24.57 (nuevos): integración del delta de contexto con el mecanismo de edición dirigida de `spec-baseline-integration`, condiciones de parada del archivado (incluida la de un almacén de artefactos sin ficheros, REQ-24.62) y «Sin impacto» explícito. El contrato de archive queda así en un solo sitio.

## MODIFIED Requirements

### Requirement: Precondición de Integración o Despliegue para Ejecutar `archive` (REQ-21.16)

La fase `archive` NO DEBE ejecutarse como un paso inmediato tras la finalización de las pruebas locales de desarrollo, aunque `verify-report.md` (o el consolidado de REQ-21.15) tenga veredicto favorable. `archive` SOLO DEBE ejecutarse cuando exista evidencia de que el incremento se ha integrado o desplegado hacia un entorno preproductivo o productivo, mediante un PR formal fusionado hacia la rama principal (u otro evento de despliegue equivalente ya configurado para el proyecto). Si se solicita `archive` sin esa evidencia, el orquestador DEBE rechazar la operación de forma explícita, indicando la precondición concreta que falta, y el cambio DEBE permanecer sin mover a `openspec/changes/archive/`. En las topologías `multirepo` y `monorepo-decoupled`, la evidencia `pr_merged` se verifica contra el grafo git del repositorio de especificaciones (`workspaceRoot`); la integración de un repositorio de código se acredita con las evidencias `deployment` o `attestation`.
(Previously: el rechazo se refería a `openspec/specs/` y no describía la verificación de `pr_merged` en multirepo)

#### Scenario: PR fusionado habilita el archivado

- **DADO** un cambio con `verify-report.md` en veredicto favorable y un PR fusionado hacia la rama principal que integra ese incremento
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

- **DADO** el workspace canónico, un commit fusionado en la rama principal de `repo-specs` y un commit presente solo en `repo-backend`
- **CUANDO** se registra la evidencia `pr_merged` con el commit de `repo-specs`
- **ENTONCES** la evidencia se acepta
- **Y** registrada con el commit presente solo en `repo-backend`, se rechaza por no figurar en el grafo del repositorio de especificaciones

---

### Requirement: Contenido del Archivado (REQ-21.17)

Al ejecutarse `archive` con su precondición satisfecha (REQ-21.16), el sistema DEBE: (1) integrar los deltas del incremento en la baseline declarada, que es la del proyecto si `axiom.yaml` declara `baseline:` (capacidad `spec-baseline-integration`) y `openspec/specs/` por defecto, sin crear nunca una segunda baseline; (2) con `technical_context.maintained` declarado, integrar el delta de contexto del cambio en las rutas mantenidas con el mismo mecanismo de edición dirigida con verificación nativa, registrando la traza por entrada (REQ-24.55), sin modificar nunca las rutas del modo `reference` (REQ-24.30); (3) transferir el incremento a `openspec/changes/archive/`; y (4) con baseline por defecto, actualizar el inventario maestro (`openspec/INDEX.md`) y los roadmaps declarados que referencien el incremento, y con baseline del proyecto no crear ni actualizar `openspec/INDEX.md`. Las integraciones (1) y (2) DEBEN completarse y verificarse antes de la transferencia (3), porque un incremento archivado es inmutable (REQ-21.18).
(Previously: actualizaba solo la especificación viva de `openspec/specs/` y el inventario `openspec/INDEX.md`, sin baseline del proyecto ni contexto técnico)

#### Scenario: Archivado exitoso actualiza incremento, especificación viva e inventario

- **DADO** un cambio cuya precondición de archivado ya está satisfecha y una baseline OpenSpec por defecto
- **CUANDO** se ejecuta `archive`
- **ENTONCES** el incremento se transfiere a `openspec/changes/archive/`
- **Y** la especificación viva en `openspec/specs/` incorpora los requerimientos `ADDED`/`MODIFIED`/`REMOVED` del cambio
- **Y** `openspec/INDEX.md` y los roadmaps afectados reflejan el incremento como archivado

#### Scenario: Archivado con baseline del proyecto integra en los documentos declarados

- **DADO** un cambio con deltas para dos dominios declarados en `baseline.documents` y la precondición satisfecha
- **CUANDO** se ejecuta `archive`
- **ENTONCES** los dos documentos del proyecto incorporan los requisitos `ADDED`/`MODIFIED`/`REMOVED` según el modo de integración de su dominio
- **Y** no se crea `openspec/specs/` ni `openspec/INDEX.md`
- **Y** el incremento se transfiere a `openspec/changes/archive/`

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

---

## ADDED Requirements

### Requirement: Integración del Delta de Contexto en las Rutas Mantenidas (REQ-24.55)

Con `technical_context.maintained` declarado, archive DEBE integrar el delta de contexto del cambio en las rutas del modo `maintained` mediante el mecanismo de edición dirigida con verificación nativa y traza de la capacidad `spec-baseline-integration` (REQ-24.50 a REQ-24.52): por cada entrada del delta, archive edita el documento y la sección ancla según la acción (`add`, `modify` o `remove`) y la traza registra el documento, el ancla, la acción y el estado. Si una entrada `add` apunta a un documento que aún no existe, archive PUEDE crearlo solo dentro de las rutas del modo `maintained` y DEBE registrarlo en la traza con la acción `created`. Archive NO DEBE modificar rutas del modo `reference`, ni documentos no nombrados en las entradas del delta, ni generar un contexto técnico inicial a partir del código. El texto que archive integra en los documentos mantenidos DEBE contener los IDs definitivos en lugar de los IDs provisionales del cambio (REQ-24.59), y el delta de contexto de la carpeta del cambio se archiva sin alteración.

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

#### Scenario: Los IDs provisionales se sustituyen en el texto integrado del contexto
- **DADO** una entrada `modify` cuyo motivo cita `RF-9001`, ID provisional de un requisito ADDED del cambio con ID definitivo `RF-008`
- **CUANDO** archive integra el cambio
- **ENTONCES** el texto integrado en el documento mantenido cita `RF-008` y no `RF-9001`
- **Y** el delta de contexto de la carpeta del cambio conserva `RF-9001` sin alteración
- **Y** la traza registra la correspondencia de `RF-9001` a `RF-008`

---

### Requirement: Condiciones de Parada del Archivado con Baseline o Contexto del Proyecto (REQ-24.56)

Archive DEBE detenerse sin mover el cambio a `openspec/changes/archive/`, nombrando la causa y la continuación, cuando: (a) con `technical_context.maintained` declarado, el cambio no lleva delta de contexto; (b) el delta de contexto es inválido (REQ-24.31 y REQ-24.32); (c) falla una precondición de la integración (REQ-24.49); (d) un dominio del cambio no está declarado en la baseline (REQ-24.48); (e) falla la composición (REQ-24.46, incluida la frontera de bloque de REQ-24.42) o la verificación nativa (REQ-24.51); (f) la declaración de `axiom.yaml` es inválida (REQ-24.18); o (g) con baseline del proyecto o `technical_context.maintained` declarados, el almacén de artefactos es `engram` o `none` (REQ-24.62), porque sin la carpeta del cambio en ficheros no hay delta de contexto ni traza que integrar; la continuación es declarar `artifact_store: openspec` o `hybrid` en `openspec/config.yaml`, o retirar la declaración. Archive NO DEBE asumir «Sin impacto» ante la ausencia del delta. Cuando la parada ocurre tras ediciones dirigidas ya realizadas, estas DEBEN quedar como cambios sin confirmar en el árbol de trabajo, informadas y revertibles con git, y el cambio permanece en `openspec/changes/`.

#### Scenario: Falta el delta de contexto
- **DADO** `technical_context.maintained` declarado y un cambio sin delta de contexto
- **CUANDO** se solicita el archivado
- **ENTONCES** archive se detiene pidiendo registrar el impacto o «Sin impacto»
- **Y** el cambio permanece en `openspec/changes/`

#### Scenario: Falla una precondición de la integración
- **DADO** un documento afectado con cambios sin confirmar
- **CUANDO** se solicita el archivado
- **ENTONCES** archive se detiene nombrando el documento y la continuación
- **Y** ningún documento se modifica

#### Scenario: Falla la verificación nativa tras una edición dirigida
- **DADO** una edición dirigida cuyo diff toca un documento no declarado
- **CUANDO** la verificación nativa falla
- **ENTONCES** archive se detiene nombrando el fichero
- **Y** las ediciones realizadas quedan como cambios sin confirmar, informadas y revertibles con git
- **Y** el cambio permanece en `openspec/changes/`

#### Scenario: Declaración inválida
- **DADO** un `axiom.yaml` con un bloque `baseline:` inválido
- **CUANDO** se solicita el archivado
- **ENTONCES** archive se detiene nombrando la causa y `axiom workspace validate`

#### Scenario: Almacén de artefactos sin ficheros
- **DADO** una baseline del proyecto o `technical_context.maintained` declarados y `artifact_store: engram` (o `none`) en `openspec/config.yaml`
- **CUANDO** se solicita el archivado
- **ENTONCES** archive se detiene nombrando el almacén declarado y la continuación (declarar `openspec` o `hybrid`, o retirar la declaración)
- **Y** no se modifica ningún documento ni se archiva el cambio

#### Scenario: Sin baseline del proyecto ni contexto mantenido el almacén Engram no detiene el archivado
- **DADO** un proyecto con `artifact_store: engram`, sin baseline del proyecto ni contexto mantenido, y con la precondición de archivado satisfecha
- **CUANDO** se solicita el archivado
- **ENTONCES** archive no se detiene por el almacén y se comporta como antes de esta capacidad

---

### Requirement: «Sin Impacto» Explícito en el Archivado (REQ-24.57)

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
