# Delta for change-branch-lifecycle

## ADDED Requirements

### Requirement: Plantilla de Nombre de Rama: Motor y Variables (prov:BRA:1)

El sistema DEBE poder renderizar el nombre de una rama a partir de una plantilla de texto (`branch_template`) que contiene variables entre llaves. El renderizado DEBE ser una función pura, sin lectura de ficheros, de git ni de la red: recibe la plantilla y los valores ya resueltos, y devuelve el nombre o un rechazo tipado. Las variables admitidas son exactamente estas siete, y cualquier otra DEBE rechazarse:

| Variable | Valor |
|---|---|
| `{prefix}` | El `branch_prefix` efectivo, incluido el que fija `--prefix` en la invocación. |
| `{kind}` | `fix` cuando se conoce que el cambio tiene perfil bug; si no, `feature`. |
| `{change}` | El nombre del cambio. |
| `{slug}` | El nombre del cambio sin el prefijo del ID de work item (`us-2525-alta-de-usuarios` produce `alta-de-usuarios`). Se quita el prefijo `^<letras>-<ref>-` (letras ASCII, el `{ref}` conocido y un guion), sin distinguir mayúsculas, y solo cuando `{ref}` es conocido. Es igual a `{change}` si no hay `{ref}` conocido, si el nombre no casa con ese prefijo o si al quitarlo quedaría vacío (`us-2525`): `{slug}` nunca es vacío. |
| `{role}` | El identificador del rol al que se atribuye la rama. |
| `{ref}` | El `id` del vínculo `kind: work_item` (por ejemplo `2525`), nunca la referencia externa completa (`ado:2525`). |
| `{task_ref}` | El `id` del vínculo `kind: task` del rol al que se atribuye la rama. |

Cada valor sustituido, salvo `{prefix}`, DEBE ser un componente de rama válido para git (letras ASCII, dígitos, `.`, `_` y `-`); `{prefix}` PUEDE contener `/` y sigue las reglas del prefijo (RF-BRA-03). Un valor que no sea seguro para git NO DEBE sanearse ni sustituirse en silencio: la resolución se rechaza con un error tipado que nombra la variable y el valor (RF-BRA-03 de esta capacidad, prov:BRA:4). El nombre resultante DEBE ser un nombre de referencia git válido.

#### Scenario: Plantilla con prefijo, referencia y slug
- **DADO** la plantilla `{prefix}/{ref}-{slug}`, el prefijo `inc`, el cambio `us-2525-alta-de-usuarios` y el vínculo `work_item` con `id` `2525`
- **CUANDO** se renderiza el nombre de la rama del cambio
- **ENTONCES** el nombre es `inc/2525-alta-de-usuarios`

#### Scenario: `{ref}` es el id del vínculo y no la referencia externa
- **DADO** un cambio con la referencia externa `ado:2525` y el vínculo `work_item` con `id` `2525`
- **CUANDO** se renderiza `{prefix}/{ref}`
- **ENTONCES** el nombre contiene `2525` y no contiene `ado:`

#### Scenario: `{kind}` vale `feature` por defecto y `fix` con perfil bug
- **DADO** la plantilla `{kind}/{change}` y el cambio `x`
- **CUANDO** se renderiza sin perfil bug conocido y de nuevo con perfil bug conocido
- **ENTONCES** los nombres son `feature/x` y `fix/x`

#### Scenario: `{slug}` coincide con `{change}` sin work item
- **DADO** la plantilla `{slug}` y el cambio `plantilla-ramas` sin work item
- **CUANDO** se renderiza
- **ENTONCES** el nombre es `plantilla-ramas`

#### Scenario: `{slug}` no distingue mayúsculas al quitar el prefijo
- **DADO** la plantilla `{slug}`, el `{ref}` `2525` y el cambio `US-2525-alta-de-usuarios`
- **CUANDO** se renderiza
- **ENTONCES** el nombre es `alta-de-usuarios`

#### Scenario: `{slug}` nunca queda vacío
- **DADO** la plantilla `{prefix}/{slug}`, el `{ref}` `2525` y el cambio `us-2525`
- **CUANDO** se renderiza
- **ENTONCES** `{slug}` vale `us-2525` y el nombre es `<prefijo>/us-2525`

#### Scenario: `{slug}` sin prefijo que case vale `{change}`
- **DADO** la plantilla `{slug}`, el `{ref}` `2525` y el cambio `alta-de-usuarios`
- **CUANDO** se renderiza
- **ENTONCES** el nombre es `alta-de-usuarios`

#### Scenario: `{prefix}` admite barras
- **DADO** el prefijo `equipo/inc` y la plantilla `{prefix}/{change}`
- **CUANDO** se renderiza para el cambio `x`
- **ENTONCES** el nombre es `equipo/inc/x`

#### Scenario: `{task_ref}` y `{role}` en una rama de rol
- **DADO** la plantilla `feature/TASK-{task_ref}`, el rol `frontend` y su Task con `id` `1235`
- **CUANDO** se renderiza el nombre de la rama atribuida a `frontend`
- **ENTONCES** el nombre es `feature/TASK-1235`

#### Scenario: Un valor no seguro para git no se sanea
- **DADO** la plantilla `feature/{task_ref}` y una Task con `id` `T#12`
- **CUANDO** se renderiza
- **ENTONCES** se rechaza con un error tipado que nombra `task_ref` y el valor `T#12`
- **Y** no se devuelve ningún nombre

#### Scenario: El nombre resultante inválido se rechaza
- **DADO** la plantilla `{prefix}/{change}/` y el cambio `x`
- **CUANDO** se renderiza
- **ENTONCES** se rechaza con `invalid_name`

### Requirement: Origen de los Valores de las Variables de la Plantilla (prov:BRA:2)

`ensure` y el cálculo de la instantánea (RF-KOF-06 de `sdd-lane-kickoff`) DEBEN compartir el mismo cálculo de valores y obtener cada uno de su fuente, con este orden de precedencia:

- `{ref}`: el valor congelado en la instantánea del kickoff si existe (prov:KOF:1 de `sdd-lane-kickoff`); si no, el `id` del vínculo `kind: work_item` de `integrations.yaml`; si el vínculo aún no existe, el `id` del work item del staging de `change start`, que se lee del git common dir y que la fase de adopción retira después del seal.
- `{kind}`: el valor congelado en la instantánea si existe; si no hay instantánea (el cambio se está creando), `fix` si el staging o el vínculo de `change start` indican el perfil bug y, si no, `feature`; en un cambio sellado con plantilla pero sin `kind` en la instantánea, `feature` (prov:KOF:1).
- `{task_ref}`: el `id` del vínculo `kind: task` del rol al que se atribuye la rama.
- `{change}`, `{slug}`, `{prefix}` y `{role}` salen del nombre del cambio, del `id` del work item, de la política y del rol atribuido.

Cuando hay un plugin de work items activo y `integrations.yaml` no se puede leer, el sistema DEBE fallar cerrado con `integrations_links_invalid` si la plantilla usa cualquiera de `{ref}`, `{slug}`, `{kind}` o `{task_ref}` y su valor no está congelado en la instantánea: no produce ningún nombre con valores incompletos, igual que la barrera `task_not_bound` (RF-BRA-33). Un valor ausente NUNCA DEBE sustituirse por una cadena vacía.

#### Scenario: `{ref}` del staging en el primer `ensure`, antes de la adopción
- **DADO** un `axiom.yaml` con `git.branch_template: "{prefix}/{ref}-{slug}"`, el prefijo `inc`, el staging de `change start` del cambio `us-2525-alta-de-usuarios` con `id` `2525` y sin vínculo `work_item` todavía en `integrations.yaml`
- **CUANDO** se ejecuta `ensure --change us-2525-alta-de-usuarios`
- **ENTONCES** la rama del cambio es `inc/2525-alta-de-usuarios`
- **Y** no se rechaza por `{ref}` sin valor

#### Scenario: `{ref}` del vínculo cuando existe
- **DADO** el vínculo `work_item` con `id` `2525` en `integrations.yaml` y sin staging
- **CUANDO** se resuelve `{ref}`
- **ENTONCES** vale `2525`

#### Scenario: `{kind}` del perfil bug
- **DADO** un staging o vínculo de `change start` con perfil bug y la plantilla `{kind}/{change}`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** la rama del cambio es `fix/x`

#### Scenario: `{ref}` sin work item ni staging se rechaza
- **DADO** la plantilla `{prefix}/{ref}`, un cambio sin vínculo `work_item` ni staging
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con el código tipado de variable sin valor (prov:BRA:4) que nombra `ref`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: `integrations.yaml` ilegible con plugin activo falla cerrado
- **DADO** un plugin de work items activo, un `integrations.yaml` ilegible y una plantilla con `{ref}`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con un error tipado con continuación
- **Y** no se crea ninguna rama ni worktree ni se escribe el registro

#### Scenario: `integrations.yaml` ilegible con `{slug}` o `{kind}` sin congelar falla cerrado
- **DADO** un plugin de work items activo, un `integrations.yaml` ilegible, una instantánea sin `kind` ni `ref` y la plantilla `{kind}/{slug}`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `integrations_links_invalid`
- **Y** no se crea ninguna rama ni worktree ni se escribe el registro

#### Scenario: `integrations.yaml` ilegible no impide usar valores congelados
- **DADO** un plugin de work items activo, un `integrations.yaml` ilegible y una instantánea con `kind` y `ref` congelados para la plantilla `{kind}/{ref}-{slug}`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** la rama se nombra con los valores congelados y no se rechaza

