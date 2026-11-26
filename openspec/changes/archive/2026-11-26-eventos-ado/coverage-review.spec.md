# Revisión de cobertura de la spec — eventos-ado

## Ronda 1 — 2026-11-26

Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), a ciegas y en paralelo. Entradas: `proposal.md` y las 5 deltas de `specs/`. Ninguno leyó el contexto opcional de `docs/plan/backlog-2026-11/07-integraciones.md`. Unión deduplicada: 22 huecos. Informativo: no bloquea fases ni compuertas.

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | edge_case | `integration-plugins` · `prov:INT:7` | En un segundo cierre del rol (rechazado y reabierto), el reloj vuelve a contar desde `started_at`, la primera entrada, y no descuenta lo imputado en el cierre anterior: doble imputación del mismo tiempo. | A, B |
| 2 | edge_case | `azure-devops-adapter` · `prov:ADO:6` | Reentrada del rol: la clave `role.start/<cambio>/<rol>` ya está consumida y la Task se queda en `Tech Review`. Sin definir: una Task en un estado posterior al destino (`Closed`) o ya asignada a otra persona. | A, B |
| 3 | error | `azure-devops-adapter` · `prov:ADO:6` | Una transición que prohíben las reglas del flujo, aunque el estado exista: no se dice si es permanente o transitoria ni qué muestra. | A, B |
| 4 | error | `azure-devops-adapter` · `prov:ADO:6` | `task_not_bound` es permanente en `Deliver`. Un `sync` en una máquina que aún no tiene el `integrations.yaml` con el vínculo abandona la operación sin reintento. | A |
| 5 | error | `integration-plugins` · `prov:INT:8` | El PATCH de horas tiene éxito pero falla la escritura local de `integrations.yaml`: las horas quedan sin `TimeEntry` y un reintento las duplica. | A, B |
| 6 | edge_case | `integration-plugins` · `prov:INT:8` | Precisión de `--hours`: 0.001 h redondea a 0 minutos, contra «entero positivo». Sin tope superior ni precisión decimal para ADO. | A |
| 7 | use_case | `integration-plugins` · `prov:INT:3` | No se puede corregir un vínculo erróneo ni sustituir una Task borrada o cerrada. La resolución de `task_already_bound` (`tasks create`) no resuelve nada. | A |
| 8 | use_case | `integration-plugins` · `prov:INT:2` | Si se itera el design con otra estimación tras crear las Tasks, `tasks create` reutiliza el vínculo y no propaga las horas nuevas. | A, B |
| 9 | edge_case | `integration-plugins` · `prov:INT:2` | `--estimate` para un rol fuera del roster, o en `bind` para un rol distinto del de `--role`: sin definir. | A |
| 10 | uncovered_requirement | `azure-devops-adapter` · `prov:ADO:1` | Sin definir el formato del título de la Task ni si hereda `System.AreaPath` y `System.IterationPath` del WI padre. | A |
| 11 | edge_case | `azure-devops-adapter` · `prov:ADO:5` | Con `req_tags_over_limit` no se retiran las `req:` provisionales ya presentes, lo que contradice `prov:ARC:1` («no dejar ningún provisional»). | A |
| 12 | edge_case | `integration-plugins` · `prov:INT:5` | Renderizador: secciones `REMOVED`/`RENAMED`, primer bloque que no es párrafo, marcado sin cerrar, tamaño máximo del HTML, y `ul`/`li` sin escenario. Los IDs `REMOVED` también recibirían etiqueta. | A, B |
| 13 | edge_case | `integration-plugins` · `prov:INT:4` | `summary push` sobre un WI cerrado o de solo lectura tras `archive`; orden y carrera con la transición; varias carpetas archivadas con el mismo nombre; descripción editada a mano sobrescrita sin aviso. | A, B |
| 14 | edge_case | `integration-plugins` · `prov:INT:9` | Sin escenario `multirepo` con varios roles: dónde vive `integrations.yaml`, qué raíz usan los verbos y cierres en distinto orden. | A |
| 15 | edge_case | `integration-plugins` · `prov:INT:8` | `time record`, `time propose` y `ensure --role` no revalidan la Task: borrada, movida de padre o cerrada tras el vínculo, sin error tipado. | A, B |
| 16 | edge_case | `integration-plugins` · `prov:INT:2` | Dos `tasks create` concurrentes en máquinas distintas crean dos Tasks con la misma marca; solo se cubre la ambigüedad posterior. | A |
| 17 | edge_case | `sdd-orchestrator-assets` · `prov:ORQ:4` | En `continuous`, una propuesta de 0 h (fin de semana, minutos, reloj desfasado) choca con el rechazo de `--hours 0`: sin comportamiento definido. | B |
| 18 | error | `sdd-orchestrator-assets` · `prov:ORQ:4` | En `continuous`, sin definir qué hace el orquestador si falla `time propose` (`role_not_started`, `task_not_bound`, red). | B |
| 19 | edge_case | `azure-devops-adapter` · `prov:ADO:5` | ID que produce una etiqueta inválida (más de 400 caracteres, `,` o `;`); comparación sensible a mayúsculas frente a un ADO que no las distingue (PATCH perpetuos). | B |
| 20 | uncovered_requirement | `integration-plugins` · `prov:INT:7` | Redondeo y zona horaria remitidos al diseño, sin regla observable: DST, jornada que cruza medianoche y `started_at` en UTC frente a días locales. | B |
| 21 | edge_case | `azure-devops-adapter` · `prov:ADO:1` | Búsqueda de duplicados: paginación de hijas, Tasks en estado `Removed`, marca copiada a mano en otro tipo o cambio, y recuperación de una Task sin padre si falla WIQL. | B |
| 22 | use_case | `integration-plugins` · `prov:INT:9` | `work_item_not_linked` no nombra ningún comando de resolución (trinquete de rechazos). Compuerta `design` vuelta a `pending` por una edición de `design.md` con las Tasks ya creadas. | B |

Tratamiento: en `continuous` los huecos van al resumen final. Los que son decisiones de diseño se trasladaron como entrada a la fase `design`, que estaba en curso; su resolución consta en `design.md` y en `coverage-review.design.md`.
