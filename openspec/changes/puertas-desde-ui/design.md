# Diseño: Registrar decisiones de puerta desde el dashboard (C-06-f)

## Enfoque técnico

Se aplica el enfoque 1 de la propuesta: **un único núcleo de registro en `internal/cli`, expuesto al dashboard mediante un puerto**, sin reimplementar ninguna regla en `internal/dashboard`.

1. **Refactor sin cambio de comportamiento.** El cuerpo de `runSDDGateRecord` (`internal/cli/sdd_gate.go:55-231`) pasa a `RecordGate(ctx, GateRecordInput) (GateRecordOutcome, error)` en un fichero nuevo `internal/cli/sdd_gate_core.go`. El verbo CLI analiza los argumentos como hoy, llama al núcleo, vuelca los avisos a sus sumideros de siempre y formatea texto o `axiom.sdd-gate-record/v1` (prov:GAT:1).
2. **Publicación reutilizable.** `PublishChange` se añade a `internal/cli/sdd_publish.go`, junto a `MarksPublisher`, sobre el mismo `sddpublish.Publisher` y con la misma guarda A que `RunSDDPublish`. `sdd_publish.go` sigue siendo el único importador de `sddpublish` (RF-PUB-07) (prov:WEB:3).
3. **Puerto `dashboard.GatesGateway`** (`Stop`, `Record`, `Publish`), implementado por `cli.GatesGateway` e inyectado desde `cmd/axiom/main.go` con `svc.SetGatesGateway(cli.NewGatesGateway())`, igual que `SetMarksPublisher` y `SetIntegrationsGateway`. `Stop` se apoya en `sddstatus.Resolve`, así que la parada es la misma que proyecta `axiom sdd status` (prov:WEB:1).
4. **Tres rutas locales** con el patrón de `server_marks.go`/`service_marks.go`:
   - `GET /api/increments/gate/stop?change=<c>`: consulta.
   - `POST /api/increments/gate`: registro.
   - `POST /api/increments/gate/publish`: publicación.

   Las dos `POST` comparten el preflight, `confirm`, el cerrojo por cambio y las negativas tipadas (prov:WEB:2, prov:WEB:3).
5. **Un módulo de pantalla nuevo**, `assets/js/screens/detalle-puerta.js`, pinta la parada en una sección propia del detalle del incremento, encima de las pestañas. Usa `confirmDialog`/`promptDialog` y clases CSS, sin estilos ni scripts en línea (prov:WEB:4 a prov:WEB:8).

Registrar y publicar siguen siendo **dos pasos no atómicos**, igual que en la CLI con `gate record` seguido de `axiom sdd publish`. La UI los encadena tras un único clic de confirmación. El estado intermedio («aprobada sin publicar») se reconstruye siempre desde el status nativo (`publication`) y desde el ledger, nunca desde la memoria del navegador.

## Decisiones de arquitectura

### D1 — Núcleo `RecordGate` con entrada tipada y validación compartida

**Elección**: `RecordGate(ctx, GateRecordInput)` recibe campos tipados y empieza por la misma validación que la CLI:

- `kickoff.ValidateGateRecordArgs(parsed)`: función exportada nueva que contiene las comprobaciones posteriores al bucle de `ParseGateRecordArgs` (`internal/kickoff/args.go`, tramo «requiere --change/--gate/--decision», `validateHandOffShape`, motivo obligatorio con `rejected`, `--commit` con `pr_merged`) más las de vocabulario por campo (`validateChangeName`, `validateGateKey`, decisión, tipo de evidencia). `ParseGateRecordArgs` la llama en su cola, así que la regla vive una sola vez.
- `validateCommitReviewShape` y `parseCommitReviewMode(value)`, extraída de `splitCommitReviewFlag` con el mismo mensaje.

Después ejecuta, en el mismo orden que hoy, todo el cuerpo de `runSDDGateRecord`:

1. `resolveGovernanceChangeRoot` y `completeGateBaseRef`.
2. Guarda A, con contexto `archive` para `integration`.
3. `kickoff.Load` y las comprobaciones de estado: `requireHandOffStops`, `requireCommitReviewStops`, la comprobación de puerta pendiente y el roster.
4. El actor (`defaultActor` si `Actor` está vacío), `gateArtifactDigest`, `prepareHandOff` y `verifyGateEvidence`.
5. `AppendGate` o `AppendGateChecked`, `emitGateIntegrationEvent` (una sola vez) y `resolveGatePublish`.
6. `lastBlockingRoleJustClosed`.

**Alternativas descartadas**:
- Construir un argv en el dashboard y pasar por `ParseGateRecordArgs`: garantiza la validación, pero rechazaría los motivos que empiezan por `-` (`requireFlagValue`) y obliga a serializar y volver a analizar.
- Validar en el dashboard: duplica reglas, lo que viola la regla dura de la propuesta.

**Motivo**: una sola implementación de cada regla. La CLI sigue analizando el argv como hoy y la revalidación tipada es idempotente: los errores de bucle saltan antes, así que los mensajes y su orden no cambian.

### D2 — Los avisos viajan como datos (`Notices`) y la CLI los vuelca igual que hoy

**Elección**: `GateRecordOutcome.Notices []GateNotice{Source, Text}` recoge, en orden, tres tipos de aviso:

- `branch_guard`: aviso de la guarda A, hoy `writerBranchWarnings`.
- `actor_identity`: identidad incompleta, hoy `actorIdentityWarnings`, solo si el registro se escribe.
- `integration_dispatch`: avisos del despacho, hoy `integrationWarnings`.

El núcleo devuelve el resultado **también cuando hay error**, con los avisos acumulados hasta ese punto.

Para conseguirlo, los ayudantes ganan variantes con un recolector:

- `guardChangeWriterCollect(changeRoot, change, op, ctx, warn func(string)) error`: `guardChangeWriter` pasa a ser su envoltorio, que escribe en `writerBranchWarnings`.
- `emitGateIntegrationEvent(loc, changeRoot, record, warn)`, con `dispatchIntegrationEventCollect`/`reportDispatch` parametrizados. `dispatchIntegrationEvent` conserva su firma para `role.start` y `archive`.

El verbo CLI llama a `flushGateNotices(outcome.Notices)`, que escribe cada aviso en su sumidero original con el mismo prefijo, antes de escribir nada en stdout y antes de devolver un error.

**Alternativas descartadas**:
- Seguir escribiendo en las variables globales desde el núcleo: el dashboard perdería los avisos y las variables no son seguras con peticiones concurrentes.
- Un sumidero inyectado por petición: más superficie y el mismo efecto.

**Motivo**: hoy todos los avisos salen antes que cualquier línea de stdout (cabecera tras `:182-183`). Volcarlos al final del núcleo conserva el orden byte a byte en stderr y en stdout, y las pruebas que sustituyen esas variables siguen capturándolos.

### D3 — Errores tipados que conservan el mensaje de la CLI

**Elección**: el núcleo envuelve cada negativa en `*GateRecordError{Code string; Err error}`. Su `Error()` devuelve exactamente `Err.Error()` y tiene `Unwrap()`. Las negativas del motor de ramas siguen viajando como `*branch.RefusalError` (con su `Code` y su continuación). Los errores sin envolver son fallos internos.

| Código | Origen |
|---|---|
| `gate_args_invalid` | `ValidateGateRecordArgs`, `parseCommitReviewMode`, `validateCommitReviewShape` |
| `change_not_found` | `resolveGovernanceChangeRoot` (no existe) |
| `change_archived` | `kickoff.RefuseArchivedRoot` |
| `gate_no_stops` | `requireHandOffStops`, `requireCommitReviewStops` |
| `gate_kickoff_missing` | `role-apply` sin kickoff sellado |
| `gate_role_not_in_roster` | `gateKeyInRoster` |
| `hand_off_unavailable` | almacén `engram`/`none` en `prepareHandOff` |
| `gate_artifact_missing` | digest vacío en `prepareHandOff` |
| `gate_already_resolved` | `gateAlreadyResolved` y la comprobación bajo cerrojo |
| `gate_digest_changed` | nuevo, solo con `ExpectedDigest` (D4) |
| `gate_evidence_refused` | `verifyGateEvidence` |

**Alternativa descartada**: clasificar por texto en el dashboard, que es frágil.

**Motivo**: la CLI imprime lo mismo (el envoltorio es transparente) y el dashboard proyecta a HTTP sin analizar cadenas.

**Trinquete de rechazos**: la clave de la línea base es `fichero + mensaje`. Los rechazos que se mueven de `sdd_gate.go` a `sdd_gate_core.go` ya nombran su salida o llevan `// refusal:by-design`, así que no aparecen en `.refusal-ratchet-baseline.txt` y siguen satisfechos. Ninguno de `sdd_gate*.go` figura hoy en ella. El rechazo nuevo `gate_digest_changed` nombra su resolución: volver a abrir el detalle o `axiom sdd status <c> --cwd <raíz>`. No se espera tocar la línea base. Si el análisis lo exigiera, se regenera en la misma unidad de trabajo.

### D4 — Comprobación de «puerta aún pendiente» y digest visto, dentro del núcleo y bajo el cerrojo del ledger

**Elección**: `GateRecordInput.ExpectedDigest`, que solo rellena el dashboard. Con valor:

- Tras calcular `gateArtifactDigest`, si difiere, el núcleo devuelve `gate_digest_changed` sin escribir.
- El registro se anexa **siempre** con `AppendGateChecked`, con una comprobación que niega (`gate_already_resolved`) en dos casos:
  - la puerta ya tiene aprobación (`ledger.Approval`);
  - el último registro de esa puerta es un rechazo del mismo digest visto, es decir, otra persona ya iteró esta misma versión.

Antes de llamar al núcleo, el servicio exige además que la puerta pedida sea la que proyecta el status en ese momento (`gate_not_pending`, D8).

La CLI nunca rellena `ExpectedDigest`, así que conserva su comportamiento exacto: comprobación bajo cerrojo solo con `--hand-off` o `--commit-review`.

**Alternativa descartada**: comprobar en el dashboard antes de llamar. Dejaría una ventana entre la comprobación y la escritura (RF-GAT-09).

**Motivo**: la única comprobación sin carrera es la que corre bajo el cerrojo de `gates.yaml`. Queda una ventana residual aceptada: un rechazo de otro canal entre la lectura del status y el cerrojo, con el mismo digest y seguido de una aprobación desde la UI. La cubre el segundo caso de la comprobación.

### D5 — Evento de integración emitido una sola vez, en el núcleo

**Elección**: `emitGateIntegrationEvent` solo se llama desde `RecordGate`, tras un `Append` correcto. Ni el verbo CLI, ni el adaptador, ni el dashboard lo vuelven a llamar.

Un fallo del despacho no deshace el registro ni se reintenta en la petición. Llega como aviso `integration_dispatch` con su orden de reintento (`axiom integrations emit|sync`), igual que en la CLI. La clave de idempotencia (huella + ordinal) no cambia.

**Motivo**: es la condición de prov:GAT:1 («una sola vez») y evita el doble efecto ADO señalado en la exploración.

### D6 — `PublishChange` comparte el cuerpo de `RunSDDPublish`

**Elección**: se extrae de `RunSDDPublish` la función `publishChangeAt(ctx, loc, change, archive, branch, warn)`, que aplica la guarda A (con la pista de `--archive`) y llama a `sddpublish.Publisher.Run`. La usan:

- `RunSDDPublish`, sin cambio de salida;
- `PublishChange(ctx, ChangePublishRequest{CWD, Change, Branch}) (ChangePublishResult, error)`, siempre con `Archive: false`, sin `Only` ni `Replay`.

`Branch` siempre va atado. No hay fetch y nunca se fuerza: eso lo garantiza `sddpublish`. Las negativas de `sddpublish` viajan en `ChangePublishResult.Refusal{Code, Label, Message, Continuation, Details}`. La negativa de la guarda viaja como `*branch.RefusalError` en `error`.

**Alternativas descartadas**:
- Reutilizar `MarksPublisher.PublishMarks`: está acotado a `pruebas-estado.yaml`.
- Llamar a `app.RunArgs`: errores no tipados.

**Motivo**: es la misma lógica que `axiom sdd publish`, sin duplicar la guarda.

### D7 — Puerto `GatesGateway` y raíz de cada operación

**Elección**:

```go
// internal/dashboard/gates_port.go
type GatesGateway interface {
	Stop(ctx context.Context, cwd, change string) (cli.GateStop, error)
	Record(ctx context.Context, in cli.GateRecordInput) (cli.GateRecordOutcome, error)
	Publish(ctx context.Context, req cli.ChangePublishRequest) (cli.ChangePublishResult, error)
}
```

La raíz de cada operación sale así:

- **`Stop`** llama a `sddstatus.Resolve(ResolveOptions{CWD: <raíz del dashboard>, ChangeName})`. Con ello reutiliza el reenraizado en worktrees registrados (RF-CLI-07) y obtiene `actionContext.workspaceRoot`, la misma raíz que usa el orquestador para `gate record` y `publish`.
- **Raíz de registro** (`GateStop.RecordRoot`):
  - para `spec`, `design`, `tasks` y `role-apply:<rol>`, es esa raíz del status;
  - para `integration`, es el worktree de archivado si el motor lo tiene registrado (`engine.RegisteredChangesByName`, `Kind == archive`, la precedencia que ya usa el dashboard) y, si no, la raíz del status. En ese caso la guarda A del núcleo, en contexto `archive`, niega con la continuación de la CLI (`axiom branch ensure --archive ...`).
- **El digest** que devuelve `Stop` se calcula con `gateArtifactDigest` sobre la carpeta del cambio de `RecordRoot`, la misma que usará el núcleo.
- **`Publish`** usa siempre la raíz del status.
- **Actor**: el dashboard nunca envía `Actor`; el núcleo aplica `defaultActor(workspaceRoot)`, es decir, la identidad git que resolvería la CLI en esa raíz sin `--actor`. El cuerpo HTTP no admite un campo `actor` (`DisallowUnknownFields`).

**Motivo**:
- La UI decide en la misma raíz y con la misma identidad que la CLI, lo que hace indistinguibles los registros (prov:GAT:1).
- En multirepo, la rama de `publication` del status es la de integración del repositorio de specs.
- `TestImportsDirection` no se ve afectado, porque la dirección `dashboard -> cli` ya existe.

### D8 — Servicio: orden de comprobaciones y cerrojo compartido

**Elección**: un cerrojo por carpeta de cambio, `gatesLocks sync.Map` de `*sync.Mutex` con `TryLock`, **compartido por registro y publicación**. Si está tomado, la respuesta es `gate_busy` (409).

Orden en `RecordGate` del servicio. Cada paso responde y para; ninguno escribe:

1. Nombre de cambio válido (`validMarksChange`): 400 `gate_body_invalid`.
2. `confirm` verdadero: 422 `gate_confirm_missing`, antes de cualquier lectura.
3. Pasarela inyectada: 503 `gates_gateway_missing`.
4. Localización del cambio:
   - inexistente: 404 `change_not_found`;
   - feature ODD: 409 `gate_not_sdd`;
   - archivado: 409 `gate_change_archived` (reutiliza la lógica de `marksMissingChange`, generalizada).
5. `TryLock`: 409 `gate_busy`.
6. `Stop`: con status ilegible, 409 `status_unreadable`. Con una puerta proyectada distinta de la pedida, o sin parada, 409 `gate_not_pending`, con `details{seenGate, projectedGate}`.
7. `Record` con `ExpectedDigest`. Las negativas se proyectan con la tabla de D11.

Orden en `PublishGate`, con los pasos 1 a 6 equivalentes y después:

- `executionStyle` distinto de `checkpointed`: 409 `gate_not_checkpointed`.
- Almacén declarado `engram`/`none`: 409 `publish_unavailable` con `details.store`.
- Último registro del ledger que no sea una aprobación de una puerta distinta de `integration`: 409 `gate_approval_missing`.
- `publication` del status:
  - ausente: 200 `nothing_to_publish`, idempotente frente al doble clic, sin git;
  - `code_unpublished`: 409 `publication_code_unpublished`, con `details.pushCommand` y `details.codeCommits` para mostrarlos. **Nunca se ejecuta `pushCommand`**;
  - `unavailable`: 409 `publication_unavailable`, con `details.unavailable`;
  - `pending`: se continúa.
- `branch` pedida distinta de `publication.branch`: 409 `publication_branch_mismatch`.
- `Publish`: una negativa de `sddpublish` responde 409 con su `code`, `label`, `message`, `continuation` y `details`; un éxito, 200.

**Alternativas descartadas**:
- Reutilizar el cerrojo de marcas: su indicador `publishing` tiene otra semántica.
- Cerrojos separados para registro y publicación: permitirían publicar a mitad de un registro.

**Motivo**: es el patrón probado de `service_marks.go:464-515` y deja cerrados los huecos 4, 5, 7 y 15 de la revisión de cobertura.

### D9 — Origen de «aprobada sin publicar» y reintento

**Elección**: la respuesta del `GET` incluye siempre, haya o no parada:

- `publication`: proyección del status, con `state`, `reasons`, `remote`, `branch`, `prTarget`, `invocation`, `pushCommand` y `unavailable`;
- `lastDecision`: el último registro del ledger, con `gate`, `decision`, `handedOff`, `handedOffTo` y `recordedAt`;
- `executionStyle`.

La UI muestra «aprobada sin publicar» y el botón de reintento solo cuando se cumplen las cuatro condiciones:

- `executionStyle == checkpointed`;
- `lastDecision.decision == approved`;
- `lastDecision.gate != integration`;
- `publication.state == pending`.

Son las mismas precondiciones que aplica el servidor (D8). El reintento solo llama a `/api/increments/gate/publish`; nunca vuelve a registrar.

**Motivo**: tras aprobar, el status deja de proyectar `await-gate`. El estado intermedio tiene que salir de datos persistentes, sobrevivir a una recarga y coincidir con lo que haría el orquestador al reanudar (`publication.state: pending`).

**Riesgo aceptado**: si tras la aprobación se escriben documentos de la fase siguiente, el reintento también los publica, igual que la `publication.invocation` que el orquestador ejecuta al reanudar.

### D10 — Presentación en la UI

**Elección**: una sección `<section id="inc-gate" class="gate-panel" hidden>` en `index.html`, entre `#inc-stepper` y `#inc-tabs`, pintada por `detalle-puerta.js` (`renderGate(section, { change, onChanged })`), que `detalle.js` invoca al cargar el detalle.

- **Sin parada** (`state: none`): la sección queda oculta salvo el bloque de «aprobada sin publicar» (D9). En `continuous` no aparece ningún control (prov:WEB:4).
- **Con parada**:
  - Contenido: titular, motivo, criterios, evidencia, digest abreviado, y enlaces a las pestañas `explicacion` y `pruebas` cuando existen.
  - Botones: uno por `choices[]`, en su orden, con `label` y `effect` tal cual.
  - Correspondencia de cada respuesta con el cuerpo de la petición: `rejected` → `{decision: rejected, reason}`; `hand_off` → `{decision: approved, handOff: true}`; `approved` → `{decision: approved}`.
  - Sin salida `hand_off` en `choices` (almacén `engram`/`none`), no hay botón de cesión ni publicación.
- **Iterar**: `promptDialog` pide el motivo. Aceptar el diálogo es el clic de confirmación (`confirm: true`); cancelarlo no envía nada. El servidor valida el motivo vacío con la regla de la CLI (422 `gate_args_invalid`). Iterar no publica ni emite evento, y la parada se vuelve a consultar.
- **`tasks` aprobada** (salidas 2 y 3): grupo de radio `notify` (marcado por defecto), `stop` y `stop-before-commit`, enviado como `commitReview`. No aparece al iterar ni en otras puertas (prov:WEB:5). Una petición manual sin `commitReview` registra sin campo, equivalente a la CLI sin bandera.
- **`integration`**: dos botones en el orden de la `GateQuestion`. Aprobar abre un formulario con `kind` (`pr_merged` por defecto, `deployment`, `attestation`), `commit` (para `pr_merged`), `baseRef` opcional y `evidence` (texto para `deployment`/`attestation`). La validación la hace el núcleo (`--commit` obligatorio con `pr_merged`, ancestro confirmado) y la negativa se muestra tipada. La CLI admite aprobar `integration` sin evidencia y el servidor no añade reglas, pero el formulario envía siempre un tipo, coherente con la doctrina «aprobar con evidencia de merge». La respuesta muestra `verified` (`pr_merged` comprobado frente a declarado). No publica.
- **Salidas 2 y 3**:
  - Un único `confirmDialog` con un mensaje fijo: se registra la decisión y después se ejecuta una operación remota (commit de los documentos del cambio y push de la rama `<branch>` a `<remote>`, sin forzar, sin fetch, sin push de código, PR ni merge).
  - Al aceptar: `POST /gate` y, si la respuesta trae `record.publish`, `POST /gate/publish` con la rama mostrada. Si trae `publishUnavailable`, se muestra el código y no se publica.
  - Si el registro falla (4xx o 5xx): se muestra la negativa, **no se publica** y se vuelve a consultar la parada.
  - Si falla la publicación: «aprobada sin publicar» con código, etiqueta, mensaje y continuación, y la parada se vuelve a consultar.
- **Tras la salida 2**: se muestra `handedOffTo`, «la sesión no continúa», las dos órdenes de reanudación (`delivery.resume`) y, si las hay, las órdenes de push de código del rol (`delivery.deliveries[].steps`) como chips de texto. **Nunca se ejecutan.**
- **Tras aprobar `role-apply`**: con `lastRoleClosed`, se muestra el aviso de último rol y `pendingNonBlockingRoles`.
- **`notices`**: se muestran como lista informativa.
- **Estilos**: solo clases nuevas en `style.css` (`gate-panel`, `gate-choices`, `gate-choice-effect`, `gate-form`, `gate-unpublished`). No hay `style=`, `<script>` en línea ni `confirm()` nativo.

**Motivo**: es una parada a la vez y su presentación es fiel a la `GateQuestion` (RF-GAT-14), con el patrón de `detalle-pruebas.js:153-170`. Cubre los huecos 1, 2, 4, 9 y 13 de la revisión de cobertura.

### D11 — Proyección HTTP de las negativas

Todas usan el cuerpo de `MarksRefusal.body()`: `{status, code, error, continuation, label?, details?}`. El preflight se extrae de `marksPreflight` como `localJSONPreflight(w, r, into, codes)`. `marksPreflight` queda como envoltorio con sus códigos actuales y las rutas de puertas usan `gate_content_type` y `gate_body_invalid`.

