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
   - **Integraciones:** pantalla de activación y configuración de los plugins de integración, generada desde su manifiesto (RF-WEB-33), con entrada propia en la navegación.
   - **Catálogo de Skills:** visor de skills disponibles en el proyecto.
3. El detalle del cambio incluye una pestaña «Integraciones» con el work item enlazado, las Tasks por rol, la outbox y los actores (RF-WEB-34).

(Previously: la navegación no incluía «Integraciones» ni el detalle del cambio tenía pestaña de integraciones.)

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

#### Scenario: «Integraciones» tiene entrada de navegación y ruta propia
- **DADO** la interfaz cargada
- **CUANDO** el usuario selecciona «Integraciones» o abre la ruta por *hash* de esa pantalla
- **ENTONCES** se muestra la pantalla de integraciones sin recargar la página
- **Y** la entrada de navegación aparece marcada como activa

#### Scenario: El detalle del cambio ofrece la pestaña «Integraciones»
- **DADO** el detalle de un cambio abierto
- **CUANDO** se recorren sus pestañas
- **ENTONCES** existe la pestaña «Integraciones»

#### Scenario: Topología no muestra la política de ramas
- **DADO** un proyecto que declara `git.integration_branch: develop`
- **CUANDO** el usuario abre Ecosistema (tarjeta de Topología) o el selector de proyecto
- **ENTONCES** ninguna de esas vistas pinta las filas de política de ramas
- **Y** cada una ofrece un enlace a «Ajustes del proyecto»

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

El dashboard NO DEBE editar `axiom.yaml` por ninguna ruta que no sea `PUT /api/settings` (RF-WEB-19), que aplica el catálogo y el escritor de `workspace-topology` (RF-TOP-21 a RF-TOP-23) con validación del documento completo y revisión esperada. Esto incluye el bloque `integrations.<plugin>`: se edita solo por esa ruta. El PAT se guarda y se elimina por sus endpoints propios (RF-WEB-30) en el almacén de secretos del sistema y NUNCA toca `axiom.yaml`. `GET /api/workspace` DEBE seguir siendo de solo lectura: `POST`, `PUT`, `PATCH` y `DELETE` sobre `/api/workspace` DEBEN responder `405`. Las claves de solo lectura del catálogo (incluidas `workspace.topology` y `workspace.specs_repository`) NO DEBEN poder editarse desde el dashboard.

Las preferencias personales editables desde el dashboard son las del hub: modo de trabajo y raíz de worktrees (RF-HUB-07), mediante `worktree-prefs`, y el autofetch del proyecto (RF-HUB-13), mediante `/api/settings/personal` (RF-WEB-20), siempre sobre el proyecto activo y con la validación de RF-HUB-07. Editar las preferencias personales NO DEBE modificar ningún fichero del proyecto. Toda ruta del servidor que escribe, sea `PUT /api/settings`, `PUT /api/settings/personal`, `PUT /api/projects/active/worktree-prefs`, el guardado o la eliminación del PAT, la conexión o el alta con work item, DEBE exigir una petición local (`isLocalRequest`) con `Host` y `Origin` locales (RF-WEB-28) y rechazar cualquier otra con `403` antes de leer el cuerpo o escribir.

(Previously: `integrations.*` no era editable desde el dashboard, el PAT no tenía endpoint y la exigencia de petición local no incluía `Host` ni las rutas de integraciones.)

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

#### Scenario: `integrations.<plugin>` se edita solo por `/api/settings`
- **DADO** un plugin de integración disponible y una petición local
- **CUANDO** el usuario guarda `integrations[<plugin>].enabled` desde «Integraciones»
- **ENTONCES** el cambio viaja por `PUT /api/settings` y se escribe en `axiom.yaml`
- **Y** ninguna otra ruta modifica `axiom.yaml`

#### Scenario: El PAT nunca toca `axiom.yaml`
- **DADO** un PAT guardado por su endpoint propio
- **CUANDO** se compara `axiom.yaml` antes y después
- **ENTONCES** el fichero no cambia

#### Scenario: Las rutas de integraciones que escriben exigen petición local con `Host` y `Origin` locales
- **DADO** un cliente con `Origin` o `Host` ajeno
- **CUANDO** envía el guardado del PAT, la conexión o el alta con work item
- **ENTONCES** el servidor responde `403` sin escribir ni llamar al servicio externo

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
### Requirement: Pestaña «Pruebas» en el Detalle del Incremento, con Marcas por Entorno solo en Multirepo (RF-WEB-17)

`GET /api/increments/<n>` DEBE incluir en el detalle de un incremento SDD un campo de la batería de pruebas con el contenido de `pruebas.md` y su cobertura, calculada con el mismo `acceptancetests.Check` que usan `axiom sdd tests check` y `axiom sdd status` (RF-KOF-20). El campo DEBE incluir los casos de la tabla, las cifras de escenarios M/M, requisitos M/M y número de casos, y los hallazgos. Sin `pruebas.md`, el campo DEBE indicar su ausencia y, si la función está activa, el aviso de RF-KOF-22. La lectura NO DEBE añadir ninguna ruta, y el endpoint de ficheros de RF-WEB-15 NO DEBE ampliar su lista blanca. Las únicas rutas añadidas son las dos rutas POST de marcas (RF-WEB-23 y RF-WEB-24).
(Previously: la pestaña era de solo lectura y el requisito prohibía añadir cualquier ruta)

El detalle DEBE incluir una pestaña «Pruebas»: muestra la tabla de casos con sus columnas, la cobertura y los hallazgos; no ofrece edición de `pruebas.md` ni ejecución. Las marcas por entorno (RF-WEB-22) solo se ofrecen cuando están disponibles (RF-WEB-25); en los demás casos la pestaña sigue siendo de solo lectura. Todo texto proveniente de `pruebas.md` DEBE insertarse como texto, nunca como HTML. Con la función no activa y sin `pruebas.md`, la pestaña DEBE indicar que las pruebas de aceptación no están activadas para el proyecto. Los hallazgos nunca bloquean ninguna otra acción del detalle. Las pruebas de pines de pestañas y de assets, y el canario de secretos, DEBEN seguir en verde.

#### Scenario: El detalle devuelve la batería y su cobertura
- **DADO** un incremento con `pruebas.md` válido y su cobertura completa
- **CUANDO** se consulta `GET /api/increments/<n>`
- **ENTONCES** el detalle incluye los casos, los escenarios M/M, los requisitos M/M, el número de casos y cero hallazgos

#### Scenario: La cobertura coincide con la del verbo
- **DADO** un incremento con `pruebas.md` con hallazgos
- **CUANDO** se consulta el detalle y se ejecuta `axiom sdd tests check --json`
- **ENTONCES** cifras y hallazgos son iguales

#### Scenario: La pestaña es de solo lectura donde las marcas no están disponibles
- **DADO** un incremento con `pruebas.md` en un proyecto `monorepo-embedded`
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** se muestran la tabla, la cobertura y los hallazgos
- **Y** no hay controles de edición, de ejecución ni de marca

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

