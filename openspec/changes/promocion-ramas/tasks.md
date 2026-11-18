# Tareas: promoción de ramas configurable por repositorio y por cambio (C-02-a)

Rol único: `fullstack` (kickoff sellado). Fuente de las unidades de trabajo y del orden de commits: `design.md` (sección «Unidades de trabajo y estimación»). Entrega en una sola PR con `size:exception`.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~2500 (≈1000 de código y documentación + ≈1500 de tests), sin contar el delta de spec |
| 400-line budget risk | High |
| Chained PRs recommended | No |
| Suggested split | PR única con `size:exception`, organizada en 5 commits (uno por unidad de trabajo) |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

La estrategia está decidida por el mantenedor: una sola PR con `size:exception`, sin PRs encadenadas. Los 5 commits permiten revisar por unidad y revertir por unidad (la 3 depende de la 2 y la 4 de la 3; la 1 y la 2 son independientes del motor).

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Claves `main_branch` y `promotion` en `git:` y `repositories[]`, herencia en `branch.Policy` y aviso de `workspace validate` | PR única, commit 1 | `go test ./internal/workspace/ ./internal/branch/ -run "GitPolicy\|Promotion\|Policy"` | `go test ./cmd/axiom/ -run "WorkspaceValidate"` (validate con `promotion` sin `main_branch` avisa sin invalidar) | `internal/workspace/gitpolicy*.go`, `cmd/axiom/main.go` y los campos de `internal/branch/policy.go`; sin consumidores aguas abajo |
| 2 | Dominio puro: tokens, `ResolvePromotion`, `MergeDestination`, `PromotionStage`, plan (`BaseBranch`), registro, rechazos y resultado | PR única, commit 2 | `go test ./internal/branch/ ./internal/branch/registry/` | `go test ./internal/branch/...` entero (sin `main_branch` ningún test existente cambia) | `internal/branch/promotion*.go` (nuevos) y los campos `omitempty` de `planner.go`, `types.go`, `registry.go`, `result.go`; revertible sin tocar el motor |
| 3 | `ensure` resuelve, pregunta (`decision_required`) y registra el modo; bandera `--promotion` | PR única, commit 3 | `go test ./internal/branch/engine/ -run "EnsurePromotion\|Ensure"` y `go test ./internal/cli/ -run "BranchPromotion\|BranchEnsure"` | Escenarios `ensure` de `ensure_promotion_test.go` con fixtures git reales (`master`/`develop`, multirepo con dos repos de código) | `engine/prefs.go`, `ensure*.go`, `cli/branch*.go`; los campos de la unidad 2 quedan inertes |
| 4 | `status` propone la siguiente PR, `done` confirma contra el destino, instantánea del kickoff | PR única, commit 4 | `go test ./internal/branch/engine/ -run "StatusPromotion\|DonePromotion\|Snapshot"` y `go test ./internal/kickoff/` | Escenarios con merge commit y squash reales en dos etapas (`done_promotion_test.go`, `status_promotion_test.go`) | `engine/status.go`, `done.go`, `merge_layers.go`, `snapshot.go`, `kickoff/gitsnapshot.go` |
| 5 | Scripts de dogfooding, ejemplo, manual y doctrina de skills | PR única, commit 5 | `go test . -run "TestScriptsAxiomWorktree\|TestDocs"` (ver 5.5 para el paquete exacto) | `scripts/axiom-worktree.ps1`/`.sh` con `stubRepository.IntegrationBranch` | `scripts/axiom-worktree.*`, `axiom.example.yaml`, `docs/manual-de-inicio.md`, las dos skills |

Restricciones de verificación (todas las unidades): nada de compilar binarios sueltos de axiom (Cortex XDR los borra); `go test` por paquete y `go test ./internal/cli/ -run <patrón>` troceado, nunca `./internal/cli/` entero. Sin TDD estricto configurado: cada tarea lleva sus tests junto al comportamiento y la comprobación se ejecuta al cerrar la tarea. Cada unidad cierra con un commit convencional sin atribución de IA.

