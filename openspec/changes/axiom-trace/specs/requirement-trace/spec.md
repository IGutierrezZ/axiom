# Delta para requirement-trace

Esta capacidad es nueva (C-03-d). Define la consulta de solo lectura `axiom trace <ID>`, que responde qué incrementos (cambios SDD archivados y documentos ODD integrados) crearon, modificaron, retiraron, renombraron o cubrieron un requisito, con su tipo, sus tareas y su commit de integración en la línea `--first-parent`.

Notación: los requisitos llevan identificadores provisionales de serie nueva `prov:trace:<n>` (REQ-47.1 de `requirement-identifiers`); el número definitivo se asigna al archivar. En los escenarios, `«ID»` es un identificador válido bajo el patrón del proyecto y `«prov»` un provisional válido. La forma exacta del JSON (nombres de campo salvo los que fija esta spec) la fija el diseño. El mapa `REQ-n.m` a ID nuevo de C-03-e queda fuera: solo se exige el punto de extensión.

## ADDED Requirements

### Requirement: Verbo `axiom trace <ID>` y Sus Opciones (prov:trace:1)

El sistema MUST ofrecer el verbo `axiom trace <ID>` con las opciones `--json`, `--integration-branch <rama>` y `--cwd <dir>`. El ID MUST darse como argumento posicional y es obligatorio. El verbo MUST aparecer en la ayuda global (`axiom --help`) y tener ayuda propia (`axiom trace --help`). Sin ID, o con una opción desconocida, MUST fallar con un mensaje que cite el uso y con código de salida distinto de 0. Sin `--json` MUST escribir texto legible; con `--json` MUST escribir únicamente el documento JSON del requisito prov:trace:9.

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

### Requirement: Validación del ID Contra el Patrón del Proyecto con Error Tipado (prov:trace:2)

El sistema MUST validar el ID de entrada contra el patrón de identificadores efectivo del proyecto (`axiom.yaml`, REQ-44.1). Un ID que sea válido bajo el patrón MUST aceptarse. Si no lo es, el sistema MUST intentar interpretarlo como identificador provisional válido bajo ese patrón (REQ-47.1). Si no es ni lo uno ni lo otro, MUST fallar con un error tipado que nombre el ID recibido, el patrón efectivo y la continuación (escribir un ID con la forma del patrón o un provisional válido), con código de salida distinto de 0, sin consultar git ni escanear ningún fichero.

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
- **Y** `axiom trace REQ-1.1` falla con el error tipado

#### Scenario: Un provisional válido se acepta

- **DADO** el patrón por defecto y un provisional `«prov»` con una etiqueta de serie nueva
- **CUANDO** se ejecuta `axiom trace «prov»`
- **ENTONCES** la consulta se ejecuta sin error de validación

### Requirement: Resolución Entre Provisional y Definitivo por la Traza (prov:trace:3)

El sistema MUST resolver el conjunto de IDs equivalentes al consultado mediante la traza `axiom.requirement-ids/v1` (REQ-47.6): el `ids.yaml` de cada cambio SDD archivado y la sección `## Identificadores asignados` de cada documento ODD. Consultar por el identificador definitivo o por el provisional de una asignación MUST devolver el mismo conjunto de incrementos y el mismo conjunto de IDs resueltos. Los `Previous` de cada entrada de traza MUST contarse como equivalentes. La asignación (provisional y definitivo) MUST ser visible en el incremento que la contiene. Un ID sin entrada de traza se resuelve solo a sí mismo.

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

### Requirement: Punto de Extensión de Alias (prov:trace:4)

La resolución de equivalentes MUST pasar por un punto de extensión de alias sustituible, de modo que una capacidad futura (el mapa `REQ-n.m` a ID nuevo de C-03-e) pueda añadir equivalentes sin modificar la consulta. La implementación por defecto MUST devolver únicamente el propio ID, su pareja provisional/definitivo y los `Previous` de las trazas. Este requisito NO exige implementar ningún mapa de IDs antiguos.

#### Scenario: Un alias inyectado se incorpora a la consulta

- **DADO** un resolutor de alias que declara `«ID-b»` equivalente a `«ID»`
- **CUANDO** se consulta `«ID»`
- **ENTONCES** los IDs resueltos incluyen `«ID-b»`
- **Y** los incrementos que tocan `«ID-b»` forman parte del resultado

#### Scenario: Sin resolutor adicional solo cuentan las trazas

- **DADO** la implementación por defecto y un ID sin entrada de traza
- **CUANDO** se consulta
- **ENTONCES** los IDs resueltos son solo el ID consultado

