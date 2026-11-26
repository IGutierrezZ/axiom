# Tasks: Marcas de ejecución por entorno de `pruebas.md` publicadas a la integración del repo de specs (C-06-d)

> Cambio `marcas-pruebas`. Rol único `fullstack`. `delivery_strategy` single-pr: un commit por unidad de trabajo en una sola PR. TDD no configurado: cada tarea cierra con pruebas funcionales localizadas por paquete (nunca suite completa ni binarios sueltos). Los casos de la matriz de amenazas llevan su prueba RED antes del código de producción.

## Delivery Forecast

| Field | Value |
|-------|-------|
| Estimated size | XL |
| Estimated changed lines | 5000-6500 (código, pruebas, JS, doctrina; goldens excluidos) |

### Work Units

| Unit | Goal | Commit | Focused test command | Runtime harness | Rollback boundary | 
|------|------|--------|----------------------|-----------------|-------------------|
| U1 | Núcleo puro `marks` + `docpaths.ExactAllowed` | `feat(marks): núcleo puro de marcas de pruebas y ruta exacta permitida` | `go test ./internal/acceptancetests/marks ./internal/docpaths -run 'TestExactAllowed\|Test'` | N/A: paquete puro sin E/S | `internal/acceptancetests/marks/`, `internal/docpaths/` |
| U2 | `Engine.PublishedChangeFile` + `AcceptanceMarksFor` + proyección `acceptanceTests.marks` | `feat(status): proyectar el resumen de marcas de pruebas` | `go test ./internal/branch/engine -run TestPublishedChangeFile` y `go test ./internal/sddstatus -run TestAcceptanceMarks` | Repos git de fixture con remoto desnudo local, en proceso | `internal/branch/engine/published_file.go`, `internal/sddstatus/acceptance_marks.go` y campo `Marks` |
| U3 | `sddpublish`: `Only`, `Replay`, puerto git, códigos nuevos | `feat(sddpublish): publicación acotada y reproducción única` | `go test ./internal/sddpublish -run 'TestOnly\|TestReplay\|TestIsolation'` | Git real con remotos desnudos locales en proceso | Campos nuevos de `Request`/`Result` y ramas `Only` de `publish.go` |
| U4 | Dashboard: puerto, disponibilidad, cerrojo, rutas, DTO, canario | `feat(dashboard): rutas de marca y publicación de marcas` | `go test ./internal/dashboard -run 'TestMarks\|TestEveryDashboardRouteIsClassifiedForTheSecretsCanary'` | `httptest` con `MarksPublisher` falso (sin `t.Parallel` con `freePort`/hub) | `service_marks.go`, `marks_port.go`, rutas en `server.go` |
| U5 | `cli.MarksPublisher` + inyección en `cmd/axiom` | `feat(cli): publicador de marcas inyectado en el dashboard` | `go test ./internal/cli -run 'TestMarksPublisher\|TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign'` | Repo multirepo de fixture con remoto local, en proceso | `MarksPublisher` en `sdd_publish.go` y una línea en `main.go` |
| U6 | UI de la pestaña «Pruebas» | `feat(dashboard): matriz de marcas y publicación en la pestaña Pruebas` | `go test ./internal/dashboard -run TestAcceptanceMarksAssets` | Pruebas de assets sobre el JS embebido | `detalle-pruebas.js` y su prueba |
| U7 | Doctrina y goldens | `docs(sdd): resumen de marcas en archive e integration` | `go test ./internal/components -run TestGolden` | Goldens regenerados con `-update` y reverificados sin él | Tres ficheros de doctrina y `testdata/golden/*` |

## Orden y dependencias

U1 -> (U2 ∥ U3) -> U4 -> (U5 ∥ U6); U7 depende solo de U2 y puede ir tras U2. Secuencia recomendada de apply: U1, U2, U3, U4, U5, U6, U7. Las exportaciones de U1-U4 solo tienen uso real al cerrar U5 (trinquete de código muerto en CI): no se evalúa antes.

## Phase 1: Núcleo puro (U1)

