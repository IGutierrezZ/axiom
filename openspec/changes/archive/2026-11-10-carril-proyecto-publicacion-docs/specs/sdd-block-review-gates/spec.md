# Delta para Compuertas de Revisión por Bloque de SDD (carril-proyecto-publicacion-docs)

> **Cambio:** `carril-proyecto-publicacion-docs` · **Tipo:** delta sobre la capacidad viva `sdd-block-review-gates`.
> **Idioma:** castellano de España; identificadores, tokens y códigos en inglés.

Cambios de este delta:

- REQ-27.12 (nuevo): aprobar una parada (salidas 2 y 3) publica los documentos con `axiom sdd publish`; la pregunta nombra operación, remoto y rama; una negativa de `publish` es informativa y no anula la aprobación.
- REQ-26.1: aprobar publica; se retira la referencia a la parada de ODD (REQ-26.10, retirado).
- REQ-26.3: la cesión ya no implica una PR fusionada; el escenario de registro no exige artefactos confirmados.
- REQ-26.4 (renombrado y relajado): `gate record` ya no exige artefactos confirmados porque `publish` los confirma; conserva la guarda `wrong_branch`.
- REQ-26.5 (renombrado y reescrito): `gate record` no ejecuta git; `publish` sí. «Subir» deja de significar PR fusionada.
- REQ-26.6: disponibilidad de la salida 2 y de la publicación según el almacén.
- REQ-26.8: el texto único es exclusivo de SDD e incorpora la publicación en lugar de la PR fusionada.
- REQ-26.2 y REQ-26.7 no cambian. La puerta `integration` no cambia (REQ-26.1).

## ADDED Requirements

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

## RENAMED Requirements

### Requirement: La Salida 2 Comprueba que los Artefactos Están Confirmados antes de Registrar (REQ-26.4) → La Salida 2 No Exige Artefactos Confirmados y Conserva la Guarda de Rama (REQ-26.4)

(Reason: `axiom sdd publish` confirma los documentos del cambio, incluido `gates.yaml`, después de registrar la decisión (REQ-27.6 de `sdd-docs-publication`); exigir los artefactos confirmados antes de registrar ya no es necesario ni coherente con aprobar y publicar.)
(Migration: las pruebas y textos que exigían `uncommitted_artifacts` antes de registrar la salida 2 pasan a verificar que `gate record` registra sin esa exigencia; el trabajo de implementación de un rol sin subir lo detecta `publish` con `code_commits_unpublished` (REQ-27.8).)

### Requirement: La Salida 2 Devuelve el Destino y la Orden de PR sin Ejecutar Git Remoto (REQ-26.5) → La Salida 2 No Ejecuta Git en `gate record` y Entrega el Destino de la Publicación (REQ-26.5)

(Reason: ahora `axiom sdd publish` sí confirma y sube los documentos, pero `gate record` sigue sin ejecutar git; «subir» ya no significa una PR fusionada.)
(Migration: los textos y pruebas que describían la PR fusionada como condición para que otra persona empiece la fase siguiente pasan a describir la publicación de la rama y la continuación con `git fetch` y `axiom branch ensure` (REQ-27.15 de `change-branch-lifecycle`).)

## MODIFIED Requirements

### Requirement: Tres Salidas en Cada Parada de Planificación y de `role-apply` (REQ-26.1)

En modalidad "con paradas" (`execution_style: checkpointed`), cada parada de las compuertas `spec`, `design`, `tasks` y `role-apply:<rol>` DEBE ofrecer exactamente tres salidas, siempre en este orden y con la misma semántica:

1. **Iterar:** seguir trabajando sobre el artefacto actual. Se registra como decisión `rejected` con motivo (REQ-21.12).
2. **Aprobar, subir y parar:** se registra `approved` con la marca de cesión de la fase siguiente a otra persona, se suben los documentos del cambio con `axiom sdd publish` (REQ-27.12) y la sesión se detiene (REQ-26.3 a REQ-26.5).
3. **Aprobar y seguir:** se registra `approved` sin marca, se suben los documentos del cambio con `axiom sdd publish` (REQ-27.12) y el orquestador continúa él mismo con la fase siguiente.

