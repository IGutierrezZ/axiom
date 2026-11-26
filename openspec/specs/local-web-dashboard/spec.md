# Especificación Viva: Servidor HTTP Local Embebido y Dashboard Web

## Propósito

Definir de forma rigurosa, ejecutable y verificable los requerimientos funcionales y escenarios BDD para el servidor HTTP local en Go, la API REST JSON, los assets frontend embebidos (`//go:embed`) y el subcomando de la CLI `axiom ui` en el runtime de Axiom.

---

## 1. Capacidad: `axiom-local-dashboard-server`

El paquete `internal/dashboard` implementa el servidor HTTP ligero basado en `net/http` que agrega y expone el estado del espacio de trabajo y del ciclo SDD mediante una API REST en formato JSON.

### Requirement: Endpoint REST de Información del Workspace (RF-WEB-01)

El servidor DEBE responder a la ruta `GET /api/workspace` retornando una estructura JSON con los metadatos del espacio de trabajo declarados en `axiom.yaml`: nombre, topología (`monorepo-embedded`, `multirepo`), repositorio canónico de especificaciones, lista de roles con sus rutas y tecnologías, la política de ramas efectiva (`integration_branch` y `branch_prefix`, con los valores por defecto aplicados cuando `axiom.yaml` no declara `git:`), las entradas de `repositories:` y el estado general de validación de la topología. `/api/workspace` es de solo lectura: ninguna ruta del servidor DEBE modificar `axiom.yaml`, salvo `PUT /api/settings` (RF-WEB-19), que es la única vía de escritura de ese fichero (RF-WEB-10).
(Previously: la respuesta era de solo lectura y ninguna ruta del servidor DEBÍA modificar `axiom.yaml`, sin la excepción de `PUT /api/settings`)

#### Scenario: Retorno exitoso de topología y roles
- **DADO** un espacio de trabajo con un archivo `axiom.yaml` válido
- **CUANDO** un cliente realiza una petición HTTP `GET /api/workspace`
- **ENTONCES** la respuesta tiene código de estado `200 OK`
- **Y** el cuerpo contiene un JSON con `name`, `topology`, `specs_repository` y la lista de roles declarados

#### Scenario: Reporte de error cuando el archivo de configuración es inválido
- **DADO** un espacio de trabajo donde `axiom.yaml` no existe o contiene sintaxis corrupta
- **CUANDO** se invoca `GET /api/workspace`
- **ENTONCES** la respuesta tiene código `500 Internal Server Error`
- **Y** el cuerpo JSON detalla el motivo del fallo de lectura o validación

#### Scenario: La respuesta incluye la política de ramas efectiva
- **DADO** un `axiom.yaml` con `git.integration_branch: develop` y sin `git.branch_prefix`
- **CUANDO** se invoca `GET /api/workspace`
- **ENTONCES** el JSON informa de `integration_branch` igual a `develop` y `branch_prefix` igual a `feat`

#### Scenario: La respuesta incluye los ajustes por repositorio
- **DADO** un `axiom.yaml` con `repositories: [{path: repo-backend, integration_branch: release}]`
- **CUANDO** se invoca `GET /api/workspace`
- **ENTONCES** el JSON incluye la entrada de `repo-backend` con su rama de integración `release`

#### Scenario: `/api/workspace` sigue siendo de solo lectura
- **DADO** el servidor en ejecución
- **CUANDO** un cliente envía `POST`, `PUT`, `PATCH` o `DELETE` a `/api/workspace`
- **ENTONCES** el servidor responde `405 Method Not Allowed`
- **Y** las escrituras de `axiom.yaml` solo existen mediante `PUT /api/settings` (RF-WEB-19)

---

### Requirement: Endpoints REST de Consulta de Incrementos (RF-WEB-02)

El servidor DEBE responder a `GET /api/increments` listando todos los incrementos encontrados en el sistema de ficheros tanto en `openspec/changes/` (activos) como en `openspec/changes/archive/` (archivados), indicando su identificador, estado (`active` o `archived`), fase SDD deducida, y conteo de tareas completadas/totales.
Asimismo, DEBE responder a `GET /api/increments/{name}` proveyendo el detalle de los artefactos del incremento especificado (`proposal`, `spec`, `design`, `tasks`, `verifyReport`, `archiveReport`).

#### Scenario: Listado consolidado de incrementos activos y archivados
- **DADO** un repositorio con cambios archivados (ej. `inc-01`, `inc-02`, `inc-03`) y un cambio en curso
- **CUANDO** se realiza una petición HTTP `GET /api/increments`
- **ENTONCES** la respuesta tiene código `200 OK`
- **Y** el JSON contiene una lista con todos los incrementos categorizados por su estado y fase

#### Scenario: Consulta de detalle de un incremento existente
- **DADO** un incremento archivado o activo existente en disco
- **CUANDO** se solicita `GET /api/increments/{name}`
- **ENTONCES** se devuelve código `200 OK` con el resumen de sus artefactos y contenido estructurado

#### Scenario: Petición de incremento no existente
- **DADO** un identificador de cambio que no existe en `openspec/changes/` ni en su histórico
- **CUANDO** se solicita `GET /api/increments/cambio-inexistente`
- **ENTONCES** el servidor responde con código `404 Not Found`

---

### Requirement: Endpoint REST de Estado Multi-Rol y Barrera (RF-WEB-03)

El servidor DEBE responder a `GET /api/roles?change={name}` analizando el desglose de roles del cambio solicitado mediante `internal/multirole`, retornando la lista de roles, sus políticas de compuerta (`blocking`, `deferred`, `optional`), el avance de sus tareas (`tasks.<rol>.md`), el estado de verificación (`verify-report.<rol>.md`) y el dictamen global de la barrera de sincronización (*Satisfied* o *Blocked*).

#### Scenario: Consulta de roles y diagnóstico de barrera para un cambio
- **DADO** un cambio con roles concurrentes declarados en `design.md`
- **CUANDO** se realiza una petición `GET /api/roles?change={name}`
- **ENTONCES** el servidor retorna código `200 OK`
- **Y** el JSON incluye la evaluación de cada rol y el estado de la barrera de sincronización

---

### Requirement: Endpoint REST de Consulta de Handoffs (RF-WEB-04)

El servidor DEBE responder a `GET /api/handoffs?change={name}` deserializando el archivo `handoff.md` del cambio solicitado mediante `internal/handoff`, proveyendo los metadatos de la transición (fases origen y destino, roles involucrados, timestamp, status) y el contenido de las cinco secciones canónicas.

#### Scenario: Consulta de handoff existente
- **DADO** un cambio con un documento `handoff.md` generado
- **CUANDO** se ejecuta `GET /api/handoffs?change={name}`
- **ENTONCES** el servidor responde `200 OK` con los metadatos y las cinco secciones parseadas

#### Scenario: Consulta de handoff en cambio sin relevo activo
- **DADO** un cambio donde no existe el archivo `handoff.md`
- **CUANDO** se ejecuta `GET /api/handoffs?change={name}`
- **ENTONCES** el servidor responde con código `404 Not Found` y un mensaje JSON informativo

---

### Requirement: Endpoint REST de Catálogo de Skills (RF-WEB-05)

El servidor DEBE responder a `GET /api/skills` escaneando los directorios locales de skills (`skills/` e `internal/assets/skills/`) y retornando la lista de skills disponibles, sus nombres, rutas relativas y una descripción resumida extraída de sus respectivos encabezados `SKILL.md`.

#### Scenario: Listado exitoso de skills del proyecto
- **DADO** un proyecto con skills definidas en `skills/` o `internal/assets/skills/`
- **CUANDO** se solicita `GET /api/skills`
- **ENTONCES** el servidor devuelve `200 OK` con el array de skills detectadas y sus metadatos

