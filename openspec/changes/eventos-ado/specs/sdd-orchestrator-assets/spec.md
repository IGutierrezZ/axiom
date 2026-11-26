# Delta para Activos del Orquestador SDD (sdd-orchestrator-assets)

> **Cambio:** `eventos-ado` (C-07-e). **Idioma:** castellano de España; identificadores, comandos y claves en inglés.
> Doctrina de invocación de los verbos deterministas de `axiom integrations` por el orquestador y las skills, igual en ambos estilos de ejecución. Se aplica a todos los runtimes desde una única fuente (como RF-ORQ-21) y se verifica con las pruebas de paridad y los goldens.

## ADDED Requirements

### Requirement: El Orquestador Invoca los Verbos de Tasks y de Resumen con Tracker Activo (prov:ORQ:1)

Cuando el cambio tiene un vínculo `kind: work_item` y `axiom change start --json` no responde `not_applicable` (hay un plugin de work items activo), la doctrina DEBE ordenar al orquestador invocar los verbos de `axiom integrations` en estos momentos, con el mismo contenido en `checkpointed` y en `continuous` y sin añadir ninguna pregunta de gobierno: al terminar `spec`, `summary push`; al terminar `design`, `tasks create` (con la estimación de la sección `## Estimación por rol` de `design.md`, sin `--estimate` salvo que la persona pida cambiarla); al cerrar cada rol, `time propose` y `time record`; y al terminar `archive`, `summary push` sobre la carpeta archivada. En `checkpointed` DEBE invocarlos tras registrar la compuerta correspondiente (`gate record`), nunca antes ni sustituyéndola; en `continuous` DEBE invocarlos al terminar la fase, sin parada. Sin vínculo de WI o sin plugin activo la doctrina NO DEBE invocar ninguno ni anunciarlo. La doctrina NO DEBE ordenar ninguno de estos verbos en el carril ODD. Un fallo de un verbo NO DEBE deshacer ni repetir la aprobación de la compuerta: en `checkpointed` el orquestador presenta el rechazo tipado con sus resoluciones y la parada ofrece sus tres salidas habituales; en `continuous` no se detiene por el fallo de `summary push`, de `time record` ni de la transición de estado, y los recoge para el resumen final.

#### Scenario: `checkpointed` tras la compuerta
- **DADO** un cambio `checkpointed` con tracker activo y la compuerta `design` recién aprobada
- **CUANDO** el orquestador sigue la doctrina
- **ENTONCES** invoca `axiom integrations tasks create --change <cambio>` después de `gate record`

#### Scenario: `continuous` sin parada
- **DADO** un cambio `continuous` con tracker activo y `design` terminado
- **CUANDO** el orquestador sigue la doctrina
- **ENTONCES** invoca `tasks create` al terminar la fase y encadena la siguiente sin presentar ninguna parada ni pregunta

#### Scenario: Sin tracker activo no se invoca nada
- **DADO** un cambio sin plugin de work items activo
- **CUANDO** el orquestador sigue la doctrina
- **ENTONCES** no invoca ningún verbo `tasks`, `summary` ni `time` y no lo menciona

#### Scenario: Fallo de `tasks create` en `checkpointed`
- **DADO** un `tasks create` que falla con la compuerta `design` aprobada
- **CUANDO** el orquestador presenta la parada
- **ENTONCES** muestra el rechazo con su resolución y la parada ofrece sus tres salidas
- **Y** la aprobación de `design` no se repite ni se deshace

#### Scenario: Fallo de `summary push` en `continuous`
- **DADO** un cambio `continuous` y un `summary push` que falla
- **CUANDO** el orquestador sigue la doctrina
- **ENTONCES** no se detiene y registra el fallo para el resumen final

#### Scenario: Carril ODD
- **DADO** una característica ODD con tracker activo
- **CUANDO** el orquestador sigue la doctrina de ODD
- **ENTONCES** no invoca ninguno de los verbos nuevos

### Requirement: `ensure --role` Antes del Apply con Tracker Activo (prov:ORQ:2)

Con tracker activo y vínculo `kind: work_item`, la doctrina DEBE ordenar al orquestador invocar `axiom branch ensure --change <cambio> --role <rol>` antes de lanzar `apply` de cada rol, también con el único rol `fullstack` y en ambos estilos de ejecución, para que la barrera `task_not_bound` y el registro de `started_at` ocurran siempre. Si `ensure --role` rechaza con `task_not_bound`, el orquestador DEBE NO lanzar `apply`, presentar el rechazo con sus dos resoluciones (`tasks create` y `tasks bind`) y esperar: es un bloqueo reportado a la persona, no una parada de compuerta, y se produce igualmente en `continuous`. Sin tracker activo NO DEBE invocarlo por esta regla.

#### Scenario: Apply del rol único
- **DADO** un cambio `continuous` con rol `fullstack` y tracker activo
- **CUANDO** el orquestador va a lanzar `apply`
- **ENTONCES** invoca antes `axiom branch ensure --change x --role fullstack`