#### Scenario: `{task_ref}` del rol sin Task se rechaza
- **DADO** la plantilla `feature/TASK-{task_ref}` para el rol `backend` y ese rol sin vínculo `kind: task`
- **CUANDO** se intenta resolver el nombre de su rama
- **ENTONCES** se rechaza con el código tipado de variable sin valor que nombra `task_ref` y el rol `backend`

### Requirement: Plantilla Efectiva por Rol, Repositorio y Proyecto, y Precedencia de Nombres (prov:BRA:3)

La plantilla efectiva de cada rama DEBE resolverse con la precedencia rol (`roles.<id>.branch_template`) > repositorio (`repositories[].branch_template`) > proyecto (`git.branch_template`). Sin plantilla declarada en ningún nivel, el nombre de cada rama DEBE ser idéntico, byte a byte, al que producía el sistema antes de esta capacidad: `<prefijo>/<cambio>` para la rama del cambio, `<rama del cambio>-<rol>` para la de rol y `<rama del cambio>-archive` para la de archivado, y la instantánea, el registro y el JSON de `ensure` NO DEBEN cambiar.

La atribución de cada rama a un rol y la plantilla que le corresponde son:

- **Rama del cambio de un repositorio de código tocado por un solo rol** (incluido `fullstack`): se atribuye a ese rol; plantilla efectiva rol > repositorio > proyecto, con `{role}` y `{task_ref}` de ese rol.
- **Rama de seguimiento de un repositorio tocado por dos o más roles**: no se atribuye a ningún rol; plantilla efectiva repositorio > proyecto.
- **Rama de rol de un repositorio tocado por dos o más roles**: se atribuye a ese rol; usa su `roles.<id>.branch_template` si la declara y, si no, `<nombre de la rama de seguimiento>-<rol>`.
- **Rama que lleva `openspec/`** (la del cambio en `monorepo-embedded`, la del repositorio de specs en `multirepo`): sin rol; plantilla efectiva repositorio > proyecto, sin las plantillas de rol.
- **Rama de archivado**: sigue derivándose de la rama del cambio ya resuelta (`<rama del cambio>-archive`); no tiene plantilla propia. La rama del cambio ya resuelta es la que sale, por este orden, del registro, de la instantánea o del renderizado, y todos los consumidores del archivado (el plan de archivado, `branch done`, los cambios pendientes de archivar y la autocaptura) DEBEN obtenerla del mismo cálculo. Con plantilla, `--prefix` en `ensure --archive` NO cambia esa derivación; sin plantilla, el resultado es el de siempre.

Para cada rama, la precedencia de su nombre DEBE ser, de mayor a menor: (1) la excepción por cambio `--branch` (RF-BRA-12), que solo sustituye el nombre de la rama del cambio (la de seguimiento en un repositorio con ramas de rol) y es válida también con `--role`, con la misma semántica; si esa rama estaba pendiente por `{task_ref}`, `--branch` le da nombre y deja de estar pendiente, de modo que se crea en ese `ensure`; (2) el nombre de la instantánea del kickoff; (3) el nombre congelado en el registro local (RF-BRA-15); (4) la plantilla efectiva, de la instantánea o, si el cambio no tiene instantánea, de la política; (5) el nombre por defecto. La precedencia se aplica por rama: una instantánea sin nombre para una rama (la que usa `{task_ref}`) deja pasar al registro y, si este no lo tiene, a la plantilla. `--prefix` solo alimenta `{prefix}` y los nombres por defecto: si la plantilla no usa `{prefix}`, no tiene efecto y no da aviso. Las ramas de rol sin plantilla de rol siguen siendo `<rama de seguimiento>-<rol>`. El nombre de la carpeta del worktree NO DEBE cambiar: sigue usando el nombre del cambio. La colisión de nombres DEBE detectarse en el planificador sobre todos los nombres resueltos del repositorio en ese plan, incluidos los congelados del registro; al entrar un rol, su nombre renderizado se compara con los ya registrados del mismo repositorio. Si dos ramas del mismo repositorio resuelven al mismo nombre, el sistema DEBE rechazar con `invalid_name` nombrando ambas ramas y el repositorio, sin crear nada.

#### Scenario: Sin plantilla, nombres idénticos a los actuales
- **DADO** un `axiom.yaml` sin `branch_template` en ningún nivel, el prefijo `feat` y dos roles `core` y `qa` sobre el mismo repositorio de código
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** las ramas son `feat/x`, `feat/x-core` y `feat/x-qa`
- **Y** el JSON y el registro coinciden con los de antes de esta capacidad

#### Scenario: El rol prevalece sobre el repositorio y este sobre el proyecto
- **DADO** `git.branch_template: "{prefix}/{change}"`, `repositories[].branch_template: "feature/{ref}"` para `repo-backend` y `roles.backend.branch_template: "feature/TASK-{task_ref}"` con `backend` como único rol sobre `repo-backend`
- **CUANDO** se resuelve la plantilla efectiva de la rama del cambio de `repo-backend` y de `repo-frontend` (con el rol `frontend` único sobre él y sin plantilla de rol)
- **ENTONCES** la de `repo-backend` es la del rol `backend`
- **Y** la de `repo-frontend` es la del proyecto

#### Scenario: Cada repositorio de un multirepo puede nombrar su rama de forma distinta
- **DADO** un multirepo con `roles.backend.branch_template: "feature/{task_ref}"` y `roles.frontend.branch_template: "feature/TASK-{task_ref}"`, ambos con su Task `1235` vinculada y cada uno sobre su repositorio
- **CUANDO** se ejecuta `ensure --change x --role backend` y después `ensure --change x --role frontend`
- **ENTONCES** `repo-backend` recibe `feature/1235` y `repo-frontend` recibe `feature/TASK-1235`

#### Scenario: La rama de rol sin plantilla de rol deriva de la de seguimiento
- **DADO** `repositories[].branch_template: "{prefix}/{ref}-{slug}"` para el repositorio compartido por los roles `core` y `qa`, y ninguno con plantilla de rol, con `ref` `2525` y slug `alta`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** la rama de seguimiento es `<prefijo>/2525-alta`
- **Y** las ramas de rol son `<prefijo>/2525-alta-core` y `<prefijo>/2525-alta-qa`

#### Scenario: `--branch` prevalece sobre la plantilla
- **DADO** `git.branch_template: "{prefix}/{ref}-{slug}"` y la invocación `ensure --change x --branch hotfix/x`
- **CUANDO** se planifica
- **ENTONCES** la rama del cambio es `hotfix/x`

#### Scenario: `--prefix` alimenta `{prefix}`
- **DADO** `git.branch_template: "{prefix}/{change}"` y la invocación `ensure --change x --prefix fix`
- **CUANDO** se planifica
- **ENTONCES** la rama del cambio es `fix/x`

#### Scenario: `--prefix` no tiene efecto si la plantilla no usa `{prefix}`
- **DADO** `git.branch_template: "feature/{change}"` y la invocación `ensure --change x --prefix fix`
- **CUANDO** se planifica
- **ENTONCES** la rama del cambio es `feature/x`
- **Y** no se emite ningún aviso por el prefijo

#### Scenario: `--branch` prevalece sobre el nombre de la instantánea
- **DADO** un cambio sellado cuya instantánea nombra la rama del cambio `inc/2525-alta-de-usuarios` y la invocación `ensure --change us-2525-alta-de-usuarios --branch hotfix/alta`
- **CUANDO** se planifica
- **ENTONCES** la rama del cambio es `hotfix/alta`

#### Scenario: `--branch` en un repositorio con ramas de rol nombra solo la de seguimiento
- **DADO** un repositorio compartido por los roles `core` y `qa` y la invocación `ensure --change x --branch hotfix/x`
- **CUANDO** se planifica
- **ENTONCES** la rama de seguimiento es `hotfix/x`
- **Y** las ramas de rol siguen siendo `hotfix/x-core` y `hotfix/x-qa`

#### Scenario: `--branch` con `--role` da nombre a la rama pendiente y la crea
- **DADO** `repositories[].branch_template: "feature/TASK-{task_ref}"` para `repo-frontend`, el rol `frontend` único sobre él con su Task vinculada y la invocación `ensure --change x --role frontend --branch hotfix/front`
- **CUANDO** se ejecuta
- **ENTONCES** `repo-frontend` recibe la rama `hotfix/front` y esa rama deja de estar pendiente
- **Y** el nombre queda registrado

#### Scenario: Instantánea sin nombre y registro con nombre
- **DADO** una instantánea que solo guarda la plantilla `feature/TASK-{task_ref}` de `repo-frontend` y un registro que congela `feature/TASK-1235`, con la plantilla de `axiom.yaml` cambiada después
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** el nombre es `feature/TASK-1235`, el del registro
- **Y** no se renderiza la plantilla de `axiom.yaml`

#### Scenario: Con plantilla, `--prefix` no cambia la rama de archivado
- **DADO** la rama del cambio `inc/2525-alta-de-usuarios` resuelta por plantilla y registrada
- **CUANDO** se ejecuta `ensure --change us-2525-alta-de-usuarios --archive --prefix fix`
- **ENTONCES** la rama de archivado es `inc/2525-alta-de-usuarios-archive`

#### Scenario: Los consumidores del archivado usan la rama del cambio resuelta
- **DADO** un cambio pendiente de archivar con rama del cambio `inc/2525-alta-de-usuarios` por plantilla, sin instantánea completa ni registro
- **CUANDO** el plan de archivado, `branch done` y `status` (cambios pendientes de archivar) calculan el nombre de su rama
- **ENTONCES** los tres obtienen `inc/2525-alta-de-usuarios` y su archivado `inc/2525-alta-de-usuarios-archive`, del mismo cálculo
- **Y** sin plantilla el resultado es idéntico, byte a byte, al de antes de esta capacidad

