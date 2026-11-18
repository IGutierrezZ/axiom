# Propuesta: promoción de ramas configurable por repositorio y por cambio (C-02-a)

## Intención

Hoy un proyecto puede declarar `git.integration_branch: develop`. Las ramas salen de develop y la PR apunta a develop (`internal/branch/planner.go:392`, `internal/branch/engine/status.go:170-182`). Pero Axiom no conoce la rama principal ni el segundo salto hacia ella. Además, `axiom branch done` borra la rama en cuanto la confirma fusionada en la integración (`internal/branch/engine/done.go:441-499` `mergeEvidence`, `:394` `DeleteBranch`). Con develop configurado, la rama desaparece al llegar a develop, justo cuando el equipo aún necesita llevarla a master.

El caso real (kvp25) mezcla dos flujos dentro de un mismo proyecto e incluso de un mismo cambio:

- **`branch`**: la rama sale de la rama principal, entra en la integración sin borrarse y después pasa a la rama principal, momento en que se borra.
- **`release`**: la rama sale de la integración, entra en ella y se borra. El paso a la rama principal es la release develop→master, ajena al cambio. Es el comportamiento actual.

El usuario lo resume así: «por defecto main recibe cada feature, pero hay casos en que recibe develop completo». La elección depende del repositorio y, en algunos repositorios, del cambio (por ejemplo, migraciones de Entity Framework). Por eso hace falta un modo `ask` que decida la persona al crear la rama.

Éxito: cada repositorio de cada cambio sigue uno de los dos flujos, con la base correcta desde la creación de la rama. Axiom acompaña también la segunda PR, hacia la rama principal, y `done` solo borra la rama cuando el destino final la tiene. Sin `main_branch` declarado, el comportamiento actual no cambia.

## Alcance

### Dentro del alcance

1. **Esquema de `axiom.yaml`.**
   - Clave nueva `git.main_branch`, solo en `git:` y no por repositorio. Debe ser un nombre de referencia válido.
   - Clave nueva `promotion` (`branch` | `release` | `ask`), en `git:` y en cada entrada de `repositories[]`, con la herencia campo a campo de hoy.
   - Validación por clave con los errores acumulados actuales y los textos de error ampliados.
   - `resolveGit` no rellena `main_branch` por defecto.
2. **Semántica de los defectos.**
   - Sin `main_branch`, o con `main_branch` igual a la integración, hay una sola etapa: comportamiento actual intacto y `promotion` sin efecto.
   - Con `main_branch` y sin `promotion`, el modo es `branch`.
3. **Política y plan.**
   - `branch.Policy` y `RepoPolicy` incorporan `MainBranch` y `Promotion`.
   - El plan puro recibe el modo ya resuelto por repositorio y calcula la base efectiva: `main_branch` en `branch`, la integración en `release` o con una sola etapa.
   - `IntegrationBranch` no cambia de semántica y `prTarget` de la rama del cambio sigue siendo la integración.
   - El repositorio de specs de multirepo queda exento y las ramas de rol siguen apuntando a la rama de seguimiento.
4. **Resolución del modo en `axiom branch ensure`**, por repositorio y con la precedencia **registro > instantánea > `--promotion` > `repositories[].promotion` > `git.promotion` > defecto**.
   - Con `ask` y sin respuesta, `ensure` devuelve `decision_required`, con el mismo mecanismo y esquema que la pregunta worktree o in place. Ofrece las respuestas `branch` y `release`, con un repositorio por llamada y de forma secuencial si hay varios. Cada invocación propuesta conserva las respuestas ya dadas.
   - Flag nuevo `--promotion <modo>`, o `--promotion <repo>=<modo>`, repetible en multirepo. `ask` no es una respuesta válida.
   - Si se pide un modo distinto del registrado, el resultado es `plan_conflict` y nunca un rebase.
