# Delta para sdd-preflight-configuration

> Parte del estado vigente: el resto de requisitos (REQ-26.9 incluido) NO cambia. Esta delta hace que la topología del proyecto (`workspace.topology` de `axiom.yaml`) decida qué pregunta el cuestionario de pre-vuelo, el ritmo del cambio y los roles admitidos: `multirepo` sella siempre `checkpointed` y pregunta solo la estrategia de entrega; `monorepo-embedded` pregunta el ritmo y la estrategia de entrega y solo admite el rol `fullstack`. El almacén de artefactos de la sesión es siempre `hybrid` y deja de preguntarse. El conjunto de grupos requeridos es variable por topología y puede ser vacío, de modo que otros cambios puedan añadir o retirar grupos sin alterar los demás.

## RENAMED Requirements

### Requirement: La Preferencia de Revisión Independiente se Recoge en una Interacción Propia Tras el Pre-Vuelo, sin Alterar sus Tres Grupos Canónicos (REQ-28.14) → La Preferencia de Revisión Independiente se Recoge en una Interacción Propia Tras el Pre-Vuelo, sin Reutilizar sus Marcadores ni sus Grupos (REQ-28.14)

(Reason: el pre-vuelo ya no tiene tres grupos canónicos fijos: el conjunto de grupos depende de la topología y puede ser vacío)
(Migration: ninguna; los pines de texto y las referencias a REQ-28.14 se actualizan al nuevo nombre)

## MODIFIED Requirements

### Requirement: Bloqueo de Avance sin Modalidad Sellada (REQ-21.5)

Al entrar en el carril SDD, el orquestador DEBE bloquear la creación de `proposal.md` hasta que se defina la modalidad de avance (`continuo` o `con paradas y reviews por bloque`), y su origen depende de la topología (`workspace.topology` de `axiom.yaml`):

- En `monorepo-embedded` la modalidad DEBE definirse explícitamente: el pre-vuelo pregunta el ritmo de la sesión y el sellado exige `--execution-style` o `--from-session-pace`.
- En `multirepo` la modalidad NO se pregunta: la topología la fija y el cambio se sella siempre `checkpointed`. El sellado sin `--execution-style` ni `--from-session-pace` deriva `checkpointed` sin error, y `continuous` se rechaza (REQ-28.19).

El orquestador DEBE persistir la modalidad de forma sellada para el cambio activo y NO DEBE volver a preguntarla mientras el cambio permanezca activo, salvo lo indicado en REQ-21.4. El orquestador NO DEBE preguntar ni exigir ninguna política de relevos: la entrega del trabajo a otra persona se decide en cada parada con las tres salidas (REQ-26.1 de `sdd-block-review-gates`), no en el pre-vuelo.
(Previously: la modalidad debía definirse explícitamente en cualquier topología; en `multirepo` ahora la fija la topología y no se pregunta)

#### Scenario: Modalidad de avance sellada antes de crear la propuesta en monorepo

- **DADO** que el usuario ha elegido el carril SDD en un proyecto `monorepo-embedded`
- **CUANDO** responde explícitamente el ritmo de la sesión
- **ENTONCES** el orquestador sella la modalidad para el cambio
- **Y** procede a crear `proposal.md`
- **Y** no ha formulado ninguna pregunta sobre política de relevos

#### Scenario: En multirepo la modalidad se sella `checkpointed` sin preguntarla

- **DADO** que el usuario ha elegido el carril SDD en un proyecto `multirepo`
- **CUANDO** el orquestador sella el kickoff sin `--execution-style` ni `--from-session-pace`
- **ENTONCES** el sellado se completa con `execution_style: checkpointed`
- **Y** el orquestador no ha formulado ninguna pregunta sobre el ritmo

#### Scenario: Respuesta ambigua o ausente no crea la propuesta ni asume un valor por defecto en monorepo

- **DADO** un proyecto `monorepo-embedded` en el que el usuario no responde con una modalidad reconocible entre las opciones válidas
- **CUANDO** el orquestador evalúa la respuesta
- **ENTONCES** no crea `proposal.md`
- **Y** vuelve a presentar la pregunta pendiente hasta obtener una respuesta válida, sin asumir un valor por defecto no confirmado

#### Scenario: En monorepo el sellado sin modalidad se rechaza

- **DADO** un proyecto `monorepo-embedded` y un cambio sin `kickoff.yaml`
- **CUANDO** se ejecuta `axiom sdd kickoff seal` sin `--execution-style` ni `--from-session-pace`
- **ENTONCES** el sellado se rechaza por falta de modalidad
- **Y** no se escribe `kickoff.yaml`

#### Scenario: El sellado funciona sin política de relevos

- **DADO** un cambio SDD con la modalidad y los roles definidos
- **CUANDO** se ejecuta `axiom sdd kickoff seal` sin `--handoff-policy`
- **ENTONCES** el sellado se completa y se escribe `kickoff.yaml`


---

### Requirement: Asignación Obligatoria del Rol `fullstack` sin Subdivisión Especializada (REQ-21.6)

Si, al concluir el cuestionario de pre-vuelo, ni el usuario ni el diseño previsto subdividen el trabajo en roles especializados (por ejemplo `core`, `web`, `qa`), el orquestador DEBE asignar obligatoriamente el rol único `fullstack`, con política de compuerta `blocking`, fichero de tareas `tasks.md` y fichero de verificación `verify-report.md`. El orquestador NO DEBE dejar un cambio SDD sin al menos un rol informado en su configuración de kickoff.

Los roles especializados solo existen en `multirepo`. En `monorepo-embedded` el rol de un cambio es siempre `fullstack`: el orquestador NO DEBE proponer ni declarar roles especializados y el sellado rechaza cualquier rol distinto (REQ-28.19). Por tanto, en un monorepo no hay `tasks.<rol>.md` por rol, ni parada `role-apply` por rol, ni ramas por rol.
(Previously: los roles especializados se admitían en cualquier topología; ahora solo en `multirepo`, y en `monorepo-embedded` el rol es siempre `fullstack`)

#### Scenario: Sin roles declarados, se asigna fullstack por defecto

- **DADO** un cuestionario de pre-vuelo en el que el usuario no declara ningún rol especializado, en cualquier topología
- **CUANDO** concluye el cuestionario
- **ENTONCES** el orquestador asigna el rol único `fullstack` con política `blocking`, `tasks.md` y `verify-report.md`
- **Y** el cambio queda con exactamente un rol informado

#### Scenario: En monorepo no se proponen roles especializados

