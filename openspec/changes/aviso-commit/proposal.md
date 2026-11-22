# Propuesta: Aviso por commit, parada opcional por tarea y validación implícita al pedir la PR (C-02-d)

## Intención

Con la entrega por defecto en una sola PR (C-02-c), la división del trabajo pasa a ser un commit por tarea. Hoy nada aprovecha esa granularidad: la persona solo ve el resultado al final, en una PR grande, y no puede revisar commit a commit mientras el modelo avanza. Tampoco hay forma de pedir que la sesión espere tras cada tarea, ni de revisar el diff antes de que se haga el commit.

El usuario lo pidió así (2026-11-12): «en el chat se avisará cada vez que se termine una tarea con el commit, pero si no quiere paradas el modelo continuará; si al final solicita el PR a develop se da por validados todos los commits». Las decisiones P1 y P2 (README del backlog, apartado 6.2) y la corrección del 2026-11-20 fijan el comportamiento:

- **Aviso por commit** en los dos carriles: al cerrar cada tarea, el orquestador avisa en el chat con el sha, la tarea y la orden `git show <sha>`.
- **Modo de revisión por commit solo en SDD `checkpointed`**, elegido en la puerta `tasks`, con tres modos: (1) avisar y seguir con commit automático (por defecto); (2) parar tras cada tarea con commit automático; (3) parar tras cada tarea sin commit, de modo que la persona revisa el diff, aprueba y entonces se hace el commit.
- **ODD siempre en modo 1**, sin pregunta ni paradas, tampoco a petición en el chat. `continuous` también va en modo 1 y no pregunta (REQ-21.7).
- **Pedir la PR valida todos los commits pendientes**; no hay aprobación commit a commit y nada bloquea la PR.
- **Recomendación de fusionar sin squash** (merge commit), para que los commits revisados lleguen a la rama destino y `done` confirme por ascendencia (P1). Axiom no fusiona y `done` sigue tolerando squash.

Éxito: la doctrina generada de todos los runtimes avisa por commit y aplica las reglas comunes; `axiom sdd gate record --gate tasks` persiste el modo; `axiom sdd status` lo proyecta; `sdd-apply` respeta el modo, y sin registro el comportamiento es el modo 1.

## Alcance

### Dentro del alcance

1. **Persistencia del modo (opción a del explore):** bandera `--commit-review <modo>` en `axiom sdd gate record`, válida solo con `--gate tasks` y `--decision approved` (con o sin `--hand-off`), que escribe el campo opcional `commit_review` (`omitempty`) en el `GateRecord` de `gates.yaml`. Precedente exacto: `handed_off`. Ausente equivale al modo 1. En un cambio `continuous`, con otra puerta o con `--decision rejected`, la orden se rechaza sin escribir. No se toca el kickoff (inmutable, REQ-25.36) ni el digest de las puertas (es del artefacto, no del registro).
2. **Vocabulario cerrado de tres tokens canónicos**, con nombres provisionales `notify` (modo 1), `stop` (modo 2) y `stop-before-commit` (modo 3). Spec o design fijan los definitivos; cualquier otro valor se rechaza.
3. **Proyección en `axiom sdd status`:** el modo del registro `approved` que cierra la puerta `tasks` aparece en la vista de esa puerta (`gateV2`) y el orquestador lo pasa a `sdd-apply`. Con la salida 2 (`--hand-off`), el modo sobrevive al traspaso y quien recoge `apply` lo lee de `status`.
4. **Interacción propia en la puerta `tasks` (`checkpointed`):** cuando la persona elige una salida de aprobación (2 o 3), el orquestador formula una pregunta separada de la pregunta tipada de tres salidas, que sigue con exactamente tres opciones (REQ-26.2). La aprobación se registra una sola vez con el modo, porque el primer registro `approved` cierra la puerta y un registro posterior se rechaza (REQ-26.3). Con la salida 1 (iterar) no se pregunta.
5. **Sección nueva de doctrina SDD «Commit Review»** en la fuente única `internal/assets/skills/_shared/sdd-orchestrator-sections.md`, referenciada con `{{GENTLE_AI_SDD_SECTION:Commit Review}}` desde los orquestadores de cada runtime y registrada en `sharedOrchestratorSectionNames`. Recoge las reglas comunes:
   - aviso con sha, tarea y `git show <sha>` al cerrar cada tarea;
   - los tres modos y su efecto (en los modos 2 y 3 hay una pausa por tarea que no es una puerta: no usa las tres salidas ni escribe en `gates.yaml`);
   - pedir la PR valida todos los commits pendientes, y no hay aprobación commit a commit ni bloqueo de la PR;
   - solo se registran los commits sobre los que se pidieron cambios;
   - cambios sobre un commit ya superado: `git commit --fixup=<sha>` y autosquash no interactivo antes de abrir la PR si la rama no está publicada (el autosquash se difiere hasta pedir la PR), o commit correctivo nuevo si ya se publicó; nunca se reescribe historia publicada;
   - cómo intercalar el `fixup!` sin perder la tarea en curso: terminar o aparcar primero la tarea en curso;
   - recomendación de fusionar sin squash (P1).
   Además, ajuste de «SDD Change Kickoff and Block Gates» para enlazar la interacción de la puerta `tasks`.
