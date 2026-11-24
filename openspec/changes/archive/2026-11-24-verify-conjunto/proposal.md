# Propuesta: Verificación conjunta del cambio antes de la PR a la rama principal (C-02-b)

## Intención

Cada rol y cada repositorio de un cambio se verifican por separado (verify por rol, REQ-28.4 de `sdd-independent-review`). Nadie verifica lo que aportan **todos los roles juntos**, y hoy no se podría hacer con rigor:

- La skill `sdd-verify` solo define un diff cuando recibe un rol (`internal/assets/skills/sdd-verify/SKILL.md:41,62`, `references/technical-compliance.md`). Sin rol se limita a «inspeccionar los artefactos y la implementación disponibles», sin alcance definido.
- No hay código nativo que calcule el alcance de un cambio: `internal/branch/gitcli/ops.go:369-380` solo expone `IsAncestor`, sin `merge-base` ni diff. Un alcance calculado en prosa por el agente no se puede fijar con un test (p. ej., que un commit ajeno en develop no entre), y una rama ausente en local se omite en silencio.
- En modo `release`, tras fusionar la rama en develop, `git merge-base develop <rama>` devuelve la punta de la propia rama: el diff calculado de la forma ingenua queda vacío justo cuando la verificación conjunta tiene sentido.

El momento natural es cuando todas las ramas del cambio ya están fusionadas en la integración (develop), antes de la PR a la rama principal. C-02-a dejó la base de cada repositorio según su modo (`RepoPlan.Base()`, `internal/branch/planner.go:169-180`) y la instantánea sellada de ramas y repositorios (`internal/kickoff/gitsnapshot.go:24-70`); C-02-b se apoya en ellas.

**Por qué ahora.** C-02-a está en `main` y C-02-d (revisión de commits en la compuerta `tasks`) también (`internal/cli/sdd_gate_commit_review.go`, `sdd-orchestrator-sections.md:85`). El plan cita la verificación conjunta como el siguiente paso del flujo de promoción (`docs/plan/backlog-2026-11/02-entrega-git.md:84-118`).

**Resultado esperado.** Axiom calcula, de forma nativa y de solo lectura, qué aporta el cambio en cada repositorio de código; `sdd-verify` sin rol verifica exactamente esa unión; y el orquestador **ofrece** esa verificación conjunta cuando todas las ramas están fusionadas en la integración y **deja constancia** de si se hizo o no, sin bloquear nada.

## Desviación del plan (decisión del usuario del 2026-11-23)

El criterio del plan «la compuerta `integration` no se aprueba hasta que cada rama tenga `pr_merged` verificado en su integración y exista un informe de `sdd-verify` sin rol» (`02-entrega-git.md:115`) **se sustituye**. Motivo, decidido por el usuario el 2026-11-23 (opción B, Engram `sdd/verify-conjunto/decision-b`):

- Los verify son opcionales y se fomentan; **nunca bloquean** ni la compuerta `integration` ni el archivado (coherente con REQ-28.3 de `sdd-independent-review`).
- Se mantiene el modelo de evidencia: REQ-21.16 de `sdd-archive-lifecycle` (en `multirepo`, `pr_merged` se verifica solo contra el repositorio de especificaciones; los repositorios de código se acreditan con `deployment`/`attestation`) **no cambia**; no se añade `--repo` a `pr_merged` (diferido en `openspec/specs/planning-home-resolution/spec.md:46`); un `GateRecord` sigue llevando una sola evidencia (`internal/kickoff/types.go:183-207`).
- En su lugar: el verify conjunto se **ofrece** cuando todas las ramas del cambio (de cada rol y repositorio de código) están fusionadas en la integración, cada rol fusiona cuando termina sin esperar a los demás, y se **documenta** si se hizo o no, como información.

## Alcance

### Dentro del alcance

