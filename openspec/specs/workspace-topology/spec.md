# Especificación Viva: Topología de Workspace y CLI de Axiom

## Propósito

Definir de forma canónica el comportamiento del punto de entrada CLI `axiom` y el subsistema de validación de topologías de espacio de trabajo (`internal/workspace`), gobernando las reglas para monorrepositorios y multirrepositorios bajo una carpeta maestra común.

---

## 1. Capacidad: `axiom-cli-identity`

El binario ejecutable `axiom` proporciona la interfaz unificada de línea de comandos para la plataforma.

### Requirement: Información de Versión e Identidad (RF-TOP-01)

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

### Requirement: Subcomando axiom workspace validate (RF-TOP-02)

El binario `axiom` DEBE proporcionar el subcomando `axiom workspace validate`, el cual evalúa el archivo `axiom.yaml` del directorio actual (o de la ruta especificada mediante `--path`) y verifica la conformidad de la topología y de los repositorios asociados a los roles. Además DEBE validar los bloques opcionales `technical_context`, `git` y `repositories` de `axiom.yaml`: la sintaxis y la contención de rutas se reportan como errores y las rutas de contexto técnico inexistentes como advertencias que no alteran el resultado. También DEBE reportar como error que un contexto técnico mantenido se declare con un almacén de artefactos sin ficheros (`engram` o `none`, RF-CTX-17), DEBE rechazar un bloque `baseline:` residual con el error tipado de RF-TOP-12 y DEBE rechazar la topología retirada `monorepo-decoupled` con el mensaje accionable de RF-TOP-08. Una clave `governance.semantic_analysis` residual NO DEBE invalidar la configuración: se sigue leyendo, se ignora y DEBE reportarse como advertencia que nombra la clave y no altera el resultado.
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

### Requirement: Estructura del Esquema axiom.yaml (RF-TOP-03)

El motor de configuración DEBE deserializar y validar el archivo `axiom.yaml`, exigiendo los campos obligatorios:
- `workspace.name` (cadena no vacía)
- `workspace.topology` (uno de: `monorepo-embedded`, `multirepo`)
- `workspace.specs_repository` (ruta relativa al directorio de especificaciones)
- `roles` (mapa con al menos 1 rol definido, conteniendo `name` y una lista de `repositories`)

Además DEBE aceptar los bloques opcionales de primer nivel `technical_context` (RF-CTX-01), `git` (RF-TOP-09), `repositories` (RF-TOP-10), `workflow` (RF-TOP-11) y `delivery` (RF-TOP-14): su ausencia no altera la validación y, cuando están presentes, se validan de forma estricta (claves desconocidas, rutas y valores). Un bloque `baseline` NO es un bloque admitido: se rechaza con el error tipado de RF-TOP-12. Las claves desconocidas fuera de esos bloques se siguen ignorando.
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
- **ENTONCES** la validación falla con el error tipado de RF-TOP-12 y no lo trata como clave desconocida ignorable

### Requirement: Topología Monorepo Embebido (RF-TOP-04)

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

### Requirement: Topología Monorepo Desacoplado (RF-TOP-05) [SUPERSEDED / DEPRECADO]
> [!CAUTION]
> **REQUISITO HISTÓRICO SUPERADO / DEPRECADO:** Este requisito ha sido superado o deprecado.
> Motivo: `monorepo-decoupled` no se comportaba de forma distinta a `multirepo`: ambas separaban el repositorio de especificaciones y se resolvían con la misma lógica, y ninguna superficie de usuario —`axiom init`, el dashboard, el hub— la ofrecía ya. Retirarla elimina un valor de enumeración y mensajes redundantes, sin cambio de comportamiento para `multirepo`.


En la topología `monorepo-decoupled`, el repositorio de especificaciones DEBE residir en un directorio independiente dentro de la carpeta maestra común, distinto del repositorio de código del monorrepo. Esta topología DEBE admitir el patrón canónico: una carpeta maestra con `.axiom-workspace` que apunta a `<specs_repository>/axiom.yaml`, en el que el planning home es `<repositorio de especificaciones>/openspec` (capacidad `planning-home-resolution`, RF-PLH-03).
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

### Requirement: Topología Multirepo Federado (RF-TOP-06)

En la topología `multirepo`, múltiples repositorios de código coexisten bajo la carpeta maestra común. Es REQUISITO OBLIGATORIO que exista un repositorio canónico de especificaciones dentro de dicha carpeta. Esta topología DEBE admitir el patrón canónico: una carpeta maestra no versionada con `.axiom-workspace` que apunta a `<specs_repository>/axiom.yaml`, en el que el planning home es `<repositorio canónico de especificaciones>/openspec` y los verbos SDD resuelven el mismo planning home desde la carpeta maestra, desde el repositorio de especificaciones o desde un repositorio de código (capacidad `planning-home-resolution`, RF-PLH-03 y RF-PLH-08).
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

