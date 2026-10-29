# Especificación Viva: Hub Multi-Proyecto, Selector Dinámico en Dashboard Web y CLI axiom init (INC-08)

## Propósito

Definir formal y deterministamente los requerimientos funcionales, no funcionales y escenarios BDD para dotar a Axiom de una arquitectura multi-proyecto de nivel empresarial: registro global central de workspaces (`~/.axiom/workspaces.json`), comando CLI `axiom init` con auto-detección tecnológica, comandos CLI `axiom project`, y un Dashboard Web (`axiom ui`) transformado en Hub con selector dinámico de workspaces en tiempo real y pantalla de bienvenida para repositorios sin inicializar.

---

## 1. Capacidad: `multi-project-hub`

El paquete `internal/hub` provee persistencia, auditoría y gestión de múltiples workspaces en la máquina del desarrollador.

### Requirement: Registro Global Centralizado de Workspaces (REQ-1.1)

El sistema DEBE persistir la lista de proyectos conocidos en un archivo JSON ubicado en el perfil de usuario: `$HOME/.axiom/workspaces.json` (o `%USERPROFILE%\.axiom\workspaces.json` en Windows). El archivo DEBE almacenar:
- Versión del esquema (`"1.0"`).
- ID o ruta del proyecto activo por defecto (`active_workspace`).
- Lista de proyectos registrados (`workspaces`), donde cada entrada contiene:
  - `id`: Identificador alfanumérico único derivado del nombre o slug del directorio.
  - `name`: Nombre descriptivo del proyecto.
  - `path`: Ruta absoluta canónica en el sistema de archivos local.
  - `topology`: Tipo de topología (`monorepo-embedded`, `multirepo`, o `unconfigured`).
  - `registered_at`: Timestamp RFC3339 de registro.
  - `last_accessed`: Timestamp RFC3339 del último acceso.

#### Scenario: Creación del archivo de registro en el primer uso
- **DADO** que no existe el directorio ni el archivo `~/.axiom/workspaces.json`
- **CUANDO** se invoca cualquier operación del gestor de Hub
- **ENTONCES** crea automáticamente el directorio `.axiom` y el archivo `workspaces.json` inicial con un array vacío de proyectos
- **Y** no produce ningún fallo de ejecución

#### Scenario: Registro idempotente de un proyecto existente
- **DADO** un proyecto ubicado en `C:\repos\axiom`
- **CUANDO** se llama a `Register(path, name, topology)`
- **ENTONCES** si el proyecto ya estaba registrado por su ruta normalizada, actualiza su nombre, topología y `last_accessed` sin duplicar la entrada
- **Y** persiste el cambio de forma atómica

#### Scenario: Desregistro de un proyecto
- **DADO** un proyecto registrado con ID `ludeka`
- **CUANDO** se ejecuta `Unregister("ludeka")`
- **ENTONCES** elimina la entrada de `workspaces.json`
- **Y** no elimina ni modifica ningún archivo físico del disco del proyecto

---

### Requirement: Conmutación y Resolución de Proyecto Activo (REQ-1.2)

El gestor DEBE permitir consultar y establecer cuál es el proyecto activo del desarrollador. Si el proyecto marcado como activo ya no existe físicamente en el disco, el gestor DEBE realizar un fallback inteligente hacia el directorio actual si contiene `axiom.yaml`, o hacia el primer proyecto válido del catálogo.

#### Scenario: Conmutación de proyecto activo
- **DADO** un Hub con dos proyectos registrados: `axiom` y `ludeka`
- **CUANDO** se invoca `SetActive("ludeka")`
- **ENTONCES** el campo `active_workspace` pasa a ser el ID de `ludeka`
- **Y** su timestamp `last_accessed` se actualiza al momento actual

#### Scenario: Fallback cuando la ruta configurada fue eliminada
- **DADO** un proyecto activo cuya ruta fue borrada del disco
- **CUANDO** se invoca `GetActive()`
- **ENTONCES** advierte la inexistencia del directorio y retorna el primer proyecto disponible o el directorio de trabajo actual

