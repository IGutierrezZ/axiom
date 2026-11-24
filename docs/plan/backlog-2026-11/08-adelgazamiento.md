# 08 · Adelgazamiento

Análisis de solo lectura del 2026-11-24 contra `main` en `d580368d`. Incluye tres exploraciones (RDD, GGA y telemetría; agentes y componentes; pruebas y CI) y una ejecución completa de `go test ./... -count=1` en Windows. Las rutas y líneas salen de ese inventario y la sesión hija las confirma antes de tocar nada. Detalle en Engram: `analisis/adelgazamiento-2026-11`.

## Origen

En palabras del usuario, resumidas:

1. «Cada vez tardamos más en implementar cambios en Axiom: está creciendo y cada vez se tarda más en lanzar las pruebas.»
2. «Me gustaría adelgazar, además de lo que ya esté preparado en los cambios pendientes, la aplicación y los tests para agilizar los cambios.»
3. «Axiom es un fork de gentle-ai, pero en realidad es algo más ligero: no quiero telemetría, ni GGA, ni RDD, y me sobran muchos de inicio; con Claude, VS Code y OpenCode se puede arrancar.»
4. (2026-11-24) Acepta convertir el análisis en esta sección con C-01-e adelantada.
5. (2026-11-24) Sobre los agentes:
   - «Copilot CLI se queda.»
   - De los otros 13, se van todos menos Gemini CLI, Codex, Cursor, Antigravity y Pi.
   - «De los que se van, que no se vayan los que prácticamente no ahorramos nada quitándolos.»

## Diagnóstico

| Medida | Valor |
|---|---|
| Go de producción | 229.863 líneas en 886 ficheros |
| Go de test | 380.532 líneas en 1.323 ficheros (1,65 líneas de test por línea de código) |
| `go test ./... -count=1` en Windows local | 786 s de pared y 4.159 s sumados entre paquetes. **No termina**: `internal/cli` y `internal/reviewtransaction` agotan el límite de 10 minutos de `go test` con 574 de unos 2.229 tests y 75 de 823 ejecutados |
| Paquetes de más de 150 s que sí terminan | `branch/engine` 434 s, `components/sdd` 228 s, `app` 217 s, `update` 205 s, `sddpublish` 161 s, `dashboard` 158 s |
| Tests que lanzan el toolchain | 4 tests, unos 395 s (ver C-08-b) |
| Fallos locales en `main` | 20 tests en 5 paquetes: `backup`, `filemerge`, `cli`, `app` y `reviewtransaction` |
| Paralelismo | 493 `t.Parallel()` para 9.487 `func Test` (5 %). En `internal/cli` lo impiden 347 `t.Setenv` |
| Linux frente a Windows | `branch/engine`: 29 s frente a 341-568 s (`odd/tasks/inc-25-pruebas-git-baratas.md`). Lo caro en Windows es arrancar procesos git |

Peso de lo que sobra:

| Bloque | Producción | Tests | Otros |
|---|---|---|---|
| RDD (C-01-e) | ~61.300 | ~100.000 | 11.175 de `contracts/review-*`, 20 specs `rdd-*` más `review-findings-ledger`, ~3.200 de docs y 91 de unos 120 journeys de `bench/` |
| 6 agentes (C-08-d) | ~12.300 en total | | incluye ~5.500 de ficheros propios (assets y tests dedicados) y ~3.300 de goldens |
| Componentes de arranque heredados (C-08-f) | ~13.500-14.500 en total | | |
| GGA (C-08-e) | ~975 | ~2.140 | ~160 de docs y spec |
| Telemetría (C-08-e) | ~460 | ~1.100 | 219 de fixtures. No se envía nada: solo quedan lápidas de retirada |
| Paquetes sin importadores (C-08-a) | 2.345 | 1.677 | |

En conjunto, en torno a un tercio del Go del repositorio.

## Cambios resultantes

### C-08-a · Retirar los paquetes sin importadores

- **Objetivo.** Borrar `internal/recoverytrace` (2.052 líneas de producción y 1.444 de test) e `internal/absorptionledger` (293 y 233). Ningún paquete los importa (`rg 'axiom/v3/internal/<paquete>"'` fuera del propio paquete, comprobado).
- **Decisiones.**
  - (Análisis) No son RDD, aunque C-01-e listaba `recoverytrace` como satélite «a confirmar». Se retiran aparte porque no dependen de nada.
  - (Análisis) Antes de borrar se buscan referencias fuera de Go: docs (`recoverytrace/rows.go` solo apunta a docs), `.deadcode-baseline.txt`, `scripts/` y la CI.
