# Delta para odd-living-spec-integration

La integración ODD pasa a asignar el número de los ADDED provisionales y a reescribir el documento ODD dentro del mismo conjunto atómico (REQ-30.2), a reconocer lo ya integrado y a reintegrar tras un rebase (REQ-30.3), a aceptar el provisional o el definitivo en el ADDED con errores tipados para el identificador inexistente y el provisional fuera de ADDED (REQ-30.4), y gana una opción de solo lectura (REQ-47.7).

Notación de los escenarios: `«prov-a»`, `«prov-b»` son marcadores neutros de un identificador provisional; NO son el formato real, que fija el diseño (véase REQ-47.1 de `requirement-identifiers`).

## ADDED Requirements

### Requirement: `axiom archive integrate --odd <feature> --check` Proyecta los Requisitos y el Mapa Previsto sin Escribir (REQ-47.7)

`axiom archive integrate --odd <feature>` DEBE aceptar una opción de solo lectura `--check` que valide el documento ODD y planifique la integración exactamente como lo haría la integración real (secciones, colisiones, identificadores inexistentes, provisionales mal usados, duplicados entre capacidades, comprobación del contexto), pero NO DEBE escribir ningún fichero: ni la spec viva, ni `openspec/INDEX.md`, ni el documento ODD, ni temporales. La salida JSON del informe (`axiom.odd-integration/v1`) DEBE proyectar, con la opción `--check`, un campo `requirements` con las listas `modifies` (los identificadores existentes que la delta toca: MODIFIED, REMOVED y el nombre antiguo de RENAMED) y `creates` (los requisitos que la delta crea: ADDED), y el mapa previsto de provisional a definitivo con la capacidad de cada entrada (REQ-47.6). El campo `requirements` es aditivo: no cambia ningún campo anterior del informe. Con la opción `--check` el código de salida es `0` si la integración real sería aceptada y distinto de cero, con el mismo motivo tipado, si sería rechazada. Es el equivalente mínimo de `axiom sdd status` para ODD: no existe un status ODD completo.

«Exactamente como la integración real» significa la misma secuencia de comprobaciones: configuración (`baseline:` residual, negativa ISO de REQ-30.7, patrón accesible de REQ-44.9), documento y secciones (REQ-30.1, REQ-30.5), plan único (REQ-47.3), clasificación de lo ya integrado (REQ-30.3), colisiones (REQ-30.4), composición en memoria, duplicados entre capacidades (REQ-47.5) y comprobación estructural del contexto técnico de la rama (REQ-30.6). Solo se omiten las dos cosas que existen para escribir: la guarda de rama del escritor y la regeneración de `openspec/INDEX.md`, que es un derivado de la spec viva y no puede rechazar una integración. Cada entrada del mapa previsto lleva su estado: `assigned` (número nuevo), `kept` (número que el mapa guardado ya asignó y se conserva) o `reassigned` (número ocupado por otro cambio que se sustituiría, con el definitivo anterior). Sobre un documento ya integrado, `--check` termina con `0`, informa de que no hay nada que integrar, proyecta `requirements` con los identificadores que el documento ya lleva y el mapa guardado con todas sus entradas en estado `kept`.

#### Scenario: `--check` no escribe nada

- **DADO** un documento ODD con un ADDED provisional y un MODIFIED de un requisito existente
- **CUANDO** se ejecuta `axiom archive integrate --odd <feature> --check`
- **ENTONCES** ningún fichero del árbol cambia (spec viva, `openspec/INDEX.md`, documento ODD)
- **Y** el código de salida es `0`

#### Scenario: `--check` proyecta `requirements` y el mapa previsto

- **DADO** un documento ODD con un ADDED provisional en `facturacion` y un MODIFIED sobre `REQ-30.1`
- **CUANDO** se ejecuta la integración con `--check` y salida JSON
- **ENTONCES** `requirements.modifies` contiene `REQ-30.1`
- **Y** `requirements.creates` contiene el requisito ADDED
- **Y** el mapa previsto asocia el provisional con el identificador definitivo que se asignaría y la capacidad `facturacion`

