# ODD: `axiom ui` instalable como PWA, con puerto fijo y arranque al clicar

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/ui-pwa-puerto-fijo.md`.
> Espejo de recuperación en Engram: topic `odd/ui-pwa-puerto-fijo/tasks`, proyecto `axiom`.
> Rama: `inc/ui-pwa-puerto-fijo` (worktree `axiom-wt/ui-pwa-puerto-fijo`).

## Objetivo

Que el dashboard de `axiom ui` se pueda instalar como aplicación (PWA) desde Edge o Chrome, quede anclado a la barra de tareas de Windows y se abra con un clic la siguiente vez, aunque el servidor no esté en marcha.

## Problema y porqué

- El dashboard no sirve manifiesto ni service worker, así que el navegador no ofrece instalarlo.
- El puerto por defecto es `8080`, muy ocupado, y `ListenAndServe` reasigna en silencio al siguiente libre (`internal/dashboard/server.go`, `findAvailablePort`). Una PWA instalada queda ligada a su origen exacto: si el puerto cambia, el icono deja de funcionar.
- Una PWA no arranca procesos locales: si `axiom ui` no está corriendo, el icono muestra "no se puede conectar". Hace falta un acceso directo que levante el servidor, o un arranque al iniciar sesión.

## Decisiones

- **Puerto fijo por defecto `47700`**: dentro del rango registrado (1024–49151), fuera del rango dinámico de Windows (49152–65535) y de reservas habituales de Hyper-V. `--port` sigue permitiendo cambiarlo.
- **Sin reasignación silenciosa.** Si el puerto está ocupado: si responde un Axiom (`GET /api/health`), se abre la ventana y se sale con éxito; si es otra aplicación, error claro que sugiere `--port`.
- **`axiom ui` idempotente**: ejecutarlo varias veces nunca levanta dos servidores.
- **Arranque al clicar por defecto**; arranque con Windows como opción del instalador. Ambos conviven con la PWA.
- **Sin endpoint de apagado**: la API tiene CORS `*`, y un `POST /api/shutdown` permitiría a cualquier web local tumbar el servidor. El servidor desacoplado vive hasta cerrar sesión.
- **Service worker sin caché** (solo red): evita servir assets obsoletos tras actualizar Axiom.
- **Iconos generados en Go** en tiempo de ejecución a partir del logo triangular del dashboard: sin binarios en el repo ni herramientas de rasterizado.
- **Instalador**: solo `scripts/install.ps1` (Windows). `install.sh` queda fuera de alcance.

## Alcance autorizado

`cmd/axiom` (`runUI`, `openBrowser` y ficheros auxiliares), `internal/dashboard` (servidor, assets, tests), `scripts/install.ps1` y la documentación que menciona `axiom ui` o su puerto.

## Restricciones

- No compilar binarios sueltos (`go build -o`, `go run`): Cortex XDR los borra. Verificación solo con `go test` y `go vet`.
- Respetar los tests de assets: sin recursos externos, sin emojis, orden de `<head>` de `TestIndexHTMLShell`.
- TDD: desactivado (`openspec/config.yaml`, `strict_tdd: false`). Runner: `go test ./...`. Comprobaciones funcionales: `go test ./internal/dashboard/... ./cmd/axiom/...` y `go vet ./...`.
- Revisión por recibos (RDD): desactivada (`axiom review mode status`: off, por defecto).

## Tareas

- [x] **T1 · Puerto fijo, `/api/health` e idempotencia** — ruta: delegada (escritor único; toca `server.go`, `main.go` y tests, 2+ ficheros no triviales).
  - `GET /api/health` → `{"service":"axiom-dashboard", ...}`.
  - Puerto por defecto `47700`; retirar la reasignación silenciosa y adaptar `TestPortFallback`.
  - `runUI`: sondeo previo de health; Axiom ya activo → abrir navegador y salir 0; puerto ajeno → error claro.
- [x] **T2 · PWA: manifiesto, iconos, service worker y botón de instalar** — ruta: delegada.
  - `manifest.webmanifest` (`display: standalone`, `start_url: /`, colores del tema), tipo MIME `application/manifest+json`.
  - Iconos PNG 192/512 y maskable generados en Go; favicon.
  - `sw.js` en la raíz, solo red; registro desde `main.js`.
  - Botón "Instalar aplicación" con `beforeinstallprompt` cuando el navegador lo ofrezca.
- [ ] **T3 · Ventana de aplicación y arranque desacoplado** — ruta: delegada.
  - `axiom ui --app`: abre Edge o Chrome con `--app=<url>`; si no hay ninguno, navegador por defecto.
  - `axiom ui --detach`: lanza el servidor en segundo plano sin consola, espera a `/api/health`, abre la ventana y sale.
- [ ] **T4 · Instalador de Windows y documentación** — ruta: delegada.
  - `install.ps1`: acceso directo en el menú Inicio "Axiom" → `axiom.exe ui --detach --app` (minimizado); opción `-Autostart` que lo copia a la carpeta de inicio con `--no-browser`.
  - Actualizar README, `docs/quickstart.md`, `docs/manual-de-inicio.md` (corrige la mención obsoleta a `4040`), `docs/usage.md`, `docs/codebase/dashboard.md`.

## Criterios de aceptación

- Con el servidor parado, el acceso directo levanta el servidor y abre la ventana; con el servidor activo, solo abre la ventana.
- Edge y Chrome muestran la opción de instalar la aplicación en `http://127.0.0.1:47700`.
- Un puerto ocupado por otra aplicación produce un error explícito, nunca otro puerto.
- `go test ./internal/dashboard/... ./cmd/axiom/...` y `go vet ./...` en verde.