---

## 2. Capacidad: `project-initializer`

El gestor provee inicialización asistida y automática de proyectos (`axiom init`), permitiendo adoptar proyectos existentes o inicializar nuevos sin fricción.

### Requirement: Detección Heurística de Tecnologías (REQ-2.1)

El inicializador DEBE inspeccionar los archivos del repositorio para deducir la tecnología principal y configurar adecuadamente los roles y herramientas en `axiom.yaml`:
- Si detecta `go.mod` ➔ Tecnología: `go` / `go-test`.
- Si detecta `package.json` ➔ Tecnología: `nodejs` / `typescript` (o `react`/`vue`/`svelte` si están en dependencias).
- Si detecta `*.csproj` o `*.sln` ➔ Tecnología: `dotnet` / `csharp`.
- Si detecta `Cargo.toml` ➔ Tecnología: `rust`.
- Si detecta `pyproject.toml` o `requirements.txt` ➔ Tecnología: `python`.

#### Scenario: Detección automática en proyecto Go
- **DADO** un directorio con un archivo `go.mod`
- **CUANDO** se ejecuta el detector tecnológico
- **ENTONCES** retorna `go` como tecnología base del rol `core` y `go-test` para el rol `qa`

#### Scenario: Detección automática en proyecto .NET / C#
- **DADO** un directorio con un archivo `Ludeka.sln` o `*.csproj`
- **CUANDO** se ejecuta el detector tecnológico
- **ENTONCES** retorna `csharp` y `dotnet` en la configuración inicial

---

### Requirement: Generación de Andamiaje `axiom.yaml` y Registro Automático (REQ-2.2)

Al ejecutar `axiom init` sobre un directorio:
1. Si NO existe `axiom.yaml`, genera un archivo canónico con el nombre del proyecto, topología seleccionada (`monorepo-embedded`), repositorio de especificaciones (`openspec`), roles detectados (`core` y `qa`) y gobernanza en español.
2. Crea las carpetas base necesarias: `.axiom/inbox/skills/`, `openspec/specs/` y `openspec/changes/`, salvo que `axiom.yaml` ya declare una baseline del proyecto (`baseline:`): en ese caso NO crea `openspec/specs/` ni `openspec/INDEX.md` (tampoco con el perfil `knowledge`), informa de la omisión y continúa con el resto del andamiaje (REQ-24.23).
3. Registra automáticamente el proyecto en `~/.axiom/workspaces.json` y lo marca como proyecto activo.
4. Si YA existe `axiom.yaml`, no lo sobreescribe; valida su estructura (incluidos los bloques opcionales `baseline` y `technical_context`) y lo registra en el Hub global si no estaba registrado.

Antes de escribir nada, `axiom init` DEBE rechazarse cuando el propio directorio contiene un puntero `.axiom-workspace` cuyo planning home, resuelto con el primitivo de la capacidad `planning-home-resolution` (REQ-24.1), es distinto de `<directorio>/openspec`: crear ese directorio produciría un segundo planning home y la ambigüedad de REQ-24.6 en todos los verbos. La comparación es entre rutas y NO exige que `<directorio>/openspec` exista, porque `init` es quien lo crea; la existencia del repositorio de especificaciones que exige REQ-24.7 (caso 4) es una condición del primitivo y no de esta comparación. El rechazo DEBE aplicarse con cualquier perfil, finalizar con código de salida distinto de cero, no escribir ni registrar nada y nombrar el puntero, el planning home resuelto y la continuación: inicializar el `workspaceRoot` resuelto con `axiom init --path <workspaceRoot>` o montar la topología a mano según la receta documentada (REQ-24.12).

Cuando el puntero del propio directorio no puede resolverse (por ejemplo, un puntero roto o un `axiom.yaml` apuntado ilegible, REQ-24.7), el primitivo rechaza la resolución y `axiom init` DEBE rechazarse con ese mismo error, que nombra la causa y la continuación, con código de salida distinto de cero y sin escribir ni registrar nada.

