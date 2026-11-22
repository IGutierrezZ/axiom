# ODD: Quitar Serena (quitar-serena)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/quitar-serena.md`. Espejo Engram: `odd/quitar-serena/tasks`.
> Rama: `inc/quitar-serena` · worktree `C:\repos\axiom-wt\quitar-serena\axiom`. Cambio C-01-d del plan maestro (`docs/plan/backlog-2026-11/01-poda-friccion.md`), ola 4; decisión P9 del apartado 6.2 del README (ODD con `axiom archive integrate --odd`).

## Objetivo

Quitar Serena del producto para quedarse con Axiom, Engram y CodeGraph. Axiom no instala Serena: solo la detecta, la prioriza en `auto` y la escribe en el `.mcp.json` del perfil `knowledge`. Tras el cambio, el conector semántico es CodeGraph o el motor nativo `native-ast`.

## Problema

Serena aparece en el detector, el selector de conector, el estado de la CLI y del dashboard, el alta del perfil `knowledge`, dos skills, la configuración de ejemplo, la documentación y cuatro specs vivas, sin aportar nada que no cubran CodeGraph, el motor nativo y Engram.

## Por qué

Poda de fricción (sección 01 del plan). Decisión del usuario: «solo se mantienen Axiom, Engram y CodeGraph». No queda hueco funcional: la edición simbólica de Serena nunca se integró; `find_referencing_symbols` lo cubre CodeGraph (`codegraph_explore`); los símbolos de Go los cubre `axiom semantic symbols`; las memorias, Engram.

## Decisiones

- **Selector (`internal/semantic/service.go`).** Valores reconocidos de `governance.semantic_analysis`: `auto`, `codegraph`, `native-ast`. En `auto` gana CodeGraph si está disponible y, si no, `native-ast`.
- **Valor obsoleto `serena`.** Un `axiom.yaml` con `semantic_analysis: serena` sigue cargando: `GetStatus` añade a `Warnings` un aviso de valor obsoleto y se comporta como `auto`. `ConfiguredConnector` informa el valor tal como está en `axiom.yaml`. Los demás valores desconocidos conservan el comportamiento actual (actúan como `auto`, sin aviso), para no ampliar el alcance.
- **API y CLI.** `SemanticStatus` pierde `serena_available`, `serena_installed` y `serena_configured`; `axiom semantic status` deja de imprimir la línea de Serena; el detector deja de buscar `serena`/`serena-mcp` en el `PATH` y en las configuraciones MCP de los agentes.
- **`.mcp.json` del perfil `knowledge` (`internal/hub/mcp.go`).** `DefaultKnowledgeMCPServers` declara solo `engram` y `codegraph`. `InjectKnowledgeMCPServers` retira la entrada `serena` solo si es exactamente la que escribía Axiom (`command: "serena"`, `args: ["serve"]` y ninguna otra clave); cualquier otra entrada `serena` se conserva. Solo escribe el fichero si cambia algo.
- **Skills `knowledge-query` y `knowledge-crawl`.** Se reescriben sobre CodeGraph (`codegraph_explore`) y el motor nativo (`axiom semantic symbols` e `inspect`). No están embebidas en el binario (`internal/assets/skills/` no las contiene): no hay goldens que regenerar; el índice de `AGENTS.md:59-60` se actualiza a mano.
- **Spec viva.** Las deltas de requisitos van en `## Spec viva` y se integran con `axiom archive integrate --odd quitar-serena`. El título y el propósito de `openspec/specs/semantic-code/spec.md` no son requisitos y la delta no los puede expresar: se editan a mano en el mismo commit, y la fila y la cabecera de `openspec/INDEX.md` que los reflejan, si `integrate` no las regenera.
- **Fuera de alcance.** Instalar CodeGraph de serie (sigue siendo herramienta de comunidad opcional). Corregir `serve` frente a `serve --mcp` en la entrada `codegraph` de `mcp.go` (posible inconsistencia observada, no verificada; se anota como seguimiento).
- **Huecos del plan (resuelto en la parada de plan, 2026-11-18).** El plan no lista `docs/redesign/README.md:158` (documenta los campos `serena_*` de `/api/semantic/status`) ni las maquetas `docs/redesign/mockups/{Codigo,Main,Ecosistema}.dc.html` (5 menciones), ni los documentos ODD cerrados `odd/tasks/{codegraph-indice,inc-24-seguimientos-w1-s4,ui-dashboard-ux-governance-and-sync-fixes}.md`. Aprobado: se actualizan `docs/redesign/README.md` y las maquetas (son diseño vigente) y se tratan los `odd/tasks/` cerrados como historial, igual que `openspec/changes/archive`. Este documento también menciona Serena y es historial del cambio.

