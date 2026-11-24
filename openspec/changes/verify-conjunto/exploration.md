## Exploración: verify-conjunto (C-02-b, verificación conjunta del cambio antes de la PR a la rama principal)

### Estado actual

**Compuerta `integration`.**
- Un registro de `gates.yaml` lleva UNA sola evidencia: `GateRecord.EvidenceKind/EvidenceRef/EvidenceBaseRef/Verified` (`internal/kickoff/types.go:183-207`). `GateRecordArgs` solo admite un `--commit` y un `--base-ref` (`internal/kickoff/args.go:395-420`, parseo `:505-565`).
- `VerifyIntegrationEvidence` (`internal/kickoff/integration.go:44`) solo comprueba, para `pr_merged`, que `Ref` sea ancestro de `BaseRef`; `deployment` y `attestation` quedan `Verified:false`.
- El adaptador (`internal/cli/sdd_gate_evidence.go:42-47`) construye el comprobador con `SnapshotBuilder{Repo: workspaceRoot}`: el ancestro se mira SIEMPRE en el repositorio del planning home (monorepo: el propio; multirepo: el de specs). `completeGateBaseRef` (`internal/cli/sdd_gate_base_ref.go:14-18`) rellena el `--base-ref` con `engine.PlanningIntegration(loc)` (`internal/branch/engine/integration.go:19-25`), es decir `git.integration_branch` del planning home, nunca `main_branch`.
- Orden en `runSDDGateRecord` (`internal/cli/sdd_gate.go:64,132-151`): la evidencia se verifica antes de `AppendGate`; solo si `parsed.EvidenceKind != ""`.
- **Aprobar `integration` NO exige evidencia ni informe de verify en el código**: `GateRecord.Validate` (`internal/kickoff/schema.go:194-208`) solo valida el vocabulario, y nada en `sdd_gate.go` rechaza una aprobación sin `--evidence-kind`. La exigencia vive solo en prosa (spec REQ-21.16, `sdd-archive/SKILL.md:76`) y en el mensaje de `archiveDependencyFromGovernance` (`internal/sddstatus/governance.go:568-581`), que solo comprueba `gate.Status == "approved"`.
- El estado de una compuerta es el de su último registro (`internal/kickoff/machine.go:56,143-150`); varias aprobaciones sucesivas no se acumulan, la primera ya la abre.
- La compuerta solo existe con kickoff sellado y no `continuous` (`machine.go:52-54`, `governanceGated`). Un cambio `continuous` (como este) no tiene compuerta `integration`.
- `--hand-off` está prohibido en `integration` (`args.go:578`). Criterios de la pregunta tipada: `internal/sddstatus/governance.go:211-214` ("¿Hay evidencia de integración o despliegue?", "¿De qué clase y verificable?"). `gateOwningPhase` enruta un rechazo de `integration` a `archive` (`internal/sddstatus/status.go:1864`).
- Consumidores de la evidencia verificada: `done` capa 3 (`internal/branch/engine/done.go:488-517`, exige `Verified` y que no sea modo `branch` contra master, REQ-25.x en `change-branch-lifecycle/spec.md:1839`) y `sdd_trace.go:177`.

**Base del modo por repositorio (lo que dejó C-02-a).**
- `RepoPlan.Base()` y `TwoStage()` (`internal/branch/planner.go:169-180`): `BaseBranch` = `MainBranch` en `branch`, `IntegrationBranch` en `release`/una etapa. `GitSnapshotRepo.Base/Promotion/Branches` ya están congeladas en `kickoff.yaml` (`internal/kickoff/gitsnapshot.go:47-70`); `ChangeBranch` en `:36`.
- `axiom branch status --json` ya proyecta por repositorio `base`, y por rama `worktree`, `ahead`, `behind` y `comparedTo` (`internal/branch/engine/status.go:265-312`, `comparisonFor` `:334-342`): rama de rol contra su rama de seguimiento; rama del cambio contra la base del repositorio (la del modo, o `origin/<base>` si existe). Es la fuente que ya usa el verify por rol.
- `branch.MergeDestination`/`PromotionStage` (`internal/branch/promotion.go:266,292`) deciden el destino de la fusión y la capa `pr_merged`; en modo `branch` `pr_merged` NO acredita la fusión en `main_branch` (spec `change-branch-lifecycle:1839`).

