# 08 · Adelgazamiento

Revisión 10 (2026-11-26, agilidad): amplía C-08-f con la TUI, reorienta C-08-g y C-08-h, y añade C-08-j, C-08-k y C-08-l. Su diagnóstico está en «Diagnóstico de agilidad».

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
6. (2026-11-26, revisión 10) «Necesito aplicar más medidas para agilizar los desarrollos: el lanzamiento de los test está siendo un infierno de tiempo. Llevo días o semanas para implementar, y la mayoría es por tener sobreingeniería y test que tardan un siglo, más políticas de lanzarlos a cada pequeño paso, más los lanza la CI.»
7. (2026-11-26) Sobre el análisis de agilidad:
   - Aplicar la política de pruebas (C-08-j), borrar los centinelas de redacción (C-08-h) y sacar los meta-tests del toolchain a un job aparte (C-08-k).
   - CI en `main`: «una vez al día a la noche si ha habido commits» (C-08-k).
   - «Quitamos knowledge y semantic, revisamos en semantic que ahora se use directamente codegraph» (C-08-l).
   - «La tui no la uso y estoy metiendo el contenido que me interesa todo en el dashboard, por lo que tui se puede quitar lo que quieras y test más aún.» Va en C-08-f, y `axiom` sin argumentos abrirá la UI.
   - Modo ISO: «nada, dejamos como está; ya se analizará más a fondo».
   - «Del plan no quiero recortar nada.»

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

### Diagnóstico de agilidad (revisión 10, 2026-11-26)

Análisis de solo lectura contra `main` en `f85ae071`, con A1 a A3 fusionadas. Se basa en cuatro exploraciones (tiempos, necesidad de los tests, mapa de código y políticas de ejecución) y en los logs de CI. No se ejecutó ninguna batería local. Detalle en Engram: `analisis/agilidad-tests-2026-11`.

| Medida | Valor |
|---|---|
| Go de producción y de test | 141.853 y 270.146 líneas (1,9 de test por línea de código); 7.268 `func Test`, 319 `t.Parallel()` y 739 `t.Setenv` |
| CI de Linux | `Unit Tests` 2:21 en `main` (ejecución 45718033589). De apertura a merge, 3-17 min en 14 de las últimas 20 PRs. No es el cuello de botella |
| Windows frente a Linux, por paquete | `sddstatus` 33× (2,1 → 70 s), `sddpublish` 26× (4,7 → 122 s), `techcontext` 24×, `dashboard` 11×, `update` 5,5×. `internal/cli`: 52 s en Linux y más de 10 min en Windows local. La causa son los procesos git por test; por ejemplo, `sddpublish/helpers_test.go:84` lanza cuatro por repositorio y el paquete no tiene `TestMain` |
| Reglas que obligan a lanzar baterías | `internal/components/agentguidance/delegation.go:67` («all applicable full checks when a task closes», que llega a todos los agentes); el tier más alto del verificador independiente cuando no está claro; `AGENTS.md:25` (toda la suite por paquetes, con 20 y 60 min de límite); `openspec/config.yaml:39-40` (`go test ./...`) |
| Acoplamiento de prosa y test | 18 de los 21 commits que solo tocaban prosa de `internal/assets` tuvieron que tocar tests (86 %). Los ocho ficheros de test que más cambian son centinelas; `assets_test.go`, 23 commits en 20 días |
| Tests de lo retirado | 314 funciones de ausencia o lápida (unas 3.600 líneas) |
| CI en el push a `main` | Repite la batería entera tras cada merge (media de 529 s) |

Peso de lo que retira la revisión 10:

| Bloque | Producción | Tests |
|---|---|---|
| TUI (C-08-f ampliada) | 13.181 | 17.471 |
| `knowledge` y `semantic` (C-08-l) | ~2.900, más comandos de `cmd/axiom` y el panel del dashboard | ~2.000 |
| Centinelas de redacción y lápidas (C-08-h) | — | ~10.000-15.000 (estimación) |

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

