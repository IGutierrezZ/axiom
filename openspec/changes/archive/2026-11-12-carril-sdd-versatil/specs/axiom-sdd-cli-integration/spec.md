# Delta para axiom-sdd-cli-integration

> Alcance reducido: esta delta no toca `blocked(wrong_branch)` (REQ-25.42) ni la rama esperada de la fase `archive`, y no introduce `story.md` ni ninguna validación de `design.md`.

## ADDED Requirements

### Requirement: Preparación del Cambio con Bug sin Delta (con Requisito Existente o sin Impacto en la Spec), Aceptando el Formato Actual (REQ-29.15)

`axiom sdd status` DEBE considerar cumplido el artefacto de specs del cambio cuando existe al menos una delta de spec, o cuando el cambio está declarado como bug sin delta. Esta ruta está disponible en todo proyecto, sea o no ISO. Un bug sin delta se declara en su `proposal.md` con una sección fija `## Informe de bug`, que es la propia declaración (no hay front matter), y admite dos variantes:

| Variante | Qué declara | Qué más lleva |
|---|---|---|
| Bug con requisito existente | Restaura un requisito que ya existe en la baseline. | `Cubre:` en las tareas con el requisito existente; en ISO se añaden las comprobaciones de REQ-29.5 de `sdd-iso-evidence`. |
| Bug sin impacto en la spec | No toca ningún requisito. | La sección `## Impacto en la spec` con la marca de abajo; sin delta y sin `Cubre:`; en ISO rige REQ-29.19 de `sdd-iso-evidence`. |

**Formato de `## Informe de bug`** (contrato fijo): un encabezado de nivel 2 con ese texto exacto y, en su contenido, una línea `Observado:` (comportamiento observado) y una línea `Esperado:` (comportamiento esperado), ambas obligatorias y con texto no vacío tras los dos puntos, y una línea `Referencia:` opcional (referencia externa). Cada línea empieza por su etiqueta y lleva su valor en la misma línea (supuesto visible). Falta de `Observado:` o de `Esperado:`, o valor vacío, deja el informe incompleto: no cuenta como declaración de bug.

**Formato de la marca de impacto** (contrato fijo, que otros carriles reutilizan tal cual): un encabezado de nivel 2 `## Impacto en la spec` cuyo contenido es la línea literal `Sin impacto en la spec` y una línea `Motivo: <una línea>` con texto no vacío. La línea `Sin impacto en la spec` se compara de forma exacta, sensible a mayúsculas y sin más texto en la línea, ignorando solo el espacio en blanco inicial y final (supuesto visible). La marca solo es válida junto a un `## Informe de bug` completo.

**Reglas de evaluación**, por orden de precedencia:

1. **Con delta.** Un cambio con delta se evalúa como cambio normal aunque declare ser un bug o lleve la sección `## Impacto en la spec`: la sección se ignora, sin aviso (supuesto visible).
2. **Sin delta y con la sección `## Impacto en la spec`.** Se evalúa como bug sin impacto en la spec: el artefacto de specs se cumple solo si el `## Informe de bug` está completo y la sección contiene la línea literal y un `Motivo:` no vacío. Si falta el informe, o está incompleto, o falta la línea literal, o falta el motivo o está vacío, el artefacto NO se cumple, el estado proyecta la fase `spec` pendiente y el motivo nombra qué falta (supuesto visible: una declaración incompleta nunca cuenta como declaración).
3. **Sin delta y sin esa sección.** Si hay un `## Informe de bug` completo, se evalúa como bug con requisito existente y el artefacto se cumple; si no hay informe, o está incompleto, el artefacto NO se cumple y el estado proyecta la fase `spec` pendiente, nombrando la línea que falta cuando el informe existe.

Un bug sin delta llega a la fase `archive` sin que la ausencia de delta la bloquee, y su archivado no integra ningún requisito en la baseline porque no hay nada que componer. Un bug cuyo requisito estaba mal especificado o faltaba NO es un bug sin delta: se tramita como cambio normal, con delta. Un cambio en el formato actual (`proposal.md` y delta) DEBE producir el mismo estado que antes de este cambio, sin migración. La proyección es solo aditiva: ninguna clave existente del status v2 cambia de nombre, tipo ni valor.

#### Scenario: El formato actual produce el mismo estado

- **DADO** un cambio con `proposal.md` y una delta de spec
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs se considera cumplido y el estado coincide con el anterior a este cambio

