# Delta para Publicación Ligera de Documentación SDD (sdd-docs-publication)

> **Cambio:** `publicacion-pendiente` (C-05-c). **Idioma:** castellano de España; identificadores, códigos y tokens en inglés.
> REQ-26.5 (`gate record` no ejecuta git), REQ-27.5, REQ-27.8, REQ-27.11, REQ-27.12 y REQ-27.13 no cambian de texto.

## ADDED Requirements

### Requirement: El Status Nativo Proyecta la Publicación Pendiente (REQ-27.18)

`axiom sdd status --json` DEBE proyectar, para el cambio seleccionado, un campo opcional `publication` en la **raíz** del status (junto a `gateQuestion`, no dentro de `governance`), con `omitempty`: cuando no hay estado que proyectar, el JSON del status NO DEBE cambiar ni un byte y la clave `publication` NO DEBE aparecer. El cálculo DEBE ser de solo lectura y sin red: NO DEBE ejecutar `fetch`, `pull`, `ls-remote`, `commit`, `add` ni `push`, y solo DEBE usar referencias locales (`refs/remotes/origin/<rama>`, `refs/remotes/origin/<integración>` y, como último recurso, la rama de integración local). La proyección NO DEBE alterar `nextRecommended`, las dependencias ni los motivos de bloqueo del status.

**Condición (a), según el caso.** No se lee nada de git si no se cumple (a):

- **Cambio activo `checkpointed`:** existe al menos una compuerta aprobada cuya clave no es `integration`.
- **Cambio activo `continuous`:** el kickoff registra `docs_publication: auto`; al menos uno de los artefactos `spec`, `design` o `tasks` está completo; no hay tareas de `apply` completadas; y el informe de `verify` no está hecho. Las dos últimas condiciones excluyen la ventana de `apply` y `verify` (REQ-27.13: no se publica tras `apply`).
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

**Exclusiones.** NO DEBE proyectarse `publication` cuando: no hay cambio seleccionado; el almacén de artefactos es `engram` o `none`; no se cumple (a) (sin aprobaciones, solo la compuerta `integration`, `continuous` sin `docs_publication: auto` o en la ventana de `apply`/`verify`); o el repositorio no tiene el remoto `origin` configurado (no hay publicación posible). La configuración de `origin` se lee con una consulta de solo lectura de la configuración; no se deduce de la ausencia de referencias remotas, porque un remoto añadido y aún no traído sí permite publicar.

**Multirepo.** En el repositorio de specs de un `multirepo`, la rama de publicación es su rama de integración in situ (REQ-27.7) y el adelanto se mide contra `origin/<integración>`.

**Una sola lista de rutas.** La clasificación de una ruta como documentación o código DEBE usar la misma lista cerrada de REQ-27.5 que `publish`, compuesta en un único punto y sin duplicarla. `kickoff.yaml` y `gates.yaml` son documentación; el estado local con punto de la raíz de la carpeta del cambio no lo es. Las carpetas sin seguimiento se evalúan fichero a fichero, no como una sola entrada de directorio. El comportamiento de `axiom sdd publish` NO DEBE cambiar.

#### Scenario: Compuerta aprobada con documentación sin confirmar

- **DADO** un cambio `checkpointed` con la compuerta `spec` aprobada y ficheros de `openspec/changes/<x>/` sin confirmar
- **CUANDO** se ejecuta `axiom sdd status --change <x> --json`
- **ENTONCES** `publication.state` es `pending` con el motivo `docs_uncommitted`
- **Y** su `invocation` es idéntica a `gateQuestion.publication.invocation`
- **Y** la clave está en la raíz del status, no dentro de `governance`
- **Y** `nextRecommended` y las dependencias no cambian

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

---

### Requirement: El Orquestador Resuelve la Publicación Pendiente Antes de la Fase Siguiente (REQ-27.19)

Cuando el status proyecta `publication` con `state: pending`, la doctrina del orquestador DEBE ordenar ejecutar su `invocation` exactamente una vez antes de lanzar la fase siguiente y exactamente una vez al reanudar el cambio; no se repite dentro de la misma transición ni de la misma reanudación. Su negativa es informativa: el orquestador la presenta, no anula ninguna aprobación registrada y no interrumpe el encadenado de fases ni la parada de la compuerta (REQ-27.12, REQ-27.13). Con `state: code_unpublished`, el orquestador DEBE presentar su `pushCommand` a la persona y NO DEBE ejecutarlo ni ejecutar `publish` por ese motivo. Con `state: unavailable`, DEBE presentar su código como información. El campo NUNCA DEBE cambiar `nextRecommended`. La doctrina NO DEBE hacer que `gate record` ejecute git (REQ-26.5, REQ-27.11). Esta regla completa el párrafo «When to publish» del orquestador sin cambiar REQ-27.12 ni REQ-27.13.

#### Scenario: Se ejecuta la invocación una vez antes de la fase siguiente

- **DADO** un status con `publication.state: pending` tras aprobar una compuerta
- **CUANDO** el orquestador va a lanzar la fase siguiente
- **ENTONCES** ejecuta la `invocation` proyectada exactamente una vez antes de lanzarla

#### Scenario: Se ejecuta una vez al reanudar

- **DADO** una sesión reanudada cuyo status proyecta `publication.state: pending`
- **CUANDO** el orquestador retoma el cambio
- **ENTONCES** ejecuta la `invocation` exactamente una vez antes de continuar

#### Scenario: La negativa es informativa

- **DADO** un status con `publication.state: pending` y un `publish` que se niega con un código tipado
- **CUANDO** el orquestador recibe la negativa
- **ENTONCES** la presenta como información sin anular la aprobación registrada
- **Y** continúa la cadena o mantiene la parada según el estilo sellado, sin repetir `publish` en esa transición

#### Scenario: `code_unpublished` presenta el `pushCommand` sin ejecutarlo

- **DADO** un status con `publication.state: code_unpublished`
- **CUANDO** el orquestador va a lanzar la fase siguiente
- **ENTONCES** presenta el `pushCommand` a la persona
- **Y** no ejecuta `git push` ni `axiom sdd publish`
- **Y** continúa con la fase siguiente

#### Scenario: `unavailable` se presenta como información

- **DADO** un status con `publication.state: unavailable` y un código
- **CUANDO** el orquestador va a lanzar la fase siguiente
- **ENTONCES** presenta el código como información y continúa

#### Scenario: El campo nunca cambia el enrutado

- **DADO** un status con `publication` en cualquiera de sus tres estados
- **CUANDO** el orquestador consulta `nextRecommended`
- **ENTONCES** es el mismo que sin el campo

#### Scenario: Sin proyección no se publica por esta regla

- **DADO** un status sin clave `publication`
- **CUANDO** el orquestador va a lanzar la fase siguiente
- **ENTONCES** no ejecuta `publish` por esta regla
