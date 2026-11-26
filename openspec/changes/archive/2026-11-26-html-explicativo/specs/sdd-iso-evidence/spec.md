# Delta para Declaración ISO y Bloque `sdd:` (sdd-iso-evidence)

> **Cambio:** `html-explicativo` (C-06-b). **Idioma:** castellano de España; identificadores, códigos y tokens en inglés.
> Solo cambia RF-ISO-01: amplía la lista cerrada de claves del bloque `sdd:` con `explainer_html`. El resto de requisitos de la capacidad no cambia.

## MODIFIED Requirements

### Requirement: Declaración ISO Opcional por Proyecto y Comportamiento sin Ella (RF-ISO-01)

`axiom.yaml` PUEDE declarar el modo ISO del proyecto en un bloque nuevo de primer nivel `sdd:` con tres claves: `iso` (booleano: `true` o `false`), la del patrón de identificador de requisito (RF-RID-01 de `requirement-identifiers`; recomendada: `id_pattern`) y `explainer_html` (cadena: `ask`, `always` o `never`; por defecto `ask`; su efecto lo fija la capacidad `sdd-lane-kickoff`), distinto de `workflow:`, cuyo bloque, claves y valores (`workflow.lane` admite `odd`, `sdd` y `ask`, y toma `sdd` por defecto) NO se modifican. El bloque `sdd:` se decodifica de forma estricta por lista cerrada de claves, con el mismo patrón que `workflow:` (RF-TOP-11 de `workspace-topology`): una clave desconocida dentro de `sdd:`, un valor no admitido de `iso`, un valor no admitido de `explainer_html`, un patrón de identificador inválido (RF-RID-01 de `requirement-identifiers`) y un `sdd:` que no sea un mapa (incluido el nulo) DEBEN reportarse como error accionable que nombre la clave o el valor y los admitidos, tanto al cargar la configuración como en `axiom workspace validate`, con mensajes en español y código de salida `1`. La opción es admisible en toda topología. **Validación cruzada con `workflow:`:** un `axiom.yaml` que declara a la vez `sdd.iso: true` y `workflow.lane: odd` es una declaración contradictoria, porque un proyecto ISO tramita todo el trabajo por el carril SDD (RF-RUT-19 de `organic-agent-trigger-rules`), y DEBE reportarse como error accionable, tanto al cargar la configuración como en `axiom workspace validate`, con mensaje en español y código de salida `1`. El mensaje DEBE nombrar las dos claves (`sdd.iso` y `workflow.lane`) y la corrección: retirar `workflow.lane: odd` (o fijarlo en `sdd`) o retirar `iso` (o ponerlo en `false`). Es una regla nueva entre dos bloques ya válidos por separado: NO cambia la decodificación de `workflow:` ni sus valores admitidos (RF-TOP-11 de `workspace-topology` queda como está, sin MODIFIED), y solo rechaza la combinación `iso: true` con `lane: odd`. Son combinaciones válidas: `iso: true` con `workflow.lane: sdd`, `iso: true` con `workflow.lane: ask` (la doctrina de RF-RUT-19 no formula entonces la pregunta de carril), `iso: true` sin `workflow.lane` o sin bloque `workflow:` (el carril por defecto es `sdd`), e `iso: false` o sin `sdd:` con cualquier `workflow.lane`, incluido `odd`. La declaración es por proyecto, nunca por cambio: nadie elige ISO para un cambio concreto, para no mezclar dos reglas de evidencia en un mismo proyecto auditado. `axiom sdd kickoff seal` copia el valor del proyecto en el `kickoff.yaml` del cambio (RF-KOF-12) y, desde entonces, el cambio es ISO o no ISO según esa instantánea, no según `axiom.yaml` en vivo; el patrón de identificador NO se copia a la instantánea: se lee siempre en vivo de `axiom.yaml` (RF-RID-01 de `requirement-identifiers`). Con el cambio sellado como ISO se activan, a la vez, las reglas de RF-ISO-03 a RF-ISO-09 y de RF-ISO-10, la regla de escenarios de RF-ARC-09 y el bloqueo del archivado de RF-ARC-10. En un cambio no ISO (sellado sin `iso`, o sin instantánea), ninguna de esas reglas DEBE bloquear: un cambio no ISO DEBE recorrer `continuous` y `checkpointed` con las mismas compuertas y paradas que antes de este cambio, y DEBE archivar con un `verify-report.md` libre o ausente. Ser un cambio ISO NO DEBE modificar el ritmo sellado, las compuertas `spec`, `design`, `tasks`, `role-apply:<rol>` e `integration`, las paradas de tres salidas ni el modo de archivado. Una declaración inválida, incluida la contradictoria de la validación cruzada, DEBE fallar cerrada, también al sellar: nunca se degrada a «cambio no ISO». Un binario anterior a este cambio ignora el bloque `sdd:` como cualquier clave de primer nivel desconocida y trata el cambio como no ISO: la declaración solo tiene efecto con el binario que la implementa. Un `axiom.yaml` sin el bloque `sdd:` se valida exactamente como antes de este cambio.
(Previously: el bloque `sdd:` admitía dos claves, `iso` y la del patrón de identificador; ahora admite también `explainer_html`, con el mismo decodificador estricto y sin cambiar nada de `iso` ni del patrón.)