## Alcance autorizado

- `internal/semantic/{types,detector,service,semantic_test}.go`.
- `internal/hub/mcp.go`, `internal/hub/init_knowledge_test.go` (y un test nuevo de la retirada).
- `internal/dashboard/assets/js/screens/semantic.js`, `internal/dashboard/knowledge_assets_test.go`.
- `cmd/axiom/main.go` (`:149` ayuda, `:1604` línea de estado).
- `internal/knowledge/types.go:59` (comentario).
- `skills/knowledge-query/SKILL.md`, `skills/knowledge-crawl/SKILL.md`, `AGENTS.md:59-60`.
- `axiom.example.yaml:160`, `docs/manual-de-inicio.md` (incluido `semantic_analysis: "enabled"` → `"auto"`), `docs/architecture/AXIOM_ENTERPRISE_VISION.md`, `docs/redesign/README.md` y `docs/redesign/mockups/*.dc.html`.
- `openspec/specs/{semantic-code,knowledge-profile-init,knowledge-query-engine,knowledge-sweep-command}/spec.md` e `openspec/INDEX.md`, vía `integrate` salvo título y propósito.
- No se tocan: `internal/reviewtransaction/*` (coincidencias falsas de `compactReclaimPhaseRenamed`), `docs/archive`, `openspec/changes/archive`, `docs/audits`, `docs/plan`.

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y no hay elección del usuario. Comprobaciones funcionales con `go test` por paquetes afectados, sin binarios sueltos de `axiom` (el EDR los borra).

## Entrega

Estrategia `exception-ok`: una sola PR con `type:refactor` y `size:exception` si pasa de 400 líneas (previsión: unas 350-500 líneas). Push, PR y fusión los decide el usuario.

## Tareas

- [x] **T1 · Código Go y dashboard sin Serena** (ruta: delegada, un escritor; disparador: 8+ ficheros no triviales en 5 paquetes)
  - `semantic`: fuera `ConnectorSerena`, campos `Serena*`, `CheckSerena*` y detección en JSON; selector `auto` = CodeGraph > `native-ast`; aviso de valor obsoleto para `serena`; mensaje de reindexado sin Serena.
  - `hub/mcp.go`: solo `engram` y `codegraph`; retirada de la entrada exacta `serena` gestionada.
  - `main.go` (`:149`, `:1604`), `knowledge/types.go:59`, `semantic.js` (`:14`, `:17`, `:31`, `:75`).
  - Tests: `semantic_test.go` (auto con y sin CodeGraph, `serena` obsoleto con aviso), `init_knowledge_test.go` (solo `engram` y `codegraph`; retirada de la `serena` exacta; conservación de una `serena` modificada y de `custom-db`), `knowledge_assets_test.go`.
  - Comprobación: `go test ./internal/semantic/... ./internal/hub/... ./internal/dashboard/... ./internal/knowledge/...` y `go vet ./cmd/axiom/`.
