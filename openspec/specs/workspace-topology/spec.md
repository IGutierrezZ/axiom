# Especificación Viva: Topología de Workspace y CLI de Axiom

## Propósito

Definir de forma canónica el comportamiento del punto de entrada CLI `axiom` y el subsistema de validación de topologías de espacio de trabajo (`internal/workspace`), gobernando las reglas para monorrepositorios y multirrepositorios bajo una carpeta maestra común.

---

## 1. Capacidad: `axiom-cli-identity`

El binario ejecutable `axiom` proporciona la interfaz unificada de línea de comandos para la plataforma.

### Requirement: Información de Versión e Identidad (REQ-1.1)

El ejecutable `axiom` DEBE soportar la bandera `--version`, `-v` y el subcomando `version`, imprimiendo el nombre de la plataforma, la versión semántica y la información del release.

#### Scenario: Consulta de versión mediante flag --version
- **DADO** que el usuario dispone del binario `axiom` compilado
- **CUANDO** ejecuta `axiom --version`
- **ENTONCES** la salida contiene la cadena `axiom version` seguida de la versión semántica (ej. `v0.1.0`)
- **Y** el proceso finaliza con código de salida `0`

#### Scenario: Consulta de versión mediante subcomando version
- **DADO** que el usuario invoca la CLI
- **CUANDO** ejecuta `axiom version`
- **ENTONCES** la salida muestra la versión y metadatos de compilación (commit, arquitectura y fecha)
- **Y** el código de salida es `0`

---

### Requirement: Subcomando axiom workspace validate (REQ-1.2)

El binario `axiom` DEBE proporcionar el subcomando `axiom workspace validate`, el cual evalúa el archivo `axiom.yaml` del directorio actual (o de la ruta especificada mediante `--path`) y verifica la conformidad de la topología y de los repositorios asociados a los roles. Además DEBE validar los bloques opcionales `technical_context`, `git` y `repositories` de `axiom.yaml`: la sintaxis y la contención de rutas se reportan como errores y las rutas de contexto técnico inexistentes como advertencias que no alteran el resultado. También DEBE reportar como error que un contexto técnico mantenido se declare con un almacén de artefactos sin ficheros (`engram` o `none`, REQ-30.19), DEBE rechazar un bloque `baseline:` residual con el error tipado de REQ-30.21 y DEBE rechazar la topología retirada `monorepo-decoupled` con el mensaje accionable de REQ-25.25. Una clave `governance.semantic_analysis` residual NO DEBE invalidar la configuración: se sigue leyendo, se ignora y DEBE reportarse como advertencia que nombra la clave y no altera el resultado.
(Previously: `governance.semantic_analysis` elegía el conector de `axiom semantic` y se validaba en silencio; C-08-l retira ese comando y la clave pasa a advertencia.)

#### Scenario: Validación de espacio de trabajo conforme
- **DADO** un directorio con un archivo `axiom.yaml` válido y las carpetas de repositorios existentes
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida reporta un estado `COMPLIANT` indicando la topología detectada y los roles verificados
- **Y** el código de salida del proceso es `0`

#### Scenario: Validación con ruta explícita mediante --path
- **DADO** un espacio de trabajo ubicado en una ruta externa `/tmp/test-workspace`
- **CUANDO** el usuario ejecuta `axiom workspace validate --path /tmp/test-workspace`
- **ENTONCES** la validación se ejecuta sobre el directorio especificado
- **Y** reporta el resultado correspondiente a esa ubicación

#### Scenario: Espacio de trabajo no conforme finaliza con error
- **DADO** un espacio de trabajo donde faltan repositorios obligatorios o el archivo de configuración es inválido
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida detalla de forma explícita en español los motivos de no conformidad
- **Y** el proceso finaliza con código de salida `1`

#### Scenario: Bloque `baseline:` residual finaliza con error
- **DADO** un `axiom.yaml` que declara `baseline:`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida detalla en español que el bloque se retiró y la continuación `axiom archive adopt`
- **Y** el proceso finaliza con código de salida `1`

#### Scenario: Ruta de contexto técnico inexistente solo avisa
- **DADO** un `axiom.yaml` conforme cuyo `technical_context.maintained.paths` incluye una ruta que no existe
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida incluye una advertencia que nombra la ruta y reporta `COMPLIANT`
- **Y** el código de salida del proceso es `0`

#### Scenario: Declaración incompatible con el almacén de artefactos finaliza con error
- **DADO** un `axiom.yaml` con `technical_context.maintained` válido y un `openspec/config.yaml` con `artifact_store: engram`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida detalla en español que el contexto mantenido exige un almacén de ficheros, y la continuación
- **Y** el proceso finaliza con código de salida `1`

#### Scenario: Bloque `git:` o `repositories:` inválido finaliza con error
- **DADO** un `axiom.yaml` con `git.branch_prefix: "a b//"` o con una entrada de `repositories:` cuya ruta no existe
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida detalla en español la clave o la ruta y la regla incumplida
- **Y** el proceso finaliza con código de salida `1`

#### Scenario: Topología retirada finaliza con error accionable
- **DADO** un `axiom.yaml` con `workspace.topology: monorepo-decoupled`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida indica en español que la topología fue retirada y que debe sustituirse por `multirepo`
- **Y** el proceso finaliza con código de salida `1`


