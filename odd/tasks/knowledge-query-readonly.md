# ODD: Consulta de Conocimiento de Solo Lectura y Búsqueda Léxica Ponderada (knowledge-query-readonly)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/knowledge-query-readonly.md`.  
> Rama de trabajo: `fix/query-readonly`.

## Objetivo

Corregir el motor de búsqueda y desacoplar el auto-enriquecimiento en `axiom knowledge query` (`internal/knowledge/query.go`):
1. **Desacoplar Mutación de Consulta (CQS):** Convertir `axiom knowledge query` en una operación estrictamente de solo lectura por defecto. El auto-enriquecimiento de especificaciones vivas solo se ejecutará bajo petición explícita (`--enrich` / `Enrich: true`) y condicionado a un umbral alto de relevancia.
2. **Matching Léxico Riguroso por Límites de Palabra:** Reemplazar `strings.Contains` por coincidencia de límites de palabra (`\b`, camelCase, separadores de identificadores) y ampliar la lista de palabras vacías (stop words) para erradicar falsos positivos por subcadenas (ej. `"dar"` dentro de `"calendar"`).
3. **Scoring y Ranking Multitérmino:** Implementar un sistema de puntuación conjuntiva ponderada (AND suave) que priorice líneas y archivos con múltiples términos coincidentes, recorriendo todo el árbol de código antes de seleccionar las evidencias más relevantes.
4. **Validación Estricta en `searchLivingSpecs`:** Exigir correlación léxica de alta densidad dentro del bloque de requerimiento antes de marcar `resolved_from_spec: true`, evitando cortocircuitos falsos sobre especificaciones existentes.
5. **Suite de Pruebas Unitarias:** Cubrir en `query_test.go` y adaptar `knowledge_test.go` y `knowledge_cli_test.go` para verificar el comportamiento de solo lectura, la ausencia de falsos positivos y el ranking de evidencias.

---

## Tareas

- [x] **T1 · Filtrado léxico y matching por límites de palabra (`extractKeywords` y `matchesKeyword`)**
  - Ampliar diccionario de stop words con verbos auxiliares e interrogativos comunes.
  - Implementar tokenización y coincidencia por límites de palabra/identificador en código (camelCase, snake_case, delimitadores).
- [x] **T2 · Scoring ponderado y selección de evidencias en `searchCodebase`**
  - Eliminar el corte prematuro en 5 coincidencias sobre el recorrido alfabético.
  - Evaluar densidad de coincidencia por línea y fichero (prioridad a coincidencias múltiples).
  - Ordenar y devolver el top de evidencias por score.
- [x] **T3 · Desacople de auto-enriquecimiento y validación en `searchLivingSpecs`**
  - Añadir opción `Enrich` (por defecto `false`) en `QueryOptions` y bandera `--enrich` en CLI.
  - `RunQuery` no muta `spec.md` a menos que `opts.Enrich == true`.
  - Reforzar `searchLivingSpecs` para exigir correlación densa dentro del bloque de requerimiento.
- [x] **T4 · Adaptación de CLI en `cmd/axiom/main.go`**
  - Registrar flag `--enrich` en `runKnowledgeQuery`.
  - Documentar en el texto de ayuda que `query` es de solo lectura por defecto.
- [x] **T5 · Batería de pruebas y verificación formal**
  - Crear `internal/knowledge/query_test.go` validando casos de falsos positivos ("dar" en "calendar"), ranking y solo lectura.
  - Adaptar pruebas existentes en `knowledge_test.go` y `knowledge_cli_test.go`.
  - Ejecutar `go test ./internal/knowledge/...` y `go test ./cmd/axiom -run TestCLI`.

---

## Verificación Ejecutable

- `go test -v ./internal/knowledge -run TestQuery`
- `go test -v ./cmd/axiom -run TestCLI`
- `go test ./...`
