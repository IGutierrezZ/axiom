# Especificación de Requerimientos: Adopción de OpenSpec como Spec Viva Única (RF-ADP-01 a RF-ADP-06)

> **Cambio:** `spec-viva-openspec` · **Tipo:** capacidad nueva (especificación completa, no delta)
> **Idioma:** Español (castellano peninsular)

## Propósito

Definir cómo un proyecto que llega con documentación de requisitos previa, en cualquier formato, la migra a `openspec/specs/` mediante un cambio SDD de adopción: un inventario determinista de los orígenes (`axiom archive adopt scan`), un formato del cambio de adopción y una verificación determinista de ese cambio (`axiom archive adopt check`), con la aprobación humana de la compuerta `spec`.

**Resultado esperado:** toda unidad del origen queda migrada a un requisito OpenSpec con su procedencia (`Origen:`) o descartada con un motivo explícito, con identificadores únicos y al menos un escenario por requisito, y un humano ha aprobado la migración.

## Términos

| Término | Definición |
|---|---|
| Origen | Fichero o conjunto de ficheros de requisitos previos al proyecto, fuera de `openspec/specs/`. |
| Unidad | Parte direccionable de un origen: un fichero o un encabezado, identificada por ruta y ancla. |
| Inventario | Lista determinista de orígenes y unidades que `adopt scan` produce; es la referencia de cobertura. |
| Cambio de adopción | Cambio SDD cuya delta solo contiene `## ADDED Requirements` que migran unidades del inventario. |
| `Origen:` | Línea del cuerpo de un requisito migrado con la ruta, el ancla y el identificador original de su unidad. |
| `Descartado:` | Lista de unidades del inventario que no se migran, cada una con su motivo. |

## Requerimientos de un vistazo

| ID | Requerimiento | Garantía principal |
|---|---|---|
| RF-ADP-01 | `axiom archive adopt scan` | Inventario determinista; sin `--change` no escribe nada |
| RF-ADP-02 | Formato del cambio de adopción | Solo ADDED, `Origen:` por requisito, `Descartado:` con motivo |
| RF-ADP-03 | `axiom archive adopt check --change <c>` | Verificaciones deterministas con todos los fallos reportados |
| RF-ADP-04 | Series de identificadores | Los IDs del cliente se conservan según el patrón declarado; sin IDs, provisionales |
| RF-ADP-05 | Aprobación humana y archivado | La compuerta `spec` aprueba; `Origen:` se conserva al archivar |
| RF-ADP-06 | Multirepo | Opera sobre el repositorio de especificaciones |

## No cubre esta capacidad

- Que Go juzgue la fidelidad semántica de la migración: cubre la compuerta `spec` humana.
- Importadores por formato (Word, hojas de cálculo) y cualquier destino distinto de OpenSpec.
- Migrar los `REQ-x.y` propios de Axiom (C-03-e), validar normas ISO publicadas, normalizar o reescribir automáticamente los identificadores del cliente y que `adopt` escriba `axiom.yaml`.
- Que el dashboard ejecute modelos: solo muestra el informe y la instrucción (`multi-project-hub`).
- Que `axiom init` escriba `technical_context:` por defecto: solo se propone en el informe de `adopt scan`.

---

## 1. Capacidad: `openspec-adoption`

### Requirement: `axiom archive adopt scan` Inventaría los Orígenes (RF-ADP-01)