#### Scenario: `--check` rechaza lo que la integración rechazaría

- **DADO** un documento ODD con un MODIFIED sobre un identificador inexistente
- **CUANDO** se ejecuta la integración con `--check`
- **ENTONCES** finaliza con código distinto de cero con el mismo error tipado que la integración real
- **Y** no se escribe nada

#### Scenario: `--check` detecta el duplicado entre capacidades

- **DADO** una spec viva con `REQ-47.1` en dos capacidades y un documento ODD que escribe uno de esos identificadores
- **CUANDO** se ejecuta la integración con `--check`
- **ENTONCES** finaliza con el error tipado de duplicado entre capacidades (REQ-47.5)

#### Scenario: Sin `--check` el informe conserva sus campos

- **DADO** una integración real sin la opción `--check`
- **CUANDO** se consulta el informe
- **ENTONCES** conserva todos los campos que tenía antes de este cambio

#### Scenario: `--check` sobre un documento ya integrado

- **DADO** un documento ODD con un ADDED provisional ya integrado, con el documento reescrito y el mapa guardado
- **CUANDO** se ejecuta la integración con `--check` y salida JSON
- **ENTONCES** el código de salida es `0` y el informe indica que no hay nada que integrar
- **Y** `requirements.creates` contiene el identificador definitivo que el documento ya lleva
- **Y** el mapa previsto contiene la entrada guardada en estado `kept`

#### Scenario: `--check` distingue una reasignación

- **DADO** un documento ODD ya integrado cuyo definitivo asignado ocupó después otro cambio con un cuerpo distinto, tras un rebase
- **CUANDO** se ejecuta la integración con `--check` y salida JSON
- **ENTONCES** el mapa previsto contiene esa entrada en estado `reassigned`, con el definitivo anterior y el nuevo
- **Y** ningún fichero cambia

#### Scenario: `--check` aplica la comprobación del contexto y no regenera el índice

- **DADO** un proyecto con `technical_context` cuya rama modifica una ruta de referencia, y un `openspec/INDEX.md` existente
- **CUANDO** se ejecuta la integración con `--check`
- **ENTONCES** finaliza con el mismo motivo tipado de la comprobación del contexto que la integración real
- **Y** `openspec/INDEX.md` conserva sus bytes

## MODIFIED Requirements

### Requirement: `axiom archive integrate --odd <feature>` Compone la Delta sobre la Spec Viva (REQ-30.2)

`axiom archive integrate --odd <feature>` DEBE leer `odd/tasks/<feature>.md` de la rama actual, validar sus secciones (REQ-30.1, REQ-30.5), comprobar las colisiones de identificadores (REQ-30.4) y la comprobación estructural del contexto (REQ-30.6), planificar la asignación de los identificadores provisionales de todas las capacidades del documento en un plan único (REQ-47.3, REQ-47.2) y componer cada delta sobre `openspec/specs/<capacidad>/spec.md` con la misma composición nativa que usa el archivado SDD (REQ-30.22 de `sdd-archive-lifecycle`), ya con los identificadores definitivos. Todas las composiciones y la reescritura del documento ODD DEBEN validarse y componerse en memoria antes de escribir ninguna: o se aplican todas o ninguna (REQ-47.4). Cuando el documento contiene identificadores provisionales, la orden DEBE reescribirlos al definitivo en el propio documento ODD (la valla `openspec` de `## Spec viva` y sus líneas `Cubre:`) dentro del mismo conjunto atómico de escrituras que la spec viva, y DEBE dejar el mapa de provisional a definitivo en el documento y en el informe (REQ-47.6). La orden DEBE dejar el índice `openspec/INDEX.md` coherente con las especificaciones integradas cuando el proyecto lo mantiene. La orden NO DEBE tener compuertas: ni paradas, ni preguntas, ni aprobación humana (REQ-27.17 de `organic-agent-trigger-rules`); NO DEBE crear artefactos SDD (`openspec/changes/<cambio>/`, `kickoff.yaml`, `gates.yaml`), NO DEBE escribir fuera de `openspec/specs/`, `openspec/INDEX.md` y el documento ODD, y NO DEBE modificar el documento ODD cuando este no contiene provisionales que asignar (conserva sus bytes). Un documento inexistente, una sección inválida, una composición no aplicable o un rechazo de las comprobaciones DEBE finalizar con código distinto de cero y con un motivo tipado que nombre la sección o el requisito y la continuación, sin escribir nada, tampoco en el documento ODD. La opción de solo lectura `--check` se rige por REQ-47.7.
(Previously: la orden no modificaba nunca el documento ODD y escribía solo en `openspec/specs/` e `openspec/INDEX.md`; ahora asigna los provisionales y reescribe el documento ODD dentro del mismo conjunto atómico, y la garantía de «no modifica el documento» se limita a un documento sin provisionales y a `--check`.)