- **Fuera de alcance.** Otros huérfanos no inventariados. Si aparecen, se anotan para otro cambio.
- **Carril propuesto.** ODD. No hay comportamiento ni spec viva que cambie.
- **Tamaño.** S.
- **Depende de / bloquea a.** No depende de nada ni bloquea nada. Reduce el alcance de C-01-e.
- **Ficheros y paquetes que toca.** `internal/recoverytrace/`, `internal/absorptionledger/`, `.deadcode-baseline.txt` y lo que devuelva `rg -l 'recoverytrace|absorptionledger'` fuera de los históricos.
- **Criterio de terminado.**
  - `rg -l 'recoverytrace|absorptionledger'` sin resultados fuera de `docs/archive`, `openspec/changes/archive` y `docs/audits`.
  - `go vet ./...` en verde y `scripts/deadcode-ratchet.sh` en verde.
- **Riesgos y avisos.** Ninguno relevante. Los históricos archivados no se tocan.

### C-08-b · Suite local que termina, sin compilar con el toolchain

- **Objetivo.** Que la suite vuelva a servir de semáforo en Windows local. Cada paquete termina y pasa sin compilar binarios, salvo los fallos que desaparecen con otros cambios de esta sección.
- **Decisiones.**
  - (Análisis) Estos cuatro tests lanzan el toolchain de Go y salen de la ejecución por defecto. Pasan a la puerta que ya usa la CI (`AXIOM_TEST_BUILD_BINARIES=1`, `testenv.RequireBinaryBuilds`) o a una propia, según decida la sesión. En la CI se siguen ejecutando.

    | Test | Ubicación | Tiempo |
    |---|---|---|
    | `TestReleaseDistributionPolicyAssertionFailsClosed` | `internal/update/windows_distribution_policy_test.go:82` | 157 s |
    | `TestDocumentedInvocationsRunAsDocumented` | `internal/app/documented_invocation_test.go:637` | 156 s |
    | `TestWindowsFullSuiteShardsCoverEveryTestName` | `internal/assets/formatter_ordering_test.go:302`, con un `go test -list` anidado en `:371` que compila `cli` y `reviewtransaction` dentro de otro test | 68 s |
    | `TestCanonicalReleasePublicKeysControlRealLinkerBuild` | `internal/update/release_security_test.go:644` | 14 s |

  - (Análisis) Los fallos locales de `main` se clasifican uno a uno:
    - Si es un defecto real, se arregla.
    - Si depende del entorno (sockets Unix, junctions sin privilegios, estado del `HOME` real que se filtra al test), se aísla o se omite con `t.Skip`, dejando escritos el motivo y la condición.
  - (Análisis) Los fallos de RDD y GGA no se tocan aquí: desaparecen con C-01-e y C-08-e. Son `TestCodexReviewAdapterDispatchesSavedRoles` (adaptador de revisión de RDD), `TestManagedReviewerAsset*`, `TestNegotiated*` y `TestRunDoctor_*GGA*`. `TestDeferredSyncIncludesCodexPermissionsArgs` sí se trata aquí, porque Codex se queda.
  - (Análisis) Se documenta la orden local recomendada: por paquete, con `-timeout` explícito para `internal/cli` hasta que se fusione C-01-e.
- **Fuera de alcance.** Paralelizar (C-08-g). Retirar o reescribir los tests de prosa (C-08-h).
- **Carril propuesto.** ODD.
- **Tamaño.** S-M.
- **Depende de / bloquea a.** No depende de nada ni bloquea nada.
- **Ficheros y paquetes que toca.**
  - Los cuatro tests anteriores e `internal/testenv`.
  - `internal/backup`: `TestSnapshotterSkipsUnixSockets`.
  - `internal/components/filemerge`: `TestWriteFileAtomicFollowsDirectoryJunction` y `TestWriteFileAtomicFollowsRelativeDirectoryJunction`.
  - `internal/cli`: `TestCheckInstalledAssetVersion_*` y `TestCheckStateJSON_*`.
  - `internal/app`: `TestTuiInstallOnThenSyncPreservesAndRefreshesOpenCodeActivation` y `TestDeferredSyncIncludesCodexPermissionsArgs`.
  - `.github/workflows/ci.yml`, solo si cambia la puerta, para que la CI siga ejecutando los cuatro tests.
  - `AGENTS.md`, para la orden local.
- **Criterio de terminado.**
  - En Windows local, `go test ./... -count=1` sin `internal/cli` ni `internal/reviewtransaction` termina sin fallos.
  - En `internal/cli` solo fallan los tests atribuidos a RDD o GGA, y quedan listados en el documento ODD.
  - El log del job `unit-tests` de la CI muestra que los cuatro tests del toolchain se ejecutan.
- **Riesgos y avisos.** Hay que conservar la protección que dan esos cuatro tests. Salen de la ejecución local, no de la CI.

### C-08-c · Extraer de `reviewtransaction` las primitivas que usan otros paquetes

