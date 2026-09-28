# Especificación de Requerimientos: Perfil de Inicialización Knowledge, Barrido Autónomo Multirrepo y Consulta Spec-First con Auto-Enriquecimiento (INC-23)

> **Incremento:** `inc-23-knowledge-profile`  
> **Fase del Roadmap:** Fase 4 — Gobernanza de Agentes, Integración de Conocimiento y Descubrimiento Semántico  
> **Responsabilidad:** Hub & Init / Dominio Knowledge / CLI Commands / OpenSpec Living Documentation / MCP Integration  
> **Estado:** En desarrollo (fase de especificación)  
> **Idioma:** Español (castellano peninsular)  

---

## Propósito

Definir de forma rigurosa, ejecutable y verificable los requerimientos funcionales y los escenarios BDD para:
1. El perfil de inicialización ligero en Axiom (`axiom init --profile=knowledge` y su alias `--spec-only`), configurado en topología `multirepo` con repositorio canónico OpenSpec independiente y desde cero, memoria compartida Engram y conectores semánticos (`serena`, `codegraph`), excluyendo suites de compilación, ejecución (`sdd-apply`) o runners de pruebas.
2. El comando de barrido rápido técnico y funcional (`axiom knowledge sweep`), diseñado para ser invocado de forma autónoma/headless por orquestadores backend sobre CLIs de agentes (Claude Code, Codex, Copilot, Gemini CLI), con entrega diferida y no bloqueante de dudas y ambigüedades al finalizar.
3. El motor de consultas fundamentadas (`axiom knowledge query`), dotado de estrategia *Spec-First*, triangulación profunda de evidencias sobre código/AST/grafo cuando la spec no cubre la pregunta, contrato de respuesta en dos niveles (respuesta directa y bloque de evidencias), y auto-enriquecimiento obligatorio e idempotente de la especificación viva (`openspec/specs/`), el catálogo `openspec/INDEX.md` y la memoria Engram.

---

## Alcance de esta Especificación

| # | Capacidad | Tipo | Requerimientos |
|---|---|---|---|
| 1 | `knowledge-profile-init` | Nueva | REQ-23.1 – REQ-23.6 |
| 2 | `knowledge-sweep-command` | Nueva | REQ-23.7 – REQ-23.11 |
| 3 | `knowledge-query-engine` | Nueva | REQ-23.12 – REQ-23.16 |

---

## 1. Capacidad: `knowledge-profile-init`

Permite inicializar un espacio de trabajo ligero de Axiom enfocado en documentación viva, indexación semántica y catalogación, prescindiendo del arnés de ejecución de código.

### Requirement: Flag de Perfil Knowledge y Alias en CLI Init (REQ-23.1)

El comando `axiom init` DEBE admitir la bandera `--profile=knowledge` y su alias idéntico `--spec-only`. Si no se especifica perfil, se mantiene el comportamiento predeterminado de Axiom (`full`).

#### Scenario: Inicialización con bandera --profile=knowledge
- **DADO** un directorio destino para un nuevo proyecto
- **CUANDO** se ejecuta `axiom init --profile=knowledge`
- **ENTONCES** el inicializador activa la configuración del perfil ligero
- **Y** no genera arneses de ejecución de código ni de pruebas

#### Scenario: Inicialización con bandera alias --spec-only
- **DADO** un directorio destino para un nuevo proyecto
- **CUANDO** se ejecuta `axiom init --spec-only`
- **ENTONCES** el comportamiento y los artefactos generados son exactamente idénticos a los de `--profile=knowledge`

---

### Requirement: Configuración Canónica de axiom.yaml en Topología Multirepo (REQ-23.2)

Bajo el perfil `knowledge`, el archivo `axiom.yaml` DEBE generarse con:
- `workspace.topology: "multirepo"`.
- `workspace.specs_repository: "openspec"` (como repositorio separado e independiente desde cero).
- `roles.knowledge` con política `advisory`, sin compuertas bloqueantes de código.
- `governance.language: "es"`, `governance.shared_memory: "engram"` y `governance.semantic_analysis: "auto"`.

