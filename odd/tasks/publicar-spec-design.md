# ODD: Publicar también tras spec y design en `continuous` + `auto` (publicar-spec-design)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/publicar-spec-design.md`. Espejo Engram: `odd/publicar-spec-design/tasks`.
> Rama: `inc/publicar-spec-design` · worktree `C:\repos\axiom-wt\publicar-spec-design\axiom`. Cambio C-05-b del plan maestro (`docs/plan/backlog-2026-11/05-conocimiento-equipo.md`), ola 7.

## Objetivo

En SDD `continuous` con `docs_publication: auto`, publicar los documentos del cambio también al terminar `spec` y al terminar `design`, además de al cerrar `tasks` y al archivar.

## Problema y porqué

Hoy, con la publicación autorizada, spec y design no llegan al equipo hasta que cierra `tasks`. Decisión del usuario: la documentación tiene que quedar subida en cada parada aprobada y, en SDD sin paradas con `docs_publication: auto`, «subir también tras spec y design».

## Decisiones

- La autorización sellada no cambia: solo publica documentos (REQ-27.4). Los kickoffs ya sellados con `auto` heredan los momentos nuevos sin volver a preguntar.
- `continuous` sigue sin paradas y una negativa de `publish` sigue siendo informativa (REQ-27.13).
- Fuera de alcance: `checkpointed` (ya publica en cada aprobación, REQ-27.12), ODD y los kickoffs `continuous` sin `docs_publication`.
- El texto del pre-vuelo vive hoy en `sdd-orchestrator-sections.md:68` (la opción 3 del estilo de ejecución); `session_preflight.go` y `sdd_preflight_hook.go` ya no lo contienen, así que no se tocan.

## Alcance autorizado

`internal/assets/skills/_shared/sdd-orchestrator-sections.md` (opción del estilo y «When to publish»), `sdd-status-contract.md`, el comentario de `internal/kickoff/types.go`, `internal/assets/assets_test.go`, goldens de `testdata/golden/`, `docs/manual-de-inicio.md`, una prueba en `internal/branch/engine/publication_test.go` y las specs `sdd-docs-publication` (REQ-27.13) y `sdd-lane-kickoff` (REQ-27.4).

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y no hay elección del usuario. Comprobaciones: `go test` de los paquetes tocados con `-run` (norma del usuario: nada de baterías completas en local; la suite entera, en la CI).

## Entrega

`single-pr`: una PR con un commit por tarea. Etiqueta `type:feature`. Previsión: unas 150 líneas.

## Tareas

- [x] T1 · Prueba: `PublicationTarget` resuelve la rama del cambio antes de `tasks` (cambio de un rol con solo `proposal.md` y spec). Evidencia: `go test ./internal/branch/engine -run TestPublicationTarget` ok.
- [x] T2 · Doctrina: la opción del pre-vuelo y «When to publish» publican tras `spec`, `design` y `tasks` y al archivar; contrato de status, comentario de `DocsPublicationAuto`, manual, prueba de assets y goldens. Evidencia: `go test ./internal/assets/... ./internal/kickoff/... ./internal/components/...` ok; goldens regenerados con `-update` (2 líneas por fichero).
- [x] T3 · `## Spec viva` (REQ-27.13 y REQ-27.4) integrada con `axiom archive integrate --odd publicar-spec-design`. Evidencia: integración `composed` de ambas capacidades; ninguna prueba cita el texto anterior.

Ruta de todas las tareas: inline (unidades de menos de ~100 líneas; sin unidades independientes que justifiquen escritores en paralelo).

## Criterio de terminado

- REQ-27.13 nombra cuatro momentos: spec, design, tasks y archivo.
- La sección del orquestador ordena `axiom sdd publish --cwd <root> --change <name> --json` al terminar `spec` y `design` con `docsPublication: auto`.
- La opción del pre-vuelo dice que sube los documentos a `origin/<rama>` al cerrar spec, design y tasks y al archivar.
- Goldens regenerados y `go test` de `./internal/assets/...`, `./internal/components/...`, `./internal/kickoff/...` y `./internal/branch/engine` (con `-run`) en verde.

