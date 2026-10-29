<!-- Especificación Viva generada a partir de '2026-09-22-inc-21-upfront-flow-governance' -->

# Especificación de Requerimientos: Selección Temprana y Bloqueante de Carril ODD/SDD en el Kickoff (INC-21)

## Propósito

Definir de forma rigurosa, ejecutable y verificable los requerimientos funcionales y escenarios BDD para la compuerta temprana y bloqueante que determina si una solicitud de trabajo nueva se aborda mediante el carril ágil (ODD — Organic Driven Development) o el carril formal (SDD — Spec-Driven Development), sellando esa decisión para todo el ciclo de vida del cambio.

---

## 1. Capacidad: `sdd-lane-kickoff`

El orquestador determina el carril de ejecución (ODD o SDD) en el primer contacto con cualquier solicitud de trabajo, antes de crear ningún artefacto, y sella esa decisión para el resto del ciclo de vida del cambio.

### Requirement: Evaluación de Alcance y Pregunta Bloqueante de Selección de Carril (REQ-21.1)

El orquestador DEBE evaluar el alcance de toda solicitud de trabajo nueva antes de crear cualquier artefacto de trabajo (documento vivo ODD o propuesta SDD). Si el alcance detectado es acotado y admite razonablemente ambos carriles, el orquestador DEBE formular explícitamente la pregunta bloqueante de selección de carril ("¿Deseas abordarla mediante vía ágil ODD o vía formal SDD?") y DEBE detenerse a esperar la respuesta del usuario antes de continuar. El orquestador NO DEBE crear `odd/tasks/<feature>.md` ni `openspec/changes/<cambio>/proposal.md` mientras la pregunta esté pendiente de respuesta.

#### Scenario: Alcance acotado formula la pregunta y bloquea hasta obtener respuesta

- **DADO** una solicitud de trabajo de alcance acotado que razonablemente admite tanto ODD como SDD
- **CUANDO** el orquestador recibe la solicitud
- **ENTONCES** formula la pregunta de selección de carril y se detiene a esperar respuesta
- **Y** no crea ningún artefacto de trabajo antes de recibir la respuesta

#### Scenario: Alcance inequívocamente arquitectónico evita la pregunta binaria

- **DADO** una solicitud cuyo alcance es inequívocamente de arquitectura o de gran envergadura
- **CUANDO** el orquestador evalúa el alcance
- **ENTONCES** entra directamente al cuestionario de pre-vuelo de SDD (capacidad `sdd-preflight-configuration`) sin necesidad de formular la pregunta binaria ODD/SDD

---

### Requirement: Carril ODD sin Fricción Adicional (REQ-21.2)

Si el usuario elige el carril ODD, el orquestador NO DEBE formular ninguna pregunta adicional de gobernanza, DEBE ejecutar el trabajo "del tirón" sin dividirlo en roles ni generar ningún `handoff.md`, y DEBE mantener el trabajo exclusivamente en `odd/tasks/<feature>.md`.

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

---

### Requirement: Entrada al Cuestionario de Pre-Vuelo de SDD (REQ-21.3)

Si el usuario elige el carril SDD en respuesta a la pregunta de REQ-21.1, o si el alcance detectado ya es inequívocamente de arquitectura o gran alcance (REQ-21.1, segundo escenario), el orquestador DEBE entrar al cuestionario de pre-vuelo de SDD descrito en la capacidad `sdd-preflight-configuration` antes de crear `proposal.md`.

#### Scenario: Entrada al pre-vuelo tras elección explícita o alcance ya determinado

- **DADO** que el usuario elige el carril SDD, o que el alcance ya se determinó inequívocamente como de arquitectura
- **CUANDO** el orquestador continúa el flujo
- **ENTONCES** entra al cuestionario de pre-vuelo de SDD antes de crear `proposal.md`

---

### Requirement: Idempotencia del Kickoff ante un Cambio con Artefactos o Configuración Ya Sellada (REQ-21.4)

El orquestador NO DEBE volver a formular la pregunta de selección de carril (REQ-21.1) ni el cuestionario de pre-vuelo de SDD (capacidad `sdd-preflight-configuration`) para un cambio que ya tiene al menos un artefacto persistido (`proposal.md`, un `spec.md` bajo `specs/`, `design.md` o `tasks.md`) o cuya configuración de kickoff ya está sellada. En ese caso, el orquestador DEBE continuar respetando el carril y la configuración ya selladas en el primer kickoff, sin interrogar de nuevo al usuario.

