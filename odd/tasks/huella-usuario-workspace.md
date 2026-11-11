# Huella de usuario mínima: Axiom inerte fuera de sus proyectos

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/huella-usuario-workspace.md`.
> Espejo de recuperación en Engram: topic `odd/huella-usuario-workspace/tasks`, proyecto `axiom`.
> Worktree `C:\repos\axiom-wt\huella-usuario-workspace\axiom`, rama `fix/huella-usuario-workspace`, base `origin/main` `7940a5cf`.

## Objetivo

Que un equipo con Axiom instalado pueda trabajar en proyectos ajenos a Axiom sin que Axiom modifique ficheros, cambie permisos ni condicione el comportamiento de los agentes. Todo lo instalable debe vivir en el workspace del proyecto. A nivel de usuario solo queda lo que sea necesariamente de máquina, declarado y reversible.

## Problema

Análisis de solo lectura sobre `main` `c780061a` (Engram `analysis/user-level-footprint`, 2026-10-29):

1. **El hook del registro de skills no es inerte.**
   - Corre en `UserPromptSubmit` en Claude, en `SessionStart` en Codex, como plugin de OpenCode y como hook de Kiro.
   - En cualquier repositorio git crea `.atl/` sin `.gitignore`.
   - Reescribe la sección `## Skills` de un `AGENTS.md` existente (`internal/skillregistry/agents.go:146-160`).
   - Guarda un tema en Engram con el nombre de la carpeta.
   - Contradice REQ-25.22 y `docs/manual-de-inicio.md:239`.
