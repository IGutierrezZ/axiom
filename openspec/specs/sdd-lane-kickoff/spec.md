<!-- Especificación Viva generada a partir de '2026-09-22-inc-21-upfront-flow-governance' -->

# Especificación de Requerimientos: Selección Temprana y Bloqueante de Carril ODD/SDD en el Kickoff (INC-21)

## Propósito

Definir de forma rigurosa, ejecutable y verificable los requerimientos funcionales y escenarios BDD para la compuerta temprana y bloqueante que determina si una solicitud de trabajo nueva se aborda mediante el carril ágil (ODD — Organic Driven Development) o el carril formal (SDD — Spec-Driven Development), sellando esa decisión para todo el ciclo de vida del cambio.

---

## 1. Capacidad: `sdd-lane-kickoff`

El orquestador determina el carril de ejecución (ODD o SDD) en el primer contacto con cualquier solicitud de trabajo, antes de crear ningún artefacto, y sella esa decisión para el resto del ciclo de vida del cambio.

### Requirement: Evaluación de Alcance y Pregunta Bloqueante de Selección de Carril (RF-KOF-01)

El orquestador DEBE evaluar el alcance de toda solicitud de trabajo nueva antes de crear cualquier artefacto de trabajo (documento vivo ODD o propuesta SDD). Cuando el carril proyectado por el proyecto es `ask`, o cuando no hay carril declarado utilizable (RF-KOF-09), y el alcance detectado es acotado y admite razonablemente ambos carriles, el orquestador DEBE formular explícitamente la pregunta bloqueante de selección de carril ("¿Deseas abordarla mediante vía ágil ODD o vía formal SDD?") y DEBE detenerse a esperar la respuesta del usuario antes de continuar. El orquestador NO DEBE crear `odd/tasks/<feature>.md` ni `openspec/changes/<cambio>/proposal.md` mientras la pregunta esté pendiente de respuesta. Con carril proyectado `sdd` u `odd` el orquestador NO DEBE formular esta pregunta (RF-KOF-09).

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
- **Y** sigue el carril declarado según RF-KOF-09

---

### Requirement: Carril ODD sin Fricción Adicional (RF-KOF-02)

Si el carril es ODD, ya sea porque el usuario lo elige en respuesta a la pregunta de RF-KOF-01 o porque el proyecto lo declara (RF-KOF-09), el orquestador NO DEBE formular ninguna pregunta adicional de gobernanza (modalidad de avance, roles, política de relevos), DEBE ejecutar el trabajo "del tirón" sin dividirlo en roles ni generar ningún `handoff.md`, y DEBE mantener el trabajo exclusivamente en `odd/tasks/<feature>.md`. El carril ODD NO tiene ninguna parada: el orquestador NO DEBE detenerse tras crear el documento de feature para ofrecer salidas ni para pedir aprobación del plan (RF-RUT-18 de `organic-agent-trigger-rules`); solo se interrumpe por preguntas de producto o de entorno.

#### Scenario: Elección de ODD sin preguntas adicionales

- **DADO** que el usuario elige el carril ODD en respuesta a la pregunta de RF-KOF-01
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

### Requirement: Entrada al Cuestionario de Pre-Vuelo de SDD (RF-KOF-03)

Si el usuario elige el carril SDD en respuesta a la pregunta de RF-KOF-01, si el proyecto declara el carril `sdd` (RF-KOF-09), si el usuario pide SDD explícitamente en la solicitud, o si el alcance detectado ya es inequívocamente de arquitectura o gran alcance con el carril proyectado `ask` (RF-KOF-01, segundo escenario), el orquestador DEBE entrar al cuestionario de pre-vuelo de SDD descrito en la capacidad `sdd-preflight-configuration` antes de crear `proposal.md`.

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

### Requirement: Idempotencia del Kickoff ante un Cambio con Artefactos o Configuración Ya Sellada (RF-KOF-04)

El orquestador NO DEBE volver a formular la pregunta de selección de carril (RF-KOF-01) ni el cuestionario de pre-vuelo de SDD (capacidad `sdd-preflight-configuration`) para un cambio que ya tiene al menos un artefacto persistido (`proposal.md`, un `spec.md` bajo `specs/`, `design.md` o `tasks.md`) o cuya configuración de kickoff ya está sellada. En ese caso, el orquestador DEBE continuar respetando el carril y la configuración ya selladas en el primer kickoff, sin interrogar de nuevo al usuario. El retro-sello (`axiom sdd kickoff seal --infer`, `sealed_by: inferred`) está **exento** de la regla «`continuous` solo con el rol `fullstack`» (RF-PRE-04 de `sdd-preflight-configuration`): conserva `continuous` con los roles que la detección de roles ya vigente infiere, aunque no sean `fullstack`, y esa regla solo se aplica a los sellados explícitos.

(Previamente: esta especificación indicaba que, al retro-sellar un cambio preexistente sin kickoff sellado, el rol por defecto congelado era el rol único `fullstack`. Se corrige tras la pregunta abierta O-1 planteada en `sdd-design` y resuelta explícitamente por el usuario con evidencia de código —`internal/multirole/detector.go:35-55` cae siempre a `core`, nunca a `fullstack`; `internal/multirole/detector.go:58-64` rechaza cualquier rol ausente de `axiom.yaml`; y `axiom.yaml:7-28` no declara `fullstack`—: el retro-sello DEBE congelar el rol que la detección de roles ya vigente infiere para ese cambio, nunca introducir la identidad `fullstack`.)
(Previously: la configuración sellada incluía la política de relevos y el retro-sello inferido nombraba «sin `handoff.md` intermedios»; la política de relevos ya no forma parte de la configuración sellada)
(Previously: no decía nada sobre la regla «`continuous` solo con `fullstack`»; ahora declara expresamente que el retro-sello queda exento de ella y conserva `continuous` con los roles detectados)

#### Scenario: Reanudación de un cambio con kickoff ya sellado

- **DADO** un cambio SDD activo con `proposal.md` existente y su configuración de kickoff ya sellada (carril `sdd` y modalidad definida)
- **CUANDO** se reanuda el trabajo sobre ese cambio en una sesión posterior
- **ENTONCES** el orquestador no repite la pregunta de selección de carril ni el cuestionario de pre-vuelo
- **Y** continúa el flujo respetando la modalidad ya sellada

#### Scenario: Reanudación de un cambio sellado con la política de relevos antigua

- **DADO** un cambio SDD activo cuyo `kickoff.yaml` conserva `handoff_policy: per_checkpoint` o `none` de antes de este cambio
- **CUANDO** se reanuda el trabajo sobre ese cambio
- **ENTONCES** el orquestador no pregunta nada sobre la política de relevos
- **Y** el valor antiguo no altera el flujo ni se vuelve a escribir

#### Scenario: Cambio preexistente con artefactos pero sin kickoff sellado infiere un valor por defecto sin bloquear

- **DADO** un cambio SDD activo con artefactos existentes (por ejemplo `proposal.md` y `design.md`) pero sin ninguna configuración de kickoff sellada previamente
- **CUANDO** el orquestador retoma ese cambio
- **ENTONCES** infiere y sella la configuración por defecto documentada (modalidad continua y el rol que la detección de roles ya vigente infiere para ese cambio —hoy, `core`, cuando existe un `design.md` sin roles explícitos—) sin bloquear el trabajo ni interrogar retroactivamente al usuario
- **Y** el retro-sello no introduce la identidad `fullstack` ni ningún rol nuevo: es conservador por definición y se limita a congelar el rol y el comportamiento que la detección de roles ya aplica hoy para ese cambio, sin redefinirlo
- **Y** el retro-sello no infiere ninguna política de relevos: escribe `handoff_policy: none` solo como valor de compatibilidad (RF-PRE-03 de `sdd-preflight-configuration`)
- **Y** dicha configuración inferida queda registrada para no volver a inferirse en la siguiente reanudación

#### Scenario: El retro-sello conserva continuous con roles distintos de fullstack

- **DADO** un cambio preexistente sin kickoff cuya detección de roles infiere `core`
- **CUANDO** se ejecuta el retro-sello
- **ENTONCES** el sellado se completa con `execution_style: continuous`, `sealed_by: inferred` y el rol `core`
- **Y** no se rechaza por la regla de RF-PRE-04 ni se infiere `checkpointed`
- **Y** no se abre ninguna puerta `pending` para artefactos ya escritos

#### Scenario: El retro-sello de varios roles detectados también conserva continuous

- **DADO** un cambio preexistente sin kickoff cuyo `design.md` declara los roles `core` y `web`
- **CUANDO** se ejecuta el retro-sello
- **ENTONCES** se sella `continuous` con ambos roles detectados
- **Y** `axiom sdd status` lo trata como un cambio sin paradas, igual que antes de este cambio

#### Scenario: La regla de sellado sigue aplicándose al sellado explícito del mismo cambio

- **DADO** un cambio con artefactos pero sin kickoff, que el usuario sella de forma explícita (sin `--infer`) en `continuous` con los roles `core` y `web`
- **CUANDO** se ejecuta el sellado
- **ENTONCES** se rechaza conforme a RF-PRE-04
- **Y** la exención del retro-sello no se extiende a esa ruta

#### Scenario: Un retro-sello sin ningún rol detectable conserva su comportamiento

- **DADO** un cambio preexistente cuya detección de roles no puede determinar ningún rol y devuelve un error
- **CUANDO** se ejecuta el retro-sello
- **ENTONCES** se comporta exactamente como antes de este cambio, sin tratarlo como un caso de la regla de RF-PRE-04

#### Scenario: El retro-sello no altera qué `tasks.<rol>.md` consulta la barrera

- **DADO** un cambio SDD preexistente sin kickoff sellado, cuya barrera de sincronización ya consulta `tasks.core.md` porque la detección de roles vigente infiere el rol `core` para su `design.md`
- **CUANDO** el orquestador retro-sella la configuración de kickoff de ese cambio conforme al escenario anterior
- **ENTONCES** la barrera de sincronización sigue consultando exactamente `tasks.core.md`, el mismo fichero que consultaba antes del sellado
- **Y** el retro-sello no crea, renombra ni redirige ningún fichero `tasks.<rol>.md`, ni introduce `tasks.fullstack.md` como fuente de verdad nueva para ese cambio
### Requirement: `ensure` al Nombrar el Cambio, antes de Escribir en su Carpeta (RF-KOF-05)

Una vez determinado el carril (elegido por el usuario en RF-KOF-01 o declarado por el proyecto en RF-KOF-09) y nombrado el cambio (SDD) o la feature (ODD), el orquestador DEBE ejecutar `axiom branch ensure` para ese nombre (con `--lane odd` en ODD) **antes de crear cualquier artefacto de trabajo** (`proposal.md`, `odd/tasks/<feature>.md`) y, en SDD, antes del cuestionario de pre-vuelo y de `axiom sdd kickoff seal`. A partir de ese momento DEBE usar como `--cwd` de las órdenes posteriores las rutas absolutas que devuelve el JSON de `ensure` (RF-BRA-02 de `change-branch-lifecycle`) y NO el directorio de trabajo heredado. La ejecución de `ensure` NO es la creación de un artefacto de trabajo, por lo que no contraviene RF-KOF-01. Si `ensure` devuelve `decision_required` (modo `ask`), el orquestador DEBE presentar la elección al usuario y repetir la orden con la respuesta; esa pregunta es una preferencia de entorno personal y NO una pregunta de gobernanza, por lo que NO contraviene RF-KOF-02 (supuesto S-10). Al reanudar un cambio ya sellado, `ensure` DEBE poder repetirse de forma idempotente sin repetir la selección de carril ni el pre-vuelo (RF-KOF-04) y sin volver a preguntar el modo ya respondido.

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
### Requirement: Instantánea de Política Git en `kickoff.yaml`, Escrita una Sola Vez por `seal` (RF-KOF-06)

