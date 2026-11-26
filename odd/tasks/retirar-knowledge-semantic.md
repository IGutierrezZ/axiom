# ODD: Retirar `knowledge` y `semantic`, con CodeGraph en uso directo (retirar-knowledge-semantic)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/retirar-knowledge-semantic.md`. Espejo Engram: `odd/retirar-knowledge-semantic/tasks`.
> Rama: `inc/retirar-knowledge-semantic` · worktree `C:\repos\axiom-wt\retirar-knowledge-semantic\axiom`. Cambio C-08-l del plan maestro (`docs/plan/backlog-2026-11/08-adelgazamiento.md`), ola A6; decisiones P20 y P23 (README, apartado 6.2).

## Objetivo

Retirar los comandos `axiom knowledge` (`sweep`, `crawl`, `query`) y `axiom semantic` (`status`, `symbols`, `inspect`, `reindex`), sus paquetes y el panel «Código y grafo» del dashboard, y dejar CodeGraph como la única vía de análisis semántico, usado directamente por los agentes (MCP y CLI).

## Problema y porqué

Decisión del usuario (2026-11-26): «Quitamos knowledge y semantic, revisamos en semantic que ahora se use directamente codegraph.» Unas 2.900 líneas de producción y 2.000 de test, más los comandos de `cmd/axiom` y el panel del dashboard, que duplican lo que CodeGraph ya da a los agentes.

## Decisiones

- **P23 (usuario).** El panel semántico del dashboard (`/api/semantic/status`, `symbols`, `dependencies` y `reindex`, la pantalla «Código y grafo» y la comprobación del estado del código en Inicio) se retira sin reimplementarlo sobre CodeGraph.
- **P20 (usuario).** Sin lápidas: nada limpia lo que dejaron las versiones anteriores. La limpieza manual va en `docs/releases/notas-retirada-knowledge-semantic.md`.
- **Lectura tolerante (plan).** `governance.semantic_analysis` de `axiom.yaml` se sigue leyendo y se ignora con un aviso no bloqueante de `workspace.Validate` (lo ven `axiom workspace validate`, `doctor`, el dashboard y la validación de `init`); `internal/hub/init.go` deja de escribirlo.
- **CodeGraph en uso directo (plan).** `internal/components/communitytool` (MCP y bloque de instrucciones) queda intacto y sin referencias a `semantic`.
- **Specs vivas (plan).** Se retiran `knowledge-query-engine`, `knowledge-sweep-command` y `semantic-code`; `knowledge-profile-init` se queda (perfil `knowledge` de `axiom init`), solo pierde `semantic_analysis` de su `axiom.yaml`.
- **Skills de proyecto `skills/knowledge-*` (consecuencia directa, no listada en el plan).** Son skills locales de este repositorio, indexadas en `AGENTS.md`. `knowledge-crawl` describe un flujo que solo existe con `axiom knowledge crawl` (`--plan`, `--record-unit`, `crawl-job.json`): se retira con el comando. `knowledge-query` no depende de `axiom knowledge`; solo pierde la línea del motor AST nativo (`axiom semantic symbols`/`inspect`) y queda sobre CodeGraph directo, que es lo que pide el usuario.
- **Fuera de alcance.** El perfil `knowledge` de `axiom init`, CodeGraph y `communitytool`. La cita a `internal/knowledge/sweep.go` en REQ de `requirement-identifiers` no se toca (C-03-b trabaja en paralelo sobre identificadores).
- **Históricos.** No se tocan `docs/archive`, `openspec/changes/archive`, `docs/audits`, `docs/plan`, `docs/releases` existentes ni los `odd/tasks/` cerrados.

## Alcance autorizado

`internal/knowledge`, `internal/semantic`, `cmd/axiom/main.go` (casos `knowledge` y `semantic`, `runKnowledge*`, `runSemantic*`, ayuda), `internal/dashboard` (`service.go`, `server.go`, assets de la pantalla «Código y grafo» y sus tests), `internal/workspace` (`types.go`, aviso en `Validate`), `internal/hub/init.go`, `axiom.example.yaml`, `docs/manual-de-inicio.md` y demás docs vivas que citen los comandos, `skills/knowledge-*`, `AGENTS.md`, baselines de guardas (`.refusal-ratchet-baseline.txt`, `.deadcode-baseline.txt` y similares) y `openspec/specs/{knowledge-query-engine,knowledge-sweep-command,semantic-code}` más los MODIFIED de la `## Spec viva`.

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y no hay elección del usuario. Comprobaciones con `go build`, `go vet` y `go test` de los paquetes tocados (con `-run` cuando el paquete es grande), sin la batería completa ni binarios sueltos de `axiom`. El aviso de la lectura tolerante se prueba con un test nuevo.

