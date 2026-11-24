# Diseño: Verificación conjunta del cambio antes de la PR a la rama principal (C-02-b)

> Tercera versión (2026-11-24), **simplificada** por decisión del usuario (Engram `sdd/verify-conjunto/decision-rama-borrada`). Implementa exactamente los deltas vigentes: REQ-25.1/25.2 (MODIFIED), REQ-25.51 a REQ-25.54, REQ-28.1/28.4 (MODIFIED), REQ-28.41, REQ-28.42 y REQ-43.32. Se retiran de la versión anterior el registro de puntas (`axiom sdd tips record`, `branch-tips.yaml`, paquete `branchtips`), la localización de ramas borradas por SHA o por mensaje, la vigencia de rondas, la contención de ramas de rol, las políticas `deferred`/`optional`, el estado «ofrecida y declinada» y los motivos y avisos que la spec ya no tiene.

## Enfoque técnico

Tres piezas, de abajo arriba. Ninguna añade estado de gobierno: no cambian `kickoff.yaml`, `gates.yaml`, `axiom sdd status`, el registro local de ramas ni los contratos `axiom.branch-result/v1` y `axiom.sdd-gate-record/v1`.

1. **Cálculo nativo de solo lectura** (`internal/branch/**`): el verbo `axiom branch scope --change <c> [--json]` reutiliza el plan que ya construye `axiom branch status` (`gather` + `planWithPromotion`) y, por repositorio de código, toma la **única** rama del cambio del plan, resuelve base e integración sin red, fija el punto de partida estable por primer padre, lista commits propios, excluidos y ficheros, comprueba la fusión en la integración con `mergeLayers` y emite `axiom.branch-scope/v1` con el resumen y la oferta (REQ-25.51 a REQ-25.54).
2. **`sdd-verify` en modo conjunto** (`internal/assets/skills/sdd-verify/**`): una referencia nueva, `references/joint-verification.md`, para el verify sin rol lanzado en modo conjunto: consume el JSON de `scope`, inspecciona solo commits y añade una ronda fechada «Verificación conjunta» a `verify-report.md` (REQ-28.4, REQ-28.41).
3. **Doctrina del orquestador** (`internal/assets/skills/_shared/sdd-orchestrator-sections.md`): punto (d) de «SDD Independent Review», los tres estados de la presentación de `integration` y el no bloqueo (REQ-28.1, REQ-28.42, REQ-43.32).

Flujo `checkpointed`: cada rol fusiona cuando termina → al presentar `integration`, el orquestador ejecuta `axiom branch scope --json` y busca rondas en `verify-report.md` → declara un estado (hecha, ofrecida, no verificable) → si la persona acepta o lo pide, lanza `sdd-verify` sin rol en modo conjunto → las dos salidas de `integration` no cambian. En `continuous` no hay compuerta `integration` ni oferta.

## Decisiones de arquitectura

### D1. Verbo propio `axiom branch scope` (se mantiene)

**Elección:** quinto verbo de la familia, de solo lectura, con contrato propio (REQ-25.1 y REQ-25.2 ya lo admiten).

**Alternativas descartadas:** ampliar `status --json` (encarece una orden frecuente, cuya lista cerrada de lecturas fijan `status_test.go:352` y `dogfooding_branch_status_test.go`, y acopla dos contratos); prosa en la skill (no se puede fijar con un test que un commit ajeno no entra).

**Justificación:** el coste de rangos y listas de ficheros queda en el único punto del flujo que lo necesita.

### D2. Una rama por repositorio: la del cambio del plan

**Elección:** por cada `RepoPlan` de `planWithPromotion`, la rama es la entrada de `Branches` con `Kind == KindChange`. El planificador ya resuelve su nombre con la precedencia instantánea del kickoff > excepción `--branch` del registro > política (`internal/branch/planner.go:412-426`, `snapshotBranch`). Las ramas `role` y `archive` se ignoran.

**Alternativas descartadas:** leer `kickoff.yaml` directamente (duplicaría la precedencia del planificador y perdería las excepciones del registro); recorrer las ramas de rol (fuera de alcance por decisión del usuario).

**Justificación:** reutiliza la misma rama que `ensure` crea y `status` describe; ninguna primitiva nueva.

### D3. Rechazos y entrada del verbo

