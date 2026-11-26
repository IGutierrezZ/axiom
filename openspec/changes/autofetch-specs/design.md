# Diseño: Autofetch del repositorio de specs — avisar y avanzar (C-05-d)

> **Cambio:** `autofetch-specs`. **Idioma:** castellano de España; identificadores, comandos, claves y rutas en inglés.
> **Entrada:** `proposal.md`, `explore.md` y las deltas `specs/change-branch-lifecycle/spec.md` (RF-BRA-39, RF-BRA-40; MODIFIED RF-BRA-01, RF-BRA-10, RF-BRA-13) y `specs/multi-project-hub/spec.md` (RF-HUB-13; MODIFIED RF-HUB-07).
> **Contexto técnico mantenido:** el proyecto no lo declara (`axiom workspace context` devuelve `technicalContext: null`), así que no hay `context-impact.md`.

## Enfoque técnico

Se sigue el enfoque híbrido C de la exploración, sin abrir un paquete nuevo de git:

1. **Preferencia en el hub.** `hub.WorkspaceRecord` gana `Autofetch bool` (`json:"autofetch,omitempty"`), con `LookupAutofetch` y `SetAutofetch` en `internal/hub/prefs.go`. El esquema sigue en `"1.0"`. El puerto `branch.Prefs` gana `Autofetch(projectMaster)`, que implementa `engine.hubPrefs` (RF-HUB-07, RF-HUB-13).
2. **Primitivas de git.** `internal/branch/gitcli` gana `FetchWithin` (fetch con tiempo máximo propio, sobre `gitcommon.RunWithTimeout`) y `FastForward` (`merge --ff-only <ref>`). Ambas pasan por el único punto de ejecución (`gitcommon`) y por los puertos `branch.Git` y `engine.gitWriter` (RF-BRA-01).
3. **Fetch implícito en `ensure`.** `Engine.Ensure` lee la preferencia del proyecto y, si está encendida y no hay `--fetch`, trae `origin` en todos los repositorios del plan (los que tienen ramas y el de specs in situ de un multirepo), con `FetchWithin` de 8 s por repositorio y omitiendo sin aviso los que no tienen `origin`. Un fallo del fetch implícito es un aviso, no `fetch_failed` (RF-BRA-10, RF-BRA-13).
4. **Ciclo `run`.** Un método nuevo, `Engine.Autofetch`, en `internal/branch/engine/autofetch*.go`, hace el fetch acotado del repositorio de specs con intervalo mínimo, calcula la relevancia **antes** de avanzar, avanza la rama de integración solo por fast-forward estricto y devuelve un resultado tipado `branch.AutofetchResult` (RF-BRA-39).
5. **Verbo propio.** `axiom autofetch enable|disable|status|run` en `internal/cli/autofetch.go`, con un `case "autofetch"` en `cmd/axiom/main.go`. La familia `axiom branch` conserva sus cinco verbos (RF-BRA-01, RF-HUB-13).
6. **Hook de inicio de sesión.** `internal/components/sdd/inject.go` instala una entrada `SessionStart` en Claude Code (`settings.json`) y en Codex (`hooks.json`), con el patrón de `ensureBranchGuardHookEntries`; cada `axiom sync` poda antes las entradas de `axiom autofetch run` que ya no son la canónica. `internal/components/uninstall/service.go` la retira (RF-BRA-40).
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

- `RemoteBranches(ctx, dir, prefix string) ([]string, error)`: lectura local `git for-each-ref --format=%(refname) refs/remotes/origin/<prefix>/`, sin red (`for-each-ref` no es un subcomando remoto para `remoteSubcommand`). Solo la usa la resolución del cambio cedido de D-07.

`Fetch` no cambia: `ensure --fetch` conserva su límite de 30 s. El fetch implícito de `ensure` y el de `run` usan `FetchWithin` con 8 s. `FetchWithin` y `FastForward` entran en `branch.Git` y en `engine.gitWriter`, que es el puerto que puede mutar; `RemoteBranches` entra en `branch.Git` y en `engine.gitReader`. Los comentarios de `ports.go:9,40` y `engine.go:62-69` pasan a decir que `Fetch` y `FetchWithin` son las únicas operaciones remotas y que `FastForward` es la única escritura de ramas fuera de crear, desplegar y borrar.
**Alternativas descartadas:** (a) `git fetch origin <int>:<int>` para avanzar sin checkout: elude la comprobación del árbol limpio y contradice la política «solo `merge --ff-only`»; (b) `update-ref`: no comprueba que el avance sea fast-forward ni toca el árbol de trabajo.
**Justificación:** `merge --ff-only` no está entre los literales vigilados (`push`, `commit`) ni es un subcomando remoto para `remoteSubcommand`. Con la rama desplegada, `merge --ff-only` actualiza a la vez la rama, el índice y el árbol, y se niega a crear una fusión.

### D-03: El interruptor se lee del hub por la identidad del proyecto, sin lanzar git cuando está apagado

**Elección:**

- `hub.(*Manager).LookupAutofetch(projectPath string) (enabled, registered bool, err error)`.
- `hub.(*Manager).SetAutofetch(idOrPath string, enabled bool) (rec *WorkspaceRecord, changed bool, err error)`. No escribe si el valor no cambia, así que `enable` y `disable` repetidos dejan `workspaces.json` byte a byte igual. Si el proyecto no está registrado, devuelve un error que cumple `errors.Is(err, hub.ErrProjectNotRegistered)` y no escribe. Un hub ilegible devuelve el error de `Load` sin escribir.
- `branch.Prefs.Autofetch(projectMaster string) (enabled bool, state AutofetchPrefState, warning string)`, con `state` ∈ `enabled|disabled|unregistered|unreadable`. Un hub ilegible o un proyecto no registrado dan `enabled=false` y un texto de aviso en castellano; el del proyecto no registrado indica `axiom project add "<maestra>"`. El puerto solo **devuelve** el aviso: cada orden decide si lo muestra, según esta tabla (RF-HUB-07, RF-HUB-13, RF-BRA-39):

  | Orden | Proyecto no registrado | Hub ilegible |
  |---|---|---|
  | `axiom autofetch status` | `disabled` con aviso; exit 0 | `disabled` con aviso; exit 0 |
  | `axiom autofetch run` sin `--agent` | `disabled` con aviso; sin red ni escritura; exit 0 | igual |
  | `axiom autofetch run --agent …` (hook) | como apagado: sin salida, sin aviso, sin git; exit 0 | igual |
  | `axiom autofetch enable` | rechazo con `axiom project add "<maestra>"`; no escribe; exit 1 | rechazo sin modificar el hub; exit 1 |
  | `axiom autofetch disable` | «no hay nada que apagar»; no crea ni escribe el hub; exit 0 | rechazo sin modificar el hub; exit 1 |
  | `axiom branch ensure` | como apagado: sin aviso nuevo y sin red por esta causa | como apagado: sin aviso nuevo (el aviso de `WorktreePrefs` por hub ilegible que `ensure` ya emite hoy no cambia) |

