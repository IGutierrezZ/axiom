# ODD: Retirada de la telemetría y de los restos del upstream Gentle AI

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/retirada-telemetria-y-restos-upstream.md`.
> Espejo de recuperación en Engram: topic `odd/retirada-telemetria-y-restos-upstream/tasks`, proyecto `axiom`.
> Abierto el 2026-10-04 desde `origin/main` (`51740868`). Este documento viaja en el PR de cierre (`docs/odd-retirada-telemetria-y-restos-upstream`).

## Objetivo

Que Axiom deje de enviar datos y de dirigir acciones al proyecto upstream Gentle AI, y que quede limpio de sus restos funcionales y de marca. En concreto:

- retirar la telemetría por completo;
- corregir la skill que abre PRs en el repositorio del upstream;
- cerrar los problemas de higiene de tests que contaminan la máquina del usuario;
- corregir `RefreshSkip`;
- retirar las skills ya descatalogadas de las instalaciones existentes;
- sustituir la marca Gentle AI donde no sea un contrato.

## Problema

Mapeo sobre `origin/main` `c504f2e6`, 2026-10-04:

- **Telemetría hacia el upstream.**
  - **Destino:** `internal/telemetry/telemetry.go:36` define `DefaultEndpoint = https://telemetry.gentlemanprogramming.com/v1/events`, el servidor del upstream.
  - **Consentimiento:** activada por defecto, con un aviso único en inglés por stderr, sin pedir permiso.
  - **Eventos:** `install` y `heartbeat` cada 24 h, con un `install_id` guardado en `~/.gentle-ai/telemetry.json`, fichero compartido con el upstream.
  - **Métricas de ejecución:** se recogen mediante hooks Stop y SubagentStop de Claude y Codex (`axiom telemetry runtime ...`) y mediante un plugin de OpenCode que ejecuta `gentle-ai`.
  - **Instalaciones existentes:** `sync` no retira esos hooks.
  - **Colector y despliegue:** hay un colector (`cmd/gentle-telemetry`, `internal/telemetrycollector`) y su despliegue (`deploy/telemetry`).
  - **Decisión del usuario: quitarla del todo.**
