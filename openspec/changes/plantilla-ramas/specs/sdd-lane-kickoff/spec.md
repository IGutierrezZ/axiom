# Delta for sdd-lane-kickoff

## ADDED Requirements

### Requirement: La Instantánea Congela la Plantilla, el Tipo y la Referencia de Work Item (prov:KOF:1)

Además de la plantilla de cada rama (RF-KOF-06), `axiom sdd kickoff seal` DEBE congelar en la instantánea de política git, en campos aditivos y opcionales, los valores de work item conocidos en ese momento que alimentan la plantilla: `kind` (`feature` o `fix`, según el perfil de `change start`) y `ref` (el `id` del work item, por ejemplo `2525`). Se leen del staging de `change start`, que sigue presente al sellar porque la fase de adopción (`--adopt`) va después del seal. Un campo cuyo valor no se conoce al sellar (cambio sin work item) NO DEBE escribirse. `{task_ref}` NUNCA DEBE congelarse en el kickoff: las Tasks se crean tras aprobar el design, después del sellado. Con `kind` y `ref` congelados, todos los clones del cambio DEBEN resolver el mismo nombre de rama, aunque el staging, que es local y no versionado, no exista en ellos.

Un cambio ya sellado con plantilla pero sin esos campos DEBE conservar el comportamiento actual: `{kind}` vale `feature`, `{ref}` se lee del vínculo `work_item` o del staging si existe y, si no existe y la plantilla lo exige, se rechaza con el código tipado de variable sin valor (prov:BRA:4 de `change-branch-lifecycle`). Un cambio ya sellado sin plantilla en la instantánea NO DEBE migrarse: conserva sus nombres actuales aunque `axiom.yaml` declare después una plantilla, y NINGUNA orden reescribe su `kickoff.yaml`. Un binario anterior DEBE ignorar los campos nuevos sin error.