1. **Cálculo nativo del alcance del cambio, de solo lectura.** Un comando nuevo `axiom branch scope --change <c> --json` (contrato versionado propio; nombre final en `design`) que, para cada repositorio de código de la instantánea sellada, devuelve:
   - rama del cambio, base del modo (`main_branch` en `branch`, `integration_branch` en `release`) y rama de integración;
   - punto de partida del alcance, rango de commits (`<inicio>..<punta>`) y lista de ficheros cambiados, excluyendo la carpeta del cambio (`openspec/changes/<cambio>/`);
   - si la rama del cambio está fusionada en la integración (para decidir la oferta), con una sola rama por repositorio (la del cambio que registra `axiom branch ensure`);
   - un resultado tipado por repositorio: `ok`, `empty` (sin commits propios), `unverifiable` con motivo (`branch_missing` cuando la rama no existe en local ni como `origin/<rama>`, porque se borró tras fusionarse o falta un `fetch` autorizado; `base_missing` cuando falta la base del modo; `fork_point_undetermined` cuando no puede fijarse el punto de partida) y nunca una omisión silenciosa.
   - El repositorio de especificaciones in situ (`specs_in_place`) queda exento, como en `done` y `publish`.
   - No ejecuta `fetch`, no crea ramas ni worktrees y no modifica referencias.
2. **Punto de partida estable frente a la fusión.**
   - En `branch`: merge-base de la punta de la rama con `main_branch`, que aún no contiene el cambio.
   - En `release`: el punto de bifurcación de la rama, que no se mueve tras fusionarla en develop. Puede ser el primer padre del merge commit en la integración, el merge-base cuando la rama aún no es ascendiente (fusión squash o sin fusionar), o una base registrada al crear la rama. `design` elige el mecanismo; el requisito es que un commit ajeno llegado a develop después de la bifurcación nunca entra en el alcance, antes y después de fusionar.
3. **`sdd-verify` sin rol consume ese alcance.**
   - Nueva referencia en la skill (p. ej. `references/joint-verification.md`, análoga a `technical-compliance.md`): obtiene el JSON, verifica la unión por repositorio solo con commits (sin árbol de trabajo ni ficheros sin seguimiento: lo fusionado son commits) y lista en el informe, por repositorio, rama, base, inicio, rango y resultado.
   - Un repositorio `unverifiable` se declara en el informe con su motivo y el comando de `fetch` que la persona debe autorizar; el informe deja de ser limpio por ese hueco, pero nunca lo rellena por su cuenta.
4. **Doctrina del orquestador** (`internal/assets/skills/_shared/sdd-orchestrator-sections.md`, inyectada en los 12 orquestadores; goldens regenerados):
   - **Cuándo se ofrece:** en un cambio `checkpointed`, al presentar la compuerta `integration`, si `axiom branch scope` informa de que todas las ramas del cambio están fusionadas en la integración. Si falta alguna, la presentación lo dice («pendiente de fusionar: …») y no ofrece todavía el verify conjunto.
   - **Cómo se documenta:** la presentación de `integration` declara siempre uno de tres estados: hecha (fecha de la última ronda), ofrecida o no verificable (motivo, incluidas las ramas pendientes de fusionar). No se calcula si una ronda sigue al día: se muestra su fecha y decide la persona. Las ramas borradas tras fusionar son un límite documentado: la doctrina recomienda no borrarlas hasta hacer el verify conjunto, y guardar su último SHA queda para una tarea futura. En el informe de verify queda la sección de verificación conjunta.
   - **Que no bloquea:** las dos salidas de `integration` (REQ-26.1) y su evidencia no cambian; aprobar sin verify conjunto sigue siendo válido.
   - **`continuous`:** no existe la compuerta `integration` (`internal/kickoff/machine.go:52-54`) y el estilo exige el rol único `fullstack`, que nunca es `multirepo`. Ahí el verify del rol `fullstack` ya es el verify del cambio (REQ-28.4). No se añade parada, pregunta ni oferta nueva; el resumen final indica que el verify del rol cubre el cambio completo y, si se lanzó, usa el mismo cálculo de alcance.
5. **Especificaciones vivas** (deltas en la fase `spec`): ver *Capacidades*.

### Fuera del alcance

