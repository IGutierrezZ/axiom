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

- [x] T1 · Huella de contexto multiagente. Un detector genérico recorre los adaptadores y compara usuario frente a workspace: bloques gestionados del fichero de instrucciones, skills, subagentes, comandos y output style. La huella de `axiom doctor` gana el tipo `duplicado`, con aviso de divergencia cuando las dos copias no coinciden. Ruta: delegada (protección de contexto del padre, sesión larga).
- [x] T2 · Preferencia en el hub. `WorkspaceRecord.MCPExclusions` (agente → nombres, `omitempty`), con setter y lectura según el patrón de `hub/rddpref.go`. Ruta: inline (<100 líneas).
- [x] T3 · Aplicación en Claude Code. Fusión de `deniedMcpServers` en `.claude/settings.local.json` de la raíz y de los worktrees del proyecto: alta, baja y conservación de lo ajeno. Se lanza al guardar la preferencia, en `axiom sync` y en `axiom branch ensure`. Ruta: delegada.
- [ ] T4 · Dashboard (implementada en `8eb0c56d`; pendiente de CI y de verla en el navegador). API `GET/POST /api/projects/active/mcp-exclusions` (solo petición local, 409 sin proyecto registrado) y pantalla para marcar servidores y conectores. Candidatos: los MCP de usuario de `~/.claude.json` (solo nombres) y los nombres que se añadan a mano. Ruta: delegada.
- [ ] T5 · Cargado sin uso (implementada en `02e61754`; la parte del dashboard, pendiente de CI). Recuento de uso por servidor en las transcripciones de Claude Code del proyecto; la huella y la pantalla lo muestran como sugerencia de exclusión. Ruta: delegada.
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
- T1 · commit `fbc977ff`. `internal/components/uninstall/context_duplicates.go`: `ContextDuplicates(home, root)` recorre todos los adaptadores y compara cinco tipos (instrucciones con bloques gestionados, skills, subagentes, comandos y output styles); marca `Diverges` cuando las copias difieren. La huella del doctor gana el tipo `duplicado`. Pruebas con Claude Code, Codex y Cursor. Evidencia: `gofmt -l` limpio; `go vet ./internal/components/uninstall/ ./internal/cli/` sin avisos; `go test ./internal/components/uninstall/...` ok; `go test ./internal/cli/ -run "Footprint|Doctor"` ok (repetido por el padre).
- T2 · `internal/hub/mcpexclude.go`: `NormalizeMCPExclusions`, `LookupProjectMCPExclusions` (resuelve también desde subcarpetas y worktrees vía `ProjectForRepo`) y `SetProjectMCPExclusions`, que devuelve las exclusiones previas para retirar lo que sobra. Campo `mcp_exclusions` con `omitempty`. Evidencia: `gofmt -l internal/hub` limpio; `go vet ./internal/hub/` sin avisos; `go test ./internal/hub/` ok.

- T3 · commit `797744cf`.
  - `internal/agents/claude/mcpdeny.go` (`ApplyDeniedMCPServers`): fusiona `deniedMcpServers` en `.claude/settings.local.json`. Adopta lo ya presente, retira solo previas∖actuales, conserva lo ajeno y quita la clave vacía. Con un JSON ilegible da error sin tocar el fichero.
  - `internal/mcpexclusion` (`Apply`, `ApplyCurrent`, `ProjectDirs`, `Report`): aplica en la raíz, en sus worktrees (`gitcli.Git.Worktrees`) y, en multirepo, en los repositorios de rol. Los agentes que no son `claude-code` salen como «no soportado».
  - Disparadores: paso `sync:mcp-exclusions` en `axiom sync` y `applyMCPExclusionsAfterEnsure` en `axiom branch ensure`. Un fallo se avisa y no hace fallar la orden.
  - Decisión: `settings.local.json` no entra en `backupTargets` de sync, porque es personal, aditivo e idempotente.
  - Limitación: al reescribir, las claves de primer nivel quedan ordenadas y se pierden los comentarios JSONC.
  - Evidencia: `gofmt -l` limpio; `go vet` sin avisos; `go test ./internal/mcpexclusion ./internal/agents/claude` ok (repetido por el padre); `go test ./internal/cli -run "Sync|Ensure|MCPExclusion"` ok.
- Incidencia 2026-11-25 · Al retirar el global, Engram quedó fuera de las sesiones de Axiom. Su servidor de usuario se había ido, y el de `.mcp.json` está pendiente de aprobación. Se restauró como MCP de usuario por decisión del usuario (`claude mcp add --scope user engram -- engram mcp --tools=agent`); `claude mcp list` lo da conectado.

