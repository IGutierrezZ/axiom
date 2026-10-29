# Delta para Hub Multi-Proyecto, Selector Dinámico en Dashboard Web y CLI axiom init (INC-25)

> **Incremento:** `inc-25-ramas-y-worktrees-por-topologia` · **Tipo:** delta sobre la capacidad viva `multi-project-hub`
> **Idioma:** Español (castellano peninsular)

Cambios de este delta:

- REQ-25.29 (nuevo): preferencias personales de worktree (modo y raíz) por proyecto, en el hub y nunca en `axiom.yaml`.
- REQ-25.30 (nuevo): las preferencias se piden al registrar un proyecto y se editan después.
- REQ-25.31 (nuevo): los worktrees de los cambios no vuelven no registrable el proyecto ni se registran como proyectos.
- REQ-3.1 y REQ-3.2: los puntos de registro del dashboard (`/api/projects/init`, `/api/projects/add` y el modal de registro) admiten y piden las preferencias.

> REQ-1.1 (esquema `workspaces.json` v`1.0`) no cambia: las preferencias son campos opcionales por proyecto que añade REQ-25.29 sin cambiar la versión del esquema.

## MODIFIED Requirements

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

## ADDED Requirements

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
