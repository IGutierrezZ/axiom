# Delta for workspace-topology

## ADDED Requirements

### Requirement: Validación de `branch_template` al Cargar `axiom.yaml` (RF-TOP-24)

La plantilla de nombre de rama (`git.branch_template`, `repositories[].branch_template` y `roles.<id>.branch_template`) es un escalar de texto opcional con las variables `{prefix}`, `{kind}`, `{change}`, `{slug}`, `{role}`, `{ref}` y `{task_ref}`, y su significado se define en `change-branch-lifecycle` (RF-BRA-42 a RF-BRA-44). Al cargar `axiom.yaml` DEBE validarse, con un error que nombre la clave (y, en una entrada de `repositories:`, su ruta; en un rol, su identificador):

- que las llaves están bien formadas (cada `{` cierra con `}` sin anidar ni dejar llaves sueltas) y que toda variable es una de las siete conocidas;
- que la plantilla efectiva de la rama que lleva `openspec/` no usa `{role}` ni `{task_ref}`: la plantilla de `git:` en `monorepo-embedded`, y en `multirepo` la entrada de `repositories:` del repositorio de specs si existe y la declara, o si no la de `git:`; las plantillas de rol nunca se aplican a esa rama;
- que contiene al menos una de `{change}`, `{slug}`, `{ref}` o `{task_ref}`, para que no haya nombres constantes compartidos entre cambios;
- que el nombre, renderizado con valores de muestra seguros para git, es un nombre de referencia git válido.

En `multirepo`, la entrada de `repositories:` del repositorio de specs se identifica por el `path` declarado que coincide con el repositorio de specs de la topología. Su plantilla efectiva se valida igual aunque ese repositorio trabaje in situ (RF-BRA-05 de `change-branch-lifecycle`). Un `git.branch_template` con `{task_ref}` es válido en `multirepo` solo si el repositorio de specs declara su propia plantilla.

Una plantilla inválida DEBE reportarse como error de configuración (`config_invalid` al ejecutar `ensure`, y salida `1` en `axiom workspace validate`), acumulándose con los demás errores, sin crear nada. En `monorepo-embedded`, una `roles.<id>.branch_template` NO tiene efecto, porque la única rama es la del cambio y lleva `openspec/`; `axiom workspace validate` DEBE emitir una advertencia no bloqueante que la nombre, sin cambiar el estado `COMPLIANT` ni el código de salida (mismo precedente que `promotion` sin `main_branch`). Una plantilla vacía o ausente significa que no hay plantilla en ese nivel.

#### Scenario: Plantilla válida en los tres niveles
- **DADO** un multirepo con `git.branch_template: "{prefix}/{change}"`, `repositories: [{path: repo-backend, branch_template: "feature/{ref}-{slug}"}]` y `roles.backend.branch_template: "feature/TASK-{task_ref}"`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** el estado es `COMPLIANT` y el código de salida es `0`

#### Scenario: Variable desconocida
- **DADO** `git.branch_template: "{prefix}/{ticket}"`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `git.branch_template`, la variable `ticket` y enumera las siete variables admitidas
- **Y** el código de salida es `1`

#### Scenario: Llaves mal formadas
- **DADO** `repositories: [{path: repo-backend, branch_template: "feature/{ref"}]`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `repositories[].branch_template`, la ruta `repo-backend` y la llave sin cerrar
- **Y** el código de salida es `1`

#### Scenario: `{task_ref}` en la plantilla de la rama con `openspec/` en monorepo
- **DADO** un `monorepo-embedded` con `git.branch_template: "feature/{task_ref}"`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `git.branch_template` y explica que esa rama lleva `openspec/` y no puede usar `{task_ref}`
- **Y** el código de salida es `1`

#### Scenario: `{role}` en la plantilla del repositorio de specs de un multirepo
- **DADO** un multirepo con `git.branch_template: "{prefix}/{role}/{change}"` y sin entrada de `repositories:` para el repositorio de specs
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `git.branch_template` y la rama con `openspec/`
- **Y** el código de salida es `1`