#### Scenario: Clave `governance.semantic_analysis` residual solo avisa
- **DADO** un `axiom.yaml` conforme que declara `governance.semantic_analysis`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida incluye una advertencia que nombra `governance.semantic_analysis` y reporta `COMPLIANT`
- **Y** el código de salida del proceso es `0`

## 2. Capacidad: `workspace-topology-engine`

El paquete `internal/workspace` implementa la lógica de dominio para la lectura de `axiom.yaml` y la verificación determinista de la estructura de repositorios y roles bajo la carpeta maestra.

### Requirement: Estructura del Esquema axiom.yaml (REQ-2.1)

El motor de configuración DEBE deserializar y validar el archivo `axiom.yaml`, exigiendo los campos obligatorios:
- `workspace.name` (cadena no vacía)
- `workspace.topology` (uno de: `monorepo-embedded`, `multirepo`)
- `workspace.specs_repository` (ruta relativa al directorio de especificaciones)
- `roles` (mapa con al menos 1 rol definido, conteniendo `name` y una lista de `repositories`)

Además DEBE aceptar los bloques opcionales de primer nivel `technical_context` (REQ-24.26), `git` (REQ-25.26), `repositories` (REQ-25.27), `workflow` (REQ-27.1) y `delivery` (REQ-32.1): su ausencia no altera la validación y, cuando están presentes, se validan de forma estricta (claves desconocidas, rutas y valores). Un bloque `baseline` NO es un bloque admitido: se rechaza con el error tipado de REQ-30.21. Las claves desconocidas fuera de esos bloques se siguen ignorando.
(Previously: la lista de bloques opcionales de primer nivel era `technical_context`, `git`, `repositories` y `workflow`)

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
- **ENTONCES** la validación falla con el error tipado de REQ-30.21 y no lo trata como clave desconocida ignorable

### Requirement: Topología Monorepo Embebido (REQ-2.2)

En la topología `monorepo-embedded`, el repositorio de especificaciones DEBE apuntar a la raíz del propio monorrepo (`"."`) o a una subcarpeta interna existente (como `openspec` o `specs`). Además, el único rol admitido es `fullstack`: un monorepo no se subdivide en roles especializados, porque los roles especializados (`core`, `web`, `qa`, etc.) pertenecen a `multirepo`. `axiom workspace validate` DEBE reportar como error, en español, cualquier rol de `monorepo-embedded` distinto de `fullstack`, nombrando la topología, el rol y los pasos de migración (dejar solo el rol `fullstack` o declarar la topología `multirepo`), y DEBE finalizar con código de salida `1`. Ese rechazo es solo de validación: la carga de la configuración (`workspace.Locate`) NO DEBE fallar por ello, de modo que `axiom sdd status`, `axiom branch` y el resto de verbos siguen funcionando sobre un `axiom.yaml` existente. La regla NO se aplica a `multirepo`, que admite cualquier conjunto de roles válido.
(Previously: solo fijaba la ubicación del repositorio de especificaciones; un monorepo admitía cualquier conjunto de roles)

#### Scenario: Monorepo embebido con specs en raíz o subdirectorio válido
- **DADO** una configuración con topología `monorepo-embedded` y `specs_repository: "."`
- **Y** la carpeta existe dentro del espacio de trabajo
- **CUANDO** se ejecuta la validación de topología
- **ENTONCES** la topología se marca como válida y conforme

#### Scenario: Monorepo con el rol único `fullstack` es conforme
- **DADO** un `axiom.yaml` con topología `monorepo-embedded`, specs válido y el único rol `fullstack`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** el estado es `COMPLIANT` y no se reporta ningún error de roles

#### Scenario: Monorepo con roles especializados se rechaza con los pasos de migración
- **DADO** un `axiom.yaml` con topología `monorepo-embedded` y los roles `core` y `qa`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta un error en español que nombra `monorepo-embedded`, los roles `core` y `qa` y la migración (dejar solo `fullstack` o pasar a `multirepo`)
- **Y** el código de salida es `1`

#### Scenario: Monorepo con `fullstack` y otro rol se rechaza
- **DADO** un `axiom.yaml` con topología `monorepo-embedded` y los roles `fullstack` y `web`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta el error de roles nombrando `web`
- **Y** el código de salida es `1`

#### Scenario: La carga no se rompe con un monorepo de roles especializados
- **DADO** el mismo `axiom.yaml` de topología `monorepo-embedded` con los roles `core` y `qa`
- **CUANDO** se resuelve con `workspace.Locate` y se ejecuta `axiom sdd status`
- **ENTONCES** la carga tiene éxito y el status se proyecta sin error causado por los roles

#### Scenario: Multirepo con roles especializados sigue siendo conforme
- **DADO** un `axiom.yaml` con topología `multirepo`, specs válido y los roles `core` y `qa` con sus repositorios existentes
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** no se reporta ningún error de roles

### Requirement: Topología Monorepo Desacoplado (REQ-2.3) [SUPERSEDED / DEPRECADO]
> [!CAUTION]
> **REQUISITO HISTÓRICO SUPERADO / DEPRECADO:** Este requisito ha sido superado o deprecado.
> Motivo: `monorepo-decoupled` no se comportaba de forma distinta a `multirepo`: ambas separaban el repositorio de especificaciones y se resolvían con la misma lógica, y ninguna superficie de usuario —`axiom init`, el dashboard, el hub— la ofrecía ya. Retirarla elimina un valor de enumeración y mensajes redundantes, sin cambio de comportamiento para `multirepo`.


