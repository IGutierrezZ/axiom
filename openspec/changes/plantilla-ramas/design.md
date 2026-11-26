# Diseño: plantillas de nombre de rama y ramas diferidas a la entrada del rol (C-07-f)

## Enfoque técnico

El cambio sigue la recomendación A1 + B1 + C1 + D1 de la propuesta, con las resoluciones del orquestador (R1, R2, R3, precedencia y diferido) y las decisiones de la ronda 1 de cobertura (S1 a S12 y G1 a G11). Lo encaja en la arquitectura actual de `axiom branch`: un planificador puro (`internal/branch`) que recibe hechos ya reunidos por el motor (`internal/branch/engine`), sin E/S propia.

Las piezas son seis:

1. **Motor de plantillas puro en un paquete hoja**, `internal/branch/nametemplate`. Analiza, lista variables, exige una variable distintiva y renderiza. Lo usan tres consumidores que no se pueden importar entre sí: la carga de `axiom.yaml` (`internal/workspace`), la validación de la instantánea (`internal/kickoff`) y el planificador (`internal/branch`). Por eso no puede vivir en `internal/branch`, que importa `workspace` y `kickoff` (decisión D-01).
2. **Esquema**: `branch_template` en `git:`, en `repositories[]` y en `roles.<id>`, validado al cargar (prov:TOP:1), con un canal de error propio para las reglas que dependen de la topología y de los roles (decisión D-02).
3. **Planificador**: plantilla efectiva por rama (rol > repositorio > `git:`, con la atribución de prov:BRA:3), precedencia de nombres de prov:BRA:3, ramas **pendientes** fuera de `RepoPlan.Branches` (decisión D-07), colisiones (D-05) y rechazos tipados (prov:BRA:4).
4. **Rama del cambio resuelta en un único punto**: el helper puro `branch.ResolveChangeBranch` sustituye todas las composiciones `prefijo + "/" + cambio` y `+ "-archive"` del planificador, del archivado, de la guarda y del autofetch (decisión D-14).
5. **Motor**: resuelve los valores de las variables solo cuando alguna plantilla los necesita (prov:BRA:2), entrega al planificador los nombres congelados en el registro y el rol que entra, congela en el registro el nombre de las ramas con plantilla (RF-BRA-15) y recupera las ramas pendientes que ya existen en git (D-09). `status` proyecta las pendientes sin mutar nada (D-15).
6. **Instantánea y guarda**: el seal congela la plantilla de cada rama, el prefijo de las ramas pendientes y los valores `kind` y `ref` (prov:KOF:1, R1), y propaga cualquier rechazo de render sin escribir `kickoff.yaml`. La guarda (`verdict.go`) obtiene el nombre esperado igual que el plan, también sin registro, y distingue una rama pendiente de una rama equivocada (D-10).

**Invariante central (sin plantilla, nada cambia):** si ninguna plantilla interviene en el plan, el motor no lee `integrations.yaml` ni el staging, el planificador y `ResolveChangeBranch` siguen la composición actual, `RecordBranch.Template`, `GitSnapshotBranch.Template`, `pending_branches`, `template_values`, `pendingBranches` y `Verdict.Details` quedan vacíos y se omiten (`omitempty`), y los nombres, la instantánea, el registro y el JSON son idénticos byte a byte (criterio de éxito 1 y prov:BRA:3). La única diferencia observable sin plantilla es el caso límite de D-05 (un `--branch` nuevo tras el seal, antes de crear ninguna rama de código de un multirepo), que hoy se registra y se ignora.

## Decisiones de arquitectura

### D-01: el motor de plantillas vive en el paquete hoja `internal/branch/nametemplate`

**Elección**: paquete nuevo `internal/branch/nametemplate` que solo importa `internal/gitref` y la biblioteca estándar. Expone `Parse`, `Template.Uses`, `Template.Render`, `CheckDistinguishing`, `CheckSample` y `Slug`, con errores tipados.
**Alternativas consideradas**:
- `internal/branch/template.go`, como decía la propuesta: imposible. `workspace` necesita validar al cargar, y `branch` ya importa `workspace` y `kickoff`, así que se formaría un ciclo.
- Ponerlo en `internal/gitref`: mezcla la sintaxis de una plantilla con las reglas de nombres de git y engorda un paquete que hoy es solo de validación.
- Duplicar el analizador en `workspace` y en `branch`: dos gramáticas que pueden divergir.
**Razonamiento**: un único analizador para la carga, la instantánea y el plan. El paquete sigue bajo `internal/branch/**`, así que la prueba de arquitectura de `internal/gitcommon` también lo cubre, y es puro: no ejecuta procesos ni lee ficheros.

### D-02: validación al cargar en dos canales

**Elección**:
- La sintaxis, la variable distintiva y la muestra de `git.branch_template` y `repositories[].branch_template` se validan en `branchValidator`, dentro de `decodeStringKeys`, y sus errores llegan por los canales existentes `GitErr` y `RepositoriesErr`.
- Las reglas que necesitan la topología o los roles van a un canal nuevo, `Declarations.BranchTemplatesErr`, que `parseLayered` calcula después de validar la base:
  - la sintaxis, la variable distintiva y la muestra de `roles.<id>.branch_template`;
  - la regla de la rama con `openspec/` (S11, G6): su plantilla efectiva es la de la entrada de `repositories:` del repositorio que la aloja **si existe y declara `branch_template`**; si no, la de `git:`. Esa plantilla no puede usar `{role}` ni `{task_ref}` (`config_invalid`, que nombra la clave).
- La entrada que aloja la rama con `openspec/` se identifica solo con la configuración, sin git:
  - en `monorepo-embedded`, la entrada cuya `path` normalizada (la misma normalización que `branch.repoKey`) es `.`;
  - en `multirepo`, la entrada cuya `path` normalizada coincide con el desplazamiento léxico del repositorio de specs (`lexicalDisplacement(cfg)`, que ya resuelve `specs_repository` y el caso `…/openspec`). Con un `specs_repository` absoluto no hay desplazamiento léxico: se toma `git:`, que es la opción conservadora.
- En `multirepo` la regla se aplica aunque el repositorio de specs trabaje in situ (RF-BRA-05), por prudencia. Consecuencia: `git:` con `{task_ref}` solo es válido en `multirepo` si el repositorio de specs declara su propia plantilla.
- `joinLoadErrors` y `branch.ConfigRefusal` incluyen el canal nuevo.
- La advertencia «sin efecto» de `roles.<id>.branch_template` en `monorepo-embedded` sale de `workspace.BranchTemplateWarnings`, que se llama en `cmd/axiom/main.go` junto a `PromotionWarnings`.
**Alternativas consideradas**:
- Validar los roles en `validateBaseConfig`: una plantilla de rol rota invalidaría la configuración base entera y bloquearía órdenes ajenas a las ramas (`sdd status`, etc.).
- Pasar la topología a `decodeGit`: acopla el decodificador de un bloque a otro bloque y no resuelve los roles, que no pasan por `decodeStringKeys`.
- Tomar la entrada del repositorio de specs aunque no declare `branch_template`: una entrada que solo fija `branch_prefix` dejaría pasar un `git:` con `{task_ref}`.
**Razonamiento**: sigue el patrón de canales por declaración (D-03 de la topología). Los errores se acumulan y cada uno nombra su clave, y `config_invalid` llega a `ensure` por el mismo camino que `GitErr`. `workspace validate` detecta así, de forma estática, todo lo que no depende del plan (S5).

### D-03: vocabulario y reglas de valor

**Elección**: las variables son exactamente `prefix`, `kind`, `change`, `slug`, `role`, `ref` y `task_ref`. Una llave sin cerrar, una llave de cierre suelta, unas llaves anidadas o una variable desconocida son un error de sintaxis. **No hay escape** para llaves literales.

**Variable distintiva (S6)**: toda plantilla DEBE usar al menos una de `{change}`, `{slug}`, `{ref}` o `{task_ref}`. Si no, `CheckDistinguishing` devuelve `*ConstantTemplateError` y la carga da `config_invalid`. Así ninguna plantilla produce un nombre constante compartido por todos los cambios.

Reglas de cada valor sustituido:

| Variable | Regla |
|---|---|
| `{prefix}` | `gitref.ValidPrefix` |
| `{kind}` | `feature` o `fix` |
| Las demás | `gitref.ValidBranchComponent` |

El nombre resultante debe cumplir `gitref.ValidRefName`. `CheckSample` renderiza con valores de muestra seguros (`feat`, `feature`, `sample-change`, `sample`, `role`, `1`, `1`) para validar la forma al cargar.

**`Slug(change, ref)` (S1)**: solo actúa con `ref` no vacío. Quita del principio de `change` el prefijo `^[A-Za-z]+-<ref>-`, comparando sin distinguir mayúsculas (tanto las letras como `ref`). Si el prefijo no casa, o si al quitarlo queda vacío, devuelve `change`: `{slug}` nunca está vacío. Ejemplos: `us-2525-login` con `ref` `2525` da `login`; `US-2525-login` también; `us-2525` y `us-2525-` dan el propio nombre del cambio; `login` da `login`.
**Alternativas consideradas**:
- Escape `{{`: las llaves no aparecen en ninguna convención conocida, y git rechaza `@{`.
- Sanear los valores (sustituir los caracteres no válidos): lo prohíbe D4 de la propuesta.
- Un primer segmento cualquiera sin guion como prefijo (versión anterior): aceptaba dígitos y no fijaba qué pasaba con el resto vacío.
**Razonamiento**: es la gramática mínima, y sus errores nombran la variable y el valor sin ambigüedad.

