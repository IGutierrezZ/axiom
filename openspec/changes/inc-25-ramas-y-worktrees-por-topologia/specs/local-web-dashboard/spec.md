# Delta para Servidor HTTP Local Embebido y Dashboard Web (INC-25)

> **Incremento:** `inc-25-ramas-y-worktrees-por-topologia` · **Tipo:** delta sobre la capacidad viva `local-web-dashboard`
> **Idioma:** Español (castellano peninsular)

Cambios de este delta:

- REQ-1.1: `GET /api/workspace` retira `monorepo-decoupled` del conjunto de topologías y expone `git:` y `repositories:` en solo lectura.
- REQ-25.32 (nuevo): crear un cambio desde el dashboard (y con `axiom change create`) ejecuta `axiom branch ensure` y después escribe `proposal.md` con el escritor guardado.
- REQ-25.33 (nuevo): `git:` y `repositories:` son de solo lectura; solo se editan las preferencias personales del hub.
- REQ-25.34 (nuevo): los listados y detalles agregan los cambios que viven en los worktrees registrados, muestran los cambios pendientes de archivar y los avisan en la cabecera `X-Axiom-Notes` de `GET /api/increments`; la TUI solo lista el mismo conjunto.

> El registro con preferencias personales (`POST /api/projects/add`, `POST /api/projects/init` y el modal de registro) pertenece a la capacidad `multi-project-hub` (REQ-3.1 y REQ-3.2) y se modifica en su delta. La spec de TUI (`tui-ui-parity`) no figura entre las capacidades modificadas de la propuesta: la paridad de la TUI se cubre aquí como escenario de REQ-25.34 (supuesto S-14).

## MODIFIED Requirements

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

## ADDED Requirements

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