- T4 · commit `8eb0c56d`.
  - API `GET|POST /api/projects/active/mcp-exclusions` (`service_mcp_exclusions.go`). Responde 403 a una petición no local, 400 a datos inválidos, 409 si el proyecto no está registrado y 405 a otros métodos.
  - Candidatos: `claude.UserMCPServerNames`, que lee solo las claves de `mcpServers`.
  - Tarjeta «Servidores MCP excluidos en este proyecto» en la pantalla Ecosistema, tras «Preferencias de worktree». Módulo puro `assets/js/core/mcpexclusions.js`.
  - Evidencia: `gofmt -l` limpio; `go vet ./internal/dashboard/ ./internal/mcpexclusion/ ./internal/agents/claude/` sin avisos (repetido por el padre); `go test ./internal/mcpexclusion/ ./internal/agents/claude/` ok; el módulo JS se probó con `node` directamente.
  - **Pendiente:** `go test ./internal/dashboard/...` en CI (Cortex) y la verificación visual.
- Prueba de escritorio 2026-11-25 (sesión nueva y captura del usuario):
  - `work-copilot` sí queda bloqueado.
  - Microsoft 365 y la knowledge base siguen cargándose. La app entrega los conectores con su id como nombre de servidor (herramientas `mcp__b67fe70e-7904-498b-917a-36af6bb9c0f2__…`), y `claude.ai <Nombre>` no casa.
  - Se añadieron a mano las entradas por id (`b67fe70e-…`, `9d5c639e-…`) en el checkout principal. Falta comprobarlo en otra sesión nueva.
  - Los interruptores de «+ → Conectores» solo valen para la sesión en curso.

- T5 · commit `02e61754`.
  - `internal/mcpexclusion/usage.go`: `EncodeProjectDir` (cada carácter fuera de `[A-Za-z0-9]` pasa a `-`), `ServerSegment`, `ScanUsage`, `Analyze` y `UnusedCandidates`.
  - El escáner lee solo los nombres `mcp__<servidor>__*` de los `*.jsonl` de los últimos 30 días, en la raíz, los worktrees y los roles. «Sin datos» significa que no hay ninguna transcripción.
  - La huella del doctor gana el tipo `sin uso`, con una nota de que es una sugerencia. El DTO del dashboard gana `usage`/`usageData`, y la pantalla muestra la insignia «sin uso en 30 días».
  - No se pueden casar los conectores que el escritorio nombra por id ni los servidores de plugins: pueden salir como «sin uso» aunque se usen.
  - Evidencia: `gofmt -l` limpio; `go vet ./internal/mcpexclusion/ ./internal/cli/ ./internal/dashboard/` ok; `go test ./internal/mcpexclusion/` ok (repetido por el padre); `go test ./internal/cli/ -run "Footprint|Doctor"` ok.
  - **Pendiente:** `go test ./internal/dashboard/...` en CI.

## Siguiente paso

T6. Según el resultado de la prueba por id, ajustar T3 y T4: escribir los dos nombres de cada conector, o avisar en la pantalla de que en escritorio los conectores solo se apagan por sesión.

## Spec viva

### Capacidad: multi-project-hub

