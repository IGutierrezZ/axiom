# ODD: `sdd-propose` lee el `entry` del contexto técnico solo sin exploración (propose-lee-contexto)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/propose-lee-contexto.md`. Espejo Engram: `odd/propose-lee-contexto/tasks`.
> Rama: `inc/propose-lee-contexto` · worktree `C:\repos\axiom-wt\propose-lee-contexto\axiom`. Cambio C-04-c del plan maestro (`docs/plan/backlog-2026-11/04-consumo-agentes.md`).

## Objetivo

Que `sdd-propose` lea el `entry` de cada modo proyectado en `technicalContext` solo cuando no recibió exploración, para que «Affected Areas», «Risks» y «Dependencies» no se escriban a ciegas cuando se salta `sdd-explore`.

## Problema

La exploración es opcional para propose (`internal/assets/claude/sdd-orchestrator-workflow.md:182`, `sdd-propose | exploration (optional)`). El paso 3 de `sdd-propose` («Read Existing Specs», `internal/assets/skills/sdd-propose/SKILL.md:67-73`) solo lee `openspec/specs/`, y la convención (`internal/assets/skills/_shared/project-baseline-convention.md:17`) exime a propose de leer el contexto técnico. Sin exploración, la propuesta no ve el contexto técnico.

## Por qué

La exploración ya lee el `entry`; sin ella, nadie lo hace antes de design. Leerlo siempre duplicaría lo que design carga después y llevaría la propuesta hacia detalles de diseño.

## Decisiones

- «Sí, condicional» (usuario): propose lee el `entry` solo cuando no recibe exploración.
- La profundidad del contexto técnico sigue siendo cosa de design; con exploración, propose no lee nada nuevo; spec y tasks siguen sin leer (análisis, aceptado).
- Descartados: explore obligatorio antes de propose; que propose lea siempre el contexto técnico.
- **Disparador (de la sesión, no cambia el alcance):** «no exploration was received» significa que no hay hallazgos de exploración del orquestador, ni `exploration.md` en la carpeta del cambio, ni la observación `sdd/{change-name}/explore`. Los localizadores salen del status, como en el resto de fases (sección 1 de la convención).
- La spec viva se modifica (REQ-24.29 de `project-technical-context`) en vez de añadir un requisito nuevo, para no asignar un identificador antes de C-03-b (mismo criterio que C-04-b).

## Alcance autorizado

- `internal/assets/skills/sdd-propose/SKILL.md` (paso 3).
- `internal/assets/skills/_shared/project-baseline-convention.md` (sección 2, línea de fases lectoras).
- `internal/assets/assets_test.go` (`TestReadingPhaseSkillsReadTheTechnicalContextEntry`).
- `## Spec viva` de este documento e integración con `axiom archive integrate --odd propose-lee-contexto`.

Fuera de alcance: lectura de `paths` además del `entry`; cambios en spec, tasks o design; la fila de propose del orquestador; goldens (el apartado 4 del README dice que C-04-c no los toca: si un golden falla, se para y se avisa).

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y no hay elección del usuario. Comprobaciones funcionales ordinarias con `go test` troceado por paquetes o `-run` (sin binarios sueltos).

## Tareas

- [x] **T1 · Lectura condicional en propose, convención y test** (ruta: en línea; disparador: 3 ficheros, dos de un párrafo y un test ya entendido, sin investigación pendiente)
  - Paso 3 de `sdd-propose`: párrafo con el disparador «no exploration was received», leer solo el `entry` de cada modo antes de «Affected Areas», «Risks» y «Dependencies», nunca el resto de `paths`; con exploración no lee nada más. Remite a la convención.
  - Convención, sección 2: la excepción condicional de propose; spec y tasks siguen sin leer.
  - Test: subcaso `sdd-propose` con la frase condicional, sin la frase incondicional de las cuatro fases lectoras, y la excepción en la convención.
  - Comprobación: `go test ./internal/assets/...` y `go test ./internal/components/ -run Golden`.
- [x] **T2 · Spec viva** (ruta: en línea)
  - MODIFIED REQ-24.29 (`project-technical-context`) en `## Spec viva`; integración con `axiom archive integrate --odd propose-lee-contexto` en el commit que cierra la feature.
  - Comprobación: la integración termina sin rechazo y repetirla da `already_integrated`.