## Entrega

- Previsión: ~700 líneas de autoría (más de 400). Estrategia: `exception-ok` (decisión del usuario, 2026-11-05): una sola PR con excepción de tamaño, un commit por tarea.

## Progreso y evidencias

### T1

- Nuevo `internal/dashboard/health.go`: `DefaultPort = 47700`, `ServiceID`, `ErrPortInUse`, `GET /api/health` (solo GET) y `ProbeAxiom` (timeout 800 ms). `ListenAndServe` ya no reasigna; detecta puerto ocupado también por los códigos Winsock 10048/10013 (en Windows el error no coincide con `EADDRINUSE` y su texto está localizado). Los puertos reservados (10013) se tratan como ocupados.
- `runUI`: sondeo previo; Axiom activo → abre la ventana y sale 0; puerto ajeno → error con `--port`.
- Tests: `TestListenAndServeNoReassign` (sustituye a `TestPortFallback`), `TestHealthEndpoint`, `TestDefaultPort`, `TestProbeAxiom`.
- Verificación del escritor: `go test ./internal/dashboard/... ./cmd/axiom/...` → ok; `go vet` → limpio; `gofmt -l` → vacío. Comprobación del orquestador: `go test ./internal/dashboard/... -run 'TestHealth|TestProbe|TestListenAndServeNoReassign|TestDefaultPort' -count=1` → ok.
- Riesgo (`axiom review assess`): `medium`; RDD desactivado → autoverificación del escritor más comprobación del orquestador.
- Commit: `ab5c582a`.

### T2

- Nuevos: `assets/manifest.webmanifest`, `assets/icons/icon.svg`, `assets/sw.js` (solo red: `fetch` vacío sin `respondWith`), `assets/js/core/pwa.js` (registro del SW y botón `#btn-install-app` con `beforeinstallprompt`), `pwa_icons.go` (PNG 192/512/maskable rasterizados en Go con supersampling 4x4, generados una vez), `pwa_test.go`. Cambios en `index.html`, `main.js` y `server.go` (MIME `.webmanifest` y rutas `/icons/*.png`).
- Verificación del escritor: `go test ./internal/dashboard/... -count=1` → ok; `go vet` → limpio; `gofmt -l` → vacío.
- Comprobación del orquestador con el arnés `TestDashboardPreview` en el puerto 47711 y el navegador integrado: `/api/health` → 200; manifiesto `standalone`; los 4 iconos → 200 con su tipo; icono 512 revisado visualmente; `/sw.js` → 200 `text/javascript`.
- **Pendiente**: el navegador integrado no registra service workers (devuelve el mismo "unknown error" con un script inexistente), así que la instalabilidad real en Edge/Chrome queda para la prueba manual final.

## Siguiente paso

T3.
