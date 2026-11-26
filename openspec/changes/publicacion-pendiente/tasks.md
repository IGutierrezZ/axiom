# Tareas: Estado «aprobada sin publicar» en `axiom sdd status` (C-05-c)

Rol único `fullstack` (`kickoff.yaml`). Fuentes: `proposal.md`, `specs/sdd-docs-publication/spec.md` (REQ-27.18 y REQ-27.19) y `design.md`.

Normas de ejecución para todo el plan:

- TDD: en cada comportamiento, primero la prueba que falla (RED observado) y después la producción (GREEN); REFACTOR solo con las pruebas en verde.
- Verificación localizada, nunca la batería completa: no se ejecuta `go test ./...` ni `go test ./internal/cli` entero; en `internal/cli`, `internal/assets` e `internal/components` siempre con `-run`. Tampoco `go build -o` de binarios sueltos. La suite completa la pasa la CI.
- Cada unidad cierra con un único commit convencional, sin atribución a IA, con código, pruebas y documentación juntos.
- No hay umbral de tamaño que gestionar: no se pide `size:exception`.

## Delivery Forecast

| Field | Value |
|-------|-------|
| Estimated size | M |
| Estimated changed lines | 800 a 950 (adiciones + eliminaciones, sin ficheros generados; unas 360 de producción, 450 de pruebas y el resto de documentación) |

### Work Units

| Unit | Goal | Commit | Focused test command | Runtime harness | Rollback boundary |
|------|------|--------|----------------------|-----------------|-------------------|
| U1 | Composición única de la lista cerrada: `docpaths.FromWorkspace` y `PublicationTarget.DocPaths`; `composeSet` delega | `refactor(docpaths): componer la lista cerrada de publicación en un único punto` | `go test ./internal/docpaths/...` y `go test ./internal/branch/ -run 'Publish\|Publication'` y `go test ./internal/sddpublish/...` | N/A: biblioteca; la cubren las pruebas de tabla y las de `publish` existentes | `internal/docpaths/*`, `internal/branch/publication.go`, `internal/sddpublish/publish.go` |
| U2 | Lecturas nuevas del puerto git: `RemoteConfigured` y `DirtyPathsUnder` | `feat(branch): lecturas de solo lectura de remoto configurado y rutas sucias en el puerto git` | `go test ./internal/branch/gitcli/... -run 'RemoteConfigured\|DirtyPathsUnder'` y `go test ./internal/branch/engine/... -run 'TheGitPortHasNoPush'` | Git real en `t.TempDir()` | `internal/branch/ports.go`, `internal/branch/gitcli/ops.go`, `internal/branch/engine/engine.go`, `pending_test.go` |
| U3 | `Engine.PublicationState`: hechos de solo lectura, rango sin subir y archivado | `feat(branch): calcular los hechos de publicación pendiente sin red ni escrituras` | `go test ./internal/branch/engine/... -run 'PublicationState\|PublicationTarget\|PublicationAndHandOff\|PromotionContracts'` y `go test ./internal/sddpublish/...` | Git real con remoto desnudo en `t.TempDir()` | `internal/branch/publication.go`, `internal/branch/handoff.go`, `internal/branch/engine/publication*.go` |
| U4 | Proyección `publication` en `sddstatus` | `feat(sddstatus): proyectar publication pendiente en el status del cambio` | `go test ./internal/sddstatus/... -run 'Publication\|Gate\|Status'` y `go test ./internal/cli -run 'TestPublishIsolation'` | `axiom sdd status --json` en proceso (sin binario suelto) | `internal/sddstatus/{publication,governance_handoff,status,status_v2}.go` |
| U5 | Doctrina del orquestador, contrato del status y goldens | `docs(sdd): ordenar ejecutar la publicación pendiente una vez antes de la fase siguiente` | `go test ./internal/assets/...` y `go test ./internal/components -run TestGolden` | N/A: solo texto; los goldens comprueban el contenido renderizado | `internal/assets/skills/_shared/*.md` y `testdata/golden/*.golden` |