`axiom archive adopt scan [--path <dir>] [--change <cambio>] [--json]` DEBE producir un inventario de solo lectura de los orígenes candidatos del proyecto (ficheros de requisitos fuera de `openspec/specs/` y de `openspec/changes/`) y de sus unidades: para cada origen, sus ficheros, sus encabezados, los identificadores ajenos reconocidos por patrón (por ejemplo `RF-120`), los candidatos a `technical_context` (documentación mantenida y de referencia) propuestos para declarar en `axiom.yaml` y, cuando los identificadores ajenos lo permiten, una propuesta de patrón de identificadores (`sdd.id_pattern`). El inventario DEBE ser determinista: el mismo árbol produce el mismo inventario con el mismo orden y las mismas anclas. Sin `--change`, la orden NO DEBE escribir ningún fichero. Con `--change <cambio>`, DEBE escribir únicamente el inventario en la carpeta del cambio (`adoption-inventory.yaml`, formato `axiom.openspec-adoption-inventory/v1`) y nada más. Como escritor Go de la carpeta del cambio, DEBE consultar la guarda A de RF-BRA-20 (`change-branch-lifecycle`) en el contexto ordinario antes de escribir y, ante `wrong_branch`, DEBE negarse sin escribir nada y devolver la continuación de `axiom branch ensure`. Sin ningún origen candidato, la orden DEBE informarlo y finalizar con código `0`. La orden NO DEBE requerir que exista `openspec/`.

**Propuesta de patrón.** A partir de los identificadores ajenos de TODAS las unidades de todos los orígenes, `scan` DEBE calcular un patrón candidato único con su evidencia y sus conflictos, y mostrarlo en el informe de texto junto a los candidatos a `technical_context`. En el inventario JSON se expresa como el campo aditivo `proposed_id_pattern`, omitido cuando no hay identificadores ajenos; el formato `axiom.openspec-adoption-inventory/v1` NO sube de versión y el campo no interviene en la detección de inventario desactualizado. La propuesta DEBE cumplir:

1. La plantilla propuesta es un patrón válido según `reqid` y reconoce como válidos TODOS los identificadores ajenos de los que se deriva; si no existe una plantilla única que lo cumpla, NO se propone plantilla y se informan los conflictos.
2. Con un único prefijo, un único separador y el mismo ancho con ceros iniciales (`RF-001`…`RF-120`), propone `RF-<NNN>`; con un único esquema de ancho mixto (`RF-1`, `RF-001`), propone `RF-<m>`; con dominio en todos los identificadores, usa `<DOM>`.
3. Cualquier otra mezcla (varios prefijos como `RF` y `RNF`, con y sin dominio, separadores distintos, esquemas incompatibles) NO se admite: se informa como conflicto nombrado, con el recuento por prefijo, separador, dominio y ancho como evidencia, y sin plantilla cerrada. Axiom NO normaliza ni reescribe los identificadores del cliente: normalizarlos es tarea de la persona.
4. La propuesta es determinista: el mismo árbol produce la misma propuesta y las mismas evidencias en el mismo orden.
5. `scan` NO DEBE escribir `axiom.yaml` ni ofrecer una opción que lo escriba, con o sin `--change`; la persona declara el patrón.

Supuesto: el inventario con `--change` es la referencia de cobertura de RF-ADP-03; sin esa opción, el detector del hub lo reutiliza sin escribir (RF-HUB-10 de `multi-project-hub`). Los identificadores ajenos que `scan` reconoce son los de su lista cerrada de prefijos; con un prefijo fuera de ella no hay propuesta y la persona puede declarar el patrón a mano.
(Previously: el inventario solo listaba los identificadores ajenos y los candidatos a `technical_context`; no proponía ningún patrón de identificadores.)

#### Scenario: El inventario lista orígenes y unidades sin escribir
- **DADO** un proyecto con `docs/requisitos.md` con dos encabezados que contienen `RF-001` y `RF-002`
- **CUANDO** se ejecuta `axiom archive adopt scan`
- **ENTONCES** el inventario lista el origen, sus dos unidades con su ruta y ancla, y los identificadores `RF-001` y `RF-002`
- **Y** no se crea ni se modifica ningún fichero

#### Scenario: El inventario es determinista
- **DADO** el mismo árbol de ficheros
- **CUANDO** se ejecuta `adopt scan --json` dos veces
- **ENTONCES** las dos salidas son idénticas, incluido `proposed_id_pattern`

