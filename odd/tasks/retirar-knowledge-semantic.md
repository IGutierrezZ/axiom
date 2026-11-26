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
- [x] T4 · `## Spec viva` integrada con `axiom archive integrate --odd retirar-knowledge-semantic`, borrado de las tres specs retiradas e `INDEX.md`, notas de limpieza manual y criterio `rg`.

Ruta: todas en línea (borrado guiado por el compilador; ninguna unidad necesita un mapeo que exceda el presupuesto de evidencia).

## Criterio de terminado

- `rg -l 'internal/(knowledge|semantic)"' -g '*.go'` sin resultados.
- `axiom knowledge` y `axiom semantic` responden como un comando desconocido.
- Un `axiom.yaml` con `semantic_analysis` carga con un aviso.
- `communitytool` sin referencias a `semantic`.
- `go vet` y `go test` de los paquetes tocados en verde; la batería completa, en la CI de la PR.

## Spec viva

Retira las tres capacidades de `axiom knowledge` y `axiom semantic` (comandos, motor AST nativo y panel semántico del dashboard) y adapta los requisitos que las citaban. `knowledge-query-engine`, `knowledge-sweep-command` y `semantic-code` se quedan sin requisitos y se borran tras integrar, con sus entradas de `openspec/INDEX.md`. RF-TOP-02 de `workspace-topology` recoge el aviso de la clave obsoleta `governance.semantic_analysis`.

### Capacidad: knowledge-query-engine

```openspec
## REMOVED Requirements

### Requirement: Subcomando de Consulta Técnica/Funcional (REQ-23.12)

(Reason: C-08-l retira `axiom knowledge query`; los agentes consultan la spec viva y el código con CodeGraph directamente.)

### Requirement: Resolución Spec-First con Cortocircuito Inmediato (REQ-23.13)

(Reason: C-08-l retira `axiom knowledge query`; los agentes consultan la spec viva y el código con CodeGraph directamente.)

### Requirement: Triangulación Profunda con Evidencias ante Lagunas en la Spec (REQ-23.14)

(Reason: C-08-l retira `axiom knowledge query`; los agentes consultan la spec viva y el código con CodeGraph directamente.)

### Requirement: Contrato de Respuesta en Dos Niveles (REQ-23.15)

(Reason: C-08-l retira `axiom knowledge query`; los agentes consultan la spec viva y el código con CodeGraph directamente.)

### Requirement: Auto-Enriquecimiento Obligatorio de la Spec Viva (REQ-23.16)

(Reason: C-08-l retira `axiom knowledge query`; los agentes consultan la spec viva y el código con CodeGraph directamente.)
```

### Capacidad: knowledge-sweep-command

```openspec
## REMOVED Requirements

### Requirement: Invocación Autónoma Headless (REQ-23.7)

(Reason: C-08-l retira `axiom knowledge sweep` y `internal/knowledge`.)

### Requirement: Detección Técnica Multidimensional (REQ-23.8)

(Reason: C-08-l retira `axiom knowledge sweep` y `internal/knowledge`.)

### Requirement: Inferencia Funcional de Módulos y Dominios (REQ-23.9)

(Reason: C-08-l retira `axiom knowledge sweep` y `internal/knowledge`.)

### Requirement: Política No Bloqueante y Sobre Diferido de Ambigüedades (REQ-23.10)

(Reason: C-08-l retira `axiom knowledge sweep` y `internal/knowledge`.)

### Requirement: Siembra Inicial en OpenSpec y Persistencia en Engram (REQ-23.11)

(Reason: C-08-l retira `axiom knowledge sweep` y `internal/knowledge`.)
```

### Capacidad: semantic-code