Entrega `single-pr`: los cinco commits forman una sola PR.

## Review Workload Forecast

| Unidad | Producción (líneas) | Pruebas (líneas) | Docs y pins (líneas) | Total aproximado |
|--------|---------------------|------------------|----------------------|------------------|
| U1 | 70 | 80 | 0 | 150 |
| U2 | 60 | 90 | 0 | 150 |
| U3 | 110 | 150 | 0 | 260 |
| U4 | 140 | 130 | 0 | 270 |
| U5 | 0 | 10 | 10 | 20 (más goldens regenerados) |
| **Total** | **380** | **460** | **10** | **~850** |

Previsión de revisión: lectura por commit. Las dos piezas separables, si hiciera falta recortar, son el momento de archivo y `RemoteConfigured` (véase `design.md`).

---

## Fase 1 (U1): Composición única de la lista cerrada

Depende de: nada. Paquetes: `internal/docpaths`, `internal/branch`, `internal/sddpublish`.

- [ ] 1.1 RED: en `internal/docpaths/docpaths_test.go`, tabla de `FromWorkspace(planningRel, change, maintained, reference)`: `planningRel` vacío, `openspec` y `sub/openspec`; rutas declaradas fuera de la raíz se descartan; el `Set` resultante coincide con el que producía el `composeSet` anterior (mismos `Candidates`, `Contains` y `Root`).
  Cubre: REQ-27.18
  Verificación: `go test ./internal/docpaths -run 'FromWorkspace'`
- [ ] 1.2 GREEN: mover `workspaceRelative` y `fromWorkspace` desde `internal/sddpublish/publish.go` a `internal/docpaths/docpaths.go` como `FromWorkspace` (pura, sin importar `workspace`).
  Cubre: REQ-27.18
  Verificación: `go test ./internal/docpaths/...`
- [ ] 1.3 RED y GREEN: en `internal/branch/publication.go`, `PublicationTarget.DocPaths(declarations, change)` desempaqueta `technical_context` y llama a `docpaths.FromWorkspace`; `sddpublish.composeSet` pasa a delegar en `target.DocPaths(location.Declarations, change)` y se eliminan los dos helpers movidos. Prueba de paridad: `DocPaths` da el mismo `Set` que el cálculo previo para monorepo y multirepo. `PublicationTarget` conserva destino, negativas y orden (sin cambio de semántica).
  Cubre: REQ-27.18
  Verificación: `go test ./internal/branch/ -run 'Publish\|Publication'` y `go test ./internal/sddpublish/...`
- [ ] 1.4 Cierre de U1: `go vet ./internal/docpaths/... ./internal/branch/ ./internal/sddpublish/...` y commit `refactor(docpaths): componer la lista cerrada de publicación en un único punto`.
  Cubre: REQ-27.18
  Verificación: `go test ./internal/docpaths/... ./internal/sddpublish/...`

## Fase 2 (U2): Lecturas nuevas del puerto git

Depende de: nada (paralelizable con U1; superficies disjuntas salvo `internal/branch`). Paquetes: `internal/branch`, `internal/branch/gitcli`, `internal/branch/engine`.

- [ ] 2.1 RED: en `internal/branch/gitcli`, pruebas con git real: `RemoteConfigured` con y sin `origin`; `DirtyPathsUnder` expande una carpeta sin seguimiento a ficheros (`--untracked-files=all`), cuenta cambios preparados, sin preparar y borrados, trata como literal una ruta con espacios o `*` (`:(top,literal)`) y no ve nada fuera de las rutas candidatas.
  Cubre: REQ-27.18
  Verificación: `go test ./internal/branch/gitcli/... -run 'RemoteConfigured|DirtyPathsUnder'`