- [x] 1.1 RED: crear `internal/acceptancetests/marks/imports_test.go` (solo `internal/acceptancetests`, `yaml.v3` y librería estándar; sin `os/exec`, sin E/S) y las pruebas de `ExactAllowed` en `internal/docpaths/docpaths_test.go`: acepta solo `pruebas-estado.yaml`; rechaza `design.md`, `internal/x.go`, `../pruebas-estado.yaml`, ruta absoluta y `sub/pruebas-estado.yaml`.
- [x] 1.2 Crear `internal/acceptancetests/marks/{doc,model,codec}.go`: `Entry`, `Result`, `File`, `Parse` (válido/ilegible/no secuencia; entradas inválidas aparte), `Append` (el contenido previo es prefijo exacto; `\n` final garantizado), `Merge(local, remote)` sin duplicados (también entradas idénticas ya en el remoto) y `Pending` (multiconjunto por los ocho campos, orden local). Modificar `internal/docpaths/docpaths.go` con `ExactAllowed`.
  Cubre: prov:KOF:1, prov:PUB:1, prov:PUB:2, RF-PUB-01
  Comprobación: `go test ./internal/acceptancetests/marks -run 'TestParse|TestAppend|TestMerge|TestPending|TestPackageImports'` y `go test ./internal/docpaths -run TestExactAllowed`
- [x] 1.3 Crear `internal/acceptancetests/marks/validate.go`: `Input`, `Rejection`, `ValidateInput` con códigos `marks_case_unknown`, `marks_environment_invalid`, `marks_result_invalid`, `marks_note_invalid`, `marks_note_too_long` (500 runas, sin controles ni UTF-8 inválido), `marks_commit_invalid` (100 runas, `[A-Za-z0-9._/+:#@-]`); prueba de ida y vuelta con notas hostiles (`: `, `#`, comillas, `- `, `yes`, `null`, `<script>`, Unicode, espacios iniciales).
  Cubre: prov:WEB:2, prov:KOF:1
  Comprobación: `go test ./internal/acceptancetests/marks -run 'TestValidate|TestRoundTrip'`
- [x] 1.4 Crear `internal/acceptancetests/marks/summary.go`: `Summarize` y `Summary.Text()` con la semántica de D4 (categorías disjuntas con suma igual al total, `n/a` aparte, vigente por mayor `at` con desempate por orden, total 0 = `sin casos`, huérfanas e inválidas aparte, CRLF, caso repetido cuenta una vez) y texto canónico `pre: 5/5 ok; local: 1/2 ok, 1 caducada`.
  Cubre: RF-KOF-22, prov:KOF:1
  Comprobación: `go test ./internal/acceptancetests/marks -run 'TestSummarize|TestSummaryText'`
- [x] 1.5 Cierre U1: ejecutar las dos líneas de comprobación de 1.2-1.4 y commit `feat(marks): núcleo puro de marcas de pruebas y ruta exacta permitida`.

## Phase 2: Lectura publicada y proyección en status (U2; depende de U1)

- [x] 2.1 RED: `internal/branch/engine/published_file_test.go` (`TestPublishedChangeFile`): con y sin referencia remota, con y sin fichero en ella; sin red ni escrituras.
- [x] 2.2 Añadir `PublishedChangeFile` en `internal/branch/publication.go` y crear `internal/branch/engine/published_file.go` (`Engine.PublishedChangeFile` reutilizando `resolvePublication` y `FileAt`).
  Cubre: RF-KOF-22
  Comprobación: `go test ./internal/branch/engine -run TestPublishedChangeFile`
- [x] 2.3 RED: `internal/sddstatus/acceptance_marks_test.go` (`TestAcceptanceMarks`): proyección en multirepo; ausencia en monorepo, sin fichero y con función inactiva; `nextRecommended`, dependencias, `blockedReasons`, notas y digests idénticos con y sin fichero; `pending` y `pendingUnavailable`; DTO y status con el mismo resumen; criterio de hueco 7: el resumen solo se proyecta con entradas válidas (fichero vacío o solo con entradas inválidas no genera `summary`) y el `archive-report` no incluye sección sin marcas.
- [x] 2.4 Crear `internal/sddstatus/acceptance_marks.go` (`AcceptanceMarksFor`, `AcceptanceMarksResult`, `AcceptanceMarksProjection`); modificar `internal/sddstatus/acceptance_tests.go` (campo `Marks`, decoración solo con multirepo + activo + `pruebas.md` + fichero con entradas válidas; `pruebas-estado.yaml` fuera de `TasksGateArtifactPaths`). La ubicación del cambio se resuelve con `PlanningHome` in situ, nunca con `worktreePath` registrado.
  Cubre: RF-KOF-22, prov:KOF:1
  Comprobación: `go test ./internal/sddstatus -run TestAcceptanceMarks`
