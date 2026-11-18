## Exploración: promoción de ramas configurable (C-02-a, `promocion-ramas`)

Todas las rutas son relativas a `C:\repos\axiom-wt\promocion-ramas\axiom`. Los números de línea corresponden al `main` en el que nace el worktree (591840be).

### 1. Estado actual (flujo ensure → status → done)

**Esquema de `axiom.yaml`** (`internal/workspace/gitpolicy.go`)
- `GitConfig{IntegrationBranch, BranchPrefix}` (:34-37) y `RepositoryPolicy{Path, IntegrationBranch, BranchPrefix}` (:47-51). Listas cerradas `gitKeys = {"integration_branch","branch_prefix"}` y `repositoryKeys = {"path","integration_branch","branch_prefix"}` (:55-56).
- `decodeGit` (:86-97) y `decodeRepositories` (:111-164) recorren el `yaml.Node` con `decodeStringKeys` (:207-230); el validador por clave sale de `branchValidator` (:169, `ValidPrefix` si es `branch_prefix`, si no `ValidRefName`) y `repositoryValidator` (:177). Los errores se acumulan y nombran clave, valor y regla. Los textos de error fijan literalmente las claves admitidas (:89, :114, :128) y los tests los comprueban (`gitpolicy_test.go:146,176,539` con `wantErrorContains(... "integration_branch", "branch_prefix")`).
- `resolveGit` (:62-73) completa `IntegrationBranch`/`BranchPrefix` por defecto; `DefaultIntegrationBranch = "main"` (:20). `workspace/loader.go:107` hace `cfg.Git = resolveGit(declarations.Git)`.
- Un valor de clave admite hoy solo "cadena que valida con una función". `promotion` (enumerado `branch|release|ask`) y `main_branch` (ref válido) encajan con `gitValue` + un validador nuevo; basta ampliar `branchValidator`/`repositoryValidator` y los dos `values[...]`.

**Política** (`internal/branch/policy.go`)
- `Policy{IntegrationBranch, BranchPrefix, PerRepository map[string]RepoPolicy}` (:17-24); `RepoPolicy{IntegrationBranch, BranchPrefix}` (:28). `ResolvePolicy` (:37-61) y `For` (:68-81) heredan campo a campo (vacío = hereda). Es el punto exacto donde añadir `MainBranch` y `Promotion` con la misma herencia (git → repositories[]).

**Planificador puro** (`internal/branch/planner.go`, `planner_snapshot.go`)
- `RepoPlan` (:135-149) lleva `IntegrationBranch` y `BaseRef` (solo la excepción `--base`). `BranchPlan.PRTarget` (:160).
- `planRepository` (:350-416): el repositorio de specs de multirepo sale sin ramas (:367-369, `RepoSpecsInPlace`); la rama del cambio lleva `PRTarget: effective.IntegrationBranch` (:392); las de rol, `From: changeBranch` y `PRTarget: changeBranch` (:395-412), solo con 2 o más roles (:395).
- La instantánea solo aporta nombres de rama (`planner_snapshot.go:14-35`); el planificador NO lee `GitSnapshotRepo.Base` hoy.
- `planner_archive.go:96-100,134` fija base e `PRTarget` de la rama de archivado en la integración (debe seguir así, ver riesgos).

