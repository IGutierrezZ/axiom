<!-- Especificación Viva generada a partir de '2026-09-22-inc-21-upfront-flow-governance' -->

# Especificación de Requerimientos: Cuestionario Pre-Vuelo de SDD, Modalidad de Avance y Asignación Obligatoria de Roles (INC-21)

## Propósito

Definir de forma rigurosa, ejecutable y verificable los requerimientos funcionales y escenarios BDD para el cuestionario de pre-vuelo que sella, antes de crear la propuesta SDD, la modalidad de avance del cambio (continua o con paradas y reviews por bloque) y la asignación de roles, incluyendo la asignación obligatoria del rol `fullstack` cuando no existe subdivisión de roles especializados.

---

## 1. Capacidad: `sdd-preflight-configuration`

El cuestionario de pre-vuelo sella, antes de crear la propuesta, la modalidad de avance del cambio SDD, su política de relevos y la asignación de roles.

### Requirement: Bloqueo de Avance sin Modalidad Sellada (REQ-21.5)

Al entrar en el carril SDD, el orquestador DEBE bloquear la creación de `proposal.md` hasta que se defina explícitamente la modalidad de avance (`continuo` o `con paradas y reviews por bloque`). El orquestador DEBE persistir esa decisión de forma sellada para el cambio activo y NO DEBE volver a preguntarla mientras el cambio permanezca activo, salvo lo indicado en REQ-21.4. El orquestador NO DEBE preguntar ni exigir ninguna política de relevos: la entrega del trabajo a otra persona se decide en cada parada con las tres salidas (REQ-26.1 de `sdd-block-review-gates`), no en el pre-vuelo.
(Previously: «Bloqueo de Avance sin Modalidad y Política de Relevos Selladas»: el orquestador exigía además sellar la política de relevos (`sin handoffs intermedios` o `con handoff.md en cada parada`))

#### Scenario: Modalidad de avance sellada antes de crear la propuesta

- **DADO** que el usuario ha elegido el carril SDD
- **CUANDO** responde explícitamente la modalidad de avance
- **ENTONCES** el orquestador sella la decisión para el cambio
- **Y** procede a crear `proposal.md`
- **Y** no ha formulado ninguna pregunta sobre política de relevos

#### Scenario: Respuesta ambigua o ausente no crea la propuesta ni asume un valor por defecto

- **DADO** que el usuario no responde con una modalidad reconocible entre las opciones válidas
- **CUANDO** el orquestador evalúa la respuesta
- **ENTONCES** no crea `proposal.md`
- **Y** vuelve a presentar la pregunta pendiente hasta obtener una respuesta válida, sin asumir un valor por defecto no confirmado

#### Scenario: El sellado funciona sin política de relevos

- **DADO** un cambio SDD con la modalidad y los roles definidos
- **CUANDO** se ejecuta `axiom sdd kickoff seal` sin `--handoff-policy`
- **ENTONCES** el sellado se completa y se escribe `kickoff.yaml`

### Requirement: Asignación Obligatoria del Rol `fullstack` sin Subdivisión Especializada (REQ-21.6)

Si, al concluir el cuestionario de pre-vuelo, ni el usuario ni el diseño previsto subdividen el trabajo en roles especializados (por ejemplo `core`, `web`, `qa`), el orquestador DEBE asignar obligatoriamente el rol único `fullstack`, con política de compuerta `blocking`, fichero de tareas `tasks.md` y fichero de verificación `verify-report.md`. El orquestador NO DEBE dejar un cambio SDD sin al menos un rol informado en su configuración de kickoff.

#### Scenario: Sin roles declarados, se asigna fullstack por defecto

- **DADO** un cuestionario de pre-vuelo en el que el usuario no declara ningún rol especializado
- **CUANDO** concluye el cuestionario
- **ENTONCES** el orquestador asigna el rol único `fullstack` con política `blocking`, `tasks.md` y `verify-report.md`
- **Y** el cambio queda con exactamente un rol informado

#### Scenario: Un único rol especializado declarado no se combina con fullstack

