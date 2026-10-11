# Delta para Topología de Workspace y CLI de Axiom (INC-24)

> **Incremento:** `inc-24-spec-baseline-profile` · **Tipo:** delta sobre la capacidad viva `workspace-topology`

Cambios de este delta:

- REQ-1.2: `axiom workspace validate` valida además los bloques opcionales `baseline` y `technical_context` (REQ-24.17 y REQ-24.27) y su compatibilidad con el almacén de artefactos (REQ-24.62).
- REQ-2.1: el esquema de `axiom.yaml` admite los bloques opcionales `baseline` y `technical_context` (REQ-24.13 y REQ-24.26).
- REQ-2.3 y REQ-2.4: el patrón canónico con `.axiom-workspace` se admite en `monorepo-decoupled` y `multirepo`, y el planning home es `<repositorio de especificaciones>/openspec` (REQ-24.3).

## MODIFIED Requirements

### Requirement: Subcomando axiom workspace validate (REQ-1.2)

El binario `axiom` DEBE proporcionar el subcomando `axiom workspace validate`, el cual evalúa el archivo `axiom.yaml` del directorio actual (o de la ruta especificada mediante `--path`) y verifica la conformidad de la topología y de los repositorios asociados a los roles. Además DEBE validar los bloques opcionales `baseline` y `technical_context` de `axiom.yaml`: la sintaxis y la contención de rutas se reportan como errores y las rutas de contexto técnico inexistentes como advertencias que no alteran el resultado. También DEBE reportar como error que una baseline del proyecto o un contexto técnico mantenido se declaren con un almacén de artefactos sin ficheros (`engram` o `none`, REQ-24.62).
(Previously: validaba solo la topología y los repositorios asociados a los roles)

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

---

### Requirement: Estructura del Esquema axiom.yaml (REQ-2.1)

El motor de configuración DEBE deserializar y validar el archivo `axiom.yaml`, exigiendo los campos obligatorios:
- `workspace.name` (cadena no vacía)
- `workspace.topology` (uno de: `monorepo-embedded`, `monorepo-decoupled`, `multirepo`)
- `workspace.specs_repository` (ruta relativa al directorio de especificaciones)
- `roles` (mapa con al menos 1 rol definido, conteniendo `name` y una lista de `repositories`)

Además DEBE aceptar los bloques opcionales de primer nivel `baseline` (REQ-24.13) y `technical_context` (REQ-24.26): su ausencia no altera la validación y, cuando están presentes, se validan de forma estricta (claves desconocidas, rutas y perfiles). Las claves desconocidas fuera de esos bloques se siguen ignorando.
(Previously: el esquema no contemplaba bloques opcionales de declaración de baseline ni de contexto técnico)

#### Scenario: Archivo axiom.yaml completo y bien formado
- **DADO** un contenido YAML con todos los campos obligatorios
- **CUANDO** se invoca `LoadConfig`
- **ENTONCES** se obtiene una estructura `WorkspaceConfig` poblada correctamente sin errores

#### Scenario: Topología desconocida o ausente
- **DADO** un archivo `axiom.yaml` con `workspace.topology: "desconocido"`
- **CUANDO** se intenta cargar o validar la configuración
- **ENTONCES** se devuelve un error descriptivo indicando que la topología debe ser `monorepo-embedded`, `monorepo-decoupled` o `multirepo`

#### Scenario: Lista de roles vacía
- **DADO** un archivo `axiom.yaml` sin ningún rol declarado
- **CUANDO** se valida la configuración
- **ENTONCES** se devuelve un error indicando que al menos un rol de desarrollo DEBE estar configurado

#### Scenario: Los bloques opcionales ausentes no alteran la validación
- **DADO** un `axiom.yaml` con todos los campos obligatorios y sin `baseline` ni `technical_context`
- **CUANDO** se invoca `LoadConfig`
- **ENTONCES** se obtiene la misma estructura que antes, con los bloques opcionales vacíos y sin errores

#### Scenario: Los bloques opcionales presentes y válidos se cargan
- **DADO** un `axiom.yaml` con los campos obligatorios y un bloque `baseline` y otro `technical_context` válidos
- **CUANDO** se invoca `LoadConfig`
- **ENTONCES** la estructura `WorkspaceConfig` incluye ambos bloques poblados

#### Scenario: Clave desconocida dentro de un bloque nuevo
- **DADO** un `axiom.yaml` con una clave desconocida dentro de `baseline` o de `technical_context`
- **CUANDO** se invoca `LoadConfig`
- **ENTONCES** se devuelve un error que nombra la clave desconocida

---

### Requirement: Topología Monorepo Desacoplado (REQ-2.3)

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