- **DADO** un proyecto `monorepo-embedded` y un diseño previsto que tocaría varias áreas de código
- **CUANDO** el orquestador sella el kickoff
- **ENTONCES** sella el rol único `fullstack`
- **Y** no ha propuesto ni declarado ningún rol especializado

#### Scenario: Un único rol especializado declarado no se combina con fullstack en multirepo

- **DADO** un proyecto `multirepo` y un cuestionario de pre-vuelo en el que el usuario declara exactamente un rol especializado (por ejemplo, únicamente `core`)
- **CUANDO** concluye el cuestionario
- **ENTONCES** el orquestador respeta ese único rol declarado como el rol activo del cambio
- **Y** no asigna adicionalmente el rol `fullstack` junto al rol ya declarado

#### Scenario: Varios roles especializados declarados en multirepo tampoco activan fullstack

- **DADO** un proyecto `multirepo` y un cuestionario de pre-vuelo en el que el usuario declara varios roles especializados (por ejemplo `core`, `web` y `qa`)
- **CUANDO** concluye el cuestionario
- **ENTONCES** el orquestador no asigna el rol `fullstack`
- **Y** cada rol declarado conserva su propia política de compuerta

---

### Requirement: `continuous` Solo se Sella con el Rol `fullstack` (REQ-28.12)

`axiom sdd kickoff seal` con `--execution-style continuous` DEBE rechazarse salvo que el roster sea exactamente `[fullstack]`: un único rol cuya identidad es `fullstack`. La regla compara la **identidad** de los roles y NO su política de compuerta: `continuous` no tiene compuertas, así que `fullstack:blocking`, `fullstack:deferred` y `fullstack:optional` se aceptan por igual. Un roster con roles repetidos (por ejemplo, `fullstack` dos veces) NO es exactamente `[fullstack]` y se rechaza con el mismo mensaje. Un único rol especializado (por ejemplo, solo `backend`) y cualquier roster con varios roles, incluso si uno es `fullstack`, también se rechazan. El rechazo DEBE ocurrir antes de escribir nada, NO DEBE crear ni modificar `kickoff.yaml`, y su mensaje DEBE nombrar las dos salidas: sellar con `--execution-style checkpointed`, o quitar los roles especializados para sellar con el rol único `fullstack`. La regla es una validación de **sellado**: NO DEBE aplicarse al leer ni al validar un `kickoff.yaml` ya existente. Un `kickoff.yaml` existente (sellado o archivado) con `continuous` y un roster distinto de `[fullstack]` DEBE seguir leyéndose sin error y conservar su comportamiento. El retro-sello `seal --infer` (`sealed_by: inferred`) queda exento (REQ-21.4 de `sdd-lane-kickoff`). El estilo `checkpointed` admite cualquier roster válido, incluido `[fullstack]`.

La regla es independiente de la topología y se conserva como red de seguridad de REQ-28.19. En un proyecto con `axiom.yaml` la comprobación por topología se evalúa antes (orden en el sellado: topología, modalidad obligatoria y esta regla) y su mensaje, que nombra la topología, gana: en `multirepo`, `continuous` se rechaza siempre; en `monorepo-embedded` (también cuando la topología no está declarada, es desconocida o el `axiom.yaml` no se puede cargar, REQ-28.22), un roster distinto de `[fullstack]` se rechaza siempre. Por tanto esta regla es inalcanzable por la vía normal; solo se ejercita sin `axiom.yaml` (donde no se aplica ninguna regla de topología) y en las pruebas de la regla de roster. NO se relaja: sigue siendo condición necesaria de `continuous` en cualquier caso.
(Previously: la regla era la única que rechazaba `continuous` con un roster distinto de `[fullstack]`; ahora REQ-28.19 la precede por topología y ella queda como red de seguridad)

#### Scenario: `continuous` con el rol fullstack sella

- **DADO** un cambio sin `kickoff.yaml`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --execution-style continuous --role fullstack:blocking`
- **ENTONCES** el sellado se completa
- **Y** `kickoff.yaml` contiene `execution_style: continuous` y el rol único `fullstack`

#### Scenario: La política de compuerta de fullstack no se comprueba

- **DADO** un cambio sin `kickoff.yaml`
- **CUANDO** se ejecuta el sellado `continuous` con el rol `fullstack:deferred` (o `fullstack:optional`)
- **ENTONCES** el sellado se completa
- **Y** `kickoff.yaml` conserva la política indicada para `fullstack`

#### Scenario: Un roster con fullstack repetido se rechaza

- **DADO** un cambio sin `kickoff.yaml`
- **CUANDO** se ejecuta el sellado `continuous` con `--role fullstack --role fullstack`
- **ENTONCES** se rechaza con el mismo mensaje que el resto de rechazos de la regla
- **Y** no se escribe `kickoff.yaml`

#### Scenario: Un roster con fullstack y otro rol se rechaza

- **DADO** un cambio sin `kickoff.yaml`
- **CUANDO** se ejecuta el sellado `continuous` con los roles `fullstack` y `qa`
- **ENTONCES** se rechaza con el mismo mensaje
- **Y** no se escribe `kickoff.yaml`

#### Scenario: `continuous` con varios roles se rechaza con la salida

- **DADO** un cambio sin `kickoff.yaml`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --execution-style continuous` con los roles `core` y `web`
- **ENTONCES** el sellado se rechaza sin escribir `kickoff.yaml`
- **Y** el mensaje nombra la salida `--execution-style checkpointed` y la de quitar los roles especializados

#### Scenario: `continuous` con un único rol especializado se rechaza

- **DADO** un cambio sin `kickoff.yaml`
- **CUANDO** se ejecuta el sellado `continuous` con el único rol `backend`
- **ENTONCES** se rechaza con el mismo mensaje
- **Y** no se escribe nada

#### Scenario: `continuous` sin ningún rol declarado sella con fullstack

- **DADO** un cambio sin roles declarados por el usuario ni por el diseño previsto
- **CUANDO** se sella con `--execution-style continuous`
- **ENTONCES** se asigna el rol `fullstack` conforme a REQ-21.6 y el sellado se completa

#### Scenario: `checkpointed` admite varios roles y también fullstack

- **DADO** un proyecto `multirepo` y el sellado `checkpointed` con `core` y `web`, y otro con solo `fullstack`
- **CUANDO** se ejecutan
- **ENTONCES** ambos se completan
- **Y** esta regla no limita a `checkpointed`: la limitación de roles de un `monorepo-embedded` es la de REQ-28.19

#### Scenario: Con topología declarada el rechazo por topología precede y nombra la topología

