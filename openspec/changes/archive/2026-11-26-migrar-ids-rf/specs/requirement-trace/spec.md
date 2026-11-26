# Delta para Trazabilidad de Requisitos (requirement-trace)

> **Cambio:** `migrar-ids-rf` (C-03-e). **Idioma:** castellano de España; identificadores, códigos y tokens en inglés.
> Esta delta se compone al archivar, cuando `openspec/specs` ya está migrada a `RF-<DOM>-<NN>` y `axiom.yaml` declara ese patrón. Por eso los títulos MODIFIED citan el ID que el requisito tiene tras la migración (`RF-TRC-02` era REQ-48.2 y `RF-TRC-04` era REQ-48.4; el orden de aparición en la spec viva da 01 a 15 en el mismo orden que 48.1 a 48.15) y las referencias cruzadas del texto usan los IDs nuevos. Los demás requisitos de la capacidad no cambian de texto.
> En los escenarios, `«ID-antiguo»` es un identificador heredado `REQ-n.m` presente en el mapa versionado, `«ID-fuera»` uno con esa forma que no figura en él, y `«ID-nuevo»` un identificador válido bajo el patrón del proyecto.

## ADDED Requirements

### Requirement: Mapa Versionado de Identificadores Heredados como Fuente de Equivalencias de la Traza (RF-TRC-16)

El sistema MUST ofrecer un mapa versionado de los identificadores heredados `REQ-n.m` a los identificadores nuevos del proyecto, fuente de las equivalencias que la traza añade a través del punto de extensión de alias (RF-TRC-04). El mapa MUST ser un único fichero YAML en `openspec/id-map.yaml`, con esquema versionado declarado en su campo `schema` (`axiom.id-map/v1`); un cambio incompatible MUST subir la versión. Cada entrada MUST llevar la capacidad (carpeta de `openspec/specs/`), el ID antiguo o su ausencia (un requisito que no tenía ID lleva, en su lugar, el título del requisito) y el ID nuevo.

El mapa MUST cumplir estos invariantes:

1. Es completo respecto de los requisitos vigentes en el momento de la migración: una entrada por cada `### Requirement:` de cada capacidad de `openspec/specs/` (513 entradas, con o sin ID antiguo).
2. Es determinista: regenerarlo sobre las mismas specs y la misma tabla de códigos de dominio MUST producir un fichero idéntico byte a byte. La numeración de los IDs nuevos MUST ser `01`, `02`… por orden de aparición del requisito en su spec.
3. Los IDs nuevos MUST ser únicos en todo el mapa y válidos bajo el patrón efectivo del proyecto.
4. Los IDs antiguos MUST ser únicos por capacidad. Un mismo ID antiguo MAY repetirse en capacidades distintas (`REQ-1.1` figura en siete); la clave de una entrada es el par (capacidad, ID antiguo).
5. Es inmutable una vez publicado: una entrada ya publicada MUST NOT cambiar su ID nuevo ni borrarse. Solo se admiten entradas añadidas por una nueva versión del esquema o por un cambio explícito que lo declare.

El mapa MUST ser de solo lectura para la consulta (RF-TRC-10).

#### Scenario: El mapa cubre todos los requisitos vigentes

- **DADO** las specs de `openspec/specs/` en el momento de la migración
- **CUANDO** se compara el mapa con los títulos `### Requirement:` de cada capacidad
- **ENTONCES** existe exactamente una entrada por requisito, con la capacidad, el ID antiguo (o el título si no tenía) y el ID nuevo
- **Y** el número de entradas es el de requisitos vigentes (513)

#### Scenario: La regeneración es idéntica

- **DADO** las mismas specs y la misma tabla de códigos de dominio
- **CUANDO** se genera el mapa dos veces
- **ENTONCES** los dos ficheros son idénticos byte a byte
- **Y** los IDs nuevos de cada capacidad siguen el orden de aparición de sus requisitos, desde `01`

#### Scenario: Un ID antiguo ambiguo tiene una entrada por capacidad

- **DADO** `REQ-1.1` presente en las capacidades `autoskills` y `workspace-topology`
- **CUANDO** se leen las entradas del mapa
- **ENTONCES** hay una entrada para (`autoskills`, `REQ-1.1`) y otra para (`workspace-topology`, `REQ-1.1`)
- **Y** sus IDs nuevos son distintos entre sí y pertenecen a la serie de su propia capacidad

#### Scenario: Invariantes violados por un mapa

