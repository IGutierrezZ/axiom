<!-- Especificación Viva generada a partir de '2026-09-22-inc-21-upfront-flow-governance' -->

# Especificación de Requerimientos: Compuertas de Revisión por Bloque (spec, design, tasks, apply) (INC-21)

## Propósito

Definir de forma rigurosa, ejecutable y verificable los requerimientos funcionales y escenarios BDD para las cuatro compuertas de revisión por bloque que operan cuando un cambio SDD se sella en modalidad "con paradas y reviews": validación de `spec` (cobertura funcional), validación de `design` (conformidad arquitectónica), validación de `tasks` (repartición y granularidad), y validación de `apply` por rol (calidad e implementación).

---

## 1. Capacidad: `sdd-block-review-gates`

Cuando el cambio se sella en modalidad "con paradas", el orquestador se detiene al concluir cada bloque del ciclo de vida SDD y exige una aprobación explícita antes de abrir la siguiente fase.

### Requirement: Aplicabilidad de las Compuertas Condicionada a la Modalidad Sellada (REQ-21.7)

Las compuertas descritas en REQ-21.8 a REQ-21.11 DEBEN aplicarse única y exclusivamente cuando la modalidad de avance sellada en el pre-vuelo (REQ-21.5) es "con paradas". Cuando la modalidad sellada es "continua", el orquestador NO DEBE detenerse entre fases ni presentar ninguna de esas compuertas, y DEBE encadenar las fases automáticamente en cuanto cada una cumpla sus propios criterios de completitud.

#### Scenario: Modalidad con paradas detiene el flujo entre spec y design

- **DADO** un cambio sellado con modalidad "con paradas"
- **CUANDO** concluye la fase `spec`
- **ENTONCES** el orquestador se detiene y presenta la compuerta de `spec` antes de iniciar `design`

#### Scenario: Modalidad continua encadena las fases sin compuertas

- **DADO** un cambio sellado con modalidad "continua"
- **CUANDO** concluye la fase `spec`
- **ENTONCES** el orquestador continúa directamente a `design` sin presentar ninguna compuerta de revisión por bloque

---

### Requirement: Compuerta de `spec` (REQ-21.8)

En modalidad "con paradas", al completarse la fase `spec` el orquestador DEBE detenerse y presentar al usuario el spec producido, evaluando explícitamente si cubre la intención original de la petición y si existen huecos o dudas abiertas, y DEBE ofrecer las tres salidas de REQ-26.1. El orquestador NO DEBE iniciar la fase `design` mientras esta compuerta no reciba una salida de aprobación (salida 2 o salida 3 de REQ-26.1). Con la salida 2 la fase `design` queda cedida a otra persona y el orquestador NO DEBE iniciarla en esa sesión (REQ-26.3).
(Previously: la compuerta solo admitía aprobación o rechazo y no preveía ceder la fase siguiente)

#### Scenario: Spec completo aprobado con la salida 3 habilita design

- **DADO** un spec recién producido que cubre la intención original sin huecos detectados
- **CUANDO** se presenta la compuerta de `spec`
- **Y** el usuario elige aprobar y seguir (salida 3)
- **ENTONCES** el orquestador inicia la fase `design`

#### Scenario: Spec aprobado con la salida 2 cede design y para

- **DADO** un spec recién producido que cubre la intención original sin huecos detectados
- **CUANDO** se presenta la compuerta de `spec`
- **Y** el usuario elige aprobar, subir y parar (salida 2)
- **ENTONCES** la compuerta de `spec` queda `approved` con la marca de cesión (REQ-26.3)
- **Y** el orquestador no inicia `design` en esa sesión y devuelve al usuario el destino de la PR y la orden (REQ-26.5)

#### Scenario: Huecos detectados se enumeran explícitamente en la compuerta

- **DADO** un spec que deja al menos un escenario funcional o caso borde sin cubrir
- **CUANDO** se presenta la compuerta de `spec`
- **ENTONCES** el orquestador enumera explícitamente los huecos o dudas abiertas detectados como parte de la presentación de la compuerta
- **Y** no inicia `design` hasta que el usuario elija una salida de aprobación, con o sin remediación previa de esos huecos

---

### Requirement: Compuerta de `design` (REQ-21.9)

En modalidad "con paradas", al completarse la fase `design` el orquestador DEBE detenerse y evaluar explícitamente si el diseño satisface todos los requerimientos del spec y si respeta las directrices de arquitectura, las tecnologías declaradas en `axiom.yaml` y los patrones existentes del repositorio, y DEBE ofrecer las tres salidas de REQ-26.1. El orquestador NO DEBE iniciar la fase `tasks` mientras esta compuerta no reciba una salida de aprobación (salida 2 o salida 3); con la salida 2 la fase `tasks` queda cedida a otra persona y el orquestador NO DEBE iniciarla en esa sesión. Con `technical_context.maintained` declarado, la compuerta DEBE además presentar el delta de contexto con el impacto previsto y evaluar si es coherente con el diseño; si el delta falta, DEBE señalar esa ausencia explícitamente como parte de la presentación (REQ-24.36).
(Previously: la compuerta solo admitía aprobación o rechazo y no preveía ceder la fase siguiente; la parte del delta de contexto no cambia)

#### Scenario: Diseño conforme aprobado con la salida 3 habilita tasks

- **DADO** un `design.md` que cubre todos los requerimientos del spec aprobado y respeta la arquitectura declarada
- **CUANDO** se presenta la compuerta de `design` y el usuario elige aprobar y seguir (salida 3)
- **ENTONCES** el orquestador inicia la fase `tasks`

#### Scenario: Diseño aprobado con la salida 2 cede tasks y para

- **DADO** un `design.md` conforme y confirmado en la rama del cambio
- **CUANDO** se presenta la compuerta de `design` y el usuario elige aprobar, subir y parar (salida 2)
- **ENTONCES** la compuerta de `design` queda `approved` con la marca de cesión
- **Y** el orquestador no inicia `tasks` en esa sesión

#### Scenario: Desviación arquitectónica se señala explícitamente en la compuerta

