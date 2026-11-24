# Delta para sdd-iso-evidence

> Parte del estado vigente de `openspec/specs/sdd-iso-evidence/spec.md`. Los requisitos REQ-29.1, REQ-29.3, REQ-29.4 y REQ-29.9 dejan de fijar la forma literal `REQ-<n>.<m>` y pasan a pedir un identificador válido según el patrón declarado del proyecto (REQ-44.1 a REQ-44.5 de `requirement-identifiers`). Con el patrón por defecto, que admite exactamente los `REQ-<n>.<m>` vigentes, ninguna regla ni mensaje cambia.
>
> REQ-29.3 cambia de título: se expresa con RENAMED seguido de MODIFIED bajo el título nuevo, para que el archivado funcione con el compositor que empareja por título exacto y con el que empareja por identificador (REQ-30.22 de `sdd-archive-lifecycle`).
>
> Las tablas «Términos» y «Requerimientos de un vistazo» de esta capacidad (que describen el identificador como `REQ-<n>.<m>`) no son bloques de requisito y no se editan con una delta; describen el patrón por defecto.

## MODIFIED Requirements

### Requirement: Declaración ISO Opcional por Proyecto y Comportamiento sin Ella (REQ-29.1)

`axiom.yaml` PUEDE declarar el modo ISO del proyecto en un bloque nuevo de primer nivel `sdd:` con dos claves: `iso` (booleano: `true` o `false`) y la del patrón de identificador de requisito (REQ-44.1 de `requirement-identifiers`; recomendada: `id_pattern`), distinto de `workflow:`, cuyo bloque, claves y valores (`workflow.lane` admite `odd`, `sdd` y `ask`, y toma `sdd` por defecto) NO se modifican. El bloque `sdd:` se decodifica de forma estricta por lista cerrada de claves, con el mismo patrón que `workflow:` (REQ-27.1 de `workspace-topology`): una clave desconocida dentro de `sdd:`, un valor no admitido de `iso`, un patrón de identificador inválido (REQ-44.1 de `requirement-identifiers`) y un `sdd:` que no sea un mapa (incluido el nulo) DEBEN reportarse como error accionable que nombre la clave o el valor y los admitidos, tanto al cargar la configuración como en `axiom workspace validate`, con mensajes en español y código de salida `1`. La opción es admisible en toda topología. **Validación cruzada con `workflow:`:** un `axiom.yaml` que declara a la vez `sdd.iso: true` y `workflow.lane: odd` es una declaración contradictoria, porque un proyecto ISO tramita todo el trabajo por el carril SDD (REQ-29.18 de `organic-agent-trigger-rules`), y DEBE reportarse como error accionable, tanto al cargar la configuración como en `axiom workspace validate`, con mensaje en español y código de salida `1`. El mensaje DEBE nombrar las dos claves (`sdd.iso` y `workflow.lane`) y la corrección: retirar `workflow.lane: odd` (o fijarlo en `sdd`) o retirar `iso` (o ponerlo en `false`). Es una regla nueva entre dos bloques ya válidos por separado: NO cambia la decodificación de `workflow:` ni sus valores admitidos (REQ-27.1 de `workspace-topology` queda como está, sin MODIFIED), y solo rechaza la combinación `iso: true` con `lane: odd`. Son combinaciones válidas: `iso: true` con `workflow.lane: sdd`, `iso: true` con `workflow.lane: ask` (la doctrina de REQ-29.18 no formula entonces la pregunta de carril), `iso: true` sin `workflow.lane` o sin bloque `workflow:` (el carril por defecto es `sdd`), e `iso: false` o sin `sdd:` con cualquier `workflow.lane`, incluido `odd`. La declaración es por proyecto, nunca por cambio: nadie elige ISO para un cambio concreto, para no mezclar dos reglas de evidencia en un mismo proyecto auditado. `axiom sdd kickoff seal` copia el valor del proyecto en el `kickoff.yaml` del cambio (REQ-29.13) y, desde entonces, el cambio es ISO o no ISO según esa instantánea, no según `axiom.yaml` en vivo; el patrón de identificador NO se copia a la instantánea: se lee siempre en vivo de `axiom.yaml` (REQ-44.1 de `requirement-identifiers`). Con el cambio sellado como ISO se activan, a la vez, las reglas de REQ-29.3 a REQ-29.9 y de REQ-29.19, la regla de escenarios de REQ-29.10 y el bloqueo del archivado de REQ-29.11. En un cambio no ISO (sellado sin `iso`, o sin instantánea), ninguna de esas reglas DEBE bloquear: un cambio no ISO DEBE recorrer `continuous` y `checkpointed` con las mismas compuertas y paradas que antes de este cambio, y DEBE archivar con un `verify-report.md` libre o ausente. Ser un cambio ISO NO DEBE modificar el ritmo sellado, las compuertas `spec`, `design`, `tasks`, `role-apply:<rol>` e `integration`, las paradas de tres salidas ni el modo de archivado. Una declaración inválida, incluida la contradictoria de la validación cruzada, DEBE fallar cerrada, también al sellar: nunca se degrada a «cambio no ISO». Un binario anterior a este cambio ignora el bloque `sdd:` como cualquier clave de primer nivel desconocida y trata el cambio como no ISO: la declaración solo tiene efecto con el binario que la implementa. Un `axiom.yaml` sin el bloque `sdd:` se valida exactamente como antes de este cambio.
(Previously: el bloque `sdd:` tenía una única clave, `iso`; ahora admite también la clave del patrón de identificador, con el mismo decodificador estricto.)

