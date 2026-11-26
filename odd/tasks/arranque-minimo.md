# ODD: Arranque mínimo, sin TUI, persona, presets, temas, agent builder ni logo heredados (arranque-minimo)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/arranque-minimo.md`. Espejo Engram: `odd/arranque-minimo/tasks`.
> Rama: `inc/arranque-minimo` · worktree `C:\repos\axiom-wt\arranque-minimo\axiom`. Cambio C-08-f del plan maestro (`docs/plan/backlog-2026-11/08-adelgazamiento.md`), ola A5; decisiones P18, P19 y P20, y ampliación del usuario del 2026-11-26 (retirar la TUI).

## Objetivo

Que la instalación sea detectar agentes, elegir componentes e instalar, sin pasos heredados de gentle-ai, y que `axiom` sin argumentos abra la UI en lugar de la TUI.

## Problema y porqué

Decisión del usuario: «Me sobran muchos de inicio». Persona `gentleman` y `neutral`, los cuatro presets, la migración de temas, el logo de OpenCode y el agent builder son herencia de gentle-ai: entre 13.500 y 14.500 líneas que cuestan en cada cambio y en cada ejecución de pruebas.

## Decisiones

- **P18 (usuario: «Sí, retirarlo»).** Se retira el agent builder: `internal/agentbuilder`, sus pantallas, la opción «Crear agente personalizado» del menú y `PRD-AGENT-BUILDER.md`.
- **P19 (usuario: «Sí, solo Axiom»).** Solo queda la persona de Axiom (`persona-axiom.md` y `output-style-axiom.md`). Se retiran `gentleman`, `neutral`, el alias `gentleman-neutral-artifacts`, la opción «sin persona gestionada» (`custom`) y la pantalla de elección. Quien no quiera persona desmarca el componente `persona` en el árbol de componentes.
- **Ampliación: se retira la TUI (usuario, 2026-11-26).** Pedida desde la sesión «Optimizar tiempos de test y Axiom» con la cita del usuario («quitamos la tui y si es en c-08-f ya esta en marcha una session con ello le tendras que avisar a la session, y la duda axiom si abrira la ui.») y confirmada por el usuario en esta sesión («si»). Sale `internal/tui` entero; `axiom` sin argumentos en una terminal abre la UI como `axiom ui` (sin TTY, el error de siempre); desaparece el verbo `axiom tui`. `install`, `sync`, `upgrade`, `uninstall`, `restore`, `backup` y `doctor` siguen en la CLI; instalar agentes desde la UI es C-06-g y no se adelanta. Sigue en ODD: el requisito de arranque lo fija el usuario. Con la TUI fuera, T4 (persona) y T5 (presets) solo tocan lo que no es TUI (flags, componentes, assets y lectores tolerantes).
- **P20 (usuario: «Sí, borrarlas ya»).** Sin lápidas: ni limpieza en `uninstall`, `sync` o `upgrade` de lo retirado (output styles y personas de `gentleman`/`neutral`, temas, `gentle-logo.tsx`). La persona usuaria limpia a mano: `docs/releases/notas-retirada-arranque.md`.
- **Presets.** Los cuatro (`full-gentleman`, `ecosystem-only`, `minimal`, `custom`) se sustituyen por un conjunto por defecto: el núcleo que instalaba `full-gentleman` (Engram, SDD, skills, Context7, permisos y persona), editable siempre en el árbol de componentes. Desaparecen las opciones `--persona` y `--preset` de `axiom install`.
- **Lector tolerante (plan, C-08-f).** Un estado con `persona` o `preset` retirados carga con un aviso en `sync` y la siguiente escritura ya no los guarda; los componentes `theme`, `claude-theme` y `opencode-gentle-logo` se ignoran con el mecanismo de C-08-e (`model.IsKnownComponentID`).
- **Coordinación (2026-11-26).** Sigue en ODD con `## Spec viva`: los MODIFIED son consecuencias mecánicas de P18 y P19 y el único cambio de semántica (fallback a Axiom con aviso) lo fija el plan. Si aparece un requisito con conducta nueva no fijada por P18, P19 o la lectura tolerante, se para y se avisa (sería SDD).
- **Errata del plan: `components/opencodedefault` no es el logo (coordinación, 2026-11-26).** La sección 08 lo cita como `ComponentOpenCodeGentleLogo`, pero gestiona el `default_agent` de OpenCode (`axiom-orchestrator`) y su fichero de propiedad, que usan `components/sdd`, `agentguidance`, `cli/run` y `uninstall`: es núcleo SDD y se conserva. El logo heredado es el componente `opencode-gentle-logo`, el plugin `gentle-logo` de `opencodeplugin` y su `.tsx`, y eso es lo que se retira.
- **Criterio `rg -il gentleman`.** No es alcanzable literalmente: Engram y su tap de Homebrew viven en la organización `Gentleman-Programming`, y hay atribución (`LICENSE`, `TRADEMARKS.md`, `CONTRIBUTORS.md`, `author:` de las skills). Se aplica a la persona, presets y temas `gentleman`; quedan esas referencias, la migración del agente SDD `gentleman` de OpenCode (`components/sdd` y su limpieza en `uninstall`, núcleo fuera de alcance), la retirada de `agent.gentleman` que ya hacía la persona de Axiom, la cabecera `description:` de las instrucciones SDD de VS Code (`filemerge.SDDInstructionsFrontmatter`: cambiarla rompería el reconocimiento por bytes de los ficheros ya instalados), los literales de los lectores tolerantes y sus tests, y los históricos.
- **Históricos.** No se tocan `docs/archive`, `openspec/changes/archive`, `docs/audits`, `docs/plan`, `docs/releases` existentes, `docs/upstream-absorption-ledger.md`, `CONTRIBUTORS.md` ni los `odd/tasks/` cerrados.

