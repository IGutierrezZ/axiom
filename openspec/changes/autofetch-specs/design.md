# Diseño: Autofetch del repositorio de specs — avisar y avanzar (C-05-d)

> **Cambio:** `autofetch-specs`. **Idioma:** castellano de España; identificadores, comandos, claves y rutas en inglés.
> **Entrada:** `proposal.md`, `explore.md` y las deltas `specs/change-branch-lifecycle/spec.md` (RF-BRA-39, RF-BRA-40; MODIFIED RF-BRA-01, RF-BRA-10, RF-BRA-13) y `specs/multi-project-hub/spec.md` (RF-HUB-13; MODIFIED RF-HUB-07).
> **Contexto técnico mantenido:** el proyecto no lo declara (`axiom workspace context` devuelve `technicalContext: null`), así que no hay `context-impact.md`.

## Enfoque técnico

Se sigue el enfoque híbrido C de la exploración, sin abrir un paquete nuevo de git:

1. **Preferencia en el hub.** `hub.WorkspaceRecord` gana `Autofetch bool` (`json:"autofetch,omitempty"`), con `LookupAutofetch` y `SetAutofetch` en `internal/hub/prefs.go`. El esquema sigue en `"1.0"`. El puerto `branch.Prefs` gana `Autofetch(projectMaster)`, que implementa `engine.hubPrefs` (RF-HUB-07, RF-HUB-13).
2. **Primitivas de git.** `internal/branch/gitcli` gana `FetchWithin` (fetch con tiempo máximo propio, sobre `gitcommon.RunWithTimeout`) y `FastForward` (`merge --ff-only <ref>`). Ambas pasan por el único punto de ejecución (`gitcommon`) y por los puertos `branch.Git` y `engine.gitWriter` (RF-BRA-01).
3. **Fetch implícito en `ensure`.** `Engine.Ensure` lee la preferencia del proyecto y, si está encendida y no hay `--fetch`, trae `origin` en los mismos repositorios que `--fetch`. Un fallo del fetch implícito es un aviso, no `fetch_failed` (RF-BRA-10, RF-BRA-13).
4. **Ciclo `run`.** Un método nuevo, `Engine.Autofetch`, en `internal/branch/engine/autofetch*.go`, hace el fetch acotado del repositorio de specs con intervalo mínimo, calcula la relevancia **antes** de avanzar, avanza la rama de integración solo por fast-forward estricto y devuelve un resultado tipado `branch.AutofetchResult` (RF-BRA-39).
5. **Verbo propio.** `axiom autofetch enable|disable|status|run` en `internal/cli/autofetch.go`, con un `case "autofetch"` en `cmd/axiom/main.go`. La familia `axiom branch` conserva sus cinco verbos (RF-BRA-01, RF-HUB-13).
6. **Hook de inicio de sesión.** `internal/components/sdd/inject.go` instala una entrada `SessionStart` en Claude Code (`settings.json`) y en Codex (`hooks.json`), con el patrón de `ensureBranchGuardHookEntries`. `internal/components/uninstall/service.go` la retira (RF-BRA-40).
7. **Doctrina.** Una frase canónica sustituye a la regla de `--fetch` en todos los textos, más un párrafo que ordena ejecutar `axiom autofetch run` antes de cada parada. Se regeneran los 10 goldens en el mismo commit (RF-BRA-40).

## Decisiones de arquitectura

### D-01: El ciclo vive en `internal/branch/engine`, no en un paquete nuevo

**Elección:** `Engine.Autofetch(ctx, AutofetchRequest) branch.AutofetchResult` en ficheros nuevos de `internal/branch/engine/`. Reutiliza `gitReader`, `gitWriter`, `registry.Store`, `branch.ResolvePolicy` y `branch.EffectivePolicy`, `projectMaster`, `relativeWithin` y la lectura del registro de cambios activos que ya usa la guarda (`activeChangesOf`).
**Alternativas descartadas:** (A) un paquete `internal/autofetch` con un puerto git propio, que duplica `Fetch`, `DirtyPaths` y la resolución de roles, y queda fuera de las pruebas de arquitectura de `internal/branch/**`; (B) un verbo `axiom branch autofetch`, que obliga a ampliar los cinco subcomandos de RF-BRA-01.
**Justificación:** el git del motor ya está cubierto por `gitcommon/architecture_test.go` (solo `exec.go` lanza procesos) y por `sddpublish/isolation_test.go` (ni `push` ni `commit` en `internal/branch/**`). Poner aquí el ciclo hereda esas dos garantías sin cambiar las pruebas. El verbo `axiom autofetch` es solo la puerta de la CLI.

### D-02: Primitivas `FetchWithin` y `FastForward` en `gitcli`

**Elección:**

- `FetchWithin(ctx, dir, remote string, timeout time.Duration) error`: `git fetch <remote>` sin `--prune`, con `gitcommon.RunWithTimeout`. Si el tiempo se agota, el error envuelve `gitcommon.ErrTimeout`.
- `FastForward(ctx, dir, ref string) error`: `git merge --ff-only <ref>`, con `plainArgument(ref)` y `runWrite`.

`Fetch` no cambia: `ensure` conserva su límite de 30 s, tanto con `--fetch` como con el fetch implícito. Las dos primitivas nuevas entran en `branch.Git` y en `engine.gitWriter`, que es el puerto que puede mutar. Los comentarios de `ports.go:9,40` y `engine.go:62-69` pasan a decir que `Fetch` y `FetchWithin` son las únicas operaciones remotas y que `FastForward` es la única escritura de ramas fuera de crear, desplegar y borrar.
**Alternativas descartadas:** (a) `git fetch origin <int>:<int>` para avanzar sin checkout: elude la comprobación del árbol limpio y contradice la política «solo `merge --ff-only`»; (b) `update-ref`: no comprueba que el avance sea fast-forward ni toca el árbol de trabajo.
**Justificación:** `merge --ff-only` no está entre los literales vigilados (`push`, `commit`) ni es un subcomando remoto para `remoteSubcommand`. Con la rama desplegada, `merge --ff-only` actualiza a la vez la rama, el índice y el árbol, y se niega a crear una fusión.