Aviso de ámbito: esta lista no toca `internal/branch/engine/publication.go`, `handoff.go`, `verdict.go`, `pending.go`, `planner_archive.go`, `livespec` ni el dashboard (fuera de alcance, ver `design.md`).

## Fase 1: Esquema y política (Unidad 1)

- [x] 1.1 En `internal/workspace/gitpolicy.go`: añadir `MainBranch` y `Promotion` a `GitConfig` y `Promotion` a `RepositoryPolicy`; constantes `PromotionBranch`, `PromotionRelease`, `PromotionAsk`; ampliar `gitKeys` y `repositoryKeys` (`main_branch` en `repositories[]` queda como clave desconocida); `branchValidator(key)` pasa a `switch` con `validPromotion` (acepta solo `branch`, `release`, `ask`, distingue mayúsculas, el error enumera los tres); `resolveGit` copia ambos campos sin valor por defecto; ampliar los textos de error que enumeran claves (`:89`, `:114`, `:128`).
  Cubre: REQ-25.26, REQ-25.27
  Aceptación: `promotion: Branch` y `main_branch` dentro de `repositories[]` producen error; los errores acumulados nombran las cuatro claves; `resolveGit` sin `promotion` devuelve vacío.
- [x] 1.2 En `internal/workspace/gitpolicy.go`: función pura `PromotionWarnings(cfg *WorkspaceConfig) []string` con los dos casos de D1 (`git.promotion` sin `main_branch` o con `main_branch` igual a `integration_branch`; `repositories[i].promotion` cuando la integración efectiva de la entrada iguala a `main_branch` o `main_branch` no existe).
  Cubre: REQ-25.26, REQ-25.27
- [x] 1.3 En `cmd/axiom/main.go` (junto a `ValidateRepositoryPolicies`, `:606`): añadir los avisos de `PromotionWarnings` a `report.Warnings` sin tocar `report.Valid`.
  Cubre: REQ-25.26
- [x] 1.4 En `internal/branch/policy.go`: `Policy` y `RepoPolicy` ganan `MainBranch` y `Promotion`; `RepoPolicy` gana `PromotionSource` (`repository` | `git` | vacío); `ResolvePolicy` copia los campos y carga `RepositoryPolicy.Promotion` en `PerRepository`; `For(rel)` hereda campo a campo (vacío hereda; `MainBranch` siempre del workspace). Crear `internal/branch/promotion.go` solo con `type Promotion string` y sus tres constantes (el resto de D3 va en 2.1).
  Cubre: REQ-25.3, REQ-25.46
- [x] 1.5 En `internal/branch/planner.go`: `repoPolicy()` (`:445`) propaga `MainBranch` y `Promotion` del workspace para un repositorio fuera de la carpeta maestra.
  Cubre: REQ-25.3
- [x] 1.6 Tests en `internal/workspace/gitpolicy_test.go`: tablas de claves nuevas válidas e inválidas, `main_branch` en `repositories[]` rechazada, `promotion: Branch` rechazada, `resolveGit` sin defecto, `PromotionWarnings` (sin `main`, `main` igual a la integración, entrada que iguala la integración); añadir las claves nuevas a `wantErrorContains` en `:146`, `:176` y `:539`.
  Cubre: REQ-25.26, REQ-25.27
- [x] 1.7 Tests en `internal/branch/policy_test.go`: herencia `git` → `repositories[]` de `Promotion`, `MainBranch` solo global, repositorio fuera de la carpeta maestra. Test de `workspace validate` en `cmd/axiom/` (patrón `WorkspaceValidate` existente): `promotion` sin `main_branch` avisa y `Valid` sigue en `true`.
  Cubre: REQ-25.3, REQ-25.26, REQ-25.27
