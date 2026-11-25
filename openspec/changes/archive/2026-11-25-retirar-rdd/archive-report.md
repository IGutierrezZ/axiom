# Reporte de Archivado Formal: retirar-rdd

## Resumen de Cierre
- Fecha de Archivado: 2026-11-25
- Estado de Verificación: verify limpio en el momento de su redacción (`verify-report.md`: 0 CRITICAL, 2 WARNING, 3 SUGGESTION, `**Estado:** limpio`). Cierre posterior documentado en la sección «Estado final» de este reporte.
- Destino en Archivo: `openspec/changes/archive/2026-11-25-retirar-rdd/`
- Modo de cierre: `continuous`, rol único `fullstack`, almacén `openspec`, monorepo-embedded.
- Estado de la PR de código: #306 fusionada en `main` con merge commit `d3e189e6` (cabeza de la rama: `5e9ae41e`). Sin revisión de juicio (Judgment Day) por decisión de la persona: solo `verify`.

## Estado final (fuente de mayor rango: petición de archivado)
Las cifras y afirmaciones de `verify-report.md` son un registro de su momento (rango inferior). Hechos posteriores:
- `5e9ae41e fix(releasepolicy)`: `policy.go` esperaba 6 archivos de release y ahora espera 5. Lo detectó la CI («Unit Tests») porque esas pruebas se saltan en local sin `AXIOM_TEST_BUILD_BINARIES=1`. Según el orquestador, la CI de la PR completa está en verde, incluida `windows-full-suite` con sus 5 shards.
- Autosquash del fixup de C5 (`4d4d910d`, restauración de las 19 pruebas de seguridad de release): los sha del Registro de `tasks.md` se remapearon tras esa reescritura.
- Tareas: 78 de 78 marcadas en `tasks.md`; 0 pendientes en el archivo.
- Los dos WARNING del verify (W1: pruebas inestables bajo carga completa, no atribuidas al diff; W2: cobertura de `ProjectForRepo` perdida con `rddpref_test.go`) no figuran como resueltos en las fuentes disponibles. Se registran como informativos y abiertos; no hay evidencia en este cierre de que se hayan corregido, salvo que la CI no reproduce W1.

## Especificaciones Históricas Superadas (ODD-4.2)
No se usó `--supersede`. Los requisitos retirados salen de la especificación viva sin marca `[SUPERSEDED]`: cada delta archivado conserva su `(Reason: ...)` y su `(Migration: ...)`. Motivo: la retirada de RDD elimina código y comandos; no hay requisito sustituto dentro de la misma capacidad, salvo los traslados indicados abajo.

Traslados de requisitos (sin pérdida de contenido):
| Dominio | Requisito previo (rdd-decoupling-v3-stability) | Estado | Nueva referencia |
|---------|-----------------------------------------------|--------|------------------|
| axiom-sdd-cli-integration | REQ-18.3 (decodificación de rutas Windows) | Trasladado como REQ-46.7 | `axiom-sdd-cli-integration/spec.md` |
| axiom-user-state-and-env | REQ-18.4 (aislamiento del CWD) | Trasladado como REQ-46.8 | `axiom-user-state-and-env/spec.md` |
| axiom-skills-index-governance | REQ-18.5 (preset de skills) | Trasladado como REQ-46.9 | `axiom-skills-index-governance/spec.md` |
| engram-protocol-injection | REQ-18.6 (proyecto ambiguo) | Trasladado como REQ-46.10 | `engram-protocol-injection/spec.md` |
| sdd-independent-review | REQ-28.11 (frontera con RDD) y requisito de Judgment Day de `review-findings-ledger` | REQ-28.11 retirado; Judgment Day trasladado como REQ-28.43 | `sdd-independent-review/spec.md` |

## Especificaciones Integradas
Composición con `axiom sdd-archive-compose` (sin `--supersede`), 35 deltas, 0 fallos. Cada composición se escribió con `.compose-tmp` y `mv`. Registros en `archive-run` de la sesión de archivado.

**Compuestas en el vivo (14):** axiom-distribution-identity, axiom-sdd-cli-integration, axiom-skills-index-governance, axiom-user-state-and-env, change-branch-lifecycle, engram-protocol-injection, local-web-dashboard, organic-agent-trigger-rules, sdd-block-review-gates, sdd-independent-review, sdd-orchestrator-assets, sdd-preflight-configuration, tui-spanish-localization, upstream-absorption-protocol.

**Retiradas (21): vaciadas por sus REMOVED y directorio borrado** (D14, tarea 12.2): las 20 `rdd-*` (rdd-authority-disposition-plan, rdd-authority-graph-classification, rdd-authority-store, rdd-backlog-disposition, rdd-candidate-identity, rdd-candidate-relation-algebra, rdd-closure-disposition-execution, rdd-decoupling-v3-stability, rdd-delivery-exception-removal, rdd-freeze-expansion-policy, rdd-leaf-disposition-execution, rdd-new-lineage-activation, rdd-ownership-inventory, rdd-post-verify-review-offer, rdd-receipt-only-gates, rdd-review-core-transitions, rdd-sdd-receipt-consumption, rdd-shadow-evaluation, rdd-simplification-design, rdd-transport-capability) y `review-findings-ledger`. Requisitos previos eliminados de esos dominios: 7+4+5+3+5+8+7+6+4+3+11+5+3+5+7+7+5+7+9+4+18 = 133 (conteo de `### Requirement` en las specs vivas antes de componer).

