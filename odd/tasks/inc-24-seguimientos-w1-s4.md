# ODD: seguimientos de inc-24 (W1 y S4)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/inc-24-seguimientos-w1-s4.md`, en la rama `test/inc-24-seguimientos-w1-s4` (worktree `C:\repos\axiom-wt\inc-24-seguimientos-w1-s4`, creado desde `origin/main` `c0258679` el 2026-10-12).
> Espejo de recuperación en Engram: topic `odd/inc-24-seguimientos-w1-s4/tasks`, proyecto `axiom`.
> Abierto en la sesión de cierre de inc-24; **se ejecuta desde otra sesión** (decisión del usuario).

## Objetivo

Cerrar los dos seguimientos que la verificación global de inc-24 dejó abiertos (`openspec/changes/archive/2026-10-12-inc-24-spec-baseline-profile/verify-report.md` y `archive-report.md`):

- **W1.** REQ-24.12 y REQ-21.16 están parciales solo en prueba: ninguna prueba monta el espacio canónico multirepo y comprueba que la evidencia `pr_merged` se verifica contra el repositorio de especificaciones.
- **S4.** Las pruebas que cubren REQ-24.16 y REQ-24.44 no llevan la etiqueta `REQ-`, así que la trazabilidad por `rg` no las encuentra.

## Problema y porqué

El comportamiento ya es correcto por construcción: `internal/cli/sdd_gate.go` toma `loc.WorkspaceRoot` y se lo pasa a `verifyGateEvidence` (`internal/cli/sdd_gate_evidence.go:42`), que comprueba la ascendencia con el grafo de ese repositorio (`kickoff.VerifyIntegrationEvidence`, `internal/kickoff/integration.go:44`). Falta la prueba que lo fije. Sin ella, una regresión en la resolución del `workspaceRoot` desde la carpeta maestra aceptaría o rechazaría la evidencia equivocada sin que nada falle.

Escenario de la spec (`openspec/specs/planning-home-resolution/spec.md`, REQ-24.12, escenario «`pr_merged` se verifica contra el repositorio de especificaciones»):

- **DADO** el espacio canónico, un commit `S` fusionado en la rama principal de `repo-specs` y un commit `C` presente solo en `repo-backend`.
- **CUANDO** se ejecuta `axiom sdd gate record --cwd <maestra> --change inc-99-ejemplo --gate integration --decision approved --evidence-kind pr_merged --commit S`.
- **ENTONCES** la evidencia se acepta.
- **Y** con `--commit C` la evidencia se rechaza, porque `C` no está en el grafo del repositorio de especificaciones.

Espacio canónico (fixture de la spec): `<maestra>` sin versionar, con `.axiom-workspace` (`config: repo-specs/axiom.yaml`); `<maestra>/repo-specs`, repositorio git con `axiom.yaml` (`topology: multirepo`, `specs_repository: repo-specs`) y `openspec/changes/`; y los repositorios de código hermanos `<maestra>/repo-backend` y `<maestra>/repo-frontend`.

## Alcance autorizado

- Solo pruebas y comentarios de pruebas. No cambia código de producción. Si la prueba de W1 revelase un defecto real, se detiene y se informa al usuario antes de tocar producción.
- Ficheros previstos: `internal/cli/sdd_gate_test.go` (o un fichero nuevo junto a él), `internal/workspace/declarations_test.go` y `internal/specbaseline/compose_test.go`.
- Fuera de alcance: los límites de las pruebas en Windows (ciclo de revisión de `internal/cli`, `internal/reviewtransaction`, `bench/`). Se tratan aparte.

## Restricciones

- Commits en castellano, Conventional Commits y **sin `Co-Authored-By` ni atribución de IA**.
- El checkout `C:\repos\axiom` es de otra sesión: no se toca. Se trabaja solo en este worktree.
- Pruebas git con `t.TempDir()`, entorno git aislado (HOME, GIT_CONFIG_GLOBAL, GIT_CONFIG_NOSYSTEM, autor fijo) y sin red. Respetar el `TestMain` con techo de descubrimiento que ya tiene `internal/cli`.
- Fixtures con texto inventado.
- En `internal/cli`, ejecutar con `-run` focalizado: el paquete completo no termina en Windows.