- La CLI valida antes del motor: falta `--change`, nombre inválido (`gitref.ValidBranchComponent`), bandera desconocida o `--fetch` (el `FlagSet` de `scope` no la declara) → salida `2` sin leer nada (REQ-25.51). Banderas: `--change`, `--cwd`, `--json`; sin `--lane` (REQ-25.1 no se la da a `scope`).
- El motor llama a `gather(ctx, "scope", cwd, change, LaneSDD)`; `sealedRoles` ya devuelve `hasKickoff` y la instantánea `git:` (`engine.go:322-344`).
- Sin kickoff: si existe el documento ODD `odd/tasks/<cambio>.md` bajo la raíz del workspace → `scope_not_applicable` (código nuevo); si no → `kickoff_not_sealed` (código existente). Kickoff sin bloque `git:` → `scope_not_applicable`.
- Kickoff, `axiom.yaml` o instantánea ilegibles → `config_invalid` (lo produce ya `gather`/`sealedRoles`). Un fallo de git no cubierto → `git_failed` vía `gitRefusal`, que aborta sin resultado parcial.
- `planWithPromotion` se invoca en modo no interactivo, como `status`; el modo de promoción lo congela la instantánea.

### D4. Resolución de referencias sin red

- **Punta:** `refs/heads/<rama>`; si no existe, `refs/remotes/origin/<rama>` (la entrada declara `source: remote-tracking`). Si ninguna existe → `unverifiable` / `branch_missing`, mensaje «la rama ya no está o falta un fetch autorizado» y `continuation: git -C "<gitRoot>" fetch origin` (`pathquote.Quote`), que nunca se ejecuta.
- **Base:** `resolveBase(repo)` existente (`status.go:187`), que honra `RepoPlan.Base()` (`main_branch` en `branch`, integración en `release` y en una sola etapa) y la excepción `--base`, y prefiere `origin/<rama>` ya descargada.
- **Integración:** la primera referencia existente de `integrationReferences(repo.IntegrationBranch)` (`merge_layers.go:15`).
- Si falta la base o la integración → `unverifiable` / `base_missing` con `details.missingRef` y la misma continuación de `fetch`.
- Solo `origin`, como el resto de la familia (`originRemote`, REQ-25.10).

### D5. Punto de partida por primer padre, commits propios y excluidos

Una sola función auxiliar sobre la primitiva nueva `RevList`:

```
firstParentFork(R, t):
  E = rev-list --first-parent --parents <R> ^<t>
  E vacío           → sha(R)                (R ya está contenido en t)
  último de E sin padre → indeterminado
  en otro caso      → primer padre del último de E
```

Es el commit de la cadena de primeros padres de `R` más reciente que ya contiene `t`. No se mueve al fusionar `t` en `R` con commit de fusión ni con squash: el commit de fusión o de squash y cualquier commit ajeno `F` posterior a la bifurcación quedan en `E`, y la cadena acaba en el mismo punto que antes de fusionar.

Con `B` = base resuelta e `I` = integración resuelta:

| Paso | Base igual a la integración (`release`, una etapa) | Base distinta (`branch`, o `--base`) |
|---|---|---|
| Inicio | `fB = firstParentFork(B, t)` | `fB = firstParentFork(B, t)`; `fI = firstParentFork(I, t)` |
| `fB == t` | Reflog (D6) | `empty` |
| Revisiones propias | `[t, ^fB]` | `[t, ^fB, ^fI]`, salvo que `fI` esté en la línea de primeros padres de `t` posterior a `fB` (ver aviso) |
| Excluidos | `O` = el más antiguo de `rev-list --first-parent t ^fB`; `excluded = rev-list fB ^O^1` si `O^1 != fB` | `excluded = rev-list fI ^fB` |
| Aviso | — | Si `fI` está en `rev-list --first-parent t ^fB` (la integración entró en la rama por avance rápido, o la rama entró así en la integración): revisiones `[t, ^fB]`, sin excluidos y `foreign_commits_possible` |

- **`develop` fusionada dentro de la rama** en `release`: `fB` pasa a ser la última punta incorporada de la integración, de modo que `F` queda fuera de `t ^fB` y aparece en `excluded` (los commits de la integración entre la bifurcación original `O^1` y esa punta).
- En `branch`, `^fI` quita del alcance lo que entró desde la integración y `rev-list fI ^fB` lo lista como excluido. Sin integración fusionada en la rama, `fI` es ancestro de `fB` y no se excluye nada.
- `start = fB` (o el SHA del reflog), `range = "<start>..<tip>"` y `revisions` es el conjunto exacto que la skill reproduce (`git -C … log <revisions>`).
- `firstParentFork` indeterminado (commit sin padre: historias sin relación o clon superficial) → `unverifiable` / `fork_point_undetermined`.

