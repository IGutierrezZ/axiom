# Diseño: plantillas de nombre de rama y ramas diferidas a la entrada del rol (C-07-f)

## Enfoque técnico

El cambio sigue la recomendación A1 + B1 + C1 + D1 de la propuesta, con las resoluciones del orquestador (R1, R2, R3, precedencia y diferido), y lo encaja en la arquitectura actual de `axiom branch`: un planificador puro (`internal/branch`) que recibe hechos ya reunidos por el motor (`internal/branch/engine`), sin E/S propia.

Las piezas son cinco:

1. **Motor de plantillas puro en un paquete hoja**, `internal/branch/nametemplate`. Analiza, lista variables y renderiza. Lo usan tres consumidores que no se pueden importar entre sí: la carga de `axiom.yaml` (`internal/workspace`), la validación de la instantánea (`internal/kickoff`) y el planificador (`internal/branch`). Por eso no puede vivir en `internal/branch`, que importa `workspace` y `kickoff` (decisión D-01).
2. **Esquema**: `branch_template` en `git:`, en `repositories[]` y en `roles.<id>`, validado al cargar (prov:TOP:1), con un canal de error propio para las reglas que dependen de la topología y de los roles (decisión D-02).
3. **Planificador**: plantilla efectiva por rama (rol > repositorio > `git:`, con la atribución de prov:BRA:3), precedencia de nombres de prov:BRA:3, ramas **pendientes** fuera de `RepoPlan.Branches` (decisión D-07) y rechazos tipados (prov:BRA:4).
4. **Motor**: resuelve los valores de las variables solo cuando alguna plantilla los necesita (prov:BRA:2), entrega al planificador los nombres congelados en el registro y el rol que entra, congela en el registro el nombre de las ramas con plantilla (RF-BRA-15) y recupera las ramas pendientes que ya existen en git.
5. **Instantánea y guarda**: el seal congela la plantilla de cada rama, el prefijo de las ramas pendientes y los valores `kind` y `ref` (prov:KOF:1, R1). La guarda (`verdict.go`) deja de recomponer `prefijo/cambio` cuando hay plantilla y lee los nombres de la instantánea o del registro (RF-BRA-19).

**Invariante central (sin plantilla, nada cambia):** si ninguna plantilla interviene en el plan, el motor no lee `integrations.yaml` ni el staging, el planificador sigue el camino de código actual, `RecordBranch.Template`, `GitSnapshotBranch.Template`, `pending_branches`, `template_values` y `pendingBranches` quedan vacíos y se omiten (`omitempty`), y los nombres, la instantánea, el registro y el JSON son idénticos byte a byte (criterio de éxito 1 y prov:BRA:3).

## Decisiones de arquitectura

### D-01: el motor de plantillas vive en el paquete hoja `internal/branch/nametemplate`

**Elección**: paquete nuevo `internal/branch/nametemplate` que solo importa `internal/gitref` y la biblioteca estándar. Expone `Parse`, `Template.Uses`, `Template.Render`, `CheckSample` y `Slug`, con errores tipados.
**Alternativas consideradas**:
- `internal/branch/template.go`, como decía la propuesta: imposible. `workspace` necesita validar al cargar, y `branch` ya importa `workspace` y `kickoff`, así que se formaría un ciclo.
- Ponerlo en `internal/gitref`: mezcla la sintaxis de una plantilla con las reglas de nombres de git y engorda un paquete que hoy es solo de validación.
- Duplicar el analizador en `workspace` y en `branch`: dos gramáticas que pueden divergir.
**Razonamiento**: un único analizador para la carga, la instantánea y el plan. El paquete sigue bajo `internal/branch/**`, así que la prueba de arquitectura de `internal/gitcommon` también lo cubre, y es puro: no ejecuta procesos ni lee ficheros.

### D-02: validación al cargar en dos canales

**Elección**:
- La sintaxis y la muestra de `git.branch_template` y `repositories[].branch_template` se validan en `branchValidator`, dentro de `decodeStringKeys`, y sus errores llegan por los canales existentes `GitErr` y `RepositoriesErr`.
- Las reglas que necesitan la topología o los roles van a un canal nuevo, `Declarations.BranchTemplatesErr`, que `parseLayered` calcula después de validar la base. Son la sintaxis y la muestra de `roles.<id>.branch_template`, y la regla de la rama con `openspec/`: `git:` en `monorepo-embedded`; en `multirepo`, la entrada de `repositories:` del repositorio de specs o, si no existe, `git:`.
- `joinLoadErrors` y `branch.ConfigRefusal` incluyen el canal nuevo.
- La advertencia «sin efecto» de `roles.<id>.branch_template` en `monorepo-embedded` sale de `workspace.BranchTemplateWarnings`, que se llama en `cmd/axiom/main.go` junto a `PromotionWarnings`.
**Alternativas consideradas**:
- Validar los roles en `validateBaseConfig`: una plantilla de rol rota invalidaría la configuración base entera y bloquearía órdenes ajenas a las ramas (`sdd status`, etc.).
- Pasar la topología a `decodeGit`: acopla el decodificador de un bloque a otro bloque y no resuelve los roles, que no pasan por `decodeStringKeys`.
**Razonamiento**: sigue el patrón de canales por declaración (D-03 de la topología). Los errores se acumulan y cada uno nombra su clave, y `config_invalid` llega a `ensure` por el mismo camino que `GitErr`.

