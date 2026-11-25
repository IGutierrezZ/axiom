# Especificación de Requerimientos: Revisión Independiente en SDD (verify técnico por rol, juicio opcional y revisión de cobertura)

## Propósito

Definir el comportamiento observable de las revisiones independientes que el orquestador SDD ofrece, o aplica según la preferencia del usuario, durante un cambio: un `verify` por rol con cumplimiento técnico tras cada `apply`, un juicio opcional sobre el diff de ese rol tras un `verify` limpio, y una revisión de cobertura opcional tras `spec` y tras `design`. Las tres son **informativas**: informan, no bloquean ninguna fase ni el archivado, y la decisión es siempre humana. Son ajenas a RDD.

Esta capacidad define el QUÉ. La forma de proyectar el texto a los runtimes, los nombres de ficheros de informe, el número de revisores de cobertura y los identificadores internos de la preferencia pertenecen a `design`.

Definiciones usadas en esta especificación:

- **Preferencia de revisión independiente:** respuesta del usuario en una interacción propia tras el pre-vuelo (REQ-28.14 de `sdd-preflight-configuration`) con dos partes: *revisión tras el apply de cada rol* (`verify`, `verify-judgment` o `none`) y *revisión de cobertura tras spec y design* (`yes` o `no`).
- **Verify limpio:** informe de verify del rol sin ningún hallazgo de severidad crítica (funcional o de cumplimiento técnico) y sin ninguna comprobación obligatoria fallida (compilación, pruebas, análisis estático declarados). Los hallazgos de aviso o sugerencia NO impiden que un verify sea limpio. Un informe parcial por herramienta no disponible NO es limpio (supuesto A-1).
- **Informe vigente del rol:** el contenido actual de su `verify_file` (`verify-report.md` para `fullstack`, `verify-report.<rol>.md` para el resto).

---

## 1. Capacidad: `sdd-independent-review`

### Requirement: El Orquestador Ofrece la Revisión Independiente en Cuatro Puntos del Flujo (REQ-28.1)

El orquestador DEBE contemplar la revisión independiente en cuatro puntos y solo en ellos: (a) al concluir el `apply` de cada rol activo, incluido el rol único `fullstack`, el `verify` de ese rol y, tras un verify limpio, el juicio; (b) al concluir la fase `spec`, la revisión de cobertura del spec; (c) al concluir la fase `design`, la revisión de cobertura de spec y design; (d) al presentar la compuerta `integration` de un cambio `checkpointed`, la **oferta** de la verificación conjunta cuando el resumen de `axiom branch scope` indica `offerable` (REQ-25.54 de `change-branch-lifecycle`). En los puntos (a), (b) y (c) DEBE aplicar la preferencia del usuario según el estilo sellado (REQ-28.2). El punto (d) es solo una oferta: el orquestador NO DEBE lanzar la verificación conjunta por iniciativa propia, con ninguna preferencia, y NO DEBE añadir una pregunta ni una opción nueva a la preferencia de revisión independiente; la persona decide en la propia presentación de `integration`. La oferta NO DEBE añadir estados, campos de ledger ni puertas: es texto del orquestador. Con la preferencia `none` o cobertura `no`, el orquestador NO DEBE lanzar la revisión por iniciativa propia en los puntos (a), (b) y (c), pero DEBE lanzarla si el usuario la pide de forma explícita, y DEBE atender siempre una petición explícita de verificación conjunta, aunque la oferta no sea `offerable`, declarando los huecos (REQ-28.41).
(Previously: «tres puntos y solo en ellos», sin ninguna oferta en `integration`)

#### Scenario: Un rol que concluye su apply activa el verify según la preferencia
- **DADO** un cambio con el rol `core` y la preferencia `verify` tras el apply
- **CUANDO** `core` concluye su `apply`
- **ENTONCES** el orquestador lanza el `verify` de `core` sobre el diff de `core`
- **Y** no lanza ningún juicio

#### Scenario: Cada rol activa su propia oferta
- **DADO** un cambio con los roles `core` y `web` y la preferencia `verify`
- **CUANDO** `core` concluye su `apply` y, más tarde, `web` concluye el suyo
- **ENTONCES** el orquestador lanza un `verify` tras cada uno, cada uno sobre el diff de su rol
- **Y** el verify de `web` no se lanza antes de que `web` concluya

#### Scenario: Con la preferencia none no se lanza nada por iniciativa propia
- **DADO** la preferencia `none` y cobertura `no`
- **CUANDO** concluyen `spec`, `design` y el `apply` de un rol
- **ENTONCES** el orquestador no lanza ningún verify, juicio ni revisión de cobertura por iniciativa propia
- **Y** el flujo continúa según el estilo sellado

#### Scenario: Una petición explícita se atiende con la preferencia none
- **DADO** la preferencia `none`
- **CUANDO** el usuario pide explícitamente un verify del rol `core` ya aplicado
- **ENTONCES** el orquestador lanza el verify de `core` conforme a esta capacidad

#### Scenario: Las ofertas no crean estado de gobierno
- **DADO** un cambio en el que se han ejecutado verify, juicio, cobertura y verificación conjunta
- **CUANDO** se inspeccionan `gates.yaml` y `kickoff.yaml`
- **ENTONCES** no contienen ningún campo, decisión ni puerta nueva atribuible a estas revisiones

#### Scenario: Un rol sin diff propio no produce un verify vacío engañoso
- **DADO** un rol cuyo apply no modificó ningún fichero
- **CUANDO** se aplica la preferencia `verify` tras su apply
- **ENTONCES** el verify informa de que el diff del rol está vacío y no emite un veredicto de cumplimiento técnico sobre nada
- **Y** no se ofrece juicio

#### Scenario: La compuerta integration ofrece la verificación conjunta sin lanzarla
- **DADO** un cambio `checkpointed` con la preferencia `verify-judgment` y `offer` igual a `offerable`
- **CUANDO** el orquestador presenta la compuerta `integration`
- **ENTONCES** ofrece la verificación conjunta en la presentación
- **Y** no la lanza hasta que la persona la acepta
- **Y** no formula ninguna pregunta de preferencia nueva

