# Visión de producto frente al estado real — auditoría y hoja de ruta

> Auditoría de solo lectura hecha el 2026-11-03 sobre `main` en `f7158af0`. Se leyó el
> código y la documentación; no se compiló ni se ejecutó nada. Las conclusiones clave se
> contrastaron directamente en el código (se citan con `fichero:línea`). Este documento es
> la referencia para cerrar la brecha paso a paso: cada paso de la hoja de ruta (§5) se
> abre como su propio cambio y actualiza aquí su estado al cerrarse.

## 1. La visión

| Id | Objetivo |
|---|---|
| O1 | Instalar Axiom en cualquier equipo con **un solo comando de terminal**, contra el repositorio público y sin clonarlo. |
| O2 | La UI **detecta actualizaciones**, avisa de que no estás en la última y, **con un clic, actualiza y sincroniza**. |
| O3 | Abrir la UI ejecuta **doctor sobre el proyecto abierto**; al cambiar de proyecto, se vuelve a ejecutar sobre el nuevo. |
| O4 | Al dar de alta un proyecto, la **configuración se elige y se valida en la UI**: topología y ruta global si es multirepo, roles, repositorio de cada rol, si hay review y qué adaptadores instalar. La instalación es **siempre en modo workspace**. |
| O5 | Debe funcionar en **VS Code + Copilot, VS Code + extensión de Claude Code, Claude Code Desktop, Claude Code CLI, Copilot CLI y OpenCode**; en cada uno, si se eligen modelos por fase, se elige **proveedor, modelo y razonamiento**. |
| O6 | Si el repositorio de specs ya trae contenido, Axiom **lo analiza** para ver qué adaptar; si tiene **spec viva y contexto técnico**, los agentes **saben leerlos e integrar en ellos al archivar**. Los incrementos y bugs siguen el formato de trabajo de Axiom (OpenSpec u ODD), pero la integración es sobre la spec viva del proyecto. |

## 2. Veredicto por objetivo

Leyenda: ✅ cumple · 🟡 parcial · ❌ no cumple.

### O1 — Instalación con un comando · 🟡

| Hueco | Evidencia |
|---|---|
| G1.1 La última release publicada es `v3.5.0` (2026-09-21); `main` lleva mucho trabajo posterior. Instalar «la última» instala algo desfasado. | `gh release list` |
| G1.2 No hay binario de Windows. Es una **retención deliberada** hasta tener firma Authenticode pública, no un olvido. | `.goreleaser.yaml` (solo linux/darwin); `internal/update/instructions.go:10` |
| G1.3 `install.ps1` solo instala con `go install`: un comando, sin clonar, pero exige Go 1.25.10+. | `scripts/install.ps1:67-80` |
| G1.4 `install.sh` intenta primero Homebrew contra un tap de otra organización. | `scripts/install.sh:20-21,66`; `internal/update/upgrade/strategy.go:437` |
| G1.5 `install.sh` exige `git` aunque no lo usa. | `scripts/install.sh:24-26` |
| G1.6 El README y el quickstart solo documentan `git clone` + `go install ./cmd/axiom`; no aparecen las líneas de `install.sh` ni `irm … \| iex`. | `README.md:143-148`; `docs/quickstart.md:13-20` |
| G1.7 Residuos `gentle-ai` en nombres de assets, variables `GENTLE_AI_*` y en el registro de actualización (corregido en tiempo de ejecución por `cmd/axiom/main.go:38-48`). | `.goreleaser.yaml:49,83`; `internal/update/registry.go:20-21` |

### O2 — Aviso de actualización y actualización con un clic · ❌

| Hueco | Evidencia |
|---|---|
| G2.1 El dashboard no comprueba versión: no importa `internal/update`, no tiene endpoint ni aviso. Solo avisan la TUI y el arranque de la CLI. | `internal/dashboard/server.go:67-102`; `internal/tui/model.go:1055`; `internal/app/selfupdate.go:116` |
| G2.2 Existe el botón y la cadena upgrade → sync, pero la sync **se salta siempre** que el binario se actualiza (`RestartRequired`). | `internal/app/upgrade_report.go:66-73,200`; `internal/dashboard/service.go:1344-1439` |
| G2.3 En Windows no hay reemplazo del ejecutable en uso (el dashboard es ese mismo proceso); la vía de respaldo hace `os.Rename` sobre el exe activo. | `internal/update/upgrade/download.go:415`; `executor.go:714-722` |

