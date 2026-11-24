# Informe de verificación: puerto-reqid (C-03-a)

Fecha: 2026-11-23. Rol: `fullstack`. Rama `inc/puerto-reqid`, diff `d580368d..HEAD` (11 commits). `strict_tdd: false` (no se evalúa evidencia RED/GREEN histórica). Verificación de solo lectura; no se editó código ni tareas.

## Alcance y progreso observado

- Artefactos leídos: `tasks.md`, `design.md`, las seis specs delta, `kickoff.yaml`.
- Tareas: 1.1 a 9.3, 10.1 y 10.3 marcadas completas; **10.2 abierta** (`internal/cli` completo sin terminar en esta máquina). No se reescribe ningún estado.
- Worktree sin cambios sin confirmar antes de este informe.

## Comandos ejecutados

| Comando | Resultado |
|---|---|
| `go vet ./...` | sin salida, código 0 |
| `gofmt -l .` | vacío |
| `go test -count=1` de reqid, workspace, openspecdoc, isoevidence, sddstatus, livespec, livingdoc, adoption/..., assets/..., components | todos `ok` (11 paquetes) |
| `go test ./internal/cli -count=1 -timeout 30m` con `-run` acotado a ReqidRF, ArchiveCheck, SDDTrace, Kickoff, ArchiveIntegrate, ArchiveCompose y ArchiveAdopt | `ok` 62.7 s |
| `go test ./internal/reqid -run NoRequirementIDFormat -v` | PASS |
| rg del criterio de terminado (`REQ-` seguido de no cifra, sin `_test.go`) | solo `internal/reqid` y comentarios (`//`) |
| `go test ./internal/cli` completo y `go test ./...` | **No observados** (el cli completo no termina en esta máquina; no se lanzó). Los fallos ajenos conocidos en Windows no se reprodujeron porque no se ejecutó `./...`. |

Comandos obligatorios del proyecto (`openspec/config.yaml`): `go test ./...` no observado completo; `go vet ./...` pasa.

## Matriz de cumplimiento

| Requisito | Evidencia (código) | Prueba que lo cubre | Estado |
|---|---|---|---|
| REQ-44.1 clave `sdd.id_pattern`, estricta, en vivo, falla cerrado | internal/workspace/sdd.go:37 `decodeSDD`, :83 `sddPattern`; declarations.go:57-62 `IDPattern` (envuelve `SDDErr` con `%w`) | workspace/sdd_test.go; cli Kickoff; example_yaml_test.go | Cumple |
| REQ-44.2 defecto `REQ-<n>.<m>` y equivalencia con regex heredadas | internal/reqid/reqid.go:21 `defaultTemplate`, valor cero = `Default()` | reqid/legacy_equivalence_test.go, reqid_test.go | Cumple |
| REQ-44.3 gramática, serie, número, formato, desbordamiento | reqid/template.go, reqid.go:37 `Parse`, :138 `Series`, :156 `Number`, :168 `Format`; series.go | reqid/template_test.go, reqid_test.go, fuzz_test.go | Cumple |
| REQ-44.4 puerto único, sin regex propia (excepción `adoptcheck`) | rg del criterio §8: solo reqid y comentarios; `adoptcheck` usa `reqid.Default()` | reqid/arch_test.go `TestNoRequirementIDFormatOutsideReqid` (PASS); imports_test.go de reqid, openspecdoc, isoevidence | Cumple |
| REQ-44.5 candidato, mal formado, repetidos | reqid.go:108 `Scan`, :95 `InTitle`; isoevidence/reqid.go:15 | reqid_test.go; isoevidence/reqid_pattern_test.go | Cumple |
| REQ-44.6 sintético fuera del espacio válido, aviso con `SDDErr` | reqid/synthetic.go:13; livingdoc/indexer.go:36 `WithPattern`, service.go | reqid/synthetic_test.go, fuzz_test.go; livingdoc/indexer_pattern_test.go | Cumple |
| REQ-44.7 extremo a extremo con `RF-<DOM>-<NN>` | cli (integrate, compose, archive-check, status ISO, bug, trace) | cli/reqid_rf_e2e_test.go (ok) | Cumple |
| REQ-44.8 paridad del «Sin impacto en la spec» | isoevidence/bug.go:15 `NoImpactReasonLabel`; livespec/impact.go | livespec/impact_parity_test.go | Cumple |
| REQ-44.9 patrón inaccesible falla cerrado con continuación de `validate` | workspace/declarations.go:57; sddstatus/iso.go (D-07); cli archive_integrate, sdd_archive_compose, sdd_archive_check, sdd_trace | cli ArchiveCheck, SDDTrace, ArchiveIntegrate, ArchiveCompose; sddstatus/iso_pattern_test.go | Cumple |
| REQ-29.1 bloque `sdd:` con lista cerrada | workspace/sdd.go (`sddKeys`) | workspace/sdd_test.go | Cumple |
| REQ-29.3 / 29.14 / 29.4 identificador en la delta, existencia, cobertura | isoevidence/reqid.go `CheckRequirementIDs`; cubre.go; bug.go | isoevidence/reqid_pattern_test.go (ByID, baseline ambigua, bug con patrón) | Cumple (ver W-1) |
| REQ-29.9 `sdd trace` con el patrón | cli/sdd_trace.go | cli SDDTrace, reqid_rf_e2e_test.go | Cumple |
| REQ-29.16 estado ISO con patrón en vivo y D-07 | sddstatus/iso.go (`patternErr`, `findingAt`) | sddstatus/iso_pattern_test.go | Cumple |
| REQ-30.4 registro y siguiente libre con el patrón | livespec/registry.go:144 `NextFree(reqid.Series)` | livespec/registry_pattern_test.go | Cumple |
| REQ-30.22 resolución por ID con vuelta al título exacto | reqid/resolve.go:38; openspecdoc/resolve.go:27, compose.go; livespec/plan.go:198 `classify`; cli/sdd_archive_check.go:211 | reqid/resolve_test.go, openspecdoc/resolve_test.go, compose_byid_test.go, compose_corpus_test.go | Cumple con divergencia (W-2) |