### D-03: El interruptor se lee del hub por la identidad del proyecto, sin lanzar git cuando está apagado

**Elección:**

- `hub.(*Manager).LookupAutofetch(projectPath string) (enabled, registered bool, err error)`.
- `hub.(*Manager).SetAutofetch(idOrPath string, enabled bool) (rec *WorkspaceRecord, changed bool, err error)`. No escribe si el valor no cambia, así que `enable` y `disable` repetidos dejan `workspaces.json` byte a byte igual. Si el proyecto no está registrado, devuelve un error que cumple `errors.Is(err, hub.ErrProjectNotRegistered)` y no escribe. Un hub ilegible devuelve el error de `Load` sin escribir.
- `branch.Prefs.Autofetch(projectMaster string) (enabled bool, state AutofetchPrefState, warning string)`, con `state` ∈ `enabled|disabled|unregistered|unreadable`. Un hub ilegible o un proyecto no registrado dan `enabled=false` y un aviso en castellano; el aviso del proyecto no registrado indica `axiom project add "<maestra>"`.
- La identidad es la carpeta maestra del checkout principal (`projectMaster`, la misma que usan las preferencias de worktree). En `run`, `status`, `enable` y `disable` se resuelve **solo con ficheros**: `workspace.IsAxiomProject`, `workspace.Locate`, `gitcommon.GitRootFromFiles`, `gitcommon.CommonDirFromFiles` y `gitcommon.MainWorktreeFromFiles`. Así, con el interruptor apagado o fuera de un proyecto Axiom, no se lanza ningún proceso de git. Si los ficheros no bastan para resolverla, se usa `location.Master`.
- `axiom.yaml` no se consulta nunca para el interruptor. Una clave `autofetch` en `axiom.yaml` se ignora (`workspace.WorkspaceConfig` no la modela).

**Alternativas descartadas:** guardar el interruptor en `axiom.yaml`, que la proposal deja fuera de alcance; o resolver la identidad con `git rev-parse`, que lanzaría procesos aunque el interruptor esté apagado y contradice RF-BRA-40.
**Justificación:** es el mismo patrón que las preferencias de worktree (D-08 de INC-25). `Register` solo reescribe nombre, topología, acceso y configuración, `Unregister` borra la entrada y `SetWorktreePrefs` solo toca sus dos campos. Por eso el nuevo campo sobrevive al re-registro y al dashboard, y desaparece al desregistrar, sin tocar esas funciones.

### D-04: `ensure` con el interruptor encendido es un `--fetch` implícito que degrada a aviso

**Elección:** `checkPlan` y `checkRepository` reciben `fetchMode` (`fetchNone | fetchExplicit | fetchImplicit`) en lugar del booleano:

- `fetchExplicit` (`--fetch`): sin cambios. Un fallo devuelve `fetch_failed` y no crea nada, también con el interruptor encendido.
- `fetchImplicit` (sin `--fetch`, con `prefs.Autofetch == true`): se ejecuta `writer.Fetch` en el mismo punto (antes de `resolveBase`) y en el mismo conjunto de repositorios que `--fetch`. Si falla, se añade al resultado el aviso `el autofetch no pudo traer origin en <repo> (<resumen>): se sigue con la base local`, `fetched` queda en `false` y la ejecución sigue.
- La preferencia se lee en `Ensure` después de `resolveMode` y de planificar la promoción, de modo que `decision_required` sigue llegando antes de cualquier red. `done`, `status` y `scope` no la leen.
- `ensure` no consulta el sello del intervalo (D-06) y no avanza ninguna rama local: `Fetch` solo actualiza `refs/remotes/origin/*`.
- `ensure --archive` comparte `checkRepository`, así que hereda el fetch implícito. `branch.ChangeNotIntegrated` gana el parámetro `autofetch bool`. Si es verdadero, ni el mensaje ni la orden de reintento llevan `--fetch`:
  - fetch implícito correcto: «… ni tras el fetch del autofetch …»;
  - fetch implícito fallido: «… el autofetch no pudo traer el remoto; comprueba la red y repite `axiom branch ensure --archive …` …».

  `ChangeIntermediateMerge` no cambia, porque ya no sugiere `--fetch`.
- Los avisos de preferencia de `Autofetch` (proyecto no registrado) **no** se añaden a `ensure`. El hub ilegible ya produce el aviso de `WorktreePrefs` en la misma lectura (ver Riesgos, R-1).

**Alternativas descartadas:** rechazar con `fetch_failed` también en el fetch implícito, que bloquearía `ensure` cada vez que falla la red (la proposal recomienda avisar y seguir); o hacer que `ensure` ejecute el ciclo completo de `run`, que avanzaría la integración desde `ensure` y respetaría el intervalo, en contra de RF-BRA-10.
**Justificación:** es el cambio mínimo sobre una ruta ya probada. Las pruebas de RF-BRA-10 que construyen el motor sin `prefs` (`prefs == nil` equivale a apagado) siguen pasando sin cambios.

### D-05: El repositorio de specs y la rama que se avanza

**Elección:** el repositorio de specs es el del planning home (`gather`/`repositoryFacts` sobre `location.WorkspaceRoot`). En `monorepo-embedded` es el propio repositorio; en `multirepo`, el de specs in situ. La rama que se avanza es `branch.EffectivePolicy(policy, planning).IntegrationBranch`. `MainBranch` solo se lee para no tocarla: cuando es distinta de la integración, ninguna orden de `run` la recibe. Cuando coinciden (repositorio de una sola etapa), avanzar la integración es el propósito del ciclo. El fetch y el avance se ejecutan solo en ese repositorio. Los repositorios de código no reciben ninguna orden remota ni de escritura: solo lecturas locales para el aviso (b) de D-07.
**Alternativas descartadas:** traer también los repositorios de código desde `run`, que RF-BRA-39 y el escenario multirepo prohíben.
**Justificación:** reutiliza el papel `RepoSpecsInPlace` del planificador y la política efectiva por repositorio que ya usa la promoción.