#### Scenario: Un nombre ya registrado de otro rol colisiona al entrar un rol
- **DADO** el rol `core` con su rama `feature/7` registrada en un repositorio compartido y el rol `qa` cuya plantilla de rol renderiza `feature/7`
- **CUANDO** se ejecuta `ensure --change x --role qa`
- **ENTONCES** se rechaza con `invalid_name` nombrando ambas ramas y el repositorio
- **Y** no se crea ninguna rama

#### Scenario: El worktree conserva el nombre del cambio
- **DADO** una plantilla que produce la rama `inc/2525-alta-de-usuarios` para el cambio `us-2525-alta-de-usuarios` en modo `worktree`
- **CUANDO** se ejecuta `ensure`
- **ENTONCES** la carpeta del worktree usa el nombre del cambio y no el de la rama

#### Scenario: La rama de archivado deriva de la del cambio con plantilla
- **DADO** una rama del cambio `inc/2525-alta-de-usuarios` resuelta por plantilla
- **CUANDO** se ejecuta `ensure --change us-2525-alta-de-usuarios --archive`
- **ENTONCES** la rama de archivado es `inc/2525-alta-de-usuarios-archive`

#### Scenario: Dos ramas del mismo repositorio con el mismo nombre se rechazan
- **DADO** un repositorio compartido por los roles `core` y `qa` cuyos `roles.core.branch_template` y `roles.qa.branch_template` son `feature/{ref}` y producen el mismo nombre
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `invalid_name` nombrando ambas ramas y el repositorio
- **Y** no se crea ninguna rama

### Requirement: Rechazos Tipados de la Resolución de Plantillas (prov:BRA:4)

Todo rechazo de la resolución de plantillas DEBE ser un rechazo tipado de `branch` con código, mensaje y continuación, DEBE producirse antes de mutar nada (ni ramas, ni worktrees, ni registro) y NO DEBE sustituir el valor ausente o inválido por un respaldo silencioso. Los casos son:

- **Variable sin valor** al resolver (`{ref}` sin work item ni staging, `{task_ref}` sin Task del rol, integraciones ilegibles con plugin activo): código tipado propio de variable sin valor, cuyo nombre exacto fija el diseño; la continuación DEBE nombrar la variable, el rol cuando proceda y la orden o la clave de `axiom.yaml` que lo resuelve.
- **Nombre renderizado inválido** o valor de variable no seguro para git: `invalid_name`, nombrando la variable, el valor y la clave de plantilla.
- **Rama compartida por dos o más roles**: la rama de seguimiento de un repositorio tocado por dos o más roles, o la rama que lleva `openspec/`, cuya plantilla efectiva use `{role}` o `{task_ref}`, DEBE rechazarse al planificar con el código `branch_template_not_applicable` (motivo `shared_branch` u `openspec_branch`) que nombra el repositorio y la clave `repositories[].branch_template` como continuación; no existe rama ni nombre alternativo. El mismo rechazo se produce en `axiom sdd kickoff seal`, que planifica, propaga el rechazo y no escribe `kickoff.yaml` (para no sellar una instantánea irresoluble), en `ensure` (SDD y ODD) y en `axiom workspace validate` solo para lo detectable de forma estática (la rama con `openspec/`, prov:TOP:1 de `workspace-topology`).
- **Carril ODD**: `{role}` y `{task_ref}` NO resuelven en ODD, que no tiene roles ni Tasks (RF-BRA-14): su uso en la plantilla aplicable DEBE rechazarse con `branch_template_not_applicable` (motivo `odd_lane`) que nombra la variable y la clave.
- **Nombre que choca con una rama reservada**: un nombre renderizado igual a la rama de integración o a `main_branch` del repositorio DEBE rechazarse con `invalid_name` en el `ensure`, sin crear nada. La colisión con la rama de otro cambio activo queda fuera de esta capacidad como limitación declarada: una plantilla sin variable propia del cambio (`{change}`, `{slug}`, `{ref}` o `{task_ref}`) se rechaza al cargar (prov:TOP:1), que es la mitigación.
- **Plantilla inválida en `axiom.yaml`** (variable desconocida, llaves mal formadas): se rechaza al cargar con `config_invalid` (prov:TOP:1 de `workspace-topology`).

Los códigos nuevos DEBEN figurar con su etiqueta y su continuación en el conjunto de rechazos de producción, de modo que el trinquete de resolución de rechazos los cubra.

#### Scenario: Variable sin valor con continuación
- **DADO** la plantilla `feature/TASK-{task_ref}` y el rol `backend` sin Task vinculada
- **CUANDO** se ejecuta `ensure --change x --role backend` con la barrera `task_not_bound` inactiva
- **ENTONCES** se rechaza con el código tipado de variable sin valor
- **Y** la continuación nombra `task_ref`, el rol `backend` y `axiom integrations tasks bind --change x --role backend=<id>`
- **Y** no se crea ninguna rama, worktree ni registro

#### Scenario: Nombre renderizado inválido
- **DADO** la plantilla `{prefix}/{change}.lock` y el cambio `x`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `invalid_name` que nombra la clave de la plantilla y el nombre resultante
- **Y** no se crea nada

#### Scenario: Rama de seguimiento con `{task_ref}`
- **DADO** un repositorio compartido por los roles `core` y `qa` cuya plantilla efectiva de seguimiento es `repositories[].branch_template: "feature/TASK-{task_ref}"`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con el código tipado de rama compartida
- **Y** la continuación nombra el repositorio y `repositories[].branch_template`
- **Y** no se crea ninguna rama

#### Scenario: Rama de seguimiento con `{role}`
- **DADO** un repositorio compartido por dos roles con `git.branch_template: "{prefix}/{role}/{change}"` como plantilla efectiva de seguimiento
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con el código tipado de rama compartida y no se crea nada

