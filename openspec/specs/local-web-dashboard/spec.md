# Especificación Viva: Servidor HTTP Local Embebido y Dashboard Web

## Propósito

Definir de forma rigurosa, ejecutable y verificable los requerimientos funcionales y escenarios BDD para el servidor HTTP local en Go, la API REST JSON, los assets frontend embebidos (`//go:embed`) y el subcomando de la CLI `axiom ui` en el runtime de Axiom.

---

## 1. Capacidad: `axiom-local-dashboard-server`

El paquete `internal/dashboard` implementa el servidor HTTP ligero basado en `net/http` que agrega y expone el estado del espacio de trabajo y del ciclo SDD mediante una API REST en formato JSON.

### Requirement: Endpoint REST de Información del Workspace (REQ-1.1)

El servidor DEBE responder a la ruta `GET /api/workspace` retornando una estructura JSON con los metadatos del espacio de trabajo declarados en `axiom.yaml`: nombre, topología (`monorepo-embedded`, `multirepo`), repositorio canónico de especificaciones, lista de roles con sus rutas y tecnologías, la política de ramas efectiva (`integration_branch` y `branch_prefix`, con los valores por defecto aplicados cuando `axiom.yaml` no declara `git:`), las entradas de `repositories:` y el estado general de validación de la topología. La respuesta es de solo lectura: ninguna ruta del servidor DEBE modificar `axiom.yaml` (REQ-25.33).
(Previously: la topología admitía también `monorepo-decoupled` y la respuesta no incluía la política de ramas ni `repositories:`)

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

### Requirement: Endpoints REST de Consulta de Incrementos (REQ-1.2)

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

### Requirement: Endpoint REST de Estado Multi-Rol y Barrera (REQ-1.3)

El servidor DEBE responder a `GET /api/roles?change={name}` analizando el desglose de roles del cambio solicitado mediante `internal/multirole`, retornando la lista de roles, sus políticas de compuerta (`blocking`, `deferred`, `optional`), el avance de sus tareas (`tasks.<rol>.md`), el estado de verificación (`verify-report.<rol>.md`) y el dictamen global de la barrera de sincronización (*Satisfied* o *Blocked*).

#### Scenario: Consulta de roles y diagnóstico de barrera para un cambio
- **DADO** un cambio con roles concurrentes declarados en `design.md`
- **CUANDO** se realiza una petición `GET /api/roles?change={name}`
- **ENTONCES** el servidor retorna código `200 OK`
- **Y** el JSON incluye la evaluación de cada rol y el estado de la barrera de sincronización

---

### Requirement: Endpoint REST de Consulta de Handoffs (REQ-1.4)

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

### Requirement: Endpoint REST de Catálogo de Skills (REQ-1.5)

El servidor DEBE responder a `GET /api/skills` escaneando los directorios locales de skills (`skills/` e `internal/assets/skills/`) y retornando la lista de skills disponibles, sus nombres, rutas relativas y una descripción resumida extraída de sus respectivos encabezados `SKILL.md`.

#### Scenario: Listado exitoso de skills del proyecto
- **DADO** un proyecto con skills definidas en `skills/` o `internal/assets/skills/`
- **CUANDO** se solicita `GET /api/skills`
- **ENTONCES** el servidor devuelve `200 OK` con el array de skills detectadas y sus metadatos

---

## 2. Capacidad: `axiom-web-ui-assets`

El paquete `internal/dashboard` incorpora los ficheros estáticos de la interfaz web embebidos directamente en el binario compilado mediante `//go:embed assets/*`, garantizando portabilidad absoluta y cero dependencias de NodeJS o NPM.

### Requirement: Servido de Assets Estáticos Embebidos (REQ-2.1)

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

### Requirement: Estructura de Navegación y Vistas en la Web UI (REQ-2.2)

La interfaz web en `index.html` y `app.js` DEBE estructurarse en una experiencia de usuario fluida y reactiva en español que incluya:
1. Encabezado institucional con identidad de Axiom, nombre del workspace y badge de topología.
2. Navegación por pestañas (*tabs*):
   - **Tablero de Incrementos:** vista Kanban o lista con tarjetas interactivas agrupadas por fase SDD y porcentaje de tareas.
   - **Monitor Multi-Rol:** vista de detalle del cambio seleccionado con el estado de cada rol (`blocking` vs `deferred`), tareas y estado de la barrera.
   - **Visor de Handoffs:** renderizado claro de relevos con sus metadatos y acordeón de secciones.
   - **Topología y Salud:** inspección de repositorios asociados y repositorio de especificaciones.
   - **Catálogo de Skills:** visor de skills disponibles en el proyecto.

#### Scenario: Navegación interactiva entre vistas del dashboard
- **DADO** la interfaz web cargada en el navegador
- **CUANDO** el usuario selecciona una pestaña en la barra de navegación
- **ENTONCES** la vista correspondiente se activa dinámicamente sin recargar la página completa
- **Y** los datos se obtienen de los endpoints REST correspondientes

