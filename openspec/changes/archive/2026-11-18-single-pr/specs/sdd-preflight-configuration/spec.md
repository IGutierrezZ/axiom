# Delta para sdd-preflight-configuration

> Parte del estado vigente: el resto de requisitos (REQ-21.5, REQ-21.6, REQ-26.9, REQ-28.12, REQ-28.13 y REQ-28.19 incluidos) NO cambia. Esta delta retira la estrategia de PR del cuestionario de pre-vuelo (decisión P10): la entrega es `single-pr` sin preguntar, el único grupo requerido es `pace` en `monorepo-embedded` y `multirepo` no tiene ninguno. El grupo `pr-strategy` pasa a ser un grupo heredado, reconocido, tolerado e ignorado, igual que `artifacts`. El bloque derivado pasa de seis a cinco líneas (se retira `Review policy`) y ninguna superficie del pre-vuelo contiene un umbral numérico de entrega. La entrega encadenada (`chained`) se trata en `sdd-orchestrator-assets` y en `organic-agent-trigger-rules`.

## MODIFIED Requirements

### Requirement: La Preferencia de Revisión Independiente se Recoge en una Interacción Propia Tras el Pre-Vuelo, sin Reutilizar sus Marcadores ni sus Grupos (REQ-28.14)

Tras validar los grupos requeridos del pre-vuelo según la topología (REQ-28.15), o tras suprimirse el cuestionario por no requerir ninguno (REQ-28.18), y antes del primer lanzamiento de fase de la sesión, el orquestador DEBE presentar **una única interacción propia** con dos preguntas de selección única: (1) revisión tras el apply de cada rol, con las opciones de token canónico `verify`, `verify-judgment` y `none`; (2) revisión de cobertura tras spec y design, con las opciones `yes` y `no`. Las preguntas tienen los tokens canónicos `after_apply` y `coverage`. Las etiquetas visibles PUEDEN traducirse al idioma de la conversación junto con las descripciones; los tokens canónicos no se traducen y son los que el orquestador cachea. Cada opción DEBE llevar una descripción que declare su efecto y su coste; la de `verify-judgment` DEBE declarar que en modalidad `continuous` incluye las correcciones acotadas del juicio dentro de las raíces de edición autorizadas.

La preferencia NO DEBE presentarse como un grupo más del pre-vuelo ni reutilizar sus marcadores: el texto de la interacción NO DEBE contener el prefijo `Axiom SDD preflight`, ningún marcador nominal del pre-vuelo (`pace`, `pr-strategy`) ni un marcador numerado heredado (`N/3`, `4/…`). Los marcadores nominales del pre-vuelo (REQ-28.16), sus etiquetas y mapeos canónicos, y la corroboración que hace el hook del runtime no cambian por esta interacción; una respuesta con exactamente una opción por cada grupo requerido sigue estableciendo la autoridad del pre-vuelo. La preferencia NO establece autoridad de despacho: solo decide si el orquestador lanza diagnósticos informativos. Ninguna de las dos preguntas ofrece un presupuesto de líneas ni una estrategia de entrega.

En runtimes con interfaz nativa de preguntas se usa esa interfaz; en los demás, el orquestador DEBE presentar el contenido completo en texto (ambas preguntas, todas sus opciones y descripciones) con la sintaxis de respuesta `After apply: <label>; Coverage: <label>`, y detenerse hasta recibir una respuesta válida. La respuesta se acepta solo si cada parte coincide con exactamente una opción presentada de su propia pregunta (insensible a mayúsculas y tras recortar espacios). Una respuesta ausente, ambigua o fuera del dominio NO DEBE sustituirse por un valor por defecto: se vuelve a presentar la interacción completa. La preferencia se cachea en la sesión, sin campo nuevo en `kickoff.yaml` ni en `gates.yaml` y sin observación de memoria persistente (REQ-28.2).
(Previously: el texto declaraba que la política de revisión de 400 líneas «sigue sin preguntarse» y que el pre-vuelo «sigue declarando la política fija», y cacheaba la preferencia «como la estrategia de entrega»; ahora el pre-vuelo no declara ninguna política de revisión ni número, y la entrega es `single-pr` sin pregunta)

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