### C-08-f · Arranque mínimo: retirar los componentes de inicio heredados y la TUI

- **Objetivo.** Que la instalación sea detectar agentes, elegir componentes e instalar, sin pasos heredados de gentle-ai. Desde la revisión 10, además, que no quede TUI: la única interfaz interactiva es la UI web (dashboard).
- **Decisiones.**
  - (Usuario) «Me sobran muchos de inicio.»
  - (Usuario, 2026-11-26, revisión 10) «La tui no la uso y estoy metiendo el contenido que me interesa todo en el dashboard, por lo que tui se puede quitar lo que quieras y test más aún.» Y sobre dónde va: «quitamos la tui y si es en c-08-f ya está en marcha una sesión con ello». La sesión en curso recibió la ampliación el 2026-11-26, con T1 (agent builder) ya cerrada.
  - (Usuario, 2026-11-26) `axiom` sin argumentos en una terminal abre la UI, igual que `axiom ui` (`cmd/axiom/main.go:414`). Sin TTY se conserva el comportamiento actual (`internal/app/app.go:85`).
  - (Análisis, revisión 10) Se retira `internal/tui` entero (`screens`, `styles` y `testdata`) con sus tests, más el arranque desde `internal/app/app.go` (`runTUI`, `tui.NewModel` en `:251` y los envoltorios `tuiSync`, `tuiUpgrade`, `tuiUninstall` y `tuiUninstallWithProfiles` que solo use la TUI). Con ello caen las pantallas de persona, presets, bienvenida y agent builder, así que las tareas de persona y presets se quedan en lo que no es TUI: flags, componentes, assets y lectores tolerantes.
  - (Análisis, revisión 10) `install`, `sync`, `upgrade`, `uninstall` y `backup` siguen como comandos de la CLI. La instalación de agentes desde la UI llega con C-06-g (ola 7); no se adelanta.
  - (Análisis, según P19) Persona: se conserva solo la de Axiom (`persona-axiom.md` y `output-style-axiom.md`), que define la voz del orquestador. Se retiran `gentleman` y `neutral` y la pantalla de elección.
  - (Análisis) Presets: los cuatro actuales (`model/types.go:203-209`: `full-gentleman`, `ecosystem-only`, `minimal` y `custom`) se sustituyen por un conjunto por defecto con el núcleo más la selección manual. Los nombres los fija la sesión.
  - (Análisis) Se retiran el código de migración de temas (`components/theme`, ya fuera de los presets) y `components/opencodedefault` (`ComponentOpenCodeGentleLogo`).
  - (Análisis, según P18) Se retira el agent builder.
  - (Análisis) Se quitan las entradas de `ScreenWelcome` que llevan a lo retirado.
  - (Análisis) La lectura es tolerante: un estado con persona o preset retirados carga con un aviso.
- **Fuera de alcance.** Los componentes del núcleo: SDD/ODD, skills, Engram, guías, MCP, permisos y CodeGraph.
- **Carril propuesto.** ODD con `## Spec viva`. Si al implantar hay que reescribir requisitos de `openspec/specs/persona-behavior-contract`, o los de la TUI en lugar de retirarlos, pasa a SDD (misma regla que C-01-c).
- **Tamaño.** XL desde la revisión 10, casi todo borrado: unas 13.500-14.500 líneas de inicio heredado más las de la TUI (13.181 de producción y 17.471 de test, que se solapan en parte con las anteriores).
- **Depende de / bloquea a.**
  - Depende de C-08-d y C-08-e, con las que comparte `catalog/components.go`, `model/presets.go` y la pantalla de presets.
  - Depende de P18, P19 y P22.
  - Bloquea a C-08-g, C-08-h y C-08-l.
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
  - **TUI (revisión 10).**
    - `internal/tui` entero e `internal/app/app.go`.
    - Tests de paridad: `internal/app/parity_test.go` e `internal/dashboard/listing_tui_parity_test.go`.
    - La entrada `internal/tui` de `writerRoots` en `internal/branch/engine/writer_inventory_test.go:56`.
    - Specs vivas que se retiran (REMOVED con `(Reason: ...)`): `axiom-tui-branding`, `installer-picker-navigation`, `tui-spanish-localization` y `tui-ui-parity`. Se añade un requisito para «`axiom` sin argumentos abre la UI».