- **DADO** un proyecto `monorepo-embedded` y un cambio sin `kickoff.yaml`
- **CUANDO** se ejecuta el sellado `continuous` con los roles `core` y `web`
- **ENTONCES** se rechaza con el mensaje de REQ-28.19, que nombra la topología `monorepo-embedded`
- **Y** no se escribe `kickoff.yaml`

#### Scenario: Un kickoff existente que incumple la regla se sigue leyendo

- **DADO** un `kickoff.yaml` existente con `execution_style: continuous` y los roles `core` y `web`
- **CUANDO** se carga con la lectura habitual (`ParseKickoff`) y se ejecuta `axiom sdd status`
- **ENTONCES** la carga tiene éxito sin error ni aviso bloqueante
- **Y** el cambio conserva el comportamiento de un cambio `continuous` (sin paradas)

#### Scenario: Un kickoff archivado que incumple la regla no se reescribe

- **DADO** un cambio archivado cuyo `kickoff.yaml` incumple la regla
- **CUANDO** se consulta
- **ENTONCES** se lee igual que antes de este cambio
- **Y** ninguna operación de este cambio lo migra ni lo reescribe


---

### Requirement: `--from-session-pace auto` con Roles Distintos de `fullstack` se Rechaza sin Derivar `checkpointed` (REQ-28.13)

`axiom sdd kickoff seal` con `--from-session-pace auto` deriva hoy `continuous`. Cuando el roster no es exactamente `[fullstack]` (REQ-28.12: se compara la identidad de los roles, no su política de compuerta), DEBE rechazarse con el mismo mensaje y las mismas salidas de REQ-28.12, sin escribir `kickoff.yaml`. NO DEBE derivar `checkpointed` en silencio, porque la modalidad de avance no puede asumirse (REQ-21.5). Con el rol único `fullstack`, o sin roles declarados (se asigna `fullstack`), el comportamiento NO cambia: deriva `continuous`. `--from-session-pace interactive` NO cambia.

La regla es independiente de la topología y se conserva como red de seguridad de REQ-28.19, con la misma precedencia que REQ-28.12: con `axiom.yaml`, el rechazo por topología se evalúa primero y gana (en `multirepo`, `--from-session-pace auto` se rechaza siempre y el ritmo derivado es `interactive`; en `monorepo-embedded`, los roles distintos de `fullstack` se rechazan siempre) y nombra la topología. Esta regla es inalcanzable por la vía normal: solo se ejercita sin `axiom.yaml` y en las pruebas de la regla de roster.
(Previously: la regla era la única que rechazaba `--from-session-pace auto` con roles distintos de `fullstack`; ahora REQ-28.19 la precede por topología)

#### Scenario: Ritmo automático con fullstack deriva continuous

- **DADO** una sesión con ritmo `auto` y un cambio con el rol único `fullstack`
- **CUANDO** se ejecuta `seal --from-session-pace auto`
- **ENTONCES** se sella `continuous` con `fullstack`

#### Scenario: Ritmo automático con varios roles se rechaza

- **DADO** una sesión con ritmo `auto` y un cambio con los roles `core` y `web`
- **CUANDO** se ejecuta `seal --from-session-pace auto`
- **ENTONCES** se rechaza con el mensaje de REQ-28.12
- **Y** no se escribe `kickoff.yaml` ni se deriva `checkpointed`

#### Scenario: Ritmo automático con un único rol especializado se rechaza

- **DADO** una sesión con ritmo `auto` y un cambio con el único rol `backend`
- **CUANDO** se ejecuta `seal --from-session-pace auto`
- **ENTONCES** se rechaza

#### Scenario: Ritmo interactivo no cambia

- **DADO** una sesión con ritmo `interactive`
- **CUANDO** se ejecuta `seal --from-session-pace interactive` con roles especializados
- **ENTONCES** se sella `checkpointed` como antes de este cambio

#### Scenario: Tras el rechazo el usuario puede elegir y repetir

- **DADO** un sellado rechazado por REQ-28.13
- **CUANDO** el usuario elige `checkpointed` y se repite el sellado con los mismos roles
- **ENTONCES** el sellado se completa
- **Y** el orquestador no vuelve a preguntar nada ajeno a esa elección


---

### Requirement: La Preferencia de Revisión Independiente se Recoge en una Interacción Propia Tras el Pre-Vuelo, sin Reutilizar sus Marcadores ni sus Grupos (REQ-28.14)

Tras validar los grupos requeridos del pre-vuelo según la topología (REQ-28.15), o tras suprimirse el cuestionario por no requerir ninguno (REQ-28.18), y antes del primer lanzamiento de fase de la sesión, el orquestador DEBE presentar **una única interacción propia** con dos preguntas de selección única: (1) revisión tras el apply de cada rol, con las opciones de token canónico `verify`, `verify-judgment` y `none`; (2) revisión de cobertura tras spec y design, con las opciones `yes` y `no`. Las preguntas tienen los tokens canónicos `after_apply` y `coverage`. Las etiquetas visibles PUEDEN traducirse al idioma de la conversación junto con las descripciones; los tokens canónicos no se traducen y son los que el orquestador cachea. Cada opción DEBE llevar una descripción que declare su efecto y su coste; la de `verify-judgment` DEBE declarar que en modalidad `continuous` incluye las correcciones acotadas del juicio dentro de las raíces de edición autorizadas.

La preferencia NO DEBE presentarse como un grupo más del pre-vuelo ni reutilizar sus marcadores: el texto de la interacción NO DEBE contener el prefijo `Axiom SDD preflight`, ningún marcador nominal del pre-vuelo (`pace`, `pr-strategy`) ni un marcador numerado heredado (`N/3`, `4/…`). Los marcadores nominales del pre-vuelo (REQ-28.16), sus etiquetas y mapeos canónicos, y la corroboración que hace el hook del runtime no cambian por esta interacción; una respuesta con exactamente una opción por cada grupo requerido sigue estableciendo la autoridad del pre-vuelo. La preferencia NO establece autoridad de despacho: solo decide si el orquestador lanza diagnósticos informativos. La política de revisión de 400 líneas sigue sin preguntarse.

