# Progreso de apply: ui-integraciones

Modo: estándar (sin strict_tdd), con RED→GREEN observado por la tarea. Entrega: `single-pr`, `commit_review: notify`.

## Tareas completadas

- [x] 1.1 Barrera de red en `internal/dashboard/server.go` (D-02).
- [x] 1.2 Retirar estilos y script de tema en línea (D-01).
- [x] 2.1 Ámbito `integrations[<plugin>].<clave>` en el catálogo de ajustes (D-03).
- [x] 3.1 `cli.IntegrationsGateway` y `secrets.MaxValueBytes` (D-04).
- [x] 3.2 Puerto y rutas del dashboard `/api/integrations*` (D-05, D-06).
- [x] 4.1 Pantalla «Integraciones» genérica (D-09 cliente, D-10).
- [x] 5.1 `OutboxView` y pestaña «Integraciones» del cambio (D-07, D-09).
- [x] 6.1 Alta desde un work item: servicio y rutas (D-08).
- [x] 6.2 Interfaz del alta con work item.
- [x] 7.1 Documentación de usuario del dashboard y guías.

## Evidencia de la unidad 1.1

| Evidencia | Valor |
|---|---|
| RED | `security_headers_test.go` escrito primero; sin `handler()` no compilaba; con `handler()` sobre el `withHeaders` antiguo fallaban 7 pruebas (cabeceras, CORS, OPTIONS, guarda de Host y Origin). |
| GREEN | `go test ./internal/dashboard -run 'TestSecurityHeaders\|TestNoAccessControl\|TestOptionsIs\|TestExplainer\|TestLocalGuard\|TestSettingsPutFromAForeignHost\|TestWorktreePrefsPutFromAForeignHost'`: todas en verde. |
| Regresión localizada | `go test ./internal/dashboard -run 'Security\|LocalGuard\|Options\|Explainer\|Canary\|Server\|Settings\|Marks\|Worktree\|NoAccess\|Restart\|Relaunch\|Update\|PWA\|Pwa\|Health\|Listen\|Port' -count=1`: ok (65 s). `go vet ./internal/dashboard`: limpio. |
| Arnés de ejecución | `httptest` sobre `handler()` (envoltorio completo). Sin binarios sueltos. |
| Límite de reversión | `internal/dashboard/server.go` y `internal/dashboard/security_headers_test.go`. |

## Decisiones y desviaciones

- Guarda de Host y Origin aplicada a todo `/api/*` con cualquier método (hueco 3 de la revisión de cobertura: `GET /api/settings` expondrá `integrations[...]`) y a cualquier método que mute en cualquier ruta; los assets estáticos en `GET`/`HEAD` quedan sin guarda. Las pruebas existentes usan `Router()` sin `withHeaders`, así que no se rompió ninguna.
- Lista de Host: `127.0.0.1`, `localhost` y `::1`, sin distinguir mayúsculas y con el punto final; `net.SplitHostPort` devuelve `[::1]` sin corchetes y `[::1]` sin puerto se recorta a mano. `Origin` se valida con `url.Parse`: esquema `http`, sin ruta, credenciales, consulta ni fragmento.
- `TestSecurityHeadersNoStoreOnIntegrationsRoutes` comprueba `no-store` solo en la respuesta 403 de la guarda: las rutas `/api/integrations*` no existen hasta 3.2 y el servidor de ficheros (Go) descarta `Cache-Control` en sus errores 404. Los handlers de 3.2 probarán `no-store` con Host local.
- `secrets_canary_test.go` no se cambia en esta tarea: sigue con `Router()`; el paso a `handler()` corresponde a 3.2.

## Riesgos residuales

