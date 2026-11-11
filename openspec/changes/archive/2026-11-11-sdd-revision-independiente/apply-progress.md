# Progreso de apply: sdd-revision-independiente

> Rol único `fullstack` · rama `inc/sdd-revision-independiente` · modo estándar (`strict_tdd: false`) · entrega `exception-ok` (una sola PR con `size:exception` aceptado).

## Estado acumulado

| Unidad | Estado | Commit |
|--------|--------|--------|
| (planificación) | hecho | `32faf4b3` docs(sdd): planificar sdd-revision-independiente |
| U1 — regla de sellado en Go | hecho (1.1–1.10) | `99d439d2` feat(kickoff): exigir el rol fullstack al sellar continuous |
| U2 — verify por rol y cumplimiento técnico | hecho (2.1–2.6) | `65047e2f` feat(sdd-verify): añadir cumplimiento técnico y estado limpio al verify por rol |
| U3 — juicio por rol y revisión de cobertura | hecho (3.1–3.7) | `74c12aa8` feat(judgment-day): admitir el diff de un rol SDD y la revisión de cobertura |
| U4 — sección compartida y proyección | hecho (4.1–4.9) | `ad2b2ae0` feat(sdd): proyectar la sección SDD Independent Review a los orquestadores |
| U5 — pins finales, documentación y verificación global | hecho (5.1–5.4) | commit `docs(sdd): documentar la revisión independiente y fijar su frontera` (ver `git log`) |

## Work Unit Evidence

### U1

| Evidencia | Valor |
|-----------|-------|
| Comando focalizado | `go test ./internal/kickoff/...`: ok (incluye `rdd_boundary_test.go`). `go test ./internal/cli/ -run 'Kickoff|HandOff|SealBranch|GuardProject' -v`: ok, 44 pruebas de primer nivel pasan. `go test ./internal/branch/...`: ok. |
| RED observado | `go test ./internal/kickoff/ -run 'ValidateForSeal|ParseKickoff'` no compilaba (`SealedByInferred`, `ValidateForSeal`, `ErrContinuousRequiresFullstack` sin definir); `-run ContinuousRoster` falló en 4 casos antes de `args.go`; la ayuda falló antes de `sdd_kickoff.go`. |
| Harness de ejecución | Pruebas en proceso de `RunSDDKickoff seal` (sin binarios sueltos). |
| Límite de reversión | `internal/kickoff/{types,schema,args,infer}.go` y sus pruebas, `internal/cli/sdd_kickoff.go`, `sdd_kickoff_test.go`, `sdd_kickoff_branch_test.go`, `sdd_gate_handoff_test.go`. |

### U2

| Evidencia | Valor |
|-----------|-------|
| Comando focalizado | `go test ./internal/assets/ ./internal/components/sdd/`: ok / ok. |
| RED observado | `go test ./internal/assets/ -run IndependentReview`: pánico por `technical-compliance.md` inexistente antes de crearlo. |
| Harness de ejecución | N/A: solo prosa de skills y su pin de texto; sin superficie de ejecución. |
| Límite de reversión | `internal/assets/skills/sdd-verify/**`, `openspec-convention.md` y el tramo verify de `independent_review_boundary_test.go`. |

### U3

| Evidencia | Valor |
|-----------|-------|
| Comando focalizado | `go test ./internal/assets/ ./internal/components/sdd/`: ok / ok (158 s). `-run IndependentReview` incluye el tramo del juicio y la cobertura. |
| Harness de ejecución | N/A: solo prosa y pins. |
| Límite de reversión | `internal/assets/skills/judgment-day/**`, `_shared/sdd-coverage-review.md`, reglas de `sdd-spec`/`sdd-design` y el tramo del pin. |

## Desviaciones del diseño

- U2 incluye la edición completa de `openspec-convention.md` (también las filas de `coverage-review` y `review-ledger` que el plan asignaba a 3.5), porque el fichero se editó de una sola vez; 3.5 queda cubierta por el commit de U2.
- La prueba focalizada de U1 en `internal/cli` se acotó (`Kickoff|HandOff|SealBranch|GuardProject`): el paquete completo no termina en un tiempo razonable en esta máquina (ver riesgos).

## Incidencias

- `go test ./internal/cli/...` completo tarda más de 25 minutos en esta máquina (2034 pruebas de primer nivel, p. ej. `TestShippedReviewValidateAllGatesIgnoreAuthorityStateWithoutMutation` tarda más de 24 minutos por sí sola). No hay fallos observados en las pruebas del área tocada; se anota como fallo ambiental de duración, no de comportamiento.

### U4

