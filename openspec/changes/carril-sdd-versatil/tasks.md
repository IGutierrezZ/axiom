# Tareas: Evidencias ISO y mejoras del SDD (carril-sdd-versatil)

> Fase `tasks` · rama `inc/carril-sdd-versatil` · 2026-11-10 · kickoff `checkpointed`, rol único `fullstack` (`blocking`), por eso un único `tasks.md` (sin `tasks.<rol>.md`).
> Entrada: `proposal.md`, las 13 deltas de `specs/` (compuerta `spec` aprobada) y `design.md` (compuerta `design` aprobada), sección «Unidades de trabajo» U1 a U16.
> TDD: desactivado (`strict_tdd: false`). Cada unidad lleva sus pruebas en el mismo commit; las pruebas RED de la matriz de amenazas del design se escriben **antes** del código de producción. Solo `go test` acotado por paquete, **nunca** `go build` ni `go run` de binarios sueltos (Cortex XDR borra `axiom.exe`).
> Cada unidad es un commit Conventional sin atribución de IA. Entrega: una sola PR con `size:exception` (estrategia `exception-ok`, nunca cadena).

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 5.000 a 7.000 autoradas (central ≈ 6.000), adiciones + eliminaciones con pruebas, skills y doctrina; sin contar goldens ni `.codex/AGENTS.md` regenerados |
| 400-line budget risk | High |
| Chained PRs recommended | No (el mantenedor aceptó `size:exception`; preferencia del usuario: nunca cadena) |
| Suggested split | Una sola PR; revisión por commits, uno por unidad (U1 a U16) |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

Nota: la previsión supera la de `proposal.md` §9 (2.500 a 3.500) porque aquella no contaba las pruebas por escenario (254 escenarios en las deltas) ni los textos de skills y doctrina. Es orientativa y no un límite.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| U1 | `verify` solo exige los `role-apply` de roles `blocking` | PR única | `go test ./internal/multirole/ ./internal/sddstatus/ -run 'Barrier\|GatePolicy\|RoleGates\|Verify' -count=1` | `go test` en proceso con fixtures de `kickoff.yaml` y `gates.yaml` en `t.TempDir()` | `multirole/{types,barrier}.go`, `sddstatus/governance_handoff.go`; revert del commit |
| U2 | `kickoff seal` crea la carpeta tras la guarda de rama | PR única | `go test ./internal/cli/ -run 'KickoffSeal' -count=1` | `RunSDD...` en proceso sobre repositorio git temporal | `cli/sdd_kickoff.go`, `sdd_governance.go` |
| U3 | Bloque `sdd:` decodificado y validado | PR única | `go test ./internal/workspace/ -run 'SDD' -count=1` | `axiom workspace validate` simulado vía `parseLayered` y el validador en proceso | `workspace/sdd.go` y canal `SDD` de `declarations.go`, `loader.go`, `types.go` |
| U4 | Proyección `sdd`, `sdd_declaration_invalid`, `iso_store_unsupported`, decorador y `evidenceEvaluator` | PR única | `go test ./internal/sddevidence/ ./internal/sddstatus/ -run 'Evidence\|SDDProjection\|Declaration' -count=1` | `sddstatus.Resolve` en proceso con `axiom.yaml` temporal | `sddevidence/{finding,evaluate}.go` mínimos y `sddstatus/evidence.go` |
| U5 | Frontera `ArchiveTargetResolver` sin cambio observable | PR única | `go test ./internal/workspace/ ./internal/specbaseline/ -count=1` y `go test ./internal/cli/ -run 'ArchiveCompose' -count=1` | Paridad: las pruebas existentes de composición no cambian sus aserciones | `workspace/archive_target.go` y sus tres consumidores |
| U6 | Delta, IDs, consulta REQ-28.25 e identidad ISO | PR única | `go test ./internal/sddevidence/ -run 'Delta\|Lookup\|Identity' -count=1` | Baseline implícita y declarada en `t.TempDir()` | `sddevidence/{delta,lookup}.go` y reglas de identidad |
| U7 | Regla de escenarios (status, `archive-compose`, `specbaseline`) | PR única | `go test ./internal/sddevidence/ ./internal/specbaseline/ -run 'Scenario' -count=1` y `go test ./internal/cli/ -run 'ArchiveCompose.*Scenario' -count=1` | `archive-compose --document` en proceso con deltas válidas e inválidas | Llamadas de `ScenarioViolations` en `cli` y `specbaseline` |
| U8 | `story.md` y perfil de bug | PR única | `go test ./internal/sddevidence/ ./internal/sddstatus/ -run 'Profile\|Story\|CoreReady' -count=1` | Cambios temporales con `story.md` y sin delta | `sddevidence/profile.go`, localizador y `coreReady` |
| U9 | `Cubre:` y cobertura (incluido bug sin delta) | PR única | `go test ./internal/sddevidence/ -run 'Cubre\|Coverage' -count=1` | `tasks*.md` temporales con líneas `Cubre:` | `sddevidence/cubre.go` y reglas de cobertura |
| U10 | Design proporcional, `design.<rol>.md`, plan propio, criterios de la compuerta y aviso de verify | PR única | `go test ./internal/sddevidence/ ./internal/sddstatus/ ./internal/kickoff/ -run 'Design\|RolePlan\|Trigger' -count=1` | `git diff --name-only` real sobre repositorio temporal para el aviso | `sddevidence/{design,plan,git}.go` y `sddstatus/governance*.go` |
| U11 | Matriz de verify y bloqueo del archivado ISO | PR única | `go test ./internal/sddevidence/ -run 'Matrix' -count=1` y `go test ./internal/cli/ -run 'ArchiveCompose.*Matrix' -count=1` | `verify-report.md` temporal con filas pass, fail y manuales | `sddevidence/matrix.go` y su decorador |
| U12 | Archivado en el mismo PR (rama, veredicto, publicación, compuerta `integration`), más U12a y U12b | PR única | `go test ./internal/branch/... ./internal/kickoff/ -count=1` y `go test ./internal/cli/ -run 'Gate\|Publish\|Archive' -count=1` | Repositorios git temporales con observador de git para el push | `branch/*`, `branch/engine/*`, `kickoff/machine.go`, `cli/sdd_gate*.go` |
| U13 | `IntegrationCommit` y verbo `axiom sdd trace` | PR única | `go test ./internal/sddevidence/ ./internal/gitcommon/ -run 'Trace\|Integration\|Architecture' -count=1` y `go test ./internal/cli/ -run 'SDDTrace' -count=1` | Repositorios git temporales con squash, merge y rebase | `sddevidence/{git,trace}.go`, `cli/sdd_trace.go`, registro en `cmd/axiom/main.go` |
| U14 | Skills `sdd-*`, doctrina compartida, manual y ejemplo | PR única | `go test ./internal/assets/... ./internal/components/agentguidance/ -count=1` y `go test ./internal/workspace/ -run 'ManualDeInicio' -count=1` | N/A: texto; la paridad se comprueba con los pines de los tests | Ficheros de prosa de la unidad |
| U15 | `sdd-orchestrator-sections.md` y `routing.go` (D4) | PR única | `go test ./internal/assets/... ./internal/components/agentguidance/ ./internal/components/sdd/ -count=1` | N/A: texto; goldens en U16 | `sdd-orchestrator-sections.md`, `routing.go` |
| U16 | Regeneración de goldens y copias versionadas | PR única | `go test ./internal/components/ -run 'Golden' -count=1` (sin `-update`) | `go test ./internal/components/ -run 'Golden' -update -count=1` y repaso sin el indicador | `testdata/golden/*.golden` y `.codex/AGENTS.md` |