#### Scenario: `{task_ref}` en `git:` es válido si el repositorio de specs declara su propia plantilla
- **DADO** un multirepo con `git.branch_template: "feature/{task_ref}"` y una entrada de `repositories:` para el repositorio de specs con `branch_template: "{prefix}/{change}"`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** el estado es `COMPLIANT`

#### Scenario: `{task_ref}` en `git:` de un multirepo sin plantilla propia del repositorio de specs
- **DADO** un multirepo con `git.branch_template: "feature/{task_ref}"` y sin entrada de `repositories:` para el repositorio de specs
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `git.branch_template` y la rama con `openspec/`
- **Y** el código de salida es `1`

#### Scenario: La plantilla del repositorio de specs se valida aunque trabaje in situ
- **DADO** un multirepo con el repositorio de specs trabajando in situ y una entrada de `repositories:` para él con `branch_template: "{prefix}/{role}/{change}"`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `repositories[].branch_template`, la ruta del repositorio de specs y la rama con `openspec/`
- **Y** el código de salida es `1`

#### Scenario: Una plantilla sin ninguna variable propia del cambio se rechaza
- **DADO** `git.branch_template: "{prefix}/trabajo"`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `git.branch_template` y enumera `{change}`, `{slug}`, `{ref}` y `{task_ref}`
- **Y** el código de salida es `1` y `ensure` rechazaría con `config_invalid`

#### Scenario: Nombre renderizado inválido
- **DADO** `git.branch_template: "{prefix}/{change}.lock"`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `git.branch_template` y la regla de nombre de referencia incumplida
- **Y** el código de salida es `1`

#### Scenario: Varios errores de plantilla se acumulan
- **DADO** `git.branch_template: "{x}"` y `roles.backend.branch_template: "{y"`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida reporta ambos errores, uno por clave

#### Scenario: `roles.<id>.branch_template` en monorepo solo avisa
- **DADO** un `monorepo-embedded` con el rol `fullstack` y `roles.fullstack.branch_template: "feature/{task_ref}"`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida incluye una advertencia que nombra `roles.fullstack.branch_template` y explica que no tiene efecto en esta topología
- **Y** el estado sigue siendo `COMPLIANT` y el código de salida es `0`

## MODIFIED Requirements

### Requirement: Estructura del Esquema axiom.yaml (RF-TOP-03)

El motor de configuración DEBE deserializar y validar el archivo `axiom.yaml`, exigiendo los campos obligatorios:
- `workspace.name` (cadena no vacía)
- `workspace.topology` (uno de: `monorepo-embedded`, `multirepo`)
- `workspace.specs_repository` (ruta relativa al directorio de especificaciones)
- `roles` (mapa con al menos 1 rol definido, conteniendo `name` y una lista de `repositories`, y, opcionalmente, `branch_template`: la plantilla de nombre de rama del rol, que prevalece sobre la del repositorio y la de `git:` y se valida al cargar según RF-TOP-24; las demás claves desconocidas de un rol se siguen ignorando)

Además DEBE aceptar los bloques opcionales de primer nivel `technical_context` (RF-CTX-01), `git` (RF-TOP-09), `repositories` (RF-TOP-10), `workflow` (RF-TOP-11) y `delivery` (RF-TOP-14): su ausencia no altera la validación y, cuando están presentes, se validan de forma estricta (claves desconocidas, rutas y valores). Un bloque `baseline` NO es un bloque admitido: se rechaza con el error tipado de RF-TOP-12. Las claves desconocidas fuera de esos bloques se siguen ignorando.
(Previously: la lista de bloques opcionales de primer nivel era `technical_context`, `git`, `repositories` y `workflow`)
(Previously: la entrada de cada rol contenía solo `name` y `repositories`)