- Bloquear la compuerta `integration` o el archivado por falta de verify conjunto o de evidencia por repositorio (decisión del usuario, opción B).
- Cambiar REQ-21.16, añadir `--repo` a `pr_merged` o permitir varias evidencias por `GateRecord`.
- Verificar el estado integrado de develop (interferencias con otras features: CI de develop, compromiso aceptado el 2026-11-11), abrir la PR a la rama principal y ejecutar el CI del cliente.
- RDD y `axiom review` (sección 01, decisión del usuario).
- `pruebas.md` (lo genera `sdd-tasks`, C-06-c) y las marcas por entorno (C-06-d).
- Verify o juicio en el archivado (no decidido; posible seguimiento: volver a ofrecerlo solo si hay commits nuevos tras el verify conjunto).
- Ejecutar `fetch`: nunca sin autorización explícita; el comando solo informa de que hace falta.
- **ODD:** no tiene roles ni compuerta `integration` ni instantánea de kickoff; su verify sigue siendo el normal y opcional. El comando de alcance no se expone para `--lane odd` en este cambio (seguimiento posible: leer la base del registro local).

## Capacidades

### Capacidades nuevas

- Ninguna.

### Capacidades modificadas

- `change-branch-lifecycle`: **ADDED** un requisito para el comando de alcance del cambio por repositorio (solo lectura, base por modo, punto de partida estable en `release`, ficheros sin la carpeta del cambio, una rama por repositorio, resultados tipados `ok`/`empty`/`unverifiable` con `branch_missing`/`base_missing`/`fork_point_undetermined`, sin `fetch`; repositorio de specs exento). Numeración sugerida: siguiente libre tras REQ-25.50.
- `sdd-independent-review`:
  - **MODIFIED** REQ-28.1: hoy fija «tres puntos y solo en ellos»; se añade el punto de la verificación conjunta en la presentación de `integration`.
  - **ADDED** un requisito del verify conjunto: alcance = unión por repositorio del comando anterior; solo commits; informe con la sección de verificación conjunta; repositorio no verificable declarado y no limpio.
  - **MODIFIED** REQ-28.3, si `spec` lo considera necesario, para nombrar el verify conjunto entre las revisiones que nunca bloquean.
  - Revisar REQ-28.4, que declara que el verify global multirrol «NO cambia».
- `sdd-block-review-gates`: **ADDED** un requisito: la presentación de la puerta `integration` declara el estado de la verificación conjunta (hecha, ofrecida o no verificable) como información, sin alterar sus dos salidas ni su evidencia. REQ-26.1 no cambia.
- Sin cambios, citadas para dejar constancia: `sdd-archive-lifecycle` (REQ-21.16), `planning-home-resolution` (REQ-24.12), `sdd-role-closure-handoff` (REQ-21.15, salvo que `spec` decida que el verify global consume también el nuevo alcance; ver riesgos).

## Enfoque

Enfoque **B** de la exploración (prosa + comando nativo), sin el bloqueo del enfoque C, que la decisión del usuario descarta.

1. **Go, de solo lectura:**
   - en `internal/branch/gitcli`, primitivas nuevas (`MergeBase`, lista de commits de un rango, ficheros cambiados de un rango, resolución local/remota de una referencia sin red);
   - en `internal/branch/engine`, un `Scope` que recorre los repositorios de la instantánea con `RepoPlan.Base()` y produce el resultado tipado;
   - en `internal/cli`, el subcomando con su contrato JSON.

   Se recomienda **un comando nuevo frente a ampliar `axiom branch status --json`**:
   - `status` es barato, se ejecuta a menudo, tiene contrato estable y tests de dogfooding (`dogfooding_branch_status_test.go`);
   - el alcance recorre rangos y listas de ficheros, tiene resultados tipados propios y solo se pide en un punto del flujo;
   - mezclarlos encarece `status` y acopla dos contratos.

   La decisión final es de `design`.
2. **Skill `sdd-verify`:** referencia nueva para el caso sin rol que se apoya en el comando; la tabla de decisión de `SKILL.md` gana una fila. El informe consolidado (`verify-report.md`, REQ-21.15) lleva la sección de verificación conjunta como contenido adicional, sin tocar el veredicto ni la matriz ISO (REQ-28.4). Ver la pregunta abierta sobre el fichero.
3. **Orquestador:** un párrafo propio en el bloque compartido, junto a «SDD Independent Review» y sin reflujo de la línea 81 de la compuerta, para no reabrir texto que fijan los tests (`internal/assets/assets_test.go`, `independent_review_boundary_test.go`, `internal/components/sdd/orchestrator_shared_sections_test.go`). Goldens de orquestador regenerados con `-update`.
4. **Sin estado de gobierno nuevo:** ni campos en `kickoff.yaml`/`gates.yaml` ni cambios en `axiom sdd status` (REQ-28.1: «la oferta no añade estados»). Si `design` decide añadir un criterio informativo a la pregunta tipada de `integration` (`internal/sddstatus/governance.go:211-214`), debe dejar intactos `nextRecommended` y las dependencias.