### D-04: atribución de rama a rol y plantilla efectiva

**Elección**: una función pura del planificador, `effectiveTemplate`, decide la plantilla de cada rama:

| Rama | Rol atribuido | Plantilla efectiva |
|---|---|---|
| La que lleva `openspec/`: la del cambio en el repositorio del planning home de un `monorepo-embedded`, o la de la feature ODD en ese repositorio | ninguno | repositorio (si declara plantilla) > `git:` |
| La del cambio en un repositorio de código tocado por **un** rol, incluido `fullstack` | ese rol | rol > repositorio > `git:` |
| La de seguimiento de un repositorio tocado por **dos o más** roles | ninguno | repositorio > `git:` |
| La de rol de un repositorio compartido | ese rol | **solo** `roles.<id>.branch_template`; sin ella, `<seguimiento>-<rol>` |
| La de un repositorio incluido con `--repo` en ODD | ninguno | repositorio > `git:` |
| La de archivado | — | ninguna: `<rama del cambio resuelta>-archive` (D-14) |

`fullstack` es reservado y no tiene entrada en `roles:`, así que nunca tiene plantilla de rol; pero es un rol del kickoff, se le atribuye la rama del cambio de cada repositorio de código que toca, `ensure --role fullstack` es válido y su Task se vincula con `axiom integrations tasks create|bind` de C-07-e como la de cualquier otro rol (G7).
**Alternativas consideradas**: aplicar rol > repositorio > `git:` también a las ramas de rol de un repositorio compartido. Con una plantilla de repositorio o de `git:` sin `{role}` ni `{task_ref}`, todas las ramas de rol renderizarían el mismo nombre que la de seguimiento. Y si la plantilla las llevara, la rama de seguimiento ya estaría rechazada por R2.
**Razonamiento**: coincide con prov:BRA:3 y conserva `<seguimiento>-<rol>` como hoy.

### D-05: precedencia de nombres por rama y colisiones

**Elección**: en `planRepository`, por rama y de mayor a menor (prov:BRA:3, G8):

1. `--branch`, **solo** para la rama del cambio (la de seguimiento en un repositorio compartido). Vale igual con `--role` (S4).
2. El nombre de la instantánea (`GitSnapshotBranch.Name`).
3. El nombre congelado en el registro: una `RecordBranch` con `Template` no vacío, que llega en `RepoFacts.Frozen`.
4. La plantilla efectiva:
   - en un cambio **sellado**, solo la que congela su instantánea (la entrada pendiente o el campo `template`);
   - en un cambio sin sellar o en ODD, la de la política de `axiom.yaml`.
5. El nombre por defecto: `<prefijo>/<cambio>` o `<seguimiento>-<rol>`.

Reglas añadidas por S4:
- Si la rama del cambio estaba pendiente por `{task_ref}` y la invocación trae `--branch`, deja de estar pendiente: se nombra con `--branch` y se crea en ese mismo `ensure`. `--branch` nunca nombra una rama de rol.
- `--prefix` solo alimenta `{prefix}` y los nombres por defecto. Si la plantilla efectiva no usa `{prefix}`, `--prefix` no tiene efecto ni produce aviso.
- Las ramas de rol sin plantilla de rol siguen siendo `<rama de seguimiento>-<rol>`, con la rama de seguimiento ya resuelta.

Un cambio sellado cuya instantánea no lleva plantillas (sellado antes de esta capacidad o sin plantilla) **nunca** aplica la plantilla de la política: D5 y prov:KOF:1.

**Colisiones (S6, G11)**: `checkTemplateCollisions(repo RepoPlan, effective RepoPolicy)` corre en el planificador al final de `planRepository`, **solo** si alguna rama del repositorio tiene `Template` no vacío o procede de `Frozen` (sin plantilla no se evalúa nada nuevo). Compara todos los nombres resueltos del repositorio en ese plan, incluidos los congelados del registro y el de la rama de un rol que entra. Rechaza con `invalid_name`, sin crear nada:
- dos ramas del mismo repositorio con el mismo nombre (nombra ambas ramas y el repositorio);
- un nombre renderizado igual a la rama de integración del repositorio o a `main_branch` (`Details.field` = `branch_template`, `conflict` = `integration_branch` | `main_branch`).

Las ramas pendientes sin nombre no participan; dos roles con el mismo `{task_ref}` chocan al entrar el segundo, porque el primero ya está en `Frozen`. La colisión con la rama de otro cambio activo queda fuera: limitación declarada (S6).
**Alternativas consideradas**:
- Conservar el orden actual («instantánea > `--branch`»). Es incompatible con prov:BRA:3 y con S4. Además, en el único caso en que difieren (un multirepo con un `--branch` nuevo tras el seal y antes de que exista ninguna rama de código), hoy la excepción se registra pero se ignora, lo que es incoherente.
- Leer del registro todos los nombres, no solo los congelados por plantilla: cambiaría el comportamiento sin plantilla (un prefijo cambiado después de crear una rama en ODD).
- Comparar también los `Candidate` de las pendientes: rechazaría un `ensure` sin `--role` por roles que aún no han entrado.
**Razonamiento**: en todos los flujos alcanzables, `--branch` coincide con el nombre de la instantánea porque esta lo absorbió al sellar y `mergeExceptions` rechaza las contradicciones con ramas materializadas, así que el orden nuevo solo cambia ese caso límite. La guarda adopta el mismo orden (D-10) para que el plan y el veredicto nunca discrepen.

### D-06: congelado en la instantánea (R1)

**Elección**: hay tres campos aditivos en `kickoff.GitSnapshot`, todos `omitempty` y solo cuando alguna plantilla interviene en el plan:
- `GitSnapshotBranch.Template`: la plantilla cruda con la que se nombró la rama. Es informativa y auditable; el nombre sigue mandando.
- `GitSnapshotRepo.PendingBranches []GitSnapshotPendingBranch{Kind, Role, Template, Prefix}`: las ramas cuya plantilla usa `{task_ref}`. **No llevan `name`**. `Prefix` congela el `{prefix}` efectivo de ese repositorio al sellar, porque `repositories[].branch_prefix` no se congela en ningún otro sitio.
- `GitSnapshot.TemplateValues *GitTemplateValues{Kind, Ref}` (`template_values`), con estas reglas (G10):
  - `kind` se congela **siempre** que interviene una plantilla: `fix` si el staging de `change start` declara el perfil `bug`, y `feature` en cualquier otro caso. El staging sigue presente porque `--adopt` va después del seal.
  - `ref` solo se congela si se conoce al sellar (vínculo o staging). Un `ref` desconocido no se escribe.

`Validate()`:
- `Template` y `PendingBranches[].Template` se analizan con `nametemplate.Parse` y `CheckDistinguishing`;
- toda entrada pendiente lleva `role` y usa `{task_ref}`;
- `kind` de la entrada pendiente ∈ {`change`, `role`};
- `TemplateValues.Kind` ∈ {`feature`, `fix`};
- `TemplateValues.Ref` cumple `ValidBranchComponent`;
- `Prefix` cumple `ValidPrefix`.

**El seal planifica y propaga (S5, G10)**: `Engine.Snapshot` ejecuta el mismo planificador con `sealing=true` y `EnteringRole=""`. Cualquier rechazo (`branch_template_not_applicable` con `shared_branch`, `openspec_branch` u `odd_lane`; `branch_template_unresolved`; `integrations_links_invalid`; `invalid_name`) vuelve como `*branch.RefusalError` y `applySealBranchPolicy` lo devuelve antes de `kickoff.Seal`, así que **no se escribe `kickoff.yaml`**. Es el camino actual (`cli/sdd_kickoff.go:107`); el diseño solo añade las pruebas.
**Alternativas consideradas**:
- Escribir la entrada pendiente dentro de `branches` con `name` vacío. **Rechazada**: el `Validate()` de un binario anterior exige `ValidRefName(name)` y fallaría al cargar el `kickoff.yaml` entero (`kickoff/schema.go:131`). Una lista aparte la ignora sin error (la lectura no es estricta).
- Prerenderizar en la instantánea todo salvo `{task_ref}` (por ejemplo `feature/2525-TASK-{task_ref}`). Es correcto, pero guarda algo que no es la plantilla declarada y R1 pide congelar la plantilla y los valores.
- Congelar `{kind}` y `{ref}` solo en el registro: es local y no garantiza el mismo nombre en otro clon.
- Congelar `kind` solo si el staging existe: cada clon lo resolvería por su cuenta (hueco G10).
**Razonamiento**: todas las entradas de un nombre diferido quedan versionadas:
- la plantilla, el prefijo, `kind` y `ref` en `kickoff.yaml`;
- `{task_ref}` en `integrations.yaml`;
- `{change}` y `{slug}` derivados del nombre y de `ref`;
- `{role}` del rol.

Por eso otro clon con `ensure --role`, sin registro y con la rama ya publicada, obtiene el mismo nombre y adopta la rama publicada (RF-BRA-27, S2).

### D-07: ramas pendientes fuera de `RepoPlan.Branches`

**Elección**: una rama cuya plantilla efectiva usa `{task_ref}` se planifica como `PendingBranch` en `RepoPlan.Pending` cuando se cumplen las tres condiciones siguientes:
- no tiene nombre congelado (ni en la instantánea ni en el registro);
- no es la rama del cambio con `--branch` en la invocación (S4);
- su rol atribuido no es `PlanInput.EnteringRole`.

