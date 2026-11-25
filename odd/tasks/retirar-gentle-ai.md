# retirar-gentle-ai

## Objetivo

Que Axiom deje de usar los nombres heredados `gentle-ai`, `gentle_ai`, `GentleAI` y `GENTLE_AI_*` como identificadores del producto: rutas de estado, marcadores, esquemas, variables de entorno, contratos de release, nombres internos, pruebas y documentación viva. Solo queda una migración que, en la siguiente ejecución, reconoce lo viejo para sustituirlo o moverlo.

## Problema y porqué

Al renombrar el producto se conservaron estos nombres como «identificadores heredados de compatibilidad» (`docs/agents.md:42`, `docs/audits/2026-11-03-vision-producto.md:138`). Axiom es un fork personal: no hay otros usuarios ni repos que dependan de ellos (comprobado el 2026-11-25: ningún repo de `C:\repos` tiene `.git/gentle-ai`). Esa compatibilidad solo añade ruido y rutas dobles.

El inventario (2026-11-25, HEAD `3c55c493`) encontró unas 2.200 líneas:
- 435 de código Go fuera de pruebas;
- 1.353 en pruebas;
- 375 en plantillas embebidas;
- el resto en scripts, workflows, docs y specs.

Hallazgo en el camino: `internal/app/selfupdate.go:119,126,164` filtra el autoupdate por el nombre `"gentle-ai"`, pero `cmd/axiom/main.go` renombra el registro a `"axiom"` al arrancar. El autoupdate al arrancar nunca encuentra su propia herramienta, así que en la v3.8.0 está apagado de hecho.

## Decisión

Decisión del usuario (2026-11-25), opción «corte con migración»:

- **Todo pasa a `axiom`.** Rutas, marcadores, ids de plugin, esquemas, variables, assets de release, identificadores Go, pruebas y docs.
- **Se retira la compatibilidad heredada.** Fallbacks de lectura, variables `GENTLE_AI_*`, esquemas `Legacy*` y nombre de herramienta `gentle-ai`.
- **Queda solo la migración.** En la siguiente ejecución que toque cada cosa (`axiom sync`, la apertura del estado o del registro de runtime):
  - Reconoce marcadores `<!-- gentle-ai:… -->`, ids de plugin y entradas MCP viejos, para sustituirlos por los nuevos o retirarlos.
  - Mueve los ficheros y directorios de estado viejos a su ruta nueva si la nueva no existe.
  - Acepta al leer los esquemas `gentle-ai.*` de ficheros ya escritos en disco y los reescribe con el nombre nuevo cuando vuelva a escribirlos.
- **Se conservan las atribuciones e historia.** El nombre gentle-ai se mantiene donde es atribución o historia, no identificador: `TRADEMARKS.md`, `CONTRIBUTORS.md`, las referencias al upstream `Gentleman-Programming/gentle-ai`, `docs/upstream-absorption-ledger.md`, `docs/releases/`, `docs/audits/`, `docs/archive/` y `openspec/changes/archive/`.

El actualizador de un binario v3.8.0 no usa ningún nombre `gentle-ai`. Descarga `axiom_<ver>_<os>_<arch>.tar.gz`, `checksums.txt` y `.minisig`, con el trusted comment `repo=IGutierrezZ/axiom;tag=v…`. Por eso los assets de release se renombran sin transición.

## Alcance

- Código Go de `cmd/` e `internal/`.
- Plantillas de `internal/assets`.
- `scripts/`, `.github/workflows/`, `.goreleaser.yaml`, `.gitignore`, `e2e/` y `contracts/`.
- Docs vivas, `openspec/specs/` (mediante `## Spec viva`), `openspec/INDEX.md` y `openspec/config.yaml`.

Fuera de alcance:
- Los nombres de secretos de GitHub (`DISCORD_GENTLE_AI_*`). El workflow se renombra, pero los secretos los renombra el usuario en GitHub.
- La documentación histórica y de atribución citada arriba.