- **DADO** un `design.md` que introduce una tecnología o patrón no declarado en `axiom.yaml` ni presente en el repositorio
- **CUANDO** se presenta la compuerta de `design`
- **ENTONCES** el orquestador señala explícitamente esa desviación como parte de la presentación de la compuerta
- **Y** no inicia `tasks` hasta recibir una salida de aprobación

#### Scenario: El impacto previsto en el contexto mantenido se presenta en la compuerta

- **DADO** `technical_context.maintained` declarado y un diseño con su delta de contexto
- **CUANDO** se presenta la compuerta de `design`
- **ENTONCES** la presentación incluye el delta de contexto y evalúa su coherencia con el diseño
- **Y** no inicia `tasks` hasta recibir una salida de aprobación

#### Scenario: La ausencia del delta de contexto se señala en la compuerta

- **DADO** `technical_context.maintained` declarado y un diseño sin delta de contexto
- **CUANDO** se presenta la compuerta de `design`
- **ENTONCES** el orquestador señala explícitamente que falta el delta de contexto
- **Y** no asume «Sin impacto»

#### Scenario: Sin contexto mantenido la compuerta no cambia

- **DADO** un `axiom.yaml` sin `technical_context.maintained`
- **CUANDO** se presenta la compuerta de `design`
- **ENTONCES** la presentación no incluye ni exige delta de contexto

#### Scenario: Las plantillas del orquestador prescriben la presentación del delta en la compuerta de diseño

- **DADO** los textos de las plantillas de orquestador que describen la compuerta `design`
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** indican presentar el delta de contexto, y señalar su ausencia, cuando el status proyecta `technicalContext.maintained`

---

### Requirement: Compuerta de `tasks` (REQ-21.10)

En modalidad "con paradas", al completarse la fase `tasks` el orquestador DEBE presentar el documento `tasks.md` (o cada `tasks.<rol>.md` cuando existan varios roles) y DEBE detenerse antes de iniciar cualquier fase `apply`, evaluando si la repartición de responsabilidades entre roles es coherente y si el plan de trabajo es suficientemente granular y verificable, y DEBE ofrecer las tres salidas de REQ-26.1. El orquestador NO DEBE iniciar ninguna fase `apply` mientras esta compuerta no reciba una salida de aprobación (salida 2 o salida 3); con la salida 2 la fase `apply` de los roles queda cedida a otra persona y el orquestador NO DEBE iniciarla en esa sesión.
(Previously: la compuerta solo admitía aprobación o rechazo y no preveía ceder la fase siguiente)

#### Scenario: Tasks de un único rol aprobadas con la salida 3 habilitan apply

- **DADO** un cambio con el rol único `fullstack` y su `tasks.md` recién generado
- **CUANDO** se presenta la compuerta de `tasks`
- **Y** el usuario elige aprobar y seguir (salida 3)
- **ENTONCES** el orquestador inicia la fase `apply` del rol `fullstack`

#### Scenario: Tasks aprobadas con la salida 2 ceden apply y paran

- **DADO** un cambio con el rol único `fullstack`, su `tasks.md` recién generado y confirmado en la rama del cambio
- **CUANDO** se presenta la compuerta de `tasks` y el usuario elige aprobar, subir y parar (salida 2)
- **ENTONCES** la compuerta de `tasks` queda `approved` con la marca de cesión
- **Y** el orquestador no inicia `apply` en esa sesión

#### Scenario: Tasks multi-rol se presentan juntas antes de iniciar cualquier apply

- **DADO** un cambio con varios roles, cada uno con su propio `tasks.<rol>.md`
- **CUANDO** se presenta la compuerta de `tasks`
- **ENTONCES** el orquestador presenta la repartición de responsabilidades entre todos los roles a la vez, no solo el desglose de un rol individual
- **Y** no inicia `apply` para ningún rol hasta que el usuario elija una salida de aprobación

---

### Requirement: Compuerta de `apply` por Rol (REQ-21.11)

En modalidad "con paradas", al finalizar la implementación de cada rol activo (incluido el rol único `fullstack`), el orquestador DEBE detenerse y evaluar: conformidad con el diseño asignado a ese rol, cumplimiento de las funcionalidades de la `spec` asignadas a ese rol, y estándares de calidad (compilación limpia, pruebas unitarias ejecutadas, cobertura y estilo), y DEBE ofrecer las tres salidas de REQ-26.1. El orquestador NO DEBE marcar el rol como concluido, ni permitir que otro rol pendiente inicie su propia fase `apply`, mientras esta compuerta no reciba una salida de aprobación (salida 2 o salida 3). Con la salida 2 el rol queda concluido y lo que sigue (la fase `apply` de otro rol pendiente o, si era el último, la fase `verify`) queda cedido a otra persona: el orquestador NO DEBE iniciarlo en esa sesión. Con `technical_context.maintained` declarado, la compuerta DEBE además presentar el delta de contexto actualizado y evaluar que el impacto final refleja lo implementado por ese rol.
(Previously: la compuerta solo admitía aprobación o rechazo y no preveía ceder lo que sigue al rol)

#### Scenario: Rol concluido y aprobado con la salida 3 habilita el siguiente rol pendiente

- **DADO** un cambio con los roles `core` y `web`, donde `core` termina su implementación primero
- **CUANDO** se presenta la compuerta de `apply` del rol `core` y el usuario elige aprobar y seguir (salida 3)
- **ENTONCES** el orquestador marca `core` como concluido y habilita el inicio de la fase `apply` del rol `web`

#### Scenario: Rol concluido y aprobado con la salida 2 cede el siguiente rol

- **DADO** un cambio con los roles `core` y `web`, donde `core` termina su implementación primero y su trabajo está confirmado en la rama
- **CUANDO** se presenta la compuerta de `apply` del rol `core` y el usuario elige aprobar, subir y parar (salida 2)
- **ENTONCES** `core` queda concluido, su compuerta queda `approved` con la marca de cesión
- **Y** el orquestador no inicia la fase `apply` del rol `web` en esa sesión

#### Scenario: Último rol aprobado con la salida 2 cede verify

- **DADO** un cambio cuyo único rol pendiente es `web`
- **CUANDO** se aprueba la compuerta de `apply` de `web` con la salida 2
- **ENTONCES** el cambio queda con todos los `role-apply` aprobados y la fase `verify` cedida a otra persona
- **Y** el orquestador no inicia `verify` en esa sesión