Supuesto: la actualización de `openspec/INDEX.md` sigue el criterio de `living-documentation` (REQ-1.2): el diseño fija si la regeneración es siempre o solo cuando el índice ya existe.

#### Scenario: La delta se integra en la spec viva
- **DADO** un documento ODD con `## Spec viva` que añade un requisito `(REQ-31.1)` a la capacidad `facturacion`, sin colisión
- **CUANDO** se ejecuta `axiom archive integrate --odd <feature>`
- **ENTONCES** `openspec/specs/facturacion/spec.md` contiene el requisito
- **Y** el código de salida es `0`
- **Y** no se crea ninguna carpeta bajo `openspec/changes/`

#### Scenario: Todo o nada entre capacidades
- **DADO** un documento ODD con dos capacidades, la segunda con un MODIFIED sobre un requisito que no existe
- **CUANDO** se ejecuta la integración
- **ENTONCES** finaliza con código distinto de cero nombrando la capacidad, la sección y el requisito
- **Y** ninguna especificación de `openspec/specs/` cambia, tampoco la de la primera capacidad
- **Y** el documento ODD conserva sus bytes

#### Scenario: Sin compuertas ni artefactos SDD
- **DADO** una integración válida de un documento sin provisionales
- **CUANDO** se ejecuta en una rama ODD
- **ENTONCES** no pide confirmación ni se detiene
- **Y** no se crean `kickoff.yaml`, `gates.yaml` ni un cambio SDD
- **Y** el documento ODD no se modifica

#### Scenario: Documento inexistente
- **DADO** una rama sin `odd/tasks/inexistente.md`
- **CUANDO** se ejecuta `axiom archive integrate --odd inexistente`
- **ENTONCES** finaliza con código distinto de cero con un motivo tipado que nombra la ruta esperada

#### Scenario: Se asigna el número y se reescribe el documento ODD
- **DADO** un documento ODD con un ADDED con un provisional con la serie `31` fijada en la capacidad `facturacion`, una spec viva con `REQ-31.1` y una línea `Cubre:` del documento que cita ese provisional
- **CUANDO** se ejecuta `axiom archive integrate --odd <feature>`
- **ENTONCES** `openspec/specs/facturacion/spec.md` contiene el requisito con `REQ-31.2`
- **Y** la valla `openspec` del documento ODD y su línea `Cubre:` citan `REQ-31.2` en lugar del provisional
- **Y** el resto del documento conserva sus bytes
- **Y** ningún provisional queda en `openspec/specs/`

#### Scenario: El documento ODD forma parte del conjunto atómico
- **DADO** un documento ODD con provisionales y un fallo inyectado al escribir el último destino
- **CUANDO** se ejecuta la integración
- **ENTONCES** la spec viva, `openspec/INDEX.md` y el documento ODD conservan sus bytes originales