## Alcance autorizado

Lo que lista C-08-f en la sección 08: `internal/components/{persona,theme,opencodedefault}`, `internal/agentbuilder` y `tui/screens/agent_builder_*`, `model` (`PersonaID`, `PresetID`, `presets.go`), `catalog/components.go`, TUI (`router.go`, `persona.go`, `preset.go`, `welcome.go`, `model.go`), assets de persona, goldens `persona-*`, `tui/preset_flow_test.go`; y lo que los consume: `cli` (`install`, `validate`, `run`, `sync`, `dryrun`), `state`, `planner`, `app`, `components/skills/presets.go`, `components/uninstall`, `update/upgrade`, docs y e2e.

Fuera de alcance: el núcleo (SDD/ODD, skills, Engram, guías, MCP, permisos, CodeGraph).

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y no hay elección del usuario. Comprobaciones con `go vet` y `go test` de los paquetes tocados (`-run` en `cli`), sin la batería completa ni binarios sueltos de `axiom`. El lector tolerante se prueba con un test nuevo.

## Entrega

`single-pr`: una PR con un commit por tarea. Etiqueta `type:refactor`. Tamaño XL tras la ampliación: unas 45.000 líneas, casi todo borrado (unas 14.000 del alcance original y unas 30.700 de la TUI).

## Tareas

- [x] T1 · Retirar el agent builder: paquete, pantallas, rutas, opción del menú, `PRD-AGENT-BUILDER.md` y sus tests.
- [x] T2 · Retirar temas y logo de OpenCode: `components/theme`, componentes `theme`/`claude-theme`/`opencode-gentle-logo` y plugin `gentle-logo`, su limpieza en `uninstall`/`upgrade`/`sync`, y tests.
- [x] T3 · Retirar la TUI: `internal/tui` entero, su arranque en `app` (`runTUI` y envoltorios), el verbo `axiom tui`, los tests de paridad TUI↔UI y el código que solo usaba la TUI; `axiom` sin argumentos abre la UI.
- [x] T4 · Persona solo Axiom: `PersonaID`, componente `persona`, assets y goldens de `gentleman`/`neutral`, `--persona`, fallback de `sync` y lector tolerante con aviso.
- [x] T5 · Presets fuera: conjunto por defecto, `--preset`, `SkillsForPreset` y lector tolerante con aviso.
- [x] T6 · Docs, e2e, notas de limpieza manual, `## Spec viva` integrada con `axiom archive integrate --odd arranque-minimo` y criterio `rg`.

Ruta: T1, en línea. T2, escritor delegado (disparador: proteger el contexto del padre, sesión larga; ~40 ficheros mecánicos), revisado por el padre con una comprobación puntual; la retirada de `--include-theme` la hizo el padre en línea. T3, escritor delegado (mismo disparador; borrado guiado por el compilador y las guardas de CI), y un lector delegado clasifica los requisitos de las specs vivas que hablan de la TUI (disparador de mapeo: decenas de specs).

## Criterio de terminado

- No queda TUI: `axiom` sin argumentos abre la UI y no hay pasos de persona ni presets.
- Solo se instala la persona de Axiom.
- Un estado con persona o preset retirados carga con un aviso.
- `rg -il gentleman` sin resultados fuera de los históricos, del lector tolerante y de las excepciones de la decisión anterior.
- `go vet` y `go test` de los paquetes tocados en verde; la batería completa, en la CI de la PR.

## Spec viva

Retira requisitos y escenarios que solo hablaban de la persona `gentleman`/`neutral`, los presets, los temas, el logo de OpenCode, el agent builder y la TUI, y adapta los que siguen en vigor (Axiom como única persona; `axiom` sin argumentos abre la UI). REQ-14.2 de `axiom-tui-branding` alinea además la spec con el código sin TTY: la spec decía «imprime la ayuda con código 0» y el código ya devolvía un error, que es el comportamiento que el usuario manda conservar. `tui-spanish-localization` e `installer-picker-navigation` se quedan sin requisitos y se borran tras integrar; el texto narrativo fuera de requisitos (cabeceras y propósitos que citan la TUI) se corrige a mano en el mismo commit.

### Capacidad: axiom-tui-branding