### D-03: vocabulario y reglas de valor

**Elección**: las variables son exactamente `prefix`, `kind`, `change`, `slug`, `role`, `ref` y `task_ref`. Una llave sin cerrar, una llave de cierre suelta, unas llaves anidadas o una variable desconocida son un error de sintaxis. **No hay escape** para llaves literales.

Reglas de cada valor sustituido:

| Variable | Regla |
|---|---|
| `{prefix}` | `gitref.ValidPrefix` |
| `{kind}` | `feature` o `fix` |
| Las demás | `gitref.ValidBranchComponent` |

El nombre resultante debe cumplir `gitref.ValidRefName`. `CheckSample` renderiza con valores de muestra seguros (`feat`, `feature`, `sample-change`, `sample`, `role`, `1`, `1`) para validar la forma al cargar.

`Slug(change, ref)` elimina un prefijo inicial `<segmento sin guion>-<ref>-`, con la comparación del primer segmento sin distinguir mayúsculas. Si el prefijo no existe, o `ref` está vacío, devuelve `change`.
**Alternativas consideradas**:
- Escape `{{`: las llaves no aparecen en ninguna convención conocida, y git rechaza `@{`.
- Sanear los valores (sustituir los caracteres no válidos): lo prohíbe D4 de la propuesta.
**Razonamiento**: es la gramática mínima, y sus errores nombran la variable y el valor sin ambigüedad.

### D-04: atribución de rama a rol y plantilla efectiva

**Elección**: una función pura del planificador, `effectiveTemplate`, decide la plantilla de cada rama:

| Rama | Rol atribuido | Plantilla efectiva |
|---|---|---|
| La que lleva `openspec/`: la del cambio en el repositorio del planning home de un `monorepo-embedded`, o la de la feature ODD en ese repositorio | ninguno | repositorio > `git:` |
| La del cambio en un repositorio de código tocado por **un** rol, incluido `fullstack` | ese rol | rol > repositorio > `git:` |
| La de seguimiento de un repositorio tocado por **dos o más** roles | ninguno | repositorio > `git:` |
| La de rol de un repositorio compartido | ese rol | **solo** `roles.<id>.branch_template`; sin ella, `<seguimiento>-<rol>` |
| La de un repositorio incluido con `--repo` en ODD | ninguno | repositorio > `git:` |
| La de archivado | — | ninguna: `<rama del cambio>-archive` |

`fullstack` es reservado y no tiene entrada en `roles:`, así que nunca tiene plantilla de rol.
**Alternativas consideradas**: aplicar rol > repositorio > `git:` también a las ramas de rol de un repositorio compartido. Con una plantilla de repositorio o de `git:` sin `{role}` ni `{task_ref}`, todas las ramas de rol renderizarían el mismo nombre que la de seguimiento. Y si la plantilla las llevara, la rama de seguimiento ya estaría rechazada por R2.
**Razonamiento**: coincide con prov:BRA:3, nunca produce colisiones por construcción y conserva `<seguimiento>-<rol>` como hoy.

### D-05: precedencia de nombres por rama

**Elección**: en `planRepository`, por rama y de mayor a menor (prov:BRA:3):

1. `--branch`, solo para la rama del cambio.
2. El nombre de la instantánea (`GitSnapshotBranch.Name`).
3. El nombre congelado en el registro: una `RecordBranch` con `Template` no vacío, que llega en `RepoFacts.Frozen`.
4. La plantilla efectiva:
   - en un cambio **sellado**, solo la que congela su instantánea (la entrada pendiente o el campo `template`);
   - en un cambio sin sellar o en ODD, la de la política.
5. El nombre por defecto: `<prefijo>/<cambio>` o `<seguimiento>-<rol>`.

Un cambio sellado cuya instantánea no lleva plantillas (sellado antes de esta capacidad o sin plantilla) **nunca** aplica la plantilla de la política: D5 y prov:KOF:1.
**Alternativas consideradas**:
- Conservar el orden actual («instantánea > `--branch`», D-10). Es incompatible con la resolución del orquestador y con prov:BRA:3. Además, en el único caso en que difieren (un multirepo con un `--branch` nuevo tras el seal y antes de que exista ninguna rama de código), hoy la excepción se registra pero se ignora, lo que es incoherente.
- Leer del registro todos los nombres, no solo los congelados por plantilla: cambiaría el comportamiento sin plantilla (un prefijo cambiado después de crear una rama en ODD).
**Razonamiento**: en todos los flujos alcanzables, `--branch` coincide con el nombre de la instantánea porque esta lo absorbió al sellar y `mergeExceptions` rechaza las contradicciones con ramas materializadas, así que el orden nuevo solo cambia ese caso límite. La guarda adopta el mismo orden (D-10 de este diseño) para que el plan y el veredicto nunca discrepen.

### D-06: congelado en la instantánea (R1)