#### Scenario: Plantilla con `{role}` en el carril ODD
- **DADO** `git.branch_template: "{prefix}/{role}/{change}"`
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature`
- **ENTONCES** se rechaza con un código tipado que nombra `{role}` y `git.branch_template`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: El seal rechaza la rama compartida con `{role}` o `{task_ref}`
- **DADO** un repositorio compartido por los roles `core` y `qa` con `git.branch_template: "feature/{task_ref}"` como plantilla de seguimiento
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** se rechaza con `branch_template_not_applicable` (motivo `shared_branch`) y la continuación nombra `repositories[].branch_template`
- **Y** no se escribe `kickoff.yaml`

#### Scenario: La rama con `openspec/` rechaza `{role}` en `ensure`
- **DADO** un `monorepo-embedded` con `git.branch_template: "{prefix}/{role}/{change}"`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `branch_template_not_applicable` (motivo `openspec_branch`) y no se crea nada

#### Scenario: Un nombre igual a la rama de integración se rechaza
- **DADO** `git.integration_branch: develop` y la plantilla `{change}` con el cambio `develop`
- **CUANDO** se ejecuta `ensure --change develop`
- **ENTONCES** se rechaza con `invalid_name` que nombra la rama de integración
- **Y** no se crea ninguna rama

#### Scenario: Un nombre igual a `main_branch` se rechaza
- **DADO** `git.main_branch: master` y la plantilla `{change}` con el cambio `master`
- **CUANDO** se ejecuta `ensure --change master`
- **ENTONCES** se rechaza con `invalid_name` que nombra `main_branch`
- **Y** no se crea ninguna rama

#### Scenario: Cada rechazo lleva continuación
- **DADO** el conjunto de rechazos de producción de `branch`
- **CUANDO** se ejecuta el trinquete de resolución de rechazos
- **ENTONCES** los códigos nuevos de plantilla tienen etiqueta y continuación no vacía

### Requirement: Diferido de las Ramas con `{task_ref}` a la Entrada del Rol (prov:BRA:5)

Una rama cuya plantilla efectiva usa `{task_ref}` DEBE estar **pendiente** hasta que entre el rol al que se atribuye: NO DEBE crearse ni nombrarse en un `ensure` sin `--role`, ni en un `ensure --role <otro rol>`, ni al arrancar el cambio. DEBE crearse en el `ensure --change <c> --role <rol>` de su rol, después de la barrera `task_not_bound` (RF-BRA-33) y de la resolución de `{task_ref}`. El motor de `ensure` DEBE recibir el rol de `--role` solo para ese fin. Una rama cuya plantilla no usa `{task_ref}`, y toda rama sin plantilla, DEBE crearse en el mismo momento en que se crea hoy. Una rama pendiente DEBE listarse como pendiente, con su repositorio, su rol y su plantilla, tanto en la salida de `ensure` como en la de `status` (RF-BRA-16), mediante un campo aditivo del JSON; un binario anterior ignora ese campo. `ensure --role` DEBE ser idempotente: repetirlo no vuelve a crear la rama ni cambia su nombre. La entrada de un rol NO DEBE crear ni nombrar ramas pendientes de otro rol.

`status` NUNCA DEBE mutar nada. Una rama pendiente se lista con su plantilla, su rol y la orden que la crea (`axiom branch ensure --cwd <ruta> --change <c> --role <rol>`), y sin nombre esperado si aún no se puede resolver. Con un plugin de work items activo y `integrations.yaml` ilegible, `status` DEBE informar el código del rechazo (`integrations_links_invalid`) en la entrada afectada, sin fallar la orden entera. La base, el destino de PR y la resolución del modo de promoción de una rama diferida, incluida la pregunta `ask` (RF-BRA-28 y RF-BRA-29), se resuelven en el `ensure --role` que la crea, con la misma regla que si se creara al arrancar el cambio. El worktree de una rama diferida solo se materializa al crearla: mientras tanto, un repositorio sin ninguna rama creada no tiene worktree.

#### Scenario: La rama con `{task_ref}` nace al entrar el rol
- **DADO** un multirepo con `repositories[].branch_template: "feature/TASK-{task_ref}"` para `repo-frontend`, el rol `frontend` único sobre él y su Task `1235` vinculada, y el cambio ya sellado
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** `repo-frontend` recibe la rama `feature/TASK-1235` desde su base
- **Y** el nombre queda registrado

#### Scenario: Antes de la entrada del rol la rama está pendiente
- **DADO** el mismo multirepo y un `ensure --change x` sin `--role` posterior al sellado
- **CUANDO** termina
- **ENTONCES** la rama de `repo-frontend` no existe ni se ha nombrado
- **Y** el JSON de `ensure` y el de `status` listan una rama pendiente con el repositorio `repo-frontend`, el rol `frontend` y la plantilla `feature/TASK-{task_ref}`
- **Y** las ramas sin `{task_ref}` de los demás repositorios sí se han creado

#### Scenario: La entrada de un rol no toca las ramas pendientes de otro
- **DADO** los roles `backend` y `frontend` con Task vinculada, cada uno con su plantilla con `{task_ref}`
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** se crea solo la rama de `backend`
- **Y** la de `frontend` sigue pendiente

#### Scenario: Repetir `ensure --role` es idempotente
- **DADO** la rama `feature/TASK-1235` creada por `ensure --change x --role frontend`
- **CUANDO** se repite la misma orden
- **ENTONCES** la rama se reutiliza sin modificarla y su nombre no cambia

#### Scenario: La barrera `task_not_bound` precede a la creación
- **DADO** un plugin activo con `items.create`, un vínculo `work_item` y el rol `frontend` sin Task
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** se rechaza con `task_not_bound` y no se crea la rama pendiente

#### Scenario: Segundo clon con `ensure --role`, sin registro y con la rama ya publicada
- **DADO** un cambio sellado con `repositories[].branch_template: "feature/TASK-{task_ref}"` para `repo-frontend`, la Task `1235` del rol `frontend` vinculada y versionada en `integrations.yaml`, la rama `feature/TASK-1235` ya publicada en el remoto por otra persona, y un segundo clon sin registro ni staging
- **CUANDO** se ejecuta `ensure --change x --role frontend` en el segundo clon
- **ENTONCES** el nombre es `feature/TASK-1235`, obtenido de la plantilla de la instantánea y de los vínculos versionados
- **Y** se adopta la rama publicada (RF-BRA-27) sin crear otra ni modificarla
- **Y** el registro se reconstruye con ese nombre

#### Scenario: `status` no muta y lista la rama pendiente con su orden
- **DADO** una rama pendiente de `repo-frontend` para el rol `frontend` cuyo `{task_ref}` aún no se puede resolver
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** la entrada pendiente incluye la plantilla, el rol `frontend` y la orden `axiom branch ensure --cwd <ruta> --change x --role frontend`, sin nombre esperado
- **Y** no se crea ninguna rama, worktree ni registro

#### Scenario: `status` con `integrations.yaml` ilegible informa el código sin fallar
- **DADO** un plugin de work items activo, un `integrations.yaml` ilegible y una rama pendiente por `{task_ref}`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** la entrada afectada informa `integrations_links_invalid`
- **Y** el resto del estado se devuelve y la orden no falla

#### Scenario: La promoción `ask` de una rama diferida se resuelve al crearla
- **DADO** `git.main_branch: master`, `git.promotion: ask` y una rama pendiente de `repo-frontend`
- **CUANDO** se ejecuta `ensure --change x` y después `ensure --change x --role frontend`
- **ENTONCES** la primera no pregunta por el modo de promoción de `repo-frontend` ni crea su worktree
- **Y** la segunda resuelve la base, el destino de PR y el modo (incluida la pregunta `ask`) con la misma regla que si la rama se hubiera creado al arrancar

#### Scenario: Sin `{task_ref}` nada se difiere
- **DADO** plantillas sin `{task_ref}` o sin plantillas
- **CUANDO** se ejecuta `ensure --change x` y después `ensure --change x --role backend`
- **ENTONCES** ambas invocaciones producen el mismo plan, las mismas ramas y el mismo JSON
- **Y** el JSON no lista ramas pendientes

## MODIFIED Requirements

### Requirement: Resolución de la Política de Ramas (RF-BRA-03)

La política de cada repositorio DEBE resolverse con esta precedencia, de mayor a menor: (1) las excepciones por cambio de `ensure` (`--prefix`, `--branch`, `--base`, RF-BRA-12); (2) la entrada del repositorio en `repositories:` de `axiom.yaml`, si existe; (3) el bloque `git:` de `axiom.yaml`; (4) los valores por defecto: rama de integración `main` y prefijo `feat`. La política incorpora además la plantilla de nombre de rama (`branch_template`), con su propia precedencia de plantilla efectiva, rol (`roles.<id>.branch_template`) > repositorio (`repositories[].branch_template`) > `git:` (`git.branch_template`), definida en prov:BRA:3 y sin valor por defecto: su ausencia en todos los niveles significa los nombres de siempre. Sin bloque `git:`, `integration_branch` DEBE valer `main` y `branch_prefix` DEBE valer `feat`; sin `repositories:`, ningún repositorio tiene ajustes propios y todos heredan los valores del workspace. La política incorpora además la rama principal (`git.main_branch`, solo en `git:` y sin valor por repositorio ni por defecto: su ausencia significa una sola etapa) y el modo de promoción por defecto (`promotion`, con herencia `repositories:` > `git:`; el valor vacío hereda); el modo efectivo de cada repositorio y cambio se resuelve en RF-BRA-28. Los nombres de rama resultantes DEBEN ser nombres de referencia git válidos: un prefijo o nombre inválido indicado en la invocación (`--prefix`, `--branch`, `--change`) DEBE rechazarse con `invalid_name`, y un valor inválido (incluida una plantilla `branch_template` inválida) o una configuración ilegible en `axiom.yaml` o en el hub personal con `config_invalid`, en ambos casos sin crear nada. Una variable de plantilla sin valor y un nombre renderizado inválido se rechazan con los códigos tipados de prov:BRA:4.
(Previously: la política no incluía la rama principal ni el modo de promoción)
(Previously: la política no incluía la plantilla de nombre de rama ni sus rechazos tipados)

#### Scenario: Sin configuración se aplican los valores por defecto
- **DADO** un `axiom.yaml` sin bloque `git:` ni `repositories:`
- **CUANDO** se planifica el cambio `x`
- **ENTONCES** la rama de integración es `main` y la rama del cambio es `feat/x`

#### Scenario: El bloque `git:` sustituye los valores por defecto
- **DADO** un `axiom.yaml` con `git.integration_branch: develop` y `git.branch_prefix: inc`
- **CUANDO** se planifica el cambio `x`
- **ENTONCES** la rama de integración es `develop` y la rama del cambio es `inc/x`

#### Scenario: Un repositorio con ajustes propios prevalece sobre `git:`
- **DADO** un `axiom.yaml` con `git.integration_branch: main` y una entrada de `repositories:` para `repo-backend` con rama de integración `release`
- **CUANDO** se planifica un cambio que toca `repo-backend` y `repo-frontend`
- **ENTONCES** la base de `repo-backend` es `release` y la de `repo-frontend` es `main`

#### Scenario: La excepción por cambio prevalece sobre todo lo demás
- **DADO** la política anterior y la invocación `ensure --change x --prefix fix`
- **CUANDO** se planifica el cambio
- **ENTONCES** la rama del cambio es `fix/x` en todos los repositorios del plan

#### Scenario: Un prefijo inválido de la invocación se rechaza sin efectos
- **DADO** un `ensure` con `--prefix "mi prefijo//"`
- **CUANDO** se ejecuta
- **ENTONCES** se rechaza con el código tipado `invalid_name`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: Un prefijo inválido de `axiom.yaml` se rechaza como configuración
- **DADO** un `axiom.yaml` con `git.branch_prefix: "inc//"`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con el código tipado `config_invalid`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: La política hereda la rama principal y el modo de promoción
- **DADO** un `axiom.yaml` con `git.main_branch: master`, `git.promotion: branch` y una entrada de `repositories:` para `repo-backend` con `promotion: release`
- **CUANDO** se resuelve la política de `repo-backend` y de `repo-frontend`
- **ENTONCES** ambos tienen la rama principal `master`
- **Y** el modo por defecto de `repo-backend` es `release` y el de `repo-frontend` es `branch`

#### Scenario: Sin `main_branch` la política no inventa una rama principal
- **DADO** un `axiom.yaml` con `git.integration_branch: develop` y sin `git.main_branch`
- **CUANDO** se resuelve la política
- **ENTONCES** la rama principal queda vacía y el repositorio es de una sola etapa

#### Scenario: La plantilla de `git:` entra en la política y nombra la rama del cambio
- **DADO** un `axiom.yaml` con `git.branch_prefix: inc` y `git.branch_template: "{prefix}/{ref}-{slug}"`, y el cambio `us-2525-alta-de-usuarios` con work item `2525`
- **CUANDO** se resuelve la política y se planifica el cambio
- **ENTONCES** la política declara esa plantilla y la rama del cambio es `inc/2525-alta-de-usuarios`

#### Scenario: Una plantilla inválida de `axiom.yaml` se rechaza como configuración
- **DADO** un `axiom.yaml` con `git.branch_template: "{prefix}/{desconocida}"`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con el código tipado `config_invalid` que nombra `git.branch_template`
- **Y** no se crea ninguna rama ni worktree


### Requirement: Modelo de Ramas por Número de Roles que Tocan Cada Repositorio (RF-BRA-04)

El modelo de ramas de cada repositorio con rama de cambio DEBE depender de cuántos roles del cambio lo tocan (roles sellados en el `kickoff.yaml` del cambio intersecados con los roles de `axiom.yaml` cuyos repositorios resuelven a esa raíz git):

- **Un rol** (lo habitual, también `fullstack`): solo existe la rama `<prefijo>/<cambio>`; el rol trabaja en ella y su PR se dirige a la rama de integración.
- **Dos o más roles** (solo posible en `multirepo`: en `monorepo-embedded` el único rol es `fullstack`, RF-TOP-04 de `workspace-topology`, y por tanto nunca hay rama de rol): `<prefijo>/<cambio>` es la rama de seguimiento y cada rol saca `<prefijo>/<cambio>-<rol>` desde ella. Con plantillas de nombre de rama (prov:BRA:3), el nombre de la rama de seguimiento sale de la plantilla efectiva del repositorio o de `git:` y el de la rama de rol, de `roles.<id>.branch_template` o, si no se declara, de `<rama de seguimiento>-<rol>`; la rama de seguimiento NO PUEDE usar `{role}` ni `{task_ref}` (prov:BRA:4). Hay un PR de cada rol contra la rama de seguimiento y un PR final a la rama de integración que lleva el cambio (código y artefactos), sin el archive. Se RECOMIENDA el merge commit, sin imponerlo.
- **Rama de archivado** (solo en repositorios con rama de cambio): una vez fusionado en la rama de integración el PR del cambio, el archive se hace en `<prefijo>/<cambio>-archive`, que sale de la rama de integración (RF-BRA-13) y se entrega en un PR documental aparte contra la rama de integración. NO existe en `multirepo`.

El modelo DEBE evaluarse por repositorio: en un mismo cambio un repositorio puede tener rama de rol y otro no. Un rol cuyo nombre sea `archive` colisionaría con la rama de archivado y DEBE rechazarse con `invalid_name` al planificar las ramas de rol (supuesto S-15). `ensure` NO DEBE publicar la rama de seguimiento: DEBE avisar de que la persona la sube al remoto antes del primer PR de rol. La rama destino propuesta para cada PR (rol contra seguimiento, cambio contra integración, archivado contra integración) DEBE poder obtenerse del plan (`status`, RF-BRA-16) para que `sdd-apply` la proponga en lugar de preguntarla.
(Previously: el PR final a la rama de integración llevaba el archive y no existía rama de archivado; los escenarios de ramas de rol, de la colisión con `archive` y del destino de los PR se ejecutaban sobre un monorepo con los roles `core` y `qa`, y ahora se ejecutan sobre `multirepo` o sobre el rol `fullstack`)
(Previously: los nombres de la rama de seguimiento y de la rama de rol eran siempre `<prefijo>/<cambio>` y `<prefijo>/<cambio>-<rol>`)

#### Scenario: Un rol, rama única
- **DADO** un monorepo con el rol único `fullstack` sellado en el kickoff del cambio `x`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** existe solo la rama `feat/x`
- **Y** no existe ninguna rama `feat/x-<rol>`

#### Scenario: Dos roles sobre el mismo repositorio crean seguimiento y ramas de rol
- **DADO** un multirepo con los roles `core` y `qa` sellados en el kickoff del cambio `x`, ambos con repositorios que resuelven a la misma raíz git de código `repo-backend`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** en `repo-backend` existe la rama de seguimiento `feat/x` desde la base
- **Y** existen `feat/x-core` y `feat/x-qa`, ambas creadas desde `feat/x`
- **Y** la salida avisa de que la rama de seguimiento debe publicarla la persona antes del primer PR de rol
- **Y** no se ejecuta ningún `git push`

#### Scenario: En multirepo cada repositorio de código con un rol usa la rama del cambio
- **DADO** un multirepo con el rol `backend` sobre `repo-backend` y el rol `frontend` sobre `repo-frontend`, ambos sellados en el kickoff del cambio `x`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** `repo-backend` y `repo-frontend` reciben cada uno la rama `feat/x` desde su rama de integración
- **Y** ninguno recibe rama de rol ni rama de archivado

#### Scenario: Dos roles que comparten un repositorio de código en multirepo
- **DADO** un multirepo con los roles `api` y `jobs` sobre el mismo `repo-backend` y el rol `frontend` sobre `repo-frontend`
- **CUANDO** se ejecuta `ensure --change x` con los tres roles sellados
- **ENTONCES** `repo-backend` recibe `feat/x`, `feat/x-api` y `feat/x-jobs`
- **Y** `repo-frontend` recibe solo `feat/x`

#### Scenario: Un rol sellado que no toca ningún repositorio declarado no crea ramas
- **DADO** un rol sellado en el kickoff que no figura en `axiom.yaml` y que no es `fullstack`
- **CUANDO** se planifica el cambio
- **ENTONCES** ese rol no contribuye a ningún repositorio ni a ninguna rama
- **Y** la salida incluye un aviso que nombra el rol

#### Scenario: Un rol llamado `archive` colisiona con la rama de archivado
- **DADO** un multirepo con los roles `core` y `archive` sellados sobre el mismo repositorio de código
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `invalid_name` nombrando el rol `archive` y la colisión con el nombre reservado `feat/x-archive`
- **Y** no se crea ninguna rama de rol

#### Scenario: La rama de archivado no pertenece al modelo de roles
- **DADO** un monorepo con el rol único `fullstack` y las ramas `feat/x` y `feat/x-archive`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el destino propuesto del PR de `feat/x` es la rama de integración
- **Y** el de la rama de archivado `feat/x-archive` es la rama de integración
- **Y** el plan no contiene ninguna rama de rol



#### Scenario: Con plantilla en un solo rol la rama del cambio sigue siendo única
- **DADO** un multirepo con el rol `backend` sobre `repo-backend` y `roles.backend.branch_template: "feature/{task_ref}"` con Task `1235`
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** `repo-backend` recibe solo la rama `feature/1235`, sin rama de seguimiento ni de rol

#### Scenario: Dos roles con plantilla de rol propia
- **DADO** un repositorio compartido por los roles `api` y `jobs`, con `repositories[].branch_template: "{prefix}/{ref}"`, `roles.api.branch_template: "feature/API-{task_ref}"` y `roles.jobs.branch_template: "feature/JOBS-{task_ref}"`, con Tasks `11` y `12`
- **CUANDO** se ejecuta `ensure --change x`, después `ensure --change x --role api` y después `ensure --change x --role jobs`
- **ENTONCES** la primera crea la rama de seguimiento `<prefijo>/<ref>`
- **Y** la segunda crea `feature/API-11` desde la rama de seguimiento y la tercera `feature/JOBS-12` desde la misma

### Requirement: Carril ODD con la Misma Política (RF-BRA-14)

`ensure --lane odd --change <feature>` DEBE aplicar la misma política que SDD, incluida la plantilla de nombre de rama: la rama de la feature se nombra con la plantilla efectiva (`git.branch_template`, o `repositories[].branch_template` en un repositorio de código incluido con `--repo`) y su nombre se congela en el registro local (RF-BRA-15) en el primer `ensure`. El carril ODD no tiene roles ni Tasks, por lo que `{role}` y `{task_ref}` NO resuelven y su uso se rechaza con el código tipado de prov:BRA:4; `roles.<id>.branch_template` no se aplica nunca. Como no hay `kickoff.yaml`, los valores salen así: `{ref}` del vínculo `work_item` de `integrations.yaml` (que ODD también escribe al adoptar) o, si aún no existe, del staging de `change start`; `{kind}` del perfil del staging o del documento adoptado y, si no hay, `feature`; `{slug}` según prov:BRA:1. Una plantilla con `{ref}` y sin work item en ODD se rechaza con el código de variable sin valor de prov:BRA:4. Con estas particularidades: el documento ODD es `<workspaceRoot>/odd/tasks/<feature>.md`, en el repositorio del planning home. En `monorepo-embedded`, la rama es `<prefijo>/<feature>` con worktree según el modo; en `multirepo`, el repositorio de specs se trabaja in situ en su rama de integración (RF-BRA-05) y los repositorios de código se incluyen solo bajo demanda con `--repo <ruta>`, porque ODD no divide en roles y cada repositorio incluido recibe solo la rama `<prefijo>/<feature>`. En el carril ODD no existe `kickoff.yaml` ni instantánea ni rama de archivado: las únicas protecciones que se aplican son la propia política de `ensure` y la guarda D (RF-BRA-22), que bloquea las escrituras del agente en `odd/tasks/` fuera de la rama que corresponde, porque no hay escritor Go del documento ODD.
(Previously: la rama de la feature era siempre `<prefijo>/<feature>`)

#### Scenario: ODD en monorepo prepara rama y worktree
- **DADO** un monorepo `monorepo-embedded` en modo `worktree`
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature`
- **ENTONCES** se crea la rama `feat/mi-feature` y su worktree
- **Y** el JSON indica `<worktree>/odd/tasks/mi-feature.md` como ubicación del documento ODD