`axiom init` NO DEBE rechazarse cuando el puntero resuelve a `<directorio>/openspec` (exista ya ese directorio o no), cuando el propio directorio no contiene puntero (aunque lo contenga un directorio ancestro: el marcador más cercano gana, REQ-24.2) ni cuando no hay ningún marcador: en esos casos se comporta como describen los puntos 1 a 4.

(Previously: creaba siempre `openspec/specs/`, también cuando `axiom.yaml` ya declaraba una baseline del proyecto, y no comprobaba si el directorio contenía un puntero hacia otro planning home ni si ese puntero podía resolverse)

#### Scenario: Inicialización exitosa de proyecto virgen
- **DADO** un directorio limpio sin archivos de Axiom
- **CUANDO** se ejecuta `axiom init --name "MiProyecto"`
- **ENTONCES** se crea `axiom.yaml` con la estructura canónica
- **Y** se crean las carpetas `openspec/specs/`, `openspec/changes/` y `.axiom/inbox/skills/`
- **Y** el proyecto queda registrado en `~/.axiom/workspaces.json`

#### Scenario: Inicialización sobre un `axiom.yaml` que declara una baseline del proyecto
- **DADO** un directorio con un `axiom.yaml` que declara `baseline:` con al menos un documento
- **CUANDO** se ejecuta `axiom init`
- **ENTONCES** no se crea `openspec/specs/` ni `openspec/INDEX.md`
- **Y** se crean `openspec/changes/` y `.axiom/inbox/skills/`
- **Y** la salida informa de que la baseline es del proyecto y el código de salida es `0`
- **Y** el `axiom.yaml` existente no se sobreescribe

#### Scenario: Inicialización rechazada en una carpeta maestra con puntero hacia otro planning home
- **DADO** el workspace canónico de la capacidad `planning-home-resolution`, cuya carpeta maestra `<maestra>` contiene `.axiom-workspace` (`config: repo-specs/axiom.yaml`) y no contiene `openspec/`
- **CUANDO** se ejecuta `axiom init --path <maestra>`, con o sin `--profile=knowledge`
- **ENTONCES** el comando se rechaza con código de salida distinto de cero
- **Y** el mensaje nombra el puntero, el planning home resuelto `<maestra>/repo-specs/openspec` y la continuación `axiom init --path <maestra>/repo-specs` o la receta manual documentada
- **Y** no se crea `<maestra>/openspec/`, ni `axiom.yaml`, ni `.axiom/inbox/skills/`, y `~/.axiom/workspaces.json` no se modifica

#### Scenario: Un puntero propio que no puede resolverse rechaza la inicialización
- **DADO** un directorio con un `.axiom-workspace` cuyo `config` apunta a un fichero inexistente, o a un `axiom.yaml` que no es YAML válido
- **CUANDO** se ejecuta `axiom init --path <directorio>`
- **ENTONCES** el comando se rechaza con código de salida distinto de cero y con el error del primitivo de resolución, que nombra la causa y la continuación
- **Y** no se crea ningún fichero ni directorio y `~/.axiom/workspaces.json` no se modifica

#### Scenario: Un puntero que resuelve a `<directorio>/openspec` no se rechaza aunque ese directorio no exista todavía
- **DADO** una carpeta `M` con `.axiom-workspace` (`config: axiom.yaml`), un `axiom.yaml` `monorepo-embedded` y sin el directorio `M/openspec`
- **CUANDO** se ejecuta `axiom init --path M`
- **ENTONCES** no se rechaza, porque el planning home que resuelve el puntero es `M/openspec` y la comparación no exige que exista
- **Y** la inicialización se comporta como describen los puntos 1 a 4 y crea `M/openspec/`