#### Scenario: La lectura no añade rutas y las de marcas son solo POST
- **DADO** el servidor del dashboard
- **CUANDO** se enumeran sus rutas
- **ENTONCES** la lectura de la batería no añade ninguna ruta GET, la lista blanca de RF-WEB-15 no cambia y las únicas rutas añadidas son las dos POST de marcas
### Requirement: Pantalla «Ajustes del Proyecto» Generada desde el Catálogo, con Reglas por Topología e ISO y sin Controles de RDD (RF-WEB-18)

El dashboard DEBE ofrecer una pantalla «Ajustes del proyecto» con ruta por *hash* y entrada en la navegación. La pantalla DEBE generarse desde el catálogo que devuelve `GET /api/settings` (RF-WEB-19) sin conocer las claves: cada entrada se pinta con su etiqueta, descripción, tipo, valores permitidos y efecto descrito, y un control acorde (selector para vocabularios cerrados, casilla para booleanos, número, texto o lista). Todo texto del catálogo y del `axiom.yaml` DEBE insertarse como texto, nunca como HTML.

La pantalla DEBE mostrar de forma visible el aviso «Editar solo afecta a cambios nuevos» y comunicar que los cambios ya sellados conservan su instantánea de `kickoff.yaml`; ninguna acción de la pantalla DEBE modificar un `kickoff.yaml` existente. Al guardar, DEBE enviar la revisión leída, mostrar el error del validador (castellano) sin perder lo tecleado cuando se rechaza, y pedir recargar cuando la revisión está obsoleta. Retirar un valor DEBE enviarse como `null`. La pantalla DEBE enviar todos los cambios pendientes en una sola petición (lote atómico, RF-WEB-19). Junto a la ruta del fichero DEBE indicar que lo guardado queda sin confirmar en git, en el árbol de trabajo del repositorio que contiene `axiom.yaml` (en `multirepo`, el de specs), y que confirmarlo es decisión de la persona. Cuando el servidor informa `writable: false` (documento con anclas, alias, `<<` o varios documentos YAML), DEBE deshabilitar los controles de edición y mostrar el motivo.

Las claves con `editable: false` (`technical_context`, `workspace.root`, `skill_roots`, `governance.*`, `workspace.topology`, `workspace.specs_repository`) DEBEN mostrarse sin control de edición, con su valor y la acción «Abrir en el editor»: un enlace `vscode://file/<ruta absoluta de axiom.yaml>:<línea>` etiquetado «Abrir en el editor», que abre el fichero en la línea de la clave cuando está declarada (sin `:<línea>` cuando no lo está), más un botón «Copiar ruta» que copia esa ruta absoluta, sin línea. Las claves conocidas de `governance` DEBEN mostrarse con nombre legible (`language`: «Idioma»; `shared_memory`: «Memoria compartida»); una clave desconocida se muestra con su ruta. El servidor NO DEBE lanzar ningún proceso para abrir el editor.

La pantalla DEBE incluir un bloque «Preferencias personales» con el autofetch (RF-WEB-20) y `worktree_mode` y `worktree_root` (estos dos sobre `worktree-prefs`, RF-WEB-10), identificado como personal y no versionado. En `monorepo-embedded` la pantalla NO DEBE mostrar `roles`, `gate_policy`, `repositories` ni `acceptance_tests`; en `multirepo` DEBE mostrar `repositories[]`, `acceptance_tests` y `roles[].gate_policy` con `optional` entre sus opciones. En un proyecto ISO, `workflow.lane` NO DEBE ofrecer `odd`. La pantalla NO DEBE incluir ningún control de RDD (global, por proyecto o por clon), ni `commit_review`, ni `docs_publication`. Tampoco DEBE pintar el bloque `integrations`: DEBE enlazar a «Integraciones» (RF-WEB-33), que es donde se consulta y edita.

(Previously: el enlace «Abrir en el editor» era `vscode://file/<ruta absoluta de axiom.yaml>`, sin línea, y las claves de `governance` se etiquetaban con su ruta.)

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
- **Y** aparece un enlace «Abrir en el editor» con destino `vscode://file/<ruta absoluta de axiom.yaml>:<línea de la clave>` y un botón «Copiar ruta» que copia la ruta sin línea
- **Y** `governance.language` y `governance.shared_memory` se etiquetan «Idioma» y «Memoria compartida»
- **Y** el servidor no lanza ningún proceso

#### Scenario: Clave de solo lectura sin declarar abre el fichero sin línea
- **DADO** un `axiom.yaml` sin `technical_context`
- **CUANDO** se inspecciona `technical_context`
- **ENTONCES** el enlace «Abrir en el editor» apunta a `vscode://file/<ruta absoluta de axiom.yaml>`, sin `:<línea>`

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

#### Scenario: Ningún control de RDD ni de integraciones, con enlace a «Integraciones»
- **DADO** la pantalla abierta en cualquier topología
- **CUANDO** se recorren todos sus controles
- **ENTONCES** no existe ningún control de RDD, de `commit_review` ni de `docs_publication`
- **Y** no se pinta ningún control de `integrations.*`, sino un enlace a «Integraciones»

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

### Requirement: Endpoint `GET|PUT /api/settings` con Catálogo, Revisión, Escritura Validada y Errores Tipados en Castellano (RF-WEB-19)

`GET /api/settings` DEBE devolver el catálogo (RF-TOP-21) ya filtrado por la topología y el carácter ISO del proyecto activo, con el valor actual de cada clave, la revisión de `axiom.yaml` (RF-TOP-23) y la ruta absoluta del fichero (la de `Locate().ConfigPath`). La respuesta incluye `integrations`: sus entradas (`integrations[<plugin>].<clave>` y `integrations[<plugin>].enabled`) con su valor leído de `axiom.yaml`, derivadas del manifiesto del plugin (RF-TOP-21). La respuesta NO DEBE contener valor alguno de credenciales o secretos: el PAT no es clave del catálogo y los secretos solo se informan por su procedencia en `GET /api/integrations` (RF-WEB-29).

(Previously: la respuesta no contenía `integrations` y `integrations.*` era `unknown_key`.)

`GET /api/settings` DEBE informar además si el documento es escribible (`writable`) y, si no lo es, el motivo (`writeBlockedReason`), y los errores que el documento ya tiene en disco (`documentErrors`); los valores se leen del documento tal como está escrito, de modo que un `axiom.yaml` inválido escrito a mano se muestra con su valor real.

`PUT /api/settings` DEBE recibir un cuerpo `{"revision": "<sha256>", "changes": {"<clave>": <valor>|null, ...}}` con la revisión esperada y un parche de claves del catálogo (con `null` para retirar) y aplicarlo con el escritor (RF-TOP-22). Las claves por elemento se direccionan en el parche como `repositories[<path>].<campo>` y `roles[<id>].<campo>`. Todos los cambios del parche se aplican y validan juntos, como un lote atómico de todo o nada: si cualquiera es rechazado, no se escribe ninguno. DEBE exigir petición local y comprobarla antes de leer el cuerpo (RF-WEB-10). Un fichero con varios documentos YAML, anclas, alias o `<<` DEBE rechazarse, porque reescribirlo perdería o alteraría contenido.