6. **`sdd-apply`:** nuevo parámetro de lanzamiento `commit_review` (como `delivery_strategy`). En `notify` mantiene el lote actual y devuelve el sha de cada commit por tarea. En `stop` y `stop-before-commit` devuelve el control al orquestador tras cada tarea. En `stop-before-commit` deja el cambio sin confirmar hasta la aprobación de la persona. El modo 3 es comportamiento nuevo solo aquí.
7. **ODD, paso 6 (`routing.go`):** una frase de aviso del modo 1 (sha, tarea, `git show <sha>`) al cerrar cada tarea, sin parar, más las reglas comunes de `fixup!`, de validación al pedir la PR y de registro de los commits con cambios pedidos en `odd/tasks/<feature>.md`. REQ-27.17 no cambia y su pin (`routing_test.go:810`, `routing_stops_test.go`) sigue valiendo.
8. **`work-unit-commits`:** las dos copias (`internal/assets/skills/work-unit-commits/SKILL.md` y `skills/work-unit-commits/SKILL.md`, con paridad vigilada) reciben las reglas de `fixup!`/autosquash y la recomendación de merge sin squash.
9. Tests de Go (argumentos, esquema, registro, status), tests de doctrina y paridad, y goldens regenerados con `-update` (sin compilar binarios).

### Fuera del alcance

- Revisión automática de commits y cualquier uso de RDD.
- Bloquear la PR por commits pendientes o introducir una aprobación commit a commit.
- La interfaz de usuario para elegir el modo o validar commits (C-06-f se construirá sobre este contrato).
- Cualquier pregunta, parada o modo en ODD; modificar REQ-27.17.
- Preguntar el modo en el pre-vuelo o en `kickoff seal`; tocar `internal/kickoff/args.go` (`SealArgs`) o `internal/cli/sdd_kickoff.go`; modificar `sdd-lane-kickoff`.
- Cambiar el modo después de aprobar la puerta `tasks` (el registro es append-only y la aprobación es terminal).
- La verificación conjunta (C-02-b) y el puerto de IDs (C-03-a), aunque compartan ficheros.
- Que Axiom fusione: la estrategia de merge la elige la persona o su plataforma.

## Capacidades

### Capacidades nuevas

Ninguna.

### Capacidades modificadas

- `sdd-block-review-gates`: requisito nuevo (siguiente serie libre, REQ-33.x) para el campo `commit_review` del registro de la puerta `tasks`, la bandera de `gate record` con sus rechazos (otra puerta, rechazo, `continuous`, valor fuera de vocabulario), la compatibilidad hacia atrás (`omitempty`, ausente = `notify`), su proyección en `status` y la interacción propia en la puerta `tasks`. Ajustes, si proceden, en REQ-21.10 (la puerta `tasks` añade la interacción del modo tras una salida de aprobación) y aclaraciones de no regresión en REQ-26.2 (la pregunta tipada se queda en tres opciones) y REQ-26.3 (un único registro `approved` lleva tanto la cesión como el modo).
- `sdd-orchestrator-assets`: requisito nuevo para la sección «Commit Review» como fuente única proyectada a todos los runtimes (paridad), con el aviso por commit, los tres modos, la validación implícita al pedir la PR, el registro de los commits con cambios pedidos, `fixup!`/commit correctivo, la recomendación de merge sin squash y el contrato `commit_review` de `sdd-apply`.
- `organic-agent-trigger-rules`: requisito ADDED para el aviso por commit en el paso 6 de ODD (modo 1 fijo, sin pregunta ni paradas) y las reglas comunes de `fixup!` y validación al pedir la PR. REQ-27.17 y REQ-32.4 no se modifican.

