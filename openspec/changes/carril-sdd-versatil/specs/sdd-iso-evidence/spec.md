# Especificación: Evidencias ISO Opcionales del SDD (carril-sdd-versatil)

> **Idioma:** Español (castellano peninsular) · **Tipo:** capacidad nueva · **Serie:** REQ-29.x

## Propósito

Definir el **modo ISO**, una opción declarada por el proyecto en `axiom.yaml` y sellada en cada cambio al arrancarlo (REQ-29.13 de `sdd-lane-kickoff`), que extiende el SDD actual con la evidencia que pide un auditor de procesos (familia ISO/IEC 330xx, atributo PA 2.2 de gestión de productos de trabajo, con apoyo de ISO/IEC/IEEE 29148 para la calidad de los requisitos): requisitos con identificador estable, cobertura requisito → tarea, matriz de verificación con una fila de evidencia contrastable por escenario, archivado condicionado a esa matriz (REQ-29.11 de `sdd-archive-lifecycle`) y trazabilidad hasta los commits de integración y de archivado en la rama principal.

El modo ISO **no sustituye nada**: no cambia el ritmo (`continuous` o `checkpointed`), no retira compuertas ni paradas de tres salidas, no cambia el modo de archivado (la rama `<prefijo>/<cambio>-archive` sigue siendo el único) y no altera los cambios activos o archivados que no se sellaron como ISO. Un cambio no sellado como ISO se comporta como hasta ahora.

## Términos

| Término | Definición |
|---|---|
| Declaración ISO | Valor `iso` del bloque de primer nivel `sdd:` de `axiom.yaml` (REQ-29.1). |
| Cambio ISO | Cambio cuyo `kickoff.yaml` se selló con `iso` activo: copia del valor que el proyecto declaraba en `axiom.yaml` al sellar (REQ-29.13). Las reglas ISO de un cambio se leen de esa instantánea, no de `axiom.yaml` en vivo. Un cambio sin instantánea es un cambio no ISO. |
| Baseline implícita | La baseline OpenSpec por defecto, `openspec/specs/`, que rige cuando `axiom.yaml` no declara `baseline:` (REQ-24.13). |
| Delta | Conjunto de requisitos ADDED, MODIFIED, REMOVED y RENAMED de las specs del cambio. |
| Bug sin delta | Cambio declarado como bug en su `proposal.md` (observado, esperado y, opcionalmente, referencia externa) que restaura un requisito ya existente de la baseline y por eso no lleva delta (REQ-29.15 de `axiom-sdd-cli-integration`). Un bug cuyo requisito estaba mal especificado o faltaba NO es un bug sin delta: es un cambio normal, con delta. |
| `Cubre:` | Línea de una tarea que declara los requisitos que esa tarea cubre. |
| Escenario afectado | Cada escenario de un bloque ADDED o MODIFIED de la delta; en un bug sin delta, cada escenario del requisito existente que cita `Cubre:`. Se identifica por el par (requisito, nombre del escenario). |
| Matriz de verify | Sección de `verify-report.md` con una fila por escenario afectado. |
| Fila automática | Fila de tipo `unitario`, `integración` o `e2e`. |
| Fila manual | Fila de tipo `manual`. |
| Referencia de prueba | Identificación de la prueba que respalda una fila automática, con una de dos formas: `ruta/fichero:símbolo` (cualquier lenguaje) o `TestX` (atajo para proyectos en Go). |
| Árbol de trabajo | Árbol de ficheros del repositorio en el momento de validar (la rama del cambio al verificar; la rama de archivado al archivar). |
| Motivo tipado | Motivo de bloqueo de `axiom sdd status` con un token estable, el elemento afectado y la continuación. El token exacto de cada motivo lo fija el diseño. |

## Requerimientos de un vistazo

| ID | Requerimiento | Garantía principal |
|---|---|---|
| REQ-29.1 | Declaración ISO opcional por proyecto | Un cambio no ISO no bloquea ni cambia; declaración inválida falla cerrada |
| REQ-29.2 | Proyección de `sdd.iso` en el status | Solo aditiva |
| REQ-29.3 | Identidad `id` en los documentos afectados | Bloqueo accionable antes de design |
| REQ-29.4 | Cobertura `Cubre:` | Cada requisito de la delta, al menos una tarea |
| REQ-29.5 | Bug sin delta en ISO | El requisito citado existe y tiene escenarios |
| REQ-29.6 | Matriz de verify: forma y completitud | Una fila por escenario afectado, sin líneas `verdict:` |
| REQ-29.7 | Filas automáticas contrastables | La referencia resuelve a un símbolo del árbol; comando y código de salida registrados |
| REQ-29.8 | Filas manuales justificadas | Quién, cuándo y por qué no se automatiza |
| REQ-29.9 | Trazabilidad por `--first-parent` | Independiente de squash, merge y rebase; con la rama `-archive` |

