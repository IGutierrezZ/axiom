# Revisión de cobertura de la spec — arranque-desde-wi

Revisión informativa: no bloquea fases, compuertas ni archivo. Revisores ciegos `jd-judge-a` (A) y `jd-judge-b` (B), en paralelo, sobre `proposal.md` y `specs/*/spec.md`.

## Ronda 1 — 2026-11-26

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | edge_case | `change-start-from-work-item` prov:74:14 | Contradicción: `--adopt` retira el staging al completar contexto y vínculo, pero un segundo `--adopt` (idempotente o reintento del enlace de vuelta) se rechazaría con `staging_not_found`. Falta de dónde sale la identidad del WI en el reintento (vínculo de `integrations.yaml` o staging conservado hasta completar la etiqueta) y, en ODD, sin `integrations.yaml`. | A, B |
| 2 | edge_case | `change-start-from-work-item` prov:74:14 | `--adopt` interrumpido a mitad del movimiento o con `context/` ya existente en destino: fusionar, sobrescribir o rechazar. | A, B |
| 3 | edge_case | `change-start-from-work-item` prov:74:5 y prov:74:8 | El nombre incluye el slug del título; si el título cambia entre dos fases 1 quedan dos stagings del mismo WI. La detección de staging existente debe ser por WI, no solo por nombre. WI ya adoptado en un cambio ODD abierto no se detecta (ODD no escribe `integrations.yaml`). | A, B |
| 4 | error | `change-start-from-work-item` prov:74:6 | Sin código tipado para PAT ausente o configuración incompleta (`organization`/`project`), `--cwd` fuera de un repositorio git (sin common dir) y fallo del descubrimiento de categorías con el WI ya leído. | A, B |
| 5 | edge_case | `change-start-from-work-item` prov:74:12 | Imágenes incrustadas: `data:`, rutas relativas, esquemas `file:`/`ftp:`, URL repetida (descarga doble y presupuesto); reescritura del `<img src>` a la ruta local; adjunto sin `Content-Type` o con `application/octet-stream`; coherencia extensión/tipo en Office. | A, B |
| 6 | use_case | `organic-agent-trigger-rules` prov:74:25 | Perfil bug en ODD: cómo se refleja (`Tipo: bug`, `## Spec viva` o `## Impacto en la spec`) y quién lo escribe. | A |
| 7 | edge_case | `change-start-from-work-item` prov:74:1 | `--lane` con `--from` o `--discard`: rechazo de uso (código 2) no fijado. | A |
| 8 | edge_case | `change-start-from-work-item` prov:74:14 | En ODD, `change_not_prepared` no está definido (`ensure --lane odd` no crea `odd/context/<cambio>/`); `--lane` incongruente con el cambio. | B |
| 9 | uncovered_requirement | `odd-living-document` RF-ODO-01, `dashboard-sdd-orchestration` RF-DSO-04 | La propuesta las declara condicionales y no hay delta ni decisión registrada de dejarlas fuera. | B |
| 10 | edge_case | `sdd-lane-kickoff` prov:74:24 | Fallo de `ensure` (`decision_required` o error) o de `seal` tras una fase 1 correcta, y rechazos de `--adopt` desde el orquestador. | B |