#### Scenario: Incumplimiento de calidad bloquea la conclusión del rol

- **DADO** un rol cuya implementación no supera sus pruebas unitarias o no cumple la funcionalidad de spec asignada
- **CUANDO** se presenta la compuerta de `apply` de ese rol
- **ENTONCES** el orquestador la rechaza en el criterio de calidad correspondiente (salida 1) y no marca el rol como concluido
- **Y** ese rol no puede disparar el aviso de último rol (REQ-21.13) hasta que su compuerta se apruebe

#### Scenario: El impacto final en el contexto mantenido se presenta en la compuerta

- **DADO** `technical_context.maintained` declarado y un rol que termina su implementación con el delta de contexto actualizado
- **CUANDO** se presenta la compuerta de `apply` de ese rol
- **ENTONCES** la presentación incluye el delta de contexto actualizado y evalúa que refleja lo implementado por el rol
- **Y** no marca el rol como concluido hasta recibir una salida de aprobación

#### Scenario: Un impacto final no reflejado en el delta puede rechazarse

- **DADO** un rol cuya implementación cambia algo documentado en el contexto mantenido mientras el delta declara «Sin impacto»
- **CUANDO** se presenta la compuerta de `apply` de ese rol
- **ENTONCES** el orquestador señala la discrepancia como parte de la presentación
- **Y** el usuario puede elegir iterar (salida 1) para que se remedie el delta antes de reintentar el avance (REQ-21.12)

#### Scenario: Las plantillas del orquestador prescriben la presentación del delta en la compuerta de apply

- **DADO** los textos de las plantillas de orquestador que describen la compuerta `apply` por rol
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** indican presentar el delta de contexto actualizado cuando el status proyecta `technicalContext.maintained`

---

### Requirement: Rechazo de una Compuerta Bloquea el Avance y Exige Remediación (REQ-21.12)

Si el usuario elige iterar (salida 1 de REQ-26.1), es decir, rechaza o solicita cambios en cualquiera de las compuertas descritas en REQ-21.8 a REQ-21.11, el orquestador NO DEBE avanzar a la siguiente fase o rol, DEBE registrar el motivo del rechazo (obligatorio) como decisión `rejected`, y DEBE mantener el artefacto de la fase o rol actual abierto a remediación. Tras la remediación, el orquestador DEBE volver a presentar la misma compuerta, con sus tres salidas, antes de reintentar el avance. No hay límite de iteraciones. La reapertura por digest del artefacto tras un rechazo no cambia.
(Previously: el rechazo era la única alternativa a la aprobación y no se nombraba como una de las tres salidas)

#### Scenario: Rechazo de la compuerta de spec exige remediación antes de reintentar

- **DADO** que el usuario rechaza la compuerta de `spec` señalando un caso borde no cubierto
- **CUANDO** el orquestador registra el rechazo
- **ENTONCES** permanece en la fase `spec` para remediación y no inicia `design`
- **Y**, tras remediar el spec, vuelve a presentar la compuerta de `spec` antes de reintentar el avance

#### Scenario: Rechazo de la compuerta de apply de un rol impide el cierre de ese rol

- **DADO** que el usuario rechaza la compuerta de `apply` de un rol por incumplimiento de un requerimiento de spec
- **CUANDO** el orquestador registra el rechazo
- **ENTONCES** ese rol permanece abierto a remediación y no se marca como concluido
- **Y** no se dispara el aviso de último rol (REQ-21.13) para ese rol hasta que su compuerta de `apply` se apruebe explícitamente

#### Scenario: Iterar exige un motivo

- **DADO** una compuerta pendiente
- **CUANDO** se ejecuta `axiom sdd gate record --decision rejected` sin `--reason`
- **ENTONCES** la orden se rechaza y no se escribe nada en `gates.yaml`

#### Scenario: La remediación reabre la puerta por digest sin vocabulario nuevo

- **DADO** una compuerta de `design` con un registro `rejected` y el `design.md` modificado después del rechazo
- **CUANDO** se evalúa el estado de la compuerta
- **ENTONCES** vuelve a `pending` marcada como reabierta
- **Y** `axiom sdd status` vuelve a proyectar la pregunta tipada con sus tres opciones

### Requirement: Tres Salidas en Cada Parada de Planificación y de `role-apply` (REQ-26.1)

En modalidad "con paradas" (`execution_style: checkpointed`), cada parada de las compuertas `spec`, `design`, `tasks` y `role-apply:<rol>` DEBE ofrecer exactamente tres salidas, siempre en este orden y con la misma semántica:

1. **Iterar:** seguir trabajando sobre el artefacto actual. Se registra como decisión `rejected` con motivo (REQ-21.12).
2. **Aprobar, subir y parar:** se registra `approved` con la marca de cesión de la fase siguiente a otra persona, se suben los documentos del cambio con `axiom sdd publish` (REQ-27.12) y la sesión se detiene (REQ-26.3 a REQ-26.5).
3. **Aprobar y seguir:** se registra `approved` sin marca, se suben los documentos del cambio con `axiom sdd publish` (REQ-27.12) y el orquestador continúa él mismo con la fase siguiente.

La puerta `integration` NO es una parada de planificación: DEBE conservar sus dos salidas (aprobar con la evidencia de fusión exigida, o rechazar). En modalidad "continua" NO DEBE existir ninguna parada ni, por tanto, ninguna de las tres salidas (REQ-21.7). Las tres salidas son exclusivas de las paradas de SDD: el carril ODD no tiene paradas (REQ-27.17 de `organic-agent-trigger-rules`).

#### Scenario: Cada puerta de planificación ofrece las tres salidas

- **DADO** un cambio `checkpointed` con una compuerta pendiente de `spec`, `design`, `tasks` o `role-apply:<rol>`
- **CUANDO** se presenta la parada
- **ENTONCES** se ofrecen tres salidas en el orden iterar, aprobar-subir-y-parar, aprobar-y-seguir
- **Y** las tres se resuelven con `axiom sdd gate record`

#### Scenario: La puerta `integration` conserva dos salidas