Una rama pendiente no se crea ni se nombra, no se registra, no se mide (`checkPathLength`), no se coloca (`checkPlacement`) y no materializa worktree.

Con `EnteringRole` igual a su rol, se renderiza con su `{task_ref}` y entra en `Branches`. Un repositorio con todas sus ramas pendientes queda con `Branches` vacío, y `checkPlan` ya lo salta (`len(repo.Branches) == 0` y papel `code`).

**Base, destino de PR y promoción de una rama diferida (S7)**: se resuelven en el `ensure --role` que la crea, con la misma regla que si se creara al arrancar. `applyPromotion` sigue siendo por repositorio, pero `planWithPromotion` (`engine/prefs.go`) no pregunta (`ask`, RF-BRA-28/29) por un repositorio sin ramas planificadas: la pregunta llega en su `ensure --role`. Un repositorio con una rama ya creada conserva la promoción ya resuelta.
**Alternativas consideradas**: `BranchPlan` con `Name` vacío y una marca `Pending`. Obliga a proteger cada consumidor de `Branches`: `checkPlan`, `apply`, `describe`, `newRecord`, `handoff`, `scope`, `done`, la colocación y la longitud.
**Razonamiento**: los consumidores existentes no cambian, y el diferido es aditivo en el tipo y en el JSON.

### D-08: el rol llega al motor solo para el diferido

**Elección**: `engine.EnsureRequest.Role` (canónico, en minúsculas) se copia a `PlanInput.EnteringRole`. El motor no valida el rol: lo hace ya el adaptador `validateEnsureRole` (archive, odd, kickoff, roster y barrera `task_not_bound`, en ese orden), antes de llamar al motor. `fullstack` pasa el roster porque es un rol sellado. `Snapshot`, `Status` y el servicio del panel pasan `""`.
**Alternativas consideradas**: crear la rama en cuanto existe la Task, sin mirar el rol (opción D2 de la exploración). Rompe «cuando el rol entra», porque en `continuous` `tasks` y `apply` se encadenan.
**Razonamiento**: con un solo `ensure` idempotente, RF-BRA-33 solo cambia en las ramas pendientes de ese rol: sin `{task_ref}`, el plan con `--role` y sin él es el mismo.

### D-09: congelado en el registro y recuperación

**Elección**: `RecordBranch.Template` (`json:"template,omitempty"`) guarda la plantilla con la que se renderizó la rama. Solo las entradas con `Template` no vacío son «nombres congelados». El motor las entrega al planificador en `RepoFacts.Frozen`, emparejando cada repositorio por identidad del git common dir, como ya hace `recordedBranches`. **El registro es la fuente de verdad del nombre congelado** (S3).

**Recuperación con el registro borrado (S3, G5)**:
- Las ramas sin `{task_ref}` se renderizan con las entradas versionadas actuales (instantánea y vínculos). Si la plantilla o la Task cambiaron después de crear la rama, el nombre puede no coincidir con el de la rama existente: es una limitación declarada, con dos salidas, `--branch` o restaurar el vínculo.
- Para las ramas pendientes, el planificador anota en cada `PendingBranch` su `Candidate`: el nombre renderizado si el `{task_ref}` de su rol ya existe. Tras `branch.Plan`, `recoverPending` comprueba con `BranchExists` (referencias locales, sin red) si cada candidato existe:
  - un error de `BranchExists` distinto de «no existe» se propaga como error de la orden, nunca como ausencia;
  - si dos candidatos de roles distintos del mismo repositorio tienen el mismo nombre, se rechaza con `invalid_name` (los dos roles y el repositorio), sin crear nada;
  - si alguno existe, lo añade a `RepoFacts.Frozen` y planifica **una sola vez** más.
- Una Task desvinculada deja la rama sin `Candidate`: sigue pendiente y se lista con su continuación.
- Con `integrations.yaml` ilegible, no hay `Candidate` y rige D-11 (fallo cerrado solo si una plantilla lo necesita).
**Alternativas consideradas**:
- Explorar `refs/heads` buscando nombres que encajen con la plantilla: es ambiguo con varias Tasks.
- Leer también `refs/remotes/origin/<candidato>`: adoptaría una rama publicada de un rol que no entra, lo que contradice prov:BRA:5. La adopción remota sigue siendo cosa del `ensure --role` de ese rol.
- Tratar un error de `BranchExists` como ausencia: podría duplicar una rama existente con otro nombre.
**Razonamiento**: es determinista, usa solo las entradas versionadas y nunca crea ni renombra nada (RF-BRA-15).

### D-10: la guarda usa el mismo orden que el plan, también sin registro

**Elección**: `expectedNames` (`engine/verdict.go`) resuelve la rama del cambio de cada repositorio con `branch.ResolveChangeBranch` (D-14) y estas fuentes, en este orden (G1, G8):
1. `record.Exceptions.Branch`;
2. el nombre de la instantánea;
3. el nombre registrado para el repositorio;
4. si la instantánea tiene una entrada pendiente de tipo `change` para el repositorio: el render de esa plantilla con el `{task_ref}` del rol si ya se conoce (`Candidate`); si no, ningún nombre, y el rol pasa a `ExpectedNames.PendingRoles`;
5. sin instantánea, la plantilla efectiva de la política renderizada con los mismos valores que el plan;
6. el prefijo (`record.Exceptions.Prefix` o el de la política) y el cambio, como hoy.

Los valores se obtienen con el mismo resolutor que el plan (`resolveTemplateValues`, D-13), que solo lee ficheros (RF-BRA-22). Con la plantilla de la instantánea y los vínculos versionados, una rama `feature/TASK-1235` correcta sale `correct` aunque no haya registro.

Las ramas de rol esperadas:
- con instantánea, los nombres de sus entradas `role`, más los registrados, más el render de la plantilla de cada entrada pendiente `role` con el `{task_ref}` de ese rol cuando se conoce; si no se conoce, el rol pasa a `PendingRoles`;
- sin instantánea, los registrados, más el render de la plantilla de rol, o `<seguimiento>-<rol>`.

**Repositorio que espera a un rol (G2)**: `branch.Decide` añade un caso después de las exenciones y de la regla del repositorio de código: si `Expected.Change` está vacío, no hay ramas de rol esperadas y `Expected.PendingRoles` no está vacío, el veredicto es `not_applicable` con:
- `Details["reason"]` = `awaiting_role` y `Details["roles"]` = los roles pendientes, separados por comas;
- `Continuation` = `axiom branch ensure --cwd <ruta> --change <c> --role <rol>`, con el rol literal si es uno solo y `<rol>` si son varios.

Si un rol ya entró, el veredicto se calcula sobre las ramas creadas y los roles pendientes se informan aparte en `Details["pendingRoles"]`, sin cambiar el tipo del veredicto. Una rama inexistente sin plantilla pendiente sigue saliendo `wrong_branch`, como hoy.

Si la plantilla no se puede resolver (pasos 4 y 5), `VerdictFor` devuelve un `*branch.RefusalError` con `branch_template_unresolved` o `integrations_links_invalid`.

**Archivado**: la rama de archivado esperada sale de D-14: la registrada; con plantilla, `record.ArchivePrefix` se ignora; si no, la de la instantánea; si no, `<rama del cambio resuelta>-archive`.
**Alternativas consideradas**:
- Devolver `not_applicable` con aviso ante una plantilla irresoluble. **Rechazada**: `applySealBranchPolicy` sella **sin instantánea** cuando la guarda avisa, y el cambio quedaría exento de las guardas para siempre.
- Reutilizar `wrong_branch` para el repositorio pendiente: bloquearía `apply` en un repositorio que nadie puede tener aún en su rama.
**Razonamiento**: elimina las composiciones `prefijo + "/" + cambio` de la guarda. Ante lo que no puede calcular, la guarda falla cerrada, igual que con un `kickoff.yaml` ilegible; ante lo que todavía no existe, lo dice con su continuación.

### D-11: códigos de rechazo

**Elección**: dos códigos nuevos en `internal/branch/types.go`, con etiqueta en `labels.go` y constructores en `internal/branch/template_refusal.go`, todos con `Continuation` no vacía:

| Código | Etiqueta | Cuándo | Continuación (resumen) |
|---|---|---|---|
| `branch_template_unresolved` | «variable de plantilla sin valor» | `{ref}` sin vínculo, staging ni instantánea (también en ODD sin work item, S8); `{task_ref}` del rol que entra sin Task | `{ref}`: inicia el cambio desde su work item (`axiom change start --from <id\|url> --cwd <dir>`), o retira `{ref}` de `<clave>`, o fija el nombre con `--branch`. `{task_ref}`: `axiom integrations tasks bind --change <c> --role <r>=<id> --cwd <dir>` o `tasks create`, y repite `axiom branch ensure --change <c> --role <r> --cwd <dir>` |
| `branch_template_not_applicable` | «plantilla no aplicable» | `{role}`/`{task_ref}` en la rama de seguimiento de un repositorio compartido (R2), en la rama con `openspec/` (red de seguridad de la carga) o en el carril ODD (R3); `Details.reason` = `shared_branch` \| `openspec_branch` \| `odd_lane`. Igual en `ensure` (SDD y ODD) y en `kickoff seal` (S5) | quita `{role}` y `{task_ref}` de `<clave>` (las ramas de rol usan `roles.<id>.branch_template`) y comprueba con `axiom workspace validate --cwd <master>`; en ODD, además, fija el nombre con `axiom branch ensure --lane odd --change <c> --branch <rama> --cwd <dir>` |