**Elección**: hay tres campos aditivos en `kickoff.GitSnapshot`, todos `omitempty` y solo cuando alguna plantilla interviene en el plan:
- `GitSnapshotBranch.Template`: la plantilla cruda con la que se nombró la rama. Es informativa y auditable; el nombre sigue mandando.
- `GitSnapshotRepo.PendingBranches []GitSnapshotPendingBranch{Kind, Role, Template, Prefix}`: las ramas cuya plantilla usa `{task_ref}`. **No llevan `name`**. `Prefix` congela el `{prefix}` efectivo de ese repositorio al sellar, porque `repositories[].branch_prefix` no se congela en ningún otro sitio.
- `GitSnapshot.TemplateValues *GitTemplateValues{Kind, Ref}` (`template_values`): `kind` (`feature`|`fix`) y `ref` conocidos al sellar, leídos del staging de `change start`, que sigue presente porque `--adopt` va después del seal. Un valor desconocido no se escribe.

`Validate()`:
- `Template` y `PendingBranches[].Template` se analizan con `nametemplate.Parse`;
- toda entrada pendiente lleva `role` y usa `{task_ref}`;
- `kind` de la entrada pendiente ∈ {`change`, `role`};
- `TemplateValues.Kind` ∈ {`feature`, `fix`};
- `TemplateValues.Ref` cumple `ValidBranchComponent`;
- `Prefix` cumple `ValidPrefix`.

**Alternativas consideradas**:
- Escribir la entrada pendiente dentro de `branches` con `name` vacío. **Rechazada**: el `Validate()` de un binario anterior exige `ValidRefName(name)` y fallaría al cargar el `kickoff.yaml` entero (`kickoff/schema.go:131`). Una lista aparte la ignora sin error (la lectura no es estricta).
- Prerenderizar en la instantánea todo salvo `{task_ref}` (por ejemplo `feature/2525-TASK-{task_ref}`). Es correcto, pero guarda algo que no es la plantilla declarada y R1 pide congelar la plantilla y los valores.
- Congelar `{kind}` y `{ref}` solo en el registro: es local y no garantiza el mismo nombre en otro clon.
**Razonamiento**: todas las entradas de un nombre diferido quedan versionadas:
- la plantilla, el prefijo, `kind` y `ref` en `kickoff.yaml`;
- `{task_ref}` en `integrations.yaml`;
- `{change}` y `{slug}` derivados del nombre y de `ref`;
- `{role}` del rol.

Por eso otro clon con `ensure --role` obtiene el mismo nombre y adopta la rama publicada (RF-BRA-27).

### D-07: ramas pendientes fuera de `RepoPlan.Branches`

**Elección**: una rama cuya plantilla efectiva usa `{task_ref}` se planifica como `PendingBranch` en `RepoPlan.Pending` cuando se cumplen las dos condiciones siguientes:
- no tiene nombre congelado (ni en la instantánea ni en el registro);
- su rol atribuido no es `PlanInput.EnteringRole`.

Una rama pendiente no se crea ni se nombra, no se registra, no se mide (`checkPathLength`) y no se coloca (`checkPlacement`).

Con `EnteringRole` igual a su rol, se renderiza con su `{task_ref}` y entra en `Branches`. Un repositorio con todas sus ramas pendientes queda con `Branches` vacío, y `checkPlan` ya lo salta (`len(repo.Branches) == 0` y papel `code`). La resolución de la promoción tampoco pregunta por un repositorio sin ramas planificadas: la pregunta llega en su `ensure --role`.
**Alternativas consideradas**: `BranchPlan` con `Name` vacío y una marca `Pending`. Obliga a proteger cada consumidor de `Branches`: `checkPlan`, `apply`, `describe`, `newRecord`, `handoff`, `scope`, `done`, la colocación y la longitud.
**Razonamiento**: los consumidores existentes no cambian, y el diferido es aditivo en el tipo y en el JSON.

### D-08: el rol llega al motor solo para el diferido

**Elección**: `engine.EnsureRequest.Role` (canónico, en minúsculas) se copia a `PlanInput.EnteringRole`. El motor no valida el rol: lo hace ya el adaptador `validateEnsureRole` (archive, odd, kickoff, roster y barrera `task_not_bound`, en ese orden), antes de llamar al motor. `Snapshot`, `Status` y el servicio del panel pasan `""`.
**Alternativas consideradas**: crear la rama en cuanto existe la Task, sin mirar el rol (opción D2 de la exploración). Rompe «cuando el rol entra», porque en `continuous` `tasks` y `apply` se encadenan.
**Razonamiento**: con un solo `ensure` idempotente, RF-BRA-33 solo cambia en las ramas pendientes de ese rol: sin `{task_ref}`, el plan con `--role` y sin él es el mismo.

### D-09: congelado en el registro y recuperación

**Elección**: `RecordBranch.Template` (`json:"template,omitempty"`) guarda la plantilla con la que se renderizó la rama. Solo las entradas con `Template` no vacío son «nombres congelados». El motor las entrega al planificador en `RepoFacts.Frozen`, emparejando cada repositorio por identidad del git common dir, como ya hace `recordedBranches`.

Para recuperar un registro borrado, el planificador anota en cada `PendingBranch` su `Candidate`: el nombre renderizado si el `{task_ref}` de su rol ya existe. Tras `branch.Plan`, el motor comprueba con `BranchExists` (referencias locales, sin red) si cada candidato existe. Si alguno existe, lo añade a `RepoFacts.Frozen` y planifica **una sola vez** más.
**Alternativas consideradas**:
- Explorar `refs/heads` buscando nombres que encajen con la plantilla: es ambiguo con varias Tasks.
- Leer también `refs/remotes/origin/<candidato>`: adoptaría una rama publicada de un rol que no entra, lo que contradice prov:BRA:5. La adopción remota sigue siendo cosa del `ensure --role` de ese rol.
**Razonamiento**: es determinista, usa solo las entradas versionadas y nunca crea ni renombra nada (RF-BRA-15).