- **DADO** un cambio con la puerta `integration` pendiente
- **CUANDO** se presenta su pregunta
- **ENTONCES** ofrece solo aprobar (con la evidencia de fusión) o rechazar
- **Y** `axiom sdd gate record --gate integration` con el indicador de la salida 2 se rechaza como no disponible para esa puerta, sin escribir en `gates.yaml`

#### Scenario: Modalidad continua no tiene paradas ni salidas

- **DADO** un cambio sellado con modalidad "continua"
- **CUANDO** concluye cualquier fase
- **ENTONCES** el orquestador encadena la fase siguiente sin presentar ninguna parada
- **Y** `axiom sdd status` no proyecta ninguna pregunta de compuerta

#### Scenario: Registrar la salida 2 en un cambio continuo se rechaza

- **DADO** un cambio sellado con modalidad "continua"
- **CUANDO** se ejecuta `axiom sdd gate record` con el indicador de la salida 2
- **ENTONCES** la orden se rechaza como no disponible porque el cambio no tiene paradas
- **Y** no se escribe en `gates.yaml`

#### Scenario: Aprobar y seguir no cede nada

- **DADO** una compuerta de `spec` pendiente
- **CUANDO** el usuario elige aprobar y seguir (salida 3)
- **ENTONCES** `gates.yaml` contiene un registro `approved` sin marca de cesión, idéntico al que se escribía antes de este cambio
- **Y** `axiom sdd status` no muestra ninguna cesión para esa puerta
- **Y** el orquestador publica los documentos según REQ-27.12

#### Scenario: Las tres salidas no existen en ODD

- **DADO** el carril ODD
- **CUANDO** el orquestador completa el documento de feature
- **ENTONCES** no presenta ninguna parada con las tres salidas

---

### Requirement: Pregunta Tipada de la Puerta con Tres Opciones sobre `axiom sdd gate record` (REQ-26.2)

Cuando `axiom sdd status` proyecta una compuerta pendiente de `spec`, `design`, `tasks` o `role-apply:<rol>` en un cambio `checkpointed`, la pregunta tipada (`gateQuestion`) DEBE tener exactamente tres opciones, en el orden de REQ-26.1. Las tres invocaciones DEBEN comenzar por `axiom sdd gate record` y compartir `--cwd`, `--change` y `--gate`: la salida 1 con `--decision rejected` y motivo; la salida 2 con `--decision approved` más el indicador de la salida 2 (nombre fijado en design); la salida 3 con `--decision approved` sin ese indicador. Cada opción DEBE tener un token de respuesta propio y distinto: los de las salidas 1 y 3 siguen siendo `rejected` y `approved`; el de la salida 2 lo fija design. La validación de esta pregunta DEBE rechazar un número de opciones que no sea tres (salvo el caso de REQ-26.6), opciones cuya invocación no comience por `axiom sdd gate record`, y dos opciones con el mismo token o la misma invocación. El sobre de autoridad de edición (`axiom.sdd-integration.consent/v1`) DEBE seguir exigiendo exactamente dos opciones y rechazando cualquier otro número. La pregunta DEBE relatarse de forma completa y sin alterar el orden, las etiquetas ni los efectos de las opciones (contrato de *Lossless Blocking Prompt*), con el dominio de respuesta formado por los tres tokens.

#### Scenario: La pregunta tiene tres opciones sobre el mismo verbo

- **DADO** una compuerta de `design` pendiente en un cambio `checkpointed`
- **CUANDO** se construye la pregunta tipada
- **ENTONCES** tiene tres opciones con los tokens de respuesta `rejected`, el de la salida 2 y `approved`, en ese orden
- **Y** las tres invocaciones comienzan por `axiom sdd gate record` con el mismo `--cwd`, `--change` y `--gate`
- **Y** las dos que usan `--decision approved` se distinguen únicamente por el indicador de la salida 2

#### Scenario: Una pregunta de compuerta con dos opciones o con una opción de otro verbo se rechaza

- **DADO** una pregunta de compuerta con exactamente dos opciones en un caso distinto de REQ-26.6, o con una opción cuya invocación no comienza por `axiom sdd gate record`
- **CUANDO** se valida
- **ENTONCES** la validación la rechaza

#### Scenario: Dos opciones con el mismo token se rechazan

- **DADO** una pregunta de compuerta con tres opciones de las que dos comparten el mismo token de respuesta
- **CUANDO** se valida
- **ENTONCES** la validación la rechaza

#### Scenario: El sobre de autoridad de edición sigue exigiendo dos opciones

- **DADO** el sobre `axiom.sdd-integration.consent/v1`
- **CUANDO** se valida con exactamente dos opciones
- **ENTONCES** es válido
- **Y** con tres opciones se rechaza

#### Scenario: El contrato de Lossless Blocking Prompt se cumple con tres opciones

- **DADO** la pregunta de una compuerta con sus tres opciones
- **CUANDO** el orquestador la relata al usuario
- **ENTONCES** conserva el orden, las etiquetas, los efectos y el dominio de respuesta de las tres opciones sin resumirlos ni fusionarlos
- **Y** una respuesta solo se acepta si coincide con exactamente una opción presentada

---

### Requirement: La Salida 2 Registra `approved` con la Marca de Cesión, Compatible con Binarios Anteriores (REQ-26.3)

La salida 2 DEBE registrarse con `axiom sdd gate record --decision approved` y el indicador de la salida 2, y DEBE añadir a `gates.yaml` un registro `approved` con un campo opcional que marca la fase siguiente como cedida a otra persona (nombre del campo fijado en design). El campo DEBE escribirse únicamente con la salida 2: los registros de las salidas 1 y 3 NO DEBEN contenerlo. El vocabulario de decisiones del registro sigue siendo `approved|rejected`: NO se añade ninguna decisión nueva. Un binario anterior a este cambio DEBE poder leer un `gates.yaml` con el campo y tratar la puerta como `approved` normal, sin error. La aprobación sigue siendo terminal: un registro posterior no reabre la puerta. El primer registro `approved` es el que cierra la puerta; un registro de salida 2 sobre una puerta que ya está `approved` DEBE rechazarse sin escribir, porque la cesión no sería visible. El motivo (`--reason`) es opcional en la salida 2. La marca de cesión indica que la fase siguiente la hace otra persona; NO implica que exista una PR fusionada ni exige ninguna operación de git previa.