**Límites aceptados** (no los pide la spec; nunca meten commits ajenos): (a) si la rama se fusionó parcialmente en la integración y después la integración se fusionó en la rama, los commits propios ya integrados quedan fuera del alcance; (b) en `branch`, tras la PR hacia `main_branch`, la base ya contiene la rama y el resultado es `empty`: el verify conjunto se hace antes de esa PR.

**Alternativas descartadas:** `git merge-base` (tras fusionar devuelve la punta y el alcance queda vacío); `merge-base --fork-point` (depende del reflog remoto, no determinista); el bucle acotado de la versión anterior (cubría el límite (a) a costa de un algoritmo iterativo que la spec no pide).

### D6. Avance rápido en `release`: reflog de creación de la rama local (se mantiene)

Cuando `fB == t` y la base es la integración, `t` puede ser una rama recién creada o una rama fusionada por avance rápido. Se lee la entrada más antigua del reflog de la rama **local** con la primitiva nueva `BranchCreation` (`git reflog show --format=%H%x09%gs refs/heads/<rama> --`), válida solo si su asunto empieza por `branch: Created from ` y el origen no es la copia remota de la propia rama (adopción de REQ-27.15). Con su SHA `C`: `C == t` → `empty`; `IsAncestor(C, t)` → `start = C` (`source: reflog`), revisiones `[t, ^C]`, excluidos por la misma regla de la columna izquierda de D5; en otro caso, o sin entrada válida (rama solo remota, adoptada, reflog caducado) → `fork_point_undetermined`.

**Justificación:** es la única forma local y sin red de distinguir «sin commits» de «fusionada por avance rápido»; el resto de casos no lee el reflog.

### D7. Ficheros, commits y resultado

Una sola lectura con la primitiva nueva `CommitFiles` sobre las revisiones propias:

```
git log -z --no-renames --no-show-signature --format=<centinela>%H%x09%P --name-only <revisions…> --
```

- `commits`: cada SHA con `merge: true` si tiene más de un padre (git no lista ficheros de los commits de fusión con `--name-only`).
- `files`: unión ordenada y sin repetidos de las rutas, quitando las que empiezan por `<WorkspaceRel>/openspec/changes/<cambio>/` en el repositorio del planning home (`facts.input.WorkspaceRel`; `PlanningHome` es siempre `<WorkspaceRoot>/openspec`, `workspace/locate.go:48`). Con `--no-renames` un renombrado aporta las dos rutas.
- Resultado: `ok` si hay commits propios y `files` no queda vacío; `empty` si no hay commits propios o todos tocan solo la carpeta del cambio (REQ-25.53).

**Alternativa descartada:** `git diff --name-only <start> <tip>` (el diff neto mezcla lo que entró por fusiones de la integración).

### D8. Fusión en la integración y oferta

- `merged` = `mergeLayers(gitRoot, tipRef, integrationReferences(I))` existente (`merge_layers.go:28`): capa `ancestry` y, si no basta, `equivalent_tree`; `mergedBy` declara la capa. Sin soporte de `merge-tree --write-tree` (git < 2.38) y sin ascendencia, `merged: false`: un squash se verá pendiente y la persona aún puede pedir la verificación de forma explícita (límite documentado).
- Solo se calcula para entradas `ok` (las `empty`/`exempt` no son pendientes).
- `branch.SummarizeScope(repos)`, función pura: `pending` lista cada repositorio `unverifiable` (con su motivo) y cada `ok` con `merged: false` (motivo `not_merged`); `offer` = `unverifiable` > `pending_merge` > `nothing_to_verify` (ningún `ok`) > `offerable`. `pending` se serializa siempre como lista.

### D9. Repositorios exentos y topologías

- `RepoSpecsInPlace` → `exempt`, sin ninguna lectura de git.
- `monorepo-embedded`: una sola entrada (`change_branch`), con la carpeta del cambio excluida (D7).
- Instantánea con solo el repositorio de specs → todas `exempt`, `offer: nothing_to_verify`, salida `0`.
- Repositorios declarados que no están en esta máquina (`plan.Skipped`): ver P1.

### D10. Contrato `axiom.branch-scope/v1` (se mantiene la envoltura de la familia)

Tipo `branch.ScopeResult` con `schema`, `operation: "scope"`, `status`/`statusLabel` (`ready` o `refused`), `code`/`message`/`continuation`/`details` de `branch.Refusal`, tokens en inglés con etiqueta en castellano (`branch.Label`), rutas absolutas y JSON estable (sin marcas de tiempo, claves de `details` ordenadas). Salida `0` cuando el alcance se calculó, aunque haya `unverifiable` o pendientes; `1` ante un rechazo tipado; `2` ante un uso incorrecto. Sin `--json`, `WriteText` imprime en castellano una línea por repositorio y la oferta.