## Progreso

- Worktree creado desde `origin/main` (`814b3ce5`).
- T1 `d3e1dcdb`, T2 `5bc962ce`, T3 en el commit que cierra la feature.

## Siguiente paso

Abrir la PR (`type:feature`), CI en verde, merge commit y `done`.

## Spec viva

### Capacidad: sdd-docs-publication

```openspec
## MODIFIED Requirements

### Requirement: Publicación Automática en el Estilo `continuous` Solo con Autorización Sellada (REQ-27.13)

En un cambio sellado con `execution_style: continuous`, el orquestador DEBE publicar los documentos con `axiom sdd publish`, sin preguntar, exactamente en cuatro momentos: una vez al terminar la fase `spec`, una vez al terminar la fase `design`, una vez al cerrar la fase `tasks` y una vez al archivar (REQ-27.14), y solo si `kickoff.yaml` registra la autorización de publicación del cambio (`docs_publication: auto`, REQ-27.4 de `sdd-lane-kickoff`). Un kickoff sellado con `docs_publication: auto` antes de que existieran los momentos de `spec` y `design` los hereda sin volver a preguntar ni a sellar: la autorización sigue siendo publicar solo documentos. Un kickoff sin ese campo (incluidos los sellados antes de este cambio) NO DEBE publicar nada de forma automática: el orquestador DEBE ofrecer `axiom sdd publish --change <x>` como siguiente paso y esperar la decisión de la persona. En el estilo `continuous` NO DEBE existir ninguna parada adicional por la publicación (REQ-21.7 de `sdd-block-review-gates`). Una negativa de `publish` en este flujo es informativa: el orquestador la presenta y no interrumpe el encadenado de fases.
(Previously: con la autorización sellada se publicaba solo en dos momentos, al cerrar `tasks` y al archivar)

#### Scenario: Con autorización sellada se publica al cerrar `tasks`

- **DADO** un cambio `continuous` con `docs_publication: auto` en su `kickoff.yaml` y la fase `tasks` recién completada
- **CUANDO** el orquestador cierra la fase
- **ENTONCES** ejecuta `axiom sdd publish --change <x>` sin preguntar
- **Y** continúa con la fase siguiente sin presentar ninguna parada

#### Scenario: Con autorización sellada se publica al archivar

- **DADO** un cambio `continuous` con `docs_publication: auto` y el archivado completado
- **CUANDO** el orquestador cierra el archivado
- **ENTONCES** ejecuta `axiom sdd publish --change <x> --archive` sobre la rama de archivado (REQ-27.14)

#### Scenario: Sin el campo no se publica de forma automática

- **DADO** un cambio `continuous` cuyo `kickoff.yaml` no contiene `docs_publication`
- **CUANDO** el orquestador cierra la fase `tasks`
- **ENTONCES** no ejecuta `publish`
- **Y** ofrece `axiom sdd publish --change <x>` como siguiente paso

#### Scenario: Con autorización sellada se publica al terminar `spec` y al terminar `design`

- **DADO** un cambio `continuous` con `docs_publication: auto` y la fase `spec` o la fase `design` recién completada
- **CUANDO** el orquestador cierra la fase
- **ENTONCES** ejecuta `axiom sdd publish --change <x>` sin preguntar
- **Y** continúa con la fase siguiente sin presentar ninguna parada

#### Scenario: Un kickoff sellado antes hereda los momentos nuevos

- **DADO** un cambio `continuous` sellado con `docs_publication: auto` antes de que se publicara tras `spec` y `design`
- **CUANDO** el orquestador cierra la fase `spec`
- **ENTONCES** ejecuta `axiom sdd publish --change <x>` sin preguntar
- **Y** no vuelve a preguntar ni a sellar el kickoff

#### Scenario: No se publica tras `apply`

- **DADO** un cambio `continuous` con `docs_publication: auto`
- **CUANDO** concluye la fase `apply`
- **ENTONCES** el orquestador no ejecuta `publish`

#### Scenario: Una negativa no interrumpe el encadenado

- **DADO** un cambio `continuous` con `docs_publication: auto` y un commit de código sin subir
- **CUANDO** `publish` se niega con `code_commits_unpublished`
- **ENTONCES** el orquestador presenta la negativa con la orden de push
- **Y** encadena la fase siguiente

#### Scenario: Almacén sin ficheros no publica

- **DADO** un cambio `continuous` con `docs_publication: auto` y almacén `engram`
- **CUANDO** el orquestador cierra la fase `tasks`
- **ENTONCES** no ejecuta `publish`

```