- **Criterio de terminado.**
  - No queda `internal/tui`, y `bubbletea` y sus dependencias salen de `go.mod` si ningún otro paquete las usa.
  - `axiom` sin argumentos en una terminal abre la UI; sin TTY, el error de hoy.
  - Solo se instala la persona de Axiom.
  - Un estado con persona o preset retirados carga con un aviso.
  - `rg -il gentleman` sin resultados fuera de los históricos y del lector tolerante.
  - Tests en verde.
- **Riesgos y avisos.**
  - Los usuarios con el output style `gentleman` instalado: `sync` debe sustituirlo por el de Axiom o dejarlo como no gestionado. Lo decide la sesión y lo documenta.
  - Hasta C-06-g no hay instalador interactivo: la primera instalación se hace con `axiom install --agent ...`. Se documenta en las notas de la versión.

### C-08-g · Pruebas en paralelo y repositorios git de plantilla

- **Objetivo.** Bajar el tiempo de pared de la suite: paralelizar los paquetes que hoy no lo hacen y abaratar la creación de repositorios git en los tests.
- **Decisiones.**
  - (Análisis, de inc-25) El patrón que funcionó en `internal/branch/engine` dio ×2,4 en Windows (327 s → ~137 s): entorno fijado una sola vez en `TestMain` (HOME aislado, gitconfig, autor) y `t.Parallel()`.
  - (Análisis, revisión 10) El orden sale del cociente medido entre Windows y Linux, que señala dónde pesan los procesos git:
    1. `sddpublish` (26×; sin `TestMain`; `newRepo` en `helpers_test.go:84`).
    2. `sddstatus` (33×; su `TestMain` no aísla git).
    3. `dashboard` (11×).
    4. `techcontext` (24×; ya tiene `treeTemplate`, pero solo para un fixture).
    5. `cli`: 232 `t.Setenv` y unos 125 usos de `isolateIntegrateGit`.
    6. Después, si compensa, `components/sdd`, `app` y `update`.
  - (Análisis, revisión 10) Además de HOME y gitconfig, el `TestMain` desactiva `fsync`, `gc`, el mantenimiento y `fsmonitor`, como ya hace `cli`.
  - (Análisis, revisión 10) `tui` sale de la lista: la retira C-08-f.
  - (Análisis) `t.Setenv` impide `t.Parallel()`. Se sustituye en los helpers `isolate*` por entorno inyectado o fijado en `TestMain`. Los helpers son `isolateIntegrateGit` (`cli/gittest_helpers_test.go:16`), `isolateBranchGit` e `isolateGateGitEnv` (`cli/sdd_gate_planning_home_test.go:20`).
  - (Análisis) Un helper común de repositorio de plantilla en `internal/testenv`:
    - Inicializa una vez por paquete y copia el directorio (o hace `git clone --local`), en lugar de `git init` y commits en cada test.
    - Unifica 12 `runGit`, 7 `requireGit`, 4 `newRepository` y 3 `newRepo` duplicados.
  - (Análisis) Va después de C-01-e, C-08-d y C-08-f, para no paralelizar tests que se van a borrar. Desde la revisión 10 va también después de C-08-h y C-08-l, por el mismo motivo.
- **Fuera de alcance.** Cambiar comportamiento de producción. Si un paquete lee entorno global y hay que inyectarlo, se anota en el documento ODD.
- **Carril propuesto.** ODD.
- **Tamaño.** L.
- **Depende de / bloquea a.** Depende de C-01-e, C-08-d, C-08-f, C-08-h y C-08-l. No bloquea nada.
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

### C-08-h · Centinelas de prosa: borrar los que fijan redacción