#### Scenario: La salida 2 escribe un registro approved con la marca

- **DADO** una compuerta de `spec` pendiente
- **CUANDO** se ejecuta `axiom sdd gate record --gate spec --decision approved` con el indicador de la salida 2
- **ENTONCES** `gates.yaml` contiene un registro `approved` de `spec` con el campo de cesión
- **Y** la compuerta resulta `approved`

#### Scenario: Un binario anterior lee el registro con marca como approved

- **DADO** un `gates.yaml` con un registro `approved` que lleva el campo de cesión
- **CUANDO** lo decodifica el esquema previo a este cambio (decodificador no estricto)
- **ENTONCES** la decodificación termina sin error
- **Y** la puerta resulta `approved`

#### Scenario: Las salidas 1 y 3 no escriben la marca

- **DADO** una compuerta pendiente
- **CUANDO** se registra `--decision rejected` con motivo o `--decision approved` sin el indicador
- **ENTONCES** el registro resultante no contiene el campo de cesión

#### Scenario: El vocabulario del registro no cambia

- **DADO** un `gates.yaml` con una decisión distinta de `approved` y `rejected`
- **CUANDO** se lee con el esquema vigente
- **ENTONCES** se rechaza como hoy

#### Scenario: Un rechazo posterior no reabre una puerta aprobada con la salida 2

- **DADO** una compuerta de `design` aprobada con la salida 2
- **CUANDO** se registra después `--decision rejected` sobre esa puerta
- **ENTONCES** la puerta sigue `approved`
- **Y** `axiom sdd status` sigue proyectando la cesión del registro que la cerró

#### Scenario: La salida 2 sobre una puerta ya aprobada se rechaza

- **DADO** una compuerta de `spec` ya `approved` con la salida 3
- **CUANDO** se ejecuta `axiom sdd gate record --gate spec --decision approved` con el indicador de la salida 2
- **ENTONCES** la orden se rechaza indicando que la puerta ya está resuelta
- **Y** no se escribe en `gates.yaml`

#### Scenario: La cesión no exige una PR fusionada

- **DADO** una compuerta de `spec` aprobada con la salida 2 y publicada, sin PR abierta ni fusionada
- **CUANDO** otra persona consulta `axiom sdd status`
- **ENTONCES** la cesión se proyecta igual que antes y ningún estado exige una PR fusionada

---

### Requirement: La Salida 2 No Exige Artefactos Confirmados y Conserva la Guarda de Rama (REQ-26.4)

`axiom sdd gate record` con el indicador de la salida 2 NO DEBE exigir que los artefactos que juzga la puerta ni `gates.yaml` estén confirmados (commit) antes de registrar: la confirmación de los documentos la hace `axiom sdd publish` después de registrar (REQ-27.12). La guarda de rama vigente (`wrong_branch`) DEBE aplicarse antes de cualquier escritura y NO se debilita: fuera de la rama que corresponde la orden se rechaza con `wrong_branch` nombrando la rama esperada y sin escribir en `gates.yaml`. La salida 2 NO DEBE comprobar el trabajo de implementación de un rol: ese trabajo sin subir lo detecta `publish` (REQ-27.8 de `sdd-docs-publication`). Los antiguos rechazos por artefactos sin confirmar (`uncommitted_artifacts`) dejan de existir en `gate record`.

#### Scenario: Artefactos sin confirmar no impiden registrar

- **DADO** una compuerta de `spec` pendiente cuyo `spec.md` tiene modificaciones sin confirmar
- **CUANDO** se ejecuta la salida 2
- **ENTONCES** el registro se escribe en `gates.yaml`
- **Y** la orden no se rechaza por artefactos sin confirmar

#### Scenario: Un artefacto nuevo sin añadir a git no impide registrar

- **DADO** una compuerta de `spec` pendiente cuya delta spec nueva no está seguida por git
- **CUANDO** se ejecuta la salida 2
- **ENTONCES** el registro se escribe

#### Scenario: La salida 2 de un rol no comprueba el trabajo del rol

- **DADO** una compuerta `role-apply:core` pendiente con cambios de implementación sin confirmar
- **CUANDO** se ejecuta la salida 2
- **ENTONCES** el registro se escribe
- **Y** la orden no menciona los cambios de implementación

#### Scenario: Fuera de la rama que corresponde se rechaza con la guarda de rama

- **DADO** un monorepo con el repositorio desplegado en la rama de integración en lugar de la del cambio
- **CUANDO** se ejecuta la salida 2
- **ENTONCES** la orden se rechaza con `wrong_branch`
- **Y** no se escribe nada

---

### Requirement: La Salida 2 No Ejecuta Git en `gate record` y Entrega el Destino de la Publicación (REQ-26.5)

`axiom sdd gate record` NO DEBE ejecutar `git commit`, `push`, `fetch`, `pull`, `ls-remote` ni ninguna otra operación de git que modifique el repositorio o acceda al remoto, ni abrir la PR: la confirmación y la subida de los documentos las hace `axiom sdd publish` (REQ-27.12), y la PR y el merge siguen siendo de la persona. Tras registrar la salida 2, la orden DEBE devolver, en su salida de texto y en `--json` y en campos separados: la rama del cambio, el remoto de destino de la publicación (`origin`), el `prTarget` que reporta `axiom branch status --json` para esa rama y la orden de publicación `axiom sdd publish --change <x>`. El orquestador DEBE ejecutar esa publicación tras registrar la decisión (REQ-27.12) y DEBE detenerse después de la salida 2 sin iniciar la fase siguiente. La salida DEBE indicar que otra persona continúa tras `git fetch` (o `axiom branch ensure --fetch` con su autorización) y `axiom branch ensure`, que adopta la rama publicada (REQ-27.15), y que NO hace falta una PR fusionada. Según la topología (`workspace-topology`):

- **`monorepo-embedded`:** la publicación sube la rama del cambio a `origin`; la PR hacia `prTarget` y su merge son decisión de la persona y no condicionan la continuación.
- **`multirepo`, repositorio de specs:** se trabaja in situ en la rama de integración, sin rama de cambio ni PR; la publicación sube esa rama de integración (REQ-27.7 de `sdd-docs-publication`).

#### Scenario: Monorepo devuelve rama, remoto y orden de publicación

