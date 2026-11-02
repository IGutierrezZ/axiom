# Delta para Compuertas de Revisión por Bloque (spec, design, tasks, apply) (paradas-tres-salidas)

> **Cambio:** `paradas-tres-salidas` · **Tipo:** delta sobre la capacidad viva `sdd-block-review-gates`
> **Idioma:** Español (castellano peninsular); identificadores en inglés.

Cambios de este delta:

- Cada parada de planificación (`spec`, `design`, `tasks`) y la de `role-apply:<rol>` ofrece **tres salidas** en lugar de «aprobar o rechazar»: (1) iterar, (2) aprobar, subir y parar, (3) aprobar y seguir. REQ-21.8 a REQ-21.12 se reescriben sobre ese modelo; REQ-21.7 (modalidad continua sin paradas) no cambia.
- Requisitos nuevos REQ-26.1 a REQ-26.8: modelo de salidas, sobre tipado de tres opciones, semántica de la salida 2 (registro `approved` con marca de cesión, comprobación de artefactos confirmados, orden de PR sin git remoto, disponibilidad), visibilidad de la cesión y texto único proyectado.
- El vocabulario del registro (`approved|rejected`) NO cambia. La aprobación sigue siendo terminal (D-08): este cambio NO añade reapertura de puertas aprobadas.
- La puerta `integration` conserva sus dos salidas y el sobre de autoridad de edición sigue exigiendo exactamente dos opciones.

Supuestos de la fase de spec que `sdd-design` debe confirmar o corregir (no son requisitos):

- **A-1.** Un registro con la salida 2 sobre una puerta que ya está `approved` se rechaza sin escribir, porque el primer `approved` es el que cierra la puerta y la marca quedaría invisible.
- **A-2.** En `role-apply:<rol>`, «artefactos confirmados» incluye el trabajo de implementación del rol en el repositorio del cambio (árbol de trabajo sin cambios sin confirmar), además de `tasks.<rol>.md`; el criterio exacto lo fija design.
- **A-3.** Con almacén `engram` o `none` la pregunta tipada de la puerta presenta solo dos opciones (salidas 1 y 3) y declara que la salida 2 no está disponible; por tanto la validación del sobre admite dos o tres opciones según el caso, y el de autoridad de edición sigue en dos.
- **A-4.** En un repositorio con rama de seguimiento (dos o más roles), la orden de PR de la salida 2 de un `role-apply` usa el `prTarget` que reporta `axiom branch status` (la rama de seguimiento), sin afirmar que ya se integra en la rama de integración.
- **A-5.** El comando concreto de creación de la PR depende de la plataforma del remoto; la salida entrega siempre la orden de push exacta y los datos de la PR (rama de origen y `prTarget`), y design decide si añade además la orden de una plataforma reconocible.

## MODIFIED Requirements

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

## ADDED Requirements

### Requirement: Tres Salidas en Cada Parada de Planificación y de `role-apply` (REQ-26.1)

En modalidad "con paradas" (`execution_style: checkpointed`), cada parada de las compuertas `spec`, `design`, `tasks` y `role-apply:<rol>` DEBE ofrecer exactamente tres salidas, siempre en este orden y con la misma semántica:

1. **Iterar:** seguir trabajando sobre el artefacto actual. Se registra como decisión `rejected` con motivo (REQ-21.12).
2. **Aprobar, subir y parar:** se registra `approved` con la marca de cesión de la fase siguiente a otra persona, se deja constancia de lo que debe subirse y la sesión se detiene (REQ-26.3 a REQ-26.5).
3. **Aprobar y seguir:** se registra `approved` sin marca y el orquestador continúa él mismo con la fase siguiente.

La puerta `integration` NO es una parada de planificación: DEBE conservar sus dos salidas (aprobar con la evidencia de fusión exigida, o rechazar). En modalidad "continua" NO DEBE existir ninguna parada ni, por tanto, ninguna de las tres salidas (REQ-21.7). El carril ODD tiene su propia parada, con las mismas tres salidas, definida en `organic-agent-trigger-rules` (REQ-26.10).

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

La salida 2 DEBE registrarse con `axiom sdd gate record --decision approved` y el indicador de la salida 2, y DEBE añadir a `gates.yaml` un registro `approved` con un campo opcional que marca la fase siguiente como cedida a otra persona (nombre del campo fijado en design). El campo DEBE escribirse únicamente con la salida 2: los registros de las salidas 1 y 3 NO DEBEN contenerlo. El vocabulario de decisiones del registro sigue siendo `approved|rejected`: NO se añade ninguna decisión nueva. Un binario anterior a este cambio DEBE poder leer un `gates.yaml` con el campo y tratar la puerta como `approved` normal, sin error. La aprobación sigue siendo terminal: un registro posterior no reabre la puerta. El primer registro `approved` es el que cierra la puerta; un registro de salida 2 sobre una puerta que ya está `approved` DEBE rechazarse sin escribir, porque la cesión no sería visible. El motivo (`--reason`) es opcional en la salida 2.

