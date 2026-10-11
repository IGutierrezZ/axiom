# Delta para Paridad Bidireccional entre TUI y Web UI (INC-24)

> **Incremento:** `inc-24-spec-baseline-profile` · **Tipo:** delta sobre la capacidad viva `tui-ui-parity`

Cambios de este delta: la pantalla `ScreenLivingDoc` lista las especificaciones del planning home resuelto (REQ-24.10) y, con baseline del proyecto declarada, informa de que la baseline es externa y rechaza la sincronización (REQ-24.23 y REQ-24.25). Este requisito entra porque el valor por defecto de la pregunta Q11, aceptado, hace que las lecturas de catálogo avisen de una baseline externa.

## MODIFIED Requirements

### Requirement: Pantalla de Catálogo de Especificaciones Vivas en TUI (REQ-17.10)
La TUI DEBE implementar la pantalla `ScreenLivingDoc` que liste las especificaciones vivas disponibles en `<planning home>/specs/` (en la baseline por defecto, `openspec/specs/`) con sus dominios y versiones, y permita disparar la sincronización del catálogo maestro `INDEX.md` con un atajo de teclado `s`. Con baseline del proyecto declarada, la pantalla DEBE mostrar un aviso de baseline externa con los dominios y rutas declarados en lugar del catálogo, y el atajo `s` DEBE mostrar el rechazo con su continuación sin escribir `INDEX.md`.
(Previously: listaba siempre `openspec/specs/` y sincronizaba siempre el catálogo)

#### Scenario: Sincronización del catálogo maestro desde la TUI
- **DADO** la pantalla `ScreenLivingDoc`
- **CUANDO** el usuario presiona la tecla `s` (sincronizar)
- **ENTONCES** se invoca `livingdoc.Service.Sync()`
- **Y** se muestra un mensaje de confirmación con el total de dominios y requisitos actualizados

#### Scenario: Pantalla con baseline del proyecto
- **DADO** la pantalla `ScreenLivingDoc` sobre un workspace con baseline del proyecto
- **CUANDO** el usuario accede a la pantalla y presiona la tecla `s`
- **ENTONCES** la pantalla muestra el aviso de baseline externa con los dominios y rutas declarados
- **Y** la sincronización muestra un mensaje de rechazo con su continuación y no se escribe `INDEX.md`
