# Propuesta: Perfil Knowledge y Descubrimiento Autónomo Multirrepo (inc-23-knowledge-profile)

## Propósito (Intent)

Proporcionar un perfil de inicialización ligero en Axiom (`axiom init --profile=knowledge` / `--spec-only`) orientado a la integración con `app-knowledge-agent` y agentes de catalogación, junto con comandos y skills especializadas de barrido autónomo y consulta evidenciada sobre especificaciones vivas. 

El perfil permite operar en topología `multirepo` con repositorio canónico OpenSpec independiente y desde cero, memoria compartida Engram y conectores semánticos (`serena`, `codegraph`), excluyendo por completo ejecutores de código (`sdd-apply`), suites de testeo y arneses pesados de compilación.

---

## Alcance (Scope)

### Dentro de Alcance (In Scope)

1. **Perfil de Inicialización Ligero (`axiom init --profile=knowledge`, `--spec-only`):**
   - Configuración de `axiom.yaml` con `topology: "multirepo"`, `specs_repository: "openspec"`, gobernanza en castellano (`language: "es"`), memoria compartida (`shared_memory: "engram"`) y conector semántico automático.
   - Creación del andamiaje canónico OpenSpec: `INDEX.md`, directorio `specs/` y `changes/`.
   - Generación de `.mcp.json` en la raíz del workspace declarando los servidores MCP esenciales: `engram`, `serena` y `codegraph`.
   - Exclusión explícita de ejecutores de código (`sdd-apply`), pipelines de CI locales pesados y runners de pruebas automatizadas.
   - Habilitación de `axiom change create` para sembrar borradores `proposal.md` y `spec.md` listos para ser retomados por el carril completo de desarrollo.

2. **Caso 1 — Barrido Inicial Técnico/Funcional Autónomo (`axiom knowledge sweep`):**
   - Diseñado para invocación autónoma y headless desde el backend hacia CLIs de agentes (Claude Code, Codex, Copilot, Gemini CLI).
   - Detección de arquitectura técnica (lenguajes, frameworks, librerías y puntos de entrada) y modularidad funcional (servicios, dominios y APIs).
   - Política no bloqueante de ambigüedades: las dudas, incoherencias o lagunas no interrumpen la ejecución interactiva; se recopilan y se entregan al final en un informe estructurado de incertidumbres.
   - Siembra inicial de las especificaciones maestras en `openspec/specs/` y consolidación en `openspec/INDEX.md` y Engram.

3. **Caso 2 — Consulta Asistida y Auto-enriquecimiento de la Spec Viva (`axiom knowledge query`):**
   - Estrategia *Spec-First*: si la consulta (técnica o funcional) ya está resuelta en las especificaciones vivas, se responde inmediatamente desde la spec sin necesidad de inspeccionar el código.
   - Inspección profunda con evidencia: si la spec carece del detalle necesario, triangula con Serena (símbolos y AST), CodeGraph (grafo de dependencias y llamadas) y Engram.
   - Contrato de respuesta en dos niveles:
     1. Respuesta directa, precisa y fundamentada a la consulta.
     2. Evidencias concretas (rutas de fichero, líneas y relaciones del grafo de llamadas).
   - Auto-enriquecimiento obligatorio: síntesis y actualización automática del conocimiento adquirido en `openspec/specs/<dominio>/spec.md`, regeneración de `openspec/INDEX.md` y registro en Engram (`type: discovery` / `architecture`).

### Fuera de Alcance (Out of Scope)

- Ejecución de suites de prueba automáticas o compiladores en proyectos bajo el perfil `knowledge`.
- Fases de implementación de código (`sdd-apply`) o verificación con ejecución de tests (`sdd-verify`).
- Alteración de los contratos de gobernanza y compuertas de bloque para perfiles estándar de desarrollo completo en Axiom.

---

## Capacidades (Capabilities)

### Nuevas Capacidades
- `knowledge-profile-init`: Inicializador de workspace ligero con topología multirepo, andamiaje OpenSpec y configuración `.mcp.json` (Engram + Serena + CodeGraph).
- `knowledge-sweep-command`: Barrido estático rápido técnico/funcional headless con reporte diferido de ambigüedades.
- `knowledge-query-engine`: Motor de consulta spec-first con triangulación de código/AST/grafo y auto-enriquecimiento de especificaciones vivas.

---

## Enfoque de Implementación (Approach)

1. **internal/hub:** Extender `InitOptions` con el campo `Profile` (`knowledge`, `spec-only`), adaptar `buildAxiomYamlWithRoles` para topología `multirepo` con rol consultor `knowledge`, y generar `.mcp.json` con los tres servidores MCP.
2. **internal/knowledge:** Crear el paquete de dominio con la lógica del barrido (`sweep`) y del motor de consultas (`query`) con estrategia spec-first y enriquecimiento de `openspec/`.
3. **cmd/axiom:** Registrar los subcomandos `axiom init --profile=knowledge` y `axiom knowledge [sweep|query]`.
4. **Verificación Formal:** Batería de pruebas unitarias y de caracterización para la generación de archivos, detección semántica, reporte diferido de ambigüedades y actualización idempotente de la spec viva.
