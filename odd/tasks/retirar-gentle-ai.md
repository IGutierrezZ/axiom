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
- [x] T6. Release y CI.
  - `axiom-release-provenance-v1.tar.gz` y esquema `axiom.release-provenance/v1` en `.goreleaser.yaml`, `releasepolicy`, `releaseprovenance` y los scripts de verificación y promoción.
  - Nombres de clave y marcador en los workflows, `axiom.exe` en CI, imágenes e2e y comentario no confiable de la firma.
  - Revisar la consulta GraphQL de `scripts/promote-stable-preflight.sh` al upstream.
  - Evidencia (commit «ci(release): nombres axiom en la procedencia, la firma y la CI»; ruta delegada, motivo: protección de contexto del orquestador; escritor único):
    - `go build ./...`: correcto.
    - `go vet ./internal/releasepolicy/... ./internal/releaseprovenance/... ./internal/releaseprovenancecmd/... ./internal/update/... ./e2e/...`: sin avisos.
    - `go test -count=1 ./internal/releasepolicy/... ./internal/releaseprovenance/... ./internal/releaseprovenancecmd/... ./internal/update/...`: ok (incluye `update/upgrade`).
    - `bash -n` sobre `promote-stable-preflight.sh`, `verify-release-assets.sh`, `release-signing-preflight.sh` y `e2e/docker-test.sh`: correcto. `gofmt -l`: vacío.
    - Renombrado: `axiom-release-provenance-v1.tar.gz` (goreleaser, policy, scripts y pruebas), esquemas `axiom.release-provenance/v1` y `/local-build` (sin lectura del viejo: las releases ya publicadas no se re-verifican), comentario de firma «signature from axiom release» y el comentario `-c` del canario de `release-signing-preflight.sh`, ficheros del runner `axiom-release.key`, `axiom-release-signing.pub` y `axiom-release-policy-snapshot-start`, `axiom.exe` en `ci.yml` y `windows-full-suite.yml`, imágenes `axiom-e2e-*` (CI y `docker-test.sh`), cabeceras de los `Dockerfile.*` y secretos `DISCORD_AXIOM_{RELEASES,BETA,ISSUES}_WEBHOOK`.
    - Decisión GraphQL: `promote-stable-preflight.sh` consultaba si la release estable ya existía en `Gentleman-Programming/gentle-ai`, no en este repo (heredado del fork, errónea: la comprobación de recuperación solo tiene sentido sobre el propio repo). Ahora usa `$GITHUB_REPOSITORY` (ya validado como `IGutierrezZ/axiom`) como `owner`/`name`.
    - Se deja a propósito: la regex de aviso retirado de `ci.yml` (comprueba que el binario canónico no emita avisos del nombre antiguo; es una guarda, no un identificador), comentarios históricos de `deadcode-ratchet.sh`, `.goreleaser.yaml:85` (homebrew-tap upstream) y `policy.go:298` (binario retirado), y fixtures negativas de `windows_distribution_policy_test.go` (T7).
    - Acción manual del usuario: si alguna vez activa `discord-notifications.yml`, debe renombrar en GitHub los secretos a `DISCORD_AXIOM_RELEASES_WEBHOOK`, `DISCORD_AXIOM_BETA_WEBHOOK` y `DISCORD_AXIOM_ISSUES_WEBHOOK`.
    - Verificado solo en CI y en el preflight de la próxima release: los workflows (`ci.yml`, `release.yml`, `promote-stable-rc.yml`, `windows-full-suite.yml`), `goreleaser`, `scripts/verify-release-assets.sh` y las pruebas de release que exigen `AXIOM_TEST_BUILD_BINARIES=1`.
    - Docs y specs que citan lo cambiado (para T8): `docs/release-signing.md` (`gentle-ai-release.pub/.key`), `docs/docker-e2e-testing.md`, y las specs de release en `openspec/specs/*` que fijen el archivo de procedencia o el esquema (buscar `gentle-ai-release-provenance` y `gentle-ai.release-provenance`).
