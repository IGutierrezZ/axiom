# Revisión de cobertura del design — ui-integraciones

Revisión informativa (`_shared/sdd-coverage-review.md`): no bloquea fases ni archivo y no modifica el design. Dos revisores ciegos de solo lectura (A y B) sobre `proposal.md`, las deltas de `specs/` y `design.md`. El informe es la unión deduplicada de sus huecos.

## Ronda 1 — 2026-11-26

Entradas: `proposal.md`, `specs/local-web-dashboard/spec.md` (con la redacción de prov:WEB:9 ya ajustada: el alta con work item no sella `kickoff.yaml`), `specs/workspace-topology/spec.md` y `design.md`. Ambos revisores contrastaron el código (`internal/dashboard/server.go`, `service_branch.go`, `internal/changestart/adopt.go`, `internal/integrations`).

Validador de contrato de fase del design (contexto limpio): **PASS**. Nota menor: RF-WEB-07 se trata en D-07 y D-10, pero no se cita por su ID.

Resultado: 12 huecos tras deduplicar (A: 7, B: 7). Ninguno es de producto.

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | edge_case | design.md#D-02 | La lista cerrada de `Host` es ambigua. Incluye `[::1]`, que es inalcanzable porque el servidor solo escucha en `127.0.0.1`, y `net.SplitHostPort` lo devuelve sin corchetes. No dice nada de mayúsculas (`LOCALHOST`) ni del punto final (`localhost.`). Tampoco explica por qué se aparta de «el `Host` del propio servidor» que dice la spec. | A, B |
| 2 | edge_case | design.md#D-02 | La CSP propia de `/api/explainer` prevalece sobre la general y pierde `frame-ancestors 'none'`, `base-uri 'none'` y `form-action 'self'`. No se evalúa ese riesgo residual. Tampoco hay prueba de que el iframe del mismo origen siga cargando con `default-src 'self'`. | A, B |
| 3 | edge_case | design.md#D-02 | `GET /api/settings` empieza a devolver los valores de `integrations[<plugin>].<clave>`, pero queda fuera de la guarda de `Host` en GET. Por DNS rebinding seguiría siendo legible la misma configuración que `GET /api/integrations` protege. | A |
| 4 | edge_case | design.md#D-08 | Dar otra vez el mismo work item tras una recarga entra por la ruta con `ref`. Esa ruta hereda de `writeIncrement` el rechazo «ya existe como cambio activo» (`service_branch.go:82-84`), así que la reanudación prometida puede fallar. Además, la ruta no distingue una carpeta de este *staging* de un cambio activo ajeno con el mismo nombre. | B |
| 5 | error | design.md#D-06, D-08 | Falta el contrato de error de `POST /api/integrations/start` cuando ya no hay plugin activo, `WorkItemApplicability` es falso o el PAT ha desaparecido. Tampoco se dice cómo decide la SPA si mostrar el campo cuando falla `GET /api/integrations`. | B |
| 6 | error | design.md#D-04 | Sin puerto inyectado, no está definido si `GET /api/integrations` se degrada (con `workItems` no aplicable) o responde 503. Es justo la situación de las pruebas y del canario. | B |
| 7 | edge_case | design.md#D-05 | `ensureIncrementBranch` usa `context.Background()` (`service_branch.go:38`). El paso 3 del alta no queda acotado por el plazo ni se cancela, y mientras dura tiene tomado el cerrojo `TryLock` de `start` (409 `integration_busy` para reintentar y descartar). | B |
| 8 | edge_case | design.md#D-08 | Los motivos de `change_not_prepared` en la fase 2 (`head_other_branch`, `lane_mismatch`, `record_unreadable`), sobre todo en modo `branch` (en el sitio), no tienen mensaje ni prueba propios. Solo se trata `folder_missing`. | A |
| 9 | uncovered_requirement | design.md#D-07 | prov:WEB:8 pide «identificador, título si existe y enlace» del work item, pero el DTO no lleva título. `integrations.Link` no lo guarda y solo está en `context/work-item.json`, que es dato de terceros. Falta decidir si se omite, se lee saneado o se ajusta la spec. | A |
| 10 | edge_case | design.md#D-06 | `secrets.MaxValueBytes` vale 4096, pero el almacén de credenciales de Windows admite blobs más pequeños (unos 2,5 KB). Un PAT de tamaño intermedio falla como `500 secret_store_failed`, sin un mensaje que guíe a la persona. | A |
| 11 | edge_case | design.md#D-01 | `observeCssData` aplica `data-pct` y `data-ix-cols` con un `MutationObserver` asíncrono. Mientras no lo hace, las barras y el tablero pueden parpadear, y el CSS no tiene valor por defecto. Un nodo que el observador no vea rompe el tablero sin dar error. La comprobación en navegador es manual (H-05). | A |
| 12 | uncovered_requirement | design.md#Huecos de cobertura (H-20), Preguntas abiertas | H-20 y la pregunta abierta dicen que la spec habla de «sellado del kickoff». La spec ya está ajustada, así que ese texto ha quedado obsoleto. La consecuencia que sí sigue en pie es que un cambio creado desde el dashboard no tiene `kickoff.yaml`, y la pestaña y los actores deben aceptar que falte `SealedBy`. | B |

### Tratamiento

En `continuous` el padre no relanza `sdd-design` por un hueco. Los huecos 1, 4, 5, 6, 7, 8 y 12 son de implementación y están dentro del alcance del design (rutas y contratos ya definidos). Se pasan a `sdd-apply` como notas de implementación, sin cambiar el alcance. Los huecos 2, 3, 9, 10 y 11 se resuelven en apply si cuestan poco; si no, se declaran en el verify y en el resumen final.
