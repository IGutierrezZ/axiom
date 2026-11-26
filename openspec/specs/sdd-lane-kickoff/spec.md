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

`axiom sdd kickoff seal` DEBE escribir en `kickoff.yaml`, junto al resto de la configuración sellada y en la misma única escritura, la instantánea de política git del cambio: la rama de integración y el prefijo resueltos, la base (nombre de rama, no SHA) de cada repositorio y los nombres de rama derivados del cambio (rama del cambio, por repositorio con dos o más roles las ramas de rol y, en los repositorios con rama de cambio, la rama de archivado `<prefijo>/<cambio>-archive`), calculados con la política de `axiom.yaml`, las excepciones por cambio de `ensure` (`--prefix`, `--branch`, `--base`) y los roles que se sellan. Siempre que haya dos etapas al sellar (`git.main_branch` declarada y distinta de la rama de integración de algún repositorio del plan, RF-BRA-28 de `change-branch-lifecycle`), la instantánea DEBE guardar además el campo opcional `main_branch` (la rama principal declarada al sellar), que actúa de marcador: distingue un cambio sellado con dos etapas de uno sellado antes de existir esta capacidad. Además, la instantánea DEBE copiar el modo de promoción (`branch` o `release`) de un repositorio solo cuando ese modo ya estaba resuelto en el registro local al sellar (la rama del cambio se creó antes del sellado, como en el monorepo SDD), y la base de ese repositorio DEBE ser su base efectiva (la rama principal en `branch`, la de integración en `release`). El modo es un campo opcional de cada repositorio: NO DEBE escribirse en los repositorios de una sola etapa ni en los repositorios cuyo modo no estaba resuelto en el registro al sellar (como los repositorios de código de un `multirepo`, cuyas ramas y respuestas nacen en un `ensure` posterior al sellado); su ausencia, con `main_branch` presente en la instantánea, significa que el modo se resuelve al crear la rama, nunca que el repositorio sea de una sola etapa. Sin dos etapas al sellar, la instantánea no contiene `main_branch`. La instantánea contiene entradas de política, nunca resultados: NO DEBE incluir SHAs, rutas absolutas ni rutas de worktree, que son personales o cambian. La instantánea se escribe una sola vez: conserva la semántica de sellado único, en la que gana la primera publicación, y NINGUNA otra orden, incluida `ensure`, DEBE modificarla. Un segundo `seal` desde la rama del cambio NO DEBE reescribir `kickoff.yaml`: DEBE terminar con éxito, devolver el kickoff ya sellado e indicar que no escribió (texto «Kickoff ya estaba sellado»; `wrote: false` en `--json`). Un segundo `seal` desde otra rama DEBE rechazarse con `wrong_branch` (veredicto de RF-BRA-19 de `change-branch-lifecycle`) sin modificar nada. La lectura de `kickoff.yaml` DEBE tolerar su ausencia y un binario anterior DEBE ignorarla sin error, incluidos los campos opcionales `main_branch` y el modo de promoción.
(Previously: la instantánea no incluía `main_branch` ni el modo de promoción y la base de cada repositorio era siempre la rama de integración)

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