En la topología `monorepo-decoupled`, el repositorio de especificaciones DEBE residir en un directorio independiente dentro de la carpeta maestra común, distinto del repositorio de código del monorrepo. Esta topología DEBE admitir el patrón canónico: una carpeta maestra con `.axiom-workspace` que apunta a `<specs_repository>/axiom.yaml`, en el que el planning home es `<repositorio de especificaciones>/openspec` (capacidad `planning-home-resolution`, REQ-24.3).
(Previously: no contemplaba el patrón canónico con puntero ni la ubicación del planning home)

#### Scenario: Monorepo desacoplado con repositorio de specs presente
- **DADO** una carpeta maestra que contiene `app-monorepo/` y `especificacion/`
- **Y** `specs_repository` apunta a `especificacion`
- **CUANDO** se ejecuta la validación
- **ENTONCES** la validación es exitosa

#### Scenario: Monorepo desacoplado sin repositorio de specs
- **DADO** una configuración `monorepo-decoupled` donde la ruta indicada en `specs_repository` no existe en el disco
- **CUANDO** se ejecuta la validación
- **ENTONCES** se genera un error de validación indicando que el repositorio de especificaciones es obligatorio y no fue encontrado

#### Scenario: Patrón canónico con puntero en monorepo desacoplado
- **DADO** una carpeta maestra con `.axiom-workspace` (`config: especificacion/axiom.yaml`), `app-monorepo/` y `especificacion/` con su `axiom.yaml` (`monorepo-decoupled`, `specs_repository: especificacion`)
- **CUANDO** se resuelve el planning home desde cualquier directorio del workspace
- **ENTONCES** el planning home es `<maestra>/especificacion/openspec` y `workspaceRoot` es `<maestra>/especificacion`

---

### Requirement: Topología Multirepo Federado (REQ-2.4)

En la topología `multirepo`, múltiples repositorios de código coexisten bajo la carpeta maestra común. Es REQUISITO OBLIGATORIO que exista un repositorio canónico de especificaciones dentro de dicha carpeta. Esta topología DEBE admitir el patrón canónico: una carpeta maestra no versionada con `.axiom-workspace` que apunta a `<specs_repository>/axiom.yaml`, en el que el planning home es `<repositorio canónico de especificaciones>/openspec` y los verbos SDD resuelven el mismo planning home desde la carpeta maestra, desde el repositorio de especificaciones o desde un repositorio de código (capacidad `planning-home-resolution`, REQ-24.3 y REQ-24.8).
(Previously: no contemplaba el patrón canónico con puntero ni la ubicación del planning home)

#### Scenario: Multirepo conforme con todos sus repositorios
- **DADO** una carpeta maestra con `especificacion/`, `frontend/` y `backend/`
- **Y** la configuración mapea los roles a `frontend` y `backend`, con `specs_repository: "especificacion"`
- **CUANDO** se ejecuta la validación
- **ENTONCES** el informe declara todos los repositorios y la topología como válidos

#### Scenario: Multirepo sin repositorio canónico de especificaciones
- **DADO** un proyecto multirrepositorio donde el directorio de `specs_repository` no existe
- **CUANDO** se ejecuta la validación de topología
- **ENTONCES** la validación es rechazada con un error crítico indicando que los proyectos multirrepositorio requieren obligatoriamente un repositorio canónico de especificación

#### Scenario: Repositorio asignado a un rol no existe físicamente
- **DADO** un rol `devops` configurado para gestionar la ruta `infra/k8s`
- **Y** la carpeta `infra/k8s` no existe dentro de la carpeta maestra
- **CUANDO** se ejecuta la validación
- **ENTONCES** la validación falla reportando específicamente qué ruta de qué rol no ha sido localizada en el sistema de archivos

#### Scenario: Patrón canónico con puntero en multirepo
- **DADO** una carpeta maestra con `.axiom-workspace` (`config: repo-specs/axiom.yaml`), `repo-specs/` (repositorio git con `axiom.yaml`, `multirepo`, `specs_repository: repo-specs`) y los repositorios `repo-backend/` y `repo-frontend/`
- **CUANDO** se ejecuta `axiom sdd status --json` desde la carpeta maestra, desde `repo-specs/` y desde `repo-backend/src`
- **ENTONCES** los tres devuelven `planningHome.path` igual a `<maestra>/repo-specs/openspec`

---

### Requirement: Reporte de Validación Estructurado (REQ-2.5)

La función `ValidateTopology` DEBE retornar un objeto estructurado `ValidationReport` que contenga:
- `Valid` (booleano: `true` si cumple todas las reglas, `false` si hay errores)
- `Topology` (tipo de topología evaluada)
- `CheckedPaths` (lista de rutas verificadas)
- `Errors` (lista de mensajes de error explicativos en español)
- `Warnings` (lista de advertencias o recomendaciones, si aplican)

#### Scenario: Generación de reporte completo en caso de fallo
- **DADO** una ejecución de validación con errores
- **CUANDO** se obtiene el resultado
- **ENTONCES** el objeto `ValidationReport` contiene `Valid: false`, la lista de errores y las rutas inspeccionadas
### Requirement: Retirada de la Topología `monorepo-decoupled` con Error Accionable (REQ-25.25)

