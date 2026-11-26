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
- [x] T2. Spec viva (MODIFIED REQ-27.18), contrato del status y cierre con `axiom archive integrate --odd publicacion-frontera-fase`. Ruta: inline.
  - Spec viva: condición (a) con la frontera de fase en `checkpointed` y el parcial en `continuous`, exclusiones, el primer escenario con la fase `design` sin empezar, y tres escenarios nuevos (fase siguiente en iteración, apply en curso, artefacto parcial en `continuous`).
  - Contrato: `sdd-status-contract.md` describe el momento de publicación con la fase siguiente sin empezar.
  - Evidencia: `axiom archive integrate --odd publicacion-frontera-fase --check` → `sdd-docs-publication: composed`; `go test -count=1 ./internal/components -run TestGolden` y `./internal/assets -run 'Publication|StatusContract|Orchestrator'` → `ok` (los goldens no incluyen el contrato). La integración real va en este commit.

## Criterios de aceptación

- Con `spec` aprobada y la compuerta `design` abierta con `design.md` presente, no se proyecta `publication`; con `design.md` ausente, sí.
- Con `tasks` aprobada y alguna tarea completada (compuerta `role-apply` abierta), no se proyecta; sin tareas completadas, sí.
- En `continuous` con `auto`, un artefacto de planificación parcial impide la proyección.
- Las pruebas existentes de `TestPublicationDue` siguen en verde.

## Spec viva

### Capacidad: sdd-docs-publication