---

## 3. Capacidad: `axiom-cli-ui`

El comando `axiom ui` en la CLI de Axiom permite a los desarrolladores iniciar el servidor local y abrir el panel de control de forma desatendida o personalizada.

### Requirement: Subcomando axiom ui y Control del Servidor (REQ-3.1)

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
### Requirement: Crear un Cambio desde el Dashboard Invoca `ensure` Antes de Escribir (REQ-25.32)

La creación de un incremento desde el dashboard y con `axiom change create` DEBE ejecutar `axiom branch ensure` para el cambio, con el mismo motor que la CLI, y después escribir con el escritor guardado (REQ-25.19 de `change-branch-lifecycle`), antes de crear la carpeta del cambio o escribir `proposal.md`, y DEBE escribir los ficheros en la ruta del planning home que devuelve `ensure` (en modo `worktree`, la del worktree del cambio y no la del checkout principal). El escritor guardado se niega fuera de rama aunque se invoque sin `ensure`. La TUI solo lista cambios (REQ-25.34): no ejecuta `ensure` ni crea incrementos. Si `ensure` devuelve `decision_required` (modo `ask`), el servidor DEBE devolver ese resultado tipado sin escribir nada; la interfaz DEBE presentar la elección (`worktree` o `in_place`) y repetir la creación con la respuesta, que se registra por cambio (REQ-25.8). Si `ensure` se rechaza por cualquier causa (`dirty_worktree`, `path_too_long`, `invalid_name`, `specs_repo_not_on_integration`, etc.), el servidor DEBE devolver el código tipado y la continuación, sin crear la carpeta del cambio.

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

### Requirement: `git:` y `repositories:` en Solo Lectura; Solo se Editan las Preferencias Personales (REQ-25.33)

En esta versión, el dashboard NO DEBE ofrecer ni exponer ningún medio de editar `axiom.yaml`: `git:` y `repositories:` se muestran en solo lectura, con la indicación de que se editan en `axiom.yaml`. Ninguna ruta HTTP del servidor DEBE escribir `axiom.yaml`. Las únicas preferencias editables desde el dashboard son las personales del hub (modo de trabajo y raíz de worktrees, REQ-25.29), mediante lectura y escritura sobre el proyecto activo con la validación de REQ-25.29. Editar las preferencias NO DEBE modificar ningún fichero del proyecto.

#### Scenario: `git:` se muestra pero no se puede editar
- **DADO** la SPA con un proyecto que declara `git.integration_branch: develop`
- **CUANDO** el usuario abre la vista de topología
- **ENTONCES** se muestra `develop` en un control de solo lectura
- **Y** no existe ningún control de edición para `git:` ni `repositories:`

#### Scenario: Ninguna ruta escribe `axiom.yaml`
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

---

### Requirement: Los Listados Agregan los Cambios de los Worktrees Registrados (REQ-25.34)

Los cambios en curso viven en worktrees. El servidor DEBE agregar, a los cambios activos del checkout principal, los cambios activos de los worktrees registrados en el registro local de `axiom branch` (REQ-25.14) para el proyecto activo en `GET /api/increments`, y DEBE resolver también esos cambios en `GET /api/increments/{name}`, `GET /api/roles?change={name}` y `GET /api/handoffs?change={name}`. Cada incremento listado desde un worktree DEBE indicar la ruta absoluta de su worktree y su rama. Si un mismo cambio activo figura en el checkout principal (por ejemplo, un sumidero acumulativo) y en un worktree, prevalece el del worktree; si figura en el worktree de la rama del cambio y en el de la rama de archivado, prevalece el de la rama de archivado. El dashboard DEBE mostrar los cambios pendientes de archivar (REQ-25.20 de `change-branch-lifecycle`), es decir, los cambios activos presentes en la rama de integración que aún no se han archivado, con la continuación para archivarlos. El aviso de cambios pendientes de archivar DEBE viajar en la cabecera `X-Axiom-Notes` de la respuesta de `GET /api/increments`; los avisos de este requisito, incluido el de una entrada del registro cuyo worktree ya no existe en disco (que DEBE omitirse sin error), viajan en esa misma cabecera. Los cambios pendientes NO producen ningún error ni bloqueo. Las escrituras del dashboard sobre un cambio que vive en un worktree DEBEN dirigirse a ese worktree. La pantalla de ciclo de vida de incrementos de la TUI DEBE listar el mismo conjunto de cambios y solo lista: no escribe.
(Previously: el requisito se apoyaba en un invariante de la rama de integración, eliminado por D17, y los avisos viajaban en el cuerpo de la respuesta)

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

#### Scenario: La TUI lista el mismo conjunto
- **DADO** un proyecto con un cambio activo `x` en un worktree registrado
- **CUANDO** el usuario abre la pantalla de ciclo de vida de incrementos de la TUI
- **ENTONCES** la pantalla lista `x`, igual que `GET /api/increments`