- **DADO** una respuesta con exactamente una opción para cada grupo requerido por la topología (por ejemplo, `pace` en `monorepo-embedded`)
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

- **DADO** una respuesta del pre-vuelo en la que un grupo lleva una opción de la preferencia (por ejemplo, `verify` como ritmo)
- **CUANDO** se valida
- **ENTONCES** se rechaza como fuera del dominio de ese grupo

#### Scenario: La preferencia se cachea sin escribir en el kickoff

- **DADO** una preferencia respondida
- **CUANDO** se sella el kickoff del cambio
- **ENTONCES** `kickoff.yaml` no contiene la preferencia
- **Y** el orquestador la aplica durante la sesión según REQ-28.2

#### Scenario: La interacción no ofrece presupuesto de líneas ni estrategia de entrega

- **DADO** el pre-vuelo y la interacción de la preferencia
- **CUANDO** se inspecciona su contenido
- **ENTONCES** ninguno ofrece un presupuesto de líneas seleccionable ni una estrategia de entrega seleccionable
- **Y** ninguno contiene un número de líneas como umbral de entrega

#### Scenario: Una sesión nueva vuelve a preguntar la preferencia una vez

- **DADO** un cambio con kickoff ya sellado reanudado en una sesión nueva
- **CUANDO** el orquestador ejecuta el pre-vuelo de esa sesión
- **ENTONCES** recoge la preferencia una vez en su interacción propia, porque es de sesión y no del cambio
- **Y** no repite el cuestionario de sellado del kickoff (REQ-21.4)

#### Scenario: Una sesión sin la preferencia deja su ausencia tratada por REQ-28.2

- **DADO** una sesión cuyo pre-vuelo se estableció sin la preferencia (orquestador anterior a este cambio)
- **CUANDO** concluye un `apply`
- **ENTONCES** se aplica el tratamiento de preferencia ausente de REQ-28.2

---

### Requirement: Los Grupos del Pre-Vuelo Dependen de la Topología y los Valores No Preguntados se Derivan (REQ-28.15)

El cuestionario de pre-vuelo SDD DEBE preguntar solo los grupos que la topología del proyecto (`workspace.topology` de `axiom.yaml`) requiere. El conjunto de grupos requeridos es variable por topología y PUEDE ser vacío (REQ-28.18): este requisito fija la tabla vigente, pero el contrato NO DEBE suponer un número fijo de grupos, de modo que otro cambio pueda añadir o retirar un grupo sin alterar el tratamiento de los demás. El identificador estable de grupo requerido es `pace` (ritmo: `Interactive` → `interactive`, `Automatic` → `auto`); `artifacts` (almacén) y `pr-strategy` (estrategia de PR: `Ask me`, `Single PR`, `Auto`) solo existen como grupos heredados (REQ-28.16). Tabla vigente:

- `multirepo`: ningún grupo. El ritmo NO se pregunta: se deriva `interactive` (REQ-21.5).
- `monorepo-embedded`: solo `pace`.

Ni el almacén de artefactos ni la estrategia de entrega se preguntan en ninguna topología: la política de sesión de almacén es siempre `hybrid` y la entrega es siempre `single-pr` (decisión P10). El bloque derivado que el runtime antepone al prompt de cada fase DEBE tener siempre las mismas **cinco** líneas (cabecera, confirmación del padre, `Pace`, `Artifact store` y `Delivery strategy`) y el valor de cada una DEBE ser el respondido para un grupo requerido o el derivado para uno no requerido: `Pace` es `interactive` en `multirepo`, `Artifact store` es siempre `hybrid` y `Delivery strategy` es siempre `single-pr`. El bloque NO DEBE contener ninguna línea de política de revisión. El texto del cuestionario que se proyecta a los orquestadores es global (el mismo para todo proyecto): DEBE ordenar al agente leer `workspace.topology` de `axiom.yaml` antes de preguntar y presentar únicamente los grupos requeridos por esa topología, en una sola llamada, sin ofrecer estrategia de PR.
(Previously: la tabla era `multirepo` → `pr-strategy` y `monorepo-embedded` → `pace` y `pr-strategy`; el bloque derivado tenía seis líneas, con la política de revisión de 400 líneas, y la estrategia de entrega reflejaba la respuesta del grupo `pr-strategy`)

