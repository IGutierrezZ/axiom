# Informe de verificación: paradas-tres-salidas

> Fase `verify` · rama `inc/paradas-tres-salidas` (11 commits de implementación sobre `origin/main`) · 2026-11-03.
> Almacén: hybrid · TDD estricto: desactivado (`strict_tdd: false`) · Verificación de solo lectura: este informe es el único fichero escrito.

## Veredicto

**pass_with_warnings** · CRITICAL: 0 · WARNING: 2 · SUGGESTION: 3.

Los requisitos del alcance (REQ-26.1 a REQ-26.11, REQ-21.2, 21.4, 21.5, 21.8 a 21.13, 21.15, REQ-25.20, 25.40, 25.45 y la retirada de REQ-21.14) tienen implementación y al menos una prueba que cubre sus escenarios. `go vet ./...` está limpio y las pruebas pasan salvo el fallo ambiental ya presente en `main` (`filemerge`).

## Estado de las tareas (observado, sin reescribir)

`tasks.md`: 24 de 24 tareas marcadas `[x]` (T-01 a T-24), 0 pendientes. 11 commits de unidad de trabajo coherentes con WU-1 a WU-11.

## Órdenes ejecutadas

Todas con `-count=1`, desde la raíz del worktree `paradas-tres-salidas`. No se compiló ni ejecutó ningún binario suelto; no se ejecutó el ratchet largo de `internal/cli` ni `scripts/deadcode-ratchet.sh`.

- `go test -count=1 ./internal/kickoff/ ./internal/consentenvelope/ ./internal/sddstatus/ ./internal/branch/... ./internal/components/... ./internal/assets/ ./internal/workspace/ ./internal/dashboard/`: salida 1, solo por el fallo ambiental. `ok`: kickoff (1,1 s), consentenvelope, sddstatus (67,6 s), branch, branch/engine (264,8 s), branch/gitcli, branch/registry, components, components/agentguidance, communitytool, engram, gga, mcp, mutationjournal, opencodedefault, opencodeplugin, permissions, persona, components/sdd (97,6 s), skills, telemetryruntime, theme, uninstall, assets (16,1 s), workspace (32,0 s), dashboard (45,2 s). `FAIL`: `internal/components/filemerge`, `TestWriteFileAtomicFollowsDirectoryJunction` y `TestWriteFileAtomicFollowsRelativeDirectoryJunction` (`createtemp ... file already exists`): el fallo ambiental conocido, ya presente en `main`; no cuenta.
- `go vet ./...`: salida 0, sin avisos.
- `go test -count=1 ./internal/cli/` acotada con `-run` a `GateRecord`, `RunSDDGate`, `RunSDDKickoff`, `WriterGuard` y `Guard.*Gate` (adicional; no es el ratchet): `ok  internal/cli 65.282s`.

## Cobertura requisito, evidencia y estado

Rutas relativas a `internal/`. Código = fichero:línea de la implementación; Prueba = fichero y función.

### Compuertas de bloque (`sdd-block-review-gates`)