#### Scenario: Con la preferencia none la oferta de integration no se lanza sola
- **DADO** la preferencia `none` y `offer` igual a `offerable`
- **CUANDO** el orquestador presenta la compuerta `integration`
- **ENTONCES** declara el estado «ofrecida» sin lanzarla por iniciativa propia
- **Y** la lanza si la persona la pide de forma explícita

### Requirement: Aplicación de la Preferencia según el Estilo Sellado, sin Violar `continuous` (REQ-28.2)

La preferencia de revisión independiente DEBE aplicarse sin preguntar de nuevo al usuario cuando el estilo sellado es `continuous`: el orquestador lanza las revisiones que la preferencia indica y encadena la fase siguiente al terminar, sin detenerse (REQ-21.7 de `sdd-block-review-gates` no cambia). Cuando el estilo sellado es `checkpointed`, la preferencia es la respuesta por defecto que el orquestador aplica antes de la parada correspondiente y cuyos resultados presenta en ella (REQ-28.10); la parada NO se sustituye ni se adelanta. El orquestador DEBE guardar la preferencia en la caché de la sesión del mismo modo que la estrategia de entrega y NO DEBE escribirla en `kickoff.yaml` ni en `gates.yaml`. Cada sesión recoge la preferencia tras su pre-vuelo (REQ-28.14). Cuando una sesión carece de preferencia cacheada (por ejemplo, un pre-vuelo establecido por un orquestador anterior a este cambio), el orquestador NO DEBE reconstruirla por inferencia: en `continuous` NO DEBE lanzar ninguna revisión por iniciativa propia ni preguntar; en `checkpointed` DEBE ofrecer la elección en la parada correspondiente como parte de su presentación.

#### Scenario: En continuous se aplica la preferencia sin detenerse

- **DADO** un cambio `continuous` con rol `fullstack`, preferencia `verify-judgment` y cobertura `yes`
- **CUANDO** concluyen `spec`, `design` y el `apply`
- **ENTONCES** el orquestador lanza la revisión de cobertura tras `spec`, la de `design` tras `design`, y el verify (y el juicio si el verify es limpio) tras el `apply`
- **Y** no formula ninguna pregunta ni presenta ninguna parada entre fases

#### Scenario: En checkpointed la preferencia es el valor por defecto y la parada se conserva

- **DADO** un cambio `checkpointed` con preferencia `verify` tras el apply
- **CUANDO** el rol concluye su `apply`
- **ENTONCES** el orquestador lanza el verify y, con su informe, presenta la parada `role-apply:<rol>` con las tres salidas de REQ-26.1
- **Y** el usuario puede, en esa parada, pedir además un juicio o repetir el verify antes de elegir una salida

#### Scenario: Sesión sin preferencia cacheada en continuous

- **DADO** un cambio `continuous` en una sesión sin preferencia cacheada (pre-vuelo de un orquestador anterior)
- **CUANDO** el rol concluye su `apply`
- **ENTONCES** el orquestador no lanza ninguna revisión por iniciativa propia ni formula ninguna pregunta
- **Y** continúa el flujo

#### Scenario: Sesión sin preferencia cacheada en checkpointed

- **DADO** un cambio `checkpointed` en una sesión sin preferencia cacheada (pre-vuelo de un orquestador anterior)
- **CUANDO** el orquestador presenta la parada `role-apply:<rol>`
- **ENTONCES** ofrece en la propia presentación las opciones `verify`, `verify-judgment` y `none`
- **Y** no escribe la elección en `kickoff.yaml` ni en `gates.yaml`

#### Scenario: La preferencia no se persiste en el kickoff

- **DADO** cualquier cambio sellado tras responder la preferencia
- **CUANDO** se inspecciona `kickoff.yaml`
- **ENTONCES** no contiene ningún campo de preferencia de revisión independiente

#### Scenario: La modalidad continua sigue sin presentar paradas aunque haya juicio

- **DADO** un cambio `continuous` con preferencia `verify-judgment`
- **CUANDO** el juicio concluye
- **ENTONCES** `axiom sdd status` no proyecta ninguna pregunta de compuerta
- **Y** el orquestador continúa con la fase siguiente

---

### Requirement: Las Revisiones Son Informativas y No Bloquean (REQ-28.3)

Ni el verify por rol, ni el juicio, ni la revisión de cobertura DEBEN bloquear una fase, una puerta o el archivado. Un hallazgo, sea cual sea su severidad, NO DEBE alterar `nextRecommended`, las dependencias proyectadas por `axiom sdd status`, ni impedir `sdd-archive`. La decisión de actuar sobre un hallazgo es humana: en `checkpointed` se toma en la parada; en `continuous` el orquestador informa en su resumen final y no corrige por su cuenta, salvo las correcciones acotadas del juicio (REQ-28.6). El verify NO DEBE mutar código, tareas ni artefactos de planificación; solo escribe su informe. **Excepción declarada:** en un cambio sellado como ISO (REQ-29.1 de `sdd-iso-evidence` y REQ-29.13 de `sdd-lane-kickoff`), la matriz de verify del informe SÍ condiciona el archivado: `archive` se bloquea mientras la matriz esté ausente, sea inválida o tenga alguna fila fallida (REQ-29.11 de `sdd-archive-lifecycle`). Ese bloqueo procede de la validez de la matriz —evidencia declarada y contrastada con el árbol de trabajo— y nunca de un hallazgo de revisión: los hallazgos, el juicio `ESCALATED` y la revisión de cobertura siguen sin alterar `nextRecommended` ni las dependencias, también en un cambio ISO.
(Previously: ninguna revisión ni informe de verify bloqueaba el archivado en ningún caso; ahora un cambio ISO tiene la excepción de la matriz de verify)

#### Scenario: Un hallazgo crítico no bloquea el archivado