## Restricciones

- Cortex XDR borra ejecutables Go sueltos: nada de `go build -o`, `go run` ni `go install`, y tampoco `go test ./...` local. Se usa `go test` por paquete sin `AXIOM_TEST_BUILD_BINARIES`. Los paquetes con arneses de binario (`internal/app`, `internal/dashboard`, `internal/opencode`, `internal/update`) se verifican en CI cuando local no basta.
- Un commit por tarea, conventional commits sin atribución de IA. Identidad: `git -c user.name="Iñigo Gutiérrez Zufia" -c user.email="igutierrezz@hiberus.com"`.
- Push, PR y merge los decide el usuario.

## TDD

Modo: desactivado (no hay configuración en `axiom.yaml` ni petición). Checks funcionales por tarea: `go build ./...`, `go vet` de los paquetes tocados, `go test` de los paquetes tocados y `go run ./internal/gofmtcheck`.

`go run ./internal/gofmtcheck` es un comando del repo que el preflight de release ya ejecuta. Compila un binario temporal en la caché de Go; si el EDR lo bloquea, se usa `gofmt -l` sobre los ficheros tocados.

Cada migración lleva su prueba: el nombre viejo en disco acaba en el nombre nuevo.

## Tareas

Ruta común: un escritor delegado por tarea, de uno en uno, para proteger el contexto del orquestador (sesión larga). Cada uno hace su commit.

- [x] T1. Autoupdate y registro del actualizador.
  - Arreglar el filtro del self-tool en `internal/app/selfupdate.go` para que use el nombre real (`axiom`).
  - El registro `internal/update/registry.go` se llama `axiom` y se quita `selfToolNames` gentle-ai y la reescritura en `cmd/axiom/main.go` si ya no hace falta.
  - Variables `AXIOM_NO_SELF_UPDATE`, `AXIOM_SELF_UPDATE_DONE` y `AXIOM_YES`, sin `GENTLE_AI_*`. `scripts/install.ps1` fija `AXIOM_NO_SELF_UPDATE`.
  - Renombrar los identificadores Go `*GentleAI*` de `internal/update` e `internal/app`.
  - Evidencia (commit «fix(update): autoupdate y registro con el nombre axiom»; ruta delegada, motivo: protección de contexto del orquestador; escritor único):
    - `go build ./...`: correcto.
    - `go vet ./internal/update/... ./internal/app ./internal/dashboard ./internal/cli ./cmd/... ./internal/tui/...`: sin avisos.
    - `go test ./internal/update/... -count=1`: ok (`update` y `update/upgrade`).
    - `go test ./internal/cli -run 'Doctor|Channel|Update|CheckOneTool|Axiom|Version' -count=1`: ok.
    - `gofmt -l` sobre los `.go` modificados: salida vacía.
    - Regresión del bug: `TestSelectSelfToolResultFindsTheRegistryEntry` y `TestSelfToolRegistryNameMatchesTheIdentity` (`internal/update/identity_test.go`); la selección vive en `update.SelectSelfToolResult`, que usa `selfupdate.go`.
    - Verificado solo en CI: pruebas de `internal/app`, `internal/tui`, `internal/dashboard` y `cmd/axiom` (renombrados, literales y `AXIOM_*` ajustados; solo `go vet` y `go build` en local).
    - Decisión: se retira el `init()` de `cmd/axiom/main.go` (el registro ya trae las coordenadas reales) y `selfToolNames`; `update.SelfToolName` es la única fuente del nombre. Quedan fuera de T1 las variables de canal y ámbito (T2), los prefijos de temporales y rutas (T3) y el comentario de `main.go:406` (T7).
