# Delta para sdd-archive-lifecycle

> Alcance reducido: esta delta solo añade las dos reglas ISO del archivado. El modo de archivado (la rama `<prefijo>/<cambio>-archive` y el PR documental aparte), la precondición de integración (REQ-21.16), REQ-25.39 y la publicación de documentación (REQ-27.14) NO cambian.

## ADDED Requirements

### Requirement: Al Menos un Escenario por Requisito en el Archivado de un Proyecto ISO (REQ-29.10)

En un cambio ISO (sellado con `iso`, REQ-29.1 de `sdd-iso-evidence` y REQ-29.13 de `sdd-lane-kickoff`), la composición del archivado (`archive` y `axiom sdd archive-compose` cuando resuelve el dominio en la baseline del proyecto (`openspec/specs/`) y puede determinar el cambio, por `--change` o porque la delta está en la carpeta de un cambio; supuesto visible) DEBE rechazar, sin escribir ningún documento y con un motivo tipado que nombre el dominio, la sección y el requisito, todo requisito ADDED o MODIFIED de la delta que no contenga al menos un escenario (un encabezado `#### Scenario:`). NO se aplica a los requisitos REMOVED ni RENAMED. `axiom sdd status` DEBE adelantar el rechazo con el mismo motivo tipado en cuanto la delta existe (REQ-29.16 de `axiom-sdd-cli-integration`). Fuera de un cambio ISO la composición NO cambia. Un cambio sellado sin `iso`, o sin instantánea, es un cambio no ISO y la regla NO se aplica. Cuando `axiom sdd archive-compose` se invoca con `--canonical` y `--delta` sin contexto de proyecto ni de cambio, no hay instantánea que consultar y la regla NO se aplica.

#### Scenario: Un ADDED sin escenarios se rechaza en ISO

- **DADO** un cambio ISO y una delta con un requisito ADDED sin ningún `#### Scenario:`
- **CUANDO** se ejecuta el archivado
- **ENTONCES** se rechaza con un motivo tipado que nombra el dominio, la sección ADDED y el requisito
- **Y** ningún documento de la baseline se modifica
- **Y** el cambio permanece en `openspec/changes/<cambio>/`

#### Scenario: Un MODIFIED sin escenarios se rechaza en ISO

- **DADO** un cambio ISO y una delta con un MODIFIED cuyo bloque no contiene escenarios
- **CUANDO** se ejecuta `axiom sdd archive-compose --cwd <raíz> --document <dominio> --delta <ruta>`
- **ENTONCES** se rechaza con el motivo tipado nombrando el requisito
- **Y** no se escribe el fichero de salida

#### Scenario: REMOVED y RENAMED no exigen escenarios

- **DADO** un cambio ISO y una delta con un REMOVED y un RENAMED, ambos sin escenarios
- **CUANDO** se ejecuta el archivado
- **ENTONCES** la regla de escenarios no los rechaza

#### Scenario: Status adelanta el rechazo

- **DADO** un cambio ISO con una delta con un ADDED sin escenarios, todavía sin archivar
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se proyecta el mismo motivo tipado que nombra el dominio, la sección y el requisito

#### Scenario: Fuera de ISO el archivado no cambia

- **DADO** un cambio no ISO y una delta con un ADDED sin escenarios
- **CUANDO** se ejecuta el archivado
- **ENTONCES** la composición procede como antes de este cambio

#### Scenario: Sin contexto de proyecto la regla no se aplica

- **DADO** una invocación `axiom sdd archive-compose --canonical <ruta> --delta <ruta>` con un ADDED sin escenarios
- **CUANDO** se ejecuta
- **ENTONCES** la composición procede como antes de este cambio

---

### Requirement: En un Cambio ISO el Archivado Exige la Matriz de Verify Válida y con Éxito (REQ-29.11)