**Motor** (`internal/branch/engine/`)
- `Ensure` (`ensure.go:79-206`): `gather` (reúne hechos y carga el registro) → `resolveMode` (`prefs.go:107-126`) → `branch.Plan` → `checkPlan` (resuelve la base por repositorio con `resolveBase(repo)`, `ensure.go:288`) → `apply` → `newRecord`/`registry.Save` → `describe`.
- `baseCandidates` (`status.go:170-182`): sin `--base`, `refs/remotes/origin/<IntegrationBranch>` y luego `refs/heads/<IntegrationBranch>`. Es el único cálculo de la base (lo usan `ensure`, `describeRepository` `status.go:258` y `PendingArchive` `pending.go:64`). Para modo `branch` la base debe ser `main_branch`: se cambia aquí, no en la semántica de `IntegrationBranch`.
- `describeRepository` (`status.go:236-308`) es donde `status` lee git por rama; `prTarget` sale de `NewRepositoryResult` (`result.go:281-289`) desde el plan estático. No hay hoy cálculo dinámico del destino.
- `done` (`done.go`): `doneBranches` (:211-239) reconstruye ramas del registro con `PRTarget: repo.IntegrationBranch` (:228); `mergeEvidence` (:441-499) tiene capas 1 ascendencia, 2 árbol equivalente, 3 `pr_merged`, contra `origin/<integración>` y `<integración>` (:443-446; el rol, contra su rama de seguimiento :448-449); `Switch` a la integración en `in_place` (:389) y `DeleteBranch` (:394). Los repositorios de specs se omiten porque no tienen ramas (`repositories()` :191-205 salta repos sin ramas; el "`done.go:189`" del backlog es esa función).
- `ConfirmMerge` (`merge.go:28-40`): puro; orden ascendencia > árbol equivalente (si `TreeLayerSupported`) > `pr_merged` (no para archivo) > `force`. **La tolerancia a squash ya existe** (capa 2 + `--force`): `done_layers_test.go:92 TestDoneConfirmsASquashMergeByEquivalentTreeEvenWhenMainAdvanced`. Lo que cambia es el destino contra el que se evalúa.
- `done_refusal.go:38` usa `planned.PRTarget` como "destino" en `MergeNotConfirmed`: en modo `branch` debe ser `main_branch`.
- Consumidores de `IntegrationBranch` que NO deben cambiar de semántica: `publication.go:75,81-87` (`PublicationTarget`, rechaza publicar desde la integración y fija specs in situ), `handoff.go:104-108` (`HandOff`), `ensure_repos.go:219` (`checkSpecsRepository`), `verdict.go:237-264` (nombres esperados de la guarda), `integration.go:19-25` (`PlanningIntegration`), `ensure_refusal.go`, `archive_refusal.go`, `kickoff snapshot`, `sdd_trace.go`, dashboard (`service_branch.go:164-186`, `types.go:28,36`). `PublicationTarget` y `HandOff` copian `planned.PRTarget` del plan estático (`publication.go:94`, `handoff.go:114,157`), así que si el "siguiente PR" se calcula dinámicamente en `status`, esos dos caminos seguirían dando el destino de la primera etapa salvo que compartan el mismo cálculo.

**`decision_required` y `--mode`** (`prefs.go:107-126`, `ensure_refusal.go:122-187`, `result.go:573-588`, `cli/branch.go:209-276`)
- `resolveMode`: registro > `--mode` > hub > defecto; con hub=`ask` y sin respuesta devuelve `branch.DecisionRequired` (:123); `--mode` distinto del registrado es `ModeConflict` → `plan_conflict` (:109-110). Es exactamente el patrón pedido para "cambiar el modo una vez creada la rama se rechaza".
- `Decision{Question, Choices[]DecisionChoice{Answer,Label,Effect,Invocation}}` es genérico, pero `modeInvocation` (`ensure_refusal.go:122-128`) y los textos están cableados a `--mode`. Hace falta un `promotionInvocation` análogo y que cada invocación conserve las respuestas ya dadas (`--mode`, `--prefix`/`--branch`/`--base`; hoy `modeInvocation` NO repite las excepciones, un hueco previo).
- CLI: flags en `cli/branch.go:210-223` con `flag.NewFlagSet`; `--mode` se valida en :246-255 detectando "presente pero vacío" con `flags.Visit`; `branchBlankFlag` (:256) para prefix/branch/base/repo; `branchRepeatedFlag` para `--repo` (:219). `EnsureRequest` (`ensure.go:27-36`) es donde viaja. Ayuda en `branchEnsureHelp` (no leída, localizar en `cli/`).