Todo error DEBE llevar un código tipado y mensaje en castellano, con el cuerpo `{"code", "error", "violations"?}`, según esta correspondencia:

| Código | HTTP | Cuándo |
|---|---|---|
| `invalid_request` | `400` | cuerpo mal formado, revisión esperada ausente o `changes` vacío |
| `config_missing` | `404` | no hay proyecto activo o `axiom.yaml` no existe |
| `revision_mismatch` | `409` | revisión esperada obsoleta o cambio externo antes del reemplazo |
| `unsupported_document` | `422` | anclas, alias, `<<`, varios documentos YAML o clave repetida |
| `unknown_key` | `422` | clave que no está en el catálogo (incluida una clave de `integrations` que el manifiesto del plugin no declara, un plugin desconocido o una clave de PAT o de secreto) |
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
- **Y** incluye las entradas de `integrations[<plugin>].<clave>` con su valor actual

#### Scenario: GET no filtra secretos
- **DADO** un `axiom.yaml` con `integrations` y una credencial centinela en el entorno y en el almacén
- **CUANDO** el cliente consulta `GET /api/settings`
- **ENTONCES** la respuesta no contiene la credencial
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
- **DADO** una petición con `workspace.topology`
- **CUANDO** el cliente envía el `PUT`
- **ENTONCES** responde `422` con el código de clave no editable
- **Y** el fichero no cambia

#### Scenario: Una clave de integración del manifiesto se escribe
- **DADO** el plugin `azuredevops` con un campo `organization` declarado en su manifiesto y una petición local
- **CUANDO** el cliente envía `PUT /api/settings` con `integrations[azuredevops].organization`
- **ENTONCES** responde `200` y `axiom.yaml` contiene el valor en `integrations.azuredevops`
- **Y** el documento vuelve a cargarse como `COMPLIANT` con el decodificador estricto

#### Scenario: Una clave de integración no declarada por el manifiesto devuelve 422
- **DADO** una petición con `integrations[azuredevops].no_existe` o con un plugin desconocido
- **CUANDO** el cliente envía el `PUT`
- **ENTONCES** responde `422` con `unknown_key` y el fichero no cambia

#### Scenario: Un valor fuera del manifiesto devuelve 422
- **DADO** un campo de enumeración, un número con mínimo y máximo o un formato del manifiesto
- **CUANDO** el cliente envía un valor que el manifiesto no admite
- **ENTONCES** responde `422` con `invalid_value` nombrando la clave y el fichero no cambia

#### Scenario: El PAT no es clave del catálogo
- **DADO** una petición con una clave de PAT o de secreto de integración
- **CUANDO** el cliente envía el `PUT`
- **ENTONCES** responde `422` con `unknown_key` y el fichero no cambia

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
### Requirement: Pantalla de Marcas por Caso y Entorno con Caducidad en la Pestaña «Pruebas» (RF-WEB-22)

Cuando las marcas están disponibles (RF-WEB-25), la pestaña «Pruebas» DEBE mostrar una tabla de casos con una columna por entorno de `acceptance_tests.environments`. Cada celda DEBE mostrar la última marca vigente del par caso-entorno (resultado y nota), o su ausencia. Una marca cuyo `plan_digest` difiere del digest actual de `pruebas.md` (RF-KOF-25) DEBE mostrarse como caducada y NO DEBE contar como vigente. La pestaña DEBE ofrecer en cada celda aplicable el control para marcar `ok`, `ko`, `blocked` o `n/a`, con un campo opcional `commit` (vacío por defecto) y un campo opcional `note`. DEBE listar las entradas pendientes de publicar. Todo texto de marcas, notas y `pruebas.md` DEBE insertarse como texto, nunca como HTML. Las celdas de un entorno no aplicable al caso NO DEBEN ofrecer el control de marca.

#### Scenario: Celda con marca vigente
- **DADO** un cambio con `pruebas.md` y una marca `ok` del caso `P-01` en `pre` con el `plan_digest` actual
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** la celda de `P-01` en `pre` muestra `ok` y su nota, sin indicador de caducidad

#### Scenario: Marca caducada tras cambiar `pruebas.md`
- **DADO** una marca de `P-01` en `pre` y un `pruebas.md` modificado después
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** la marca aparece como caducada y no cuenta como vigente

#### Scenario: Entorno no aplicable al caso
- **DADO** un caso cuyos entornos aplicables no incluyen `pro`
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** la celda de ese caso en `pro` no ofrece control de marca

#### Scenario: Contenido hostil en una nota
- **DADO** una marca cuya nota contiene `<script>`
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** la nota aparece literal y no se interpreta como HTML

#### Scenario: Entradas pendientes visibles
- **DADO** dos marcas escritas en local y aún no publicadas
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** se listan ambas como pendientes de publicar

### Requirement: Ruta de Marca con Validación en Servidor y Escritura Local de Solo Añadir (RF-WEB-23)

El dashboard DEBE exponer una ruta POST de marca, protegida con `isLocalRequest` y con cuerpo JSON, que añada una entrada a `pruebas-estado.yaml` del cambio (formato en RF-KOF-25). La ruta DEBE validar en el servidor, sin escribir nada si falla: que el cambio exista, no esté archivado y tenga `pruebas.md`; que `case` exista en `pruebas.md`; que `environment` pertenezca a `acceptance_tests.environments` y a los entornos aplicables del caso; que `result` pertenezca a `ok|ko|blocked|n/a`; y que `note` no supere la longitud máxima. DEBE rellenar `by` con la identidad git (`user.name` y `user.email`) del repositorio de specs, `at` con la fecha y hora, y `plan_digest` con el digest actual de `pruebas.md`; `commit` es el valor opcional que da la persona y queda vacío por defecto, y NUNCA se infiere del repositorio de código. La escritura DEBE ser atómica y con exclusión mutua por cambio, y el fichero DEBE ser de solo añadir: la ruta NO DEBE editar ni borrar entradas existentes y una corrección es una entrada nueva. La ruta NO DEBE ejecutar ninguna operación git de escritura ni de red. Cada rechazo DEBE tener código tipado, mensaje en castellano y continuación.

#### Scenario: Marca válida añade una entrada
- **DADO** un cambio SDD multirepo con `pruebas.md` con el caso `P-01` aplicable a `pre`
- **CUANDO** se envía la marca `P-01`, `pre`, `ok`
- **ENTONCES** `pruebas-estado.yaml` recibe una entrada con `by` (identidad git), `at`, `commit` vacío y el `plan_digest` actual
- **Y** el observador de git no registra ninguna operación

#### Scenario: Caso inexistente
- **DADO** un `pruebas.md` sin el caso `P-99`
- **CUANDO** se envía una marca para `P-99`
- **ENTONCES** se rechaza con un código tipado de caso inexistente y su continuación
- **Y** el fichero no cambia

