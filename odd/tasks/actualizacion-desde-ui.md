# ODD · actualizacion-desde-ui (P2 de la hoja de ruta de visión de producto)

## Objetivo

Al abrir el dashboard, el usuario ve si su Axiom no es la última versión. Con un clic, Axiom:

1. se actualiza;
2. relanza el dashboard con el binario nuevo;
3. ejecuta la sincronización del workspace abierto.

Todo esto debe funcionar también en Windows, donde el `axiom.exe` que sirve el dashboard está en uso.

## Problema y porqué

Auditoría `docs/audits/2026-11-03-vision-producto.md`, objetivo O2, huecos G2.1–G2.3. Lo verificado al mapear:

- **No hay comprobación de versión.** El dashboard ya encadena la actualización y la sincronización (`service.go`, `app.RunUpgradeReportWithChannel`), pero no comprueba la versión: no hay endpoint ni aviso.
  - `update.CheckAllWithCooldown` devuelve `nil` cuando su caché está vigente, así que no sirve para el aviso.
- **La sincronización se salta siempre.** Tras autoactualizarse, `ResolveSyncSkip` la salta (`restart-required`). Tiene sentido: el proceso viejo sincronizaría con los assets viejos.
  - Nada la ejecuta después con el binario nuevo.
  - `runUI` no lee `PendingSync`.
- **No hay relanzamiento.** No existe código de autorrelanzamiento.
  - El puerto no es fijo: 8080 más un sondeo de hasta 10 puertos.
- **Las acciones largas se cortan.** Son POST síncronos con `WriteTimeout` de 15 s, y la actualización y la sincronización pueden superarlo.
- **Windows.** La actualización del propio Axiom va por `go install …@vX`.
  - El repositorio no tiene lógica propia de apartar el exe en uso; depende de lo que haga la toolchain de Go.
  - El preflight rechaza si `GOBIN` no coincide con el `axiom` del `PATH`.

## Decisiones

- **Comprobación automática:** al cargar el dashboard, con caché en memoria (TTL de unas 6 h) y un botón para volver a comprobar. Lo pide la visión O2: «la UI debe detectarlas y avisar».
- **Sincronización:** la ejecuta el binario nuevo tras relanzarse, nunca el proceso viejo. La regla de `ResolveSyncSkip` del proceso viejo se mantiene.
- **Windows sin Go o con `GOBIN` distinto del `PATH`:** la UI muestra el comando manual. Es el comportamiento actual; no se añade compilación desde fuentes.
- **Entrega (decisión del usuario, 2026-11-05):** una sola PR con `size:exception` y un commit por tarea. Previsión: unas 900 líneas.

## Alcance autorizado

- `internal/dashboard/**`: servicio, servidor, handlers, assets y sus pruebas.
- `cmd/axiom/main.go`: `runUI`, flags y relanzamiento.
- `internal/app/**`: solo lo necesario para sincronizar tras actualizar, con sus pruebas.
- `internal/update/**`: solo si hace falta exponer algo.
- `docs/usage.md` y este documento.

Fuera: GoReleaser, políticas de release, contratos `gentle-ai.*` / `GENTLE_AI_*` y el flujo de la TUI.

## Configuración

- **TDD:** off (fuente: `axiom.yaml` no lo configura). Se hacen las comprobaciones funcionales normales.
- **RDD:** off (`axiom review mode status`: decidido por defecto).
- **Ejecutor de pruebas:** `go test` en proceso. **Prohibido compilar binarios sueltos** (Cortex).
- **Entrega:** `single-pr` con `size:exception`, un commit por tarea.

## Tareas

- [x] **T1 · Servicio y endpoint de versión.**
  - `GET /api/system/version` devuelve `{current, latest, update_available, status, release_url}`.
  - Caché en memoria con TTL; reloj y comprobador inyectables.
  - Un fallo de red da `status: check_failed` sin romper el dashboard.
  - Las compilaciones de desarrollo dan `dev_build` sin aviso.
  - Comprobación: `go test ./internal/dashboard/...`.
- [x] **T2 · Aviso en la UI.**
  - Al cargar, banner «Hay una versión nueva (actual → última)» con botón «Actualizar» y enlace a la release.
  - Sin banner si está al día, en compilación de desarrollo o si la comprobación falla.
  - Botón para volver a comprobar.
  - Comprobación: pruebas de assets de `internal/dashboard`.
