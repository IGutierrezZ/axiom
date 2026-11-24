# Revisión de cobertura · spec · puerto-reqid

Revisión informativa (`_shared/sdd-coverage-review.md`): no bloquea fases, puertas ni archivo, y no corrige la spec. En `continuous` los huecos van al resumen final.

## Ronda 1 — 2026-11-22

Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y en paralelo. Entradas: `proposal.md` y las seis deltas de `specs/`. Ambos devolvieron un resultado utilizable. Informe = unión deduplicada.

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | requisito sin cubrir | `requirement-identifiers` · REQ-44.1 | No se fija si el patrón se lee de `axiom.yaml` en vivo o de una instantánea en `kickoff.yaml` (como `sdd.iso`, REQ-29.13), ni qué pasa con un cambio ya sellado si el patrón cambia. REQ-29.16 dice que las reglas ISO salen de la instantánea. | A, B |
| 2 | caso límite | `requirement-identifiers` · REQ-44.3 / REQ-44.5 | Gramática sin regla para literales con metacaracteres (`.`, `(`, `)`, `+`, `,`, espacios), su choque con `(ID)` en títulos y con `Cubre: A, B`; patrón que empieza por un token (sin literal inicial: el candidato queda indefinido); escalares YAML no textuales o `id_pattern:` nulo; espacios en los extremos. | A, B |
| 3 | caso límite | `requirement-identifiers` · REQ-44.5 | Límites del ID dentro de un título: prefijo o extensión de un ID válido (`RF-USU-01-bis`, `REQ-30.4.1`, `REQ-30.1abc`, `XREQ-30.1`), minúsculas (`rf-usu-01`) e ID repetido en el mismo título. | A, B |
| 4 | caso límite | `sdd-archive-lifecycle` · REQ-30.22 | El emparejamiento por ID solo se especifica para MODIFIED: faltan REMOVED con título cambiado, nombre antiguo de un RENAMED resuelto por ID, ADDED con un ID que ya existe bajo otro título, dos entradas de la delta con el mismo ID, MODIFIED que cambia el ID conservando el título, y MODIFIED cuyo ID y cuyo título apuntan a requisitos distintos. | A, B |
| 5 | error | `sdd-archive-lifecycle` · REQ-30.22 | Sin escenario para la interacción con la tolerancia D17 del archivado interrumpido (ADDED ya compuesto, MODIFIED con título cambiado ya aplicado). La propuesta lo delega al design. | A, B |
| 6 | caso de uso | `requirement-identifiers` · REQ-44.7 / REQ-44.6 | Proyecto que declara `RF-<DOM>-<NN>` y conserva specs vivas en `REQ-n.m` (estado transitorio hasta C-03-e): colisión, siguiente libre, índice de baseline, MODIFIED/REMOVED sobre IDs antiguos, consulta del bug y encabezados de `livingdoc` que pasarían a sintéticos. | A, B |
| 7 | requisito sin cubrir / error | `requirement-identifiers` · REQ-44.4 | Consumidores sin escenario con patrón declarado: `ScenarioKey` y matriz del verify (REQ-29.6 a 29.8), `axiom sdd trace`, textos de ayuda de `archive integrate --odd` y `archive adopt`, y qué hace `adopt check` con un patrón no por defecto mientras exige `REQ-n.m` (A-03-2), con salida accionable. | A, B |
| 8 | caso límite | `requirement-identifiers` · REQ-44.6 | `livingdoc` con patrón declarado: encabezado `RF-USU-01` que conserva su ID, cómo recibe el patrón sin workspace, sintéticos distintos y estables entre ejecuciones, encabezado con un ID válido de otro patrón. | A, B |
| 9 | caso límite | `requirement-identifiers` · REQ-44.3 | Ceros a la izquierda equivalentes (`RF-USU-01` frente a `RF-USU-001`, `REQ-30.04`): ¿mismo ID para unicidad, colisión, `Cubre:` y máximo?; desbordamiento al calcular máximo + 1; siguiente libre de una serie vacía; huecos en la serie. | A, B |
| 10 | caso límite | `requirement-identifiers` · REQ-44.7 | Relación entre `<DOM>` y la capacidad: si `RF-FAC-01` es válido en la capacidad `usuarios`, qué serie se propone cuando el ADDED no trae ID, y la mezcla de «dominio `USU`» y «capacidad `usuarios`» en los escenarios. | B |
| 11 | caso límite | `requirement-identifiers` · REQ-44.1 | `multirepo`: de qué `axiom.yaml` sale el patrón al ejecutar desde el repositorio de specs o desde el de código, y qué pasa si ambos declaran patrones distintos. | B |

Huecos abiertos: 11. Ninguno se corrige en esta fase (`continuous`): el design puede cerrar los que le competen (1, 2, 4, 5, 6, 8, 9, 10, 11) y el resto queda para el resumen final.