#### Scenario: Entorno no válido
- **DADO** `acceptance_tests.environments` con `local` y `pre`
- **CUANDO** se envía una marca en `staging`, o en un entorno no aplicable al caso
- **ENTONCES** se rechaza con un código tipado de entorno no válido
- **Y** el fichero no cambia

#### Scenario: Resultado fuera de dominio
- **DADO** una marca con `result` igual a `pass`
- **CUANDO** se envía
- **ENTONCES** se rechaza con un código tipado de resultado fuera de dominio
- **Y** el fichero no cambia

#### Scenario: Nota demasiado larga
- **DADO** una `note` que supera la longitud máxima
- **CUANDO** se envía la marca
- **ENTONCES** se rechaza con un código tipado de nota demasiado larga y el fichero no cambia

#### Scenario: Solo añadir
- **DADO** una marca `ko` ya escrita para `P-01` en `pre`
- **CUANDO** se envía una nueva marca `ok` para el mismo par
- **ENTONCES** el fichero conserva la entrada `ko` y añade la entrada `ok` después
- **Y** la celda muestra `ok` como marca vigente

#### Scenario: Petición no local
- **DADO** una petición con `Host` u `Origin` que no son del propio dashboard
- **CUANDO** se envía a la ruta de marca
- **ENTONCES** se rechaza y no se escribe nada

#### Scenario: Escrituras concurrentes del mismo cambio
- **DADO** dos marcas enviadas a la vez para el mismo cambio
- **CUANDO** el servidor las procesa
- **ENTONCES** `pruebas-estado.yaml` contiene ambas entradas íntegras y bien formadas

### Requirement: Ruta de Publicación de Marcas con Confirmación Explícita y Resultado Tipado (RF-WEB-24)

El dashboard DEBE exponer una ruta POST «Publicar marcas», protegida con `isLocalRequest` y con cuerpo JSON, que exija el campo `confirm` verdadero en el cuerpo. Sin `confirm`, DEBE rechazarse con un código tipado y NO DEBE ejecutar ninguna operación git. Antes del clic, la UI DEBE mostrar el remoto, la rama de integración del repositorio de specs y las entradas pendientes. Con `confirm`, el dashboard DEBE delegar la publicación en un publicador de marcas definido por una interfaz propia del dashboard e inyectado desde fuera; el dashboard NO DEBE importar `sddpublish` ni ejecutar git por su cuenta (RF-PUB-07). La respuesta DEBE mostrar el resultado (commit, rama y remoto) o el rechazo tipado con su etiqueta en castellano y su continuación, y DEBE avisar de los demás documentos sucios del repositorio de specs que no se confirmaron (RF-PUB-13). Sin entradas pendientes, la respuesta DEBE indicar que no hay nada que publicar y no hacer commit. Nunca DEBE haber push sin el clic de confirmación.

#### Scenario: Publicar sin confirm no ejecuta git
- **DADO** entradas pendientes y un observador de git
- **CUANDO** se envía la publicación sin `confirm`
- **ENTONCES** se rechaza con un código tipado de confirmación ausente
- **Y** el observador no registra ninguna operación git

#### Scenario: Publicar con confirm muestra el resultado
- **DADO** entradas pendientes y un repositorio de specs con remoto
- **CUANDO** se envía la publicación con `confirm`
- **ENTONCES** la respuesta incluye el commit creado, la rama y el remoto

#### Scenario: Datos visibles antes del clic
- **DADO** entradas pendientes
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** se muestran el remoto, la rama de integración y las entradas pendientes antes de ofrecer «Publicar marcas»

#### Scenario: Rechazo tipado visible
- **DADO** una publicación que termina con un rechazo tipado (por ejemplo, `push_non_fast_forward` tras el reintento)
- **CUANDO** el dashboard responde
- **ENTONCES** muestra el código, la etiqueta en castellano y la continuación
- **Y** las entradas siguen en el fichero local

#### Scenario: Sin entradas pendientes
- **DADO** que todas las marcas ya están publicadas
- **CUANDO** se envía la publicación con `confirm`
- **ENTONCES** la respuesta indica que no hay nada que publicar y no se crea ningún commit

#### Scenario: El dashboard no importa el publicador
- **DADO** el código fuente del producto
- **CUANDO** se ejecuta la prueba de aislamiento de RF-PUB-07
- **ENTONCES** el paquete del dashboard no importa `sddpublish` y la prueba sigue en verde

### Requirement: Marcas Solo en Multirepo SDD con Estado «No Disponible» Explícito (RF-WEB-25)

Las marcas y sus dos rutas DEBEN estar disponibles únicamente para un cambio SDD activo con `pruebas.md` en un proyecto `multirepo` con `acceptance_tests` activo. En `monorepo-embedded`, la pestaña y las dos rutas DEBEN responder un estado «no disponible» explícito con código tipado y motivo en castellano, NO un 404 ni un error genérico. Para un cambio ODD, un cambio archivado, un cambio sin `pruebas.md` o un proyecto sin `acceptance_tests` activo, DEBEN responder igualmente «no disponible» con su motivo y NO DEBEN escribir nada. Las marcas NO DEBEN usar ninguna clave nueva de `axiom.yaml`: reutilizan `acceptance_tests.environments`.

#### Scenario: Monorepo responde no disponible
- **DADO** un proyecto `monorepo-embedded` con `pruebas.md`
- **CUANDO** se abre la pestaña «Pruebas» y se envía una marca
- **ENTONCES** ambas responden «no disponible» con motivo de topología
- **Y** no se escribe `pruebas-estado.yaml`

#### Scenario: Cambio archivado
- **DADO** un cambio archivado
- **CUANDO** se envía una marca
- **ENTONCES** se responde «no disponible» por cambio archivado y no se escribe nada

#### Scenario: Sin `pruebas.md`
- **DADO** un cambio multirepo sin `pruebas.md`
- **CUANDO** se envía una marca
- **ENTONCES** se responde «no disponible» por falta de `pruebas.md` y no se escribe nada

#### Scenario: Función `acceptance_tests` no activa
- **DADO** un proyecto multirepo sin `acceptance_tests` activo
- **CUANDO** se envía una marca
- **ENTONCES** se responde «no disponible» por función no activa

### Requirement: Rutas de Marcas Clasificadas en el Canario de Secretos (RF-WEB-26)

Las dos rutas POST nuevas DEBEN clasificarse en el canario de secretos como excluidas con el motivo «mutación (POST)», de modo que la prueba que exige clasificar toda ruta de `server.go` siga en verde. Ningún DTO de marcas DEBE contener secretos, y las pruebas de pines de pestañas y de assets DEBEN seguir en verde.

#### Scenario: Rutas clasificadas
- **DADO** las rutas registradas en `server.go`
- **CUANDO** se ejecuta la prueba de clasificación del canario
- **ENTONCES** las rutas de marca y de publicación figuran como excluidas con el motivo «mutación (POST)» y la prueba pasa