Se reutilizan cuatro códigos existentes:
- `invalid_name` para un nombre renderizado inválido o un valor no seguro (`Details.field` = `branch_template` o el nombre de la variable, más `key` y `value`), para la colisión de dos ramas del mismo repositorio, para un nombre igual a la rama de integración o a `main_branch` (D-05) y para dos candidatos de roles distintos con el mismo nombre (D-09);
- `config_invalid` para la plantilla inválida al cargar, incluida la que no tiene variable distintiva;
- `integrations_links_invalid` (G4): con un plugin activo e `integrations.yaml` ilegible, se rechaza si alguna plantilla que hay que renderizar usa `{ref}`, `{slug}`, `{kind}` o `{task_ref}` y ese valor no está congelado en la instantánea (`template_values.ref` cubre `{ref}` y `{slug}`; `template_values.kind` cubre `{kind}`; `{task_ref}` nunca se congela en la instantánea). Una rama ya nombrada (instantánea, registro o `--branch`) no se renderiza y no falla. Constructor nuevo `IntegrationsLinksUnreadable(operation, change, master, cause)`, sin rol.

El token del motivo de la rama pendiente, `awaiting_role` («pendiente de la entrada del rol»), también se da de alta en `labels.go`.
**Alternativas consideradas**:
- Un código por motivo estructural (`shared`, `openspec`, `odd`): triplica las etiquetas sin aportar una continuación distinta.
- Usar `config_invalid` para R2 y R3: no permite a una máquina distinguir «configuración ilegible» de «plantilla válida que no aplica a esta rama».
- Fallar cerrado solo con `{ref}` y `{task_ref}`: con `{slug}` o `{kind}` se calcularía un nombre con valores incompletos (hueco G4).
**Razonamiento**: es el mínimo que pide prov:BRA:4, y el trinquete de rechazos y la prueba de constructores lo cubren.

### D-12: la validación de Ajustes reutiliza el revalidado completo

**Elección**: tres filas `TypeString` en `editableSpecs`:
- `git.branch_template`;
- `repositories[].branch_template` (`scopeRepo`, `multirepoOnly`);
- `roles[].branch_template` (`scopeRole`, `multirepoOnly`).

`PUT /api/settings` no llama a ningún validador nuevo: `settings.Write` ya revalida el documento resultante con `workspace.ParseConfig` (`write.go:84`), que acumula `GitErr`, `RepositoriesErr` y `BranchTemplatesErr`. Por eso el rechazo de `{role}`/`{task_ref}` en la plantilla que aloja la rama con `openspec/`, la variable distintiva y los errores con la ruta del elemento (`repositories[<i>].branch_template`, `roles.<id>.branch_template`) llegan por la API sin código nuevo (S10).
**Alternativas consideradas**: una llamada explícita a `workspace.ValidateBranchTemplate` en `patch.go`. Duplica la validación y no ve las reglas de topología.
**Razonamiento**: es una única fuente de verdad (RF-TOP-22), y la regla de topología de RF-TOP-21 se respeta con `multirepoOnly`.

### D-13: `{kind}` y `{ref}` según sus fuentes (prov:BRA:2, S8, G9)

**Elección**: un resolutor del motor, `resolveTemplateValues` (`engine/template_values.go`), solo lee ficheros y solo actúa cuando alguna plantilla del plan usa `ref`, `kind`, `slug` o `task_ref`. Lo comparten `ensure`, `Snapshot`, `Status` y `VerdictFor`. Si hay un plugin activo e `integrations.yaml` no se puede leer, no rechaza en ese momento: entrega `LinksErr`, y el planificador o la guarda rechazan solo según D-11.

| Valor | Cambio SDD sellado | Cambio SDD sin sellar y carril ODD |
|---|---|---|
| `kind` | `TemplateValues.Kind` de la instantánea; sin `template_values`, `feature` (G9) | perfil del staging de `change start` (`bug` → `fix`) > perfil del `work-item.json` adoptado (SDD: `<changeRoot>/context/`; ODD: `odd/context/<feature>/`) > `feature` |
| `ref` | `TemplateValues.Ref` de la instantánea > `id` del vínculo `work_item` de `integrations.yaml` > `id` del staging > sin valor | `id` del vínculo `work_item` de `integrations.yaml` (ODD también lo escribe al adoptar) > `id` del staging > sin valor |
| `slug` | `Slug(change, ref)` (D-03) | igual |
| `task_ref` (por rol) | `id` del vínculo `kind: task` del rol (sin distinguir mayúsculas) de un plugin activo | no aplica: `{task_ref}` se rechaza en ODD (R3) |

El vínculo `work_item` que cuenta es el del primer plugin activo que lo tiene, por orden de declaración. El resolutor marca `KindFrozen` y `RefFrozen` cuando el valor sale de la instantánea, para que D-11 sepa qué está congelado.

En ODD no hay kickoff: el nombre se congela en el registro en el primer `ensure` (RF-BRA-15), y `{ref}` sin work item da `branch_template_unresolved`.

El staging y el documento adoptado se leen con la función exportada nueva `changestart.WorkItemFacts(commonDir, change, adoptedDir string) (WorkItemFacts, bool)`, que envuelve `stagingFor` y `readContextDoc`. `engine` → `changestart` no forma ciclo: `changestart` importa `branch` y `branch/registry`, pero no `engine`.
**Alternativas consideradas**:
- Una bandera `--kind`: fuera del alcance.
- Leer la declaración de bug de `proposal.md`: no existe al crear la rama.
- Sacar `kind` del perfil adoptado también en un cambio sellado: contradice prov:KOF:1, que fija `feature` cuando la instantánea no lo congeló (hueco G9).
**Razonamiento**: `ensure`, `Snapshot`, `Status` y la guarda comparten el resolutor y el planificador. Sin plantilla, no hay lecturas nuevas ni modos de fallo nuevos.

### D-14: un único helper para la rama del cambio resuelta (G3, S9)

**Elección**: función pura nueva `branch.ResolveChangeBranch` (`internal/branch/change_branch.go`). Recibe las fuentes con nombre **en el orden de precedencia del consumidor** y, si ninguna tiene valor, decide entre pendiente, render de la plantilla o composición `<prefijo>/<cambio>`. Es el único sitio del código que compone el nombre de la rama del cambio; `ArchiveBranchFor(change)` es el único que añade `-archive`. Sus consumidores:

| Consumidor | Fuentes con nombre, en orden | Plantilla y prefijo |
|---|---|---|
| `planner.go` (`planRepository`) | `--branch`, instantánea, `Frozen` | D-04 y D-05; `--prefix` > prefijo de la entrada pendiente > política |
| `engine/verdict.go` (`expectedNames`) | `record.Exceptions.Branch`, instantánea, registrado | D-10; `record.Exceptions.Prefix` > política |
| `engine/archive_plan.go` (`registeredChangeBranch`) | registrado, instantánea, `Exceptions.Branch` | plantilla de la rama del cambio del planning home; prefijo de `Exceptions.Prefix` **solo sin plantilla** |
| `planner_archive.go` (`archiveBranchName`) | `ArchiveBranch` registrada, `ChangeBranch` resuelta | rama del cambio resuelta por el helper + `-archive` |
| `engine/autofetch_relevance.go` (`handedOffChange`) | nombre de la instantánea leída de la referencia remota | D-16 |

Reglas del archivado (S9):
- La rama de archivado es la rama del cambio ya resuelta (registro, instantánea o render) más `-archive`.
- Con plantilla en la rama del cambio, `--prefix` en `ensure --archive` no cambia la derivación: no se compone `<prefijo>/<cambio>-archive`, `engine/archive.go` no guarda `ArchivePrefix` y la guarda ignora un `ArchivePrefix` registrado.
- Sin plantilla, el orden y los nombres son los de hoy: el caso `Exceptions.Prefix` sigue primero en `archiveBranchName`, y `ArchivePrefix` se guarda y se lee igual.

Cada consumidor conserva su orden actual de fuentes con nombre, así que sin plantilla la salida es byte a byte la de hoy (salvo el caso límite de D-05 en el planificador y en la guarda).
**Alternativas consideradas**:
- Un orden único de fuentes para todos los consumidores: cambiaría el archivado sin plantilla (hoy el registro manda sobre la instantánea al archivar) y rompería la invariante.
- Parchear cada composición por separado: cinco sitios que pueden divergir, que es justo el hueco G3.
**Razonamiento**: una sola regla de composición y de render, con las fuentes con nombre explícitas en cada llamada, cubre todos los consumidores sin cambiar su semántica actual.

### D-15: `status` proyecta las pendientes y nunca muta (S7)

**Elección**: `Engine.Status` planifica con `PlanInput.Describe = true`. En ese modo, y solo en él, un `branch_template_unresolved` o un `integrations_links_invalid` de una rama concreta no rechaza el plan entero: la rama queda en `RepoPlan.Pending` con `Refusal` relleno, y `describe` la lista con `reason` = el código y `continuation` = la del rechazo. `branch_template_not_applicable`, `invalid_name` y `config_invalid` siguen rechazando `status`, como hoy un `config_invalid`.