### Requirement: Reporte de Validación Estructurado (RF-TOP-07)

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
### Requirement: Retirada de la Topología `monorepo-decoupled` con Error Accionable (RF-TOP-08)

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
- **ENTONCES** se rechaza nombrando la topología retirada, `multirepo` y `axiom workspace validate`, conforme a RF-PLH-07
- **Y** no se devuelve ningún status

#### Scenario: Ninguna documentación presenta el valor como válido
- **DADO** los textos de `openspec/specs/`, `docs/manual-de-inicio.md` y `axiom.example.yaml`
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** ninguno presenta `monorepo-decoupled` como valor válido de `workspace.topology`
- **Y** cualquier mención restante lo declara retirado y remite a `multirepo`

---

### Requirement: Bloque `git:` Opcional de `axiom.yaml` (RF-TOP-09)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `git:` con las claves opcionales `integration_branch`, `branch_prefix`, `main_branch` y `promotion`. Sin el bloque, o sin una de las claves, `integration_branch` vale `main` y `branch_prefix` vale `feat`; `main_branch` NO tiene valor por defecto (su ausencia significa que la rama de integración es la única etapa y la rama principal no existe como concepto distinto) y `promotion` NO tiene valor por defecto declarado: cuando hay dos etapas y no se declara, el modo de promoción efectivo es `branch` (RF-BRA-28 de `change-branch-lifecycle`). `integration_branch`, `branch_prefix` y `main_branch` DEBEN ser cadenas no vacías que formen nombres de referencia git válidos (`branch_prefix` sin `/` inicial ni final, sin `..` ni espacios). `promotion` DEBE ser una de las cadenas `branch`, `release` o `ask`; `ask` es válido en la configuración y significa que la persona decide el modo al crear la rama de cada cambio. Una clave desconocida dentro de `git:` y un valor inválido DEBEN reportarse como error nombrando la clave, y el mensaje que enumera las claves admitidas DEBE listar las cuatro. Los valores del bloque son política del proyecto y se versionan: el modo de trabajo y la raíz de los worktrees NO DEBEN declararse en `axiom.yaml` (viven en el hub personal, capacidad `multi-project-hub`).

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

### Requirement: Bloque `repositories:` de Primer Nivel con Ajustes por Repositorio (RF-TOP-10)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `repositories:` con una lista de entradas, cada una con `path` (obligatorio, relativo a la carpeta maestra) y, opcionalmente, `integration_branch`, `branch_prefix` y `promotion` propios del repositorio, con las mismas reglas de validez que `git:` (supuesto S-13): `promotion` es `branch`, `release` o `ask`. `main_branch` NO es una clave de `repositories:`: la rama principal es única del proyecto y declararla en una entrada DEBE reportarse como clave desconocida. Un repositorio sin entrada, o sin una de las claves, hereda el valor del bloque `git:`, campo a campo. `path` NO DEBE ser absoluta ni salir de la carpeta maestra y DEBE existir como directorio; DOS entradas que resuelvan a la misma raíz git DEBEN rechazarse nombrando ambas. Una entrada cuyo `path` no pertenezca a ningún repositorio git NO DEBE ser un error de validación: DEBE emitir una advertencia que la nombre (RF-BRA-06). Una clave desconocida en una entrada DEBE reportarse como error, y el mensaje que enumera las claves admitidas DEBE listar `path`, `integration_branch`, `branch_prefix` y `promotion`. Una entrada con `promotion` cuyo repositorio no tenga dos etapas (sin `git.main_branch` o con `git.main_branch` igual a la rama de integración resuelta de ese repositorio) DEBE emitir una advertencia no bloqueante que la nombre, sin cambiar el estado ni el código de salida. La resolución de la raíz git de cada entrada sigue RF-BRA-06.
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
### Requirement: Bloque `workflow:` Opcional de `axiom.yaml` con el Carril del Proyecto (RF-TOP-11)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `workflow:` con la clave opcional `lane`, cuyos valores válidos son `odd`, `sdd` y `ask`. Sin el bloque, o sin la clave, el carril efectivo del proyecto vale `sdd`. El bloque DEBE decodificarse de forma estricta por lista cerrada de claves, con el mismo patrón que `git:` (RF-TOP-09): una clave desconocida dentro de `workflow:` y un valor de `lane` fuera de los tres admitidos DEBEN reportarse como error accionable que nombre la clave (`workflow.lane` o la clave desconocida) y los valores admitidos, tanto al cargar la configuración como en `axiom workspace validate`, con código de salida `1`. Un `workflow:` que no sea un mapa también DEBE reportarse como error. Las claves desconocidas de primer nivel fuera de los bloques reconocidos se siguen ignorando (con la excepción de `baseline`, que se rechaza con el error tipado de RF-TOP-12): un binario anterior a este cambio DEBE cargar un `axiom.yaml` con el bloque `workflow:` sin error y sin interpretarlo. El carril es política del proyecto y se versiona; la elección del modo de trabajo personal NO DEBE declararse en `axiom.yaml`. El valor por defecto `sdd` rige en `monorepo-embedded` y en `multirepo`; en `multirepo` el carril `odd` sigue estando permitido.
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

