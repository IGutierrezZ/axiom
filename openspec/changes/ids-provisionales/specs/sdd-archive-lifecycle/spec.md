# Delta para sdd-archive-lifecycle

El archivado SDD pasa a estar dirigido por un paso de Go que cubre todo el cambio: asigna los identificadores provisionales, compone todas las capacidades (también las nuevas), reescribe las tareas y las deltas y escribe la traza, antes de transferir el cambio (REQ-21.17, REQ-47.8). `axiom sdd archive-compose` conserva su salida byte a byte para toda composición aceptada hasta hoy, rechaza con un error tipado un ADDED provisional remitiendo al paso nuevo, y distingue el identificador inexistente con un error tipado propio (REQ-30.22).

Notación de los escenarios: `«prov-a»`, `«prov-b»` son marcadores neutros de un identificador provisional; NO son el formato real, que fija el diseño (véase REQ-47.1 de `requirement-identifiers`). El nombre del verbo o del modo del paso nuevo lo fija el diseño: aquí se llama «el paso de archivado del cambio».

## ADDED Requirements

### Requirement: Paso de Archivado del Cambio Entero Dirigido por Go, Capacidades Nuevas Incluidas (REQ-47.8)

El archivado SDD DEBE disponer de un paso nativo (verbo o modo; el nombre lo fija el diseño) que cubra todo el cambio y que, antes de la instantánea y del movimiento de la carpeta del cambio a `openspec/changes/archive/`, DEBE:

1. Planificar las asignaciones de todos los identificadores provisionales de todas las capacidades del cambio en un plan único (REQ-47.3, REQ-47.2), validado antes de escribir.
2. Componer todas las capacidades del cambio sobre la spec viva, tanto las que modifican una especificación existente como las capacidades nuevas (sin spec canónica), que hoy se copian sin pasar por Go y dejarían los provisionales en `openspec/specs/`. Una capacidad nueva con ADDED provisionales se archiva con identificadores definitivos, incluida la apertura de una serie nueva (`REQ-<n>.<m>` y `RF-<DOM>-<NN>`).
3. Reescribir las líneas `Cubre:` de los ficheros de tareas del cambio y las deltas del cambio (REQ-47.4).
4. Escribir la traza del mapa de provisional a definitivo con la capacidad de cada entrada en la carpeta del cambio (REQ-47.6).

Todo el paso DEBE ser atómico (REQ-47.4): un fallo en cualquier punto no deja a medias la spec viva, las tareas, las deltas ni la traza. Las reescrituras DEBEN ocurrir antes de la instantánea previa al movimiento de la carpeta, de modo que los ficheros del cambio archivado conservan, byte a byte, el contenido de esa instantánea (`diff -r` de la skill de archivado). Un cambio sin provisionales produce la misma spec viva, las mismas tareas y las mismas deltas que producía el archivado por dominio hasta hoy.

El archivado interrumpido DEBE ser reanudable: `axiom sdd archive-check` DEBE reconocer la asignación ya hecha usando el mapa guardado (REQ-47.6) y no volver a asignar números, de modo que tras una interrupción la reejecución del paso termina el archivado sin duplicar requisitos ni cambiar los identificadores ya asignados. Si el estado del cambio reevalúa puertas por el digest de las tareas o de la delta, la reescritura NO DEBE dejar la reejecución bloqueada por puertas en `pending`.

Dos cambios SDD que se archivan uno tras otro con ADDED provisionales en la misma capacidad DEBEN recibir números distintos sin que el autor renumere a mano: el segundo calcula el siguiente libre sobre la spec viva que ya contiene el primero.

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

## MODIFIED Requirements

### Requirement: Contenido del Archivado (REQ-21.17)

Al ejecutarse `archive` con su precondición satisfecha (REQ-21.16), el sistema DEBE: (1) integrar los deltas del incremento en la spec viva `openspec/specs/`, sin crear nunca una segunda spec viva, asignando antes el número definitivo de los identificadores provisionales de todas las capacidades en un plan único, componiendo también las capacidades nuevas, reescribiendo el provisional en la spec viva, las líneas `Cubre:` de las tareas y las deltas del cambio y escribiendo la traza del mapa de provisional a definitivo (REQ-47.8); (2) con `technical_context.maintained` declarado, integrar el delta de contexto del cambio en las rutas mantenidas con el mismo mecanismo de edición dirigida con verificación nativa, registrando la traza por entrada (REQ-24.55), sin modificar nunca las rutas del modo `reference` (REQ-24.30); (3) transferir el incremento a `openspec/changes/archive/`; y (4) actualizar el inventario maestro (`openspec/INDEX.md`) y los roadmaps declarados que referencien el incremento. Las integraciones (1) y (2), incluida la asignación y la reescritura, DEBEN completarse y verificarse antes de la transferencia (3), porque un incremento archivado es inmutable (REQ-21.18).
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

### Requirement: Composición Nativa de OpenSpec con `axiom sdd archive-compose --canonical --delta` (REQ-30.22)

