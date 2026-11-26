# Progreso de apply: plantilla-ramas (C-07-f)

## Tareas completadas

- [x] 1.1 Motor `nametemplate` (paquete puro `internal/branch/nametemplate`). Commit `feat(branch): motor puro de plantillas de nombre de rama` (bba16f6d4).
- [x] 2.1 Esquema y carga de `branch_template` (`GitConfig`, `RepositoryPolicy`, `RoleConfig`, `Declarations.BranchTemplatesErr`, `ValidateBranchTemplate`, `BranchTemplateWarnings`, `ConfigRefusal`), `axiom.example.yaml` y `docs/usage.md`. Commit `feat(workspace): branch_template en git, repositories y roles`.

- [x] 2.2 Tres filas `branch_template` en el catálogo de Ajustes (`git.`, `repositories[].`, `roles[].`). Commit `feat(settings): filas de branch_template en el catálogo`. RED observado (3 pruebas fallan sin las filas); GREEN: `go test ./internal/settings/ -run 'Catalog|Patch|Write'` y `go test ./internal/dashboard/ -run Settings` ok; goldens sin cambios.

- [x] 2.3 Instantánea del kickoff: `GitSnapshotBranch.Template`, `GitSnapshotRepo.PendingBranches`, `GitSnapshot.TemplateValues` y su `Validate()`. Commit `feat(kickoff): la instantánea congela plantilla, pendientes y valores`. RED observado (fallo de compilación); GREEN: `go test ./internal/kickoff/ -run 'GitSnapshot|Snapshot'` ok, `go vet ./internal/kickoff/` ok, sin plantilla no se emite ninguna clave nueva.

## Pendientes

3.1, 3.2, 3.3, 4.1, 4.2, 5.1.

## Desviaciones

- Constantes de variable con prefijo `Var*` (`VarSlug`, etc.) en lugar de `Slug`, que chocaba con la función `Slug` del mismo paquete. Las tareas posteriores deben usar `nametemplate.VarRef`, etc.
- TDD: en 1.1 la implementación y las pruebas se escribieron juntas; en 2.1 se observó RED (fallo de compilación de las pruebas) antes de implementar.
- 2.1: `openspecHostPath` trata `specs_repository` vacío, `.` u `openspec` como entrada `.` (el desplazamiento léxico vacío del repositorio de specs es la propia carpeta maestra); solo un `specs_repository` absoluto cae a `git:`.
- 2.1: la sección de usuario se añadió como `## Plantillas de nombre de rama (branch_template)` en `docs/usage.md` y la lista de claves editables ya cita las tres claves, aunque las filas del catálogo llegan en 2.2 (misma PR).
