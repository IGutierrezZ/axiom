# ODD: Bloqueos de la prueba de Axiom en un multirepo real

> Documento vivo ODD. Fichero autoritativo: `odd/tasks/bloqueos-prueba-multirepo.md`.
> Espejo de recuperación en Engram: topic `odd/bloqueos-prueba-multirepo/tasks`, proyecto `axiom`.
> **CERRADO el 2026-10-14.** Las cuatro unidades se fusionaron en `main` en PRs independientes, #148–#151. Los worktrees y las ramas de trabajo se han eliminado.
> Se abrió el 2026-10-14 sobre `origin/main` 17d4029e, en la rama `fix/bloqueos-prueba-multirepo`.

## Objetivo

Quitar los defectos que impiden probar Axiom sobre un multirepo real. En esa topología la carpeta maestra no es un repositorio git: contiene `.axiom-workspace` y repositorios hermanos, y puede vivir en rutas con espacios.

## Problema

1. `review assess`, `status` y `start` ejecutan `git init` en cualquier directorio sin git (`internal/reviewtransaction/repository_bootstrap.go:44`), también en una carpeta maestra multirepo y aunque RDD esté apagado. `assess`, documentado como de solo lectura, también lo hace.
2. Seis mensajes de `review_facade.go` y `review_capabilities.go` nombran los IDs del contrato heredado. Con ello empujan al agente al dialecto `gentle-ai`, cuyas transiciones posteriores emiten `gentle-ai review …`.
3. Hay rutas sin comillas en las continuaciones (8 `--cwd %s` crudos, 2 `%q` y 5 sitios más).
4. `pathLikeToken` descarta cualquier ruta con espacios, así que la autoridad de edición falla abierta.

## Restricciones

- El dialecto heredado no se toca: `reviewTransitionCommandTool`, `review_capabilities.go:61-62` y el START directo. Lo fijan los schemas v1 congelados y la decisión F3-a de `odd/tasks/marca-seguimientos.md`. La prosa pasa a `axiom` según D3.
- No se tocan los IDs de protocolo `gentle-ai.*/vN`, los directorios de estado `<git-common-dir>/gentle-ai`, las variables `GENTLE_AI_*`, `gentle-ai-bench`, la marca del lanzador ni `internal/assets/**`.
- El schema publicado `contracts/sdd-integration/v1/schemas/consent.schema.json` (`--cwd \S+`) no se modifica; su límite con rutas con espacios queda documentado.
- Los literales de formato de los rechazos no cambian; solo cambian sus argumentos. Así la línea base del ratchet de rechazos se mantiene.
- Los commits siguen Conventional Commits, sin atribución. El checkout principal no se toca.

## Configuración

- **TDD**: desactivado (`openspec/config.yaml` tiene `strict_tdd: false`). Se hacen comprobaciones funcionales con `go test` dirigido (`-run`) por paquete, más `go vet ./...` y `go build ./...`.
- **RDD**: desactivado (global `off`). No se lanza review nativa.
- **Entrega**: unas 780 líneas en cuatro unidades independientes. Precedente del repo: PRs independientes contra `main`, uno por unidad. Push y PR los decide el usuario.
- **Ruta**: un único writer delegado, que se activa porque hay 2 o más ficheros no triviales y mapeo previo con más de 4 ficheros.

## Tareas

- [x] **T1 · Review no inicializa git en una carpeta maestra.**
  - En `PrepareReviewRepositoryRoot` se rechaza la inicialización si el directorio o un ancestro contiene `.axiom-workspace`, o si algún hijo directo tiene `.git`. El mensaje nombra `axiom review status --cwd <repo>` con rutas entrecomilladas.
  - `review assess` no inicializa nunca.
  - Los llamadores de `cli` envuelven el rechazo en `reviewPreflightError`.
  - Se añaden pruebas y se actualiza la documentación de `assess`.
  - Unas 300 líneas.
- [x] **T2 · Prosa del review con el contrato axiom.** Corrige los mensajes `review_facade.go:26-27,34-35,40,46-47,2169-2170` y `review_capabilities.go:187,190`, que pasan a nombrar `AxiomReviewIntegrationContractV2` y el binario `axiom`. Actualiza `review_status_lineage_test.go:55-60` y `review_start_runtime_identity_test.go`. Unas 100 líneas.
- [x] **T3 · Rutas entrecomilladas en continuaciones.**
  - Los 8 `--cwd %s` crudos, los 2 `%q`, `--path %s` en `sdd_archive_compose.go:213,238` y las concatenaciones de `review_facade.go:1849`, `review_mode.go:175` y `tui/screens/review_store_reset.go:124` pasan a usar `pathquote.Quote` (o `ShellWord` donde ya se usa para comandos máquina).
  - Se añade una prueba con un directorio temporal con espacios por familia.
  - Unas 190 líneas.
