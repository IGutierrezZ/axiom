<!-- Especificación Viva generada a partir de '2026-09-22-inc-21-upfront-flow-governance' -->

# Especificación de Requerimientos: Compuertas de Revisión por Bloque (spec, design, tasks, apply) (INC-21)

## Propósito

Definir de forma rigurosa, ejecutable y verificable los requerimientos funcionales y escenarios BDD para las cuatro compuertas de revisión por bloque que operan cuando un cambio SDD se sella en modalidad "con paradas y reviews": validación de `spec` (cobertura funcional), validación de `design` (conformidad arquitectónica), validación de `tasks` (repartición y granularidad), y validación de `apply` por rol (calidad e implementación).

---

## 1. Capacidad: `sdd-block-review-gates`

Cuando el cambio se sella en modalidad "con paradas", el orquestador se detiene al concluir cada bloque del ciclo de vida SDD y exige una aprobación explícita antes de abrir la siguiente fase.

### Requirement: Aplicabilidad de las Compuertas Condicionada a la Modalidad Sellada (RF-GAT-01)

Las compuertas descritas en RF-GAT-02 a RF-GAT-05 DEBEN aplicarse única y exclusivamente cuando la modalidad de avance sellada en el pre-vuelo (RF-PRE-01) es "con paradas". Cuando la modalidad sellada es "continua", el orquestador NO DEBE detenerse entre fases ni presentar ninguna de esas compuertas, y DEBE encadenar las fases automáticamente en cuanto cada una cumpla sus propios criterios de completitud.

#### Scenario: Modalidad con paradas detiene el flujo entre spec y design

- **DADO** un cambio sellado con modalidad "con paradas"
- **CUANDO** concluye la fase `spec`
- **ENTONCES** el orquestador se detiene y presenta la compuerta de `spec` antes de iniciar `design`

#### Scenario: Modalidad continua encadena las fases sin compuertas

- **DADO** un cambio sellado con modalidad "continua"
- **CUANDO** concluye la fase `spec`
- **ENTONCES** el orquestador continúa directamente a `design` sin presentar ninguna compuerta de revisión por bloque

---

### Requirement: Compuerta de `spec` (RF-GAT-02)

En modalidad "con paradas", al completarse la fase `spec` el orquestador DEBE detenerse y presentar al usuario el spec producido, evaluando explícitamente si cubre la intención original de la petición y si existen huecos o dudas abiertas, y DEBE ofrecer las tres salidas de RF-GAT-07. El orquestador NO DEBE iniciar la fase `design` mientras esta compuerta no reciba una salida de aprobación (salida 2 o salida 3 de RF-GAT-07). Con la salida 2 la fase `design` queda cedida a otra persona y el orquestador NO DEBE iniciarla en esa sesión (RF-GAT-09).
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
- **ENTONCES** la compuerta de `spec` queda `approved` con la marca de cesión (RF-GAT-09)
- **Y** el orquestador no inicia `design` en esa sesión y devuelve al usuario el destino de la PR y la orden (RF-GAT-11)

#### Scenario: Huecos detectados se enumeran explícitamente en la compuerta

- **DADO** un spec que deja al menos un escenario funcional o caso borde sin cubrir
- **CUANDO** se presenta la compuerta de `spec`
- **ENTONCES** el orquestador enumera explícitamente los huecos o dudas abiertas detectados como parte de la presentación de la compuerta
- **Y** no inicia `design` hasta que el usuario elija una salida de aprobación, con o sin remediación previa de esos huecos

---

### Requirement: Compuerta de `design` (RF-GAT-03)

En modalidad "con paradas", al completarse la fase `design` el orquestador DEBE detenerse y evaluar explícitamente si el diseño satisface todos los requerimientos del spec y si respeta las directrices de arquitectura, las tecnologías declaradas en `axiom.yaml` y los patrones existentes del repositorio, y DEBE ofrecer las tres salidas de RF-GAT-07. El orquestador NO DEBE iniciar la fase `tasks` mientras esta compuerta no reciba una salida de aprobación (salida 2 o salida 3); con la salida 2 la fase `tasks` queda cedida a otra persona y el orquestador NO DEBE iniciarla en esa sesión. Con `technical_context.maintained` declarado, la compuerta DEBE además presentar el delta de contexto con el impacto previsto y evaluar si es coherente con el diseño; si el delta falta, DEBE señalar esa ausencia explícitamente como parte de la presentación (RF-CTX-11).
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

### Requirement: Compuerta de `tasks` (RF-GAT-04)

En modalidad "con paradas", al completarse la fase `tasks` el orquestador DEBE presentar el documento `tasks.md` (o cada `tasks.<rol>.md` cuando existan varios roles) y DEBE detenerse antes de iniciar cualquier fase `apply`, evaluando si la repartición de responsabilidades entre roles es coherente y si el plan de trabajo es suficientemente granular y verificable, y DEBE ofrecer las tres salidas de RF-GAT-07. El orquestador NO DEBE iniciar ninguna fase `apply` mientras esta compuerta no reciba una salida de aprobación (salida 2 o salida 3); con la salida 2 la fase `apply` de los roles queda cedida a otra persona y el orquestador NO DEBE iniciarla en esa sesión.
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

### Requirement: Compuerta de `apply` por Rol (RF-GAT-05)

En modalidad "con paradas", al finalizar la implementación de cada rol activo (incluido el rol único `fullstack`), el orquestador DEBE detenerse y evaluar: conformidad con el diseño asignado a ese rol, cumplimiento de las funcionalidades de la `spec` asignadas a ese rol, y estándares de calidad (compilación limpia, pruebas unitarias ejecutadas, cobertura y estilo), y DEBE ofrecer las tres salidas de RF-GAT-07. El orquestador NO DEBE marcar el rol como concluido, ni permitir que otro rol pendiente inicie su propia fase `apply`, mientras esta compuerta no reciba una salida de aprobación (salida 2 o salida 3). Con la salida 2 el rol queda concluido y lo que sigue (la fase `apply` de otro rol pendiente o, si era el último, la fase `verify`) queda cedido a otra persona: el orquestador NO DEBE iniciarlo en esa sesión. Con `technical_context.maintained` declarado, la compuerta DEBE además presentar el delta de contexto actualizado y evaluar que el impacto final refleja lo implementado por ese rol.
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
- **Y** ese rol no puede disparar el aviso de último rol (RF-RCL-01) hasta que su compuerta se apruebe

#### Scenario: El impacto final en el contexto mantenido se presenta en la compuerta

- **DADO** `technical_context.maintained` declarado y un rol que termina su implementación con el delta de contexto actualizado
- **CUANDO** se presenta la compuerta de `apply` de ese rol
- **ENTONCES** la presentación incluye el delta de contexto actualizado y evalúa que refleja lo implementado por el rol
- **Y** no marca el rol como concluido hasta recibir una salida de aprobación

#### Scenario: Un impacto final no reflejado en el delta puede rechazarse

- **DADO** un rol cuya implementación cambia algo documentado en el contexto mantenido mientras el delta declara «Sin impacto»
- **CUANDO** se presenta la compuerta de `apply` de ese rol
- **ENTONCES** el orquestador señala la discrepancia como parte de la presentación
- **Y** el usuario puede elegir iterar (salida 1) para que se remedie el delta antes de reintentar el avance (RF-GAT-06)