| HTTP | Códigos |
|---|---|
| 405 | método distinto de `POST` (también `GET` sobre `/gate` y `/gate/publish`) |
| 403 | `request_not_local` |
| 415 | `gate_content_type` |
| 400 | `gate_body_invalid` (JSON inválido, más de 16 KiB, campo desconocido, más de un objeto, nombre de cambio inválido) |
| 422 | `gate_confirm_missing`, `gate_args_invalid` |
| 404 | `change_not_found` |
| 409 | `gate_busy`, `gate_not_pending`, `gate_digest_changed`, `gate_already_resolved`, `gate_not_sdd`, `gate_change_archived`, `change_archived`, `gate_no_stops`, `gate_kickoff_missing`, `gate_role_not_in_roster`, `hand_off_unavailable`, `gate_artifact_missing`, `gate_evidence_refused`, negativas del motor de ramas (`wrong_branch`, etc., con su continuación), `status_unreadable`, `gate_not_checkpointed`, `publish_unavailable`, `gate_approval_missing`, `publication_*`, negativas de `sddpublish` |
| 503 | `gates_gateway_missing` |
| 500 | `gate_record_failed` (error sin tipar, por ejemplo de E/S del ledger) |

El `GET` de la parada responde:

- 200 con `state: await_gate`, o con `state: none` y uno de estos `reason`: `continuous`, `no_gate`, `not_sealed`, `blocked` (más `blockedReasons` y `nextRecommended`), `archived` u `odd`;
- 404 si el cambio no existe;
- 503 sin pasarela;
- 409 `status_unreadable`.

### D12 — Guardas de calidad

- **Canario de secretos** (`secrets_canary_test.go`): `/api/increments/gate/stop` entra en `coveredRoutes` como `/api/increments/gate/stop?change=x`; la prueba inyecta `cli.NewGatesGateway()` para que la ruta recorra `Resolve` sobre el fixture. `/api/increments/gate` y `/api/increments/gate/publish` entran en `excludedRoutes` con su motivo y con el nombre de la prueba con centinela que las ejercita (`TestGateRoutesNeverEchoTheSecret`).
- **Inventario de escritores** (`writer_inventory_test.go`): la entrada `internal/cli.runSDDGateRecord` se sustituye por `internal/cli.RecordGate` (la función que llama a `kickoff.AppendGate`) y `guardChangeWriterCollect` se añade a `guardCalls`. El dashboard no llama a `AppendGate`.
- **Código muerto** (`.deadcode-baseline.txt`): no se deja ningún ayudante sin uso; la CLI y la pasarela llaman a todo lo exportado.
- **CSP**: `TestNoInlineStyleOrScriptInAssets` cubre `detalle-puerta.js` e `index.html`.
- **Cabeceras**: las pruebas de rutas usan `s.handler()` con `Host: 127.0.0.1`.
- **Push**: ninguna prueba hace push real. Las de `internal/cli` usan los remotos desnudos locales de `sdd_publish_gate_test.go`; las del dashboard, una pasarela doble.

### D13 — Fuera de alcance confirmado

- Verbos del tracker tras aprobar (`summary push`, `tasks create`, `time propose|record`): los sigue lanzando el orquestador. Solo se conserva el evento de integración de C-07-e.
- Lanzar verify, judgment, revisión de cobertura o verificación conjunta.
- Mostrar los informes de revisión independiente o el resumen de marcas en la parada: quedan en sus pestañas, enlazadas.
- `fetch`, `axiom autofetch run`, push de código, PR y merge.
- Editar el actor.
- Publicar el archivado (`--archive`).

## Flujo de datos

```
Navegador (detalle-puerta.js)
   │ GET /api/increments/gate/stop?change=c
   ▼
Server.handleGateStop ──► Service.GateStop ──► GatesGateway.Stop (cli)
                                               ├─ sddstatus.Resolve(CWD=raíz dashboard, c)
                                               │    → gateQuestion, publication, governance
                                               ├─ RecordRoot (status | worktree de archivado si integration)
                                               ├─ gateArtifactDigest(RecordRoot/changes/c, gate)
                                               └─ kickoff.LoadGates → lastDecision
   ◄── {state, stop{gate,digest,choices,...}, publication, lastDecision, executionStyle}

Clic (salida 2/3) → confirmDialog(remoto, rama) → OK
   │ POST /api/increments/gate {change,gate,digest,decision,handOff?,commitReview?,reason?,evidence?,confirm}
   ▼
localJSONPreflight ─► Service.RecordGate ─► [confirm, gateway, locate, TryLock(cambio)]
                                           ├─ Stop: ¿puerta proyectada == gate?
                                           └─ GatesGateway.Record ─► cli.RecordGate (núcleo único)
                                                 validación · guarda A · paradas · roster · actor
                                                 digest==ExpectedDigest · cesión · evidencia
                                                 AppendGateChecked(aún pendiente)  ── gates.yaml
                                                 emitGateIntegrationEvent (×1)     ── cola/ADO
                                                 resolveGatePublish · último rol · Notices
   ◄── {record: <axiom.sdd-gate-record/v1>, notices[]}
   │ (si record.publish) POST /api/increments/gate/publish {change, branch, confirm}
   ▼
Service.PublishGate ─► [confirm, gateway, locate, TryLock(cambio), Stop:
                        checkpointed, almacén, lastDecision aprobada, publication.state==pending,
                        branch==publication.branch]
                     └─ GatesGateway.Publish ─► cli.PublishChange ─► publishChangeAt
                                                guarda A · sddpublish.Run(Branch atada)
   ◄── {status, remote, branch, commit, pushed, uncommittedDocs, resume[]} | 409 negativa tipada

CLI: axiom sdd gate record ... ─► runSDDGateRecord ─► parse argv ─► cli.RecordGate
                                   ─► flushGateNotices (stderr) ─► texto | JSON v1 (sin cambios)
```

