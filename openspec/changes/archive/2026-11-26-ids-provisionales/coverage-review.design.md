# Revisión de cobertura · design · ids-provisionales

Revisión informativa (`skills/_shared/sdd-coverage-review.md`): no bloquea ninguna fase, puerta ni archivado, y no modifica la spec ni el design. Estilo `continuous`: los huecos van al resumen final.

## Ronda 1 — 2026-11-25

- Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y de solo lectura. Los dos dieron un resultado utilizable.
- Entradas: `proposal.md`, las delta specs de `specs/**` y `design.md`.
- Unión deduplicada: 16 huecos.

| # | Tipo | Ubicación | Hueco | Base | Revisor |
|---|---|---|---|---|---|
| 1 | caso límite | D5, fila 4 | Si el autor edita el cuerpo de un ADDED ya integrado (traza P→D) y reintegra, el design lo trata como ocupado por otro y le asigna D'. En la spec viva quedan el bloque viejo y el nuevo. REQ-30.3 exige rechazarlo con un motivo tipado. | REQ-30.3 («Un requisito ya presente con un cuerpo distinto … DEBE rechazarse»); D5 («Cualquier otra aparición es ajena y gana») | A, B |
| 2 | requisito sin cubrir | D4 (canónica inexistente) | Una capacidad nueva cuya delta trae MODIFIED, REMOVED o RENAMED se copia tal cual. No se valida el ID inexistente y esas secciones llegan a `openspec/specs/`. | REQ-47.3 punto 2; D4 («copia del texto de la delta») | A, B |
| 3 | caso límite | D4 | `--superseded-requirements` es una lista única para todo el cambio. No se dice a qué capacidad se aplica cada nombre, qué pasa si casa con ninguna o con varias, ni cómo actúa `--supersede` en un cambio con varias capacidades. | D4; REQ-30.22 | A |
| 4 | error | D9 | El design omite `requirements` también con un patrón inaccesible, con almacén engram o none y con el cambio archivado. REQ-47.9 solo admite omitirlo cuando no hay deltas. Además, `modifies` deja sin `id` un MODIFIED o REMOVED por título. | REQ-47.9 («DEBE omitirse cuando el cambio no tiene deltas»); D9 | A, B |
| 5 | caso límite | D12 / D11 | La sección ODD `## Identificadores asignados`, con su valla `yaml`, no se contrasta con los validadores del documento ODD (REQ-30.1 y REQ-30.5) ni con el lector de la valla `openspec`. Solo se comprueba el dashboard. | REQ-30.2 («validar sus secciones (REQ-30.1, REQ-30.5)»); D12 | A, B |
| 6 | error | D12 / D7 | No hay comportamiento ni código para una traza (`ids.yaml` o la sección ODD) ilegible, con un `pattern` distinto del efectivo, con una capacidad que ya no existe, con definitivos no válidos o borrada en un rebase. La reintegración depende de esa traza. | D5 («La traza es la fuente de identidad de lo ya asignado») | A, B |
| 7 | requisito sin cubrir | D4 / A3 | El `--check` de ODD no menciona la comprobación estructural del contexto (REQ-30.6), ni que ejecute exactamente las mismas validaciones que la integración real. Tampoco aclara si participa `INDEX.md`. | REQ-47.7 («… comprobación del contexto») | A |
| 8 | caso límite | D3 (orden) | Las etiquetas de serie nueva se asignan por primera aparición, con las capacidades en orden alfabético de carpeta. Renombrar una carpeta, o añadir después una capacidad que va antes en ese orden, cambia los números. No se define cómo se comporta frente a una traza previa. | REQ-47.2 («orden fijo y reproducible»); D3 | A |
| 9 | error | D3 (atomicidad) | No se define qué se informa ni cómo se recupera si falla la restauración en orden inverso (por ejemplo, un fichero bloqueado en Windows). Tampoco qué hacen el plan y `archive-check` ante un `.<base>.axiom-tmp` huérfano de un proceso que se mató. | REQ-47.4 («no queda ningún fichero temporal en el árbol») | A |
| 10 | requisito sin cubrir | D7 / D3 | `UnknownRequirementIDError{Section, Requirement, ID}` no lleva la capacidad que exige REQ-47.3. La validación previa (D3, paso 2) no incluye el ID inexistente, así que `--check` no lo detecta antes de escribir. | REQ-47.3 («nombre la capacidad, la sección, el requisito y el identificador») | B |
| 11 | requisito sin cubrir | D7 | Hay una tensión sin decidir: REQ-47.3, punto 2, exige que MODIFIED y REMOVED citen un definitivo existente, pero el design conserva la resolución por título de REQ-30.22. No figura en «Alineación con la spec». | REQ-47.3 punto 2; D7 | B |
| 12 | caso límite | D7 (`ProvisionalIDError`) | Un provisional que solo aparece en el cuerpo de un ADDED o MODIFIED podría pasar por `archive-compose` y llegar a la spec viva. No se dice que `Compose` escanee los cuerpos. | REQ-47.1 punto 4 | B |
| 13 | caso límite | D6 | No hay forma de escapar un token con forma de provisional en el cuerpo de una delta, en las tareas o en el documento ODD (por ejemplo, para documentar el propio formato). | D6 | B |
| 14 | error | D4 (`partially_integrated`) | `partially_integrated` no tiene continuación: si se usó `archive-compose` en un dominio, el archivado no puede completarse. REQ-47.8 lo exige reanudable. | REQ-47.8 («DEBE ser reanudable») | B |
| 15 | caso límite | D1 / D7 | Un ADDED con dos IDs en el nombre (provisional y definitivo, o dos provisionales) fuera de ISO: `ProvisionalInTitle` toma el último y no hay código para el nombre ambiguo. | REQ-29.3 («Bloquea: es ambiguo»); D1 | B |
| 16 | caso límite | D3 (ocupación) | No se define qué hacer con deltas activas ilegibles o con definitivos que colisionan entre sí al calcular el siguiente libre. Tampoco hay coordinación entre dos integraciones simultáneas en dos worktrees: el duplicado solo se detecta después. | D3 paso 3; REQ-47.5 | B |

## Desalineaciones que declara el propio design

- **Destinos de reescritura.** D8 reescribe la columna `Requisito` de la matriz de `verify-report*.md`, que REQ-47.4 no incluye entre sus destinos. Hay que añadirla a REQ-47.4 o a REQ-29.6 (cierra el hueco 3 de `coverage-review.spec.md`).
- **Ocupación.** El design cuenta también los definitivos de los cambios activos. REQ-47.2 solo cita «la spec viva».