5. **Persistencia del modo resuelto.**
   - El registro local de ramas guarda, por repositorio, el modo y su origen en campos `omitempty`. Es la fuente operativa en todos los carriles, incluido ODD. `newRecord` conserva el valor previo.
   - Cuando el kickoff se sella con la rama ya creada (monorepo SDD), la instantánea git copia el modo de cada repositorio en un campo opcional `omitempty`. Su `base` refleja la base efectiva.
   - La instantánea se sigue escribiendo una sola vez y `ensure` nunca la modifica. Si la instantánea trae el modo y el registro no, `ensure` lo respeta.
6. **`axiom branch status --json`.**
   - En modo `branch`, después de confirmar con git local, sin red, que la rama está fusionada en la integración, presenta la **siguiente PR**: head = rama del cambio, base = `main_branch`.
   - Hasta esa fusión, `prTarget` sigue siendo la integración.
   - Expone de forma aditiva el modo resuelto y la rama principal por repositorio.
7. **`axiom branch done`.**
   - En `branch`, confirma la fusión en `main_branch` con las mismas capas de `ConfirmMerge`: ascendencia, árbol equivalente, `pr_merged` y `--force`. En `release` o con una sola etapa, la confirma en la integración, como hoy.
   - El rechazo `MergeNotConfirmed` nombra el destino real.
   - El repositorio de specs se omite, como hoy.
   - La tolerancia a squash se mantiene (P1 de README 6.2): árbol equivalente o `--force`.
8. **Archivado.** La rama de archivado (REQ-25.45) sigue saliendo de la integración y solo espera la fusión en ella, nunca en `main_branch`. Se hace explícito en la spec.
9. **Scripts de dogfooding** (`scripts/axiom-worktree.ps1` y `.sh`).
   - `done` deja de fijar `main` a mano y usa `repositories[].integrationBranch` del `status --json`. Si no hay JSON legible, cae a `main`.
   - `new`, `pr` y `done` siguen funcionando igual con la configuración actual del repositorio.
10. **Documentación.**
    - `axiom.example.yaml` (bloques `git:` y `repositories:`) y `docs/manual-de-inicio.md`, sin reformular las líneas fijadas por `docs_contract_test.go`.
    - El manual incluye un aviso de compatibilidad: un binario anterior rechaza `git.main_branch` y `git.promotion` por clave desconocida.
    - Ajuste de la doctrina de skills (`sdd-apply`, `sdd-archive`, `_shared/sdd-status-contract.md`) solo donde afirme algo que deje de ser cierto en modo `branch`.
11. **Tests** de cada capa: esquema, política, plan, motor (`ensure`, `status`, `done`, `snapshot`), CLI, registro, instantánea y scripts. Incluyen squash real en las dos etapas.

### Fuera del alcance

- Abrir PRs, hacer push o fusionar: siguen siendo de la persona.
- Detectar automáticamente migraciones u otros criterios para elegir el modo. Como mucho, el orquestador sugiere y la persona decide.
- `main_branch` por repositorio.
- Migración retroactiva de cambios ya sellados o registrados: sin el campo, se tratan como una sola etapa, o con el modo que resuelva la política vigente si la rama aún no existe.
- Cambiar `PublicationTarget` (`sdd publish`), `HandOff` o `livespec`: siguen con el destino de la primera etapa.
- Hacer que un binario anterior tolere las claves nuevas: solo se documenta.
- La verificación conjunta antes de la PR a la rama principal (C-02-b).
- La recomendación doctrinal de merge commit frente a squash (C-02-d).
- Reflejar `main_branch` y `promotion` en el dashboard (REQ-25.33): lo cataloga C-06-e.
- Una lista de N etapas de promoción: descartada en el backlog.

## Capacidades

### Capacidades nuevas

Ninguna.

### Capacidades modificadas