#### Scenario: Un puntero en un directorio ancestro no impide inicializar
- **DADO** el mismo workspace canónico y el directorio `<maestra>/repo-backend`, que no contiene `.axiom-workspace` propio
- **CUANDO** se ejecuta `axiom init --path <maestra>/repo-backend`
- **ENTONCES** no se rechaza por el puntero de `<maestra>` y la inicialización se comporta como antes de este incremento

---

## 3. Capacidad: `dashboard-project-switcher`

El Dashboard Web (`axiom ui`) opera como un Centro de Control Multi-Proyecto con conmutación en tiempo real.

### Requirement: Endpoints REST de Gestión Multi-Proyecto (REQ-3.1)

El servidor HTTP del dashboard DEBE exponer endpoints bajo `/api/projects`:
- `GET /api/projects`: Retorna el listado de todos los proyectos registrados, indicando cuál es el activo, sus rutas, si tienen `axiom.yaml` válido y sus preferencias personales de worktree (REQ-25.29).
- `POST /api/projects/switch`: Recibe `{"id": "..."}` o `{"path": "..."}` y conmuta el workspace activo del servicio en caliente sin requerir reinicio del servidor.
- `POST /api/projects/add`: Recibe `{"path": "...", "name": "..."}` y, opcionalmente, las preferencias personales de worktree, para incorporar una carpeta existente al registro.
- `POST /api/projects/init`: Recibe parámetros de inicialización y, opcionalmente, las preferencias personales de worktree, y ejecuta `axiom init` sobre la ruta actual o indicada.
(Previously: `add` e `init` no admitían preferencias y el listado no las devolvía)

#### Scenario: Conmutación en caliente de workspace en la UI
- **DADO** el dashboard ejecutándose en el puerto 8080 mostrando el proyecto `Axiom`
- **CUANDO** el cliente envía `POST /api/projects/switch` con el ID de `Ludeka`
- **ENTONCES** las subsiguientes llamadas a `/api/workspace`, `/api/increments` y `/api/archive/specs` devuelven inmediatamente los datos del proyecto `Ludeka`

#### Scenario: Registro con preferencias personales
- **DADO** el dashboard en ejecución
- **CUANDO** el cliente envía `POST /api/projects/add` con una ruta, un nombre y el modo `in_place`
- **ENTONCES** el proyecto queda registrado con el modo `in_place`
- **Y** `GET /api/projects` devuelve ese modo para el proyecto

#### Scenario: Registro sin preferencias aplica los valores por defecto
- **DADO** el dashboard en ejecución
- **CUANDO** el cliente envía `POST /api/projects/add` sin preferencias
- **ENTONCES** el proyecto queda registrado sin preferencias guardadas y su modo efectivo es `worktree`

---

### Requirement: Componente UI Selector de Proyectos y Pantalla de Bienvenida (REQ-3.2)

La interfaz web (SPA) DEBE incorporar:
1. **Selector Desplegable en el Navbar:** Muestra el nombre del proyecto activo actual con un icono de switch. Al desplegarlo, lista los proyectos disponibles con acceso rápido para alternar entre ellos y un botón "+ Añadir Proyecto".
2. **Modal de Registro de Proyecto:** Formulario simple para ingresar la ruta de un repositorio y agregarlo al catálogo, con los campos de preferencias personales de worktree: el modo (`worktree` por defecto, `in_place` o `ask`) y la raíz de worktrees, mostrando el valor por defecto que se aplicará si se deja vacía.
3. **Estado "Zero-Config / Welcome":** Si el proyecto seleccionado o la carpeta donde se lanzó la UI no cuenta con `axiom.yaml`, la UI NO muestra un error 500 ni pantalla en blanco; muestra una tarjeta visual informativa:
   - *"Este repositorio no está gobernado con Axiom aún."*
   - *"Tecnología detectada: [X]."*
   - Botón de acción: *"[ Inicializar Proyecto con Axiom ]"*.
(Previously: el modal de registro solo pedía la ruta del repositorio)