- [ ] 2.2 GREEN: añadir `RemoteConfigured` (`git config --get remote.<r>.url`) y `DirtyPathsUnder` (`status --porcelain=v1 -z --untracked-files=all --no-renames -- :(top,literal)<p>...`, con validación de rutas y `GIT_OPTIONAL_LOCKS=0`) a `branch.Git` (`internal/branch/ports.go`), `gitcli/ops.go` y `gitReader` (`internal/branch/engine/engine.go`); dos stubs en `fakeGit` (`engine/pending_test.go`).
  Cubre: REQ-27.18
  Verificación: `go test ./internal/branch/gitcli/... ./internal/branch/engine/... -run 'RemoteConfigured|DirtyPathsUnder|TheGitPortHasNoPush'`
- [ ] 2.3 Cierre de U2: `go vet ./internal/branch/...` y commit `feat(branch): lecturas de solo lectura de remoto configurado y rutas sucias en el puerto git`.
  Cubre: REQ-27.18
  Verificación: `go test ./internal/branch/gitcli/... -run 'RemoteConfigured|DirtyPathsUnder'` y `go test ./internal/branch/engine/... -run 'TheGitPortHasNoPush'`

## Fase 3 (U3): `Engine.PublicationState`

Depende de: U1 y U2. Paquetes: `internal/branch`, `internal/branch/engine`, `internal/sddpublish` (solo prueba de paridad).

- [ ] 3.1 RED: en `internal/branch/engine/publication_state_test.go`, con git real y remoto desnudo (helpers `handOffMonorepo`, `remoteWithMain`, `plannedWorktree`, `observe`): (1) carpeta del cambio sin seguimiento con solo `.gates.lock` da `DirtyDocs` vacío y con `kickoff.yaml` da `[…/kickoff.yaml]`; (2) commit de documentación sin `origin/<b>` da `Unpushed=1` y `RemoteBranch=false`; (3) commits de `requirements.txt` y de `docs/x.mdx` no declarado dan `CodeCommits`, y uno en `openspec/changes/x/design.md` no; (4) tras subir al remoto desnudo todo queda vacío; (5) sin `origin` da `RemoteMissing`; (6) fusión limpia de `origin/main` sin código; (7) multirepo con el repositorio de specs in situ mide contra `origin/<integración>`; (8) rama de archivado con `Archive: true`; (9) más de 500 commits da `RangeTooBig`; (10) ni `origin/<integración>` ni la local da `RangeUnknown`; (11) documento preparado y documento borrado cuentan como sucios; (12) estado desde un worktree del cambio y desde el repositorio de specs de un multirepo (`GitRoot` absoluto).
  Cubre: REQ-27.18
  Verificación: `go test ./internal/branch/engine/... -run 'PublicationState'`
- [ ] 3.2 RED: solo lectura. El observador del punto único de git no registra `fetch`, `pull`, `ls-remote`, `commit`, `add` ni `push`, y `repositoryState` no cambia tras calcular el estado.
  Cubre: REQ-27.18
  Verificación: `go test ./internal/branch/engine/... -run 'PublicationState.*(ReadOnly|NoNetwork)'`
- [ ] 3.3 GREEN: en `internal/branch/publication.go`, tipo `PublicationState` y constante `PublicationCodeRangeUnknown`; en `internal/branch/handoff.go`, `PublishArchiveInvocation` (`PublishInvocation(...) + " --archive"`); en `internal/branch/engine/publication.go`, extraer `resolvePublication` y dejar `PublicationTarget` como envoltorio sin cambio de semántica; crear `internal/branch/engine/publication_state.go` con `Engine.PublicationState` (rutas sucias con `DirtyPathsUnder`, exclusiones del rango como el paso 5 de `publish`, `CommitFiles(HEAD, ^<excl>...)`, umbral de 500).
  Cubre: REQ-27.18
  Verificación: `go test ./internal/branch/engine/... -run 'PublicationState|PublicationTarget|PublicationAndHandOff|PromotionContracts'`