Cada `PendingBranchResult` lleva su plantilla, su rol, el motivo, la continuación y `expectedName` solo cuando ya se puede resolver (el `Candidate`). `Snapshot` y `ensure` nunca activan `Describe`, así que el seal y la creación siguen fallando cerrados (S5). `status` no llama a `recoverPending` con escritura ni a `apply`: solo lee referencias, como hoy.
**Alternativas consideradas**:
- Reutilizar `Operation == "status"` como interruptor: `Snapshot` también planifica con `status` y dejaría de propagar los rechazos al seal.
- Fallar `status` entero ante un `integrations.yaml` ilegible: oculta el resto del estado del cambio.
**Razonamiento**: `status` informa sin decidir; el rechazo tipado y su continuación se ven en la entrada afectada.

### D-16: el autofetch reconoce un cambio cedido con plantilla

**Elección**: la regla 3 de `resolveCurrentChange` (`handedOffChange`) hoy lista `refs/remotes/origin/<prefijo>/` y deduce el cambio del nombre de la rama. Con una plantilla efectiva en la rama del cambio del repositorio de specs, el nombre no se puede invertir, así que la regla cambia:
- lista las referencias remotas bajo el prefijo literal de la plantilla: el texto fijo hasta la última `/` anterior a la primera variable; si la plantilla empieza por `{prefix}/`, el prefijo efectivo; si empieza por `{kind}/`, `feature/` y `fix/`;
- para cada referencia (con el mismo límite `autofetchHandoffLimit`), lista con `TreeDirs` las carpetas de `<planRel>/changes` de esa referencia, lee con `FileAt` su `kickoff.yaml` y toma como candidato el cambio cuya instantánea nombra esa misma rama como rama del cambio del planning home;
- se detiene en el segundo candidato (más de uno, no hay cambio actual), y la comprobación de rama local usa ese nombre resuelto y no `<prefijo>/<cambio>`;
- si la plantilla no tiene prefijo literal (empieza por `{ref}`, `{change}` o `{slug}`), la regla 3 no se aplica y deja un aviso en `result.Warnings`. Es una limitación declarada; las reglas 1 y 2 (registro) no cambian.

Sin plantilla, la regla 3 es la de hoy.
**Alternativas consideradas**:
- Desactivar siempre la regla 3 con plantilla: pierde la relevancia del cambio cedido en el caso más común (`feature/…`).
- Listar todas las referencias remotas: coste sin cota en repositorios con muchas ramas.
**Razonamiento**: solo lee referencias ya traídas (sin red), usa la instantánea versionada como fuente del nombre y mantiene la cota actual.

## Flujo de datos

### `ensure` sin rol (y `status`, `Snapshot`)

```
CLI branch ensure ──→ engine.Ensure(EnsureRequest{Role:""})
                          │
                          ├─ gatherWith: Locate + ConfigRefusal (incluye BranchTemplatesErr)
                          │    registro (lock) ─→ RepoFacts.Frozen  (solo RecordBranch.Template != "")
                          │    kickoff ─→ Snapshot (plantillas, pending_branches, template_values)
                          │    needs := templatesInPlay(policy, snapshot)
                          │    needs ≠ ∅ ─→ resolveTemplateValues(snapshot, integrations.yaml, staging, work-item.json)
                          │                  └─→ PlanInput.TemplateValues{Kind, Ref, TaskRefs, LinksErr, *Frozen}
                          ▼
                     branch.Plan (puro)
                          │  rama del cambio: ResolveChangeBranch(--branch, instantánea, Frozen │ pendiente │ render │ prefijo/cambio)
                          │  ramas de rol: instantánea > Frozen > plantilla de rol > <seguimiento>-<rol>
                          │  {task_ref} sin --branch y rol ≠ EnteringRole ─→ RepoPlan.Pending (+Candidate)
                          │  R2/R3/openspec ─→ branch_template_not_applicable
                          │  variable sin valor ─→ branch_template_unresolved (Describe: Pending con Refusal)
                          │  LinksErr y valor no congelado ─→ integrations_links_invalid (Describe: Pending con Refusal)
                          │  checkTemplateCollisions ─→ invalid_name
                          ▼
                     recoverPending: BranchExists(Candidate) en refs/heads
                          │  error ≠ «no existe» ─→ error;  dos roles, mismo nombre ─→ invalid_name
                          │  existe ─→ Frozen += … ─→ Plan (una vez)
                          ▼
                     checkPlan ─→ apply ─→ gather + Plan ─→ newRecord (Template en RecordBranch)
                          ▼
                     describe ─→ RepositoryResult.PendingBranches (JSON aditivo) + texto
```

`status` recorre el mismo camino con `Describe = true` y sin `apply`. `Snapshot` lo recorre con `sealing = true` y sin `Describe`.

### `ensure --role <rol>`

```
CLI: canonicalEnsureRole ─→ validateEnsureRole (archive · odd · kickoff · roster · task_not_bound)
     │ rechazo ─→ sin efectos
     ▼
engine.Ensure(EnsureRequest{Role: rol}) ─→ PlanInput.EnteringRole = rol
     ▼
Plan: la rama pendiente de <rol> se renderiza con TaskRefs[rol] ─→ Branches (From = seguimiento si es de rol)
      las pendientes de otros roles siguen en Pending
      checkTemplateCollisions con los nombres ya registrados del repositorio
     ▼
planWithPromotion: primera vez que el repositorio tiene ramas ─→ base, destino de PR y pregunta `ask`
     ▼
checkPlan (adopta origin/<nombre> si ya está traída, RF-BRA-27) ─→ apply (crea rama y worktree) ─→ registro congela el nombre
     ▼
CLI: recordEnsureRoleStarted + emitEnsureRoleStart (sin cambios)
```

### `kickoff seal` (secuencia `change start` → `ensure` → seal → `--adopt`)

```
change start (fase 1) ─→ staging {id, profile} en <commonDir>/axiom/change-start/v1/<cambio>
ensure ─→ rama del cambio renderizada con ref/kind del staging ─→ registro (Template)
seal ─→ guarda A (VerdictFor: nombre registrado) ─→ Engine.Snapshot
        plan con sealing=true, EnteringRole="" ─→ ramas con nombre (+template), Pending (+prefix)
        template_values: kind siempre (feature|fix), ref solo si se conoce
        rechazo del plan ─→ applySealBranchPolicy devuelve el error ─→ sin kickoff.yaml
        ─→ kickoff.Seal (PublishNoReplace, escritura única)
change start --adopt ─→ vínculo work_item en integrations.yaml; staging retirado
```

### Carril ODD

```
change start (fase 1, opcional) ─→ staging {id, profile}
ensure --lane odd ─→ plantilla de la política (sin kickoff)
     ref: vínculo work_item > staging;  kind: staging > odd/context/<feature>/work-item.json > feature
     {role}/{task_ref} ─→ branch_template_not_applicable (odd_lane)
     {ref} sin work item ─→ branch_template_unresolved
     ─→ apply ─→ registro congela el nombre (los ensure siguientes leen Frozen)
change start --adopt --lane odd ─→ vínculo work_item en integrations.yaml
```

### Archivado

```
ensure --archive ─→ archivePlanInput
     ArchiveBranch registrada ─→ se usa
     si no: ChangeBranch = registeredChangeBranch (registro > instantánea > --branch; --prefix solo sin plantilla)
     si no: ResolveChangeBranch(plantilla del planning home │ prefijo/cambio)
     ─→ ArchiveBranchFor(rama del cambio) = <rama del cambio>-archive
     con plantilla: --prefix sin efecto y ArchivePrefix no se registra
```

