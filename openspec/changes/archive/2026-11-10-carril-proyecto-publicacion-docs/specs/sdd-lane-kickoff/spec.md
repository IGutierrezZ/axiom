# Delta para Selección Temprana y Bloqueante de Carril ODD/SDD en el Kickoff (carril-proyecto-publicacion-docs)

> **Cambio:** `carril-proyecto-publicacion-docs` · **Tipo:** delta sobre la capacidad viva `sdd-lane-kickoff`.
> **Idioma:** castellano de España; identificadores y claves en inglés.

Cambios de este delta:

- REQ-27.3 (nuevo): el carril declarado en el proyecto (`workflow.lane`, proyectado por `axiom sdd status`) determina si el orquestador pregunta el carril.
- REQ-27.4 (nuevo): el kickoff registra la autorización de publicación automática de documentos (`docs_publication: auto`) del estilo `continuous`.
- REQ-21.1: la pregunta bloqueante de carril solo se formula con carril `ask` o sin carril declarado utilizable (REQ-27.3).
- REQ-21.2: se retira la excepción de la parada única de ODD y sus dos escenarios (ODD ya no tiene paradas, REQ-27.17 de `organic-agent-trigger-rules`).
- REQ-21.3: entrada directa al pre-vuelo con carril `sdd` declarado.
- REQ-25.35: «una vez elegido el carril» pasa a «una vez determinado el carril».
- REQ-21.4, REQ-25.36 a REQ-25.38 no cambian.

## ADDED Requirements

### Requirement: El Carril Declarado en el Proyecto Gobierna la Pregunta de Carril (REQ-27.3)

Antes de decidir el carril de una solicitud de trabajo sustancial, el orquestador DEBE leer el campo `workflow` de `axiom sdd status` (REQ-27.2 de `axiom-sdd-cli-integration`) y DEBE actuar según el carril proyectado, sin formular la pregunta bloqueante de selección de carril (REQ-21.1) cuando el carril no es `ask`:

- `sdd`: el orquestador entra directamente al cuestionario de pre-vuelo de SDD (capacidad `sdd-preflight-configuration`) para el trabajo sustancial, sin preguntar el carril.
- `odd`: el orquestador trabaja en el carril ODD sin preguntar el carril, incluso ante un alcance inequívocamente arquitectónico. La persona puede pedir SDD explícitamente en una solicitud concreta, y esa petición prevalece para esa solicitud.
- `ask`: el comportamiento es el de REQ-21.1 sin cambios.

El carril declarado cuenta como petición explícita de ese carril. NO modifica el trabajo pequeño y entendido, que en cualquier carril no crea artefactos duraderos. Si `workflow.source` es `no_workspace` o `invalid` (carril proyectado `ask`), o si el orquestador no puede obtener la proyección, DEBE comportarse como `ask`, nunca como `sdd` automático. La lectura de la proyección NO es la creación de un artefacto de trabajo ni una pregunta de gobernanza. El orden de la regla de rama (REQ-25.35) no cambia: `axiom branch ensure` se ejecuta después de determinar el carril y antes del primer artefacto.

#### Scenario: Carril `sdd` entra al pre-vuelo sin preguntar el carril

- **DADO** un proyecto cuyo `axiom sdd status` proyecta `workflow.lane: sdd` y una solicitud sustancial de alcance acotado
- **CUANDO** el orquestador recibe la solicitud
- **ENTONCES** no formula la pregunta de selección de carril
- **Y** entra al cuestionario de pre-vuelo de SDD antes de crear `proposal.md`

#### Scenario: Carril `odd` trabaja en ODD sin preguntar

- **DADO** un proyecto que proyecta `workflow.lane: odd` y una solicitud sustancial de alcance acotado
- **CUANDO** el orquestador recibe la solicitud
- **ENTONCES** no formula la pregunta de selección de carril
- **Y** registra el trabajo exclusivamente en `odd/tasks/<feature>.md`

#### Scenario: Carril `odd` no pregunta ni ante un alcance arquitectónico