El valor `monorepo-decoupled` NO DEBE aceptarse en ningún punto: ni en la carga (`LoadConfig`), ni en la validación (`axiom workspace validate`), ni en la resolución del workspace, ni en las superficies de usuario (CLI, dashboard, TUI, hub). Un `axiom.yaml` que lo declare DEBE rechazarse con un mensaje en español que indique que la topología fue retirada y que la corrección es `multirepo`, y `axiom workspace validate` DEBE finalizar con código de salida `1`. No existe alias ni migración automática: el mensaje es la migración. La especificación viva, el manual de inicio (`docs/manual-de-inicio.md`) y `axiom.example.yaml` NO DEBEN presentar el valor como válido.

#### Scenario: La carga rechaza la topología retirada
- **DADO** un `axiom.yaml` con `workspace.topology: monorepo-decoupled`
- **CUANDO** se invoca `LoadConfig`
- **ENTONCES** se devuelve un error en español que nombra `monorepo-decoupled` como retirada y `multirepo` como corrección

#### Scenario: La validación termina con código 1 y la corrección
- **DADO** el mismo `axiom.yaml`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `monorepo-decoupled` y `multirepo`
- **Y** el código de salida es `1`

#### Scenario: Un puntero hacia una configuración con la topología retirada rechaza la resolución
- **DADO** un `.axiom-workspace` cuyo destino es un `axiom.yaml` con `workspace.topology: monorepo-decoupled`
- **CUANDO** se ejecuta `axiom sdd status --json` desde la carpeta del puntero
- **ENTONCES** se rechaza nombrando la topología retirada, `multirepo` y `axiom workspace validate`, conforme a REQ-24.7
- **Y** no se devuelve ningún status

#### Scenario: Ninguna documentación presenta el valor como válido
- **DADO** los textos de `openspec/specs/`, `docs/manual-de-inicio.md` y `axiom.example.yaml`
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** ninguno presenta `monorepo-decoupled` como valor válido de `workspace.topology`
- **Y** cualquier mención restante lo declara retirado y remite a `multirepo`

---

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
### Requirement: Bloque `workflow:` Opcional de `axiom.yaml` con el Carril del Proyecto (REQ-27.1)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `workflow:` con la clave opcional `lane`, cuyos valores válidos son `odd`, `sdd` y `ask`. Sin el bloque, o sin la clave, el carril efectivo del proyecto vale `sdd`. El bloque DEBE decodificarse de forma estricta por lista cerrada de claves, con el mismo patrón que `git:` (REQ-25.26): una clave desconocida dentro de `workflow:` y un valor de `lane` fuera de los tres admitidos DEBEN reportarse como error accionable que nombre la clave (`workflow.lane` o la clave desconocida) y los valores admitidos, tanto al cargar la configuración como en `axiom workspace validate`, con código de salida `1`. Un `workflow:` que no sea un mapa también DEBE reportarse como error. Las claves desconocidas de primer nivel fuera de los bloques reconocidos se siguen ignorando (con la excepción de `baseline`, que se rechaza con el error tipado de REQ-30.21): un binario anterior a este cambio DEBE cargar un `axiom.yaml` con el bloque `workflow:` sin error y sin interpretarlo. El carril es política del proyecto y se versiona; la elección del modo de trabajo personal NO DEBE declararse en `axiom.yaml`. El valor por defecto `sdd` rige en `monorepo-embedded` y en `multirepo`; en `multirepo` el carril `odd` sigue estando permitido.
(Previously: las claves desconocidas de primer nivel se ignoraban citando REQ-24.17 de `spec-baseline-declaration`, que se retira; `baseline` deja de ser un bloque reconocido y se rechaza)

#### Scenario: Sin bloque el carril efectivo es `sdd`

- **DADO** un `axiom.yaml` sin `workflow:`
- **CUANDO** se resuelve la declaración del proyecto
- **ENTONCES** el carril efectivo es `sdd` con origen por defecto

#### Scenario: Bloque con `lane: odd`

- **DADO** un `axiom.yaml` con `workflow: {lane: odd}`
- **CUANDO** se resuelve la declaración del proyecto
- **ENTONCES** el carril efectivo es `odd` con origen declarado

#### Scenario: Bloque con `lane: ask`

- **DADO** un `axiom.yaml` con `workflow: {lane: ask}`
- **CUANDO** se resuelve la declaración del proyecto
- **ENTONCES** el carril efectivo es `ask` con origen declarado

#### Scenario: Bloque vacío completa con el valor por defecto

- **DADO** un `axiom.yaml` con `workflow: {}`
- **CUANDO** se resuelve la declaración del proyecto
- **ENTONCES** el carril efectivo es `sdd` y no se reporta ningún error

#### Scenario: Valor desconocido de `lane`

- **DADO** un `axiom.yaml` con `workflow: {lane: agil}`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `workflow.lane`, el valor `agil` y los valores admitidos `odd`, `sdd` y `ask`
- **Y** el código de salida es `1`

#### Scenario: Clave desconocida dentro de `workflow:`

- **DADO** un `axiom.yaml` con `workflow: {lane: sdd, publish: true}`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra la clave `publish` como desconocida
- **Y** el código de salida es `1`

#### Scenario: `workflow:` que no es un mapa

- **DADO** un `axiom.yaml` con `workflow: sdd`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra la clave `workflow` y la regla incumplida
- **Y** el código de salida es `1`

#### Scenario: Un binario anterior ignora el bloque

