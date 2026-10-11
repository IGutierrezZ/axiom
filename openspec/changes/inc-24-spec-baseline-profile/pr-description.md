# feat: baseline del proyecto y raíz multirepo para SDD (inc-24-spec-baseline-profile)

> Descripción del PR único. Rama `feat/inc-24-spec-baseline-profile`, cabeza `0ea60f2e` antes del commit de cierre de la fase 8, 61 commits sobre `e6870d49`. Sin `push` ni PR abierto: ambos son decisión del usuario.

## Resumen

Axiom resuelve hoy el `planning home` desde el directorio de partida y asume que el catálogo de especificaciones es del propio repositorio. Este cambio lo generaliza para espacios multirepo y para proyectos cuya baseline de especificaciones vive fuera de `openspec/specs/`:

- **Resolución única del planning home** (`workspace.Locate`, `LocateAt`, `ResolveOwnPointer`): los verbos SDD (`sdd status`, `continue`, `kickoff`, `gate`, `sdd-attempt grant`, `handoff`, `role`) y los consumidores de catálogo (livingdoc, knowledge, dashboard, TUI) resuelven el mismo planning home desde la carpeta maestra, el repositorio de especificaciones o un repositorio de código. El cambio de comportamiento desde un subdirectorio sin marcadores propios (antes `<subdirectorio>/openspec`) está documentado en `docs/manual-de-inicio.md`.
- **Declaración `baseline:` y `technical_context:`** en `axiom.yaml`: decodificación estricta, validación contra el sistema de ficheros y el almacén declarado, proyección aditiva en el status v2 (`baseline`, `technicalContext`) y bloqueo del archivado solo por sus causas. `axiom workspace validate` informa de las declaraciones.
- **Guardas contra una segunda baseline** en los creadores (`init`, archive sync y coldstart, knowledge sweep, query y crawl, dashboard, TUI), con respeto al puntero propio y a `--force`. El catálogo informa de la baseline externa.
- **Composición con perfil de requisito** (`internal/specbaseline`, paquete nuevo): MODIFIED, REMOVED, RENAMED y ADDED por ID, IDs definitivos sustituidos, ubicación explícita y `scenarios: omit`. La composición nativa (`internal/sddstatus/openspec_archive_compose.go`) no se toca: R1 se cumple por construcción.
- **Integración dirigida** (`axiom sdd integrate check|prepare|verify|show`, `archive-compose --change --in-place`): delta de contexto con anclas, lectura git de solo lectura, plan `integration.yaml`, verificador nativo de la edición dirigida y traza.
- **Skills, compuertas y ODD**: rama de integración de proyecto en `sdd-archive`, lectura de la baseline en las skills de planificación, delta de contexto en design, apply y verify, presentación y remediación en las compuertas, cierre de ODD con el contexto técnico mantenido, documentación y goldens alineados.

## `size:exception`

`size:exception` aprobado de antemano por el usuario (estrategia `single-pr`, `chain_strategy: size-exception`).

- **Tamaño real.** Líneas de autoría (adiciones más eliminaciones, sin goldens, sin `.codex/AGENTS.md` y sin `tasks.md`): 34.765 en 162 ficheros, de ellas 5.463 son los artefactos de planificación de la Fase 0. Sin la carpeta del cambio: 29.209 en 143 ficheros, de las que 18.397 son pruebas y fixtures, 435 documentación Markdown y 10.377 código de producción y otros ficheros. La previsión de `tasks.md` era de 8.660 a 12.900 líneas.
- **Justificación.** Las siete unidades dependen en cadena (un único resolver, después declaración, guardas, composición, integración y skills) y cada una deja el árbol verde solo con la anterior. Dividirlas en PR encadenados no reduce la superficie que hay que revisar y el usuario eligió un PR único. La desviación sobre la previsión procede casi toda de las pruebas, que no se recortaron: en el tramo C2, 5.988 de las 10.704 líneas son de pruebas, con repositorios git reales.
- **Cómo se mantiene revisable.** Un commit por unidad de trabajo (61 commits), cada uno con sus pruebas y su documentación, agrupados en siete tramos de revisión ordenados. Se revisa un tramo por sesión, commit a commit. No hay PR encadenados.