#### Scenario: Las plantillas del orquestador prescriben la presentación del delta en la compuerta de apply

- **DADO** los textos de las plantillas de orquestador que describen la compuerta `apply` por rol
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** indican presentar el delta de contexto actualizado cuando el status proyecta `technicalContext.maintained`

---

### Requirement: Rechazo de una Compuerta Bloquea el Avance y Exige Remediación (RF-GAT-06)

Si el usuario elige iterar (salida 1 de RF-GAT-07), es decir, rechaza o solicita cambios en cualquiera de las compuertas descritas en RF-GAT-02 a RF-GAT-05, el orquestador NO DEBE avanzar a la siguiente fase o rol, DEBE registrar el motivo del rechazo (obligatorio) como decisión `rejected`, y DEBE mantener el artefacto de la fase o rol actual abierto a remediación. Tras la remediación, el orquestador DEBE volver a presentar la misma compuerta, con sus tres salidas, antes de reintentar el avance. No hay límite de iteraciones. La reapertura por digest del artefacto tras un rechazo no cambia.
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
- **Y** no se dispara el aviso de último rol (RF-RCL-01) para ese rol hasta que su compuerta de `apply` se apruebe explícitamente

#### Scenario: Iterar exige un motivo

- **DADO** una compuerta pendiente
- **CUANDO** se ejecuta `axiom sdd gate record --decision rejected` sin `--reason`
- **ENTONCES** la orden se rechaza y no se escribe nada en `gates.yaml`

#### Scenario: La remediación reabre la puerta por digest sin vocabulario nuevo

- **DADO** una compuerta de `design` con un registro `rejected` y el `design.md` modificado después del rechazo
- **CUANDO** se evalúa el estado de la compuerta
- **ENTONCES** vuelve a `pending` marcada como reabierta
- **Y** `axiom sdd status` vuelve a proyectar la pregunta tipada con sus tres opciones

### Requirement: Tres Salidas en Cada Parada de Planificación y de `role-apply` (RF-GAT-07)

En modalidad "con paradas" (`execution_style: checkpointed`), cada parada de las compuertas `spec`, `design`, `tasks` y `role-apply:<rol>` DEBE ofrecer exactamente tres salidas, siempre en este orden y con la misma semántica:

1. **Iterar:** seguir trabajando sobre el artefacto actual. Se registra como decisión `rejected` con motivo (RF-GAT-06).
2. **Aprobar, subir y parar:** se registra `approved` con la marca de cesión de la fase siguiente a otra persona, se suben los documentos del cambio con `axiom sdd publish` (RF-GAT-15) y la sesión se detiene (RF-GAT-09 a RF-GAT-11).
3. **Aprobar y seguir:** se registra `approved` sin marca, se suben los documentos del cambio con `axiom sdd publish` (RF-GAT-15) y el orquestador continúa él mismo con la fase siguiente.

La puerta `integration` NO es una parada de planificación: DEBE conservar sus dos salidas (aprobar con la evidencia de fusión exigida, o rechazar). En modalidad "continua" NO DEBE existir ninguna parada ni, por tanto, ninguna de las tres salidas (RF-GAT-01). Las tres salidas son exclusivas de las paradas de SDD: el carril ODD no tiene paradas (RF-RUT-18 de `organic-agent-trigger-rules`).

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
- **Y** el orquestador publica los documentos según RF-GAT-15

#### Scenario: Las tres salidas no existen en ODD

- **DADO** el carril ODD
- **CUANDO** el orquestador completa el documento de feature
- **ENTONCES** no presenta ninguna parada con las tres salidas

---

### Requirement: Pregunta Tipada de la Puerta con Tres Opciones sobre `axiom sdd gate record` (RF-GAT-08)

Cuando `axiom sdd status` proyecta una compuerta pendiente de `spec`, `design`, `tasks` o `role-apply:<rol>` en un cambio `checkpointed`, la pregunta tipada (`gateQuestion`) DEBE tener exactamente tres opciones, en el orden de RF-GAT-07. Las tres invocaciones DEBEN comenzar por `axiom sdd gate record` y compartir `--cwd`, `--change` y `--gate`: la salida 1 con `--decision rejected` y motivo; la salida 2 con `--decision approved` más el indicador de la salida 2 (nombre fijado en design); la salida 3 con `--decision approved` sin ese indicador. Cada opción DEBE tener un token de respuesta propio y distinto: los de las salidas 1 y 3 siguen siendo `rejected` y `approved`; el de la salida 2 lo fija design. La validación de esta pregunta DEBE rechazar un número de opciones que no sea tres (salvo el caso de RF-GAT-12), opciones cuya invocación no comience por `axiom sdd gate record`, y dos opciones con el mismo token o la misma invocación. El sobre de autoridad de edición (`axiom.sdd-integration.consent/v1`) DEBE seguir exigiendo exactamente dos opciones y rechazando cualquier otro número. La pregunta DEBE relatarse de forma completa y sin alterar el orden, las etiquetas ni los efectos de las opciones (contrato de *Lossless Blocking Prompt*), con el dominio de respuesta formado por los tres tokens.

#### Scenario: La pregunta tiene tres opciones sobre el mismo verbo

- **DADO** una compuerta de `design` pendiente en un cambio `checkpointed`
- **CUANDO** se construye la pregunta tipada
- **ENTONCES** tiene tres opciones con los tokens de respuesta `rejected`, el de la salida 2 y `approved`, en ese orden
- **Y** las tres invocaciones comienzan por `axiom sdd gate record` con el mismo `--cwd`, `--change` y `--gate`
- **Y** las dos que usan `--decision approved` se distinguen únicamente por el indicador de la salida 2

#### Scenario: Una pregunta de compuerta con dos opciones o con una opción de otro verbo se rechaza

- **DADO** una pregunta de compuerta con exactamente dos opciones en un caso distinto de RF-GAT-12, o con una opción cuya invocación no comienza por `axiom sdd gate record`
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

### Requirement: La Salida 2 Registra `approved` con la Marca de Cesión, Compatible con Binarios Anteriores (RF-GAT-09)

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

### Requirement: La Salida 2 No Exige Artefactos Confirmados y Conserva la Guarda de Rama (RF-GAT-10)

`axiom sdd gate record` con el indicador de la salida 2 NO DEBE exigir que los artefactos que juzga la puerta ni `gates.yaml` estén confirmados (commit) antes de registrar: la confirmación de los documentos la hace `axiom sdd publish` después de registrar (RF-GAT-15). La guarda de rama vigente (`wrong_branch`) DEBE aplicarse antes de cualquier escritura y NO se debilita: fuera de la rama que corresponde la orden se rechaza con `wrong_branch` nombrando la rama esperada y sin escribir en `gates.yaml`. La salida 2 NO DEBE comprobar el trabajo de implementación de un rol: ese trabajo sin subir lo detecta `publish` (RF-PUB-04 de `sdd-docs-publication`). Los antiguos rechazos por artefactos sin confirmar (`uncommitted_artifacts`) dejan de existir en `gate record`.

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

### Requirement: La Salida 2 No Ejecuta Git en `gate record` y Entrega el Destino de la Publicación (RF-GAT-11)