### O3 — Doctor por proyecto en la UI · 🟡

| Hueco | Evidencia |
|---|---|
| G3.1 El dashboard ejecuta el doctor de la CLI y **descarta su salida**; `Healthy` es casi siempre `true`. | `internal/dashboard/service.go:1202-1211,1273` |
| G3.2 El doctor resuelve el proyecto con el `cwd` del servidor, no con el proyecto abierto. | `internal/cli/install_state_lock.go:45-55` |
| G3.3 Al cambiar de proyecto sí se vuelve a pedir, pero solo cambia el check `workspace-axiom.yaml`; no se invalida el resultado anterior. | `internal/dashboard/assets/js/projects.js:64-69`; `internal/dashboard/assets/js/main.js:61,99` |
| G3.4 Ningún doctor comprueba drift de assets gestionados del proyecto, adaptadores, topología multirepo, repos de rol ni repositorio de specs. | `internal/cli/doctor.go`; `service.go:1202-1277` |
| G3.5 El doctor de la CLI produce texto; `doctor.Report`/`Result` existen pero el dashboard no los usa. | `internal/doctor/doctor.go:114-124` |

### O4 — Alta de proyecto desde la UI · 🟡

| Hueco | Evidencia |
|---|---|
| G4.1 `hub.Init` escribe `axiom.yaml` sin validar lo que genera; la UI solo comprueba que la ruta no esté vacía. | `internal/hub/init.go:108,253`; `internal/dashboard/server.go:254-277` |
| G4.2 No hay campo de raíz ni de `specs_repository` para multirepo: se escribe siempre `root: "."` y `specs_repository: openspec`. El puntero `.axiom-workspace` solo se monta a mano. | `internal/hub/init.go`; `internal/workspace/loader.go` |
| G4.3 Roles y repos por rol se piden pero no se validan (ni claves duplicadas); `tech` no se edita; tras el alta es solo lectura. | `projects.js:394-440`; `internal/hub/init.go:380-405` |
| G4.4 Defecto: el dashboard fija `GatePolicy: "blocking"` e ignora el valor real del rol. | `internal/dashboard/service.go:174` |
| G4.5 La review (RDD) es un interruptor global de máquina, no por proyecto, y no está en la UI web. | `internal/state/state.go:142-155`; `internal/cli/review_mode.go:45` |
| G4.6 La UI web no elige ni instala adaptadores; `axiom.yaml` no los declara. `/api/ecosystem/sync` solo sincroniza lo ya instalado. | `internal/dashboard/service.go:1302`; `internal/cli/install.go:61` |
| G4.7 El modo workspace es el ámbito por defecto (✅), con huellas fuera de él: backup del dashboard en HOME, `engram setup` manual, binarios, registro del hub y `rdd_mode`. | `internal/cli/scope.go:34-43`; `internal/dashboard/service.go:1500`; `odd/tasks/huella-usuario-workspace.md` |

### O5 — Adaptadores y modelo por fase · 🟡

| Entorno | Adaptador | Modelo por fase | Proveedor | Razonamiento |
|---|---|---|---|---|
| VS Code + Copilot | ✅ `vscode` | ❌ sin subagentes | ❌ | ❌ |
| VS Code + extensión Claude · Claude Desktop · Claude CLI | ✅ un único `claude-code` | ✅ `model:` en frontmatter | 🟡 solo alias Anthropic | ✅ `effort:` |
| Copilot CLI | ❌ no existe | ❌ | ❌ | ❌ |
| OpenCode | ✅ | ✅ | ✅ `provider/model` | ✅ `variant` |

