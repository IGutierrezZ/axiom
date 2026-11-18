# Diseño: promoción de ramas configurable por repositorio y por cambio (C-02-a)

## Enfoque técnico

Se sigue la opción C de la propuesta: el registro local es la fuente operativa del modo de promoción y la instantánea del kickoff guarda una copia de lo que ya se conoce al sellar. El cambio no crea mecanismos nuevos. Reutiliza cinco piezas que ya existen:

- la herencia campo a campo de `Policy.For` para `main_branch` y `promotion`;
- el patrón `resolveMode` / `ModeConflict` / `DecisionRequired` (`engine/prefs.go`, `branch/ensure_refusal.go`) para resolver el modo por repositorio, rechazar un cambio de modo con `plan_conflict` y preguntar con `decision_required`;
- el plan puro (`branch.Plan`), que recibe el modo ya resuelto y calcula la base efectiva en un campo nuevo, `RepoPlan.BaseBranch`, sin tocar `IntegrationBranch` ni el `PRTarget` estático;
- las capas de `ConfirmMerge` (ascendencia, árbol equivalente, `pr_merged` y `--force`), que `done` evalúa contra un destino que depende del modo;
- la lectura sin red de `describeRepository`, que en `status` decide la «siguiente PR» con las mismas capas 1 y 2.

El invariante central: **sin `main_branch` efectivo no cambia ni un byte**. Afecta al plan, al registro, a la instantánea y al JSON de `status`. Todos los campos nuevos son `omitempty`, y todo el código nuevo queda detrás de `RepoPlan.TwoStage()`. Ningún test actual cambia de resultado.

Hay dos decisiones cerradas por el orquestador que este diseño concreta:

- en modo `branch`, `done` no acepta la capa `pr_merged` como prueba de la fusión en `main_branch` (D7);
- `promotion` sin `main_branch` efectivo no tiene efecto, y `axiom workspace validate` lo avisa sin bloquear (D1).

## Decisiones de arquitectura

### D1. Esquema: dos claves nuevas en `git:` y una en `repositories[]`

**Elección**:

- `GitConfig` gana `MainBranch` (`main_branch`) y `Promotion` (`promotion`).
- `RepositoryPolicy` gana solo `Promotion`.
- Las listas cerradas pasan a ser:
  - `gitKeys = {"integration_branch", "branch_prefix", "main_branch", "promotion"}`;
  - `repositoryKeys = {"path", "integration_branch", "branch_prefix", "promotion"}`.
- `main_branch` dentro de `repositories[]` sigue siendo una clave desconocida y, por tanto, un error. Así queda «solo en `git:`» sin regla aparte.
- `branchValidator(key)` pasa a un `switch`:
  - `branch_prefix` → `gitref.ValidPrefix`;
  - `promotion` → `validPromotion` (nuevo; acepta exactamente `branch`, `release` y `ask`, distingue mayúsculas y su error enumera los tres valores);
  - el resto (`integration_branch`, `main_branch`) → `gitref.ValidRefName`.
- `resolveGit` copia `MainBranch` y `Promotion` tal como se declararon, **sin valor por defecto**. El defecto `branch` lo aplica la resolución del motor, que así conserva el origen (`default`).
- Los textos de error que enumeran las claves admitidas se amplían (`gitpolicy.go:89`, `:114`, `:128`). Los tests que los comprueban (`gitpolicy_test.go:146`, `:176`, `:539`) añaden las claves nuevas a su lista `wantErrorContains`.
- Nueva función pura `PromotionWarnings(cfg *WorkspaceConfig) []string` en `gitpolicy.go`. Avisa en dos casos:
  - `git.promotion` declarado sin `main_branch`, o con `main_branch` igual a `integration_branch`;
  - `repositories[i].promotion` declarado cuando la integración efectiva de esa entrada (la suya o la heredada) es igual a `main_branch`, o `main_branch` no existe.
- `cmd/axiom/main.go` (junto a `ValidateRepositoryPolicies`, `:606`) añade esos avisos a `report.Warnings`, sin tocar `report.Valid`.

**Alternativas consideradas**: un `main_branch` por repositorio (fuera del alcance de la propuesta); que `promotion` sin `main_branch` sea un error (descartado por la decisión cerrada); rellenar `promotion: branch` en `resolveGit` (se perdería el origen `default`).

**Justificación**: es el patrón que ya existe (`decodeStringKeys` con un validador por clave y errores acumulados). Las claves cerradas impiden por sí solas un `main_branch` por repositorio.

### D2. Política: `MainBranch` y `Promotion` con la herencia actual

**Elección**: `branch.Policy` gana `MainBranch string` y `Promotion Promotion`, y `branch.RepoPolicy` gana los mismos dos campos.

- `ResolvePolicy` copia `cfg.Git.MainBranch` y `cfg.Git.Promotion`, y carga `RepositoryPolicy.Promotion` en `PerRepository`.
- `For(rel)` hereda campo a campo con la regla actual (el valor vacío hereda). `MainBranch` siempre sale del workspace.
- `repoPolicy()` (`planner.go:445`), para un repositorio fuera de la carpeta maestra, propaga también `MainBranch` y `Promotion` del workspace.
- `RepoPolicy` gana un campo informativo, `PromotionSource`: `repository` si lo declara la entrada y `git` si lo hereda del bloque. Así la resolución sabe de qué capa sale el valor declarado.

**Alternativas consideradas**: resolver la herencia en el motor (duplicaría `For`).

**Justificación**: un solo punto de herencia, probado con tablas en `policy_test.go`.

### D3. Tokens y vocabulario cerrado (`internal/branch/promotion.go`, nuevo)

**Elección**: `type Promotion string` con tres valores:

- `PromotionBranch = "branch"`;
- `PromotionRelease = "release"`;
- `PromotionAsk = "ask"`: es valor de política y, en el plan, la marca de «pendiente».

`workspace` define los mismos literales como constantes (`workspace.PromotionBranch`…), porque `branch` importa `workspace` y no al revés. `kickoff` repite los dos valores materializables (`branch`, `release`) en su propio mapa, igual que hoy repite papeles y tipos de rama (`gitsnapshot.go:67-70`).

Los orígenes (`PromotionSource*`) son estos:

| Token | Cuándo |
|---|---|
| `answer` | `--promotion` cuando la política del repositorio es `ask` |
| `flag` | `--promotion` cuando la política no es `ask` |
| `repository` | `repositories[].promotion` |
| `git` | `git.promotion` |
| `default` | nada declarado: `branch` |
| `snapshot` | tomado de la instantánea porque no había registro |
| `legacy` | registro o instantánea sin el campo y rama ya creada: una sola etapa, que equivale a `release` |

Los dos estados de `status` (`PromotionStage*`) son `integration` (la siguiente PR va a la integración) y `main` (la siguiente PR va a la rama principal).