`axiom sdd gate record` NO DEBE ejecutar `git commit`, `push`, `fetch`, `pull`, `ls-remote` ni ninguna otra operación de git que modifique el repositorio o acceda al remoto, ni abrir la PR: la confirmación y la subida de los documentos las hace `axiom sdd publish` (RF-GAT-15), y la PR y el merge siguen siendo de la persona. Tras registrar la salida 2, la orden DEBE devolver, en su salida de texto y en `--json` y en campos separados: la rama del cambio, el remoto de destino de la publicación (`origin`), el `prTarget` que reporta `axiom branch status --json` para esa rama y la orden de publicación `axiom sdd publish --change <x>`. El orquestador DEBE ejecutar esa publicación tras registrar la decisión (RF-GAT-15) y DEBE detenerse después de la salida 2 sin iniciar la fase siguiente. La salida DEBE indicar que otra persona continúa tras `git fetch` (o `axiom branch ensure --fetch` con su autorización) y `axiom branch ensure`, que adopta la rama publicada (RF-BRA-27), y que NO hace falta una PR fusionada. Según la topología (`workspace-topology`):

- **`monorepo-embedded`:** la publicación sube la rama del cambio a `origin`; la PR hacia `prTarget` y su merge son decisión de la persona y no condicionan la continuación.
- **`multirepo`, repositorio de specs:** se trabaja in situ en la rama de integración, sin rama de cambio ni PR; la publicación sube esa rama de integración (RF-PUB-03 de `sdd-docs-publication`).

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

### Requirement: Disponibilidad de la Salida 2 según el Almacén, el Estilo y la Puerta (RF-GAT-12)

La salida 2 solo es real cuando hay ficheros que subir: DEBE estar disponible únicamente con el almacén de artefactos `openspec` o `hybrid`, en un cambio `checkpointed` y en las compuertas `spec`, `design`, `tasks` y `role-apply:<rol>`. Con el almacén `engram` o `none`, `axiom sdd gate record` con el indicador de la salida 2 DEBE rechazarse con un error que nombre el almacén y ofrezca como continuación las salidas 1 o 3, sin escribir en `gates.yaml`; y la pregunta tipada de la compuerta NO DEBE ofrecer la salida 2: presenta solo las salidas 1 y 3 y declara en su texto que la salida 2 no está disponible por el almacén declarado (supuesto A-3). Con esos almacenes la salida 3 aprueba sin publicar (RF-GAT-15), porque `axiom sdd publish` tampoco está disponible (RF-PUB-06 de `sdd-docs-publication`). El estilo continuo y la puerta `integration` rechazan la salida 2 según RF-GAT-07.

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

### Requirement: La Cesión es Visible en `axiom sdd status` y Quien la Recoge Empieza la Fase Siguiente (RF-GAT-13)

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

### Requirement: Texto Único de las Tres Salidas Proyectado a Todos los Runtimes (RF-GAT-14)

El texto que describe las tres salidas (iterar, aprobar-subir-y-parar, aprobar-y-seguir) y la regla de parar DEBE definirse en una sola fuente (`stop-exits.md`) y proyectarse sin diferencias a los orquestadores SDD de todos los runtimes, de modo que un pin de texto lo encuentre idéntico en cada uno. El texto es exclusivo de SDD: NO DEBE mencionar ODD ni proyectarse a la doctrina ODD generada (RF-RUT-18 de `organic-agent-trigger-rules`). Ningún asset del producto (skills, orquestadores por runtime, comandos, guías) DEBE ofrecer «generar un relevo formal `handoff.md` o continuar» como pregunta de transición tras `spec` o `design`, ni nombrar `--handoff-policy` como argumento obligatorio de `kickoff seal` (RF-PRE-03 de `sdd-preflight-configuration`). El texto DEBE indicar que `axiom sdd gate record` es el único modo de registrar la decisión, que toda aprobación (salidas 2 y 3) se completa con `axiom sdd publish`, que esa operación sube solo los documentos del cambio a la rama del cambio en `origin` (en el repositorio de specs de un multirepo, a su rama de integración), que el push de código, la PR y el merge los hace la persona, y que con almacén `engram` o `none` la salida 2 no está disponible.

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
### Requirement: Aprobar una Parada Publica los Documentos y la Pregunta lo Declara (RF-GAT-15)

En modalidad `checkpointed`, y solo con el almacén `openspec` o `hybrid`, toda aprobación de una parada de las compuertas `spec`, `design`, `tasks` y `role-apply:<rol>` (salida 2, «aprobar, subir y parar», y salida 3, «aprobar y seguir») DEBE ir seguida de `axiom sdd publish --change <x>` (capacidad `sdd-docs-publication`): el orquestador DEBE encadenar `axiom sdd gate record` y `publish`, en ese orden, y `gate record` NO DEBE ejecutar git (RF-GAT-11). Las opciones de aprobación de la pregunta tipada (`gateQuestion`, RF-GAT-08) DEBEN nombrar la operación, el remoto y la rama, p. ej. «aprobar y subir los documentos a `origin/<rama>` con las credenciales de git configuradas»; la aprobación de la parada constituye la autorización explícita del destino, la operación y la credencial de esa subida de documentación, y de ninguna otra: el push de código sigue siendo de la persona. La salida 1 (iterar) NO publica. Con la salida 2 el orquestador DEBE publicar y, después, detenerse sin iniciar la fase siguiente; con la salida 3 DEBE publicar y continuar. Una negativa de `publish` (RF-PUB-04 y RF-PUB-05 de `sdd-docs-publication`) NO anula la decisión ya registrada en `gates.yaml`: el orquestador DEBE presentarla como informativa, con su código tipado y su continuación, y (con la salida 3) continuar. Con el almacén `engram` o `none` la aprobación no publica (RF-GAT-12). La puerta `integration` y el estilo `continuous` quedan fuera de esta regla (RF-GAT-07, RF-PUB-08).

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

### Requirement: Las Paradas Presentan los Resultados de la Revisión Independiente cuando Existen (RF-GAT-16)

En modalidad `checkpointed`, la parada de cada compuerta DEBE presentar, junto al artefacto que juzga y sin cambiar su semántica ni sus tres salidas (RF-GAT-07), los resultados de revisión independiente que existan en ese momento: en `spec`, el informe de cobertura del spec; en `design`, el informe de cobertura de spec y design; en `role-apply:<rol>`, el informe vigente del verify de ese rol (con su sección «Cumplimiento técnico») y, si hubo juicio, su veredicto y los arreglos aplicados (RF-REV-08 de `sdd-independent-review`). La presentación DEBE enumerar los hallazgos abiertos y DEBE indicar explícitamente cuándo una revisión no se ha ejecutado, no ha respondido o no ha sido limpia; NO DEBE presentar como limpio un resultado ausente. Los resultados son informativos: NO DEBEN condicionar la aprobación, NO DEBEN registrarse en `gates.yaml`, ni su existencia o contenido DEBE cambiar el digest que juzga la puerta ni reabrirla. Cuando no existe ninguna revisión para esa puerta, la presentación NO cambia respecto a RF-GAT-02, RF-GAT-03 y RF-GAT-05. Esta capacidad NO cambia RF-GAT-01: en `continuous` no existen paradas, y los resultados se resumen al final del cambio según `sdd-independent-review`.
(Previously: el último escenario remitía a REQ-28.11 de `sdd-independent-review` («Frontera con RDD»), requisito que se retira con RDD (C-01-e); la ausencia de ese vocabulario pasa a RF-ORQ-25 de `sdd-orchestrator-assets`.)

#### Scenario: La parada de spec presenta el informe de cobertura

