# Revisión de cobertura del diseño — arranque-desde-wi

Revisión informativa: no bloquea fases, compuertas ni archivo. Revisores ciegos `jd-judge-a` (A) y `jd-judge-b` (B), en paralelo, sobre `proposal.md`, `specs/*/spec.md` (ya corregida por el gatekeeper) y `design.md`.

## Ronda 1 — 2026-11-26

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | uncovered_requirement | design D-21 | La doctrina no recoge: fallo de `ensure` (`decision_required` o error) o de `seal` tras la fase 1 (conservar staging, no ejecutar fase 2, reintentar o `--discard`); rechazo de `--adopt` que detiene el arranque antes de `proposal.md`/`sdd-explore`/documento ODD; no ejecutar `--discard` por cuenta propia; aviso de enlace pendiente que no detiene; perfil bug en ODD (`Tipo: bug`, `## Spec viva` o `## Impacto en la spec`). Sin pruebas de paridad para ello. | A, B |
| 2 | edge_case | design D-04 | `ReadItem` consulta los estados antes de validar el tipo mapeado: un tipo sin mapear con fallo de esa consulta daría `tracker_unavailable` en vez de `work_item_type_unmapped`, contra el orden de prov:74:6. | A, B |
| 3 | edge_case | design D-03/D-04 | Todo 404 se mapea a `work_item_not_found`; el 404 de estados, relaciones o adjuntos debe ser `tracker_unavailable` u omisión. | B |
| 4 | edge_case | design D-11, D-18, D-20 | La etiqueta `axiom:<otro>` y el tipo son texto del tracker: se usan sin validar en rutas (`changes/archive/…-<otro>`, posible recorrido), mensajes y continuaciones. Validar con `gitref.ValidBranchComponent` y neutralizar; `TestChangeStartJSONCarriesNoTrackerText` contradice los escenarios que exigen nombrarlos. | A, B |
| 5 | edge_case | design D-11 | «Archivado localmente» solo se define en SDD; una etiqueta de un cambio ODD terminado o archivado en otra máquina bloquea el WI para siempre. | B |
| 6 | uncovered_requirement | design D-19/D-05 | Falta deduplicación por URL de imágenes y adjuntos (una descarga, un consumo del presupuesto), omisión sin petición de `file:`/`ftp:`/`http:`/relativas con motivo propio, y `Content-Type` ausente tratado como `application/octet-stream`. | A, B |
| 7 | edge_case | design D-19 | `http.DetectContentType` no reconoce OLE ni distingue OOXML: hacen falta bytes mágicos explícitos (D0 CF 11 E0, ZIP con `[Content_Types].xml`). | A |
| 8 | uncovered_requirement | design D-15/D-12 | `os.Rename` del `context/` (que contiene `work-item.json`) vacía el staging en el paso 2: un fallo en el paso 3 contradice «el staging permanece intacto». Copiar y verificar, o redefinir. | A |
| 9 | error | design D-12/D-18 | Sin definir fallos de E/S en pasos 4 (exclude), 5 (`work-item.json`), 6 (borrado del staging), 8 (registro del enlace), en `--discard` y en cerrojos; riesgo de `context/` visible para git. | A, B |
| 10 | edge_case | design D-13/D-15 | Fase 1 repetida sobre un WI ya adoptado por el mismo cambio (falta `already_adopted` o rechazo); destino `context/` sin `work-item.json` legible → `staging_conflict`. | A, B |
| 11 | edge_case | design D-14 | `--lane` incongruente: no se mira el registro del otro carril para componer la continuación con `--lane odd`. | B |
| 12 | edge_case | design D-17 | Posición de `git_repository_required` en el orden de comprobaciones y su aplicación a `--discard`, `--adopt` y la consulta sin argumentos. | B |
| 13 | edge_case | design D-14/D-12 | Multirepo: qué common dir usa la fase 1 y dónde escribe la exclusión cuando la raíz del workspace no es git. | A |
| 14 | edge_case | design D-12 | Temporales `.tmp-*` y cerrojos huérfanos sin limpieza; el cerrojo por nombre permite dos stagings del mismo WI con títulos distintos. | A |
| 15 | use_case | design D-07 | El perfil bug compara literalmente `Bug`: sin vía de configuración en procesos localizados o personalizados. | A |
| 16 | edge_case | design D-21 | `sdd-propose`: con `context/` pero sin vínculo, la spec prohíbe escribir `external_ref` y el diseño lo extraería; otras fases que leen `context/` no reciben la regla de dato no fiable. | B |
| 17 | use_case | spec prov:74:4 | URLs de tablero (`?workitem=`), con consulta o fragmento, barra final y de la API no están definidas; ni si el proyecto de la URL debe coincidir. | A |