#### Scenario: El bloque `workflow` es opcional y se valida cuando está presente

- **DADO** un `axiom.yaml` con los campos obligatorios y, en una variante, además `workflow: {lane: ask}`
- **CUANDO** el usuario ejecuta `axiom workspace validate` sobre ambas variantes
- **ENTONCES** las dos son `COMPLIANT`
- **Y** con `workflow: {lane: x}` la validación falla nombrando `workflow.lane`

#### Scenario: El bloque `delivery` es opcional y se valida cuando está presente

- **DADO** un `axiom.yaml` con los campos obligatorios y, en una variante, además `delivery: {split_hint_lines: 600}`
- **CUANDO** el usuario ejecuta `axiom workspace validate` sobre ambas variantes
- **ENTONCES** las dos son `COMPLIANT`
- **Y** con `delivery: {split_hint_lines: 0}` la validación falla nombrando `delivery.split_hint_lines`

#### Scenario: El bloque `baseline` ya no es un bloque admitido

- **DADO** un `axiom.yaml` con los campos obligatorios y un bloque `baseline:`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la validación falla con el error tipado de RF-TOP-12 y no lo trata como clave desconocida ignorable

#### Scenario: Un rol admite `branch_template` y sigue ignorando otras claves

- **DADO** un `axiom.yaml` con un rol `backend` con `branch_template: "feature/{task_ref}"` y una clave desconocida `color: rojo`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la plantilla del rol queda cargada como `roles.backend.branch_template`
- **Y** la clave `color` se ignora sin error

#### Scenario: Un rol sin `branch_template` no cambia

- **DADO** un `axiom.yaml` cuyos roles solo tienen `name` y `repositories`
- **CUANDO** se carga la configuración
- **ENTONCES** ningún rol declara plantilla y la validación es la de antes

### Requirement: Bloque `git:` Opcional de `axiom.yaml` (RF-TOP-09)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `git:` con las claves opcionales `integration_branch`, `branch_prefix`, `main_branch`, `promotion` y `branch_template`. Sin el bloque, o sin una de las claves, `integration_branch` vale `main` y `branch_prefix` vale `feat`; `main_branch` NO tiene valor por defecto (su ausencia significa que la rama de integración es la única etapa y la rama principal no existe como concepto distinto) y `promotion` NO tiene valor por defecto declarado: cuando hay dos etapas y no se declara, el modo de promoción efectivo es `branch` (RF-BRA-28 de `change-branch-lifecycle`). `integration_branch`, `branch_prefix` y `main_branch` DEBEN ser cadenas no vacías que formen nombres de referencia git válidos (`branch_prefix` sin `/` inicial ni final, sin `..` ni espacios). `promotion` DEBE ser una de las cadenas `branch`, `release` o `ask`; `ask` es válido en la configuración y significa que la persona decide el modo al crear la rama de cada cambio. Una clave desconocida dentro de `git:` y un valor inválido DEBEN reportarse como error nombrando la clave, y el mensaje que enumera las claves admitidas DEBE listar las cinco. `branch_template` NO tiene valor por defecto (su ausencia en todos los niveles significa los nombres de rama de siempre), es una cadena que DEBE cumplir RF-TOP-24 y es la plantilla de nombre de rama de menor precedencia, tras la del rol y la del repositorio (RF-BRA-44 de `change-branch-lifecycle`). `axiom.example.yaml` DEBE incluir ejemplos comentados de `branch_template` (en `git:`, en una entrada de `repositories:` y en un rol) que validen al descomentarse, comprobado por `TestExampleGitBlockIsValidOnceUncommented`, y el bloque `git:` DEBE documentarse con sus cinco claves. Los valores del bloque son política del proyecto y se versionan: el modo de trabajo y la raíz de los worktrees NO DEBEN declararse en `axiom.yaml` (viven en el hub personal, capacidad `multi-project-hub`).