### D-06: Intervalo mínimo con sello propio bajo el git common dir

**Elección:**

- Sello en `<git common dir del repositorio de specs>/axiom/autofetch/v1/state.json`, con el contenido `{"lastFetch":"<RFC 3339 UTC>"}`. Se escribe de forma atómica (temporal y `rename`), como el registro de `branch/registry` (`<commonDir>/axiom/branch/v1/...`).
- Solo `run` lo escribe, y solo tras un fetch correcto.
- Las constantes son variables de paquete para que las pruebas las reduzcan, como `gitcommon.runTimeout`: `autofetchInterval = 5 * time.Minute`, `autofetchFetchTimeout = 8 * time.Second`.
- Un sello ausente, ilegible o con fecha futura cuenta como «sin fetch previo». Si no se puede escribir, se añade un aviso y el ciclo sigue.
- El reloj se inyecta en `Engine` (`now func() time.Time`; `nil` equivale a `time.Now`).
- Dentro del intervalo solo se omite la red. El avance y la relevancia, que son lecturas y escrituras locales, se ejecutan igual, de modo que la doctrina de antes de cada parada siempre devuelve avisos al día respecto de las referencias locales.

**Alternativas descartadas:** el mtime de `FETCH_HEAD`, que depende de un detalle interno de git, es por worktree y también lo cambian los fetch parciales; un campo en el hub, que obliga a escribir el hub en cada arranque, con bloqueos, y mezcla estado con preferencia (ya descartado en la exploración).
**Justificación:** es estado derivado, local a la máquina y por repositorio, en el mismo sitio que el registro de ramas. Se prueba sin depender del reloj del sistema de ficheros.

### D-07: La relevancia se calcula antes de avanzar y por rutas

**Elección:**

1. **Cambio actual.** Se lee `HEAD` del checkout de `cwd` (`gitcommon.ReadHead`, sin procesos). Es el cambio activo registrado (carriles SDD y ODD, `registry.List`) que tiene esa rama (de cambio, de rol o de archivado) para el repositorio de `cwd`. Si ninguna rama coincide y hay **exactamente un** cambio activo registrado en el planning home, se toma ese. Si no hay ninguno, o hay varios, no hay cambio actual y el ciclo no emite avisos de relevancia (RF-BRA-39 paso 3).
2. **Prefijos relevantes**, relativos a la raíz git del repositorio de specs:
   - SDD: `<planRel>/changes/<c>/` y `<planRel>/specs/<cap>/` por cada `<cap>` de `TreeDirs(<planRel>/changes/<c>/specs)`. Se toma la unión de lo que hay en la rama del cambio (`refs/heads/<rama>`, si existe en el repositorio de specs) y en `refs/remotes/origin/<int>`.
   - ODD: `<wsRel>/odd/tasks/<c>.md`.
   - `<planRel>` es `relativeWithin(planning.GitRoot, location.PlanningHome)` y `<wsRel>` es `relativeWithin(planning.GitRoot, location.WorkspaceRoot)`.
3. **Aviso (a), commits entrantes en la integración.** Se calcula **antes** del avance con `CommitFiles(dir, "refs/heads/<int>..refs/remotes/origin/<int>")`. Cuentan los commits que no son de fusión y tienen al menos una ruta bajo un prefijo relevante. Las fusiones no traen rutas y sus padres ya están en el rango. Se emite un aviso por la rama `origin/<int>`, con el número de commits, el alcance (los prefijos tocados) y la orden `git -C "<checkout>" log --oneline <sha local previo>..origin/<int> -- <prefijos>`, que sigue siendo válida después del avance. Si la rama local de integración no existe, (a) se omite.
4. **Aviso (b), commits nuevos en la rama del cambio.** Para cada rama del registro del cambio actual, en cada repositorio de su registro, si existen `refs/heads/<b>` y `refs/remotes/origin/<b>`, se llama a `AheadBehind(dir, "refs/heads/<b>", "refs/remotes/origin/<b>")`. El valor `ahead` son los commits del remoto que la rama local no tiene. Si es mayor que 0, se emite un aviso con la orden `git -C "<worktree o checkout>" merge --ff-only origin/<b>` (sin commits locales propios) o `git -C "<…>" log --oneline <b>..origin/<b>` (con divergencia). Solo se leen referencias locales: los repositorios de código no se traen (D-05).
5. Los avisos se ordenan de forma determinista: primero la integración y después las ramas del cambio por repositorio y nombre.

**Alternativas descartadas:** calcular la relevancia sobre el delta del fetch (`FETCH_HEAD`), que se pierde cuando el intervalo omite la red y no detecta lo traído por un `ensure`; o calcularla después del avance, cuando el rango ya está vacío.
**Justificación:** el rango «local frente a remota» se deduplica solo. Lo que ya avanzó deja de avisarse en la siguiente ejecución, y lo que no pudo avanzar sigue avisándose mientras siga pendiente.

### D-08: Avance por fast-forward estricto con degradación a aviso

**Elección:** con `L = refs/heads/<int>` y `R = refs/remotes/origin/<int>`:

| Situación | Acción | Aviso (`kind`) y orden exacta |
|---|---|---|
| Sin `R`, o `L == R` | nada | ninguno |
| `R` es ancestro de `L` (la local va por delante) | nada | ninguno |
| Ni `L` es ancestro de `R` ni `R` de `L` | nada | `integration_diverged`: `git -C "<checkout>" rebase origin/<int>` |
| `L` atrasada y no desplegada en ningún worktree (`Worktrees`, `Branch == <int>` y no desacoplado) | nada | `integration_not_deployed`: `git -C "<checkout principal>" fetch . refs/remotes/origin/<int>:refs/heads/<int>` (solo local y solo fast-forward) |
| `L` atrasada, desplegada y con `DirtyPaths(<checkout>)` no vacío | nada | `integration_dirty`: `git -C "<checkout>" merge --ff-only origin/<int>`, que la persona ejecuta cuando el árbol esté limpio |
| `L` atrasada, desplegada y con el árbol limpio | `FastForward(<checkout>, R)` | ninguno; `integration.advanced=true` y los SHA previo y nuevo |
| `FastForward` falla (`index.lock`, carrera u otro error) | nada más; sin reintento | `integration_ff_failed`: la misma orden que en `integration_dirty`, más el resumen de git |