- [x] 1.8 Verificar y cerrar: `go test ./internal/workspace/ ./internal/branch/` y `go test ./cmd/axiom/ -run "WorkspaceValidate"`; commit `feat(workspace): claves main_branch y promotion en git y repositories`.
  Aceptación: tests en verde y sin cambios en los resultados de tests existentes salvo la lista de claves.

## Fase 2: Dominio puro de la promoción (Unidad 2)

- [x] 2.1 En `internal/branch/promotion.go`: `PromotionSource*`, `PromotionStage*`, mapa `promotionLabels` y `PromotionLabel(token)` (no se reutiliza `Label()`), `PromotionRequest`, `ParsePromotionRequests(values) ([]PromotionRequest, error)` (errores: valor vacío, `<repo>` vacío, modo distinto de `branch` y `release` con `ask` no válido como respuesta, dos valores sin cualificar, mismo `<repo>` repetido; D10).
  Cubre: REQ-25.46, REQ-25.47
- [x] 2.2 En `internal/branch/promotion.go`: `PromotionFacts`, `RecordedPromotion`, `SnapshotPromotion`, `ResolvedPromotion`, `PromotionResolution` y `ResolvePromotion` con la precedencia de D4: exento o de una etapa; registro (con `legacy` si hay rama `change` y no campo); instantánea (con `snapshot` y `legacy` por marcador `SealedWithMain`); `--promotion` (cualificado > no cualificado; `answer` si la política es `ask`, `flag` si no); política (`repository`/`git`); `ask` pendiente; `branch` por defecto con origen `default`; conflicto cuando el modo materializado difiere del pedido.
  Cubre: REQ-25.46, REQ-25.47
- [x] 2.3 En `internal/branch/planner.go`: `PlanInput.Promotions map[string]ResolvedPromotion` (por `GitRoot`); `RepoPlan.MainBranch`, `Promotion`, `PromotionSource`, `BaseBranch`; métodos `TwoStage()` y `Base()`; `planRepository` los rellena; el repositorio de specs (`:367`) y la rama de archivado (`planner_archive.go:89` no se edita) siguen sin modo; `PRTarget` de la rama del cambio sigue siendo la integración; `BaseRef` sigue prevaleciendo.
  Cubre: REQ-25.46, REQ-25.47, REQ-25.50
- [x] 2.4 En `internal/branch/promotion.go`: `MergeDestination(repo, planned) (destination, prMergedApplies)` con la tabla de D7 (role → seguimiento con capa 3; archive → integración sin capa 3; change en dos etapas con `branch` o `ask` → `main_branch` sin capa 3; resto → integración con capa 3) y `PromotionStage(repo, aheadOfMain, evidence)` con las reglas de D8 (0 o nil por delante → `integration`; fusionada por capa 1 o 2 con soporte → `main`; capa 2 sin soporte → `integration` inconcluso).
  Cubre: REQ-25.48, REQ-25.49
- [x] 2.5 En `internal/branch/types.go`: `RecordRepo.Promotion` y `PromotionSource` con `omitempty`. En `internal/branch/registry/registry.go` (`validate`, `:280`): vocabulario cerrado `promotion` ∈ {"", `branch`, `release`} y `promotionSource` ∈ {"", `answer`, `flag`, `repository`, `git`, `default`, `snapshot`, `legacy`}; valor desconocido → `config_invalid` («bórralo, es derivado»).
  Cubre: REQ-25.14, REQ-25.46
- [x] 2.6 En `internal/branch/promotion_refusal.go` (nuevo): `EnsureArgs` con `Invocation()` (sin banderas opcionales produce exactamente la cadena de `modeInvocation` actual; rutas con `pathquote.Quote`), `PromotionDecisionRequired` (dos `DecisionChoice` `branch`/`release` con `Effect` que nombra las ramas reales y `Invocation` con `--promotion <modo>` o `<repo>=<modo>` si `qualify`), `PromotionConflict` (`plan_conflict`), `UnknownPromotionRepository` (`invalid_name` con lista válida) e `InvalidPromotion` (`config_invalid`).
  Cubre: REQ-25.47
