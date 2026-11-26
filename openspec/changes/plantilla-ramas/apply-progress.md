# Progreso de apply: plantilla-ramas (C-07-f)

## Tareas completadas

- [x] 1.1 Motor `nametemplate` (paquete puro `internal/branch/nametemplate`). Commit `feat(branch): motor puro de plantillas de nombre de rama`.

## Pendientes

2.1, 2.2, 2.3, 3.1, 3.2, 3.3, 4.1, 4.2, 5.1.

## Desviaciones

- Constantes de variable con prefijo `Var*` (`VarSlug`, etc.) en lugar de `Slug`, que chocaba con la función `Slug` del mismo paquete. Las tareas posteriores deben usar `nametemplate.VarRef`, etc.
- TDD: la implementación y las pruebas se escribieron juntas; no se observó un RED separado.