### Requirement: Incrementos SDD Archivados como Fuente (prov:trace:5)

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

### Requirement: Clasificación de Bug en Cambios SDD (prov:trace:6)

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

### Requirement: Documentos ODD como Fuente (prov:trace:7)

El sistema MUST recorrer los documentos `odd/tasks/*.md` y reconocer un incremento ODD cuando el ID (o un equivalente) aparece en: un título de requisito de la sección `## Spec viva` (ADDED, MODIFIED, REMOVED o RENAMED, con los roles de prov:trace:5); una línea `Cubre:` del documento; o la traza `## Identificadores asignados`. Un documento con la línea `Tipo: bug` MUST clasificarse como `bug`; cualquier otro, como `increment`. Solo MUST contar un documento ODD integrado en la rama de integración: uno que la línea `--first-parent` de esa rama no contiene MUST omitirse del resultado.

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

### Requirement: Commit de Integración en la Línea `--first-parent` (prov:trace:8)

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

### Requirement: Contrato JSON `axiom.trace/v1` y Salida de Texto (prov:trace:9)

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

### Requirement: Solo Lectura (prov:trace:10)

La consulta MUST ser de solo lectura: NUNCA MUST crear, modificar ni borrar ficheros del proyecto, ni crear estado, caché o índice persistente, y MUST recalcular el resultado en cada ejecución. Respecto a git, MUST ejecutar únicamente comandos de lectura y NUNCA MUST ejecutar `fetch`, `checkout`, `commit`, `merge`, `reset`, `push` ni otro comando que cambie el repositorio o la red.

#### Scenario: Ninguna escritura ni git mutante

- **DADO** un repositorio de prueba con incrementos SDD y ODD y un registro de los comandos de git ejecutados
- **CUANDO** se ejecuta `axiom trace «ID»` con y sin `--json`
- **ENTONCES** todos los comandos de git registrados son de lectura
- **Y** el árbol de trabajo y el índice son idénticos antes y después
- **Y** no aparece ningún fichero nuevo en el proyecto

### Requirement: ID Válido sin Apariciones (prov:trace:11)

Un ID válido (o provisional válido) que no aparezca en ningún incremento MUST producir un resultado vacío y código de salida 0: con `--json`, `increments` es una lista vacía; sin `--json`, un texto que indica que no hay apariciones. Esto MUST distinguirse del error de validación del requisito prov:trace:2.

#### Scenario: ID válido sin apariciones

- **DADO** un ID válido bajo el patrón que ningún cambio archivado ni documento ODD cita
- **CUANDO** se ejecuta `axiom trace «ID» --json`
- **ENTONCES** `increments` es una lista vacía
- **Y** el código de salida es 0

### Requirement: Rama de Integración Ausente con Remedio (prov:trace:12)

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

### Requirement: Orden Determinista (prov:trace:13)

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

### Requirement: Topología Multirepo con Rutas Ancladas en su Raíz Git (prov:trace:14)

En `monorepo-embedded` el sistema MUST soportar la consulta completa sobre el único repositorio. En `multirepo` MUST anclar cada ruta (carpeta archivada de specs, documento `odd/tasks/…`) con la raíz git del repositorio que la contiene, y calcular su commit de integración con la historia de ese repositorio y su rama de integración, sin mezclar historias. La ruta de cada incremento en el resultado MUST ser relativa a la raíz git que la contiene. Un repositorio inaccesible o sin rama de integración MUST tratarse con el error de prov:trace:12 o con una advertencia que nombre el repositorio, sin dar un commit de otro repositorio.

#### Scenario: Specs y ODD en repositorios distintos

- **DADO** un `multirepo` cuyo repositorio de specs contiene la carpeta archivada de un cambio y cuya raíz del workspace contiene el documento ODD, ambos sobre `«ID»`
- **CUANDO** se consulta `«ID»`
- **ENTONCES** el commit del incremento SDD procede de la historia del repositorio de specs
- **Y** el commit del incremento ODD procede de la historia del repositorio que contiene `odd/tasks`
- **Y** cada ruta es relativa a su propia raíz git

### Requirement: Criterio de Terminado de Trazabilidad de Punta a Punta (prov:trace:15)

Dado un requisito tocado por un cambio SDD archivado y por una feature ODD, ambos integrados, `axiom trace <ID> --json` MUST devolver ambos incrementos, cada uno con su commit de integración en la línea `--first-parent`. Un bug SDD cuyas tareas cubren ese ID con `Cubre:` MUST aparecer en el mismo resultado con el tipo `bug` y el rol `covered`. Consultar por el provisional o por el definitivo de una asignación MUST dar el mismo resultado (prov:trace:3).

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
