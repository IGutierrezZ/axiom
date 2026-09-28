# Diseño Técnico: Perfil de Inicialización Knowledge, Barrido Autónomo Multirrepo y Consulta Spec-First con Auto-Enriquecimiento (inc-23-knowledge-profile)

> **Incremento:** `inc-23-knowledge-profile`  
> **Fase:** `sdd-design` · **Fecha:** 2026-09-29  
> **Fuente de alcance:** `openspec/changes/inc-23-knowledge-profile/proposal.md` y `openspec/changes/inc-23-knowledge-profile/spec.md` (REQ-23.1 – REQ-23.16)  
> **Idioma:** Español (castellano peninsular). Identificadores Go, rutas, banderas de CLI y claves YAML/JSON en inglés.  
> **Verificación de árbol:** Ficheros, paquetes y contratos comprobados en el árbol de trabajo (`internal/hub/`, `internal/semantic/`, `internal/workspace/`, `cmd/axiom/main.go`).  

---

## 0. Resumen del Diseño

| Pregunta | Respuesta de Arquitectura |
|---|---|
| ¿Cuál es la pieza central? | Dos componentes coordinados: (1) Extensión de `internal/hub` para el perfil `knowledge` (manifiesto multirrepo, andamiaje OpenSpec y `.mcp.json` con Engram, Serena y CodeGraph sin ejecutores); (2) Nuevo dominio `internal/knowledge` que aloja el motor de barrido autónomo (`sweep`) y el motor de consultas (`query`). |
| ¿Qué topología se aplica? | `topology: "multirepo"` estricta, desacoplando el repositorio de código del repositorio canónico `openspec/specs/`. |
| ¿Cómo opera el barrido? | Modo autónomo/headless para invocación desde el backend de orquestación hacia CLIs (Claude Code, Codex, Copilot, Gemini CLI). No bloquea con preguntas interactivas; acumula dudas y las entrega en un sobre estructurado de ambigüedades al final. |
| ¿Cómo opera la consulta? | *Spec-First*: si la consulta está respondida en `openspec/specs/`, responde directamente sin analizar código. Si requiere inspección, triangula Serena (AST), CodeGraph (llamadas) y Engram, estructura la salida (Respuesta Directa $\to$ Evidencias) y auto-enriquece la spec viva y Engram de forma idempotente. |
| ¿Qué NO es esto? | No es un motor de ejecución ni de testeo. No instala ni configura `sdd-apply`, test runners ni pipelines pesados de compilación local. |
| ¿Qué ficheros se modifican? | `internal/hub/types.go`, `internal/hub/init.go`, `internal/hub/init_test.go`, `cmd/axiom/main.go`. Se crea el nuevo paquete `internal/knowledge/` (`types.go`, `sweep.go`, `query.go`, `mcp.go`, `knowledge_test.go`). |

---

## 1. Enfoque Técnico y Verificación de Árbol

### 1.1 Verificación de Componentes Existentes

1. **`internal/hub/init.go` y `internal/hub/types.go`**:
   - `InitOptions` recibe hoy `Path`, `Name`, `Topology`, `Force`, `Roles`.
   - `buildAxiomYamlWithRoles` genera la estructura `workspace`, `roles` y `governance`.
   - `init.go` escribe `.axiom/inbox/skills` y las carpetas `openspec/specs` y `openspec/changes`.
   - **Adaptación requerida:** Añadir `Profile string` a `InitOptions`. Si `Profile == "knowledge"` o `Profile == "spec-only"`, forzar `Topology = "multirepo"`, omitir `.axiom/inbox/skills`, registrar rol único `knowledge` con `gate_policy: "advisory"`, y generar/fusionar `.mcp.json`.

2. **`internal/semantic/detector.go` y `internal/semantic/service.go`**:
   - `Detector` ya sabe detectar agentes y configuraciones de Serena y CodeGraph a nivel de workspace (`.mcp.json`) y usuario.
   - `Service` implementa `GetStatus` y `FindSymbols`.
   - **Adaptación requerida:** Reutilizar el detector y servicio semántico en el nuevo paquete `internal/knowledge` para el barrido técnico y la búsqueda de símbolos sin duplicar lógica de inspección.

3. **`internal/workspace/types.go`**:
   - `TopologyMultirepo` ya está tipada como `"multirepo"`.
   - `WorkspaceConfig` y `GovernanceConfig` soportan `specs_repository`, `language`, `shared_memory` y `semantic_analysis`.

---

## 2. Decisiones de Diseño (Design Decisions)

### D-01: Perfil Declarativo en `InitOptions` y Alias `--spec-only`
`InitOptions` incorpora el campo `Profile string`. Las banderas `--profile=knowledge` y `--spec-only` son tratadas de forma canónica y equivalente en `cmd/axiom/main.go`. Si se pasa `--spec-only`, se normaliza a `Profile = "knowledge"`.

### D-02: Topología `multirepo` con Repositorio de Especificaciones Decoupled
El perfil `knowledge` fija `topology: "multirepo"` y `specs_repository: "openspec"`. Esto garantiza que las especificaciones canónicas puedan residir en un repositorio Git independiente o en la raíz dedicada sin acoplarse rígidamente a la estructura de directorios del código fuente.