- **DADO** un mapa con dos entradas que comparten ID nuevo, o con dos entradas de la misma capacidad que comparten ID antiguo, o con un ID nuevo no válido bajo el patrón
- **CUANDO** se carga el mapa
- **ENTONCES** se rechaza con el error tipado de mapa inválido (RF-TRC-17)

### Requirement: Mapa Ausente Degrada sin Alias y Mapa Inválido Falla con Error Tipado (RF-TRC-17)

Si `openspec/id-map.yaml` no existe, la consulta MUST funcionar sin los alias del mapa: se comporta como la implementación por defecto del punto de extensión (RF-TRC-04) y no emite error ni advertencia por la ausencia. Si el fichero existe y no es legible, su esquema es desconocido o incumple un invariante (RF-TRC-16), el sistema MUST fallar con un error tipado que nombre el fichero, el motivo (código estable) y la continuación (corregir o restaurar el mapa), con código de salida distinto de 0, sin consultar git. El sistema NUNCA MUST degradar en silencio a «sin alias» ante un mapa que existe pero no es válido.

#### Scenario: Sin mapa, la traza funciona

- **DADO** un proyecto sin `openspec/id-map.yaml`
- **CUANDO** se ejecuta `axiom trace «ID-nuevo»`
- **ENTONCES** la consulta termina con código de salida 0
- **Y** los IDs resueltos son los de las trazas, sin equivalentes heredados
- **Y** `warnings` no menciona la ausencia del mapa

#### Scenario: Mapa con esquema desconocido

- **DADO** un `openspec/id-map.yaml` cuyo `schema` es una versión que el sistema no conoce
- **CUANDO** se ejecuta `axiom trace «ID-nuevo»`
- **ENTONCES** falla con un error tipado que nombra el fichero y el motivo
- **Y** el código de salida es distinto de 0
- **Y** no se ejecuta ningún comando de git

#### Scenario: Mapa corrupto

- **DADO** un `openspec/id-map.yaml` que no es YAML válido
- **CUANDO** se ejecuta `axiom trace «ID-nuevo»`
- **ENTONCES** falla con el mismo error tipado, sin resultado parcial ni degradación a «sin alias»

## MODIFIED Requirements

### Requirement: Validación del ID Contra el Patrón del Proyecto con Error Tipado (RF-TRC-02)

El sistema MUST validar el ID de entrada contra el patrón de identificadores efectivo del proyecto (`axiom.yaml`, RF-RID-01). Un ID que sea válido bajo el patrón MUST aceptarse. Si no lo es, el sistema MUST intentar interpretarlo como identificador provisional válido bajo ese patrón (RF-RID-09). Si tampoco lo es, MUST comprobar si es un identificador heredado `REQ-n.m` presente en el mapa versionado de identificadores heredados (RF-TRC-16): en ese caso MUST aceptarse aunque no sea válido bajo el patrón del proyecto, y se resuelve a sus IDs nuevos (RF-TRC-04). Si no es nada de lo anterior (incluido un `REQ-n.m` que no figura en el mapa, o cualquier `REQ-n.m` cuando el mapa no existe), MUST fallar con un error tipado que nombre el ID recibido, el patrón efectivo y la continuación (escribir un ID con la forma del patrón o un provisional válido), con código de salida distinto de 0, sin consultar git ni escanear ningún fichero.

#### Scenario: ID que no cumple el patrón

- **DADO** el patrón por defecto `REQ-<n>.<m>`
- **CUANDO** se ejecuta `axiom trace RF-USU-01`
- **ENTONCES** falla con un error tipado que nombra `RF-USU-01`, el patrón `REQ-<n>.<m>` y la continuación
- **Y** el código de salida es distinto de 0
- **Y** no se ejecuta ningún comando de git

#### Scenario: El patrón del proyecto manda

- **DADO** un proyecto con el patrón `RF-<DOM>-<NN>`
- **CUANDO** se ejecuta `axiom trace RF-USU-01`
- **ENTONCES** el ID se acepta

#### Scenario: Un ID antiguo del mapa se acepta bajo el patrón RF

- **DADO** un proyecto con el patrón `RF-<DOM>-<NN>` y un mapa versionado que contiene `«ID-antiguo»`
- **CUANDO** se ejecuta `axiom trace «ID-antiguo»`
- **ENTONCES** el ID se acepta sin error de validación
- **Y** la consulta se resuelve a los IDs nuevos del mapa

#### Scenario: Un ID antiguo fuera del mapa sigue fallando