- [x] 2.5 Documentar la clave `marks` en `internal/assets/skills/_shared/sdd-status-contract.md` (viñeta de `acceptanceTests`).
- [x] 2.6 Cierre U2: ejecutar las comprobaciones de 2.2 y 2.4 y commit `feat(status): proyectar el resumen de marcas de pruebas`.

## Phase 3: Publicación acotada y reproducción (U3; depende de U1)

- [x] 3.1 RED (amenazas): en `internal/sddpublish` (`TestOnly`) un caso por clase: `design.md`, `internal/x.go`, `../pruebas-estado.yaml`, ruta absoluta y `sub/pruebas-estado.yaml` en `Only` dan `only_path_rejected` sin ninguna llamada git; destino no `InPlace` da `only_requires_in_place`; `Replay` sin `Only` rechazado; worktree obsoleto -> commit en el repo de specs.
- [x] 3.2 RED (estado del commit y del push): otro documento preparado y otro sucio sin preparar quedan intactos en árbol e índice y se avisan en `UncommittedDocs`; `wrong_branch`, `identity_missing`, `remote_missing`; nada sucio con commit de marcas sin subir hace push sin commit nuevo; `other_commits_unpublished` sin crear commit; hook `pre-push` sin `fetch`; `axiom sdd publish` sin `fetch` ni reintento.
- [x] 3.3 RED (reproducción, `TestReplay`): éxito tras segundo clon que sube antes (`Replayed`); doble `non-fast-forward` sin tercer intento y con commit conservado; fallos de R1-R5 con el estado descrito en D10 (puerto falso); `up_to_date` cuando todo estaba ya publicado; sin `rebase`, `+`, `--force` ni `--no-verify`.
- [x] 3.4 Modificar `internal/sddpublish/result.go` (`Request.Only`, `Request.Replay`, `Result.UncommittedDocs`, `Result.Replayed` con `omitempty`; códigos `only_path_rejected`, `only_requires_in_place`, `other_commits_unpublished`, `replay_failed` con etiqueta; `other_commits_unpublished` lleva continuación concreta: el comando `git -C <raíz> log origin/<rama>..HEAD --stat` para ver los commits ajenos y la orden `axiom sdd publish --change <x>` o la subida manual de esos commits antes de repetir; cada código nuevo declara su resolución para el trinquete de rechazos), `ports.go` y `gitport.go` (`Fetch`, `FileAt`, `ResetKeep`) y `publish.go` (ramas `Only` y reintento R1-R6 de D10).
  Cubre: prov:PUB:1, prov:PUB:2, RF-PUB-01, RF-PUB-05, RF-PUB-07
  Comprobación: `go test ./internal/sddpublish -run 'TestOnly|TestReplay|TestIsolation'` y `go test ./internal/gitcommon -run TestArchitecture`
- [x] 3.5 Cierre U3: comprobar que el JSON de `axiom sdd publish` no cambia y commit `feat(sddpublish): publicación acotada y reproducción única`.

## Phase 4: Dashboard backend (U4; depende de U1 y U2)