- [x] 2.7 En `internal/branch/ensure_refusal.go`: `DecisionRequired(args EnsureArgs)` (conserva excepciones, `--repo` y `--promotion` en las invocaciones) y `BaseNotFound` nombra la rama principal y `git.main_branch` cuando `Base() != IntegrationBranch`. En `internal/branch/done_refusal.go`: `MergeNotConfirmed(..., destination, targets)` usa `destination` en mensaje, orden `git log <destino>..<rama>` y `Details.target`. Actualizar los llamadores existentes (incluidos los de test) sin cambiar valores esperados.
  Cubre: REQ-25.47, REQ-25.49
- [x] 2.8 En `internal/branch/result.go`: campos aditivos `omitempty` en `RepositoryResult` (`mainBranch`, `promotion`, `promotionLabel`, `promotionSource`, `promotionSourceLabel`) y `BranchResult` (`promotionStage`, `promotionStageLabel`); `NewRepositoryResult` los rellena; el texto añade «Promoción: …» y «siguiente PR hacia …». Ninguna clave nueva se llama `name`, `kind`, `worktree` ni `prTarget`.
  Cubre: REQ-25.48
- [x] 2.9 Tests de la unidad, junto al comportamiento: `internal/branch/promotion_test.go` (una tabla por función: `ResolvePromotion` con cada fila de D4, `legacy` por registro y por instantánea con y sin marcador y con y sin rama, conflictos, `answer` frente a `flag`, exento y de una etapa; `ParsePromotionRequests` con los cinco errores de D10 y los duplicados; `MergeDestination`; `PromotionStage`), `planner_test.go` (`BaseBranch` según el modo, `IntegrationBranch` y `PRTarget` intactos, repositorio de specs sin modo, ramas de rol intactas, sin `Promotions` plan idéntico al actual), `ensure_decision_test.go` y `promotion_refusal_test.go` (`EnsureArgs.Invocation()` byte a byte, `DecisionRequired` conserva excepciones y `--promotion`, `PromotionDecisionRequired` con `qualify`), `registry/registry_test.go` (ida y vuelta, bytes idénticos sin los campos, vocabulario inválido).
  Cubre: REQ-25.14, REQ-25.46, REQ-25.47, REQ-25.48, REQ-25.49, REQ-25.50
- [x] 2.10 Verificar y cerrar: `go test ./internal/branch/ ./internal/branch/registry/`, después `go test ./internal/branch/...`; commit `feat(branch): dominio puro de la promoción de ramas`.
  Aceptación: sin `main_branch` ningún test existente cambia de resultado.

## Fase 3: `ensure` y bandera `--promotion` (Unidad 3)

Depende de la Fase 2.

- [ ] 3.1 En `internal/branch/engine/prefs.go` (junto a `resolveMode`): `promotionRequest`, `promotionFacts` y `(*Engine).planWithPromotion(ctx, facts, req) (ChangePlan, []string, *Refusal)`: plan provisional con `Promotions == nil`, `PromotionFacts` por repositorio (registro por `CommonDir`, instantánea por `Path`, `BranchExists` solo para `legacy` de instantánea, petición, `repoPolicy`), `ResolvePromotion`, validación de `--promotion <repo>=…` con `<repo>` desconocido → `invalid_name`, primer repositorio `ask` en modo interactivo → `decision_required`, primer conflicto → `plan_conflict`, fija `Promotions` y replanifica. Sin `--archive` no se resuelve.
  Cubre: REQ-25.46, REQ-25.47