#### Scenario: Manifiesto axiom.yaml generado bajo perfil knowledge
- **DADO** un comando de inicialización con `--profile=knowledge`
- **CUANDO** se escribe `axiom.yaml`
- **ENTONCES** la sección `workspace` declara `topology: "multirepo"` y `specs_repository: "openspec"`
- **Y** la sección `governance` declara `language: "es"` y `shared_memory: "engram"`
- **Y** los roles definidos tienen política `gate_policy: "advisory"`

---

### Requirement: Andamiaje Estándar OpenSpec (REQ-23.3)

El inicializador DEBE crear la estructura canónica de documentación viva dentro del repositorio de especificaciones:
- `openspec/specs/` para las especificaciones vivas organizadas por dominio.
- `openspec/changes/` para las solicitudes de cambio y propuestas.
- `openspec/INDEX.md` inicializado como catálogo maestro vacío o con el encabezado de proyecto.

#### Scenario: Creación de directorios y catálogo OpenSpec
- **DADO** la ejecución de `axiom init --profile=knowledge`
- **CUANDO** finaliza la creación de archivos
- **ENTONCES** existen en disco los directorios `openspec/specs/` y `openspec/changes/`
- **Y** existe el archivo `openspec/INDEX.md` con la metadata inicial del catálogo maestro

---

### Requirement: Configuración Automática de Servidores MCP en .mcp.json (REQ-23.4)

El inicializador DEBE generar o actualizar el archivo `.mcp.json` en la raíz del workspace, registrando de forma predeterminada los servidores MCP:
- `engram`: memoria persistente de proyecto y sesiones.
- `serena`: indexación de código y símbolos AST.
- `codegraph`: grafo de llamadas y dependencias.
Si `.mcp.json` ya contenía otros servidores, el inicializador DEBE fusionar las entradas sin borrar las preexistentes.

#### Scenario: Inyección de servidores MCP en .mcp.json nuevo
- **DADO** un workspace sin archivo `.mcp.json` previo
- **CUANDO** se ejecuta `axiom init --profile=knowledge`
- **ENTONCES** se crea `.mcp.json` conteniendo las definiciones para `engram`, `serena` y `codegraph`

#### Scenario: Fusión no destructiva con .mcp.json existente
- **DADO** un workspace con un archivo `.mcp.json` que ya declara un servidor `custom-tool`
- **CUANDO** se ejecuta `axiom init --profile=knowledge`
- **ENTONCES** `.mcp.json` preserva `custom-tool` y añade `engram`, `serena` y `codegraph`

---

### Requirement: Oclusión Estricta de Ejecutores de Código y Suites de Testeo (REQ-23.5)

El perfil `knowledge` NO DEBE crear arneses de ejecución de código (`sdd-apply`), ni carpetas de sincronización pesadas de agentes ejecutores (`.axiom/inbox/skills/` con skills de compilación/testeo), ni scripts de pipelines de CI locales para suites de prueba.

#### Scenario: Ausencia de ejecutores y pipelines de testeo
- **DADO** la inicialización bajo el perfil `knowledge`
- **CUANDO** se verifica el árbol de archivos resultante
- **ENTONCES** no se instalan skills de ejecución de tests ni pipelines de build/test en el workspace

---

### Requirement: Creación de Borradores de Cambio para Traspaso Futuro (REQ-23.6)

El comando `axiom change create <slug>` DEBE operar con normalidad dentro del perfil `knowledge`, sembrando `proposal.md` y `spec.md` en `openspec/changes/<slug>/` para que agentes externos o sesiones posteriores de desarrollo en Axiom puedan adoptar el cambio e implementarlo.

#### Scenario: Creación de cambio en modo borrador
- **DADO** un workspace inicializado con el perfil `knowledge`
- **CUANDO** se invoca `axiom change create nuevo-modulo --intent "Definir nueva capacidad"`
- **ENTONCES** se crea `openspec/changes/nuevo-modulo/proposal.md`
- **Y** queda disponible para su posterior adopción por el carril de implementación de Axiom

---

## 2. Capacidad: `knowledge-sweep-command`

Permite realizar un barrido inicial rápido sobre los repositorios del proyecto para inferir la arquitectura técnica y los módulos funcionales sin bloqueos interactivos.

### Requirement: Invocación Autónoma Headless (REQ-23.7)

