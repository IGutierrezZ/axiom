# Delta para sdd-archive-lifecycle

> Parte del estado vigente: la precondición de integración (REQ-21.16), el sellado post-archive (REQ-21.18), el «Sin impacto» explícito (REQ-24.57), el modo de archivado en monorepo (REQ-25.39) y la publicación de la documentación (REQ-27.14) NO cambian. Esta delta (a) deja la spec viva única `openspec/specs/` como destino de la integración, (b) actualiza REQ-24.55 y REQ-24.56 al contexto sin baseline ajena ni identificadores provisionales, y (c) recibe las reglas de composición que sobreviven de `spec-baseline-integration`.
> Coordinación con `carril-sdd-versatil`: su primera enmienda retiró su MODIFIED de REQ-24.46 y la identidad `id` (C10), así que REQ-30.22 y REQ-30.24 reubican el texto vigente de REQ-24.38 y REQ-24.46 sin ninguna pieza ISO de baseline declarada. La regla ISO de escenarios de REQ-29.10 es de carril y no se redefine aquí.

## ADDED Requirements

### Requirement: Composición Nativa de OpenSpec con `axiom sdd archive-compose --canonical --delta` (REQ-30.22)

La composición de una delta sobre una especificación canónica y `axiom sdd archive-compose --canonical <ruta> --delta <ruta>` DEBEN producir la salida con identidad por el texto exacto del nombre del requisito, orden de aplicación RENAMED, MODIFIED, REMOVED y ADDED, rechazo sin escribir ante una delta no aplicable y opciones `--supersede` y `--superseded-requirements`. Toda composición aceptada produce la misma salida byte a byte que antes de este cambio. `archive-compose` NO DEBE aceptar los modos `--document` ni `--in-place`: su uso DEBE rechazarse con un motivo tipado que indique que la baseline ajena se retiró y que la composición es `--canonical --delta`. (Es REQ-24.38 de `spec-baseline-integration`, aplicable a OpenSpec, más la retirada de los modos de documento.)

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

---

### Requirement: Fidelidad de Bytes al Componer (REQ-30.23)

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

### Requirement: Validación Completa antes de Escribir (REQ-30.24)

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

### Requirement: Retirada de `axiom sdd integrate` (REQ-30.25)

El subcomando `axiom sdd integrate` NO DEBE existir. Su invocación DEBE rechazarse con un motivo tipado que remita a `axiom archive context check|prepare|verify|show` para el contexto técnico (REQ-30.15 de `project-technical-context`) y a `axiom archive adopt` para la migración (REQ-30.9 y REQ-30.11 de `openspec-adoption`), sin escribir nada. Las skills y la doctrina gestionadas NO DEBEN citar `axiom sdd integrate`.