## Mapa de tramos de revisión

Cada tramo deja el árbol verde al cerrarse. Orden en la rama: A1, A2, B1, B2, C1, C2, C3.

| Tramo | Inicio | Fin | Depende de | Fuera de alcance | Líneas reales (previsión) |
|---|---|---|---|---|---|
| Fase 0. Artefactos de planificación | `f085c585` | `f085c585` | | Código | 5.463 (no previsto: tarea 0.2) |
| A1. Resolver, verbos SDD, aislamiento de pruebas y CI | `387693a5` | `030830a0` | | Declaraciones (B1), catálogo (A2), composición (C1) | 2.638 (1.150–1.580) |
| A2. Consumidores de catálogo | `62d2290e` | `bd5d3e2b` | A1 | Guardas contra una segunda baseline y baseline externa (B2) | 1.805 (600–930) |
| B1. Declaración, validación y proyección en el status | `93690640` | `90d1f7dc` | A1 | Guardas de creadores (B2), skills (B2b, C3) | 3.544 (1.440–2.050) |
| B2. Guardas nativas y skills de lectura | `c187e998` | `79dba680` | B1 y A2 | Integración de archive y composición (C1 a C3) | 3.090 (1.250–1.960) |
| C1. Composición con perfil, ADDED e IDs | `b707c674` | `27ae649d`, más la corrección `3122d35e` | B1 | Plan, git, verificador y verbo `integrate` (C2) | 6.049; 6.201 con la corrección (1.700–2.400) |
| C2. Plan, `integrate` y verificador | `030e8fac` | `bc1e8d12`, más las correcciones `df05d3ba` y `6b317c06` | C1 | Skills y compuertas (C3) | 10.704; 11.449 con las correcciones (1.900–2.880) |
| C3. Skills de integración, compuertas, ODD y documentación | `2868a9a9` | `2ba1c316`, más `0ea60f2e` | C2 y B2 | Nuevos comportamientos nativos | 943 (620–1.100) |

Notas del mapa:

- B2 y C1 solo dependen de B1 y comparten pocos ficheros; en el PR único se integran como commits secuenciales y su orden es intercambiable.
- Los tramos A1 y A2 tienen un commit cada uno intercalado tras el rebase sobre `main`: `223b2570` (aislamiento de `AXIOM_SDD_STATUS_ENGRAM` en las pruebas de `sdd status`) y `059e0a1b` (ajuste del plan). Ambos están dentro del rango de A2 de la tabla.
- Los rangos se calculan con `git diff --shortstat` entre el fin del tramo anterior y el fin del tramo, sin `testdata/golden`, sin `.codex/AGENTS.md` y sin `tasks.md`.
- Los tramos que superan en más de un 25 % el límite superior de su previsión son A1, A2, B1, B2, C1 y C2. Solo C3 queda dentro. El total real sin la Fase 0 es de 29.370 líneas frente a 8.660–12.900 previstas. No se rehízo nada por tamaño.

## Commits por tramo

SHAs vigentes tras el último rebase local. Las copias previas están en `backup/inc-24-pre-rebase`, `backup/inc-24-pre-rebase-2` y `backup/inc-24-pre-c3b-fixup`.

**Fase 0**

| SHA | Commit |
|---|---|
| `f085c585` | docs(openspec): artefactos de planificación de inc-24 |

**A1**