**Creadas:** ninguna. **Specs vivas tras el archivado:** 54 (antes 75).

Resultado del catálogo (`axiom archive sync`): 54 especificaciones, 533 requisitos activos, 2145 escenarios. `openspec/INDEX.md` regenerado con el comando nativo; no se editó a mano.

## Comprobación de ausencia
`git grep -n -E 'bench-model-picker|bench_fixture|rdd_harness|reviewtransaction|reviewerprovider|rddharness' -- openspec/specs` NO queda vacío. Resultado: 3 líneas, todas en `openspec/specs/axiom-sdd-cli-integration/spec.md` (líneas 503, 505 y 525), dentro del requisito REQ-46.1 «El Binario No Expone `axiom review*`, `axiom review stop-hook` ni `axiom bench-model-picker`». Son el texto normativo que prohíbe esos verbos, no referencias a código existente. Decisión: no se modifica la spec; el criterio de ausencia debe excluir ese requisito o reformularse en la siguiente comprobación.

## Verificación mecánica (diff -r)
Copia y movimiento con `git mv` desde el snapshot previo a la operación. Salida verbatim:

```
git mv ok
--- diff -r (snapshot vs destination):
diff_exit=0
```
Diff vacío: el contenido de `openspec/changes/archive/2026-11-25-retirar-rdd/` es idéntico al de `openspec/changes/retirar-rdd/` antes del movimiento. `archive-report.md` es aditivo y no existía en el origen.

## Artefactos Consolidados
- `proposal.md`, `design.md`, `explore.md`, `tasks.md` (78/78 marcadas), `verify-report.md`, `kickoff.yaml`
- `specs/` (35 deltas)
- `coverage-review.spec.md` y `coverage-review.design.md`
- `archive-report.md` (este reporte)

No hay `apply-progress` en el cambio.

## Revisión de cobertura (informativa, no bloquea)
Según el propio informe de cada revisión («Tratamiento en esta corrida»), el hueco está cubierto por tareas o pruebas cuando lo indica el informe. Esta fase no re-verificó cada prueba citada.

Spec (14 huecos):
- Declarados cubiertos por pruebas o tareas: 5, 6, 8, 9, 11, 14.
- Abiertos: 1 (`bench/` y ratchet no recogidos), 2 (MODIFIED de `sdd-independent-review` no entregado: el propósito conserva «Son ajenas a RDD»), 3 (notas de versión sin requisito), 4 (escenarios de restos heredados de REQ-46.4), 7 (`change-branch-lifecycle` sin escenario de almacén residual), 10 (404 y pantallas `review_mode` en dashboard), 12 (ausencia de `rdd-advisory-transport`, `axiom-bench` y agentes `review-*`), 13 (REQ-18.1 retirado sin traslado).

Diseño (9 huecos):
- Cubiertos o tratados por tareas/diseño: 1 (D16 asigna pruebas a REQ-46.7 a 46.10), 2, 4, 6, 7 y, en parte, 8 (migración manual menciona la prosa `Review Execution Contract` y `_shared/review-ledger-contract*.md`).
- Abiertos: 3 (comprobaciones obligatorias distintas de `claude-network-none`), 5 (promoción de RC antigua o en curso), 9 (shard `cli-r` sin medir frente al timeout).

## Acciones humanas pendientes
- Cortar una RC nueva (una anterior lleva el asset del contrato de proveedor).
- Migración manual antes de instalar: limpiar en `~/.claude/settings.json` las entradas `axiom review stop-hook --agent claude-code` de `hooks.Stop` y `hooks.SessionStart` (`docs/releases/notas-retirada-rdd.md`).
- Fuera de alcance que aún nombra RDD: `AI_POLICY.md`, `CONTRIBUTORS.md`, `GEMINI.md`.
- El ruleset de `main` ya no exige «Claude Network-None Runtime (Required)» (hecho según el orquestador).

## Trazabilidad de almacén
Almacén declarado por el orquestador: hybrid. Estado nativo: `openspec`. El archivo es la fuente de trabajo; no se leyeron observaciones de Engram en esta fase, por lo que no hay IDs de observación que registrar. El reporte se persiste en el archivo y en Engram con la clave `sdd/retirar-rdd/archive-report`.

## Resumen de cierre del ciclo
El cambio `retirar-rdd` queda archivado. Implementación: completa en la PR #306, con un fix posterior en `5e9ae41e`. Verificación: `verify` limpio en su momento; CI de la PR en verde según el orquestador. Tareas no completadas: ninguna en el archivo. Hallazgos abiertos: W1 y W2 del verify; huecos de cobertura listados arriba. Riesgo de cierre: el criterio de `git grep` no queda vacío por el requisito REQ-46.1.
