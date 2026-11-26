# Delta para Ciclo de Vida del Archivado (sdd-archive-lifecycle)

> **Cambio:** `marcas-pruebas` (C-06-d). **Idioma:** castellano de España; identificadores, claves y tokens en inglés.

## ADDED Requirements

### Requirement: El `archive-report` Incluye el Resumen de Marcas y las Marcas Nunca Bloquean el Archivado (RF-ARC-18)

Cuando el cambio tiene `pruebas-estado.yaml` con entradas, el `archive-report` DEBE incluir una sección con el resumen de marcas por entorno (casos `ok` vigentes sobre el total aplicable y marcas caducadas aparte), calculado con el mismo cálculo que el status (RF-KOF-22). Sin marcas, el informe NO DEBE incluir la sección. El archivado NO DEBE bloquearse, rechazarse ni condicionarse por marcas ausentes, `ko`, `blocked`, caducadas o por un `pruebas-estado.yaml` ilegible; un fichero ilegible se declara como aviso. El `pruebas-estado.yaml` DEBE archivarse junto con el resto de la carpeta del cambio, sin modificarlo. El resumen NO DEBE cambiar `nextRecommended`, dependencias ni digests.

#### Scenario: El informe incluye el resumen
- **DADO** un cambio con marcas vigentes `pre: 5/5 ok`
- **CUANDO** se archiva
- **ENTONCES** el `archive-report` contiene la sección de marcas con `pre: 5/5 ok`

#### Scenario: Marcas `ko` o caducadas no bloquean
- **DADO** un cambio con una marca `ko` y otra caducada
- **CUANDO** se archiva
- **ENTONCES** el archivado termina con éxito y el informe refleja ambas

#### Scenario: Sin marcas no hay sección
- **DADO** un cambio sin `pruebas-estado.yaml`
- **CUANDO** se archiva
- **ENTONCES** el `archive-report` no incluye la sección de marcas

#### Scenario: Fichero ilegible
- **DADO** un `pruebas-estado.yaml` con YAML inválido
- **CUANDO** se archiva
- **ENTONCES** el archivado termina con éxito y el informe declara un aviso por el fichero ilegible

#### Scenario: El fichero se archiva intacto
- **DADO** un `pruebas-estado.yaml` con entradas
- **CUANDO** se archiva
- **ENTONCES** la carpeta archivada contiene el mismo fichero byte a byte