`promotion` sin `main_branch`, o con `main_branch` igual a la rama de integración resuelta, NO tiene efecto: `axiom workspace validate` DEBE emitir una advertencia no bloqueante que lo indique, sin cambiar el estado `COMPLIANT` ni el código de salida.
(Previously: `git:` admitía solo `integration_branch` y `branch_prefix`)
(Previously: `git:` admitía cuatro claves, sin `branch_template`)

#### Scenario: Sin bloque rigen los valores por defecto
- **DADO** un `axiom.yaml` sin `git:`
- **CUANDO** se resuelve la política de ramas
- **ENTONCES** la rama de integración es `main` y el prefijo es `feat`
- **Y** no hay rama principal

#### Scenario: Bloque parcial completa con los valores por defecto
- **DADO** un `axiom.yaml` con `git: {integration_branch: develop}`
- **CUANDO** se resuelve la política de ramas
- **ENTONCES** la rama de integración es `develop` y el prefijo es `feat`
- **Y** no hay rama principal ni modo de promoción declarado

#### Scenario: Valor inválido
- **DADO** un `axiom.yaml` con `git.branch_prefix: "inc//"`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `git.branch_prefix` y la regla incumplida
- **Y** el código de salida es `1`

#### Scenario: Clave desconocida en `git:`
- **DADO** un `axiom.yaml` con `git: {worktree_mode: worktree}`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra la clave `worktree_mode` como desconocida y enumera `integration_branch`, `branch_prefix`, `main_branch`, `promotion` y `branch_template` como admitidas
- **Y** el código de salida es `1`

#### Scenario: `main_branch` y `promotion` válidos
- **DADO** un `axiom.yaml` con `git: {integration_branch: develop, main_branch: master, promotion: ask}`
- **CUANDO** el usuario ejecuta `axiom workspace validate` y se resuelve la política de ramas
- **ENTONCES** el estado es `COMPLIANT`
- **Y** la rama principal es `master`, la de integración es `develop` y el modo de promoción declarado es `ask`

#### Scenario: `main_branch` con un nombre de referencia inválido
- **DADO** un `axiom.yaml` con `git.main_branch: "mi rama"`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `git.main_branch` y la regla incumplida
- **Y** el código de salida es `1`

#### Scenario: `promotion` con un valor fuera del vocabulario
- **DADO** un `axiom.yaml` con `git.promotion: merge`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `git.promotion`, el valor `merge` y los valores admitidos `branch`, `release` y `ask`
- **Y** el código de salida es `1`

#### Scenario: Varios errores se acumulan
- **DADO** un `axiom.yaml` con `git.main_branch: "a b"` y `git.promotion: merge`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida reporta ambos errores, uno por clave
- **Y** el código de salida es `1`

#### Scenario: `promotion` sin `main_branch` solo avisa
- **DADO** un `axiom.yaml` con `git: {promotion: release}` y sin `main_branch`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida incluye una advertencia que nombra `git.promotion` y explica que no tiene efecto sin `git.main_branch`
- **Y** el estado sigue siendo `COMPLIANT` y el código de salida es `0`

#### Scenario: `main_branch` igual a la integración solo avisa
- **DADO** un `axiom.yaml` con `git: {integration_branch: main, main_branch: main, promotion: branch}`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida incluye una advertencia que indica que `promotion` no tiene efecto porque `main_branch` coincide con la rama de integración
- **Y** el estado sigue siendo `COMPLIANT` y el código de salida es `0`


#### Scenario: `branch_template` válido en `git:`
- **DADO** un `axiom.yaml` con `git: {branch_prefix: inc, branch_template: "{prefix}/{ref}-{slug}"}`
- **CUANDO** el usuario ejecuta `axiom workspace validate` y se resuelve la política de ramas
- **ENTONCES** el estado es `COMPLIANT` y la política declara esa plantilla