- `/api/explainer` conserva su CSP exacta (RF-WEB-15), sin `frame-ancestors` ni `X-Frame-Options`: ese documento se enmarca en un iframe del mismo origen. Riesgo residual: otro origen podría enmarcarlo, pero el contenido es un HTML de solo lectura bajo `default-src 'none'`, y la guarda de Host/Origin se aplica igualmente a esa ruta.
- Hasta la tarea 1.2 la CSP `style-src 'self'` bloquea los estilos en línea de los assets actuales (la interfaz queda degradada visualmente entre 1.1 y 1.2).
- Un `Host` o `Origin` ajeno en `GET /api/*` ahora responde 403 también a clientes que antes funcionaban con cualquier Host (p. ej. un proxy inverso); el dashboard está pensado solo para bucle local.

## Evidencia de la unidad 1.2

| Evidencia | Valor |
|---|---|
| Pruebas nuevas | `TestNoInlineStyleOrScriptInAssets`, `TestThemeInitScriptIsExternalAndFirst`, `TestCssVarsModuleBoundsItsInput`; `TestIndexHTMLShell` ajustada al script externo. |
| GREEN | `go test ./internal/dashboard -run 'Assets|Inline|Theme|Workflow|Acceptance|Settings|Explainer|Security|PWA|Pwa|TestIndexHTMLShell|CssVars' -count=1`: ok; `go vet ./internal/dashboard` limpio. |
| Grep | Sin `style=`, `setAttribute('style'` ni `<script>` con cuerpo en `internal/dashboard/assets`. |
| Arnés | Sin ejecutor de JS: comprobación visual en navegador queda para `verify` (H-05). |

Decisiones: `applyCssData` se llama también de forma síncrona tras pintar el tablero de incrementos (hueco 11); `--ix-cols` tiene valor por defecto `repeat(auto-fit, minmax(9.25rem, 1fr))` y `.progress > span` parte de `width: 0`. Las barras de progreso en otras pantallas dependen del observador (microtarea, previa al pintado). RED no observado antes del GREEN (las pruebas se escribieron tras el cambio de assets).

## Evidencia de la unidad 2.1

| Evidencia | Valor |
|---|---|
| RED | Con el catálogo ya derivado, fallaron las pruebas de exclusión que pasan a inclusión (`TestCatalogReadOnlyAndExclusions`, `TestReadDoesNotWalkIntegrations`): el bloque `integrations` aparecía en el catálogo y un ancla en él bloqueaba la escritura. |
| GREEN | `go test ./internal/settings -count=1`: ok. `go test ./internal/dashboard -run 'Settings\|Ajustes\|Assets\|Canary' -count=1`: ok. `go test ./internal/integrations -run 'TestImportsDirection' -count=1`: ok (con subprueba nueva: `settings` solo importa la hoja `manifest`). `go test ./internal/workspace -run 'Integrations' -count=1`: ok. `go vet` de `settings` y `dashboard`: limpio. |
| Arnés | `settings.Write` sobre `t.TempDir()`; sin binarios sueltos ni ADO real. |

Decisiones: `keySpec.uiRequired` lleva la obligatoriedad del manifiesto solo para la interfaz (`required` queda a false para poder retirar el plugin entero); `pluginSpecs` y `findPluginSpec` se construyen al vuelo desde `manifest.Catalog()`/`Lookup`, sin tabla estática; un secreto del manifiesto da `unknown_key`. La pantalla de ajustes omite la sección `integrations` y enlaza a `#/integraciones` (la pantalla llega en 4.1); el control `int` usa `min`/`max` del catálogo (1 por defecto). Riesgo: un plugin con ancla, alias o `<<` en `integrations` ahora hace el documento no escribible (RF-TOP-22, intencionado).

## Evidencia de la unidad 3.1

