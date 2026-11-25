# contexto-por-proyecto

Rama: `feat/contexto-por-proyecto`. Worktree: `C:\repos\axiom-wt\contexto-por-proyecto\axiom`.

## Objetivo

Que cada sesión de agente cargue solo el contexto del proyecto en el que trabaja:

1. `axiom doctor` detecta, para todos los agentes, el contexto de Axiom duplicado entre la instalación de usuario y la del workspace, y los servidores MCP cargados que no se usan.
2. La persona elige en el dashboard, por proyecto, qué servidores MCP y conectores excluir, y Axiom lo aplica a la configuración local del agente en ese proyecto y en sus worktrees.

## Problema y porqué

- Claude Code suma la configuración de usuario y la de proyecto; una instalación de workspace no sustituye a la global. En este equipo eran unos 35k tokens por turno de instrucciones del orquestador, la mitad duplicadas y con la copia global desfasada (aún describía RDD). La huella de `axiom doctor` (`collectMachineFootprint`, `internal/cli/doctor_footprint.go`) no lo detectaba: solo cubre GGA, `~/.agents/skills`, OpenCode, Kilo, Kiro, Codex, CodeGraph y el PATH.
- Los conectores de claude.ai y los MCP de usuario se cargan en todos los proyectos: `claude mcp list` en este repositorio da 28 servidores. Aunque sus herramientas estén diferidas, las `instructions` de cada servidor ocupan contexto en cada sesión.

## Hechos verificados (2026-11-25)

- `deniedMcpServers` es válido en cualquier fichero de settings, y las listas de todos los ficheros se fusionan. Formato: `[{"serverName": "<nombre>"}]`. Un conector se identifica por su nombre visible, `claude.ai <Nombre>`. Fuentes: code.claude.com/docs/en/settings-reference.md y code.claude.com/docs/en/mcp.md.
- Prueba en el checkout principal: con `deniedMcpServers` = `work-copilot`, `claude.ai Microsoft 365` y `claude.ai hiberus-knowledgebase` en `.claude/settings.local.json`, `claude mcp list` deja de listar los tres. Siguen `claude.ai hiberus-iametrics`, `claude.ai Claude Docs` y `claude.ai Claude Data`.
- La app de escritorio entrega sus conectores en proceso. `disableClaudeAiConnectors` no los afecta, y la documentación no dice si `deniedMcpServers` sí. **Pendiente:** comprobarlo en una sesión nueva de escritorio.
- Git ignora `.claude/settings.local.json` y hoy Axiom no lo lee ni lo escribe. Los worktrees no lo heredan.
- `disabledMcpServers` (lo que escribe `/mcp`) vive por proyecto en `~/.claude.json`, sin formato documentado. Se descarta escribirlo: ese fichero guarda la sesión OAuth.
- Tras retirar la instalación global, `engram` llega por el `.mcp.json` del proyecto y la CLI lo muestra «pendiente de aprobación». Aprobarlo es decisión de la persona, no de Axiom.

## Alcance y restricciones

- La preferencia es personal. Vive en el hub (`WorkspaceRecord` de `workspaces.json`) y nunca en `axiom.yaml` ni en ficheros versionados.
- Se aplica en `.claude/settings.local.json`, en la raíz del proyecto y en cada worktree que prepara `axiom branch ensure`. Se conservan las claves y las entradas ajenas. Al quitar una exclusión, Axiom retira solo las entradas de la preferencia anterior que ya no están en la nueva.
- Las entradas puestas a mano en la prueba del checkout principal se adoptan cuando la preferencia guarda los mismos nombres.
- Exclusiones v1: solo Claude Code. Los demás agentes no cargan conectores de claude.ai. La capacidad del adaptador es opcional, y la UI marca «no soportado» en los agentes que no la tienen.
- Huella: cubre todos los adaptadores y es informativa (no cambia el estado del doctor).
- «Cargado sin uso»: solo Claude Code. Se cuentan las llamadas `mcp__<servidor>__*` en las transcripciones locales del proyecto de los últimos 30 días. Solo se leen nombres de herramienta, nunca contenido.
- TDD: no configurado (no hay clave en `axiom.yaml` ni elección en la sesión). Las comprobaciones funcionales son `go test` por paquete.
- Sin binarios sueltos, por Cortex. `go test ./internal/dashboard` fue marcado por Cortex el 2026-11-20: la lógica va en paquetes fuera de `dashboard`, y las pruebas de `dashboard` se validan en CI.
- Fuera de alcance, para proponer aparte: el fallo de `uninstall --agent` que también retira GGA (`partialUninstall` usa `allManagedComponents`), y el falso positivo de `axiom branch guard` al escribir ficheros ignorados por git en `main`.

## Tareas

- [ ] T1 · Huella de contexto multiagente. Un detector genérico recorre los adaptadores y compara usuario frente a workspace: bloques gestionados del fichero de instrucciones, skills, subagentes, comandos y output style. La huella de `axiom doctor` gana el tipo `duplicado`, con aviso de divergencia cuando las dos copias no coinciden. Ruta: delegada (protección de contexto del padre, sesión larga).
- [ ] T2 · Preferencia en el hub. `WorkspaceRecord.MCPExclusions` (agente → nombres, `omitempty`), con setter y lectura según el patrón de `hub/rddpref.go`. Ruta: inline (<100 líneas).
- [ ] T3 · Aplicación en Claude Code. Fusión de `deniedMcpServers` en `.claude/settings.local.json` de la raíz y de los worktrees del proyecto: alta, baja y conservación de lo ajeno. Se lanza al guardar la preferencia, en `axiom sync` y en `axiom branch ensure`. Ruta: delegada.
- [ ] T4 · Dashboard. API `GET/POST /api/projects/active/mcp-exclusions` (solo petición local, 409 sin proyecto registrado) y pantalla para marcar servidores y conectores. Candidatos: los MCP de usuario de `~/.claude.json` (solo nombres) y los nombres que se añadan a mano. Ruta: delegada.
- [ ] T5 · Cargado sin uso. Recuento de uso por servidor en las transcripciones de Claude Code del proyecto; la huella y la pantalla lo muestran como sugerencia de exclusión. Ruta: delegada.
- [ ] T6 · Documentación de uso, revisión de la spec viva (`## Spec viva` si `openspec/specs` cubre el doctor o el dashboard) y cierre: `go test` de los paquetes tocados y `axiom doctor`. Ruta: inline.

## Criterio de aceptación

- Con una instalación global y otra de workspace del mismo agente, `axiom doctor` lista cada duplicado con sus dos rutas. Sin duplicados, no añade nada.
- Marcar Microsoft 365, la knowledge base y `work-copilot` para `axiom` en el dashboard deja esas tres entradas en `deniedMcpServers` del `.claude/settings.local.json` de la raíz y de cada worktree. Desmarcarlas las retira sin tocar `enabledMcpjsonServers` ni entradas ajenas. En otro proyecto no cambia nada.
- `claude mcp list` en el proyecto no muestra los servidores excluidos.

## Comprobaciones

- `go test` de `internal/hub`, del paquete del detector, de `internal/components/...` tocados y de `internal/cli` (troceado).
- Pruebas Node de los módulos JS puros del dashboard.
- `internal/dashboard`: CI.

## Entrega

`single-pr`, con un commit por tarea.

## Progreso y evidencia

- 2026-11-25 · Retirada la instalación global de Axiom en Claude Code (copia `20261125100156.937750600`). Prueba manual de la exclusión en el checkout principal (ver «Hechos verificados»).

## Siguiente paso

T1, y en paralelo comprobar en una sesión nueva de escritorio si los conectores excluidos desaparecen.