```openspec
## REMOVED Requirements

### Requirement: Logotipo ASCII y Lema de Axiom en TUI (REQ-14.1)

(Reason: C-08-f retira la TUI; la pantalla de bienvenida de la TUI, su logotipo ASCII y su lema desaparecen con `internal/tui`.)

## MODIFIED Requirements

### Requirement: Lanzamiento de TUI interactiva por defecto y subcomando axiom tui (REQ-14.2)
Al invocar `axiom` sin argumentos, si la sesión cuenta con terminal interactivo (TTY) tanto en entrada estándar como en salida estándar, DEBE abrirse la UI web, con el mismo comportamiento que `axiom ui` (REQ-3.1 de `local-web-dashboard`). Si no hay TTY disponible, DEBE terminar con un error y código distinto de `0` que indica que, para uso no interactivo, se usen `--version`, `axiom update` o `--help`. La opción `--help` imprime la ayuda textual. El subcomando `axiom tui` deja de existir: no queda ninguna interfaz de terminal interactiva.
(Previously: sin argumentos y con TTY se iniciaba la TUI de Bubbletea; sin TTY se imprimía la ayuda con código `0`; el subcomando `axiom tui` lanzaba siempre la TUI.)

#### Scenario: Ejecución de axiom sin argumentos en sesión interactiva TTY
- **DADO** un entorno con terminal interactivo TTY (stdin y stdout son TTY)
- **CUANDO** el usuario ejecuta `axiom` sin argumentos adicionales
- **ENTONCES** se abre la UI web igual que con `axiom ui`

#### Scenario: Ejecución de axiom sin argumentos en entorno no interactivo (CI o tubería)
- **DADO** un entorno donde stdin o stdout no son un terminal TTY
- **CUANDO** se invoca `axiom` sin argumentos
- **ENTONCES** el comando termina con un error y código distinto de `0`
- **Y** el mensaje indica que para uso no interactivo se usen `--version`, `axiom update` o `--help`

#### Scenario: El subcomando axiom tui ya no existe
- **DADO** un terminal interactivo
- **CUANDO** el usuario ejecuta `axiom tui`
- **ENTONCES** no se lanza ninguna interfaz de terminal
```

### Capacidad: tui-spanish-localization

```openspec
## REMOVED Requirements

### Requirement: Desacoplamiento de Comunicados y Avisos Remotos Upstream (REQ-16.1)

(Reason: C-08-f retira la TUI; trata solo de la cabecera de la TUI.)

### Requirement: Localización al Castellano del Menú de Bienvenida (REQ-16.2)

(Reason: C-08-f retira la TUI; trata solo del menú de bienvenida de la TUI. Sustituye al MODIFIED previo de este mismo documento, que debe eliminarse.)

### Requirement: Localización de Submenús y Pantallas de Configuración (REQ-16.3)

(Reason: C-08-f retira la TUI; trata solo de las pantallas de la TUI.)

### Requirement: Determinismo en la Máquina de Estados y Navegación (REQ-16.4)

(Reason: C-08-f retira la TUI; trata solo de la máquina de estados Bubbletea de la TUI.)
```

### Capacidad: tui-ui-parity

```openspec
## REMOVED Requirements

### Requirement: Submenú Unificado de Proyectos y Gobernanza SDD en TUI (REQ-17.5)

(Reason: C-08-f retira la TUI; trata solo de la pantalla de bienvenida de la TUI.)

### Requirement: Pantalla de Gestión Multi-Proyecto en TUI (REQ-17.6)

(Reason: C-08-f retira la TUI; trata solo de `ScreenHubProjects`; la gestión multi-proyecto sigue en la UI web.)

### Requirement: Pantalla de Ciclo de Vida de Incrementos SDD en TUI (REQ-17.7)

(Reason: C-08-f retira la TUI; trata solo de `ScreenSDDIncrements`; la UI web conserva el ciclo de vida de incrementos.)

### Requirement: Pantalla de Monitor Multi-Rol y Barrera Fan-In en TUI (REQ-17.8)

(Reason: C-08-f retira la TUI; trata solo de `ScreenMultiRole`.)

### Requirement: Pantalla de Visor de Handoffs Estructurados en TUI (REQ-17.9)

(Reason: C-08-f retira la TUI; trata solo de `ScreenHandoffs`.)

### Requirement: Pantalla de Catálogo de Especificaciones Vivas en TUI (REQ-17.10)

(Reason: C-08-f retira la TUI; trata solo de `ScreenLivingDoc`.)
```

### Capacidad: installer-picker-navigation

```openspec
## REMOVED Requirements

### Requirement: Declarative picker flow ordering

(Reason: C-08-f retira la TUI; trata solo de la cadena de selectores del instalador TUI.)

### Requirement: Symmetric reverse navigation

(Reason: C-08-f retira la TUI; trata solo de la navegación inversa de esa cadena de selectores.)

### Requirement: Cross-cutting guards remain outside the slice

(Reason: C-08-f retira la TUI; trata solo de las guardas de navegación de la TUI.)
```

### Capacidad: axiom-updater-resilience

```openspec
## REMOVED Requirements

### Requirement: Paridad TUI: encadenamiento y protección de reinicio (REQ-22.6)

(Reason: C-08-f retira la TUI; trata solo de `ScreenUpgradeSync`; el encadenamiento `upgrade` ➔ `sync` con protección de reinicio sigue cubierto en la Web UI.)

## MODIFIED Requirements

### Requirement: No-regresión del verbo CLI axiom upgrade (REQ-22.5)

El comando de terminal `axiom upgrade` DEBE mantenerse estrictamente como una operación **solo-binario**; NO DEBE encadenar automáticamente `sync` ni `install`. La composición de `upgrade` ➔ `sync` es exclusiva de la superficie reactiva interactiva (Web UI).
(Previously: la composición era exclusiva de las superficies reactivas interactivas (Web UI y TUI).)

#### Scenario: Ejecución de axiom upgrade desde consola
- **DADO** la invocación de `axiom upgrade` en CLI
- **CUANDO** se completa la actualización de herramientas
- **ENTONCES** solo se actualizan los binarios
- **Y** no se mutan configuraciones ni se disparan sincronizaciones de agentes

### Requirement: Identidad visible del fork en vistas de actualización (REQ-22.7)

Todos los mensajes de confirmación, cabeceras, títulos y diálogos en la Web UI DEBEN referirse exclusivamente a la identidad de `axiom`.
(Previously: la identidad aplicaba a TUI y Web UI; el escenario comprobaba la pantalla `ScreenUpgradeSync`.)

#### Scenario: Textos de interfaz limpios
- **DADO** la vista de actualización y sincronización de la Web UI
- **CUANDO** se renderizan los diálogos de progreso
- **ENTONCES** todos los textos nombran `axiom`
```

