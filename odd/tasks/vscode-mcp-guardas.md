# ODD: MCP de proyecto en `.mcp.json` y guarda de rama con hooks en VS Code Copilot (vscode-mcp-guardas)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/vscode-mcp-guardas.md`. Espejo Engram: `odd/vscode-mcp-guardas/tasks`.
> Rama: `inc/vscode-mcp-guardas` · worktree `C:\repos\axiom-wt\vscode-mcp-guardas\axiom`.

## Objetivo

Dos huecos del adaptador de VS Code Copilot en el ámbito workspace:

- **A.** Declarar los servidores MCP de proyecto en su destino nativo, `<raíz>/.mcp.json` con `mcpServers`, y retirar las entradas que Axiom gestionaba en `.vscode/mcp.json`.
- **B.** Dar a Copilot la guarda de rama automática que ya tienen Claude Code (hooks) y OpenCode (plugin), con los hooks de agente de VS Code en `.github/hooks/`. Evaluar el preflight SDD.

## Problema y porqué

El equipo usa VS Code Copilot a diario. Hoy Axiom escribe `.vscode/mcp.json` (`servers`), que VS Code asocia al arnés Local, deprecado y que se retira en febrero de 2027. El arnés de Copilot no lee ese fichero. Además, en VS Code ninguna guarda automática protege los documentos de Axiom de escribirse en la rama equivocada.

## Evidencia de la documentación oficial (consultada el 26/11/2026; páginas fechadas el 25/11/2026)

MCP:

- https://code.visualstudio.com/docs/agents/reference/mcp-configuration: «The Local harness is deprecated and will be removed in February 2027». Para configuraciones nuevas, los destinos nativos son `.mcp.json` en la raíz del workspace y `~/.copilot/mcp-config.json`, con `mcpServers`, y no `.vscode/mcp.json`. El arnés no lee directamente `.vscode/mcp.json`: solo lo reenvía el editor, sin migrarlo, y descarta `envFile`, `dev` y las variables `${input:…}`.
- El formato portable usa `${NAME}` en lugar de `${env:NAME}`, y no admite `inputs`, `envFile`, `dev` ni `sandboxEnabled`.
- https://code.visualstudio.com/docs/agents/run/manage-mcp-servers: «Workspace `.vscode/mcp.json` and `.mcp.json` configurations inherit Workspace Trust». En modo restringido no arrancan.
- La documentación no dice qué pasa si el mismo nombre de servidor aparece en los dos ficheros. Por eso se retiran las entradas de Axiom de `.vscode/mcp.json` en lugar de mantenerlas duplicadas.

Hooks:

- https://code.visualstudio.com/docs/agent-customization/hooks y https://docs.github.com/en/copilot/reference/hooks-reference: el arnés de Copilot (VS Code, app y CLI) lee `.github/hooks/*.json` con `{"version":1,"hooks":{"preToolUse":[{"type":"command","command":"…","timeoutSec":10}]}}`. Las entradas admiten `bash`/`powershell`/`command`, `cwd`, `env`, `timeoutSec` y `matcher`. En `preToolUse` y `postToolUse`, `matcher` es una regex anclada sobre `toolName`.
- Carga camelCase de `preToolUse`: `sessionId`, `timestamp`, `cwd`, `toolName`, `toolArgs` (JSON codificado como cadena). `postToolUse` añade `toolResult`. **No incluye el nombre del evento**. La variante PascalCase (`PreToolUse`) manda campos snake_case y nombres de herramienta de Claude.
- Nombres de herramienta del arnés: `create` (→ `Write`), `edit`, `str_replace_editor` y `apply_patch` (→ `Edit`), `task` (→ `Agent`), `bash`, `powershell`, `view`, etc. La documentación **no fija los campos de los argumentos** de esas herramientas.
- Salida: `preToolUse` deniega con `{"permissionDecision":"deny","permissionDecisionReason":"…"}` en `stdout` (`hookSpecificOutput` no existe). `postToolUse` añade contexto para el modelo con `{"additionalContext":"…"}`. Vacío o `{}` no decide nada.
- Códigos de salida: `0` lee `stdout`. `2` deniega en `preToolUse` y solo es un aviso en `postToolUse`. Cualquier otro código distinto de cero **deniega en `preToolUse` (fail-closed)** y continúa en el resto. Un tiempo agotado es fail-open siempre.
- Los hooks de repositorio exigen una carpeta de confianza. `chat.useHooks: false` (ajuste de usuario) los apaga. Es una función en preview.
- El arnés de Copilot **también lee `.claude/settings.json` y `.claude/settings.local.json` del repositorio**. Ejecuta juntos los hooks de todas las fuentes, sin deduplicar.
- `subagentStart` no puede bloquear. Su carga (`transcriptPath`, `agentName`…) no trae el prompt. `preToolUse` y `postToolUse` no traen `transcriptPath`. El agente `general-purpose` no emite `subagentStart`.

