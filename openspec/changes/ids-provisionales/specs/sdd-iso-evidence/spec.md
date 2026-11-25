# Delta para sdd-iso-evidence

En un cambio ISO, un ADDED puede llevar un identificador provisional como forma válida de su identificador (REQ-29.3), y `Cubre:` acepta los provisionales declarados por la propia delta y se reescribe al archivar (REQ-29.4).

Notación de los escenarios: `«prov-a»`, `«prov-b»` son marcadores neutros de un identificador provisional; NO son el formato real, que fija el diseño (véase REQ-47.1 de `requirement-identifiers`).

## MODIFIED Requirements

### Requirement: Identificador Válido según el Patrón Declarado en el Nombre de los Requisitos de la Delta (REQ-29.3)

En un cambio ISO, el nombre de todo requisito de la delta (ADDED, MODIFIED, REMOVED o RENAMED) DEBE contener un identificador válido según el patrón declarado del proyecto (REQ-44.1 de `requirement-identifiers`; sin declaración, el valor por defecto de REQ-44.2, que admite exactamente `REQ-<n>.<m>` con `<n>` y `<m>` enteros decimales sin signo). La única excepción es el ADDED, cuyo nombre PUEDE llevar en su lugar un identificador provisional (REQ-47.1 de `requirement-identifiers`), que la integración sustituye por el definitivo al archivar (REQ-47.8 de `sdd-archive-lifecycle`). Es el identificador que citan `Cubre:` (REQ-29.4), la consulta de un requisito existente (REQ-29.14 de `spec-baseline-integration`) y la matriz de verify (REQ-29.6). La baseline es `openspec/specs/`, el único formato de spec viva: no existe ninguna otra identidad que declarar, y el identificador es solo el que figura en el nombre del requisito. El análisis del nombre (qué es un identificador, un candidato mal formado o una ausencia) lo hace el puerto único (REQ-44.4 y REQ-44.5 de `requirement-identifiers`). La comprobación DEBE hacerse en el estado del cambio, una vez escrita la delta y antes de habilitar la fase `design`, y DEBE aplicar estas reglas:
(Previously: el nombre debía contener siempre un identificador válido según el patrón declarado del proyecto, también el ADDED; ahora el ADDED admite además un identificador provisional de la propia delta, y un provisional fuera de ADDED bloquea.)

| Caso | Regla |
|---|---|
| Nombre sin identificador, o con una forma que no es un identificador válido según el patrón ni un provisional de un ADDED | Bloquea. |
| Nombre con más de un identificador distinto (válidos o provisionales, en cualquier combinación) | Bloquea: es ambiguo. |
| ADDED con un provisional | No bloquea por falta de identificador definitivo. Bloquea si repite el provisional de otro ADDED de la delta, en el mismo dominio o en otro (error tipado de provisional duplicado, REQ-47.1). |
| MODIFIED, REMOVED o RENAMED con un provisional en cualquiera de sus nombres | Bloquea: un provisional solo es válido en un ADDED. |
| ADDED con identificador definitivo cuyo identificador ya existe en la baseline, o que repite el de otro ADDED de la delta | Bloquea (supuesto visible: sin identificadores únicos, `Cubre:` y la consulta del bug no resuelven sin ambigüedad). **Excepción (archivado interrumpido):** NO bloquea un ADDED cuyo identificador figura en la baseline del mismo dominio en un requisito con el mismo nombre y el mismo bloque, byte a byte (encabezado `### Requirement:` y cuerpo), que el bloque ADDED de la delta. La comparación se hace tras la única normalización que aplica el compositor (`axiom sdd archive-compose` solo asegura que el bloque termine en un salto de línea); no se normaliza nada más (espacios, finales de línea ni orden). Es el propio requisito, ya compuesto por un archivado que se interrumpió antes de mover la carpeta del cambio. Cualquier diferencia de nombre, de bloque o de dominio sigue bloqueando. |
| MODIFIED o REMOVED | Su nombre es el del requisito existente, de modo que el identificador es el que ya contiene. Si el existente no lo contiene, bloquea. |
| RENAMED | El nombre nuevo contiene identificador. Si el nombre antiguo también lo contiene, ambos son el mismo (supuesto visible: renombrar no cambia la identidad). Un RENAMED puede añadir el identificador a un requisito que no lo tenía. |

Un bloqueo DEBE ser un motivo tipado accionable que nombre el dominio, la sección, el requisito, el patrón efectivo en su texto legible y la continuación (añadir o corregir el identificador en el nombre según el patrón; si el requisito ya existe sin él, añadírselo con un RENAMED de la delta). Con el patrón por defecto, el patrón se nombra como `REQ-<n>.<m>` y el texto del motivo y de la continuación es idéntico al de antes de este cambio. Solo se comprueban los requisitos de la delta, nunca el resto de la baseline. Sin la declaración ISO, esta regla NO DEBE bloquear. Si el patrón es inaccesible (bloque `sdd:` inválido o configuración ilegible), un cambio ISO NO DEBE evaluarse con un patrón no declarado: la etapa `specs` lleva el hallazgo `iso_declaration_invalid` en lugar de estas reglas, sin código de motivo nuevo (REQ-29.16 de `axiom-sdd-cli-integration` y REQ-44.9 de `requirement-identifiers`).