### D-10: la guarda usa el mismo orden que el plan

**Elección**: `expectedNames` (`engine/verdict.go`) resuelve la rama del cambio con este orden:
1. `record.Exceptions.Branch`;
2. el nombre de la instantánea;
3. el nombre registrado;
4. si la instantánea tiene una entrada pendiente de tipo `change` para el repositorio, ningún nombre todavía (la rama no existe y nadie puede estar en ella);
5. sin instantánea y con plantilla efectiva en la política, el nombre renderizado con los mismos valores, que el veredicto lee de ficheros, sin procesos (RF-BRA-22);
6. el prefijo y el cambio, como hoy.

Las ramas de rol esperadas se resuelven así:
- con instantánea, los nombres de sus entradas `role`, más los nombres registrados para las entradas pendientes;
- sin ella, los registrados, más el nombre renderizado con la plantilla de rol cuando no usa `{task_ref}`, o `<cambio>-<rol>`.

Si la plantilla no se puede resolver (paso 5), `VerdictFor` devuelve un `*branch.RefusalError` con `branch_template_unresolved`.
**Alternativas consideradas**: devolver `not_applicable` con aviso. **Rechazada**: `applySealBranchPolicy` sella **sin instantánea** cuando la guarda avisa, y el cambio quedaría exento de las guardas para siempre.
**Razonamiento**: elimina las tres composiciones `prefijo + "/" + cambio` cuando hay plantilla. Ante lo que no puede calcular, la guarda falla cerrada, igual que con un `kickoff.yaml` ilegible.

### D-11: códigos de rechazo

**Elección**: dos códigos nuevos en `internal/branch/types.go`, con etiqueta en `labels.go` y constructores en `internal/branch/template_refusal.go`, todos con `Continuation` no vacía:

| Código | Etiqueta | Cuándo | Continuación (resumen) |
|---|---|---|---|
| `branch_template_unresolved` | «variable de plantilla sin valor» | `{ref}` sin vínculo, staging ni instantánea; `{task_ref}` del rol que entra sin Task | `{ref}`: inicia el cambio desde su work item (`axiom change start --from <id\|url> --cwd <dir>`), o retira `{ref}` de `<clave>`, o fija el nombre con `--branch`. `{task_ref}`: `axiom integrations tasks bind --change <c> --role <r>=<id> --cwd <dir>` o `tasks create`, y repite `axiom branch ensure --change <c> --role <r> --cwd <dir>` |
| `branch_template_not_applicable` | «plantilla no aplicable» | `{role}`/`{task_ref}` en la rama de seguimiento de un repositorio compartido (R2), en la rama con `openspec/` (red de seguridad de la carga) o en el carril ODD (R3); `Details.reason` = `shared_branch` \| `openspec_branch` \| `odd_lane` | quita `{role}` y `{task_ref}` de `<clave>` (las ramas de rol usan `roles.<id>.branch_template`) y comprueba con `axiom workspace validate --cwd <master>`; en ODD, además, fija el nombre con `axiom branch ensure --lane odd --change <c> --branch <rama> --cwd <dir>` |

Se reutilizan cuatro códigos existentes:
- `invalid_name` para un nombre renderizado inválido o un valor no seguro (`Details.field` = `branch_template` o el nombre de la variable, más `key` y `value`) y para la colisión de dos ramas del mismo repositorio (prov:BRA:3);
- `config_invalid` para la plantilla inválida al cargar;
- `integrations_links_invalid` para un `integrations.yaml` ilegible con un plugin activo cuando una plantilla necesita `{ref}` o `{task_ref}`, con el constructor nuevo `IntegrationsLinksUnreadable(operation, change, master, cause)`, sin rol.

El token del motivo de la rama pendiente, `awaiting_role` («pendiente de la entrada del rol»), también se da de alta en `labels.go`.
**Alternativas consideradas**:
- Un código por motivo estructural (`shared`, `openspec`, `odd`): triplica las etiquetas sin aportar una continuación distinta.
- Usar `config_invalid` para R2 y R3: no permite a una máquina distinguir «configuración ilegible» de «plantilla válida que no aplica a esta rama».
**Razonamiento**: es el mínimo que pide prov:BRA:4, y el trinquete de rechazos y la prueba de constructores lo cubren.

### D-12: la validación de Ajustes reutiliza el revalidado completo

**Elección**: tres filas `TypeString` en `editableSpecs`:
- `git.branch_template`;
- `repositories[].branch_template` (`scopeRepo`, `multirepoOnly`);
- `roles[].branch_template` (`scopeRole`, `multirepoOnly`).