**Registro local** (`internal/branch/registry/registry.go`, `types.go:166-203`, `engine/ensure_registry.go`)
- Fichero JSON por carril y cambio en `<git common dir>/axiom/branch/v1/<carril>/<cambio>.json`; `Record{Schema, Lane, Change, Mode, ModeSource, Exceptions, ArchivePrefix, PlanningRepo, Repositories[]RecordRepo{Name, GitRoot, CommonDir, Role, Branches}}`.
- `Load` ignora campos desconocidos (:279 "el esquema v1 puede crecer"), `validate` (:280-331) exige modo/origen/rutas absolutas. Añadir un campo `omitempty` en `RecordRepo` es compatible hacia delante y hacia atrás.
- **Es estado derivado y reconstruible**: `newRecord` (`ensure_registry.go:25-47`) recompone el registro entero en cada `ensure` y solo conserva `ArchivePrefix` y las ramas de archivo del previo; los errores de registro dicen "bórralo, es derivado" (`engine.go:272-288`). Un modo de promoción que provenga de una respuesta humana NO es derivable: borrar el registro lo pierde.
- El registro está en el git common dir: no se versiona ni viaja a otra máquina ni a otra persona.
- Lock por cambio (`Lock`, :53-70), escritura atómica (`Save`, :92-141).

**Instantánea del kickoff** (`internal/kickoff/gitsnapshot.go`, `cli/sdd_kickoff_branch.go`, `engine/snapshot.go`)
- `GitSnapshot{IntegrationBranch, BranchPrefix, Exceptions, ChangeBranch, Repositories[]GitSnapshotRepo{Path, Name, Role, IntegrationBranch, Base, Branches}}` (:24-63); `Validate` (:75-140) con vocabularios cerrados. Aditivo: "un binario anterior lo ignora sin error" (:19-21).
- `Engine.Snapshot` (`snapshot.go:30-78`) calcula la instantánea desde el plan con `sealing=true` (incluye los repositorios de código de los roles que se sellan aunque aún no tengan rama). `entry.Base = repo.IntegrationBranch` (:61), o `BaseRef` si hay `--base` (:63-65). Hoy `Base` no se consume al planificar.
- Se escribe **una vez**, en `kickoff seal` (`sdd_kickoff_branch.go:60-68`, tras la guarda A), y REQ-25.36 prohíbe expresamente que "NINGUNA otra orden, incluida `ensure`" la modifique.
- **Orden real**: `ensure` va ANTES del `seal` (doctrina CLAUDE.md del repo y REQ-25.35 `sdd-lane-kickoff`). Pero antes del sellado el plan solo contiene la rama del cambio del planning home: `selectRepositories` (`planner.go:230-251`) solo añade repositorios de código si `len(sealed) > 0`, y `sealedRoles` (:273-292) devuelve nada sin kickoff. Es decir, **en multirepo SDD las ramas de los repositorios de código se crean en un segundo `ensure`, posterior al sellado**, cuando la instantánea ya es inmutable. En monorepo embebido el repositorio es el planning home y se crea en el primer `ensure`, antes del sellado.

**Scripts de dogfooding** (`scripts/axiom-worktree.ps1`, `.sh`)
- ps1: `done` ejecuta `axiom branch status --json` y parsea JSON (:223-239), `axiom branch done --fetch` (:244) y después `if ($currentBranch -eq 'main') { fetch origin main; merge --ff-only origin/main }` (:249-260). sh: :220-225, con parseo por `sed/awk/grep` sobre JSON de sangrado fijo (`json_paths`, :111-113; el `awk` de `pr` depende de columnas exactas, :145-152).
- `new` ya usa `--mode worktree` (ps1:102, sh:124); `pr` toma la base de `prTarget` del status (sh:143-166), por lo que se adapta solo si `status` da el destino dinámico.
- `status.repositories[].integrationBranch` ya está en el JSON (`result.go:180`), de modo que el script puede leer la integración resuelta sin nuevo contrato. `$status` ya está parseado en ps1 (:226) pero con alcance de `try`; en sh `STATUS` ya se captura en :206.
- Tests: `internal/workspace/scripts_axiom_worktree_test.go` (stubs de `axiom` con `status.json`; `demoChange(f, "inc/demo", "main", true)`, fixture `newWrapperFixture`), casos `...ShellDoneRunsAxiomBranchDone` (:346), `...PrTakesTheBaseFromPrTarget` (:300), `...PowerShellDrivesAxiomBranch` (:482); además `docs_contract_test.go:340-417` pinea el contenido de los scripts. Existe una memoria del repo de que `axiom-worktree.ps1 done` falló tras un squash; la capa 2 + `--fetch` lo cubre pero conviene un caso de test con squash real.

