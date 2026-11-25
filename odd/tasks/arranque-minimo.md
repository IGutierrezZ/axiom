# ODD: Arranque mínimo, sin persona, presets, temas, agent builder ni logo heredados (arranque-minimo)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/arranque-minimo.md`. Espejo Engram: `odd/arranque-minimo/tasks`.
> Rama: `inc/arranque-minimo` · worktree `C:\repos\axiom-wt\arranque-minimo\axiom`. Cambio C-08-f del plan maestro (`docs/plan/backlog-2026-11/08-adelgazamiento.md`), ola A5; decisiones P18, P19 y P20.

## Objetivo

Que la instalación sea detectar agentes, elegir componentes e instalar, sin pasos heredados de gentle-ai.

## Problema y porqué

Decisión del usuario: «Me sobran muchos de inicio». Persona `gentleman` y `neutral`, los cuatro presets, la migración de temas, el logo de OpenCode y el agent builder son herencia de gentle-ai: entre 13.500 y 14.500 líneas que cuestan en cada cambio y en cada ejecución de pruebas.

## Decisiones

- **P18 (usuario: «Sí, retirarlo»).** Se retira el agent builder: `internal/agentbuilder`, sus pantallas, la opción «Crear agente personalizado» del menú y `PRD-AGENT-BUILDER.md`.
- **P19 (usuario: «Sí, solo Axiom»).** Solo queda la persona de Axiom (`persona-axiom.md` y `output-style-axiom.md`). Se retiran `gentleman`, `neutral`, el alias `gentleman-neutral-artifacts`, la opción «sin persona gestionada» (`custom`) y la pantalla de elección. Quien no quiera persona desmarca el componente `persona` en el árbol de componentes.
- **P20 (usuario: «Sí, borrarlas ya»).** Sin lápidas: ni limpieza en `uninstall`, `sync` o `upgrade` de lo retirado (output styles y personas de `gentleman`/`neutral`, temas, `gentle-logo.tsx`). La persona usuaria limpia a mano: `docs/releases/notas-retirada-arranque.md`.
- **Presets.** Los cuatro (`full-gentleman`, `ecosystem-only`, `minimal`, `custom`) se sustituyen por un conjunto por defecto: el núcleo que instalaba `full-gentleman` (Engram, SDD, skills, Context7, permisos y persona), editable siempre en el árbol de componentes. La TUI va de agentes al árbol de componentes. Desaparecen las opciones `--persona` y `--preset` de `axiom install`.
- **Lector tolerante (plan, C-08-f).** Un estado con `persona` o `preset` retirados carga con un aviso en `sync` y la siguiente escritura ya no los guarda; los componentes `theme`, `claude-theme` y `opencode-gentle-logo` se ignoran con el mecanismo de C-08-e (`model.IsKnownComponentID`).
- **Coordinación (2026-11-26).** Sigue en ODD con `## Spec viva`: los MODIFIED son consecuencias mecánicas de P18 y P19 y el único cambio de semántica (fallback a Axiom con aviso) lo fija el plan. Si aparece un requisito con conducta nueva no fijada por P18, P19 o la lectura tolerante, se para y se avisa (sería SDD).
- **Criterio `rg -il gentleman`.** No es alcanzable literalmente: Engram y su tap de Homebrew viven en la organización `Gentleman-Programming`, y hay atribución (`LICENSE`, `TRADEMARKS.md`, `CONTRIBUTORS.md`, `author:` de las skills). Se aplica a la persona, presets y temas `gentleman`; quedan esas referencias, la migración del agente SDD `gentleman` de OpenCode (`components/sdd`, núcleo fuera de alcance) y los históricos.
- **Históricos.** No se tocan `docs/archive`, `openspec/changes/archive`, `docs/audits`, `docs/plan`, `docs/releases` existentes, `docs/upstream-absorption-ledger.md`, `CONTRIBUTORS.md` ni los `odd/tasks/` cerrados.

## Alcance autorizado

Lo que lista C-08-f en la sección 08: `internal/components/{persona,theme,opencodedefault}`, `internal/agentbuilder` y `tui/screens/agent_builder_*`, `model` (`PersonaID`, `PresetID`, `presets.go`), `catalog/components.go`, TUI (`router.go`, `persona.go`, `preset.go`, `welcome.go`, `model.go`), assets de persona, goldens `persona-*`, `tui/preset_flow_test.go`; y lo que los consume: `cli` (`install`, `validate`, `run`, `sync`, `dryrun`), `state`, `planner`, `app`, `components/skills/presets.go`, `components/uninstall`, `update/upgrade`, docs y e2e.

