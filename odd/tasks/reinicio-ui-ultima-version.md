# ODD: reiniciar el dashboard con la última versión (reinicio-ui-ultima-version)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/reinicio-ui-ultima-version.md`. Espejo Engram: `odd/reinicio-ui-ultima-version/tasks`.
> Rama: `inc/reinicio-ui-ultima-version` · worktree `C:\repos\axiom-wt\reinicio-ui-ultima-version\axiom`.

## Objetivo

Que el dashboard distinga «hay que reiniciar» de «hay que actualizar», y que ofrezca siempre un botón «Reiniciar con la última versión» que instale la última versión si hace falta y relance el dashboard.

## Problema y porqué

Caso observado el 26/11/2026 en Windows:

- La UI (`axiom ui`, arrancada el 23/11) siguió en marcha mientras un `go install ./cmd/axiom` externo escribía el binario nuevo. Windows no deja sobrescribir un `.exe` en uso: Go apartó la imagen de la UI a `go\bin\axiom.exe~` y escribió el nuevo `axiom.exe`, que ya era la última versión (`56e2de9`).
- La UI compara la última versión con la suya en memoria (`versionChecker.Get` con `app.Version`; la herramienta propia no tiene `DetectCmd`), así que ofreció «Actualizar». El `go install` que lanzó falló: Go no puede reutilizar `axiom.exe~`, que es la imagen de la propia UI, y otro proceso tenía `axiom.exe` abierto en ese momento.
- Además, la UI vieja mostraba el repositorio como «no gobernado» porque no entiende el `axiom.yaml` actual. El binario nuevo lo valida sin errores (`axiom workspace validate`: `COMPLIANT`).
- La orden manual que propone la UI es siempre `@latest` (`manualUpdateHint`), aunque en el canal beta la actualización instala `@main`.

Bastaba con reiniciar, pero la UI no lo detecta ni lo ofrece.

## Decisiones

- **Versión instalada = la que arrancaría un reinicio.** Se resuelve con `resolveRelaunchExecutable` y se lee ejecutando `<binario> version` (formato de `printVersion`: `<plataforma> version <Version> (<os>/<arch>) commit:<x>`), con un tiempo límite corto. No se lee `debug/buildinfo`, porque no ve la versión que se inyecta con `-ldflags` en las releases.
- **Estado `restart_required`.** Si la versión instalada difiere de la del proceso, la comprobación contra la última versión se hace con la instalada. Si la instalada está por detrás de la última, el estado sigue siendo `update_available` (instalar y reiniciar); si no, pasa a `restart_required` (solo reiniciar). Se compara por igualdad, no por orden: reiniciar carga siempre el binario instalado. Si la versión instalada no se puede leer, todo sigue como antes.
- **El trabajo decide, no el botón.** `POST /api/system/update` reinicia sin `go install` cuando el estado es `restart_required`. `POST /api/system/restart` (nuevo, solo desde el propio dashboard) fuerza el reinicio: instala si hay una versión más reciente y relanza siempre, también cuando ya está al día, es una compilación de desarrollo o la comprobación falla.
- **Recarga por instancia.** `GET /api/system/version` añade `instance`, un identificador del proceso servidor. La UI recarga cuando responde otra instancia, porque un reinicio forzado puede mantener la misma versión.
- **Orden manual según el canal:** `@main` si la última versión es `main@<sha>`, y `@latest` en el canal estable.
- **Instalación en Windows sin `axiom.exe~`.** En Windows, el propio binario se compila con `GOBIN` en un directorio temporal dentro del destino. Después se aparta el binario anterior con un nombre único terminado en `.old`, que `resolveRelaunchExecutable` ya descarta, y se mueve el nuevo a su sitio. Si mover falla, se restaura el anterior. Los restos `.old` y `~` se borran si están libres.
- **Botón permanente** en el pie de la barra lateral («Reiniciar con la última versión»). El aviso de fallo también ofrece «Reiniciar» además de «Reintentar».

## Alcance autorizado

`internal/update/upgrade/` (estrategia `go install` y sus pruebas), `internal/dashboard/` (`version_check.go`, `update_job.go`, `relaunch.go`, `server.go` y sus pruebas), `internal/dashboard/assets/` (`index.html`, `js/update.js`, `style.css` si hace falta), este documento y la spec viva `axiom-updater-resilience` mediante `axiom archive integrate`.

## Modo TDD

Sin configuración de TDD en el proyecto (`axiom sdd status`: `tdd: null`) ni en la sesión: comprobaciones funcionales. Ejecutor: `go test` localizado con `-run` en `./internal/update/upgrade/` y `./internal/dashboard/`. La suite entera la pasa la CI.

## Entrega

`single-pr`: una PR con un commit por tarea. Etiqueta `type:feature`. `axiom.yaml` no declara `delivery.split_hint_lines`.

## Tareas

- [x] T1 · Instalación propia en Windows independiente de `axiom.exe~`: `go install` con `GOBIN` temporal dentro del destino, sustitución con nombre único `.old`, restauración si falla y limpieza de restos, con pruebas. Ruta: inline (una sola unidad, sin decisiones abiertas). Evidencia: `go vet ./internal/update/upgrade/` limpio; `go test ./internal/update/upgrade/ -run 'SelfWindows|SwapSelfBinary|UniqueAside|WindowsBeta|AxiomWindows|GoInstall|RunStrategy|SelfUpgrade'` ok. `TestGoInstallSelfWindowsStagesTheBuildAndSwapsItIn` mantiene abierto `axiom.exe~` y la sustitución funciona igual. `TestWindowsBetaAxiomUpgradeUsesShippedRegistryGoTarget` se adapta: su `go install` simulado deja el binario en el directorio temporal.
  Cubre: prov:UPD:3