**Spec viva**
- `openspec/specs/change-branch-lifecycle/spec.md`: tabla "de un vistazo" :82-110; REQ-25.3 :211 (política), REQ-25.5 :309 (specs in situ), REQ-25.7 :371, REQ-25.8 :427 (ask/`--mode`), REQ-25.10 :502 (base), REQ-25.12 :605, REQ-25.13 :722 (ODD), REQ-25.14 :750 (registro), REQ-25.15 :777 (status), REQ-25.16 :816 (capas), REQ-25.17 :872 (done seguro), REQ-25.45 :655 (archivo), REQ-25.24 :1295 (dogfooding).
- Otras specs afectadas: `workspace-topology` REQ-25.26 (:252, bloque `git:`) y REQ-25.27 (:280, `repositories:`); `sdd-lane-kickoff` REQ-25.36 (:196, instantánea; "una sola vez"), REQ-25.35 (:159); `local-web-dashboard` REQ-25.33 (:203, `git:`/`repositories:` solo lectura); `organic-agent-trigger-rules` REQ-25.40.
- **Numeración**: el máximo REQ-25.x en uso es REQ-25.45 (grep sin resultados para 25.46+). Siguiente libre: **REQ-25.46** (las capacidades afectadas ya tienen su serie 25; el backlog menciona REQ-24.29 por otra capacidad).

**Docs y tests que fijan claves/textos**
- `axiom.example.yaml:35-42` (bloque `git:`) y :73-85 (`repositories:`); `docs/manual-de-inicio.md:317-331` (`git:`) y :333-350 (`repositories:`; "una clave desconocida dentro de `git:` es un error").
- `internal/workspace/docs_contract_test.go:167-200` pinea literalmente líneas de ambos documentos (`"git:\n  integration_branch: main"`, `"rigen \`integration_branch: main\` y \`branch_prefix: feat\`"`, `"    integration_branch: release"`, `"# git:"`...). Añadir texto es seguro; reformular las líneas pineadas, no.
- `internal/workspace/dogfooding_branch_status_test.go:24-37` pinea `origin/main`/`main` del propio repo.
- `axiom.yaml` del repo: `git.integration_branch: "main"` (:12); sin `main_branch` el comportamiento queda idéntico.
- Doctrina de skills a revisar: `internal/assets/skills/sdd-apply/SKILL.md:99,266` (afirma que "la rama de seguimiento apunta a la integración" y que el destino sale de `prTarget`), `sdd-archive/SKILL.md:93,98`, y `_shared/sdd-status-contract.md:187`; hay `assets_test.go` que puede fijar textos.

**Tests de rama existentes para extender** (helpers de fixture git)
- Motor: `engine/status_test.go` (`runGit` :102, `commit` :111, `newRepository` :134, `initRepository` :142, `newProject` :170, `runTests` :46); `engine/status_registry_test.go` (`writeRegistry` :73, `newEngine(goos)` :94, `newMultirepo(t, roles, codeRepositories...)` :362); `ensure_exception_test.go:28 commitAll`; `done_layers_test.go:27 commitFile` y casos de squash; `ensure_multirepo_test.go`, `ensure_mode_test.go`, `ensure_odd_test.go`, `ensure_registry_test.go`, `done_test.go`, `publication_test.go`, `handoff_test.go`, `snapshot_test.go`, `integration_test.go`.
- Puros: `branch/policy_test.go`, `planner_test.go`, `planner_snapshot_test.go`, `merge_test.go`, `result_test.go`, `ensure_decision_test.go`, `registry/registry_test.go`.
- CLI: `cli/branch_mode_test.go` (validación de `--mode`, `runBranchCLI`, `branchGitOutput`), `branch_ensure_test.go`, `branch_done_layers_test.go`, `sdd_kickoff_branch_test.go`.
- Workspace: `gitpolicy_test.go` (tablas de claves/errores :80-120, :414-430, :487-539), `scripts_axiom_worktree_test.go`.
- Kickoff: `kickoff/gitsnapshot_test.go`.