- **DADO** un verify de rol con un hallazgo de cumplimiento técnico de severidad crítica
- **CUANDO** se consulta `axiom sdd status` y se solicita el archivado
- **ENTONCES** `nextRecommended` y las dependencias son idénticas a las de un cambio sin ese hallazgo
- **Y** el archivado no se rechaza por el informe

#### Scenario: Un juicio ESCALATED no bloquea

- **DADO** un juicio del rol que termina con `JUDGMENT: ESCALATED`
- **CUANDO** el orquestador continúa el flujo
- **ENTONCES** el veredicto se presenta como informativo
- **Y** ninguna fase ni puerta queda bloqueada por él

#### Scenario: El verify no modifica el trabajo del rol

- **DADO** un verify de rol que detecta un incumplimiento
- **CUANDO** termina
- **ENTONCES** no ha modificado código, `tasks*.md` ni `spec.md`/`design.md`
- **Y** el único fichero escrito es su informe

#### Scenario: Una revisión de cobertura con huecos no impide iniciar la fase siguiente

- **DADO** un informe de cobertura con huecos abiertos en un cambio `continuous`
- **CUANDO** concluye la revisión
- **ENTONCES** el orquestador continúa con la fase siguiente
- **Y** los huecos quedan en el informe y en el resumen final

#### Scenario: En un cambio ISO una matriz inválida bloquea el archivado sin ningún hallazgo

- **DADO** un cambio ISO cuyo informe de verify no contiene ningún hallazgo y cuya matriz tiene un escenario sin fila
- **CUANDO** se solicita el archivado
- **ENTONCES** `archive` se bloquea con el motivo tipado de matriz (REQ-29.11 de `sdd-archive-lifecycle`)
- **Y** el motivo no cita ningún hallazgo de revisión

#### Scenario: En un cambio ISO un hallazgo crítico con la matriz válida no bloquea

- **DADO** un cambio ISO con una matriz válida y con éxito y un verify con un hallazgo de cumplimiento técnico de severidad crítica
- **CUANDO** se consulta `axiom sdd status` y se solicita el archivado
- **ENTONCES** `nextRecommended` y las dependencias son idénticas a las de un cambio ISO sin ese hallazgo
- **Y** el archivado no se rechaza por el informe

#### Scenario: En un cambio ISO un juicio ESCALATED no bloquea

- **DADO** un cambio ISO con una matriz válida y con éxito y un juicio que termina con `JUDGMENT: ESCALATED`
- **CUANDO** el orquestador continúa el flujo
- **ENTONCES** el veredicto se presenta como informativo
- **Y** `archive` no queda bloqueado por él

#### Scenario: Fuera de ISO la matriz no condiciona el archivado

- **DADO** un cambio no ISO cuyo informe de verify no contiene matriz
- **CUANDO** se solicita el archivado
- **ENTONCES** el archivado no se rechaza por el informe

---

### Requirement: Verify por Rol con Sección de Cumplimiento Técnico (REQ-28.4)

El `verify` de un rol DEBE inspeccionar únicamente el diff de ese rol y escribir su informe en el `verify_file` que el kickoff asigna a ese rol. Además de la verificación funcional existente (requisitos, pruebas, compilación), el informe DEBE incluir una sección **Cumplimiento técnico** que contraste el diff con las reglas aplicables (REQ-28.5). Cuando no hay hallazgos, la sección DEBE existir y declararlo, con la lista de fuentes consultadas. El verify global multirrol, que se evalúa cuando REQ-21.15 lo habilita, no cambia en cuanto a su habilitación ni a la semántica del veredicto que lee la barrera de `multi-role-fan-out`: la sección nueva es contenido adicional del informe. **Alcance del verify sin rol:** cuando el verify global se ejecuta sin rol en modo conjunto sobre un cambio con instantánea sellada de repositorios, DEBE inspeccionar la **unión del alcance** que devuelve `axiom branch scope` (REQ-25.51 de `change-branch-lifecycle`) y no un alcance definido por el agente; ese alcance solo delimita lo que se inspecciona: la suite completa de pruebas que REQ-21.15 exige no se acota por él. Cuando `axiom branch scope` no es aplicable o falla, el verify sin rol se comporta como antes y su informe declara el motivo; nunca falla por ello ni cambia su veredicto. En un cambio ISO, el informe consolidado (o el del rol único `fullstack`) incluye además la matriz de verify (REQ-29.6 de `sdd-iso-evidence`) como otra sección adicional, que NO escribe líneas `verdict:` y NO altera la sección «Cumplimiento técnico»; los informes por rol de un cambio multirrol NO necesitan matriz. En un cambio con el rol único `fullstack`, el verify del rol y el verify del cambio son el mismo informe y NO se ejecutan dos veces sin motivo. Nota: el verify por rol es un diagnóstico del rol que se ofrece antes de aprobar `role-apply`; NO es la fase `verify` global de REQ-21.15, que sigue evaluándose cuando REQ-21.15 la habilita.
(Previously: el verify global multirrol «NO cambia» sin matiz y no tenía alcance definido; ahora, sin rol y en modo conjunto, su alcance es la unión del de `axiom branch scope`, sin alterar su habilitación ni el veredicto que lee la barrera)

#### Scenario: El informe de un rol contiene la sección Cumplimiento técnico
- **DADO** el rol `web` con su `apply` concluido y un diff en `web`
- **CUANDO** se ejecuta su verify
- **ENTONCES** `verify-report.web.md` contiene una sección «Cumplimiento técnico»
- **Y** el informe no incluye hallazgos sobre ficheros modificados solo por otro rol

#### Scenario: El rol fullstack escribe en verify-report.md
- **DADO** un cambio con el rol único `fullstack`
- **CUANDO** se ejecuta su verify
- **ENTONCES** el informe se escribe en `verify-report.md`
- **Y** contiene la sección «Cumplimiento técnico»

#### Scenario: Sin hallazgos la sección existe y lo declara
- **DADO** un diff que cumple todas las reglas aplicables
- **CUANDO** se genera el informe
- **ENTONCES** la sección «Cumplimiento técnico» declara que no hay hallazgos
- **Y** enumera las fuentes de reglas contrastadas