- [ ] 3.2 En `internal/branch/engine/ensure.go`: `EnsureRequest.Promotions []branch.PromotionRequest`; orden `gather` → `resolveMode` → `planWithPromotion` → `checkPlan`, antes de cualquier `fetch` y de cualquier escritura; reaplicar el mapa ya resuelto a `facts.input.Promotions` tras `apply` (segundo `gather`, `:166`) sin volver a resolver; avisos de `--promotion` sin efecto (repositorio de una etapa, repositorio de specs, petición sin cualificar sin `main_branch`); revalidación defensiva (`InvalidPromotion`).
  Cubre: REQ-25.47
- [ ] 3.3 En `internal/branch/engine/status.go`: `baseCandidates` (`:170-182`) usa `repo.Base()` en lugar de `repo.IntegrationBranch`; `missingBaseWarning` (`:202`) nombra la rama principal y `git.main_branch` cuando `Base() != IntegrationBranch`. (La etapa de `status` va en la Fase 4.)
  Cubre: REQ-25.10, REQ-25.47
- [ ] 3.4 En `internal/branch/engine/ensure_registry.go`: `recordRepository` (`:51`) escribe `Promotion` y `PromotionSource` del plan definitivo (nunca `ask`) y conserva los previos cuando el plan no tiene modo (como `ArchivePrefix`); sin `main_branch` los bytes del registro son idénticos a los actuales.
  Cubre: REQ-25.14, REQ-25.46
- [ ] 3.5 En `internal/cli/branch.go` y `internal/cli/branch_flags.go`: bandera `--promotion` como `branchRepeatedFlag`; validación con `ParsePromotionRequests` (errores de uso, salida 2: vacío, `ask`, `x=`, `=branch`, sin cualificar duplicado, repositorio duplicado); `--promotion` junto a `--archive` es error de uso (igual que `--repo`); `EnsureRequest.Promotions`; ayuda `branchEnsureHelp` documenta `--promotion branch|release|<repo>=branch|release`, repetible.
  Cubre: REQ-25.47, REQ-25.45
- [ ] 3.6 Tests en `internal/branch/engine/ensure_promotion_test.go` (nuevo), con fixtures git reales y sin red (`observe`, `wantNoRemote`): con `main_branch: master` e `integration_branch: develop`, modo `branch` crea desde `origin/master`, `release` desde `develop`; `ask` → `decision_required` sin crear nada ni registro y al repetir con la invocación sale de la base correcta y el registro guarda `answer`; `--promotion` distinto del registrado → `plan_conflict`; sin `main_branch` registro idéntico byte a byte; multirepo con dos repos de código en `ask` → dos llamadas secuenciales con `Details.name` y la segunda invocación conserva la primera respuesta; repositorio de specs exento; `branch` sin `master` → `base_not_found` que nombra `git.main_branch`; con `--base` manda la excepción; `--promotion x=…` con repositorio desconocido → `invalid_name`.
  Cubre: REQ-25.46, REQ-25.47, REQ-25.14, REQ-25.10
- [ ] 3.7 Tests en `internal/cli/branch_promotion_test.go` (nuevo, con `runBranchCLI`): `--promotion` vacío, `ask`, `x=`, `=branch`, sin cualificar duplicado, repositorio duplicado y `--archive` junto a `--promotion` → salida 2 (casos de error de uso D10); la ayuda documenta la bandera.
  Cubre: REQ-25.47, REQ-25.45
- [ ] 3.8 Verificar y cerrar: `go test ./internal/branch/engine/ -run "EnsurePromotion|Ensure"`, `go test ./internal/cli/ -run "BranchPromotion|BranchEnsure"` y `go test ./internal/branch/...`; commit `feat(branch): ensure resuelve y registra el modo de promoción`.
  Aceptación: `ensure` sin `main_branch` no cambia ni un byte del registro ni de la salida.

## Fase 4: `status`, `done` e instantánea (Unidad 4)

Depende de la Fase 3.