`sdd-lane-kickoff` no cambia.

## Enfoque

1. **Contrato en Go, por el precedente de `HandedOff`:** campo `CommitReview` en `kickoff.GateRecord` (`internal/kickoff/types.go`), validación del vocabulario y de su combinación con la puerta y la decisión en el esquema y en los argumentos de `gate record` (`args.go`, solo la parte de `gate record`), escritura desde `internal/cli/sdd_gate.go` (ayuda, registro y JSON de salida), lectura del registro `approved` que cierra la puerta en `machine.go` → `GateState`, y proyección en `internal/sddstatus/status_v2.go` (`gateV2`). El digest no cambia: `sdd_gate_digest.go` queda intacto.
2. **Interacción en dos tomas:** la pregunta tipada de la puerta `tasks` no cambia. Tras una salida de aprobación, el orquestador pregunta el modo con una interacción propia, sujeta al contrato de *Lossless Blocking Prompt*, y registra la aprobación una sola vez con `--commit-review <token>`. Queda para design si `status` proyecta además una pregunta tipada del modo, que sería útil para C-06-f, o si la interacción vive solo en la doctrina (como «Preference» de *SDD Independent Review*), y cómo se completa la invocación de la salida elegida con la bandera sin alterar el resto de sus tokens.
3. **Doctrina de fuente única:** una sección nueva en `sdd-orchestrator-sections.md` con marcadores, y solo el marcador en cada orquestador, de modo que la paridad la garantiza `TestSharedOrchestratorSectionsHaveOneSource`. En ODD, solo prosa del paso 6 (y del 7 si design lo coloca allí), sin vocabulario de paradas.
4. **Ejecución por tarea en `sdd-apply`:** en los modos 2 y 3 el orquestador lanza `apply` con una tarea por lanzamiento, presenta el aviso o el diff y espera. En el modo 3, tras la aprobación de la persona se hace el commit y se pasa a la siguiente tarea. Design decide si confirma el orquestador (Bash de git) o un relanzamiento de `apply`.
5. **Registro de los commits con cambios pedidos:** en ODD, en `odd/tasks/<feature>.md`. En SDD, en un artefacto de seguimiento del rol que design fija (sección del fichero de tareas del rol o `apply-progress`). Como la aprobación de `tasks` es terminal, escribir en el fichero de tareas no reabre la puerta.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/kickoff/types.go` | Modificado | Campo `CommitReview` (`omitempty`) en `GateRecord`; campo en `GateState`. |
| `internal/kickoff/` (esquema, argumentos de `gate record`, `machine.go`) | Modificado | Vocabulario, rechazos y lectura del registro `approved` de `tasks`. `SealArgs` no se toca. |
| `internal/cli/sdd_gate.go` | Modificado | Bandera `--commit-review`, ayuda, escritura y JSON. |
| `internal/sddstatus/status_v2.go` (y `governance*.go` si procede) | Modificado | Proyección del modo en la vista de la puerta `tasks`. |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modificado | Sección nueva «Commit Review» y ajuste de «SDD Change Kickoff and Block Gates». |
| `internal/assets/<agente>/sdd-orchestrator.md` (11) y `internal/assets/claude/sdd-orchestrator-workflow.md` | Modificado | Solo el marcador de la sección nueva. |
| `internal/assets/skills/sdd-apply/SKILL.md` | Modificado | Parámetro `commit_review`, avance tarea a tarea y modo 3. |
| `internal/assets/skills/work-unit-commits/SKILL.md`, `skills/work-unit-commits/SKILL.md` | Modificado | `fixup!`/autosquash, commit correctivo, merge sin squash (en paridad). |
| `internal/components/agentguidance/routing.go` | Modificado | Aviso por commit y reglas comunes en el paso 6 de ODD. |
| Tests (`internal/kickoff/*_test.go`, `internal/cli/sdd_gate_test.go`, `internal/sddstatus/status_v2_test.go`, `routing_test.go`, `orchestrator_shared_sections_test.go`, `orchestrator_no_drift_test.go`, `internal/assets/skill_copies_parity_test.go`) | Modificado | Cobertura del contrato y de la doctrina. |
| `testdata/golden/*` | Modificado | Regeneración con `-update`. |
| `openspec/specs/{sdd-block-review-gates,sdd-orchestrator-assets,organic-agent-trigger-rules}` (vía delta) | Modificado | Requisitos nuevos descritos arriba. |

Paquetes de `internal/` afectados: `kickoff`, `cli`, `sddstatus`, `assets`, `components/agentguidance` (y `components` por los goldens).

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| En SDD modo 1, `sdd-apply` trabaja por lote y el orquestador solo recupera el control al volver el lote: el aviso por tarea llega agrupado al final del lote, no «mientras el modelo sigue». | Alta | Apply devuelve el sha por tarea y el orquestador emite un aviso por commit al volver. Design valora lotes pequeños. Se señala al orquestador por si la persona quiere otro equilibrio, sin reabrir la decisión 6. |
| Completar la invocación de la pregunta tipada con `--commit-review` puede chocar con la disciplina de ejecutar invocaciones emitidas por el proveedor sin alterarlas. | Media | Design fija el mecanismo: `status` emite la bandera o un marcador explícito, o una regla doctrinal acotada. Tests de la pregunta tipada intactos. |
| Con la salida 2, quien aprueba elige el modo para la persona que recoge `apply`. | Media | Queda persistido y visible en `status`. Cambiarlo después queda fuera de alcance. Se señala como pregunta abierta de producto no bloqueante. |
| El autosquash cambia los sha posteriores y puede invalidar avisos ya emitidos. | Media | Diferirlo hasta pedir la PR, `GIT_SEQUENCE_EDITOR=true`, decidir «publicada» por la rama remota que contiene el sha y no reescribir nunca historia publicada. |
| La prosa de ODD puede activar los pines de REQ-27.17 (`three exits`, `Plan stop`). | Baja | Redactar el aviso sin vocabulario de paradas y ejecutar `routing_test.go` y `routing_stops_test.go`. |
| Solape con C-02-b y C-03-a en `routing.go`, las secciones, los orquestadores y `args.go`. | Media | No adelantar su alcance y rebasar sobre `main` antes de la PR. |
| Tamaño M-alto (unas 600-750 líneas sin goldens) en una sola PR. | Media | Estrategia `single-pr` ya fijada; un commit por tarea facilita la revisión. |

## Plan de reversión

El cambio es aditivo y compatible hacia atrás:

- Revertir el merge commit de la PR deja `gate record`, `status`, la doctrina, `sdd-apply` y `work-unit-commits` como estaban. Hay que regenerar los goldens y ejecutar `axiom sync` para reinstalar los recursos anteriores.
- Un `gates.yaml` escrito con `commit_review` sigue siendo legible por el binario anterior: el campo es `omitempty` y una clave desconocida se ignora (el mismo comportamiento verificado para `handed_off`, REQ-26.3), así que la puerta se ve `approved` y el comportamiento cae al modo 1.
- No hay migraciones de datos ni cambios en `kickoff.yaml` que deshacer.

## Dependencias

- C-02-c (entrega en una sola PR, sin el número 400) ya integrado en `main`: este cambio parte de su doctrina y sus skills.
- C-01-c ya retiró la frase de RDD del paso 7 de `routing.go`.
- C-06-f depende de este contrato (bandera, campo y proyección).

## Criterios de éxito

- [ ] La doctrina generada de todos los runtimes indica avisar tras cada commit de tarea con el sha, la tarea y `git show <sha>`, continuar en `notify`, esperar en `stop`, esperar sin commit en `stop-before-commit`, y dar por validados todos los commits al pedir la PR.
- [ ] La doctrina incluye las reglas de registro de los commits con cambios pedidos, `fixup!` con autosquash diferido o commit correctivo según la publicación, cómo intercalar sin perder la tarea en curso, y la recomendación de merge sin squash.
- [ ] `axiom sdd gate record --gate tasks --decision approved --commit-review <modo>` persiste `commit_review` en `gates.yaml`, y se rechaza sin escribir con otra puerta, con `--decision rejected`, en `continuous` o con un valor fuera del vocabulario.
- [ ] `axiom sdd status` proyecta el modo de la puerta `tasks`; sin campo, el modo efectivo es `notify`.
- [ ] La pregunta tipada de la puerta sigue con exactamente tres opciones (REQ-26.2) y el digest de las puertas no cambia.
- [ ] `sdd-apply` documenta el parámetro `commit_review` y devuelve el control tras cada tarea en los modos 2 y 3.
- [ ] La doctrina ODD no contiene ninguna parada ni pregunta del modo, y los pines de REQ-27.17 siguen en verde.
- [ ] `go test ./...` y `go vet ./...` en verde, con los goldens regenerados.