#### Scenario: En multirepo no se pregunta nada

- **DADO** un proyecto `multirepo`
- **CUANDO** el hook evalúa el despacho de una fase SDD
- **ENTONCES** no exige ninguna respuesta de preguntas
- **Y** el bloque derivado dice `Pace: interactive`, `Artifact store: hybrid` y `Delivery strategy: single-pr`

#### Scenario: En monorepo solo se pregunta el ritmo

- **DADO** un proyecto `monorepo-embedded` y un pre-vuelo respondido con el grupo `pace` (`Automatic`)
- **CUANDO** el hook corrobora el pre-vuelo
- **ENTONCES** el bloque derivado dice `Pace: auto`, `Artifact store: hybrid` y `Delivery strategy: single-pr`

#### Scenario: El bloque derivado tiene cinco líneas y ninguna política de revisión

- **DADO** el bloque derivado que el hook antepone al prompt de una fase
- **CUANDO** se inspecciona
- **ENTONCES** tiene exactamente cinco líneas: cabecera, confirmación del padre, `Pace`, `Artifact store` y `Delivery strategy`
- **Y** no contiene `Review policy` ni ningún número de líneas como umbral

#### Scenario: Ni el almacén ni la estrategia de PR se preguntan en ninguna topología

- **DADO** el texto del cuestionario proyectado a cada runtime
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** no presenta ningún grupo de almacén de artefactos ni de estrategia de PR, ni las opciones `OpenSpec`, `Engram`, `Both`, `Ask me`, `Single PR` o `Auto` como pregunta
- **Y** no contiene el mapeo `Both -> hybrid` ni los mapeos `Ask me`, `Single PR` o `Auto` como respuesta elegible

#### Scenario: El texto proyectado manda leer la topología antes de preguntar

- **DADO** el texto del cuestionario proyectado a un orquestador
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** ordena leer `workspace.topology` de `axiom.yaml` antes de preguntar
- **Y** declara los grupos de `multirepo` (ninguno) y de `monorepo-embedded` (solo `pace`)
- **Y** declara la entrega `single-pr` fija sin preguntarla y no contiene ningún número de líneas como umbral

#### Scenario: La sintaxis de respuesta sin interfaz nativa solo cubre el ritmo

- **DADO** el texto proyectado para un runtime sin interfaz de preguntas nativa
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** la sintaxis de respuesta es `Pace: <label>`
- **Y** no menciona `PR strategy`

#### Scenario: Un conjunto variable de grupos no rompe a los demás

- **DADO** una tabla de topologías a la que se añade o de la que se retira un grupo
- **CUANDO** el hook calcula los grupos requeridos y valida la respuesta
- **ENTONCES** exige exactamente un marcador por cada grupo requerido, sea cual sea su número
- **Y** el resto de grupos conserva su marcador, sus etiquetas y su mapeo canónico

#### Scenario: Un bloque con un almacén distinto de `hybrid` no es válido

- **DADO** un bloque derivado cuya línea `Artifact store` declara `engram`, `none` u `openspec`
- **CUANDO** el hook valida el bloque que va a anteponer
- **ENTONCES** lo considera inválido

#### Scenario: Un bloque con una entrega distinta de `single-pr` no es válido

- **DADO** un bloque derivado cuya línea `Delivery strategy` declara `ask-on-risk`, `auto-chain`, `exception-ok` o `chained`
- **CUANDO** el hook valida el bloque que va a anteponer
- **ENTONCES** lo considera inválido

---

### Requirement: Cada Pregunta del Pre-Vuelo Lleva un Marcador por Nombre de Grupo y se Toleran los Marcadores Numerados Heredados (REQ-28.16)

