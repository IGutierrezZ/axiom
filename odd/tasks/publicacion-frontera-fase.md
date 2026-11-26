# publicacion-frontera-fase

Tipo: bug

## Objetivo

Corregir el aviso W1 del verify de C-05-c (`publicacion-pendiente`): `axiom sdd status` proyecta `publication: pending` mientras la fase siguiente a la última aprobación está en curso, y la doctrina (REQ-27.19) haría que el orquestador publicara un artefacto aún no aprobado o a medio escribir.

## Problema y porqué

`publicationDue` (`internal/sddstatus/publication.go`) da por cumplida la condición (a) de REQ-27.18:

- en `checkpointed`, con cualquier compuerta aprobada distinta de `integration`. Con `spec` aprobada y `design.md` en iteración (compuerta `design` pendiente o rechazada), `publish` confirmaría y subiría un design no aprobado, en contra de «iterar no publica nada» (REQ-27.12);
- en `continuous` con `docs_publication: auto`, con cualquier artefacto de planificación hecho. Si `design` está a medio escribir, se publicaría entre momentos, cuando REQ-27.13 solo publica al terminar cada fase.

Lo detectaron las dos revisiones de cobertura (hueco 1 de `coverage-review.design.md`) y el verify (W1) de C-05-c.

## Alcance

- Condición (a) en `checkpointed`: además de una aprobación distinta de `integration`, la primera compuerta abierta (en el orden `spec → design → tasks → role-apply:<rol> → integration`) no puede tener su fase en curso. Fase en curso: `design` con el artefacto `design` presente (parcial o hecho); `tasks` con el artefacto `tasks` presente; `role-apply:<rol>` con alguna tarea completada o con progreso de apply presente. Sin compuerta abierta o con solo `integration` abierta, la condición se mantiene.
- Condición (a) en `continuous` con `auto`: además de lo actual, ningún artefacto de planificación (`specs`, `design`, `tasks`) puede estar parcial.
- Spec viva: MODIFIED REQ-27.18 (`sdd-docs-publication`), con la condición nueva y escenarios de iteración.
- Contrato del status (`sdd-status-contract.md`): la descripción del momento de publicación.

Fuera de alcance:

- W2 (commits de fusión en `CommitFiles`), límite aceptado.
- Un artefacto hecho que se sigue editando en `continuous` sin quedar parcial: el status no puede distinguirlo de uno terminado.
- La doctrina REQ-27.19 no cambia: solo cambia cuándo el status proyecta.

## Restricciones

- Cortex XDR borra ejecutables Go sueltos: sin `go build -o`, `go run` ni `go install`.
- Pruebas localizadas: `./internal/sddstatus` y `./internal/assets` con `-run`. Nunca la suite entera en local.
- Sin push, fetch ni merge sin el ciclo de PR del repositorio.

## Decisiones

- **Frontera de fase en lugar de rutas por compuerta.** Acotar las rutas sucias a las de artefactos aprobados exigiría cambiar `publish`, que confirma toda la lista cerrada (REQ-27.5). Es más simple y coherente no proyectar mientras la fase siguiente está abierta: la publicación pendiente se resuelve en la aprobación siguiente, que vuelve a cumplir (a).
- **Se reutiliza `firstOpenGate`**, la misma búsqueda que usan el enrutado y los motivos de bloqueo.

## TDD

Modo: desactivado (sin configuración en `axiom.yaml` ni petición). La prueba que reproduce el fallo se escribe primero y se observa en rojo. Ejecutor: `go test -count=1` por paquete.

## Tareas

- [x] T1. Frontera de fase en `publicationDue` con sus pruebas (`TestPublicationDue`). Ruta: inline (cambio localizado en `sddstatus`, menos de 100 líneas).
  - Pruebas: diez casos nuevos en `TestPublicationDue` (design por empezar, en iteración, parcial y rechazado; tasks en iteración; apply sin empezar, en curso y con progreso; solo `integration` abierta; `continuous` con design parcial). RED observado: los siete casos de fase en curso daban `due`.
  - Arreglo: `publicationDue` consulta `firstOpenGate` y la nueva `phaseInProgress` (reconoce `role-apply` con `kickoff.RoleFromGate`); en `continuous`, un artefacto de planificación parcial corta la proyección.
  - Pruebas que fijaban el fallo, ajustadas: `TestStatusProjectsPendingPublicationWithTheRealEngine` (ahora comprueba que con `spec` aprobada y design en iteración no se proyecta, y compara la invocación con `resolveGatePublication`, porque `pending` ya no coincide con una pregunta de compuerta abierta), `TestStatusProjectsCodeUnpublishedWithTheRealEngine` y `TestPublishIsolationNoVerbButPublishPushesCommitsOrFetches` (aprueban las compuertas hasta dejar abierta solo la siguiente fase sin empezar).
  - Evidencia: `go test -count=1 ./internal/sddstatus` → `ok`; `go test -count=1 ./internal/cli -run 'Publication|SDDStatus|GatePublish|SDDPublish|PublishIsolation'` → `ok`; `gofmt -l` y `go vet ./internal/sddstatus` limpios.
- [ ] T2. Spec viva (MODIFIED REQ-27.18), contrato del status y cierre con `axiom archive integrate --odd publicacion-frontera-fase`. Ruta: inline.

## Criterios de aceptación

- Con `spec` aprobada y la compuerta `design` abierta con `design.md` presente, no se proyecta `publication`; con `design.md` ausente, sí.
- Con `tasks` aprobada y alguna tarea completada (compuerta `role-apply` abierta), no se proyecta; sin tareas completadas, sí.
- En `continuous` con `auto`, un artefacto de planificación parcial impide la proyección.
- Las pruebas existentes de `TestPublicationDue` siguen en verde.

## Spec viva

(Se completa en T2.)

## Progreso

- Documento creado. Siguiente: T1.
