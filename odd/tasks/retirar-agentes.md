# ODD: Retirar los agentes que no se usan (retirar-agentes)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/retirar-agentes.md`. Espejo Engram: `odd/retirar-agentes/tasks`.
> Rama: `inc/retirar-agentes` · worktree `C:\repos\axiom-wt\retirar-agentes\axiom`. Cambio C-08-d del plan maestro (`docs/plan/backlog-2026-11/08-adelgazamiento.md`), ola A3; decisiones P17 y P20 (apartado 6.2 del README y coordinación).

## Objetivo

Retirar seis agentes: `kiro-ide`, `windsurf`, `kimi`, `hermes`, `openclaw` y `kilocode`. Se quedan once: `claude-code`, `vscode-copilot`, `opencode`, `github-copilot-cli`, `gemini-cli`, `codex`, `cursor`, `antigravity`, `pi`, `trae-ide` y `qwen-code` (este sigue oculto al instalar).

## Problema y porqué

Los seis agentes no se usan y cuestan unas 12.300 líneas (adaptadores, assets, goldens, ramas por agente en la inyección, la desinstalación y la TUI). Decisión del usuario (P17, 2026-11-24): de los agentes heredados se van todos menos Gemini CLI, Codex, Cursor, Antigravity, Pi, Copilot CLI y los que apenas ahorran (Trae y Qwen).

## Decisiones

- **P20 (usuario: «Sí, borrarlas ya»).** Sin lápidas: `uninstall` deja de conocer las rutas de los agentes retirados y no hay comandos tumba. Lo que Axiom escribió en equipos con instalaciones antiguas lo limpia la persona a mano (lista al cerrar, en `## Siguiente paso`).
- **Lector tolerante.** Un estado, copia o selección guardados con un `AgentID` retirado siguen cargando: el agente se ignora con un aviso y se sincronizan los demás.
- **Código compartido.** Se queda con el agente que lo conserva; solo se borra la rama del retirado. Kilo Code comparte con OpenCode `agents/opencode/adapter.go`, `agents/skills_dir.go` y la familia OpenCode de `components/sdd/inject.go` y `uninstall`: lo generado para OpenCode no cambia (goldens de OpenCode sin diferencias).
- **Specs vivas.** Solo se retiran menciones a estos agentes (requisitos y escenarios que solo hablan de ellos, y listas que los nombran). Si hiciera falta reescribir requisitos, el cambio pasaría a SDD.
- **Históricos.** No se tocan `docs/archive`, `openspec/changes/archive`, `docs/audits`, `docs/plan`, `docs/releases` ni los `odd/tasks/` cerrados, salvo marcar como superados `odd/tasks/kiro-workspace-parity.md` y la parte de Windsurf de `odd/tasks/codex-sync-sin-temas-y-rollback.md`.

## Alcance autorizado

Lo que lista C-08-d en la sección 08: registro (`model`, `catalog`, `agents/factory.go`, `capabilitymanifest`, `discovery`, paquetes de adaptador), instalación y CLI (`installcmd`, `cli/{validate,run,sync,workspace_*}`), TUI (selector de modelos de Kiro y referencias), componentes (`sdd`, `persona`, `permissions`, `engram`, `mcp`, `uninstall`), assets, goldens, docs, specs vivas y e2e. Más las rutas de configuración de esos agentes en `system/config_scan.go`, `skillregistry`, `semantic/detector.go` y `dashboard`.

Fuera de alcance: los once agentes que se quedan, persona/presets/temas/agent builder (C-08-f), GGA (C-08-e).

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y no hay elección del usuario. Comprobaciones con `go vet` y `go test` por paquetes, sin binarios sueltos de `axiom` (el EDR los borra). El lector tolerante (T7) se prueba con un test nuevo.

## Entrega

`single-pr`: una PR con un commit por tarea (un commit por agente). Etiqueta `type:refactor`. Previsión: unas 12.300 líneas, casi todo borrado.

## Tareas

- [x] T1 · Retirar Kilo Code (adaptador, registro, ramas de la familia OpenCode, instalación, desinstalación, tests). Goldens de OpenCode sin diferencias.
- [ ] T2 · Retirar OpenClaw.
- [ ] T3 · Retirar Hermes (incluida la skill `hermes-ephemeral-delegation`).
- [ ] T4 · Retirar Kimi (assets y agentes `review-*` que queden).
- [ ] T5 · Retirar Windsurf (assets y goldens).
- [ ] T6 · Retirar Kiro IDE (modelos de Kiro, selector de la TUI, assets y goldens).
- [ ] T7 · Lector tolerante: un estado con agentes retirados carga y sincroniza el resto con un aviso.
- [ ] T8 · Docs, e2e, documentos ODD superados, `## Spec viva` integrada con `axiom archive integrate --odd retirar-agentes` y criterio `rg`.

Ruta de todas las tareas: inline (borrado guiado por el compilador; sin unidades independientes con superficies disjuntas que justifiquen escritores en paralelo).

## Criterio de terminado

- `internal/model` y el catálogo solo conocen los once agentes conservados.
- La instalación (TUI y CLI) ofrece los diez seleccionables; Qwen Code sigue oculto.
- Un estado con agentes retirados carga y sincroniza con un aviso.
- `rg -il 'kiro|windsurf|kimi|hermes|openclaw|kilocode|kilo code'` no encuentra nada fuera de los históricos y del lector tolerante.
- Goldens regenerados y revisados por diff; los de OpenCode, sin cambios. `go vet ./...` y `go test` de los paquetes tocados en verde.

## Progreso

### T1 · Kilo Code

- Borrado `internal/agents/kilocode` y sus ramas: registro, `capabilitymanifest`, `installcmd`, `validate`, familia OpenCode en `components/sdd` (overlay, perfiles, plugins), `persona`, `permissions`, `mcp`, `engram`, `agentguidance`, `uninstall` (restos globales de Kilo), `workspace_instructions`, `skillregistry` y `config_scan`. Funciones muertas retiradas: `stripOpenCodeNativeFallbackAgents`, `cleanupKilocodeProfileJDPermissions`, `restoreKilocodeManagedAgentTools*`, `usesManagedMCPShape`, `kiloPluginNames`, `kiloConfigHasAxiomEntries`.
- Pruebas: borradas las propias de Kilo; las mixtas se quedan con OpenCode; recuentos de agentes 17 → 16.
- Evidencia: `go vet ./...` limpio; `go test` ok en `agents/...`, `catalog`, `model`, `installcmd`, `components/...` (goldens de OpenCode sin diferencias), `skillregistry`, `system`, `opencode`, `assets`, `tui`, `cli`, `app`, `dashboard`, `update/...`, `state` y `workspace`.

## Siguiente paso

T2.