### Capacidad: change-branch-lifecycle

```openspec
## MODIFIED Requirements

### Requirement: Guarda A en Todo Escritor Go de la Carpeta del Cambio (REQ-25.19)

Todo escritor Go que cree o modifique ficheros dentro de la carpeta de un cambio DEBE consultar el veredicto de REQ-25.18 antes de escribir y, ante `wrong_branch`, DEBE negarse sin escribir nada y devolver el error tipado con la continuación. Esto incluye como mínimo: la creación de un incremento (el escritor compartido por el dashboard y `axiom change create`), la migración al acumulativo, la creación de handoff (`axiom handoff create` y el servicio del dashboard), `kickoff seal`, el registro de compuertas (`gate record`), el marcador de instancia del consentimiento de autoridad de edición, `axiom archive context prepare` y `axiom archive context verify` (plan y traza del contexto, que consultan el veredicto en el contexto de archivado), `axiom archive adopt scan --change` (inventario de adopción) y `axiom sdd archive-compose` (que consulta el veredicto en el contexto de archivado). El dashboard y `axiom change create` DEBEN ejecutar `ensure` y después escribir con el escritor guardado; el escritor guardado DEBE negarse fuera de rama aunque se invoque sin haber ejecutado `ensure`. Cualquier escritura que alcance la carpeta de un cambio pasa por un escritor guardado. La guarda NO DEBE bloquear cambios exentos (REQ-25.18). Cada escritor DEBE tener una prueba propia de negativa.
(Previously: la lista incluía `axiom sdd integrate` (`integration.yaml`), retirado; ahora incluye `axiom archive context prepare|verify` y `axiom archive adopt scan --change`)
(Previously: añadía que la TUI solo lista cambios y no ejecuta `ensure`, con un escenario propio.)

#### Scenario: Cada escritor se niega fuera de rama
- **DADO** un cambio con instantánea y el repositorio del planning home desplegado en la rama de integración
- **CUANDO** se invoca, por separado, cada escritor de la lista
- **ENTONCES** cada uno se niega con `wrong_branch` y la continuación de `ensure`
- **Y** ningún fichero de la carpeta del cambio se crea ni se modifica

#### Scenario: Cada escritor escribe en la rama correcta
- **DADO** el mismo cambio con el repositorio desplegado en `feat/x`
- **CUANDO** se invoca cada escritor que opera en el contexto ordinario
- **ENTONCES** escribe con normalidad

#### Scenario: `archive-compose` solo escribe en la rama de archivado
- **DADO** un cambio fusionado y el repositorio desplegado, por separado, en `feat/x` y en `feat/x-archive`
- **CUANDO** se invoca `axiom sdd archive-compose`
- **ENTONCES** se niega con `wrong_branch` y la continuación `axiom branch ensure --cwd <ruta> --change x --archive` en `feat/x`
- **Y** escribe con normalidad en `feat/x-archive`

#### Scenario: `axiom archive context prepare` solo escribe en la rama de archivado
- **DADO** un cambio fusionado y el repositorio desplegado, por separado, en `feat/x` y en `feat/x-archive`
- **CUANDO** se invoca `axiom archive context prepare --change x`
- **ENTONCES** se niega con `wrong_branch` y la continuación `axiom branch ensure --cwd <ruta> --change x --archive` en `feat/x`
- **Y** escribe con normalidad en `feat/x-archive`

#### Scenario: El dashboard y `axiom change create` ejecutan `ensure` y después escriben
- **DADO** un monorepo en modo `worktree` y el checkout principal en `main`
- **CUANDO** se crea el incremento `x` desde el dashboard o con `axiom change create`
- **ENTONCES** se ejecuta `ensure` para `x` antes de escribir
- **Y** el escritor guardado escribe `proposal.md` en el worktree de `feat/x`

#### Scenario: El escritor guardado invocado sin `ensure` en rama equivocada se niega
- **DADO** un monorepo con la política por defecto y el checkout en `main`
- **CUANDO** se invoca directamente el escritor guardado de creación del incremento `x` sin haber ejecutado `ensure`
- **ENTONCES** se niega con `wrong_branch` y la continuación `axiom branch ensure --change x`
- **Y** no se crea la carpeta `openspec/changes/x`

#### Scenario: Un sumidero acumulativo está exento por nombre
- **DADO** el checkout en la rama de integración
- **CUANDO** se escribe en `openspec/changes/e2e-cumulative` o en `openspec/changes/cumulative-core`
- **ENTONCES** la escritura procede
```