- `change-branch-lifecycle` (principal).
  - Requisitos nuevos a partir de **REQ-25.46**:
    - resolución del modo de promoción por repositorio y cambio, con su precedencia y la persistencia en el registro;
    - base efectiva y `ask`/`--promotion`/`plan_conflict` en `ensure`;
    - «siguiente PR» en `status`;
    - destino de confirmación de `done` según el modo.
  - Requisitos existentes que se modifican, solo donde cambie su comportamiento:
    - REQ-25.3: política;
    - REQ-25.8: `decision_required` deja de ser exclusiva de worktree o in place;
    - REQ-25.10: base;
    - REQ-25.14: registro;
    - REQ-25.15: status;
    - REQ-25.16 y REQ-25.17: capas y `done` seguro contra el destino del modo;
    - REQ-25.45: el archivado espera solo a la integración;
    - REQ-25.24: scripts de dogfooding.
- `workspace-topology`.
  - REQ-25.26: claves `main_branch` y `promotion` en `git:`, sus reglas y sus defectos.
  - REQ-25.27: `promotion` en `repositories[]`, con herencia.
- `sdd-lane-kickoff`. REQ-25.36: la instantánea admite el modo de promoción opcional por repositorio y la base efectiva. Se sigue escribiendo una sola vez y `ensure` no la modifica. Se incluye porque el requisito enumera el contenido de la instantánea.

## Enfoque

Se sigue la opción C del explore, con el registro como fuente operativa, y se reutilizan los patrones existentes en lugar de inventar mecanismos:

1. **Esquema** (`internal/workspace/gitpolicy.go`).
   - Se amplían `GitConfig`, `RepositoryPolicy`, `gitKeys`, `repositoryKeys` y los validadores por clave: `ValidRefName` para `main_branch` y un validador de enumerado para `promotion`.
   - `resolveGit` no completa `main_branch`.
2. **Política** (`internal/branch/policy.go`). `MainBranch` y `Promotion` heredan en `ResolvePolicy` y `For` con la regla actual: el valor vacío hereda.
3. **Resolución en el motor** (`internal/branch/engine/prefs.go`, `ensure.go`).
   - `resolvePromotion` por repositorio, análoga a `resolveMode`. Se ejecuta tras `gather` y antes de `branch.Plan`, sobre los repositorios que el plan incluiría, salvo el de specs.
   - Un `PromotionConflict`, análogo a `ModeConflict`, produce `plan_conflict`.
   - `ask` sin respuesta produce `decision_required` con `Details.repository` y una invocación (`promotionInvocation`) que conserva `--mode` y las excepciones.
4. **Plan puro** (`internal/branch/planner.go`).
   - `PlanInput` recibe el modo resuelto. `RepoPlan` lleva el modo, `main_branch` y la base efectiva.
   - `IntegrationBranch` y `PRTarget` estáticos no cambian.
   - `baseCandidates` (`engine/status.go:170-182`), único cálculo de base, usa la base efectiva.
5. **Registro e instantánea.**
   - Se añaden campos `omitempty` en `RecordRepo` (`internal/branch/types.go`, `registry/registry.go`, `engine/ensure_registry.go`) y en `GitSnapshotRepo` (`internal/kickoff/gitsnapshot.go`, `engine/snapshot.go`).
   - `Validate` admite el vocabulario cerrado.
6. **Status** (`engine/status.go` `describeRepository`, `internal/branch/result.go`).
   - En modo `branch`, se comprueba con referencias locales si la rama del cambio ya está en la integración, por ascendencia o árbol equivalente.
   - Si lo está, se añade la siguiente PR y `prTarget` pasa a `main_branch`. Si la comprobación no concluye, no se finge certeza: se mantiene la integración con un aviso.
7. **Done** (`engine/done.go`, `done_refusal.go`, `merge.go`).
   - `doneBranches` y `mergeEvidence` reciben el destino de confirmación según el modo, en lugar de `repo.IntegrationBranch`.
   - El `Switch` de `in_place` sigue yendo a la integración.
