# Revisión de cobertura · spec · puertas-desde-ui

Informe informativo (`skills/_shared/sdd-coverage-review.md`): no bloquea fases, puertas ni archivo, y no forma parte de los artefactos que digieren las puertas.

## Ronda 1 — 2026-11-27

Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y en paralelo. Ambos devolvieron un resultado utilizable. Entradas: `proposal.md`, `specs/local-web-dashboard/spec.md` y `specs/sdd-block-review-gates/spec.md`.

| # | Tipo | Ubicación | Hueco | Revisor |
|---|---|---|---|---|
| 1 | use_case | WEB:4 | La salida 2 no exige mostrar la cesión (`handedOffTo`) ni que la sesión no continúa; tampoco el aviso de último rol bloqueante cerrado (`lastRoleClosed`, `pendingNonBlocking`) tras aprobar `role-apply`. | A, B |
| 2 | use_case | WEB:4 | La evidencia de `integration` no fija qué campos introduce la persona (`kind`, `ref`, `base`), cómo se muestra una evidencia no verificada ni cómo se presenta la negativa tipada. | B |
| 3 | edge_case | WEB:2, GAT:1 | `integration` se registra con contexto `archive`: falta escenario de guarda de rama o worktree de archivado incorrecto desde la UI, y qué raíz usa el dashboard. | A, B |
| 4 | error | WEB:2, WEB:6 | Si el registro encadenado falla (409, `busy`, 422, guarda de rama) no se dice que no se publique ni que la UI muestre el error sin pasar a «aprobada sin publicar»; tampoco qué hace la UI tras un 409, una carrera con otro canal o una recarga (volver a consultar la parada). | A, B |
| 5 | edge_case | WEB:2 | Puerta pedida distinta de la proyectada, o `role-apply:<rol>` fuera del roster sellado: falta escenario equivalente al rechazo de la CLI (`gateKeyInRoster`). | B |
| 6 | edge_case | WEB:2, WEB:5 | Peticiones manuales con combinaciones que la UI no ofrece (hand-off sin salida 2, hand-off con rechazo, `commit-review` fuera de `tasks` o con rechazo, `integration` con hand-off) deben rechazarse como la CLI (`requireHandOffStops`, `requireCommitReviewStops`). | A, B |
| 7 | error | WEB:3, WEB:7 | Publicación sin aprobación registrada, con `publication.state` distinto de `pending` (ya publicada, doble clic), o con `code_unpublished` (solo mostrar `pushCommand`) o `unavailable`: comportamiento del servidor no definido. | A, B |
| 8 | error | WEB:7 | Modos de fallo de publicación distintos de `code_commits_unpublished` (credenciales, remoto que rechaza, sin upstream, nada que confirmar) y fallo tras la cesión de la salida 2. | A, B |
| 9 | use_case | WEB:1, WEB:7 | Tras aprobar, el status deja de proyectar `await-gate`: la consulta responde «sin parada» y no dice de dónde sale el estado «aprobada sin publicar» ni dónde vive el reintento. | A |
| 10 | edge_case | WEB:1 | Cambio inexistente, archivado, sin sellar, ODD o con status ilegible o `blocked`; identificación del cambio en la ruta; proyección HTTP de las negativas del núcleo. | A, B |
| 11 | edge_case | GAT:1 | Identidad del actor escrita desde la UI y aviso de identidad incompleta, frente a «registros indistinguibles». | B |
| 12 | uncovered_requirement | GAT:1 | Los avisos que hoy van a stderr pasan a un campo del resultado, pero ningún requisito exige mostrarlos; tampoco qué pasa si el despacho del evento de integración falla (sin deshacer ni duplicar). | A, B |
| 13 | edge_case | WEB:4 | Falta el escenario de Iterar en sí: no publica, no emite evento de aprobación y la parada se vuelve a presentar; tampoco el rechazo de `integration` con motivo. | A |
| 14 | edge_case | WEB:3 | En multirepo la publicación va a la rama de integración del repositorio de specs; la spec habla solo de «la rama del cambio». Sin remoto `origin` o sin rama disponible, sin escenario. | A |
| 15 | edge_case | WEB:3 | Preflight, 422 sin `confirm` y cerrojo de la ruta de publicación solo por referencia; no se dice si registro y publicación comparten cerrojo. | A |
| 16 | use_case | WEB:4 | La parada no exige mostrar los resultados de revisión independiente ni el resumen de marcas que el status ya proyecta. | B |

Sin corrección automática: en `continuous` los huecos pasan al resumen final y, como nota, a `sdd-design` y `sdd-apply`.
