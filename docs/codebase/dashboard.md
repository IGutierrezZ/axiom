# Dashboard

[Back to Codebase Guide](../CODEBASE-GUIDE.md)

El panel web local (`axiom ui`) vive en `internal/dashboard/` (servidor HTTP, API y assets embebidos) y se lanza desde `runUI` en `cmd/axiom/main.go`. Esta página documenta solo el arranque, el puerto y la instalación como PWA; el contrato de cada endpoint está en `internal/dashboard/server.go`.

## Servidor y puerto

- Escucha siempre en `127.0.0.1:47700` (`dashboard.DefaultPort`). `--port` lo cambia.
- No hay reasignación silenciosa: si el puerto está ocupado por otra aplicación, `axiom ui` falla con un error que sugiere `--port`.
- `GET /api/health` devuelve `{"service":"axiom-dashboard", ...}` (`internal/dashboard/health.go`). `ProbeAxiom` lo usa para decidir si ya hay un Axiom activo: en ese caso `axiom ui` solo abre la ventana, de modo que ejecutarlo varias veces nunca levanta dos servidores.
- No existe endpoint de apagado: aunque la API ya no acepta CORS, un endpoint así seguiría siendo una acción de administración sin autenticación.
- **Barrera de red** (`withHeaders` en `server.go`, accesible en las pruebas con `handler()`): toda respuesta lleva una CSP general (`default-src 'self'`, `style-src 'self'`, sin estilos ni scripts en línea), `nosniff` y `Referrer-Policy: no-referrer`; `/api/explainer` conserva su propia CSP, una sola vez. No se emite ninguna cabecera `Access-Control-*` y `OPTIONS` responde `405`. Los assets no pueden llevar `style=` ni `<script>` con cuerpo: el tema inicial vive en `assets/js/theme-init.js` y los valores dinámicos se pasan con `data-*` y `core/cssvars.js`.

## PWA

El dashboard es instalable desde Edge o Chrome:

| Pieza | Ubicación |
|---|---|
| Manifiesto (`display: standalone`, `start_url: /`) | `internal/dashboard/assets/manifest.webmanifest` (MIME `application/manifest+json`) |
| Service worker, solo red (sin caché, para no servir assets obsoletos tras actualizar) | `internal/dashboard/assets/sw.js`, servido en la raíz |
| Registro del SW y botón "Instalar aplicación" (`beforeinstallprompt`) | `internal/dashboard/assets/js/core/pwa.js` |
| Iconos PNG 192/512 y maskable, generados en Go a partir del logo | `internal/dashboard/pwa_icons.go` (rutas `/icons/*.png`) |

Una PWA instalada queda ligada a su origen exacto, por eso el puerto es fijo. Tampoco arranca procesos: el servidor debe estar activo.

## Arranque desacoplado

En `cmd/axiom/ui_launch.go`:

- `--detach`: relanza `axiom ui --no-browser` como proceso hijo sin consola (`detach_windows.go`: `CREATE_NO_WINDOW`; `detach_other.go`: `Setsid`), registra en `<UserCacheDir>/axiom/ui.log`, espera a `/api/health` (hasta 10 s) y sale. Sin `axiom.yaml` en el directorio actual, usa el workspace activo del hub.
- `scripts/install.ps1` crea el acceso directo `Axiom` del menú Inicio (`ui --detach`) y registra la tarea programada de usuario `Axiom` (al iniciar sesión, `ui --detach --no-browser`); `-NoAutostart` y `-NoShortcut` los omiten.

Guía de usuario: [usage.md](../usage.md#ui).

## Dashboard invariants

- **No unauthenticated admin actions** más allá del bucle local: el servidor solo escucha en `127.0.0.1`.
- **No full API duplication**: enlaza al código del endpoint cuando exista.
- **No direct local DB assumptions**: no leer `.engram/engram.db` desde el dashboard salvo decisión explícita de arquitectura.
- **No silent cloud coupling**: el estado remoto va detrás de una interfaz de transporte/cliente.
- **Assets sin recursos externos** ni emojis (lo vigilan los tests de `internal/dashboard`).
- **`axiom.yaml` solo se escribe por `PUT /api/settings`**, mediante `internal/settings` (catálogo, escritor de round-trip y revisión). Ninguna otra ruta modifica `axiom.yaml`, y `GET /api/workspace` sigue siendo solo lectura. `internal/dashboard/settings_api.go` solo traduce entre HTTP y ese paquete.
- **Guarda local de Host y Origin** en `withHeaders`, antes de cualquier handler y de leer el cuerpo: todo `/api/*` con cualquier método y cualquier método que mute en cualquier ruta exigen `Host` en `127.0.0.1`, `localhost` o `::1` (con el puerto enlazado) y, si hay `Origin`, el mismo origen local; si no, `403 forbidden_origin`. Un proxy inverso o un cliente de otra máquina quedan fuera por diseño. Solo los assets estáticos en `GET`/`HEAD` quedan sin guarda. Una ruta nueva se registra en `server.go` con literal de cadena y se clasifica en `secrets_canary_test.go`.
- **El PAT nunca sale**: `/api/integrations*` (estado, secreto, `connect`, `discover`, `change`, `start`) usa `no-store`, y el PAT solo entra por `POST` (límite `secrets.MaxValueBytes`, 2048 bytes). Las respuestas solo informan del `source` (`keyring`, `env` o `missing`), y una variable de entorno `AXIOM_INTEGRATION_<PLUGIN>_<SECRETO>` lo tapa (`shadowedByEnv`). En Linux no hay almacén del sistema: solo vale la variable de entorno. El canario de secretos siembra un centinela y comprueba que ninguna respuesta, cabecera ni log lo contiene.
- **El dashboard no sella `kickoff.yaml`**: el alta desde un work item (`POST /api/integrations/start`) ejecuta las fases 1 y 2 de `axiom change start` y prepara rama y carpeta, pero el sellado del kickoff se hace al retomar el cambio con el orquestador.

## Navigation

Previous: [Sync and cloud](sync-and-cloud.md) | Next: [Integrations](integrations.md)
