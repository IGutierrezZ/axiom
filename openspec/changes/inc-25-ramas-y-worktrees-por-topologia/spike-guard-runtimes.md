# Verificación de supuestos de `inc-25` (decisión T6)

Este documento registra el resultado de los supuestos del diseño que solo se pueden comprobar al implementar. Cada supuesto tiene su sección. Si un supuesto no se puede comprobar, se anota «no verificado» y se detiene el trabajo para consultar al usuario; el resultado no se inventa.

| Supuesto | Rebanada | Sección | Estado |
|---|---|---|---|
| A-4 | S6-c2 | «A-4: sonda de linajes RDD (decisión T6)» | Verificado (2026-10-26) |
| A-2 y A-5 | S11-c3 | Se añaden en S11.11 | Pendiente |

## A-4: sonda de linajes RDD (decisión T6)

**Supuesto (design A-4).** La sonda de linajes RDD abiertos que consulta `axiom branch done` debe leer el almacén de `internal/reviewtransaction` sin lanzar git, para no escapar al observador del motor. Si la API de `reviewtransaction` exige git, la sonda queda fuera del motor, solo local, y su exención se documenta.

**Comprobación.** Se ha leído la API de `internal/reviewtransaction` que permite listar los linajes de un repositorio (`InventoryAuthority` y `ExactReviewLineageForeignWorktree`) y el camino que recorre para resolver la autoridad de revisión.

**Resultado: el supuesto no se cumple; se aplica la salida prevista por el propio diseño.**

1. `reviewtransaction` exige git. La resolución de la raíz de autoridad (`reviewAuthorityRoot`) pasa por `SnapshotBuilder.repositoryRoot` y por `OpenRepositoryIdentityLease`, que lanzan git con su propia ejecución de procesos (no usan `gitcommon.Run`). No existe un camino de solo lectura del almacén que prescinda de git.
2. El inventario de linajes no expone el worktree de cada linaje. Solo con el inventario no se puede decidir qué linajes pertenecen al worktree que `done` va a eliminar; hace falta el filtro por worktree que ofrece `ExactReviewLineageForeignWorktree`.

**Decisión.**

- La sonda vive en `internal/cli/branch_lineage.go` (adaptador `reviewLineageProbe`, que implementa el puerto `branch.LineageProbe` con `OpenLineages(ctx, worktreeRoot)`), **fuera del motor**. El motor (`internal/branch/engine`) solo conoce el puerto y no importa `reviewtransaction`.
- La sonda solo se ejecuta en local y es de **solo lectura**: no crea, modifica ni abandona ningún linaje. El aviso nombra el linaje y propone `axiom review abandon`; abandonarlo es decisión de quien ejecuta `done`.
- La sonda queda **exenta del observador** del motor: sus invocaciones de git no pasan por `gitcli` y no se registran. Es una exención deliberada y acotada a esta sonda; el resto de las operaciones de `done` siguen pasando por el observador.
- Un linaje abierto solo avisa: no impide la eliminación (REQ-25.17, escenario «Linaje RDD abierto genera un aviso»). El aviso se emite después de confirmar la fusión y antes de borrar, y solo cuando hay un worktree vinculado o el modo es `in_place`.
- Un fallo de la sonda no bloquea `done`; se informa como aviso.
- `engine.DefaultWithLineages(probe)` monta el motor con la sonda; `engine.Default()` queda como antes (sin sonda). `RunBranch` usa `DefaultWithLineages(newBranchLineageProbe())`, donde `newBranchLineageProbe` es una variable de prueba.

**Cobertura.** `TestDoneWarnsOfAnOpenRDDLineageWithoutPreventingTheRemoval` en `internal/branch/engine/done_layers_test.go` (sonda doble: un linaje abierto añade el aviso con `axiom review abandon` y la eliminación se realiza); las pruebas `TestBranchLineageProbe*` de `internal/cli/branch_lineage_test.go` (el adaptador sobre el inventario de `reviewtransaction`, incluido un repositorio real sin revisiones); y `TestBranchDoneMountsTheLineageProbeAndWarnsOfAnOpenLineage` en `internal/cli/branch_done_layers_test.go` (el verbo monta la sonda y sale con 0).

**Alcance.** Cubierto por la excepción que ya prevé design A-4; no cambia ningún contrato ni requisito.
