# Informe de verificación: arranque-desde-wi (C-07-d)

Rol: `fullstack` (informe del cambio). Ámbito: `git diff main...HEAD` de `inc/arranque-desde-wi` (12 commits propios; 84 ficheros `.go`, +11417/-30).

## Tareas (observado, sin reescribir casillas)
T0..T9 marcadas completas en `tasks.md` (incluida 8.6 de RF-GAT-32). T9 sin commit propio (sin ficheros que ajustar); su evidencia consta en `tasks.md`.

## Cumplimiento técnico
| Comprobación | Resultado |
|---|---|
| `go build ./...` | ok |
| `go vet` (changestart, integrations, state, docpaths, workspace, components, cli, cmd/axiom) | limpio |
| `gofmt -l internal cmd` | vacío |
| `go test -count=1 ./internal/changestart/... ./internal/integrations/... ./internal/state/... ./internal/docpaths/... ./internal/workspace/... ./internal/components/agentguidance/... ./internal/components/sdd/... ./cmd/axiom` | todos ok |
| `go test ./internal/components -run 'TestGolden\|TestSkills\|TestPropose'` | ok (goldens sin `-update`) |
| `go test ./internal/assets` | ok |
| `go test ./internal/branch/engine -run TestBranchDoneIgnoresIgnoredContext` | ok |
| `go test ./internal/sddpublish/... -run Context`; `./internal/sddstatus -run 'ExternalRef\|Context'` | ok |
| `go test ./internal/cli -run 'ChangeStart\|Canary\|EveryProductionRefusal'` | ok |
| `scripts/deadcode-ratchet.sh` | no ejecutado aquí (hace `go install`, binario suelto); resultado del orquestador: sin funciones nuevas inalcanzables |
| `go.mod`/`go.sum`, `git status --porcelain` | sin cambios; árbol limpio |
| Atribución de IA en commits | ninguna |

## Requisitos (prov:ARR:1..27, RF-INT-09, RF-ADO-01/02/16)
Los 27 identificadores existen en las deltas, sin pérdidas. Cada uno tiene pruebas de la matriz de `tasks.md` presentes y en verde (comprobada la existencia de las pruebas clave): uso/orden de comprobación y `git_repository_required` (TestChangeStartCheckOrder, ...GitRepositoryRequired), resolución de referencia (TestParseItemRef), rechazos (TestStartRefusals, TestStartLinked), perfil bug (TestStartProfile*), inyección y neutralización (TestNeutralize*, TestRender*, FuzzConvert), adjuntos y PAT (TestAttachment*, TestDownload*), fases y atomicidad (TestStart*, TestAdopt*, TestAdoptStagingIntactOnFailure, TestAdoptContextNeverUntracked), cerrojos (TestLockOrderPhase2NeverTakesItemLock, TestStartConcurrent*), enlace de vuelta (TestLinkBack*, TestAdoptTagAddedOnce), `context/` ignorado (TestRepoGitignoreIgnoresContext, TestLocalStateIncludesContext, TestPublishExcludesContext), hermeticidad (adotest, TestChangeStartEndToEndLocalOnly), doctrina y goldens (TestOrchestratorWorkItemRuleParity, TestODDRoutingWorkItemRule, TestRoutingAddsNoStop), skills (TestSkillsTreatWorkItemContextAsUntrusted, TestProposeExternalRefRule, TestStatusDoesNotRequireExternalRef). RF-INT-09, RF-ADO-01, RF-ADO-02 y RF-ADO-16 cubiertos por T1/T2/T6.

## Conformidad con el diseño (D-01..D-29)
Sin divergencias de comportamiento. Desviaciones mínimas ya registradas en `tasks.md` (ubicación de ficheros y pruebas, `DeriveName` sin palabras vacías con spec/diseño realineados).

## Criterio de terminado de C-07-d
Sin WI el verbo se niega (TestChangeStartWorkItemRequired); WI válido crea `context/` con Markdown y adjuntos permitidos, nombre derivado del ID, `external_ref` condicionado al vínculo y enlace de vuelta; Bug produce perfil bug; WI cerrado, tipo sin mapear o ligado a otro cambio dan rechazo tipado; `context/` ignorado en `.gitignore`. Cumplido.

## Hallazgos
- CRITICAL: ninguno.
- WARNING: ninguno.
- SUGGESTION: (1) el Registro de commits de `tasks.md` no menciona que T9 no generó commit. (2) Las pruebas de paridad de T8 viven en `components/sdd/workitem_rule_test.go` y `agentguidance/routing_workitem_test.go`, no en el fichero que citaba la tarea (documentado). (3) FuzzConvert/FuzzDeriveName solo se han ejercitado con su corpus de semillas en esta verificación.

## Limitaciones
Trinquete de código muerto no reejecutado por mí; sin Azure DevOps real (por diseño).

**Estado:** limpio