#### Scenario: Un ADDED sin identificador bloquea antes de design

- **DADO** un cambio ISO y una delta con un requisito ADDED llamado `Cobro con IVA` (sin `REQ-<n>.<m>` ni provisional)
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

#### Scenario: Un ADDED con un provisional no bloquea

- **DADO** un cambio ISO y una delta con un ADDED llamado `Cobro con IVA («prov-a»)`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta el motivo tipado de identificador
- **Y** la fase `design` puede proyectarse como siguiente

#### Scenario: Con un patrón declarado, un ADDED con un provisional con el dominio fijado no bloquea

- **DADO** un cambio ISO de un proyecto con `sdd: {iso: true, id_pattern: "RF-<DOM>-<NN>"}` y un ADDED con un provisional con el dominio `USU` fijado
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta el motivo tipado de identificador

#### Scenario: Dos ADDED con el mismo provisional bloquean

- **DADO** un cambio ISO con dos ADDED, en el mismo dominio o en dominios distintos, ambos con `«prov-a»`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de identificador, que nombra `«prov-a»` y los dos requisitos

#### Scenario: Un provisional en un MODIFIED, REMOVED o RENAMED bloquea

- **DADO** un cambio ISO y un MODIFIED cuyo nombre lleva `«prov-a»`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de identificador, que nombra la sección MODIFIED y `«prov-a»`
- **Y** lo mismo ocurre con un provisional en un REMOVED o en cualquiera de los dos nombres de un RENAMED

#### Scenario: Un nombre con un provisional y un identificador válido bloquea por ambiguo

- **DADO** un cambio ISO y un ADDED llamado `Cobro con IVA («prov-a», REQ-30.1)`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de identificador, que indica que el nombre es ambiguo

---

### Requirement: Cobertura `Cubre:` de los Requisitos de la Delta (REQ-29.4)

En un cambio ISO, cada tarea de `tasks.md` o de `tasks.<rol>.md` DEBE poder declarar los requisitos que cubre con una línea `Cubre: REQ-x[, REQ-y ...]`. El identificador citado es el identificador válido, según el patrón declarado del proyecto (REQ-44.1 de `requirement-identifiers`), del nombre del requisito (REQ-29.3), o el provisional de un ADDED de la propia delta (REQ-47.1). El motor DEBE analizar esas líneas en todos los ficheros de tareas del cambio y validar la cobertura conjunta, en cuanto existan los ficheros de tareas: cada requisito de la delta (identificado por el identificador válido o el provisional de su nombre; en un RENAMED, el de su nombre nuevo) DEBE aparecer en al menos un `Cubre:` del conjunto de ficheros. Un requisito de la delta sin cobertura DEBE bloquear con un motivo tipado que lo nombre. Un identificador de `Cubre:` que no existe ni en la delta ni en la baseline DEBE bloquear con un motivo tipado que lo nombre, y un provisional citado que ningún ADDED de la delta declara DEBE bloquear igual (no se confunde con un identificador de la baseline); uno que existe en la baseline pero no en la delta se admite y no cuenta para la cobertura. Las líneas `Cubre:` con provisionales de la delta DEBEN reescribirse al identificador definitivo al archivar, conservando la posición del token en la línea (REQ-47.4 de `requirement-identifiers` y REQ-47.8 de `sdd-archive-lifecycle`). Fuera de ISO, `Cubre:` es informativo y opcional y NUNCA DEBE bloquear.
(Previously: el identificador citado era solo el identificador válido, según el patrón declarado del proyecto, del nombre del requisito, y un provisional se habría rechazado como identificador desconocido; ahora `Cubre:` admite los provisionales de los ADDED de la propia delta, cuenta como cobertura de ese ADDED y se reescribe al archivar.)

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

#### Scenario: `Cubre:` acepta el provisional de un ADDED de la delta y lo cuenta como cobertura

- **DADO** un cambio ISO con un ADDED `Cobro con IVA («prov-a»)` y una tarea con `Cubre: «prov-a»`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta ningún motivo de cobertura ni de identificador desconocido

#### Scenario: Un ADDED provisional sin ninguna tarea que lo cubra bloquea

- **DADO** un cambio ISO con un ADDED `Cobro con IVA («prov-a»)` y tareas sin ninguna línea `Cubre:` que cite `«prov-a»`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de cobertura, que nombra `«prov-a»`

#### Scenario: Un provisional citado que ningún ADDED declara bloquea

- **DADO** un cambio ISO y una tarea con `Cubre: «prov-b»`, sin ningún ADDED que declare `«prov-b»`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con un motivo tipado que nombra `«prov-b»`

#### Scenario: El `Cubre:` con provisional se reescribe al archivar

- **DADO** un cambio ISO con una tarea `Cubre: REQ-29.1, «prov-a»` y un ADDED `«prov-a»` al que el archivado asigna `REQ-31.2`
- **CUANDO** se ejecuta el archivado del cambio
- **ENTONCES** la línea de la tarea archivada queda `Cubre: REQ-29.1, REQ-31.2`
- **Y** el resto de `tasks.md` conserva sus bytes

---