```openspec
## ADDED Requirements

### Requirement: Exclusiones Personales de Servidores MCP por Proyecto en el Hub (REQ-30.33)

Además de las preferencias de REQ-25.29, cada proyecto registrado en `~/.axiom/workspaces.json` PUEDE almacenar **exclusiones personales de servidores MCP**: un mapa de agente a lista de nombres de servidor o de conector (campo `mcp_exclusions`, omitido cuando está vacío). Las exclusiones son personales y NO DEBEN escribirse nunca en `axiom.yaml` ni en ningún fichero versionado. Los nombres DEBEN normalizarse (sin espacios sobrantes, sin vacíos ni duplicados) y la consulta DEBE resolverse también desde una subcarpeta o un worktree del proyecto. Guardarlas DEBE devolver las exclusiones previas para que quien las aplica retire solo lo que sobra. Registrar de nuevo un proyecto NO DEBE borrarlas; desregistrarlo SÍ DEBE eliminarlas junto con su entrada. No DEBEN cambiar la versión del esquema (`"1.0"`): un binario anterior las ignora.

#### Scenario: Persistencia sin tocar `axiom.yaml`
- **DADO** un proyecto registrado
- **CUANDO** se guardan las exclusiones `work-copilot` y `claude.ai Microsoft 365` para `claude-code`
- **ENTONCES** `workspaces.json` las contiene en la entrada de ese proyecto
- **Y** `axiom.yaml` no se modifica

#### Scenario: Se resuelven desde un worktree
- **DADO** un proyecto registrado con exclusiones guardadas
- **CUANDO** se consultan desde uno de sus worktrees
- **ENTONCES** se devuelven las exclusiones del proyecto

### Requirement: Las Exclusiones MCP se Aplican como `deniedMcpServers` en `.claude/settings.local.json` (REQ-30.34)

Axiom DEBE aplicar las exclusiones de `claude-code` fusionando `deniedMcpServers` (`[{"serverName": "<nombre>"}]`) en `.claude/settings.local.json` de la raíz del proyecto, de cada uno de sus worktrees y, en `multirepo`, de los repositorios de rol. La aplicación DEBE lanzarse al guardar la preferencia, en `axiom sync` y en `axiom branch ensure`. DEBE conservar las claves y las entradas ajenas, adoptar las entradas ya presentes cuyo nombre coincida con la preferencia y, al quitar una exclusión, retirar solo las de la preferencia anterior que ya no están en la nueva, y eliminar la clave si queda vacía. Con un JSON ilegible DEBE fallar sin tocar el fichero. Un fallo al aplicar en `sync` o en `ensure` DEBE avisarse y NO DEBE hacer fallar la orden. Los agentes distintos de `claude-code` DEBEN informarse como no soportados. Un conector de claude.ai se identifica en la CLI por `claude.ai <Nombre>`. La aplicación de escritorio de Claude entrega sus conectores en proceso con su id como nombre: esa forma no queda cubierta por `claude.ai <Nombre>` y su bloqueo por id queda pendiente de verificar.

#### Scenario: Alta de una exclusión conservando lo ajeno
- **DADO** un `.claude/settings.local.json` con otras claves y una entrada ajena en `deniedMcpServers`
- **CUANDO** se aplica la exclusión `work-copilot`
- **ENTONCES** `deniedMcpServers` contiene la entrada ajena y `work-copilot`
- **Y** las demás claves permanecen

#### Scenario: Baja de una exclusión
- **DADO** una preferencia anterior con `work-copilot` y `claude.ai Microsoft 365`, aplicada
- **CUANDO** la nueva preferencia solo contiene `work-copilot`
- **ENTONCES** se retira `claude.ai Microsoft 365` y se conservan `work-copilot` y las entradas ajenas

#### Scenario: Se aplica en los worktrees de `ensure`
- **DADO** un proyecto con exclusiones guardadas
- **CUANDO** `axiom branch ensure` prepara un worktree
- **ENTONCES** el `.claude/settings.local.json` del worktree contiene las exclusiones

#### Scenario: Agente no soportado
- **DADO** exclusiones guardadas para un agente distinto de `claude-code`
- **CUANDO** se aplican
- **ENTONCES** el informe las marca como no soportadas y no se escribe ningún fichero
```

### Capacidad: local-web-dashboard

```openspec
## ADDED Requirements

### Requirement: El Dashboard Edita las Exclusiones MCP del Proyecto Activo (REQ-25.55)

El dashboard DEBE ofrecer en la pantalla Ecosistema la tarjeta «Servidores MCP excluidos en este proyecto» y la API `GET` y `POST /api/projects/active/mcp-exclusions`. La API DEBE responder 403 a una petición que no sea local, 400 a datos inválidos, 409 si el proyecto activo no está registrado y 405 a otros métodos. Los candidatos DEBEN ser los servidores de `mcpServers` de `~/.claude.json` (solo sus nombres, nunca sus valores) y los nombres que la persona añada a mano; los servidores cargados sin uso en 30 días DEBEN marcarse como sugerencia. Guardar DEBE aplicar la preferencia según REQ-30.34. Los agentes sin la capacidad DEBEN mostrarse como no soportados.

#### Scenario: Guardar aplica la exclusión
- **DADO** un proyecto registrado y abierto en el dashboard
- **CUANDO** se marca `work-copilot` y se guarda
- **ENTONCES** el hub guarda la exclusión
- **Y** el `.claude/settings.local.json` del proyecto contiene `work-copilot` en `deniedMcpServers`

#### Scenario: Proyecto no registrado
- **DADO** un proyecto abierto que no está en `workspaces.json`
- **CUANDO** se guarda una exclusión
- **ENTONCES** la API responde 409 y no se escribe nada

#### Scenario: Petición no local
- **DADO** una petición que no procede del propio equipo
- **CUANDO** llama a `/api/projects/active/mcp-exclusions`
- **ENTONCES** la API responde 403
```