#### Scenario: Navegación a proyecto no inicializado
- **DADO** que la UI se abre en una carpeta sin `axiom.yaml`
- **CUANDO** carga la página en el navegador
- **ENTONCES** se presenta la pantalla de bienvenida con el botón para inicializar
- **Y** al pulsar el botón, invoca `/api/projects/init` y recarga la UI con el proyecto ya operativo

#### Scenario: El modal de registro pide las preferencias personales
- **DADO** la SPA cargada
- **CUANDO** el usuario abre el modal "+ Añadir Proyecto"
- **ENTONCES** el formulario incluye el selector de modo con `worktree` preseleccionado y el campo de raíz de worktrees con su valor por defecto visible
- **Y** al confirmar, la petición a `/api/projects/add` incluye las preferencias elegidas

### Requirement: Preferencias Personales de Worktree por Proyecto en el Hub (REQ-25.29)

Además de los campos de REQ-1.1, cada proyecto registrado en `~/.axiom/workspaces.json` PUEDE almacenar dos preferencias personales opcionales: el **modo de trabajo** (`worktree`, `in_place` o `ask`) y la **raíz de worktrees** (ruta absoluta). Las preferencias son personales y NO DEBEN escribirse nunca en `axiom.yaml` ni en ningún fichero versionado. Sin preferencias guardadas para un proyecto, estén o no registrado, el modo efectivo DEBE ser `worktree` y la raíz efectiva DEBE ser `<padre de la carpeta maestra>/<nombre de la carpeta maestra>-wt`. Un modo distinto de los tres valores, o una raíz que no sea una ruta absoluta, DEBEN rechazarse con un mensaje en español, sin modificar el registro. Registrar de nuevo un proyecto ya registrado (REQ-1.1, registro idempotente) NO DEBE borrar ni restablecer sus preferencias; desregistrarlo SÍ DEBE eliminarlas junto con su entrada. Las preferencias NO DEBEN cambiar la versión del esquema (`"1.0"`): un binario anterior las ignora.

#### Scenario: Sin preferencias el modo efectivo es `worktree`
- **DADO** un proyecto registrado sin preferencias guardadas
- **CUANDO** se consulta su modo y su raíz de worktrees efectivos
- **ENTONCES** el modo es `worktree`
- **Y** la raíz es `<padre de la carpeta maestra>/<nombre de la carpeta maestra>-wt`

#### Scenario: Proyecto no registrado también recibe los valores por defecto
- **DADO** un proyecto que no está en `workspaces.json`
- **CUANDO** `axiom branch ensure` consulta su modo de trabajo
- **ENTONCES** el modo efectivo es `worktree`

#### Scenario: Persistencia de las preferencias
- **DADO** un proyecto registrado
- **CUANDO** se guarda el modo `ask` y la raíz `D:\worktrees\axiom`
- **ENTONCES** `workspaces.json` contiene ambas preferencias en la entrada de ese proyecto
- **Y** `axiom.yaml` no se modifica

#### Scenario: Modo inválido rechazado
- **DADO** un proyecto registrado
- **CUANDO** se intenta guardar el modo `siempre`
- **ENTONCES** se rechaza con un mensaje en español que lista los tres modos válidos
- **Y** el registro no se modifica

#### Scenario: Raíz relativa rechazada
- **DADO** un proyecto registrado
- **CUANDO** se intenta guardar la raíz `../wt`
- **ENTONCES** se rechaza indicando que la raíz debe ser una ruta absoluta

#### Scenario: Re-registrar conserva las preferencias
- **DADO** un proyecto registrado con el modo `in_place`
- **CUANDO** se ejecuta de nuevo `Register(path, name, topology)` para ese proyecto
- **ENTONCES** el modo sigue siendo `in_place`

#### Scenario: Desregistrar elimina las preferencias
- **DADO** un proyecto registrado con preferencias
- **CUANDO** se ejecuta `Unregister`
- **ENTONCES** la entrada y sus preferencias desaparecen de `workspaces.json`