#### Scenario: Con `--change` solo se escribe el inventario
- **DADO** un cambio `adopcion-requisitos`
- **CUANDO** se ejecuta `axiom archive adopt scan --change adopcion-requisitos`
- **ENTONCES** la carpeta del cambio contiene el inventario
- **Y** ningún otro fichero del proyecto cambia, `axiom.yaml` incluido

#### Scenario: `adopt scan --change` se niega fuera de rama
- **DADO** un cambio `adopcion-requisitos` con instantánea y el repositorio desplegado en la rama de integración
- **CUANDO** se ejecuta `axiom archive adopt scan --change adopcion-requisitos`
- **ENTONCES** se niega con `wrong_branch` y la continuación de `axiom branch ensure`
- **Y** no se crea el inventario ni ningún otro fichero de la carpeta del cambio

#### Scenario: Sin orígenes candidatos
- **DADO** un proyecto sin documentación de requisitos fuera de `openspec/`
- **CUANDO** se ejecuta `adopt scan`
- **ENTONCES** informa de que no hay orígenes candidatos y el código de salida es `0`
- **Y** no hay propuesta de patrón

#### Scenario: Candidatos a `technical_context`
- **DADO** un proyecto con documentación técnica y de requisitos sin bloque `technical_context:`
- **CUANDO** se ejecuta `adopt scan`
- **ENTONCES** el informe propone candidatos a `maintained` y a `reference` sin modificar `axiom.yaml`

#### Scenario: Se propone `RF-<NNN>` para `RF-001`…`RF-120`
- **DADO** un origen cuyas unidades llevan los identificadores `RF-001` a `RF-120`
- **CUANDO** se ejecuta `adopt scan --json`
- **ENTONCES** `proposed_id_pattern` propone la plantilla `RF-<NNN>` con la evidencia de 120 identificadores del prefijo `RF` con ancho 3
- **Y** no informa ningún conflicto
- **Y** la plantilla reconoce como válidos los 120 identificadores

#### Scenario: Anchos mixtos del mismo esquema proponen `RF-<m>`
- **DADO** un origen con `RF-1`, `RF-2` y `RF-010`
- **CUANDO** se ejecuta `adopt scan --json`
- **ENTONCES** `proposed_id_pattern` propone la plantilla `RF-<m>`
- **Y** no informa ningún conflicto

#### Scenario: Prefijos distintos son un conflicto nombrado
- **DADO** un origen con `RF-001` y `RNF-001`
- **CUANDO** se ejecuta `adopt scan`
- **ENTONCES** el informe nombra el conflicto entre los prefijos `RF` y `RNF` con su recuento
- **Y** no propone ninguna plantilla cerrada
- **Y** no se modifica ningún fichero

#### Scenario: Esquemas incompatibles son un conflicto
- **DADO** un origen con `RF-001` y `RF-USU-01` (con y sin dominio), o con `RF-001` y `RF_001` (separadores distintos)
- **CUANDO** se ejecuta `adopt scan`
- **ENTONCES** el informe nombra el conflicto con su evidencia y no propone plantilla

#### Scenario: Sin identificadores ajenos no hay propuesta
- **DADO** un origen sin ningún identificador ajeno reconocido
- **CUANDO** se ejecuta `adopt scan --json`
- **ENTONCES** el inventario NO contiene el campo `proposed_id_pattern`
- **Y** un lector del formato `v1` anterior lo procesa sin cambios

#### Scenario: `scan` nunca escribe `axiom.yaml`
- **DADO** un proyecto con identificadores ajenos y un cambio de adopción
- **CUANDO** se ejecuta `adopt scan --change <cambio>`, con o sin opciones adicionales
- **ENTONCES** `axiom.yaml` queda byte a byte igual
- **Y** la orden no admite ninguna opción que escriba el patrón

---

### Requirement: Formato del Cambio de Adopción (RF-ADP-02)

