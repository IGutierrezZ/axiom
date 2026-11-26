# Revisión de cobertura · spec · adopcion-ids

Revisión informativa tras la fase `spec` (`skills/_shared/sdd-coverage-review.md`). No bloquea ninguna fase y no cambia ningún artefacto revisado.

## Ronda 1 — 2026-11-26

Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y en paralelo. Entradas: `proposal.md` y las tres deltas de `specs/`. Los dos dieron un resultado utilizable. A contrastó además `internal/reqid` (`provisional.go`, `reqid.go`, `allocate.go`); B no consultó código.

Huecos (unión deduplicada):

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | edge_case | REQ-30.11 (f) | La regla (f) solo compara el ID del título con el `<ID original>` escrito en `Origen:`. Si la unidad del inventario trae un ID ajeno pero `Origen:` lo omite, un título provisional o con otro definitivo pasa sin aviso y el ID del cliente se pierde. Falta validar contra los `ForeignIDs` de la unidad en el inventario. Tampoco se define si un definitivo es válido cuando la unidad no trae ID (un ID inventado). | A, B |
| 2 | edge_case | REQ-30.10 / REQ-30.12 | No se define qué ID se conserva cuando una unidad trae varios IDs ajenos (`RF-001 / RF-002`), ni qué pasa si el mismo ID ajeno aparece en dos unidades u orígenes. Falta decir cómo lo trata `scan` (¿conflicto?) y `check`. | A, B |
| 3 | edge_case | REQ-30.9 (regla 2) | La inferencia solo tiene escenarios para `<NNN>` y `<m>`. Faltan: números sin relleno de ancho único (`RF-1`…`RF-9`, ¿`<n>` o `<m>`?), ancho fijo de 2 (`<NN>`), anchos mixtos con ceros (`RF-01` y `RF-100`), un solo ID, y el esquema con dominio (`RF-USU-01` → `RF-<DOM>-<NN>`), incluido el dominio con anchos mixtos. | A, B |
| 4 | error | REQ-30.11 (g) | `pattern_undeclared` exige citar las líneas `sdd:`/`id_pattern:` a copiar, pero con un conflicto (RF + RNF) no hay plantilla propuesta. No se define el motivo en ese caso ni si se emiten a la vez `pattern_undeclared` y el fallo de formato cuando el patrón declarado solo cubre parte de los IDs. | A, B |
| 5 | edge_case | REQ-30.11 (g) / REQ-30.9 | No se dice si `pattern_undeclared` y la propuesta de `scan` excluyen los IDs ajenos de unidades descartadas (`Descartado:`) o no migradas. Un glosario descartado con `RNF-001` forzaría un conflicto innecesario. | A, B |
| 6 | edge_case | REQ-30.12 / REQ-30.13 | Falta el origen mixto (unidades con ID y sin ID bajo un patrón declarado). Hay que aclarar: que los provisionales toman la serie del patrón declarado, que el orden de asignación frente a los definitivos conservados del mismo cambio es determinista y sin colisión, y qué ocurre al rebasar el ancho (`RF-999` → `RF-1000` con `<NNN>`). | A, B |
| 7 | error | REQ-30.10 / REQ-30.11 (a) | La regla (a) acepta «un provisional válido (REQ-47.1)» sin exigir que su serie sea compatible con el patrón efectivo. Un `(prov:40:1)` con `RF-<NNN>` pasaría `check` y fallaría luego en `archive-integrate`. Tampoco se fija la sintaxis del provisional con un patrón de serie fija sin `<n>` ni `<DOM>`, para el que `reqid` rechaza las etiquetas de serie nueva (`internal/reqid/provisional.go:197,213`). | A |
| 8 | edge_case | REQ-30.11 | No se cubre un inventario reutilizado tras cambiar `sdd.id_pattern`, porque la clave (serie, número) depende del patrón vigente. Tampoco se cubre un `Origen:` cuyo ID original sigue otro patrón en un requisito con provisional, ni se especifica que editar `axiom.yaml` no dispare `inventory_stale`. | B |
| 9 | use_case | REQ-30.9 | Falta `scan` con `sdd.id_pattern` ya declarado (re-escaneo tras declarar): ¿muestra la propuesta junto al patrón declarado, la omite, avisa si difieren? | A |

Total: 9 huecos (7 comunes, 1 solo de A, 1 solo de B). Se entregaron al design como contexto mientras se redactaba. En `continuous` no se relanza `sdd-spec`: los huecos que el design no cierre van al resumen final.