### D11. `sdd-verify` en modo conjunto (simplifica la D12 anterior)

Solo hay un modo nuevo. El verify global sin rol de REQ-21.15 que no se lance en modo conjunto no cambia. `references/joint-verification.md` fija:

1. El orquestador lanza `sdd-verify` sin rol con `mode: joint` solo cuando la persona acepta la oferta o la pide de forma explícita.
2. Ejecuta `axiom branch scope --cwd "<workspaceRoot>" --change <c> --json` en el momento (el resultado recalculado manda). Si falla, no es aplicable o el `schema` no es `axiom.branch-scope/v1`, se comporta como el verify sin rol de siempre y el informe declara el motivo (código o «consulta no disponible»); nunca falla por ello ni cambia su veredicto.
3. Inspecciona, por repositorio `ok`, solo commits: `git -C "<gitRoot>" log`/`show` sobre `revisions`, `commits` y `files` del JSON. Nunca el árbol de trabajo, el índice ni los ficheros sin seguimiento.
4. La suite completa de pruebas de REQ-21.15 no se acota por el alcance; se ejecuta donde la ejecuta el verify global, y el informe declara el `HEAD` usado y si coincide con la `tip`.
5. Si todos los repositorios son `empty`/`exempt`, no escribe ronda y lo declara.
6. Si no, añade al final de `verify-report.md` (el `verify_file` de `fullstack` cuando es el único rol; en `multirepo`, el de la carpeta del cambio del planning home):

```
## Verificación conjunta (AAAA-MM-DD)

Alcance: axiom.branch-scope/v1 · solo commits

### <repositorio>
- resultado: ok | empty | unverifiable (<motivo>) | exempt
- rama: <rama> · <local|remote-tracking> · fusionada en <integración>: sí | no
- tip: <sha>   rango: <start>..<tip>
- excluidos: <sha…> | ninguno   avisos: foreign_commits_possible | ninguno
- fetch a autorizar: git -C "<gitRoot>" fetch origin     (solo con branch_missing o base_missing)

**Resultado conjunto:** limpio | no limpio — <huecos>
```

La ronda es de solo añadir: nunca borra rondas anteriores, nunca escribe líneas `verdict:` ni `**Estado:**`, y no toca «Cumplimiento técnico» ni la matriz ISO. Es «no limpio» con cualquier repositorio `unverifiable` o rama sin fusionar; `empty` y `exempt` se declaran y no son huecos. Nunca ejecuta `fetch`.

**Alternativas descartadas:** fichero propio (la spec lo descarta); modo `global` que también consuma `scope` (REQ-28.4 lo limita al modo conjunto).

### D12. Doctrina del orquestador (simplifica la D13 anterior)

Sin sección compartida nueva ni marcadores nuevos en las doce plantillas:

- **Puertas** (`sdd-orchestrator-sections.md`, una frase nueva tras la de `Commit Review` de la línea 85, sin reflujo de la línea 81): «At the `integration` gate, the presentation also declares the state of the joint verify, as `SDD Independent Review` describes; it never changes the two exits of `integration` nor the merge evidence it requires.»
- **«SDD Independent Review»** (líneas 95-120): «Three points, and only these.» pasa a «Four points, and only these.» con el punto (d), que es solo una oferta (ninguna preferencia la lanza y no se añade pregunta a la preferencia). Un párrafo **Joint verify.**:
  1. Al presentar `integration` en `checkpointed`, ejecutar `axiom branch scope --cwd <root> --change <c> --json` y buscar encabezados `## Verificación conjunta (AAAA-MM-DD)` en el informe consolidado.
  2. Declarar exactamente un estado: **hecha (fecha de la última ronda)** si hay alguna ronda, sin calcular vigencia; **ofrecida** si no hay ronda y `offer` es `offerable`; **no verificable (motivo)** en otro caso: el motivo de cada repositorio `unverifiable`, «ramas pendientes de fusionar» con las ramas de `pending`, «sin commits propios que verificar», o el código del rechazo o «consulta no disponible» cuando la orden falla. La declaración nunca retrasa la presentación.
  3. Para `branch_missing` y `base_missing`, mostrar el `fetch` que la persona debe autorizar; nunca ejecutarlo. Recomendar no borrar las ramas del cambio hasta haber hecho la verificación conjunta.
  4. Lanzar `sdd-verify` sin rol en modo conjunto solo si la persona acepta, y atender siempre una petición explícita, también con la preferencia `none` o con huecos.
  5. Nunca bloquea: las dos salidas, la evidencia de fusión, `nextRecommended` y las dependencias no cambian. En `continuous` no hay compuerta ni oferta.