#### Scenario: El verify global multirrol no cambia su habilitación
- **DADO** un cambio con los roles `core` y `web`, ambos `blocking`, y todos los `role-apply` aprobados
- **CUANDO** se evalúa la dependencia `verify`
- **ENTONCES** se habilita conforme a REQ-21.15, sin ninguna condición añadida por este requisito
- **Y** el veredicto que lee la barrera de sincronización se obtiene como antes, sin tener en cuenta la sección nueva

#### Scenario: El verify sin rol en modo conjunto usa el alcance de axiom branch scope
- **DADO** un cambio multirrepo con instantánea sellada y el verify global lanzado sin rol en modo conjunto
- **CUANDO** el verify determina qué inspecciona
- **ENTONCES** inspecciona, por repositorio, los commits y ficheros que devuelve `axiom branch scope`
- **Y** no incluye commits ajenos que ese comando excluye

#### Scenario: Si el comando de alcance no es aplicable o falla, el verify lo declara y continúa
- **DADO** un cambio sin instantánea sellada, o una ejecución de `axiom branch scope` que termina con un error tipado
- **CUANDO** se ejecuta el verify sin rol
- **ENTONCES** el informe declara el motivo concreto y que se aplicó el comportamiento anterior
- **Y** el verify no falla por ello ni cambia su veredicto

#### Scenario: El alcance no acota la suite completa de pruebas
- **DADO** un verify sin rol en modo conjunto con un alcance de dos repositorios
- **CUANDO** se ejecutan las pruebas que REQ-21.15 exige
- **ENTONCES** se ejecuta la suite completa y no solo la de los ficheros del alcance
- **Y** el informe indica que el alcance solo delimita lo inspeccionado

#### Scenario: En un cambio ISO el informe consolidado incluye la matriz como sección adicional
- **DADO** un cambio ISO con los roles `core` y `web` y todos los `role-apply` aprobados
- **CUANDO** se genera el `verify-report.md` consolidado
- **ENTONCES** contiene la matriz de verify junto a las demás secciones
- **Y** la barrera lee la misma línea `verdict:` que sin la matriz
- **Y** los informes `verify-report.core.md` y `verify-report.web.md` no necesitan matriz

#### Scenario: Un rol con herramienta de análisis no disponible informa la limitación
- **DADO** una comprobación de estilo declarada cuyo ejecutable no está disponible
- **CUANDO** se genera la sección «Cumplimiento técnico»
- **ENTONCES** declara la comprobación como no ejecutada con su motivo
- **Y** no la presenta como superada

### Requirement: Fuentes de Reglas y Forma de los Hallazgos Técnicos (REQ-28.5)

La sección «Cumplimiento técnico» DEBE contrastar el diff del rol con: (a) las skills que `skill_resolution` del apply del rol declara cargadas; (b) las skills del registro cuyos disparadores casan con los ficheros tocados por el rol; (c) el contexto técnico (`entry` de `technicalContext`) cuando la proyección del estado lo declara; (d) las reglas y el contexto de `openspec/config.yaml` aplicables a la fase `apply` y a la verificación. Cada hallazgo DEBE citar la regla incumplida (fuente y, cuando exista, identificador o frase) y la ubicación en el diff (fichero y línea o rango), e indicar su severidad. Un hallazgo sin regla citable NO DEBE presentarse como incumplimiento: se presenta, como mucho, como observación. Las fuentes ausentes (por ejemplo, sin `technicalContext`) NO DEBEN producir hallazgos ni error: se omiten y se declaran en la lista de fuentes. El verify NO DEBE crear, modificar ni borrar ficheros bajo las rutas de `technicalContext.reference`.

#### Scenario: Un hallazgo cita regla y ubicación

- **DADO** un diff que incumple una regla de una skill cargada por el apply del rol
- **CUANDO** se genera el informe
- **ENTONCES** el hallazgo nombra la skill y la regla
- **Y** cita el fichero y la línea o rango del diff

#### Scenario: Una skill del registro que casa con los ficheros tocados se contrasta aunque el apply no la cargara

- **DADO** un rol cuyo diff toca ficheros que activan una skill del registro no declarada en `skill_resolution`
- **CUANDO** se genera el informe
- **ENTONCES** la sección contrasta el diff con las reglas de esa skill
- **Y** la lista de fuentes la nombra

#### Scenario: Un proyecto sin contexto técnico declarado

- **DADO** un proyecto cuya proyección de estado no declara `technicalContext`
- **CUANDO** se genera el informe
- **ENTONCES** la fuente (c) se declara como no aplicable
- **Y** no se emite ningún hallazgo ni error por su ausencia

#### Scenario: Las reglas de config.yaml se aplican

- **DADO** un `openspec/config.yaml` con reglas para `apply` y `verify` y un diff que incumple una de ellas
- **CUANDO** se genera el informe
- **ENTONCES** el hallazgo cita `openspec/config.yaml` y la regla

#### Scenario: Una opinión sin regla no es un incumplimiento

- **DADO** una observación de estilo del revisor que ninguna fuente respalda
- **CUANDO** se genera el informe
- **ENTONCES** no figura como hallazgo de incumplimiento
- **Y**, si se incluye, se marca como observación sin regla asociada

#### Scenario: Un hallazgo en una ruta de referencia no se corrige

- **DADO** un hallazgo sobre un fichero bajo `technicalContext.reference`
- **CUANDO** se genera el informe
- **ENTONCES** el verify lo informa
- **Y** no modifica ese fichero

---

### Requirement: El Juicio Solo se Ofrece o Lanza Tras un Verify Limpio del Rol (REQ-28.6)