- 🔴 **La skill propia `skills/branch-pr/SKILL.md`** (líneas 16 y 249) ejecuta `gh pr create --repo Gentleman-Programming/gentle-ai`, así que un agente abriría PRs en el upstream.
- **Hub contaminado.** `cmd/axiom/knowledge_cli_test.go:11,48,119` llama a `runInit` dentro del proceso, y `cmd/axiom` no tiene `TestMain` con HOME aislado. Como en Windows `hub.NewManager` lee `USERPROFILE`, cada ejecución registra directorios temporales en el hub real `~/.axiom/workspaces.json`. Ahora mismo tiene 37 de 44 entradas muertas, y el workspace activo es un huérfano.
- **PATH del usuario.** Hay dos entradas muertas, `AppData\Local\Temp\codegraph-install-test-kvp10` y `kvp25` (`\current\bin`), seguramente del instalador externo de CodeGraph. El escritor de PATH de Axiom está protegido en `go test`, pero un futuro test que lance el binario real con `install` o `sync` lo esquivaría.
- **`RefreshSkip`** (`internal/skillregistry/guard.go:36-53`) compara `cwd == home` y la raíz del sistema de ficheros de forma léxica. En Windows, una diferencia de mayúsculas o un nombre 8.3 hace que el índice se regenere dentro del HOME (`~/.atl`, Engram, `~/AGENTS.md`).
- **Skills retiradas** (`branch-pr` y `gentle-ai-bench`, #70). Las instalaciones existentes conservan las copias, que compiten con `axiom-branch-pr`.
  - `uninstall` ya no las limpia.
  - Si `state.json` las lista y falta la copia, la verificación de `sync` falla.
- **Marca Gentle AI.**
  - La cabecera de `AGENTS.md` («Gentle AI™ — Agent Skills Index») y su línea sobre la convención `gentle-ai-*`.
  - 164 mensajes visibles para el usuario que dicen `gentle-ai <verbo>` en 58 ficheros. El PATH del usuario aún tiene `AppData\Local\gentle-ai\bin`, así que ese consejo puede lanzar el binario del upstream.
  - La skill embebida `chained-pr` duplica a `axiom-chained-pr`.
  - `skills/axiom-collab-perfect` es una skill para contribuir al upstream.
  - Hay contratos con instalaciones existentes: los IDs de protocolo `gentle-ai.*/vN` y la marca `gentle-ai:managed-opencode-launcher/v1`.

## Alcance y tareas

- [ ] **T1 · 🔴 La skill `branch-pr` no debe apuntar al upstream.** Corregir `skills/branch-pr/SKILL.md`, y cualquier otra skill propia, para que use el repositorio de Axiom o el actual. Tamaño S.
- [ ] **T2 · Aislar los tests de `cmd/axiom` del hub real.** Añadir un `TestMain` con `HOME` y `USERPROFILE` temporales (el patrón de `internal/dashboard/main_test.go`) y revisar otros paquetes sin aislamiento que toquen el hub. Después, **limpieza local del hub del usuario**: `project prune`, quitar las entradas `axiom-test-*` que aún existen y volver a activar el workspace `axiom`. Tamaño S.
- [ ] **T3 · PATH.** Quitar del PATH de usuario las dos entradas muertas `codegraph-install-test-*` (acción local). Añadir una salvaguarda preventiva: que una variable de entorno, por ejemplo `AXIOM_NO_PERSISTENT_PATH`, impida escribir el PATH persistente, y que los `TestMain` la fijen. Tamaño S.
- [ ] **T4 · `RefreshSkip` por identidad.** Comparar `home` y la raíz del sistema de ficheros por identidad (`pathidentity.SameDirectory` o equivalente), después del atajo léxico, y añadir una guarda en `Regenerate`. Tamaño S.
- [ ] **T5 · Retirar skills descatalogadas en instalaciones existentes.**
  - Registro de retiradas `{ID, sha256 conocidos}`.
  - `sync` borra solo las copias sin modificar y avisa si alguna está editada.
  - `uninstall` incluye esos IDs.
  - Se podan de `state.json`.
  - La verificación de `sync` deja de exigirlas.

  Tamaño M.
- [ ] **T6 · Retirar la telemetría por completo.** Tamaño L, en varios PRs ordenados para respetar `deadcode-ratchet.sh`:
  - dejar de enviar y de instalar hooks y plugins;
  - migración en `sync` que retire los hooks y el plugin ya instalados;
  - eliminar el cliente, el CLI `axiom telemetry`, los contratos y la documentación;
  - eliminar el colector y `deploy/telemetry`;
  - registro de absorción (pasar las filas de telemetría a `revertido`, regla 7);
  - journey de bench `j4395`.
- [ ] **T7 · Marca Gentle AI.** Tamaño M-L, en varios PRs:
  - cabecera y convención de `AGENTS.md`;
  - mensajes `gentle-ai <verbo>` que pasan a `axiom <verbo>`, con sus goldens;
  - duplicado de `chained-pr`;
  - `axiom-collab-perfect`, que queda pendiente de decidir si se conserva o se retira;
  - contratos (IDs `gentle-ai.*/vN` y marca del lanzador): lectura dual de ambos nombres sin romper instalaciones. Se diseña antes de implementarlo.

## Restricciones

- Los contratos con instalaciones existentes solo cambian con lectura dual o migración, nunca a secas.
- `metadata.author` de las skills es atribución de licencia y se conserva.
- El checkout principal `C:\repos\axiom` pertenece a otra sesión del usuario (inc-24): solo se trabaja en worktrees.
- **Seguridad de la ejecución:**
  - nunca matar procesos por nombre de imagen;
  - nunca ejecutar `axiom install`, `setup`, `sync`, `upgrade` ni `doctor` contra el HOME o el PATH reales, salvo en las acciones locales de T2 y T3 que el usuario ha autorizado;
  - no ejecutar `./internal/cli` completo en local.
- `scripts/deadcode-ratchet.sh`: ningún PR puede dejar funciones sin llamadores.
- Unas 400 líneas por tarea es una heurística orientativa, no un límite.

## Configuración de ejecución

- **TDD:** desactivado (`openspec/config.yaml:16`). Runner: `go test`.
- **Entrega:** `ask-on-risk`, con una previsión muy superior a 400 líneas. La estrategia de encadenado **está pendiente de decisión del usuario**.
- **RDD:** desactivado. Se evalúa el riesgo por commit con `axiom review assess`; si sale `high`, hace falta un verificador independiente.

## Criterios de aceptación

- Ningún binario ni hook de Axiom envía datos a dominios del upstream. Las instalaciones existentes quedan sin hooks ni plugins de telemetría tras `sync`.
- Ninguna skill propia dirige acciones (PRs, issues) al repositorio del upstream.
- `go test ./...` no modifica el hub ni el PATH del usuario.
- `RefreshSkip` no regenera el índice dentro del HOME por diferencias de forma en la ruta.
- `sync` retira `branch-pr` y `gentle-ai-bench` sin modificar de las instalaciones existentes.
- No quedan mensajes visibles que recomienden `gentle-ai <verbo>`. Los contratos se leen con los dos nombres.
- El CI está en verde en todos los PRs.

## Progreso

- 2026-10-04: ODD abierto. Mapeo previo hecho por dos agentes de exploración. Decisiones del usuario: carril ODD, quitar la telemetría del todo e incluir todos los seguimientos.

## Siguiente paso

Decidir la estrategia de entrega y después T1, que es urgente.