#### Scenario: ODD en multirepo trabaja in situ en el repositorio de specs
- **DADO** un multirepo con el repositorio de specs en su rama de integración
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature`
- **ENTONCES** no se crea rama ni worktree en el repositorio de specs
- **Y** el JSON indica `<repositorio de specs>/odd/tasks/mi-feature.md`

#### Scenario: Un repositorio de código se incluye con `--repo`
- **DADO** un multirepo con `repo-backend`
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature --repo repo-backend`
- **ENTONCES** `repo-backend` recibe solo la rama `feat/mi-feature`, sin rama de rol

#### Scenario: ODD no crea `kickoff.yaml`
- **DADO** un monorepo
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature`
- **ENTONCES** no se crea ningún fichero bajo `openspec/changes/`


#### Scenario: ODD aplica la plantilla de `git:` y congela el nombre
- **DADO** un monorepo con `git.branch_template: "{kind}/{change}"`
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature`
- **ENTONCES** se crea la rama `feature/mi-feature`
- **Y** el registro guarda ese nombre y un `ensure` posterior lo reutiliza aunque cambie la plantilla

#### Scenario: ODD en un repositorio de código usa la plantilla del repositorio
- **DADO** un multirepo con `repositories[].branch_template: "feature/{change}"` para `repo-backend`
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature --repo repo-backend`
- **ENTONCES** `repo-backend` recibe la rama `feature/mi-feature`

#### Scenario: ODD sin plantilla conserva el nombre actual
- **DADO** un `axiom.yaml` sin plantillas y el prefijo `feat`
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature`
- **ENTONCES** la rama es `feat/mi-feature`