## Cambios de ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/kickoff/args.go` | Modificar | Extraer `ValidateGateRecordArgs` (validación de forma y vocabulario); `ParseGateRecordArgs` la llama en su cola |
| `internal/cli/sdd_gate_core.go` | Crear | `GateRecordInput`, `GateRecordOutcome`, `GateNotice`, `GateRecordError`, `RecordGate`, comprobación por digest visto, `gateDigestChanged` |
| `internal/cli/sdd_gate.go` | Modificar | `runSDDGateRecord`: analiza argv, llama a `RecordGate`, `flushGateNotices` y formatea; sin lógica propia |
| `internal/cli/sdd_gate_commit_review.go` | Modificar | Extraer `parseCommitReviewMode` de `splitCommitReviewFlag` |
| `internal/cli/sdd_gate_handoff.go` | Modificar | `gateRecordDocument(outcome)` construye el v1; `writeGateRecordJSON` lo usa |
| `internal/cli/sdd_writer_guard.go` | Modificar | `guardChangeWriterCollect`; `guardChangeWriter` como envoltorio |
| `internal/cli/integrations_events.go` | Modificar | `emitGateIntegrationEvent(..., warn)`, `dispatchIntegrationEventCollect`, `reportDispatch` parametrizado |
| `internal/cli/sdd_publish.go` | Modificar | `publishChangeAt` compartido; `ChangePublishRequest`, `ChangePublishResult`, `ChangePublishRefusal`, `PublishChange` |
| `internal/cli/gates_gateway.go` | Crear | `GatesGateway`, `NewGatesGateway`, `GateStop` (`Stop`, `Record`, `Publish`) |
| `internal/dashboard/gates_port.go` | Crear | Interfaz `GatesGateway`, `SetGatesGateway`, `gatesGateway()` |
| `internal/dashboard/server_gates.go` | Crear | `handleGateStop`, `handleGateRecord`, `handleGatePublish` |
| `internal/dashboard/server_marks.go` | Modificar | Extraer `localJSONPreflight`; `marksPreflight` como envoltorio sin cambios de códigos |
| `internal/dashboard/service_gates.go` | Crear | DTOs, `GateStop`, `RecordGate`, `PublishGate`, `gatesLockFor`, proyección de negativas (D8, D11) |
| `internal/dashboard/service_marks.go` | Modificar | Generalizar `marksMissingChange` para que la reutilicen las puertas (mismos códigos para marcas) |
| `internal/dashboard/service.go` | Modificar | Campos `gatesMu`, `gatesGW`, `gatesLocks` |
| `internal/dashboard/server.go` | Modificar | Registrar las tres rutas antes del prefijo `/api/increments/` |
| `internal/dashboard/assets/index.html` | Modificar | `<section id="inc-gate" class="gate-panel" hidden>` |
| `internal/dashboard/assets/js/screens/detalle-puerta.js` | Crear | Parada, salidas, diálogos, formulario de `integration`, modo de revisión, «aprobada sin publicar» y reintento |
| `internal/dashboard/assets/js/screens/detalle.js` | Modificar | Invocar `renderGate` al cargar y refrescar el detalle |
| `internal/dashboard/assets/style.css` | Modificar | Clases `gate-*` |
| `cmd/axiom/main.go` | Modificar | `svc.SetGatesGateway(cli.NewGatesGateway())` |
| `internal/dashboard/secrets_canary_test.go` | Modificar | Clasificar las tres rutas (D12) |
| `internal/branch/engine/writer_inventory_test.go` | Modificar | `internal/cli.RecordGate` en `guardedWriters`; `guardChangeWriterCollect` en `guardCalls` |
| `docs/plan/backlog-2026-11/06-ui-human-loop.md` | Modificar | Entrada C-06-f: decisiones confirmadas y criterio de terminado |
| Pruebas nuevas | Crear | Ver «Estrategia de pruebas» |

## Interfaces y contratos

```go
// internal/cli/sdd_gate_core.go
type GateRecordInput struct {
	CWD          string
	Change       string
	Gate         kickoff.GateKey
	Decision     kickoff.GateDecision
	Reason       string
	Actor        string // vacío: defaultActor(workspaceRoot), como la CLI sin --actor
	HandOff      bool
	CommitReview kickoff.CommitReviewMode // vacío: sin modo (equivale a notify)
	EvidenceKind kickoff.EvidenceKind
	Commit       string
	BaseRef      string
	Evidence     string
	// ExpectedDigest solo lo rellena el dashboard (D4).
	ExpectedDigest string
}

type GateNotice struct {
	Source string // branch_guard | actor_identity | integration_dispatch
	Text   string
}

type GateRecordOutcome struct {
	Change, WorkspaceRoot string
	Record                kickoff.GateRecord // tal como se anexó
	HandedOffTo           string
	Delivery              *branch.HandOffPlan // solo con la salida 2
	CommitReview          kickoff.CommitReviewMode
	Publish               *GatePublishTarget // Remote, Branch, PRTarget, Invocation
	PublishUnavailable    string
	LastRoleClosed        *bool
	PendingNonBlocking    []kickoff.PendingRole
	Notices               []GateNotice // también con error
}

type GateRecordError struct {
	Code string
	Err  error
}

func (e *GateRecordError) Error() string { return e.Err.Error() }
func (e *GateRecordError) Unwrap() error { return e.Err }

func RecordGate(ctx context.Context, in GateRecordInput) (GateRecordOutcome, error)

// internal/cli/sdd_publish.go
type ChangePublishRequest struct{ CWD, Change, Branch string }

type ChangePublishRefusal struct {
	Code, Label, Message, Continuation string
	Details                            map[string]any
}

type ChangePublishResult struct {
	Status, Remote, Branch, GitRoot, Commit, PushedSHA string
	UncommittedDocs                                    []string
	Resume                                             []string
	Refusal                                            *ChangePublishRefusal
	Notices                                            []GateNotice
}

func PublishChange(ctx context.Context, req ChangePublishRequest) (ChangePublishResult, error)

// internal/cli/gates_gateway.go
type GateStop struct {
	Change, WorkspaceRoot, RecordRoot string
	ExecutionStyle, ArtifactStore     string
	NextRecommended                   string
	BlockedReasons                    []string
	Question                          *sddstatus.SDDGovernanceGateResult // nil: sin parada
	Digest                            string                             // de Question.Gate en RecordRoot
	Publication                       *sddstatus.PublicationProjection
	LastDecision                      *kickoff.GateRecord
}
```