## Cambios de ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/branch/nametemplate/nametemplate.go` | Crear | `Variable`, `Variables()`, `Parse`, `Template` (`Raw`, `Uses`, `IsZero`, `Render`), `CheckDistinguishing`, `CheckSample`, `Slug` y errores tipados |
| `internal/branch/nametemplate/nametemplate_test.go` | Crear | Tablas de sintaxis, de render, de valores no seguros, de variable distintiva, de slug (S1) y de muestra |
| `internal/workspace/gitpolicy.go` | Modificar | `BranchTemplate` en `GitConfig` y `RepositoryPolicy`; `branch_template` en `gitKeys` y `repositoryKeys`; caso en `branchValidator`; mensajes de claves admitidas; `ValidateBranchTemplate`, `checkBranchTemplates` (con la identificación de la entrada que aloja `openspec/`, D-02) y `BranchTemplateWarnings`; `resolveGit` copia la plantilla |
| `internal/workspace/types.go` | Modificar | `RoleConfig.BranchTemplate` (`yaml:"branch_template,omitempty"`) |
| `internal/workspace/declarations.go` | Modificar | `Declarations.BranchTemplatesErr`; `checkBranchTemplates` reutiliza `lexicalDisplacement` |
| `internal/workspace/loader.go` | Modificar | `parseLayered` calcula `BranchTemplatesErr` tras `validateBaseConfig`; `joinLoadErrors` lo incluye |
| `cmd/axiom/main.go` | Modificar | `workspace validate` añade `BranchTemplateWarnings(cfg)` |
| `internal/branch/policy.go` | Modificar | `Policy.BranchTemplate` y `RoleTemplates`; `RepoPolicy.BranchTemplate` y `BranchTemplateKey`; `For` hereda; `ConfigRefusal` incluye `BranchTemplatesErr` |
| `internal/branch/change_branch.go` | Crear | `ResolveChangeBranch`, `ChangeBranchInput`, `ResolvedChangeBranch` y `ArchiveBranchFor` (D-14); puro |
| `internal/branch/planner.go` | Modificar | Campos nuevos de `PlanInput`, `RepoFacts`, `RepoPlan` y `BranchPlan`; `planRepository` usa `ResolveChangeBranch` (sustituye la composición de la línea 412) con la precedencia de D-05; `checkTemplateCollisions` |
| `internal/branch/planner_template.go` | Crear | `effectiveTemplate`, `renderBranch`, `pendingFor`, `checkTemplateCollisions` y su mapeo a rechazos; es puro |
| `internal/branch/planner_snapshot.go` | Modificar | `snapshotTemplate` y `snapshotPending` |
| `internal/branch/planner_archive.go` | Modificar | `archiveBranchName` usa `ResolveChangeBranch` y `ArchiveBranchFor`; con plantilla, el caso `Exceptions.Prefix` no aplica (líneas 150-160) |
| `internal/branch/verdict.go` | Modificar | `ExpectedNames.PendingRoles`; `Verdict.Details`; caso `awaiting_role` en `Decide` con su continuación `--role` |
| `internal/branch/template_refusal.go` | Crear | `TemplateUnresolved`, `TemplateNotApplicable`, `TemplateInvalidName`, `TemplateCollision`, `TemplateReservedName`, `TemplateCandidateCollision` e `IntegrationsLinksUnreadable` |
| `internal/branch/types.go` | Modificar | `CodeBranchTemplateUnresolved`, `CodeBranchTemplateNotApplicable`, `PendingReasonAwaitingRole`; `RecordBranch.Template` |
| `internal/branch/labels.go` | Modificar | Etiquetas de los dos códigos y del motivo |
| `internal/branch/result.go` | Modificar | `RepositoryResult.PendingBranches` y `PendingBranchResult` (con `expectedName` y `code`); líneas de texto «pendiente» |
| `internal/branch/engine/ensure.go` | Modificar | `EnsureRequest.Role` → `EnteringRole`; `recoverPending` (errores de `BranchExists` y candidatos repetidos) |
| `internal/branch/engine/engine.go` | Modificar | `gatherWith`: `Frozen` por repositorio, `needs` y `resolveTemplateValues` |
| `internal/branch/engine/template_values.go` | Crear | `templatesInPlay` y `resolveTemplateValues` (solo ficheros; fuentes de D-13; marcas de congelado) |
| `internal/branch/engine/ensure_registry.go` | Modificar | `recordRepository` copia `BranchPlan.Template` en `RecordBranch.Template` |
| `internal/branch/engine/snapshot.go` | Modificar | Escribe `Template`, `PendingBranches` y `TemplateValues` (kind siempre, ref si se conoce) |
| `internal/branch/engine/status.go` | Modificar | `Status` planifica con `Describe`; `describeRepository` proyecta `RepoPlan.Pending` |
| `internal/branch/engine/verdict.go` | Modificar | `expectedNames` con el orden de D-10 vía `ResolveChangeBranch` (líneas 247-256); render sin registro; `PendingRoles`; archivado sin `ArchivePrefix` con plantilla (línea 296) |
| `internal/branch/engine/archive_plan.go` | Modificar | `registeredChangeBranch` usa `ResolveChangeBranch`; con plantilla, ignora `Exceptions.Prefix` (líneas 70-72) |
| `internal/branch/engine/archive.go` | Modificar | Con plantilla, no guarda `ArchivePrefix` (líneas 120 y 128) |
| `internal/branch/engine/autofetch_relevance.go` | Modificar | `handedOffChange` según D-16 (líneas 122-165): prefijo literal, `TreeDirs` + `FileAt`, nombre resuelto en la comprobación local y en la `RecordBranch` |
| `internal/branch/engine/prefs.go` | Modificar | `planWithPromotion` salta los repositorios sin ramas planificadas al resolver la promoción y al buscar la pregunta `ask` |
| `internal/kickoff/gitsnapshot.go` | Modificar | `GitSnapshotBranch.Template`, `GitSnapshotRepo.PendingBranches`, `GitSnapshot.TemplateValues` y `Validate` |
| `internal/changestart/staging.go` | Modificar | `WorkItemFacts` exportada |
| `internal/cli/branch.go` | Modificar | Pasa `Role: *role` a `EnsureRequest` |
| `internal/cli/sdd_kickoff_branch_test.go` | Modificar | Pruebas del seal que propaga el rechazo sin escribir `kickoff.yaml` (S5); el código no cambia |
| `internal/settings/catalog.go` | Modificar | Tres filas `TypeString` |
| `internal/settings/testdata/*` | Modificar | Goldens del catálogo comentado, si enumeran las claves editables |
| `axiom.example.yaml` | Modificar | `#   branch_template: "{prefix}/{change}"` en `git:`; `#     branch_template: "feature/TASK-{task_ref}"` en `repositories:`; una línea doblemente comentada en el rol `backend`; explicación de las siete variables y de la variable distintiva; todos validan al descomentarse (S12) |
| `docs/` (bloque `git:`, si existe documentación de usuario) | Modificar | Variables, precedencia, diferido y limitaciones declaradas (S3, S6, D-16) |

## Interfaces y contratos

```go
// Package nametemplate: pure branch-name templates (no I/O).
package nametemplate

type Variable string

const (
	Prefix  Variable = "prefix"
	Kind    Variable = "kind"
	Change  Variable = "change"
	Slug    Variable = "slug"
	Role    Variable = "role"
	Ref     Variable = "ref"
	TaskRef Variable = "task_ref"
)

func Variables() []Variable // the seven, in this order

type Template struct{ /* raw + parsed segments */ }

func Parse(raw string) (Template, error) // *SyntaxError | *UnknownVariableError
func (t Template) Raw() string
func (t Template) IsZero() bool
func (t Template) Uses(v Variable) bool

type Values map[Variable]string

// Render substitutes every variable; never sanitises.
// Errors: *MissingValueError{Variable}, *UnsafeValueError{Variable, Value, Cause},
// *InvalidNameError{Name, Cause}.
func (t Template) Render(v Values) (string, error)

func CheckDistinguishing(t Template) error // *ConstantTemplateError unless change, slug, ref or task_ref is used
func CheckSample(t Template) error         // renders with safe sample values, checks gitref.ValidRefName

// Slug strips the case-insensitive prefix ^[A-Za-z]+-<ref>- when ref != "";
// it returns change when the prefix does not match or the remainder is empty.
func Slug(change, ref string) string

type SyntaxError struct{ Template string; Offset int; Reason string }
type UnknownVariableError struct{ Template, Name string }
type ConstantTemplateError struct{ Template string }
type MissingValueError struct{ Variable Variable }
type UnsafeValueError struct{ Variable Variable; Value string; Cause error }
type InvalidNameError struct{ Name string; Cause error }
```

```go
// internal/workspace
type GitConfig struct {
	// ...existing fields...
	BranchTemplate string `yaml:"branch_template,omitempty"`
}
type RepositoryPolicy struct {
	// ...existing fields...
	BranchTemplate string `yaml:"branch_template,omitempty"`
}
type RoleConfig struct {
	// ...existing fields...
	BranchTemplate string `yaml:"branch_template,omitempty"`
}
type Declarations struct {
	// ...existing fields...
	BranchTemplatesErr error // roles.<id>.branch_template and the openspec-branch rule
}

func ValidateBranchTemplate(raw string) error              // Parse + CheckDistinguishing + CheckSample
func BranchTemplateWarnings(cfg *WorkspaceConfig) []string // role templates without effect in monorepo-embedded
```

