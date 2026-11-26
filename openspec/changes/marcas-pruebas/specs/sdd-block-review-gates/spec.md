# Delta para Compuertas de Revisión por Bloques (sdd-block-review-gates)

> **Cambio:** `marcas-pruebas` (C-06-d). **Idioma:** castellano de España; identificadores, claves y tokens en inglés.
> Complementa RF-GAT-26 sin modificarlo.

## ADDED Requirements

### Requirement: La Presentación de `integration` Muestra el Resumen de Marcas por Entorno como Información (prov:GAT:1)

En un cambio `checkpointed`, cuando el status proyecta el resumen de marcas de `pruebas.md` (RF-KOF-22), la presentación de la compuerta `integration` DEBE mostrarlo por entorno (casos `ok` vigentes sobre el total aplicable, y marcas caducadas aparte), junto a la declaración de la verificación conjunta (RF-GAT-26). Sin resumen proyectado, la presentación NO DEBE mencionarlo. La doctrina del orquestador y el skill `sdd-archive` DEBEN fijar esta regla. El resumen es solo informativo: NO DEBE cambiar las dos salidas de `integration`, NO DEBE alterar la evidencia de fusión exigida, NO DEBE registrarse en `gates.yaml` ni cambiar ningún digest, y NO DEBE condicionar la aprobación ni retrasar la presentación. Un resumen con casos `ko`, `blocked`, sin marcar o caducados se presenta como tal y nunca como limpio. En `continuous` no hay compuerta y el resumen solo aparece en el resumen final.

#### Scenario: Resumen presente en `integration`
- **DADO** un cambio `checkpointed` con marcas vigentes en `pre` (5 de 5 `ok`)
- **CUANDO** el orquestador presenta `integration`
- **ENTONCES** la presentación muestra `pre: 5/5 ok` como información

#### Scenario: Marcas pendientes o fallidas se declaran tal cual
- **DADO** un entorno con un caso `ko` y otro caducado
- **CUANDO** se presenta `integration`
- **ENTONCES** el resumen refleja el `ko` y la caducada, y no se presenta como limpio

#### Scenario: Sin marcas no se menciona
- **DADO** un cambio sin `pruebas-estado.yaml`
- **CUANDO** se presenta `integration`
- **ENTONCES** la presentación no menciona las marcas

#### Scenario: No condiciona las salidas
- **DADO** un resumen con `ko`
- **CUANDO** la persona aprueba `integration` con la evidencia de fusión exigida
- **ENTONCES** la aprobación se registra sin cambios en sus dos salidas ni en la evidencia exigida
- **Y** `gates.yaml` no contiene el resumen