`axiom sdd kickoff seal` DEBE escribir en `kickoff.yaml`, junto al resto de la configuración sellada y en la misma única escritura, la instantánea de política git del cambio: la rama de integración y el prefijo resueltos, la base (nombre de rama, no SHA) de cada repositorio y los nombres de rama derivados del cambio (rama del cambio, por repositorio con dos o más roles las ramas de rol y, en los repositorios con rama de cambio, la rama de archivado `<prefijo>/<cambio>-archive`), calculados con la política de `axiom.yaml`, las excepciones por cambio de `ensure` (`--prefix`, `--branch`, `--base`) y los roles que se sellan. Cuando la plantilla efectiva de una rama de código procede de `axiom.yaml` (RF-BRA-44 de `change-branch-lifecycle`), la instantánea DEBE guardar además, en un campo aditivo y opcional de esa rama, la plantilla (la política, no el resultado); si la plantilla usa `{task_ref}`, la instantánea NO DEBE escribir el nombre de esa rama, que queda pendiente hasta la entrada del rol, y la rama que lleva `openspec/` y las ramas cuya plantilla no usa `{task_ref}` siguen llevando su nombre resuelto. Sin plantilla, la instantánea es idéntica a la de antes. Los valores `kind` y `ref` de la plantilla se congelan según RF-KOF-26. Siempre que haya dos etapas al sellar (`git.main_branch` declarada y distinta de la rama de integración de algún repositorio del plan, RF-BRA-28 de `change-branch-lifecycle`), la instantánea DEBE guardar además el campo opcional `main_branch` (la rama principal declarada al sellar), que actúa de marcador: distingue un cambio sellado con dos etapas de uno sellado antes de existir esta capacidad. Además, la instantánea DEBE copiar el modo de promoción (`branch` o `release`) de un repositorio solo cuando ese modo ya estaba resuelto en el registro local al sellar (la rama del cambio se creó antes del sellado, como en el monorepo SDD), y la base de ese repositorio DEBE ser su base efectiva (la rama principal en `branch`, la de integración en `release`). El modo es un campo opcional de cada repositorio: NO DEBE escribirse en los repositorios de una sola etapa ni en los repositorios cuyo modo no estaba resuelto en el registro al sellar (como los repositorios de código de un `multirepo`, cuyas ramas y respuestas nacen en un `ensure` posterior al sellado); su ausencia, con `main_branch` presente en la instantánea, significa que el modo se resuelve al crear la rama, nunca que el repositorio sea de una sola etapa. Sin dos etapas al sellar, la instantánea no contiene `main_branch`. La instantánea contiene entradas de política, nunca resultados: NO DEBE incluir SHAs, rutas absolutas ni rutas de worktree, que son personales o cambian. La instantánea se escribe una sola vez: conserva la semántica de sellado único, en la que gana la primera publicación, y NINGUNA otra orden, incluida `ensure`, DEBE modificarla. Un segundo `seal` desde la rama del cambio NO DEBE reescribir `kickoff.yaml`: DEBE terminar con éxito, devolver el kickoff ya sellado e indicar que no escribió (texto «Kickoff ya estaba sellado»; `wrote: false` en `--json`). Un segundo `seal` desde otra rama DEBE rechazarse con `wrong_branch` (veredicto de RF-BRA-19 de `change-branch-lifecycle`) sin modificar nada. La lectura de `kickoff.yaml` DEBE tolerar su ausencia y un binario anterior DEBE ignorarla sin error, incluidos los campos opcionales `main_branch`, el modo de promoción, la plantilla de rama, `kind` y `ref`.
(Previously: la instantánea no incluía `main_branch` ni el modo de promoción y la base de cada repositorio era siempre la rama de integración)
(Previously: la instantánea guardaba solo nombres de rama, sin la plantilla de cada rama de código ni la omisión del nombre de las ramas con `{task_ref}`)

#### Scenario: El sellado escribe la instantánea
- **DADO** un monorepo con `git.integration_branch: main`, `git.branch_prefix: feat` y el rol único `fullstack`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...` desde la rama del cambio
- **ENTONCES** `kickoff.yaml` contiene la integración `main`, el prefijo `feat`, la base `main` del repositorio y los nombres `feat/x` y la rama de archivado `feat/x-archive`
- **Y** no contiene ninguna rama de rol
- **Y** no contiene SHAs ni rutas absolutas
- **Y** no contiene `main_branch` ni modo de promoción

#### Scenario: El sellado en multirepo escribe las ramas de rol y ninguna rama de archivado
- **DADO** un multirepo con `git.integration_branch: main`, `git.branch_prefix: feat` y los roles `core` y `qa` sobre el mismo repositorio de código `repo-backend`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...` desde la rama que corresponde (RF-KOF-08)
- **ENTONCES** `kickoff.yaml` contiene la integración `main`, el prefijo `feat`, la base `main` de `repo-backend` y, para `repo-backend`, los nombres `feat/x`, `feat/x-core` y `feat/x-qa`
- **Y** no contiene rama de archivado para `repo-backend`
- **Y** no contiene SHAs ni rutas absolutas

#### Scenario: La instantánea refleja las excepciones por cambio
- **DADO** un cambio materializado con `ensure --change x --branch inc/x`
- **CUANDO** se sella el cambio
- **ENTONCES** la instantánea nombra la rama del cambio `inc/x`

#### Scenario: La instantánea no se reescribe
- **DADO** un cambio sellado con instantánea
- **CUANDO** se ejecutan de nuevo, desde la rama del cambio, `kickoff seal` con otra configuración y después `axiom branch ensure --change x`
- **ENTONCES** el segundo `seal` termina con éxito, indica «Kickoff ya estaba sellado» y devuelve la configuración del primero
- **Y** `kickoff.yaml` no se modifica en ninguna de las dos órdenes

#### Scenario: Un segundo sellado desde otra rama se rechaza
- **DADO** un cambio sellado con instantánea cuya rama del cambio es `feat/x`
- **CUANDO** se ejecuta de nuevo `kickoff seal` desde `main`
- **ENTONCES** la orden se rechaza con `wrong_branch`, nombrando la rama actual `main` y la esperada `feat/x`
- **Y** `kickoff.yaml` no se modifica

#### Scenario: La lectura tolera la ausencia de instantánea
- **DADO** un `kickoff.yaml` sin instantánea
- **CUANDO** se carga con `ParseKickoff`
- **ENTONCES** la carga tiene éxito y la instantánea queda vacía

#### Scenario: Un binario anterior ignora la instantánea
- **DADO** un `kickoff.yaml` con instantánea
- **CUANDO** lo lee un binario anterior a esta capacidad
- **ENTONCES** carga el resto de la configuración sellada sin error

#### Scenario: El sellado copia el modo de promoción cuando ya estaba resuelto en el registro
- **DADO** un monorepo con `git.main_branch: master`, `git.integration_branch: develop`, el cambio `x` ya preparado por `ensure` con el modo `branch` registrado
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** `kickoff.yaml` contiene `main_branch: master`, el modo `branch` del repositorio y su base efectiva `master`
- **Y** la integración sigue siendo `develop`

#### Scenario: El sellado copia el modo `release` con la integración como base
- **DADO** el mismo monorepo con el modo `release` registrado
- **CUANDO** se sella el cambio
- **ENTONCES** la instantánea contiene el modo `release` y la base `develop`

#### Scenario: Sin modo resuelto en el registro al sellar la instantánea guarda solo `main_branch`
- **DADO** un multirepo con `git.main_branch: master` y `git.promotion: ask`, cuyos repositorios de código no tienen modo registrado al sellar
- **CUANDO** se sella el cambio
- **ENTONCES** la instantánea contiene `main_branch: master` y no contiene modo de promoción para esos repositorios
- **Y** `kickoff.yaml` no contiene ninguna pregunta ni respuesta pendiente

#### Scenario: Una sola etapa no escribe `main_branch` ni modo de promoción
- **DADO** un monorepo con `git.integration_branch: develop` y sin `git.main_branch`
- **CUANDO** se sella el cambio
- **ENTONCES** la instantánea no contiene `main_branch` ni modo de promoción

#### Scenario: Multirepo con el registro borrado tras responder después del sellado vuelve a preguntar
- **DADO** un multirepo con `git.main_branch: master` y `git.promotion: ask`, el cambio sellado con `main_branch: master` y sin modo en la instantánea, la respuesta `branch` dada y registrada con un `ensure` posterior al sellado, y el registro local borrado después
- **CUANDO** se ejecuta `axiom branch ensure --change x`
- **ENTONCES** `ensure` devuelve `decision_required` por el modo de promoción del repositorio, porque la instantánea tiene `main_branch`
- **Y** no trata el repositorio como de una sola etapa ni lo resuelve como `legacy`
- **Y** `kickoff.yaml` no se modifica

#### Scenario: `ensure` respeta el modo de la instantánea y no la reescribe
- **DADO** un cambio sellado con la instantánea con el modo `branch` y el registro local borrado
- **CUANDO** se ejecuta `axiom branch ensure --change x`
- **ENTONCES** el modo resuelto es `branch` y se registra con el origen de instantánea
- **Y** `kickoff.yaml` no se modifica

#### Scenario: La instantánea con modo de promoción sigue siendo inmutable
- **DADO** un cambio sellado con la instantánea con el modo `branch`
- **CUANDO** se cambia `git.promotion` a `release` en `axiom.yaml` y se ejecuta de nuevo `kickoff seal` y `ensure`
- **ENTONCES** `kickoff.yaml` conserva el modo `branch` sin modificarse


#### Scenario: La instantánea guarda la plantilla y omite el nombre de la rama con `{task_ref}`
- **DADO** un multirepo con `repositories[].branch_template: "feature/TASK-{task_ref}"` para `repo-frontend`, el rol `frontend` único sobre él y `repo-backend` sin plantilla
- **CUANDO** se sella el cambio `x`
- **ENTONCES** la instantánea guarda para `repo-frontend` la plantilla `feature/TASK-{task_ref}` y no guarda el nombre de su rama
- **Y** guarda para `repo-backend` el nombre `<prefijo>/x`, sin plantilla
- **Y** no contiene SHAs ni rutas absolutas

#### Scenario: La rama con `openspec/` lleva su nombre resuelto aunque tenga plantilla
- **DADO** un monorepo con `git.branch_template: "{prefix}/{ref}-{slug}"`, el prefijo `inc` y el staging con `id` `2525` para el cambio `us-2525-alta-de-usuarios`
- **CUANDO** se sella el cambio
- **ENTONCES** la instantánea contiene el nombre `inc/2525-alta-de-usuarios` y la plantilla de esa rama
- **Y** contiene la rama de archivado `inc/2525-alta-de-usuarios-archive`

