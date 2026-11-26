# Informe de verificación: eventos-ado (C-07-e)

**Rol:** `fullstack` (informe del rol = informe del cambio). **Alcance:** `git diff 7a7f8ee9c..HEAD` (14 commits: planificación y U1 a U13). **Rama:** `inc/eventos-ado`. **Árbol:** limpio. **Fecha:** 2026-11-26. `strict_tdd: false`.

## Estado de las tareas (observado, sin reescribir)

Las 13 unidades (U1 a U13) y todas sus subtareas constan `[x]` en `tasks.md`, con su entrada en el Registro de ejecución. No hay trabajo sin terminar.

## Comandos ejecutados (resultado observado)

| Comando | Resultado |
|---|---|
| `gofmt -l cmd internal` | salida vacía |
| `go vet` de `./cmd/axiom`, `./internal/branch`, `./internal/cli`, `./internal/components/agentguidance`, `./internal/components/sdd`, `./internal/integrations/...`, `./internal/specsummary`, `./internal/trackersync`, `./internal/workspace` | sin salida, exit 0 |
| `go test -count=1` de `./cmd/axiom ./internal/branch ./internal/components/agentguidance ./internal/components/sdd ./internal/integrations/... ./internal/specsummary ./internal/trackersync ./internal/workspace ./internal/idmigration` | todos `ok` (16 paquetes) |
| `go test -count=1 ./internal/cli -run 'TestIntegrations|TestDispatchIntegration|TestEmit|TestEveryProductionRefusal|TestEveryNamedReview|TestADOCanary|TestGateRecord|TestBranchEnsure|TestEnsure|Branch.*Role'` | `ok` (106,6 s) |
| `go test -count=1 ./internal/components/ -run TestGolden` y `./internal/assets/` | `ok` |
| `bash scripts/deadcode-ratchet.sh` | «no new unreachable functions» |

No ejecutado: `./internal/branch/engine` (no está entre los paquetes tocados por el diff), ni el paquete `internal/cli` entero ni `./...` (regla del proyecto; los ejecuta CI). No se compiló ningún binario suelto; nada contra Azure DevOps real (todo con `adotest` y `integrationstest`).

## Matriz requisito a código y prueba

| Requisito | Código | Prueba |
|---|---|---|
| prov:ADO:1 Task hija con marca | `internal/integrations/azuredevops/tasks.go:278` (`EnsureChildTask`) | `tasks_test.go` del adaptador con `adotest` (marca, respuesta perdida, ambigua, WIQL caído) |
| prov:ADO:2 Validación de Task | `azuredevops/tasks.go:476` (`ValidateTask`) | `tasks_test.go` (404, tipo, otro padre) |
| prov:ADO:3 Horas con `test /rev` | `azuredevops/tasks.go:582` (`RecordHours`) | `tasks_test.go`; CLI `TestIntegrationsTime*` |
| prov:ADO:4 y ADO:5 Descripción y etiquetas | `azuredevops/summary.go:27` (`UpdateSummary`) | `summary_test.go`; `TestIntegrationsSummary*` |
| prov:ADO:6 y ADO:7 Estado, asignación, archivo | `azuredevops/lifecycle.go:103` (`TransitionTask`), `deliver.go:24` (`Deliver`) | `lifecycle_test.go`, `deliver_test.go` |
| prov:ADO:8 Hermético y sin filtrar el PAT | `adotest/`; puerto con saneado | `internal/cli/integrations_ado_canary_test.go:352` |
| RF-ADO-01, RF-ADO-02, RF-ADO-14 | `manifest/azuredevops.go`, `workspace/integrations.go`, `deliver.go` | `azuredevops_test.go`, `lifecycle_internal_test.go:17` (`TestADOLifecycleConfigDefaults`), `TestGateRecordADO*` |
| prov:INT:1 Puerto de Tasks | `internal/integrations/tasks.go:171` (`WithTaskTracker`) | `tasks_test.go`, `integrationstest/tasks_test.go` |
| prov:INT:2 y INT:3 `tasks create` y `bind` | `internal/cli/integrations_tasks.go:152` | `integrations_tasks_test.go` |
| prov:INT:4 y INT:5 `summary push` y renderizador | `cli/integrations_summary.go`, `specsummary/render.go:58`, `trackersync/specmodel.go:52` | `integrations_summary_test.go`, `render_test.go` (con fuzz de escape) |
| prov:INT:6 Estimación | `trackersync/estimate.go:72` | pruebas de `trackersync` y doctrina de `sdd-design` |
| prov:INT:7 y INT:8 `time propose` y `record` | `trackersync/clock.go:83`, `cli/integrations_time.go:517` | `clock_test.go`, `integrations_time_test.go` |
| prov:INT:9 Precondiciones | `cli/integrations_lifecycle.go:160` (`trackerTarget`) | pruebas de los verbos |
| prov:INT:10 JSON sin secretos | esquemas `axiom.integrations-*/v1` | canario U11 |
| RF-INT-09 Vínculos v2 | `integrations/links.go:26` (`LinksSchemaV2`), `pending.go`, `notes.go` | `links_test.go`, `pending_test.go`, `notes_test.go` |
| RF-INT-18 Ayuda | `cli/integrations.go`, `cmd/axiom/main.go` | `integrations_help_test.go` |
| prov:BRA:1 y RF-BRA-33 Barrera y `started_at` | `cli/branch_role.go:28` (`validateEnsureRole`), `:119` (`recordEnsureRoleStarted`), `integrations/links.go:438` | `branch_role_tracker_test.go`, `branch/role_refusal_test.go` |
| prov:ARC:1 y ARC:2 Archivado y resumen final | `internal/assets/skills/sdd-archive/SKILL.md` (líneas 211 y 336) | `components/sdd/tracker_doctrine_test.go:92` |
| prov:ORQ:1 a ORQ:5 Doctrina y paridad | `sdd-orchestrator-sections.md`, `agentguidance/routing.go`, skills `sdd-design`, `sdd-apply`, `sdd-archive`; goldens | `TestTrackerWritesRuleParity`, `TestODDRoutingNeverInvokesTrackerVerbs`, `TestTrackerDoctrineNamesNoSecretsOrTrackerRoutes`, `TestGolden` |

