# ODD: Mensajes visibles con la marca del upstream

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/marca-mensajes-visibles.md`.
> Espejo de recuperación en Engram: topic `odd/marca-mensajes-visibles/tasks`, proyecto `axiom`.
> Rama: `fix/marca-mensajes-visibles`, worktree `C:\repos\axiom-wt\fix-marca-mensajes-visibles`, creado el 2026-10-09 desde `origin/main` `770914de` y rebasado sobre `93cd7c65` antes del push.

---

## 0. Cómo retomar este ODD

1. `mem_context`; `mem_search "odd/marca-mensajes-visibles"` en el proyecto `axiom`; `mem_get_observation`; y **este fichero**.
2. `git -C C:\repos\axiom fetch --prune origin`, `git worktree list`, `gh pr list --state open`.
3. Concilia y continúa con la primera tarea sin marcar de la sección 6.

## 1. Objetivo

`TRADEMARKS.md` («Forks and modified distributions») exige que los forks usen un nombre principal distinto. Cambiar a «Axiom»/`axiom` solo el texto **visible para el usuario** que presenta Gentle AI como el producto: errores, ayuda/usage, salida de CLI, TUI y dashboard.

## 2. Alcance y restricciones

- **Sí:** cadenas visibles que nombran Gentle AI como el producto, o que piden ejecutar `gentle-ai <verbo>` cuando el binario real es `axiom`.
- **No:** identificadores técnicos (marcadores, IDs de esquema/contrato `gentle-ai.*/vN`, rutas, nombres de fichero, variables `GENTLE_AI_*`, orígenes de backup, cadenas de detección de instalaciones antiguas, valores JSON persistidos o con hash). La propia política los excluye y cambiarlos rompe compatibilidad.
- **No:** comentarios Go, assets agent-facing (`internal/assets`), herramientas internas (`bench/`, `scripts/`) salvo que un test de producto las acople.
- **Se conserva:** la atribución veraz «basado en Gentle AI» en README y LICENSE, y las referencias veraces a binarios o backups históricos del upstream.
- Sin push ni PR sin preguntar. Un commit por unidad de trabajo, Conventional Commits en castellano, sin atribución de IA.

## 3. Convenciones

- TDD desactivado (`openspec/config.yaml`, `strict_tdd: false`); runner `go test` con `-run` por nombre exacto. Nunca `go test ./internal/cli/` completo en local.
- RDD: leer `axiom review mode status` antes del primer commit; riesgo con `axiom review assess --cwd . --base-ref origin/main --committed-only --json`.
- Todas las llamadas a `Agent` llevan `model: sonnet`.

## 4. Inventario (verificado el 2026-10-09)

Exploración delegada de solo lectura (Explore, sonnet) más comprobación directa del orquestador de `review_mode.go:573,589`, `review_defect_report.go:23,211`, `uninstall.go:150`, `opencodeplugin/plugin.go:93`, `help.go:54`, `backup/manifest.go:33` y `bug_report.yml:67`.

**Corrección a la petición original:** `internal/app/app.go` ya dice `usage: axiom skill-registry` (`app.go:389`); no hay nada que cambiar ahí.

### 4.1 Visibles

| ID | fichero:línea | Texto actual | Tests que lo fijan |
|---|---|---|---|
| V1 | `internal/agents/hermes/adapter.go:175` | `before Gentle AI can configure it` | ninguno |
| V2 | `internal/agents/openclaw/adapter.go:171` | ídem | ninguno |
| V3 | `internal/components/communitytool/tool.go:197,549,552` | `Gentle AI is written against`, `rerun Gentle AI` | ninguno |
| V4 | `internal/components/communitytool/pi_codegraph.go:319,675` | `Gentle AI will not overwrite it`, `no Gentle-AI ownership record` | ninguno |
| V5 | `internal/installcmd/resolver.go:139` | `before installing Gentle AI Pi packages` | ninguno (el test casa solo un prefijo) |
| V6 | `internal/update/upgrade/strategy.go:530` | `Trust only this Gentle AI %s` | ninguno |
| V7 | `internal/components/engram/download.go:647` | prefijo `gentle-ai: engram stop:` | ninguno |
| V8 | `internal/components/opencodeplugin/plugin.go:93` | logo `✦ Gentle AI ✦` | ninguno |
| V9 | `internal/tui/screens/community_tools.go:16` | `tools Gentle AI can install` | ninguno |
| V10 | `internal/cli/uninstall.go:150,152` | `remove gentle-ai managed configuration` | ninguno |
| V11 | `internal/cli/review_assess.go:70` | `a defect in gentle-ai itself` | ninguno |
| V12 | `internal/cli/review_capabilities.go:195,398,402,407` | `gentle-ai package version`, `gentle-ai executable` | ninguno |
| V13 | `internal/cli/review_incident.go:127` | `gentle-ai never provisions…` | ninguno |
| V14 | `internal/cli/review_mode.go:498` | `applied for this gentle-ai only` (la mención al binario anterior se conserva) | `review_mode_cross_version_test.go:66` |
| V15 | `internal/cli/review_mode.go:573,599,602,603,760` | consentimiento: `Gentle AI can review…`, `Gentle AI reviewed…`, `could not read…`, `did not recognize…` | `review_consent_relay_test.go:161,236,528`, `review_negotiated_stderr_silence_test.go:246,298`, `review_mode_test.go:860,870`, `bench/metrics_test.go:12,236`, `bench/testdata/observations.json:89`, `bench/classify_test.go:150` |
| V16 | `internal/cli/review_mode.go:589` | `reviewConsentOffPathCommand = "gentle-ai review mode disable"` (el esquema consent/v3 admite ambas herramientas) | tests que usan la constante; `review_narration_test.go:96` |
| V17 | `internal/cli/review_consent_contract.go:101` | titular ES `Gentle AI puede revisar…` | `review_consent_locale_test.go:18` |
| V18 | `internal/cli/review_narration.go:142,148` | `this version of Gentle AI`, `Gentle AI does not recognize` | ninguno |
| V19 | `internal/app/help.go:54` | `(or 'gentle-ai review mode disable')` → se propone quitar el paréntesis | `help_test.go:111` |
| V20 | `internal/reviewtransaction/compact_store.go:2069,2070` | `a newer gentle-ai…`, `upgrade the reading gentle-ai` | ninguno |
| V21 | `internal/reviewtransaction/rdd_mode.go:76,284` | `every gentle-ai on this machine`, `for this gentle-ai` | ninguno |
| V22 | `internal/reviewtransaction/store_reset.go:188` | `no gentle-ai code writes or reads this path` | ninguno |
| V23 | `internal/cli/review_defect_report.go:205,211` + `.github/ISSUE_TEMPLATE/bug_report.yml:67` | `Gentle AI reached…`, `## Gentle AI Version` | ninguno |
| V24 | `internal/cli/review_defect_report.go:23` | URL de issues del **upstream** `Gentleman-Programming/gentle-ai`: los defectos de Axiom se mandan al tracker ajeno. Propuesta: `IGutierrezZ/axiom` | ninguno |