- [x] **T3 · Actualización como trabajo asíncrono.**
  - `POST /api/system/update` arranca un único trabajo; si ya hay uno, devuelve el existente.
  - `GET /api/system/update/status` devuelve la fase (`checking`, `upgrading`, `restarting`, `syncing`, `done`, `failed`), el mensaje y la sugerencia manual si falla.
  - La UI consulta el estado periódicamente; así la petición no depende del `WriteTimeout`.
  - Comprobación: `go test ./internal/dashboard/...`.
- [x] **T4 · Relanzamiento con el binario nuevo.**
  - Si la autoactualización termina bien, el proceso viejo lanza `<exe> ui --path <raíz> --port <puerto> --no-browser --post-update-sync`, cierra su servidor y sale.
  - El nuevo reintenta enlazar el mismo puerto durante unos segundos.
  - El lanzamiento de procesos va detrás de una costura inyectable.
  - La UI detecta el cambio de versión y recarga.
  - Comprobación: `go test ./cmd/axiom/... ./internal/dashboard/...`.
- [x] **T5 · Sincronización con el binario nuevo.**
  - Con `--post-update-sync`, el proceso nuevo ejecuta la sincronización del workspace antes de atender peticiones de acción.
  - Expone su resultado en `GET /api/system/update/status`, y la UI lo muestra.
  - Comprobación: `go test ./internal/dashboard/... ./internal/app/...`.
- [x] **T5b · Correcciones de la verificación independiente.**
  - La recarga tras actualizar ya no depende de ver la fase `restarting`: se recarga también si responde un proceso con otro `from_version` o tras fallos de red previos.
  - `POST /api/system/update` rechaza con 403 un `Host` u `Origin` que no sea `127.0.0.1`/`localhost` en el puerto realmente enlazado (CSRF y DNS rebinding).
  - El proceso viejo comprueba que el hijo sigue vivo (ventana de 1,5 s, inyectable) antes de cerrar su servidor; si murió, falla con el comando manual y sigue sirviendo.
  - La sincronización posterior se arranca solo tras enlazar el puerto (antes se anuncia como `syncing` pendiente).
  - Comandos manuales copiables: `axiom ui --path "<ruta>"` con comillas dobles simples; el tope de 90 s de la UI reutiliza ese comando.
- [ ] **T6 · Documentación y cierre.**
  - `docs/usage.md` (actualizar desde el dashboard y caso de Windows).
  - Coherencia con `openspec/specs/axiom-updater-resilience/spec.md`: el proceso viejo sigue saltando la sincronización.
  - PR. Al fusionar, marcar P2 en la auditoría.
  - Prueba manual en Windows por el usuario.

## Criterios de aceptación

- Con una versión publicada más nueva, el dashboard muestra el aviso al abrirse, sin pulsar nada.
- Un clic actualiza, relanza el dashboard en el mismo puerto con el binario nuevo y sincroniza el workspace abierto. La UI muestra cada fase y el resultado de la sincronización.
- Si falla cualquier fase, la UI muestra el error y el comando manual, y el dashboard viejo sigue sirviendo.
- `go test ./internal/dashboard/... ./internal/app/... ./internal/update/... ./cmd/axiom/...` en verde (salvo el fallo ambiental conocido de OpenCode).

## Riesgos

- **Windows, `go install` sobre el exe en uso.** Hay que comprobarlo a mano: aquí no se compilan binarios sueltos.
- **EDR.** Cortex puede borrar el binario recién instalado y dejar al usuario sin dashboard. En ese caso, la UI ya habrá mostrado el comando manual antes de relanzar.

## Progreso y evidencias