El subcomando `axiom knowledge sweep` DEBE soportar ejecución autónoma no interactiva (`--headless`, `--json`), adecuada para ser invocada desde el backend hacia CLIs de agentes de codificación (Claude Code, Codex, Copilot, Gemini CLI) sin requerir confirmación por terminal.

#### Scenario: Ejecución headless de barrido emitiendo JSON
- **DADO** un workspace con código preexistente
- **CUANDO** el backend ejecuta `axiom knowledge sweep --json`
- **ENTONCES** el comando concluye con código de salida 0 sin solicitar confirmación interactiva
- **Y** emite un JSON estructurado con la arquitectura técnica y los módulos funcionales detectados

---

### Requirement: Detección Técnica Multidimensional (REQ-23.8)

El barrido DEBE inspeccionar el árbol de código y las definiciones AST/símbolos disponibles (vía Serena o detector nativo) para identificar:
- Lenguajes de programación principales y secundarios.
- Frameworks y dependencias clave.
- Puntos de entrada (`main`, controladores, handlers, configuraciones).
- Estructura y capas de la arquitectura (monolito, microservicios, hexagonal, MVC, etc.).

#### Scenario: Identificación de stack y puntos de entrada
- **DADO** un repositorio con un proyecto Go y frontend React
- **CUANDO** se ejecuta el barrido `axiom knowledge sweep`
- **ENTONCES** el reporte técnico incluye `go` y `typescript/react` en lenguajes/frameworks
- **Y** señala los puntos de entrada principales detectados en el sistema

---

### Requirement: Inferencia Funcional de Módulos y Dominios (REQ-23.9)

El barrido DEBE agrupar la funcionalidad del código en módulos o dominios funcionales preliminares, deduciéndolos a partir de nombres de directorios, servicios, entidades de dominio y rutas/endpoints públicos.

#### Scenario: Inferencia de módulos de negocio
- **DADO** una estructura de código con directorios `billing/`, `auth/` y `catalog/`
- **CUANDO** finaliza el barrido
- **ENTONCES** el resultado clasifica preliminarmente los dominios `billing`, `auth` y `catalog` con sus componentes principales

---

### Requirement: Política No Bloqueante y Sobre Diferido de Ambigüedades (REQ-23.10)

Durante el barrido, el sistema NO DEBE detenerse ni formular preguntas al usuario ante dudas, incoherencias técnicas o código huérfano. Todas las lagunas, incertidumbres o ambigüedades detectadas DEBEN recopilarse en un sobre estructurado (`ambiguities`) que se devuelve al concluir el proceso.

#### Scenario: Detección de ambigüedades sin bloqueo
- **DADO** un repositorio con archivos huérfanos o dos frameworks en conflicto aparente
- **CUANDO** se ejecuta el barrido
- **ENTONCES** el proceso completa la ejecución sin pausar
- **Y** la salida incluye un apartado de `ambigüedades` describiendo las dudas encontradas para aclaración posterior

---

### Requirement: Siembra Inicial en OpenSpec y Persistencia en Engram (REQ-23.11)

Al finalizar con éxito el barrido, el comando DEBE:
1. Sembrar borradores de especificación viva en `openspec/specs/<dominio>/spec.md` para cada módulo funcional detectado.
2. Actualizar el índice general `openspec/INDEX.md`.
3. Persistir un resumen estructurado del análisis en Engram bajo el tópico `knowledge/sweep` con tipo `architecture`.

#### Scenario: Persistencia y catalogación del barrido inicial
- **DADO** un barrido completado con éxito
- **CUANDO** concluye la fase de persistencia
- **ENTONCES** se generan los archivos `spec.md` para los dominios detectados en `openspec/specs/`
- **Y** `openspec/INDEX.md` refleja los dominios agregados
- **Y** se guarda una memoria persistente en Engram con el mapa del proyecto

---

## 3. Capacidad: `knowledge-query-engine`

Permite formular consultas técnicas o funcionales sobre el proyecto, respondiendo con fundamentación y enriqueciendo automáticamente las especificaciones vivas.

### Requirement: Subcomando de Consulta Técnica/Funcional (REQ-23.12)