- [x] **T2 · Skills, configuración y documentación** (ruta: delegada, un escritor; disparador: 6+ ficheros de prosa)
  - Reescritura de `knowledge-query` y `knowledge-crawl` sobre `codegraph_explore` y `axiom semantic symbols|inspect`; `AGENTS.md:59-60`.
  - `axiom.example.yaml:160` → `"auto"`; `manual-de-inicio.md` (`:115`, `:307`, `:703`); `AXIOM_ENTERPRISE_VISION.md`; `docs/redesign/README.md:158` y maquetas.
  - Comprobación: relectura del diff.
- [x] **T3 · Spec viva** (ruta: en línea)
  - Título y propósito de `semantic-code` a mano; `axiom archive integrate --odd quitar-serena` con el `axiom` instalado; segunda ejecución `already_integrated`.
  - Comprobación: `rg -in serena openspec/specs openspec/INDEX.md` sin resultados.
- [x] **T4 · Verificación final** (ruta: en línea)
  - Batería del criterio de terminado.

## Criterio de terminado

- `rg -i serena` sin resultados fuera de los históricos archivados (`docs/archive`, `openspec/changes/archive`, `docs/audits`, `docs/plan`, `odd/tasks` cerrados y este documento) y de las coincidencias falsas de `internal/reviewtransaction`. Excepción obligada por los dos criterios siguientes: el valor obsoleto `serena` y la retirada de la entrada gestionada (`internal/semantic/service.go`, `internal/hub/mcp.go` y sus tests).
- `axiom semantic status` no informa de Serena (test de `GetStatus` y relectura de `main.go`).
- En `auto` gana CodeGraph si está disponible y, si no, `native-ast`.
- Un `axiom.yaml` con `semantic_analysis: serena` sigue cargando, con un aviso de valor obsoleto y caída a CodeGraph o `native-ast`.
- El alta del perfil `knowledge` escribe solo `engram` y `codegraph` en `.mcp.json`, y retira una entrada `serena` gestionada que ya existiera.
- `go test ./internal/semantic/... ./internal/hub/... ./internal/dashboard/...` en verde.

## Spec viva

### Capacidad: semantic-code