#### Scenario: `axiom sdd integrate` remite a los verbos nuevos
- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom sdd integrate check --change <cambio>`
- **ENTONCES** finaliza con código distinto de cero con un motivo tipado que nombra `axiom archive context` y `axiom archive adopt`
- **Y** no se escribe ningún fichero

#### Scenario: Las skills no citan el verbo retirado
- **DADO** los textos de las skills y la doctrina gestionadas
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** ninguno contiene `axiom sdd integrate`

## MODIFIED Requirements

### Requirement: Contenido del Archivado (REQ-21.17)

Al ejecutarse `archive` con su precondición satisfecha (REQ-21.16), el sistema DEBE: (1) integrar los deltas del incremento en la spec viva `openspec/specs/`, sin crear nunca una segunda spec viva; (2) con `technical_context.maintained` declarado, integrar el delta de contexto del cambio en las rutas mantenidas con el mismo mecanismo de edición dirigida con verificación nativa, registrando la traza por entrada (REQ-24.55), sin modificar nunca las rutas del modo `reference` (REQ-24.30); (3) transferir el incremento a `openspec/changes/archive/`; y (4) actualizar el inventario maestro (`openspec/INDEX.md`) y los roadmaps declarados que referencien el incremento. Las integraciones (1) y (2) DEBEN completarse y verificarse antes de la transferencia (3), porque un incremento archivado es inmutable (REQ-21.18).
(Previously: la integración (1) podía dirigirse a una baseline del proyecto declarada con `baseline:` y, con ella, no se creaba ni actualizaba `openspec/INDEX.md`; la baseline ajena se retira)

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

---

### Requirement: Integración del Delta de Contexto en las Rutas Mantenidas (REQ-24.55)

Con `technical_context.maintained` declarado, archive DEBE integrar el delta de contexto del cambio en las rutas del modo `maintained` mediante el mecanismo de edición dirigida con verificación nativa y traza de la capacidad `project-technical-context` (REQ-30.16 a REQ-30.18): por cada entrada del delta, archive edita el documento y la sección ancla según la acción (`add`, `modify` o `remove`) y la traza registra el documento, el ancla, la acción y el estado. Si una entrada `add` apunta a un documento que aún no existe, archive PUEDE crearlo solo dentro de las rutas del modo `maintained` y DEBE registrarlo en la traza con la acción `created`. Archive NO DEBE modificar rutas del modo `reference`, ni documentos no nombrados en las entradas del delta, ni generar un contexto técnico inicial a partir del código. El delta de contexto de la carpeta del cambio se archiva sin alteración.
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

### Requirement: Condiciones de Parada del Archivado con Baseline o Contexto del Proyecto (REQ-24.56)

Archive DEBE detenerse sin mover el cambio a `openspec/changes/archive/`, nombrando la causa y la continuación, cuando: (a) con `technical_context.maintained` declarado, el cambio no lleva delta de contexto; (b) el delta de contexto es inválido (REQ-24.31 y REQ-24.32); (c) falla una precondición de la integración del contexto (REQ-30.29); (d) falla la composición (REQ-30.24) o la verificación nativa (REQ-30.17); (e) con `technical_context.maintained` declarado, el almacén de artefactos es `engram` o `none` (REQ-30.19), porque sin la carpeta del cambio en ficheros no hay delta de contexto ni traza que integrar; la continuación es declarar `artifact_store: openspec` o `hybrid` en `openspec/config.yaml`, o retirar la declaración; o (f) la declaración de `axiom.yaml` es inválida: un `technical_context:` que no supera la validación (REQ-30.30) o un bloque `baseline:` residual (REQ-30.21 de `workspace-topology`). Archive NO DEBE asumir «Sin impacto» ante la ausencia del delta. Cuando la parada ocurre tras ediciones dirigidas ya realizadas, estas DEBEN quedar como cambios sin confirmar en el árbol de trabajo, informadas y revertibles con git, y el cambio permanece en `openspec/changes/`.
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
- **ENTONCES** archive se detiene con el motivo tipado de REQ-30.21 y la continuación de retirar `baseline:` y migrar con `axiom archive adopt`
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

### Requirement: Precondición de Integración o Despliegue para Ejecutar `archive` (REQ-21.16)

La fase `archive` NO DEBE ejecutarse como un paso inmediato tras la finalización de las pruebas locales de desarrollo, aunque `verify-report.md` (o el consolidado de REQ-21.15) tenga veredicto favorable. `archive` SOLO DEBE ejecutarse cuando exista evidencia de que el incremento se ha integrado o desplegado hacia un entorno preproductivo o productivo, mediante un PR formal fusionado hacia la rama de integración (`git.integration_branch`) u otro evento de despliegue equivalente ya configurado para el proyecto. Si se solicita `archive` sin esa evidencia, el orquestador DEBE rechazar la operación de forma explícita, indicando la precondición concreta que falta, y el cambio DEBE permanecer sin mover a `openspec/changes/archive/`. En la topología `multirepo`, la evidencia `pr_merged` se verifica contra el grafo git del repositorio de especificaciones (`workspaceRoot`); la integración de un repositorio de código se acredita con las evidencias `deployment` o `attestation`.
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

### Requirement: En Monorepo el Archive se Ejecuta en la Rama de Archivado y se Entrega en un PR Documental Aparte (REQ-25.39)

En los repositorios con rama de cambio (`monorepo-embedded`), `archive` DEBE ejecutarse en la rama de archivado `<prefijo>/<cambio>-archive` (REQ-25.45 de `change-branch-lifecycle`), que se crea con `axiom branch ensure --change <c> --archive` **después** de fusionar en la rama de integración el PR del cambio, y su resultado DEBE entregarse en un PR documental aparte contra la rama de integración. La precondición de `archive` es la de REQ-21.16, sin excepción: evidencia `pr_merged` del PR del cambio (verificada contra el grafo git del repositorio del planning home), veredicto favorable de verificación y compuertas aprobadas. Archivar en cualquier otra rama (la del cambio, la de integración, una de rol o una ajena) DEBE rechazarse: `axiom sdd status` proyecta `blocked(wrong_branch)` en la fase archive con la continuación exacta `axiom branch ensure --cwd <ruta> --change <c> --archive` (REQ-25.42 de `axiom-sdd-cli-integration`) y el escritor de archive se niega con `wrong_branch` (REQ-25.19), nombrando la rama esperada y sin mover el cambio. El PR documental contiene solo documentación: el movimiento del cambio a `openspec/changes/archive/`, la integración en `openspec/specs/` y los índices. Mientras el cambio no se haya archivado, el cambio fusionado figura como pendiente de archivar (REQ-25.20) sin bloquear nada. Tras fusionar el PR documental, `axiom branch done` elimina la rama de archivado (REQ-25.16 y REQ-25.17). En el repositorio de specs de un `multirepo` NO existe rama de archivado: `archive` se ejecuta in situ en la rama de integración, conserva la precondición de REQ-21.16 (evidencia `pr_merged` contra el repositorio de especificaciones, `deployment` o `attestation`) y no crea ramas ni worktrees (REQ-25.5). La skill `sdd-archive` DEBE indicar este orden y la rama.
(Previously: el PR documental contenía la integración «en la baseline declarada»; la baseline ajena se retira)

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

---

### Requirement: Publicación de la Documentación Tras el Archivado (REQ-27.14)

Una vez completado `archive`, la documentación que toca el archivado DEBE poder publicarse con `axiom sdd publish --change <x> --archive` (capacidad `sdd-docs-publication`) sobre la rama donde se archivó (el indicador `--archive` selecciona la rama de archivado como rama esperada de `publish`): en `monorepo-embedded`, la rama de archivado `<prefijo>/<cambio>-archive` que prepara `axiom branch ensure --change <x> --archive` (REQ-25.45 de `change-branch-lifecycle`); en el repositorio de specs de un `multirepo`, su rama de integración, donde se archiva in situ. La publicación DEBE incluir únicamente rutas de la lista cerrada de documentación (REQ-27.5): la carpeta archivada del cambio bajo `openspec/changes/archive/`, el borrado del directorio `openspec/changes/<x>/`, las especificaciones vivas de `openspec/specs/` que integró el archivado, `openspec/INDEX.md` y las rutas de contexto técnico mantenido a las que el archivado integró su delta (REQ-24.55). NO DEBE incluir código ni rutas de `technical_context.reference`. En un cambio `continuous` con `docs_publication: auto` el orquestador DEBE ejecutar la publicación al cerrar el archivado, sin preguntar (REQ-27.13); en un cambio sin ese campo DEBE ofrecer `axiom sdd publish --change <x> --archive` como siguiente paso, y en un cambio `checkpointed` DEBE ofrecerlo igualmente, sin parada adicional. `archive` NO DEBE ejecutar `git commit` ni `git push` por sí mismo: la publicación es una orden aparte. Una negativa de `publish` (rama equivocada, almacén sin ficheros, remoto inexistente) es informativa y NO revierte el archivado. La PR documental contra la integración y su merge siguen siendo decisión de la persona (REQ-25.39).
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