En runtimes con interfaz nativa de preguntas se usa esa interfaz; en los demás, el orquestador DEBE presentar el contenido completo en texto (ambas preguntas, todas sus opciones y descripciones) con la sintaxis de respuesta `After apply: <label>; Coverage: <label>`, y detenerse hasta recibir una respuesta válida. La respuesta se acepta solo si cada parte coincide con exactamente una opción presentada de su propia pregunta (insensible a mayúsculas y tras recortar espacios). Una respuesta ausente, ambigua o fuera del dominio NO DEBE sustituirse por un valor por defecto: se vuelve a presentar la interacción completa. La preferencia se cachea en la sesión como la estrategia de entrega, sin campo nuevo en `kickoff.yaml` ni en `gates.yaml` y sin observación de memoria persistente (REQ-28.2).
(Previously: el título y el texto fijaban «tres grupos canónicos» (ritmo, almacén y estrategia de PR) con los marcadores `1/3`, `2/3` y `3/3`; ahora el pre-vuelo tiene un conjunto de grupos variable por topología con marcador nominal)

#### Scenario: La interacción presenta dos preguntas con sus opciones y descripciones

- **DADO** una sesión SDD cuyo pre-vuelo ya está validado
- **CUANDO** el orquestador presenta la preferencia
- **ENTONCES** la interacción tiene dos preguntas de selección única: `after_apply` con `verify`, `verify-judgment` y `none`, y `coverage` con `yes` y `no`
- **Y** cada opción lleva una descripción de efecto y coste
- **Y** la de `verify-judgment` declara que en `continuous` incluye las correcciones acotadas del juicio

#### Scenario: Las etiquetas se pueden traducir y el token canónico se conserva

- **DADO** una conversación en español y una etiqueta visible traducida para `verify-judgment`
- **CUANDO** el usuario la elige
- **ENTONCES** el orquestador cachea el token canónico `verify-judgment`
- **Y** ninguna etiqueta en un idioma concreto forma parte del dominio canónico de respuesta

#### Scenario: Los grupos requeridos siguen estableciendo la autoridad del pre-vuelo

- **DADO** una respuesta con exactamente una opción para cada grupo requerido por la topología (por ejemplo, `pace` y `pr-strategy` en `monorepo-embedded`)
- **CUANDO** el hook del runtime corrobora el pre-vuelo
- **ENTONCES** la autoridad del pre-vuelo se establece
- **Y** la interacción de la preferencia no ha alterado los marcadores, las etiquetas ni los mapeos de esos grupos

#### Scenario: La interacción de la preferencia no reutiliza los marcadores del pre-vuelo

- **DADO** el texto de la interacción de la preferencia
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** no contiene `Axiom SDD preflight`, ni los marcadores nominales `pace` y `pr-strategy`, ni un marcador numerado heredado como `1/3` o `4/`
- **Y** el hook ignora esa interacción al corroborar el pre-vuelo

#### Scenario: Se presenta después del pre-vuelo y antes de la primera fase

- **DADO** un pre-vuelo validado en una sesión nueva, o suprimido por no requerir ningún grupo (REQ-28.18)
- **CUANDO** el orquestador va a lanzar la primera fase de la sesión
- **ENTONCES** ya ha recogido la preferencia en su interacción propia
- **Y** no ha incluido su pregunta dentro de la llamada agrupada del pre-vuelo

#### Scenario: Un runtime sin interfaz nativa usa la sintaxis de texto

- **DADO** un runtime sin interfaz de preguntas nativa
- **CUANDO** el orquestador presenta la preferencia
- **ENTONCES** muestra ambas preguntas con todas sus opciones y descripciones y la sintaxis `After apply: <label>; Coverage: <label>`
- **Y** se detiene hasta recibir una respuesta válida

#### Scenario: Una respuesta de texto válida se acepta

- **DADO** la respuesta `After apply: verify; Coverage: no` en un runtime sin interfaz nativa
- **CUANDO** el orquestador la valida
- **ENTONCES** cachea `after_apply = verify` y `coverage = no`

#### Scenario: Una respuesta fuera del dominio vuelve a preguntarse

- **DADO** una respuesta en la que una parte no coincide con exactamente una opción de su pregunta, o en la que falta una de las dos partes
- **CUANDO** el orquestador la valida
- **ENTONCES** la rechaza
- **Y** vuelve a presentar la interacción completa sin asumir ningún valor por defecto

#### Scenario: Una opción de la preferencia en un grupo del pre-vuelo se rechaza

- **DADO** una respuesta del pre-vuelo en la que un grupo lleva una opción de la preferencia (por ejemplo, `verify` como estrategia de PR)
- **CUANDO** se valida
- **ENTONCES** se rechaza como fuera del dominio de ese grupo

#### Scenario: La preferencia se cachea sin escribir en el kickoff

- **DADO** una preferencia respondida
- **CUANDO** se sella el kickoff del cambio
- **ENTONCES** `kickoff.yaml` no contiene la preferencia
- **Y** el orquestador la aplica durante la sesión según REQ-28.2

#### Scenario: La política de revisión de 400 líneas no se pregunta

- **DADO** el pre-vuelo y la interacción de la preferencia
- **CUANDO** se inspecciona su contenido
- **ENTONCES** ninguno ofrece un presupuesto de líneas seleccionable
- **Y** el pre-vuelo sigue declarando la política fija

#### Scenario: Una sesión nueva vuelve a preguntar la preferencia una vez

- **DADO** un cambio con kickoff ya sellado reanudado en una sesión nueva
- **CUANDO** el orquestador ejecuta el pre-vuelo de esa sesión
- **ENTONCES** recoge la preferencia una vez en su interacción propia, porque es de sesión y no del cambio
- **Y** no repite el cuestionario de sellado del kickoff (REQ-21.4)

#### Scenario: Una sesión sin la preferencia deja su ausencia tratada por REQ-28.2

- **DADO** una sesión cuyo pre-vuelo se estableció sin la preferencia (orquestador anterior a este cambio)
- **CUANDO** concluye un `apply`
- **ENTONCES** se aplica el tratamiento de preferencia ausente de REQ-28.2

## ADDED Requirements

### Requirement: Los Grupos del Pre-Vuelo Dependen de la Topología y los Valores No Preguntados se Derivan (REQ-28.15)

El cuestionario de pre-vuelo SDD DEBE preguntar solo los grupos que la topología del proyecto (`workspace.topology` de `axiom.yaml`) requiere. El conjunto de grupos requeridos es variable por topología y PUEDE ser vacío (REQ-28.18): este requisito fija la tabla vigente, pero el contrato NO DEBE suponer un número fijo de grupos, de modo que otro cambio pueda añadir o retirar un grupo sin alterar el tratamiento de los demás. Los identificadores estables de grupo son `pace` (ritmo: `Interactive` → `interactive`, `Automatic` → `auto`) y `pr-strategy` (estrategia de entrega: `Ask me` → `ask-on-risk`, `Single PR` → `single-pr`, `Auto` → `auto-chain`); `artifacts` solo existe como grupo heredado (REQ-28.16). Tabla vigente:

- `multirepo`: solo `pr-strategy`. El ritmo NO se pregunta: se deriva `interactive` (REQ-21.5).
- `monorepo-embedded`: `pace` y `pr-strategy`.

El almacén de artefactos NO se pregunta en ninguna topología: la política de sesión es siempre `hybrid`. El bloque derivado que el runtime antepone al prompt de cada fase DEBE tener siempre las mismas líneas (cabecera, confirmación del padre, `Pace`, `Artifact store`, `Delivery strategy` y política de revisión de 400 líneas) y el valor de cada una DEBE ser el respondido para un grupo requerido o el derivado para uno no requerido: `Pace` es `interactive` en `multirepo`, `Artifact store` es siempre `hybrid` y la política de revisión sigue siendo fija. El texto del cuestionario que se proyecta a los orquestadores es global (el mismo para todo proyecto): DEBE ordenar al agente leer `workspace.topology` de `axiom.yaml` antes de preguntar y presentar únicamente los grupos requeridos por esa topología, en una sola llamada.

#### Scenario: En multirepo no se pregunta ritmo ni almacén

- **DADO** un proyecto `multirepo` y un pre-vuelo respondido con solo el grupo `pr-strategy`
- **CUANDO** el hook corrobora el pre-vuelo
- **ENTONCES** la autoridad se establece
- **Y** el bloque derivado dice `Pace: interactive` y `Artifact store: hybrid`
- **Y** la línea de estrategia de entrega refleja la respuesta del grupo `pr-strategy`

#### Scenario: En monorepo se pregunta el ritmo y la estrategia de entrega

- **DADO** un proyecto `monorepo-embedded` y un pre-vuelo respondido con los grupos `pace` (`Automatic`) y `pr-strategy`
- **CUANDO** el hook corrobora el pre-vuelo
- **ENTONCES** el bloque derivado dice `Pace: auto` y `Artifact store: hybrid`

#### Scenario: El almacén no se pregunta en ninguna topología

- **DADO** el texto del cuestionario proyectado a cada runtime
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** no presenta ningún grupo de almacén de artefactos ni las opciones `OpenSpec`, `Engram` o `Both` como pregunta
- **Y** no contiene el mapeo `Both -> hybrid` como respuesta elegible

#### Scenario: El texto proyectado manda leer la topología antes de preguntar

- **DADO** el texto del cuestionario proyectado a un orquestador
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** ordena leer `workspace.topology` de `axiom.yaml` antes de preguntar
- **Y** declara los grupos de `multirepo` (solo `pr-strategy`) y de `monorepo-embedded` (`pace` y `pr-strategy`)
- **Y** conserva la política de revisión fija de 400 líneas sin preguntarla

#### Scenario: Un conjunto variable de grupos no rompe a los demás

- **DADO** una tabla de topologías a la que se añade o de la que se retira un grupo
- **CUANDO** el hook calcula los grupos requeridos y valida la respuesta
- **ENTONCES** exige exactamente un marcador por cada grupo requerido, sea cual sea su número
- **Y** el resto de grupos conserva su marcador, sus etiquetas y su mapeo canónico

#### Scenario: Un bloque con un almacén distinto de `hybrid` no es válido

- **DADO** un bloque derivado cuya línea `Artifact store` declara `engram`, `none` u `openspec`
- **CUANDO** el hook valida el bloque que va a anteponer
- **ENTONCES** lo considera inválido

---

### Requirement: Cada Pregunta del Pre-Vuelo Lleva un Marcador por Nombre de Grupo y se Toleran los Marcadores Numerados Heredados (REQ-28.16)

El marcador de cada pregunta del pre-vuelo DEBE ser nominal: `<prefijo> <id de grupo>:`, con el id estable en minúsculas (`pace`, `pr-strategy`) y el prefijo vigente `Axiom SDD preflight` o el heredado `Gentle AI SDD preflight`, ambos aceptados. Durante la transición, hasta que la persona ejecute `axiom sync`, el hook del runtime DEBE seguir aceptando los marcadores numerados heredados (`N/3`), con el mapeo posicional heredado 1 → `pace`, 2 → `artifacts` y 3 → `pr-strategy`; un cuestionario heredado puede mezclarse con marcadores nominales siempre que cada grupo requerido tenga exactamente un marcador. Una respuesta de almacén heredada (grupo `artifacts`) DEBE aceptarse e ignorarse: nunca determina el almacén del bloque derivado. Un grupo requerido con más de un marcador, o con ninguno, DEBE denegarse con una causa que nombre el grupo.

#### Scenario: Marcador nominal aceptado con cualquiera de los dos prefijos

- **DADO** un pre-vuelo cuyas preguntas llevan el marcador nominal con el prefijo vigente, y otro con el prefijo heredado
- **CUANDO** el hook los corrobora
- **ENTONCES** ambos establecen la autoridad del pre-vuelo

#### Scenario: Los marcadores numerados heredados toleran un cuestionario antiguo de tres preguntas

- **DADO** un proyecto `monorepo-embedded` y una llamada heredada con los marcadores `1/3` (ritmo), `2/3` (almacén) y `3/3` (estrategia de entrega) y sus tres respuestas
- **CUANDO** el hook la corrobora
- **ENTONCES** la autoridad se establece
- **Y** la respuesta de almacén se ignora y el bloque derivado dice `Artifact store: hybrid`

#### Scenario: Un cuestionario heredado en multirepo ignora ritmo y almacén

- **DADO** un proyecto `multirepo` y una llamada heredada con los tres grupos respondidos, entre ellos el ritmo `Automatic` y el almacén `Engram`
- **CUANDO** el hook la corrobora
- **ENTONCES** la autoridad se establece con el grupo `pr-strategy`
- **Y** el bloque derivado dice `Pace: interactive` y `Artifact store: hybrid`

#### Scenario: Un grupo requerido con el marcador repetido se deniega nombrándolo

- **DADO** un pre-vuelo en el que el marcador del grupo `pr-strategy` aparece en dos preguntas
- **CUANDO** el hook lo evalúa
- **ENTONCES** deniega el despacho
- **Y** la causa nombra el grupo `pr-strategy` y que su marcador aparece más de una vez

#### Scenario: Un grupo requerido sin marcador se deniega nombrándolo

- **DADO** un proyecto `monorepo-embedded` y una llamada con solo el marcador de `pr-strategy`
- **CUANDO** el hook la evalúa
- **ENTONCES** deniega el despacho
- **Y** la causa nombra el grupo `pace` como el requerido que falta

