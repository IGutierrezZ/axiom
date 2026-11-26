# Revisión de cobertura · design · marcas-pruebas

Revisión informativa: no bloquea fases, puertas ni archivado y no edita el design.

## Ronda 1 — 2026-11-26

Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y en paralelo. Ambos devolvieron un resultado utilizable. Informe = unión deduplicada.

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | edge_case | `design.md` · D3 / D15 | Las pendientes se calculan contra `refs/remotes/origin/<rama>` sin red: con la referencia obsoleta o ausente todo aparece pendiente, se fuerza un reintento innecesario y `nothing_to_publish` puede no salir. El límite no se declara en la UI. Con `pendingUnavailable` no se dice de dónde salen remoto y rama que la UI debe mostrar antes del clic. | A, B |
| 2 | error | `design.md` · D15 / D12 | Con `pendingUnavailable` (p. ej. `wrong_branch`, sin remoto) la pantalla deja de listar pendientes y solo deshabilita el botón: no se muestra el rechazo tipado ni su continuación (prov:WEB:1, prov:WEB:3). | B |
| 3 | edge_case | `design.md` · D2 / D13 `marks_file_invalid` | Dos personas que añaden al final producen conflicto de texto en un `git pull` manual; el fichero queda ilegible y la continuación propuesta (`git checkout -- <ruta>`) descarta entradas locales, contra «las pendientes no deben perderse». | A |
| 4 | error | `design.md` · D9 paso 4 / D13 | `other_commits_unpublished` sin continuación concreta definida (D13 la cita con «…»), sin distinguirlo en la UI y sin salida para las marcas que quedan pendientes. Debe nombrar resolución para el trinquete de rechazos. | A |
| 5 | edge_case | `design.md` · D4 | «Vigente» por mayor `at`: con relojes desfasados una corrección posterior puede perder. El riesgo no se documenta como aceptado. | A |
| 6 | edge_case | `design.md` · D4 | Un par con marca vigente y otra caducada anterior no cuenta como caducada; no se decide la celda de UI para vigente `ko` + caducada `ok`. | B |
| 7 | edge_case | `design.md` · D12 | RF-KOF-22 proyecta el resumen solo con `pruebas-estado.yaml` «con entradas»; D12 lo proyecta si el fichero existe (vacío o solo inválidas), y prov:ARC:1 exige no incluir la sección sin marcas. | B |
| 8 | edge_case | `design.md` · D7 / D10 | Cerrojo solo de proceso: ediciones manuales o git externo durante R4–R5 no cubiertas; no se trata `pruebas-estado.yaml` sucio al llegar a `reset --keep`. | B |
| 9 | use_case | `design.md` · D14 | `marks_publisher_missing` se evalúa primero: en monorepo sin publicador se respondería ese código y no el motivo de topología que exige prov:WEB:4; orden con varias condiciones sin garantizar. | B |
| 10 | uncovered_requirement | `design.md` · D13 / D6 | El campo `change` del cuerpo no se valida contra recorrido de rutas (`../x`, separadores) antes de `filepath.Join(PlanningHome, "changes", <x>)`; falta en la matriz de amenazas. | B |
| 11 | error | `design.md` · D10 | Éxito parcial del reintento (commit creado y segundo push rechazado) y `reset --keep` abortado por documentos sucios cuando el remoto avanzó: `replay_failed` sin salida concreta distinta de `ORIG_HEAD`. | B |

Tratamiento: en `continuous` los huecos van al resumen final y no se relanza `sdd-design`. Los que la spec ya exige (7, 9, 10 y la visibilidad del rechazo en 2 y 4) se pasan a `sdd-tasks` como requisitos de conformidad de la implementación.