### Capacidad: sdd-lane-kickoff

```openspec
## MODIFIED Requirements

### Requirement: `kickoff.yaml` Registra la Autorización de Publicación Automática en el Estilo `continuous` (REQ-27.4)

Cuando la persona elige el estilo `continuous` en el cuestionario de pre-vuelo, la pregunta DEBE nombrar la publicación de documentos en la propia opción: «sin paradas; sube los documentos a `origin/<rama>` al cerrar spec, design y tasks y al archivar», con la rama real del cambio. Si la persona elige esa opción, `axiom sdd kickoff seal` DEBE registrar en `kickoff.yaml` el campo opcional `docs_publication: auto`, dentro de la misma única escritura de sellado y con la semántica de sellado único de REQ-25.36. El campo NO DEBE escribirse con el estilo `checkpointed` ni cuando la persona no autoriza la publicación, y la ausencia del campo significa «no se publica automáticamente». La serialización DEBE omitir el campo cuando está vacío (`omitempty`); la lectura de `kickoff.yaml` DEBE tolerar su ausencia y un binario anterior DEBE ignorarlo sin error. Sellar con `continuous` sin esa autorización DEBE seguir siendo posible. El campo NO modifica el estilo ni las compuertas y NO autoriza ninguna operación distinta de la publicación de documentos de REQ-27.13 de `sdd-docs-publication`.
(Previously: la opción decía que subía los documentos al cerrar `tasks` y al archivar)

#### Scenario: La opción `continuous` nombra remoto y rama

- **DADO** el cuestionario de pre-vuelo de un cambio cuya rama es `inc/x`
- **CUANDO** se presenta la opción `continuous`
- **ENTONCES** su texto indica que sube los documentos a `origin/inc/x` al cerrar spec, design y tasks y al archivar

#### Scenario: Autorizar la publicación sella el campo

- **DADO** una persona que elige `continuous` con la publicación de documentos autorizada
- **CUANDO** se ejecuta `axiom sdd kickoff seal` con esa configuración
- **ENTONCES** `kickoff.yaml` contiene `docs_publication: auto`

#### Scenario: Sin autorización no se escribe el campo

- **DADO** una persona que elige `continuous` sin autorizar la publicación
- **CUANDO** se sella el cambio
- **ENTONCES** `kickoff.yaml` no contiene `docs_publication`

#### Scenario: El estilo `checkpointed` no escribe el campo

- **DADO** una persona que elige `checkpointed`
- **CUANDO** se sella el cambio
- **ENTONCES** `kickoff.yaml` no contiene `docs_publication`

#### Scenario: Un kickoff sin el campo se lee sin error

- **DADO** un `kickoff.yaml` sellado antes de este cambio, sin `docs_publication`
- **CUANDO** se lee y se ejecutan `status` y `continue`
- **ENTONCES** no hay error y el cambio se comporta como no autorizado para la publicación automática

#### Scenario: Un binario anterior ignora el campo

- **DADO** un `kickoff.yaml` con `docs_publication: auto`
- **CUANDO** lo decodifica el esquema previo a este cambio
- **ENTONCES** la decodificación termina sin error

#### Scenario: El sellado único no reescribe el campo

- **DADO** un cambio sellado con `docs_publication: auto`
- **CUANDO** se ejecuta un segundo `kickoff seal` desde la rama del cambio sin el campo
- **ENTONCES** termina con éxito indicando «Kickoff ya estaba sellado»
- **Y** `kickoff.yaml` conserva `docs_publication: auto`

```
