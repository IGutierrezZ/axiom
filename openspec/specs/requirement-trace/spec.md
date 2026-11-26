# Especificación de requirement-trace

## Purpose

Trazabilidad de un requisito de punta a punta (C-03-d). La consulta de solo lectura `axiom trace <ID>` responde qué incrementos lo crearon, modificaron, retiraron, renombraron o cubrieron: cambios SDD archivados y documentos ODD integrados. De cada uno da el tipo (incremento o bug), las tareas y el commit de integración en la línea `--first-parent`. Un identificador que fue provisional se resuelve por la traza de identificadores (RF-RID-14 de `requirement-identifiers`). En los escenarios, `«ID»` es un identificador válido bajo el patrón del proyecto y `«prov»` un provisional válido. El mapa de `REQ-n.m` al identificador nuevo de C-03-e existe en `openspec/id-map.yaml` y `axiom trace` lo usa como fuente de equivalencias cuando está presente (RF-TRC-16); el punto de extensión de alias (RF-TRC-04) permite añadirlo sin modificar la consulta.

## Requirements

### Requirement: Verbo `axiom trace <ID>` y Sus Opciones (RF-TRC-01)

El sistema MUST ofrecer el verbo `axiom trace <ID>` con las opciones `--json`, `--integration-branch <rama>` y `--cwd <dir>`. El ID MUST darse como argumento posicional y es obligatorio. El verbo MUST aparecer en la ayuda global (`axiom --help`) y tener ayuda propia (`axiom trace --help`). Sin ID, o con una opción desconocida, MUST fallar con un mensaje que cite el uso y con código de salida distinto de 0. Sin `--json` MUST escribir texto legible; con `--json` MUST escribir únicamente el documento JSON del requisito RF-TRC-09.

#### Scenario: La ayuda muestra el verbo

- **DADO** el binario de Axiom
- **CUANDO** se ejecuta `axiom --help` y `axiom trace --help`
- **ENTONCES** la ayuda global lista el verbo `trace`
- **Y** la ayuda propia describe el argumento `<ID>` y las opciones `--json`, `--integration-branch` y `--cwd`

#### Scenario: Falta el ID

- **DADO** un proyecto Axiom
- **CUANDO** se ejecuta `axiom trace` sin argumentos
- **ENTONCES** falla con un mensaje de uso
- **Y** el código de salida es distinto de 0
- **Y** no se escribe ningún fichero

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

### Requirement: Resolución Entre Provisional y Definitivo por la Traza (RF-TRC-03)

El sistema MUST resolver el conjunto de IDs equivalentes al consultado mediante la traza `axiom.requirement-ids/v1` (RF-RID-14): el `ids.yaml` de cada cambio SDD archivado y la sección `## Identificadores asignados` de cada documento ODD. Consultar por el identificador definitivo o por el provisional de una asignación MUST devolver el mismo conjunto de incrementos y el mismo conjunto de IDs resueltos. Los `Previous` de cada entrada de traza MUST contarse como equivalentes. La asignación (provisional y definitivo) MUST ser visible en el incremento que la contiene. Un ID sin entrada de traza se resuelve solo a sí mismo.

#### Scenario: Provisional y definitivo dan el mismo resultado

- **DADO** un cambio SDD archivado cuyo `ids.yaml` asigna `«prov»` a `«ID»`
- **CUANDO** se consulta `axiom trace «ID»` y `axiom trace «prov»`
- **ENTONCES** ambas respuestas contienen los mismos incrementos y los mismos IDs resueltos
- **Y** el incremento muestra la asignación `«prov»` a `«ID»`

#### Scenario: Un ID anterior se resuelve al vigente

- **DADO** una entrada de traza con el ID `«ID»` y un `Previous` `«ID-viejo»`
- **CUANDO** se consulta por `«ID-viejo»`
- **ENTONCES** los IDs resueltos incluyen `«ID»` y `«ID-viejo»`
- **Y** el resultado incluye los incrementos que tocaron cualquiera de los dos

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
### Requirement: Incrementos SDD Archivados como Fuente (RF-TRC-05)