- **DADO** un proyecto con el patrón `RF-<DOM>-<NN>` y un mapa versionado que no contiene `«ID-fuera»`
- **CUANDO** se ejecuta `axiom trace «ID-fuera»`
- **ENTONCES** falla con el error tipado que nombra `«ID-fuera»`, el patrón `RF-<DOM>-<NN>` y la continuación
- **Y** el código de salida es distinto de 0
- **Y** no se ejecuta ningún comando de git

#### Scenario: Un provisional válido se acepta

- **DADO** el patrón por defecto y un provisional `«prov»` con una etiqueta de serie nueva
- **CUANDO** se ejecuta `axiom trace «prov»`
- **ENTONCES** la consulta se ejecuta sin error de validación

### Requirement: Punto de Extensión de Alias (RF-TRC-04)

La resolución de equivalentes MUST pasar por un punto de extensión de alias sustituible, de modo que el mapa de identificadores heredados (RF-TRC-16) añada equivalentes sin modificar la consulta. La implementación por defecto MUST devolver únicamente el propio ID, su pareja provisional/definitivo y los `Previous` de las trazas. El mapa de identificadores heredados MUST estar cableado como resolutor de alias de `axiom trace` cuando existe, con estas reglas:

1. Un ID antiguo MUST resolverse a todos sus IDs nuevos: un ID antiguo repetido en varias capacidades (`REQ-1.1`) devuelve el ID nuevo de cada una de ellas.
2. Un ID nuevo MUST resolverse a su ID antiguo (cuando lo tiene), para encontrar los incrementos archivados que lo citan.
3. La coincidencia de un ID antiguo en los incrementos archivados MUST acotarse por la capacidad: un ID antiguo que aparece en el título de una delta archivada de la capacidad `C` (carpeta `specs/C/`), o en una línea `Cubre:` o una entrada de `ids.yaml` cuyo contexto nombra `C`, equivale solo al ID nuevo de (`C`, ID antiguo). MUST NOT contar como incremento del ID nuevo de otra capacidad que usaba el mismo ID antiguo.
4. Consultar por un ID nuevo MUST traer los incrementos archivados que tocaron su ID antiguo en su capacidad, con los roles de RF-TRC-05, además de los que citan el ID nuevo.

Un alias que no sea válido bajo el patrón del proyecto MUST aceptarse cuando proviene del mapa de identificadores heredados.

#### Scenario: Un alias inyectado se incorpora a la consulta

- **DADO** un resolutor de alias que declara `«ID-b»` equivalente a `«ID»`
- **CUANDO** se consulta `«ID»`
- **ENTONCES** los IDs resueltos incluyen `«ID-b»`
- **Y** los incrementos que tocan `«ID-b»` forman parte del resultado

#### Scenario: Sin resolutor adicional solo cuentan las trazas

- **DADO** la implementación por defecto y un ID sin entrada de traza
- **CUANDO** se consulta
- **ENTONCES** los IDs resueltos son solo el ID consultado

#### Scenario: Un ID antiguo ambiguo resuelve a todos sus IDs nuevos

- **DADO** el mapa con `REQ-1.1` en siete capacidades
- **CUANDO** se consulta `axiom trace REQ-1.1`
- **ENTONCES** los IDs resueltos incluyen el ID nuevo de cada una de las siete capacidades y el propio `REQ-1.1`

#### Scenario: Un ID antiguo no ambiguo resuelve a su ID nuevo

- **DADO** el mapa con `REQ-24.3` en una sola capacidad
- **CUANDO** se consulta `axiom trace REQ-24.3`
- **ENTONCES** los IDs resueltos incluyen el ID nuevo de esa capacidad y `REQ-24.3`

#### Scenario: La coincidencia en archivados se acota por capacidad

- **DADO** una delta archivada de la capacidad `autoskills` que modifica `REQ-1.1` y otra de `workspace-topology` que modifica el mismo `REQ-1.1`
- **CUANDO** se consulta el ID nuevo de `autoskills` para ese requisito
- **ENTONCES** el resultado incluye el incremento de `autoskills`
- **Y** no incluye el de `workspace-topology`

#### Scenario: Consultar por el ID nuevo trae los incrementos del ID antiguo

- **DADO** un cambio archivado cuya delta de la capacidad `C` modifica `«ID-antiguo»` y el mapa que lo asocia a `«ID-nuevo»` en `C`
- **CUANDO** se consulta `«ID-nuevo»`
- **ENTONCES** el resultado incluye ese cambio archivado con el rol `modified`
