# ODD: Agentes del proyecto en la UI: disponibles, instalados e instalación con un clic (agentes-ui)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/agentes-ui.md`. Espejo Engram: `odd/agentes-ui/tasks`.
> Rama: `inc/agentes-ui` · worktree `C:\repos\axiom-wt\agentes-ui\axiom`. Cambio C-06-g del plan maestro (`docs/plan/backlog-2026-11/06-ui-human-loop.md`), ola 7.

## Objetivo

Ver en el dashboard qué agentes admite Axiom, cuáles detecta en el equipo y cuáles están instalados en el proyecto abierto, e instalarlos en el proyecto sin salir de la UI.

## Problema y porqué

- El dashboard no tiene vista ni endpoint de agentes; solo la pantalla Ecosistema lanza `axiom sync` en proceso.
- `axiom sync` nunca escribe `installed_agents`: con la lista sin registrar descubre los agentes por la carpeta de usuario, pero no los guarda, y `state.ProjectView` no hereda los del estado global a propósito. Por eso `axiom doctor` avisa en `state:json` aunque el sync funcione. Solo `axiom install` registra los agentes del proyecto.
- Petición del usuario (2026-11-20): «Estaría bien tener en la UI qué agentes tienes disponibles, cuáles están instalados y poder instalarlos fácilmente desde la UI.»

## Decisiones

- **Instalación (plan).** Reutiliza el núcleo de `axiom install --scope workspace --agent <id>` en proceso (`cli.InstallInWorkspace`), el mismo que ya usa el alta de proyectos (`installAdapters`). Con `--agent`, install fusiona el agente en `installed_agents` sin pisar el resto (`persistInstallState` → `mergeExplicitAgentInstallState`). Sin `--component`, instala el conjunto por defecto que dejó C-08-f (`model.DefaultComponents()`).
- **Tres marcas (plan).** Admitido = `catalog.IsSelectableAgent`; detectado = `agents.DiscoverInstalled` con el registro por defecto y el HOME del proceso (comprobación de ficheros, sin subprocesos); instalado = `installed_agents` del estado del proyecto activo, solo cuando el proyecto lo ha registrado (`Store.AgentsRecorded`). Si no lo ha registrado, la respuesta lo dice (`recorded: false`) y ningún agente sale como instalado.
- **Sin modo workspace (plan).** Los agentes que no admiten el ámbito workspace (`cli.AgentSupportsWorkspace`: Trae, Antigravity, Pi, Copilot CLI) y los no seleccionables no ofrecen el botón y muestran el motivo (`cli.WorkspaceUnsupportedReason`). El servidor rechaza su instalación con 400 aunque se pida a mano.
- **Aviso previo (plan).** La UI confirma antes de instalar que se escribe en el repositorio y, según el agente, en la carpeta de usuario.
- **Un agente por petición (análisis).** El botón es por fila; el endpoint acepta un único `agent`. Las instalaciones se serializan en el servicio con un cerrojo.
- **Proyecto activo (análisis).** La raíz del servicio nunca está vacía (cae en `.`), así que «sin proyecto activo» es una raíz sin marcador de workspace propio (`workspace.LocateAt` con origen `start`): se rechaza con 409 para no escribir `.axiom/` en una carpeta cualquiera. `GET /api/agents` lo indica con `project`.
- **Fuera de alcance.** Desinstalar agentes desde la UI y la instalación en ámbito global.

## Alcance autorizado

`internal/dashboard` (`server.go`, `service_agents.go` nuevo, `types.go`, `assets/index.html`, `assets/js/screens/ecosystem.js`, `assets/js/core/ecosystem.js`, `assets/style.css` si hace falta, y sus tests), lectura de `internal/agents`, `internal/state`, `internal/catalog` e `internal/cli/workspace_agents.go`, `docs/` vivas del dashboard y `openspec/specs/local-web-dashboard/` vía `## Spec viva`.

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y no hay elección del usuario. Comprobaciones: `go vet` y `go test ./internal/dashboard` (con `-run` cuando baste), sin la batería completa ni binarios sueltos de `axiom`.

## Entrega

`single-pr`: una PR con un commit por tarea. Etiqueta `type:feature`. Previsión: unas 600 líneas.

## Tareas

- [x] T1 · Backend: `GET /api/agents` (agentes con admitido, detectado, instalado, workspace y motivo, más si el proyecto registra su lista) y `POST /api/agents/install` (un agente; valida antes de escribir; reutiliza `installAdapters`), tipos y tests.
- [x] T2 · UI: tarjeta «Agentes» en Ecosistema con las tres marcas, botón «Instalar» con confirmación del aviso de escritura, motivo para los no instalables, resultado en «Última operación» y refresco del Doctor; tests de assets.
- [ ] T3 · `## Spec viva` integrada con `axiom archive integrate --odd agentes-ui` y documentación viva del dashboard.

Ruta: todas en línea (unidades de menos de unas 300 líneas con el mapa ya hecho; ningún disparador de delegación).

## Criterio de terminado

- La UI lista cada agente con tres marcas: admitido, detectado en el equipo e instalado en el proyecto.
- Instalar desde la UI deja el agente en `.axiom/state.json` del proyecto, y tras ello `axiom doctor` deja de avisar en `state:json`.
- Un agente sin modo workspace no ofrece el botón y dice por qué.
- `go vet` y `go test ./internal/dashboard` en verde; la batería completa, en la CI de la PR.

## Spec viva

Añade a `local-web-dashboard` la consulta de los agentes del proyecto y su instalación en ámbito workspace desde el dashboard.

### Capacidad: local-web-dashboard

```openspec
## ADDED Requirements

### Requirement: Endpoint de Agentes del Proyecto Activo (prov:1:1)

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

### Requirement: Instalar un Agente en el Proyecto Activo desde el Dashboard (prov:1:2)

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
```

## Progreso y evidencia

- T1: `go vet ./internal/dashboard` limpio y `go test ./internal/dashboard -count=1` en verde (78 s) tras clasificar las rutas nuevas en el canario de secretos (`/api/agents` cubierta, `/api/agents/install` excluida por ser POST).
- T2: `go test ./internal/dashboard -count=1` en verde (69 s) con la prueba nueva `TestEcosystemAgentsBlock`; `node --check` de `screens/ecosystem.js` y `core/ecosystem.js` limpio. No se ha visto en el navegador: arrancar el dashboard exige compilar `axiom`.

## Siguiente paso

T3.