### Requirement: Un Bloque `baseline:` Residual se Rechaza con Error Tipado y Guía de Migración (RF-TOP-12)

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

### Requirement: El Almacén de Artefactos Declarado Solo Admite Ficheros: `engram` y `none` se Reportan (RF-TOP-13)

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
### Requirement: Bloque `delivery:` Opcional de `axiom.yaml` con el Umbral del Aviso de Cambio Grande (RF-TOP-14)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `delivery:` con la clave opcional `split_hint_lines`, un entero positivo (mayor o igual que 1) que fija el número de líneas a partir del cual el orquestador avisa de que un cambio es grande (RF-TOP-15). Sin el bloque, o sin la clave, el aviso está desactivado: no hay umbral por defecto y ninguna superficie lo sustituye por un número. El valor `0` NO es una forma de desactivar el aviso: se rechaza; la forma de desactivarlo es no declarar la clave. El bloque DEBE decodificarse de forma estricta por lista cerrada de claves, con el mismo patrón y el mismo canal de error que `git:` (RF-TOP-09) y `workflow:` (RF-TOP-11), tanto al cargar la configuración como en `axiom workspace validate`, con código de salida `1`. DEBEN reportarse como error accionable que nombre la clave: una clave desconocida o repetida dentro de `delivery:`, un `delivery:` que no sea un mapa, y un valor de `split_hint_lines` que no sea un entero, que sea `0` o negativo, o que sea nulo (`delivery.split_hint_lines`). Las claves desconocidas de primer nivel fuera de los bloques reconocidos se siguen ignorando: un binario anterior a este cambio DEBE cargar un `axiom.yaml` con el bloque `delivery:` sin error y sin interpretarlo. `axiom init --force` DEBE preservar un bloque `delivery:` ya declarado, igual que los demás bloques declarados. La clave es política del proyecto y se versiona. El manual de inicio (`docs/manual-de-inicio.md`) DEBE documentar la clave, su dominio y que su ausencia desactiva el aviso.

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

### Requirement: Con `delivery.split_hint_lines` Fijada el Orquestador Avisa una Vez de un Cambio Grande y Nunca Bloquea (RF-TOP-15)

Cuando `axiom.yaml` declara `delivery.split_hint_lines` (RF-TOP-14) y la previsión de líneas del trabajo la supera, el orquestador DEBE emitir **un único aviso** por cambio (SDD) o por feature (ODD) indicando que el cambio es grande y que la persona puede pedir PRs encadenadas, y DEBE continuar sin esperar respuesta. El aviso NUNCA bloquea ni detiene el avance, no cambia la estrategia de entrega (sigue siendo `single-pr` salvo petición expresa de la persona) y no exige ninguna decisión. La previsión que se compara es la estimación cualitativa de tamaño de `sdd-tasks` en SDD y la previsión del documento de la feature en ODD. El orquestador lee la clave directamente de `axiom.yaml` (no se proyecta en `axiom sdd status`). Sin la clave NO hay aviso ni ningún número de comparación: ni la doctrina ni los orquestadores DEBEN contener un umbral por defecto.

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
### Requirement: Bloque `integrations:` Opcional y Estricto de `axiom.yaml` (RF-TOP-16)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `integrations:`, un mapeo `<plugin>: <mapeo de configuración>`. SOLO DEBE admitir identificadores presentes en el catálogo de manifiestos (RF-INT-04) y, dentro de cada uno, solo las claves de configuración de su manifiesto (RF-INT-03). Son error de carga, en `ParseConfig`, `LoadConfig` y `axiom workspace validate`: un plugin desconocido, una clave desconocida, un valor del bloque nulo o que no es un mapeo, una clave duplicada un valor de plugin nulo o que no es un mapeo, un valor de tipo distinto del declarado en el esquema del manifiesto, un valor fuera del enumerado y una clave obligatoria ausente (esta última solo si `enabled` no es `false`). Dentro de cada plugin, además de las claves del manifiesto, se admite la clave reservada `enabled` (booleano estricto, por defecto `true`); un plugin con `enabled: false` no configura nada y no recibe eventos. Los valores por defecto del manifiesto se aplican al proyectar el bloque. Los errores se acumulan todos, con línea y alternativas (con el catálogo vacío, «plugins admitidos: ninguno en esta versión»). `integrations: {}` es válido y no configura nada. El mensaje DEBE incluir la línea y la lista de claves o plugins admitidos, con el mismo formato que el resto de bloques. Sin el bloque, NO DEBE cambiar ningún comportamiento (RF-INT-05). Ningún secreto DEBE poder declararse en este bloque: los secretos no son claves de configuración del manifiesto (RF-INT-03), así que una clave con el nombre de un secreto es una clave desconocida.

