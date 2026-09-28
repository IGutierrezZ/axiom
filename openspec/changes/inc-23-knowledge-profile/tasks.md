# Tareas: Perfil de Inicialización Knowledge, Barrido Autónomo Multirrepo y Consulta Spec-First con Auto-Enriquecimiento (inc-23-knowledge-profile)

> **Fuentes de alcance:** `spec.md` (3 capacidades, REQ-23.1–REQ-23.16), `design.md` (Decisiones D-01 a D-06, contratos de tipos y CLI), `proposal.md` como contexto de intención.  
> **Idioma del artefacto:** Español (castellano peninsular, tuteo profesional). Identificadores Go, rutas, banderas de CLI, claves YAML y nombres de test permanecen en inglés.  

---

## 1. Fase 1: Perfil de Inicialización Knowledge en `internal/hub`

- [x] `T-23.1.1` **Extensión de contratos de inicialización en `internal/hub/types.go`**
  - Añadir el campo `Profile string` (`"knowledge"`, `"spec-only"`, `"full"`) a la estructura `InitOptions`.
  - Exportar constantes o helpers de validación para perfiles admitidos.

- [x] `T-23.1.2` **Lógica de inicialización del perfil knowledge en `internal/hub/init.go`**
  - Si `Profile == "knowledge"` o `Profile == "spec-only"`:
    - Forzar topología `multirepo` y repositorio canónico `specs_repository: "openspec"`.
    - Generar rol consultor único `knowledge` con `gate_policy: "advisory"` y tecnologías `["markdown", "openspec"]`.
    - Crear directorios canónicos `openspec/specs/` y `openspec/changes/`.
    - Crear `openspec/INDEX.md` inicial si no existe.
    - Omitir la creación del directorio `.axiom/inbox/skills/` y de arneses de ejecución.

- [x] `T-23.1.3` **Generación y fusión no destructiva de `.mcp.json` en `internal/hub/mcp.go`**
  - Implementar la función `injectKnowledgeMCPServers(workspaceRoot string) error` que asegure los servidores `engram`, `serena` y `codegraph`.
  - Si el archivo `.mcp.json` ya existe, fusionar las claves dentro de `mcpServers` preservando servidores preexistentes del usuario.

- [x] `T-23.1.4` **Pruebas unitarias de inicialización en `internal/hub/init_test.go`**
  - `TestInitKnowledgeProfile`: Comprobar que `axiom.yaml` se genera con topología multirrepo, rol `knowledge` advisory y sin carpetas de skills de testeo.
  - `TestInitKnowledgeProfileMergeMCP`: Comprobar la fusión no destructiva cuando `.mcp.json` ya contiene herramientas previas.

---

## 2. Fase 2: Motor de Dominio Knowledge (`internal/knowledge`)

- [x] `T-23.2.1` **Modelos de datos y tipos en `internal/knowledge/types.go`**
  - Definir `SweepOptions`, `SweepResult`, `ModuleSummary` y `AmbiguityItem`.
  - Definir `QueryOptions`, `QueryResult`, `EvidenceItem` y `QueryType` (`auto`, `technical`, `functional`).

- [x] `T-23.2.2` **Motor de barrido autónomo en `internal/knowledge/sweep.go`**
  - Implementar `RunSweep(ctx context.Context, opts SweepOptions) (*SweepResult, error)`.
  - Inspeccionar el árbol de código y dependencias usando `hub.Detector` para deducir lenguajes, frameworks y entrypoints.
  - Inferir módulos funcionales a partir de la estructura de directorios y nombres de paquetes.
  - Acumular incoherencias o dudas en el sobre diferido `Ambiguities` sin bloquear la ejecución ni solicitar entrada interactiva.
  - Sembrar borradores de especificación en `openspec/specs/<modulo>/spec.md` y actualizar `openspec/INDEX.md`.

- [x] `T-23.2.3` **Motor de consulta con estrategia Spec-First y auto-enriquecimiento en `internal/knowledge/query.go`**
  - Implementar `RunQuery(ctx context.Context, opts QueryOptions) (*QueryResult, error)`.
  - **Paso Spec-First:** Comprobar si `openspec/specs/` ya contiene la respuesta; en caso afirmativo, responder inmediatamente sin tocar código (`ResolvedFromSpec: true`).
  - **Paso Triangulación con Evidencias:** Si la spec no responde, buscar definiciones AST/símbolos y callers/callees.
  - **Contrato de Salida:** Formatear la salida estructurada con *Respuesta Directa* primero y *Evidencias Concretas* después.
  - **Paso Auto-Enriquecimiento:** Redactar y añadir el nuevo requerimiento a `openspec/specs/<modulo>/spec.md` de forma idempotente y actualizar `openspec/INDEX.md`.

- [x] `T-23.2.4` **Pruebas unitarias y de caracterización en `internal/knowledge/knowledge_test.go`**
  - `TestSweepAutonomousHeadless`: Verificar barrido sin bloqueos y con reporte diferido de ambigüedades.
  - `TestQuerySpecFirstCache`: Verificar que ante una regla ya documentada en la spec viva no se analiza código.
  - `TestQueryDeepInspectionAndSpecEnrichment`: Verificar que una consulta no documentada extrae evidencias del código y actualiza la spec viva e `INDEX.md`.

---

## 3. Fase 3: Integración en CLI (`cmd/axiom/main.go`)

- [x] `T-23.3.1` **Soporte de banderas en `axiom init`**
  - Agregar `--profile` y `--spec-only` al `flag.FlagSet` de `runInit`.
  - Mapear `--spec-only` hacia `Profile = "knowledge"`.

- [x] `T-23.3.2` **Nuevo subcomando `axiom knowledge` en `cmd/axiom/main.go`**
  - Añadir enrutamiento para `axiom knowledge [sweep|query]`.
  - Implementar despachador `runKnowledge` soportando banderas `--json`, `--headless`, `--type`, `--cwd` y `--deep`.

- [x] `T-23.3.3` **Verificación de integración CLI y ayuda**
  - Verificar salida de `axiom --help` y `axiom knowledge --help`.
  - Probar ejecución de extremo a extremo en entorno temporal.