#### Scenario: Bloque `sdd:` válido con `iso`

- **DADO** un `axiom.yaml` con `sdd:` que declara `iso: true`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** no se reporta ningún error por el bloque `sdd:`

#### Scenario: Un cambio no ISO recorre `continuous` sin cambios

- **DADO** un cambio no ISO sellado `continuous`
- **CUANDO** el cambio recorre `spec`, `design`, `tasks`, `apply`, `verify` y `archive` con un `verify-report.md` libre
- **ENTONCES** `axiom sdd status` no proyecta ningún motivo de bloqueo de REQ-29.3 a REQ-29.11 ni de REQ-29.19
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
- **ENTONCES** se reporta en español la clave `sdd.id_pattern`, el valor y la regla incumplida (REQ-44.1 de `requirement-identifiers`)
- **Y** el código de salida es `1`

---

### Requirement: Identificador Válido según el Patrón Declarado en el Nombre de los Requisitos de la Delta (REQ-29.3)

En un cambio ISO, el nombre de todo requisito de la delta (ADDED, MODIFIED, REMOVED o RENAMED) DEBE contener un identificador válido según el patrón declarado del proyecto (REQ-44.1 de `requirement-identifiers`; sin declaración, el valor por defecto de REQ-44.2, que admite exactamente `REQ-<n>.<m>` con `<n>` y `<m>` enteros decimales sin signo). Es el identificador que citan `Cubre:` (REQ-29.4), la consulta de un requisito existente (REQ-29.14 de `spec-baseline-integration`) y la matriz de verify (REQ-29.6). La baseline es `openspec/specs/`, el único formato de spec viva: no existe ninguna otra identidad que declarar, y el identificador es solo el que figura en el nombre del requisito. El análisis del nombre (qué es un identificador, un candidato mal formado o una ausencia) lo hace el puerto único (REQ-44.4 y REQ-44.5 de `requirement-identifiers`). La comprobación DEBE hacerse en el estado del cambio, una vez escrita la delta y antes de habilitar la fase `design`, y DEBE aplicar estas reglas:
(Previously: el nombre debía contener un identificador con la forma literal `REQ-<n>.<m>`; ahora basta un identificador válido según el patrón declarado del proyecto, y el motivo nombra ese patrón.)

| Caso | Regla |
|---|---|
| Nombre sin identificador, o con una forma que no es un identificador válido según el patrón | Bloquea. |
| Nombre con más de un identificador distinto | Bloquea: es ambiguo. |
| ADDED cuyo identificador ya existe en la baseline, o que repite el de otro ADDED de la delta | Bloquea (supuesto visible: sin identificadores únicos, `Cubre:` y la consulta del bug no resuelven sin ambigüedad). **Excepción (archivado interrumpido):** NO bloquea un ADDED cuyo identificador figura en la baseline del mismo dominio en un requisito con el mismo nombre y el mismo bloque, byte a byte (encabezado `### Requirement:` y cuerpo), que el bloque ADDED de la delta. La comparación se hace tras la única normalización que aplica el compositor (`axiom sdd archive-compose` solo asegura que el bloque termine en un salto de línea); no se normaliza nada más (espacios, finales de línea ni orden). Es el propio requisito, ya compuesto por un archivado que se interrumpió antes de mover la carpeta del cambio. Cualquier diferencia de nombre, de bloque o de dominio sigue bloqueando. |
| MODIFIED o REMOVED | Su nombre es el del requisito existente, de modo que el identificador es el que ya contiene. Si el existente no lo contiene, bloquea. |
| RENAMED | El nombre nuevo contiene identificador. Si el nombre antiguo también lo contiene, ambos son el mismo (supuesto visible: renombrar no cambia la identidad). Un RENAMED puede añadir el identificador a un requisito que no lo tenía. |