(Placement: section "Bloque `workflow:` Opcional de `axiom.yaml` con el Carril del Proyecto (RF-TOP-11)", a continuación)

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

### Requirement: `Locate` Expone el Error del Bloque sin Rechazar y no Bloquea el Archivado (RF-TOP-17)

`workspace.Locate` NO DEBE rechazar por un bloque `integrations:` inválido: DEBE exponerlo como `Declarations.IntegrationsErr`, igual que `git:`, `workflow:` y `sdd:`. El bloque `integrations:` NO DEBE bloquear el archivado (`axiom sdd status` no lo cuenta entre las declaraciones inválidas que bloquean). `axiom init --force` DEBE tratar `integrations:` como bloque propio del usuario y no sobrescribirlo (`declares()` conoce `Integrations`).

#### Scenario: `Locate` con bloque inválido
- **DADO** un `axiom.yaml` con `integrations:` inválido
- **CUANDO** se llama a `workspace.Locate`
- **ENTONCES** no devuelve error por ello
- **Y** `Declarations.IntegrationsErr` contiene el error de RF-TOP-16

#### Scenario: Archivado no bloqueado
- **DADO** un `axiom.yaml` con `integrations:` inválido y un cambio listo para archivar
- **CUANDO** se consulta `axiom sdd status`
- **ENTONCES** la fase de archivado no queda bloqueada por esa declaración

#### Scenario: `init --force` conserva el bloque
- **DADO** un `axiom.yaml` con un bloque `integrations:`
- **CUANDO** se ejecuta `axiom init --force`
- **ENTONCES** el bloque `integrations:` se conserva sin cambios

### Requirement: Plantilla y Manual Documentan `integrations:` (RF-TOP-18)

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
### Requirement: Bloque `acceptance_tests:` Opcional y Estricto de `axiom.yaml`, Solo en `multirepo` (RF-TOP-19)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `acceptance_tests: { enabled: <bool>, environments: [<cadena>, ...] }`. El decodificador DEBE ser estricto, en `ParseConfig`, `LoadConfig` y `axiom workspace validate`, y DEBE acumular todos los errores con su línea:

- una clave distinta de `enabled` y `environments` es un error de clave desconocida, que lista las admitidas;
- un valor del bloque nulo o que no es un mapeo, y una clave duplicada, son error;
- `enabled` DEBE ser un booleano estricto (`true` o `false`); cualquier otro tipo o valor es error y no se asume nunca un valor;
- `environments` DEBE ser una lista de cadenas no vacías; un elemento no cadena o vacío es error, y también lo es un elemento que contenga `;`, porque `;` separa los entornos de cada caso en `pruebas.md`;
- con `enabled: true`, `environments` DEBE estar presente, no vacío y sin duplicados (comparación exacta, sensible a mayúsculas); su ausencia, una lista vacía o un duplicado son error;
- con `enabled: false`, `environments` es opcional y no se valida más allá del tipo.

`acceptance_tests: { enabled: false }` y la ausencia del bloque son equivalentes: no activan nada. El accesor del bloque (patrón de `ExplainerHTML()`) NO DEBE asumir un valor cuando la declaración es inválida: DEBE devolver el error.

`workspace.Validate` DEBE rechazar `enabled: true` cuando la topología no es `multirepo`, en el paso 3b junto a la regla de roles de RF-TOP-04, con un error que nombra la clave y la topología declarada. `enabled: false` y la ausencia del bloque son válidos en cualquier topología. La regla vive en `Validate` y no en la carga, para que `Locate` y el archivado toleren el error (RF-TOP-20).