### D-03: Generación e Inyección No Destructiva de `.mcp.json`
El inicializador genera `.mcp.json` en la raíz del workspace con:
```json
{
  "mcpServers": {
    "engram": {
      "command": "engram",
      "args": ["serve"]
    },
    "serena": {
      "command": "serena",
      "args": ["serve"]
    },
    "codegraph": {
      "command": "codegraph",
      "args": ["serve"]
    }
  }
}
```
Si `.mcp.json` ya existía, se deserializa como objeto genérico JSON, se fusionan las entradas en `mcpServers` respetando los servidores preexistentes y se serializa ordenado.

### D-04: Exclusión Estricta de Arneses de Ejecución
Bajo el perfil `knowledge`, no se crea la carpeta `.axiom/inbox/skills/`, no se despliegan plantillas de test runners (`go test`, `npm test`, etc.) y el rol generado es:
```yaml
roles:
  knowledge:
    name: "Knowledge Explorer"
    gate_policy: "advisory"
    repositories:
      - path: "."
    tech:
      - "markdown"
      - "openspec"
```

### D-05: Barrido Inicial Autónomo con Reporte Diferido de Ambigüedades
El barrido técnico (`axiom knowledge sweep`) ejecuta un análisis en dos fases:
1. **Fase Técnica:** Lee dependencias (p. ej., `go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`), detecta frameworks, localiza archivos principales (`main.go`, `index.ts`, `app.py`) e inspecciona símbolos públicos.
2. **Fase Funcional:** Identifica módulos a partir de directorios temáticos (`auth/`, `billing/`, `api/`, `domain/`, etc.) y analiza comentarios o interfaces clave.
3. **Manejo de Incoherencias:** Si se detectan dependencias conflictivas, rutas huérfanas o módulos sin entrypoint aparente, no se interrumpe la ejecución; se emiten como entradas en `Ambiguities []AmbiguityItem` en la respuesta JSON o en la sección final de texto.
4. **Siembra Canónica:** Crea `openspec/specs/<modulo>/spec.md` con un esquema estándar BDD preliminar y añade la fila correspondiente a `openspec/INDEX.md`.

### D-06: Estrategia de Consulta Spec-First con Triangulación Progresiva
El comando `axiom knowledge query` implementa un algoritmo de resolución estratificado:
1. **Paso 1 (Spec-First Cache):** Lee los archivos en `openspec/specs/**/*.md`. Realiza un análisis léxico/semántico buscando el requerimiento o escenario BDD que conteste la pregunta. Si la confianza es alta, devuelve la respuesta de inmediato y marca `ResolvedFromSpec: true`.
2. **Paso 2 (Triangulación de Código):** Si la spec viva no contiene la respuesta:
   - Consulta a Serena para ubicar símbolos, interfaces y tipos vinculados a las palabras clave.
   - Consulta a CodeGraph para descubrir callers y callees del flujo.
   - Lee fragmentos de código relevantes.
   - Triangula con recuerdos previos en Engram (`mem_search`).
3. **Paso 3 (Formato de Respuesta):** Produce un bloque de texto que inicia con la **Respuesta Directa**, seguido de la sección **Evidencias** (tabla con fichero, líneas y símbolo).
4. **Paso 4 (Auto-enriquecimiento Idempotente):** Determina el módulo funcional de destino. Agrega o actualiza una sección en `openspec/specs/<modulo>/spec.md`, recalcula los totales de requerimientos y escenarios en `openspec/INDEX.md` y guarda una memoria en Engram con `type: discovery`.

---

## 3. Estructuras de Datos y Tipos (`internal/knowledge/types.go`)