Las etiquetas van en un mapa propio, `promotionLabels`, con la función `PromotionLabel(token)`, y **no** en el `labels` global. El motivo es que `flag` ya está etiquetado allí como «bandera --mode» y `ask` como «preguntar» del modo de trabajo; mezclarlos daría una etiqueta falsa.

**Alternativas consideradas**: reutilizar `Label()` (choque de tokens); un tipo `PromotionSource`, que se descarta porque el registro guarda `ModeSource` como `string` y se sigue el mismo patrón.

**Justificación**: tokens estables en inglés y etiqueta en un campo hermano, como en REQ-25.2, sin colisiones.

### D4. Resolución por repositorio: función pura y precedencia

**Elección**: la resolución vive en una función pura, `branch.ResolvePromotion(PromotionFacts) PromotionResolution`, probada con tablas. El motor (`engine/prefs.go`, junto a `resolveMode`) reúne los hechos y traduce el resultado a rechazos. El algoritmo, por repositorio, es este:

1. **Exento o de una etapa.** Si es el repositorio de specs de un multirepo (`RepoSpecsInPlace`) o `!TwoStage` (sin `main_branch`, o igual a su integración): `Mode = ""` y no hay nada que resolver. Un `--promotion` cualificado que lo nombre solo produce un aviso.
2. **Registro.** Se mira la entrada `RecordRepo` del repositorio, buscada por `CommonDir`:
   - con `Promotion` → ese modo y su origen (`Recorded = true`);
   - sin `Promotion` pero con una rama `change` registrada → `release` con origen `legacy` (`Recorded = true`). Es la migración retroactiva de la propuesta: un cambio registrado antes sigue siendo de una etapa.
   - Si hay modo y `--promotion` pide otro: **conflicto** (`plan_conflict`). Si pide el mismo, la operación es idempotente.
3. **Instantánea.** Se mira la entrada `GitSnapshotRepo` del repositorio, buscada por `Path`:
   - con `Promotion` → ese modo, con origen `snapshot`;
   - sin `Promotion`, con la instantánea sellada sin `main_branch` (`GitSnapshot.MainBranch == ""`, ver D9) y la rama del cambio ya existente en local → `release` / `legacy`;
   - si no, la instantánea no aporta nada y se sigue.
   - Mismo conflicto que con el registro.
4. **Bandera.** `--promotion` aplicable al repositorio. El cualificado `<repo>=<modo>` prevalece sobre el que no lo está. El origen es `answer` si la política del repositorio es `ask` y `flag` si no.
5. **Política.** `RepoPolicy.Promotion` declarado (`branch` o `release`), con su origen (`repository` o `git`).
6. **`ask` sin respuesta.** `Mode = ask` (pendiente).
7. **Nada declarado.** `branch` con origen `default`.

El resultado de cada repositorio entra en `PlanInput.Promotions`, un mapa indexado por `GitRoot`. Un repositorio que falta en el mapa se planifica como de una etapa a efectos de base (`BaseBranch = IntegrationBranch`). `MainBranch` sigue siendo informativo. Esto conserva intactos los consumidores que la propuesta deja fuera: `PublicationTarget`, `HandOff` y `livespec`.

**Alternativas consideradas**:

- Resolver dentro de `branch.Plan`: obligaría a meter en la entrada pura el registro, la existencia de ramas y la interactividad.
- Duplicar `selectRepositories` en el motor: dos fuentes de la selección.
- El marcador de la instantánea (paso 3), frente a la regla literal «sin campo, una etapa»: la regla literal trataría como de una etapa un repositorio de código de multirepo cuya respuesta llegó **después** del sellado si se borra el registro. En modo `branch`, `done` lo borraría antes de llegar a `main_branch`, justo el fallo que el cambio corrige.

**Justificación**: es la precedencia de la propuesta (registro > instantánea > `--promotion` > `repositories[].promotion` > `git.promotion` > defecto `branch`). Es pura y tabulable, y con la regla `legacy` cumple «sin el campo, una sola etapa; con la política vigente si la rama aún no existe».

### D5. Planificar dos veces: plan provisional, resolución y plan definitivo

**Elección**: el motor llama a `branch.Plan` dos veces. La función nueva `(*Engine).planWithPromotion(ctx, facts *changeFacts, req promotionRequest) (branch.ChangePlan, *branch.Refusal)` hace esto:

1. `branch.Plan(facts.input)` con `Promotions == nil`. Da los repositorios, sus nombres, su papel y el nombre de la rama del cambio. Ninguno depende del modo.
2. Por cada repositorio del plan, reúne `PromotionFacts`:
   - del registro (`facts.record`);
   - de la instantánea (`facts.input.Snapshot`);
   - de la rama del cambio en local (`e.git.BranchExists`), solo cuando la regla `legacy` de la instantánea la necesita;
   - de la petición;
   - de `repoPolicy`.
   Después llama a `branch.ResolvePromotion`.
3. Valida la petición: un `--promotion <repo>=…` cuyo `<repo>` no es ningún repositorio del plan es `invalid_name`, con la lista de nombres válidos.
4. En modo interactivo (`ensure`), el primer repositorio, en el orden del plan, que queda `ask` produce `decision_required` y no se crea nada. El primer conflicto produce `plan_conflict`.
5. Fija `facts.input.Promotions` y vuelve a llamar a `branch.Plan`.

`Ensure`, `Status`, `Done` y `Snapshot` usan `planWithPromotion`; `PublicationTarget` y `HandOff`, no. En `Ensure`, el segundo `gather` tras `apply` (`ensure.go:166`) **reaplica** el mapa ya resuelto a `facts.input.Promotions` antes de replanificar, sin volver a resolver.

**Alternativas consideradas**: una función pura `ApplyPromotion(&plan, …)` que parchee el plan. Es más corta, pero rompe «el plan puro recibe el modo resuelto y calcula la base».

**Justificación**: `branch.Plan` es pura y barata, y `Ensure` ya planifica dos veces. La base efectiva se calcula en un solo sitio (`planRepository`).

### D6. Base efectiva en `RepoPlan.BaseBranch`; `IntegrationBranch` no cambia

**Elección**: `RepoPlan` gana cuatro campos:

- `MainBranch`: `git.main_branch` efectivo, vacío con una etapa;
- `Promotion`: `""` (una etapa o sin resolver), `branch`, `release` o `ask` (pendiente);
- `PromotionSource`;
- `BaseBranch`: `MainBranch` si `Promotion == branch`; si no, `IntegrationBranch`.

`planRepository` los rellena desde `in.Promotions[facts.GitRoot]`. El repositorio de specs (`planner.go:367`) y la rama de archivado (`planner_archive.go:89`) siguen sin modo. `BaseRef` (`--base`) sigue prevaleciendo sobre todo.