```openspec
## REMOVED Requirements

### Requirement: Detección y Diagnóstico de Serena MCP (REQ-1.1)

(Reason: C-01-d retira Serena del producto; solo se mantienen Axiom, Engram y CodeGraph. El detector deja de buscarla en el `PATH` y en las configuraciones MCP de los agentes.)

## MODIFIED Requirements

### Requirement: Resolución del Conector Semántico Activo (REQ-1.3)

El servicio semántico DEBE resolver el conector activo basándose en la configuración `governance.semantic_analysis` de `axiom.yaml` (`codegraph`, `native-ast` o `auto`):
1. Si se configura `codegraph` y está disponible, se selecciona CodeGraph; si no está disponible, el estado DEBE advertirlo y activar el **motor nativo Go AST** (`native-ast`).
2. Si se configura `native-ast`, se selecciona el motor nativo.
3. Si se configura `auto` o no se especifica, se selecciona CodeGraph si está disponible y, si no, `native-ast`.
4. El valor obsoleto `serena` DEBE seguir cargando: el estado DEBE incluir un aviso de valor obsoleto y el servicio DEBE resolver el conector como en `auto`.
(Previously: se reconocía `serena`, que tenía prioridad sobre CodeGraph en `auto`.)

#### Scenario: Resolución automática con CodeGraph disponible
- **DADO** un archivo `axiom.yaml` con `semantic_analysis: "auto"` (o no especificado) y CodeGraph disponible
- **CUANDO** se inicializa el servicio semántico
- **ENTONCES** se selecciona el conector `codegraph`

#### Scenario: Resolución automática con fallback a AST nativo
- **DADO** un archivo `axiom.yaml` con `semantic_analysis: "auto"` (o no especificado) y sin CodeGraph disponible
- **CUANDO** se inicializa el servicio semántico
- **ENTONCES** se selecciona el conector `native-ast`
- **Y** se asegura la operatividad semántica del 100% de consultas sobre repositorios Go

#### Scenario: Conector explícito no disponible advierte y activa fallback
- **DADO** `axiom.yaml` con `semantic_analysis: "codegraph"` pero sin CodeGraph instalado
- **CUANDO** se consulta el estado semántico
- **ENTONCES** el estado reporta una advertencia indicando que CodeGraph no está disponible
- **Y** conmuta al motor nativo `native-ast` para evitar el bloqueo del agente

#### Scenario: Valor obsoleto `serena`
- **DADO** `axiom.yaml` con `semantic_analysis: "serena"`
- **CUANDO** se consulta el estado semántico
- **ENTONCES** la configuración carga sin error
- **Y** el estado reporta un aviso de valor obsoleto
- **Y** el conector activo es `codegraph` si está disponible y, si no, `native-ast`

### Requirement: Comando `axiom semantic status` (REQ-3.1)

El comando `axiom semantic status [--path <dir>]` DEBE mostrar:
1. El conector semántico configurado en `axiom.yaml`.
2. El conector activo en ejecución (`codegraph` o `native-ast`).
3. La disponibilidad de CodeGraph en el sistema.
4. El total de paquetes Go y símbolos indexados en el workspace.
(Previously: también mostraba la disponibilidad de Serena MCP y podía informar `serena` como conector activo.)

#### Scenario: Ejecución de `axiom semantic status`
- **DADO** un workspace válido de Axiom
- **CUANDO** el usuario ejecuta `axiom semantic status`
- **ENTONCES** la CLI imprime un informe detallado con código de salida 0
- **Y** muestra el estado del conector activo y los agentes verificados

### Requirement: Interfaz Web SPA con Panel Semántico y Buscador (REQ-4.2)

La interfaz SPA embebida (`assets/index.html`, `app.js`, `style.css`) DEBE incluir una nueva sección de navegación "Semántica & Grafo" que permita:
1. Visualizar la tarjeta de salud del conector semántico con badges distintivos (`CodeGraph`, `AST Nativo Go`).
2. Disponer de un buscador en tiempo real para filtrar símbolos por texto y tipo.
3. Explorar la tabla de símbolos con enlaces relativos al filesystem y signaturas de funciones/métodos.
4. Visualizar las dependencias internas de los módulos.
(Previously: la tarjeta de salud incluía también el badge de Serena MCP.)

#### Scenario: Exploración de símbolos desde el navegador web
- **DADO** el usuario con el Dashboard abierto en la pestaña "Semántica & Grafo"
- **CUANDO** escribe un término en el campo de búsqueda de símbolos
- **ENTONCES** la tabla de resultados se filtra dinámicamente mostrando los símbolos coincidentes
- **Y** se indican su tipo, paquete y línea sin requerir recargar la página
```

### Capacidad: knowledge-profile-init

```openspec
## MODIFIED Requirements

### Requirement: Configuración Automática de Servidores MCP en .mcp.json (REQ-23.4)

El inicializador DEBE generar o actualizar el archivo `.mcp.json` en la raíz del workspace, registrando de forma predeterminada los servidores MCP:
- `engram`: memoria persistente de proyecto y sesiones.
- `codegraph`: grafo de llamadas y dependencias.
Si `.mcp.json` ya contenía otros servidores, el inicializador DEBE fusionar las entradas sin borrar las preexistentes, salvo la entrada `serena` que escribían versiones anteriores de Axiom: DEBE retirarla solo cuando coincida exactamente con la gestionada (`command: "serena"`, `args: ["serve"]`, sin otras claves).
(Previously: también registraba `serena` y nunca retiraba entradas.)

#### Scenario: Inyección de servidores MCP en .mcp.json nuevo
- **DADO** un workspace sin archivo `.mcp.json` previo
- **CUANDO** se ejecuta `axiom init --profile=knowledge`
- **ENTONCES** se crea `.mcp.json` conteniendo solo las definiciones para `engram` y `codegraph`

#### Scenario: Fusión no destructiva con .mcp.json existente
- **DADO** un workspace con un archivo `.mcp.json` que ya declara un servidor `custom-tool`
- **CUANDO** se ejecuta `axiom init --profile=knowledge`
- **ENTONCES** `.mcp.json` preserva `custom-tool` y añade `engram` y `codegraph`

#### Scenario: Retirada de la entrada serena gestionada
- **DADO** un `.mcp.json` con la entrada `serena` que escribía Axiom (`command: "serena"`, `args: ["serve"]`)
- **CUANDO** se ejecuta `axiom init --profile=knowledge`
- **ENTONCES** `.mcp.json` ya no contiene `serena`
- **Y** una entrada `serena` con otro comando, argumentos o claves se conserva intacta
```

