# Delta para project-technical-context

> Parte del estado vigente: el resto de requisitos NO cambia. `axiom workspace validate` pasa a reportar `artifact_store` `engram` o `none` aunque no haya contexto técnico mantenido (REQ-2.6 de `workspace-topology`); lo que exige este requisito al contexto mantenido, el bloqueo del archivado en el status y la parada de archive no cambian.

## MODIFIED Requirements

### Requirement: El Contexto Técnico Mantenido Exige un Almacén de Artefactos de Ficheros (REQ-30.19)

Un contexto técnico mantenido (`technical_context.maintained`) DEBE usarse con un almacén de ficheros (`openspec` o `hybrid`), porque la integración de archive lee y escribe ficheros de la carpeta del cambio: el delta de contexto (REQ-24.31) y la traza (REQ-30.18). El almacén es el que declara `artifact_store` en `openspec/config.yaml` y, si no se declara, el que resuelve el status para el cambio. Con `engram` o `none`:

- `axiom workspace validate` DEBE reportar un error en español que nombre `technical_context.maintained`, el almacén declarado y la continuación (declarar `artifact_store: openspec` o `hybrid`, o retirar la declaración), y finalizar con código de salida `1`. La validación solo evalúa el valor declarado: sin `artifact_store` declarado no reporta nada por este motivo. Con independencia del contexto mantenido, `axiom workspace validate` reporta además cualquier `artifact_store` `engram` o `none` (REQ-2.6 de `workspace-topology`); cuando hay contexto mantenido, el error nombra también `technical_context.maintained`.
- El status DEBE marcar el archivado como bloqueado (`dependencies.archive` en `blocked`) con esa causa y la continuación. Las dependencias de las fases de planificación y de `apply` NO DEBEN pasar a `blocked` por ella y el status sigue proyectando `technicalContext`.
- Archive DEBE detenerse (REQ-24.56, caso e).

Una declaración solo de contexto de referencia (`technical_context.reference`) y la ausencia de contexto mantenido NO exigen un almacén de ficheros para el status ni para el archivado (aunque `axiom workspace validate` siga reportando `engram` y `none` por REQ-2.6 de `workspace-topology`). (Este requisito es la parte de contexto de REQ-24.62 de `spec-baseline-declaration`.)
(Previously: la validación no reportaba `engram` ni `none` sin contexto mantenido, y el escenario «Sin contexto mantenido el almacén no se exige» decía que no se reportaba error por el almacén)

#### Scenario: Contexto mantenido con almacén `none`
- **DADO** un `axiom.yaml` con `technical_context.maintained` válido y un `openspec/config.yaml` con `artifact_store: none`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta en español que el contexto mantenido exige un almacén de ficheros, nombrando `technical_context.maintained`, `none` y la continuación
- **Y** el código de salida es `1`

#### Scenario: Los almacenes de ficheros y la ausencia de declaración no se reportan
- **DADO** la misma declaración con `artifact_store: openspec`, con `hybrid` y sin `artifact_store` declarado
- **CUANDO** se ejecuta `axiom workspace validate` en cada caso
- **ENTONCES** ninguno reporta error por el almacén

#### Scenario: El status bloquea el archivado sin bloquear la planificación
- **DADO** un contexto mantenido, `artifact_store: engram` declarado y un cambio resuelto en ese almacén
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `dependencies.archive` es `blocked` con una causa que nombra el almacén `engram` y la continuación
- **Y** `technicalContext` sigue proyectado y las dependencias de planificación y de `apply` no son `blocked` por esa causa

#### Scenario: Sin contexto mantenido el almacén no bloquea el status ni el archivado, pero la validación lo reporta
- **DADO** un `axiom.yaml` sin contexto mantenido (o con solo `technical_context.reference`) y `artifact_store: engram`
- **CUANDO** se ejecutan `axiom workspace validate` y `axiom sdd status --json`
- **ENTONCES** `axiom workspace validate` reporta el error de REQ-2.6 de `workspace-topology` (sin nombrar `technical_context.maintained`) y finaliza con código de salida `1`
- **Y** `dependencies.archive` no se bloquea por esa causa en el status
