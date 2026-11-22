# Tareas: Aviso por commit, parada opcional por tarea y validación implícita al pedir la PR (C-02-d)

## Previsión de entrega

| Campo | Valor |
|-------|-------|
| Tamaño estimado | L |
| Líneas modificadas estimadas | 1.400-1.900 (adiciones + eliminaciones, sin goldens generados) |

### Unidades de trabajo

| Unidad | Objetivo | Commit | Comando de prueba acotado | Arnés de ejecución | Frontera de reversión |
|--------|----------|--------|---------------------------|--------------------|-----------------------|
| 1 | Esquema `commit_review` + bandera `--commit-review` en `gate record` (forma, estado, cerrojo, ayuda, JSON) | `feat(sdd): registrar el modo de revisión por commit en la puerta tasks` | `go test ./internal/kickoff/ ./internal/cli/ -run "CommitReview\|GateRecord\|GateRecordHandOff\|Gate"` | Prueba de integración sobre `RunSDDGate` en directorio temporal (sin compilar binario) | Ficheros de `internal/kickoff` y `internal/cli` de la unidad; el campo es `omitempty` |
| 2 | Proyección en `status` (`gateV2.commitReview`) y contrato de estado | `feat(sdd): proyectar commitReview en el estado de la puerta tasks` | `go test ./internal/sddstatus/ ./internal/kickoff/ -run "Gate\|CommitReview\|Governance"` | Fixtures de ledger sobre `status_v2_test.go` | `status_v2.go`, `machine.go` y el contrato |
| 3 | Doctrina SDD: sección `Commit Review` de fuente única en los 12 orquestadores | `docs(sdd): sección Commit Review de fuente única en los orquestadores` | `go test ./internal/components/sdd/ ./internal/assets/ -run "SharedOrchestrator\|NoDrift\|CommitReview\|Orchestrator"` | Pruebas de ausencia de deriva y pines de doctrina | Fuente única, marcadores y pines |
| 4 | `sdd-apply` (una tarea por lanzamiento, `pending_commit`, `fixup_of`) y `work-unit-commits` en paridad | `docs(sdd): contrato commit_review de sdd-apply y fixups en work-unit-commits` | `go test ./internal/assets/ -run "CommitReview\|SkillCopiesParity\|SddApply\|WorkUnitCommits"` | Pines de skill y prueba de paridad | Dos skills y sus pines |
| 5 | ODD: aviso por commit en el paso 6 y viñeta de commits y PR en `routing.go` | `feat(odd): avisar por commit y reglas comunes de commits en la doctrina ODD` | `go test ./internal/components/agentguidance/ -run "Routing"` | Pines de `routing_test.go`; `routing_stops_test.go` y REQ-27.17 sin tocar | `routing.go` y `routing_test.go` |
| 6 | Regeneración de goldens y barrido de tests afectados | `test(golden): regenerar goldens por la doctrina de revisión por commit` | `go test ./internal/components/ -run TestGolden` (sin `-update`) | `go test ./internal/components/ -run TestGolden -update` y repetir sin `-update` | `testdata/golden/*.golden` |

## Fase 1: Esquema y CLI (unidad 1)

- [x] 1.1 En `internal/kickoff/types.go`: tipo `CommitReviewMode`, constantes `notify`/`stop`/`stop-before-commit`, `CommitReviewModes()`, `EffectiveCommitReview()`; campo `CommitReview` (`yaml:"commit_review,omitempty"`) en `GateRecord` y en `GateState`.
  Cubre: REQ-33.1, REQ-33.2, REQ-33.3
- [x] 1.2 En `internal/kickoff/schema.go`: `validCommitReviewModes` y regla en `GateRecord.Validate` (valor fuera del vocabulario o en registro distinto de `tasks`+`approved` se rechaza al escribir y al leer). Pruebas en `internal/kickoff/schema_test.go`: acepta los tres tokens, rechaza `Stop`, `pause`, `always`, cadena vacía explícita en CLI, otra puerta y rechazo; ledger antiguo sin campo; decodificador no estricto anterior ve `approved`.
  Cubre: REQ-33.1, REQ-33.3
- [x] 1.3 En `internal/kickoff/machine.go`: `resolveGateStatus` devuelve el registro que cierra la puerta y `EvaluateGates` copia `CommitReview` (junto a `HandedOff`/`HandedOffTo`). Pruebas en `internal/kickoff/machine_test.go`: el modo viene del primer `approved`, un rechazo posterior no lo cambia, ledger antiguo da modo vacío (efectivo `notify`).
  Cubre: REQ-33.4, REQ-33.6