- [x] 4.1 RED: `internal/dashboard/service_marks_test.go` (`TestMarks…`, sin `t.Parallel`; publicador falso): cada código de D13 y D14; sin escritura en rechazos; solo añadir; dos marcas concurrentes íntegras; `marks_busy`; publicar sin `confirm` sin llamar al puerto ni leer git; `nothing_to_publish` sin llamar al publicador; `identity_missing` nombrando claves y órdenes `git config`; nota hostil literal; DTO del detalle igual al de `axiom sdd status`.
- [x] 4.2 RED (conformidad, huecos de cobertura): (a) `change` con `../x`, separadores o ruta absoluta se rechaza (`marks_body_invalid`) antes de componer `filepath.Join(PlanningHome, "changes", <x>)`, sin tocar disco (hueco 10); (b) disponibilidad con varias condiciones: en monorepo sin publicador inyectado se responde `marks_topology_unsupported` y nunca `marks_publisher_missing` (hueco 9, prov:WEB:4); (c) con `pendingUnavailable` el DTO lleva el rechazo tipado con `code`, `error` y `continuation` además de remoto y rama (hueco 2, prov:WEB:1, prov:WEB:3); (d) `other_commits_unpublished` llega al DTO/respuesta con su continuación concreta (hueco 4).
- [x] 4.3 RED (arquitectura): `TestMarksFilesRunNoGit` (los ficheros de marcas del dashboard no importan `os/exec`, `gitcommon`, `gitidentity` ni `sddpublish`).
- [x] 4.4 Crear `internal/dashboard/marks_port.go` (`MarksPublisher`, `SetMarksPublisher`) y `internal/dashboard/service_marks.go`: `marksAvailability` en el orden de D14 pero con el motivo de topología evaluado antes que `marks_publisher_missing`, `marksLocks` (D7), `RecordMark` (D5, D6, identidad vía puerto, `Append` + `fileprim.ReplaceAtomic`, `at` UTC en milisegundos sin repetir) y `PublishMarks` (D3, D7, D13). Validar `change` antes de componer ruta.
  Cubre: prov:WEB:2, prov:WEB:3, prov:WEB:4, prov:KOF:1, prov:PUB:1
- [x] 4.5 Modificar `internal/dashboard/types.go` (`AcceptanceMarksDTO` con `available`, `unavailable{code,error,continuation}`, `environments`, `cells`, `summary`, `summaryText`, `pending`, `pendingUnavailable`, `remote`, `branch`, `noteMax`, `commitMax`, `error`), `service.go` (campos `marks`/`marksLocks`, reloj inyectable; `incrementAcceptanceTests` rellena `Marks`) y `server.go` (`POST /api/increments/marks` y `/marks/publish`; orden 405, `request_not_local`, 415, 400 con 16 KiB y `DisallowUnknownFields`, `confirm`, disponibilidad, cerrojo, validación, acción). Todo rechazo lleva `code`, `error` y `continuation`.
  Cubre: RF-WEB-17, prov:WEB:1, prov:WEB:2, prov:WEB:3
- [x] 4.6 Modificar `internal/dashboard/secrets_canary_test.go`: añadir ambas rutas a `excludedRoutes` con «mutación (POST): …».
  Cubre: prov:WEB:5
  Comprobación de 4.1-4.6: `go test ./internal/dashboard -run 'TestMarks|TestMarksFilesRunNoGit|TestEveryDashboardRouteIsClassifiedForTheSecretsCanary'`
- [x] 4.7 Cierre U4: commit `feat(dashboard): rutas de marca y publicación de marcas`.

## Phase 5: Publicador CLI e inyección (U5; depende de U3 y U4)

- [x] 5.1 RED: `internal/cli/sdd_publish_marks_test.go` (`TestMarksPublisher`, siempre con `-run`): repo multirepo de fixture con remoto local; identidad ausente; `other_commits_unpublished`; destino no in situ; resultado tipado de rechazos.
- [x] 5.2 Modificar `internal/cli/sdd_publish.go` (no crear otro fichero que importe `sddpublish`): `MarksPublisher`, `NewMarksPublisher`, `MarksPublishRequest`, `MarksPublishResult`, `MarksIdentity` (con `gitidentity.Actor`) y `PublishMarks` (`Only: [marks.FileName]`, `Replay: marks.Merge`, `Branch`). Cada rechazo nuevo declara resolución (`TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign`). La ayuda del verbo no cambia.
  Cubre: prov:PUB:1, prov:PUB:2, prov:WEB:3, RF-PUB-07
- [x] 5.3 Modificar `cmd/axiom/main.go` (~l. 1207): `svc.SetMarksPublisher(cli.NewMarksPublisher())` tras `NewServiceWithHub`.
  Comprobación: `go test ./internal/cli -run 'TestMarksPublisher|TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign'` y `go test ./internal/sddpublish -run TestIsolation`