```openspec
## MODIFIED Requirements

### Requirement: El Status Nativo Proyecta la Publicación Pendiente (REQ-27.18)

`axiom sdd status --json` DEBE proyectar, para el cambio seleccionado, un campo opcional `publication` en la **raíz** del status (junto a `gateQuestion`, no dentro de `governance`), con `omitempty`: cuando no hay estado que proyectar, el JSON del status NO DEBE cambiar ni un byte y la clave `publication` NO DEBE aparecer. El cálculo DEBE ser de solo lectura y sin red: NO DEBE ejecutar `fetch`, `pull`, `ls-remote`, `commit`, `add` ni `push`, y solo DEBE usar referencias locales (`refs/remotes/origin/<rama>`, `refs/remotes/origin/<integración>` y, como último recurso, la rama de integración local). La proyección NO DEBE alterar `nextRecommended`, las dependencias ni los motivos de bloqueo del status.

**Condición (a), según el caso.** No se lee nada de git si no se cumple (a):

- **Cambio activo `checkpointed`:** existe al menos una compuerta aprobada cuya clave no es `integration`, y la fase que juzga la primera compuerta abierta (en el orden `spec`, `design`, `tasks`, `role-apply:<rol>`, `integration`) no ha empezado. Ha empezado cuando su artefacto existe, parcial o completo (`design`, `tasks`), o cuando hay alguna tarea completada o progreso de `apply` (`role-apply:<rol>`); `integration` no tiene fase propia. Así, iterar no publica nada (REQ-27.12): mientras se itera una fase, o mientras su compuerta espera decisión, la publicación pendiente espera a la aprobación siguiente.
- **Cambio activo `continuous`:** el kickoff registra `docs_publication: auto`; al menos uno de los artefactos `spec`, `design` o `tasks` está completo y ninguno está a medio escribir (parcial); no hay tareas de `apply` completadas; y el informe de `verify` no está hecho. La condición del parcial excluye una fase en curso, y las dos últimas excluyen la ventana de `apply` y `verify` (REQ-27.13: no se publica tras `apply`).
- **Cambio archivado:** solo si es `continuous` con `docs_publication: auto`; entonces la publicación se calcula sobre la rama de archivado y su `invocation` DEBE incluir `--archive` (REQ-27.14). Un cambio `checkpointed` archivado NO DEBE proyectar nada (el archivado se ofrece a la persona).

**Condición (b).** Con (a) cumplida, `publication` se proyecta solo si hay algo que `publish` haría: documentación sin confirmar (rutas de la lista cerrada de REQ-27.5, con el estado local con punto excluido) o al menos un commit sin subir (mismo rango que REQ-27.8). La ausencia de `refs/remotes/origin/<rama>` (primera publicación) es solo un **motivo** (`remote_branch_missing`) y NO basta por sí sola: sin documentación sin confirmar y sin commits sin subir, `publish` respondería `up_to_date` y NO DEBE proyectarse nada.

**Estados.** El campo lleva `state` con uno de tres valores, `reasons` (cuando procede), `remote`, `branch`, `prTarget` (informativo, vacío en la integración in situ) y `archive` (en el momento de archivado). Se aplica la primera regla que se cumpla:

| Hechos | `state` | `reasons` | Campos propios |
|--------|---------|-----------|----------------|
| Negativa del cálculo del destino de publicación o fallo de lectura de git | `unavailable` | — | código (`git_failed` en una lectura de git, o el código de la negativa) |
| Ni `origin/<integración>` ni la integración local existen | `unavailable` | — | `unpublished_range_unknown` |
| Documentación sin confirmar | `pending` | `docs_uncommitted` y, si procede, `branch_ahead`, `remote_branch_missing` y `code_commits` | `invocation` |
| Más de 500 commits sin subir | `code_unpublished` | `range_too_big` | `pushCommand` |
| Algún commit sin subir toca rutas que no son documentación (código) | `code_unpublished` | `code_commits` y, si procede, `remote_branch_missing` | `pushCommand` y `codeCommits` (SHA cortos) |
| Solo commits de documentación sin subir | `pending` | `branch_ahead` y, si procede, `remote_branch_missing` | `invocation` |
| Nada que confirmar ni subir | (campo ausente) | | |

- `pending`: la `invocation` DEBE ser idéntica, byte a byte, a `gateQuestion.publication.invocation` (REQ-27.12) cuando ambas aparecen; en el momento de archivado es esa invocación más `--archive`.
- `code_unpublished` (estado informativo, decisión de producto D1): `publish` se negaría siempre con `code_commits_unpublished` (REQ-27.8). Lleva `pushCommand` (`git -C <raíz> push origin <rama>:<rama>`, con las rutas citadas) solo para la persona y NO lleva `invocation`. Nadie DEBE ejecutarlo salvo la persona.
- `unavailable`: informativo; lleva el código y nunca hace fallar el status.
- Con documentación sin confirmar y commits de código a la vez, el estado es `pending` (con el motivo `code_commits`) para que `publish` confirme en local; el status siguiente proyecta `code_unpublished`.

**Exclusiones.** NO DEBE proyectarse `publication` cuando: no hay cambio seleccionado; el almacén de artefactos es `engram` o `none`; no se cumple (a) (sin aprobaciones, solo la compuerta `integration`, la fase de la primera compuerta abierta en curso, `continuous` sin `docs_publication: auto`, con un artefacto de planificación parcial o en la ventana de `apply`/`verify`); o el repositorio no tiene el remoto `origin` configurado (no hay publicación posible). La configuración de `origin` se lee con una consulta de solo lectura de la configuración; no se deduce de la ausencia de referencias remotas, porque un remoto añadido y aún no traído sí permite publicar.

**Multirepo.** En el repositorio de specs de un `multirepo`, la rama de publicación es su rama de integración in situ (REQ-27.7) y el adelanto se mide contra `origin/<integración>`.

**Una sola lista de rutas.** La clasificación de una ruta como documentación o código DEBE usar la misma lista cerrada de REQ-27.5 que `publish`, compuesta en un único punto y sin duplicarla. `kickoff.yaml` y `gates.yaml` son documentación; el estado local con punto de la raíz de la carpeta del cambio no lo es. Las carpetas sin seguimiento se evalúan fichero a fichero, no como una sola entrada de directorio. El comportamiento de `axiom sdd publish` NO DEBE cambiar.

#### Scenario: Compuerta aprobada con documentación sin confirmar

- **DADO** un cambio `checkpointed` con la compuerta `spec` aprobada, la fase `design` sin empezar y ficheros de `openspec/changes/<x>/` sin confirmar
- **CUANDO** se ejecuta `axiom sdd status --change <x> --json`
- **ENTONCES** `publication.state` es `pending` con el motivo `docs_uncommitted`
- **Y** su `invocation` es idéntica a la `publication.invocation` de la pregunta de compuerta (REQ-27.12)
- **Y** la clave está en la raíz del status, no dentro de `governance`
- **Y** `nextRecommended` y las dependencias no cambian

#### Scenario: Con la fase siguiente en iteración no se proyecta

- **DADO** un cambio `checkpointed` con la compuerta `spec` aprobada, la compuerta `design` pendiente o rechazada, `design.md` presente y documentación sin confirmar
- **CUANDO** se consulta el status
- **ENTONCES** no hay clave `publication`, porque iterar no publica nada (REQ-27.12)
- **Y** al aprobar `design`, con `tasks` sin empezar, se proyecta `pending`

#### Scenario: Con apply en curso tras aprobar tasks no se proyecta

- **DADO** un cambio `checkpointed` con la compuerta `tasks` aprobada, su compuerta `role-apply:<rol>` abierta y alguna tarea completada o progreso de `apply`
- **CUANDO** se consulta el status
- **ENTONCES** no hay clave `publication`
- **Y** con todas las compuertas aprobadas salvo `integration`, se proyecta si hay algo que publicar

#### Scenario: En `continuous`, un artefacto de planificación a medio escribir no proyecta

- **DADO** un cambio `continuous` con `docs_publication: auto`, el artefacto `spec` completo, el artefacto `design` parcial y documentación sin confirmar
- **CUANDO** se consulta el status
- **ENTONCES** no hay clave `publication`

#### Scenario: Varios motivos a la vez

- **DADO** una compuerta aprobada, documentación sin confirmar, commits sin subir en la rama y sin `refs/remotes/origin/<rama>`
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `pending`
- **Y** `reasons` contiene `docs_uncommitted`, `branch_ahead` y `remote_branch_missing`

#### Scenario: Tras un `publish` correcto deja de proyectarse

- **DADO** el primer escenario y un `axiom sdd publish --change <x>` correcto
- **CUANDO** se vuelve a consultar el status
- **ENTONCES** no hay clave `publication`

#### Scenario: Rama por delante de origin solo con commits de documentación

- **DADO** una compuerta aprobada, ningún fichero de documentación sin confirmar y commits sin subir que solo tocan rutas de documentación
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `pending` con el motivo `branch_ahead`
- **Y** no se proyecta `code_unpublished`

#### Scenario: Primera publicación con algo que subir

- **DADO** una compuerta aprobada, `origin` configurado, ninguna `refs/remotes/origin/<rama>` y al menos un commit de documentación sin subir
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `pending` con los motivos `branch_ahead` y `remote_branch_missing`
- **Y** el status no falla

#### Scenario: Sin referencia remota y sin nada que subir no se proyecta

- **DADO** una compuerta aprobada, ninguna `refs/remotes/origin/<rama>`, sin documentación sin confirmar y sin commits sin subir
- **CUANDO** se consulta el status
- **ENTONCES** no hay clave `publication` (`publish` respondería `up_to_date`)

#### Scenario: Repositorio sin remoto `origin` no proyecta

- **DADO** una compuerta aprobada, documentación sin confirmar y ningún remoto `origin` configurado
- **CUANDO** se consulta el status
- **ENTONCES** no hay clave `publication`

#### Scenario: Commits de código sin subir proyectan un estado informativo aparte

- **DADO** una compuerta `role-apply:<rol>` aprobada, toda la documentación confirmada y commits que tocan código sin subir
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `code_unpublished` con el motivo `code_commits`
- **Y** lleva `pushCommand` (`git -C <raíz> push origin <rama>:<rama>`) y `codeCommits`, y no lleva `invocation`

#### Scenario: Documentación sin confirmar prevalece sobre el código sin subir

- **DADO** una compuerta aprobada, documentación sin confirmar y commits de código sin subir
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `pending` con los motivos `docs_uncommitted` y `code_commits` y con `invocation`
- **Y** tras el `publish` que confirma la documentación, el status proyecta `code_unpublished`

#### Scenario: Más de 500 commits sin subir

- **DADO** una compuerta aprobada, la documentación confirmada y más de 500 commits sin subir
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `code_unpublished` con el motivo `range_too_big` y `pushCommand`

#### Scenario: Una negativa del destino o un fallo de git es `unavailable`

- **DADO** una compuerta aprobada y un cálculo del destino de publicación que se niega, o una lectura de git que falla
- **CUANDO** se consulta el status
- **ENTONCES** el status termina correctamente con `publication.state` `unavailable` y el código (el de la negativa o `git_failed`)

#### Scenario: Rango sin subir desconocido es `unavailable`

- **DADO** una compuerta aprobada, `origin` configurado, y ni `origin/<integración>` ni la integración local existen
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `unavailable` con el código `unpublished_range_unknown`

#### Scenario: Almacén `engram` o `none` no proyecta

- **DADO** un cambio con aprobaciones registradas y almacén `engram` o `none`
- **CUANDO** se consulta el status
- **ENTONCES** no hay clave `publication`

#### Scenario: Sin aprobaciones ni momentos cumplidos no proyecta

- **DADO** un cambio `checkpointed` sin ninguna compuerta aprobada, o un cambio `continuous` sin `docs_publication: auto`, con documentación sin confirmar
- **CUANDO** se consulta el status
- **ENTONCES** no hay clave `publication`

#### Scenario: La compuerta `integration` no cuenta

- **DADO** un cambio `checkpointed` cuya única aprobación es la de `integration` y documentación sin confirmar
- **CUANDO** se consulta el status
- **ENTONCES** no hay clave `publication`

#### Scenario: Un cambio `continuous` con `auto` proyecta al cumplirse un momento

- **DADO** un cambio `continuous` con `docs_publication: auto`, el artefacto `spec` completo, sin tareas completadas ni informe de `verify`, y documentación sin confirmar
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `pending` con la `invocation` de `publish`

#### Scenario: En `continuous`, durante `apply` o `verify` no se proyecta

- **DADO** un cambio `continuous` con `docs_publication: auto`, con `tasks` cerrada, al menos una tarea de `apply` completada o el informe de `verify` hecho, y documentación sin confirmar
- **CUANDO** se consulta el status
- **ENTONCES** no hay clave `publication` (REQ-27.13)

#### Scenario: Archivado `continuous` con `auto` lleva `--archive`

- **DADO** un cambio archivado `continuous` con `docs_publication: auto` y documentación archivada sin confirmar
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `pending` y `archive` es verdadero
- **Y** su `invocation` es la de publicación más `--archive`

#### Scenario: Archivado `checkpointed` no proyecta

- **DADO** un cambio archivado `checkpointed` con documentación archivada sin confirmar
- **CUANDO** se consulta el status
- **ENTONCES** no hay clave `publication`

#### Scenario: Multirepo mide contra la integración del repositorio de specs

- **DADO** un `multirepo` con una compuerta aprobada y el repositorio de specs en su rama de integración con commits de documentación sin subir respecto de `origin/<integración>`
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `pending` y su rama es la de integración del repositorio de specs

#### Scenario: Una carpeta sin seguimiento con solo estado local no cuenta

- **DADO** una compuerta aprobada y la carpeta `openspec/changes/<x>/` sin seguimiento que solo contiene `.gates.lock`
- **CUANDO** se consulta el status
- **ENTONCES** no hay documentación sin confirmar y, sin commits sin subir, no hay clave `publication`

#### Scenario: Una carpeta sin seguimiento con `kickoff.yaml` es documentación

- **DADO** una compuerta aprobada y la carpeta `openspec/changes/<x>/` sin seguimiento que contiene `kickoff.yaml` además de `.gates.lock`
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `pending` con el motivo `docs_uncommitted`
- **Y** `kickoff.yaml` y `gates.yaml` cuentan como documentación mientras el fichero con punto no

#### Scenario: Paridad con `publish`

- **DADO** un cambio con documentación sin confirmar
- **CUANDO** se comparan las rutas que el status considera documentación sin confirmar con las que `axiom sdd publish` confirma
- **ENTONCES** son las mismas, porque ambos usan la misma lista cerrada de REQ-27.5
- **Y** tras un `publish` correcto el status no proyecta nada
- **Y** el comportamiento, las negativas y los resultados de `publish` son los que ya fijan REQ-27.5 a REQ-27.14

#### Scenario: El cálculo no toca la red ni escribe

- **DADO** el observador del punto único de git de la prueba
- **CUANDO** se calcula `publication` en cualquiera de los escenarios anteriores
- **ENTONCES** el observador no registra `fetch`, `pull`, `ls-remote`, `commit`, `add` ni `push`
- **Y** el estado del repositorio no cambia
- **Y** la prueba de aislamiento de `sddpublish` sigue en verde

#### Scenario: `gate record` sigue sin ejecutar git

- **DADO** una aprobación registrada con `axiom sdd gate record`
- **CUANDO** el registro termina
- **ENTONCES** devuelve la orden de publicación sin ejecutarla (REQ-26.5)
- **Y** la publicación pendiente solo se proyecta después en el status
```

## Progreso

- T1 (`d731a3f3`) y T2 (este commit) cerradas, un commit por tarea.
- Comprobaciones localizadas en verde: `./internal/sddstatus`, `./internal/cli` con `-run`, `./internal/assets` con `-run`, `./internal/components -run TestGolden`. La suite entera queda para la CI.
- Siguiente: PR con etiqueta `bug`.