- **Objetivo.** Que ningún paquete ajeno a RDD importe `internal/reviewtransaction`. Así C-01-e queda en un borrado y puede adelantarse sin chocar con los cambios que reescriben `kickoff`, `sddstatus` y el registro de ramas.
- **Decisiones.**
  - (Análisis, de la sección 01) Se extraen las cuatro primitivas ya identificadas: `AcquireAuthorityFileLock`, `ReplaceFileAtomic`, `SyncReviewDirectory` y `PublishFileNoReplace`.
  - (Análisis, nuevo) La sección 01 no listaba estos usos externos:

    | Símbolo | Dónde se usa |
    |---|---|
    | `SnapshotBuilder` | `internal/kickoff/integration.go:13`, `internal/sddstatus/runtime_ledger.go:212`, `internal/cli/sdd_gate_evidence.go:16` |
    | `CompactAuthoritativeStore` | `internal/sddstatus/runtime_ledger.go:227` |
    | `ErrStoreLockContended`, `ErrConcurrentUpdate` y el tipo `AuthorityFileLock` | (sin ubicar en el inventario) |
    | `InventoryAuthority` y `AuthorityStatusReport` | `internal/cli/branch_lineage.go:28,36,55` |
    | `OpenPhysicalPath` | `internal/cli/compatibility_transaction_windows.go:218` |

  - (Análisis) Cada uso se clasifica antes de moverlo:
    - Si sirve a algo ajeno a RDD, se extrae.
    - Si es la parte RDD de un paquete ajeno, se deja como está y lo borra C-01-e. Por ejemplo, la lectura del almacén de revisiones en `sddstatus/runtime_ledger.go` o el inventario de linajes en `cli/branch_lineage.go`.
    - La clasificación queda en el documento ODD.
  - (Análisis) Es un refactor puro: `reviewtransaction` pasa a importar el paquete neutral y no cambia ningún comportamiento.
- **Fuera de alcance.** Borrar código de RDD.
- **Carril propuesto.** ODD. No toca specs vivas.
- **Tamaño.** M.
- **Depende de / bloquea a.** No depende de nada. Bloquea a C-01-e.
- **Ficheros y paquetes que toca.**
  - `internal/reviewtransaction` (origen).
  - `internal/kickoff`: `ledger.go`, `seal.go` e `integration.go`.
  - `internal/sddstatus`: `runtime_ledger.go` y `edit_authority_consent.go`.
  - `internal/branch/registry/registry.go`, `internal/statecoord` e `internal/filecoord`.
  - `internal/cli`: `branch_lineage.go`, `compatibility_transaction_windows.go` y `sdd_gate_evidence.go`.
  - El paquete neutral nuevo. Su nombre y su forma los decide la sesión.
- **Criterio de terminado.**
  - `rg -l 'internal/reviewtransaction"' -g '*.go'` solo devuelve ficheros de la lista de C-01-e o usos clasificados como «parte RDD» en el documento ODD.
  - Las primitivas extraídas se mueven con sus pruebas, también las `_windows.go`.
  - `go test` de los paquetes tocados en verde en Windows local y en la CI de Linux.
- **Riesgos y avisos.**
  - Las primitivas tratan violaciones de compartición y rutas físicas de Windows, así que sus pruebas no pueden quedarse atrás.
  - El paso de estrés de `ci.yml:623` (`-count=100`) apunta a tests de contexto de repositorio de RDD y desaparece con C-01-e. Si alguno prueba una primitiva extraída, se conserva contra el paquete nuevo.

### C-01-e adelantada

- **Decisión.** (Usuario, 2026-11-24) «Sí, redacta la 08», sobre la propuesta de convertir el análisis en esta sección con C-01-e adelantada.
- **Motivo.** C-01-e pasa de la ola 16 a la A2.
  - La razón de dejarla al final era que movía primitivas que reescriben C-03-b, C-05-c, C-06-b, C-06-c y C-06-f.
  - C-08-c hace antes ese movimiento en un cambio propio.
  - Esos cambios aún no han empezado: arrancarán desde un `main` con las primitivas ya movidas y no tendrán que rebasar.
- **Contenido.** La definición completa sigue en la sección 01, corregida con este inventario:
  - `internal/handoff` y `internal/recoverytrace` no son RDD.
  - `internal/consentenvelope` es mixto: lo usa el consentimiento de edición de SDD.
  - Faltaban `release.yml`, `windows-full-suite.yml`, `bench/`, la TUI, el dashboard, `hub/rddpref`, los 25 agentes `review-*` embebidos, el plugin `opencode-review-transport`, las specs y las docs.
  - Se distinguen las lápidas que se quedan.

### C-08-d · Retirar los agentes que no se usan