| Requisito | Código | Prueba | Estado |
|---|---|---|---|
| REQ-26.1 tres salidas; `integration` con dos; continuo sin paradas ni salida 2 | `kickoff/args.go:488-498` (`validateHandOffShape`), `cli/sdd_gate_handoff.go:30` (`requireHandOffStops`), `sddstatus/governance_handoff.go:47-61` | `cli/sdd_gate_handoff_test.go` `TestGateRecordHandOffRefusals` (continuo, integration, rejected con hand-off), `TestGateRecordExitsOneAndThreeNeverWriteTheHandOffMark`; `sddstatus/governance_handoff_test.go` `TestGateQuestionIntegrationKeepsItsTwoChoices`; `kickoff/args_test.go` `TestParseGateRecordArgsHandOff` | cumple |
| REQ-26.2 pregunta de tres opciones, `Validate`, sobre de autoridad en dos, Lossless Blocking Prompt | `sddstatus/governance.go:254-300` (`newGovernanceGateQuestion`), `:130-150` (`Validate`); `sddstatus/governance_handoff.go:68-81, 103-138`; `consentenvelope/envelope.go:62` (`ValidateCompleteness(answers ...string)`) | `sddstatus/governance_handoff_test.go` `TestGateQuestionOffersThreeExitsInOrder`, `TestGateQuestionValidateRejectsMalformedThreeExitShapes`, `TestEditAuthorityConsentStillRejectsThreeChoices`; `consentenvelope/envelope_test.go` `TestValidateCompletenessDeclaredAnswers`, `TestValidateCompleteness`; pin de `assets` (contrato de bloqueo, en verde) | cumple |
| REQ-26.3 `approved` con marca, vocabulario sin cambios, binario anterior, un `rejected` posterior no reabre, A-1 | `kickoff/types.go:154` (`handed_off,omitempty`), `kickoff/machine.go:203-206`, `kickoff/handed_off.go:14` (`Approval`), `kickoff/ledger.go:91` (`AppendGateChecked` bajo cerrojo), `cli/sdd_gate_handoff.go:67-73, 98-105` | `kickoff/ledger_test.go` `TestGateRecordWithoutHandedOffSerializesLikeTheLegacySchema`, `TestLegacyBinaryReadsAGatesFileWithHandedOff`; `kickoff/machine_test.go` `TestEvaluateGatesHandedOffMarkComesFromTheFirstApproval`; `kickoff/handed_off_test.go` `TestAppendGateChecked*`; `cli/sdd_gate_handoff_test.go` `TestGateRecordConcurrentHandOffAndPlainApprovalNeverHideTheMark` y el caso «ya aprobada» | cumple |
| REQ-26.4 comprobación de confirmado, A-2, `wrong_branch` primero, solo lecturas locales | `branch/engine/handoff.go:32-87, 109-143` (DD-03), `branch/handoff.go:108-128` (`uncommitted_artifacts`), `cli/sdd_gate.go:70` (guarda A antes de `prepareHandOff`), `branch/types.go:145` | `branch/engine/handoff_test.go` `TestHandOffRefusesUncommittedArtifactsOfTheChangeFolder`, `...EntirelyUntrackedChangeFolderWithOneEntry`, `...OfARoleApplyGateRequiresTheWholeWorkingTreeOfTheRoleRepository`; `cli/sdd_gate_handoff_test.go` `TestGateRecordHandOffReportsWrongBranchBeforeUncommittedArtifacts`, `TestGateRecordHandOffOfARoleApplyRequiresTheWholeWorkingTreeCommitted`, caso «gates.yaml sin confirmar no cuenta» | cumple (criterio más estricto que la letra; desviación R-3 declarada en DD-03) |
| REQ-26.5 destino y órdenes sin git remoto; texto y `--json`; topologías | `branch/engine/handoff.go:149-189, 195-237`, `branch/handoff.go:65-102`, `cli/sdd_gate_handoff.go:148-224` | `branch/handoff_test.go` (citado, `-C` absoluto, pathspec, rama concreta); `branch/engine/handoff_test.go` `...NeverRunsRemoteOperationsOrCommit`, `...DeliversTheChangeBranchToTheIntegrationAndARoleBranchToTheTracking`, `...OfTheSpecsRepositoryOfAMultirepoIsInPlaceWithoutAPR`; `cli/sdd_gate_handoff_test.go` `TestGateRecordHandOffNeverRunsRemoteGitOrCommit`, `TestGateRecordJSONInEveryExit`, `...InTheSpecsRepositoryOfAMultirepoIsInPlace` | cumple |
| REQ-26.6 disponibilidad por almacén, estilo y puerta | `cli/sdd_gate_handoff.go:54-57`, `sddstatus/governance_handoff.go:31-37`, `sddstatus/governance.go:265-268` | `cli/sdd_gate_handoff_test.go` `TestGateRecordHandOffRefusals` (engram, none, hybrid); `sddstatus/governance_handoff_test.go` `TestGateQuestionWithoutFilesDeclaresHandOffUnavailable`, `...StorePrefersTheDeclaredStore` | cumple |
| REQ-26.7 cesión visible; no bloquea; marca del registro que cierra | `kickoff/handed_off.go:35-88` (`HandedOffTo`), `sddstatus/status_v2.go:138-142, 168`, `sddstatus/governance_handoff.go:154-181`, `sddstatus/status.go:779` | `sddstatus/governance_handoff_test.go` `TestStatusShowsTheHandOffOfSpecWithoutChangingTheRoute`, `...ProjectsNoHandOffForExits1And3`, `...HandOffMarkIsTheOneOfTheRecordThatClosedTheGate`, `TestHandedOffToNamesTheNextRoleOrVerify` | cumple (ver W-01) |
| REQ-26.8 texto único proyectado a todos los runtimes; sin «Pregunta de Transición» | `assets/generic/stop-exits.md`, `components/agentguidance/stop_exits.go:19`, `components/sdd/orchestrator.go:75-85`, `assets/skills/_shared/sdd-orchestrator-sections.md`, `assets/skills/sdd-spec/SKILL.md`, `sdd-design/SKILL.md` | `components/sdd/orchestrator_stop_exits_test.go` (12 plantillas y Pi), `agentguidance/stop_exits_test.go`, `assets/assets_test.go:2491-2497` (pin de ausencia) | cumple |
| REQ-21.8 spec, 21.9 design, 21.10 tasks, 21.11 apply por rol, 21.12 iterar | `assets/skills/_shared/sdd-orchestrator-sections.md` (tercer párrafo), `kickoff/machine.go`, `cli/sdd_gate.go:116-147` | `components/sdd/orchestrator_stop_exits_test.go` `TestEveryOrchestratorRecordsTheThreeExitsWithGateRecordOnly`; pines de `assets_test.go` (delta de contexto, `SAME gate`); `cli/sdd_gate_test.go` | cumple |