- **DADO** un proyecto que proyecta `workflow.lane: odd` y una solicitud inequívocamente arquitectónica
- **CUANDO** el orquestador evalúa el alcance
- **ENTONCES** trabaja en ODD sin entrar al pre-vuelo de SDD

#### Scenario: SDD explícito en una solicitud concreta prevalece sobre `odd`

- **DADO** un proyecto que proyecta `workflow.lane: odd` y una solicitud en la que la persona pide explícitamente SDD
- **CUANDO** el orquestador la recibe
- **ENTONCES** entra al cuestionario de pre-vuelo de SDD para esa solicitud

#### Scenario: Carril `ask` conserva la pregunta bloqueante

- **DADO** un proyecto que proyecta `workflow.lane: ask` y una solicitud sustancial de alcance acotado que admite ambos carriles
- **CUANDO** el orquestador recibe la solicitud
- **ENTONCES** formula la pregunta de selección de carril y se detiene a esperar respuesta
- **Y** no crea ningún artefacto de trabajo antes de la respuesta

#### Scenario: Sin `axiom.yaml` se comporta como `ask`

- **DADO** un repositorio sin `axiom.yaml` cuyo status proyecta `workflow.source: no_workspace`
- **CUANDO** el orquestador recibe una solicitud sustancial de alcance acotado
- **ENTONCES** formula la pregunta de selección de carril

#### Scenario: Un bloque inválido se comporta como `ask`

- **DADO** un `axiom.yaml` con `workflow.lane` inválido cuyo status proyecta `workflow.source: invalid`
- **CUANDO** el orquestador recibe una solicitud sustancial de alcance acotado
- **ENTONCES** formula la pregunta de selección de carril
- **Y** no entra al pre-vuelo de SDD sin preguntar

#### Scenario: El trabajo pequeño no crea artefactos con ningún carril

- **DADO** un proyecto con `workflow.lane: sdd` y una petición pequeña y entendida
- **CUANDO** el orquestador la ejecuta
- **ENTONCES** no crea ni `odd/tasks/<feature>.md` ni `proposal.md`

#### Scenario: El paso 0 de la doctrina instruye a leer la proyección

- **DADO** el texto de la doctrina ODD generada
- **CUANDO** se inspecciona (pin de texto) el paso 0
- **ENTONCES** instruye a leer `workflow` de `axiom sdd status` y define el comportamiento de `sdd`, `odd` y `ask`
- **Y** indica el comportamiento `ask` ante `no_workspace`, `invalid` o ausencia de proyección

---

### Requirement: `kickoff.yaml` Registra la Autorización de Publicación Automática en el Estilo `continuous` (REQ-27.4)

Cuando la persona elige el estilo `continuous` en el cuestionario de pre-vuelo, la pregunta DEBE nombrar la publicación de documentos en la propia opción: «sin paradas; sube los documentos a `origin/<rama>` al cerrar `tasks` y al archivar», con la rama real del cambio. Si la persona elige esa opción, `axiom sdd kickoff seal` DEBE registrar en `kickoff.yaml` el campo opcional `docs_publication: auto`, dentro de la misma única escritura de sellado y con la semántica de sellado único de REQ-25.36. El campo NO DEBE escribirse con el estilo `checkpointed` ni cuando la persona no autoriza la publicación, y la ausencia del campo significa «no se publica automáticamente». La serialización DEBE omitir el campo cuando está vacío (`omitempty`); la lectura de `kickoff.yaml` DEBE tolerar su ausencia y un binario anterior DEBE ignorarlo sin error. Sellar con `continuous` sin esa autorización DEBE seguir siendo posible. El campo NO modifica el estilo ni las compuertas y NO autoriza ninguna operación distinta de la publicación de documentos de REQ-27.13 de `sdd-docs-publication`.

#### Scenario: La opción `continuous` nombra remoto y rama

- **DADO** el cuestionario de pre-vuelo de un cambio cuya rama es `inc/x`
- **CUANDO** se presenta la opción `continuous`
- **ENTONCES** su texto indica que sube los documentos a `origin/inc/x` al cerrar `tasks` y al archivar

#### Scenario: Autorizar la publicación sella el campo

