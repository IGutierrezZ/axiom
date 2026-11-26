# Informe de verificación: migrar-ids-rf (C-03-e)

> **Rol:** `fullstack` (único). **Alcance:** `git diff f596a985..HEAD` (rama `inc/migrar-ids-rf`, 12 commits, 648 ficheros). **Modo:** solo lectura; sin compilar binarios.

## Estado de las tareas (observado)

Las fases 1 a 8 de `tasks.md` figuran todas `[x]` (sin tareas pendientes). Tarea 8.1: la reproducibilidad de d3 desde d2 no se ejecutó, por decisión del orquestador (anotado en la evidencia de la tarea); se sustituye por la idempotencia de `plan` en HEAD, comprobada abajo.

## Comandos ejecutados

| Comando | Resultado |
|---|---|
| `go vet ./...` | exit 0, sin salida |
| `go test -count=1` idmigration, legacyid, reqtrace, openspecdoc, livespec, reqid, isoevidence, techcontext, kickoff, sddstatus, assets, components, livingdoc | todos `ok` |
| `go test -count=1 ./internal/cli -run 'Trace\|Archived\|SDDKickoffSeal\|SDDGateRecord'` | `ok` (150 s) |
| `go test ./internal/reqtrace -run TestTraceRealMapResolvesLegacyID -v` | PASS |
| `AXIOM_IDMIGRATION=plan go test ./internal/idmigration -run TestMigrateRepository -count=1 -v` (HEAD) | PASS: 1977 ficheros, 0 reescritos, 0 sustituciones, 0 ambiguos, 0 fallos, 0 candidatos a residuo |
| `gofmt -l` de los paquetes tocados | sin salida |
| `scripts/deadcode-ratchet.sh` | NO EJECUTADO: requiere `go install` de un ejecutable (EDR Cortex). No verificado |

## Escenarios de las deltas contra código y pruebas

- **RF-TRC-02** (cinco escenarios: no cumple patrón, patrón del proyecto, antiguo del mapa aceptado, antiguo fuera del mapa rechazado, provisional): `ParseID(raw, p, legacy)` en `internal/reqtrace` y cableado en `internal/cli/trace.go`; cubiertos por `internal/reqtrace/legacy_test.go` y `internal/cli/trace_legacy_test.go`. CUMPLE.
- **RF-TRC-04** (alias inyectado, solo trazas, ambiguo `REQ-1.1` a siete, `REQ-24.3` único, acotación por capacidad, ID nuevo trae incrementos del antiguo): `LegacyResolver`, coincidencia acotada y `WarnLegacyAmbiguous`; pruebas de `legacy_test.go` y `TestTraceRealMapResolvesLegacyID` sobre el mapa real (`REQ-24.3` resuelve a `RF-PLH-03`, `REQ-1.1` a siete con `legacy_ambiguous`, `REQ-27.2b` aceptado). CUMPLE.
- **prov:TRC:1** (cobertura 513, regeneración idéntica, ambiguo por capacidad, invariantes): `internal/legacyid`, `TestGenerate*`, `TestIDMapMatchesLivingSpecs`, `TestIDMapFingerprint`. CUMPLE. Nota: la inmutabilidad se protege con una huella sha256 global (mecanismo barato declarado en 6.2).
- **prov:TRC:2** (sin mapa funciona, esquema desconocido, mapa corrupto, sin git): `TestAxiomTraceInvalidMapFailsTypedWithoutGit` y pruebas de `legacyid`. CUMPLE.
- **prov:RID:1** (patrón declarado, títulos sin `REQ-n.m`, cita no declarada rompe, entrada caduca rompe, residuos declarados aceptados): `TestAxiomYAMLDeclaresRFPattern`, `TestRFCorpusTitlesHaveIDs`, `TestRepositoryLegacyIDResidues`, `TestResidue*`, `TestCubreLinesUseExistingRFIDs`. CUMPLE. `axiom.yaml` declara `sdd.id_pattern: "RF-<DOM>-<NN>"` (línea 22).

## Criterio de terminado del plan

- `REQ-n.m` fuera de archivo: cubierto por residuos (la prueba permanente pasa; 0 en `internal/assets/skills`; 0 en literales de producción, vetados por la prueba). OBSERVADO.
- `axiom trace REQ-24.3` a `RF-PLH-03`: comprobado con la prueba sobre el mapa real (no se ejecutó el binario). OBSERVADO por prueba.
- `INDEX.md` regenerado: 513 requisitos, sincronizado 2026-11-26. OK.
- Idempotencia de `plan` en HEAD: 0 sustituciones. OK.
- Historia intacta: el diff no toca `openspec/changes/archive/`; `openspec/id-map.yaml` solo se añade (1542 líneas).

## Cumplimiento técnico

Fuentes consultadas: skill `go-testing` (declarada en el encargo), `openspec/config.yaml` (`rules.verify`: pruebas de paquetes tocados, `go vet`, comparación con escenarios), `design.md`. Contexto técnico mantenido: no aplicable (sin `technicalContext` proyectado en este encargo).

- Build (`go vet ./...`): pasa.
- Pruebas: pasan (ver tabla). Pruebas de tablas y `testing/fstest` conformes con `go-testing`.
- Estilo `gofmt`: limpio.
- Conformidad con design: la herramienta es solo código de prueba (sin ficheros de producción en `internal/idmigration`); `legacyid` y `reqtrace` sin `os/exec` ni `gitcommon` (guardas `imports_test.go`); el mensaje de producción de `techcontext/plan.go` ya no cita el requisito retirado (D13).
- Trinquete de código muerto: no verificado (ver arriba). Riesgo bajo: no hay producción nueva fuera de `legacyid`/`reqtrace`/`cli/trace.go`, alcanzados desde `RunTrace`.

## Hallazgos

- CRITICAL: ninguno.
- WARNING: ninguno.
- SUGGESTION · Observación sin regla asociada · mensajes de commit `c6d2ded8` («...patron RF») y `38f336a1` («...patron RF de Axiom») sin tilde en «patrón». Los artefactos y mensajes van en castellano; se corregiría solo si se reescribe el historial local antes de publicar.
- SUGGESTION · la huella de inmutabilidad es global (6.2); refinar a huella por conjunto de entradas si se añaden entradas legítimas.

## Limitaciones

No se ejecutaron: `scripts/deadcode-ratchet.sh`, la reproducción de d3 desde d2, ni `./internal/cli` completo ni `go test ./...` (por política; los corre la CI).

**Estado:** limpio
