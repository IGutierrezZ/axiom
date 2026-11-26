# Revisión de cobertura · spec · marcas-pruebas

Revisión informativa: no bloquea fases, puertas ni archivado y no edita la spec.

## Ronda 1 — 2026-11-26

Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y en paralelo. Ambos devolvieron un resultado utilizable. Informe = unión deduplicada.

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | uncovered_requirement | `specs/local-web-dashboard/spec.md` · prov:WEB:3 | No se define qué es una entrada «pendiente de publicar» en un fichero de solo añadir (contra qué se compara: remoto, último commit publicado o registro local), ni cómo se evita duplicarlas al reproducir. Sin ello no son verificables «sin entradas pendientes» ni prov:PUB:2. | A, B |
| 2 | error | `specs/local-web-dashboard/spec.md` · prov:WEB:2 | Falta el comportamiento de la ruta de marca cuando no hay identidad git (`user.name`/`user.email`) en el repo de specs: `identity_missing` solo está tipado en la publicación (prov:PUB:1). | A, B |
| 3 | edge_case | `specs/local-web-dashboard/spec.md` · prov:WEB:2 | El campo `commit` no se valida (longitud, formato) y no hay escenario de serialización segura de `note`/`commit` con saltos de línea o caracteres especiales de YAML. | A, B |
| 4 | edge_case | `specs/sdd-lane-kickoff/spec.md` · RF-KOF-22 / prov:KOF:1 | El resumen no define cómo cuentan `n/a`, `ko`, `blocked` y los casos sin marca, varias marcas vigentes del mismo caso y entorno, un entorno sin casos aplicables (denominador 0), ni marcas de casos o entornos que ya no existen. Las tres presentaciones deben salir del mismo cálculo. | A, B |
| 5 | error | `specs/sdd-docs-publication/spec.md` · prov:PUB:2 | El reintento no cubre los fallos de sus propios pasos (`fetch` inalcanzable, avance rápido imposible, YAML remoto inválido), el estado local tras agotarlo (¿se conserva el commit local? RF-PUB-05), la interrupción entre descartar y reañadir, ni entradas idénticas ya presentes en el remoto. | A, B |
| 6 | edge_case | `specs/local-web-dashboard/spec.md` · prov:WEB:2 / prov:WEB:3 | La exclusión mutua solo se exige entre marcas: falta qué ocurre con una marca durante una publicación y con dos publicaciones solapadas (doble clic, dos pestañas): serializar o rechazo tipado de ocupado. | A, B |
| 7 | edge_case | `specs/local-web-dashboard/spec.md` · prov:WEB:2 | No se fija sobre qué copia del repo de specs escribe la marca (solo prov:PUB:1 exige `InPlace` al publicar) ni de qué ubicación lee el status el resumen; dashboard y status podrían divergir. | A |
| 8 | use_case | `specs/sdd-archive-lifecycle/spec.md` · prov:ARC:1 | Marcas locales sin publicar al archivar: quedan sin publicar sin aviso, porque prov:WEB:4 deja la ruta «no disponible» para un cambio archivado. | A |

Tratamiento: en `continuous` los huecos van al resumen final; no se relanza `sdd-spec`. Se pasan a `sdd-design` como información para que fije las decisiones técnicas que les afecten.
