# Revisión de cobertura — spec de `publicacion-pendiente`

Revisión informativa tras la fase `spec` (`sdd-coverage-review.md`). Dos revisores ciegos (`jd-judge-a`, `jd-judge-b`) con las mismas entradas: `proposal.md` y `specs/sdd-docs-publication/spec.md`. El informe es la unión deduplicada de sus huecos; no bloquea ninguna fase ni cambia ningún digest. En `continuous` los huecos van al resumen final y no se relanza `sdd-spec` por ellos.

## Ronda 1 — 2026-11-26

| # | Tipo | Ubicación | Hueco | Base | Revisores |
|---|---|---|---|---|---|
| 1 | edge_case | `spec.md#REQ-27.18` | Carpeta del cambio sin seguimiento que `git status` muestra como una sola entrada `dir/`: no se fija si cuenta como documentación sin confirmar ni cómo se separa del estado local con punto; falta escenario con git real. | Propuesta, Riesgos: «Una carpeta del cambio sin seguimiento aparece en `git status` como un único `dir/`… Se cubre con una prueba con git real.» | A, B |
| 2 | edge_case | `spec.md#Condición de proyección` | Primera publicación (sin `origin/<rama>`) sin nada que publicar, o con commits de código: no se fija la precedencia entre «primera publicación» y `code_unpublished` ni qué orden se proyecta. | Propuesta: «Si no existe, es la primera publicación y cuenta como pendiente, no como error.» | A, B |
| 3 | edge_case | `spec.md#REQ-27.18` (momentos `auto`) | Sin escenario para el momento de archivado, ni para el límite cuando el status no resuelve el cambio archivado o su rama (`PublicationRequest{Archive: true}`); tampoco escenarios para los momentos `design` y `tasks`, solo `spec`. | Propuesta: «Si no los resuelve, el momento de archivo queda documentado como límite y no se proyecta.» | A, B |
| 4 | edge_case | `spec.md#Condición de proyección` | No se define cuándo deja de contar un momento ya publicado en `continuous` (p. ej. `spec` publicada y `design` en curso), ni el caso `checkpointed` con aprobación cuyo digest cambió después. | Spec (a): «existe una aprobación registrada de una compuerta distinta de integration». | A |
| 5 | edge_case | `spec.md#Exclusiones` | Sin escenarios para varias aprobaciones en `checkpointed` (una ya publicada y otra nueva), `integration` aprobada junto a otra compuerta, ni compuertas solo rechazadas. | Propuesta, alcance 3(a). | B |
| 6 | uncovered_requirement | `spec.md#Una sola lista de rutas` | Falta escenario verificable de paridad entre la lista del status y la de `publish` (`docpaths`) y de no regresión de `publish` tras la extracción. | Propuesta, Riesgos: «Una sola composición en `docpaths`, usada por ambos, con prueba de paridad.» | A, B |
| 7 | edge_case | `spec.md#REQ-27.19` | Tras un `publish` con éxito parcial (commit local y push fallido) o correcto, no se fija la reevaluación del status (paso a rama por delante o a `code_unpublished`) ni cómo se acota «una vez» cuando coinciden transición y reanudación o la sesión se corta. | Propuesta, Riesgos: «una vez por transición o reanudación». | A, B |
| 8 | edge_case | `spec.md#Multirepo` | Multirepo: no se aclara el repositorio de código, `code_unpublished` en el repo de specs, ni el repo de specs fuera de su rama de integración; en `monorepo-embedded`, worktree en otra rama o HEAD desacoplado, o destino negado por `PublicationTarget`. | Propuesta, alcance 5 e Invariantes: «una negativa de `PublicationTarget` nunca hace fallar el status». | A, B |
| 9 | edge_case | `spec.md#Estados (code_unpublished)` | Commit sin subir mixto (documentación y código) y rango que excluye `origin/<integración>` (mismo rango que REQ-27.8) sin escenario. | Propuesta, alcance 1: «con el mismo rango que `publish` (REQ-27.8)». | B |
| 10 | edge_case | `spec.md#Estados` | Tokens de los motivos sin nombrar; sin escenario con varios motivos simultáneos (sin confirmar y por delante) ni que `code_unpublished` omite `invocation`. | Propuesta, alcance 2: «estado, motivos, remoto, rama e invocación». | B |

Revisores sin resultado utilizable: ninguno.
