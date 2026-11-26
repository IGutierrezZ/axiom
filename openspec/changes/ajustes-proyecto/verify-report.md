# Informe de verificación: ajustes-proyecto (C-06-e)

Rol `fullstack`. Alcance: `git diff origin/main...HEAD`. Verificación localizada; sin suite completa ni binarios sueltos.

## Tareas

9/9 marcadas `[x]` en `tasks.md`; 8 commits de unidad (71c56d0ff, b6e0fbff6, 717a013b7, af2d8507c, aae568291, a1299919d, 82a825cde, 979545eb7). La tarea 9 no produjo commit. `kickoff.yaml` solo se añade (sellado) y no hay `gates.yaml` en el diff.

## Comandos ejecutados

| Comando | Resultado |
|---|---|
| `go test ./internal/settings` | ok |
| `go test ./internal/workspace ./internal/multirole ./internal/hub -run 'Promotion\|WorkflowLane\|ExplainerHTML\|GatePolic\|PolicyViolations'` | ok |
| `go test ./internal/dashboard -run 'Settings\|WorktreePrefs\|BranchPolicy\|Ecosystem\|SecretsCanary\|Classified\|WorkspaceIsReadOnly\|Assets\|Docs'` | ok |
| `go test ./internal/idmigration -run Residue` | FAIL: `openspec/changes/ajustes-proyecto/tasks.md:124 el token heredado: cita heredada sin residuo declarado` |
| `go vet` (settings, workspace, multirole, hub, dashboard) | sin salida |
| `gofmt -l internal/` | sin salida |

## Requisitos

- RF-TOP-21 (vocabularios cerrados): cumplido; accesores con uso en validadores y catálogo; pruebas de catálogo.
- RF-TOP-22/23 (escritura con round-trip, revisión, validación): cumplido; `write.go`, `patch.go`, `layout.go`, golden y round-trip de ficheros reales.
- RF-WEB-01/07/10 (MODIFIED): cumplido; ruta `#/ajustes`, GET abiertos, PUT con `rejectNonLocal`.
- RF-WEB-18..21: cumplido; `/api/settings`, `/api/settings/personal`, Ecosistema enlaza a Ajustes.

## Comprobaciones manuales

- `isLocalRequest` cubre los tres PUT (`settings_api.go:23,68,173` y `worktree_prefs.go` PUT) antes de leer el cuerpo.
- Canario de secretos: `/api/settings` y `/api/settings/personal` en `coveredRoutes`; prueba verde.
- Sin controles RDD ni `integrations` en `ajustes.js` ni `index.html`.
- Trinquete de código muerto: `settings.Read`, `settings.Write` y los cuatro accesores tienen uso productivo (dashboard y validadores). Sin otras exportaciones nuevas relevantes.
- Hueco 2: 400 con cuerpo mal formado, `{}` o sin `autofetch`, cubierto (`settings_api_test.go:423-445`).
- Hueco 4: prueba de líneas en blanco en literales `|` (`layout_test.go:71-114`).
- Hueco 9: `editorHref` con `encodeURIComponent` y prueba con `#`, `?`, `&`.

## Cumplimiento técnico

Fuentes consultadas: `openspec/config.yaml` (reglas de verify), skills del registro por trigger (Go, JS), contexto técnico: no aplicable.

- CRITICAL · openspec/config.yaml (comprobación obligatoria de residuos de IDs) · `TestRepositoryLegacyIDResidues` · `openspec/changes/ajustes-proyecto/tasks.md:124` · la línea 8.3 cita el token heredado sin residuo declarado; la CI fallará. Corrección: reescribir la línea sin el token (o declarar el residuo).
- Sin otros hallazgos. SUGGESTION: `coverage-review.design.md` huecos 1, 3, 5-8, 10, 11 quedan informativos.

**Estado:** no limpio — `TestRepositoryLegacyIDResidues` falla por el token heredado en `tasks.md:124`

## Corrección posterior (2026-11-26)

El orquestador corrigió el CRITICAL: retiró el token heredado de la línea 8.3 de `tasks.md` y de las citas de este informe. Volvió a ejecutar la comprobación que fallaba: `go test ./internal/idmigration -run Residue` → `ok`. No se ha tocado código. Las demás comprobaciones de este informe siguen como se registraron arriba.

**Estado:** limpio