## Decisiones

- **A. Destino `.mcp.json` compartido con Claude Code.** VS Code escribe en `<raíz>/.mcp.json` con `mcpServers` y la misma fusión que Claude Code (`MergeJSONObjects`). Con los dos clientes instalados, cada servidor debe tener una única forma válida para ambos. Si no, cada `sync` reescribiría la entrada del otro y dejaría de ser idempotente. Detalle en T1.
- **A. Forma única de cada entrada.** VS Code escribe la misma orden que Claude Code (Engram: `engram mcp --tools=agent`; Context7: `npx` con el paquete fijado, no el remoto por http de antes) y añade `type: "stdio"` y `tools: ["*"]`. Las dos claves aparecen en todos los ejemplos del arnés de Copilot, y la migración de VS Code añade `tools: ["*"]`. Que Claude Code las acepta se comprobó con `claude mcp get` sobre un `.mcp.json` de prueba: la entrada sale como servidor stdio. La fusión profunda de Claude (`command`/`args`) conserva esas claves, así que el fichero converge y `sync` es idempotente en cualquier orden.
- **A. Orden de Engram.** Se mantiene `stableEngramCommandForMergedConfig`: si el fichero ya trae una orden de Engram, se conserva. Si VS Code llega primero, escribe la ruta absoluta (no es un agente «estándar», porque una aplicación gráfica puede no ver el `PATH`). Si llega primero Claude Code, escribe `engram`. Las dos órdenes funcionan para los dos clientes en esa máquina. La lectura de la orden de VS Code busca `servers` (perfil de usuario) y, si no lo hay, `mcpServers` (proyecto).
- **A. Migración.** En `sync` se retiran de `.vscode/mcp.json` solo las entradas que escribió Axiom: la misma forma y sin claves añadidas (`env`, `headers`…). El texto, los comentarios, las `inputs` y los servidores del usuario se conservan. El fichero se borra solo si queda sin miembros, y la carpeta `.vscode` solo si queda vacía. Un `.vscode/mcp.json` ilegible no bloquea y no se toca. La desinstalación retira de él las entradas con la forma de Axiom, con el criterio de siempre.
- **A. Desinstalación compartida.** `.mcp.json` es de Claude Code y de VS Code. Si se desinstala uno con el otro instalado en el proyecto, las entradas de Axiom se quedan para el otro (`projectMCPJSONStaysInUse`, el mismo patrón que `.agents/skills`).
- **A. Preflight.** `WorkspaceUserConfigFiles` pasa a ser `.mcp.json`: uno ilegible se rechaza antes de escribir, porque la fusión genérica lo sustituiría por un documento vacío.
- **B. Formato camelCase nativo** (`preToolUse`/`postToolUse`) en `.github/hooks/axiom-branch-guard.json`. Es el formato primario documentado. La carga no trae el evento, así que la orden lo declara: `axiom branch guard --agent vscode-copilot --event pre|post`. El nombre del agente es el identificador del modelo (`vscode-copilot`), como `claude-code`, `codex` y `opencode`. `--event` es obligatorio con `vscode-copilot` y no se admite con los demás. El `matcher` limita la guarda a las herramientas de escritura (`create|edit|str_replace_editor|apply_patch`), para no lanzar un proceso en cada lectura.
- **B. Protocolo de salida de `vscode-copilot`.** Siempre termina con código `0`, porque cualquier otro código distinto de `2` deniega en `preToolUse` y rompería el fail-open. El bloqueo sale como JSON con `permissionDecision: "deny"` y `permissionDecisionReason`, el aviso como `additionalContext` y «nada que hacer» sin salida.
- **B. Rutas.** Como la documentación no fija los argumentos, `toolArgs` se acepta como cadena JSON o como objeto. Se leen `path`, `file_path` y `filePath`, y las cabeceras de parche (`*** Add/Update/Delete File:`) de `apply_patch`, venga el parche en un campo o como cadena entera. Solo cuentan `create`, `edit`, `str_replace_editor` (salvo su orden `view`) y `apply_patch`, aunque el `matcher` falle. Cualquier otra forma no da rutas y la guarda no hace nada (fail-open). Lo fijan las pruebas de carga.
- **B. Orden que nunca deniega por fallo.** En `preToolUse` del arnés, un código distinto de 0 deniega (2 de forma explícita; cualquier otro por fail-closed). Un `axiom` ausente del `PATH`, o uno anterior que no conoce `vscode-copilot` (sale con 2 por error de uso), bloquearía todas las escrituras. Por eso el fichero declara `bash` con `… || true` y `powershell` con `…; exit 0`: la decisión va solo por `stdout`, y un fallo permite la escritura (fail-open, como exige RF-BRA-22). Un tiempo agotado (`timeoutSec: 10`) también es fail-open en el arnés.
- **B. Fichero propio.** `.github/hooks/axiom-branch-guard.json` es entero de Axiom, con bytes deterministas: `sync` lo reescribe (como los plugins gestionados de OpenCode) y la desinstalación lo borra solo si no cambió, y si cambió avisa. Los demás ficheros de `.github/hooks` no se leen ni se tocan. Solo se instala en el ámbito workspace (`BranchGuardHooksFile` devuelve "" fuera del proyecto) y con el componente SDD, como el de Claude Code.
- **B. Manifiesto de capacidades: sin cambios.** La premisa de que el manifiesto debía reflejar los hooks no se sostiene. `AgentFeatureClaims` (v1) no declara hooks para ningún agente: Claude Code, Codex y OpenCode instalan la guarda sin una claim. Sus claims alimentan las proyecciones `Supports*()` del adaptador, y sus digests están fijados en `manifest_test.go`. Un campo nuevo cambiaría el JSON canónico de todos los agentes sin que nada lo leyera. El cambio queda en el comentario del adaptador, en `docs/agents.md` y en la spec viva.
- **B. Convivencia con `.claude/settings.json`.** Con Claude Code instalado, VS Code ejecuta también su guarda de `.claude/settings.json` (alias `PreToolUse`, carga snake_case). Se acepta y se justifica: las dos guardas usan el mismo motor, así que dan el mismo veredicto. Un bloqueo duplicado sigue siendo un bloqueo, y el aviso de Claude (código `2` en `PostToolUse`) solo queda en el registro de VS Code, así que el modelo no lo ve dos veces. Además, la guarda `claude-code` lee `tool_input.file_path`, y si Copilot manda los argumentos de su arnés (`path`) no encuentra ruta y no hace nada. Evitarlo exigiría distinguir en la carga a Claude Code de Copilot, y la documentación no da ninguna señal estable para hacerlo.
- **B. Preflight SDD: no viable hoy.** `axiom sdd-preflight-hook` (`internal/cli/sdd_preflight_hook.go:110-185`) se apoya en tres cosas que el arnés de Copilot no da:
  1. **Autoridad por transcripción.** Con una topología que exige grupos, deniega si la carga no trae `transcript_path` (con nombre `<session_id>.jsonl`). Después lee de ella las respuestas de `AskUserQuestion` del padre. En Copilot, `preToolUse` no trae `transcriptPath`: solo lo traen `agentStop`, `subagentStart`, `subagentStop` y `preCompact`. El formato de esa transcripción tampoco está documentado.
  2. **Bloquear y reescribir el prompt.** Deniega el lanzamiento o antepone el bloque del preflight al `prompt` con `updatedInput`. `subagentStart` trae la transcripción, pero «cannot block creation» y su carga no trae el prompt: solo puede anteponer `additionalContext`. `preToolUse` sobre `task` puede denegar y reescribir con `modifiedArgs`, pero la documentación no fija los argumentos de `task` (tipo de agente, prompt) y la carga no trae la transcripción.
  3. **Identidad del lanzador.** Distingue al padre de un hijo (`agent_id`/`agent_type`). Copilot añade `agentId`/`agentType` a los hooks de subagentes, pero el agente `general-purpose` no emite `subagentStart`.

  Forzarlo daría una guarda que no puede comprobar la autoridad. Sería peor que la doctrina actual, en la que el orquestador pide el preflight en el chat. Se reconsiderará cuando el arnés documente `transcriptPath` en `preToolUse` y los argumentos de `task`.