### 4.2 Fuera de alcance, por decisión de diseño (no es redacción)

Dialecto heredado `gentle-ai review status … --contract gentle-ai.review-integration/vN` en comandos de refresco y textos de error (`review_capabilities.go:61-62`, `review_next_transition.go:1310`, `review_operation_contract.go:285`, ~13 textos de error, ~25 tests y fixtures de contrato); `Package.Name` emitido en JSON; `facadeReviewPolicy` y prompt de lentes (con hash); prefijo `Gentle AI SDD preflight` (protocolo hook + assets); frontmatter `Gentle AI Persona` (goldens + detección en `cleaners.go`); línea SDD inyectada en CLAUDE.md (goldens); README del bundle y títulos de esquema (con hash); User-Agent HTTP y `clientInfo.name` MCP; `# Generated by gentle-ai` en la config de GGA. Se conservan por veraces: `backup/manifest.go:33` («Gentle AI histórico») y `tui/screens/backups.go:31`.

## 5. Decisiones del usuario

| Decisión | Valor |
|---|---|
| Carril | ODD (2026-10-09) |
| Lista de candidatos (sección 4.1) | **Confirmada V1-V24 tal cual** (2026-10-09), incluido quitar el paréntesis de V19 y redirigir V24 a `IGutierrezZ/axiom` |
| Entrega | `single-pr`; push y PR solo tras preguntar |

## 6. Tareas