### Capacidad: local-web-dashboard

```openspec
## MODIFIED Requirements

### Requirement: Los Listados Agregan los Cambios de los Worktrees Registrados (REQ-25.34)

Los cambios en curso viven en worktrees. El servidor DEBE agregar, a los cambios activos del checkout principal, los cambios activos de los worktrees registrados en el registro local de `axiom branch` (REQ-25.14) para el proyecto activo en `GET /api/increments`, y DEBE resolver también esos cambios en `GET /api/increments/{name}`, `GET /api/roles?change={name}` y `GET /api/handoffs?change={name}`. Cada incremento listado desde un worktree DEBE indicar la ruta absoluta de su worktree y su rama. Si un mismo cambio activo figura en el checkout principal (por ejemplo, un sumidero acumulativo) y en un worktree, prevalece el del worktree; si figura en el worktree de la rama del cambio y en el de la rama de archivado, prevalece el de la rama de archivado. El dashboard DEBE mostrar los cambios pendientes de archivar (REQ-25.20 de `change-branch-lifecycle`), es decir, los cambios activos presentes en la rama de integración que aún no se han archivado, con la continuación para archivarlos. El aviso de cambios pendientes de archivar DEBE viajar en la cabecera `X-Axiom-Notes` de la respuesta de `GET /api/increments`; los avisos de este requisito, incluido el de una entrada del registro cuyo worktree ya no existe en disco (que DEBE omitirse sin error), viajan en esa misma cabecera. Los cambios pendientes NO producen ningún error ni bloqueo. Las escrituras del dashboard sobre un cambio que vive en un worktree DEBEN dirigirse a ese worktree.
(Previously: el requisito se apoyaba en un invariante de la rama de integración, eliminado por D17, y los avisos viajaban en el cuerpo de la respuesta)
(Previously: añadía que la pantalla de ciclo de vida de incrementos de la TUI listaba el mismo conjunto de cambios y solo listaba, con un escenario propio.)

#### Scenario: Un cambio de un worktree figura en el listado
- **DADO** un proyecto cuyo checkout principal tiene solo `archive/` bajo `openspec/changes/` y un cambio activo `x` en un worktree registrado
- **CUANDO** se realiza `GET /api/increments`
- **ENTONCES** el listado incluye `x` como activo con la ruta absoluta de su worktree y su rama

#### Scenario: El detalle de un cambio en worktree se resuelve
- **DADO** el mismo cambio `x` en un worktree registrado
- **CUANDO** se solicita `GET /api/increments/x`
- **ENTONCES** se devuelve `200 OK` con los artefactos del cambio leídos del worktree

#### Scenario: El worktree prevalece sobre el checkout principal
- **DADO** un cambio activo `e2e-cumulative` presente en el checkout principal y en un worktree registrado
- **CUANDO** se realiza `GET /api/increments`
- **ENTONCES** el cambio aparece una sola vez, con los datos del worktree

#### Scenario: El worktree de la rama de archivado prevalece sobre el de la rama del cambio
- **DADO** el cambio `x` presente en los worktrees de `feat/x` y de `feat/x-archive`, ambos registrados
- **CUANDO** se realiza `GET /api/increments`
- **ENTONCES** `x` aparece una sola vez, con los datos y la rama del worktree de `feat/x-archive`

#### Scenario: Un worktree desaparecido se omite con aviso
- **DADO** una entrada del registro local cuyo worktree fue borrado del disco
- **CUANDO** se realiza `GET /api/increments`
- **ENTONCES** la respuesta es `200 OK` sin ese cambio
- **Y** la cabecera `X-Axiom-Notes` incluye un aviso que nombra la ruta ausente

#### Scenario: Un cambio fusionado y sin archivar se muestra como pendiente
- **DADO** un cambio `y` fusionado y presente en la rama de integración, sin archivar
- **CUANDO** se realiza `GET /api/increments`
- **ENTONCES** el listado marca `y` como pendiente de archivar con la continuación `axiom branch ensure --change y --archive`
- **Y** la cabecera `X-Axiom-Notes` incluye el aviso de cambios pendientes de archivar con el nombre `y`
- **Y** la respuesta es `200 OK`

#### Scenario: Sin cambios pendientes no hay aviso de pendientes
- **DADO** una rama de integración cuyo `changes/` solo contiene `archive/` y los sumideros exentos
- **CUANDO** se realiza `GET /api/increments`
- **ENTONCES** la cabecera `X-Axiom-Notes` no incluye ningún aviso de cambios pendientes de archivar

#### Scenario: Una escritura sobre un cambio en worktree se dirige al worktree
- **DADO** un cambio `x` en un worktree registrado
- **CUANDO** el cliente crea un handoff para `x` desde el dashboard
- **ENTONCES** `handoff.md` se escribe en la carpeta del cambio del worktree
```

### Capacidad: planning-home-resolution