---

## 2. Capacidad: `axiom-web-ui-assets`

El paquete `internal/dashboard` incorpora los ficheros estáticos de la interfaz web embebidos directamente en el binario compilado mediante `//go:embed assets/*`, garantizando portabilidad absoluta y cero dependencias de NodeJS o NPM.

### Requirement: Servido de Assets Estáticos Embebidos (RF-WEB-06)

El servidor HTTP DEBE servir los archivos estáticos HTML, CSS y JavaScript desde el sistema de archivos embebido de Go (`embed.FS`). La ruta raíz `/` DEBE entregar el archivo `index.html` con tipo de contenido `text/html; charset=utf-8`, y las rutas asociadas (`/style.css`, `/app.js`) DEBEN servirse con sus respectivos MIME types correctos.

#### Scenario: Carga de la página principal del dashboard
- **DADO** el servidor HTTP de Axiom en ejecución
- **CUANDO** se realiza una petición HTTP `GET /`
- **ENTONCES** el servidor responde con código `200 OK`
- **Y** entrega el documento HTML de la aplicación

#### Scenario: Carga de recursos estáticos CSS y JS
- **DADO** el servidor HTTP en ejecución
- **CUANDO** el cliente solicita `/style.css` o `/app.js`
- **ENTONCES** el servidor responde `200 OK` con las cabeceras `Content-Type: text/css` y `Content-Type: application/javascript` respectivamente

---

### Requirement: Estructura de Navegación y Vistas en la Web UI (RF-WEB-07)

La interfaz web en `index.html` y `app.js` DEBE estructurarse en una experiencia de usuario fluida y reactiva en español que incluya:
1. Encabezado institucional con identidad de Axiom, nombre del workspace y badge de topología.
2. Navegación por pestañas (*tabs*):
   - **Tablero de Incrementos:** vista Kanban o lista con tarjetas interactivas agrupadas por fase SDD y porcentaje de tareas.
   - **Monitor Multi-Rol:** vista de detalle del cambio seleccionado con el estado de cada rol (`blocking` vs `deferred`), tareas y estado de la barrera.
   - **Visor de Handoffs:** renderizado claro de relevos con sus metadatos y acordeón de secciones.
   - **Topología y Salud:** inspección de repositorios asociados y repositorio de especificaciones, sin mostrar la política de ramas (que se consulta y edita en «Ajustes del proyecto»); la tarjeta de Topología de Ecosistema y el selector de proyecto enlazan a esa pantalla.
   - **Ajustes del proyecto:** pantalla de consulta y edición de las claves de `axiom.yaml` (RF-WEB-18), con entrada propia en la navegación.
   - **Catálogo de Skills:** visor de skills disponibles en el proyecto.

(Previously: la navegación no incluía «Ajustes del proyecto» y la vista de topología mostraba la política de ramas.)

#### Scenario: Navegación interactiva entre vistas del dashboard
- **DADO** la interfaz web cargada en el navegador
- **CUANDO** el usuario selecciona una pestaña en la barra de navegación
- **ENTONCES** la vista correspondiente se activa dinámicamente sin recargar la página completa
- **Y** los datos se obtienen de los endpoints REST correspondientes

#### Scenario: «Ajustes del proyecto» tiene entrada de navegación y ruta propia
- **DADO** la interfaz cargada
- **CUANDO** el usuario selecciona «Ajustes del proyecto» o abre la ruta por *hash* de esa pantalla
- **ENTONCES** se muestra la pantalla de ajustes sin recargar la página
- **Y** la entrada de navegación aparece marcada como activa

#### Scenario: Topología no muestra la política de ramas
- **DADO** un proyecto que declara `git.integration_branch: develop`
- **CUANDO** el usuario abre Ecosistema (tarjeta de Topología) o el selector de proyecto
- **ENTONCES** ninguna de esas vistas pinta las filas de política de ramas
- **Y** cada una ofrece un enlace a «Ajustes del proyecto»

---

## 3. Capacidad: `axiom-cli-ui`

El comando `axiom ui` en la CLI de Axiom permite a los desarrolladores iniciar el servidor local y abrir el panel de control de forma desatendida o personalizada.

### Requirement: Subcomando axiom ui y Control del Servidor (RF-WEB-08)

El comando `axiom ui` DEBE iniciar el servidor HTTP en el puerto indicado por `--port` (por defecto `8080`). Si el puerto especificado o por defecto ya está ocupado, el comando DEBE buscar de forma incremental el siguiente puerto libre disponible e informar en consola. Salvo que se indique `--no-browser`, el comando DEBE lanzar automáticamente el navegador predeterminado del sistema operativo apuntando a la URL local. Además, DEBE escuchar señales de interrupción (`SIGINT` / `SIGTERM`) para realizar un cierre ordenado (*graceful shutdown*).

#### Scenario: Inicio del servidor en puerto por defecto con apertura de navegador
- **DADO** el comando `axiom ui` ejecutado sin parámetros
- **CUANDO** el puerto 8080 está libre
- **ENTONCES** el servidor se inicia en `http://127.0.0.1:8080`
- **Y** se envía la orden de abrir el navegador en esa URL
- **Y** se muestra en consola el mensaje de inicio y la instrucción para detener con `Ctrl+C`

#### Scenario: Detección y fallback ante puerto ocupado
- **DADO** que el puerto 8080 está en uso por otro proceso
- **CUANDO** el usuario ejecuta `axiom ui`
- **ENTONCES** el sistema detecta la ocupación y se vincula al siguiente puerto libre (ej. `8081`)
- **Y** notifica en consola el puerto final utilizado

#### Scenario: Ejecución en modo headless con --no-browser
- **DADO** el comando ejecutado con la bandera `axiom ui --no-browser`
- **CUANDO** el servidor arranca
- **ENTONCES** el servidor HTTP queda a la escucha normalmente pero NO se invoca el navegador del sistema
### Requirement: Crear un Cambio desde el Dashboard Invoca `ensure` Antes de Escribir (RF-WEB-09)

La creación de un incremento desde el dashboard y con `axiom change create` DEBE ejecutar `axiom branch ensure` para el cambio, con el mismo motor que la CLI, y después escribir con el escritor guardado (RF-BRA-20 de `change-branch-lifecycle`), antes de crear la carpeta del cambio o escribir `proposal.md`, y DEBE escribir los ficheros en la ruta del planning home que devuelve `ensure` (en modo `worktree`, la del worktree del cambio y no la del checkout principal). El escritor guardado se niega fuera de rama aunque se invoque sin `ensure`. La TUI solo lista cambios (RF-WEB-11): no ejecuta `ensure` ni crea incrementos. Si `ensure` devuelve `decision_required` (modo `ask`), el servidor DEBE devolver ese resultado tipado sin escribir nada; la interfaz DEBE presentar la elección (`worktree` o `in_place`) y repetir la creación con la respuesta, que se registra por cambio (RF-BRA-08). Si `ensure` se rechaza por cualquier causa (`dirty_worktree`, `path_too_long`, `invalid_name`, `specs_repo_not_on_integration`, etc.), el servidor DEBE devolver el código tipado y la continuación, sin crear la carpeta del cambio.

#### Scenario: La creación prepara la rama y escribe en el worktree
- **DADO** un proyecto en modo `worktree` y el checkout principal en `main`
- **CUANDO** el cliente solicita crear el incremento `x`
- **ENTONCES** `ensure` crea la rama `feat/x` y su worktree
- **Y** `proposal.md` se escribe en el `openspec/changes/x/` del worktree
- **Y** el checkout principal no contiene `openspec/changes/x/`