#### Scenario: Una ruta nueva sin clasificar falla
- **DADO** una ruta añadida sin clasificar
- **CUANDO** se ejecuta la prueba de clasificación
- **ENTONCES** la prueba falla

### Requirement: Cabeceras de Seguridad Generales con CSP Fija, sin CORS y sin Respuesta Genérica a `OPTIONS` (RF-WEB-27)

Toda respuesta del servidor del dashboard DEBE llevar una `Content-Security-Policy` general con exactamente estas directivas: `default-src 'self'`, `script-src 'self'`, `img-src 'self' data:`, `font-src 'self'`, `connect-src 'self'`, `object-src 'none'`, `base-uri 'none'`, `form-action 'self'` y `frame-ancestors 'none'`, más una directiva `style-src` (cuyo valor fija el diseño). La política NO DEBE incluir `'unsafe-inline'` ni `'unsafe-eval'` en `script-src`, ni ningún origen externo. El script de tema de `index.html` DEBE cargarse desde un fichero propio embebido, de forma síncrona en `<head>`, y `index.html` NO DEBE contener ningún `<script>` con código en línea.

La ruta `/api/explainer` DEBE conservar su CSP propia (RF-WEB-15), que prevalece sobre la general y NO DEBE duplicarse en la cabecera. Ninguna respuesta DEBE llevar cabeceras `Access-Control-*`. El servidor NO DEBE responder `200` de forma genérica a `OPTIONS`: una petición `OPTIONS` DEBE responderse sin cabeceras `Access-Control-*`.

#### Scenario: Toda respuesta lleva la CSP general
- **DADO** el servidor en ejecución
- **CUANDO** un cliente consulta `GET /`, un asset estático y una ruta `GET /api/...` cualquiera
- **ENTONCES** cada respuesta lleva una `Content-Security-Policy` con las nueve directivas fijas indicadas
- **Y** `script-src` vale exactamente `'self'`

#### Scenario: Ninguna respuesta lleva cabeceras CORS
- **DADO** el servidor en ejecución
- **CUANDO** un cliente envía una petición con `Origin` ajeno a una ruta `GET` y a una ruta que muta
- **ENTONCES** ninguna respuesta contiene cabecera alguna que empiece por `Access-Control-`

#### Scenario: `OPTIONS` no responde con CORS ni con un 200 genérico
- **DADO** el servidor en ejecución
- **CUANDO** un cliente envía `OPTIONS` a una ruta de la API
- **ENTONCES** la respuesta no contiene cabeceras `Access-Control-*`
- **Y** el servidor no la trata como una petición aceptada para el origen que la envía

#### Scenario: El script de tema ya no está en línea
- **DADO** el `index.html` embebido
- **CUANDO** se inspecciona su contenido
- **ENTONCES** no contiene ningún `<script>` con cuerpo en línea
- **Y** el script de tema se carga con `src` propio, de forma síncrona, antes de la hoja de estilos de tokens

#### Scenario: `/api/explainer` conserva su CSP propia
- **DADO** una petición a `/api/explainer` que sirve un resumen
- **CUANDO** se leen las cabeceras de la respuesta
- **ENTONCES** lleva la CSP propia de RF-WEB-15 una sola vez
- **Y** no se le añade la CSP general

### Requirement: `Host` y `Origin` Locales en las Rutas que Mutan y en Todas las Rutas `/api/integrations*` (RF-WEB-28)

Toda ruta que muta estado (cualquier método distinto de `GET`, `HEAD` y `OPTIONS`) y toda ruta `/api/integrations*`, también con `GET`, DEBEN exigir que la cabecera `Host` designe un destino local (bucle local o el `Host` del propio servidor) y que, si la petición lleva `Origin`, este sea local. Una petición con `Host` u `Origin` ajeno DEBE rechazarse con `403` antes de leer el cuerpo, de resolver secretos o de escribir nada. Esta comprobación se suma a la de petición local de RF-WEB-10.

#### Scenario: `Origin` ajeno a una ruta que muta se rechaza
- **DADO** el servidor en ejecución
- **CUANDO** un cliente envía `PUT /api/settings` con `Origin: https://sitio-ajeno.invalid`
- **ENTONCES** el servidor responde `403`
- **Y** `axiom.yaml` no se modifica

#### Scenario: `Host` ajeno a una ruta que muta se rechaza
- **DADO** el servidor en ejecución
- **CUANDO** un cliente envía una petición que muta con una cabecera `Host` que no es local
- **ENTONCES** el servidor responde `403` antes de leer el cuerpo

#### Scenario: `Host` ajeno a `GET /api/integrations` se rechaza
- **DADO** un plugin de integraciones activo con un PAT guardado
- **CUANDO** un cliente envía `GET /api/integrations` con `Host` ajeno
- **ENTONCES** el servidor responde `403`
- **Y** la respuesta no contiene datos de integración alguno

#### Scenario: `Origin` ajeno a `GET /api/integrations/...` se rechaza
- **DADO** el servidor en ejecución
- **CUANDO** un cliente envía `GET` a cualquier ruta `/api/integrations*` con `Origin` ajeno
- **ENTONCES** el servidor responde `403`

#### Scenario: Una petición local sin `Origin` se atiende
- **DADO** una petición con `Host` local y sin cabecera `Origin`
- **CUANDO** el cliente consulta `GET /api/integrations`
- **ENTONCES** el servidor la atiende con normalidad

### Requirement: `GET /api/integrations` Devuelve Manifiesto, Valores, Estado del Secreto e Identidad sin Revelar Nunca un Secreto (RF-WEB-29)

`GET /api/integrations` DEBE devolver, para cada plugin de integración disponible, su manifiesto (campos de configuración, tipos de work item, eventos, estados y jornada), los valores actuales de `integrations.<plugin>` leídos de `axiom.yaml`, el estado del secreto y la identidad resuelta persistida. El estado del secreto DEBE limitarse a su procedencia (`source`), con uno de los valores `env`, `keyring`, `missing` o `keyring_unavailable`, y NUNCA incluir el valor del secreto, un fragmento ni su longitud. El descubrimiento de proyectos, tipos y estados DEBE servirse bajo demanda y no en la consulta general.

La respuesta NO DEBE contener el PAT ni ningún valor derivado de él, tampoco en los mensajes de error. Cuando no hay proyecto activo o `axiom.yaml` no existe, DEBE responder `404` con código tipado; para métodos distintos de `GET` DEBE responder `405`.

#### Scenario: El estado del secreto es solo la procedencia
- **DADO** un PAT guardado en el almacén del sistema
- **CUANDO** el cliente consulta `GET /api/integrations`
- **ENTONCES** el plugin informa `source: keyring`
- **Y** el cuerpo no contiene el valor del PAT ni ningún fragmento

#### Scenario: Una variable de entorno que tapa el almacén se indica
- **DADO** la variable de entorno del PAT definida y un PAT guardado en el almacén
- **CUANDO** el cliente consulta `GET /api/integrations`
- **ENTONCES** `source` vale `env`