### Ramas y worktrees (`change-branch-lifecycle`)

| Requisito | Código | Prueba | Estado |
|---|---|---|---|
| REQ-26.11 fusión intermedia (criterio, sin red, ilegible como antes, no bloquea) | `kickoff/handed_off.go:98-118` (`IntermediateMerge`), `branch/engine/intermediate.go:26-60`, `branch/gitcli/ops.go` (`FileAt`: ls-tree y cat-file blob) | `kickoff/handed_off_test.go` `TestIntermediateMerge`; `branch/engine/intermediate_test.go` `TestPendingArchiveSkipsAnIntermediateMergeAndListsAFinishedChange`, `...TreatsUnreadableFilesAsBefore`, `TestIntermediateMergeUsesNoNetworkAndDoesNotBlockStatus`, `TestEnsurePicksUpAnIntermediateMergeFromTheMergedBaseWithoutNetwork`; `branch/gitcli/ops_test.go` | cumple |
| REQ-25.20 pendientes excluyen la fusión intermedia | `branch/engine/pending.go:79-98` | `branch/engine/intermediate_test.go` (primer test); `pending_test.go`; dashboard y TUI heredan la lista (suite `dashboard` en verde) | cumple |
| REQ-25.45 `ensure --archive` rechaza la intermedia sin sugerir `--fetch` | `branch/engine/archive.go:73-77`, `branch/archive_refusal.go:103-116` | `branch/engine/intermediate_test.go` `TestEnsureArchiveRefusesAnIntermediateMergeWithoutSuggestingFetch`, `TestEnsureArchiveBehavesAsBeforeWithoutAnIntermediateMerge` | cumple |

### Kickoff, pre-vuelo y cierre de rol