- [x] T7. Nombres internos.
  - Placeholders `{{GENTLE_AI_*}}` de las plantillas, identificadores Go, comentarios y nombres de pruebas.
  - `.gitignore`, `scripts/deadcode-ratchet.sh`, `e2e/` y la skill retirada `gentle-ai-bench`.
  - Evidencia (commit «refactor: nombres internos, plantillas y pruebas sin gentle-ai»; ruta delegada, motivo: protección de contexto del orquestador; escritor único):
    - `go build ./...` y `go vet ./...`: correctos. `gofmt -l` sobre los `.go` modificados desde `3c55c493`: vacío. `bash -n` de `e2e/lib.sh`, `e2e/e2e_test.sh` y `scripts/deadcode-ratchet.sh`: correcto.
    - `go test -count=1` ok en los paquetes tocados permitidos (`agents/*`, `assets`, `backup`, `branch*`, `cli`, `components/*`, `consentenvelope`, `doctor`, `installcmd`, `model`, `pathidentity`, `releaseprovenancecmd`, `sddstatus`, `semantic`, `skillregistry`, `state`, `system`, `tui/...`, `update/...`, `verify`, `versions`). `branch/engine` y `cli` agotaron los 10 min por defecto al lanzarse en paralelo (carga, no fallo); repetidos con `-timeout 45m -p 1`: ok (616 s y 583 s).
    - Recuento de líneas (`git grep -n -i 'gentle.ai|gentle_ai|gentleai'` en `cmd internal e2e scripts .gitignore`): antes 1.484, después 725 (121 fuera de pruebas).
    - Cambios: `{{GENTLE_AI_*}}` pasa a `{{AXIOM_*}}` (secciones SDD, id de agente, ciclo de investigación, autoridad del preflight de Windsurf; 351 líneas en 22 ficheros); `shippedPluginDigestGentleAIV3` pasa a `shippedPluginDigestUpstreamV3` (mismo valor); comentarios y doc-strings de producto pasan a Axiom; `.gitignore` sin `/gentle-ai` ni `cmd/gentle-ai/*`; `e2e/lib.sh` y `e2e_test.sh` solo axiom (`.axiom`, `.axiom-e2e`, marcador `axiom:`); `semantic/engine.go` ya no trata `gentle-ai` como import interno (heurística heredada, no atribución); `status.go` y `run.go` citan `axiom.sdd-integration.consent` y `axiom:sdd-orchestrator`, y su línea congelada en `.refusal-ratchet-baseline.txt`; e2e `organicruntime` sin shim ni copia `gentle-ai` (método `axiom`, contenedor `<common>/axiom`, solo esquema y continuación `axiom`); pruebas de `app`, `tui`, `update` y `cli` que corrompían `~/.gentle-ai/state.json` ahora usan `~/.axiom`.
    - Plugins TS: se cambian solo comentarios de `plugins/model-variants.ts` y `plugins/skill-registry.ts` (v1); antes se añaden a `opencode_plugin_digests.go` los hashes de los bytes anteriores (`ad88348e…` y `9fb2486c…`) y se ajusta el recuento de `TestHistoricalOpenCodePluginDigestTable` (6 y 8).
    - Pruebas: `retired_review_absence_test.go` comprueba los tres placeholders SDD concretos (`{{AXIOM_` es ahora también el prefijo de `{{AXIOM_SKILLS_DIR}}`); el ratchet de refusals y `documented_invocation_test` aceptan solo `axiom `.
    - Restos por categoría:
      - (a) migración y retirada (código y sus pruebas): constantes `legacy*` y limpiadores de marcadores `<!-- gentle-ai:` (`sdd/inject.go`, `filemerge`, `uninstall`, `persona`, `communitytool`, `agentbuilder`, `skillregistry`); rutas viejas que se mueven (`legacyhome`, `.gentle-ai-*`, `.pi/gentle-ai`, `<common>/gentle-ai`, `~/.config/gentle-ai`); esquemas `gentle-ai.*` que se leen (`sddstatus`, `opencodedefault`, `telemetryruntime`); comandos viejos que se retiran de hooks (`gentle-ai skill-registry refresh`, `telemetry runtime`, `sdd-preflight-hook`, `review stop-hook`); prompts viejos (`gentle-ai.mdc`, `.md`, `.instructions.md`); `__managed_by` `gentle-ai/sdd`; `LegacyOwnershipMarker` y `LegacyBinDir` de OpenCode; skill retirada `gentle-ai-bench` (`skills/retired.go`: sigue retirando copias reales); `name: Gentle AI Persona`.
      - (b) pruebas negativas: nombre `gentle-ai` rechazado por identidad de self-tool (`update`, `tui`, `cmd/axiom`, `doctor`), `GENTLE_AI_*` ignoradas, listas de prohibidos de `assets`, `cmd/gentle-ai` inexistente, pistas de residuo en `branch`. Fixtures de `windows_distribution_policy_test.go`: el artefacto `dist/gentle-ai_windows_amd64_v1/gentle-ai.exe` es negativa legítima (b); la llamada `gh api` apunta ahora a `IGutierrezZ/axiom`.
      - (c) atribución upstream: URLs y rutas `Gentleman-Programming/gentle-ai` en `update/check_test.go`, `tui`, `engram/inject.go`, `model/selection.go`; fixtures `Owner: Gentleman-Programming` con `Repo: "gentle-ai"` (`update/upgrade/*_test.go`, `source_install_test.go`); módulo `github.com/gentleman-programming/gentle-ai` en `module_test.go`; digest upstream del plugin de telemetría; `Project: "gentle-ai"` en fixtures de Engram de `sddstatus` (nombre de proyecto de fixture; cambiarlo rompe el descubrimiento).
      - (d) históricos: `telemetryruntime/testdata`, comentarios de `scripts/deadcode-ratchet.sh:34-35` (historia del binario retirado), `releasepolicy/policy.go:298`.
    - Verificado solo en CI: `internal/app` (selfupdate_test con `.axiom`, app_test sin alternativa `gentle-ai help`, pruebas de invocaciones documentadas), `internal/dashboard`, `internal/opencode`, `e2e/...` y `cmd/axiom` (solo `go vet` y `go build`).
    - Decisión: se prefirió renombrar solo mensajes, comentarios, nombres de caso y fixtures sin acoplamiento; los pares `Owner`/`Repo` y los nombres de proyecto Engram se dejan porque cambiar solo uno rompe URLs y descubrimiento.