```go
package knowledge

import "time"

// AmbiguityItem representa una duda, inconsistencia o laguna técnica detectada durante el barrido.
type AmbiguityItem struct {
	Category    string `json:"category"`    // "architecture", "dependency", "module", "orphan"
	Severity    string `json:"severity"`    // "low", "medium", "high"
	Path        string `json:"path"`
	Description string `json:"description"`
	Remediation string `json:"remediation,omitempty"`
}

// ModuleSummary describe un módulo funcional detectado.
type ModuleSummary struct {
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	Description string   `json:"description"`
	Components  []string `json:"components,omitempty"`
}

// SweepOptions define los parámetros para el comando axiom knowledge sweep.
type SweepOptions struct {
	WorkspaceRoot string
	SpecsRoot     string
	Headless      bool
	Format        string // "text", "json"
}

// SweepResult contiene los resultados del barrido técnico y funcional.
type SweepResult struct {
	PrimaryLanguage string          `json:"primary_language"`
	Frameworks      []string        `json:"frameworks"`
	Entrypoints     []string        `json:"entrypoints"`
	Modules         []ModuleSummary `json:"modules"`
	Ambiguities     []AmbiguityItem `json:"ambiguities"`
	CreatedSpecs    []string        `json:"created_specs"`
	Duration        time.Duration   `json:"duration"`
}

// QueryType define la lente de análisis para la consulta.
type QueryType string

const (
	QueryTypeAuto       QueryType = "auto"
	QueryTypeTechnical  QueryType = "technical"
	QueryTypeFunctional QueryType = "functional"
)

// EvidenceItem detalla una evidencia de código que respalda la respuesta.
type EvidenceItem struct {
	Source  string `json:"source"` // "spec", "serena", "codegraph", "code"
	File    string `json:"file"`
	Lines   string `json:"lines,omitempty"`
	Symbol  string `json:"symbol,omitempty"`
	Context string `json:"context,omitempty"`
}

// QueryOptions define los parámetros para el comando axiom knowledge query.
type QueryOptions struct {
	WorkspaceRoot string
	SpecsRoot     string
	Question      string
	Type          QueryType
	ForceDeep     bool
}

// QueryResult detalla la respuesta estructurada y los efectos secundarios de auto-enriquecimiento.
type QueryResult struct {
	DirectAnswer     string         `json:"direct_answer"`
	Evidences        []EvidenceItem `json:"evidences"`
	ResolvedFromSpec bool           `json:"resolved_from_spec"`
	SpecUpdated      bool           `json:"spec_updated"`
	TargetSpecPath   string         `json:"target_spec_path,omitempty"`
}
```

---

## 4. Paquetes y Contratos de Implementación

### 4.1 `internal/hub`
- **`types.go`**:
  ```go
  type InitOptions struct {
      Path     string
      Name     string
      Topology string
      Force    bool
      Roles    []RoleInput
      Profile  string // "knowledge", "spec-only", "full"
  }
  ```
- **`init.go`**:
  - Si `opts.Profile == "knowledge"` o `opts.Profile == "spec-only"`:
    - `opts.Topology = "multirepo"`.
    - Omite la creación de `.axiom/inbox/skills/`.
    - Asegura `openspec/specs/`, `openspec/changes/` y `openspec/INDEX.md`.
    - Genera o actualiza `.mcp.json` con `engram`, `serena` y `codegraph`.

### 4.2 `internal/knowledge`
- **`mcp.go`**: Lógica de generación y fusión no destructiva de `.mcp.json`.
- **`sweep.go`**: Ejecutor de barrido. Lee la estructura del árbol, detecta stack y módulos, acumula ambigüedades sin bloquear y siembra los borradores en `openspec/specs/` e `INDEX.md`.
- **`query.go`**: Ejecutor de consultas. Implementa la búsqueda spec-first, la triangulación profunda (código + AST), el formateo de salida (Respuesta Directa $\to$ Evidencias) y la actualización de la spec viva.
- **`index.go`**: Utilidades para regenerar o actualizar las filas del catálogo maestro `openspec/INDEX.md`.

### 4.3 `cmd/axiom/main.go`
- **`runInit`**: Soporta banderas `-profile` / `--profile` y `-spec-only` / `--spec-only`.
- **`runKnowledge`**: Nuevo despachador para `axiom knowledge [sweep|query]`.

---

## 5. Contratos de Comandos CLI

```bash
# Inicialización con perfil knowledge (topología multirepo + .mcp.json)
axiom init --profile=knowledge [--path <ruta>] [--name <nombre>]
axiom init --spec-only [--path <ruta>]

# Barrido inicial rápido técnico/funcional
axiom knowledge sweep [--cwd <ruta>] [--json] [--headless]

# Consulta spec-first con auto-enriquecimiento de especificaciones vivas
axiom knowledge query "<pregunta>" [--cwd <ruta>] [--type technical|functional] [--deep]
```

---

## 6. Plan de Pruebas y Casos Límite

1. **`TestInitKnowledgeProfile`**:
   - Verifica que `axiom.yaml` contiene `topology: "multirepo"` y `specs_repository: "openspec"`.
   - Verifica que `.mcp.json` contiene `engram`, `serena` y `codegraph`.
   - Verifica que no se crea la carpeta `.axiom/inbox/skills/`.
   - Verifica que `openspec/INDEX.md` se crea con la estructura de catálogo.
2. **`TestInitKnowledgeProfileMergeMCP`**:
   - Verifica que un `.mcp.json` preexistente conserva los servidores previos al inyectar `engram`, `serena` y `codegraph`.
3. **`TestKnowledgeSweep`**:
   - Ejecuta el barrido sobre un directorio de prueba con Go y React.
   - Comprueba la detección de lenguajes, frameworks y módulos funcionales.
   - Comprueba que las ambigüedades no causan pánico ni bloqueo interactivo.
   - Verifica la creación de las specs iniciales y la actualización de `INDEX.md`.
4. **`TestKnowledgeQuerySpecFirst`**:
   - Comprueba que si la spec viva ya contiene la respuesta, no se examina código y `ResolvedFromSpec == true`.
5. **`TestKnowledgeQueryDeepWithEnrichment`**:
   - Comprueba que ante una regla ausente en la spec, se extraen evidencias del código y se actualiza la spec viva con el nuevo requerimiento de forma idempotente.