- [x] T2. Variables de entorno.
  - Quitar todos los fallbacks `GENTLE_AI_*`: canal, ámbito de instalación, subagentes en segundo plano de OpenCode y Pi, modo de Engram y `GENTLE_AI_SDD_STATUS_ENGRAM`. Quitar también el helper `lookupEnvWithGentleAIFallback` y `EnvChannelGentleAI`.
  - Scripts de instalación.
  - Variables solo de pruebas y CI pasan a `AXIOM_*`.
  - Evidencia (commit «refactor(env): retirar las variables GENTLE_AI_*»; ruta delegada, motivo: protección de contexto del orquestador; escritor único):
    - `go build ./...`: correcto.
    - `go vet` de `./internal/cli ./internal/update/... ./internal/system ./internal/components/engram ./internal/components/communitytool ./internal/sddstatus ./internal/assets ./internal/components/agentguidance ./internal/tui ./internal/app ./internal/filecoord ./internal/fileprim ./e2e/... ./cmd/...`: sin avisos.
    - `go test` con `-count=1`: `./internal/system`, `./internal/components/engram`, `./internal/components/communitytool`, `./internal/sddstatus`, `./internal/components/agentguidance`, `./internal/update/...` ok; `./internal/cli -run 'Channel|Scope|Install|Background|Sync|Ratchet|Getwd|Guard|Legacy|Protocol'` ok; `./internal/assets ./internal/tui -run 'Getwd|Relay|Guard|Plugin|Runtime|Identity|Forbidden|Productive|Env'` ok.
    - `gofmt -l` sobre los `.go` modificados: salida vacía.
    - Verificado solo en CI: `internal/app` (`app_test.go` espera `AXIOM_CHANNEL` en la ayuda; `project_state_test.go` sin la variable vieja), `e2e/organicruntime` (variables `AXIOM_*`), `internal/filecoord` y `internal/fileprim` (solo `go vet`), `cmd/axiom`.
    - Decisiones: se eliminan `lookupEnvWithGentleAIFallback` y todas las constantes `*GentleAI*`; `system.Getenv` queda con una sola clave. Las pruebas del fallback pasan a comprobar que la variable vieja se ignora y conservan el literal `GENTLE_AI_*` a propósito (cli, engram, sddstatus, update). Se mantienen con literal viejo: `GENTLE_AI_PRODUCTIVE_RUNTIME` y `GENTLE_AI_REVIEW_BINDING` (listas de prohibidas), `GENTLE_AI_STATE_DIR` (prueba negativa) y `GENTLE_AI_CONFIRM_UPDATE` (prueba negativa en `internal/app`). Restos fuera de T2: `gentle-ai.exe` en `ci.yml` (T6), `GENTLE_AI_TELEMETRY` en `internal/components/telemetryruntime/testdata` (fixtures históricas), `{{GENTLE_AI_*}}` (T7), `DISCORD_GENTLE_AI_*` (T6), specs `openspec/specs/axiom-user-state-and-env/spec.md` y `openspec/INDEX.md` REQ-12.3 (T8).