- **DADO** un cambio `checkpointed` con un informe de cobertura del spec con dos huecos abiertos
- **CUANDO** se presenta la compuerta de `spec`
- **ENTONCES** la presentación enumera los dos huecos junto al spec
- **Y** ofrece las tres salidas sin alterarlas

#### Scenario: La parada de design presenta la cobertura de spec y design

- **DADO** un informe de cobertura posterior a `design`
- **CUANDO** se presenta la compuerta de `design`
- **ENTONCES** la presentación incluye sus hallazgos junto a la evaluación de conformidad de RF-GAT-03
- **Y** con `technical_context.maintained` declarado, sigue presentando el delta de contexto

#### Scenario: La parada role-apply presenta el verify vigente y el juicio

- **DADO** un rol con verify limpio y un juicio `APPROVED` con un arreglo aplicado
- **CUANDO** se presenta la compuerta `role-apply:<rol>`
- **ENTONCES** la presentación incluye el informe vigente del verify, el veredicto del juicio y el arreglo aplicado
- **Y** la evaluación de conformidad con el diseño y la spec de RF-GAT-05 se mantiene

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
- **Y** no contienen `axiom review` ni vocabulario de revisión por recibos (RF-ORQ-25 de `sdd-orchestrator-assets`)
### Requirement: Aprobar `spec`, `design` o `role-apply` Emite su Evento de Integración (RF-GAT-17)

`axiom sdd gate record` con `--decision approved` DEBE despachar un evento de integración (RF-INT-06) DESPUÉS de persistir la decisión en `gates.yaml` y antes de cualquier código que pueda devolver error: `spec` emite `gate.spec.approved`, `design` emite `gate.design.approved` y `role-apply:<rol>` emite `role.apply.approved` con el rol extraído del sufijo. El evento DEBE llevar el cambio, el actor registrado (RF-ACT-01) y la clave de idempotencia de RF-INT-07. La emisión NUNCA DEBE hacer fallar el verbo ni cambiar su código de salida ni su salida (RF-INT-10). Una decisión `rejected` NO DEBE emitir nada.

(Placement: after RF-GAT-16)

#### Scenario: Aprobar `spec`
- **DADO** el doble configurado y un cambio con kickoff sellado
- **CUANDO** se ejecuta `gate record --gate spec --decision approved`
- **ENTONCES** `gates.yaml` contiene la aprobación
- **Y** el doble recibe `gate.spec.approved` con el actor registrado y su clave de idempotencia

#### Scenario: Aprobar `design`
- **DADO** el doble configurado
- **CUANDO** se aprueba `design`
- **ENTONCES** el doble recibe `gate.design.approved`

#### Scenario: Aprobar `role-apply` de un rol
- **DADO** el doble configurado y el rol `backend` sellado
- **CUANDO** se ejecuta `gate record --gate role-apply:backend --decision approved`
- **ENTONCES** el doble recibe `role.apply.approved` con el rol `backend`

#### Scenario: Rechazar no emite
- **DADO** el doble configurado
- **CUANDO** se ejecuta `gate record --gate spec --decision rejected --reason x`
- **ENTONCES** el doble no recibe ningún evento y no se crea ningún fichero de integración

#### Scenario: El fallo de la integración no rompe la aprobación
- **DADO** el doble configurado para fallar
- **CUANDO** se aprueba `spec`
- **ENTONCES** el verbo termina con código 0 y la salida de siempre
- **Y** la aprobación consta en `gates.yaml` y la operación queda pendiente

#### Scenario: Reaprobar un artefacto cambiado
- **DADO** `spec` aprobada y ya entregada, y después un artefacto modificado
- **CUANDO** se aprueba `spec` de nuevo
- **ENTONCES** el doble recibe un evento con una clave de idempotencia distinta de la primera

### Requirement: `tasks` e `integration` no Emiten y el Contrato `gate-record/v1` no Cambia (RF-GAT-18)

La aprobación de las compuertas `tasks` e `integration` NO DEBE despachar ningún evento. El contrato `axiom.sdd-gate-record/v1` (campos de la salida JSON y de texto) NO DEBE cambiar por la integración: los avisos van únicamente a stderr.

(Placement: after RF-GAT-17)

#### Scenario: `tasks` no emite
- **DADO** el doble configurado
- **CUANDO** se aprueba `tasks`
- **ENTONCES** el doble no recibe ningún evento

#### Scenario: `integration` no emite
- **DADO** el doble configurado
- **CUANDO** se aprueba `integration`
- **ENTONCES** el doble no recibe ningún evento

#### Scenario: Salida JSON idéntica con y sin integración
- **DADO** la misma aprobación ejecutada con y sin plugin configurado
- **CUANDO** se ejecuta con `--json`
- **ENTONCES** el JSON de stdout es idéntico, con el mismo `schema` `axiom.sdd-gate-record/v1`

### Requirement: Vocabulario Cerrado del Modo de Revisión por Commit (RF-GAT-19)

El modo de revisión por commit DEBE ser un vocabulario cerrado de tres tokens canónicos, que no se traducen: `notify` (modo 1: avisar tras cada tarea y continuar con commit automático; es el modo por defecto), `stop` (modo 2: parar tras cada tarea con commit automático) y `stop-before-commit` (modo 3: parar tras cada tarea sin hacer el commit hasta que la persona apruebe). Cualquier otro valor, incluida la cadena vacía y las variantes de mayúsculas, DEBE rechazarse. La ausencia del modo equivale a `notify`.

#### Scenario: Los tres tokens se aceptan

- **DADO** una puerta `tasks` pendiente de un cambio `checkpointed`
- **CUANDO** se registra la aprobación con cada uno de `notify`, `stop` y `stop-before-commit`
- **ENTONCES** cada registro se acepta y conserva el token tal cual

#### Scenario: Un valor fuera del vocabulario se rechaza

- **DADO** una puerta `tasks` pendiente de un cambio `checkpointed`
- **CUANDO** se registra la aprobación con el modo `Stop`, `pause` o la cadena vacía
- **ENTONCES** la orden se rechaza indicando el vocabulario válido
- **Y** no se escribe en `gates.yaml`

### Requirement: `gate record` Persiste el Modo Solo en la Aprobación de la Puerta `tasks` de un Cambio `checkpointed` (RF-GAT-20)

`axiom sdd gate record` DEBE aceptar la bandera `--commit-review <modo>` únicamente con `--gate tasks` y `--decision approved` (con o sin `--hand-off`), en un cambio cuyo estilo sellado es `checkpointed`. Cuando se acepta, el registro `approved` DEBE persistir el modo en un campo opcional `commit_review` del registro de `gates.yaml`, escrito en la misma llamada que aprueba. El campo NO DEBE escribirse cuando la bandera está ausente. La orden DEBE rechazarse sin escribir en `gates.yaml` cuando la bandera se combina con otra puerta (`spec`, `design`, `role-apply:<rol>`, `integration`), con `--decision rejected`, con un cambio `continuous`, o con una puerta `tasks` que ya está `approved` (el primer registro `approved` cierra la puerta). La salida JSON de la orden DEBE incluir el modo cuando se persiste. El kickoff sellado y el digest de las puertas NO DEBEN cambiar.

#### Scenario: La aprobación con modo escribe el campo