```openspec
## REMOVED Requirements

### Requirement: Detección de CodeGraph CLI y Configuración (REQ-1.2)

(Reason: C-08-l retira `axiom semantic`, `internal/semantic` y el panel semántico del dashboard (P23); CodeGraph queda en uso directo por su MCP y su CLI.)

### Requirement: Resolución del Conector Semántico Activo (REQ-1.3)

(Reason: C-08-l retira `axiom semantic`, `internal/semantic` y el panel semántico del dashboard (P23); CodeGraph queda en uso directo por su MCP y su CLI.)

### Requirement: Extracción de Símbolos de Código en Go (REQ-2.1)

(Reason: C-08-l retira `axiom semantic`, `internal/semantic` y el panel semántico del dashboard (P23); CodeGraph queda en uso directo por su MCP y su CLI.)

### Requirement: Análisis del Grafo de Dependencias de Paquetes (REQ-2.2)

(Reason: C-08-l retira `axiom semantic`, `internal/semantic` y el panel semántico del dashboard (P23); CodeGraph queda en uso directo por su MCP y su CLI.)

### Requirement: Búsqueda y Filtrado de Símbolos (REQ-2.3)

(Reason: C-08-l retira `axiom semantic`, `internal/semantic` y el panel semántico del dashboard (P23); CodeGraph queda en uso directo por su MCP y su CLI.)

### Requirement: Comando `axiom semantic status` (REQ-3.1)

(Reason: C-08-l retira `axiom semantic`, `internal/semantic` y el panel semántico del dashboard (P23); CodeGraph queda en uso directo por su MCP y su CLI.)

### Requirement: Comando `axiom semantic symbols` (REQ-3.2)

(Reason: C-08-l retira `axiom semantic`, `internal/semantic` y el panel semántico del dashboard (P23); CodeGraph queda en uso directo por su MCP y su CLI.)

### Requirement: Comando `axiom semantic inspect` (REQ-3.3)

(Reason: C-08-l retira `axiom semantic`, `internal/semantic` y el panel semántico del dashboard (P23); CodeGraph queda en uso directo por su MCP y su CLI.)

### Requirement: Endpoints REST Semánticos en el Servidor Web (REQ-4.1)

(Reason: C-08-l retira `axiom semantic`, `internal/semantic` y el panel semántico del dashboard (P23); CodeGraph queda en uso directo por su MCP y su CLI.)

### Requirement: Interfaz Web SPA con Panel Semántico y Buscador (REQ-4.2)

(Reason: C-08-l retira `axiom semantic`, `internal/semantic` y el panel semántico del dashboard (P23); CodeGraph queda en uso directo por su MCP y su CLI.)
```

### Capacidad: workspace-topology

```openspec
## MODIFIED Requirements

### Requirement: Subcomando axiom workspace validate (RF-TOP-02)

El binario `axiom` DEBE proporcionar el subcomando `axiom workspace validate`, el cual evalúa el archivo `axiom.yaml` del directorio actual (o de la ruta especificada mediante `--path`) y verifica la conformidad de la topología y de los repositorios asociados a los roles. Además DEBE validar los bloques opcionales `technical_context`, `git` y `repositories` de `axiom.yaml`: la sintaxis y la contención de rutas se reportan como errores y las rutas de contexto técnico inexistentes como advertencias que no alteran el resultado. También DEBE reportar como error que un contexto técnico mantenido se declare con un almacén de artefactos sin ficheros (`engram` o `none`, RF-CTX-17), DEBE rechazar un bloque `baseline:` residual con el error tipado de RF-TOP-12 y DEBE rechazar la topología retirada `monorepo-decoupled` con el mensaje accionable de RF-TOP-08. Una clave `governance.semantic_analysis` residual NO DEBE invalidar la configuración: se sigue leyendo, se ignora y DEBE reportarse como advertencia que nombra la clave y no altera el resultado.
(Previously: `governance.semantic_analysis` elegía el conector de `axiom semantic` y se validaba en silencio; C-08-l retira ese comando y la clave pasa a advertencia.)

#### Scenario: Validación de espacio de trabajo conforme
- **DADO** un directorio con un archivo `axiom.yaml` válido y las carpetas de repositorios existentes
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida reporta un estado `COMPLIANT` indicando la topología detectada y los roles verificados
- **Y** el código de salida del proceso es `0`

#### Scenario: Validación con ruta explícita mediante --path
- **DADO** un espacio de trabajo ubicado en una ruta externa `/tmp/test-workspace`
- **CUANDO** el usuario ejecuta `axiom workspace validate --path /tmp/test-workspace`
- **ENTONCES** la validación se ejecuta sobre el directorio especificado
- **Y** reporta el resultado correspondiente a esa ubicación

#### Scenario: Espacio de trabajo no conforme finaliza con error
- **DADO** un espacio de trabajo donde faltan repositorios obligatorios o el archivo de configuración es inválido
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida detalla de forma explícita en español los motivos de no conformidad
- **Y** el proceso finaliza con código de salida `1`

#### Scenario: Bloque `baseline:` residual finaliza con error
- **DADO** un `axiom.yaml` que declara `baseline:`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida detalla en español que el bloque se retiró y la continuación `axiom archive adopt`
- **Y** el proceso finaliza con código de salida `1`

#### Scenario: Ruta de contexto técnico inexistente solo avisa
- **DADO** un `axiom.yaml` conforme cuyo `technical_context.maintained.paths` incluye una ruta que no existe
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida incluye una advertencia que nombra la ruta y reporta `COMPLIANT`
- **Y** el código de salida del proceso es `0`

#### Scenario: Declaración incompatible con el almacén de artefactos finaliza con error
- **DADO** un `axiom.yaml` con `technical_context.maintained` válido y un `openspec/config.yaml` con `artifact_store: engram`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida detalla en español que el contexto mantenido exige un almacén de ficheros, y la continuación
- **Y** el proceso finaliza con código de salida `1`

#### Scenario: Bloque `git:` o `repositories:` inválido finaliza con error
- **DADO** un `axiom.yaml` con `git.branch_prefix: "a b//"` o con una entrada de `repositories:` cuya ruta no existe
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida detalla en español la clave o la ruta y la regla incumplida
- **Y** el proceso finaliza con código de salida `1`

#### Scenario: Topología retirada finaliza con error accionable
- **DADO** un `axiom.yaml` con `workspace.topology: monorepo-decoupled`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida indica en español que la topología fue retirada y que debe sustituirse por `multirepo`
- **Y** el proceso finaliza con código de salida `1`


#### Scenario: Clave `governance.semantic_analysis` residual solo avisa
- **DADO** un `axiom.yaml` conforme que declara `governance.semantic_analysis`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida incluye una advertencia que nombra `governance.semantic_analysis` y reporta `COMPLIANT`
- **Y** el código de salida del proceso es `0`
```