El `<checkout>` es el worktree que tiene la rama desplegada. En `monorepo-embedded` suele ser el checkout principal y en `multirepo`, el repositorio de specs in situ.
**Alternativas descartadas:** avanzar una rama no desplegada con `update-ref` o con `fetch .`, que mueve una rama sin pasar por la comprobación de árbol que exige RF-BRA-39; o reintentar ante `index.lock`, que puede chocar con `scripts/axiom-worktree.ps1 done`.
**Justificación:** cumple «SOLO por fast-forward estricto y SOLO si está desplegada, limpia y sin divergencia». Todo lo demás es un aviso con la orden exacta, y el código de salida sigue siendo 0.

### D-09: Contrato de `run` y de los subcomandos de preferencia

**Elección:**

- `run`:
  - Con `--json`, devuelve `branch.AutofetchResult` (esquema `axiom.autofetch-result/v1`).
  - Sin `--json`, imprime un texto en castellano: estado del interruptor, fetch, integración y los avisos con su orden.
  - Con `--agent claude-code|codex` (modo hook), escribe por la salida estándar solo cuando hay avisos: `{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"<avisos>"}}`. Sin avisos, no escribe nada.
  - En modo hook, `cwd` sale de `--cwd`. Si no se da, sale del campo `cwd` de la carga JSON del runtime por la entrada estándar, leída con un límite de 256 KiB y fallando en abierto. Si tampoco está, es el directorio del proceso.
  - Todo `run` va acotado por un presupuesto global (`autofetchRunBudget = 12 s`, mayor que los 8 s del fetch) y protegido con `recover`.
  - Código de salida: 0 siempre, salvo un error de uso en modo CLI (2). En modo hook, también 0 ante un error de uso.
- `enable` y `disable`: `axiom.autofetch-preference/v1` con `{schema, operation, autofetch, project, registered, changed, warnings, message}`. Terminan con 0 si tienen éxito (también cuando no cambia nada). Terminan con 1 si el proyecto no está registrado (mensaje en castellano con `axiom project add "<maestra>"`), si el hub es ilegible o si no es un proyecto Axiom. Ninguno escribe `axiom.yaml`.
- `status`: el mismo esquema, sin escribir y sin lanzar git (D-03). Con el hub ilegible o el proyecto no registrado, informa `disabled` con aviso y termina con 0.
- `--agent` se acepta en los cuatro subcomandos (RF-HUB-13). Solo cambia la forma de salida en `run`.

```go
// internal/branch/autofetch.go
const AutofetchResultSchema = "axiom.autofetch-result/v1"

type AutofetchResult struct {
	Schema      string                `json:"schema"`
	Operation   string                `json:"operation"`             // "autofetch run"
	Autofetch   string                `json:"autofetch"`             // enabled | disabled
	Project     string                `json:"project,omitempty"`     // carpeta maestra
	Repository  string                `json:"repository,omitempty"`  // raíz git del repositorio de specs
	Fetch       string                `json:"fetch"`                 // done | skipped_interval | failed | not_attempted
	LastFetch   string                `json:"lastFetch,omitempty"`   // RFC 3339 del sello
	Integration *AutofetchIntegration `json:"integration,omitempty"`
	Change      *AutofetchChange      `json:"change,omitempty"`      // cambio actual resuelto, si lo hay
	Notices     []AutofetchNotice     `json:"notices"`               // [] si no hay nada que avisar
	Warnings    []string              `json:"warnings,omitempty"`    // proyecto no registrado, hub ilegible, sello
}

type AutofetchIntegration struct {
	Branch   string `json:"branch"`
	Checkout string `json:"checkout,omitempty"`
	Before   string `json:"before,omitempty"`
	After    string `json:"after,omitempty"`
	State    string `json:"state"` // up_to_date | advanced | ahead | not_advanced | no_remote_ref
	Advanced bool   `json:"advanced"`
}

type AutofetchChange struct {
	Lane branch.Lane `json:"lane"`
	Name string      `json:"name"`
}

type AutofetchNotice struct {
	Kind    string   `json:"kind"`    // incoming_relevant | change_branch_behind | integration_diverged |
	                                   // integration_not_deployed | integration_dirty | integration_ff_failed |
	                                   // fetch_failed | fetch_timeout | no_origin
	Branch  string   `json:"branch"`
	Commits int      `json:"commits,omitempty"`
	Scope   []string `json:"scope,omitempty"`
	Message string   `json:"message"`
	Command string   `json:"command,omitempty"`
}

// internal/branch/engine/autofetch.go
type AutofetchRequest struct{ CWD string }

func (e *Engine) Autofetch(ctx context.Context, r AutofetchRequest) branch.AutofetchResult
func (e *Engine) AutofetchProject(cwd string) (AutofetchProject, *branch.Refusal) // solo ficheros (D-03)
func DefaultWithHubPath(path string) *Engine                                        // pruebas de internal/cli
```

**Alternativas descartadas:** salida en texto plano por la salida estándar del hook, que Claude Code admite pero que no es el contrato documentado de Codex; o el código 2 para avisar, que en Claude Code y Codex bloquea o se muestra como error.
**Justificación:** el contrato tipado es el que consumirá C-06-e. La salida `hookSpecificOutput.additionalContext` es el canal de contexto adicional de `SessionStart` de ambos runtimes. Callar sin avisos cumple «solo cuando hay algo relevante».