`PUT /api/settings` no llama a ningún validador nuevo: `settings.Write` ya revalida el documento resultante con `workspace.ParseConfig` (`write.go:84`), que acumula `GitErr`, `RepositoriesErr` y `BranchTemplatesErr`.
**Alternativas consideradas**: una llamada explícita a `workspace.ValidateBranchTemplate` en `patch.go`. Duplica la validación y no ve las reglas de topología.
**Razonamiento**: es una única fuente de verdad (RF-TOP-22), y la regla de topología de RF-TOP-21 se respeta con `multirepoOnly`.

### D-13: `{kind}` y `{ref}` según sus fuentes (prov:BRA:2)

**Elección**: un resolutor del motor, `resolveTemplateValues` (`engine/template_values.go`), solo lee ficheros y solo actúa cuando alguna plantilla del plan usa `ref`, `kind`, `slug` o `task_ref`. Si hay un plugin activo e `integrations.yaml` no se puede leer, no rechaza en ese momento: entrega `LinksErr`, y el planificador rechaza solo si una plantilla lo necesita.

| Valor | Fuentes, en orden |
|---|---|
| `kind` | `TemplateValues.Kind` de la instantánea > perfil del staging de `change start` (`bug` → `fix`) > perfil del `context/work-item.json` adoptado (SDD: `<changeRoot>/context/`; ODD: `odd/context/<feature>/`) > `feature` |
| `ref` | `TemplateValues.Ref` de la instantánea > `id` del vínculo `work_item` de `integrations.yaml` cuyo `plugin` es el primer plugin activo con ese vínculo (por orden de declaración) > `id` del staging > `id` del `work-item.json` adoptado > sin valor |
| `task_ref` (por rol) | `id` del vínculo `kind: task` del rol (sin distinguir mayúsculas) de un plugin activo |

El staging y el documento adoptado se leen con la función exportada nueva `changestart.WorkItemFacts(commonDir, change, adoptedDir string) (WorkItemFacts, bool)`, que envuelve `stagingFor` y `readContextDoc`. `engine` → `changestart` no forma ciclo: `changestart` importa `branch` y `branch/registry`, pero no `engine`.
**Alternativas consideradas**:
- Una bandera `--kind`: fuera del alcance.
- Leer la declaración de bug de `proposal.md`: no existe al crear la rama.
**Razonamiento**: `ensure` y `Snapshot` comparten el resolutor y el planificador. Sin plantilla, no hay lecturas nuevas ni modos de fallo nuevos.

## Flujo de datos

### `ensure` sin rol (y `status`, `Snapshot`)

```
CLI branch ensure ──→ engine.Ensure(EnsureRequest{Role:""})
                          │
                          ├─ gatherWith: Locate + ConfigRefusal (incluye BranchTemplatesErr)
                          │    registro (lock) ─→ RepoFacts.Frozen  (solo RecordBranch.Template != "")
                          │    kickoff ─→ Snapshot (plantillas, pending_branches, template_values)
                          │    needs := templatesInPlay(policy, snapshot)
                          │    needs ≠ ∅ ─→ resolveTemplateValues(integrations.yaml, staging, work-item.json)
                          │                  └─→ PlanInput.TemplateValues{Kind, Ref, TaskRefs, LinksErr}
                          ▼
                     branch.Plan (puro)
                          │  por rama: --branch > nombre de la instantánea > Frozen > plantilla > defecto
                          │  {task_ref} y rol ≠ EnteringRole ─→ RepoPlan.Pending (+Candidate)
                          │  R2/R3/openspec ─→ branch_template_not_applicable
                          │  variable sin valor ─→ branch_template_unresolved
                          ▼
                     recoverPending: ¿existe Candidate en refs/heads? ─→ Frozen += … ─→ Plan (una vez)
                          ▼
                     checkPlan ─→ apply ─→ gather + Plan ─→ newRecord (Template en RecordBranch)
                          ▼
                     describe ─→ RepositoryResult.PendingBranches (JSON aditivo) + texto
```

### `ensure --role <rol>`

```
CLI: canonicalEnsureRole ─→ validateEnsureRole (archive · odd · kickoff · roster · task_not_bound)
     │ rechazo ─→ sin efectos
     ▼
engine.Ensure(EnsureRequest{Role: rol}) ─→ PlanInput.EnteringRole = rol
     ▼
Plan: la rama pendiente de <rol> se renderiza con TaskRefs[rol] ─→ Branches (From = seguimiento si es de rol)
      las pendientes de otros roles siguen en Pending
     ▼
checkPlan (adopta origin/<nombre> si ya está traída, RF-BRA-27) ─→ apply ─→ registro congela el nombre
     ▼
CLI: recordEnsureRoleStarted + emitEnsureRoleStart (sin cambios)
```

### `kickoff seal` (secuencia `change start` → `ensure` → seal → `--adopt`)

```
change start (fase 1) ─→ staging {id, profile} en <commonDir>/axiom/change-start/v1/<cambio>
ensure ─→ rama del cambio renderizada con ref/kind del staging ─→ registro (Template)
seal ─→ guarda A (VerdictFor: nombre registrado) ─→ Engine.Snapshot
        plan con sealing=true, EnteringRole="" ─→ ramas con nombre (+template), Pending (+prefix)
        template_values {kind, ref} del staging  (solo si alguna plantilla interviene)
        ─→ kickoff.Seal (PublishNoReplace, escritura única)
change start --adopt ─→ vínculo work_item en integrations.yaml; staging retirado
```