El texto evita el vocabulario de frontera de `independent_review_boundary_test.go:18`.

### D13. Sin cambios en `axiom sdd status` (se mantiene)

Ni criterios en `internal/sddstatus/governance.go` ni campos en la proyección (REQ-28.1: «la oferta NO DEBE añadir estados»; REQ-43.32). `status` no ejecuta lecturas por repositorio.

## Flujo de datos

```
kickoff.yaml (git:) ─┐
axiom.yaml ──────────┼─→ gather ─→ planWithPromotion ─→ RepoPlan[] (rama KindChange)
registro local ──────┘                                    │
                       ┌──────────────────────────────────┘
                       ▼
  specs_in_place → exempt
  tip: refs/heads → origin/<rama> → branch_missing
  base/integración: resolveBase · integrationReferences → base_missing
  firstParentFork (RevList) ─→ reflog (BranchCreation) solo si fB == tip en release
  CommitFiles(revisions) → commits, files (sin openspec/changes/<c>/)
  mergeLayers(tip, integración) → merged
                       │
                       ▼
          SummarizeScope → offer, pending
                       │
          axiom.branch-scope/v1
             ├──→ orquestador en `integration` (tres estados; nunca fetch)
             └──→ sdd-verify sin rol, modo conjunto → verify-report.md += ronda
```

## Cambios de ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/branch/scope.go` | Crear | `ScopeResult`, `RepositoryScope`, `ScopeRef`, `ScopeStart`, `ScopeCommit`, `ScopeSummary`, `ScopePending`; tokens; `SummarizeScope`; `WriteJSON`, `WriteText`, `ExitCode` |
| `internal/branch/scope_test.go` | Crear | Precedencia de la oferta, `pending` nunca `null`, JSON estable y etiquetas |
| `internal/branch/types.go` | Modificar | `CodeScopeNotApplicable`; `RevEntry`, `CommitPaths` |
| `internal/branch/labels.go` | Modificar | Etiquetas de los tokens nuevos (resultados, motivos, oferta, `not_merged`, `first_parent`, `reflog`, `foreign_commits_possible`) |
| `internal/branch/ports.go` | Modificar | `RevList`, `CommitFiles`, `BranchCreation` en el puerto `Git` |
| `internal/branch/gitcli/scope.go` | Crear | Las tres lecturas con `gitcommon.Run`, `plainArgument` y `commandFailed` |
| `internal/branch/gitcli/ops.go` | Modificar | Doc del paquete: `log` y `reflog show` en la lista cerrada de lecturas |
| `internal/branch/gitcli/scope_test.go` | Crear | Las tres lecturas con git real (`-z` con espacios y no ASCII, reflog tras `branch` y tras adopción) |
| `internal/branch/gitcli/ops_test.go` | Modificar | `TestReadsLaunchOnlyTheClosedListOfReadSubcommands` (`:441`): nuevas lecturas y `log`/`reflog` en la lista |
| `internal/branch/engine/scope.go` | Crear | `Engine.Scope`: rechazos (D3), recorrido del plan, D4 a D9 |
| `internal/branch/engine/engine.go` | Modificar | `gitReader` con las tres lecturas |
| `internal/branch/engine/pending_test.go` | Modificar | Stubs de `fakeGit` para las lecturas nuevas |
| `internal/branch/engine/scope_test.go` | Crear | Pruebas con git real (ver *Estrategia de pruebas*) |
| `internal/branch/result_test.go` | Modificar | `CodeScopeNotApplicable` en la lista de códigos (`:21`) si la prueba exige exhaustividad |
| `internal/cli/branch_scope.go`, `branch_scope_test.go` | Crear | `runBranchScope`, `branchScopeHelp` (`--change`, `--cwd`, `--json`; sin `--fetch` ni `--lane`) |
| `internal/cli/branch.go` | Modificar | `case "scope"`, ayuda de cinco verbos, comentario de cabecera (`:20`) y mensaje «los disponibles son ensure, status, scope, done y guard» (`:223`) |
| `internal/cli/branch_test.go` | Modificar | Escenario «La ayuda lista los cinco verbos» (`:294`) |
| `cmd/axiom/main.go` | Modificar | Línea `branch scope` en la ayuda global (`:123-126`) |
| `cmd/axiom/branch_help_test.go` | Modificar | `scope` en `implemented` y comentario de cinco verbos |
| `internal/assets/skills/sdd-verify/references/joint-verification.md` | Crear | D11 |
| `internal/assets/skills/sdd-verify/SKILL.md` | Modificar | Regla y fila de *Decision Gates* del verify sin rol en modo conjunto; entrada en *References* |
| `internal/assets/skills/sdd-verify/references/report-format.md` | Modificar | Elemento «Verificación conjunta» (ronda fechada, de solo añadir, sin `verdict:` ni `**Estado:**`) |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modificar | D12 |
| `internal/assets/assets_test.go` | Modificar | Registrar `joint-verification.md` en la lista de assets (`:370`) |
| `internal/assets/independent_review_boundary_test.go` | Modificar | Pasajes nuevos en la comprobación de vocabulario de frontera y pines (ver *Estrategia de pruebas*) |
| `internal/components/sdd/review_ledger_contract_test.go` | Modificar | SHA-256 de los ajustes de Kilocode (`TestKilocodeReviewSettingsMatchCurrentMainBaseline`, `:356-375`) con su comentario de cambio deliberado |
| `testdata/golden/**` | Modificar | Orquestadores regenerados con `-update` |
| `docs/manual-de-inicio.md`, `docs/architecture/AXIOM_ENTERPRISE_VISION.md` | Modificar | `axiom branch scope` en «Ramas y Worktrees» y en la lista de verbos |
| `internal/workspace/docs_contract_test.go` | Modificar | Pines del manual (`:463-497`) y de la visión (`:520`, «`ensure`, `status`, `scope`, `done` y `guard`») |
| `internal/sddstatus/joint_verify_nonblocking_test.go` | Crear | Con y sin ronda «Verificación conjunta» en `verify-report.md`: `nextRecommended` y dependencias idénticos |

