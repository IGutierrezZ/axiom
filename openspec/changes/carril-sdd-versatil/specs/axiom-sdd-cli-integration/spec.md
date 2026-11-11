# Delta para axiom-sdd-cli-integration

> Alcance reducido: esta delta no toca `blocked(wrong_branch)` (REQ-25.42) ni la rama esperada de la fase `archive`, y no introduce `story.md` ni ninguna validación de `design.md`.

## ADDED Requirements

### Requirement: Preparación del Cambio con Bug sin Delta, Aceptando el Formato Actual (REQ-29.15)

`axiom sdd status` DEBE considerar cumplido el artefacto de specs del cambio cuando existe al menos una delta de spec, o cuando el cambio está declarado como bug sin delta. Un bug sin delta se declara en su `proposal.md` con el informe del bug (comportamiento observado, comportamiento esperado y, de forma opcional, una referencia externa); la forma exacta de la declaración la fija el diseño (pregunta abierta Q3b). Un bug sin delta restaura un requisito que ya existe en la baseline y lo cita con `Cubre:` en las tareas, de modo que NO se exige delta. Esta ruta está disponible en todo proyecto, sea o no ISO; en un cambio ISO se añaden las comprobaciones de REQ-29.5 de `sdd-iso-evidence`. Un cambio con delta se evalúa como cambio normal aunque declare ser un bug. Un bug cuyo requisito estaba mal especificado o faltaba NO es un bug sin delta: se tramita como cambio normal, con delta. Un cambio sin delta que no está declarado como bug sin delta NO DEBE considerarse preparado, y el estado proyecta la fase pendiente correspondiente. Un bug sin delta llega a la fase `archive` sin que la ausencia de delta la bloquee, y su archivado no integra ningún requisito en la baseline porque no hay nada que componer. Un cambio en el formato actual (`proposal.md` y delta) DEBE producir el mismo estado que antes de este cambio, sin migración. La proyección es solo aditiva (REQ-24.20 de `spec-baseline-declaration`): ninguna clave existente del status v2 cambia de nombre, tipo ni valor.

#### Scenario: El formato actual produce el mismo estado

- **DADO** un cambio con `proposal.md` y una delta de spec
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs se considera cumplido y el estado coincide con el anterior a este cambio

#### Scenario: Un bug sin delta queda preparado fuera de ISO

- **DADO** un cambio no ISO declarado como bug sin delta en su `proposal.md`, sin ninguna delta
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs se considera cumplido sin exigir delta

#### Scenario: Un cambio sin delta y sin declaración de bug no está preparado

- **DADO** un cambio con `proposal.md`, sin delta y sin declaración de bug
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs no se considera cumplido y se proyecta la fase pendiente

#### Scenario: Un cambio con delta que declara ser un bug se trata como cambio normal

- **DADO** un cambio declarado como bug que además contiene una delta de spec
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se evalúa como un cambio con delta
- **Y** la cobertura `Cubre:` de ISO, si aplica, se calcula sobre los requisitos de esa delta

#### Scenario: Un bug sin delta llega al archivado sin integrar nada

- **DADO** un bug sin delta con sus fases completadas y el resto de precondiciones de archivado satisfechas
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json` en la fase `archive`
- **ENTONCES** la ausencia de delta no bloquea el archivado
- **Y** el archivado no modifica ningún documento de la baseline

#### Scenario: El informe del bug va en `proposal.md`

- **DADO** un bug sin delta
- **CUANDO** se inspeccionan los artefactos del cambio
- **ENTONCES** `proposal.md` contiene el comportamiento observado y el esperado
- **Y** no existe ningún otro artefacto de informe de bug

#### Scenario: El contrato del status solo crece

- **DADO** el status de un cambio con delta
- **CUANDO** se compara con el de antes de este cambio
- **ENTONCES** ninguna clave existente cambia de nombre, tipo ni valor

---

### Requirement: Motivos de Bloqueo Tipados de ISO, Cobertura, Escenarios y Matriz en `status` (REQ-29.16)

`axiom sdd status` DEBE proyectar, con un motivo tipado que nombre el elemento afectado y la continuación, los bloqueos de un cambio ISO (sellado con `iso`, REQ-29.13 de `sdd-lane-kickoff`; las reglas se leen de su instantánea y no de `axiom.yaml` en vivo) definidos en `sdd-iso-evidence` y `sdd-archive-lifecycle`: identidad `id` de los documentos afectados (REQ-29.3), cobertura `Cubre:` y `Cubre:` a un requisito inexistente (REQ-29.4), requisito de un bug sin delta inexistente, sin citar o sin escenarios (REQ-29.5), requisito ADDED o MODIFIED sin escenarios (REQ-29.10), y matriz de verify ausente, inválida o con filas fallidas en la fase `archive` (REQ-29.6 a REQ-29.8 y REQ-29.11). Fuera de un cambio ISO, ninguna de esas reglas DEBE proyectar ni bloqueo ni aviso. `axiom sdd continue` DEBE reflejar cualquier bloqueo sin autorizar la fase siguiente. Los nuevos motivos son solo aditivos: ninguna clave existente del status v2 cambia, y el orden de precedencia de los motivos existentes (por ejemplo, `wrong_branch` antes de `edit_authority_missing`) se conserva. Cuando concurren varios bloqueos, el status DEBE proyectar todos con su continuación, no solo el primero.

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

- **DADO** un cambio ISO con un documento afectado con `identity: text` y un requisito sin cobertura
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se proyectan los motivos de identidad y de cobertura, cada uno con su continuación

#### Scenario: La precedencia de los motivos existentes se conserva

- **DADO** un cambio ISO bloqueado por `wrong_branch` y por cobertura
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** `wrong_branch` se proyecta con la precedencia que ya tenía
- **Y** el motivo de cobertura se proyecta además

#### Scenario: El contrato del status solo crece

- **DADO** el status de un cambio no ISO
- **CUANDO** se compara con el de antes de este cambio
- **ENTONCES** ninguna clave existente cambia de nombre, tipo ni valor