El bloqueo del archivado por la matriz es el REQ-29.11 de `sdd-archive-lifecycle`, y la regla de al menos un escenario por requisito es el REQ-29.10 de esa misma capacidad.

## No cubre esta capacidad

- Reejecutar las pruebas para confirmar el resultado de una fila: que la prueba pase sigue siendo **declarado**; lo contrastable es que la prueba existe en el árbol y que el comando y su código de salida quedan registrados para repetirla.
- Que el símbolo de una referencia `ruta/fichero:símbolo` sea sintácticamente una prueba: en lenguajes distintos de Go la resolución es textual (REQ-29.7).
- Otras formas de referencia de prueba (por ejemplo, identificadores de casos de una herramienta e2e).
- Una aprobación humana de la matriz como parada nueva.
- La derivación de la traza en `multirepo` (REQ-29.9) y el archivado en el mismo PR.
- La publicación del ID definitivo de un requisito hacia herramientas externas.
- El modelo de referencia de procesos y el nivel de capacidad objetivo.

---

## Requirements

### Requirement: Declaración ISO Opcional por Proyecto y Comportamiento sin Ella (REQ-29.1)

`axiom.yaml` PUEDE declarar el modo ISO del proyecto en un bloque nuevo de primer nivel `sdd:` con una única clave, `iso` (booleano; el tipo definitivo lo fija el diseño), distinto de `workflow:` y de `baseline:`, que NO se modifican. El bloque `sdd:` se decodifica de forma estricta por lista cerrada de claves, con el mismo patrón que `workflow:` (REQ-27.1 de `workspace-topology`): una clave desconocida dentro de `sdd:`, un valor no admitido de `iso` y un `sdd:` que no sea un mapa (incluido el nulo) DEBEN reportarse como error accionable que nombre la clave o el valor y los admitidos, tanto al cargar la configuración como en `axiom workspace validate`, con mensajes en español y código de salida `1`. La opción es admisible en toda topología. La declaración es por proyecto, nunca por cambio: nadie elige ISO para un cambio concreto, para no mezclar dos reglas de evidencia en un mismo proyecto auditado. `axiom sdd kickoff seal` copia el valor del proyecto en el `kickoff.yaml` del cambio (REQ-29.13) y, desde entonces, el cambio es ISO o no ISO según esa instantánea, no según `axiom.yaml` en vivo. Con el cambio sellado como ISO se activan, a la vez, las reglas de REQ-29.3 a REQ-29.9, la regla de escenarios de REQ-29.10 y el bloqueo del archivado de REQ-29.11. En un cambio no ISO (sellado sin `iso`, o sin instantánea), ninguna de esas reglas DEBE bloquear: un cambio no ISO DEBE recorrer `continuous` y `checkpointed` con las mismas compuertas y paradas que antes de este cambio, y DEBE archivar con un `verify-report.md` libre o ausente. Ser un cambio ISO NO DEBE modificar el ritmo sellado, las compuertas `spec`, `design`, `tasks`, `role-apply:<rol>` e `integration`, las paradas de tres salidas ni el modo de archivado. Una declaración inválida DEBE fallar cerrada (REQ-24.18 de `spec-baseline-declaration`), también al sellar: nunca se degrada a «cambio no ISO». La validación de la declaración NO DEBE fallar por documentos de la baseline sin identidad `id`: esa comprobación pertenece al estado del cambio (REQ-29.3). Un binario anterior a este cambio ignora el bloque `sdd:` como cualquier clave de primer nivel desconocida (REQ-24.17 de `spec-baseline-declaration`) y trata el cambio como no ISO: la declaración solo tiene efecto con el binario que la implementa. Un `axiom.yaml` sin el bloque `sdd:` se valida exactamente como antes de este cambio.

#### Scenario: Bloque `sdd:` válido con `iso`

- **DADO** un `axiom.yaml` con `sdd:` que declara `iso: true`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** no se reporta ningún error por el bloque `sdd:`

#### Scenario: Un cambio no ISO recorre `continuous` sin cambios

- **DADO** un cambio no ISO sellado `continuous`
- **CUANDO** el cambio recorre `spec`, `design`, `tasks`, `apply`, `verify` y `archive` con un `verify-report.md` libre
- **ENTONCES** `axiom sdd status` no proyecta ningún motivo de bloqueo de REQ-29.3 a REQ-29.11
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

#### Scenario: `workflow:` y `baseline:` no cambian