### 2. Hechos que condicionan la decisión de persistencia

1. `ensure` corre antes del `seal` y el sello es inmutable (REQ-25.36).
2. En monorepo embebido SDD, la rama del repositorio único se crea en el primer `ensure` (antes del sello): la respuesta a `ask` existe antes del sello y podría copiarse a la instantánea.
3. En multirepo SDD, las ramas de los repositorios de código nacen después del sello: cualquier respuesta `ask` de esos repositorios llega con la instantánea ya cerrada y NO puede escribirse en ella sin violar REQ-25.36. La instantánea sí enumera esos repositorios en el sello (`Snapshot` con `sealing=true`), pero no puede conocer su respuesta.
4. ODD no tiene kickoff ni instantánea (`sealedRoles` devuelve vacío en ODD, `engine.go:322-325`): el registro local es el único sitio de persistencia estructurada. Existe además el documento versionado `odd/tasks/<feature>.md`, que no tiene parser Go.
5. El registro es local y derivado; la instantánea es versionada y compartida (viaja con el cambio a otra máquina: `axiom branch ensure` en otra máquina recrea la rama desde la base fusionada).
6. La respuesta de `ask` del modo (worktree/in_place) ya se persiste en el registro por cambio (`ModeSource: answer`) y "un modo distinto se rechaza": mismo ciclo de vida que se necesita.

### 3. Aproximaciones para persistir el modo resuelto

| Opción | Pros | Contras | Esfuerzo |
|---|---|---|---|
| A. Solo registro local (`RecordRepo.Promotion`, `PromotionSource`) | Único sitio válido en ODD y para repos de código de multirepo post-sello; reutiliza `resolveMode`/`ModeConflict`/lock/`Save`; campo `omitempty`, compatible hacia atrás | No viaja a otra máquina ni persona (re-pregunta en `ask`, riesgo de elegir distinto); no derivable si se borra el registro | Bajo-medio |
| B. Solo instantánea del kickoff | Versionado y compartido; inmutable por diseño | Imposible en ODD; imposible para la respuesta de repos de código de multirepo SDD (post-sello, REQ-25.36 prohíbe reescribir); `ensure` no puede escribir ahí | No viable por sí sola |
| C. Ambos: registro como fuente operativa, instantánea como copia para lo conocido al sellar | Cubre ODD y multirepo; en monorepo SDD y repos con decisión pre-sello el modo viaja con el cambio y es recuperable si se borra el registro; coherente con la doctrina "instantánea > registro > política" (D-10) | Dos fuentes: hay que fijar precedencia y qué hace `ensure` si registro e instantánea discrepan; cambia el contrato de REQ-25.36 (campo nuevo, sigue escrito una vez) | Medio |
| D. A + pista en `odd/tasks/<feature>.md` | Da portabilidad a ODD sin Go | Texto libre sin parser, deriva frágil, acopla Go a doctrina de agente | Medio, no recomendable |