- **Objetivo.** Retirar seis agentes: `kiro-ide`, `windsurf`, `kimi`, `hermes`, `openclaw` y `kilocode`.
  - Se quedan once: `claude-code`, `vscode-copilot`, `opencode`, `github-copilot-cli`, `gemini-cli`, `codex`, `cursor`, `antigravity`, `pi`, `trae-ide` y `qwen-code`.
  - `qwen-code` sigue oculto al instalar, como hoy.
- **Decisiones.**
  - (Usuario, 2026-11-24) «Copilot CLI se queda» (resuelve P17).
  - (Usuario, 2026-11-24) De los otros 13 agentes se van todos menos Gemini CLI, Codex, Cursor, Antigravity y Pi.
  - (Usuario, 2026-11-24) «De los que se van, que no se vayan los que prácticamente no ahorramos nada quitándolos.»
  - (Análisis, aplicando esa regla con la medición de `d580368d`) Se queda quien, al quitarlo, ahorra menos de unas 700 líneas y no tiene assets ni goldens que mantener. Medición:

    | Agente | Adaptador | Ficheros propios | Goldens | Referencias fuera (`model.Agent…`) | Ahorro aprox. | Resultado |
    |---|---|---|---|---|---|---|
    | Kiro IDE | 644 | 2.346 | 1.624 | 86 | ~4.700 | se va |
    | Windsurf | 433 | 405 | 1.671 | 62 | ~2.570 | se va |
    | Kimi Code (oculto) | 608 | 1.504 | — | 61 | ~2.170 | se va |
    | Hermes (oculto) | 391 | 641 | — | 63 | ~1.100 | se va |
    | OpenClaw | 346 | 608 | — | 54 | ~1.000 | se va |
    | Kilo Code | 659 | — | — | 117 (52 de producción, 10 en `components/sdd/inject.go`) | ~780 | se va |
    | Qwen Code (oculto) | 435 | 173 | — | 46 (17 de producción) | ~650 | se queda |
    | Trae IDE | 473 | — | — | 33 (11 de producción) | ~500 | se queda |

    Kilo Code queda justo por encima del corte. Se va porque sus 52 referencias de producción están en el núcleo de la inyección y su retirada simplifica más que lo que dicen sus líneas.
  - (Análisis) El código compartido se queda con el agente que lo conserva y se borra solo la rama del agente retirado. Kilo Code comparte con OpenCode `internal/agents/opencode/adapter.go` e `internal/agents/skills_dir.go`.
  - (Análisis) La lectura es tolerante: los estados, copias y selecciones guardados con un `AgentID` retirado siguen cargando y lo ignoran con un aviso.
  - (Análisis) `uninstall` sigue limpiando lo que Axiom escribió para los agentes retirados en instalaciones antiguas, según P20.
- **Fuera de alcance.**
  - Los once agentes que se quedan, incluidos los motores Gemini y Codex del agent builder.
  - Persona, presets, temas y agent builder: C-08-f.
  - GGA: C-08-e.
- **Carril propuesto.** ODD con `## Spec viva`. Las decisiones están cerradas y casi todo es borrado. Si al implantar hay que reescribir requisitos de specs vivas, y no solo retirar menciones a estos agentes, pasa a SDD (misma regla que C-01-c).
- **Tamaño.** L. Unas 12.300 líneas, casi todo borrado, con un commit por agente.
- **Depende de / bloquea a.**
  - Depende de C-01-e: comparten `components/sdd/inject.go`, los agentes `review-*` de Kimi y Kiro, los goldens y los tests de `cli`.
  - Depende de P20.
  - Bloquea a C-08-e, C-08-f, C-08-g y C-08-h.
- **Ficheros y paquetes que toca.**
  - **Registro.**
    - `internal/model/types.go:58-79` y `model/kiro_model*.go`.
    - `internal/catalog/agents.go:12-30` y `:73-77` (`unselectableAgents`, del que salen Kimi y Hermes).
    - `internal/agents/factory.go`: imports, `defaultAgentIDs`, `NewAdapter`, `NewDefaultRegistry` y `BindWorkspace`.
    - `internal/agents/capability_manifest.go`, `discovery.go` y los paquetes `internal/agents/{kiro,windsurf,kimi,hermes,openclaw,kilocode}/`.
  - **Instalación y CLI.**
    - `internal/installcmd/resolver.go:41-54` (`resolveKimiInstall` y la rama de Kilo Code) y `:126-136`.
    - `internal/cli/validate.go:191-219`, `run.go`, `sync.go` y `workspace_instructions.go`.
    - `internal/cli/openclaw_orchestration_test.go` (470).
  - **TUI.**
    - `internal/tui/router.go:15` (`ScreenKiroModelPicker`).
    - `screens/kiro_model_picker.go` (362 con tests) y las referencias en `tui/model.go`.
  - **Componentes.**
    - Ramas por agente en `components/sdd/orchestrator.go` e `inject.go` (10 de Kilo Code), y `sdd/kiro_hook_drift_test.go`.
    - `components/persona/inject.go` y `components/permissions/inject.go`.
    - `components/uninstall/{service,global_agents,workspace}.go`.
    - `components/openclaw_integration_test.go` (138).
  - **Assets.** `internal/assets/{kiro,windsurf,kimi,hermes}/`, sus agentes `review-*` (con C-01-e), la skill `hermes-ephemeral-delegation` y `internal/assets/windsurf_workflow_test.go`.
  - **Goldens.** Los `sdd-kiro-*`, `skills-kiro-*` y los de Windsurf (~3.300 líneas).
  - **Docs.** `docs/kiro.md`. Marcar como superados `odd/tasks/kiro-workspace-parity.md` y la parte de Windsurf de `codex-sync-sin-temas-y-rollback.md`.
  - **e2e.** Las menciones a estos agentes en `e2e/e2e_test.sh` y `e2e/lib.sh`.