#### Scenario: Se escribe solo en los destinos permitidos
- **DADO** una integración válida con provisionales
- **CUANDO** termina
- **ENTONCES** los únicos ficheros modificados están en `openspec/specs/`, `openspec/INDEX.md` y el propio documento ODD

---

### Requirement: Idempotencia de la Integración ODD (REQ-30.3)

La integración DEBE ser idempotente: si la spec canónica ya contiene cada requisito ADDED o MODIFIED de la delta con el mismo cuerpo, o ya no contiene cada requisito REMOVED, la orden DEBE terminar con éxito sin modificar ningún fichero e informar de que no había nada que integrar. Un requisito ya presente con un cuerpo distinto del de la delta DEBE rechazarse con un motivo tipado que nombre el requisito, sin escribir. Un requisito ya integrado NO DEBE contarse como colisión de identificadores (REQ-30.4). Con identificadores provisionales, la integración DEBE reconocer lo que ya integró aunque el título haya cambiado, usando el mapa guardado en el documento (REQ-47.6) y la igualdad del cuerpo módulo la sustitución de los provisionales por sus definitivos, y no DEBE volver a asignar un número a un requisito ya integrado. Un requisito de la spec viva es «de esta integración» cuando está en la capacidad de la entrada del mapa, lleva su identificador definitivo y, o bien su cuerpo coincide con el del ADDED módulo la sustitución, o bien su título es el que el mapa guardó al integrar. Si es de esta integración y su cuerpo difiere del del ADDED (el autor editó el requisito después de integrarlo), la orden DEBE rechazarlo con el motivo tipado de requisito ya presente con otro cuerpo, sin reasignar ni escribir, con la continuación de restaurar la especificación de esa capacidad desde la base de la rama y volver a integrar (el mapa conserva el número). Tras un rebase sobre un cambio concurrente, la integración se puede volver a ejecutar: si otro cambio ocupó, con un requisito que no es de esta integración, un identificador definitivo que esta integración había asignado, la orden DEBE reasignar el siguiente número libre a ese requisito, reescribir todos los destinos de forma atómica (REQ-47.4), incluido el propio bloque de la spec viva si ya estaba, y actualizar el mapa con el definitivo anterior, sin que el autor renumere a mano. Si el identificador reasignado pertenece a una serie que esta integración abrió con una etiqueta de serie nueva, se reasigna la etiqueta entera a una serie nueva, con números consecutivos desde `1` en todas las capacidades del documento; con una serie fijada se reasigna solo la entrada afectada. La orden DEBE además detectar los identificadores duplicados entre capacidades de la spec viva que git no marcó como conflicto (REQ-47.5). Si un rebase produce un conflicto de git en `openspec/specs/`, el autor conserva la versión de la base y vuelve a ejecutar la integración, que reasigna lo necesario. Tras un rebase, si el nombre exacto de un requisito MODIFIED, REMOVED o RENAMED ya no casa con la spec canónica, la composición DEBE fallar cerrada y el autor corrige la delta.
(Previously: la idempotencia comparaba el cuerpo bajo el mismo identificador y no contemplaba provisionales, la reasignación de un número ocupado por otro cambio ni los duplicados entre capacidades.)

#### Scenario: Repetir la integración no cambia nada
- **DADO** una integración ya aplicada
- **CUANDO** se repite `axiom archive integrate --odd <feature>`
- **ENTONCES** termina con código `0` informando de que no había nada que integrar
- **Y** `openspec/specs/` no cambia byte a byte

#### Scenario: Un requisito ya presente con otro cuerpo se rechaza
- **DADO** un requisito ADDED `(REQ-31.1)` ya integrado con un cuerpo que difiere del de la delta
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza con un motivo tipado que nombra el requisito
- **Y** no se escribe nada

#### Scenario: Un REMOVED ya aplicado es un no-op
- **DADO** una delta con un REMOVED de un requisito que ya no está en la spec canónica
- **CUANDO** se ejecuta la integración
- **ENTONCES** termina con código `0` sin modificar ningún fichero