- [ ] 3.4 Paridad con `publish`: en `internal/sddpublish`, prueba con git real y remoto desnudo (el motor se importa solo desde la prueba): tras un `Run` correcto `PublicationState` no ve nada; tras editar un documento, `DirtyDocs` coincide con `CommitResult.Paths`; `branch.PublicationCodeRangeUnknown == sddpublish.CodeUnpublishedRangeUnknown`; `isolation_test.go` sigue en verde.
  Cubre: REQ-27.18
  Verificación: `go test ./internal/sddpublish/...`
- [ ] 3.5 Cierre de U3: `go vet ./internal/branch/... ./internal/sddpublish/...` y commit `feat(branch): calcular los hechos de publicación pendiente sin red ni escrituras`.
  Cubre: REQ-27.18
  Verificación: `go test ./internal/branch/engine/... -run 'PublicationState|PublicationTarget|PublicationAndHandOff|PromotionContracts'` y `go test ./internal/branch/ -run 'Publish|Publication'`

## Fase 4 (U4): Proyección en `sddstatus`

Depende de: U3. Paquetes: `internal/sddstatus`, `internal/cli` (solo prueba).

- [ ] 4.1 RED: en `internal/sddstatus/publication_test.go`, `publicationDue` en tabla: `checkpointed` con `spec` aprobada (sí); solo `integration` aprobada (no); `continuous` sin `auto` (no); `auto` con `specs` en `done` y 0 completadas (sí); `completed > 0` (no); `verifyReport` en `done` (no); archivado `continuous` con `auto` (sí, con `Archive`); archivado `checkpointed` (no); almacén `engram` o `none` (no); sin cambio seleccionado (no); sin motor (no); sin kickoff (no).
  Cubre: REQ-27.18
  Verificación: `go test ./internal/sddstatus/... -run 'Publication'`
- [ ] 4.2 RED: `projectPublication`, una fila por estado de la tabla de precedencia: `unavailable` (negativa y `git_failed`), `unavailable` con `unpublished_range_unknown`, `pending` con `docs_uncommitted` y motivos adicionales, `code_unpublished` con `range_too_big`, `code_unpublished` con `code_commits` y `codeCommits`, `pending` solo con `branch_ahead`, sin `origin` (ausente), sin nada que confirmar ni subir (ausente, también sin `refs/remotes/origin/<rama>`); `pushCommand` exacto (`git -C <raíz> push origin <b>:<b>`) con una ruta con espacios; sin `invocation` en `code_unpublished`.
  Cubre: REQ-27.18
  Verificación: `go test ./internal/sddstatus/... -run 'Publication'`
- [ ] 4.3 GREEN: crear `internal/sddstatus/publication.go` (`PublicationProjection`, `decoratePublication`, `publicationDue`, `projectPublication`, `publicationPushCommand`); extraer `gatePublicationFrom` de `resolveGatePublication` en `governance_handoff.go` y usarla en ambos sitios (en el archivado, `branch.PublishArchiveInvocation`); campo `Publication *PublicationProjection` con `omitempty` tras `GateQuestion` en `status.go` y `status_v2.go`; llamada a `decoratePublication` en `Resolve` tras `decorateRequirements`.
  Cubre: REQ-27.18
  Verificación: `go test ./internal/sddstatus/... -run 'Publication|Gate|Status'`
- [ ] 4.4 RED y GREEN: integración con el motor real: `checkpointed` con `spec` aprobada y un documento sucio da `pending` con `invocation` byte a byte igual a `gateQuestion.publication.invocation`, la clave en la raíz y no en `governance`; `nextRecommended`, dependencias y motivos de bloqueo idénticos con y sin el campo; tras un `publish` correcto no hay clave; con estado ausente el JSON es byte-idéntico al anterior (la clave `"publication"` no aparece).
  Cubre: REQ-27.18
  Verificación: `go test ./internal/sddstatus/... -run 'Publication'`