- [x] T8. Documentación y specs vivas.
  - Docs vivas y `.md` raíz, salvo atribución e historia.
  - `## Spec viva` con el delta de las specs que fijan nombres o compatibilidad gentle-ai.
  - `axiom archive integrate --odd retirar-gentle-ai` en el commit que cierra.
  - Evidencia (commit «docs: documentación y specs vivas sin los nombres gentle-ai»; ruta delegada, motivo: protección de contexto del orquestador; escritor único):
    - `axiom archive integrate --odd retirar-gentle-ai`: «Impacto en la spec: delta»; compuestas `axiom-distribution-identity`, `axiom-user-state-and-env`, `engram-protocol-injection`, `sdd-preflight-configuration` y `sdd-research`; escrito `openspec/specs/*` de esas cinco e `openspec/INDEX.md` (534 requisitos, 2153 escenarios). REQ-12.3 se retira y lo sustituye REQ-12.4 (el compositor aplica MODIFIED antes que RENAMED); REQ-20.17 nuevo (procedencia, firma y CI de release).
    - `go build ./...`: correcto. `go test -count=1 ./internal/livespec ./internal/openspecdoc ./internal/reqid`: ok.
    - Docs: `README.md`, `PRD-AGENT-BUILDER.md`, `docs/{agents,components,docker-e2e-testing,opencode-profiles,pi,platforms,prd-opencode-profiles,release-signing,rollback,testing-agents-deterministically,usage}.md` y `docs/assets/diagrams/deterministic.svg`.
    - Restos justificados: atribución y fork (`README`, `AGENTS`, `GEMINI`, `openspec/config.yaml`, `docs/architecture`, `docs/consulting`, `TRADEMARKS`, `CONTRIBUTORS`); `PRD.md` (PRD original del upstream, histórico); backlog (`docs/plan/backlog-2026-11`, análisis histórico del upstream); `docs/telemetry.md` (fichero del upstream que Axiom no lee); rutas del paquete externo gentle-pi (`.pi/gentle-ai/models.json`, `support/`, `GENTLE_PI_CONFIG_HOME`); migración vigente (`~/.gentle-ai` en `usage`, `rollback`, `agents`, retirada de `gentle-ai.mdc`/`.md`/`.instructions.md`); guarda de `ci.yml`; `discord-notifications.yml` (desactivado, heredado); specs de historia o negativas (`axiom-tui-branding`, `visual-decoupling`, `tui-spanish-localization`, REQ-20.11, REQ-46.2, REQ-46.4, REQ-11.4, REQ-13.4, REQ-22.3, `change-branch-lifecycle`, `organic-agent-trigger-rules`, y el Propósito de `engram-protocol-injection`, que un delta no puede cambiar).