Un cambio de adopción es un cambio SDD ordinario (kickoff, compuertas y archivado normales) cuya delta de spec DEBE contener únicamente `## ADDED Requirements`. Cada requisito DEBE llevar en su título exactamente un identificador: o bien un identificador definitivo válido según el patrón declarado (`(RF-001)` con `RF-<NNN>`; `(REQ-n.m)` con el patrón por defecto), o bien un identificador provisional de RF-RID-09 (`(prov:<series>:<ordinal>)`). Cada requisito DEBE llevar en su cuerpo al menos una línea `Origen: <ruta>#<ancla> (<ID original>)`, donde `<ID original>` es el identificador de la unidad en el origen cuando existe y se omite cuando la unidad no tiene identificador. Las unidades del inventario que no se migran DEBEN figurar en una lista `Descartado:` del cambio, cada una con su motivo (`<ruta>#<ancla>: <motivo>`, no vacío); la ubicación exacta de la lista dentro de la carpeta del cambio la fija el diseño. La composición al archivar conserva el texto `Origen:` del cuerpo sin tratamiento especial.
(Previously: el título llevaba un identificador `(REQ-n.m)` fijo del patrón por defecto, sin identificadores del cliente ni provisionales.)

#### Scenario: Requisito migrado con su procedencia
- **DADO** un proyecto con `sdd.id_pattern: RF-<NNN>` y un requisito `(RF-001)` con `Origen: docs/requisitos.md#rf-001 (RF-001)` y un escenario
- **CUANDO** se analiza el cambio de adopción
- **ENTONCES** el requisito se reconoce con su identificador `RF-001`, su procedencia y su identificador original `RF-001`

#### Scenario: Requisito con patrón por defecto
- **DADO** un proyecto sin `sdd.id_pattern` y un requisito `(REQ-40.1)` con `Origen: docs/requisitos.md#rf-001`
- **CUANDO** se analiza el cambio de adopción
- **ENTONCES** el requisito se reconoce con el identificador `REQ-40.1` y su procedencia

#### Scenario: Requisito de un origen sin identificador con provisional
- **DADO** un requisito con título `(prov:30:<k>)` y `Origen: docs/vision.md#alcance` sin identificador original
- **CUANDO** se analiza el cambio de adopción
- **ENTONCES** el requisito se reconoce con su provisional y su procedencia

#### Scenario: Una unidad descartada con motivo
- **DADO** una lista `Descartado:` con `docs/requisitos.md#glosario: no es un requisito`
- **CUANDO** se analiza el cambio
- **ENTONCES** la unidad se reconoce como descartada con su motivo

#### Scenario: Un requisito puede citar varios orígenes
- **DADO** un requisito con dos líneas `Origen:` que apuntan a dos unidades
- **CUANDO** se analiza el cambio
- **ENTONCES** ambas unidades cuentan como cubiertas por ese requisito

---

### Requirement: `axiom archive adopt check --change <cambio>` Verifica el Cambio de Adopción (RF-ADP-03)