- [x] T3. Estado local y su migración.
  - Rutas activas a `axiom`:
    - `<git common dir>/axiom/sdd-runtime`;
    - `~/.axiom/pi-codegraph.json`;
    - `~/.axiom/cache/…` en los plugins TS;
    - `~/.config/axiom/custom-agents.json`;
    - `.pi/axiom/persona.json`;
    - `<base>/axiom/background-subagents.json`;
    - `.axiom-telemetry-runtime.json` y `.axiom-default-agent.json`;
    - prefijos de temporales.
  - Migración que mueve lo viejo a lo nuevo si lo nuevo no existe. Sustituye los fallbacks de `~/.gentle-ai/state.json`, `~/.gentle-ai/backups`, `~/.gentle-ai/bin` y `.gentle-ai-instance`.
  - Evidencia (commit «refactor(state): rutas de estado axiom con migración desde gentle-ai»; ruta delegada, motivo: protección de contexto del orquestador; escritor único):
    - `go build ./...`: correcto.
    - `go vet` de `./internal/legacyhome ./internal/state ./internal/backup ./internal/sddstatus ./internal/components/... ./internal/agents/... ./internal/assets ./internal/update/... ./internal/cli ./internal/tui/... ./internal/agentbuilder ./internal/app ./internal/dashboard ./internal/opencode ./cmd/...`: sin avisos.
    - `go test -count=1`: ok en `internal/legacyhome`, `state`, `backup`, `sddstatus`, `agentbuilder`, `assets`, `update/...`, `tui/...`, `agents/...`, `components/{persona,telemetryruntime,opencodedefault,uninstall,filemerge,engram,opencodeplugin}`; `internal/cli -run` de las zonas tocadas (Pi, Persona, Backup, Restore, Doctor, Footprint, CodeGraph, Compat, Rollback, Uninstall, ComponentPaths, Sync, Run, Install, Status, Consent): ok.
    - `gofmt -l` sobre los `.go` modificados: salida vacía.
    - Migraciones añadidas (cada una con su prueba: vieja en disco, acaba en la nueva, idempotente, no pisa la nueva):
      - `internal/legacyhome` (`MigrateHome`, llamado al arrancar desde `cmd/axiom/main.go`): `~/.gentle-ai/{state.json,pi-codegraph.json,backups}` a `AxiomDir`; los backups se mueven por id sin pisar y se reescriben `root_dir` y `snapshot_path` del manifiesto; `cache` heredada se descarta; `bin` no se toca; `~/.gentle-ai` se retira solo si queda vacío.
      - `sddstatus.migrateLegacyRuntimeContainer` (al abrir `OpenRuntimeStore`): `<common>/gentle-ai/sdd-runtime` a `<common>/axiom/sdd-runtime`.
      - `sddstatus.migrateLegacyChangeInstanceMarker`: `.gentle-ai-instance` a `.axiom-instance`.
      - `agentbuilder.RegistryPath`: `~/.config/gentle-ai/custom-agents.json` a `~/.config/axiom/`.
      - `persona.InjectPiPersona`: `.pi/gentle-ai/persona.json` a `.pi/axiom/`.
      - `cli.piBackgroundProjectionPlan.Apply`: `<base>/gentle-ai/background-subagents.json` a `<base>/axiom/`.
      - `telemetryruntime.migrateLegacyManifest` y `opencodedefault.migrateLegacyOwnership`: marcadores `.gentle-ai-*` a `.axiom-*`.
    - Rutas renombradas sin migración: plugins TS de OpenCode (`~/.axiom/cache/...`, caché regenerable), prefijos de temporales `axiom-*`.
    - Fallbacks retirados: lectura de `~/.gentle-ai/state.json` (`state.Read`), segunda raíz de backups (`LegacyBackupRootFor`, `BackupOriginGentleAI`), fallback de `doctor` en disco y `.gentle-ai-instance`.
    - Verificado solo en CI: `internal/app`, `internal/dashboard`, `internal/opencode` y `cmd/axiom` (solo `go vet` y `go build`; en `internal/app` se retiró `TestListBackupsKeepsDuplicateIDsRestorableBySelectedRoot`).
    - Decisiones: `LegacyBinDir` y `LegacyOwnershipMarker` de `internal/opencode/background.go` se dejan: el lanzador viejo ya se retira en la activación de OpenCode y el marcador es de T4. `retainedPiResources` y el desinstalador siguen limpiando las rutas viejas (y suman las nuevas `.pi/axiom`).