### Capacidad: knowledge-profile-init

```openspec
## MODIFIED Requirements

### Requirement: Configuración Canónica de axiom.yaml en Topología Multirepo (RF-KNW-02)

Bajo el perfil `knowledge`, el archivo `axiom.yaml` DEBE generarse con:
- `workspace.topology: "multirepo"`.
- `workspace.specs_repository: "openspec"` (como repositorio separado e independiente desde cero).
- `roles.knowledge` con política `advisory`, sin compuertas bloqueantes de código.
- `governance.language: "es"` y `governance.shared_memory: "engram"`.

(Previously: también escribía `governance.semantic_analysis: "auto"`, clave que C-08-l deja sin efecto.)

#### Scenario: Manifiesto axiom.yaml generado bajo perfil knowledge
- **DADO** un comando de inicialización con `--profile=knowledge`
- **CUANDO** se escribe `axiom.yaml`
- **ENTONCES** la sección `workspace` declara `topology: "multirepo"` y `specs_repository: "openspec"`
- **Y** la sección `governance` declara `language: "es"` y `shared_memory: "engram"`
- **Y** los roles definidos tienen política `gate_policy: "advisory"`
```

### Capacidad: planning-home-resolution

```openspec
## MODIFIED Requirements

### Requirement: Los Consumidores de Catálogo Comparten la Convención de Raíz (RF-PLH-10)

El servicio de documentación viva (livingdoc, incluido el sintetizador de arranque en frío) y el dashboard web DEBEN obtener la raíz de especificaciones y el planning home del primitivo de RF-PLH-01 y NO DEBEN mantener una convención propia. Las especificaciones vivas, los cambios y el índice residen en `<planning home>/specs`, `<planning home>/changes` y `<planning home>/INDEX.md`. Cuando en un proyecto `monorepo-embedded` con `specs_repository` distinto de `""`, `.` y `openspec` existe `<raíz>/<specs_repository>/specs` y difiere de `<planning home>/specs`, los consumidores de catálogo DEBEN leer solo `<planning home>/specs` y emitir un aviso que nombre ambas rutas.
(Previously: los consumidores de catálogo incluían `knowledge` (raíz de especificaciones por defecto de `axiom knowledge sweep`), que C-08-l retira, y antes la TUI.)

#### Scenario: Catálogo del workspace canónico
- **DADO** el workspace canónico con `<maestra>/repo-specs/openspec/specs/workspace-topology/spec.md`
- **CUANDO** se ejecuta `axiom archive list` desde `<maestra>`
- **ENTONCES** el listado incluye el dominio `workspace-topology`

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

### Capacidad: axiom-binary-ci-coverage

```openspec
## MODIFIED Requirements