- [x] **T4 · Autoridad de edición con rutas con espacios.**
  - En `pathLikeToken` deja de rechazarse por igual todo token con espacios. Un token con espacios se acepta solo si está anclado como ruta: absoluto, `./`, `../`, `~/`, primer segmento existente bajo `workspaceRoot`, o contexto explícito.
  - Se siguen rechazando comandos (palabras con `-` inicial, `&&`, `||`, `;`, `|`, `>`, `<`, `$(`) y `://`.
  - Se añaden filas de regresión para `go test ./...` y `GET /api/health`.
  - Unas 150 líneas.
- [x] **T5 · Reinstalar el binario** desde esta rama, con T1 a T4 incluidas.
  - Hecho el 2026-10-14 a las 02:11 con `go install ./cmd/axiom` desde `dbc5a9d4`.
  - `axiom version` sigue mostrando `v3.5.1 commit:dev` y el binario no incluye la revisión de git, así que la comprobación es de comportamiento: smoke test con `axiom review assess --cwd <carpeta con .axiom-workspace y un repo hijo>`, que termina con salida 1 sin crear `.git`.
- [x] **T6 · PATH de usuario.**
  - Se quita `C:\Users\igutierrezz\AppData\Local\gentle-ai\bin`.
  - También dos entradas filtradas por ejecuciones del bench (`…\Temp\gentle-ai-bench-…\home\AppData\Local\engram\bin`).
  - Antes se guarda una copia del valor anterior y después se comprueba que `gentle-ai` ya no se resuelve.
  - Hecho por el orquestador el 2026-10-14: 3 entradas quitadas, quedan 24. En una shell nueva `gentle-ai` no se resuelve y `axiom` y `engram` apuntan a sus binarios. La copia del valor anterior está en el scratchpad de la sesión (`user-path-backup-2026-10-14.txt`). Los ficheros de `AppData\Local\gentle-ai` no se han borrado.
  - La fuga del bench al PATH real queda propuesta como tarea aparte.

## Criterios de aceptación

- `axiom review assess` y `axiom review status`, ejecutados en una carpeta con `.axiom-workspace` o con repositorios hijos, no crean `.git` y terminan con un rechazo tipado que indica la continuación.
- Siguiendo un flujo negociado con el contrato axiom, ningún mensaje propone `gentle-ai`.
- Las continuaciones con rutas con espacios se pueden copiar y ejecutar tal cual.
- Un `tasks.md` con una ruta con espacios fuera de las raíces autorizadas bloquea con `edit_authority_missing`.
- Comprobaciones verdes: pruebas dirigidas, `go vet ./...` y `go build ./...`.

## Progreso

Todas las tareas están hechas y fusionadas (2026-10-14). Se entregaron en un PR independiente por unidad, cada uno con un cherry-pick de su commit sobre 17d4029e y fusionado con squash.

| Tarea | Commit de trabajo | PR | Merge en `main` | Cambio |
|---|---|---|---|---|
| T1 | `5515385b` | #148 | `1e4a789d` | 8 ficheros, +644/−11 (`size:exception`, aplicada por indicación del propietario) |
| T2 | `66a3da94` | #149 | `40bc2984` | 6 ficheros, +96/−15 |
| T3 | `f0632eb3` | #150 | `74a9028c` | 19 ficheros, +196/−22 |
| T4 | `dbc5a9d4` | #151 | `703aebaa` | 3 ficheros, +182/−3 |

Antes de subirla, cada rama de PR pasó por separado `go build ./...`, `go vet ./...`, `go run ./internal/gofmtcheck`, `bash scripts/deadcode-ratchet.sh`, los ratchets de guardas, rechazos y continuaciones, y sus pruebas dirigidas. En GitHub, el CI de los cuatro PRs quedó en verde. En #151 hubo que relanzar «Verify PR Origin & Authority»: la concurrencia del workflow había cancelado la ejecución más reciente al abrir el PR con etiqueta, y el ruleset de `main` la tomaba como el estado del check obligatorio.

Verificación:
- Writer, por unidad: `go build ./...`, `go vet ./...`, `bash scripts/deadcode-ratchet.sh`, el ratchet de rechazos, `TestEveryNamedReviewContinuationIsStructurallyReal` y pruebas dirigidas por familia, todo en verde.
- Fallos previos, reproducidos también en la base 17d4029e: `internal/tui` `TestRuntimeCatalogDiscoveryIgnoresStaleProjectResults` e `internal/cli` `TestReviewStatusSelectorFreeResolvesRepositoryRoot/safe_symlink` (privilegio de enlaces simbólicos en Windows).
- Evaluación de riesgo: `axiom review assess --base-ref 17d4029e --committed-only` da `medium` (`executable_change`). Con RDD apagado basta la autoverificación del writer más una comprobación del orquestador.
- Comprobación del orquestador: las pruebas dirigidas de T1 en `reviewtransaction` y las de autoridad de edición en `sddstatus`, más `go build ./...`, en verde.

Seguimientos fuera de este ODD:
- La fuga del bench al PATH de usuario está propuesta como tarea aparte.
- El componente `permission` se trata en su propio ODD.