#### Scenario: El sellado congela `kind` y `ref` del staging
- **DADO** el staging de `change start` con perfil bug e `id` `2525`, y `git.branch_template: "{kind}/{ref}-{slug}"`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change us-2525-alta-de-usuarios ...`
- **ENTONCES** la instantánea contiene `kind: fix` y `ref: "2525"`
- **Y** el nombre de la rama del cambio es `fix/2525-alta-de-usuarios`

#### Scenario: Otro clon resuelve el mismo nombre sin staging
- **DADO** el cambio anterior sellado y un segundo clon sin staging ni registro, con el vínculo `work_item` ya versionado o no
- **CUANDO** se ejecuta `ensure --change us-2525-alta-de-usuarios` en el segundo clon
- **ENTONCES** la rama se nombra `fix/2525-alta-de-usuarios`, con `kind` y `ref` leídos de la instantánea

#### Scenario: Sin work item no se escriben `kind` ni `ref`
- **DADO** un cambio sin work item y sin plantilla con `{ref}`
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

#### Scenario: Un cambio sellado sin plantilla no se migra
- **DADO** un cambio sellado antes de esta capacidad, sin plantilla en la instantánea, con las ramas `feat/x` y `feat/x-archive`, y un `axiom.yaml` que declara después `git.branch_template: "{kind}/{change}"`
- **CUANDO** se ejecuta `ensure --change x` y `ensure --change x --archive`
- **ENTONCES** las ramas siguen siendo `feat/x` y `feat/x-archive`
- **Y** `kickoff.yaml` no se modifica

#### Scenario: Un binario anterior ignora los campos nuevos
- **DADO** un `kickoff.yaml` con plantilla, `kind` y `ref` en la instantánea
- **CUANDO** lo lee un binario anterior a esta capacidad
- **ENTONCES** carga el resto de la configuración sellada sin error

## MODIFIED Requirements

### Requirement: Instantánea de Política Git en `kickoff.yaml`, Escrita una Sola Vez por `seal` (RF-KOF-06)

`axiom sdd kickoff seal` DEBE escribir en `kickoff.yaml`, junto al resto de la configuración sellada y en la misma única escritura, la instantánea de política git del cambio: la rama de integración y el prefijo resueltos, la base (nombre de rama, no SHA) de cada repositorio y los nombres de rama derivados del cambio (rama del cambio, por repositorio con dos o más roles las ramas de rol y, en los repositorios con rama de cambio, la rama de archivado `<prefijo>/<cambio>-archive`), calculados con la política de `axiom.yaml`, las excepciones por cambio de `ensure` (`--prefix`, `--branch`, `--base`) y los roles que se sellan. Cuando la plantilla efectiva de una rama de código procede de `axiom.yaml` (prov:BRA:3 de `change-branch-lifecycle`), la instantánea DEBE guardar además, en un campo aditivo y opcional de esa rama, la plantilla (la política, no el resultado); si la plantilla usa `{task_ref}`, la instantánea NO DEBE escribir el nombre de esa rama, que queda pendiente hasta la entrada del rol, y la rama que lleva `openspec/` y las ramas cuya plantilla no usa `{task_ref}` siguen llevando su nombre resuelto. Sin plantilla, la instantánea es idéntica a la de antes. Los valores `kind` y `ref` de la plantilla se congelan según prov:KOF:1. Siempre que haya dos etapas al sellar (`git.main_branch` declarada y distinta de la rama de integración de algún repositorio del plan, RF-BRA-28 de `change-branch-lifecycle`), la instantánea DEBE guardar además el campo opcional `main_branch` (la rama principal declarada al sellar), que actúa de marcador: distingue un cambio sellado con dos etapas de uno sellado antes de existir esta capacidad. Además, la instantánea DEBE copiar el modo de promoción (`branch` o `release`) de un repositorio solo cuando ese modo ya estaba resuelto en el registro local al sellar (la rama del cambio se creó antes del sellado, como en el monorepo SDD), y la base de ese repositorio DEBE ser su base efectiva (la rama principal en `branch`, la de integración en `release`). El modo es un campo opcional de cada repositorio: NO DEBE escribirse en los repositorios de una sola etapa ni en los repositorios cuyo modo no estaba resuelto en el registro al sellar (como los repositorios de código de un `multirepo`, cuyas ramas y respuestas nacen en un `ensure` posterior al sellado); su ausencia, con `main_branch` presente en la instantánea, significa que el modo se resuelve al crear la rama, nunca que el repositorio sea de una sola etapa. Sin dos etapas al sellar, la instantánea no contiene `main_branch`. La instantánea contiene entradas de política, nunca resultados: NO DEBE incluir SHAs, rutas absolutas ni rutas de worktree, que son personales o cambian. La instantánea se escribe una sola vez: conserva la semántica de sellado único, en la que gana la primera publicación, y NINGUNA otra orden, incluida `ensure`, DEBE modificarla. Un segundo `seal` desde la rama del cambio NO DEBE reescribir `kickoff.yaml`: DEBE terminar con éxito, devolver el kickoff ya sellado e indicar que no escribió (texto «Kickoff ya estaba sellado»; `wrote: false` en `--json`). Un segundo `seal` desde otra rama DEBE rechazarse con `wrong_branch` (veredicto de RF-BRA-19 de `change-branch-lifecycle`) sin modificar nada. La lectura de `kickoff.yaml` DEBE tolerar su ausencia y un binario anterior DEBE ignorarla sin error, incluidos los campos opcionales `main_branch`, el modo de promoción, la plantilla de rama, `kind` y `ref`.
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

#### Scenario: Cambiar la plantilla después del sellado no modifica la instantánea
- **DADO** un cambio sellado con la plantilla `feature/TASK-{task_ref}` en la instantánea
- **CUANDO** se cambia la plantilla en `axiom.yaml` y se ejecuta `ensure --change x --role frontend`
- **ENTONCES** la rama se nombra con la plantilla de la instantánea
- **Y** `kickoff.yaml` no se modifica