- [ ] 4.1 Crear `internal/branch/engine/merge_layers.go` con `(*Engine).mergeLayers(ctx, dir, branchRef, targets) (existing []string, evidence branch.MergeEvidence, err error)`, extraída de `done.go:441-493` (capas 1 y 2); cada consumidor emite su aviso de capa 2 sin soporte.
  Cubre: REQ-25.16, REQ-25.49
- [ ] 4.2 En `internal/branch/engine/done.go`: `Done` usa `planWithPromotion` (sin interacción); `mergeEvidence` (`:441`) usa `MergeDestination` en sus candidatos (`origin/<destino>`, `<destino>`) y solo rellena `evidence.PRMerged` si `prMergedApplies`; con `ask` pendiente sin registro ni instantánea el destino es `main_branch`; el `Switch` de `in_place` y `DoneCheckedOutInMainCheckout` siguen yendo a la integración.
  Cubre: REQ-25.49, REQ-25.16, REQ-25.17
- [ ] 4.3 En `internal/branch/engine/status.go`: `Status` usa `planWithPromotion` (sin interacción); `describeRepository` calcula `promotionStage` para la rama `change` de un repositorio `TwoStage()` con `Promotion == branch`, con `aheadOfMain` por `AheadBehind` contra la referencia de la rama principal (`baseCandidates` sobre una copia del plan con `BaseRef` vacía) y `mergeLayers` contra la integración; con `stage == main`, `BranchResult.PRTarget` pasa a `MainBranch`; aviso si capa 2 sin soporte («no se puede comprobar sin git 2.38…»); `ask` pendiente → aviso sin etapa.
  Cubre: REQ-25.48, REQ-25.15
- [ ] 4.4 En `internal/kickoff/gitsnapshot.go`: `GitSnapshot.MainBranch` (`yaml:"main_branch,omitempty"`) y `GitSnapshotRepo.Promotion` (`yaml:"promotion,omitempty"`); `Validate` comprueba `MainBranch` con `gitref.ValidRefName` si no está vacía y `Promotion` ∈ {`branch`, `release`}. En `internal/branch/engine/snapshot.go`: `Snapshot` usa `planWithPromotion` sin interacción, copia `Promotion` solo si la resolución viene del registro (`Recorded`) y es `branch` o `release`, `entry.Base = repo.Base()` (o `BaseRef` con `--base`), `snapshot.MainBranch = policy.MainBranch`; la instantánea se escribe una sola vez.
  Cubre: REQ-25.36, REQ-25.46
- [ ] 4.5 Tests en `internal/branch/engine/status_promotion_test.go` (nuevo): `prTarget: develop` antes de la fusión y `master` con `promotionStage: main` después (merge commit y squash en `develop`); rama recién creada desde `master` con `master` ascendiente de `develop` → `integration`; sin capa 2 → `integration` con aviso; `ask` pendiente → aviso; sin red; `status` desde un worktree enlazado en modo `branch` (patrón `TestStatusSeesTheRegistryFromALinkedWorktree`).
  Cubre: REQ-25.48, REQ-25.15
- [ ] 4.6 Tests en `internal/branch/engine/done_promotion_test.go` (nuevo, con `recordIntegration`, `squashedBranch`, `requireMergeTree`): dos etapas con merge commit fusionada solo en `develop` → `merge_not_confirmed` con `master` como destino; fusionada en `master` → eliminada por `ancestry`; squash real en las dos etapas → `equivalent_tree`; `pr_merged` verificada en `gates.yaml` sin ascendencia en `master` → conservada; `--force` → `forced` con la punta; `release` → eliminada al fusionarse en `develop`; rama de archivado → integración; registro borrado con `ask` → conservada; `done` desde el worktree del cambio.
  Cubre: REQ-25.49, REQ-25.16, REQ-25.17, REQ-25.50