| Evidencia | Valor |
|-----------|-------|
| Comando focalizado | `go test ./internal/assets/ ./internal/components/ ./internal/components/sdd/ -count=1`: ok / ok / ok. `go test ./internal/cli/ -run TestSDDPreflightHook -count=1`: ok (incluye el caso nuevo de la interacción de preferencia). `go test ./internal/sddstatus/ -run ReviewOfferAbsence`, `./internal/multirole/`: ok (sin tocar su código). |
| RED observado | El pin de paridad y de la sección compartida se escribió antes de la proyección; la regeneración de goldens (`go test ./internal/components/ -run TestGolden -update`, 13 ficheros) se repitió sin `-update` en verde. `TestKilocodeReviewSettingsMatchCurrentMainBaseline` falló al proyectar la sección (el hash de Kilocode cambia) y se actualizó con comentario «Deliberate, not drift», como manda la convención del propio test. |
| Harness de ejecución | Los goldens regenerados y revisados por diff son la prueba de proyección en los doce runtimes (solo aparece la sección nueva y las ediciones de 4.3). |
| Límite de reversión | Sección compartida, doce plantillas, `claude/sdd-orchestrator-workflow.md`, registros, caso del hook, hash de Kilocode y `testdata/golden/*.golden`. |

### U5

| Evidencia | Valor |
|-----------|-------|
| Comando focalizado | `go test ./internal/assets/ -run IndependentReview -count=1 -v`: 6 pruebas PASS (verify, juicio y cobertura, orquestador, paridad de plantillas, mutación en memoria, no-barrido de ficheros completos). |
| RED por mutación | `TestIndependentReviewBoundaryPinDetectsMutations` retira en memoria cada frase fijada y comprueba que el contenido mutado ya no la contiene, e inyecta `receipt`, `lineage`, `axiom review`, `burn`, `RDD`, `sdd-attempt` y `acknowledged` en un pasaje limpio: el pin los detecta en todos los casos; con el contenido real pasa. Ningún fichero se modificó. |
| Documentación | `docs/manual-de-inicio.md` (commit de U4): regla `continuous` solo con `fullstack`, rechazo del `seal` repetido, preferencia, verify por rol, juicio y cobertura. `axiom.yaml` no declara `technical_context`. |
| Límite de reversión | Tramo final de `independent_review_boundary_test.go`, `docs/manual-de-inicio.md`, `tasks.md`, `apply-progress.md`. |

## Verificación global (5.3)

- `go vet ./...`: sin salida (limpio).
- `go test` de todos los paquetes salvo cli (60 min de tope; en la práctica la lista incluyó `internal/cli`): 91 paquetes ok, entre ellos `internal/assets`, `internal/kickoff`, `internal/components/sdd`, `internal/sddstatus`, `internal/multirole`, `internal/branch/engine`, `cmd/axiom`, `e2e/organicruntime`.
- Fallos ambientales, idénticos en `main` salvo el último: `internal/app` `TestTuiInstallOnThenSyncPreservesAndRefreshesOpenCodeActivation` (runtime de OpenCode no disponible; reproducido en `C:\repos\axiom` sobre `main`), `internal/backup` `TestSnapshotterSkipsUnixSockets` (sockets Unix en Windows; reproducido en `main`), `internal/components/filemerge` `TestWriteFileAtomicFollowsDirectoryJunction` y `...RelativeDirectoryJunction` (uniones de directorio en Windows; el paquete no tiene diff frente a `main`).
- `internal/cli` y `internal/reviewtransaction` no terminan: se cortaron a los 60 min (`Test killed: ran too long`) sin ningún `--- FAIL`; las pruebas lentas son las de revisión nativa ajenas a este cambio. En `internal/cli` se ejecutaron acotadas: `-run 'Kickoff|HandOff|SealBranch|GuardProject' -v` ok (44 pruebas de primer nivel) y `-run TestSDDPreflightHook` ok.
- `TestKilocodeReviewSettingsMatchCurrentMainBaseline` se actualizó deliberadamente (U4).
- No se compiló ningún binario suelto.

## Comparación con las specs (REQ-28.1 a REQ-28.14, REQ-21.4)

Sin discrepancias de comportamiento observadas. Notas: REQ-28.1 a REQ-28.3, REQ-28.10 y REQ-28.14 son prosa del orquestador, comprobada por pins de texto y goldens, no por ejecución de un runtime; la política de `fullstack:deferred` no se comprueba (DD-02) y tiene prueba; los escenarios de `axiom sdd status` sobre `continuous` con roles preexistentes se apoyan en la lectura tolerante (`ParseKickoff`), cubierta por prueba. `.codex/AGENTS.md` sin cambios.