- **Criterio de terminado.**
  - `internal/model` y el catálogo solo conocen los once agentes conservados.
  - La instalación (TUI y CLI) ofrece los diez seleccionables. Qwen Code sigue oculto.
  - Un proyecto con selecciones antiguas que incluyen agentes retirados carga y sincroniza con un aviso.
  - `rg -il 'kiro|windsurf|kimi|hermes|openclaw|kilocode|kilo code'` no encuentra nada fuera de los históricos, de las lápidas que decida P20 y del lector tolerante.
  - Goldens regenerados y revisados por diff. Tests de los paquetes tocados en verde.
- **Riesgos y avisos.**
  - Los tests de paridad y drift iteran sobre todos los agentes, así que hay que recortar sus tablas además de borrar ficheros: `skill_copies_parity_test`, `orchestrator_drift_ratchet_test` y `internal/components/golden_test.go`.
  - Kilo Code y OpenCode comparten ramas («familia OpenCode») en `components/sdd/inject.go` y `uninstall`. Al quitar Kilo Code no puede cambiar nada de lo que se genera para OpenCode; se comprueba con sus goldens sin diferencias.

### C-08-e · Retirar GGA y las lápidas de telemetría

- **Objetivo.** Quitar GGA (Gentleman Guardian Angel, el revisor IA previo al commit de gentle-ai) y los restos de la telemetría ya retirada.
- **Decisiones.**
  - (Usuario) «No quiero telemetría, ni GGA.»
  - (Análisis) Axiom deja de instalar, actualizar y diagnosticar GGA. Una selección guardada que lo incluya sigue cargando con un aviso. Lo que Axiom instaló (shims, `pr_mode.sh` y configuración) se trata según P20.
  - (Análisis) La telemetría no envía nada desde la retirada de `odd/tasks/retirada-telemetria-y-restos-upstream.md`, cerrada el 2026-10-07. Quedan estas lápidas, que se borran o se conservan según P20:

    | Lápida | Tamaño |
    |---|---|
    | `internal/components/telemetryruntime` | 261 + 666 líneas, más 219 de fixtures |
    | `components/sdd/telemetry_retire.go` | 121 líneas |
    | `cli/opencode_telemetry_retirement_test.go` | 405 líneas |
    | Llamadas en `cli/run.go:988,1467-1491`, `cli/sync.go:730`, `components/sdd/inject.go:1918-1971` y `components/uninstall/service.go:665,1730` | |

- **Fuera de alcance.** Una instalación de GGA que el usuario hiciera fuera de Axiom.
- **Carril propuesto.** ODD con `## Spec viva`. Retira `openspec/specs/gga` (80 líneas) y se integra con `axiom archive integrate --odd`, igual que C-01-d.
- **Tamaño.** S-M. GGA son unas 975 líneas de producción y 2.140 de test; la telemetría, ~460 y ~1.100.
- **Depende de / bloquea a.**
  - Depende de C-08-d, con la que comparte `cli/run.go`, `sync.go`, `uninstall`, `model` y la pantalla de presets.
  - Depende de P20.
  - Bloquea a C-08-f.
- **Ficheros y paquetes que toca.**
  - **Componente y assets.** `internal/components/gga` (433 + 1.186), `internal/assets/gga` (313) y los tests `cli/gga_*` y `sync_gga_provider_test.go` (560).
  - **Registro.** `catalog/components.go:18`, `planner/graph.go:48`, `model/types.go` y `model/presets.go`.
  - **CLI.**
    - `cli/run.go`: `:508`, `:1997-2052`, `:2205-2290` y `:2902`.
    - `cli/sync.go`: `:396`, `:492-501` (`RestoreDetectedGGA`) y `:1456-1471`.
    - `cli/doctor.go:147`, `doctor_footprint.go` y `machine_footprint.go:69`.
  - **Desinstalación.** `components/uninstall/service.go:157` y `global.go`.
  - **Versiones y actualización.**
    - `versions/versions.go:23`.
    - `update/registry.go:69`, `update/instructions.go:114`, `update/upgrade/strategy.go` y `executor.go`.
  - **TUI.** `tui/screens/{preset,complete,upgrade_sync}.go`.
  - **e2e.** `e2e/e2e_test.sh` y `e2e/lib.sh`.
  - **Docs.** `docs/gga-powershell-shim.md`, `PRD.md`, `docs/components.md` y `docs/usage.md`.
  - Más las lápidas de telemetría de la tabla anterior.