- [x] 1.4 Crear `internal/cli/sdd_gate_commit_review.go` con `splitCommitReviewFlag`, `validateCommitReviewShape` y `requireCommitReviewStops`. Test de tabla en `internal/cli/sdd_gate_commit_review_test.go`: bandera ausente, en cualquier posición, repetida, sin valor, valor inválido y mayúsculas; resto de argumentos intacto y en orden.
  Cubre: REQ-33.1, REQ-33.2
- [x] 1.5 En `internal/cli/sdd_gate_handoff.go`: generalizar `notApprovedCheck` a `gateStillPendingCheck(parsed, workspaceRoot)` (con `--hand-off`, `--commit-review` o ambas), `gateRecordJSON.CommitReview` (`commitReview,omitempty`) y cabecera de texto con el modo; ajustar firmas de `writeGateRecordJSON`/`writeGateRecordHeader`.
  Cubre: REQ-33.2, REQ-33.6
- [x] 1.6 En `internal/cli/sdd_gate.go`: llamar a `splitCommitReviewFlag` antes de `kickoff.ParseGateRecordArgs` (sin tocar `internal/kickoff/args.go`), comprobaciones de forma y de estado (`checkpointed`), `record.CommitReview`, comprobación bajo el cerrojo, y ayuda con `[--commit-review notify|stop|stop-before-commit]` y su restricción.
  Cubre: REQ-33.2, REQ-33.7
- [x] 1.7 Pruebas de integración en `internal/cli/sdd_gate_commit_review_test.go` sobre `RunSDDGate record`: escribe cada token; viaja con `--hand-off` en un único registro; sin bandera no escribe el campo; rechaza sin escribir con `design`, `rejected`, `--evidence-kind`, `continuous`, sin sellar y con `tasks` ya aprobada (comparación de `gates.yaml`, digest y `kickoff.yaml` byte a byte); el JSON lleva `commitReview`; la ayuda menciona bandera, tres tokens y restricción.
  Cubre: REQ-33.2, REQ-33.6, REQ-33.7

## Fase 2: Proyección en `status` (unidad 2)

- [x] 2.1 En `internal/sddstatus/status_v2.go`: `gateV2.CommitReview` (`commitReview,omitempty`), relleno solo con clave `tasks` y estado `approved` mediante `kickoff.EffectiveCommitReview`. Pruebas en `internal/sddstatus/status_v2_test.go`: `stop` registrado; `notify` efectivo sin campo; con cesión, ambos campos; ausente en `tasks` pendiente, en `design` aprobada y en `continuous`; `nextRecommended` y dependencias idénticos con y sin modo; la pregunta tipada de `tasks` conserva tres opciones y no menciona el modo.
  Cubre: REQ-33.4, REQ-33.5
- [x] 2.2 En `internal/assets/skills/_shared/sdd-status-contract.md`: documentar `commitReview?` en la forma de `gates` y su significado (ausente = `notify`).
  Cubre: REQ-33.4

## Fase 3: Doctrina SDD (unidad 3)

- [x] 3.1 RED: en `internal/components/sdd/orchestrator_shared_sections_test.go` registrar `"Commit Review"` en `sharedOrchestratorSectionNames`, y en `internal/assets/assets_test.go` añadir pines de contenido (a)-(h) de REQ-33.9, de la frase de Block Gates, de la composición ` --commit-review <token>` y de la matriz de amenazas (cada orden git empieza por `git -C <worktree>`; contiene `add -- <paths>` y «never fetch»; no contiene `commit -a`, `--force` ni `git push`; no menciona RDD ni recibos).
  Cubre: REQ-33.8, REQ-33.9
- [x] 3.2 GREEN: sección `### Commit Review` en `internal/assets/skills/_shared/sdd-orchestrator-sections.md` siguiendo el borrador del diseño (aviso, modo, interacción propia de la puerta `tasks`, tarea a tarea, pausa que no es puerta, PR, cambios sobre commits ya pasados, sin subagentes), más la frase de enlace en «SDD Change Kickoff and Block Gates» (sin nombrar `gateQuestion`).
  Cubre: REQ-33.5, REQ-33.8, REQ-33.9