#### Scenario: Tras un rebase el nombre ya no casa
- **DADO** una delta con un MODIFIED cuyo requisito fue renombrado por un cambio concurrente ya fusionado
- **CUANDO** se ejecuta la integración
- **ENTONCES** falla cerrada nombrando el requisito que no se aplica
- **Y** no se escribe nada

#### Scenario: Repetir una integración con provisionales no reasigna nada
- **DADO** un documento ODD con un ADDED provisional ya integrado, con el documento reescrito y el mapa guardado
- **CUANDO** se repite la integración
- **ENTONCES** termina con código `0` informando de que no había nada que integrar
- **Y** no se asigna ningún número nuevo y ningún fichero cambia

#### Scenario: Un título cambiado no rompe el reconocimiento de lo ya integrado
- **DADO** un ADDED provisional ya integrado y un documento ODD en el que el autor cambió después el título del requisito, con el mismo cuerpo módulo la sustitución del provisional
- **CUANDO** se repite la integración
- **ENTONCES** el requisito se reconoce como ya integrado gracias al mapa guardado
- **Y** no se añade un segundo requisito

#### Scenario: Dos ramas concurrentes reciben números distintos sin renumerar a mano
- **DADO** dos ramas ODD con un ADDED provisional cada una en la misma capacidad `facturacion`, ambas con el mismo máximo previo `REQ-31.1`, de modo que la primera integración asignó `REQ-31.2` y la segunda también
- **Y** la primera rama se fusiona en `main`
- **CUANDO** la segunda rama se rebasa sobre `main` y vuelve a ejecutar `axiom archive integrate --odd <feature>`
- **ENTONCES** `openspec/specs/facturacion/spec.md` contiene ambos requisitos con identificadores distintos, el de la segunda rama con `REQ-31.3`
- **Y** el documento ODD y las líneas `Cubre:` de la segunda rama citan `REQ-31.3`
- **Y** el mapa de la segunda rama refleja `REQ-31.3`
- **Y** ningún identificador queda duplicado en la spec viva

#### Scenario: Un duplicado entre capacidades sin conflicto de git no pasa en silencio
- **DADO** dos ramas que asignaron el mismo número de una serie nueva en capacidades distintas, fusionadas sin conflicto de git
- **CUANDO** se vuelve a ejecutar la integración tras el rebase
- **ENTONCES** se detecta el duplicado con el error tipado de REQ-47.5 o, si el requisito es el de la propia rama, se reasigna el siguiente libre
- **Y** nunca se termina con éxito dejando el mismo identificador en dos capacidades

#### Scenario: Editar el cuerpo de un ADDED ya integrado se rechaza
- **DADO** un ADDED provisional ya integrado como `REQ-31.2`, con el mapa guardado, y un documento ODD en el que el autor cambió después el cuerpo de ese requisito sin cambiar su título
- **CUANDO** se repite la integración
- **ENTONCES** se rechaza con el motivo tipado de requisito ya presente con otro cuerpo, que nombra el requisito y `REQ-31.2`
- **Y** no se reasigna ningún número ni se escribe nada

#### Scenario: Una serie nueva ocupada por otro cambio se reasigna entera
- **DADO** un documento ODD que abrió la serie `REQ-47` con una etiqueta de serie nueva para dos ADDED en dos capacidades, ya integrado, y un cambio concurrente fusionado en `main` que también abrió `REQ-47`
- **CUANDO** la rama se rebasa y vuelve a ejecutar la integración
- **ENTONCES** los dos ADDED reciben `REQ-48.1` y `REQ-48.2`, en el orden del documento
- **Y** el mapa guarda `REQ-47.1` y `REQ-47.2` como definitivos anteriores
- **Y** ningún identificador queda duplicado en la spec viva

---

### Requirement: Comprobación de Colisión de Identificadores de Requisito según el Patrón Declarado (REQ-30.4)