## Zonas afectadas

| Zona | Impacto | Descripción |
|---|---|---|
| `internal/branch/gitcli/ops.go` | Modificado | Primitivas de solo lectura: merge-base, rango, ficheros, resolución de referencia sin red |
| `internal/branch/engine/` (nuevo `scope.go`) | Nuevo | Cálculo del alcance por repositorio desde la instantánea y `RepoPlan.Base()` |
| `internal/branch/planner.go` | Lectura | Reutiliza `Base()`/`TwoStage()`; sin cambios previstos |
| `internal/cli/` (subcomando `branch scope`) | Nuevo | Contrato JSON versionado y códigos de salida |
| `internal/assets/skills/sdd-verify/SKILL.md`, `references/joint-verification.md` | Modificado/Nuevo | Verify sin rol con alcance definido; registro en `assets_test.go` |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modificado | Oferta, documentación y no bloqueo de la verificación conjunta |
| `testdata/golden/` (orquestadores) | Modificado | Regeneración por el texto inyectado |
| `openspec/changes/verify-conjunto/specs/{change-branch-lifecycle,sdd-independent-review,sdd-block-review-gates}/` | Nuevo | Deltas de la fase `spec` |
| `docs/` (manual de ramas, si documenta `axiom branch`) | Modificado | Mención del subcomando nuevo, si `design` lo confirma |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|---|---|---|
| Punto de partida mal fijado en `release` tras la fusión (merge commit, squash o *fast-forward*) | Media | Tests con git real para los tres tipos de fusión; `fork_point_undetermined` tipado cuando no es determinable (p. ej. *fast-forward* sin base registrada), nunca un diff vacío presentado como limpio |
| En `branch`, si la persona fusiona develop dentro de la rama del cambio, commits ajenos de develop entran en el rango frente a `main_branch` | Media | `design` decide si los detecta y los excluye o los declara; el informe nunca los mezcla en silencio (**pregunta abierta**) |
| Rama solo en `origin` (hand-off entre máquinas) | Alta en multirepo | `unverifiable: branch_missing`; el mensaje nombra el `fetch` a autorizar; nunca `--fetch` implícito |
| Rama borrada tras fusionarse (borrado automático de la PR o `axiom branch done`) | Alta con ADO | `unverifiable: branch_missing`; límite documentado; la doctrina recomienda no borrar hasta el verify conjunto |
| Texto nuevo del orquestador rompe pines de tests o goldens | Media | Párrafo propio; ejecutar `go test ./...` con regeneración de goldens |
| Confusión entre el verify global (REQ-21.15, al aprobarse los `role-apply`) y el verify conjunto (en `integration`, tras las fusiones) | Media | `spec` fija la relación (**pregunta abierta**); el verify conjunto se presenta como diagnóstico informativo sin estado propio |

### Preguntas abiertas (no resueltas aquí; con evidencia)