El marcador de cada pregunta del pre-vuelo DEBE ser nominal: `<prefijo> <id de grupo>:`, con el id estable en minúsculas (`pace` para el único grupo requerido vigente) y el prefijo vigente `Axiom SDD preflight` o el heredado `Gentle AI SDD preflight`, ambos aceptados. Durante la transición, hasta que la persona ejecute `axiom sync`, el hook del runtime DEBE seguir aceptando los marcadores numerados heredados (`N/3`), con el mapeo posicional heredado 1 → `pace`, 2 → `artifacts` y 3 → `pr-strategy`; un cuestionario heredado puede mezclarse con marcadores nominales siempre que cada grupo requerido tenga exactamente un marcador. Las respuestas de los grupos heredados `artifacts` (almacén) y `pr-strategy` (estrategia de PR: `Ask me`, `Single PR`, `Auto`) DEBEN aceptarse e ignorarse: nunca determinan el almacén ni la estrategia de entrega del bloque derivado, y no se valida su contenido. Un grupo requerido con más de un marcador, o con ninguno, DEBE denegarse con una causa que nombre el grupo.
(Previously: `pr-strategy` era un grupo requerido y solo `artifacts` era heredado; ahora ambos son heredados, tolerados e ignorados)

#### Scenario: Marcador nominal aceptado con cualquiera de los dos prefijos

- **DADO** un pre-vuelo cuyas preguntas llevan el marcador nominal `pace` con el prefijo vigente, y otro con el prefijo heredado
- **CUANDO** el hook los corrobora
- **ENTONCES** ambos establecen la autoridad del pre-vuelo

#### Scenario: Los marcadores numerados heredados toleran un cuestionario antiguo de tres preguntas

- **DADO** un proyecto `monorepo-embedded` y una llamada heredada con los marcadores `1/3` (ritmo), `2/3` (almacén) y `3/3` (estrategia de entrega) y sus tres respuestas
- **CUANDO** el hook la corrobora
- **ENTONCES** la autoridad se establece
- **Y** las respuestas de almacén y de estrategia de PR se ignoran y el bloque derivado dice `Artifact store: hybrid` y `Delivery strategy: single-pr`

#### Scenario: Una respuesta heredada `Ask me` o `Auto` no produce una entrega distinta de `single-pr`

- **DADO** un proyecto `monorepo-embedded` y una llamada con el grupo `pace` y el grupo heredado `pr-strategy` respondido `Ask me` (y otra con `Auto`)
- **CUANDO** el hook las corrobora
- **ENTONCES** ambas establecen la autoridad
- **Y** el bloque derivado de ambas dice `Delivery strategy: single-pr`, nunca `ask-on-risk` ni `auto-chain`

#### Scenario: Un cuestionario heredado en multirepo ignora ritmo, almacén y estrategia

- **DADO** un proyecto `multirepo` y una llamada heredada con los tres grupos respondidos, entre ellos el ritmo `Automatic`, el almacén `Engram` y la estrategia `Auto`
- **CUANDO** el hook la corrobora
- **ENTONCES** el despacho se permite
- **Y** el bloque derivado dice `Pace: interactive`, `Artifact store: hybrid` y `Delivery strategy: single-pr`

#### Scenario: Un grupo requerido con el marcador repetido se deniega nombrándolo

- **DADO** un proyecto `monorepo-embedded` y un pre-vuelo en el que el marcador del grupo `pace` aparece en dos preguntas
- **CUANDO** el hook lo evalúa
- **ENTONCES** deniega el despacho
- **Y** la causa nombra el grupo `pace` y que su marcador aparece más de una vez

#### Scenario: Un grupo requerido sin marcador se deniega nombrándolo

- **DADO** un proyecto `monorepo-embedded` y una llamada con solo el marcador heredado `pr-strategy`
- **CUANDO** el hook la evalúa
- **ENTONCES** deniega el despacho
- **Y** la causa nombra el grupo `pace` como el requerido que falta

#### Scenario: La interacción de la preferencia de revisión no se confunde con un grupo

- **DADO** una llamada de la interacción de la preferencia de revisión independiente (REQ-28.14)
- **CUANDO** el hook busca marcadores de grupo
- **ENTONCES** no encuentra ninguno y la ignora

---

### Requirement: El Hook Valida el Significado del Pre-Vuelo según la Topología, no su Presentación (REQ-28.17)

