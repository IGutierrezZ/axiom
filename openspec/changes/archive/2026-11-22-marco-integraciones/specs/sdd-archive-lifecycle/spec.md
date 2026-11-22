# Delta para sdd-archive-lifecycle

> Parte del estado vigente: el resto de requisitos NO cambia. Esta delta acota la inmutabilidad post-archive (REQ-21.18) para que la cola local de integraciones (C-07-b, capacidad `integration-plugins`, REQ-43.28) pueda escribir estado local con punto en la raíz de la carpeta archivada, sin reabrir el contenido versionado del incremento. Decisión B-1 del usuario (opción B).

## MODIFIED Requirements

### Requirement: Sellado Inmutable Post-Archive y Gestión Exclusiva vía Bug o Nuevo Incremento (REQ-21.18)

Una vez archivado un incremento, su alcance queda formalmente congelado. El sistema NO DEBE reabrir un incremento ya archivado para aplicar un ajuste, un comportamiento anómalo o una regresión detectados con posterioridad. Cualquier corrección posterior DEBE gestionarse mediante un ticket de bug independiente o un nuevo incremento de evolución que referencie la especificación viva ya sellada, nunca modificando directamente el contenido versionado bajo `openspec/changes/archive/`. El estado local de la carpeta archivada, es decir, los ficheros cuyo nombre empieza por punto situados directamente en su raíz, que nunca se versionan, no forma parte del contenido del incremento y su escritura no lo reabre. El único escritor de Axiom autorizado a escribir estado local en la raíz de una carpeta archivada es la cola de salida de integraciones (`.integrations-outbox.yaml` y sus temporales de escritura atómica). Ningún otro verbo DEBE escribir bajo `openspec/changes/archive/`, e `integrations.yaml` archivado NO DEBE modificarse.
(Previously: prohibía modificar cualquier contenido bajo `openspec/changes/archive/`, sin distinguir el contenido versionado del estado local con punto de la raíz de la carpeta archivada.)

#### Scenario: Intento de reabrir un incremento archivado se rechaza

- **DADO** un incremento ya presente en `openspec/changes/archive/`
- **CUANDO** se intenta reabrirlo o modificarlo directamente para aplicar un ajuste
- **ENTONCES** el sistema rechaza la operación
- **Y** señala que la vía correcta es un ticket de bug o un nuevo incremento

#### Scenario: Regresión post-archive se gestiona mediante bug o nuevo incremento

- **DADO** que se detecta una regresión en un comportamiento definido por un incremento ya archivado
- **CUANDO** se gestiona esa regresión
- **ENTONCES** se abre un ticket de bug o un nuevo incremento de evolución que referencia la especificación viva afectada
- **Y** el contenido del incremento archivado original permanece sin modificaciones

#### Scenario: La cola de integraciones escribe estado local en la carpeta archivada

- **DADO** un incremento archivado y un plugin configurado que escucha `archive`
- **CUANDO** `sdd publish --archive` termina sin rechazo y la entrega falla
- **ENTONCES** existe `.integrations-outbox.yaml` en la raíz de la carpeta archivada
- **Y** ningún fichero versionado de la carpeta archivada cambia, incluido `integrations.yaml`

#### Scenario: Ningún otro escritor abre la excepción

- **DADO** un incremento archivado
- **CUANDO** un verbo de Axiom intenta escribir en su carpeta archivada cualquier fichero distinto de la cola de integraciones y sus temporales, incluidos `integrations.yaml`, `gates.yaml` u otro fichero con punto
- **ENTONCES** se rechaza con el mensaje de REQ-21.18
