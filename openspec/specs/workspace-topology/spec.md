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

El binario `axiom` DEBE proporcionar el subcomando `axiom workspace validate`, el cual evalúa el archivo `axiom.yaml` del directorio actual (o de la ruta especificada mediante `--path`) y verifica la conformidad de la topología y de los repositorios asociados a los roles. Además DEBE validar los bloques opcionales `baseline`, `technical_context`, `git` y `repositories` de `axiom.yaml`: la sintaxis y la contención de rutas se reportan como errores y las rutas de contexto técnico inexistentes como advertencias que no alteran el resultado. También DEBE reportar como error que una baseline del proyecto o un contexto técnico mantenido se declaren con un almacén de artefactos sin ficheros (`engram` o `none`, REQ-24.62), y DEBE rechazar la topología retirada `monorepo-decoupled` con el mensaje accionable de REQ-25.25.
(Previously: validaba los bloques `baseline` y `technical_context` pero no `git` ni `repositories`, y la topología `monorepo-decoupled` era válida)

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

#### Scenario: Declaración de baseline inválida finaliza con error
- **DADO** un `axiom.yaml` con un perfil de requisito sin los grupos `id` y `title`, con una identidad desconocida o con un documento inexistente
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida detalla en español el dominio o la ruta y la regla incumplida
- **Y** el proceso finaliza con código de salida `1`

#### Scenario: Ruta de contexto técnico inexistente solo avisa
- **DADO** un `axiom.yaml` conforme cuyo `technical_context.maintained.paths` incluye una ruta que no existe
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida incluye una advertencia que nombra la ruta y reporta `COMPLIANT`
- **Y** el código de salida del proceso es `0`

#### Scenario: Declaración incompatible con el almacén de artefactos finaliza con error
- **DADO** un `axiom.yaml` con `baseline:` válido y un `openspec/config.yaml` con `artifact_store: engram`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida detalla en español que la baseline del proyecto exige un almacén de ficheros, y la continuación
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

---

## 2. Capacidad: `workspace-topology-engine`

El paquete `internal/workspace` implementa la lógica de dominio para la lectura de `axiom.yaml` y la verificación determinista de la estructura de repositorios y roles bajo la carpeta maestra.

### Requirement: Estructura del Esquema axiom.yaml (REQ-2.1)

El motor de configuración DEBE deserializar y validar el archivo `axiom.yaml`, exigiendo los campos obligatorios:
- `workspace.name` (cadena no vacía)
- `workspace.topology` (uno de: `monorepo-embedded`, `multirepo`)
- `workspace.specs_repository` (ruta relativa al directorio de especificaciones)
- `roles` (mapa con al menos 1 rol definido, conteniendo `name` y una lista de `repositories`)

Además DEBE aceptar los bloques opcionales de primer nivel `baseline` (REQ-24.13), `technical_context` (REQ-24.26), `git` (REQ-25.26) y `repositories` (REQ-25.27): su ausencia no altera la validación y, cuando están presentes, se validan de forma estricta (claves desconocidas, rutas y perfiles). Las claves desconocidas fuera de esos bloques se siguen ignorando.
(Previously: `workspace.topology` admitía también `monorepo-decoupled` y el esquema no contemplaba los bloques opcionales `git` ni `repositories`)

#### Scenario: Archivo axiom.yaml completo y bien formado
- **DADO** un contenido YAML con todos los campos obligatorios
- **CUANDO** se invoca `LoadConfig`
- **ENTONCES** se obtiene una estructura `WorkspaceConfig` poblada correctamente sin errores

#### Scenario: Topología desconocida o ausente
- **DADO** un archivo `axiom.yaml` con `workspace.topology: "desconocido"`
- **CUANDO** se intenta cargar o validar la configuración
- **ENTONCES** se devuelve un error descriptivo indicando que la topología debe ser `monorepo-embedded` o `multirepo`

#### Scenario: Lista de roles vacía
- **DADO** un archivo `axiom.yaml` sin ningún rol declarado
- **CUANDO** se valida la configuración
- **ENTONCES** se devuelve un error indicando que al menos un rol de desarrollo DEBE estar configurado

#### Scenario: Los bloques opcionales ausentes no alteran la validación
- **DADO** un `axiom.yaml` con todos los campos obligatorios y sin `baseline`, `technical_context`, `git` ni `repositories`
- **CUANDO** se invoca `LoadConfig`
- **ENTONCES** se obtiene la misma estructura que antes, con los bloques opcionales vacíos y sin errores