| SHA | Commit |
|---|---|
| `387693a5` | test: aislar del checkout las pruebas que lo escriben o consultan Engram y comprobar el checkout limpio en CI |
| `0b357cd4` | refactor(workspace): extraer parseLayered de la carga de configuración base |
| `0f655051` | feat(workspace): primitivo Locate con descubrimiento ascendente y enraizado por topología |
| `41a7b26a` | feat(kickoff): resolver cambios archivados con prefijo de fecha |
| `20b9afbb` | feat(sddstatus): resolver el planning home con Locate y avisar de la ubicación divergente |
| `6c619428` | feat(cli): resolver kickoff, gate y attempt con Locate y cambios archivados con fecha |
| `e043621b` | feat(cmd): resolver handoff y role con Locate |
| `468c69e0` | docs: regla de ejecutores, receta del espacio canónico y evidencia pr_merged |
| `030830a0` | docs(spike): completar R17 en Claude Code y cerrar A1 |

**A2**

| SHA | Commit |
|---|---|
| `62d2290e` | feat(livingdoc): resolver el catálogo con Locate y avisar de la ubicación divergente |
| `776b4788` | fix(livingdoc): tomar el nombre del proyecto de INDEX.md de workspace.name |
| `223b2570` | test(cmd): aislar también AXIOM_SDD_STATUS_ENGRAM en las pruebas de sdd status |
| `059e0a1b` | docs(openspec): ajustar el plan de inc-24 tras rebasar sobre main |
| `de5c1740` | feat(knowledge): tomar la raíz de especificaciones del planning home y avisar de la ubicación divergente |
| `8c485d34` | docs(openspec): corregir las referencias a testmain_test.go y SyncIndexForProject en el diseño |
| `b4645f7d` | feat(dashboard): resolver las rutas de openspec con el planning home en cada llamada |
| `f0938a63` | feat(tui): cargar SDD y documentación viva desde el planning home |
| `bd5d3e2b` | docs(openspec): cerrar el tramo A2 en tasks.md |

**B1**

| SHA | Commit |
|---|---|
| `93690640` | feat(workspace): decodificación estricta de baseline y technical_context |
| `a1868291` | feat(workspace): Locate rellena las declaraciones y rechaza un puntero con configuración inutilizable |
| `0c1b96f2` | feat(workspace): validar las declaraciones contra el sistema de ficheros y el almacén declarado |
| `eacda0ed` | feat(cmd): workspace validate informa de las declaraciones y de las advertencias con COMPLIANT |
| `6e72f5f2` | feat(sddstatus): proyectar baseline y technicalContext en el status v2 |
| `d2bb72a4` | feat(sddstatus): bloquear solo el archivado por declaración inválida, delta de contexto ausente o almacén sin ficheros |
| `9d0946a0` | docs(assets): contrato del status con baseline y technicalContext |
| `4925cab9` | refactor(workspace): diferir ReadDeclarations a B2, con su primer llamador |
| `8a82d306` | docs(openspec): cerrar el tramo B1 en tasks.md (3.27) |
| `90d1f7dc` | docs: escribir en inglés la frontera de configuración con axiom.yaml |

**B2**

| SHA | Commit |
|---|---|
| `c187e998` | feat(workspace): consulta única de baseline y guarda contra una segunda baseline |
| `facac323` | feat(hub): init respeta el puntero propio, la baseline del proyecto y --force |
| `e39131d7` | feat(livingdoc): guardas de sync y coldstart y baseline externa en catálogo, dashboard y TUI |
| `8a08a61f` | feat(knowledge): guardas de siembra, enriquecimiento y crawl y salidas de la CLI |
| `0ba8519d` | feat(assets): convención compartida de la baseline del proyecto y guarda de sdd-init |
| `d9ec71c7` | feat(assets): lectura del contexto técnico y prohibición de escribir en rutas de referencia |
| `79dba680` | docs(openspec): cerrar el tramo B2 en tasks.md (4.27) |

**C1**