- **DADO** una persona que elige `continuous` con la publicación de documentos autorizada
- **CUANDO** se ejecuta `axiom sdd kickoff seal` con esa configuración
- **ENTONCES** `kickoff.yaml` contiene `docs_publication: auto`

#### Scenario: Sin autorización no se escribe el campo

- **DADO** una persona que elige `continuous` sin autorizar la publicación
- **CUANDO** se sella el cambio
- **ENTONCES** `kickoff.yaml` no contiene `docs_publication`

#### Scenario: El estilo `checkpointed` no escribe el campo

- **DADO** una persona que elige `checkpointed`
- **CUANDO** se sella el cambio
- **ENTONCES** `kickoff.yaml` no contiene `docs_publication`

#### Scenario: Un kickoff sin el campo se lee sin error

- **DADO** un `kickoff.yaml` sellado antes de este cambio, sin `docs_publication`
- **CUANDO** se lee y se ejecutan `status` y `continue`
- **ENTONCES** no hay error y el cambio se comporta como no autorizado para la publicación automática

#### Scenario: Un binario anterior ignora el campo

- **DADO** un `kickoff.yaml` con `docs_publication: auto`
- **CUANDO** lo decodifica el esquema previo a este cambio
- **ENTONCES** la decodificación termina sin error

#### Scenario: El sellado único no reescribe el campo

- **DADO** un cambio sellado con `docs_publication: auto`
- **CUANDO** se ejecuta un segundo `kickoff seal` desde la rama del cambio sin el campo
- **ENTONCES** termina con éxito indicando «Kickoff ya estaba sellado»
- **Y** `kickoff.yaml` conserva `docs_publication: auto`

## MODIFIED Requirements

### Requirement: Evaluación de Alcance y Pregunta Bloqueante de Selección de Carril (REQ-21.1)

El orquestador DEBE evaluar el alcance de toda solicitud de trabajo nueva antes de crear cualquier artefacto de trabajo (documento vivo ODD o propuesta SDD). Cuando el carril proyectado por el proyecto es `ask`, o cuando no hay carril declarado utilizable (REQ-27.3), y el alcance detectado es acotado y admite razonablemente ambos carriles, el orquestador DEBE formular explícitamente la pregunta bloqueante de selección de carril ("¿Deseas abordarla mediante vía ágil ODD o vía formal SDD?") y DEBE detenerse a esperar la respuesta del usuario antes de continuar. El orquestador NO DEBE crear `odd/tasks/<feature>.md` ni `openspec/changes/<cambio>/proposal.md` mientras la pregunta esté pendiente de respuesta. Con carril proyectado `sdd` u `odd` el orquestador NO DEBE formular esta pregunta (REQ-27.3).
(Previously: la pregunta se formulaba siempre que el alcance era acotado y admitía ambos carriles, sin tener en cuenta un carril declarado por el proyecto)

#### Scenario: Alcance acotado formula la pregunta y bloquea hasta obtener respuesta

- **DADO** un proyecto con carril proyectado `ask` y una solicitud de trabajo de alcance acotado que razonablemente admite tanto ODD como SDD
- **CUANDO** el orquestador recibe la solicitud
- **ENTONCES** formula la pregunta de selección de carril y se detiene a esperar respuesta
- **Y** no crea ningún artefacto de trabajo antes de recibir la respuesta

#### Scenario: Alcance inequívocamente arquitectónico evita la pregunta binaria

- **DADO** un proyecto con carril proyectado `ask` y una solicitud cuyo alcance es inequívocamente de arquitectura o de gran envergadura
- **CUANDO** el orquestador evalúa el alcance
- **ENTONCES** entra directamente al cuestionario de pre-vuelo de SDD (capacidad `sdd-preflight-configuration`) sin necesidad de formular la pregunta binaria ODD/SDD

#### Scenario: Con carril declarado `sdd` u `odd` no se formula la pregunta

- **DADO** un proyecto con carril proyectado `sdd` o `odd` y una solicitud de alcance acotado que admite ambos carriles
- **CUANDO** el orquestador recibe la solicitud
- **ENTONCES** no formula la pregunta de selección de carril
- **Y** sigue el carril declarado según REQ-27.3

---