#### Scenario: Bloque válido en multirepo
- **DADO** un `axiom.yaml` `multirepo` con `acceptance_tests: { enabled: true, environments: [local, dev, pre] }`
- **CUANDO** se carga la configuración y se ejecuta `workspace.Validate`
- **ENTONCES** ambos terminan sin error
- **Y** el accesor devuelve activación `true` y los entornos `local`, `dev` y `pre` en ese orden

#### Scenario: Sin bloque o con `enabled: false`
- **DADO** un `axiom.yaml` sin `acceptance_tests` o con `acceptance_tests: { enabled: false }`
- **CUANDO** se carga la configuración
- **ENTONCES** la carga termina sin error y la función se considera no activa

#### Scenario: Clave desconocida
- **DADO** `acceptance_tests: { enabled: true, environments: [dev], modo: x }`
- **CUANDO** se carga la configuración
- **ENTONCES** falla con un error que nombra `modo`, su línea y las claves admitidas `enabled` y `environments`

#### Scenario: Activo sin entornos o con entornos vacíos
- **DADO** `acceptance_tests: { enabled: true }` o `acceptance_tests: { enabled: true, environments: [] }`
- **CUANDO** se carga la configuración
- **ENTONCES** falla con un error que indica que `environments` debe ser no vacío

#### Scenario: Entornos duplicados
- **DADO** `acceptance_tests: { enabled: true, environments: [dev, pre, dev] }`
- **CUANDO** se carga la configuración
- **ENTONCES** falla con un error que nombra `dev` como duplicado

#### Scenario: `enabled` no booleano o bloque no mapeo
- **DADO** `acceptance_tests: { enabled: "sí" }`, `acceptance_tests:` sin valor o `acceptance_tests: [x]`
- **CUANDO** se carga la configuración
- **ENTONCES** cada caso falla con un error tipado y no se asume ningún valor de `enabled`

#### Scenario: Monorepo rechaza `enabled: true`
- **DADO** un `axiom.yaml` `monorepo-embedded` con `acceptance_tests: { enabled: true, environments: [dev] }`
- **CUANDO** se ejecuta `workspace.Validate`
- **ENTONCES** falla con un error que nombra `acceptance_tests` y la topología `monorepo-embedded`

#### Scenario: Monorepo admite `enabled: false` y la ausencia del bloque
- **DADO** un `axiom.yaml` `monorepo-embedded` con `acceptance_tests: { enabled: false }`, y otro sin el bloque
- **CUANDO** se ejecuta `workspace.Validate`
- **ENTONCES** ambos terminan sin error por esta regla

### Requirement: `Locate` Expone el Error del Bloque sin Rechazar, no Bloquea el Archivado y `init --force` lo Conserva; Plantilla y Manual lo Documentan (RF-TOP-20)

`workspace.Locate` NO DEBE rechazar por un bloque `acceptance_tests:` inválido: DEBE exponerlo como `Declarations.AcceptanceTestsErr`, igual que `git:`, `workflow:`, `sdd:` e `integrations:`. El bloque NO DEBE bloquear el archivado (`axiom sdd status` no lo cuenta entre las declaraciones inválidas que bloquean). `axiom init --force` DEBE tratar `acceptance_tests:` como bloque propio del usuario y no sobrescribirlo (`declares()` conoce el bloque). `axiom.example.yaml` DEBE incluir un bloque `acceptance_tests:` comentado con `enabled` y `environments`, y `docs/manual-de-inicio.md` y `docs/openspec-config.md` DEBEN describirlo: claves, que solo se admite `enabled: true` en `multirepo`, que `environments` es el dominio cerrado de la columna `Entornos` de `pruebas.md` y que sin el bloque no cambia ningún comportamiento. Las pruebas de contrato DEBEN fijarlo.

#### Scenario: `Locate` con bloque inválido
- **DADO** un `axiom.yaml` con `acceptance_tests:` inválido
- **CUANDO** se llama a `workspace.Locate`
- **ENTONCES** no devuelve error por ello
- **Y** `Declarations.AcceptanceTestsErr` contiene el error de RF-TOP-19

#### Scenario: Archivado no bloqueado
- **DADO** un `axiom.yaml` con `acceptance_tests:` inválido y un cambio listo para archivar
- **CUANDO** se consulta `axiom sdd status`
- **ENTONCES** la fase de archivado no queda bloqueada por esa declaración

#### Scenario: `init --force` conserva el bloque
- **DADO** un `axiom.yaml` con un bloque `acceptance_tests:`
- **CUANDO** se ejecuta `axiom init --force`
- **ENTONCES** el bloque se conserva sin cambios