**Recomendación: C con el registro como fuente operativa.**
- Fuente de verdad en ejecución: `Record.Repositories[i].Promotion` (+ `PromotionSource`: `answer|flag|repository|git|default` análogo a `ModeSource*`). Precedencia al resolver por repositorio: registro > `--promotion` > `repositories[].promotion` > `git.promotion` > defecto. Defecto cuando no hay `main_branch`: una sola etapa (sin estado de promoción, comportamiento actual). Con `main_branch` y sin `promotion`: hay que fijar el defecto (el backlog no lo dice; candidato `release`, que es el comportamiento de hoy con integración distinta de main, o `ask`). Es una decisión de producto pendiente.
- Copia en la instantánea: `GitSnapshotRepo.Promotion` y `GitSnapshot.MainBranch`, ambos `omitempty` y rellenados en `Engine.Snapshot` desde el registro cuando el repositorio ya tiene decisión al sellar (monorepo, repositorio del planning home), vacíos si no. `ensure` no modifica la instantánea.
- Si el registro falta y la instantánea trae el modo, el registro se reconstruye desde ella (nueva capa entre "registro" y "política" en D-10). Si faltan ambos y la rama del repositorio ya existe, la inferencia desde git no es fiable; recomendación: tratar `--promotion` como declaración (sin verificar) y avisar, o rechazar con `plan_conflict`.
- Alternativa a valorar: dejar fuera de la instantánea (A pura) si se acepta que otra máquina re-pregunte. Es la opción más barata y la que menos toca contratos versionados; el coste es la inconsistencia entre personas, que el kvp25 del backlog sí sufre (decisión por cambio y repositorio con dos personas).

### 4. Puntos de extensión concretos

1. `workspace/gitpolicy.go`: `GitConfig.MainBranch`, `GitConfig.Promotion`, `RepositoryPolicy.Promotion` (¿también `MainBranch` por repositorio? el backlog solo lo pide global); ampliar `gitKeys`, `repositoryKeys`, `branchValidator`/`repositoryValidator`, los dos `values[...]` y los textos de error literales; `resolveGit` NO debe rellenar `main_branch` por defecto (ausencia = una etapa). Nuevo validador de enumerado `branch|release|ask`; validar `main_branch != integration_branch` y que `promotion` sin `main_branch` sea error o aviso explícito (decisión de producto).
2. `branch/policy.go`: `Policy.MainBranch/Promotion`, `RepoPolicy` con los dos campos, herencia en `ResolvePolicy`/`For`.
3. `branch/planner.go`: `PlanInput.Promotion map[repoKey]Promotion` (ya resuelto por el motor, porque el planificador es puro), `RepoPlan.Promotion`, `RepoPlan.MainBranch` y un campo de base `RepoPlan.BaseBranch` calculado (main_branch en `branch`, integración en `release`/una etapa), manteniendo `IntegrationBranch` intacto. `RepoSpecsInPlace` se queda exento (:367-369). Ramas de rol sin cambio (:395-412).
4. `engine/status.go`: `baseCandidates` usa `BaseBranch`; `describeRepository` calcula el "siguiente PR" con git (¿rama del cambio fusionada en integración? ascendencia o `MergeTreeEquivalent`, solo referencias locales, sin red) y rellena `prTarget` y campos aditivos (p. ej. `repositories[].promotion`, `repositories[].mainBranch`, `branches[].prStage`). Sacar este cálculo a una función compartida que reutilicen `PublicationTarget` y `HandOff`.
5. `engine/done.go`: `mergeEvidence` recibe el destino de confirmación (`main_branch` en `branch`, integración en `release`/una etapa) en vez de `repo.IntegrationBranch` (:443-446); `Switch` de `in_place` (:389) sigue yendo a la integración; `doneBranches` (:228) debe propagar el destino; `done_refusal.go:38` informa el destino correcto.
6. `engine/prefs.go` (+ `ensure.go`): `resolvePromotion` por repositorio, tras `gather` y antes de `Plan`, con la misma prioridad que `resolveMode`; `ask` sin respuesta devuelve `decision_required`; respuesta distinta de la registrada con rama creada → `plan_conflict` (nuevo `PromotionConflict` análogo a `ModeConflict`). Debe ejecutarse sobre los repositorios que `selectRepositories` incluiría (planning home, código post-sello, ODD `--repo`), excluyendo el de specs.
7. `branch/ensure_refusal.go`: `DecisionRequired` genérico o `PromotionDecisionRequired` con `Details.repository`, y `promotionInvocation` que conserve `--mode` ya dado.
8. `cli/branch.go`: flag `--promotion`; para multirepo conviene la forma repetible `--promotion <repo>=branch|release` con valor sin prefijo aplicable al único repositorio pendiente; validación igual que `--mode` (presente pero vacío, `ask` no es respuesta). Actualizar `branchEnsureHelp` y `ensureInvocation` (`ensure_exception.go:17-32`).
9. `branch/types.go`/`registry/registry.go`: `Promotion`, `PromotionSource`, `validate` (vocabulario) y `Record` en `newRecord`/`recordRepository` (conservar el valor del registro previo igual que `ArchivePrefix`).
10. `kickoff/gitsnapshot.go`, `engine/snapshot.go`: campos aditivos y `Validate`.
11. `labels.go`: etiquetas en castellano de los tokens nuevos; `result.go`: texto de `WriteText`.
12. Scripts: `ps1:249-260` y `sh:220-225` pasan a leer `repositories[0].integrationBranch` del `status` ya capturado (en `done` el status se lee antes de `done`); si no hay JSON legible, caer a `main` como hoy para no romper `new`/`pr`/`done`. Los scripts actuales usan `--mode worktree`; con `promotion: ask` `new` recibiría `decision_required`: el script debería reenviar `--promotion` o fallar con un mensaje accionable (el propio repo no declara `main_branch`, no le afecta).
13. Spec: nuevo REQ-25.46 (política `main_branch`/`promotion` en `workspace-topology` + resolución por repositorio), REQ-25.47 (`ensure`: base y `ask`/`--promotion`), REQ-25.48 (`status`: siguiente PR), REQ-25.49 (`done`: destino por modo, squash). Modificar REQ-25.3, 25.8, 25.14, 25.15, 25.16, 25.17, 25.26, 25.27, 25.36 y 25.24. Docs: `axiom.example.yaml`, `manual-de-inicio.md`, skills `sdd-apply`/`sdd-archive`.