- **DADO** un `axiom.yaml` con `workflow: {lane: odd}`, `baseline:` válida y `sdd:` válido
- **CUANDO** se carga la configuración
- **ENTONCES** `workflow` y `baseline` se resuelven exactamente igual que sin el bloque `sdd:`

#### Scenario: Documentos sin identidad `id` no invalidan la declaración

- **DADO** un `axiom.yaml` con `sdd: {iso: true}` y un documento de baseline con `identity: text`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** no se reporta error por esa identidad

#### Scenario: Un binario anterior ignora el bloque `sdd:`

- **DADO** un `axiom.yaml` con el bloque `sdd:` válido
- **CUANDO** lo carga un binario anterior a este cambio
- **ENTONCES** la carga tiene éxito sin interpretar el bloque
- **Y** el cambio se trata como no ISO

#### Scenario: Sin el bloque `sdd:` la validación no cambia

- **DADO** un `axiom.yaml` sin `sdd:`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** el resultado es idéntico al anterior a este cambio

---

### Requirement: Proyección Aditiva de `sdd.iso` en el Status v2 (REQ-29.2)

`axiom sdd status` DEBE proyectar de forma aditiva, sin que ninguna skill ni orquestador tenga que parsear `axiom.yaml`: (a) el valor vigente del proyecto (si declara ISO), que usa el orquestador antes de que exista un cambio (por ejemplo, para tramitar un bug, REQ-29.18), y (b) el valor efectivo del cambio consultado, leído de su instantánea en `kickoff.yaml` (REQ-29.13 de `sdd-lane-kickoff`): si el cambio es ISO. La proyección va junto a la de `workflow` (REQ-27.2 de `axiom-sdd-cli-integration`) y con o sin cambio seleccionado. Cuando el valor sellado del cambio difiere del vigente del proyecto, el status DEBE emitir la nota no bloqueante de REQ-29.13. Un cambio sin instantánea se proyecta como no ISO. Con un `sdd:` inválido, el status DEBE proyectar la declaración como inválida en lugar de asumir el valor por defecto. Los nombres de las claves proyectadas los fija el diseño. Ninguna clave existente del status v2 DEBE cambiar de nombre, tipo ni valor (REQ-24.20 de `spec-baseline-declaration`). Sin el bloque `sdd:`, las claves nuevas indican «no ISO», o se omiten, de modo que un consumidor anterior no observa diferencia. El contrato publicado (`sdd-status-contract.md`) DEBE documentar la proyección.

#### Scenario: El status proyecta el cambio ISO

- **DADO** un `axiom.yaml` con `sdd: {iso: true}` y un cambio sellado con `iso`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el status indica que el proyecto declara ISO y que el cambio es ISO

#### Scenario: Sin cambio seleccionado se proyecta el valor del proyecto

- **DADO** un `axiom.yaml` con `sdd: {iso: true}` y ningún cambio seleccionado
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `changeName` es `null`
- **Y** el status indica que el proyecto declara ISO

#### Scenario: El status proyecta el valor sellado aunque el proyecto haya cambiado

- **DADO** un cambio sellado con `iso` y un `axiom.yaml` que ya no declara `iso`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el status indica que el proyecto no declara ISO y que el cambio es ISO
- **Y** incluye una nota no bloqueante con la discrepancia

#### Scenario: Un cambio sin instantánea se proyecta como no ISO

- **DADO** un cambio sellado antes de esta versión y un `axiom.yaml` que declara `iso`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el status indica que el cambio es no ISO

#### Scenario: Un `sdd:` inválido se proyecta como inválido

- **DADO** un `axiom.yaml` con `sdd:` inválido
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el status proyecta la declaración como inválida
- **Y** no asume «no ISO»

#### Scenario: Sin el bloque un consumidor anterior no ve diferencia

- **DADO** un `axiom.yaml` sin `sdd:`
- **CUANDO** se compara el status con el de antes de este cambio
- **ENTONCES** ninguna clave existente cambia de nombre, tipo ni valor

#### Scenario: El contrato publicado documenta la proyección

- **DADO** `sdd-status-contract.md`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** documenta la proyección del valor del proyecto y del valor del cambio y que un cambio sin instantánea es no ISO

---

### Requirement: Identidad Estable `id` en los Documentos de Baseline Afectados (REQ-29.3)

