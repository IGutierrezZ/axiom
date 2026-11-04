# ODD: Absorción upstream — adaptadores de agentes (OpenCode y Copilot CLI)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/adaptadores-agentes-upstream.md`.
> Espejo de recuperación en Engram: topic `odd/adaptadores-agentes-upstream/tasks`, proyecto `axiom`.
> Estado: **alcance validado por el usuario (2026-11-04), con T5 como primera rebanada de P5. No se ha implementado nada.**

## Objetivo

Absorber de upstream gentle-ai (revisado hasta `origin/main` 3ea08cdf, después de v4.3.1) los arreglos y capacidades de adaptadores de agentes que afectan a los agentes que el equipo usa de verdad: **OpenCode**, **Claude Code**, **Copilot CLI** y **VS Code con Copilot**.

## Problema y porqué

- El overlay SDD de OpenCode escribe la clave `__managed_by` en cada agente. Los proveedores con validación estricta del esquema (opencode-go, Fireworks, NIM) rechazan la configuración (upstream issue #4471).
- El catálogo de modelos de OpenCode se lee en buffer, con techo de 1 MiB y solo con `models --verbose`. Con catálogos grandes o con OpenCode V2, el selector de modelos falla o se queda vacío sin explicar el motivo.
- El equipo usa Copilot CLI y Axiom no tiene adaptador para él. Solo existe el de VS Code Copilot.

## Fuera de alcance (decidido)

- **Agentes que el equipo no usa** (decisión del usuario, 2026-11-04): Pi (#4892, #5121, #5132, #5135, #5040, #4985), Antigravity (#5931-#5948, #5956), Gemini CLI (#5949-#5955, #5961), Codex `CODEX_HOME` (#5526, #5429), Kimi y Conductor (#5060).
- **Lo que ya cubren otros bloques de la absorción**, para no duplicarlo:
  - Bloque seguridad y datos:
    - modo de fichero y `ExistingFileMode` (#5014/#5024, 6c729f36, 209d8867);
    - reglas deny de Claude (#5867, #5629);
    - permisos de OpenCode (#5311).
  - Bloque arreglos de instalación, sync y TUI: Sonnet sin `xhigh` (#5639).
- **Aparcado:**
  - #3731: sonda de versión en el plugin de review de OpenCode. RDD está apagado por defecto. Se retomará si el equipo usa la revisión nativa en OpenCode.
  - Escritores JSONC con varios ficheros #5016 y #5053. Solo afectan con `opencode.jsonc` de proyecto y `opencode.json` global a la vez, y #5053 exige portar cinco helpers de `filemerge`.
  - El resto de #5115: gestor de SDK, sync parcial, TUI de consentimiento y review V2, unas 6000 líneas.
- **Descartado:**
  - #5028, que limpia reglas ligadas a la retirada de SDD de upstream.
  - La parte de #4998 que repone agentes borrados por esa retirada.
  - Cualquier cambio de telemetría.
  - El registro por `init()` y los imports `/v4`.
- **Prohibido:** tocar `docs/upstream-absorption-ledger.md` (decisión D4).

## Restricciones

- Axiom conserva SDD/OpenSpec. Upstream retiró SDD (e219644b2, #5297/#5299/#5308), así que **no se copian diffs a ciegas**: se porta la intención.
- Sin telemetría. Marca propia: binario `axiom`, marcadores `<!-- axiom:`.
- Módulo `github.com/IGutierrezZ/axiom/v3`: hay que reescribir los imports.
- No se compilan binarios sueltos, porque el antivirus borra `axiom.exe`. Toda la verificación se hace con `go test` en proceso.
- Se trabaja en el worktree `C:\repos\axiom-wt\adaptadores-agentes-upstream\axiom`, rama `inc/adaptadores-agentes-upstream`.

## Veredictos verificados

Evidencia completa en las notas compartidas de la sesión (`adaptadores-agentes-veredictos.md`). Tabla resumida:

| Upstream | Qué cambia | Estado en Axiom | Veredicto | Tamaño |
|---|---|---|---|---|
| #4998 / issue #4471 | El overlay deja de escribir `__managed_by` | Lo escriben `internal/assets/opencode/sdd-overlay-single.json` y `sdd-overlay-multi.json` (23 veces cada uno). Lo leen `internal/opencode/config.go:322` y `internal/components/uninstall/workspace.go:527` | ADAPTAR | S/M |
| #4043 | Catálogo en streaming, techo 16 MiB, errores veraces en el selector | `internal/opencode/catalog.go:17` (1 MiB) y `catalog.go:56` (`models --verbose`, en buffer) | ADAPTAR | M |
| #5842 | Aviso de catálogo degradado en el selector | `internal/tui/screens/model_picker.go` no tiene `CatalogError` | ADOPTAR | S |
| #5115 (solo catálogo) | Sonda `opencode --version`; en V2, `opencode api get /api/model` | No existe. Axiom ya detecta V1/V2 para los plugins (`internal/opencode/runtime.go:17-65`) | ADAPTAR (condicionado) | M |
| #5841 | V2 trunca la salida por tubería a 256 KiB; se captura en fichero temporal | No existe | ADAPTAR (condicionado) | S/M |
| #4896 | Diagnóstico de fallo de spawn en el plugin skill-registry | El plugin ya lanza `axiom` (`internal/assets/opencode/plugins/skill-registry.ts:60`, `plugins-v2/skill-registry.ts:23`) | ADAPTAR | S |
| #5052 (parte) | Limpieza de reglas sobre el fichero de settings efectivo | Hueco en `internal/cli/run.go:1024` (`stripLegacyTriggerRules` usa `adapter.SettingsPath`) | ADAPTAR | S |
| #5656 | Adaptador nuevo de Copilot CLI | No existe. VS Code Copilot en `internal/agents/vscode/adapter.go` | ADAPTAR | M |

### Sobre Copilot CLI frente a VS Code Copilot

La premisa «Copilot CLI es igual que VS Code Copilot» se ha verificado **solo en parte**:

- **Comparten** las skills en `~/.copilot/skills` (`vscode/adapter.go:96-99`), `AGENTS.md` y las instrucciones de repositorio `.github/copilot-instructions.md`.
- **Difieren** en lo demás:
  - Prompt global: VS Code usa `<VSCode User>/prompts/axiom.instructions.md`; el CLI, `<COPILOT_HOME|~/.copilot>/copilot-instructions.md`.
  - MCP: VS Code usa `mcp.json` con raíz `servers`; el CLI, `~/.copilot/mcp-config.json` con raíz `mcpServers` y campos `type` y `tools`.
  - Settings: el CLI usa `~/.copilot/settings.json`.
  - Detección e instalación: VS Code, binario `code` más extensión; el CLI, binario `copilot` instalado con npm `@github/copilot`.

Decisión: **adaptador nuevo y delgado** (`github-copilot-cli`), no una variante parametrizada del de VS Code.

## Tareas

Las tareas son unidades de trabajo de unas 400 líneas y van todas en un único PR (`exception-ok`). Cada tarea cierra con un commit de unidad de trabajo, con sus tests.

- [ ] **T1 · E1 — Overlay de OpenCode sin `__managed_by` (#4998/#4471 adaptado)** (~250-350 líneas)
  - Quitar el marcador de `sdd-overlay-single.json` y `sdd-overlay-multi.json` y regenerar `testdata/golden/sdd-opencode-multi-settings.golden`.
  - En sync, purgar el `__managed_by` residual de los agentes gestionados:
    - los de `managedOpenCodeAgentKeys` (`internal/opencode/config.go:335-340`);
    - los perfiles `sdd-orchestrator-<perfil>`.
    El merge JSON es profundo y no borra claves.
  - Al elegir entre json y jsonc (`config.go:322`), seguir aceptando el marcador como señal legacy, pero sin depender de él.
  - En uninstall (`internal/components/uninstall/workspace.go:527`), identificar por nombre y forma (`looksLikeManagedOpenCodeAgent`, `config.go:296-306`) además del marcador legacy, incluidos los perfiles.
  - Revisar `GenerateProfileOverlay` y Kilo, que comparten el overlay.
  - Tests: `internal/components/sdd/inject_test.go` (casos de `__managed_by`), `internal/components/uninstall/global_agents_test.go`, `internal/opencode/config_test.go`.
- [ ] **T2 · E2 — Catálogo de OpenCode en streaming (#4043 + #5842)** (~400 líneas)
  - Lectura con `json.Decoder` en streaming, techo de 16 MiB con cancelación y stderr concurrente. Hay que conservar `catalog_process_unix.go` y `catalog_process_windows.go`.
  - Añadir `CatalogError` y mensajes veraces en `model_picker.go`, más el aviso de catálogo degradado de #5842.
  - Tests en `internal/opencode` e `internal/tui/screens`.
- [ ] **T3 · Comprobación previa a E3: versión de OpenCode del equipo**
  - Confirmar con el usuario la salida de `opencode --version` del equipo. Si todos usan V1, T4 queda aparcada.
- [ ] **T4 · E3 — Catálogo de OpenCode V2 (#5115-catálogo + #5841)** (~400 líneas, condicionada a T3, depende de T2)
  - Sondear `opencode --version`.
  - En V2, ejecutar `opencode api get /api/model?location[directory]=<dir>` y parsear con `parseV2ModelAPI` (exige `capabilities.tools` y `enabled`).
  - Capturar la salida en un fichero temporal regular para evitar el truncado a 256 KiB.
  - Mantener `models --verbose` para V1.
  - Portar la intención del **estado final** del `internal/opencode/catalog.go` de upstream, no los PR uno a uno.
  - Riesgo: `api get` es una API interna de OpenCode V2.
- [ ] **T5 · E4 — Adaptador de Copilot CLI (#5656 adaptado, sin telemetría)** (~300-400 líneas)
  - Adaptador `internal/agents/copilotcli`, raíz `COPILOT_HOME` o `~/.copilot`, registro directo sin `init()`, catálogo, factory, doctor (binario `copilot`), validate, TUI y contrato de CodeGraph.
  - Decidir y probar:
    - la cohorte SDD (`internal/components/sdd/orchestrator.go:114,189`, `sddOrchestratorAsset`);
    - el overlay MCP con `mcpServers`, `type: local` y `tools`;
    - los permisos;
    - `engram setup`.
  - Las skills se comparten con VS Code en `~/.copilot/skills`. Hay que probar la idempotencia con los dos adaptadores seleccionados y que el uninstall de uno no borre las skills del otro.
  - Si pasa de unas 400 líneas, dividir en registro y adaptador por un lado, y ajustes de SDD, Engram y MCP por otro.
- [ ] **T6 · E5 — Pequeños de OpenCode (#4896 + parte de #5052)** (~170 líneas)
  - Portar `describeRefreshFailure` a `plugins/skill-registry.ts` con el texto `axiom skill-registry refresh`, y su variante V2. Guarda en `internal/assets/skill_registry_plugin_guard_test.go`.
  - En `stripLegacyTriggerRulesFromOrchestrator` (`internal/cli/run.go:1024`), usar el fichero de settings efectivo (`effectiveOpenCodeSettingsPath`) y `MergeJSONObjectsForPath`.

Dependencias: T2 → T4 y T3 → T4. T1, T5 y T6 son independientes entre sí. T5 tiene además dependencias externas (ver la sección siguiente).

## Relación con la hoja de ruta de visión de producto (P1, P5, D3)

Fuente: `docs/audits/2026-11-03-vision-producto.md`, en la rama `inc/auditoria-vision-producto`, secciones O5, D3 y P5. Engram: `planning/vision-producto-hoja-de-ruta`.

- **T5 (Copilot CLI) coincide con el primer criterio de P5 (hueco G5.1).** Propuesta de fusión: T5 se convierte en la primera rebanada de P5 y entrega el adaptador base (instalación, sync, uninstall, skills, MCP y prompt). El resto de P5 se construye después sobre ese adaptador:
  - modelo, proveedor y razonamiento por fase;
  - custom agents de VS Code Copilot (G5.2);
  - detección de Claude sin CLI (G5.3);
  - selección desde la UI (G5.4).
  Queda por verificar si Copilot CLI admite custom agents con `model`. Eso define cuánto de P5 cabe en este adaptador.
- **P5 depende de P4** (alta de proyecto con elección e instalación de adaptadores). T5 no necesita P4: P4 consume adaptadores, y T5 lo crea. Si se fusiona, P5 se divide en «T5, sin dependencia» y «resto de P5, depende de P4».
- **D3 (distinguir la extensión de VS Code, Desktop y CLI de Claude Code)** solo afecta al adaptador de Claude Code. No condiciona ninguna de las seis tareas. Si D3 decide separar las variantes de Claude, la detección de T5 debe seguir el mismo patrón (binario y directorio de configuración).
- **T2 y T4 (catálogo de OpenCode) alimentan P5 y G5.4**: la selección por fase desde la UI para OpenCode necesita un catálogo fiable. No hay dependencia de bloqueo, pero conviene que `CatalogError` quede disponible para que el dashboard lo lea (`internal/dashboard/service.go:1577-1608`).
- **P1 (distribución con un comando, rama `inc/distribucion-un-comando`, sin fusionar)** toca:
  - `scripts/install.*`, `internal/update`, `internal/workspace` e `internal/installcmd`;
  - la documentación `README.md`, `docs/platforms.md`, `docs/quickstart.md` y `docs/usage.md`.
  - **T5 bloqueada en parte hasta que P1 esté en `main`**: el comando de instalación npm de Copilot CLI va en `internal/installcmd/resolver.go`, que es donde se resuelve hoy cada agente (`resolver.go:43-49`), y su documentación toca `docs/usage.md` y `docs/platforms.md`. El resto del adaptador puede avanzar, pero no se fusiona antes que P1.
  - **T1, T2, T3, T4 y T6 no tocan esos paquetes.** `internal/components/uninstall/workspace.go` no es `internal/workspace`.
  - **Actualización del 2026-11-05: P1 ya está fusionado (PR #220).** La rama se ha rebasado sobre `origin/main`. Con eso desaparece el bloqueo de T5 por P1, pero T5 sigue sujeta al orden entre bloques de la sección siguiente.

## Orden de ejecución entre bloques de la absorción

Lo fija el cruce de ficheros entre las sesiones de la absorción upstream, enviado el 2026-11-05 por la sesión coordinadora. **No se implementa nada hasta que el usuario lo autorice.**

- **Se pueden hacer ya, en paralelo:** T2 y T6. T4 también, si T3 confirma que el equipo usa OpenCode V2.
- **T1 es el eslabón 1 de la cadena en serie de goldens del orquestador.** Toca `internal/components/sdd/inject.go` y `testdata/golden/sdd-opencode-multi-settings.golden`.
  - Va junto a la T9 del bloque de arreglos.
  - La cadena sigue este orden: T1 de adaptadores y T9 de arreglos → T6 de revisión → T1 de guía → T2/T3 de guía → carril SDD versátil.
  - Coincide con la T4 de arreglos en las entradas de agente de OpenCode (`variant` frente a `__managed_by`). La propuesta es que **T1 entre antes**.
- **T5 (Copilot CLI) va la última de todas**, porque comparte ficheros con muchos bloques:

  | Fichero | Lo toca también |
  |---|---|
  | `internal/components/sdd/orchestrator.go:114,189` | guía T1, que lo cambia por D3 |
  | `internal/agents/capabilitymanifest/manifest.go:334` | guía T2/T3 (cambia los digests) |
  | `internal/components/permissions/inject.go:138` | seguridad T3/T4 |
  | `internal/components/uninstall/service.go` | seguridad T3/T5 y arreglos T3 |
  | `internal/installcmd/resolver.go` | arreglos T5 |
  | `internal/components/mcp/inject.go` y `internal/components/engram/healthprobe.go` | seguridad T1/T5 |

  Además, T5 heredará la nueva semántica de modo de fichero de `WriteFileAtomic` (seguridad T5). Los ficheros que escriba en `~/.copilot` deben seguirla.
- **Consecuencia para el PR único (`exception-ok`):** T2, T6 (y T4) pueden ir pronto. T1 y T5 dependen del avance de otros bloques. Si el PR se abre antes de que T5 pueda entrar, T5 sale del PR y se entrega aparte, como primera rebanada de P5.

## Criterios de aceptación

- Ningún fichero que Axiom escriba en la configuración de OpenCode contiene `__managed_by`. Las instalaciones previas quedan limpias tras un sync. El uninstall sigue retirando los agentes gestionados, y solo esos.
- El selector de modelos de OpenCode carga catálogos de hasta 16 MiB. Si el catálogo falla, lo explica sin pila y avisa de que el catálogo está degradado cuando hay modelos configurados.
- Solo si T3 confirma V2: con OpenCode V2 el catálogo se obtiene por la API y no se trunca. V1 sigue funcionando.
- `axiom` puede instalar, sincronizar y desinstalar Copilot CLI sin telemetría y sin romper VS Code Copilot.
- Ningún cambio toca `docs/upstream-absorption-ledger.md`.

## Verificación aplicable

- `go test` en proceso de los paquetes tocados: `./internal/opencode/...`, `./internal/components/sdd/...`, `./internal/components/uninstall/...`, `./internal/tui/screens/...`, `./internal/agents/...`, `./internal/cli/...` (filtrado), `./internal/assets/...`.
- `go vet` de los paquetes tocados.
- No se compila `cmd/axiom`.

## Configuración de trabajo

- **TDD:** desactivado. Lo decidió el usuario el 2026-11-04. Se ejecutan las comprobaciones funcionales ordinarias con `go test` en proceso.
- **Entrega:** `exception-ok`, decidido por el usuario el 2026-11-04. Habrá un único PR con excepción de tamaño (unas 1700 líneas). Cada tarea sigue cerrando con su commit de unidad de trabajo. La fusión de T5 sigue condicionada a P1: si P1 no está en `main` cuando el PR esté listo, T5 sale del PR o el PR espera.
- **Ruta:** escritor delegado para cada tarea que toque 2 o más ficheros no triviales; las comprobaciones de estado se hacen en línea.
- **RDD:** se rige por el interruptor del usuario (`axiom review mode status`); está apagado por defecto.

## Progreso

- 2026-11-04: exploración y veredictos completados. Documento creado en `inc/adaptadores-agentes-upstream`. Pendiente: validación del alcance por el usuario.
- 2026-11-04: se añade la relación con la hoja de ruta. T5 se propone como primera rebanada de P5 y depende en parte de P1 (`installcmd`, docs). D3 no condiciona ninguna tarea.
- 2026-11-04: el usuario valida el alcance y la fusión de T5 en P5.
- 2026-11-05:
  - Rebase sobre `origin/main`, que ya incluye P1 (PR #220).
  - Se anota el orden entre bloques: T2, T6 y T4 primero; T1 en la cadena de goldens; T5 la última.

## Siguiente paso

Esperar la autorización del usuario. Después:
1. T2 y T6 (y T3/T4), sin TDD y en un único PR con excepción de tamaño.
2. T1, cuando le toque en la cadena de goldens.
3. T5, la última.