Se añaden dos métodos:

- `func (r RepoPlan) TwoStage() bool { return r.MainBranch != "" && r.MainBranch != r.IntegrationBranch }`;
- `func (r RepoPlan) Base() string`, que devuelve `BaseBranch` o, si está vacía, `IntegrationBranch`.

`Base()` existe para los `RepoPlan` construidos a mano: `PendingArchive` (`pending.go:64`), el plan de archivado y muchos tests.

Además:

- `baseCandidates` (`status.go:170-182`) usa `repo.Base()` en lugar de `repo.IntegrationBranch`. Es el único cálculo de base, así que `ensure`, `describeRepository` y `checkPlan` lo heredan.
- `BaseNotFound` (`ensure_refusal.go:25`) nombra la rama principal y `git.main_branch` cuando `Base() != IntegrationBranch`.
- `missingBaseWarning` (`status.go:202`) hace lo mismo.
- `PRTarget` de la rama del cambio sigue siendo `effective.IntegrationBranch` (`planner.go:392`). Las ramas de rol no cambian: siguen saliendo de la rama de seguimiento y apuntando a ella.

**Alternativas consideradas**: redefinir `IntegrationBranch` como «base», que rompe `PublicationTarget`, `HandOff`, la guarda (`verdict.go`) y el repositorio de specs (`ensure_repos.go:219`).

**Justificación**: la base es la única semántica nueva del plan. Todo consumidor de la integración conserva su significado.

### D7. `done`: destino de la confirmación según el modo, sin `pr_merged` para `main_branch`

**Elección**: una función pura nueva, `branch.MergeDestination(repo RepoPlan, planned BranchPlan) (destination string, prMergedApplies bool)`:

| Rama | Destino | Capa 3 (`pr_merged`) |
|---|---|---|
| `role` | `planned.From` (seguimiento, local) | sí (como hoy) |
| `archive` | `IntegrationBranch` | no (como hoy) |
| `change`, `TwoStage()` y `Promotion` ∈ {`branch`, `ask`} | `MainBranch` | **no** (decisión cerrada) |
| `change`, en cualquier otro caso | `IntegrationBranch` | sí (como hoy) |

`mergeEvidence` (`done.go:441`) usa `destination` en sus candidatos (`origin/<destino>`, `<destino>`) y solo rellena `evidence.PRMerged` si `prMergedApplies`. `ConfirmMerge` no cambia de firma: la exclusión la decide `MergeDestination`, que se prueba con tablas.

`Promotion == ask` (pendiente) solo aparece en `done` sin registro ni instantánea. En ese caso se toma el destino **más exigente**, `main_branch`, porque borrar una rama es irreversible y `--force` sigue siendo la salida. `MergeNotConfirmed` gana el parámetro `destination string` (sustituye a `planned.PRTarget`, `done_refusal.go:38`) para el mensaje, la orden `git log <destino>..<rama>` y `Details.target`.

El `Switch` de `in_place` (`done.go:389`) y `DoneCheckedOutInMainCheckout` siguen yendo a la integración. La tolerancia a squash es la que ya existe: la capa 2 contra el destino, con `--force` como salida.

Las capas 1 y 2 se extraen a `(*Engine).mergeLayers(ctx, dir, branchRef string, targets []reference) (existing []string, evidence branch.MergeEvidence, err error)`. Así `done` y `status` comparten la misma evidencia; el aviso de la capa 2 sin soporte lo emite cada consumidor.

**Alternativas consideradas**: que `pr_merged` acredite `main_branch` con una evidencia nueva en `gates.yaml` (descartado por la decisión cerrada); confirmar contra la integración cuando `ask` está pendiente (puede borrar la rama antes de tiempo).

**Justificación**: `done` solo borra cuando el destino final la tiene. La capa 3 acredita la PR del cambio hacia la integración, nunca la segunda PR.

### D8. `status`: siguiente PR con git local, sin fingir certeza

**Elección**: en `describeRepository`, tras describir las ramas, para un repositorio con `TwoStage()` y `Promotion == branch`, la rama `change` que existe recibe un estado nuevo, `promotionStage`. Lo decide la función pura `branch.PromotionStage(repo RepoPlan, aheadOfMain *int, evidence MergeEvidence) (stage, target string, inconclusive bool)`, con estas reglas:

- `aheadOfMain == nil` o `*aheadOfMain == 0`: `integration`, destino la integración. La rama no lleva nada que la rama principal no tenga. Esto evita que una rama recién creada desde `master` parezca ya fusionada en `develop`, porque `master` suele ser ascendiente de `develop`.
- La rama está fusionada en la integración por la capa 1, o por la capa 2 con soporte: `main`, destino `MainBranch`.
- La capa 2 no tiene soporte y la capa 1 no confirma: `integration` e `inconclusive = true`. `status` añade un aviso («no se puede comprobar sin git 2.38…; la siguiente PR puede ser ya hacia `<main>`»).
- En cualquier otro caso: `integration`.

`aheadOfMain` sale de `AheadBehind` contra la referencia de la rama principal (`origin/<main>` y luego `<main>`, resuelta con `baseCandidates` sobre una copia del plan con `BaseRef` vacía). Sin `--base` coincide con `ComparedTo`, que ya se calcula. La evidencia de la integración sale de `mergeLayers` contra `origin/<integración>` y `<integración>`.

Con `stage == main`, `BranchResult.PRTarget` pasa a `MainBranch`. Es la siguiente PR: head = rama del cambio, base = `main_branch`. El `pr` de los scripts la toma sin cambios. El texto de `status` añade «siguiente PR hacia `<main>`».

Con `Promotion == ask` pendiente (solo sin registro), `status` no calcula etapa y avisa: «el modo de promoción de `<repo>` está pendiente: `axiom branch ensure` lo preguntará».

El JSON gana, de forma aditiva y `omitempty`:

- en `RepositoryResult`: `mainBranch`, `promotion`, `promotionLabel`, `promotionSource` y `promotionSourceLabel`;
- en `BranchResult`: `promotionStage` y `promotionStageLabel`.

Ninguna clave se llama `name`, `kind`, `worktree` ni `prTarget`, de modo que el `awk` de `axiom-worktree.sh` (`:145-152`) no cambia de comportamiento.

**Limitación documentada**: tras un squash en `main_branch`, la rama sigue «por delante» de la rama principal. `status` mantiene `main` hasta que `done` la confirma por árbol equivalente; sin red no se puede distinguir mejor.

**Alternativas consideradas**:

- Un objeto `nextPullRequest {head, base}`: duplica `name` y `prTarget`, y una clave `base` anidada confundiría el `sed` de los scripts.
- Calcular la etapa en `PublicationTarget` y `HandOff`: queda fuera del alcance (decisión 5 de la propuesta).

