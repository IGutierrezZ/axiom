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

- [x] **T1 · E1 — Overlay de OpenCode sin `__managed_by` (#4998/#4471 adaptado)** (~250-350 líneas)
  - **Hecho**, commit `098ec1ea`: 11 ficheros, +319/−90. Ruta: escritor delegado, por el disparador de 2 o más ficheros no triviales.
  - Purga en sync con `purgeLegacyManagedByMarker` (`internal/components/sdd/inject.go`): solo los valores `axiom/sdd` y `gentle-ai/sdd`, y solo en agentes con nombre gestionado, perfiles incluidos. También se aplica en V2 y en Kilo.
  - Detección por nombre y forma con `IsManagedAgent` (`internal/opencode/config.go`), que usan la elección json/jsonc y el uninstall. `general` y `explore` exigen además `hidden`.
  - Hash baseline de Kilocode actualizado en `review_ledger_contract_test.go`. Golden regenerado: el único cambio son las 23 líneas del marcador.
  - Evidencia:
    - `go test ./internal/opencode/... ./internal/components/sdd/... ./internal/components/uninstall/... ./internal/tui/...`: ok.
    - `go test ./internal/components/`: ok.
    - `go vet`: limpio. `gofmt -l`: vacío.
    - `go test ./internal/cli/` completo: **pendiente, sin resultado concluyente en Windows.**
      - Con el tiempo límite por defecto (10 min) agotó el tiempo en `review_artifact_test.go:99`.
      - Con `-timeout 45m` volvió a agotarlo (2704 s), esta vez en `reviewtransaction.runGitIsolated`, a través de `review_disabled_mutation_test.go:40` y `review_context_entrypoints_test.go:12`.
      - Cuelgues en tests distintos, ambos de revisión que lanzan git. T1 no toca ese código. Coincidieron con las pruebas de los escritores de T2, T4 y T6. Ya consta (Engram #503) que esos tests agotan el tiempo bajo carga en Windows.
      - Queda para la CI del PR o para una ejecución aislada sin carga.
  - RDD apagado. `axiom review assess`: riesgo `medium` (`configuration_change`), sin revisión, porque RDD está apagado.
  - Riesgo aceptado: un agente del usuario con nombre de Axiom (p. ej. `sdd-apply`) que tenga `prompt` y `permission` se retira en el uninstall.
  - Quitar el marcador de `sdd-overlay-single.json` y `sdd-overlay-multi.json` y regenerar `testdata/golden/sdd-opencode-multi-settings.golden`.
  - En sync, purgar el `__managed_by` residual de los agentes gestionados:
    - los de `managedOpenCodeAgentKeys` (`internal/opencode/config.go:335-340`);
    - los perfiles `sdd-orchestrator-<perfil>`.
    El merge JSON es profundo y no borra claves.
  - Al elegir entre json y jsonc (`config.go:322`), seguir aceptando el marcador como señal legacy, pero sin depender de él.
  - En uninstall (`internal/components/uninstall/workspace.go:527`), identificar por nombre y forma (`looksLikeManagedOpenCodeAgent`, `config.go:296-306`) además del marcador legacy, incluidos los perfiles.
  - Revisar `GenerateProfileOverlay` y Kilo, que comparten el overlay.
  - Tests: `internal/components/sdd/inject_test.go` (casos de `__managed_by`), `internal/components/uninstall/global_agents_test.go`, `internal/opencode/config_test.go`.
- [x] **T2 · E2 — Catálogo de OpenCode en streaming (#4043 + #5842)** (~400 líneas)
  - **Verificador independiente (2026-11-05): PASS con observaciones, sin hallazgos bloqueantes ni importantes.**
    - Comprobado con sondas: el límite exacto de 16 MiB, salida completa con código ≠ 0 da `command_failed`, la cancelación da `timeout`.
    - `-race` no se pudo ejecutar porque falta cgo.
    - Observaciones menores, que quedan como seguimiento:
      - la prueba de nieto huérfano en Windows (`catalog_process_windows_test.go:61`) aún ejercita `runCatalogCommand` y no `streamCatalogCommand`;
      - faltan tests explícitos de cancelación con el hijo vivo, de salida completa con código ≠ 0 y del límite exacto;
      - `readLine` reescanea el búfer, con coste cuadrático acotado por el techo.
  - **Commit `6647d70b`**: 7 ficheros, +643/−53. Ruta: escritor delegado.
  - Diseño:
    - `CatalogRunner` devuelve un `io.Reader`, que se envuelve en un `ceilingReader` de 16 MiB.
    - `parseVerboseCatalog` usa `json.Decoder`.
    - Stderr se descarta.
    - Se conservan el grupo de procesos y el Job Object.
    - El selector muestra `CatalogError` con mensajes por tipo y el aviso de catálogo degradado.
  - Evidencia:
    - `go test ./internal/opencode/... ./internal/tui/...`: ok.
    - `go test ./internal/dashboard/...`: ok.
    - `go test ./internal/cli/ -run 'Catalog|ModelPicker|OpenCode'`: ok (933 s).
    - `go vet`: limpio. `gofmt`: vacío.
  - `axiom review assess`: riesgo `high` (`process_boundary`). Como RDD está apagado, la tabla de niveles pide un verificador independiente; está en curso. La tarea se marca cuando lo cierre.
  - Nota: los mensajes nuevos del selector van en castellano. La TUI ya mezcla idiomas: las pantallas recientes están en castellano.
  - Lectura con `json.Decoder` en streaming, techo de 16 MiB con cancelación y stderr concurrente. Hay que conservar `catalog_process_unix.go` y `catalog_process_windows.go`.
  - Añadir `CatalogError` y mensajes veraces en `model_picker.go`, más el aviso de catálogo degradado de #5842.
  - Tests en `internal/opencode` e `internal/tui/screens`.
- [x] **T3 · Comprobación previa a E3: versión de OpenCode del equipo** — Respuesta del usuario (2026-11-05): **el equipo mezcla V1 y V2**. T4 se implementa manteniendo `models --verbose` para V1. Ruta: en línea, una pregunta al usuario. No hay commit propio: se registra en el commit de documentación de la feature.
  - Confirmar con el usuario la salida de `opencode --version` del equipo. Si todos usan V1, T4 queda aparcada.
- [x] **T4 · E3 — Catálogo de OpenCode V2 (#5115-catálogo + #5841)** (~400 líneas, condicionada a T3, depende de T2)
  - **Verificador independiente (2026-11-05): PASS con observaciones.** No hay hallazgos bloqueantes ni importantes. `parseV2ModelAPI` es idéntico al de upstream, la codificación del comando también, y el ciclo de vida del fichero temporal es correcto en Windows. `-race` no se pudo ejecutar (sin cgo) ni se probó contra un `opencode` real.
  - Observaciones menores, que quedan como seguimiento:
    - Una V2 prerrelease cae a `models --verbose`.
    - El `--version` extra va fuera del plazo de 15 s, con un peor caso de unos 18 s.
    - Falta un test con una ruta no ASCII (`ñ`, `é`).
    - El monitor de tamaño muestrea cada 100 ms.
    - Un `os.Remove` fallido en Windows no se registra.
  - **Commit `a5bff229`**: 3 ficheros, +581/−8, de los que unas 290 líneas son tests. Ruta: escritor delegado.
  - Selección de versión con `DetectRuntimeMajor`. V1, una versión desconocida o un fallo de detección usan `models --verbose`.
  - V2 ejecuta `opencode api get /api/model?location%5Bdirectory%5D=<dir>`. Su salida se captura en un fichero temporal, que siempre se borra, y se lee con el techo de 16 MiB.
  - `parseV2ModelAPI` falla de forma cerrada con `unsupported_schema`.
  - Evidencia:
    - `go test ./internal/opencode/... ./internal/tui/...`: ok.
    - Tests V2 con `-count=3`: ok.
    - `go vet` limpio y `gofmt` sin cambios.
  - `axiom review assess` da `high` (`process_boundary`). Hay un verificador independiente en curso.
  - Sondear `opencode --version`.
  - En V2, ejecutar `opencode api get /api/model?location[directory]=<dir>` y parsear con `parseV2ModelAPI` (exige `capabilities.tools` y `enabled`).
  - Capturar la salida en un fichero temporal regular para evitar el truncado a 256 KiB.
  - Mantener `models --verbose` para V1.
  - Portar la intención del **estado final** del `internal/opencode/catalog.go` de upstream, no los PR uno a uno.
  - Riesgo: `api get` es una API interna de OpenCode V2.
- [x] **T5 · E4 — Adaptador de Copilot CLI (#5656 adaptado, sin telemetría)** (~300-400 líneas)
  - **Verificador independiente (2026-11-11): PASS con observaciones**, sin hallazgos bloqueantes ni importantes. Repetidos sobre `75f6dbd8`:
    - `go test` de `agents`, `copilotconfig`, `mcp`, `engram`, `uninstall`, `sdd` e `installcmd`: ok.
    - `go test ./internal/cli/ -run 'Copilot|Uninstall'`: ok.
    - Sin telemetría, sin `init()`, sin `/v4` y sin marca `gentle-ai` en las líneas añadidas.
  - Observaciones menores, que quedan como seguimiento:
    - `HasCLIState` (`internal/copilotconfig/root.go:41-52`) da por instalado Copilot CLI si `~/.copilot` tiene cualquier entrada distinta de `skills`.
    - Quien solo usa VS Code Copilot verá en el siguiente sync que `_shared` pasa a `<runtime>`. Es inocuo, porque no es un runtime con revisión nativa.
    - El destino `copilot-cli` de CodeGraph no se ha contrastado de forma independiente.
  - **Reanudada el 2026-11-11** en el mismo worktree, recreado con `axiom branch ensure --lane odd` desde `origin/main` 65eb17dc. El PR #227 (T1–T4, T6) se fusionó el 2026-11-05 como d34b5321.
  - Dependencias ya fusionadas en `main`:
    - guía T1 (#238) y T2 (#239);
    - seguridad T3–T5 (#228, #232);
    - arreglos T3/T4 (#229, #231).
  - Cambios de `main` que condicionan T5:
    - la regla de delegación se genera en Go (`internal/components/agentguidance/delegation.go`), y las plantillas de orquestador referencian `_shared/sdd-orchestrator-sections.md`, sin copiar cuerpos (`orchestrator_no_drift_test.go`);
    - el manifiesto ya no tiene campos por número de ficheros, y hay que regenerar los digests;
    - `filemerge.WriteFileAtomic` conserva el modo del fichero existente.
  - Guía T3 corre en paralelo sobre la prosa del orquestador. Si chocamos en `_shared`, se rebasa sobre la que entre primero.
  - **Implementación** (escritor delegado): 4 commits, 51 ficheros, +1814/−98.
    - `a29b473b`: adaptador `internal/agents/copilotcli`, `internal/copilotconfig`, registro, catálogo, manifiesto, doctor, validate, TUI, installcmd y CodeGraph (`copilot-cli` nativo).
    - `2de90554`: MCP y Engram en `mcp-config.json` (`mcpServers`, `type`, `tools`), SDD, permisos sin efecto, skills compartidas y uninstall.
    - `fbd131fc`: documentación.
    - `75f6dbd8`: test de selección por defecto.
  - Decisiones:
    - `COPILOT_HOME` se respeta solo si es una ruta absoluta y el home es el real del usuario, igual que VS Code.
    - Una `~/.copilot` que solo contiene `skills` no cuenta como Copilot CLI. VS Code se detecta por su extensión.
    - Para SDD sigue a VS Code: plantilla genérica y preflight de sesión, sin plantilla nueva.
    - Solo ámbito global, sin modo workspace.
    - Con las skills compartidas, `_shared/review-ledger-contract.md` lleva `<runtime>` genérico (`agents.SharesSkillsDirWithAnotherAdapter`). Además, el uninstall global no borra la carpeta mientras otro agente instalado la use.
  - Evidencia:
    - `go build ./...`: ok.
    - Suite de paquetes: ok, salvo `filemerge` `TestWriteFileAtomicFollowsDirectoryJunction`, que el orquestador reprodujo también en `main` 65eb17dc. Es preexistente.
    - `go test ./internal/cli/ -run 'Doctor|Validate|Copilot|VSCode|Uninstall|Workspace|Install'`: ok (334 s).
    - `go vet` y `gofmtcheck`: limpios.
  - `axiom review assess`: `high` (`process_boundary`). Verificador independiente en curso.
  - Preguntas abiertas de producto: si Copilot CLI recibe la cohorte SDD (se eligió la opción conservadora, igual que VS Code) y si necesita modo workspace (queda fuera).
  - Adaptador `internal/agents/copilotcli`, raíz `COPILOT_HOME` o `~/.copilot`, registro directo sin `init()`, catálogo, factory, doctor (binario `copilot`), validate, TUI y contrato de CodeGraph.
  - Decidir y probar:
    - la cohorte SDD (`internal/components/sdd/orchestrator.go:114,189`, `sddOrchestratorAsset`);
    - el overlay MCP con `mcpServers`, `type: local` y `tools`;
    - los permisos;
    - `engram setup`.
  - Las skills se comparten con VS Code en `~/.copilot/skills`. Hay que probar la idempotencia con los dos adaptadores seleccionados y que el uninstall de uno no borre las skills del otro.
  - Si pasa de unas 400 líneas, dividir en registro y adaptador por un lado, y ajustes de SDD, Engram y MCP por otro.
- [x] **T6 · E5 — Pequeños de OpenCode (#4896 + parte de #5052)** (~170 líneas)
  - Ruta: escritor delegado.
  - **Commit `21493fff`** (3 ficheros, +122/−4), que porta #4896:
    - `describeRefreshFailure` en los plugins V1 y V2. Antes, V2 ignoraba el error en silencio.
    - Los textos nombran `axiom` y no `gentle-ai`.
    - Guarda estática `TestSkillRegistryPluginsDiagnoseSpawnFailures`.
  - **Commit `2eaabf66`** (2 ficheros, +113/−6), que porta parte de #5052:
    - `stripLegacyTriggerRules` limpia el fichero de settings efectivo (`.jsonc` de proyecto antes que el `.json` global) con `MergeJSONObjectsForPath`.
    - Tests en `internal/cli/run_legacy_trigger_settings_test.go`.
  - Evidencia:
    - `go test ./internal/assets/...`: ok; repetido por el orquestador, ok en 28,9 s.
    - `go test ./internal/cli/ -run 'LegacyTrigger|StripLegacy|TriggerRules|OpenCodeSettings|RoutingGuidance'`: ok.
    - `go vet`: limpio. `gofmt`: vacío.
  - `axiom review assess` sobre los dos commits: `medium` (`executable_change`). Con RDD apagado, basta la verificación del escritor y la comprobación puntual.
  - Las guardas que ejecutan el plugin con node se saltan en Windows. La salida de V1 se comprobó a mano con node.
  - Seguimiento fuera de alcance: el plugin V2 conserva el marcador `// gentle-ai:managed skill-registry/v2` y el id `gentle-ai.skill-registry`, que dependen del instalador y de tests. Es un resto de marca para el bloque de marca.
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

Plan stop: approved 2026-11-04; implementation continues

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

- 2026-11-05: el usuario autoriza arrancar las tareas desbloqueadas, a través de la sesión coordinadora («me gustaria que arranquen las que pueden empezar»).
  - T1, T2, T3, T4 y T6 hechas: 6 commits, unas +1778/−161 líneas.
  - Los verificadores independientes de T2 y T4 dan PASS con observaciones menores.
  - Pendiente: la suite completa de `internal/cli` en Windows (ver T1).

## Siguiente paso

1. **T5 (Copilot CLI)**: fusionada el 2026-11-11 con el PR [#241](https://github.com/IGutierrezZ/axiom/pull/241) (merge `633b2225`). Sus seguimientos menores van en S2.
2. **T1 libera arreglos T4**, que esperaba a que se fusionara. La decisión de push y PR es del usuario.
3. Seguimientos menores de T2 y T4: hechos el 2026-11-05 (ver más abajo). El de marca de T6 (`gentle-ai:managed` en el plugin V2) queda para el bloque de marca.

## Entrega

- 2026-11-05: el usuario lo autoriza y se abre el PR [#227](https://github.com/IGutierrezZ/axiom/pull/227) hacia `main`, con las etiquetas `type:feature` y `size:exception`.
  - Contiene T1–T4, T6 y S1, rebasados sobre `origin/main` (8bf0fb8f).
  - T5 queda fuera y se entregará en un PR propio, como primera rebanada de P5.
- 2026-11-05: correcciones por la CI del PR #227.
  - `79e44f43`:
    - El uninstall reconoce también las lentes `review-*` (`ReviewLensPhases`) y los agentes de Kilo que declaran `tools` en lugar de `permission`. Antes solo los identificaba `__managed_by`.
    - El plugin V2 tipa el error de `execFile` (`code` puede ser `number | null`).
    - Evidencia local: `go test ./internal/cli/ -run TestUninstall`: ok.
  - `061c1364`: se retira `DiscoverCatalogWithRunner`, que había quedado sin llamadores fuera de los tests y fallaba el deadcode ratchet. Los tests usan el helper `discoverV1CatalogWithRunner`.
    - Evidencia: `go test ./internal/opencode/`: ok. `go build ./...`: ok.
  - El typecheck del SDK V2 (`scripts/test-opencode-v2-contracts.sh`) no se puede ejecutar en Windows, así que solo lo cubre la CI.

## Seguimientos de verificación (T2/T4)

- [x] **S1 · Seguimientos menores del catálogo** — Ruta: escritor delegado. Commits `18037897` y `76ce8751`, 5 ficheros, +230/−1.
  - Se hizo:
    - **Nieto huérfano en Windows**: `TestCatalogCancellationLeavesNoOrphanGrandchildWindows` cubre `streamCatalogCommand` y `fileCatalogCommand`. Usa el binario de test relanzado, sin `go build`.
    - **Tests de T2 en V1 y V2**:
      - cancelación con el hijo vivo → `timeout`;
      - salida válida con código ≠ 0 → `command_failed`;
      - límite exacto de 16 MiB.
    - **Ruta no ASCII en V2** (`ñ`, `é`, espacio): mismo resultado que `url.QueryEscape` de upstream.
    - **Plazo**: la detección de versión comparte ahora el plazo único de 15 s del catálogo, y `DetectRuntimeMajor` conserva su tope de 3 s.
  - No se hizo: registrar el fallo de `os.Remove`. El paquete no tiene mecanismo de log; queda un comentario que lo explica.
  - Evidencia:
    - `go test ./internal/opencode/... ./internal/tui/...`: ok.
    - `-run 'Catalog|V2|Process' -count=3`: ok.
    - Comprobación puntual del orquestador (`-run 'Catalog|V2'`): ok.
    - `go vet` limpio y `gofmt` vacío.
  - `axiom review assess`: `medium`. Con RDD apagado basta la verificación del escritor.
  - Riesgo: el test del nieto espera 300 ms a que el hijo quede asignado al Job Object.

## Seguimientos de verificación (T5)

Autorizados por el usuario el 2026-11-11 («Sí, hazlos aquí»). Se hacen en la rama de la feature, recreada con `axiom branch ensure --lane odd --mode worktree` desde `origin/main` `588b0433`. TDD desactivado (configuración de la feature).

- [x] **S2 · Seguimientos menores de Copilot CLI**. Ruta: en línea (2 ficheros de código mecánicos, ya entendidos) más un investigador delegado de solo lectura para las fuentes primarias.
  - **CodeGraph**: contrastado el 2026-11-11 con `codegraph install --print-config copilot-cli` (1.6.2). Escribe `~/.copilot/mcp-config.json` con `mcpServers`, `type: stdio` y `tools: ["*"]`, que es el destino que declara `internal/components/communitytool/codegraph_contract.go:52-54`. No hace falta cambiar código.
  - **`HasCLIState`** (`internal/copilotconfig/root.go`): deja de contar «cualquier entrada distinta de `skills`» y pasa a una lista cerrada de entradas propias del CLI.
    - Su estado: `config.json`, `session-state`, `session-store.db`, `command-history-state`, `logs`, `permissions-config.json`, `installed-plugins`, `plugin-data` e `ide`.
    - Lo que solo él lee: `settings.json`, `mcp-config.json` y `copilot-instructions.md`.
    - Fuente: [referencia del directorio de configuración](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-config-dir-reference). VS Code lee también `~/.copilot/skills` y `~/.copilot/agents` ([skills](https://code.visualstudio.com/docs/copilot/customization/agent-skills), [agents](https://code.visualstudio.com/docs/agents/reference/custom-agent-file-reference)), así que esas dos y cualquier entrada desconocida no cuentan.
    - Test de tabla `TestHasCLIStateOnlyCountsCopilotCLIEntries`. La documentación de detección está en `docs/agents.md`.
  - **Identidad de las skills compartidas**: documentado en `docs/agents.md`, sección «GitHub Copilot CLI». Al instalar Copilot CLI junto a VS Code Copilot, `_shared/review-ledger-contract.md` pasa a `<runtime>` en el siguiente sync, sin efecto práctico.
  - **Custom agents con `model` en Copilot CLI (para P5)**: existen.
    - Están en `~/.copilot/agents/*.agent.md` o `*.md`, con frontmatter `name`, `description`, `tools`, `model` (ID cualificado por proveedor) y `model-policy: required` ([docs](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/create-custom-agents-for-cli)).
    - Un `--model` explícito gana al modelo del agente, salvo que el agente lo exija (changelog 1.0.106/1.0.107). También existen `--model` y `COPILOT_MODEL`.
    - Consecuencia: el modelo por fase de P5 cabe en este adaptador con agentes de fichero.
    - Riesgo: `~/.copilot/agents` lo lee también VS Code, con su propio formato de frontmatter. P5 debe decidir si los agentes son compartidos o se separan con `COPILOT_HOME`.
    - Hueco: no consta la versión que introdujo `model` ni si `settings.json` tiene una clave `model` general.
  - Evidencia:
    - `go vet ./internal/copilotconfig/`: limpio; `gofmt -l`: vacío.
    - `go test ./internal/copilotconfig/ ./internal/agents/... ./internal/system/`: ok.
    - `go test ./internal/cli/ -run 'Copilot|VSCode|Doctor|Detect'`: ok.