- **Objetivo.** Que cambiar la redacción de una skill, de una instrucción o de un asset no obligue a tocar tests. Se borran los centinelas que fijan redacción y las lápidas de ausencia, y se conservan solo los contratos que lee el código. Hoy son unas 20.000-25.000 líneas en unos 50 ficheros.
- **Decisiones.**
  - (Usuario, 2026-11-26, revisión 10) Se aplica la medida de borrar en lugar de pasar a goldens. Un golden traslada la fricción a `-update` y a revisar el diff, pero no la quita: el 86 % de los commits que solo tocaban prosa tuvo que tocar tests.
  - (Análisis, revisión 10) Se **borran**:
    - las aserciones de ausencia de frases retiradas (por ejemplo, `internal/assets/assets_test.go:75`);
    - los `Contains` sobre redacción libre que ningún código consume;
    - la misma frase comprobada en varios paquetes (se deja como mucho una);
    - los tests de funcionalidades retiradas que no protegen una migración;
    - las lápidas de ficheros: 314 funciones `*Retired`, `*Removed`, `*Absent`, `*NoLonger`, `*Legacy` y `*NotContain`, unas 3.600 líneas, salvo las de migración.
  - (Análisis, revisión 10) Se **conservan**:
    - los tokens y marcadores que parsea el código (`{{AXIOM_*}}`, `axiom.sdd-integration.consent/v1`, `granted` y `declined`, nombres de comandos y flags, claves YAML);
    - las invariantes de seguridad (permisos, nada de push sin autorización);
    - la paridad de plantillas entre runtimes;
    - la tolerancia a estado antiguo (por ejemplo, `state/retired_rdd_roles_test.go`);
    - los goldens que ya renderizan el asset completo.
  - (Análisis) Antes de quitar un token se mira con `git log -S` por qué entró, y el commit lo justifica.
  - (Análisis) Va tras C-01-e, C-08-d y C-08-f: muchas de estas pruebas ya habrán desaparecido y este cambio trata lo que quede.
  - (Análisis, revisión 10) Pasa delante de C-08-g: no se paralelizan tests que se van a borrar, y este cambio quita antes la fricción diaria.
- **Fuera de alcance.**
  - Los tests de comportamiento que además comprueban texto (por ejemplo, los que llaman a `Inject`). Solo se recortan sus aserciones de prosa redundantes.
  - Los tests de `internal/knowledge`, `internal/semantic` y el panel semántico del dashboard (`knowledge_assets_test.go`): los borra C-08-l.
  - Los tests del modo ISO: el usuario lo deja como está.
- **Carril propuesto.** ODD.
- **Tamaño.** M.
- **Depende de / bloquea a.**
  - Depende de C-01-e, C-08-d y C-08-f.
  - Va en paralelo con C-08-l (ficheros disjuntos según el fuera de alcance) y antes que C-08-g.
  - Bloquea a C-08-g.
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
  - Los goldens que queden, regenerados con `-update` y revisados.
  - Tests en verde.
  - Prueba de la fricción: reescribir una frase de `internal/assets/skills/sdd-apply/SKILL.md` que no sea un contrato solo hace fallar, como mucho, goldens que se regeneran. Se registra en el documento ODD y no se commitea.
- **Riesgos y avisos.** Volver a meter prosa retirada sin que nada lo detecte. Se acepta: ningún test comprueba el comportamiento del modelo, y esas guardas pesaban más de lo que protegían. Lo que se reintroduzca se corrige en la revisión de la PR.

### C-08-i · CI de PR por jobs