**Justificación**: es la propuesta («tras confirmar con git local que está fusionada en la integración…; si no concluye, no se finge certeza»), con la protección de la rama recién creada que la ascendencia sola no da.

### D9. Registro e instantánea: campos `omitempty`, escritura y copia

**Elección — registro**:

- `RecordRepo` gana `Promotion Promotion `json:"promotion,omitempty"`` y `PromotionSource string `json:"promotionSource,omitempty"``.
- `recordRepository` (`ensure_registry.go:51`) los toma de `repo.Promotion` y `repo.PromotionSource` del plan definitivo, nunca `ask`.
- Si el plan no tiene modo (repositorio de una etapa) y el registro previo sí lo tenía, **conserva el previo**, igual que `newRecord` conserva `ArchivePrefix`. Si `main_branch` vuelve, la respuesta sigue ahí.
- `registry.validate` (`registry.go:280`) admite en `promotion` el vocabulario cerrado {`""`, `branch`, `release`} y en `promotionSource` {`""`, `answer`, `flag`, `repository`, `git`, `default`, `snapshot`, `legacy`}. Un valor desconocido produce el `config_invalid` de siempre: «bórralo, es derivado».
- Sin `main_branch`, los bytes del registro son idénticos a los de hoy.

**Elección — instantánea**:

- `GitSnapshotRepo` gana `Promotion string `yaml:"promotion,omitempty"``.
- `GitSnapshot` gana `MainBranch string `yaml:"main_branch,omitempty"``: la rama principal declarada al sellar, que es el marcador de D4.
- `Engine.Snapshot` (`snapshot.go:30`) usa `planWithPromotion` sin interacción. Copia `Promotion` **solo** cuando la resolución viene del registro (`Recorded`) y es `branch` o `release`, es decir, cuando la rama ya se creó antes del sellado (monorepo SDD).
- `entry.Base = repo.Base()`, o `BaseRef` con `--base`: la base efectiva.
- `snapshot.MainBranch = policy.MainBranch`.
- `Validate` comprueba `MainBranch` con `gitref.ValidRefName` si no está vacía, y `Promotion` contra {`branch`, `release`}.
- La instantánea se sigue escribiendo una sola vez y `ensure` nunca la modifica: solo se lee, en el paso 3 de D4.

**Alternativas consideradas**: copiar a la instantánea el modo de la política aunque la rama no exista. Congelaría una política que todavía no se ha aplicado y, en multirepo, nunca vería la respuesta posterior al sellado.

**Justificación**: es la opción C de la propuesta, con la retrocompatibilidad (`omitempty`, lectura no estricta) que ya tienen ambos ficheros.

### D10. `ensure`: `decision_required` por repositorio, `--promotion` y conservación de respuestas

**Elección**:

- **Bandera.** `--promotion` es un `branchRepeatedFlag` (`cli/branch_flags.go`). Cada valor es `<modo>` o `<repo>=<modo>`. Lo analiza una función pura, `branch.ParsePromotionRequests(values []string) ([]PromotionRequest, error)`, y son errores de uso (salida 2):
  - un valor vacío (además de `branchBlankFlag`);
  - un `<repo>` vacío;
  - un modo distinto de `branch` y `release` (`ask` no es respuesta, con el mismo texto que `--mode`);
  - dos valores sin cualificar;
  - el mismo `<repo>` dos veces.
  `--promotion` con `--archive` también es un error de uso, igual que hoy `--repo`. `EnsureRequest.Promotions []branch.PromotionRequest` lo lleva al motor, que lo revalida por defensa (`InvalidPromotion`, `config_invalid`).
- **Orden.** El orden es `gather` → `resolveMode` → `planWithPromotion` → `checkPlan`. La pregunta del modo de trabajo va primero y la de promoción después, ambas antes de cualquier `fetch` y de cualquier escritura. Con `--archive` no se resuelve la promoción: el plan de archivado no la usa.
- **`EnsureArgs`.** El tipo nuevo `branch.EnsureArgs` reúne `Operation`, `Change`, `CWD`, `Lane`, `Mode`, `Exceptions`, `Repos` y `Promotions`. Su método `Invocation()` compone `axiom branch ensure --change c [--lane odd] [--mode m] [--prefix p] [--branch b] [--base r] [--repo q]… [--promotion x]… --cwd q`, con las rutas citadas por `pathquote.Quote`. Sin banderas opcionales produce exactamente la cadena de `modeInvocation` actual.
- **Pregunta del modo de trabajo.** `DecisionRequired` pasa a recibir `EnsureArgs` y conserva en sus invocaciones las excepciones, los `--repo` y los `--promotion` dados. Las cadenas de los tests actuales, sin esas banderas, no cambian.
- **Pregunta de promoción.** `PromotionDecisionRequired(args EnsureArgs, repo RepoPlan, qualify bool) *Refusal` devuelve `Code = decision_required` con:
  - `Details = {change, repository: <GitRoot>, name: <repo.Name>, promotion: "ask"}`;
  - dos `DecisionChoice` con `Answer` igual a `branch` o `release`, `Label` de `PromotionLabel` y un `Effect` que nombra las ramas reales («sale de `origin/<main>`, su primera PR va a `<integración>` sin borrarse y la segunda a `<main>`; `done` la elimina al fusionarse en `<main>`» y «sale de `<integración>`…; el paso a `<main>` es la release»);
  - una `Invocation` que añade a `args` `--promotion <modo>` o, con `qualify`, `--promotion <repo>=<modo>`. `qualify` es cierto cuando el plan tiene dos o más repositorios con ramas.
- **Conflicto.** `PromotionConflict(args EnsureArgs, repo RepoPlan, materialized, requested Promotion, source string) *Refusal` devuelve `plan_conflict`. La continuación repite la orden con el modo materializado o sin `--promotion`. Nunca hay rebase.
- **Repositorio desconocido.** `UnknownPromotionRepository(args EnsureArgs, name string, available []string) *Refusal` devuelve `invalid_name`.
- **Avisos.** `ensure` avisa de `--promotion` sin efecto: un repositorio de una etapa, el repositorio de specs o una petición sin cualificar en un proyecto sin `main_branch`.
- **Ayuda.** `branchEnsureHelp` documenta `--promotion branch|release|<repo>=branch|release`, repetible.

**Alternativas consideradas**:

- Una sola decisión compuesta para varios repositorios: no cabe en `Decision{Choices}` y rompería la presentación sin pérdidas.
- `--promotion` solo global: kvp25 necesita elegir por repositorio.

**Justificación**: es el mismo mecanismo y esquema que la pregunta del modo de trabajo (REQ-25.8), secuencial, con un repositorio por llamada y con las respuestas previas en cada invocación propuesta.

### D11. Scripts de dogfooding: `done` usa la integración resuelta

