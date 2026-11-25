# Delta para axiom-sdd-cli-integration

`axiom sdd status --json` proyecta, de forma aditiva, los identificadores de requisito que el cambio modifica y los que crea (C-03-b, A-03-3: la declaración es la delta).

Notación de los escenarios: `«prov-a»` es un marcador neutro de un identificador provisional; NO es el formato real, que fija el diseño (véase REQ-47.1 de `requirement-identifiers`).

## ADDED Requirements

### Requirement: `axiom sdd status --json` Proyecta los Requisitos que el Cambio Modifica y Crea (REQ-47.9)

`axiom sdd status --json` DEBE proyectar un campo aditivo `requirements` con dos listas, `modifies` y `creates`, derivadas de las deltas del cambio:

- `creates`: un elemento por cada requisito ADDED, con su identificador (el definitivo o el provisional, tal como está escrito en la delta) y la capacidad.
- `modifies`: un elemento por cada requisito MODIFIED, REMOVED y RENAMED (el nombre antiguo), con el identificador existente que cita y la capacidad.

Un elemento NO DEBE aparecer en las dos listas. El campo DEBE omitirse cuando el cambio no tiene deltas (cambio sin `specs/`, o con deltas sin ninguna sección ADDED, MODIFIED, REMOVED ni RENAMED), de modo que la salida de un cambio sin deltas es idéntica byte a byte a la anterior. Las listas DEBEN tener un orden determinista (capacidades por nombre y requisitos por aparición en la delta) y un `status` repetido sin cambios en las deltas DEBE producir la misma salida. El campo es una proyección informativa: NO DEBE alterar `nextRecommended`, `dependencies`, los motivos de bloqueo ni ningún campo anterior; no resuelve identificadores inexistentes (eso lo rechaza la integración, REQ-47.3). Con un provisional en `creates` el elemento muestra el provisional, no el número que la integración asignará. Su contrato se documenta en `internal/assets/skills/_shared/sdd-status-contract.md`.

#### Scenario: Un cambio con deltas proyecta `modifies` y `creates`

- **DADO** un cambio con una delta que añade un requisito `(REQ-31.1)` en `facturacion`, modifica `REQ-30.1` en `pagos` y elimina `REQ-30.2` en `pagos`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** `requirements.creates` contiene `REQ-31.1` con la capacidad `facturacion`
- **Y** `requirements.modifies` contiene `REQ-30.1` y `REQ-30.2` con la capacidad `pagos`

#### Scenario: Un RENAMED cuenta como modificación con el identificador existente

- **DADO** una delta con un RENAMED `Cobro (REQ-30.1) → Cobro con IVA (REQ-30.1)`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** `requirements.modifies` contiene `REQ-30.1`
- **Y** `requirements.creates` no lo contiene

#### Scenario: Un ADDED provisional se proyecta con su provisional

- **DADO** una delta con un ADDED `Cobro con IVA («prov-a»)`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** `requirements.creates` contiene `«prov-a»` con su capacidad
- **Y** no contiene ningún número definitivo asignado

#### Scenario: Sin deltas el campo se omite

- **DADO** un cambio sin carpeta `specs/` o con una delta sin ninguna sección de requisitos
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** la salida no contiene el campo `requirements`
- **Y** es idéntica byte a byte a la que producía antes de este cambio

#### Scenario: La proyección no altera el resto del estado

- **DADO** el mismo cambio con y sin el campo `requirements`
- **CUANDO** se comparan `nextRecommended`, `dependencies` y los motivos de bloqueo
- **ENTONCES** son idénticos

#### Scenario: La proyección es determinista

- **DADO** un cambio con deltas en varias capacidades
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json` dos veces sin cambiar las deltas
- **ENTONCES** las listas `modifies` y `creates` salen en el mismo orden en ambas salidas

#### Scenario: El contrato documenta el campo

- **DADO** el contrato de status de las skills (`sdd-status-contract.md`)
- **CUANDO** se inspecciona su texto
- **ENTONCES** describe `requirements.modifies` y `requirements.creates` y su omisión sin deltas