## Spec viva

### Capacidad: axiom-distribution-identity

```openspec
## MODIFIED Requirements

### Requirement: Taxonomía de interoperabilidad vs. identidad (REQ-20.7)

El sistema DEBE clasificar cada superficie de nombre publicado en exactamente una de dos categorías: interoperabilidad (contrato de cable consumido por otro software, cuyo renombrado rompe a consumidores) o identidad (superficie leída por una persona, renombrable sin romper ningún contrato). El namespace de protocolo bajo `contracts/**` y los marcadores de contenido gestionado que un componente Go analiza de forma literal DEBEN clasificarse como interoperabilidad. El instalador, el tap de Homebrew, las compuertas de release, el nombre de binario citado en la documentación y los nombres de servicio de despliegue DEBEN clasificarse como identidad.
(Previously: el escenario usaba el esquema `contracts/review-integration/v2/schemas/status.schema.json`, que se retira con RDD (C-01-e); se sustituye por el esquema de consentimiento de `contracts/sdd-integration`, que permanece.)

#### Scenario: El namespace de contracts/** se clasifica como interoperabilidad
- **DADO** el fichero `contracts/sdd-integration/v1/schemas/consent.schema.json` con `"$id": "urn:axiom:contracts:sdd-integration:v1:consent"`
- **CUANDO** se clasifica esa superficie
- **ENTONCES** se clasifica como interoperabilidad
- **Y** ninguna tanda de identidad la modifica
(Previously: el escenario citaba el `$id` `https://gentle-ai.dev/contracts/sdd-integration/v1/schemas/consent.schema.json`, retirado al pasar el contrato de consentimiento a la identidad `axiom`.)

### Requirement: Namespace de protocolo en contracts/** sin cambios (REQ-20.10)
El sistema NO DEBE modificar el namespace de protocolo bajo `contracts/**` fuera de un cambio de contrato deliberado: los valores `$id` (el del consentimiento es `urn:axiom:contracts:sdd-integration:v1:consent`), los prefijos de contrato (`axiom.sdd-integration/v1`, `axiom.sdd-integration.consent/v1` y equivalentes) y los esquemas asociados DEBEN permanecer byte a byte idénticos a la versión publicada con la identidad `axiom`, cuyo digest fija `internal/consentenvelope/consent_contract_test.go`. Ningún nombre `gentle-ai` DEBE aparecer en un `$id` ni en un prefijo de contrato.

#### Scenario: contracts/** permanece sin cambios
- **DADO** el estado de `contracts/**/*.schema.json`
- **CUANDO** se auditan los esquemas
- **ENTONCES** cada fichero bajo `contracts/**` coincide con el digest fijado en la prueba del contrato
- **Y** ningún `$id` ni prefijo de contrato contiene `gentle-ai`
- **Y** los agentes y demás consumidores continúan interoperando con el formato publicado
(Previously: el `$id` y los prefijos de ejemplo eran los del namespace `gentle-ai` y el escenario exigía identidad byte a byte sin referencia al digest del contrato.)

### Requirement: Binario gentle-ai retirado y shim de crosslane conservado (REQ-20.11)

El sistema NO DEBE compilar ni publicar el binario `gentle-ai`: el paquete `cmd/gentle-ai` y el build `gentle-ai-deprecated` de `.goreleaser.yaml` se retiraron por decisión de producto (ODD `seguimientos-restos-upstream`). El shim ejecutable `gentle-ai` de `scripts/crosslane` ya no existe en el repositorio.

#### Scenario: El release publica solo el binario canónico
- **DADO** la configuración de release `.goreleaser.yaml`
- **CUANDO** se resuelven sus builds
- **ENTONCES** el único build es el que compila `./cmd/axiom` como `axiom`
- **Y** ningún build, archivo ni compuerta de release referencia `cmd/gentle-ai` ni un binario `gentle-ai`
(Previously: el requisito obligaba a conservar el shim `gentle-ai` de `scripts/crosslane/battery.go` y `host.go`; esos ficheros ya no existen y ningún artefacto del producto usa el nombre `gentle-ai`.)

### Requirement: Raíz de respaldos resuelta exclusivamente a través de internal/backup (REQ-20.13)

Todo escritor de producción que cree o localice el directorio raíz de respaldos DEBE resolverlo invocando la función de resolución canónica exportada de `internal/backup` (`backup.BackupRootFor(homeDir)`), y NO DEBE construir esa ruta mediante un literal `filepath.Join` propio. Los respaldos que sobrevivan bajo `~/.gentle-ai/backups` DEBEN moverse una sola vez, en el arranque, a la raíz de respaldos de Axiom sin pisar los existentes y reescribiendo `root_dir` y `snapshot_path` de su manifiesto; `ListBackups` NO DEBE leer ninguna segunda raíz.

#### Scenario: Un escritor de producción resuelve la raíz a través del paquete
- **DADO** cualquier componente de backup, sync, run, upgrade o uninstall
- **CUANDO** necesita el directorio raíz de respaldos
- **ENTONCES** lo obtiene invocando la función de resolución canónica de `internal/backup`
(Previously: los respaldos de `~/.gentle-ai/backups` seguían siendo legibles por `ListBackups` sin migración obligatoria.)

## ADDED Requirements

### Requirement: Procedencia, firma y CI de release con identidad axiom (REQ-20.17)

El archivo de procedencia de release DEBE llamarse `axiom-release-provenance-v1.tar.gz` y sus esquemas DEBEN ser `axiom.release-provenance/v1` y `axiom.release-provenance/local-build`. Las claves de firma del runner DEBEN llamarse `axiom-release.key` y `axiom-release.pub`, el ejecutable de CI en Windows `axiom.exe` y las imágenes de los e2e `axiom-e2e-*`. `.goreleaser.yaml`, `internal/releasepolicy`, `internal/releaseprovenance` y los scripts de verificación y promoción NO DEBEN esperar ni producir ningún nombre `gentle-ai`. `scripts/promote-stable-preflight.sh` DEBE consultar la existencia de la release estable en el repositorio propio (`$GITHUB_REPOSITORY`).

#### Scenario: La procedencia y la firma usan la identidad axiom
- **DADO** una release simulada con los artefactos `axiom_*`
- **CUANDO** se ejecutan la verificación de assets y las pruebas de `internal/releasepolicy` e `internal/releaseprovenance`
- **ENTONCES** esperan `axiom-release-provenance-v1.tar.gz` con el esquema `axiom.release-provenance/v1`
- **Y** ningún fichero de política ni script de release contiene un nombre de artefacto `gentle-ai`

#### Scenario: La promoción consulta el repositorio propio
- **DADO** `GITHUB_REPOSITORY=IGutierrezZ/axiom`
- **CUANDO** `promote-stable-preflight.sh` comprueba si la release estable ya existe
- **ENTONCES** la consulta usa ese propietario y nombre de repositorio
```

### Capacidad: axiom-user-state-and-env

```openspec
## MODIFIED Requirements

### Requirement: Migración defensiva desde ~/.gentle-ai (REQ-12.2)
Al arrancar, el sistema DEBE mover una sola vez el estado heredado de `~/.gentle-ai/` a `~/.axiom/`: `state.json`, `pi-codegraph.json` y `backups/` (cada respaldo por su id, sin pisar los existentes). Solo mueve lo que no existe ya en el destino, es idempotente, descarta la caché heredada, no toca `bin/` y retira `~/.gentle-ai` únicamente si queda vacío. Tras migrar, ninguna ruta de lectura DEBE consultar `~/.gentle-ai/`.

#### Scenario: Migración automática en el primer arranque
- **DADO** un equipo con `~/.gentle-ai/state.json` configurado y sin `~/.axiom/state.json`
- **CUANDO** se invoca cualquier comando de Axiom
- **ENTONCES** el archivo se mueve a `~/.axiom/state.json`
- **Y** la ejecución continúa normalmente utilizando el nuevo archivo

#### Scenario: La migración no pisa el destino y es idempotente
- **DADO** un `~/.axiom/state.json` existente y un `~/.gentle-ai/state.json`
- **CUANDO** se repite el arranque
- **ENTONCES** `~/.axiom/state.json` conserva su contenido
- **Y** la segunda ejecución no cambia nada
(Previously: la migración copiaba solo `state.json` al resolver el estado y las lecturas conservaban `~/.gentle-ai/state.json` como alternativa.)

## REMOVED Requirements

### Requirement: Precedencia de variables AXIOM_* sobre GENTLE_AI_* (REQ-12.3)
(Reason: las variables `GENTLE_AI_*` ya no se leen como respaldo; lo sustituye REQ-12.4.)

## ADDED Requirements

### Requirement: Variables de entorno exclusivamente AXIOM_* (REQ-12.4)
El sistema DEBE leer la configuración del entorno únicamente de variables con prefijo `AXIOM_*`. Las variables `GENTLE_AI_*` NO DEBEN leerse como alternativa ni tener efecto alguno.

#### Scenario: Variable AXIOM_* toma el valor
- **DADO** `AXIOM_NO_ANIMATION=1` definido en el entorno
- **CUANDO** se consulta la política de animación de la interfaz
- **ENTONCES** el sistema adopta el valor `1` (animación desactivada)

#### Scenario: GENTLE_AI_* se ignora
- **DADO** únicamente `GENTLE_AI_OPENCODE_BACKGROUND_SUBAGENTS=true` definido en el entorno
- **CUANDO** se resuelve la intención de subagentes en segundo plano
- **ENTONCES** el sistema no adopta ese valor
```

### Capacidad: engram-protocol-injection

```openspec
## MODIFIED Requirements

### Requirement: Single canonical Codex protocol asset

The system MUST render the Codex AGENTS.md protocol section, the `model_instructions_file`, and the compact prompt from one canonical source asset rather than three independently maintained files. Codex-only content (the PASSIVE CAPTURE section) MUST be preserved in the rendered `model_instructions_file` output. No normative rule present in today's rendered Codex outputs MUST be lost after consolidation.

#### Scenario: Codex outputs render from one source

- GIVEN the canonical protocol asset
- WHEN axiom renders the Codex AGENTS.md section, `model_instructions_file`, and compact prompt
- THEN all three outputs MUST derive from that single canonical asset

#### Scenario: PASSIVE CAPTURE content survives consolidation

- GIVEN the canonical asset includes Codex-only PASSIVE CAPTURE content
- WHEN the `model_instructions_file` is rendered
- THEN the rendered output MUST include the PASSIVE CAPTURE section unchanged

#### Scenario: No normative rule regression

- GIVEN the pre-consolidation rendered Codex outputs (AGENTS.md section, `model_instructions_file`, compact prompt)
- WHEN the same outputs are rendered after consolidation
- THEN every normative rule (MUST/SHALL/SHOULD statement) present before consolidation MUST still be present after
(Previously: the scenarios named the product `gentle-ai`.)

### Requirement: Conditional per-adapter section slimming

The system MUST slim an adapter's system-prompt Engram protocol section ONLY when a redundant channel delivering equivalent protocol content is verified for that adapter's runtime. The set of adapters with a verified redundant channel is owned by design. For any adapter where no redundant channel is verified, the system MUST inject the full section unchanged. When a section is slimmed, the slim variant MUST include a pointer directing the model to where the full protocol content lives.

#### Scenario: Verified adapter gets the slim section

- GIVEN an adapter is on the design-verified redundant-channel list
- WHEN axiom injects the Engram protocol section for that adapter
- THEN it MUST inject the slim variant
- AND the slim variant MUST contain a pointer to the full protocol location

#### Scenario: Unverified adapter keeps the full section

- GIVEN an adapter is NOT on the design-verified redundant-channel list
- WHEN axiom injects the Engram protocol section for that adapter
- THEN it MUST inject the full section, unchanged from today's content
(Previously: the scenarios named the product `gentle-ai`.)

### Requirement: Version-gated protocol-verbosity forwarding to engram setup

When invoking `engram setup`, the system MUST detect whether the installed engram binary supports a protocol-verbosity option. When supported, it MUST pass the configured verbosity value. When not supported, it MUST invoke setup exactly as today, without any unknown flag, and MUST continue using full-text injection. Setup invocation MUST NOT fail as a result of this detection or forwarding.

#### Scenario: Supported binary receives the verbosity flag

- GIVEN the installed engram binary supports the protocol-verbosity option
- WHEN axiom invokes `engram setup`
- THEN it MUST pass the configured verbosity value to the invocation

#### Scenario: Unsupported binary is invoked unchanged

- GIVEN the installed engram binary does not support the protocol-verbosity option
- WHEN axiom invokes `engram setup`
- THEN it MUST invoke setup with no unknown flags, identical to current behavior
- AND full-text injection MUST remain in effect

#### Scenario: Setup never fails due to verbosity detection

- GIVEN any installed engram binary, supported or unsupported
- WHEN axiom performs verbosity detection and forwarding around `engram setup`
- THEN the setup invocation MUST NOT fail as a result of that detection or forwarding logic
(Previously: the scenarios named the product `gentle-ai`.)
```

### Capacidad: sdd-preflight-configuration

```openspec
## MODIFIED Requirements

### Requirement: Cada Pregunta del Pre-Vuelo Lleva un Marcador por Nombre de Grupo y se Toleran los Marcadores Numerados Heredados (REQ-28.16)

El marcador de cada pregunta del pre-vuelo DEBE ser nominal: `<prefijo> <id de grupo>:`, con el id estable en minúsculas (`pace` para el único grupo requerido vigente) y el prefijo `Axiom SDD preflight`; el prefijo anterior `Gentle AI SDD preflight` ya no se acepta. Durante la transición, hasta que la persona ejecute `axiom sync`, el hook del runtime DEBE seguir aceptando los marcadores numerados heredados (`N/3`), con el mapeo posicional heredado 1 → `pace`, 2 → `artifacts` y 3 → `pr-strategy`; un cuestionario heredado puede mezclarse con marcadores nominales siempre que cada grupo requerido tenga exactamente un marcador. Las respuestas de los grupos heredados `artifacts` (almacén) y `pr-strategy` (estrategia de PR: `Ask me`, `Single PR`, `Auto`) DEBEN aceptarse e ignorarse: nunca determinan el almacén ni la estrategia de entrega del bloque derivado, y no se valida su contenido. Un grupo requerido con más de un marcador, o con ninguno, DEBE denegarse con una causa que nombre el grupo.
(Previously: `pr-strategy` era un grupo requerido y solo `artifacts` era heredado; ahora ambos son heredados, tolerados e ignorados)

#### Scenario: Marcador nominal aceptado con cualquiera de los dos prefijos

- **DADO** un pre-vuelo cuyas preguntas llevan el marcador nominal `pace` con el prefijo vigente, y otro con el prefijo heredado
- **CUANDO** el hook los corrobora
- **ENTONCES** ambos establecen la autoridad del pre-vuelo

#### Scenario: Los marcadores numerados heredados toleran un cuestionario antiguo de tres preguntas

- **DADO** un proyecto `monorepo-embedded` y una llamada heredada con los marcadores `1/3` (ritmo), `2/3` (almacén) y `3/3` (estrategia de entrega) y sus tres respuestas
- **CUANDO** el hook la corrobora
- **ENTONCES** la autoridad se establece
- **Y** las respuestas de almacén y de estrategia de PR se ignoran y el bloque derivado dice `Artifact store: hybrid` y `Delivery strategy: single-pr`

#### Scenario: Una respuesta heredada `Ask me` o `Auto` no produce una entrega distinta de `single-pr`

- **DADO** un proyecto `monorepo-embedded` y una llamada con el grupo `pace` y el grupo heredado `pr-strategy` respondido `Ask me` (y otra con `Auto`)
- **CUANDO** el hook las corrobora
- **ENTONCES** ambas establecen la autoridad
- **Y** el bloque derivado de ambas dice `Delivery strategy: single-pr`, nunca `ask-on-risk` ni `auto-chain`

#### Scenario: Un cuestionario heredado en multirepo ignora ritmo, almacén y estrategia

- **DADO** un proyecto `multirepo` y una llamada heredada con los tres grupos respondidos, entre ellos el ritmo `Automatic`, el almacén `Engram` y la estrategia `Auto`
- **CUANDO** el hook la corrobora
- **ENTONCES** el despacho se permite
- **Y** el bloque derivado dice `Pace: interactive`, `Artifact store: hybrid` y `Delivery strategy: single-pr`

#### Scenario: Un grupo requerido con el marcador repetido se deniega nombrándolo

- **DADO** un proyecto `monorepo-embedded` y un pre-vuelo en el que el marcador del grupo `pace` aparece en dos preguntas
- **CUANDO** el hook lo evalúa
- **ENTONCES** deniega el despacho
- **Y** la causa nombra el grupo `pace` y que su marcador aparece más de una vez

#### Scenario: Un grupo requerido sin marcador se deniega nombrándolo

- **DADO** un proyecto `monorepo-embedded` y una llamada con solo el marcador heredado `pr-strategy`
- **CUANDO** el hook la evalúa
- **ENTONCES** deniega el despacho
- **Y** la causa nombra el grupo `pace` como el requerido que falta

#### Scenario: La interacción de la preferencia de revisión no se confunde con un grupo

- **DADO** una llamada de la interacción de la preferencia de revisión independiente (REQ-28.14)
- **CUANDO** el hook busca marcadores de grupo
- **ENTONCES** no encuentra ninguno y la ignora
(Previously: se aceptaban el prefijo `Axiom SDD preflight` y el heredado `Gentle AI SDD preflight`.)
```

### Capacidad: sdd-research

```openspec
## MODIFIED Requirements

### Requirement: Open Capability Admission for Optional Research (formerly: Closed Capability Admission)

The selected-research lane is optional, not mandatory, following the absorption of upstream's replacement of closed admission (commit `ba3ed690`). The lane MUST continue to accept only the classes `documentation` and `open-web`; admission MUST continue to verify declaration and exact grant. Bash, generic MCP, and unnamed legacy tools MUST continue to be denied. The absence of a selected-research invocation for a given change MUST NOT by itself block the proposal or any other SDD artifact.

(Previously: the lane MUST accept only `gentle-ai.sdd-research-capability/v1` and classes `documentation` and `open-web` under a closed, mandatory admission model, with no clause about the absence of research.)

#### Scenario: Request admitted when research is invoked

- GIVEN the orchestrator decides to invoke selected research for a change
- AND a known, selected class has its exact grant declared
- WHEN admission runs
- THEN it records the grant and permits the research

#### Scenario: Unknown or denied capability still blocks that invocation

- GIVEN a request has an unknown class/version or unverifiable tool
- WHEN admission runs
- THEN it records denial, emits no claim, and blocks that invocation
- AND that block does not prevent the change from continuing through a planning lane that does not depend on that research

#### Scenario: Absence of selected research does not block the proposal

- GIVEN a change for which the orchestrator did not invoke selected research
- WHEN that change's proposal is drafted
- THEN the absence of that research does not by itself block the proposal
- AND any block declared in `sdd-propose` comes from a reason other than the absence of selected research
(Previously: the lane named the schema `gentle-ai.sdd-research-capability/v1`, which no Axiom code or asset uses any more.)
```

## Progreso

T1 hecha (commit «fix(update): autoupdate y registro con el nombre axiom»). T2 hecha (commit «refactor(env): retirar las variables GENTLE_AI_*»). T3 hecha (commit «refactor(state): rutas de estado axiom con migración desde gentle-ai»). T4 hecha (commit «refactor(sync): marcadores, plugins y MCP con el prefijo axiom»). T5 hecha (commit «refactor(sddstatus): esquemas axiom con lectura de los gentle-ai persistidos»). T6 hecha (commit «ci(release): nombres axiom en la procedencia, la firma y la CI»). T7 hecha (commit «refactor: nombres internos, plantillas y pruebas sin gentle-ai»). T8 hecha (commit «docs: documentación y specs vivas sin los nombres gentle-ai»). Progreso: T1–T8 hechas.

## Siguiente paso

Abrir la PR.

## Acciones manuales

- Renombrar en GitHub los secretos `DISCORD_GENTLE_AI_*` a `DISCORD_AXIOM_*` (`RELEASES`, `BETA`, `ISSUES`), solo si se activa `discord-notifications.yml`.