| SHA | Commit |
|---|---|
| `b707c674` | feat(specbaseline): modelo de documento por líneas y reconocimiento de requisitos con perfil |
| `f52bee5c` | feat(specbaseline): composición con perfil de MODIFIED, REMOVED y RENAMED por ID |
| `d2740f97` | feat(cli): archive-compose con --cwd y --document |
| `9a108f0e` | feat(specbaseline): asignación y sustitución de IDs provisionales |
| `00fe7870` | feat(specbaseline): ADDED con ubicación explícita y scenarios omit |
| `27ae649d` | docs(openspec): cerrar el tramo C1 en tasks.md (5.19) |
| `3122d35e` | fix(specbaseline): separar con una línea en blanco el ADDED insertado tras un bloque sin blancos finales |

**C2**

| SHA | Commit |
|---|---|
| `030e8fac` | feat(specbaseline): delta de contexto y anclas por ruta de encabezados |
| `0bb3451a` | feat(specbaseline): lectura git de solo lectura y plan de integración |
| `84692215` | feat(specbaseline): check y prepare de la integración |
| `2c22c56f` | feat(cli): axiom sdd integrate check y prepare |
| `3bfba6b4` | feat(cli): archive-compose --change --in-place con reaplicación idempotente |
| `0d120b97` | feat(specbaseline): verificador nativo de la integración dirigida |
| `11cfe2d8` | feat(cli): axiom sdd integrate verify y show |
| `bc1e8d12` | docs(openspec): cerrar el tramo C2 en tasks.md (6.27) |
| `df05d3ba` | test(specbaseline): acelerar las pruebas con git real con un repositorio plantilla y ejecución en paralelo |
| `6b317c06` | fix(specbaseline): verify no reescribe el índice de git al comparar con el commit del plan |

**C3**

| SHA | Commit |
|---|---|
| `2868a9a9` | feat(assets): rama de integración de proyecto en sdd-archive |
| `1448eabb` | feat(assets): lectura de la baseline y reglas de IDs en las skills de planificación |
| `943a21d5` | feat(assets): delta de contexto en sdd-design, sdd-apply y sdd-verify |
| `97d037e6` | feat(assets): presentación y remediación de las compuertas por el delta de contexto y modelo de archive |
| `58626491` | feat(agentguidance): cierre de ODD con contexto técnico mantenido |
| `b98c22d0` | docs(manual): baseline del proyecto, modelo de archive por runtime y corrección de INDEX.md |
| `2ba1c316` | docs(openspec): cerrar el tramo C3 en tasks.md (7.21) |
| `0ea60f2e` | docs(codex): añadir a .codex/AGENTS.md el párrafo de compuertas con el delta de contexto |

El commit de cierre de la fase 8 (`docs(openspec): cerrar la fase 8 en tasks.md`) se añade encima de esta lista y solo toca `tasks.md` y este fichero.

## Correcciones posteriores al plan

Decisiones que ni la spec ni el diseño fijaban, tomadas por el usuario durante la aplicación y registradas en la tabla «Decisiones tomadas en esta fase» de `tasks.md`.

| Decisión | Commit | Qué cambia | Alternativa descartada |
|---|---|---|---|
| D-12 separación (C1), 2026-10-10 | `3122d35e` | Un ADDED insertado tras una línea no en blanco recibe una línea en blanco delante. Detrás hereda la serie del bloque de referencia con mínimo 1 salvo al final del fichero. Solo inserta bytes (REQ-24.45). | Heredar 0 blancos delante (bloque pegado, asimétrico) o quitar el mínimo detrás (contradice D-12). |
| Tiempo de pruebas de specbaseline (C2), 2026-10-10 | `df05d3ba` | Repositorio plantilla copiado por prueba, entorno git fijado en `TestMain` y `t.Parallel`. De 496 s (1.270 s medidos antes por el usuario) a 109–189 s en Windows, con la misma lista de 918 pruebas. | Shard propio en el CI de Windows (oculta el coste). |
| Índice de git en verify (C2), 2026-10-11 | `6b317c06` | `git diff` contra el commit del plan se ejecuta con `GIT_INDEX_FILE` sobre una copia temporal del índice. El índice real no se escribe. Una auditoría con git 2.41.0.windows.2 mostró que solo `git diff --name-status` reescribía el índice, incluso con `GIT_OPTIONAL_LOCKS=0`. | Confiar en `GIT_OPTIONAL_LOCKS=0`, o usar `git diff-index` sin refresco (falsos positivos por marcas de tiempo). |
| Regeneración de `.codex/AGENTS.md` (C3), 2026-10-11 | `58626491` y `0ea60f2e` | `axiom sync` de Codex escribe en el HOME aunque se pase `--scope workspace`. Se regeneró con HOME aislado y se llevó al repositorio solo la oración nueva del paso 7. El párrafo de compuertas del commit C3-c4 (2 líneas) que el sync aislado sí genera se añadió después en `0ea60f2e`. Corrige la premisa de D-19 y de la tarea 7.16. | Edición manual de la oración (no prueba que el binario la genere). |

