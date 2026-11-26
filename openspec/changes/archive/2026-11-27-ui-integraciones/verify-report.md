# Informe de verificación — ui-integraciones (C-07-h), rol fullstack

## Resumen
Diagnóstico sobre `git diff origin/main...HEAD` (70 ficheros). Tareas: 10 completadas, 0 abiertas (`tasks.md`).

## Comandos ejecutados (todos con -count=1, localizados)
| Comando | Resultado |
|---|---|
| `go test ./internal/dashboard -run '<patrón del encargo>'` | ok (130 s) |
| `go test ./internal/settings` | ok |
| `go test ./internal/integrations -run 'TestOutboxView\|TestImportsDirection\|TestAdapterAssemblyImportRules'` | ok |
| `go test ./internal/integrations/secrets` | ok |
| `go test ./internal/cli -run 'TestIntegrationsGateway\|TestEveryProductionRefusal...\|TestIntegrations\|TestChangeStart'` | ok |
| `go test ./internal/workspace -run 'TestDocs\|Integrations'` | ok |
| `go test ./internal/idmigration -run TestRepositoryLegacyIDResidues` | ok |
| `go vet` (dashboard, settings, integrations/..., cli, cmd/axiom) | exit 0, sin salida |
| `bash scripts/deadcode-ratchet.sh` | sin funciones inalcanzables nuevas (aviso: 1 entrada de la línea base ya alcanzable) |
| `gofmt -l` sobre los .go tocados | sin salida |

## Cumplimiento técnico
- Spec: prov:WEB:1..11, RF-WEB-07/10/18/19 y RF-TOP-21/22 cubiertos por pruebas del paquete dashboard y settings (cabeceras/CSP, guarda Host/Origin, GET /api/integrations sin secretos, endpoints del PAT que responden solo `source`, connect/discover, pantalla y pestaña del cambio, alta con work item, URL segura, canario, ajustes validados por el decodificador estricto).
- Criterio del plan: PAT fuera de respuestas y de axiom.yaml (pruebas `TestIntegrationsAPISecret*`, canario); Origin ajeno rechazado (`TestLocalGuardRejectsForeignHostAndOrigin`, `TestIntegrationsAPIRejectsForeignHostAndOriginWithoutTouchingAnything`); vista del cambio con vínculos, outbox y actores (`TestChangeIntegrations*`, `TestOutboxView`).
- Diseño: desviaciones registradas en apply-progress.md; ninguna contradice la spec.

## Hallazgos
- CRITICAL: ninguno.
- WARNING: ninguno.
- SUGGESTION: ninguna prueba lleva «CSRF» ni «rebinding» en su nombre; se cubren por el rechazo de Origin (CSRF) y de Host (DNS rebinding) ajenos. Nombrarlas lo haría rastreable (`internal/dashboard/security_headers_test.go:161`).
- SUGGESTION: `deadcode-ratchet` indica que la línea base puede ajustarse (`--update`).

## No verificado
- Comprobación visual de la CSP en navegador real (H-05); el arnés no ejecuta JS (invariantes por texto).
- multirepo con git real.

**Estado:** limpio