### Capacidad: knowledge-query-engine

```openspec
## MODIFIED Requirements

### Requirement: Resolución Spec-First con Cortocircuito Inmediato (REQ-23.13)

El motor de consulta DEBE evaluar primero si la pregunta se encuentra resuelta con suficiente detalle y precisión dentro de las especificaciones vivas existentes (`openspec/specs/`). Si la respuesta está contenida en la spec, el motor DEBE responder directamente a partir de ella sin necesidad de inspeccionar el código ni invocar conectores semánticos.
(Previously: el escenario citaba Serena entre los conectores no invocados.)

#### Scenario: Respuesta directa desde la spec viva sin tocar código
- **DADO** una especificación viva `openspec/specs/billing/spec.md` que detalla las reglas de reembolso
- **CUANDO** se consulta `axiom knowledge query "¿Cuáles son las condiciones de reembolso?"`
- **ENTONCES** el motor responde utilizando la especificación viva
- **Y** no realiza llamadas a CodeGraph ni al motor semántico nativo para inspeccionar código

### Requirement: Triangulación Profunda con Evidencias ante Lagunas en la Spec (REQ-23.14)

Si la consulta no está cubierta en la spec viva o requiere detalle de implementación no documentado, el motor DEBE realizar una investigación profunda triangulando:
1. Símbolos y AST mediante el motor semántico nativo (`axiom semantic symbols` e `inspect`).
2. Símbolos, jerarquía de llamadas y dependencias mediante CodeGraph.
3. Decisiones y descubrimientos previos en la memoria Engram.
4. Código fuente correspondiente en los repositorios locales.
(Previously: los símbolos y el AST se obtenían mediante Serena.)

#### Scenario: Consulta que requiere inspección profunda de código
- **DADO** una consulta cuya respuesta no figura en `openspec/specs/`
- **CUANDO** el motor detecta la ausencia en la spec
- **ENTONCES** consulta a CodeGraph o al motor semántico nativo para localizar las firmas relevantes y a CodeGraph para rastrear los invocadores
- **Y** examina las líneas de código afectadas
```

### Capacidad: knowledge-sweep-command

```openspec
## MODIFIED Requirements

### Requirement: Detección Técnica Multidimensional (REQ-23.8)

El barrido DEBE inspeccionar el árbol de código y las definiciones AST/símbolos disponibles (vía CodeGraph o detector nativo) para identificar:
- Lenguajes de programación principales y secundarios.
- Frameworks y dependencias clave.
- Puntos de entrada (`main`, controladores, handlers, configuraciones).
- Estructura y capas de la arquitectura (monolito, microservicios, hexagonal, MVC, etc.).
(Previously: las definiciones AST/símbolos se obtenían vía Serena o detector nativo.)

#### Scenario: Identificación de stack y puntos de entrada
- **DADO** un repositorio con un proyecto Go y frontend React
- **CUANDO** se ejecuta el barrido `axiom knowledge sweep`
- **ENTONCES** el reporte técnico incluye `go` y `typescript/react` en lenguajes/frameworks
- **Y** señala los puntos de entrada principales detectados en el sistema
```