En un cambio ISO, `archive` DEBE bloquearse con un motivo tipado mientras la matriz de verify del `verify-report.md` esté ausente (salvo en un bug sin impacto en la spec, que no tiene escenarios afectados y cuya matriz ausente o vacía es válida, REQ-29.19 de `sdd-iso-evidence`), sea inválida (REQ-29.6 a REQ-29.8 de `sdd-iso-evidence`) o contenga alguna fila con resultado `fail`, con independencia del ritmo sellado (`continuous` o `checkpointed`). `axiom sdd status` DEBE reflejar el bloqueo en la dependencia `archive` nombrando la fila o el escenario afectado, `axiom sdd continue` NO DEBE autorizarla, y el escritor de `archive` DEBE negarse sin mover el cambio ni modificar la baseline. El bloqueo se **suma** a las demás precondiciones del archivado (integración o despliegue, rama y compuertas) y no sustituye a ninguna de ellas. Esta regla es una **excepción declarada** a la doctrina «Missing or failed verification never gates archive» (`persistence-contract.md`) y a REQ-28.3 de `sdd-independent-review`, que siguen vigentes íntegras para los cambios no ISO y para los hallazgos de revisión: en ISO, lo que bloquea es la validez de la matriz (evidencia declarada y contrastada con el árbol), nunca un hallazgo de verify, un juicio `ESCALATED` ni una revisión de cobertura. Fuera de un cambio ISO, un verify ausente, libre o fallido NO DEBE bloquear el archivado por sí mismo.

#### Scenario: Una matriz con una fila fallida bloquea el archivado en ISO

- **DADO** un cambio ISO con una matriz válida en la que una fila tiene resultado `fail`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json` en la fase `archive`
- **ENTONCES** el archivado queda bloqueado con el motivo tipado de matriz
- **Y** el motivo nombra la fila fallida

#### Scenario: Una matriz inválida bloquea el archivado en ISO aunque el cambio sea `continuous`

- **DADO** un cambio ISO sellado `continuous` y una matriz con un escenario sin fila
- **CUANDO** se solicita el archivado
- **ENTONCES** se bloquea con el motivo tipado de matriz
- **Y** el cambio permanece en `openspec/changes/<cambio>/`

#### Scenario: Una matriz ausente bloquea el archivado en ISO

- **DADO** un cambio ISO cuyo `verify-report.md` no contiene matriz
- **CUANDO** se solicita el archivado
- **ENTONCES** se bloquea con el motivo tipado de matriz ausente

#### Scenario: Un bug sin impacto en la spec archiva sin matriz

- **DADO** un cambio ISO declarado como bug sin impacto en la spec, con `Motivo:` no vacío, sin `verify-report.md` o con la matriz vacía, y el resto de precondiciones de archivado satisfechas
- **CUANDO** se solicita el archivado
- **ENTONCES** el archivado no se bloquea por la matriz
- **Y** el archivado no integra ningún requisito en la baseline

#### Scenario: Una referencia de prueba que no resuelve bloquea el archivado

- **DADO** un cambio ISO con una fila automática cuya referencia no resuelve a ningún símbolo del árbol de trabajo de la rama de archivado
- **CUANDO** se solicita el archivado
- **ENTONCES** se bloquea con el motivo tipado de matriz, que nombra la fila y la referencia

#### Scenario: Una matriz válida y con éxito permite el archivado en ISO

- **DADO** un cambio ISO con una matriz válida, todas las filas con resultado `pass`, y el resto de precondiciones de archivado satisfechas
- **CUANDO** se solicita el archivado
- **ENTONCES** el archivado no se bloquea por la matriz

#### Scenario: Una matriz válida no sustituye las demás precondiciones

- **DADO** un cambio ISO con una matriz válida y con éxito, sin evidencia de integración o despliegue
- **CUANDO** se solicita el archivado
- **ENTONCES** se rechaza por la ausencia de integración, conforme a REQ-21.16

#### Scenario: Un hallazgo de verify no bloquea el archivado en ISO

- **DADO** un cambio ISO con una matriz válida y con éxito y un verify con un hallazgo de severidad crítica y un juicio `ESCALATED`
- **CUANDO** se solicita el archivado
- **ENTONCES** el archivado no se bloquea por esos hallazgos

#### Scenario: El escritor de `archive` se niega sin mover el cambio

- **DADO** un cambio ISO con una matriz inválida
- **CUANDO** se invoca el escritor de `archive`
- **ENTONCES** se niega con el motivo tipado de matriz
- **Y** no se mueve el cambio ni se modifica ningún documento de la baseline

#### Scenario: Fuera de ISO un verify fallido no bloquea el archivado

- **DADO** un cambio no ISO y un `verify-report.md` con veredicto fallido o ausente, y el resto de precondiciones satisfechas
- **CUANDO** se solicita el archivado
- **ENTONCES** el archivado no se bloquea por la verificación, como antes de este cambio
