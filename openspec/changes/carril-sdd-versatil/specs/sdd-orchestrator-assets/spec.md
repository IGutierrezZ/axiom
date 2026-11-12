# Delta para sdd-orchestrator-assets

## ADDED Requirements

### Requirement: Guía y Doctrina del Modo ISO, `Cubre:`, la Matriz y el Bug sin Delta, con la Excepción ISO, en Paridad entre Runtimes (REQ-29.17)

Las skills `sdd-*` (sobre todo `sdd-verify`, con su referencia del formato de informe, `sdd-tasks` y `sdd-archive`), la doctrina compartida (`_shared/persistence-contract.md`, `_shared/sdd-status-contract.md` y `_shared/sdd-orchestrator-sections.md`) y los orquestadores proyectados a cada runtime DEBEN describir las opciones de este cambio: el modo ISO y la proyección de `sdd.iso` en el status (REQ-29.2), `kickoff seal` copiando `iso` del proyecto en `kickoff.yaml` (REQ-29.13), la línea `Cubre:` de las tareas, la matriz de verify con sus filas automáticas y manuales y las formas de referencia de prueba admitidas (`ruta/fichero:símbolo` y `TestX`; REQ-29.6 a REQ-29.8), el perfil de bug con sus tres rutas (REQ-29.15): bug con requisito existente (sin delta, con `Cubre:`), bug sin impacto en la spec (sin delta, sin `Cubre:`, con `## Impacto en la spec` y la línea literal `Sin impacto en la spec` más `Motivo:`; en ISO el motivo es la evidencia y la matriz vacía es válida, REQ-29.19) y bug con delta como cambio normal; el formato de `## Informe de bug` (`Observado:` y `Esperado:` obligatorias, `Referencia:` opcional); y la derivación de la traza (REQ-29.9). La guía de bug DEBE reproducir esos textos exactamente, porque el carril ODD usa la misma sección y la misma línea. Las skills describen los formatos; la validación reside en el motor. La doctrina `persistence-contract.md` DEBE conservar íntegra la regla «Missing or failed verification never gates archive» para los cambios no ISO y DEBE declarar la excepción del modo ISO (REQ-29.11) sin reescribir la regla general: en ISO, la matriz de verify válida y con éxito bloquea el archivado, y los hallazgos de revisión siguen siendo informativos. La skill `sdd-verify` DEBE indicar que, en un cambio ISO, la matriz es una sección adicional del informe sin líneas `verdict:`. La skill `sdd-archive` NO DEBE cambiar su descripción del flujo de archivado: la rama `-archive` y el PR documental aparte siguen siendo el único modo. Las skills y la guía de kickoff DEBEN dejar de indicar que se cree a mano la carpeta del cambio antes de `kickoff seal` (REQ-29.12). Toda esa guía DEBE salir de una fuente única, con pines de texto que verifiquen la paridad entre runtimes, y las copias versionadas y los goldens DEBEN regenerarse con la herramienta de actualización, nunca a mano.

#### Scenario: La doctrina conserva la regla general y declara la excepción ISO

- **DADO** el texto de `_shared/persistence-contract.md` proyectado
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** contiene la regla «Missing or failed verification never gates archive» para cambios no ISO
- **Y** declara que en un cambio ISO la matriz de verify válida y con éxito bloquea el archivado y que los hallazgos de revisión no lo bloquean

#### Scenario: Paridad de la guía en todos los runtimes

- **DADO** cada runtime soportado y su orquestador proyectado
- **CUANDO** se renderizan los activos mediante el renderizador real
- **ENTONCES** todas las variantes contienen la misma guía de `Cubre:`, matriz, perfil de bug (tres rutas) y modo ISO
- **Y** la excepción ISO figura en todas

#### Scenario: La guía de bug describe las tres rutas y los textos literales

- **DADO** el texto de la guía de bug de las skills y de la doctrina compartida
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** describe las rutas con requisito existente, sin impacto en la spec y con delta
- **Y** contiene literalmente `## Informe de bug`, `Observado:`, `Esperado:`, `Referencia:`, `## Impacto en la spec`, `Sin impacto en la spec` y `Motivo:`
- **Y** indica que en ISO el motivo es la evidencia del bug sin impacto y que la matriz vacía es válida

#### Scenario: La skill de verify describe la matriz sin líneas `verdict:`

- **DADO** el texto de `sdd-verify` y de su referencia del formato de informe
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** describen la matriz como una sección adicional que convive con «Cumplimiento técnico»
- **Y** prohíben escribir en ella líneas que empiecen por `verdict:`
- **Y** nombran las dos formas de referencia de prueba, el comando y el código de salida de las filas automáticas y la justificación de las manuales

#### Scenario: La skill de tareas describe `Cubre:`

- **DADO** el texto de `sdd-tasks`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** describe la línea `Cubre: REQ-x[, REQ-y ...]` y que en un cambio ISO cada requisito de la delta debe figurar en alguna

#### Scenario: La skill de archive conserva su flujo

- **DADO** el texto de `sdd-archive/SKILL.md`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** sigue indicando la rama `-archive` y el PR documental aparte
- **Y** no describe ningún archivado en la rama del cambio

#### Scenario: La guía de kickoff no pide crear la carpeta

- **DADO** los textos de las skills y plantillas que describen el sellado
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** no indican crear a mano la carpeta del cambio antes de `axiom sdd kickoff seal`

#### Scenario: El contrato del status documenta la proyección ISO

- **DADO** `_shared/sdd-status-contract.md`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** documenta la proyección de `sdd.iso`, los motivos tipados de ISO (REQ-29.16) y que un cambio sin instantánea es no ISO

#### Scenario: Los goldens y las copias versionadas están en paridad

- **DADO** la guía actualizada
- **CUANDO** se regeneran los goldens y las copias versionadas con la herramienta de actualización
- **ENTONCES** las pruebas de paridad pasan sin ediciones manuales de los goldens
