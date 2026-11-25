# ODD: herramientas de Engram en los agentes de judgment-day

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/jd-engram.md`, en la rama `fix/jd-engram` (worktree `C:\repos\axiom-wt\jd-engram\axiom`, creado desde `origin/main` `13d2832f` el 2026-11-25).
> Espejo de recuperación en Engram: topic `odd/jd-engram/tasks`, proyecto `axiom`.
> **Estado: IMPLEMENTADO el 2026-11-25**, pendiente de PR.

Tipo: bug

## Objetivo

Que los agentes de judgment-day de Claude Code (`jd-judge-a`, `jd-judge-b` y `jd-fix-agent`) reciban las herramientas de Engram con las dos formas de registro posibles, igual que los agentes `sdd-*`, y que una prueba de assets impida que vuelva a aparecer un agente con un solo prefijo.

## Problema y porqué

- Engram se registra en Claude Code de dos formas válidas: `claude mcp add engram` expone `mcp__engram__*`, y la ruta de plugin expone `mcp__plugin_engram_engram__*`.
- Los `sdd-*` declaran sus herramientas con el marcador `{{ENGRAM_TOOL_PREFIX}}<herramienta>`, y `expandEngramToolNames` (`internal/components/sdd/inject.go`) lo expande a los dos prefijos al renderizar (#2698/#3778).
- Los tres `jd-*` de `internal/assets/claude/agents/` escribían a mano solo `mcp__plugin_engram_engram__*`. En una instalación con el MCP `engram` (la de Axiom: el plugin `engram@engram` solo aporta hooks), esas herramientas no existen y los jueces y el agente de correcciones trabajan sin memoria.

## Exploración

- Única ruta de renderizado de los agentes embebidos: el bucle de `Inject` en `internal/components/sdd/inject.go`, que aplica `expandEngramToolNames` a todos los ficheros de `claude/agents/`, `jd-*` incluidos. `internal/cli/run.go` (`sddSubAgentPaths`) solo lista rutas.
- Otros runtimes, sin el defecto:
  - Kiro concede `@engram` a nivel de servidor (`tools: ["@builtin", "@engram"]`), sin prefijos.
  - Copilot, Cursor, Kimi, OpenCode y el resto no declaran herramientas de Engram por nombre en sus agentes.
- Pruebas existentes:
  - `internal/components/sdd/engram_tool_prefix_test.go` comprueba los dos prefijos solo en el `sdd-apply` renderizado.
  - Ninguna prueba del paquete `internal/assets` vigilaba las declaraciones de los assets.

## Alcance autorizado

- Frontmatter `tools:` de `internal/assets/claude/agents/jd-judge-a.md`, `jd-judge-b.md` y `jd-fix-agent.md`: usar `{{ENGRAM_TOOL_PREFIX}}`, como los `sdd-*`.
- Prueba nueva en `internal/assets`: todo agente embebido que declara herramientas de Engram las declara con los dos prefijos (el marcador cuenta como ambos).
- Ampliar `TestInjectGrantsBothEngramToolShapes` para que cubra también los `jd-*` renderizados.
- **Fuera de alcance:** el hook del plugin de Engram que pide cargar las herramientas `mcp__plugin_engram_engram__*` con `ToolSearch` en cada prompt. Pertenece al repositorio de Engram, no a Axiom.

## TDD

Modo: desactivado. Fuente: sin configuración de TDD en `axiom.yaml` ni en la sesión. Ejecutor: `go test`. Aun así, la prueba de assets se escribe primero para ver el rojo con los `jd-*` actuales.

## Tareas

- [x] T1 — Prueba de assets de los dos prefijos, corrección de los tres `jd-*` y ampliación de la prueba de renderizado. Ruta: inline (unidad de menos de 100 líneas).

## Criterios de aceptación

- Ningún agente embebido declara una herramienta de Engram con un solo prefijo.
- `go test ./internal/assets -count=1` en verde.
- `go test ./internal/components/sdd -run 'TestInject(GrantsBothEngramToolShapes|LeavesNoRawEngramToolPlaceholder|EngramToolExpansionIsAmbientIndependent)' -count=1` en verde.

## Comprobaciones aplicables

- `go test ./internal/assets -count=1`
- `go test ./internal/components/sdd -run 'TestInject(GrantsBothEngramToolShapes|LeavesNoRawEngramToolPlaceholder|EngramToolExpansionIsAmbientIndependent)' -count=1`
- No se ejecutan `go test ./...` ni las pruebas de `internal/app`, `internal/dashboard`, `internal/update`, `internal/opencode`, `internal/components/gga`, `bench` y `e2e` (Cortex XDR).

## Progreso y evidencia

- **T1** (2026-11-25):
  - Prueba nueva `TestEmbeddedAssetsGrantEngramToolsUnderBothPrefixes` (`internal/assets/engram_tool_grants_test.go`). Recorre todos los assets embebidos y exige que cada herramienta de Engram aparezca con los dos prefijos. El marcador `{{ENGRAM_TOOL_PREFIX}}` cuenta como ambos, pero solo dentro de un directorio `agents`, que es donde lo expande `Inject`. Además fija que los tres `jd-*` siguen declarando herramientas de Engram, para que la guarda no pierda su objeto.
  - Rojo observado antes de corregir: 8 errores, uno por herramienta de `jd-judge-a` (2), `jd-judge-b` (2) y `jd-fix-agent` (4), todos «grants `mcp__plugin_engram_engram__X` without `mcp__engram__X`».
  - Corrección: la línea `tools:` de los tres `jd-*` usa `{{ENGRAM_TOOL_PREFIX}}` en lugar del prefijo de plugin escrito a mano.
  - `TestInjectGrantsBothEngramToolShapes` comprueba ahora también `jd-judge-a`, `jd-judge-b` y `jd-fix-agent` renderizados, no solo `sdd-apply`.
  - Verde: `go test ./internal/assets -count=1` → `ok` (16,3 s).
  - Verde: `go test ./internal/components/sdd -run 'TestInject(GrantsBothEngramToolShapes|LeavesNoRawEngramToolPlaceholder|EngramToolExpansionIsAmbientIndependent)|TestDedicatedReviewAndJudgmentAssetsRenderRoleContracts' -count=1 -v` → las cuatro en `PASS`.
  - Verde: `go test ./internal/components/sdd -count=1 -timeout 20m` → `ok` (77,2 s), porque ese paquete renderiza los agentes tocados.
  - `gofmt -l` sin salida y `go vet ./internal/assets` limpio.
  - `axiom archive integrate --odd jd-engram --json` → `kind: bug`, `impact: no_impact`, `nothingToIntegrate: true`, sin escrituras.
  - Commit: el de unidad de trabajo `fix(assets): conceder a los agentes de judgment-day las herramientas de Engram con los dos prefijos`, que incluye este documento (`git log -1 -- odd/tasks/jd-engram.md`).

## Impacto en la spec

Sin impacto en la spec
Motivo: corrige la declaración de herramientas de tres agentes embebidos para alinearla con la convención ya vigente de los `sdd-*`; no cambia ningún requisito de `openspec/specs/`.

## Siguiente paso

Push, PR (`scripts/axiom-worktree.ps1 pr jd-engram -Lane odd`) y merge, a decisión de la persona. Tras el merge, `scripts/axiom-worktree.ps1 done jd-engram -Lane odd`.