## Cambios de ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/branch/nametemplate/nametemplate.go` | Crear | `Variable`, `Variables()`, `Parse`, `Template` (`Raw`, `Uses`, `IsZero`, `Render`), `CheckSample`, `Slug` y errores tipados |
| `internal/branch/nametemplate/nametemplate_test.go` | Crear | Tablas de sintaxis, de render, de valores no seguros, de slug y de muestra |
| `internal/workspace/gitpolicy.go` | Modificar | `BranchTemplate` en `GitConfig` y `RepositoryPolicy`; `branch_template` en `gitKeys` y `repositoryKeys`; caso en `branchValidator`; mensajes de claves admitidas; `ValidateBranchTemplate`, `checkBranchTemplates` y `BranchTemplateWarnings`; `resolveGit` copia la plantilla |
| `internal/workspace/types.go` | Modificar | `RoleConfig.BranchTemplate` (`yaml:"branch_template,omitempty"`) |
| `internal/workspace/declarations.go` | Modificar | `Declarations.BranchTemplatesErr` |
| `internal/workspace/loader.go` | Modificar | `parseLayered` calcula `BranchTemplatesErr` tras `validateBaseConfig`; `joinLoadErrors` lo incluye |
| `cmd/axiom/main.go` | Modificar | `workspace validate` añade `BranchTemplateWarnings(cfg)` |
| `internal/branch/policy.go` | Modificar | `Policy.BranchTemplate` y `RoleTemplates`; `RepoPolicy.BranchTemplate` y `BranchTemplateKey`; `For` hereda; `ConfigRefusal` incluye `BranchTemplatesErr` |
| `internal/branch/planner.go` | Modificar | Campos nuevos de `PlanInput`, `RepoFacts`, `RepoPlan` y `BranchPlan`; `planRepository` con la precedencia de D-05; colisiones |
| `internal/branch/planner_template.go` | Crear | `effectiveTemplate`, `renderBranch`, `pendingFor` y su mapeo a rechazos; es puro |
| `internal/branch/planner_snapshot.go` | Modificar | `snapshotTemplate` y `snapshotPending` |
| `internal/branch/template_refusal.go` | Crear | `TemplateUnresolved`, `TemplateNotApplicable`, `TemplateInvalidName`, `TemplateCollision` e `IntegrationsLinksUnreadable` |
| `internal/branch/types.go` | Modificar | `CodeBranchTemplateUnresolved`, `CodeBranchTemplateNotApplicable`, `PendingReasonAwaitingRole`; `RecordBranch.Template` |
| `internal/branch/labels.go` | Modificar | Etiquetas de los dos códigos y del motivo |
| `internal/branch/result.go` | Modificar | `RepositoryResult.PendingBranches` y `PendingBranchResult`; líneas de texto «pendiente» |
| `internal/branch/engine/ensure.go` | Modificar | `EnsureRequest.Role` → `EnteringRole`; `recoverPending` |
| `internal/branch/engine/engine.go` | Modificar | `gatherWith`: `Frozen` por repositorio, `needs` y `resolveTemplateValues` |
| `internal/branch/engine/template_values.go` | Crear | `templatesInPlay` y `resolveTemplateValues` (solo ficheros) |
| `internal/branch/engine/ensure_registry.go` | Modificar | `recordRepository` copia `BranchPlan.Template` en `RecordBranch.Template` |
| `internal/branch/engine/snapshot.go` | Modificar | Escribe `Template`, `PendingBranches` y `TemplateValues` |
| `internal/branch/engine/status.go` | Modificar | `describeRepository` proyecta `RepoPlan.Pending` |
| `internal/branch/engine/verdict.go` | Modificar | `expectedNames` con el orden de D-10; entradas pendientes de la instantánea |
| `internal/branch/engine/promotion*.go` | Modificar | No pregunta la promoción de un repositorio sin ramas planificadas |
| `internal/kickoff/gitsnapshot.go` | Modificar | `GitSnapshotBranch.Template`, `GitSnapshotRepo.PendingBranches`, `GitSnapshot.TemplateValues` y `Validate` |
| `internal/changestart/staging.go` | Modificar | `WorkItemFacts` exportada |
| `internal/cli/branch.go` | Modificar | Pasa `Role: *role` a `EnsureRequest` |
| `internal/settings/catalog.go` | Modificar | Tres filas `TypeString` |
| `internal/settings/testdata/*` | Modificar | Goldens del catálogo comentado, si enumeran las claves editables |
| `axiom.example.yaml` | Modificar | `#   branch_template: "{prefix}/{change}"` en `git:`; `#     branch_template: "feature/TASK-{task_ref}"` en `repositories:`; una línea doblemente comentada en el rol `backend`; explicación de las siete variables |
| `docs/` (bloque `git:`, si existe documentación de usuario) | Modificar | Variables, precedencia y diferido |

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

func CheckSample(t Template) error   // renders with safe sample values, checks gitref.ValidRefName
func Slug(change, ref string) string // strips "<segment>-<ref>-"; otherwise returns change

type SyntaxError struct{ Template string; Offset int; Reason string }
type UnknownVariableError struct{ Template, Name string }
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