## Resoluciones de la compuerta design

Se aplican tal cual las decisiones tomadas al aprobar `design`. No se edita `design.md` ni ninguna delta (su digest reabriría las compuertas).

| Pregunta | Resolución | Dónde se materializa |
|---|---|---|
| P1 | Verbo nuevo `axiom sdd trace`, de solo lectura (D16); no se amplía `integrate show`. | U13 |
| P2 | `status` adelanta `iso_scenarios_missing` en ISO desde que existe la delta, además del rechazo al componer. | U7 |
| P3 | Un `sdd:` inválido bloquea todas las fases (fallo cerrado, D4). | U4 |
| P4 | El modo ISO exige almacén `openspec` o `hybrid` (`iso_store_unsupported`). | U4 |
| P5 | `sdd.archive` se fija en la instantánea del kickoff. | U12a (hueco de spec, ver abajo) |
| P6 | `axiom branch done` con `same_pr` acepta la derivación por `--first-parent` como evidencia de fusión. | U12b (hueco de spec, ver abajo) |
| P7 | Regeneración de copias versionadas con `go test` en proceso y copia por bloques de `.codex/AGENTS.md`; método concreto abajo. | U16 |

### Huecos de spec (para la compuerta `tasks`)

P5 y P6 **no tienen requisito en las deltas aprobadas**. Se planifican como U12a y U12b sin `Cubre:`, y el orquestador debe presentarlo en la compuerta `tasks` para decidir entre (a) añadir un requisito por delta antes de aplicar (reabre `spec`), (b) aplicarlos como decisiones de diseño sin requisito, o (c) retirarlos. Además:

- **P5 choca con el design aprobado.** D13 descarta la opción por cambio y el design declara que `kickoff.yaml` no cambia de esquema (disparador `data-model` en `no`). Fijar `sdd.archive` en la instantánea añade una clave a `kickoff.yaml`, y D6 recuerda que un binario anterior rechaza claves nuevas de ese fichero. U12a debe ser aditivo y omitible (clave ausente = lectura en vivo, como hoy) y revisar con la compuerta que el disparador `data-model` pase a `sí`.
- **P6 no aparece en D13.** La capa de evidencia de fusión de `axiom branch done` con `same_pr` no está descrita en el design; U12b lo cubre con la prueba que indica el apartado de la tarea.

## Resumen del orden y de las dependencias

```
Sin dependencias:   U1   U2   U3   U5
Tras U3:            U4 (U3)    U12 (U3)
Tras U4 y U5:       U6
Tras U6:            U7    U9 (U6,U8)    U11 (U6,U9)
Tras U4:            U8    U10
Tras U9 y U11:      U13
Tras U1 a U13:      U14
Tras U14 y cadena:  U15 -> U16
```

Se aplican en el orden del design (U1 a U16) con un único escritor. El paralelismo lógico es: U1, U2, U3 y U5 no comparten dependencias entre sí; U8 y U10 son independientes entre sí tras U4; U12 solo espera a U3.

Estado de la cadena de prosa del orquestador (C14), comprobado sobre `main` b7ddbf51 sin ejecutar git (la sesión no dispone de Bash):