### D-10: Hook `SessionStart` fail-open con orden estable

**Elección:** constantes en `inject.go`. La identidad de la entrada es la orden exacta:

| Runtime | Orden | Matcher | `timeout` |
|---|---|---|---|
| Claude Code (no Windows) | `axiom autofetch run --agent claude-code \|\| true` | `startup\|resume` | 15 |
| Claude Code (Windows) | `powershell -NoProfile -Command 'axiom autofetch run --agent claude-code; exit 0'` | `startup\|resume` | 15 |
| Codex | `axiom autofetch run --agent codex \|\| true` | `startup\|resume` | 15 |

- La forma de cada orden sigue la del hook del registro de skills de cada runtime: PowerShell en Claude Code para Windows y `|| true` en Codex.
- `ensureAutofetchHookEntries(path, label, matcher, command)` generaliza `ensureBranchGuardHookEntries` a un solo evento (`SessionStart`): añade la entrada solo si `claudeHookListContains` no la encuentra, rechaza las formas inesperadas sin tocar el fichero y escribe con `filemerge.WriteFileAtomic` solo si cambió algo.
- Se llama desde `installSkillRegistryAutomation`, en las ramas de Codex y de Claude Code. Ningún otro adaptador recibe el hook.
- La instalación no depende del interruptor.
- Desinstalación: `removeSkillRegistryHook` retira también las órdenes que contienen `axiom autofetch run`, en cualquier evento.
- Dentro del proceso, el orden de comprobaciones garantiza la inercia (D-03): `IsAxiomProject` → `Locate` → identidad por ficheros → hub → (solo si está encendido) git.

**Alternativas descartadas:** una orden sin envoltorio (como `axiom branch guard`), que con un `axiom` ausente o antiguo devuelve un código distinto de 0; o cambiar la orden en versiones futuras, que en Codex obliga a confiar de nuevo en el hook, porque la confianza se liga al hash de la orden.
**Justificación:** RF-BRA-40 exige terminar siempre con 0. La orden es constante y la retirada busca por subcadena, como las demás órdenes de Axiom.

### D-11: Doctrina con una frase canónica y un párrafo antes de cada parada

**Elección:** la frase canónica es «Never pass `--fetch` without the explicit authorization of the user, unless the user enabled autofetch (`axiom autofetch enable`).». Sustituye a la regla actual en:

- `routing.go:144`;
- `sdd-orchestrator-sections.md:66`;
- `sdd-apply/SKILL.md:111`.

En `sdd-orchestrator-sections.md`, justo después del párrafo de puertas `checkpointed` (línea 70), se añade un párrafo:

«Before presenting each stop, run `axiom autofetch run --cwd <root> --json` once. With autofetch off it does nothing. With it on, enabling it was the user's persistent authorization of that fetch: relay its `notices` beside the stop as information, run no other fetch, and never let them change the stop, its exits or the gate.»

`AGENTS.md:17,20`, `GEMINI.md:48` y `docs/manual-de-inicio.md:227,238,247` reciben el equivalente en castellano. El manual gana además una entrada `axiom autofetch` (subcomandos, hook, intervalo y prohibiciones). `stop-exits.md` no se toca (ver Riesgos, R-2).
**Alternativas descartadas:** una frase distinta en cada texto, que deriva; o reescribir `stop-exits.md`, que exige una delta de `sdd-block-review-gates`.
**Justificación:** una sola frase facilita fijarla en las pruebas (`assets_test.go:1096,1103`, `routing_test.go:119`) y regenerar los 10 goldens (`go test ./internal/components -run TestGolden -update`) en un único commit.

## Flujo de datos

```
SessionStart (Claude/Codex) ─┐
doctrina antes de parada ────┼─→ axiom autofetch run ──→ cli.runAutofetch ──→ engine.Autofetch
CLI de la persona ───────────┘                                                   │
                                                                                 ├─ IsAxiomProject/Locate/ficheros (sin git)
                                                                                 ├─ hubPrefs.Autofetch(maestra) ── apagado ─→ resultado «disabled», exit 0
                                                                                 ├─ repositoryFacts(planning)   (git local)
                                                                                 ├─ sello state.json ── dentro del intervalo ─→ omite la red
                                                                                 ├─ writer.FetchWithin(main, origin, 8 s)   ← único fetch de run
                                                                                 ├─ cambio actual (ReadHead + registry.List)
                                                                                 ├─ relevancia (CommitFiles, TreeDirs, AheadBehind)  ← antes del avance
                                                                                 ├─ avance (Worktrees, DirtyPaths, IsAncestor, writer.FastForward)
                                                                                 └─ AutofetchResult ─→ JSON | texto | hookSpecificOutput

axiom branch ensure ──→ Engine.Ensure ──→ prefs.Autofetch ──→ checkPlan(fetchImplicit)
                                                         └─ writer.Fetch (30 s); un fallo es un aviso y `fetched:false`
```