| Requisito | Código | Prueba | Estado |
|---|---|---|---|
| REQ-21.4 retro-sello sin política ni relevos | `kickoff/infer.go:90` (siempre `HandoffNone`) | `kickoff/handoff_policy_deprecation_test.go` `TestInferredKickoffWritesNoneWithoutInferringPolicy`; `cli/sdd_kickoff_test.go` `TestRunSDDKickoffSealInferWritesNoneAsCompatibilityValue` | cumple |
| REQ-21.5 sellado sin política de relevos | `kickoff/args.go:201-211, 297`, `cli/sdd_kickoff.go:36` | `kickoff/handoff_policy_deprecation_test.go` `TestSealWithoutHandoffPolicyWritesNone`, `TestSealRequiresNoHandoffPolicy`; `cli/sdd_kickoff_test.go` `...SealWithoutHandoffPolicyWritesNoneAndNeverMentionsIt` | cumple |
| REQ-26.9 `handoff_policy` deprecado: aceptar, ignorar, `none`; texto y JSON; assets | `kickoff/types.go:26-33, 112`, `kickoff/schema.go:89-93`, `cli/sdd_kickoff.go:128-133, 160`, `sddstatus/status_v2.go:127`, `cli/sdd_gate.go:91` | `kickoff/handoff_policy_deprecation_test.go` (sin flag, flag ignorado, valor desconocido rechazado, `Validate`, legado `per_checkpoint`); `cli/sdd_kickoff_test.go` `...IgnoresDeprecatedFlagAndRejectsUnknownValue`, `...HelpAndGateErrorDoNotNameHandoffPolicy`, `...JSONEmitsHandoffPolicyOnlyWhenTheFileCarriesIt`; `sddstatus/role_gates_dependency_test.go` `TestVerifyDependencyIgnoresHandoffPolicy`; pin de assets sin el flag como obligatorio | cumple |
| REQ-21.13 aviso de último rol sin relevo (salidas 2 y 3) | `cli/sdd_gate.go:160-180`, `:273` (`lastRoleJustClosed`) | `cli/sdd_gate_test.go` `TestRunSDDGateRecordEmitsLastRoleNoticeExactlyOnceForSingleFullstackRole`, `...LastRoleOfSeveralEmitsNoticeWithoutHandoffFile`; `cli/sdd_writer_guard_test.go` `TestGuardGateRecordOfTheLastRoleGuardsTheLedgerAndWritesNoHandoff`; `cli/sdd_gate_handoff_test.go` `TestGateRecordHandOffOfARoleApplyNamesWhatIsHandedOver` | cumple |
| REQ-21.14 (retirado) sin `handoff.md` de integración | `closure.go` y `writeIntegrationHandoff` eliminados; `kickoff/rdd_boundary_test.go` sin la arista a `internal/handoff` | `cli/sdd_gate_test.go` (sin `handoff.md`); `branch/engine/writer_inventory_test.go` sin la entrada | cumple |
| REQ-21.15 `verify` por `role-apply` aprobados, sin `handoff.md` | `sddstatus/governance_handoff.go:198-220`, `sddstatus/status.go:730` | `sddstatus/role_gates_dependency_test.go` (bloqueado, listo sin fichero, `handoff.md` ignorado, rechazado, rol único y continuo, política ignorada) | cumple |

### Reglas ODD (`organic-agent-trigger-rules` y `sdd-lane-kickoff`)

| Requisito | Código | Prueba | Estado |
|---|---|---|---|
| REQ-26.10 parada única de ODD con tres salidas | `components/agentguidance/routing.go:117` (viñeta `Plan stop`), `:143` (`### Stops and exits`), paso 5 y línea de reanudación | `agentguidance/routing_stops_test.go` (proyección desde la fuente única, excepción citada, viñeta, reanudación sin espejo, sin órdenes remotas, determinismo entre adaptadores) | cumple |
| REQ-21.2 la parada no es pregunta de gobernanza | `components/agentguidance/routing.go:54` (conserva la regla del paso 0 y añade la excepción) | `agentguidance/routing_stops_test.go` `TestRenderRoutingCitesThePlanStopAsAnException`; pines de los pasos 0 y 6 en `routing_test.go` | cumple |
| REQ-25.40 el resto del protocolo no cambia | `components/agentguidance/routing.go` (solo los puntos citados) | `agentguidance/routing_test.go` (pines previos, en verde) | cumple |