- **DADO** un `axiom.yaml` válido con `workflow: {lane: ask}`
- **CUANDO** lo carga el decodificador previo a este cambio (el que ignora las claves desconocidas de primer nivel)
- **ENTONCES** la carga termina sin error
- **Y** el resto de la configuración se lee igual que sin el bloque

#### Scenario: Multirepo admite `odd`

- **DADO** un `axiom.yaml` de topología `multirepo` con `workflow: {lane: odd}`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** el resultado es `COMPLIANT`
- **Y** el código de salida es `0`

#### Scenario: La plantilla y el manual documentan el bloque

- **DADO** la plantilla `axiom.example.yaml` y el manual de inicio
- **CUANDO** se inspeccionan (prueba de pin)
- **ENTONCES** ambos documentan `workflow.lane`, sus tres valores y el valor por defecto `sdd`

### Requirement: Un Bloque `baseline:` Residual se Rechaza con Error Tipado y Guía de Migración (REQ-30.21)

`axiom.yaml` NO DEBE admitir un bloque de primer nivel `baseline:`: la spec viva es únicamente `openspec/specs/`. Un `baseline:` residual NO DEBE ignorarse en silencio ni degradar a «sin baseline». DEBE producir una causa tipada que nombre el bloque `baseline:` y la continuación («retira `baseline:` y migra los requisitos con `axiom archive adopt`») en: (a) la carga de `axiom.yaml`; (b) `axiom workspace validate`, que DEBE finalizar con código de salida `1`; (c) `axiom sdd status`, que DEBE marcar el archivado como bloqueado (`dependencies.archive` en `blocked`) con esa causa sin bloquear las dependencias de las fases de planificación ni la resolución del planning home; y (d) el archivado, que DEBE detenerse sin escribir en `openspec/specs/` ni mover el cambio. El status NO DEBE proyectar ninguna clave `baseline`. Un `axiom.yaml` sin `baseline:` NO DEBE verse afectado.

#### Scenario: `axiom workspace validate` rechaza el bloque
- **DADO** un `axiom.yaml` válido que además declara `baseline:` con un documento
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** la salida detalla en español que el bloque `baseline:` se retiró y la continuación `axiom archive adopt`
- **Y** el proceso finaliza con código de salida `1`

#### Scenario: El status bloquea solo el archivado
- **DADO** el mismo `axiom.yaml`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `dependencies.archive` es `blocked` con una causa que nombra `baseline:` y la continuación
- **Y** las dependencias de las fases de planificación no son `blocked` por esa causa
- **Y** el JSON no contiene la clave `baseline`

#### Scenario: El archivado se detiene sin escribir una segunda spec viva
- **DADO** el mismo `axiom.yaml` y un cambio listo para archivar
- **CUANDO** se solicita el archivado
- **ENTONCES** archive se detiene con el motivo tipado
- **Y** `openspec/specs/` no se modifica y el cambio permanece en `openspec/changes/`

#### Scenario: Un `axiom.yaml` sin `baseline:` no se ve afectado
- **DADO** un `axiom.yaml` válido sin `baseline:`
- **CUANDO** se ejecutan `axiom workspace validate` y `axiom sdd status --json`
- **ENTONCES** no se reporta ninguna causa por `baseline:`

#### Scenario: La carga produce la causa tipada
- **DADO** un `axiom.yaml` con `baseline: {}` vacío
- **CUANDO** se carga la configuración
- **ENTONCES** la carga reporta la causa tipada que nombra `baseline:`
- **Y** no se degrada a «sin baseline»

### Requirement: El Almacén de Artefactos Declarado Solo Admite Ficheros: `engram` y `none` se Reportan (REQ-2.6)

En Axiom el almacén de artefactos de una sesión SDD es siempre `hybrid`. `axiom workspace validate` DEBE reportar como error, en español, un `artifact_store` (clave `artifact_store` o `artifactStore` de `openspec/config.yaml`) declarado `engram` o `none`, con independencia de que exista contexto técnico mantenido. El mensaje DEBE nombrar la clave, el valor declarado, la causa (con esos almacenes no hay paradas de tres salidas, ni publicación de compuertas ni relevo con `--hand-off`) y la migración (retirar la clave o declarar `hybrid`), y la validación DEBE finalizar con código de salida `1`. Un `artifact_store` declarado `openspec` o `hybrid`, o la ausencia de declaración, NO DEBE reportarse. El rechazo es solo de validación: la carga (`workspace.Locate`), `axiom sdd status` (que sin declaración sigue resolviendo `openspec`), la publicación y el archivado NO cambian de comportamiento. Este binario NO reescribe `openspec/config.yaml`.

#### Scenario: `engram` declarado se reporta con la migración
- **DADO** un `openspec/config.yaml` con `artifact_store: engram` y un `axiom.yaml` válido sin contexto técnico mantenido
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta un error en español que nombra `artifact_store`, `engram`, la ausencia de paradas, publicación y relevo, y la migración (retirar la clave o declarar `hybrid`)
- **Y** el código de salida es `1`

#### Scenario: `none` declarado se reporta
- **DADO** un `openspec/config.yaml` con `artifact_store: none`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta el mismo error nombrando `none`
- **Y** el código de salida es `1`

#### Scenario: `openspec`, `hybrid` y la ausencia de declaración se aceptan
- **DADO** tres proyectos válidos: con `artifact_store: openspec`, con `artifact_store: hybrid` y sin `artifact_store` declarado
- **CUANDO** se ejecuta `axiom workspace validate` en cada uno
- **ENTONCES** ninguno reporta error por el almacén