## Cambios por fichero

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/hub/types.go` | Modificar | Campo `Autofetch bool \`json:"autofetch,omitempty"\`` |
| `internal/hub/prefs.go` | Modificar | `LookupAutofetch`, `SetAutofetch` (no escribe sin cambio) y `ErrProjectNotRegistered` |
| `internal/hub/prefs_autofetch_test.go` | Crear | Persistencia, clave ausente, idempotencia, re-registro, desregistro, hub ilegible y binario anterior (unmarshal en un struct sin el campo) |
| `internal/branch/ports.go` | Modificar | `Git`: `FetchWithin`, `FastForward`; `Prefs`: `Autofetch`; comentario de operaciones remotas |
| `internal/branch/autofetch.go` | Crear | `AutofetchResult`, avisos, preferencia, `WriteText` y constantes de esquema |
| `internal/branch/archive_refusal.go` | Modificar | `ChangeNotIntegrated(..., fetched, autofetch bool)` sin `--fetch` cuando el interruptor está encendido |
| `internal/branch/gitcli/ops.go` | Modificar | `FetchWithin` y `FastForward` |
| `internal/branch/gitcli/ops_write_test.go` | Modificar | Pruebas propias de las dos primitivas (argumentos exactos, sin `--prune`, `ErrTimeout`, rechazo de la fusión no ff) |
| `internal/branch/engine/engine.go` | Modificar | `gitWriter` ampliado, campo `now` y comentario del paquete |
| `internal/branch/engine/prefs.go` | Modificar | `hubPrefs{path}` con `Autofetch`; `DefaultWithHubPath` |
| `internal/branch/engine/ensure.go` | Modificar | `fetchMode`, fetch implícito con aviso, lectura de la preferencia |
| `internal/branch/engine/archive.go` | Modificar | Propaga `autofetch` a `ChangeNotIntegrated` |
| `internal/branch/engine/autofetch.go` | Crear | `Autofetch`, `AutofetchProject` y el ciclo |
| `internal/branch/engine/autofetch_stamp.go` | Crear | Lectura y escritura atómica del sello |
| `internal/branch/engine/autofetch_relevance.go` | Crear | Cambio actual, prefijos y avisos (a) y (b) |
| `internal/branch/engine/autofetch_advance.go` | Crear | Tabla de avance de D-08 |
| `internal/branch/engine/ensure_mode_test.go` | Modificar | El doble de `Prefs` implementa `Autofetch` |
| `internal/branch/engine/ensure_autofetch_test.go` | Crear | Escenarios de RF-BRA-10 y RF-BRA-13 con el interruptor |
| `internal/branch/engine/autofetch_test.go` | Crear | Escenarios de RF-BRA-39 con repositorios git reales y un remoto local desnudo |
| `internal/cli/autofetch.go` | Crear | `RunAutofetch(args, stdin, stdout, stderr)`, banderas, ayuda, modo hook y presupuesto |
| `internal/cli/autofetch_test.go` | Crear | Subcomandos, salidas y códigos (hub temporal con `DefaultWithHubPath`) |
| `cmd/axiom/main.go` | Modificar | `case "autofetch"` y la línea en `printHelp` |
| `internal/components/sdd/inject.go` | Modificar | `AutofetchHookCommand(agent, goos)`, `ensureAutofetchHookEntries` y las llamadas para Claude y Codex |
| `internal/components/sdd/*_test.go` | Modificar/Crear | Instalación, idempotencia byte a byte, conservación de lo ajeno y ausencia en otros runtimes |
| `internal/components/uninstall/service.go` | Modificar | Retirar `axiom autofetch run` |
| `internal/components/uninstall/*_test.go` | Modificar | Retirada con entradas ajenas conservadas |
| `internal/components/agentguidance/routing.go` | Modificar | Frase canónica |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modificar | Frase canónica y párrafo de antes de cada parada |
| `internal/assets/skills/sdd-apply/SKILL.md` | Modificar | Frase canónica |
| `internal/assets/assets_test.go`, `internal/components/agentguidance/routing_test.go` | Modificar | Fijan la frase nueva y el párrafo |
| `testdata/golden/*.golden` (10) | Regenerar | Los que contienen la regla y el párrafo de paradas |
| `AGENTS.md`, `GEMINI.md`, `docs/manual-de-inicio.md` | Modificar | Política local y manual |

## Pruebas

Solo pruebas localizadas por paquete con `-run`; la batería entera queda para la CI. No se compila ningún binario: la CLI se prueba en proceso (`cli.RunAutofetch`) y la orden del hook se prueba por su composición.

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unidad (`hub`) | RF-HUB-07 y RF-HUB-13 en el hub | `hub.NewManager(<tmp>/workspaces.json)`; comparación de bytes antes y después; `json.Unmarshal` en un struct sin `Autofetch` para el binario anterior |
| Unidad (`gitcli`) | `FetchWithin`, `FastForward` | `gitcommon.Observe` registra los argumentos exactos; un remoto inexistente (`file://`); una divergencia rechazada sin fusión |
| Motor (`engine`, `ensure`) | Fetch implícito, fallo con aviso, `--fetch` explícito con `fetch_failed`, `done` sin fetch, `--archive` sin sugerir `--fetch`, intervalo ignorado, código sin avance local | Repositorios reales con un remoto desnudo local; doble de `Prefs` con `Autofetch=true`; `Observe` |
| Motor (`engine`, `run`) | Los 15 escenarios de RF-BRA-39 | Igual que la fila anterior. `autofetchInterval` y el sello se fijan en la prueba. El tiempo agotado se simula con un doble de `gitWriter` que devuelve `ErrTimeout`. Multirepo con repositorios de código cuyo remoto tiene commits nuevos: se comprueba que no hay fetch ni movimiento |
| CLI (`cli`) | Subcomandos, `--json`, `--agent` sin salida o con `hookSpecificOutput`, códigos 0/1/2, proyecto no registrado, hub ilegible, `branch --help` sin `autofetch` | Hub temporal mediante `DefaultWithHubPath` y la dependencia del hub de `runAutofetch`. Sin `t.Parallel`, porque las pruebas de `internal/cli` comparten el HOME de `TestMain`: nunca se escribe en `GIT_CONFIG_GLOBAL` ni se supone un hub vacío |
| Componentes (`sdd`, `uninstall`) | RF-BRA-40: instalación en Claude y Codex, segunda sincronización byte a byte igual, entradas ajenas y `branch guard` intactos, otros runtimes sin hook, retirada | Ficheros temporales de configuración |
| Doctrina (`assets`, `agentguidance`, goldens) | Frase canónica y párrafo de paradas | Fijación de texto y `go test ./internal/components -run TestGolden -update` |
| Arquitectura | Sin regresión | Se ejecutan con `-run` `TestArchitectureOnlyExecGoLaunchesProcesses` y `TestIsolationBranchPackageNeverPassesPushOrCommitToGit`, sin cambiarlas |