#### Scenario: Un bug con requisito existente queda preparado fuera de ISO

- **DADO** un cambio no ISO con un `## Informe de bug` completo (`Observado:` y `Esperado:`) en su `proposal.md`, sin delta y sin la sección `## Impacto en la spec`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs se considera cumplido sin exigir delta

#### Scenario: La referencia del informe es opcional

- **DADO** un `## Informe de bug` con `Observado:` y `Esperado:` y sin `Referencia:`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el informe se considera completo

#### Scenario: Un informe sin `Esperado:` o con valor vacío está incompleto

- **DADO** un cambio sin delta con `## Informe de bug` que contiene `Observado:` con texto y `Esperado:` vacío (o sin la línea `Esperado:`)
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs no se considera cumplido y se proyecta la fase `spec` pendiente
- **Y** el motivo nombra la línea `Esperado:`

#### Scenario: Un cambio sin delta y sin declaración de bug no está preparado

- **DADO** un cambio con `proposal.md`, sin delta y sin `## Informe de bug`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs no se considera cumplido y se proyecta la fase pendiente

#### Scenario: Un cambio con delta que declara ser un bug se trata como cambio normal

- **DADO** un cambio con `## Informe de bug` que además contiene una delta de spec
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se evalúa como un cambio con delta
- **Y** la cobertura `Cubre:` de ISO, si aplica, se calcula sobre los requisitos de esa delta

#### Scenario: Un bug sin impacto en la spec queda preparado en cualquier proyecto

- **DADO** un cambio, ISO o no, con `## Informe de bug` completo y una sección `## Impacto en la spec` con las líneas `Sin impacto en la spec` y `Motivo: el defecto estaba en el script de despliegue`, sin delta y sin `Cubre:`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs se considera cumplido sin exigir delta

#### Scenario: La marca sin `Motivo:` no prepara el cambio

- **DADO** un cambio sin delta con `## Informe de bug` completo y `## Impacto en la spec` que contiene solo `Sin impacto en la spec`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs no se considera cumplido y se proyecta la fase `spec` pendiente
- **Y** el motivo nombra la línea `Motivo:` que falta

#### Scenario: La marca con motivo vacío no prepara el cambio

- **DADO** la misma declaración con la línea `Motivo:` sin texto
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs no se considera cumplido
- **Y** el motivo nombra la línea `Motivo:` vacía

#### Scenario: La sección sin la línea literal no prepara el cambio

- **DADO** un cambio sin delta con `## Informe de bug` completo y `## Impacto en la spec` con `Impacto: ninguno` y `Motivo: ninguno`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs no se considera cumplido
- **Y** el motivo nombra la línea literal `Sin impacto en la spec` que falta

#### Scenario: El texto de la marca debe ser exactamente el literal

- **DADO** una sección `## Impacto en la spec` con `Sin impacto en la spec.` (con punto final) o con `sin impacto en la spec`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** la línea no se reconoce como la marca y el artefacto de specs no se considera cumplido

#### Scenario: `## Impacto en la spec` sin `## Informe de bug` no prepara el cambio

- **DADO** un cambio sin delta con `## Impacto en la spec` válida y sin `## Informe de bug`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs no se considera cumplido y se proyecta la fase `spec` pendiente
- **Y** el motivo nombra la sección `## Informe de bug` que falta

#### Scenario: La marca junto a una delta se ignora y se evalúa como cambio con delta

- **DADO** un cambio con `## Informe de bug`, `## Impacto en la spec` válida y una delta de spec
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se evalúa como un cambio con delta y el artefacto de specs se considera cumplido por la delta
- **Y** la sección `## Impacto en la spec` no genera ningún motivo

#### Scenario: Un bug sin delta llega al archivado sin integrar nada

- **DADO** un bug sin delta, con requisito existente o sin impacto en la spec, con sus fases completadas y el resto de precondiciones de archivado satisfechas
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json` en la fase `archive`
- **ENTONCES** la ausencia de delta no bloquea el archivado
- **Y** el archivado no modifica ningún documento de la baseline

#### Scenario: El informe del bug va en `proposal.md`

- **DADO** un bug sin delta
- **CUANDO** se inspeccionan los artefactos del cambio
- **ENTONCES** `proposal.md` contiene la sección `## Informe de bug` con el comportamiento observado y el esperado
- **Y** no existe ningún otro artefacto de informe de bug ni front matter que lo declare

#### Scenario: El contrato del status solo crece

