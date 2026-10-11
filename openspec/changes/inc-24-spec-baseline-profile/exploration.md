# Exploración: inc-24-spec-baseline-profile

> Fase `explore` · rama `feat/inc-24-spec-baseline-profile` · 2026-10-03 · solo lectura (no se ha modificado ningún fichero del proyecto).
> Acceso limitado a `C:\repos\axiom`. Los datos del proyecto de referencia (512 encabezados `### RF-NNN` con 496 IDs distintos, documentos 00–07, `context/TECHNICAL_CONTEXT.md`) se toman como dados por el usuario; no se han podido contrastar.
> No se ha hecho investigación externa: las incertidumbres restantes son internas al repositorio o al repositorio del cliente (spike recomendado en §8).

## Resumen ejecutivo

1. **El hueco no es el loader, es la raíz.** `workspace.LoadConfig` ya resuelve `.axiom-workspace` y lo usan skillregistry, livingdoc, dashboard, semantic y autoskill. Falta un primitivo que, desde cualquier directorio de partida (carpeta maestra, repo de specs o repo de código), devuelva carpeta maestra + raíz del repo de specs + planning home, y que lo consuman `sddstatus.resolveWorkspaceRoot` y `cli.resolveGovernanceChangeRoot`. Hoy ambos asumen `<cwd>/openspec`.
2. **El ciclo de vida del cambio es change-local.** Estado, kickoff, gates, handoff y `dependencies.archive` nunca leen la baseline (`status_test.go:1169-1210`). Externalizar la baseline solo afecta a `sdd-archive`, al compose, a las skills de lectura (`explore`, `propose`, `spec`), a init/detect y a los consumidores de catálogo (livingdoc, knowledge).
3. **Hay tres convenciones de layout incompatibles dentro del propio repo** (§2.4): el motor SDD usa `<raíz>/openspec/...`; livingdoc, dashboard y skillregistry usan `<specs_repository>/{specs,changes,INDEX.md,skills}`; el perfil knowledge (inc-23) escribe `specs_repository: "openspec"` relativo al propio `axiom.yaml`. Es la primera decisión a cerrar (Q1).
4. **Camino más barato y compatible (inclinación del explorador, pendiente de Q1 y Q2):** reubicar la raíz. `workspaceRoot` pasa a ser la raíz del repo de specs y `openspec/` sigue siendo literal debajo. Bastan dos puntos de entrada en lugar de ~70 apariciones; skills y 17 comandos casi intactos; coherente con los grants multi-repo, el ledger en git y `RefuseArchivedRoot`.
5. **Baseline externa = generalizar el compose.** Hoy solo reconoce `### Requirement:` con identidad por texto exacto (`openspec_archive_compose.go:33,166,310`). El proyecto de referencia exige patrón de encabezado, identidad por ID, plantilla de render y detección de duplicados (496 IDs en 512 encabezados). Los documentos 03–07 no son requisito: §4.3 recoge cuatro modos de integración, a decidir.
6. **La configuración estructurada debe vivir en `axiom.yaml`.** `openspec/config.yaml` es una convención de prompts "sin parser ni validador Go" (`docs/openspec-config.md:5-19`) y Go solo escanea una línea (`artifact_store`). `axiom.yaml` es tipado, valida y resuelve el puntero.
7. **El contexto técnico no existe como concepto:** solo `governance.context` (sin lector Go) y un `context:` de ≤10 líneas en `openspec/config.yaml`.
8. **Tamaño y entrega.** Estimación sin medir: 2.800–4.300 líneas cambiadas (código, tests, assets, goldens), es decir, 7–11 unidades de 400 líneas, frente a la estrategia `single-pr` de la sesión. Recomendación: tres rebanadas encadenadas A, B, C, y valorar que C sea un cambio propio (§8).
9. **Correcciones a la evidencia aportada (§2.1):** livingdoc sí resuelve el puntero (por fallback de `LoadConfig`); los "72 ficheros de assets" incluyen 3 tests Go. Los sitios ciegos al puntero son otros (`main.go` y `sdd_kickoff.go`).

## 1. Contexto y decisiones tomadas

Decisiones del usuario (no se reabren):

| # | Decisión |
|---|---|
| D1 | Los cambios son de Axiom (ciclo nativo `openspec/changes/<change>/` sin cambios); la spec viva y el contexto técnico son del proyecto. Archive integra el cambio en la baseline del proyecto, no en `openspec/specs/`. El trabajo en vuelo del proyecto adoptado termina con su flujo antiguo. |
| D2 | Topología objetivo: multirepo canónico. Carpeta maestra no versionada con `.axiom-workspace` que apunta a `<specs_repository>/axiom.yaml`; repo de specs siempre en master; repos de código hermanos. |
| D3 | Tabla de adopción: direccionable y fiable, la baseline propia vía perfil; fiable pero en prosa o tablas, normalizar (añadir IDs) o integración dirigida por agente con verificación; sin spec, OpenSpec desde cero; spec mala o desactualizada, contexto de solo lectura y OpenSpec desde cero. |

Alcance previsto (a confirmar): (1) resolver la raíz de specs desde `axiom.yaml`/`.axiom-workspace` en verbos SDD, comandos, knowledge, livingdoc e init; (2) perfil de baseline por proyecto, con OpenSpec como defecto sin configuración; (3) skills de lectura y archive usan la baseline y nunca crean una segunda; (4) contexto técnico de primera clase.

Fuera de alcance: cold-start de knowledge (`####` frente a `###`, dominios falsos del sweep, sobrescritura del coldstart), `git init` del review en raíces no git, nombre `gentle-ai` en continuaciones del review, `--cwd` con espacios sin comillas, comando `axiom adopt`, integración con Azure DevOps.