`axiom archive adopt check --change <cambio> [--path <dir>] [--json]` DEBE verificar de forma determinista, sin modelo y sin escribir ningún fichero, midiendo con el patrón efectivo del proyecto (el declarado en `axiom.yaml`, `sdd.id_pattern`, o el por defecto; el mismo que usan `archive-integrate` y el resto de verbos): (a) formato: la delta solo contiene `## ADDED Requirements` y cada requisito tiene en su título un identificador definitivo conforme al patrón efectivo o un provisional válido (RF-RID-09); (b) unicidad: ningún identificador se repite dentro del cambio ni coincide con uno de `openspec/specs/` o de otro cambio activo, comparando por clave de identidad (serie, número entero) y no por cadena, de modo que `RF-1` y `RF-001` cuentan como el mismo identificador y por tanto duplicado; (c) escenarios: cada requisito tiene al menos un `#### Scenario:`; (d) procedencia: cada requisito tiene al menos una línea `Origen:` y cada `Origen:` resuelve a un fichero y a un ancla existentes; (e) cobertura: cada unidad del inventario figura en algún `Origen:` o en `Descartado:` con motivo no vacío; (f) coincidencia del identificador original: cuando el título lleva un identificador definitivo, y alguna línea `Origen:` trae `<ID original>`, el identificador del título DEBE coincidir con el `<ID original>` de al menos una de esas líneas (comparando por clave de identidad); un título con identificador provisional cuyo `Origen:` trae `<ID original>` se rechaza, porque el identificador del cliente se conserva; (g) patrón declarado (`pattern_undeclared`): cuando el inventario guardado trae identificadores ajenos y el patrón efectivo no los reconoce como válidos (no hay `sdd.id_pattern` y el por defecto no los cubre, o el declarado no los cubre), la orden DEBE rechazar con un motivo que cite las líneas exactas de `sdd:` / `id_pattern:` que hay que copiar en `axiom.yaml`. La regla se deriva de los identificadores ajenos del inventario guardado, de modo que también funciona con inventarios escritos antes de este cambio. La regla de serie única por cambio ya NO existe: un cambio de adopción PUEDE usar la serie, el prefijo o el dominio que el patrón declarado admita. La orden DEBE reportar todos los fallos, no solo el primero, cada uno con un motivo tipado que nombre el requisito o la unidad y la regla incumplida, y finalizar con código distinto de cero si hay alguno. Sin inventario en la carpeta del cambio, DEBE rechazar con un motivo tipado que remita a `axiom archive adopt scan --change <cambio>`. Un patrón declarado inaccesible o inválido DEBE dar el mismo error accionable que el resto de verbos.
(Previously: formato y unicidad se medían con el patrón por defecto fijo `REQ-n.m`, no admitía provisionales, comparaba la unicidad por cadena, no comparaba el identificador original con el del título, no existía `pattern_undeclared` y una regla de serie única rechazaba mezclas y series existentes.)

#### Scenario: Un cambio válido pasa
- **DADO** un cambio de adopción con solo ADDED, identificadores únicos conformes al patrón efectivo, escenarios, `Origen:` resolubles y todas las unidades cubiertas o descartadas con motivo
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** finaliza con código `0`

#### Scenario: Identificadores del cliente con el patrón declarado pasan
- **DADO** `sdd.id_pattern: RF-<NNN>` y requisitos `(RF-001)`…`(RF-120)` cada uno con `Origen:` que trae su mismo `<ID original>`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** no se reporta ningún fallo de formato, unicidad ni coincidencia de identificador original

#### Scenario: Un MODIFIED se rechaza
- **DADO** una delta de adopción con un `## MODIFIED Requirements`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla nombrando la sección y la regla de formato

#### Scenario: Identificador fuera del patrón declarado
- **DADO** `sdd.id_pattern: RF-<NNN>` y un requisito `(REQ-40.1)` o `(RF-USU-01)`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla la regla de formato nombrando el requisito, su identificador y el patrón declarado

#### Scenario: Identificador duplicado dentro del cambio
- **DADO** dos requisitos `(RF-001)` en el cambio
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla nombrando el identificador y los dos requisitos

#### Scenario: Colisión con la spec viva o con otro cambio activo
- **DADO** un requisito `(RF-001)` y un `RF-001` ya presente en `openspec/specs/` o en un cambio activo
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla nombrando el identificador y dónde ya existe

#### Scenario: Anchos distintos del mismo número son duplicado
- **DADO** `sdd.id_pattern: RF-<m>` y requisitos `(RF-1)` y `(RF-001)`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla como duplicado nombrando ambos requisitos y la clave (serie `RF`, número `1`)

#### Scenario: Una mezcla que no cubre el patrón declarado se rechaza
- **DADO** `sdd.id_pattern: RF-<NNN>` y requisitos `(RF-001)` y `(RNF-001)`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla la regla de formato en el requisito `RNF-001`
- **Y** la orden no reescribe ningún identificador