En un cambio ISO con baseline declarada (`baseline:`), todo documento de la baseline afectado por una delta del cambio DEBE tener un perfil de requisito cuya identidad efectiva sea `id` (declarada o por omisión, REQ-24.15 de `spec-baseline-declaration`). La baseline implícita `openspec/specs/` cuenta como identidad `id`: la comprobación NO se aplica cuando `axiom.yaml` no declara `baseline:`, y en ella el identificador de un requisito es el `REQ-<n>.<m>` que contiene su nombre (supuesto visible). La comprobación DEBE hacerse en el estado del cambio, una vez escrita la delta y antes de habilitar la fase `design`. Un documento afectado de una baseline declarada, sin perfil de requisito o con `identity: text`, DEBE bloquear el cambio con un motivo tipado accionable que nombre el dominio, el documento y la continuación (declarar en la baseline un perfil con identidad `id`). Se admite `scenarios: omit` (REQ-24.60 de `spec-baseline-declaration`). Solo se comprueban los documentos afectados por la delta, nunca el resto de la baseline. Sin la declaración ISO, cualquier perfil es válido y esta regla NO DEBE bloquear.

#### Scenario: Documento afectado con identidad `text` bloquea antes de design

- **DADO** un cambio ISO y una delta para el dominio `requisitos-funcionales`, cuyo documento declara `identity: text`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de identidad ISO
- **Y** el motivo nombra el dominio `requisitos-funcionales`, su documento y la continuación
- **Y** la fase `design` no se proyecta como siguiente

#### Scenario: Documento afectado con identidad `id` no bloquea

- **DADO** un cambio ISO y una delta para un dominio cuyo documento tiene perfil con identidad `id`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta el motivo tipado de identidad ISO

#### Scenario: `scenarios: omit` se admite en ISO

- **DADO** un cambio ISO y un documento afectado con perfil de identidad `id` y `scenarios: omit`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta el motivo tipado de identidad ISO

#### Scenario: La baseline implícita cuenta como identidad `id` en ISO

- **DADO** un cambio ISO cuyo `axiom.yaml` no declara `baseline:` y una delta para un dominio de `openspec/specs/`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta el motivo tipado de identidad ISO

#### Scenario: Documento afectado de una baseline declarada sin perfil de requisito bloquea en ISO

- **DADO** un cambio ISO con `baseline:` declarada y una delta para un dominio cuyo documento no declara perfil de requisito
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con el motivo tipado de identidad ISO
- **Y** la continuación indica declarar un perfil con identidad `id` en la baseline

#### Scenario: Los documentos no afectados no se comprueban