### Requirement: Carril ODD sin Fricción Adicional (REQ-21.2)

Si el carril es ODD, ya sea porque el usuario lo elige en respuesta a la pregunta de REQ-21.1 o porque el proyecto lo declara (REQ-27.3), el orquestador NO DEBE formular ninguna pregunta adicional de gobernanza (modalidad de avance, roles, política de relevos), DEBE ejecutar el trabajo "del tirón" sin dividirlo en roles ni generar ningún `handoff.md`, y DEBE mantener el trabajo exclusivamente en `odd/tasks/<feature>.md`. El carril ODD NO tiene ninguna parada: el orquestador NO DEBE detenerse tras crear el documento de feature para ofrecer salidas ni para pedir aprobación del plan (REQ-27.17 de `organic-agent-trigger-rules`); solo se interrumpe por preguntas de producto o de entorno.
(Previously: la única interrupción prevista por defecto era la parada única de ODD tras crear el documento de feature y antes de la primera escritura de código, con las tres salidas de REQ-26.1 (REQ-26.10 de `organic-agent-trigger-rules`))

#### Scenario: Elección de ODD sin preguntas adicionales

- **DADO** que el usuario elige el carril ODD en respuesta a la pregunta de REQ-21.1
- **CUANDO** el orquestador inicia el trabajo
- **ENTONCES** no formula ninguna pregunta adicional de configuración de gobernanza
- **Y** el trabajo se registra exclusivamente en `odd/tasks/<feature>.md`

#### Scenario: Trabajo ODD multi-área sigue sin roles ni handoff

- **DADO** un trabajo bajo el carril ODD que toca varias áreas o módulos del repositorio
- **CUANDO** se ejecuta el trabajo hasta su cierre
- **ENTONCES** el orquestador no genera ningún artefacto `handoff.md`
- **Y** no introduce una división de roles aunque el trabajo abarque varias áreas

#### Scenario: ODD no se detiene tras crear el documento de feature

- **DADO** el carril ODD y el orquestador acaba de crear `odd/tasks/<feature>.md`
- **CUANDO** continúa el trabajo
- **ENTONCES** pasa a la primera tarea sin presentar ninguna parada ni salidas
- **Y** sigue sin haber formulado ninguna pregunta de modalidad, roles ni política de relevos

#### Scenario: Carril ODD declarado en el proyecto sin fricción adicional

- **DADO** un proyecto que proyecta `workflow.lane: odd`
- **CUANDO** el orquestador inicia un trabajo sustancial
- **ENTONCES** no formula la pregunta de carril ni ninguna pregunta de gobernanza
- **Y** el trabajo se registra exclusivamente en `odd/tasks/<feature>.md`

---

### Requirement: Entrada al Cuestionario de Pre-Vuelo de SDD (REQ-21.3)

Si el usuario elige el carril SDD en respuesta a la pregunta de REQ-21.1, si el proyecto declara el carril `sdd` (REQ-27.3), si el usuario pide SDD explícitamente en la solicitud, o si el alcance detectado ya es inequívocamente de arquitectura o gran alcance con el carril proyectado `ask` (REQ-21.1, segundo escenario), el orquestador DEBE entrar al cuestionario de pre-vuelo de SDD descrito en la capacidad `sdd-preflight-configuration` antes de crear `proposal.md`.
(Previously: la entrada al pre-vuelo solo contemplaba la elección explícita del usuario o el alcance ya determinado como arquitectónico)

#### Scenario: Entrada al pre-vuelo tras elección explícita o alcance ya determinado

- **DADO** que el usuario elige el carril SDD, o que el alcance ya se determinó inequívocamente como de arquitectura con el carril proyectado `ask`
- **CUANDO** el orquestador continúa el flujo
- **ENTONCES** entra al cuestionario de pre-vuelo de SDD antes de crear `proposal.md`

#### Scenario: Entrada directa al pre-vuelo con el carril `sdd` declarado

- **DADO** un proyecto que proyecta `workflow.lane: sdd` y una solicitud sustancial
- **CUANDO** el orquestador continúa el flujo
- **ENTONCES** entra al cuestionario de pre-vuelo de SDD antes de crear `proposal.md`
- **Y** no formula antes la pregunta de selección de carril