- **DADO** un monorepo con la rama `inc/x` y `prTarget` `main`
- **CUANDO** se registra la salida 2 de la compuerta `spec`
- **ENTONCES** la salida nombra la rama `inc/x`, el remoto `origin`, el destino `main` y la orden `axiom sdd publish --change x`
- **Y** indica que otra persona continúa con `git fetch` y `axiom branch ensure` sin necesidad de una PR fusionada

#### Scenario: Multirepo en el repositorio de specs publica la integración

- **DADO** un multirepo con el repositorio de specs en su rama de integración `main`
- **CUANDO** se registra la salida 2
- **ENTONCES** la salida indica que la publicación sube `main` a `origin`
- **Y** no menciona ninguna rama de cambio ni PR

#### Scenario: `gate record` no ejecuta git

- **DADO** un observador instalado en el único punto del producto que ejecuta git
- **CUANDO** se registra la salida 2
- **ENTONCES** el observador no registra ningún `commit`, `push`, `fetch`, `pull` ni `ls-remote`

#### Scenario: El orquestador publica y se detiene

- **DADO** una salida 2 recién registrada en la compuerta de `spec`
- **CUANDO** el orquestador ejecuta la publicación y presenta el resultado
- **ENTONCES** se detiene sin iniciar `design`
- **Y** devuelve el control a la persona

#### Scenario: La salida `--json` contiene los datos de la entrega

- **DADO** la salida 2 ejecutada con `--json`
- **CUANDO** se lee su salida
- **ENTONCES** contiene la rama, el remoto, el `prTarget` y la orden de publicación en campos separados, sin texto libre como único portador

---

### Requirement: Disponibilidad de la Salida 2 según el Almacén, el Estilo y la Puerta (REQ-26.6)

La salida 2 solo es real cuando hay ficheros que subir: DEBE estar disponible únicamente con el almacén de artefactos `openspec` o `hybrid`, en un cambio `checkpointed` y en las compuertas `spec`, `design`, `tasks` y `role-apply:<rol>`. Con el almacén `engram` o `none`, `axiom sdd gate record` con el indicador de la salida 2 DEBE rechazarse con un error que nombre el almacén y ofrezca como continuación las salidas 1 o 3, sin escribir en `gates.yaml`; y la pregunta tipada de la compuerta NO DEBE ofrecer la salida 2: presenta solo las salidas 1 y 3 y declara en su texto que la salida 2 no está disponible por el almacén declarado (supuesto A-3). Con esos almacenes la salida 3 aprueba sin publicar (REQ-27.12), porque `axiom sdd publish` tampoco está disponible (REQ-27.10 de `sdd-docs-publication`). El estilo continuo y la puerta `integration` rechazan la salida 2 según REQ-26.1.

#### Scenario: Almacén engram rechaza la salida 2

- **DADO** un cambio con el almacén declarado `engram`
- **CUANDO** se ejecuta `axiom sdd gate record` con el indicador de la salida 2
- **ENTONCES** la orden se rechaza nombrando el almacén `engram` y proponiendo las salidas 1 o 3
- **Y** `gates.yaml` no se modifica

#### Scenario: Almacén none rechaza la salida 2

- **DADO** un cambio con el almacén declarado `none`
- **CUANDO** se ejecuta la salida 2
- **ENTONCES** la orden se rechaza como no disponible

#### Scenario: La pregunta con almacén engram o none no ofrece la salida 2

- **DADO** una compuerta pendiente en un cambio con el almacén `engram` o `none`
- **CUANDO** se construye la pregunta tipada
- **ENTONCES** tiene dos opciones (iterar y aprobar y seguir)
- **Y** su texto declara que aprobar, subir y parar no está disponible por el almacén declarado

#### Scenario: Almacén hybrid y openspec ofrecen la salida 2

- **DADO** un cambio con el almacén `openspec` o `hybrid` y una compuerta pendiente de `tasks`
- **CUANDO** se construye la pregunta tipada
- **ENTONCES** tiene las tres opciones

#### Scenario: Con almacén engram la salida 3 aprueba sin publicar

- **DADO** un cambio con el almacén `engram` y una compuerta pendiente
- **CUANDO** se aprueba con la salida 3
- **ENTONCES** la decisión se registra y el orquestador continúa
- **Y** no se ejecuta `axiom sdd publish`

---

### Requirement: La Cesión es Visible en `axiom sdd status` y Quien la Recoge Empieza la Fase Siguiente (REQ-26.7)

`axiom sdd status` DEBE mostrar, en el JSON de `governance` y en el texto, que una compuerta `approved` lleva la marca de cesión y que la fase siguiente (o, en `role-apply:<rol>`, el siguiente rol o `verify`) queda cedida a otra persona (forma exacta en design). La marca se toma del registro `approved` que cierra la puerta. La cesión es información: la ruta proyectada hacia la fase siguiente DEBE ser la misma que tras una aprobación normal, y quien recoge el cambio empieza esa fase siguiente sin revisar de nuevo la fase anterior. La cesión NO DEBE bloquear ninguna fase ni ninguna orden. Un cambio sin marca de cesión DEBE proyectarse exactamente como antes de este cambio.

#### Scenario: Status muestra la cesión de spec

- **DADO** un cambio con la compuerta `spec` aprobada con la salida 2 y `design` sin empezar
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `governance` indica que `spec` está `approved` con la fase siguiente cedida
- **Y** la ruta siguiente es la fase `design`, igual que tras una aprobación normal

#### Scenario: Status muestra la cesión de un rol hacia verify

- **DADO** un cambio con todos los `role-apply` aprobados y el último con la marca de cesión
- **CUANDO** se ejecuta `axiom sdd status`
- **ENTONCES** el texto indica que `verify` queda cedida a otra persona

#### Scenario: Status muestra la cesión de un rol hacia otro rol

- **DADO** un cambio con los roles `core` y `web` y `role-apply:core` aprobado con la salida 2
- **CUANDO** se ejecuta `axiom sdd status`
- **ENTONCES** indica que el `apply` de `web` queda cedido a otra persona

#### Scenario: Sin marca no hay cesión

- **DADO** un cambio con todas sus puertas aprobadas con la salida 3
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** ninguna puerta indica cesión y la salida no cambia respecto a la anterior a este cambio