```go
// internal/branch
type Policy struct {
	// ...existing fields...
	BranchTemplate string            // git.branch_template
	RoleTemplates  map[string]string // lower(role) -> roles.<id>.branch_template
}
type RepoPolicy struct {
	// ...existing fields...
	BranchTemplate    string
	BranchTemplateKey string // "git.branch_template" | "repositories[<path>].branch_template"
}

type FrozenBranch struct {
	Kind     BranchKind
	RoleID   string
	Name     string
	Template string
}
type RepoFacts struct {
	// ...existing fields...
	Frozen []FrozenBranch // template-named branches from the registry
}

type TemplateValues struct {
	Kind       string            // "feature" | "fix"; "" means feature
	Ref        string            // "" means unknown
	KindFrozen bool              // Kind comes from the kickoff snapshot
	RefFrozen  bool              // Ref comes from the kickoff snapshot
	TaskRefs   map[string]string // lower(role) -> task id
	LinksErr   error             // integrations.yaml unreadable with an active plugin
}
type PlanInput struct {
	// ...existing fields...
	EnteringRole   string
	TemplateValues TemplateValues
	Describe       bool // status only: per-branch template refusals become pending entries
}

type BranchPlan struct {
	// ...existing fields...
	Template string // effective template that rendered Name; "" without template
}
type PendingBranch struct {
	Kind      BranchKind
	RoleID    string // attributed role, always set
	Template  string
	Key       string   // axiom.yaml key or kickoff.yaml locator
	Prefix    string   // effective {prefix}, frozen by the snapshot
	Candidate string   // rendered name when the role's task_ref is known; "" otherwise
	Refusal   *Refusal // Describe only: why the branch cannot be resolved yet
}
type RepoPlan struct {
	// ...existing fields...
	Pending []PendingBranch
}

// ResolveChangeBranch is the single place that names a change branch (D-14).
type ChangeBranchInput struct {
	Change      string
	Named       []string // named sources in the consumer's precedence order; first non-empty wins
	Template    string   // effective template of the change branch; "" without template
	TemplateKey string
	PendingRole string // attributed role when Template uses {task_ref}
	Entering    bool   // PendingRole is entering (ensure --role)
	Prefix      string // effective {prefix}, already validated by the caller
	Values      TemplateValues
	Refusals    RefusalContext // operation, change, lane and master for typed refusals
}
type ResolvedChangeBranch struct {
	Name      string // "" only when Pending
	Template  string // template that rendered Name; "" when named or composed
	Pending   bool
	Candidate string
}
func ResolveChangeBranch(in ChangeBranchInput) (ResolvedChangeBranch, *Refusal)
func ArchiveBranchFor(changeBranch string) string // changeBranch + "-archive"

type ExpectedNames struct {
	// ...existing fields...
	PendingRoles []string // roles whose branch in this repository awaits their entry
}
type Verdict struct {
	// ...existing fields...
	Details map[string]string `json:"details,omitempty"` // reason=awaiting_role, roles, pendingRoles
}

type RecordBranch struct {
	// ...existing fields...
	Template string `json:"template,omitempty"`
}

const (
	CodeBranchTemplateUnresolved    Code = "branch_template_unresolved"
	CodeBranchTemplateNotApplicable Code = "branch_template_not_applicable"
	PendingReasonAwaitingRole            = "awaiting_role"
)

type PendingBranchResult struct {
	Kind         BranchKind `json:"kind"`
	KindLabel    string     `json:"kindLabel"`
	Role         string     `json:"role"`
	Template     string     `json:"template"`
	ExpectedName string     `json:"expectedName,omitempty"` // only when it can already be resolved
	Reason       string     `json:"reason"`                 // awaiting_role, or the refusal code in status
	ReasonLabel  string     `json:"reasonLabel"`
	Continuation string     `json:"continuation"` // axiom branch ensure --change <c> --role <r> --cwd <dir>
}
// RepositoryResult gains: PendingBranches []PendingBranchResult `json:"pendingBranches,omitempty"`
```

```go
// internal/kickoff
type GitSnapshotBranch struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	Role     string `yaml:"role,omitempty"`
	Template string `yaml:"template,omitempty"`
}
type GitSnapshotPendingBranch struct {
	Kind     string `yaml:"kind"` // change | role
	Role     string `yaml:"role"`
	Template string `yaml:"template"`
	Prefix   string `yaml:"prefix"`
}
// GitSnapshotRepo gains: PendingBranches []GitSnapshotPendingBranch `yaml:"pending_branches,omitempty"`
type GitTemplateValues struct {
	Kind string `yaml:"kind,omitempty"` // feature | fix; always written when a template is in play
	Ref  string `yaml:"ref,omitempty"`  // only when known at seal time
}
// GitSnapshot gains: TemplateValues *GitTemplateValues `yaml:"template_values,omitempty"`
```

```go
// internal/branch/engine
type EnsureRequest struct {
	// ...existing fields...
	Role string // canonical --role; only feeds PlanInput.EnteringRole
}

// internal/changestart
type WorkItemFacts struct{ Plugin, ID, Profile string }
func WorkItemFacts(commonDir, change, adoptedDir string) (WorkItemFacts, bool) // staging first, then adopted work-item.json
```

**Alcanzabilidad (trinquete de código muerto).** Toda función exportada nueva se llama desde `main`:
- `nametemplate.*`: desde la carga de `axiom.yaml`, desde `kickoff.Validate` y desde el planificador;
- `workspace.ValidateBranchTemplate`: desde `branchValidator` y `checkBranchTemplates`;
- `workspace.BranchTemplateWarnings`: desde `cmd/axiom/main.go`;
- `branch.ResolveChangeBranch` y `branch.ArchiveBranchFor`: desde el planificador, el archivado, la guarda y el autofetch;
- `changestart.WorkItemFacts`: desde `resolveTemplateValues`;
- los constructores de `template_refusal.go`: desde el planificador, `recoverPending` y la guarda.

No se exporta nada que solo usen las pruebas. `nametemplate.Variables` se usa en los mensajes de error y en el catálogo de la descripción.

## Estrategia de pruebas

Las pruebas son localizadas: solo los paquetes tocados y con `-run` mientras se itera; la batería completa corre en CI. TDD: para cada unidad, primero la prueba RED observada.

| Capa | Qué probar | Cómo (paquete y `-run`) |
|---|---|---|
| Unitaria pura | Sintaxis (llaves sin cerrar, sueltas o anidadas; variable desconocida), render de los escenarios de prov:BRA:1, valor no seguro (`T#12`) sin saneado, `.lock` y `/` final, plantilla sin variable distintiva (`{prefix}/{kind}`), `Slug` (S1: mayúsculas, `us-2525`, `us-2525-`, sin `ref`, sin prefijo que case) | `go test ./internal/branch/nametemplate/` |
| Carga | Las tres claves; acumulación de errores; variable distintiva; regla de `openspec/` en monorepo (entrada `.`) y en multirepo (entrada del repositorio de specs con y sin `branch_template`, solo con `branch_prefix`, specs absoluto, specs in situ); `git:` con `{task_ref}` válido solo si specs declara la suya; aviso de rol en monorepo; ejemplo descomentado | `go test ./internal/workspace/ -run 'BranchTemplate\|TestGitUnknownKeysAreRejected\|TestRepositoriesInvalidBranchValues\|TestExampleGitBlockIsValidOnceUncommented\|TestExampleRepositories'` |
| Helper de la rama del cambio | Orden de las fuentes con nombre; pendiente con y sin `Candidate`; render; composición sin plantilla idéntica a `prefijo + "/" + cambio`; `ArchiveBranchFor` | `go test ./internal/branch/ -run 'ResolveChangeBranch\|ArchiveBranchFor'` |
| Planificador | Sin plantilla: igualdad byte a byte (las pruebas existentes de `planner_test.go` sin cambios); atribución de D-04 (incluido `fullstack`); precedencia de D-05 con la mezcla «instantánea sin nombre + registro» y «instantánea con nombre + registro distinto»; `--branch` sobre una rama del cambio pendiente (se crea), con `--role`, y nunca sobre una de rol; `--prefix` → `{prefix}` y sin efecto si no se usa; R2, R3 y `openspec`; colisión entre ramas, con la integración y con `main_branch`; colisión al entrar el segundo rol con el mismo `{task_ref}`; pendiente frente a `EnteringRole`; `Candidate`; `LinksErr` con `{ref}`, `{slug}`, `{kind}` y `{task_ref}` congelados o no; `Describe` | `go test ./internal/branch/ -run 'Plan\|Template\|Pending\|Snapshot\|Collision\|Describe'` |
| Archivado (planificador) | Con plantilla, `--prefix` no cambia la rama de archivado; sin plantilla, el caso `--prefix` de hoy | `go test ./internal/branch/ -run 'Archive'` |
| Veredicto puro | `awaiting_role` con uno y con varios roles, su continuación y `Details`; repositorio con un rol dentro y otro pendiente; rama inexistente sin pendiente sigue `wrong_branch` | `go test ./internal/branch/ -run 'Decide\|Verdict'` |
| Constructores | Todos los códigos nuevos llevan etiqueta y `Continuation` no vacía | `go test ./internal/branch/ -run 'Refusal\|Label'` |
| Motor | `{ref}` del staging antes de `--adopt`; vínculo después; `kind` de un cambio sellado sin `template_values` = `feature` (G9); fallo cerrado de `integrations.yaml` (G4); congelado y lectura del registro aunque cambie la plantilla; recuperación con el registro borrado, error de `BranchExists`, candidatos repetidos y Task desvinculada (G5); segundo clon con `ensure --role`, sin registro y con la rama publicada (S2); diferido y `ensure --role` idempotente; la entrada de un rol no toca otro; promoción `ask` en el `ensure --role` que crea la rama; JSON `pendingBranches`; `status` con `integrations.yaml` ilegible (entrada con código, sin fallar); instantánea con `template`, `pending_branches` y `template_values` (kind siempre); cambio sellado sin plantilla no migra; ODD con `{ref}` del vínculo, del staging y sin work item | `go test ./internal/branch/engine/ -run 'Ensure\|Snapshot\|Status\|Template\|Pending\|ODD'` |
| Guarda y archivado (motor) | Veredicto con nombres de plantilla sin registro (`feature/TASK-1235` correcto, G1); pendiente de rol renderizado con su `{task_ref}`; `awaiting_role` de un repositorio de código; rechazo tipado; `ArchivePrefix` ignorado con plantilla; `registeredChangeBranch` con y sin plantilla | `go test ./internal/branch/engine/ -run 'Verdict\|ExpectedNames\|Archive'` |
| Autofetch | Regla 3 con plantilla `feature/…` (cambio cedido reconocido por la instantánea), con dos candidatos, con plantilla sin prefijo literal (aviso) y sin plantilla (igual que hoy) | `go test ./internal/branch/engine/ -run 'Autofetch\|HandedOff'` |
| Instantánea | `Validate` de los campos nuevos; un binario anterior ignora `pending_branches` y `template_values` (se simula decodificando con el struct anterior); el rechazo de `name` vacío en `branches` se mantiene | `go test ./internal/kickoff/ -run 'GitSnapshot\|Snapshot'` |
| Ajustes | Filtrado por topología de las tres filas; `PUT` de `repositories[].branch_template` y `roles[].branch_template` con la ruta del elemento en el error; rechazo por la API de `{role}`/`{task_ref}` en la plantilla de la rama con `openspec/`; goldens | `go test ./internal/settings/ -run 'Catalog\|Patch\|Write'` y `go test ./internal/dashboard/ -run Settings` |
| CLI | `ensure --role` pasa el rol (también `fullstack`); la barrera precede a la creación; texto y JSON con pendientes; seal con plantilla; seal que propaga `branch_template_not_applicable` y `branch_template_unresolved` sin escribir `kickoff.yaml` (S5); trinquete de rechazos | `go test ./internal/cli/ -run 'BranchEnsure\|KickoffBranch\|SealBranch\|EnsureRole\|TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign'` |
| Changestart | `WorkItemFacts`: staging, documento adoptado y ninguno | `go test ./internal/changestart/ -run WorkItemFacts` |