2. **Los permisos ignoran el ámbito y fijan modos de autoaprobación.**
   - Siempre se inyectan en home: `permissions.Inject(s.homeDir, …)` en `internal/cli/run.go:1746` y `internal/cli/sync.go:1357`.
   - Los modos impuestos son `bypassPermissions` en Claude, `auto_edit` en Gemini y Qwen, `chat.tools.autoApprove` en VS Code y `bash "*": allow` en OpenCode y Kilo.
   - Ya existía un ODD `permission-scope-workspace` del 2026-10-14 (Engram #646). Nunca se implementó y su documento no está en `main`. La tarea T2 lo absorbe.
3. **El modo solo workspace no funciona en Claude Code.**
   - Los subagentes llevan fija la ruta `~/.claude/skills/...` (`internal/assets/claude/agents/sdd-apply.md:17-18` y el resto).
   - El MCP de Engram en workspace se escribe en `<ws>/.claude/mcp/engram.json` (`internal/agents/claude/adapter.go:172`), que Claude probablemente no lee.
   - La raíz del workspace es `os.Getwd()`, no la raíz del proyecto.
   - La instalación desde la TUI fija `ScopeGlobal` (`internal/cli/run.go:1907`).
4. **Huella de máquina amplia incluso con `--scope workspace`.**
   - GGA: binario, `~/bin` y PATH de usuario.
   - Engram: binario, PATH y `engram setup <agente>`.
   - `~/.axiom/state.json`: selección, persona, asignación de modelos y `rdd_mode` globales.
   - Backups, `bin` y caché.
   - Copia de skills de compatibilidad en `~/.agents/skills`.
   - Limpieza de skills retirados en home y refresco de plugins de OpenCode en home.
   - CodeGraph con `--location global`.
5. **Cinco adaptadores no tienen modo workspace** y escriben siempre en home: Codex, VS Code Copilot, Trae, Windsurf y Antigravity (`internal/cli/run.go:2680-2690`).
6. **`axiom uninstall` solo actúa sobre home.** No revierte workspaces, entradas del PATH ni `~/.axiom`.
7. **La documentación está desfasada.**
   - `docs/usage.md:248` dice que el predeterminado es `global`.
   - `docs/manual-de-inicio.md:204` promete no tocar `~/.claude`.
   - Rutas `~/.gentle-ai/bin` y `~/.gentle-ai/cache` en la documentación de OpenCode.
   - `AXIOM_NO_SELF_UPDATE` en `scripts/install.ps1:153`, que no tiene efecto.

## Por qué

En los equipos de cliente se trabaja en varios proyectos a la vez. Hoy, una instalación global, o las fugas de una instalación en workspace, imponen ODD y SDD, el output style, los hooks y los permisos en todos los proyectos. También ensucian repositorios ajenos con `.atl/` y con cambios en `AGENTS.md`.

## Alcance autorizado

- Registro del plan (este documento): **autorizado** el 2026-10-29 (carril ODD elegido por el usuario).
- Implementación: **pendiente de autorización** por fase. Las fases 3 y 4 dependen además de las decisiones D1–D4.

## Restricciones

- Con `--scope workspace`, `HOME` queda idéntico byte a byte salvo la huella de máquina declarada en T9.
- Axiom no escribe modos de autoaprobación ni borra los que el usuario ya tenga. Las listas `deny` se unen sin duplicados y con el orden del usuario primero.
- Los hooks instalados son fail-open y no escriben nada fuera de un proyecto Axiom.
- La instalación en global sigue disponible y explícita, con aviso. No se rompen las instalaciones existentes sin una ruta de migración (D4).
- No se compilan binarios sueltos: Cortex XDR los borra. Los arneses corren en proceso con `go test`.
- Artefactos y mensajes en castellano; identificadores en inglés.

## Decisiones pendientes (producto)

Las cuatro quedaron resueltas por el usuario el 2026-10-29: acepta las recomendaciones de D1, D2 y D4, y para D3 elige solo `axiom.yaml`.

- **D1 · Preset por defecto.** **Resuelta: según la recomendación.**
  - Recomendación: sacar GGA y CodeGraph del preset `full-gentleman` y dejarlos como opcionales.
  - El componente `permission` se queda, pero solo con `deny` y sin modos (lo cubre T2).
- **D2 · Dónde vive el estado por proyecto** (selección, persona, asignación de modelos). **Resuelta: según la recomendación (`<ws>/.axiom/state.json` sin versionar).**
  - Recomendación: `<ws>/.axiom/state.json` sin versionar, con `~/.axiom` solo para el registro del hub y los backups globales.
  - Alternativa: `axiom.yaml`, versionado. El equipo compartiría modelos y componentes, pero mezclaría preferencias personales con configuración del proyecto.
- **D3 · Criterio de «proyecto Axiom»** para hooks y guardas. **Resuelta el 2026-10-29: solo `axiom.yaml`.**
  - Cuenta el `axiom.yaml` propio en un ancestro y también el que se alcanza mediante el puntero multirepo `.axiom-workspace`. Ese puntero no es un marcador independiente: es la vía canónica para llegar al `axiom.yaml` del repositorio de specs (`internal/workspace/loader.go:185-229`). Excluirlo rompería los workspaces multirepo.
  - `openspec/` deja de valer como marcador para hooks y guardas.
  - Hoy `markerAt` lo acepta como `OriginOpenSpec` (`internal/workspace/locate.go:219-229`), lo que activa Axiom en repositorios ajenos que usan OpenSpec.
  - T3 decide si el cambio va en `markerAt` o solo en la entrada de las guardas. Antes hay que revisar qué usos legítimos tiene `OriginOpenSpec` (REQ-24.2).
- **D4 · Migración de equipos con instalación global.** **Resuelta: según la recomendación.**
  - Recomendación: un comando o guía basados en `axiom uninstall` global seguido de `axiom setup` por proyecto, con un `doctor` que detecte restos globales.

## Tareas

### Fase 1 · Inercia fuera de proyectos Axiom (sin decisiones bloqueantes salvo D3)

- [x] **T1 · Hook del registro de skills inerte fuera de proyectos Axiom** (ruta: delegada; disparador: guarda Go, plugin TS de OpenCode y Kilo, pruebas). Commit `9941eb18`.
  - Nuevo `workspace.IsAxiomProject(start) (root, ok)` en `internal/workspace/project.go`. Cuenta `axiom.yaml` o un puntero `.axiom-workspace` resoluble, nunca `openspec/` solo.
  - `RefreshSkip` en `internal/skillregistry/guard.go` añade el motivo `SkipNoAxiomProject`. Lo aplican los hooks, la CLI manual, el paso posterior a sync y `RegenerateSkillsIndex`, así que los hooks ya instalados quedan inertes sin volver a sincronizar.
  - Fuera de un proyecto Axiom no se escribe `.atl/`, no se toca `AGENTS.md` y no se guarda nada en Engram. Termina con código 0 y, con `--quiet`, sin salida.
  - Los plugins `skill-registry.ts` solo invocan la CLI y no cambian.
  - Desviación aceptada: el aviso sin `--quiet` sale por stdout, no por stderr, porque el camino heredado solo tiene escritor de stdout. Los hooks usan `--quiet`.
  - Verificación:
    - `go test ./internal/skillregistry/... ./internal/workspace/... -count=1`: ok (repetido por el padre).
    - `go test ./internal/app/...`: un fallo previo de la base (`TestTuiInstallOnThenSyncPreservesAndRefreshesOpenCodeActivation`, falta node/npm).
    - `go test ./internal/cli/ -run 'SkillRegistry|SkillIndex|Sync|PostSync|Skill'`: ok.
    - `go vet ./...`, `go build ./...` y deadcode-ratchet: ok.
  - Tamaño: +609/−36, de ellas unas 440 líneas de pruebas. Nivel `axiom review assess`: medium. RDD desactivado.
- [x] **T2 · Permisos con ámbito y sin modos de autoaprobación** (ruta: delegada; absorbe las tareas T1–T4 de `permission-scope-workspace`). Commit `8eb668f4`.
  - Destino según el ámbito en install, sync, backup y verify, mediante `permissionsRoot` y `noteSkippedPermissions` en `internal/cli/run.go` y `internal/cli/sync.go`. Un agente sin modo workspace se omite con un aviso en castellano por stderr y nunca cae a home.
  - Overlays retirados: `bypassPermissions`, `auto_edit`, `chat.tools.autoApprove`, y `bash "*": allow` y `read "*": allow` de OpenCode y Kilo.
    - Retirar `read` amplía lo planificado, pero responde al mismo criterio de no imponer comodines.
    - Gemini, Qwen y VS Code se quedan sin overlay y no se escribe nada para ellos.
  - `filemerge.UnionJSONArrays` une solo las rutas declaradas (`permissions.deny` de Claude): las del usuario primero y sin duplicados. Es idempotente.
  - Documentación de permisos actualizada: `components.md`, `agents.md`, `kiro.md`, `quickstart.md` y `manual-de-inicio.md`.
  - Verificación:
    - `go test ./internal/components/permissions/...`: ok (repetido por el padre).
    - `go test ./internal/components/filemerge/...`: dos fallos previos de la base (`TestWriteFileAtomicFollows{,Relative}DirectoryJunction`, unión de directorios en Windows).
    - `go test ./internal/cli/ -run 'Permission|ComponentPath|IncludePermissions|Scope|Backup|Verify'` y `-run 'TestRunInstall|TestRunSync|TestInstall|TestUninstall'`: ok.
    - `go vet`, `go build`, deadcode-ratchet y `gofmt`: ok.
  - Tamaño: +996/−147, de ellas unas 880 líneas de pruebas. Nivel: medium.
- [x] **T3 · Detección de proyecto y guardas en repositorios ajenos** (ruta: delegada; depende de D3). Commit `2f05d8b3`.
  - Decisión: `markerAt` y `Locate` no cambian. REQ-24.2 (`planning-home-resolution`) exige `openspec/` como marcador para los comandos explícitos, y hay 32 llamadas a `Locate`. El filtro `IsAxiomProject` se aplica en la entrada de cada hook:
    - `branch guard`, en sus caminos pre y post de `internal/branch/engine`;
    - `sdd-preflight-hook`, que usa el `cwd` del payload;
    - `review stop-hook`: fuera de proyecto no lee el modo RDD ni escribe marcas en `~/.gentle-ai/review-stop-hook`.
  - Fallo real confirmado: sin los filtros, un repo con `openspec/changes/x/` y sin `axiom.yaml` recibía un bloqueo de escritura y una denegación de preflight.
  - Sin binario `axiom`: los comandos fallan con código 127 en bash y 1 en PowerShell. No es 2, así que Claude no bloquea, aunque muestra un error no bloqueante en cada uso.
    - No se envuelven: la cadena del comando es la identidad de la entrada (hash de confianza de Codex, retirada por coincidencia exacta y pruebas de contrato).
    - `branch guard` sigue devolviendo 2 en errores de uso, por contrato fijado en una prueba.
  - Verificación:
    - `go test ./internal/workspace/... ./internal/branch/... ./internal/components/sdd/...`: ok, con `internal/branch/engine` ejecutado sin `AXIOM_STATE_DIR`, porque una prueba escribe su propio hub.
    - `go test ./internal/cli/ -run 'BranchGuard|Guard|Preflight|StopHook|Hook'`: ok. El padre repitió las 6 pruebas de repo ajeno: ok.
    - `go test ./internal/assets/...`, `go vet`, `go build`, deadcode-ratchet y `gofmt`: ok.
  - Tamaño: +263/−23. Nivel: medium.
- [x] **T3b · Plugin `sdd-task-result-artifacts.ts` inerte fuera de proyectos Axiom** (ruta: delegada; surge de T3 porque el plugin bloquea en memoria, no invoca la CLI). Commit `386e0cc0`.
  - Bloque `axiom-project-gate` idéntico en v1 y v2. Los cuatro hooks activos lo consultan y, fuera de un proyecto, no hacen nada. Si hay error, el resultado es inerte.
  - Digests históricos añadidos en `opencode_plugin_digests.go` para que se refresquen las copias instaladas.
  - Paridad con `workspace.IsAxiomProject` en 19 directorios de prueba, ejecutada con node.
  - Verificación:
    - `go test ./internal/assets/... ./internal/sddtaskresult/... ./internal/components/sdd/... ./internal/opencode/...`: ok. El padre repitió las pruebas `ProjectGate`: ok.
    - `go test ./internal/cli/ -run 'Plugin|OpenCode|Digest|Managed' -skip TestOpenCodeReviewTransport`: ok.
    - El grupo `TestOpenCodeReviewTransport*` se cuelga también en la base.
  - Tamaño: +499/−9. Nivel: medium.
  - Sin verificar: el host real de OpenCode v2 (`scripts/test-opencode-v2-host.py`) y el arnés v1 con bun, que no está instalado.

### Fase 2 · Modo solo workspace viable en Claude Code

- [x] **T4 · Rutas de skills resolubles por ámbito en subagentes y prompts** (ruta: delegada; disparador de escritura en 2+ ficheros). Commit `75faa77a`, +467/−133.
  - Marcador `{{AXIOM_SKILLS_DIR}}` (y `{{AXIOM_SKILLS_DIR_WIN}}` para la grafía de Kiro), resuelto en `internal/components/sdd/skills_scope.go` a partir de `adapter.SkillsDir("")`: relativo al proyecto en workspace, `~/...` en global.
  - Cubre subagentes, comandos, el prompt del orquestador de Kimi y el flujo perezoso `_shared/sdd-orchestrator-workflow.md` de Claude, Cursor, Kimi y Kiro. Las líneas que ya resolvían «workspace primero, si no home» se mantienen.
  - Salida global idéntica salvo una frase de Kiro («Kiro skills directory»); 10 goldens de Kiro regenerados.
  - Verificación: `go build`, `go vet`, deadcode-ratchet, `go test ./internal/components/... ./internal/assets/... ./internal/agents/...` y `internal/cli` filtrado: ok, salvo los fallos conocidos de filemerge.
- [x] **T4b · Rutas de skills de los overlays de OpenCode y Kilo** (ruta: delegada). Hallazgo de T4: misma fuga en `internal/assets/opencode/sdd-overlay-{single,multi}.json`. Commit `869d3556`, +185/−13.
  - Mismo marcador, resuelto con `resolveOverlaySkillsDir`, que valida que el JSON resultante sigue siendo válido.
  - OpenCode global sin cambios. Kilo global pasa de `~/.config/opencode/skills` a `~/.config/kilo/skills`, que es su `SkillsDir` real (`internal/agents/kilocode/adapter.go:94`): la ruta anterior era un fallo latente. Huella base de Kilo rederivada.
  - Verificación: `go test ./internal/components/... ./internal/assets/...` y `internal/cli -run 'OpenCode|Kilo|Overlay|Sync'`: ok, salvo filemerge.
- [x] **T5 · MCP de Engram en workspace en una ruta que el agente lee** (ruta: delegada). Commit `b68c3008`, +473/−5.
  - Claude en workspace fusiona `mcpServers.engram` en `<ws>/.mcp.json`, conservando lo del usuario. Un `.mcp.json` que no es JSON válido se rechaza sin tocarlo.
  - `<ws>/.claude/mcp/engram.json` se retira solo si tiene exactamente la forma que genera Axiom. Verify y backup cubren la ruta nueva, como context7.
  - Revisión de `MCPConfigPath` del resto: Kiro (`.kiro/settings/mcp.json`) es correcto; OpenCode, Kilo, Gemini, Qwen, Kimi, Cursor y Hermes no tienen evidencia de que el agente lea esa ruta en el proyecto. Pasa a R1.
  - Verificación: `go test ./internal/components/engram/... ./internal/components/mcp/...`, `internal/app` y `internal/cli` filtrado: ok, salvo el fallo conocido de node/npm.
- [x] **T6 · Raíz del proyecto en lugar de la carpeta actual** (ruta: delegada). Commit `f3bcc8dd`, +527/−10.
  - `workspace.ResolveRoot(start, homeDir)`: raíz del proyecto Axiom, si no la raíz git (la del worktree enlazado, nunca la del checkout principal; un repo git en HOME no cuenta), si no la carpeta actual.
  - Install, setup, sync y la sincronización de la TUI la usan con `--scope workspace`. Aviso en castellano por stderr cuando difiere de la carpeta actual. `findProjectRoot` no se reutiliza: busca marcadores de lenguaje.
  - Verificación: `go test ./internal/workspace/... ./internal/gitcommon/...`, `internal/app` e `internal/cli` filtrado (`TestRunSync|TestSync`, `TestRunInstall|TestInstall|TestTui|Backup|Verify`, `Scope|Workspace`): ok, salvo node/npm.
  - Abierto (producto): en multirepo con puntero, la raíz desde un repo de código es la carpeta maestra. Es correcto si el agente se abre en la maestra; si se abre en el repo de código, su `.claude` no se escribe.
- [x] **T7 · La TUI respeta el ámbito** (ruta: en línea; un fichero más su prueba). Commit `df59151f`, +79/−1.
  - `executeTUIInstallWithBackground` resuelve el ámbito con `ResolveInstallScope("")`: workspace por defecto, global solo con `AXIOM_INSTALL_SCOPE=global`. Las pruebas de transacción de compatibilidad de Windows fijan global.
  - Verificación: `go test ./internal/cli -run 'TUIInstall|TestExecuteTUIInstall|Scope'`: ok; `internal/app`: ok salvo node/npm.

### Fase 3 · Huella de máquina mínima (depende de D1, D2)

- [x] **T8 · Preset por defecto según D1** (ruta: delegada; el cambio alcanzó sync, TUI, docs y 12 ficheros). Commit `3be1d092`, +282/−13.
  - GGA sale de `full-gentleman` y de `ecosystem-only`; sigue disponible con `--component gga`, `custom` y la TUI. CodeGraph ya era opcional; ahora lo fijan pruebas.
  - Instalaciones existentes: `sync` conserva GGA si está en la selección guardada o, sin selección guardada, si `gga.Managed` lo detecta (`RestoreDetectedGGA`). Nunca desinstala.
  - Verificación: `internal/model`, `components/gga`, `tui/screens` e `internal/cli -run 'Preset|GGA|CodeGraph|CommunityTool|Sync|Install'`: ok.
- [x] **T9 · Inventario declarado de la huella de máquina** (ruta: delegada). Commits `1ab483fc` (+661/−31) y `c1c5eb34` (+485/−31).
  - Resultado con `--scope workspace`: `engram setup` se omite con un aviso (el MCP ya está en el proyecto). No se tocan `~/.agents/skills`, las skills retiradas en HOME ni los plugins de OpenCode en HOME. El hub de prompts de Kimi pasa al proyecto.
  - GGA, CodeGraph y los plugins de OpenCode elegidos explícitamente siguen escribiendo en HOME, con aviso de rutas y de PATH: es consentimiento explícito (D1).
  - `doctor`: apartado «Huella de máquina (nivel de usuario)», con cada elemento marcado `declarado`, `opcional` o `resto global`. GGA sale de `coreTools`.
  - Verificación: `components/gga`, `communitytool`, `doctor` e `internal/cli -run 'Doctor|Engram|GGA|CodeGraph|Compat|Retired|Sync|Install|Workspace|Footprint|Kimi'`: ok. Mutación comprobada: sin las guardas, `TestRunSyncWorkspaceLeavesHomeUntouched` falla.
  - Abierto: CodeGraph no tiene ubicación de proyecto (tarea aparte si se quiere). Los lanzadores de OpenCode en segundo plano escriben en `~/.axiom/bin` y PATH sin aviso al instalar; `doctor` los lista.
  - Dejar a nivel de usuario solo el binario `axiom`, el binario de Engram y el registro del hub.
  - El resto de pasos que escriben en home se omiten con `--scope workspace`:
    - copia de skills de compatibilidad en `~/.agents/skills`;
    - limpieza de skills retirados en home;
    - refresco de plugins de OpenCode;
    - `engram setup <agente>`;
    - CodeGraph `--location global`;
    - configuración de GGA.
  - `doctor` lista esa huella.
  - Hallazgo de T5: `engram setup claude-code` se sigue ejecutando con `--scope workspace` y escribe en home.
  - Hallazgos de V1 (fases 1 y 2): con `--scope workspace`, GGA escribe `~/bin/gga.{cmd,ps1}`, `AppData/Roaming/gga/` y `~/.local/share/gga/`, y añade `~/bin` al PATH de usuario. También se lanzan `engram setup claude-code` y `engram setup opencode`.
- [x] **T10 · Estado por proyecto según D2** (ruta: delegada). Commits `acfff69a` (+1408/−155) y `c9cadff0` (+138/−8).
  - `state.Store` global o de proyecto. Con `--scope workspace`, install, setup, sync y la TUI usan `<raíz>/.axiom/state.json`, con cerrojo propio (`statecoord.WithStoreLock`).
  - De proyecto: agentes, selección, persona, modelos, community tools, intents y huellas de binario y activos. De máquina, siempre global: `last_update_check`, `pending_sync`, `rdd_mode`.
  - Sin fichero de proyecto, el global se lee como semilla de solo lectura, sin campos de máquina y sin `InstalledAgents` (`c9cadff0`, decisión del orquestador: heredarlos haría que un sync escribiera en HOME la configuración de agentes sin modo workspace).
  - Ignorado con `.git/info/exclude` (solo `state.json` y su cerrojo; el resto de `.axiom/` tiene otros usos).
  - Verificación: `internal/state` (con git real y worktree enlazado), `statecoord`, `agents`, `app`, `tui` e `internal/cli` filtrado en tandas: ok, salvo los fallos conocidos.
  - **`rdd_mode` por clone: descartado por el usuario el 2026-10-30.** Se mantiene el contrato actual: por clone solo se puede desactivar (`ErrRDDModeRepositoryForcedOn` en `internal/reviewtransaction/rdd_mode.go:761`, «any off wins» en `docs/trigger-rules.md:72`). `rdd_mode` sigue siendo de máquina.
    - 2026-11-10: el usuario revocó esta decisión (D2 de `upstream-revision-nativa`). El RDD pasa a ser una preferencia de cada usuario en cada proyecto, sin versionar: vive en la entrada del proyecto del hub (`rdd_mode` en `~/.axiom/workspaces.json`, compartida por los worktrees); el global de máquina queda como valor por defecto y el off del clon sigue ganando.
  - Hallazgo para T12: sin fichero de proyecto, `sync --scope workspace` descubre agentes por las carpetas de HOME, y un agente sin modo workspace (Codex) escribe en `~/.codex` (`componentInjectionDirScoped` devuelve HOME). Ya ocurría antes de T10.
- [x] **T11 · Una única raíz de estado** (ruta: delegada). Commits `92d1ec65` (+434/−29) y `55c06cb8` (+89/−4).
  - Backups, `bin`, caché, dashboard y doctor resuelven `system.AxiomDir` (respetan `AXIOM_STATE_DIR`) en lugar de `~/.axiom` fijo.
  - Hallazgos de V1:
    - los backups caen en `~/.axiom/backups` aunque `AXIOM_STATE_DIR` esté fijado;
    - cada `sync` crea un backup completo (~335 KB), también cuando informa de que no hay cambios;
    - el stop-hook de review, dentro de un proyecto Axiom, escribe `~/.gentle-ai/review-stop-hook/v1/<sesión>.json`, una raíz de estado heredada.
  - Resultado: backups, `bin` de OpenCode, dashboard, `checkDiskSpace` de doctor, caché de uninstall, estado del stop-hook y marcador `consent-stale` resuelven `system.AxiomDir` y siguen `AXIOM_STATE_DIR`. Lo heredado en `~/.gentle-ai` queda como lectura de compatibilidad. Sin la variable, rutas idénticas.
  - Un sync que termina sin cambios borra la copia que acaba de crear (`55c06cb8`).
  - Verificación: `opencode`, `system`, `hub`, `components/uninstall`, `update`, `dashboard`, `tui` e `internal/cli` filtrado en tandas: ok. `internal/backup` `TestSnapshotterSkipsUnixSockets` falla también en la base (sockets Unix en Windows).
  - `internal/reviewtransaction` completo se cuelga (30 min y 12 min) en `TestRepairClassifiedAuthorityReplayRequiresUniqueRecordAndGlobalInventory/cancelled_bounded_discovery`. Esa prueba pasa aislada en la rama (4,9 s) y la rama no toca el paquete. No se pudo comparar con la base: falló la creación de un worktree temporal en la base.
  - Abierto: el manifiesto de Pi (`~/.gentle-ai/pi-codegraph.json`) y la caché del plugin `model-variants` de OpenCode (TypeScript, con digests) siguen en rutas fijas.
  - Regresión de T1 detectada aquí: dos pruebas de `cmd/axiom` (`TestSkillRegistryCompat*`) no creaban `axiom.yaml`. Corregido en `e853c651`; `go test ./cmd/axiom/`: ok.

### Fase 3b · Correcciones detectadas por V1 de la fase 3

- [x] **T11b · `doctor` reconoce el MCP de Engram del proyecto.** `checkEngramReachable` solo leía HOME (`engram.ReadPersistedStdioCommands(homeDir, …)`). En una instalación solo workspace avisaba «no persisted MCP configuration found» y recomendaba un `axiom sync` que no lo arreglaba. Ruta: delegada. Commit `74bc8429`, +230/−48.
  - `ReadPersistedProjectStdioCommands` lee HOME y, si el almacén efectivo es el del proyecto (`effectiveStateStoreWithRoot`), `<raíz>/.mcp.json` para Claude y `MCPConfigPath(raíz)` para el resto, sin parsing nuevo. Una instalación global no lee el `.mcp.json` de un repo cualquiera.
- [x] **T11c · Rollback con `AXIOM_STATE_DIR` fuera de HOME.** `rollbackRoots` (`internal/cli/run.go`) solo admitía HOME y el workspace, y desde T11 el lanzador de OpenCode vive en `system.BinDir`: un install fallido no se restauraba. Ruta: delegada. Commit `0a8e3ce7`, +184/−5.
  - Se admite además `system.BinDir(home)` cuando queda fuera de HOME (`backup.MachineRoots`), y `RestoreService` (comando `restore` y TUI) usa la misma lista. Solo `bin`: es lo único de la raíz de estado que entra en un manifiesto. Las raíces relativas se rechazan.
- [x] **T11d · El informe de sync no lista rutas heredadas que no escribe.** «OpenCode background launcher paths» incluía `~/.gentle-ai/bin/opencode.{cmd,ps1}`. Ruta: delegada. Commit `61558d41`, +96/−2.
  - `ActivationPlan.managedPaths()` filtra los heredados para el informe y la guía de reinicio; la limpieza de heredados no cambia.
- Verificación de la fase 3b: build, vet, gofmt y deadcode-ratchet ok; `internal/components/engram` ok; `internal/backup` ok salvo `TestSnapshotterSkipsUnixSockets` (base); `internal/cli` en tandas (`Doctor`, `Engram|Rollback|Restore|Backup`, `OpenCode|Background`, `Sync`): ok. `internal/opencode`: falló una vez `TestRunCatalogCommandDeadlineNotBlockedByInheritingDescendantWindows` (plazo de 300 ms superado con la máquina cargada por la otra sesión); pasa aislada dos veces y el paquete completo pasa al repetirlo. Esa prueba no toca el código cambiado.
- Abierto: `AXIOM_STATE_DIR` con un valor patológico (p. ej. `/`) haría admitir `/bin` en el rollback; no hay guarda para eso.

### Fase 4 · Adaptadores sin modo workspace (depende de R1)

- Ejecutada en otra sesión, en el worktree `C:\repos\axiom-wt\huella-usuario-workspace-f4\axiom` (rama `fix/huella-usuario-workspace-f4`, apilada sobre `5e8e3c25`). Integrada en esta rama el 2026-11-01 con `7689cd4c`, sin conflictos.
- El hallazgo de V1 de la fase 3 sobre `codex` persistido en `<raíz>/.axiom/state.json` queda cubierto por T12a: el estado heredado se conserva, pero ya no escribe en HOME.

- [x] **R1 · Investigación con fuentes primarias** (ruta: delegada, solo lectura).
  - Qué configuración de proyecto carga cada agente:
    - Codex: `.codex/` y `AGENTS.md` en la raíz.
    - VS Code Copilot: `.github/copilot-instructions.md`, `.github/prompts`, `.vscode/mcp.json`.
    - Windsurf: `.windsurf/rules`.
    - Trae y Antigravity.
  - Verificar además si OpenCode carga `<ws>/.config/opencode` o solo `opencode.json` y `.opencode/`, y lo equivalente en Gemini, Qwen, Kimi y Hermes.
  - **Hecha (2026-10-30, delegada a un investigador; documentación oficial, sin ejecutar agentes).** Tabla completa en Engram `odd/huella-usuario-workspace/fase-4`. Conclusiones:
    - OpenCode y Kilo, que ya tienen modo workspace, escriben en `<ws>/.config/opencode` y `<ws>/.config/kilo`, que el agente **no lee**. Leen `opencode.json`, `.opencode/` y `AGENTS.md` en la raíz (OpenCode), y `.kilo/` con `kilo.jsonc` (Kilo).
    - Gemini y Qwen escriben instrucciones en `.gemini/GEMINI.md` y `.qwen/QWEN.md`, que no se leen; leen `GEMINI.md` y `QWEN.md` en la raíz. Skills y `settings.json` sí coinciden.
    - Kimi: las skills en `.config/agents/skills` no se leen; MCP y configuración solo globales. Hermes: MCP solo global.
    - Cursor coincide casi entero.
    - Codex (verificado además por el orquestador): MCP, agentes y hooks en `<ws>/.codex/`. Las instrucciones pueden ir en `developer_instructions` de `<ws>/.codex/config.toml`, que se añaden a las base. Las skills, en `<ws>/.agents/skills`. Todo exige proyecto de confianza. La configuración de proyecto gana a la de usuario clave a clave.
    - VS Code: `.github/instructions/*.instructions.md`, `.github/skills`, `.github/agents`, `.github/prompts` y `.vscode/mcp.json`. Antigravity: `.agents/{rules,skills,workflows,mcp_config.json}`. Windsurf, ahora Devin Desktop: `.devin/` o `.windsurf/`. Trae: `.trae/`, con confianza baja.
  - Decisión abierta con el usuario: dónde van las instrucciones ODD/SDD (del equipo, no personales) en los agentes que solo leen un fichero raíz. El usuario prioriza VS Code Copilot, Copilot CLI, Codex, Claude, OpenCode y Gemini, y pide que prevalezca lo del proyecto sobre lo del usuario. Copilot CLI no tiene adaptador en Axiom. Investigación R1b de precedencia en curso.
- [x] **T12 · Modo workspace por adaptador según R1** (ruta: delegada; una rebanada por grupo de adaptadores). Hechas T12a–T12g; T12h (Antigravity) queda aplazada por decisión del usuario.
  - [x] **T12a · Ningún agente sin modo workspace escribe en HOME** (ruta: delegada; disparador: escritura en 2+ ficheros). Commit `e10f6bfb`, +657/−30.
    - `splitAgentsForScope` (`internal/cli/workspace_agents.go`) filtra en un único punto por flujo: `RunInstall`, `RunSync` tras `unique(agentIDs)`, `runSyncWithSelectionScoped` (TUI y API) y `executeTUIInstallWithBackground`. Cubre `--agent`, el estado de proyecto ya persistido y el descubrimiento por carpetas de HOME. Un aviso en castellano por agente, por stderr; en la TUI va en `SyncResult.OmittedAgents`.
    - Sin agentes válidos: `install` falla con un error claro y `sync` termina sin escribir. `componentInjectionDirScoped` ya no devuelve HOME en workspace. La TUI no persiste los agentes omitidos; un `codex` heredado en `<raíz>/.axiom/state.json` se conserva, pero ya no escribe en HOME.
    - Verificación: `go build`, `go vet`, `gofmt` y deadcode-ratchet: ok. `internal/cli -run '…|ComponentPath|Scope|Workspace'` y `-run 'TestRunSync|TestSync|TestRunInstall|TestInstall|Permission|TUIInstall'`: ok. `internal/app` e `internal/tui`: solo los fallos conocidos. Mutación: sin el arreglo, las 3 pruebas de HOME intacto fallan (96, 50 y 7 ficheros en HOME). El padre repitió las pruebas de HOME: ok.
    - Hallazgos: la TUI añade `--include-permissions` si Codex está descubierto (inocuo, revisar en T14); la pantalla de selección de la TUI sigue listando los agentes omitidos; `gga.Managed` lee `%APPDATA%\gga`, así que las pruebas de sync deben aislar `APPDATA`.
  - **R1b · Precedencia usuario/proyecto en los agentes prioritarios** (ruta: delegada, solo lectura; prioridad del usuario: VS Code Copilot, Copilot CLI, Codex, Claude, OpenCode y Gemini).
    - Criterio acordado: las instrucciones ODD/SDD son del equipo y van en el proyecto, en un fichero propio del agente. Nunca en un fichero raíz compartido (`AGENTS.md`, `GEMINI.md`). Lo del proyecto prevalece y lo del usuario convive.
    - Los seis lo admiten: Claude `.claude/CLAUDE.md`; Codex `developer_instructions` en `.codex/config.toml`; VS Code y Copilot CLI `.github/instructions/axiom.instructions.md` con `applyTo: "**"`; OpenCode `instructions` en `opencode.json`; Gemini `context.fileName` en `.gemini/settings.json` (sustituye la lista del usuario: hay que incluir `GEMINI.md`).
    - Matices: en Claude gana la skill personal con el mismo nombre (migración D4); en Copilot CLI gana el agente personal; VS Code y Copilot CLI no tienen `deny` duro de proyecto. `.claude/CLAUDE.md` hace que Claude deje de leer `AGENTS.md`. Copilot CLI lee el `.mcp.json` de Claude y no tiene adaptador en Axiom.
    - Abierto: Qwen y Hermes solo leen el fichero raíz. No son prioritarios y no se tocan.
  - [x] **T12b · OpenCode y Kilo en las rutas de proyecto que leen** (ruta: delegada; disparador: escritura en 2+ ficheros). Commit `79870a15`, +1405/−105, de ellas unas 900 de pruebas y documentación.
    - `agents.BindWorkspace` liga OpenCode y Kilo a la raíz. OpenCode usa `<ws>/opencode.json`, o el `opencode.jsonc` existente con sus comentarios, y `.opencode/{skills,commands,plugins,prompts/sdd}`. Kilo usa el `kilo.json(c)` existente o `.kilo/kilo.json`, y `.kilo/{skills,commands,plugins,prompts/sdd,rules}`.
    - Las instrucciones van en `.opencode/axiom.md` y `.kilo/rules/axiom.md`, registradas en `instructions` sin duplicados y conservando las entradas del usuario. El `AGENTS.md` raíz no se toca. Global sin cambios.
    - Un JSON de proyecto ilegible se rechaza antes de escribir. Los restos en `<ws>/.config/{opencode,kilo}` no se borran (mezclan contenido del usuario): se avisa.
    - Verificación: `go build`, `go vet`, `gofmt` y deadcode-ratchet: ok. `internal/agents/{opencode,kilocode}`, `components`, `assets` y `opencode`: ok salvo filemerge. `internal/cli -run 'OpenCode|Kilo|Overlay|Plugin|Workspace|Scope|ComponentPath'` y `-run 'TestRunSync|…|Verify|Backup'`: ok. `app` y `tui`: solo los fallos conocidos. El padre repitió las 5 pruebas nuevas de `internal/cli` y las de los adaptadores: ok. Riesgo sin evaluar (`axiom review assess` exige ejecutar el binario); la verificación independiente queda para V1.
    - Hallazgos:
      - `.gentle-ai-default-agent.json` queda suelto en la raíz del proyecto (mejor en `.opencode/`).
      - `default_agent: axiom-orchestrator` se escribe en el `opencode.json` del equipo; respeta uno existente. Pendiente de confirmar con el usuario.
      - `skillregistry` no escanea `.kilo/skills`.
      - Las fusiones JSONC reindentan en el segundo paso (fallo previo, también en global).
  - [x] **T12c · Gemini: instrucciones en un fichero propio** (ruta: delegada; disparador: escritura en 2+ ficheros). Commit `6ee78ed2`, +722/−20, de ellas unas 400 de pruebas y documentación.
    - `gemini.WorkspaceAdapter`, devuelto por `agents.BindWorkspace`: las instrucciones van en `<ws>/.gemini/AXIOM.md`, registradas en `context.fileName` de `.gemini/settings.json`. El valor son los nombres previos del proyecto (o `GEMINI.md`) más el propio, sin duplicados; una cadena previa se convierte en lista. Con un tipo inesperado se avisa y no se toca.
    - `path.join` confirmado en `memoryDiscovery.ts` (`findUpwardGeminiFiles`). El `GEMINI.md` raíz no se crea ni se toca.
    - Migración: `.gemini/GEMINI.md` se retira solo si contiene únicamente secciones de Axiom; si no, se avisa. Se corrigió además un fallo propio de la rebanada: la guía de enrutado usaba el adaptador global.
    - Qwen tiene un adaptador independiente y no cambia (lo fija una prueba).
    - Verificación: `go build`, `go vet`, `gofmt` y deadcode-ratchet: ok. `internal/agents`, `components` y `assets`: ok salvo filemerge; goldens de global sin regenerar. `internal/cli -run 'Gemini|Qwen|Workspace|Scope|ComponentPath|Instructions'` y `-run 'TestRunSync|…|Verify|Backup'`: ok. `app` y `tui`: solo los fallos conocidos. El padre repitió `-run Gemini` y `internal/agents/...`: ok.
    - Hallazgos:
      - `context.fileName` del proyecto sustituye al del usuario, así que un nombre que solo está en `~/.gemini/settings.json` deja de cargarse en ese proyecto. Va a T14 o se decide con el usuario.
      - `internal/assets/gemini/sdd-orchestrator.md:388` cita `.agent/skills/_shared/` (debería ser `.gemini/skills/_shared/`); va a T14.
      - La fusión de `settings.json` borra los comentarios (fallo previo).
  - [x] **T12d · Modo workspace de Codex** (ruta: delegada; disparador: escritura en 2+ ficheros). Commit `48a0185e`, +2085/−92, de ellas unas 1.300 de pruebas y documentación.
    - `codex.WorkspaceAdapter`, devuelto por `agents.BindWorkspace`. Codex sale de la lista de `adapterSupportsWorkspace`.
    - Las instrucciones se generan en `.codex/axiom-instructions.md` y se copian a un bloque `<!-- axiom:workspace-instructions -->` dentro de `developer_instructions` de `<ws>/.codex/config.toml`. Se conserva el texto del usuario fuera del bloque y no se usa `model_instructions_file`.
    - MCP y hooks van en `<ws>/.codex/`; las skills, en `<ws>/.agents/skills`.
    - No se escriben claves que Codex ignora en proyecto (`profiles` incluida): Engram en workspace omite perfiles y `model_instructions_file`.
    - La confianza se avisa en las acciones manuales; nada en `~/.codex`. Codex no tiene destino de permisos en ningún ámbito.
    - `filemerge.FormatTOMLString` escribe una cadena multilínea literal, o básica con escapes si hace falta. Corrige un fallo previo: los helpers TOML tomaban una línea `[...]` dentro de una cadena multilínea por una cabecera. Con TOML normal, la salida es idéntica.
    - `.codex/AGENTS.md` de instalaciones anteriores se retira solo si es de Axiom; si no, se avisa. Las pruebas de T12a usan ahora Windsurf y VS Code en lugar de Codex.
    - Verificación: `go build`, `go vet`, `gofmt` y deadcode-ratchet: ok. `internal/agents`, `components` y `assets`: ok salvo filemerge. `internal/cli -run 'Codex|Workspace|Scope|ComponentPath|Instructions|Hook'` y `-run 'TestRunSync|…|Verify|Backup'`: ok. `app` y `tui`: solo los fallos conocidos. El padre repitió `-run Codex`, `agents/codex` y las pruebas TOML de filemerge: ok.
    - Riesgos y hallazgos:
      - `.agents/skills` la leen también Gemini, OpenCode, Cursor, VS Code y Copilot CLI, que podrían ver skills duplicadas.
      - `developer_instructions` ocupa unos 100 KB y se inyecta en cada sesión, además del duplicado en `axiom-instructions.md`.
      - La nota de `engram setup codex` sigue sugiriendo escribir en HOME (T14).
      - Los residuos `.codex/skills` no se limpian.
      - T13 debe cubrir el bloque, `axiom-instructions.md` y `engram-passive-capture`.
  - [x] **T12e · Modo workspace de VS Code Copilot** (ruta: delegada; disparador: escritura en 2+ ficheros). Commit `0c4542b2`, +862/−44, de ellas unas 560 de pruebas y documentación.
    - `vscode.WorkspaceAdapter`, devuelto por `agents.BindWorkspace`. Las instrucciones van en `<ws>/.github/instructions/axiom.instructions.md` con `applyTo: "**"`. Las skills, en `.github/skills`. El MCP, en `.vscode/mcp.json` (clave `servers`, fusión JSONC que conserva servidores, `inputs` y comentarios fuera de `servers`).
    - El manifiesto de VS Code solo declara skills, instrucciones y MCP: no se crean `.github/agents`, `.github/prompts` ni `.github/hooks`. Sin permisos.
    - Una guarda impide que la limpieza de la persona antigua borre `.github/copilot-instructions.md` del equipo. Un `.vscode/mcp.json` ilegible se rechaza en el preflight. Aviso de Workspace Trust en las acciones manuales.
    - Las pruebas de T12a usan ahora Windsurf y Antigravity.
    - Verificación: `go build`, `go vet`, `gofmt` y deadcode-ratchet: ok. `internal/agents`, `components` y `assets`: ok salvo filemerge. `internal/cli -run 'VSCode|Copilot|Workspace|Scope|ComponentPath|Instructions'` y `-run 'TestRunSync|…|Verify|Backup|TUIInstall'`: ok. `app` y `tui`: solo los fallos conocidos. Mutación: sin la fusión JSONC, la guarda o el preflight, fallan 1, 2 y 1 pruebas. El padre repitió `-run VSCode` y `agents/vscode`: ok.
    - Hallazgos: posibles skills duplicadas si en el mismo proyecto están Codex (`.agents/skills`) y VS Code (`.github/skills`). T13 tiene casos de VS Code ligados al perfil de usuario.
  - **V1 de la fase 4 (2026-10-30): hecha** (ruta: delegada a un verificador independiente). Arnés temporal en proceso con `go test`, sin binarios, borrado después; `git status` limpio sobre `709cbd58`.
    - Preparación: `HOME`, `APPDATA`, `LOCALAPPDATA` y `AXIOM_STATE_DIR` temporales. HOME sembrado con configuración de usuario de Codex, OpenCode, Kilo, Gemini, Claude, Windsurf y VS Code. El proyecto tiene `AGENTS.md`, `GEMINI.md`, `.github/copilot-instructions.md`, `opencode.json`, `.vscode/mcp.json` y `.gemini/settings.json` del equipo.
    - Install con codex, opencode, kilocode, gemini-cli, vscode-copilot, claude-code y windsurf, seguido de dos sync. Después, un sync en un segundo proyecto sin estado, con descubrimiento por HOME.
    - HOME, APPDATA y LOCALAPPDATA: 0 creados, 0 modificados y 0 borrados en todos los pasos. Windsurf se omite con aviso. Nada bajo `<ws>/.config`, ni `.codex/AGENTS.md`, `.gemini/GEMINI.md` o `.qwen`.
    - Cada agente queda en sus rutas de R1. El TOML de Codex parsea, sin claves ignoradas. `instructions`, `context.fileName` y `.vscode/mcp.json` conservan lo del equipo. `GEMINI.md` y `.github/copilot-instructions.md` quedan idénticos.
    - El segundo sync no cambia nada (`noop`). Verify: 265 ok en install, 281 ok en sync y 0 fallos. Global de Codex sin regresión: sigue escribiendo en HOME.
    - Cambios esperados:
      - El primer sync añade a `AGENTS.md` el bloque `axiom:skills-index` (T1, proyecto Axiom).
      - `.codex/config.toml` reordena las tablas `[mcp_servers.*]` una vez.
      - `.claude/CLAUDE.md` pierde una línea en blanco (ya conocido).
    - Hallazgo nuevo: con varios agentes, las rutas del índice de skills de `AGENTS.md` apuntan solo a `.opencode/skills/...`.
    - No verificable en proceso: `engram setup` real, que cada agente lea de verdad las rutas (R1 es documental), hooks lanzados por shell y un perfil real de VS Code.

#### Resumen de la fase 4 (2026-10-30)

- Commits: `e10f6bfb` T12a (+657/−30), `79870a15` T12b (+1405/−105), `6ee78ed2` T12c (+722/−20), `48a0185e` T12d (+2085/−92), `0c4542b2` T12e (+862/−44). Documentación: `24b5981c`, `3219c47c`, `d7de4e01`, `90cd262b`, `709cbd58` y el commit de este cierre. Unas 5.700 líneas en total, más de la mitad de pruebas y documentación.
- Modo workspace por adaptador:
  - **Sí:** Claude, OpenCode y Kilo (rutas corregidas), Gemini (instrucciones corregidas), Codex y VS Code Copilot (nuevos), y Cursor y Kiro (sin cambios).
  - **Sin cambios, con rutas que no se leen:** Qwen y Hermes, que solo leen el fichero raíz, y Kimi, cuyas skills están en `.config/agents/skills`. Abiertos.
  - **Sin modo workspace, omitidos con aviso:** Windsurf (ahora Devin Desktop), Trae (documentación no legible) y Antigravity. R1 respalda `.agents/` para Antigravity, pero no es prioritario.
  - Copilot CLI no tiene adaptador, pero lee `.github/instructions`, `.github/skills` y el `.mcp.json` de Claude.
- Decidido por el usuario el 2026-10-30: se mantiene `default_agent: axiom-orchestrator` en el `opencode.json` del equipo. Respeta un valor previo.
- Decisiones del usuario, segunda ronda (2026-10-30):
  - **Gemini `context.fileName`:** manda el proyecto. Se documenta en T14.
  - **Qwen, Hermes y Kimi:** se retiran de lo seleccionable al instalar y se conserva el código (T12f).
  - **Antigravity:** se activará más adelante en workspace, con sus rutas reales en `.agents/`.
  - **Skills duplicadas:**
    - Gentle AI upstream no lo resuelve (`main` `9e7738a8`). Escribe una copia por agente más una copia de compatibilidad en `~/.agents/skills`.
    - En upstream, [#3293](https://github.com/Gentleman-Programming/gentle-ai/issues/3293) (colocación por runtime) se cerró por inactividad y [#5869](https://github.com/Gentleman-Programming/gentle-ai/issues/5869) (124 avisos `duplicate skill name` en OpenCode) sigue abierto.
    - Claude Code no lee `.agents/skills` (verificado en code.claude.com/docs/en/skills).
    - **Decisión:** en workspace, una sola copia compartida en `.agents/skills` y `.claude/skills` solo si Claude está instalado (T12g).
- [x] **T12f · Qwen, Hermes y Kimi fuera de la selección** (ruta: delegada; disparador: escritura en 2+ ficheros). Commit `468c1449`, +419/−29.
  - Punto de verdad único: `catalog.IsSelectableAgent` y `SelectableAgents()` (`internal/catalog/agents.go`).
  - `install --agent` con alguno de ellos falla con un error claro, también mezclado con otros. La TUI no los lista ni los preselecciona, y el descubrimiento por HOME no los añade.
  - Sync en workspace: se omiten con aviso (filtro de T12a). Sync en global con estado previo: sin cambios. Uninstall, doctor y verify no cambian. Los adaptadores y sus pruebas se conservan.
  - Verificación: `go build`, `go vet`, `gofmt` y deadcode-ratchet: ok. `internal/agents`, `model` y `components`: ok salvo filemerge. `internal/cli -run 'Qwen|Hermes|Kimi|Workspace|Scope|Discover|Agent'` y `-run 'TestRunSync|…|TUIInstall'`: ok. `app` y `tui`: solo los fallos conocidos. El padre repitió las pruebas `Unselectable` de `cli` y `tui`.
  - Hallazgos:
    - Reinstalar desde la TUI quita del estado un agente retirado que ya estuviera en él.
    - `docs/usage.md:264` (hub de prompts de Kimi) y las listas «Full (sub-agents)» quedan para T14.
- [x] **T12g · Una sola copia de las skills en workspace** (ruta: delegada; disparador: escritura en 2+ ficheros). Commit `e7e6f598`, +1705/−49.
  - Codex, VS Code Copilot, OpenCode, Kilo, Gemini y Cursor usan `<ws>/.agents/skills`. Cada uno se verificó en su documentación oficial; en VS Code, la lectura de `.agents/skills` corresponde a las sesiones Local (vista de chat).
  - Claude usa `.claude/skills`, solo si está instalado. Kiro se queda en `.kiro/skills`, porque no se confirmó que lea `.agents/skills`. Paquete `sharedskills`; Cursor recibe un adaptador de workspace solo para esto.
  - `workspaceSkillsDirs` escribe cada carpeta una vez por ejecución; backup y verify salen sin rutas repetidas. `_shared` se escribe con el hueco genérico `<runtime>`, así que el contenido no depende del agente.
  - Índice: `dedupeBySkillName` prefiere `.agents/skills`. El orden de `ProjectSkillDirs` no cambia, porque lo espeja el plugin TS.
  - Migración (`legacySkillsRetirementStep`): una copia anterior en una carpeta propia se retira solo si es una skill de Axiom, idéntica byte a byte y con la copia compartida ya escrita. Si no, se conserva con un aviso. Las skills ajenas no se tocan.
  - Verificación: `go build`, `go vet`, `gofmt` y deadcode-ratchet: ok. `agents`, `components`, `assets` y `skillregistry`: ok salvo filemerge. `internal/cli` en tandas: ok. `app` y `tui`: solo los fallos conocidos. El padre repitió las pruebas `SharedSkills|WorkspaceSkills`, `components/sdd` y `skillregistry`: ok.
- **V1b de la fase 4 (2026-10-30): hecha** (ruta: delegada a un verificador independiente; arnés temporal borrado; `git status` limpio sobre `e7e6f598`).
  - Siete agentes en workspace, todo lo seleccionable:
    - las skills de Axiom están solo en `.agents/skills` y `.claude/skills`, con las mismas 20 entradas;
    - las skills del equipo quedan intactas;
    - no hay marcadores sin resolver y el índice de `AGENTS.md` apunta a `.agents/skills`;
    - HOME no cambia;
    - el segundo sync es `noop` y verify da 0 fallos.
  - Migración: las copias idénticas se retiran, la modificada se conserva con aviso y la ajena queda intacta.
  - Qwen, Hermes y Kimi: `install --agent` falla sin escribir nada, no se descubren y, si ya estaban en el estado, se omiten con aviso.
  - Con solo Claude no se crea `.agents/skills`. El global de OpenCode sigue igual.
  - Observaciones previas, ajenas a T12g:
    - `.kilo/commands/sdd-apply.md` conserva `{{GENTLE_AI_AUTHORITY_FIRST_TERMINAL_PROCEDURE}}`, porque `rendersReviewLifecycle` no incluye a Kilo.
    - `adapterSkillBackupTargets` repite dos rutas de `judgment-day`; el backup final las deduplica.
- [ ] **T12h · Antigravity en workspace: descartado por ahora, a revisar en el futuro** (ruta: delegada; el writer paró antes de escribir código, sin commit).
  - La [documentación de reglas](https://antigravity.google/docs/rules) (verificada por el orquestador el 2026-11-01) fija tres restricciones:
    - corta cada fichero de regla a 24.000 bytes, tras expandir `@[label](path)`;
    - las reglas globales y `always_on` comparten 20.000 tokens, y al pasarse los ficheros más grandes se degradan a punteros;
    - cada `.agents/rules/*.md` exige el frontmatter `trigger`, o se descarta en silencio.
  - El orquestador de Axiom ronda los 100 KB (el asset de Antigravity pesa 40.300 bytes), así que no cabe en una regla. Probablemente ya se trunca o degrada hoy en global (`~/.gemini/GEMINI.md` cuenta en el presupuesto); no se ha comprobado ejecutando Antigravity.
  - MCP de proyecto, según la [documentación de MCP](https://antigravity.google/docs/mcp): `.agents/mcp_config.json`, con raíz `mcpServers`.
  - **Decisión del usuario (2026-11-01):** Antigravity sigue omitido en workspace con aviso, como hasta ahora. Se revisará cuando esté el plugin de Antigravity que el usuario tiene pendiente de subir a gentle-ai. Al retomarlo queda por decidir cómo repartir el orquestador en `.agents/rules`: trozos de menos de 24 KB, o un núcleo `always_on` y el resto `model_decision`.
- Commits añadidos al resumen: `468c1449` T12f (+419/−29), `e7e6f598` T12g (+1705/−49), y la documentación `a64753ae`, `d90156d4` y este cierre. Antigravity queda descartado en workspace por ahora (T12h). La rama sigue lista para integrarse en `fix/huella-usuario-workspace`.
- Decisiones abiertas para el usuario:
  - Skills duplicadas entre `.agents/skills` (Codex) y las carpetas propias de otros agentes.
  - `context.fileName` de Gemini sustituye la lista del usuario.
  - Qwen, Hermes y Kimi.
  - Antigravity: aplazado hasta el plugin de gentle-ai (T12h).
- Para T13: el bloque de `developer_instructions`, `axiom-instructions.md`, `.opencode/axiom.md`, `.gemini/AXIOM.md`, `.github/instructions/axiom.instructions.md` y las entradas de `instructions` y `context.fileName`.
- Para T14: el asset de Gemini con `.agent/skills`, la nota de `engram setup codex` y el índice de skills que apunta a una sola carpeta.
- Rama lista para integrarse en `fix/huella-usuario-workspace`.

### Fase 5 · Reversibilidad y documentación

- [x] **T13 · `axiom uninstall` con ámbito** (ruta: delegada; disparador: 2+ ficheros no triviales). Commits `f8038ac8`, `3c678572`, `56c937b6`, `ea5bcde6`, `886721c6`, `172d4288` (+3.557/−140 en 27 ficheros).
  - `--scope workspace|global` con `ResolveInstallScope`; en workspace, raíz con `resolveWorkspaceDir` en CLI y TUI, y `--all` solo recorre agentes con modo workspace.
  - Reversión por adaptador (Claude, OpenCode, Kilo, Gemini, Codex, VS Code, Cursor, Kiro y skills compartidas). Solo se retira lo que tiene la forma de Axiom; lo editado por el usuario se conserva con aviso.
  - Permisos: solo las entradas del overlay de Axiom; no toca `defaultMode`, `defaultApprovalMode` ni `chat.tools.autoApprove`.
  - Estado: sin agentes restantes se borran `.axiom/state.json`, su cerrojo y las líneas de `.git/info/exclude` (`state.RemoveProjectStateIgnored`).
  - Global: con `--all` retira las entradas de PATH gestionadas (Engram solo en Windows, `~/bin` solo con GGA de Axiom, `<estado>/bin`). `--purge-state` (y `--purge-hub`) purga la raíz de estado con guardas contra HOME, el proyecto y la raíz del disco. `doctor` sin restos tras desinstalar.
  - Comprobación: build, vet, gofmt y deadcode ok; `uninstall`, `permissions`, `persona`, `engram`, `state` ok; `internal/cli` en tandas (`Uninstall|Restore|Doctor|Footprint|Path|OpenCode`, `TestUninstallWorkspace|TestUninstallGlobal`) ok. Repetido por el orquestador: `uninstall` y `TestUninstallWorkspace|TestUninstallGlobal` ok.
  - Hallazgos: la guía de enrutado de VS Code en workspace escribe `<raíz>/AppData/Roaming/Code/User/prompts/axiom.instructions.md` (bug de install, T13b); `share:"disabled"` de `opencode.json` se retira aunque lo pusiera el usuario; los JSON se reescriben sin comentarios; no se retiran copias de migración en `.opencode/skills` o `.codex/skills`. Una prueba intermedia dejó `C:\backups\<marca>\manifest.json` fuera del worktree, pendiente de borrar a mano.
- [x] **T13b · Guía de enrutado de VS Code en workspace** (ruta: inline; un fichero de código y su prueba). La instalación en workspace ya no escribe la ruta de usuario de VS Code dentro del proyecto: `routingGuidanceOptions` pasa el adaptador ligado de cualquier agente con modo workspace (`boundToWorkspace`), y la guía va a `.github/instructions/axiom.instructions.md`. Prueba en rojo sin la corrección (faltaba `agent-routing` y aparecía `AppData` en el proyecto) y en verde con ella; `internal/cli -run 'VSCode|Workspace|Codex|Gemini|Uninstall'` ok. La limpieza de ese resto en uninstall se mantiene para instalaciones anteriores.
- [x] **T14 · Documentación alineada** (ruta: delegada; disparador: 2+ ficheros). Commits `60bd608d`, `2f4aed89`, `6d6be36b`, `85b59ede`, `33fd0206`, `8ff19b8e`, `f2a7d69d`, `1f37111a`.
  - Hecho: ámbito workspace por defecto y huella en `usage.md`, `manual-de-inicio.md`, `quickstart.md` y `non-interactive.md`; modo workspace por agente, precedencia de Gemini y `engram setup codex` en `agents.md` y `engram.md`; notas en `components.md`, `platforms.md`, `opencode-profiles.md` y `kiro.md`; REQ-25.22; `install.ps1` usa `GENTLE_AI_NO_SELF_UPDATE`; el asset de Gemini usa `{{AXIOM_SKILLS_DIR}}`; el aviso de instalación global ya no dice «(default)».
  - Comprobación: build, vet y gofmt ok; `assets`, `components/sdd`, `agents/gemini`, `update`, `app` (docs), `workspace`, `skillregistry` ok. Repetido por el orquestador: `components/sdd` ok.
  - Hallazgos: la caché `model-variants` no la lee ningún código, ignora `AXIOM_STATE_DIR` y `uninstall --scope global` solo borra la v1; OpenClaw y Pi pasan el filtro de workspace sin adaptador ligado (se revisa en V1).
  - Lista original:
  - `docs/usage.md:248`.
  - `docs/manual-de-inicio.md:204,239`.
  - Redacción de REQ-25.22 en la spec viva.
  - `docs/components.md`, `docs/platforms.md` y las rutas de `docs/opencode-profiles.md`.
  - `AXIOM_NO_SELF_UPDATE` en `scripts/install.ps1:153`, alineado con `GENTLE_AI_NO_SELF_UPDATE`.
- [ ] **V1 · Verificación de extremo a extremo** (ruta: delegada; al cierre de cada fase).
  - Con `HOME` temporal: hash de `HOME` antes y después de `setup` y `sync` en workspace.
  - Sesión simulada de hooks en un repo ajeno: ningún fichero creado ni modificado.
  - **Fases 1 y 2 (2026-10-30): hecha.** Arnés temporal en proceso con `go test` (sin binarios), borrado después; `git status` limpio sobre `375054ea`.
    - Instalación: install y dos sync con `--scope workspace` para Claude Code y OpenCode desde `proyecto/src/app`. Pasan T6 (todo en la raíz; la subcarpeta, vacía), T5 (`mcpServers.engram` en `<raíz>/.mcp.json`), T4 (19 subagentes sin `~/.claude/skills` ni marcadores sin resolver), T2 (solo `permissions.deny`, 24 entradas; el `settings.json` del home, intacto) y T4b (overlays con rutas relativas, sin `"*": "allow"`). Verify: 126 ok y 0 fallos.
    - `HOME` no queda idéntico: 23 entradas creadas (GGA y backups), 0 modificadas y 0 borradas. Lo sembrado por el usuario, intacto. No se crea nada en `~/.agents/skills`, `~/.claude` ni `~/.config/opencode`. Las escrituras quedan asignadas a T8/T9 (GGA, `engram setup`, PATH) y a T11 (backups).
    - Hooks en dos repos ajenos con `openspec/` y `AGENTS.md`, con RDD apagado y encendido: refresco del registro de skills, `branch guard` (Claude, Codex y OpenCode, antes y después), preflight SDD y stop-hook de review. Todos salen con 0 y no escriben nada. Control positivo con `axiom.yaml`: el guard bloquea con 2, el refresco escribe `.atl/` y el preflight deniega.
    - No verificable en proceso: autoactualización, descargas reales, `engram setup` real, hooks lanzados por shell y que Claude y OpenCode lean de verdad las rutas de proyecto (R1).
    - Menores: `sync` dice «Skills indexed: 7 in AGENTS.md» sin `AGENTS.md` en la raíz, y quita una línea en blanco de `.claude/CLAUDE.md` respecto a install.
  - **Fase 3 (2026-10-30): hecha.** Arnés temporal en proceso en `internal/cli` e `internal/app`, borrado; `git status` limpio sobre `5e8e3c25`.
    - Instalación en workspace con el preset por defecto (CLI y TUI): `HOME` sin cambios tras install y dos sync (las fases 1 y 2 dejaban 23 entradas). Sin `engram setup` ni cambios de PATH, con el aviso de `engram setup` manual. Backups en `AXIOM_STATE_DIR/backups`; el segundo sync, sin cambios, no deja copia. GGA fuera del preset. Verify: 126 ok y 0 fallos.
    - GGA explícito: escribe solo lo declarado en `HOME` (`~/bin/gga.*`, `AppData/Roaming/gga`, `~/.local/share/gga`, `~/bin` en el PATH) y lo avisa.
    - Estado por proyecto: A y B independientes, sin `codex` heredado; el global con el mismo hash; `.axiom/state.json` y su cerrojo ignorados con `.git/info/exclude`, el resto de `.axiom/` no. Install global sigue escribiendo el global.
    - `doctor`: apartado «Huella de máquina» presente, sin fallo por GGA ausente y con los restos globales marcados.
    - Global sin `AXIOM_STATE_DIR`: `engram setup` se ejecuta y las rutas son `~/.axiom`, como antes; permisos solo con `deny`.
    - Hallazgos nuevos: T11b, T11c y T11d (fase 3b), y el estado persistido de Codex (fase 4). Las asignaciones de modelos de la semilla global pasan al proyecto, como prevé T10.
  - **Fases 4 y 5 (2026-11-01): hecha, con hallazgos.** Arnés temporal en proceso en `internal/cli`, borrado; `git status` limpio sobre `7d620117`.
    - Workspace por agente (Claude, OpenCode, Kilo, Gemini, Codex, VS Code, Cursor, Kiro; por separado y juntos): HOME, APPDATA y LOCALAPPDATA sin cambios; cada agente solo en sus rutas de proyecto; sin `AppData`, `Library` ni `.config` en el proyecto; el segundo sync no cambia nada.
    - Uninstall workspace con contenido del usuario sembrado: 0 creados y 0 borrados frente al estado previo; 8 ficheros solo reformateados; estado y `.git/info/exclude` restaurados; idempotente; HOME intacto.
    - Hooks en repo ajeno: todo sale con 0 y sin cambios.
    - Paquetes: build, vet, gofmt y deadcode ok; `uninstall`, `agents`, `state` ok; `internal/cli` en tandas ok; `internal/app` solo con el fallo de entorno conocido (runtime de OpenCode).
    - Hallazgos (fase 5b):
      - H1 (media): OpenClaw pasa el filtro de workspace y escribe `~/.openclaw/openclaw.json` (MCP de Engram) en HOME; uninstall no lo retira y deja `SOUL.md` en el proyecto.
      - H2 (media): Pi lanza `pi install` sin `-l` en workspace (`internal/agents/pi/adapter.go:266`): en real instalaría paquetes a nivel de usuario. No verificable en proceso.
      - H3 (media-alta): tras install y `uninstall --scope global --all` quedan `~/.config/kilo/opencode.json`, `~/.config/kilo/plugins/*.ts`, `~/.kiro/steering/axiom.md` y en `~/.codex/config.toml` `mcp_servers.context7`, `features.multi_agent` y `agents.*`. `doctor` solo detecta los plugins de Kilo (y los rotula como de OpenCode). Sin comparar con la base.
      - H4 (media): si `AGENTS.md` tenía un `## Skills` del equipo, el sync lo adopta como índice gestionado y uninstall lo borra sin restaurarlo.
      - Menores: `.gentle-ai-default-agent.json` en la raíz del proyecto (OpenCode); uninstall global borra `mcp.json` vacíos que eran del usuario.

### Fase 5b · Correcciones detectadas por V1 de la fase 5 (autorizada el 2026-11-01)

- [x] **T15 · Pi en workspace sin escribir en HOME (H2)** (ruta: delegada; disparador: lectura que prepara la escritura y comprobación de la documentación oficial de Pi). Con `--scope workspace`, los paquetes se instalan en el proyecto (`pi install -l`) y el MCP de Engram no se declara a nivel de usuario. Si la documentación oficial no permite dejarlo todo en el proyecto, Pi se omite en workspace con el aviso común, como Trae y Windsurf.
  - Hecho (`22478be0`): Pi se omite en workspace. `pi install -l` y `.pi/mcp.json` existen (`docs/packages.md` de badlogic/pi-mono). En cambio, `pi-engram init`, `~/.pi/npm/package.json` y `APPEND_SYSTEM.md` no tienen ruta de proyecto documentada.
- [x] **T16 · OpenClaw en workspace (H1)** (ruta: delegada, mismo escritor que T15). Si OpenClaw no tiene configuración de proyecto según su documentación oficial, se omite en workspace con el aviso común. Así no se escribe `~/.openclaw/openclaw.json` ni `SOUL.md` en el proyecto. Prueba de que el plan de workspace no lo incluye.
  - Hecho (`22478be0`, mismo commit que T15): OpenClaw se omite en workspace. Su configuración es `~/.openclaw/openclaw.json` y no tiene configuración por proyecto (docs.openclaw.ai/gateway/configuration).
  - Pruebas: `splitAgentsForScope` los omite, y un install en workspace con Pi y OpenClaw deja HOME y el proyecto idénticos, sin lanzar comandos.
  - Docs: `agents.md`, `usage.md`, `platforms.md` y `manual-de-inicio.md`.
- [x] **T17 · `## Skills` del equipo en `AGENTS.md` (H4)** (ruta: delegada, mismo escritor). El sync solo adopta una sección `## Skills` sin marcadores cuando tiene la forma del índice que generaba Axiom. Si es del equipo, la conserva y añade el bloque gestionado aparte. Uninstall solo retira el bloque con marcadores. Prueba de ida y vuelta: `AGENTS.md` idéntico tras install, sync y uninstall.
  - Hecho (`851d1f22`): `AdoptSkillsIndexMarkers` solo adopta una tabla con la cabecera de cuatro o tres columnas de Axiom, su fila separadora y únicamente filas de tabla.
  - Pruebas de ida y vuelta en `skillregistry` y en `internal/cli` (install, sync y `uninstall --all`): `AGENTS.md` queda byte a byte igual.
  - Límites conocidos: la retirada deja el fichero con un único salto de línea final, y una tabla del equipo con la cabecera exacta de Axiom se sigue adoptando.
  - De paso, `c0d57454` regenera la golden de Gemini, desfasada desde `2f4aed89` (T14).
  - Comprobación: build, vet, gofmt y deadcode ok; `agents`, `components` (salvo filemerge, fallo conocido) y `skillregistry` ok; `internal/cli` en tandas (`Workspace|Scope|Uninstall|Skills|Index|Registry|Agents`, `Pi|OpenClaw`) ok. El orquestador repitió `components`, `skillregistry` y las pruebas nuevas de `internal/cli`: ok.
- [x] **T18 · Restos de `uninstall --scope global` y `doctor` (H3)** (ruta: delegada; disparador: 2+ ficheros no triviales). Primero se compara con la base `7940a5cf` para separar lo heredado de lo nuevo. Después, uninstall global retira lo que tenga la forma de Axiom en:
  - `~/.config/kilo/opencode.json` y `~/.config/kilo/plugins/*.ts`;
  - `~/.kiro/steering/axiom.md`;
  - `~/.codex/config.toml` (`mcp_servers.context7`, `features.multi_agent` y `agents.*`).

  `doctor` detecta esos restos con el agente correcto (Kilo, no OpenCode). De paso: no borrar `mcp.json` del usuario que ya existían antes, aunque queden vacíos.
  - Hecho: `bfe3d6a7` (uninstall), `a95fe9bf` (doctor) y `f1fd692b` (docs); +745/−54 en 9 ficheros.
  - Heredado de la base `7940a5cf`, no introducido por la rama:
    - En global, el bloque SDD solo se aplicaba a OpenCode, no a Kilo, y Kilo usa `mcp`, no `mcpServers`.
    - El steering de Kiro solo perdía sus secciones con marcadores.
    - De Codex solo se retiraba `mcp_servers.engram`.
  - Criterio: se retira solo lo que tiene la forma y los valores de Axiom.
    - Kilo: agentes con `__managed_by`, MCP `engram` y `context7`, overlay, `share` y plugins con nombre gestionado. El fichero se borra si queda vacío.
    - Kiro: `axiom.md` se borra si solo queda la cabecera o la persona. Si no, se conserva con aviso.
    - Codex: `context7` con su URL, `features.multi_agent=true`, `agents.max_threads=4` y `agents.max_depth=2`.
    - `mcp.json`: si queda vacío, solo se borra cuando el servidor retirado tenía la forma de Axiom; si no, queda `{}`.
  - `doctor` delega en `uninstall.GlobalAgentLeftovers` y rotula cada resto con su agente.
  - Comprobación: build, vet, gofmt y deadcode ok; `uninstall` y `agents` ok; `internal/cli` `Uninstall|Doctor|Footprint` ok. `Workspace|Scope` agota los 8 minutos por las pruebas de review; sin ellas, ok (142 pruebas). Repetido por el orquestador: `TestUninstallGlobal|Doctor` y `uninstall` ok.
  - Abierto:
    - Un `mcp.json` vacío que ya existía antes es indistinguible del que crea Axiom; habría que registrar en el estado qué ficheros crea el install.
    - Siguen sin retirarse `~/.codex/sdd-*.config.toml`, `model`/`model_reasoning_effort` de Codex y `~/.claude.json`, fuera del alcance de H3.
- [x] **V1 · Fase 5b**: repetir el ciclo de V1 de la fase 5. Install y uninstall en workspace y en global con HOME temporal. Con OpenClaw y Pi seleccionados en workspace, HOME queda intacto.
  - **Hecha el 2026-11-02, sin hallazgos nuevos relevantes.** Arnés temporal en proceso en `internal/cli`, borrado después; `git status` limpio sobre `b6e6c6d1`.
    - **Workspace con los 10 agentes:** install y tres sync. HOME, APPDATA, LOCALAPPDATA y XDG sin cambios. Pi y OpenClaw salen con su aviso y no se lanza ningún comando. No aparecen `SOUL.md`, `.pi/` ni `.openclaw/`, y el segundo sync no cambia nada.
    - **`AGENTS.md` del equipo** (lista y tabla propia): la sección se conserva y el índice va aparte. Tras el uninstall, el fichero queda byte a byte igual.
    - **Uninstall en workspace:** 0 ficheros creados y 0 borrados, 6 solo reformateados. `.git/info/exclude` queda restaurado y la operación es idempotente.
    - **Global:**
      - Install, `doctor`, uninstall y `doctor`: no quedan restos de Kilo, Kiro ni Codex, y el contenido del usuario se conserva.
      - `doctor` informa de 6 restos antes del uninstall y de 0 después.
      - Quedan `~/.claude.json` con `{}` (abierto en T18) y cinco carpetas vacías (cosmético).
    - **Hooks en un repo ajeno:** salen con 0 y no cambian nada; el control positivo con `axiom.yaml` bloquea.
    - **Paquetes:** build, vet y deadcode ok; `agents`, `components` (salvo filemerge), `skillregistry` y `state` ok; `internal/cli` en cinco tandas ok; `internal/app` solo con el fallo de entorno conocido.
    - **Tiempo:** la tanda `Workspace|Scope` con las pruebas de review supera los 8 minutos. El orquestador la repitió entera en segundo plano, con un límite de 25 minutos: ok (580 s).

## Fase 5c · CI de la PR #217

La CI de Linux (`ubuntu-latest`, `go test ./...`) de la PR IGutierrezZ/axiom#217 falla en `internal/cli` con dos pruebas, ambas por esta rama. Evidencia: log de la CI (`TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` y `TestSyncWithoutChangesLeavesNoNewBackup`, `FAIL internal/cli 289.566s`).

- [x] **T19 · Rechazos nuevos sin salida nombrada** (ruta: delegada; disparador: 2+ ficheros y lectura previa del ratchet). La prueba `TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` marcaba cinco rechazos introducidos por la rama: `uninstall.go` (`--purge-hub requiere --purge-state`, `--purge-state solo se admite con --all`, `resolve workspace directory`), `validate.go` (agente no disponible) y `workspace_agents.go` (ningún agente admite workspace). El ratchet solo reconoce como resolución un `axiom <verbo>` en el texto (o una anotación `refusal:by-design`); los cinco tienen salida real, así que cada mensaje la nombra con su comando (`axiom uninstall --all --purge-state --purge-hub`, `axiom uninstall --all --purge-state`, `axiom uninstall`, `axiom install --agent claude-code`, `axiom install --scope global`). Sin anotaciones, sin tocar la línea base ni techos.
  - Hecho: `3c32d523`, +5/−5 en 3 ficheros. Los textos antiguos no los asertaba ninguna prueba con su literal (las pruebas existentes buscan `--scope global` y `no está disponible de momento`, que se conservan).
  - Comprobación: `TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` y `TestRefusalRatchet*`, `TestErrNoWorkspaceAgents*`, `TestAsInstallAgentIDs*` ok.
  - Corrección (2026-11-02, inline): la CI marcó `TestNoGoSourceBindsALiteralRuntimeIdentityIntoPrintedCommands` porque el mensaje de `validate.go` fijaba `--agent claude-code` (issue #2440: solo quien llama sabe qué runtime ejecuta). Pasa a `axiom install --agent <agente>`. Comprobación: esa prueba, el ratchet y `Validate` ok en local.
- [x] **T20 · `sync` no idempotente en Linux** (ruta: delegada; disparador: diagnóstico con reproducción en WSL). `TestSyncWithoutChangesLeavesNoNewBackup` devolvía `ChangedFiles = [<raíz>/.claude/CLAUDE.md]` en el sync #1.
  - Causa raíz: install sondea la versión de Engram con el binario que resolvió (`resolveEngramInstalledPath`, en la prueba `/usr/local/bin/engram`) y la pasa a `engram.InjectOptions.Version`; sync sondeaba el nombre a secas (`resolveEngramVersion("engram")`). Si el binario resuelto responde con una versión ≥ 1.4.0 y el nombre a secas no (en las pruebas, `TestMain` lo fija a error), install escribe la sección slim del protocolo de Engram en `CLAUDE.md` y el primer sync la reinflaba a la full. Es el mismo defecto del bug #1824 por otra vía.
  - Reproducción en Linux (WSL Ubuntu-24.04, sin Go: pruebas compiladas con `GOOS=linux go test -exec` y ejecutadas en WSL): la prueba pasa sin engram y falla con el mismo mensaje de la CI al poner un `/usr/local/bin/engram` que imprime `engram 1.18.0`. No se pudo comprobar qué `engram` había en el runner de la CI; la reproducción demuestra el mecanismo, no su origen en el runner.
  - Corrección: `resolveSyncEngramCommand` (`run.go`) devuelve el binario que resuelve PATH (`cmdLookPath("engram")`) y, si no hay, `engram`; sync lo usa para la versión. En producción es el mismo binario que el nombre a secas, así que solo cambia que install y sync comparten sondeo. La prueba `TestSyncProbesTheSameEngramCommandAsInstall` (rojo antes, verde después, también en Windows) fija la regla sin depender de ningún binario real. La prueba original no era incorrecta y no se tocó.
  - Hecho: `12a3f30a`, +80/−1 en 3 ficheros. Con la corrección, en WSL: la prueba pasa con y sin el `engram` falso.

## Criterios de aceptación

- `axiom setup` y `axiom sync` en workspace dejan `HOME` idéntico byte a byte, salvo la huella declarada en T9.
- En un repositorio ajeno, los hooks de Axiom instalados no crean ni modifican ningún fichero ni escriben en Engram.
- Ninguna ejecución escribe modos de autoaprobación. El `deny` del usuario se conserva y la inyección es idempotente.
- Con una instalación solo en workspace de Claude Code, los subagentes `sdd-*` cargan sus skills y el MCP de Engram está disponible.
- `axiom uninstall --scope workspace` revierte lo instalado en el workspace.
- La documentación describe el comportamiento real.

## Comprobaciones aplicables

- `go test ./internal/skillregistry/... ./internal/components/... ./internal/cli/... -count=1`, acotado a los paquetes tocados en cada tarea.
- `go vet ./...` y `go build ./...`.
- `bash scripts/deadcode-ratchet.sh`.
- Comprobación manual con `HOME` temporal (V1).

## Modo

- **TDD estricto: desactivado.** Fuente: `openspec/config.yaml` (`strict_tdd: false`). Runner: `go test`. Se ejecutan las pruebas funcionales de cada tarea.
- **RDD: desactivado** (`axiom review mode status`: off, decidido por defecto).
- **Entrega:** previsión de unas 2.300 líneas de autor en total (F1 ~400, F2 ~450, F3 ~600, F4 ~550, F5 ~300). Supera el presupuesto de ~400 líneas.
  - Estrategia `single-pr` con la etiqueta `size:exception`, elegida por el usuario el 2026-10-29. Toda la feature va en una PR desde `fix/huella-usuario-workspace`.
  - Se mantienen los commits por unidad de trabajo (al menos uno por tarea) para que la revisión pueda seguirse commit a commit.
- Push, PR y merge: decisión del usuario.

## Progreso

- 2026-10-29: plan registrado en este documento, con 14 tareas, R1 y V1 (`851c7eef`). Sin implementación.
- 2026-10-29: D3 resuelta por el usuario: solo `axiom.yaml`, directo o mediante el puntero `.axiom-workspace`. D1, D2 y D4 aceptadas según la recomendación.
- 2026-10-29: la migración (D4) no tiene tarea propia; queda cubierta por T13 (desinstalación global) y T9 (`doctor` con restos globales).
- 2026-10-29: entrega elegida por el usuario: una sola PR con la etiqueta `size:exception`.
- 2026-10-29: la base real del worktree es `origin/main` `7940a5cf` (incluye #213), no `c780061a`.
- 2026-10-29: fase 1 implementada (T1 `9941eb18`, T2 `8eb668f4`, T3 `2f05d8b3`, T3b `386e0cc0`), unas 2.400 líneas, la mayoría pruebas. V1 de la fase 1 (comprobación manual con `HOME` temporal) sigue pendiente.

- 2026-10-29: fase 2 implementada (T4 `75faa77a`, T4b `869d3556`, T5 `b68c3008`, T6 `f3bcc8dd`, T7 `df59151f`), unas 1.900 líneas, la mayoría pruebas. V1 de las fases 1 y 2 sigue pendiente.
- 2026-10-29: con `AXIOM_STATE_DIR` exportado fallan en la base `internal/agents`, `internal/components/communitytool` e `internal/components/uninstall`; se ejecutan sin esa variable.

- 2026-10-30: V1 de las fases 1 y 2 hecha: hooks inertes en repos ajenos y comprobaciones de T2–T6 en verde. La huella que queda en `HOME` está asignada a T8, T9 y T11.

- 2026-10-30: fase 3 implementada (T8 `3be1d092`; T9 `1ab483fc`, `c1c5eb34`; T10 `acfff69a`, `c9cadff0`; T11 `92d1ec65`, `55c06cb8`), unas 3.770 líneas, más de la mitad pruebas. Más `e853c651`, que corrige dos pruebas de `cmd/axiom` rotas por T1. V1 de la fase 3 pendiente.

- 2026-10-30: V1 de la fase 3 hecha: `HOME` intacto en workspace con el preset por defecto. Tres correcciones nuevas (T11b–T11d). La fase 4 arranca en otra sesión, en `fix/huella-usuario-workspace-f4`.

- 2026-10-30: fase 3b implementada (T11b `74bc8429`, T11c `0a8e3ce7`, T11d `61558d41`), unas 510 líneas.

- 2026-11-01: fase 4 integrada (`7689cd4c`, sin conflictos; +7.821/−199 en 93 ficheros). Verificación del resultado combinado con la fase 3b: build, vet, gofmt y deadcode-ratchet ok; `internal/cli` en cinco tandas (`Workspace|Scope|ComponentPath|TUIInstall`, `TestRunSync|TestSync`, `TestRunInstall|TestInstall|Permission`, `Doctor|Engram|Rollback|Restore|Backup`, `OpenCode|Kilo|Codex|Gemini|VSCode|Skills|Background`) ok; `internal/app` ok entero; `agents`, `state`, `components`, `skillregistry`, `backup`, `opencode`, `workspace`, `cmd/axiom` y `tui` ok salvo los fallos conocidos de la base (filemerge, sockets de backup y la prueba de la TUI dependiente del orden).

- 2026-11-01: fase 5 implementada (T13, T13b `825ee3c0`, T14) y V1 de las fases 4 y 5 hecha, con cuatro hallazgos (H1–H4).

- 2026-11-01: el usuario autoriza la fase 5b; T15–T18 registradas.

- 2026-11-02: fase 5b implementada (T15 y T16 `22478be0`, T17 `851d1f22`, golden de Gemini `c0d57454`, T18 `bfe3d6a7`, `a95fe9bf` y `f1fd692b`). La V1 de la fase 5b se hizo sin hallazgos nuevos relevantes.

- 2026-11-02: fase 5c implementada tras el fallo de la CI de la PR #217: T19 `3c32d523` (rechazos de uninstall y workspace con su comando) y T20 `12a3f30a` (sync sondea la versión de Engram como install). Pendiente de que la CI de la PR lo confirme.

## Siguiente paso

Subir la fase 5c y comprobar la CI de la PR #217 (decisión del usuario). Después, abrir/actualizar la PR desde `fix/huella-usuario-workspace` con la etiqueta `size:exception`, cuando el usuario lo decida. Abiertos de la fase 5b: el `mcp.json` vacío que ya tenía el usuario y que uninstall borra; y los restos globales fuera de H3 (`~/.codex/sdd-*.config.toml`, `model` de Codex, `~/.claude.json` y carpetas vacías). Abiertos para decidir: Antigravity en workspace (T12h, aplazado), Qwen y Hermes con fichero raíz, y los temas sueltos de la fase 3 (manifiesto de Pi, caché `model-variants`, CodeGraph sin ubicación de proyecto, lanzadores de OpenCode sin aviso).