- La identidad es la carpeta maestra del checkout principal (`projectMaster`, la misma que usan las preferencias de worktree). En `run`, `status`, `enable` y `disable` se resuelve **solo con ficheros**: `workspace.IsAxiomProject`, `workspace.Locate` (que sigue el puntero `.axiom-workspace` de un multirepo hasta su maestra), `gitcommon.GitRootFromFiles`, `gitcommon.CommonDirFromFiles` y `gitcommon.MainWorktreeFromFiles`. Desde un worktree de feature o un subdirectorio, la maestra se traslada al checkout principal con la misma regla que `projectMaster` (misma ruta relativa dentro del checkout principal). Así, con el interruptor apagado o fuera de un proyecto Axiom, no se lanza ningún proceso de git. Si los ficheros no bastan para resolverla, se usa `location.Master`.
- **Normalización de la identidad (hueco 8).** `LookupAutofetch` y `SetAutofetch` buscan la entrada con `sameProject` (`internal/hub/prefs.go`), igual que `LookupWorktreePrefs` y `SetWorktreePrefs`: identidad de directorio (`pathidentity.SameDirectory`, que cubre nombres cortos 8.3 y enlaces) y, si alguna ruta no existe, `filepath.Clean` con `strings.EqualFold` en Windows. Es coherente con la coincidencia por `strings.EqualFold(filepath.Clean(...))` de `hub.Register`, de modo que la entrada que `enable` escribe es la misma que `Register` reconoce al re-registrar. Pruebas declaradas: (1) `enable` desde un worktree de feature enlazado (`<maestra>-wt/<cambio>/<repo>`) y desde un subdirectorio escribe en la entrada de la maestra y `status` desde el checkout principal informa de `enabled`; (2) en Windows, la entrada registrada como `C:\Repos\X` se encuentra desde `c:\repos\x`; (3) en multirepo, `status` desde un repositorio de código con puntero `.axiom-workspace` lee la entrada de la maestra.
- `axiom.yaml` no se consulta nunca para el interruptor. Una clave `autofetch` en `axiom.yaml` se ignora (`workspace.WorkspaceConfig` no la modela).

**Alternativas descartadas:** guardar el interruptor en `axiom.yaml`, que la proposal deja fuera de alcance; o resolver la identidad con `git rev-parse`, que lanzaría procesos aunque el interruptor esté apagado y contradice RF-BRA-40.
**Justificación:** es el mismo patrón que las preferencias de worktree (D-08 de INC-25). `Register` solo reescribe nombre, topología, acceso y configuración, `Unregister` borra la entrada y `SetWorktreePrefs` solo toca sus dos campos. Por eso el nuevo campo sobrevive al re-registro y al dashboard, y desaparece al desregistrar, sin tocar esas funciones.

### D-04: `ensure` con el interruptor encendido es un `--fetch` implícito que degrada a aviso

**Elección:** `checkPlan` y `checkRepository` reciben `fetchMode` (`fetchNone | fetchExplicit | fetchImplicit`) en lugar del booleano:

- `fetchExplicit` (`--fetch`): sin cambios. Un fallo devuelve `fetch_failed` y no crea nada, también con el interruptor encendido.
- `fetchImplicit` (sin `--fetch`, con `prefs.Autofetch == true`):
  - **Repositorios:** todos los del plan. Los que tienen ramas, en `checkRepository`, en el mismo punto que `--fetch` (antes de `resolveBase`). El de specs in situ de un multirepo (`RepoSpecsInPlace`, sin ramas), en `checkPlan` antes de `checkSpecsRepository`: solo fetch, sin base ni avance.
  - **Sin `origin`:** antes de traer se consulta `RemoteConfigured(<checkout principal>, "origin")` (puerto ya existente, sin red). Si es `false`, el repositorio se omite sin aviso y `fetched` queda en `false`. Si la consulta falla, se trata como un fetch fallido (aviso y `fetched:false`).
  - **Tiempo máximo:** `writer.FetchWithin(<checkout principal>, "origin", autofetchFetchTimeout)`, 8 s por repositorio, en serie y en el orden del plan. Con la red caída, un plan de N repositorios espera como mucho unos 8·N s; la spec fija la cota por repositorio.
  - **`--base <ref>`:** no cambia nada; el fetch se hace igual, antes de resolver la base sustituida (RF-BRA-12).
  - **Fallo:** se añade al resultado el aviso `el autofetch no pudo traer origin en <repo> (<resumen>): se sigue con la base local`, `fetched` queda en `false` y la ejecución sigue.
  - **JSON:** los repositorios con base siguen informando en `base.fetched`. El de specs in situ, que no tiene base, informa en un campo nuevo `RepositoryResult.Fetched *bool \`json:"fetched,omitempty"\``, que solo se rellena con `fetchImplicit`; con el interruptor apagado la salida de `ensure` queda byte a byte igual que hoy.
  - `--fetch` explícito no cambia su conjunto de repositorios ni su límite (30 s): el repositorio de specs in situ sigue sin traerse con `--fetch`, para no convertir en `fetch_failed` un caso que hoy procede (ver R-3).
- La preferencia se lee en `Ensure` después de `resolveMode` y de planificar la promoción, de modo que `decision_required` sigue llegando antes de cualquier red: un `ensure` que termina en `decision_required` no hace fetch, y el `ensure --mode <respuesta>` siguiente sí. `done`, `status` y `scope` no la leen.
- `ensure` no consulta el sello del intervalo (D-06) y no avanza ninguna rama local: `Fetch` solo actualiza `refs/remotes/origin/*`.
- `ensure --archive` comparte `checkRepository`, así que hereda el fetch implícito. `branch.ChangeNotIntegrated` gana el parámetro `autofetch bool`. Si es verdadero, ni el mensaje ni la orden de reintento llevan `--fetch`:
  - fetch implícito correcto: «… ni tras el fetch del autofetch …»;
  - fetch implícito fallido: «… el autofetch no pudo traer el remoto; comprueba la red y repite `axiom branch ensure --archive …` …».

  `ChangeIntermediateMerge` no cambia, porque ya no sugiere `--fetch`.
- Los avisos de preferencia de `Autofetch` (proyecto no registrado, hub ilegible) **no** se añaden a `ensure`: la spec corregida (RF-HUB-07) ordena que `ensure` se comporte exactamente como con el interruptor apagado y con la salida sin cambios. El aviso de `WorktreePrefs` por hub ilegible que `ensure` ya emite hoy sigue igual, porque es previo y ajeno a este cambio (tabla de D-03).

**Alternativas descartadas:** rechazar con `fetch_failed` también en el fetch implícito, que bloquearía `ensure` cada vez que falla la red (la proposal recomienda avisar y seguir); o hacer que `ensure` ejecute el ciclo completo de `run`, que avanzaría la integración desde `ensure` y respetaría el intervalo, en contra de RF-BRA-10.
**Justificación:** es el cambio mínimo sobre una ruta ya probada. Las pruebas de RF-BRA-10 que construyen el motor sin `prefs` (`prefs == nil` equivale a apagado) siguen pasando sin cambios.

### D-05: El repositorio de specs y la rama que se avanza

**Elección:** el repositorio de specs es el del planning home (`gather`/`repositoryFacts` sobre `location.WorkspaceRoot`). En `monorepo-embedded` es el propio repositorio; en `multirepo`, el de specs in situ. La rama que se avanza es `branch.EffectivePolicy(policy, planning).IntegrationBranch`. `MainBranch` solo es intocable cuando **difiere** de `IntegrationBranch`: entonces ninguna orden de `run` escribe su rama local (ningún `merge` ni otra escritura; el `fetch origin` solo actualiza `refs/remotes/origin/*`), y una prueba lo fija con `Observe` y comparando su SHA. Cuando coinciden (monorepo de una sola etapa, o el `main` del repositorio de specs de un multirepo, que hace de integración), avanzarla por fast-forward estricto **es** el avance de la integración del paso 2 de RF-BRA-39, con las mismas condiciones de D-08; no hay una regla aparte para `main_branch`. El fetch y el avance se ejecutan solo en ese repositorio. Los repositorios de código no reciben ninguna orden remota ni de escritura: solo lecturas locales para el aviso (b) de D-07.
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