- 2026-11-05: mapa hecho (delegado, solo lectura).
- 2026-11-05: T1 hecha (`eff3ea87`). `GET /api/system/version` (`internal/dashboard/version_check.go`) con caché en memoria de 6 h, reloj y comprobador inyectables, `?refresh=1`; fallo de red = `check_failed` con HTTP 200 (no se cachea). `go test ./internal/dashboard/... ./internal/update/... ./cmd/axiom/...`: ok; `internal/app`: solo falla el fallo ambiental conocido de OpenCode; `go vet`: ok; `gofmt -l`: vacío.
- 2026-11-05: T2 hecha (`41a246e0`). Módulo `assets/js/update.js` + contenedor `#update-banner` en `index.html`: aviso «Hay una versión nueva de Axiom: actual → última» con «Actualizar», «Ver la release» (solo https) y «Comprobar de nuevo» (`?refresh=1`); sin aviso si está al día, en dev_build o si falla. Pruebas de assets en `update_assets_test.go`. `go test ./internal/dashboard/... ./cmd/axiom/...`: ok; `go vet`: ok; `gofmt -l`: vacío. El botón «Actualizar» solo hace POST a `/api/system/update` (la ruta llega en T3).
- 2026-11-05: T3 hecha (`fc53c73f`). `internal/dashboard/update_job.go`: trabajo único en memoria (mutex) lanzado en una goroutine independiente de la petición; `POST /api/system/update` (202, devuelve el existente si ya hay uno) y `GET /api/system/update/status` (`{phase, running, message, manual_hint, sync}`, fases checking|upgrading|restarting|syncing|done|failed, más `idle`). Reutiliza `RunUpgradeSequence`; el proceso viejo sigue saltando el sync (restart-required). UI: `update.js` consulta cada 1,5 s y muestra fases, resultado y comando manual. Provisional hasta T4: tras autoactualizarse la fase `restarting` pasa enseguida a `done` con «reinicia el dashboard». `go test ./internal/dashboard/... ./cmd/axiom/...`: ok; `go vet`: ok; `gofmt -l`: vacío.
- 2026-11-05: T4 hecha (`bed07f83`). `relaunch.go` (+ `relaunch_windows.go` con DETACHED_PROCESS y `relaunch_unix.go` con Setsid): tras autoactualizarse, el proceso viejo lanza `<exe> ui --path <raiz> --port <puerto> --no-browser --post-update-sync`, cierra su servidor y sale; ejecutable, lanzamiento y salida detrás de costuras (`executablePathFn`, `lookPathFn`, `startProcessFn`, `exitProcessFn`), sin procesos reales en las pruebas. Si `os.Executable` apunta al exe apartado por `go install` (`~`/`.old`) o ya no existe, usa el `axiom` del PATH. `Server.ListenAndServeExact` reintenta el MISMO puerto; `runUI` parsea `--post-update-sync` (`parseUIFlags`, `startDashboardServer`). Si falla el relanzamiento: fase `failed` con `axiom ui --path ...` y el dashboard viejo sigue sirviendo. UI: en `restarting` consulta `/api/system/version` hasta que responde otra versión y recarga (tope de 90 s con comando manual). `go test ./internal/dashboard/... ./cmd/axiom/...`: ok; `go vet`: ok; `GOOS=linux go vet ./internal/dashboard/`: ok; `gofmt -l`: vacío. Pendiente de prueba manual: el relanzamiento real en Windows.
- 2026-11-05: T5 hecha (`cf3e9e03`). `Service.StartPostUpdateSync` (`update_job.go`): con `--post-update-sync`, `runUI` deja el estado en `syncing` antes de enlazar el puerto y sincroniza en segundo plano (`RunSync("workspace")`, es decir `sync --scope workspace`); resultado en `GET /api/system/update/status` (`done`/`failed` con `sync.{executed,success,output,error}` y comando manual `axiom sync --scope workspace` si falla). Sin carrera de `os.Chdir`: la raíz es la carpeta donde `runUI` ya dejó el proceso (documentado en el código). Decisión de seguridad: una carpeta sin `axiom.yaml` no se sincroniza (`skipped_reason: no-workspace`). UI: tras recargar, `resumeUpdate` recupera el estado y muestra el resultado con el detalle plegable. `go test ./internal/dashboard/... ./internal/app/... ./internal/update/... ./cmd/axiom/...`: ok salvo el fallo ambiental conocido `TestTuiInstallOnThenSyncPreservesAndRefreshesOpenCodeActivation`; `go vet`: ok; `gofmt -l`: vacío.
- 2026-11-05: T5b hecha. Correcciones de la verificación independiente en `update.js` (`newProcessAnswered`, `trackUpdate`), `server.go` (`isLocalRequest`, aplicado solo a `POST /api/system/update`), `relaunch.go` y sus variantes por sistema (`startProcessFn` devuelve el canal de salida del hijo; `relaunchLiveCheck`), `update_job.go` (`MarkPostUpdateSyncPending`) y `cmd/axiom/main.go` (sincronización tras el enlace). Pruebas nuevas: `update_guard_test.go`, `update_reload_assets_test.go` y casos en `relaunch_test.go`.

## Seguimientos fuera de alcance

- `RunSync` ya cambiaba de directorio (`os.Chdir`) en proceso: cambiar de workspace durante una sincronización sigue siendo un riesgo previo.
- Los `POST /api/ecosystem/*` existentes no tienen protección de `Origin`/`Host`; `isLocalRequest` queda disponible para aplicarla.
- `versionChecker.Get` mantiene su mutex durante la consulta de red (aceptable: máximo 10 s y solo bloquea otras comprobaciones).