#### Scenario: Coincidencia del identificador original con el del título
- **DADO** un requisito `(RF-001)` con `Origen: docs/requisitos.md#rf-002 (RF-002)`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla nombrando el requisito, el identificador del título y el `<ID original>` de su `Origen:`

#### Scenario: Con varios orígenes basta que el título coincida con alguno
- **DADO** un requisito `(RF-001)` con dos líneas `Origen:`, una con `(RF-001)` y otra con `(RF-002)`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** no se reporta fallo de coincidencia de identificador original

#### Scenario: Con varios orígenes ninguno coincide
- **DADO** un requisito `(RF-001)` con dos líneas `Origen:` con `(RF-002)` y `(RF-003)`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla nombrando el requisito y los dos identificadores originales

#### Scenario: Un provisional cuyo `Origen:` trae identificador se rechaza
- **DADO** un requisito titulado con un provisional y `Origen: docs/requisitos.md#rf-001 (RF-001)`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla nombrando el requisito e indicando que debe conservar el identificador del cliente `RF-001`

#### Scenario: Un provisional con origen sin identificador pasa
- **DADO** un requisito titulado con un provisional válido y `Origen:` sin `<ID original>`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** no se reporta fallo de formato ni de coincidencia de identificador original

#### Scenario: `pattern_undeclared` sin patrón declarado
- **DADO** un inventario con identificadores ajenos `RF-001`…`RF-120`, un proyecto sin `sdd.id_pattern` y un cambio válido en lo demás
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla con la regla `pattern_undeclared`
- **Y** el motivo cita las líneas exactas `sdd:` e `id_pattern: "RF-<NNN>"` para copiar en `axiom.yaml`

#### Scenario: `pattern_undeclared` con un patrón declarado que no cubre el origen
- **DADO** el mismo inventario y `sdd.id_pattern: HU-<NNN>`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla con la regla `pattern_undeclared` citando el patrón propuesto

#### Scenario: `pattern_undeclared` con un inventario anterior a este cambio
- **DADO** un inventario sin el campo `proposed_id_pattern` pero con identificadores ajenos que el patrón efectivo no cubre
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla con la regla `pattern_undeclared`

#### Scenario: Origen sin identificadores no exige declarar patrón
- **DADO** un inventario sin identificadores ajenos y un proyecto sin `sdd.id_pattern`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** no se reporta `pattern_undeclared`

#### Scenario: Ya no hay regla de serie única
- **DADO** un cambio con requisitos `(REQ-40.1)` y `(REQ-41.1)`, o con una serie `28` ya presente en `openspec/specs/` y un número libre dentro de ella
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** no se reporta ningún fallo de serie
- **Y** la unicidad se evalúa solo por clave de identidad (serie, número)

#### Scenario: Requisito sin escenario
- **DADO** un requisito sin ningún `#### Scenario:`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla nombrando el requisito

#### Scenario: `Origen:` que no resuelve
- **DADO** un `Origen:` con una ruta inexistente y otro con un ancla que no existe en el fichero
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla nombrando cada `Origen:`, la ruta o el ancla que no resuelve

#### Scenario: Unidad sin cubrir ni descartar
- **DADO** una unidad del inventario ausente de todo `Origen:` y de `Descartado:`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla nombrando la unidad

#### Scenario: Descartado sin motivo
- **DADO** una unidad en `Descartado:` sin motivo
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla nombrando la unidad y la falta de motivo

#### Scenario: Se reportan todos los fallos
- **DADO** un cambio con un requisito sin escenario, una unidad sin cubrir y un identificador duplicado
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** el informe contiene los tres fallos

#### Scenario: Sin inventario
- **DADO** un cambio de adopción sin inventario en su carpeta
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** se rechaza remitiendo a `axiom archive adopt scan --change <cambio>`