El sistema MUST recorrer los cambios archivados `openspec/changes/archive/<fecha>-<cambio>/` del árbol de trabajo y reconocer un incremento SDD cuando el ID (o uno de sus equivalentes) aparece en: el título de un requisito de una delta `specs/*/spec.md` bajo ADDED, MODIFIED, REMOVED o RENAMED (en RENAMED, tanto el nombre origen como el destino cuando los incluyen); una línea `Cubre:` de un fichero `tasks*.md` del cambio; o una entrada de `ids.yaml`. El rol de la aparición MUST ser `created` (ADDED), `modified`, `removed`, `renamed` o `covered` (línea `Cubre:`). Cuando un mismo cambio toca el ID con varios roles, MUST informarse un incremento por rol. Las tareas que citan el ID en `Cubre:` MUST listarse con su fichero y línea. Los cambios activos (no archivados) MUST ignorarse. Un incremento SDD MUST anclarse por su carpeta archivada, no por el nombre del cambio, de modo que dos carpetas con el mismo nombre de cambio no se confundan.

#### Scenario: Roles por sección de la delta

- **DADO** cuatro cambios archivados que tocan `«ID»` respectivamente en ADDED, MODIFIED, REMOVED y RENAMED
- **CUANDO** se consulta `«ID»`
- **ENTONCES** aparecen cuatro incrementos SDD con los roles `created`, `modified`, `removed` y `renamed`

#### Scenario: Cubre con tareas

- **DADO** un cambio archivado cuyo `tasks.md` tiene dos líneas `Cubre:` que citan `«ID»`
- **CUANDO** se consulta `«ID»`
- **ENTONCES** el incremento tiene el rol `covered`
- **Y** lista las dos tareas con su fichero y su línea

#### Scenario: Un cambio activo no cuenta

- **DADO** un cambio en `openspec/changes/<cambio>/` sin archivar que añade `«ID»`
- **CUANDO** se consulta `«ID»`
- **ENTONCES** ese cambio no aparece en el resultado

#### Scenario: Carpetas archivadas con el mismo nombre de cambio

- **DADO** dos carpetas archivadas de fechas distintas con el mismo nombre de cambio, una sola de las cuales cita `«ID»`
- **CUANDO** se consulta `«ID»`
- **ENTONCES** solo aparece la carpeta que lo cita, con su ruta completa

### Requirement: Clasificación de Bug en Cambios SDD (RF-TRC-06)

El sistema MUST clasificar un incremento SDD como `bug` cuando la propuesta (`proposal.md`) del cambio contiene una declaración de bug reconocida por el parseo de la declaración de bug de ISO (`ParseBugDeclaration`: `## Informe de bug` o `## Impacto en la spec`), y como `increment` en caso contrario. La clasificación MUST aplicarse a todos los incrementos del cambio, con cualquier rol. Un cambio de bug cuyo requisito estaba mal especificado, que lleva delta y ninguna de las dos secciones, MUST informarse como `increment`; esta limitación MUST quedar documentada.

#### Scenario: Bug SDD que cubre un requisito

- **DADO** un cambio archivado con `## Informe de bug` en su propuesta y una línea `Cubre:` a `«ID»` en sus tareas
- **CUANDO** se consulta `«ID»`
- **ENTONCES** el incremento aparece con el tipo `bug` y el rol `covered`

#### Scenario: Bug sin impacto en la spec

- **DADO** un cambio archivado cuya propuesta declara `## Impacto en la spec` y cuyas tareas citan `«ID»` en `Cubre:`
- **CUANDO** se consulta `«ID»`
- **ENTONCES** el incremento tiene el tipo `bug`

#### Scenario: Cambio sin declaración de bug

- **DADO** un cambio archivado cuya propuesta no tiene ninguna de las dos secciones
- **CUANDO** se consulta un ID que añade
- **ENTONCES** el incremento tiene el tipo `increment`

### Requirement: Documentos ODD como Fuente (RF-TRC-07)

El sistema MUST recorrer los documentos `odd/tasks/*.md` y reconocer un incremento ODD cuando el ID (o un equivalente) aparece en: un título de requisito de la sección `## Spec viva` (ADDED, MODIFIED, REMOVED o RENAMED, con los roles de RF-TRC-05); una línea `Cubre:` del documento; o la traza `## Identificadores asignados`. Un documento con la línea `Tipo: bug` MUST clasificarse como `bug`; cualquier otro, como `increment`. Solo MUST contar un documento ODD integrado en la rama de integración: uno que la línea `--first-parent` de esa rama no contiene MUST omitirse del resultado.

