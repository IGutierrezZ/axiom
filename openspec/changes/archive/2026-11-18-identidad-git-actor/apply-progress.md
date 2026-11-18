# Progreso de apply: identidad-git-actor (C-07-a)

Modo: estándar (`strict_tdd: false`). Entrega: `exception-ok`, una sola PR con `size:exception`. Rama `inc/identidad-git-actor`.

## Tareas completadas

- [x] U1 (1.1 a 1.6): `internal/gitidentity` y prueba de arquitectura. Commit `6720cf2d`.
- [x] U2 (2.1 a 2.3): `kickoff.FallbackActor`, comentarios de tipo y pruebas de compatibilidad. Commit `dc4f8984`.
- [x] U3 (3.1 a 3.7): adaptador CLI (`sdd_actor_identity.go`), inyección en `gate record` y `kickoff seal`, ayuda, `TestMain` determinista y pruebas. Commit `9c133770`.
- [x] U4 (4.1, 4.2): verificación de cierre. Pendiente para `verify`: `go test ./internal/cli/...` completo (agotó 40 min por un test de `review` colgado en un proceso git, ajeno al cambio).

## Work Unit Evidence

| Unidad | Prueba focalizada | Harness de ejecución | Límite de reversión |
|---|---|---|---|
| U1 | `go test ./internal/gitidentity/... ./internal/gitcommon/...`: ok | `go test` en proceso con repositorios git temporales y entorno aislado | Revertir `6720cf2d` |
| U2 | `go test ./internal/kickoff/...`: ok | `AppendGate`/`LoadGates` y `Seal`/`Load` sobre directorios temporales | Revertir `dc4f8984` (tras U3) |
| U3 | `go test ./internal/cli/ -run "ActorIdentity|SDDGate|KickoffSeal|Seal|Gate|TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign"`: ok (200 s); `bash scripts/deadcode-ratchet.sh`: `no new unreachable functions` | Workspaces temporales reales, multirepo con `workspacetest.Canonical`, y dos pruebas con git real aislado | Revertir `9c133770` |

## Desviaciones del diseño

- Las pruebas de `kickoff` viven en un fichero nuevo `internal/kickoff/actor_identity_test.go` en lugar de ampliar `schema_test.go` (la tarea 2.1 admitía otro fichero; no cambia el comportamiento).
- En U1 el commit pasó los hooks sin rechazo del ratchet; el ratchet se ejecutó de nuevo en el cierre de U3 y pasa.

## Notas

- Tamaño: el código y las pruebas de U1 a U3 suman ≈ 1.200 líneas de autor (pruebas ≈ 70 %), por encima de la previsión de 600 a 750; cubierto por `size:exception` (`exception-ok`).
- La prueba `TestGateRecordRejectedEmitsNoIdentityWarning` con `--hand-off` rechaza antes de resolver la identidad (la guarda de hand-off precede a `defaultActor`); la de evidencia no confirmada sí resuelve (1 llamada) y no avisa.