Cuerpos HTTP (JSON, como máximo 16 KiB, campos desconocidos rechazados):

```json
// POST /api/increments/gate
{ "change": "c", "gate": "tasks", "digest": "sha256:…", "decision": "approved",
  "handOff": false, "commitReview": "notify", "reason": "",
  "evidence": { "kind": "pr_merged", "commit": "abc", "baseRef": "", "ref": "" },
  "confirm": true }
// 201 → { "record": { …axiom.sdd-gate-record/v1… }, "notices": [ { "source": "…", "text": "…" } ] }

// POST /api/increments/gate/publish
{ "change": "c", "branch": "inc/c", "confirm": true }
// 200 → { "status": "published|up_to_date|nothing_to_publish", "remote": "origin", "branch": "…",
//         "commit": "…", "pushed": "…", "uncommittedDocs": [], "resume": ["…","…"], "notices": [] }

// GET /api/increments/gate/stop?change=c
// 200 → { "change": "c", "state": "await_gate|none", "reason": "…", "executionStyle": "checkpointed",
//         "nextRecommended": "await-gate", "blockedReasons": [],
//         "stop": { "gate", "digest", "headline", "reason", "value", "criteria", "evidence",
//                   "choices": [ { "answer", "label", "effect", "invocation" } ],
//                   "handOffUnavailable", "publication": { "remote", "branch", "prTarget", "invocation" },
//                   "publicationUnavailable", "commitReview": { "offered", "modes", "default" } },
//         "publication": { "state", "reasons", "remote", "branch", "prTarget", "invocation", "pushCommand", "unavailable" },
//         "lastDecision": { "gate", "decision", "handedOff", "handedOffTo", "recordedAt" } }
```

El dashboard no expone el actor en ninguna respuesta.

## Cobertura de requisitos (tabla propia del diseño; no sustituye a la revisión independiente)

| Requisito | Decisiones |
|---|---|
| prov:WEB:1 | D7, D9, D11 (`GET` de la parada; una parada a la vez porque sale de `Resolve`) |
| prov:WEB:2 | D1, D3, D4, D8, D11 |
| prov:WEB:3 | D6, D7, D8 |
| prov:WEB:4 | D10, D1 (`integration` con la validación del núcleo), D8 (en `continuous` no hay parada) |
| prov:WEB:5 | D10, D1 (`parseCommitReviewMode`, `validateCommitReviewShape`) |
| prov:WEB:6 | D10 (un `confirmDialog`, sin petición antes del clic) |
| prov:WEB:7 | D9, D8, D10 |
| prov:WEB:8 | D12 |
| prov:GAT:1 | D1, D2, D3, D5, D7 |

Huecos de `coverage-review.spec.md` resueltos aquí: 1 y 13 (D10), 2 (D10), 3 (D7), 4 (D10, D8), 5 (D8 paso 6), 6 (D1: la validación y las comprobaciones del núcleo rechazan como la CLI), 7 (D8), 8 (D6, D8: negativas de `sddpublish` reenviadas; D10: fallo tras la cesión), 9 (D9), 10 (D8, D11), 11 (D7), 12 (D2, D5), 14 (D7), 15 (D8), 16 (D13).

## Estrategia de pruebas