#### Scenario: Los ejemplos comentados de `axiom.example.yaml` validan al descomentarse
- **DADO** `axiom.example.yaml` con los ejemplos comentados de `branch_template`
- **CUANDO** se descomentan y se ejecuta `TestExampleGitBlockIsValidOnceUncommented`
- **ENTONCES** el resultado es válido y `axiom workspace validate` lo declara `COMPLIANT`

#### Scenario: Sin `branch_template` rige el nombre de siempre
- **DADO** un `axiom.yaml` con `git: {branch_prefix: inc}`
- **CUANDO** se resuelve la política de ramas
- **ENTONCES** la política no declara plantilla y la rama del cambio `x` es `inc/x`

#### Scenario: `branch_template` con una variable desconocida
- **DADO** un `axiom.yaml` con `git.branch_template: "{prefix}/{ticket}"`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `git.branch_template` y la variable `ticket`
- **Y** el código de salida es `1`

### Requirement: Bloque `repositories:` de Primer Nivel con Ajustes por Repositorio (RF-TOP-10)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `repositories:` con una lista de entradas, cada una con `path` (obligatorio, relativo a la carpeta maestra) y, opcionalmente, `integration_branch`, `branch_prefix`, `promotion` y `branch_template` propios del repositorio, con las mismas reglas de validez que `git:` (supuesto S-13): `promotion` es `branch`, `release` o `ask`, y `branch_template` cumple RF-TOP-24 y prevalece sobre `git.branch_template` para ese repositorio. `main_branch` NO es una clave de `repositories:`: la rama principal es única del proyecto y declararla en una entrada DEBE reportarse como clave desconocida. Un repositorio sin entrada, o sin una de las claves, hereda el valor del bloque `git:`, campo a campo. `path` NO DEBE ser absoluta ni salir de la carpeta maestra y DEBE existir como directorio; DOS entradas que resuelvan a la misma raíz git DEBEN rechazarse nombrando ambas. Una entrada cuyo `path` no pertenezca a ningún repositorio git NO DEBE ser un error de validación: DEBE emitir una advertencia que la nombre (RF-BRA-06). Una clave desconocida en una entrada DEBE reportarse como error, y el mensaje que enumera las claves admitidas DEBE listar `path`, `integration_branch`, `branch_prefix`, `promotion` y `branch_template`. Una entrada con `promotion` cuyo repositorio no tenga dos etapas (sin `git.main_branch` o con `git.main_branch` igual a la rama de integración resuelta de ese repositorio) DEBE emitir una advertencia no bloqueante que la nombre, sin cambiar el estado ni el código de salida. La resolución de la raíz git de cada entrada sigue RF-BRA-06.
(Previously: las entradas admitían solo `integration_branch` y `branch_prefix`)
(Previously: las entradas admitían `integration_branch`, `branch_prefix` y `promotion`, sin `branch_template`)

#### Scenario: Entrada con rama de integración propia
- **DADO** un `axiom.yaml` con `git.integration_branch: main` y `repositories: [{path: repo-backend, integration_branch: release}]`
- **CUANDO** se resuelve la política de ramas para `repo-backend` y `repo-frontend`
- **ENTONCES** `repo-backend` usa `release` y `repo-frontend` usa `main`

#### Scenario: Ruta inexistente o fuera de la carpeta maestra
- **DADO** una entrada con `path: ../fuera` y otra con una ruta que no existe
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** se reporta un error por cada una nombrando la ruta y la regla incumplida
- **Y** el código de salida es `1`

#### Scenario: Dos entradas de la misma raíz git
- **DADO** dos entradas con rutas `app/web` y `app/api` dentro del mismo repositorio git
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** se reporta un error que nombra ambas rutas y la raíz git compartida

#### Scenario: Ruta que no es un repositorio git solo avisa
- **DADO** una entrada cuyo `path` existe pero no pertenece a ningún repositorio git
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida incluye una advertencia que nombra la ruta y el resultado sigue siendo `COMPLIANT`
- **Y** el código de salida es `0`

