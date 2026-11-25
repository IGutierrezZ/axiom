# contexto-por-proyecto

Rama: `feat/contexto-por-proyecto`. Worktree: `C:\repos\axiom-wt\contexto-por-proyecto\axiom`. PR: IGutierrezZ/axiom#302.

## Objetivo

Que `axiom doctor` detecte, para todos los agentes, el contexto de Axiom duplicado entre la instalación de usuario y la del workspace, para que cada sesión no cargue dos veces lo mismo.

La exclusión de servidores MCP, conectores y plugins por proyecto se aparcó el 2026-11-25 (ver «Trabajo futuro»).

## Problema y porqué

- Claude Code suma la configuración de usuario y la de proyecto; una instalación de workspace no sustituye a la global. En este equipo eran unos 35k tokens por turno de instrucciones del orquestador, la mitad duplicadas y con la copia global desfasada (aún describía RDD).
- La huella de `axiom doctor` (`collectMachineFootprint`, `internal/cli/doctor_footprint.go`) no lo detectaba: solo cubre GGA, `~/.agents/skills`, OpenCode, Kilo, Kiro, Codex, CodeGraph y el PATH.

## Alcance y restricciones

- Cubre todos los adaptadores y es informativa: no cambia el estado del doctor.
- Un fichero sin marcas de Axiom nunca se informa.
- TDD: no configurado (no hay clave en `axiom.yaml` ni elección en la sesión). Las comprobaciones funcionales son `go test` por paquete.
- Sin binarios sueltos, por Cortex.
- No cambia ninguna spec viva: ninguna cubre la huella de `axiom doctor`.

## Tareas

- [x] T1 · Huella de contexto multiagente. Un detector genérico recorre los adaptadores y compara usuario frente a workspace: bloques gestionados del fichero de instrucciones, skills, subagentes, comandos y output style. La huella de `axiom doctor` gana el tipo `duplicado`, con aviso de divergencia cuando las dos copias no coinciden. Ruta: delegada (protección de contexto del padre, sesión larga).
- Aparcada · T2 · Preferencia de exclusiones MCP en el hub (`1336bf44`).
- Aparcada · T3 · Aplicación como `deniedMcpServers` en `.claude/settings.local.json` (`797744cf`).
- Aparcada · T4 · Pantalla y API del dashboard (`8eb0c56d`).
- Aparcada · T5 · Servidores MCP cargados sin uso (`02e61754`).
- [x] T6 · Documentación de uso y cierre. Ruta: inline.

## Criterio de aceptación

- Con una instalación global y otra de workspace del mismo agente, `axiom doctor` lista cada duplicado con sus dos rutas. Sin duplicados, no añade nada.

## Comprobaciones

- `go test` de `internal/components/uninstall` y de `internal/cli` (filtrado a footprint y doctor).
- `go vet ./...` (compila también las pruebas sin enlazar ejecutables).

## Entrega

`single-pr`. Se recomienda fusionar con merge commit, sin squash.

## Progreso y evidencia

- 2026-11-25 · Retirada la instalación global de Axiom en Claude Code (copia `20261125100156.937750600`).
- T1 · commit `fbc977ff`. `internal/components/uninstall/context_duplicates.go`: `ContextDuplicates(home, root)` recorre todos los adaptadores y compara cinco tipos; marca `Diverges` cuando las copias difieren. La huella del doctor gana el tipo `duplicado`. Pruebas con Claude Code, Codex y Cursor. Evidencia: `gofmt -l` limpio; `go vet ./internal/components/uninstall/ ./internal/cli/` sin avisos; `go test ./internal/components/uninstall/...` ok; `go test ./internal/cli/ -run "Footprint|Doctor"` ok (repetido por el padre).
- Incidencia 2026-11-25 · Al retirar el global, Engram quedó fuera de las sesiones de Axiom. Su servidor de usuario se había ido, y el de `.mcp.json` está pendiente de aprobación. Se restauró como MCP de usuario por decisión del usuario (`claude mcp add --scope user engram -- engram mcp --tools=agent`).
- T2 a T5 se implementaron (commits `1336bf44`, `797744cf`, `8eb0c56d` y `02e61754`) y se retiraron en el commit de retirada que sigue a `0dd7ce92`.
  - Se devolvió el código y las specs vivas (`multi-project-hub`, `local-web-dashboard` y `openspec/INDEX.md`) al estado tras T1.
  - La PR ya estaba publicada, así que no se reescribió la historia.
  - Motivo: decisión del usuario. Solo funcionaba en la CLI de Claude Code, no en la app de escritorio, y faltaba resolverlo en el resto de agentes.
- T6 · `docs/usage.md` («Contexto duplicado entre usuario y workspace» y la fila de la huella) y `docs/manual-de-inicio.md`. Sin `## Spec viva`.
- Evidencia tras la retirada:
  - `git diff --name-status fbc977ff -- . ':!odd' ':!docs'` vacío: el código y las specs son idénticos a T1.
  - `go vet ./...` ok.
  - `go test ./internal/hub/ ./internal/components/uninstall/...` ok.
  - `go test ./internal/cli/ -run "Footprint|Doctor"` ok.

## Trabajo futuro: exclusión de conectores y MCP por proyecto

Aparcado hasta que las herramientas ofrezcan un mecanismo estable en todos los clientes y agentes. Las `instructions` de cada servidor MCP y de cada conector ocupan contexto en todas las sesiones, aunque sus herramientas estén diferidas. Lo aprendido el 2026-11-25:

- **Claude Code, CLI:**
  - `deniedMcpServers` en `.claude/settings.local.json` (ignorado por git) excluye servidores solo en ese proyecto.
  - Formato: `[{"serverName": "<nombre>"}]`.
  - Las listas de todos los ficheros de settings se fusionan.
  - Un conector de claude.ai se nombra `claude.ai <Nombre>`. Verificado con `claude mcp list`.
  - Los worktrees no heredan `settings.local.json`.
- **Claude Code, app de escritorio:**
  - `deniedMcpServers` bloquea los servidores MCP propios (verificado con `work-copilot`).
  - No bloquea los conectores que entrega la app: llegan con su id como nombre (por ejemplo `mcp__b67fe70e-7904-498b-917a-36af6bb9c0f2__…`).
  - `disableClaudeAiConnectors` tampoco les afecta, según la documentación.
  - Solo se apagan por sesión, en «+ → Conectores».
  - Quedó sin comprobar si `deniedMcpServers` con el id del conector los bloquea.
- **Descartado:** `disabledMcpServers` (lo que escribe `/mcp`) vive en `~/.claude.json`, que guarda la sesión OAuth.
- **Plugins:** `enabledPlugins: {"x": false}` en los settings del proyecto gana al de usuario, según la documentación (sin probar).
- **Otros agentes:** sin investigar.
- **Diseño que se probó y se retiró:**
  - La preferencia personal se guardaba en el hub (`mcp_exclusions`) y se aplicaba en la raíz, los worktrees y los repositorios de rol.
  - El dashboard tenía una tarjeta para elegir las exclusiones.
  - Un escáner de transcripciones detectaba los servidores sin uso en 30 días, leyendo solo los nombres de herramienta.
  - El código está en los commits citados arriba, por si se retoma.

## Siguiente paso

CI de la PR. Después, fusionar (decisión del usuario).