#### Scenario: Un binario anterior ignora las preferencias
- **DADO** un `workspaces.json` con entradas que incluyen las preferencias
- **CUANDO** lo carga un binario anterior a esta capacidad
- **ENTONCES** carga el registro sin error y conserva el resto de campos

---

### Requirement: Las Preferencias se Piden al Registrar y se Pueden Editar (REQ-25.30)

Al registrar un proyecto (`axiom init`, `axiom project add` y el registro del dashboard), el sistema DEBE pedir las preferencias personales de REQ-25.29 ofreciendo los valores por defecto. En la CLI, cuando hay una terminal interactiva, DEBE preguntar con el valor por defecto preseleccionado; sin terminal interactiva, o con las opciones de línea de comandos que fije `sdd-design` (una por preferencia), NO DEBE preguntar y DEBE usar el valor indicado o el valor por defecto, para no romper la automatización. Las preferencias DEBEN poder editarse después desde el dashboard (capacidad `local-web-dashboard`, REQ-25.33). El registro NO DEBE fallar por no indicar preferencias.

#### Scenario: Registro interactivo pregunta con valores por defecto
- **DADO** una terminal interactiva y un proyecto sin registrar
- **CUANDO** se ejecuta `axiom project add <ruta>`
- **ENTONCES** la CLI pregunta el modo (con `worktree` preseleccionado) y la raíz (mostrando el valor por defecto)
- **Y** al aceptar los valores por defecto el proyecto queda registrado sin preferencias guardadas

#### Scenario: Registro sin terminal no pregunta
- **DADO** una ejecución sin terminal interactiva
- **CUANDO** se ejecuta `axiom init`
- **ENTONCES** la CLI no espera entrada alguna
- **Y** el proyecto queda registrado con el modo efectivo `worktree`

#### Scenario: Registro con las opciones de preferencia
- **DADO** una ejecución sin terminal interactiva
- **CUANDO** se registra el proyecto con la opción de modo establecida a `in_place`
- **ENTONCES** el proyecto queda registrado con el modo `in_place`

---

### Requirement: Los Worktrees de los Cambios no Vuelven Irregistrable el Proyecto (REQ-25.31)

Un proyecto cuyo checkout principal no es un worktree enlazado DEBE poder registrarse y seguir registrado con independencia de que existan worktrees de sus cambios en la raíz de worktrees por defecto o en cualquier otra. Un worktree enlazado DEBE identificarse por evidencia de git (su `.git` es un fichero que apunta a un git dir de worktree), y NO por un segmento del nombre de la ruta. Los worktrees que crea `axiom branch ensure` NO DEBEN registrarse como proyectos propios: pertenecen al proyecto del que proceden y no aparecen como entradas separadas en `GET /api/projects` ni en `axiom project list`. Registrar la ruta de un worktree enlazado como proyecto SIGUE rechazándose, con un mensaje que nombre el checkout principal.

#### Scenario: Un checkout principal con un segmento `-wt` en su ruta se registra
- **DADO** un checkout principal (su `.git` es un directorio) ubicado en una ruta que contiene un segmento llamado `axiom-wt`
- **CUANDO** se registra el proyecto
- **ENTONCES** el registro se acepta

#### Scenario: Un worktree enlazado no se registra
- **DADO** un worktree enlazado creado por `ensure`
- **CUANDO** se intenta registrar su ruta como proyecto
- **ENTONCES** se rechaza con un mensaje que nombra el checkout principal

#### Scenario: Crear worktrees no altera el registro del proyecto
- **DADO** un proyecto registrado y activo
- **CUANDO** `ensure` crea worktrees de un cambio en la raíz de worktrees por defecto
- **ENTONCES** `GET /api/projects` sigue listando una única entrada para el proyecto
- **Y** el proyecto sigue siendo activo y registrable

#### Scenario: Los worktrees no aparecen como proyectos
- **DADO** un proyecto con dos worktrees de cambios
- **CUANDO** se ejecuta `axiom project list`
- **ENTONCES** la salida no lista los worktrees como proyectos
