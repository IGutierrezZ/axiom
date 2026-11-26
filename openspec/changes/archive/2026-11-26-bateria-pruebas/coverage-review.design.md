# Revisión de cobertura — spec y design de `bateria-pruebas`

Revisión informativa: no bloquea fases, puertas ni archivado, y no entra en ningún digest. En `continuous`, los huecos pasan al resumen final y no relanzan `sdd-design`.

## Ronda 1 — 2026-11-26

Revisores: A (`jd-judge-a`) y B (`jd-judge-b`), ciegos y en paralelo. Los dos devolvieron resultado utilizable. Entradas: `proposal.md`, las cuatro specs delta y `design.md`.

Contexto: antes de esta ronda, el orquestador ajustó tres puntos de la spec para alinearla con las preguntas abiertas del design:

- En RF-KOF-21, el JSON del verbo admite claves adicionales.
- En RF-KOF-22, omitir `acceptanceTests` equivale a proyectarlo como no activo.
- En RF-TOP-19, un entorno con `;` es un error.

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | uncovered_requirement | design D1 | D1 no recoge el rechazo de un entorno con `;`, que fija RF-TOP-19, y lo deja como pregunta abierta. | A, B |
| 2 | uncovered_requirement | design D6 | Falta el caso de la función no activa con `pruebas.md` presente, que RF-KOF-21 sí cubre. Con `environments` vacío, todas las filas darían `entorno_invalido`. Tampoco se dice qué pasa con el bloque inválido y el fichero presente. | A, B |
| 3 | uncovered_requirement | design D9 | RF-GAT-32 se delega en la skill `sdd-tasks` sin tocar el orquestador. No cubre estos casos: reiterar la parada, reanudarla en otra sesión, o que `sdd-tasks` no escriba el fichero (hay que declarar su ausencia). Tampoco hay test ni pin. | A, B |
| 4 | edge_case | design D8 | No se dice qué devuelve `incrementAcceptanceTests` para un incremento ODD o un cambio archivado. Puede salir un `error` espurio en la pestaña. | A, B |
| 5 | edge_case | design D3 | Una entrada `ID#` con escenario vacío se confunde con la cita del requisito. No se dice si las entradas repetidas son hallazgo o se deduplican. | A, B |
| 6 | edge_case | design D3 | Una segunda tabla se ignora en silencio, aunque RF-KOF-18 exige una tabla única. | B |
| 7 | uncovered_requirement | design D1 | D1 no describe el bloque nulo o que no es un mapeo, que es un error tipado en RF-TOP-19. | B |
| 8 | uncovered_requirement | design, estrategia de pruebas | Ningún test fija «Archivado no bloqueado» de RF-TOP-20, ni `Locate` con el bloque inválido. | A |
| 9 | error | design D7 | No se dice qué hace el archivado con una tabla inválida o con un provisional sin correspondencia en `pruebas.md`. | A |

Total: 9 huecos únicos (A: 7, B: 5; 3 compartidos).

### Nota del orquestador

En `continuous` no se relanza `sdd-design`. Los huecos 1, 2, 5, 6, 7 y 8 se resuelven en `tasks` aplicando la spec vigente, que manda sobre el design. Los huecos 3, 4 y 9 quedan como riesgo para el resumen final. El 3 en concreto necesita una frase en el texto compartido de la parada `tasks` del orquestador, que no se toca en este cambio porque C-05-d edita la doctrina en paralelo.