- [ ] 4.5 RED y GREEN: extender `TestPublishIsolationNoVerbButPublishPushesCommitsOrFetches` en `internal/cli`: `sdd status --json` proyecta `pending` y no ejecuta `push`, `commit` ni `fetch`; `gate record` sigue sin ejecutar git (REQ-26.5).
  Cubre: REQ-27.18
  Verificación: `go test ./internal/cli -run 'TestPublishIsolation'`
- [ ] 4.6 Cierre de U4: `go vet ./internal/sddstatus/... ./internal/cli` y commit `feat(sddstatus): proyectar publication pendiente en el status del cambio`.
  Cubre: REQ-27.18
  Verificación: `go test ./internal/sddstatus/... -run 'Publication|Gate|Status'` y `go test ./internal/cli -run 'TestPublishIsolation'`

## Fase 5 (U5): Doctrina, contrato y goldens

Depende de: U4. Paquetes: `internal/assets`, `internal/components`, `testdata/golden`.

- [ ] 5.1 RED: pin en `internal/assets/assets_test.go`: «When to publish» ordena ejecutar `publication.invocation` exactamente una vez antes de lanzar la fase siguiente y una vez al reanudar; `code_unpublished` presenta `pushCommand` sin ejecutarlo; `unavailable` presenta el código; el campo nunca cambia `nextRecommended`; `sdd-status-contract.md` documenta el campo `publication` (estados, motivos, campos y omisiones).
  Cubre: REQ-27.19
  Verificación: `go test ./internal/assets -run 'Orchestrator|StatusContract'`
- [ ] 5.2 GREEN: añadir la frase al final del párrafo «When to publish» de `internal/assets/skills/_shared/sdd-orchestrator-sections.md` y una viñeta en `sdd-status-contract.md` (sección *Governance*, tras la de `gateQuestion`), con el texto de `design.md`; sin cambiar REQ-27.12 ni REQ-27.13 ni hacer que `gate record` ejecute git.
  Cubre: REQ-27.19
  Verificación: `go test ./internal/assets/...`
- [ ] 5.3 Goldens: regenerar los 10 ficheros afectados con `go test ./internal/components -run TestGolden -update`, revisar el diff (solo la frase nueva) y volver a ejecutar sin `-update`.
  Cubre: REQ-27.19
  Verificación: `go test ./internal/components -run TestGolden`
- [ ] 5.4 Cierre de U5: `go vet ./internal/assets/... ./internal/components/...` y commit `docs(sdd): ordenar ejecutar la publicación pendiente una vez antes de la fase siguiente`.
  Cubre: REQ-27.19
  Verificación: `go test ./internal/assets/...` y `go test ./internal/components -run TestGolden`

## Fase 6: Verificación final localizada

Depende de: U1 a U5. Sin commit propio salvo corrección de lo que falle.

- [ ] 6.1 Ejecutar, paquete a paquete y sin la batería completa: `go test ./internal/docpaths/... ./internal/branch/... ./internal/sddpublish/... ./internal/sddstatus/...` y `go vet` de los mismos; anotar cada resultado observado en `verify-report.md`.
  Cubre: REQ-27.18
  Verificación: `go test ./internal/docpaths/... ./internal/branch/... ./internal/sddpublish/... ./internal/sddstatus/...`
- [ ] 6.2 Ejecutar los paquetes grandes solo con `-run`: `go test ./internal/cli -run 'TestPublishIsolation'`, `go test ./internal/assets/...` y `go test ./internal/components -run TestGolden`. La suite completa (`go test ./...`) la pasa la CI, no se ejecuta en local.
  Cubre: REQ-27.18, REQ-27.19
  Verificación: los comandos citados en esta misma tarea

## Orden y paralelismo

```
U1 ──┐
     ├──► U3 ──► U4 ──► U5 ──► Fase 6
U2 ──┘
```

- Secuencial por dependencia: (U1, U2) → U3 → U4 → U5.
- Paralelizables entre sí: U1 y U2 (superficies de edición disjuntas; ambas tocan `internal/branch` en ficheros distintos).
- Con un solo escritor (lo previsto), el orden de commits es U1, U2, U3, U4, U5.