- [x] T4. Marcadores, plugins y MCP gestionados.
  - Todos los ids de sección pasan a `axiom:`. El reconocimiento de `gentle-ai:` queda solo para sustituir bloques ya escritos.
  - Ids de plugin `axiom.*`, cabeceras `// axiom:managed` y MCP o plugin `axiom-engram(-tools)`, con retirada de las entradas viejas.
  - Reconocimiento de los hooks y comandos `gentle-ai …` solo para retirarlos.
  - Goldens.
  - Evidencia (commit «refactor(sync): marcadores, plugins y MCP con el prefijo axiom»; ruta delegada, motivo: protección de contexto del orquestador; escritor único):
    - `go build ./...`: correcto.
    - `go vet ./internal/components/... ./internal/agentbuilder ./internal/assets ./internal/sddtaskresult ./internal/cli ./internal/app ./internal/opencode ./cmd/...`: sin avisos.
    - `go test -count=1`: ok en `./internal/components/...` (incluido `sdd`, `communitytool`, `engram`, `uninstall`, `persona`, `filemerge`), `./internal/agentbuilder`, `./internal/skillregistry`, `./internal/assets`, `./internal/sddtaskresult`; `./internal/cli -run 'Sync|Persona|Skill|Hook|Preflight|Uninstall|Golden|Pi|CodeGraph|Engram'`: ok.
    - Goldens regenerados con `go test ./internal/components -update` (13 ficheros: solo cambian los marcadores `sdd-session-preflight` y `sdd-model-assignments` a `axiom:`).
    - `gofmt -l` sobre los `.go` tocados: salida vacía.
    - Se escribe ahora con `axiom`: `sdd-session-preflight`, `opencode-background-subagents`, `opencode-desktop-delegation-progress`, `sdd-model-assignments` (assets), `delegation-hard-gates-migration`, `pi-codegraph-*`, `custom-agent:<name>`, ids de plugin OpenCode (`axiom.model-variants`, `axiom.skill-registry`, `axiom.sdd-task-result-artifacts`, `axiom.branch-guard`), cabeceras `// axiom:managed`, plugin Antigravity `axiom-engram` y hooks `axiom-engram-tools`.
    - Migraciones (cada una con su prueba): bloque de preflight del nombre anterior en un prompt preservado de OpenCode sustituido por el actual (`TestMigratePreservedSDDSessionPreflightReplacesTheBlockOfThePreviousName`); bloque de subagentes en segundo plano (`upgradeLegacyOpenCodeBackgroundPolicy`, dos pruebas); bloque de delegación (las pruebas existentes de `ensurePreservedOpenCodeDelegationHardGates` parten ahora de un bloque `gentle-ai:` y exigen un único bloque `axiom:`); bloques Pi de CodeGraph (`TestRenderPiChildReplacesLegacyBlocksWithAxiomBlocks`); bloque `custom-agent` (`TestInjectSDDReference_LegacyBlockIsReplacedByAxiomBlock`); directorio `gentle-ai-engram` de Antigravity retirado al inyectar, idempotente (`TestInjectAntigravityWritesMCPToCLIConfig`); sección de CodeGraph reconocida con ambos prefijos en la huella de doctor.
    - Plugins OpenCode: la verificación de propiedad compara el hash de los bytes instalados, así que `opencode_plugin_digests.go` suma los resúmenes de las revisiones sustituidas (los cuatro plugins v2, `sdd-task-result-artifacts` v1 y, de paso, las de `model-variants` v1 y v2 que T3 cambió sin registrar). Sin eso `axiom sync` rechazaría el plugin ya instalado por «propiedad no verificada».
    - Retirada la variante `Gentle AI SDD preflight `: plugins TS (v1 y v2), hook `internal/cli/sdd_preflight_hook.go` y sus pruebas (nueva prueba que exige que el prefijo viejo no cree autoridad).
    - Quedan como solo reconocimiento (verificado con `git grep`): marcadores `legacy*` de `sdd`, `communitytool`, `agentbuilder` y `skillregistry`; `filemerge` y los limpiadores de `uninstall`/`persona` (`gentle-ai:` genérico, `name: Gentle AI Persona`); comandos `gentle-ai skill-registry refresh`, `sdd-preflight-hook`, `telemetry runtime` y `review stop-hook` (retirada de hooks); `gentle-ai/sdd` en `managed-by` (solo se purga); `LegacyOwnershipMarker` del lanzador de OpenCode (se lee para retirar el lanzador viejo); `removePiManagedSections`; nombres de ficheros de prompt viejos (`gentle-ai.mdc`, `gentle-ai.md`, `gentle-ai.instructions.md`). Ninguna ruta escribe `gentle-ai skill-registry refresh`: `ensureClaudeSkillRegistryHook` y `ensureCodexSkillRegistryHook` ya escribían `axiom`.
    - Decisión: se quita el reconocimiento de `gentle-ai:sdd-model-assignments` en `inject.go` y `profiles.go` (solo operaban sobre assets embebidos, que ya llevan `axiom:`). `ManagedCodeGraphGuidanceFiles` y `hasCodeGraphGuidance` solo miraban `gentle-ai:` aunque la guía se escribe con `axiom:`: ahora reconocen ambos.
    - Verificado solo en CI: `internal/app` (`app_test.go`, literales de marcadores ajustados), `internal/opencode`, `internal/dashboard` y `e2e/...` (solo `go vet` y `go build`).
    - Restos fuera de T4: comentarios `gentle-ai` en `sync.go`, `run.go`, `persona/inject.go`, `sdd/inject.go:65`, `model/types.go` y los comentarios de `plugins/model-variants.ts` y `plugins/skill-registry.ts` (T7; cambiar esos dos ficheros TS exige añadir antes su hash a `opencode_plugin_digests.go`); esquemas `gentle-ai.*` (T5); fixtures `telemetryruntime/testdata` (históricas).
    - Docs y specs que citan lo cambiado (para T8): `openspec/specs/sdd-preflight-configuration/spec.md`, `PRD-AGENT-BUILDER.md`, `docs/prd-opencode-profiles.md`.