(Previamente: esta especificación indicaba que, al retro-sellar un cambio preexistente sin kickoff sellado, el rol por defecto congelado era el rol único `fullstack`. Se corrige tras la pregunta abierta O-1 planteada en `sdd-design` y resuelta explícitamente por el usuario con evidencia de código —`internal/multirole/detector.go:35-55` cae siempre a `core`, nunca a `fullstack`; `internal/multirole/detector.go:58-64` rechaza cualquier rol ausente de `axiom.yaml`; y `axiom.yaml:7-28` no declara `fullstack`—: el retro-sello DEBE congelar el rol que la detección de roles ya vigente infiere para ese cambio, nunca introducir la identidad `fullstack`.)

#### Scenario: Reanudación de un cambio con kickoff ya sellado

- **DADO** un cambio SDD activo con `proposal.md` existente y su configuración de kickoff ya sellada (carril `sdd`, modalidad y política de relevos definidas)
- **CUANDO** se reanuda el trabajo sobre ese cambio en una sesión posterior
- **ENTONCES** el orquestador no repite la pregunta de selección de carril ni el cuestionario de pre-vuelo
- **Y** continúa el flujo respetando la modalidad y la política de relevos ya selladas

#### Scenario: Cambio preexistente con artefactos pero sin kickoff sellado infiere un valor por defecto sin bloquear

- **DADO** un cambio SDD activo con artefactos existentes (por ejemplo `proposal.md` y `design.md`) pero sin ninguna configuración de kickoff sellada previamente
- **CUANDO** el orquestador retoma ese cambio
- **ENTONCES** infiere y sella la configuración por defecto documentada (modalidad continua, sin `handoff.md` intermedios, y el rol que la detección de roles ya vigente infiere para ese cambio —hoy, `core`, cuando existe un `design.md` sin roles explícitos—) sin bloquear el trabajo ni interrogar retroactivamente al usuario
- **Y** el retro-sello no introduce la identidad `fullstack` ni ningún rol nuevo: es conservador por definición y se limita a congelar el rol y el comportamiento que la detección de roles ya aplica hoy para ese cambio, sin redefinirlo
- **Y** dicha configuración inferida queda registrada para no volver a inferirse en la siguiente reanudación

#### Scenario: El retro-sello no altera qué `tasks.<rol>.md` consulta la barrera

- **DADO** un cambio SDD preexistente sin kickoff sellado, cuya barrera de sincronización ya consulta `tasks.core.md` porque la detección de roles vigente infiere el rol `core` para su `design.md`
- **CUANDO** el orquestador retro-sella la configuración de kickoff de ese cambio conforme al escenario anterior
- **ENTONCES** la barrera de sincronización sigue consultando exactamente `tasks.core.md`, el mismo fichero que consultaba antes del sellado
- **Y** el retro-sello no crea, renombra ni redirige ningún fichero `tasks.<rol>.md`, ni introduce `tasks.fullstack.md` como fuente de verdad nueva para ese cambio
### Requirement: `ensure` al Nombrar el Cambio, antes de Escribir en su Carpeta (REQ-25.35)

Una vez elegido el carril (REQ-21.1) y nombrado el cambio (SDD) o la feature (ODD), el orquestador DEBE ejecutar `axiom branch ensure` para ese nombre (con `--lane odd` en ODD) **antes de crear cualquier artefacto de trabajo** (`proposal.md`, `odd/tasks/<feature>.md`) y, en SDD, antes del cuestionario de pre-vuelo y de `axiom sdd kickoff seal`. A partir de ese momento DEBE usar como `--cwd` de las órdenes posteriores las rutas absolutas que devuelve el JSON de `ensure` (REQ-25.2 de `change-branch-lifecycle`) y NO el directorio de trabajo heredado. La ejecución de `ensure` NO es la creación de un artefacto de trabajo, por lo que no contraviene REQ-21.1. Si `ensure` devuelve `decision_required` (modo `ask`), el orquestador DEBE presentar la elección al usuario y repetir la orden con la respuesta; esa pregunta es una preferencia de entorno personal y NO una pregunta de gobernanza, por lo que NO contraviene REQ-21.2 (supuesto S-10). Al reanudar un cambio ya sellado, `ensure` DEBE poder repetirse de forma idempotente sin repetir la selección de carril ni el pre-vuelo (REQ-21.4) y sin volver a preguntar el modo ya respondido.

#### Scenario: SDD ejecuta `ensure` antes del sellado y de `proposal.md`
- **DADO** que el usuario eligió el carril SDD y se nombra el cambio `x`
- **CUANDO** el orquestador inicia el flujo
- **ENTONCES** ejecuta `axiom branch ensure --change x` antes del cuestionario de pre-vuelo, de `kickoff seal` y de crear `proposal.md`
- **Y** las órdenes posteriores usan el `--cwd` absoluto del JSON de `ensure`