#### Scenario: Sin plantilla la instantánea no cambia
- **DADO** un `axiom.yaml` sin `branch_template`
- **CUANDO** se sella el cambio
- **ENTONCES** `kickoff.yaml` es idéntico, byte a byte, al que se escribía antes de esta capacidad

#### Scenario: Un seal con una variable sin valor se rechaza sin sellar
- **DADO** `git.branch_template: "{prefix}/{ref}"` y un cambio sin work item ni staging
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** se rechaza con el código tipado de variable sin valor que nombra `ref`
- **Y** no se escribe `kickoff.yaml`

#### Scenario: Un seal con una rama compartida con `{role}` se rechaza sin sellar
- **DADO** un repositorio compartido por los roles `core` y `qa` con `git.branch_template: "{prefix}/{role}/{change}"` como plantilla de seguimiento
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** se rechaza con `branch_template_not_applicable` (motivo `shared_branch`) que nombra `repositories[].branch_template`
- **Y** no se escribe `kickoff.yaml`

#### Scenario: Cambiar la plantilla después del sellado no modifica la instantánea
- **DADO** un cambio sellado con la plantilla `feature/TASK-{task_ref}` en la instantánea
- **CUANDO** se cambia la plantilla en `axiom.yaml` y se ejecuta `ensure --change x --role frontend`
- **ENTONCES** la rama se nombra con la plantilla de la instantánea
- **Y** `kickoff.yaml` no se modifica
### Requirement: Exención de Cambios Preexistentes sin Instantánea y de `seal --infer` (RF-KOF-07)

Las guardas de rama (veredicto de RF-BRA-19 de `change-branch-lifecycle`) NO DEBEN bloquear un cambio cuyo `kickoff.yaml` existe y no contiene instantánea (cambio preexistente sellado antes de esta capacidad). El retro-sellado `axiom sdd kickoff seal --infer` (RF-KOF-04) NO DEBE exigir una rama determinada y NO DEBE escribir instantánea, de modo que un cambio retro-sellado permanezca exento. El producto NO añade ninguna otra exención para cambios preexistentes: solo quedan exentos por nombre los sumideros acumulativos `e2e-cumulative` y `cumulative-<rol>` (RF-BRA-19).

#### Scenario: Cambio preexistente sin instantánea no se bloquea
- **DADO** un cambio con `kickoff.yaml` sin instantánea y el repositorio desplegado en la rama de integración
- **CUANDO** se ejecuta `axiom sdd status` y un escritor Go modifica la carpeta del cambio
- **ENTONCES** no se proyecta `blocked(wrong_branch)` y el escritor procede

#### Scenario: `seal --infer` funciona desde cualquier rama y no escribe instantánea
- **DADO** un cambio preexistente con artefactos y sin kickoff sellado, y el repositorio desplegado en la rama de integración
- **CUANDO** se ejecuta `axiom sdd kickoff seal --infer --change x`
- **ENTONCES** el retro-sellado se completa sin exigir rama
- **Y** el `kickoff.yaml` resultante no contiene instantánea

#### Scenario: El cambio retro-sellado sigue exento
- **DADO** el cambio retro-sellado del escenario anterior
- **CUANDO** se consulta el veredicto de rama
- **ENTONCES** es `not_applicable`

---

### Requirement: `kickoff seal` se Niega Fuera de la Rama que Corresponde (RF-KOF-08)

`axiom sdd kickoff seal` DEBE consultar el veredicto de rama (RF-BRA-19 de `change-branch-lifecycle`) para el repositorio del planning home antes de escribir y, ante `wrong_branch`, DEBE negarse sin escribir `kickoff.yaml`, con código de salida distinto de cero, el código tipado `wrong_branch` y la continuación exacta (`axiom branch ensure` con `--cwd`, `--change` y, si procede, `--lane`). En `monorepo-embedded` exige estar en la rama del cambio (o en una rama de rol del cambio). En `multirepo` exige que el repositorio de specs esté en su rama de integración y no en una rama de rol; los repositorios de código no se comprueban en el sellado. Si el cambio aún no tiene `kickoff.yaml`, los nombres esperados se obtienen de la política de `axiom.yaml`.

#### Scenario: Sellar en la rama de integración de un monorepo se rechaza
- **DADO** un monorepo con la política por defecto y el repositorio desplegado en `main`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** se rechaza con el código tipado `wrong_branch`, rama actual `main` y rama esperada `feat/x`
- **Y** la continuación es `axiom branch ensure --cwd <ruta> --change x`
- **Y** no se escribe `kickoff.yaml`
- **Y** el código de salida es distinto de cero

#### Scenario: Sellar en la rama del cambio procede
- **DADO** el mismo monorepo con el repositorio desplegado en `feat/x`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** el sellado se completa y se escribe `kickoff.yaml`

#### Scenario: Sellar en un multirepo exige la integración del repositorio de specs
- **DADO** un multirepo con el repositorio de specs desplegado en `feat/x-core`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** se rechaza con `wrong_branch` y rama esperada `main`

#### Scenario: Sellar en un multirepo con el repositorio de specs en su integración procede
- **DADO** un multirepo con el repositorio de specs en `main` y `repo-backend` desplegado en `main`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** el sellado se completa
- **Y** no se comprueba la rama de `repo-backend`
### Requirement: El Carril Declarado en el Proyecto Gobierna la Pregunta de Carril (RF-KOF-09)

Antes de decidir el carril de una solicitud de trabajo sustancial, el orquestador DEBE leer el campo `workflow` de `axiom sdd status` (RF-CLI-08 de `axiom-sdd-cli-integration`) y DEBE actuar según el carril proyectado, sin formular la pregunta bloqueante de selección de carril (RF-KOF-01) cuando el carril no es `ask`:

- `sdd`: el orquestador entra directamente al cuestionario de pre-vuelo de SDD (capacidad `sdd-preflight-configuration`) para el trabajo sustancial, sin preguntar el carril.
- `odd`: el orquestador trabaja en el carril ODD sin preguntar el carril, incluso ante un alcance inequívocamente arquitectónico. La persona puede pedir SDD explícitamente en una solicitud concreta, y esa petición prevalece para esa solicitud.
- `ask`: el comportamiento es el de RF-KOF-01 sin cambios.

El carril declarado cuenta como petición explícita de ese carril. NO modifica el trabajo pequeño y entendido, que en cualquier carril no crea artefactos duraderos. Si `workflow.source` es `no_workspace` o `invalid` (carril proyectado `ask`), o si el orquestador no puede obtener la proyección, DEBE comportarse como `ask`, nunca como `sdd` automático. La lectura de la proyección NO es la creación de un artefacto de trabajo ni una pregunta de gobernanza. El orden de la regla de rama (RF-KOF-05) no cambia: `axiom branch ensure` se ejecuta después de determinar el carril y antes del primer artefacto.

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

### Requirement: `kickoff.yaml` Registra la Autorización de Publicación Automática en el Estilo `continuous` (RF-KOF-10)

Cuando la persona elige el estilo `continuous` en el cuestionario de pre-vuelo, la pregunta DEBE nombrar la publicación de documentos en la propia opción: «sin paradas; sube los documentos a `origin/<rama>` al cerrar spec, design y tasks y al archivar», con la rama real del cambio. Si la persona elige esa opción, `axiom sdd kickoff seal` DEBE registrar en `kickoff.yaml` el campo opcional `docs_publication: auto`, dentro de la misma única escritura de sellado y con la semántica de sellado único de RF-KOF-06. El campo NO DEBE escribirse con el estilo `checkpointed` ni cuando la persona no autoriza la publicación, y la ausencia del campo significa «no se publica automáticamente». La serialización DEBE omitir el campo cuando está vacío (`omitempty`); la lectura de `kickoff.yaml` DEBE tolerar su ausencia y un binario anterior DEBE ignorarlo sin error. Sellar con `continuous` sin esa autorización DEBE seguir siendo posible. El campo NO modifica el estilo ni las compuertas y NO autoriza ninguna operación distinta de la publicación de documentos de RF-PUB-08 de `sdd-docs-publication`.
(Previously: la opción decía que subía los documentos al cerrar `tasks` y al archivar)

#### Scenario: La opción `continuous` nombra remoto y rama

- **DADO** el cuestionario de pre-vuelo de un cambio cuya rama es `inc/x`
- **CUANDO** se presenta la opción `continuous`
- **ENTONCES** su texto indica que sube los documentos a `origin/inc/x` al cerrar spec, design y tasks y al archivar

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

### Requirement: `kickoff seal` Prepara la Carpeta del Cambio si No Existe (RF-KOF-11)

`axiom sdd kickoff seal` DEBE dejar preparada la carpeta del cambio en el planning home (`changes/<cambio>/`) cuando no existe, en lugar de exigir que el orquestador o la persona la cree antes. La carpeta se crea después de superar la guarda de rama (RF-KOF-08): ante `wrong_branch` el comando se niega sin crear la carpeta ni escribir `kickoff.yaml`. Si la carpeta ya existe, el comando NO DEBE modificar ni borrar ninguno de sus ficheros. Preparar la carpeta NO crea ningún otro artefacto (`proposal.md`, specs, `design.md` ni `tasks.md`): solo la carpeta y `kickoff.yaml`. Un sellado que se rechaza por cualquier otro motivo (argumentos inválidos, declaración `sdd:` inválida) NO DEBE dejar la carpeta creada. El retro-sellado `seal --infer` (RF-KOF-04) conserva su comportamiento. El comando DEBE ser idempotente: repetirlo no cambia la carpeta ni su contenido. El orquestador y la skill de kickoff NO DEBEN pedir que se cree a mano la carpeta antes de sellar.

#### Scenario: Sellar un cambio sin carpeta la crea y escribe `kickoff.yaml`

- **DADO** un cambio `x` cuya carpeta `changes/x/` no existe y el repositorio desplegado en la rama que corresponde
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** existe `changes/x/kickoff.yaml`
- **Y** el comando termina con código de salida `0`
- **Y** no existe ningún otro fichero en `changes/x/`

#### Scenario: Sellar en la rama equivocada no crea la carpeta

- **DADO** un monorepo con la política por defecto, el repositorio desplegado en `main` y el cambio `x` sin carpeta
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** se rechaza con el código tipado `wrong_branch`
- **Y** no se crea `changes/x/`

#### Scenario: Un sellado rechazado por otro motivo no deja la carpeta

- **DADO** un cambio `x` sin carpeta y un `axiom.yaml` con `sdd:` inválido
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** se rechaza nombrando la clave o el valor inválido
- **Y** no se crea `changes/x/`

#### Scenario: Una carpeta existente se respeta

- **DADO** un cambio `x` con la carpeta ya creada y un `proposal.md` dentro
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** `proposal.md` permanece sin modificar
- **Y** se escribe `kickoff.yaml` junto a él

#### Scenario: Repetir el sellado es idempotente respecto de la carpeta

- **DADO** un cambio ya sellado cuya carpeta preparó el propio comando
- **CUANDO** se repite el comando con los mismos argumentos
- **ENTONCES** la carpeta y sus ficheros no cambian

#### Scenario: La skill de kickoff no pide crear la carpeta a mano

- **DADO** el texto de la skill y de las plantillas del orquestador que describen el sellado
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** no indican crear la carpeta del cambio antes de ejecutar `axiom sdd kickoff seal`