Los escenarios de ODD que solo la doctrina garantiza (la salida 2 confirma solo el documento; multirepo sin PR; trabajo pequeño sin parada) se comprueban por pines de texto, no por ejecución: son instrucciones al orquestador, no código.

## Conformidad con el design (DD-01 a DD-14)

| DD | Estado | Evidencia |
|---|---|---|
| DD-01 marca `handed_off`, flag `--hand-off`, token `hand_off`, validación de forma | cumple | `kickoff/types.go:154`, `kickoff/args.go:364, 488-498`, `sddstatus/governance.go:46` |
| DD-02 orden de comprobaciones y bloqueo bajo cerrojo | cumple | `cli/sdd_gate.go:53-186` (parse, raíz, guarda A, kickoff, roster, almacén/digest/ya aprobada/HandOff, AppendGateChecked) |
| DD-03 «confirmado» (spec/design/tasks y árbol entero en `role-apply`, exenciones) | cumple | `branch/engine/handoff.go:109-143` |
| DD-04 `ValidateCompleteness(answers ...string)`; sobre de autoridad en dos | cumple | `consentenvelope/envelope.go:62`; llamadores de autoridad y RDD intactos |
| DD-05 pregunta de tres o dos opciones con `handOffUnavailable` | cumple | `sddstatus/governance_handoff.go:47-138` |
| DD-06 cesión en status (JSON v2 y nota) | cumple | `sddstatus/status_v2.go:138-168`, `governance_handoff.go:154-181` (ver W-01) |
| DD-07 `engine.HandOff` solo lectura, sin cerrojo ni Fetch | cumple | `branch/engine/handoff.go:32-87` |
| DD-08 push exacto y `head`/`base` sin plataforma | cumple | `branch/handoff.go:83-89`; sin `gh`, `glab` ni `&&` (`branch/handoff_test.go`) |
| DD-09 fusión intermedia sobre la referencia local | cumple | `branch/engine/intermediate.go`, `pending.go`, `archive.go:73-77` |
| DD-10 `verify` por `LastRoleClosed`; retirada de `closure.go` | cumple | `sddstatus/governance_handoff.go:198`; `closure.go` ausente |
| **DD-11 enmendado** | **cumple** | `ToKickoff` fija `HandoffNone` (`kickoff/args.go:297`) y `seal --infer` también (`kickoff/infer.go:90`): ambos escriben siempre `handoff_policy: none`. El flag se parsea y valida (valor desconocido rechazado, `args.go:206-209`) y se ignora (`SealArgs.HandoffPolicy` sin uso). `Validate` acepta vacío, `none` y `per_checkpoint` (`schema.go:91`) |
| DD-12 texto único y proyección | cumple | `assets/generic/stop-exits.md`; marcador `{{AXIOM_STOP_EXITS}}` resuelto en un segundo paso (`orchestrator.go:75-85`); `.codex/AGENTS.md` con «### Stops and exits» |
| DD-13 parada ODD en la doctrina | cumple | `routing.go:54, 61, 64, 117, 143` |
| DD-14 ediciones mínimas en ficheros compartidos | cumple | código nuevo en ficheros nuevos; goldens regenerados una sola vez (WU-11); `git status` limpio |

## Referencias vivas retiradas

Búsqueda con `rg --hidden` en el árbol de trabajo, excluyendo `.git`:

- `IntegrationHandoff`, `writeIntegrationHandoff` y `verifyDependencyFromHandoff`: sin ninguna aparición en código, tests, assets ni docs. Solo se citan en los artefactos de este cambio (proposal, explore, design, tasks) y en `openspec/changes/archive/**` (historial).
- `closure.go`: solo en `docs/archive/ROADMAP.md:313` (hoja de ruta archivada) y en los artefactos del cambio y archivados. Ver S-02.
- El flag `--handoff-policy`: en código solo `kickoff/args.go` (parseo y validación del flag deprecado); en documentación solo `docs/manual-de-inicio.md:500`, que lo describe como deprecado e ignorado. Ningún asset (`internal/assets`, `internal/components`, `.codex/`) lo prescribe como argumento de `kickoff seal`; el pin de `assets_test.go` lo comprueba. Las fixtures `bench/journeys_*.go` siguen con `handoff_policy: none` a propósito (compatibilidad, design §10).
- La «Pregunta de Transición» que ofrecía generar `handoff.md` o continuar: ninguna en assets; solo el literal del pin de ausencia en `assets_test.go:2491-2497`.