#### Scenario: Bloque `sdd:` válido con `iso`

- **DADO** un `axiom.yaml` con `sdd:` que declara `iso: true`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** no se reporta ningún error por el bloque `sdd:`

#### Scenario: Un cambio no ISO recorre `continuous` sin cambios

- **DADO** un cambio no ISO sellado `continuous`
- **CUANDO** el cambio recorre `spec`, `design`, `tasks`, `apply`, `verify` y `archive` con un `verify-report.md` libre
- **ENTONCES** `axiom sdd status` no proyecta ningún motivo de bloqueo de RF-ISO-03 a RF-ARC-10 ni de RF-ISO-10
- **Y** las fases se encadenan sin ninguna parada, como antes de este cambio

#### Scenario: Un cambio no ISO recorre `checkpointed` con las mismas compuertas

- **DADO** un cambio no ISO sellado `checkpointed`
- **CUANDO** el cambio avanza por sus fases
- **ENTONCES** se presentan exactamente las compuertas y paradas de tres salidas que existían antes de este cambio
- **Y** el archivado procede con un `verify-report.md` libre o ausente

#### Scenario: La declaración ISO no cambia el ritmo, las compuertas ni el modo de archivado

- **DADO** un cambio ISO sellado `continuous` y otro sellado `checkpointed`
- **CUANDO** concluye cada fase
- **ENTONCES** el orquestador sigue encadenando las fases del primero sin ninguna parada nueva
- **Y** el segundo presenta las mismas compuertas que uno no ISO
- **Y** ambos archivan por la rama `<prefijo>/<cambio>-archive`

#### Scenario: Clave desconocida dentro de `sdd:`

- **DADO** un `axiom.yaml` con `sdd:` que contiene la clave `archive` o `design_triggers`, o cualquier otra fuera de la lista cerrada
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta en español la clave desconocida y las admitidas
- **Y** el código de salida es `1`

#### Scenario: Valor de `iso` no admitido

- **DADO** un `axiom.yaml` con `sdd: {iso: quizás}`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta en español el valor y los admitidos
- **Y** el código de salida es `1`

#### Scenario: `sdd:` que no es un mapa

- **DADO** un `axiom.yaml` con `sdd:` sin contenido o con una lista
- **CUANDO** se carga la configuración
- **ENTONCES** se reporta que `sdd` debe ser un mapa con claves de la lista cerrada
- **Y** el código de salida de `axiom workspace validate` es `1`

#### Scenario: Una declaración inválida falla cerrada, también al sellar

- **DADO** un `axiom.yaml` con el bloque `sdd:` inválido
- **CUANDO** se ejecuta `axiom sdd kickoff seal` o `axiom sdd status` sobre un cambio de ese proyecto
- **ENTONCES** se reporta la declaración inválida, nombrando el valor y la continuación
- **Y** el cambio NO se trata como no ISO
- **Y** el sellado se rechaza sin escribir `kickoff.yaml`

#### Scenario: `iso: true` con `workflow.lane: odd` es una declaración contradictoria

- **DADO** un `axiom.yaml` con `sdd: {iso: true}` y `workflow: {lane: odd}`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta en español un error que nombra `sdd.iso` y `workflow.lane`
- **Y** indica la corrección: retirar `workflow.lane: odd` o retirar `iso`
- **Y** el código de salida es `1`

#### Scenario: La contradicción se reporta también al cargar la configuración

- **DADO** el mismo `axiom.yaml` con `sdd: {iso: true}` y `workflow: {lane: odd}`
- **CUANDO** se carga la configuración (por ejemplo, con `axiom sdd status`)
- **ENTONCES** la carga reporta el mismo error accionable, con las dos claves y la corrección
- **Y** no asume «no ISO» ni carril `odd`