#### Scenario: Plantilla y manual cubiertos por las pruebas de contrato
- **CUANDO** se ejecutan las pruebas de contrato de `axiom.example.yaml`, del manual y de `docs/openspec-config.md`
- **ENTONCES** comprueban la presencia del bloque comentado y de su apartado, y fallan si se eliminan
### Requirement: Catálogo Único de Claves de `axiom.yaml` Derivado del Esquema, Filtrado por Topología e ISO (RF-TOP-21)

El sistema DEBE ofrecer un catálogo de las claves de `axiom.yaml` que se pueden consultar o editar, como única fuente para cualquier pantalla o cliente. Cada entrada DEBE declarar su ruta, etiqueta, descripción, tipo, valores permitidos (si el vocabulario es cerrado), si es editable, el efecto descrito de editarla y las topologías en las que aplica. Los valores permitidos de los vocabularios cerrados (`git.promotion` y `repositories[].promotion`, `workflow.lane`, `sdd.explainer_html`, `roles[].gate_policy`) DEBEN derivarse de las constantes de su paquete dueño (las del esquema de `workspace` y, para `roles[].gate_policy`, las de `multirole`), expuestas como funciones que usan a la vez el validador y el catálogo; el catálogo NO DEBE duplicar esas listas. Una clave nueva de un vocabulario cerrado DEBE aparecer en el catálogo sin editar el catálogo.

Claves editables: `workspace.name`, `git.integration_branch`, `git.branch_prefix`, `git.main_branch`, `git.promotion`, `workflow.lane`, `delivery.split_hint_lines`, `sdd.iso`, `sdd.id_pattern`, `sdd.explainer_html`; y, solo en `multirepo`, `repositories[].integration_branch`, `repositories[].branch_prefix`, `repositories[].promotion`, `acceptance_tests.enabled`, `acceptance_tests.environments` y `roles[].gate_policy` (con `blocking`, `deferred` y `optional`). `sdd.iso` y `sdd.id_pattern` DEBEN llevar un efecto descrito explícito.

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
- **ENTONCES** incluye `workspace.name`, `git.*`, `workflow.lane`, `delivery.split_hint_lines` y `sdd.*`
- **Y** no incluye `roles`, `gate_policy`, `repositories` ni `acceptance_tests`

#### Scenario: Claves por topología en multirepo
- **DADO** un proyecto `multirepo`
- **CUANDO** se consulta el catálogo
- **ENTONCES** incluye `repositories[].{integration_branch,branch_prefix,promotion}`, `acceptance_tests.{enabled,environments}` y `roles[].gate_policy` con `optional` entre sus valores

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

---

### Requirement: Escritor de Round-Trip de `axiom.yaml` que Conserva Comentarios y Orden y Revalida el Documento Completo (RF-TOP-22)

El sistema DEBE ofrecer un escritor que aplique un parche de claves del catálogo sobre `axiom.yaml` operando sobre el árbol YAML, de modo que el resto del documento conserve sus comentarios (de cabecera, de línea y de pie), el orden de las claves y los bloques que no se tocan. El escritor DEBE aplicar solo claves editables del catálogo y rechazar cualquier otra con un error tipado.