#### Scenario: La cesión no bloquea

- **DADO** una puerta aprobada con la salida 2
- **CUANDO** quien recoge el cambio ejecuta `axiom sdd status` y empieza la fase siguiente
- **ENTONCES** ninguna orden ni fase se bloquea por la marca de cesión

#### Scenario: La marca es la del registro que cierra la puerta

- **DADO** una puerta aprobada con la salida 2 y, después, un registro `approved` sin marca (salida 3) sobre la misma puerta
- **CUANDO** se ejecuta `axiom sdd status`
- **ENTONCES** la cesión mostrada es la del primer registro `approved`, el que cerró la puerta
---

### Requirement: Texto Único de las Tres Salidas Proyectado a Todos los Runtimes (REQ-26.8)

El texto que describe las tres salidas (iterar, aprobar-subir-y-parar, aprobar-y-seguir) y la regla de parar DEBE definirse en una sola fuente (`stop-exits.md`) y proyectarse sin diferencias a los orquestadores SDD de todos los runtimes, de modo que un pin de texto lo encuentre idéntico en cada uno. El texto es exclusivo de SDD: NO DEBE mencionar ODD ni proyectarse a la doctrina ODD generada (REQ-27.17 de `organic-agent-trigger-rules`). Ningún asset del producto (skills, orquestadores por runtime, comandos, guías) DEBE ofrecer «generar un relevo formal `handoff.md` o continuar» como pregunta de transición tras `spec` o `design`, ni nombrar `--handoff-policy` como argumento obligatorio de `kickoff seal` (REQ-26.9 de `sdd-preflight-configuration`). El texto DEBE indicar que `axiom sdd gate record` es el único modo de registrar la decisión, que toda aprobación (salidas 2 y 3) se completa con `axiom sdd publish`, que esa operación sube solo los documentos del cambio a la rama del cambio en `origin` (en el repositorio de specs de un multirepo, a su rama de integración), que el push de código, la PR y el merge los hace la persona, y que con almacén `engram` o `none` la salida 2 no está disponible.

#### Scenario: El texto de las tres salidas es idéntico en todos los runtimes

- **DADO** las plantillas de orquestador de todos los runtimes soportados
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** contienen el mismo texto de las tres salidas, procedente de la misma fuente

#### Scenario: Ningún asset ofrece el relevo `handoff.md` como transición

- **DADO** el conjunto de assets del producto
- **CUANDO** se inspecciona (prueba de pin)
- **ENTONCES** ninguno contiene la «Pregunta de Transición» que ofrece generar `handoff.md` o continuar
- **Y** `sdd-spec` y `sdd-design` remiten a las tres salidas de la compuerta

#### Scenario: El texto prescribe el significado de «subir»

- **DADO** el texto proyectado de las tres salidas
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica que toda aprobación se completa con `axiom sdd publish`, que sube solo los documentos del cambio a la rama del cambio en `origin`, y que el push de código, la PR y el merge los hace la persona
- **Y** indica que en el repositorio de specs de un multirepo se sube su rama de integración
- **Y** indica que con almacén `engram` o `none` la salida 2 no está disponible

#### Scenario: El texto de las salidas no menciona ODD

- **DADO** `stop-exits.md` y la doctrina ODD generada
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** `stop-exits.md` no menciona ODD
- **Y** la doctrina ODD generada no contiene el texto de las tres salidas
### Requirement: Aprobar una Parada Publica los Documentos y la Pregunta lo Declara (REQ-27.12)

En modalidad `checkpointed`, y solo con el almacén `openspec` o `hybrid`, toda aprobación de una parada de las compuertas `spec`, `design`, `tasks` y `role-apply:<rol>` (salida 2, «aprobar, subir y parar», y salida 3, «aprobar y seguir») DEBE ir seguida de `axiom sdd publish --change <x>` (capacidad `sdd-docs-publication`): el orquestador DEBE encadenar `axiom sdd gate record` y `publish`, en ese orden, y `gate record` NO DEBE ejecutar git (REQ-26.5). Las opciones de aprobación de la pregunta tipada (`gateQuestion`, REQ-26.2) DEBEN nombrar la operación, el remoto y la rama, p. ej. «aprobar y subir los documentos a `origin/<rama>` con las credenciales de git configuradas»; la aprobación de la parada constituye la autorización explícita del destino, la operación y la credencial de esa subida de documentación, y de ninguna otra: el push de código sigue siendo de la persona. La salida 1 (iterar) NO publica. Con la salida 2 el orquestador DEBE publicar y, después, detenerse sin iniciar la fase siguiente; con la salida 3 DEBE publicar y continuar. Una negativa de `publish` (REQ-27.8 y REQ-27.9 de `sdd-docs-publication`) NO anula la decisión ya registrada en `gates.yaml`: el orquestador DEBE presentarla como informativa, con su código tipado y su continuación, y (con la salida 3) continuar. Con el almacén `engram` o `none` la aprobación no publica (REQ-26.6). La puerta `integration` y el estilo `continuous` quedan fuera de esta regla (REQ-26.1, REQ-27.13).

#### Scenario: La salida 3 registra, publica y continúa

- **DADO** una compuerta de `spec` pendiente en un cambio `checkpointed` con almacén `hybrid` y documentación sin subir
- **CUANDO** la persona elige aprobar y seguir (salida 3)
- **ENTONCES** el orquestador ejecuta `axiom sdd gate record --decision approved` y después `axiom sdd publish --change <x>`
- **Y** `origin/<rama del cambio>` contiene el commit con `spec.md` y `gates.yaml`
- **Y** el orquestador continúa con la fase `design`

#### Scenario: La salida 2 registra con cesión, publica y se detiene

- **DADO** una compuerta de `design` pendiente
- **CUANDO** la persona elige aprobar, subir y parar (salida 2)
- **ENTONCES** se registra `approved` con la marca de cesión, se ejecuta `publish` y el orquestador se detiene
- **Y** no inicia la fase `tasks`

#### Scenario: La salida 1 no publica

- **DADO** una compuerta de `spec` pendiente
- **CUANDO** la persona elige iterar (salida 1)
- **ENTONCES** se registra `rejected` con motivo
- **Y** no se ejecuta `publish`