- **DADO** un cuestionario de pre-vuelo en el que el usuario declara exactamente un rol especializado (por ejemplo, únicamente `core`)
- **CUANDO** concluye el cuestionario
- **ENTONCES** el orquestador respeta ese único rol declarado como el rol activo del cambio
- **Y** no asigna adicionalmente el rol `fullstack` junto al rol ya declarado

#### Scenario: Varios roles especializados declarados tampoco activan fullstack

- **DADO** un cuestionario de pre-vuelo en el que el usuario declara varios roles especializados (por ejemplo `core`, `web` y `qa`)
- **CUANDO** concluye el cuestionario
- **ENTONCES** el orquestador no asigna el rol `fullstack`
- **Y** cada rol declarado conserva su propia política de compuerta
### Requirement: `handoff_policy` Deprecado: se Acepta, se Ignora y se Fija a `none` (REQ-26.9)

El campo `handoff_policy` de `kickoff.yaml` y el flag `--handoff-policy` de `axiom sdd kickoff seal` quedan deprecados y SIN efecto:

- `axiom sdd kickoff seal` NO DEBE exigir `--handoff-policy`. Si se pasa con `none` o `per_checkpoint`, DEBE aceptarse sin error e ignorarse: su valor NO DEBE llegar a `kickoff.yaml`. Un valor fuera de esos dos (supuesto A-6) sigue rechazándose.
- Un `kickoff.yaml` nuevo, del sellado o del retro-sello `seal --infer`, DEBE contener `handoff_policy: none`, con independencia del flag. Es un valor de compatibilidad: los binarios anteriores a este cambio rechazan un kickoff sin la clave, y con `none` su `verify` no consulta `handoff.md`. La retirada de la clave queda para un cambio posterior.
- Un `kickoff.yaml` existente con `handoff_policy: none` o `per_checkpoint` DEBE leerse y validarse sin error; un `kickoff.yaml` sin la clave, también (la validación acepta el valor vacío).
- Ningún comportamiento DEBE depender del valor del campo: en particular, `per_checkpoint` NO DEBE hacer que `verify` consulte `handoff.md` ni que se genere ningún relevo intermedio.
- El texto de `kickoff seal` y `kickoff show` NO DEBE mencionar la política de relevos. El JSON de `kickoff seal`, `kickoff show` y `axiom sdd status` (`governance.kickoff`) PUEDE emitir el valor leído y NO DEBE emitir el campo cuando está vacío.
- El texto de ayuda y los mensajes de error NO DEBEN sugerir `--handoff-policy`.

Supuesto A-6 (a confirmar en design): un valor fuera de `none|per_checkpoint` se rechaza como hoy, porque los orquestadores sincronizados solo pasan esos dos y el rechazo protege de erratas sin riesgo de compatibilidad.

#### Scenario: Sellar sin la política de relevos funciona