- **Criterio de terminado.**
  - `rg -il 'gga|guardian'` sin resultados fuera de los históricos y del lector tolerante.
  - La instalación no ofrece GGA y `axiom doctor` no lo menciona.
  - Una selección guardada con GGA carga con un aviso.
  - La spec `gga` queda retirada.
  - Tests en verde.
- **Riesgos y avisos.** GGA deja huella del usuario en `~/.config` y shims en Windows, que hoy gestionan `uninstall` y `doctor_footprint`. Hay que fijar qué se borra y qué se documenta.

### C-08-f · Arranque mínimo: retirar los componentes de inicio heredados

- **Objetivo.** Que la instalación sea detectar agentes, elegir componentes e instalar, sin pasos heredados de gentle-ai.
- **Decisiones.**
  - (Usuario) «Me sobran muchos de inicio.»
  - (Análisis, según P19) Persona: se conserva solo la de Axiom (`persona-axiom.md` y `output-style-axiom.md`), que define la voz del orquestador. Se retiran `gentleman` y `neutral` y la pantalla de elección.
  - (Análisis) Presets: los cuatro actuales (`model/types.go:203-209`: `full-gentleman`, `ecosystem-only`, `minimal` y `custom`) se sustituyen por un conjunto por defecto con el núcleo más la selección manual. Los nombres los fija la sesión.
  - (Análisis) Se retiran el código de migración de temas (`components/theme`, ya fuera de los presets) y `components/opencodedefault` (`ComponentOpenCodeGentleLogo`).
  - (Análisis, según P18) Se retira el agent builder.
  - (Análisis) Se quitan las entradas de `ScreenWelcome` que llevan a lo retirado.
  - (Análisis) La lectura es tolerante: un estado con persona o preset retirados carga con un aviso.
- **Fuera de alcance.** Los componentes del núcleo: SDD/ODD, skills, Engram, guías, MCP, permisos y CodeGraph.
- **Carril propuesto.** ODD con `## Spec viva`. Si al implantar hay que reescribir requisitos de `openspec/specs/persona-behavior-contract`, pasa a SDD (misma regla que C-01-c).
- **Tamaño.** L. Unas 13.500-14.500 líneas.
- **Depende de / bloquea a.**
  - Depende de C-08-d y C-08-e, con las que comparte `catalog/components.go`, `model/presets.go` y la pantalla de presets.
  - Depende de P18 y P19.
  - Bloquea a C-08-g y C-08-h.
- **Ficheros y paquetes que toca.**
  - **Componentes.** `internal/components/persona` (1.243 + 3.922), `theme` (221 + 182) y `opencodedefault` (261 + 108).
  - **Agent builder.**
    - `internal/agentbuilder` (775 + 1.306).
    - Las pantallas `tui/screens/agent_builder_*` (931) y `tui/router.go:47-54`.
    - `PRD-AGENT-BUILDER.md`.
  - **Modelo y catálogo.** `model/types.go:151-161` (`PersonaID`) y `:203-209`, `model/presets.go` y `catalog/components.go`.
  - **TUI.** `tui/router.go:11-14` (`ScreenPersona` y `ScreenPreset`), `screens/persona.go`, `preset.go` y `welcome.go`.
  - **Assets.**
    - `claude/output-style-{gentleman,neutral}.md`.
    - `claude/persona-{gentleman,neutral-residual}.md`.
    - `generic/persona-*.md`. Los de Hermes se van con C-08-d.
  - **Goldens.** `persona-*` (~600 líneas).
  - **Tests.** `tui/preset_flow_test.go`, con `TestPresetSelectionNextScreenFlowMatrix`.
- **Criterio de terminado.**
  - El flujo de la TUI va de agentes al árbol de dependencias sin pasar por persona ni presets.
  - Solo se instala la persona de Axiom.
  - Un estado con persona o preset retirados carga con un aviso.
  - `rg -il gentleman` sin resultados fuera de los históricos y del lector tolerante.
  - Tests en verde.
- **Riesgos y avisos.**
  - Los usuarios con el output style `gentleman` instalado: `sync` debe sustituirlo por el de Axiom o dejarlo como no gestionado. Lo decide la sesión y lo documenta.

### C-08-g · Pruebas en paralelo y repositorios git de plantilla