#### Scenario: La declaración contradictoria falla cerrada al sellar

- **DADO** un `axiom.yaml` con `sdd: {iso: true}` y `workflow: {lane: odd}`
- **CUANDO** se ejecuta `axiom sdd kickoff seal` sobre un cambio de ese proyecto
- **ENTONCES** el sellado se rechaza con el error de la contradicción
- **Y** NO se escribe `kickoff.yaml`

#### Scenario: `iso: true` con `workflow.lane: sdd` o sin carril declarado es válido

- **DADO** un `axiom.yaml` con `sdd: {iso: true}` y `workflow: {lane: sdd}`, y otro con `sdd: {iso: true}` sin `workflow.lane` o sin bloque `workflow:`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** no se reporta ningún error por la combinación en ninguno de los dos
- **Y** el carril efectivo es `sdd`

#### Scenario: `iso: true` con `workflow.lane: ask` es válido

- **DADO** un `axiom.yaml` con `sdd: {iso: true}` y `workflow: {lane: ask}`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** no se reporta ningún error por la combinación

#### Scenario: `iso: false` con `workflow.lane: odd` es válido

- **DADO** un `axiom.yaml` con `sdd: {iso: false}` y `workflow: {lane: odd}`, y otro con `workflow: {lane: odd}` sin bloque `sdd:`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** no se reporta ningún error por la combinación en ninguno de los dos

#### Scenario: `workflow:` no cambia

- **DADO** un `axiom.yaml` con `workflow: {lane: odd}` y `sdd: {iso: false}`
- **CUANDO** se carga la configuración
- **ENTONCES** `workflow` se resuelve exactamente igual que sin el bloque `sdd:`
- **Y** el bloque `workflow:` y sus valores admitidos (`odd`, `sdd`, `ask`) no cambian por la validación cruzada

#### Scenario: Un binario anterior ignora el bloque `sdd:`

- **DADO** un `axiom.yaml` con el bloque `sdd:` válido
- **CUANDO** lo carga un binario anterior a este cambio
- **ENTONCES** la carga tiene éxito sin interpretar el bloque
- **Y** el cambio se trata como no ISO

#### Scenario: Sin el bloque `sdd:` la validación no cambia

- **DADO** un `axiom.yaml` sin `sdd:`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** el resultado es idéntico al anterior a este cambio

#### Scenario: La clave del patrón de identificador es admitida en `sdd:`

- **DADO** un `axiom.yaml` con `sdd: {iso: true, id_pattern: "RF-<DOM>-<NN>"}`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** no se reporta ningún error por el bloque `sdd:`

#### Scenario: Un patrón de identificador inválido es un error del bloque `sdd:`

- **DADO** un `axiom.yaml` con `sdd: {iso: true, id_pattern: "RF-"}`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta en español la clave `sdd.id_pattern`, el valor y la regla incumplida (RF-RID-01 de `requirement-identifiers`)
- **Y** el código de salida es `1`

#### Scenario: La clave `explainer_html` es admitida en `sdd:` con sus tres valores

- **DADO** tres `axiom.yaml`, cada uno con `sdd: {explainer_html: ask}`, `sdd: {explainer_html: always}` y `sdd: {explainer_html: never}`
- **CUANDO** se ejecuta `axiom workspace validate` sobre cada uno
- **ENTONCES** no se reporta ningún error por el bloque `sdd:` en ninguno de los tres

#### Scenario: Un valor no admitido de `explainer_html` es un error del bloque `sdd:`

- **DADO** un `axiom.yaml` con `sdd: {explainer_html: quizás}`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta en español la clave `sdd.explainer_html`, el valor y los admitidos (`ask`, `always`, `never`)
- **Y** el código de salida es `1`

#### Scenario: `explainer_html` es independiente de `iso` y de `workflow.lane`

- **DADO** un `axiom.yaml` con `sdd: {iso: true, explainer_html: never}` y `workflow: {lane: sdd}`, y otro con `sdd: {explainer_html: always}` y `workflow: {lane: odd}`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** no se reporta ningún error por la combinación en ninguno de los dos

#### Scenario: `explainer_html` fuera del bloque `sdd:` no es una declaración

- **DADO** un `axiom.yaml` con `explainer_html: always` como clave de primer nivel
- **CUANDO** se carga la configuración
- **ENTONCES** la clave no tiene efecto sobre ningún cambio, como cualquier clave de primer nivel no registrada
- **Y** el valor vigente del proyecto sigue siendo `ask`