El hook de despacho del runtime DEBE leer la topología con el workspace del directorio de trabajo del payload, calcular los grupos requeridos (REQ-28.15) y exigir exactamente una respuesta válida para cada uno: enlazada por etiqueta de opción y no por posición, dentro del dominio cerrado del grupo y con selección única. DEBE tolerar e ignorar los grupos que la topología no requiere y los heredados (`artifacts` y `pr-strategy`, y por ejemplo un ritmo respondido en `multirepo`), el orden de las preguntas y las diferencias de presentación. Cuando falte o sea inválida la respuesta de un grupo requerido, DEBE denegar el despacho con una causa que nombre el grupo. Mientras haya al menos un grupo requerido se conservan todas las protecciones de autoridad: un agente hijo NO despacha, el texto del modelo NO crea autoridad, el payload queda enlazado a la transcripción, solo una respuesta corroborada con su `tool_result` cuenta y, si hay varias, gana la última válida. Fuera de un proyecto Axiom (sin `axiom.yaml`) el hook NO interviene.
(Previously: el grupo `pr-strategy` era requerido en toda topología; ahora se tolera e ignora como heredado)

#### Scenario: Un grupo heredado o extra se ignora

- **DADO** un proyecto `monorepo-embedded` y una llamada con los grupos `pace` y el heredado `pr-strategy`
- **CUANDO** el hook la corrobora
- **ENTONCES** la autoridad se establece usando solo `pace`
- **Y** la estrategia respondida se ignora

#### Scenario: Falta un grupo requerido y la causa lo nombra

- **DADO** un proyecto `monorepo-embedded` y una llamada de preguntas sin respuesta del grupo `pace`
- **CUANDO** el hook la evalúa
- **ENTONCES** deniega el despacho
- **Y** la causa nombra el grupo `pace`

#### Scenario: El orden de las preguntas y de las opciones no importa

- **DADO** una llamada con las preguntas en orden distinto del canónico y con las opciones de un grupo en otro orden
- **CUANDO** el hook la corrobora
- **ENTONCES** enlaza cada respuesta por etiqueta de opción y establece la autoridad

#### Scenario: Una respuesta fuera del dominio o de selección múltiple se deniega

- **DADO** una respuesta del grupo requerido `pace` con una etiqueta que no pertenece a su dominio, o con más de una opción
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

- **DADO** dos respuestas válidas del pre-vuelo en la misma sesión, con ritmos distintos
- **CUANDO** el hook corrobora el despacho
- **ENTONCES** el bloque derivado usa el ritmo de la última

#### Scenario: Fuera de un proyecto Axiom el hook no interviene

- **DADO** un directorio de trabajo sin `axiom.yaml`
- **CUANDO** se despacha una fase SDD
- **ENTONCES** el hook no deniega ni antepone ningún bloque

---

### Requirement: Sin Grupos Requeridos el Cuestionario se Suprime y el Bloque es Enteramente Derivado (REQ-28.18)

Si la topología no requiere ningún grupo (conjunto vacío, REQ-28.15), el hook DEBE suprimir el cuestionario: DEBE permitir el despacho y anteponer el bloque enteramente derivado (`Pace` y `Artifact store` derivados por topología y `Delivery strategy` con su valor derivado `single-pr`, sin preguntar, decisión P10), sin exigir interacción de preguntas ni autoridad de transcripción, y el texto del orquestador NO DEBE lanzar la herramienta de preguntas. `multirepo` entra siempre por esta ruta. Una respuesta de grupos heredados que llegue en una transcripción se ignora y no altera el bloque. Con al menos un grupo requerido NO DEBE aplicarse ninguna relajación (REQ-28.17). Un prompt que ya traiga el bloque sigue sin duplicarse.
(Previously: la ruta sin grupos solo se alcanzaba con una topología de conjunto vacío; `multirepo` exigía el grupo `pr-strategy`)

#### Scenario: Cero grupos requeridos permite el despacho con el bloque derivado

