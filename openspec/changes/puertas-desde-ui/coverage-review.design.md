# Revisión de cobertura · design · puertas-desde-ui

Informe informativo (`skills/_shared/sdd-coverage-review.md`): no bloquea fases, puertas ni archivo, y no forma parte de los artefactos que digieren las puertas.

## Ronda 1 — 2026-11-27

Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y en paralelo. Ambos devolvieron un resultado utilizable. Entradas: `proposal.md`, las dos specs del cambio y `design.md`.

| # | Tipo | Artefacto | Ubicación | Hueco | Revisor |
|---|---|---|---|---|---|
| 1 | uncovered_requirement | design | D10, D1 | WEB:4 pide rechazar «aprobar `integration` sin evidencia de merge válida» con la negativa de la CLI, pero la CLI admite aprobar `integration` sin evidencia (`internal/kickoff/args.go`) y el servidor no añade reglas. Una petición POST sin `evidence` se registraría. Hay que decidir y probar el escenario. | A, B |
| 2 | edge_case | design | D4, D8 | `ExpectedDigest` es opcional en el núcleo, y el servicio no define qué responde si la petición omite `digest`. Sin él se salta la comprobación de digest. | A |
| 3 | error | design | Interfaces (cuerpo POST) | El cuerpo HTTP trae `evidence` como objeto con `ref`, pero `GateRecordInput` solo tiene `EvidenceKind`, `Commit`, `BaseRef` y `Evidence`. Falta el mapeo. | A |
| 4 | edge_case | design | D4 | Negar con `gate_already_resolved` cuando el último registro es un rechazo del mismo digest impide volver a decidir la misma versión, cosa que la CLI sí permite. Contradice GAT:1 («misma lógica») y el escenario «Aprobación válida». | B |
| 5 | uncovered_requirement | design | D10 | WEB:5 «Valor por defecto»: aprobar `tasks` sin elegir modo registra `notify`, pero el servidor registra sin campo. Solo la UI aplica el valor por defecto. | B |
| 6 | edge_case | design | D8 | El reintento exige que el último registro sea una aprobación. Si después se registra otra decisión, queda bloqueado (`gate_approval_missing`) aunque `publication.state` siga en `pending`. | B |
| 7 | edge_case | design | D4, D8 | El cerrojo del dashboard solo funciona dentro del proceso. Queda una ventana residual frente a la CLI u otro dashboard que no está analizada para un rechazo de otro canal con digest distinto. | A |
| 8 | edge_case | design | D3 frente a la propuesta | La propuesta prevé desplazar la línea base del trinquete de rechazos y el diseño dice que no hará falta. Falta un criterio de comprobación explícito, también para el rechazo nuevo `gate_digest_changed`. | A |
| 9 | edge_case | design | D7 | En `integration` sin worktree de archivado, la guarda niega de forma previsible. La parada no avisa de `axiom branch ensure --archive` ni define el digest esperado si no hay artefacto. | B |
| 10 | edge_case | spec | WEB:1 | El diseño añade razones de «sin parada» (`continuous`, `not_sealed`, `blocked`, `archived`, `odd`) y los códigos 404/409/503, pero la spec no tiene escenarios para ellos. | A |
| 11 | edge_case | spec | WEB:7 | Faltan escenarios de publicación idempotente (`nothing_to_publish`), `code_unpublished`/`unavailable`, reconstrucción tras recarga y ámbito (`checkpointed`, no `integration`). D8 y D9 los resuelven. | A, B |
| 12 | edge_case | spec | WEB:2 | Ni la spec ni sus escenarios cubren la identidad del actor (no se envía, `defaultActor`, aviso `actor_identity`). | A |
| 13 | use_case | spec | WEB:4 | Ningún requisito hace verificable lo que se ve tras decidir: cesión (`handedOffTo`), último rol, roles no bloqueantes y avisos. D10 lo define. | B |
| 14 | error | spec | WEB:2 | No hay escenarios para `gate_not_pending`, cambio inexistente/archivado/ODD, pasarela ausente (503), negativas de guarda/roster/cesión ni repetición tras un registro correcto. D8 y D11 los resuelven. | B |

Sin corrección automática: en `continuous` los huecos pasan al resumen final y, como notas, a `sdd-apply`.
