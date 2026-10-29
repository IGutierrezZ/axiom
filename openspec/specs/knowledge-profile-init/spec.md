# Especificación de Requerimientos: Perfil de Inicialización Knowledge (INC-23)

> **Incremento:** `inc-23-knowledge-profile` · **Capacidad:** `knowledge-profile-init` · **Tipo:** capacidad nueva
> **Fase del Roadmap:** Fase 4 — Gobernanza de Agentes, Integración de Conocimiento y Descubrimiento Semántico  
> **Responsabilidad:** Hub & Init / Dominio Knowledge / CLI Commands / OpenSpec Living Documentation / MCP Integration  
> **Idioma:** Español (castellano peninsular)  

## Propósito

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