#### Scenario: La salida 2 escribe un registro approved con la marca

- **DADO** una compuerta de `spec` pendiente y sus artefactos confirmados
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

---

### Requirement: La Salida 2 Comprueba que los Artefactos Están Confirmados antes de Registrar (REQ-26.4)

Antes de registrar la salida 2, `axiom sdd gate record` DEBE comprobar que los artefactos que juzga la puerta (los mismos cuyo digest se registra) están confirmados (commit) en la rama del cambio: seguidos por git y sin modificaciones sin confirmar. En `role-apply:<rol>` la comprobación incluye además el trabajo de implementación del rol en el repositorio del cambio, sin cambios sin confirmar (supuesto A-2). El propio `gates.yaml` queda fuera de la comprobación: lo modifica la orden y se confirma después (REQ-26.5). Si algún artefacto no está confirmado, la orden DEBE rechazarse con el código de salida distinto de cero, nombrar los artefactos pendientes, dar como continuación la confirmación (commit) y no escribir nada en `gates.yaml`. La comprobación solo usa lecturas locales de git. La guarda de rama vigente (`wrong_branch`) se aplica antes de esta comprobación y no se debilita.

#### Scenario: Artefactos confirmados permiten registrar

- **DADO** una compuerta de `design` pendiente y `design.md` confirmado en la rama del cambio
- **CUANDO** se ejecuta la salida 2
- **ENTONCES** el registro se escribe

#### Scenario: Un artefacto modificado sin confirmar rechaza la salida 2

- **DADO** una compuerta de `spec` pendiente cuyo `spec.md` tiene modificaciones sin confirmar
- **CUANDO** se ejecuta la salida 2
- **ENTONCES** la orden se rechaza nombrando ese artefacto y la continuación de confirmarlo
- **Y** `gates.yaml` no se modifica

#### Scenario: Un artefacto nuevo sin añadir a git rechaza la salida 2

- **DADO** una compuerta de `spec` pendiente cuya delta spec nueva no está seguida por git
- **CUANDO** se ejecuta la salida 2
- **ENTONCES** la orden se rechaza nombrando ese fichero
- **Y** `gates.yaml` no se modifica

#### Scenario: El propio `gates.yaml` no cuenta como pendiente

- **DADO** una compuerta de `tasks` pendiente, sus artefactos confirmados y un `gates.yaml` con registros anteriores sin confirmar
- **CUANDO** se ejecuta la salida 2
- **ENTONCES** la orden no se rechaza por el estado de `gates.yaml`

#### Scenario: La salida 2 de un rol exige el trabajo del rol confirmado

- **DADO** una compuerta `role-apply:core` pendiente con cambios de implementación del rol sin confirmar
- **CUANDO** se ejecuta la salida 2
- **ENTONCES** la orden se rechaza nombrando esos cambios y no escribe en `gates.yaml`

#### Scenario: Fuera de la rama que corresponde se rechaza con la guarda de rama

- **DADO** un monorepo con el repositorio desplegado en la rama de integración en lugar de la del cambio
- **CUANDO** se ejecuta la salida 2
- **ENTONCES** la orden se rechaza con `wrong_branch` antes de comprobar los artefactos
- **Y** no se escribe nada

---

### Requirement: La Salida 2 Devuelve el Destino y la Orden de PR sin Ejecutar Git Remoto (REQ-26.5)

Tras registrar la salida 2, la orden DEBE devolver al humano, en su salida de texto y en `--json`: la rama del cambio, el destino de la PR (`prTarget` que reporta `axiom branch status --json` para esa rama), el paso previo de confirmar el `gates.yaml` actualizado en esa misma rama, la orden de push exacta y los datos exactos de la PR (rama de origen y `prTarget`). Hay que dejar claro que «subir» significa que la PR esté **fusionada** en el destino antes de que otra persona empiece la fase siguiente, porque `axiom branch ensure` en otra máquina crea la rama desde la base y no recupera artefactos sin fusionar. Axiom NO DEBE ejecutar `push`, `fetch`, `pull`, `ls-remote` ni ninguna otra operación remota, ni abrir la PR, ni hacer commit: push, PR y merge son del humano. El orquestador DEBE detenerse tras presentar esa información y NO DEBE iniciar la fase siguiente. Según la topología (`workspace-topology`):