## Interfaces y contratos

```go
// internal/branch/scope.go
const ScopeSchema = "axiom.branch-scope/v1"

type ScopeResult struct {
	Schema        string                 `json:"schema"`
	Operation     string                 `json:"operation"` // "scope"
	Status        Status                 `json:"status"`
	StatusLabel   string                 `json:"statusLabel"`
	Code          Code                   `json:"code,omitempty"`
	Message       string                 `json:"message,omitempty"`
	Continuation  string                 `json:"continuation,omitempty"`
	Details       map[string]string      `json:"details,omitempty"`
	Change        string                 `json:"change,omitempty"`
	Topology      workspace.TopologyType `json:"topology,omitempty"`
	WorkspaceRoot string                 `json:"workspaceRoot,omitempty"`
	Summary       *ScopeSummary          `json:"summary,omitempty"`
	Repositories  []RepositoryScope      `json:"repositories,omitempty"`
}

type ScopeSummary struct {
	Offer      string         `json:"offer"` // offerable | pending_merge | unverifiable | nothing_to_verify
	OfferLabel string         `json:"offerLabel"`
	Pending    []ScopePending `json:"pending"` // siempre lista
}

type ScopePending struct {
	Repository  string `json:"repository"`
	Branch      string `json:"branch,omitempty"`
	Reason      string `json:"reason"` // not_merged | branch_missing | base_missing | fork_point_undetermined
	ReasonLabel string `json:"reasonLabel"`
}

type RepositoryScope struct {
	Name          string            `json:"name"`
	GitRoot       string            `json:"gitRoot"` // absoluta
	Role          RepoRole          `json:"role"`    // change_branch | code | specs_in_place
	RoleLabel     string            `json:"roleLabel"`
	Promotion     Promotion         `json:"promotion,omitempty"`
	Outcome       string            `json:"outcome"` // ok | empty | unverifiable | exempt
	OutcomeLabel  string            `json:"outcomeLabel"`
	Reason        string            `json:"reason,omitempty"` // solo con unverifiable
	ReasonLabel   string            `json:"reasonLabel,omitempty"`
	Message       string            `json:"message,omitempty"`
	Continuation  string            `json:"continuation,omitempty"` // fetch a autorizar
	Details       map[string]string `json:"details,omitempty"`      // missingRef
	Branch        *ScopeRef         `json:"branch,omitempty"`       // name, ref, source (local|remote-tracking), sha
	Base          *ScopeRef         `json:"base,omitempty"`
	Integration   *ScopeRef         `json:"integration,omitempty"`
	Start         *ScopeStart       `json:"start,omitempty"` // sha, source: first_parent | reflog
	Tip           string            `json:"tip,omitempty"`
	Range         string            `json:"range,omitempty"`     // "<start>..<tip>"
	Revisions     []string          `json:"revisions,omitempty"` // ["<tip>", "^<start>", ("^<fI>")]
	Commits       []ScopeCommit     `json:"commits,omitempty"`   // sha, merge
	Excluded      []string          `json:"excluded,omitempty"`
	Files         []string          `json:"files,omitempty"` // relativas a GitRoot, con «/»
	Merged        *bool             `json:"merged,omitempty"` // solo en entradas ok
	MergedBy      MergeLayer        `json:"mergedBy,omitempty"` // ancestry | equivalent_tree
	MergedByLabel string            `json:"mergedByLabel,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"` // foreign_commits_possible
}

func SummarizeScope(repos []RepositoryScope) ScopeSummary

// internal/branch/types.go
type RevEntry struct {
	SHA     string
	Parents []string
}
type CommitPaths struct {
	SHA     string
	Parents []string
	Paths   []string
}

// internal/branch/ports.go (y gitReader)
RevList(ctx context.Context, dir string, firstParent bool, revisions ...string) ([]RevEntry, error) // rev-list --parents [--first-parent] <revs> --
CommitFiles(ctx context.Context, dir string, revisions ...string) ([]CommitPaths, error)             // log -z --no-renames --name-only
BranchCreation(ctx context.Context, dir, name string) (sha, from string, ok bool, err error)          // reflog show, entrada más antigua

// internal/branch/engine
func (e *Engine) Scope(ctx context.Context, cwd, change string) branch.ScopeResult
```