- [ ] 4.7 Tests de instantánea en `internal/branch/engine/snapshot_test.go` y `internal/kickoff/gitsnapshot_test.go`: monorepo SDD, `ensure` con `branch` y después el sellado → `promotion: branch`, `main_branch: master`, `base: master`; un `ensure` posterior no reescribe `kickoff.yaml`; con el registro borrado se resuelve desde la instantánea (`snapshot`); `Validate` de `main_branch` y `promotion`; lectura sin los campos.
  Cubre: REQ-25.36, REQ-25.46
- [ ] 4.8 Si algún golden cambia por los campos aditivos de `status --json` o del texto, regenerarlo con el mecanismo existente del paquete afectado y revisar el diff a mano (esperado: ninguno sin `main_branch`).
  Cubre: REQ-25.48
- [ ] 4.9 Verificar y cerrar: `go test ./internal/branch/engine/ -run "StatusPromotion|DonePromotion|Snapshot"`, `go test ./internal/kickoff/`, `go test ./internal/branch/...`; commit `feat(branch): status propone la siguiente PR y done confirma contra el destino`.
  Aceptación: `done` en modo `branch` nunca borra con la sola evidencia `pr_merged`; sin `main_branch` el comportamiento es el actual.

## Fase 5: Scripts, documentación y doctrina (Unidad 5)

Depende de la Fase 4 (los scripts leen `integrationBranch` de `status --json`).

- [ ] 5.1 En `scripts/axiom-worktree.ps1`: tomar `$integration` de `@($status.repositories)[0].integrationBranch` dentro del `try` que ya interpreta `$status` (`:225-231`), con `'main'` por defecto antes del `try`; el bloque de `done` (`:249-260`) compara `$currentBranch -eq $integration`, ejecuta `git -C $mainRoot fetch origin $integration` y `merge --ff-only "origin/$integration"` y adapta los mensajes. `new` y `pr` no cambian.
  Cubre: REQ-25.24
- [ ] 5.2 En `scripts/axiom-worktree.sh`: tras capturar `STATUS` (`:206`), `INTEGRATION="$(printf '%s\n' "$STATUS" | json_paths integrationBranch | head -n 1)"` con `${INTEGRATION:-main}`; las líneas `:220-225` usan `"$INTEGRATION"` entre comillas. `new` y `pr` no cambian.
  Cubre: REQ-25.24
- [ ] 5.3 En `axiom.example.yaml`: en `git:` (`:32-42`) dos líneas comentadas `# main_branch: master` y `# promotion: branch` más un párrafo con la semántica de una y dos etapas, valores y defecto; en `repositories:` (`:73-85`) un ejemplo `promotion: ask`. Solo se añade texto: no se reformula ninguna línea fijada por `docs_contract_test.go:167-200`.
  Cubre: REQ-25.26, REQ-25.27
- [ ] 5.4 En `docs/manual-de-inicio.md`: en `git:` (`:317-331`) dos viñetas nuevas (`main_branch`, `promotion` con los dos flujos y `ask`) y la nota de compatibilidad («un binario anterior rechaza `git.main_branch` y `git.promotion` (`config_invalid`, clave desconocida); actualizad todo el equipo antes de declararlas o retiradlas antes de volver atrás»), sin tocar la viñeta fijada «Los binarios anteriores ignoran el bloque…»; en `repositories:` (`:333-350`) una viñeta para `promotion`; en «Ramas y Worktrees» un párrafo sobre `--promotion`, la siguiente PR y `done` en dos etapas.
  Cubre: REQ-25.26, REQ-25.27, REQ-25.47, REQ-25.49
- [ ] 5.5 En `internal/assets/skills/sdd-apply/SKILL.md` (`:266`) completar con «… and, in promotion mode `branch`, once merged there `prTarget` names `git.main_branch` for the second pull request»; en `internal/assets/skills/sdd-archive/SKILL.md` (`:98`, paso 5) indicar que en modo `branch`, `done` conserva la rama del cambio (`merge_not_confirmed`) hasta su fusión en `git.main_branch` y elimina la de archivado. No tocar `_shared/sdd-status-contract.md`. Si algún golden o test de assets de skills cambia, regenerarlo y revisar el diff.
  Cubre: REQ-25.49, REQ-25.50