#### Scenario: `check` no escribe
- **DADO** cualquier cambio de adopción, válido o no
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** no se crea ni se modifica ningún fichero, `axiom.yaml` incluido

---

### Requirement: Series de Identificadores de la Migración (RF-ADP-04)

La migración DEBE conservar los identificadores funcionales del cliente como identidad de los requisitos migrados, según el patrón declarado del proyecto: cuando la unidad del origen trae identificador (`RF-001`), el requisito lleva ESE identificador definitivo en su título, y la spec viva archivada lo conserva. El proyecto DEBE declarar el patrón heredado en `sdd.id_pattern` de `axiom.yaml` (por ejemplo `RF-<NNN>`), dentro de la rama del cambio de adopción y antes de `adopt check` y de archivar; `adopt scan` lo propone pero NO lo escribe (RF-ADP-01). Los anchos mixtos del mismo esquema (`RF-1` y `RF-001`) se admiten con el marcador `<m>` del patrón y cuentan como un único identificador por clave de identidad (serie, número entero). Cualquier otra mezcla de esquemas (varios prefijos, con y sin dominio, separadores distintos) NO se admite con el patrón único: `scan` la informa como conflicto y `adopt check` la rechaza; normalizar los identificadores del origen es tarea de la persona y Axiom NUNCA reescribe identificadores del cliente.

Cuando la unidad del origen NO trae identificador, el requisito lleva un identificador provisional de RF-RID-09 y su número se asigna al archivar el cambio con `axiom sdd archive-integrate`, como siguiente número libre del patrón por defecto o del declarado. Un requisito nuevo integrado después en el mismo proyecto continúa la serie del cliente con el siguiente número libre (tras `RF-001`…`RF-120`, recibe `RF-121`). El identificador original del origen consta también en la línea `Origen:`. Los identificadores `REQ-x.y` propios de Axiom y su migración (C-03-e) quedan fuera de esta capacidad.

Supuesto: los identificadores retirados o archivados no se reservan; «no se reutilizan» rige frente al registro actual (especificaciones vivas y cambios activos).
(Previously: todos los requisitos usaban una única serie `REQ-n.*` nueva para el proyecto, `adopt check` rechazaba mezclas y series existentes, y el identificador del cliente se conservaba solo en `Origen:`.)

#### Scenario: Los identificadores del cliente se conservan
- **DADO** un origen con `RF-001`…`RF-120`, `sdd.id_pattern: RF-<NNN>` declarado y un cambio de adopción válido
- **CUANDO** se archiva el cambio
- **ENTONCES** la spec viva identifica cada requisito por su `RF-nnn`
- **Y** la línea `Origen:` conserva el identificador original

#### Scenario: Un requisito nuevo continúa la serie del cliente
- **DADO** la spec viva adoptada con `RF-001`…`RF-120` y `sdd.id_pattern: RF-<NNN>`
- **CUANDO** se integra después un ADDED con un provisional de la serie `RF`
- **ENTONCES** recibe el identificador `RF-121`

#### Scenario: Origen sin identificadores recibe identificadores del patrón por defecto
- **DADO** un origen sin identificadores, un proyecto sin `sdd.id_pattern` y requisitos con provisionales de la serie `40`
- **CUANDO** se archiva el cambio con `axiom sdd archive-integrate`
- **ENTONCES** los requisitos reciben `REQ-40.1`, `REQ-40.2`… por orden de aparición
- **Y** la spec viva no contiene ningún provisional

#### Scenario: Anchos mixtos con `<m>`
- **DADO** `sdd.id_pattern: RF-<m>` y un origen con `RF-1` y `RF-010`
- **CUANDO** se ejecuta `adopt check` sobre un cambio que conserva ambos
- **ENTONCES** no se reporta fallo de formato y los dos identificadores son distintos por clave de identidad