Justificación de las primitivas nuevas: `RevList` es la única forma de leer la cadena de primeros padres con sus padres (`AheadBehind` solo cuenta); `CommitFiles` da commits y ficheros por commit en una sola lectura, que ninguna primitiva existente ofrece; `BranchCreation` solo se usa en el avance rápido de D6. Se reutilizan sin cambios `ResolveCommit`, `IsAncestor`, `resolveBase`, `integrationReferences`, `mergeLayers`, `gather`, `planWithPromotion`, `RepoPlan.Base()` y `gitRefusal`.

## Estrategia de pruebas

TDD con el runner del proyecto: `go test` acotado por paquete y, al cerrar, `go test ./...` y `go vet ./...`. **Nunca se compila un binario `axiom` suelto:** la CLI se prueba en proceso (`cli.RunBranch`); los repositorios git son temporales (`t.TempDir()`, con el `TestMain` aislado de `internal/branch/engine/status_test.go`, `sealChangeAt` y la configuración multirrepo de `ensure_multirepo_test.go`); los remotos son repos *bare* locales y el único `fetch` ocurre al preparar la prueba.

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unidad (`internal/branch`) | `SummarizeScope`: precedencia `unverifiable` > `pending_merge` > `nothing_to_verify` > `offerable`; `empty`/`exempt` no pendientes; `pending` nunca `null`; tokens con etiqueta | Tabla sin git |
| Integración (`gitcli`) | `RevList`, `CommitFiles` (ruta con espacio y no ASCII, renombrado con dos rutas), `BranchCreation` (creada con `branch` → válida; adoptada de `origin/<rama>` → no válida); lista cerrada de lecturas ampliada | Repos temporales |
| **E1 `release` con commit ajeno** | `feat/x` sale de `develop`, llega `F` (toca `otro.go`), un commit propio toca también `openspec/changes/x/tasks.md`. Subcasos: sin fusionar, fusión `--no-ff`, squash → mismos commits y ficheros, sin `F`, sin `otro.go` ni la carpeta del cambio; `merged` sí tras fusionar (squash solo si `merge-tree` está disponible). En el mismo test: observador de `gitcommon` sin operaciones remotas; referencias, índice y worktrees idénticos antes y después, con cambios sin confirmar y un fichero sin seguimiento fuera de `files` | Git real |
| **E2 `branch` con commit ajeno** | Rama desde `main`, `F` en `develop`, rama fusionada en `develop` → solo commits propios frente a `main`; subcaso con `develop` fusionada dentro de la rama → `F` en `excluded` y fuera de `files` | Git real, `main_branch` en el `axiom.yaml` de la prueba |
| **E3 avance rápido** | `release`: rama local fusionada por avance rápido → `ok`, `start.source: reflog`; rama recién creada → `empty`; rama solo remota fusionada por avance rápido → `fork_point_undetermined` | Git real |
| **E4 multirepo** | Dos repos de código y specs in situ: `api` fusionada → `ok`; `web` sin rama → `branch_missing` con el mensaje y la continuación, sin `fetch`; specs → `exempt`; `offer: unverifiable`. Subcaso con `web` presente sin fusionar → `pending_merge` con `web` en `pending` | Git real |
| CLI (`branch scope`) | `--fetch` y falta de `--change` → `2` sin leer repositorios; sin kickoff → `kickoff_not_sealed` (`1`); documento ODD → `scope_not_applicable` (`1`); `--json` con `schema`; ayuda de `axiom branch` con cinco verbos | `cli.RunBranch` en proceso |
| No bloqueo (`sddstatus`) | Con y sin ronda en `verify-report.md`: `nextRecommended` y dependencias idénticos | En proceso |
| Assets | `sdd-verify/SKILL.md` contiene la regla y la fila nuevas; `joint-verification.md` contiene `axiom branch scope`, `git -C`, «Verificación conjunta», `**Resultado conjunto:**`, `branch_missing` y «consulta no disponible», y no contiene `ls-files --others` ni rangos `...HEAD` (`assertNoHeadRangeScope`); la sección compartida contiene «Four points, and only these.», `axiom branch scope` y la frase de `integration`; vocabulario de frontera limpio en todos los pasajes nuevos; frases fijadas de `integration` (`assets_test.go`, `orchestrator_stop_exits_test.go`) intactas | Pines existentes ampliados |
| Goldens | Orquestadores regenerados; hash de Kilocode actualizado | `go test ./... -update` y revisión del diff |