- **`monorepo-embedded`:** PR desde la rama del cambio hacia `prTarget` (la rama de integración o, en un repositorio con rama de seguimiento, el destino que reporte `status`; supuesto A-4).
- **`multirepo`, repositorio de specs:** se trabaja in situ en la rama de integración, sin rama de cambio ni PR. La salida devuelve la indicación de confirmar `gates.yaml` y los artefactos en esa rama y el `git push` de la rama de integración, a cargo del humano.

#### Scenario: Monorepo devuelve rama, destino y órdenes

- **DADO** un monorepo con la rama `inc/x` y `prTarget` `main`
- **CUANDO** se registra la salida 2 de la compuerta `spec`
- **ENTONCES** la salida nombra la rama `inc/x`, el destino `main`, el paso de confirmar `gates.yaml`, la orden `git push` de `inc/x` y los datos de la PR de `inc/x` hacia `main`
- **Y** advierte que la fase siguiente solo debe empezar cuando la PR esté fusionada

#### Scenario: Multirepo en el repositorio de specs indica commit y push sobre la integración

- **DADO** un multirepo con el repositorio de specs en su rama de integración `main`
- **CUANDO** se registra la salida 2
- **ENTONCES** la salida indica confirmar `gates.yaml` y los artefactos en `main` y el push de `main` por el humano
- **Y** no menciona ninguna rama de cambio ni PR

#### Scenario: No se ejecuta ninguna operación remota ni commit

- **DADO** un observador instalado en el único punto del motor que ejecuta git
- **CUANDO** se registra la salida 2
- **ENTONCES** el observador no registra ningún `push`, `fetch`, `pull`, `ls-remote` ni `commit`

#### Scenario: El orquestador para y no inicia la fase siguiente

- **DADO** una salida 2 recién registrada en la compuerta de `spec`
- **CUANDO** el orquestador presenta el destino y las órdenes
- **ENTONCES** se detiene sin iniciar `design`
- **Y** devuelve el control al humano

#### Scenario: La salida `--json` contiene los datos de la entrega

- **DADO** la salida 2 ejecutada con `--json`
- **CUANDO** se lee su salida
- **ENTONCES** contiene la rama, el `prTarget` y las órdenes en campos separados, sin texto libre como único portador

---

### Requirement: Disponibilidad de la Salida 2 según el Almacén, el Estilo y la Puerta (REQ-26.6)

La salida 2 solo es real cuando hay ficheros que subir: DEBE estar disponible únicamente con el almacén de artefactos `openspec` o `hybrid`, en un cambio `checkpointed` y en las compuertas `spec`, `design`, `tasks` y `role-apply:<rol>`. Con el almacén `engram` o `none`, `axiom sdd gate record` con el indicador de la salida 2 DEBE rechazarse con un error que nombre el almacén y ofrezca como continuación las salidas 1 o 3, sin escribir en `gates.yaml`; y la pregunta tipada de la compuerta NO DEBE ofrecer la salida 2: presenta solo las salidas 1 y 3 y declara en su texto que la salida 2 no está disponible por el almacén declarado (supuesto A-3). El estilo continuo y la puerta `integration` rechazan la salida 2 según REQ-26.1.

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

El texto que describe las tres salidas (iterar, aprobar-subir-y-parar, aprobar-y-seguir) y la regla de parar DEBE definirse en una sola fuente y proyectarse sin diferencias a los orquestadores SDD de todos los runtimes y a la doctrina ODD generada, de modo que un pin de texto lo encuentre idéntico en cada uno. Ningún asset del producto (skills, orquestadores por runtime, comandos, guías) DEBE ofrecer «generar un relevo formal `handoff.md` o continuar» como pregunta de transición tras `spec` o `design`, ni nombrar `--handoff-policy` como argumento obligatorio de `kickoff seal` (REQ-26.9 de `sdd-preflight-configuration`). El texto DEBE indicar que `axiom sdd gate record` es el único modo de registrar la decisión, que la salida 2 pide una PR fusionada y que, en el repositorio de specs de un multirepo, es commit en la integración y push humano.

#### Scenario: El texto de las tres salidas es idéntico en todos los runtimes

- **DADO** las plantillas de orquestador de todos los runtimes soportados y la doctrina ODD generada
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
- **ENTONCES** indica que la salida 2 requiere una PR fusionada en el destino y que push, PR y merge los hace el humano
- **Y** indica el caso de commit en la integración y push humano para el repositorio de specs de un multirepo
- **Y** indica que con almacén `engram` o `none` la salida 2 no está disponible
