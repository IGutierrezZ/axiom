# ODD: SDD multirol nativo (P0 de la prueba multirepo)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/sdd-multirol-nativo.md`.
> Espejo de recuperación en Engram: topic `odd/sdd-multirol-nativo/tasks`, proyecto `axiom`.
> Worktree: `C:\repos\axiom-wt\sdd-multirol-nativo`, rama `fix/sdd-multirol-nativo`, base `origin/main` `00e26206`. El worktree se creó con `git worktree add` y sin fetch, por indicación del usuario.

## Objetivo

Que un cambio SDD con kickoff multirol, en multirepo, avance de forma nativa (status → apply → archive) sin que el orquestador tenga que suplir pasos a mano. El caso de referencia es la prueba EMT del 2026-10-14, con roles backend y frontend, modo `continuous` y tres repos.

## Problema

Defectos P0 observados en la prueba, registrados en `HALLAZGOS_AXIOM.md` de la carpeta de la prueba. Están verificados en el código de esta base.

1. **`sdd status` ignora `tasks.<rol>.md`.**
   - `internal/sddstatus/status.go:1385` solo resuelve `tasks.md`.
   - Con varios roles, el artefacto `tasks` queda `missing`, `taskProgress` en 0, `applyState: blocked` y `nextRecommended: tasks`. El apply y el archive nativos no se alcanzan nunca (issue #153).
   - El mismo supuesto aparece en `edit_authority.go:479`, que lee el texto de la autoridad de edición, en `governance.go:207-210,403`, donde está el digest `tasks` de la puerta, y en `internal/cli/sdd_gate_digest.go:50`.
   - Consecuencia: `allowedEditRoots` solo contiene el repo de specs, y el sobre `axiom.sdd-integration.consent/v1` para los repos de código nunca se emite.
2. **Con `continuous` el estado no proyecta los roles.**
   - `governance.go:343` devuelve nil, así que no hay roster ni progreso por rol.
   - REQ-21.7 (`openspec/specs/sdd-block-review-gates/spec.md:15`) solo prohíbe las **puertas** en `continuous`. No prohíbe proyectar el roster. La decisión D-05 (`inc-21`, `design.md:239`) extendió la ausencia a todo el bloque `governance`.
3. **Subagentes de Claude Code sin las herramientas que exige su skill.**
   - `sdd-spec` y `sdd-design` no tienen Bash, pero deben ejecutar `axiom sdd integrate check` (`sdd-spec/SKILL.md:185`, `sdd-design/SKILL.md:69`).
   - `sdd-explore` no tiene Write, pero debe persistir `exploration.md` (`sdd-explore/SKILL.md:103-114`).
   - Las definiciones están en `internal/assets/claude/agents/*.md`. La prueba `TestInjectClaudeSubAgentsScopedTools` (`internal/components/sdd/inject_test.go:8036`) fija hoy la ausencia de esas herramientas.

## Por qué

- La prueba EMT solo terminó porque el orquestador sustituyó esas piezas a mano: lanzó `integrate check` él mismo, hizo el apply con una autorización escrita en el prompt y archivó sin el estado nativo.
- El valor diferencial de Axiom en multirepo (roles y autoridad de edición) no llegó a ejercitarse.

## Alcance

- `internal/sddstatus/`: `status.go`, `edit_authority.go`, `governance.go`, `status_v2.go` si procede, y sus pruebas.
- `internal/cli/sdd_gate_digest.go`: digest de la puerta `tasks` con varios roles.
- `internal/assets/claude/agents/sdd-{spec,design,explore}.md` y `internal/components/sdd/inject_test.go`.
- Documentación del contrato: `internal/assets/skills/_shared/sdd-status-contract.md` y la de los agentes si las nombra.

## Restricciones

- **Cambios no sellados y `fullstack`.** El estado no cambia ni un byte: el sellado `fullstack` sigue usando `tasks.md`. Las pruebas de byte-identidad existentes (`kickoff_absence_regression_test.go`, `TestSingleRepoTasksKeepStatusByteIdentical`) deben seguir en verde.
- **Puertas en `continuous`.** No se evalúa ni se presenta ninguna puerta (REQ-21.7). Proyectar el roster no puede activar la puerta de integración que bloquea el archive (`archiveDependencyFromGovernance`) ni la dependencia de `handoff.md` en verify (`verifyDependencyFromHandoff`, REQ-21.15). En `continuous` no existen ni la una ni el otro.
- **Autoridad de edición multirol.** Cada rol aplica en paralelo en su repo, así que la autoridad se evalúa sobre la unidad de trabajo actual de **cada** `tasks.<rol>.md` y se unen las raíces no autorizadas. Así un único sobre de consentimiento cubre los repos de todos los roles activos.
- **Herramientas de los subagentes.**
  - `sdd-explore` solo recibe Write, para `exploration.md`; no recibe Bash.
  - `sdd-spec` y `sdd-design` reciben Bash para `integrate check`.
  - La skill de cada uno sigue limitando lo que puede escribir.
- Los cambios en TUI y dashboard (`internal/tui/model.go:6022`, `internal/dashboard/service.go:491,506`) quedan **fuera de alcance**, como seguimiento.

## Tareas

| ID | Tarea | Ruta | Evidencia del disparador | Estado |
|---|---|---|---|---|
| T1 | Herramientas de los subagentes `sdd-spec`, `sdd-design` y `sdd-explore` | delegada | escritor único; 3 assets, una prueba y la documentación | Hecha, `7956969b` |
| T2 | `sdd status` resuelve las tareas por rol desde el kickoff sellado | delegada | más de 4 ficheros de `sddstatus` y `cli` | Hecha, `9e1b1a5d` |
| T3 | Roster y progreso por rol en `continuous`, sin puertas | delegada | `governance.go`, `status.go`, `status_v2.go` y sus pruebas | Hecha, `2477bcc7` |

- [x] **T1 · Herramientas de los subagentes.** Commit `7956969b`.
  - Añadir Bash a `sdd-spec.md` y `sdd-design.md`, y Write a `sdd-explore.md`, en `internal/assets/claude/agents/`.
  - Actualizar `TestInjectClaudeSubAgentsScopedTools`:
    - explore: con Write, sin Edit, Bash ni Task;
    - spec y design: con Bash, sin Task;
    - propose y tasks: sin cambios.
  - Revisar si alguna documentación lista las herramientas de cada agente y alinearla.
  - Anotar como seguimiento la herramienta equivalente en Kimi (`internal/assets/kimi/agents/`, `exclude_tools` con `WriteFile`).
- [x] **T2 · Tareas por rol en `sdd status`.** Commit `9e1b1a5d`.
  - Con un kickoff sellado de más de un rol (o con un rol cuyo `TasksFile` no sea `tasks.md`), `resolveArtifactPaths` resuelve los ficheros `TasksFile` de cada rol. Usar `kickoff.Load` (`internal/kickoff/seal.go:35`).
  - `artifacts.tasks`: `done` si existen todos, `partial` si falta alguno.
  - `taskProgress`: el agregado de todos los roles.
  - `applyEditAuthorityBlock`: se evalúa por rol, sobre la unidad de trabajo actual de cada fichero, y se unen las raíces.
  - `artifactBlockedReasons` nombra los ficheros por rol.
  - El digest `tasks` de `governanceArtifactInputs` y el de `cli/sdd_gate_digest.go` cubren los ficheros por rol. Así se abre `tasksGateOpen` en `checkpointed` multirol (`kickoff/machine.go:92-110`).
  - Pruebas nuevas, con un helper que selle un kickoff multirol:
    - el estado pasa a `apply` con las dos tareas;
    - el progreso es el agregado;
    - el sobre de consentimiento nombra los repos hermanos de ambos roles;
    - un rol sin fichero deja `partial`;
    - `fullstack` y no sellado no cambian ni un byte.
- [x] **T3 · Roster en `continuous`.** Commit `2477bcc7`.
  - En `continuous`, el estado proyecta `governance` con `kickoff`, `roster` y el progreso por rol, y con las puertas vacías u omitidas.
  - En `continuous` no se aplican `firstOpenGate`, `archiveDependencyFromGovernance`, `verifyDependencyFromHandoff` ni `governanceNextRecommended`. Hay que blindarlo con pruebas.
  - Actualizar las pruebas que fijan `Governance == nil` en `continuous`: `governance_test.go:201`, `status_test.go:1047` y `status_v2_test.go:154`.
  - Documentar en `sdd-status-contract.md` el bloque `governance` (roster, puertas y progreso por rol) y cuándo aparece.
  - Dejar constancia de que D-05 se acota: la ausencia se aplica a las puertas, no al roster.

## Criterios de aceptación

- Con un cambio sellado con los roles backend y frontend y ambos `tasks.<rol>.md` con tareas pendientes:
  - `axiom sdd status` informa `tasks: done`, el progreso agregado y `applyState: ready`, o bien `blocked(edit_authority_missing)` con un sobre que nombra los repos de código de ambos roles;
  - tras la concesión, `nextRecommended: apply`;
  - con todo completado, el archive está listo.
- En `continuous` aparece el roster, sin ninguna puerta y sin bloqueo del archive ni de verify por la puerta de integración o por `handoff.md`.
- `fullstack` y los cambios no sellados no cambian ni un byte.
- Los subagentes `sdd-spec` y `sdd-design` pueden ejecutar `axiom sdd integrate check`, y `sdd-explore` puede escribir `exploration.md`.

## Verificación aplicable

- `go test ./internal/sddstatus/ -count=1`. Lanza git, así que es moderadamente lenta.
- `go test ./internal/kickoff/ ./internal/components/sdd/ -run 'ScopedTools|SubAgents' -count=1`
- `go test ./internal/assets/ -count=1`
- `go test ./internal/cli/ -run 'SDDGate|SDDStatus|Kickoff' -count=1`. Las pruebas de `internal/cli` son lentas; usa siempre `-run`.
- `go vet ./...`, `go build ./...` y `bash scripts/deadcode-ratchet.sh`.
- Prueba manual: `axiom sdd status` sobre una copia del cambio archivado de la prueba EMT, devuelto a `openspec/changes/` en un directorio temporal.

## Modo de trabajo

- **TDD:** modo estándar. Fuente: `openspec/config.yaml` (`strict_tdd: false`). Runner: `go test`.
- **RDD:** desactivado (global `off`).
- **Entrega:**
  - Previsión: T1 unas 40 líneas, T2 unas 300 y T3 unas 200, en total unas 540 líneas con pruebas y documentación.
  - Supera el presupuesto de 400 líneas. La estrategia es `ask-on-risk`, y el usuario eligió la cadena `stacked-to-main` el 2026-10-14.
  - Cortes: un PR por tarea contra `main`, fusionados en orden (T1, T2, T3).
  - Push y PR los decide el usuario.

## Seguimientos fuera de alcance

- TUI y dashboard: siguen leyendo solo `tasks.md` (`internal/tui/model.go:6022`, `internal/dashboard/service.go:491,506`).
- Kimi: los agentes de fase excluyen `WriteFile`.
- Issue #153: actualizar su descripción con lo que falta (la ausencia de roster en `continuous` y `allowedEditRoots`). Es una acción externa y necesita autorización.

## Progreso

- 2026-10-14: ODD registrado tras el mapeo delegado del código (`ff3888ee`). Las referencias anteriores proceden de ese mapeo.
- 2026-10-14: T1 hecha (`7956969b`).
  - Cambios: `Bash` en `sdd-spec` y `sdd-design`, y `Write` en `sdd-explore`. El cuerpo de `sdd-explore` limita `Write` a `exploration.md`.
  - Se actualizaron `TestInjectClaudeSubAgentsScopedTools` y los golden `testdata/golden/sdd-claude-agent-sdd-{explore,spec,design}.golden`.
  - Verificación del escritor:
    - `go test ./internal/components/sdd/ -run TestInjectClaudeSubAgents`: PASS
    - `go test ./internal/assets/`: PASS
    - `go test ./internal/components/ -run Golden`: PASS
    - `go vet`: limpio
    - `go build ./...`: OK
  - Comprobación del padre: `TestInjectClaudeSubAgentsScopedTools` PASS. `axiom review assess --base-ref ff3888ee --committed-only`: riesgo `medium`. Con RDD desactivado basta la autoverificación del escritor.
  - Seguimiento: Kimi (`internal/assets/kimi/agents/sdd-{explore,spec,design}.yaml`) excluye `WriteFile`. Los `sdd-explore.md` de Kimi, Cursor y Kiro siguen diciendo «Do NOT create or modify project files».
- 2026-10-14: T2 hecha (`9e1b1a5d`, `Refs #153`). Son unas 775 líneas añadidas y 28 borradas, de las que unas 440 son pruebas; supera la previsión por la cobertura multirol y multirepo.
  - Código:
    - `kickoff.RoleTaskFiles` (en `types.go`), que comparten `status` y el digest de la CLI.
    - `internal/sddstatus/role_tasks.go`, nuevo.
    - Cambios en `status.go`, `edit_authority.go`, `governance.go`, `workspace_declarations.go` y `cli/sdd_gate_digest.go`.
  - Decisiones:
    - `allComplete` exige que no quede ninguna tarea pendiente, que exista al menos una, que cada fichero de rol tenga casillas y que no falte ningún fichero.
    - Un fichero de rol sin casillas bloquea el apply.
    - El digest `tasks` cubre todos los ficheros de rol y solo se calcula cuando existen todos; es el mismo en `status` y en la CLI.
    - La autoridad de edición se evalúa por rol, sobre la unidad actual de cada fichero, y une las raíces.
    - `fullstack` y los cambios no sellados no cambian ni un byte.
  - Verificación del escritor:
    - `go test ./internal/sddstatus/`: ok (612,8 s)
    - `go test ./internal/kickoff/`: ok
    - `go test ./internal/cli/ -run 'SDDGate|SDDStatus|Kickoff|GateDigest'`: ok
    - `go vet ./...`: limpio
    - `go build ./...`: OK
    - Ratchet de código muerto: sin funciones nuevas inalcanzables
  - Comprobación del padre:
    - `assess --base-ref 35766431 --committed-only`: riesgo `medium`.
    - Prueba real: sobre una copia del cambio EMT `tablon-anuncios-caducidad` con las tareas desmarcadas, el binario anterior da `tasks: missing`, `taskProgress` 0 y `next: tasks`. El de la rama da `tasks: done`, `pending: 42` y `blocked(edit_authority_missing)` nombrando `kvp.emt.api` y `kvp.emt.ui`. Sin marcador de instancia, pide `axiom sdd continue`, que es lo esperado. La copia se borró.
  - Seguimientos:
    - Los textos del sobre de consentimiento (`edit_authority_consent.go` y `consent_contract.go`) siguen diciendo «tasks.md».
    - La pista de fase «Write tasks.md…» no ha cambiado.
    - `blockerReasons.finalize` oculta las razones de planificación mientras la siguiente fase es `tasks`. Es un comportamiento previo.
- 2026-10-14: T3 hecha (`2477bcc7`, `Refs #153`). Son 295 líneas añadidas y 65 borradas.
  - Código:
    - Nuevo `governanceGated`: el gobierno existe y el modo no es `continuous`.
    - En `continuous`, `loadGovernance` devuelve el kickoff, el roster y `gates: []`, sin leer el registro de puertas.
    - Quedan protegidos `firstOpenGate` (que cubre `gateQuestion`, `await-gate` y el bloqueo por puerta), `resolveNextRecommended`, `artifactBlockedReasonsForRoleTasks`, `archiveDependencyFromGovernance` (sin puerta de integración) y `verifyDependencyFromHandoff` (sin `handoff.md`).
    - La v2 lleva `governance` con el roster y `gates: []`; el contrato congelado lo admite.
    - Se acota D-05: REQ-21.7 solo prohíbe las puertas. La nota está en el comentario del código.
    - `sdd-status-contract.md` documenta el bloque `governance`.
  - Decisión: no se añade progreso por rol. `taskProgress` ya agrega todos los roles y `artifactPaths.tasks` sigue el orden del roster; queda como seguimiento opcional.
  - Pruebas:
    - Nuevas en `governance_continuous_test.go` (5).
    - Renombradas al nuevo invariante: las de `governance_test.go`, `status_test.go` y `status_v2_test.go`, y la subprueba de `declarations_archive_block_test.go`.
  - Verificación del escritor:
    - `go test ./internal/sddstatus/`: ok (414 s)
    - `go test ./internal/kickoff/`: ok
    - `go test ./internal/assets/`: ok
    - `go test ./internal/cli/ -run 'SDDGate|SDDStatus|Kickoff|GateDigest|Handoff'`: ok
    - `go vet ./...`: limpio
    - `go build ./...`: OK
    - Ratchet de código muerto: OK
    - `gofmt`: limpio
  - Comprobación del padre:
    - `assess --base-ref d9aa7f3b --committed-only`: riesgo `medium`.
    - Prueba real con la copia del cambio EMT completado: roster `[backend, frontend]`, `gates: []`, sin `gateQuestion`, `tasks: done` con 42/42, `applyState: all_done`, `next: archive`, `archive: ready` y sin razones de bloqueo. La copia se borró.

## Cortes de entrega (`stacked-to-main`)

| PR | Commits en la rama | Commit en `main` | Contenido |
|---|---|---|---|
| [IGutierrezZ/axiom#154](https://github.com/IGutierrezZ/axiom/pull/154) | `ff3888ee`, `7956969b`, `35766431` | `2d3918bc` | Registro del ODD y T1 |
| [IGutierrezZ/axiom#155](https://github.com/IGutierrezZ/axiom/pull/155) | `cb7fef96`, `3861040f` (rebase de `9e1b1a5d` y `d9aa7f3b`) | `ba7dde1e` | T2, con `size:exception` (834 líneas, unas 440 de pruebas) |
| [IGutierrezZ/axiom#156](https://github.com/IGutierrezZ/axiom/pull/156) | `2477bcc7`, `e863a91a` y las fusiones de sincronización `7c7b99ec` y `b1257e7a` | `3ebb9f3f` | T3 |

Los tres PR se fusionaron con squash el 2026-10-14, con la CI en verde.

Incidencias de la entrega:
- Tras fusionar el #154 con squash, el #155 se rebasó sobre `main` y se subió con `--force-with-lease`.
- Con el autofix ya activo, el #156 se sincronizó fusionando su base (primero la rama del #155 y después `main`), sin rebase.
- En ambos casos se comprobó que el árbol resultante era idéntico byte a byte al ya verificado y que el diff frente a la base era solo T3.

## Estado

**Cerrado.** La issue #153 se actualiza y se cierra con referencia a #155 y #156.

## Siguiente paso

Repetir el escenario (c) de la prueba EMT con el binario reinstalado, en modo `checkpointed`, con el rol QA, `handoff_policy: per_checkpoint` y worktrees para los repos de código. Los seguimientos fuera de alcance siguen anotados en la sección correspondiente.