- **DADO** el status de un cambio con delta
- **CUANDO** se compara con el de antes de este cambio
- **ENTONCES** ninguna clave existente cambia de nombre, tipo ni valor

---

### Requirement: Motivos de Bloqueo Tipados de ISO, Cobertura, Escenarios y Matriz en `status` (REQ-29.16)

`axiom sdd status` DEBE proyectar, con un motivo tipado que nombre el elemento afectado y la continuación, los bloqueos de un cambio ISO (sellado con `iso`, REQ-29.13 de `sdd-lane-kickoff`; las reglas se leen de su instantánea y no de `axiom.yaml` en vivo) definidos en `sdd-iso-evidence` y `sdd-archive-lifecycle`: identificador `REQ-<n>.<m>` ausente, ambiguo o repetido en el nombre de los requisitos de la delta (REQ-29.3), cobertura `Cubre:` y `Cubre:` a un requisito inexistente (REQ-29.4), requisito de un bug con requisito existente inexistente, sin citar o sin escenarios (REQ-29.5), `Cubre:` en un bug sin impacto en la spec (REQ-29.19), requisito ADDED o MODIFIED sin escenarios (REQ-29.10), y matriz de verify ausente, inválida o con filas fallidas en la fase `archive` (REQ-29.6 a REQ-29.8 y REQ-29.11). Fuera de un cambio ISO, ninguna de esas reglas DEBE proyectar ni bloqueo ni aviso. `axiom sdd continue` DEBE reflejar cualquier bloqueo sin autorizar la fase siguiente. Los nuevos motivos son solo aditivos: ninguna clave existente del status v2 cambia, y el orden de precedencia de los motivos existentes (por ejemplo, `wrong_branch` antes de `edit_authority_missing`) se conserva. Cuando concurren varios bloqueos, el status DEBE proyectar todos con su continuación, no solo el primero.

#### Scenario: Un bloqueo ISO nombra el elemento y la continuación

- **DADO** un cambio ISO y un requisito de la delta sin cobertura
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se proyecta el motivo tipado de cobertura nombrando el requisito
- **Y** incluye la continuación (añadir un `Cubre:` en alguna tarea)

#### Scenario: Un requisito sin escenarios se proyecta antes del archivado

- **DADO** un cambio ISO y una delta con un ADDED sin escenarios
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se proyecta el motivo tipado de escenarios nombrando el dominio, la sección y el requisito

#### Scenario: Una matriz inválida bloquea la dependencia `archive`

- **DADO** un cambio ISO en la fase `archive` con una matriz con un escenario sin fila
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** la dependencia `archive` aparece bloqueada con el motivo tipado de matriz que nombra el escenario
- **Y** incluye la continuación (completar la matriz del `verify-report.md`)

#### Scenario: Fuera de ISO la cobertura y la matriz no proyectan nada

- **DADO** un cambio no ISO, requisitos sin `Cubre:` y un `verify-report.md` libre
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta ni bloqueo ni aviso de cobertura o de matriz

#### Scenario: `axiom sdd continue` refleja el bloqueo ISO

- **DADO** un cambio ISO bloqueado por cobertura
- **CUANDO** se ejecuta `axiom sdd continue <cambio>`
- **ENTONCES** no emite ninguna fase siguiente autorizada
- **Y** reporta el mismo motivo tipado

#### Scenario: Varios bloqueos concurrentes se proyectan todos

- **DADO** un cambio ISO con un requisito ADDED sin identificador `REQ-<n>.<m>` en su nombre y otro requisito sin cobertura
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se proyectan los motivos de identificador y de cobertura, cada uno con su continuación

#### Scenario: Un `Cubre:` junto a la marca de bug sin impacto se proyecta como motivo tipado

- **DADO** un cambio ISO declarado como bug sin impacto en la spec y una tarea con `Cubre: REQ-13.1`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se proyecta el motivo tipado de declaración contradictoria, con el fichero de tareas, el requisito y la continuación

#### Scenario: La precedencia de los motivos existentes se conserva

- **DADO** un cambio ISO bloqueado por `wrong_branch` y por cobertura
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** `wrong_branch` se proyecta con la precedencia que ya tenía
- **Y** el motivo de cobertura se proyecta además

#### Scenario: El contrato del status solo crece

- **DADO** el status de un cambio no ISO
- **CUANDO** se compara con el de antes de este cambio
- **ENTONCES** ninguna clave existente cambia de nombre, tipo ni valor