## Criterio de terminado

`sdd-propose/SKILL.md` indica leer el `entry` de cada modo proyectado en `technicalContext` solo cuando no se recibió exploración, remitiendo a la convención; la convención recoge la excepción; el test de fases lectoras cubre propose; `go test ./internal/assets/...` en verde.

## Entrega

Una sola PR (`single-pr`) con etiqueta `type:feat`; previsión, unas 100-150 líneas (sin `size:exception`). Push, PR y fusión los decide el usuario.

## Spec viva

### Capacidad: project-technical-context

```openspec
## MODIFIED Requirements

### Requirement: Lectura del Contexto por las Fases Lectoras y Presupuesto de Lectura (REQ-24.29)

Cuando el status proyecta `technicalContext`, las skills `sdd-explore`, `sdd-design`, `sdd-apply` y `sdd-verify` DEBEN leer el fichero `entry` de cada modo declarado y DEBEN consultar el resto de `paths` solo bajo demanda, sin cargarlo por anticipado. `sdd-propose` DEBE leer el fichero `entry` de cada modo declarado solo cuando no recibió exploración (ni hallazgos de exploración del orquestador, ni `exploration.md`, ni la observación `sdd/{change-name}/explore`), y NO DEBE consultar el resto de `paths`; con exploración NO DEBE leer nada del contexto técnico. Las demás fases NO están obligadas a leerlo. Las skills DEBEN obtener los localizadores del status y NO DEBEN interpretar `axiom.yaml`.
(Previously: `sdd-propose` estaba entre las fases no obligadas a leer el contexto técnico, también cuando se saltaba la exploración)

#### Scenario: Las cuatro fases lectoras leen la entrada
- **DADO** los textos de `sdd-explore`, `sdd-design`, `sdd-apply` y `sdd-verify`
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** cada uno indica leer la `entry` de cada modo de `technicalContext` y consultar el resto bajo demanda

#### Scenario: La propuesta lee la entrada solo sin exploración
- **DADO** el texto de `sdd-propose` y la convención del contexto técnico
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** `sdd-propose` indica leer la `entry` de cada modo de `technicalContext` solo cuando no recibió exploración, sin consultar el resto de `paths`
- **Y** la convención recoge esa excepción condicional

#### Scenario: Modo sin entrada
- **DADO** un modo declarado sin `entry`
- **CUANDO** una fase lectora prepara su entrada
- **ENTONCES** no precarga ningún fichero de ese modo y solo lo consulta cuando lo necesita

#### Scenario: Sin contexto declarado
- **DADO** un status sin `technicalContext`
- **CUANDO** se ejecutan las fases
- **ENTONCES** operan como antes de esta capacidad
```

## Progreso

Plan stop: approved 2026-11-18; implementation continues

- T1 · commit `26539131`. Párrafo nuevo al final del paso 3 de `sdd-propose`, válido en cualquier modo de almacén, con el disparador explícito; línea de excepción condicional en la sección 2 de la convención; subcaso `sdd-propose` en `TestReadingPhaseSkillsReadTheTechnicalContextEntry` (frases condicionales, ausencia de la frase incondicional y excepción en la convención). Sin cambios en goldens.
- T2 · commit de cierre `docs(spec)`. `axiom archive integrate --odd propose-lee-contexto`: `project-technical-context: composed`; repetirlo da `already_integrated`.
- RDD: desactivado (`axiom review mode status`: off por defecto); sin revisión nativa.

## Verificación

- `gofmt -l internal/assets/`: sin salida.
- `go test ./internal/assets/...`: ok.
- `go test ./internal/components/ -run Golden`: ok. Ningún golden ni fichero de `testdata` fija el texto tocado.
- `axiom archive integrate --odd propose-lee-contexto`: compuesto; segunda ejecución `already_integrated`.
- Aviso: la integración quita la regla `---` que separaba REQ-24.29 de REQ-24.30 en `openspec/specs/project-technical-context/spec.md`. Es el mismo defecto de composición de un MODIFIED que anotaron C-04-a y C-04-b. No se corrige aquí: fuera de alcance.
