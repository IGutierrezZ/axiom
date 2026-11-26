# Progreso de apply: ui-integraciones

Modo: estándar (sin strict_tdd), con RED→GREEN observado por la tarea. Entrega: `single-pr`, `commit_review: notify`.

## Tareas completadas

- [x] 1.1 Barrera de red en `internal/dashboard/server.go` (D-02).
- [x] 1.2 Retirar estilos y script de tema en línea (D-01).

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
