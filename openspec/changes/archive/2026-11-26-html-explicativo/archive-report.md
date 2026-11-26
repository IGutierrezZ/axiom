# Reporte de Archivado Formal: html-explicativo

## Resumen de Cierre
- Fecha de Archivado: 2026-11-26
- Estado de Verificación: Verificación local limpia (`verify-report.md`, en el momento de la verificación: 0 críticos, 0 avisos, 1 sugerencia). Tras el verify, la primera CI falló en tres puntos, corregidos en el commit `339b5c2c` (ver «Estado final»). PR https://github.com/IGutierrezZ/axiom/pull/350 (`type:feature`) fusionada con merge commit `410105d9`; CI verde.
- Modo de artefactos: `hybrid` (ficheros en `openspec/` y trazas en Engram).
- Kickoff: `continuous`, rol único `fullstack`, `handoff_policy: none`, `post_archive_policy: bug_only`, sin `docs_publication` y sin `explainer_html` (sellado anterior a esa función).
- Destino en Archivo: `openspec/changes/archive/2026-11-26-html-explicativo/`

## Estado final (a fecha de cierre)
Fuentes, por orden de autoridad: (1) `tasks.md` archivado; (2) instrucciones de cierre del orquestador; (3) `verify-report` y `apply-progress` (instantáneas).
- Tareas: 12/12 completadas (`tasks.md`, observación Engram #2677). Sin tareas pendientes.
- Implementación: un commit por tarea (`apply-progress` #2680, en el momento del último lote). El commit `339b5c2c` se añadió a la rama del PR tras el verify y antes de la fusión `410105d9`.
- Correcciones posteriores al verify (fuente: orquestador, commit `339b5c2c`):
  - Ratchet de código muerto: `Validate`, `CitedIDs` y `Kinds` movidos de `internal/explainer/explainer.go` a `internal/explainer/contract_test.go`, junto al test `TestExplainerHTMLSkeletonMatchesExample`, que salió de `internal/assets`.
  - Color fijo `#fff` en `.explainer-frame` sustituido por `var(--surface)`.
  - `/api/explainer` clasificada en `excludedRoutes` del canario de secretos.
- Verify: según `verify-report` (#2681, en el momento de la verificación) el estado fue limpio con 1 sugerencia abierta: la CSP no lleva directiva `sandbox`. Esa sugerencia no fue corregida en los commits posteriores registrados y queda como hallazgo no bloqueante. Los tres fallos de la CI anterior a `339b5c2c` no figuran en el verify, por ser posteriores a él.
- Revisión de cobertura (informativa, no bloqueante): 11 huecos en la spec (`coverage-review.spec.md`, #2671) y 11 en el design (`coverage-review.design.md`, #2674). Los del design se corrigieron con una única relanzada del gatekeeper.
- Pruebas: los números finales de pruebas no se han copiado de la instantánea de verify; los datos de la CI del PR son los vigentes.

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. No hay requisitos REMOVED ni superados por este cambio. RF-ISO-01 se modifica en sitio (MODIFIED) y no queda deprecado.

## Identificadores Asignados
| Capacidad | Requisito | Provisional | Definitivo | Estado |
|-----------|-----------|-------------|------------|--------|
| local-web-dashboard | Endpoint de Solo Lectura que Sirve Únicamente `resumen-spec.html` y `resumen-design.html` | prov:WEB:1 | RF-WEB-15 | assigned |
| local-web-dashboard | Pestaña «Explicación» en el Detalle del Incremento con `iframe` en Sandbox y Aviso de Desactualizado | prov:WEB:2 | RF-WEB-16 | assigned |
| sdd-block-review-gates | Contrato del HTML Explicativo que Generan `sdd-spec` y `sdd-design` cuando el Kickoff Sella `yes` | prov:GAT:1 | RF-GAT-27 | assigned |
| sdd-block-review-gates | Contenido Fiel del HTML: Cita de IDs, Nada Fuera del md y Aviso de que se Aprueba el md | prov:GAT:2 | RF-GAT-28 | assigned |
| sdd-block-review-gates | El HTML Queda Fuera del Digest de las Puertas | prov:GAT:3 | RF-GAT-29 | assigned |
| sdd-block-review-gates | Frescura Informativa `fresh`, `stale` o `missing` del HTML en `axiom sdd status` | prov:GAT:4 | RF-GAT-30 | assigned |
| sdd-lane-kickoff | Declaración de Proyecto `sdd.explainer_html` con Valor por Defecto `ask` | prov:KOF:1 | RF-KOF-13 | assigned |
| sdd-lane-kickoff | `kickoff.yaml` Registra el Campo Sellado `explainer_html` con Valores `yes` o `no` | prov:KOF:2 | RF-KOF-14 | assigned |
| sdd-lane-kickoff | La Bandera `--explainer-html` y la Resolución contra el Valor del Proyecto | prov:KOF:3 | RF-KOF-15 | assigned |
| sdd-lane-kickoff | El Status Proyecta el Valor de Proyecto y el Valor Sellado de `explainer_html` | prov:KOF:4 | RF-KOF-16 | assigned |
| sdd-preflight-configuration | La Pregunta del HTML Explicativo es una Interacción Propia de Marcador `Axiom SDD kickoff explainer:`, Una Vez por Cambio y Solo con `ask` | prov:PRE:1 | RF-PRE-16 | assigned |

Requisito modificado sin provisional: RF-ISO-01 (sdd-iso-evidence, MODIFIED).
Duplicados históricos fuera del cambio: ninguno (`duplicates[]` vacío).

## Integración de Especificaciones
- Comando: `axiom sdd archive-integrate --cwd C:\repos\axiom-wt\html-explicativo-archive\axiom --change html-explicativo --json`, código de salida 0 (`axiom.sdd-archive-integrate/v1`, `nothingToIntegrate: false`).
- Previo: `axiom sdd archive-check --change html-explicativo` con código 0, `iso: false`, sin hallazgos.
- Capacidades integradas (outcome `composed`): local-web-dashboard, sdd-block-review-gates, sdd-iso-evidence, sdd-lane-kickoff, sdd-preflight-configuration.
- Escritos por el verbo: 5 specs de `openspec/specs/`; en el cambio, las deltas de `specs/` y `tasks.md` (identificadores `Cubre:` reescritos) y `ids.yaml`.
- Índice maestro: `axiom archive sync --path` con código 0; `openspec/INDEX.md` regenerado (49 especificaciones, 527 requisitos activos, 2380 escenarios).
- Comprobación de provisionales: ningún `prov:` queda como identificador de requisito en `openspec/specs/`. Las apariciones de `prov:` en `openspec-adoption/spec.md` (línea 136 y 150) y en `sdd-orchestrator-assets/spec.md` (líneas 502 y 508) son texto descriptivo de un formato, no identificadores asignados; no se han tocado.

## Trazabilidad Engram (hybrid)
Observaciones leídas o referenciadas para este cierre: proposal #2667, spec #2670, design #2673, tasks #2677, apply-progress #2680, verify-report #2681, coverage-review spec #2671, coverage-review design #2674, explore #2665.

## Verificación de la Movida
- `diff -r` (snapshot previo a la movida vs. destino): sin diferencias (salida vacía, código 0).
- `git mv` del directorio completo; el origen `openspec/changes/html-explicativo/` ya no existe.

## Artefactos Consolidados
- `proposal.md`
- `specs/` (5 capacidades)
- `design.md`
- `tasks.md`
- `verify-report.md`
- `coverage-review.spec.md`, `coverage-review.design.md`, `exploration.md`, `kickoff.yaml`, `ids.yaml`

## Riesgos y Hallazgos Abiertos
- Sugerencia del verify sin resolver: la CSP no lleva directiva `sandbox`; el aislamiento es por atributo del `iframe`, conforme a la spec.
- Los 11 huecos de la revisión de cobertura de la spec no se han recogido en la spec; son informativos.
- Los tres fallos de la CI corregidos en `339b5c2c` no tienen un verify posterior registrado.
- Los ficheros `tasks.md` y las deltas del cambio cambiaron en bytes por la reescritura de identificadores del verbo `archive-integrate`; no es una alteración manual.