El sistema DEBE proveer el comando `axiom knowledge query "<pregunta>" [--type=technical|functional]` para recibir consultas en lenguaje natural orientadas a la arquitectura o a las reglas de negocio del proyecto.

#### Scenario: Invocación de consulta funcional
- **DADO** un workspace con documentación y código
- **CUANDO** se ejecuta `axiom knowledge query "¿Cómo se procesan los reembolsos?" --type=functional`
- **ENTONCES** el motor procesa la consulta bajo la lente funcional

---

### Requirement: Resolución Spec-First con Cortocircuito Inmediato (REQ-23.13)

El motor de consulta DEBE evaluar primero si la pregunta se encuentra resuelta con suficiente detalle y precisión dentro de las especificaciones vivas existentes (`openspec/specs/`). Si la respuesta está contenida en la spec, el motor DEBE responder directamente a partir de ella sin necesidad de inspeccionar el código ni invocar conectores semánticos.

#### Scenario: Respuesta directa desde la spec viva sin tocar código
- **DADO** una especificación viva `openspec/specs/billing/spec.md` que detalla las reglas de reembolso
- **CUANDO** se consulta `axiom knowledge query "¿Cuáles son las condiciones de reembolso?"`
- **ENTONCES** el motor responde utilizando la especificación viva
- **Y** no realiza llamadas a Serena ni a CodeGraph para inspeccionar código

---

### Requirement: Triangulación Profunda con Evidencias ante Lagunas en la Spec (REQ-23.14)

Si la consulta no está cubierta en la spec viva o requiere detalle de implementación no documentado, el motor DEBE realizar una investigación profunda triangulando:
1. Símbolos y AST mediante Serena.
2. Jerarquía de llamadas y dependencias mediante CodeGraph.
3. Decisiones y descubrimientos previos en la memoria Engram.
4. Código fuente correspondiente en los repositorios locales.

#### Scenario: Consulta que requiere inspección profunda de código
- **DADO** una consulta cuya respuesta no figura en `openspec/specs/`
- **CUANDO** el motor detecta la ausencia en la spec
- **ENTONCES** consulta a Serena para localizar las firmas relevantes y a CodeGraph para rastrear los invocadores
- **Y** examina las líneas de código afectadas

---

### Requirement: Contrato de Respuesta en Dos Niveles (REQ-23.15)

Toda respuesta emitida por `axiom knowledge query` DEBE estructurarse estrictamente en dos bloques:
1. **Respuesta Directa:** Explicación clara, concisa y rigurosa en castellano que resuelve la duda funcional o técnica.
2. **Evidencias Concretas:** Lista detallada de evidencias que sustentan la respuesta (ficheros, rangos de línea, declaraciones de tipos y relaciones de callers/callees identificadas).

#### Scenario: Formato de la respuesta estructurada
- **DADO** una consulta resuelta mediante evidencias de código
- **CUANDO** se presenta la salida al usuario
- **ENTONCES** la primera sección contiene la respuesta directa a la pregunta formulada
- **Y** la segunda sección lista las evidencias con rutas a los ficheros, números de línea y símbolos analizados

---

### Requirement: Auto-Enriquecimiento Obligatorio de la Spec Viva (REQ-23.16)

Cuando una consulta aporte nuevo conocimiento, reglas de negocio o detalles arquitectónicos que no figuraban previamente en la especificación viva, el motor DEBE:
1. Sintetizar el nuevo requerimiento o aclaración en el archivo `openspec/specs/<dominio>/spec.md` correspondiente.
2. Actualizar el índice general `openspec/INDEX.md`.
3. Registrar la observación aprendida en Engram con tipo `discovery` o `architecture`.
Si la consulta no generó conocimiento nuevo (fue resuelta por la spec existente), no se reescriben los archivos.

#### Scenario: Auto-enriquecimiento de la spec tras investigación
- **DADO** una consulta investigada en el código cuya regla no existía en `openspec/specs/`
- **CUANDO** concluye la respuesta
- **ENTONCES** el motor incorpora el nuevo requerimiento a `openspec/specs/<dominio>/spec.md`
- **Y** actualiza `openspec/INDEX.md`
- **Y** asienta el descubrimiento en la memoria Engram con `mem_save`