- **Objetivo.** Que el check `Unit Tests` deje de ser un único job secuencial y que la suite de Windows quede equilibrada tras retirar RDD.
- **Decisiones.**
  - (Análisis) `unit-tests` (`ci.yml`, hacia `:239-299`) se parte en jobs paralelos:
    - `go test` por grupos de paquetes.
    - Binario y humo.
    - `deadcode-ratchet`.
    - Lo que quede de `bench` según P21.
  - (Análisis) Se mantiene un job agregador con el nombre del check obligatorio, para no tocar la protección de rama.
  - (Usuario, 2026-11-24) El atajo de PR solo de planificación ya cubre `docs/plan/`: se hizo en la PR #295, que añadió esta sección. No hay que rehacerlo.
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
  - El atajo `artifacts_only` (`odd/`, `openspec/` y `docs/plan/`) sigue funcionando con la nueva partición.
  - Lista de pasos antes y después sin pérdidas no justificadas.
- **Riesgos y avisos.** Si cambia el nombre del check requerido, se rompe la protección de rama.

### C-08-j · Política de pruebas: en local, solo lo tocado

- **Objetivo.** Que ni las instrucciones que genera Axiom ni las de este repositorio pidan baterías completas en local. Al iterar y al cerrar una tarea se prueban los paquetes tocados; la suite completa es de la CI. Así se acelera cada sesión de las olas siguientes y de cualquier proyecto que use Axiom.
- **Decisiones.**
  - (Usuario, 2026-11-26) «Me interesa aplicar» la política de pruebas del análisis de agilidad.
  - (Análisis) La regla de proporcionalidad de `internal/components/agentguidance/delegation.go:67` cambia «run focused checks while iterating and all applicable full checks when a task closes» por: comprobaciones de los paquetes tocados al iterar y al cerrar la tarea, y la suite completa solo en la CI. Llega a todos los agentes por los goldens (`testdata/golden/*`) y por el `.claude/CLAUDE.md` versionado de este repositorio.
  - (Análisis) El verificador independiente del «Delegated Verification Gate» queda para cambios de seguridad, ejecutables operativos o de modo. Sale la regla «si no está claro, el tier más alto», que en un repositorio de código lo activaba casi siempre. La autoverificación del escritor y la comprobación del padre se mantienen.
  - (Análisis) En este repositorio:
    - `AGENTS.md:21` define «verificar en PASS» como los tests de los paquetes tocados.
    - `AGENTS.md:25` deja de pedir toda la suite por paquetes y pide paquetes tocados con `-run`, nunca `internal/cli` entero.
    - `openspec/config.yaml:39-40,50,72` cambia `go test ./...` por los paquetes tocados y quita lo desfasado (las skills `gentleman-*`).
  - (Análisis) La memoria del usuario («Pruebas localizadas») ya lo aplica en la práctica: este cambio lo convierte en la regla escrita.
- **Fuera de alcance.** CI (C-08-k). Las skills `sdd-verify` y `work-unit-commits` ya piden comprobaciones focalizadas: solo se tocan si contradicen la regla nueva.
- **Carril propuesto.** ODD. Si una spec viva fija la regla de proporcionalidad o los tiers de verificación, va con `## Spec viva`.
- **Tamaño.** S.
- **Depende de / bloquea a.** No depende de nada. No bloquea nada. Va en la A5, en paralelo con C-08-f y C-08-k.
- **Ficheros y paquetes que toca.**
  - `internal/components/agentguidance/delegation.go` y sus tests.
  - La plantilla del «Delegated Verification Gate» (la localiza la sesión dentro de `internal/assets`).
  - `testdata/golden/*` (regenerados).
  - `.claude/CLAUDE.md`, `AGENTS.md` y `openspec/config.yaml`.
- **Criterio de terminado.**
  - `rg 'all applicable full checks'` sin resultados.
  - Goldens regenerados y revisados.
  - Tests de `agentguidance` y de los paquetes de goldens en verde.
- **Riesgos y avisos.**
  - Solape blando con C-08-f en los goldens: el segundo en fusionarse rebasa y regenera.
  - Una regresión que se escapa en local la encuentra la CI de la PR. Es el precio aceptado.

### C-08-k · CI ágil: nocturna en `main`, meta-tests aparte y menos disparadores

