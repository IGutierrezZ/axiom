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
- [ ] **T2 · Aviso en la UI.**
  - Al cargar, banner «Hay una versión nueva (actual → última)» con botón «Actualizar» y enlace a la release.
  - Sin banner si está al día, en compilación de desarrollo o si la comprobación falla.
  - Botón para volver a comprobar.
  - Comprobación: pruebas de assets de `internal/dashboard`.
- [ ] **T3 · Actualización como trabajo asíncrono.**
  - `POST /api/system/update` arranca un único trabajo; si ya hay uno, devuelve el existente.
  - `GET /api/system/update/status` devuelve la fase (`checking`, `upgrading`, `restarting`, `syncing`, `done`, `failed`), el mensaje y la sugerencia manual si falla.
  - La UI consulta el estado periódicamente; así la petición no depende del `WriteTimeout`.
  - Comprobación: `go test ./internal/dashboard/...`.
- [ ] **T4 · Relanzamiento con el binario nuevo.**
  - Si la autoactualización termina bien, el proceso viejo lanza `<exe> ui --path <raíz> --port <puerto> --no-browser --post-update-sync`, cierra su servidor y sale.
  - El nuevo reintenta enlazar el mismo puerto durante unos segundos.
  - El lanzamiento de procesos va detrás de una costura inyectable.
  - La UI detecta el cambio de versión y recarga.
  - Comprobación: `go test ./cmd/axiom/... ./internal/dashboard/...`.
- [ ] **T5 · Sincronización con el binario nuevo.**
  - Con `--post-update-sync`, el proceso nuevo ejecuta la sincronización del workspace antes de atender peticiones de acción.
  - Expone su resultado en `GET /api/system/update/status`, y la UI lo muestra.
  - Comprobación: `go test ./internal/dashboard/... ./internal/app/...`.
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
- 2026-11-05: T1 hecha. `GET /api/system/version` (`internal/dashboard/version_check.go`) con caché en memoria de 6 h, reloj y comprobador inyectables, `?refresh=1`; fallo de red = `check_failed` con HTTP 200 (no se cachea). `go test ./internal/dashboard/... ./internal/update/... ./cmd/axiom/...`: ok; `internal/app`: solo falla el fallo ambiental conocido de OpenCode; `go vet`: ok; `gofmt -l`: vacío.