**Skill `sdd-verify`.**
- `SKILL.md:41` ("When a role is supplied, inspect only that role's diff") y `:62` (fila de la tabla de decisión). Solo hay un diff definido para ROL: `references/technical-compliance.md:5-17` (`axiom branch status --json` → `worktree` y `comparedTo` → `git merge-base <comparedTo> HEAD` → `git diff <base>` + untracked, excluyendo la carpeta del cambio).
- Sin rol, la skill no define ningún diff: "Inspect available artifacts and implementation" (`SKILL.md:32`). No existe hoy «unión por repositorio frente a merge-base con la base del modo». Tampoco hay instrucción de fetch ni de qué hacer si falta una rama (hand-off entre máquinas).
- La skill es solo prosa: no hay código Go que calcule ese diff. `branch/gitcli` no expone `merge-base` ni `diff` (solo `IsAncestor`, `ops.go:369-380`).

**Verify global multirrol ya existe.** REQ-21.15 (`openspec/specs/sdd-role-closure-handoff/spec.md:95-132`): verify global habilitado con todos los `role-apply` blocking aprobados, informe consolidado `verify-report.md` distinto de `verify-report.<rol>.md`; `verifyDependencyFromRoleGates` (`internal/sddstatus/governance_handoff.go:271-298`). REQ-28.4 (`sdd-independent-review/spec.md:174-177`) lo declara explícitamente «sin cambios» y con el rol único `fullstack` el verify del rol y del cambio son el mismo informe. El paso 2 del plan ("lanzar sdd-verify sin rol") ya existe como concepto; lo que falta es su ALCANCE y su vínculo con la compuerta.

**Orquestadores.** El texto de la compuerta vive en un solo sitio editable: `internal/assets/skills/_shared/sdd-orchestrator-sections.md:81` («The `integration` gate keeps its two exits: approve with the merge evidence it requires, or reject») inyectado en los 12 `internal/assets/<agente>/sdd-orchestrator.md` por `internal/components/sdd/orchestrator.go` (los 5 ficheros de agente que contienen «integration» no hablan de la compuerta). `claude/sdd-orchestrator-workflow.md` no menciona la compuerta; `:180` es otra cosa. Otros textos relacionados: `sdd-archive/SKILL.md:76,95-102`, `_shared/sdd-status-contract.md:187-188`. Tests que fijan texto: `internal/assets/assets_test.go`, `sdd_iso_pins_test.go`, `independent_review_boundary_test.go`, `internal/components/sdd/orchestrator_shared_sections_test.go`.

**Spec viva `openspec/specs/sdd-block-review-gates/spec.md`.** Último REQ: REQ-43.23 (`:725`); numeración mezclada (21.x, 26.x, 27.12, 28.10, 43.22-23). La puerta `integration` solo aparece en REQ-26.1 (`:229-251`, «aprobar con la evidencia de fusión exigida, o rechazar»). La precondición de evidencia y su regla multirepo están en OTRA capacidad: `sdd-archive-lifecycle` REQ-21.16 (`spec.md:15-37`). REQ-24.12 de `planning-home-resolution` (`:408-421`) fija que `pr_merged` se verifica contra el repositorio de specs.

**Tests/goldens.** `internal/kickoff/integration_test.go`, `schema_test.go:184-189`, `internal/cli/sdd_gate_planning_home_test.go:62-163`, `internal/sddstatus/status_test.go:1129,1200`, `governance_*_test.go`, `internal/branch/engine/done_layers_test.go:165`. `testdata/golden/` NO contiene nada de la compuerta `integration` ni de `pr_merged` (búsqueda sin coincidencias): el riesgo de goldens del plan para C-02-b es nulo salvo que se cambie texto proyectado a goldens de orquestador.

### Contradicciones entre el plan y `main`