#### Scenario: Bloqueo por Task sin vincular
- **DADO** un `ensure --role` que rechaza con `task_not_bound`
- **CUANDO** el orquestador lo recibe
- **ENTONCES** no lanza `apply` y presenta las resoluciones `axiom integrations tasks create --change x` y `axiom integrations tasks bind --change x --role fullstack=<id>`

#### Scenario: Sin tracker
- **DADO** un cambio sin plugin activo
- **CUANDO** el orquestador va a lanzar `apply`
- **ENTONCES** no añade el `ensure --role` por esta regla

### Requirement: `sdd-design` Propone la Estimación por Rol (prov:ORQ:3)

Con tracker activo, la skill `sdd-design` DEBE escribir en `design.md` la sección fija `## Estimación por rol` con una línea `- <rol>: <horas> h` por cada rol del roster sellado, con horas decimales positivas y una justificación breve fuera de las líneas de datos. En `checkpointed`, la aprobación de la compuerta `design` confirma esas horas porque forman parte de su digest; para cambiarlas se itera el diseño o se pasa `--estimate`, que prevalece. Sin tracker activo la skill NO DEBE escribir la sección.

#### Scenario: Sección presente con tracker activo
- **DADO** un cambio con tracker activo y roster `fullstack`
- **CUANDO** `sdd-design` termina
- **ENTONCES** `design.md` contiene `## Estimación por rol` con una línea `- fullstack: <horas> h`

#### Scenario: Sin tracker
- **DADO** un cambio sin tracker activo
- **CUANDO** `sdd-design` termina
- **ENTONCES** `design.md` no contiene la sección

#### Scenario: Una línea por rol del roster
- **DADO** un roster sellado con dos roles
- **CUANDO** `sdd-design` escribe la sección
- **ENTONCES** hay exactamente una línea por cada uno de los dos roles

### Requirement: `sdd-apply` Propone e Imputa las Horas al Cerrar el Rol (prov:ORQ:4)

Con tracker activo, al cerrar el apply de un rol la doctrina DEBE ordenar `axiom integrations time propose --change <cambio> --role <rol>` y después `time record` con las horas definitivas. En `checkpointed` DEBE presentar la propuesta a la persona, aceptar sus horas (que PUEDEN diferir de la propuesta) y llamar a `time record` solo tras esa confirmación, y solo con la compuerta `role-apply:<rol>` aprobada. En `continuous` DEBE llamar a `time record` con las horas de la propuesta, sin parada y sin pedir confirmación. Un `rev_conflict` NO DEBE reintentarse ni reaplicarse automáticamente: se presenta con los valores releídos y la decisión es de la persona (en `continuous` queda en el resumen final como imputación pendiente). La skill NO DEBE imputar en el carril ODD ni sin tracker activo.

#### Scenario: `checkpointed` con confirmación
- **DADO** un rol cerrado con `role-apply:fullstack` aprobada y una propuesta de 11 horas
- **CUANDO** la persona confirma 9 horas
- **ENTONCES** el orquestador ejecuta `time record --hours 9` y no 11

#### Scenario: `continuous` sin parada
- **DADO** un rol cerrado en `continuous` con una propuesta de 11 horas
- **CUANDO** el orquestador sigue la doctrina
- **ENTONCES** ejecuta `time record --hours 11` sin presentar ninguna parada

#### Scenario: Conflicto de revisión
- **DADO** un `time record` que termina con `rev_conflict`
- **CUANDO** el orquestador lo recibe
- **ENTONCES** no lo reintenta, presenta los valores releídos y deja la decisión a la persona

#### Scenario: Sin tracker o en ODD
- **DADO** un cambio sin tracker activo, o una característica ODD
- **CUANDO** se cierra el rol o la característica
- **ENTONCES** no se propone ni se imputa ninguna hora

### Requirement: Doctrina de los Verbos de Integración en Paridad entre Runtimes y Goldens (prov:ORQ:5)

La doctrina de los requisitos anteriores DEBE tener una única fuente proyectada a todos los runtimes que proyectan el orquestador (junto al texto de `axiom change start` en el enrutado), de modo que el orquestador, `sdd-design`, `sdd-apply` y `sdd-archive` contengan las mismas reglas con las mismas palabras de comando en cada runtime, y las copias espejo y los goldens DEBEN regenerarse con ellas. La doctrina NO DEBE nombrar ni valores de secretos ni rutas del tracker.

#### Scenario: Todos los runtimes contienen la doctrina
- **DADO** la proyección del orquestador a cada runtime
- **CUANDO** se comprueba su texto
- **ENTONCES** todos contienen los comandos `tasks create`, `summary push`, `time propose` y `time record` y la regla de `ensure --role` antes del apply

#### Scenario: Goldens actualizados
- **DADO** el texto de la doctrina cambiado
- **CUANDO** se ejecutan las pruebas de goldens y de paridad de las skills
- **ENTONCES** las copias espejo y los goldens coinciden con la fuente

#### Scenario: Sin secretos en la doctrina
- **DADO** el texto proyectado
- **CUANDO** se escanea
- **ENTONCES** no contiene ningún valor de secreto