Un documento ODD anterior a la traza (sin `## Identificadores asignados`) MUST encontrarse por el ID escrito en sus títulos y no MUST producir error. Un documento ODD que no se pueda interpretar con el parseo estricto de documentos de feature MUST caer a una búsqueda textual del ID, MUST añadir una advertencia que nombre el documento y NUNCA MUST abortar la consulta.

#### Scenario: Feature ODD con Spec viva

- **DADO** un documento ODD integrado cuya `## Spec viva` modifica `«ID»`
- **CUANDO** se consulta `«ID»`
- **ENTONCES** aparece un incremento ODD con el rol `modified` y el tipo `increment`

#### Scenario: Bug ODD

- **DADO** un documento ODD integrado con la línea `Tipo: bug` y una línea `Cubre:` a `«ID»`
- **CUANDO** se consulta `«ID»`
- **ENTONCES** el incremento ODD tiene el tipo `bug` y el rol `covered`

#### Scenario: Traza ODD

- **DADO** un documento ODD con `## Identificadores asignados` que asigna `«prov»` a `«ID»`
- **CUANDO** se consulta `«ID»` o `«prov»`
- **ENTONCES** el incremento ODD aparece en ambas respuestas con la asignación visible

#### Scenario: Documento antiguo sin traza

- **DADO** un documento ODD anterior a la traza que cita `«ID»` en el título de su delta
- **CUANDO** se consulta `«ID»`
- **ENTONCES** el documento aparece como incremento
- **Y** el código de salida es 0

#### Scenario: Documento que no parsea

- **DADO** un documento ODD integrado, mal formado para el parseo estricto, que cita `«ID»` en su texto
- **CUANDO** se consulta `«ID»`
- **ENTONCES** la consulta termina con código de salida 0
- **Y** `warnings` contiene una advertencia que nombra ese documento
- **Y** el documento se encuentra por búsqueda textual

#### Scenario: Documento no integrado

- **DADO** un documento ODD presente en el árbol de trabajo pero ausente de la línea `--first-parent` de la rama de integración
- **CUANDO** se consulta un ID que cita
- **ENTONCES** el documento no aparece en el resultado

### Requirement: Commit de Integración en la Línea `--first-parent` (RF-TRC-08)

El sistema MUST asociar a cada incremento su commit de integración, calculado solo sobre la línea `--first-parent` de la rama de integración, con la historia de git leída mediante el puerto de historia existente. Para un incremento SDD, el commit de código y el de archivado MUST derivarse como en `axiom sdd trace`, anclados por la carpeta archivada. Para un incremento ODD, el commit MUST ser el primero de la línea `--first-parent` que incorpora `odd/tasks/<feature>.md`. La asociación MUST funcionar con integración por merge, squash y rebase. Cuando no se encuentra el commit, el incremento MUST informarse con `found: false` y la consulta NUNCA MUST fallar por ello. El resultado MUST documentar que, en ODD, el commit es el que trae el documento y no necesariamente el que añadió su `## Spec viva`.

#### Scenario: Integración por merge, squash y rebase

- **DADO** un cambio SDD archivado y una feature ODD que tocan `«ID»`, integrados en la rama de integración por merge, por squash y por rebase en tres repositorios de prueba
- **CUANDO** se consulta `«ID»` en cada repositorio
- **ENTONCES** ambos incrementos llevan un commit de integración con `found: true`, situado en la línea `--first-parent`

#### Scenario: Commit no encontrado

- **DADO** un incremento cuyo commit no aparece en la línea `--first-parent`
- **CUANDO** se consulta su ID
- **ENTONCES** el incremento figura con `found: false`
- **Y** el código de salida es 0

### Requirement: Contrato JSON `axiom.trace/v1` y Salida de Texto (RF-TRC-09)