#### Scenario: Sin PAT disponible
- **DADO** ni variable de entorno ni PAT guardado
- **CUANDO** el cliente consulta `GET /api/integrations`
- **ENTONCES** `source` vale `missing`

#### Scenario: Almacén no disponible
- **DADO** un sistema donde el almacén de secretos no está disponible
- **CUANDO** el cliente consulta `GET /api/integrations`
- **ENTONCES** `source` vale `keyring_unavailable`

#### Scenario: Centinela de secreto no aparece en la respuesta
- **DADO** un PAT centinela guardado y una identidad persistida
- **CUANDO** el cliente consulta `GET /api/integrations`
- **ENTONCES** el cuerpo no contiene el centinela
- **Y** incluye la identidad resuelta persistida

#### Scenario: Sin proyecto activo
- **DADO** que no hay proyecto activo o que su `axiom.yaml` no existe
- **CUANDO** el cliente consulta `GET /api/integrations`
- **ENTONCES** responde `404` con código tipado

#### Scenario: Métodos no admitidos
- **DADO** el servidor en ejecución
- **CUANDO** el cliente envía `POST`, `PUT` o `DELETE` a `/api/integrations`
- **ENTONCES** responde `405`

### Requirement: Guardar y Eliminar el PAT por Endpoints Propios que Responden Solo `source`, con Guía de Variable de Entorno cuando No Hay Almacén (RF-WEB-30)

El dashboard DEBE ofrecer un endpoint propio para guardar el PAT (`POST`, solo JSON, cuerpo acotado, petición local con `Host` y `Origin` locales) y otro para eliminarlo. El PAT DEBE guardarse únicamente en el almacén de secretos del sistema y las dos operaciones DEBEN responder solo con la procedencia (`source`) resultante, nunca con el valor. El PAT NO DEBE escribirse en `axiom.yaml`, en ningún otro fichero versionado, en logs ni en ninguna respuesta HTTP, y NO DEBE ser clave del catálogo de ajustes.

Cuando el sistema no tiene un almacén escribible (por ejemplo, Linux sin servicio de claves), el guardado DEBE fallar con un error tipado que no simule éxito y que guíe a definir la variable de entorno derivada del manifiesto del plugin (para Azure DevOps, `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`); la interfaz DEBE mostrar esa guía. Si una variable de entorno tapa el valor del almacén, la respuesta DEBE reflejarlo con `source: env`. Un cuerpo que no sea JSON, que supere el límite o que no incluya el valor DEBE rechazarse con `400` en castellano sin tocar el almacén.

#### Scenario: Guardar el PAT responde solo `source`
- **DADO** una petición local con un PAT centinela válido
- **CUANDO** el cliente envía el `POST` de guardado
- **ENTONCES** responde `200` con `source: keyring`
- **Y** el cuerpo no contiene el centinela

#### Scenario: El PAT no llega a `axiom.yaml`
- **DADO** un PAT guardado desde el dashboard
- **CUANDO** se inspeccionan `axiom.yaml` y los ficheros versionados del repositorio
- **ENTONCES** ninguno contiene el PAT
- **Y** `axiom.yaml` solo puede contener la procedencia declarada (`source`), nunca el valor

#### Scenario: Linux sin almacén escribible guía a la variable de entorno
- **DADO** un sistema donde el almacén no admite escritura (`ErrNoWritableStore`)
- **CUANDO** el cliente envía el guardado del PAT
- **ENTONCES** responde con un error tipado en castellano que nombra la variable de entorno derivada del manifiesto
- **Y** no se guarda nada ni se simula un guardado, y la interfaz muestra esa guía

#### Scenario: Eliminar el PAT
- **DADO** un PAT guardado en el almacén
- **CUANDO** el cliente envía la eliminación
- **ENTONCES** responde con la procedencia resultante (`missing` o `env`) sin ningún valor

#### Scenario: Petición con `Origin` ajeno no guarda
- **DADO** una petición de guardado con `Origin` ajeno
- **CUANDO** el servidor la procesa
- **ENTONCES** responde `403` y el almacén no cambia

#### Scenario: Cuerpo inválido o vacío
- **DADO** un cuerpo no JSON, vacío o por encima del límite
- **CUANDO** el cliente envía el guardado
- **ENTONCES** responde `400` en castellano y el almacén no cambia

#### Scenario: El PAT no es clave editable
- **DADO** el catálogo de ajustes
- **CUANDO** se busca cualquier clave de PAT o secreto
- **ENTONCES** no existe, y una escritura por `PUT /api/settings` que la nombre se rechaza como clave desconocida

### Requirement: «Probar Conexión» Resuelve y Persiste la Identidad con la Misma Semántica que `axiom integrations connect` (RF-WEB-31)

El dashboard DEBE ofrecer una ruta para probar la conexión de un plugin con la misma semántica que `axiom integrations connect`: resolver la identidad con el secreto disponible, mostrarla y persistirla en la identidad local no versionada del usuario (`~/.axiom/integration-identities.json`), nunca en `axiom.yaml`. La ruta DEBE exigir petición local con `Host` y `Origin` locales y DEBE acotar su duración con un plazo propio inferior a la espera de escritura del servidor.

Cuando la resolución produce varios candidatos (`IdentitySelectionError`), la respuesta DEBE devolverlos y la interfaz DEBE permitir elegir uno y reenviar la elección, que se persiste. Cuando falla la autenticación, el plazo vence o no hay PAT, la respuesta DEBE ser un error tipado en castellano sin revelar el secreto.

#### Scenario: Conexión correcta muestra y persiste la identidad
- **DADO** un PAT válido y un doble del servicio de work items que resuelve una identidad
- **CUANDO** el cliente prueba la conexión
- **ENTONCES** la respuesta contiene la identidad resuelta
- **Y** la identidad queda persistida en el almacén local de identidades y no en `axiom.yaml`

#### Scenario: Varios candidatos permiten elegir
- **DADO** un doble que devuelve varias identidades candidatas
- **CUANDO** el cliente prueba la conexión
- **ENTONCES** la respuesta lista los candidatos
- **Y** al reenviar la elección de uno, esa identidad queda persistida

#### Scenario: Sin PAT disponible
- **DADO** ningún PAT disponible (`source: missing`)
- **CUANDO** el cliente prueba la conexión
- **ENTONCES** responde con un error tipado en castellano que indica guardar el PAT
- **Y** no se llama al servicio externo

#### Scenario: Plazo agotado
- **DADO** un doble que no responde dentro del plazo de la ruta
- **CUANDO** el cliente prueba la conexión
- **ENTONCES** responde con un error tipado de plazo agotado en castellano antes de que venza la espera de escritura del servidor

#### Scenario: Petición con `Host` ajeno
- **DADO** una petición con `Host` ajeno
- **CUANDO** el cliente prueba la conexión
- **ENTONCES** responde `403` sin resolver ningún secreto