- **DADO** un cambio `checkpointed` con la puerta `tasks` pendiente
- **CUANDO** se ejecuta `axiom sdd gate record --gate tasks --decision approved --commit-review stop`
- **ENTONCES** `gates.yaml` contiene un único registro `approved` de `tasks` con `commit_review: stop`
- **Y** la puerta resulta `approved`

#### Scenario: El modo viaja junto con la cesión en un único registro

- **DADO** un cambio `checkpointed` con la puerta `tasks` pendiente
- **CUANDO** se ejecuta `gate record --gate tasks --decision approved --hand-off --commit-review stop-before-commit`
- **ENTONCES** `gates.yaml` contiene un único registro `approved` con la marca de cesión y con `commit_review: stop-before-commit`

#### Scenario: Sin la bandera no se escribe el campo

- **DADO** un cambio `checkpointed` con la puerta `tasks` pendiente
- **CUANDO** se ejecuta `gate record --gate tasks --decision approved` sin `--commit-review`
- **ENTONCES** el registro resultante no contiene `commit_review`

#### Scenario: Otra puerta rechaza la bandera

- **DADO** un cambio `checkpointed` con la puerta `design` pendiente
- **CUANDO** se ejecuta `gate record --gate design --decision approved --commit-review notify`
- **ENTONCES** la orden se rechaza con un error que indica que el modo solo se admite en la puerta `tasks`
- **Y** no se escribe en `gates.yaml`

#### Scenario: Un rechazo no admite el modo

- **DADO** un cambio `checkpointed` con la puerta `tasks` pendiente
- **CUANDO** se ejecuta `gate record --gate tasks --decision rejected --reason "x" --commit-review notify`
- **ENTONCES** la orden se rechaza sin escribir en `gates.yaml`

#### Scenario: Un cambio `continuous` rechaza la bandera

- **DADO** un cambio sellado `continuous`
- **CUANDO** se ejecuta `gate record --gate tasks --decision approved --commit-review stop`
- **ENTONCES** la orden se rechaza indicando que el modo solo existe en cambios `checkpointed`
- **Y** no se escribe en `gates.yaml`

#### Scenario: Una puerta `tasks` ya aprobada rechaza un segundo registro con modo

- **DADO** una puerta `tasks` ya `approved` sin modo
- **CUANDO** se ejecuta `gate record --gate tasks --decision approved --commit-review stop`
- **ENTONCES** la orden se rechaza indicando que la puerta ya está resuelta
- **Y** no se escribe en `gates.yaml` y el modo efectivo sigue siendo `notify`

#### Scenario: El kickoff y el digest no cambian

- **DADO** una puerta `tasks` con su digest calculado
- **CUANDO** se aprueba con `--commit-review stop`
- **ENTONCES** `kickoff.yaml` no se modifica
- **Y** el digest de la puerta es idéntico al que habría con la aprobación sin modo

### Requirement: Compatibilidad Hacia Atrás del Campo `commit_review` (RF-GAT-21)

El campo `commit_review` DEBE ser opcional en la lectura y en la escritura. Un `gates.yaml` anterior a este cambio, sin el campo, DEBE leerse sin error y su modo efectivo DEBE ser `notify`. Un binario anterior a este cambio DEBE poder leer un `gates.yaml` con el campo y tratar la puerta como `approved` normal, sin error. Un `commit_review` presente en un registro con un valor fuera del vocabulario DEBE rechazarse al leer el esquema vigente, como cualquier valor inválido del registro. El vocabulario de decisiones del registro sigue siendo `approved|rejected`.

#### Scenario: Un ledger antiguo se lee como `notify`

- **DADO** un `gates.yaml` con la puerta `tasks` aprobada y sin `commit_review`
- **CUANDO** se calcula el estado de las puertas
- **ENTONCES** el modo efectivo de la puerta `tasks` es `notify`

#### Scenario: Un decodificador anterior ignora el campo

- **DADO** un `gates.yaml` con un registro `approved` de `tasks` que lleva `commit_review: stop`
- **CUANDO** lo decodifica el esquema previo a este cambio (decodificador no estricto)
- **ENTONCES** la decodificación termina sin error
- **Y** la puerta resulta `approved`

#### Scenario: Un valor inválido persistido se rechaza al leer

- **DADO** un `gates.yaml` con `commit_review: always`
- **CUANDO** se lee con el esquema vigente
- **ENTONCES** se rechaza como un valor inválido del registro

### Requirement: `axiom sdd status` Proyecta el Modo de Revisión por Commit (RF-GAT-22)

`axiom sdd status` DEBE proyectar en la vista de la puerta `tasks` el modo del registro `approved` que la cierra, en un campo `commitReview` junto al resto de los campos de la puerta, cuando el cambio es `checkpointed` y la puerta está `approved`; con un registro sin el campo DEBE proyectar `notify`. Mientras la puerta no esté `approved`, en las demás puertas y en un cambio `continuous` el campo NO DEBE aparecer, y quien consume el estado DEBE tratar su ausencia como `notify`. El modo DEBE sobrevivir a la salida 2 (`--hand-off`): quien recoge `apply` lo lee de `status`. La proyección es informativa: NO DEBE bloquear ni cambiar `nextRecommended` ni las dependencias. El orquestador DEBE pasar el modo proyectado a `sdd-apply`.

#### Scenario: Se proyecta el modo registrado

- **DADO** un cambio `checkpointed` con la puerta `tasks` aprobada con `commit_review: stop`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** la puerta `tasks` contiene `commitReview: "stop"`

#### Scenario: Sin campo se proyecta `notify`

- **DADO** un cambio `checkpointed` con la puerta `tasks` aprobada sin `commit_review`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** la puerta `tasks` contiene `commitReview: "notify"`

#### Scenario: Sobrevive al traspaso

- **DADO** una puerta `tasks` aprobada con `--hand-off --commit-review stop-before-commit`
- **CUANDO** otra persona ejecuta `axiom sdd status --json`
- **ENTONCES** la puerta proyecta a la vez la cesión y `commitReview: "stop-before-commit"`

#### Scenario: No aparece donde no aplica

- **DADO** una puerta `tasks` pendiente, o la puerta `design` aprobada, o un cambio `continuous`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** ninguna de esas vistas contiene `commitReview`

#### Scenario: La proyección no cambia la ruta

- **DADO** la misma puerta `tasks` aprobada con y sin modo
- **CUANDO** se compara `nextRecommended` y las dependencias de ambos estados
- **ENTONCES** son idénticos

### Requirement: El Modo se Pregunta en una Interacción Propia de la Puerta `tasks` y Solo en `checkpointed` (RF-GAT-23)

En un cambio `checkpointed`, cuando la persona elige en la puerta `tasks` una salida de aprobación (la 2 o la 3), la doctrina DEBE indicar al orquestador que formule el modo de revisión por commit en una interacción propia, separada de la pregunta tipada de la puerta y sujeta al contrato de bloqueo sin pérdida, con exactamente los tres tokens de RF-GAT-19 (con descripción de su efecto traducida a la lengua de la conversación; los tokens no se traducen) y `notify` como valor por defecto declarado. Después DEBE registrar la aprobación una sola vez, con el modo, en la misma llamada de `gate record` que aprueba. Con la salida 1 (iterar) NO DEBE preguntarse el modo. En un cambio `continuous` NO DEBE preguntarse ni escribirse el modo. En ODD NO DEBE preguntarse. La pregunta tipada de la puerta DEBE seguir con exactamente tres opciones y el mismo texto (RF-GAT-08) y NO DEBE incluir ninguna opción del modo. Cómo se compone la invocación de la salida elegida con `--commit-review` es decisión de design; el comportamiento observable es que el resto de los tokens de la invocación no se altera.

