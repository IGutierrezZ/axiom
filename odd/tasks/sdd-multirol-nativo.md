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
| T2 | `sdd status` resuelve las tareas por rol desde el kickoff sellado | delegada | más de 4 ficheros de `sddstatus` y `cli` | Pendiente |
| T3 | Roster y progreso por rol en `continuous`, sin puertas | delegada | `governance.go`, `status.go`, `status_v2.go` y sus pruebas | Pendiente |

- [x] **T1 · Herramientas de los subagentes.** Commit `7956969b`.
  - Añadir Bash a `sdd-spec.md` y `sdd-design.md`, y Write a `sdd-explore.md`, en `internal/assets/claude/agents/`.
  - Actualizar `TestInjectClaudeSubAgentsScopedTools`:
    - explore: con Write, sin Edit, Bash ni Task;
    - spec y design: con Bash, sin Task;
    - propose y tasks: sin cambios.
  - Revisar si alguna documentación lista las herramientas de cada agente y alinearla.
  - Anotar como seguimiento la herramienta equivalente en Kimi (`internal/assets/kimi/agents/`, `exclude_tools` con `WriteFile`).
- [ ] **T2 · Tareas por rol en `sdd status`.**
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
- [ ] **T3 · Roster en `continuous`.**
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

## Siguiente paso

T2.
