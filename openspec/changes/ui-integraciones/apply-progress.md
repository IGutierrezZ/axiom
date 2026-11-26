# Progreso de apply: ui-integraciones

Modo: estándar (sin strict_tdd), con RED→GREEN observado por la tarea. Entrega: `single-pr`, `commit_review: notify`.

## Tareas completadas

- [x] 1.1 Barrera de red en `internal/dashboard/server.go` (D-02).
- [x] 1.2 Retirar estilos y script de tema en línea (D-01).
- [x] 2.1 Ámbito `integrations[<plugin>].<clave>` en el catálogo de ajustes (D-03).
- [x] 3.1 `cli.IntegrationsGateway` y `secrets.MaxValueBytes` (D-04).
- [x] 3.2 Puerto y rutas del dashboard `/api/integrations*` (D-05, D-06).

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