Un bloqueo DEBE ser un motivo tipado accionable que nombre el dominio, la sección, el requisito, el patrón efectivo en su texto legible y la continuación (añadir o corregir el identificador en el nombre según el patrón; si el requisito ya existe sin él, añadírselo con un RENAMED de la delta). Con el patrón por defecto, el patrón se nombra como `REQ-<n>.<m>` y el texto del motivo y de la continuación es idéntico al de antes de este cambio. Solo se comprueban los requisitos de la delta, nunca el resto de la baseline. Sin la declaración ISO, esta regla NO DEBE bloquear. Si el patrón es inaccesible (bloque `sdd:` inválido o configuración ilegible), un cambio ISO NO DEBE evaluarse con un patrón no declarado: la etapa `specs` lleva el hallazgo `iso_declaration_invalid` en lugar de estas reglas, sin código de motivo nuevo (REQ-29.16 de `axiom-sdd-cli-integration` y REQ-44.9 de `requirement-identifiers`).

#### Scenario: Un ADDED sin identificador bloquea antes de design

- **DADO** un cambio ISO y una delta con un requisito ADDED llamado `Cobro con IVA` (sin `REQ-<n>.<m>`)
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de identificador
- **Y** el motivo nombra el dominio, la sección ADDED, el requisito y la continuación
- **Y** la fase `design` no se proyecta como siguiente

#### Scenario: Un nombre con identificador válido no bloquea

- **DADO** un cambio ISO y una delta con un ADDED `Cobro con IVA (REQ-30.1)` y un MODIFIED `Subcomando axiom sdd status (REQ-13.1)` que existe con ese nombre en la baseline
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta el motivo tipado de identificador

#### Scenario: Un identificador mal formado no cuenta

- **DADO** un cambio ISO y un ADDED llamado `Cobro con IVA (REQ-30)` o `Cobro con IVA (REQ-x.1)`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de identificador

#### Scenario: Un nombre con dos identificadores distintos bloquea

- **DADO** un cambio ISO y un ADDED llamado `Cobro y devolución (REQ-30.1, REQ-30.2)`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de identificador, que indica que el nombre es ambiguo

#### Scenario: Un ADDED con un identificador que ya existe bloquea

- **DADO** un cambio ISO y un ADDED llamado `Otro requisito (REQ-13.1)`, identificador que ya existe en la baseline
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de identificador, que nombra `REQ-13.1`

#### Scenario: Archivado interrumpido: el ADDED ya compuesto no bloquea y el archivado se completa

- **DADO** un cambio ISO con un ADDED `Cobro con IVA (REQ-30.1)` en el dominio `pagos`, cuyo archivado se interrumpió después de componerlo en `openspec/specs/pagos/spec.md` y antes de mover la carpeta del cambio, de modo que la baseline contiene un requisito con el mismo nombre y el mismo bloque, byte a byte, que el ADDED
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta el motivo tipado de identificador
- **Y** la ruta no vuelve a `spec`
- **Y** el archivado se reanuda y se completa sin modificar la delta

#### Scenario: El mismo identificador con un bloque distinto sigue bloqueando

- **DADO** un cambio ISO con un ADDED `Cobro con IVA (REQ-30.1)` en el dominio `pagos` y una baseline cuyo `openspec/specs/pagos/spec.md` ya contiene un requisito con el mismo nombre e identificador pero con un bloque que difiere en cualquier byte del ADDED (por ejemplo, un escenario más, otro texto o un espacio distinto)
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de identificador, que nombra `REQ-30.1`

#### Scenario: Dos ADDED con el mismo identificador bloquean

- **DADO** un cambio ISO con dos ADDED, en el mismo dominio o en dominios distintos, ambos con `REQ-30.1`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de identificador, que nombra `REQ-30.1` y los dos requisitos

#### Scenario: Un RENAMED que conserva el identificador no bloquea

- **DADO** un cambio ISO y un RENAMED de `Cobro (REQ-30.1)` a `Cobro con IVA (REQ-30.1)`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta el motivo tipado de identificador

#### Scenario: Un RENAMED que cambia el identificador bloquea

