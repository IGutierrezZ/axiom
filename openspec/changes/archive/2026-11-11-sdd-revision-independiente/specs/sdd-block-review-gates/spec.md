# Delta para `sdd-block-review-gates`

## ADDED Requirements

### Requirement: Las Paradas Presentan los Resultados de la Revisión Independiente cuando Existen (REQ-28.10)

En modalidad `checkpointed`, la parada de cada compuerta DEBE presentar, junto al artefacto que juzga y sin cambiar su semántica ni sus tres salidas (REQ-26.1), los resultados de revisión independiente que existan en ese momento: en `spec`, el informe de cobertura del spec; en `design`, el informe de cobertura de spec y design; en `role-apply:<rol>`, el informe vigente del verify de ese rol (con su sección «Cumplimiento técnico») y, si hubo juicio, su veredicto y los arreglos aplicados (REQ-28.8 de `sdd-independent-review`). La presentación DEBE enumerar los hallazgos abiertos y DEBE indicar explícitamente cuándo una revisión no se ha ejecutado, no ha respondido o no ha sido limpia; NO DEBE presentar como limpio un resultado ausente. Los resultados son informativos: NO DEBEN condicionar la aprobación, NO DEBEN registrarse en `gates.yaml`, ni su existencia o contenido DEBE cambiar el digest que juzga la puerta ni reabrirla. Cuando no existe ninguna revisión para esa puerta, la presentación NO cambia respecto a REQ-21.8, REQ-21.9 y REQ-21.11. Esta capacidad NO cambia REQ-21.7: en `continuous` no existen paradas, y los resultados se resumen al final del cambio según `sdd-independent-review`.

#### Scenario: La parada de spec presenta el informe de cobertura

- **DADO** un cambio `checkpointed` con un informe de cobertura del spec con dos huecos abiertos
- **CUANDO** se presenta la compuerta de `spec`
- **ENTONCES** la presentación enumera los dos huecos junto al spec
- **Y** ofrece las tres salidas sin alterarlas

#### Scenario: La parada de design presenta la cobertura de spec y design

- **DADO** un informe de cobertura posterior a `design`
- **CUANDO** se presenta la compuerta de `design`
- **ENTONCES** la presentación incluye sus hallazgos junto a la evaluación de conformidad de REQ-21.9
- **Y** con `technical_context.maintained` declarado, sigue presentando el delta de contexto

#### Scenario: La parada role-apply presenta el verify vigente y el juicio

- **DADO** un rol con verify limpio y un juicio `APPROVED` con un arreglo aplicado
- **CUANDO** se presenta la compuerta `role-apply:<rol>`
- **ENTONCES** la presentación incluye el informe vigente del verify, el veredicto del juicio y el arreglo aplicado
- **Y** la evaluación de conformidad con el diseño y la spec de REQ-21.11 se mantiene

#### Scenario: Una revisión no ejecutada se declara y no se presenta como limpia

- **DADO** un cambio con la preferencia `none` y sin ningún informe para el rol
- **CUANDO** se presenta la compuerta `role-apply:<rol>`
- **ENTONCES** la presentación no afirma que el verify sea limpio
- **Y** puede ofrecer lanzarlo antes de que el usuario elija una salida

#### Scenario: Un verify no limpio no impide aprobar

- **DADO** un verify del rol con un hallazgo crítico
- **CUANDO** el usuario elige una salida de aprobación en la parada
- **ENTONCES** la decisión se registra en `gates.yaml` como cualquier otra aprobación
- **Y** el hallazgo consta en la presentación pero no condiciona el registro

#### Scenario: El informe de cobertura no reabre la puerta

- **DADO** una puerta de `spec` con un registro `rejected` y un informe de cobertura generado después
- **CUANDO** se evalúa el estado de la puerta
- **ENTONCES** su digest, calculado sobre el artefacto, no ha cambiado
- **Y** no vuelve a `pending` por la presencia del informe

#### Scenario: Sin ninguna revisión la presentación no cambia

- **DADO** una puerta sin ningún informe de revisión independiente
- **CUANDO** se presenta
- **ENTONCES** la presentación es idéntica a la anterior a este cambio

#### Scenario: Las plantillas del orquestador prescriben la presentación de los resultados

- **DADO** los textos de las plantillas de orquestador que describen las compuertas `spec`, `design` y `role-apply`
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** indican presentar los resultados de revisión independiente cuando existen y declarar los ausentes
- **Y** no contienen vocabulario de RDD (REQ-28.11 de `sdd-independent-review`)
