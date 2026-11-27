# Informe de verificación: puertas-desde-ui (C-06-f)

Rol `fullstack`. Ámbito: `git diff ed36cb30c..HEAD` (rama `inc/puertas-desde-ui`). Solo pruebas localizadas.

## Tareas observadas

Las 8 unidades (1.1 a 8.2) figuran marcadas; apply-progress (Engram #2794) las confirma con sus commits. Hay un fixup sin plegar (98b3de7d3 sobre 9fd64b3bf), esperado: se pliega con autosquash al pedir la PR.

## Comandos ejecutados

| Comando | Resultado |
|---|---|
| `go vet ./internal/cli ./internal/dashboard ./internal/kickoff ./cmd/axiom` | exit 0 |
| `go build ./...` (sin `-o`) | exit 0 |
| `go test ./internal/kickoff -run 'GateRecordArgs'` | ok (en caché) |
| `go test ./internal/cli -run 'TestRecordGate\|TestSDDGate\|TestRunSDDGate\|TestGateRecord\|TestPublishChange\|TestSDDPublish\|TestGatesGateway\|TestParseCommitReviewMode\|TestEveryProductionRefusal'` | ok (180,6 s) |
| `go test ./internal/dashboard -run 'Gate\|Marks\|Canary\|Asset\|NoInline\|Workflow\|Origin\|Headers'` | ok (23,8 s) |
| `go test ./internal/branch/engine -run TestWriterInventory` | ok (en caché) |
| `go test ./internal/integrations -run TestImportsDirection` | ok (en caché) |

No ejecutado: trinquete de código muerto (`.deadcode-baseline.txt`, exige instalar binario; lo cubre la CI), suite completa (política local), prueba manual en navegador (no hay batería).

## Cumplimiento por requisito (evidencia por pruebas y código)

- prov:WEB:1 (consulta de parada): `internal/dashboard/service_gates.go` (`GateStopDTO`), `server_gates.go`, `internal/cli/gates_gateway.go`; pruebas `TestGatesGatewayStop`, `server_gates_test.go`. Cumple.
- prov:WEB:2 (POST registro, preflight, 409, digest obligatorio): `service_gates.go:424` (digest exigido), `server_gates.go`, núcleo `sdd_gate_core.go` con `ExpectedDigest`. Cumple.
- prov:WEB:3 (publicación atada a rama, sin fetch ni forzar): `sdd_publish.go` (`PublishChange`), `TestPublishChange` con remotos desnudos locales. Cumple.
- prov:WEB:4 (salidas, integration sin publicar): `detalle-puerta.js`, `gates_assets_test.go`. Cumple.
- prov:WEB:5 (modo de revisión): `parseCommitReviewMode` en `sdd_gate_commit_review.go`, prueba `sdd_gate_commit_review_mode_test.go`. Cumple.
- prov:WEB:6 (un solo diálogo, sin fetch ni push de código): `confirmDialog` en `detalle-puerta.js`; la prueba estática lo verifica. Cumple.
- prov:WEB:7 (aprobada sin publicar, reintento): `detalle-puerta.js`, fixup 9fd64b3bf reconstruye desde `publication.state`. Cumple (validado solo por inspección estática, sin prueba de navegador).
- prov:WEB:8 (canario, CSP, inventario): `secrets_canary_test.go` (ruta cubierta y dos excluidas con motivo, centinela), `TestNoInlineStyleOrScriptInAssets`, `TestWriterInventory`. Cumple.
- prov:GAT:1 (misma lógica que la CLI): `sdd_gate_core.go` compartido, `TestRecordGateMatchesCLI`, pruebas `TestSDDGate` sin cambiar expectativas. Cumple.

## Notas de cobertura de tasks.md (1 a 6)

Las seis están aplicadas: 1 (evidencia malformada rechazada por el núcleo), 2 (digest obligatorio, 422), 3 (mapeo de `evidence`), 4 (re-decisión tras rechazo permitida), 5 (`commitReview` explícito), 6 (reintento solo con `checkpointed` y `pending`). Verificadas por las pruebas de las unidades 3, 6 y 7.

## Conformidad con el diseño

Núcleo extraído sin cambiar la CLI, evento de integración emitido solo desde `RecordGate`, `sdd_publish.go` único importador de `sddpublish`, cerrojo compartido registro/publicación, rutas registradas antes del prefijo `/api/increments/`. Sin desviaciones observadas.

## Cumplimiento técnico

Fuentes consultadas: skill `go-testing` (declarada), `openspec/config.yaml` (reglas de verify), `technicalContext`: no aplicable. Sin hallazgos de estilo con regla citable.

## Hallazgos

- CRITICAL: ninguno.
- WARNING: ninguno.
- SUGGESTION: (1) el trinquete de código muerto y la prueba manual del dashboard quedan sin ejecutar localmente; confirmar en CI. (2) Plegar el fixup 98b3de7d3 con autosquash antes de la PR.

**Estado:** limpio