- [x] T2 · Reinicio pendiente: sonda de la versión instalada, estado `restart_required` y campos `installed` y `restart_required` en `GET /api/system/version`; el trabajo de actualización reinicia sin reinstalar; orden manual según el canal; aviso «Reiniciar» en `update.js`; pruebas. Ruta: inline. El campo `instance` pasa a T3, que es quien lo necesita. Evidencia: `go vet ./internal/dashboard/` limpio; `go test ./internal/dashboard/ -run 'Version|Update|Restart|Relaunch|ParseVersion|ManualUpdateHint|SandboxHome'` ok (nuevas en `restart_required_test.go`, más `TestUpdateBannerOffersRestartWhenTheInstalledBinaryDiffers`); `node --check` de `update.js` ok. `TestMain` sustituye la sonda para no ejecutar el binario de pruebas.
  Cubre: prov:UPD:1
- [ ] T3 · Botón «Reiniciar con la última versión»: `POST /api/system/restart` con trabajo forzado, botón en el pie de la barra lateral y en el aviso de fallo, recarga por cambio de instancia; pruebas. Ruta: inline.
  Cubre: prov:UPD:2

## Progreso

T1 hecha en `0b2b7e75e`. T2 hecha (commit en el registro de la rama). Siguiente paso: T3.

## Criterio de terminado

- Con un binario instalado distinto del proceso y al día, la UI ofrece «Reiniciar», no «Actualizar», y el reinicio no ejecuta `go install`.
- «Reiniciar con la última versión» relanza el dashboard siempre e instala antes si hay una versión más reciente; la página se recarga al responder el proceso nuevo.
- En Windows, un `axiom.exe~` ocupado no impide actualizar.
- La orden manual del canal beta es `@main`.
- Las pruebas localizadas pasan y la CI está en verde.

## Spec viva

### Capacidad: axiom-updater-resilience

```openspec
## ADDED Requirements

### Requirement: Reinicio pendiente detectado en el dashboard (prov:UPD:1)

El dashboard DEBE comparar la versión del proceso en ejecución con la del binario de Axiom que arrancaría al reiniciarse. Cuando difieren y el binario instalado no está por detrás de la última versión publicada, DEBE ofrecer reiniciar el dashboard y NO DEBE ofrecer ni ejecutar una actualización. Si no puede leer la versión del binario instalado, DEBE comprobar con la versión del proceso, como hasta ahora.

#### Scenario: Binario ya actualizado por fuera
- **DADO** un dashboard en ejecución con la versión A
- **Y** un binario instalado que ya es la última versión B
- **CUANDO** se consulta `GET /api/system/version`
- **ENTONCES** la respuesta trae `status: restart_required`, `installed: B` y `restart_required: true`
- **Y** el aviso de la interfaz ofrece «Reiniciar» en lugar de «Actualizar»
- **Y** `POST /api/system/update` relanza el dashboard con el binario instalado sin ejecutar `go install`

#### Scenario: Binario instalado también por detrás
- **DADO** un dashboard con la versión A y un binario instalado B, distinto de A y anterior a la última versión C
- **CUANDO** se consulta `GET /api/system/version`
- **ENTONCES** el estado es `update_available` con `restart_required: true`
- **Y** la actualización instala C y después relanza el dashboard

### Requirement: Reinicio forzado con la última versión (prov:UPD:2)

El dashboard DEBE ofrecer en todo momento la acción «Reiniciar con la última versión» (`POST /api/system/restart`, solo desde el propio dashboard). Esa acción DEBE instalar la última versión publicada cuando el binario instalado esté por detrás y DEBE relanzar siempre el dashboard con el binario instalado, también cuando ya está al día. Si la instalación falla, el dashboard en ejecución DEBE seguir sirviendo y mostrar el motivo con la orden manual del canal en uso: `@main` en el canal beta y `@latest` en el estable.

#### Scenario: Reinicio forzado sin versión nueva
- **DADO** un dashboard al día
- **CUANDO** el usuario pulsa «Reiniciar con la última versión»
- **ENTONCES** no se instala nada
- **Y** el dashboard se relanza en el mismo puerto y la página se recarga cuando responde el proceso nuevo, aunque la versión sea la misma

#### Scenario: Reinicio forzado con versión nueva
- **DADO** un binario instalado por detrás de la última versión
- **CUANDO** el usuario pulsa «Reiniciar con la última versión»
- **ENTONCES** se instala la última versión y el dashboard se relanza con ella

### Requirement: Instalación propia en Windows independiente del nombre apartado (prov:UPD:3)

En Windows, la actualización del propio binario con `go install` DEBE compilar en un directorio temporal dentro del destino y sustituir el binario apartando el anterior con un nombre único, de modo que un `axiom.exe~` ocupado por un proceso anterior no impida la actualización. Si la sustitución falla, el binario anterior DEBE quedar restaurado en su ruta.

#### Scenario: Nombre apartado ocupado por un dashboard anterior
- **DADO** un `axiom.exe~` en uso por un proceso anterior en el directorio de destino
- **CUANDO** se actualiza Axiom con `go install` en Windows
- **ENTONCES** el binario nuevo queda en `axiom.exe`
- **Y** el anterior queda apartado con un nombre único terminado en `.old`

#### Scenario: Falla la sustitución
- **DADO** una compilación correcta en el directorio temporal
- **CUANDO** no se puede mover el binario nuevo a su ruta
- **ENTONCES** el binario anterior vuelve a su ruta y la actualización informa del fallo
```