- **DADO** una topología cuyo conjunto de grupos requeridos es vacío y una transcripción sin ninguna llamada de preguntas
- **CUANDO** el hook evalúa el despacho de una fase SDD
- **ENTONCES** lo permite
- **Y** antepone un bloque con `Pace` y `Artifact store: hybrid` derivados y `Delivery strategy: single-pr`

#### Scenario: Multirepo entra siempre por la ruta sin grupos

- **DADO** un proyecto `multirepo` y una transcripción sin ninguna llamada de preguntas
- **CUANDO** el hook evalúa el despacho de una fase SDD
- **ENTONCES** lo permite sin exigir autoridad de transcripción
- **Y** antepone el bloque derivado con `Pace: interactive`

#### Scenario: El orquestador no lanza la herramienta de preguntas con cero grupos

- **DADO** el texto del cuestionario proyectado
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica que, si la topología no requiere ningún grupo, no se lanza la herramienta de preguntas

#### Scenario: Con un grupo requerido no hay relajación

- **DADO** una topología con al menos un grupo requerido (`monorepo-embedded`) y una transcripción sin llamada de preguntas
- **CUANDO** el hook evalúa el despacho
- **ENTONCES** lo deniega con la causa del grupo que falta

#### Scenario: El prompt que ya trae el bloque no se duplica

- **DADO** el conjunto vacío y un prompt que ya contiene el bloque derivado
- **CUANDO** el hook evalúa el despacho
- **ENTONCES** no antepone un segundo bloque

---

### Requirement: La Doctrina del Orquestador Fija el Almacén `hybrid` y Hace que la Topología Decida el Ritmo (REQ-28.20)

La doctrina de los orquestadores SDD de todos los runtimes (incluidas las reescrituras que hace el instalador para los runtimes de reserva) y el contrato de persistencia compartido DEBEN declarar el almacén de artefactos `hybrid` como fijo: NO DEBEN indicar al orquestador que pregunte el almacén, ni que elija `engram` o `none` según la disponibilidad de Engram. Del mismo modo DEBEN declarar la estrategia de entrega `single-pr` como fija en el pre-vuelo: NO DEBEN indicar al orquestador que pregunte la estrategia de PR. La sección «Execution Mode» NO DEBE decir que por defecto el ritmo es automático: DEBE indicar que lo decide la topología (en `multirepo`, con paradas y reviews por bloque; en `monorepo-embedded`, el ritmo elegido en el pre-vuelo). El texto DEBE ser idéntico en todos los runtimes que lo proyectan. Las fases SDD siguen usando el almacén que devuelve `axiom sdd status` (autoridad nativa del binario): el valor por defecto de `status` sin `artifact_store` declarado sigue siendo `openspec` y NO cambia.
(Previously: solo fijaba el almacén `hybrid` y el ritmo por topología; la estrategia de PR se preguntaba)

#### Scenario: La doctrina no pregunta el almacén ni elige `engram` o `none`

- **DADO** la doctrina proyectada a cada runtime y el contrato de persistencia compartido
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** declaran el almacén `hybrid` como fijo
- **Y** no contienen la instrucción de preguntar el almacén ni la de usar `engram` o `none` por defecto

#### Scenario: La doctrina no pregunta la estrategia de PR

- **DADO** la doctrina proyectada a cada runtime
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** declara la entrega `single-pr` como fija en el pre-vuelo
- **Y** no contiene la instrucción de preguntar la estrategia de PR ni ofrece `ask-on-risk`, `auto-chain` o `exception-ok` como elección del pre-vuelo

#### Scenario: El ritmo por defecto ya no es automático

- **DADO** la sección «Execution Mode» de cada runtime
- **CUANDO** se inspecciona
- **ENTONCES** no contiene la frase que fija por defecto el ritmo automático
- **Y** dice que el ritmo lo decide la topología

#### Scenario: Todos los runtimes proyectan el mismo texto

- **DADO** los goldens de proyección de la doctrina de todos los runtimes
- **CUANDO** se regeneran y se comparan
- **ENTONCES** el texto de almacén, de entrega y de ritmo es el mismo en todos

#### Scenario: El valor por defecto de `status` no cambia

- **DADO** un proyecto sin `artifact_store` declarado en `openspec/config.yaml`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el almacén resuelto sigue siendo `openspec`