El orquestador DEBE ofrecer o lanzar el juicio de un rol únicamente cuando el verify de ese rol, en su estado vigente, es limpio. Con un verify no limpio, el orquestador NO DEBE lanzarlo por iniciativa propia, DEBE informar de que el juicio no se lanza por no ser limpio el verify, y DEBE aceptar una petición explícita del usuario de lanzarlo igualmente (supuesto A-2: la petición explícita de un usuario informado prevalece). Con la preferencia `verify-judgment` y un verify no limpio, el juicio simplemente no se lanza y el flujo continúa. En un cambio ISO cuyo informe vigente del rol contiene la matriz de verify (el rol único `fullstack`), una matriz ausente, inválida o con alguna fila fallida (REQ-29.6 a REQ-29.8 de `sdd-iso-evidence`) cuenta como comprobación obligatoria fallida: el verify no es limpio y el juicio no se lanza por iniciativa propia. En un cambio multirrol, la matriz vive en el informe consolidado y no afecta a si el verify de un rol es limpio. Antes de lanzar el juicio, el orquestador DEBE informar del coste previsto (dos jueces ciegos y hasta dos rondas con el actor de arreglos) en una línea; en `continuous` la información no detiene el flujo.
(Previously: «limpio» no consideraba la matriz de verify de un cambio ISO)

#### Scenario: Verify limpio habilita el juicio

- **DADO** la preferencia `verify-judgment` y un verify del rol sin hallazgos críticos ni comprobaciones fallidas
- **CUANDO** termina el verify
- **ENTONCES** el orquestador informa del coste y lanza el juicio sobre el diff del rol

#### Scenario: Verify con hallazgo crítico no lanza el juicio

- **DADO** la preferencia `verify-judgment` y un verify con un hallazgo de severidad crítica
- **CUANDO** termina el verify
- **ENTONCES** el orquestador no lanza el juicio
- **Y** informa de que no lo lanza porque el verify no es limpio

#### Scenario: Un verify con solo avisos y sugerencias es limpio

- **DADO** un verify cuyo único contenido de hallazgos son avisos y sugerencias
- **CUANDO** el orquestador decide si lanza el juicio
- **ENTONCES** lo trata como limpio y lo lanza

#### Scenario: Un verify parcial por herramienta no disponible no es limpio

- **DADO** un verify en el que las pruebas no pudieron ejecutarse
- **CUANDO** el orquestador decide si lanza el juicio
- **ENTONCES** no lo lanza por iniciativa propia
- **Y** informa del motivo

#### Scenario: Petición explícita con verify no limpio

- **DADO** un verify no limpio y la petición explícita del usuario de lanzar el juicio
- **CUANDO** el orquestador la atiende
- **ENTONCES** lanza el juicio avisando de que el verify no era limpio

#### Scenario: Sin verify previo no hay juicio por iniciativa del orquestador

- **DADO** una preferencia que no incluye verify
- **CUANDO** concluye el `apply` del rol
- **ENTONCES** el orquestador no lanza el juicio por iniciativa propia

#### Scenario: En ISO una matriz inválida hace que el verify no sea limpio

- **DADO** un cambio ISO con el rol único `fullstack`, la preferencia `verify-judgment` y un verify sin hallazgos cuya matriz tiene un escenario sin fila
- **CUANDO** termina el verify
- **ENTONCES** el verify no es limpio
- **Y** el orquestador no lanza el juicio e informa del motivo

#### Scenario: En ISO una fila fallida hace que el verify no sea limpio

- **DADO** un cambio ISO con el rol único `fullstack` y una matriz válida con una fila de resultado `fail`
- **CUANDO** el orquestador decide si lanza el juicio
- **ENTONCES** no lo lanza por iniciativa propia
- **Y** informa de que la matriz tiene una fila fallida

#### Scenario: En ISO una matriz válida y con éxito no impide un verify limpio

- **DADO** un cambio ISO con el rol único `fullstack`, una matriz válida y con éxito y un verify sin hallazgos críticos ni comprobaciones fallidas
- **CUANDO** el orquestador decide si lanza el juicio
- **ENTONCES** lo trata como limpio

#### Scenario: Fuera de ISO la matriz no interviene en «limpio»

- **DADO** un cambio no ISO sin matriz y un verify sin hallazgos críticos ni comprobaciones fallidas
- **CUANDO** el orquestador decide si lanza el juicio
- **ENTONCES** lo trata como limpio
### Requirement: El Juicio Toma como Objetivo el Diff de un Rol SDD y Acota sus Arreglos (REQ-28.7)

El juicio DEBE aceptar como objetivo el diff de un rol SDD. Que el usuario acepte la oferta del orquestador, o que su preferencia incluya el juicio, cuenta como la petición explícita que exige el contrato de activación del juicio. Se conservan las reglas del juicio: dos jueces ciegos en paralelo, corroboración por acuerdo, arreglo solo de hallazgos severos confirmados por ambos, como mucho dos rondas y veredicto terminal `APPROVED` o `ESCALATED`; la corroboración es por intersección. Los arreglos DEBEN limitarse a las raíces de edición autorizadas del cambio y NO DEBEN tocar ficheros fuera de ellas ni de las rutas de referencia del contexto técnico. Los arreglos aplicados DEBEN quedar registrados en el progreso del rol. En `checkpointed`, el orquestador DEBE pedir confirmación antes de la corrección de la primera ronda, como establece el contrato del juicio; en `continuous`, la preferencia `verify-judgment` incluye esa autorización, y la descripción de esa opción en la interacción de la preferencia DEBE decirlo (REQ-28.14).

#### Scenario: El juicio recibe el diff del rol como objetivo

- **DADO** un verify limpio del rol `core` y la preferencia `verify-judgment`
- **CUANDO** se lanza el juicio
- **ENTONCES** el objetivo declarado es el diff de `core`
- **Y** los dos jueces trabajan en paralelo sin ver el veredicto del otro

#### Scenario: Solo se arreglan los severos confirmados por ambos jueces

- **DADO** un hallazgo severo señalado solo por un juez y otro severo señalado por ambos
- **CUANDO** el juicio decide las correcciones
- **ENTONCES** solo el segundo se arregla
- **Y** el primero se informa como sospecha sin corregir