## Modo de trabajo

- **TDD:** modo estándar. Fuente: `openspec/config.yaml` (`strict_tdd: false`). Runner: `go test`. W1 prueba un comportamiento que ya existe, así que se espera verde desde el principio. Para comprobar que la prueba sirve, se hace una comprobación de mutación local que no se commitea: pasar a `verifyGateEvidence` un repositorio distinto y ver que la prueba falla.
- **RDD:** `axiom review mode status` da `off (decided by global)` el 2026-10-12. No hay revisión nativa.
- **Entrega:** previsión de 150–250 líneas de autoría, por debajo de 400. Un solo PR con `type:chore`; no hace falta `size:exception`. Push, PR y merge los decide el usuario.

## Tareas

- [ ] **T1 · W1: `pr_merged` contra el repositorio de especificaciones en el espacio canónico**
  - Ruta: delegado (writer único). Disparador: preparar la escritura exige leer los helpers de `sdd_gate_test.go` (`newGovernanceGitWorkspace`, línea 280; casos `pr_merged` hacia las líneas 303-360) y la resolución de `workspace.Locate`.
  - Montar el espacio canónico de la spec con dos repositorios git reales: `repo-specs` con el cambio `inc-99-ejemplo` sellado y un commit `S` en su rama principal, y `repo-backend` con un commit `C`.
  - Desde `--cwd <maestra>`: con `--commit S` se acepta y `gates.yaml` anexa el registro; con `--commit C` se rechaza, `gates.yaml` no anexa nada y el mensaje nombra la causa.
  - Etiquetar la prueba con `REQ-24.12` y `REQ-21.16`.
  - Comprobación de mutación local (no se commitea) y registro de su resultado aquí.
  - Commit: `test(cli): evidencia pr_merged contra el repositorio de especificaciones en el espacio canónico`.
  - Checks: `go test ./internal/cli -run 'SDDGateRecord|PlanningHome|TestEveryProductionRefusal|TestRefusalRatchet' -count=1`, `gofmt -l internal/cli`.
- [ ] **T2 · S4: etiquetas REQ-24.16 y REQ-24.44**
  - Ruta: en línea (cambio mecánico, solo comentarios).
  - Añadir `REQ-24.16` al comentario de `TestDeclarationsBaselineValidation` (`internal/workspace/declarations_test.go:177`) o a los casos que validan `integration: directed|register` en documentos de prosa. Comprobar que esos casos lo cubren de verdad; si no, se informa y no se etiqueta.
  - Añadir `REQ-24.44` a las pruebas de MODIFIED, REMOVED y RENAMED por ID de `internal/specbaseline/compose_test.go` (por ejemplo, `TestComposeModifiedReplacesOnlyTheBlockAndDropsTheNotes`, `TestComposeRenamedWithIDIdentityIsRejected` y `TestComposeRenamedWithTextIdentityRetitlesTheBlock`).
  - Commit: `test: etiquetar con REQ-24.16 y REQ-24.44 las pruebas que los cubren`.
  - Checks: `rg -n "REQ-24.16|REQ-24.44" --glob '*_test.go' internal` encuentra las etiquetas; `go test ./internal/workspace ./internal/specbaseline -count=1 -timeout 30m`; `gofmt -l internal/workspace internal/specbaseline`.
- [ ] **T3 · Cierre**
  - `go build ./...`, `go vet ./internal/cli/... ./internal/workspace/... ./internal/specbaseline/...` y `git status --porcelain` vacío.
  - Actualizar este documento y su espejo en Engram con la evidencia y los SHAs.
  - Preparar el PR (`type:chore`) y dejar push, PR y merge a decisión del usuario.

## Criterios de aceptación

- Existe una prueba que monta el espacio canónico y demuestra que `pr_merged` acepta `S` y rechaza `C` desde la carpeta maestra, sin anexar nada en el rechazo.
- La comprobación de mutación muestra que esa prueba falla si la evidencia se verifica contra otro repositorio.
- `rg` encuentra REQ-24.16 y REQ-24.44 en pruebas que de verdad los cubren.
- No cambia código de producción.

## Progreso

- 2026-10-12: documento abierto (3 tareas). Pendiente de ejecución desde otra sesión.

## Siguiente paso

T1.