La matriz se apoya en la existencia del código, en las pruebas en verde y en la cobertura de escenarios declarada en el Registro de ejecución; no se releyó cada escenario línea por línea contra su prueba.

## Conformidad con el diseño (D1 a D21)

Sin desviaciones detectadas en lo comprobado: puerto opcional por aserción de tipo, esquema v2 con escritura solo al cambiar, barrera `task_not_bound` sin red, `started_at` de escritura única, reloj de jornada, diario pendiente, marca determinista, resumen con un solo `PATCH`, transiciones y asignación, orden de precondiciones de `trackerTarget`, `.tracker-notes.yaml` y `archive` sobre el tipo del propio WI. Las decisiones tomadas en la apply (`TaskFake` separado de `Fake`, `RevConflictError.AfterRetry`, `kickoff_not_sealed`) constan en el Registro de ejecución y son coherentes con el diseño.

## Cumplimiento técnico

- Formato: `gofmt -l` vacío. Análisis estático: `go vet` limpio en los paquetes tocados.
- Pruebas focalizadas de todos los paquetes tocados: verdes. Trinquetes: `deadcode-ratchet` limpio; `TestRepositoryLegacyIDResidues` (en `./internal/idmigration`) verde; `TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` verde.
- Sin `go build -o` ni scripts nuevos en el diff; sin llamadas a ADO real.

## Hallazgos

**CRITICAL:** ninguno.

**WARNING (1):**
- `TestEveryNamedReviewContinuationIsStructurallyReal` no existe como función (`internal/cli/refusal_resolution_ratchet_test.go:7,71` solo la menciona en comentarios). Las tareas 7.7, 10.5 y 13.3 la citan en un `-run`, que por tanto no ejecuta ninguna prueba (ya anotado en U7). El trinquete real que cubre los rechazos y continuaciones nuevos es `TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` (verde). Conviene corregir las referencias de `tasks.md` o crear el trinquete.

**SUGGESTION (2):**
- `internal/branch/engine` no se ejecutó (no tocado por el diff); si CI lo cubre, sin acción.
- El `summary push` de la carpeta archivada no se anota en `.tracker-notes.yaml` (decisión U8/U12); queda documentado para el informe de archivo.

## Limitaciones

Verificación con dobles (`adotest`, `integrationstest`); ningún contraste con Azure DevOps real. Suite completa de `internal/cli` y `./...` pendientes de CI.

**Estado:** limpio
