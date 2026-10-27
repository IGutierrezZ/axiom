# Verificación de supuestos de `inc-25` (decisión T6)

Este documento registra el resultado de los supuestos del diseño que solo se pueden comprobar al implementar. Cada supuesto tiene su sección. Si un supuesto no se puede comprobar, se anota «no verificado» y se detiene el trabajo para consultar al usuario; el resultado no se inventa.

| Supuesto | Rebanada | Sección | Estado |
|---|---|---|---|
| A-4 | S6-c2 | «A-4: sonda de linajes RDD (decisión T6)» | Verificado (2026-10-26) |
| A-2 | S11-c3 | «A-2: hook de Codex para `apply_patch`» | No verificado en runtime (Codex no instalado; forma tomada de la documentación oficial) |
| A-5 | S11-c3 | «A-5: matcher de Claude Code» | Verificado parcialmente (2026-10-27) |

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

## A-2: hook de Codex para `apply_patch`

**Supuesto (design A-2).** Codex ejecuta hooks `PreToolUse` y `PostToolUse` para `apply_patch`; el parche llega por la carga y la salida 2 bloquea (`PreToolUse`) o devuelve retroalimentación al agente (`PostToolUse`). La forma exacta se verifica al implementar S11.

**Comprobación.** Codex **no está instalado** en esta máquina (`codex --version` no existe), así que no se ha podido ejecutar. Por decisión del usuario, la entrada se implementa según la documentación oficial y A-2 queda registrado como «no verificado en runtime». Nada de lo siguiente está observado en una ejecución real de Codex.

**Fuentes.**

- Documentación oficial de hooks: <https://learn.chatgpt.com/docs/hooks>.
- Pull request que añadió `PreToolUse` y `PostToolUse` para `apply_patch`: <https://github.com/openai/codex/pull/18391>.
- Versión que activa los hooks por defecto: <https://github.com/openai/codex/releases/tag/rust-v0.124.0>.

**Forma documentada.**

| Aspecto | Valor documentado |
|---|---|
| Fichero | `~/.codex/hooks.json`: `{"hooks":{"<Evento>":[{"matcher":"<regex>","hooks":[{"type":"command","command":"…","timeout":N}]}]}}`. Es el mismo fichero que ya gestiona `ensureCodexSkillRegistryHook` |
| Activación | Hooks activos por defecto desde Codex 0.124.0 (estables); no hace falta tocar `config.toml` |
| Eventos | `PreToolUse` y `PostToolUse` disparan para `apply_patch` desde 0.123.0 |
| Matcher | `^apply_patch$`; `tool_name` es siempre `"apply_patch"` |
| Carga por `stdin` | `session_id`, `transcript_path`, `cwd`, `hook_event_name`, `model`, `permission_mode`, `tool_name`, `tool_use_id`, `tool_input` (y `tool_response` en `PostToolUse`) |
| Parche | Completo, en `tool_input.command` (cadena). La cabecera `*** Add File:` está confirmada por un ejemplo real de la documentación; `*** Update File:`, `*** Delete File:` y `*** Move to:` proceden de fuentes secundarias |
| `PreToolUse`, salida 2 | Bloquea la herramienta; el motivo se lee de `stderr` |
| `PostToolUse`, salida 2 | Con texto por `stderr`, Codex sustituye el resultado de la herramienta por ese texto. El parche ya está aplicado y no se deshace. Es el canal de retroalimentación que usa la guarda |
| Otro canal | Existe `additionalContext` en JSON; la guarda no lo usa |

**Decisión.**

- La entrada instalada por `ensureCodexBranchGuardHook` usa matcher `^apply_patch$`, la orden literal `axiom branch guard --agent codex` y `timeout` 30 (el mismo que el hook de skill-registry), en `PreToolUse` y en `PostToolUse`.
- La guarda ya lee el parche de `tool_input.command` y sus cabeceras (`codexPaths`, búsqueda en todos los textos de `tool_input`), e ignora el resto de la carga (`session_id`, `tool_response`, etc.). `internal/branch/engine/guard_hook_test.go` fija la carga documentada completa para los dos eventos.
- El canal de aviso de Codex (`PostToolUse` con salida 2) **existe en la documentación**, por lo que Codex recibe bloqueo y aviso. Si en una ejecución real el canal no devolviera el texto al agente, Codex quedaría solo con el bloqueo.

**Riesgos documentados.**

1. **`PostToolUse` puede no dispararse con `apply_patch`** en alguna versión: <https://github.com/openai/codex/issues/26729> (0.136.0 en Windows; cerrado sin causa identificada). En ese caso Codex pierde el aviso, pero el bloqueo de `PreToolUse` no depende de él.
2. **Hooks no gestionados:** Codex no ejecuta un hook que no sea gestionado hasta que la persona lo confía con `/hooks`. La confianza queda ligada al hash de la entrada: si `axiom sync` cambia la entrada (orden, matcher o `timeout`), hay que volver a confiar. Por eso la orden, el matcher y el `timeout` son literales y estables.
3. **Código de salida 2 con doble significado.** El código 2 es a la vez el error de uso de `axiom branch` (verbo o flag no reconocidos) y la señal de bloqueo en Claude Code y Codex. Un binario `axiom` sin el verbo `guard` (una versión anterior que aún esté en el `PATH`) bloquearía toda escritura del agente. No se cambia la orden literal: `axiom sync` instala la entrada con el mismo binario que conoce el verbo. Si `axiom` no está en el `PATH`, el shell devuelve 127 y el runtime no bloquea.

**Estado: A-2 no verificado en runtime.** Pendiente de comprobar con Codex instalado: que `PreToolUse` y `PostToolUse` disparen para `apply_patch` con la forma descrita, que `PostToolUse` con salida 2 devuelva el texto al agente y el comportamiento tras confiar el hook con `/hooks`.

## A-5: matcher de Claude Code

**Supuesto (design A-5).** `MultiEdit` sigue existiendo en el matcher `Write|Edit|MultiEdit|NotebookEdit` de Claude Code; si no existe, conservarlo es inocuo.

**Comprobación.** Versión instalada: `claude --version` devuelve `2.1.307 (Claude Code)`. En una sesión real de Claude Code 2.1.307 observada por el orquestador, las herramientas de edición expuestas son `Write`, `Edit` y `NotebookEdit`; `MultiEdit` no aparece.

**Resultado.** El matcher de un hook de Claude Code es una expresión regular, por lo que conservar `MultiEdit` en `Write|Edit|MultiEdit|NotebookEdit` no tiene efecto negativo y cubre las versiones que sí la expongan. La entrada instalada por `ensureClaudeBranchGuardHook` usa ese matcher y la orden literal `axiom branch guard --agent claude-code`, en `PreToolUse` y en `PostToolUse`, con `timeout` 30.

**Canales.** Los dos canales (`PreToolUse` con salida 2 para bloquear y `PostToolUse` con salida 2 para devolver el aviso al agente) están verificados en este repositorio por `scripts/claude-hooks/guard-primary-checkout.ps1` (design D-18).

**Estado: A-5 verificado parcialmente.** La versión y la ausencia de `MultiEdit` en las herramientas expuestas están observadas; no se ha ejecutado un hook de la guarda en una sesión real, y se mantiene el riesgo del código 2 descrito en A-2 (aplica también a Claude Code).