---

### Requirement: Instantánea de `sdd.iso` en `kickoff.yaml`, Escrita una Sola Vez por `seal` (RF-KOF-12)

`axiom sdd kickoff seal` DEBE copiar en `kickoff.yaml`, dentro de la misma única escritura de sellado y con la semántica de sellado único de RF-KOF-06, el valor que el proyecto declara en el bloque `sdd:` de `axiom.yaml` para la declaración ISO (clave `iso`, RF-ISO-01 de `sdd-iso-evidence`). Es una instantánea, igual que la de política git (RF-KOF-06) y que el campo `docs_publication` (RF-KOF-10): el valor sigue siendo del proyecto y nadie elige ISO por cambio. Las reglas ISO de ESE cambio (RF-ISO-03 a RF-ARC-10 y RF-ISO-10) DEBEN leerse de su instantánea y NO de `axiom.yaml` en vivo; la declaración de `axiom.yaml` solo es la fuente en el momento de sellar. Un proyecto que no declara `iso`, o lo declara desactivado, sella un cambio no ISO. `iso` es un booleano y la serialización DEBE omitir los campos vacíos (`omitempty`): con `iso` falso la clave no se escribe, de modo que solo un cambio ISO lleva la clave `iso` en `kickoff.yaml`. La lectura de `kickoff.yaml` DEBE tolerar la ausencia de la clave, y un cambio sin la clave `iso` cuenta como no ISO, igual que uno sin instantánea. Un binario anterior DEBE ignorar la clave sin error. Un cambio sellado antes de esta versión, sin instantánea de `sdd:`, DEBE contar como no ISO, que es el comportamiento actual. Cambiar `sdd:` en `axiom.yaml` solo afecta a los cambios sellados después: un cambio ya sellado conserva su valor hasta que se archive. `axiom sdd status` PUEDE avisar con una nota no bloqueante de la discrepancia entre la instantánea del cambio y la declaración vigente del proyecto, y NO DEBE bloquear por ella. El retrosellado `axiom sdd kickoff seal --infer` de un cambio preexistente DEBE congelar lo que ya rige ese cambio, como exige RF-KOF-04: NO escribe la clave `iso` (que, por la regla `omitempty` de este mismo requisito, queda ausente) y NO lee la declaración `sdd:` de `axiom.yaml`. Un cambio retro-sellado cuenta así como no ISO, igual que uno sin instantánea, y ningún cambio en curso pasa a ISO por retrosellarse. El retrosellado NO escribe la instantánea de política git de RF-KOF-06: RF-KOF-07 no cambia y el cambio retro-sellado sigue exento de las guardas de rama. Un `sdd:` inválido DEBE rechazar el sellado sin escribir nada (RF-ISO-01): nunca se sella «no ISO» por defecto ante una declaración inválida. Un segundo `seal` NO DEBE reescribir la instantánea (RF-KOF-06).

#### Scenario: El sellado copia `iso` en `kickoff.yaml`

- **DADO** un `axiom.yaml` con `sdd: {iso: true}`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...` desde la rama del cambio
- **ENTONCES** `kickoff.yaml` contiene `iso` activo junto a la instantánea de política git y en la misma escritura
- **Y** el proyecto sigue siendo la fuente: no existe ninguna opción por cambio para elegir otro valor

#### Scenario: Sin declaración se sella un cambio no ISO

- **DADO** un `axiom.yaml` sin `sdd:`
- **CUANDO** se sella un cambio
- **ENTONCES** `kickoff.yaml` no contiene la clave `iso`
- **Y** `axiom sdd status` proyecta el cambio como no ISO

#### Scenario: Activar `iso` a mitad no exige la matriz a un cambio ya sellado

- **DADO** un cambio `x` sellado sin `iso` y después el proyecto activa `iso` en `axiom.yaml`
- **CUANDO** se ejecuta `axiom sdd status x --json` en la fase `archive` con un `verify-report.md` libre
- **ENTONCES** no se proyecta ningún motivo de matriz, cobertura ni identidad ISO para `x`
- **Y** un cambio sellado después sí es ISO

#### Scenario: Desactivar `iso` a mitad sigue exigiendo la matriz a un cambio ya sellado

- **DADO** un cambio `x` sellado con `iso` y después el proyecto retira `iso` de `axiom.yaml`
- **CUANDO** se ejecuta `axiom sdd status x --json` en la fase `archive` con una matriz inválida
- **ENTONCES** el archivado sigue bloqueado con el motivo tipado de matriz (RF-ARC-10)

#### Scenario: Un cambio sellado antes de esta versión cuenta como no ISO

- **DADO** un `kickoff.yaml` sin instantánea de `sdd:` y un `axiom.yaml` actual con `sdd: {iso: true}`
- **CUANDO** se ejecutan `axiom sdd status` y `axiom sdd continue` sobre ese cambio
- **ENTONCES** el cambio se comporta como no ISO, sin errores
- **Y** no se proyecta ningún bloqueo de las reglas ISO

#### Scenario: La discrepancia entre la instantánea y el proyecto es una nota no bloqueante

- **DADO** un cambio sellado con `iso` y un `axiom.yaml` que ya no declara `iso`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el status incluye una nota que nombra la clave, el valor sellado y el valor vigente del proyecto
- **Y** ninguna dependencia queda bloqueada por esa nota

#### Scenario: Un segundo `seal` no reescribe la instantánea

- **DADO** un cambio sellado sin `iso` y un `axiom.yaml` que ahora declara `iso`
- **CUANDO** se ejecuta de nuevo `kickoff seal` desde la rama del cambio
- **ENTONCES** termina con éxito indicando «Kickoff ya estaba sellado»
- **Y** `kickoff.yaml` conserva los valores de la primera escritura

#### Scenario: El retrosellado no escribe `iso` y el cambio cuenta como no ISO

- **DADO** un cambio preexistente con artefactos, sin kickoff sellado, y un `axiom.yaml` con `sdd: {iso: true}`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --infer --change x`
- **ENTONCES** el cambio queda sellado sin la clave `iso` en `kickoff.yaml`, sin leer la declaración `sdd:`
- **Y** `axiom sdd status x --json` lo proyecta como no ISO, igual que un cambio sin instantánea
- **Y** el retrosellado no escribe la instantánea de política git (RF-KOF-07) ni cambia el rol que ya infería

#### Scenario: El retrosellado sigue exento de las guardas de rama

- **DADO** el cambio retro-sellado del escenario anterior y el repositorio desplegado en la rama de integración
- **CUANDO** se consulta el veredicto de rama
- **ENTONCES** es `not_applicable`
- **Y** el retrosellado no ha escrito la instantánea de política git ni la clave `iso`

#### Scenario: Un `sdd:` inválido rechaza el sellado

- **DADO** un `axiom.yaml` con `sdd:` inválido
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** se rechaza nombrando la clave o el valor inválido y no se escribe `kickoff.yaml`
- **Y** no se sella el cambio como no ISO por defecto

#### Scenario: Un binario anterior ignora el campo

- **DADO** un `kickoff.yaml` con `iso` activo
- **CUANDO** lo decodifica el esquema previo a este cambio
- **ENTONCES** la decodificación termina sin error
### Requirement: Declaración de Proyecto `sdd.explainer_html` con Valor por Defecto `ask` (RF-KOF-13)

`axiom.yaml` PUEDE declarar, dentro del bloque `sdd:`, la clave `explainer_html` con uno de tres valores: `ask`, `always` o `never`. El valor por defecto, cuando la clave o el bloque `sdd:` están ausentes, DEBE ser `ask`. El valor DEBE validarse con el decodificador estricto del bloque `sdd:` (RF-ISO-01 de `sdd-iso-evidence`): un valor no admitido DEBE fallar en `axiom workspace validate`, al cargar la configuración y al sellar, con un mensaje en español que nombre `sdd.explainer_html`, el valor y los admitidos. `axiom.example.yaml` DEBE documentar la clave y sus tres valores. La declaración es por proyecto y exclusiva de SDD: no afecta al carril ODD.

#### Scenario: Sin declaración el valor vigente es `ask`

- **DADO** un `axiom.yaml` sin el bloque `sdd:` o con `sdd:` sin la clave `explainer_html`
- **CUANDO** se resuelve el valor vigente del proyecto
- **ENTONCES** el valor es `ask`

#### Scenario: Valor inválido en `axiom.yaml` falla en validación y al sellar

- **DADO** un `axiom.yaml` con `sdd: {explainer_html: siempre}`
- **CUANDO** se ejecuta `axiom workspace validate` y, por separado, `axiom sdd kickoff seal` sobre un cambio de ese proyecto
- **ENTONCES** ambos reportan en español la clave `sdd.explainer_html`, el valor y los admitidos
- **Y** el código de salida es distinto de cero en ambos
- **Y** el sellado no escribe `kickoff.yaml` ni degrada el valor a `ask`

#### Scenario: La plantilla de ejemplo documenta la clave

- **DADO** `axiom.example.yaml`
- **CUANDO** se valida con `axiom workspace validate`
- **ENTONCES** es válido
- **Y** documenta `explainer_html` con `ask`, `always` y `never`

---

### Requirement: `kickoff.yaml` Registra el Campo Sellado `explainer_html` con Valores `yes` o `no` (RF-KOF-14)

`kickoff.yaml` PUEDE registrar el campo `explainer_html` con uno de dos valores: `yes` o `no`. El campo DEBE omitirse cuando no está sellado (`omitempty`) y su ausencia DEBE equivaler a `no`. El campo DEBE escribirse una sola vez, por `axiom sdd kickoff seal`, y DEBE ser independiente del estilo de ejecución (`continuous` o `checkpointed`), del rol y de `docs_publication`. Un valor distinto de `yes` o `no` en un `kickoff.yaml` DEBE rechazarse como inválido. Un cambio sellado antes de este cambio, sin el campo, NO DEBE cambiar de comportamiento. Un binario anterior ignora el campo al leer `kickoff.yaml`.

#### Scenario: Sellado con `yes` queda registrado