Todo requisito ADDED de una delta ODD DEBE llevar en su título un identificador válido según el patrón declarado del proyecto (REQ-44.1 de `requirement-identifiers`; con el patrón por defecto, `(REQ-n.m)`), elegido por el autor (el siguiente número libre de la serie), o un identificador provisional (REQ-47.1 de `requirement-identifiers`) que la integración sustituye por el siguiente número libre de la serie (REQ-47.2). La integración DEBE rechazar sin escribir un requisito ADDED con identificador definitivo cuyo identificador ya existe en `openspec/specs/` o en una delta activa de `openspec/changes/<cambio>/specs/` (la carpeta `archive/` queda excluida), así como dos requisitos ADDED con el mismo identificador definitivo dentro del propio documento. Un provisional NO colisiona con la delta de otro cambio activo ni con ningún identificador definitivo: solo es único dentro del propio documento (REQ-47.1). El motivo tipado DEBE nombrar el identificador, dónde colisiona (la capacidad o el cambio) y el siguiente número libre de la serie del identificador, calculada por el patrón (REQ-44.3 de `requirement-identifiers`): `REQ-31.2` tras `REQ-31.1` con el patrón por defecto, `RF-USU-02` tras `RF-USU-01` con `RF-<DOM>-<NN>`. Un requisito ADDED sin identificador válido según el patrón ni provisional en el título DEBE rechazarse nombrando el requisito y el patrón en su texto legible (REQ-44.5). Un MODIFIED o el nombre antiguo de un RENAMED pendientes de integrar cuyo nombre lleva un identificador válido DEBEN citar un identificador definitivo que exista en la especificación canónica de su capacidad: si no existe, se rechaza sin escribir con el error tipado de identificador inexistente (REQ-47.3), que nombra la capacidad, la sección, el requisito y el identificador. Un nombre sin identificador se resuelve por el título exacto (REQ-30.22 de `sdd-archive-lifecycle`), un REMOVED que ya no se resuelve es lo ya integrado (REQ-30.3) y una capacidad sin especificación canónica con MODIFIED, REMOVED o RENAMED se rechaza con el error tipado de capacidad inexistente, todo según REQ-47.3. Un provisional en un MODIFIED, un REMOVED o un RENAMED, un provisional repetido en el documento, un nombre con un provisional y otro identificador, y un provisional citado (en `Cubre:` o en el texto que valida REQ-47.4) sin un ADDED que lo declare se rechazan sin escribir con un error tipado propio (REQ-47.1, REQ-47.3). Un requisito que la spec canónica ya contiene con el mismo cuerpo no es una colisión (REQ-30.3).
(Previously: el ADDED debía llevar siempre un identificador definitivo elegido por el autor, un ADDED sin identificador se rechazaba con `added_without_id` y un MODIFIED con un identificador inexistente solo fallaba con el mensaje genérico de composición; ahora el ADDED admite el provisional, y el identificador inexistente y el uso indebido del provisional son errores tipados propios. Un ADDED con identificador definitivo se acepta con la misma comprobación de colisión de antes.)

Supuestos (propuesta §13, P5): el alcance de la comprobación es `openspec/specs/` y las deltas de los cambios activos, sin `archive/`; la obligación de que todo ADDED lleve identificador es una decisión de esta spec porque el indexador exige identificador (`living-documentation`, REQ-1.1); un provisional nunca llega al indexador porque se sustituye antes de escribir en `openspec/specs/`.

#### Scenario: Colisión con la spec viva
- **DADO** una delta ODD con un ADDED `(REQ-31.1)` y un `REQ-31.1` ya existente en `openspec/specs/facturacion/spec.md`
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza nombrando `REQ-31.1`, la capacidad `facturacion` y el siguiente número libre `REQ-31.2`
- **Y** no se escribe nada

#### Scenario: Colisión con la delta de un cambio activo
- **DADO** un ADDED `(REQ-31.2)` y un cambio SDD activo cuya delta ya añade `REQ-31.2`
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza nombrando `REQ-31.2`, el cambio activo y el siguiente número libre