### 5. Riesgos

- **Squash en modo `branch`** (R1): la misma rama entra dos veces. Capa 2 (`merge-tree --write-tree`, git ≥ 2.38) puede dar falso negativo si el destino modificó después las mismas líneas; `--force` queda como salida (`done` ya lo soporta, `TipSha` para recuperar). `status` NO tiene `--force` ni capa 3: si detecta mal "ya fusionada en develop" propondrá otra vez PR a develop. Mitigación: cuando la capa 2 no está soportada o no concluye, no fingir certeza (aviso en `status`), y tests con squash real en las dos etapas.
- **`pr_merged` de la capa 3 es ambiguo**: `gates.yaml` acredita "el PR del cambio" (`done.go:501-` `prMergedEvidence`); en modo `branch` podría confirmar la fusión final con la evidencia de la primera PR. Hay que decidir si la capa 3 aplica al destino `main_branch` (probablemente no, o solo con evidencia explícita de ese destino).
- **Compatibilidad**: campos nuevos `omitempty` en registro, instantánea y JSON de `status`; un registro o kickoff sin el campo = una etapa. Un binario anterior ignora `main_branch`/`promotion` en `axiom.yaml` (claves de primer nivel desconocidas del bloque `git:`, ¡pero `git:` rechaza claves desconocidas!): un binario anterior con `git.main_branch` declarado daría `GitErr`/`config_invalid`, rompiendo equipos con versiones mezcladas. El manual afirma que "los binarios anteriores ignoran el bloque" (:331), lo cual solo vale mientras el bloque sea válido para ellos. Hay que documentarlo o aceptarlo como cambio rompedor de equipo mixto.
- **`ask` en multirepo**: si hay varios repositorios de código con `ask`, hay que decidir si se devuelve un `decision_required` por repositorio (secuencial) o uno compuesto. Recomendado secuencial y por orden del plan, con `Details.repository`. Además `ask` de modo (hub) y `ask` de promoción son dos decisiones independientes: las dos pueden encadenarse en llamadas sucesivas; el contrato "Lossless Blocking Prompts" prohíbe partir una decisión atómica, pero son dos preguntas de negocio distintas.
- **Orden ensure/seal en multirepo**: la decisión de repos de código llega después del sello (hecho 3). La instantánea no puede garantizar consistencia para ellos; sin registro (otra máquina) se re-pregunta. Esto condiciona la opción C.
- **Registro borrado** (derivado): se pierde la respuesta de `ask`; con rama ya existente, inferir el modo desde git es poco fiable.
- **ODD sin kickoff**: depende solo del registro local (R del backlog). El documento ODD no sirve como fuente.
- **Rama de archivado** (`planner_archive.go`, `archive_refusal.go`, `checkIntegrated`): sale de la integración y exige el cambio fusionado ahí (`change_not_integrated`). En modo `branch` la primera PR llega a la integración, así que el archive sigue siendo válido tras ella; hay que fijar que el archive NO esperará a la fusión en `main_branch`. Documentarlo en REQ-25.45.
- **`PublicationTarget`/`HandOff`/`livespec`** copian `prTarget` estático: si solo se hace dinámico `status`, `sdd publish`/hand-off/`livespec/context.go:82-99` seguirán con el destino de la primera etapa. Decidir si el "siguiente PR" se aplica a todos o solo a `status` (el backlog solo exige `status`).
- **Scripts**: `.sh` parsea JSON con `awk` por columnas exactas; añadir campos al JSON no rompe (aditivo), pero cualquier campo nuevo con el mismo nombre de clave que los parseados (`name`, `kind`, `worktree`, `prTarget`) alteraría el `awk`. `docs_contract_test.go` pinea contenido de los scripts (:340-417).
- **Textos pineados**: tests de `gitpolicy_test.go` comprueban los mensajes de error con las claves enumeradas; ampliar los textos exige actualizar esos casos.
- **Dashboard** (REQ-25.33): `BranchPolicyDTO`/`types.go` exponen `git:` y `repositories:` en solo lectura; no es obligatorio ampliarlos, pero un campo nuevo sin reflejar crea deriva (C-06-e lo catalogará).
- **Tamaño**: el backlog estima 700-1000 líneas para C-02-a; con la ampliación a PublicationTarget/HandOff, scripts y spec puede acercarse a 1000+. Dado el criterio del usuario (una PR con `size:exception` al pasar de 400), no se encadena.

