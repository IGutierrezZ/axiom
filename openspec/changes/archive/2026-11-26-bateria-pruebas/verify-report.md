# Informe de verificación: bateria-pruebas (C-06-c), rol `fullstack`

Alcance: diff `89d81b31b..HEAD` del worktree. Tareas T1-T10 marcadas `[x]` en `tasks.md`; `apply-progress.md` no declara pendientes.

## Cumplimiento técnico

| Comprobación | Resultado |
|---|---|
| `go build ./...` | OK |
| `go vet` (acceptancetests, workspace, sddstatus, livespec, dashboard, cli, isoevidence, hub, assets, cmd/axiom) | OK, sin salida |
| `go test -count=1` acceptancetests, isoevidence, livespec, workspace | OK |
| sddstatus `-run AcceptanceTests\|TasksGate\|Governance\|Explainer\|StatusV2\|ArchiveBlocked` | OK |
| cli `-run SDDTests\|GateArtifactDigest\|RunSDDGateRecord` | OK |
| cmd/axiom `-run RunSDDHelp\|SDDVerbRouting\|SDDTests\|RunSDDUnknown` | OK |
| dashboard `-run IncrementDetail\|AcceptanceTests\|Tabs\|Router\|Canary\|AssetsAreServed\|ExplainerTab` | OK |
| assets `-run AcceptanceTestsDoctrine\|TestOrchestratorSectionDriftDoesNotGrow\|StatusContract` | OK |
| components `-run TestGolden`; hub `-run Declares\|Init`; kickoff `-run EvaluateGates` | OK |
| `scripts/deadcode-ratchet.sh` | NO ejecutado (compila una herramienta; por norma, lo mide la CI) |

Revisiones estáticas: el status sin el bloque no cambia (campo `omitempty`, test de JSON idéntico); sin rutas `/api` nuevas; sin literales de color en el diff de dashboard; no se tocaron `internal/kickoff/machine.go`, `_shared/sdd-orchestrator-sections.md` ni la skill `sdd-apply`.

## Desvíos evaluados

- T6, «sin specs delta» como error de uso también con la función inactiva y sin `pruebas.md`: conforme. RF-KOF-23 dice literalmente «Un cambio inexistente o sin specs delta es un error de uso» sin condicionarlo.
- T8, campo informado para todo cambio SDD activo y omitido solo en archivados: conforme con RF-WEB-17 («detalle de un incremento SDD»; sin `pruebas.md` indica ausencia). El aviso solo sale con la función activa.
- T1, exige la clave `enabled`: compatible con RF-TOP-19 («`enabled` DEBE ser un booleano estricto», sin asumir valor).
- RF-GAT-32 sin tocar el orquestador: ver W1.

## Hallazgos

- CRITICAL: ninguno.
- WARNING W1: RF-GAT-32 (la parada `tasks` presenta `pruebas.md` y su resultado, y declara su ausencia) solo se cumple de forma indirecta: el status proyecta `acceptanceTests` y la skill `sdd-tasks` lo documenta, pero el texto del orquestador que presenta la parada no lo indica, y no hay test que lo fije. Aplazado por C-05-d y ya anotado como riesgo en `apply-progress.md` y en `coverage-review.design.md` (hueco 3).
- SUGGESTION S1: ejecutar `scripts/deadcode-ratchet.sh` en CI (la revisión manual de 39 funciones sin código muerto no es prueba de ejecución).
- SUGGESTION S2: documentar en el manual que `enabled` es obligatorio dentro del bloque.

**Estado:** limpio