Con `--json` el sistema MUST escribir un único documento JSON con `schema` igual a `axiom.trace/v1` y, como mínimo: el ID consultado (`id`), los IDs resueltos (`resolvedIds`), la rama de integración (`integrationBranch`), la lista `increments` y la lista `warnings` (presente y vacía si no hay advertencias; la lista `increments` presente y vacía si no hay apariciones). Cada incremento MUST llevar: el carril (`lane`: `sdd` u `odd`), su ruta relativa al repositorio, el nombre, el tipo (`increment` o `bug`), el rol (`created`, `modified`, `removed`, `renamed` o `covered`), las tareas (fichero y línea), el commit de integración con su indicador `found` y, si existe, la asignación (provisional y definitivo). El esquema es versionado: un cambio incompatible MUST subir la versión. Sin `--json`, el sistema MUST escribir un texto legible que muestre, por incremento, el carril, el nombre, el tipo, el rol, el commit abreviado o su ausencia y las advertencias; el texto no es un contrato estable.

#### Scenario: Forma del documento JSON

- **DADO** un ID tocado por un cambio SDD archivado y una feature ODD
- **CUANDO** se ejecuta `axiom trace «ID» --json`
- **ENTONCES** la salida es un único JSON válido con `schema` `axiom.trace/v1`
- **Y** contiene `id`, `resolvedIds`, `integrationBranch`, `increments` y `warnings`
- **Y** cada incremento contiene `lane`, ruta, tipo, rol, tareas y commit de integración con `found`

#### Scenario: Salida de texto

- **DADO** el mismo ID
- **CUANDO** se ejecuta `axiom trace «ID»` sin `--json`
- **ENTONCES** la salida es texto que menciona ambos incrementos con su carril, tipo y rol
- **Y** no es un documento JSON

### Requirement: Solo Lectura (RF-TRC-10)

La consulta MUST ser de solo lectura: NUNCA MUST crear, modificar ni borrar ficheros del proyecto, ni crear estado, caché o índice persistente, y MUST recalcular el resultado en cada ejecución. Respecto a git, MUST ejecutar únicamente comandos de lectura y NUNCA MUST ejecutar `fetch`, `checkout`, `commit`, `merge`, `reset`, `push` ni otro comando que cambie el repositorio o la red.

#### Scenario: Ninguna escritura ni git mutante

- **DADO** un repositorio de prueba con incrementos SDD y ODD y un registro de los comandos de git ejecutados
- **CUANDO** se ejecuta `axiom trace «ID»` con y sin `--json`
- **ENTONCES** todos los comandos de git registrados son de lectura
- **Y** el árbol de trabajo y el índice son idénticos antes y después
- **Y** no aparece ningún fichero nuevo en el proyecto

### Requirement: ID Válido sin Apariciones (RF-TRC-11)

Un ID válido (o provisional válido) que no aparezca en ningún incremento MUST producir un resultado vacío y código de salida 0: con `--json`, `increments` es una lista vacía; sin `--json`, un texto que indica que no hay apariciones. Esto MUST distinguirse del error de validación del requisito RF-TRC-02.

#### Scenario: ID válido sin apariciones

- **DADO** un ID válido bajo el patrón que ningún cambio archivado ni documento ODD cita
- **CUANDO** se ejecuta `axiom trace «ID» --json`
- **ENTONCES** `increments` es una lista vacía
- **Y** el código de salida es 0

### Requirement: Rama de Integración Ausente con Remedio (RF-TRC-12)

La rama de integración MUST resolverse como en `axiom sdd trace`: la indicada con `--integration-branch` o, si no se da, la que determina esa lógica. Si la rama no existe en el clon, el sistema MUST fallar con un error que nombre la rama y ofrezca el remedio de indicar otra con `--integration-branch`, con código de salida distinto de 0, sin ejecutar `fetch`. Cuando la rama se da con `--integration-branch`, MUST prevalecer sobre la inferida.

#### Scenario: Rama inexistente

- **DADO** un clon sin la rama de integración inferida
- **CUANDO** se ejecuta `axiom trace «ID»`
- **ENTONCES** falla con un error que nombra la rama y cita `--integration-branch` como remedio
- **Y** el código de salida es distinto de 0
- **Y** no se ejecuta `fetch`

#### Scenario: Rama explícita