## Entrega

`single-pr`: una PR con un commit por tarea. Etiqueta `type:breaking-change` (desaparecen dos comandos públicos). Previsión: unas 5.500 líneas, casi todo borrado.

## Tareas

- [x] T1 · Retirar el panel «Código y grafo» del dashboard: rutas `/api/semantic/*`, métodos de `Service`, pantalla `semantic.js`, sección de `index.html`, navegación, fuente `semantic` de Inicio, CSS y tests.
- [x] T2 · Retirar `axiom knowledge` y `axiom semantic`: casos y funciones de `cmd/axiom`, ayuda, `internal/knowledge`, `internal/semantic`, skills `knowledge-*`, `AGENTS.md`, docs y baselines.
- [x] T3 · Lectura tolerante de `governance.semantic_analysis`: aviso en `workspace.Validate`, `hub/init.go` deja de escribirla, `axiom.example.yaml`, y test.
- [ ] T4 · `## Spec viva` integrada con `axiom archive integrate --odd retirar-knowledge-semantic`, borrado de las tres specs retiradas e `INDEX.md`, notas de limpieza manual y criterio `rg`.

Ruta: todas en línea (borrado guiado por el compilador; ninguna unidad necesita un mapeo que exceda el presupuesto de evidencia).

## Criterio de terminado

- `rg -l 'internal/(knowledge|semantic)"' -g '*.go'` sin resultados.
- `axiom knowledge` y `axiom semantic` responden como un comando desconocido.
- Un `axiom.yaml` con `semantic_analysis` carga con un aviso.
- `communitytool` sin referencias a `semantic`.
- `go vet` y `go test` de los paquetes tocados en verde; la batería completa, en la CI de la PR.

## Spec viva

(Se completa en T4.)

## Progreso

- 2026-11-26: documento creado; worktree desde `origin/main` en `7464c5a7`.
- 2026-11-26: T1 hecha. Fuera las rutas `/api/semantic/*`, los métodos semánticos de `Service`, `semantic.js`, `core/lazy.js` (solo lo usaba Código), la sección `view-codigo` y su entrada de navegación, la ruta `codigo`, la fuente `semantic` de Inicio y su fila de ayuda, y el CSS que solo usaba la pantalla. Tests: fuera `TestSemanticEndpoints`, `TestSemanticReindex*`, `TestSymbolFiltersAreServerSide`, `TestCodeScreenLoadsLazily` y los helpers del CodeGraph falso; la prueba de la caché de Inicio usa ahora `specs` como fuente que falla (5 fuentes). `go vet` limpio y `go test ./internal/dashboard` en verde (el primer pase completo solo lo paró la guarda del repositorio por mis ediciones simultáneas en otros paquetes; se repite al cerrar T2).

- 2026-11-26: T2 hecha. Fuera los casos, la ayuda y las funciones `runKnowledge*`/`runSemantic*` de `cmd/axiom`, `internal/knowledge`, `internal/semantic`, los tests CLI de `knowledge` (incluidos los de baseline residual) y el helper `jsonOf`; test nuevo `TestCLIHelpOmitsRetiredKnowledgeAndSemantic`. Skill `knowledge-crawl` retirada y `knowledge-query` sobre CodeGraph directo; `AGENTS.md`, manual (sección «CodeGraph en Uso Directo»), `.deadcode-baseline.txt` y humo `semantic` de `ci.yml`. `go vet` y `go test ./cmd/axiom -run` focalizado en verde.

- 2026-11-26: T3 hecha. `workspace.ObsoleteKeyWarnings` avisa desde `Validate` de `governance.semantic_analysis` sin invalidar (lo ven `workspace validate`, `doctor`, el dashboard y la validación de `init`); el campo se sigue leyendo; `hub/init.go` y `axiom.example.yaml` ya no lo escriben. Test nuevo `TestValidateWarnsAboutObsoleteSemanticAnalysis`. `go vet` limpio y `go test` focalizado en verde (`workspace -run ObsoleteSemantic|ValidateTopology|ValidateSkillRoots|Docs|Example|Declarations|Template|Manual`, `hub -run Init|Knowledge`).

## Siguiente paso

T4 (Spec viva: REMOVED de las tres specs; MODIFIED de `knowledge-profile-init` REQ-23.2, `planning-home-resolution` REQ-24.10, `workspace-topology` REQ-1.2 y `axiom-binary-ci-coverage` REQ-20.15; notas de limpieza) y la repetición de `go test ./internal/dashboard` completo. Pendiente de revisar `docs/redesign/README.md` (propuesta con mockups; se dejó intacta).
