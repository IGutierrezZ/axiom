# Delta para workspace-topology

## MODIFIED Requirements

### Requirement: Plantilla y Manual Documentan `integrations:` (REQ-43.21)

`axiom.example.yaml` DEBE incluir un bloque `integrations:` comentado que contenga el bloque `integrations.azure-devops` con las claves `organization`, `project` y `team`, y `docs/manual-de-inicio.md` (sección 6) DEBE incluir un apartado `integrations:` que describa el bloque, el catálogo de plugins (`azure-devops`), el secreto `pat` y su variable `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`, los scopes mínimos del PAT por operación (`vso.work_write`, `vso.project` y `vso.code`), que el PAT es de una sola organización y que los secretos nunca se escriben en `axiom.yaml`, y los verbos `axiom integrations`, incluidos `connect` y `discover`. Las pruebas de contrato (`example_config_test.go`, `docs_contract_test.go`) DEBEN fijar ambos y NO DEBEN afirmar que el catálogo está vacío.
(Previously: la plantilla y el manual describían un bloque genérico con el catálogo vacío hasta C-07-c, sin el bloque `azure-devops`, scopes, secreto `pat` ni los verbos `connect` y `discover`)

#### Scenario: Plantilla y manual cubiertos por las pruebas de contrato
- **CUANDO** se ejecutan las pruebas de contrato de la plantilla y del manual
- **ENTONCES** comprueban la presencia del bloque comentado y del apartado, y fallan si se eliminan

#### Scenario: La plantilla documenta `azure-devops`
- **DADO** `axiom.example.yaml`
- **CUANDO** se descomenta el bloque `integrations.azure-devops` y se carga con el catálogo por defecto
- **ENTONCES** la carga termina sin error y el bloque declara `organization`, `project` y `team`

#### Scenario: El manual documenta secreto, scopes y verbos
- **DADO** `docs/manual-de-inicio.md`
- **CUANDO** se ejecuta la prueba de contrato del manual
- **ENTONCES** comprueba que el apartado menciona `pat`, `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`, `vso.work_write`, `vso.project`, `vso.code`, `integrations connect` e `integrations discover`
- **Y** la prueba falla si alguno falta

#### Scenario: Ya no se afirma catálogo vacío
- **DADO** la prueba de contrato de la plantilla
- **CUANDO** se ejecuta
- **ENTONCES** no exige que el catálogo de plugins esté vacío ni que el bloque descomentado sea rechazado