func ValidateBranchTemplate(raw string) error            // Parse + CheckSample
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
	Kind     string            // "feature" | "fix"; "" means feature
	Ref      string            // "" means unknown
	TaskRefs map[string]string // lower(role) -> task id
	LinksErr error             // integrations.yaml unreadable with an active plugin
}
type PlanInput struct {
	// ...existing fields...
	EnteringRole   string
	TemplateValues TemplateValues
}

type BranchPlan struct {
	// ...existing fields...
	Template string // effective template that rendered Name; "" without template
}
type PendingBranch struct {
	Kind      BranchKind
	RoleID    string // attributed role, always set
	Template  string
	Key       string // axiom.yaml key or kickoff.yaml locator
	Prefix    string // effective {prefix}, frozen by the snapshot
	Candidate string // rendered name when the role's task_ref is known; "" otherwise
}
type RepoPlan struct {
	// ...existing fields...
	Pending []PendingBranch
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
	Reason       string     `json:"reason"`
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
	Kind string `yaml:"kind,omitempty"` // feature | fix
	Ref  string `yaml:"ref,omitempty"`
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
- `changestart.WorkItemFacts`: desde `resolveTemplateValues`;
- los constructores de `template_refusal.go`: desde el planificador y la guarda.

No se exporta nada que solo usen las pruebas. `nametemplate.Variables` se usa en los mensajes de error y en el catálogo de la descripción.

## Estrategia de pruebas

Las pruebas son localizadas: solo los paquetes tocados y con `-run` mientras se itera; la batería completa corre en CI. TDD: para cada unidad, primero la prueba RED observada.

| Capa | Qué probar | Cómo (paquete y `-run`) |
|---|---|---|
| Unitaria pura | Sintaxis (llaves sin cerrar, sueltas o anidadas; variable desconocida), render de los escenarios de prov:BRA:1, valor no seguro (`T#12`) sin saneado, `.lock` y `/` final, `Slug` | `go test ./internal/branch/nametemplate/` |
| Carga | Las tres claves; acumulación de errores; regla de `openspec/` en monorepo y multirepo (con y sin entrada del repositorio de specs); aviso de rol en monorepo; ejemplo descomentado | `go test ./internal/workspace/ -run 'BranchTemplate\|TestGitUnknownKeysAreRejected\|TestRepositoriesInvalidBranchValues\|TestExampleGitBlockIsValidOnceUncommented\|TestExampleRepositories'` |
| Planificador | Sin plantilla: igualdad byte a byte (las pruebas existentes de `planner_test.go` sin cambios); atribución de D-04; precedencia de D-05; `--prefix` → `{prefix}`; R2, R3 y `openspec`; colisión; pendiente frente a `EnteringRole`; `Candidate`; `LinksErr` solo si se necesita | `go test ./internal/branch/ -run 'Plan\|Template\|Pending\|Snapshot'` |
| Constructores | Todos los códigos nuevos llevan etiqueta y `Continuation` no vacía | `go test ./internal/branch/ -run 'Refusal\|Label'` |
| Motor | `{ref}` del staging antes de `--adopt`; vínculo después; fallo cerrado de `integrations.yaml`; congelado y lectura del registro aunque cambie la plantilla; recuperación con el registro borrado; diferido y `ensure --role` idempotente; la entrada de un rol no toca otro; JSON `pendingBranches`; `status`; instantánea con `template`, `pending_branches` y `template_values`; cambio sellado sin plantilla no migra; veredicto con nombres de plantilla y rechazo tipado | `go test ./internal/branch/engine/ -run 'Ensure\|Snapshot\|Verdict\|Status\|Template\|Pending'` |
| Instantánea | `Validate` de los campos nuevos; un binario anterior ignora `pending_branches` y `template_values` (se simula decodificando con el struct anterior); el rechazo de `name` vacío en `branches` se mantiene | `go test ./internal/kickoff/ -run 'GitSnapshot\|Snapshot'` |
| Ajustes | Filtrado por topología de las tres filas; un `PUT` inválido se rechaza por `ParseConfig`; goldens | `go test ./internal/settings/ -run 'Catalog\|Patch\|Write'` y `go test ./internal/dashboard/ -run Settings` |
| CLI | `ensure --role` pasa el rol; la barrera precede a la creación; texto y JSON con pendientes; seal con plantilla; trinquete de rechazos | `go test ./internal/cli/ -run 'BranchEnsure\|KickoffBranch\|EnsureRole\|TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign'` |
| Changestart | `WorkItemFacts`: staging, documento adoptado y ninguno | `go test ./internal/changestart/ -run WorkItemFacts` |

Los trinquetes de CI (código muerto y arquitectura de `internal/gitcommon`) se verifican en CI. En local solo se ejecuta `go test ./internal/gitcommon/ -run Architecture` si se crea `nametemplate`. No se compilan binarios sueltos.

## Matriz de amenazas

El cambio toca la automatización de VCS: los nombres de rama que salen de datos externos (los ID del tracker) se convierten en argumentos de `git branch`, `git worktree add` y `git switch`.

| Frontera | Casos adversarios mínimos | Aplicabilidad | Respuesta del diseño | Pruebas RED previstas |
|---|---|---|---|---|
| Rutas con apariencia de documentación | `requirements.txt`, Markdown ejecutable | N/A: no se clasifican ni ejecutan ficheros | — | — |
| Selección del repositorio git | `git -C`, rutas relativas o absolutas | N/A: la selección no cambia (`cmd.Dir` en el checkout principal, `pathidentity`) | — | — |
| Estado del commit | staged, `commit -a`, índice vacío | N/A: `branch` no hace commits | — | — |
| Estado del push / rama de seguimiento | rama de seguimiento, primer push, refspec explícita | **Aplicable** (solo la adopción local de `origin/<nombre>`, RF-BRA-27; nunca hay push) | El nombre renderizado pasa `ValidRefName` antes de componer `refs/remotes/origin/<nombre>`. La recuperación (D-09) solo mira `refs/heads`, nunca adopta una rama remota de un rol que no entra | RED: una rama pendiente de un rol que no entra, con `origin/<candidato>` traída, no se crea en `ensure` sin `--role` ni con `--role` de otro rol |
| Composición de argumentos (inyección por nombre de rama) | `{task_ref}` = `-D`, `--force`, `T#12`, `a b`, `..`, `x.lock`, `CON`; plantilla `-{ref}` o `{prefix}/` | **Aplicable** | Cada valor pasa `ValidBranchComponent` (empieza por letra o dígito, ASCII seguro, sin nombres reservados de Windows), o `ValidPrefix` en el caso de `{prefix}`, y el nombre completo pasa `ValidRefName` (sin `-` inicial). Nada se sanea: el rechazo es `invalid_name` antes de mutar nada. git se invoca con `gitcommon.Run` (sin shell) | RED en `nametemplate`: un valor por clase rechazado con su variable; RED en el motor: `ensure --role` con `{task_ref}` = `-D` no crea rama, worktree ni registro |
| Órdenes de PR | `--head` explícito, entorno, composición | N/A: no hay automatización de PR | — | — |

## Migración y despliegue

No requiere migración de datos. La compatibilidad queda así:
- Sin `branch_template`, el comportamiento es idéntico.
- Los cambios ya sellados no se migran (D5): su instantánea sin plantilla nunca aplica la de la política (D-05).
- El registro, la instantánea y el JSON solo ganan campos `omitempty`.

**Binario anterior**:
- Ignora `template`, `pending_branches`, `template_values` y `pendingBranches`.
- Rechaza `branch_template` en `git:` y `repositories:` como clave desconocida (mismo precedente que `promotion`, `TestGitBlockIsToleratedByThePreviousBinary`).
- Ante una rama pendiente, la planificaría con el nombre antiguo. Por eso un equipo que use `{task_ref}` debe actualizar todos los binarios antes de sellar con esa plantilla.

**Reversión**: hay dos vías.
- Retirar `branch_template` de `axiom.yaml` devuelve el comportamiento actual en los cambios nuevos.
- Los cambios con nombres congelados conservan sus ramas, que el binario anterior no relee del registro (se completan con `--branch`).

Se recomienda probar el cambio en uso antes de depender de una plantilla en un cambio de producto.

## Preguntas abiertas

- [ ] Orden de `--branch` frente a la instantánea (D-05): el diseño sigue prov:BRA:3 y la resolución del orquestador. Solo cambia un caso límite hoy incoherente: un multirepo con un `--branch` nuevo tras el seal y antes de crear ninguna rama de código. No bloquea.
- [ ] `axiom branch scope` (RF-BRA-35) con un repositorio cuyas ramas siguen pendientes: hoy lo veía con su rama. El diseño propone que aparezca sin ramas (resultado `unverifiable` con `branch_missing`, sin código nuevo). Hay que confirmarlo en `sdd-tasks` con la prueba de `scope_test.go`. No bloquea.
- [ ] Flujo `continuous` con `fullstack` en multirepo y una plantilla con `{task_ref}`: las ramas de código nacen en `ensure --change <c> --role fullstack`. Hay que comprobar que la orquestación de `apply` ya lo invoca para el rol único; si no, `ensure` y `status` las listan como pendientes con su continuación. No bloquea.

## Unidades de trabajo (un commit por unidad, una sola PR)

1. **`nametemplate`**: el paquete puro con sus pruebas, incluida la fila de inyección de la matriz.
2. **Esquema y carga**: las claves en `git:`, `repositories:` y `roles:`, `BranchTemplatesErr`, `BranchTemplateWarnings` con su llamada en `main.go`, `ConfigRefusal`, `axiom.example.yaml` y la documentación de usuario.
3. **Catálogo de Ajustes**: las tres filas y los goldens.
4. **Instantánea del kickoff**: los campos aditivos y `Validate`.
5. **Plantilla en el plan**:
   - `Policy` y `RepoPolicy`, `effectiveTemplate` y la precedencia de D-05;
   - los códigos, las etiquetas y los constructores;
   - `resolveTemplateValues` y `changestart.WorkItemFacts`;
   - `Engine.Snapshot` escribe los campos nuevos, y el seal queda cubierto.
6. **Registro y guarda**: `RecordBranch.Template`, `RepoFacts.Frozen`, la recuperación de D-09 y `expectedNames` (D-10).
7. **Diferido por rol**: `EnsureRequest.Role`, `RepoPlan.Pending`, la promoción sin ramas, `pendingBranches` en `ensure` y `status`, la CLI y el contrato nuevo de RF-BRA-33.

**Tamaño previsto**: unas 1.500 a 2.000 líneas, la mitad en pruebas.