- **DADO** un cambio ISO y un RENAMED de `Cobro (REQ-30.1)` a `Cobro con IVA (REQ-30.2)`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de identificador, que nombra los dos identificadores

#### Scenario: Un RENAMED puede añadir el identificador a un requisito que no lo tenía

- **DADO** un cambio ISO y un RENAMED de `Cobro` (sin identificador, existente en la baseline) a `Cobro (REQ-30.3)`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta el motivo tipado de identificador

#### Scenario: Un MODIFIED de un requisito existente sin identificador bloquea con la continuación del RENAMED

- **DADO** un cambio ISO y un MODIFIED de un requisito existente cuyo nombre no contiene `REQ-<n>.<m>`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de identificador
- **Y** la continuación indica añadir el identificador al nombre con un RENAMED de la delta

#### Scenario: Los requisitos que la delta no toca no se comprueban

- **DADO** un cambio ISO y una baseline con un requisito existente sin identificador que la delta no menciona
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta el motivo tipado de identificador

#### Scenario: En un cambio no ISO el identificador no se exige

- **DADO** un cambio no ISO y un ADDED llamado `Cobro con IVA` (sin identificador)
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta ningún motivo de identificador

#### Scenario: Con un patrón declarado, un nombre con un identificador válido no bloquea

- **DADO** un cambio ISO de un proyecto con `sdd: {iso: true, id_pattern: "RF-<DOM>-<NN>"}` y una delta con un ADDED `Alta de usuario (RF-USU-01)`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta el motivo tipado de identificador

#### Scenario: Con un patrón declarado, el formato por defecto no es un identificador

- **DADO** el mismo proyecto y un ADDED llamado `Alta de usuario (REQ-30.1)`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de identificador
- **Y** el motivo nombra el patrón `RF-<DOM>-<NN>`

#### Scenario: Con un patrón declarado, un identificador mal formado bloquea

- **DADO** el mismo proyecto y un ADDED llamado `Alta de usuario (RF-USU-1)`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de identificador, que nombra `RF-USU-1` como candidato mal formado

#### Scenario: Con un patrón declarado, dos identificadores distintos bloquean

- **DADO** el mismo proyecto y un ADDED llamado `Alta y baja (RF-USU-01, RF-USU-02)`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de identificador, que indica que el nombre es ambiguo

#### Scenario: Con un patrón declarado, un ADDED con un identificador existente bloquea

- **DADO** el mismo proyecto, una baseline con `RF-USU-01` y un ADDED `Otra alta (RF-USU-01)`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de identificador, que nombra `RF-USU-01`

#### Scenario: Con el patrón inaccesible no se evalúa con un patrón no declarado

- **DADO** un cambio ISO sellado con un ADDED `Alta de usuario (RF-USU-01)` y un `axiom.yaml` cuyo `sdd.id_pattern` es inválido
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** la etapa `specs` proyecta el hallazgo `iso_declaration_invalid` en lugar de las reglas de identificador
- **Y** la etapa `tasks` no se evalúa

---

### Requirement: Cobertura `Cubre:` de los Requisitos de la Delta (REQ-29.4)

En un cambio ISO, cada tarea de `tasks.md` o de `tasks.<rol>.md` DEBE poder declarar los requisitos que cubre con una línea `Cubre: REQ-x[, REQ-y ...]`. El identificador citado es el identificador válido, según el patrón declarado del proyecto (REQ-44.1 de `requirement-identifiers`), del nombre del requisito (REQ-29.3). El motor DEBE analizar esas líneas en todos los ficheros de tareas del cambio y validar la cobertura conjunta, en cuanto existan los ficheros de tareas: cada requisito de la delta (identificado por el identificador válido de su nombre; en un RENAMED, el de su nombre nuevo) DEBE aparecer en al menos un `Cubre:` del conjunto de ficheros. Un requisito de la delta sin cobertura DEBE bloquear con un motivo tipado que lo nombre. Un identificador de `Cubre:` que no existe ni en la delta ni en la baseline DEBE bloquear con un motivo tipado que lo nombre; uno que existe en la baseline pero no en la delta se admite y no cuenta para la cobertura. Fuera de ISO, `Cubre:` es informativo y opcional y NUNCA DEBE bloquear.
(Previously: el identificador citado era la forma literal `REQ-<n>.<m>`; ahora es el identificador válido según el patrón declarado del proyecto, que con el patrón por defecto coincide con `REQ-<n>.<m>`.)

