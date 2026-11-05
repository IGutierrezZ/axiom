# Dashboard

[Back to Codebase Guide](../CODEBASE-GUIDE.md)

El panel web local (`axiom ui`) vive en `internal/dashboard/` (servidor HTTP, API y assets embebidos) y se lanza desde `runUI` en `cmd/axiom/main.go`. Esta página documenta solo el arranque, el puerto y la instalación como PWA; el contrato de cada endpoint está en `internal/dashboard/server.go`.

## Servidor y puerto

- Escucha siempre en `127.0.0.1:47700` (`dashboard.DefaultPort`). `--port` lo cambia.
- No hay reasignación silenciosa: si el puerto está ocupado por otra aplicación, `axiom ui` falla con un error que sugiere `--port`.
- `GET /api/health` devuelve `{"service":"axiom-dashboard", ...}` (`internal/dashboard/health.go`). `ProbeAxiom` lo usa para decidir si ya hay un Axiom activo: en ese caso `axiom ui` solo abre la ventana, de modo que ejecutarlo varias veces nunca levanta dos servidores.
- No existe endpoint de apagado: la API acepta CORS `*` y permitiría a cualquier web local parar el servidor.

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

## Navigation

Previous: [Sync and cloud](sync-and-cloud.md) | Next: [Integrations](integrations.md)