- **DADO** un cambio SDD sin `kickoff.yaml`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --execution-style checkpointed` con los roles y sin `--handoff-policy`
- **ENTONCES** el sellado se completa
- **Y** `kickoff.yaml` contiene `handoff_policy: none`

#### Scenario: Un orquestador antiguo que pasa el flag sigue funcionando

- **DADO** un orquestador con instrucciones sincronizadas antes de este cambio
- **CUANDO** ejecuta `axiom sdd kickoff seal ... --handoff-policy per_checkpoint`
- **ENTONCES** el sellado se completa sin error
- **Y** `kickoff.yaml` contiene `handoff_policy: none`, no `per_checkpoint`, y el flag no tiene ningún efecto

#### Scenario: Un binario anterior lee un kickoff nuevo

- **DADO** un `kickoff.yaml` sellado por esta versión, con o sin `seal --infer`
- **CUANDO** un binario anterior a este cambio lo carga y lo valida
- **ENTONCES** la validación tiene éxito, porque la clave está presente con un valor que ese binario reconoce
- **Y** su `verify` no consulta `handoff.md`, porque el valor es `none`

#### Scenario: Un valor desconocido del flag se rechaza

- **DADO** una ejecución de `axiom sdd kickoff seal` con `--handoff-policy otra`
- **CUANDO** se validan los argumentos
- **ENTONCES** se rechaza como valor no válido y no se escribe `kickoff.yaml`

#### Scenario: Un kickoff antiguo con la clave se lee sin error

- **DADO** un `kickoff.yaml` existente con `handoff_policy: per_checkpoint`
- **CUANDO** se carga y se valida
- **ENTONCES** la carga tiene éxito
- **Y** el valor no altera ninguna proyección de `axiom sdd status`

#### Scenario: Un kickoff sin la clave valida

- **DADO** un `kickoff.yaml` sin `handoff_policy`
- **CUANDO** se valida
- **ENTONCES** la validación acepta el valor vacío

#### Scenario: per_checkpoint no hace que verify consulte handoff.md

- **DADO** un cambio de un solo rol sellado antiguamente con `handoff_policy: per_checkpoint` y sin `handoff.md`
- **CUANDO** se evalúa la dependencia `verify`
- **ENTONCES** es idéntica a la de un cambio sellado con `none`

#### Scenario: La salida de kickoff y status no emite el campo vacío

- **DADO** un `kickoff.yaml` sin la clave `handoff_policy`
- **CUANDO** se ejecuta `axiom sdd kickoff show --json` y `axiom sdd status --json`
- **ENTONCES** ni `handoffPolicy` ni su equivalente aparecen en el JSON

#### Scenario: El texto de kickoff no menciona la política de relevos

- **DADO** un cambio sellado por esta versión, con `handoff_policy: none`
- **CUANDO** se ejecuta `axiom sdd kickoff seal` o `axiom sdd kickoff show` sin `--json`
- **ENTONCES** el texto no menciona la política de relevos

#### Scenario: Ningún asset nombra el flag como obligatorio

- **DADO** el conjunto de assets del producto (orquestadores, contrato de status, convención de openspec, skills)
- **CUANDO** se inspecciona (prueba de pin)
- **ENTONCES** ninguno prescribe `--handoff-policy` como argumento de `kickoff seal`
- **Y** el mensaje de error que guía a sellar el kickoff no lo incluye

### Requirement: `continuous` Solo se Sella con el Rol `fullstack` (REQ-28.12)

`axiom sdd kickoff seal` con `--execution-style continuous` DEBE rechazarse salvo que el roster sea exactamente `[fullstack]`: un único rol cuya identidad es `fullstack`. La regla compara la **identidad** de los roles y NO su política de compuerta: `continuous` no tiene compuertas, así que `fullstack:blocking`, `fullstack:deferred` y `fullstack:optional` se aceptan por igual. Un roster con roles repetidos (por ejemplo, `fullstack` dos veces) NO es exactamente `[fullstack]` y se rechaza con el mismo mensaje. Un único rol especializado (por ejemplo, solo `backend`) y cualquier roster con varios roles, incluso si uno es `fullstack`, también se rechazan. El rechazo DEBE ocurrir antes de escribir nada, NO DEBE crear ni modificar `kickoff.yaml`, y su mensaje DEBE nombrar las dos salidas: sellar con `--execution-style checkpointed`, o quitar los roles especializados para sellar con el rol único `fullstack`. La regla es una validación de **sellado**: NO DEBE aplicarse al leer ni al validar un `kickoff.yaml` ya existente. Un `kickoff.yaml` existente (sellado o archivado) con `continuous` y un roster distinto de `[fullstack]` DEBE seguir leyéndose sin error y conservar su comportamiento. El retro-sello `seal --infer` (`sealed_by: inferred`) queda exento (REQ-21.4 de `sdd-lane-kickoff`). El estilo `checkpointed` admite cualquier roster válido, incluido `[fullstack]`.

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

- **DADO** el sellado `checkpointed` con `core` y `web`, y otro con solo `fullstack`
- **CUANDO** se ejecutan
- **ENTONCES** ambos se completan

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

### Requirement: La Preferencia de Revisión Independiente se Recoge en una Interacción Propia Tras el Pre-Vuelo, sin Alterar sus Tres Grupos Canónicos (REQ-28.14)

Tras validar los tres grupos canónicos del pre-vuelo (ritmo, almacén y estrategia de PR) y antes del primer lanzamiento de fase de la sesión, el orquestador DEBE presentar **una única interacción propia** con dos preguntas de selección única: (1) revisión tras el apply de cada rol, con las opciones de token canónico `verify`, `verify-judgment` y `none`; (2) revisión de cobertura tras spec y design, con las opciones `yes` y `no`. Las preguntas tienen los tokens canónicos `after_apply` y `coverage`. Las etiquetas visibles PUEDEN traducirse al idioma de la conversación junto con las descripciones; los tokens canónicos no se traducen y son los que el orquestador cachea. Cada opción DEBE llevar una descripción que declare su efecto y su coste; la de `verify-judgment` DEBE declarar que en modalidad `continuous` incluye las correcciones acotadas del juicio dentro de las raíces de edición autorizadas.

La preferencia NO DEBE presentarse como cuarto grupo del pre-vuelo ni reutilizar sus marcadores: el texto de la interacción NO DEBE contener el prefijo `Axiom SDD preflight` ni un marcador `N/3` o `4/…`. Los marcadores `Axiom SDD preflight 1/3`, `2/3` y `3/3`, sus etiquetas, orden y mapeos canónicos, y la corroboración que hace el hook del runtime NO cambian; una respuesta con exactamente una opción por cada uno de esos tres grupos sigue estableciendo la autoridad del pre-vuelo. La preferencia NO establece autoridad de despacho: solo decide si el orquestador lanza diagnósticos informativos. La política de revisión de 400 líneas sigue sin preguntarse.

En runtimes con interfaz nativa de preguntas se usa esa interfaz; en los demás, el orquestador DEBE presentar el contenido completo en texto (ambas preguntas, todas sus opciones y descripciones) con la sintaxis de respuesta `After apply: <label>; Coverage: <label>`, y detenerse hasta recibir una respuesta válida. La respuesta se acepta solo si cada parte coincide con exactamente una opción presentada de su propia pregunta (insensible a mayúsculas y tras recortar espacios). Una respuesta ausente, ambigua o fuera del dominio NO DEBE sustituirse por un valor por defecto: se vuelve a presentar la interacción completa. La preferencia se cachea en la sesión como la estrategia de entrega, sin campo nuevo en `kickoff.yaml` ni en `gates.yaml` y sin observación de memoria persistente (REQ-28.2).

#### Scenario: La interacción presenta dos preguntas con sus opciones y descripciones

- **DADO** una sesión SDD cuyo pre-vuelo canónico ya está validado
- **CUANDO** el orquestador presenta la preferencia
- **ENTONCES** la interacción tiene dos preguntas de selección única: `after_apply` con `verify`, `verify-judgment` y `none`, y `coverage` con `yes` y `no`
- **Y** cada opción lleva una descripción de efecto y coste
- **Y** la de `verify-judgment` declara que en `continuous` incluye las correcciones acotadas del juicio

#### Scenario: Las etiquetas se pueden traducir y el token canónico se conserva

- **DADO** una conversación en español y una etiqueta visible traducida para `verify-judgment`
- **CUANDO** el usuario la elige
- **ENTONCES** el orquestador cachea el token canónico `verify-judgment`
- **Y** ninguna etiqueta en un idioma concreto forma parte del dominio canónico de respuesta

#### Scenario: Los tres grupos canónicos siguen siendo válidos y no cambian

- **DADO** una respuesta con exactamente una opción para cada grupo `1/3`, `2/3` y `3/3`
- **CUANDO** el hook del runtime corrobora el pre-vuelo
- **ENTONCES** la autoridad del pre-vuelo se establece como antes de este cambio
- **Y** los marcadores, etiquetas y orden de esos tres grupos no han cambiado

#### Scenario: La interacción de la preferencia no reutiliza los marcadores del pre-vuelo

- **DADO** el texto de la interacción de la preferencia
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** no contiene `Axiom SDD preflight` ni un marcador `4/`
- **Y** el hook ignora esa interacción al corroborar el pre-vuelo

#### Scenario: Se presenta después del pre-vuelo y antes de la primera fase

- **DADO** un pre-vuelo canónico validado en una sesión nueva
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

#### Scenario: Una opción de la preferencia en un grupo canónico se rechaza

- **DADO** una respuesta del pre-vuelo canónico en la que un grupo lleva una opción de la preferencia (por ejemplo, `verify` como estrategia de PR)
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