- **Objetivo.** Bajar el tiempo de pared de la suite: paralelizar los paquetes que hoy no lo hacen y abaratar la creación de repositorios git en los tests.
- **Decisiones.**
  - (Análisis, de inc-25) El patrón que funcionó en `internal/branch/engine` dio ×2,4 en Windows (327 s → ~137 s): entorno fijado una sola vez en `TestMain` (HOME aislado, gitconfig, autor) y `t.Parallel()`. Se extiende por orden de tiempo medido a `cli`, `components`, `app`, `update`, `sddpublish`, `dashboard`, `sddstatus`, `techcontext`, `workspace` y `tui`.
  - (Análisis) `t.Setenv` impide `t.Parallel()`. Se sustituye en los helpers `isolate*` por entorno inyectado o fijado en `TestMain`. Los helpers son `isolateIntegrateGit` (`cli/gittest_helpers_test.go:16`), `isolateBranchGit` e `isolateGateGitEnv` (`cli/sdd_gate_planning_home_test.go:20`).
  - (Análisis) Un helper común de repositorio de plantilla en `internal/testenv`:
    - Inicializa una vez por paquete y copia el directorio (o hace `git clone --local`), en lugar de `git init` y commits en cada test.
    - Unifica 12 `runGit`, 7 `requireGit`, 4 `newRepository` y 3 `newRepo` duplicados.
  - (Análisis) Va después de C-01-e, C-08-d y C-08-f, para no paralelizar tests que se van a borrar.
- **Fuera de alcance.** Cambiar comportamiento de producción. Si un paquete lee entorno global y hay que inyectarlo, se anota en el documento ODD.
- **Carril propuesto.** ODD.
- **Tamaño.** L.
- **Depende de / bloquea a.** Depende de C-01-e, C-08-d y C-08-f. No bloquea nada.
- **Ficheros y paquetes que toca.**
  - Los `TestMain` y helpers de los paquetes citados, e `internal/testenv`.
  - Los mayores consumidores de git: `sddpublish/publish_test.go` y `gitport_test.go`; `branch/engine/{ensure,archive,done}_test.go`; `branch/gitcli/ops_write_test.go`.
- **Criterio de terminado.**
  - Tiempos antes y después, con `go test -json` en Windows local y en la CI de Linux, registrados en el documento ODD por paquete.
  - `go test -race -shuffle=on` de los paquetes paralelizados en verde en Linux.
- **Riesgos y avisos.**
  - Carreras sobre estado global.
  - Bloqueos de ficheros en Windows al copiar plantillas.
  - `-race` no está disponible en Windows local sin gcc, así que se comprueba en Linux.

### C-08-h · Centinelas de prosa a goldens

- **Objetivo.** Reducir los tests que fijan literales de markdown a goldens revisables por diff y a unas pocas guardas de ausencia. Hoy son unas 20.000-25.000 líneas en unos 50 ficheros.
- **Decisiones.**
  - (Análisis) Se conserva lo que protege una regresión real. Las listas de subcadenas por asset se cambian por:
    - un golden del asset renderizado, o
    - una guarda de ausencia agregada para la prosa retirada (RDD, Serena, telemetría).
  - (Análisis) Antes de quitar un token se mira con `git log -S` por qué entró.
  - (Análisis) Va tras C-01-e, C-08-d y C-08-f: muchas de estas pruebas ya habrán desaparecido y este cambio trata lo que quede.
- **Fuera de alcance.** Los tests de comportamiento que además comprueban texto (por ejemplo, los que llaman a `Inject`). Solo se recortan sus aserciones de prosa redundantes.
- **Carril propuesto.** ODD.
- **Tamaño.** M.
- **Depende de / bloquea a.**
  - Depende de C-01-e, C-08-d y C-08-f.
  - Va tras C-08-g, porque los dos tocan ficheros de test de `components/sdd`.
  - No bloquea nada.
- **Ficheros y paquetes que toca.**
  - `internal/assets/assets_test.go` (4.622 líneas):
    - `:36` es `TestSDDOrchestratorsCarryNoRetiredWorkRunCeremony`: 12 orquestadores × 15 tokens = 180 subtests.
    - `:4243` es un escaneo línea a línea.
  - `internal/assets/language_contract_test.go`.
  - Los tests de contrato y vocabulario de `components/sdd` (~4.400 líneas) y los de `components/agentguidance`.
  - `workspace/docs_contract_test.go` (664 líneas).
  - `cli/persona_language_contract_test.go`.
  - El mecanismo de goldens de `internal/components/golden_test.go` (`-update` en `:32`; se aplica en `:999`).
- **Criterio de terminado.**
  - Líneas de test de `internal/assets` y de los contratos de `components/sdd` antes y después, registradas en el documento ODD.
  - Cada token retirado lleva su justificación en el commit.
  - Goldens regenerados con `-update` y revisados.
  - Tests en verde.
