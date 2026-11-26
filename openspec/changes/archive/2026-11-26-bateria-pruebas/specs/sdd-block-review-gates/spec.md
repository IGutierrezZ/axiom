# Delta para Compuertas de Revisión por Bloques (sdd-block-review-gates)

> **Cambio:** `bateria-pruebas` (C-06-c). **Idioma:** castellano de España; identificadores, claves y tokens en inglés.
> RF-GAT-01 a RF-GAT-30 no cambian de texto; RF-GAT-04 no se modifica, solo se complementa.

## ADDED Requirements

### Requirement: `pruebas.md` Entra en el Digest de la Compuerta `tasks` solo Cuando Existe, con Cálculo Único en las Dos Rutas (RF-GAT-31)

El digest de la compuerta `tasks` DEBE incluir `pruebas.md` del cambio únicamente cuando el fichero existe. El conjunto de ficheros del digest (los `tasks*.md` más `pruebas.md` si existe) DEBE resolverse con una única función compartida por las dos rutas, la de `gate record` (`cli/sdd_gate_digest.go`) y la del status (`sddstatus/governance.go`), de modo que el digest de `axiom sdd gate record` y el de `axiom sdd status` coincidan siempre. Un cambio sin `pruebas.md` DEBE conservar exactamente el digest que tenía antes de este cambio, con la función activa o no. Editar `pruebas.md` DEBE producir un digest distinto.

La reapertura sigue la regla de `tasks.md` y NO la amplía: una compuerta `tasks` `rejected` vuelve a `pending` (reabierta) cuando cambia el digest, y una compuerta `approved` es terminal y sigue `approved` aunque `pruebas.md` se edite después (D-08), para no bloquear a `sdd-apply` contra sus propias casillas.

**Desvío documentado del plan C-06-c:** el criterio literal «editar tras aprobar reabre la puerta» se sustituye por dos garantías verificables: editar `pruebas.md` cambia el digest, y una compuerta rechazada se reabre al editar solo `pruebas.md`.

La existencia de `pruebas.md` y su cobertura NO DEBEN alterar `nextRecommended`, las dependencias ni los motivos de bloqueo.

#### Scenario: Sin `pruebas.md` el digest no cambia
- **DADO** un cambio sin `pruebas.md`, con la función `acceptance_tests` activa o no
- **CUANDO** se calcula el digest de la compuerta `tasks`
- **ENTONCES** es idéntico al calculado antes de este cambio

#### Scenario: Editar `pruebas.md` cambia el digest
- **DADO** un cambio con `tasks.md` y `pruebas.md`
- **CUANDO** se edita solo `pruebas.md`
- **ENTONCES** el digest de la compuerta `tasks` es distinto del anterior

#### Scenario: El digest coincide en `gate record` y en el status
- **DADO** un cambio con `pruebas.md`
- **CUANDO** se calcula el digest con `axiom sdd gate record` y con `axiom sdd status`
- **ENTONCES** ambos valores son iguales

#### Scenario: Una compuerta `tasks` rechazada se reabre al editar solo `pruebas.md`
- **DADO** una compuerta `tasks` en `rejected` sobre un cambio con `pruebas.md`
- **CUANDO** se edita solo `pruebas.md` y se consulta el status
- **ENTONCES** la compuerta pasa a `pending`, reabierta

#### Scenario: Una compuerta `tasks` aprobada sigue aprobada
- **DADO** una compuerta `tasks` en `approved`
- **CUANDO** se edita `pruebas.md` después de la aprobación
- **ENTONCES** la compuerta sigue `approved`
- **Y** el comportamiento es el mismo que al editar `tasks.md`

### Requirement: La Parada `tasks` Presenta `pruebas.md` junto a los Ficheros de Tareas y su Resultado Informativo (RF-GAT-32)

En modalidad "con paradas", cuando el status proyecta `acceptanceTests` activo (RF-KOF-22) y existe `pruebas.md`, la parada de la compuerta `tasks` DEBE presentar `pruebas.md` junto a cada `tasks.<rol>.md`, con el resultado de `axiom sdd tests check` (casos, escenarios y requisitos cubiertos y hallazgos abiertos). La presentación DEBE declarar la ausencia de `pruebas.md` (con el aviso de RF-KOF-22) cuando la función está activa y el fichero falta, y NO DEBE presentarlo ni declararlo cuando la función no está activa. Un resultado con hallazgos se presenta como tal y nunca como limpio. El resultado NUNCA condiciona una aprobación, no se registra en `gates.yaml` y no cambia las tres salidas de RF-GAT-07 ni el contrato de RF-GAT-04.

#### Scenario: Parada `tasks` con `pruebas.md`
- **DADO** un cambio en `checkpointed` con `acceptanceTests` activo y `pruebas.md` presente
- **CUANDO** se presenta la compuerta de `tasks`
- **ENTONCES** se presentan los ficheros de tareas y `pruebas.md` con su cobertura y hallazgos
- **Y** se ofrecen las tres salidas de RF-GAT-07 sin cambios

#### Scenario: Función activa y `pruebas.md` ausente
- **DADO** un cambio con `acceptanceTests` activo sin `pruebas.md`
- **CUANDO** se presenta la compuerta de `tasks`
- **ENTONCES** la presentación declara que falta `pruebas.md`
- **Y** la aprobación no queda bloqueada

#### Scenario: Función no activa
- **DADO** un proyecto con `enabled: false` o sin el bloque
- **CUANDO** se presenta la compuerta de `tasks`
- **ENTONCES** no se menciona `pruebas.md`
