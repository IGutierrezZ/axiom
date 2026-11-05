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
- [x] **T3 · Ventana de aplicación y arranque desacoplado** — ruta: delegada.
  - `axiom ui --app`: abre Edge o Chrome con `--app=<url>`; si no hay ninguno, navegador por defecto.
  - `axiom ui --detach`: lanza el servidor en segundo plano sin consola, espera a `/api/health`, abre la ventana y sale.
- [x] **T4 · Instalador de Windows y documentación** — ruta: delegada.
  - `install.ps1`: acceso directo en el menú Inicio "Axiom" → `axiom.exe ui --detach --app` (minimizado); opción `-Autostart` que lo copia a la carpeta de inicio con `--no-browser`.
  - Actualizar README, `docs/quickstart.md`, `docs/manual-de-inicio.md` (corrige la mención obsoleta a `4040`), `docs/usage.md`, `docs/codebase/dashboard.md`.

- [x] **T5 · Alinear con el modelo de work-copilot: PWA instalada + servidor al iniciar sesión** — ruta: delegada (toca `cmd/axiom`, `install.ps1`, tests y docs). Reabre parte de T3 y T4 tras la prueba manual del usuario.
  - Motivo: `axiom ui --detach --app` abre `msedge --app=<url>`, que es una ventana de Edge y Windows le asigna el icono de Edge en la barra. La PWA instalada desde Chrome sí mantiene su icono (comprobado por el usuario el 2026-11-05). work-copilot no lanza el navegador: el usuario instala la PWA y una tarea programada al iniciar sesión mantiene el servidor arriba.
  - Retirar `--app` (`resolveAppBrowser`, `openAppWindow` y sus tests).
  - `install.ps1`: arranque al iniciar sesión **por defecto** con una tarea programada de usuario (`AtLogOn`, `RunLevel Limited`, reintentos, `MultipleInstances IgnoreNew`) que ejecuta `axiom ui --detach --no-browser`; `-NoAutostart` lo desactiva; se retira `-Autostart` y el acceso directo de la carpeta de inicio.
  - Acceso directo del menú Inicio: `axiom ui --detach` (pestaña normal, desde donde se instala la PWA la primera vez).
  - Docs: el camino recomendado es instalar la PWA desde el navegador, anclar ese icono y dejar el servidor arrancando con Windows; cómo quitar la tarea.

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
- Commit: `b4baca95`.
- **Pendiente**: el navegador integrado no registra service workers (devuelve el mismo "unknown error" con un script inexistente), así que la instalabilidad real en Edge/Chrome queda para la prueba manual final.

### T3

- Nuevos en `cmd/axiom`: `ui_launch.go` (`resolveAppBrowser`, `openAppWindow`, `openUIWindow`, `detachedChildArgs`, `waitForAxiom`, `portIsFree`, `uiLogPath`, `startDetachedServer`, `runUIDetached`), `detach_windows.go` (`CREATE_NO_WINDOW | CREATE_NEW_PROCESS_GROUP`, `HideWindow`; se prefiere a `DETACHED_PROCESS` porque los descendientes heredan la consola oculta y ninguno abre una visible), `detach_other.go` (`Setsid`), `ui_launch_test.go`. `runUI` gana `--app` y `--detach`.
- `--detach` comprueba antes el puerto con `net.Listen` (error inmediato si lo ocupa otra aplicación), lanza el hijo con `ui --no-browser --port --path`, registra en `<UserCacheDir>/axiom/ui.log` y espera `/api/health` hasta 10 s.
- Verificación del escritor: `go test ./cmd/axiom/... -count=1` → ok; `go vet` (windows, `GOOS=linux`, `GOOS=darwin`) → limpio; `gofmt -l` → vacío. Comprobación del orquestador: `go test ./cmd/axiom/ -run 'ResolveAppBrowser|DetachedChildArgs|WaitForAxiom'` → 4 PASS.
- **Pendiente**: el lanzamiento real del hijo y de la ventana no se puede ejecutar aquí (prohibido ejecutar `axiom.exe` recién compilado por el antivirus); queda para la prueba manual. Commit: `b3ca5636`. `ui.log` se abre en modo *append* sin rotación: aceptable por ahora.