La composición de una delta sobre una especificación canónica y `axiom sdd archive-compose --canonical <ruta> --delta <ruta>` DEBEN producir la salida con identidad del requisito por su identificador cuando el nombre de la delta contiene un identificador válido según el patrón del proyecto (REQ-44.4 de `requirement-identifiers`; sin contexto de proyecto rige el patrón por defecto) y por el texto exacto del nombre si no lo contiene, orden de aplicación RENAMED, MODIFIED, REMOVED y ADDED, rechazo sin escribir ante una delta no aplicable y opciones `--supersede` y `--superseded-requirements`. El identificador de un nombre es el último identificador válido entre paréntesis.

**Resolución por identificador.** Se aplica a MODIFIED, REMOVED y al nombre antiguo de un RENAMED. Con un identificador en el nombre, se buscan los requisitos de la canónica que NO estén superados (los que no llevan el sufijo ` [SUPERSEDED / DEPRECADO]`) con ese identificador: si hay exactamente uno, es ese; si no hay ninguno, solo vale el requisito cuyo título coincide exactamente con el nombre (así se conserva la composición de hoy cuando la delta cita tal cual un requisito superado o un título con dos identificadores, REQ-30.23), y sin él la composición se rechaza sin escribir con el error tipado de identificador inexistente, que nombra la sección y el identificador y conserva el nombre del requisito (REQ-47.3); si hay varios, rige la coincidencia por título exacto cuando exactamente uno lo tiene, y si no la composición DEBE rechazarse sin escribir con un motivo tipado que nombre el identificador y los requisitos afectados (no se resuelve en silencio hacia uno de ellos). Sin identificador en el nombre, se busca el título exacto (primera coincidencia), como hasta ahora, y un nombre sin identificador que no coincide se rechaza como hoy (`no canonical requirement named "<nombre>"`). Un MODIFIED cuyo título cambió pero conserva el identificador se aplica sobre el requisito que lleva ese identificador y lo sustituye por el bloque de la delta, que pasa a ser el nombre del requisito para las operaciones siguientes de la misma delta. Un REMOVED por identificador elimina (o, con `--supersede`, marca como superado) el requisito de la canónica resuelto, y la anotación usa el nombre que ese requisito tiene en la canónica. Un RENAMED sigue exigiendo ambos nombres (`Antiguo → Nuevo`) y la nota `(Reason: …)`. Los requisitos superados quedan excluidos de la resolución por identificador (su identificador está retirado y un MODIFIED no los reabre), pero conservan su `(ID)` en el título y siguen contando como identificador usado en el registro de la spec viva.

**Por título exacto, sin cambios.** ADDED (existencia previa), la colisión del nombre nuevo de un RENAMED y `--superseded-requirements` siguen emparejándose por título exacto: el compositor no rechaza un ADDED ni un nuevo nombre de RENAMED cuyo identificador ya figura bajo otro título; esa unicidad la imponen antes de componer REQ-29.3 (ISO) y REQ-30.4 (ODD).

**Provisionales.** `axiom sdd archive-compose` NO DEBE aceptar un ADDED cuyo nombre lleve un identificador provisional (REQ-47.1): DEBE rechazarlo sin escribir con un error tipado que remita al paso de archivado del cambio (REQ-47.8), que es el que asigna el número. Tampoco DEBE aceptar un provisional en MODIFIED, REMOVED o RENAMED. La composición de una delta sin provisionales no cambia.

**Archivado interrumpido (D17).** `axiom sdd archive-check` calcula «ya compuesto» por nombre para ADDED, MODIFIED y RENAMED, y por la misma resolución por identificador para REMOVED (un REMOVED se da por compuesto cuando ya no se resuelve en la canónica); con provisionales, reconoce la asignación ya hecha por el mapa guardado (REQ-47.8). La clasificación idempotente de la integración ODD (REQ-30.2) resuelve MODIFIED y REMOVED con esa misma resolución: un MODIFIED integrado deja la canónica con el nombre nuevo y una segunda ejecución informa `already`. Compositor, `archive-check` y clasificación ODD DEBEN coincidir siempre.

Toda composición aceptada antes de este cambio produce la misma salida byte a byte (REQ-30.23). `archive-compose` NO DEBE aceptar los modos `--document` ni `--in-place`: su uso DEBE rechazarse con un motivo tipado que indique que la baseline ajena se retiró y que la composición es `--canonical --delta`. (Es REQ-24.38 de `spec-baseline-integration`, aplicable a OpenSpec, más la retirada de los modos de documento.) Con un bloque `sdd:` inválido la composición en el contexto de un proyecto se niega sin escribir (REQ-44.9 de `requirement-identifiers`).
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
- **ENTONCES** cada salida es idéntica byte a byte a la anterior (REQ-30.23)

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
- **Y** el compositor no valida la unicidad del identificador (la imponen REQ-29.3 y REQ-30.4 antes de componer)

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

#### Scenario: Un REMOVED con un identificador inexistente tiene su error tipado
- **DADO** una canónica sin `REQ-30.9` y una delta con un REMOVED `Cobro (REQ-30.9)`
- **CUANDO** se compone
- **ENTONCES** finaliza con código distinto de cero con el error tipado de identificador inexistente que nombra la sección REMOVED y `REQ-30.9`
- **Y** no se escribe el fichero de salida