- **DADO** un cambio sin sellar en un proyecto con `explainer_html: ask`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --execution-style continuous --explainer-html yes`
- **ENTONCES** `kickoff.yaml` contiene `explainer_html: yes`

#### Scenario: Ausente equivale a `no`

- **DADO** un cambio sellado antes de este cambio, cuyo `kickoff.yaml` no tiene `explainer_html`
- **CUANDO** se consulta el valor sellado
- **ENTONCES** el valor efectivo es `no`
- **Y** el resto del kickoff se lee exactamente igual que antes

#### Scenario: Es independiente del estilo de ejecución

- **DADO** dos cambios sellados con `--explainer-html yes`, uno `continuous` y otro `checkpointed`
- **CUANDO** se leen sus `kickoff.yaml`
- **ENTONCES** ambos registran `explainer_html: yes`
- **Y** el estilo, los roles y `docs_publication` de cada uno no cambian por ello

#### Scenario: Valor inválido en `kickoff.yaml`

- **DADO** un `kickoff.yaml` con `explainer_html: quizás`
- **CUANDO** se lee el kickoff
- **ENTONCES** se rechaza como inválido, nombrando el campo y los admitidos (`yes`, `no`)

---

### Requirement: La Bandera `--explainer-html` y la Resolución contra el Valor del Proyecto (RF-KOF-15)

`axiom sdd kickoff seal` DEBE aceptar la bandera `--explainer-html yes|no`. El valor sellado DEBE resolverse así:

- Con `sdd.explainer_html: always`, se sella `yes` sin preguntar; con `never`, se sella `no` sin preguntar.
- Una bandera cuyo valor contradiga `always` (`--explainer-html no`) o `never` (`--explainer-html yes`) DEBE rechazarse con un error que nombre ambos valores, sin escribir nada. Una bandera coherente con `always` o `never` se acepta y no cambia el resultado.
- Con `ask` y la bandera presente, se sella el valor de la bandera.
- Con `ask` y sin bandera, se sella `no` y la salida DEBE incluir un aviso de que no se ha preguntado y de cómo activar el HTML explicativo; el sellado termina con éxito.
- Un valor de la bandera distinto de `yes` o `no` DEBE rechazarse, nombrando los admitidos, sin escribir nada.
- `--infer` (retrosellado de un cambio preexistente) DEBE rechazar la bandera `--explainer-html`, no DEBE escribir el campo ni heredarlo del proyecto, y DEBE dejarlo ausente (equivale a `no`).

La salida de `seal`, en texto y en JSON, DEBE informar del valor sellado. La ayuda del subcomando DEBE documentar la bandera.

#### Scenario: `always` sella `yes` sin bandera

- **DADO** un proyecto con `sdd.explainer_html: always`
- **CUANDO** se ejecuta `axiom sdd kickoff seal` sin `--explainer-html`
- **ENTONCES** `kickoff.yaml` contiene `explainer_html: yes`
- **Y** no se emite ningún aviso

#### Scenario: `never` sella `no` sin bandera

- **DADO** un proyecto con `sdd.explainer_html: never`
- **CUANDO** se ejecuta `axiom sdd kickoff seal` sin `--explainer-html`
- **ENTONCES** `kickoff.yaml` contiene `explainer_html: no`

#### Scenario: Una bandera contradictoria con `always` o `never` se rechaza sin escribir

- **DADO** un proyecto con `sdd.explainer_html: never`, y otro con `always`
- **CUANDO** se ejecuta `seal --explainer-html yes` en el primero y `seal --explainer-html no` en el segundo
- **ENTONCES** ambos se rechazan con un error que nombra el valor del proyecto y el de la bandera
- **Y** no se escribe `kickoff.yaml` ni se crea la carpeta del cambio

#### Scenario: Una bandera coherente con `always` se acepta

- **DADO** un proyecto con `sdd.explainer_html: always`
- **CUANDO** se ejecuta `seal --explainer-html yes`
- **ENTONCES** el sellado tiene éxito con `explainer_html: yes`

#### Scenario: Con `ask` la bandera decide

- **DADO** un proyecto con `sdd.explainer_html: ask`
- **CUANDO** se ejecuta `seal --explainer-html no` y, en otro cambio, `seal --explainer-html yes`
- **ENTONCES** se sellan `no` y `yes` respectivamente, sin aviso

#### Scenario: `ask` sin bandera sella `no` con aviso

- **DADO** un proyecto con `sdd.explainer_html: ask` (o sin declararlo)
- **CUANDO** se ejecuta `axiom sdd kickoff seal` sin `--explainer-html`
- **ENTONCES** `kickoff.yaml` contiene `explainer_html: no`
- **Y** la salida incluye un aviso de que no se ha preguntado
- **Y** el sellado termina con código de salida `0`

#### Scenario: Valor inválido en la bandera

- **DADO** cualquier proyecto válido
- **CUANDO** se ejecuta `seal --explainer-html quizás`
- **ENTONCES** se rechaza nombrando los valores admitidos `yes` y `no`
- **Y** no se escribe nada

#### Scenario: `--infer` rechaza la bandera y no escribe el campo

- **DADO** un cambio preexistente con artefactos y sin `kickoff.yaml`, en un proyecto con `sdd.explainer_html: always`
- **CUANDO** se ejecuta `seal --infer --explainer-html yes`
- **ENTONCES** se rechaza porque `--infer` no admite la bandera
- **Y** al ejecutar `seal --infer` sin bandera, el `kickoff.yaml` resultante no contiene `explainer_html`

#### Scenario: Cambio sin carpeta

- **DADO** un cambio nombrado cuya carpeta no existe
- **CUANDO** se ejecuta `seal --explainer-html yes` con un valor válido
- **ENTONCES** el sellado crea la carpeta (RF-KOF-11) y registra `explainer_html: yes`

#### Scenario: La salida informa del valor sellado

- **DADO** un sellado correcto con `--json`
- **CUANDO** se lee la salida
- **ENTONCES** incluye el valor sellado de `explainer_html`

---

### Requirement: El Status Proyecta el Valor de Proyecto y el Valor Sellado de `explainer_html` (RF-KOF-16)

`axiom sdd status --json` DEBE proyectar de forma aditiva: (a) el valor vigente del proyecto (`ask`, `always` o `never`), con o sin cambio seleccionado, para que el orquestador sepa si debe preguntar antes de sellar; y (b) el valor sellado del cambio consultado dentro de `governance.kickoff` (`yes` o `no`), para que las fases sepan si deben generar el HTML. Un cambio sin el campo sellado se proyecta como `no` o sin la clave, de modo que un consumidor anterior no observa diferencia. Con un `sdd:` inválido el status DEBE proyectar la declaración como inválida en lugar de asumir `ask`. Ninguna clave existente del status DEBE cambiar de nombre, tipo ni valor, y la proyección NO DEBE alterar `nextRecommended`, las dependencias ni los motivos de bloqueo. El contrato publicado (`sdd-status-contract.md`) DEBE documentar ambas proyecciones.

#### Scenario: Valor de proyecto sin cambio seleccionado

- **DADO** un `axiom.yaml` con `sdd: {explainer_html: always}` y ningún cambio seleccionado
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `changeName` es `null`
- **Y** el status proyecta el valor de proyecto `always`

#### Scenario: Valor sellado en `governance.kickoff`

- **DADO** un cambio sellado con `explainer_html: yes`
- **CUANDO** se ejecuta `axiom sdd status --change <x> --json`
- **ENTONCES** `governance.kickoff` proyecta el valor sellado `yes`

#### Scenario: Cambio sellado antes de este cambio

- **DADO** un cambio cuyo `kickoff.yaml` no tiene `explainer_html`
- **CUANDO** se consulta el status
- **ENTONCES** el valor sellado se proyecta como `no` o se omite
- **Y** ninguna otra clave del status cambia

#### Scenario: Declaración inválida

- **DADO** un `axiom.yaml` con `sdd: {explainer_html: quizás}`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el status proyecta la declaración como inválida con el error de `sdd.explainer_html`
- **Y** no asume `ask`

#### Scenario: La proyección no altera el encadenado

- **DADO** el mismo cambio con `explainer_html: yes` y con `explainer_html: no`
- **CUANDO** se consulta el status
- **ENTONCES** `nextRecommended`, las dependencias y los motivos de bloqueo son idénticos
### Requirement: `sdd-tasks` Genera `pruebas.md` solo en SDD, con la Función Activa y Después del Design (RF-KOF-17)

La fase `sdd-tasks` DEBE escribir `pruebas.md` en la carpeta del cambio si y solo si se cumplen a la vez: el cambio va por el carril SDD, y `axiom sdd status --json` proyecta `acceptanceTests` activo (RF-KOF-22), esto es, `acceptance_tests.enabled: true` en `axiom.yaml`. Con `enabled: false`, sin el bloque o con un bloque inválido, NO DEBE generar nada, y el digest de `tasks`, el status y el resto del flujo NO DEBEN cambiar. Un cambio ODD NUNCA genera `pruebas.md`. El carácter `continuous` o `checkpointed` del cambio no altera la regla.

Al generarlo, `sdd-tasks` DEBE minimizar el número de casos aplicando, por este orden: (1) trazado obligatorio, esto es, cada requisito y cada escenario ADDED o MODIFIED queda citado; (2) particiones de equivalencia y valores límite; (3) pairwise solo con tres o más parámetros independientes; (4) fusión de casos con las mismas precondiciones. DEBE ejecutar después `axiom sdd tests check` (RF-KOF-21) y corregir el fichero hasta que no queden hallazgos o dejar constancia de los que no pueda resolver. La minimización es responsabilidad del modelo: Go solo comprueba e informa. En la parada `tasks` presenta `pruebas.md` según RF-GAT-32.

`sdd-apply` NO DEBE ampliar el alcance de `pruebas.md`: solo PUEDE refinar rutas o URL en `Pasos`, sin añadir, quitar ni reasignar casos ni alterar `Cubre`. La doctrina común DEBE vivir en `_shared/sdd-acceptance-tests.md`, enlazada desde la skill `sdd-tasks` y desde sus agentes delgados (claude, cursor y copilot), y un test de pines DEBE fijar: la condición SDD y `acceptanceTests` activo, que ODD nunca genera, las cuatro capas, la ejecución de `axiom sdd tests check` y la regla de refinado de `sdd-apply`.

#### Scenario: Función activa en SDD
- **DADO** un fixture `multirepo` con `acceptance_tests.enabled: true`, un cambio SDD tras design y el status con `acceptanceTests` activo
- **CUANDO** se ejecuta la fase `sdd-tasks`
- **ENTONCES** la doctrina exige escribir `pruebas.md` con IDs `P-NN` y ejecutar `axiom sdd tests check`

#### Scenario: Función no activa no genera nada
- **DADO** un proyecto con `enabled: false` o sin el bloque
- **CUANDO** se ejecuta la fase `sdd-tasks`
- **ENTONCES** no se genera `pruebas.md`
- **Y** el status no proyecta aviso y el digest de `tasks` no cambia

#### Scenario: ODD nunca genera
- **DADO** un cambio por el carril ODD en un proyecto con la función activa
- **CUANDO** se ejecuta el flujo ODD
- **ENTONCES** no se genera `pruebas.md`
- **Y** la doctrina lo condiciona a SDD, lo que fija un test de pines

#### Scenario: `sdd-apply` solo refina rutas o URL
- **DADO** un `pruebas.md` aprobado
- **CUANDO** `sdd-apply` lo refina
- **ENTONCES** solo modifica rutas o URL de `Pasos`
- **Y** no añade casos ni cambia `Cubre`, según `_shared/sdd-acceptance-tests.md`

### Requirement: Gramática Cerrada de la Tabla de `pruebas.md` (RF-KOF-18)

`pruebas.md` DEBE contener una única tabla markdown cuyo parseo sea determinista, con estas reglas:

- **Cabecera.** Las nueve columnas, con estos nombres exactos y en castellano: `ID`, `Cubre`, `Rol`, `Tipo`, `Prioridad`, `Entornos`, `Precondiciones`, `Pasos`, `Resultado esperado`. Se localizan por nombre (tras recortar espacios, sensible a mayúsculas), en cualquier orden; una columna ausente, duplicada o con otro nombre hace la forma de la tabla inválida. Tras la cabecera va la línea separadora de markdown. Cada fila DEBE tener tantas celdas como columnas. Un `|` dentro de una celda se escribe `\|`. Todo texto fuera de la tabla (títulos, notas) se ignora.
- **ID.** `P-` seguido de al menos dos dígitos (`P-01`, `P-12`, `P-100`). DEBE ser único en el fichero y estable: no se renumera al editar, de modo que un caso conserva su ID mientras exista. Un ID mal formado o repetido es un hallazgo.
- **Tipo.** Exactamente `manual` o `automatizable`.
- **Prioridad.** Exactamente `alta`, `media` o `baja`.
- **Entornos.** Lista separada por `;` (con espacios opcionales alrededor), no vacía, sin repetidos, cuyos elementos pertenezcan, con comparación exacta, a `acceptance_tests.environments` del proyecto.
- **Rol.** Un único identificador que sea un rol sellado del cambio en `kickoff.yaml` (en multirepo el sellado es siempre `checkpointed`).
- **Cubre.** Según RF-KOF-19.
- **Precondiciones.** Texto libre; PUEDE estar vacío.
- **Pasos** y **Resultado esperado.** Texto libre no vacío.

Una celda con un valor fuera de su dominio, o una celda obligatoria vacía, es un hallazgo con el número de línea de la fila. La ausencia de tabla o de cabecera es el hallazgo de forma de tabla inválida.

#### Scenario: Tabla válida
- **DADO** un `pruebas.md` con las nueve columnas, IDs `P-01` y `P-02`, `Tipo` `manual`, `Prioridad` `alta`, `Entornos` `dev; pre` declarados y un `Rol` sellado
- **CUANDO** se ejecuta `axiom sdd tests check`
- **ENTONCES** la tabla se parsea sin hallazgos de forma

#### Scenario: Columnas en otro orden
- **DADO** una tabla con las nueve columnas en un orden distinto
- **CUANDO** se parsea
- **ENTONCES** cada celda se asocia a su columna por nombre

#### Scenario: Cabecera inválida
- **DADO** una tabla sin la columna `Cubre`, con una columna extra o con `Prioridad` escrita `prioridad`
- **CUANDO** se parsea
- **ENTONCES** se informa de la forma de tabla inválida con la columna responsable

#### Scenario: ID mal formado o duplicado
- **DADO** una fila con ID `P1` o `p-01`, y dos filas con el ID `P-03`
- **CUANDO** se parsea
- **ENTONCES** cada caso produce un hallazgo con la línea de la fila

#### Scenario: Tipo o prioridad fuera de dominio
- **DADO** una fila con `Tipo` `exploratoria` y otra con `Prioridad` `urgente`
- **CUANDO** se comprueba
- **ENTONCES** cada una produce un hallazgo con su valor y sus valores admitidos

#### Scenario: Entorno no declarado o repetido
- **DADO** `environments: [dev, pre]` y una fila con `Entornos` `prod`, otra con `dev; dev` y otra vacía
- **CUANDO** se comprueba
- **ENTONCES** cada fila produce un hallazgo de entorno inválido

#### Scenario: Rol no sellado
- **DADO** un cambio con el rol único `fullstack` y una fila con `Rol` `backend`
- **CUANDO** se comprueba
- **ENTONCES** produce un hallazgo de rol inválido que lista los roles sellados

#### Scenario: Barra escapada en una celda
- **DADO** una celda de `Pasos` con `a \| b`
- **CUANDO** se parsea
- **ENTONCES** la celda contiene `a | b` y la fila conserva sus nueve celdas

### Requirement: Sintaxis de `Cubre` y Reconocimiento de IDs solo por el Puerto `reqid` (RF-KOF-19)

La columna `Cubre` DEBE ser una lista no vacía separada por `;`, porque los nombres de escenario pueden contener comas. Cada entrada, recortada de espacios, es de una de estas dos formas:

- `ID`: cita un requisito;
- `ID#<nombre exacto del escenario>`: cita ese escenario de ese requisito. El separador es el primer `#`; el nombre es el texto del encabezado `#### Scenario:` recortado de espacios y con comparación exacta, sensible a mayúsculas.

