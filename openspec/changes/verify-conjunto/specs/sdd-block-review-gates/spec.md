# Delta para sdd-block-review-gates

> Parte del estado vigente: el resto de requisitos NO cambia. REQ-26.1 (las dos salidas de `integration`), REQ-21.7 (`continuous` sin paradas), REQ-43.23 y el contrato `axiom.sdd-gate-record/v1` se conservan tal cual. Esta delta solo añade la información que la presentación de `integration` DEBE declarar sobre la verificación conjunta (REQ-28.41 y REQ-28.42 de `sdd-independent-review`). Decisión del usuario del 2026-11-23: informar, nunca bloquear; del 2026-11-24: versión simple, tres estados y sin cálculo de vigencia. El identificador REQ-43.32 es el siguiente libre tras REQ-43.31.

## ADDED Requirements

### Requirement: La Presentación de `integration` Declara el Estado de la Verificación Conjunta como Información (REQ-43.32)

En un cambio `checkpointed`, la presentación de la compuerta `integration` DEBE declarar siempre exactamente uno de estos tres estados informativos de la verificación conjunta (REQ-28.41 de `sdd-independent-review`), consultando `axiom branch scope`:

1. **Hecha (fecha de la última ronda):** `verify-report.md` contiene al menos una ronda de la sección «Verificación conjunta»; la declaración indica la fecha de la última ronda. No se calcula vigencia: se muestra la fecha y decide la persona.
2. **Ofrecida:** no hay ninguna ronda y el resumen de `axiom branch scope` indica `offer: offerable`; el orquestador la ofrece y espera la decisión sin lanzarla.
3. **No verificable (motivo):** no hay ninguna ronda y la oferta no es `offerable`. El motivo es el de cada repositorio `unverifiable` (`branch_missing`, `base_missing` o `fork_point_undetermined`), «ramas pendientes de fusionar» cuando `offer` es `pending_merge`, o «sin commits propios que verificar» cuando es `nothing_to_verify`. Cuando la propia consulta falla (salida distinta de cero, rechazo tipado, salida ilegible), el estado es «no verificable» con el código del rechazo o «consulta no disponible».

La declaración es información: NO DEBE bloquear ni retrasar la presentación de `integration` (que se emite igualmente), NO DEBE alterar sus dos salidas, la evidencia de fusión que exige, `nextRecommended` ni las dependencias, y NO DEBE persistirse en `kickoff.yaml`, `gates.yaml` ni `axiom sdd status`. Aprobar `integration` sin verificación conjunta sigue siendo válido. Una petición explícita de la persona se atiende siempre (REQ-28.41). En un cambio `continuous` no existe la compuerta y esta declaración no se presenta.

#### Scenario: Estado hecha con la fecha de la última ronda
- **DADO** un `verify-report.md` con dos rondas de «Verificación conjunta», la última fechada el 2026-11-24
- **CUANDO** el orquestador presenta `integration`
- **ENTONCES** declara «hecha (2026-11-24)»
- **Y** no calcula si la ronda sigue vigente

#### Scenario: Estado ofrecida
- **DADO** ninguna ronda y `offer` igual a `offerable`
- **CUANDO** el orquestador presenta `integration`
- **ENTONCES** declara «ofrecida» y espera la decisión de la persona sin lanzarla

#### Scenario: Estado no verificable con ramas pendientes
- **DADO** ninguna ronda y un cambio multirrepo con la rama de `web` sin fusionar en `develop`
- **CUANDO** el orquestador presenta `integration`
- **ENTONCES** declara «no verificable (ramas pendientes de fusionar)» con la rama de `web`

#### Scenario: Estado no verificable por rama borrada
- **DADO** ninguna ronda y un repositorio `unverifiable` por `branch_missing`
- **CUANDO** el orquestador presenta `integration`
- **ENTONCES** declara «no verificable (branch_missing)» con el mensaje «la rama ya no está o falta un fetch autorizado»
- **Y** no ejecuta `fetch`

#### Scenario: La consulta de alcance falla
- **DADO** una ejecución de `axiom branch scope` que termina con un rechazo tipado, con salida ilegible o con salida distinta de cero
- **CUANDO** el orquestador presenta `integration`
- **ENTONCES** declara «no verificable» con el código del rechazo o «consulta no disponible»
- **Y** presenta la compuerta con sus dos salidas sin bloquearla

#### Scenario: Aprobar integration sin verificación conjunta sigue siendo posible
- **DADO** una compuerta `integration` pendiente, con la verificación conjunta no hecha
- **CUANDO** se registra `axiom sdd gate record --gate integration --decision approved` con la evidencia de fusión actual
- **ENTONCES** el registro se acepta con el mismo contrato `axiom.sdd-gate-record/v1`
- **Y** `nextRecommended` y las dependencias de `axiom sdd status` son idénticos a los que proyecta sin esa declaración

#### Scenario: En continuous no hay declaración
- **DADO** un cambio `continuous` con rol único `fullstack`
- **CUANDO** concluye el flujo
- **ENTONCES** no existe compuerta `integration` ni se presenta ninguna declaración de verificación conjunta