1. **Cambio actual.** Se resuelve en este orden y se para en la primera regla que da un resultado:
   1. Se lee `HEAD` del checkout de `cwd` (`gitcommon.ReadHead`, sin procesos). Es el cambio activo registrado (carriles SDD y ODD, `registry.List`) que tiene esa rama (de cambio, de rol o de archivado) para el repositorio de `cwd`.
   2. Si ninguna rama coincide y hay **exactamente un** cambio activo registrado en el planning home, se toma ese.
   3. **Cesión sin registro ni rama local (hueco 5).** Si no hay **ningún** cambio activo registrado, se listan con `RemoteBranches(<repo de specs>, <branch_prefix>)` las ramas `origin/<prefijo>/<c>` (sin el sufijo `-archive`) que no tienen rama local `refs/heads/<prefijo>/<c>` y cuyo árbol contiene `<planRel>/changes/<c>/kickoff.yaml` (`FileAt`, lectura local). Si hay **exactamente una**, `<c>` es el cambio actual (carril SDD) y su rama esperada es `<prefijo>/<c>`. El listado se acota a las 50 primeras referencias en orden de `for-each-ref`.
   4. Si no hay ninguno, o hay varios (varios registrados sin coincidencia en `HEAD`, o varias cesiones candidatas en la regla 3), no hay cambio actual: no se emiten (a) ni (b), y sí se hacen el fetch, el avance y los avisos de no-avance de D-08 (RF-BRA-39 paso 3, «sin cambio actual resoluble»).
2. **Prefijos relevantes**, relativos a la raíz git del repositorio de specs:
   - SDD: `<planRel>/changes/<c>/` y `<planRel>/specs/<cap>/` por cada `<cap>` de `TreeDirs(<planRel>/changes/<c>/specs)`. Se toma la unión de lo que hay en la rama del cambio (`refs/heads/<rama>`, si existe en el repositorio de specs) y en `refs/remotes/origin/<int>`.
   - ODD: `<wsRel>/odd/tasks/<c>.md`.
   - `<planRel>` es `relativeWithin(planning.GitRoot, location.PlanningHome)` y `<wsRel>` es `relativeWithin(planning.GitRoot, location.WorkspaceRoot)`.
3. **Aviso (a), commits entrantes en la integración.** Se calcula **antes** del avance con `CommitFiles(dir, "refs/heads/<int>..refs/remotes/origin/<int>")`. Cuentan los commits que no son de fusión y tienen al menos una ruta bajo un prefijo relevante. Las fusiones no traen rutas y sus padres ya están en el rango. Se emite un aviso por la rama `origin/<int>`, con el número de commits, el alcance (los prefijos tocados) y la orden `git -C "<checkout>" log --oneline <sha local previo>..origin/<int> -- <prefijos>`, que sigue siendo válida después del avance. Si la rama local de integración no existe, (a) se omite.
4. **Aviso (b), commits nuevos en la rama del cambio.** Para cada rama del registro del cambio actual, en cada repositorio de su registro, si existen `refs/heads/<b>` y `refs/remotes/origin/<b>`, se llama a `AheadBehind(dir, "refs/heads/<b>", "refs/remotes/origin/<b>")`. El valor `ahead` son los commits del remoto que la rama local no tiene. Si es mayor que 0, se emite un aviso con la orden `git -C "<worktree o checkout>" merge --ff-only origin/<b>` (sin commits locales propios) o `git -C "<…>" log --oneline <b>..origin/<b>` (con divergencia). Si **no existe** la rama local `<b>` pero sí `refs/remotes/origin/<b>` (cesión pendiente de adoptar, también la de la regla 3 del punto 1), se emite `change_branch_not_adopted` con el número de commits de `RevList(<int local, o origin/<int> si no hay local>..refs/remotes/origin/<b>)` y la orden `axiom branch ensure --change <c> --cwd "<maestra>"`, que adopta la rama publicada (RF-BRA-27). Solo se leen referencias locales: los repositorios de código no se traen (D-05), así que (b) sobre ellos refleja lo que trajo el último `ensure` o fetch de la persona.
   **Límite documentado:** con varios cambios activos registrados y `HEAD` fuera de todas sus ramas, o con varias cesiones candidatas, no se avisa (b) ni (a); el resto del ciclo sí se ejecuta.
5. Los avisos se ordenan de forma determinista: primero la integración y después las ramas del cambio por repositorio y nombre.

**Alternativas descartadas:** calcular la relevancia sobre el delta del fetch (`FETCH_HEAD`), que se pierde cuando el intervalo omite la red y no detecta lo traído por un `ensure`; o calcularla después del avance, cuando el rango ya está vacío.
**Justificación:** el rango «local frente a remota» se deduplica solo. Lo que ya avanzó deja de avisarse en la siguiente ejecución, y lo que no pudo avanzar sigue avisándose mientras siga pendiente.

### D-08: Avance por fast-forward estricto con degradación a aviso

**Elección:** con `L = refs/heads/<int>` y `R = refs/remotes/origin/<int>`. **Regla previa (RF-BRA-39 paso 3):** solo hay avisos de no-avance cuando `L` está **por detrás** de `R`, es decir, cuando `R` tiene commits que `L` no tiene (`L` estrictamente atrasada, o divergida). Si `L` está al día o por delante, no hay ningún aviso de no-avance aunque el árbol esté sucio, haya una operación en curso o `HEAD` esté desacoplado. Las filas se evalúan en orden y gana la primera que aplica:

| # | Situación | Acción | Aviso (`kind`) y orden exacta |
|---|---|---|---|
| 1 | Sin `R`; o sin `L` (la rama local de integración no existe, aunque exista `R`); o `L == R` | nada | ninguno |
| 2 | `R` es ancestro de `L` (la local va por delante) | nada | ninguno |
| 3 | Ni `L` es ancestro de `R` ni `R` de `L` (divergencia) | nada | `integration_diverged`: `git -C "<checkout>" rebase origin/<int>` (`<checkout>`: el que tiene `<int>` desplegada o, si no lo hay, el checkout principal del repositorio de specs) |
| 4 | `L` atrasada y algún worktree del repositorio de specs tiene una **operación git en curso sobre `<int>`**: en su git dir (`gitcommon.GitDirFromFiles`) existe `MERGE_HEAD` o `CHERRY_PICK_HEAD` y su `HEAD` es `<int>`, o existe `rebase-merge/` o `rebase-apply/` cuyo `head-name` es `refs/heads/<int>` | nada | `integration_operation_in_progress` (con la operación: `merge`, `rebase` o `cherry-pick`): `git -C "<ese checkout>" status`, para que la persona termine o aborte la operación; tras ello el siguiente `run` avanza |
| 5 | `L` atrasada, no desplegada en ningún worktree (`Worktrees`, `Branch == <int>`) y el checkout principal del repositorio de specs tiene `HEAD` **desacoplado** (`ReadHead`) | nada | `integration_detached`: `git -C "<checkout principal>" switch <int>`, y después `git -C "<checkout principal>" merge --ff-only origin/<int>` |
| 6 | `L` atrasada y no desplegada en ningún worktree (y el checkout principal en otra rama) | nada | `integration_not_deployed`: `git -C "<checkout principal>" fetch . refs/remotes/origin/<int>:refs/heads/<int>` (ver la decisión de la orden más abajo) |
| 7 | `L` atrasada, desplegada y con `DirtyPaths(<checkout>)` no vacío | nada | `integration_dirty`: `git -C "<checkout>" merge --ff-only origin/<int>`, que la persona ejecuta cuando el árbol esté limpio |
| 8 | `L` atrasada, desplegada, árbol limpio y quedan menos de `autofetchAdvanceReserve` (1 s) del presupuesto de D-09 | nada | `integration_ff_failed` con el motivo «sin tiempo dentro de la cota»: la misma orden que en la fila 7 |
| 9 | `L` atrasada, desplegada y con el árbol limpio | `FastForward(<checkout>, R)` | ninguno; `integration.advanced=true` y los SHA previo y nuevo |
| 10 | `FastForward` falla (`index.lock`, carrera u otro error) | nada más; sin reintento | `integration_ff_failed`: la misma orden que en la fila 7, más el resumen de git |