#### Scenario: Los bloques opcionales presentes y válidos se cargan
- **DADO** un `axiom.yaml` con los campos obligatorios y un bloque `baseline`, otro `technical_context`, otro `git` y otro `repositories` válidos
- **CUANDO** se invoca `LoadConfig`
- **ENTONCES** la estructura `WorkspaceConfig` incluye los cuatro bloques poblados

#### Scenario: Clave desconocida dentro de un bloque nuevo
- **DADO** un `axiom.yaml` con una clave desconocida dentro de `baseline`, `technical_context`, `git` o de una entrada de `repositories`
- **CUANDO** se invoca `LoadConfig`
- **ENTONCES** se devuelve un error que nombra la clave desconocida

#### Scenario: Una versión anterior del binario tolera los bloques nuevos
- **DADO** un `axiom.yaml` con los bloques `git` y `repositories`
- **CUANDO** lo carga un binario anterior a esta capacidad
- **ENTONCES** ignora ambos bloques como claves desconocidas de primer nivel y carga el resto sin error

---

### Requirement: Topología Monorepo Embebido (REQ-2.2)

En la topología `monorepo-embedded`, el repositorio de especificaciones DEBE apuntar a la raíz del propio monorrepo (`"."`) o a una subcarpeta interna existente (como `openspec` o `specs`).

#### Scenario: Monorepo embebido con specs en raíz o subdirectorio válido
- **DADO** una configuración con topología `monorepo-embedded` y `specs_repository: "."`
- **Y** la carpeta existe dentro del espacio de trabajo
- **CUANDO** se ejecuta la validación de topología
- **ENTONCES** la topología se marca como válida y conforme

---

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

`axiom.yaml` PUEDE declarar un bloque de primer nivel `git:` con las claves opcionales `integration_branch` y `branch_prefix`. Sin el bloque, o sin una de las claves, `integration_branch` vale `main` y `branch_prefix` vale `feat`. Ambos valores DEBEN ser cadenas no vacías que formen nombres de referencia git válidos (`branch_prefix` sin `/` inicial ni final, sin `..` ni espacios). Una clave desconocida dentro de `git:` y un valor inválido DEBEN reportarse como error nombrando la clave. Los valores del bloque son política del proyecto y se versionan: el modo de trabajo y la raíz de los worktrees NO DEBEN declararse en `axiom.yaml` (viven en el hub personal, capacidad `multi-project-hub`).

#### Scenario: Sin bloque rigen los valores por defecto
- **DADO** un `axiom.yaml` sin `git:`
- **CUANDO** se resuelve la política de ramas
- **ENTONCES** la rama de integración es `main` y el prefijo es `feat`

#### Scenario: Bloque parcial completa con los valores por defecto
- **DADO** un `axiom.yaml` con `git: {integration_branch: develop}`
- **CUANDO** se resuelve la política de ramas
- **ENTONCES** la rama de integración es `develop` y el prefijo es `feat`

#### Scenario: Valor inválido
- **DADO** un `axiom.yaml` con `git.branch_prefix: "inc//"`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `git.branch_prefix` y la regla incumplida
- **Y** el código de salida es `1`

#### Scenario: Clave desconocida en `git:`
- **DADO** un `axiom.yaml` con `git: {worktree_mode: worktree}`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra la clave `worktree_mode` como desconocida
- **Y** el código de salida es `1`

---

### Requirement: Bloque `repositories:` de Primer Nivel con Ajustes por Repositorio (REQ-25.27)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `repositories:` con una lista de entradas, cada una con `path` (obligatorio, relativo a la carpeta maestra) y, opcionalmente, `integration_branch` y `branch_prefix` propios del repositorio, con las mismas reglas de validez que `git:` (supuesto S-13). Un repositorio sin entrada, o sin una de las claves, hereda el valor del bloque `git:`. `path` NO DEBE ser absoluta ni salir de la carpeta maestra y DEBE existir como directorio; DOS entradas que resuelvan a la misma raíz git DEBEN rechazarse nombrando ambas. Una entrada cuyo `path` no pertenezca a ningún repositorio git NO DEBE ser un error de validación: DEBE emitir una advertencia que la nombre (REQ-25.6). Una clave desconocida en una entrada DEBE reportarse como error. La resolución de la raíz git de cada entrada sigue REQ-25.6.

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