| Evidencia | Valor |
|---|---|
| RED | `chain_test.go` (`MaxValueBytes`, `ValueTooLargeError` indefinidos: el paquete no compilaba) e `integrations_gateway_test.go` (`NewIntegrationsGateway` indefinido en `go vet ./internal/cli`) escritos antes del código. |
| GREEN | `go test ./internal/integrations/secrets -count=1`: ok. `go test ./internal/cli -run 'TestIntegrationsGateway\|TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign\|TestIntegrations' -count=1`: ok (38 s). `go test ./internal/integrations -run 'TestImportsDirection\|TestAdapterAssemblyImportRules' -count=1`: ok. `go vet` de `cli` y `secrets`: limpio. |
| Trinquete de código muerto | `bash scripts/deadcode-ratchet.sh` señala `NewIntegrationsGateway` y sus tres métodos como nuevos inalcanzables: la alcanzabilidad se cierra en 3.2 (`SetIntegrationsGateway` en `main.go`), cuando existe el puerto del dashboard. |
| Arnés | `adoEnv`/`adotest` y `newStartEnv`; sin ADO real ni binarios. |

Desviaciones: (1) `secrets.MaxValueBytes` vale 2048, no 4096 (hueco 10: el almacén de Windows admite unos 2,5 KB); el límite ya lo aplica `Chain.Set` con un `*ValueTooLargeError` tipado que nombra la variable de entorno y nunca lleva el valor, y `secret set` de la CLI usa la misma constante. (2) `WorkItemRefusal`, los resultados y métodos de work item (`StartWorkItem`, `WorkItemStaged`, `AdoptWorkItem`, `DiscardWorkItem`) se declaran en 6.1, que es quien los necesita; adelantarlos habría dejado más código inalcanzable. (3) Un fallo de `Set` del almacén por otra causa ya llega como `*KeyringUnavailableError` (tipado, con la variable de entorno), sin reflejar el valor.

## Evidencia de la unidad 3.2

| Evidencia | Valor |
|---|---|
| RED | `integrations_api_test.go` escrito antes del código: `go vet ./internal/dashboard` fallaba (`Server.integrationBudget` indefinido; rutas y servicio inexistentes). |
| GREEN | `go test ./internal/dashboard -run 'TestIntegrationsAPI\|Canary\|Security\|LocalGuard\|Settings\|TestEveryDashboardRouteIsClassified\|TestNoReadOnlyDashboardRoute' -count=1`: ok. `go vet ./internal/dashboard ./cmd/axiom`: limpio. `go test ./internal/cli -run 'TestIntegrationsGateway\|TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign' -count=1`: ok. |
| Trinquete de código muerto | `bash scripts/deadcode-ratchet.sh`: «no new unreachable functions» (`SetIntegrationsGateway(cli.NewIntegrationsGateway())` en `main.go` cierra la alcanzabilidad de 3.1). |
| Arnés | `handler()` con `Host: 127.0.0.1`, doble de `IntegrationsGateway`, `integrationstest.Install` y `AXIOM_STATE_DIR` temporal. Sin ADO real ni binarios. |

Notas de cobertura resueltas: hueco 6 (sin pasarela `GET /api/integrations` se degrada con `workItems` no aplicable; solo connect y discover responden 503 `integrations_gateway_missing`); hueco 10 (`*ValueTooLargeError` o longitud excesiva responde 400 `value_too_large` con `envVar` y guía; un fallo del almacén, 500 `secret_store_failed` con el valor redactado y la variable de entorno); `no-store` probado en respuestas locales de éxito y de error; el canario clasifica las cinco rutas, usa `handler()` con Host local y `TestIntegrationsAPINeverEchoesTheSecret` siembra el centinela en el almacén y en los errores del doble (respuestas, cabeceras y log).

Desviaciones: (1) códigos añadidos a la tabla de D-06: `value_too_large` (400), `port_unsupported` (422), `method_not_allowed` (405 tipado) y `settings_unreadable` (500). (2) Defensa en profundidad: los textos de error de la pasarela se redactan con los valores de secreto vigentes del plugin (`[redactado]`) aunque el plugin ya los sanee. (3) Plazos y cerrojo: `integrationBudget` (45 s) y un `TryLock` común a connect y discover en `Server`; el cerrojo de `start` llega en 6.1. (4) Los espacios laterales del PAT se recortan antes de guardar; los caracteres de control se rechazan sobre el valor recibido. (5) `Enabled` de un plugin = declarado y no `enabled: false` (igual que `workspace.Active`).