- [x] 3.3 Añadir `### Commit Review` y el marcador `{{GENTLE_AI_SDD_SECTION:Commit Review}}` tras `### Chain Strategy` en los 12 orquestadores: `internal/assets/antigravity/sdd-orchestrator.md`, `internal/assets/codex/sdd-orchestrator.md`, `internal/assets/cursor/sdd-orchestrator.md`, `internal/assets/gemini/sdd-orchestrator.md`, `internal/assets/generic/sdd-orchestrator.md`, `internal/assets/hermes/sdd-orchestrator.md`, `internal/assets/kimi/sdd-orchestrator.md`, `internal/assets/kiro/sdd-orchestrator.md`, `internal/assets/opencode/sdd-orchestrator.md`, `internal/assets/qwen/sdd-orchestrator.md`, `internal/assets/windsurf/sdd-orchestrator.md` e `internal/assets/claude/sdd-orchestrator-workflow.md`. Verificar con `TestSharedOrchestratorSectionsHaveOneSource` y la prueba de ausencia de deriva.
  Cubre: REQ-33.8
- [x] 3.4 Decisión D5 acotada: la doctrina declara la única excepción de «ejecutar sin alterar» las invocaciones del proveedor (añadir ` --commit-review <token>` al final de la invocación de la salida 2 o 3 de la puerta `tasks`), sin tocar la pregunta tipada. Incluida en el texto de 3.2; comprobar los pines de 3.1.
  Cubre: REQ-33.5

## Fase 4: Skills (unidad 4)

- [ ] 4.1 RED: pines en `internal/assets/assets_test.go` para `sdd-apply` (parámetros `commit_review` y `fixup_of`, retorno por tarea, `pending_commit`, registro en el fichero de tareas, en `model-capable` y `model-small`) y para la sección «Review and Fixups» de `work-unit-commits`.
  Cubre: REQ-33.10, REQ-33.11
- [ ] 4.2 GREEN: `internal/assets/skills/sdd-apply/SKILL.md`, ambas secciones: parámetros, una tarea por lanzamiento, `pending_commit` del modo `stop-before-commit` (`git -C <worktree> add -- <paths>` sin commit), retorno compacto, `fixup_of` (publicado con `git branch -r --contains`, `commit --fixup` o commit correctivo), línea de `## Commits con cambios pedidos` (D8), ajuste del cierre y merge sin squash.
  Cubre: REQ-33.10
- [ ] 4.3 Sección «Review and Fixups» idéntica en `internal/assets/skills/work-unit-commits/SKILL.md` y `skills/work-unit-commits/SKILL.md`; ejecutar `internal/assets/skill_copies_parity_test.go` (sin cambios en el test).
  Cubre: REQ-33.11

## Fase 5: ODD (unidad 5)

- [ ] 5.1 RED: pines en `internal/components/agentguidance/routing_test.go` (aviso en el paso 6, viñeta con las reglas comunes y `odd/tasks/<feature>.md`, ausencia de `` `stop` ``, `stop-before-commit`, `--commit-review`, «commit review», «three exits» y «plan stop»); `routing_stops_test.go` y el pin de REQ-27.17 no se tocan.
  Cubre: REQ-33.12
- [ ] 5.2 GREEN: en `internal/components/agentguidance/routing.go`, frase del aviso al final del paso 6 y viñeta «Commits and the pull request» tras «Delivery follows work units», con el texto fijado por el diseño.
  Cubre: REQ-33.12

## Fase 6: Goldens y barrido (unidad 6)

- [ ] 6.1 Regenerar `testdata/golden/*.golden` con `go test ./internal/components/ -run TestGolden -update`, repetir sin `-update` y revisar el diff (orquestadores, `sdd-apply` y documentos que proyectan routing).
  Cubre: REQ-33.8, REQ-33.12
- [ ] 6.2 Barrido de tests afectados por paquete: `go test ./internal/kickoff/ ./internal/sddstatus/ ./internal/cli/ -run "Gate|Status|CommitReview"`, `go test ./internal/assets/`, `go test ./internal/components/sdd/ ./internal/components/agentguidance/`; los paquetes lentos en Windows se lanzan aparte. Cierre: `go vet ./...` (no se compilan binarios de `axiom`).
  Cubre: REQ-33.1, REQ-33.12

## Orden y paralelismo

- Secuencial: unidad 1 antes de la 2 (la 2 consume `GateState.CommitReview`).
- La unidad 3 puede ir en paralelo con 1-2 (sin dependencia de código), pero se aplica en orden de commits.
- La 4 depende de 3 (la doctrina fija el contrato); la 5 es independiente de 3-4.
- La unidad 6 va siempre la última (regenera sobre todo lo anterior).
