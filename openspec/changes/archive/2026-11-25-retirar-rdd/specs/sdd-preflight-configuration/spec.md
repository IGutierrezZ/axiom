# Delta for sdd-preflight-configuration

## MODIFIED Requirements

### Requirement: El Pre-Vuelo y su Bloque Derivado no Contienen Ningún Umbral Numérico de Entrega (REQ-28.23)

El texto del cuestionario de pre-vuelo proyectado a los orquestadores, el bloque derivado que el hook y el plugin de OpenCode (en sus dos copias) anteponen al prompt de cada fase y los comandos `sdd-*` de OpenCode que describen el contenido exigido del pre-vuelo NO DEBEN contener ningún número de líneas como umbral, presupuesto o política de revisión de la entrega, ni la expresión «chained PR strategy and review budget». La entrega del pre-vuelo es siempre `single-pr`: `chained` solo se activa cuando la persona lo pide durante la sesión, fuera del pre-vuelo, y no altera la línea `Delivery strategy` del bloque (la precedencia de un `delivery_strategy: chained` reenviado por el orquestador se fija en `sdd-orchestrator-assets`). Quedan fuera de este requisito, por no ser umbrales de entrega, los literales de migración de prompts antiguos de OpenCode de `internal/components/sdd/inject.go`, el umbral de barrido de los agentes `jd-judge-a` y `jd-judge-b` de judgment-day.
(Previously: la lista de exclusiones citaba también las lentes `review-*` de RDD y el umbral de riesgo del paquete de revisión de RDD; RDD se retira (C-01-e, P20).)

#### Scenario: El texto y el bloque del pre-vuelo no contienen un umbral de entrega

- **DADO** el texto del cuestionario proyectado, el bloque derivado del hook y el de cada copia del plugin
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** ninguno contiene `400` como umbral de líneas ni la expresión `Review policy`
- **Y** el bloque contiene `Delivery strategy: single-pr`

#### Scenario: Los comandos de OpenCode no exigen presupuesto de revisión en el pre-vuelo

- **DADO** los nueve comandos `sdd-*` de OpenCode
- **CUANDO** se inspecciona el contenido exigido del pre-vuelo
- **ENTONCES** piden modo de ejecución y almacén de artefactos
- **Y** no exigen estrategia de PR encadenada ni presupuesto de revisión

#### Scenario: Un valor heredado de entrega en un bloque no se acepta como bloque derivado

- **DADO** un prompt que ya trae un bloque con `Delivery strategy: ask-on-risk`
- **CUANDO** el hook evalúa el despacho
- **ENTONCES** no lo trata como un bloque derivado válido ya presente
- **Y** antepone el bloque derivado vigente con `Delivery strategy: single-pr`
