# Diseño: Ramas y worktrees por topología (inc-25-ramas-y-worktrees-por-topologia)

> Fase `design` · rama `inc/inc-25-ramas-y-worktrees-por-topologia` · 2026-10-17 · **revisión 1**.
> Entradas: `proposal.md`, `exploration.md` y las specs reabiertas y reaprobadas. La capacidad nueva `change-branch-lifecycle` (REQ-25.1 a REQ-25.24 y REQ-25.45) se reescribió entera, y hay nueve deltas (REQ-25.25 a REQ-25.44). Decisiones de producto: D1–D14 (Engram #619, #637, #663), correcciones de la propuesta (#665) y D16–D19 (#681; D15 queda retirada). Lecturas del design aceptadas: #712.
> Revisión 1: incorpora D16 a D19 y las lecturas aceptadas, y corrige los defectos internos que encontró la validación independiente (§0.3).
> Idioma: castellano. Los identificadores de código, paquetes, tipos, funciones, rutas, flags y claves YAML y JSON se conservan en inglés.
> Las decisiones de este documento se numeran **D-01…D-25** para no confundirlas con las decisiones de producto **D1…D19**.
> Este documento fija el CÓMO a nivel de arquitectura. No contiene tareas.

## 0. Resumen

inc-25 se construye sobre seis piezas:

1. **Un dominio puro, `internal/branch`.** Resuelve la política (excepción por cambio > `repositories:` > `git:` > valores por defecto), planifica ramas y worktrees por topología y por número de roles por repositorio (incluida la rama de archivado), decide el veredicto de rama en sus dos contextos (ordinario y de archivado) y la confirmación de fusión por capas, y define el contrato JSON con tokens en inglés y etiquetas en castellano. No ejecuta procesos ni toca el disco.
2. **Dos hojas sin dependencias internas.**
   - `internal/gitref` valida nombres de referencia y es pura.
   - `internal/gitcommon` contiene el **único punto que ejecuta git** (`gitcommon.Run`, con observador) y las lecturas de ficheros que no lanzan procesos (raíz git, common dir, `HEAD`, worktree enlazado y checkout principal).
3. **Dos adaptadores.** `internal/branch/gitcli` expone las operaciones git tipadas sobre `gitcommon.Run`; la única operación remota es `Fetch`. `internal/branch/registry` es el registro local en `<git common dir>/axiom/branch/v1/`. Las preferencias personales se leen del hub a través de un puerto.
4. **Un servicio de aplicación, `internal/branch/engine`.**
   - Orquesta `ensure` (también con `--archive`), `status`, `done` y `guard`.
   - Calcula los cambios pendientes de archivar.
   - Expone el **veredicto único** (`VerdictFor`) y la **guarda A** (`RequireChangeBranch`). El veredicto se calcula solo con lecturas de ficheros, sin lanzar procesos.
5. **Guardas.**
   - A: en cada punto de entrada de los escritores Go de la carpeta del cambio.
   - B: en `axiom sdd status`.
   - D: `axiom branch guard`, que instala `axiom sync` en Claude Code, Codex y OpenCode. Bloquea las escrituras del agente en documentos de Axiom en la rama equivocada y avisa sin bloquear en el resto de ficheros de repositorios con un cambio activo registrado.
   - La capa C desaparece (D17).
6. **Doctrina en un punto.** `routing.go` genera el paso ODD con `axiom branch ensure --lane odd` y la única línea que prohíbe `EnterWorktree`/`ExitWorktree`. Las skills invocan el verbo y usan las rutas absolutas de su JSON. `sdd-archive` sigue el flujo D16: rama `-archive` tras fusionar el PR del cambio y PR documental aparte.

`monorepo-decoupled` se retira con un mensaje que nombra `multirepo`.

### 0.1 Decisiones clave

| ID | Decisión | Sección |
|---|---|---|
| D-01 | Arquitectura hexagonal: dominio puro `internal/branch`; hojas `internal/gitref` (pura) e `internal/gitcommon` (único `exec` de git más lecturas de ficheros); adaptadores `gitcli` y `registry`; servicio `engine`; CLI en ficheros nuevos de `internal/cli`. Grafo de importaciones verificado y sin ciclos | 2.1 |
| D-02 | Cuatro verbos de primer nivel (`ensure`, `status`, `done`, `guard`); contrato `axiom.branch-result/v1` con rutas absolutas, tokens en inglés y su etiqueta, códigos tipados de la spec y salida 0, 1 y 2 | 2.2 |
| D-03 | Retirada de `monorepo-decoupled` con el inventario real de apariciones; `git:` y `repositories:` con decodificación estricta, `config_invalid` para la configuración e `invalid_name` para la invocación | 2.3 |
| D-04 | Planificador puro: papel del repositorio, roles por raíz git, modelo de D12, rol `archive` prohibido (S-15), `<repo>` por último segmento (S-17), límite fijo de 220 caracteres en Windows (A-1) | 2.4 |
| D-05 | Dos fases (comprobación sin efectos y aplicación); el JSON describe el estado final, lo que da la idempotencia byte a byte; `--archive` es un plan aparte | 2.5 |
| D-06 | Base sin red: `origin/<integración>` si existe en local y, si no, la local; `--fetch` explícito antes de crear nada | 2.6 |
| D-07 | Q2: registro en `<common dir>/axiom/branch/v1/`, conviviendo con `gentle-ai/` sin tocarlo. `ensure` lo localiza con `git rev-parse --git-common-dir` y los lectores con ficheros (prueba de igualdad); se reconstruye desde git | 2.7 |
| D-08 | Modo y raíz en el hub: campos `omitempty`, métodos aparte y `Register` sin cambios; la identidad del proyecto es la maestra del checkout principal | 2.8 |
| D-09 | Q1: `done` por capas (`ancestry`, `equivalent_tree` si git ≥ 2.38, `pr_merged`), con detección de la capacidad en ejecución; la rama `-archive` solo admite las dos primeras capas | 2.9 |
| D-10 | Veredicto puro `branch.Decide` en dos contextos (`ordinary`, `archive`) y cuatro consultantes; hechos solo de ficheros (`HEAD` leído sin procesos) | 2.10 |
| D-11 | Guarda A: una función, `engine.RequireChangeBranch`, que falla en cerrado, con contexto por escritor; escritor de incrementos nuevo `writeIncrement`, extraído de `CreateIncrement`; prueba de inventario | 2.11 |
| D-12 | Instantánea `git:` aditiva en `kickoff.yaml`, con la rama de archivado; `seal` se niega fuera de rama; `--infer` exento | 2.12 |
| D-13 | Guarda B: contexto de archivado en la fase archive; `wrong_branch` antes de la autoridad de edición; costura única antes de `applyEditAuthorityBlock`; sin tocar la compuerta `integration` | 2.13 |
| D-14 | R5: raíces registradas en `allowedEditRoots`; tareas por rol contra su worktree; reubicación en los dos detectores de autoridad | 2.14 |
| D-15 | R4 y S-11: ascensión acotada al worktree enlazado y, sin marcador, resolución desde el checkout principal, solo con lecturas de ficheros | 2.15 |
| D-16 | R6: listado único de cambios registrados; prevalece el worktree de archivado, después el del cambio y por último el checkout principal | 2.16 |
| D-17 | D17: cambios pendientes de archivar, informativos, en `status` y en el dashboard; sin código de fallo, CI ni invariante | 2.17 |
| D-18 | D19: `guard` bloquea documentos de Axiom (unión de contextos) y avisa en el resto; sin procesos; fallo en abierto; canal de aviso decidido por runtime | 2.18 |
| D-19 | Doctrina: `routing.go`, sección de kickoff del orquestador y ediciones de las skills (incluido el flujo de archivado D16) | 2.19 |
| D-20 | Dashboard: `ensure` y después el escritor guardado; `decision_required` como 409; preferencias editables; política en solo lectura; pendientes en `X-Axiom-Notes` | 2.20 |
| D-21 | Q6 resuelta: `axiom-worktree.*` pasa a ser un envoltorio fino; `cleanup-worktree.*` se retira; `guard-primary-checkout.ps1` se conserva como política local por ruta (G-12) | 2.21 |
| D-22 | Literales `main` sustituidos por la integración resuelta; inventario en las skills | 2.22 |
| D-23 | Ratchets: rebanadas verticales con primer llamador alcanzable desde `./cmd/axiom`, sin actualizar la línea base de deadcode | 2.23 |
| D-24 | Lo que no cambia | 2.24 |
| D-25 | D16 y D18: rama de archivado `<prefijo>/<cambio>-archive` y `ensure --archive` (REQ-25.45) | 2.25 |

### 0.2 Lecturas aceptadas, retiradas y abiertas

**Aceptadas (#712) e incorporadas a la spec:**
- G-1: `config_invalid` y `git_failed`.
- G-4: tokens en inglés con etiqueta en castellano, incluido `correct`.
- G-5 y S-17: `<repo>` es el último segmento del `path`; para `.`, el directorio de la raíz git; una colisión da `invalid_name`.
- G-6: `ensure` y después el escritor guardado; la TUI solo lista.
- G-7: el observador y la prueba de arquitectura cubren `internal/branch/**` e `internal/gitcommon`.
- G-9: aviso en la cabecera `X-Axiom-Notes`.
- A-1: margen de 40 caracteres, es decir, 220 en Windows.
- A-2: el formato del hook de Codex se verifica al implementar.
- A-3: `branch_prefix: inc` en este repositorio.

**Retiradas:** G-2 (D17 retira el comprobador), G-3 (lo sustituye D19) y G-8, porque con D16 la compuerta `integration` vuelve a exigirse sin excepciones.

**Abiertas para la compuerta `design`:**

| ID | Punto | Lectura que adopta el diseño |
|---|---|---|
| G-10 | REQ-25.18 asigna el contexto de archivado solo a la fase archive de `sdd status` y a `archive-compose`. Pero `axiom sdd integrate prepare\|verify` (que escribe `integration.yaml`) y `gate record --gate integration` (que escribe la evidencia `pr_merged` en `gates.yaml`) también se ejecutan en la fase archive, y solo pueden hacerse después de la fusión, en la rama `-archive`. En el contexto ordinario se negarían allí | Esas dos escrituras consultan el veredicto en el contexto de archivado |
| G-11 | La guarda D no conoce la fase. Con el contexto ordinario bloquearía las escrituras legítimas del archive en la rama `-archive` | En los documentos de Axiom, la guarda D da por correcta la rama que sea correcta en cualquiera de los dos contextos. La exactitud por fase la garantizan A y B |
| G-12 | REQ-25.24 cita «envoltorio fino o retirada» para `guard-primary-checkout.ps1`, pero ninguna de las dos opciones conserva su protección: deniega cualquier escritura en el checkout principal y avisa tras órdenes Bash que lo sacan de `main`, que es el patrón #615 y que la guarda D no cubre | Se conserva como política local con mensajes que remiten a `axiom branch` (D-21) |
| G-13 | `ensure --archive` sobre un cambio cuya base ya lo contiene archivado (sin `changes/<c>/` pero con `changes/archive/<fecha>-<c>/`) solo dispone del código `change_not_integrated` | Mismo código, con un mensaje que nombra la entrada archivada y dice que no hay nada que archivar |
| A-4 | La sonda de linajes RDD de `done` debe leer el almacén sin lanzar git para no escapar al observador | Supuesto: se verifica al implementar S6 sobre `reviewtransaction`; si su API exige git, la sonda queda fuera del motor, solo local, y su exención se documenta |
| A-5 | Canal de aviso de Codex en `PostToolUse` y `MultiEdit` en el matcher de Claude | Supuestos que se verifican al implementar S11 (A-2 cubre Codex); si el canal de Codex no existe, Codex queda solo con el bloqueo |

### 0.3 Revisión 1: cambios por hallazgo

| Origen | Cambio | Dónde |
|---|---|---|
| A1 (D16) | Archive en la rama `-archive` y PR documental; REQ-21.16 sin excepción; se elimina la exención de la compuerta `integration` | D-13, D-19, D-25 |
| A2 (D17) | Se elimina el verbo de CI y el invariante; REQ-25.20 pasa a «pendientes de archivar», informativo | D-02, D-17 |
| A3 (D18) | `ensure --archive`, `change_not_integrated` con sugerencia de `--fetch`, `archive_branch_not_applicable`, solo `--prefix` (S-16), `done` sobre `-archive` con dos capas, contexto de archivado en B | D-09, D-10, D-13, D-25 |
| A4 (D19) | `guard` bloquea los documentos de Axiom y avisa en el resto; `HEAD` leído sin procesos; canal por runtime | D-10, D-18 |
| A5 (G y A) | Lecturas aceptadas incorporadas; `status` sin `--change`; sin referencia de integración, solo aviso | §0.2, D-02, D-17 |
| B1 | La validación de nombres sale a `internal/gitref`, una hoja; grafo verificado | D-01, D-03, §4.6 |
| B2 | Rebanadas verticales con primer llamador desde `./cmd/axiom`; `VerdictFor` asignado a S8 | D-23, §11 |
| B3 | Inventario real de `monorepo-decoupled`; el marcador lo escribe solo `RunSDDContinue` (`sdd_status.go:59`); reubicación también en `detectUnauthorizedEditRootsForCurrentWorkUnit`; un único `VerdictQuery` con `Context` y `Consultant` | D-03, D-11, D-14, D-10 |
| B4 | La costura de B va después de `resolveApplyState` y antes de `applyEditAuthorityBlock`; no hay `governanceOptions` | D-13 |
| B5 | Pseudocódigo con `append` anidado válido en Go | D-13 |
| B6 | Q6 resuelta | D-21 |
| B7 | Estimación recalculada y cuadrada | §11 |

### 0.4 Unidades de entrega

Doce rebanadas verticales encadenadas (§11): 6.270 líneas de autoría estimadas (5.650–6.900 con ±10 %), sin goldens. La estrategia de la sesión es `single-pr` con un presupuesto de 400 líneas, así que `sdd-tasks` debe declarar `Decision needed before apply: Yes`.

## 1. Enfoque técnico

| Principio | Cómo se aplica |
|---|---|
| Las ramas se crean siempre; los worktrees son una preferencia | El plan produce ramas en todos los modos; el modo decide si hay worktree (`worktree`) o checkout en el propio repositorio (`in_place`) |
| Planificar puro, ejecutar fino (D2) | `branch.Plan`, `branch.Decide` y `branch.ConfirmMerge` son puros; `engine` reúne hechos y aplica con `gitcli` |
| Rutas absolutas como contrato (D10) | Todo resultado lleva rutas absolutas; cada orden posterior recibe `--cwd` desde el JSON |
| Un veredicto para tres guardas (D9, D17) | A, B y D consultan `engine.VerdictFor`, que solo lee ficheros |
| Un único punto que ejecuta git (REQ-25.1, G-7) | Solo `internal/gitcommon/exec.go` lanza procesos dentro de `internal/branch/**` e `internal/gitcommon`; el observador `gitcommon.Observe` registra cada invocación y una prueba de arquitectura lo fija |
| Fallar en cerrado donde se escribe y en abierto donde se avisa | A se niega ante un error interno; B lo proyecta; D nunca bloquea ni avisa por error. Fuera de git, el veredicto es `not_applicable` con aviso |
| Sin red por defecto (D14-c) | La única operación remota es `gitcli.Fetch`, solo con `--fetch` |
| Contratos aditivos | `axiom.yaml`, `kickoff.yaml`, `workspaces.json` y el status v2 solo ganan claves opcionales |
| Código nuevo en ficheros nuevos | `internal/cli`, `internal/sddstatus` e `internal/reviewtransaction` se comparten con upstream (R9): allí solo hay costuras de pocas líneas |
| Identidad de rutas por el sistema de ficheros | Comparaciones con `internal/pathidentity`; las rutas de git (`C:/…`) se normalizan con `filepath.FromSlash` y `Clean`; los mensajes citan rutas con `pathquote.Quote` |

## 2. Decisiones de arquitectura

### 2.1 D-01 — Paquetes, fronteras y grafo de importaciones

**Decisión.**

```
cmd/axiom/main.go ── case "branch" ──▶ internal/cli/branch.go (RunBranch)
                                            │
             ┌──────────────────────────────┼─────────────────────────────┐
             ▼                              ▼                             ▼
  internal/branch/engine  ◀── internal/sddstatus (costura)    internal/dashboard, internal/tui
  (Ensure, Status, Done, Guard, PendingArchive, VerdictFor, RequireChangeBranch, RegisteredChanges, Snapshot)
      │            │                 │                   │
      ▼            ▼                 ▼                   ▼
 internal/branch  branch/gitcli    branch/registry     puerto Prefs ◀── internal/hub
 (dominio puro)   (ops tipadas)    (JSON + bloqueo)    puerto LineageProbe ◀── internal/cli
      │                │
      ▼                ▼
 internal/gitref  internal/gitcommon  ◀── internal/workspace (solo lecturas de ficheros), internal/hub
 (hoja pura)      (hoja: Run único + lecturas de ficheros)
```

- **`internal/gitref`** (hoja pura, solo biblioteca estándar). Contiene `ValidRefName` (reglas de `git check-ref-format`), `ValidBranchComponent` (componente de cambio o rol, que debe ser válido a la vez como rama y como nombre de directorio de Windows) y `ValidPrefix`. La usan `workspace` (validación de `git:` y `repositories:`), `kickoff` (validación de la instantánea) y `branch`. Corrige el ciclo B1: `workspace` y `kickoff` ya no necesitan importar `branch`.
- **`internal/gitcommon`** (hoja, biblioteca estándar). Tiene dos partes:
  - *Ejecución:* `Run(ctx, dir, args...)` en `exec.go`, el único `exec.Command` permitido. Retira las variables que redirigen el repositorio (`GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`, `GIT_COMMON_DIR`…), fija `GIT_OPTIONAL_LOCKS=0` y `GIT_TERMINAL_PROMPT=0`, y aplica tiempo máximo y límite de salida, siguiendo `specbaseline/gitview.go`. La variable de prueba `Observe func(dir string, args []string)` es nula en producción. Sobre `Run` están `CommonDir`, que ejecuta `git rev-parse --path-format=absolute --git-common-dir`, y `TopLevel`, que ejecuta `git rev-parse --show-toplevel`.
  - *Ficheros, sin procesos:*
    - `GitRootFromFiles`: asciende hasta una entrada `.git`.
    - `GitDirFromFiles`: lee el `.git` directorio o el fichero `gitdir:`.
    - `CommonDirFromFiles`: usa `commondir` si existe y, si no, el propio git dir.
    - `LinkedWorktreeRoot`: detecta el `.git` fichero cuyo git dir contiene `commondir`, lo que excluye los submódulos.
    - `MainWorktreeFromFiles`: devuelve el padre del common dir cuando se llama `.git`; si no, no se resuelve.
    - `ReadHead`: lee `<gitdir>/HEAD`; `ref: refs/heads/<x>` es una rama, un hash es un HEAD desacoplado y cualquier otra forma (por ejemplo reftable, `refs/heads/.invalid`) es desconocida.
- **`internal/branch`** (dominio). Tipos, política, `Plan`, `Decide`, `ConfirmMerge`, `Result` y continuaciones. Declara los puertos `Git`, `Registry`, `Prefs` y `LineageProbe`.
- **`internal/branch/gitcli`**. Implementa `branch.Git` sobre `gitcommon.Run` con una lista cerrada de subcomandos.
- **`internal/branch/registry`**. Implementa `branch.Registry` con `reviewtransaction.AcquireAuthorityFileLock` y `ReplaceFileAtomic` (las primitivas de `kickoff.AppendGate`).
- **`internal/branch/engine`**. `engine.Default(hubManager *hub.Manager, lineages branch.LineageProbe)` monta los adaptadores reales. No importa `sddstatus`, `cli`, `dashboard` ni `tui`.
- **CLI.** Ficheros nuevos `internal/cli/branch.go` (`RunBranch(args []string, stdout, stderr io.Writer) int`), `branch_guard.go` (cargas de cada runtime) y `branch_lineage.go` (adaptador `LineageProbe`).

**Grafo de importaciones** (las relaciones existentes se han verificado con rg sobre los ficheros que no son de prueba):

| Paquete | Importa (internos) | Estado |
|---|---|---|
| `gitref` | — | Nuevo |
| `gitcommon` | — | Nuevo |
| `workspace` | `pathidentity`, `pathquote`, **`gitref`**, **`gitcommon`** | Existentes y nuevos |
| `handoff` | `workspace` | Existente |
| `multirole` | `workspace` | Existente |
| `kickoff` | `multirole`, `reviewtransaction`, `handoff`, `workspace`, **`gitref`** | Existentes y nuevo |
| `hub` | `workspace`, `pathquote`, **`gitcommon`**, **`pathidentity`** | Existentes y nuevos |
| `branch` | `workspace`, `kickoff`, `gitref` | Nuevo |
| `branch/gitcli` | `branch`, `gitcommon` | Nuevo |
| `branch/registry` | `branch`, `reviewtransaction` | Nuevo |
| `branch/engine` | `branch`, `gitcli`, `registry`, `gitcommon`, `workspace`, `kickoff`, `hub` | Nuevo |
| `sddstatus` | (existentes) + **`branch`**, **`engine`** | Costura |
| `cli` | (existentes) + `branch`, `engine` | Costura |
| `dashboard` | (existentes, incluido `cli`) + `engine` | Costura |
| `tui` | (existentes) + `engine` | Costura |

Ninguno de `gitref`, `gitcommon`, `workspace`, `handoff`, `multirole`, `kickoff`, `hub` ni `reviewtransaction` importa `branch`, `engine`, `sddstatus`, `cli` ni `dashboard`. La única importación de `cli` desde `reviewtransaction` está en una prueba (`review_mode_disable_guidance_cli_unix_test.go`). No hay ciclos.

**Alternativas descartadas.**
- *`ValidRefName` dentro de `branch`.* Crea el ciclo `workspace → branch → workspace` (B1).
- *Ejecutar git también en `gitcli` y en `gitcommon` por separado.* Daría dos puntos de ejecución y el observador de G-7 no vería todo.
- *Un único paquete con exec dentro.* Las tablas de D12 necesitarían repositorios reales y «sin red» dependería de la disciplina de cada función.
- *Reutilizar `reviewtransaction.runGit`.* Es código compartido con upstream (R9) y no tiene lista cerrada de subcomandos.

**Justificación.** Las reglas de política y veredicto se prueban con tablas. La ausencia de operaciones remotas es una propiedad estructural (un solo `exec`, con observador) y la ausencia de procesos en el veredicto también (solo lecturas de ficheros).

### 2.2 D-02 — Verbo `axiom branch` y contrato de salida

**Decisión.** `cmd/axiom/main.go` gana `case "branch": os.Exit(cli.RunBranch(os.Args[2:], os.Stdout, os.Stderr))` y `printHelp` lo lista.

| Subcomando | Flags |
|---|---|
| `ensure` | `--change <n>` (obligatorio), `--lane sdd\|odd` (por defecto `sdd`), `--cwd`, `--json`, `--mode worktree\|in_place`, `--prefix`, `--branch`, `--base`, `--repo <ruta>` (repetible), `--fetch`, `--archive` |
| `status` | `--change <n>` (opcional, REQ-25.15), `--lane`, `--cwd`, `--json` |
| `done` | `--change <n>`, `--lane`, `--cwd`, `--json`, `--force`, `--fetch` |
| `guard` | `--agent claude-code\|codex\|opencode`; lee la carga del runtime por la entrada estándar |

Cualquier otro subcomando (`push`, `pr` o el retirado `check`) se rechaza como no reconocido con código 2 (REQ-25.1). La ayuda lista exactamente los cuatro.

**Contrato `axiom.branch-result/v1`** (§4.3), con claves en camelCase como el status v2:
- **Tokens y etiquetas (G-4).** Cada campo enumerado lleva su token en inglés y, en un campo hermano `<campo>Label`, la etiqueta en castellano: `status`/`statusLabel`, `verdict`/`verdictLabel`, `confirmedBy`/`confirmedByLabel`.

  | Campo | Tokens y etiquetas |
  |---|---|
  | `status` | `ready` (listo), `decision_required` (decisión requerida), `refused` (rechazado), `partial` (parcial; solo `done`), `not_started` (no iniciado; solo `status`), `error` |
  | Veredicto | `correct` (correcto), `wrong_branch` (rama equivocada), `not_applicable` (no aplicable) |
  | Capa de `done` | `ancestry` (ascendencia), `equivalent_tree` (árbol equivalente), `pr_merged` (PR fusionado), `forced` (forzado) |

- **`code`.** Aparece cuando `status` es `refused`, `decision_required`, `partial` o `error`, y es uno de los dieciséis códigos de la spec: `decision_required`, `dirty_worktree`, `wrong_branch`, `specs_repo_not_on_integration`, `base_not_found`, `fetch_failed`, `path_too_long`, `invalid_name`, `config_invalid`, `git_failed`, `plan_conflict`, `branch_checked_out_elsewhere`, `not_a_git_repository`, `merge_not_confirmed`, `change_not_integrated` y `archive_branch_not_applicable`.
- **Mensajes.** `message` y `continuation` van en castellano; `continuation` es siempre una orden ejecutable o una acción concreta.
- **Rutas.** Todas son absolutas.

**Códigos de salida.** `0` cuando se alcanza el estado pedido (`ready`, o `not_started` en `status`); `1` ante cualquier rechazo tipado, `decision_required`, `partial` o `error`; `2` ante un error de uso. Sin `--json`, la salida es texto en castellano con los mismos códigos.

**Alternativas descartadas.** *`axiom sdd branch`*: acopla ODD a SDD (D13). *Valores en castellano en el JSON*: no son estables entre idiomas (G-4, aceptada).

### 2.3 D-03 — Retirada de `monorepo-decoupled` y bloques `git:` y `repositories:`

**Decisión.**

1. **Inventario real** (rg sobre el worktree, excluido `openspec/`):

   | Dónde | Apariciones | Acción |
   |---|---|---|
   | `internal/workspace/types.go:10-11` | Constante `TopologyMonorepoDecoupled` | Se elimina |
   | `internal/workspace/loader.go:126,130` | `validateBaseConfig` y su mensaje | El literal retirado da un mensaje propio: «la topología `monorepo-decoupled` fue retirada: sustitúyela por `multirepo` en `workspace.topology` y ejecuta `axiom workspace validate`»; el resto de topologías desconocidas conserva el mensaje con dos valores |
   | `internal/workspace/validator.go:78-90` (mensaje en :82) | Caso del validador | Se elimina |
   | `internal/workspace/locate.go:403` (código) y :397, :411, :450 (comentarios) | `separatesSpecs` y comentarios | Solo `multirepo` |
   | `internal/skillregistry/registry.go:121` | Comentario «Monorrepo desacoplado» | Se corrige |
   | Pruebas: `workspace/validator_test.go:102-161`, `workspace/locate_test.go:281,319,458,507`, `workspace/declarations_test.go:325`, `sddstatus/planning_home_test.go:183` | Fixtures con el valor | Pasan a `multirepo` o pasan a probar el rechazo |
   | `axiom.example.yaml:13,19` | Opción documentada | Se retira; se añaden `git:` y `repositories:` comentados |
   | `docs/manual-de-inicio.md:120,158,189,248` | Opción documentada | Se retira y se remite a `multirepo` |
   | `docs/architecture/AXIOM_ENTERPRISE_VISION.md:62,74` | Opción documentada | Se marca como retirada |
   | `docs/archive/ROADMAP.md:30,82` | Registro histórico de INC-01 | Se conserva: es un registro archivado y el pin de REQ-25.25 no lo cubre |

   `cmd/axiom/main.go`, `internal/hub/types.go` y `internal/dashboard/assets/index.html` no ofrecen el valor; solo listan `monorepo-embedded` y `multirepo` (verificado) y no cambian. Con puntero, la resolución se rechaza con `config_unloadable` y el mensaje anterior (REQ-25.25). Con un `axiom.yaml` directo se aplica el caso 3 de REQ-24.7: resolución embebida y nota, sin darle semántica separada.
2. **Bloques nuevos (REQ-25.26, REQ-25.27).** Fichero nuevo `internal/workspace/gitpolicy.go` con:
   - `GitConfig{IntegrationBranch, BranchPrefix}`;
   - `RepositoryPolicy{Path, IntegrationBranch, BranchPrefix}`.

   Se decodifican como `baseline:`: recorrido del `yaml.Node` contra listas cerradas de claves, errores acumulados y canales propios (`Declarations.Git`, `GitErr`, `Repositories`, `RepositoriesErr`). `WorkspaceConfig` gana `Git` y `Repositories` con `yaml:"-"`. `joinLoadErrors` los incluye. `Locate` nunca se rechaza por ellos.
3. **Reglas en carga.** Los valores se validan con `gitref.ValidRefName` y `gitref.ValidPrefix` (sin `/` inicial ni final, sin `..` ni espacios). `path` es relativo y contenido en la maestra; las rutas repetidas se rechazan.
4. **Reglas de sistema de ficheros** (`workspace.ValidateRepositoryPolicies(cfg, master)`, desde `runWorkspaceValidate`). Cada `path` existe como directorio; su raíz git se obtiene con `gitcommon.GitRootFromFiles`, sin procesos; sin git, advertencia y `COMPLIANT`; dos entradas con la misma raíz son un error que nombra las dos.
5. **Códigos (REQ-25.3).** Un valor inválido o una configuración ilegible en `axiom.yaml` o en el hub dan `config_invalid` en `axiom branch`. Un `--prefix`, `--branch` o `--change` inválidos dan `invalid_name`.

**Alternativas descartadas.** Las de la versión anterior (alias, `KnownFields` global, ajustes por rol) y *validar nombres con `branch`* (ciclo B1).

### 2.4 D-04 — Planificador puro

**Decisión.** `branch.Plan(in PlanInput) (Plan, *Refusal)` es determinista: repositorios en orden estable (primero el del planning home y después por nombre) y ramas en orden estable (cambio, roles alfabéticos y archivado). `engine` reúne los hechos (§3.1).

**Papel de cada repositorio (REQ-25.5).**

| Topología | Repositorio | Papel | Tratamiento |
|---|---|---|---|
| `monorepo-embedded` | Raíz git del planning home y cualquier otra raíz git tocada por un rol | `change_branch` | Modelo de REQ-25.4; rama de archivado solo con `--archive` (D-25) |
| `multirepo` | Raíz git de `workspaceRoot` (repositorio de specs) | `specs_in_place` | Nunca recibe ramas, worktrees ni checkout; fuera de su integración o en una rama de rol, `specs_repo_not_on_integration`; no tiene rama de archivado |
| `multirepo` | Raíz git de cada repositorio declarado por un rol del cambio | `code` | Modelo de REQ-25.4 desde su propia integración; sin rama de archivado |

Un rol cuyo único repositorio resuelve a la raíz del repositorio de specs (perfiles `knowledge` y `spec-only`) no genera ramas.

**Roles por repositorio (REQ-25.4, S-3).** `fullstack` toca la unión de los repositorios declarados y, en embebido, el del planning home. Un rol declarado toca las raíces git de sus `path`. Cualquier otro rol no aporta nada y genera un aviso. Las ramas son:
- con un rol, `<prefijo>/<cambio>`, con PR hacia la integración;
- con dos o más, la de seguimiento más `<prefijo>/<cambio>-<rol>`, con PR hacia la de seguimiento, y el PR final del cambio, sin archive, hacia la integración.

El aviso «publica la rama de seguimiento antes del primer PR de rol» va en `warnings`. Un rol llamado `archive` se rechaza con `invalid_name`, nombrando la colisión con `<prefijo>/<cambio>-archive` (S-15).

**Antes del sellado (S-1).** Sin `kickoff.yaml`, el plan contiene solo la rama del cambio en el planning home (embebido) o la comprobación del repositorio de specs (multirepo). Después del sellado se completa de forma incremental.

**Carril ODD (REQ-25.13).** Sin roles ni archivado. En embebido, `<prefijo>/<feature>`; en multirepo, el repositorio de specs in situ más los `--repo`. `oddDocument` es `<workspaceRoot del plan>/odd/tasks/<feature>.md`.

**Política (REQ-25.3).** `branch.ResolvePolicy(cfg)` aplica los valores por defecto (`main`, `feat`) y `Policy.For(repoRelPath)`, la entrada de `repositories:`. Las excepciones (`--prefix`, `--branch`, `--base`) van encima. Con instantánea, sus nombres prevalecen (D-10).

**Nombres.** `<cambio>` y `<rol>` deben pasar `gitref.ValidBranchComponent`:
- `^[A-Za-z0-9][A-Za-z0-9._-]*$`;
- sin `..`;
- sin terminar en `.` ni en `.lock`;
- sin nombres reservados de Windows (`CON`, `PRN`, `AUX`, `NUL`, `COM1`–`COM9`, `LPT1`–`LPT9`).

La rama completa debe pasar `gitref.ValidRefName`. Si algo falla, `invalid_name` antes de crear nada.

**Rutas (REQ-25.11, S-4, S-17).**
- El worktree está en `<raíz>/<cambio>[-<rol>|-archive]/<repo>`.
- `<repo>` es el último segmento del `path` declarado (en `repositories:` o en el rol); para `.`, el nombre del directorio de la raíz git.
- Dos repositorios del plan con el mismo `<repo>` dan `invalid_name`, nombrando ambas rutas.
- La raíz es la preferencia del hub o, sin ella, `<padre de la maestra>/<nombre de la maestra>-wt` (`branch.DefaultWorktreeRoot`), siempre con la maestra del checkout principal (D-08).
- `plan_conflict` cuando la raíz queda dentro de un repositorio del plan, o cuando una ruta planificada queda dentro de un worktree existente distinto del suyo (por ejemplo, uno antiguo de los scripts locales).

**Límite de longitud (A-1).** Con `PathRules.GOOS == "windows"`, una ruta de worktree de más de **220** caracteres (260 − 40) se rechaza con `path_too_long`, nombrando la ruta, su longitud y el máximo; 220 exactos se admiten. En otros sistemas no hay límite propio. `PathRules` forma parte de la entrada, así que la regla de Windows se prueba en Linux.

**Alternativas descartadas.** *Límite medido con la ruta versionada más larga* (versión anterior): la spec fija ahora 220 (A-1). *Ramas de rol por los roles de `axiom.yaml`*: contradice D12.

### 2.5 D-05 — Ejecución en dos fases e idempotencia

**Decisión.** `Engine.Ensure`:
1. **Hechos (solo lectura).**
   - `Locate` (D-15).
   - Raíces git y common dir con `gitcli` (`TopLevel`, `CommonDir`; REQ-25.6).
   - Kickoff y registro.
   - Preferencias.
   - Ramas, `git worktree list --porcelain`, `HEAD` y suciedad (solo en `in_place`).
   - Base (D-06).
2. **Plan** (`branch.Plan`).
3. **Comprobación sin efectos** de todos los repositorios antes de mutar ninguno:
   - `decision_required` (sin crear nada, tampoco el registro);
   - `plan_conflict` (`--mode` o una excepción contradicen lo materializado; la repetición idéntica es un no-op);
   - `branch_checked_out_elsewhere`;
   - `dirty_worktree` en `in_place` (`git status --porcelain=v1 -z --untracked-files=normal`; ya en la rama esperada es un no-op);
   - `specs_repo_not_on_integration` y `base_not_found`;
   - `fetch_failed`, con `--fetch`;
   - con `--archive`, además `change_not_integrated` y `archive_branch_not_applicable` (D-25).
4. **Aplicación**, solo lo que falta:
   - `git branch <rama> <base>`;
   - la de rol, desde la de seguimiento;
   - en `worktree`, `git worktree add <ruta> <rama>` con `cmd.Dir` en el checkout principal, que nunca cambia;
   - en `in_place`, `git switch <rama>`;
   - nunca mueve ni reinicia una rama existente; reutiliza los worktrees registrados o encontrados en la ruta planificada.
5. **Registro** (D-07). Solo se escribe si cambian sus bytes.
6. **Resultado.** El JSON describe el **estado final** y nunca las acciones (sin `created`, `changed` ni marcas de tiempo): la segunda ejecución da el mismo JSON byte a byte (REQ-25.7).

Un fallo de git durante la aplicación da `error` con `git_failed`, que nombra la operación y resume el error; repetir `ensure` completa lo pendiente. Un bloqueo exclusivo por cambio impide dos `ensure` a la vez; con el bloqueo ocupado, `plan_conflict` («otra ejecución de `axiom branch` está modificando el cambio»). `ensure --archive` es un plan aparte con las mismas garantías y nunca toca las ramas del cambio (REQ-25.7).

### 2.6 D-06 — Base sin red y `--fetch`

**Decisión.**
- **Base.** `refs/remotes/origin/<integración>` si existe; si no, `refs/heads/<integración>`; si no existe ninguna, `base_not_found`. `--base <ref>` se resuelve con `git rev-parse --verify --quiet <ref>^{commit}`. Las ramas de rol salen de la de seguimiento local.
- **JSON.** `base.ref`, `base.sha`, `base.source` (`remote-tracking` o `local`) y `base.fetched`.
- **`--fetch`.** `git fetch origin`, sin `--prune`, en cada repositorio del plan durante la fase 3. Si falla, `fetch_failed` y no empieza la fase 4 (S-8).
- **Doctrina.** Las skills nunca añaden `--fetch` sin autorización explícita (contrato de autorización remota).
- **Misma referencia en todo el motor.** La resolución la usan también `ensure --archive` (D-25) y la lista de pendientes de archivar (D-17). En esta última, la falta de la referencia de integración solo deja un aviso.

### 2.7 D-07 — Registro local (Q2)

**Decisión.** Un fichero por cambio en `<common dir>/axiom/branch/v1/<lane>/<cambio>.json`, con su `<cambio>.lock`. El common dir es el del repositorio del planning home:
- `ensure`, `status` y `done` lo obtienen con `gitcommon.CommonDir` (`git rev-parse --git-common-dir`, REQ-25.14);
- los lectores que no deben lanzar procesos (`VerdictFor`, guarda D, listados) usan `gitcommon.CommonDirFromFiles`.

Una prueba comprueba que las dos vías dan el mismo directorio en un clon normal, en un worktree enlazado y con `git init --separate-git-dir`.

- **Contenido** (`axiom.branch-registry/v1`, §4.4): carril, cambio, modo y su origen (`answer`, `flag`, `hub` o `default`), excepciones, prefijo de archivado si se indicó, raíz git del planning home y, por repositorio, nombre, raíz git, **common dir**, papel y ramas. Cada rama lleva su tipo (`change`, `role` o `archive`), su rol y el worktree absoluto o `inPlace`. Sin marcas de tiempo, con serialización estable.
- **Coexistencia con `gentle-ai/`.** Espacios de nombres separados: el registro nunca toca `gentle-ai/` y el reinicio del almacén de revisión solo actúa sobre su lista blanca de rutas bajo `gentle-ai/`. Los linajes RDD no cambian.
- **Reconstrucción.** Sin fichero, `ensure` lo reconstruye desde las ramas planificadas existentes, los worktrees en las rutas planificadas y el modo deducido. `status` informa de `registry.present: false`.
- **Fichero corrupto.** `config_invalid`, nombrando el fichero y la continuación («bórralo; `ensure` lo reconstruye»).
- **Vistas de candidato de RDD.** Son worktrees desacoplados y no cuentan para `branch_checked_out_elsewhere`.

**Alternativas descartadas.** *Dentro de `gentle-ai/`*: mezcla ciclos de vida con un espacio de upstream. *Migrar `gentle-ai/`*: fuera de alcance. *Un fichero para todos los cambios*: contención. *El registro en `kickoff.yaml`*: datos personales en un fichero versionado.

### 2.8 D-08 — Modo de trabajo y preferencias del hub

**Decisión.**
- **Datos.** `hub.WorkspaceRecord` gana `WorktreeMode` y `WorktreeRoot` (`json:"worktree_mode,omitempty"` y `json:"worktree_root,omitempty"`). El esquema sigue en `"1.0"`.
- **API.** `Register` no cambia y ya conserva los campos de un registro que actualiza; `Unregister` borra la entrada entera. Métodos nuevos en `internal/hub/prefs.go`:
  - `SetWorktreePrefs`;
  - `LookupWorktreePrefs`, que busca por `pathidentity.SameDirectory`;
  - `ValidateWorktreePrefs`.
- **Identidad del proyecto.** Es la maestra que resuelve `Locate` desde el checkout principal del repositorio del planning home (`gitcommon.MainWorktreeFromFiles`), nunca desde un worktree.
- **Modo efectivo.** Prioridad: registro > `--mode` > hub > defecto (`worktree`). Si el hub no se puede leer, se usan los valores por defecto con un aviso. La respuesta a `ask` se registra por cambio y vale también para `--archive` (REQ-25.8).
- **Registro (REQ-25.30).** `axiom init` y `axiom project add` ganan `--worktree-mode` y `--worktree-root`. Con terminal interactiva (el criterio de `reviewConsoleInteractive`) y sin esas flags, preguntan con los valores por defecto; aceptarlos no guarda nada.
- **Worktrees no registrables (REQ-25.31).** `hub.IsGitWorktree` pierde la heurística del segmento `axiom-wt` y usa `gitcommon.LinkedWorktreeRoot`. El rechazo nombra el checkout principal.

### 2.9 D-09 — `done`: fusión por capas y desmontaje (Q1)

**Decisión.**

**Destinos.** Rama de rol → seguimiento local. Rama del cambio y rama de archivado → integración, evaluada contra cada referencia existente entre `origin/<integración>` y `<integración>`. Solo con `--fetch` se actualiza antes.

**Capas** (`branch.ConfirmMerge`; se detiene en la primera que confirma):
1. `ancestry`: `git merge-base --is-ancestor <rama> <destino>`.
2. `equivalent_tree`: `git merge-tree --write-tree --no-messages <destino> <rama>` con salida 0 y primera línea igual a `git rev-parse <destino>^{tree}`.
3. `pr_merged`: el `gates.yaml` del cambio (activo o archivado, con `kickoff.ArchivedChangeRoot`) tiene un registro aprobado de la compuerta `integration` con `evidence_kind: pr_merged` y `verified: true`. **No aplica a la rama de archivado** (REQ-25.16), que solo admite las capas 1 y 2.
4. Ninguna: `merge_not_confirmed`; con `--force`, `forced`.

**Capacidad (Q1).** `gitcli` lee `git version` una vez por proceso y habilita la capa 2 con 2.38.0 o superior. Si la versión no se puede leer, intenta la orden y trata la salida 129 como «no admitida». La capa omitida genera un aviso. La variable de prueba `gitcli.MergeTreeSupportedOverride *bool` cubre la omisión. Git local: 2.41.0 (verificado). El de CI no está declarado: el diseño funciona con ambos y la prueba real se salta por debajo de 2.38.

**Desmontaje (REQ-25.17).** Ramas de rol primero, después la del cambio y la de archivado, cada una por separado:
- árbol sucio → `dirty_worktree`, salvo `--force` (con la lista de lo descartado);
- sin fusión confirmada → `merge_not_confirmed`, salvo `--force` (con la punta SHA);
- en `worktree`, `git worktree remove [--force]` desde el checkout principal y, solo si tiene éxito, `git branch -D`;
- en `in_place`, con el árbol limpio, `git switch <integración>` y `git branch -D`; con el árbol sucio, `dirty_worktree` también con `--force`;
- el registro se actualiza y se borra cuando queda vacío; nunca se toca una rama remota;
- resultado `ready` o `partial` (código 1);
- `LineageProbe.OpenLineages(worktree)` añade el aviso con `axiom review abandon` si hay un linaje RDD abierto (A-4).

En Windows, un worktree que es el directorio actual de algún proceso no se puede borrar: la rama se conserva y el mensaje pide `--cwd` sobre el checkout principal.

### 2.10 D-10 — Veredicto único

**Decisión.** `branch.Decide(in VerdictInput) Verdict` es puro y devuelve `correct`, `wrong_branch` o `not_applicable` (con su etiqueta).

**Contexto y consultante** (un único `VerdictQuery` corrige B3):
- `Context`: `ordinary` o `archive`.
  - En la guarda A, cada escritor fija el contexto (D-11).
  - En la guarda B es `archive` en la fase archive (D-13).
  - La guarda D usa la unión de los dos (G-11).
- `Consultant`: `write`, `status`, `apply` o `guard`. Solo `apply` evalúa los repositorios de código de un `multirepo`.

| Papel | `ordinary`: correcto | `archive`: correcto | `wrong_branch` | `not_applicable` |
|---|---|---|---|---|
| `change_branch` | Rama del cambio o cualquiera de rol | Solo la rama de archivado | Cualquier otra, incluidas la integración, la de archivado en el contexto ordinario y un HEAD desacoplado | — |
| `specs_in_place` | Solo la integración | Solo la integración (sin rama de archivado) | Cualquier otra | — |
| `code` | Rama del cambio o de rol, con consultante `apply` | Igual | Otra, con consultante `apply` | Consultante distinto de `apply` |
| Exenciones | — | — | — | Sin instantánea (REQ-25.37); sumideros `e2e-cumulative` y `cumulative-<rol>`; ruta fuera de git o `HEAD` ilegible (con aviso) |

**Nombres esperados.**
- Rama del cambio y de rol: (1) instantánea; (2) sin `kickoff.yaml`, registro (excepciones de REQ-25.12); (3) política.
- Rama de archivado: registro si `ensure --archive` recibió `--prefix`; si no, la instantánea; si no, `<rama del cambio>-archive`.

**Hechos sin procesos.** `Engine.VerdictFor(ctx, VerdictQuery)` obtiene todo leyendo ficheros:
- `Locate`;
- `gitcommon.GitRootFromFiles`;
- `gitcommon.ReadHead`;
- `kickoff.Load`;
- registro (por `CommonDirFromFiles`);
- política.

Nunca lanza git, de modo que las tres guardas cumplen a la vez REQ-25.21 («lee `HEAD` sin lanzar procesos») y REQ-25.42 («si git no está disponible, no bloquea»). El `error` se reserva para fallos internos (kickoff, registro o configuración ilegibles).

**Continuación.** `axiom branch ensure --cwd <maestra del checkout principal> --change <c>`, más `--lane odd` (ODD) o `--archive` (contexto de archivado en un repositorio con rama de cambio), citada con `pathquote.Quote`.

El veredicto no consulta los cambios activos de la integración (REQ-25.18).

### 2.11 D-11 — Guarda A

**Decisión.** `engine.RequireChangeBranch(ctx, WriteTarget) (warning string, err error)` llama a `VerdictFor` con `Consultant: write` y el contexto del escritor:
- con `correct` o `not_applicable`, devuelve `nil` y el aviso;
- con `wrong_branch`, devuelve `*branch.RefusalError` (`wrong_branch` y la continuación);
- ante un fallo interno, devuelve `config_invalid`: la guarda falla en cerrado.

No puede ir dentro de `kickoff.Seal`, `kickoff.AppendGate` ni `handoff.WriteFile`: `engine` importa `kickoff`, `kickoff` importa `handoff`, y esos escritores no conocen el cambio ni el contexto.

| Escritor (REQ-25.19) | Punto de entrada | Contexto |
|---|---|---|
| Creación de incremento | **Nuevo** `Service.writeIncrement` en `internal/dashboard/service.go`, extraído del cuerpo de escritura de `CreateIncrement` (hoy no existe). `CreateIncrement`, que también usa `axiom change create` (`cmd/axiom/main.go:547`), ejecuta `ensure` y después llama a `writeIncrement` (G-6) | `ordinary` |
| Migración al acumulativo | `Service.MigrateIncompleteTasksToCumulative` (`service.go:864`): cambio de origen; el destino `cumulative-<rol>` está exento por nombre | `ordinary` |
| Handoff (CLI) | `runHandoffCreate` (`cmd/axiom/main.go`, antes de `handoff.WriteFile`, :750) | `ordinary` |
| Handoff (dashboard) | `Service.CreateHandoff` (`service.go:1123`) | `ordinary` |
| `kickoff seal` | `runSDDKickoffSeal` (`internal/cli/sdd_kickoff.go`), salvo `--infer` | `ordinary` |
| `gate record` (y el handoff de integración tras la última compuerta de rol) | `runSDDGateRecord` (`internal/cli/sdd_gate.go`), antes de `AppendGate` | `ordinary`; **`archive` con `--gate integration`** (G-10) |
| Marcador de instancia del consentimiento | `RunSDDContinue` (`internal/cli/sdd_status.go:59`), el único llamador de `PrepareChangeInstanceConsent`. `sdd-attempt grant` no escribe el marcador (verificado) | `ordinary` |
| `integration.yaml` | `axiom sdd integrate prepare` y `verify` (`internal/cli/sdd_integrate.go`) | **`archive`** (G-10) |
| `archive-compose` sobre un cambio | Modo con `--change` o delta bajo `changes/<c>/` (`internal/cli/sdd_archive_compose.go`) | `archive` (REQ-25.19) |

**`RunSDDContinue`.** Si el status ya proyecta `blocked(wrong_branch)`, no prepara el marcador y muestra el bloqueo (REQ-25.42). En ese caso la preparación sería un no-op de todos modos, porque la autoridad de edición no se evalúa con la rama equivocada (D-13). La guarda A solo protege la ventana entre la resolución y la escritura.

`ContinueIncrement` del dashboard llama a `RunSDDContinue` y hereda la guarda. La TUI no escribe en carpetas de cambio (verificado: su único `MkdirAll` es el del índice de skills).

**Prueba de inventario.** `internal/branch/engine/writer_inventory_test.go` recorre con `go/parser` `cmd/axiom`, `internal/cli`, `internal/dashboard`, `internal/tui` e `internal/sddstatus`. Exige que toda función que llame a `kickoff.Seal`, `kickoff.AppendGate`, `handoff.WriteFile` o `sddstatus.PrepareChangeInstanceConsent`, o que escriba bajo `changes/`, figure en la tabla. Cada escritor tiene además su prueba de negativa.

### 2.12 D-12 — Instantánea en `kickoff.yaml` y `seal`

**Decisión.** `kickoff.Kickoff` gana `Git *GitSnapshot` (`yaml:"git,omitempty"`), en el fichero nuevo `internal/kickoff/gitsnapshot.go` (§4.5).
- **Contenido** (solo política). Integración y prefijo resueltos, excepciones y, por repositorio, ruta relativa a la maestra, nombre, papel, integración, base (nombre de rama) y ramas derivadas: `change`, `role` y, en los repositorios con rama de cambio, `archive` con su nombre (REQ-25.36). Sin SHAs ni rutas absolutas.
- **Lectura.** `ParseKickoff` sigue siendo no estricto (los binarios anteriores ignoran `git:`). `Validate` comprueba la forma con `gitref` si el bloque existe.
- **Orden en `runSDDKickoffSeal`.**
  1. Sin `--infer`, `RequireChangeBranch`. Sin kickoff todavía, los nombres salen del registro o la política más las ramas de rol de los roles que se van a sellar.
  2. `engine.Snapshot(ctx, loc, change, roles)`.
  3. `kickoff.Seal`.
- **Escritura única.** Un segundo `seal` devuelve el ganador; ninguna otra orden modifica la instantánea.
- **`--infer`.** Ni comprueba la rama ni escribe instantánea (REQ-25.37).
- **Multirepo.** Solo se comprueba el repositorio de specs (REQ-25.38).

### 2.13 D-13 — Guarda B en `axiom sdd status`

**Decisión.** Fichero nuevo `internal/sddstatus/branch_context.go` con `resolveBranchContext(...) branchContext`, que llama a `engine.VerdictFor` a través de la variable de prueba `branchEngine`.

**Fase archive.** En un repositorio con rama de cambio, el status está en la fase archive cuando `applyState == ApplyComplete` (todas las tareas cerradas) y `artifacts["verify-report"] == ArtifactDone`. Entonces el contexto es `archive` y la única rama correcta es la de archivado, con la continuación `ensure --archive` (REQ-25.42, REQ-25.39). En el resto de casos el contexto es `ordinary`. Con un `verify-report` desfavorable, el orquestador reabre tareas y el estado vuelve a `ordinary`. En el repositorio de specs de un `multirepo` el esperado es siempre la integración.

**Costura** en `resolveByPreferenceOrder`: un único punto, después de `resolveApplyState` (el `applyState := resolveApplyState(...)` actual) y de `loadGovernance`, y antes de `applyEditAuthorityBlock`. No se pasa ninguna opción nueva a `loadGovernance` (corrige B4).

```go
// internal/sddstatus/status.go (fragmento ilustrativo de la costura)
bctx := resolveBranchContext(branchQuery{
	workspaceRoot: workspaceRoot,
	changeRoot:    changeRoot,
	change:        changeName,
	archivePhase:  applyState == ApplyComplete && artifacts["verify-report"] == ArtifactDone,
	apply:         applyState == ApplyReady || applyState == ApplyInProgress,
	tasksSources:  tasksSources(changeRoot, governance), // tasks.md y tasks.<rol>.md
})
var unauthorizedRoots []string
if bctx.wrongBranch != nil {
	blockedReasons.genuine = append([]string{bctx.wrongBranchReason()}, blockedReasons.genuine...)
	applyState = ApplyBlocked // S-9: edit authority is not evaluated on the wrong branch
} else {
	allowed := append(append([]string{workspaceRoot}, grantedRoots...), bctx.registeredRoots...)
	applyState, unauthorizedRoots = applyEditAuthorityBlock(applyState, &blockedReasons, bctx.tasksSources, allowed, bctx.rebase)
}
blockedReasons.notes = append(blockedReasons.notes, bctx.notes...)
// … dependencies and nextRecommended as today …
if bctx.wrongBranch != nil {
	nextRecommended = "resolve-blockers"
}
// baseStatus receives append(append([]string{}, grantedRoots...), bctx.registeredRoots...)
```

- **Motivo.** `blocked(wrong_branch): <repo> está en <actual>; se esperaba <esperada>; ejecuta <continuación>` va en primera posición. Sin sobre de consentimiento, porque la continuación es una orden.
- **Orden S-9.** Con la rama equivocada no se evalúa la autoridad de edición, así que tampoco hay sobre de consentimiento; tras corregir la rama aparece `edit_authority_missing`.
- **`sdd continue`.** Hereda el mismo status: con `resolve-blockers` no autoriza ninguna fase (REQ-25.42).
- **Contrato v2.** Solo crece: un texto en `blockedReasons` y raíces adicionales en `allowedEditRoots`.
- **Compuertas.** `loadGovernance`, `archiveDependencyFromGovernance` y la compuerta `integration` **no cambian**. Con D16, la evidencia `pr_merged` del PR del cambio existe antes del archive y se registra en la rama `-archive` (G-10). REQ-21.16 se cumple sin excepciones.
- **Exenciones y avisos.** Los cambios exentos no se proyectan; los avisos (sin git, ruta no git, `HEAD` ilegible) van a `notes`.

**Alternativas descartadas.** *Un decorador posterior* (como el de inc-24): el orden S-9 y las raíces tienen que entrar antes de `applyEditAuthorityBlock`. *Detectar la fase archive con `nextRecommended`*: `nextRecommended` depende de `applyState`, que la propia guarda modifica (dependencia circular).

### 2.14 D-14 — Raíces registradas y resolución de rutas de tareas (R5)

**Decisión.**
- **Raíces.** `registeredRoots` son los worktrees del cambio en el registro (incluido el de archivado) y, en `in_place`, la raíz del repositorio. Sin registro, la lista está vacía.
- **Fuentes de tareas.** `tasks.md`, con base en el worktree del cambio (o `workspaceRoot`), y cada `tasks.<rol>.md` del kickoff, con base en el worktree de su rama de rol si existe. Hoy solo se lee `tasks.md`.
- **Reubicación.** Una ruta que, resuelta, cae dentro del **checkout principal** de un repositorio registrado se reescribe con el mismo sufijo en el worktree correspondiente (el del rol para `tasks.<rol>.md`). Las rutas de `tasks.md` siguen siendo relativas al repositorio (por ejemplo, `../repo-backend/…`) y nunca personales.
- **Costura (B3).** Ganan el parámetro `rebase func(string) string`:
  - `detectUnauthorizedEditRoots`;
  - `detectUnauthorizedEditRootsForCurrentWorkUnit`, la variante que usa la rama de apply a través de `applyEditAuthorityBlock` y que pasa `rebase` a sus dos llamadas internas;
  - `applyEditAuthorityBlock`, que además recibe las fuentes de tareas en lugar de un único texto.

  La identidad (`func(p string) string { return p }`) y una única fuente `tasks.md` reproducen el comportamiento actual.
- **Repositorio no registrado.** Sigue exigiendo `axiom sdd attempt grant`.

### 2.15 D-15 — Resolución desde worktrees enlazados (R4, S-11)

**Decisión.** En `workspace.locate`, con ascensión, y **solo con lecturas de ficheros** (`workspace` nunca lanza procesos):
1. `gitcommon.LinkedWorktreeRoot(start)` detecta si `start` está dentro de un worktree enlazado.
2. Dentro de un worktree enlazado, la ascensión se acota a su raíz y gana el marcador más cercano dentro de él (REQ-25.28.1). En embebido, el `axiom.yaml` del propio worktree.
3. Sin marcador dentro, se resuelve desde `gitcommon.MainWorktreeFromFiles`, con nota (punto 2). Si el checkout principal no se puede determinar por ficheros (common dir que no se llama `.git`), o tampoco tiene marcador, la resolución es `OriginStart` con nota.
4. La ubicación del worktree nunca alimenta a `deduceMaster` (punto 3).

`runWorkspaceValidate` conserva `--path` como maestra. Si `--path` no tiene marcador y está en un worktree enlazado, toma la maestra y la configuración de `Locate(--path)`. `LocateAt` no cambia.

### 2.16 D-16 — Cambios que viven en worktrees registrados (R6)

**Decisión.** `engine.RegisteredChanges(ctx, loc) ([]RegisteredChange, []string)` lista los registros SDD del common dir del planning home (por ficheros). Por cada entrada con worktree del planning home devuelve:
- nombre, rama, tipo (`change` o `archive`), raíz del worktree y su planning home;
- `changeRoot = <planning home del worktree>/changes/<cambio>`, si existe.

Una entrada con el worktree desaparecido se omite con aviso. Cada worktree aporta solo su cambio. **Precedencia:** worktree de archivado > worktree del cambio > checkout principal (REQ-25.34, REQ-25.44).

| Consumidor | Uso |
|---|---|
| `sddstatus.Resolve` (fichero nuevo `worktree_changes.go`) | Antes de `resolveByPreferenceOrder`. Con nombre, se reenraíza en el worktree que gane la precedencia. Sin nombre, la unión de candidatos sigue el orden de preferencia vigente: con uno, se selecciona; con varios, el `select-change` actual sobre la unión |
| `axiom sdd continue` | Hereda `Resolve` |
| Dashboard (`GetIncrements`, `FindIncrementPath`, `requireActiveIncrement`, detalles de roles y handoffs) | Agrega los cambios registrados con la misma precedencia; `IncrementSummaryDTO` gana `worktreePath`, `branch` y `pendingArchive` (`omitempty`) |
| TUI | Mismo conjunto que `GET /api/increments`; solo lista |

### 2.17 D-17 — Cambios pendientes de archivar (REQ-25.20, D17)

**Decisión.** `engine.PendingArchive(ctx, loc) ([]PendingChange, []string)`:
- **Aplicabilidad.** Solo en repositorios con rama de cambio. En el repositorio de specs de un `multirepo` la lista está vacía (no aplica).
- **Referencia.** La de integración de D-06 (`origin/<integración>` si existe en local y, si no, la local), sin cambiar de rama ni operaciones remotas. Sin referencia, la lista se omite con un aviso y sin error.
- **Lectura.** `gitcli.TreeDirs(ref, "<ruta del planning home relativa al repositorio>/changes")` (`git ls-tree -z`): solo cuentan las entradas de tipo `tree`; se excluyen `archive`, `e2e-cumulative` y `^cumulative-[A-Za-z0-9._-]+$`.
- **Resultado.** Cada pendiente lleva su nombre y la continuación `axiom branch ensure --change <c> --archive`.

**Consumidores.**
- `axiom branch status` sin `--change` lista los registrados y los pendientes. Con `--change`, añade `pendingArchive` del cambio. Siempre con salida 0 si nada más falla.
- El dashboard marca `pendingArchive` en el listado y emite en `X-Axiom-Notes` el aviso con los nombres (G-9).

No hay código de fallo, CI ni bloqueo: el veredicto y las guardas no consultan esta lista. Se elimina el invariante D11 (la capa C desaparece).

**Alternativa descartada.** *Leer el sistema de ficheros*: no es determinista con directorios sin seguimiento y exigiría estar desplegado en la integración.

### 2.18 D-18 — `axiom branch guard` (D19) e instalación por `axiom sync`

**Decisión.**

**Algoritmo.** Presupuesto de 1 s. No lanza procesos (lo comprueba un observador de procesos) y falla en abierto: cualquier error o pánico recuperado termina sin bloquear ni avisar. No escribe ficheros.
1. **Carga.** Lee la entrada estándar (máximo 256 KiB) y extrae el evento (`pre` o `post`), la herramienta, las rutas destino y `cwd`:
   - Claude Code: `hook_event_name`, `tool_input.file_path` o `notebook_path`;
   - Codex: `hook_event_name` y las cabeceras `*** Add/Update/Delete File:` y `*** Move to:` del parche de `apply_patch` (A-2);
   - OpenCode: `{event, tool, filePath, cwd}`, que envía su plugin.

   Las rutas se limpian y se hacen absolutas contra `cwd`.
2. **Clasificación léxica** (sin procesos, sin distinguir mayúsculas en Windows). Un **documento de Axiom** tiene los segmentos `openspec/changes/<x>/…` (en `changes/archive/<AAAA-MM-DD>-<x>/…`, el cambio es `<x>`) o `odd/tasks/<f>.md`. Cualquier otra ruta es **otro fichero**.
3. **Bloqueo** (solo evento `pre` y documento de Axiom). `Locate` (ficheros): con `OriginStart`, nada. Se confirma la contención bajo `loc.PlanningHome/changes` o `loc.WorkspaceRoot/odd/tasks`; los sumideros están exentos. `VerdictFor{Consultant: guard}` evalúa los dos contextos y **bloquea si es `wrong_branch` en ambos** (G-11). Rige sin cambio registrado, con los nombres de la política (S-18). El mensaje incluye la continuación (`--lane odd` para `odd/tasks`).
4. **Aviso** (solo evento `post` y otro fichero).
   1. Se calcula la raíz git de la ruta (`GitRootFromFiles`) y se ejecuta `Locate`: con `OriginStart`, nada; si la topología es `multirepo` y la raíz es la del repositorio de specs, nada (exento).
   2. Se leen los registros activos (SDD y ODD) del common dir del planning home (`CommonDirFromFiles`) cuyos `repositories[].commonDir` coinciden con el common dir de la raíz de la ruta: sin coincidencias, nada.
   3. Se lee el `HEAD` del checkout que contiene el fichero (`ReadHead`): si es una rama registrada de esos cambios (cambio, rol o archivado), nada.
   4. Si no, se avisa nombrando, por cada cambio, el worktree correcto o, en `in_place`, la rama esperada.
5. Cualquier otra combinación (`pre` con otro fichero, `post` con un documento) no hace nada: cada evento tiene una única responsabilidad, así que el aviso nunca se duplica.

**Canales por runtime.**

| Runtime | Bloqueo (documentos de Axiom) | Aviso sin bloquear (otros ficheros) | Verificado o supuesto |
|---|---|---|---|
| Claude Code | `PreToolUse` con matcher `Write\|Edit\|MultiEdit\|NotebookEdit` y `axiom branch guard --agent claude-code`: salida 2 con el motivo por `stderr` | `PostToolUse` con el mismo matcher y la misma orden: salida 2 con el aviso por `stderr`. Claude Code devuelve el texto al agente y, como la escritura ya ocurrió, no bloquea nada | **Verificado en este repositorio:** `scripts/claude-hooks/guard-primary-checkout.ps1` usa exactamente esos dos canales (`PreToolUse` con salida 2 para denegar y `PostToolUse` con salida 2 para «devolver el aviso a Claude») y `ensureClaudeSDDPreflightHook` instala entradas `PreToolUse`. **Supuesto:** que `MultiEdit` siga existiendo (es inocuo si no existe; A-5) |
| Codex | `PreToolUse` sobre `apply_patch`, salida 2 | `PostToolUse` sobre `apply_patch` con el canal de retroalimentación al agente que documente Codex. Si no existe, Codex queda sin aviso y solo bloquea | **Supuesto (A-2, A-5):** la exploración cita la documentación oficial (PreToolUse y PostToolUse para `apply_patch`; la salida 2 bloquea). El canal de retroalimentación de `PostToolUse` y la forma de la carga se verifican al implementar S11 |
| OpenCode | Plugin nuevo `axiom-branch-guard.ts`, `tool.execute.before` para `write`, `edit` y `patch`: `spawn` de `axiom branch guard --agent opencode` con la carga por `stdin`; con `{"action":"block"}` lanza un `Error` con el mensaje | `tool.execute.after` del mismo plugin: con `{"action":"warn"}` añade el aviso a `output.output` | **Verificado en este repositorio:** `opencode-review-transport.ts` reescribe `output.output` en `tool.execute.after` (pin en `internal/assets/assets_test.go:607`) y `sdd-task-result-artifacts.ts` usa `tool.execute.before`. **No verificado:** que un `throw` en `before` bloquee en todos los carriles (la exploración cita excepciones tragadas; R10) |
| Pi | — | — | Sin canal (REQ-25.22) |

**Protocolo de salida.** Con `--agent claude-code` y `--agent codex`, el resultado es el código de salida más el texto por `stderr`. Con `--agent opencode`, la salida es siempre 0 y por `stdout` va `{"action":"none|block|warn","message":"…"}`, que interpreta el plugin.

**Instalación por `axiom sync`** (patrón `ensureClaudeSDDPreflightHook`: identidad por la cadena exacta de la orden, `filemerge.WriteFileAtomic` y conservación de las entradas ajenas):
- Claude: `ensureClaudeBranchGuardHook` instala las dos entradas (`PreToolUse` y `PostToolUse`).
- Codex: `ensureCodexBranchGuardHook` en `hooks.json`.
- OpenCode: el plugin se añade a la lista de `inject.go` y a `opencode_plugin_digests.go`.
- Pi: no se instala nada.
- Una segunda ejecución da bytes idénticos. La retirada sigue el patrón `retireClaudeTelemetryHooks` cuando el componente SDD del runtime ya no está instalado.
- Nunca hay hooks de git.
- Con la instalación global de Claude, en un repositorio ajeno o sin cambio registrado, el hook no hace nada y solo lee ficheros (REQ-25.22).

**Alternativas descartadas.**
- *Avisar en `PreToolUse` con salida 0 y JSON*: no hay evidencia en el repositorio de que el texto llegue al agente.
- *Bloquear también el código*: lo descarta D19.
- *Consultar git para leer `HEAD`*: lo prohíbe REQ-25.21.
- *Hook de Bash*: sin ruta destino no hay nada que clasificar.

### 2.19 D-19 — Doctrina: `routing.go`, orquestadores y skills

**Decisión.**
1. **`internal/components/agentguidance/routing.go`** (REQ-25.40, REQ-25.41).
   - **Paso 6.** Solo cambia la cláusula «on the feature branch, branch first when on the default branch», que pasa a ser «on the branch that `axiom branch ensure --lane odd --change <feature-name>` prepared before the first write of `odd/tasks/<feature-name>.md` (see *Branches and worktrees*)». Los demás pasos no cambian.
   - **Subsección nueva «### Branches and worktrees»**, idéntica para todos los adaptadores:
     - `axiom branch ensure --lane odd --change <feature-name> --json` antes de la primera escritura del documento o de código;
     - `--cwd` y `git -C` con las rutas absolutas del JSON;
     - `decision_required` es la única interacción;
     - nunca `--fetch` sin autorización;
     - la línea única: «Never use `EnterWorktree` or `ExitWorktree`; work in another worktree only through the absolute paths that `axiom branch ensure` returns, with `--cwd`, `git -C`, and absolute paths.»
2. **`_shared/sdd-orchestrator-sections.md`, sección de kickoff** (REQ-25.35), fuente de los doce `assets/<runtime>/sdd-orchestrator.md`. Tras elegir el carril y nombrar el cambio, y antes del pre-vuelo y de `kickoff seal`, se ejecuta `axiom branch ensure --change <name> --json` y se usan sus rutas con `--cwd`. `decision_required` es una preferencia de entorno (S-10). Repetirlo al reanudar es idempotente.
3. **Skills** (REQ-25.23):

| Skill | Cambio |
|---|---|
| `sdd-apply` | Se retiran el Paso 2c y las reglas ODD-5.1, 5.2, 5.5 y 5.6. Pasan a ser: `axiom branch ensure --cwd "<workspaceRoot>" --change <c> --json` antes de la primera unidad; edición solo en `repositories[].branches[].worktree` del rol activo; destino del PR a partir de `prTarget` de `axiom branch status --json`; la persona publica la rama de seguimiento antes del primer PR de rol |
| `sdd-spec`, `sdd-design` | Se retiran ODD-5.1 y ODD-5.2. Pasan a ser: la rama la prepara `ensure`; se escribe bajo `workspaceRoot`; antes del handoff, los artefactos se confirman en la rama del cambio; publicar es decisión de la persona |
| `sdd-propose`, `sdd-explore` | Una línea: rama preparada por `ensure`; `--cwd` y rutas del status |
| `sdd-archive` | Flujo D16 en monorepo: tras fusionar el PR del cambio, `axiom branch ensure --change <c> --archive` (con `change_not_integrated`, sugiere `--fetch` con la autorización de la persona); en la rama `-archive`, `gate record --gate integration --evidence-kind pr_merged`, `integrate prepare`, `archive-compose`, movimiento a `changes/archive/` e `integrate verify`; PR documental aparte hacia la integración; tras fusionarlo, `axiom branch done`. Si el PR documental se rechaza, se descarta la rama y el cambio sigue pendiente de archivar. En multirepo, in situ con la evidencia de REQ-21.16 |
| `_shared/sdd-phase-common.md` (sección G) | Rutas absolutas del JSON, `git -C` y `--cwd` |
| `_shared/sdd-status-contract.md` | `blocked(wrong_branch)` como segundo motivo, sin sobre, con continuación ejecutable, precedencia sobre `edit_authority_missing` y la rama `-archive` en la fase archive |
| Literales | «la rama de integración (`git.integration_branch`)» en lugar de `main`/`master` (D-22) |

Los goldens y `.codex/AGENTS.md` se regeneran con `-update`. El pin de `assets_test.go` comprueba que `EnterWorktree` y `ExitWorktree` solo aparecen en la línea de `routing.go` y que esa línea es idéntica en todos los adaptadores.

### 2.20 D-20 — Dashboard

**Decisión.**
- **Creación (REQ-25.32).** `CreateIncrement` ejecuta `engine.Ensure(ctx, {Change, Lane: sdd, Mode: req.Mode})` y, solo con `ready`, llama al escritor guardado nuevo `writeIncrement(result.PlanningHome, …)` (D-11).
  - `decision_required` responde 409 con `{status, statusLabel, code, decision}`; la SPA repite la petición con `mode`.
  - Cualquier otro rechazo responde 409 con `{code, error, continuation}` y no crea carpeta.
- **Preferencias.**
  - `POST /api/projects/add` y `POST /api/projects/init` admiten `worktree_mode` y `worktree_root`.
  - `GET /api/projects` devuelve las preferencias guardadas y las efectivas.
  - `GET|PUT /api/projects/active/worktree-prefs` responde 400 con un valor inválido, sin cambiar el hub.
  - El modal de registro gana los dos campos.
- **Solo lectura.** `GET /api/workspace` gana `branchPolicy{integrationBranch, branchPrefix}` y `repositories[]`. `/api/workspace` ya responde 405 a cualquier método que no sea `GET` (verificado en `server.go:283`).
- **Listados (D-16, D-17).** Agregación con precedencia y `pendingArchive` en el listado. La cabecera `X-Axiom-Notes` (array JSON) lleva los pendientes de archivar, los worktrees desaparecidos y la falta de la referencia de integración. El cuerpo sigue siendo un array.

### 2.21 D-21 — Dogfooding y destino de los scripts locales (Q6)

**Decisión** (Q6 queda resuelta; nada se aplaza):

| Elemento | Destino | Motivo |
|---|---|---|
| `axiom.yaml` | `git: {integration_branch: main, branch_prefix: inc}` (A-3) | REQ-25.24 |
| `scripts/axiom-worktree.ps1` y `.sh` | **Envoltorio fino.** `new <slug> [-Prefix p]` → `axiom branch ensure --cwd <checkout principal> --change <slug> [--prefix p] --json`, y muestra la ruta del worktree. `done <slug>` → `axiom branch done --cwd <checkout principal> --change <slug>` y, después, el avance rápido de `main` que ya hace hoy. Se retira el paso de desregistro del worktree en el hub (REQ-25.31 ya no los registra). `pr <slug>` se conserva como entrega personal (`push` y PR), con la base tomada de `prTarget` de `axiom branch status --json`. Si `axiom branch --help` falla, el envoltorio termina con el mensaje «instala un `axiom` con `axiom branch` (`go install ./cmd/axiom`)» | Una sola implementación de ramas y worktrees (la del producto). El script aporta lo que D3 deja fuera del producto (`pr`) y la costumbre local. Los worktrees pasan de `axiom-wt/<slug>` a `axiom-wt/<slug>/axiom` |
| `scripts/cleanup-worktree.ps1` y `.sh` | **Retirada** | Solo los citan ODD-5.6 (que se retira) y el manual; los sustituye `axiom branch done` |
| `scripts/claude-hooks/guard-primary-checkout.ps1` | **Se conserva como política local por ruta** (G-12) y sus mensajes remiten a `axiom branch ensure`, en lugar de a `axiom-worktree.ps1 new` | Ni el envoltorio ni la retirada conservan su protección: deniega *cualquier* escritura en el checkout principal y avisa tras Bash o PowerShell si el checkout sale de `main` (patrón #615). La guarda D no cubre Bash y en el código solo avisa con un cambio registrado |
| `CLAUDE.md`, `AGENTS.md`, `GEMINI.md` | Remiten a `axiom branch ensure` y `done` (y al envoltorio `axiom-worktree`); retiran «lo crea en apply» (`CLAUDE.md:3`) y conservan la prohibición local de `EnterWorktree` | Pin de REQ-25.24 |
| `inc-23-knowledge-profile` | Figura como pendiente de archivar sin bloquear nada (REQ-25.24, D-17) | Sin exenciones para cambios preexistentes |
| inc-25 | Su `kickoff.yaml` no tiene instantánea, así que está exento (R15) | No se bloquea a sí mismo |
| Archivado de inc-25 | `sdd-archive` edita a mano las tablas no normativas de `planning-home-resolution` | Notas del delta |

### 2.22 D-22 — Literales `main` (R16)

- `kickoff.ParseGateRecordArgs` deja de fijar `BaseRef = "main"`, y `runSDDGateRecord` completa un `--base-ref` vacío con la integración resuelta del repositorio del planning home.
- `GetSpecsSyncStatus` (`service.go:1643`) usa la integración resuelta.
- `main` solo queda como valor por defecto de `branch.ResolvePolicy`.
- Una prueba de inventario sobre `internal/assets/skills/sdd-*` y `_shared` rechaza cualquier `main` o `master` con sentido de integración que no esté en una lista blanca.

### 2.23 D-23 — Ratchets, pins y goldens

- **Deadcode** (`scripts/deadcode-ratchet.sh`, alcanzabilidad desde `./cmd/axiom` con GOOS=linux). Las rebanadas son **verticales**: cada función nueva entra en la rebanada que la hace alcanzable desde `./cmd/axiom`. La tabla de §11 nombra el primer llamador de cada una. **No se actualiza la línea base.** Las variables de prueba (`gitcommon.Observe`, `gitcli.MergeTreeSupportedOverride` y `sddstatus.branchEngine`) son variables, no funciones. Los métodos que implementan los puertos se alcanzan por la llamada de interfaz desde el verbo de su rebanada. `VerdictFor` y `branch.Decide` entran en S8, con `kickoff seal` como primer llamador.
- **Refusals** (`internal/cli/refusal_resolution_ratchet_test.go`, que audita `cli`, `sddstatus` y `reviewtransaction`). Todo rechazo nuevo nombra su continuación o lleva `refusal:by-design`. La línea base no crece.
- **Pins:**
  - primitivas (D-19);
  - skills (REQ-25.23);
  - documentación (REQ-25.24, `docs_contract_test.go`);
  - `monorepo-decoupled` (REQ-25.25);
  - inventario de escritores (D-11);
  - literales `main` (D-22);
  - arquitectura de procesos (G-7).
- **Goldens.** Se regeneran con `-update` y nunca se resuelven a mano.

### 2.24 D-24 — Lo que no cambia

No cambian:
- el ciclo de fases;
- las compuertas, su ledger y la compuerta `integration`;
- REQ-21.16, salvo la rama de integración configurable y la retirada de `monorepo-decoupled`;
- el formato de handoff;
- RDD y sus linajes;
- el almacén Engram;
- la resolución de inc-24 fuera de los worktrees enlazados;
- `ResolveConfigFile`, `LoadConfig` (salvo los bloques nuevos) y `LocateAt`;
- el esquema `1.0` del hub.

No hay operaciones remotas nuevas por defecto.

### 2.25 D-25 — Rama de archivado y `ensure --archive` (REQ-25.45, D16, D18)

**Decisión.** `ensure --change <c> --archive`:
1. **Aplicabilidad.** Con `--lane odd` o en `multirepo`, `archive_branch_not_applicable`.
2. **Flags.** Solo admite `--prefix` (además de `--cwd`, `--json`, `--mode` y `--fetch`). `--branch` o `--base` dan `plan_conflict` (S-16).
3. **Nombre.**
   - Con `--prefix p`: `p/<c>-archive`, sin `plan_conflict` con el prefijo del cambio; el prefijo se registra.
   - Sin `--prefix`: `<rama del cambio>-archive`. La rama del cambio sale del registro, la instantánea o la política, en ese orden; por ejemplo, `inc/inc-25-x-archive`.
4. **Base.** La de integración de D-06, con `--fetch` opcional.
5. **Comprobación de integración.** Si el árbol de la base no contiene `<ruta relativa del planning home>/changes/<c>/` (`gitcli.TreeDirs`), `change_not_integrated` sin crear nada, nombrando la base evaluada y sugiriendo `--fetch`. Si la base contiene `changes/archive/<fecha>-<c>/`, el mismo código con un mensaje que nombra la entrada archivada (G-13).
6. **Modo.** El registrado del cambio (también la respuesta a `ask`). Si no hay registro, el modo personal. En `worktree`, `<raíz>/<c>-archive/<repo>`; en `in_place`, `git switch`, sujeto a REQ-25.9.
7. **Registro.** La rama se añade al registro del cambio con `kind: archive`. Sin publicar.
8. **Idempotencia.** Las ramas del cambio y de rol no se tocan. La repetición es idempotente.

**Integración con el resto del motor.**
- `status` incluye la rama de archivado con `prTarget` igual a la integración.
- `done` la confirma solo con `ancestry` o `equivalent_tree` (D-09).
- El veredicto la exige en el contexto de archivado (D-10).
- `RegisteredChanges` le da precedencia (D-16).
- La instantánea la nombra (D-12).

**Flujo D16 en monorepo:**

```
feat/x ──PR del cambio──▶ main (fusionado; x figura como pendiente de archivar)
                          │
   axiom branch ensure --change x --archive   (change_not_integrated → sugiere --fetch)
                          ▼
feat/x-archive: gate record --gate integration (pr_merged) → integrate prepare → archive-compose
                → mover a changes/archive/<fecha>-x → integrate verify
                          │
                          └──PR documental──▶ main ──▶ axiom branch done (feat/x y feat/x-archive)
```

## 3. Arquitectura y flujo de datos

### 3.1 `ensure`

```
CLI / dashboard / skill
   │  --change x [--lane] [--cwd] [--mode] [--prefix|--branch|--base] [--repo] [--fetch] [--archive]
   ▼
engine.Ensure
   ├─ workspace.Locate(cwd)                       (ficheros; worktree enlazado → checkout principal)
   ├─ gitcli.CommonDir(planning repo) → registry.Load(lane, x) [+ bloqueo]
   ├─ kickoff.Load → roles sellados, instantánea
   ├─ Prefs.WorktreePrefs(maestra del checkout principal)
   ├─ gitcli: TopLevel por ruta de rol, worktree list, ramas, HEAD, base, [status], [ls-tree en --archive]
   ▼
branch.Plan (puro) ─▶ Plan | *Refusal
   ▼
fase 3: comprobación sin efectos ─▶ fase 4: solo lo que falta ─▶ registry.Save ─▶ Result (estado final)
```

### 3.2 Veredicto y guardas

```
                 ┌── A: escritores Go (RequireChangeBranch, contexto por escritor) ── se niega; falla en cerrado
VerdictFor ──────┼── B: sdd status (archive en la fase archive) ── blocked(wrong_branch) → resolve-blockers
(solo ficheros)  └── D: guard pre + documento de Axiom (unión de contextos) ── bloquea; falla en abierto
      └── branch.Decide (puro) ◀── instantánea > registro > política; HEAD leído de ficheros; exenciones

guard post + otro fichero ── registro (ficheros) + HEAD (ficheros) ── aviso sin bloquear; falla en abierto
PendingArchive ── git ls-tree <integración>:<openspec>/changes ── informativo (status, dashboard)
```

### 3.3 Status desde el checkout principal (REQ-25.44)

```
sdd status x --cwd <principal>
  └─ Resolve ─ RegisteredChanges ─ worktree de archivado > worktree del cambio > principal ─▶ WorkspaceRoot
     └─ resolveByPreferenceOrder ─ resolveApplyState ─ branchContext(archive si la fase es archive)
          ├─ wrong_branch ─▶ motivo primero, ApplyBlocked, sin autoridad de edición, resolve-blockers
          └─ correcto ─────▶ applyEditAuthorityBlock(fuentes de tareas, raíces + registradas, rebase)
```

### 3.4 Ficheros de datos

| Fichero | Versionado | Escritor | Lectores |
|---|---|---|---|
| `axiom.yaml` (`git:`, `repositories:`) | Sí | La persona | `workspace`, `branch`, dashboard |
| `openspec/changes/<c>/kickoff.yaml` (`git:`) | Sí | `kickoff.Seal` (una vez) | `engine`, `sddstatus` |
| `<common dir>/axiom/branch/v1/<lane>/<c>.json` | No | `engine` (ensure, done) | `engine`, `sddstatus`, guarda D, dashboard, TUI |
| `~/.axiom/workspaces.json` | No | `hub` | `engine` |

## 4. Interfaces y contratos

### 4.1 `internal/branch` (dominio)

```go
package branch

type Lane string        // "sdd" | "odd"
type Mode string        // "worktree" | "in_place" | "ask"
type RepoRole string    // "change_branch" | "specs_in_place" | "code"
type BranchKind string  // "change" | "role" | "archive"
type Context string     // "ordinary" | "archive"
type Consultant string  // "write" | "status" | "apply" | "guard"
type VerdictKind string // "correct" | "wrong_branch" | "not_applicable"
type MergeLayer string  // "ancestry" | "equivalent_tree" | "pr_merged" | "forced"
type Code string

const (
	CodeDecisionRequired           Code = "decision_required"
	CodeDirtyWorktree              Code = "dirty_worktree"
	CodeWrongBranch                Code = "wrong_branch"
	CodeSpecsRepoNotOnIntegration  Code = "specs_repo_not_on_integration"
	CodeBaseNotFound               Code = "base_not_found"
	CodeFetchFailed                Code = "fetch_failed"
	CodePathTooLong                Code = "path_too_long"
	CodeInvalidName                Code = "invalid_name"
	CodeConfigInvalid              Code = "config_invalid"
	CodeGitFailed                  Code = "git_failed"
	CodePlanConflict               Code = "plan_conflict"
	CodeBranchCheckedOutElsewhere  Code = "branch_checked_out_elsewhere"
	CodeNotAGitRepository          Code = "not_a_git_repository"
	CodeMergeNotConfirmed          Code = "merge_not_confirmed"
	CodeChangeNotIntegrated        Code = "change_not_integrated"
	CodeArchiveBranchNotApplicable Code = "archive_branch_not_applicable"
)

// Label returns the Spanish label paired with every token (G-4).
func Label(token string) string

type Policy struct {
	IntegrationBranch string
	BranchPrefix      string
	PerRepository     map[string]RepoPolicy // key: slash-separated path relative to the master folder
}
type RepoPolicy struct{ IntegrationBranch, BranchPrefix string }

func ResolvePolicy(cfg *workspace.WorkspaceConfig) Policy
func (p Policy) For(repoRelPath string) RepoPolicy
func DefaultWorktreeRoot(master string) string

type RepoFacts struct {
	GitRoot      string // absolute, cleaned
	CommonDir    string // absolute
	RelPath      string // relative to master, slash-separated ("." for the master repo)
	DeclaredPath string // path as declared (repositories: or role), for <repo> naming (S-17)
	Roles        []string
}

type Exceptions struct{ Prefix, Branch, Base string }
type PathRules struct{ GOOS string } // Windows limit: 260 - 40 = 220 (A-1)

type PlanInput struct {
	Lane              Lane
	Change            string
	Archive           bool // ensure --archive (REQ-25.45)
	Topology          workspace.TopologyType
	Master            string
	WorkspaceRoot     string
	PlanningRepo      RepoFacts
	CodeRepos         []RepoFacts
	Skipped           []SkippedPath
	HasKickoff        bool
	SealedRoles       []string
	Snapshot          *kickoff.GitSnapshot
	Registered        *Record // materialized plan, if any
	Policy            Policy
	Exceptions        Exceptions
	RepoFilter        []string
	Mode              Mode
	WorktreeRoot      string
	ExistingWorktrees []WorktreeEntry
	PathRules         PathRules
}

type Plan struct {
	Lane          Lane
	Change        string
	Mode          Mode
	WorktreeRoot  string
	PlanningHome  string
	WorkspaceRoot string
	OddDocument   string
	Repositories  []RepoPlan
	Skipped       []SkippedPath
	Warnings      []string
}
type RepoPlan struct {
	Name, GitRoot, CommonDir, RelPath string
	Role                              RepoRole
	IntegrationBranch                 string
	BaseRef                           string
	Branches                          []BranchPlan
}
type BranchPlan struct {
	Name     string
	Kind     BranchKind
	RoleID   string // only for Kind == role
	From     string
	Worktree string // absolute; "" in in_place or specs_in_place
	PRTarget string
}

type Refusal struct {
	Code         Code
	Message      string // es
	Continuation string
	Details      map[string]string
}
type RefusalError struct{ Refusal } // Error() = Message + ". " + Continuation

func Plan(in PlanInput) (Plan, *Refusal)

type ExpectedNames struct {
	Change      string
	Roles       []string
	Archive     string
	Integration string
}
type VerdictInput struct {
	Role       RepoRole
	Change     string
	Lane       Lane
	Context    Context
	Consultant Consultant
	Expected   ExpectedNames
	Current    string
	Detached   bool
	Exempt     string // reason when exempt
}
type Verdict struct {
	Kind         VerdictKind
	Repo         string
	Current      string
	Detached     bool
	Expected     []string
	Detail       string
	Warning      string
	Continuation string
}

func Decide(in VerdictInput) Verdict

type MergeEvidence struct {
	IsAncestor         bool
	TreeEquivalent     bool
	TreeLayerSupported bool
	PRMergedVerified   bool // never considered for Kind == archive
}

func ConfirmMerge(kind BranchKind, e MergeEvidence, force bool) (MergeLayer, bool)

// Ports.
type Git interface {
	TopLevel(ctx context.Context, dir string) (string, error)
	CommonDir(ctx context.Context, dir string) (string, error)
	ResolveCommit(ctx context.Context, dir, ref string) (sha string, ok bool, err error)
	RefKind(ctx context.Context, dir, ref string) (string, error) // "remote-tracking" | "local"
	BranchExists(ctx context.Context, dir, name string) (bool, error)
	Worktrees(ctx context.Context, dir string) ([]WorktreeEntry, error)
	DirtyPaths(ctx context.Context, dir string) ([]string, error)
	CreateBranch(ctx context.Context, dir, name, startPoint string) error
	AddWorktree(ctx context.Context, dir, path, branch string) error
	Switch(ctx context.Context, dir, branch string) error
	RemoveWorktree(ctx context.Context, dir, path string, force bool) error
	DeleteBranch(ctx context.Context, dir, name string) error
	IsAncestor(ctx context.Context, dir, ancestor, descendant string) (bool, error)
	MergeTreeEquivalent(ctx context.Context, dir, target, branch string) (equivalent, supported bool, err error)
	AheadBehind(ctx context.Context, dir, base, branch string) (ahead, behind int, err error)
	TreeDirs(ctx context.Context, dir, rev, relPath string) ([]string, bool, error) // dirs, exists, err
	Fetch(ctx context.Context, dir, remote string) error // the only remote operation
}
type Registry interface {
	Lock(ctx context.Context, commonDir string, lane Lane, change string) (unlock func(), err error)
	Load(commonDir string, lane Lane, change string) (*Record, error) // nil, nil when absent
	Save(commonDir string, rec Record) error                          // no-op when bytes are unchanged
	Delete(commonDir string, lane Lane, change string) error
	List(commonDir string) ([]Record, error)
}
type Prefs interface {
	WorktreePrefs(projectMaster string) (mode Mode, root string, source string, warning string)
}
type LineageProbe interface {
	OpenLineages(ctx context.Context, worktreeRoot string) ([]string, error)
}
```

### 4.2 `internal/branch/engine`

```go
package engine

type Engine struct { /* git branch.Git; reg branch.Registry; prefs branch.Prefs; lineages branch.LineageProbe; goos string */ }

func New(git branch.Git, reg branch.Registry, prefs branch.Prefs, lineages branch.LineageProbe) *Engine
func Default(hubManager *hub.Manager, lineages branch.LineageProbe) *Engine

type EnsureRequest struct {
	CWD, Change string
	Lane        branch.Lane
	Mode        branch.Mode // "" = no --mode
	Exceptions  branch.Exceptions
	Repos       []string
	Fetch       bool
	Archive     bool
}
func (e *Engine) Ensure(ctx context.Context, r EnsureRequest) branch.Result
func (e *Engine) Status(ctx context.Context, cwd, change string, lane branch.Lane) branch.Result // change optional
func (e *Engine) Done(ctx context.Context, cwd, change string, lane branch.Lane, force, fetch bool) branch.Result
func (e *Engine) Guard(ctx context.Context, payload GuardPayload) GuardDecision // never errors, never spawns

type VerdictQuery struct {
	Path       string // change root, file path or repository path
	Change     string
	Lane       branch.Lane
	Context    branch.Context
	Consultant branch.Consultant
	SealRoles  []string // roles about to be sealed (seal only)
}
func (e *Engine) VerdictFor(ctx context.Context, q VerdictQuery) (branch.Verdict, error) // file reads only

type WriteTarget struct {
	ChangeRoot, Change string
	Lane               branch.Lane
	Context            branch.Context
	Op                 string
	SealRoles          []string
}
func (e *Engine) RequireChangeBranch(ctx context.Context, t WriteTarget) (warning string, err error)

func (e *Engine) Snapshot(ctx context.Context, loc workspace.Location, change string, roles []string) (*kickoff.GitSnapshot, error)

type RegisteredChange struct{ Name, Branch, Kind, WorktreeRoot, PlanningHome, ChangeRoot string }
func (e *Engine) RegisteredChanges(ctx context.Context, loc workspace.Location) ([]RegisteredChange, []string)

type PendingChange struct{ Name, Continuation string }
func (e *Engine) PendingArchive(ctx context.Context, loc workspace.Location) ([]PendingChange, []string)

type GuardDecision struct {
	Action  string // "none" | "block" | "warn"
	Message string
}
```

### 4.3 JSON `axiom.branch-result/v1` (ejemplo: `ensure`, monorepo, dos roles)

```json
{
  "schema": "axiom.branch-result/v1",
  "operation": "ensure",
  "status": "ready",
  "statusLabel": "listo",
  "change": "x",
  "lane": "sdd",
  "topology": "monorepo-embedded",
  "mode": "worktree",
  "modeSource": "hub",
  "worktreeRoot": "C:\\repos\\axiom-wt",
  "workspaceRoot": "C:\\repos\\axiom-wt\\x\\axiom",
  "planningHome": "C:\\repos\\axiom-wt\\x\\axiom\\openspec",
  "repositories": [
    {
      "name": "axiom",
      "gitRoot": "C:\\repos\\axiom",
      "role": "change_branch",
      "roles": ["core", "qa"],
      "integrationBranch": "main",
      "base": {"ref": "origin/main", "sha": "3ebb9f3f…", "source": "remote-tracking", "fetched": false},
      "branches": [
        {"name": "feat/x", "kind": "change", "worktree": "C:\\repos\\axiom-wt\\x\\axiom", "prTarget": "main"},
        {"name": "feat/x-core", "kind": "role", "role": "core", "worktree": "C:\\repos\\axiom-wt\\x-core\\axiom", "prTarget": "feat/x"},
        {"name": "feat/x-qa", "kind": "role", "role": "qa", "worktree": "C:\\repos\\axiom-wt\\x-qa\\axiom", "prTarget": "feat/x"}
      ]
    }
  ],
  "skipped": [],
  "warnings": ["Publica la rama de seguimiento feat/x en el remoto antes del primer PR de rol; axiom branch nunca ejecuta push."]
}
```

Variantes:
- **Rechazo:** `"status": "refused", "statusLabel": "rechazado", "code": "change_not_integrated", "message": "…", "continuation": "axiom branch ensure --cwd \"…\" --change x --archive --fetch"`.
- **`decision_required`:** `"decision": {"question": "…", "choices": [{"answer": "worktree", "label": "…", "effect": "…", "invocation": "…"}, {"answer": "in_place", …}]}`.
- **`status`:** por rama, `currentBranch`, `dirty`, `ahead`, `behind` y `prTarget`; además `registry.present`. Sin `--change`, `registered[]` y `pendingArchive[]`; con `--change`, `pendingArchive: true|false`. Sin ramas, `"status": "not_started", "statusLabel": "no iniciado"`.
- **`done`:** por rama, `outcome`, `confirmedBy`/`confirmedByLabel`, `discarded[]` y `tipSha`.

### 4.4 Registro local `axiom.branch-registry/v1`

```json
{
  "schema": "axiom.branch-registry/v1",
  "lane": "sdd",
  "change": "x",
  "mode": "worktree",
  "modeSource": "answer",
  "exceptions": {"prefix": "", "branch": "inc/x", "base": ""},
  "archivePrefix": "",
  "planningRepo": "C:\\repos\\axiom",
  "repositories": [
    {"name": "axiom", "gitRoot": "C:\\repos\\axiom", "commonDir": "C:\\repos\\axiom\\.git", "role": "change_branch",
     "branches": [
       {"name": "inc/x", "kind": "change", "worktree": "C:\\repos\\axiom-wt\\x\\axiom"},
       {"name": "inc/x-archive", "kind": "archive", "worktree": "C:\\repos\\axiom-wt\\x-archive\\axiom"}
     ]}
  ]
}
```

### 4.5 Instantánea en `kickoff.yaml`

```go
package kickoff

type GitSnapshot struct {
	IntegrationBranch string            `yaml:"integration_branch"`
	BranchPrefix      string            `yaml:"branch_prefix"`
	Exceptions        *GitExceptions    `yaml:"exceptions,omitempty"`
	ChangeBranch      string            `yaml:"change_branch"`
	Repositories      []GitSnapshotRepo `yaml:"repositories"`
}
type GitExceptions struct {
	Prefix string `yaml:"prefix,omitempty"`
	Branch string `yaml:"branch,omitempty"`
	Base   string `yaml:"base,omitempty"`
}
type GitSnapshotRepo struct {
	Path              string              `yaml:"path"` // relative to the master folder, slash-separated
	Name              string              `yaml:"name"`
	Role              string              `yaml:"role"` // change_branch | specs_in_place | code
	IntegrationBranch string              `yaml:"integration_branch"`
	Base              string              `yaml:"base"` // branch name, never a SHA
	Branches          []GitSnapshotBranch `yaml:"branches,omitempty"`
}
type GitSnapshotBranch struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"` // change | role | archive
	Role string `yaml:"role,omitempty"`
}
```

```yaml
git:
  integration_branch: main
  branch_prefix: feat
  change_branch: feat/x
  repositories:
    - path: "."
      name: axiom
      role: change_branch
      integration_branch: main
      base: main
      branches:
        - {name: feat/x, kind: change}
        - {name: feat/x-core, kind: role, role: core}
        - {name: feat/x-qa, kind: role, role: qa}
        - {name: feat/x-archive, kind: archive}
```

### 4.6 Hojas, `workspace` y `hub`

```go
package gitref // pure leaf
func ValidRefName(name string) error
func ValidBranchComponent(name string) error // git component + Windows directory name
func ValidPrefix(prefix string) error

package gitcommon // leaf: the only git exec point plus file-only readers
var Observe func(dir string, args []string) // nil in production (G-7)
func Run(ctx context.Context, dir string, args ...string) (Result, error) // the only exec.Command
func CommonDir(ctx context.Context, dir string) (string, error) // git rev-parse --git-common-dir (REQ-25.14)
func TopLevel(ctx context.Context, dir string) (string, error)  // git rev-parse --show-toplevel (REQ-25.6)
func GitRootFromFiles(path string) (string, bool)
func GitDirFromFiles(gitRoot string) (string, error)
func CommonDirFromFiles(gitRoot string) (string, error)
func LinkedWorktreeRoot(path string) string
func MainWorktreeFromFiles(commonDir string) (string, bool)
func ReadHead(gitRoot string) (branch string, detached bool, err error) // no process (REQ-25.21)

package workspace
type GitConfig struct{ IntegrationBranch, BranchPrefix string }          // yaml: integration_branch, branch_prefix
type RepositoryPolicy struct{ Path, IntegrationBranch, BranchPrefix string } // yaml: path, integration_branch, branch_prefix
// WorkspaceConfig: Git *GitConfig `yaml:"-"`; Repositories []RepositoryPolicy `yaml:"-"`
// Declarations: Git, GitErr, Repositories, RepositoriesErr
func ValidateRepositoryPolicies(cfg *WorkspaceConfig, master string) (errs, warnings []string)

package hub
type WorktreePrefs struct{ Mode, Root string }
func ValidateWorktreePrefs(p WorktreePrefs) error
func (m *Manager) SetWorktreePrefs(idOrPath string, p WorktreePrefs) (*WorkspaceRecord, error)
func (m *Manager) LookupWorktreePrefs(projectPath string) (WorktreePrefs, bool, error)
```

### 4.7 Costuras en `sddstatus`

```go
// branch_context.go (new)
var branchEngine func() *engine.Engine // nil in production → engine.Default(nil, nil)

type tasksSource struct {
	text string
	base string // directory the task paths resolve against
}
type branchQuery struct {
	workspaceRoot, changeRoot, change string
	archivePhase, apply                bool
	tasksSources                       []tasksSource
}
type branchContext struct {
	wrongBranch     *branch.Verdict
	registeredRoots []string
	rebase          func(string) string
	tasksSources    []tasksSource
	notes           []string
}
func resolveBranchContext(q branchQuery) branchContext

// edit_authority.go (signature changes; identity rebase and a single tasks.md source keep today's behaviour)
func detectUnauthorizedEditRoots(tasksText, base string, allowedEditRoots []string, rebase func(string) string) []string
func detectUnauthorizedEditRootsForCurrentWorkUnit(tasksText, base string, allowedEditRoots []string, rebase func(string) string) (blocking, future []string)
func applyEditAuthorityBlock(applyState ApplyState, reasons *blockerReasons, sources []tasksSource, allowedEditRoots []string, rebase func(string) string) (ApplyState, []string)
```

## 5. Mapa de ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/gitref/gitref.go` | Crear | Validación pura de nombres (B1) |
| `internal/gitcommon/{exec,files}.go` | Crear | Único `exec` de git con observador; lecturas de ficheros (`HEAD`, common dir, worktrees) |
| `internal/branch/{types,labels,policy,naming,planner,verdict,merge,result,continuation,ports}.go` | Crear | Dominio puro y puertos |
| `internal/branch/gitcli/{ops,capability}.go` | Crear | Operaciones tipadas sobre `gitcommon.Run`; capacidad de `merge-tree` |
| `internal/branch/registry/registry.go` | Crear | Registro local |
| `internal/branch/engine/{engine,ensure,archive,status,pending,done,guard,verdict,listing,snapshot,prefs}.go` | Crear | Servicio de aplicación |
| `internal/cli/branch.go`, `branch_guard.go`, `branch_lineage.go` | Crear | CLI, cargas de hook y adaptador RDD |
| `cmd/axiom/main.go` | Modificar | `case "branch"`, ayuda, flags de preferencias en `init` y `project add`, guarda A en `runHandoffCreate`, `runWorkspaceValidate` desde worktrees |
| `internal/workspace/{types,loader,validator,locate}.go` y `gitpolicy.go` (nuevo) | Modificar y crear | Retirada de decoupled; `git:` y `repositories:`; ascensión acotada |
| `internal/skillregistry/registry.go` | Modificar | Comentario de la topología retirada |
| `internal/hub/{types,manager,detector,init}.go` y `prefs.go` (nuevo) | Modificar y crear | Preferencias e `IsGitWorktree` por evidencia |
| `internal/kickoff/{types,schema,args}.go` y `gitsnapshot.go` (nuevo) | Modificar y crear | Instantánea; `BaseRef` sin literal |
| `internal/cli/{sdd_kickoff,sdd_gate,sdd_integrate,sdd_archive_compose,sdd_status}.go` | Modificar | Guarda A con su contexto; instantánea; integración resuelta |
| `internal/sddstatus/{status,edit_authority}.go`; `branch_context.go` y `worktree_changes.go` (nuevos) | Modificar y crear | Guarda B, S-9, contexto de archivado, raíces, reubicación, cambios en worktrees |
| `internal/dashboard/{service,server}.go`, `assets/{index.html,app.js}` | Modificar | `ensure` y `writeIncrement` (nuevo); preferencias; política; listados; `X-Axiom-Notes`; literal `main` |
| `internal/tui/model.go` | Modificar | Listado con los cambios registrados |
| `internal/components/sdd/inject.go`, `opencode_plugin_digests.go` | Modificar | Hooks `Pre` y `Post` en Claude y Codex; plugin de OpenCode |
| `internal/assets/opencode/plugins/axiom-branch-guard.ts` | Crear | Plugin `before` (bloqueo) y `after` (aviso) |
| `internal/components/agentguidance/routing.go` | Modificar | Paso 6 y «Branches and worktrees» |
| `internal/assets/skills/sdd-{apply,spec,design,propose,explore,archive}/SKILL.md`, `_shared/{sdd-phase-common,sdd-status-contract,sdd-orchestrator-sections}.md` | Modificar | D-19 |
| `internal/assets/assets_test.go` | Modificar | Pins |
| `axiom.yaml`, `axiom.example.yaml`, `docs/manual-de-inicio.md`, `docs/architecture/AXIOM_ENTERPRISE_VISION.md` | Modificar | `git:`; retirada de decoupled; `axiom branch` |
| `scripts/axiom-worktree.ps1`, `scripts/axiom-worktree.sh` | Modificar | Envoltorio fino (D-21) |
| `scripts/cleanup-worktree.ps1`, `scripts/cleanup-worktree.sh` | Eliminar | Sustituidos por `axiom branch done` |
| `scripts/claude-hooks/guard-primary-checkout.ps1` | Modificar | Mensajes que remiten a `axiom branch ensure` |
| `CLAUDE.md`, `AGENTS.md`, `GEMINI.md`, `.codex/AGENTS.md` | Modificar | Dogfooding y regeneración |
| `testdata/golden/**` | Regenerar | Doctrina, orquestadores y skills |
| `openspec/specs/…` | Modificar al archivar | Deltas y spec nueva; tablas no normativas a mano |

## 6. Conformidad con la spec

### 6.1 `change-branch-lifecycle`

| Requisito | Diseño |
|---|---|
| REQ-25.1 Familia de cuatro verbos, sin remoto, único punto de git | D-02; D-01 y §1 (G-7) |
| REQ-25.2 Contrato, tokens, `config_invalid`, `git_failed` | D-02, §4.3 |
| REQ-25.3 Política e `invalid_name` frente a `config_invalid` | D-03, D-04 |
| REQ-25.4 Modelo por roles, PR del cambio sin archive, rol `archive` (S-15) | D-04, D-25 |
| REQ-25.5 Papel del repositorio | D-04, D-05 |
| REQ-25.6 Raíz git | D-04, D-05 |
| REQ-25.7 Idempotencia, también con `--archive` | D-05, D-25 |
| REQ-25.8 Modos (respuesta registrada válida para `--archive`) | D-05, D-08 |
| REQ-25.9 `in_place` sucio | D-05 |
| REQ-25.10 Base sin red | D-06 |
| REQ-25.11 Rutas, `-archive`, 220 en Windows, S-17 | D-04 |
| REQ-25.12 Excepciones; con `--archive` solo `--prefix` (S-16) | D-04, D-25 |
| **REQ-25.45 Rama de archivado y `ensure --archive`** | **D-25**, D-06, D-07 |
| REQ-25.13 ODD | D-04, D-18 |
| REQ-25.14 Registro local | D-07 |
| REQ-25.15 `status` (`--change` opcional, archivado, pendientes) | D-02, D-17, §4.3 |
| REQ-25.16 Capas de fusión; archivado solo con 1 y 2 | D-09 |
| REQ-25.17 Desmontaje | D-09 |
| REQ-25.18 Veredicto con contextos | D-10 (G-10, G-11) |
| REQ-25.19 Guarda A | D-11 (G-10) |
| **REQ-25.20 Cambios pendientes de archivar** | **D-17**, D-20 |
| REQ-25.21 `guard` bloquea y avisa sin procesos | D-18, D-10 |
| REQ-25.22 Instalación por `sync` | D-18 |
| REQ-25.23 Skills y doctrina | D-19, D-22 |
| REQ-25.24 Dogfooding y Q6 | D-21 (G-12) |

### 6.2 Deltas

| Capacidad y requisitos | Diseño |
|---|---|
| `workspace-topology` REQ-1.2, REQ-2.1, REQ-2.3 (REMOVED), REQ-25.25 a 25.27 | D-03 |
| `planning-home-resolution` REQ-24.3 (MODIFIED y RENAMED), REQ-24.5, REQ-24.7 | D-03 |
| `planning-home-resolution` REQ-24.9 | D-14 |
| `planning-home-resolution` REQ-25.28 | D-15 |
| `multi-project-hub` REQ-3.1, REQ-3.2, REQ-25.29 a 25.31 | D-08, D-20 |
| `local-web-dashboard` REQ-1.1, REQ-25.32 a 25.34 | D-20, D-16, D-17, D-11 |
| `sdd-lane-kickoff` REQ-25.35 | D-19 |
| `sdd-lane-kickoff` REQ-25.36 a 25.38 | D-12 |
| `sdd-archive-lifecycle` REQ-21.16 (forma original) y REQ-25.39 | D-24 (REQ-21.16 sin excepción), D-25, D-13, D-19 |
| `organic-agent-trigger-rules` REQ-25.40, REQ-25.41 | D-19 |
| `axiom-sdd-cli-integration` REQ-25.42 | D-13, D-10 |
| `axiom-sdd-cli-integration` REQ-25.43 | D-14 |
| `axiom-sdd-cli-integration` REQ-25.44 | D-16 |
| `spec-baseline-declaration` REQ-24.13 | D-03 |

**Requisitos no satisfechos:** ninguno. G-10 a G-13 son lecturas que se someten a la compuerta, y A-4 y A-5 son supuestos que se verifican al implementar.

**Tecnologías y patrones.** Go 1.25 con la biblioteca estándar y `yaml.v3`, sin dependencias nuevas; TypeScript solo en el plugin de OpenCode. Se siguen los patrones del repositorio:
- carga por capas;
- git aislado (`gitview.go`);
- publicación y bloqueo de `reviewtransaction`;
- hooks de `sync` con `filemerge.WriteFileAtomic`;
- variables de prueba en lugar de *setters*;
- pins en `assets_test.go`;
- goldens con `-update`;
- `pathquote.Quote` en los mensajes.

## 7. Estrategia de pruebas

### 7.1 Por capa

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unidad (`gitref`) | `ValidRefName`, `ValidBranchComponent` (`"a b:c"`, `"mi prefijo//"`, nombres reservados de Windows, `.lock`, `..`) y `ValidPrefix` | Tablas |
| Unidad (`gitcommon`, ficheros) | `ReadHead` (rama, desacoplado, ilegible y reftable), `CommonDirFromFiles` igual a `git rev-parse --git-common-dir` (clon, worktree enlazado y `--separate-git-dir`), `LinkedWorktreeRoot` frente a un submódulo y `MainWorktreeFromFiles` | Repositorios git reales en `t.TempDir()` |
| Unidad (`branch`) | `ResolvePolicy` (REQ-25.3); `Plan`: embebido con 1 y 2 roles, antes y después del sellado, multirepo (specs in situ, código, repositorio compartido, rol desconocido), ODD, `fullstack`, subcarpeta, `.`, deduplicación, excepciones, `--archive` (nombre derivado, `--prefix`, `--branch`/`--base` → `plan_conflict`, multirepo y ODD → `archive_branch_not_applicable`), rol `archive` → `invalid_name`, `<repo>` repetido, anidamiento, 220 y 221 caracteres con `GOOS: "windows"`; `Decide` en los dos contextos y con los cuatro consultantes (todos los escenarios de REQ-25.18); `ConfirmMerge` (la rama de archivado ignora `pr_merged`); `Label` para todos los tokens | Tablas sin git ni disco |
| Unidad (`workspace`, `kickoff`, `hub`) | Bloques `git:` y `repositories:` (claves, valores, `config_invalid`); mensaje de la topología retirada; instantánea con la rama de archivado (ida y vuelta, ausencia, binario anterior); preferencias | Tablas sobre `t.TempDir()` |
| Integración (`engine` con git real) | `ensure` en sus modos y rechazos; idempotencia byte a byte (JSON, `git worktree list`, `git branch --list`); registro borrado y reconstruido; `branch_checked_out_elsewhere`; base `origin/main` frente a `main` con un remoto desnudo; `--fetch` contra un remoto inalcanzable; `ensure --archive` (fusionado, sin fusionar → `change_not_integrated` sin efectos, ya archivado, `in_place` limpio y sucio, respuesta a `ask` reutilizada); `done` (fusión real, squash con `main` avanzado, capa 2 omitida, `pr_merged`, `-archive` con squash y sin fusionar, sucio, `--force`, `in_place`, `partial`, aviso RDD con sonda doble); `status` con y sin `--change` y pendientes (sumideros, `.gitkeep`, multirepo, sin referencia de integración → aviso) | `git init -b main` en `t.TempDir()`, entorno aislado (`HOME` temporal, `GIT_CONFIG_GLOBAL` vacío, `GIT_CONFIG_NOSYSTEM=1`), patrón de `review_atomic_start_integration_test.go`; sin red |
| Arquitectura (G-7) | Ningún fichero de `internal/branch/**` ni de `internal/gitcommon` importa `os/exec`, `net` ni `net/http`, salvo `gitcommon/exec.go`; el observador no registra `fetch`, `pull`, `push` ni `ls-remote` en `ensure`, `status` y `done` sin `--fetch`; `VerdictFor` y `guard` no registran ninguna invocación | `go/parser` sobre las fuentes y `gitcommon.Observe` |
| Integración (`sddstatus`) | `blocked(wrong_branch)` en spec y en archive (en monorepo, el archivado exige `-archive`; `feat/x`, `main` y `feat/x-core` bloquean); el repositorio de specs de un multirepo exige la integración también en archive; el repositorio de código solo en apply; exentos; sin git; S-9 y después `edit_authority_missing`; `continue` sin fase autorizada; raíces registradas y reubicación en los dos detectores; `tasks.core.md` contra `x-core`; sin registro, el comportamiento anterior; REQ-25.44 (precedencia del worktree de archivado y worktree desaparecido); `archiveDependencyFromGovernance` sin cambios; el contrato v2 solo crece | `Resolve` sobre fixtures con worktrees reales |
| Integración (CLI) | Ayuda con cuatro verbos; `push`, `pr` y `check` como no reconocidos; códigos de salida; `--cwd`; `seal` (fuera de rama, en rama de rol, multirepo, `--infer`); una prueba de negativa por escritor con su contexto (incluidos `integrate` y `gate record --gate integration` en archivado); inventario de escritores; `workspace validate` desde un worktree externo y con decoupled; `gate record` sin `--base-ref` | `RunBranch`, `RunSDDKickoff`, `RunSDDGate` y funciones `run*` |
| HTTP | Creación con `ensure` y `writeIncrement` (worktree, `ask` → 409, repetición, rechazo sin carpeta, multirepo); `writeIncrement` invocado sin `ensure` en `main` → `wrong_branch`; preferencias; `/api/workspace` (política y 405); listados con precedencia, `pendingArchive` y `X-Axiom-Notes`; handoff dirigido al worktree | `httptest` y un hub temporal |
| TUI | Lista el mismo conjunto y no escribe | `Model.Update` directo |
| `guard` | Bloqueo `pre` en documento de Axiom en rama equivocada, también sin cambio registrado (S-18); permitido en `feat/x` y en `feat/x-archive` (unión de contextos, G-11); `odd/tasks`; aviso `post` con uno y con varios cambios registrados, en `in_place`, sin aviso en el worktree correcto, sin cambio registrado, en el repositorio de specs de un multirepo; inerte fuera de Axiom; fallo en abierto (configuración o registro ilegibles, `HEAD` ilegible, tiempo agotado, pánico); cero procesos; cargas de los tres runtimes; protocolo JSON de OpenCode | Tablas sobre cargas y fixtures, con el observador de procesos |
| `sync` | Entradas `Pre` y `Post` en Claude, entradas de Codex y plugin de OpenCode; segunda ejecución byte a byte idéntica; entradas ajenas intactas; retirada; Pi sin entrada; ningún hook de git | `HOME` temporales, como las pruebas de `inject.go` |
| Assets y goldens | Pins de D-19, D-22 y REQ-25.24; goldens | `assets_test.go`, `docs_contract_test.go` y `-update` |

### 7.2 Windows y portabilidad

- **Rutas de git.** Las rutas `C:/…` se normalizan y se comparan por identidad, porque los temporales usan nombres cortos 8.3.
- **Regla de 220 caracteres.** Se prueba con `PathRules`. La prueba real solo corre en Windows.
- **CRLF.** El registro y la instantánea se escriben con `\n`; `ReadHead` tolera CRLF y espacios finales.
- **Borrado de un worktree que es el directorio actual.** Se prueba en Windows: la rama se conserva.
- **Higiene.** Los `TestMain` fijan `workspace.DiscoveryCeilingOverride` y ninguna prueba crea worktrees del repositorio real.

## 8. Matriz de amenazas

El diseño añade subprocesos git que escriben, un hook de proceso en tres runtimes y la automatización de ramas, así que la matriz aplica.

| Frontera | Casos adversarios mínimos | Aplicabilidad | Respuesta de diseño | Pruebas RED previstas |
|---|---|---|---|---|
| Rutas con aspecto de documentación | `requirements.txt`, `CMakeLists.txt`, Markdown o MDX ejecutable, `README.sh` | Aplicable, solo en clasificación: la guarda D clasifica por segmentos y nunca ejecuta ni interpreta el contenido | `odd/tasks/` exige `.md`; cualquier fichero bajo `changes/<x>/` es un documento de Axiom; el resto es «otro fichero» (aviso) | `odd/tasks/x.md` se bloquea; `odd/tasks/README.sh` solo puede avisar; `openspec/changes/x/run.sh` se bloquea; `openspec/changes/x/../../internal/a.go` se limpia y es otro fichero; mayúsculas distintas en Windows |
| Selección de repositorio git | `git -C`, rutas relativas, rutas absolutas | Aplicable: `ensure`, `status` y `done` actúan sobre varios repositorios | `cmd.Dir` siempre es la raíz absoluta de `show-toplevel`; se retiran `GIT_DIR`, `GIT_WORK_TREE`, `GIT_COMMON_DIR` e `GIT_INDEX_FILE`; las creaciones y los borrados se hacen desde el checkout principal; el veredicto y la guarda localizan repositorio y `HEAD` por ficheros a partir de la ruta destino, nunca por el directorio del proceso | Con `GIT_DIR` hacia otro repositorio, la selección no cambia; `--cwd` desde un directorio ajeno; ruta de rol con `..`; ruta no git → `skipped`; worktree de un multirepo fuera de la maestra |
| Estado del commit | staged, `commit -a`, índice vacío | Aplicable: `in_place` y `done` dependen del árbol limpio; Axiom nunca hace `add` ni `commit` | `git status --porcelain=v1 -z --untracked-files=normal` con `GIT_OPTIONAL_LOCKS=0` | Staged, sin stage y sin seguimiento → `dirty_worktree`; ignorado → limpio; repositorio sin commits → `base_not_found` |
| Estado del push | Rama con upstream, primer push, refspec explícito | Aplicable en negativo: nunca hay `push`; la persona publica la rama de seguimiento y la de archivado | El puerto no tiene ningún método de push; `--fetch` es la única operación remota | Observador sin verbos remotos en los tres verbos sin `--fetch`; con `--fetch`, un único `fetch origin` por repositorio |
| Comandos de PR | `--head` explícito, prefijo de entorno, comandos compuestos | N/A en el producto: `axiom branch` no crea PR (D3). El envoltorio local `axiom-worktree pr` no es producto y conserva el comportamiento actual, con la base tomada de `prTarget` | — | — |
| Integración de proceso (hook) | Carga mal formada o enorme, `..` o enlaces simbólicos, inyección en la orden, tiempo agotado | Aplicable | Órdenes literales sin interpolar datos de la carga; carga de 256 KiB como máximo; rutas limpiadas y confirmadas por `Locate`; ningún proceso lanzado; 1 s de presupuesto; fallo en abierto; `spawn` con array de argumentos en OpenCode | 1 MiB → nada; JSON inválido → nada; `..` que sale del cambio → otro fichero; tiempo agotado → nada; orden instalada literal; observador de procesos vacío |
| Borrado de ramas y worktrees | Rama no fusionada, árbol sucio, worktree ajeno, rama remota | Aplicable | Capas de confirmación (la de archivado sin `pr_merged`); `--force` explícito con informe; solo ramas del plan; nunca ramas remotas; sin worktree borrado, la rama se conserva | Escenarios de REQ-25.16 y 25.17, incluida la rama de archivado |

## 9. Migración y despliegue

- **`monorepo-decoupled`.** Única rotura deliberada, con un mensaje que nombra `multirepo`.
- **Datos.** Sin migración; todas las claves nuevas son opcionales y el registro se reconstruye desde git.
- **Cambios preexistentes.** Sin instantánea, exentos. Un cambio fusionado y sin archivar aparece como pendiente sin bloquear nada (por ejemplo, inc-23).
- **Adopción.**
  1. Instalar la versión con `axiom branch`.
  2. Ejecutar `axiom sync` para el hook D.
  3. Opcionalmente, declarar `git:` y `repositories:` y ejecutar `axiom workspace validate`.
  4. En monorepo, adoptar el flujo D16: PR del cambio, `ensure --archive` y PR documental.
- **Equipo mixto.** Un binario anterior no protege y descarta las preferencias del hub al reescribir `workspaces.json` (R24).
- **Reversión.** Las rebanadas se revierten con `git revert` en orden inverso. El hook D se retira con `axiom sync`. Ramas y worktrees son objetos git normales. El registro local se puede borrar.

## 10. Riesgos y mitigaciones

| ID | Riesgo | Prob. | Mitigación |
|---|---|---|---|
| R4 | Resolución errónea desde worktrees externos | Media | D-15 y `--cwd` desde el JSON |
| R5 | `edit_authority_missing` falso en worktrees | Media | D-14 |
| R6 | Listados vacíos | Media | D-16 |
| R9 | Divergencia con upstream | Media | Ficheros nuevos; costuras pequeñas en `status.go` y `edit_authority.go` |
| R10 | Capa D de mejor esfuerzo: `throw` de OpenCode tragado en algunos carriles; Codex quizá sin canal de aviso (A-5); sin Bash | Media | Las garantías son A y B; D es complementaria |
| R14 | `MAX_PATH` en Windows | Media | 220 fijo (A-1) |
| R15 | Bloqueo de cambios preexistentes | Baja | Exención por ausencia de instantánea; `--infer` |
| R16 | Literales `main` residuales | Media | D-22 |
| R18 | Base atrasada sin fetch, incluido `change_not_integrated` tras una fusión remota | Media | Mensaje que sugiere `--fetch`; `base.source` y `sha` |
| R19 | Guarda A olvidada en un escritor futuro | Media | Prueba de inventario |
| R20 | Codex: formato de la carga y del canal distinto de lo supuesto (A-2, A-5) | Media | Verificación en S11; fallo en abierto |
| R21 | Windows no borra un worktree que es el directorio actual | Media | La rama se conserva y se pide `--cwd` |
| R22 | Versión de git de CI desconocida (Q1) | Baja | Capa omitible y pruebas con override |
| R23 | Worktrees antiguos de los scripts en `C:\repos\axiom-wt` | Baja | `plan_conflict` o `branch_checked_out_elsewhere` antes de crear nada |
| R24 | Un binario anterior descarta las preferencias del hub | Baja | Documentado; el defecto (`worktree`) es seguro |
| R26 | `in_place` con ramas de rol solo despliega la de seguimiento | Baja | Limitación documentada |
| R27 | Dos adaptadores git parecidos (`gitview.go` y `gitcommon`) | Baja | Seguimiento para unificarlos |
| R28 | Entrega muy por encima de 400 líneas | Alta | §11 y la decisión en `sdd-tasks` |
| R29 | `HEAD` con el backend reftable (`refs/heads/.invalid`) no se puede leer por ficheros | Baja | Veredicto `not_applicable` con aviso; la guarda D no hace nada; A y B no bloquean (fallo en abierto documentado) |
| R30 | Los pendientes de archivar crecen si se olvida el PR documental | Media | Visibles en `status`, en el dashboard y en `X-Axiom-Notes`, cada uno con su continuación |
| R31 | Las lecturas G-10 y G-11 amplían el contexto de archivado más allá del texto de REQ-25.18 | Media | Se someten a la compuerta; sin ellas, el flujo D16 se bloquea a sí mismo |
| R32 | El envoltorio `axiom-worktree` exige un binario con `axiom branch` | Baja | Mensaje claro y `go install ./cmd/axiom` en este repositorio |

## 11. Unidades de entrega y estimación

Rebanadas **verticales** (B2): cada una deja el árbol verde (`go test ./...`, `go vet ./...` y ratchets sin empeorar) y todas sus funciones nuevas son alcanzables desde `./cmd/axiom` por el primer llamador indicado. No hay que actualizar la línea base de deadcode. Estimaciones sin medir, en líneas de autoría (código, pruebas, assets y documentación; sin goldens ni `.codex/AGENTS.md`).

| Rebanada | Alcance | Primer llamador desde `./cmd/axiom` | Estimación | Depende de |
|---|---|---|---|---|
| S1 | D-03: `gitref`, retirada de decoupled, `git:` y `repositories:`, `workspace validate`, ejemplo y manual | `axiom workspace validate` → `LoadConfig` y `ValidateRepositoryPolicies` | 450 | — |
| S2 | `axiom branch status` de solo lectura: CLI y ayuda (cuatro verbos; el resto da código 2), dominio (tipos, `Label`, política, `Plan` sin `--archive`), `gitcommon` (`Run`, `Observe`, `CommonDir`, `TopLevel`, `ReadHead`, `CommonDirFromFiles`, `GitRootFromFiles`), `gitcli` de lectura, `registry.Load` y `List`, `PendingArchive`, `status` con y sin `--change`, prueba de arquitectura | `axiom branch status` | 750 | S1 |
| S3 | `ensure` (núcleo): modo `worktree`, base y `--fetch`, rutas y límite de 220, registro (`Save`, `Lock`, reconstrucción), idempotencia, `branch_checked_out_elsewhere`, `git_failed`, `config_invalid` | `axiom branch ensure` | 550 | S2 |
| S4 | `ensure` (variantes): `in_place` y `dirty_worktree`, `ask` y `decision_required`, excepciones y `plan_conflict`, multirepo y specs in situ, ODD con `--repo`, preferencias de lectura (`hub.LookupWorktreePrefs`, `DefaultWorktreeRoot`) | `axiom branch ensure` | 350 | S3 |
| S5 | D-25: `ensure --archive`, `change_not_integrated`, `archive_branch_not_applicable`, S-15 y S-16, rama de archivado en `status` | `axiom branch ensure --archive` | 300 | S4 |
| S6 | `done`: capas, capacidad de `merge-tree`, desmontaje, `in_place`, rama de archivado; adaptador `LineageProbe` (A-4) | `axiom branch done` | 420 | S5 |
| S7 | Hub: `SetWorktreePrefs`, `ValidateWorktreePrefs`, `IsGitWorktree` por evidencia (`LinkedWorktreeRoot`, `MainWorktreeFromFiles`), flags y pregunta de `init` y `project add`, rutas de preferencias del dashboard y modal | `axiom init`, `axiom project add`, `axiom ui` | 450 | S4 |
| S8 | `VerdictFor` y `branch.Decide` (contextos y consultantes); guarda A en todos los escritores con su contexto (incluido `writeIncrement`, nuevo, extraído de `CreateIncrement`); instantánea y negativa de `seal`; literales `main`; inventario de escritores | `axiom sdd kickoff seal` (y el resto de escritores) | 600 | S5 |
| S9 | `sdd status`: guarda B (fase archive), S-9, raíces y reubicación en los dos detectores, fuentes por rol, REQ-25.44 con precedencia; `Locate` desde worktrees (REQ-25.28) y `workspace validate` desde un worktree | `axiom sdd status` y `axiom sdd continue` | 650 | S8 |
| S10 | Dashboard: `ensure` antes de `writeIncrement`, 409 tipado, política en solo lectura, listados con precedencia, `pendingArchive` y `X-Axiom-Notes`; TUI | `axiom ui` y `axiom change create` | 450 | S7, S9 |
| S11 | `guard` (bloqueo y aviso sin procesos, cargas de los tres runtimes) e instalación por `sync` (Claude `Pre` y `Post`, Codex, plugin de OpenCode, retirada) | `axiom branch guard`; `axiom sync` → `inject.go` | 700 | S8 |
| S12 | Doctrina (`routing.go`, sección de kickoff), skills (incluido el flujo D16 de `sdd-archive`), dogfooding (`axiom.yaml`, `CLAUDE.md`, `AGENTS.md`, `GEMINI.md`, envoltorio `axiom-worktree`, retirada de `cleanup-worktree`, mensajes de `guard-primary-checkout`), documentación y goldens | `axiom sync` (assets); pins | 600 | S9, S10, S11 |

- **Total:** 450 + 750 + 550 + 350 + 300 + 420 + 450 + 600 + 650 + 450 + 700 + 600 = **6.270 líneas** (5.650–6.900 con ±10 %).
- **Diferencia con la versión anterior** (5.200–6.200):
  - **Quita:**
    - el comprobador de CI y el invariante (unas 250 líneas);
    - la exención de la compuerta `integration` (unas 40);
    - el límite medido (unas 40).
  - **Añade:**
    - la rama de archivado (unas 380, repartidas en S5, S6, S8, S9 y S10);
    - el aviso de la guarda D con sus canales `Post` y `after` (unas 250);
    - las lecturas de ficheros sin procesos (unas 80);
    - los pendientes de archivar (unas 180);
    - `status` sin `--change` (unas 60);
    - el envoltorio de los scripts (unas 120);
    - las rebanadas verticales, que repiten cableado de CLI y pruebas de extremo a extremo (unas 100).
- **Dependencias:** S1 → S2 → S3 → S4 → {S5, S7}; S5 → S6; S5 → S8 → {S9, S11}; S7 + S9 → S10; S9 + S10 + S11 → S12.
- **Estrategia.** `single-pr` con un presupuesto de 400 líneas: ninguna rebanada cabe holgadamente en él. `sdd-tasks` debe emitir `Decision needed before apply: Yes`, `Chained PRs recommended: Yes` y `400-line budget risk: High`.

## 12. Puntos abiertos para la compuerta `design`

- [ ] **G-10:** contexto de archivado para `integrate prepare|verify` y `gate record --gate integration`.
- [ ] **G-11:** unión de contextos en la guarda D para los documentos de Axiom.
- [ ] **G-12:** `guard-primary-checkout.ps1` se conserva como política local (ni envoltorio ni retirada).
- [ ] **G-13:** `change_not_integrated` con un mensaje propio cuando la base ya contiene el cambio archivado.
- [ ] **A-4:** la sonda RDD de `done` sin lanzar git; se verifica en S6.
- [ ] **A-5:** canal de aviso de Codex (`PostToolUse`) y `MultiEdit` en Claude; se verifican en S11 junto con A-2.
- [ ] **Entrega:** la decide `sdd-tasks` (§11).