| Eslabón | Estado observado | Evidencia |
|---|---|---|
| 1. Arreglos T9 y adaptadores T1 | Hecho | #229 y #227 (`proposal.md` §7) |
| 2. Revisión nativa T6 (#233) | **En `main`** | `git log` del arranque: `8c520295 Merge pull request #233` |
| 3. Guía del orquestador T1 | Fusionada (#226) | `odd/tasks/upstream-revision-nativa.md:226` («La guía del orquestador T1 (#226) se fusionó antes que T6»). El documento `odd/tasks/guia-orquestador-upstream.md` de `main` aún la muestra sin marcar: está desfasado |
| 4. Guía T2 y T3 | **Falta** (sin evidencia de fusión) | Las dos tareas siguen sin marcar en `odd/tasks/guia-orquestador-upstream.md:90-111`; existen la rama local `inc/guia-orquestador-upstream` y `origin/feat/guia-orquestador-upstream` sin merge en la historia reciente de `main` |

Consecuencia: **U15 y U16 esperan a la guía T2/T3** (tocan `sdd-orchestrator-sections.md`, `routing.go` y sus goldens). **U1 a U14 pueden aplicarse ya**: ninguna toca esa prosa. Antes de U15, rebasar sobre `origin/main` y reconfirmar con `git log origin/main --oneline -- internal/assets/skills/_shared/sdd-orchestrator-sections.md` que T2/T3 están dentro.

## Método de regeneración de copias versionadas (P7)

Sacado de las tareas T-07 y T-27 de `openspec/changes/archive/2026-11-10-carril-proyecto-publicacion-docs/tasks.md` (#235), de su `verify-report.md:50` y de `odd/tasks/upstream-revision-nativa.md` (T6, T14, T16):

1. **Goldens** (13 en `testdata/golden/`): `go test ./internal/components/ -run Golden -update -count=1` y, a continuación, `go test ./internal/components/ -run Golden -count=1` sin `-update`. El indicador `update` está declarado en `internal/components/golden_test.go:32`. Solo el paquete `internal/components`, nunca sus subpaquetes que compilan binarios. Los goldens no se concilian a mano.
2. **`.codex/AGENTS.md`**: se actualiza copiando por **bloques** el texto cambiado del bloque gestionado de `testdata/golden/sdd-codex-agentsmd.golden`. No se copia el fichero entero porque su marca y su tabla de modelos difieren del golden (#236, `verify-report.md:50`). No se ejecuta `axiom sync`.
3. **`AGENTS.md` y `GEMINI.md` de la raíz**: no son salida generada (no contienen el bloque gestionado del orquestador); son documentos del repositorio y se editan a mano solo si el texto que describen cambia (en #235 ocurrió con la parada de plan y las tres salidas). Para este cambio, solo si U15 cambia lo que ellos citan sobre el bloque ODD (D4); si no, no se tocan.
4. **Pines**: tras regenerar, ajustar los pines de `internal/workspace/docs_contract_test.go` y, si cambian los tamaños del protocolo, los techos de coste y el hash de Kilocode de `internal/components/sdd/review_ledger_contract_test.go` con el comentario «Deliberate, not drift», como hicieron T6, T14 y T16 de la revisión nativa.
5. **Comprobación final**: `go test ./internal/components/... ./internal/assets/... -count=1`. El único fallo de entorno conocido en Windows es `filemerge` (2 tests de junction), ajeno al cambio.

Comando concreto de U16:

```
go test ./internal/components/ -run Golden -update -count=1
go test ./internal/components/ -run Golden -count=1
```

## Fase 1: Correcciones independientes (piezas 1 y 3)

- [ ] U1 · QA diferido no bloquea `verify`: partición `blocking` única (pieza 1, D11)
  - Objetivo: que `verify` solo exija el `role-apply` aprobado de los roles `blocking`, con la misma partición en la barrera y en el status; los `deferred` y `optional` se informan.
  - Ficheros: `internal/multirole/types.go` (`GatePolicy.Blocks()`: verdadero para `blocking`, vacío y desconocido, fallo cerrado), `internal/multirole/barrier.go` (`EvaluateBarrier` usa `Blocks()`), `internal/sddstatus/governance_handoff.go` (`verifyDependencyFromRoleGates` recorre `governance.Kickoff.Config.Roles` y exige solo los bloqueantes; nota `verify_nonblocking_roles`).
  - Pruebas: RED primero en `internal/multirole/barrier_test.go` y en las pruebas de `internal/sddstatus/` (nombres nuevos `TestGatePolicyBlocks*`, `TestVerifyDependencyIgnoresNonBlockingRoles*`): `blocking` aprobado + `deferred` sin aprobar desbloquea verify; `blocking` sin aprobar bloquea; política vacía o desconocida bloquea; la nota informa rol, política y estado de `role-apply`. Un solo rol `fullstack` no cambia.
  - Comando: `go test ./internal/multirole/ ./internal/sddstatus/ -run 'Barrier|GatePolicy|RoleGates|Verify' -count=1`
  - Commit: `fix(multirole): desbloquear verify solo con los role-apply de roles blocking`
  - Dependencias: ninguna.
  - Cubre: REQ-21.15, REQ-2.1

- [ ] U2 · `kickoff seal` crea la carpeta del cambio (pieza 3, D12)
  - Objetivo: que `axiom sdd kickoff seal` funcione sin crear la carpeta a mano, creándola solo tras la guarda de rama y sin tocar una carpeta existente.
  - Ficheros: `internal/cli/sdd_kickoff.go` (`runSDDKickoffSeal`: resolver la raíz sin exigir existencia, validar el nombre con `gitref.ValidBranchComponent`, rechazar un cambio ya archivado, ejecutar `applySealBranchPolicy`, después `os.MkdirAll`; limpiar la carpeta que creó esta ejecución si `kickoff.Seal` falla y sigue vacía), `internal/cli/sdd_governance.go` (variante de `resolveGovernanceChangeRoot` sin exigir existencia). `--infer` sigue exigiendo la carpeta.
  - Pruebas: RED primero en `internal/cli/` (fichero nuevo `sdd_kickoff_seal_folder_test.go`): carpeta inexistente se crea; nombre inválido se rechaza sin crear nada; guarda de rama que rechaza no deja carpeta; fallo de `Seal` retira la carpeta vacía creada y respeta una existente; `--infer` sin carpeta sigue fallando; cambio archivado se rechaza.
  - Comando: `go test ./internal/cli/ -run 'KickoffSeal' -count=1`
  - Commit: `feat(kickoff): crear la carpeta del cambio al sellar el kickoff`
  - Dependencias: ninguna.
  - Cubre: REQ-28.20

## Fase 2: Declaración `sdd:` y capa de evidencia

- [ ] U3 · Bloque `sdd:` en `axiom.yaml` y `workspace validate` (pieza 4, D1)
  - Objetivo: decodificar con lista cerrada de claves el bloque `sdd:` (`iso`, `archive`, `design_triggers`) y exponerlo ya resuelto.
  - Ficheros: `internal/workspace/sdd.go` (nuevo: `SDDConfig`, `ArchiveMode`, `BaseDesignTriggers`, `decodeSDD`, `resolveSDD`, `SamePRArchive`), `internal/workspace/declarations.go` (canal `SDD`/`SDDErr`; `readDeclarationBlocks` recibe la topología), `internal/workspace/loader.go` (`joinLoadErrors`), `internal/workspace/types.go` (`WorkspaceConfig.SDD`).
  - Pruebas: RED primero en `internal/workspace/sdd_test.go` (nuevo), tablas sobre `parseLayered` para todos los escenarios de REQ-28.23: `iso` no booleano (`"true"`, `yes`, `1`) rechazado; `archive` fuera de `branch|same_pr`; `same_pr` en multirepo rechazado con opción y topología nombradas; `design_triggers` vacío, repetido, fuera del patrón `^[a-z0-9]+(-[a-z0-9]+)*$` o igual a un base; `sdd:` nulo, escalar o lista; claves desconocidas y repetidas; valores por defecto (`iso: false`, `archive: branch`, base primero). `workspace validate` informa los errores.
  - Comando: `go test ./internal/workspace/ -run 'SDD' -count=1`
  - Commit: `feat(workspace): declarar el bloque sdd en axiom.yaml`
  - Dependencias: ninguna.
  - Cubre: REQ-28.23, REQ-28.1

- [ ] U4 · Proyección `sdd`, `sdd_declaration_invalid`, `iso_store_unsupported`, decorador y `evidenceEvaluator` (pieza 4, D3, D4, D5, D17)
  - Objetivo: dejar montada la capa de proyección: paquete `sddevidence` mínimo (`Finding`, `Severity`, `Reason`, `Input`, `Evaluate` con los dos hallazgos de declaración), decorador del status y composición de bloqueos y notas.
  - Ficheros: `internal/sddevidence/finding.go` y `internal/sddevidence/evaluate.go` (nuevos), `internal/sddstatus/evidence.go` (nuevo: `decorateEvidence`, variable `evidenceEvaluator`, ruta y dependencias según las reglas 1 a 5 de D3), `internal/sddstatus/{status,status_v2,workspace_declarations}.go` (clave aditiva `sdd` omitible; `sdd.source: invalid`; entrada en `invalidDeclarationErrors`), `internal/sddstatus/main_test.go` (`TestMain` desactiva `evidenceEvaluator` para las pruebas antiguas).
  - Pruebas: RED primero: `sdd` omitido sin bloque y proyectado con él (REQ-28.24); `blocked(sdd_declaration_invalid)` en cualquier fase con la continuación `axiom workspace validate --path <maestra>` (P3); `blocked(iso_store_unsupported)` con almacén `engram` o `none` y continuación `openspec`/`hybrid` (P4); `wrong_branch` sigue el primero y mantiene `resolve-blockers`; todos los bloqueos concurrentes se proyectan en el orden de la tabla; las notas no cambian ruta ni dependencias; `axiom sdd continue` refleja el bloqueo; el JSON solo crece. Prueba de arquitectura: `sddevidence` no importa `specbaseline`.
  - Comando: `go test ./internal/sddevidence/ ./internal/sddstatus/ -run 'Evidence|SDDProjection|Declaration|Architecture' -count=1`
  - Commit: `feat(sddstatus): proyectar el bloque sdd y los hallazgos de evidencia en el status`
  - Dependencias: U3.
  - Cubre: REQ-28.1, REQ-28.24, REQ-28.27, REQ-25.42

- [ ] U5 · Frontera del destino del archivado y su adopción sin cambio observable (pieza 11, D15)
  - Objetivo: introducir `ArchiveTargetResolver` con una sola implementación y que `composeDocumentMode` y la resolución de dominios de `specbaseline` la usen, conservando los mensajes de error byte a byte.
  - Ficheros: `internal/workspace/archive_target.go` (nuevo: `ArchiveTargetKind`, `ArchiveTarget`, `ArchiveTargetResolver`, `ResolveArchiveTarget`), `internal/cli/sdd_archive_compose.go` (`composeDocumentMode`), `internal/specbaseline/plan.go`.
  - Pruebas: paridad, sin tocar las aserciones existentes de `specbaseline` y `cli`; pruebas nuevas de `ResolveArchiveTarget` para `native`, `profiled` y `prose` y de que los mensajes de error no cambian. La interfaz tiene un único implementador usado (ratchet de código muerto).
  - Comando: `go test ./internal/workspace/ ./internal/specbaseline/ -count=1` y `go test ./internal/cli/ -run 'ArchiveCompose' -count=1`
  - Commit: `refactor(workspace): resolver el destino del archivado tras una frontera`
  - Dependencias: ninguna.
  - Cubre: REQ-24.47

- [ ] U6 · Delta, IDs, consulta REQ-28.25 e identidad ISO (pieza 4, D2, D7)
  - Objetivo: analizar las deltas, extraer el identificador de cada requisito (perfil declarado o baseline implícita) y consultar un requisito existente en la baseline; bloquear en ISO la falta de `identity: id`.
  - Ficheros: `internal/sddevidence/delta.go` (`ParseDelta`, `DeltaRequirement`), `internal/sddevidence/lookup.go` (`LookupRequirement`), `internal/sddevidence/evaluate.go` (hallazgos `iso_identity_required` e `iso_requirement_id_missing`), `internal/sddevidence/headings.go` (escáner ATX propio fuera de vallas, sin importar `specbaseline`).
  - Pruebas: RED primero en `internal/sddevidence/delta_test.go` y `lookup_test.go`: baseline declarada con `identity: text` bloquea en ISO con motivo accionable; con `identity: id` y `scenarios: omit` pasa; baseline implícita cuenta como `id` (ID del nombre `\((REQ-[0-9]+\.[0-9]+)\)\s*$`); ADDED/MODIFIED/REMOVED/RENAMED; requisito sin ID extraíble; consulta de REQ existente e inexistente; fuera de ISO no bloquea nada.
  - Comando: `go test ./internal/sddevidence/ -run 'Delta|Lookup|Identity|Heading' -count=1`
  - Commit: `feat(sddevidence): identidad de requisitos ISO, analisis de deltas y consulta en la baseline`
  - Dependencias: U4, U5.
  - Cubre: REQ-28.2, REQ-28.25

## Fase 3: Reglas ISO y perfiles

- [ ] U7 · Regla de escenarios en status, `archive-compose` y `specbaseline` (pieza 2, D14, P2)
  - Objetivo: rechazar en ISO un requisito ADDED o MODIFIED sin escenarios, antes de escribir nada, y adelantar el mismo hallazgo en el status.
  - Ficheros: `internal/sddevidence/scenarios.go` (`ScenarioViolations`), `internal/sddevidence/evaluate.go` (`iso_scenarios_missing` desde que existe la delta, P2), `internal/cli/sdd_archive_compose.go` (valida **todas** las deltas del cambio antes de escribir, vía `changeRootOfDelta`; nunca con `--canonical`), `internal/specbaseline/plan.go` (`Prepare` y `ApplyCompose`, validación previa a cualquier escritura).
  - Pruebas: RED primero: ADDED y MODIFIED sin escenarios rechazados con `sdd-archive-compose: blocked(iso_scenarios_missing): dominio <d>, sección <S>, requisito <n>`; `scenarios: omit` sigue aplicando la regla a la delta; REMOVED y RENAMED no la exigen; fuera de ISO no cambia; sin contexto de proyecto (`--canonical`) no aplica; un dominio sin escenarios impide modificar los demás (todo o nada, también con baseline implícita); paridad nativa byte a byte sin ISO.
  - Comando: `go test ./internal/sddevidence/ ./internal/specbaseline/ -run 'Scenario' -count=1` y `go test ./internal/cli/ -run 'ArchiveCompose' -count=1`
  - Commit: `feat(archive): exigir al menos un escenario por requisito en ISO`
  - Dependencias: U6.
  - Cubre: REQ-28.21, REQ-24.38, REQ-24.46

- [ ] U8 · `story.md` y perfil de bug (pieza 8, D6)
  - Objetivo: aceptar `story.md` en lugar de `proposal.md` y el perfil de bug (con y sin delta), sin romper el formato actual.
  - Ficheros: `internal/sddevidence/profile.go` (`ReadProfile`, front matter de lista cerrada `profile`/`external_ref`), `internal/sddstatus/status.go` (`resolveArtifactPaths` toma `proposal.md` y, si falta, `story.md` como localizador de `artifactPaths.proposal`; marcadores de cambio en `openSpecChangeContainer`, `explorationOnlyChangeDir` y en `status.go` cerca de `:442` y `:491`; `coreReady` por perfil; bug sin delta: `dependencies.specs` en `all_done`, compuerta `spec` sin abrir), `internal/sddstatus/status_v2.go`.
  - Pruebas: RED primero: el formato actual produce el mismo estado; `story.md` con delta prepara el núcleo; bug sin delta prepara el núcleo; sin propuesta ni historia no está preparado; sin delta fuera del bug no está preparado; `proposal.md` y `story.md` juntos prevalece `proposal.md` con `note(story_ignored)`; front matter con clave desconocida, `profile` inválido o YAML ilegible da `blocked(story_profile_invalid)`; en `hybrid` el topic sigue siendo `sdd/<cambio>/proposal`.
  - Comando: `go test ./internal/sddevidence/ ./internal/sddstatus/ -run 'Profile|Story|CoreReady' -count=1`
  - Commit: `feat(sddstatus): aceptar story.md y el perfil de bug como núcleo del cambio`
  - Dependencias: U4.
  - Cubre: REQ-28.10, REQ-28.11, REQ-28.26

- [ ] U9 · `Cubre:` y cobertura, incluido el bug sin delta (pieza 5, D7)
  - Objetivo: atribuir cada línea `Cubre:` a su tarea y bloquear en ISO si algún requisito de la delta (o el REQ existente del bug sin delta) no aparece en ninguna.
  - Ficheros: `internal/sddevidence/cubre.go` (`ParseCubre`, `CubreLine`), `internal/sddevidence/evaluate.go` (`iso_coverage_missing`, `iso_coverage_unknown_requirement`, `note(iso_coverage_orphan)`).
  - Pruebas: RED primero: identificadores con el patrón `[A-Za-z][A-Za-z0-9]*-[0-9]+(\.[0-9]+)*` (`REQ-28.1`, `RF-120`, `NFR-001`); línea fuera de una tarea no cuenta y da nota en ISO; requisito de la delta sin tarea bloquea; `Cubre:` a requisito inexistente bloquea; bug sin delta exige al menos un `Cubre:` y que cada uno exista en la baseline (consulta de U6); varias `tasks.<rol>.md` se evalúan en conjunto; fuera de ISO no proyecta nada.
  - Comando: `go test ./internal/sddevidence/ -run 'Cubre|Coverage' -count=1`
  - Commit: `feat(sddevidence): validar la cobertura Cubre de los requisitos en ISO`
  - Dependencias: U6, U8.
  - Cubre: REQ-28.3, REQ-28.11

- [ ] U10 · Design proporcional, `design.<rol>.md`, plan propio, criterios de la compuerta y aviso de verify (pieza 9, D9, D10)
  - Objetivo: validar el nivel mínimo y completo del design por disparadores, el detalle por rol, el plan propio de `tasks.<rol>.md`, presentar los disparadores marcados en la compuerta `design` y avisar en verify de los no marcados.
  - Ficheros: `internal/sddevidence/design.go` (`ParseDesign`, secciones y alias, tabla `| Disparador | Marca | Motivo |`), `internal/sddevidence/plan.go` (`ParseRolePlan`: identificadores únicos, commit no vacío, referencias existentes, sin ciclos ni autodependencia, sin par «en paralelo» con dependencia), `internal/sddevidence/git.go` (`ChangedPaths` con `gitcommon.RunWithTimeout`; heurísticas por ruta solo para disparadores base marcados `no`), `internal/sddevidence/evaluate.go` (`design_incomplete`, `design_role_unknown`, `role_plan_missing`, `design_triggers_marked`, `design_trigger_unmarked`, severidad por ISO), `internal/sddstatus/{governance,governance_handoff}.go` (criterio «Disparadores marcados: <ids>; revisor esperado: arquitecto», proyección `designTriggers: {marked, reviewer}`), `internal/kickoff/` (`RoleTaskFiles` para `design.<rol>.md` y su roster).
  - Pruebas: RED primero por escenario de REQ-28.12 a REQ-28.17: mínimo y mínimo de bug; nivel completo con alguna marca `sí`; `maintained-context` mantiene `context-impact.md`; disparadores propios de `design_triggers`; aviso de verify solo en `verify` o `archive` y sin git no avisa ni bloquea; `design.<rol>.md` de un rol fuera del roster da `design_role_unknown` y su ausencia nunca es hallazgo; plan con ciclo, referencia inexistente o paralelo con dependencia inválido; fuera de ISO todo es nota, en ISO bloquea; en `continuous` nota y en `checkpointed` criterio de la compuerta; sin marcas la compuerta no cambia.
  - Comando: `go test ./internal/sddevidence/ ./internal/sddstatus/ ./internal/kickoff/ -run 'Design|RolePlan|Trigger|Gate' -count=1`
  - Commit: `feat(sddevidence): design proporcional, design por rol y plan propio por rol`
  - Dependencias: U4.
  - Cubre: REQ-28.12, REQ-28.13, REQ-28.14, REQ-28.15, REQ-28.16, REQ-28.17, REQ-21.9, REQ-1.2

- [ ] U11 · Matriz de verify y bloqueo del archivado ISO (pieza 6, D8)
  - Objetivo: validar la tabla de nueve columnas de `verify-report.md` y exigir en ISO que sea válida y con éxito para archivar.
  - Ficheros: `internal/sddevidence/matrix.go` (`ParseMatrix`, `MatrixRow`), `internal/sddevidence/evaluate.go` (`iso_matrix_invalid`, `iso_matrix_failed` en fase archive; bloquean solo `dependencies.archive`), `internal/cli/sdd_archive_compose.go` (defensa en profundidad con la delta de un cambio).
  - Pruebas: RED primero por escenario de REQ-28.4 y REQ-28.5: falta el fichero, la sección o los encabezados exactos; escenario sin fila; fila con REQ o escenario inexistente; `Tipo` o `Resultado` fuera de dominio; manual sin `Quién`, `Cuándo` (`AAAA-MM-DD`) y `Motivo`; varias filas para un escenario deben pasar todas; bug sin delta con REQ sin escenarios exige la fila `—`; resultado `fail` bloquea; una matriz pendiente no bloquea la planificación ni apply; fuera de ISO `verify-report.md` libre o ausente archiva como hoy.
  - Comando: `go test ./internal/sddevidence/ -run 'Matrix' -count=1` y `go test ./internal/cli/ -run 'ArchiveCompose' -count=1`
  - Commit: `feat(sddevidence): matriz de verify por escenario y bloqueo del archivado ISO`
  - Dependencias: U6, U9.
  - Cubre: REQ-28.4, REQ-28.5

## Fase 4: Archivado en el mismo PR y trazabilidad

- [ ] U12 · Archivado en el mismo PR: rama, veredicto, publicación y compuerta `integration` (pieza 10, D13)
  - Objetivo: con `sdd.archive: same_pr` en monorepo, archivar en la rama del cambio, sin rama `-archive`, sin `change_not_integrated` y sin compuerta `integration`.
  - Ficheros: `internal/branch/planner.go`, `internal/branch/planner_archive.go`, `internal/branch/verdict.go` (`PlanInput.SamePRArchive`, `VerdictInput.SamePRArchive`, `ArchiveBranchApplies`, continuación `axiom branch ensure --cwd <ruta> --change <c>` sin `--archive`), `internal/branch/engine/{engine,verdict,publication,archive_plan}.go` (relleno en `gatherWith`; `PublicationTarget` y `CheckArchiveRequest` usan el predicado único; rechazo `archive_branch_not_applicable`), `internal/kickoff/machine.go` (`Inputs.SkipIntegration`), `internal/sddstatus/governance.go` (`loadGovernance` lo recibe; `archiveDependencyFromGovernance` no bloquea con `same_pr`), `internal/cli/sdd_gate.go` y `internal/cli/sdd_gate_handoff.go` (`integration_gate_not_applicable` sin escribir en `gates.yaml`; `gate show` no la lista).
  - RED de la matriz de amenazas (antes del código de producción):
    - Estado del commit: `publish --archive` en la rama del cambio con un fichero de código modificado; el commit no lo incluye (solo rutas de `docpaths`).
    - Estado del push: rama de rol y `main` con `same_pr` dan `wrong_branch` nombrando la rama del cambio, con observador de git que verifica que no hay `push`.
  - Pruebas: por escenario de REQ-28.22, REQ-25.18, REQ-25.45, REQ-25.42, REQ-25.39, REQ-21.16, REQ-27.14 y REQ-26.1: sin rama `-archive` ni PR documental; `publish --archive` usa la rama del cambio; desde integración se rechaza nombrándola; con varios roles el archivado va en la rama del cambio y una rama de rol bloquea en `archive`; commit de código sin subir se informa y el archivado se conserva; sin evidencia `pr_merged` el archivado procede; la compuerta `integration` no se aplica y registrarla se rechaza; `ensure --archive` rechazado con `archive_branch_not_applicable`; la verificación sigue siendo precondición; `docs_publication: auto` publica tras archivar; sin la opción y en multirepo nada cambia (tablas de `Decide`, `CheckArchiveRequest`, `PublicationTarget`).
  - Comando: `go test ./internal/branch/... ./internal/kickoff/ ./internal/sddstatus/ -count=1` y `go test ./internal/cli/ -run 'Gate|Publish|Archive' -count=1`
  - Commit: `feat(branch): archivar en el mismo PR como opcion por proyecto en monorepo`
  - Dependencias: U3.
  - Cubre: REQ-28.22, REQ-25.18, REQ-25.45, REQ-25.42, REQ-25.39, REQ-21.16, REQ-27.14, REQ-26.1

- [ ] U12a · Fijar `sdd.archive` en la instantánea del kickoff (P5) — HUECO DE SPEC, sin REQ
  - Objetivo: que cambiar `sdd.archive` con cambios a medio archivar no mezcle reglas: el modo se congela al sellar.
  - Ficheros: `internal/kickoff/` (clave aditiva y omitible en la instantánea de `kickoff.yaml`; ausente = lectura en vivo, como hoy), `internal/cli/sdd_kickoff.go` (relleno al sellar), `internal/workspace/sdd.go` y `internal/branch/` (el predicado lee la instantánea si existe).
  - Pruebas: RED primero: cambio sellado con `branch` sigue con rama `-archive` aunque `axiom.yaml` pase a `same_pr`, y al revés; cambio anterior sin la clave lee en vivo; un `kickoff.yaml` con la clave se carga sin error con el esquema actual.
  - Comando: `go test ./internal/kickoff/ ./internal/branch/... -count=1` y `go test ./internal/cli/ -run 'KickoffSeal' -count=1`
  - Commit: `feat(kickoff): fijar sdd.archive en la instantánea del kickoff`
  - Dependencias: U2, U12.
  - Sin REQ: hueco de spec; decidir en la compuerta `tasks` (ver «Huecos de spec»). Si se retira, U12 queda con lectura en vivo y la skill y el manual lo advierten.

- [ ] U12b · `axiom branch done` con `same_pr` acepta `--first-parent` como evidencia de fusión (P6) — HUECO DE SPEC, sin REQ
  - Objetivo: que con squash y git anterior a 2.38 `done` confirme la fusión por derivación `--first-parent`, sin exigir `--force`.
  - Ficheros: `internal/branch/engine/` (camino de `done`; localizar con `rg "done" internal/branch/engine`), reutilizando `sddevidence.IntegrationCommit` de U13 (por eso va después de U13 en el orden de aplicación aunque cuelgue de U12 en el design).
  - Pruebas: RED primero con repositorios git temporales: squash, merge y rebase confirman la fusión; ruta archivada ausente en la principal no la confirma; sigue existiendo `--force`.
  - Comando: `go test ./internal/branch/... ./internal/sddevidence/ -run 'Done|Integration' -count=1`
  - Commit: `feat(branch): confirmar la fusión de done con first-parent en same_pr`
  - Dependencias: U12, U13.
  - Sin REQ: hueco de spec; decidir en la compuerta `tasks`.

- [ ] U13 · `IntegrationCommit` y verbo `axiom sdd trace` (pieza 7, D16, P1)
  - Objetivo: reconstruir la cadena requisito → tareas `Cubre:` → filas de la matriz → commit de integración, para cambios activos o archivados, sin escribir nada.
  - Ficheros: `internal/sddevidence/git.go` (`IntegrationCommit`, `ChangedPaths`), `internal/sddevidence/trace.go`, `internal/cli/sdd_trace.go` (nuevo; esquema `axiom.sdd-trace/v1`), `cmd/axiom/main.go` (registro del verbo), `internal/gitcommon/architecture_test.go` (incluir `internal/sddevidence` en `TestArchitectureOnlyExecGoLaunchesProcesses`).
  - RED de la matriz de amenazas (antes del código de producción), selección del repositorio git:
    - Raíz git relativa rechazada.
    - Ruta archivada con `..` o que empieza por `-` rechazada (mismo criterio que `gitcli.pathspecArgument`); `ref` validado con `plainArgument`.
    - Ejecutar desde otro cwd da el mismo commit.
  - Pruebas: por escenario de REQ-28.6: squash, merge y rebase derivan el mismo commit con `git log --first-parent --diff-filter=A --reverse` sobre `origin/<integración>` o `<integración>`, sin `fetch`; ruta ausente da `found=false` sin error; los trailers intermedios no se validan; `trace` no escribe; con baseline implícita funciona sin `integration.yaml`; con baseline declarada incluye `pr_merged` e `idMap`.
  - Comando: `go test ./internal/sddevidence/ ./internal/gitcommon/ -run 'Trace|Integration|Architecture' -count=1` y `go test ./internal/cli/ -run 'SDDTrace' -count=1`
  - Commit: `feat(sdd): verbo trace y commit de integración por first-parent`
  - Dependencias: U9, U11.
  - Cubre: REQ-28.6, REQ-28.22

## Fase 5: Prosa, doctrina y copias versionadas

- [ ] U14 · Skills `sdd-*`, doctrina compartida, manual y ejemplo (pieza 13)
  - Objetivo: describir en las skills las opciones nuevas, la excepción ISO a «Missing or failed verification never gates archive» y los tokens y claves del status, en paridad entre runtimes. No toca `sdd-orchestrator-sections.md` ni `routing.go`.
  - Ficheros: `internal/assets/skills/sdd-propose/SKILL.md`, `sdd-spec/SKILL.md`, `sdd-design/SKILL.md`, `sdd-tasks/SKILL.md`, `sdd-apply/SKILL.md`, `sdd-verify/SKILL.md`, `sdd-archive/SKILL.md` (`story.md`, bug, design proporcional, `Cubre:`, plan, matriz, mismo PR), `internal/assets/skills/_shared/persistence-contract.md` (excepción ISO declarada, línea ~69), `_shared/sdd-status-contract.md` (tokens y claves nuevas), `_shared/openspec-convention.md` (carpeta creada por `seal`), `docs/manual-de-inicio.md`, `axiom.example.yaml` (bloque `sdd:`, ISO, perfiles, mismo PR, `axiom sdd trace`; advertir de cambiar `sdd.archive` con cambios a medio archivar), `internal/workspace/docs_contract_test.go` (pines).
  - Pruebas: pines de texto y paridad entre runtimes en `internal/assets/` y `internal/components/agentguidance/`; pines del manual intactos más los nuevos; el ejemplo `axiom.example.yaml` se carga con el validador. Si tras esta unidad algún golden de skills queda rojo, regenerarlo en este mismo commit y dejar para U16 solo lo del orquestador (desvío de «una sola vez» a decidir al aplicar; ver riesgos).
  - Comando: `go test ./internal/assets/... ./internal/components/agentguidance/ -count=1` y `go test ./internal/workspace/ -run 'ManualDeInicio|DocsContract' -count=1`
  - Commit: `docs(skills): documentar el modo ISO, los perfiles y el archivado en el mismo PR`
  - Dependencias: U1 a U13.
  - Cubre: REQ-28.28

- [ ] U15 · Prosa del orquestador y D4 en el bloque ODD (pieza 12) — DEPENDE DE LA CADENA DE PROSA
  - Objetivo: que la compuerta `design` presente los disparadores marcados, que `integration` figure como no aplicable con `same_pr`, que se nombren `story.md` y el bug y el archivado en el mismo PR, y que ODD exprese que nunca integra en la spec viva y remite los bugs de un proyecto ISO al SDD.
  - Ficheros: `internal/assets/skills/_shared/sdd-orchestrator-sections.md`, `internal/components/agentguidance/routing.go` (D4).
  - Pruebas: pines de las plantillas del orquestador sobre la presentación del delta y de los disparadores en la compuerta `design`; pin de D4 en el bloque ODD; paridad entre runtimes.
  - Comando: `go test ./internal/assets/... ./internal/components/agentguidance/ ./internal/components/sdd/ -count=1`
  - Commit: `docs(orchestrator): disparadores en la compuerta design, integration no aplicable y D4 en ODD`
  - Dependencias: U14 y la guía T2/T3 fusionada en `main` (rebasar antes de empezar).
  - Cubre: REQ-28.28, REQ-28.29, REQ-21.9, REQ-26.1

- [ ] U16 · Regeneración de goldens y copias versionadas (pieza 13, P7) — DEPENDE DE U15
  - Objetivo: regenerar una sola vez los goldens y las copias versionadas con el método de P7, sin compilar binarios.
  - Ficheros: `testdata/golden/*.golden` (13), `.codex/AGENTS.md` (por bloques), `AGENTS.md` y `GEMINI.md` solo si cambia lo que describen, `internal/workspace/docs_contract_test.go` y `internal/components/sdd/review_ledger_contract_test.go` (pines, techos y hash de Kilocode solo si el protocolo cambia de tamaño).
  - Pasos: `go test ./internal/components/ -run Golden -update -count=1`, repaso `go test ./internal/components/ -run Golden -count=1`, copia por bloques a `.codex/AGENTS.md`, `go test ./internal/components/... ./internal/assets/... -count=1`.
  - Aceptación: goldens en verde sin `-update`; el bloque gestionado de `.codex/AGENTS.md` coincide con el golden salvo marca y tabla de modelos; ningún golden contiene texto de la parada de plan retirada; `go vet ./...` y ratchets de negativas y código muerto sin empeorar.
  - Comando: `go test ./internal/components/ -run 'Golden' -count=1`
  - Commit: `chore(golden): regenerar goldens y copias versionadas del orquestador`
  - Dependencias: U15.
  - Cubre: REQ-28.28

## Tabla de cobertura REQ → tareas

ADDED (24) y MODIFIED (14) de las 13 deltas. Ningún requisito queda sin tarea.

| REQ | Delta | Tipo | Tareas |
|---|---|---|---|
| REQ-28.1 | sdd-iso-evidence | ADDED | U3, U4 |
| REQ-28.2 | sdd-iso-evidence | ADDED | U6 |
| REQ-28.3 | sdd-iso-evidence | ADDED | U9 |
| REQ-28.4 | sdd-iso-evidence | ADDED | U11 |
| REQ-28.5 | sdd-iso-evidence | ADDED | U11 |
| REQ-28.6 | sdd-iso-evidence | ADDED | U13 |
| REQ-28.10 | sdd-artifact-profiles | ADDED | U8 |
| REQ-28.11 | sdd-artifact-profiles | ADDED | U8, U9 |
| REQ-28.12 | sdd-artifact-profiles | ADDED | U10 |
| REQ-28.13 | sdd-artifact-profiles | ADDED | U10 |
| REQ-28.14 | sdd-artifact-profiles | ADDED | U10 |
| REQ-28.15 | sdd-artifact-profiles | ADDED | U10 |
| REQ-28.16 | sdd-artifact-profiles | ADDED | U10 |
| REQ-28.17 | sdd-artifact-profiles | ADDED | U10 |
| REQ-28.20 | sdd-lane-kickoff | ADDED | U2 |
| REQ-28.21 | sdd-archive-lifecycle | ADDED | U7 |
| REQ-28.22 | sdd-archive-lifecycle | ADDED | U12, U13 |
| REQ-28.23 | spec-baseline-declaration | ADDED | U3 |
| REQ-28.24 | spec-baseline-declaration | ADDED | U4 |
| REQ-28.25 | spec-baseline-integration | ADDED | U6 |
| REQ-28.26 | axiom-sdd-cli-integration | ADDED | U8 |
| REQ-28.27 | axiom-sdd-cli-integration | ADDED | U4 |
| REQ-28.28 | sdd-orchestrator-assets | ADDED | U14, U15, U16 |
| REQ-28.29 | organic-agent-trigger-rules | ADDED | U15 |
| REQ-21.15 | sdd-role-closure-handoff | MODIFIED | U1 |
| REQ-2.1 | multi-role-fan-out | MODIFIED | U1 |
| REQ-1.2 | multi-role-fan-out | MODIFIED | U10 |
| REQ-21.9 | sdd-block-review-gates | MODIFIED | U10, U15 |
| REQ-26.1 | sdd-block-review-gates | MODIFIED | U12, U15 |
| REQ-21.16 | sdd-archive-lifecycle | MODIFIED | U12 |
| REQ-25.39 | sdd-archive-lifecycle | MODIFIED | U12 |
| REQ-27.14 | sdd-archive-lifecycle | MODIFIED | U12 |
| REQ-25.18 | change-branch-lifecycle | MODIFIED | U12 |
| REQ-25.45 | change-branch-lifecycle | MODIFIED | U12 |
| REQ-25.42 | axiom-sdd-cli-integration | MODIFIED | U4, U12 |
| REQ-24.38 | spec-baseline-integration | MODIFIED | U7 |
| REQ-24.46 | spec-baseline-integration | MODIFIED | U7 |
| REQ-24.47 | spec-baseline-integration | MODIFIED | U5 |

Comprobación: 38 requisitos en las deltas (24 ADDED + 14 MODIFIED), 38 filas en la tabla, 0 sin tarea. Tareas sin REQ: U12a (P5) y U12b (P6), huecos de spec declarados arriba.

## Previsión de líneas por unidad (orientativa; adiciones + eliminaciones, con pruebas)

| Unidad | Líneas | Unidad | Líneas |
|---|---|---|---|
| U1 | 120 | U9 | 400 |
| U2 | 150 | U10 | 700 |
| U3 | 450 | U11 | 450 |
| U4 | 600 | U12 | 700 |
| U5 | 250 | U12a | 150 |
| U6 | 550 | U12b | 150 |
| U7 | 350 | U13 | 450 |
| U8 | 350 | U14 | 500 |
| | | U15 | 200 |
| | | U16 | 60 (más goldens regenerados, no contados) |

Total ≈ 5.780 (rango 5.000 a 7.000). Una sola PR con `size:exception`.

## Riesgos de las dependencias y del reparto

- **Cuello de botella U4.** U6, U8 y U10 cuelgan de U4; un error de composición de bloqueos (D3) se arrastra. Mitigación: la prueba de arquitectura y el `evidenceEvaluator` desactivado en `TestMain` van en U4.
- **U14 puede dejar goldens rojos** hasta U16 si las skills entran en ellos; ver nota de U14. Mientras U15 espere a la cadena, la PR no puede cerrarse: U14 a U16 deben aplicarse seguidas o U15 y U16 diferirse a un seguimiento (decisión del orquestador).
- **P5 reabre un disparador del design** (`data-model`), ver «Huecos de spec».
- **Entorno Windows:** los fallos conocidos y ajenos (`filemerge` junctions, symlinks, texto del SO en castellano de `TestOpaqueRepositoryContext*`) no deben confundirse con regresiones.
- **Un solo escritor** y sin ramas paralelas: el paralelismo lógico de U1, U2, U3 y U5 no se explota.

## Progreso

- 2026-11-10: tasks escrito. Ninguna unidad aplicada.