- **Objetivo.** Que la CI no repita trabajo: ni la batería completa tras cada merge a `main`, ni los meta-tests del toolchain en cada PR que no los afecta, ni ejecuciones de `PR Validation` que no pueden cambiar el resultado.
- **Decisiones.**
  - (Usuario, 2026-11-26) En `main`: «una vez al día a la noche si ha habido commits». `ci.yml` deja el disparador `push` a `main` y gana un `schedule` nocturno con la misma puerta que `windows-full-suite.yml` (salta si el último `headSha` en verde de `main` es el actual; si no puede leer el historial, ejecuta). Se conserva `workflow_dispatch`.
  - (Análisis) `pr-check.yml` deja `edited`: sus dos comprobaciones (una sola etiqueta `type:*` y autor de confianza o `safe:external`) no miran ni el título ni la descripción. **`unlabeled` se queda**: quitar una etiqueta sobrante o `safe:external` tiene que revalidar.
  - (Análisis) Los meta-tests que compilan o lanzan el toolchain salen de los jobs `Go Tests` y van a un job propio. Ese job corre en `schedule`, en `workflow_dispatch` y en las PR que tocan sus entradas (`.github/workflows/**`, `.goreleaser.yaml`, `internal/update/**`, `scripts/**`, la documentación de comandos y `cmd/axiom/**`). Son estos:
    - `TestDocumentedInvocationsRunAsDocumented` (`internal/app/documented_invocation_test.go:610`);
    - `TestReleaseDistributionPolicyAssertionFailsClosed` (`internal/update/windows_distribution_policy_test.go:83`);
    - `TestCanonicalReleasePublicKeysControlRealLinkerBuild` (`internal/update/release_security_test.go:636`);
    - `TestWindowsFullSuiteShardsCoverEveryTestName` (`internal/assets/formatter_ordering_test.go:293`);
    - el `go build` de `internal/app/tty_boundary_test.go`, si sobrevive a C-08-f.
  - (Análisis) La separación se hace en los workflows con `-skip` en los `Go Tests` y `-run` en el job nuevo, sin tocar los ficheros de test. Así no solapa con C-08-f.
  - (Análisis) Las guardas de la partición (`internal/assets/workflow_shards_test.go`, `formatter_ordering_test.go` y `internal/update/windows_distribution_policy_test.go`) se amplían para que ningún meta-test quede fuera de todos los jobs.