La puerta `integration` NO es una parada de planificación: DEBE conservar sus dos salidas (aprobar con la evidencia de fusión exigida, o rechazar). En modalidad "continua" NO DEBE existir ninguna parada ni, por tanto, ninguna de las tres salidas (REQ-21.7). Las tres salidas son exclusivas de las paradas de SDD: el carril ODD no tiene paradas (REQ-27.17 de `organic-agent-trigger-rules`).
(Previously: la salida 2 solo dejaba constancia de lo que debía subirse, la salida 3 no subía nada, y el carril ODD tenía su propia parada con las mismas tres salidas (REQ-26.10))

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

### Requirement: La Salida 2 Registra `approved` con la Marca de Cesión, Compatible con Binarios Anteriores (REQ-26.3)

La salida 2 DEBE registrarse con `axiom sdd gate record --decision approved` y el indicador de la salida 2, y DEBE añadir a `gates.yaml` un registro `approved` con un campo opcional que marca la fase siguiente como cedida a otra persona (nombre del campo fijado en design). El campo DEBE escribirse únicamente con la salida 2: los registros de las salidas 1 y 3 NO DEBEN contenerlo. El vocabulario de decisiones del registro sigue siendo `approved|rejected`: NO se añade ninguna decisión nueva. Un binario anterior a este cambio DEBE poder leer un `gates.yaml` con el campo y tratar la puerta como `approved` normal, sin error. La aprobación sigue siendo terminal: un registro posterior no reabre la puerta. El primer registro `approved` es el que cierra la puerta; un registro de salida 2 sobre una puerta que ya está `approved` DEBE rechazarse sin escribir, porque la cesión no sería visible. El motivo (`--reason`) es opcional en la salida 2. La marca de cesión indica que la fase siguiente la hace otra persona; NO implica que exista una PR fusionada ni exige ninguna operación de git previa.
(Previously: sin la aclaración final de que la cesión no implica una PR fusionada; el escenario de registro exigía los artefactos confirmados)

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
(Previously: antes de registrar la salida 2 la orden comprobaba que los artefactos de la puerta y, en `role-apply:<rol>`, el trabajo de implementación estaban confirmados, y se rechazaba con los pendientes nombrados)

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
(Previously: la orden devolvía el destino de la PR, el paso de confirmar `gates.yaml`, la orden de push exacta y los datos de la PR, exigía una PR fusionada antes de la fase siguiente y no ejecutaba ninguna operación de git, y en el repositorio de specs de un multirepo indicaba commit y push a cargo del humano)

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
(Previously: sin la precisión de que la salida 3 aprueba sin publicar con los almacenes `engram` y `none`)

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

### Requirement: Texto Único de las Tres Salidas Proyectado a Todos los Runtimes (REQ-26.8)

El texto que describe las tres salidas (iterar, aprobar-subir-y-parar, aprobar-y-seguir) y la regla de parar DEBE definirse en una sola fuente (`stop-exits.md`) y proyectarse sin diferencias a los orquestadores SDD de todos los runtimes, de modo que un pin de texto lo encuentre idéntico en cada uno. El texto es exclusivo de SDD: NO DEBE mencionar ODD ni proyectarse a la doctrina ODD generada (REQ-27.17 de `organic-agent-trigger-rules`). Ningún asset del producto (skills, orquestadores por runtime, comandos, guías) DEBE ofrecer «generar un relevo formal `handoff.md` o continuar» como pregunta de transición tras `spec` o `design`, ni nombrar `--handoff-policy` como argumento obligatorio de `kickoff seal` (REQ-26.9 de `sdd-preflight-configuration`). El texto DEBE indicar que `axiom sdd gate record` es el único modo de registrar la decisión, que toda aprobación (salidas 2 y 3) se completa con `axiom sdd publish`, que esa operación sube solo los documentos del cambio a la rama del cambio en `origin` (en el repositorio de specs de un multirepo, a su rama de integración), que el push de código, la PR y el merge los hace la persona, y que con almacén `engram` o `none` la salida 2 no está disponible.
(Previously: el texto se proyectaba también a la doctrina ODD, indicaba que la salida 2 pedía una PR fusionada y, en el repositorio de specs de un multirepo, commit en la integración y push humano)

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
