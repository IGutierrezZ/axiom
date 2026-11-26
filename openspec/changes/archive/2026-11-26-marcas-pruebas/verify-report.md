# Informe de verificación — marcas-pruebas (C-06-d)

Rol único `fullstack`. Ámbito: diff `63eaaab04..HEAD` (8 commits).

## Tareas
35 casillas `[x]`, 0 pendientes en `tasks.md`.

## Cumplimiento técnico
Comandos ejecutados (solo paquetes tocados, sin suite completa ni binarios), todos con resultado `ok`:
- `go test ./internal/acceptancetests/marks ./internal/docpaths ./internal/sddpublish`: ok.
- `go test ./internal/branch/engine -run TestPublishedChangeFile`: ok.
- `go test ./internal/sddstatus -run 'TestAcceptanceMarks|TestAcceptanceTests|TestTasksGate'`: ok.
- `go test ./internal/gitcommon -run TestArchitecture`: ok.
- `go test ./internal/dashboard -run 'TestMarks|TestAcceptance|Canary|Secret|TestEveryDashboardRouteIsClassified|Workflow|Tab'`: ok.
- `go test ./internal/cli -run 'TestMarksPublisher|TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign|TestSddPublish'`: ok.
- `go test ./internal/components -run TestGolden`, `./internal/assets -run 'Archive|Orchestrator|Sections|Skill'`, `./internal/idmigration -run TestRepositoryLegacyIDResidues`: ok.
- `go vet` de los paquetes tocados y `./cmd/axiom`: sin salida (limpio).

Seguridad (inspección estática, apoyada en pruebas): `marksPreflight` exige POST, `isLocalRequest`, Content-Type JSON, cuerpo acotado y campos conocidos; `change` se valida contra vacío, `.`, `..` y espacios (service_marks.go:170); `Confirm` se comprueba antes de leer git (service_marks.go:468); la UI escapa con `escapeHtml` los datos interpolados en `innerHTML`; el `Push` solo aparece en `sddpublish/only.go` (rutas de publicación). Trinquetes de rechazos con resolución, canario y arquitectura de `gitcommon`: verdes.

## Hallazgos
- CRITICAL: ninguno.
- WARNING: ninguno.
- SUGGESTION: no se ejecutó la suite completa (política local; la cubre CI). La revisión de exportados nuevos sin uso fue manual y parcial.

**Estado:** limpio