- **Fuera de alcance**: el ámbito global de VS Code (`~/.copilot/mcp-config.json` y `~/.copilot/hooks/`). El perfil de usuario `mcp.json` también es del arnés Local, así que queda como seguimiento.

## Alcance autorizado

Adaptador `internal/agents/vscode/`, inyección MCP (`internal/components/mcp`, `internal/components/engram`), desinstalación y `sync` de workspace, aviso de Workspace Trust (`internal/cli/workspace_instructions.go`), `internal/branch/engine/guard_hook.go` y su orden de la CLI, instalación del hook de la guarda, manifiesto de capacidades, `docs/agents.md`, pruebas de esos paquetes y la spec viva `change-branch-lifecycle` (RF-BRA-23).

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y no hay elección del usuario. Comprobaciones: `go test` de los paquetes tocados con `-run`. La suite entera la pasa la CI.

## Entrega

`single-pr`: una PR con un commit por tarea. Etiqueta `type:feature`.

## Tareas

- [x] T1 · Bloque A: VS Code declara sus MCP de proyecto en `.mcp.json` (`mcpServers`). `sync` y la desinstalación retiran las entradas de Axiom de `.vscode/mcp.json`. Aviso de Workspace Trust, comentario del adaptador y docs (`agents.md`, `usage.md`, `engram.md`, `platforms.md`, `manual-de-inicio.md`) actualizados. Pruebas nuevas: migración con comentarios e idempotencia, fichero vaciado, Claude y VS Code compartiendo `.mcp.json`, `.mcp.json` ilegible rechazado, `.vscode/mcp.json` ilegible intacto, desinstalación con Claude instalado, `RemoveJSONCObjectMember` y `RetireLegacyVSCodeServer`. Evidencia: `go test ./internal/agents/... ./internal/components/{mcp,engram,uninstall,filemerge}/ -run 'VSCode|Workspace|MCP|Context7|Uninstall|JSONC'` ok; `go test ./internal/cli/ -run 'VSCode|WorkspaceTrust|UninstallWorkspace|ComponentPaths|WorkspaceClaude|EngramClaudeWorkspace'` ok.
- [x] T2 · Bloque B: `axiom branch guard --agent vscode-copilot --event pre|post` con el protocolo de Copilot (`ParseGuardPayloadForEvent`, `RunGuardForEvent`, ayuda de la orden y de `axiom`). Pruebas de carga y de salida: bloqueo con `permissionDecision`, aviso con `additionalContext`, rutas por `path`/`file_path`/`filePath`/objeto/parche, sin evento, lecturas, `view` y cargas inválidas sin salida, errores de uso de `--event`. Evidencia: `go test ./internal/branch/engine/ -run 'GuardHook|ParseGuardPayload|GuardWithin|GuardSources'` ok; `go test ./internal/cli/ -run BranchGuard` ok; `go test ./cmd/axiom/ -run 'Help|Usage'` ok.
- [x] T3 · Bloque B: instalación, `sync` y desinstalación de `.github/hooks/axiom-branch-guard.json` sin tocar los hooks del usuario (`BranchGuardHooksFileProvider`, `VSCodeBranchGuardHooksJSON`, copia de seguridad del componente SDD). Comentario del adaptador y docs (`agents.md`, `usage.md`, `manual-de-inicio.md`). El manifiesto no cambia (ver Decisiones). Pruebas: formato del fichero (eventos, `matcher`, `|| true` y `; exit 0`), instalación idempotente en workspace y ninguna en global, borrado solo sin edición, hook del equipo intacto en la instalación y en la desinstalación de extremo a extremo. Evidencia: `go test ./internal/components/sdd/ -run 'BranchGuard|SkillRegistry|VSCode'` ok; `go test ./internal/components/uninstall/ ./internal/agents/vscode/ -run 'VSCode|Workspace'` ok; `go test ./internal/cli/ -run 'VSCode|WorkspaceTrust|UninstallWorkspace|ComponentPaths|WorkspaceClaude|BranchGuard|WorkspaceInstallSyncUninstall'` ok.
- [x] T4 · Preflight SDD: evaluación documentada en Decisiones (no viable). `## Spec viva` (RF-BRA-23) integrada con `axiom archive integrate --odd vscode-mcp-guardas`. Evidencia: integración `composed` de `change-branch-lifecycle` (más `INDEX.md`); una segunda ejecución no cambia nada.