#### Scenario: Sin bloque `repositories:` todos heredan
- **DADO** un `axiom.yaml` con `git.branch_prefix: inc` y sin `repositories:`
- **CUANDO** se resuelve la política de cualquier repositorio
- **ENTONCES** el prefijo es `inc`

#### Scenario: Entrada con modo de promoción propio
- **DADO** un `axiom.yaml` con `git.main_branch: master`, `git.promotion: branch` y `repositories: [{path: repo-backend, promotion: release}]`
- **CUANDO** se resuelve la política de ramas para `repo-backend` y `repo-frontend`
- **ENTONCES** `repo-backend` declara `release` y `repo-frontend` hereda `branch`
- **Y** ambos comparten la rama principal `master`

#### Scenario: La herencia de `promotion` es campo a campo
- **DADO** un `axiom.yaml` con `git.promotion: ask` y una entrada de `repo-backend` con solo `integration_branch: release`
- **CUANDO** se resuelve la política de `repo-backend`
- **ENTONCES** `repo-backend` usa la rama de integración `release` y hereda `promotion: ask`

#### Scenario: `promotion` con un valor fuera del vocabulario en una entrada
- **DADO** una entrada con `promotion: merge`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `repositories[].promotion`, la ruta de la entrada, el valor `merge` y los valores admitidos
- **Y** el código de salida es `1`

#### Scenario: `main_branch` en una entrada es una clave desconocida
- **DADO** una entrada con `main_branch: master`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `main_branch` como clave desconocida de la entrada y enumera `path`, `integration_branch`, `branch_prefix`, `promotion` y `branch_template` como admitidas
- **Y** el código de salida es `1`

#### Scenario: `promotion` en un repositorio sin dos etapas solo avisa
- **DADO** un `axiom.yaml` sin `git.main_branch` y una entrada con `promotion: release`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida incluye una advertencia que nombra la entrada y explica que `promotion` no tiene efecto
- **Y** el estado sigue siendo `COMPLIANT` y el código de salida es `0`

#### Scenario: Entrada con plantilla de rama propia
- **DADO** un `axiom.yaml` con `git.branch_template: "{prefix}/{change}"` y `repositories: [{path: repo-frontend, branch_template: "feature/TASK-{task_ref}"}]`
- **CUANDO** se resuelve la política de ramas para `repo-frontend` y `repo-backend`
- **ENTONCES** `repo-frontend` declara `feature/TASK-{task_ref}` y `repo-backend` hereda `{prefix}/{change}`

#### Scenario: `branch_template` inválido en una entrada
- **DADO** una entrada con `branch_template: "feature/{ref"`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `repositories[].branch_template`, la ruta de la entrada y la regla incumplida
- **Y** el código de salida es `1`

### Requirement: Catálogo Único de Claves de `axiom.yaml` Derivado del Esquema, Filtrado por Topología e ISO (RF-TOP-21)

El sistema DEBE ofrecer un catálogo de las claves de `axiom.yaml` que se pueden consultar o editar, como única fuente para cualquier pantalla o cliente. Cada entrada DEBE declarar su ruta, etiqueta, descripción, tipo, valores permitidos (si el vocabulario es cerrado), si es editable, el efecto descrito de editarla y las topologías en las que aplica. Los valores permitidos de los vocabularios cerrados (`git.promotion` y `repositories[].promotion`, `workflow.lane`, `sdd.explainer_html`, `roles[].gate_policy`) DEBEN derivarse de las constantes de su paquete dueño (las del esquema de `workspace` y, para `roles[].gate_policy`, las de `multirole`), expuestas como funciones que usan a la vez el validador y el catálogo; el catálogo NO DEBE duplicar esas listas. Una clave nueva de un vocabulario cerrado DEBE aparecer en el catálogo sin editar el catálogo.