#### Scenario: La salida de aprobación dispara la pregunta del modo

- **DADO** un cambio `checkpointed` en la puerta `tasks`
- **CUANDO** la persona elige la salida 2 o la salida 3
- **ENTONCES** el orquestador presenta una pregunta aparte con los tres modos
- **Y** registra una sola aprobación que incluye el modo elegido

#### Scenario: Iterar no pregunta

- **DADO** un cambio `checkpointed` en la puerta `tasks`
- **CUANDO** la persona elige la salida 1 (iterar)
- **ENTONCES** no se pregunta el modo y no se escribe nada en `gates.yaml`

#### Scenario: `continuous` no pregunta

- **DADO** un cambio sellado `continuous`
- **CUANDO** concluye la fase `tasks`
- **ENTONCES** no se pregunta el modo y el comportamiento es `notify`

#### Scenario: La pregunta tipada conserva tres opciones

- **DADO** la proyección de la pregunta tipada de la puerta `tasks`
- **CUANDO** se inspeccionan sus opciones
- **ENTONCES** son exactamente tres, las mismas de RF-GAT-08, y ninguna menciona el modo de revisión por commit

#### Scenario: El texto de la doctrina fija la interacción

- **DADO** la sección «Commit Review» proyectada a un runtime
- **CUANDO** se lee la regla de la puerta `tasks`
- **ENTONCES** declara que el modo se pregunta tras una salida de aprobación, en interacción propia, una sola vez y con la misma llamada de registro, y que no se pregunta en `continuous` ni con la salida 1

### Requirement: El Modo es Inmutable tras la Aprobación de `tasks` (RF-GAT-24)

Una vez aprobada la puerta `tasks`, el modo NO DEBE poder cambiarse por `gate record`: la aprobación es terminal (RF-GAT-09) y un registro posterior se rechaza o no cambia la puerta. Cambiar el modo en medio de `apply` queda fuera de este requisito.

#### Scenario: Un rechazo posterior no cambia el modo

- **DADO** una puerta `tasks` aprobada con `commit_review: stop`
- **CUANDO** se registra después `--decision rejected`
- **ENTONCES** la puerta sigue `approved` y `status` sigue proyectando `commitReview: "stop"`

### Requirement: La Ayuda de `gate record` Documenta la Bandera (RF-GAT-25)

La ayuda de `axiom sdd gate record` DEBE describir `--commit-review`, su vocabulario, que solo se admite con `--gate tasks --decision approved` en un cambio `checkpointed` y que su ausencia equivale a `notify`.

#### Scenario: La ayuda menciona la bandera

- **DADO** el comando `axiom sdd gate record --help`
- **CUANDO** se lee su salida
- **ENTONCES** contiene `--commit-review`, los tres tokens y la restricción a la puerta `tasks`
### Requirement: La Presentación de `integration` Declara el Estado de la Verificación Conjunta como Información (RF-GAT-26)

En un cambio `checkpointed`, la presentación de la compuerta `integration` DEBE declarar siempre exactamente uno de estos tres estados informativos de la verificación conjunta (RF-REV-10 de `sdd-independent-review`), consultando `axiom branch scope`:

1. **Hecha (fecha de la última ronda):** `verify-report.md` contiene al menos una ronda de la sección «Verificación conjunta»; la declaración indica la fecha de la última ronda. No se calcula vigencia: se muestra la fecha y decide la persona.
2. **Ofrecida:** no hay ninguna ronda y el resumen de `axiom branch scope` indica `offer: offerable`; el orquestador la ofrece y espera la decisión sin lanzarla.
3. **No verificable (motivo):** no hay ninguna ronda y la oferta no es `offerable`. El motivo es el de cada repositorio `unverifiable` (`branch_missing`, `base_missing` o `fork_point_undetermined`), «ramas pendientes de fusionar» cuando `offer` es `pending_merge`, o «sin commits propios que verificar» cuando es `nothing_to_verify`. Cuando la propia consulta falla (salida distinta de cero, rechazo tipado, salida ilegible), el estado es «no verificable» con el código del rechazo o «consulta no disponible».

La declaración es información: NO DEBE bloquear ni retrasar la presentación de `integration` (que se emite igualmente), NO DEBE alterar sus dos salidas, la evidencia de fusión que exige, `nextRecommended` ni las dependencias, y NO DEBE persistirse en `kickoff.yaml`, `gates.yaml` ni `axiom sdd status`. Aprobar `integration` sin verificación conjunta sigue siendo válido. Una petición explícita de la persona se atiende siempre (RF-REV-10). En un cambio `continuous` no existe la compuerta y esta declaración no se presenta.

#### Scenario: Estado hecha con la fecha de la última ronda
- **DADO** un `verify-report.md` con dos rondas de «Verificación conjunta», la última fechada el 2026-11-24
- **CUANDO** el orquestador presenta `integration`
- **ENTONCES** declara «hecha (2026-11-24)»
- **Y** no calcula si la ronda sigue vigente

#### Scenario: Estado ofrecida
- **DADO** ninguna ronda y `offer` igual a `offerable`
- **CUANDO** el orquestador presenta `integration`
- **ENTONCES** declara «ofrecida» y espera la decisión de la persona sin lanzarla

#### Scenario: Estado no verificable con ramas pendientes
- **DADO** ninguna ronda y un cambio multirrepo con la rama de `web` sin fusionar en `develop`
- **CUANDO** el orquestador presenta `integration`
- **ENTONCES** declara «no verificable (ramas pendientes de fusionar)» con la rama de `web`

#### Scenario: Estado no verificable por rama borrada
- **DADO** ninguna ronda y un repositorio `unverifiable` por `branch_missing`
- **CUANDO** el orquestador presenta `integration`
- **ENTONCES** declara «no verificable (branch_missing)» con el mensaje «la rama ya no está o falta un fetch autorizado»
- **Y** no ejecuta `fetch`

#### Scenario: La consulta de alcance falla
- **DADO** una ejecución de `axiom branch scope` que termina con un rechazo tipado, con salida ilegible o con salida distinta de cero
- **CUANDO** el orquestador presenta `integration`
- **ENTONCES** declara «no verificable» con el código del rechazo o «consulta no disponible»
- **Y** presenta la compuerta con sus dos salidas sin bloquearla

#### Scenario: Aprobar integration sin verificación conjunta sigue siendo posible
- **DADO** una compuerta `integration` pendiente, con la verificación conjunta no hecha
- **CUANDO** se registra `axiom sdd gate record --gate integration --decision approved` con la evidencia de fusión actual
- **ENTONCES** el registro se acepta con el mismo contrato `axiom.sdd-gate-record/v1`
- **Y** `nextRecommended` y las dependencias de `axiom sdd status` son idénticos a los que proyecta sin esa declaración

#### Scenario: En continuous no hay declaración
- **DADO** un cambio `continuous` con rol único `fullstack`
- **CUANDO** concluye el flujo
- **ENTONCES** no existe compuerta `integration` ni se presenta ninguna declaración de verificación conjunta
### Requirement: Contrato del HTML Explicativo que Generan `sdd-spec` y `sdd-design` cuando el Kickoff Sella `yes` (RF-GAT-27)