**Elección**:

- **`axiom-worktree.ps1`.** Dentro del `try` que ya interpreta `$status` (`:225-231`), se toma `$integration` de `@($status.repositories)[0].integrationBranch`, con `'main'` por defecto antes del `try` y conservado si el JSON no se lee. El bloque `:249-260` pasa a usar `$integration`: compara `$currentBranch -eq $integration`, ejecuta `git -C $mainRoot fetch origin $integration` y `merge --ff-only "origin/$integration"`, y adapta los mensajes.
- **`axiom-worktree.sh`.** Tras capturar `STATUS` (`:206`), `INTEGRATION="$(printf '%s\n' "$STATUS" | json_paths integrationBranch | head -n 1)"` con `${INTEGRATION:-main}`. Las líneas `:220-225` usan `"$INTEGRATION"` entre comillas.
- `new`, `pr` y las cadenas que fija `docs_contract_test.go:343-371` no cambian. `pr` ya toma `prTarget`, de modo que la siguiente PR hacia `main_branch` le llega sin código nuevo.

**Alternativas consideradas**: leer `axiom.yaml` desde el script (otra fuente de la política).

**Justificación**: es REQ-25.24 con la integración del producto y la caída a `main` cuando no hay JSON. El valor llega validado por `ValidRefName`.

### D12. Documentación y doctrina

**Elección**:

- **`axiom.example.yaml`**: en el bloque `git:` (`:32-42`), dos líneas comentadas `# main_branch: master` y `# promotion: branch`, más un párrafo con la semántica de una y dos etapas, los valores y el defecto. En `repositories:` (`:73-85`), un ejemplo `promotion: ask`. No se reformula ninguna línea fijada (`docs_contract_test.go:167-200`): solo se añade texto.
- **`docs/manual-de-inicio.md`**:
  - en `git:` (`:317-331`), dos viñetas nuevas (`main_branch`, `promotion`, los dos flujos y `ask`) y un **aviso de compatibilidad** nuevo, sin tocar la viñeta fijada «Los binarios anteriores ignoran el bloque…»: «un binario anterior rechaza `git.main_branch` y `git.promotion` (`config_invalid`, clave desconocida); actualizad todo el equipo antes de declararlas o retiradlas antes de volver atrás»;
  - en `repositories:` (`:333-350`), una viñeta para `promotion`;
  - en «Ramas y Worktrees», un párrafo para `--promotion`, la siguiente PR y `done` en dos etapas.
- **Skills**, solo donde algo deja de ser cierto:
  - `sdd-apply/SKILL.md:266`: «the tracking branch targets the integration branch» se completa con «… and, in promotion mode `branch`, once merged there `prTarget` names `git.main_branch` for the second pull request»;
  - `sdd-archive/SKILL.md:98` (paso 5): en modo `branch`, `done` conserva la rama del cambio (`merge_not_confirmed`) hasta su fusión en `git.main_branch` y elimina la de archivado.
  - `_shared/sdd-status-contract.md:187` sigue siendo cierto (la publicación usa el destino de la primera etapa) y no se toca.

**Justificación**: es lo mínimo que la propuesta exige, sin reformular contenido fijado por tests.

## Flujo de datos

```
axiom.yaml ──decodeGit/decodeRepositories──► GitConfig{+MainBranch,+Promotion}
                                             RepositoryPolicy{+Promotion}
                                                   │ ResolvePolicy / For (herencia)
                                                   ▼
ensure ─► gather ─► resolveMode ─► planWithPromotion ──────────────────────────┐
                                     │ 1) Plan(Promotions=nil)  (nombres, roles)│
                                     │ 2) por repo: registro > instantánea >    │
                                     │    --promotion > repo > git > branch     │
                                     │    (ResolvePromotion, puro)              │
                                     │    ask ⇒ decision_required (1 repo)      │
                                     │    distinto ⇒ plan_conflict              │
                                     │ 3) Plan(Promotions) ⇒ RepoPlan.BaseBranch│
                                     ▼                                          │
                         checkPlan/resolveBase(baseCandidates(repo.Base()))     │
                                     ▼                                          │
                         apply ─► newRecord(RecordRepo.Promotion/Source) ──► registro
                                                                                │
kickoff seal ─► Snapshot ─► planWithPromotion (sin preguntar) ─► GitSnapshot{MainBranch,
                                  repos[].Promotion si Recorded, Base=Base()} (una vez)

status ─► planWithPromotion (sin preguntar) ─► describeRepository
             └─ modo branch: aheadOfMain + mergeLayers(integración) ─► PromotionStage
                  ⇒ prTarget = main_branch y promotionStage=main (o aviso si no concluye)

done ─► planWithPromotion (sin preguntar) ─► MergeDestination(repo, rama)
             └─ mergeLayers(destino) + pr_merged solo si aplica ─► ConfirmMerge ─► borra o conserva
```