- **Fuera de alcance.** `windows-full-suite.yml`, que ya solo corre con etiqueta, a mano o los lunes. Los jobs manuales.
- **Carril propuesto.** ODD.
- **Tamaño.** S.
- **Depende de / bloquea a.** Depende de C-08-i (fusionada en #317). No bloquea nada. Va en la A5, en paralelo con C-08-f y C-08-j.
- **Ficheros y paquetes que toca.** `.github/workflows/ci.yml`, `.github/workflows/pr-check.yml`, las guardas citadas, `docs/testing/` y la documentación de release.
- **Criterio de terminado.**
  - Un merge a `main` no lanza la CI; la nocturna salta si no hay commits nuevos y corre si los hay.
  - En una PR que no toca las entradas del job, los meta-tests no corren; en una que las toca, sí.
  - `Unit Tests` conserva el nombre y la semántica de agregador.
  - Tiempos antes y después registrados en el documento ODD.
- **Riesgos y avisos.**
  - **Release.** `scripts/require-ci-success.sh` exige una ejecución de `ci.yml` en verde sobre el sha exacto que se etiqueta, sea cual sea el evento. Sin `push` a `main`, antes de etiquetar hay que lanzar `gh workflow run ci.yml --ref main` o etiquetar el sha que pasó la nocturna. Se documenta en la guía de release, y el mensaje de error del script lo indica.
  - Dos PR que pasan por separado y fallan juntas se detectan en la nocturna, no al fusionar.

### C-08-l · Retirar `knowledge` y `semantic`

- **Objetivo.** Retirar los comandos `axiom knowledge` (`sweep`, `crawl` y `query`/`ask`) y `axiom semantic` (`status`, `symbols`, `inspect` y `reindex`) con sus paquetes, y dejar CodeGraph como la única vía de análisis semántico, usado directamente por los agentes.
- **Decisiones.**
  - (Usuario, 2026-11-26) «Quitamos knowledge y semantic, revisamos en semantic que ahora se use directamente codegraph.»
  - (Análisis) `internal/knowledge` solo lo importa `cmd/axiom`. `internal/semantic` lo importan `cmd/axiom` y el dashboard (`internal/dashboard/service.go:51` y `server.go:96-99`). Ninguna skill ni instrucción de `internal/assets` menciona `axiom knowledge` ni `axiom semantic`. Solo aparecen en `docs/manual-de-inicio.md:310,776`.
  - (Análisis, propuesta aceptada por el usuario el 2026-11-26) El panel semántico del dashboard (`/api/semantic/status`, `symbols`, `dependencies` y `reindex`) se retira y no se reimplementa llamando a CodeGraph: los agentes ya lo usan por su MCP y su CLI.
  - (Análisis) Comprobación de que CodeGraph queda en uso directo: lo que instala y registra `internal/components/communitytool` (MCP y, si existe, el bloque de instrucciones) sigue intacto y sin referencias a `semantic`.
  - (Análisis) `governance.semantic_analysis` de `axiom.yaml` (`internal/workspace/types.go:47`) se sigue leyendo, se ignora con un aviso, y `internal/hub/init.go:451` deja de escribirlo.
  - (Análisis) Specs vivas que se retiran: `knowledge-query-engine`, `knowledge-sweep-command` y `semantic-code`. **`knowledge-profile-init` se queda**: es el perfil `knowledge` (`spec-only`) de `axiom init` y no usa el paquete.
- **Fuera de alcance.** El perfil `knowledge` de `axiom init`, CodeGraph y `communitytool`.
- **Carril propuesto.** ODD con `## Spec viva`.
- **Tamaño.** M. Unas 2.900 líneas de producción y 2.000 de test en los paquetes, más los comandos de `cmd/axiom`, el panel del dashboard (Go, JS y tests) y la documentación.
- **Depende de / bloquea a.**
  - Depende de C-08-f: las dos tocan `cmd/axiom/main.go` y el dashboard.
  - Va en paralelo con C-08-h.
  - Bloquea a C-08-g.
- **Ficheros y paquetes que toca.**
  - `internal/knowledge` e `internal/semantic`.
  - `cmd/axiom/main.go`: los casos `knowledge` y `semantic`, `runKnowledge` y `runSemantic*`.
  - `internal/dashboard`: `service.go`, `server.go`, el panel y sus tests (`dashboard_test.go:421-531`, `knowledge_assets_test.go`, `home_assets_test.go:175` y `secrets_canary_test.go:56-59`).
  - `internal/workspace/types.go` e `internal/hub/init.go`.
  - `docs/manual-de-inicio.md`.
  - `openspec/specs/{knowledge-query-engine,knowledge-sweep-command,semantic-code}`.
- **Criterio de terminado.**
  - `rg -l 'internal/(knowledge|semantic)"' -g '*.go'` sin resultados.
  - `axiom knowledge` y `axiom semantic` responden como un comando desconocido.
  - Un `axiom.yaml` con `semantic_analysis` carga con un aviso.
  - Tests de los paquetes tocados en verde.
- **Riesgos y avisos.** Alguien que use `axiom knowledge query` a mano pierde el comando. Se documenta en las notas de la versión.

## Decisiones pendientes

Las pregunta la sesión padre, una cada vez, antes de lanzar la ola que bloquean. Se recogen también en el apartado 6.1 del README.

| Id | Pregunta | Contexto | Recomendación | Bloquea |
|---|---|---|---|---|
| P17 | ¿Se retira también Copilot CLI (`github-copilot-cli`)? | **Resuelta por el usuario (2026-11-24): se queda.** La lista completa de agentes está en C-08-d | — | — |
| P18 | ¿Se retira el agent builder? | Herencia de gentle-ai (`PRD-AGENT-BUILDER.md`). 775 + 1.306 líneas más 931 de TUI | Retirarlo | C-08-f (A5) |
| P19 | ¿Se conserva solo la persona de Axiom y se retiran `gentleman` y `neutral`? | La persona y el output style de Axiom definen la voz del orquestador. Las otras son herencia | Sí, solo Axiom | C-08-f (A5) |
| P20 | ¿Cuánto viven las lápidas de retirada? Son tres grupos: (a) RDD (`rdd_retire.go` y `axiom review stop-hook` como no-op); (b) telemetría (`telemetryruntime` y `telemetry_retire.go`); (c) limpieza en `uninstall` de agentes, GGA y personas retirados | Sin ellas, las instalaciones antiguas se quedan con hooks, plugins o ficheros gestionados huérfanos | Mantener cada lápida hasta la primera versión estable publicada después de su retirada y borrarla en el cambio siguiente. La sesión comprueba si (b) ya lo cumple | C-01-e (A2), C-08-d (A3), C-08-e (A4) |
| P21 | ¿Sobrevive `bench/` a RDD? | Es un módulo Go propio (`bench/go.mod`) que mide la fricción del ciclo de revisión. 91 de unos 120 journeys usan RDD, y en cada PR se compila, se prueba y hace 5 corridas en `unit-tests` | Retirarlo entero en C-01-e, salvo que el usuario use sus informes | C-01-e (A2), C-08-i (A4) |
| P22 | ¿Qué hace `axiom` sin argumentos sin la TUI? | La TUI se retira en C-08-f (revisión 10) | **Resuelta por el usuario (2026-11-26): abre la UI** («axiom sí abrirá la ui») | — |
| P23 | ¿Se reimplementa el panel semántico del dashboard sobre CodeGraph? | C-08-l retira `semantic`, que alimenta ese panel | **Resuelta (2026-11-26): no se reimplementa.** Propuesta aceptada por el usuario con «Sí, adelante» | — |
| P24 | ¿Cuándo corre la batería completa en `main`? | Hoy se repite tras cada merge (media de 529 s) | **Resuelta por el usuario (2026-11-26): «una vez al día a la noche si ha habido commits»** (C-08-k) | — |
| P25 | ¿Se adelgaza el modo ISO? | 2.951 líneas de producción y 4.864 de test para los 10 requisitos de `sdd-iso-evidence` | **Resuelta por el usuario (2026-11-26): se queda como está**; «ya se analizará más a fondo» | — |

## Descartado

- **Paralelizar antes de borrar.** Paralelizar `internal/cli` y `reviewtransaction` ahora obligaría a refactorizar el entorno de unos 50.000 tests de RDD que C-01-e borra. C-08-g va después.
- **Partir C-01-e en varias PR encadenadas.** El plan prohíbe las PR encadenadas. La parte que no era borrado (las primitivas) sale a C-08-c, que es un cambio propio con su PR.
- **Retirar las lápidas sin plazo.** Dejaría huérfanos los hooks y plugins de las instalaciones antiguas. Se decide en P20.
- **Recortar cambios del backlog (revisión 10).** El análisis de agilidad propuso recortar o simplificar C-06-d, C-06-f, C-07-e a C-07-h, C-03-e y C-05-c. El usuario lo descarta: «del plan no quiero recortar nada».
- **Quitar `unlabeled` de `pr-check.yml` (revisión 10).** Se propuso por error: hace falta para revalidar al quitar una etiqueta `type:*` o `safe:external`.
- **Pasar los centinelas a goldens (revisión 10).** Era el enfoque inicial de C-08-h. Traslada la fricción en vez de quitarla.
- **Retirar `consentenvelope` con RDD.** Lo usa el consentimiento de edición de SDD (`sddstatus/consent_contract.go:8`, `governance.go:10`, `governance_handoff.go:11` y `edit_authority_consent.go:13`). Solo `cli/review_consent_contract.go:8` es de RDD.