- [x] **T1 · Adaptadores, componentes y TUI** (V1-V9): `03bcb038`, 9 ficheros, +12/−12. Ruta: delegada (writer sonnet; disparador: >2 ficheros).
- [x] **T2 · CLI de revisión, consentimiento y ayuda** (V10-V22): `31708073`. Ruta: delegada (mismo writer); corrección del fixture inline (un fichero de test, mecánica). El comando de desactivación del sobre sigue el dialecto negociado (`reviewConsentOffPathCommandFor`), así que v1 sigue recibiendo `gentle-ai review mode disable` (su esquema lo fija como `const`). `bench`: el clasificador de avisos ya ignoraba el prefijo de marca; se añade `TestConsentNoticeCountingIsIndependentOfTheBrandPrefix`.
- [x] **T3 · Informe de defectos** (V23-V24): `f2f54415`, 2 ficheros, +5/−5. Incluye `bug_report.yml:2` («File a bug report for Axiom»). Ruta: delegada junto con T1-T2.
- [x] **T4 · Cierre**: verificación independiente y este documento. Sin push ni PR.

Total: 29 ficheros, +71/−50 → un solo PR (`single-pr`).

## 7. Progreso y evidencias

- 2026-10-09: worktree creado; inventario verificado; lista confirmada.
- RDD desactivado (`axiom review mode status`: off, decidido por la configuración global). `axiom review assess --base-ref origin/main --committed-only --untracked-scope=exclude` → **`high`**. Motivo: la heurística de rutas sensibles marca `reviewtransaction/compact_store.go` (auth) y `update/upgrade/strategy.go` (update); el cambio es solo de texto. Por eso hubo verificador independiente.
- Verificador independiente (sonnet, solo lectura): **FAIL**. El writer había editado el `headline` de tres fixtures de contrato (`contracts/review-integration/v1/fixtures/consent.fixture.json`, `v2/fixtures/consent.fixture.json`, `v2/fixtures/consent-v3.fixture.json`). Eso rompía `TestReviewProviderArtifactV1/V20/V21ContractsArePinned` (digests fijados) y contradecía `v1/FREEZE.md` («MUST stay byte-unchanged»). El resto (alcance, completitud, compatibilidad de esquemas, redacción, commits) pasó.
- Corrección: se restauran los tres fixtures desde `origin/main` y `TestConsentQuestionMatchesVersionedFixture` normaliza el titular actual al congelado (`frozenConsentFixtureHeadline`); el esquema define `headline` como `{"type":"string","minLength":1}`. Integrada en el commit de T2.
- Comprobaciones observadas tras la corrección: `go test ./internal/cli/ -run '^(TestConsentQuestionMatchesVersionedFixture|TestReviewProviderArtifactV1ContractsArePinned|TestReviewProviderArtifactV20ContractsArePinned|TestReviewProviderArtifactV21ContractsArePinned|TestReviewConsentEnvelopeSerializedBytesUnchanged|TestConsentValidateAcceptsHistoricalShapes)$'` ok; todos los tests de `review_consent_runtime_identity_test.go`, `review_consent_envelope_bytes_test.go` y `review_dialect_test.go` ok; `go test ./internal/sddstatus/ -run Consent` ok; comprobación de muestra del orquestador: `go test ./internal/app/ -run Help` ok, `hermes` y `openclaw` ok. Evidencia del writer: `gofmt -l` vacío, `go build ./...` y `go vet` ok, paquetes de T1 ok, 122 tests de `internal/cli` en 3 grupos ok, `reviewtransaction` ok (204 s), tests del informe de defectos ok.
- **Fallos de entorno conocidos:** `cd bench && go test ./...` da 19 fallos en Windows (helpers de TTY, sandbox, ejecutables falsos: «executable file not found in %PATH%», «Access is denied»), sin relación con las cadenas. No ejecutados: `go test ./...` completo, E2E y `scripts/test-review-contract-package.sh`.

## 8. Seguimientos detectados (fuera del alcance aprobado)

- `.github/ISSUE_TEMPLATE/bug_report.yml:18`, `feature_request.yml:2,18,25` y `config.yml:4` siguen nombrando Gentle AI o enlazando al tracker y a las discusiones del upstream. Tras V24 esto queda incoherente.
- `bug_report.yml` pide `gga version`.
- El error de `axiom review assess` cuando hay ficheros sin seguimiento pide ejecutar `gentle-ai review status …` y `gentle-ai review assess`, un binario que ya no existe. Pertenece al grupo del dialecto heredado (sección 4.2), pero es visible y erróneo.
- Grupo 4.2 completo (dialecto heredado, preflight SDD, Persona, textos con hash): necesita su propio ODD.

## 9. Siguiente paso

Que el usuario decida el push y el PR (plantilla del repositorio, `type:fix`).
