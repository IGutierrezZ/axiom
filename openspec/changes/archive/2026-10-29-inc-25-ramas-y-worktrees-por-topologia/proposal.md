# Propuesta: Ramas y worktrees por topología (inc-25-ramas-y-worktrees-por-topologia)

> Fase `propose` · rama `inc/inc-25-ramas-y-worktrees-por-topologia` · 2026-10-14 · entrada: `exploration.md` (Engram `sdd/inc-25-ramas-y-worktrees-por-topologia/explore`, #634).
> Kickoff sellado: `checkpointed`, `handoff_policy: per_checkpoint`, rol único `fullstack` (`blocking`).
> Decisiones de producto cerradas con el usuario: D1–D10 (Engram #619), D11–D13 (#637) y D14 (#663). Cuando una decisión posterior refina otra anterior, manda la posterior: D12 sustituye el modelo de ramas de D4; D13 sustituye el nombre del verbo, el comprobador de CI de D9-C y el hook de D10; D11 acota el invariante de D5. Esta propuesta no reabre ninguna; los huecos reales están en §9.

## 0. Resumen

Axiom no tiene una política de ramas ni de worktrees: la doctrina dice "crea una rama si estás en la rama por defecto", el worktree es opcional y llega tarde (en apply), las specs se escriben "siempre en main" y no existe configuración ni guarda mecánica. inc-25 convierte esa doctrina en un **motor determinista de la CLI** (`axiom branch ensure|status|done|check|guard`) que crea la rama del cambio, y opcionalmente su worktree, **al nombrar el cambio y antes de cualquier escritura en su carpeta**, según la topología y el número de roles que tocan cada repo. Cuatro capas de guarda (escritores Go, `sdd status`, comprobación de CI y hook opcional) impiden trabajar en la rama equivocada. De paso, se retira `monorepo-decoupled`.

Principio rector: **las ramas se crean siempre; los worktrees son una preferencia personal.** La política del proyecto (rama de integración, prefijo, repos) se versiona en `axiom.yaml`; cómo trabaja cada persona en su máquina (worktree, in situ o preguntar, y la raíz de los worktrees) vive en el hub personal.

## 1. Propósito (Intent)

### 1.1 Problema

| Hueco | Evidencia |
|---|---|
| La doctrina ODD ordena crear la rama en el cwd, no en un worktree, y no dice cuándo ni con qué nombre. | `internal/components/agentguidance/routing.go:62` ("branch first when on the default branch") |
| En SDD el worktree es opcional, se decide con un cuestionario en apply y usa una ruta propia (`../wt-{change}-{rol}`), distinta de la del script local. Proposal, spec, design y tasks ya se han escrito antes, en la rama en la que estuviera el cwd. | `internal/assets/skills/sdd-apply/SKILL.md:118-132` (Paso 2c), `:270-280` y `:301-303` (ODD-5.x) |
| "Las specs siempre en main" (ODD-5.1) contradice el monorepo: los artefactos del cambio en curso acaban en la rama de integración y se mezclan con el trabajo de otros. | `internal/assets/skills/sdd-spec/SKILL.md:248,269`; `internal/assets/skills/sdd-design/SKILL.md:192,208` |
| Incidente real: sesiones de agente hicieron checkout de ramas en el checkout principal de este repo (tres veces entre el 2026-09-29 y el 2026-10-12, una de ellas con un commit). La regla de worktree solo existe en `AGENTS.md`/`GEMINI.md`, que el runtime afectado no lee, y no hay guarda mecánica. | Engram #615; reflog del checkout principal |
| No hay configuración: la rama de integración es el literal `main` y no existen claves para prefijo ni para repos. | `internal/kickoff/args.go:464-466`; `internal/dashboard/service.go:1643`; `internal/workspace/types.go:56-62` |
| `monorepo-decoupled` no se comporta de forma distinta a `multirepo` y ya no se ofrece en ninguna superficie de usuario, pero sigue aceptándose y documentándose. | `internal/workspace/locate.go:398-407`; `internal/workspace/loader.go:125-131`; `exploration.md` §1.1 |

### 1.2 Por qué ahora

El incidente #615 demuestra que la regla en prosa no basta y que el coste de un error (rama ajena en el checkout principal, artefactos de un cambio en la rama de integración) recae en la persona. inc-24 dejó resuelta la raíz del planning home en multirepo, que es el prerrequisito para decidir en qué repo y rama vive cada artefacto. El resto de decisiones de producto ya están cerradas.

### 1.3 Resultado esperado

- Al nombrar un cambio (SDD) o una feature (ODD), una sola orden deja creada su rama y, según la preferencia personal, su worktree; devuelve en JSON rutas absolutas que todos los runtimes usan con `--cwd`.
- Ninguna escritura de Axiom en la carpeta de un cambio ocurre fuera de la rama que le corresponde.
- En la rama de integración de un repo con rama de cambio, `openspec/changes/` solo contiene `archive/` (más los sumideros exentos).
- El comportamiento es idéntico en Claude Code, Codex, OpenCode, Pi y el resto de runtimes, sin depender de primitivas propias de uno de ellos.

## 2. Decisiones consolidadas

| ID | Decisión (vigente) |
|---|---|
| D1 | Se elimina `monorepo-decoupled`, sin alias ni migración. |
| D2 | La creación de ramas y worktrees es un verbo determinista e idempotente de la CLI; las skills solo lo invocan. |
| D3 | Subcomandos sin `pr` ni `push`. |
| D5 | La rama de integración es configurable. Archive se ejecuta en la rama del cambio, antes del PR de integración. |
| D6 | `axiom.yaml` gana `git:` (`integration_branch`, `branch_prefix`) y `repositories:` de primer nivel. |
| D7 | El modo de trabajo (`worktree` por defecto, `in_place` o `ask`) y la raíz de los worktrees son personales y viven en el hub (`~/.axiom/workspaces.json`), no en `axiom.yaml`. Se piden al registrar y se editan en el dashboard. |
| D8 | `ensure` se ejecuta al nombrar el cambio, antes de escribir en su carpeta; `kickoff seal` se niega si la rama no es la que corresponde; el dashboard invoca `ensure`. |
| D9 | Guardas sobre la rama: A (escritores Go), B (`sdd status` con `blocked(wrong_branch)`), C (comprobación de CI) y D (hook opcional). Sin hooks de git. Registro compartido en `kickoff.yaml` y local en el git common dir. |
| D10 | Se prohíben `EnterWorktree` y `ExitWorktree` en los assets del producto. Mecanismo común: JSON con rutas absolutas, `--cwd`, `git -C` y rutas absolutas. El hook de la capa D lo instala `axiom sync` donde haya hooks y falla en abierto. |
| D11 | Invariante y guarda por papel del repo. Repos con rama de cambio: la integración no contiene cambios activos en `openspec/changes/`, salvo `e2e-cumulative` y `cumulative-<rol>`, exentos por nombre. Repo de specs en multirepo: sin invariante; la guarda exige estar en la rama de integración y no en una rama de rol; se trabaja in situ. Repos de código en multirepo: la rama solo se comprueba en apply. |
| D12 | Las ramas dependen de cuántos roles del cambio tocan cada repo (roles del kickoff ∩ `RoleConfig.Repositories`). Un rol (también `fullstack`): solo `<prefijo>/<cambio>`, con PR a la integración. Dos o más: `<prefijo>/<cambio>` es la rama de seguimiento y cada rol saca `<prefijo>/<cambio>-<rol>`; PR de cada rol contra la de seguimiento y PR final a integración con el archive. Se recomienda merge commit, sin imponerlo. `sdd-apply` propone la rama destino del PR. El sufijo de rol en la ruta del worktree solo aparece si hay rama de rol. La rama de seguimiento la sube la persona antes del primer PR de rol. |
| D13 | Familia `axiom branch` con los subcomandos `ensure`, `status`, `done`, `check` y `guard`, de primer nivel y neutral respecto al carril. `check` es el comprobador de CI del invariante; `guard` es el hook de la capa D. |
| D14 | (a) El documento ODD vive en `<workspaceRoot>/odd/tasks/`; `ensure --lane odd` aplica la misma política (monorepo: rama del cambio; multirepo: repo de specs, in situ, en su integración). (c) Sin fetch por defecto: base `origin/<integración>` si existe en local, si no la rama local; el JSON devuelve `base.source`, `sha` y `fetched: false`; `--fetch` es explícito. (d) `done` detecta la fusión por capas: ascendencia, árbol equivalente con `git merge-tree --write-tree`, evidencia `pr_merged`; si nada la confirma, o el árbol está sucio, exige `--force`. (e) Dashboard v1: `git:` y `repositories:` en solo lectura. (f) `ensure` admite `--prefix`, `--branch` y `--base` como excepciones por cambio. |

## 3. Alcance (Scope)

### 3.1 Dentro de alcance (In Scope)

1. **Retirada de `monorepo-decoupled` (D1).** Fuera de tipos, loader, validador, localizador, specs vivas, manual y `axiom.example.yaml`. Un `axiom.yaml` con ese valor falla con un mensaje accionable que nombra la corrección (`multirepo`).
2. **Esquema `git:` y `repositories:` en `axiom.yaml` (D6).** Tipos, decodificación, validación (`axiom workspace validate`) y resolución de la raíz git de cada repo (subcarpetas, `.`, repos compartidos por varios roles, entradas no git).
3. **Preferencias personales en el hub (D7).** Modo (`worktree|in_place|ask`) y raíz de worktrees por proyecto registrado; se piden al registrar (CLI y dashboard) y se editan en el dashboard. Ningún valor personal se escribe en `axiom.yaml`.
4. **Familia `axiom branch` (D2, D3, D13, D14).**
   - `ensure`: planificador por topología y por número de roles por repo (D12); ejecutor git idempotente; JSON con rutas absolutas de cada repo, planning home y base (`ref`, `sha`, `source`, `fetched`); `ask` como resultado tipado `decision_required`, sin interacción; `--lane sdd|odd`, `--change`, `--repo`, `--prefix`, `--branch`, `--base`, `--fetch`; rechazo en `in_place` con árbol sucio.
   - `status`: estado de ramas y worktrees del cambio; atraso solo con refs locales.
   - `done`: detección de fusión por capas (D14-d), desmontaje del worktree, rechazo con árbol sucio salvo `--force` y aviso si queda un linaje RDD abierto.
   - `check`: comprobador de CI del invariante por papel del repo (D11).
   - `guard`: hook de la capa D, casi gratuito, fail-open e inerte fuera de proyectos Axiom.
5. **Modelo de ramas por número de roles por repo (D12)** en monorepo y multirepo, incluido el sufijo de rol en la ruta solo cuando hay rama de rol.
6. **Rama de integración configurable y archive en la rama del cambio (D5).** Sustitución de los literales `main` del motor y de la prosa de skills por la integración resuelta.
7. **Momento de `ensure` (D8).** Al nombrar el cambio y antes de cualquier escritura en su carpeta, en SDD y en ODD; `kickoff seal` se niega fuera de la rama que corresponde al papel del repo.
8. **Guardas A/B/C/D (D9, D11).** A: guarda común en los escritores Go de la carpeta del cambio (`exploration.md` §1.5). B: `blocked(wrong_branch)` en `sdd status`, segundo motivo tipado. C: `axiom branch check`. D: `axiom branch guard` instalado por `axiom sync` en Claude Code, Codex y OpenCode. Alcance de cada guarda según el papel del repo (D11).
9. **Registro.** Compartido en `kickoff.yaml` (instantánea de entradas de política: integración, prefijo, base por repo y nombres de rama derivados) y local en el git common dir (rutas absolutas de todos los repos y respuesta a `ask`), resuelto con `git rev-parse --git-common-dir`.
10. **Paridad entre runtimes (D10).** Prohibición de `EnterWorktree`/`ExitWorktree` en los assets del producto, con un punto único en la doctrina generada (`routing.go`); uso de `--cwd`, `git -C` y rutas absolutas en skills y orquestadores.
11. **Autoridad de edición.** Las raíces registradas por `ensure` se proyectan en `allowedEditRoots` y las rutas de `tasks.md` se resuelven respecto al worktree que corresponda (R5).
12. **Dashboard y TUI.** Registro con preferencias, edición de preferencias personales, `git:` y `repositories:` en solo lectura, `ensure` al crear un cambio y listado de los cambios que viven en worktrees registrados (R6).
13. **Skills y doctrina.** `sdd-apply` (sustitución del Paso 2c y de ODD-5.x), `sdd-spec`, `sdd-design`, `sdd-propose`, `sdd-explore`, `sdd-archive`, `_shared` (sección G y contrato de status), orquestadores por runtime y paso ODD de `routing.go`.
14. **Dogfooding en este repo.** `git:` declarado; alineación o retirada de `scripts/axiom-worktree.ps1`/`.sh` y de `scripts/claude-hooks/guard-primary-checkout.ps1`; actualización de `CLAUDE.md`, `AGENTS.md` y `GEMINI.md`; transición de los cambios activos en `main` (`e2e-cumulative` queda exento por nombre; `inc-23-knowledge-profile`, ver Q5).

### 3.2 Fuera de alcance (Out of Scope)

- `push` y creación de PR desde la familia `axiom branch` (D3); siguen siendo decisiones de la persona.
- Hooks de git (`pre-commit`, `pre-push`...) en cualquier repo, incluidos los de terceros (D9).
- Edición de `axiom.yaml` desde la UI, incluidos `git:` y `repositories:` (D14-e). Un escritor que conserve comentarios queda como seguimiento.
- `fetch` implícito o cualquier operación remota por defecto (D14-c; contrato de autorización remota).
- Migración automática o alias de `monorepo-decoupled` (D1).
- Cobertura del hook en Pi: no hay canal demostrado; la capa D es de mejor esfuerzo.
- Escritor Go del documento ODD: ODD sigue siendo doctrina de agente, cubierta solo por doctrina y capa D (D14-a).

### 3.3 Qué no cambia

El ciclo de fases SDD, las compuertas y su ledger, handoff, RDD (los worktrees de `ensure` ya son compatibles: `exploration.md` §1.9 y §2.g), el almacén Engram y la resolución del planning home de inc-24, salvo el descubrimiento desde worktrees externos (R4).

## 4. Capacidades (Capabilities)

> Contrato con `sdd-spec`. Baseline: `openspec/specs/` (este repo no declara `baseline:`).

### 4.1 Nuevas capacidades (New Capabilities)

- `change-branch-lifecycle`: familia `axiom branch ensure|status|done|check|guard`; modelo de ramas por número de roles por repo y por topología; resolución de base sin fetch; resultado tipado `decision_required` para `ask`; detección de fusión por capas en `done`; registro compartido (`kickoff.yaml`) y local (git common dir); invariante por papel del repo y comprobador de CI; hook de la capa D fail-open; JSON con rutas absolutas como mecanismo común entre runtimes. El nombre definitivo puede ajustarse en la fase de spec.

### 4.2 Capacidades modificadas (Modified Capabilities)

- `workspace-topology`: retirada de `monorepo-decoupled` y su mensaje de error accionable; esquema `git:` y `repositories:` de primer nivel y su validación.
- `planning-home-resolution`: retirada de `monorepo-decoupled`; resolución del planning home cuando el cwd es un worktree fuera de la carpeta maestra (R4).
- `multi-project-hub`: preferencias personales (modo y raíz de worktrees) en el registro y su edición; los worktrees creados en la raíz por defecto no vuelven no registrable el proyecto.
- `local-web-dashboard`: retirada de `monorepo-decoupled`; registro con preferencias; edición de preferencias personales; `git:` en solo lectura; `ensure` al crear un cambio; listado de cambios en worktrees registrados.
- `sdd-lane-kickoff`: `kickoff seal` se niega fuera de la rama que corresponde; instantánea de política git en `kickoff.yaml`; exención de cambios preexistentes sin instantánea y de `seal --infer` (R15); `ensure --lane odd` antes de escribir `odd/tasks/`.
- `sdd-archive-lifecycle`: retirada de `monorepo-decoupled`; archive en la rama del cambio, antes del PR de integración.
- `organic-agent-trigger-rules`: el paso ODD "branch first" se sustituye por `axiom branch ensure --lane odd`; prohibición de `EnterWorktree`/`ExitWorktree`.
- `axiom-sdd-cli-integration`: `blocked(wrong_branch)` en `sdd status`; proyección de las raíces registradas en `allowedEditRoots`.
- `spec-baseline-declaration`: solo si su texto normativo cita `monorepo-decoupled` (`exploration.md` §1.1, línea 94); la fase de spec lo confirma.

## 5. Enfoque (Approach)

- **Un planificador puro y un ejecutor fino.** El planificador toma topología, `git:`, `repositories:`, roles del kickoff y preferencias personales y produce un plan determinista (repos, ramas, rutas, base). El ejecutor solo aplica el plan con `git -C`. Así la idempotencia y el modelo de D12 se prueban sin git.
- **Raíz git real de cada repo** con `git rev-parse --show-toplevel` y `--git-common-dir`, deduplicada entre roles; nunca `.git` literal (R8).
- **Rutas absolutas como contrato.** Cada orden posterior a `ensure` recibe `--cwd` explícito tomado del JSON; ninguna fase depende del cwd heredado (alineado con `sdd-phase-common.md` G).
- **Guardas con una sola consulta.** A, B y C preguntan lo mismo ("¿esta rama es la que corresponde a este cambio en este repo según su papel?"); D la reutiliza en modo fail-open. Los cambios sin instantánea en `kickoff.yaml` quedan exentos (R15).
- **Compatibilidad hacia atrás de los datos.** El loader de `axiom.yaml` y `ParseKickoff` ignoran claves desconocidas: binarios anteriores toleran `git:`, `repositories:` y la instantánea (`exploration.md` §1.1 y §1.3).
- **Sin red por defecto.** Base desde refs locales y `--fetch` explícito (D14-c).
- **Doctrina en un punto.** La prohibición de primitivas de un runtime y el paso ODD se generan desde `routing.go`; las skills solo invocan el verbo.

## 6. Áreas afectadas (Affected Areas)

| Área | Impacto | Qué cambia |
|---|---|---|
| `internal/workspace/` (`types.go`, `loader.go`, `validator.go`, `locate.go`) | Modificado | Retirada de decoupled; `git:`/`repositories:`; resolución desde worktrees |
| Paquete nuevo del motor de ramas (ubicación en design) | Nuevo | Planificador, ejecutor, registro local, detección de fusión, invariante |
| `cmd/axiom/main.go` | Modificado | Familia `axiom branch`; mensajes de topología |
| `internal/hub/` (`types.go`, `manager.go`, `detector.go`, `init.go`) | Modificado | Preferencias personales; heurística `IsGitWorktree` |
| `internal/kickoff/` (`seal.go`, `types.go`, `schema.go`, `args.go`) | Modificado | Negativa fuera de rama; instantánea; literal `main` |
| `internal/sddstatus/` (`status.go`, `edit_authority.go`, `edit_authority_consent.go`) | Modificado | `blocked(wrong_branch)`; raíces registradas en `allowedEditRoots` |
| `internal/cli/` (`sdd_kickoff.go`, `sdd_integrate.go`, `sdd_archive_compose.go`, `sdd_governance.go`) y `internal/handoff/writer.go` | Modificado | Guarda A en escritores de la carpeta del cambio |
| `internal/dashboard/` (`service.go`, `server.go`, `assets/`) e `internal/tui/model.go` | Modificado | Guarda A; preferencias; `ensure` al crear; listado de worktrees; literal `main` |
| `internal/components/sdd/inject.go` y adaptadores de runtime | Modificado | Instalación de `axiom branch guard` por `axiom sync` |
| `internal/components/agentguidance/routing.go` y tests | Modificado | Paso ODD; prohibición de `EnterWorktree`/`ExitWorktree` |
| `internal/assets/skills/sdd-{apply,spec,design,propose,explore,archive}/SKILL.md`, `_shared/`, `assets/<runtime>/sdd-orchestrator.md` | Modificado | Doctrina de ramas; retirada de ODD-5.x; `--cwd` |
| `openspec/specs/` (capacidades de §4) | Modificado y nuevo | Deltas y spec nueva |
| `docs/manual-de-inicio.md`, `axiom.example.yaml`, `axiom.yaml` de este repo | Modificado | Decoupled; `git:` |
| `scripts/axiom-worktree.ps1`/`.sh`, `scripts/claude-hooks/guard-primary-checkout.ps1`, `CLAUDE.md`, `AGENTS.md`, `GEMINI.md` | Modificado o retirado | Dogfooding |
| `testdata/golden/` | Regenerado | Goldens de assets y doctrina |

## 7. Riesgos (Risks)

IDs heredados de `exploration.md` §6.

| ID | Riesgo | Prob. | Mitigación |
|---|---|---|---|
| R4 | `workspace.Locate` asciende desde el cwd: un worktree fuera de la carpeta maestra no encuentra el marcador; `deduceMaster` y `workspace validate` dan falsos resultados. | Alta | `ensure` devuelve planning home y rutas en JSON; toda orden con `--cwd` explícito; pruebas de resolución desde worktrees externos. |
| R5 | `allowedEditRoots` = `workspaceRoot` + concesiones: los worktrees quedan fuera y darían `blocked(edit_authority_missing)`. | Alta | Proyectar las raíces registradas; resolver rutas de tareas respecto al worktree. |
| R6 | Dashboard, TUI y `sdd status` listan solo el árbol registrado; con los cambios en worktrees, los listados quedarían vacíos. | Alta | Agregar los cambios de los worktrees registrados. |
| R8 | `Repositories[].path` puede ser subcarpeta, `.` o no git; un repo con varios roles genera varias ramas. | Media | Resolver raíz git, deduplicar, tolerar no git con mensaje. |
| R9 | Conviven el estado local actual (`<GitCommonDir>/gentle-ai/`) y el nuevo directorio local. | Media | Decisión de coexistencia o migración en design (Q2). |
| R10 | Capa D de mejor esfuerzo: settings globales en Claude (corre en todo repo), confianza explícita en Codex, excepciones tragadas en OpenCode, sin canal en Pi; ODD solo cubierto por doctrina y D. | Media | Las garantías son A, B y C; `guard` inerte fuera de proyectos Axiom y fail-open. |
| R13 | `in_place` con árbol sucio mezclaría trabajo ajeno en la rama nueva. | Media | `ensure` rechaza. |
| R14 | Windows `MAX_PATH` con `<raíz>/<cambio>-<rol>/<repo>`. | Media | Validar longitud de nombres y rutas en el planificador. |
| R15 | Cambios preexistentes sin instantánea (p. ej. inc-23) bloqueados por las guardas nuevas. | Media | Exención explícita; `seal --infer` no exige rama. |
| R16 | Literales `main` residuales en código y prosa. | Media | Inventario en `exploration.md` §6 y prueba que lo fije. |
| R18 | Sin fetch, la base puede estar atrasada respecto al remoto. | Media | `base.source` y `sha` en el JSON; aviso de atraso con refs locales; `--fetch` explícito. |
| — | Entrega muy por encima del presupuesto de revisión (§8). | Alta | Decisión de entrega en `sdd-tasks`. |

Riesgo global: **alto**, por R4, R5, R6 y el tamaño.

## 8. Entrega (Review Workload Forecast)

- Previsión: **~4.900 líneas autoradas** (código, pruebas, assets, skills, specs y docs; goldens excluidos) más ~1.000 de artefactos SDD (`exploration.md` §3).
- La estrategia de la sesión es `single-pr` y el presupuesto de revisión es de 400 líneas por PR: la previsión lo supera unas doce veces.
- **Esta propuesta no decide la entrega.** La decisión entre PRs encadenados y una `size:exception` explícita para un único PR se toma en `sdd-tasks` con la guarda de carga de revisión.

```
Decision needed before apply: Yes
Chained PRs recommended: Yes
400-line budget risk: High
```

Rebanado orientativo de la exploración, solo como sugerencia (cifras sin medir; la parte de specs se reparte entre rebanadas):

| Rebanada | Contenido (§3 de `exploration.md`) | Estimación |
|---|---|---|
| S1 | Retirada de decoupled + esquema `git:`/`repositories:` | ~460 |
| S2 | Motor `axiom branch` (`ensure`, `status`, `done`) | ~1.300 |
| S3 | Preferencias personales en el hub | ~450 |
| S4 | Guardas A y B, `seal`, `allowedEditRoots` | ~700 |
| S5 | Skills, doctrina, dogfooding y docs | ~560 |
| S6 | Dashboard y TUI | ~380 |
| S7 | `axiom branch check` y `axiom branch guard` con instalación por sync | ~640 |

Orden sugerido: S1 → S2 → (S3, S4, S7) → S6 → S5. S2 seguiría muy por encima de 400 líneas incluso como rebanada propia.

## 9. Preguntas abiertas

Solo huecos reales; ninguno bloquea la spec de las capacidades ya decididas. Los valores entre paréntesis son orientativos y **no están decididos**.

| ID | Pregunta | Bloquea | Quién la cierra |
|---|---|---|---|
| Q1 | ¿Hay `git >= 2.38` en los entornos objetivo para `git merge-tree --write-tree`? No se pudo verificar en la exploración. (Si no, la capa se omite y `done` pasa a `pr_merged` o `--force`.) | `done` | design, con verificación |
| Q2 | Coexistencia o migración entre `<GitCommonDir>/gentle-ai/` y el nuevo directorio local de registro (R9). | registro local | design |
| Q3 | **Resuelta (D6).** Sin bloque `git:`, `integration_branch` vale `main` y `branch_prefix` vale `feat`. Sin `repositories:`, ningún repo tiene ajustes propios y todos heredan los valores del workspace. | — | cerrada |
| Q4 | **Resuelta (D6 y D7).** Por defecto, la raíz de worktrees es `<padre de la carpeta maestra>/<workspace>-wt`, fuera de cualquier repo. Si un proyecto no tiene preferencias guardadas en el hub, esté registrado o no, el modo es `worktree`. | — | cerrada |
| Q5 | **Resuelta (usuario, 2026-10-14).** `inc-23-knowledge-profile` se archiva en una tarea aparte, fuera de inc-25 y con su propio PR, antes de activar `axiom branch check` en la CI de este repo. El producto no añade ninguna exención para cambios preexistentes; solo quedan exentos por nombre los sumideros acumulativos (D11). | — | cerrada |
| Q6 | Destino de `scripts/axiom-worktree.ps1`/`.sh` y de `guard-primary-checkout.ps1` una vez publicado `axiom branch`: envoltorio fino o retirada (la exploración recomienda mantenerlos hasta publicar). | dogfooding | design |

## 10. Plan de reversión (Rollback Plan)

- **Por código.** Cada rebanada se entrega en commits o PRs revertibles con `git revert`, en orden inverso al de entrega. S1 (retirada de decoupled) es la única rotura deliberada: revertirla restaura el valor aceptado.
- **Por datos.** No hay migraciones de datos. `git:`, `repositories:` y la instantánea de `kickoff.yaml` las ignoran los binarios anteriores, así que revertir el binario no invalida ficheros ya escritos. El registro local del git common dir puede borrarse sin efecto sobre el repo. Las ramas y worktrees creados son objetos git normales: se conservan o se retiran con `git worktree remove` y `git branch -d`.
- **Por guarda.** La capa D falla en abierto y se desinstala con `axiom sync`; la B se resuelve ejecutando `axiom branch ensure` o volviendo a la rama del cambio; los cambios sin instantánea nunca quedan bloqueados (R15).
- **Señal para revertir:** un cambio legítimo bloqueado por `blocked(wrong_branch)` o por la guarda A sin continuación válida, `axiom branch check` en rojo sobre una rama de integración conforme, o cualquier pérdida de trabajo sin confirmar en `done` sin `--force`.

## 11. Dependencias (Dependencies)

- Resolver del planning home de inc-24 (`planning-home-resolution`), ya archivado.
- `git` en el PATH; versión mínima pendiente de Q1.
- Orden interno: S1 antes que el resto (esquema); S2 antes que guardas, dashboard y doctrina.
- Sin dependencias externas nuevas previstas.

## 12. Criterios de éxito (Success Criteria)

- [ ] **Decoupled retirado.** Un `axiom.yaml` con `topology: monorepo-decoupled` falla en `axiom workspace validate` con código `1` y un mensaje en español que nombra `multirepo`; ninguna spec viva, doc ni ejemplo menciona el valor como válido.
- [ ] **Esquema validado.** `axiom workspace validate` acepta `git:` y `repositories:` válidos y rechaza, con mensaje accionable, claves o rutas inválidas.
- [ ] **Monorepo de un rol.** En un fixture embebido con `fullstack`, `axiom branch ensure --change x` crea `<prefijo>/x` y, en modo `worktree`, un worktree en la raíz personal sin sufijo de rol; una segunda ejecución no cambia nada y devuelve el mismo JSON (idempotencia).
- [ ] **Monorepo multirrol.** Con dos roles sobre el mismo repo, `ensure` crea la rama de seguimiento `<prefijo>/x` y las ramas `<prefijo>/x-<rol>` desde ella, con sufijo de rol en sus rutas.
- [ ] **Multirepo.** El repo de specs se trabaja in situ en su integración; cada repo de código con un rol recibe `<prefijo>/x` desde su integración; el JSON lista rutas absolutas de todos los repos.
- [ ] **Sin red.** `ensure` sin `--fetch` no ejecuta ninguna operación remota (prueba) y devuelve `base.source`, `sha` y `fetched: false`.
- [ ] **`ask` tipado.** Con modo `ask`, `ensure` devuelve `decision_required` sin crear nada ni esperar entrada.
- [ ] **`in_place` sucio.** Con modo `in_place` y árbol sucio, `ensure` se niega sin modificar el repo.
- [ ] **`done` seguro.** `done` desmonta tras una fusión detectada por ascendencia o por árbol equivalente, y se niega sin `--force` cuando nada confirma la fusión o el árbol está sucio.
- [ ] **Guardas.** En una rama que no corresponde: `kickoff seal` se niega; `axiom sdd status` devuelve `blocked(wrong_branch)` con continuación; cada escritor Go de la carpeta del cambio se niega (prueba por escritor). Un cambio sin instantánea no se bloquea.
- [ ] **Invariante.** `axiom branch check` falla en una rama de integración con un cambio activo bajo `openspec/changes/` y pasa con solo `archive/`, `e2e-cumulative` o `cumulative-<rol>`; en el repo de specs de un multirepo no aplica el invariante.
- [ ] **Autoridad de edición.** Un cambio cuyas tareas tocan su worktree no queda en `blocked(edit_authority_missing)`.
- [ ] **Paridad.** Ningún asset del producto contiene `EnterWorktree` ni `ExitWorktree` (prueba de pin); la doctrina ODD generada invoca `axiom branch ensure --lane odd` y ya no dice "branch first".
- [ ] **Hook.** `axiom sync` instala `axiom branch guard` en Claude Code, Codex y OpenCode; fuera de un proyecto Axiom, o ante cualquier error interno, `guard` no bloquea.
- [ ] **Hub y dashboard.** El registro pide modo y raíz; el dashboard edita solo las preferencias personales, muestra `git:` en solo lectura, invoca `ensure` al crear un cambio y lista los cambios de los worktrees registrados (también la TUI).
- [ ] **Dogfooding.** Este repo declara `git:`; `CLAUDE.md`, `AGENTS.md` y `GEMINI.md` remiten a `axiom branch`; la transición de los cambios activos en `main` queda resuelta según Q5.
- [ ] **CI verde.** `go test ./...` y `go vet ./...` pasan; los ratchets de refusals y deadcode no empeoran; los goldens están regenerados.