#### Scenario: Modo `ask` devuelve la decisión sin escribir
- **DADO** un proyecto en modo `ask` sin respuesta registrada para `x`
- **CUANDO** el cliente solicita crear el incremento `x`
- **ENTONCES** la respuesta contiene `decision_required`
- **Y** no se crea ninguna rama, worktree ni carpeta

#### Scenario: Repetir con la respuesta completa la creación
- **DADO** la respuesta `decision_required` anterior
- **CUANDO** el cliente repite la creación con la elección `in_place`
- **ENTONCES** `ensure` hace checkout de `feat/x` en el repositorio y `proposal.md` se escribe allí
- **Y** una nueva creación de otro cambio `y` en modo `ask` vuelve a devolver `decision_required`

#### Scenario: Un rechazo de `ensure` no deja carpeta
- **DADO** un proyecto en modo `in_place` con el árbol sucio
- **CUANDO** el cliente solicita crear el incremento `x`
- **ENTONCES** la respuesta contiene el código `dirty_worktree` y su continuación
- **Y** no se crea `openspec/changes/x/`

#### Scenario: Multirepo escribe in situ en el repositorio de specs
- **DADO** un multirepo con el repositorio de specs en su rama de integración
- **CUANDO** el cliente solicita crear el incremento `x`
- **ENTONCES** `proposal.md` se escribe en el planning home del repositorio de specs, sin worktree

---

### Requirement: Las Claves de `axiom.yaml` se Editan Solo por `/api/settings`; `/api/workspace` Sigue de Solo Lectura y Toda Escritura Exige Petición Local (RF-WEB-10)

El dashboard NO DEBE editar `axiom.yaml` por ninguna ruta que no sea `PUT /api/settings` (RF-WEB-19), que aplica el catálogo y el escritor de `workspace-topology` (RF-TOP-21 a RF-TOP-23) con validación del documento completo y revisión esperada. `GET /api/workspace` DEBE seguir siendo de solo lectura: `POST`, `PUT`, `PATCH` y `DELETE` sobre `/api/workspace` DEBEN responder `405`. Las claves de solo lectura del catálogo (incluidas `workspace.topology` y `workspace.specs_repository`) NO DEBEN poder editarse desde el dashboard.

Las preferencias personales editables desde el dashboard son las del hub: modo de trabajo y raíz de worktrees (RF-HUB-07), mediante `worktree-prefs`, y el autofetch del proyecto (RF-HUB-13), mediante `/api/settings/personal` (RF-WEB-20), siempre sobre el proyecto activo y con la validación de RF-HUB-07. Editar las preferencias personales NO DEBE modificar ningún fichero del proyecto. Toda ruta del servidor que escribe, sea `PUT /api/settings`, `PUT /api/settings/personal` o `PUT /api/projects/active/worktree-prefs`, DEBE exigir una petición local (`isLocalRequest`) y rechazar cualquier otra con `403` antes de leer el cuerpo o escribir.

(Previously: el dashboard no ofrecía ningún medio de editar `axiom.yaml`, `git:` y `repositories:` eran de solo lectura, ninguna ruta escribía `axiom.yaml`, el autofetch no era editable y el `PUT` de `worktree-prefs` no exigía petición local.)

#### Scenario: `git:` se edita desde Ajustes, no desde Topología
- **DADO** la SPA con un proyecto que declara `git.integration_branch: develop`
- **CUANDO** el usuario abre «Ajustes del proyecto»
- **ENTONCES** `develop` se muestra en un control editable de `git.integration_branch`
- **Y** la vista de topología no ofrece ningún control de edición de `git:` ni `repositories:`

#### Scenario: `/api/workspace` no escribe `axiom.yaml`
- **DADO** el servidor en ejecución
- **CUANDO** un cliente envía `POST`, `PUT`, `PATCH` o `DELETE` a `/api/workspace`
- **ENTONCES** el servidor responde `405 Method Not Allowed`
- **Y** `axiom.yaml` no se modifica

#### Scenario: Edición de las preferencias personales
- **DADO** un proyecto registrado y activo
- **CUANDO** el usuario cambia el modo a `ask` desde el dashboard
- **ENTONCES** la preferencia se persiste en `~/.axiom/workspaces.json`
- **Y** `axiom.yaml` y el resto de ficheros del proyecto no se modifican

#### Scenario: Una preferencia inválida se rechaza
- **DADO** un proyecto registrado
- **CUANDO** el cliente intenta guardar el modo `siempre`
- **ENTONCES** el servidor responde con un error `400` en español que lista los modos válidos
- **Y** el hub no se modifica

#### Scenario: El `PUT` de `worktree-prefs` exige petición local
- **DADO** un cliente cuya petición no es local
- **CUANDO** envía `PUT /api/projects/active/worktree-prefs` con un cuerpo válido
- **ENTONCES** el servidor responde `403`
- **Y** el hub no se modifica
- **Y** el `GET` del mismo endpoint sigue respondiendo con normalidad

#### Scenario: Toda escritura nueva exige petición local
- **DADO** un cliente cuya petición no es local
- **CUANDO** envía `PUT /api/settings` o `PUT /api/settings/personal`
- **ENTONCES** el servidor responde `403` sin escribir ni `axiom.yaml` ni el hub

### Requirement: Los Listados Agregan los Cambios de los Worktrees Registrados (RF-WEB-11)

Los cambios en curso viven en worktrees. El servidor DEBE agregar, a los cambios activos del checkout principal, los cambios activos de los worktrees registrados en el registro local de `axiom branch` (RF-BRA-15) para el proyecto activo en `GET /api/increments`, y DEBE resolver también esos cambios en `GET /api/increments/{name}`, `GET /api/roles?change={name}` y `GET /api/handoffs?change={name}`. Cada incremento listado desde un worktree DEBE indicar la ruta absoluta de su worktree y su rama. Si un mismo cambio activo figura en el checkout principal (por ejemplo, un sumidero acumulativo) y en un worktree, prevalece el del worktree; si figura en el worktree de la rama del cambio y en el de la rama de archivado, prevalece el de la rama de archivado. El dashboard DEBE mostrar los cambios pendientes de archivar (RF-BRA-21 de `change-branch-lifecycle`), es decir, los cambios activos presentes en la rama de integración que aún no se han archivado, con la continuación para archivarlos. El aviso de cambios pendientes de archivar DEBE viajar en la cabecera `X-Axiom-Notes` de la respuesta de `GET /api/increments`; los avisos de este requisito, incluido el de una entrada del registro cuyo worktree ya no existe en disco (que DEBE omitirse sin error), viajan en esa misma cabecera. Los cambios pendientes NO producen ningún error ni bloqueo. Las escrituras del dashboard sobre un cambio que vive en un worktree DEBEN dirigirse a ese worktree.
(Previously: el requisito se apoyaba en un invariante de la rama de integración, eliminado por D17, y los avisos viajaban en el cuerpo de la respuesta)
(Previously: añadía que la pantalla de ciclo de vida de incrementos de la TUI listaba el mismo conjunto de cambios y solo listaba, con un escenario propio.)

#### Scenario: Un cambio de un worktree figura en el listado
- **DADO** un proyecto cuyo checkout principal tiene solo `archive/` bajo `openspec/changes/` y un cambio activo `x` en un worktree registrado
- **CUANDO** se realiza `GET /api/increments`
- **ENTONCES** el listado incluye `x` como activo con la ruta absoluta de su worktree y su rama

#### Scenario: El detalle de un cambio en worktree se resuelve
- **DADO** el mismo cambio `x` en un worktree registrado
- **CUANDO** se solicita `GET /api/increments/x`
- **ENTONCES** se devuelve `200 OK` con los artefactos del cambio leídos del worktree