## Criterio de terminado

- Una instalación de workspace con VS Code deja Engram y Context7 en `.mcp.json` y no deja entradas de Axiom en `.vscode/mcp.json`. Las del usuario se conservan.
- `sync` repetido es byte a byte idéntico, también con Claude Code y VS Code a la vez.
- `axiom branch guard --agent vscode` bloquea un documento de Axiom en la rama equivocada y avisa del resto, con el protocolo de Copilot. Siempre sale con código `0`.
- `.github/hooks/axiom-branch-guard.json` se instala, se sincroniza y se retira sin tocar otros ficheros de `.github/hooks/`.
- `go test` (con `-run`) de los paquetes tocados en verde.

Ruta de todas las tareas: inline. Las unidades se escribieron en un solo hilo: el repositorio no admite escritores en paralelo sin worktrees aislados. La exploración amplia (mapa de MCP, hooks y desinstalación) y la lectura de la documentación oficial se delegaron en exploradores de solo lectura, porque superaban el presupuesto de evidencia en línea.

## Progreso

- Worktree creado desde `origin/main` (`b4b5aa10`).
- T1 `dacd6a84`, T2 `911ad6db`, T3 `7cdd3613`, T4 `f2e589dc`. PR IGutierrezZ/axiom#323.
- Corrección de la CI (commit nuevo, sin reescribir la rama publicada):
  - `Deadcode Ratchet` marcaba `ParseGuardPayload` y `runGuard`, que solo llamaban las pruebas. Ahora cada uno recibe el evento y no quedan variantes `ForEvent`.
  - `docs_contract_test.go` fijaba la frase anterior del manual sobre los runtimes con guarda.