- **DADO** un cambio ISO con dos documentos en la baseline, uno con `identity: text` sin relación con la delta
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json` para una delta que solo afecta al otro documento
- **ENTONCES** no se proyecta el motivo tipado de identidad ISO

#### Scenario: En un cambio no ISO cualquier perfil es válido

- **DADO** un cambio no ISO y una delta para un documento con `identity: text`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta ningún motivo de identidad ISO

---

### Requirement: Cobertura `Cubre:` de los Requisitos de la Delta (REQ-29.4)

En un cambio ISO, cada tarea de `tasks.md` o de `tasks.<rol>.md` DEBE poder declarar los requisitos que cubre con una línea `Cubre: REQ-x[, REQ-y ...]`. En la baseline implícita el identificador citado es el `REQ-<n>.<m>` del nombre del requisito (REQ-29.3). El motor DEBE analizar esas líneas en todos los ficheros de tareas del cambio y validar la cobertura conjunta, en cuanto existan los ficheros de tareas: cada requisito de la delta (ADDED por su ID provisional, y MODIFIED, REMOVED y RENAMED por su ID existente) DEBE aparecer en al menos un `Cubre:` del conjunto de ficheros. Un requisito de la delta sin cobertura DEBE bloquear con un motivo tipado que lo nombre. Un identificador de `Cubre:` que no existe ni en la delta ni en la baseline DEBE bloquear con un motivo tipado que lo nombre; uno que existe en la baseline pero no en la delta se admite y no cuenta para la cobertura. Fuera de ISO, `Cubre:` es informativo y opcional y NUNCA DEBE bloquear.

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

---

### Requirement: Bug sin Delta en un Cambio ISO: el Requisito Citado Existe y Tiene Escenarios (REQ-29.5)

En un cambio ISO declarado como bug sin delta (REQ-29.15 de `axiom-sdd-cli-integration`), el motor DEBE comprobar que el cambio cita, en los `Cubre:` de sus tareas, al menos un requisito, y que todo requisito citado existe en la baseline, con la consulta de REQ-29.14 de `spec-baseline-integration`. Un `Cubre:` que apunta a un requisito inexistente DEBE bloquear con un motivo tipado que nombre el identificador; un bug sin delta cuyas tareas no citan ningún requisito DEBE bloquear con un motivo tipado que lo indique. Los escenarios del requisito existente citado son los escenarios afectados de ese cambio: la matriz de verify (REQ-29.6) DEBE tener una fila por cada uno. Si el requisito citado existe pero no tiene ningún escenario, el cambio DEBE bloquear con un motivo tipado que lo nombre, porque no hay nada que verificar; la continuación es tramitar el trabajo como un cambio normal con delta que añada los escenarios, ya que el requisito estaba mal especificado. Fuera de ISO, un bug sin delta queda preparado con la sola declaración y esta comprobación NO DEBE bloquear.

#### Scenario: Un bug sin delta con `Cubre:` a un requisito inexistente bloquea en ISO

- **DADO** un cambio ISO declarado como bug sin delta y una tarea con `Cubre: REQ-13.99`, requisito que no existe en la baseline
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con un motivo tipado que nombra `REQ-13.99`

#### Scenario: Un bug sin delta con `Cubre:` a un requisito existente avanza

- **DADO** un cambio ISO declarado como bug sin delta y una tarea con `Cubre: REQ-13.1`, requisito existente con escenarios
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta ningún motivo de bloqueo de esta regla

#### Scenario: Un bug sin delta sin ningún `Cubre:` bloquea en ISO

- **DADO** un cambio ISO declarado como bug sin delta cuyas tareas no contienen `Cubre:`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con un motivo tipado que indica que ningún requisito existente está citado

#### Scenario: Una fila de la matriz por cada escenario del requisito existente

- **DADO** un bug sin delta ISO que cita un requisito existente con dos escenarios, y una matriz con una sola fila
- **CUANDO** el motor valida el `verify-report.md`
- **ENTONCES** la matriz es inválida con un motivo tipado que nombra el escenario sin fila

#### Scenario: El requisito citado sin escenarios bloquea y remite al cambio normal

- **DADO** un bug sin delta ISO que cita un requisito existente sin ningún escenario
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda bloqueado con un motivo tipado que nombra el requisito
- **Y** la continuación indica tramitar el trabajo como un cambio normal con delta

#### Scenario: Fuera de ISO un bug sin delta no se comprueba

- **DADO** un cambio no ISO declarado como bug sin delta y una tarea con `Cubre: REQ-13.99`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el cambio queda preparado sin delta
- **Y** no se proyecta ningún motivo de bloqueo por el identificador

---

### Requirement: Matriz de Verify con una Fila por Escenario, Dentro del Informe de Verify (REQ-29.6)

En un cambio ISO, el `verify-report.md` del cambio (el consolidado, en un cambio con varios roles; el mismo informe del rol, con el rol único `fullstack`) DEBE contener, como una sección adicional, una matriz tabular con **una fila por cada escenario afectado** y, por fila: requisito, escenario, tipo (`unitario`, `integración`, `e2e` o `manual`), referencia, resultado (`pass` o `fail`) y commit. La columna commit es opcional e informativa: el motor NO DEBE validarla contra git. La distribución exacta en columnas la fija el diseño. La matriz es una sección más del informe de REQ-28.4 de `sdd-independent-review`: NO sustituye a la sección «Cumplimiento técnico», NO DEBE escribir ninguna línea que empiece por `verdict:` (la barrera de sincronización lee la primera línea de ese tipo y su significado sigue siendo funcional) y NO altera el carácter informativo de las revisiones. Los informes por rol (`verify-report.<rol>.md`) NO necesitan matriz ni se validan. El motor DEBE validar que: (a) todo escenario afectado tiene fila; (b) ninguna fila apunta a un escenario o a un requisito que no figura entre los afectados (supuesto visible: invalida la matriz); (c) ningún escenario tiene más de una fila; (d) todo resultado es `pass` o `fail`; (e) las filas automáticas cumplen REQ-29.7 y las manuales REQ-29.8. Una matriz ausente cuenta como inválida. Una matriz válida es la que cumple (a) a (e); que sea **de éxito** exige además que ninguna fila tenga resultado `fail`. Cuando el informe conserva contenido de verificaciones anteriores como contexto histórico (REQ-28.8 de `sdd-independent-review`), solo la matriz del contenido vigente se valida. Los escenarios se identifican por el par (requisito, nombre del escenario): si un requisito repite el nombre de un escenario, el motor DEBE reportarlo con un motivo tipado que nombre el requisito y el escenario. Fuera de un cambio ISO, `verify-report.md` NO cambia de formato ni se valida más allá de lo que ya se valida.

#### Scenario: Matriz completa y válida

- **DADO** un cambio ISO y una delta con tres escenarios, con una fila por escenario, cada una válida según REQ-29.7 o REQ-29.8
- **CUANDO** el motor valida el `verify-report.md`
- **ENTONCES** la matriz es válida y no se proyecta ningún motivo de matriz

#### Scenario: Un escenario sin fila invalida la matriz

- **DADO** un cambio ISO y una delta con tres escenarios, y una matriz con solo dos filas
- **CUANDO** el motor valida el `verify-report.md`
- **ENTONCES** la matriz es inválida con un motivo tipado que nombra el escenario sin fila

#### Scenario: Una fila que apunta a un escenario inexistente invalida la matriz

- **DADO** una matriz con una fila cuyo escenario no figura en la delta
- **CUANDO** el motor valida el `verify-report.md`
- **ENTONCES** la matriz es inválida con un motivo tipado que nombra la fila y el escenario inexistente

#### Scenario: Dos filas para el mismo escenario invalidan la matriz

- **DADO** una matriz con dos filas para el mismo escenario del mismo requisito
- **CUANDO** el motor valida el `verify-report.md`
- **ENTONCES** la matriz es inválida con un motivo tipado que nombra el escenario duplicado

#### Scenario: Una fila sin resultado o con resultado desconocido invalida la matriz

- **DADO** una matriz con una fila cuya columna resultado está vacía y otra con el valor `quizás`
- **CUANDO** el motor valida el `verify-report.md`
- **ENTONCES** la matriz es inválida con un motivo tipado por cada fila, que nombra la fila

#### Scenario: Una fila fallida hace que la matriz no sea de éxito

- **DADO** una matriz válida con una fila de resultado `fail`
- **CUANDO** el motor evalúa la matriz
- **ENTONCES** la matriz es válida pero no de éxito
- **Y** el motivo nombra la fila fallida

#### Scenario: Una matriz ausente es inválida en ISO

- **DADO** un cambio ISO con `verify-report.md` sin matriz, o sin `verify-report.md`
- **CUANDO** el motor evalúa la matriz
- **ENTONCES** la matriz es inválida con un motivo tipado de matriz ausente

#### Scenario: La matriz no escribe líneas `verdict:`

- **DADO** un `verify-report.md` de un cambio ISO con matriz y una línea funcional `verdict: pass` previa
- **CUANDO** la barrera de sincronización lee el informe
- **ENTONCES** lee la misma línea `verdict:` que sin la matriz
- **Y** ninguna línea de la matriz empieza por `verdict:`

#### Scenario: La matriz convive con «Cumplimiento técnico»

- **DADO** un cambio ISO con el rol único `fullstack` y un `verify-report.md` con la sección «Cumplimiento técnico» y la matriz
- **CUANDO** se evalúa el informe
- **ENTONCES** ambas secciones coexisten sin que una altere a la otra

#### Scenario: Los informes por rol no se validan

- **DADO** un cambio ISO con los roles `core` y `web`, un `verify-report.core.md` sin matriz y un `verify-report.md` consolidado con matriz válida
- **CUANDO** el motor evalúa la matriz
- **ENTONCES** solo se valida la del consolidado
- **Y** no se proyecta ningún motivo por la ausencia de matriz en el informe de `core`

#### Scenario: Solo se valida la matriz del contenido vigente

- **DADO** un informe con una matriz inválida marcada como contexto histórico de una verificación anterior y una matriz vigente válida
- **CUANDO** el motor valida el `verify-report.md`
- **ENTONCES** la matriz es válida

#### Scenario: Nombre de escenario repetido dentro de un requisito

- **DADO** una delta con un requisito que contiene dos escenarios con el mismo nombre
- **CUANDO** el motor valida la matriz
- **ENTONCES** reporta un motivo tipado que nombra el requisito y el escenario repetido

#### Scenario: La columna commit no se valida

- **DADO** una fila válida con un commit que no figura en el repositorio
- **CUANDO** el motor valida el `verify-report.md`
- **ENTONCES** la fila es válida

#### Scenario: Fuera de ISO el informe de verify es libre

- **DADO** un cambio no ISO y un `verify-report.md` sin matriz
- **CUANDO** se evalúa el cambio
- **ENTONCES** no se proyecta ningún motivo de matriz

---

### Requirement: Las Filas Automáticas Apuntan a una Prueba que Existe en el Árbol y Registran su Ejecución (REQ-29.7)

En un cambio ISO, toda fila automática (tipo `unitario`, `integración` o `e2e`) DEBE: (a) llevar una referencia de prueba con una de las dos formas admitidas: `ruta/fichero:símbolo`, válida para cualquier lenguaje, o el atajo `TestX` para proyectos en Go; (b) registrar el comando ejecutado, y (c) registrar su código de salida (un entero). El motor DEBE comprobar que la referencia **resuelve a una prueba existente en el árbol de trabajo**: con la forma `ruta/fichero:símbolo`, que el fichero existe en el árbol y que el símbolo figura en él como identificador completo (no como subcadena de otro identificador; supuesto visible: en lenguajes distintos de Go la resolución es textual, no sintáctica); con el atajo `TestX`, que algún fichero `*_test.go` del árbol declara la función de nivel superior `TestX`. Una ruta absoluta o que salga del árbol NO DEBE resolver. El motor DEBE comprobar además que el resultado es coherente con el código de salida: una fila `pass` con código de salida distinto de `0` es inválida (supuesto visible). Una fila automática sin referencia con forma admitida, con una referencia que no resuelve, sin comando o sin código de salida DEBE invalidar la matriz con un motivo tipado que nombre la fila y el dato ausente o no resuelto. El motor NO DEBE reejecutar el comando ni confirmar que la prueba pase: el resultado sigue siendo declarado, y lo contrastable es que la prueba existe y que el comando y su código de salida quedan registrados para repetirla.

#### Scenario: Una fila automática con referencia resoluble, comando y código de salida es válida

- **DADO** una fila `unitario` con la referencia `internal/pagos/cobro_test.go:TestCobraIVA`, símbolo presente en ese fichero, el comando `go test ./internal/pagos -run TestCobraIVA -count=1`, código de salida `0` y resultado `pass`
- **CUANDO** el motor valida la matriz
- **ENTONCES** la fila es válida

#### Scenario: El atajo `TestX` resuelve contra los ficheros de prueba de Go

- **DADO** una fila `integración` con la referencia `TestArchiveBlocksOnMatrix` y una función con ese nombre declarada en un `*_test.go` del árbol
- **CUANDO** el motor valida la matriz
- **ENTONCES** la referencia resuelve y la fila es válida

#### Scenario: El atajo `TestX` no resuelve si ningún fichero de prueba lo declara

- **DADO** una fila con la referencia `TestInexistente` y ningún `*_test.go` que la declare
- **CUANDO** el motor valida la matriz
- **ENTONCES** la matriz es inválida con un motivo tipado que nombra la fila y la referencia no resuelta

#### Scenario: Un fichero inexistente no resuelve

- **DADO** una fila con la referencia `e2e/checkout.spec.ts:pagaConTarjeta` y ningún fichero `e2e/checkout.spec.ts` en el árbol
- **CUANDO** el motor valida la matriz
- **ENTONCES** la matriz es inválida con un motivo tipado que nombra la fila y el fichero ausente

#### Scenario: Un símbolo ausente del fichero no resuelve

- **DADO** una fila con la referencia `e2e/checkout.spec.ts:pagaConTarjeta` y un fichero que existe pero no contiene `pagaConTarjeta`
- **CUANDO** el motor valida la matriz
- **ENTONCES** la matriz es inválida con un motivo tipado que nombra la fila y el símbolo ausente

#### Scenario: El símbolo debe figurar como identificador completo

- **DADO** una fila con la referencia `e2e/checkout.spec.ts:paga` y un fichero que solo contiene `pagaConTarjeta`
- **CUANDO** el motor valida la matriz
- **ENTONCES** la referencia no resuelve

#### Scenario: Una referencia en lenguaje distinto de Go con la forma general resuelve

- **DADO** una fila con la referencia `tests/test_pago.py:test_cobra_iva` y un fichero que declara `test_cobra_iva`
- **CUANDO** el motor valida la matriz
- **ENTONCES** la referencia resuelve

#### Scenario: Una ruta que sale del árbol no resuelve

- **DADO** una fila con la referencia `../otro/prueba_test.go:TestX` o con una ruta absoluta
- **CUANDO** el motor valida la matriz
- **ENTONCES** la matriz es inválida con un motivo tipado que nombra la fila y la ruta

#### Scenario: Sin comando o sin código de salida la fila es inválida

- **DADO** una fila automática con referencia resoluble pero sin comando ejecutado, y otra sin código de salida
- **CUANDO** el motor valida la matriz
- **ENTONCES** la matriz es inválida con un motivo tipado por cada fila, que nombra el dato ausente

#### Scenario: Un resultado `pass` con código de salida distinto de cero es inválido

- **DADO** una fila automática con resultado `pass` y código de salida `1`
- **CUANDO** el motor valida la matriz
- **ENTONCES** la matriz es inválida con un motivo tipado que nombra la fila y la incoherencia

#### Scenario: Una referencia sin ninguna de las dos formas es inválida

- **DADO** una fila automática con la referencia `la prueba de pagos`
- **CUANDO** el motor valida la matriz
- **ENTONCES** la matriz es inválida con un motivo tipado que nombra la fila y las formas admitidas

#### Scenario: El motor no reejecuta el comando

- **DADO** una fila válida cuyo comando registrado fallaría si se ejecutara
- **CUANDO** el motor valida la matriz
- **ENTONCES** la fila es válida porque la validación contrasta la existencia de la prueba y el registro de la ejecución, no su resultado
- **Y** no se ejecuta ningún comando

---

### Requirement: Las Filas Manuales Justifican Quién, Cuándo y Por Qué No se Automatiza (REQ-29.8)

En un cambio ISO, toda fila manual (tipo `manual`) DEBE justificar la evidencia con tres datos: quién la obtuvo (una persona identificable), cuándo (una fecha en formato ISO 8601, `AAAA-MM-DD`) y por qué no se automatiza (un texto no vacío). La referencia de una fila manual es opcional, informativa y NO se resuelve contra el árbol; una fila manual NO necesita comando ni código de salida. Una fila manual a la que le falte alguno de los tres datos, o cuya fecha no sea válida, DEBE invalidar la matriz con un motivo tipado que nombre la fila y el dato que falta.

#### Scenario: Una fila manual con justificación completa es válida

- **DADO** una fila `manual` que indica quién (`Ana Pérez`), cuándo (`2026-11-12`) y por qué no se automatiza
- **CUANDO** el motor valida la matriz
- **ENTONCES** la fila es válida

#### Scenario: Una fila manual sin justificación invalida la matriz

- **DADO** una fila `manual` que no indica quién, cuándo o por qué no se automatiza
- **CUANDO** el motor valida la matriz
- **ENTONCES** la matriz es inválida con un motivo tipado que nombra la fila y el dato que falta

#### Scenario: Una fecha no válida invalida la fila manual

- **DADO** una fila `manual` con la fecha `ayer`
- **CUANDO** el motor valida la matriz
- **ENTONCES** la matriz es inválida con un motivo tipado que nombra la fila y la fecha

#### Scenario: La referencia de una fila manual no se resuelve

- **DADO** una fila `manual` completa con la referencia `acta-de-pruebas-12` que no corresponde a ningún fichero del árbol
- **CUANDO** el motor valida la matriz
- **ENTONCES** la fila es válida

---

### Requirement: Trazabilidad por `git log --first-parent` Independiente de la Estrategia de Fusión y Compatible con la Rama `-archive` (REQ-29.9)

Para un cambio ISO archivado por la rama `<prefijo>/<cambio>-archive` (`monorepo-embedded`), Axiom DEBE poder derivar, con `git log --first-parent <rama de integración>` y sin ningún estado persistido nuevo, dos commits de la línea principal: (a) el **commit de integración del código**, el primero que incorpora la carpeta `openspec/changes/<cambio>/` con sus `tasks.md` y `verify-report.md`; y (b) el **commit de archivado**, el primero que incorpora `openspec/changes/archive/<fecha>-<cambio>/`. La derivación DEBE dar el mismo resultado con fusión por squash, por commit de merge y por rebase, tanto para el PR del cambio como para el PR documental del archivado. Cuando la compuerta `integration` registra evidencia `pr_merged` con un commit (`internal/kickoff`), el commit de código derivado DEBE contrastarse con ella: DEBE ser ese commit o un ancestro suyo en la línea principal, y una discrepancia se informa sin error de ejecución. DEBE reutilizar la traza de integración existente (`integration.yaml` y la asignación de IDs, que relaciona el ID provisional con el definitivo) y NO DEBE introducir estado persistido nuevo. La cadena requisito definitivo → ID provisional → `Cubre:` de las tareas archivadas → fila de la matriz del informe archivado → commit de integración del código → commit de archivado DEBE poder reconstruirse a partir de los artefactos archivados y del historial de la rama principal. Los trailers de los commits intermedios son informativos y NUNCA se validan. Si la carpeta archivada, o la carpeta del cambio, no existe todavía en la rama principal, la derivación DEBE informar de la ausencia del commit correspondiente sin error de ejecución. En `multirepo` la derivación no se especifica en este cambio (supuesto visible).

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

#### Scenario: Un ID provisional se enlaza con el definitivo por la traza existente

- **DADO** un cambio ISO con baseline declarada y `identity: id`, archivado, cuyo requisito ADDED `RF-9001` se archivó con el ID definitivo `RF-142` y la entrada correspondiente de `integration.yaml`
- **CUANDO** se reconstruye la cadena desde `RF-142`
- **ENTONCES** se llega a `RF-9001`, a su `Cubre:` en las tareas archivadas y a su fila de la matriz