Otros ajustes sin decisión del usuario, registrados en los commits y en `tasks.md`:

- `4925cab9`: `ReadDeclarations` se difiere a B2 para que nazca con su primer llamador y no deje código inalcanzable en B1.
- `00fe7870`: reescrito con `--amend` para dar continuación a cinco rechazos de C1-c3 que rompían el ratchet de refusals de `internal/cli`.
- `97d037e6`: rederivada la constante SHA-256 de los ajustes de Kilocode (`TestKilocodeReviewSettingsMatchCurrentMainBaseline`), porque esos ajustes embeben la sección compartida de compuertas.
- Decisiones del plan (m9, m11, m12, m13, firma de `parseLayered`, erratas): ver la tabla de `tasks.md`; se aplicaron como estaban previstas.

## Verificación realizada y sus límites

Batería ejecutada sobre la cabeza `0ea60f2e`, con `git status --porcelain` vacío al empezar y al terminar.

**En verde**

- `go build ./...`, `go vet ./...`, `go run ./internal/gofmtcheck` (código de salida 0) y `bash scripts/deadcode-ratchet.sh` («no new unreachable functions»).
- `go test ./... -count=1 -p 4` sin `internal/cli`, `internal/reviewtransaction` ni `internal/specbaseline`: 75 paquetes en verde. Los fallos son solo los ambientales conocidos de la base (ver abajo) y, por carga de la máquina, tiempos agotados de `e2e/organicruntime`, `internal/sddstatus` (git de más de dos minutos) e `internal/cli` (que entró por error en esa ejecución), que se repiten aislados en las líneas siguientes.
- `go test ./internal/specbaseline -count=1 -timeout 30m`: verde (136 s).
- `go test ./internal/sddstatus -count=1`: verde (85 s), con las 9 pruebas de composición nativa y `TestSDDStatusV2CleanBreak` sin cambios.
- `go test ./e2e/organicruntime -count=1 -timeout 40m`: verde (551 s), incluida `TestNativeProviderCaptureFailureReoffersTheSameBinding`.
- `internal/cli`: las 791 pruebas definidas fuera de los ficheros `review*_test.go`, en dos tandas por el límite de unos 32 K de la línea de comandos de Windows. Solo fallan las tres pruebas ambientales de `sync_test.go`. Más el ratchet de refusals (`TestEveryProductionRefusal`, `TestRefusalRatchet`, `TestEveryNamedReviewContinuation`), en verde.
- R1: `git diff --stat e6870d49..HEAD` de `internal/sddstatus/openspec_archive_compose.go` y su prueba, vacío; las 9 pruebas nativas y las 3 de `internal/cli/sdd_archive_compose_test.go` pasan sin cambios de aserciones. `internal/reviewtransaction` no se toca (diff vacío).
- Arnés con el binario de la rama: `sdd status inc-24-spec-baseline-profile --cwd . --json` desde la raíz y desde `internal/specbaseline` da el mismo `actionContext.workspaceRoot` y bytes idénticos, también idénticos a los del binario de `e6870d49` desde la raíz (sin claves nuevas); `workspace validate --path .` es `COMPLIANT`. Espacio canónico multirepo montado con la receta del manual, con texto inventado y repositorios git locales sin remoto: `sdd status` desde la carpeta maestra, el repositorio de especificaciones y el código de dos repositorios devuelve bytes idénticos y el mismo planning home, y no escribe en ningún repositorio ni en el HOME.