---

### Requirement: El Plugin de OpenCode Aplica el Mismo Contrato de Grupos que el Hook (REQ-28.21)

El plugin de OpenCode que corrobora el pre-vuelo (en sus dos copias) DEBE aplicar el mismo contrato que el hook: DEBE leer la topología del proyecto (`workspace.topology` de `axiom.yaml`, con el mismo tratamiento de REQ-28.22) y exigir los mismos grupos requeridos; marcadores nominales y numerados heredados, número variable de grupos, enlace de cada respuesta por etiqueta de opción (no por posición, y sin asignar preguntas sin marcador a un grupo), tolerancia de los grupos heredados `artifacts` y `pr-strategy` (aceptados e ignorados) y bloque derivado con las mismas cinco líneas, `Artifact store: hybrid` y `Delivery strategy: single-pr`, sin línea de política de revisión. Con cero grupos requeridos (siempre en `multirepo`) antepone el bloque derivado sin exigir pregunta (REQ-28.18). La lectura de la topología del plugin DEBE coincidir con la del binario para los mismos `axiom.yaml` (prueba de paridad). Cuando un plugin reciba una nueva revisión, el digest de la revisión que sustituye DEBE quedar registrado para que la actualización reconozca el plugin anterior.
(Previously: el contrato exigía los grupos de la topología con `pr-strategy` requerido y un bloque con la política de revisión; ahora solo `pace` es requerido y el bloque tiene cinco líneas)

#### Scenario: El plugin acepta el marcador nominal y el heredado

- **DADO** una llamada de preguntas con marcadores nominales, y otra con marcadores numerados heredados
- **CUANDO** el plugin las corrobora
- **ENTONCES** ambas establecen el bloque derivado

#### Scenario: El plugin enlaza por etiqueta y no por posición

- **DADO** una llamada con las opciones de un grupo en un orden distinto del canónico
- **CUANDO** el plugin construye el bloque
- **ENTONCES** cada respuesta se asocia a su token por la etiqueta elegida

#### Scenario: El plugin exige los grupos de la topología que lee

- **DADO** un proyecto `multirepo` sin ninguna llamada de preguntas, y otro `monorepo-embedded` con una llamada que solo responde el grupo heredado `pr-strategy`
- **CUANDO** el plugin las corrobora
- **ENTONCES** la de `multirepo` establece el bloque con `Pace: interactive`, `Artifact store: hybrid` y `Delivery strategy: single-pr`
- **Y** la de `monorepo-embedded` se deniega nombrando el grupo `pace`

#### Scenario: El plugin con topología ilegible usa los grupos de monorepo

- **DADO** un `axiom.yaml` cuya topología el plugin no puede leer (valor desconocido o fichero no cargable)
- **CUANDO** corrobora una llamada con solo `pr-strategy`
- **ENTONCES** se deniega nombrando el grupo `pace`, como en `monorepo-embedded`

#### Scenario: La lectura de topología del plugin coincide con la del binario

- **DADO** un conjunto de `axiom.yaml` (estilo de bloque, comillas, comentarios, puntero a otro `axiom.yaml`, valor desconocido y sin fichero)
- **CUANDO** se compara la topología que lee el plugin con la del binario
- **ENTONCES** coinciden en todos los casos

#### Scenario: El plugin ignora los grupos de almacén y de estrategia heredados

- **DADO** una llamada heredada con el grupo de almacén respondido `Engram` y el de estrategia respondido `Auto`
- **CUANDO** el plugin construye el bloque
- **ENTONCES** el bloque dice `Artifact store: hybrid` y `Delivery strategy: single-pr`

#### Scenario: El bloque del plugin no contiene política de revisión

- **DADO** el bloque que construye cada copia del plugin
- **CUANDO** se inspecciona
- **ENTONCES** tiene cinco líneas y no contiene `Review policy` ni ningún número de líneas como umbral

#### Scenario: La revisión anterior del plugin queda registrada

- **DADO** el plugin con una nueva revisión en sus dos copias
- **CUANDO** se ejecutan las pruebas de digests del instalador
- **ENTONCES** el digest de la revisión sustituida figura entre los digests conocidos

