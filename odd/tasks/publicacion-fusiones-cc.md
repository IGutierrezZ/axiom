# publicacion-fusiones-cc

Tipo: bug

## Objetivo

Corregir el aviso W2 del verify de C-05-c (`publicacion-pendiente`): el status clasifica los commits de fusión sin subir como «sin código», aunque la fusión haya resuelto conflictos sobre código, y proyecta `publication: pending` donde `axiom sdd publish` se negaría con `code_commits_unpublished`.

## Problema y porqué

`Engine.PublicationState` (`internal/branch/engine/publication_state.go`) lee el rango sin subir con `CommitFiles` (`git log --name-only`), que no lista rutas en los commits de fusión. `publish`, en cambio, lee cada commit con `diff-tree --cc` (`internal/sddpublish/gitport.go`, `CommitPaths`), que en una fusión devuelve las rutas que la propia fusión resolvió. En una fusión con conflictos resueltos sobre código, el status dice `pending` (o nada) y `publish` se niega. REQ-27.18 ya exige la misma clasificación que `publish` («mismo rango que REQ-27.8»); el código no la cumplía y la design de C-05-c lo dejó como límite aceptado.

## Alcance

- Una lectura nueva del puerto `branch.Git`, `MergeResolvedPaths`, con `git diff-tree -r --no-commit-id --name-only -z --no-renames --cc <sha>`, la misma que usa `publish`.
- `PublicationState` la usa solo para los commits de fusión del rango (más de un padre); el resto sigue saliendo de `CommitFiles` en una sola lectura.
- `CommitFiles` no cambia: también lo usa `axiom branch scope`, cuyo comportamiento queda fuera.

Fuera de alcance: `axiom branch scope` y la verificación conjunta.

## Impacto en la spec

Sin impacto en la spec
Motivo: REQ-27.18 ya exige clasificar los commits sin subir con el mismo rango y la misma lista que `publish` (REQ-27.8); el arreglo hace que el código lo cumpla en las fusiones.

## Restricciones

- Cortex XDR borra ejecutables Go sueltos: sin `go build -o`, `go run` ni `go install`.
- Los métodos del puerto git no pueden contener push, pull, lsremote, publish ni upload en el nombre (`TestTheGitPortHasNoPushMethodAndFetchIsTheOnlyRemoteOperation`).
- Pruebas localizadas: `./internal/branch/gitcli` y `./internal/branch/engine` con `-run`. Nunca la suite entera en local.

## Decisiones

- **Lectura aparte solo para fusiones**, en lugar de añadir `--cc` a `CommitFiles`: no altera `axiom branch scope` y mantiene una sola lectura en el caso común (sin fusiones).

## TDD

Modo: desactivado (sin configuración en `axiom.yaml` ni petición). La prueba que reproduce el fallo, con git real, se escribe primero y se observa en rojo. Ejecutor: `go test -count=1` por paquete.

## Tareas

- [x] T1. `MergeResolvedPaths` en el puerto, `gitcli` y el lector del motor; `PublicationState` la usa en las fusiones; prueba con una fusión con conflicto resuelto sobre código. Ruta: inline (menos de 100 líneas en `branch`).
  - Pruebas: `TestPublicationStateAMergeThatResolvesCodeIsCode` (motor, git real: la rama sube su cambio de código, main cambia la misma línea y la fusión resuelta a mano es el único commit sin subir) y `TestMergeResolvedPaths` (`gitcli`: fusión limpia sin rutas, fusión resuelta con `main.go`, SHA con forma de opción rechazado). RED observado: `CodeCommits` vacío con la fusión resuelta.
  - Arreglo: `MergeResolvedPaths` (`diff-tree --cc`, la lectura de `publish`) en `branch.Git`, `gitcli` y `gitReader`; `PublicationState` la usa para los commits con más de un padre. `CommitFiles` y `axiom branch scope` no cambian.
  - Evidencia: `go test -count=1 ./internal/branch/engine -run 'PublicationState|TheGitPort|Scope'` → `ok`; `./internal/branch/gitcli -run 'TestMergeResolvedPaths|TestCommitFiles'` → `ok`; `./internal/sddstatus -run Publication` y `./internal/sddpublish -run 'Parity|Isolation'` → `ok`; `go vet ./internal/branch/...`, `gofmt -l` y `go build ./...` limpios.

## Criterios de aceptación

- Con una fusión sin subir que resolvió un conflicto sobre código, `PublicationState` la cuenta en `CodeCommits`.
- Una fusión limpia sigue sin contar como código.
- El aviso del límite aceptado desaparece del comentario de `PublicationState`.

## Progreso

- T1 cerrada en este commit. Sin `## Spec viva`: el bug no cambia la spec (ver `## Impacto en la spec`); `axiom archive integrate --odd` lo registra sin escribir en `openspec/specs/`.
- Siguiente: PR con etiqueta `bug`.