#### Scenario: El arreglo no sale de las raíces de edición autorizadas

- **DADO** un hallazgo severo confirmado cuya corrección exigiría editar un fichero fuera de las raíces autorizadas del cambio
- **CUANDO** el actor de arreglos intenta corregirlo
- **ENTONCES** no edita ese fichero
- **Y** el hallazgo se informa como pendiente de decisión humana y el veredicto no puede ser `APPROVED` por ese hallazgo

#### Scenario: Máximo dos rondas

- **DADO** un hallazgo confirmado que persiste tras dos rondas
- **CUANDO** se agota el presupuesto de rondas
- **ENTONCES** el veredicto es `ESCALATED`
- **Y** no se abre una tercera ronda

#### Scenario: En checkpointed se pide confirmación antes de corregir

- **DADO** un cambio `checkpointed` con hallazgos severos confirmados en la primera ronda
- **CUANDO** el juicio llega a la corrección
- **ENTONCES** el orquestador pide confirmación antes de que el actor de arreglos edite

#### Scenario: En continuous la preferencia autoriza la corrección acotada

- **DADO** un cambio `continuous` con preferencia `verify-judgment` y un hallazgo severo confirmado
- **CUANDO** el juicio llega a la corrección
- **ENTONCES** el actor de arreglos corrige dentro de las raíces autorizadas sin que el orquestador se detenga a preguntar

#### Scenario: Un juicio sin hallazgos severos termina sin correcciones

- **DADO** un diff sin hallazgos severos confirmados
- **CUANDO** el juicio concluye
- **ENTONCES** el veredicto es `APPROVED`
- **Y** no se ha editado ningún fichero

---

### Requirement: Si el Juicio Corrige, su Verificación Final Pasa a Ser el Informe Vigente (REQ-28.8)

Si el juicio modifica código, su verificación final posterior a la última corrección DEBE convertirse en el informe vigente del rol, y el contenido del verify anterior DEBE conservarse como contexto histórico dentro del mismo informe (fecha, origen y hallazgos que ya no se mantienen y por qué), sin perder ningún hallazgo no resuelto. Si el juicio no modifica código, el informe vigente es el verify previo, sin cambios. El informe vigente es el que presentan las paradas (REQ-28.10) y el que lee cualquier consumidor del `verify_file`. La verificación final NO DEBE bloquear (REQ-28.3).

#### Scenario: Tras una corrección el informe vigente es la verificación final

- **DADO** un juicio que corrige un hallazgo severo confirmado
- **CUANDO** termina la verificación final posterior a la corrección
- **ENTONCES** el `verify_file` del rol refleja esa verificación como informe vigente
- **Y** el verify anterior consta en él como contexto histórico con su fecha y origen

#### Scenario: Los hallazgos no resueltos se conservan

- **DADO** un verify previo con un hallazgo de aviso que la corrección no resolvió
- **CUANDO** se escribe el informe vigente
- **ENTONCES** ese hallazgo sigue constando como abierto
- **Y** no se declara resuelto sin evidencia

#### Scenario: Sin correcciones el informe no cambia

- **DADO** un juicio `APPROVED` sin correcciones
- **CUANDO** termina
- **ENTONCES** el `verify_file` conserva el contenido del verify previo
- **Y** el resultado del juicio se informa por separado

#### Scenario: Una verificación final no limpia no se oculta

- **DADO** un juicio cuya corrección introduce un fallo detectado por la verificación final
- **CUANDO** se escribe el informe vigente
- **ENTONCES** el informe refleja el fallo
- **Y** el veredicto del juicio no lo oculta ni se presenta como limpio

---

### Requirement: Revisión de Cobertura por Unión, de Solo Lectura y sin Autocorrección (REQ-28.9)

La revisión de cobertura, ofrecida o aplicada tras `spec` (revisa el spec contra la propuesta) y tras `design` (revisa spec y design contra la propuesta), DEBE ejecutarla uno o más revisores de contexto limpio y solo lectura, que reciben la propuesta y los artefactos revisados, y devuelven los casos de uso, casos límite y escenarios de error que faltan. El informe DEBE ser la **unión** de los hallazgos de todos los revisores, deduplicada, sin exigir acuerdo entre ellos. La revisión NO DEBE modificar el artefacto revisado ni su contenido, y el informe se guarda como artefacto informativo de la fase, fuera de los artefactos que juzgan las puertas, de modo que su presencia no cambie el digest de ninguna puerta ni la reabra. Las correcciones DEBEN hacerlas `sdd-spec` o `sdd-design` por la salida «iterar» (REQ-26.1) en `checkpointed`, o por una nueva ejecución de la fase en `continuous`. La tabla de cobertura que el propio autor incluye en su resumen NO sustituye a esta revisión. Un informe sin huecos DEBE existir y declararlo.

#### Scenario: La cobertura del spec une los hallazgos de varios revisores

- **DADO** dos revisores que señalan huecos parcialmente distintos sobre el mismo spec
- **CUANDO** se genera el informe
- **ENTONCES** contiene la unión deduplicada de ambos conjuntos
- **Y** un hueco señalado por un solo revisor consta igualmente

#### Scenario: Un hueco duplicado aparece una vez

- **DADO** dos revisores que señalan el mismo caso límite con distinta redacción
- **CUANDO** se genera el informe
- **ENTONCES** el caso límite aparece una sola vez

#### Scenario: La revisión no modifica el artefacto ni su digest

- **DADO** una puerta de `spec` con su digest calculado
- **CUANDO** se ejecuta la revisión de cobertura y se guarda su informe
- **ENTONCES** `spec.md` no cambia
- **Y** el digest de la puerta es idéntico y la puerta no se reabre

#### Scenario: La corrección en checkpointed pasa por iterar

- **DADO** un cambio `checkpointed` con huecos en el informe de cobertura del spec
- **CUANDO** el usuario decide corregirlos
- **ENTONCES** elige la salida «iterar» y `sdd-spec` actualiza el artefacto
- **Y** la revisión de cobertura no edita el spec