#### Scenario: La interacción de la preferencia de revisión no se confunde con un grupo

- **DADO** una llamada de la interacción de la preferencia de revisión independiente (REQ-28.14)
- **CUANDO** el hook busca marcadores de grupo
- **ENTONCES** no encuentra ninguno y la ignora

---

### Requirement: El Hook Valida el Significado del Pre-Vuelo según la Topología, no su Presentación (REQ-28.17)

El hook de despacho del runtime DEBE leer la topología con el workspace del directorio de trabajo del payload, calcular los grupos requeridos (REQ-28.15) y exigir exactamente una respuesta válida para cada uno: enlazada por etiqueta de opción y no por posición, dentro del dominio cerrado del grupo y con selección única. DEBE tolerar e ignorar los grupos que la topología no requiere y los heredados (por ejemplo, un ritmo respondido en `multirepo`), el orden de las preguntas y las diferencias de presentación. Cuando falte o sea inválida la respuesta de un grupo requerido, DEBE denegar el despacho con una causa que nombre el grupo. Mientras haya al menos un grupo requerido se conservan todas las protecciones de autoridad: un agente hijo NO despacha, el texto del modelo NO crea autoridad, el payload queda enlazado a la transcripción, solo una respuesta corroborada con su `tool_result` cuenta y, si hay varias, gana la última válida. Fuera de un proyecto Axiom (sin `axiom.yaml`) el hook NO interviene.

#### Scenario: Un grupo extra se ignora

- **DADO** un proyecto `multirepo` y una llamada con los grupos `pace` y `pr-strategy`
- **CUANDO** el hook la corrobora
- **ENTONCES** la autoridad se establece usando solo `pr-strategy`
- **Y** el ritmo respondido se ignora

#### Scenario: Falta un grupo requerido y la causa lo nombra

- **DADO** un proyecto `monorepo-embedded` y una llamada con solo `pace` respondido
- **CUANDO** el hook la evalúa
- **ENTONCES** deniega el despacho
- **Y** la causa nombra el grupo `pr-strategy`

#### Scenario: El orden de las preguntas y de las opciones no importa

- **DADO** una llamada con las preguntas en orden distinto del canónico y con las opciones de un grupo en otro orden
- **CUANDO** el hook la corrobora
- **ENTONCES** enlaza cada respuesta por etiqueta de opción y establece la autoridad

#### Scenario: Una respuesta fuera del dominio o de selección múltiple se deniega

- **DADO** una respuesta de un grupo requerido con una etiqueta que no pertenece a su dominio, o con más de una opción
- **CUANDO** el hook la evalúa
- **ENTONCES** deniega el despacho nombrando el grupo

#### Scenario: Un agente hijo no despacha

- **DADO** un proyecto con al menos un grupo requerido y un despacho originado en un agente hijo
- **CUANDO** el hook lo evalúa
- **ENTONCES** lo deniega

#### Scenario: El texto del modelo no crea autoridad

- **DADO** un proyecto con al menos un grupo requerido y una transcripción en la que el modelo escribe las respuestas pero no hay `tool_result` de la llamada de preguntas
- **CUANDO** el hook evalúa el despacho
- **ENTONCES** lo deniega

#### Scenario: Un payload sin transcripción enlazada se deniega

- **DADO** un proyecto con al menos un grupo requerido y un payload sin transcripción enlazada
- **CUANDO** el hook evalúa el despacho
- **ENTONCES** lo deniega

#### Scenario: Gana la última respuesta válida

- **DADO** dos respuestas válidas del pre-vuelo en la misma sesión, con estrategias de entrega distintas
- **CUANDO** el hook corrobora el despacho
- **ENTONCES** el bloque derivado usa la última

#### Scenario: Fuera de un proyecto Axiom el hook no interviene

- **DADO** un directorio de trabajo sin `axiom.yaml`
- **CUANDO** se despacha una fase SDD
- **ENTONCES** el hook no deniega ni antepone ningún bloque

---

### Requirement: Sin Grupos Requeridos el Cuestionario se Suprime y el Bloque es Enteramente Derivado (REQ-28.18)

Si la topología no requiere ningún grupo (conjunto vacío, REQ-28.15), el hook DEBE suprimir el cuestionario: DEBE permitir el despacho y anteponer el bloque enteramente derivado (`Pace` y `Artifact store` derivados por topología y `Delivery strategy` con su valor derivado `single-pr`, sin preguntar, decisión P10), sin exigir interacción de preguntas ni autoridad de transcripción, y el texto del orquestador NO DEBE lanzar la herramienta de preguntas. Con al menos un grupo requerido NO DEBE aplicarse ninguna relajación (REQ-28.17). Un prompt que ya traiga el bloque sigue sin duplicarse.

#### Scenario: Cero grupos requeridos permite el despacho con el bloque derivado

- **DADO** una topología cuyo conjunto de grupos requeridos es vacío y una transcripción sin ninguna llamada de preguntas
- **CUANDO** el hook evalúa el despacho de una fase SDD
- **ENTONCES** lo permite
- **Y** antepone un bloque con `Pace` y `Artifact store: hybrid` derivados y `Delivery strategy: single-pr`

#### Scenario: El orquestador no lanza la herramienta de preguntas con cero grupos

- **DADO** el texto del cuestionario proyectado
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica que, si la topología no requiere ningún grupo, no se lanza la herramienta de preguntas

#### Scenario: Con un grupo requerido no hay relajación

- **DADO** una topología con al menos un grupo requerido y una transcripción sin llamada de preguntas
- **CUANDO** el hook evalúa el despacho
- **ENTONCES** lo deniega con la causa del grupo que falta

#### Scenario: El prompt que ya trae el bloque no se duplica

- **DADO** el conjunto vacío y un prompt que ya contiene el bloque derivado
- **CUANDO** el hook evalúa el despacho
- **ENTONCES** no antepone un segundo bloque

---

### Requirement: El Sellado del Kickoff Aplica la Topología al Estilo de Ejecución y a los Roles (REQ-28.19)

`axiom sdd kickoff seal` DEBE comprobar la topología del workspace resuelto antes de escribir nada, y la comprobación NO DEBE crear ni modificar `kickoff.yaml` cuando rechaza:

- `multirepo`: DEBE rechazar `--execution-style continuous` y `--from-session-pace auto` con un mensaje que nombre la topología `multirepo` y la salida (sellar `checkpointed`). Sin `--execution-style` ni `--from-session-pace`, DEBE derivar `checkpointed` sin error. `--execution-style checkpointed` y `--from-session-pace interactive` se aceptan.
- `monorepo-embedded`: DEBE rechazar cualquier `--role` distinto de `fullstack` con un mensaje que nombre la topología `monorepo-embedded` y la salida (quitar el rol, o declarar `multirepo`). Sin `--role`, DEBE sellar el rol único `fullstack`. La modalidad sigue exigiéndose explícita (REQ-21.5) y `continuous` con `fullstack` se acepta (REQ-28.12).

El retro-sellado `seal --infer` queda exento de esta comprobación (REQ-21.4 de `sdd-lane-kickoff`), y un `kickoff.yaml` ya existente (sellado o archivado) NO DEBE revalidarse ni reescribirse por ella. El orden de evaluación del sellado es: (1) la regla de topología, que gana y cuyo mensaje nombra la topología; (2) la modalidad obligatoria (REQ-21.5), que solo se exige donde la topología no la deriva; (3) la regla de REQ-28.12 y REQ-28.13, que se conserva como red de seguridad. El análisis de los argumentos del sellado es una función pura que no consulta la topología: la comprobación se hace después de resolver el workspace.

#### Scenario: En multirepo `continuous` se rechaza nombrando la topología

- **DADO** un proyecto `multirepo` y un cambio sin `kickoff.yaml`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --execution-style continuous`
- **ENTONCES** se rechaza con un mensaje que nombra `multirepo` y la salida `checkpointed`
- **Y** no se escribe `kickoff.yaml`

#### Scenario: En multirepo `--from-session-pace auto` se rechaza nombrando la topología

- **DADO** un proyecto `multirepo` y un cambio sin `kickoff.yaml`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --from-session-pace auto`
- **ENTONCES** se rechaza con un mensaje que nombra `multirepo`
- **Y** no se escribe `kickoff.yaml`

#### Scenario: En multirepo sin estilo explícito se deriva `checkpointed`

- **DADO** un proyecto `multirepo` y un cambio sin `kickoff.yaml`
- **CUANDO** se ejecuta `axiom sdd kickoff seal` sin `--execution-style` ni `--from-session-pace`
- **ENTONCES** el sellado se completa con `execution_style: checkpointed`

#### Scenario: En multirepo `checkpointed` e `interactive` se aceptan con roles especializados

- **DADO** un proyecto `multirepo`
- **CUANDO** se sella con `--execution-style checkpointed` y los roles `core` y `web`, y otro cambio con `--from-session-pace interactive`
- **ENTONCES** ambos sellados se completan con `execution_style: checkpointed`

#### Scenario: En monorepo un rol especializado se rechaza nombrando la topología

- **DADO** un proyecto `monorepo-embedded` y un cambio sin `kickoff.yaml`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --execution-style checkpointed --role core`
- **ENTONCES** se rechaza con un mensaje que nombra `monorepo-embedded` y el rol `core`
- **Y** no se escribe `kickoff.yaml`

#### Scenario: En monorepo `fullstack` con otro rol se rechaza

- **DADO** un proyecto `monorepo-embedded`
- **CUANDO** se ejecuta el sellado con `--role fullstack --role qa`
- **ENTONCES** se rechaza con el mismo mensaje y no se escribe `kickoff.yaml`

#### Scenario: En monorepo sin `--role` se sella `fullstack`

- **DADO** un proyecto `monorepo-embedded` y un cambio sin `kickoff.yaml`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --execution-style checkpointed` sin `--role`
- **ENTONCES** `kickoff.yaml` contiene el rol único `fullstack` con política `blocking`

#### Scenario: En monorepo `continuous` con `fullstack` se acepta

- **DADO** un proyecto `monorepo-embedded`
- **CUANDO** se ejecuta el sellado `continuous` con `--role fullstack`
- **ENTONCES** el sellado se completa

#### Scenario: El retro-sellado queda exento