#### Scenario: La pregunta nombra operación, remoto y rama

- **DADO** una compuerta de `tasks` pendiente de un cambio cuya rama es `inc/x`
- **CUANDO** se construye la pregunta tipada
- **ENTONCES** las etiquetas o efectos de las salidas 2 y 3 nombran la subida de los documentos a `origin/inc/x`
- **Y** indican que el push de código sigue siendo de la persona

#### Scenario: Una negativa de `publish` no anula la aprobación

- **DADO** una compuerta `role-apply:core` pendiente y un commit de código sin subir en la rama
- **CUANDO** la persona aprueba con la salida 3 y `publish` se niega con `code_commits_unpublished`
- **ENTONCES** `gates.yaml` conserva el registro `approved`
- **Y** el orquestador presenta la negativa como informativa con la orden de push de código
- **Y** continúa con la fase siguiente

#### Scenario: La salida 2 con una negativa de `publish` se detiene igualmente

- **DADO** el mismo caso con la salida 2
- **CUANDO** `publish` se niega
- **ENTONCES** la aprobación con cesión sigue registrada
- **Y** el orquestador presenta la negativa y la orden de push, y se detiene sin iniciar la fase siguiente

#### Scenario: Con almacén engram o none no hay publicación

- **DADO** un cambio con el almacén `engram` o `none`
- **CUANDO** la persona aprueba con la salida 3
- **ENTONCES** no se ejecuta `publish`
- **Y** la aprobación se registra como hasta ahora

#### Scenario: La puerta `integration` no publica

- **DADO** la puerta `integration` pendiente
- **CUANDO** se aprueba con la evidencia de fusión
- **ENTONCES** no se ejecuta `publish`

#### Scenario: La doctrina del orquestador encadena ambas órdenes

- **DADO** los orquestadores SDD de todos los runtimes
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** todos prescriben ejecutar `axiom sdd gate record` seguido de `axiom sdd publish` tras una aprobación y tratar la negativa de `publish` como informativa

### Requirement: Las Paradas Presentan los Resultados de la Revisión Independiente cuando Existen (REQ-28.10)

En modalidad `checkpointed`, la parada de cada compuerta DEBE presentar, junto al artefacto que juzga y sin cambiar su semántica ni sus tres salidas (REQ-26.1), los resultados de revisión independiente que existan en ese momento: en `spec`, el informe de cobertura del spec; en `design`, el informe de cobertura de spec y design; en `role-apply:<rol>`, el informe vigente del verify de ese rol (con su sección «Cumplimiento técnico») y, si hubo juicio, su veredicto y los arreglos aplicados (REQ-28.8 de `sdd-independent-review`). La presentación DEBE enumerar los hallazgos abiertos y DEBE indicar explícitamente cuándo una revisión no se ha ejecutado, no ha respondido o no ha sido limpia; NO DEBE presentar como limpio un resultado ausente. Los resultados son informativos: NO DEBEN condicionar la aprobación, NO DEBEN registrarse en `gates.yaml`, ni su existencia o contenido DEBE cambiar el digest que juzga la puerta ni reabrirla. Cuando no existe ninguna revisión para esa puerta, la presentación NO cambia respecto a REQ-21.8, REQ-21.9 y REQ-21.11. Esta capacidad NO cambia REQ-21.7: en `continuous` no existen paradas, y los resultados se resumen al final del cambio según `sdd-independent-review`.

#### Scenario: La parada de spec presenta el informe de cobertura

- **DADO** un cambio `checkpointed` con un informe de cobertura del spec con dos huecos abiertos
- **CUANDO** se presenta la compuerta de `spec`
- **ENTONCES** la presentación enumera los dos huecos junto al spec
- **Y** ofrece las tres salidas sin alterarlas

#### Scenario: La parada de design presenta la cobertura de spec y design

- **DADO** un informe de cobertura posterior a `design`
- **CUANDO** se presenta la compuerta de `design`
- **ENTONCES** la presentación incluye sus hallazgos junto a la evaluación de conformidad de REQ-21.9
- **Y** con `technical_context.maintained` declarado, sigue presentando el delta de contexto

#### Scenario: La parada role-apply presenta el verify vigente y el juicio

- **DADO** un rol con verify limpio y un juicio `APPROVED` con un arreglo aplicado
- **CUANDO** se presenta la compuerta `role-apply:<rol>`
- **ENTONCES** la presentación incluye el informe vigente del verify, el veredicto del juicio y el arreglo aplicado
- **Y** la evaluación de conformidad con el diseño y la spec de REQ-21.11 se mantiene

#### Scenario: Una revisión no ejecutada se declara y no se presenta como limpia

- **DADO** un cambio con la preferencia `none` y sin ningún informe para el rol
- **CUANDO** se presenta la compuerta `role-apply:<rol>`
- **ENTONCES** la presentación no afirma que el verify sea limpio
- **Y** puede ofrecer lanzarlo antes de que el usuario elija una salida

#### Scenario: Un verify no limpio no impide aprobar

- **DADO** un verify del rol con un hallazgo crítico
- **CUANDO** el usuario elige una salida de aprobación en la parada
- **ENTONCES** la decisión se registra en `gates.yaml` como cualquier otra aprobación
- **Y** el hallazgo consta en la presentación pero no condiciona el registro

#### Scenario: El informe de cobertura no reabre la puerta

- **DADO** una puerta de `spec` con un registro `rejected` y un informe de cobertura generado después
- **CUANDO** se evalúa el estado de la puerta
- **ENTONCES** su digest, calculado sobre el artefacto, no ha cambiado
- **Y** no vuelve a `pending` por la presencia del informe

#### Scenario: Sin ninguna revisión la presentación no cambia

- **DADO** una puerta sin ningún informe de revisión independiente
- **CUANDO** se presenta
- **ENTONCES** la presentación es idéntica a la anterior a este cambio

#### Scenario: Las plantillas del orquestador prescriben la presentación de los resultados

- **DADO** los textos de las plantillas de orquestador que describen las compuertas `spec`, `design` y `role-apply`
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** indican presentar los resultados de revisión independiente cuando existen y declarar los ausentes
- **Y** no contienen vocabulario de RDD (REQ-28.11 de `sdd-independent-review`)