## Hallazgos

### CRITICAL

Ninguno.

### WARNING

- **W-01. La cesión de un `role-apply` anterior se proyecta contra el estado actual del roster, no contra el del momento de la aprobación.** `kickoff/handed_off.go:35-88` (`HandedOffTo`) fuerza el estado de la puerta a `approved` y evalúa `LastRoleClosed` sobre los estados actuales; `sddstatus/governance_handoff.go:154-159` lo usa para cada puerta con marca. Si `core` se aprobó con la salida 2 mientras `web` estaba pendiente (cedió `apply:web`) y después `web` se aprueba con la salida 3, `axiom sdd status` pasa a decir que `core` cedió `verify`, que no es lo que se cedió. El efecto es solo informativo (la nota y `handedOffTo` no bloquean ni cambian la ruta, REQ-26.7). Falta una prueba de ese caso: `TestHandedOffToNamesTheNextRoleOrVerify` cubre la marca en el último rol y la marca en el primero con `web` aún pendiente, pero no la marca en el primero con el roster ya cerrado sin marca en el último.
- **W-02. Verificación parcial por restricción del entorno.** No se ejecutaron el ratchet largo de `internal/cli` (apply lo dio en verde, 675 s) ni `scripts/deadcode-ratchet.sh` (compila un binario, prohibido por el EDR). Lo cubren `go vet ./...` limpio y la ejecución acotada de `internal/cli` (65 s, en verde) con las pruebas de `gate record` y `kickoff`; el ratchet de rechazos y el de deadcode quedan sin reobservar en esta fase.

### SUGGESTION

- **S-01.** `sddstatus/status.go:754` pasa `ArtifactStoreOpenSpec` como almacén de reserva a `gateQuestionStore`, y el design (DD-05) dice `status.ArtifactStore`. Es equivalente hoy, porque esa rama de `status` solo se alcanza con openspec o hybrid, que ofrecen la salida 2 (la ruta de Engram pasa gobierno `nil`); conviene alinear el texto del design o usar el valor resuelto.
- **S-02.** `docs/archive/ROADMAP.md:313` nombra `internal/kickoff/closure.go`, ya retirado. Es historial archivado; basta con dejarlo.
- **S-03.** Quedan pendientes para el archivado las notas no normativas de las deltas: glosario «Cambio pendiente de archivar» y fila de REQ-25.20, supuesto S-16, sección «Propósito» de `sdd-preflight-configuration` y nombre de `sdd-role-closure-handoff`. Además, el dashboard hereda la exclusión de intermedias de `PendingArchive` sin prueba propia (cubierto por la prueba del motor y por la suite de `dashboard` en verde).

## Limitaciones

- Los escenarios de la doctrina ODD y de las plantillas SDD se verifican por pines de texto, no por el comportamiento de un agente.
- No se ejecutó nada fuera del repositorio ni operaciones de red; no se tocaron `gates.yaml` ni `kickoff.yaml` de este cambio.

## Corrección posterior a la verificación

- **W-01 corregido**: la fase cedida se persiste en `handed_off_to` al registrar la salida 2 y `status` la lee de ahí (cálculo solo como reserva para ledgers anteriores); pruebas en `kickoff`, `sddstatus` y `cli`.
- **S-01 no aplicado**: en `status.go` el almacén resuelto aún no existe en ese punto (`status` se construye después con `baseStatus(ArtifactStoreOpenSpec, ...)`), y ese mismo valor es el que se pasa; se deja como está.

## Siguiente paso recomendado

`sdd-archive`: la implementación está completa y no hay CRITICAL. W-01 puede abrirse como seguimiento o corregirse antes del archivado si se quiere que la nota de cesión sea exacta tras cerrar el roster.
