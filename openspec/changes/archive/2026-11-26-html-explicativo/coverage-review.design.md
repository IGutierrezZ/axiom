# Revisión de cobertura — spec y design de `html-explicativo`

Revisión informativa: no bloquea ninguna fase, puerta ni archivo, y no modifica los artefactos. Revisores ciegos: `jd-judge-a` (A) y `jd-judge-b` (B). El informe es la unión deduplicada de sus huecos.

## Ronda 1 — 2026-11-26

Entradas: `proposal.md`, los cinco `specs/*/spec.md` y `design.md` (primera versión). Los dos revisores devolvieron un resultado utilizable.

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | requisito sin cubrir | `design.md` · `decorateExplainer` | El diseño proyecta la frescura también cuando solo existe un `resumen-*.html` (`sealedYes \|\| hasFile`), aunque el kickoff selle `no` o no tenga el campo; prov:GAT:4 lo prohíbe. | A, B |
| 2 | requisito sin cubrir | `design.md` · tabla de frescura | El diseño asigna `missing` o `stale` a un artefacto sin md de origen; prov:GAT:4 dice que entonces no se proyecta estado. | A, B |
| 3 | requisito sin cubrir | `design.md` · `ExplainerDTO` y pestaña | El DTO no lleva el valor sellado: la pestaña no puede indicar «explicación no activada» ni evitar el `iframe` con `no` o sin campo (prov:WEB:2). | A, B |
| 4 | requisito sin cubrir | `design.md` · agentes | Las copias por agente solo remiten al paso de la skill; no exigen las restricciones del HTML ni hay comprobación que falle si desaparecen (prov:GAT:1). | A, B |
| 5 | requisito sin cubrir | `design.md` · hook | Falta el escenario del plugin de OpenCode, que también debe ignorar `Axiom SDD kickoff explainer:` (prov:PRE:1). | A, B |
| 6 | caso borde | `design.md` · `ReadExplainer` | Un HTML mayor de 5 MiB queda indefinido: se serviría truncado con 200. `ChangeExplainers` lee sin el mismo límite. | A, B |
| 7 | error | `design.md` · `ExplainerProjection` | Con `sdd:` inválido solo se emite `project: invalid`, sin el mensaje de error que pide prov:KOF:4. | A |
| 8 | caso borde | `design.md` · `decorateExplainer` | Si `ChangeArtifactPaths` o `ArtifactDigest` fallan con `yes`, se omiten `spec` y `design` y no se distingue «sin activar» de «error de cálculo». | A |
| 9 | requisito sin cubrir | `design.md` · pruebas de `Validate` | Ningún mecanismo ni prueba comprueba que los IDs citados existen en el md de origen (prov:GAT:2), ni siquiera sobre el esqueleto. | B |
| 10 | caso de uso | `design.md` · pruebas de status | No hay prueba de que un cambio ISO con HTML `stale` o `missing` no se bloquee ni condicione el archivado. | B |
| 11 | caso borde | `design.md` · dashboard | El dashboard llama a `ChangeExplainers` sin la condición de almacén ni el caso archivado; la spec exige la misma frescura en dashboard y status. | B |

Los huecos 1 a 5, 7 y 11 son contradicciones o requisitos de la spec sin diseño. El gatekeeper del modo automático (deriva respecto a las entradas) relanza `sdd-design` una vez con esas correcciones. No es una autocorrección por la revisión de cobertura. Los demás huecos pasan a tareas o al resumen final.