#### Scenario: La carga no se rompe con `engram` o `none`
- **DADO** un proyecto con `artifact_store: engram` declarado
- **CUANDO** se resuelve con `workspace.Locate` y se ejecuta `axiom sdd status --json`
- **ENTONCES** la carga y el status funcionan como antes de este cambio
- **Y** el almacén declarado sigue leyéndose como `engram`
### Requirement: Bloque `delivery:` Opcional de `axiom.yaml` con el Umbral del Aviso de Cambio Grande (REQ-32.1)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `delivery:` con la clave opcional `split_hint_lines`, un entero positivo (mayor o igual que 1) que fija el número de líneas a partir del cual el orquestador avisa de que un cambio es grande (REQ-32.2). Sin el bloque, o sin la clave, el aviso está desactivado: no hay umbral por defecto y ninguna superficie lo sustituye por un número. El valor `0` NO es una forma de desactivar el aviso: se rechaza; la forma de desactivarlo es no declarar la clave. El bloque DEBE decodificarse de forma estricta por lista cerrada de claves, con el mismo patrón y el mismo canal de error que `git:` (REQ-25.26) y `workflow:` (REQ-27.1), tanto al cargar la configuración como en `axiom workspace validate`, con código de salida `1`. DEBEN reportarse como error accionable que nombre la clave: una clave desconocida o repetida dentro de `delivery:`, un `delivery:` que no sea un mapa, y un valor de `split_hint_lines` que no sea un entero, que sea `0` o negativo, o que sea nulo (`delivery.split_hint_lines`). Las claves desconocidas de primer nivel fuera de los bloques reconocidos se siguen ignorando: un binario anterior a este cambio DEBE cargar un `axiom.yaml` con el bloque `delivery:` sin error y sin interpretarlo. `axiom init --force` DEBE preservar un bloque `delivery:` ya declarado, igual que los demás bloques declarados. La clave es política del proyecto y se versiona. El manual de inicio (`docs/manual-de-inicio.md`) DEBE documentar la clave, su dominio y que su ausencia desactiva el aviso.

#### Scenario: Sin bloque el aviso está desactivado

- **DADO** un `axiom.yaml` sin `delivery:`
- **CUANDO** se resuelve la declaración del proyecto
- **ENTONCES** `split_hint_lines` no tiene valor y el aviso está desactivado
- **Y** `axiom workspace validate` no reporta ningún error por ello

#### Scenario: Bloque vacío o sin clave mantiene el aviso desactivado

- **DADO** un `axiom.yaml` con `delivery: {}`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** el resultado es `COMPLIANT` y el aviso sigue desactivado

#### Scenario: Valor válido

- **DADO** un `axiom.yaml` con `delivery: {split_hint_lines: 600}`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** el resultado es `COMPLIANT`
- **Y** la configuración resuelta expone `split_hint_lines` con el valor `600`

#### Scenario: Valor cero, negativo, no entero o nulo

- **DADO** un `axiom.yaml` con `delivery: {split_hint_lines: 0}` (y variantes con `-5`, `"mucho"`, `12.5` y `null`)
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `delivery.split_hint_lines` y la regla incumplida (entero positivo)
- **Y** el código de salida es `1`

#### Scenario: Clave desconocida o repetida dentro de `delivery:`

- **DADO** un `axiom.yaml` con `delivery: {split_hint_lines: 600, budget: 400}` (y otro con `split_hint_lines` repetida)
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra la clave `budget` como desconocida (y la repetida como repetida)
- **Y** el código de salida es `1`

#### Scenario: `delivery:` que no es un mapa

- **DADO** un `axiom.yaml` con `delivery: 600`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra la clave `delivery` y la regla incumplida
- **Y** el código de salida es `1`

#### Scenario: La carga de la configuración rechaza el valor inválido por el mismo canal

- **DADO** un `axiom.yaml` con `delivery: {split_hint_lines: -1}`
- **CUANDO** se carga la configuración del workspace
- **ENTONCES** la carga reporta el error de `delivery.split_hint_lines` junto con los de los demás bloques, no un error genérico

#### Scenario: Un binario anterior ignora el bloque

- **DADO** un `axiom.yaml` válido con `delivery: {split_hint_lines: 600}`
- **CUANDO** lo carga el decodificador previo a este cambio (el que ignora las claves desconocidas de primer nivel)
- **ENTONCES** la carga termina sin error
- **Y** el resto de la configuración se lee igual que sin el bloque

#### Scenario: `axiom init --force` preserva el bloque

- **DADO** un `axiom.yaml` con `delivery: {split_hint_lines: 600}`
- **CUANDO** el usuario ejecuta `axiom init --force`
- **ENTONCES** el `axiom.yaml` resultante conserva el bloque `delivery:` con su valor

#### Scenario: Ambas topologías admiten el bloque

- **DADO** un `axiom.yaml` de topología `monorepo-embedded` y otro `multirepo`, ambos con `delivery: {split_hint_lines: 600}`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** el resultado es `COMPLIANT` en ambos

#### Scenario: El manual documenta la clave

- **DADO** el manual de inicio
- **CUANDO** se inspecciona (prueba de pin)
- **ENTONCES** documenta `delivery.split_hint_lines`, que es un entero positivo y que su ausencia desactiva el aviso