Un ID SOLO se reconoce con el puerto `internal/reqid`: `REQ-<n>.<m>`, `RF-<DOM>-<NN>`, los identificadores provisionales y el `sdd.id_pattern` del proyecto. No hay ningún reconocedor adicional. Una entrada cuyo ID no reconoce `reqid`, o que reconociéndolo no figura en la delta del cambio, es un hallazgo de ID desconocido; una entrada `ID#nombre` cuyo requisito existe pero no tiene ese escenario es un hallazgo de escenario desconocido; una celda `Cubre` vacía o con una entrada vacía es el hallazgo de caso sin cubre. Un escenario solo queda cubierto si se cita explícitamente con `ID#nombre`; un requisito queda cubierto por su ID suelto o por cualquiera de sus escenarios citados. Un nombre de escenario que contenga `;` no puede citarse: es un hallazgo de escenario no citable.

#### Scenario: Requisito citado por su ID suelto
- **DADO** una fila con `Cubre` `RF-KOF-17`
- **CUANDO** se calcula la cobertura
- **ENTONCES** el requisito queda cubierto
- **Y** ninguno de sus escenarios queda cubierto por esa entrada

#### Scenario: Escenario citado con `ID#nombre`
- **DADO** una fila con `Cubre` `RF-KOF-17#Función activa en SDD; RF-KOF-17#ODD nunca genera`
- **CUANDO** se calcula la cobertura
- **ENTONCES** ambos escenarios y el requisito quedan cubiertos

#### Scenario: Nombre de escenario con comas
- **DADO** un escenario cuyo nombre contiene comas y una entrada `RF-XXX-01#nombre, con comas`
- **CUANDO** se calcula la cobertura
- **ENTONCES** la entrada se interpreta como una sola cita

#### Scenario: ID no reconocido por `reqid`
- **DADO** una entrada `ABC-9` que `reqid` no reconoce
- **CUANDO** se comprueba
- **ENTONCES** produce un hallazgo de ID desconocido

#### Scenario: ID o escenario inexistente en la delta
- **DADO** una entrada con un ID válido ausente de la delta, y otra `ID#nombre` con un nombre inexistente
- **CUANDO** se comprueba
- **ENTONCES** la primera produce un hallazgo de ID desconocido y la segunda uno de escenario desconocido

#### Scenario: Caso sin cubre
- **DADO** una fila con `Cubre` vacío o con `RF-X-01;;RF-X-02`
- **CUANDO** se comprueba
- **ENTONCES** produce un hallazgo de caso sin cubre con la línea de la fila

#### Scenario: Identificador provisional y patrón del proyecto
- **DADO** un requisito con ID provisional y otro con un ID del `sdd.id_pattern` del proyecto
- **CUANDO** se citan en `Cubre`
- **ENTONCES** `reqid` los reconoce y quedan cubiertos

### Requirement: `acceptancetests.Check` Cruza `pruebas.md` con las Specs Delta y Define los Hallazgos y el Informe (RF-KOF-20)

`internal/acceptancetests.Check` DEBE parsear `pruebas.md` y cruzarlo con los requisitos y escenarios de los bloques ADDED y MODIFIED de las specs delta del cambio (`specs/<capacidad>/spec.md`), usando `Scenarios` y `AffectedFromDelta` de `internal/isoevidence`. Los bloques REMOVED y RENAMED quedan fuera de la base de cobertura. Es la única implementación del cálculo, compartida por el verbo, el status y el dashboard. Es de solo lectura y determinista: la misma entrada da el mismo informe, con los hallazgos ordenados por fichero, línea y código.

Son hallazgos que hacen fallar la comprobación, con código estable:

| Código | Condición |
|--------|-----------|
| `pruebas_ausente` | La función está activa, hay `tasks` y falta `pruebas.md` |
| `tabla_invalida` | Tabla o cabecera ausente o con forma no válida (RF-KOF-18) |
| `celda_invalida` | Celda obligatoria vacía o valor fuera de dominio (`Tipo`, `Prioridad`) |
| `id_caso_invalido` | `P-NN` mal formado |
| `id_caso_duplicado` | `P-NN` repetido |
| `caso_sin_cubre` | `Cubre` vacío o con una entrada vacía |
| `entorno_invalido` | Entorno ausente, repetido o fuera de `acceptance_tests.environments` |
| `rol_invalido` | Rol que no es un rol sellado del cambio |
| `id_desconocido` | ID no reconocido por `reqid` o ausente de la delta |
| `escenario_desconocido` | `ID#nombre` cuyo requisito no tiene ese escenario |
| `requisito_sin_id` | Requisito ADDED o MODIFIED sin ID citable |
| `id_duplicado_en_delta` | El mismo ID figura en dos requisitos ADDED o MODIFIED de la delta |
| `escenario_ambiguo` | Nombre de escenario repetido dentro de un mismo requisito, o que contiene `;` |
| `requisito_sin_cubrir` | Requisito con ID que ningún caso cita |
| `escenario_sin_cubrir` | Escenario que ningún caso cita con `ID#nombre` |

El informe DEBE contener: los escenarios cubiertos sobre el total (`M/M`), los requisitos cubiertos sobre el total (`M/M`, donde el total incluye los requisitos sin ID, que no pueden cubrirse), el número de casos y la lista de hallazgos con código, línea y mensaje. Sin hallazgos, el informe declara cobertura completa. El Check NO DEBE modificar ningún fichero.

#### Scenario: Cobertura completa
- **DADO** una delta con 2 requisitos y 5 escenarios, y un `pruebas.md` válido con 4 casos que citan todos con `ID#nombre`
- **CUANDO** se ejecuta `Check`
- **ENTONCES** el informe declara escenarios 5/5, requisitos 2/2, 4 casos y ningún hallazgo

#### Scenario: Escenario sin cubrir
- **DADO** un requisito citado solo por su ID suelto y con 2 escenarios
- **CUANDO** se ejecuta `Check`
- **ENTONCES** informa de `escenario_sin_cubrir` por cada escenario
- **Y** el requisito cuenta como cubierto

#### Scenario: Requisito sin cubrir
- **DADO** un requisito ADDED que ningún caso cita
- **CUANDO** se ejecuta `Check`
- **ENTONCES** informa de `requisito_sin_cubrir` y de `escenario_sin_cubrir` por cada uno de sus escenarios

#### Scenario: Requisito sin ID
- **DADO** un requisito ADDED o MODIFIED sin ID reconocible por `reqid`
- **CUANDO** se ejecuta `Check`
- **ENTONCES** informa de `requisito_sin_id` y no lo ignora en silencio
- **Y** cuenta en el total de requisitos del informe

#### Scenario: Escenario ambiguo
- **DADO** un requisito con dos escenarios del mismo nombre
- **CUANDO** se ejecuta `Check`
- **ENTONCES** informa de `escenario_ambiguo` con el nombre repetido

#### Scenario: MODIFIED cuenta y REMOVED no
- **DADO** una delta con un requisito MODIFIED, otro REMOVED y otro RENAMED
- **CUANDO** se ejecuta `Check`
- **ENTONCES** solo el MODIFIED entra en la base de cobertura
- **Y** citar el REMOVED produce `id_desconocido`

#### Scenario: Un hallazgo por cada causa
- **DADO** casos con un ID desconocido, un escenario desconocido, un entorno no declarado, un rol no sellado, un tipo fuera de dominio, una prioridad fuera de dominio y un `P-NN` duplicado
- **CUANDO** se ejecuta `Check`
- **ENTONCES** informa de `id_desconocido`, `escenario_desconocido`, `entorno_invalido`, `rol_invalido`, `celda_invalida` (dos veces) e `id_caso_duplicado`