**Límites**

- **Pruebas del ciclo de revisión de `internal/cli` no completadas en Windows.** Son las 860 pruebas de los ficheros `review*_test.go`, que esta rama no toca. El paquete completo supera los 40 minutos de esta máquina y el filtro `-skip` no las excluye. En una ejecución anterior pasaron 627 antes de interrumpirla, sin ningún fallo. `internal/reviewtransaction` completo tampoco termina en tiempo razonable y la rama no lo toca.
- **Fallos ambientales conocidos de la base**, que fallan igual sin esta rama: `internal/backup` (`TestSnapshotterSkipsUnixSockets`), `internal/components/filemerge` (dos pruebas de junctions), `internal/opencode` (`TestRunCatalogCommandCancelsOverflowingChild`), `internal/pathidentity` (cuatro pruebas), `internal/releasepolicy` (`TestValidateRunMarkerRejectsAMarkerInsideASymlinkedDist`), `internal/semantic` (`TestServiceReindexCodeGraph`), `internal/tui` (`TestRuntimeCatalogDiscoveryIgnoresStaleProjectResults`) y tres pruebas de `internal/cli/sync_test.go` (privilegio de enlaces simbólicos y selección de CodeGraph).
- **Módulo `bench/`.** `go build ./...` y `go vet ./...` pasan. `go test ./...` falla en 19 pruebas (sandbox sin `git` en el PATH, TTY y limpieza de directorios temporales con acceso denegado). Son las mismas 19 que fallan en un `git worktree` temporal de `e6870d49`, y esta rama no toca `bench/`: no son atribuibles al cambio, pero no figuran en la lista de fallos ambientales conocidos.
- **Pruebas de enlaces simbólicos.** Se omiten en Windows por falta de privilegio. Las de `internal/specbaseline` se ejecutaron en WSL con `GOOS=linux` durante C2 (todo el paquete en verde, también con `-shuffle=on`). Las de `internal/reviewtransaction` y `sync_test.go` no se ejecutan aquí.
- `go test -race` no está disponible (Windows sin cgo y WSL sin Go).
- Tiempo de `internal/specbaseline` en Windows: de 136 s a 190 s según la carga de la máquina.

## Estado de las coordinaciones

- **T3 (aislamiento de pruebas del dashboard).** Fusionada en `main`: `ccff924c` (#71) y `7aa6e9bc` (#72), ambos ancestros de `e6870d49`. El tramo A2 se secuencia después de ellos, como pedía el diseño.
- **Retirada de skills de la otra sesión.** Fusionada en `main`: #81, #82, #83, #90 y #91, ancestros de `e6870d49`. `chained-pr` vive ya en `internal/assets/skills/chained-pr`, que es la ruta que usa `tasks.md`.
- **`main` ha avanzado después de rebasar la rama.** Las cifras de esta descripción se calculan contra `e6870d49`, la base del último rebase. La referencia local `origin/main` es ahora `60e75cb2`, diez commits más (#120 a #128, cambios de marca Axiom en textos de agentes, revisión y contrato). La rama está 10 commits por detrás de `main` y 61 por delante. Una fusión de prueba (`git merge-tree`) prevé conflicto de contenido en dos ficheros: `internal/components/sdd/review_ledger_contract_test.go` (la constante SHA-256 de los ajustes de Kilocode) y `testdata/golden/sdd-opencode-multi-settings.golden`. `internal/assets/assets_test.go` y doce goldens más se fusionan solos. Antes de abrir el PR hay que rebasar y regenerar los goldens, y volver a derivar la constante de Kilocode. Esta rama no lo ha hecho: es decisión del usuario.
- Sin `push`, sin PR y sin `fetch` desde esta sesión.
