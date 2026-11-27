## Exploración: puertas-desde-ui (C-06-f)

### Estado actual
- La UI solo lee `gates.yaml`: `internal/dashboard/service.go:425` (`kickoff.LoadGates`) para marcar roles pendientes; `detalle.js` no tiene ninguna noción de puerta.
- El registro vive entero en `runSDDGateRecord` (`internal/cli/sdd_gate.go:55-231`), que mezcla E/S de CLI con la lógica: `splitCommitReviewFlag`, `ParseGateRecordArgs`, `validateCommitReviewShape`, `resolveGovernanceChangeRoot`, `completeGateBaseRef`, `guardChangeWriter` (:80, contexto `archive` si `integration`), `requireHandOffStops` (:89), `requireCommitReviewStops` (:94), `gateArtifactDigest` (:132), `prepareHandOff` (:139), `verifyGateEvidence` (:162), `AppendGate`/`AppendGateChecked` (:171-178), `emitGateIntegrationEvent` (:183, eventos ADO C-07-e), `resolveGatePublish` (:187). Escribe avisos por stderr (`emitActorIdentityWarning`, :182).
- `gate record` NO publica: calcula `publish.invocation` o `publishUnavailable` (`sdd_gate_handoff.go:171-192`, RF-GAT-11/15). `integration` no publica (:172). Con almacén `engram|none` no hay `--hand-off` (:53) ni publicación (:179).
- Publicación en proceso: `sddpublish.Publisher{Git: sddpublish.NewGit(), Targets: engine.Default()}.Run(ctx, Request{CWD, Change, Archive, Branch})` (`sdd_publish.go:104-107`); solo `sdd_publish.go` importa `sddpublish` (RF-PUB-07, comentario :219-220). `MarksPublisher` (`sdd_publish.go:221-267`) es el patrón de puerto: `dashboard/marks_port.go:12-32` importa `cli` y lo recibe por `SetMarksPublisher`; inyección en `cmd/axiom/main.go:1211-1212`.
- Máquina de estados: `kickoff.EvaluateGates` (`machine.go:51`) devuelve `nil` en `continuous` (:52-54): no hay puertas. Orden spec, design, tasks, role-apply:<rol>, integration. `HandedOffTo` en `kickoff/handed_off.go:35`.
- Estado de la parada para presentarla: `sddstatus.Resolve` (`status.go:540`) rellena `GateQuestion` solo si `nextRecommended == "await-gate"` (`status.go:845-858`), con gate, criterios, evidencia, opciones y `publication{remote,branch,prTarget,invocation}`. El dashboard solo importa `sddstatus` para rutas/frescura (`service.go:611,1785,1823`), no para `Resolve`.
- `--commit-review`: solo `--gate tasks`, solo `approved`, nunca con `--evidence-kind` (`sdd_gate_commit_review.go:68-85`), solo `checkpointed` (:90); sin modo equivale a `notify`. Con modo, la pendiente se comprueba bajo cerrojo (`sdd_gate.go:99-107,171-175`).

### Restricciones heredadas (comprobadas)
- Guarda Host/Origin: `server.go:259-272` (`needsLocalGuard`, todo `/api/*`) y `:288-294` (403 `forbidden_origin`); OPTIONS 405 `:296-299`; `isLocalRequest` `server.go:1040`. Las pruebas usan `s.handler()` con Host `127.0.0.1` (`server_test.go:73`).
- Patrón de ruta mutante: `server_marks.go:17-46` (`marksPreflight`: 405, local 403, `application/json` 415, 16 KiB, `DisallowUnknownFields`, un solo objeto) y `service_marks.go:464-515` (`confirm` 422 antes de git, `TryLock` -> `busy`, negativas tipadas 409). Reutilizable casi tal cual.
- Canario: toda ruta nueva de `server.go` debe ir a `excludedRoutes`/`coveredRoutes` con motivo (`secrets_canary_test.go:45-88,118-149`).
- CSP `style-src 'self'` y sin estilos/scripts en línea: `assets_test.go:421` (`TestNoInlineStyleOrScriptInAssets`); diálogos con `confirmDialog`/`promptDialog` (`core/components.js:157,162`, sin `confirm()` nativo); el patrón de botón con aviso es `detalle-pruebas.js:160-163`.
- Inventario de escritores: `writer_inventory_test.go:56` incluye `internal/cli` e `internal/dashboard`; sigue por selector `kickoff.AppendGate` (:63) y NO `AppendGateChecked`. Hoy `internal/cli.runSDDGateRecord` figura guardado (:113). Si el núcleo se extrae a otra función, hay que registrarla en `guardedWriters` (o `via`), o el test falla. El dashboard no debe llamar a `AppendGate` directamente.
- Trinquete de rechazos `TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` (`refusal_resolution_ratchet_test.go:617`): todo `fmt.Errorf` de rechazo nuevo en `internal/cli` debe nombrar la salida o llevar `// refusal:by-design <categoría>: ...`; mover los rechazos existentes cambia sus coordenadas en la línea base.
- Dirección de importaciones: `TestImportsDirection` (`integrations/imports_test.go:100`) solo restringe `internal/integrations`; el dashboard ya importa `cli` y `kickoff`, así que un puerto `dashboard -> cli` es legal.
- Dos efectos al aprobar: `emitGateIntegrationEvent` solo vive en `sdd_gate.go:183` (`integrations_events.go:190-201`, spec/design/role-apply; tasks e integration no emiten). Mientras el núcleo único lo dispare una vez, no se duplica; la clave de idempotencia lleva huella+ordinal (`integrations_events.go:168-171`).