#### Scenario: ODD toma `{ref}` y `{kind}` del staging
- **DADO** `git.branch_template: "{kind}/{ref}-{slug}"`, el staging de `change start` del cambio `us-2525-alta` con `id` `2525` y perfil bug, y sin vínculo `work_item` todavía
- **CUANDO** se ejecuta `ensure --lane odd --change us-2525-alta`
- **ENTONCES** la rama es `fix/2525-alta`
- **Y** el registro congela ese nombre

#### Scenario: ODD con `{kind}` sin perfil conocido vale `feature`
- **DADO** `git.branch_template: "{kind}/{change}"` y un cambio ODD sin staging ni perfil en el documento
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature`
- **ENTONCES** la rama es `feature/mi-feature`

#### Scenario: ODD con `{ref}` sin work item se rechaza
- **DADO** `git.branch_template: "{prefix}/{ref}"` y un cambio ODD sin vínculo `work_item` ni staging
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature`
- **ENTONCES** se rechaza con el código de variable sin valor que nombra `ref`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: ODD rechaza `{task_ref}`
- **DADO** `git.branch_template: "feature/{task_ref}"`
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature`
- **ENTONCES** se rechaza con el código tipado de prov:BRA:4 que nombra `{task_ref}`
- **Y** no se crea ninguna rama ni worktree

### Requirement: Registro Local en el Git Common Dir (RF-BRA-15)

`ensure` DEBE mantener un registro local por cambio, fuera de cualquier árbol versionado, en el git common dir del repositorio del planning home, obtenido con `git rev-parse --git-common-dir` (nunca asumiendo `.git`). El registro DEBE contener las rutas absolutas de todos los repositorios, ramas (incluida la de archivado) y worktrees del cambio, la respuesta a `ask` del modo de trabajo (RF-BRA-08) y, por cada repositorio con dos etapas, el modo de promoción resuelto y su origen (RF-BRA-28). Los campos del modo de promoción DEBEN ser opcionales: un registro sin ellos sigue siendo válido y un binario anterior los ignora. Cuando el nombre de una rama se obtiene de una plantilla (prov:BRA:3), el primer `ensure` que crea esa rama DEBE congelar el nombre resuelto en el registro, y las ejecuciones posteriores DEBEN leer ese nombre sin volver a renderizar la plantilla, de modo que un cambio posterior de la plantilla en `axiom.yaml` no renombra ni duplica una rama ya creada. El registro es la fuente de verdad del nombre congelado. Si el registro se borra, `ensure` DEBE recuperar el nombre de la instantánea o, si esta no lo guarda, de renderizar la plantilla de la instantánea con las entradas versionadas actuales, y reutilizar la rama existente con ese nombre sin renombrar nada; un error de git distinto de «la rama no existe» al comprobarla se propaga como error y nunca se trata como ausencia. Es una limitación declarada: si la Task de un rol se revinculó a otro `id` después de crear la rama y el registro se perdió, el nombre renderizado puede diferir del de la rama existente; la salida es `--branch` con el nombre existente o restaurar el vínculo anterior. Una rama pendiente (prov:BRA:5) NO DEBE figurar en el registro. El registro es la fuente operativa del modo de promoción en todos los carriles. Debe ser accesible con el mismo contenido desde el checkout principal y desde cualquier worktree enlazado del mismo repositorio. Si el registro se borra, el repositorio NO DEBE verse afectado y `ensure` DEBE reconstruirlo de forma idempotente a partir del estado de git (ramas y worktrees existentes) y, para el modo de promoción, de la instantánea del `kickoff.yaml` si la hay (RF-BRA-28), mientras `status` DEBE informar que el cambio no tiene registro.
(Previously: el registro no guardaba el modo de promoción ni su origen y la reconstrucción no consultaba la instantánea)
(Previously: el registro no congelaba el nombre de las ramas nombradas por plantilla ni leía ese nombre al planificar)
(Previously: la cláusula sobre la coexistencia con el estado local de RDD (`<git common dir>/gentle-ai/`) y sus linajes; RDD se retira (C-01-e) y `ensure` ya no tiene nada que preservar de él.)

#### Scenario: El registro se resuelve con el git common dir
- **DADO** un repositorio cuyo directorio git no se llama `.git` (git dir enlazado)
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el registro se crea bajo el directorio devuelto por `git rev-parse --git-common-dir`

#### Scenario: El registro es visible desde un worktree enlazado
- **DADO** un cambio preparado por `ensure` en modo `worktree`
- **CUANDO** se ejecuta `axiom branch status --change x --cwd <ruta del worktree>`
- **ENTONCES** el resultado incluye las rutas absolutas de todos los repositorios y worktrees del cambio registrados desde el checkout principal

#### Scenario: Borrar el registro no daña el repositorio
- **DADO** un cambio preparado y su registro borrado
- **CUANDO** se ejecuta `ensure --change x` de nuevo
- **ENTONCES** las ramas y worktrees existentes se reutilizan sin modificarlos
- **Y** el registro se reconstruye con las mismas rutas absolutas

#### Scenario: Registro ausente informado por `status`
- **DADO** un cambio sin registro local
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON indica que el cambio no tiene registro y no falla

#### Scenario: El registro guarda el modo de promoción y su origen por repositorio
- **DADO** un multirepo con dos repositorios de código con dos etapas, uno resuelto por respuesta y otro por la política
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el registro contiene, por cada repositorio, su modo y su origen

#### Scenario: Un registro sin modo de promoción sigue siendo válido
- **DADO** un registro escrito antes de esta capacidad, sin los campos del modo
- **CUANDO** se ejecutan `status` y `ensure --change x`
- **ENTONCES** ambas órdenes lo leen sin error y no lo consideran corrupto

#### Scenario: `ensure` conserva el modo registrado al reescribir el registro
- **DADO** un cambio con el modo `branch` registrado para `repo-backend`
- **CUANDO** se ejecuta `ensure --change x --archive` o de nuevo `ensure --change x`
- **ENTONCES** el modo de `repo-backend` sigue siendo `branch` en el registro

#### Scenario: El primer `ensure` que crea la rama congela su nombre
- **DADO** `repositories[].branch_template: "feature/TASK-{task_ref}"` y la Task `1235` del rol `frontend`
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** el registro guarda el nombre `feature/TASK-1235` para esa rama

#### Scenario: Un cambio de la plantilla no renombra una rama ya creada
- **DADO** la rama `feature/TASK-1235` creada y registrada, y después `axiom.yaml` con `repositories[].branch_template: "feat/{task_ref}"`
- **CUANDO** se ejecuta de nuevo `ensure --change x --role frontend`
- **ENTONCES** el plan usa `feature/TASK-1235`, la lee del registro y no crea `feat/1235`

#### Scenario: Registro borrado con ramas de plantilla
- **DADO** el cambio anterior con la rama `feature/TASK-1235` creada y el registro borrado
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** la rama se reutiliza sin modificarla y el registro se reconstruye con el nombre `feature/TASK-1235`

#### Scenario: Task revinculada y registro borrado: limitación declarada
- **DADO** la rama `feature/TASK-1235` creada y registrada, la Task del rol `frontend` revinculada después a `1300` en `integrations.yaml` y el registro borrado
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** el nombre se renderiza con el vínculo actual (`feature/TASK-1300`) y no se renombra ni se borra la rama `feature/TASK-1235`
- **Y** la continuación ofrece `--branch feature/TASK-1235` o restaurar el vínculo anterior

#### Scenario: La rama pendiente no figura en el registro
- **DADO** una rama pendiente por `{task_ref}` y un `ensure --change x` sin `--role`
- **CUANDO** termina
- **ENTONCES** el registro no contiene ninguna entrada para esa rama

### Requirement: `status`: Estado de Ramas y Worktrees del Cambio (RF-BRA-16)

`axiom branch status` DEBE informar, para cada repositorio del cambio, su papel (repositorio con rama de cambio, repositorio de specs in situ o repositorio de código), la rama esperada y la rama actual, la ruta absoluta del worktree si existe, si el árbol tiene cambios sin confirmar, el modo de trabajo materializado, la base, y el número de commits que la rama lleva de atraso o de adelanto respecto de su base calculado solo con referencias locales, y la rama destino propuesta para el PR (RF-BRA-04). En un repositorio con dos etapas, la base es la base efectiva de su modo de promoción (RF-BRA-29) y DEBE informar además de ese modo, de la rama principal y, en modo `branch` con la rama del cambio ya fusionada en la integración, de la siguiente PR con destino la rama principal (RF-BRA-30). Cuando existe la rama de archivado, DEBE incluirla con los mismos datos. `--change` es opcional: sin él, `status` lista los cambios registrados y los cambios pendientes de archivar (RF-BRA-21); con él, además informa de si el cambio solicitado está pendiente de archivar. NUNCA DEBE realizar operaciones remotas. DEBE poder ejecutarse desde el checkout principal, desde cualquier worktree del cambio y con `--cwd`. Un cambio sin ramas creadas DEBE informarse como no iniciado (`not_started`), no como error. DEBE listar además las ramas pendientes por `{task_ref}` (prov:BRA:5), con su repositorio, su rol y su plantilla, en un campo aditivo del JSON; una rama pendiente no tiene rama esperada ni actual, y se lista con la orden que la crea (prov:BRA:5). Con `integrations.yaml` ilegible y plugin activo, `status` informa el código del rechazo en la entrada afectada sin fallar la orden entera.
(Previously: `status` no listaba ramas pendientes)
(Previously: `status` no informaba del modo de promoción ni de la siguiente PR, y la base era siempre la de integración)

#### Scenario: Estado de un cambio con rama y worktree
- **DADO** un cambio preparado por `ensure` en modo `worktree`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON incluye, por repositorio, la rama esperada, la rama actual, la ruta absoluta del worktree y si el árbol está limpio

#### Scenario: Atraso calculado solo con referencias locales
- **DADO** una rama `feat/x` con 3 commits menos que `main` y sin acceso a la red
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON informa de 3 commits de atraso respecto de la base
- **Y** no se ejecuta ninguna operación remota

#### Scenario: Cambio no iniciado
- **DADO** un nombre de cambio sin rama ni worktree
- **CUANDO** se ejecuta `axiom branch status --change nuevo --json`
- **ENTONCES** el estado es el token `not_started` con la etiqueta `no iniciado` y el código de salida es `0`

#### Scenario: Rama destino propuesta para el PR
- **DADO** un cambio con las ramas de rol `feat/x-core` y `feat/x-qa` y seguimiento `feat/x`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el destino propuesto del PR de cada rama de rol es `feat/x` y el de `feat/x` es la rama de integración

#### Scenario: La rama de archivado figura en el plan
- **DADO** un cambio con la rama de archivado `feat/x-archive` y su worktree
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON incluye la rama `feat/x-archive` con la ruta absoluta de su worktree
- **Y** su destino propuesto de PR es la rama de integración

#### Scenario: `status` sin `--change` lista registrados y pendientes
- **DADO** un proyecto con el cambio `x` registrado y el cambio `y` fusionado y pendiente de archivar
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** el JSON lista `x` entre los cambios registrados y `y` entre los pendientes de archivar

#### Scenario: La base y el modo de un repositorio con dos etapas figuran en el estado
- **DADO** un cambio en modo `branch` con `main_branch: master` y `integration_branch: develop`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** la base del repositorio es `master`
- **Y** el JSON informa del modo `branch` y de la rama principal `master`, además de los campos ya existentes


#### Scenario: `status` lista las ramas pendientes
- **DADO** un multirepo sellado con `repositories[].branch_template: "feature/TASK-{task_ref}"` para `repo-frontend` y ningún `ensure --role frontend` todavía
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON lista una rama pendiente con el repositorio `repo-frontend`, el rol `frontend` y la plantilla
- **Y** esa rama no figura con rama esperada ni actual

#### Scenario: Sin ramas pendientes el JSON de `status` no cambia
- **DADO** un cambio sin plantillas con `{task_ref}`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON es idéntico al de antes de esta capacidad y no incluye ramas pendientes

### Requirement: Veredicto Único de Rama, por Papel del Repositorio (RF-BRA-19)

Las guardas A (RF-BRA-20), B (`blocked(wrong_branch)` de `axiom-sdd-cli-integration`) y D (`guard`, RF-BRA-22) DEBEN consultar un único veredicto: «¿esta rama es la que corresponde a este cambio en este repositorio según su papel?». El veredicto DEBE ser uno de los tokens `correct`, `wrong_branch` o `not_applicable` (RF-BRA-02). La consulta se hace en uno de dos contextos: el **ordinario** y el de **archivado** (la fase archive de `axiom sdd status` y el escritor de `axiom sdd archive-compose`). Las reglas son:

- **Repositorio con rama de cambio, contexto ordinario**: es correcta la rama del cambio o, si existen ramas de rol de ese cambio, cualquiera de ellas; cualquier otra rama (incluida la de integración, la de archivado y un HEAD desacoplado) es `wrong_branch`.
- **Repositorio con rama de cambio, contexto de archivado**: es correcta únicamente la rama de archivado `<prefijo>/<cambio>-archive`; cualquier otra (la del cambio, la de integración, una de rol o una ajena) es `wrong_branch`, y su continuación DEBE ser `axiom branch ensure --cwd <ruta> --change <c> --archive`.
- **Repositorio de specs en multirepo**: es correcta únicamente la rama de integración, también en el contexto de archivado (no hay rama de archivado); una rama de rol del cambio o cualquier otra es `wrong_branch`.
- **Repositorio de código en multirepo**: el veredicto solo se evalúa cuando el consultante es apply (el estado de aplicación de `sdd status` o la guarda D sobre una ruta de ese repositorio); en otras fases es `not_applicable`.
- **Exenciones**: es `not_applicable` para un cambio cuyo `kickoff.yaml` no contiene instantánea (RF-KOF-07), para los sumideros `e2e-cumulative` y `cumulative-<rol>` por nombre, y cuando la ruta no pertenece a ningún repositorio git o git no está disponible (en esos casos la consulta NO DEBE bloquear y DEBE advertirlo).

Los nombres esperados DEBEN obtenerse de la instantánea del `kickoff.yaml` cuando existe (RF-KOF-06) y, para las ramas nombradas por plantilla cuyo nombre no está en la instantánea (las que usan `{task_ref}`), del nombre congelado en el registro local (RF-BRA-15) o, si el registro no lo tiene, de renderizar la plantilla de la instantánea con los vínculos versionados; si el cambio aún no tiene kickoff (se está creando), de la política de `axiom.yaml` (RF-BRA-03). El veredicto NUNCA DEBE recomponer el nombre como `<prefijo>/<cambio>` cuando el cambio usa plantilla, de modo que una rama nombrada por plantilla no se marca como `wrong_branch`. Sin registro, el nombre esperado se obtiene igual que en el plan: el de la instantánea, si no el del registro, si no el renderizado de la plantilla de la instantánea con los vínculos versionados; para las ramas de rol pendientes, el de la plantilla de rol con el `{task_ref}` de ese rol. Un repositorio sin ninguna rama creada y con todas sus ramas pendientes (prov:BRA:5) es `not_applicable` con `Details.reason=awaiting_role`, los roles pendientes y la continuación `axiom branch ensure --cwd <ruta> --change <c> --role <rol>`; si algún rol ya entró, el veredicto se calcula sobre las ramas creadas y las pendientes se informan aparte; el nombre de la rama de archivado se toma del registro local si `ensure --archive` recibió `--prefix` (RF-BRA-12). Un `wrong_branch` DEBE incluir la rama actual, la esperada, el repositorio y la continuación exacta: la orden `axiom branch ensure` con `--cwd`, `--change` y, si procede, `--lane` o `--archive`. El veredicto NO consulta si la rama de integración contiene cambios activos: con el archive en un PR documental aparte es normal que los contenga temporalmente (RF-BRA-21).
(Previously: lo consultaban cuatro capas, incluida una de CI; el veredicto no distinguía el contexto de archivado y los valores eran etiquetas en castellano)
(Previously: los nombres esperados salían de la instantánea o de `<prefijo>/<cambio>` y no leían el registro ni conocían las ramas pendientes)

#### Scenario: Rama del cambio en un monorepo es correcta
- **DADO** un cambio con instantánea y el repositorio del planning home desplegado en `feat/x`
- **CUANDO** se consulta el veredicto en el contexto ordinario
- **ENTONCES** es `correct`

#### Scenario: La rama de integración es incorrecta para un cambio en monorepo
- **DADO** un cambio con instantánea y el repositorio del planning home desplegado en `main`
- **CUANDO** se consulta el veredicto en el contexto ordinario
- **ENTONCES** es `wrong_branch` con rama actual `main` y rama esperada `feat/x`
- **Y** la continuación es la orden exacta de `axiom branch ensure --cwd <ruta> --change x`

#### Scenario: Una rama de rol es correcta cuando hay ramas de rol
- **DADO** un multirepo con los roles `core` y `qa` sobre el mismo repositorio de código y ese repositorio desplegado en `feat/x-qa`
- **CUANDO** se consulta el veredicto en el contexto ordinario
- **ENTONCES** es `correct`

#### Scenario: La rama de archivado es incorrecta en el contexto ordinario
- **DADO** un cambio con instantánea y el repositorio desplegado en `feat/x-archive`
- **CUANDO** se consulta el veredicto en el contexto ordinario
- **ENTONCES** es `wrong_branch` con rama esperada `feat/x`

#### Scenario: En el contexto de archivado solo la rama de archivado es correcta
- **DADO** un cambio con instantánea y el repositorio desplegado en `feat/x-archive`
- **CUANDO** se consulta el veredicto en el contexto de archivado
- **ENTONCES** es `correct`

#### Scenario: En el contexto de archivado la rama del cambio, la de integración y una ajena son incorrectas
- **DADO** un monorepo con instantánea y el repositorio desplegado, por separado, en `feat/x`, en `main` y en una rama ajena `feat/otra`
- **CUANDO** se consulta el veredicto en el contexto de archivado
- **ENTONCES** es `wrong_branch` en los tres casos con rama esperada `feat/x-archive`
- **Y** la continuación es `axiom branch ensure --cwd <ruta> --change x --archive`

#### Scenario: Un HEAD desacoplado es incorrecto
- **DADO** un cambio con instantánea y el repositorio con HEAD desacoplado
- **CUANDO** se consulta el veredicto
- **ENTONCES** es `wrong_branch` y el detalle indica que el HEAD está desacoplado

#### Scenario: El repositorio de specs de un multirepo exige su rama de integración
- **DADO** un multirepo con un cambio con instantánea y el repositorio de specs desplegado en `feat/x-core`
- **CUANDO** se consulta el veredicto
- **ENTONCES** es `wrong_branch` con rama esperada `main`

#### Scenario: El repositorio de código de un multirepo solo se evalúa en apply
- **DADO** un multirepo con `repo-backend` desplegado en `main` y un cambio con instantánea
- **CUANDO** se consulta el veredicto desde la fase `spec`
- **ENTONCES** es `not_applicable`
- **Y** desde la fase de aplicación de una tarea que toca `repo-backend` es `wrong_branch`

#### Scenario: Un cambio preexistente sin instantánea queda exento
- **DADO** un cambio con `kickoff.yaml` sin instantánea y el repositorio desplegado en `main`
- **CUANDO** se consulta el veredicto
- **ENTONCES** es `not_applicable`

#### Scenario: Los cambios activos en la rama de integración no alteran el veredicto
- **DADO** un cambio `y` fusionado y presente en el árbol de la rama de integración, y un cambio `x` con instantánea en `feat/x`
- **CUANDO** se consulta el veredicto de `x` desplegado en `feat/x`
- **ENTONCES** es `correct`

#### Scenario: Fuera de un repositorio git no se bloquea
- **DADO** un directorio de proyecto que no pertenece a ningún repositorio git
- **CUANDO** se consulta el veredicto
- **ENTONCES** no se bloquea y se emite un aviso



#### Scenario: Una rama nombrada por plantilla es correcta
- **DADO** un cambio con instantánea cuya rama del cambio, nombrada por plantilla, es `inc/2525-alta-de-usuarios`, y el repositorio desplegado en ella
- **CUANDO** se consulta el veredicto en el contexto ordinario
- **ENTONCES** es `correct`

#### Scenario: Una rama con `{task_ref}` se reconoce por el registro
- **DADO** la rama `feature/TASK-1235` creada por `ensure --role frontend`, registrada, con la instantánea que solo guarda su plantilla, y el repositorio desplegado en ella
- **CUANDO** se consulta el veredicto en la fase de aplicación
- **ENTONCES** es `correct`

#### Scenario: La rama de plantilla se reconoce sin registro
- **DADO** el mismo caso con el registro local borrado
- **CUANDO** se consulta el veredicto
- **ENTONCES** es `correct`, porque el nombre se obtiene de renderizar la plantilla de la instantánea con los vínculos versionados

#### Scenario: Una rama pendiente es `not_applicable`
- **DADO** un repositorio de código con su rama pendiente por `{task_ref}` y desplegado en `main`
- **CUANDO** se consulta el veredicto desde la aplicación de una tarea de otro rol
- **ENTONCES** es `not_applicable`
- **Y** el detalle indica `awaiting_role`, nombra el rol pendiente `frontend` y la continuación es `axiom branch ensure --cwd <ruta> --change x --role frontend`

#### Scenario: Con un rol ya dentro el veredicto se calcula sobre las ramas creadas
- **DADO** un repositorio compartido por `core` (rama creada, desplegado en ella) y `qa` (rama pendiente por `{task_ref}`)
- **CUANDO** se consulta el veredicto en la fase de aplicación
- **ENTONCES** es `correct`
- **Y** la rama pendiente de `qa` se informa aparte y no cambia el veredicto

### Requirement: `ensure --role` Validado contra el Roster Sellado (RF-BRA-33)

`axiom branch ensure` DEBE aceptar la bandera opcional `--role <id>`. Solo es válida en el carril `sdd`, es incompatible con `--archive` y DEBE referirse a un rol presente en el kickoff sellado del cambio. En caso contrario DEBE rechazar con un error tipado y código distinto de cero ANTES de mutar nada (ni ramas, ni worktrees, ni registro), con los códigos `plan_conflict` (con `--archive`), `role_not_applicable` (fuera del carril `sdd`), `kickoff_not_sealed` (sin kickoff sellado) y `role_not_in_roster` (rol fuera del roster, con el detalle de los roles sellados); un `--role` en blanco es error de uso (salida 2). Además, cuando el cambio tiene un vínculo `kind: work_item` en `integrations.yaml` y hay activo un plugin con la capacidad `items.create` (RF-INT-02), DEBE rechazar con el código tipado `task_not_bound` si el rol no tiene vínculo `kind: task`, también ANTES de mutar nada y tras las comprobaciones anteriores; la resolución DEBE nombrar `axiom integrations tasks create --change <cambio>` y `axiom integrations tasks bind --change <cambio> --role <rol>=<id>`. Un cambio sin vínculo `kind: work_item`, o sin plugin activo con `items.create`, NO DEBE recibir `task_not_bound`. Sin `--role`, el comportamiento y el JSON de `ensure` NO DEBEN cambiar. El motor de `ensure` DEBE recibir el rol de `--role` únicamente para crear las ramas pendientes de ese rol (prov:BRA:5). Con `--role`, el plan, las ramas creadas y el JSON del motor NO DEBEN diferir de los de la misma invocación sin la bandera, salvo en las ramas de ese rol cuya plantilla usa `{task_ref}` y que estaban pendientes: esas se crean y dejan de listarse como pendientes. Sin plantillas con `{task_ref}`, la invariancia se mantiene íntegra.

(Placement: after RF-BRA-27)

(Previously: sin la comprobación `task_not_bound`; el rechazo del rol solo dependía del carril, de `--archive` y del roster sellado)
(Previously: el motor no conocía el rol y, con `--role`, el plan, las ramas y el JSON nunca diferían de los de la invocación sin la bandera)

#### Scenario: Rol sellado
- **DADO** un kickoff sellado con el rol `backend` y sin plantillas con `{task_ref}`
- **CUANDO** se ejecuta `axiom branch ensure --change x --role backend`
- **ENTONCES** `ensure` termina con estado `ready` y el mismo JSON que sin `--role`

#### Scenario: Rol ausente del kickoff
- **DADO** un kickoff sellado con el rol `backend`
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** se rechaza con un error tipado que nombra los roles sellados
- **Y** no se crea ninguna rama, worktree ni registro

#### Scenario: Incompatible con `--archive`
- **CUANDO** se ejecuta `ensure --change x --archive --role backend`
- **ENTONCES** se rechaza con un error tipado y no se muta nada

#### Scenario: Fuera del carril `sdd`
- **CUANDO** se ejecuta `ensure --lane odd --change f --role backend`
- **ENTONCES** se rechaza con un error tipado y no se muta nada

#### Scenario: Sin kickoff sellado
- **DADO** un cambio sin `kickoff.yaml`
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** se rechaza con un error tipado que indica que el cambio no tiene roles sellados y no se muta nada

#### Scenario: El motor recibe el rol solo para las ramas pendientes
- **DADO** el rol `frontend` con una rama pendiente por `{task_ref}` y el rol `backend` sin plantilla con `{task_ref}`
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** se crea la rama pendiente de `frontend`
- **Y** el plan de las demás ramas es idéntico al de `ensure --change x` sin la bandera

#### Scenario: Con `--role` el plan solo difiere en las ramas pendientes de ese rol
- **DADO** un plan con la rama pendiente de `frontend` y la de `backend`
- **CUANDO** se comparan `ensure --change x` y `ensure --change x --role frontend`
- **ENTONCES** la única diferencia es la creación de la rama de `frontend`
- **Y** la rama de `backend` sigue pendiente

#### Scenario: Rol sin Task vinculada con tracker activo
- **DADO** un cambio con vínculo `kind: work_item`, un plugin activo con `items.create` y el rol `backend` sin vínculo `kind: task`
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** se rechaza con `task_not_bound` y código distinto de cero
- **Y** la resolución nombra `axiom integrations tasks create --change x` y `axiom integrations tasks bind --change x --role backend=<id>`
- **Y** no se crea ninguna rama, worktree ni registro y el doble no recibe ningún evento

#### Scenario: Con la Task vinculada la barrera se levanta
- **DADO** el rol `backend` con vínculo `kind: task`
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** `ensure` termina con estado `ready` y el mismo JSON que sin `--role`

#### Scenario: El rechazo de roster precede a la barrera
- **DADO** un rol fuera del roster sellado en un cambio con tracker activo
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** el código del rechazo es `role_not_in_roster`, no `task_not_bound`

#### Scenario: Cambio sin vínculo de WI
- **DADO** un cambio sin vínculo `kind: work_item` y un plugin activo
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** no se rechaza por `task_not_bound`

#### Scenario: Sin plugin activo con `items.create`
- **DADO** un cambio con vínculo `kind: work_item` y sin plugin activo (por ejemplo con `integrations.azure-devops.enabled: false`)
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** la salida y el JSON coinciden con los de antes de este cambio y no se rechaza por `task_not_bound`

## Registro de remediación

2026-11-26: cobertura ronda 1; se cierran S1 a S12 y las decisiones G2, G4, G8, G9, G10 y G11: `{slug}` (prefijo, mayúsculas, nunca vacío), segundo clon con `--role`, limitación declarada de la Task revinculada, `--branch`/`--prefix`, rechazo de rama compartida en seal/ensure/ODD/validate, nombres reservados, `status` sin mutar, origen de valores en ODD, archivado con rama resuelta, `awaiting_role`, fallo cerrado con `{slug}`/`{kind}`/`{task_ref}`, precedencia por rama y colisión sobre nombres registrados; en `sdd-lane-kickoff` y `workspace-topology` se ajustan kind siempre congelado, cambio sellado sin `kind`, plantilla con variable propia del cambio, repositorio de specs en multirepo, claves de la API de ajustes y ejemplos de `axiom.example.yaml`.

