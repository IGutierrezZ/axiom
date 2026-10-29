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

- [ ] **T4 · Rutas de skills resolubles por ámbito en subagentes y prompts** (ruta: delegada).
  - Sustituir `~/.claude/skills/...` por un marcador que se resuelva al inyectar: en workspace, una ruta relativa al proyecto (`.claude/skills/...`); en global, `~/.claude/skills/...`.
  - Alcance: `internal/assets/claude/agents/*.md`, la referencia perezosa a `_shared/sdd-orchestrator-workflow.md` y los equivalentes de Cursor, Kimi y Kiro.
- [ ] **T5 · MCP de Engram en workspace en una ruta que el agente lee** (ruta: delegada).
  - Claude: fusionar en `<ws>/.mcp.json`, como context7.
  - Revisar `MCPConfigPath` del resto de adaptadores con raíz en el workspace.
- [ ] **T6 · Raíz del proyecto en lugar de la carpeta actual** (ruta: delegada).
  - Generalizar la búsqueda de la raíz del proyecto (`findProjectRoot`) para install, setup y sync.
  - Desde una subcarpeta no se crea `<subcarpeta>/.claude`.
- [ ] **T7 · La TUI respeta el ámbito** (ruta: en línea; un fichero más su prueba).
  - Eliminar `ScopeGlobal` fijo en `internal/cli/run.go:1907`. Workspace por defecto y global solo de forma explícita.

### Fase 3 · Huella de máquina mínima (depende de D1, D2)

- [ ] **T8 · Preset por defecto según D1** (ruta: en línea en `internal/model/presets.go` y sus pruebas).
- [ ] **T9 · Inventario declarado de la huella de máquina** (ruta: delegada).
  - Dejar a nivel de usuario solo el binario `axiom`, el binario de Engram y el registro del hub.
  - El resto de pasos que escriben en home se omiten con `--scope workspace`:
    - copia de skills de compatibilidad en `~/.agents/skills`;
    - limpieza de skills retirados en home;
    - refresco de plugins de OpenCode;
    - `engram setup <agente>`;
    - CodeGraph `--location global`;
    - configuración de GGA.
  - `doctor` lista esa huella.
- [ ] **T10 · Estado por proyecto según D2** (ruta: delegada).
  - Selección, persona y asignación de modelos por workspace, con lectura de compatibilidad del estado global.
  - `rdd_mode`: permitir activarlo por clone, además del off actual.
- [ ] **T11 · Una única raíz de estado** (ruta: delegada).
  - Backups, `bin`, caché, dashboard y doctor resuelven `system.AxiomDir` (respetan `AXIOM_STATE_DIR`) en lugar de `~/.axiom` fijo.

### Fase 4 · Adaptadores sin modo workspace (depende de R1)

- [ ] **R1 · Investigación con fuentes primarias** (ruta: delegada, solo lectura).
  - Qué configuración de proyecto carga cada agente:
    - Codex: `.codex/` y `AGENTS.md` en la raíz.
    - VS Code Copilot: `.github/copilot-instructions.md`, `.github/prompts`, `.vscode/mcp.json`.
    - Windsurf: `.windsurf/rules`.
    - Trae y Antigravity.
  - Verificar además si OpenCode carga `<ws>/.config/opencode` o solo `opencode.json` y `.opencode/`, y lo equivalente en Gemini, Qwen, Kimi y Hermes.
- [ ] **T12 · Modo workspace por adaptador según R1** (ruta: delegada; una rebanada por grupo de adaptadores).

### Fase 5 · Reversibilidad y documentación

- [ ] **T13 · `axiom uninstall` con ámbito** (ruta: delegada).
  - `--scope workspace` revierte `<ws>/...`.
  - Hallazgo de T2: hoy la desinstalación borra el objeto `permissions` completo de Claude, incluida la lista `deny` del usuario. También borra `defaultApprovalMode` de Gemini y `chat.tools.autoApprove` de VS Code, que ahora son valores del usuario. Debe retirar solo lo que Axiom añadió.
  - El modo global además retira las entradas de PATH gestionadas (directorio de Engram, `~/bin`, `~/.axiom/bin`) y ofrece purgar `~/.axiom`.
- [ ] **T14 · Documentación alineada** (ruta: delegada).
  - `docs/usage.md:248`.
  - `docs/manual-de-inicio.md:204,239`.
  - Redacción de REQ-25.22 en la spec viva.
  - `docs/components.md`, `docs/platforms.md` y las rutas de `docs/opencode-profiles.md`.
  - `AXIOM_NO_SELF_UPDATE` en `scripts/install.ps1:153`, alineado con `GENTLE_AI_NO_SELF_UPDATE`.
- [ ] **V1 · Verificación de extremo a extremo** (ruta: delegada; al cierre de cada fase).
  - Con `HOME` temporal: hash de `HOME` antes y después de `setup` y `sync` en workspace.
  - Sesión simulada de hooks en un repo ajeno: ningún fichero creado ni modificado.

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

## Siguiente paso

Autorización para la fase 2 (T4–T7).