### Requirement: «Descubrir» Rellena los Desplegables con el Descubrimiento del Plugin (RF-WEB-32)

El dashboard DEBE ofrecer una ruta de descubrimiento que, bajo demanda y con el secreto disponible, devuelva las opciones que el plugin descubre (por ejemplo proyectos, tipos de work item y estados) para rellenar los desplegables de la pantalla de integraciones. La ruta DEBE exigir `Host` y `Origin` locales, acotar su duración con un plazo propio y devolver errores tipados en castellano sin revelar el secreto. Descubrir NO DEBE escribir `axiom.yaml`: la configuración solo se escribe con `PUT /api/settings`.

#### Scenario: Descubrimiento correcto
- **DADO** un PAT válido y un doble de descubrimiento con proyectos, tipos y estados
- **CUANDO** el cliente pide descubrir
- **ENTONCES** la respuesta contiene esas opciones
- **Y** `axiom.yaml` no se modifica

#### Scenario: Sin PAT disponible
- **DADO** ningún PAT disponible
- **CUANDO** el cliente pide descubrir
- **ENTONCES** responde con un error tipado en castellano que indica guardar el PAT

#### Scenario: Plazo agotado
- **DADO** un doble que no responde en plazo
- **CUANDO** el cliente pide descubrir
- **ENTONCES** responde con un error tipado de plazo agotado y los desplegables conservan su valor anterior

### Requirement: Pantalla «Integraciones» Generada desde el Manifiesto del Plugin, Genérica por Plugin (RF-WEB-33)

El dashboard DEBE ofrecer una pantalla «Integraciones» con entrada en la navegación y ruta por *hash*, generada desde el manifiesto que devuelve `GET /api/integrations` sin conocer ningún plugin concreto. Por cada plugin DEBE ofrecer: activación (`integrations[<plugin>].enabled`), conexión (guardar y eliminar el PAT, mostrar `source`, «Probar conexión» e identidad), y los campos del manifiesto (tipos, eventos, estados, tags y jornada) con un control acorde a su tipo, obligatoriedad, valores permitidos, formato y límites. Los valores se guardan por `PUT /api/settings` en un lote validado por el decodificador estricto (RF-WEB-19). Todo texto del manifiesto, de la configuración y del servicio externo DEBE insertarse como texto, nunca como HTML.

La pantalla NO DEBE duplicar los controles de `branch_template`: DEBE enlazar a «Ajustes del proyecto». Cuando el almacén no es escribible, DEBE mostrar la guía de la variable de entorno y no ofrecer un guardado simulado. Cuando `source` es `env`, DEBE indicar que la variable tapa el almacén.

#### Scenario: La pantalla se genera desde el manifiesto
- **DADO** un manifiesto con un campo de enumeración, uno obligatorio y uno numérico con mínimo y máximo
- **CUANDO** se abre «Integraciones»
- **ENTONCES** el campo de enumeración se pinta como selector con exactamente sus valores, el obligatorio se marca como tal y el numérico respeta su mínimo y máximo
- **Y** la pantalla no contiene literales del plugin que no vengan del manifiesto

#### Scenario: Activar y configurar solo desde la UI
- **DADO** un proyecto sin bloque `integrations`
- **CUANDO** la persona activa el plugin, completa los campos obligatorios y guarda
- **ENTONCES** `axiom.yaml` contiene `integrations.<plugin>` con esos valores y vuelve a cargarse como `COMPLIANT` con el decodificador estricto

#### Scenario: Un valor inválido no pierde lo tecleado
- **DADO** un valor rechazado por el validador
- **CUANDO** la persona guarda
- **ENTONCES** la pantalla muestra el error en castellano y conserva lo tecleado

#### Scenario: Entrada de navegación y ruta propia
- **DADO** la interfaz cargada
- **CUANDO** el usuario selecciona «Integraciones» o abre su ruta por *hash*
- **ENTONCES** se muestra la pantalla sin recargar y la entrada aparece marcada como activa

#### Scenario: Las plantillas de rama no se duplican
- **DADO** la pantalla abierta
- **CUANDO** se recorren sus controles
- **ENTONCES** no existe ningún control de `branch_template`
- **Y** hay un enlace a «Ajustes del proyecto»

#### Scenario: Linux sin almacén muestra la guía y no simula guardado
- **DADO** una respuesta del servidor que indica almacén no escribible
- **CUANDO** la persona intenta guardar el PAT
- **ENTONCES** la pantalla muestra la variable de entorno derivada del manifiesto
- **Y** no muestra el PAT como guardado

#### Scenario: Variable de entorno que tapa el almacén
- **DADO** `source: env`
- **CUANDO** se abre la conexión
- **ENTONCES** la pantalla indica que la variable de entorno tiene prioridad sobre el almacén

#### Scenario: Texto hostil del manifiesto o del servicio externo se muestra como texto
- **DADO** una etiqueta o un nombre descubierto con `<script>` en su valor
- **CUANDO** se abre la pantalla
- **ENTONCES** el valor aparece literal y no se interpreta como HTML

### Requirement: Pestaña «Integraciones» en el Detalle del Cambio con WI, Tasks por Rol, Outbox y Actores (RF-WEB-34)

El detalle del cambio DEBE incluir una pestaña «Integraciones», también para cambios archivados (`Folder`), que muestre: el work item enlazado (identificador, título si existe y enlace); por cada rol, su Task enlazada con la estimación y las imputaciones de tiempo registradas y el tiempo pendiente de imputar; las operaciones pendientes de la outbox del cambio (de solo lectura); y los actores por fase (quién selló el kickoff, quién decidió cada puerta y quién imputó cada tiempo). La pestaña NO DEBE ejecutar ni reintentar operaciones de la outbox ni modificar vínculos. Sin vínculos ni plugin activo, DEBE mostrar un estado vacío explícito. Los datos NO DEBEN contener secretos y todo texto DEBE insertarse como texto.

#### Scenario: Cambio con WI y Tasks por rol
- **DADO** un cambio enlazado a un work item con dos roles que tienen Task, estimación y dos imputaciones
- **CUANDO** se abre la pestaña «Integraciones»
- **ENTONCES** se muestra el WI enlazado y, por rol, su Task con su estimación y sus imputaciones
- **Y** se muestra el tiempo pendiente de imputar

#### Scenario: Outbox pendiente
- **DADO** un cambio con operaciones pendientes en la outbox
- **CUANDO** se abre la pestaña
- **ENTONCES** se listan esas operaciones sin ofrecer ninguna acción sobre ellas

#### Scenario: Actores por fase
- **DADO** un kickoff sellado por una persona y puertas decididas por otras
- **CUANDO** se abre la pestaña
- **ENTONCES** se muestra quién selló, quién decidió cada puerta y quién imputó cada tiempo

#### Scenario: Cambio archivado
- **DADO** un cambio archivado con vínculos
- **CUANDO** se abre su detalle
- **ENTONCES** la pestaña muestra los mismos datos leídos de su carpeta archivada