Los trinquetes de CI (código muerto y arquitectura de `internal/gitcommon`) se verifican en CI. En local solo se ejecuta `go test ./internal/gitcommon/ -run Architecture` si se crea `nametemplate`. No se compilan binarios sueltos.

## Matriz de amenazas

El cambio toca la automatización de VCS: los nombres de rama que salen de datos externos (los ID del tracker) se convierten en argumentos de `git branch`, `git worktree add` y `git switch`.

| Frontera | Casos adversarios mínimos | Aplicabilidad | Respuesta del diseño | Pruebas RED previstas |
|---|---|---|---|---|
| Rutas con apariencia de documentación | `requirements.txt`, Markdown ejecutable | N/A: no se clasifican ni ejecutan ficheros | — | — |
| Selección del repositorio git | `git -C`, rutas relativas o absolutas | N/A: la selección no cambia (`cmd.Dir` en el checkout principal, `pathidentity`) | — | — |
| Estado del commit | staged, `commit -a`, índice vacío | N/A: `branch` no hace commits | — | — |
| Estado del push / rama de seguimiento | rama de seguimiento, primer push, refspec explícita | **Aplicable** (solo la adopción local de `origin/<nombre>`, RF-BRA-27, y la lectura de referencias remotas ya traídas en el autofetch, D-16; nunca hay push ni red) | El nombre renderizado pasa `ValidRefName` antes de componer `refs/remotes/origin/<nombre>`. La recuperación (D-09) solo mira `refs/heads`, nunca adopta una rama remota de un rol que no entra. La regla 3 del autofetch solo lee (`RemoteBranches`, `TreeDirs`, `FileAt`) con la cota `autofetchHandoffLimit`, y el prefijo literal pasa `plainArgument` | RED: una rama pendiente de un rol que no entra, con `origin/<candidato>` traída, no se crea en `ensure` sin `--role` ni con `--role` de otro rol; RED: la regla 3 con plantilla no crea ninguna rama local |
| Composición de argumentos (inyección por nombre de rama) | `{task_ref}` = `-D`, `--force`, `T#12`, `a b`, `..`, `x.lock`, `CON`; plantilla `-{ref}` o `{prefix}/` | **Aplicable** | Cada valor pasa `ValidBranchComponent` (empieza por letra o dígito, ASCII seguro, sin nombres reservados de Windows), o `ValidPrefix` en el caso de `{prefix}`, y el nombre completo pasa `ValidRefName` (sin `-` inicial). Nada se sanea: el rechazo es `invalid_name` antes de mutar nada. git se invoca con `gitcommon.Run` (sin shell) | RED en `nametemplate`: un valor por clase rechazado con su variable; RED en el motor: `ensure --role` con `{task_ref}` = `-D` no crea rama, worktree ni registro |
| Órdenes de PR | `--head` explícito, entorno, composición | N/A: no hay automatización de PR | — | — |

## Migración y despliegue

No requiere migración de datos. La compatibilidad queda así:
- Sin `branch_template`, el comportamiento es idéntico.
- Los cambios ya sellados no se migran (D5): su instantánea sin plantilla nunca aplica la de la política (D-05).
- El registro, la instantánea, el veredicto y el JSON solo ganan campos `omitempty`.

**Binario anterior**:
- Ignora `template`, `pending_branches`, `template_values`, `pendingBranches` y `details`.
- Rechaza `branch_template` en `git:` y `repositories:` como clave desconocida (mismo precedente que `promotion`, `TestGitBlockIsToleratedByThePreviousBinary`).
- Ante una rama pendiente, la planificaría con el nombre antiguo. Por eso un equipo que use `{task_ref}` debe actualizar todos los binarios antes de sellar con esa plantilla.

**Reversión**: hay dos vías.
- Retirar `branch_template` de `axiom.yaml` devuelve el comportamiento actual en los cambios nuevos.
- Los cambios con nombres congelados conservan sus ramas, que el binario anterior no relee del registro (se completan con `--branch`).

**Limitaciones declaradas**:
- Con el registro borrado, si la plantilla o la Task cambiaron después de crear la rama, el nombre renderizado puede no coincidir (S3); salidas: `--branch` o restaurar el vínculo.
- La colisión con la rama de otro cambio activo no se detecta (S6).
- La regla 3 del autofetch no reconoce un cambio cedido si la plantilla no tiene prefijo literal (D-16).

Se recomienda probar el cambio en uso antes de depender de una plantilla en un cambio de producto.

## Preguntas abiertas

- [ ] `axiom branch scope` (RF-BRA-35) con un repositorio cuyas ramas siguen pendientes: hoy lo veía con su rama. El diseño propone que aparezca sin ramas (resultado `unverifiable` con `branch_missing`, sin código nuevo). Hay que confirmarlo en `sdd-tasks` con la prueba de `scope_test.go`. No bloquea.

Resueltas en la ronda 1: el orden `--branch` > instantánea (D-05, con el escenario de S4) y el rol `fullstack` en multirepo con `{task_ref}` (D-04 y D-08: `ensure --role fullstack` es válido y su Task se vincula con C-07-e; quién lo invoca en la orquestación queda fuera de alcance y se declara como riesgo).

## Unidades de trabajo (un commit por unidad, una sola PR)

1. **`nametemplate`**: el paquete puro con sus pruebas, incluidas la variable distintiva, el `Slug` de S1 y la fila de inyección de la matriz.
2. **Esquema y carga**: las claves en `git:`, `repositories:` y `roles:`, `BranchTemplatesErr` con la regla de `openspec/` de S11, `BranchTemplateWarnings` con su llamada en `main.go`, `ConfigRefusal`, `axiom.example.yaml` y la documentación de usuario.
3. **Catálogo de Ajustes**: las tres filas, los goldens y las pruebas de `PUT` de S10.
4. **Instantánea del kickoff**: los campos aditivos y `Validate`.
5. **Helper de la rama del cambio**: `ResolveChangeBranch` y `ArchiveBranchFor`, y su adopción sin plantilla en `planner.go`, `planner_archive.go`, `engine/archive_plan.go`, `engine/verdict.go` y `engine/autofetch_relevance.go`, con las pruebas existentes en verde como prueba de identidad byte a byte.
6. **Plantilla en el plan**:
   - `Policy` y `RepoPolicy`, `effectiveTemplate`, la precedencia de D-05 y las colisiones;
   - los códigos, las etiquetas y los constructores;
   - `resolveTemplateValues` (fuentes de D-13, marcas de congelado) y `changestart.WorkItemFacts`;
   - `Engine.Snapshot` escribe los campos nuevos, y el seal que propaga los rechazos queda cubierto;
   - ODD con plantilla.
7. **Registro y guarda**: `RecordBranch.Template`, `RepoFacts.Frozen`, la recuperación de D-09, `expectedNames` (D-10), `awaiting_role` en `Decide` y el archivado con plantilla (D-14).
8. **Diferido por rol**: `EnsureRequest.Role`, `RepoPlan.Pending`, `--branch` sobre una rama pendiente, la promoción sin ramas, `pendingBranches` en `ensure` y en `status` (`Describe`, D-15), la CLI y el contrato nuevo de RF-BRA-33.
9. **Autofetch**: la regla 3 con plantilla (D-16).

**Tamaño previsto**: unas 2.000 a 2.600 líneas, la mitad en pruebas.

## Registro de remediación

- 2026-11-26: cobertura ronda 1; se cierran G1 a G11: la guarda renderiza sin registro y declara `awaiting_role` con continuación `--role` (D-10), un helper único `ResolveChangeBranch` sustituye las composiciones de `planner.go`, `planner_archive.go`, `engine/archive_plan.go`, `engine/verdict.go` y `engine/autofetch_relevance.go` (D-14, D-16), fallo cerrado de `integrations.yaml` también con `{slug}` y `{kind}` (D-11), recuperación con errores de `BranchExists` y candidatos repetidos (D-09), regla de `openspec/` con «existe y declara» e identificación por `path` (D-02), `fullstack` como rol del kickoff (D-04, D-08), precedencia y mezcla instantánea/registro (D-05), `kind` = `feature` sin `template_values` (D-13), `kind` siempre congelado y seal que propaga (D-06), colisiones con integración, `main_branch` y roles ya registrados (D-05); además, S1 (`Slug`), S4 (`--branch` y `--prefix`), S6 (variable distintiva), S7 (`status` con `Describe`, D-15), S8 (ODD) y S9 (archivado).
