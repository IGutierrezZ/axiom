# Revisión de cobertura: spec y diseño de `retirar-rdd`

Es informativa: no bloquea fases ni el archivo, y no corrige el diseño. Con el estilo `continuous`, los huecos pasan al resumen final.

## Ronda 1 — 2026-11-25

Entradas: `proposal.md`, los 35 deltas de `specs/` y `design.md`. Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y de solo lectura. Los dos dieron un resultado utilizable.

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | requisito sin cubrir | `design.md#Estrategia de pruebas` / C8 | REQ-46.7 a 46.10, trasladados desde `rdd-decoupling-v3-stability`, no tienen prueba trazable en el diseño. Podrían perder cobertura sin aviso si vivían en tests mixtos que se recortan. | A, B |
| 2 | requisito sin cubrir | `design.md#C9` | C9 borra `cli/dual_contract_integration_test.go` entero, pero `TestDualContract_SDDStatusDualSerialization` y `TestDualContract_SchemaValidationWithCompiledSchemas` no son RDD: el primero es la prueba candidata de REQ-46.7 y el segundo valida esquemas que se quedan. | A |
| 3 | caso límite | `design.md#Acciones humanas` / D8 | Solo se trata `claude-network-none` como comprobación obligatoria. Desaparecen o cambian de nombre otras comprobaciones: los pasos de bench, `opencode-v2-contracts` y los shards `cli-review-*` y `reviewtransaction-*`. Falta inventariar las obligatorias. | A, B |
| 4 | caso límite | `design.md#D4` | `skills.Installable` solo se aplica en `cli.selectedSkillIDs`. Faltan los otros lectores de la selección persistida (TUI/`skill_picker`, `doctor`, `install`, `--skills` explícito sin aviso) y el escenario de que `sync` no vuelva a persistir el id. | A, B |
| 5 | caso límite | `design.md#C5` / Candidatas de release | Ante una RC ya cortada solo dice «hace falta una RC nueva». No trata la promoción de una RC antigua con el workflow nuevo, una promoción en curso en el momento del merge ni la respuesta de las compuertas a un asset `review-provider-contract` de más. | A, B |
| 6 | caso límite | `design.md#Orden de commits` (C4 a C9) | No comprueba que los tests e2e conservados no invoquen `review` mientras el binario aún lo incluye. | A |
| 7 | caso límite | `design.md#C7` | C7 no lista `internal/dashboard/secrets_canary_test.go:45`, que declara la ruta `/api/project/review-mode`, ni comprueba que ningún HTML o JS cargue `reviewmode.js`. | B |
| 8 | caso límite | `design.md#D6` | Borrar los recortadores (`removeBoundedReviewSection`, etc.) deja sin limpiar las instalaciones anteriores a C-01-c, con prosa `Review Execution Contract` y `_shared/review-ledger-contract*.md`. La migración manual tampoco las menciona. | B |
| 9 | caso límite | `design.md#D8` | El shard único `cli-r` (`^TestR`) se declara sin medir. No hay criterio de aceptación ni plan alternativo si supera el timeout en Windows. | B |

Comprobación del orquestador sobre el hueco 3: el ruleset activo de `main` (id 23886794) solo exige «Go Format», «Unit Tests» y «Claude Network-None Runtime (Required)». Los shards de Windows, bench y `opencode-v2-contracts` no son obligatorios. El caso de `claude-network-none` está consultado con la sesión coordinadora.

### Tratamiento en esta corrida

Con `continuous` no se relanza `sdd-design`. Las tareas recogen como comprobaciones los huecos que son verificables en código: 1, 2, 4, 6 y 7, y la mención de los restos de prosa y de `_shared/review-ledger-contract*.md` en la migración manual (8). El resto pasa al resumen final.