---

### Requirement: `ensure` al Nombrar el Cambio, antes de Escribir en su Carpeta (REQ-25.35)

Una vez determinado el carril (elegido por el usuario en REQ-21.1 o declarado por el proyecto en REQ-27.3) y nombrado el cambio (SDD) o la feature (ODD), el orquestador DEBE ejecutar `axiom branch ensure` para ese nombre (con `--lane odd` en ODD) **antes de crear cualquier artefacto de trabajo** (`proposal.md`, `odd/tasks/<feature>.md`) y, en SDD, antes del cuestionario de pre-vuelo y de `axiom sdd kickoff seal`. A partir de ese momento DEBE usar como `--cwd` de las órdenes posteriores las rutas absolutas que devuelve el JSON de `ensure` (REQ-25.2 de `change-branch-lifecycle`) y NO el directorio de trabajo heredado. La ejecución de `ensure` NO es la creación de un artefacto de trabajo, por lo que no contraviene REQ-21.1. Si `ensure` devuelve `decision_required` (modo `ask`), el orquestador DEBE presentar la elección al usuario y repetir la orden con la respuesta; esa pregunta es una preferencia de entorno personal y NO una pregunta de gobernanza, por lo que NO contraviene REQ-21.2 (supuesto S-10). Al reanudar un cambio ya sellado, `ensure` DEBE poder repetirse de forma idempotente sin repetir la selección de carril ni el pre-vuelo (REQ-21.4) y sin volver a preguntar el modo ya respondido.
(Previously: la regla arrancaba «una vez elegido el carril» y no contemplaba un carril declarado por el proyecto)

#### Scenario: SDD ejecuta `ensure` antes del sellado y de `proposal.md`
- **DADO** que el carril es SDD (elegido o declarado) y se nombra el cambio `x`
- **CUANDO** el orquestador inicia el flujo
- **ENTONCES** ejecuta `axiom branch ensure --change x` antes del cuestionario de pre-vuelo, de `kickoff seal` y de crear `proposal.md`
- **Y** las órdenes posteriores usan el `--cwd` absoluto del JSON de `ensure`

#### Scenario: ODD ejecuta `ensure --lane odd` antes de escribir `odd/tasks/`
- **DADO** que el carril es ODD (elegido o declarado) para la feature `mi-feature`
- **CUANDO** el orquestador va a crear `odd/tasks/mi-feature.md`
- **ENTONCES** ejecuta antes `axiom branch ensure --lane odd --change mi-feature`
- **Y** no formula ninguna pregunta adicional de gobernanza

#### Scenario: `ensure` no se ejecuta mientras la pregunta de carril está sin responder
- **DADO** una solicitud de alcance acotado, un carril proyectado `ask` y la pregunta de selección de carril sin respuesta
- **CUANDO** el orquestador espera la respuesta
- **ENTONCES** no ha ejecutado `ensure` ni creado ninguna rama, worktree o artefacto

#### Scenario: Modo `ask` presenta la elección sin considerarla gobernanza
- **DADO** un usuario en modo `ask` que eligió ODD
- **CUANDO** `ensure --lane odd --change mi-feature` devuelve `decision_required`
- **ENTONCES** el orquestador presenta la elección entre `worktree` e `in_place` como preferencia de entorno
- **Y** repite `ensure` con la respuesta y continúa sin otra pregunta

#### Scenario: Reanudar un cambio sellado no repite preguntas
- **DADO** un cambio SDD con `kickoff.yaml` sellado y `ensure` ya ejecutado
- **CUANDO** se reanuda en una sesión posterior
- **ENTONCES** `ensure` puede repetirse sin efectos
- **Y** no se repite la pregunta de carril, el pre-vuelo ni la pregunta de modo

#### Scenario: Doctrina presente en los orquestadores
- **DADO** el texto de la doctrina ODD generada (`routing.go`) y de los orquestadores SDD por runtime
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica ejecutar `axiom branch ensure` antes de escribir en la carpeta del cambio y usar las rutas absolutas del resultado con `--cwd`
