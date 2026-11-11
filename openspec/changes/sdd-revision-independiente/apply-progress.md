# Progreso de apply: sdd-revision-independiente

> Rol único `fullstack` · rama `inc/sdd-revision-independiente` · modo estándar (`strict_tdd: false`) · entrega `exception-ok` (una sola PR con `size:exception` aceptado).

## Estado acumulado

| Unidad | Estado | Commit |
|--------|--------|--------|
| (planificación) | hecho | `32faf4b3` docs(sdd): planificar sdd-revision-independiente |
| U1 — regla de sellado en Go | hecho (1.1–1.10) | `99d439d2` feat(kickoff): exigir el rol fullstack al sellar continuous |
| U2 — verify por rol y cumplimiento técnico | hecho (2.1–2.6) | `65047e2f` feat(sdd-verify): añadir cumplimiento técnico y estado limpio al verify por rol |
| U3 — juicio por rol y revisión de cobertura | hecho (3.1–3.7), commit pendiente de registrar | ver registro |
| U4 — sección compartida y proyección | pendiente | — |
| U5 — pins finales, documentación y verificación global | pendiente | — |

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
| Comando focalizado | `go test ./internal/assets/ ./internal/components/sdd/` (ver registro final). |
| Harness de ejecución | N/A: solo prosa y pins. |
| Límite de reversión | `internal/assets/skills/judgment-day/**`, `_shared/sdd-coverage-review.md`, reglas de `sdd-spec`/`sdd-design` y el tramo del pin. |

## Desviaciones del diseño

- U2 incluye la edición completa de `openspec-convention.md` (también las filas de `coverage-review` y `review-ledger` que el plan asignaba a 3.5), porque el fichero se editó de una sola vez; 3.5 queda cubierta por el commit de U2.
- La prueba focalizada de U1 en `internal/cli` se acotó (`Kickoff|HandOff|SealBranch|GuardProject`): el paquete completo no termina en un tiempo razonable en esta máquina (ver riesgos).

## Incidencias

- `go test ./internal/cli/...` completo tarda más de 25 minutos en esta máquina (2034 pruebas de primer nivel, p. ej. `TestShippedReviewValidateAllGatesIgnoreAuthorityStateWithoutMutation` tarda más de 24 minutos por sí sola). No hay fallos observados en las pruebas del área tocada; se anota como fallo ambiental de duración, no de comportamiento.