#### Scenario: Determinismo y solo lectura
- **DADO** el mismo cambio
- **CUANDO** se ejecuta `Check` dos veces
- **ENTONCES** los informes son idénticos
- **Y** ningún fichero del cambio se modifica

### Requirement: Verbo de Solo Lectura `axiom sdd tests check` (RF-KOF-21)

`axiom sdd tests check --cwd <repo> --change <c> [--json]` DEBE ejecutar `acceptancetests.Check` y presentar el informe de RF-KOF-20. Es de solo lectura, vive bajo un subdespachador `tests` (como `gate`) y se registra en `cmd/axiom/main.go`, en su ayuda y en `main_test.go`. En texto DEBE imprimir los escenarios M/M, los requisitos M/M, el número de casos y cada hallazgo con código, línea y mensaje; con `--json` DEBE emitir un objeto con `active`, `environments`, `present`, `cases`, `scenarios {covered, total}`, `requirements {covered, total}` y `findings [{code, line, message}]`. El objeto PUEDE llevar claves adicionales de identificación (por ejemplo `schema`, `change`, `file` o `error`); un consumidor NO DEBE rechazarlas.

El verbo DEBE terminar con éxito si no hay hallazgos y con error (código de salida distinto de cero) si hay algún hallazgo, o si la función está activa y falta `pruebas.md` después de `tasks` (`pruebas_ausente`). Con la función no activa y sin `pruebas.md`, DEBE terminar con éxito indicando que las pruebas de aceptación no están activadas, sin hallazgos. Con la función no activa y con `pruebas.md` presente, comprueba el fichero igualmente. Un cambio inexistente o sin specs delta es un error de uso, no un informe.

#### Scenario: Informe en texto y JSON
- **DADO** un cambio con `pruebas.md` válido
- **CUANDO** se ejecuta `axiom sdd tests check --cwd <repo> --change <c>` y con `--json`
- **ENTONCES** ambos salen con código 0 y muestran los mismos escenarios M/M, requisitos M/M y número de casos

#### Scenario: Con hallazgos sale en error
- **DADO** un `pruebas.md` con un escenario sin cubrir
- **CUANDO** se ejecuta el verbo
- **ENTONCES** sale con código distinto de cero e imprime `escenario_sin_cubrir` con su línea

#### Scenario: Activa y falta `pruebas.md`
- **DADO** la función activa, `tasks` terminado y sin `pruebas.md`
- **CUANDO** se ejecuta el verbo
- **ENTONCES** sale con error y `pruebas_ausente`

#### Scenario: No activa y sin fichero
- **DADO** `enabled: false` o sin bloque y sin `pruebas.md`
- **CUANDO** se ejecuta el verbo
- **ENTONCES** sale con código 0 e indica que las pruebas de aceptación no están activadas

#### Scenario: Registro del verbo
- **CUANDO** se ejecutan `axiom help` y las pruebas de `main_test.go`
- **ENTONCES** la ayuda lista `sdd tests check` y la prueba falla si se elimina del despachador

### Requirement: El Status Proyecta `acceptanceTests` de Forma Aditiva y Avisa sin Bloquear la Ausencia de `pruebas.md` (RF-KOF-22)

`axiom sdd status --json` DEBE proyectar la clave aditiva `acceptanceTests` con el estado vivo de `axiom.yaml`: la activación y los entornos, o el error del bloque si es inválido (sin asumir un valor), y, si existe `pruebas.md`, el resultado de `acceptancetests.Check`: número de casos, escenarios y requisitos cubiertos (M/M) y número de hallazgos. Si la función está activa, `tasks` está terminado y falta `pruebas.md`, DEBE emitir un aviso informativo. Con `enabled: false` o sin el bloque, DEBE proyectar la función como no activa y NO DEBE emitir ningún aviso; omitir la clave `acceptanceTests` equivale a proyectarla como no activa, para que el JSON del status de un proyecto sin el bloque no cambie.

Cuando existe `pruebas.md` y `pruebas-estado.yaml` (RF-KOF-25) con entradas, la proyección DEBE añadir de forma aditiva el resumen de marcas: por cada entorno, el número de casos con marca vigente `ok` sobre el total de casos aplicables a ese entorno, y el número de marcas caducadas aparte. Sin `pruebas-estado.yaml`, o en un proyecto `monorepo-embedded`, la clave de marcas NO DEBE proyectarse. Las tres presentaciones del resumen (status, presentación de `integration` y `archive-report`) DEBEN salir del mismo cálculo.
(Previously: la proyección no incluía ningún resumen de marcas)

La proyección y el aviso NO DEBEN alterar `nextRecommended`, las dependencias, los motivos de bloqueo ni los digests, y NO DEBEN bloquear la compuerta ni el avance. Ninguna clave existente del status DEBE cambiar de nombre, tipo ni valor. El contrato publicado (`_shared/sdd-status-contract.md`) DEBE documentar la clave y el resumen de marcas.

#### Scenario: Función activa con `pruebas.md`
- **DADO** la función activa y un `pruebas.md` con 2 hallazgos
- **CUANDO** se ejecuta `axiom sdd status --change <c> --json`
- **ENTONCES** `acceptanceTests` proyecta activación, entornos, cobertura y 2 hallazgos

#### Scenario: Aviso por ausencia sin bloqueo
- **DADO** la función activa, `tasks` terminado y sin `pruebas.md`
- **CUANDO** se consulta el status
- **ENTONCES** `acceptanceTests` incluye un aviso
- **Y** `nextRecommended` y las dependencias son idénticos a los del mismo cambio con `pruebas.md` ausente y la función no activa

#### Scenario: Función no activa
- **DADO** `enabled: false` o sin el bloque
- **CUANDO** se consulta el status
- **ENTONCES** no hay aviso y los digests no cambian

#### Scenario: Declaración inválida
- **DADO** un bloque `acceptance_tests` inválido
- **CUANDO** se consulta el status
- **ENTONCES** `acceptanceTests` proyecta el error del bloque
- **Y** no se asume activación

#### Scenario: Resumen por entorno
- **DADO** 5 casos aplicables a `pre` con 5 marcas vigentes `ok`, y 2 casos aplicables a `local` con 1 marca vigente `ok` y 1 marca caducada
- **CUANDO** se consulta el status
- **ENTONCES** el resumen proyecta `pre` con 5 de 5 `ok` y `local` con 1 de 2 `ok` y 1 marca caducada

#### Scenario: Sin marcas no hay clave
- **DADO** un cambio con `pruebas.md` y sin `pruebas-estado.yaml`
- **CUANDO** se consulta el status
- **ENTONCES** `acceptanceTests` no incluye el resumen de marcas y el resto del JSON no cambia

#### Scenario: Las marcas no alteran el enrutado
- **DADO** el mismo cambio con y sin `pruebas-estado.yaml`
- **CUANDO** se consulta el status
- **ENTONCES** `nextRecommended`, las dependencias, los motivos de bloqueo y los digests son idénticos
### Requirement: El Archivado Reescribe los Provisionales de la Columna `Cubre` de `pruebas.md` (RF-KOF-23)

Al archivar, `internal/livespec` DEBE reescribir los IDs provisionales a su ID definitivo también en la columna `Cubre` de `pruebas.md`, como un tercer tipo de fichero del cambio (junto a los dos existentes), con la misma tabla de correspondencia que para el resto de ficheros. La reescritura DEBE tocar solo el ID de cada entrada de `Cubre` (incluida la forma `ID#nombre`, donde el nombre del escenario no cambia), conservar el resto del fichero byte a byte, y ser idempotente. Sin `pruebas.md`, DEBE ser una operación nula. Un ID de `Cubre` que no es provisional no se toca.

#### Scenario: Provisionales reescritos en `Cubre`
- **DADO** un `pruebas.md` con `Cubre` `RF-PROV-1#Escenario A; RF-PROV-1`
- **CUANDO** se archiva el cambio y `RF-PROV-1` se integra como `RF-KOF-30`
- **ENTONCES** `Cubre` queda `RF-KOF-30#Escenario A; RF-KOF-30`

#### Scenario: El resto del fichero no cambia
- **DADO** el mismo `pruebas.md`
- **CUANDO** se archiva
- **ENTONCES** las demás columnas y el texto fuera de la tabla no cambian

#### Scenario: Idempotencia y ausencia
- **DADO** un `pruebas.md` ya reescrito, y un cambio sin `pruebas.md`
- **CUANDO** se vuelve a archivar o se archiva
- **ENTONCES** no hay cambios y no se produce error

#### Scenario: IDs definitivos intactos
- **DADO** una entrada `Cubre` con un ID no provisional
- **CUANDO** se archiva
- **ENTONCES** la entrada no cambia
### Requirement: Con Plugin Activo el WI y la Fase 1 Preceden a `ensure` y a `seal` (RF-KOF-24)

Con el plugin de work items activo (la respuesta de `axiom change start` no es «no aplica»), el orquestador MUST, tras elegir el carril SDD y antes de nombrar el cambio, ejecutar `axiom change start` para saber que el WI es obligatorio y, si la persona no lo ha dado, MUST pedirlo y no avanzar. Con el WI, MUST ejecutar la fase 1 (`axiom change start --from <id|url>`) antes de `axiom branch ensure` y de `axiom sdd kickoff seal`, MUST usar como nombre del cambio el nombre derivado que devuelve la fase 1 para `ensure --change`, `seal --change` y todo comando posterior, y MUST ejecutar la fase 2 (`axiom change start --adopt --change <cambio>`) después de `seal` y antes de crear `proposal.md` y de lanzar `sdd-explore`, que recibe `context/work-item.md` como contexto. Un rechazo de la fase 1 MUST detener el arranque sin crear rama, worktree, carpeta ni `kickoff.yaml`, y el orquestador MUST relayar su mensaje y su continuación a la persona. Si la persona abandona el flujo tras la fase 1, el orquestador MUST ofrecer `axiom change start --discard --change <cambio>`. Si `axiom branch ensure` falla tras una fase 1 correcta (un resultado `decision_required` que se resuelve con su procedimiento propio, o un error) o si `axiom sdd kickoff seal` falla, el orquestador MUST conservar el staging, MUST NOT ejecutar la fase 2, MUST relayar el mensaje a la persona y MUST ofrecer reintentar con el mismo nombre derivado (sin repetir la fase 1) o abandonar con `axiom change start --discard --change <cambio>`. Un rechazo de `axiom change start --adopt` (`staging_not_found`, `staging_conflict`, `change_not_prepared`, `change_start_busy` o `write_failed`) MUST relayarse con su mensaje y su continuación, MUST detener el arranque antes de crear `proposal.md` y de lanzar `sdd-explore`, y el orquestador MUST NOT ejecutar la continuación por su cuenta si implica una decisión de la persona (como `--discard`); un aviso de la fase 2 (por ejemplo, el enlace de vuelta pendiente) no detiene el arranque y MUST presentarse con su continuación de reintento. Este orden MUST aplicarse también a un proyecto ISO y al perfil de bug. Sin plugin activo, el flujo de RF-KOF-05 MUST ser el actual, sin ninguna llamada a `axiom change start` más allá de la consulta que responde «no aplica».