#### Scenario: Los cambios archivados no cuentan
- **DADO** un `REQ-31.3` presente solo en `openspec/changes/archive/<cambio>/specs/`
- **CUANDO** se integra un ADDED `(REQ-31.3)`
- **ENTONCES** no hay colisión por ese motivo

#### Scenario: Duplicado dentro del documento
- **DADO** dos subapartados con un ADDED `(REQ-31.4)` cada uno
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza nombrando `REQ-31.4` y ambas capacidades

#### Scenario: ADDED sin identificador
- **DADO** un ADDED cuyo título no contiene un identificador válido (con el patrón por defecto, `(REQ-n.m)`) ni un provisional
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza nombrando el requisito y el patrón

#### Scenario: Una integración ya aplicada no colisiona consigo misma
- **DADO** un ADDED `(REQ-31.1)` ya integrado con el mismo cuerpo
- **CUANDO** se repite la integración
- **ENTONCES** no se reporta colisión y termina con código `0`

#### Scenario: Colisión con un patrón declarado
- **DADO** un proyecto con `sdd: {id_pattern: "RF-<DOM>-<NN>"}`, un `RF-USU-01` ya existente en `openspec/specs/usuarios/spec.md` y una delta ODD con un ADDED `(RF-USU-01)`
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza nombrando `RF-USU-01`, la capacidad `usuarios` y el siguiente número libre de la serie `USU`, `RF-USU-02`
- **Y** no se escribe nada

#### Scenario: Un ADDED con el formato por defecto se rechaza con un patrón declarado
- **DADO** el mismo proyecto y un ADDED `(REQ-31.1)`
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza nombrando el requisito y el patrón `RF-<DOM>-<NN>`
- **Y** no se escribe nada

#### Scenario: El siguiente libre no cruza series
- **DADO** el mismo proyecto con `RF-USU-05` en `usuarios` y `RF-FAC-02` en `facturacion`, y un ADDED `(RF-FAC-02)` colisionando
- **CUANDO** se ejecuta la integración
- **ENTONCES** el siguiente libre propuesto es `RF-FAC-03`

#### Scenario: Un ADDED con un provisional se acepta y se asigna
- **DADO** un ADDED con un provisional con la serie `31` fijada y una spec viva con `REQ-31.1`
- **CUANDO** se ejecuta la integración
- **ENTONCES** no se rechaza por falta de identificador
- **Y** el requisito se integra con `REQ-31.2`

#### Scenario: Un provisional no colisiona con otro cambio activo
- **DADO** un ADDED con el provisional `«prov-a»` y un cambio SDD activo cuya delta declara también un provisional `«prov-a»`
- **CUANDO** se ejecuta la integración
- **ENTONCES** no se rechaza por esa coincidencia

#### Scenario: Un MODIFIED con un identificador inexistente se rechaza con un error tipado
- **DADO** un MODIFIED `Cobro con IVA (REQ-31.99)` y una spec viva sin `REQ-31.99`
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza con el error tipado de identificador inexistente, que nombra la capacidad, la sección MODIFIED y `REQ-31.99`
- **Y** no se escribe nada

#### Scenario: Un provisional fuera de ADDED se rechaza
- **DADO** un REMOVED cuyo nombre lleva `«prov-a»`
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza con un error tipado que nombra la sección REMOVED y `«prov-a»`
- **Y** no se escribe nada

#### Scenario: Un provisional citado sin ADDED que lo declare se rechaza
- **DADO** un documento ODD cuya línea `Cubre:` cita `«prov-b»` sin ningún ADDED que lo declare
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza con un error tipado que nombra `«prov-b»`
- **Y** no se escribe nada

#### Scenario: Un provisional duplicado en el documento se rechaza
- **DADO** dos ADDED del documento con el mismo `«prov-a»`
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza con un error tipado de provisional duplicado que nombra `«prov-a»` y los dos requisitos
