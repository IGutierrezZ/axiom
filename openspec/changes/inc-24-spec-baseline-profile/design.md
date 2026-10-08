# Diseño: Baseline del proyecto y raíz multirepo para SDD (inc-24-spec-baseline-profile)

> Fase `design` · rama `feat/inc-24-spec-baseline-profile` sobre `main` 3a35fc66 · 2026-10-04 · **revisión 1**.
> Entradas: `proposal.md` (aprobada con S1–S6 y los valores por defecto de Q3–Q16, incluida Q15), `specs/*/spec.md` (enmendadas durante esta revisión: 79 requisitos, 335 escenarios), `exploration.md` y `spike.md`.
> Revisión 1: el usuario rechazó la compuerta `design` para corrección tras una validación independiente (veredicto «PASS WITH NOTES», 78/78 requisitos trazados). Esta revisión corrige los hallazgos H1–H14, aplica las decisiones del usuario sobre la §13 anterior, se alinea con la spec enmendada (§0.4) e incorpora las correcciones con las que el usuario condicionó la aprobación de las compuertas (§0.5).
> Idioma: castellano. Los identificadores de código, paquetes, tipos, funciones, rutas, flags y claves YAML se conservan en su forma original. El cliente se cita solo como «proyecto de referencia».
> Este documento fija el CÓMO a nivel de arquitectura. No contiene tareas.

## 0. Resumen

inc-24 se construye sobre cuatro piezas:

1. **Un primitivo de resolución**, `workspace.Locate`, que desde cualquier directorio de partida devuelve carpeta maestra, `workspaceRoot`, planning home, origen y configuración. Re-enraíza `workspaceRoot` en el repositorio de especificaciones y deja `openspec/` literal debajo (B'). Los dos puntos de entrada del motor y todos los consumidores lo usan.
2. **Una declaración por capas en `axiom.yaml`.** Los bloques `baseline:` y `technical_context:` se decodifican de forma estricta y separada de la configuración base. El status v2 los proyecta como campos aditivos `omitempty`, y una consulta única («¿la baseline es del proyecto?») gobierna todas las guardas.
3. **Un paquete nuevo solo-fork, `internal/specbaseline`.** Compone con perfil de requisito sin tocar la composición nativa, asigna y sustituye los IDs definitivos, planifica la integración del cambio y verifica de forma nativa la edición dirigida (E2). Se expone con `axiom sdd archive-compose --cwd --document` y con el verbo nuevo `axiom sdd integrate check|prepare|verify|show`.
4. **Skills que leen localizadores del status** a través de una convención compartida nueva, `_shared/project-baseline-convention.md`, con cambios de pocas líneas en cada skill de fase.

La composición nativa (`### Requirement:`) no se modifica: R1 se cumple por construcción, no solo por pruebas.

### 0.1 Decisiones clave

| ID | Decisión | Sección |
|---|---|---|
| D-01 | `workspace.Locate` en `internal/workspace`; `Location` como único resultado; rechazos tipados `*LocateError`; búsqueda de cambios archivados con prefijo de fecha (`kickoff.ArchivedChangeRoot`) | 2.1 |
| D-02 | Descubrimiento ascendente por proximidad; sin deducción por sufijo en `monorepo-embedded`; aviso de `<maestra>/openspec` divergente; techo de búsqueda solo para pruebas (`workspace.DiscoveryCeilingOverride`) | 2.2 |
| D-03 | Carga por capas. Con puntero, un `axiom.yaml` ilegible o con base inválida rechaza; sin puntero, se conserva la resolución embebida. Los errores de los bloques nuevos nunca hacen rechazar | 2.3 |
| D-04 | Esquema de `baseline:` y `technical_context:` con decodificación estricta y errores acumulados; `scenarios: include` exige `level` ≤ 3; validación de enlaces simbólicos e identidad de rutas con `internal/pathidentity`; almacén de ficheros exigido | 2.4 |
| D-05 | Campos aditivos `baseline` y `technicalContext`; decorador fuera de `baseStatus`; causa en `notes` y en `blockedReasons` solo cuando la ruta sería `archive` | 2.5 |
| D-06 | Consulta única `Location.BaselineOwnership()` y guarda `Location.GuardSpecsWrite(op)` en todos los creadores; `axiom init` valida las declaraciones del `axiom.yaml` que encuentre y rechaza, sin escribir ni registrar, solo si el propio destino contiene un puntero que no se puede resolver o que resuelve otro planning home | 2.6 |
| D-07 | livingdoc, knowledge, dashboard y TUI resuelven con `Locate` en cada llamada; todos emiten el aviso de ubicación divergente (Q14) | 2.7 |
| D-08 | Composición con perfil en `internal/specbaseline`; los cuatro parsers existentes no cambian | 2.8 |
| D-09 | Bloque por nivel y cola de secciones contiguas con aviso; el cuerpo del delta se comprueba pronto (declaración e `integrate check`) y archive es la última guarda | 2.9 |
| D-10 | Delta nativo, render por plantilla, notas fuera, escenarios `include`/`omit` | 2.10 |
| D-11 | Ubicación explícita de ADDED; correspondencia de IDs calculada una vez; texto pre-sustituido para el agente | 2.11 |
| D-12 | Modelo por líneas con EOL por línea y BOM; validación completa en memoria antes de escribir | 2.12 |
| D-13 | Integración por cambio: `integrate prepare` → `archive-compose --change --in-place` → edición dirigida → `integrate verify`; plan y traza en `integration.yaml`; `INDEX.md` regenerado con `axiom archive sync` en la baseline por defecto | 2.13 |
| D-14 | La edición dirigida la hace `sdd-archive`, lanzado con el modelo de `sdd-apply` cuando hay trabajo dirigido; la elevación automática solo existe en Claude Code (R20) | 2.14 |
| D-15 | Verificador nativo E2 sobre git de solo lectura, comparado con el `HEAD` que registra el plan y con una foto acotada de la suciedad previa | 2.15 |
| D-16 | Delta de contexto `context-impact.md` con gramática determinista y anclas por ruta de encabezados | 2.16 |
| D-17 | Convención compartida nueva y líneas de pin por skill; detalle en `references/` | 2.17 |
| D-18 | Las compuertas `design` y `role-apply` presentan el delta de contexto; la remediación por el delta reabre la compuerta con una línea de registro, sin cambiar el digest | 2.18 |
| D-19 | Regla de ODD en el paso 7 que genera `routing.go`; regeneración de `.codex/AGENTS.md` | 2.19 |
| D-20 | Cumplimiento de ratchets, pins y goldens, con cada función nueva en la unidad que introduce su primer llamador | 2.20 |
| D-21 | Baseline del proyecto o contexto mantenido exigen almacén de ficheros (`openspec` o `hybrid`, REQ-24.62): almacén efectivo declarado o resuelto; archive se detiene (caso (g)) y `workspace validate` reporta el valor declarado | 2.21 |
| D-22 | Lo que no cambia | 2.22 |

### 0.2 Unidades de entrega candidatas (detalle en §12)

La fase de tareas decide la agrupación en PR. Estas unidades respetan las dependencias y dejan el árbol verde por separado.

| Unidad | Contenido | Depende de |
|---|---|---|
| A1 | Resolver, verbos SDD, aislamiento de pruebas | — |
| A2 | Consumidores de catálogo (livingdoc, knowledge, dashboard, TUI) | A1 |
| B1a | Declaración y validación en `internal/workspace` | A1 |
| B1b | Proyección en el status y bloqueo del archivado | B1a |
| B2a | Guardas nativas contra una segunda baseline y baseline externa en el catálogo | B1b, A2 |
| B2b | Skills de lectura (convención compartida, `sdd-init`, contexto técnico, prohibición de referencia) | B2a |
| C1a | Modelo de documento, perfil y composición sin ADDED; modo documento de `archive-compose` | B1a |
| C1b | ADDED, ubicación, asignación y sustitución de IDs, `scenarios: omit` | C1a |
| C2a | Plan: delta de contexto, anclas, `integrate check` y `prepare`, `archive-compose --change --in-place` | C1b |
| C2b | Verificador E2 e `integrate verify` y `show` | C2a |
| C3a | Skills de integración y archive | C2b, B2b |
| C3b | Compuertas, regla de ODD, documentación, goldens y `.codex/AGENTS.md` | C3a |

### 0.3 Revisión 1: cambios por hallazgo

| Hallazgo | Cambio | Dónde |
|---|---|---|
| H1 | `TestRunSDDStatusJSON` y `TestRunSDDStatusNonexistentChange` se aíslan con `t.Chdir(t.TempDir())`; con el descubrimiento ascendente llegarían a la raíz del repositorio, donde `.engram/` está versionado y no hay `artifact_store`, y ejecutarían `engram export`. CI comprueba después de las pruebas que el checkout sigue limpio | §7.4, R28, A1 |
| H2 | `kickoff.ArchivedChangeRoot` busca el nombre exacto y la entrada más reciente `YYYY-MM-DD-<cambio>`; lo usan `resolveGovernanceChangeRoot` e `integrate show` | D-01, D-13 |
| H3 | El rechazo tardío del cuerpo se sustituye por detección temprana: regla de declaración (`include` exige `level` ≤ 3), `integrate check` en la fase spec y archive como última guarda | D-04, D-09 |
| H4 | No figura en la lista de correcciones de esta revisión | — |
| Decisiones del usuario | D-03, D-02 y D-14 confirmadas; D-06 y D-21 aceptadas e incorporadas a la spec (multi-project-hub REQ-2.2, REQ-24.62) | §0.4, §6.3, §13 |
| H5 | D-03 distingue con y sin puntero en cada caso | D-03 |
| H6 | El plan registra el `HEAD` de cada repositorio y verify compara contra él; la foto de suciedad se limita a 1.000 entradas | D-13, D-15, §4.5 |
| H7 | Con baseline por defecto solo se permiten `specs/<dominio>/spec.md` de los dominios del cambio e `INDEX.md` | D-15 |
| H8 | Procedimiento de reapertura de compuertas documentado (REQ-21.12), sin tocar el digest, con su justificación | D-18, D-17 |
| H9 | §9 corregida; nota de `Location` cuando `<maestra>/openspec` existe y difiere del planning home | D-02, §9 |
| H10 | Tabla función → unidad para el ratchet de deadcode; ningún golden contiene el paso 7; regeneración de `.codex/AGENTS.md` | D-19, D-20 |
| H11 | `sdd-tasks` coherente entre D-17 y D-22; andamiaje del perfil `knowledge` corregido; las causas de cara al usuario no citan IDs de diseño | D-05, D-06, D-22 |
| H12 | Modo documento con `--supersede`, `--superseded-requirements` y detección de «ya integrado»; pin de REQ-24.47 en `sdd-archive/SKILL.md`; tipos que faltaban definidos; Q14 en todos los consumidores de catálogo | D-07, D-13, D-17, §4 |
| H13 | Cita del spike corregida: 511 es el número de encabezados; el máximo, que el spike ya registra, es `RF-622` | D-11 |
| H14 | Identidad y contención de rutas con `internal/pathidentity`; enlaces simbólicos validados también en el contexto técnico; rutas citadas con `pathquote.Quote` | D-02, D-04, D-15 |
| Docs | Los pins de `docs/` los lee `internal/workspace/docs_contract_test.go` | §7.1 |
| REQ-21.17(4) | `INDEX.md` lo produce el paso `axiom archive sync` de la rama por defecto; no hay roadmaps declarables | D-13 |
| Techo | `AXIOM_CEILING_DIRECTORIES` deja de ser variable de entorno y sale de la documentación | D-02 |

### 0.4 Alineación con la spec enmendada

| Lectura de la spec | Cambio en el diseño | Dónde |
|---|---|---|
| multi-project-hub REQ-2.2 y REQ-24.6: `axiom init` se rechaza, sin escribir ni registrar, solo cuando el propio destino contiene un `.axiom-workspace` que resuelve otro planning home o que no se puede resolver; un puntero en un ancestro no lo impide | `init` solo evalúa los marcadores del propio destino. Resuelve su puntero con el primitivo (`ResolveOwnPointer`) y rechaza con el mismo error ante cualquier causa de REQ-24.7. Si resuelve, compara el planning home con `<ruta>/openspec` por ruta, sin exigir que exista ninguno de los dos. La continuación es `axiom init --path "<workspaceRoot>"` o la receta documentada | D-02 (pasos 4 y 8), D-06 |
| REQ-24.62 (nuevo): baseline del proyecto o contexto mantenido exigen almacén `openspec` o `hybrid` | Almacén efectivo: el declarado en `openspec/config.yaml` y, sin declaración, el resuelto por el status. Bloquea solo el archivado (caso (g) de REQ-24.56) y no invalida la declaración: `baseline` y `technicalContext` se siguen proyectando. `workspace validate` solo evalúa el valor declarado | D-04, D-05, D-21 |
| REQ-24.42, REQ-24.60, REQ-24.54 y REQ-24.56 (e): cuerpo sin encabezados de nivel ≤ `level`, detectado en tres puntos; `sdd-spec` ejecuta `axiom sdd integrate check` | Se mantiene el nombre del verbo. Las detecciones nombran dominio, requisito, encabezado, línea y continuación; `check` no exige git ni autorización | D-04, D-09, D-13, D-17 |
| REQ-24.7, REQ-24.18 y REQ-24.22: un puntero a un `axiom.yaml` ilegible o con base inválida se rechaza y no hay respuesta de baseline; un puntero a una configuración legible cuyo único defecto es un bloque nuevo inválido no se rechaza | D-03 ya lo cumplía; D-06 explicita que los creadores rechazan con ese mismo error | D-03, D-06 |
| REQ-24.5: excepción embebida (sin deducción por sufijo en `monorepo-embedded`) | La spec la fija; D-02 la cita y deja de presentarla como interpretación del diseño | D-02 |
| REQ-24.51 (c): `<planning home>/INDEX.md` en el conjunto permitido con la baseline por defecto | La spec lo fija; D-15 (c) lo cita | D-15 |
| REQ-2.2, punto 4: `init` valida la estructura de `baseline:` y `technical_context:` de cualquier `axiom.yaml` que encuentre | `init` usa el mismo validador de carga (`ReadDeclarations`, sobre `parseLayered`) que `LoadConfig`, `Locate` y `workspace validate`, e informa de cada violación | D-06 |

### 0.5 Correcciones de la aprobación condicionada

| Punto | Corrección | Dónde |
|---|---|---|
| M1 | `TestArchiveEndpoints` se aísla en A1 en un espacio temporal; la comprobación de checkout limpio de CI se mantiene y debe pasar en la fusión de A1; nota de coordinación con la tarea T3 de la otra sesión. Resuelto por main (#71/#72): el aislamiento de la prueba ya está fusionado y se omitió el hunk al rebasar | §5 (A1), §7.4, §10 (R28), §11, §12 |
| M2 | `init` resuelve su puntero propio con el primitivo (`ResolveOwnPointer`). Rechaza con el mismo error ante cualquier causa de REQ-24.7 (`pointer_broken`, `config_unloadable`, `specs_repository_missing`) o ante la ambigüedad de REQ-24.6. Si resuelve, rechaza solo cuando el planning home difiere de `<ruta>/openspec` (`SamePlanningHome`, por ruta y sin exigir existencia). El paso 8 exige `workspaceRoot` y no el planning home, así que un `multirepo` con `specs_repository: openspec` no rechaza antes de inicializar | D-02 (pasos 4 y 8), D-06, §4.1, D-20, §7.1 |
| m1 | Cita del máximo del spike (`RF-622`, con huecos en la numeración) | D-11 |
| m3 | `Declarations.Readable` se sustituye por `ParseErr`, nulo cuando no hay `axiom.yaml`; prueba de equivalencia sin `axiom.yaml` | D-03, D-05, §4.1, §7.1, §7.2 |
| m5 | Excepción embebida de REQ-24.5, `INDEX.md` en REQ-24.51 (c) y validación de `init` (REQ-2.2, punto 4) citadas como texto de la spec | §0.4, D-02, D-06, D-15, §6 |
| m14 | Resumen de D-06 alineado con el rechazo por puntero propio no resoluble | §0.1, §0.4, D-06, §13 |
| m2, m4, m6–m13 | Sin corregir en esta revisión: anotados con su corrección para la fase de tareas | §13 |

## 1. Enfoque técnico

| Principio | Cómo se aplica |
|---|---|
| Re-enraizar, no hilar (S1, D4) | `workspaceRoot` pasa a ser el padre del planning home. Las ~70 hojas que construyen `<workspaceRoot>/openspec/...` siguen siendo válidas y se cambian solo los puntos de entrada |
| Invariante B' | `PlanningHome == filepath.Join(WorkspaceRoot, "openspec")` en todas las topologías. `PrepareChangeInstanceConsent`, `RefuseArchivedRoot`, el ledger en el git common dir y `ArchivedProjection.Path` siguen siendo válidos sin generalizarlos |
| Contrato solo aditivo (S4, REQ-24.20) | Los campos nuevos se añaden fuera de `baseStatus`; la firma que usa `status_v2_clean_break_test` no cambia |
| Fallar cerrado (REQ-24.18) | Una declaración inválida nunca degrada a la baseline por defecto: bloquea el archivado y hace rechazar a los creadores |
| Lógica nueva en código solo-fork (R9) | `internal/workspace` y `internal/specbaseline` concentran la lógica. En `sddstatus` y `cli` (código compartido con upstream) solo hay costuras de pocas líneas y ficheros nuevos |
| Determinismo verificable | Composición, asignación de IDs, plan y verificación son nativos y deterministas. El modelo solo interviene en la edición dirigida de prosa, y esa edición se verifica después de forma nativa |
| Identidad de rutas por el sistema de ficheros | Igualdad y contención de directorios con `internal/pathidentity` (`SameDirectory`, `Contains`) y de ficheros con `os.SameFile`; nunca comparando cadenas. Las rutas que aparecen en mensajes se citan con `pathquote.Quote` |
| Sin operaciones remotas | Axiom no ejecuta `fetch`, `pull` ni `push`. El aviso de rama por detrás usa referencias locales |

## 2. Decisiones de arquitectura

### 2.1 D-01 — Primitivo `workspace.Locate`

**Decisión.** Fichero nuevo `internal/workspace/locate.go` con:

- `Locate(start string) (Location, error)`, determinista y de solo lectura.
- `Roots(cfg, master) (workspaceRoot, planningHome string)`, el enraizado por topología puro.
- `*LocateError`, el rechazo tipado, que siempre nombra la causa y una continuación accionable con las rutas citadas por `pathquote.Quote`.

Todos los consumidores obtienen de `Location` la carpeta maestra, `workspaceRoot`, el planning home, el origen, la ruta efectiva de `axiom.yaml`, la configuración base y las declaraciones. `ResolveConfigFile` y `LoadConfig` se mantienen con su semántica actual para los consumidores que no resuelven raíces (skillregistry, semantic, autoskill), y en particular conservan el fallback de subdirectorios.

Costuras en los consumidores:

| Consumidor | Cambio |
|---|---|
| `sddstatus.resolveWorkspaceRoot` | Con `ResolveOptions.WorkspaceRoot` explícito conserva su comportamiento actual, incluido el mensaje que figura en la línea base del ratchet. Con `CWD` delega en `Locate` |
| `sddstatus.Resolve` | Resuelve una vez, pasa la raíz resuelta como `WorkspaceRoot` explícito a `resolveByPreferenceOrder` (sin segunda resolución) y aplica el decorador de D-05 |
| `cli.resolveGovernanceChangeRoot` | Devuelve `(workspace.Location, changeRoot, error)`. Busca primero `openspec/changes/<cambio>` y después, con `kickoff.ArchivedChangeRoot`, el cambio archivado: el nombre exacto (forma heredada que usan las pruebas existentes) o la entrada más reciente `YYYY-MM-DD-<cambio>`, que es la que produce `sdd-archive`. Así `RefuseArchivedRoot` rechaza por cambio archivado en lugar de responder «no existe» (REQ-24.9, escenario 3). `kickoff seal --infer` toma la configuración de `loc.Config` y desaparece el `os.Stat(<raíz>/axiom.yaml)` previo |
| `cli` `sdd-attempt grant` | Resuelve `--cwd` con `Locate` y abre el `RuntimeStore` sobre `loc.WorkspaceRoot`, que es el repositorio git de especificaciones |
| `cmd/axiom` `resolveChangeDir`, `resolveHandoffFile`, `runHandoffCreate`, `runHandoffValidate`, `runRoleList`, `runRoleStatus`, `runRoleBarrier` | Usan `loc.WorkspaceRoot` para `openspec/changes/...` y `loc.Config` para los roles. Desaparecen los cuatro `os.Stat(<raíz>/axiom.yaml)` que cegaban el puntero |
| `cli` `archive-compose` | Con `--cwd`, resuelve el planning home y la declaración (D-13) |

`kickoff.ArchivedChangeRoot` es una función nueva en `internal/kickoff/archived.go`, junto a `RefuseArchivedRoot`. Reproduce la regla de `sddstatus.newestArchivedOpenSpecEntry` (`status.go:326`: prefijo de fecha completo y sufijo exacto) sin modificar ese fichero compartido con upstream. La usan también `integrate show` (D-13) y, con ello, cualquier verbo que deba encontrar un cambio ya archivado.

**Alternativas descartadas.**

- *Ampliar `ResolveConfigFile`.* Mezclaría el descubrimiento de configuración (con el fallback de subdirectorios, que REQ-24.2 prohíbe como marcador) con la resolución del planning home, y cambiaría el comportamiento de skillregistry y del dashboard.
- *Un `PlanningHome` hilado por todo `sddstatus` y las skills.* Son ~70 apariciones, más los pins y la generalización de `PrepareChangeInstanceConsent:152`. Ya se descartó en la propuesta (S1).
- *Resolver en cada verbo por separado.* Reproduce las tres convenciones incompatibles que inc-24 elimina.
- *Exportar `newestArchivedOpenSpecEntry` desde `sddstatus`.* Añade superficie pública a un paquete compartido con upstream para un uso solo-fork.

**Justificación.** `internal/workspace` es código solo-fork, está fuera del ratchet de refusals y ya contiene el puntero y el loader. Un único resultado tipado hace que REQ-24.8 («mismo planning home desde cualquier punto») sea una propiedad del primitivo y no de cada verbo.

### 2.2 D-02 — Descubrimiento ascendente, precedencia y techo de búsqueda

**Decisión.** El algoritmo de `Locate` es el siguiente.

1. **Directorio de partida.** `filepath.Abs` y `Clean`. Si no existe, no es un directorio o es la raíz del sistema de ficheros, se devuelve `LocateError{Kind: start_invalid}` con la continuación `axiom sdd status --cwd "<directorio-del-proyecto>" --json` (REQ-24.2).
2. **Recorrido.** Se evalúa el propio directorio de partida y después sus ancestros. Se detiene en el primero que contiene `.axiom-workspace` (fichero regular), `axiom.yaml` (fichero regular) o `openspec/` (directorio). Si coinciden varios en un mismo directorio, la precedencia es `pointer` > `config` > `openspec`. El fallback de subdirectorios (`repo-specs`, `specs`, `openspec`, `.openspec`, `especificacion`) nunca actúa como marcador.
3. **Sin marcador.** `Master = WorkspaceRoot = start`, `PlanningHome = start/openspec`, `Origin = start`. No se crea nada.
4. **Marcador `pointer` en `D`.** `ResolveOwnPointer(D)` (§4.1) aplica al puntero de `D`, sin ascender, las reglas del primitivo para un puntero: resuelve el puntero con `resolveFromPointerFile` y después aplica los pasos 7, 8 y 9. Un puntero ilegible, un destino inexistente o un puntero sin `config` ni `specs` producen `pointer_broken` (REQ-24.7, caso 1). Un destino que no se puede cargar como YAML, o cuya configuración base es inválida, produce `config_unloadable` con la continuación `axiom workspace validate --path <D>` (caso 2): el planning home es desconocido y no se adivina. En otro caso, `Master = D` y las raíces salen de `Roots`. `Locate` delega en `ResolveOwnPointer` cuando el marcador es un puntero, y `axiom init` la usa para el puntero de su propio destino (D-06).
5. **Marcador `config` en `D`.** Carga por capas (D-03). Si la configuración base no es cargable, se conserva la resolución embebida (`WorkspaceRoot = D`) y se añade una nota a `Location.Notes` (caso 3). Si es cargable:
   - en `monorepo-embedded`, `Master = D`;
   - en `monorepo-decoupled` y `multirepo`, `Master` se deduce según REQ-24.5 en sus cuatro pasos: vacío o `.` → `D`; `D` contiene la ruta `specs_repository` → `D`; la ruta de `D` acaba en los segmentos de `specs_repository` → `D` sin el sufijo; en otro caso, `D`. La comparación trata `/` y `\` como equivalentes y no distingue mayúsculas en Windows.
6. **Marcador `openspec` en `D`.** `Master = WorkspaceRoot = D`, `Origin = openspec`.
7. **`Roots(cfg, master)`.** En `monorepo-embedded`, o sin configuración, `(master, master/openspec)`. En `monorepo-decoupled` y `multirepo`, `repo = master/specs_repository` (una ruta absoluta se usa tal cual); si `Base(repo) == "openspec"`, el resultado es `(Dir(repo), repo)`, y si no, `(repo, repo/openspec)`. Un `specs_repository` vacío o `.` en estas dos topologías se trata como la propia carpeta maestra; lo reporta `workspace validate`, no `Locate`.
8. **Repositorio de especificaciones inexistente.** En `monorepo-decoupled` y `multirepo`, si `workspaceRoot` no existe, se devuelve `specs_repository_missing` con la ruta esperada y la acción de crearlo o corregir `specs_repository` (caso 4). Bajo B', el repositorio de especificaciones es `workspaceRoot`, que coincide con `<maestra>/<specs_repository>` salvo cuando `specs_repository` termina en `openspec`. En ese caso el último segmento es el planning home y no un repositorio, así que su ausencia no rechaza. El planning home nunca tiene que existir para resolver, igual que en `monorepo-embedded` o sin marcador. Así, un `multirepo` con `specs_repository: openspec` resuelve antes de que `axiom init` cree `openspec/`.
9. **Ambigüedad (REQ-24.6).** Para cualquier origen, si la carpeta maestra efectiva contiene `.axiom-workspace` y un directorio `openspec/`, y ese directorio no es el planning home (`!pathidentity.SameDirectory(PlanningHome, master/openspec)`), se devuelve `ambiguous_planning_home`. El mensaje nombra los dos directorios y las dos continuaciones: consolidar los cambios de `<maestra>/openspec` en el planning home resuelto y retirar el directorio sobrante, o retirar el puntero. Así el rechazo es idéntico desde la maestra, desde el repositorio de especificaciones y desde un repositorio de código.
10. **Planning home divergente sin puntero.** Si no hay puntero, la topología es `monorepo-decoupled` o `multirepo`, y `<maestra>/openspec` existe pero no es el planning home, `Location.Notes` gana un aviso no bloqueante que nombra las dos rutas y la acción (consolidar los cambios en el planning home resuelto). Es el caso en que inc-24 cambia el resultado respecto a hoy, que resolvía `<cwd>/openspec` (§9).

`Location.MarkerDir` registra el directorio del marcador (vacío con `Origin = start`).

**Excepción embebida de REQ-24.5.** La fija la spec: en `monorepo-embedded` la carpeta maestra es siempre el directorio del `axiom.yaml`, y la deducción por sufijo solo se aplica a `monorepo-decoupled` y `multirepo`. Sin ella, un proyecto embebido cuyo directorio se llamara igual que `specs_repository` (por ejemplo `/trabajo/specs` con `specs_repository: specs`) movería `workspaceRoot` al padre y rompería la invariancia exacta de REQ-24.4.

**Techo de búsqueda solo para pruebas.** Variable de paquete exportada `workspace.DiscoveryCeilingOverride []string`, nula en producción. Si no es nula, el recorrido nunca evalúa esos directorios ni sus ancestros (semántica de `GIT_CEILING_DIRECTORIES`); la comparación con el techo es por identidad (`pathidentity.SameDirectory`), para que el directorio temporal de macOS, accesible por `/var` y por `/private/var`, se reconozca con cualquiera de las dos grafías. Sigue el patrón existente de `LookPathOverride` y `VersionRunnerOverride`: la fijan los `TestMain` de los paquetes afectados (con `os.TempDir()`), de modo que un marcador en un ancestro del directorio temporal no altera los resultados. No hay variable de entorno ni documentación de usuario, y el producto siempre recorre hasta la raíz del sistema de ficheros, como fija REQ-24.2.

**Alternativas descartadas.**

- *Buscar solo el puntero* (la inclinación original de la exploración). Los proyectos embebidos y el repositorio de especificaciones no tienen puntero, y Q3 aceptó «gana el más cercano».
- *Parar el recorrido en la primera raíz git.* La carpeta maestra canónica no está versionada, así que desde `repo-backend/src` nunca se llegaría al puntero.
- *Un techo por variable de entorno pública.* El usuario la rechazó: añade superficie de producto para resolver un problema de pruebas.
- *Una opción no exportada.* No la pueden fijar las pruebas de `sddstatus`, `cli`, `livingdoc`, `knowledge`, `dashboard`, `tui`, `hub` ni `cmd/axiom`. Una función setter solo llamada desde pruebas, además, haría saltar el ratchet de deadcode.

**Justificación.** Las reglas son las de REQ-24.2 a REQ-24.7. Comprobar la ambigüedad en la carpeta maestra efectiva hace que el resultado no dependa del punto de partida. La comparación por identidad evita falsos «distintos» por mayúsculas o enlaces simbólicos. La variable de prueba sigue un patrón que el repositorio ya usa y no cambia el comportamiento del producto.

### 2.3 D-03 — Carga por capas: configuración base frente a declaraciones

**Decisión.** Función interna nueva `parseLayered(data []byte) (cfg *WorkspaceConfig, baseErr error, decl Declarations)`:

1. `yaml.Unmarshal` a un `yaml.Node`. Si falla, `baseErr` y `decl.ParseErr` llevan la causa. `ParseErr` solo es no nulo cuando existe un `axiom.yaml` que no se puede leer: sin fichero, `Declarations` conserva su valor cero, que equivale a «sin declaraciones», de modo que un proyecto sin `axiom.yaml` nunca bloquea el archivado (R1).
2. La configuración base se decodifica desde el nodo con los campos `Baseline` y `TechnicalContext` marcados `yaml:"-"`. Las claves `baseline` y `technical_context` quedan fuera de la decodificación base, y el resto de claves desconocidas se sigue ignorando como hasta ahora (REQ-24.17). Las validaciones base actuales (nombre, topología, roles) se conservan con sus mensajes y su orden, y su fallo va a `baseErr`.
3. Las declaraciones se decodifican aparte desde sus nodos, de forma estricta (D-04). Sus errores van a `decl.BaselineErr` y `decl.TechnicalContextErr`, aunque la base haya fallado.

Comportamiento por caso (confirmado por el usuario):

| Caso | Con puntero `.axiom-workspace` | Sin puntero (`axiom.yaml` directo) | `ParseConfig` / `LoadConfig` |
|---|---|---|---|
| YAML ilegible | Rechazo `config_unloadable` (REQ-24.7, caso 2): el planning home es desconocido | Resolución embebida y nota (caso 3). Consulta de baseline `indeterminate`, status operativo y archivado bloqueado (REQ-24.18) | Error, como hoy |
| Base inválida (nombre, topología, roles) | Rechazo `config_unloadable` (caso 2) | Resolución embebida y nota (caso 3). Las declaraciones se evalúan igualmente; sin bloques nuevos, el archivado no se bloquea por esta causa | Error, como hoy |
| Declaración inválida, base válida | No rechaza. Status operativo, archivado bloqueado (D-05), creadores rechazan (D-06) | Igual que con puntero | Error que acumula todas las violaciones (REQ-24.17) |
| Declaración inválida, base inválida y sin puntero | — | Resolución embebida y nota; archivado bloqueado por la declaración | Error |

Los errores de los bloques nuevos nunca hacen rechazar a `Locate`. Un puntero hacia un `axiom.yaml` que no se puede leer o cuya base es inválida sí rechaza, porque sin esa configuración no hay forma de saber dónde está el planning home.

**Alternativas descartadas.**

- *Que una declaración inválida haga fallar `Locate`.* Desde la carpeta maestra, una baseline mal escrita impediría consultar el status y planificar, lo que contradice REQ-24.18: «las dependencias de las fases de planificación NO DEBEN pasar a `blocked`» y «el status DEBE señalar la causa».
- *Un `UnmarshalYAML` estricto en los tipos nuevos dentro de una sola decodificación.* Un bloque inválido haría fallar la configuración entera y no habría forma de separar los canales de error.

**Justificación.** Con dos canales se cumplen a la vez REQ-24.7 (rechazo cuando el puntero no lleva a una configuración utilizable), REQ-24.17 («la carga falla» para la API estricta), REQ-24.18 (status operativo y archivado bloqueado) y REQ-24.22 (`del proyecto` cuando el bloque es válido aunque fallen los roles). Los verbos de roles siguen funcionando con una declaración inválida porque leen `loc.Config`, que es la configuración base.

### 2.4 D-04 — Esquema y validación estricta de `baseline:` y `technical_context:`

**Decisión.** Tipos en el fichero nuevo `internal/workspace/declarations.go` (firmas en §4.1). La validación es la siguiente.

**Claves.** Un recorrido explícito de los nodos de mapeo comprueba las listas cerradas de claves:

| Nodo | Claves admitidas |
|---|---|
| `baseline` | `documents` |
| Documento | `domain`, `path`, `requirement`, `integration`, `scenarios` |
| `requirement` | `heading`, `level`, `identity`, `render` |
| `technical_context` | `maintained`, `reference` |
| Modo de contexto | `entry`, `paths` |

Cada clave desconocida se reporta con su nombre, su línea y las claves admitidas. `yaml.v3` no ofrece `KnownFields` en `Node.Decode`, por eso se recorre el nodo.

**Errores acumulados.** Todas las violaciones de ambos bloques se reúnen con `errors.Join`, en castellano, y cada una nombra el dominio o la ruta y la regla. `workspace validate` reporta así un error por ruta (escenarios de REQ-24.14).

**Reglas sintácticas** (en carga, sin acceder al disco):

| Elemento | Regla |
|---|---|
| `baseline` | Al menos un documento |
| `domain` | `^[a-z0-9]+(-[a-z0-9]+)*$` y único |
| `path` | Relativo, sin unidad de disco ni barra inicial (mismo criterio que `CleanSkillRoot`, idéntico en todos los sistemas) y contenido léxicamente en `workspaceRoot`; único entre documentos |
| `requirement.heading` | RE2 que compila y define los grupos con nombre `id` y `title` |
| `requirement.level` | Entero entre 1 y 6 |
| `requirement.render` | Empieza por exactamente `level` signos `#` seguidos de un espacio y contiene `{id}` y `{title}` |
| `identity` | `id` o `text`; por defecto `id` |
| `integration` | `directed` o `register`; por defecto `directed`; prohibida si hay `requirement` |
| `scenarios` | `include` u `omit`; por defecto `include`; prohibida sin `requirement` |
| `scenarios` efectivo `include` | Exige `level` ≤ 3, porque los escenarios del delta son `#### Scenario:` y con un nivel mayor cortarían el bloque. El error nombra el dominio, el `level` declarado y las dos salidas: `scenarios: omit` o un `level` ≤ 3 (REQ-24.60) |
| Modos de contexto | Al menos `maintained` o `reference`; `paths` no vacío |
| `entry` | Contenida en `paths` del mismo modo |
| Rutas de contexto | Relativas y contenidas léxicamente en la carpeta maestra |
| Solapes | Ninguna ruta en ambos modos ni en `baseline.documents`. Dos rutas solapan si son iguales o una contiene a la otra. En carga se comparan sin distinguir mayúsculas, porque `axiom.yaml` se comparte entre Windows y Linux |

Para la contención en la carpeta maestra se calcula el desplazamiento léxico de la maestra a `workspaceRoot` a partir de la propia configuración: vacío en embebido; `specs_repository`, o su directorio padre si termina en `openspec`, en las demás topologías. Se comprueba que `path.Clean(desplazamiento + "/" + ruta)` no escapa. Si la topología no es legible se usa el desplazamiento vacío, coherente con el caso 3.

**Reglas que dependen del sistema de ficheros** (`workspace.Validate`, que conserva su firma y toma `--path` como carpeta maestra, como hoy):

- Cada `baseline.documents[].path` existe como fichero regular y, tras `filepath.EvalSymlinks`, `pathidentity.Contains(workspaceRoot, ruta)` es cierto (error).
- Cada ruta de `technical_context` (de los dos modos) que exista se resuelve con `EvalSymlinks` y debe cumplir `pathidentity.Contains(maestra, ruta)`; un enlace que sale de la carpeta maestra es un error. Una ruta inexistente produce una advertencia que no altera el resultado (REQ-24.27).
- Los solapes se vuelven a comprobar por identidad sobre las rutas que existen, para detectar grafías distintas del mismo directorio.
- Si se declara `baseline:` con documentos o `technical_context.maintained`, y `openspec/config.yaml` declara `artifact_store` `engram` o `none`, se reporta un error por cada declaración afectada que nombra la declaración (`baseline` o `technical_context.maintained`), el almacén declarado y la continuación: declarar `artifact_store: openspec` o `hybrid` en `openspec/config.yaml`, o retirar la declaración (REQ-24.62, D-21). Sin `artifact_store` declarado no se reporta nada por este motivo. El valor lo lee `workspace.DeclaredArtifactStore(workspaceRoot)`, con las mismas reglas que `sddstatus.declaredArtifactStore` (ficheros `config.yaml` y `config.yml`, claves `artifact_store` y `artifactStore`, comentarios, comillas y mayúsculas). Se duplica porque `workspace` no puede importar `sddstatus` sin crear un ciclo (`sddstatus` → `kickoff` → `workspace`), y una prueba de paridad en `internal/sddstatus` ejecuta los dos lectores sobre las mismas fixtures.

`runWorkspaceValidate` imprime las advertencias también cuando el resultado es `COMPLIANT`; hoy solo lo hace en la rama no conforme.

**Alternativas descartadas.**

- *`KnownFields(true)` sobre todo el documento.* Rompe REQ-24.17 («claves desconocidas fuera de los bloques nuevos se ignoran»).
- *Validación que se detiene en el primer error.* Incumple «se reporta un error por cada una».
- *Detectar el problema de nivel solo al archivar.* Un `level` ≥ 4 con `include` haría fallar todos los deltas con escenarios, y solo se sabría al final del ciclo (H3).

**Justificación.** El mismo validador sirve a `LoadConfig`, a `Locate` y a `workspace validate`. La contención léxica se comprueba en carga sin acceder al disco, y la que depende de enlaces simbólicos queda para `workspace validate`, como fija REQ-24.17.

### 2.5 D-05 — Proyección aditiva en el status v2 y bloqueo del archivado

**Decisión.**

- **Campos.** `Status` y `StatusV2Projection` ganan dos campos opcionales: `Baseline *...` (`json:"baseline,omitempty"`) y `TechnicalContext *...` (`json:"technicalContext,omitempty"`). La forma está en §4.2.
  - `baseline.documents[]` lleva `domain`, `path` absoluto e `integration` (`compose` | `directed` | `register`).
  - `technicalContext.maintained` y `technicalContext.reference` llevan `paths` absolutos y, si se declaró, `entry` absoluta.
  - Un bloque inválido no se proyecta, ni parcial ni por defecto (REQ-24.18).
- **Decorador.** Fichero nuevo `internal/sddstatus/workspace_declarations.go` con `decorateWorkspaceDeclarations(status *Status, loc workspace.Location)`. `Resolve` lo invoca después de `resolveByPreferenceOrder` o de la ruta Engram, y vuelve a generar `PhaseInstructions` si se pidieron. `baseStatus` no cambia de firma, de modo que `status_v2_clean_break_test` pasa sin tocarse (REQ-24.20).
- **Canales.** `notes` recibe siempre los avisos no bloqueantes y las causas de bloqueo del archivado. `blockedReasons` recibe esas causas solo cuando, sin ellas, la ruta sería `archive`. Así ninguna fase de planificación ni `apply` queda bloqueada por estas causas (REQ-24.18), siguiendo el mismo patrón que `archiveDependencyFromGovernance`.

Reglas del decorador para un estado con cambio seleccionado y no archivado. Los textos están pensados para el usuario y no citan identificadores de este diseño:

| Causa | Condición | Efecto y texto de la causa |
|---|---|---|
| Declaración inválida o `axiom.yaml` ilegible | `decl.ParseErr`, `decl.BaselineErr` o `decl.TechnicalContextErr` no nulos (los tres son nulos cuando no hay `axiom.yaml`) | `dependencies.archive = blocked`. «La declaración de `axiom.yaml` no es válida: <error>. Ejecuta `axiom workspace validate --path "<maestra>"` y corrígela antes de archivar» |
| Falta el delta de contexto | `technicalContext.maintained` válido, almacén efectivo `openspec` o `hybrid` y sin `<changeRoot>/context-impact.md` | `dependencies.archive = blocked`. «Falta el delta de contexto `"<changeRoot>/context-impact.md"`: registra las entradas del impacto o «Sin impacto: <motivo>»» (REQ-24.36). Funciona igual en modalidad continua |
| Almacén sin ficheros (REQ-24.62) | Bloque `baseline:` válido con documentos o `technicalContext.maintained` válido, y almacén efectivo `engram` o `none` | `dependencies.archive = blocked`. «La baseline del proyecto (o el contexto técnico mantenido) exige un almacén de ficheros y el almacén es `<almacén>`: declara `artifact_store: openspec` o `hybrid` en `openspec/config.yaml`, o retira la declaración» |
| Ruta `archive` bloqueada | `nextRecommended == "archive"` y alguna causa anterior | `nextRecommended = "resolve-blockers"`. Equivale a recalcular con `resolveNextRecommended`, que con `archive` bloqueado y todo lo demás hecho devuelve ese mismo token |

**Almacén efectivo (REQ-24.62).** Es el que declara `openspec/config.yaml` (`declaredArtifactStore(workspaceRoot)`) y, solo si no hay declaración, el que resolvió el status (`status.ArtifactStore`). El declarado prevalece porque hoy el resolutor informa `openspec` para un `none` declarado cuando el cambio está en disco. La causa de almacén no invalida la declaración a efectos de REQ-24.18: `baseline` y `technicalContext` se siguen proyectando, la consulta de baseline sigue respondiendo `project` y solo se bloquea el archivado.

Avisos solo informativos (`notes`):

- Las notas de `Location`: resolución embebida por un `axiom.yaml` no cargable (REQ-24.7, caso 3) y planning home divergente sin puntero (D-02, paso 10).
- **Rutas relativas a la carpeta maestra en multirepo.** Si `tasks.md` cita entre comillas invertidas una ruta relativa cuyo primer segmento coincide con el nombre de un repositorio de rol declarado, hermano de `workspaceRoot`, y esa ruta no existe bajo `workspaceRoot`, se avisa de que en multirepo los repositorios de código se nombran como `../<repo>/...` o con ruta absoluta (R30).
- Con baseline del proyecto o contexto mantenido, `phaseInstructions.archive` gana una línea que nombra `axiom sdd integrate prepare --cwd "<workspaceRoot>" --change <cambio>`.

**Alternativas descartadas.**

- *Un verbo aparte (`axiom sdd context --json`).* Obligaría a cambiar 12 plantillas de orquestador y sus goldens para que lo invocaran.
- *Insertar las comprobaciones dentro de `resolveByPreferenceOrder`.* Aumenta la divergencia con upstream en el fichero compartido más grande del motor.
- *Poner la causa siempre en `blockedReasons`.* El contrato prohíbe `apply` con `blockedReasons` no vacío (`sdd-status-contract.md`), así que una baseline mal declarada bloquearía la implementación.
- *Proyectar el delta de contexto en el status.* REQ-24.28 lo prohíbe.

**Justificación.** El precedente aditivo existe (`Consent`, `Archived`, `Governance`, `GateQuestion`, `PhaseInstructions`). La ausencia del delta se detecta de forma nativa y temprana, también en modalidad continua, sin proyectarlo: es solo una comprobación de existencia de un fichero que `resolveArtifactPaths` nunca lista.

### 2.6 D-06 — Consulta única de baseline y guarda contra una segunda baseline

**Decisión.** Fichero nuevo `internal/workspace/baseline_guard.go`:

- `Location.BaselineOwnership() (BaselineOwnership, error)` devuelve `default` (sin `axiom.yaml`, o legible sin `baseline:` aunque tenga errores ajenos al bloque), `project` (bloque válido) o `indeterminate` (YAML ilegible o bloque inválido; el error lleva la causa). Delega en `Declarations.Ownership()`, que es la única implementación de la consulta (REQ-24.22). Una causa de almacén (REQ-24.62) no altera la respuesta.
- Cuando el primitivo rechaza la resolución, por cualquier causa de REQ-24.7 (como un puntero roto o un destino que no se puede leer como YAML o tiene la configuración base inválida) o por la ambigüedad de REQ-24.6, no hay respuesta (REQ-24.22). Los creadores que solo escriben rechazan con ese mismo error; `knowledge sweep` y `knowledge query --enrich` omiten la escritura, anotan la causa y completan el resto de su resultado; y `axiom init` rechaza con ese error, sin escribir ni registrar, solo cuando el puntero es el del propio destino (procedimiento siguiente).
- `Location.GuardSpecsWrite(op string) error` devuelve `nil` con `default` y un `*SecondBaselineError` en los otros dos casos.
  - Con `project`, el mensaje nombra la baseline declarada (ruta de `axiom.yaml` y dominios) y la continuación: integrar con el flujo de archive o retirar `baseline:` para volver a OpenSpec.
  - Con `indeterminate`, nombra la causa y `axiom workspace validate --path "<maestra>"`.

Comportamiento por creador (REQ-24.23 y REQ-24.24):

| Creador | Con `project` | Con `indeterminate` | Dónde |
|---|---|---|---|
| `axiom init` | No crea `openspec/specs/` ni `openspec/INDEX.md`. Conserva el resto del andamiaje de su perfil, que no cambia: `full` crea `openspec/changes/` y `.axiom/inbox/skills/`; `knowledge` crea `openspec/changes/` y `.mcp.json` (`hub/init.go` nunca ha creado `.axiom/inbox/skills/` en ese perfil). Informa y sale con `0` | Omite la escritura de la baseline, informa con la causa y sale con `0` | `hub.Initializer.Init`. La validación, la guarda y el rechazo solo consideran los marcadores del propio destino (procedimiento siguiente) |
| `axiom archive sync` y `coldstart` | Rechazo, sin escribir, código ≠ 0 | Rechazo | `livingdoc.Service.Sync` y `ColdStart` |
| `POST /api/archive/sync` y tecla `s` de la TUI | 409 con la continuación; la TUI muestra el rechazo | Igual | El servicio devuelve `*SecondBaselineError`; `server.go` lo traduce a 409 |
| `axiom knowledge sweep` | Omite la siembra y el índice, completa el análisis, `created_specs: []`, nota y código `0` | Igual, con la causa | `knowledge.RunSweep` |
| `axiom knowledge query --enrich` | Responde sin enriquecer y anota la omisión | Igual | `knowledge.RunQuery` |
| `axiom knowledge crawl --record-unit` y `--finalize` | Rechazo, sin escribir | Rechazo | `knowledge.RecordUnit` y `FinalizeCrawl`, que resuelven `Locate(job.WorkspaceRoot)` en el momento de escribir |
| Skill `sdd-init` y rama sin baseline de `sdd-archive` | Texto prescrito (pin) | Texto prescrito | §2.17 |

**`axiom init` y los marcadores del propio destino** (aceptado por el usuario; multi-project-hub REQ-2.2 y REQ-24.6). Antes de escribir ni registrar nada, también antes del `os.MkdirAll` actual, `init` evalúa solo los marcadores del propio destino `<ruta>`, nunca los de sus ancestros. Nunca llama a `Locate`.

1. **Puntero propio.** Si `<ruta>/.axiom-workspace` es un fichero regular, `init` resuelve ese puntero con el primitivo: `ResolveOwnPointer(ruta)`, la rama de puntero de `Locate` sin ascender (D-02, paso 4). Con un puntero en `<ruta>`, `Locate(ruta)` daría el mismo resultado.
   - Si el primitivo rechaza, `init` rechaza con el mismo `*LocateError`, que nombra la causa y la continuación, con cualquier perfil, código ≠ 0, sin escribir ni registrar.
   - Si resuelve y `!SamePlanningHome(loc.PlanningHome, <ruta>/openspec)`, rechaza con `*hub.ForeignPlanningHomeError`, con cualquier perfil, código ≠ 0, sin escribir ni registrar. El mensaje nombra el puntero, el planning home resuelto y la continuación: `axiom init --path "<workspaceRoot>"` o montar la topología a mano según la receta documentada (REQ-24.12).
   - En otro caso no rechaza, aunque `<ruta>/openspec` no exista todavía: es el caso normal de un `multirepo` con `specs_repository: openspec` antes de inicializar, que el paso 8 no rechaza. El `axiom.yaml` que `init` considera existente es `loc.ConfigPath`, así que no escribe `<ruta>/axiom.yaml` cuando el puntero indica otro fichero. La consulta de baseline es `loc.BaselineOwnership()`.
2. **`axiom.yaml` propio sin puntero.** `init` lee sus declaraciones con `ReadDeclarations(<ruta>/axiom.yaml)`, y la consulta es su `Ownership()`, la misma función en la que delega `Location.BaselineOwnership()`. Al no llamar a `Locate`, un rechazo ajeno (por ejemplo, un repositorio de especificaciones aún inexistente) no detiene `init`.
3. **Sin marcadores.** `init` se comporta como antes de este incremento, aunque un ancestro contenga un puntero, también roto, o una declaración: el marcador más cercano gana (REQ-24.2), y el nuevo `axiom.yaml` de `<ruta>` pasa a serlo.

**Errores que hacen rechazar a `init`.** Todos los `LocateErrorKind` que `ResolveOwnPointer` puede devolver, siempre con el mismo error del primitivo:

| `LocateErrorKind` | Causa |
|---|---|
| `pointer_broken` | REQ-24.7, caso 1: puntero ilegible, destino inexistente o sin `config` ni `specs` |
| `config_unloadable` | REQ-24.7, caso 2: `axiom.yaml` del destino ilegible o con la configuración base inválida |
| `specs_repository_missing` | REQ-24.7, caso 4: `workspaceRoot` inexistente (paso 8) |
| `ambiguous_planning_home` | REQ-24.6: `<ruta>/openspec` existe y no es el planning home del puntero |

El caso 3 de REQ-24.7 (un `axiom.yaml` directo no cargable) no tiene `LocateErrorKind`, porque no rechaza, y no se aplica a un puntero. `start_invalid` no aparece porque `ResolveOwnPointer` solo actúa sobre un puntero existente dentro de un directorio existente. A esa lista se suma el rechazo propio `*hub.ForeignPlanningHomeError` cuando el puntero resuelve otro planning home.

**Comparación de planning homes.** REQ-2.2 compara por ruta y no exige que existan `<ruta>/openspec` ni el planning home resuelto. `SamePlanningHome(a, b)` usa `pathidentity.SameDirectory` y, cuando devuelve falso porque alguna de las dos rutas no existe, compara las rutas absolutas limpias, sin distinguir mayúsculas en Windows. Ese respaldo léxico queda fuera de la política de `internal/pathidentity`, que pide declararlo donde se use. Aquí es aceptable porque las dos rutas se derivan léxicamente del mismo `<ruta>` y de su configuración antes de que exista nada que comparar por identidad.

**Validación de la estructura (REQ-2.2, punto 4).** En los casos 1 y 2, `init` valida el `axiom.yaml` que encuentra con el mismo validador de carga que `LoadConfig`, `Locate` y `workspace validate` (`parseLayered` y la validación estricta de D-04): configuración base, `baseline:` y `technical_context:`. Hoy `init` no valida el `axiom.yaml` existente.

- Informa de cada violación en su salida y en `InitResult.Notes`, con la continuación `axiom workspace validate --path "<ruta>"`. Las reglas que dependen del sistema de ficheros siguen en `workspace validate`.
- La validación no cambia el código de salida: REQ-24.23 sitúa `init` entre las operaciones que completan el resto de su resultado con `0`.
- Su efecto en la escritura lo decide la consulta. Con `baseline:` válido (`project`) o inválido (`indeterminate`), `init` no crea `openspec/specs/` ni `openspec/INDEX.md` (REQ-24.18, REQ-24.23). Un `technical_context:` inválido o un error de la configuración base solo se informan.

**Alternativas descartadas.**

- *Guardas solo en `cmd/axiom`.* `knowledge.RunSweep` debe completar el análisis y omitir solo la siembra, así que la decisión tiene que estar dentro de la operación. Además, el dashboard y la TUI llaman a livingdoc sin pasar por `cmd/axiom`.
- *Una consulta por paquete.* Daría respuestas divergentes para un mismo `axiom.yaml`, lo que prohíbe REQ-24.22.
- *Que `init` rechace también cuando un ancestro resuelve otro planning home* (la lectura de la revisión anterior). Haría regresar `init` en subdirectorios, y REQ-2.2 lo excluye de forma explícita.
- *Un procedimiento propio de `init` que omita alguna causa del primitivo (por ejemplo, `specs_repository_missing`).* Daría a `init` una resolución distinta de la del resto de verbos; REQ-2.2 pide rechazar con el mismo error del primitivo ante cualquier causa de REQ-24.7. El caso normal de un `multirepo` con `specs_repository: openspec` antes de inicializar ya no rechaza, porque el paso 8 no exige el planning home.
- *Que el paso 8 exija `<maestra>/<specs_repository>` también cuando termina en `openspec`.* Exigiría que existiera el planning home, y `init` rechazaría precisamente el caso que debe inicializar.
- *Que `init` falle ante una declaración inválida.* Impediría registrar proyectos existentes por errores que `workspace validate` ya reporta, y REQ-24.23 sitúa `init` entre las operaciones que completan su resultado con `0`.

**Justificación.** Una función y un tipo de error únicos hacen que los creadores coincidan por construcción. Que el tipo sea distinguible con `errors.As` permite a cada canal (CLI, HTTP, TUI) presentar el rechazo de la forma adecuada.

### 2.7 D-07 — Consumidores de catálogo con una sola convención

**Decisión.**

**Aviso de ubicación divergente (Q14), común a todos.** `Location.LegacySpecsDivergence() (string, bool)` devuelve el aviso cuando la topología es `monorepo-embedded`, `specs_repository` está fuera de {`""`, `.`, `openspec`} y `<raíz>/<specs_repository>/specs` existe y no es el mismo directorio que `<planning home>/specs`. Lo emiten todos los consumidores de catálogo (REQ-24.10): livingdoc en `LivingCatalog.Warnings` (y de ahí `axiom archive list` y `show`, el JSON del dashboard y el mensaje de la TUI) y knowledge en las `notes` de `sweep` y `query`. Todos leen solo `<planning home>/specs`.

**livingdoc.** `NewService(start, ...)` conserva su firma y resuelve con `Locate` en cada llamada pública, para que el dashboard y la TUI, que viven mucho tiempo, no trabajen con una raíz obsoleta.

- `specsRoot = <planning home>/specs` e `indexPath = <planning home>/INDEX.md`, con independencia de `specs_repository`.
- Con baseline del proyecto, `GetCatalog` no escanea y devuelve `Specs: []` más `ExternalBaseline` (`json:"external_baseline,omitempty"`, con `documents[].domain` y `documents[].path`).
- `GetSpecDetail` de un dominio declarado devuelve `*ExternalBaselineError` con su ruta. El dashboard ya responde 404 a cualquier error de detalle, que es un 4xx con el mensaje (REQ-4.1).
- `Synthesizer.ColdStart` recibe el `workspaceRoot` ya resuelto.
- Si `Locate` falla, los métodos devuelven ese error.

**knowledge.** Cuando `SpecsRoot` está vacío, `RunSweep`, `RunQuery` y `GenerateCrawlPlan` lo toman de `Locate(WorkspaceRoot).PlanningHome`. El `WorkspaceRoot` de knowledge sigue siendo la raíz de código que se analiza; las dos raíces ya están separadas en el modelo de inc-23.

**dashboard.** Accesor nuevo `(*Service).planningRoot() (string, error)` que resuelve `Locate(rootPath)` en cada llamada. Sustituye a `getRootPath()` en las cinco rutas de `openspec/...`: `GetIncrements`, `FindIncrementPath`, `CreateIncrement`, `MigrateIncompleteTasksToCumulative` (el acumulativo pasa a `<planning home>/changes/cumulative-<rol>`) y el texto de `CreateHandoff`. `rootPath` sigue siendo la ruta registrada en el Hub (la carpeta maestra en la topología canónica). `runUI` considera también el puntero al decidir si el directorio está configurado.

**TUI.** `loadSDDIncrements`, `loadMultiRoleState`, `loadActiveHandoff` y `loadLivingDocs` usan `Locate(".")`. Con baseline externa, `GovernanceMessage` muestra los dominios y rutas declarados. El rechazo de la tecla `s` ya llega con su continuación porque procede de `livingdoc.Service.Sync`.

**Alternativas descartadas.**

- *Mantener `<specs_repository>/specs` en livingdoc.* Conserva la divergencia con el motor que Q14 resuelve a favor del motor.
- *Calcular el aviso Q14 en cada consumidor.* Reproduciría la divergencia de convenciones que inc-24 elimina.
- *Cachear `Location` en el constructor.* Deja una raíz obsoleta en procesos largos.
- *Indexar la baseline externa.* Queda fuera de alcance (REQ-24.25).

**Justificación.** Con una sola convención y un aviso calculado en un solo sitio, REQ-24.10 se cumple igual en los cuatro consumidores. Los campos nuevos de JSON son aditivos y en `snake_case`, coherentes con `LivingCatalog`.

### 2.8 D-08 — Composición con perfil en un paquete solo-fork

**Decisión.** La composición con perfil vive en el paquete nuevo `internal/specbaseline`. `ComposeOpenSpecCanonicalSpecWithOptions` y el resto de `openspec_archive_compose.go` no se modifican. La CLI `archive-compose` elige entre los dos motores:

- sin baseline declarada, o con `--canonical`, motor nativo;
- con documento con perfil, `specbaseline.ComposeDocument`;
- con documento de prosa, rechazo (REQ-24.47).

Los cuatro parsers de `### Requirement:` (compose nativo, `livingdoc/indexer.go`, `knowledge/query.go`, `knowledge/index.go`) no cambian. El parser con perfil es nuevo y solo lo usa `specbaseline`.

**Alternativas descartadas.**

- *Un `ComposeOptions.Heading *HeadingProfile` dentro de `sddstatus`* (el esbozo de la exploración y de C1 en la propuesta). Modifica un fichero compartido con upstream y sometido al ratchet, y R1 dependería solo de las pruebas.
- *Un parser compartido para los cuatro consumidores.* Toca código de upstream (compose) y de inc-23 (knowledge) sin necesidad, porque los catálogos no indexan la baseline del proyecto (REQ-24.25), y aumenta el riesgo de R1.

**Justificación.** R1 se cumple por construcción: los 10 tests de compose y los 3 de la CLI siguen ejercitando código sin cambios. La duplicación de unas 40 líneas de detección de vallas se acepta y se documenta; la unificación queda para un cambio posterior.

### 2.9 D-09 — Modelo de bloque, cola contigua y cuerpo del delta

**Decisión.**

- **Encabezados ATX.** Una línea es un encabezado ATX si, fuera de vallas (` ``` ` y `~~~`, con la misma regla de apertura y cierre que el compose nativo), tiene entre 0 y 3 espacios iniciales, de 1 a 6 `#` y después un espacio o el fin de línea. El título se obtiene sin los `#` de cierre opcionales. Los encabezados setext no se reconocen (limitación documentada, R33).
- **Encabezado de requisito.** Encabezado ATX de nivel `level` cuya línea completa, sin EOL, cumple `heading`.
- **Bloque de requisito.** Desde el encabezado hasta la siguiente línea ATX de nivel ≤ `level`, fuera de vallas (REQ-24.41).
- **Cola contigua.** Los encabezados no-requisito de nivel `level` que siguen directamente al bloque, hasta el siguiente requisito de ese nivel o el siguiente encabezado de nivel < `level`. MODIFIED y REMOVED no los tocan, y cada operación MODIFIED o REMOVED sobre un requisito con cola emite un aviso por sección (encabezado y línea) por la salida de error y en la traza. Encaja con el spike: 97 h3 no-requisito justo después de un requisito y 12 entre dos.
- **Cuerpo del delta.** Si el cuerpo de un MODIFIED o ADDED, ya sin notas y, con `omit`, sin escenarios (D-10), contiene, fuera de vallas, un encabezado ATX de nivel ≤ `level`, integrarlo íntegro cortaría el bloque y la siguiente composición leería el resto como una cola. Se detecta en tres capas, de la más temprana a la última:
  1. **Declaración** (D-04, REQ-24.60): `scenarios: include` exige `level` ≤ 3, así que los `#### Scenario:` del delta nunca chocan con el nivel del requisito.
  2. **Fase spec**: `axiom sdd integrate check` analiza los cuerpos de todos los deltas con perfil antes de la compuerta `spec`, sin exigir las precondiciones de git ni la autorización de rutas externas, y sale con código ≠ 0 si encuentra un encabezado conflictivo. Con baseline del proyecto con perfil de requisito, `sdd-spec` lo ejecuta antes de dar por terminada la fase y corrige lo que reporte (REQ-24.54).
  3. **Archive** (`prepare` y `archive-compose`): repite la comprobación como última guarda y rechaza antes de escribir ningún documento (caso (e) de REQ-24.56).

  Las detecciones de los puntos 2 y 3 nombran el dominio, el requisito, el encabezado infractor con su línea y la continuación: rebajar el encabezado a un nivel mayor que `level` o convertirlo en texto. La integración nunca rebaja ni reescribe esos encabezados. Con esto, todo cuerpo que llega a archive tras pasar `check` se integra íntegro (REQ-24.42).

**Alternativas descartadas.**

- *Absorber las colas en el bloque.* Un MODIFIED borraría secciones independientes, que es el defecto actual del compose nativo con `###` no-requisito.
- *Una regla configurable para la cola.* Añade superficie sin datos que la justifiquen; el aviso permite revisar caso a caso.
- *Re-nivelar el cuerpo del delta.* Altera bytes en silencio.
- *Comprobar el cuerpo solo en archive.* Con `level` ≥ 4 y el `include` por defecto, todos los deltas con escenarios fallarían al final del ciclo (H3).

**Justificación.** La frontera por nivel incluye los 119 h4 del documento de requisitos del spike. El aviso hace visibles las 97 colas sin decidir por el autor, y la detección en tres capas preserva la estructura sin sorpresas al archivar.

### 2.10 D-10 — Formato del delta, render y escenarios

**Decisión.**

- **Formato del delta.** Es el nativo (REQ-24.42). Con perfil, el nombre tras `### Requirement:` se reconstruye como `strings.Repeat("#", level) + " " + nombre`; esa línea debe cumplir `heading`, y de ella se extraen `id` y `title` (REQ-24.39). Si no la cumple, la operación se rechaza nombrando el requisito y el patrón.
- **Encabezado integrado.** `render` sustituye `{id}` y `{title}`. En ADDED, `{id}` es el ID definitivo.
- **Notas.** Las líneas completas `(Placement: ...)`, `(Previously: ...)`, `(Reason: ...)` y `(Migration: ...)` se eliminan del cuerpo antes de integrar.
- **`scenarios: omit` (REQ-24.61).** Se elimina cada fragmento que empieza en `#### Scenario:` y termina antes del siguiente encabezado de nivel ≤ 4, fuera de vallas, o al final del cuerpo. Un `#### Notas` que no sea escenario se conserva. La entrada de la traza registra `scenarios_omitted: true` solo si el requisito tenía escenarios. REMOVED no cambia, y la baseline por defecto siempre los incluye porque usa el motor nativo.
- **Identidad `text`.** Las cuatro operaciones se comportan como el motor nativo tomando el título como nombre. ADDED se inserta tras el último bloque de requisito y su cola, salvo que se indique `(Placement: ...)`.
- **Identidad `id`.**
  - RENAMED se rechaza indicando que se use MODIFIED para cambiar el título y que renumerar no está soportado.
  - MODIFIED puede cambiar el título, pero no el `id`.
  - REMOVED con `--supersede` conserva el bloque con el encabezado renderizado más ` [SUPERSEDED / DEPRECADO]` y la misma nota `> [!CAUTION]` del motor nativo. Su motivo pasa por la sustitución de IDs (REQ-24.59).
  - `--superseded-requirements` recibe IDs (identidad `id`) o títulos (identidad `text`).

**Alternativas descartadas.**

- *Un formato de delta propio del proyecto* (Q6). Las skills, los pins y el compose nativo ya hablan el formato nativo.
- *Integrar los escenarios como texto plano.* Rompe el «cuerpo íntegro» con `include`.

**Justificación.** El delta es el mismo en las dos baselines, así que `sdd-spec` cambia poco. El render por plantilla respeta el formato del proyecto, y `omit` evita imponer el formato de escenarios OpenSpec (D1).

### 2.11 D-11 — Ubicación de ADDED, asignación y sustitución de IDs

**Decisión.**

**Ubicación (REQ-24.43).** Es obligatoria con identidad `id`:

- `(Placement: after <ID>)` inserta el bloque tras el bloque de `<ID>` y su cola contigua.
- `(Placement: section "<título>")` lo inserta al final de la sección de nivel `level-1` con ese título, que debe ser único entre los encabezados de ese nivel; el final es antes del siguiente encabezado de nivel ≤ `level-1`, o el fin del fichero.

Los ADDED con la misma ancla se insertan en el orden del delta, cada uno tras el anterior. La ubicación se evalúa sobre el documento después de RENAMED, MODIFIED y REMOVED (el mismo orden del motor nativo). Un `<ID>` duplicado, inexistente o eliminado en el mismo delta sin `--supersede` provoca rechazo, igual que un título ausente o repetido.

**Familia y asignación.**

- La familia de un ID es el prefijo previo a la última serie de dígitos (`RF-` en `RF-123`). Un ID sin sufijo numérico detiene el archivado (REQ-24.59 a).
- El máximo de la familia se calcula sobre los encabezados de requisito del documento original, antes de aplicar ninguna operación y contando los duplicados.
- Los definitivos son máximo+1, máximo+2... en el orden del delta. Se rellenan con ceros hasta el ancho del ID de mayor número.
- Una familia sin ningún ID en el documento destino detiene el archivado (REQ-24.59 b) con esta continuación, que responde a la nota (b) de la compuerta `spec`: «siembra la numeración añadiendo al documento, en un commit propio del repositorio de especificaciones, el primer requisito de la familia con su ID inicial (por ejemplo `NFR-001`, con el relleno de ceros que heredará la familia), o cambia el ADDED a una familia existente; después ejecuta `axiom sdd integrate check --cwd "<workspaceRoot>" --change <cambio>`».

**Correspondencia única (REQ-24.59).** `specbaseline.PlanIDs` la calcula una sola vez para todo el cambio, recorriendo los dominios con perfil en el orden de `baseline.documents` y los ADDED en el orden de cada delta. Forma parte de la validación previa a cualquier escritura. Detiene sin escribir si:

- un provisional coincide con un ID existente del documento (REQ-24.43);
- dos ADDED del cambio declaran el mismo provisional (c);
- un provisional coincide con el definitivo de otro requisito del cambio (d).

Que un provisional coincida con su propio definitivo se admite.

**Sustitución.** `IDMap.Substitute` reemplaza tokens completos. Un token es una aparición exacta, sensible a mayúsculas, ni precedida ni seguida por letra Unicode, dígito, `_` o `-`. Se aplica:

- en el compose, a encabezados, cuerpos y nota de motivo de `--supersede`;
- en `integrate prepare`, a los textos de los elementos de trabajo dirigidos y de contexto que recibe el agente. El agente recibe el texto ya sustituido y, como referencia, la tabla de correspondencias;
- nunca al texto preexistente de los documentos ni a los ficheros del cambio.

**Rango provisional fuera de rango** (nota (a) de la compuerta `spec`). `integrate check` sugiere, por familia y documento, el inicio provisional: el primer candidato de la forma `9·10^d + 1`, con `d = max(3, dígitos(máximo + K))` y `K` el número de ADDED de la familia, cuyos IDs `prefijo + número` cumplan `heading`. Si el patrón fija el ancho (por ejemplo `\d{3}`), se prueba también `9·10^(ancho-1)`. Si ningún candidato cumple, `check` lo reporta. `sdd-spec` instruye usar ese inicio y volver a ejecutar `check` tras redactar los deltas. Ejemplo: el spike registra 511 encabezados de requisito `RF` con 490 IDs distintos y un máximo de `RF-622`, con huecos en la numeración. Con menos de 378 ADDED de la familia, máximo + K < 1.000 y el inicio sugerido es `RF-9001`, que además queda por encima de los IDs fuera de secuencia (5.454 a 6.237) que citan los incrementos en curso del flujo antiguo.

**Rama por detrás del upstream** (nota del orquestador). `integrate prepare` ejecuta `git rev-list --count HEAD..@{upstream}` sobre el repositorio de especificaciones, solo con referencias locales. Si la rama va por detrás, emite un aviso no bloqueante que indica que el dato procede de referencias locales sin `git fetch` y recomienda integrar esos commits (`git pull --ff-only`) antes de archivar. Sin upstream o con HEAD desacoplado, emite una nota.

**Alternativas descartadas.**

- *Insertar por orden de ID.* El spike encontró 23 transiciones descendentes.
- *Que el agente sustituya los IDs.* No es determinista; la verificación (e) quedaría como único control.
- *Una opción para sembrar familias en Axiom.* La especificación pide detener; la siembra es normalización del proyecto, como los duplicados.
- *Hacer `fetch` antes de asignar.* Es una operación remota sin autorización.

**Justificación.** La asignación en archive evita las colisiones que el spike atribuye a dos flujos que asignaban los mismos IDs siguientes. Pre-sustituir el texto convierte la sustitución en una propiedad nativa, y la verificación (e) queda como red.

### 2.12 D-12 — Fidelidad de bytes y validación completa antes de escribir

**Decisión.**

**Modelo de documento.** Se detecta y retira el BOM UTF-8 al leer y se vuelve a anteponer al escribir si existía; nunca se añade. Cada línea conserva su EOL (`\r\n`, `\n`, o vacío en la última línea sin salto). Los bloques sustituidos usan el EOL de la línea de encabezado que sustituyen, y los insertados el del encabezado de referencia (`<ID>` o la sección). Las líneas del delta, sea cual sea su EOL, se emiten con ese EOL.

**Separación.** Un bloque sustituido conserva la serie de líneas en blanco finales del bloque original. Uno insertado hereda la del bloque de referencia (mínimo una línea, salvo al final del fichero). Si se inserta tras una última línea sin salto, antes del bloque nuevo se añade el EOL del documento, que es el único byte añadido fuera de un bloque editado (documentado). Si el documento no termina en salto y el último bloque no se edita, el final se conserva (REQ-24.45).

**Todo o nada (REQ-24.46).** `integrate prepare` valida y compone en memoria todas las operaciones de todos los documentos con perfil, la correspondencia de IDs y el delta de contexto antes de escribir nada en la baseline. La escritura posterior (D-13) comprueba las huellas de todos los documentos antes de escribir el primero, escribe cada uno con un fichero temporal en el mismo directorio y un renombrado, y si un renombrado falla restaura desde memoria los documentos ya renombrados. La edición dirigida y la del contexto solo empiezan después de que la escritura de composiciones haya salido con `0`.

**`core.autocrlf`.** El proyecto de referencia trabaja en CRLF con el índice en LF. Por eso nunca se comparan bytes de un commit con bytes del árbol de trabajo:

- las huellas del plan se calculan sobre el árbol de trabajo;
- el estado limpio y el alcance se obtienen de git (`status` y `diff` contra un commit), que ya normaliza los finales de línea;
- la comprobación de IDs provisionales (D-15 e) cuenta tokens, una medida que no depende del EOL.

**Alternativas descartadas.**

- *Normalizar a LF al componer.* Rompe REQ-24.45.
- *`git apply` con parches* (modo E4 de la exploración). Las specs se escribirían sin ver el documento final.

**Justificación.** El modelo por líneas con EOL propio de cada línea reproduce los documentos CRLF, con BOM y mixtos que describe el spike. La validación en memoria resuelve el «todo o nada» sin transacciones del sistema de ficheros.

### 2.13 D-13 — Protocolo de integración por cambio

**Decisión.** Verbo nuevo `axiom sdd integrate` (adaptador `internal/cli/sdd_integrate.go`, lógica en `internal/specbaseline`) y modo de cambio en `archive-compose`. Hay tres flujos de archive, según lo que proyecte el status.

**Flujo 1: baseline del proyecto** (el status proyecta `baseline`).

| Paso | Ejecuta | Qué hace | Escribe |
|---|---|---|---|
| 1 | `axiom sdd integrate prepare --cwd "<ws>" --change <c> [--supersede] [--superseded-requirements L]` | Valida la declaración, el almacén efectivo (REQ-24.62), los dominios (REQ-24.48), las precondiciones git (REQ-24.49), la autorización de edición de rutas mantenidas fuera del repositorio de especificaciones (REQ-24.37, con `allowedEditRoots` del status), los cuerpos de los deltas (D-09), las composiciones en memoria, la correspondencia de IDs y el delta de contexto con sus anclas. Registra el `HEAD` y la foto de suciedad de cada repositorio afectado y avisa de la rama por detrás | Solo `<changeRoot>/integration.yaml`, que es el plan |
| 2 | El comando que imprime el paso 1, sin cambios: `axiom sdd archive-compose --cwd "<ws>" --change <c> --document d1 [--document d2 ...] --in-place` | Comprueba que la lista coincide exactamente con los dominios `compose` del plan y que todos los documentos están en su huella `before`; compone, comprueba que el resultado coincide con la huella `after` y escribe todos | Documentos con perfil y estado `integrated` en el plan |
| 3 | Agente (`sdd-archive`) | Edición dirigida de los documentos de prosa `directed` (la sección del cambio) y, si se declaró, del contexto mantenido, con los textos ya sustituidos del paso 1 | Documentos `directed` y rutas mantenidas |
| 4 | `axiom sdd integrate verify --cwd "<ws>" --change <c>` | Verificación nativa E2 (D-15). Si pasa, marca las entradas dirigidas y de contexto como `integrated` y registra `created` | Solo `integration.yaml` |
| 5 | `Step 3` de la skill (sin cambios) | Mover el cambio a `openspec/changes/archive/YYYY-MM-DD-<c>` | El archivo |
| 6 | `axiom sdd integrate show --cwd "<ws>" --change <c> --format markdown >> "<archivado>/archive-report.md"` | Añade la traza al informe de forma mecánica (REQ-24.52). Encuentra el cambio ya archivado con `kickoff.ArchivedChangeRoot` | El informe |

**Flujo 2: baseline por defecto con contexto mantenido** (el status proyecta `technicalContext.maintained` y no `baseline`).

1. `integrate prepare`: foto de cada repositorio, validación del delta de contexto y sus anclas, autorización. No hay composiciones con perfil.
2. Los bloques fijados del `Step 2` de la skill componen `openspec/specs/<dominio>/spec.md` con el motor nativo.
3. `axiom archive sync --path "<ws>"` regenera `<planning home>/INDEX.md` (REQ-21.17(4)).
4. El agente edita el contexto mantenido.
5. `integrate verify`; después, mover, informe y `show`, como en el flujo 1.

**Flujo 3: baseline por defecto sin contexto mantenido.** Es el flujo actual (bloques fijados del `Step 2`, `Step 3` y el informe) más el paso `axiom archive sync --path "<ws>"` entre el `Step 2` y el `Step 3`.

**`INDEX.md` y roadmaps (REQ-21.17(4)).** Con la baseline por defecto, `openspec/INDEX.md` lo produce `axiom archive sync`, el productor nativo existente (`livingdoc.Indexer.SyncIndex`), invocado por la skill en el punto indicado. Hoy ninguna skill lo ejecuta y el manual lo atribuye por error a `archive-compose`; se corrige el manual. Ni `axiom.yaml` ni `openspec/config.yaml` permiten declarar roadmaps, así que archive no actualiza ninguno: la cláusula no tiene objeto declarado, y declarar roadmaps queda para un cambio posterior. Con baseline del proyecto no hay `INDEX.md` (la guarda de D-06 rechazaría la sincronización).

Modos de `archive-compose`:

| Modo | Flags | Uso |
|---|---|---|
| Nativo (sin cambios) | `--canonical --delta [--output] [--supersede] [--superseded-requirements]` | Baseline por defecto; bloques fijados de la skill |
| Documento (REQ-24.47) | `--cwd --document <d> --delta <ruta> [--output] [--supersede] [--superseded-requirements]` | Previsualización o uso suelto. Sin baseline declarada, compone `<planning home>/specs/<d>/spec.md` con el motor nativo. Con perfil, calcula la correspondencia del cambio si el delta está en `<planning home>/changes/<c>/specs/<d>/`, y si no, la del delta suelto. Si ese cambio tiene plan y registra el documento como integrado, o el documento coincide con su huella `after`, rechaza como ya integrado (REQ-24.43) y nombra la correspondencia |
| Cambio | `--cwd --change <c> --document <d>... --in-place` | Paso 2 del flujo 1. Exige plan y toma de él las opciones de supersede |

Reglas de flags: `--document` y `--canonical` son excluyentes (REQ-24.47). `--in-place` exige `--change` y excluye `--output`, `--delta` y las opciones de supersede. Con `--cwd`, las rutas relativas se resuelven contra `workspaceRoot`. Un dominio de prosa en `--document` se rechaza indicando que se integra por edición dirigida o solo registro. Un dominio no declarado se rechaza (REQ-24.48) nombrando la continuación: declararlo en `baseline.documents` o retirar la delta.

**Reentrada y reaplicación.** El plan guarda `inputs_sha256` (huella de los deltas, del delta de contexto y de la instantánea de la declaración), las huellas `before` y `after` de cada documento con perfil y el `HEAD` de cada repositorio afectado.

- `prepare` con un plan existente reemite el mismo plan si las entradas y las huellas coinciden.
- Si todas las composiciones ya están integradas, rechaza como ya integrado, nombra la correspondencia (REQ-24.43) y propone `integrate verify` o `show`.
- Si las entradas cambiaron, rechaza y nombra la continuación: devolver los documentos con `git checkout -- <documentos>`, retirar `integration.yaml` y volver a preparar.
- `archive-compose --in-place` sobre un documento ya integrado rechaza como ya integrado y nombra `axiom sdd integrate verify`.

**Límite de la foto de suciedad.** Si un repositorio afectado tiene más de 1.000 entradas sucias (`MaxDirtyEntries`), `prepare` rechaza sin escribir y nombra la continuación: confirmar, guardar (`git stash -u`) o descartar los cambios ajenos y volver a ejecutar `axiom sdd integrate prepare`. El límite acota el tamaño de `integration.yaml`.

**`integrate check`.** Hace las mismas validaciones que `prepare` salvo las precondiciones git y de autorización, no escribe nada, tampoco en la carpeta del cambio, y sale con código ≠ 0 si hay errores (REQ-24.42). Lo usan `sdd-spec` (deltas, cuerpos, correspondencia, inicio provisional sugerido) y `sdd-design` y `sdd-apply` (sintaxis y anclas del delta de contexto). Un delta de contexto ausente es aviso en `check` y error en `prepare`. Con baseline del proyecto o contexto mantenido y un almacén efectivo `engram` o `none`, `check` y `prepare` reportan la causa de REQ-24.62 (caso (g) de REQ-24.56).

**Alternativas descartadas.**

- *Que `prepare` escriba también las composiciones.* La skill dejaría de invocar `archive-compose --cwd --document`, que es lo que pide el pin de REQ-24.47, y el paso de escritura dejaría de ser idempotente por documento.
- *Una invocación de `archive-compose` por documento, con temporales y `mv` desde la shell.* Cada invocación revalidaría el cambio entero contra documentos ya compuestos, y el modelo tendría que secuenciar N+N pasos.
- *Un trazador escrito por el agente.* No es fiable como registro de auditoría (Mechanical Copy Contract).
- *Un fichero de traza Markdown.* Su análisis es frágil; el YAML con esquema sigue el precedente de `kickoff.yaml` y `gates.yaml`, y `show` genera el Markdown.
- *Dejar `INDEX.md` fuera de la skill.* El cambio no cumpliría el escenario de REQ-21.17 que esta misma propuesta reafirma.

**Justificación.** Una sola invocación nativa escribe todas las composiciones (todo o nada), y la skill sigue invocando `archive-compose --cwd --document` sin plantillar rutas. El plan con huellas y `HEAD` hace verificable cada paso y permite reanudar tras una interrupción sin duplicar requisitos.

### 2.14 D-14 — Actor y modelo de la edición dirigida (R20)

**Decisión** (confirmada por el usuario). La edición dirigida (secciones del cambio en documentos de prosa `directed` y entradas del delta de contexto) la realiza el propio `sdd-archive`. No hay fase ni agente nuevos. La edición queda acotada entre pasos nativos: `prepare` fija qué se edita, dónde y con qué texto; `verify` comprueba presencia, anclas, alcance y IDs.

- **Modelo.** La plantilla perezosa de Claude Code (`internal/assets/claude/sdd-orchestrator-workflow.md`) gana la regla «Archive Directed Integration Model (MANDATORY)». Cuando el status proyecta `technicalContext.maintained` o algún `baseline.documents[].integration == "directed"`, `sdd-archive` se lanza con el modelo asignado a `sdd-apply` (que en los presets actuales es `sonnet`); en otro caso, con el de la fila `sdd-archive`. Claude Code admite `model` por llamada, así que la elevación automática solo existe allí.
- **Otros runtimes.** Donde el modelo va fijado por agente (OpenCode, Kiro y otros), `docs/manual-de-inicio.md` y la convención compartida indican que, al declarar integración dirigida o contexto mantenido, conviene asignar a `sdd-archive` un modelo de nivel de implementación con el selector de modelos existente.
- **Skill.** `sdd-archive` aclara que el Mechanical Copy Contract rige copias y movimientos, y que la edición dirigida usa ediciones acotadas (`Edit`), nunca reescrituras completas de ficheros.

**Alternativas descartadas.**

- *Subir `sdd-archive` a `sonnet` en todos los presets.* Cambia el coste y el comportamiento para todos los proyectos, también los que no declaran nada, diverge de upstream en `internal/model/claude_model.go` y regenera muchas tablas.
- *Una fase o agente nuevos (`sdd-integrate`).* Afectaría a las definiciones de agentes de 8 runtimes, a las tablas de modelos, al grafo de dependencias, a la enumeración de `nextRecommended` y a muchos goldens.
- *Delegar la edición en un escritor genérico del orquestador entre dos ejecuciones de archive.* Mezcla el carril orgánico con una fase SDD, cosa que las reglas de delegación prohíben.

**Justificación.** La mayoría de los cambios no tienen trabajo dirigido y siguen en el nivel ligero. Cuando lo hay, la elevación es determinista (sale del status) y el verificador nativo protege el resultado con independencia del modelo. El riesgo que queda en runtimes sin modelo por llamada se declara en §10 (R20).

### 2.15 D-15 — Verificador nativo E2

**Decisión.** `specbaseline.Verify` con git de solo lectura (`internal/specbaseline/gitview.go`). Las invocaciones usan `cmd.Dir` = raíz absoluta obtenida con `git rev-parse --show-toplevel`, `GIT_OPTIONAL_LOCKS=0`, tiempo máximo y límite de salida. Nunca se usa `-C` con rutas relativas.

**Repositorios afectados.** El de `workspaceRoot` (especificaciones), los que contienen rutas mantenidas o de referencia y los repositorios de rol declarados (defensa adicional). Si un repositorio que contiene un documento afectado no es git, el resultado es la parada (a) de REQ-24.49 en `prepare`.

**Línea base por repositorio.** `prepare` registra en el plan, para cada repositorio afectado:

- `head`: el commit de `HEAD` (o vacío si el repositorio no tiene commits, en cuyo caso se compara con el árbol vacío);
- `dirty`: la foto de la suciedad previa, con la ruta y la huella de cada fichero (o borrado), como máximo 1.000 entradas (D-13).

**Diff de archive.** `verify` calcula, en cada repositorio, `git diff --name-status -z --no-renames <head>` (árbol de trabajo contra el commit registrado, con o sin stage) más `git ls-files --others --exclude-standard -z`, y quita las entradas de la foto cuya huella no ha cambiado. Al comparar con el `HEAD` del plan y no con el actual, un commit hecho entre `prepare` y `verify` no puede ocultar ediciones fuera de alcance.

Comprobaciones de REQ-24.51:

| Comprobación | Regla |
|---|---|
| (a) Presencia | Las entradas esperadas se recalculan desde los ficheros actuales del cambio y deben coincidir con las del plan; si cambiaron después de `prepare`, se rechaza. Cada documento con entradas `directed` o de contexto (salvo `remove` y «Sin impacto») debe aparecer en el diff de archive |
| (b) Anclas | Entradas `directed`: exactamente un encabezado cuyo título empieza por el nombre del cambio como token completo. Contexto: `modify` y `add` resuelven exactamente una sección; `remove` no resuelve ninguna. Las composiciones se comprueban por huella `after`, no por ancla, porque son deterministas |
| (c) Alcance | Cada ruta del diff de archive pertenece al conjunto permitido, y la pertenencia se decide por identidad (`os.SameFile` para ficheros, `pathidentity.Contains` para directorios), nunca comparando cadenas. Conjunto permitido: documentos de los dominios del cambio con integración `compose` o `directed`; documentos nombrados en el delta de contexto; carpeta del cambio; y, solo con baseline por defecto, `<planning home>/specs/<dominio>/spec.md` de los dominios del cambio y `<planning home>/INDEX.md`, como fija REQ-24.51 (c). Un fichero borrado fuera de la carpeta del cambio es siempre una violación, porque archive nunca borra documentos. Hay errores explícitos para rutas de `reference`, documentos `register`, rutas mantenidas con «Sin impacto» (contradicción con la decisión registrada, REQ-24.57) y cualquier otro fichero, incluidos temporales huérfanos |
| (d) Traza | `integration.yaml` existe, se decodifica de forma estricta y declara `schema: axiom.sdd-integration/v1` |
| (e) IDs | Por cada documento cambiado y cada provisional `P` con definitivo `D ≠ P`, el número de tokens `P` en el contenido actual no puede superar el del commit registrado (`git show <head>:<ruta>`, vacío si el fichero no existía). Contar en lugar de comparar líneas tolera el reajuste de texto preexistente y no depende del EOL |

**Resultado.**

- Si todo pasa, sale con `0`, marca como `integrated` las entradas dirigidas y de contexto, registra `created` en las entradas `add` que crearon documento y escribe `verified_at`.
- Si algo falla, sale con `1`, lista todas las violaciones (entrada, comprobación, fichero y, en (e), el provisional y su definitivo) y no modifica documentos ni plan. Las ediciones quedan sin confirmar y revertibles con git (REQ-24.56), y el cambio no se mueve.

**Alternativas descartadas.**

- *Comparar con el `HEAD` actual.* Un commit entre `prepare` y `verify` escondería ediciones fuera de alcance (H6).
- *Ignorar la suciedad previa.* Atribuiría al archive los cambios ajenos sin confirmar que REQ-24.49 permite.
- *Comparar rutas como cadenas.* Falla con mayúsculas o enlaces simbólicos en Windows y macOS (H14).
- *Verificación semántica.* Queda fuera de alcance por especificación.
- *Reutilizar el runner de `reviewtransaction`.* Acopla SDD a RDD, cuando SDD nunca lanza RDD.

**Justificación.** Las cinco comprobaciones son deterministas y baratas. El `HEAD` registrado, la foto previa acotada y el recuento de tokens resuelven los falsos negativos y positivos que harían poco fiable el verificador en un repositorio real.

### 2.16 D-16 — Delta de contexto `context-impact.md` y gramática de anclas

**Decisión.** Se mantiene el nombre de trabajo `<changeRoot>/context-impact.md`. Fuera de las vallas, el formato es:

```markdown
# Delta de contexto: <cambio>

## Entrada: context/ARQUITECTURA.md

- Ancla: Componentes > Servicio de pagos
- Acción: modify
- Motivo: se añade el reintento con retroceso exponencial (RF-9001)

<contenido opcional que integrar, hasta la siguiente «## Entrada:» o el fin del fichero>
```

o bien, sin entradas:

```markdown
# Delta de contexto: <cambio>

Sin impacto: el cambio solo corrige una errata en un mensaje de error
```

**Reglas de análisis.**

- `## Entrada: <ruta>`: la ruta es relativa a `workspaceRoot`, con `/` o `\`, y puede ir entre comillas invertidas.
- Le siguen exactamente los campos `- Ancla:`, `- Acción:` (`add` | `modify` | `remove`) y `- Motivo:`. Un campo desconocido o ausente es un error.
- El cuerpo es opcional: el ejemplo no normativo de REQ-24.31 no lo trae. Cuando falta, el agente deriva la edición del motivo y de los artefactos del cambio.
- `Sin impacto: <motivo>` exige un motivo no vacío y no puede convivir con entradas. Un fichero sin entradas ni «Sin impacto» es inválido.
- Las entradas reciben los identificadores `C1..Cn` por orden de aparición, para la traza.
- Un par (documento, ancla) repetido es un error.
- El documento debe estar bajo `maintained.paths` y no bajo `reference.paths` ni en `baseline.documents` (REQ-24.30). En `prepare` la pertenencia se comprueba por identidad (`pathidentity.Contains`), igual que en D-15.

**Anclas (REQ-24.32).** La sintaxis es `Título`, `Padre > Título` o `Abuelo > Padre > Título`, con un ordinal opcional ` #N` en el último elemento.

- Un encabezado coincide si su título es igual al último elemento y los elementos anteriores coinciden con su cadena de ancestros inmediatos, contigua desde el padre hacia arriba.
- `#N` elige la N-ésima coincidencia en orden del documento.
- Más de una coincidencia es un rechazo que lista línea y ruta completa de cada una. Ninguna coincidencia solo es válida con `add`.
- Si el documento no existe, solo `add` es válido, y el documento se crea en archive dentro de `maintained.paths` (REQ-24.55).
- Limitaciones documentadas: títulos que contienen ` > ` y encabezados setext (R33).

**Alternativas descartadas.**

- *Una sección dentro de `design.md`.* `apply` reabriría un `design.md` ya aprobado (REQ-24.34).
- *YAML.* Lo redactan agentes y lo revisan personas en las compuertas; el Markdown con campos fijos es legible y sigue siendo determinista.
- *Anclas por línea.* Son frágiles ante cualquier edición.

**Justificación.** El formato cubre los escenarios de REQ-24.31 y REQ-24.32 con un analizador de unas 150 líneas. Las anclas por ruta de encabezados toleran los 40 títulos repetidos que el spike encontró en `context/`.

### 2.17 D-17 — Skills y convención compartida

**Decisión.** Fichero nuevo `internal/assets/skills/_shared/project-baseline-convention.md`, que el instalador despliega como el resto de `_shared` (README de `_shared`). Concentra:

1. los localizadores `baseline` y `technicalContext` del status, con la regla de no interpretar `axiom.yaml`;
2. la lectura (entrada de cada modo y el resto bajo demanda);
3. las prohibiciones (rutas de referencia; rutas mantenidas solo vía archive; nunca `openspec/specs/` con baseline del proyecto);
4. los deltas para una baseline con perfil (nombre, IDs existentes, provisionales fuera de rango, `(Placement: ...)`, cuerpos sin encabezados de nivel ≤ `level`, `axiom sdd integrate check`);
5. el formato de `context-impact.md`;
6. la remediación de compuertas por el delta de contexto (D-18);
7. la regla de modelo de D-14 para runtimes sin modelo por llamada.

Cada skill gana pocas líneas, con una frase fija que pueda fijar un pin:

| Skill | Cambio | Requisito |
|---|---|---|
| `_shared/sdd-phase-common.md` | Sección G: las rutas `openspec/...` y cualquier ruta relativa de los artefactos se resuelven contra `actionContext.workspaceRoot`; los bloques shell se ejecutan con `cd "<workspaceRoot>" && …` en la misma invocación; en multirepo, el código se nombra con `../<repo>/...` o con ruta absoluta | REQ-24.11, R17, R30 |
| `_shared/sdd-status-contract.md` | Documenta `baseline` y `technicalContext` como campos opcionales, con la regla de leerlos del status | REQ-24.19 |
| `_shared/openspec-convention.md` | Fila de `sdd-init` condicionada; filas nuevas de `context-impact.md` e `integration.yaml` en el árbol | REQ-24.24, D-13, D-16 |
| `sdd-init` (+ `references/init-details.md`) | Si el status proyecta `baseline`, no se crea `openspec/specs/` | REQ-24.24 |
| `sdd-explore`, `sdd-propose`, `sdd-spec` (y, como DEBERÍA, `sdd-onboard`) | Leen la baseline de `baseline.documents`; `openspec/specs/` solo sin `baseline`. `sdd-spec` añade las reglas de IDs y, con baseline del proyecto con perfil de requisito, la frase fija que ordena ejecutar `axiom sdd integrate check --cwd "<workspaceRoot>" --change <cambio>` antes de dar por terminada la fase y corregir las infracciones que reporte | REQ-24.54, REQ-24.42 |
| `sdd-explore`, `sdd-design`, `sdd-apply`, `sdd-verify` | Leen la `entry` de cada modo de `technicalContext` y el resto bajo demanda | REQ-24.29 |
| Las 8 skills de fase, incluida `sdd-tasks` | «Never create, modify, or delete files under `technicalContext.reference` paths.» `sdd-tasks` no gana nada más (D-22) | REQ-24.30 |
| `sdd-design` | Con contexto mantenido, redacta `context-impact.md` (impacto previsto o «Sin impacto» razonado) y no edita el contexto. Al remediar una compuerta `design` rechazada por el delta, añade la línea de remediación de D-18 | REQ-24.33, REQ-21.12 |
| `sdd-apply` | Mantiene `context-impact.md` al día sin reabrir `design.md`; un solo delta por cambio aunque haya varios roles. Al remediar una compuerta `role-apply` rechazada por el delta, añade la línea de remediación de D-18 | REQ-24.34, REQ-21.12 |
| `sdd-verify` (+ `references/report-format.md`) | Aviso no bloqueante con la lista cerrada de patrones de REQ-24.35 (más `CMakeLists.txt`) en una sección «Avisos de contexto»; nunca toca `nextRecommended` ni las dependencias | REQ-24.35 |
| `sdd-archive/SKILL.md` | (1) Rama nueva al inicio del `Step 2`: con `baseline` o `technicalContext.maintained` proyectados, seguir `references/project-baseline-integration.md`; si el status marca el archivado como bloqueado o `integrate prepare` rechaza (incluido el almacén sin ficheros, caso (g) de REQ-24.56), parar como `blocked` con la causa literal, sin deducir el almacén por su cuenta. (2) En el propio `SKILL.md`, la forma de la invocación de composición, `axiom sdd archive-compose --cwd "<workspaceRoot>" --change "{change-name}" --document "<dominio>" [--document ...] --in-place`, con la instrucción de ejecutar literalmente el comando que imprime `prepare`; ese texto lleva el pin de REQ-24.47 y no contiene `openspec/specs/{domain}`. (3) En la rama por defecto, un paso nuevo `axiom archive sync --path "<workspaceRoot>"` entre la copia de specs y el `Step 3` (REQ-21.17(4)). (4) Las prohibiciones de REQ-24.24 | REQ-21.17, 24.24, 24.47, 24.55–24.57 |

Los bloques shell fijados y ejecutados por `assets_test.go` (`#### If Main Spec Does NOT Exist`, `### Step 3: Move to Archive`, la recuperación histórica) **no cambian**. El paso de `INDEX.md` es un subapartado nuevo, con su propio bloque, situado tras el bloque de copia y antes de `### Step 3`: las pruebas actuales solo comprueban la presencia y el orden de fragmentos dentro de ese tramo, así que siguen pasando. La sección G se aplica con una instrucción en prosa previa, no editando los bloques fijados.

**Alternativas descartadas.**

- *Detalle completo en cada `SKILL.md`.* Aumenta la divergencia con upstream y el presupuesto de cuerpo (`sdd-archive` ya ocupa 395 líneas).
- *El pin de REQ-24.47 solo en `references/`.* El escenario de REQ-24.47 inspecciona el texto de la skill.
- *Introducir `cd` en los bloques fijados.* El arnés de pruebas los ejecuta con marcadores que no sustituiría.

**Justificación.** Una sola convención compartida es más fácil de mantener, y las frases fijas por skill satisfacen los pins de REQ-24.29, REQ-24.30, REQ-24.47 y REQ-24.54 sin duplicar el contenido.

### 2.18 D-18 — Presentación y remediación de las compuertas `design` y `role-apply`

**Decisión.**

**Presentación.** Se añade una frase a la sección compartida `SDD Change Kickoff and Block Gates` de `_shared/sdd-orchestrator-sections.md`, que se compone en las 12 plantillas de orquestador:

- con `technicalContext.maintained` en el status, la compuerta `design` presenta `<changeRoot>/context-impact.md`, evalúa su coherencia con el diseño y señala de forma explícita si falta, sin asumir «Sin impacto»;
- cada compuerta `role-apply` presenta el delta actualizado y evalúa que refleja lo implementado por ese rol.

El sobre nativo `gateQuestion` (`gateCriteria`, `gateEvidence`) **no cambia**.

**Remediación (REQ-21.12).** El digest de la compuerta `design` cubre solo `design.md`, y el de `role-apply` solo el fichero de tareas del rol (`cli/sdd_gate_digest.go`). `resolveGateStatus` (`kickoff/machine.go`) solo devuelve un rechazo a `pending` cuando ese digest cambia. Si una compuerta se rechaza solo por el delta de contexto, corregir `context-impact.md` no basta para reabrirla. El procedimiento documentado es:

1. La fase que remedia (`sdd-design` o `sdd-apply`) corrige `context-impact.md`.
2. Añade una línea fechada a la sección `## Registro de remediación` del artefacto que juzga la compuerta (`design.md` para `design`, el fichero de tareas del rol para `role-apply`), con el motivo del rechazo y lo que cambió en el delta. La línea nunca es una casilla de tarea, así que no altera el progreso.
3. El digest cambia, la compuerta vuelve a `pending` con `Reopened=true` y el orquestador la presenta de nuevo, ahora con el delta corregido.

El procedimiento figura en la convención compartida, en `sdd-design` y `sdd-apply` (D-17) y en la frase de la sección compartida del orquestador.

**Alternativas descartadas.**

- *Añadir `context-impact.md` al digest de esas compuertas.* El digest se calcula de forma idéntica en tres sitios (`kickoff/machine.go`, `cli/sdd_gate_digest.go` y `sddstatus/governance.go`) y lo fija `sdd_gate_artifact_digest_test.go`. Hacerlo depender de `axiom.yaml` acoplaría el código de compuertas a las declaraciones del workspace y cambiaría el digest de los registros ya existentes, de modo que los rechazos vigentes se reabrirían solos. Además, una aprobación es terminal con cualquier digest, así que no protegería más.
- *Un verbo de reapertura manual.* Contradice D-08 de inc-21, que satisface REQ-21.12 sin verbo de reapertura.
- *Añadir criterios y evidencias al sobre nativo.* Cambia un contrato tipado con pruebas propias e introduce la ruta del delta en el status, al borde de REQ-24.28.

**Justificación.** Los escenarios de REQ-21.9 y REQ-21.11 piden un pin sobre las plantillas de orquestador, y la sección compartida cubre todos los runtimes con un único cambio de texto. La remediación documentada cuesta unas pocas líneas de texto y no toca ningún mecanismo con pruebas propias.

### 2.19 D-19 — Regla mínima del cierre de ODD (REQ-24.58)

**Decisión.** En `internal/components/agentguidance/routing.go`, la frase del paso 7 («Close.») gana esta oración en inglés, determinista:

> When the project declares a maintained technical context (`technical_context.maintained` in `axiom.yaml`), a work unit that changes something that context documents updates the affected documentation in that same work-unit commit; documentation declared as reference (`technical_context.reference`) is never modified, and without a maintained context this adds no requirement.

El resto de pasos y el carril SDD no cambian. `routing_test.go` comprueba la frase y que dos renderizados son idénticos.

**Copias generadas.** Ningún golden de `testdata/golden` contiene el paso 7, así que esta regla no regenera goldens. Las copias del texto fuera de `internal/` son:

| Fichero | Estado en git | Tratamiento |
|---|---|---|
| `.codex/AGENTS.md` | Versionado (contiene el paso 7) | Se regenera en la unidad C3b con el binario de la rama, desde la raíz: primero `go run ./cmd/axiom sync --scope workspace --agent codex --dry-run` para ver los destinos y después sin `--dry-run`. Se confirma solo la oración nueva de `.codex/AGENTS.md`; cualquier otro fichero escrito o cualquier otra deriva se deja fuera del commit y se trata como hallazgo aparte |
| `.claude/CLAUDE.md` | Ignorado (`.gitignore`) | No se toca; cada máquina lo regenera con su propia sincronización |
| `.kiro/steering/axiom.md` | No versionado | No se toca |

**Alternativas descartadas.** Ponerla en el paso 6 contradice la ubicación que fija el delta («paso de cierre»). Dejar `.codex/AGENTS.md` desactualizado hasta la siguiente sincronización publicaría en el repositorio una guía que contradice la del binario.

**Justificación.** Es la extensión mínima aceptada en Q15, la proyección sigue siendo determinista y la única copia versionada queda alineada en la misma unidad.

### 2.20 D-20 — Cumplimiento de ratchets, pins y goldens (R10)

| Ratchet o guarda | Estrategia |
|---|---|
| Refusals (`internal/cli`, `internal/reviewtransaction`, `internal/sddstatus`) | Los rechazos nuevos viven en `internal/workspace`, `internal/specbaseline` e `internal/kickoff`, fuera del ratchet, y aun así nombran su continuación. En `cli` y `sddstatus`, los errores nuevos con mensaje estático nombran un comando `axiom ...` (por ejemplo `axiom workspace validate`, `axiom sdd archive-compose --help`, `axiom sdd integrate verify`) o envuelven con `%w`, que queda exento. En `sdd_governance.go` se retira, junto con su comentario, el rechazo con marcador `--cwd %q no es un directorio`, porque su caso pasa a `Locate`; se retira el par completo para no dejar un marcador huérfano. Se mantiene la rama `WorkspaceRoot` explícito con su mensaje en la línea base. Si alguna entrada de la línea base deja de corresponder a un sitio analizado, se encoge con `GENTLE_AI_REFUSAL_RATCHET_UPDATE=1`. Los campos `Reason`/`Message` nuevos nombran continuación o no son literales, para no subir el techo de 70 |
| Deadcode (`scripts/deadcode-ratchet.sh`) | Se ejecuta en cada unidad, y cada función nueva entra en la unidad que introduce su primer llamador alcanzable desde `cmd/axiom` (tabla siguiente). El techo de pruebas es una variable, no una función, así que el ratchet no lo evalúa. Si `loadWorkspaceConfigIfPresent` deja de usarse, se elimina en A1. Los ayudantes de fixtures viven en un paquete solo de pruebas (`internal/workspace/workspacetest`) que `cmd/axiom` no importa. `deadcode` sin `-test` solo carga el objetivo (`./cmd/axiom`) y sus dependencias, así que no analiza ese paquete; se confirma al ejecutar el ratchet en A1 |
| Pins (`internal/assets/assets_test.go`) | Inventario de ficheros embebidos ampliado con los nuevos. Pins nuevos en la misma unidad que el texto. Los pins existentes (bloques de `sdd-archive`, dos copias de la frase de `sdd-propose`, ubicación local del cambio en `sdd-spec`) se conservan intactos |
| Pins de `docs/` | `internal/workspace/docs_contract_test.go` (nuevo) lee `../../docs/manual-de-inicio.md` y `../../docs/openspec-config.md`, con el mismo patrón que `internal/absorptionledger/ledger_test.go` |
| Goldens (`testdata/golden/*`) | Se regeneran con la vía `-update` del repositorio en la unidad que cambia el texto (B2b: `sdd-init`; C3b: orquestadores y skills) y se vuelven a ejecutar sin `-update`. Ningún golden contiene el paso 7 de ODD (D-19) |
| `status_v2_clean_break_test` | Sin cambios (D-05) |

Primer llamador de cada función nueva:

| Funciones nuevas | Primer llamador alcanzable | Unidad |
|---|---|---|
| `workspace.Locate`, `ResolveOwnPointer` (rama de puntero de `Locate`), `Roots`, `parseLayered` (solo base), `kickoff.ArchivedChangeRoot` | Verbos SDD y `resolveGovernanceChangeRoot` | A1 |
| `Location.LegacySpecsDivergence` | livingdoc y knowledge | A2 |
| Decodificación y validación de declaraciones, `RequirementProfile.Compile`, `BaselineDocument.EffectiveScenarios` (regla de nivel), `DeclaredArtifactStore` | `ParseConfig` y `workspace.Validate` | B1a |
| `BaselineDocument.EffectiveIntegration` | Decorador del status | B1b |
| `Location.BaselineOwnership`, `Declarations.Ownership`, `ReadDeclarations`, `SamePlanningHome`, `GuardSpecsWrite`, `ExternalBaselineError`, `hub.ForeignPlanningHomeError` | Creadores, `init` y catálogo | B2a |
| `specbaseline.ComposeDocument`, `CompileProfile`, `RequirementProfile.EffectiveIdentity` | `archive-compose`, modo documento | C1a |
| `PlanIDs`, `IDMap.Substitute` | `archive-compose`, modo documento con ADDED | C1b |
| `Check`, `Prepare`, `ApplyCompose`, `ParseContextDelta`, `ParseAnchor`, `ResolveAnchor`, lectura del plan («ya integrado») | `integrate check` y `prepare`, `archive-compose --change` y modo documento | C2a |
| `Verify`, `RenderTrace`, `IDMap.CountProvisional` | `integrate verify` y `show` | C2b |

### 2.21 D-21 — Almacenes sin ficheros y baseline del proyecto

**Decisión** (aceptada por el usuario; REQ-24.62 y caso (g) de REQ-24.56). Con `baseline:` con documentos o `technical_context.maintained` declarados, la integración exige un almacén de ficheros, `openspec` o `hybrid`. El almacén efectivo es el que declara `artifact_store` en `openspec/config.yaml` y, solo sin declaración, el que resuelve el status para el cambio. Con `engram` o `none`:

- `axiom workspace validate` lo reporta como error, con código `1`, nombrando la declaración afectada, el almacén y la continuación. Solo evalúa el valor declarado: sin `artifact_store` no reporta nada por este motivo (D-04).
- El status bloquea solo el archivado, con la causa y la continuación (D-05). La planificación y `apply` no se bloquean, `baseline` y `technicalContext` se siguen proyectando y la consulta de baseline sigue respondiendo `project`: la declaración no es inválida a efectos de REQ-24.18.
- `integrate check` e `integrate prepare` reportan la causa, porque sin la carpeta del cambio en ficheros no hay delta de contexto ni traza.
- `sdd-archive` para como `blocked` con la causa literal (caso (g)).

La continuación es siempre la misma: declarar `artifact_store: openspec` o `hybrid` en `openspec/config.yaml`, o retirar la declaración. Un contexto declarado solo en modo `reference`, o la ausencia de baseline del proyecto y de contexto mantenido, no exige almacén de ficheros: esos proyectos conservan el comportamiento actual con `engram` o `none`.

**Alternativas descartadas.**

- *Materializar los deltas de Engram en ficheros temporales.* Añade una vía de integración sin la verificación por diff de la carpeta del cambio.
- *Saltar la integración en silencio,* que es lo que haría hoy la rama `engram` de la skill. Incumpliría REQ-21.17.
- *Validar también el almacén resuelto cuando no hay declaración.* `workspace validate` no resuelve cambios ni consulta Engram, y REQ-24.62 limita la validación al valor declarado; el status cubre el caso sin declaración.

**Justificación.** REQ-24.31 define el delta de contexto como «un fichero propio del cambio, en su carpeta», y la verificación E2 se basa en ficheros versionados. Leer primero el valor declarado corrige que el resolutor informe `openspec` para un `none` declarado con el cambio en disco.

### 2.22 D-22 — Lo que no cambia

| Elemento | Por qué no cambia |
|---|---|
| Kickoff, mecánica de compuertas (comandos, ledger y digest), handoff, cierre de roles | El ciclo de vida del cambio es change-local |
| `sdd-tasks` | Solo gana la línea de prohibición de REQ-24.30 (D-17); su procedimiento no cambia |
| Claves obligatorias del status v2, enumeración `nextRecommended`, sobre `gateQuestion` | Contrato solo aditivo |
| Composición nativa y sus pruebas | R1 por construcción |
| Los 17 comandos con `git rev-parse --show-toplevel \|\| pwd` | El descubrimiento ascendente vive en Go (REQ-24.11) |
| Detector de autoridad de edición | Con B', las rutas siguen resolviéndose contra `workspaceRoot` |
| `skillregistry` y `ScopeForPath` | Fuera de alcance |
| RDD (`reviewtransaction/risk.go`, R14) | Es opt-in y SDD no lo lanza; la clasificación por segmento `openspec` queda como riesgo aceptado |
| `RefuseArchivedRoot` | El prefijo `openspec/changes/archive` sigue siendo válido bajo B'; solo se añade a su lado `ArchivedChangeRoot` |
| Inferencia del proyecto Engram (R12) | Bajo B' usa el remoto del repositorio de especificaciones |

## 3. Arquitectura y flujo de datos

### 3.1 Componentes

```
 cmd/axiom (handoff, role, archive, knowledge, init, ui, workspace validate, sdd *)
        │                                   internal/cli (status, continue, kickoff, gate,
        │                                    attempt, archive-compose, integrate)
        ▼                                                    │
 ┌─────────────────────────── internal/workspace ───────────┴────────────┐
 │ Locate(start) ─► Location{Master, WorkspaceRoot, PlanningHome, Origin, │
 │                           ConfigPath, Config, Declarations, Notes}    │
 │ parseLayered ─► base │ Declarations (baseline, technical_context)     │
 │ BaselineOwnership() · GuardSpecsWrite(op) · LegacySpecsDivergence()   │
 │ Validate (sistema de ficheros, pathidentity)                           │
 └──────┬───────────────────┬─────────────────────┬──────────────────────┘
        │                   │                     │
        ▼                   ▼                     ▼
 internal/sddstatus   internal/specbaseline   livingdoc · knowledge · hub
 (Resolve + decorador  (compose con perfil,    dashboard · tui
  aditivo: baseline,    IDs, plan, contexto,   (catálogo en el planning home,
  technicalContext,     verificador E2, traza)  guardas, baseline externa, Q14)
  bloqueo de archive)          │
                               ▼
                git de solo lectura (status, diff, show, rev-parse, rev-list)
 internal/kickoff: RefuseArchivedRoot · ArchivedChangeRoot (cambios archivados con fecha)
```

### 3.2 Resolución del status (A y B)

```
axiom sdd status --cwd X --json
  └─ sddstatus.Resolve
       ├─ workspace.Locate(X) ──► rechazo tipado (salida ≠ 0, continuación) | Location
       ├─ resolveByPreferenceOrder(WorkspaceRoot = loc.WorkspaceRoot)   [sin cambios]
       │     └─ baseStatus(..., workspaceRoot, ...)  ► planningHome = <ws>/openspec
       ├─ decorateWorkspaceDeclarations(&status, loc)
       │     ├─ baseline / technicalContext (si válidos)
       │     ├─ notes: notas de Location, rutas multirepo
       │     └─ archive: declaración inválida | falta context-impact.md | almacén efectivo sin ficheros
       └─ renderPhaseInstructions (si se pidieron)
```

### 3.3 Archive con baseline del proyecto (flujo 1)

```
Orquestador         sdd-archive (modelo de sdd-apply si hay trabajo dirigido)     Nativo (axiom)
    │ status: archive ready   │                                                   │
    │──── lanzar ────────────►│ cd <ws> && integrate prepare ────────────────────►│ valida todo en memoria,
    │                         │◄──────────────── plan JSON + integration.yaml ────│ HEAD y foto por repo, IDs
    │                         │ ejecutar compose_command literal ────────────────►│ archive-compose --change
    │                         │◄───────────────────────────── 0 | rechazo ────────│ --in-place (todo o nada)
    │                         │ editar secciones del cambio y anclas de contexto  │
    │                         │ (textos ya sustituidos, Edit acotado)             │
    │                         │ integrate verify ────────────────────────────────►│ (a)–(e) contra el HEAD
    │                         │◄────────────────────────── 0 | violaciones ───────│ del plan
    │                         │ Step 3: mover (bloque fijado)                     │
    │                         │ escribir archive-report.md                        │
    │                         │ integrate show --format markdown >> informe ─────►│ traza (REQ-24.52), cambio
    │◄──── resultado ─────────│                                                   │ archivado con fecha
```

Ante cualquier salida ≠ 0, el agente para como `blocked` con el mensaje literal. El cambio permanece en `openspec/changes/` y las ediciones ya hechas quedan sin confirmar (REQ-24.56). Los flujos 2 y 3 de D-13 sustituyen el paso de composición con perfil por los bloques fijados del `Step 2` seguidos de `axiom archive sync`.

### 3.4 TUI: pantalla de documentación viva con baseline externa (`rules.design`)

```
Usuario      Model.Update (ScreenLivingDoc)        livingdoc.Service            workspace
   │ Enter          │                                      │                        │
   │───────────────►│ loadLivingDocs()                     │                        │
   │                │── NewService(".").GetCatalog() ─────►│── Locate(".") ────────►│
   │                │                                      │◄─ Location (project) ──│
   │                │◄──── {Specs: [], ExternalBaseline} ──│                        │
   │                │ GovernanceMessage = dominios y rutas │                        │
   │ tecla s        │                                      │                        │
   │───────────────►│── Sync() ───────────────────────────►│── GuardSpecsWrite ────►│
   │                │◄──── *SecondBaselineError ───────────│◄─ rechazo + continuación│
   │                │ GovernanceMessage = "Error al sincronizar catálogo: …"        │
```

### 3.5 Ficheros de datos nuevos

| Fichero | Dueño | Escritores | Ciclo de vida |
|---|---|---|---|
| `axiom.yaml` (`baseline:`, `technical_context:`) | Proyecto | Persona | Versionado en el repositorio de especificaciones |
| `<changeRoot>/context-impact.md` | Cambio | `sdd-design` y `sdd-apply` | Se archiva con el cambio, sin alterar |
| `<changeRoot>/integration.yaml` | Cambio | `integrate prepare`, `archive-compose --in-place`, `integrate verify` (solo nativo) | Se archiva con el cambio |
| Sección «Traza de integración» del informe | Cambio | `integrate show` añadido por shell | Parte de `archive-report.md` |

## 4. Interfaces y contratos

### 4.1 `internal/workspace`, `internal/kickoff` e `internal/hub`

```go
// locate.go
type Origin string

const (
	OriginPointer  Origin = "pointer"  // «puntero»
	OriginConfig   Origin = "config"   // «configuración»
	OriginOpenSpec Origin = "openspec" // «openspec»
	OriginStart    Origin = "start"    // «directorio de partida»
)

// DiscoveryCeilingOverride lo fijan solo las pruebas: directorios que el
// descubrimiento ascendente no evalúa, ni a ellos ni a sus ancestros. Es nil
// en producción (mismo patrón que LookPathOverride).
var DiscoveryCeilingOverride []string

type Location struct {
	Start         string           // directorio de partida absoluto y limpio
	MarkerDir     string           // directorio del marcador; "" con OriginStart
	Master        string           // carpeta maestra
	WorkspaceRoot string           // padre del planning home
	PlanningHome  string           // siempre filepath.Join(WorkspaceRoot, "openspec")
	Origin        Origin
	ConfigPath    string           // axiom.yaml efectivo; "" si no hay
	Config        *WorkspaceConfig // configuración base; nil si no hay o no es cargable
	ConfigErr     error            // causa de no carga de un axiom.yaml directo (REQ-24.7, caso 3)
	Declarations  Declarations
	Notes         []string         // avisos no bloqueantes (caso 3, planning home divergente)
}

// Locate es determinista y de solo lectura (REQ-24.1).
func Locate(start string) (Location, error)

// ResolveOwnPointer aplica a <dir>/.axiom-workspace, sin ascender, las reglas de
// Locate para un puntero (pasos 4, 7, 8 y 9). Nunca exige que exista el planning
// home. found es false si dir no contiene puntero. Errores: *LocateError de tipo
// pointer_broken, config_unloadable, specs_repository_missing o
// ambiguous_planning_home. Locate delega en ella y axiom init la usa.
func ResolveOwnPointer(dir string) (loc Location, found bool, err error)

// SamePlanningHome compara por identidad (pathidentity.SameDirectory) y, si
// alguna ruta no existe, por ruta absoluta limpia (sin distinguir mayúsculas en
// Windows). Ese respaldo léxico queda fuera de la política de pathidentity.
func SamePlanningHome(a, b string) bool

// Roots aplica el enraizado por topología (REQ-24.3, REQ-24.4).
func Roots(cfg *WorkspaceConfig, master string) (workspaceRoot, planningHome string)

// LegacySpecsDivergence devuelve el aviso Q14 de los consumidores de catálogo (REQ-24.10).
func (l Location) LegacySpecsDivergence() (string, bool)

type LocateErrorKind string

const (
	LocateStartInvalid           LocateErrorKind = "start_invalid"
	LocatePointerBroken          LocateErrorKind = "pointer_broken"
	LocateConfigUnloadable       LocateErrorKind = "config_unloadable"
	LocateSpecsRepositoryMissing LocateErrorKind = "specs_repository_missing"
	LocateAmbiguousPlanningHome  LocateErrorKind = "ambiguous_planning_home"
)

type LocateError struct {
	Kind         LocateErrorKind
	Path         string // ruta principal implicada
	Cause        string // causa en castellano
	Continuation string // acción o comando accionable, con rutas en pathquote.Quote
	Err          error
}

func (e *LocateError) Error() string
func (e *LocateError) Unwrap() error
```

```go
// declarations.go
type BaselineConfig struct {
	Documents []BaselineDocument `yaml:"documents"`
}

type BaselineDocument struct {
	Domain      string              `yaml:"domain"`
	Path        string              `yaml:"path"`
	Requirement *RequirementProfile `yaml:"requirement,omitempty"`
	Integration IntegrationMode     `yaml:"integration,omitempty"` // directed | register
	Scenarios   ScenariosMode       `yaml:"scenarios,omitempty"`   // include | omit
}

type RequirementProfile struct {
	Heading  string       `yaml:"heading"`
	Level    int          `yaml:"level"`
	Identity IdentityKind `yaml:"identity,omitempty"` // id | text
	Render   string       `yaml:"render"`
}

type (
	IntegrationMode string // "compose" (derivado), "directed", "register"
	ScenariosMode   string // "include", "omit"
	IdentityKind    string // "id", "text"
)

func (d BaselineDocument) EffectiveIntegration() IntegrationMode
func (d BaselineDocument) EffectiveScenarios() ScenariosMode
func (p RequirementProfile) EffectiveIdentity() IdentityKind
func (p RequirementProfile) Compile() (*regexp.Regexp, error) // verifica los grupos id y title

type TechnicalContextConfig struct {
	Maintained *ContextMode `yaml:"maintained,omitempty"`
	Reference  *ContextMode `yaml:"reference,omitempty"`
}

type ContextMode struct {
	Entry string   `yaml:"entry,omitempty"`
	Paths []string `yaml:"paths"`
}

// Declarations: el valor cero equivale a «sin declaraciones» (no hay axiom.yaml,
// o es legible y no declara bloques nuevos).
type Declarations struct {
	ParseErr            error // no nulo solo si existe axiom.yaml y no se puede leer como YAML
	Baseline            *BaselineConfig
	BaselineErr         error
	TechnicalContext    *TechnicalContextConfig
	TechnicalContextErr error
}

// ReadDeclarations lee configPath y aplica la carga por capas; solo devuelve las
// declaraciones (lo usa axiom init para el axiom.yaml propio del destino).
func ReadDeclarations(configPath string) Declarations

// DeclaredArtifactStore lee artifact_store de <workspaceRoot>/openspec/config.yaml
// (o config.yml) con las mismas reglas que sddstatus.declaredArtifactStore.
func DeclaredArtifactStore(workspaceRoot string) (store string, declared bool)

// types.go (modificado): campos decodificados aparte y de forma estricta.
type WorkspaceConfig struct {
	Workspace        WorkspaceSection        `yaml:"workspace"`
	Roles            map[string]RoleConfig   `yaml:"roles"`
	Governance       GovernanceConfig        `yaml:"governance,omitempty"`
	Baseline         *BaselineConfig         `yaml:"-"`
	TechnicalContext *TechnicalContextConfig `yaml:"-"`
}
```

```go
// baseline_guard.go
type BaselineOwnership string

const (
	BaselineDefault       BaselineOwnership = "default"       // «por defecto»
	BaselineProject       BaselineOwnership = "project"       // «del proyecto»
	BaselineIndeterminate BaselineOwnership = "indeterminate" // «indeterminada»
)

// Ownership es la única implementación de la consulta (REQ-24.22).
func (d Declarations) Ownership() (BaselineOwnership, error)

// BaselineOwnership delega en l.Declarations.Ownership().
func (l Location) BaselineOwnership() (BaselineOwnership, error)

type SecondBaselineError struct {
	Operation  string
	Ownership  BaselineOwnership
	ConfigPath string
	Domains    []string
	Cause      error
}

func (e *SecondBaselineError) Error() string
func (e *SecondBaselineError) Unwrap() error

// GuardSpecsWrite devuelve nil si op puede crear o actualizar
// <planning home>/specs o <planning home>/INDEX.md.
func (l Location) GuardSpecsWrite(op string) error
```

```go
// internal/kickoff/archived.go (añadido)

// ArchivedChangeRoot devuelve la carpeta archivada de change: primero
// openspec/changes/archive/<change> (forma heredada) y, si no existe, la entrada
// más reciente openspec/changes/archive/YYYY-MM-DD-<change>.
func ArchivedChangeRoot(workspaceRoot, change string) (string, bool)
```

```go
// internal/hub/init.go (añadido)

// ForeignPlanningHomeError es el rechazo de axiom init cuando el puntero propio del
// destino resuelve un planning home distinto de <destino>/openspec (multi-project-hub
// REQ-2.2). Su mensaje nombra el puntero, el planning home y la continuación.
type ForeignPlanningHomeError struct {
	PointerPath   string
	PlanningHome  string
	WorkspaceRoot string // continuación: axiom init --path "<WorkspaceRoot>"
}

func (e *ForeignPlanningHomeError) Error() string
```

### 4.2 Status v2 (campos aditivos)

```json
{
  "baseline": {
    "documents": [
      { "domain": "requisitos-funcionales", "path": "/w/repo-specs/01_Requisitos_Funcionales.md", "integration": "compose" },
      { "domain": "modelo-de-datos", "path": "/w/repo-specs/03_Modelo_Datos.md", "integration": "directed" },
      { "domain": "flujos-de-negocio", "path": "/w/repo-specs/04_Flujos_Negocio.md", "integration": "register" }
    ]
  },
  "technicalContext": {
    "maintained": { "paths": ["/w/repo-specs/context"], "entry": "/w/repo-specs/context/TECHNICAL_CONTEXT.md" },
    "reference": { "paths": ["/w/repo-specs/00_Glosario.md"] }
  }
}
```

Las dos claves desaparecen cuando no hay declaración válida. `planningHome`, `actionContext` y el resto de claves conservan nombre, tipo y semántica.

### 4.3 `internal/specbaseline` y `internal/livingdoc`

```go
type Profile struct {
	Heading   *regexp.Regexp
	Level     int
	Identity  workspace.IdentityKind
	Render    string
	Scenarios workspace.ScenariosMode
}

func CompileProfile(doc workspace.BaselineDocument) (Profile, error)

// PlannedDocument es un documento con perfil que participa en la composición del cambio.
type PlannedDocument struct {
	Domain   string
	Path     string // absoluta
	RelPath  string // relativa a workspaceRoot, con '/'
	Profile  Profile
	Original []byte // bytes del árbol de trabajo
	Delta    []byte
}

type ComposeInput struct {
	Document           PlannedDocument
	IDs                IDMap // correspondencia del cambio entero
	Supersede          bool
	ExplicitSuperseded []string
}

type ComposeResult struct {
	Output   []byte
	Entries  []TraceEntry // una por requisito del delta
	Warnings []string     // colas de secciones contiguas (REQ-24.41)
}

func ComposeDocument(in ComposeInput) (ComposeResult, error) // *ComposeError: dominio, sección, requisito, motivo, continuación

type IDAssignment struct {
	Provisional string `yaml:"provisional" json:"provisional"`
	Definitive  string `yaml:"definitive" json:"definitive"`
	Domain      string `yaml:"domain" json:"domain"`
}

type IDMap struct{ /* provisional → asignación, orden estable */ }

func PlanIDs(docs []PlannedDocument) (IDMap, error)         // REQ-24.43 y REQ-24.59 (a)–(d)
func (m IDMap) Substitute(text string) string               // tokens completos
func (m IDMap) CountProvisional(text string) map[string]int // para (e)

type (
	EntryKind   string // compose | directed | register | context
	TraceAction string // added | modified | removed | created
	TraceState  string // pending | integrated
)

// TraceEntry es una entrada de la traza (REQ-24.52).
type TraceEntry struct {
	ID               string      `yaml:"id" json:"id"`
	Provisional      string      `yaml:"provisional,omitempty" json:"provisional,omitempty"`
	Kind             EntryKind   `yaml:"kind" json:"kind"`
	Domain           string      `yaml:"domain,omitempty" json:"domain,omitempty"`
	Document         string      `yaml:"document" json:"document"` // relativo a workspaceRoot, con '/'
	Anchor           string      `yaml:"anchor" json:"anchor"`
	Action           TraceAction `yaml:"action" json:"action"`
	State            TraceState  `yaml:"state" json:"state"`
	Reason           string      `yaml:"reason,omitempty" json:"reason,omitempty"`
	ScenariosOmitted bool        `yaml:"scenarios_omitted,omitempty" json:"scenariosOmitted,omitempty"`
	Warnings         []string    `yaml:"warnings,omitempty" json:"warnings,omitempty"`
}

const MaxDirtyEntries = 1000

// Plan es el contenido de integration.yaml (axiom.sdd-integration/v1).
type Plan struct {
	Schema       string           `yaml:"schema"`
	Change       string           `yaml:"change"`
	PreparedAt   time.Time        `yaml:"prepared_at"`
	VerifiedAt   *time.Time       `yaml:"verified_at,omitempty"`
	InputsSHA256 string           `yaml:"inputs_sha256"`
	Options      PlanOptions      `yaml:"options"`
	Repositories []PlanRepository `yaml:"repositories"`
	IDMap        []IDAssignment   `yaml:"id_map"`
	Documents    []PlanDocument   `yaml:"documents"`
	Context      ContextDecision  `yaml:"context"`
	Entries      []TraceEntry     `yaml:"entries"`
}

type PlanOptions struct {
	Supersede              bool     `yaml:"supersede"`
	SupersededRequirements []string `yaml:"superseded_requirements"`
}

type PlanRepository struct {
	Repo  string       `yaml:"repo"`  // relativo a la carpeta maestra, con '/'
	Head  string       `yaml:"head"`  // commit en prepare; "" si no hay commits
	Dirty []DirtyEntry `yaml:"dirty"` // como máximo MaxDirtyEntries
}

type DirtyEntry struct {
	Path   string `yaml:"path"`   // relativa al repositorio, con '/'
	SHA256 string `yaml:"sha256"` // "" si el fichero estaba borrado
}

type PlanDocument struct {
	Domain       string                    `yaml:"domain"`
	Path         string                    `yaml:"path"` // relativo a workspaceRoot, con '/'
	Integration  workspace.IntegrationMode `yaml:"integration"`
	BeforeSHA256 string                    `yaml:"before_sha256"`
	AfterSHA256  string                    `yaml:"after_sha256,omitempty"` // solo compose
	State        TraceState                `yaml:"state,omitempty"`        // solo compose
}

type ContextDecision struct {
	Decision string `yaml:"decision"` // entries | no_impact | none
	Reason   string `yaml:"reason,omitempty"`
}

// Report es la salida de check (axiom.sdd-integration-check/v1) y de prepare (axiom.sdd-integration-plan/v1).
type Report struct {
	Schema         string           `json:"schema"`
	Change         string           `json:"change"`
	Baseline       string           `json:"baseline"` // project | default
	Documents      []DocumentReport `json:"documents"` // operaciones y familias por documento
	IDMap          []IDAssignment   `json:"idMap"`
	Context        ContextReport    `json:"context"`  // declarado, presente, decisión y entradas
	ComposeCommand string           `json:"composeCommand,omitempty"`
	Directed       []DirectedItem   `json:"directed,omitempty"`
	ContextItems   []ContextItem    `json:"contextItems,omitempty"`
	Register       []TraceEntry     `json:"register,omitempty"`
	Warnings       []string         `json:"warnings"`
	Errors         []string         `json:"errors"`
}

// VerifyReport es la salida de verify (axiom.sdd-integration-verify/v1).
type VerifyReport struct {
	Schema     string      `json:"schema"`
	Change     string      `json:"change"`
	Passed     bool        `json:"passed"`
	Violations []Violation `json:"violations"`
}

type Violation struct {
	Check  string `json:"check"` // presence | anchor | scope | trace | ids | compose
	Entry  string `json:"entry,omitempty"`
	File   string `json:"file,omitempty"`
	Detail string `json:"detail"`
}

type ContextAction string // add | modify | remove

type ContextEntry struct {
	ID       string // C1..Cn
	Document string // relativo a workspaceRoot, con '/'
	Anchor   Anchor
	Action   ContextAction
	Reason   string
	Body     string
	Line     int
}

type ContextDelta struct {
	NoImpact bool
	Reason   string
	Entries  []ContextEntry
}

func ParseContextDelta(data []byte) (ContextDelta, error)

type Anchor struct {
	Path    []string
	Ordinal int // 0 = sin ordinal
}

// HeadingRef identifica un encabezado ATX del documento.
type HeadingRef struct {
	Line  int      // 1-based
	Level int
	Path  []string // títulos desde el encabezado raíz hasta este
}

func ParseAnchor(s string) (Anchor, error)
func ResolveAnchor(doc []byte, a Anchor) []HeadingRef

type ChangeTarget struct {
	Location     workspace.Location
	Change       string
	ChangeRoot   string   // activa o, para show, archivada (kickoff.ArchivedChangeRoot)
	AllowedRoots []string // actionContext.allowedEditRoots del status
}

func Check(ctx context.Context, t ChangeTarget) (Report, error)
func Prepare(ctx context.Context, t ChangeTarget, opts PlanOptions) (Plan, Report, error)
func ApplyCompose(ctx context.Context, t ChangeTarget, domains []string) (Report, error)
func Verify(ctx context.Context, t ChangeTarget) (VerifyReport, error)
func RenderTrace(plan Plan, format string) ([]byte, error) // markdown | json
```

```go
// internal/livingdoc (añadido)

// ExternalBaseline informa de que la baseline es del proyecto (REQ-24.25).
type ExternalBaseline struct {
	Documents []ExternalBaselineDocument `json:"documents"`
}

type ExternalBaselineDocument struct {
	Domain string `json:"domain"`
	Path   string `json:"path"` // absoluta
}

// ExternalBaselineError lo devuelve GetSpecDetail para un dominio de la baseline del proyecto.
type ExternalBaselineError struct {
	Domain     string
	Path       string // documento declarado, absoluto
	ConfigPath string
}

func (e *ExternalBaselineError) Error() string
```

### 4.4 CLI

| Comando | Salida | Código |
|---|---|---|
| `axiom sdd archive-compose ...` (modos de §2.13) | Composición por stdout o `--output`; en `--in-place`, resumen | `0` o `1` con mensaje que nombra sección, requisito y continuación |
| `axiom sdd integrate check --cwd <ws> --change <c> [--json]` | `axiom.sdd-integration-check/v1`: dominios, operaciones, cuerpos conflictivos, familias (máximo, rango definitivo, inicio provisional sugerido), correspondencia, estado del delta de contexto, avisos y errores | `0` sin errores; `1` con errores |
| `axiom sdd integrate prepare --cwd <ws> --change <c> [--supersede] [--superseded-requirements L] [--json]` | `axiom.sdd-integration-plan/v1`: `composeCommand`, `directed` (documento, sección, entradas con texto sustituido), `contextItems` (documento, ancla, acción, motivo, texto sustituido, `create`), `register`, `idMap` y `warnings` | `0` o `1` |
| `axiom sdd integrate verify --cwd <ws> --change <c> [--json]` | `axiom.sdd-integration-verify/v1`: comprobaciones y violaciones | `0` o `1` |
| `axiom sdd integrate show --cwd <ws> --change <c> [--format markdown\|json]` | Traza, también de un cambio ya archivado con prefijo de fecha | `0` o `1` si no hay traza |

`runSDD` gana el caso `integrate` y su línea de ayuda.

### 4.5 `integration.yaml` (`axiom.sdd-integration/v1`)

```yaml
schema: axiom.sdd-integration/v1
change: inc-99-ejemplo
prepared_at: 2026-10-04T10:00:00Z
inputs_sha256: 3f2a…            # deltas + context-impact.md + instantánea de la declaración
options: {supersede: false, superseded_requirements: []}
repositories:                   # relativos a la carpeta maestra; como máximo 1.000 entradas sucias por repo
  - {repo: repo-specs, head: 9c1e…, dirty: [{path: notas/borrador.md, sha256: d4…}]}
  - {repo: repo-backend, head: 7ab2…, dirty: []}
id_map:
  - {provisional: RF-9001, definitive: RF-008, domain: requisitos-funcionales}
documents:                      # rutas relativas a workspaceRoot, con '/'
  - {domain: requisitos-funcionales, path: 01_Requisitos_Funcionales.md, integration: compose,
     before_sha256: a1…, after_sha256: b2…, state: integrated}
  - {domain: modelo-de-datos, path: 03_Modelo_Datos.md, integration: directed, before_sha256: c3…}
context: {decision: entries, reason: ""}      # entries | no_impact | none
entries:
  - {id: RF-008, provisional: RF-9001, kind: compose, domain: requisitos-funcionales,
     document: 01_Requisitos_Funcionales.md, anchor: RF-008, action: added, state: integrated,
     scenarios_omitted: true}
  - {id: "Reintento de pago", kind: directed, domain: modelo-de-datos,
     document: 03_Modelo_Datos.md, anchor: inc-99-ejemplo, action: added, state: pending}
  - {id: C1, kind: context, document: context/ARQUITECTURA.md,
     anchor: "Componentes > Servicio de pagos", action: modified, reason: "…", state: pending}
  - {id: "Flujo de devolución", kind: register, domain: flujos-de-negocio,
     document: 04_Flujos_Negocio.md, anchor: inc-99-ejemplo, action: added, state: pending}
```

El fichero se decodifica de forma estricta. Las rutas son relativas, así que no guarda rutas de la máquina. `verify` añade `verified_at`. `show --format markdown` genera la tabla «Identificador | Documento | Ancla | Acción | Estado». El identificador de un ADDED con perfil se muestra como `RF-9001 → RF-008`. Al final se añaden la decisión «Sin impacto» con su motivo, los escenarios omitidos y los avisos.

### 4.6 Catálogo y knowledge (JSON aditivo)

| Tipo | Campos nuevos |
|---|---|
| `livingdoc.LivingCatalog` | `external_baseline` (`documents[].domain`, `documents[].path`) y `warnings` (incluido el aviso Q14) |
| `knowledge.SweepResult` y `knowledge.QueryResult` | `notes` (omisión de la siembra o del enriquecimiento, aviso Q14, la búsqueda spec-first no cubre la baseline externa) |
| `hub.InitResult` | `Notes []string` (omisión de `openspec/specs/` e `INDEX.md`, y violaciones de la validación del punto 4 de REQ-2.2) |

## 5. Mapa de cambios por unidad candidata

### A1 — Resolver, verbos SDD y aislamiento de pruebas

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/workspace/locate.go` | Crear | `Locate`, `Roots`, `Location`, `LocateError`, `DiscoveryCeilingOverride`, nota de planning home divergente |
| `internal/workspace/loader.go` | Modificar | Extraer `parseLayered`, solo con la base en esta unidad; `ParseConfig` y `LoadConfig` sin cambio de comportamiento |
| `internal/workspace/locate_test.go`, `workspacetest/fixtures.go` | Crear | Tablas de REQ-24.1 a REQ-24.7; constructores solo de pruebas (canónico multirepo y embebido) |
| `internal/kickoff/archived.go` (+ prueba) | Modificar | `ArchivedChangeRoot` (nombre exacto y prefijo de fecha) |
| `internal/sddstatus/status.go` | Modificar | Costura en `resolveWorkspaceRoot` y `Resolve` (pocas líneas) |
| `internal/sddstatus/workspace_declarations.go` | Crear | Decorador: en esta unidad solo las notas de `Location` |
| `internal/sddstatus/planning_home_test.go`, `main_test.go` | Crear | Equivalencia embebida, canónico, ambigüedad, notas; `TestMain` con el techo de pruebas |
| `internal/cli/sdd_governance.go`, `sdd_kickoff.go`, `sdd_attempt.go` | Modificar | Resolución por `Locate`; cambios archivados con fecha; configuración desde `loc.Config` |
| `internal/cli/sdd_planning_home_test.go`; `protocol_probe_test.go` | Crear; Modificar | `kickoff`, `gate` (incluido el cambio archivado con fecha) y `attempt` desde tres puntos de partida; techo en el `TestMain` existente |
| `cmd/axiom/main.go` | Modificar | `resolveChangeDir`, `resolveHandoffFile`, `runHandoff*`, `runRole*` con `Location` |
| `cmd/axiom/main_test.go` | Modificar | `TestRunSDDStatusJSON` y `TestRunSDDStatusNonexistentChange` con `t.Chdir(t.TempDir())` (H1) |
| `cmd/axiom/planning_home_test.go`, `main_ceiling_test.go` | Crear | `handoff` y `role` en el canónico; techo |
| `internal/tui/model_test.go` | Modificar | Aislar `TestGovernanceScreensNavigationAndActions` en un espacio temporal (R28) |
| `internal/dashboard/dashboard_test.go` | Modificar | Aislar `TestArchiveEndpoints` en un espacio temporal con su propio `openspec/specs/`, en lugar de `NewService("../..")` (M1, R28). Ya lo fijó main (#71/#72): se omitió este hunk al rebasar y el fichero no entra en A1-c1 |
| `.github/workflows/ci.yml` | Modificar | Paso nuevo tras `Run unit tests` (`go test ./...`): fallar si `git status --porcelain` no está vacío. Debe estar en verde en la fusión de A1 |
| `internal/assets/skills/_shared/sdd-phase-common.md`; `internal/assets/assets_test.go` | Modificar | Sección G (REQ-24.11) y su pin |
| `internal/workspace/docs_contract_test.go`; `docs/manual-de-inicio.md` | Crear; Modificar | Receta canónica y evidencia `pr_merged`, con su pin (REQ-24.12) |

### A2 — Consumidores de catálogo

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/workspace/locate.go` | Modificar | `LegacySpecsDivergence` |
| `internal/livingdoc/service.go`, `types.go` (+ prueba) | Modificar | `Locate` en cada llamada; aviso Q14; `Warnings` |
| `internal/knowledge/sweep.go`, `query.go`, `planner.go`, `types.go` (+ prueba) | Modificar | `SpecsRoot` por defecto desde `Locate`; aviso Q14 en `notes` |
| `internal/dashboard/service.go` (+ prueba) | Modificar | `planningRoot()` en cinco rutas |
| `internal/tui/model.go` (+ prueba) | Modificar | Cargadores con `Locate(".")` |
| `cmd/axiom/main.go` | Modificar | `runUI` reconoce el puntero; `archive list` imprime avisos |

### B1a — Declaración y validación

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/workspace/declarations.go`, `types.go`, `loader.go` | Crear; Modificar; Modificar | Tipos, decodificación estricta, validación acumulada, regla de nivel; `parseLayered` con declaraciones |
| `internal/workspace/validator.go`, `artifact_store.go` | Modificar; Crear | Sistema de ficheros con `pathidentity`, enlaces simbólicos de los dos modos de contexto, solapes por identidad; `DeclaredArtifactStore` y error de almacén declarado (REQ-24.62) |
| `internal/workspace/declarations_test.go`, `validator_test.go` | Crear; Modificar | Tablas de REQ-24.13 a 24.18, 24.26, 24.27, 24.60 y 24.62 (parte de la validación) |
| `cmd/axiom/main.go` | Modificar | `workspace validate` imprime avisos con `COMPLIANT` |
| `docs/openspec-config.md`; `internal/workspace/docs_contract_test.go` | Modificar | Frontera de configuración y su pin (REQ-24.21) |

### B1b — Proyección en el status

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/sddstatus/status.go`, `status_v2.go` | Modificar | Campos opcionales y tipos de proyección |
| `internal/sddstatus/workspace_declarations.go` | Modificar | Proyección, bloqueo de archive, aviso multirepo, instrucción de archive |
| `internal/sddstatus/declarations_status_test.go` | Crear | REQ-24.18 a 24.20, 24.28, 24.36 y 24.62 (parte del status, incluido el `none` declarado con el cambio en disco) |
| `internal/sddstatus/artifact_store_parity_test.go` | Crear | Paridad entre `declaredArtifactStore` y `workspace.DeclaredArtifactStore` sobre las mismas fixtures |
| `internal/assets/skills/_shared/sdd-status-contract.md`, `internal/assets/assets_test.go` | Modificar | Contrato y pin (REQ-24.19) |

### B2a — Guardas nativas y baseline externa

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/workspace/baseline_guard.go` (+ prueba) | Crear | Consulta única, error tipado y `SamePlanningHome` |
| `internal/hub/init.go`, `types.go` (+ prueba) | Modificar | Puntero propio con `ResolveOwnPointer`, `ForeignPlanningHomeError`, validación del punto 4 de REQ-2.2, guarda y `Notes`, todo limitado a los marcadores del propio destino |
| `internal/livingdoc/service.go`, `types.go` (+ prueba) | Modificar | Guarda en `Sync` y `ColdStart`; baseline externa en catálogo y detalle |
| `internal/knowledge/sweep.go`, `query.go`, `recorder.go`, `finalize.go`, `types.go` (+ prueba) | Modificar | Guardas y `notes` |
| `internal/dashboard/server.go`, `assets/app.js` (+ prueba) | Modificar | 409 en sync; aviso de baseline externa en la pestaña |
| `internal/tui/model.go` (+ prueba) | Modificar | Mensaje de baseline externa |
| `cmd/axiom/main.go` (+ prueba) | Modificar | Salidas de `init`, `archive *` y `knowledge *` |

### B2b — Skills de lectura

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/assets/skills/_shared/project-baseline-convention.md` | Crear | Localizadores, lectura y prohibiciones |
| `internal/assets/skills/sdd-init/SKILL.md`, `references/init-details.md`, `_shared/openspec-convention.md` | Modificar | Guarda de REQ-24.24 |
| `internal/assets/skills/{sdd-explore,sdd-design,sdd-apply,sdd-verify}/SKILL.md` | Modificar | Lectura del contexto técnico (REQ-24.29) |
| Las 8 `SKILL.md` de fase | Modificar | Prohibición de escribir en rutas de referencia (REQ-24.30) |
| `internal/assets/assets_test.go`, `testdata/golden/*` | Modificar | Inventario, pins y goldens de `sdd-init` |

### C1a — Composición con perfil (sin ADDED)

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/specbaseline/doc.go`, `document.go`, `profile.go`, `delta.go`, `compose.go` | Crear | Modelo de documento, perfil, delta, MODIFIED, REMOVED y RENAMED, colas, cuerpo, fidelidad de bytes |
| `internal/specbaseline/*_test.go`, `testdata/reference/` | Crear | Fixtures sintéticos con las formas del spike; tablas por requisito |
| `internal/cli/sdd_archive_compose.go` | Modificar | `--cwd` y `--document` (modo documento con supersede), exclusión con `--canonical`, flag repetible |
| `internal/cli/sdd_archive_compose_document_test.go` | Crear | REQ-24.47 y 24.48; equivalencia nativa (R1) |

### C1b — ADDED e IDs

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/specbaseline/ids.go`, `compose.go` (+ prueba) | Crear; Modificar | Ubicación, familias, asignación, sustitución, `scenarios: omit` |

### C2a — Plan

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/specbaseline/contextdelta.go`, `anchors.go`, `plan.go`, `trace.go`, `gitview.go` (+ prueba) | Crear | Delta de contexto, anclas, `Check`, `Prepare`, `ApplyCompose`, lectura y escritura del plan, `HEAD` y foto acotada |
| `internal/cli/sdd_integrate.go` (+ prueba) | Crear | `check` y `prepare` |
| `internal/cli/sdd_archive_compose.go` (+ prueba) | Modificar | Modo de cambio `--change --in-place`; «ya integrado» en modo documento |
| `cmd/axiom/main.go` | Modificar | Caso `integrate` y ayuda |

### C2b — Verificador

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/specbaseline/verify.go`, `trace.go` (+ prueba) | Crear; Modificar | `Verify`, `RenderTrace`, `CountProvisional` |
| `internal/cli/sdd_integrate.go` (+ prueba) | Modificar | `verify` y `show` (cambios archivados con fecha) |

### C3a — Skills de integración y archive

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/assets/skills/sdd-archive/SKILL.md`; `references/project-baseline-integration.md` | Modificar; Crear | Ramas de D-13, invocación de REQ-24.47 en el `SKILL.md`, paso de `INDEX.md`, prohibiciones |
| `internal/assets/skills/{sdd-explore,sdd-propose,sdd-spec,sdd-onboard}/SKILL.md` | Modificar | Lectura de la baseline (REQ-24.54); reglas de IDs y `integrate check` en `sdd-spec` |
| `internal/assets/skills/{sdd-design,sdd-apply}/SKILL.md` | Modificar | Delta de contexto y remediación de compuertas |
| `internal/assets/skills/sdd-verify/SKILL.md`, `references/report-format.md` | Modificar | Aviso de contexto |
| `internal/assets/skills/_shared/project-baseline-convention.md`, `openspec-convention.md` | Modificar | Formatos, IDs, comandos `integrate`, remediación, árbol |
| `internal/assets/assets_test.go` | Modificar | Inventario y pins |

### C3b — Compuertas, ODD, documentación y goldens

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modificar | Presentación y remediación de compuertas (D-18) |
| `internal/assets/claude/sdd-orchestrator-workflow.md` | Modificar | Regla de modelo de archive (D-14) |
| `internal/components/agentguidance/routing.go`, `routing_test.go` | Modificar | Regla de ODD (D-19) |
| `.codex/AGENTS.md` | Regenerar | Salida de `axiom sync --scope workspace --agent codex` con el binario de la rama |
| `internal/assets/assets_test.go`, `testdata/golden/*` | Modificar | Pins de compuertas y regeneración de orquestadores |
| `docs/manual-de-inicio.md` | Modificar | §8 baseline del proyecto; nota de modelo para otros runtimes; corrección de la línea que atribuye `INDEX.md` a `archive-compose` |

## 6. Trazabilidad requisito → diseño

### 6.1 Capacidades nuevas

| Requisito | Elemento de diseño | Unidad |
|---|---|---|
| REQ-24.1 Primitivo único y de solo lectura | D-01, `Locate`, `LocateError` | A1 |
| REQ-24.2 Descubrimiento por proximidad | D-02, pasos 1–3, precedencia; techo solo de pruebas | A1 |
| REQ-24.3 Enraizado por topología | D-02, paso 7 (`Roots`) | A1 |
| REQ-24.4 Invariancia embebida | D-02, excepción embebida de REQ-24.5; prueba de equivalencia, también sin `axiom.yaml` | A1 |
| REQ-24.5 Deducción de la carpeta maestra | D-02, paso 5; la excepción embebida la fija la spec (solo desacoplado y multirepo deducen por sufijo) | A1 |
| REQ-24.6 Ambigüedad maestra con `openspec/` y puntero | D-02, paso 9 (identidad); `init` rechaza con un puntero propio hacia otro planning home (D-06, multi-project-hub REQ-2.2) | A1, B2a |
| REQ-24.7 Configuración no resoluble | D-02, pasos 4 (`ResolveOwnPointer`), 5 y 8 (caso 4: se exige `workspaceRoot`, nunca el planning home); D-03 (caso 2: destino del puntero ilegible o con base inválida; un bloque nuevo inválido no rechaza; caso 3: embebido con aviso); notas (D-05) | A1 |
| REQ-24.8 Consumo idéntico por los verbos | D-01, tabla de costuras | A1 (C1a para `archive-compose`) |
| REQ-24.9 `workspaceRoot`, `planningHome` y autoridad | Invariante B'; `sdd-attempt` sobre `loc.WorkspaceRoot`; cambio archivado con fecha (`ArchivedChangeRoot`, escenario 3); aviso multirepo | A1, B1b |
| REQ-24.10 Consumidores de catálogo | D-07; aviso Q14 en todos (`LegacySpecsDivergence`) | A2 |
| REQ-24.11 Regla de ejecutores | D-17, sección G de `sdd-phase-common` | A1 |
| REQ-24.12 Receta y `pr_merged` | `docs/manual-de-inicio.md` con pin en `docs_contract_test.go`; `verifyGateEvidence` sobre `loc.WorkspaceRoot` | A1 |
| REQ-24.13 Bloque `baseline:` y defecto OpenSpec | D-03, D-04, D-05, D-06 | B1a |
| REQ-24.14 Documentos por dominio | D-04 (unicidad, formato, contención por identidad, existencia) | B1a |
| REQ-24.15 Perfil de requisito | D-04 (`Compile`, nivel, render, identidad) | B1a |
| REQ-24.16 Documentos de prosa e `integration` | D-04 | B1a |
| REQ-24.17 Validación por loader y `workspace validate` | D-03, D-04 (incluida la compatibilidad con el almacén de REQ-24.62 en `workspace validate`) | B1a |
| REQ-24.18 Fallar cerrado | D-03 (planning home determinable; con puntero ilegible, rechazo previo; `ParseErr` nulo sin `axiom.yaml`), D-05 (bloqueo de archive), D-06 (`indeterminate`; los creadores rechazan con el error de la resolución) | B1a, B1b, B2a |
| REQ-24.19 Proyección de `baseline` | D-05, §4.2; contrato del status | B1b |
| REQ-24.20 Status solo aditivo | D-05 (decorador fuera de `baseStatus`) | B1b |
| REQ-24.21 Frontera de configuración | `docs/openspec-config.md` con pin en `docs_contract_test.go`; `axiom.yaml` como única fuente | B1a |
| REQ-24.22 Consulta única | D-06 (`Declarations.Ownership`, en la que delega `BaselineOwnership`; sin respuesta cuando el primitivo rechaza la resolución) | B2a |
| REQ-24.23 Guarda en CLI, dashboard y TUI | D-06 (tabla, andamiaje por perfil), D-07 | B2a |
| REQ-24.24 Guarda en las skills | D-17 (`sdd-init`, `sdd-archive`) | B2b, C3a |
| REQ-24.25 Catálogo informa de la baseline externa | D-07 (`external_baseline`, `ExternalBaselineError`, salidas de la CLI) | B2a |
| REQ-24.26 `technical_context:` con dos modos | D-04 | B1a |
| REQ-24.27 Validación del contexto | D-04 (sintaxis en carga; avisos y enlaces simbólicos en `validate`; error de almacén con `maintained`, REQ-24.62) | B1a |
| REQ-24.28 Proyección de `technicalContext` | D-05, §4.2; `resolveArtifactPaths` sin cambios | B1b |
| REQ-24.29 Lectura por las fases | D-17 (cuatro skills) | B2b |
| REQ-24.30 Rutas de referencia intactas | D-16 (validación del delta), D-15 (c), D-17 (8 skills, incluida `sdd-tasks`) | B2b, C2a, C2b |
| REQ-24.31 Delta de contexto | D-16 | C2a |
| REQ-24.32 Anclas con títulos repetidos | D-16 (anclas) | C2a |
| REQ-24.33 Impacto previsto en `sdd-design` | D-17 | C3a |
| REQ-24.34 Impacto final en `sdd-apply` | D-17 | C3a |
| REQ-24.35 Aviso de `sdd-verify` | D-17 (lista cerrada y sección de avisos) | C3a |
| REQ-24.36 La ausencia nunca se asume | D-05 (bloqueo nativo), D-13 (`prepare`), D-18 (compuerta) | B1b, C2a, C3b |
| REQ-24.37 Rutas mantenidas fuera del repositorio | D-13 (autorización en `prepare`), D-15 (repos afectados) | C2a, C2b |
| REQ-24.38 Compose nativo intacto | D-08 (paquete aparte), modo nativo sin cambios | C1a |
| REQ-24.39 Reconocimiento e identidad | D-09, D-10 | C1a |
| REQ-24.40 IDs duplicados | D-11 (rechazo por ID ambiguo, resto operable) | C1a, C1b |
| REQ-24.41 Frontera y colas | D-09 | C1a |
| REQ-24.42 Delta, render y cuerpo | D-10; cuerpo sin encabezados de nivel ≤ `level`, detectado en tres puntos (D-04, `integrate check` y archive) con dominio, requisito, encabezado, línea y continuación (D-09, D-13) | C1a, C2a |
| REQ-24.43 ADDED con ubicación e ID definitivo | D-11; reaplicación en modo documento y de cambio (D-13) | C1b, C2a |
| REQ-24.44 MODIFIED, REMOVED y RENAMED por ID | D-10 | C1a |
| REQ-24.45 Fidelidad de bytes | D-12 | C1a |
| REQ-24.46 Validación completa antes de escribir | D-12, D-13 (pasos 1–2) | C1a, C2a |
| REQ-24.47 `archive-compose` con `--cwd` y `--document` | D-13 (modos y flags, supersede en modo documento); pin en `sdd-archive/SKILL.md` (D-17) | C1a, C2a, C3a |
| REQ-24.48 Dominio no declarado | D-13 (rechazo en `prepare`, `check` y `archive-compose`) | C1a, C2a |
| REQ-24.49 Precondiciones | D-13 (paso 1), D-15 (`HEAD` y foto previa acotada); registros no declarados fuera del alcance | C2a, C2b |
| REQ-24.50 Integración dirigida | D-13 (paso 3), D-14 | C2a, C3a |
| REQ-24.51 Verificación nativa | D-15 (alcance por identidad; con baseline por defecto, `specs/<dominio>/spec.md` de los dominios del cambio e `INDEX.md`, como fija REQ-24.51 c) | C2b |
| REQ-24.52 Traza por entrada | D-13, §4.5, `integrate show` (también tras archivar) | C2b, C3a |
| REQ-24.53 Modo solo registro | D-13 (`register`), D-15 (c) | C2a, C2b |
| REQ-24.54 Lectura de la baseline por las fases | D-17 (lectura del localizador; `sdd-spec` ejecuta `axiom sdd integrate check`, con pin) | C3a |
| REQ-24.59 Sustitución de IDs provisionales | D-11; D-15 (e) | C1b, C2b |
| REQ-24.60 Clave `scenarios` | D-04 (`include` exige `level` ≤ 3; el error nombra dominio, `level` y las dos salidas) | B1a |
| REQ-24.61 Escenarios `include`/`omit` | D-10 | C1b |
| REQ-24.62 Almacén de ficheros exigido | D-21; D-04 (`workspace validate`, solo el valor declarado); D-05 (almacén efectivo declarado o resuelto; bloquea solo archive; proyección intacta); D-13 (`check` y `prepare`); D-17 (parada de `sdd-archive`); prueba de paridad de lectores | B1a, B1b, C2a, C3a |

### 6.2 Deltas sobre capacidades existentes

| Requisito | Elemento de diseño | Unidad |
|---|---|---|
| sdd-archive-lifecycle REQ-21.16 (M) | Re-enraizado: `pr_merged` contra el repositorio de especificaciones; la parada no toca ni la baseline ni el contexto; documentación | A1, C3b |
| sdd-archive-lifecycle REQ-21.17 (M) | D-13 (tres flujos; `INDEX.md` con `axiom archive sync`; sin roadmaps declarables), D-14, D-15; rama de `sdd-archive` | C2a, C2b, C3a |
| sdd-archive-lifecycle REQ-24.55 (A) | D-13 (contexto en el plan), D-16 (`add` crea dentro de `maintained`), D-15 (`created`) | C2a, C2b |
| sdd-archive-lifecycle REQ-24.56 (A) | D-05 (casos (a), (f) y (g) en el status), D-13 ((a)–(g) en `prepare`), D-09 ((e): frontera de bloque), D-15 ((e): verificación nativa), D-21 ((g): almacén sin ficheros), protocolo de parada de la skill | B1b, C2a, C2b, C3a |
| sdd-archive-lifecycle REQ-24.57 (A) | D-15 (c: rutas mantenidas con «Sin impacto»), traza de la decisión (§4.5) | C2b |
| sdd-block-review-gates REQ-21.9 (M) | D-18 (presentación y remediación) | C3b |
| sdd-block-review-gates REQ-21.11 (M) | D-18 (presentación y remediación) | C3b |
| organic-agent-trigger-rules REQ-24.58 (A) | D-19 | C3b |
| workspace-topology REQ-1.2 (M) | D-04 (`workspace validate`: bloques nuevos, avisos con `COMPLIANT` y error de almacén de REQ-24.62) | B1a |
| workspace-topology REQ-2.1 (M) | D-03, D-04 | B1a |
| workspace-topology REQ-2.3 (M) | D-02 (puntero en desacoplado) | A1 |
| workspace-topology REQ-2.4 (M) | D-02 (canónico multirepo) | A1 |
| living-documentation REQ-1.1 (M) | D-07 (raíz, aviso Q14, baseline externa) | A2, B2a |
| living-documentation REQ-1.2 (M) | D-06 (`Sync`), D-07 (`INDEX.md` en el planning home) | A2, B2a |
| living-documentation REQ-2.1 (M) | D-06 (`ColdStart`), D-07 | A2, B2a |
| living-documentation REQ-3.1 (M) | D-06, D-07 (`--path` como directorio de partida) | A2, B2a |
| living-documentation REQ-3.4 (M) | D-06 | B2a |
| living-documentation REQ-4.1 (M) | D-07 (`external_baseline`, 404 en detalle), D-06 (409 en sync) | B2a |
| living-documentation REQ-4.2 (M) | D-07 (aviso en la pestaña), `app.js` | B2a |
| multi-project-hub REQ-2.2 (M) | D-06 (`init` con baseline del proyecto; validación del punto 4 con el mismo validador de carga; rechazo, sin escribir ni registrar, solo con un puntero propio que el primitivo rechaza —`pointer_broken`, `config_unloadable`, `specs_repository_missing`, `ambiguous_planning_home`— o cuyo planning home difiere de `<ruta>/openspec`, comparado por ruta sin exigir existencia; continuación `axiom init --path "<workspaceRoot>"` o receta) | B2a |
| tui-ui-parity REQ-17.10 (M) | D-07 (TUI), §3.4 | A2, B2a |

### 6.3 Requisitos enmendados en esta revisión

| Decisión | Requisitos | Diseño |
|---|---|---|
| `scenarios: include` exige `level` ≤ 3 | REQ-24.60, REQ-24.42 | D-04, D-09 |
| Cuerpo sin encabezados de nivel ≤ `level`, detectado en la declaración, en `axiom sdd integrate check` y en archive | REQ-24.42, REQ-24.54, REQ-24.56 (e) | D-09, D-13, D-17 |
| Almacén de ficheros exigido: error de validación del valor declarado, archivado bloqueado con la proyección intacta, parada de archive | REQ-24.62 (nuevo), REQ-24.56 (g), REQ-24.17, REQ-24.27, workspace-topology REQ-1.2 | D-21, D-04, D-05 |
| Con puntero, configuración ilegible o base inválida rechaza y no hay respuesta de baseline; sin puntero, status operativo y archivado bloqueado | REQ-24.7, REQ-24.18, REQ-24.22 | D-03, D-06 |
| `axiom init` rechaza solo con un puntero propio no resoluble o hacia otro planning home, y valida las declaraciones del `axiom.yaml` que encuentra | multi-project-hub REQ-2.2 (incluido el punto 4), REQ-24.6 | D-06 |
| Excepción embebida de la deducción por sufijo | REQ-24.5 | D-02 |
| `INDEX.md` en el conjunto permitido con la baseline por defecto | REQ-24.51 (c) | D-15 |

## 7. Estrategia de pruebas

### 7.1 Por capa

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unidad (`internal/workspace`) | Descubrimiento, precedencia, deducción, ambigüedad por identidad, planning home divergente, casos con y sin puntero de D-03; validación acumulada, regla de nivel (mensaje con dominio, `level` y salidas), claves desconocidas, contención y enlaces simbólicos; almacén declarado (`engram`, `none`, `openspec`, `hybrid` y sin declaración, con `config.yaml` y `config.yml`); consulta única con `Declarations.Ownership`, incluido el valor cero sin `axiom.yaml`; `ResolveOwnPointer` (puntero ilegible, destino inexistente, sin `config` ni `specs`, YAML ilegible, base inválida, `workspaceRoot` inexistente, ambigüedad, y planning home inexistente sin error, incluido `multirepo` con `specs_repository: openspec` sin `openspec/`); `SamePlanningHome` (identidad y respaldo léxico); aviso Q14 | Tablas con `t.Run`, fixtures en `t.TempDir()` y `DiscoveryCeilingOverride` |
| Unidad (`internal/kickoff`) | `ArchivedChangeRoot`: nombre exacto, entrada con fecha, varias fechas (gana la más reciente), sufijo parcial que no debe coincidir | Tabla sobre `t.TempDir()` |
| Unidad (`internal/specbaseline`) | Reconocimiento por perfil, vallas, nivel, colas, cuerpos conflictivos, render, notas, `omit`, duplicados, ubicación, familias, relleno, (a)–(d) de REQ-24.59, tokens completos, CRLF, BOM, final sin salto; gramática del delta de contexto y de las anclas | Tablas sobre fixtures sintéticos en tres variantes de bytes |
| Integración (git temporal) | `prepare` (limpio, staged, sin seguimiento, no git, suciedad ajena, más de 1.000 entradas, autorización), `archive-compose --in-place` (todo o nada, reaplicación, huellas), `verify` (a)–(e), commit entre `prepare` y `verify`, repo hermano, enlace simbólico, `core.autocrlf=true`, rama por detrás | `git init` en `t.TempDir()`, entorno git aislado, sin red |
| Integración (status) | Proyección, ausencia de claves sin declaración, bloqueo de archive y canales; proyecto sin `axiom.yaml` (solo `openspec/`) sin claves nuevas, sin notas y sin bloqueo de archive (R1); almacén efectivo de REQ-24.62 (`engram` declarado con el cambio en Engram, `none` declarado con el cambio en disco, sin declaración) con la proyección intacta y la planificación y `apply` sin bloquear; paridad de los dos lectores de `artifact_store`; equivalencia embebida | `Resolve` sobre fixtures; `ProjectStatusV2` + JSON |
| Integración (CLI) | Modos de `archive-compose` (incluidos supersede y «ya integrado» en modo documento); verbos `integrate`, con `check` durante la fase spec (deltas sin confirmar, sin compuerta ni autorizaciones) y su código ≠ 0 ante un encabezado conflictivo; `show` tras archivar con fecha; `workspace validate`; `init` con puntero propio hacia otro planning home (rechazo sin escribir ni registrar, con y sin `--profile=knowledge`), con puntero propio que resuelve `<ruta>/openspec` cuando ese directorio aún no existe (`multirepo` con `specs_repository: openspec`, sin rechazo), con puntero propio roto, hacia un `axiom.yaml` ilegible, hacia un repositorio de especificaciones inexistente y con un `<ruta>/openspec` existente que no es su planning home (rechazo con `pointer_broken`, `config_unloadable`, `specs_repository_missing` y `ambiguous_planning_home`, sin escribir ni registrar), con puntero roto solo en un ancestro (sin rechazo), con baseline del proyecto y con `baseline:` o `technical_context:` inválidos (validación del punto 4: informa de cada violación y sale con `0`); `archive *`, `knowledge *`, `handoff` y `role` desde tres puntos de partida; `gate record` sobre un cambio archivado con fecha | Funciones `Run*`/`run*` sobre fixtures, códigos de salida y mensajes; para `init`, un gestor del Hub sobre un directorio temporal que permite comprobar que el registro no cambia |
| HTTP | `/api/increments`, `/api/archive/specs` (catálogo externo, 404 en detalle), `POST /api/archive/sync` (409) | `httptest` |
| TUI | Carga y tecla `s` con baseline externa | `Model.Update` directo (sin `teatest`, convención del repositorio) |
| Assets | Pins de D-17 y D-18; inventario | `assets_test.go` |
| Documentación | REQ-24.12 y REQ-24.21 | `internal/workspace/docs_contract_test.go` |
| Goldens | Orquestadores y `sdd-init` | Vía `-update` del repositorio y nueva ejecución sin `-update` |

### 7.2 Fixtures

1. **Repositorio embebido.** Se conserva el layout actual de este repositorio. La prueba de equivalencia (REQ-24.4) cubre `specs_repository` en `""`, `.`, `openspec` y `specs`, y un proyecto sin `axiom.yaml` (solo `openspec/`). Compara `actionContext.workspaceRoot`, `planningHome.path` y `planningHome.mode`, y comprueba que no aparecen claves nuevas ni notas y que `dependencies.archive` no se bloquea.
2. **Espacio canónico multirepo** (`workspacetest.Canonical`). `<maestra>/.axiom-workspace` con `config: repo-specs/axiom.yaml`; `repo-specs/` con git, `axiom.yaml` (`multirepo`, `specs_repository: repo-specs`, roles `backend` y `frontend`) y `openspec/changes/inc-99-ejemplo/`; `repo-backend/src` y `repo-frontend/` con git. Variantes: `openspec/` propio en la maestra (ambigüedad), `openspec/` en `repo-backend` (proximidad), puntero roto, puntero a un `axiom.yaml` ilegible, `specs_repository` ausente, cambio archivado como `archive/2026-10-04-inc-98-previo`.
3. **Baseline sintética del proyecto de referencia** (`internal/specbaseline/testdata/reference/`, texto inventado). Reproduce las formas del spike:
   - `01_Requisitos_Funcionales.md` con h1, varias h2, `### RF-NNN: título` con subsecciones h4, IDs duplicados con títulos distintos en dos rangos consecutivos, h3 no-requisito justo tras un requisito y entre dos, transiciones descendentes, valla con encabezados falsos y variante sin salto final;
   - `02_Requisitos_No_Funcionales.md` con `NFR` sin anomalías;
   - documentos de prosa con secciones tituladas por historia o bug, títulos únicos y títulos repetidos;
   - `context/` con títulos repetidos dentro de un mismo fichero y un fichero con BOM;
   - `REGISTRO_INCREMENTOS.md` no declarado.

   Las variantes CRLF y CRLF con BOM se generan en la prueba a partir de la versión LF, para no depender de `.gitattributes`.

### 7.3 R1 y redes de seguridad

| Red | Qué garantiza |
|---|---|
| `openspec_archive_compose_test.go` (10) y `cli/sdd_archive_compose_test.go` (3) sin cambios | La composición nativa no cambia |
| Prueba de equivalencia `archive-compose --cwd --document <d> --delta` frente a `--canonical --delta` sin baseline | Mismos bytes en las fixtures nativas |
| Pruebas existentes de `sddstatus`, `cli`, `kickoff` y `livingdoc` sin cambios en sus aserciones de ruta (REQ-24.4) | Las únicas adiciones son `TestMain` nuevos o el ajuste del `TestMain` existente de `cli` |
| `status_v2_clean_break_test` sin cambios | Contrato del status |
| `bench/` ejecutado aparte | Recorridos con la baseline por defecto |

### 7.4 Higiene de las pruebas frente al checkout (R28, R29)

Con el descubrimiento ascendente, una prueba que resuelve desde el directorio de su paquete llega a la raíz de este repositorio, y una de ellas ya escribe hoy en el checkout. Hay cuatro casos conocidos, que se aíslan en A1 con `t.Chdir(t.TempDir())`, o con un espacio temporal que contenga `openspec/` cuando la prueba lo necesite:

| Prueba | Qué pasaría sin aislar |
|---|---|
| `cmd/axiom/main_test.go` `TestRunSDDStatusJSON` | La raíz del repositorio tiene `.engram/` versionado y `openspec/config.yaml` no declara `artifact_store`, así que `shouldTryEngram` es cierto y `Resolve` ejecuta `engram export`. En CI, sin el binario, la prueba falla; en local consultaría el almacén real del desarrollador |
| `cmd/axiom/main_test.go` `TestRunSDDStatusNonexistentChange` | Lo mismo: el cambio no está activo, así que se consulta Engram antes de responder |
| `internal/tui/model_test.go` `TestGovernanceScreensNavigationAndActions` | La tecla `s` sobrescribiría `openspec/INDEX.md` del repositorio. Se retira además la limpieza `os.RemoveAll("openspec")`, que deja de tener sentido |
| `internal/dashboard/dashboard_test.go` `TestArchiveEndpoints` | Ya ocurre hoy, sin inc-24: `NewService("../..")` apunta a la raíz real, `POST /api/archive/sync` reescribe el `openspec/INDEX.md` versionado y `RenderIndexMarkdown` estampa `time.Now()`. `ci.yml` lo documenta como superficie roja conocida, y su paso de humo compara con el estado previo en lugar de con un árbol limpio. Se aísla en un espacio temporal con su propio `openspec/specs/`. Lo fijó main (#71/#72) y la rama lo adoptó al rebasar |

Sus aserciones no cambian: ninguna depende de rutas. Según el inventario que registra `ci.yml`, `TestArchiveEndpoints` es la única prueba del dashboard que modifica el checkout; las demás que usan `NewService("../..")` las aísla la tarea T3 de la otra sesión (§11), ya fusionada en main (#71/#72), y si alguna escribiera en el checkout, la comprobación nueva lo detectaría en A1.

- Los `TestMain` de `sddstatus`, `cli`, `livingdoc`, `knowledge`, `dashboard`, `tui`, `hub`, `workspace` y `cmd/axiom` fijan `workspace.DiscoveryCeilingOverride = []string{os.TempDir()}` antes de `m.Run`, para que un marcador en un ancestro del directorio temporal no altere los resultados.
- `.github/workflows/ci.yml` añade, tras el paso `Run unit tests` (`go test ./...`), un paso que falla si `git status --porcelain` no está vacío. Detecta cualquier escritura de una prueba en el checkout, en ficheros versionados o nuevos. Sin el aislamiento de `TestArchiveEndpoints` fallaría hoy mismo, así que los dos cambios van juntos en A1 y el paso debe estar en verde en su fusión. Se coordina con la otra sesión, que también aísla pruebas del dashboard (§11).

## 8. Matriz de amenazas

El diseño añade subprocesos `git` de solo lectura y bloques shell en skills, así que la matriz aplica.

| Frontera | Casos adversarios mínimos | Aplicabilidad | Respuesta de diseño | Pruebas RED previstas |
|---|---|---|---|---|
| Rutas con aspecto de documentación | `requirements.txt`, `CMakeLists.txt`, Markdown o MDX ejecutable, `README.sh` | Aplicable, solo clasificación. El aviso de `sdd-verify` clasifica manifiestos y ficheros de build o CI; el verificador E2 clasifica rutas cambiadas por conjuntos declarados. Axiom nunca ejecuta ficheros de la baseline, del contexto ni del cambio | Lista cerrada de patrones en la convención compartida. El verificador decide por identidad de rutas y nunca interpreta contenido como ejecutable; un `README.sh` o un `.mdx` fuera del conjunto permitido es una violación de alcance como cualquier otro fichero | Pin: la lista incluye `requirements.txt` y `CMakeLists.txt`. Verificador: `README.sh` creado bajo una ruta de referencia o no declarada falla (c); un `.md` en `context/` declarado y nombrado en el delta pasa |
| Selección de repositorio git | `git -C`, rutas relativas, rutas absolutas | Aplicable: `prepare` y `verify` ejecutan git en el repositorio de especificaciones, en los repositorios de rutas mantenidas o de referencia y en los de rol | `cmd.Dir` = raíz absoluta de `git rev-parse --show-toplevel` desde una ruta absoluta limpia; nunca `-C` con rutas relativas ni el cwd del proceso; las rutas de git se resuelven contra esa raíz y se comparan por identidad | Ruta mantenida con `..` resuelve al repositorio hermano correcto; ruta absoluta; directorio sin `.git` → parada de REQ-24.49; repositorio anidado tratado como propio; ejecución desde un cwd ajeno no cambia el repositorio elegido; enlace simbólico que sale de la carpeta maestra → error de validación |
| Estado del commit | staged, `commit -a`, índice vacío | Aplicable: precondición de árbol limpio, `HEAD` registrado y foto de suciedad previa | `git status --porcelain=v1 -z --untracked-files=all` con `GIT_OPTIONAL_LOCKS=0` en `prepare`; `git diff --no-renames <head>` más ficheros sin seguimiento en `verify`. Cualquier estado de un documento afectado (índice, árbol o sin seguimiento) es «sucio». Axiom nunca hace `add` ni `commit` | Cambio staged en documento afectado → parada; cambio sin stage → parada; documento afectado sin seguimiento → parada; documento que se creará (REQ-24.55) exento; staged en fichero ajeno → no para y queda en la foto previa; más de 1.000 entradas sucias → parada; commit entre `prepare` y `verify` con una edición fuera de alcance → violación (c); repositorio sin commits → comparación con el árbol vacío |
| Estado del push | Rama con upstream, primer push, refspec explícito | Aplicable solo en lectura: aviso de rama por detrás antes de asignar IDs | `git rev-list --count HEAD..@{upstream}` con referencias locales; nunca `fetch`, `pull` ni `push`; sin upstream o con HEAD desacoplado → nota, no bloquea | Rama por detrás → aviso con el número; sin upstream → nota; HEAD desacoplado → nota; ninguna prueba requiere red |
| Comandos de PR | `--head` explícito, prefijo de entorno, comandos compuestos | N/A: inc-24 no crea, actualiza ni consulta PR. La evidencia `pr_merged` existente solo cambia de repositorio por el re-enraizado (documentado) | — | — |

## 9. Migración y despliegue

- **Sin migración de datos.**
- **Qué resuelve igual que hoy.** `monorepo-embedded` (este repositorio incluido), los directorios sin `axiom.yaml` y las configuraciones `monorepo-decoupled` o `multirepo` cuyo `specs_repository` termina en `openspec` y se usan desde la carpeta maestra (por ejemplo el perfil `knowledge`, o un `axiom init --topology multirepo`, que por defecto escribe `specs_repository: "openspec"` salvo que la detección encuentre otro directorio).
- **Qué cambia aunque no haya puntero ni bloques nuevos.** Un `axiom.yaml` directo en `monorepo-decoupled` o `multirepo` con otro `specs_repository` (por ejemplo `especificacion`). Hoy los verbos resuelven `<cwd>/openspec`; con B' resuelven `<maestra>/especificacion/openspec`. Si `<maestra>/openspec` existe y difiere, el status lo avisa en `notes` (D-02, paso 10) para que los cambios que hubiera allí no queden ocultos sin aviso.
- **Archive con la baseline por defecto.** Gana el paso `axiom archive sync`, que regenera `openspec/INDEX.md` (D-13, REQ-21.17(4)). Es la única diferencia en ese flujo.
- **Adopción por configuración.**
  1. Montar a mano la topología canónica (receta en `docs/manual-de-inicio.md`).
  2. Declarar `baseline:` y `technical_context:`, con `artifact_store` `openspec` o `hybrid` si se integra en ellos.
  3. Ejecutar `axiom workspace validate`.
  4. Para cambios con baseline con perfil, ejecutar `axiom sdd integrate check` desde `sdd-spec`.
- **Compatibilidad de binarios.** Un binario anterior ignora los bloques nuevos (el YAML base no es estricto) y no ve el puntero en los verbos SDD. Un proyecto que adopte inc-24 debe usar el binario nuevo en todo el equipo, y la documentación lo indica.
- **Runtimes sin modelo por llamada.** Se asigna a `sdd-archive` un modelo de nivel de implementación cuando se declara integración dirigida o contexto mantenido (D-14).

## 10. Riesgos y mitigaciones

| ID | Riesgo | Prob. | Mitigación |
|---|---|---|---|
| R1 | Regresión del layout embebido | Media | `Roots` embebido idéntico; prueba de equivalencia; aserciones existentes intactas; compose nativo sin tocar (D-08) |
| R2 | Contrato del status | Baja | Campos `omitempty` fuera de `baseStatus`; `status_v2_clean_break_test` intacto |
| R4 | IDs duplicados | Alta en el proyecto de referencia | Rechazo por ID ambiguo, resto operable (D-11) |
| R5, R6 | Frontera y bytes | Media | D-09 y D-12; fixtures LF, CRLF y BOM |
| R7 | Segunda baseline | Media | Consulta única y guarda en todos los creadores; pruebas de no existencia |
| R8 | Flujo antiguo concurrente | Media | IDs definitivos en archive; aviso de rama por detrás; árbol limpio |
| R9 | Divergencia con upstream | Media | Paquetes solo-fork; costuras pequeñas; convención compartida |
| R10 | Ratchets, pins y goldens | Alta | D-20, con la tabla de primer llamador por unidad |
| R13 | Tamaño | Alta | La estimación revisada (§12) ronda el doble de la propuesta, sobre todo por el verificador E2 y el plan; la agrupación en PR la decide la fase de tareas |
| R14 | Clasificación de RDD por segmento `openspec` | Baja | Sin cambios; RDD es opt-in y SDD no lo lanza |
| R17 | Los ejecutores resuelven contra el cwd | Media | Sección G; prefijo `cd`; spike en Claude Code y OpenCode durante A1 (pendiente) |
| R18 | El recorrido captura el proyecto equivocado | Media | Gana el más cercano; `MarkerDir`; `init` rechaza un puntero propio hacia otro planning home; `axiom sdd status` muestra siempre el planning home resuelto |
| R20 | Modelo insuficiente para la edición dirigida | Media | Elevación en Claude Code (D-14); documentación para otros runtimes; verificador nativo |
| R22–R27 | Riesgos del delta de contexto | Media | Como en la propuesta, más D-05 (bloqueo nativo), D-15 y D-16 |
| R28 | Pruebas que resuelven o escriben contra la raíz del repositorio: dos de `cmd/axiom` ejecutarían `engram export`, una de la TUI escribiría en el checkout y `TestArchiveEndpoints` ya reescribe hoy `openspec/INDEX.md` | Alta si no se corrige | Aislamiento de las cuatro en A1 (§7.4); comprobación de checkout limpio en CI, en verde en la fusión de A1. El aislamiento de `TestArchiveEndpoints` lo fijó main (#71/#72) |
| R29 | Marcadores en ancestros del directorio temporal de un desarrollador | Baja | `DiscoveryCeilingOverride` en `TestMain` |
| R30 | En multirepo, rutas de `tasks.md` relativas a la carpeta maestra se resuelven dentro del repositorio de especificaciones | Media | Sección G; nota del status (D-05) |
| R31 | `LoadConfig` estricto oculta la configuración a otros consumidores (skillregistry, dashboard) con una declaración inválida | Baja | Fallar cerrado por diseño; `workspace validate`; los verbos SDD usan la configuración base de `Location` |
| R32 | Proyectos con almacén Engram y baseline del proyecto o contexto mantenido no pueden archivar | Baja | D-21 con causa explícita en status, validación y skill (REQ-24.62) |
| R33 | Anclas con ` > ` en el título o encabezados setext | Baja | Limitación documentada; rechazo nombrando las apariciones |
| R34 | `core.autocrlf` | Media | Nunca se comparan bytes de un commit con el árbol (D-12, D-15) |
| R35 | `integration.yaml` editado a mano | Baja | Decodificación estricta, huellas e `inputs_sha256`; `verify` falla en (d) o (a) |
| R36 | Conflictos con la sesión concurrente | Media | §11. Resuelto por main: la sesión concurrente ya fusionó y la rama se rebasó sobre `origin/main` (770914de) el 2026-10-09 |
| R37 (nuevo) | El archive por defecto pasa a regenerar `openspec/INDEX.md`, con un diff en cada archivado | Baja | Lo exige REQ-21.17(4); la regeneración es determinista salvo la marca de sincronización |
| R38 (nuevo) | Una compuerta rechazada solo por el delta de contexto no se reabre si la fase olvida la línea de remediación | Media | Procedimiento en la convención, en `sdd-design`, en `sdd-apply` y en la frase del orquestador (D-18) |
| R39 (nuevo) | `init` en un subdirectorio de un workspace (por ejemplo, un repositorio de código) crea, como antes, un proyecto independiente que captura los verbos lanzados desde él; REQ-2.2 lo admite de forma explícita | Baja | Comportamiento previo conservado; `axiom sdd status` desde ese directorio muestra el planning home nuevo; la receta documentada indica inicializar el `workspaceRoot` |
| R40 (nuevo) | Los dos lectores de `artifact_store` (`sddstatus` y `workspace`) divergen | Baja | Prueba de paridad sobre las mismas fixtures (D-04, B1b) |
| R41 (nuevo) | El paso 8 interpreta el caso 4 de REQ-24.7 bajo B': exige `workspaceRoot`, que solo difiere de `<maestra>/<specs_repository>` cuando `specs_repository` termina en `openspec` | Baja | Es la única lectura compatible con el escenario de REQ-2.2 en el que un puntero que resuelve `<directorio>/openspec` no rechaza; pruebas con `specs_repository: repo-specs` (rechaza si falta) y `openspec` (no rechaza); se señala a la fase spec para que el texto lo diga |

## 11. Coordinación con la sesión concurrente (resuelto por main)

> Resuelto: la sesión concurrente ya fusionó en main (T3 como #71/#72, la retirada de skills y el resto) y la rama se rebasó sobre `origin/main` (770914de) el 2026-10-09. La sección se conserva como registro de la coordinación; no cambia ninguna decisión.

| Punto de solape | Riesgo | Cómo se mantiene inc-24 fácil de rebasar |
|---|---|---|
| `internal/assets/skills/**` (retirada de `branch-pr` y `gentle-ai-bench`) | Ficheros distintos, pero comparten la lista de inventario de `assets_test.go` y los goldens | Los ficheros nuevos se añaden en un bloque contiguo al final del grupo de skills SDD. Los goldens se regeneran después de rebasar y nunca se resuelven a mano. B2b, C3a y C3b se rebasan sobre su fusión |
| `skillregistry.ScopeForPath` | Sin solape de código. Interacción: con una declaración inválida, `LoadConfig` falla y el índice pierde las raíces de skills declaradas | Documentado (R31); inc-24 no toca `skillregistry` |
| Pruebas del dashboard aisladas del checkout (tarea T3 de la otra sesión) | Mismo fichero `internal/dashboard/dashboard_test.go`: A1 aísla `TestArchiveEndpoints` y T3 aísla las pruebas del dashboard. También `internal/dashboard/service.go` en A2 y B2a | Quien fusione primero fija el aislamiento de `TestArchiveEndpoints` y el otro rebasa sobre él, sin duplicarlo. La comprobación de checkout limpio de CI debe estar en verde en la fusión de A1: si T3 no ha llegado, A1 aísla `TestArchiveEndpoints` por su cuenta. En `service.go`, los cambios se concentran en el accesor `planningRoot()` y cinco sustituciones en llamadas; las pruebas nuevas van en ficheros nuevos y usarán los ayudantes de T3 cuando estén fusionados. A2 y B2a se secuencian después de T3. Resuelto: T3 ya está en main (#71/#72) y fijó el aislamiento de `TestArchiveEndpoints` |
| doctor | Sin solape (inc-24 no toca `GetDoctorDiagnostics` ni doctor) | — |
| Aislamiento de pruebas y CI | Dos mecanismos paralelos y el mismo `ci.yml` | Acordar un único mecanismo: `DiscoveryCeilingOverride` más `t.Chdir` a un espacio temporal, y una sola comprobación de checkout limpio en `ci.yml` tras `Run unit tests`, la añada quien fusione primero. El paso de humo conserva su comparación con el estado previo; simplificarla queda fuera de inc-24 |

Reglas generales: código nuevo en ficheros nuevos; costuras pequeñas en ficheros compartidos; sin reformateos masivos; goldens al final de cada unidad.

## 12. Unidades de entrega y estimación

Estimaciones sin medir, en líneas de autoría (código, pruebas, assets y documentación; sin goldens ni `.codex/AGENTS.md`):

| Unidad | Alcance | Estimación | Verificación de salida | Reversión |
|---|---|---|---|---|
| A1 | D-01, D-02, D-03 (base), costuras SDD, `ArchivedChangeRoot`, regla G, receta, aislamiento de las cuatro pruebas (incluida `TestArchiveEndpoints`) y CI | 1.150–1.550 | Equivalencia embebida (también sin `axiom.yaml`), canónico desde tres puntos de partida, ambigüedad, cambio archivado con fecha, checkout limpio tras `go test ./...` con la comprobación de CI en verde, ratchets | `git revert`; sin datos |
| A2 | D-07 (raíces y Q14) | 550–850 | Catálogo, dashboard y TUI desde la carpeta maestra; aviso Q14 en todos los consumidores | `git revert` |
| B1a | D-03 (declaraciones), D-04 (incluido el almacén declarado), frontera de configuración | 750–1.050 | Tablas de validación, `workspace validate`, pins de `docs/` | `git revert`; retirar los bloques de `axiom.yaml` |
| B1b | D-05 (incluido el almacén efectivo) | 650–900 | Proyección, bloqueo de archive, canales, `none` declarado con el cambio en disco, paridad de lectores, clean break | `git revert` |
| B2a | D-06, D-07 (baseline externa) | 800–1.150 | Ningún creador escribe `openspec/specs` ni `INDEX.md`; `init` con puntero propio (planning home inexistente, distinto, roto, ilegible, repositorio inexistente, ambiguo), con puntero en un ancestro y con validación del punto 4; HTTP y TUI | `git revert` |
| B2b | Skills de lectura y `sdd-init` | 450–700 | Pins y goldens de `sdd-init` | `git revert` |
| C1a | D-08, D-09, D-10 (sin ADDED), D-12, modo documento | 900–1.300 | Fixtures del spike en tres variantes, R1, REQ-24.38 a 24.48 sin ADDED | `git revert` |
| C1b | D-11, `omit` | 800–1.100 | REQ-24.43, 24.59, 24.61 | `git revert` |
| C2a | D-13, D-16, `check`, `prepare`, modo de cambio | 1.100–1.550 | Plan, precondiciones, `HEAD`, límite de la foto, reaplicación, «ya integrado» | `git revert` |
| C2b | D-15, `verify`, `show` | 800–1.150 | (a)–(e), commit entre `prepare` y `verify`, `autocrlf`, `show` tras archivar | `git revert`; un cambio integrado se revierte con `git revert` en el repositorio de especificaciones |
| C3a | D-14 (skill), D-17 (resto) | 400–650 | Pins de skills | `git revert` |
| C3b | D-18, D-19, documentación, goldens, `.codex/AGENTS.md` | 200–400 | Pins de compuertas, prueba de enrutamiento, goldens | `git revert` |

- **Total:** unas 8.550–12.350 líneas, unas 950–1.150 más que la versión anterior por los hallazgos (búsqueda de archivados, `HEAD` y límite, aviso Q14 común, nota de planning home divergente, pins de `docs/`, regla de nivel, aislamiento de pruebas), por la spec enmendada (almacén efectivo con paridad de lectores, alcance de `init`) y por las correcciones finales (aislamiento de `TestArchiveEndpoints`, puntero propio de `init` y validación del punto 4). Sigue rondando el doble de la estimación de la propuesta (3.250–5.250), y la diferencia se concentra en C.
- **Dependencias:** A1 → A2; A1 → B1a → B1b → B2a → B2b; B1a → C1a → C1b → C2a → C2b → C3a → C3b. B2a necesita A2, y C3a necesita B2b. Tras B1a, la rama C puede avanzar en paralelo a B1b y B2.
- **Agrupación:** la decide la fase de tareas. La validación independiente sugiere separar B1, B2, C1 y C2, que es lo que reflejan las unidades a/b; también es posible agruparlas por rebanada.

## 13. Decisiones tomadas

1. **D-03** (confirmada por el usuario). Los errores de los bloques nuevos nunca hacen rechazar a `Locate`; el status sigue operativo y el archivado se bloquea. Con puntero, un `axiom.yaml` ilegible o con base inválida rechaza, porque el planning home es desconocido. Sin puntero, se conserva la resolución embebida con archivado bloqueado si la causa afecta a las declaraciones o el YAML no se lee.
2. **D-02** (confirmada por el usuario y fijada ahora por REQ-24.5). En `monorepo-embedded` no se aplica la deducción por sufijo.
3. **D-14** (confirmada por el usuario). La elevación automática del modelo de archive solo existe en Claude Code; los demás runtimes dependen de la configuración, con el verificador nativo como red.
4. **D-06** (aceptada; multi-project-hub REQ-2.2 y REQ-24.6). `axiom init` solo mira los marcadores del propio destino y nunca llama a `Locate`.
   - Resuelve el propio `.axiom-workspace` con el primitivo y rechaza, sin escribir ni registrar, con su mismo error ante cualquier causa de REQ-24.7 (`pointer_broken`, `config_unloadable`, `specs_repository_missing`) o ante la ambigüedad de REQ-24.6.
   - Si resuelve, rechaza solo cuando el planning home difiere de `<ruta>/openspec`, comparado por ruta sin exigir que exista ninguno de los dos (`*hub.ForeignPlanningHomeError`). El paso 8 exige `workspaceRoot`, no el planning home.
   - Un puntero en un ancestro no lo impide. La continuación es `axiom init --path "<workspaceRoot>"` o la receta documentada.
   - Valida las declaraciones del `axiom.yaml` que encuentra con el mismo validador de carga (punto 4), informa de cada violación y sale con `0`.
5. **D-21** (aceptada; REQ-24.62 y caso (g) de REQ-24.56). Con baseline del proyecto o contexto mantenido se exige almacén `openspec` o `hybrid`. El almacén efectivo es el declarado y, sin declaración, el resuelto; `workspace validate` solo evalúa el declarado; el status bloquea solo el archivado y sigue proyectando la declaración.
6. **Techo de búsqueda** (cambio pedido por el usuario). Es una variable de prueba, `workspace.DiscoveryCeilingOverride`, sin variable de entorno ni documentación de usuario.
7. **Entrega.** La agrupación en PR la decide la fase de tareas; §12 propone unidades más finas y la estimación revisada.
8. **R17.** El spike de cwd de ejecutores en Claude Code y OpenCode se ejecuta durante A1, antes de cerrarla.
9. **`INDEX.md` y roadmaps.** Con la baseline por defecto, la skill de archive ejecuta `axiom archive sync` entre la integración de las specs y el movimiento del cambio (REQ-21.17(4)). No hay roadmaps declarables, así que archive no actualiza ninguno.
10. **Remediación de compuertas por el delta de contexto.** Se documenta la línea de remediación que reabre la compuerta (REQ-21.12) y no se cambia el digest.
11. **Copias generadas de la guía ODD.** `.codex/AGENTS.md` se regenera en C3b; `.claude/` y `.kiro/` no están versionados.
12. **Cambios archivados.** `kickoff.ArchivedChangeRoot` resuelve el nombre exacto y la entrada con fecha para los verbos de gobierno e `integrate show`.
13. **Nombre del verbo.** Se mantiene `axiom sdd integrate check`, que la spec fija (REQ-24.42, REQ-24.54).
14. **Andamiaje por perfil.** `init` conserva el andamiaje de cada perfil: el perfil `knowledge` no crea `.axiom/inbox/skills/` (`hub/init.go`), así que el inciso de REQ-24.23 «que sigue creando `openspec/changes/` y `.axiom/inbox/skills/`» se lee referido al perfil completo; sus escenarios no lo contradicen.
15. **Declaraciones sin `axiom.yaml`.** El valor cero de `Declarations` significa «sin declaraciones»: `ParseErr` solo se rellena cuando un `axiom.yaml` existente no se puede leer, de modo que un proyecto sin `axiom.yaml` nunca bloquea el archivado.
16. **Aislamiento de `TestArchiveEndpoints` en A1.** La comprobación de checkout limpio de CI se mantiene y debe pasar en la fusión de A1; con la tarea T3 de la otra sesión, quien fusione primero fija el aislamiento y el otro rebasa. Resuelto por main (#71/#72): lo fijó main y la rama rebasó sobre ello.

### Pendientes para la fase de tareas

Notas de la revalidación que esta revisión no corrige; cada línea indica su corrección para que la fase de tareas la planifique:

- **m2.** Definir en C1a y C2a los cinco tipos que §4.3 cita sin definir (`ComposeError`, `DocumentReport`, `ContextReport`, `DirectedItem` y `ContextItem`), con los campos que ya enumeran §4.4 y D-13.
- **m4.** Añadir `internal/workspace/locate.go` al mapa de B1a (`Locate` rellena `Location.Declarations` con `parseLayered`) y asignar los `TestMain` con el techo de pruebas por unidad: A1 (`workspace`, `sddstatus`, `cli`, `cmd/axiom`), A2 (`livingdoc`, `knowledge`, `dashboard`, `tui`) y B2a (`hub`).
- **m6.** Añadir a §9 el cambio con `--cwd` (o el cwd) en un subdirectorio sin marcadores propios: hoy se resuelve `<subdirectorio>/openspec`; con inc-24, el planning home del marcador más cercano (REQ-24.2).
- **m7.** Corregir el recuento de pruebas de composición nativa en D-08 y §7.3: `openspec_archive_compose_test.go` tiene 9, no 10.
- **m8.** Añadir a `ChangeTarget` el almacén efectivo (`ArtifactStore`), que el adaptador de la CLI toma del status, para que `Check` y `Prepare` apliquen REQ-24.62 sin volver a resolverlo.
- **m9.** `workspace.Validate` recibe un `FS` abstracto, mientras que `EvalSymlinks` y `pathidentity` trabajan sobre el sistema de ficheros real: ampliar `FS` o hacer las comprobaciones de identidad fuera de `Validate`, en `runWorkspaceValidate`.
- **m10.** Corregir la justificación de la duplicación del lector de `artifact_store` (D-04): el ciclo se evitaría igual si `sddstatus` llamara a `workspace.DeclaredArtifactStore`; la razón real es R9 (no tocar el código compartido con upstream), con la prueba de paridad como red.
- **m11.** `RenderIndexMarkdown` escribe «Axiom» de forma fija en el título y en «Proyecto» de `INDEX.md`. Antes de que la skill de archive ejecute `axiom archive sync` en otros proyectos, tomar el nombre de `workspace.name` o, sin él, del directorio.
- **m12.** Definir de dónde obtiene el decorador la `Location` cuando `ResolveOptions.WorkspaceRoot` es explícito (por ejemplo, la resolución del propio directorio sin ascender), para que las llamadas internas con raíz explícita no cambien de comportamiento.
- **m13.** Fijar el orden entre `init --force` y la guarda: validar y consultar la propiedad sobre el `axiom.yaml` existente antes de sobrescribirlo, y decidir si `--force` conserva `baseline:` y `technical_context:` o se rechaza cuando existen.