## Cambios por fichero

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/workspace/gitpolicy.go` | Modificar | Campos `MainBranch` y `Promotion`, claves cerradas, `validPromotion`, `branchValidator` con `switch`, `resolveGit` sin defecto, textos de error ampliados y `PromotionWarnings` |
| `cmd/axiom/main.go` | Modificar | Añade `PromotionWarnings` a los avisos de `workspace validate` (`:606`) |
| `internal/branch/policy.go` | Modificar | `Policy` y `RepoPolicy` con `MainBranch`, `Promotion` y `PromotionSource`; herencia en `ResolvePolicy` y `For` |
| `internal/branch/promotion.go` | Crear | `Promotion` y sus valores, `PromotionSource*`, `PromotionStage*`, `promotionLabels` y `PromotionLabel`, `PromotionRequest`, `ParsePromotionRequests`, `PromotionFacts`, `ResolvePromotion`, `MergeDestination`, `PromotionStage` |
| `internal/branch/promotion_refusal.go` | Crear | `EnsureArgs.Invocation`, `PromotionDecisionRequired`, `PromotionConflict`, `UnknownPromotionRepository` e `InvalidPromotion` |
| `internal/branch/planner.go` | Modificar | `PlanInput.Promotions`; `RepoPlan.MainBranch`, `Promotion`, `PromotionSource` y `BaseBranch`; `TwoStage()` y `Base()`; `planRepository` los rellena; `repoPolicy` propaga los campos |
| `internal/branch/types.go` | Modificar | `RecordRepo.Promotion` y `PromotionSource` (`omitempty`) |
| `internal/branch/registry/registry.go` | Modificar | `validate` admite el vocabulario cerrado |
| `internal/branch/ensure_refusal.go` | Modificar | `DecisionRequired(args EnsureArgs)`; `BaseNotFound` nombra la rama principal en modo `branch` |
| `internal/branch/done_refusal.go` | Modificar | `MergeNotConfirmed(…, destination string, targets)` |
| `internal/branch/result.go` | Modificar | Campos aditivos en `RepositoryResult` y `BranchResult`, `NewRepositoryResult` los rellena y el texto añade «Promoción: …» y «siguiente PR hacia …» |
| `internal/branch/engine/prefs.go` | Modificar | `promotionRequest`, `planWithPromotion` y `promotionFacts` (junto a `resolveMode`) |
| `internal/branch/engine/ensure.go` | Modificar | `EnsureRequest.Promotions`; usa `planWithPromotion`; reaplica el mapa tras `apply`; añade los avisos |
| `internal/branch/engine/ensure_registry.go` | Modificar | `recordRepository` escribe o conserva `Promotion` y `PromotionSource` |
| `internal/branch/engine/status.go` | Modificar | `baseCandidates` usa `repo.Base()`; `missingBaseWarning`; `Status` usa `planWithPromotion`; `describeRepository` calcula la etapa |
| `internal/branch/engine/merge_layers.go` | Crear | `mergeLayers`, extraída de `done.go:441-493`, compartida con `status` |
| `internal/branch/engine/done.go` | Modificar | `planWithPromotion`; `mergeEvidence` con `MergeDestination`; `pr_merged` condicionado |
| `internal/branch/engine/snapshot.go` | Modificar | `planWithPromotion`; `MainBranch`, `Promotion` (si `Recorded`) y `Base = repo.Base()` |
| `internal/kickoff/gitsnapshot.go` | Modificar | `GitSnapshot.MainBranch` y `GitSnapshotRepo.Promotion` (`omitempty`) y su validación |
| `internal/cli/branch.go` | Modificar | Bandera `--promotion` repetible, su validación, el rechazo con `--archive` y la ayuda |
| `scripts/axiom-worktree.ps1`, `scripts/axiom-worktree.sh` | Modificar | `done` con la integración de `status --json` y caída a `main` |
| `axiom.example.yaml`, `docs/manual-de-inicio.md` | Modificar | Claves nuevas, flujos, `--promotion` y aviso de compatibilidad |
| `internal/assets/skills/sdd-apply/SKILL.md`, `internal/assets/skills/sdd-archive/SKILL.md` | Modificar | Las dos frases de D12 |
| Tests (ver la estrategia) | Crear o modificar | Una capa por unidad de trabajo |

`internal/branch/engine/publication.go`, `handoff.go`, `verdict.go`, `pending.go`, `planner_archive.go`, `livespec` y el dashboard **no** cambian (alcance de la propuesta; el dashboard lo cataloga C-06-e).

## Interfaces / contratos

```go
// internal/workspace/gitpolicy.go
type GitConfig struct {
	IntegrationBranch string `yaml:"integration_branch,omitempty"`
	BranchPrefix      string `yaml:"branch_prefix,omitempty"`
	MainBranch        string `yaml:"main_branch,omitempty"` // sin defecto: vacío = una etapa
	Promotion         string `yaml:"promotion,omitempty"`   // branch | release | ask; vacío = defecto del motor
}
type RepositoryPolicy struct {
	Path              string `yaml:"path"`
	IntegrationBranch string `yaml:"integration_branch,omitempty"`
	BranchPrefix      string `yaml:"branch_prefix,omitempty"`
	Promotion         string `yaml:"promotion,omitempty"`
}
const (PromotionBranch = "branch"; PromotionRelease = "release"; PromotionAsk = "ask")
func PromotionWarnings(cfg *WorkspaceConfig) []string

// internal/branch/policy.go
type Policy struct {
	IntegrationBranch, BranchPrefix, MainBranch string
	Promotion                                   Promotion
	PerRepository                               map[string]RepoPolicy
}
type RepoPolicy struct {
	IntegrationBranch, BranchPrefix, MainBranch string
	Promotion                                   Promotion
	PromotionSource                             string // repository | git | "" (no declarado)
}

// internal/branch/promotion.go
type Promotion string
const (
	PromotionBranch  Promotion = "branch"
	PromotionRelease Promotion = "release"
	PromotionAsk     Promotion = "ask" // política o pendiente; nunca se registra
)
const (
	PromotionSourceAnswer = "answer"; PromotionSourceFlag = "flag"
	PromotionSourceRepository = "repository"; PromotionSourceGit = "git"
	PromotionSourceDefault = "default"; PromotionSourceSnapshot = "snapshot"
	PromotionSourceLegacy = "legacy"
)
const (PromotionStageIntegration = "integration"; PromotionStageMain = "main")
func PromotionLabel(token string) string

type PromotionRequest struct {
	Repository string    // vacío: se aplica a todos los repositorios del plan
	Mode       Promotion // branch | release
}
func ParsePromotionRequests(values []string) ([]PromotionRequest, error)

type PromotionFacts struct {
	Exempt, TwoStage   bool
	Recorded           *RecordedPromotion // nil: sin entrada en el registro
	Snapshot           *SnapshotPromotion // nil: sin entrada en la instantánea
	ChangeBranchExists bool               // solo para la regla legacy de la instantánea
	Requested          Promotion          // cualificado > sin cualificar; "" si no hay
	Declared           Promotion          // RepoPolicy.Promotion
	DeclaredSource     string             // repository | git
}
type RecordedPromotion struct {
	Mode            Promotion
	Source          string
	HasChangeBranch bool
}
type SnapshotPromotion struct {
	Mode           Promotion
	SealedWithMain bool // GitSnapshot.MainBranch != ""
}
type ResolvedPromotion struct {
	Mode     Promotion // "" una etapa | branch | release | ask (pendiente)
	Source   string
	Recorded bool      // sale del registro: la instantánea lo copia
}
type PromotionResolution struct {
	Resolved     ResolvedPromotion
	Materialized Promotion // no vacío ⇒ conflicto con Requested
}
func ResolvePromotion(f PromotionFacts) PromotionResolution
func MergeDestination(repo RepoPlan, planned BranchPlan) (destination string, prMergedApplies bool)
func PromotionStage(repo RepoPlan, aheadOfMain *int, evidence MergeEvidence) (stage, target string, inconclusive bool)

// internal/branch/planner.go
type PlanInput struct { /* … */ Promotions map[string]ResolvedPromotion /* por GitRoot */ }
type RepoPlan struct {
	/* … campos actuales … */
	MainBranch      string
	Promotion       Promotion
	PromotionSource string
	BaseBranch      string
}
func (r RepoPlan) TwoStage() bool
func (r RepoPlan) Base() string