#### Scenario: En continuous los huecos no se autocorrigen

- **DADO** un cambio `continuous` con huecos en el informe de cobertura
- **CUANDO** concluye la revisión
- **ENTONCES** el orquestador no invoca la corrección por su cuenta
- **Y** informa de los huecos en su resumen y continúa

#### Scenario: Un informe sin huecos se declara

- **DADO** un spec que los revisores consideran completo
- **CUANDO** se genera el informe
- **ENTONCES** existe y declara que no se detectaron huecos

#### Scenario: La cobertura tras design incluye spec y design

- **DADO** un `design` recién concluido
- **CUANDO** se ejecuta la revisión de cobertura
- **ENTONCES** los revisores reciben la propuesta, el spec y el design
- **Y** señalan también requisitos del spec que el design no cubre

#### Scenario: Un revisor sin salida no invalida la revisión

- **DADO** una revisión con dos revisores en la que uno no devuelve resultado utilizable
- **CUANDO** se genera el informe
- **ENTONCES** el informe incluye lo del revisor que respondió
- **Y** declara que el otro no respondió, sin inventar su contenido

---

### Requirement: La Verificación Conjunta es el Verify Sin Rol sobre la Unión del Alcance por Repositorio (REQ-28.41)

La verificación conjunta DEBE ser el `verify` global sin rol (REQ-21.15 de `sdd-role-closure-handoff`) y NO un informe ni una fase adicional. Su alcance DEBE ser la **unión por repositorio** de lo que devuelve `axiom branch scope` para el cambio, y DEBE verificar **solo commits**: NO DEBE incluir el árbol de trabajo ni los ficheros sin seguimiento. Los commits ajenos que `axiom branch scope` excluya DEBEN listarse como excluidos, y un aviso `foreign_commits_possible` DEBE reproducirse como aviso; nunca se incluyen en silencio.

El resultado DEBE escribirse en el informe consolidado `verify-report.md` (el `verify_file` del rol `fullstack` cuando es el único rol) como una sección propia **«Verificación conjunta»**, fechada, **una por ronda**. La sección DEBE añadirse sin sobrescribir la matriz ISO ni ningún otro contenido; una verificación posterior añade una ronda nueva y NO borra las anteriores. Cada ronda DEBE indicar, por repositorio, la rama, el `tip` verificado (SHA de la punta de la rama) y el resultado (`ok`, `empty`, `unverifiable` o `exempt`), y DEBE declarar los huecos: cada repositorio `unverifiable` con su motivo (`branch_missing`, `base_missing` o `fork_point_undetermined`), y cada rama sin fusionar en la integración. Un repositorio `empty` o `exempt` se declara como tal y no cuenta como hueco. La ronda NO es limpia mientras tenga algún hueco. El verify NO DEBE ejecutar `fetch` ni rellenar un hueco por su cuenta.

La sección NO DEBE escribir líneas `verdict:`, NO DEBE modificar la línea `**Estado:**` del informe ni alterar el veredicto que lee la barrera de `multi-role-fan-out`, la sección «Cumplimiento técnico» ni la matriz ISO. El verify conjunto NO DEBE mutar código, tareas ni artefactos de planificación; solo escribe su informe.

**Petición explícita.** Cuando la persona pide la verificación conjunta de forma explícita, el orquestador DEBE atenderla siempre, aunque el resumen de `axiom branch scope` no sea `offerable` (ramas pendientes, repositorios `unverifiable`): se ejecuta sobre lo verificable y la ronda declara los huecos. Cuando todos los repositorios son `empty` o `exempt`, no se escribe ninguna ronda y la respuesta lo declara. Cuando el resultado de `axiom branch scope` difiere entre la presentación y la ejecución, la ejecución usa el resultado recalculado.

#### Scenario: Camino feliz, la ronda cubre la unión de los repositorios
- **DADO** un cambio con los repositorios de código `api` y `web`, ambos con commits propios fusionados en la integración
- **CUANDO** se ejecuta la verificación conjunta
- **ENTONCES** `verify-report.md` contiene una sección «Verificación conjunta» fechada con una entrada por `api` y por `web`
- **Y** cada entrada indica rama, `tip` y resultado

#### Scenario: Un commit ajeno excluido no entra en la verificación
- **DADO** un cambio en modo `release` y un commit ajeno `F` llegado a `develop` después de la bifurcación
- **CUANDO** se ejecuta la verificación conjunta
- **ENTONCES** el informe no contiene hallazgos ni ficheros procedentes de `F`
- **Y** si `axiom branch scope` lista commits excluidos o el aviso `foreign_commits_possible`, la sección los reproduce

#### Scenario: Rama borrada se declara como hueco
- **DADO** un repositorio `web` que `axiom branch scope` declara `unverifiable` con motivo `branch_missing`
- **CUANDO** se genera la ronda
- **ENTONCES** declara `web` como no verificado con ese motivo
- **Y** la ronda no es limpia por ese hueco
- **Y** no se ha ejecutado ningún `fetch`

#### Scenario: Petición explícita con huecos se ejecuta y los declara
- **DADO** un cambio con una rama sin fusionar y otro repositorio `unverifiable`
- **CUANDO** la persona pide de forma explícita la verificación conjunta
- **ENTONCES** se ejecuta sobre lo verificable y no se rechaza la petición
- **Y** la ronda declara la rama pendiente de fusionar y el repositorio no verificable

#### Scenario: Solo commits, sin árbol de trabajo
- **DADO** un cambio con un fichero modificado sin commitear y otro sin seguimiento en uno de los repositorios
- **CUANDO** se ejecuta la verificación conjunta
- **ENTONCES** la verificación no inspecciona esos ficheros
- **Y** el informe declara que verifica solo commits