| Hueco | Evidencia |
|---|---|
| G5.1 Copilot CLI no tiene adaptador; solo hereda `.github/instructions`, `.github/skills` y el `.mcp.json` de Claude (si Claude está instalado). | `docs/agents.md:61,209` |
| G5.2 VS Code + Copilot no genera custom agents (`.agent.md`), así que no hay modelo por fase. | `internal/agents/vscode/adapter.go:174-176` |
| G5.3 Claude Desktop o la extensión sin el CLI ni `~/.claude` no se autodetectan (funciona con `--agent claude-code`). No está contrastado que lean `.claude/` y `.mcp.json` del workspace. | `internal/agents/claude/adapter.go:50-60` |
| G5.4 La selección de modelos por fase solo se hace en la TUI (o `--profile` en la CLI, sin esfuerzo); la UI web solo lee la de OpenCode. | `internal/dashboard/service.go:1577-1608`; `docs/usage.md:368-371` |
| G5.5 Codex en workspace pierde los perfiles `sdd-*` (fuera del alcance de la visión, se anota). | `internal/agents/engram/codex_workspace.go:91` |

### O6 — Repositorio de specs existente y spec viva · 🟡

| Hueco | Evidencia |
|---|---|
| G6.1 No hay paso de intake que analice un repositorio de specs y diga qué adaptar. La adopción es manual. | `docs/manual-de-inicio.md:432-478` |
| G6.2 La detección solo reconoce `openspec/` o `.openspec/`; no busca otras specs, ADR ni documentación de arquitectura. | `internal/hub/detector.go:177-190`; `internal/hub/types.go` |
| G6.3 `axiom init` nunca propone bloques `baseline:` ni `technical_context:`; `axiom.example.yaml` no los incluye. | `internal/hub/init.go:360-420`; `axiom.example.yaml` |
| G6.4 La integración al archivar solo admite Markdown con un fichero por dominio dentro del workspace. | `internal/workspace/declarations.go:49-75,505-517`; `internal/cli/sdd_archive_compose.go:215-260` |
| G6.5 El trabajo ODD nunca integra en la spec viva. | `openspec/specs/project-technical-context/spec.md:68` |
| G6.6 Las instrucciones ODD no dan localizador del contexto técnico mantenido ni verifican su actualización. | `internal/agentguidance/routing.go:63` |
| G6.7 La integración en prosa (`directed`) y el contexto mantenido dependen del modelo; `verify` lo detecta después. | `docs/manual-de-inicio.md:476-478` |

## 3. Lo que ya cumple

- Los agentes SDD leen la spec viva y el contexto técnico declarados: `axiom sdd status --json` proyecta `baseline` y `technicalContext`, y explore, propose, spec, design, apply y verify los usan (`internal/sddstatus/workspace_declarations.go:32-96`).
- El archivado SDD integra en la spec viva declarada (no crea una segunda) y bloquea si falta `context-impact.md` con contexto mantenido (`internal/specbaseline/*`, `workspace_declarations.go:127`).
- El modo workspace es el ámbito por defecto de install, sync y TUI.
- Claude Code y OpenCode aplican el modelo por fase en lo que generan.

## 4. Decisiones de producto pendientes

Cada una bloquea solo el paso que la cita.