#### Scenario: Plugin activo y sin WI
- **DADO** un proyecto con el plugin activo y una petición de cambio SDD sin WI
- **CUANDO** el orquestador va a nombrar el cambio
- **ENTONCES** pide el identificador o la URL del WI y no ejecuta `ensure` ni `seal`

#### Scenario: Orden de las fases
- **DADO** un WI válido y el plugin activo
- **CUANDO** el orquestador arranca el cambio
- **ENTONCES** la fase 1 se ejecuta antes de `axiom branch ensure`, `ensure` antes de `seal` y la fase 2 después de `seal`
- **Y** `ensure` y `seal` usan el nombre devuelto por la fase 1

#### Scenario: Rechazo de la fase 1 detiene el arranque
- **DADO** un WI cerrado
- **CUANDO** la fase 1 se rechaza con `work_item_closed`
- **ENTONCES** no se ejecuta `ensure` ni `seal`, no se crea rama, worktree ni carpeta del cambio
- **Y** se presenta a la persona el mensaje y la continuación del rechazo

#### Scenario: Abandono tras la fase 1
- **DADO** una fase 1 correcta y una persona que decide no continuar
- **CUANDO** el orquestador cierra el arranque
- **ENTONCES** ofrece `axiom change start --discard --change <cambio>`

#### Scenario: Falla `ensure` o `seal` tras la fase 1
- **DADO** una fase 1 correcta y un `axiom branch ensure` (o un `axiom sdd kickoff seal`) que falla
- **CUANDO** el orquestador gestiona el fallo
- **ENTONCES** no ejecuta la fase 2, conserva el staging y ofrece reintentar con el mismo nombre derivado o `axiom change start --discard --change <cambio>`

#### Scenario: Rechazo de `--adopt`
- **DADO** una fase 2 que se rechaza con `change_not_prepared`
- **CUANDO** el orquestador la recibe
- **ENTONCES** relaya el mensaje y la continuación `axiom branch ensure --change <cambio> --json`, no crea `proposal.md` y no lanza `sdd-explore`

#### Scenario: Aviso de enlace de vuelta pendiente
- **DADO** una fase 2 que termina con código 0 y el aviso de enlace de vuelta pendiente
- **CUANDO** el orquestador la recibe
- **ENTONCES** presenta el aviso con su reintento y continúa con `proposal.md` y `sdd-explore`

#### Scenario: El explore recibe el contexto
- **DADO** una fase 2 completada
- **CUANDO** el orquestador lanza `sdd-explore`
- **ENTONCES** le indica `context/work-item.md` como dato no fiable de un tercero

#### Scenario: Proyecto ISO y bug
- **DADO** un proyecto ISO con el plugin activo y un WI de tipo Bug
- **CUANDO** el orquestador arranca el cambio
- **ENTONCES** sigue el mismo orden y el perfil resultante es el de bug

#### Scenario: Sin plugin activo no cambia nada
- **DADO** un proyecto sin plugin activo
- **CUANDO** el orquestador arranca el cambio
- **ENTONCES** el flujo es el de RF-KOF-05 y no ejecuta ninguna fase de `axiom change start`
### Requirement: Formato de `pruebas-estado.yaml` de Solo Añadir y Caducidad por `plan_digest` (RF-KOF-25)

El estado de ejecución de la batería DEBE vivir en `pruebas-estado.yaml`, en la carpeta del cambio y separado de `pruebas.md`. Cada entrada DEBE tener `case` (`P-NN`), `environment`, `result` (`ok|ko|blocked|n/a`), `by` (identidad git `user.name` y `user.email`), `at` (fecha y hora), `commit` (opcional, vacío por defecto), `plan_digest` y `note` (acotada). El fichero es de solo añadir: ninguna operación del producto DEBE editar ni borrar entradas, y una corrección es una entrada nueva; para cada par caso-entorno cuenta como vigente la última entrada que no esté caducada. `plan_digest` DEBE ser `kickoff.ArtifactDigest` de `pruebas.md` (el mismo digest que la compuerta `tasks`, con finales de línea CRLF normalizados). Una entrada cuyo `plan_digest` difiere del actual DEBE considerarse caducada: se cuenta aparte y NUNCA como vigente. Un fichero ausente equivale a ninguna marca; un fichero ilegible o inválido DEBE reportarse como hallazgo informativo, sin bloquear nada. Este fichero NO DEBE formar parte de ningún digest de compuerta ni de `kickoff.yaml`.

#### Scenario: Entrada con todos los campos
- **DADO** una marca `P-01`, `pre`, `ok`
- **CUANDO** se escribe en `pruebas-estado.yaml`
- **ENTONCES** la entrada contiene `case`, `environment`, `result`, `by`, `at`, `commit` vacío, `plan_digest` y `note`

#### Scenario: Caducidad al cambiar `pruebas.md`
- **DADO** una marca con el `plan_digest` actual
- **CUANDO** se modifica `pruebas.md`
- **ENTONCES** la marca se considera caducada y no cuenta como vigente

#### Scenario: Finales de línea no caducan
- **DADO** una marca y un `pruebas.md` cuyos finales de línea pasan de LF a CRLF sin otro cambio
- **CUANDO** se recalcula el digest
- **ENTONCES** el `plan_digest` coincide y la marca sigue vigente

#### Scenario: Gana la última entrada vigente
- **DADO** una entrada `ko` y otra posterior `ok` para el mismo caso y entorno, ambas vigentes
- **CUANDO** se calcula el estado
- **ENTONCES** el par caso-entorno cuenta como `ok`

#### Scenario: Fichero ausente
- **DADO** un cambio sin `pruebas-estado.yaml`
- **CUANDO** se calcula el estado
- **ENTONCES** no hay marcas y no hay error

#### Scenario: Fichero inválido
- **DADO** un `pruebas-estado.yaml` con YAML inválido
- **CUANDO** se calcula el estado
- **ENTONCES** se reporta un hallazgo informativo
- **Y** `nextRecommended`, las dependencias y los digests no cambian

### Requirement: La Instantánea Congela la Plantilla, el Tipo y la Referencia de Work Item (RF-KOF-26)

Además de la plantilla de cada rama (RF-KOF-06), `axiom sdd kickoff seal` DEBE congelar en la instantánea de política git, en campos aditivos y opcionales, los valores de work item conocidos en ese momento que alimentan la plantilla: `kind` (`feature` o `fix`, según el perfil de `change start`) y `ref` (el `id` del work item, por ejemplo `2525`). Se leen del staging de `change start`, que sigue presente al sellar porque la fase de adopción (`--adopt`) va después del seal. Cuando la instantánea guarda alguna plantilla, `kind` SIEMPRE DEBE congelarse (`feature` si no hay perfil bug, `fix` si lo hay), aunque el cambio no tenga work item; `ref` solo se congela si se conoce, y un `ref` desconocido NO DEBE escribirse. Sin ninguna plantilla en la instantánea no se escribe ninguno de los dos. Si el render de las plantillas al sellar se rechaza (variable sin valor, nombre inválido, rama compartida con `{role}` o `{task_ref}`), el rechazo se propaga y `kickoff.yaml` no se escribe. `{task_ref}` NUNCA DEBE congelarse en el kickoff: las Tasks se crean tras aprobar el design, después del sellado. Con `kind` y `ref` congelados, todos los clones del cambio DEBEN resolver el mismo nombre de rama, aunque el staging, que es local y no versionado, no exista en ellos.

Un cambio ya sellado con plantilla pero sin esos campos (sin `template_values`) DEBE conservar el comportamiento actual: `{kind}` vale `feature` (sin consultar el perfil del staging), `{ref}` se lee del vínculo `work_item` o del staging si existe y, si no existe y la plantilla lo exige, se rechaza con el código tipado de variable sin valor (RF-BRA-45 de `change-branch-lifecycle`). Un cambio ya sellado sin plantilla en la instantánea NO DEBE migrarse: conserva sus nombres actuales aunque `axiom.yaml` declare después una plantilla, y NINGUNA orden reescribe su `kickoff.yaml`. Un binario anterior DEBE ignorar los campos nuevos sin error.

#### Scenario: El sellado congela `kind` y `ref` del staging
- **DADO** el staging de `change start` con perfil bug e `id` `2525`, y `git.branch_template: "{kind}/{ref}-{slug}"`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change us-2525-alta-de-usuarios ...`
- **ENTONCES** la instantánea contiene `kind: fix` y `ref: "2525"`
- **Y** el nombre de la rama del cambio es `fix/2525-alta-de-usuarios`

#### Scenario: Otro clon resuelve el mismo nombre sin staging
- **DADO** el cambio anterior sellado y un segundo clon sin staging ni registro, con el vínculo `work_item` ya versionado o no
- **CUANDO** se ejecuta `ensure --change us-2525-alta-de-usuarios` en el segundo clon
- **ENTONCES** la rama se nombra `fix/2525-alta-de-usuarios`, con `kind` y `ref` leídos de la instantánea

#### Scenario: Sin work item se congela `kind` pero no `ref`
- **DADO** un cambio sin work item, con `git.branch_template: "{kind}/{change}"`
- **CUANDO** se sella
- **ENTONCES** la instantánea contiene `kind: feature` y no contiene `ref`

#### Scenario: Sin plantilla no se escriben `kind` ni `ref`
- **DADO** un `axiom.yaml` sin `branch_template` y un cambio con work item
- **CUANDO** se sella
- **ENTONCES** la instantánea no contiene `kind` ni `ref`

#### Scenario: `{task_ref}` no se congela en el kickoff
- **DADO** un multirepo con `repositories[].branch_template: "feature/TASK-{task_ref}"`
- **CUANDO** se sella el cambio
- **ENTONCES** la instantánea no contiene ningún valor de `task_ref`

#### Scenario: Un cambio sellado con plantilla pero sin `kind` ni `ref` conserva el comportamiento actual
- **DADO** un `kickoff.yaml` con plantilla en la instantánea y sin `kind` ni `ref`, con el vínculo `work_item` `2525` en `integrations.yaml`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** `{kind}` vale `feature` y `{ref}` vale `2525`
- **Y** `kickoff.yaml` no se modifica

#### Scenario: Un cambio sellado con plantilla y sin `kind` ignora el perfil bug del staging
- **DADO** un `kickoff.yaml` con la plantilla `{kind}/{change}` y sin `kind`, y un staging con perfil bug
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** la rama del cambio es `feature/x`

#### Scenario: Un cambio sellado sin plantilla no se migra
- **DADO** un cambio sellado antes de esta capacidad, sin plantilla en la instantánea, con las ramas `feat/x` y `feat/x-archive`, y un `axiom.yaml` que declara después `git.branch_template: "{kind}/{change}"`
- **CUANDO** se ejecuta `ensure --change x` y `ensure --change x --archive`
- **ENTONCES** las ramas siguen siendo `feat/x` y `feat/x-archive`
- **Y** `kickoff.yaml` no se modifica

#### Scenario: Un binario anterior ignora los campos nuevos
- **DADO** un `kickoff.yaml` con plantilla, `kind` y `ref` en la instantánea
- **CUANDO** lo lee un binario anterior a esta capacidad
- **ENTONCES** carga el resto de la configuración sellada sin error