---

### Requirement: Topología Efectiva: una Topología No Gobernable se Trata como `monorepo-embedded` y sin `axiom.yaml` no se Aplica Ninguna Regla (REQ-28.22)

El hook de despacho, el plugin de OpenCode y el sellado del kickoff DEBEN usar la misma topología efectiva, la que resuelve el workspace del proyecto. Dentro de un proyecto Axiom (con `axiom.yaml`), si la topología no está declarada, es desconocida o el `axiom.yaml` no se puede cargar, la topología efectiva es `monorepo-embedded` (el conjunto de preguntas más amplio: se pregunta de más, nunca de menos). Sin `axiom.yaml` NO se aplica ninguna regla de topología: el hook y el plugin no intervienen y el sellado se comporta como antes de este cambio (modalidad obligatoria y regla de REQ-28.12 y REQ-28.13). El bloque derivado conserva siempre sus cinco líneas (cabecera, confirmación del padre, `Pace`, `Artifact store` y `Delivery strategy`).
(Previously: el bloque derivado conservaba seis líneas, con la política de revisión)

#### Scenario: Topología desconocida en el hook se trata como monorepo

- **DADO** un `axiom.yaml` con una topología desconocida y una llamada que solo responde el grupo heredado `pr-strategy`
- **CUANDO** el hook la evalúa
- **ENTONCES** deniega el despacho nombrando el grupo `pace`

#### Scenario: Un `axiom.yaml` no cargable en el hook se trata como monorepo

- **DADO** un directorio con `axiom.yaml` que no se puede cargar y una llamada con `pace` respondido
- **CUANDO** el hook la evalúa
- **ENTONCES** establece la autoridad con el grupo `pace`

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

## ADDED Requirements

### Requirement: El Pre-Vuelo y su Bloque Derivado no Contienen Ningún Umbral Numérico de Entrega (REQ-28.23)

El texto del cuestionario de pre-vuelo proyectado a los orquestadores, el bloque derivado que el hook y el plugin de OpenCode (en sus dos copias) anteponen al prompt de cada fase y los comandos `sdd-*` de OpenCode que describen el contenido exigido del pre-vuelo NO DEBEN contener ningún número de líneas como umbral, presupuesto o política de revisión de la entrega, ni la expresión «chained PR strategy and review budget». La entrega del pre-vuelo es siempre `single-pr`: `chained` solo se activa cuando la persona lo pide durante la sesión, fuera del pre-vuelo, y no altera la línea `Delivery strategy` del bloque (la precedencia de un `delivery_strategy: chained` reenviado por el orquestador se fija en `sdd-orchestrator-assets`). Quedan fuera de este requisito, por no ser umbrales de entrega, los literales de migración de prompts antiguos de OpenCode de `internal/components/sdd/inject.go`, el umbral de barrido de los agentes `jd-judge-a` y `jd-judge-b` de judgment-day, las lentes `review-*` de RDD y `internal/reviewtransaction/risk.go`.

#### Scenario: El texto y el bloque del pre-vuelo no contienen un umbral de entrega

- **DADO** el texto del cuestionario proyectado, el bloque derivado del hook y el de cada copia del plugin
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** ninguno contiene `400` como umbral de líneas ni la expresión `Review policy`
- **Y** el bloque contiene `Delivery strategy: single-pr`

#### Scenario: Los comandos de OpenCode no exigen presupuesto de revisión en el pre-vuelo

- **DADO** los nueve comandos `sdd-*` de OpenCode
- **CUANDO** se inspecciona el contenido exigido del pre-vuelo
- **ENTONCES** piden modo de ejecución y almacén de artefactos
- **Y** no exigen estrategia de PR encadenada ni presupuesto de revisión

#### Scenario: Un valor heredado de entrega en un bloque no se acepta como bloque derivado

- **DADO** un prompt que ya trae un bloque con `Delivery strategy: ask-on-risk`
- **CUANDO** el hook evalúa el despacho
- **ENTONCES** no lo trata como un bloque derivado válido ya presente
- **Y** antepone el bloque derivado vigente con `Delivery strategy: single-pr`
