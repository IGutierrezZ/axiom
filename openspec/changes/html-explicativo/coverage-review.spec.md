# Revisión de cobertura — spec de `html-explicativo`

Revisión informativa: no bloquea ninguna fase, puerta ni archivo, y no modifica la spec. Revisores ciegos: `jd-judge-a` (A) y `jd-judge-b` (B). El informe es la unión deduplicada de sus huecos.

## Ronda 1 — 2026-11-26

Entradas: `proposal.md` y los cinco `specs/*/spec.md` del cambio, contrastados con `docs/plan/backlog-2026-11/06-ui-human-loop.md` (C-06-b). Los dos revisores devolvieron un resultado utilizable.

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | requisito sin cubrir | `sdd-block-review-gates` (sin requisito) | Ningún escenario fija que `axiom sdd publish` incluya `resumen-spec.html` y `resumen-design.html` con el resto de documentos del cambio (paradas aprobadas en checkpointed; tras spec y design con `docs_publication: auto`). | A, B |
| 2 | error | `sdd-block-review-gates` prov:GAT:1 | No se define qué ocurre si la fase con `yes` no logra escribir el HTML o lo escribe incumpliendo la autocontención: si termina con el md, si avisa y que el estado queda `missing` o `stale` sin bloquear. | A, B |
| 3 | caso borde | `local-web-dashboard` prov:WEB:1 | El endpoint no cubre un `resumen-*.html` que sea enlace simbólico fuera de la carpeta, un directorio o un fichero ilegible; el método `HEAD`; los cambios archivados; ni si se sirve con `explainer_html` ausente o `no` cuando el fichero existe. | A, B |
| 4 | caso borde | `sdd-lane-kickoff` prov:KOF:2 | La compatibilidad hacia atrás (un binario anterior ignora el campo) no tiene escenario, ni la relectura de un `kickoff.yaml` con `explainer_html: yes`. | A |
| 5 | caso de uso | `sdd-lane-kickoff` prov:KOF:3 | No hay escenario de que un llamante existente de `seal` sin orquestador (el dashboard) siga funcionando con `ask` sin bandera y reciba el aviso. | A |
| 6 | requisito sin cubrir | `sdd-block-review-gates` prov:GAT:2 | La regeneración tras rechazo no cubre el caso en que el HTML aún no existía, ni el md editado a mano sin rechazo, en el que el aviso de desactualizado no tiene salida accionable. | A |
| 7 | requisito sin cubrir | `sdd-block-review-gates` prov:GAT:2 | No se exige que, en las paradas `spec` y `design` de un cambio checkpointed con `yes`, el orquestador presente el HTML y su frescura junto al md. | B |
| 8 | caso borde | `local-web-dashboard` prov:WEB:2 | No se define qué muestra la pestaña cuando la frescura no se proyecta porque el md de origen aún no existe (p. ej. antes de design). | B |
| 9 | caso borde | `sdd-lane-kickoff` prov:KOF:2 | No hay camino para activar el HTML en un cambio ya sellado sin el campo o con `no`, ni se define el comportamiento de `seal` sobre un cambio ya sellado con otro `--explainer-html`. | B |
| 10 | caso borde | `sdd-block-review-gates` prov:GAT:4 | No se fija la frescura con varios `spec.md` por dominio ni al añadir o quitar un dominio, ni la tolerancia del parseo del meta (orden de atributos, comillas, mayúsculas, varias etiquetas). | B |
| 11 | requisito sin cubrir | `sdd-preflight-configuration` prov:PRE:1 | Ningún escenario exige una comprobación automática de que la doctrina del orquestador contiene el marcador `Axiom SDD kickoff explainer:`, su orden (tras `ensure`, antes de `seal`) y la restricción a `ask`. | B |

En `continuous` estos huecos no relanzan `sdd-spec`: pasan al resumen final, y el diseño y las tareas pueden absorberlos.
