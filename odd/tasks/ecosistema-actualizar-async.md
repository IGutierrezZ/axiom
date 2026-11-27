# ODD: Actualizar desde Ecosistema en segundo plano y aviso solo de releases (ecosistema-actualizar-async)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/ecosistema-actualizar-async.md`. Espejo Engram: `odd/ecosistema-actualizar-async/tasks`.
> Rama: `fix/ecosistema-actualizar-async` · worktree `C:\repos\axiom-wt\ecosistema-actualizar-async\axiom`.
Tipo: bug

## Objetivo

Que «Actualizar» de la tarjeta «Actualizar herramientas» de Ecosistema termine igual que el botón lateral «Reiniciar con la última versión», que el aviso automático de versión nueva solo anuncie releases y que la cabeza de `main` se consulte únicamente bajo demanda desde Ecosistema.

## Problema y porqué

- **Actualizar falla en Ecosistema y no en la barra lateral.** `POST /api/ecosystem/upgrade` ejecuta `RunUpgradeSequence` (upgrade y sync) dentro de la propia petición HTTP, y el servidor tiene `WriteTimeout` de 15 s. Cuando la cadena tarda más, el navegador recibe «No se pudo contactar con el servidor» mientras el servidor sigue instalando, y si Axiom se sustituye nadie relanza el dashboard. El botón lateral usa un trabajo en segundo plano (`startUpdateJob`), con sondeo de estado y relanzamiento, y por eso termina.
- **El aviso anuncia `main`.** `usesBetaMainHeadCheck` consulta la cabeza de `main` cuando el canal es beta **o** cuando la versión es una pseudo-versión de Go (`v3.9.1-0.<ts>-<sha>`). Un binario instalado desde `main` recibe avisos de cada commit, también con el canal estable explícito, porque la pseudo-versión gana. El canal se pasa además por la variable de entorno del proceso (`os.Setenv` en `RunUpgradeReportWithChannel`), compartida con la comprobación del aviso.
- **Comparación con releases.** `compareVersions` solo mira `X.Y.Z`: una pseudo-versión `v3.9.1-0.<ts>-<sha>` se considera igual a la release `v3.9.1`, aunque es anterior.

## Decisiones

- **Canal explícito por parámetro, no por entorno.** `update.CheckFilteredWithOptions` recibe `CheckOptions{Channel}`; `CheckFiltered` la llama sin opciones. Un canal estable explícito (`stable`) compara solo con releases aunque la versión local sea una pseudo-versión; un canal beta (`beta`, `main`, `nightly`) consulta la cabeza de `main`; sin canal se mantiene la regla actual (entorno y pseudo-versión). `RunUpgradeReportWithChannel("stable")` hereda la regla nueva por la variable de entorno.
- **Prerrelease anterior a release.** Para la herramienta propia, con las mismas cifras `X.Y.Z`, una versión local con sufijo de prerrelease (`-…`) va por detrás de la release.
- **Aviso automático = releases.** `checkLatestAxiomVersion` usa el canal estable. El trabajo de la barra lateral y del aviso (`POST /api/system/update` y `/restart`) instala también con el canal estable: lo que anuncia es lo que instala.
- **Ecosistema usa el mismo trabajo.** `POST /api/ecosystem/upgrade {channel}` arranca el trabajo único del proceso con ese canal y responde `202` con su estado. El trabajo ejecuta la cadena completa (`upgrade` ➔ `sync`), guarda el informe consolidado en `result` del estado y, si Axiom se ha sustituido, relanza el dashboard como el botón lateral. Si ya hay un trabajo en curso devuelve ese.
- **`main` bajo demanda.** `GET /api/system/version/main` compara el binario instalado con la cabeza de `main`, sin caché. La tarjeta de Ecosistema lo muestra con «Buscar en main»; para instalarlo se elige el canal Main y se pulsa Actualizar.

## Alcance autorizado

`internal/update/` (`check.go` y pruebas), `internal/dashboard/` (`version_check.go`, `update_job.go`, `server.go`, `service.go` si hace falta, y pruebas), `internal/dashboard/assets/` (`index.html`, `js/update.js`, `js/screens/ecosystem.js`, `js/core/ecosystem.js`, `style.css` si hace falta), `docs/` afectados, este documento y las specs vivas `axiom-updater-resilience` y `tui-ui-parity` mediante `axiom archive integrate`.