#### Scenario: El informe existente con matriz ISO se conserva y se añade una ronda
- **DADO** un cambio ISO con un `verify-report.md` que ya contiene la matriz, «Cumplimiento técnico», una línea `**Estado:**` y una ronda anterior de «Verificación conjunta»
- **CUANDO** se ejecuta una verificación conjunta nueva
- **ENTONCES** se añade una ronda fechada nueva y la anterior se conserva
- **Y** la matriz, las demás secciones y la línea `**Estado:**` quedan intactas
- **Y** el informe no gana ninguna línea `verdict:`

#### Scenario: Sin commits propios no se escribe ronda
- **DADO** un cambio cuyos repositorios son todos `empty` o `exempt`
- **CUANDO** la persona pide la verificación conjunta de forma explícita
- **ENTONCES** no se añade ninguna sección «Verificación conjunta»
- **Y** la respuesta declara que no hay commits propios que verificar

### Requirement: La Verificación Conjunta Nunca Bloquea (REQ-28.42)

La verificación conjunta NO DEBE bloquear ni condicionar la compuerta `integration`, `axiom sdd gate record`, el archivado, `nextRecommended` ni las dependencias de `axiom sdd status`; su ausencia, su declinación y cualquier hallazgo o hueco del informe son informativos, igual que el resto de revisiones independientes (REQ-28.3). La evidencia de REQ-21.16 de `sdd-archive-lifecycle` no cambia. En un cambio `continuous` NO existe la compuerta `integration` y el orquestador NO DEBE añadir parada, pregunta ni oferta nueva por esta capacidad.

#### Scenario: La verificación conjunta no bloquea integration
- **DADO** una compuerta `integration` pendiente y ninguna verificación conjunta hecha
- **CUANDO** se registra `axiom sdd gate record --gate integration --decision approved` con su evidencia actual
- **ENTONCES** el registro se acepta
- **Y** `nextRecommended` y las dependencias de `axiom sdd status` son idénticos a los de un cambio con verificación conjunta hecha

#### Scenario: Un hallazgo de la verificación conjunta no bloquea
- **DADO** una verificación conjunta con un hallazgo crítico
- **CUANDO** se consulta `axiom sdd status` y se solicita el archivado
- **ENTONCES** `nextRecommended` y las dependencias no cambian
- **Y** el archivado no se rechaza por el informe, salvo la excepción de la matriz ISO de REQ-28.3

#### Scenario: En continuous no hay oferta ni parada nuevas
- **DADO** un cambio `continuous` con el rol `fullstack`
- **CUANDO** concluye el flujo
- **ENTONCES** el orquestador no presenta ninguna parada ni oferta de verificación conjunta
### Requirement: Juicio Explícito con Dos Jueces Ciegos y Rondas Acotadas (REQ-28.43)

Judgment Day DEBE seleccionarse de forma explícita: por petición de la persona o porque su preferencia de sesión incluye el juicio (`verify-judgment`, REQ-28.2); nunca por iniciativa propia del orquestador ni como efecto de un hito de entrega. Cuando se selecciona, DEBE ejecutar dos jueces distintos, ciegos entre sí y de solo lectura, cuyos resultados se contrastan con un acuerdo o confirmación explícitos antes de dar por corroborado un hallazgo; no se usa ningún refutador adicional. El juicio DEBE permitir como máximo dos rondas de arreglo y dos rejuicios acotados al alcance de los arreglos. Si tras el segundo rejuicio persiste un hallazgo severo confirmado, el veredicto DEBE ser `ESCALATED` y NO DEBE abrirse una tercera ronda. El juicio NO DEBE emitir recibos, ni depender de un estado de revisión nativo, ni invocar comandos `axiom review`, y su veredicto es informativo (REQ-28.3). Sus artefactos (`sdd/{change}/review-ledger`) son los propios de Judgment Day.
(Trasladado desde el requisito «Explicit Judgment Day replacement» de la capacidad `review-findings-ledger`, retirada con RDD (C-01-e, P20), sin su acoplamiento a la revisión 4R, a `review-refuter`, al estado nativo ni a los recibos.)

#### Scenario: El juicio no se lanza sin selección explícita

- **DADO** un rol con verify limpio y una preferencia de sesión `verify` o `none` y sin petición de la persona
- **CUANDO** concluye el `apply` del rol
- **ENTONCES** el orquestador no lanza el juicio
- **Y** no lo sustituye por ninguna otra revisión automática

#### Scenario: Dos jueces ciegos de solo lectura

- **DADO** un juicio seleccionado de forma explícita sobre el diff de un rol
- **CUANDO** se lanzan los jueces
- **ENTONCES** son dos ejecuciones distintas que no ven el veredicto una de otra
- **Y** ninguna puede modificar ficheros ni lanzar otro rol
- **Y** un hallazgo solo se da por corroborado cuando ambos coinciden o se confirma de forma explícita

#### Scenario: El juicio no supera la segunda ronda

- **DADO** un hallazgo severo que persiste tras el segundo rejuicio acotado
- **CUANDO** el orquestador evalúa el presupuesto de rondas
- **ENTONCES** el veredicto es `ESCALATED`
- **Y** no se abre una tercera ronda de arreglo ni de rejuicio

#### Scenario: El juicio no emite recibos ni usa comandos de revisión

- **DADO** un juicio `APPROVED` tras un verify limpio
- **CUANDO** el flujo continúa
- **ENTONCES** no se crea ningún recibo ni consentimiento
- **Y** no se invoca ningún comando `axiom review`
- **Y** el veredicto no condiciona fase, puerta ni archivado

## Supuestos a confirmar en design

- **A-1.** «Verify limpio» se define como sin hallazgos críticos ni comprobaciones obligatorias fallidas; un verify parcial no es limpio. Design fija cómo se determina mecánicamente (severidades del informe).
- **A-2.** La petición explícita del usuario prevalece sobre «solo tras verify limpio».
- **A-3.** En `continuous`, la preferencia `verify-judgment` autoriza las correcciones acotadas del juicio; en `checkpointed` se pide confirmación antes de la primera ronda.
- **A-4.** Design fija el nombre y la ubicación de los informes de cobertura y del juicio, y el número de revisores de cobertura.