// internal/branch/promotion_refusal.go y ensure_refusal.go
type EnsureArgs struct {
	Operation, Change, CWD string
	Lane                   Lane
	Mode                   Mode
	Exceptions             Exceptions
	Repos                  []string
	Promotions             []PromotionRequest
}
func (a EnsureArgs) Invocation() string
func DecisionRequired(args EnsureArgs) *Refusal // antes: (operation, change, cwd, lane)
func PromotionDecisionRequired(args EnsureArgs, repo RepoPlan, qualify bool) *Refusal
func PromotionConflict(args EnsureArgs, repo RepoPlan, materialized, requested Promotion, source string) *Refusal
func UnknownPromotionRepository(args EnsureArgs, name string, available []string) *Refusal
func InvalidPromotion(args EnsureArgs, value string) *Refusal
func MergeNotConfirmed(change, cwd string, lane Lane, repo RepoPlan, planned BranchPlan, destination string, targets []string) *Refusal

// internal/branch/types.go
type RecordRepo struct {
	/* … */
	Promotion       Promotion `json:"promotion,omitempty"`
	PromotionSource string    `json:"promotionSource,omitempty"`
}

// internal/branch/result.go (aditivo)
// RepositoryResult: MainBranch `json:"mainBranch,omitempty"`, Promotion/PromotionLabel,
//                   PromotionSource/PromotionSourceLabel (omitempty)
// BranchResult:     PromotionStage/PromotionStageLabel (omitempty)

// internal/kickoff/gitsnapshot.go (aditivo)
// GitSnapshot.MainBranch     `yaml:"main_branch,omitempty"`
// GitSnapshotRepo.Promotion  `yaml:"promotion,omitempty"` // branch | release

