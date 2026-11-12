# Delta para tui-ui-parity

> Parte del estado vigente: el resto de pantallas y requisitos NO cambia. Esta delta solo retira el aviso de baseline externa de la pantalla `ScreenLivingDoc`, porque la baseline ajena se retira.

## MODIFIED Requirements

### Requirement: Pantalla de Catálogo de Especificaciones Vivas en TUI (REQ-17.10)

La TUI DEBE implementar la pantalla `ScreenLivingDoc` que liste las especificaciones vivas disponibles en `<planning home>/specs/` (`openspec/specs/`) con sus dominios y versiones, y permita disparar la sincronización del catálogo maestro `INDEX.md` con un atajo de teclado `s`.
(Previously: con una baseline del proyecto declarada, la pantalla mostraba un aviso de baseline externa en lugar del catálogo y el atajo `s` mostraba el rechazo; la baseline ajena se retira)

#### Scenario: Sincronización del catálogo maestro desde la TUI
- **DADO** la pantalla `ScreenLivingDoc`
- **CUANDO** el usuario presiona la tecla `s` (sincronizar)
- **ENTONCES** se invoca `livingdoc.Service.Sync()`
- **Y** se muestra un mensaje de confirmación con el total de dominios y requisitos actualizados
