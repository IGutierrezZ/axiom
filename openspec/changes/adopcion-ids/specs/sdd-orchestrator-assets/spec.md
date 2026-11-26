# Delta para sdd-orchestrator-assets

Este delta (C-03-c) actualiza la guía de adopción distribuida a todos los runtimes: describe la propuesta y la declaración del patrón, los identificadores conservados o provisionales y `pattern_undeclared`, en lugar de «un `(REQ-n.m)` de una serie nueva».

## MODIFIED Requirements

### Requirement: Skill o Referencia de Adopción de OpenSpec (REQ-30.31)

Axiom DEBE distribuir una skill o una referencia compartida de adopción que describa, en un texto idéntico para todos los runtimes, el flujo de migración a OpenSpec: ejecutar `axiom archive adopt scan --change <cambio>`; revisar el patrón de identificadores propuesto y sus conflictos y declararlo la persona en `sdd.id_pattern` de `axiom.yaml` (el escáner no lo escribe) antes de `adopt check` y de archivar; redactar la delta de adopción con solo `## ADDED Requirements`, al menos un escenario y una línea `Origen:` por requisito, y en cada título el identificador del cliente conservado (que debe coincidir con el `<ID original>` de su `Origen:`) o, cuando la unidad no trae identificador, un provisional `prov:<series>:<ordinal>` cuyo número se asigna al archivar; listar con motivo las unidades descartadas en `Descartado:`; normalizar a mano las mezclas de identificadores que el escáner informa como conflicto; ejecutar `axiom archive adopt check --change <cambio>` hasta que no reporte fallos (incluida la regla `pattern_undeclared`); someter la compuerta `spec` a la aprobación humana; y archivar con `axiom sdd archive-integrate`. El texto NO DEBE afirmar que Go juzga la fidelidad semántica de la migración, NO DEBE exigir «un `(REQ-n.m)` de una serie nueva», NO DEBE contener la excepción «hasta C-03-c» y NO DEBE ordenar `archive-compose --canonical --delta` para la adopción.
(Previously: la delta se redactaba con «un identificador `(REQ-n.m)` de una serie nueva» y el texto no mencionaba el patrón del cliente, los provisionales ni `archive-integrate`.)

#### Scenario: La guía de adopción describe el flujo completo
- **DADO** el texto de la skill o referencia de adopción
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** cita `adopt scan`, `adopt check`, `Origen:`, `Descartado:`, `sdd.id_pattern`, `pattern_undeclared`, `prov:<series>:<ordinal>`, `archive-integrate` y la compuerta `spec`

#### Scenario: La guía conserva los identificadores del cliente
- **DADO** el mismo texto
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica que el identificador del cliente se conserva en el título y coincide con el `<ID original>` de `Origen:`
- **Y** indica que la declaración del patrón la hace la persona y que `scan` no escribe `axiom.yaml`
- **Y** indica que las mezclas de esquemas se normalizan a mano

#### Scenario: La guía ya no fija una serie nueva ni la excepción temporal
- **DADO** el mismo texto, y el resto de skills que mencionen la adopción
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** ninguno contiene «una serie nueva» como exigencia del título ni «hasta C-03-c»

#### Scenario: La guía no atribuye a Go la fidelidad semántica
- **DADO** el mismo texto
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica que la fidelidad la aprueba un humano en la compuerta `spec`

#### Scenario: Paridad entre runtimes
- **DADO** dos runtimes soportados
- **CUANDO** se renderiza la guía de adopción
- **ENTONCES** el contenido normativo es idéntico entre ambos