Riesgos: la ruta de guardado del PAT usa `secrets.Default()` en cada llamada (sin caché); un keyring bloqueado queda acotado a 5 s por petición.

## Evidencia de la unidad 4.1

| Evidencia | Valor |
|---|---|
| RED | `integrations_assets_test.go` escrito antes de los assets: `go test ./internal/dashboard -run TestIntegrationsScreenAssets` fallaba (módulos 404, ruta, navegación y `index.html` sin la pantalla). |
| GREEN | `go test ./internal/dashboard -run 'TestIntegrationsScreenAssets\|Assets\|Inline\|Theme\|Settings\|Router\|Shell\|TestJSModuleImportsResolve\|TestIndexHTMLShell' -count=1`: ok (incluye `TestNoInlineStyleOrScriptInAssets`). `go vet ./internal/dashboard`: limpio. |
| Arnés | Sin ejecutor de JS en las pruebas (H-05): invariantes fijados por texto. Comprobación puntual con Node en un directorio temporal: `node --check` de los módulos nuevos y `safeExternalURL` (acepta `https://a.b/x`; rechaza `javascript:`, `//h`, credenciales). Comprobación visual en navegador queda para `verify`. |

Decisiones: la pantalla reutiliza `controlHtml` y `collectChanges` con las entradas de cada plugin y guarda con un único `PUT /api/settings`; tras éxito recarga `GET /api/integrations`. Un 409 muestra `errorView` (con «Recargar»), que recarga con `keepTyped` y reaplica lo tecleado. El campo del PAT no lleva `data-key` (no entra en el lote), se vacía siempre tras guardar (también si falla) y el borrado pide `confirmDialog`. «Probar conexión» usa `candidates` (409/422) con radios cuando hay varios candidatos de identidad y reenvía `user`. «Descubrir» deja un `<datalist>` solo en campos de texto libre sin `format` ni `enum` y marca «no encontrado en el tracker» los valores ausentes; las sugerencias se recuerdan solo en memoria. `safeExternalURL`/`externalLink` viven en `core/integrations.js`, listos para 5.1 (la pantalla de ajustes ya enlazaba a `#/integraciones`).

Riesgos: la pantalla no se ha visto en un navegador real; `externalLink` aún no tiene consumidor hasta 5.1 (JS, no afecta al trinquete de Go); el icono de navegación reutiliza `handoff`.

## Evidencia de la unidad 5.1

| Evidencia | Valor |
|---|---|
| RED | `outbox_view_test.go` escrito antes de `OutboxView`: `go test ./internal/integrations -run TestOutboxView` no compilaba (`OutboxView` y `OperationView` indefinidos). Las pruebas del dashboard (`change_integrations_test.go`) se escribieron justo después del código del servicio y pasaron a la primera: RED no observado para esa mitad. |
| GREEN | `go test ./internal/integrations -run 'TestOutboxView\|TestImportsDirection' -count=1`: ok. `go test ./internal/dashboard -run 'TestChangeIntegrations\|TestSafeExternalURL\|TestIntegrations\|Canary\|Detail\|Detalle\|Assets\|Inline\|Router\|TestJSModuleImportsResolve\|TestEveryDashboardRouteIsClassified\|TestNoReadOnlyDashboardRoute\|TestIndexHTMLShell' -count=1`: ok. `go vet ./internal/dashboard ./internal/integrations`: limpio. |
| Trinquete de código muerto | `bash scripts/deadcode-ratchet.sh`: «no new unreachable functions» (`OutboxView` se alcanza desde `handleChangeIntegrations`). |
| Arnés | `handler()` con `Host: 127.0.0.1`, `embeddedProject` y ficheros YAML reales del cambio; sin ADO real ni binarios. Sin ejecutor de JS: la pestaña se fija por texto (`TestChangeIntegrationsTabAssets`); comprobación visual para `verify`. |