## Modo TDD

Sin configuración de TDD en el proyecto ni en la sesión: comprobaciones funcionales. Ejecutor: `go test` localizado con `-run` en `./internal/update/` y `./internal/dashboard/`, y `node --check` de los JS tocados. La suite entera la pasa la CI.

## Entrega

`single-pr`: una PR con un commit por tarea. Etiqueta `type:bug`. `axiom.yaml` no declara `delivery.split_hint_lines`.

## Tareas

- [x] T1 · Canal explícito en la comprobación: `CheckOptions{Channel}` y `CheckFilteredWithOptions`; el canal estable explícito ignora la pseudo-versión; prerrelease por detrás de la release para la herramienta propia; el aviso del dashboard y el trabajo lateral usan el canal estable; pruebas. Ruta: inline (unidad pequeña, sin decisiones abiertas). Evidencia: `gofmt` limpio; `go vet ./internal/update/ ./internal/dashboard/` limpio; `go test ./internal/update/ -run 'UsesBetaMainHead|IsBetaUpdateChannel|PseudoVersionStable|HasPrerelease|CheckSingleTool|CheckFiltered|CheckAll|Homebrew'` ok; `go test ./internal/dashboard/ -run 'Version|Update|Restart|Upgrade|Sequence'` ok. `isBetaUpdateChannel` se sustituye por `effectiveChannel` + `isBetaChannel` (sin código huérfano).
  Cubre: prov:UPD:1
- [ ] T2 · Ecosistema por el trabajo en segundo plano: `POST /api/ecosystem/upgrade` arranca el trabajo con canal y responde `202`; `result` con el informe consolidado; relanzamiento si Axiom se sustituye; la tarjeta sigue el estado y pinta el resultado; pruebas. Ruta: inline.
- [ ] T3 · «Buscar en main» bajo demanda: `GET /api/system/version/main` sin caché y botón en la tarjeta de Ecosistema; pruebas e integración de la spec viva. Ruta: inline.

## Progreso

T1 hecha. Siguiente paso: T2.

## Criterio de terminado

- «Actualizar» de Ecosistema no depende del `WriteTimeout`: termina, muestra el informe por fases y, si Axiom se sustituye, relanza el dashboard y recarga la página.
- Con un binario instalado desde `main`, el aviso automático solo aparece cuando hay una release más reciente.
- «Buscar en main» dice si la cabeza de `main` es más reciente que el binario instalado.
- Las pruebas localizadas pasan y la CI está en verde.

## Spec viva

### Capacidad: axiom-updater-resilience