#### Scenario: El worktree prevalece sobre el checkout principal
- **DADO** un cambio activo `e2e-cumulative` presente en el checkout principal y en un worktree registrado
- **CUANDO** se realiza `GET /api/increments`
- **ENTONCES** el cambio aparece una sola vez, con los datos del worktree

#### Scenario: El worktree de la rama de archivado prevalece sobre el de la rama del cambio
- **DADO** el cambio `x` presente en los worktrees de `feat/x` y de `feat/x-archive`, ambos registrados
- **CUANDO** se realiza `GET /api/increments`
- **ENTONCES** `x` aparece una sola vez, con los datos y la rama del worktree de `feat/x-archive`

#### Scenario: Un worktree desaparecido se omite con aviso
- **DADO** una entrada del registro local cuyo worktree fue borrado del disco
- **CUANDO** se realiza `GET /api/increments`
- **ENTONCES** la respuesta es `200 OK` sin ese cambio
- **Y** la cabecera `X-Axiom-Notes` incluye un aviso que nombra la ruta ausente

#### Scenario: Un cambio fusionado y sin archivar se muestra como pendiente
- **DADO** un cambio `y` fusionado y presente en la rama de integración, sin archivar
- **CUANDO** se realiza `GET /api/increments`
- **ENTONCES** el listado marca `y` como pendiente de archivar con la continuación `axiom branch ensure --change y --archive`
- **Y** la cabecera `X-Axiom-Notes` incluye el aviso de cambios pendientes de archivar con el nombre `y`
- **Y** la respuesta es `200 OK`

#### Scenario: Sin cambios pendientes no hay aviso de pendientes
- **DADO** una rama de integración cuyo `changes/` solo contiene `archive/` y los sumideros exentos
- **CUANDO** se realiza `GET /api/increments`
- **ENTONCES** la cabecera `X-Axiom-Notes` no incluye ningún aviso de cambios pendientes de archivar

#### Scenario: Una escritura sobre un cambio en worktree se dirige al worktree
- **DADO** un cambio `x` en un worktree registrado
- **CUANDO** el cliente crea un handoff para `x` desde el dashboard
- **ENTONCES** `handoff.md` se escribe en la carpeta del cambio del worktree

### Requirement: El Dashboard No Expone el Modo de Revisión de RDD (RF-WEB-12)

El servidor del Dashboard NO DEBE exponer el endpoint `/api/project/review-mode` ni ningún otro de modo de revisión, y la interfaz web NO DEBE incluir el selector `reviewmode.js`. Una petición a esa ruta DEBE responder como ruta inexistente (404). El resto de endpoints del Dashboard NO cambia.

#### Scenario: El endpoint de modo de revisión no existe

- **DADO** el servidor del Dashboard en marcha
- **CUANDO** se solicita `GET /api/project/review-mode`
- **ENTONCES** la respuesta es 404

#### Scenario: La interfaz no carga el selector

- **DADO** los recursos estáticos del Dashboard
- **CUANDO** se listan
- **ENTONCES** no existe `reviewmode.js` ni ninguna referencia a él en las páginas
### Requirement: Endpoint de Agentes del Proyecto Activo (RF-WEB-13)

El servidor DEBE responder a `GET /api/agents` con la lista de agentes del catálogo de Axiom, en el orden del catálogo, y para cada uno: su identificador y nombre, si Axiom lo admite (`supported`, agente seleccionable), si se detecta en el equipo (`detected`, su carpeta de configuración global existe), si está instalado en el proyecto activo (`installed`), si admite el ámbito workspace (`workspace`) y, cuando no se puede instalar en el proyecto, el motivo en castellano (`reason`). La respuesta DEBE indicar además la raíz del proyecto activo, si esa raíz es un proyecto de Axiom (`project`, tiene en sí misma un marcador de workspace) y si el estado del proyecto registra su lista de agentes (`recorded`). Un agente solo figura como instalado cuando el proyecto ha registrado su lista (`installed_agents` no nulo en `.axiom/state.json` del proyecto) y lo incluye: la lista del estado global no cuenta. La consulta es de solo lectura y NO DEBE ejecutar subprocesos ni escribir ficheros. Si el estado del proyecto no se puede leer, la respuesta DEBE ser `200` con el motivo en `state_error` y ningún agente marcado como instalado.

#### Scenario: Las tres marcas de cada agente

- **DADO** un proyecto activo cuyo estado registra `installed_agents: ["claude-code"]` y un equipo donde existe la configuración global de Claude Code y de Cursor
- **CUANDO** se solicita `GET /api/agents`
- **ENTONCES** la respuesta es `200` y `recorded` es verdadero
- **Y** `claude-code` figura como admitido, detectado e instalado
- **Y** `cursor` figura como admitido y detectado, pero no instalado

#### Scenario: Un proyecto sin lista registrada no hereda la global

- **DADO** un proyecto activo sin `.axiom/state.json` propio y un estado global con `installed_agents: ["claude-code"]`
- **CUANDO** se solicita `GET /api/agents`
- **ENTONCES** `recorded` es falso y ningún agente figura como instalado

#### Scenario: Un agente sin modo workspace lleva el motivo

- **DADO** el catálogo de agentes
- **CUANDO** se solicita `GET /api/agents`
- **ENTONCES** `github-copilot-cli` figura con `workspace` falso y un `reason` no vacío

### Requirement: Instalar un Agente en el Proyecto Activo desde el Dashboard (RF-WEB-14)

El servidor DEBE exponer `POST /api/agents/install` con un cuerpo `{"agent": "<id>"}` que instala ese único agente en el proyecto activo con el mismo núcleo que `axiom install --scope workspace --agent <id>`, sin depender del directorio actual del proceso. La instalación DEBE añadir el agente a `installed_agents` del estado del proyecto sin quitar los ya registrados. Antes de escribir nada, el servidor DEBE rechazar con `400` y el motivo un agente vacío, desconocido, no seleccionable o sin modo workspace, y con `409` una petición cuya raíz activa no es un proyecto de Axiom (sin marcador de workspace en ella). Un fallo de la instalación DEBE responder `500` con `success: false` y el error. Las instalaciones concurrentes DEBEN ejecutarse de una en una.

La pantalla Ecosistema DEBE mostrar un bloque «Agentes» con las tres marcas de cada agente y un botón «Instalar» solo en los agentes admitidos, con modo workspace y no instalados; los que no admiten el ámbito workspace DEBEN mostrar el motivo en lugar del botón. Antes de llamar a la API, la interfaz DEBE pedir confirmación explicando que la instalación escribe en el repositorio y, según el agente, en la carpeta de usuario. Tras instalar, la interfaz DEBE refrescar la lista y el diagnóstico del Doctor.

#### Scenario: Instalar un agente lo registra sin pisar los demás

- **DADO** un proyecto activo cuyo estado registra `installed_agents: ["claude-code"]`
- **CUANDO** se envía `POST /api/agents/install` con `{"agent": "cursor"}` y la instalación termina bien
- **ENTONCES** la respuesta es `200` con `success: true`
- **Y** `installed_agents` del proyecto contiene `claude-code` y `cursor`

#### Scenario: Un agente sin modo workspace se rechaza antes de escribir

- **DADO** un proyecto activo
- **CUANDO** se envía `POST /api/agents/install` con `{"agent": "github-copilot-cli"}`
- **ENTONCES** la respuesta es `400` con el motivo
- **Y** no se ejecuta ninguna instalación

#### Scenario: Sin proyecto activo no se instala

- **DADO** el dashboard con una raíz activa sin marcador de workspace
- **CUANDO** se envía `POST /api/agents/install` con un agente válido
- **ENTONCES** la respuesta es `409` y no se ejecuta ninguna instalación

#### Scenario: La interfaz no ofrece el botón a un agente sin modo workspace