---

### Requirement: Con `delivery.split_hint_lines` Fijada el Orquestador Avisa una Vez de un Cambio Grande y Nunca Bloquea (REQ-32.2)

Cuando `axiom.yaml` declara `delivery.split_hint_lines` (REQ-32.1) y la previsión de líneas del trabajo la supera, el orquestador DEBE emitir **un único aviso** por cambio (SDD) o por feature (ODD) indicando que el cambio es grande y que la persona puede pedir PRs encadenadas, y DEBE continuar sin esperar respuesta. El aviso NUNCA bloquea ni detiene el avance, no cambia la estrategia de entrega (sigue siendo `single-pr` salvo petición expresa de la persona) y no exige ninguna decisión. La previsión que se compara es la estimación cualitativa de tamaño de `sdd-tasks` en SDD y la previsión del documento de la feature en ODD. El orquestador lee la clave directamente de `axiom.yaml` (no se proyecta en `axiom sdd status`). Sin la clave NO hay aviso ni ningún número de comparación: ni la doctrina ni los orquestadores DEBEN contener un umbral por defecto.

#### Scenario: Con la clave fijada y la previsión superada se avisa una vez

- **DADO** un `axiom.yaml` con `delivery: {split_hint_lines: 600}` y una previsión de líneas del cambio superior a 600
- **CUANDO** el orquestador evalúa la previsión
- **ENTONCES** emite un único aviso de cambio grande que menciona que se pueden pedir PRs encadenadas
- **Y** continúa con la entrega `single-pr` sin esperar respuesta

#### Scenario: El aviso no se repite en el mismo cambio

- **DADO** un cambio sobre el que ya se emitió el aviso
- **CUANDO** el orquestador vuelve a evaluar la previsión en una fase posterior del mismo cambio
- **ENTONCES** no emite un segundo aviso

#### Scenario: Con la previsión por debajo o igual no hay aviso

- **DADO** un `axiom.yaml` con `delivery: {split_hint_lines: 600}` y una previsión que no supera 600
- **CUANDO** el orquestador evalúa la previsión
- **ENTONCES** no emite ningún aviso

#### Scenario: Sin la clave no hay aviso ni número

- **DADO** un `axiom.yaml` sin `delivery:`
- **CUANDO** el orquestador evalúa la previsión de un cambio de cualquier tamaño
- **ENTONCES** no emite ningún aviso
- **Y** la doctrina que lo describe no contiene ningún número de líneas como umbral por defecto

#### Scenario: El aviso nunca bloquea

- **DADO** un aviso de cambio grande emitido
- **CUANDO** la persona no responde
- **ENTONCES** el trabajo continúa sin parada ni decisión pendiente
- **Y** `sdd-apply` no se detiene por ese motivo
### Requirement: Bloque `integrations:` Opcional y Estricto de `axiom.yaml` (REQ-43.19)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `integrations:`, un mapeo `<plugin>: <mapeo de configuración>`. SOLO DEBE admitir identificadores presentes en el catálogo de manifiestos (REQ-43.4) y, dentro de cada uno, solo las claves de configuración de su manifiesto (REQ-43.3). Son error de carga, en `ParseConfig`, `LoadConfig` y `axiom workspace validate`: un plugin desconocido, una clave desconocida, un valor del bloque nulo o que no es un mapeo, una clave duplicada un valor de plugin nulo o que no es un mapeo, un valor de tipo distinto del declarado en el esquema del manifiesto, un valor fuera del enumerado y una clave obligatoria ausente (esta última solo si `enabled` no es `false`). Dentro de cada plugin, además de las claves del manifiesto, se admite la clave reservada `enabled` (booleano estricto, por defecto `true`); un plugin con `enabled: false` no configura nada y no recibe eventos. Los valores por defecto del manifiesto se aplican al proyectar el bloque. Los errores se acumulan todos, con línea y alternativas (con el catálogo vacío, «plugins admitidos: ninguno en esta versión»). `integrations: {}` es válido y no configura nada. El mensaje DEBE incluir la línea y la lista de claves o plugins admitidos, con el mismo formato que el resto de bloques. Sin el bloque, NO DEBE cambiar ningún comportamiento (REQ-43.5). Ningún secreto DEBE poder declararse en este bloque: los secretos no son claves de configuración del manifiesto (REQ-43.3), así que una clave con el nombre de un secreto es una clave desconocida.

(Placement: section "Bloque `workflow:` Opcional de `axiom.yaml` con el Carril del Proyecto (REQ-27.1)", a continuación)

#### Scenario: Bloque válido
- **DADO** el doble registrado en el catálogo y `integrations: { <doble>: { <clave válida>: valor } }`
- **CUANDO** se carga la configuración
- **ENTONCES** la carga termina sin error y `WorkspaceConfig` proyecta el bloque

#### Scenario: Clave desconocida
- **DADO** `integrations: { <doble>: { clave_inexistente: 1 } }`
- **CUANDO** se ejecuta `ParseConfig`, `LoadConfig` o `axiom workspace validate`
- **ENTONCES** falla con un error que nombra `clave_inexistente`, su línea y las claves admitidas

#### Scenario: Plugin desconocido
- **DADO** `integrations: { azure: {} }` sin ese plugin en el catálogo
- **CUANDO** se carga la configuración
- **ENTONCES** falla con un error que nombra `azure`, su línea y los plugins admitidos

