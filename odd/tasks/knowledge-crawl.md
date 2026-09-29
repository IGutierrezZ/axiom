# ODD: Motor de Crawling Exhaustivo y Generación de Living Spec (knowledge crawl)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/knowledge-crawl.md`.  
> Espejo de recuperación en Engram: topic `odd/knowledge-crawl/tasks`, proyecto `axiom`.

## Objetivo

Implementar en Axiom el motor de generación exhaustiva de especificaciones vivas (`axiom knowledge crawl`), estructurado bajo una arquitectura desacoplada y orientada a procesos batch reanudables:
1. **Planificación Estructural Determinista:** Análisis de código (AST, paquetes, dependencias e interfaces) sin consumo de LLM para particionar el proyecto en unidades de trabajo discretas (`job.json`).
2. **Ciclo de Registro de Unidades:** Subcomando para validar e inyectar el análisis semántico producido por agentes/aplicativos en `openspec/specs/<dominio>/spec.md` con trazabilidad de estado.
3. **Consolidación Final e Integridad:** Reconciliación de enlaces cruzados, actualización de `openspec/INDEX.md` y persistencia en Engram.
4. **Inspección de Progreso:** Subcomando `--status` que expone métricas y estados para orquestadores y GUIs/TUIs.

---

## Tareas

- [x] **T1 · Modelo de datos y manifiesto de trabajo (`job.json`)**
  - Definir estructuras Go para `CrawlJob`, `CrawlUnit`, dependencias, interfaces y estados (`pending`, `in_progress`, `completed`, `failed`).
  - Serialización y deserialización atómica en `.axiom/knowledge/crawl-job.json`.
- [ ] **T2 · Motor de particionamiento determinista (`--plan`)**
  - Extender `internal/knowledge` para descomponer el repositorio a nivel de módulos y submódulos a partir de AST e importaciones.
  - Generación de contexto acotado por unidad (ficheros, firmas públicas, interfaces expuestas).
- [ ] **T3 · Inyección y validación de unidades (`--record-unit`)**
  - Validador de formato canónico OpenSpec para el contenido entregado por el agente.
  - Escritura atómica en `openspec/specs/<dominio>/spec.md` y actualización del estado de la unidad en el manifiesto.
- [ ] **T4 · Consulta de estado y progreso (`--status`)**
  - Salida legible y formato JSON con porcentaje de completitud, unidades pendientes y fallidas.
- [ ] **T5 · Consolidación final (`--finalize`)**
  - Reconciliación del catálogo maestro `openspec/INDEX.md` y persistencia de resumen en Engram.
- [ ] **T6 · Integración en CLI (`cmd/axiom/main.go`) y suite de pruebas**
  - Añadir flags y subcomandos bajo `axiom knowledge crawl`.
  - Pruebas unitarias e integración con repositorios sintéticos.

---

## Verificación Ejecutable

- `go test ./internal/knowledge/...` cubriendo planificación, registro y consolidación.
- `go test ./cmd/axiom -run TestKnowledgeCrawl` verificando la interfaz CLI.
- Verificación en worktree aislado `F:\repos\axiom-wt\knowledge-crawl` sobre rama `feat/knowledge-crawl`.