- [x] T5. Esquemas y contratos JSON.
  - `gentle-ai.*` pasa a `axiom.*`: runtime ledger, governance, archive-compose, capability manifest, telemetry ownership, opencode default y review-store-lock.
  - Se aceptan al leer los esquemas viejos de ficheros ya escritos.
  - Se quitan los `Legacy*` de sdd-status y consent y el `$id` `gentle-ai.dev` del contrato de consentimiento.
  - Evidencia (commit «refactor(sddstatus): esquemas axiom con lectura de los gentle-ai persistidos»; ruta delegada, motivo: protección de contexto del orquestador; escritor único):
    - `go build ./...`: correcto.
    - `go vet ./internal/sddstatus ./internal/kickoff ./internal/agents/... ./internal/components/... ./internal/fileprim ./internal/cli ./internal/sddpublish ./internal/assets ./cmd/... ./internal/app ./internal/dashboard ./internal/opencode ./e2e/...`: sin avisos.
    - `go test -count=1`: ok en `./internal/sddstatus`, `./internal/kickoff`, `./internal/agents/...`, `./internal/components/...` (un fallo intermitente de `communitytool` por timeout de la sonda MCP pasó al repetirlo solo), `./internal/fileprim`, `./internal/sddpublish`, `./internal/assets`; `./internal/cli -run SDD|Gate|Consent|Status|Archive|Attempt|Kickoff|Publish|Capability|Routing|Telemetry`: ok.
    - `gofmt -l`: vacío.
    - Se escribe ahora con `axiom.*`: runtime ledger (`sdd-runtime-status`, `-record`, dominios `change-identity` y `grant-request`), `sdd-governance(.gate)`, `sdd-archive-compose`, `agent-capability-manifest`, `work-routing`, `implementation-routing`, `telemetry-runtime-ownership`, `opencode-default-agent`, `review-store-lock`. Los digests de `capabilitymanifest` fijados en `manifest_test.go` se regeneraron con los valores nuevos (17 por agente más el de routing). Digest del `consent.schema.json` fijado en `consent_contract_test.go` actualizado.
    - Migraciones de lectura (cada una con prueba): ledger con registros `gentle-ai.sdd-runtime-record/v1` se lee y cada registro valida su digest de petición con el dominio con el que se escribió (`TestRuntimeLedgerReadsGrantWrittenWithLegacySchemaAndAppendsAxiomSchema`: autoridad vigente y grant nuevo encadenado con esquema axiom; `TestRuntimeLedgerLegacySchemaRecordRejectsAxiomDomainDigest`); directorio de cambio no-slug con digest de identidad heredado se mueve al nuevo al abrir el almacén, idempotente y sin pisar (`TestOpenRuntimeStoreMigratesLegacyChangeIdentityDirectory`); `opencodedefault.readOwnership` y el manifiesto de `telemetryruntime` aceptan el esquema heredado (`TestReadOwnershipAcceptsLegacySchemaAndRewritesWithAxiomSchema`; el de telemetría lo cubren las pruebas de retirada de `cli` y `uninstall`, que parten de un manifiesto `gentle-ai.*`).
    - Retirados: `LegacySchemaName`, `LegacyStatusContractV2`, `LegacySDDIntegrationConsentSchema`, `LegacySDDIntegrationContractV1`, prefijos `gentle-ai sdd-attempt grant` y `gentle-ai sdd-status` del consentimiento, y los alias `Axiom*` duplicados de consent y governance. `$id` del contrato de consentimiento: `urn:axiom:contracts:sdd-integration:v1:consent` (una sola constante `SDDIntegrationConsentSchemaID`). `gates.yaml` y el kickoff no persisten el esquema de governance (es un sobre de pregunta emitido), por lo que ningún digest de gate cambia.
    - Decisión: se actualiza `internal/assets/skills/_shared/sdd-status-contract.md` (`schemaName` y contrato v2 a `axiom.sdd-status`), porque su instrucción de reintento citaba el contrato heredado que ya no se acepta; sus pruebas se ajustaron. Limitación: un reintento idempotente de un grant escrito con el dominio viejo, con el mismo `request-id`, computa el digest nuevo y daría conflicto (caso marginal; no afecta a la lectura ni a la autoridad vigente).
    - Verificado solo en CI: `internal/app`, `internal/dashboard`, `internal/opencode`, `e2e/...` y `cmd/axiom` (solo `go vet` y `go build`).
    - Docs y specs que citan lo cambiado (para T8): `docs/assets/diagrams/deterministic.svg`, `openspec/specs/axiom-distribution-identity/spec.md`, `openspec/specs/sdd-research/spec.md`.
