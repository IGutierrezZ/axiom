# ODD: Bloqueos de la prueba de Axiom en un multirepo real

> Documento vivo ODD. Fichero autoritativo: `odd/tasks/bloqueos-prueba-multirepo.md`.
> Espejo de recuperación en Engram: topic `odd/bloqueos-prueba-multirepo/tasks`, proyecto `axiom`.
> Worktree `C:\repos\axiom-wt\bloqueos-prueba-multirepo`, rama `fix/bloqueos-prueba-multirepo`, base `origin/main` 17d4029e (abierto el 2026-10-14).

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
- [ ] **T2 · Prosa del review con el contrato axiom.** Corrige los mensajes `review_facade.go:26-27,34-35,40,46-47,2169-2170` y `review_capabilities.go:187,190`, que pasan a nombrar `AxiomReviewIntegrationContractV2` y el binario `axiom`. Actualiza `review_status_lineage_test.go:55-60` y `review_start_runtime_identity_test.go`. Unas 100 líneas.
- [ ] **T3 · Rutas entrecomilladas en continuaciones.**
  - Los 8 `--cwd %s` crudos, los 2 `%q`, `--path %s` en `sdd_archive_compose.go:213,238` y las concatenaciones de `review_facade.go:1849`, `review_mode.go:175` y `tui/screens/review_store_reset.go:124` pasan a usar `pathquote.Quote` (o `ShellWord` donde ya se usa para comandos máquina).
  - Se añade una prueba con un directorio temporal con espacios por familia.
  - Unas 190 líneas.
- [ ] **T4 · Autoridad de edición con rutas con espacios.**
  - En `pathLikeToken` deja de rechazarse por igual todo token con espacios. Un token con espacios se acepta solo si está anclado como ruta: absoluto, `./`, `../`, `~/`, primer segmento existente bajo `workspaceRoot`, o contexto explícito.
  - Se siguen rechazando comandos (palabras con `-` inicial, `&&`, `||`, `;`, `|`, `>`, `<`, `$(`) y `://`.
  - Se añaden filas de regresión para `go test ./...` y `GET /api/health`.
  - Unas 150 líneas.
- [ ] **T5 · Reinstalar el binario** desde esta rama, con T1 a T4 incluidas, y comprobar `axiom version`.
- [ ] **T6 · PATH de usuario.**
  - Se quita `C:\Users\igutierrezz\AppData\Local\gentle-ai\bin`.
  - También dos entradas filtradas por ejecuciones del bench (`…\Temp\gentle-ai-bench-…\home\AppData\Local\engram\bin`).
  - Antes se guarda una copia del valor anterior y después se comprueba que `gentle-ai` ya no se resuelve.

## Criterios de aceptación

- `axiom review assess` y `axiom review status`, ejecutados en una carpeta con `.axiom-workspace` o con repositorios hijos, no crean `.git` y terminan con un rechazo tipado que indica la continuación.
- Siguiendo un flujo negociado con el contrato axiom, ningún mensaje propone `gentle-ai`.
- Las continuaciones con rutas con espacios se pueden copiar y ejecutar tal cual.
- Un `tasks.md` con una ruta con espacios fuera de las raíces autorizadas bloquea con `edit_authority_missing`.
- Comprobaciones verdes: pruebas dirigidas, `go vet ./...` y `go build ./...`.

## Progreso

T1 hecha. Pendiente: T2.