#### Scenario: Un requisito de la delta sin tarea que lo cubra bloquea

- **DADO** un cambio ISO y una delta con los requisitos `REQ-29.1` y `REQ-29.2`, y tareas con `Cubre: REQ-29.1` únicamente
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de cobertura, que nombra `REQ-29.2`

#### Scenario: La cobertura se calcula sobre la unión de los ficheros por rol

- **DADO** un cambio ISO y una delta con `REQ-29.1` y `REQ-29.2`, con `tasks.core.md` que contiene `Cubre: REQ-29.1` y `tasks.web.md` que contiene `Cubre: REQ-29.2`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta ningún motivo de cobertura

#### Scenario: Un identificador inexistente en `Cubre:` bloquea

- **DADO** un cambio ISO y una tarea con `Cubre: REQ-999.1`, identificador que no figura en la delta ni en la baseline
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con un motivo tipado que nombra `REQ-999.1`

#### Scenario: Una tarea puede citar varios requisitos

- **DADO** un cambio ISO y una tarea con `Cubre: REQ-29.1, REQ-29.2`
- **CUANDO** el motor analiza el fichero de tareas
- **ENTONCES** ambos requisitos se consideran cubiertos por esa tarea

#### Scenario: Un requisito existente de la baseline se admite y no cubre la delta

- **DADO** un cambio ISO con una delta con `REQ-29.1`, una tarea con `Cubre: REQ-29.1` y otra con `Cubre: REQ-13.1`, requisito que existe en la baseline y no figura en la delta
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta ningún motivo de cobertura

#### Scenario: Sin ficheros de tareas no hay motivo de cobertura

- **DADO** un cambio ISO con delta y sin `tasks.md` todavía
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta ningún motivo de cobertura
- **Y** la fase `tasks` sigue como pendiente

#### Scenario: En un cambio no ISO la cobertura no bloquea

- **DADO** un cambio no ISO y una delta cuyos requisitos no aparecen en ningún `Cubre:`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta ningún motivo de bloqueo de cobertura


#### Scenario: Con un patrón declarado, `Cubre:` cita el identificador del patrón

- **DADO** un cambio ISO de un proyecto con `sdd: {iso: true, id_pattern: "RF-<DOM>-<NN>"}`, una delta con `RF-USU-01` y `RF-USU-02` y tareas con `Cubre: RF-USU-01, RF-USU-02`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta ningún motivo de cobertura

#### Scenario: Con un patrón declarado, un `Cubre:` con el formato por defecto bloquea

- **DADO** el mismo proyecto y una tarea con `Cubre: REQ-30.1`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con un motivo tipado que nombra `REQ-30.1` y el patrón `RF-<DOM>-<NN>`

---

### Requirement: Trazabilidad por `git log --first-parent` Independiente de la Estrategia de Fusión y Compatible con la Rama `-archive` (REQ-29.9)

Para un cambio ISO archivado por la rama `<prefijo>/<cambio>-archive` (`monorepo-embedded`), Axiom DEBE poder derivar, con `git log --first-parent <rama de integración>` y sin ningún estado persistido nuevo, dos commits de la línea principal: (a) el **commit de integración del código**, el primero que incorpora la carpeta `openspec/changes/<cambio>/` con sus `tasks.md` y `verify-report.md`; y (b) el **commit de archivado**, el primero que incorpora `openspec/changes/archive/<fecha>-<cambio>/`. La derivación DEBE dar el mismo resultado con fusión por squash, por commit de merge y por rebase, tanto para el PR del cambio como para el PR documental del archivado. Cuando la compuerta `integration` registra evidencia `pr_merged` con un commit (`internal/kickoff`), el commit de código derivado DEBE contrastarse con ella: DEBE ser ese commit o un ancestro suyo en la línea principal, y una discrepancia se informa sin error de ejecución. NO DEBE introducir estado persistido nuevo ni depender de ninguna traza de integración de IDs: el identificador de un requisito es el identificador válido, según el patrón declarado del proyecto, de su nombre, elegido por quien redacta la delta, y no hay ID provisional ni definitivo (supuesto visible). La cadena requisito (su identificador) → `Cubre:` de las tareas archivadas → fila de la matriz del informe archivado → commit de integración del código → commit de archivado DEBE poder reconstruirse a partir de los artefactos archivados y del historial de la rama principal. Los trailers de los commits intermedios son informativos y NUNCA se validan. Si la carpeta archivada, o la carpeta del cambio, no existe todavía en la rama principal, la derivación DEBE informar de la ausencia del commit correspondiente sin error de ejecución. En `multirepo` la derivación no se especifica en este cambio (supuesto visible).
(Previously: el identificador de la cadena era la forma literal `REQ-<n>.<m>`; ahora es el identificador válido según el patrón declarado del proyecto.)

