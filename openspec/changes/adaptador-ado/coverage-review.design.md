# Revisión de cobertura del diseño — adaptador-ado

Revisión informativa: no bloquea ninguna fase ni el archivo, y no modifica la spec ni el diseño. La hicieron en paralelo los revisores ciegos `jd-judge-a` (A) y `jd-judge-b` (B). El informe es la unión deduplicada de sus huecos.

## Ronda 1 — 2026-11-22

- Entradas: `proposal.md`, las tres specs delta (incluida la corrección de REQ-43.18 y del scope `vso.work` de las iteraciones) y `design.md`.
- Los dos revisores devolvieron un resultado utilizable.

| # | Tipo | Ubicación | Hueco | Revisor |
|---|---|---|---|---|
| 1 | uncovered_requirement | design D-05, §10 | El diseño asigna `vso.project` a las iteraciones; REQ-45.5 (corregido) exige `vso.work`. El texto de `AuthError`, el secreto `pat` y el manual enumeran solo tres scopes. | A, B |
| 2 | uncovered_requirement | spec REQ-45.5 frente a REQ-43.21 | Incoherencia dentro de la spec: REQ-45.5 nombra cuatro scopes y MODIFIED REQ-43.21 pide documentar tres (`vso.work_write`, `vso.project`, `vso.code`). | A |
| 3 | uncovered_requirement | design §7, §10 | El diseño no traza MODIFIED REQ-43.18 (cabecera, tabla de trazabilidad, unidad 6, pruebas), y su pregunta abierta, que lo da por ausente, está obsoleta. | A, B |
| 4 | edge_case | design D-10 | «Subcomando desconocido» (REQ-43.18) exige mostrar la ayuda de la familia con los cinco verbos. El diseño solo amplía la línea de error de `integrations.go:147`. | A, B |
| 5 | error | design D-10 | Plugin desactivado, bloque ausente o configuración errónea dan mensajes de texto sin centinela ni tipo, aunque REQ-45.12 pide un error tipado. | B |
| 6 | edge_case | design D-05 | No hay fila para un 200 con cuerpo no JSON que no sea `text/html`. Falta el criterio que distingue la página de inicio de sesión de un JSON inválido. | A |
| 7 | edge_case | design D-08, D-12 | La paginación de miembros se corta en silencio a las 1.000 páginas, o ante una página sin ids nuevos, y entrega un resultado parcial como completo. | A, B |
| 8 | edge_case | design D-03 | `url.PathEscape` no escapa `.` ni `..`, así que una organización, proyecto o equipo `..` podría sacar la ruta de `dev.azure.com/{organization}`. | A, B |
| 9 | uncovered_requirement | design D-03, D-07 | Ninguna API pública construye un `Client` con `SecretSource` inyectable. Sin ella no se pueden escribir algunos escenarios desde `azuredevops_test`, ni se ve cómo lo obtendrán C-07-d y C-07-e. | A |
| 10 | edge_case | design D-11 | Las tres claves van con `Required: true`, algo que la spec no pide. Falta el caso de un bloque sin `team` o con `enabled: false`, y la frontera entre el error de carga y `ErrConfig`. | A |
| 11 | edge_case | design §5 | La prueba del manual no fija «PAT de una sola organización» ni «secretos nunca en `axiom.yaml`» (REQ-43.21). Tampoco hay comprobación de que el código y los fixtures no lleven correos (REQ-45.9) ni de «ningún otro origen es consultado» (REQ-45.15). | A, B |
| 12 | edge_case | design §5, D-09 | La prueba de escritura atómica falla antes de `filemerge`, así que no ejercita «no queda ningún temporal» (REQ-45.11). | B |
| 13 | edge_case | design D-10 | El JSON de `connect` nombra la organización como `account`, y la spec habla de «organización». | B |
| 14 | edge_case | design D-03, §5 | REQ-45.16: no hay ninguna guarda que impida que una prueba de CLI sin `adotest.Install` llame a `dev.azure.com` real. | B |

Estado: 14 huecos abiertos. En `continuous` no se relanza `sdd-design`. Los que son exigencias de la spec (#1, #3-#5, #7, #8, #11-#14) se trasladan al apply, porque la spec prevalece sobre el diseño. El resto va al resumen final.