Claves editables: `workspace.name`, `git.integration_branch`, `git.branch_prefix`, `git.main_branch`, `git.promotion`, `git.branch_template`, `workflow.lane`, `delivery.split_hint_lines`, `sdd.iso`, `sdd.id_pattern`, `sdd.explainer_html`; y, solo en `multirepo`, `repositories[].integration_branch`, `repositories[].branch_prefix`, `repositories[].promotion`, `repositories[].branch_template`, `acceptance_tests.enabled`, `acceptance_tests.environments`, `roles[].gate_policy` (con `blocking`, `deferred` y `optional`) y `roles[].branch_template`. Las tres claves `branch_template` son de tipo cadena, sin vocabulario cerrado, y una escritura con un valor que incumple RF-TOP-24 (por ejemplo, por `PUT /api/settings`) DEBE rechazarse nombrando la clave, sin modificar `axiom.yaml`. `sdd.iso` y `sdd.id_pattern` DEBEN llevar un efecto descrito explícito.

Claves de solo lectura (con `editable: false`): `technical_context`, `workspace.root`, `skill_roots`, `governance.*`, `workspace.topology` y `workspace.specs_repository`. `artifact_store` NO DEBE estar en el catálogo como editable.

El catálogo NO DEBE contener `integrations.*`, ni controles de RDD, ni `commit_review` ni `docs_publication` (no son claves de `axiom.yaml`). Las claves documentadas al registrarlas, `sdd.explainer_html`, `acceptance_tests.*` y `delivery.split_hint_lines`, DEBEN estar registradas con su tipo y su vocabulario.

El catálogo para un proyecto concreto DEBE filtrarse: en `monorepo-embedded` NO DEBE incluir `roles`, `gate_policy`, `repositories` ni `acceptance_tests`; en un proyecto ISO (`sdd.iso: true`) los valores permitidos de `workflow.lane` NO DEBEN incluir `odd`.

#### Scenario: Los valores permitidos salen de las constantes del esquema
- **DADO** el catálogo construido
- **CUANDO** se leen los valores permitidos de `git.promotion`, `workflow.lane`, `sdd.explainer_html` y `roles[].gate_policy`
- **ENTONCES** son exactamente los de las constantes de su paquete dueño (`branch|release|ask`, `odd|sdd|ask`, `ask|always|never`, `blocking|deferred|optional`)
- **Y** una prueba de coherencia falla si el catálogo declara un valor que el decodificador rechaza o al revés

#### Scenario: Claves por topología en monorepo
- **DADO** un proyecto `monorepo-embedded`
- **CUANDO** se consulta el catálogo
- **ENTONCES** incluye `workspace.name`, `git.*` (con `git.branch_template`), `workflow.lane`, `delivery.split_hint_lines` y `sdd.*`
- **Y** no incluye `roles`, `gate_policy`, `repositories` (ni `repositories[].branch_template`), `roles[].branch_template` ni `acceptance_tests`

#### Scenario: Claves por topología en multirepo
- **DADO** un proyecto `multirepo`
- **CUANDO** se consulta el catálogo
- **ENTONCES** incluye `repositories[].{integration_branch,branch_prefix,promotion,branch_template}`, `acceptance_tests.{enabled,environments}`, `roles[].gate_policy` con `optional` entre sus valores y `roles[].branch_template`

#### Scenario: Claves de solo lectura
- **DADO** el catálogo de cualquier topología
- **CUANDO** se leen `technical_context`, `workspace.root`, `skill_roots`, `governance.*`, `workspace.topology` y `workspace.specs_repository`
- **ENTONCES** cada una figura con `editable: false`
- **Y** una escritura que las incluya se rechaza como clave no editable

#### Scenario: Exclusiones del catálogo
- **DADO** el catálogo de cualquier proyecto
- **CUANDO** se buscan `integrations`, `artifact_store`, `rdd`, `commit_review` y `docs_publication`
- **ENTONCES** ninguna aparece como entrada editable