```openspec
## MODIFIED Requirements

### Requirement: Endpoint Web UI encadena upgrade ➔ sync con reporte consolidado (RF-UPD-04)

El endpoint `POST /api/ecosystem/upgrade` del Dashboard Web DEBE ejecutar secuencialmente la fase de actualización de binarios (`upgrade`) seguida de la fase de sincronización de reglas y configuraciones (`sync`) en el trabajo de actualización en segundo plano del proceso, con el canal indicado en la petición, y DEBE responder `202 Accepted` con el estado del trabajo sin esperar a que termine. Al terminar, `GET /api/system/update/status` DEBE incluir en `result` el DTO consolidado con el desglose por fases (`phases.upgrade`, `phases.sync`). Si la fase `upgrade` requiere reinicio del proceso (`restart_required: true`), la fase `sync` DEBE omitirse limpiamente (`skipped_reason: "restart-required"`) y el dashboard DEBE relanzarse con el binario nuevo. Si ya hay un trabajo en curso, DEBE devolver su estado y no lanzar otro.

#### Scenario: Secuencia completada en el Dashboard
- **DADO** una petición `POST /api/ecosystem/upgrade` desde la Web UI
- **CUANDO** no se requiere reinicio inmediato del servidor
- **ENTONCES** la respuesta es `202 Accepted` con el estado del trabajo
- **Y** se ejecuta `upgrade` y posteriormente `sync`
- **Y** el estado terminado del trabajo contiene en `result` el reporte de ambas fases

#### Scenario: Axiom sustituido durante la secuencia
- **DADO** una petición `POST /api/ecosystem/upgrade` que actualiza el propio binario de Axiom
- **CUANDO** la fase `upgrade` termina con `restart_required: true`
- **ENTONCES** `sync` queda omitida con `skipped_reason: "restart-required"`
- **Y** el dashboard se relanza con el binario nuevo, que sincroniza el workspace al arrancar

## ADDED Requirements

### Requirement: Aviso de versión nueva solo con releases (prov:UPD:1)

La comprobación automática de versión del dashboard (`GET /api/system/version`) y el trabajo que lanzan el aviso y «Reiniciar con la última versión» DEBEN comparar el binario instalado solo con la última release publicada, aunque el binario sea una pseudo-versión compilada desde `main`. Con las mismas cifras `X.Y.Z`, una versión con sufijo de prerrelease DEBE considerarse anterior a la release. Un canal estable explícito DEBE comparar con releases; solo un canal beta explícito (`beta`, `main` o `nightly`) consulta la cabeza de `main`.

#### Scenario: Binario de main sin release nueva
- **DADO** un binario instalado `v3.9.1-0.<fecha>-<sha>` y la última release `v3.9.0`
- **CUANDO** se consulta `GET /api/system/version`
- **ENTONCES** el estado es `up_to_date`, aunque `main` tenga commits posteriores

#### Scenario: Release que alcanza al binario de main
- **DADO** un binario instalado `v3.9.1-0.<fecha>-<sha>` y la última release `v3.9.1`
- **CUANDO** se consulta `GET /api/system/version`
- **ENTONCES** el estado es `update_available` con `latest: 3.9.1`

### Requirement: Comprobación de main bajo demanda (prov:UPD:2)

El dashboard DEBE exponer `GET /api/system/version/main`, que compara el binario instalado con la cabeza de `main` sin caché, y la pantalla Ecosistema DEBE ofrecer «Buscar en main» en la tarjeta «Actualizar herramientas» para mostrar si hay un commit más reciente. La comprobación de `main` NO DEBE ejecutarse de forma automática.

#### Scenario: Main por delante del binario instalado
- **DADO** un binario instalado compilado desde un commit anterior a la cabeza de `main`
- **CUANDO** el usuario pulsa «Buscar en main»
- **ENTONCES** la tarjeta muestra `main@<sha>` como versión disponible
- **Y** para instalarla basta con elegir el canal Main y pulsar «Actualizar»
```

### Capacidad: tui-ui-parity

```openspec
## MODIFIED Requirements

### Requirement: Sincronización y Actualización Reactiva de Herramientas en Web UI (RF-UIP-02)

El servidor HTTP local DEBE exponer los endpoints `POST /api/ecosystem/sync` y `POST /api/ecosystem/upgrade`. El endpoint `POST /api/ecosystem/upgrade` DEBE ejecutar secuencialmente la cadena `upgrade` ➔ `sync` en segundo plano, con reporte estructurado por fases (`phases.upgrade`, `phases.sync`) disponible en el estado del trabajo, omitiendo `sync` de forma segura si `upgrade` exige reinicio del proceso (`restart_required: true`). El frontend web DEBE ofrecer botones de acción interactiva en la pestaña `⚙️ Ecosistema & Herramientas`, seguir el avance del trabajo y mostrar un registro de salida con los resultados detallados de la operación.

#### Scenario: Ejecución de sincronización de configuraciones
- **DADO** la pestaña de Ecosistema en el Dashboard Web
- **CUANDO** el usuario pulsa el botón "Sincronizar Configuraciones"
- **Y** se envía una petición `POST /api/ecosystem/sync`
- **ENTONCES** el servidor ejecuta la sincronización de archivos gestionados y reglas de agentes
- **Y** responde con código `200 OK` y el resumen de archivos sincronizados en formato JSON

#### Scenario: Ejecución de actualización encadenada a sincronización
- **DADO** la pestaña de Ecosistema en el Dashboard Web
- **CUANDO** el usuario pulsa el botón "Actualizar Herramientas"
- **Y** se envía una petición `POST /api/ecosystem/upgrade`
- **ENTONCES** el servidor responde `202 Accepted` y, en segundo plano, comprueba versiones, actualiza binarios y encadena la sincronización de agentes si no se requiere reinicio
- **Y** la interfaz muestra el informe consolidado por fases cuando el trabajo termina
```