### 6. Preguntas abiertas para el orquestador/usuario

1. Defecto de `promotion` cuando hay `main_branch` y no se declara: ¿`release` (comportamiento de hoy), `branch` o `ask`?
2. ¿`main_branch` también sobrescribible en `repositories[]`? (el backlog solo lo da en `git:`; kvp25 mezcla front/back en el mismo `master`, no hay evidencia de necesidad).
3. ¿Forma de `--promotion` en multirepo: global o `<repo>=<modo>`?
4. ¿El "siguiente PR" cambia también `publish`/`hand-off`, o solo `status`?
5. ¿Capa 3 (`pr_merged`) vale como evidencia de la fusión final en `main_branch`?
6. ¿Opción A (solo registro) o C (registro más copia en la instantánea)?

### 7. Recomendación

Avanzar a `sdd-propose` con: política `main_branch`/`promotion` en `git:` y `repositories[]`; `BaseBranch` calculado en el plan sin tocar `IntegrationBranch`; resolución por repositorio en `ensure` con el patrón `resolveMode`, persistencia en `RecordRepo` (fuente operativa) y copia opcional en la instantánea (opción C, o A si se prioriza el coste); destino de confirmación de `done` parametrizado y `status` con "siguiente PR" calculado con git local; scripts leyendo `integrationBranch` del `status`. La tolerancia a squash ya existe: el trabajo es cambiar el destino y cubrir las dos etapas con tests.

### Ready for Proposal
Sí, tras resolver al menos las preguntas 1, 4 y 6 (afectan al diseño y al tamaño). Las demás pueden fijarse en el diseño con la recomendación anterior.