- **Riesgos y avisos.** Perder guardas que impiden reintroducir prosa retirada. Se conservan como guardas de ausencia agregadas.

### C-08-i · CI de PR por jobs

- **Objetivo.** Que el check `Unit Tests` deje de ser un único job secuencial y que la suite de Windows quede equilibrada tras retirar RDD.
- **Decisiones.**
  - (Análisis) `unit-tests` (`ci.yml`, hacia `:239-299`) se parte en jobs paralelos:
    - `go test` por grupos de paquetes.
    - Binario y humo.
    - `deadcode-ratchet`.
    - Lo que quede de `bench` según P21.
  - (Análisis) Se mantiene un job agregador con el nombre del check obligatorio, para no tocar la protección de rama.
  - (Análisis) Se rebalancea `windows-full-suite.yml`. Hoy 6 de 10 shards son de `cli` y 3 de `reviewtransaction`.
  - (Análisis) `TestWindowsFullSuiteShardsCoverEveryTestName` se conserva o se sustituye según cómo se valide la nueva partición. Lo decide la sesión.
- **Fuera de alcance.** Los jobs que solo se lanzan a mano (`darwin-runtime`, `organic-runtime-e2e`, `e2e-tests`).
- **Carril propuesto.** ODD.
- **Tamaño.** S.
- **Depende de / bloquea a.** Depende de C-01-e y de P21. No bloquea nada.
- **Ficheros y paquetes que toca.** `.github/workflows/ci.yml`, `.github/workflows/windows-full-suite.yml` e `internal/assets/formatter_ordering_test.go`.
- **Criterio de terminado.**
  - Tiempo de pared de la CI de una PR, antes y después, registrado en el documento ODD.
  - El check obligatorio conserva el nombre y falla si falla cualquier job hijo.
  - Lista de pasos antes y después sin pérdidas no justificadas.
- **Riesgos y avisos.** Si cambia el nombre del check requerido, se rompe la protección de rama.

## Decisiones pendientes

Las pregunta la sesión padre, una cada vez, antes de lanzar la ola que bloquean. Se recogen también en el apartado 6.1 del README.

| Id | Pregunta | Contexto | Recomendación | Bloquea |
|---|---|---|---|---|
| P17 | ¿Se retira también Copilot CLI (`github-copilot-cli`)? | **Resuelta por el usuario (2026-11-24): se queda.** La lista completa de agentes está en C-08-d | — | — |
| P18 | ¿Se retira el agent builder? | Herencia de gentle-ai (`PRD-AGENT-BUILDER.md`). 775 + 1.306 líneas más 931 de TUI | Retirarlo | C-08-f (A5) |
| P19 | ¿Se conserva solo la persona de Axiom y se retiran `gentleman` y `neutral`? | La persona y el output style de Axiom definen la voz del orquestador. Las otras son herencia | Sí, solo Axiom | C-08-f (A5) |
| P20 | ¿Cuánto viven las lápidas de retirada? Son tres grupos: (a) RDD (`rdd_retire.go` y `axiom review stop-hook` como no-op); (b) telemetría (`telemetryruntime` y `telemetry_retire.go`); (c) limpieza en `uninstall` de agentes, GGA y personas retirados | Sin ellas, las instalaciones antiguas se quedan con hooks, plugins o ficheros gestionados huérfanos | Mantener cada lápida hasta la primera versión estable publicada después de su retirada y borrarla en el cambio siguiente. La sesión comprueba si (b) ya lo cumple | C-01-e (A2), C-08-d (A3), C-08-e (A4) |
| P21 | ¿Sobrevive `bench/` a RDD? | Es un módulo Go propio (`bench/go.mod`) que mide la fricción del ciclo de revisión. 91 de unos 120 journeys usan RDD, y en cada PR se compila, se prueba y hace 5 corridas en `unit-tests` | Retirarlo entero en C-01-e, salvo que el usuario use sus informes | C-01-e (A2), C-08-i (A4) |

## Descartado

- **Paralelizar antes de borrar.** Paralelizar `internal/cli` y `reviewtransaction` ahora obligaría a refactorizar el entorno de unos 50.000 tests de RDD que C-01-e borra. C-08-g va después.
- **Partir C-01-e en varias PR encadenadas.** El plan prohíbe las PR encadenadas. La parte que no era borrado (las primitivas) sale a C-08-c, que es un cambio propio con su PR.
- **Retirar las lápidas sin plazo.** Dejaría huérfanos los hooks y plugins de las instalaciones antiguas. Se decide en P20.
- **Retirar `consentenvelope` con RDD.** Lo usa el consentimiento de edición de SDD (`sddstatus/consent_contract.go:8`, `governance.go:10`, `governance_handoff.go:11` y `edit_authority_consent.go:13`). Solo `cli/review_consent_contract.go:8` es de RDD.
