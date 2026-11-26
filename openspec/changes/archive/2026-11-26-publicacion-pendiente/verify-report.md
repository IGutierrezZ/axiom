# Informe de verificación: publicacion-pendiente (C-05-c)

Rol único `fullstack`. Alcance: `git diff 3c6d93a2..HEAD` en la rama `inc/publicacion-pendiente` (7 commits, 41 ficheros). Verificación sin editar código y solo con pruebas localizadas.

## Cumplimiento técnico

### Comandos ejecutados (desde la raíz del worktree)

- `go test -cover ./internal/docpaths/...`: ok, 98,9 %.
- `go test -cover ./internal/sddpublish/...`: ok, 86,0 % (incluye `isolation_test.go` y `publication_parity_test.go`).
- `go test -cover ./internal/sddstatus/...`: ok, 89,6 %.
- `go test -cover ./internal/branch/gitcli/...`: ok, 89,5 %.
- `go test -cover ./internal/branch/`: ok, 93,8 %.
- `go test -cover ./internal/branch/engine/ -run 'Publication|Promotion|TheGitPort'`: ok; el 46,0 % es del paquete entero con filtro; `PublicationState` 88,9 %, `resolvePublication` 83,0 % (perfil con `-run PublicationState`).
- `go test ./internal/assets/`: ok.
- `go test ./internal/components -run TestGolden`: ok.
- `go test ./internal/cli -run 'TestPublishIsolation|SDDStatus|GatePublish|SDDPublish'`: ok.
- `go vet ./internal/docpaths/... ./internal/branch/... ./internal/sddpublish/... ./internal/sddstatus/... ./internal/assets/... ./internal/components/... ./internal/cli`: sin salida, código 0.
- `gofmt -l` sobre los paquetes tocados: sin ficheros.
- No se compiló ningún binario ni se ejecutó la suite completa.

### Invariantes pedidos

- `PublicationTarget` sin cambio de semántica: `promotion_contracts_test.go` y `publication_test.go` del motor no tienen diff y pasan; `PublicationTarget` es ahora un envoltorio de `resolvePublication` con las mismas negativas y orden (`internal/branch/engine/publication.go:30-76`).
- `internal/sddpublish` solo se importa en producción desde `internal/cli/sdd_publish.go` (grep de `v3/internal/sddpublish"`: único fichero).
- JSON sin estado sin cambios: puntero con `omitempty`; `internal/sddstatus/publication_test.go:346` comprueba que la clave no aparece tras `publish`.

### REQ-27.18 (evidencia por prueba, localizada por nombre)

- Pendiente con documentación sucia, invocación idéntica a la de la compuerta, clave en la raíz y no en `governance`, `nextRecommended` y dependencias iguales: `publication_test.go:280` (`TestStatusProjectsPendingPublicationWithTheRealEngine`, aserciones en 305-354).
- Varios motivos, `branch_ahead`, `remote_branch_missing`, ausencia sin nada que subir: `TestProjectPublication` (`publication_test.go:182`) y `TestPublicationStateUnpushedCommitsAndCodeCommits` (`publication_state_test.go:100`).
- Sin `origin`: `TestPublicationStateWithoutOriginIsRemoteMissing` (`publication_state_test.go:140`).
- `code_unpublished` con `pushCommand` y `codeCommits`, sin `invocation`: `publication_test.go:359`, `:248`.
- Más de 500 commits: `publication_state_test.go:219` y fila de `publication_test.go:208`.
- `unavailable` (negativa, `git_failed`, rango desconocido): `publication_test.go:155`, `publication_state_test.go:244,289,305`.
- Exclusiones (engram/none, solo `integration`, `continuous` sin `auto`, ventana de apply/verify, archivado `checkpointed`, sin kickoff): `TestPublicationDue` (`publication_test.go:68`, filas 106-110).
- Archivado `continuous` con `auto` con `--archive`: `publication_test.go:106,232`; rama de archivado en motor: `publication_state_test.go:198`.
- Multirepo: `publication_state_test.go:170`.
- Carpeta sin seguimiento con solo `.gates.lock` frente a `kickoff.yaml`: `publication_state_test.go:53` y `ops_publication_test.go` de `gitcli`.
- Paridad con `publish` y no regresión: `internal/sddpublish/publication_parity_test.go:27,53,70`.
- Sin red ni escrituras: `publication_state_test.go:259` (observador) y `internal/cli/sdd_publish_isolation_test.go` (status no ejecuta push, commit ni fetch; `gate record` sigue sin git).

### REQ-27.19

- Doctrina en `sdd-orchestrator-sections.md` y `sdd-status-contract.md`, fijada por pin en `internal/assets` (en verde) y 10 goldens regenerados (`TestGolden` en verde, diff de una línea por golden).

### Conformidad con la design

- Capas, nombres de ficheros y firmas coinciden con la tabla de File Changes. `publication_state.go` sigue la regla de exclusiones del paso 5 de `publish` y la precedencia de estados.
- Dependencias sin ciclos.

### Tareas

- 1.1 a 6.2 marcadas `[x]`; apply-progress (Engram #2637) reporta U1 a U5 confirmadas, una por commit. Ninguna tarea pendiente observada.

## Hallazgos

### CRITICAL

Ninguno.

### WARNING

1. Coincide con el hueco 1 de `coverage-review.design.md`: la condición (a) basta con una compuerta aprobada o un artefacto completo, así que con `spec` aprobada y `design.md` sucio en iteración se proyecta `pending` y la doctrina publicaría un design en curso. Es conforme a la spec literal (`spec.md` REQ-27.18, condición (a)), pero es un defecto de producto latente; no hay prueba que lo acote (`internal/sddstatus/publication.go`, `publicationDue`).
2. Coincide con el hueco 6 de la design: `CommitFiles` no lista rutas de commits de fusión (`publication_state.go`, bucle de `CodeCommits`), de modo que una fusión con conflictos resueltos sobre código puede dar `pending` donde `publish` se negaría. Límite aceptado en la design; la negativa es informativa.

### SUGGESTION

1. La Open Question de la design (el texto del status sin `--json` no muestra `publication`) queda sin decisión; no se observó ninguna prueba que lo fije.
2. La cobertura del paquete `internal/branch/engine` completo no se midió (el paquete tarda unos 400 s); solo con filtro. `PublicationTarget` aparece al 0 % en el perfil filtrado a `PublicationState`, pero lo cubren las pruebas de promoción, que pasan.
3. Huecos de los informes de cobertura (p. ej. diseño 4, 5, 8 y spec 4, 5) sin escenario propio; no se manifiestan como fallo en las pruebas ejecutadas.

## Limitaciones

- Escenarios mapeados por nombre y aserciones principales de prueba; no se leyó cada prueba línea a línea.
- No se ejecutó la suite completa de `internal/cli` ni de `internal/branch/engine` (por norma del proyecto); la pasa la CI.

**Estado:** limpio