Cuando el kickoff del cambio sella `explainer_html: yes`, la fase `sdd-spec` DEBE escribir, después de los md de spec, el fichero `resumen-spec.html`, y la fase `sdd-design` DEBE escribir, después de `design.md`, el fichero `resumen-design.html`, ambos en la raíz de la carpeta del cambio (uno por cambio, no uno por dominio). Cada HTML DEBE:

- estar escrito en castellano, para público no técnico, e incluir qué cambia, para quién, los flujos antes y después, diagramas SVG en línea y un glosario;
- ser autocontenido: HTML, CSS y SVG en línea, sin ninguna etiqueta `<script>` y sin ningún recurso externo (ni URL `http://` o `https://` en `src`, `href` o `url(...)`, ni hojas de estilo, fuentes, imágenes o marcos externos);
- llevar `<meta name="axiom-source-digest" content="...">` con el mismo digest que `axiom` calcula para los md de origen de esa fase (los de spec para `resumen-spec.html`, `design.md` para `resumen-design.html`), de modo que su frescura sea verificable (RF-GAT-30).

Cuando el kickoff sella `no` o no tiene el campo, las fases NO DEBEN escribir estos HTML y su comportamiento NO DEBE cambiar. El HTML es exclusivo de SDD: ninguna fase de ODD, ni `sdd-propose`, `sdd-tasks` o `sdd-verify`, lo genera. La doctrina de las skills (y de sus copias por agente) DEBE exigir estas restricciones y una comprobación automática DEBE detectar si la doctrina deja de exigirlas.

#### Scenario: Con `yes`, `sdd-spec` escribe `resumen-spec.html` tras el md

- **DADO** un cambio con `explainer_html: yes` y la fase `sdd-spec` en curso
- **CUANDO** la fase termina
- **ENTONCES** existe `resumen-spec.html` en la raíz de la carpeta del cambio, además de los md de spec
- **Y** contiene `<meta name="axiom-source-digest">`

#### Scenario: Con `yes`, `sdd-design` escribe `resumen-design.html`

- **DADO** un cambio con `explainer_html: yes` y la fase `sdd-design` en curso
- **CUANDO** la fase termina
- **ENTONCES** existe `resumen-design.html` en la raíz de la carpeta del cambio, además de `design.md`

#### Scenario: El HTML es autocontenido

- **DADO** un `resumen-spec.html` o `resumen-design.html` generado
- **CUANDO** se inspecciona su contenido
- **ENTONCES** no contiene ninguna etiqueta `<script>`
- **Y** no referencia ninguna URL externa ni recurso externo
- **Y** se abre sin red con doble clic

#### Scenario: Con `no` o sin campo no se genera

- **DADO** un cambio sellado con `explainer_html: no`, o sin el campo
- **CUANDO** terminan `sdd-spec` y `sdd-design`
- **ENTONCES** no existe ningún `resumen-*.html` en la carpeta del cambio
- **Y** las fases producen los mismos md que antes de este cambio

#### Scenario: Las skills exigen las restricciones

- **DADO** las skills `sdd-spec` y `sdd-design` y sus copias por agente
- **CUANDO** se ejecuta la comprobación automática de la doctrina
- **ENTONCES** cada una exige la autocontención (sin `<script>` ni recursos externos), el meta `axiom-source-digest`, el castellano y la cita de IDs
- **Y** la comprobación falla si alguna exigencia desaparece

---

### Requirement: Contenido Fiel del HTML: Cita de IDs, Nada Fuera del md y Aviso de que se Aprueba el md (RF-GAT-28)

Cada bloque del HTML DEBE citar el ID del requisito (en `resumen-spec.html`) o de la decisión (en `resumen-design.html`) del md del que sale. El HTML NO DEBE afirmar nada que no esté en el md de origen. El HTML DEBE decir expresamente, de forma visible, que lo que se aprueba en la puerta es el md y no el HTML. Cuando una puerta se rechaza y la fase regenera el md (RF-GAT-06), DEBE regenerar también el HTML correspondiente, con el digest del md regenerado.

#### Scenario: Cada bloque cita su ID

- **DADO** un `resumen-spec.html` generado para un spec con dos requisitos con ID válido según el patrón del proyecto
- **CUANDO** se inspecciona el HTML
- **ENTONCES** cada bloque explicativo cita el ID del requisito del que procede
- **Y** todos los IDs citados existen en el md de origen

#### Scenario: Aviso de que se aprueba el md

- **DADO** un HTML explicativo generado
- **CUANDO** se inspecciona su texto visible
- **ENTONCES** declara que lo que se aprueba es el md y que el HTML es una ayuda de lectura

#### Scenario: Tras un rechazo se regeneran md y HTML

- **DADO** un cambio con `explainer_html: yes` cuya puerta `spec` o `design` se rechaza
- **CUANDO** la fase remedia y regenera el md
- **ENTONCES** regenera también el HTML correspondiente
- **Y** su `axiom-source-digest` coincide con el digest del md regenerado

---

### Requirement: El HTML Queda Fuera del Digest de las Puertas (RF-GAT-29)

Crear, modificar o borrar un `resumen-spec.html` o un `resumen-design.html` NO DEBE cambiar el digest que juzgan las puertas `spec` y `design`, NO DEBE reabrir una puerta ya aprobada ni cambiar su estado, y NO DEBE entrar en el conjunto de artefactos que una puerta aprueba. La puerta aprueba el md, que es el contrato.

#### Scenario: Crear el HTML no cambia el digest

- **DADO** un cambio con las puertas `spec` y `design` en cualquier estado y sin HTML
- **CUANDO** se crea `resumen-spec.html` y `resumen-design.html` y se recalcula el digest de cada puerta
- **ENTONCES** los dos digests son idénticos a los de antes

#### Scenario: Modificar el HTML no reabre una puerta aprobada

- **DADO** las puertas `spec` y `design` aprobadas
- **CUANDO** se modifica el contenido de `resumen-spec.html`
- **ENTONCES** ambas puertas siguen aprobadas
- **Y** el digest de cada una no cambia

#### Scenario: Modificar el md sí cambia el digest

- **DADO** una puerta `spec` aprobada
- **CUANDO** se modifica un md de spec
- **ENTONCES** el digest de la puerta cambia, como antes de este cambio

---

### Requirement: Frescura Informativa `fresh`, `stale` o `missing` del HTML en `axiom sdd status` (RF-GAT-30)

Cuando el kickoff del cambio sella `explainer_html: yes`, `axiom sdd status --json` DEBE proyectar, para `spec` y para `design`, un estado de frescura de su HTML con tres valores, calculado de solo lectura sobre los md de origen que ya descubre el status (sin una regla paralela de descubrimiento de specs) y con independencia del estilo de ejecución, también en `continuous`:

- `missing`: el HTML de ese artefacto no existe.
- `stale`: el HTML existe pero su `axiom-source-digest` no coincide con el digest actual de los md de origen, o no tiene la etiqueta `<meta name="axiom-source-digest">`, o esta no tiene un valor legible.
- `fresh`: el HTML existe y su `axiom-source-digest` coincide con el digest actual de los md de origen.