## Conformidad con el design

- D-01 a D-04, D-06, D-08, D-09: conformes. D-02: `reqid` es hoja; `openspecdoc` e `isoevidence` solo importan `internal/reqid` (imports_test.go).
- D-05: el núcleo vive en `reqid.Pattern.Resolve` y `openspecdoc.Resolve` delega (resolve.go:27); `classify` usa `Resolve` para MODIFIED y REMOVED (livespec/plan.go:244-257); `alreadyComposed` usa `Resolve` solo para REMOVED. El ajuste del 2026-11-23 está aplicado en el núcleo, no en `isoevidence` (W-1).
- D-07: conforme (sddstatus/iso.go: etapa `specs` con `iso_declaration_invalid`, `tasks` sin evaluar, sin `iso.matrix`, sin escenarios afectados en `archive`).
- Criterio §8 cumplido. `routing.go` y ningún golden/`testdata` están en el diff; no hay trabajo de C-03-b..e (solo comentarios que los citan). Fuera de `internal/` y `openspec/changes/` solo cambian `axiom.example.yaml` y `docs/manual-de-inicio.md` (previstos en 2.4 y 8.2).

## Juicio de las desviaciones del apply

- (a) `isoevidence` no llama a `Resolve`: ver W-1.
- (b) `Resolve` cae al título exacto con 0 portadores no superados: ver W-2.
- (c) RENAMED por título exacto en `livespec.classify` (plan.go:253-262): conforme; D-05 solo pasa MODIFIED y REMOVED a `Resolve`; se conserva la idempotencia.
- (d) Sin matriz ni `CheckScenarios` en `archive` con patrón inaccesible: conforme con D-07.
- (e) Pruebas cambiadas: `TestPlanIntegrationFailsClosedWhenAModifiedNameNoLongerMatches` pasa de REQ-31.1 a REQ-31.9 porque con REQ-30.22 un MODIFIED con el mismo ID ahora se resuelve (cambio intencionado; el escenario «un MODIFIED que cambia el identificador no se resuelve» sigue cubierto); `registry_test.go` adapta la firma `NextFree(reqid.Series)`; `TestISOArchiveRefusal` no se re-ejecutó aislado, pero entra en el subconjunto de cli.
- (f) Tarea 10.2 abierta: confirmada como no observada.

## Hallazgos

### CRITICAL
Ninguno.

### WARNING
- W-1 · design D-05 (ajuste 2026-11-23) y tareas 4.2/4.3 · internal/isoevidence/reqid.go:48-125 · `CheckRequirementIDs`, `NewBaselineIndex` y `tolerated` no usan `p.Resolve`: un MODIFIED/REMOVED solo exige un identificador válido en su nombre, que es lo que dice REQ-29.3 (fila MODIFIED o REMOVED). Ningún escenario falla y `TestRequirementIDsByIDWithChangedTitle` cubre el título cambiado con mismo ID, pero la resolución del «existente» que piden design y tareas no está implementada.
- W-2 · REQ-30.22 (sdd-archive-lifecycle, «Resolución por identificador») · internal/reqid/resolve.go:64-73 · el texto exige rechazar, sin portadores no superados, «sin recurrir al título exacto»; el código recurre al título exacto (conserva la composición de hoy cuando la delta cita un superado tal cual). Los escenarios escritos pasan, pero la regla literal difiere: alinear spec o código.
- W-3 · tarea 10.2 / `rules.verify` · `go test ./...` y `go test ./internal/cli` completos no observados; cobertura solo por paquetes y un subconjunto de cli.

### SUGGESTION
- S-1 · REQ-44.2 frente a REQ-30.22 · internal/livespec/plan_test.go: un MODIFIED con título cambiado y mismo ID, antes rechazado, ahora se acepta (intencionado por REQ-30.22); dejar constancia como excepción al «ninguna entrada cambia de resultado».
- S-2 · design D-05, coverage-review.design.md fila 13 · `InTitle` devuelve el último ID válido de un título con dos; sigue sin constar como intencionado en el invariante de no regresión.

## Resumen

El diff cumple REQ-44.1 a 44.9 y los MODIFIED vinculados con pruebas en verde (11 paquetes, subconjunto de cli, guarda de arquitectura); `go vet` y `gofmt` limpios; criterio §8 cumplido; sin cambios en `routing.go`, goldens ni trabajo de C-03-b..e. Pendiente: resolución en `isoevidence` (W-1), divergencia con el texto de REQ-30.22 (W-2) y suite completa no observada (W-3). Siguiente: `sdd-archive` tras decidir W-1/W-2, o `sdd-apply` para cerrar la 10.2.

**Estado:** no limpio — `go test ./...` (comando obligatorio de `rules.verify`) no se ejecutó completo (cli completo no termina en esta máquina); ningún hallazgo CRITICAL.