### Enfoques
1. **Extraer el núcleo de `runSDDGateRecord` a `cli.RecordGate(ctx, GateRecordRequest) (GateRecordOutcome, error)` y puerto `dashboard.GatesGateway`** (Stop, Record, Publish) implementado en `cli` e inyectado desde `main.go`. El verbo CLI lo llama y solo formatea.
   - Pros: una única lógica (digest, guarda A, cerrojo, evento ADO una vez); sin duplicar reglas; mismo patrón que `MarksPublisher`/`IntegrationsGateway`.
   - Contras: refactor de una función de 175 líneas con muchos tests; mueve rechazos (línea base del trinquete); inventario de escritores a actualizar; los avisos de stderr pasan a campo del resultado.
   - Esfuerzo: M.
2. **El dashboard ejecuta `app.RunArgs(["sdd","gate","record",...])` en proceso** (como la sync, `service.go:1304,1326`).
   - Pros: cero refactor, misma lógica por construcción.
   - Contras: parseo de texto/JSON, `--actor`/identidad y avisos por stderr difíciles, errores no tipados, el resultado de la publicación hay que invocarlo aparte; frágil y poco comprobable.
   - Esfuerzo: S, deuda alta.
3. **Reimplementar en el dashboard con `kickoff.AppendGateChecked`.** Contras: duplica digest, guarda, evidencia, hand-off y evento. Descartado (viola el criterio de no duplicar).

### Recomendación
Enfoque 1, en dos puertos pequeños dentro de `cli`: `RecordGate` extraído (con `loc`/`changeRoot` ya resueltos, devuelve decisión, `publish`/`publishUnavailable`, `lastRoleClosed`, `pendingNonBlocking` y avisos) y `PublishChange` junto a `MarksPublisher` en `sdd_publish.go` (sin `Only`/`Replay`, con `--branch` atado a la rama mostrada). Dos rutas separadas, ambas POST bajo `marksPreflight`: `POST /api/increments/gate` (registra; `rejected` exige `reason`; `integration` exige evidencia y solo dos salidas) y `POST /api/increments/gate/publish` (exige `confirm: true`, `branch` visto en pantalla, aviso de operación remota en `confirmDialog`). La UI encadena registrar y publicar tras un único clic de confirmación en las salidas 2 y 3; un fallo de publicación nunca deshace la aprobación y se muestra tipado con la orden `pushCommand`/continuación (RF-GAT-15). `GET` de la parada: puerto `Stop` sobre `sddstatus.Resolve` para no recalcular `nextRecommended`; en `continuous`, o sin `gateQuestion`, la UI no ofrece botones. Mostrar `commit-review` solo en tasks/checkpointed.

### Riesgos
- Doble efecto: ADO (spec, design, role-apply) debe dispararse una vez; el núcleo único lo garantiza, y una prueba debe contar un único `Dispatch` por aprobación.
- Carrera entre UI y CLI/agente: la comprobación de «puerta aún pendiente» bajo cerrojo hay que aplicarla siempre (no solo con hand-off) para evitar aprobaciones ocultas (RF-GAT-09).
- Rama equivocada: la guarda A debe ejecutarse también desde el dashboard; el inventario de escritores falla si falta el registro del nuevo escritor.
- Publicar es remoto (commit+push): confirmación explícita, sin fetch, nunca forzado; `code_commits_unpublished` es informativo.
- Dos pasos no atómicos (registrar, publicar): estado «aprobada sin publicar» debe verse y poder reintentarse (relacionado con C-05-c, `publication.state: pending`).
- Dependencia: C-02-e/«modo de revisión por commit» (cambios de paradas de la sección 02) ya existe (`--commit-review`), pero conviene confirmar el orden de merge con C-08-c/C-01-e, que mueven primitivas de `reviewtransaction` y reescriben C-06-f.
- Requisitos nuevos de spec: `prov:WEB:<n>` en `local-web-dashboard` y, si cambia el contrato del núcleo, `prov:GAT:<n>` en `sdd-block-review-gates`.

### Listo para propuesta
Sí. Preguntas funcionales aún abiertas para la propuesta: (a) ¿se ofrecen también las puertas `role-apply` de varios roles desde la UI o solo la primera abierta?; (b) evidencia de `integration` (pr_merged/deployment/attestation) en la UI: ¿entra en este cambio o se deja la integración solo en CLI?; (c) ¿el modo `--commit-review` se elige en la UI al aprobar tasks?