- **DADO** un cambio preexistente con artefactos y sin kickoff, con `execution_style` `continuous` inferido y roles `core` y `web`, en un proyecto `monorepo-embedded` o `multirepo`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --infer`
- **ENTONCES** el retro-sellado se completa conservando lo inferido
- **Y** no se aplica la comprobación por topología

#### Scenario: Un kickoff existente que incumple la topología se sigue leyendo

- **DADO** un `kickoff.yaml` existente, sellado o archivado, con roles especializados en un proyecto `monorepo-embedded`
- **CUANDO** se carga y se ejecuta `axiom sdd status`
- **ENTONCES** la carga tiene éxito y el cambio conserva su comportamiento
- **Y** ninguna operación de este cambio lo reescribe

---

### Requirement: La Doctrina del Orquestador Fija el Almacén `hybrid` y Hace que la Topología Decida el Ritmo (REQ-28.20)

La doctrina de los orquestadores SDD de todos los runtimes (incluidas las reescrituras que hace el instalador para los runtimes de reserva) y el contrato de persistencia compartido DEBEN declarar el almacén de artefactos `hybrid` como fijo: NO DEBEN indicar al orquestador que pregunte el almacén, ni que elija `engram` o `none` según la disponibilidad de Engram. La sección «Execution Mode» NO DEBE decir que por defecto el ritmo es automático: DEBE indicar que lo decide la topología (en `multirepo`, con paradas y reviews por bloque; en `monorepo-embedded`, el ritmo elegido en el pre-vuelo). El texto DEBE ser idéntico en todos los runtimes que lo proyectan. Las fases SDD siguen usando el almacén que devuelve `axiom sdd status` (autoridad nativa del binario): el valor por defecto de `status` sin `artifact_store` declarado sigue siendo `openspec` y NO cambia.

#### Scenario: La doctrina no pregunta el almacén ni elige `engram` o `none`

- **DADO** la doctrina proyectada a cada runtime y el contrato de persistencia compartido
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** declaran el almacén `hybrid` como fijo
- **Y** no contienen la instrucción de preguntar el almacén ni la de usar `engram` o `none` por defecto

#### Scenario: El ritmo por defecto ya no es automático

- **DADO** la sección «Execution Mode» de cada runtime
- **CUANDO** se inspecciona
- **ENTONCES** no contiene la frase que fija por defecto el ritmo automático
- **Y** dice que el ritmo lo decide la topología

#### Scenario: Todos los runtimes proyectan el mismo texto

- **DADO** los goldens de proyección de la doctrina de todos los runtimes
- **CUANDO** se regeneran y se comparan
- **ENTONCES** el texto de almacén y de ritmo es el mismo en todos

#### Scenario: El valor por defecto de `status` no cambia

- **DADO** un proyecto sin `artifact_store` declarado en `openspec/config.yaml`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el almacén resuelto sigue siendo `openspec`

---

### Requirement: El Plugin de OpenCode Aplica el Mismo Contrato de Grupos que el Hook (REQ-28.21)

El plugin de OpenCode que corrobora el pre-vuelo (en sus dos copias) DEBE aplicar el mismo contrato que el hook: DEBE leer la topología del proyecto (`workspace.topology` de `axiom.yaml`, con el mismo tratamiento de REQ-28.22) y exigir los mismos grupos requeridos; marcadores nominales y numerados heredados, número variable de grupos, enlace de cada respuesta por etiqueta de opción (no por posición, y sin asignar preguntas sin marcador a un grupo), tolerancia del grupo de almacén heredado (aceptado e ignorado) y bloque derivado con las mismas líneas y `Artifact store: hybrid`. Con cero grupos requeridos antepone el bloque derivado sin exigir pregunta (REQ-28.18). La lectura de la topología del plugin DEBE coincidir con la del binario para los mismos `axiom.yaml` (prueba de paridad). Cuando un plugin reciba una nueva revisión, el digest de la revisión que sustituye DEBE quedar registrado para que la actualización reconozca el plugin anterior.

#### Scenario: El plugin acepta el marcador nominal y el heredado

- **DADO** una llamada de preguntas con marcadores nominales, y otra con marcadores numerados heredados
- **CUANDO** el plugin las corrobora
- **ENTONCES** ambas establecen el bloque derivado

#### Scenario: El plugin enlaza por etiqueta y no por posición

- **DADO** una llamada con las opciones de un grupo en un orden distinto del canónico
- **CUANDO** el plugin construye el bloque
- **ENTONCES** cada respuesta se asocia a su token por la etiqueta elegida

#### Scenario: El plugin exige los grupos de la topología que lee

- **DADO** un proyecto `multirepo` y una llamada con solo el grupo `pr-strategy`, y otro `monorepo-embedded` con solo `pr-strategy`
- **CUANDO** el plugin las corrobora
- **ENTONCES** la de `multirepo` establece el bloque con `Pace: interactive` y `Artifact store: hybrid`
- **Y** la de `monorepo-embedded` se deniega nombrando el grupo `pace`

#### Scenario: El plugin con topología ilegible usa los grupos de monorepo

- **DADO** un `axiom.yaml` cuya topología el plugin no puede leer (valor desconocido o fichero no cargable)
- **CUANDO** corrobora una llamada con solo `pr-strategy`
- **ENTONCES** se deniega nombrando el grupo `pace`, como en `monorepo-embedded`

#### Scenario: La lectura de topología del plugin coincide con la del binario

- **DADO** un conjunto de `axiom.yaml` (estilo de bloque, comillas, comentarios, puntero a otro `axiom.yaml`, valor desconocido y sin fichero)
- **CUANDO** se compara la topología que lee el plugin con la del binario
- **ENTONCES** coinciden en todos los casos

#### Scenario: El plugin ignora el grupo de almacén heredado

- **DADO** una llamada heredada con el grupo de almacén respondido `Engram`
- **CUANDO** el plugin construye el bloque
- **ENTONCES** el bloque dice `Artifact store: hybrid`

#### Scenario: La revisión anterior del plugin queda registrada

- **DADO** el plugin con una nueva revisión en sus dos copias
- **CUANDO** se ejecutan las pruebas de digests del instalador
- **ENTONCES** el digest de la revisión sustituida figura entre los digests conocidos

---

### Requirement: Topología Efectiva: una Topología No Gobernable se Trata como `monorepo-embedded` y sin `axiom.yaml` no se Aplica Ninguna Regla (REQ-28.22)

El hook de despacho, el plugin de OpenCode y el sellado del kickoff DEBEN usar la misma topología efectiva, la que resuelve el workspace del proyecto. Dentro de un proyecto Axiom (con `axiom.yaml`), si la topología no está declarada, es desconocida o el `axiom.yaml` no se puede cargar, la topología efectiva es `monorepo-embedded` (el conjunto de preguntas más amplio: se pregunta de más, nunca de menos). Sin `axiom.yaml` NO se aplica ninguna regla de topología: el hook y el plugin no intervienen y el sellado se comporta como antes de este cambio (modalidad obligatoria y regla de REQ-28.12 y REQ-28.13). El bloque derivado conserva siempre sus seis líneas (cabecera, confirmación del padre, `Pace`, `Artifact store`, `Delivery strategy` y política de revisión).

#### Scenario: Topología desconocida en el hook se trata como monorepo

- **DADO** un `axiom.yaml` con una topología desconocida y una llamada con solo `pr-strategy`
- **CUANDO** el hook la evalúa
- **ENTONCES** deniega el despacho nombrando el grupo `pace`

#### Scenario: Un `axiom.yaml` no cargable en el hook se trata como monorepo

- **DADO** un directorio con `axiom.yaml` que no se puede cargar y una llamada con `pace` y `pr-strategy` respondidos
- **CUANDO** el hook la evalúa
- **ENTONCES** establece la autoridad con los dos grupos

#### Scenario: Sin `axiom.yaml` el hook y el plugin no intervienen

- **DADO** un directorio sin `axiom.yaml`
- **CUANDO** se despacha una fase SDD
- **ENTONCES** ni el hook ni el plugin deniegan ni anteponen ningún bloque

#### Scenario: El sellado con topología desconocida aplica las reglas de monorepo

- **DADO** un `axiom.yaml` con topología desconocida
- **CUANDO** se ejecuta el sellado con `--role core`
- **ENTONCES** se rechaza con el mensaje de `monorepo-embedded`

#### Scenario: El sellado sin `axiom.yaml` no aplica la regla de topología

- **DADO** un workspace sin `axiom.yaml` y un sellado sin `--execution-style` ni `--from-session-pace`
- **CUANDO** se ejecuta `axiom sdd kickoff seal`
- **ENTONCES** se rechaza por falta de modalidad (REQ-21.5)

#### Scenario: Sin `axiom.yaml` sigue rigiendo la regla de roster

- **DADO** un workspace sin `axiom.yaml`
- **CUANDO** se ejecuta el sellado `continuous` con los roles `core` y `web`
- **ENTONCES** se rechaza con el mensaje de REQ-28.12 y no se escribe `kickoff.yaml`

#### Scenario: Con `monorepo-embedded` y `continuous` con rol especializado gana la topología

- **DADO** un proyecto `monorepo-embedded`
- **CUANDO** se ejecuta el sellado `continuous` con `--role core`
- **ENTONCES** el mensaje nombra la topología y no es el de REQ-28.12