El `<checkout>` es el worktree que tiene la rama desplegada, sea el principal o uno enlazado; un worktree bloqueado (`git worktree lock`) no cambia nada, porque el bloqueo solo impide podarlo o moverlo. En `monorepo-embedded` suele ser el checkout principal y en `multirepo`, el repositorio de specs in situ. Durante un `rebase`, `git worktree list` informa del checkout como desacoplado; por eso la fila 4 se evalúa antes que las 5 y 6 y busca la rama en `head-name`. Todas las comprobaciones de las filas 4 y 5 son lecturas de ficheros, sin procesos.

**Orden sugerida para la rama no desplegada (fila 6).** Se elige `git fetch . refs/remotes/origin/<int>:refs/heads/<int>` frente a `git switch <int>` seguido de `merge --ff-only`:

- No contradice «solo fast-forward estricto»: el refspec va sin `+`, así que git rechaza cualquier actualización que no sea fast-forward («non-fast-forward»), y se niega a escribir una rama desplegada en cualquier worktree. La fila 6 solo se alcanza cuando `L` es ancestro estricto de `R` (las filas 2 y 3 ya han filtrado el resto) y nadie la tiene desplegada, de modo que no hay árbol que comprobar.
- Es una operación solo local (`.` es el propio repositorio): no hace red ni toca ningún árbol de trabajo.
- `switch` cambiaría la rama del checkout principal de la persona, que puede estar trabajando en otra, y fallaría con el árbol sucio.
- La prohibición de D-02 sobre `fetch <remoto> <int>:<int>` se refiere al avance **automático** de `run`, que sigue siendo solo `merge --ff-only`. Esta orden es texto para la persona y `run` nunca la ejecuta.

Con `HEAD` desacoplado en el checkout principal (fila 5) sí se sugiere `switch`, porque la persona no está en ninguna rama que perder y desplegar `<int>` es la forma de que el siguiente `run` la avance.

**Alternativas descartadas:** avanzar una rama no desplegada con `update-ref` o con `fetch .` de forma automática, que mueve una rama sin pasar por la comprobación de árbol que exige RF-BRA-39; avanzar durante una operación en curso, que mezclaría el avance con un `merge` o `rebase` a medias; o reintentar ante `index.lock`, que puede chocar con `scripts/axiom-worktree.ps1 done`.
**Justificación:** cumple «SOLO por fast-forward estricto y SOLO si está desplegada, limpia, sin divergencia, sin operación en curso y con `HEAD` en la rama». Todo lo demás es un aviso con la orden exacta, solo cuando la integración está por detrás, y el código de salida sigue siendo 0.

### D-09: Contrato de `run` y de los subcomandos de preferencia

**Elección:**

- `run`:
  - Con `--json`, devuelve `branch.AutofetchResult` (esquema `axiom.autofetch-result/v1`).
  - Sin `--json`, imprime un texto en castellano: estado del interruptor, fetch, integración y los avisos con su orden.
  - Con `--agent claude-code|codex` (modo hook), escribe por la salida estándar solo cuando hay avisos: `{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"<avisos>"}}`. Sin avisos, no escribe nada.
  - En modo hook, `cwd` sale de `--cwd`. Si no se da, sale del campo `cwd` de la carga JSON del runtime por la entrada estándar, leída con un límite de 256 KiB y fallando en abierto. Si tampoco está, es el directorio del proceso.
  - Todo `run` va acotado por un presupuesto global `autofetchRunBudget = 10 s` (un `context.WithTimeout` sobre el ciclo entero), protegido con `recover`. Dentro de él, el fetch tiene su propio límite de 8 s (`autofetchFetchTimeout`), y quedan unos 2 s para las lecturas locales y el avance, que en un repositorio de documentación tardan milisegundos. Si el plazo vence a mitad de una lectura, el ciclo deja de calcular avisos, no avanza nada y devuelve lo que tenga con el aviso del fetch o del avance correspondiente. El avance solo se inicia si queda al menos `autofetchAdvanceReserve = 1 s` (fila 8 de D-08). Las tres constantes son variables de paquete para que las pruebas las reduzcan.
  - Proyecto no registrado o hub ilegible: según la tabla de D-03. Sin `--agent`, `disabled` con el aviso en `warnings`; con `--agent`, ninguna salida.
  - Código de salida: 0 siempre, salvo un error de uso en modo CLI (2). En modo hook, también 0 ante un error de uso.
- `enable` y `disable`: `axiom.autofetch-preference/v1` con `{schema, operation, autofetch, project, registered, changed, warnings, message}`. Terminan con 0 si tienen éxito (también cuando no cambia nada). `enable` termina con 1 si el proyecto no está registrado (mensaje en castellano con `axiom project add "<maestra>"`). `disable` sobre un proyecto no registrado termina con 0, `registered:false`, `changed:false` y el mensaje «no hay nada que apagar», sin crear ni escribir `workspaces.json`. Ambos terminan con 1 si el hub es ilegible (sin modificarlo) o si no es un proyecto Axiom. Ninguno escribe `axiom.yaml`.
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
	Fetch       string                `json:"fetch"`                 // done | skipped_interval | failed | timeout | no_origin | not_attempted
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
	State    string `json:"state"` // up_to_date | advanced | ahead | not_advanced | no_remote_ref | no_local_branch
	Advanced bool   `json:"advanced"`
}

type AutofetchChange struct {
	Lane branch.Lane `json:"lane"`
	Name string      `json:"name"`
}