Con `explainer_html: no` o sin el campo no DEBE proyectarse ningún estado de frescura. Un artefacto sin md de origen todavía no proyecta estado. La frescura es solo informativa: NO DEBE cambiar `nextRecommended`, las dependencias, los motivos de bloqueo ni el digest de ninguna puerta, ni bloquear una fase, una puerta o el archivado (tampoco en un cambio ISO). Con el almacén de artefactos `engram` o `none` no se proyecta. El contrato publicado (`sdd-status-contract.md`) DEBE documentar la proyección.

#### Scenario: HTML al día es `fresh`

- **DADO** un cambio con `explainer_html: yes`, `spec.md` y un `resumen-spec.html` cuyo `axiom-source-digest` coincide con el digest actual
- **CUANDO** se ejecuta `axiom sdd status --change <x> --json`
- **ENTONCES** la frescura de `spec` es `fresh`

#### Scenario: Modificar el md marca el HTML como `stale`

- **DADO** el escenario anterior
- **CUANDO** se modifica un md de spec sin regenerar el HTML
- **ENTONCES** la frescura de `spec` es `stale`
- **Y** `nextRecommended` y las dependencias son idénticos a los de antes de modificarlo

#### Scenario: Sin HTML es `missing`

- **DADO** un cambio con `explainer_html: yes` y `design.md` sin `resumen-design.html`
- **CUANDO** se consulta el status
- **ENTONCES** la frescura de `design` es `missing`
- **Y** `nextRecommended` no cambia

#### Scenario: HTML sin meta de digest es `stale`

- **DADO** un `resumen-spec.html` que no contiene `<meta name="axiom-source-digest">`, o cuyo `content` está vacío
- **CUANDO** se consulta el status
- **ENTONCES** la frescura de `spec` es `stale`

#### Scenario: Se proyecta también en `continuous`

- **DADO** un cambio `continuous` con `explainer_html: yes` y un HTML desactualizado
- **CUANDO** se consulta el status
- **ENTONCES** la frescura es `stale`
- **Y** no se detiene ninguna fase por ello

#### Scenario: Con `no` o sin campo no se proyecta

- **DADO** un cambio con `explainer_html: no` o sin el campo, aunque exista un `resumen-spec.html`
- **CUANDO** se consulta el status
- **ENTONCES** no se proyecta ningún estado de frescura

#### Scenario: Un cambio ISO no se bloquea por la frescura

- **DADO** un cambio ISO con `explainer_html: yes` y un HTML `stale` o `missing`
- **CUANDO** se consulta el status
- **ENTONCES** ningún motivo de bloqueo cita la frescura
- **Y** el archivado no se condiciona a ella
### Requirement: `pruebas.md` Entra en el Digest de la Compuerta `tasks` solo Cuando Existe, con Cálculo Único en las Dos Rutas (RF-GAT-31)

El digest de la compuerta `tasks` DEBE incluir `pruebas.md` del cambio únicamente cuando el fichero existe. El conjunto de ficheros del digest (los `tasks*.md` más `pruebas.md` si existe) DEBE resolverse con una única función compartida por las dos rutas, la de `gate record` (`cli/sdd_gate_digest.go`) y la del status (`sddstatus/governance.go`), de modo que el digest de `axiom sdd gate record` y el de `axiom sdd status` coincidan siempre. Un cambio sin `pruebas.md` DEBE conservar exactamente el digest que tenía antes de este cambio, con la función activa o no. Editar `pruebas.md` DEBE producir un digest distinto.

La reapertura sigue la regla de `tasks.md` y NO la amplía: una compuerta `tasks` `rejected` vuelve a `pending` (reabierta) cuando cambia el digest, y una compuerta `approved` es terminal y sigue `approved` aunque `pruebas.md` se edite después (D-08), para no bloquear a `sdd-apply` contra sus propias casillas.

**Desvío documentado del plan C-06-c:** el criterio literal «editar tras aprobar reabre la puerta» se sustituye por dos garantías verificables: editar `pruebas.md` cambia el digest, y una compuerta rechazada se reabre al editar solo `pruebas.md`.

La existencia de `pruebas.md` y su cobertura NO DEBEN alterar `nextRecommended`, las dependencias ni los motivos de bloqueo.

#### Scenario: Sin `pruebas.md` el digest no cambia
- **DADO** un cambio sin `pruebas.md`, con la función `acceptance_tests` activa o no
- **CUANDO** se calcula el digest de la compuerta `tasks`
- **ENTONCES** es idéntico al calculado antes de este cambio

#### Scenario: Editar `pruebas.md` cambia el digest
- **DADO** un cambio con `tasks.md` y `pruebas.md`
- **CUANDO** se edita solo `pruebas.md`
- **ENTONCES** el digest de la compuerta `tasks` es distinto del anterior

#### Scenario: El digest coincide en `gate record` y en el status
- **DADO** un cambio con `pruebas.md`
- **CUANDO** se calcula el digest con `axiom sdd gate record` y con `axiom sdd status`
- **ENTONCES** ambos valores son iguales

#### Scenario: Una compuerta `tasks` rechazada se reabre al editar solo `pruebas.md`
- **DADO** una compuerta `tasks` en `rejected` sobre un cambio con `pruebas.md`
- **CUANDO** se edita solo `pruebas.md` y se consulta el status
- **ENTONCES** la compuerta pasa a `pending`, reabierta

#### Scenario: Una compuerta `tasks` aprobada sigue aprobada
- **DADO** una compuerta `tasks` en `approved`
- **CUANDO** se edita `pruebas.md` después de la aprobación
- **ENTONCES** la compuerta sigue `approved`
- **Y** el comportamiento es el mismo que al editar `tasks.md`

### Requirement: La Parada `tasks` Presenta `pruebas.md` junto a los Ficheros de Tareas y su Resultado Informativo (RF-GAT-32)

En modalidad "con paradas", cuando el status proyecta `acceptanceTests` activo (RF-KOF-22) y existe `pruebas.md`, la parada de la compuerta `tasks` DEBE presentar `pruebas.md` junto a cada `tasks.<rol>.md`, con el resultado de `axiom sdd tests check` (casos, escenarios y requisitos cubiertos y hallazgos abiertos). La presentación DEBE declarar la ausencia de `pruebas.md` (con el aviso de RF-KOF-22) cuando la función está activa y el fichero falta, y NO DEBE presentarlo ni declararlo cuando la función no está activa. Un resultado con hallazgos se presenta como tal y nunca como limpio. El resultado NUNCA condiciona una aprobación, no se registra en `gates.yaml` y no cambia las tres salidas de RF-GAT-07 ni el contrato de RF-GAT-04.

#### Scenario: Parada `tasks` con `pruebas.md`
- **DADO** un cambio en `checkpointed` con `acceptanceTests` activo y `pruebas.md` presente
- **CUANDO** se presenta la compuerta de `tasks`
- **ENTONCES** se presentan los ficheros de tareas y `pruebas.md` con su cobertura y hallazgos
- **Y** se ofrecen las tres salidas de RF-GAT-07 sin cambios

#### Scenario: Función activa y `pruebas.md` ausente
- **DADO** un cambio con `acceptanceTests` activo sin `pruebas.md`
- **CUANDO** se presenta la compuerta de `tasks`
- **ENTONCES** la presentación declara que falta `pruebas.md`
- **Y** la aprobación no queda bloqueada

#### Scenario: Función no activa
- **DADO** un proyecto con `enabled: false` o sin el bloque
- **CUANDO** se presenta la compuerta de `tasks`
- **ENTONCES** no se menciona `pruebas.md`