#### Scenario: Una mezcla no admitida bloquea la adopción hasta que la persona normaliza
- **DADO** un origen con `RF-001` y `RNF-001`
- **CUANDO** se ejecutan `adopt scan` y `adopt check`
- **ENTONCES** `scan` informa el conflicto sin plantilla y `check` rechaza el identificador que el patrón declarado no cubre
- **Y** ningún identificador del cliente se reescribe automáticamente

---

### Requirement: Aprobación Humana de la Migración y Conservación de `Origen:` al Archivar (RF-ADP-05)

La compuerta `spec` del proyecto que migra DEBE ser la aprobación humana de la fidelidad de la migración: `adopt check` verifica la forma y la cobertura, pero NO DEBE registrar ni sustituir ninguna decisión de compuerta ni juzgar la fidelidad semántica. El cambio de adopción DEBE archivarse con `axiom sdd archive-integrate` (que ocupa los identificadores definitivos conservados y asigna el siguiente número libre a los provisionales) y la spec viva resultante DEBE conservar las líneas `Origen:` de cada requisito. La guía NO DEBE ordenar `axiom sdd archive-compose --canonical --delta` para un cambio de adopción, porque `archive-compose` rechaza los provisionales.
(Previously: el cambio de adopción se archivaba con `axiom sdd archive-compose --canonical --delta`.)

Supuesto: la spec no hace que `adopt check` condicione el archivado ni la proyección de `axiom sdd status`; la verificación es una orden explícita que la doctrina de la skill de adopción manda ejecutar antes de la compuerta.

#### Scenario: `adopt check` no registra decisiones de compuerta
- **DADO** un cambio de adopción válido
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** el registro de compuertas del cambio no cambia

#### Scenario: El archivado conserva `Origen:`
- **DADO** un cambio de adopción válido con la compuerta `spec` aprobada
- **CUANDO** se archiva con `axiom sdd archive-integrate`
- **ENTONCES** cada requisito de `openspec/specs/<capacidad>/spec.md` conserva su línea `Origen:` byte a byte

#### Scenario: Archivar un cambio con identificadores conservados y provisionales
- **DADO** un cambio de adopción con requisitos `(RF-001)` y otros con provisional, y `sdd.id_pattern: RF-<NNN>`
- **CUANDO** se archiva con `axiom sdd archive-integrate`
- **ENTONCES** los definitivos se conservan y cada provisional recibe el siguiente número libre tras el máximo existente

#### Scenario: La guía no ordena `archive-compose` para la adopción
- **DADO** el texto de la guía de adopción
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** ordena `axiom sdd archive-integrate` y no ordena `archive-compose --canonical --delta` para el cambio de adopción
### Requirement: En Multirepo la Adopción Opera sobre el Repositorio de Especificaciones (RF-ADP-06)

En una topología `multirepo`, `axiom archive adopt scan` y `axiom archive adopt check` DEBEN resolver `workspaceRoot` con el primitivo de la capacidad `planning-home-resolution` y operar sobre el repositorio de especificaciones: el inventario cubre los orígenes de ese repositorio, el cambio de adopción vive en su `openspec/changes/` y `openspec/specs/` es el de ese repositorio. En `monorepo-embedded` operan sobre el repositorio actual.

Supuesto (propuesta §13, P6): `workspaceRoot` es la raíz de operación; los orígenes que vivan en repositorios de código hermanos quedan fuera del inventario.

#### Scenario: Multirepo inventaría el repositorio de especificaciones
- **DADO** un `multirepo` con `specs_repository: repo-specs` y requisitos previos en `repo-specs/docs/`
- **CUANDO** se ejecuta `adopt scan --path <maestra>`
- **ENTONCES** el inventario contiene los orígenes de `repo-specs/docs/`
- **Y** no contiene ficheros de los repositorios de código

#### Scenario: Monorepo embebido
- **DADO** un `monorepo-embedded`
- **CUANDO** se ejecuta `adopt scan`
- **ENTONCES** el inventario cubre el repositorio actual