Notas de cobertura: hueco 9, el título del work item NO se muestra (`context/work-item.json` solo guarda datos de confianza: id, URL, plugin; el título viviría en el markdown no confiable), así que se muestra identificador, plugin y enlace («si existe» se cumple por omisión). Hueco 12, la pestaña tolera la ausencia de `kickoff.yaml` (`sealed: false`, «sin sellar») y de `sealed_by` o `actor` («sin registrar»), sin error.

Decisiones: la ruta es `GET /api/integrations/change?name=` con `no-store`, clasificada en el canario con un cambio `x` que ya tiene outbox; `safeExternalURL` (Go) vive en `service_change_integrations.go` y `TestSafeExternalURL` en `change_integrations_test.go` (6.1 la reutiliza). Los errores parciales son frases fijas (sin rutas ni texto del YAML). Las decisiones de puerta se listan todas, con tope de 100. Nombre inválido: 400 `invalid_change_name`; no localizado: 404 `change_not_found`.

Riesgos: la pestaña no se ha visto en un navegador real. `deadcode-ratchet` avisa de que una entrada de la línea base ya es alcanzable (previa a esta tarea; no se actualiza la línea base aquí).

## Evidencia de la unidad 6.1

| Evidencia | Valor |
|---|---|
| RED | `integrations_gateway_test.go` (`WorkItemRefusal` indefinido: `go vet ./internal/cli` no compilaba) y `workitem_start_test.go` (`Server.workItemBudget` y el puerto sin métodos de work item: `go vet ./internal/dashboard` no compilaba) escritos antes del código. Una primera pasada de GREEN fallaba 2 pruebas por mi arnés (el estado del hub se aislaba después de preparar el proyecto); corregido en la prueba. |
| GREEN | `go test ./internal/dashboard -run 'TestWorkItemStart\|TestSafeExternalURL\|TestIntegrationsAPI\|Canary\|CreateIncrement\|Increment' -count=1`: ok (91 s). `go test ./internal/cli -run 'TestIntegrationsGateway\|TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign\|TestChangeStart' -count=1`: ok. `go test ./internal/integrations -run TestImportsDirection -count=1`: ok. `go vet ./internal/dashboard ./internal/cli ./cmd/axiom`: limpio. |
| Trinquete de código muerto | `bash scripts/deadcode-ratchet.sh`: «no new unreachable functions» (aviso previo de una entrada de la línea base ya alcanzable). |
| Arnés | `handler()` con `Host: 127.0.0.1`, git real con `ensure` real (`embeddedProject`), doble de `IntegrationsGateway` para las fases 1 y 2 y servidor doble de ADO (`adotest`) en las pruebas de la pasarela. Sin ADO real ni binarios. |

Notas de cobertura resueltas: hueco 4 (reanudar con el mismo work item o por `change` reutiliza la carpeta del intento fallido y no pisa un `proposal.md` existente; un cambio activo o archivado ajeno con el mismo nombre derivado responde 409 `change_exists`, retira el staging recién creado por esa petición y no crea rama), hueco 5 (`work_items_not_applicable` 422 sin plugin aplicable y `secret_missing` 422 sin PAT, sin llamar al tracker ni crear nada; 503 sin pasarela), hueco 7 (`ensureIncrementBranch(ctx, …)` recibe el contexto con plazo de la petición; el alta manual sigue con `context.Background()`), hueco 8 (`change_not_prepared` conserva `reason` y su mensaje propio: `head_other_branch`, `folder_missing`, `lane_mismatch`, `record_unreadable`; la pasarela lo prueba con `no_record` real).

