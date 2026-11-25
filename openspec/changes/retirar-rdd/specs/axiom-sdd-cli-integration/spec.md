# Delta for axiom-sdd-cli-integration

## ADDED Requirements

### Requirement: El Binario No Expone `axiom review*`, `axiom review stop-hook` ni `axiom bench-model-picker` (REQ-46.1)

El binario `axiom` NO DEBE reconocer el verbo `review` ni ningún subverbo suyo (`start`, `status`, `capture`, `acknowledge`, `abandon`, `invalidate`, `recover`, `mode`, `assess`, `validate`, `stop-hook` y los demás), ni el verbo `bench-model-picker`, ni ofrecer una compatibilidad o una orden vacía (no-op) para ellos. La ayuda general y la de cada verbo NO DEBEN citarlos. Una invocación de `axiom review` o de `axiom review stop-hook` DEBE tratarse como un verbo desconocido: salida distinta de cero, mensaje de uso estándar y ningún efecto en el sistema de ficheros. Los verbos `axiom sdd-attempt grant`, `axiom sdd ...` y `axiom branch ...` NO cambian.

#### Scenario: `axiom review` es un verbo desconocido

- **DADO** el binario `axiom` construido sin etiquetas de compilación especiales
- **CUANDO** se ejecuta `axiom review status`
- **ENTONCES** termina con un código distinto de cero y el mensaje de verbo desconocido
- **Y** no se crea ni se modifica ningún fichero

#### Scenario: `axiom review stop-hook` no existe, ni siquiera como orden vacía

- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom review stop-hook --agent claude-code`
- **ENTONCES** termina con un código distinto de cero como verbo desconocido
- **Y** no emite ninguna salida de éxito ni de «no-op»

#### Scenario: La ayuda no cita los verbos retirados

- **DADO** la salida de `axiom help` y de `axiom --help`
- **CUANDO** se inspecciona
- **ENTONCES** no contiene `review`, `stop-hook` ni `bench-model-picker`
- **Y** sigue listando `sdd`, `branch`, `sdd-attempt` y el resto de verbos vigentes

#### Scenario: El consentimiento de edición SDD permanece

- **DADO** un cambio SDD con `blocked(edit_authority_missing)`
- **CUANDO** se ejecuta `axiom sdd-attempt grant` con su invocación exacta
- **ENTONCES** funciona como antes de la retirada
- **Y** el sobre `axiom.sdd-integration.consent/v1` se valida igual

### Requirement: Resiliencia en Decodificación de Rutas Windows en Salidas JSON de SDD (REQ-46.7)

Los consumidores y aserciones de la salida JSON de `sdd status` DEBEN decodificar estructuradamente en tipos Go nativos (`StatusV2Projection`) en lugar de hacer matching textual simple de rutas en Windows.
(Trasladado desde REQ-18.3 de `rdd-decoupling-v3-stability`, retirada con RDD (C-01-e).)

#### Scenario: Validación de rutas permitidas bajo Windows

- **DADO** un entorno de ejecución Windows con rutas que contienen separadores de barra invertida (`\`)
- **CUANDO** se valida la salida JSON de `RunSDDStatus`
- **ENTONCES** la validación se realiza mediante `json.Unmarshal`, tolerando el escape de backslashes sin fallos de aserción