#### Scenario: ODD ejecuta `ensure --lane odd` antes de escribir `odd/tasks/`
- **DADO** que el usuario eligió el carril ODD para la feature `mi-feature`
- **CUANDO** el orquestador va a crear `odd/tasks/mi-feature.md`
- **ENTONCES** ejecuta antes `axiom branch ensure --lane odd --change mi-feature`
- **Y** no formula ninguna pregunta adicional de gobernanza

#### Scenario: `ensure` no se ejecuta mientras la pregunta de carril está sin responder
- **DADO** una solicitud de alcance acotado y la pregunta de selección de carril sin respuesta
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

---

### Requirement: Instantánea de Política Git en `kickoff.yaml`, Escrita una Sola Vez por `seal` (REQ-25.36)

`axiom sdd kickoff seal` DEBE escribir en `kickoff.yaml`, junto al resto de la configuración sellada y en la misma única escritura, la instantánea de política git del cambio: la rama de integración y el prefijo resueltos, la base (nombre de rama, no SHA) de cada repositorio y los nombres de rama derivados del cambio (rama del cambio, por repositorio con dos o más roles las ramas de rol y, en los repositorios con rama de cambio, la rama de archivado `<prefijo>/<cambio>-archive`), calculados con la política de `axiom.yaml`, las excepciones por cambio de `ensure` (`--prefix`, `--branch`, `--base`) y los roles que se sellan. La instantánea contiene entradas de política, nunca resultados: NO DEBE incluir SHAs, rutas absolutas ni rutas de worktree, que son personales o cambian. La instantánea se escribe una sola vez: conserva la semántica de sellado único, en la que gana la primera publicación, y NINGUNA otra orden, incluida `ensure`, DEBE modificarla. Un segundo `seal` desde la rama del cambio NO DEBE reescribir `kickoff.yaml`: DEBE terminar con éxito, devolver el kickoff ya sellado e indicar que no escribió (texto «Kickoff ya estaba sellado»; `wrote: false` en `--json`). Un segundo `seal` desde otra rama DEBE rechazarse con `wrong_branch` (veredicto de REQ-25.18 de `change-branch-lifecycle`) sin modificar nada. La lectura de `kickoff.yaml` DEBE tolerar su ausencia y un binario anterior DEBE ignorarla sin error.
(Previously: decía que «un segundo `seal` se rechaza como hoy», pero el sellado único nunca ha rechazado un segundo `seal` desde la rama del cambio: devuelve el ganador sin reescribir, como fijan `internal/kickoff/seal.go` y sus pruebas)

#### Scenario: El sellado escribe la instantánea
- **DADO** un monorepo con `git.integration_branch: main`, `git.branch_prefix: feat` y los roles `core` y `qa` sobre el mismo repositorio
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...` desde la rama del cambio
- **ENTONCES** `kickoff.yaml` contiene la integración `main`, el prefijo `feat`, la base `main` del repositorio y los nombres `feat/x`, `feat/x-core`, `feat/x-qa` y la rama de archivado `feat/x-archive`
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

---

### Requirement: Exención de Cambios Preexistentes sin Instantánea y de `seal --infer` (REQ-25.37)

Las guardas de rama (veredicto de REQ-25.18 de `change-branch-lifecycle`) NO DEBEN bloquear un cambio cuyo `kickoff.yaml` existe y no contiene instantánea (cambio preexistente sellado antes de esta capacidad). El retro-sellado `axiom sdd kickoff seal --infer` (REQ-21.4) NO DEBE exigir una rama determinada y NO DEBE escribir instantánea, de modo que un cambio retro-sellado permanezca exento. El producto NO añade ninguna otra exención para cambios preexistentes: solo quedan exentos por nombre los sumideros acumulativos `e2e-cumulative` y `cumulative-<rol>` (REQ-25.18).

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

### Requirement: `kickoff seal` se Niega Fuera de la Rama que Corresponde (REQ-25.38)

`axiom sdd kickoff seal` DEBE consultar el veredicto de rama (REQ-25.18 de `change-branch-lifecycle`) para el repositorio del planning home antes de escribir y, ante `wrong_branch`, DEBE negarse sin escribir `kickoff.yaml`, con código de salida distinto de cero, el código tipado `wrong_branch` y la continuación exacta (`axiom branch ensure` con `--cwd`, `--change` y, si procede, `--lane`). En `monorepo-embedded` exige estar en la rama del cambio (o en una rama de rol del cambio). En `multirepo` exige que el repositorio de specs esté en su rama de integración y no en una rama de rol; los repositorios de código no se comprueban en el sellado. Si el cambio aún no tiene `kickoff.yaml`, los nombres esperados se obtienen de la política de `axiom.yaml`.

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
