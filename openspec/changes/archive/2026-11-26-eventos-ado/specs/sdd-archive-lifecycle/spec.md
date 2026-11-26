# Delta para Archivado del Ciclo de Vida SDD (sdd-archive-lifecycle)

> **Cambio:** `eventos-ado` (C-07-e). **Idioma:** castellano de España; identificadores, comandos y claves en inglés.
> Con tracker activo, el archivado reenvía el resumen con los IDs definitivos y, en `continuous`, el resumen final recoge lo escrito en el tracker. La carpeta archivada sigue siendo inmutable (RF-ARC-03): estos pasos leen de ella y NO la modifican; `summary push` no escribe `integrations.yaml` (RF-INT-21).

## ADDED Requirements

### Requirement: Con Tracker Activo el Archivado Reenvía el Resumen con los IDs Definitivos (RF-ARC-16)

Con un vínculo `kind: work_item` y un plugin de work items activo, cuando la fase `archive` termina, la doctrina DEBE ordenar `axiom integrations summary push --change <cambio>` sobre la carpeta archivada, para que la descripción y las etiquetas de IDs funcionales del WI pasen de los identificadores provisionales a los definitivos asignados al archivar (RF-ARC-15). El verbo DEBE leer los definitivos de la carpeta archivada y NO DEBE dejar ningún provisional en el WI. La transición del WI al estado de archivado, si está configurada, la aplica el evento `archive` (RF-INT-19) y no este paso. Un fallo del reenvío NO DEBE deshacer el archivado ni impedir su `done`: se informa en el informe de archivado con el comando de reintento. El paso NO DEBE modificar ningún fichero de la carpeta archivada ni escribir `integrations.yaml`. Sin tracker activo, el archivado NO DEBE invocar el verbo ni mencionarlo.

#### Scenario: IDs definitivos tras archivar
- **DADO** un cambio archivado cuyo WI llevaba la etiqueta `req:P1` y cuyo `ids.yaml` asigna `D1` a `P1`
- **CUANDO** termina `archive` y se ejecuta `summary push --change x`
- **ENTONCES** el WI lleva la etiqueta `req:D1`, no `req:P1`, y su descripción contiene `D1`

#### Scenario: La carpeta archivada no cambia
- **DADO** una carpeta archivada con sus bytes registrados
- **CUANDO** se ejecuta `summary push --change x`
- **ENTONCES** ningún fichero de la carpeta archivada cambia y `integrations.yaml` no se escribe

#### Scenario: Fallo del reenvío
- **DADO** un `summary push` que falla con el cambio ya archivado
- **CUANDO** termina `archive`
- **ENTONCES** el archivado sigue válido y el informe de archivado nombra `axiom integrations summary push --change x` como reintento

#### Scenario: Sin tracker
- **DADO** un cambio sin plugin de work items activo
- **CUANDO** termina `archive`
- **ENTONCES** no se invoca `summary push`

### Requirement: Resumen Final de `continuous` con Estimaciones, Horas, Quién y Cuándo (RF-ARC-17)

En un cambio sellado `continuous` con tracker activo, el resumen final (el informe de cierre de la fase `archive`) DEBE incluir una sección con, por cada Task vinculada: el rol, el identificador y la URL de la Task, la estimación (`estimate_hours`), las horas imputadas (suma de los minutos de sus imputaciones), y por cada imputación quién (`actor`), cuánto (horas) y cuándo (`date`). DEBE obtenerse de `integrations.yaml` sin consultar la red. DEBE incluir también, como pendientes o fallidos, los avisos y rechazos que la ejecución sin paradas recogió (`req_tags_over_limit`, un `rev_conflict`, un estado de cierre inexistente, una Task sin imputar). Un rol sin imputaciones DEBE aparecer con cero horas imputadas y no omitirse. En un cambio `checkpointed` o sin tracker activo esta sección NO DEBE añadirse.

#### Scenario: Resumen con una imputación
- **DADO** un cambio `continuous` con la Task 205 del rol `fullstack` con estimación de 6 horas y una imputación de 120 minutos de `Ana <ana@x.test>` con fecha `2026-11-25`
- **CUANDO** se compone el resumen final
- **ENTONCES** la sección lista el rol `fullstack`, la Task 205, 6 horas estimadas, 2 horas imputadas, el actor y la fecha

#### Scenario: Dos cierres del mismo rol
- **DADO** dos imputaciones del rol `fullstack` de 60 y 90 minutos
- **CUANDO** se compone el resumen final
- **ENTONCES** lista ambas con su actor y su fecha y 2,5 horas imputadas en total

#### Scenario: Rol sin imputación
- **DADO** una Task vinculada sin imputaciones
- **CUANDO** se compone el resumen final
- **ENTONCES** aparece con 0 horas imputadas

#### Scenario: Avisos recogidos
- **DADO** un `req_tags_over_limit` y un `rev_conflict` registrados durante la ejecución sin paradas
- **CUANDO** se compone el resumen final
- **ENTONCES** ambos aparecen como pendientes con su comando de resolución

#### Scenario: Cambios que no lo llevan
- **DADO** un cambio `checkpointed`, o un cambio sin tracker activo
- **CUANDO** se compone el resumen final
- **ENTONCES** no incluye la sección

#### Scenario: Sin red
- **DADO** un resumen final compuesto sin acceso al tracker
- **CUANDO** se genera
- **ENTONCES** el contenido sale de `integrations.yaml` y no se hace ninguna petición