```openspec
## MODIFIED Requirements

### Requirement: Los Consumidores de Catálogo Comparten la Convención de Raíz (REQ-24.10)

El servicio de documentación viva (livingdoc, incluido el sintetizador de arranque en frío), knowledge (raíz de especificaciones por defecto) y el dashboard web DEBEN obtener la raíz de especificaciones y el planning home del primitivo de REQ-24.1 y NO DEBEN mantener una convención propia. Las especificaciones vivas, los cambios y el índice residen en `<planning home>/specs`, `<planning home>/changes` y `<planning home>/INDEX.md`. Cuando en un proyecto `monorepo-embedded` con `specs_repository` distinto de `""`, `.` y `openspec` existe `<raíz>/<specs_repository>/specs` y difiere de `<planning home>/specs`, los consumidores de catálogo DEBEN leer solo `<planning home>/specs` y emitir un aviso que nombre ambas rutas.
(Previously: los consumidores de catálogo incluían la TUI, y el escenario comprobaba también su pantalla de ciclo de vida de incrementos.)

#### Scenario: Catálogo del workspace canónico
- **DADO** el workspace canónico con `<maestra>/repo-specs/openspec/specs/workspace-topology/spec.md`
- **CUANDO** se ejecuta `axiom archive list` desde `<maestra>`
- **ENTONCES** el listado incluye el dominio `workspace-topology`

#### Scenario: Raíz de especificaciones por defecto de knowledge
- **DADO** el workspace canónico
- **CUANDO** se ejecuta `axiom knowledge sweep` sin indicar raíz de especificaciones desde `<maestra>`
- **ENTONCES** la raíz de especificaciones usada es el planning home `<maestra>/repo-specs/openspec`

#### Scenario: Ubicación divergente en un proyecto embebido
- **DADO** un proyecto `monorepo-embedded` con `specs_repository: specs`, un `specs/specs/antiguo/spec.md` y un `openspec/specs/vigente/spec.md`
- **CUANDO** se ejecuta `axiom archive list`
- **ENTONCES** el listado incluye `vigente` y no incluye `antiguo`
- **Y** se emite un aviso que nombra `specs/specs` y `openspec/specs`

#### Scenario: El dashboard lee el planning home canónico
- **DADO** el dashboard web con el workspace canónico activo (ruta registrada `<maestra>`)
- **CUANDO** se consultan `GET /api/increments` y `GET /api/archive/specs`
- **ENTONCES** ambas respuestas proceden de `<maestra>/repo-specs/openspec`
```

### Capacidad: antigravity-support

```openspec
## MODIFIED Requirements

### Requirement: Unified public agent ID

The system MUST expose Antigravity as `antigravity` and MUST NOT expose a separate public `antigravity-cli` agent option.
(Previously: the scenario also checked that the option was not listed in the TUI.)

#### Scenario: Install uses the unified Antigravity agent

- GIVEN the installer is invoked for `antigravity`
- WHEN agent validation runs
- THEN `antigravity` is accepted
- AND the separate `antigravity-cli` option is not listed in the catalog.
```

### Capacidad: upstream-absorption-protocol

```openspec
## MODIFIED Requirements

### Requirement: Inventario de no-reversión por tanda (REQ-20.3)

El sistema DEBE tratar las entradas V1, V2, V3, V4, V5, V6 y V8 del inventario de divergencias deliberadas (renombrado de producto, detección dual de marcadores `axiom:`, agente `axiom-orchestrator` y comandos slash sin prefijo, supresión del logo de OpenCode, no forzado de `theme`, persona Axiom en castellano peninsular, y la capa Hub multi-repositorio) como no-reversibles en toda tanda de absorción.
(Previously: la entrada V3 hablaba del logo «en presets», retirados por C-08-f, y la V6 incluía la localización íntegra de la TUI, retirada por C-08-f.)

#### Scenario: Reversión de una divergencia firme rechazada
- **DADO** una tanda cuyo diff revierte, total o parcialmente, la detección dual de marcadores `axiom:` (V2) o cualquier otra entrada firme (V1, V3, V4, V5, V6 u V8)
- **CUANDO** se evalúa la tanda
- **ENTONCES** se rechaza
- **Y** se detiene el incremento hasta el último estado verde
```

### Capacidad: persona-behavior-contract