## Matriz de amenazas

| Frontera | Casos adversarios mínimos | Aplicabilidad | Respuesta del diseño | Pruebas RED previstas |
|---|---|---|---|---|
| Rutas con aspecto de documentación | `requirements.txt`, `CMakeLists.txt`, Markdown ejecutable, `README.sh` | N/A: `scope` lista rutas como datos; no clasifica ni ejecuta ficheros | — | — |
| Selección del repositorio git | `git -C`, rutas relativas, rutas absolutas | **Aplicable** | Todo git del motor recibe `dir = RepoPlan.GitRoot` (absoluta), nunca el directorio del proceso; `--cwd` se resuelve a absoluta; toda referencia y SHA pasan por `plainArgument`; la skill usa `git -C "<gitRoot del JSON>"`; la continuación de `fetch` se cita con `pathquote.Quote` y nunca se ejecuta | (1) `--cwd` relativa resuelta; (2) pin `git -C` en `joint-verification.md` |
| Estado de commit | stage, `commit -a`, índice vacío | **Aplicable** | Solo se leen commits con las lecturas de `gitcommon` (`GIT_OPTIONAL_LOCKS=0`); nada refresca el índice | E1: cambios con y sin stage y sin seguimiento fuera de `files`; referencias, índice y worktrees idénticos |
| Estado de push | rama de seguimiento, primer push, refspec explícito | **Aplicable** | Nunca `push`, `fetch`, `pull` ni `ls-remote`; `origin/<rama>` solo si ya está descargada, y se declara | E1: observador sin operaciones remotas; E4: `branch_missing` sin `fetch`; CLI: `--fetch` rechazada con `2` |
| Órdenes de PR | `--head` explícito, prefijo de entorno, órdenes compuestas | N/A: no se componen ni ejecutan órdenes de PR | — | — |

## Migración y despliegue

Sin migración. Aditivo: verbo `scope`, referencia de la skill y texto de doctrina. Tras un *revert*, una ronda «Verificación conjunta» que quede en un `verify-report.md` es texto inerte que nada digiere. Entrega `single-pr` en `inc/verify-conjunto`.

**Estimación de líneas de autor** (sin goldens): ≈ 700 de producción (`internal/branch` ≈ 560, CLI ≈ 90, skills y doctrina ≈ 50), ≈ 600 de pruebas y ≈ 40 de documentación y pines; total ≈ 1.300-1.400.

## Preguntas abiertas

- [ ] **P1 — Repositorio declarado que no existe en esta máquina.** REQ-25.53 exige que ningún repositorio se omita, pero su dominio cerrado de motivos (`branch_missing`, `base_missing`, `fork_point_undetermined`) no tiene uno para un repositorio ausente (`plan.Skipped`). El diseño lo declara `unverifiable` / `branch_missing` con `details.path` y la ruta en el mensaje, además de la frase obligatoria. Si se prefiere un motivo propio, hay que ampliar el dominio de REQ-25.53.