Solo pruebas localizadas: `go test ./internal/cli -run '<patrón>'`, `go test ./internal/kickoff -run ValidateGateRecordArgs`, `go test ./internal/dashboard -run '<patrón>'` y `go test ./internal/branch/engine -run WriterInventory`. La batería completa corre en CI.

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unidad (`kickoff`) | `ValidateGateRecordArgs`: cada regla y el mensaje de la CLI | Tabla; `ParseGateRecordArgs` conserva sus pruebas |
| Unidad (`cli`) | Pruebas existentes de `gate record` (texto, JSON, cesión, revisión por commit, aviso, planning home, integración, digest) | Sin cambios de expectativas: `-run 'TestSDDGate'` |
| Unidad (`cli`) | `RecordGate`: códigos de D3; `ExpectedDigest` distinto → `gate_digest_changed` sin escribir; aprobación previa → `gate_already_resolved` bajo cerrojo; rechazo previo del mismo digest → `gate_already_resolved`; avisos recogidos en orden y también con error | `sdd_gate_core_test.go` con fixtures de `sdd_gate_test.go` |
| Unidad (`cli`) | Equivalencia (prov:GAT:1): dos copias del mismo cambio, una con `RunSDDGate` y otra con `RecordGate`, dan registros idénticos salvo `recorded_at` | Comparar `gates.yaml` |
| Unidad (`cli`) | Evento único (D5): aprobar `spec` despacha exactamente un evento por cada canal | `integrationstest` contando entregas |
| Unidad (`cli`) | Guarda: rama equivocada → `*branch.RefusalError` con la misma continuación que la CLI; `integration` en contexto `archive` | Fixtures de `sdd_writer_guard_test.go` |
| Integración (`cli`) | `PublishChange` contra un remoto desnudo local: publicado, rama distinta (`wrong_branch`), código sin subir (`code_commits_unpublished`) | Helpers de `sdd_publish_gate_test.go`; sin red |
| Integración (`cli`) | `GatesGateway.Stop`: parada `spec` con tres opciones y digest, `continuous` sin parada, `publication` y `lastDecision` tras aprobar | Fixture de cambio `checkpointed` |
| Ruta (`dashboard`) | Preflight: `GET` en `/gate` → 405, `Origin` ajeno → 403, `text/plain` → 415, campo desconocido u objeto doble → 400, más de 16 KiB → 400, sin `confirm` → 422 sin llamar a la pasarela | `s.handler()`, `Host: 127.0.0.1`, pasarela doble que cuenta llamadas |
| Servicio (`dashboard`) | `gate_busy` con el cerrojo tomado (compartido registro/publicación); `gate_not_pending`; proyección de cada código de D11; `nothing_to_publish`; `publication_code_unpublished` sin ejecutar nada; `publication_branch_mismatch`; `publish_unavailable` con `engram`; `gate_approval_missing` | Pasarela doble |
| Canario (`dashboard`) | `TestNoReadOnlyDashboardRouteReturnsAnIntegrationSecret` con la ruta `GET` nueva; `TestGateRoutesNeverEchoTheSecret` con las `POST` y un despacho que falla | Pasarela real sobre fixture y secreto centinela |
| Assets (`dashboard`) | `TestNoInlineStyleOrScriptInAssets`; `gates_assets_test.go`: el módulo existe, usa `confirmDialog`/`promptDialog`, no usa `confirm(`, solo pide la publicación tras el diálogo, sin `fetch`/`push` de código | Inspección estática como `marks_assets_test.go` |
| Inventario | `writer_inventory_test.go` en verde con la entrada nueva | `go test ./internal/branch/engine -run Inventory` |
| Trinquete | `TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` | `go test ./internal/cli -run TestEveryProductionRefusal` |
| E2E | No hay batería de navegador | La prueba manual del dashboard queda en la verificación |

## Matriz de amenazas

La matriz aplica porque el diseño añade una ruta HTTP local que dispara automatización de VCS (commit y push de los documentos del cambio).

| Frontera | Casos adversariales mínimos | Aplicabilidad | Respuesta del diseño | Pruebas RED previstas |
|---|---|---|---|---|
| Rutas de tipo documentación | `requirements.txt`, `CMakeLists.txt`, Markdown ejecutable, `README.sh` | N/A: no se añade ni cambia ninguna clasificación de rutas; `PublishChange` reutiliza la selección de rutas de documentación de `sddpublish`, ya probada | — | — |
| Selección del repositorio git | `git -C`, rutas relativas, rutas absolutas | Aplicable | El cuerpo HTTP nunca lleva rutas. `change` debe ser un único componente (`validMarksChange`). La raíz sale de `Resolve` sobre la raíz del dashboard (D7). La guarda A comprueba la rama del cambio en esa raíz | `change` con `../x`, `C:\x`, `a/b` → 400 sin llamar a la pasarela; rama equivocada en la raíz → `wrong_branch` sin escribir ni publicar |
| Estado del commit | staged, `commit -a`, índice vacío | Aplicable | `sddpublish` confirma solo las rutas de documentación del cambio. El código sucio o preparado no entra. Sin cambios, `up_to_date`/`nothing_to_publish` sin commit | Código preparado en el índice más gates.yaml sucio → el commit contiene solo documentos; nada pendiente → sin commit |
| Estado del push | rama de seguimiento, primer push, refspec explícito | Aplicable | `Branch` atada a la rama mostrada y comparada con `publication.branch` antes de llamar. Push `<rama>:<rama>` a `origin`, nunca forzado. Primer push crea la rama. Commits de código sin subir → negativa `code_commits_unpublished` | Rama pedida ≠ proyectada → 409 sin push; primer push contra remoto desnudo crea la rama; remoto que rechaza (no fast-forward) → negativa tipada y la aprobación sigue en `gates.yaml` |
| Órdenes de PR | `--head` explícito, prefijo de entorno, órdenes compuestas | N/A: la UI no abre PR ni ejecuta ninguna orden de plataforma. `delivery.deliveries[].steps` y `pushCommand` solo se muestran como texto | — | — |

Además, la petición entre orígenes la frena la guarda `Host`/`Origin` local y la exigencia de `application/json` (preflight CORS). La operación remota solo ocurre tras `confirm: true`, que la UI envía únicamente después del clic en `confirmDialog`.

## Migración y despliegue

No requiere migración. Los registros de `gates.yaml` mantienen el esquema y el contrato `gate-record/v1` no cambia. Sin pasarela inyectada (por ejemplo, en pruebas o en un binario sin `main.go` actualizado), las rutas responden 503 `gates_gateway_missing` y la UI oculta la sección. Para revertir basta con revertir los commits de rutas, servicio y assets: el núcleo extraído es un refactor sin cambio de comportamiento.

## Preguntas abiertas

- [ ] Orden de merge con C-08-c y C-01-e, que tocan primitivas cercanas: confirmarlo antes de `apply` y hacer rebase sobre `origin/main` si se adelantan. No bloquea el diseño.
- [ ] Decisión de producto devuelta al orquestador: aprobar desde la UI no ejecuta los verbos del tracker (`summary push`, `tasks create`, `time *`) que el orquestador lanza tras una puerta. Queda fuera de alcance (D13). No bloquea.

## Estimación por rol

La estimación cubre el refactor del núcleo con equivalencia probada (unas 8 h), la publicación y la pasarela (unas 5 h), las rutas y el servicio con su proyección de negativas (unas 6 h), la UI con diálogos, formulario de `integration` y reintento (unas 6 h), y las guardas, el plan y las pruebas de integración locales (unas 5 h).

- fullstack: 30 h