#### Scenario: Claves registradas de cambios anteriores
- **DADO** el catálogo
- **CUANDO** se consultan `sdd.explainer_html`, `acceptance_tests.enabled`, `acceptance_tests.environments` y `delivery.split_hint_lines`
- **ENTONCES** cada una tiene tipo, descripción, efecto y, la primera, su vocabulario `ask|always|never`; `split_hint_lines` es entero mayor que cero

#### Scenario: ISO no ofrece `odd`
- **DADO** un proyecto con `sdd.iso: true`
- **CUANDO** se consulta el catálogo
- **ENTONCES** los valores permitidos de `workflow.lane` son `sdd` y `ask`



#### Scenario: Las tres claves `branch_template` figuran en el catálogo
- **DADO** el catálogo de un proyecto `multirepo`
- **CUANDO** se consultan `git.branch_template`, `repositories[].branch_template` y `roles[].branch_template`
- **ENTONCES** cada una es editable, de tipo cadena, con descripción y efecto descritos, sin vocabulario cerrado, y aplica a `multirepo` (la de `git:` también a `monorepo-embedded`)

#### Scenario: Una plantilla inválida enviada como ajuste se rechaza
- **DADO** el dashboard y `PUT /api/settings` con `git.branch_template: "{prefix}/{ticket}"`
- **CUANDO** se procesa la escritura
- **ENTONCES** se rechaza nombrando `git.branch_template` y la variable desconocida
- **Y** `axiom.yaml` no se modifica

#### Scenario: Una plantilla inválida de repositorio o de rol se rechaza con la ruta del elemento
- **DADO** el dashboard de un `multirepo` y `PUT /api/settings` con `repositories[].branch_template: "feature/{ref"` para `repo-backend`, y por separado `roles[].branch_template: "{y"` para `backend`
- **CUANDO** se procesa cada escritura
- **ENTONCES** cada una se rechaza nombrando su clave y la ruta del elemento (`repo-backend` o el rol `backend`)
- **Y** `axiom.yaml` no se modifica

#### Scenario: Una plantilla válida de repositorio o de rol enviada como ajuste se escribe
- **DADO** el dashboard de un `multirepo` y `PUT /api/settings` con `repositories[].branch_template: "feature/{ref}-{slug}"` para `repo-backend` y `roles[].branch_template: "feature/TASK-{task_ref}"` para `backend`
- **CUANDO** se procesa la escritura
- **ENTONCES** `axiom.yaml` contiene ambas plantillas en la entrada y en el rol indicados y vuelve a cargarse como `COMPLIANT`

#### Scenario: Una plantilla con `{role}` en la rama con `openspec/` se rechaza por la API
- **DADO** el dashboard de un `monorepo-embedded` y `PUT /api/settings` con `git.branch_template: "{prefix}/{role}/{change}"`
- **CUANDO** se procesa la escritura
- **ENTONCES** se rechaza nombrando `git.branch_template` y la rama con `openspec/`
- **Y** `axiom.yaml` no se modifica

#### Scenario: Una plantilla de rol enviada en `monorepo-embedded` no está disponible
- **DADO** el dashboard de un `monorepo-embedded` y `PUT /api/settings` con `roles[].branch_template`
- **CUANDO** se procesa la escritura
- **ENTONCES** se rechaza porque la clave no aplica a esa topología (no figura en su catálogo)
- **Y** `axiom.yaml` no se modifica

#### Scenario: Una plantilla válida enviada como ajuste se escribe
- **DADO** el dashboard y `PUT /api/settings` con `git.branch_template: "{prefix}/{ref}-{slug}"`
- **CUANDO** se procesa la escritura
- **ENTONCES** `axiom.yaml` contiene `git.branch_template: "{prefix}/{ref}-{slug}"` y vuelve a cargarse como `COMPLIANT`