| Id | Pregunta | Paso |
|---|---|---|
| D1 | Windows: ¿firmar con Authenticode y publicar binario, o asumir `go install` como canal oficial en Windows? Las condiciones para restaurar el binario ya están fijadas en `docs/release-signing.md` («Windows distribution restoration gate»). **Resuelta (2026-11-04):** `go install` es el canal oficial provisional en Windows; la firma con Azure Artifact Signing a nombre de Hiberus queda como trabajo posterior que restaurará el binario. | P1 |
| D2 | ¿La review (RDD) pasa a ser configuración por proyecto en `axiom.yaml`, o sigue siendo un interruptor de máquina? (El ODD `huella-usuario-workspace` descartó el RDD por clon el 2026-10-30.) **Resuelta (2026-11-10):** ni `axiom.yaml` ni interruptor de máquina. El RDD es una preferencia de cada usuario en cada proyecto, sin versionar, en la entrada del proyecto de `~/.axiom/workspaces.json`. Es el mismo sitio de las preferencias personales de worktree (REQ-25.29) y lo comparten todos los worktrees del proyecto. Se revoca la decisión del 2026-10-30. El interruptor global queda como valor por defecto cuando el proyecto no declara nada. Nada versionado puede activar ni imponer la review. Se implementó en la T6 de `odd/tasks/alta-proyecto-ui.md` (PR #243), que sustituyó a la T21 de `odd/tasks/upstream-revision-nativa.md`. | P4 |
| D3 | ¿Hay que distinguir extensión de VS Code, Desktop y CLI de Claude Code, o basta con un adaptador común bien detectado? | P5 |
| D4 | ¿Los bugs e incrementos ODD integran en la spec viva? Si sí, ¿con delta, solo registro, o exclusión explícita? | P6 |
| D5 | ¿Qué formatos de spec viva se admiten como destino además de Markdown por dominio (carpeta por requisito, otros)? | P6 |
| D6 | La spec viva debe cumplir **ISO/IEC 33000**. Por concretar: la familia 330xx evalúa la **capacidad de procesos**, no el formato de un documento. Hay que fijar qué modelo de referencia de procesos se evalúa, qué nivel de capacidad se persigue y qué evidencias deben dejar spec, cambios y archivado (trazabilidad, productos de trabajo, revisiones). Si lo que se busca es la calidad de la especificación de requisitos en sí, la norma de referencia sería ISO/IEC/IEEE 29148. | P6 |

## 5. Hoja de ruta paso a paso

Orden por dependencia: sin instalación y actualización fiables nadie llega a probar lo
demás; el doctor por proyecto es la base que el alta y los adaptadores reutilizan para
validar. Cada paso es un cambio propio (ODD o SDD según su alcance) y al cerrarse marca
aquí su estado con la PR que lo cerró.

| Paso | Objetivo | Huecos | Criterio de aceptación | Depende de | Estado |
|---|---|---|---|---|---|
| P1 Distribución | O1 | G1.1–G1.7 | Release al día desde `main`; una línea documentada por sistema (`curl … \| bash`, `irm … \| iex`) que instala sin clonar; sin tap ajeno ni `git` innecesario; en Windows, `irm … \| iex` instala con `go install` del módulo público y explica cómo obtener Go si falta (D1). | — | **Hecho (2026-11-04):** PR #220 y release `v3.6.0`. De G1.7 se corrigió el registro de actualización; los nombres `gentle-ai-*` de assets y `GENTLE_AI_*` se conservan porque son contratos. |
| P2 Actualización desde la UI | O2 | G2.1–G2.3 | La UI muestra «hay una versión nueva»; un clic actualiza, relanza el dashboard con el binario nuevo y ejecuta la sync del workspace; funciona en Windows con el exe en uso. | P1 | **Hecho (2026-11-05):** PR #225 y release `v3.7.0` (integrada con la PWA de #222 y el directorio neutro de #223). Falta la prueba manual del relanzamiento en Windows con el exe en uso, que solo puede hacerse actualizando desde `v3.7.0` a la release siguiente. |
| P3 Doctor por proyecto | O3 | G3.1–G3.5 | Doctor estructurado y parametrizado por raíz; la UI muestra sus checks reales al abrir y al cambiar de proyecto; añade checks de assets, adaptadores, topología, repos de rol y repo de specs. | — | **Hecho (2026-11-10):** PR #234. `axiom doctor` en la CLI añade las líneas `project:*` dentro de un proyecto configurado. Quedan como seguimiento: el detalle de los checks sale en inglés en la UI, un proyecto sin `axiom.yaml` ya no recibe el aviso de «no configurado» y la ausencia del repo de specs puede notificarse dos veces. |
| P4 Alta de proyecto en la UI | O4 | G4.1–G4.7 | Alta validada con `workspace.Validate` antes de escribir; campos de multirepo; roles y repos validados; elección e instalación de adaptadores en workspace; review según D2; corregido G4.4. | P3, D2 | Pendiente |
| P5 Adaptadores y modelos | O5 | G5.1–G5.4 | Adaptador de Copilot CLI; custom agents de VS Code Copilot con `model`; detección de Claude sin CLI; selección de proveedor, modelo y razonamiento por fase desde la UI para cada adaptador que lo soporte. | P4, D3 | Pendiente |
| P6 Spec viva existente | O6 | G6.1–G6.7 | Comando y flujo de UI de intake que analiza el repo de specs y propone `baseline` y `technical_context`; `axiom.example.yaml` documentado; formatos según D5; ODD según D4 con localizador de contexto; la spec viva y su ciclo dejan las evidencias que exija ISO/IEC 33000 según D6. | P4, D4, D5, D6 | Pendiente |

P3 y P6 no dependen entre sí y pueden avanzar en paralelo si hace falta.