### Cobertura de la spec

| Requisito o escenario | Diseño | Prueba |
|---|---|---|
| RF-BRA-39, apagado sin red ni escritura | D-03 (sin git), D-09 | `engine` + `Observe` |
| RF-BRA-39, ff con el árbol limpio; ya al día | D-08 | `engine` |
| RF-BRA-39, árbol sucio, divergencia, rama no desplegada | D-08 (avisos con la orden) | `engine` |
| RF-BRA-39, la rama del cambio no se mueve y avisa | D-07 (b), sin escrituras | `engine` |
| RF-BRA-39, `main_branch` y repositorios de código; multirepo | D-05 | `engine` multirepo + `Observe` |
| RF-BRA-39, sin push ni prune | D-02 | `Observe` |
| RF-BRA-39, remoto inalcanzable | D-06 y D-09 (exit 0, aviso, sin avance) | `engine` + `cli` |
| RF-BRA-39, intervalo (omite y repite) | D-06 | `engine` |
| RF-BRA-39, relevancia (cambio, spec, ODD, nada relevante) | D-07 | `engine` |
| RF-BRA-40, instalación, idempotencia, retirada y otros runtimes | D-10 | `sdd`, `uninstall` |
| RF-BRA-40, exit 0, inerte fuera de proyecto y con el interruptor apagado | D-03, D-09, D-10 | `cli` + composición de la orden |
| RF-BRA-40, contexto adicional solo con avisos | D-09 | `cli` |
| RF-BRA-40, doctrina | D-11 | `assets`, `agentguidance`, goldens |
| RF-BRA-01, los cinco verbos y la ayuda sin `autofetch` | D-01 | `cli` (`branch --help`) |
| RF-BRA-01, con el interruptor encendido no se publica; `scope` sin red | D-04 (solo `ensure` lee el interruptor) | `engine` + `Observe` |
| RF-BRA-10, todos los escenarios nuevos | D-04 | `engine` (`ensure_autofetch_test.go`) |
| RF-BRA-13, fetch implícito y rechazo sin `--fetch` | D-04 | `engine` |
| RF-HUB-07 y RF-HUB-13, todos | D-03, D-09 | `hub`, `cli` |

## Matriz de amenazas

El cambio lanza subprocesos (git), automatiza control de versiones (fetch y fast-forward) e integra procesos (hook de sesión), así que la matriz aplica.

| Frontera | Aplicabilidad | Respuesta de diseño | Pruebas RED previstas |
|---|---|---|---|
| Rutas con aspecto de documentación | N/A: no se clasifican ni se ejecutan ficheros; los prefijos de relevancia solo filtran rutas que devuelve `git log` | — | — |
| Selección del repositorio git | **Aplicable** | Todo git va con `cmd.Dir` absoluto (`RunWithTimeout` lo exige) sobre el checkout principal del repositorio de specs (fetch) o el worktree que tiene desplegada la integración (avance), nunca el directorio del proceso. `gitEnvironment` retira `GIT_DIR` y `GIT_WORK_TREE`. El `cwd` del hook solo selecciona el proyecto. | (1) `run` desde un worktree de feature avanza la integración en el checkout principal y no toca el worktree; (2) `GIT_DIR` definido en el entorno de la prueba no cambia el repositorio; (3) un `--cwd` relativo se rechaza como error de uso; (4) en multirepo, `cwd` en un repositorio de código: el fetch va solo al de specs |
| Estado de commits | **Aplicable** (índice y árbol en el avance) | Con `DirtyPaths` no vacío (incluye preparados y sin seguimiento) no se ejecuta `merge`. `merge --ff-only` nunca crea un commit. | (1) un fichero preparado bloquea el avance; (2) un fichero sin seguimiento lo bloquea; (3) con el árbol limpio, el avance no crea commits (`RevList` igual al remoto y `HEAD` sin un padre nuevo) |
| Estado de push | **Aplicable** como prohibición | El puerto no tiene `push`. El fetch no lleva `--prune` ni refspec. Ninguna ruta de `run` o `ensure` escribe en el remoto. | (1) `Observe` no ve `push`, `pull`, `ls-remote` ni `--prune` en `run` encendido; (2) `isolation_test` sigue verde |
| Órdenes de PR | N/A: no se crean PR ni se componen órdenes `gh`; las órdenes de los avisos son texto para la persona y nunca se ejecutan | — | — |
| Integración de procesos (hook) | **Aplicable** (fila añadida por el diseño) | Orden constante con `\|\| true` o `; exit 0`. `run` en modo hook siempre devuelve 0, con `recover` y un presupuesto de 12 s. Es inerte sin proyecto Axiom o con el interruptor apagado (sin git). La entrada estándar se lee con un límite de 256 KiB. | (1) la orden compuesta termina en `\|\| true` o `; exit 0` para cada runtime y sistema; (2) `runAutofetch --agent` fuera de un proyecto Axiom: salida vacía, 0 y `Observe` vacío; (3) interruptor apagado: lo mismo; (4) una carga de entrada ilegible o mayor de 256 KiB no falla; (5) un pánico simulado devuelve 0 |

## Migración y despliegue

No hace falta migrar nada. El campo `autofetch` es `omitempty`, el esquema del hub sigue en `"1.0"` y un binario anterior lo ignora. El hook se instala en el siguiente `axiom sync` y es inerte mientras nadie active el interruptor. Si hay que revertir, `axiom autofetch disable` desactiva el comportamiento al momento, y la reversión de la PR más `axiom sync` o la desinstalación retiran el hook. Con un binario que ya no conoce el hook, la entrada se borra a mano (proposal, «Plan de reversión»). Codex pedirá confiar una vez en la nueva entrada `SessionStart`.

## Plan de unidades de trabajo (un commit por unidad, para `sdd-tasks`)