### T4

- `scripts/install.ps1`: `-Autostart` y `-NoShortcut`; `Test-Installation` devuelve la ruta del binario; `New-AxiomShortcuts` crea `Axiom.lnk` en el menú Inicio (`ui --detach --app`, minimizado) y, con `-Autostart`, en la carpeta de inicio (`ui --detach --no-browser`). Cualquier fallo de accesos directos solo avisa. `Show-NextSteps` explica anclar a la barra e instalar como app.
- Test nuevo: `internal/update/install_script_test.go` `TestWindowsInstallScriptCreatesDashboardShortcuts`.
- Docs: `docs/usage.md` (sección `ui` y "Usar `axiom ui` como aplicación de escritorio"), `docs/manual-de-inicio.md` (corrige `localhost:4040` → `127.0.0.1:47700`), `README.md`, `docs/quickstart.md`, `docs/codebase/dashboard.md` (reescrito: decía que no había dashboard). `GEMINI.md` sin menciones de puerto: sin cambios.
- Verificación: parseo de `install.ps1` sin errores; revisión del orquestador de fugas de salida en funciones PowerShell (ninguna).

## Verificación final (orquestador)

- `go test ./internal/update/ ./internal/workspace/ ./internal/dashboard/... ./cmd/axiom/... -count=1` → ok en los cuatro paquetes.
- `go vet ./...` → limpio.
- **Prueba manual pendiente (no ejecutable aquí por el antivirus y el navegador integrado)**: instalar desde la rama, abrir el acceso directo con el servidor parado y activo, instalar como app en Edge y anclar a la barra.

### Prueba manual del usuario (2026-11-05)

- PWA instalada desde Chrome en `http://127.0.0.1:47700`: funciona y mantiene su icono en la barra.
- `axiom ui --detach --app`: abre la ventana, pero la barra muestra el icono de Edge. Origina T5.

### T5

- `cmd/axiom`: retirado `--app` (`resolveAppBrowser`, `openAppWindow` y sus tests); `openUIWindow(url, noBrowser)` abre el navegador por defecto.
- `install.ps1`: `Register-AxiomLogonTask` registra por defecto la tarea de usuario `Axiom` (`AtLogOn` +30 s, `Interactive`, `Limited`, sin límite de tiempo, 5 reintentos, `IgnoreNew`, `ui --detach --no-browser`) y la arranca una vez; `-NoAutostart` la omite; retirados `-Autostart` y el acceso directo de inicio; `Axiom.lnk` del menú Inicio → `ui --detach`. No hay desinstalador en el repo: la retirada está documentada (`Unregister-ScheduledTask -TaskName Axiom -Confirm:$false`).
- Test adaptado en `internal/update/install_script_test.go`; docs actualizadas (`usage.md`, `codebase/dashboard.md`, `manual-de-inicio.md`, `quickstart.md`, `README.md`).
- Verificación del escritor: `go test ./cmd/axiom/... ./internal/update/ ./internal/workspace/ -count=1` → ok (`update` en una segunda ejecución; la primera falló porque se lanzó durante un renombrado); `go vet` (windows y `GOOS=linux`) → limpio; `gofmt -l` → vacío; parseo de `install.ps1` → 0 errores. Comprobación del orquestador: `go test ./cmd/axiom/ -run 'DetachedChildArgs|WaitForAxiom'` → ok; revisión de `Register-AxiomLogonTask`.
- **Pendiente de prueba manual**: que el proceso hijo de `--detach` sobreviva al terminar la acción de la tarea programada, y el destello de consola al iniciar sesión (la acción es un ejecutable de consola).

## Siguiente paso

Prueba manual de la tarea programada y PR (`scripts/axiom-worktree.ps1 pr ui-pwa-puerto-fijo`).