El parche es un mapa de claves del catálogo a valores, con `nil` para retirar. Las claves por elemento se direccionan como `repositories[<path>].<campo>` (el selector se compara con la ruta normalizada, `\` a `/` y `path.Clean`) y `roles[<id>].<campo>`. Todos los cambios del parche se aplican sobre el mismo árbol y se validan juntos, como un lote atómico de todo o nada: si uno falla, no se escribe ninguno. La comprobación de tipo y de vocabulario de cada clave se hace contra el vocabulario sin filtrar por ISO; la contradicción ISO la decide la validación del documento resultante, de modo que desactivar ISO y elegir `odd` en el mismo lote es válido.

Un valor `null` para una clave DEBE retirarla del fichero (el motor vuelve a su valor por defecto); si un bloque queda vacío, DEBE borrarse también. Al retirar el último campo de ajuste de una entrada de `repositories`, la entrada se borra, y al quedar vacía la secuencia, el bloque. `workspace.name` es obligatoria y no admite `null`. El escritor DEBE rechazar editar un documento que contenga anclas, alias o claves de mezcla `<<`, más de un documento YAML (reescribirlo borraría los siguientes) o una clave repetida en el mapeo que se edita, con un error tipado en castellano que nombra la construcción y su línea, sin escribir.

El round-trip DEBE conservar el formato del fichero, incluidas las líneas en blanco entre bloques, los finales de línea CRLF y el BOM UTF-8 si el original los tenía: un parche sin efecto NO DEBE alterar el fichero ni un solo byte, y uno con efecto solo cambia las líneas afectadas.

Antes de escribir, el escritor DEBE revalidar el documento completo ya serializado con los mismos decodificadores estrictos y la misma validación del motor (`workspace.ParseConfig`, que incluye `readDeclarationBlocks`, `decodeGit`, `decodeWorkflow` y `checkISOWorkflow`), y las comprobaciones del motor contra el disco (`Validate`, `ValidateDeclarationPaths` y `ValidateRepositoryPolicies`). Un parche válido por bloque que resulte inválido en conjunto DEBE rechazarse. Cualquier error de `ParseConfig` bloquea. De las comprobaciones contra el disco solo bloquean los errores que INTRODUCE el parche, es decir, los presentes en el documento nuevo y ausentes en el original; los errores que el fichero ya tenía en disco se devuelven como avisos (`warnings`) y no impiden guardar. Si la validación bloquea, el fichero NO DEBE modificarse y el error del validador DEBE llegar al llamante en castellano. En particular, `workflow.lane: odd` con `sdd.iso: true` (resultante del parche o ya presente) DEBE rechazarse con la contradicción ISO.

Los errores del escritor DEBEN ser tipados, con los códigos `invalid_request`, `config_missing`, `revision_mismatch`, `unsupported_document`, `unknown_key`, `key_not_editable`, `key_not_applicable`, `invalid_value`, `validation_failed` y `write_failed`, y mensaje en castellano; `validation_failed` lleva una violación por línea.

La escritura DEBE ser atómica: fichero temporal en el mismo directorio y `rename`; un fallo no deja un fichero a medias ni un temporal. El fallo de E/S es `write_failed` y su mensaje DEBE indicar si el reemplazo llegó a producirse. La ruta del fichero DEBE obtenerse de `workspace.Locate(...).ConfigPath`, de modo que en `multirepo` se escriba el `axiom.yaml` del repositorio de specs apuntado y nunca uno inventado en la raíz del código. Lo escrito queda sin confirmar en el árbol de trabajo del repositorio que contiene el fichero: el escritor no ejecuta git.

#### Scenario: El round-trip conserva comentarios y orden (golden)
- **DADO** un `axiom.yaml` comentado de referencia con comentarios de cabecera, de línea y de pie, y bloques en un orden concreto
- **CUANDO** se cambia `workflow.lane` de `ask` a `sdd`
- **ENTONCES** el fichero resultante coincide con el golden: solo cambia esa línea
- **Y** los comentarios, el orden, el sangrado y los bloques ajenos permanecen idénticos

#### Scenario: `null` borra la clave
- **DADO** un `axiom.yaml` con `git.main_branch: trunk` y otras claves en `git`
- **CUANDO** se aplica `git.main_branch: null`
- **ENTONCES** la clave desaparece y las demás claves de `git` se conservan
- **Y** el motor reporta su valor por defecto

#### Scenario: Un bloque que queda vacío se borra
- **DADO** un `axiom.yaml` con `delivery:` que solo contiene `split_hint_lines`
- **CUANDO** se aplica `delivery.split_hint_lines: null`
- **ENTONCES** el bloque `delivery:` desaparece por completo del fichero

#### Scenario: Se rechazan anclas, alias y claves de mezcla
- **DADO** un `axiom.yaml` que usa un ancla `&base`, un alias `*base` o una clave `<<`
- **CUANDO** se intenta aplicar cualquier parche
- **ENTONCES** el escritor devuelve un error tipado en castellano que nombra la construcción no admitida
- **Y** el fichero no se modifica

#### Scenario: Se rechaza un fichero con varios documentos YAML
- **DADO** un `axiom.yaml` con dos documentos YAML
- **CUANDO** se aplica cualquier parche
- **ENTONCES** se rechaza con `unsupported_document` sin escribir
- **Y** ambos documentos permanecen en el fichero

#### Scenario: El lote es atómico, todo o nada
- **DADO** un parche con un cambio válido y otro inválido
- **CUANDO** se aplica
- **ENTONCES** se rechaza entero y el fichero no cambia

#### Scenario: Se conservan líneas en blanco, CRLF y BOM
- **DADO** un `axiom.yaml` con líneas en blanco entre bloques, finales `\r\n` y BOM UTF-8
- **CUANDO** se aplica un parche que cambia una sola línea
- **ENTONCES** solo cambia esa línea y se conservan las líneas en blanco, los finales de línea y el BOM
- **Y** un parche sin efecto no escribe y deja el fichero byte a byte idéntico

#### Scenario: Un error previo del motor no bloquea y vuelve como aviso
- **DADO** un `axiom.yaml` que ya incumple una comprobación del motor contra el disco (por ejemplo, un rol especializado en `monorepo-embedded`)
- **CUANDO** se aplica un parche válido que no toca ese error
- **ENTONCES** el parche se escribe
- **Y** el error previo se devuelve como aviso

#### Scenario: Un error nuevo del motor bloquea
- **DADO** un `axiom.yaml` válido
- **CUANDO** el parche introduce una entrada de `repositories` hacia una ruta inexistente
- **ENTONCES** se rechaza con `validation_failed` sin escribir

#### Scenario: Valor inválido rechazado con el error del validador
- **DADO** un `axiom.yaml` válido
- **CUANDO** se aplica `git.promotion: siempre`
- **ENTONCES** el escritor devuelve el error del validador del motor en castellano
- **Y** el fichero conserva su contenido y su revisión

#### Scenario: Un parche válido por bloque pero inválido en conjunto se rechaza
- **DADO** un proyecto con `sdd.iso: true` y `workflow.lane: sdd`
- **CUANDO** se aplica `workflow.lane: odd`
- **ENTONCES** se rechaza con la contradicción ISO sin escribir
- **Y** aplicar `sdd.iso: true` sobre un proyecto con `workflow.lane: odd` se rechaza del mismo modo

#### Scenario: Clave no editable rechazada
- **DADO** un parche que incluye `workspace.topology` o `integrations.<algo>`
- **CUANDO** se aplica
- **ENTONCES** se rechaza con un error tipado de clave no editable
- **Y** el fichero no cambia

#### Scenario: Escritura atómica
- **DADO** un parche válido
- **CUANDO** se escribe
- **ENTONCES** el contenido nuevo se escribe en un temporal del mismo directorio y se renombra sobre `axiom.yaml`
- **Y** si el temporal no puede escribirse o renombrarse, `axiom.yaml` conserva su contenido anterior y no queda ningún temporal

#### Scenario: Multirepo escribe el fichero del repositorio de specs
- **DADO** un `multirepo` cuyo `axiom.yaml` del código apunta al de specs
- **CUANDO** se aplica un parche
- **ENTONCES** se escribe la ruta que devuelve `Locate().ConfigPath`
- **Y** no se crea ni modifica ningún `axiom.yaml` en el repositorio de código

---

### Requirement: Revisión SHA-256 del Fichero con Recomprobación Justo Antes del Reemplazo (RF-TOP-23)

La lectura del catálogo DEBE devolver la revisión actual de `axiom.yaml`, el SHA-256 de su contenido. Toda escritura DEBE exigir la revisión esperada, es decir, la que el llamante leyó. Si la revisión esperada no coincide con la del fichero, el escritor DEBE rechazar con un error tipado de revisión obsoleta, sin escribir. Tras serializar y validar, y justo antes del `rename`, el escritor DEBE volver a calcular la revisión del fichero y, si cambió, rechazar sin escribir y sin dejar temporales. Una escritura correcta DEBE devolver la revisión nueva. Si el resultado es idéntico al original, no se escribe. Limitación declarada: ningún sistema operativo ofrece un `rename` condicional, así que subsiste una ventana residual de milisegundos entre la recomprobación y el reemplazo en la que una edición externa podría perderse; esa ventana se acepta y se documenta, y la recomprobación se hace justo antes de entregar el reemplazo atómico, no entre el temporal y su `rename`.

#### Scenario: Revisión coincidente permite escribir
- **DADO** una lectura que devolvió la revisión R
- **CUANDO** se escribe con la revisión esperada R y un parche válido
- **ENTONCES** se escribe el fichero
- **Y** la respuesta incluye la revisión nueva, distinta de R

#### Scenario: Revisión obsoleta rechazada sin escribir
- **DADO** una lectura con la revisión R y una edición externa posterior en el editor
- **CUANDO** se escribe con la revisión esperada R
- **ENTONCES** se rechaza con el error de revisión obsoleta
- **Y** el fichero conserva el contenido de la edición externa

#### Scenario: Cambio externo entre la validación y el reemplazo
- **DADO** una escritura con revisión coincidente y parche válido
- **CUANDO** el fichero cambia después de validar y antes del `rename`
- **ENTONCES** la recomprobación detecta la diferencia y rechaza sin escribir
- **Y** no queda ningún temporal

#### Scenario: Escritura sin revisión esperada
- **DADO** un parche válido sin revisión esperada
- **CUANDO** se intenta escribir
- **ENTONCES** se rechaza como petición incompleta sin escribir
