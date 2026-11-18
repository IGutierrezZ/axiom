# Delta para workspace-topology

> Parte del estado vigente: el resto de requisitos NO cambia. Esta delta añade a `axiom.yaml` la rama principal (`git.main_branch`) y el modo de promoción (`promotion`, en `git:` y en `repositories[]`) de la promoción de ramas en dos etapas (C-02-a, capacidad `change-branch-lifecycle`, REQ-25.46). La carga de la configuración (`workspace.Locate`) sigue la regla actual de `git:` y `repositories:`: un valor inválido se reporta con la política de ramas (`config_invalid` en `axiom branch`) y `axiom workspace validate` lo nombra.

## MODIFIED Requirements

### Requirement: Bloque `git:` Opcional de `axiom.yaml` (REQ-25.26)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `git:` con las claves opcionales `integration_branch`, `branch_prefix`, `main_branch` y `promotion`. Sin el bloque, o sin una de las claves, `integration_branch` vale `main` y `branch_prefix` vale `feat`; `main_branch` NO tiene valor por defecto (su ausencia significa que la rama de integración es la única etapa y la rama principal no existe como concepto distinto) y `promotion` NO tiene valor por defecto declarado: cuando hay dos etapas y no se declara, el modo de promoción efectivo es `branch` (REQ-25.46 de `change-branch-lifecycle`). `integration_branch`, `branch_prefix` y `main_branch` DEBEN ser cadenas no vacías que formen nombres de referencia git válidos (`branch_prefix` sin `/` inicial ni final, sin `..` ni espacios). `promotion` DEBE ser una de las cadenas `branch`, `release` o `ask`; `ask` es válido en la configuración y significa que la persona decide el modo al crear la rama de cada cambio. Una clave desconocida dentro de `git:` y un valor inválido DEBEN reportarse como error nombrando la clave, y el mensaje que enumera las claves admitidas DEBE listar las cuatro. Los valores del bloque son política del proyecto y se versionan: el modo de trabajo y la raíz de los worktrees NO DEBEN declararse en `axiom.yaml` (viven en el hub personal, capacidad `multi-project-hub`).

`promotion` sin `main_branch`, o con `main_branch` igual a la rama de integración resuelta, NO tiene efecto: `axiom workspace validate` DEBE emitir una advertencia no bloqueante que lo indique, sin cambiar el estado `COMPLIANT` ni el código de salida.
(Previously: `git:` admitía solo `integration_branch` y `branch_prefix`)

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
- **ENTONCES** la salida nombra la clave `worktree_mode` como desconocida y enumera `integration_branch`, `branch_prefix`, `main_branch` y `promotion` como admitidas
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

---

### Requirement: Bloque `repositories:` de Primer Nivel con Ajustes por Repositorio (REQ-25.27)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `repositories:` con una lista de entradas, cada una con `path` (obligatorio, relativo a la carpeta maestra) y, opcionalmente, `integration_branch`, `branch_prefix` y `promotion` propios del repositorio, con las mismas reglas de validez que `git:` (supuesto S-13): `promotion` es `branch`, `release` o `ask`. `main_branch` NO es una clave de `repositories:`: la rama principal es única del proyecto y declararla en una entrada DEBE reportarse como clave desconocida. Un repositorio sin entrada, o sin una de las claves, hereda el valor del bloque `git:`, campo a campo. `path` NO DEBE ser absoluta ni salir de la carpeta maestra y DEBE existir como directorio; DOS entradas que resuelvan a la misma raíz git DEBEN rechazarse nombrando ambas. Una entrada cuyo `path` no pertenezca a ningún repositorio git NO DEBE ser un error de validación: DEBE emitir una advertencia que la nombre (REQ-25.6). Una clave desconocida en una entrada DEBE reportarse como error, y el mensaje que enumera las claves admitidas DEBE listar `path`, `integration_branch`, `branch_prefix` y `promotion`. Una entrada con `promotion` cuyo repositorio no tenga dos etapas (sin `git.main_branch` o con `git.main_branch` igual a la rama de integración resuelta de ese repositorio) DEBE emitir una advertencia no bloqueante que la nombre, sin cambiar el estado ni el código de salida. La resolución de la raíz git de cada entrada sigue REQ-25.6.
(Previously: las entradas admitían solo `integration_branch` y `branch_prefix`)

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
- **ENTONCES** la salida nombra `main_branch` como clave desconocida de la entrada y enumera `path`, `integration_branch`, `branch_prefix` y `promotion` como admitidas
- **Y** el código de salida es `1`

#### Scenario: `promotion` en un repositorio sin dos etapas solo avisa
- **DADO** un `axiom.yaml` sin `git.main_branch` y una entrada con `promotion: release`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida incluye una advertencia que nombra la entrada y explica que `promotion` no tiene efecto
- **Y** el estado sigue siendo `COMPLIANT` y el código de salida es `0`