```openspec
## REMOVED Requirements

### Requirement: Neutral Mentor Behavior Parity

(Reason: C-08-f retira las personas `neutral` y `gentleman` (P19); solo queda la persona de Axiom.)

### Requirement: Neutral Interaction Discipline

(Reason: C-08-f retira la persona `neutral` (P19); la disciplina de interacción de Axiom vive en su propio output style.)

### Requirement: Claude Neutral Output Style Contract

(Reason: C-08-f retira la persona `neutral` y su output style de Claude (P19).)

### Requirement: Generic Neutral Asset Parity

(Reason: C-08-f retira la persona `neutral` y sus assets genéricos (P19).)

### Requirement: Explicit Persona Selection Preservation

(Reason: C-08-f retira la elección de persona (P19); no queda ninguna selección explícita que preservar.)

### Requirement: Persona Drift Reconciliation

(Reason: C-08-f retira las variantes `gentleman` y `neutral`, las únicas que el requisito reconciliaba (P19).)

## MODIFIED Requirements

### Requirement: Artifact Language Independence

Persona voice MUST govern only direct chat replies to the user. Generated technical artifacts MUST default to English and neutral professional wording regardless of the persona, unless the user explicitly requests a different artifact language or the existing project artifact convention requires it.
(Previously: the scenarios used the retired `neutral` and `gentleman` personas.)

#### Scenario: Axiom keeps generated artifacts free of conversational voice

- GIVEN the Axiom persona is installed
- WHEN the system generates code, identifiers, comments, UI copy, documentation, commit messages, PR descriptions, SDD artifacts, or tests without an explicit request for another language or tone
- THEN the generated artifact uses neutral professional wording without the persona's conversational register
- AND it defaults to English unless the project artifact convention requires otherwise

### Requirement: Safe Persona Fallback Semantics

When the `persona` component is selected, install and sync MUST apply the Axiom persona. Persisted persona state that is missing, empty, `axiom`, or a retired or unknown value (such as `gentleman`, `neutral`, `gentleman-neutral-artifacts` or `custom`) MUST NOT block sync and MUST NOT select any other persona. A retired or unknown persisted value MUST be reported with a warning and MUST NOT be written back to the state.
(Previously: the fallback for missing or invalid state was the `neutral` persona, and an explicit `gentleman` selection was honored.)

#### Scenario: Missing persisted persona applies Axiom

- GIVEN persisted persona state is absent
- WHEN sync applies the `persona` component
- THEN it applies the Axiom persona

#### Scenario: Retired persisted persona applies Axiom with a warning

- GIVEN persisted state contains persona `gentleman`
- WHEN sync loads the state and applies the `persona` component
- THEN it applies the Axiom persona
- AND it reports that the persisted persona no longer exists
- AND the next state write no longer contains the retired persona

### Requirement: Canonical Tone Channel for Output-Style-Capable Adapters

For any adapter with an active output-style channel — Claude Code (gated by `SupportsOutputStyles()`) — the output-style asset MUST be the exclusive source of persona tone, language, and philosophy. The system-prompt persona section for these adapters MUST be reduced to a residual block containing only `## Rules`, `## Expertise`, `## Contextual Skill Loading (MANDATORY)` and a one-line pointer to the output style. The residual MUST NOT duplicate tone/language/philosophy content.
(Previously: the scenario covered the retired `gentleman` and `neutral` personas; Kimi was an explicit carve-out before C-08-d.)

#### Scenario: Claude residual section carries no tone content

- GIVEN Claude assets are generated with the Axiom persona
- WHEN the CLAUDE.md persona section is inspected
- THEN it contains only Rules, Expertise, Contextual Skill Loading and a pointer to the output style
- AND it contains no tone, language, or philosophy prose

### Requirement: Axiom Persona Definition and Default Selection (REQ-10.1 & REQ-10.2)

The system MUST support `axiom` as its only managed persona. The persona MUST prescribe senior architect guidance in Peninsular Spanish (castellano peninsular) with professional tuteo ("tienes", "haz", "revisa"), strictly prohibiting forced Rioplatense voseo ("che", "tenés", "podés", "hacé") and informal regionalisms. The CLI MUST NOT offer a persona choice: selecting the `persona` component installs the Axiom persona, and deselecting it installs none.
(Previously: `axiom` was the default among several personas, and the TUI and CLI presented it as the default selection.)

#### Scenario: Axiom is the only persona and uses Peninsular Spanish

- GIVEN the `persona` component is selected
- WHEN Axiom installs it
- THEN the installed persona is `axiom`
- AND its instructions enforce Peninsular Spanish without voseo or Rioplatense regionalisms
```

### Capacidad: axiom-skills-index-governance

```openspec
## MODIFIED Requirements

### Requirement: Saneamiento del Preset Predeterminado de Skills (REQ-46.9)

El conjunto de skills que Axiom instala por defecto DEBE contener únicamente skills de producto para usuarios, segregando las skills de colaboración interna del repositorio (`contributorSkills`).
(Previously: hablaba del «preset básico»; C-08-f sustituye los presets por un conjunto por defecto. Trasladado desde REQ-18.5 de `rdd-decoupling-v3-stability`, retirada con RDD (C-01-e).)

#### Scenario: Inspección del conjunto de skills por defecto

- **DADO** el catálogo de skills en `internal/components/skills`
- **CUANDO** se consulta el conjunto de skills por defecto
- **ENTONCES** contiene únicamente skills fundacionales de producto y excluye las de contribución interna
```

### Capacidad: visual-decoupling

```openspec
## REMOVED Requirements

### Requirement: Supresión de gentle-logo en presets de instalación (REQ-09.1)

(Reason: C-08-f retira el componente `opencode-gentle-logo` y el plugin `gentle-logo`; Axiom ya no puede instalar el logo y, sin lápidas (P20), tampoco limpia un `gentle-logo.tsx` previo: lo borra la persona usuaria a mano.)

### Requirement: No forzado de temas en settings.json (REQ-09.2)

(Reason: C-08-f retira el componente de temas; Axiom no tiene ya ningún código que lea o escriba temas.)

### Requirement: Renombrado de archivos de temas opcionales (REQ-09.3)