#### Scenario: Cambio fusionado por squash

- **DADO** un cambio ISO cuyo PR y cuyo PR documental de archivado se fusionaron por squash en la rama de integración
- **CUANDO** se derivan los dos commits
- **ENTONCES** se obtienen el commit de squash del código y el commit de squash del archivado
- **Y** la cadena requisito → `Cubre:` → fila → commit de código → commit de archivado se reconstruye

#### Scenario: Cambio fusionado por commit de merge

- **DADO** un cambio ISO cuyos PR se fusionaron con un commit de merge
- **CUANDO** se derivan los dos commits
- **ENTONCES** se obtienen los commits de merge de la línea `--first-parent` que incorporan cada carpeta

#### Scenario: Cambio fusionado por rebase

- **DADO** un cambio ISO cuyos PR se fusionaron por rebase, con sus commits atómicos en la línea principal
- **CUANDO** se derivan los dos commits
- **ENTONCES** se obtiene, en cada caso, el commit atómico que incorpora la carpeta correspondiente

#### Scenario: El commit de código se contrasta con la evidencia `pr_merged`

- **DADO** un cambio ISO con la compuerta `integration` aprobada con evidencia `pr_merged` cuyo commit es el último de la serie rebasada
- **CUANDO** se deriva el commit de integración del código
- **ENTONCES** el commit derivado es ese commit o un ancestro suyo en la línea principal
- **Y** no se informa discrepancia

#### Scenario: Una discrepancia con la evidencia se informa sin error

- **DADO** un cambio ISO cuya evidencia `pr_merged` registra un commit que no es descendiente del commit derivado
- **CUANDO** se deriva el commit de integración del código
- **ENTONCES** se informa la discrepancia nombrando ambos commits
- **Y** la ejecución no falla

#### Scenario: Sin compuerta `integration` la derivación no exige evidencia

- **DADO** un cambio ISO sellado `continuous`, sin compuerta `integration`
- **CUANDO** se deriva el commit de integración del código
- **ENTONCES** se obtiene el commit por la línea `--first-parent`
- **Y** no se informa ninguna ausencia de evidencia

#### Scenario: Los trailers intermedios no se validan

- **DADO** un cambio ISO archivado cuyos commits intermedios no llevan trailers o llevan trailers incorrectos
- **CUANDO** se derivan los dos commits
- **ENTONCES** la derivación no falla ni se rechaza por esos trailers

#### Scenario: Ruta archivada ausente de la rama principal

- **DADO** un cambio cuya carpeta archivada aún no existe en la rama principal
- **CUANDO** se derivan los dos commits
- **ENTONCES** se informa de que no hay commit de archivado todavía
- **Y** no se produce un error de ejecución

#### Scenario: La derivación no añade estado

- **DADO** un cambio ISO archivado
- **CUANDO** se derivan los dos commits
- **ENTONCES** no se escribe ningún fichero nuevo de estado en la carpeta del cambio ni en el repositorio


#### Scenario: Con un patrón declarado, la cadena de trazabilidad parte del identificador del patrón

- **DADO** un cambio ISO archivado de un proyecto con `sdd: {iso: true, id_pattern: "RF-<DOM>-<NN>"}`, con `Cubre: RF-USU-01` en sus tareas archivadas y una fila de la matriz para ese requisito
- **CUANDO** se reconstruye la cadena requisito → `Cubre:` → fila → commit de integración del código → commit de archivado
- **ENTONCES** la cadena se reconstruye partiendo de `RF-USU-01`

## RENAMED Requirements

### Requirement: Identificador `REQ-<n>.<m>` en el Nombre de los Requisitos de la Delta (REQ-29.3) → Identificador Válido según el Patrón Declarado en el Nombre de los Requisitos de la Delta (REQ-29.3)

(Reason: el requisito deja de fijar la forma literal `REQ-<n>.<m>` y pide un identificador válido según el patrón que declara el proyecto (REQ-44.1 de `requirement-identifiers`); el título lo refleja)
(Migration: las referencias por el identificador REQ-29.3 siguen siendo válidas; las pruebas y textos que citaban el título antiguo pasan a citar el nuevo, y con el patrón por defecto el comportamiento observable no cambia)