| # | Unidad | Ficheros principales | Líneas estimadas |
|---|---|---|---|
| WU-1 | Preferencia `autofetch` en el hub y puerto `Prefs.Autofetch` (RF-HUB-07, parte de RF-HUB-13) | `hub/types.go`, `hub/prefs.go`, `branch/ports.go` (Prefs), `engine/prefs.go`, `engine/ensure_mode_test.go`, `hub/prefs_autofetch_test.go` | ~230 |
| WU-2 | Primitivas `FetchWithin` y `FastForward` | `gitcli/ops.go`, `branch/ports.go` (Git), `engine/engine.go` (gitWriter), `gitcli/ops_write_test.go` | ~170 |
| WU-3 | Fetch implícito en `ensure` y `ensure --archive` (RF-BRA-10, RF-BRA-13, RF-BRA-01) | `engine/ensure.go`, `engine/archive.go`, `branch/archive_refusal.go`, `engine/ensure_autofetch_test.go` | ~320 |
| WU-4 | Ciclo `Engine.Autofetch` y resultado tipado (RF-BRA-39) | `branch/autofetch.go`, `engine/autofetch*.go`, `engine/autofetch_test.go` | ~950 |
| WU-5 | Verbo `axiom autofetch` (RF-HUB-13, RF-BRA-39, modo hook de RF-BRA-40) | `cli/autofetch.go`, `cmd/axiom/main.go`, `cli/autofetch_test.go` | ~480 |
| WU-6 | Hook `SessionStart` en Claude Code y Codex, y retirada (RF-BRA-40) | `components/sdd/inject.go`, `components/uninstall/service.go` y sus pruebas | ~300 |
| WU-7 | Doctrina, documentación y goldens (RF-BRA-40) | `routing.go`, `sdd-orchestrator-sections.md`, `sdd-apply/SKILL.md`, pruebas de texto, 10 goldens, `AGENTS.md`, `GEMINI.md`, `docs/manual-de-inicio.md` | ~160 escritas + goldens regenerados |

Total estimado: unas 2.600 líneas escritas (sin los goldens), en una sola PR (`single-pr`). Dependencias: WU-1 y WU-2 son independientes; WU-3 depende de WU-1 y WU-2; WU-4 depende de WU-1 y WU-2; WU-5 depende de WU-4; WU-6 y WU-7 son independientes del motor, pero WU-7 cita el verbo, así que conviene después de WU-5.

## Riesgos y supuestos

- **R-1 (interpretación de RF-HUB-07, «con aviso en las órdenes que lo consultan»).** `ensure` consulta el interruptor, pero no añade el aviso de proyecto no registrado: cambiaría la salida de todo `ensure` en proyectos sin registrar y rompería la promesa de que, con el interruptor apagado, el comportamiento es idéntico. El hub ilegible ya produce el aviso de `WorktreePrefs`. El aviso de proyecto no registrado sale en `autofetch status|run`. Hay que confirmarlo en la puerta de diseño.
- **R-2 (doctrina y `stop-exits.md`).** `stop-exits.md` dice que `fetch` es de la persona y que el agente nunca lo ejecuta. El párrafo nuevo de antes de cada parada ordena ejecutar `axiom autofetch run`, que con el interruptor encendido sí hace fetch. Se resuelve con una excepción explícita en `sdd-orchestrator-sections.md` («enabling it was the user's persistent authorization of that fetch»), sin tocar `stop-exits.md`. Si la revisión considera la frase de `stop-exits.md` contradictoria, hará falta una delta de `sdd-block-review-gates`.
- **R-3 (repositorio de specs de un multirepo en `ensure`).** El fetch implícito usa los mismos repositorios que `--fetch` (los que tienen ramas). El repositorio de specs in situ de un multirepo no tiene base ni `fetched` en el JSON y no se trae en `ensure`: lo cubre `run`. RF-BRA-10 dice «todos los repositorios del plan». Su escenario solo nombra los de código y se cumple, pero la redacción literal puede pedir traer también el de specs.
- **R-4 (cambio actual).** Se resuelve por la rama de `HEAD` en el registro y, si no coincide, por el único cambio activo registrado. Con varios cambios activos y `cwd` en la integración (habitual en el repositorio de specs de un multirepo), no hay avisos de relevancia. La spec lo admite («sin cambio actual resoluble»).
- **R-5 (forma de salida de `SessionStart` en Codex).** La salida `hookSpecificOutput.additionalContext` sigue la documentación de Claude Code. En Codex se supone el mismo contrato, igual que el hook de guarda se instaló sin verificarlo en ejecución (A-2). Hay que validarlo a mano en Codex.
- **R-6 (retirada por `sync`).** No existe hoy una retirada en `sync` para los hooks de Axiom distinta de la desinstalación. «Una ejecución posterior de `axiom sync` que ya no deba instalarlo» se cubre con el mismo camino de retirada que usa el hook de `axiom branch guard`. Si la spec pide una retirada propia de `sync`, hará falta un mecanismo nuevo.
- **R-7 («el hook no lanza procesos» fuera de proyectos Axiom).** El runtime siempre lanza el proceso `axiom`. Se interpreta, como dice el cuerpo de RF-BRA-40, como «no lanza procesos de git».
- **R-8 (órdenes de los avisos).** Para la divergencia se propone `rebase origin/<int>` y, para la rama no desplegada, `fetch . refs/remotes/origin/<int>:refs/heads/<int>` (solo local). Son decisiones de redacción revisables en `sdd-tasks` sin cambiar la arquitectura.
- **R-9 (concurrencia).** Dos sesiones que arrancan a la vez pueden traer el remoto dos veces. Es inocuo: un choque de bloqueo de git se degrada a aviso y no se reintenta.

## Preguntas abiertas

- [ ] R-1: ¿basta el aviso de proyecto no registrado en `autofetch status|run`, sin añadirlo a `ensure`?
- [ ] R-2: ¿se acepta la excepción en `sdd-orchestrator-sections.md` sin tocar `stop-exits.md`?
- [ ] R-3: ¿el fetch implícito debe traer también el repositorio de specs in situ de un multirepo?