(Reason: C-08-f retira el componente de temas; Axiom ya no despliega temas, ni `axiom.json` ni `axiom-dark.json`.)
```

## Progreso

- 2026-11-26: worktree creado desde `origin/main` `814b3ce5`; documento creado. Carril confirmado por la coordinación.
- 2026-11-26: T1 hecha (en línea). `go vet ./internal/tui/... ./internal/skillcheck/` limpio; `go test ./internal/tui/ ./internal/tui/screens/` en verde. El menú pierde una opción: se desplazan los índices de cursor de los tests de navegación y `TestWelcomeView_MinimumResizePreservesNonzeroCursor` baja a 19 filas para seguir cayendo en la vista mínima. Commit `0e34d4cf`.
- 2026-11-26: T2 hecha. Fuera `components/theme`, los componentes `theme`, `claude-theme` y `opencode-gentle-logo`, el plugin `gentle-logo` (y su `.tsx`), su limpieza en `sync`, `uninstall` y `upgrade`, y la opción `axiom sync --include-theme`. `opencodedefault` se conserva (errata del plan). Lector tolerante probado con `state.TestReadIgnoresRetiredVisualComponents`. `go vet ./...` limpio; `go test` en verde de `model`, `catalog`, `planner`, `state`, `components/uninstall`, `components/opencodeplugin`, `update/upgrade`, `tui`, `tui/screens` y `cli -run 'Sync|Uninstall|Footprint|Plugin|RunInstall|Component|Install|Backup|Normalize|Restore|ParseSyncFlags'`. Goldens `preset-custom-*-next` regenerados (solo pierden la fila del logo). Commit `53af25bc`.
- 2026-11-26: T3 hecha (escritor delegado, revisado por el padre). Fuera `internal/tui` entero, su cableado en `app`, el verbo `axiom tui`, los tests de paridad TUI↔UI y 56 funciones que solo alcanzaba la TUI (`scripts/deadcode-ratchet.sh` en verde; `--update` solo quita 19 líneas de la línea base). `axiom` sin argumentos conserva TTY, SO, detección, `PendingSync` y aviso de `doctor`, y abre la UI con `runUI(nil)`. Los puntos de entrada de sync que solo usaban la TUI pero que ejercitan unas 100 pruebas de `cli` (`RunSyncWithSelection*`, `SyncResult.Warnings`, `newInstallRuntime`, `SetCommandOutputStreaming`) pasan a `cli/selection_helpers_test.go`, y los ayudantes Codex de `model` a `model/export_test.go`. `go mod tidy` saca bubbles, bubbletea, lipgloss y sus indirectas. `go vet ./...` limpio; `go test` en verde de `app` (sin los que compilan el binario), `cmd/axiom -run 'Dispatch|Help|Verbs|DeadcodeRatchet|WriterInventory'`, `dashboard`, `branch/engine -run Writer`, `assets -run 'Shard|Workflow'`, `model`, `opencode`, `pipeline`, `update`, `backup`, `skillcheck`, `uninstall`, `opencodeplugin` y `cli`/`components` con `-run` focalizado. Pendiente para T6: textos que citan la TUI fuera de Go (`AGENTS.md`, `GEMINI.md`, `axiom.yaml`, `scripts/install.*`, plantillas de `.github`). Commit `a5a53dd3`.
- 2026-11-26: T4 hecha (escritor delegado, revisado por el padre). Fuera `PersonaID`, `Selection.Persona`, `--persona`, el alias, la resolución y migración de persona en `sync`, 7 assets y 8 goldens de `gentleman`/`neutral`, y la limpieza heredada de `filemerge` (`StripLegacyPersonaBlock`) y `persona`. `persona.Inject` sigue quitando `agent.gentleman` de `opencode.json`, como ya hacía con Axiom. Lector tolerante: `state.IgnoredPersona` y aviso `noteIgnoredStatePersona`. Goldens nuevos de la persona de Axiom para no perder cobertura. Commit `49d0d7d8`.
- 2026-11-26: T5 hecha (mismo escritor). `model.DefaultComponents()` (Engram, SDD, skills, Context7, permisos y persona) y `skills.DefaultSkills()` (lo de `full-gentleman`) sustituyen a los presets; fuera `PresetID`, `--preset` y la línea de dry-run. Lector tolerante: `state.IgnoredPreset` con aviso. Cambio de conducta asumido: un estado con `preset: custom` sin skills sincroniza ahora las skills por defecto. `go vet ./...` limpio, `deadcode-ratchet` en verde y `go test` en verde de `model`, `state`, `planner`, `components/{persona,skills,filemerge,uninstall}`, `update/upgrade`, `assets`, `app`, `components -run Golden`, `dashboard -run 'Models|State'` y `cli` con `-run` focalizado. Commit `6d0a420a`.
- 2026-11-26: T6 hecha. Docs, README, PRD, GEMINI, `axiom.yaml`, `scripts/install.*`, plantillas de issues y skill `axiom-branch-pr` sin TUI, persona heredada, presets, temas, logo ni agent builder (escritor delegado, disparador: paralelismo con la integración de specs, superficie disjunta). e2e: 28 casos de presets, personas y temas fuera; quedan dry-run sin flags y con `--component` (`bash -n` en verde). Notas de limpieza manual en `docs/releases/notas-retirada-arranque.md`. `axiom archive integrate --odd arranque-minimo` compuso 13 capacidades; se borran a mano `tui-spanish-localization` e `installer-picker-navigation` (vacías) y sus entradas de `openspec/INDEX.md` (51 dominios), y se corrigen las cabeceras narrativas que citaban la TUI, el logo o los temas. `go test` de `livespec`, `livingdoc` y `workspace -run 'Docs|Declarations'` en verde. Criterio `rg -il gentleman`: solo las excepciones de «Decisiones». Queda fuera de alcance: `AXIOM_NO_ANIMATION` ya no lo lee nadie pero sigue como ejemplo en un escenario de `axiom-user-state-and-env` (REQ-12.3).
