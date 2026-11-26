# Informe de verificación: ids-provisionales (C-03-b)

Rol: `fullstack`. Diff: `814b3ce5..HEAD` (`2578f9b8` … `d61a7ccc`). Tareas: 61/61 marcadas, 0 pendientes.

## Cumplimiento funcional

| Requisito | Evidencia |
|---|---|
| REQ-47.1 forma del provisional | `internal/reqid/provisional.go`, `provisional_test.go`, `fuzz_test.go`, `arch_test.go` |
| REQ-47.2 siguiente número libre | `internal/reqid/allocate.go`, `allocate_test.go` |
| REQ-47.3 plan único | `internal/livespec/ids.go` (`PlanIDs`), `ids_test.go`, `plan.go` |
| REQ-47.4 sustitución atómica | `idmap.go`, `isoevidence.RewriteCubre/RewriteMatrix`, `livespec/odd_ids.go` |
| REQ-47.5 duplicados entre capacidades | `ids.go` + `ids_test.go` (`CollisionError`) |
| REQ-47.6 traza provisional→definitivo | `livespec/trace.go`, `trace_test.go`, `cli/sdd_trace.go` |
| REQ-47.7 ODD `--check` | `cli/archive_integrate.go`, `archive_integrate_ids_test.go` |
| REQ-47.8 `axiom sdd archive-integrate` | `cli/sdd_archive_integrate.go` (llama a `livespec.PlanChange`), test |
| REQ-47.9 `status.requirements` | `sddstatus/requirements.go:18-19`, `requirements_test.go` |
| REQ-47.10 guías | pins en `assets`, `workspace/docs_pin_test.go`, routing |
| MODIFIED (30.2/30.3/30.4, 21.17, 30.22, 29.3/29.4, 24.58) | `apply.go`, `plan.go`, `registry.go`, `sdd_archive_compose.go`, `isoevidence/reqid.go`, pruebas asociadas |

Criterios de éxito del plan:
1. Ramas concurrentes con ADDED provisional: `TestODDDocRewriteConcurrentBranchesGetDistinctNumbers` y `TestArchiveIntegrateConcurrentBranchTakesTheNextNumberAfterTheOtherMerged`. Cumplido.
2. MODIFIED con ID inexistente: `checkKnownIDs` devuelve `IDPlanError` con `unknown_requirement_id` (`ids.go:364-396`, `ids_test.go:136`). Cumplido.
3. `status --json` con `requirements.modifies/creates`: `requirements.go`, `requirements_test.go`. Cumplido.
4. Traza legible tras archivar: `trace.go` y `sdd_trace_flow_test.go`. Cumplido.

## Pruebas ejecutadas (localizadas, sin binarios)
- `go test ./internal/reqid/... ./internal/openspecdoc/... ./internal/isoevidence/... ./internal/livespec/... ./internal/kickoff/...`: ok (resultados en caché).
- `./internal/cli -run 'ArchiveIntegrate|SDDArchiveCheck|SDDTrace'`: ok, 72,6 s.
- `./internal/sddstatus -run 'Requirements|Gate|ISO|Kickoff'`, `./internal/assets -run 'SDDArchive|SDDSpec|ISO|StatusContract'`, `./internal/workspace -run DocsPin`, `./internal/components/agentguidance -run Routing`, `./internal/livingdoc -run Pattern`, `./cmd/axiom -run Help`: ok.
- `go vet` de reqid, openspecdoc, isoevidence, livespec, kickoff, sddstatus, cli, assets, workspace y livingdoc: limpio. `gofmt -l` sobre los .go tocados: sin salida.

No ejecutado: ratchet `deadcode`, `go test ./...` y la batería completa de `internal/cli` (las pasa la CI).

## Cumplimiento técnico
Fuentes consultadas: `openspec/config.yaml` (`rules.verify`, no ejecutable en su totalidad por la norma de pruebas localizadas), diseño del cambio y convenciones del repo. `technicalContext` y skills de apply: no aplicable o sin reglas citables. Sin hallazgos con regla citable.
- Errores tipados (`IDPlanError`, `CollisionError`, `MissingCapabilityError`, `DivergentRequirementError`): conformes.
- API muerta: las funciones nuevas exportadas tienen llamada de producción desde `cli` (`PlanChange`, `PlanIntegration`, `ProvisionalText`, `ProvisionalInTitle`). `SeriesOf` se movió a `export_test.go`. Sin evidencia de código muerto, aunque no pude ejecutar `deadcode`.

## Hallazgos
- CRITICAL: ninguno.
- WARNING: apply-progress declara que el RED de los pins de skills (8.1) no se observó antes de escribir la skill; la comprobación TDD de ese punto es incompleta.
- SUGGESTION: tras archivar, corregir a mano el `## Purpose` obsoleto de `openspec/specs/requirement-identifiers/spec.md` (nota 9.3 del apply).

**Estado:** limpio