### Requirement: Construcción y ejercicio bloqueante de la superficie exclusiva de axiom (RF-CIB-01)

El flujo de trabajo de CI DEBE construir `cmd/axiom` y ejercitar de forma bloqueante su superficie exclusiva: `init`, `change`, `project`, `workspace`, `handoff`, `role`, `skill`, `archive`, y el arranque/ayuda de `ui`.
(Previously: la superficie incluía `semantic`, que C-08-l retira.)

#### Scenario: Humo bloqueante sobre la superficie exclusiva
- **DADO** el binario `cmd/axiom` recién construido en CI
- **CUANDO** el paso de cobertura se ejecuta
- **ENTONCES** ejercita los comandos de la superficie exclusiva de Axiom con salida esperada y stderr limpio de avisos de deprecación
- **Y** un fallo en cualquiera de ellos bloquea la fusión
```

## Progreso

- 2026-11-26: documento creado; worktree desde `origin/main` en `7464c5a7`.
- 2026-11-26: T1 hecha. Fuera las rutas `/api/semantic/*`, los métodos semánticos de `Service`, `semantic.js`, `core/lazy.js` (solo lo usaba Código), la sección `view-codigo` y su entrada de navegación, la ruta `codigo`, la fuente `semantic` de Inicio y su fila de ayuda, y el CSS que solo usaba la pantalla. Tests: fuera `TestSemanticEndpoints`, `TestSemanticReindex*`, `TestSymbolFiltersAreServerSide`, `TestCodeScreenLoadsLazily` y los helpers del CodeGraph falso; la prueba de la caché de Inicio usa ahora `specs` como fuente que falla (5 fuentes). `go vet` limpio y `go test ./internal/dashboard` en verde (el primer pase completo solo lo paró la guarda del repositorio por mis ediciones simultáneas en otros paquetes; se repite al cerrar T2).

- 2026-11-26: T2 hecha. Fuera los casos, la ayuda y las funciones `runKnowledge*`/`runSemantic*` de `cmd/axiom`, `internal/knowledge`, `internal/semantic`, los tests CLI de `knowledge` (incluidos los de baseline residual) y el helper `jsonOf`; test nuevo `TestCLIHelpOmitsRetiredKnowledgeAndSemantic`. Skill `knowledge-crawl` retirada y `knowledge-query` sobre CodeGraph directo; `AGENTS.md`, manual (sección «CodeGraph en Uso Directo»), `.deadcode-baseline.txt` y humo `semantic` de `ci.yml`. `go vet` y `go test ./cmd/axiom -run` focalizado en verde.

- 2026-11-26: T3 hecha. `workspace.ObsoleteKeyWarnings` avisa desde `Validate` de `governance.semantic_analysis` sin invalidar (lo ven `workspace validate`, `doctor`, el dashboard y la validación de `init`); el campo se sigue leyendo; `hub/init.go` y `axiom.example.yaml` ya no lo escriben. Test nuevo `TestValidateWarnsAboutObsoleteSemanticAnalysis`. `go vet` limpio y `go test` focalizado en verde (`workspace -run ObsoleteSemantic|ValidateTopology|ValidateSkillRoots|Docs|Example|Declarations|Template|Manual`, `hub -run Init|Knowledge`).

- 2026-11-26: T4 hecha. `axiom archive integrate --odd retirar-knowledge-semantic` compuso 7 capacidades; se borran `knowledge-query-engine`, `knowledge-sweep-command` y `semantic-code` (vacías) y `axiom archive sync` regenera `openspec/INDEX.md` (48 dominios, 514 → 494 requisitos, 2212 → 2188 escenarios). MODIFIED: RF-TOP-02 (aviso de la clave), RF-KNW-02, RF-PLH-10 y RF-CIB-01 (coordinación: va en esta Spec viva). Notas de limpieza manual en `docs/releases/notas-retirada-knowledge-semantic.md`. `docs/redesign/README.md`: solo las citas operativas a `/api/semantic/*` y la navegación (coordinación). Criterio `rg -l 'internal/(knowledge|semantic)"' -g '*.go'` sin resultados. `go test` en verde de `livespec`, `livingdoc` y `dashboard` completo.

## Siguiente paso

PR con `scripts/axiom-worktree.ps1 pr retirar-knowledge-semantic -Lane odd -Label breaking-change`, CI, merge commit y `done`.