Fuera de alcance: el núcleo (SDD/ODD, skills, Engram, guías, MCP, permisos, CodeGraph).

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y no hay elección del usuario. Comprobaciones con `go vet` y `go test` de los paquetes tocados (`-run` en `cli`), sin la batería completa ni binarios sueltos de `axiom`. El lector tolerante se prueba con un test nuevo.

## Entrega

`single-pr`: una PR con un commit por tarea. Etiqueta `type:refactor`. Previsión: unas 14.000 líneas, casi todo borrado.

## Tareas

- [x] T1 · Retirar el agent builder: paquete, pantallas, rutas, opción del menú, `PRD-AGENT-BUILDER.md` y sus tests.
- [ ] T2 · Retirar temas y logo de OpenCode: `components/theme`, `components/opencodedefault`, componentes `theme`/`claude-theme`/`opencode-gentle-logo` y plugin `gentle-logo`, su limpieza en `uninstall`/`upgrade`/`sync`, y tests.
- [ ] T3 · Persona solo Axiom: `PersonaID`, componente `persona`, assets y goldens de `gentleman`/`neutral`, pantalla de persona, `--persona`, fallback de `sync` y lector tolerante con aviso.
- [ ] T4 · Presets fuera: conjunto por defecto, pantalla de presets, `--preset`, `SkillsForPreset`, flujo de la TUI de agentes al árbol de componentes y lector tolerante con aviso.
- [ ] T5 · Docs, e2e, notas de limpieza manual, `## Spec viva` integrada con `axiom archive integrate --odd arranque-minimo` y criterio `rg`.

## Criterio de terminado

- El flujo de la TUI va de agentes al árbol de dependencias sin pasar por persona ni presets.
- Solo se instala la persona de Axiom.
- Un estado con persona o preset retirados carga con un aviso.
- `rg -il gentleman` sin resultados fuera de los históricos, del lector tolerante y de las excepciones de la decisión anterior.
- `go vet` y `go test` de los paquetes tocados en verde; la batería completa, en la CI de la PR.

## Spec viva

Retira requisitos y escenarios que solo hablaban de la persona `gentleman`/`neutral`, los presets, los temas, el logo de OpenCode y el agent builder, y adapta a Axiom como única persona los que siguen en vigor.

### Capacidad: tui-spanish-localization

```openspec
## MODIFIED Requirements

### Requirement: Localización al Castellano del Menú de Bienvenida (REQ-16.2)
El menú principal de bienvenida (`WelcomeOptions`) y sus microtextos auxiliares deben presentarse íntegramente en español (castellano peninsular).
(Previously: la lista de etiquetas incluía «Reiniciar almacén de revisiones» y «Revisión formal RDD», retiradas con RDD (C-01-e), y «Crear agente personalizado», retirada con el agent builder (C-08-f).)

#### Scenario: Visualización de opciones en español
- **DADO QUE** el usuario se encuentra en la pantalla de bienvenida
- **CUANDO** se renderiza la lista de opciones
- **ENTONCES** las etiquetas deben ser:
  - "Iniciar instalación"
  - "Actualizar herramientas" / "Actualizar herramientas (al día)" / "Actualizar herramientas ★"
  - "Sincronizar configuraciones"
  - "Actualizar y sincronizar"
  - "Configurar modelos"
  - "Plugins comunitarios de OpenCode"
  - "Desinstalar plugin de OpenCode"
  - "Perfiles SDD de OpenCode"
  - "Gestionar respaldos"
  - "Desinstalación gestionada"
  - "Herramientas y plugins comunitarios"
  - "Salir"

#### Scenario: Microtextos de navegación y encabezados
- **DADO QUE** se visualiza el frame del menú
- **CUANDO** se renderizan los textos fijos
- **ENTONCES** la cabecera de la lista debe mostrar "Menú"
- **Y** la barra inferior de ayuda debe mostrar `j/k: navegar • enter: seleccionar • q: salir`.
```

## Progreso

- 2026-11-26: worktree creado desde `origin/main` `814b3ce5`; documento creado. Carril confirmado por la coordinación.
- 2026-11-26: T1 hecha (en línea). `go vet ./internal/tui/... ./internal/skillcheck/` limpio; `go test ./internal/tui/ ./internal/tui/screens/` en verde. El menú pierde una opción: se desplazan los índices de cursor de los tests de navegación y `TestWelcomeView_MinimumResizePreservesNonzeroCursor` baja a 19 filas para seguir cayendo en la vista mínima.