## Progreso

- Exploración (delegada a un mapeador: 10+ ficheros) y worktree `axiom-wt/quitar-serena` creados el 2026-11-18.
- Plan stop: approved 2026-11-18; implementation continues
- **T1** `ea871c11` (delegada, un escritor). `go build ./...`, `go vet` de los paquetes tocados y `gofmt -l`: limpios. `go test ./internal/semantic/... ./internal/hub/... ./internal/dashboard/... ./internal/knowledge/...`: ok (repetido por el padre en `semantic` y `hub`). `go test ./cmd/axiom/ -run 'Semantic|Help'`: ok. Aviso: «El valor «serena» de governance.semantic_analysis en axiom.yaml está obsoleto (Serena ya no está soportada) y se trata como «auto».» Coincidencias de `serena` que quedan a propósito en código: la constante `deprecatedConnectorSerena` y su aviso (`internal/semantic/service.go`), `isManagedSerenaEntry` y la retirada en `internal/hub/mcp.go`, y los tests que los cubren; el propio criterio de terminado exige ese valor obsoleto y esa retirada, así que no pueden desaparecer.
- **T2** (delegada, un escritor; commit siguiente a `ea871c11`, ver `git log`). Skills reescritas sobre `codegraph_explore` y `axiom semantic symbols --query --kind` / `inspect` (banderas comprobadas en `cmd/axiom/main.go:1631-1632`); `AGENTS.md:59-60`, `axiom.example.yaml` (`"auto"`), `manual-de-inicio.md` (`:115`, `:307` de `"enabled"` a `"auto"`, `:703`), `AXIOM_ENTERPRISE_VISION.md`, `docs/redesign/README.md:158` y las tres maquetas (conteos de avisos y comprobaciones ajustados). `rg -in serena` sobre esos ficheros: sin resultados. `go test ./internal/app/ -run SkillIndex` y `go test ./cmd/axiom/ -run SkillIndex`: ok. Maquetas no abiertas en navegador (bloques completos retirados).
- **T3** (en línea). Título y propósito de `semantic-code` editados a mano. Primera ejecución de `axiom archive integrate --odd quitar-serena` (el `axiom` instalado, `v3.7.1-0.20261118144407-c18dd7868805`) rechazada sin escribir: el REMOVED exige `(Reason: ...)` en lugar de `(Motivo: ...)`; corregido en la delta. Segunda: `composed` en las cuatro capacidades y `openspec/INDEX.md` regenerado (incluida la fila del título); tercera: `already_integrated`. `rg -in serena openspec/specs openspec/INDEX.md`: solo quedan el valor obsoleto `serena`, la retirada de la entrada gestionada (exigidos por el criterio) y las notas `(Previously: ...)` del formato de delta.
- **T4** (en línea). `rg -il serena` fuera de los históricos: solo `internal/reviewtransaction/*` (coincidencias falsas), el valor obsoleto y la retirada (`internal/semantic/service.go`, `internal/hub/mcp.go`), sus tests (`semantic_test.go`, `init_knowledge_test.go`, y `knowledge_assets_test.go`, que comprueba que el JS no contiene `serena`) y las specs vivas que describen ese contrato. `go test ./internal/semantic/... ./internal/hub/... ./internal/dashboard/...`: ok. `axiom semantic status` sin línea de Serena (`main.go`, relectura) y `SemanticStatus` sin claves `serena` (test). Tamaño frente a `origin/main`: 26 ficheros, +582/−183 (incluye este documento): la PR lleva `size:exception`.

## Siguiente paso

Abrir la PR cuando el usuario lo autorice: `powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/axiom-worktree.ps1 pr quitar-serena -Lane odd`, con `type:refactor` y `size:exception`. Seguimiento fuera de alcance: la entrada `codegraph` de `internal/hub/mcp.go` usa `serve` y `pi_codegraph.go` usa `serve --mcp` (no verificado).