1. **¿Dónde escribe el verify conjunto?** Opción a: una sección fechada en el `verify-report.md` consolidado. Opción b: un fichero propio. Es el `verify_file` del rol `fullstack` (`kickoff.yaml`) y el que lleva la matriz ISO que condiciona el archivado (REQ-28.3, REQ-28.4 de `sdd-independent-review`), así que sobrescribirlo podría alterar ese condicionante. Recomendación: sección adicional que no toca el veredicto ni la matriz; lo cierran `spec`/`design`.
2. **Relación con REQ-21.15.** ¿El verify global multirrol, que se habilita al aprobarse los `role-apply`, consume también el nuevo alcance, o el verify conjunto es otra ejecución en la presentación de `integration`? REQ-28.4 (`openspec/specs/sdd-independent-review/spec.md:176`) dice que el global «NO cambia»; usar el mismo alcance exigiría MODIFIED REQ-28.4 y quizá REQ-21.15 (`openspec/specs/sdd-role-closure-handoff/spec.md:97`).
3. **Solo commits.** Esta propuesta asume que el verify conjunto excluye árbol de trabajo y ficheros sin seguimiento, a diferencia del verify por rol (`references/technical-compliance.md`), porque lo fusionado son commits. Asunción a confirmar en `spec`.
4. **Commits ajenos dentro de la rama en modo `branch`** (riesgo de la tabla): ¿se excluyen, se declaran o se acepta el rango tal cual?
5. **Documentar también en `archive-report.md`.** La decisión dice «p. ej. en la presentación de la compuerta `integration` y en el informe». Esta propuesta lo limita a la presentación de `integration` y al informe de verify, y deja fuera `sdd-archive` porque el verify en el archivado está fuera de alcance. Confirmar.
6. **Contradicción con la exploración:** `exploration.md:41-47` da C-02-d (PR #289) por no fusionada, pero `main` ya contiene `--commit-review` (`internal/cli/sdd_gate_commit_review.go`, `internal/kickoff/types.go`, `sdd-orchestrator-sections.md:85`). El riesgo de colisión con C-02-d desaparece; queda el de C-06-c/C-06-d, que comparten la presentación de `integration`.

## Plan de rollback

El cambio es aditivo y sin migración:

- No modifica `kickoff.yaml`, `gates.yaml`, el contrato `gate-record/v1` ni `axiom sdd status`.
- Revertir la PR única (`git revert` del merge) elimina el subcomando, la referencia de la skill y el párrafo del orquestador, y restaura los goldens.
- Los cambios en curso no dependen de estado nuevo: un informe de verify con la sección conjunta sigue siendo un informe válido tras el revert, porque el contenido adicional no altera el veredicto.
- Si solo falla el cálculo del alcance, puede retirarse la oferta del orquestador (un párrafo) y dejar el comando sin consumidores.

## Dependencias

- C-02-a (promoción por repositorio): en `main` (`RepoPlan.Base()`, `GitSnapshotRepo.Promotion`).
- C-02-d: en `main` (ver la pregunta abierta 6).
- git ≥ 2.38 solo si `design` reutiliza la capa de árbol equivalente; el cálculo base no lo necesita.

## Estrategia de entrega

`single-pr` (preferencia de la sesión y memoria del repositorio «una sola PR por cambio»): una PR con un commit de unidad de trabajo por tarea, rama `inc/verify-conjunto` contra `main`. Tamaño estimado M (≈ 600-900 líneas de autor, sin contar goldens).

## Criterios de éxito

- [ ] **Commit ajeno, modo `release`:** con git real, una rama del cambio sale de develop, después llega un commit ajeno a develop y la rama se fusiona en develop con merge commit. `axiom branch scope --json` devuelve para ese repositorio exactamente los commits y ficheros de la rama, sin el commit ajeno ni sus ficheros. Mismo resultado antes de fusionar y con fusión squash.
- [ ] **Commit ajeno, modo `branch`:** rama desde `main_branch`, commit ajeno en develop y rama fusionada en develop. El alcance frente a `main_branch` contiene solo los commits de la rama.
- [ ] **Rama ausente:** la instantánea nombra una rama que no existe en local ni como referencia remota descargada. El repositorio aparece como `unverifiable` con motivo `branch_missing`, sin ejecutar ningún `fetch`, y el resto de repositorios se calcula igual. Un test verifica que no hay acceso a red ni cambios de referencias.
- [ ] **Multirepo con dos repositorios de código y un repositorio de specs in situ:** el resultado tiene una entrada por repositorio de código y el de specs aparece exento.
- [ ] **Solo lectura:** las referencias, el índice y los worktrees son idénticos antes y después del comando (test).
- [ ] **No bloquea:** `axiom sdd gate record --gate integration --decision approved` con su evidencia actual se acepta sin verify conjunto, y `nextRecommended` y las dependencias de `axiom sdd status` no cambian por su presencia o ausencia (tests existentes intactos más uno nuevo).
- [ ] **Doctrina:** los 12 orquestadores contienen el texto de oferta, documentación y no bloqueo; los tests de pines de texto y los goldens pasan tras regenerarlos; en `continuous` no aparece ninguna parada ni pregunta nueva.
- [ ] **Skill:** `sdd-verify` sin rol remite a la referencia nueva, registrada en `assets_test.go`.
- [ ] `go test ./...` en verde y `go vet ./...` limpio.