#### Scenario: Cambio sin integración
- **DADO** un cambio sin vínculos
- **CUANDO** se abre la pestaña
- **ENTONCES** se muestra un estado vacío que lo explica y no se muestra ningún error

#### Scenario: Ningún secreto en los datos de la pestaña
- **DADO** un PAT centinela guardado
- **CUANDO** se consulta el detalle del cambio
- **ENTONCES** la respuesta no contiene el centinela

### Requirement: Alta de Incremento con Work Item de Origen: Campo ID/URL con Plugin Activo, Mismas Fases que `change start` y Fallos con Salidas Explícitas (RF-WEB-35)

Con un plugin de work items activo, el alta de incremento del dashboard DEBE mostrar un campo de identificador o URL del work item, y el alta DEBE ejecutar las mismas fases que `axiom change start` (fase 1, preparación de la rama con el nombre derivado, creación guardada de la carpeta del cambio y fase 2), sin reimplementar su lógica. Como el alta manual, el alta con work item NO DEBE sellar `kickoff.yaml`: el estilo de ejecución es una decisión de gobierno que el dashboard no pregunta, y el kickoff se sella después al retomar el cambio. Sin plugin activo, el alta DEBE comportarse exactamente como antes y NO DEBE mostrar el campo.

Un rechazo de la fase 1 NO DEBE dejar rama, carpeta del cambio ni `kickoff.yaml`, y DEBE mostrarse con su mensaje y su continuación. Un fallo posterior a la fase 1 DEBE conservar el *staging* y DEBE ofrecer dos acciones: reintentar con el mismo nombre derivado sin repetir la fase 1, o descartar. El descarte NO DEBE ejecutarse nunca de forma automática: solo por una acción explícita de la persona. Una advertencia de la fase 2 con éxito DEBE mostrarse y no detener el flujo.

#### Scenario: Alta con WI crea el cambio con el nombre derivado
- **DADO** un plugin activo y un doble que resuelve el WI
- **CUANDO** la persona da su identificador en el alta
- **ENTONCES** se crea el cambio con el nombre derivado de la fase 1
- **Y** el cambio queda enlazado al WI

#### Scenario: Alta con URL del WI
- **DADO** un plugin activo
- **CUANDO** la persona pega la URL del WI
- **ENTONCES** se obtiene el mismo cambio que con su identificador

#### Scenario: Sin plugin activo el alta no cambia
- **DADO** un proyecto sin plugin de work items activo
- **CUANDO** se abre el alta de incremento
- **ENTONCES** no aparece el campo de work item
- **Y** el alta funciona como antes

#### Scenario: Rechazo de la fase 1 no deja rastro
- **DADO** un WI que la fase 1 rechaza
- **CUANDO** se intenta el alta
- **ENTONCES** se muestra su mensaje y su continuación
- **Y** no existe rama, carpeta del cambio ni `kickoff.yaml`

#### Scenario: Fallo posterior conserva el staging y ofrece reintentar o descartar
- **DADO** un fallo de la preparación de la rama o de la fase 2 tras una fase 1 correcta
- **CUANDO** se muestra el resultado
- **ENTONCES** el *staging* se conserva
- **Y** se ofrecen «Reintentar» (mismo nombre derivado, sin repetir la fase 1) y «Descartar»

#### Scenario: El descarte solo se ejecuta por acción explícita
- **DADO** un fallo posterior a la fase 1
- **CUANDO** la persona no pulsa «Descartar»
- **ENTONCES** el *staging* no se descarta en ningún momento

#### Scenario: Reintentar no repite la fase 1
- **DADO** un fallo posterior a la fase 1
- **CUANDO** la persona pulsa «Reintentar»
- **ENTONCES** se reanuda con el nombre derivado sin volver a ejecutar la fase 1

#### Scenario: Advertencia de la fase 2
- **DADO** una fase 2 que termina con advertencia y éxito
- **CUANDO** se muestra el resultado
- **ENTONCES** el cambio se da por creado, se muestra la advertencia con su reintento y el flujo no se detiene

#### Scenario: Petición con `Origin` ajeno no crea nada
- **DADO** un alta con `Origin` ajeno
- **CUANDO** el servidor la procesa
- **ENTONCES** responde `403` sin ejecutar ninguna fase

### Requirement: Enlace «Abrir en Azure DevOps» con URL Validada (RF-WEB-36)

Cuando un work item enlazado tiene `Link.URL`, el dashboard DEBE mostrar un enlace «Abrir en Azure DevOps» a esa URL. El enlace DEBE mostrarse solo si la URL es absoluta con esquema `http` o `https`, DEBE llevar `target="_blank"` y `rel="noopener noreferrer"`, y NO DEBE mostrarse (ni como texto activo) para cualquier otro esquema (por ejemplo `javascript:` o `data:`) ni para una URL no analizable. La URL se inserta como atributo del DOM, nunca como HTML.

#### Scenario: URL https válida
- **DADO** un WI con `Link.URL` `https` válida
- **CUANDO** se muestra el WI
- **ENTONCES** aparece «Abrir en Azure DevOps» con `target="_blank"` y `rel="noopener noreferrer"`

#### Scenario: Esquema no permitido
- **DADO** un WI cuya `Link.URL` empieza por `javascript:`
- **CUANDO** se muestra el WI
- **ENTONCES** no se pinta ningún enlace activo

#### Scenario: Sin URL
- **DADO** un WI sin `Link.URL`
- **CUANDO** se muestra el WI
- **ENTONCES** no aparece el enlace

### Requirement: Rutas de Integraciones Clasificadas en el Canario de Secretos (RF-WEB-37)

Toda ruta nueva de integraciones y de arranque desde un work item DEBE clasificarse en el canario de secretos, como cubierta (`coveredRoutes`) cuando devuelve datos o como excluida con motivo (`excludedRoutes`) cuando solo muta, de modo que la prueba que exige clasificar toda ruta de `server.go` siga en verde. El canario DEBE ejercitar con un PAT centinela las rutas cubiertas (incluida `GET /api/integrations`) y comprobar que ninguna respuesta lo contiene. Las pruebas de pines de pestañas, de assets y de cadenas del JS DEBEN seguir en verde. Esta exigencia complementa a RF-WEB-26, que no cambia.

#### Scenario: Las rutas nuevas están clasificadas
- **DADO** las rutas registradas en `server.go`
- **CUANDO** se ejecuta la prueba de clasificación del canario
- **ENTONCES** cada ruta de integraciones y de arranque desde WI figura como cubierta o excluida con motivo y la prueba pasa

#### Scenario: Una ruta de integraciones sin clasificar falla
- **DADO** una ruta `/api/integrations...` añadida sin clasificar
- **CUANDO** se ejecuta la prueba de clasificación
- **ENTONCES** la prueba falla

#### Scenario: El canario no encuentra el centinela
- **DADO** un PAT centinela guardado
- **CUANDO** el canario consulta las rutas cubiertas de integraciones
- **ENTONCES** ninguna respuesta contiene el centinela