- [ ] 5.6 Tests: en `scripts_axiom_worktree_test.go` (localizar su paquete con `rg -l "TestScriptsAxiomWorktree"`; `stubRepository` gana `IntegrationBranch`): `done` con `integrationBranch: develop` y el checkout en `develop` → `fetch origin develop` y `ff-only origin/develop`; sin el campo → `main`; `pr` con `prTarget: master` → `--base master --head inc/demo`. En `docs_contract_test.go`: añadir casos (no modificar los existentes) que comprueban que siguen las líneas fijadas y que aparecen `main_branch`, `promotion` y el aviso de compatibilidad.
  Cubre: REQ-25.24, REQ-25.26
- [ ] 5.7 Verificar y cerrar: `go test` del paquete que contiene `TestScriptsAxiomWorktree*` y `docs_contract_test.go` (`-run "TestScriptsAxiomWorktree|Docs"`), `go test ./internal/assets/...`, y una pasada final `go test ./internal/workspace/ ./internal/branch/... ./internal/kickoff/` más los trozos `go test ./internal/cli/ -run "Branch"`; commit `docs(branch): scripts, ejemplo, manual y doctrina de la promoción`.
  Aceptación: tests en verde; ninguna línea fijada por tests de docs reformulada.

## Cobertura de requisitos

| Requisito | Tareas |
|-----------|--------|
| REQ-25.26 (`git:`) | 1.1, 1.2, 1.3, 1.6, 1.7, 5.3, 5.4, 5.6 |
| REQ-25.27 (`repositories:`) | 1.1, 1.2, 1.6, 1.7, 5.3, 5.4 |
| REQ-25.36 (instantánea) | 4.4, 4.7 |
| REQ-25.46 (modo de promoción) | 1.4, 2.1, 2.2, 2.3, 2.5, 2.9, 3.1, 3.4, 3.6, 4.4, 4.7 |
| REQ-25.47 (`ensure`, `ask`, `--promotion`) | 2.1, 2.2, 2.3, 2.6, 2.7, 2.9, 3.1, 3.2, 3.3, 3.5, 3.6, 3.7, 5.4 |
| REQ-25.48 (`status`) | 2.4, 2.8, 2.9, 4.3, 4.5, 4.8 |
| REQ-25.49 (`done`) | 2.4, 2.7, 2.9, 4.1, 4.2, 4.6, 5.4, 5.5 |
| REQ-25.50 (contratos que no cambian) | 2.3, 2.9, 4.6, 5.5 |
| REQ-25.3, REQ-25.10, REQ-25.14, REQ-25.15, REQ-25.16, REQ-25.17, REQ-25.24, REQ-25.45 | 1.4, 1.5, 1.7; 3.3, 3.6; 2.5, 3.4; 4.3; 4.1, 4.2; 4.2; 5.1, 5.2; 3.5 |

## Notas de ejecución

- Pregunta abierta del diseño (`main_branch` en la instantánea, REQ-25.36): la tarea 4.4 lo implementa; si la spec no lo recoge, se ajusta el delta en `openspec/changes/promocion-ramas/specs/sdd-lane-kickoff/spec.md` antes de verify, no el código.
- Comportamientos asumidos por el diseño que la spec debe fijar y que las tareas 3.6 y 4.6 prueban: `ask` pendiente sin registro en `done` → destino `main_branch`; `--promotion <repo>=…` sin repositorio en el plan → `invalid_name`.
- Sin `main_branch` efectivo no cambia ni un byte (plan, registro, instantánea, JSON de `status`): ninguna tarea debe alterar el resultado de un test existente salvo los que enumeran claves y las firmas de `DecisionRequired` y `MergeNotConfirmed` en sus llamadores.
