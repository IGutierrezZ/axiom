# Revisión de cobertura del design — eventos-ado

## Ronda 1 — 2026-11-26

Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), a ciegas y en paralelo. Entradas: `proposal.md`, las 5 deltas de `specs/` y `design.md` (D1–D18), tras la corrección acotada de spec y design por los huecos 5, 7 y 8 de la revisión de la spec. Unión deduplicada: 16 huecos. Es informativa: no bloquea fases ni compuertas.

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | uncovered_requirement | D1 · puerto `TaskTracker` | `TransitionRequest` no lleva identidad ni asignado. No queda fijado cómo llega la asignación al `PATCH` ni dónde se exponen sus avisos (`prov:INT:1` (e) y `prov:ADO:6`). | B |
| 2 | edge_case | D14 · precondiciones | No se dice qué compuerta o artefacto exige `tasks bind`, que es el verbo de rescate. | B |
| 3 | edge_case | Doctrina U12 y D9 | En `continuous` es `time record` quien pasa la Task a `task_state_close`. Con una propuesta de 0 h, con un fallo de `time propose` o con `time_record_recovered`, la Task se queda sin la transición de cierre. | A, B |
| 4 | uncovered_requirement | D11/D17 · pruebas del adaptador | `RecordHours` con `CompletedWork` ausente en el proceso (`FieldMissingError` sin escritura) y el campo ausente en el valor (se lee como 0): ninguno tiene prueba. | B |
| 5 | uncovered_requirement | Catálogo de avisos · D18 | `estimate_not_recorded` aparece en la lista sin decisión ni JSON que lo emita en el `bind` normal. No se define cómo se muestra un vínculo sin estimación en `time propose` ni en el resumen final. | A, B |
| 6 | uncovered_requirement | Pruebas CLI · D6 | Faltan las pruebas de `prov:BRA:1` para un estado distinto de `ready`, `ensure` sin `--role` y el dashboard, que no registra el inicio. | A, B |
| 7 | uncovered_requirement | Pruebas CLI · U7 | `tasks create` sin pruebas de `partial`, de un rol que falla sin impedir los demás, ni de la aprobación conservada. El dominio de `--estimate` (decimales, tope, cero, negativo) no se valida como error de uso antes de la red, y podría quedar una Task creada sin vínculo. | A, B |
| 8 | edge_case | D9 frente a `prov:INT:7` | `time propose` es de solo lectura según `prov:INT:7`, pero D9 lo hace reconciliar y escribir `integrations.yaml`. La tensión queda sin decidir. | B |
| 9 | uncovered_requirement | D16 · pruebas | El escenario de RF-ADO-02 «Valores por defecto» (cierre `Tech Review`) se reparte entre `workspace` y el adaptador sin una prueba que lo nombre. | B |
| 10 | uncovered_requirement | Doctrina U12 · `sdd-archive` | No se fija cómo se conservan entre fases de `continuous` los avisos y rechazos que `prov:ARC:2` exige listar con su comando de resolución, ni cómo se comprueba la composición de la sección. | B |
| 11 | edge_case | D13 | `Deliver` de `archive` sin vínculo `kind: work_item` en la instantánea. `role.start` sobre una Task ya `Active` pero sin asignar. | B |
| 12 | uncovered_requirement | D13 · `prov:ADO:7` | La transición del WI al archivar no tiene decisión propia: estados del tipo del WI (User Story o Bug), WI en categoría `Completed`/`Removed`, error con los estados válidos. Las pruebas no cubren el estado destino ya alcanzado ni el estado inexistente. | A |
| 13 | edge_case | D9 | Un fallo de transporte sin respuesta, en el intento y en el reintento, borra la intención. Si el `PATCH` se aplicó, se pierde la recuperación. | A |
| 14 | edge_case | D18 | `tasks bind --replace` con una intención pendiente del rol: la reconciliación relee la Task nueva, no el `item_id` de la intención. | A |
| 15 | edge_case | D12 · renderizador | El design agrupa por operación (ADDED, MODIFIED, RENAMED, REMOVED), pero `prov:INT:5` exige el orden del fichero y un `h3` con `p` por requisito. | A |
| 16 | uncovered_requirement | Pruebas de doctrina · `prov:ORQ:5` | Ninguna prueba garantiza que la doctrina no nombre secretos ni rutas del tracker. | A |

Tratamiento: ver la ronda de corrección acotada en `design.md`, en la tabla «Huecos de cobertura del design».