- [x] 5.4 Comprobar el trinquete de código muerto: todo lo exportado de U1-U4 tiene ya uso real (`marks.*` en `sddstatus`/`dashboard`/`cli`; `PublishedChangeFile` en `sddstatus`; `NewMarksPublisher` y `SetMarksPublisher` en `cmd/axiom`). `internal/idmigration/residues.yaml` no cambia mientras los fixtures usen `RF-`, nunca `REQ-`; si un fixture de `marks` usara `REQ-`, añadir su entrada.
- [x] 5.5 Cierre U5: commit `feat(cli): publicador de marcas inyectado en el dashboard`.

## Phase 6: UI (U6; depende de U4)

- [x] 6.1 RED: `internal/dashboard/marks_assets_test.go` (`TestAcceptanceMarksAssets`): `detalle-pruebas.js` escapa notas y `commit` con `escapeHtml`; envía `confirm: true` y `branch` solo tras `confirmDialog`; no ofrece control en celdas no aplicables.
- [x] 6.2 Modificar `internal/dashboard/assets/js/screens/detalle-pruebas.js` (y su comentario de cabecera): con `available == false` tabla de solo lectura y `stateEmpty` con el motivo; con `available == true` matriz casos x entornos con badge, nota, `commit`, «caducada: <resultado>» y formulario con `maxlength` del DTO; bloque «Publicar marcas» con remoto, rama, número y lista de pendientes; botón con `confirmDialog`, deshabilitado con 0 pendientes o con `pendingUnavailable`; con `pendingUnavailable` o `other_commits_unpublished` se muestra el rechazo tipado con su `continuation`; `toast` con resultado y avisos de `uncommittedDocs`; repintado con el `marks` devuelto.
  Cubre: prov:WEB:1, prov:WEB:3, prov:WEB:4, RF-WEB-17
  Comprobación: `go test ./internal/dashboard -run 'TestAcceptanceMarksAssets|TestMarks'`
- [x] 6.3 Cierre U6: commit `feat(dashboard): matriz de marcas y publicación en la pestaña Pruebas`.

## Phase 7: Doctrina y goldens (U7; depende de U2)

- [ ] 7.1 Modificar `internal/assets/skills/sdd-archive/SKILL.md`: «Step 2c: Acceptance Marks Summary» antes del Step 3; `## Marcas de pruebas` con el texto literal solo con `summary` (entradas válidas); aviso de marcas sin publicar con `pending > 0`; aviso de fichero ilegible con `error`; sin clave, sin sección; nunca bloquea y el fichero se mueve intacto.
  Cubre: prov:ARC:1
- [ ] 7.2 Modificar `internal/assets/skills/_shared/sdd-orchestrator-sections.md`: en `integration`, el `summary` literal y las pendientes como información (no cambia las dos salidas ni la evidencia, no va a `gates.yaml`; `ko`, `blocked`, sin marcar o caducado nunca como limpio); en `continuous`, al resumen final.
  Cubre: prov:GAT:1
- [ ] 7.3 Regenerar goldens: `go test ./internal/components -run TestGolden -update` y verificar sin `-update`: `go test ./internal/components -run TestGolden`.
- [ ] 7.4 Cierre U7: commit `docs(sdd): resumen de marcas en archive e integration` (con `testdata/golden/*`).

## Criterios transversales (verificar en el apply de cada unidad)

- Trinquetes: código muerto al cerrar U5; `TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` con cada código de rechazo nuevo (`only_path_rejected`, `only_requires_in_place`, `other_commits_unpublished`, `replay_failed`, `marks_*`, `identity_missing`); canario de secretos con las dos rutas nuevas; `residues.yaml` solo si un paquete nuevo usa `REQ-`.
- Pruebas de dashboard con `freePort` o hub sin `t.Parallel`. Sin suites completas ni binarios sueltos; en `internal/cli` siempre con `-run`.
- Alcance: los huecos 1, 3, 5, 6, 8 y 11 de la revisión de cobertura quedan como informativos; no se amplía alcance.