- Seguimiento fuera de alcance: el ámbito global de VS Code sigue escribiendo el `mcp.json` del perfil de usuario (`servers`, arnés Local). Su destino nativo es `~/.copilot/mcp-config.json` (`mcpServers`), que ya escribe el adaptador de Copilot CLI, así que conviene decidir entre unificarlos o retirarlo.

## No verificado

- Ninguna prueba ejecuta VS Code de verdad. El formato de `.github/hooks/*.json`, la carga camelCase y el protocolo de salida salen de la documentación oficial (25/11/2026) y los fijan pruebas de carga. Los argumentos de `create`, `edit`, `str_replace_editor` y `apply_patch` no están documentados: la lectura de `path`, `file_path` y `filePath` es defensiva.
- Que el arnés de Copilot exponga las herramientas de un servidor de `.mcp.json` sin `tools` no está documentado. Por eso se escribe `tools: ["*"]`. Que Claude Code acepta esa clave y `type: "stdio"` se comprobó con `claude mcp get`.

## Siguiente paso

Abrir la PR (`type:feature`), CI en verde, merge commit y `done`.

## Spec viva

### Capacidad: change-branch-lifecycle

```openspec
## MODIFIED Requirements

### Requirement: Instalación del Hook por `axiom sync` y Prohibición de Hooks de Git (RF-BRA-23)

`axiom sync` DEBE instalar `axiom branch guard` como hook en los runtimes donde existan hooks demostrados (Claude Code, Codex, OpenCode y VS Code Copilot) de forma idempotente y determinista (misma entrada, mismo resultado), sin duplicar entradas en ejecuciones repetidas y sin alterar las entradas de hooks ajenas. Una ejecución posterior de `axiom sync` que ya no deba instalarlo DEBE retirarlo. NO DEBE instalar hooks de git (`pre-commit`, `pre-push`, ni otros) en ningún repositorio, propio o de terceros. NO DEBE instalarse en Pi, que no tiene canal demostrado. Con el ámbito de instalación `workspace` (el predeterminado) el hook queda en la configuración del proyecto; con `--scope global` queda en la de usuario, que corre en todo repositorio. En ambos casos la inercia de RF-BRA-22 fuera de proyectos Axiom y fuera de repositorios con un cambio activo registrado es condición de esta instalación: el hook no hace nada, no lanza procesos y no escribe en ellos. Un proyecto Axiom es el que tiene un `axiom.yaml` propio o el que se alcanza mediante un puntero `.axiom-workspace`, en el directorio o en un ancestro; un `openspec/` suelto no basta. El formato del hook de Codex se verifica en su documentación oficial al implementar (A-2). En VS Code Copilot el hook se instala solo con el ámbito `workspace`, en un fichero propio de Axiom (`.github/hooks/axiom-branch-guard.json`, con el formato del arnés de Copilot) que no altera los demás ficheros de `.github/hooks`. Como en `preToolUse` de ese arnés cualquier código de salida distinto de 0 deniega la escritura, la orden instalada DEBE terminar con código 0 aunque `axiom` falte o falle, y la decisión DEBE ir solo por la salida estándar.
(Previously: los runtimes con hook eran Claude Code, Codex y OpenCode)
(Previously: la inercia exigida era solo fuera de proyectos Axiom)
(Previously: la inercia se justificaba solo por la configuración global de Claude Code y un `openspec/` suelto valía como marcador de proyecto Axiom)

#### Scenario: Instalación en los runtimes con hooks
- **DADO** un proyecto con Claude Code, Codex, OpenCode y VS Code Copilot configurados
- **CUANDO** se ejecuta `axiom sync`
- **ENTONCES** cada uno incluye una entrada de hook que invoca `axiom branch guard`

#### Scenario: Sincronizar dos veces no duplica
- **DADO** la instalación anterior
- **CUANDO** se ejecuta `axiom sync` por segunda vez
- **ENTONCES** la configuración de cada runtime es byte a byte idéntica a la anterior

#### Scenario: No se instalan hooks de git
- **DADO** un proyecto con repositorios
- **CUANDO** se ejecuta `axiom sync`
- **ENTONCES** ningún repositorio contiene hooks de git nuevos

#### Scenario: Las entradas ajenas se conservan
- **DADO** un runtime con hooks propios del usuario
- **CUANDO** se ejecuta `axiom sync`
- **ENTONCES** los hooks del usuario permanecen sin alterar

#### Scenario: Pi queda fuera
- **DADO** Pi configurado como runtime
- **CUANDO** se ejecuta `axiom sync`
- **ENTONCES** no se instala el hook en Pi y la ejecución termina con normalidad

#### Scenario: En un repositorio ajeno la instalación global es inerte
- **DADO** el hook instalado en la configuración global de Claude Code y un repositorio de terceros sin `axiom.yaml` ni puntero `.axiom-workspace` resoluble (aunque tenga un `openspec/`)
- **CUANDO** el runtime invoca el hook para escribir en ese repositorio
- **ENTONCES** el hook no bloquea, no avisa, no lanza procesos y no escribe nada

#### Scenario: VS Code Copilot recibe un fichero de hooks propio que nunca deniega por fallo
- **DADO** un proyecto con VS Code Copilot y un fichero de hooks del equipo en `.github/hooks/`
- **CUANDO** se ejecuta `axiom sync` con el ámbito `workspace`
- **ENTONCES** `.github/hooks/axiom-branch-guard.json` declara `axiom branch guard --agent vscode-copilot` en `preToolUse` y en `postToolUse` para las herramientas de escritura
- **Y** la orden termina con código 0 aunque `axiom` no esté en el `PATH`
- **Y** el fichero de hooks del equipo queda byte a byte igual

#### Scenario: Con el ámbito global VS Code Copilot no recibe hooks
- **DADO** VS Code Copilot configurado
- **CUANDO** se ejecuta `axiom sync --scope global`
- **ENTONCES** no se escribe ningún fichero de hooks de VS Code Copilot
```