- [ ] T6. Release y CI.
  - `axiom-release-provenance-v1.tar.gz` y esquema `axiom.release-provenance/v1` en `.goreleaser.yaml`, `releasepolicy`, `releaseprovenance` y los scripts de verificación y promoción.
  - Nombres de clave y marcador en los workflows, `axiom.exe` en CI, imágenes e2e y comentario no confiable de la firma.
  - Revisar la consulta GraphQL de `scripts/promote-stable-preflight.sh` al upstream.
- [ ] T7. Nombres internos.
  - Placeholders `{{GENTLE_AI_*}}` de las plantillas, identificadores Go, comentarios y nombres de pruebas.
  - `.gitignore`, `scripts/deadcode-ratchet.sh`, `e2e/` y la skill retirada `gentle-ai-bench`.
- [ ] T8. Documentación y specs vivas.
  - Docs vivas y `.md` raíz, salvo atribución e historia.
  - `## Spec viva` con el delta de las specs que fijan nombres o compatibilidad gentle-ai.
  - `axiom archive integrate --odd retirar-gentle-ai` en el commit que cierra.

## Progreso

T1 hecha (commit «fix(update): autoupdate y registro con el nombre axiom»). T2 hecha (commit «refactor(env): retirar las variables GENTLE_AI_*»). T3 hecha (commit «refactor(state): rutas de estado axiom con migración desde gentle-ai»). T4 hecha (commit «refactor(sync): marcadores, plugins y MCP con el prefijo axiom»). T5 hecha (commit «refactor(sddstatus): esquemas axiom con lectura de los gentle-ai persistidos»). T6-T8 sin empezar.

## Siguiente paso

T6.