8. **CLI** (`internal/cli/branch.go`).
   - Flag `--promotion`, validado como `--mode`: presente pero vacío es un error y `ask` no es una respuesta.
   - Se actualizan la ayuda y `ensureInvocation`.
9. **Scripts.** `done` toma la integración del `status` que ya capturan los dos scripts y cae a `main` si no lo puede leer.
10. **Spec y documentación**, en el mismo PR.

La tolerancia a squash ya existe (capa 2 + `--force`, `done_layers_test.go:92`). El trabajo consiste en cambiar el destino contra el que se evalúa y cubrir las dos etapas con tests.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/workspace/gitpolicy.go`, `loader.go` | Modificado | Claves `main_branch` y `promotion`, validación y defectos |
| `internal/branch/policy.go` | Modificado | `MainBranch` y `Promotion` con herencia |
| `internal/branch/planner.go`, `planner_snapshot.go` | Modificado | Modo resuelto en la entrada y base efectiva en `RepoPlan` |
| `internal/branch/types.go`, `result.go`, `labels.go`, `ensure_refusal.go` | Modificado | Registro, resultado de `status`, etiquetas, `decision_required` y conflicto de promoción |
| `internal/branch/registry/registry.go` | Modificado | Campos del modo y su origen, y su validación |
| `internal/branch/engine/` (`prefs.go`, `ensure.go`, `ensure_registry.go`, `status.go`, `done.go`, `done_refusal.go`, `snapshot.go`) | Modificado | Resolución, base, siguiente PR, destino de `done` y copia en la instantánea |
| `internal/kickoff/gitsnapshot.go` | Modificado | Campo opcional del modo por repositorio |
| `internal/cli/branch.go`, ayuda de `ensure`, `ensure_exception.go` | Modificado | Flag `--promotion` |
| `scripts/axiom-worktree.ps1`, `scripts/axiom-worktree.sh` | Modificado | `done` con la integración resuelta |
| `axiom.example.yaml`, `docs/manual-de-inicio.md` | Modificado | Claves nuevas y aviso de compatibilidad |
| `internal/assets/skills/sdd-apply`, `sdd-archive`, `_shared/sdd-status-contract.md` | Modificado si procede | Solo las afirmaciones que dejen de ser ciertas en `branch` |
| `openspec/specs/change-branch-lifecycle`, `workspace-topology`, `sdd-lane-kickoff` | Delta | Requisitos nuevos desde REQ-25.46 y modificaciones listadas |
| Tests de `workspace`, `branch`, `engine`, `cli`, `registry`, `kickoff` y scripts | Modificado y nuevo | Cobertura por capa, con squash real en dos etapas |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| Squash en modo `branch`: la rama entra dos veces y deja de ser ascendiente del destino. La capa 2 (git ≥ 2.38) puede dar falsos negativos. | Media | `--force` como salida (ya existe) y tests con squash real en las dos etapas. `status` no finge certeza si la comprobación no concluye. |
| `pr_merged` (capa 3) acredita «la PR del cambio» en `gates.yaml` y podría confirmar la fusión en `main_branch` con la evidencia de la primera PR. | Media | El diseño debe garantizar que la evidencia identifica el destino `main_branch`. Si no puede distinguirlo, la capa 3 no se usa para ese destino. Se devuelve al orquestador si exige decisión de producto. |
| Compatibilidad con equipos de versiones mixtas: un binario anterior rechaza `git.main_branch` y `git.promotion` (`config_invalid`). | Media | Documentado en el manual y aceptado (decisión 8). Sin las claves no cambia nada. |
| El registro es local y derivado: se pierde al borrarlo y no viaja a otra máquina. En ODD y en los repositorios de código de multirepo, `ask` vuelve a preguntar. | Media | La instantánea cubre monorepo SDD. Un modo distinto con la rama ya creada se rechaza con `plan_conflict` cuando hay registro. Riesgo residual documentado. |
| Dos preguntas encadenadas en `ensure` (modo de worktree y promoción) y varios repositorios con `ask`. | Baja | Preguntas de negocio distintas y secuenciales, una por repositorio con `Details.repository`. Cada invocación conserva las respuestas previas. |
| Textos y contenidos fijados por tests (`gitpolicy_test.go`, `docs_contract_test.go`, `dogfooding_branch_status_test.go`, `scripts_axiom_worktree_test.go`). | Alta | Añadir texto sin reformular las líneas fijadas y actualizar los casos que enumeran claves admitidas. |
| `PublicationTarget`, `HandOff` y `livespec` siguen con el destino de la primera etapa. | Baja | Decisión explícita (5): el backlog solo exige `status`. Se anota en la spec para no prometer más. |
| El script `.sh` parsea el JSON por columnas con `awk`. | Baja | Campos nuevos con nombres de clave distintos de los parseados (`name`, `kind`, `worktree`, `prTarget`) y tests de los scripts. |
| Tamaño: unas 700-1000 líneas o más, por encima de 400. | Alta | Entrega `single-pr` con la etiqueta `size:exception`, según la preferencia del usuario. |

## Plan de rollback

- Revertir la única PR (`git revert` del merge) devuelve el comportamiento actual. No hay migración de datos.
- Los campos nuevos del registro, la instantánea y el JSON de `status` son `omitempty` y aditivos. `registry.Load` ignora campos desconocidos y la instantánea es aditiva para binarios anteriores. Tras el revert, los registros y los `kickoff.yaml` escritos con el campo siguen siendo legibles.
- Lo único que rompe al volver atrás es un `axiom.yaml` que declare `git.main_branch` o `promotion`. Hay que retirar esas claves antes o junto con el revert. El manual lo documenta.
- Las ramas creadas en modo `branch` desde `main_branch` no se tocan. Tras el revert, `done` volvería a confirmar contra la integración. La persona decide si borrarlas a mano.

## Dependencias

- No depende de ningún cambio. Bloquea C-02-b y C-06-d y es dependencia blanda de C-05-c, C-07-d y C-07-f.
- La capa de árbol equivalente requiere git ≥ 2.38, igual que hoy.
- `axiom` con `axiom branch` en el `PATH` para los scripts de dogfooding.

## Criterios de éxito

- [ ] Sin `main_branch` declarado, todos los tests de rama actuales pasan sin cambios.
- [ ] Con `main_branch: master`, `integration_branch: develop` y `promotion: branch`, se cumplen tres cosas:
  - `ensure` crea la rama desde `origin/master`;
  - `status --json` da `prTarget: develop` y, tras la fusión en develop, la siguiente PR con base `master`;
  - `done` no borra la rama hasta que está fusionada en master, con merge commit y con squash, este último por árbol equivalente o `--force`.
- [ ] Con `promotion: release`, la base es develop y `done` borra la rama al fusionarla en develop.
- [ ] Con `promotion: ask`, `ensure` devuelve `decision_required` con las respuestas `branch` y `release`, y repetirlo con la respuesta crea la rama en la base correcta.
- [ ] Pedir un modo distinto del registrado con la rama ya creada termina en `plan_conflict`.
- [ ] En multirepo, el repositorio de specs sigue in situ y `done` lo omite. Cada repositorio de código resuelve su propio modo.
- [ ] En monorepo SDD, la instantánea del kickoff incluye el modo resuelto. `ensure` lo respeta y no reescribe `kickoff.yaml`.
- [ ] Los scripts `axiom-worktree.ps1` y `.sh` funcionan igual con la configuración actual del repositorio, y `done` usa la integración resuelta.
- [ ] `go test ./...` en verde.

## Estimación de tamaño

L: unas 700-1000 líneas de autoría, quizá algo más con spec, documentación y tests de scripts. Entrega **single-pr** con la etiqueta `size:exception`, sin encadenar.