#### Scenario: Bloque nulo o no mapeo
- **DADO** `integrations:` sin valor, o `integrations: [x]`
- **CUANDO** se carga la configuración
- **ENTONCES** falla con un error tipado que indica que debe ser un mapeo

#### Scenario: Clave duplicada
- **DADO** dos entradas `<doble>:` iguales bajo `integrations:`
- **CUANDO** se carga la configuración
- **ENTONCES** falla con un error que nombra la clave duplicada y su línea

#### Scenario: Plugin desactivado
- **DADO** `integrations: { <doble>: { enabled: false } }` sin la clave obligatoria del manifiesto
- **CUANDO** se carga la configuración
- **ENTONCES** la carga termina sin error y el plugin no figura entre los activos

#### Scenario: Obligatoria ausente, tipo y enumerado
- **DADO** un plugin activo sin su clave obligatoria, otro con un valor de tipo incorrecto y otro con un valor fuera del enumerado
- **CUANDO** se carga la configuración
- **ENTONCES** falla con un error por cada problema, con su línea y las alternativas

#### Scenario: Bloque vacío
- **DADO** `integrations: {}`
- **CUANDO** se carga la configuración
- **ENTONCES** la carga termina sin error y no configura ningún plugin

#### Scenario: Un secreto no se puede declarar en `axiom.yaml`
- **DADO** un manifiesto cuyo secreto se llama `token` y `integrations: { <doble>: { token: abc } }`
- **CUANDO** se carga la configuración
- **ENTONCES** falla como clave desconocida y el mensaje no contiene `abc`

### Requirement: `Locate` Expone el Error del Bloque sin Rechazar y no Bloquea el Archivado (REQ-43.20)

`workspace.Locate` NO DEBE rechazar por un bloque `integrations:` inválido: DEBE exponerlo como `Declarations.IntegrationsErr`, igual que `git:`, `workflow:` y `sdd:`. El bloque `integrations:` NO DEBE bloquear el archivado (`axiom sdd status` no lo cuenta entre las declaraciones inválidas que bloquean). `axiom init --force` DEBE tratar `integrations:` como bloque propio del usuario y no sobrescribirlo (`declares()` conoce `Integrations`).

#### Scenario: `Locate` con bloque inválido
- **DADO** un `axiom.yaml` con `integrations:` inválido
- **CUANDO** se llama a `workspace.Locate`
- **ENTONCES** no devuelve error por ello
- **Y** `Declarations.IntegrationsErr` contiene el error de REQ-43.19

#### Scenario: Archivado no bloqueado
- **DADO** un `axiom.yaml` con `integrations:` inválido y un cambio listo para archivar
- **CUANDO** se consulta `axiom sdd status`
- **ENTONCES** la fase de archivado no queda bloqueada por esa declaración

#### Scenario: `init --force` conserva el bloque
- **DADO** un `axiom.yaml` con un bloque `integrations:`
- **CUANDO** se ejecuta `axiom init --force`
- **ENTONCES** el bloque `integrations:` se conserva sin cambios

### Requirement: Plantilla y Manual Documentan `integrations:` (REQ-43.21)

`axiom.example.yaml` DEBE incluir un bloque `integrations:` comentado que contenga el bloque `integrations.azure-devops` con las claves `organization`, `project` y `team`, y `docs/manual-de-inicio.md` (sección 6) DEBE incluir un apartado `integrations:` que describa el bloque, el catálogo de plugins (`azure-devops`), el secreto `pat` y su variable `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`, los scopes mínimos del PAT por operación (`vso.work_write`, `vso.project` y `vso.code`), que el PAT es de una sola organización y que los secretos nunca se escriben en `axiom.yaml`, y los verbos `axiom integrations`, incluidos `connect` y `discover`. Las pruebas de contrato (`example_config_test.go`, `docs_contract_test.go`) DEBEN fijar ambos y NO DEBEN afirmar que el catálogo está vacío.
(Previously: la plantilla y el manual describían un bloque genérico con el catálogo vacío hasta C-07-c, sin el bloque `azure-devops`, scopes, secreto `pat` ni los verbos `connect` y `discover`)

#### Scenario: Plantilla y manual cubiertos por las pruebas de contrato
- **CUANDO** se ejecutan las pruebas de contrato de la plantilla y del manual
- **ENTONCES** comprueban la presencia del bloque comentado y del apartado, y fallan si se eliminan

#### Scenario: La plantilla documenta `azure-devops`
- **DADO** `axiom.example.yaml`
- **CUANDO** se descomenta el bloque `integrations.azure-devops` y se carga con el catálogo por defecto
- **ENTONCES** la carga termina sin error y el bloque declara `organization`, `project` y `team`

#### Scenario: El manual documenta secreto, scopes y verbos
- **DADO** `docs/manual-de-inicio.md`
- **CUANDO** se ejecuta la prueba de contrato del manual
- **ENTONCES** comprueba que el apartado menciona `pat`, `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`, `vso.work_write`, `vso.project`, `vso.code`, `integrations connect` e `integrations discover`
- **Y** la prueba falla si alguno falta

#### Scenario: Ya no se afirma catálogo vacío
- **DADO** la prueba de contrato de la plantilla
- **CUANDO** se ejecuta
- **ENTONCES** no exige que el catálogo de plugins esté vacío ni que el bloque descomentado sea rechazado