1. **«Hay que hacerlo por cada repositorio» no es viable con la compuerta actual y contradice REQ-21.16.** `pr_merged` se verifica solo contra el grafo del repositorio de specs; la integración de un repositorio de código se acredita con `deployment`/`attestation` (declaradas, no verificables). `--repo` para `pr_merged` fue diferido explícitamente (`planning-home-resolution/spec.md:46`). Además, un registro lleva una sola evidencia y el último registro manda.
2. **`args.go:351-385` está obsoleto**: hoy son `ApplyTopology`/`ToKickoff`; los tipos de evidencia están en `types.go:43-74,183-207` y `args.go:395-420,505-565`.
3. **El criterio «la compuerta no se aprueba hasta que…» hoy no se cumple por código**: no hay ningún bloqueo en `gate record` ni en `Validate`; sería comportamiento nuevo, no ajuste.
4. **`PlanningIntegration` ≠ base del modo.** El plan cita ambas como la misma base; en modo `branch` la integración es develop y la base es master (`RepoPlan.Base()`).
5. **El verify sin rol ya existe** (REQ-21.15/REQ-28.4); el cambio real es acotar su diff y exigirlo ante la compuerta.
6. **El plan lo ubica en la ola 6, tras C-02-c y C-02-d.** Se ejecuta antes que ellas; C-02-d (PR #289) sigue sin fusionar. Pendiente confirmar si C-02-c ya está en `main` (`git log` solo muestra C-02-a y otros).

### Colisión con C-02-d (PR #289)

- Duros: `internal/assets/skills/_shared/sdd-orchestrator-sections.md` (misma zona del párrafo de paradas/compuertas, `:81-82`), los 12 `sdd-orchestrator.md` si se toca algo fuera del bloque compartido, y `internal/kickoff/args.go` (C-02-d añade `commit_review` al `GateRecord` de `tasks`; C-02-b tocaría `GateRecordArgs`/`GateRecord` si añade `--repo` o `--verify-ref`, y `types.go`).
- Sin solape previsto: `sdd-verify/SKILL.md` (C-02-d no lo toca), `sdd-archive/SKILL.md` (lo toca C-02-c/C-03-b, no C-02-d), `work-unit-commits`, `sdd-apply`.
- Mitigación: poner el texto nuevo de la compuerta en un párrafo propio, sin reflujo de `:81`, y rebasar sobre C-02-d antes de la PR.

### Zonas afectadas

- `internal/assets/skills/sdd-verify/SKILL.md` y `references/` (nueva `joint-verification.md`, análoga a `technical-compliance.md`); registrar en `assets_test.go:369-370`.
- `internal/assets/skills/_shared/sdd-orchestrator-sections.md:81`; `sdd-archive/SKILL.md:76,102`; `_shared/sdd-status-contract.md:187-188`.
- Según enfoque: `internal/cli/sdd_gate.go`, `sdd_gate_evidence.go`, `sdd_gate_base_ref.go`, `internal/kickoff/{args,types,integration}.go`, `internal/branch/engine/{integration.go,status.go}`, `internal/branch/gitcli/ops.go`, `internal/sddstatus/governance*.go`.
- Specs: `sdd-block-review-gates` (ADDED REQ-44.x o siguiente libre; MODIFIED REQ-26.1 si cambia el texto de la compuerta), `sdd-archive-lifecycle` (MODIFIED REQ-21.16 si cambia la regla multirepo), `sdd-independent-review` (MODIFIED REQ-28.4, que declara el global «sin cambios»).

### Enfoques

| # | Enfoque | Pros | Contras | Esfuerzo |
|---|---|---|---|---|
| A | Solo prosa: nueva referencia `joint-verification.md` en `sdd-verify` (diff por repo con `branch status --json`, merge-base con `comparedTo`, `git diff` + untracked, excluye carpeta del cambio) y texto de la compuerta en orquestadores | Sin código; poca colisión con C-02-d; sigue el patrón del verify por rol | El «test con un commit ajeno en develop» no puede fijar el diff (es prosa); la compuerta sigue sin bloquear; «no degradar en silencio» no es comprobable | Bajo |
| B | A + comando nativo de solo lectura (p. ej. `axiom branch scope --change <c> --json`, o ampliar `branch status` con `mergeBase`, rango y ficheros por rama) del que la skill consume el diff | El criterio del plan es comprobable con un test git real (commit ajeno en develop no entra, también con squash/merge); reporta rama ausente/sin fetch de forma tipada; reutiliza `RepoPlan.Base()`/`comparisonFor` | Código nuevo en `branch/engine` y `gitcli` (hoy sin `merge-base`/`diff`); contrato JSON nuevo; sigue sin bloquear la compuerta | Medio |
| C | B + bloqueo en `gate record --gate integration --decision approved`: exige informe de verify sin rol y evidencia por repositorio | Cumple literal «no se aprueba hasta que…» | Toca `GateRecord`/`args.go` (choca con C-02-d), contradice REQ-21.16 para repos de código (hay que decidir qué es «pr_merged por repositorio»: `--repo`, diferido), cambia el contrato `gate-record/v1` (REQ-43.23), puede romper tests de `sdd_gate_planning_home_test.go`; no aplica en `continuous` | Alto |

### Recomendación

Enfoque **B**, con una versión mínima del bloqueo de C opcional tras decidir las aperturas 1 y 2. B da lo que el plan marca como criterio verificable (diff exacto = unión por repositorio, con test de commit ajeno en develop) sin reabrir el modelo de evidencia. Concretamente:
- Calcular el alcance en Go de solo lectura (merge-base del tip de la rama del cambio con la base del modo del repositorio, rango `mergeBase..tip`, ficheros), a partir de `RepoPlan` y la instantánea sellada; sin `--fetch`; rama no presente localmente = resultado tipado «no verificable: falta fetch autorizado», nunca omisión silenciosa.
- La skill `sdd-verify` sin rol consume ese JSON; el informe es el `verify-report.md` consolidado (REQ-21.15) y lista por repositorio rama, base, merge-base y rango.
- Dejar el bloqueo de la compuerta (C) para una decisión explícita, porque exige tocar el modelo de evidencia.

### Riesgos

- **Merge-base móvil**: tras fusionar la rama en su base (merge commit, o modo `release` donde la base es develop) `git merge-base <base> <rama>` pasa a ser la punta de la rama y el diff queda vacío. El verify conjunto debe correr antes de la fusión en la base, o fijar la bifurcación de otra forma. En `release`, el paso 1 del plan («pr_merged en la integración») choca con ese orden. Con squash, la ascendencia no se rompe, pero con merge commit sí.
- **Hand-off entre máquinas**: ramas de otros roles/repositorios solo en `origin`; sin `--fetch` autorizado no hay verify conjunto y debe decirlo (plan, riesgos). `status` prefiere `origin/<base>` si existe, así que la base puede ser remota y la rama local no.
- **Diff con árbol sucio**: el verify por rol incluye staged/unstaged/untracked; la unión «de las ramas» debería ser solo commits (rango) o decidir si incluye el árbol de trabajo.
- Choque textual con C-02-d en `sdd-orchestrator-sections.md` y `args.go`; ejecución fuera del orden de olas del plan.
- Cambio `continuous` y ODD: sin compuerta; el verify conjunto sería solo opcional.
- Regla de unidad de verdad (REQ-28.4 «sin cambios» del global): modificarlo requiere MODIFIED explícito.

### Decisiones abiertas (sin resolver)

1. ¿«`pr_merged` verificado por rama/repositorio» se mantiene dado REQ-21.16 (código ⇒ `deployment`/`attestation`)? Opciones: añadir `--repo` a `pr_merged` (diferido en REQ-24.12), aceptar evidencia declarada por repo de código, o limitar C-02-b al diff y dejar la evidencia como está.
2. ¿La compuerta `integration` pasa a bloquear mecánicamente por falta de verify global/evidencia (hoy no bloquea) o basta con instrucción de orquestador y estado `sddstatus`? Afecta al contrato `gate-record/v1` (REQ-43.23) y choca con C-02-d.
3. En modo `release` ¿cuándo se ejecuta el verify conjunto respecto de la fusión en develop (el plan lo sitúa «al fusionar en develop», pero el paso 1 exige la fusión previa)? ¿Cómo se fija el merge-base tras la fusión?
4. ¿El diff incluye el árbol de trabajo sin commitear y los ficheros sin seguimiento (como el verify por rol) o solo commits de la rama?
5. ¿El alcance se implementa como comando nativo nuevo (`axiom branch scope`) o como ampliación de `branch status`? Y ¿el alcance exacto de «excluir la carpeta del cambio»?
6. En `continuous`/monorepo/ODD ¿el verify conjunto se invoca solo a petición (REQ-28.4 «mismo informe» con `fullstack`)?
7. Ola/orden: confirmar si C-02-c ya está fusionada y si se acepta rebasar sobre C-02-d (PR #289) antes de abrir la PR.
8. Numeración del REQ nuevo: el último es REQ-43.23 pero la capacidad mezcla 21.x/26.x/27.x/28.x/43.x; decidir si el delta usa 21.x (compuerta/archive), 43.24 o una numeración propia.

### ¿Listo para propuesta?
Sí, con las decisiones abiertas 1-4 resueltas o asumidas explícitamente en la propuesta (recomendado preguntarlas al usuario: 1, 2 y 3 cambian el contrato).
