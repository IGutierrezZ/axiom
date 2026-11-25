# Notas de migración: retirada de `axiom review` (RDD) y de `bench/`

Esta versión retira el desarrollo dirigido por recibos (RDD), el verbo `axiom review*`, el interruptor `review mode`, el módulo `bench/` y el verbo `axiom bench-model-picker`. Judgment Day, la pantalla «Review and Confirm» del instalador, `axiom sdd-attempt grant`, el paquete `consentenvelope` y `kickoff.CommitReview*` se conservan.

> **Esta versión no limpia lo que instalaron versiones anteriores.** Haz la limpieza de los puntos 1 a 4 **antes** de instalarla. Si instalas primero y limpias después, cada fin de turno de Claude Code mostrará un error («unknown command») hasta que lo hagas.

## Qué limpiar a mano

1. **Hooks de Claude Code (hazlo antes de instalar).** En `~/.claude/settings.json` y, si existen, en `.claude/settings.json` y `.claude/settings.local.json` de cada proyecto, abre `hooks.Stop` y `hooks.SessionStart` y borra solo los hooks cuyo `command` sea exactamente `axiom review stop-hook --agent claude-code` o `gentle-ai review stop-hook --agent claude-code`, y la entrada que quede vacía. Repite el patrón `<binario> review stop-hook --agent <id>` en cualquier otro agente. Si no lo haces, cada fin de turno mostrará un error no bloqueante («unknown command»), porque el verbo ya no existe.
2. **Agentes de revisión.** Borra `review-readability`, `review-refuter`, `review-reliability`, `review-resilience` y `review-risk` (`.md`, y `.yaml` en Kimi) de los directorios `agents` de Claude (`~/.claude/agents`), Cursor, Kiro y Kimi. En OpenCode, quita de `opencode.json` (o del overlay instalado) `review-risk`, `review-readability`, `review-reliability`, `review-resilience`, `review-refuter` y `review-validator`, con sus permisos de `task`. `axiom uninstall` ya no los borra.
3. **Plugins de OpenCode.** Borra `opencode-review-transport.ts` y el heredado `review-result-artifacts.ts` del directorio de plugins (`~/.config/opencode/plugins` o el del proyecto). `axiom sync` ya no los comprueba ni los migra.
4. **Skill y ficheros compartidos.** Borra el directorio `rdd-defect-workflow` y los ficheros `_shared/review-ledger-contract.md` y `_shared/review-ledger-contract-pi.md` de los directorios de skills de cada agente. Una selección guardada que nombre la skill ya no hace fallar `sync`: se ignora.
5. **Prosa de orquestador (después del primer `axiom sync`).** Comprueba que los ficheros de instrucciones de tus agentes (por ejemplo `~/.claude/CLAUDE.md` o `AGENTS.md`) no conservan la sección `#### Review Execution Contract` ni otra prosa de `axiom review`. `sync` reescribe la parte que gestiona; si la copiaste fuera de ella o la editaste a mano, bórrala tú.
6. **Opcional e inofensivo.** `rdd_mode` y `rdd_mode_recorded_at` del estado global y `rdd_mode` del hub se ignoran y desaparecen en la siguiente escritura. El almacén `<git common dir>/gentle-ai/review-transactions` y las asignaciones de modelo de los roles `rdd-*` de Codex pueden borrarse a mano; `axiom branch done` ignora sin aviso un almacén residual.

## Capacidades publicadas que se retiran

- El asset `gentle-ai-review-provider-contract-<semver>.tar.gz` de las releases.
- Los esquemas `contracts/review-integration/{v1,v2}`.
- El campo `provider_contract_semver` y la entrada `provider-contract` de `artifacts` en `gentle-ai.release-provenance/v1` (el esquema y el asset de procedencia siguen en `v1`).
- Los verbos `axiom review*` (incluidos `review mode`, `review store-reset` y `review stop-hook`) y `axiom bench-model-picker`.
- La entrada «Reiniciar almacén de revisiones» del menú de bienvenida, las pantallas del interruptor RDD del instalador y la ruta `/api/project/review-mode` del dashboard (devuelve 404; un cliente que envíe `review_mode` no recibe error).
- El job `claude-network-none` de la CI y su Dockerfile.

## Acciones de quien publica

- Cortar una candidata de release nueva: una RC anterior a este cambio lleva el asset del contrato y no pasaría la comparación del nuevo `promote-stable-rc.yml`.
- Pegar este texto en las notas de la primera versión que incluya el cambio.