Desviaciones: (1) los tipos y métodos de work item de la pasarela viven en `internal/cli/integrations_gateway_workitem.go` (el diseño decía `integrations_gateway.go`), sin literales de rechazo nuevos: reutilizan `changeStartRefusalText`. (2) `proposal.md` se escribe antes de la fase 2 (el diseño la ponía después): así un reintento por `change` nunca depende de un staging ya retirado y la fase 2 solo exige la carpeta. (3) `WorkItemStaged` devuelve `{Change, ExternalRef}` y lee el staging con `changestart.WorkItemFacts`; el dashboard valida el nombre con `changestart.ValidChangeName` directamente. (4) Códigos añadidos a D-08: `change_exists` (409), `invalid_change_name` (422), `ensure_failed` y `folder_write_failed` (500), `work_items_not_applicable` (422). `Fields` del rechazo expone solo `reason`, `type`, `tag`, `change`, `lane` y `record_lane` (nunca `detail`). (5) Un único `workItemLock` para alta y descarte; presupuesto 120 s en `Server.workItemBudget`.

Riesgos: la ruta de multirepo usa `filepath.Dir(planningHome)` como `cwd` de la fase 2, según el diseño, sin prueba con git real en multirepo; `already_adopted` reutiliza la carpeta y la fase 2 es idempotente, pero no se ejercita contra un cambio ya adoptado de verdad.

## Evidencia de la unidad 6.2

| Evidencia | Valor |
|---|---|
| RED | `TestIntegrationsScreenAssetsWorkItemStartForm` escrita antes de tocar `incrementos.js`: `go test ./internal/dashboard -run TestIntegrationsScreenAssetsWorkItemStartForm` falló (faltaban `/api/integrations/start`, `confirm: true`, `confirmDialog`, `startFromWorkItem`, `workItemsApplicable`, `Descartar`). |
| GREEN | `go test ./internal/dashboard -run 'TestIntegrationsScreenAssets\|Assets\|Inline\|Increment\|Incrementos\|Branches\|TestJSModuleImportsResolve\|TestIndexHTMLShell\|TestNoInlineStyle' -count=1`: ok; `go vet ./internal/dashboard`: ok; `node --check` de `incrementos.js`, `branches.js` e `integrations.js` (copias en un directorio temporal): ok. |
| Arnés | Sin ejecutor de JS en las pruebas: invariantes fijados por texto. Comprobación visual pendiente para `verify`. |

Implementación: `core/branches.js` añade `startFromWorkItem` (un 409 `decision_required` reutiliza el selector de modo y repite con `{change, mode, intent, type}` sin repetir la fase 1) y `WorkItemStartError` con `change` y `actions`; `core/integrations.js` añade `workItemsApplicable`; `index.html` añade el campo, el aviso y los botones Reintentar y Descartar; `incrementos.js` muestra el campo solo si `GET /api/integrations` dice `workItems.applicable` (si falla, el alta manual queda igual), pinta mensajes, avisos y continuaciones con `textContent`, y descarta con `confirmDialog` y `confirm: true`.

Decisiones: con work item, el campo de nombre se ignora (el cambio se deriva del work item); las advertencias del éxito se muestran en el diálogo y el botón pasa a «Abrir el cambio». Cancelar la decisión de modo ofrece reintentar o descartar.

Riesgos: la interfaz no se ha visto en un navegador real; el reintento reenvía la intención y el tipo del formulario.

## Evidencia de la unidad 7.1

| Evidencia | Valor |
|---|---|
| Documentación | `docs/usage.md` (nueva sección «Integraciones desde el dashboard», guarda de Host y Origin ampliada a todo `/api/*`, CORS retirado, ajustes sin bloque `integrations`), `docs/codebase/dashboard.md` (barrera de red, CSP, PAT, no sellado de `kickoff.yaml`) y `docs/manual-de-inicio.md` (viñeta del dashboard). |
| Pruebas | `go test ./internal/workspace -run TestDocs`: ver resultado del commit; sin otras pruebas de docs afectadas. |

Desviaciones: no existe CHANGELOG ni guía propia del dashboard para integraciones; se actualizó solo documentación existente. `docs/codebase/integrations.md` trata de integraciones de agentes y no se toca. No se tocaron goldens, skills, `docs/plan/` ni la spec viva.