type AutofetchNotice struct {
	Kind    string   `json:"kind"`    // incoming_relevant | change_branch_behind | change_branch_not_adopted |
	                                   // integration_diverged | integration_operation_in_progress | integration_detached |
	                                   // integration_not_deployed | integration_dirty | integration_ff_failed |
	                                   // fetch_failed | fetch_timeout
	                                   // (sin `origin` no hay aviso: `fetch: "no_origin"`, como en `ensure`)
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
| Claude Code (no Windows) | `axiom autofetch run --agent claude-code \|\| true` | `startup\|resume` | 10 |
| Claude Code (Windows) | `powershell -NoProfile -Command 'axiom autofetch run --agent claude-code; exit 0'` | `startup\|resume` | 10 |
| Codex | `axiom autofetch run --agent codex \|\| true` | `startup\|resume` | 10 |

- La forma de cada orden sigue la del hook del registro de skills de cada runtime: PowerShell en Claude Code para Windows y `|| true` en Codex.
- **Cotas coherentes:** fetch 8 s < presupuesto de `run` 10 s = `timeout` del hook 10 s. El `timeout` del runtime es solo el respaldo exterior: un hook que lo agota es, en Claude Code y en Codex, un error no bloqueante, así que la sesión nunca espera más de 10 s. El margen de arranque del proceso (decenas de milisegundos) cabe en los ~2 s que el presupuesto deja tras el fetch.
- `ensureAutofetchHookEntries(path, label, matcher, command)` generaliza `ensureBranchGuardHookEntries` a un solo evento (`SessionStart`): primero poda las entradas obsoletas (punto siguiente), después añade la canónica solo si `claudeHookListContains` no la encuentra, rechaza las formas inesperadas sin tocar el fichero y escribe con `filemerge.WriteFileAtomic` solo si cambió algo.
- **Retirada por `axiom sync` (hueco 1).** Hoy RF-BRA-23 no tiene un mecanismo propio de `sync` para retirar el hook de `axiom branch guard`: `installSkillRegistryAutomation` solo lo asegura, `axiom sync` nunca escribe fuera de los runtimes seleccionados (`TestSyncNeverWritesOutsideSelectedAgents`) y la única retirada es la de `axiom uninstall` (`rewriteSkillRegistryHook` → `removeSkillRegistryHook`, por orden exacta). El precedente más cercano dentro de `sync` es la poda de literales anteriores del hook del registro de skills (`pruneLegacyClaudeHook` en `ensureClaudeSkillRegistryHookWithLegacies`). Se adopta ese mismo patrón como mínimo viable: en cada `sync` de Claude Code y de Codex, `pruneStaleAutofetchHooks(root, canonical)` retira, en **cualquier** evento, toda orden que contenga `axiom autofetch run` y no sea exactamente la canónica de ese runtime y sistema bajo `SessionStart` con el matcher canónico (forma anterior, evento antiguo, la forma de otro sistema operativo o un duplicado), y borra las entradas y eventos que queden vacíos. Así, cuando una versión deja de instalar el hook en una forma o un evento, el siguiente `sync` lo retira; si algún día un runtime deja de recibirlo, su rama de `installSkillRegistryAutomation` llama solo a la poda con `canonical = ""`. Para un runtime que se deja de seleccionar, la retirada es la de `axiom uninstall`, el mismo límite que tiene hoy el hook de `axiom branch guard`. Prueba declarada: un `settings.json` y un `hooks.json` con `axiom autofetch run --agent claude-code || true` bajo `UserPromptSubmit`, otra forma bajo `SessionStart` y un hook ajeno: tras `sync` solo queda la canónica bajo `SessionStart`, el ajeno y `branch guard` quedan intactos, y un segundo `sync` deja el fichero byte a byte igual.
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

«Before presenting each stop, run `axiom autofetch run --cwd <root> --json` once. With autofetch off it does nothing. With it on, enabling it was the user's persistent authorization of that fetch, and of no other: relay its `notices` beside the stop as information, run no other fetch, and never let them change the stop, its exits or the gate. Every other rule that reserves `fetch` to the person stays as it is.»

`AGENTS.md:17,20`, `GEMINI.md:48` y `docs/manual-de-inicio.md:227,238,247` reciben el equivalente en castellano. El manual gana además una entrada `axiom autofetch` (subcomandos, hook, intervalo y prohibiciones).

**Las demás frases sobre `fetch` (hueco 6).** Se revisan y **no** necesitan una cláusula «salvo autofetch»:

- `sdd-orchestrator-sections.md:108` (verificación conjunta: «show the `fetch` the person must authorize and never run `fetch`»). Se refiere al `fetch` de los repositorios con `branch_missing` o `base_missing`, que suelen ser de código. El autofetch no cubre ese caso: `run` solo trae el repositorio de specs y `ensure` solo trae al crear o adoptar ramas. Añadir la excepción autorizaría al agente a ejecutar un `git fetch` directo, que la spec no concede.
- `sdd-orchestrator-sections.md:288` («never fetch to decide» si un commit está publicado). Prohíbe un fetch hecho para decidir; el de `axiom autofetch run` antes de una parada no es una decisión del agente, sino un disparo previo autorizado por la persona, y la regla sigue siendo cierta tal cual.
- `internal/assets/generic/stop-exits.md:7,9` («`fetch` … belong to the human»). Sigue siendo cierto: el encendido del interruptor es la decisión persistente de la persona, y la frase «and of no other … Every other rule that reserves `fetch` to the person stays as it is» del párrafo nuevo concilia ambas sin tocar el fichero. Por eso no hace falta una delta de `sdd-block-review-gates`.
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
                                                                                 ├─ cambio actual (ReadHead + registry.List; cesión: RemoteBranches + FileAt)
                                                                                 ├─ relevancia (CommitFiles, TreeDirs, AheadBehind, RevList)  ← antes del avance
                                                                                 ├─ avance (Worktrees, GitDirFromFiles/ReadHead, DirtyPaths, IsAncestor, writer.FastForward)
                                                                                 └─ AutofetchResult ─→ JSON | texto | hookSpecificOutput
                                                     (todo bajo el presupuesto de 10 s)

axiom branch ensure ──→ Engine.Ensure ──→ prefs.Autofetch ──→ checkPlan(fetchImplicit), por cada repositorio del plan (también specs in situ)
                                                         ├─ RemoteConfigured(origin) ── false ─→ se omite sin aviso, `fetched:false`
                                                         └─ writer.FetchWithin (8 s); un fallo es un aviso y `fetched:false`
```

## Cambios por fichero

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/hub/types.go` | Modificar | Campo `Autofetch bool \`json:"autofetch,omitempty"\`` |
| `internal/hub/prefs.go` | Modificar | `LookupAutofetch`, `SetAutofetch` (no escribe sin cambio) y `ErrProjectNotRegistered` |
| `internal/hub/prefs_autofetch_test.go` | Crear | Persistencia, clave ausente, idempotencia, re-registro, desregistro, hub ilegible y binario anterior (unmarshal en un struct sin el campo) |
| `internal/branch/ports.go` | Modificar | `Git`: `FetchWithin`, `FastForward`, `RemoteBranches`; `Prefs`: `Autofetch`; comentario de operaciones remotas |
| `internal/branch/autofetch.go` | Crear | `AutofetchResult`, avisos, preferencia, `WriteText` y constantes de esquema |
| `internal/branch/result.go` | Modificar | `RepositoryResult.Fetched *bool` (`omitempty`) para el repositorio de specs in situ con fetch implícito |
| `internal/branch/archive_refusal.go` | Modificar | `ChangeNotIntegrated(..., fetched, autofetch bool)` sin `--fetch` cuando el interruptor está encendido |
| `internal/branch/gitcli/ops.go` | Modificar | `FetchWithin`, `FastForward` y `RemoteBranches` |
| `internal/branch/gitcli/ops_write_test.go` | Modificar | Pruebas propias de las primitivas (argumentos exactos, sin `--prune`, `ErrTimeout`, rechazo de la fusión no ff, `for-each-ref` sin red) |
| `internal/branch/engine/engine.go` | Modificar | `gitReader` y `gitWriter` ampliados, campo `now` y comentario del paquete |
| `internal/branch/engine/prefs.go` | Modificar | `hubPrefs{path}` con `Autofetch`; `DefaultWithHubPath` |
| `internal/branch/engine/ensure.go` | Modificar | `fetchMode`, fetch implícito de 8 s con `RemoteConfigured`, también en el repositorio de specs in situ, aviso de fallo y lectura de la preferencia |
| `internal/branch/engine/archive.go` | Modificar | Propaga `autofetch` a `ChangeNotIntegrated` |
| `internal/branch/engine/autofetch.go` | Crear | `Autofetch`, `AutofetchProject`, presupuesto y el ciclo |
| `internal/branch/engine/autofetch_stamp.go` | Crear | Lectura y escritura atómica del sello |
| `internal/branch/engine/autofetch_relevance.go` | Crear | Cambio actual (cuatro reglas, con la cesión sin registro), prefijos y avisos (a) y (b) |
| `internal/branch/engine/autofetch_advance.go` | Crear | Tabla de avance de D-08, con la detección por ficheros de operación en curso y de `HEAD` desacoplado |
| `internal/branch/engine/ensure_mode_test.go` | Modificar | El doble de `Prefs` implementa `Autofetch` |
| `internal/branch/engine/ensure_autofetch_test.go` | Crear | Escenarios de RF-BRA-10 y RF-BRA-13 con el interruptor |
| `internal/branch/engine/autofetch_test.go` | Crear | Escenarios de RF-BRA-39 con repositorios git reales y un remoto local desnudo |
| `internal/cli/autofetch.go` | Crear | `RunAutofetch(args, stdin, stdout, stderr)`, banderas, ayuda, modo hook y presupuesto |
| `internal/cli/autofetch_test.go` | Crear | Subcomandos, salidas y códigos (hub temporal con `DefaultWithHubPath`) |
| `cmd/axiom/main.go` | Modificar | `case "autofetch"` y la línea en `printHelp` |
| `internal/components/sdd/inject.go` | Modificar | `AutofetchHookCommand(agent, goos)`, `ensureAutofetchHookEntries`, `pruneStaleAutofetchHooks` y las llamadas para Claude y Codex |
| `internal/components/sdd/*_test.go` | Modificar/Crear | Instalación, idempotencia byte a byte, conservación de lo ajeno, poda de entradas obsoletas por `sync` y ausencia en otros runtimes |
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
| Unidad (`hub`) | RF-HUB-07 y RF-HUB-13 en el hub; identidad por `sameProject` (mayúsculas en Windows, nombre corto) | `hub.NewManager(<tmp>/workspaces.json)`; comparación de bytes antes y después; `json.Unmarshal` en un struct sin `Autofetch` para el binario anterior |
| Unidad (`gitcli`) | `FetchWithin`, `FastForward`, `RemoteBranches` | `gitcommon.Observe` registra los argumentos exactos; un remoto inexistente (`file://`); una divergencia rechazada sin fusión |
| Motor (`engine`, `ensure`) | Fetch implícito (también en el repositorio de specs in situ, con `fetched:true` en su JSON), repositorio sin `origin` omitido sin aviso, `--base` con fetch, corte a 8 s por repositorio, fallo con aviso, `--fetch` explícito con `fetch_failed`, `decision_required` sin fetch, `done` sin fetch, `--archive` sin sugerir `--fetch`, intervalo ignorado, código sin avance local, proyecto no registrado y hub ilegible sin aviso nuevo, y salida idéntica con el interruptor apagado | Repositorios reales con un remoto desnudo local; doble de `Prefs` con `Autofetch=true` o con cada `state`; `Observe`; el corte se simula con `autofetchFetchTimeout` reducido y un doble que devuelve `ErrTimeout` |
| Motor (`engine`, `run`) | Los 20 escenarios de RF-BRA-39, más las filas 1, 4, 5 y 8 de D-08 (rama local inexistente, `merge`, `rebase` y `cherry-pick` en curso, `HEAD` desacoplado, sin presupuesto) y las reglas 3 y 4 del cambio actual (una cesión candidata con aviso `change_branch_not_adopted`; dos candidatas sin (a) ni (b)) | Igual que la fila anterior. `autofetchInterval`, `autofetchRunBudget` y el sello se fijan en la prueba. El tiempo agotado se simula con un doble de `gitWriter` que devuelve `ErrTimeout`. Las operaciones en curso se provocan con git real (un `rebase` o un `merge` con conflicto) y el `HEAD` desacoplado con `switch --detach`. Multirepo con repositorios de código cuyo remoto tiene commits nuevos: se comprueba que no hay fetch ni movimiento |
| CLI (`cli`) | Subcomandos, `--json`, `--agent` sin salida o con `hookSpecificOutput`, códigos 0/1/2, la tabla de D-03 completa (no registrado y hub ilegible en `status`, `run`, `run --agent`, `enable` y `disable`), identidad desde un worktree de feature, un subdirectorio y un puntero `.axiom-workspace`, `branch --help` sin `autofetch` | Hub temporal mediante `DefaultWithHubPath` y la dependencia del hub de `runAutofetch`. Sin `t.Parallel`, porque las pruebas de `internal/cli` comparten el HOME de `TestMain`: nunca se escribe en `GIT_CONFIG_GLOBAL` ni se supone un hub vacío |
| Componentes (`sdd`, `uninstall`) | RF-BRA-40: instalación en Claude y Codex con `timeout` 10, segunda sincronización byte a byte igual, entradas ajenas y `branch guard` intactos, poda por `sync` de las entradas obsoletas, otros runtimes sin hook, retirada | Ficheros temporales de configuración |
| Doctrina (`assets`, `agentguidance`, goldens) | Frase canónica y párrafo de paradas | Fijación de texto y `go test ./internal/components -run TestGolden -update` |
| Arquitectura | Sin regresión | Se ejecutan con `-run` `TestArchitectureOnlyExecGoLaunchesProcesses` y `TestIsolationBranchPackageNeverPassesPushOrCommitToGit`, sin cambiarlas |

### Cobertura de la spec

| Requisito o escenario | Diseño | Prueba |
|---|---|---|
| RF-BRA-39, apagado sin red ni escritura | D-03 (sin git), D-09 | `engine` + `Observe` |
| RF-BRA-39, ff con el árbol limpio; ya al día | D-08 (filas 1 y 9) | `engine` |
| RF-BRA-39, árbol sucio, divergencia, rama no desplegada | D-08 (filas 3, 6 y 7, avisos con la orden) | `engine` |
| RF-BRA-39, integración por detrás con operación git en curso o `HEAD` desacoplado | D-08 (filas 4 y 5) | `engine` (git real: `rebase` en curso y `switch --detach`) |
| RF-BRA-39, integración al día con el árbol sucio sin aviso de no-avance | D-08 (regla previa) | `engine` |
| RF-BRA-39, la rama del cambio no se mueve y avisa | D-07 (b), sin escrituras | `engine` |
| RF-BRA-39, `main_branch` distinta de la integración y repositorios de código; multirepo | D-05 | `engine` multirepo + `Observe` + SHA de `main_branch` |
| RF-BRA-39, `main_branch` igual a la integración avanza por ff | D-05, D-08 | `engine` |
| RF-BRA-39, sin push ni prune | D-02 | `Observe` |
| RF-BRA-39, remoto inalcanzable dentro de unos 10 s | D-06, D-08 (fila 8) y D-09 (presupuesto de 10 s, exit 0, aviso, sin avance) | `engine` + `cli` |
| RF-BRA-39, intervalo (omite y repite) | D-06 | `engine` |
| RF-BRA-39, relevancia (cambio, spec, ODD, nada relevante) | D-07 | `engine` |
| RF-BRA-39, proyecto no registrado o hub ilegible en `run` (manual con aviso; hook en silencio) | D-03 (tabla), D-09 | `cli` |
| Propuesta, cesión sin registro ni rama local (hueco 5) | D-07 (regla 3 y `change_branch_not_adopted`) | `engine` |
| RF-BRA-40, instalación, idempotencia y otros runtimes | D-10 | `sdd` |
| RF-BRA-40, retirada por `sync` que ya no debe instalarlo y por desinstalación | D-10 (`pruneStaleAutofetchHooks`, `removeSkillRegistryHook`) | `sdd`, `uninstall` |
| RF-BRA-40, exit 0 y cota de unos 10 s | D-09 (presupuesto), D-10 (`timeout` 10 y envoltorio) | `cli` + composición de la orden |
| RF-BRA-40, inerte fuera de proyecto y con el interruptor apagado | D-03, D-09, D-10 | `cli` + `Observe` |
| RF-BRA-40, contexto adicional solo con avisos | D-09 | `cli` |
| RF-BRA-40, doctrina | D-11 | `assets`, `agentguidance`, goldens |
| RF-BRA-01, los cinco verbos y la ayuda sin `autofetch` | D-01 | `cli` (`branch --help`) |
| RF-BRA-01, con el interruptor encendido no se publica; `scope` sin red | D-04 (solo `ensure` lee el interruptor) | `engine` + `Observe` |
| RF-BRA-10, fetch implícito, código sin mover ramas, intervalo ignorado, fallo con aviso, `--fetch` explícito, `done` | D-04 | `engine` (`ensure_autofetch_test.go`) |
| RF-BRA-10, repositorio de specs in situ, sin `origin`, `--base`, corte a 8 s | D-04 (`RemoteConfigured`, `FetchWithin`, `RepositoryResult.Fetched`) | `engine` |
| RF-BRA-13, fetch implícito y rechazo sin `--fetch` | D-04 | `engine` |
| RF-HUB-07, apagado por defecto, persistencia, re-registro, desregistro, binario anterior, `axiom.yaml` ignorado | D-03 | `hub` |
| RF-HUB-07, no registrado o hub ilegible: avisan `status` y `run` manual; hook y `ensure` sin aviso | D-03 (tabla), D-04, D-09 | `cli`, `engine` |
| RF-HUB-13, `enable`, `disable` (también sin registrar), `status`, idempotencia, hub ilegible, registro sin preguntar | D-03 (tabla), D-09 | `hub`, `cli` |
| Identidad desde worktree de feature, subdirectorio o puntero (hueco 8) | D-03 (normalización) | `hub`, `cli` |

## Matriz de amenazas

El cambio lanza subprocesos (git), automatiza control de versiones (fetch y fast-forward) e integra procesos (hook de sesión), así que la matriz aplica.

| Frontera | Aplicabilidad | Respuesta de diseño | Pruebas RED previstas |
|---|---|---|---|
| Rutas con aspecto de documentación | N/A: no se clasifican ni se ejecutan ficheros; los prefijos de relevancia solo filtran rutas que devuelve `git log` | — | — |
| Selección del repositorio git | **Aplicable** | Todo git va con `cmd.Dir` absoluto (`RunWithTimeout` lo exige) sobre el checkout principal del repositorio de specs (fetch) o el worktree que tiene desplegada la integración (avance), nunca el directorio del proceso. `gitEnvironment` retira `GIT_DIR` y `GIT_WORK_TREE`. El `cwd` del hook solo selecciona el proyecto. | (1) `run` desde un worktree de feature avanza la integración en el checkout principal y no toca el worktree; (2) `GIT_DIR` definido en el entorno de la prueba no cambia el repositorio; (3) un `--cwd` relativo se rechaza como error de uso; (4) en multirepo, `cwd` en un repositorio de código: el fetch va solo al de specs |
| Estado de commits | **Aplicable** (índice y árbol en el avance) | Con `DirtyPaths` no vacío (incluye preparados y sin seguimiento), con una operación en curso (`MERGE_HEAD`, `CHERRY_PICK_HEAD`, `rebase-merge/`, `rebase-apply/`) o con `HEAD` desacoplado no se ejecuta `merge`. `merge --ff-only` nunca crea un commit. | (1) un fichero preparado bloquea el avance; (2) un fichero sin seguimiento lo bloquea; (3) con el árbol limpio, el avance no crea commits (`RevList` igual al remoto y `HEAD` sin un padre nuevo); (4) un `rebase` y un `merge` con conflicto en curso bloquean el avance y dejan intactos el índice y los ficheros de la operación |
| Estado de push | **Aplicable** como prohibición | El puerto no tiene `push`. El fetch no lleva `--prune` ni refspec. Ninguna ruta de `run` o `ensure` escribe en el remoto. | (1) `Observe` no ve `push`, `pull`, `ls-remote` ni `--prune` en `run` encendido; (2) `isolation_test` sigue verde |
| Órdenes de PR | N/A: no se crean PR ni se componen órdenes `gh`; las órdenes de los avisos son texto para la persona y nunca se ejecutan | — | — |
| Integración de procesos (hook) | **Aplicable** (fila añadida por el diseño) | Orden constante con `\|\| true` o `; exit 0`. `run` en modo hook siempre devuelve 0, con `recover` y un presupuesto de 10 s, igual al `timeout` del hook en el runtime. Es inerte sin proyecto Axiom o con el interruptor apagado (sin git). La entrada estándar se lee con un límite de 256 KiB. | (1) la orden compuesta termina en `\|\| true` o `; exit 0` para cada runtime y sistema; (2) `runAutofetch --agent` fuera de un proyecto Axiom: salida vacía, 0 y `Observe` vacío; (3) interruptor apagado: lo mismo; (4) una carga de entrada ilegible o mayor de 256 KiB no falla; (5) un pánico simulado devuelve 0 |

## Migración y despliegue

No hace falta migrar nada. El campo `autofetch` es `omitempty`, el esquema del hub sigue en `"1.0"` y un binario anterior lo ignora. El hook se instala en el siguiente `axiom sync` y es inerte mientras nadie active el interruptor. Si hay que revertir, `axiom autofetch disable` desactiva el comportamiento al momento, y la reversión de la PR más `axiom sync` o la desinstalación retiran el hook. Con un binario que ya no conoce el hook, la entrada se borra a mano (proposal, «Plan de reversión»). Codex pedirá confiar una vez en la nueva entrada `SessionStart`.

## Plan de unidades de trabajo (un commit por unidad, para `sdd-tasks`)

| # | Unidad | Ficheros principales | Líneas estimadas |
|---|---|---|---|
| WU-1 | Preferencia `autofetch` en el hub, normalización de la identidad y puerto `Prefs.Autofetch` (RF-HUB-07, parte de RF-HUB-13) | `hub/types.go`, `hub/prefs.go`, `branch/ports.go` (Prefs), `engine/prefs.go`, `engine/ensure_mode_test.go`, `hub/prefs_autofetch_test.go` | ~260 |
| WU-2 | Primitivas `FetchWithin`, `FastForward` y `RemoteBranches` | `gitcli/ops.go`, `branch/ports.go` (Git), `engine/engine.go` (gitReader, gitWriter), `gitcli/ops_write_test.go` | ~220 |
| WU-3 | Fetch implícito en `ensure` y `ensure --archive`: 8 s, sin `origin`, `--base`, repositorio de specs in situ (RF-BRA-10, RF-BRA-13, RF-BRA-01) | `engine/ensure.go`, `engine/archive.go`, `branch/archive_refusal.go`, `branch/result.go`, `engine/ensure_autofetch_test.go` | ~450 |
| WU-4 | Ciclo `Engine.Autofetch` y resultado tipado: presupuesto, tabla de D-08 con operación en curso y `HEAD` desacoplado, cesión sin registro (RF-BRA-39) | `branch/autofetch.go`, `engine/autofetch*.go`, `engine/autofetch_test.go` | ~1.150 |
| WU-5 | Verbo `axiom autofetch` con la tabla de estados de D-03 (RF-HUB-13, RF-BRA-39, modo hook de RF-BRA-40) | `cli/autofetch.go`, `cmd/axiom/main.go`, `cli/autofetch_test.go` | ~540 |
| WU-6 | Hook `SessionStart` en Claude Code y Codex, poda por `sync` y retirada (RF-BRA-40) | `components/sdd/inject.go`, `components/uninstall/service.go` y sus pruebas | ~380 |
| WU-7 | Doctrina, documentación y goldens (RF-BRA-40) | `routing.go`, `sdd-orchestrator-sections.md`, `sdd-apply/SKILL.md`, pruebas de texto, 10 goldens, `AGENTS.md`, `GEMINI.md`, `docs/manual-de-inicio.md` | ~170 escritas + goldens regenerados |

Total estimado: unas 3.170 líneas escritas (sin los goldens), en una sola PR (`single-pr`). Dependencias: WU-1 y WU-2 son independientes; WU-3 depende de WU-1 y WU-2; WU-4 depende de WU-1 y WU-2; WU-5 depende de WU-4; WU-6 y WU-7 son independientes del motor, pero WU-7 cita el verbo, así que conviene después de WU-5.

## Riesgos y supuestos

- **R-1 (cerrado).** La spec corregida (RF-HUB-07) fija quién avisa con el proyecto no registrado o el hub ilegible: `autofetch status` y `run` sin `--agent`; `enable` rechaza; `disable` sin registrar informa sin error; el hook y `ensure` se comportan como apagado y sin aviso. Lo recoge la tabla de D-03 (cierra también el hueco 12).
- **R-2 (doctrina y `stop-exits.md`).** Se resuelve sin tocar `stop-exits.md` ni las líneas 108 y 288 de `sdd-orchestrator-sections.md`: el párrafo nuevo acota la autorización del encendido a `axiom autofetch run` («and of no other») y declara que las demás reglas que reservan `fetch` a la persona siguen igual (D-11). Si la revisión aun así ve contradicción, haría falta una delta de `sdd-block-review-gates`, que este diseño evita.
- **R-3 (cerrado, con un residuo).** El fetch implícito de `ensure` trae todos los repositorios del plan, también el de specs in situ, con 8 s por repositorio y sin aviso para los que no tienen `origin` (D-04). Residuo: `--fetch` explícito sigue sin traer el repositorio de specs in situ, para no cambiar su contrato de `fetch_failed`; la spec no lo pide.
- **R-4 (cambio actual).** Se resuelve con las cuatro reglas de D-07. Con varios cambios activos y `cwd` en la integración (habitual en el repositorio de specs de un multirepo), o con varias cesiones candidatas, no hay avisos de relevancia. La spec lo admite («sin cambio actual resoluble»). La regla 3 depende de que el `kickoff.yaml` esté publicado en la rama del cambio, que es lo que hace `axiom sdd publish`; un cambio ODD cedido no se detecta por esta vía.
- **R-5 (forma de salida de `SessionStart` en Codex).** La salida `hookSpecificOutput.additionalContext` sigue la documentación de Claude Code. En Codex se supone el mismo contrato, igual que el hook de guarda se instaló sin verificarlo en ejecución (A-2). Hay que validarlo a mano en Codex, junto con el `timeout` de 10 s.
- **R-6 (retirada por `sync`).** Se cierra con la poda de D-10 (`pruneStaleAutofetchHooks`), que sigue el precedente de `pruneLegacyClaudeHook`. Límite heredado de RF-BRA-23: un runtime que se deja de seleccionar no se toca en `sync` y su hook se retira con `axiom uninstall`.
- **R-7 («el hook no lanza procesos» fuera de proyectos Axiom).** El runtime siempre lanza el proceso `axiom`. Se interpreta, como dice el cuerpo de RF-BRA-40, como «no lanza procesos de git».
- **R-8 (órdenes de los avisos).** Para la divergencia se propone `rebase origin/<int>`; para la rama no desplegada, `fetch . refs/remotes/origin/<int>:refs/heads/<int>` (solo local, ff garantizado por git, justificado en D-08); para `HEAD` desacoplado, `switch` y `merge --ff-only`. La redacción es revisable en `sdd-tasks` sin cambiar la arquitectura.
- **R-9 (concurrencia).** Dos sesiones que arrancan a la vez pueden traer el remoto dos veces. Es inocuo: un choque de bloqueo de git se degrada a aviso y no se reintenta.
- **R-10 (cotas en el límite).** El presupuesto de `run` (10 s) coincide con el `timeout` del hook (10 s). Si el arranque del proceso en una máquina muy cargada consumiera más del margen, el runtime cortaría el hook: es un error no bloqueante, la sesión arranca y solo se pierden los avisos de esa ejecución. En `ensure` con la red caída, un plan de N repositorios puede esperar hasta unos 8·N s, como permite la cota por repositorio de RF-BRA-10.

## Preguntas abiertas

- Ninguna que bloquee. R-2 y la orden de la fila 6 de D-08 quedan para la revisión de la puerta de diseño.

## Registro de remediación

- 2026-11-26: el gatekeeper del modo automático relanzó la fase por los huecos 3, 10, 11 y 12 de la revisión de cobertura y por la spec corregida; se alinea D-05 con `main_branch` intocable solo si difiere de la integración, se fijan las cotas en 8 s de fetch, 10 s de `run` y 10 s de `timeout` del hook, se añade la tabla de avisos por orden para el proyecto no registrado y el hub ilegible (cierra R-1 y el hueco 12), los avisos de no-avance pasan a exigir la integración por detrás y se añaden la operación git en curso y `HEAD` desacoplado (con la orden justificada para la rama no desplegada), el fetch implícito de `ensure` usa 8 s, omite sin aviso los repositorios sin `origin`, se hace con `--base` e incluye el repositorio de specs in situ (cierra R-3), se define la poda por `sync` del hook (hueco 1), la normalización de la identidad (hueco 8), la cesión sin registro (hueco 5) y la decisión sobre las demás frases de `fetch` de la doctrina (hueco 6); se actualizan la cobertura, las pruebas y el plan de unidades de trabajo.