Qué no cambia: kickoff, gates, handoff, cierre de roles, estado de artefactos, `dependencies.archive`, skills `tasks`, `apply` y `verify` (con B' sus rutas relativas siguen siendo válidas), claves obligatorias del status v2, almacén Engram, RDD.

## 2. Estado actual con evidencia

### 2.1 Evidencia aportada: verificación

| Evidencia aportada | Resultado | Nota |
|---|---|---|
| Ciclo change-local; `status_test.go:1169-1192` | Confirmado | El test además asevera que `openspec/specs/` no existe antes de archivar (`:1191-1193`): status nunca lee ni crea la baseline. |
| `<cwd>/openspec` fijo en `status.go:297-316,508`, `sdd_governance.go:53-69`, `kickoff/archived.go:13-34`, `main.go`, `dashboard/service.go:406,929` | Confirmado y ampliado | Faltaban `status.go:327,353,802,1265,1456,1509`, `edit_authority_consent.go:152`, `tui/model.go`, `livingdoc/synthesizer.go`, defaults de `knowledge`, `hub`. Ver §2.2. |
| `PlanningHome{Mode,Path}`, solo `repo-local` (`status.go:89`) | Confirmado | Skills y comandos OpenCode aún citan `workspace-planning`, que Go nunca emite (vestigio de upstream). |
| Compose: regex `:33`, identidad `:166,:310`, rechazo sin requisitos `:296`, CLI | Confirmado | |
| `sdd-archive`: compose `:113-143`, rama sin baseline `:160-202`, move `:204-280` | Confirmado | Los tests extraen y EJECUTAN esos bloques shell (`assets_test.go:2601-2660`). |
| `sdd-spec` `:64,:73-75,:101`; `sdd-propose` `:68,:102,:113,:186`; `sdd-explore` `:53` | Confirmado | Pins en `assets_test.go:2927-2956` (conteo exacto 2 y frases prohibidas). |
| livingdoc `service.go:33-43` "lee solo `<root>/axiom.yaml`, ignora `.axiom-workspace`" | **Matizado** | `LoadConfig(<root>/axiom.yaml)` cae al puntero si el fichero no existe (`loader.go:106-110`; pin `loader_test.go:253-260`). Lo roto es otra cosa: layout derivado distinto del motor SDD, `Synthesizer.ColdStart` ignora `specs_repository`, y se asume raíz = carpeta maestra. |
| Regex de ID de livingdoc `indexer.go:15-19` | Confirmado | `(REQ-…)` opcional al final del título; ID sintético `REQ-<dominio>-<n>` si falta. |
| Loader con puntero `loader.go:84-128`; tipos `types.go:18-51`; `governance.context` sin lector | Confirmado | `Context` está en `types.go:43`; lo escribe `hub/init.go:260-269`; `workspace.root` (`types.go:22`) tampoco tiene lector. |
| `hub/init.go:98-143`, `detector.go:173-211` | Confirmado | Además `.openspec` detectado pero se scaffoldea `openspec/` (incoherencia previa). |
| skillregistry `registry.go:104-113` | Confirmado | Tramo real `:102-124`. |
| knowledge `query.go:233-336` | Confirmado | Es uno de cuatro parsers de `### Requirement:` (§2.5). |
| Comandos con `git rev-parse --show-toplevel \|\| pwd` | Confirmado | 17 ficheros (7 Claude, 10 OpenCode); pins `assets_test.go:2124-2203`. |
| Go solo lee `artifact_store` (`status.go:801-821`) y el detector | Confirmado | `docs/openspec-config.md:15-19`: "no hay parser ni validador Go". |
| Inferencia de proyecto Engram `status.go:852-865` | Confirmado | Con maestra sin `.git` cae al nombre de la carpeta. |
| 38 ficheros Go / 134 apariciones; 72 ficheros / 290 en assets | Confirmado con matiz | Los 72 incluyen 3 tests Go (47 apariciones): 69 ficheros activos con 243. Además 26 goldens con 141. |

### 2.2 Cómo se resuelve hoy la raíz (punto a)

Dos puntos de entrada cubren el motor SDD: `sddstatus.resolveWorkspaceRoot` (`status.go:1166-1196`; usado por `Resolve`, `continue` y, vía `ValidateCurrentChangeInstance`, por `sdd-attempt grant`) y `cli.resolveGovernanceChangeRoot` (`sdd_governance.go:31-70`; usado por `kickoff seal|show` y `gate record|show`). El resto son hojas que construyen `<raíz>/openspec/...` a mano. Solo `--cwd`/cwd participa: ningún llamador de producción rellena `ResolveOptions.WorkspaceRoot`.

| Grupo | Sitios (fichero:línea) | Qué asume |
|---|---|---|
| `sddstatus` (motor, compartido con upstream) | `status.go:297-315,326-343,349-353,376-400,508-509` (changes y archive); `:801-821` (`openspec/config.yaml`); `:1263-1266` (`PlanningHome.Path`); `:1456,1509-1510` (literales de mensaje); `:852-865` (proyecto Engram); `edit_authority_consent.go:144-158` (exige `planning == <ws>/openspec`); `governance.go:489` (texto); `runtime_ledger.go:208-234` (ledger en el git common dir) | `<raíz>/openspec` literal; raíz = cwd |
| `cli` verbos SDD | `sdd_governance.go:31-70`; `sdd_kickoff.go:114-121`; `sdd_gate_evidence.go:42-47` (grafo git de la raíz); `sdd_attempt.go:65`; `sdd_archive_compose.go:29-44` (rutas por flag y literales) | idem |
| `kickoff` | `archived.go:14,23-35` (prefijo `openspec/changes/archive`) | idem |
| `cmd/axiom` | `main.go:653,936,1004-1034,1044-1075` (handoff, `role barrier`, `resolveChangeDir`, `resolveHandoffFile`); comandos archive/knowledge `:1585-1684,2071,2147,2327` | idem |
| `livingdoc` | `service.go:33-43`; `synthesizer.go:38-50` | `Service` honra `specs_repository`; `Synthesizer` no |
| `knowledge` | `planner.go:48`, `query.go:26`, `sweep.go:42` (`SpecsRoot` por defecto `<raíz>/openspec`); uniones `specs/` en `sweep.go:76`, `query.go:77,237`, `planner.go:219`, `index.go:22-23`, `finalize.go:33` | La opción `SpecsRoot` existe y ya separa `WorkspaceRoot` (código) de la raíz de specs, pero ningún llamador de `cmd/axiom` la rellena |
| `hub` | `init.go:98-143,200-218`; `detector.go:173-211` | `openspec`/`.openspec` |
| `dashboard`, `tui` | `dashboard/service.go:168,181,406,412,418,929,935` frente a `:873-878,1561-1566,1636-1641` (sí usan `specs_repository`); `tui/model.go:5931,5946,6008,6020,6044-6045` | mezcla de ambas convenciones |
| otros | `multirole/barrier.go:144-145` (texto); `reviewtransaction/risk.go:1193` (segmento `openspec` marca un `.md` como "operativo") | cadenas |

Quién usa ya el loader y el puntero:
- **Resuelven el puntero** (por el fallback de `LoadConfig` cuando `<raíz>/axiom.yaml` no existe): `workspace validate` (`main.go:541-542`), `livingdoc.Service`, dashboard, `semantic`, `autoskill`, `skillregistry` (más `guard.go:66`).
- **Ciegos al puntero** por un `os.Stat(<raíz>/axiom.yaml)` previo: `main.go:714-716,749-751,803-805,878-880` y `cli/sdd_kickoff.go:115-119`. Con carpeta maestra sin `axiom.yaml`, `wsConfig` queda `nil` sin avisar.
- **No lo usan**: `sddstatus`, `kickoff`, `cli/sdd_governance.go`, resolvers de `main.go`, `knowledge`, `hub`, `Synthesizer`.
- Ningún comando escribe `.axiom-workspace` ni inicializa el repo de specs. El manual (`docs/manual-de-inicio.md:138-142`) indica `axiom init --topology multirepo` en la maestra, que crea `axiom.yaml` allí: receta distinta del patrón canónico de `odd/tasks/canonical-workspace-and-skills-sync.md`.

### 2.3 Multirepo ya anticipado por el modelo de autoridad de edición

- `AllowedEditRoots = [workspaceRoot] + grants` (`status.go:607`, `baseStatus`). El detector de rutas de `tasks.md` ya contempla repos hermanos: "task prose routinely names files that do not exist yet, like `../service-a/...`" (`edit_authority.go:236-239`); un repo distinto se nombra por su raíz git y exige consentimiento por cambio.
- Los grants se guardan en el git common dir y exigen repo git (`runtime_ledger.go:174-176,208-234`: `RuntimeRepositoryRequiredError`). Status degrada sin error (`status.go:1860-1881`), pero `sdd-attempt grant` falla con una raíz no git.
- Consecuencia: con `workspaceRoot` = carpeta maestra no versionada no hay grants ni ledger; con `workspaceRoot` = repo de specs (git) el modelo existente funciona tal cual.
- `gate record --evidence-kind pr_merged` verifica contra el grafo git de `workspaceRoot` (`sdd_gate_evidence.go:42-47`); en multirepo sería el repo de specs, no el de código.

### 2.4 Convenciones de layout en conflicto (punto b)

| Sitio | Convención | Evidencia |
|---|---|---|
| Motor SDD, kickoff, gates, handoff, TUI, dashboard (CRUD) | `<raíz>/openspec/{changes,specs,config.yaml}`; raíz = cwd | §2.2 |
| livingdoc `Service` | `<raíz>/<specs_repository>/{specs,INDEX.md}` si `specs_repository` no es `""` ni `.`; si no, `<raíz>/openspec/...` | `livingdoc/service.go:33-43` |
| livingdoc `Synthesizer` (coldstart) | `<raíz>/openspec/...` ignorando `specs_repository` | `synthesizer.go:38-50` |
| dashboard (deuda acumulativa) | `<raíz>/<specs_repository>/changes/cumulative-<rol>` | `dashboard/service.go:873-878` |
| skillregistry | `<raíz>/<specs_repository>/skills` y `<raíz>/<repo-de-rol>/skills` | `registry.go:102-124` |
| `axiom init` (full y embedded) | `specs_repository: "openspec"`; crea `<ruta>/openspec/{specs,changes}` | `hub/init.go:98-123` |
| `axiom init --profile=knowledge` (inc-23) | `multirepo`, `specs_repository: "openspec"` relativo al propio `axiom.yaml`; crea `<ruta>/openspec/{specs,changes,INDEX.md}` | `hub/init.go:66-76,113-141`; inc-23 `spec.md:53-57` |
| `workspace validate` | `<base>/<specs_repository>` debe existir; base = carpeta del puntero o de `axiom.yaml` | `validator.go:62-102` |
| Patrón canónico (ODD y tests) | `<maestra>/.axiom-workspace` → `<maestra>/repo-specs/axiom.yaml`; `repo-specs/skills/`; rutas relativas a la maestra | `odd/tasks/canonical-workspace-and-skills-sync.md:9,36`; `registry_test.go:808-870`; `sync_auto_skill_index_test.go:12-90` |
| Specs vivas | embedded: `specs_repository` es `.` o subcarpeta (`openspec`, `specs`); multirepo: repo canónico hermano (`especificacion/`) | `workspace-topology` REQ-2.2 y REQ-2.4; `living-documentation` REQ-1.1 |

`specs_repository` significa a la vez "directorio plano con specs y changes" (livingdoc, dashboard, embedded), "repo hermano relativo a la maestra" (canónico) y "subcarpeta del propio base" (knowledge). Solo el motor SDD ignora `specs_repository`.

### 2.5 Baseline: parsers, emisores y creadores

- **Parsers de `### Requirement:` en Go (4):** compose (`sddstatus/openspec_archive_compose.go:33`), indexador (`livingdoc/indexer.go:17,116-133`), búsqueda (`knowledge/query.go:306-311`) y recuento (`knowledge/index.go:62-63`, `#{2,4}`). Más la prosa de las skills.
- **Emisores con nivel distinto:** `knowledge/recorder.go:153` escribe `#### Requirement:`; `sweep.go:254` y `query.go:496` escriben `###` (cold-start, fuera de alcance).
- **Dos escritores de `INDEX.md` con formato distinto:** `livingdoc.Indexer.SyncIndexForProject` (`indexer.go:235-263`; antes `SyncIndex`, renombrado en el commit A2-c2 de este cambio) y `knowledge.SyncIndex` (`index.go:21-60`), ambos con `os.WriteFile` incondicional.
- **Creadores de `openspec/specs/` (riesgo de segunda baseline):** `hub/init.go:113-124` (todos los perfiles); skill `sdd-init` (`_shared/openspec-convention.md:34`); rama "Main Spec Does NOT Exist" de `sdd-archive` (`SKILL.md:160-202`); `knowledge sweep` (`sweep.go:75-95`) y grabador (`recorder.go:59-69`); `axiom archive coldstart` (`synthesizer.go:38-94`, sin comprobar existencia pese a REQ-2.2 de living-documentation).
- **Las skills escriben con rutas fijas** `openspec/changes/{change-name}/...` (`sdd-propose:60,164`, `sdd-spec:59-87`, `sdd-design:70,164`, `sdd-tasks:67,225`, `sdd-archive:111-215`); solo `sdd-apply` es "store-blind" y lee por `artifactPaths` (`SKILL.md:46,62`). La convención compartida (`openspec-convention.md:29-59`) repite las rutas.

### 2.6 Configuración y contexto técnico hoy (insumo del punto c)

| Fuente | Quién la lee | Claves |
|---|---|---|
| `axiom.yaml` | Go: `workspace.LoadConfig` (tipado, validado, resuelve puntero) | `workspace.{name,topology,specs_repository,root}`, `roles.*`, `governance.{language,shared_memory,semantic_analysis}`; `governance.context` lo escribe `hub/init.go:260-269` y nadie lo lee |
| `.axiom-workspace` | `workspace.ParsePointer` (`loader.go:66-78`) | `config` o `specs` |
| `openspec/config.yaml` | Go: escaneo de línea de `artifact_store` (`status.go:801-821`) y `hub/detector.go:188-211` (solo en init). Skills: `context`, `rules.<fase>`, `strict_tdd`, `testing` (prosa) | `docs/openspec-config.md:5-19,131-142`: sin parser ni validador, formas inconsistentes |
| Engram `sdd-init/{project}` | skills y orquestador | stack detectado, capacidades de test |

Contexto técnico: no hay concepto. `contextFiles` del status es copia de `artifactPaths` (`status.go:656,746`); la tabla "Context Protocol" del orquestador (`claude/sdd-orchestrator-workflow.md:182-191`) solo lista artefactos del cambio.

### 2.7 Absorción upstream (punto f)

- No existe política que prohíba divergir las skills SDD. La regla de rutas protegidas (`docs/upstream-absorption-ledger.md:22`; REQ-20.4) rige solo para tandas de absorción.
- `internal/workspace`, `livingdoc`, `hub`, `multirole`, `handoff`, `semantic` son la capa multi-repo V8 del fork: "Upstream no tiene equivalente" (inc-20 `design.md:519-537`). Coste de divergencia ≈ 0 ahí.
- Superficie compartida con upstream que inc-24 tocaría: `sddstatus`, `cli/sdd_*`, `assets/skills/sdd-*` y `_shared`, comandos, orquestadores. Ya divergente: 29 marcadores `ODD-/INC-/REQ-2x` en 8 ficheros de skills y 19 en 4 ficheros de `sddstatus`; `sdd-archive` ya incluye ODD-4.1/4.2 y el compose ya tiene `ComposeOptions`.
- Techo congelado en `v3.4.0` (`82a6de96`); integrar versiones posteriores es "trabajo futuro, a diseñar una vez Axiom esté terminado" (`ledger:8-12`). Referencia de coste: en F4 (259 ficheros) 170 se adoptaron tal cual, 68 se fusionaron a tres vías y 32 conflictos se resolvieron a mano (`ledger:159`).
- Mitigación: lógica nueva en ficheros o paquetes solo-fork; en ficheros compartidos, costuras de pocas líneas; detalle de skills en `references/`. `docs/skill-style-guide.md` fija un presupuesto de cuerpo (recomendado 700 tokens, máximo 1000) que `sdd-archive` (395 L) ya excede; ningún test lo impone.

## 3. Puntos de contacto con tamaño aproximado

Tamaños de fichero medidos (L = líneas). Columna "Toca" = orden de magnitud estimado, sin medir, incluyendo tests.

### 3.1 Go

| Paquete / fichero | Tamaño | Qué cambia | Rebanada | Toca |
|---|---|---|---|---|
| `workspace/loader.go`, `types.go`, `validator.go` | 187 / 62 / 130 L | `Locate()` (maestra, repo de specs, planning home, origen); campos `baseline` y `technical_context` con validación | A, B | 250–400 |
| `sddstatus/status.go` | 1881 L | `resolveWorkspaceRoot` re-enraíza (`:1166-1196`); literales (`:1456,1509-1510`); `declaredArtifactStore` | A | 60–120 |
| `sddstatus/status_v2.go` | 312 L | campo opcional `baseline` (`omitempty`) | B | 40–80 |
| `sddstatus/edit_authority_consent.go`, `edit_authority.go`, `governance.go` | 287 / 424 / 545 L | sin cambio si se re-enraíza; si no, generalizar `:152` | A | 0–60 |
| `sddstatus/openspec_archive_compose.go` | 410 L (+322 L test) | perfil de encabezado, ID, render, duplicados | C | 150–250 |
| `cli/sdd_governance.go`, `sdd_kickoff.go`, `sdd_gate_evidence.go` | 87 / 205 / 47 L | un `Locate()` antes del `Join`; `LoadConfig` sin `os.Stat` previo | A | 40–90 |
| `cli/sdd_archive_compose.go` | 101 L (+101 L test) | flags de documento y perfil; mensajes | C | 60–120 |
| `kickoff/archived.go` | 36 L | sin cambio si se re-enraíza | A | 0–20 |
| `cmd/axiom/main.go` | ~2.400 L | resolvers de cambio y handoff, 4 cargas con `os.Stat` previo, archive y knowledge | A | 60–120 |
| `livingdoc/{service,synthesizer,indexer}.go` | 101 / 94 / 250 L (+200 L test) | raíz e `INDEX.md` según la regla; guarda de segunda baseline | A, B | 80–200 |
| `knowledge/{planner,query,sweep,index,finalize,recorder}.go` | 321 / 518 / 280 / 86 / 52 / 233 L | rellenar `SpecsRoot` desde el resolver; guarda con baseline externa | A, B | 60–150 |
| `hub/init.go`, `detector.go` | 273 L / — | no scaffoldear `openspec/specs` con baseline externa; incoherencia `.openspec`; escribir perfil | B | 100–200 |
| `dashboard/service.go`, `tui/model.go`, `tui/screens/*` | ~1.700 / >6.000 L | rutas de cambios, archivo y acumulativo; textos | A (o diferido) | 100–250 |
| `multirole/barrier.go`, `reviewtransaction/risk.go` | — | referencias de texto; clasificación por segmento `openspec` | A | 10–30 |

### 3.2 Skills y assets

| Superficie | Tamaño | Qué cambia | Notas |
|---|---|---|---|
| `skills/sdd-archive/SKILL.md` | 395 L, 29 apariciones | Step 2 (compose y rama sin baseline), reporte con integración por ID; Step 3 solo si el home se reubica | Pinned verbatim y ejecutado: `assets_test.go:2476-2660` |
| `sdd-spec` (268 L, 13), `sdd-propose` (191 L, 15), `sdd-explore` (153 L, 3), `sdd-onboard` (231 L, 4) | | rutas de lectura de la baseline y de promoción | Pins `assets_test.go:2927-2956` |
| `sdd-design` (206 L, 7), `sdd-tasks` (264 L, 7) | | rutas de escritura `openspec/changes/{change-name}/...` | Solo si no se re-enraíza |
| `sdd-init` (76 L) y `references/init-details.md` | | no crear `openspec/specs/` con baseline externa | 8 goldens `sdd-*-skill-sdd-init.golden` (10 apariciones cada uno) |
| `_shared/openspec-convention.md` | 130 L, 26 | árbol, tabla de rutas, sección de baseline | |
| `_shared/sdd-phase-common.md` | 143 L | regla global: rutas `openspec/` relativas a `actionContext.workspaceRoot` | |
| `_shared/sdd-status-contract.md` | 154 L | campo `baseline`; texto de `planningHome` | Pin `assets_test.go:1940-1984` |
| `_shared/persistence-contract.md` | 131 L, 9 | rutas | |
| Comandos | 17 ficheros | ninguno si el walk-up vive en Go | Pins `assets_test.go:2124-2203` |
| `agents/sdd-*.md` (claude, cursor, kiro, kimi) | ~20 ficheros | "main specs" por "baseline" | Goldens kiro y claude |
| Orquestadores | 12 ficheros, 3–5 apariciones | probablemente ninguno (solo nombres de store) | |
| Goldens | 26 ficheros, 141 apariciones | regenerar con `-update` | |

### 3.3 Tests, contratos y ratchets (punto g)

| Qué | Qué fija | Impacto |
|---|---|---|
| `status_v2_clean_break_test.go:76-106` | conjunto exacto de claves de `StatusV2Projection` | No se rompe con campos `omitempty` aditivos (precedente: `Consent`, `Archived`, `Governance`, `GateQuestion`, `PhaseInstructions`, `status_v2.go:34-46`); sí con claves nuevas obligatorias. No hay JSON Schema publicado del status SDD en `contracts/`. |
| `assets_test.go:1940-1984` | texto del contrato ("exact frozen external", `path: <absolute path to openspec>`) | Cambiar la redacción exige editar el pin. |
| `assets_test.go:2124-2203` | pista `git rev-parse --show-toplevel` en 17 comandos | Intacto si el walk-up vive en Go. |
| `assets_test.go:2476-2660` | texto verbatim y ejecución de los bloques shell de `sdd-archive` | Parametrizar rutas en esos bloques exige actualizar pins y harness. |
| `assets_test.go:2927-2956`, `sdd_phase_rules_tier_parity_test.go`, `store_branching_absence_test.go` | ubicación change-local; paridad de reglas por tier; ausencia de ramificación por store | Revisar al tocar skills. |
| `testdata/golden/*` | 26 ficheros, 141 apariciones | Regenerar. |
| `sddstatus/*_test.go` (`status_test.go` 1.210 L, `artifact_states`, `archived_projection`, `governance`, …) | cadenas `openspec/changes/...` | Red de regresión del layout por defecto. |
| `workspace/loader_test.go` | puntero `config`/`specs`, fallback, no encontrado | Extender para `Locate()`. |
| `openspec_archive_compose_test.go` (322 L, 10 tests), `cli/sdd_archive_compose_test.go` (101 L, 3) | compose nativo | Regresión; añadir casos de perfil. |
| `cli/refusal_resolution_ratchet_test.go` + `.refusal-ratchet-baseline.txt` | todo refusal nuevo o editado en `cli`, `reviewtransaction` y `sddstatus` nombra una continuación `gentle-ai ...` o lleva `// refusal:by-design`; la línea base solo puede encoger | Colocar refusals nuevos en `internal/workspace` (no cubierto) o redactarlos con continuación. |
| `scripts/deadcode-ratchet.sh` (CI) | funciones nuevas alcanzables desde `cmd/axiom` | Todo resolver nuevo debe estar cableado. |
| `bench/` (módulo aparte, 56 journeys, fuera de `go test ./...`) | rutas `openspec/` por defecto | Ejecutar aparte. |

Capacidades vivas previsiblemente afectadas (deltas): `workspace-topology` (REQ-2.1, 2.4), `living-documentation` (REQ-1.1, 1.2, 2.1, 2.2, 3.x; 20 menciones), `sdd-archive-lifecycle` (REQ-21.17), `multi-project-hub` (init crea `openspec/specs`), y en menor medida `dashboard-sdd-orchestration`, `local-web-dashboard`, `tui-ui-parity`. Nuevas: resolución de planning home, perfil de baseline, contexto técnico.

### 3.4 Compose: puntos de estrangulamiento y generalización mínima (punto d)

Puntos en `openspec_archive_compose.go`:

| Línea | Qué | Generalización |
|---|---|---|
| `:33` | `openSpecReqHeadingLine` | patrón configurable con grupos `id` y `title` |
| `:34` | límite de bloque `^## \S` | cualquier encabezado de nivel ≤ nivel del requisito; hoy una subsección `###` no-requisito se absorbe en el requisito previo y un MODIFIED la sobrescribe |
| `:129-173` | `parseSpecDocument` | usar el perfil de la baseline |
| `:204-216` | bloques del delta | el delta sigue nativo (`### Requirement: <id o nombre>`); resolver contra la baseline por ID |
| `:227-236`, `:265-276` | `renameRequirementHeading`, `annotateRequirementSuperseded` emiten `### Requirement: ` fijo | plantilla de render del perfil (`### {id}: {title}`) |
| `:294-297` | rechazo sin requisitos | mensaje según el patrón |
| `:308-315` | `indexOf` por nombre exacto, devuelve la primera coincidencia | identidad por ID y rechazo explícito de IDs duplicados (hoy un MODIFIED sobre un nombre repetido se aplica en silencio a la primera) |
| `:326-343` | RENAMED con flecha | con identidad por ID el título cambia con MODIFIED; renumerar es otra operación (Q7) |
| `:391-399` | ADDED: duplicado e inserción tras el último requisito | asignación de ID y punto de inserción según perfil |

Boceto (no vinculante): `ComposeOptions.Heading *HeadingProfile{Pattern, Level, Render, Identity}`; `nil` = comportamiento nativo byte a byte (los 10 tests actuales son la red). CLI: mantener `--canonical`/`--delta` y añadir `--cwd` y `--document <dominio>` para resolver ruta y perfil desde `axiom.yaml`, de modo que la skill no plantille rutas en shell. Pruebas nuevas: tabla por perfil (ID, duplicados, `###` no-requisito, niveles, sufijo `[SUPERSEDED / DEPRECADO]`, títulos con flecha) y fixture CRLF. Estimación: 500–700 líneas con tests.

## 4. Opciones de convención de layout (punto b)

### 4.1 Opciones

| Opción | Regla | Pros | Contras | Esfuerzo |
|---|---|---|---|---|
| **A. `specs_repository` es el planning home (plano)** | `<maestra>/<specs_repository>/{changes,specs,INDEX.md,config.yaml,skills}` | Coincide con livingdoc, dashboard, skillregistry, embedded y knowledge; embedded sin migración | Invasivo en un repo adoptado: `changes/`, `INDEX.md` y `config.yaml` caen en la raíz del repo del cliente, y `specs/` choca con su `specs/` (proyecto de referencia); `SyncIndex` sobrescribe `INDEX.md` sin mirar; hay que hilar `PlanningHome` por ~70 apariciones, reescribir rutas de skills y pins; generalizar `PrepareChangeInstanceConsent:152` y `RefuseArchivedRoot` | Alto |
| **B. El repo de specs contiene `openspec/`** | planning home = `<repo de specs>/openspec` | Huella acotada y predecible (`openspec/` + `axiom.yaml`); no toca `specs/` ni `context/` del proyecto; es el layout actual de embedded y de knowledge | Necesita regla de compatibilidad para `specs_repository: "openspec"` (alias de la raíz); alinear livingdoc y dashboard-acumulativo con la regla | Medio |
| **C. Clave explícita `planning_home`** | independiente de `specs_repository` | Sin heurística; admite A y B | Otra clave y precedencias; mismo coste de hilado que A | Alto |
| **B'. B con reubicación de raíz** | `workspaceRoot` pasa a ser la raíz del repo de specs; `openspec/...` queda literal | Dos puntos de entrada; skills, comandos y pins casi intactos; coherente con grants, ledger git, `RefuseArchivedRoot:14`, `PrepareChangeInstanceConsent:152` y `ArchivedProjection.Path` relativo | Ejecutores deben resolver `openspec/...` contra `actionContext.workspaceRoot` (regla en `sdd-phase-common.md`); código de repos hermanos exige grant por cambio; validar cwd de sub-agentes | Bajo–Medio |

Inclinación del explorador: **B'**, sujeta a Q1 y Q2. A y C siguen siendo viables si se prefiere configurar el home explícitamente, a cambio de hilar el planning home por todo el código y las skills.

### 4.2 B': regla, precondiciones y casos a verificar en diseño

- Regla (boceto): en `monorepo-embedded`, `<raíz>/openspec` sin cambios. En `monorepo-decoupled` y `multirepo`: si `base(specs_repository)` es `openspec`, el home es `<maestra>/<specs_repository>`; si no, `<maestra>/<specs_repository>/openspec`. `workspaceRoot = padre(home)`.
- Casos: embedded con `specs_repository` en `""`, `.`, `openspec`, `specs`; multirepo knowledge (`specs_repository: openspec` relativo al propio `axiom.yaml`); multirepo canónico (`repo-specs`); receta del manual (`axiom.yaml` en la maestra); `.openspec`.
- Punto de partida: maestra (puntero), repo de specs (`axiom.yaml` presente; hay que decidir cómo hallar la maestra: `workspace.root` sin lector hoy, o sufijo de `specs_repository`) y repo de código (walk-up hasta `.axiom-workspace`). Con walk-up en Go los 17 comandos no cambian (Q3).
- Contrato para ejecutores: una regla en `sdd-phase-common.md` ("las rutas `openspec/` se resuelven contra `actionContext.workspaceRoot`, no contra el cwd") y bloques shell con `cd`. Es una hipótesis a validar con un spike en Claude Code y OpenCode, porque el cwd de bash se reinicia entre llamadas.
- Efectos: `AllowedEditRoots = [repo de specs]`; los repos de código requieren grant por cambio (flujo existente); proyecto Engram inferido del remoto del repo de specs.
- Ambigüedad a bloquear: maestra con `openspec/` propio y puntero a la vez. Hoy el puntero gana en `ResolveConfigFile` (`loader.go:106-110`); con el re-enraizado, los cambios que existieran en `<maestra>/openspec` dejarían de verse. El mensaje debe nombrar una continuación (ratchet).

### 4.3 Integración de documentos no-requisito: opciones (punto e)

Documentos 03–07 (modelo de datos, flujos, interfaces, integraciones, seguridad) no tienen forma de requisito. Opciones, sin decidir:

| Modo | Qué hace | Verificación | Pros | Contras | Esfuerzo |
|---|---|---|---|---|---|
| E1. Solo registro | archive no edita esos documentos; emite un registro por ID (documento, ancla, acción pendiente) y la edición queda al flujo del proyecto | cada ID del cambio aparece con estado pendiente o integrado | Sin riesgo de corromper prosa | La baseline queda desactualizada hasta que alguien integra | Bajo |
| E2. Agente con verificación | el agente edita (la regla de copia mecánica protege artefactos, no la edición de prosa) y un verificador nativo comprueba: todo ID ADDED/MODIFIED/REMOVED figura en el registro, cada ancla citada existe tras la edición, el `git diff` solo toca documentos declarados, y hay marca de origen (comentario `<!-- axiom:change <c> -->` o fila en `REGISTRO_INCREMENTOS.md`) | determinista en presencia y alcance, no en corrección semántica | Baseline al día con traza auditable | Anclas en tablas frágiles | Medio–Alto |
| E3. Normalizar primero | añadir IDs o anclas a 03–07 (fuera de Axiom) y tratarlos como bloques direccionables con el mismo compose | el compose determinista | Un solo motor, byte-preserving | Invasivo en el repo del cliente; el compose pasa de requisito a bloque direccionable | Medio, más coste de adopción |
| E4. Parche | el delta para esos documentos son hunks unified-diff; archive ejecuta `git apply --check` y luego `git apply` | falla si el contexto no coincide | Mecánico y válido para cualquier formato | Spec phase escribiría parches sin ver el documento final; deriva de contexto | Medio |
| Capa común (opcional) | `integration.yaml` por cambio con `id`, `doc`, `ancla`, `acción`, hash antes y después, y una verificación que recalcula hashes y diff | tamper-evident, no semántico | Componible con E2 y E4 | Más superficie | Medio |

Registro por ID (boceto): `id: RF-123`, `action: MODIFIED`, `primary: {doc, composed: true}`, `secondary: [{doc, anchor, action: edited, after_sha256}]`. Lugares posibles de la traza: sección nueva del `archive-report`, `integration.yaml` en el cambio o `REGISTRO_INCREMENTOS.md` del proyecto (Q8).

## 5. Dónde vive la configuración (punto c)

### 5.1 Opciones

| Opción | Pros | Contras |
|---|---|---|
| Solo `axiom.yaml` | Tipado (`WorkspaceConfig`), validado en `ParseConfig` y `workspace validate`, resuelve puntero, versionado en el repo de specs; YAML no estricto, así que binarios antiguos ignoran bloques nuevos | `rules`, `strict_tdd` y `artifact_store` seguirían en `openspec/config.yaml` |
| Solo `openspec/config.yaml` | Coherente con lo que leen las skills | Sin validador y con formas inconsistentes; el lector Go es un escaneo de línea; vive dentro del home (hay que resolver el home antes de leerlo); un bloque con regex y plantillas sería frágil |
| Reparto: estructura en `axiom.yaml`, reglas de conducta en `openspec/config.yaml` | Cada cosa donde ya se lee | Dos ficheros; hay que documentar la frontera |

Inclinación del explorador: **reparto**. Estructura y rutas (baseline, contexto técnico) en `axiom.yaml`; `rules.<fase>`, `strict_tdd`, `artifact_store` en `openspec/config.yaml`. `governance.context` (texto libre, sin lector) se deja como está.

### 5.2 Transporte del localizador a las skills

| Opción | Pros | Contras |
|---|---|---|
| Clave aditiva opcional `baseline` (y `technicalContext`) en el status v2 | Precedente claro; las skills ya leen localizadores del status ("never re-resolve", `sdd-phase-common.md` sección B); no rompe el test de claves exactas | Contrato "congelado externo": consumidores externos (p. ej. gentle-pi) deben tolerar campos nuevos |
| Verbo aparte de solo lectura (`axiom sdd context --json`) | No toca el contrato de status | Orquestadores (12 plantillas y goldens) deben aprender a invocarlo |
| Las skills parsean YAML | Sin cambios Go | Contradice la doctrina de localizadores; frágil |

### 5.3 Boceto ilustrativo (nombres y forma sin decidir)

```yaml
# axiom.yaml
baseline:                       # ausente = OpenSpec por defecto (<home>/specs)
  root: specs                   # relativo a la raíz del repo de specs
  documents:
    - domain: requisitos-funcionales
      path: 01_Requisitos_Funcionales.md
      requirement:
        heading: '^### (?P<id>RF-\d+): (?P<title>.+)$'
        identity: id            # id o text
        render: '### {id}: {title}'
  read_only:                    # contexto del proyecto que archive nunca escribe
    - 03_Modelo_Datos.md
technical_context:
  entry: context/TECHNICAL_CONTEXT.md
  paths: [context/]
  read_by: [explore, design, apply, verify]
```

Nota de nombre: "profile" ya significa otra cosa en `axiom init --profile full|knowledge|spec-only` (`main.go:1707`); se propone `baseline` (Q4).

## 6. Riesgos y casos límite

| ID | Sev. | Riesgo | Evidencia | Mitigación |
|---|---|---|---|---|
| R1 | Alta | Regresión del layout por defecto (Axiom y todo proyecto embedded) | 134 menciones en 38 ficheros; `axiom.yaml` propio usa `specs_repository: "openspec"` | La resolución por defecto debe producir exactamente `<raíz>/openspec`; tests existentes y `bench/` como red; test de equivalencia |
| R2 | Alta | Contrato de status "congelado externo" | `assets_test.go:1940-1984`; `status.go:213-216` (gentle-pi#538 comparte contrato) | Solo campos `omitempty` aditivos; `planningHome.path` sigue siendo la ruta absoluta del directorio `openspec` bajo B' |
| R3 | Alta | Autoridad de edición y ledger ligados a git y a `<ws>/openspec` | `runtime_ledger.go:174-176,208-234`; `edit_authority_consent.go:152` | `workspaceRoot` = repo de specs (B'); si no, generalizar `:152` y asumir ledger no git |
| R4 | Alta | IDs duplicados: `indexOf` devuelve la primera coincidencia | `openspec_archive_compose.go:308-315`; proyecto de referencia: 512 encabezados y 496 IDs | Rechazo explícito de operaciones ambiguas; política en Q7 |
| R5 | Media | Límite de bloque al cambiar el patrón de encabezado | `:34,129-173` | Frontera por nivel de encabezado; tests con `###` no-requisito |
| R6 | Media | Fidelidad CRLF/UTF-8 (hipótesis por lectura de código, no ejecutada) | `renameRequirementHeading` y `annotateRequirementSuperseded` sustituyen la línea completa (`:227-236,265-276`) | Fixture CRLF y BOM antes de implementar |
| R7 | Media | Segunda baseline accidental | §2.5; `hub` detecta `.openspec` pero crea `openspec/`; dos escritores de `INDEX.md` | Resolver compartido y guardas en todos los creadores (Q11) |
| R8 | Media | Coexistencia con el flujo antiguo: ediciones concurrentes de la misma baseline, repo de specs "siempre en master" y asignación de IDs nuevos | Decisión D1; `docs/manual-de-inicio.md:180-184` | Regla de escritura única y política de IDs (Q7, Q9) |
| R9 | Media | Superficie compartida con upstream ya divergente | §2.7 | Lógica en paquetes solo-fork; costuras mínimas; detalle en `references/` |
| R10 | Media | Ratchets de CI: refusals nuevos en `cli`/`sddstatus`, funciones inalcanzables, pins y goldens | §3.3 | Refusals nuevos en `internal/workspace` o con continuación; cablear todo desde `cmd/axiom`; regenerar goldens |
| R11 | Media | Gate `integration` en multirepo: `pr_merged` verifica el repo de specs, no el de código | `sdd_gate_evidence.go:42-47` | Documentar `deployment`/`attestation` o añadir `--repo` (Q13) |
| R12 | Media | Engram `hybrid`/`engram` con maestra no versionada: el proyecto se infiere del nombre de carpeta y puede no coincidir con el del MCP | `status.go:852-865` | `ENGRAM_PROJECT` o raíz = repo de specs; comprobar en diseño |
| R13 | Media | Entrega: previsión muy por encima de 400 líneas con `single-pr` | `sdd-phase-common.md` sección E | Decidir cadena o `size:exception` antes de `sdd-tasks` (Q12) |
| R14 | Baja | RDD clasifica `.md` bajo un segmento `openspec` como operativo; fuera de `openspec` serían pasivos | `reviewtransaction/risk.go:1193` | Solo afecta si RDD está activo (opt-in) |
| R15 | Baja | Deriva documental: el manual describe `axiom sdd archive-compose <cambio> --supersede <cap>` (sintaxis inexistente), una receta multirepo distinta del patrón canónico y lectura nativa de "cualquier Markdown/OpenSpec" | `docs/manual-de-inicio.md:138-142,318,331` | Actualizar docs en la rebanada correspondiente |
| R16 | Baja | Directorios sin trackear `.agents/`, `.gemini/`, `.kiro/` contienen copias generadas de skills | `git status` inicial | Excluidos de los recuentos; no son fuente |

## 7. Preguntas abiertas para la propuesta

Producto y gobernanza (no las decide el explorador):

| ID | Pregunta | Por qué importa | Opciones o inclinación |
|---|---|---|---|
| Q1 | ¿Cuál es la huella de Axiom dentro del repo de specs: raíz plana (A), `openspec/` anidado (B) o clave explícita (C)? | Determina coste, adopción no invasiva y qué convención se migra | Inclinación B/B' por huella acotada y compatibilidad (§4) |
| Q2 | ¿Qué es el `workspaceRoot` en multirepo: raíz del repo de specs o carpeta maestra? ¿Se pre-autorizan los repos de rol declarados en `axiom.yaml`? | Autoridad de edición, ledger git, grants por cambio | Repo de specs: usa el modelo de grants existente; la maestra no versionada no admite grants |
| Q3 | ¿Cómo se descubre la maestra desde un directorio anidado: walk-up solo de `.axiom-workspace` o también de `axiom.yaml`? ¿Qué significa `workspace.root`? ¿Qué pasa con maestra que tiene `openspec/` propio y puntero? | Evita tocar 17 comandos; riesgo de re-enraizar proyectos anidados | Walk-up solo del puntero; bloquear la ambigüedad |
| Q4 | ¿Se confirma el nombre `baseline` (y `technical_context`) en lugar de "profile"? | `axiom init --profile` ya existe con otro sentido | `baseline:` y `technical_context:` |
| Q5 | Forma de la declaración: ¿documentos por dominio o raíz más glob? ¿Cómo se mapea dominio a documento y qué se hace con un dominio nuevo sin documento (rechazar, preguntar, plantilla)? | Evita crear una segunda baseline | Por dominio, con rechazo por defecto |
| Q6 | Formato del delta para baseline externa: delta nativo con ID en el nombre o formato del proyecto; plantilla de render del encabezado; ¿se inyectan los escenarios BDD en el cuerpo del requisito del proyecto? | Define qué escribe `ADDED` y `MODIFIED` | Delta nativo, render por plantilla |
| Q7 | IDs: ¿qué se hace con duplicados existentes (rechazar, exigir normalización previa, selector por ocurrencia)? ¿Quién asigna IDs nuevos y cómo se evitan colisiones con el flujo antiguo? ¿Qué es RENAMED con identidad por ID? | R4; el proyecto de referencia tiene 16 encabezados repetidos | Rechazar ambigüedad y exigir normalización |
| Q8 | Documentos no-requisito: ¿modo E1, E2, E3 o E4, y dónde vive la traza? | Verificabilidad del archive | Ver §4.3 |
| Q9 | Coexistencia con el flujo antiguo en vuelo: ¿quién escribe la baseline y cuándo? ¿Archive actualiza `REGISTRO_INCREMENTOS.md` y `REGISTRO_BUGS.md`? | R8 | Sin inclinación |
| Q10 | Contexto técnico: ¿qué fases lo leen, con qué presupuesto (entrada más subcarpetas) y es de solo lectura? ¿Existe un modo `read_only` para baseline "mala o desactualizada"? | Cubre dos casos de la tabla de adopción | Entrada más índice bajo demanda, solo lectura |
| Q11 | ¿inc-24 añade guardas que hagan rechazar a `knowledge sweep`, `archive coldstart` y al grabador cuando hay baseline externa, o todo se difiere al cambio de cold-start? | Evita segunda baseline | Guardas mínimas en inc-24 |
| Q12 | Entrega: ¿un cambio con rebanadas encadenadas o varios cambios? ¿Se acepta `size:exception` o cadena frente a `single-pr`? | R13 | Ver §8 |
| Q13 | Gate `integration` en multirepo: ¿`--repo` o repos de rol para `pr_merged`, o se documenta `deployment`/`attestation`? | R11 | Documentar ahora |

## 8. Recomendación de alcance

### 8.1 Rebanadas

| Rebanada | Contenido | Valor (tabla de adopción) | Estimación (sin medir) | Depende de |
|---|---|---|---|---|
| **A. Resolver de raíz** | `workspace.Locate()`; `sddstatus` y `cli` governance; handoff y role en `main.go`; livingdoc, knowledge y dashboard consumen la misma raíz; regla de ejecución en `sdd-phase-common.md`; docs | El multirepo canónico funciona para todos los verbos SDD: cubre "sin spec, OpenSpec desde cero" | 800–1.200 | Q1, Q2, Q3 |
| **B. Declaración, guardas y contexto técnico** | bloques `baseline` y `technical_context` en `axiom.yaml` con validación; proyección aditiva en status; guardas anti-segunda-baseline (init/detect, `sdd-init`, sweep/coldstart/recorder); skills leen contexto técnico | Añade "spec mala, contexto de solo lectura y OpenSpec desde cero" y deja declarada la baseline | 800–1.200 | A; Q4, Q5, Q10, Q11 |
| **C. Compose, archive y skills de lectura** | perfil de encabezado, ID, render, duplicados; flags; `sdd-archive`, `sdd-spec`, `sdd-propose`, `sdd-explore`, convención; integración no-requisito y su verificación | "Direccionable y fiable, baseline propia" y "fiable en prosa, agente con verificación" | 1.200–1.900 | A, B; Q5–Q9; spike del proyecto de referencia |

Total estimado: 2.800–4.300 líneas. Cada rebanada supera por sí sola 400 líneas, así que cada una necesita 2–3 PRs encadenados.

### 8.2 Si trocear

- **Opción 1:** un solo cambio (`inc-24`) con tres rebanadas encadenadas. Mantiene un proposal, spec y design únicos, pero C arrastra Q5–Q9 y retrasa el cierre.
- **Opción 2 (inclinación del explorador):** `inc-24` = A + B, y un cambio nuevo para C (p. ej. `inc-25-baseline-compose-archive`) tras responder Q5–Q9 y ejecutar el spike. A y B son de baja ambigüedad, desbloquean el multirepo y dos de los cuatro casos de adopción; C depende de decisiones de producto y de datos del cliente.
- En ambos casos `single-pr` no es viable: decidir cadena o `size:exception` antes de `sdd-tasks`.

### 8.3 Cuestiones de diseño derivadas (para `sdd-design`)

- API y ubicación del resolver: `internal/workspace` (solo-fork, fuera del ratchet de refusals); orden de precedencia y casos de §4.2.
- Exposición: clave aditiva `baseline` frente a verbo aparte (§5.2); alcance de `planningHome.path`.
- Compose: tipo de perfil, frontera por nivel, duplicados, render, retrocompatibilidad byte a byte (§3.4); parser compartido entre compose, livingdoc y knowledge (hoy 4 parsers) frente a perfil solo en compose.
- Contrato de ejecutores sobre cwd y rutas relativas; validación empírica en Claude Code y OpenCode.
- Engram: proyecto inferido con raíz no git.

### 8.4 Spike recomendado antes de C

Análisis de solo lectura de la baseline del proyecto de referencia (por el usuario o con acceso explícito a esa ruta): por documento, número de encabezados por nivel; cobertura de `^### RF-\d+:`; IDs repetidos y si son repeticiones idénticas o conflictos; `###` no-requisito dentro de bloques de requisito; finales de línea (CRLF o LF) y BOM; longitud de bloques; y, para 03–07, si hay anclas o encabezados estables por entidad o flujo.

## 9. Listo para propuesta

Sí para el alcance A + B, con Q1, Q2, Q4 y Q12 resueltas antes de `sdd-propose` (condicionan alcance, capacidades afectadas y entrega) y Q3, Q5, Q10, Q11 en propuesta o diseño. C debe entrar como rebanada condicionada o como cambio posterior, a la espera de Q5–Q9 y del spike.

## Key Learnings

1. `workspace.LoadConfig(<root>/axiom.yaml)` ya cae al puntero `.axiom-workspace` cuando ese fichero no existe, por lo que los sitios ciegos al puntero son los `os.Stat` previos de `main.go` y `sdd_kickoff.go`.
2. Reubicar `workspaceRoot` al repo de specs mantiene literal `openspec/` y reutiliza grants multi-repo, ledger git, `RefuseArchivedRoot` y `PrepareChangeInstanceConsent` sin generalizarlos.
3. Un campo opcional `omitempty` en `StatusV2Projection` no rompe `assertExactJSONKeys`, y `Consent`, `Archived` y `Governance` son el precedente aditivo.
4. El compose resuelve identidad por texto exacto con `indexOf` y devuelve la primera coincidencia, así que una baseline con IDs duplicados exige rechazo explícito de operaciones ambiguas.
5. Los ratchets de refusals, deadcode y los pins de `sdd-archive` (que se extraen y ejecutan) condicionan dónde colocar el código nuevo y cuánto cuesta tocar las skills.
agentId: a7bc9ab0d28d7a310 (use SendMessage with to: 'a7bc9ab0d28d7a310', summary: '<5-10 word recap>' to continue this agent)
<usage>subagent_tokens: 620216
tool_uses: 193
duration_ms: 1638245</usage>