- **DADO** la pantalla Ecosistema con la lista de agentes cargada
- **CUANDO** se pinta la fila de un agente con `workspace` falso
- **ENTONCES** la fila muestra el motivo y no tiene botón «Instalar»

### Requirement: Endpoint de Solo Lectura que Sirve Únicamente `resumen-spec.html` y `resumen-design.html` (RF-WEB-15)

El dashboard DEBE exponer un endpoint de solo lectura (método GET) que sirva el contenido de `resumen-spec.html` o de `resumen-design.html` de un cambio, y ningún otro fichero. El nombre del fichero DEBE validarse contra una lista blanca de exactamente esos dos nombres. El nombre del cambio DEBE rechazarse (error de cliente) cuando contenga `..`, un separador de ruta (`/` o `\`), su forma codificada (por ejemplo `%2e%2e`, `%2f`, `%5c`) o un byte nulo, sin llegar a leer del disco. Un fichero fuera de la lista blanca DEBE rechazarse sin leer del disco. Si el cambio no existe o el fichero no existe, la respuesta DEBE ser 404. Un método distinto de GET DEBE rechazarse.

La respuesta correcta DEBE llevar, fijadas por el propio endpoint:

- `Content-Type: text/html; charset=utf-8`;
- `Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; img-src data:`, con ese valor exacto;
- `X-Content-Type-Options: nosniff`.

El endpoint NO DEBE modificar ningún fichero ni estado, y NO DEBE ampliar la lista de ficheros servibles.

#### Scenario: Sirve `resumen-spec.html` con las cabeceras de seguridad

- **DADO** un cambio con `resumen-spec.html`
- **CUANDO** se pide ese fichero al endpoint
- **ENTONCES** la respuesta es 200 con el contenido del fichero
- **Y** `Content-Security-Policy` vale exactamente `default-src 'none'; style-src 'unsafe-inline'; img-src data:`
- **Y** `X-Content-Type-Options` vale `nosniff`
- **Y** `Content-Type` es `text/html; charset=utf-8`

#### Scenario: Sirve `resumen-design.html`

- **DADO** un cambio con `resumen-design.html`
- **CUANDO** se pide ese fichero al endpoint
- **ENTONCES** la respuesta es 200 con el contenido del fichero y las mismas cabeceras

#### Scenario: Cualquier otro fichero se rechaza

- **DADO** un cambio con `spec.md`, `kickoff.yaml` y `proposal.md`
- **CUANDO** se pide al endpoint `spec.md`, `kickoff.yaml`, `proposal.md` o `resumen-tasks.html`
- **ENTONCES** todas las peticiones se rechazan con un error de cliente
- **Y** ninguna devuelve contenido del fichero

#### Scenario: Nombre de cambio con `..` o separadores

- **DADO** un directorio hermano fuera de `openspec/changes/` con un `resumen-spec.html`
- **CUANDO** se pide con el cambio `..`, `../x`, `x/..`, `%2e%2e`, `a%2fb` o `a\b`
- **ENTONCES** todas se rechazan con un error de cliente
- **Y** no se devuelve el contenido de ningún fichero fuera de la carpeta del cambio

#### Scenario: Fichero ausente

- **DADO** un cambio sin `resumen-design.html`
- **CUANDO** se pide ese fichero al endpoint
- **ENTONCES** la respuesta es 404

#### Scenario: Cambio inexistente

- **DADO** un nombre de cambio válido sin carpeta
- **CUANDO** se pide `resumen-spec.html`
- **ENTONCES** la respuesta es 404

#### Scenario: Método distinto de GET

- **DADO** el endpoint
- **CUANDO** se envía POST, PUT o DELETE
- **ENTONCES** la petición se rechaza
- **Y** no se modifica ningún fichero

---

### Requirement: Pestaña «Explicación» en el Detalle del Incremento con `iframe` en Sandbox y Aviso de Desactualizado (RF-WEB-16)

El detalle de un incremento SDD DEBE incluir una pestaña «Explicación» que muestre `resumen-spec.html` y `resumen-design.html` del cambio, cada uno en un `iframe` cuyo atributo `sandbox` NO DEBE contener nunca `allow-same-origin` ni `allow-scripts` (puede estar vacío). El `iframe` DEBE cargar el contenido desde el endpoint de RF-WEB-15, nunca incrustarlo en el documento. El detalle DEBE proyectar la frescura de cada HTML con el mismo cálculo que `axiom sdd status` (`fresh`, `stale`, `missing`): con `stale` la pestaña DEBE mostrar un aviso visible de que el resumen está desactualizado respecto del md; con `missing` DEBE indicar que no hay explicación generada, sin mostrar un `iframe` roto. La pestaña NO DEBE ofrecer edición del HTML ni ninguna acción de escritura, y NO DEBE mostrar nada de HTML cuando el cambio no lo tiene sellado (`explainer_html` ausente o `no`) más allá de indicar que la explicación no está activada para el cambio. La frescura nunca bloquea ninguna otra acción del detalle.

#### Scenario: HTML al día se muestra en un `iframe` en sandbox

- **DADO** un incremento con `explainer_html: yes` y un `resumen-spec.html` `fresh`
- **CUANDO** se abre la pestaña «Explicación»
- **ENTONCES** se muestra un `iframe` con `sandbox` que no contiene `allow-same-origin` ni `allow-scripts`
- **Y** su origen apunta al endpoint de RF-WEB-15
- **Y** no se muestra aviso de desactualizado

#### Scenario: HTML desactualizado muestra el aviso

- **DADO** un incremento cuyo `resumen-spec.html` está `stale`
- **CUANDO** se abre la pestaña «Explicación»
- **ENTONCES** se muestra un aviso visible de que el resumen está desactualizado respecto del md
- **Y** el `iframe` sigue mostrando el HTML existente

#### Scenario: HTML ausente

- **DADO** un incremento con `explainer_html: yes` sin `resumen-design.html`
- **CUANDO** se abre la pestaña «Explicación»
- **ENTONCES** la sección de design indica que no hay explicación generada
- **Y** no se muestra un `iframe` para ese fichero

#### Scenario: Cambio sin explicación activada

- **DADO** un incremento con `explainer_html: no` o sin el campo
- **CUANDO** se abre la pestaña «Explicación»
- **ENTONCES** indica que la explicación no está activada para el cambio
- **Y** no se carga ningún `iframe`

#### Scenario: La frescura del dashboard coincide con la del status

- **DADO** un incremento con HTML `stale` según `axiom sdd status`
- **CUANDO** se consulta el detalle del incremento por la API del dashboard
- **ENTONCES** la frescura proyectada es la misma
- **Y** la consulta no modifica ningún fichero

#### Scenario: Sin acciones de escritura

- **DADO** la pestaña «Explicación»
- **CUANDO** se inspeccionan sus controles
- **ENTONCES** no ofrece editar, guardar ni regenerar el HTML
### Requirement: Pestaña «Pruebas» de Solo Lectura en el Detalle del Incremento, sin Ruta Nueva (RF-WEB-17)

`GET /api/increments/<n>` DEBE incluir en el detalle de un incremento SDD un campo de la batería de pruebas con el contenido de `pruebas.md` y su cobertura, calculada con el mismo `acceptancetests.Check` que usan `axiom sdd tests check` y `axiom sdd status` (RF-KOF-20). El campo DEBE incluir los casos de la tabla, las cifras de escenarios M/M, requisitos M/M y número de casos, y los hallazgos. Sin `pruebas.md`, el campo DEBE indicar su ausencia y, si la función está activa, el aviso de RF-KOF-22. No DEBE añadirse ninguna ruta, y el endpoint de ficheros de RF-WEB-15 NO DEBE ampliar su lista blanca.

El detalle DEBE incluir una pestaña «Pruebas», de solo lectura: muestra la tabla de casos con sus columnas, la cobertura y los hallazgos; no ofrece edición, ejecución ni marcas por entorno ni ninguna acción de escritura. Todo texto proveniente de `pruebas.md` DEBE insertarse como texto, nunca como HTML. Con la función no activa y sin `pruebas.md`, la pestaña DEBE indicar que las pruebas de aceptación no están activadas para el proyecto. Los hallazgos nunca bloquean ninguna otra acción del detalle. Las pruebas de pines de pestañas y de assets, y el canario de secretos, DEBEN seguir en verde.

#### Scenario: El detalle devuelve la batería y su cobertura
- **DADO** un incremento con `pruebas.md` válido y su cobertura completa
- **CUANDO** se consulta `GET /api/increments/<n>`
- **ENTONCES** el detalle incluye los casos, los escenarios M/M, los requisitos M/M, el número de casos y cero hallazgos

#### Scenario: La cobertura coincide con la del verbo
- **DADO** un incremento con `pruebas.md` con hallazgos
- **CUANDO** se consulta el detalle y se ejecuta `axiom sdd tests check --json`
- **ENTONCES** cifras y hallazgos son iguales

#### Scenario: La pestaña es de solo lectura
- **DADO** un incremento con `pruebas.md`
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** se muestran la tabla, la cobertura y los hallazgos
- **Y** no hay controles de edición ni de ejecución

#### Scenario: Contenido hostil se muestra como texto
- **DADO** un `pruebas.md` con `<script>` en una celda
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** el texto aparece literal y no se interpreta como HTML

#### Scenario: Función no activada
- **DADO** un proyecto sin el bloque `acceptance_tests` y un incremento sin `pruebas.md`
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** indica que las pruebas de aceptación no están activadas

#### Scenario: Activa y sin fichero
- **DADO** la función activa y un incremento tras tasks sin `pruebas.md`
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** indica la ausencia y muestra el aviso de RF-KOF-22
- **Y** no bloquea ninguna otra pestaña

#### Scenario: Sin ruta nueva
- **DADO** el servidor del dashboard
- **CUANDO** se enumeran sus rutas
- **ENTONCES** no existe ninguna ruta añadida por este requisito y la lista blanca de RF-WEB-15 no cambia
### Requirement: Pantalla «Ajustes del Proyecto» Generada desde el Catálogo, con Reglas por Topología e ISO y sin Controles de RDD (RF-WEB-18)

El dashboard DEBE ofrecer una pantalla «Ajustes del proyecto» con ruta por *hash* y entrada en la navegación. La pantalla DEBE generarse desde el catálogo que devuelve `GET /api/settings` (RF-WEB-19) sin conocer las claves: cada entrada se pinta con su etiqueta, descripción, tipo, valores permitidos y efecto descrito, y un control acorde (selector para vocabularios cerrados, casilla para booleanos, número, texto o lista). Todo texto del catálogo y del `axiom.yaml` DEBE insertarse como texto, nunca como HTML.

La pantalla DEBE mostrar de forma visible el aviso «Editar solo afecta a cambios nuevos» y comunicar que los cambios ya sellados conservan su instantánea de `kickoff.yaml`; ninguna acción de la pantalla DEBE modificar un `kickoff.yaml` existente. Al guardar, DEBE enviar la revisión leída, mostrar el error del validador (castellano) sin perder lo tecleado cuando se rechaza, y pedir recargar cuando la revisión está obsoleta. Retirar un valor DEBE enviarse como `null`. La pantalla DEBE enviar todos los cambios pendientes en una sola petición (lote atómico, RF-WEB-19). Junto a la ruta del fichero DEBE indicar que lo guardado queda sin confirmar en git, en el árbol de trabajo del repositorio que contiene `axiom.yaml` (en `multirepo`, el de specs), y que confirmarlo es decisión de la persona. Cuando el servidor informa `writable: false` (documento con anclas, alias, `<<` o varios documentos YAML), DEBE deshabilitar los controles de edición y mostrar el motivo.

Las claves con `editable: false` (`technical_context`, `workspace.root`, `skill_roots`, `governance.*`, `workspace.topology`, `workspace.specs_repository`) DEBEN mostrarse sin control de edición, con su valor y la acción «Abrir en el editor»: un enlace `vscode://file/<ruta absoluta de axiom.yaml>` etiquetado «Abrir en el editor», más un botón «Copiar ruta» que copia esa ruta absoluta. El servidor NO DEBE lanzar ningún proceso para abrir el editor.

La pantalla DEBE incluir un bloque «Preferencias personales» con el autofetch (RF-WEB-20) y `worktree_mode` y `worktree_root` (estos dos sobre `worktree-prefs`, RF-WEB-10), identificado como personal y no versionado. En `monorepo-embedded` la pantalla NO DEBE mostrar `roles`, `gate_policy`, `repositories` ni `acceptance_tests`; en `multirepo` DEBE mostrar `repositories[]`, `acceptance_tests` y `roles[].gate_policy` con `optional` entre sus opciones. En un proyecto ISO, `workflow.lane` NO DEBE ofrecer `odd`. La pantalla NO DEBE incluir ningún control de RDD (global, por proyecto o por clon), ni `integrations.*`, ni `commit_review`, ni `docs_publication`.

#### Scenario: La pantalla se genera desde el catálogo
- **DADO** un catálogo con una entrada de vocabulario cerrado
- **CUANDO** se abre «Ajustes del proyecto»
- **ENTONCES** la entrada se pinta como selector con exactamente los valores que devolvió el servidor
- **Y** se muestran su descripción y su efecto

#### Scenario: Aviso de que solo afecta a cambios nuevos
- **DADO** un proyecto con un cambio ya sellado
- **CUANDO** se abre «Ajustes del proyecto»
- **ENTONCES** se muestra «Editar solo afecta a cambios nuevos»
- **Y** tras guardar `workflow.lane`, el `kickoff.yaml` del cambio sellado no cambia

#### Scenario: Cambiar el carril conserva comentarios y orden
- **DADO** un `axiom.yaml` comentado con `workflow.lane: ask`
- **CUANDO** el usuario elige `sdd` y guarda
- **ENTONCES** el servidor escribe el fichero solo con esa línea cambiada
- **Y** la pantalla muestra la revisión nueva y el valor guardado

#### Scenario: Valor inválido o revisión obsoleta no pierde lo tecleado
- **DADO** un valor rechazado por el validador o una revisión obsoleta
- **CUANDO** el usuario guarda
- **ENTONCES** la pantalla muestra el error en castellano y conserva el valor tecleado
- **Y** en el caso de la revisión obsoleta ofrece recargar

#### Scenario: Claves de solo lectura con «Abrir en el editor» y «Copiar ruta»
- **DADO** la pantalla abierta
- **CUANDO** se inspecciona `workspace.topology`, `technical_context` o `governance.*`
- **ENTONCES** se muestran sin control de edición
- **Y** aparece un enlace «Abrir en el editor» con destino `vscode://file/<ruta absoluta de axiom.yaml>` y un botón «Copiar ruta»
- **Y** el servidor no lanza ningún proceso

#### Scenario: Bloque «Preferencias personales»
- **DADO** un proyecto registrado y activo
- **CUANDO** se abre la pantalla
- **ENTONCES** el bloque «Preferencias personales» muestra el autofetch, `worktree_mode` y `worktree_root`
- **Y** indica que no se guardan en `axiom.yaml`

#### Scenario: Monorepo sin claves de multirepo
- **DADO** un proyecto `monorepo-embedded`
- **CUANDO** se abre la pantalla
- **ENTONCES** no aparecen `roles`, `gate_policy`, `repositories` ni `acceptance_tests`

#### Scenario: Multirepo con `gate_policy` incluido `optional`
- **DADO** un proyecto `multirepo`
- **CUANDO** se abre la pantalla
- **ENTONCES** aparecen `repositories[]`, `acceptance_tests` y `roles[].gate_policy`
- **Y** `gate_policy` ofrece `blocking`, `deferred` y `optional`

#### Scenario: ISO no ofrece `odd`
- **DADO** un proyecto con `sdd.iso: true`
- **CUANDO** se abre la pantalla
- **ENTONCES** `workflow.lane` no ofrece `odd`
- **Y** si un cliente envía `odd`, el servidor lo rechaza (RF-WEB-19)

#### Scenario: Ningún control de RDD ni de integraciones
- **DADO** la pantalla abierta en cualquier topología
- **CUANDO** se recorren todos sus controles
- **ENTONCES** no existe ningún control de RDD, de `integrations.*`, de `commit_review` ni de `docs_publication`

#### Scenario: Lo guardado queda sin confirmar en git
- **DADO** la pantalla abierta con la ruta de `axiom.yaml`
- **CUANDO** se inspecciona junto a esa ruta
- **ENTONCES** se indica que los cambios guardados quedan sin confirmar en git y que confirmarlos es decisión de la persona
- **Y** guardar no ejecuta ninguna operación de git

#### Scenario: Documento no editable deshabilita los controles
- **DADO** un `axiom.yaml` con anclas, alias, `<<` o varios documentos YAML
- **CUANDO** se abre la pantalla
- **ENTONCES** los controles de edición están deshabilitados y se muestra el motivo informado por el servidor

#### Scenario: Texto hostil se muestra como texto
- **DADO** un `workspace.name` con `<script>` en su valor
- **CUANDO** se abre la pantalla
- **ENTONCES** el valor aparece literal y no se interpreta como HTML

---

### Requirement: Endpoint `GET|PUT /api/settings` con Catálogo, Revisión, Escritura Validada y Errores Tipados en Castellano (RF-WEB-19)

`GET /api/settings` DEBE devolver el catálogo (RF-TOP-21) ya filtrado por la topología y el carácter ISO del proyecto activo, con el valor actual de cada clave, la revisión de `axiom.yaml` (RF-TOP-23) y la ruta absoluta del fichero (la de `Locate().ConfigPath`). La respuesta NO DEBE contener `integrations` ni valor alguno de credenciales o secretos.

`GET /api/settings` DEBE informar además si el documento es escribible (`writable`) y, si no lo es, el motivo (`writeBlockedReason`), y los errores que el documento ya tiene en disco (`documentErrors`); los valores se leen del documento tal como está escrito, de modo que un `axiom.yaml` inválido escrito a mano se muestra con su valor real.

`PUT /api/settings` DEBE recibir un cuerpo `{"revision": "<sha256>", "changes": {"<clave>": <valor>|null, ...}}` con la revisión esperada y un parche de claves del catálogo (con `null` para retirar) y aplicarlo con el escritor (RF-TOP-22). Las claves por elemento se direccionan en el parche como `repositories[<path>].<campo>` y `roles[<id>].<campo>`. Todos los cambios del parche se aplican y validan juntos, como un lote atómico de todo o nada: si cualquiera es rechazado, no se escribe ninguno. DEBE exigir petición local y comprobarla antes de leer el cuerpo (RF-WEB-10). Un fichero con varios documentos YAML, anclas, alias o `<<` DEBE rechazarse, porque reescribirlo perdería o alteraría contenido.

Todo error DEBE llevar un código tipado y mensaje en castellano, con el cuerpo `{"code", "error", "violations"?}`, según esta correspondencia:

| Código | HTTP | Cuándo |
|---|---|---|
| `invalid_request` | `400` | cuerpo mal formado, revisión esperada ausente o `changes` vacío |
| `config_missing` | `404` | no hay proyecto activo o `axiom.yaml` no existe |
| `revision_mismatch` | `409` | revisión esperada obsoleta o cambio externo antes del reemplazo |
| `unsupported_document` | `422` | anclas, alias, `<<`, varios documentos YAML o clave repetida |
| `unknown_key` | `422` | clave que no está en el catálogo (incluidas `integrations.*`) |
| `key_not_editable` | `422` | clave de solo lectura |
| `key_not_applicable` | `422` | clave de multirepo en monorepo, o rol o repositorio inexistente |
| `invalid_value` | `422` | tipo incorrecto, valor fuera del vocabulario o `null` en una clave obligatoria |
| `validation_failed` | `422` | el documento resultante no supera la validación del motor, incluida la contradicción ISO; incluye `violations` con el error del validador |
| `write_failed` | `500` | fallo de E/S al escribir el fichero |

Además: `200` con el catálogo y la revisión nuevos, `changed` y `warnings` si se escribe (o si el parche no cambia nada); `403` para petición no local; `405` para métodos distintos de `GET` y `PUT`. Los errores que `axiom.yaml` ya tenía en disco antes del parche NO DEBEN impedir guardar: se devuelven como `warnings` (RF-TOP-22). Un rechazo NO DEBE modificar `axiom.yaml`; el mensaje de `write_failed` DEBE indicar si el reemplazo llegó a producirse. Las rutas nuevas DEBEN registrarse en el canario de secretos (`coveredRoutes` o `excludedRoutes`) y este DEBE seguir en verde.

#### Scenario: GET devuelve catálogo, valores, revisión y ruta
- **DADO** un proyecto activo con `axiom.yaml`
- **CUANDO** el cliente consulta `GET /api/settings`
- **ENTONCES** la respuesta incluye las entradas del catálogo con su valor actual, la revisión y la ruta absoluta del fichero
- **Y** no incluye `integrations`

#### Scenario: GET no filtra secretos
- **DADO** un `axiom.yaml` con `integrations` que contiene una credencial centinela
- **CUANDO** el cliente consulta `GET /api/settings`
- **ENTONCES** la respuesta no contiene la credencial ni el bloque `integrations`
- **Y** el canario de secretos pasa con la ruta registrada

#### Scenario: PUT válido escribe y devuelve la revisión nueva
- **DADO** la revisión R leída y una petición local
- **CUANDO** el cliente envía `PUT /api/settings` con `workflow.lane: sdd` y la revisión R
- **ENTONCES** responde `200` con la revisión nueva
- **Y** `axiom.yaml` conserva sus comentarios y orden

#### Scenario: Revisión obsoleta devuelve 409
- **DADO** una edición externa de `axiom.yaml` tras la lectura
- **CUANDO** el cliente envía el `PUT` con la revisión anterior
- **ENTONCES** responde `409` con código tipado y mensaje en castellano
- **Y** el fichero conserva la edición externa

#### Scenario: Valor inválido devuelve 422 con el error del validador
- **DADO** una petición local con `git.promotion: siempre`
- **CUANDO** el cliente envía el `PUT`
- **ENTONCES** responde `422` con el error del validador en castellano
- **Y** la revisión del fichero no cambia

#### Scenario: `odd` en ISO se rechaza
- **DADO** un proyecto con `sdd.iso: true`
- **CUANDO** el cliente envía `workflow.lane: odd`
- **ENTONCES** responde `422` con la contradicción ISO
- **Y** el fichero no cambia

#### Scenario: Clave no editable devuelve 422
- **DADO** una petición con `workspace.topology` o `integrations.<clave>`
- **CUANDO** el cliente envía el `PUT`
- **ENTONCES** responde `422` con el código de clave no editable
- **Y** el fichero no cambia

#### Scenario: Cuerpo mal formado o sin revisión devuelve 400
- **DADO** un cuerpo que no es JSON válido o que no incluye la revisión esperada
- **CUANDO** el cliente envía el `PUT`
- **ENTONCES** responde `400` con mensaje en castellano
- **Y** el fichero no cambia

#### Scenario: Petición no local devuelve 403
- **DADO** una petición que no es local
- **CUANDO** el cliente envía `PUT /api/settings`
- **ENTONCES** responde `403`
- **Y** el fichero no cambia

#### Scenario: Sin proyecto activo o sin fichero devuelve 404
- **DADO** que no hay proyecto activo o que su `axiom.yaml` no existe
- **CUANDO** el cliente consulta `GET /api/settings` o envía `PUT /api/settings`
- **ENTONCES** responde `404` con código tipado

#### Scenario: Métodos no admitidos devuelven 405
- **DADO** el servidor en ejecución
- **CUANDO** el cliente envía `POST`, `PATCH` o `DELETE` a `/api/settings`
- **ENTONCES** responde `405`

#### Scenario: Multirepo escribe el fichero del repositorio de specs
- **DADO** un `multirepo` activo
- **CUANDO** el cliente envía un `PUT` válido
- **ENTONCES** se escribe el `axiom.yaml` del repositorio de specs y `GET` informa esa ruta

#### Scenario: El lote es atómico, todo o nada
- **DADO** un parche con un cambio válido y otro inválido
- **CUANDO** el cliente envía el `PUT`
- **ENTONCES** responde `422` y `axiom.yaml` no cambia, tampoco por el cambio válido

#### Scenario: Un lote puede pasar de `odd` a ISO de una vez
- **DADO** un proyecto con `workflow.lane: odd` y sin ISO
- **CUANDO** el cliente envía en un mismo lote `sdd.iso: true` y `workflow.lane: sdd`
- **ENTONCES** responde `200` y el documento resultante es válido

#### Scenario: Las claves por elemento se direccionan por ruta o por identificador
- **DADO** un `multirepo` con el repositorio `repo-a` y el rol `web`
- **CUANDO** el cliente envía `repositories[repo-a].promotion` y `roles[web].gate_policy`
- **ENTONCES** se escriben en la entrada de `repo-a` y en el rol `web`

#### Scenario: Varios documentos YAML se rechazan
- **DADO** un `axiom.yaml` con dos documentos YAML
- **CUANDO** el cliente envía un `PUT` válido
- **ENTONCES** responde `422` con código `unsupported_document`
- **Y** el fichero conserva ambos documentos

#### Scenario: La petición no local se rechaza antes de leer el cuerpo
- **DADO** una petición no local con un cuerpo mal formado
- **CUANDO** el cliente envía el `PUT`
- **ENTONCES** responde `403` y no `400`

#### Scenario: Un error previo del documento no impide guardar
- **DADO** un `axiom.yaml` que ya tiene un error del motor que el parche no toca
- **CUANDO** el cliente envía un `PUT` válido
- **ENTONCES** responde `200` con ese error en `warnings`
- **Y** el parche se escribe

#### Scenario: Fallo de escritura devuelve 500
- **DADO** una petición válida cuyo reemplazo del fichero falla
- **CUANDO** el cliente envía el `PUT`
- **ENTONCES** responde `500` con código `write_failed` y mensaje en castellano que dice si el reemplazo llegó a producirse
- **Y** `axiom.yaml` conserva su contenido anterior y no queda ningún temporal

---

### Requirement: Endpoint `GET|PUT /api/settings/personal` para el Autofetch del Proyecto Activo (RF-WEB-20)

`GET /api/settings/personal` DEBE devolver el autofetch personal del proyecto activo mediante `hub.LookupAutofetch`, y `PUT /api/settings/personal` DEBE guardarlo mediante `hub.SetAutofetch`, con un valor booleano. El PUT DEBE exigir petición local (RF-WEB-10). El valor DEBE persistirse en `~/.axiom/workspaces.json` y NO DEBE modificar ningún fichero del proyecto. Códigos: `200` en lectura y escritura correctas; `400` para cuerpo mal formado o valor no booleano; `403` para petición no local; `404` si el proyecto activo no está registrado en el hub (`hub.ErrProjectNotRegistered`) o no hay proyecto activo; `405` para otros métodos. Los errores DEBEN llevar mensaje en castellano. Las rutas DEBEN estar registradas en el canario de secretos.

Este requisito (P16) sustituye la línea del plan de la propuesta anterior según la cual «el autofetch… no entra en esta pantalla»: el autofetch del proyecto SÍ se edita desde «Ajustes del proyecto», en el bloque «Preferencias personales» (RF-WEB-18). `multi-project-hub` no cambia: `LookupAutofetch` y `SetAutofetch` (RF-HUB-13) se reutilizan tal cual.

#### Scenario: Lectura del autofetch
- **DADO** un proyecto registrado con el autofetch activado
- **CUANDO** el cliente consulta `GET /api/settings/personal`
- **ENTONCES** responde `200` con el autofetch en `true`

#### Scenario: Escritura del autofetch persiste en el hub
- **DADO** un proyecto registrado y activo y una petición local
- **CUANDO** el cliente envía `PUT /api/settings/personal` con el autofetch en `true`
- **ENTONCES** responde `200`
- **Y** `~/.axiom/workspaces.json` refleja el valor
- **Y** `axiom.yaml` y el resto de ficheros del proyecto no se modifican

#### Scenario: Proyecto no registrado devuelve 404
- **DADO** un proyecto activo que no está registrado en el hub
- **CUANDO** el cliente consulta o escribe `/api/settings/personal`
- **ENTONCES** responde `404` con mensaje en castellano
- **Y** el hub no se modifica

#### Scenario: Valor no booleano devuelve 400
- **DADO** un proyecto registrado
- **CUANDO** el cliente envía `PUT /api/settings/personal` con `"si"`
- **ENTONCES** responde `400` y el hub no se modifica

#### Scenario: Petición no local devuelve 403
- **DADO** una petición que no es local
- **CUANDO** el cliente envía `PUT /api/settings/personal`
- **ENTONCES** responde `403` y el hub no se modifica

---

### Requirement: Ecosistema Deja de Mostrar la Política de Ramas y, Junto con el Selector de Proyecto, Enlaza a «Ajustes del Proyecto» (RF-WEB-21)

La pantalla Ecosistema NO DEBE pintar la política de ramas (`git:` y `repositories[]`) de forma propia: DEBE retirar el uso de las filas de política de ramas (`branchPolicyRows`) y de `repoPolicyHtml`, y su tarjeta de Topología DEBE enlazar a «Ajustes del proyecto», que es el único lugar donde se consulta y edita. El selector de proyecto (`projects.js`, sin política de ramas) DEBE ofrecer un enlace a «Ajustes del proyecto». `GET /api/workspace` NO DEBE perder campos de los que dependan otras pantallas (RF-WEB-01). Las pruebas de cadenas esperadas del JS y de assets DEBEN actualizarse a este comportamiento y seguir en verde. El *service worker* no cambia.

#### Scenario: Ecosistema enlaza a Ajustes
- **DADO** un proyecto con `git.integration_branch: develop`
- **CUANDO** se abre Ecosistema
- **ENTONCES** no se pintan las filas de política de ramas
- **Y** hay un enlace a «Ajustes del proyecto»

#### Scenario: El selector de proyecto enlaza a Ajustes
- **DADO** el selector de proyecto con un proyecto activo
- **CUANDO** se abre
- **ENTONCES** ofrece un enlace a «Ajustes del proyecto»
- **Y** no pinta la política de ramas

#### Scenario: Las pruebas de assets reflejan la retirada
- **DADO** los assets embebidos
- **CUANDO** se ejecutan las pruebas de cadenas del JS
- **ENTONCES** ningún asset de Ecosistema referencia las filas de política de ramas ni `repoPolicyHtml`
- **Y** `GET /api/workspace` sigue exponiendo la política de ramas y `repositories`
