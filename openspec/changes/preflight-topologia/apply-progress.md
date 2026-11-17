# Progreso de apply: preflight-topologia (C-01-b)

Modo: estándar (TDD estricto no activo; RED observado igualmente en cada tarea). Estrategia de entrega: `exception-ok` (una PR con `size:exception`). Lote 1: U1 y U2.

## Unidades completadas

| Unidad | Estado | Commit |
|---|---|---|
| U0 | hecha | `1ca2b85c` |
| U1 | hecha | ver commit `feat(workspace): topología gobernante y reglas de roles y almacén en la validación` |

## U1 · Work Unit Evidence

| Evidencia | Valor |
|---|---|
| Prueba focalizada | `go test ./internal/workspace/ -run 'GoverningTopology\|ValidateMonorepoRejects\|ValidateMultirepoKeeps\|LocateLoadsMonorepoRoles\|LegacyArtifactStore\|DeclarationPaths'`: ok |
| Arnés de ejecución | `go test ./internal/workspace/... ./internal/sddstatus/...`: ok (el status no cambia) |
| Límite de reversión | `internal/workspace/{locate,validator}.go` y sus pruebas; U2 a U4 dependen de `GoverningTopology` |

Notas de diseño aplicadas: con contexto técnico mantenido, `storeErrors` emite el error incondicional de REQ-2.6 (que nombra también `technical_context.maintained`) y además el error existente de REQ-24.62; sin contexto mantenido, solo el primero. `ValidateDeclarationPaths` ya no sale pronto cuando no hay `technical_context`.