// internal/branch/engine
type EnsureRequest struct { /* … */ Promotions []branch.PromotionRequest }
type promotionRequest struct { interactive bool; requested []branch.PromotionRequest; args branch.EnsureArgs }
func (e *Engine) planWithPromotion(ctx context.Context, facts *changeFacts, req promotionRequest) (branch.ChangePlan, []string /*avisos*/, *branch.Refusal)
func (e *Engine) mergeLayers(ctx context.Context, dir, branchRef string, targets []mergeReference) ([]string, branch.MergeEvidence, error)
```

Ejemplo de `status --json` en modo `branch` tras la fusión en `develop` (solo los campos nuevos o cambiados):

```json
{
  "repositories": [
    {
      "integrationBranch": "develop",
      "mainBranch": "master",
      "promotion": "branch",
      "promotionLabel": "rama a rama",
      "promotionSource": "answer",
      "promotionSourceLabel": "respuesta registrada",
      "base": { "ref": "origin/master" },
      "branches": [
        {
          "name": "feat/x",
          "kind": "change",
          "prTarget": "master",
          "promotionStage": "main",
          "promotionStageLabel": "hacia la rama principal"
        }
      ]
    }
  ]
}
```

## Estrategia de pruebas

Los tests siguen los patrones del repositorio: tablas puras en `internal/branch`, y fixtures git reales en `engine` (`newProject`, `newRepository`, `commit`, `runGit` de `status_test.go`; `writeRegistry` y `newMultirepo` de `status_registry_test.go`; `commitFile`, `squashedBranch`, `requireMergeTree` y `mergeTreeSupport` de `done_layers_test.go`; `observe` y `wantNoRemote` para garantizar que no hay red).

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Esquema | Claves nuevas válidas e inválidas; `main_branch` en `repositories[]` rechazada; `promotion: Branch` rechazada; errores acumulados que nombran las cuatro claves; `resolveGit` sin defecto; `PromotionWarnings` (sin `main`, `main` igual a la integración y entrada que iguala la integración) | Tablas en `gitpolicy_test.go`, ampliando las de `:80-120`, `:414-430` y `:487-539` |
| Política | Herencia `git` → `repositories[]` de `Promotion`; `MainBranch` solo global; repositorio fuera de la carpeta maestra | `policy_test.go` |
| Resolución pura | Cada fila de la precedencia de D4, más `legacy` por registro y por instantánea (con y sin marcador, con y sin rama), conflictos con registro e instantánea, `answer` frente a `flag`, exento y de una etapa | `promotion_test.go`, una tabla por función (`ResolvePromotion`, `ParsePromotionRequests`, `MergeDestination`, `PromotionStage`) |
| Plan | `BaseBranch` según el modo; `IntegrationBranch` y `PRTarget` intactos; repositorio de specs sin modo; ramas de rol intactas; sin `Promotions`, el plan es idéntico al actual | `planner_test.go` |
| Rechazos | `EnsureArgs.Invocation()` byte a byte (sin banderas = cadena actual); `DecisionRequired` conserva excepciones y `--promotion`; `PromotionDecisionRequired` con `qualify` | `ensure_decision_test.go`, `promotion_refusal_test.go` |
| Registro | Ida y vuelta con los campos; bytes idénticos sin ellos; vocabulario inválido → error | `registry/registry_test.go` |
| Instantánea | `Validate` de `main_branch` y `promotion`; aditiva y legible sin los campos | `kickoff/gitsnapshot_test.go` |
| Motor: `ensure` | `main_branch: master`, `integration_branch: develop`: con `branch` la rama sale de `origin/master`; con `release`, de `develop`; con `ask`, `decision_required` sin crear nada ni registro, y al repetirlo con la invocación la rama sale de la base correcta y el registro guarda `answer`; con `--promotion` distinto del registrado, `plan_conflict`; sin `main_branch`, el registro es idéntico byte a byte; multirepo con dos repositorios de código en `ask`, dos llamadas secuenciales con `Details.name` y la segunda invocación conserva la primera respuesta; repositorio de specs exento | `engine/ensure_promotion_test.go` (nuevo) |
| Motor: `status` | `prTarget: develop` antes de la fusión y `master` con `promotionStage: main` después (merge commit y squash en `develop`); rama recién creada desde `master` con `master` ascendiente de `develop` → `integration`; sin capa 2 → `integration` con aviso; `ask` pendiente → aviso; sin red | `engine/status_promotion_test.go` (nuevo) |
| Motor: `done` | Dos etapas con merge commit: fusionada solo en `develop` → `merge_not_confirmed`, con `master` como destino; fusionada en `master` → eliminada por `ancestry`. Squash real en las dos etapas → `equivalent_tree`. `pr_merged` verificada en `gates.yaml` sin ascendencia en `master` → conservada. `--force` → `forced` con punta. `release` → eliminada al fusionarse en `develop`. Rama de archivado → integración | `engine/done_promotion_test.go` (nuevo), con `recordIntegration` y `squashedBranch` |
| Motor: `snapshot` | Monorepo SDD: `ensure` con `branch` y después el sellado → `promotion: branch`, `main_branch: master` y `base: master`; un `ensure` posterior no reescribe `kickoff.yaml`; con el registro borrado se resuelve desde la instantánea (`snapshot`) | `engine/snapshot_test.go` |
| CLI | `--promotion` vacío, `ask`, `x=`, `=branch`, sin cualificar duplicado, repositorio duplicado y con `--archive` → salida 2; la ayuda lo documenta | `cli/branch_promotion_test.go` (nuevo, con `runBranchCLI`) |
| Scripts | `done` con `integrationBranch: develop` y el checkout en `develop` → `fetch origin develop` y `ff-only origin/develop`; sin el campo → `main`; `pr` con `prTarget: master` → `--base master` | `scripts_axiom_worktree_test.go` (`stubRepository` gana `IntegrationBranch`) |
| Docs | Las líneas fijadas siguen ahí; aparecen `main_branch`, `promotion` y el aviso de compatibilidad | `docs_contract_test.go` (añadir casos y no tocar los existentes) |

Regresión: `go test ./...` sin `main_branch` debe pasar sin cambiar ningún test existente, salvo los que enumeran claves (`gitpolicy_test.go`) y la firma de `DecisionRequired` y `MergeNotConfirmed` en sus llamadores de test, cuyos valores esperados no cambian.

## Matriz de amenazas

El cambio toca la automatización de git (la base de creación de ramas y el destino que autoriza borrarlas) y dos scripts de shell que componen órdenes de git y de PR.

| Frontera | Casos adversos mínimos | Aplicabilidad | Respuesta de diseño | Pruebas RED previstas |
|---|---|---|---|---|
| Rutas tipo documentación | `requirements.txt`, `README.sh`, Markdown ejecutable | N/A: no se clasifica ni ejecuta ningún fichero | — | — |
| Selección de repositorio git | `git -C`, `cwd` relativo o absoluto, worktree enlazado | Aplicable: `status` y `done` leen ascendencia, árbol equivalente y adelanto en repositorios del plan; los scripts usan `git -C` | Las lecturas nuevas usan `repo.GitRoot` o el checkout principal (`listed[0].Path`), como hoy; los scripts, `$mainRoot` y `$MAIN_ROOT`; nunca el `cwd` | `status` desde un worktree enlazado con modo `branch` (patrón de `TestStatusSeesTheRegistryFromALinkedWorktree`); `done` desde el worktree del cambio |
| Estado del commit | staged, `commit -a`, índice vacío | N/A: no se crea ningún commit; las comprobaciones de árbol limpio de `done` y de `in_place` no cambian | — | — |
| Estado del push | rama de seguimiento, primer push, refspec explícito | N/A: el push de `pr` sigue siendo `push -u origin <rama del cambio>`; solo cambia la base de la PR | — | — |
| Comandos de PR | `--base` y `--head` explícitos, composición de órdenes | Aplicable: `pr` compone `gh pr create --base $PR_TARGET` y `prTarget` puede ser ahora `main_branch` | `prTarget` solo toma nombres validados (`ValidRefName` al cargar `axiom.yaml`) y los scripts los citan entre comillas | Test del script con `prTarget: master` → `--base master --head inc/demo` |
| Base de creación (específica) | `main_branch` inexistente, `--base` junto con el modo, `origin/<main>` sin traer | Aplicable | `baseCandidates(repo.Base())`: primero `origin/<main>`, después `<main>`; si no hay ninguna, `base_not_found` nombra `git.main_branch`; `--base` prevalece; sin red sin `--fetch` | `ensure` con `branch` y sin `master` → `base_not_found` que nombra `main_branch`; con `--base` → la excepción manda |
| Autoridad de borrado (específica) | `pr_merged` de la primera PR, squash en las dos etapas, `ask` pendiente sin registro | Aplicable | `MergeDestination` (D7): destino `main_branch` sin capa 3; `ask` pendiente → `main_branch`; `--force` deja la punta | `done` con `pr_merged` verificada y sin fusión en `master` → `merge_not_confirmed`; squash en las dos etapas → `equivalent_tree`; registro borrado con `ask` → conservada |

## Unidades de trabajo y estimación

Entrega `single-pr` con la etiqueta `size:exception`. Cada unidad tiene su propio commit (Conventional Commits) y deja `go test ./...` en verde. Las 400 líneas son solo orientativas.

| # | Unidad (commit) | Contenido | Líneas de autoría (código + tests) |
|---|---|---|---|
| 1 | `feat(workspace): claves main_branch y promotion en git y repositories` | D1, D2 y el aviso de `validate` | ~120 + ~180 |
| 2 | `feat(branch): dominio puro de la promoción de ramas` | D3, D4 (función pura), D6 (plan), D7 (`MergeDestination`), D8 (`PromotionStage`), registro, rechazos y resultado | ~380 + ~420 |
| 3 | `feat(branch): ensure resuelve y registra el modo de promoción` | D5, D10, `recordRepository` y CLI | ~220 + ~380 |
| 4 | `feat(branch): status propone la siguiente PR y done confirma contra el destino` | D7 y D8 en el motor, `mergeLayers`, instantánea (D9) y squash en las dos etapas | ~200 + ~450 |
| 5 | `docs(branch): scripts, ejemplo, manual y doctrina de la promoción` | D11 y D12, con sus tests | ~90 + ~90 |

Total aproximado: **~1000 líneas de código y documentación y ~1500 de tests (~2500)**, sin contar el delta de spec. Supera la estimación de la propuesta (700-1000) por el peso de los fixtures git de dos etapas. Las unidades 1 y 2 son independientes del motor; la 3 depende de la 2, y la 4 de la 3.

## Migración / despliegue

- Sin migración de datos: los campos nuevos de `axiom.yaml`, del registro, de la instantánea y del JSON son opcionales y aditivos.
- Un registro o un `kickoff.yaml` escritos antes, con una rama ya creada, se tratan como de una etapa (`legacy`). Un repositorio cuya rama aún no existe toma el modo de la política vigente.
- El único punto de ruptura es un binario anterior frente a un `axiom.yaml` con `git.main_branch` o `git.promotion`: produce `config_invalid`. Se documenta en el manual (D12) y se acepta (decisión 8 de la propuesta).
- Rollback: `git revert` de la PR. Antes, o a la vez, hay que retirar las claves nuevas de `axiom.yaml`. Los registros e instantáneas con los campos siguen siendo legibles (`registry.Load` ignora campos desconocidos y la lectura del kickoff no es estricta).

## Preguntas abiertas

- [ ] La spec (en paralelo) debe recoger en REQ-25.36 el campo `main_branch` de la instantánea, que este diseño añade como marcador de D4, además de `repositories[].promotion` y la base efectiva. No bloquea: si la spec no lo incluye, se ajusta en `tasks`.
- [ ] La spec debe fijar dos comportamientos que este diseño asume: el destino conservador de `done` con `ask` pendiente y sin registro (`main_branch`), y la respuesta a un `--promotion <repo>=…` que no nombra ningún repositorio del plan (`invalid_name`).