- **DADO** una rama existente `«rama»` distinta de la inferida
- **CUANDO** se ejecuta `axiom trace «ID» --integration-branch «rama»`
- **ENTONCES** el resultado se calcula sobre la línea `--first-parent` de `«rama»`
- **Y** `integrationBranch` del JSON es `«rama»`

### Requirement: Orden Determinista (RF-TRC-13)

El resultado MUST ordenarse de forma determinista: por la posición del commit de integración en la línea `--first-parent` (del más antiguo al más reciente); los incrementos sin commit encontrado MUST ir al final, y los empates MUST resolverse por una clave estable (carril, ruta, rol). Dos ejecuciones sobre el mismo estado MUST producir una salida idéntica byte a byte. `resolvedIds`, `warnings` y las tareas MUST emitirse también en un orden estable.

#### Scenario: Orden cronológico

- **DADO** un cambio SDD integrado antes que una feature ODD, ambos sobre `«ID»`
- **CUANDO** se consulta `«ID»`
- **ENTONCES** el incremento SDD precede al ODD

#### Scenario: Los no encontrados van al final

- **DADO** un incremento con `found: false` y otro con `found: true`
- **CUANDO** se consulta
- **ENTONCES** el incremento con `found: true` precede al de `found: false`

#### Scenario: Salida reproducible

- **DADO** un repositorio sin cambios entre ejecuciones
- **CUANDO** se ejecuta dos veces `axiom trace «ID» --json`
- **ENTONCES** ambas salidas son idénticas

### Requirement: Topología Multirepo con Rutas Ancladas en su Raíz Git (RF-TRC-14)

En `monorepo-embedded` el sistema MUST soportar la consulta completa sobre el único repositorio. En `multirepo` MUST anclar cada ruta (carpeta archivada de specs, documento `odd/tasks/…`) con la raíz git del repositorio que la contiene, y calcular su commit de integración con la historia de ese repositorio y su rama de integración, sin mezclar historias. La ruta de cada incremento en el resultado MUST ser relativa a la raíz git que la contiene. Un repositorio inaccesible o sin rama de integración MUST tratarse con el error de RF-TRC-12 o con una advertencia que nombre el repositorio, sin dar un commit de otro repositorio.

#### Scenario: Specs y ODD en repositorios distintos

- **DADO** un `multirepo` cuyo repositorio de specs contiene la carpeta archivada de un cambio y cuya raíz del workspace contiene el documento ODD, ambos sobre `«ID»`
- **CUANDO** se consulta `«ID»`
- **ENTONCES** el commit del incremento SDD procede de la historia del repositorio de specs
- **Y** el commit del incremento ODD procede de la historia del repositorio que contiene `odd/tasks`
- **Y** cada ruta es relativa a su propia raíz git

### Requirement: Criterio de Terminado de Trazabilidad de Punta a Punta (RF-TRC-15)

Dado un requisito tocado por un cambio SDD archivado y por una feature ODD, ambos integrados, `axiom trace <ID> --json` MUST devolver ambos incrementos, cada uno con su commit de integración en la línea `--first-parent`. Un bug SDD cuyas tareas cubren ese ID con `Cubre:` MUST aparecer en el mismo resultado con el tipo `bug` y el rol `covered`. Consultar por el provisional o por el definitivo de una asignación MUST dar el mismo resultado (RF-TRC-03).

#### Scenario: SDD y ODD sobre el mismo ID

- **DADO** un requisito `«ID»` añadido por un cambio SDD archivado y modificado por una feature ODD, ambos integrados en la rama de integración
- **CUANDO** se ejecuta `axiom trace «ID» --json`
- **ENTONCES** `increments` contiene un incremento `sdd` con el rol `created` y otro `odd` con el rol `modified`
- **Y** ambos tienen un commit de integración con `found: true`
- **Y** el incremento SDD precede al ODD por su posición en la línea `--first-parent`

#### Scenario: El bug SDD aparece como bug

- **DADO** el escenario anterior y, además, un cambio SDD archivado con `## Informe de bug` cuyas tareas citan `«ID»` en `Cubre:`
- **CUANDO** se ejecuta `axiom trace «ID» --json`
- **ENTONCES** `increments` contiene también un incremento `sdd` con el tipo `bug` y el rol `covered`
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

