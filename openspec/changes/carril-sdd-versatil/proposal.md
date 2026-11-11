# Propuesta: Evidencias ISO y correcciones del SDD (carril-sdd-versatil)

> Fase `propose` (remediación) · rama `inc/carril-sdd-versatil` · 2026-11-05 · sin `explore.md`: la entrada son las decisiones de producto cerradas con el usuario (Engram, `architecture/sdd-carril-versatil/*`), la segunda opinión (`architecture/sdd-carril-versatil/segunda-opinion`) y los huecos de código verificados.
> Kickoff sellado: `checkpointed`, `handoff_policy: none`, rol único `fullstack` (`blocking`).
> Pre-vuelo de la sesión (Engram `sdd/carril-sdd-versatil/preflight`): ritmo automático, artefactos `hybrid`, entrega `single-pr` con `size:exception` aprobada por el mantenedor.
> Actualización 2026-11-10: la rama está al día con `origin/main` tras #235 (`carril-proyecto-publicacion-docs`). Se incorporan Q3 (bloque `sdd:`), Q11b y la baseline implícita.
> Actualización 2026-11-10 (2): #235 archivado en `openspec/specs` (#236). Se añaden P5 (instantánea en `kickoff.yaml`) y P6 (capa first-parent de `axiom branch done`).
> **Actualización 2026-11-11 (recorte).** El usuario recorta el cambio a lo que importa: **evidencia ISO para un cliente real** y dos correcciones del SDD actual (QA diferido y arranque con carpeta). Se cortan `story.md`, el design proporcional y por rol (pieza 9), el archivado en el mismo PR (pieza 10 y P6), la frontera de D5 (pieza 11) y Q7. P5 se reduce a `sdd.iso`; el perfil de bug se reduce a la ruta sin delta. La matriz de verify pasa a exigir **evidencia contrastable** y se integra en el `verify-report.md` que existe tras #249. La rama está en `origin/main` `1b632f03`, con `sdd-revision-independiente` archivado (#251): sus requisitos ocupan REQ-28.1 a REQ-28.14, así que los requisitos nuevos de este cambio pasan a la serie **REQ-29.x**.
> Esta versión **extiende** el SDD actual, no lo sustituye (rechazo de la compuerta `spec` del 2026-11-05, Engram `architecture/sdd-carril-versatil/decisiones-convivencia`). El slug se conserva.

## 0. Resumen

El SDD actual funciona y se conserva entero: el preflight Pace como eje de ritmo (`continuous`/`checkpointed`), las compuertas `spec`, `design`, `tasks`, `role-apply:<rol>` e `integration`, las paradas de tres salidas, el archivado en la rama `<prefijo>/<cambio>-archive`, la doctrina «Missing or failed verification never gates archive» para proyectos no ISO, las revisiones independientes informativas de #249 y los cambios activos en su formato actual. Lo que le falta es la evidencia que pide un auditor ISO para un cliente real: requisitos con ID estable, cada escenario verificado con evidencia contrastable y trazabilidad requisito → tarea → verificación → commit de `main`.

Este cambio **extiende** el SDD con un **modo ISO opcional por proyecto**, declarado en un bloque nuevo de primer nivel `sdd:` de `axiom.yaml` con una sola clave, `iso`. El modo activa: al menos un escenario por requisito, `identity: id` en los documentos de baseline afectados, `Cubre: REQ-x` en las tareas con cobertura validada por Go, una matriz de verify por escenario cuya evidencia automática Go puede contrastar con el árbol, un `verify` que sí bloquea el archivado y la trazabilidad por `git log --first-parent`, compatible con la rama `-archive`. Añade el perfil de bug **sin delta** (`Cubre:` a un requisito existente), necesario porque en un proyecto ISO los bugs van por el SDD (D4). Corrige de paso dos fricciones del SDD actual: el QA `deferred` que bloquea `verify` y el `kickoff seal` que no crea la carpeta del cambio. Resuelve D4 y D6 (esta última vía el modo ISO); D5 queda para su propio cambio.

## 1. Propósito (Intent)

### 1.1 Problema

| Hueco | Evidencia |
|---|---|
| Un rol con `gate_policy: deferred` u `optional` bloquea `verify` igual que uno `blocking`: la condición exige aprobado el `role-apply` de **todos** los roles del roster, sin mirar la política. La barrera de `multi-role-fan-out` ya distingue por política, así que estado y barrera discrepan. | `internal/sddstatus/governance_handoff.go:263-285`; `internal/multirole/barrier.go:124`; REQ-21.15 en `openspec/specs/sdd-role-closure-handoff/spec.md:68-70`; REQ-2.1 en `openspec/specs/multi-role-fan-out/spec.md:69-74` |
| `axiom sdd kickoff seal` rechaza un cambio cuya carpeta no existe y no la crea (fricción observada al arrancar este mismo cambio). | `internal/cli/sdd_kickoff_test.go:239-253` (la prueba fija el rechazo) |
| La composición del archivado no valida escenarios: un requisito puede integrarse sin ningún criterio de aceptación verificable. | `internal/sddstatus/openspec_archive_compose.go` |
| No hay forma de declarar un proyecto como ISO en `axiom.yaml`; la baseline admite perfiles sin ID estable (`identity: text`). | `internal/workspace/declarations.go` (`BaselineDocument`, `RequirementProfile`) |
| `verify-report.md` no tiene cadena escenario → evidencia: la barrera solo lee `verdict:` y la sección «Cumplimiento técnico» de #249 contrasta reglas, no escenarios. | `internal/multirole/barrier.go:12` y `:110-119`; `internal/assets/skills/sdd-verify/references/report-format.md:11` |
| Las tareas no declaran qué requisito cubren; nada comprueba que cada requisito de la delta tenga al menos una tarea. | Ausencia de `Cubre:` en `internal/` |
| La doctrina declara que la verificación nunca bloquea el archivado, y #249 lo repite para las revisiones independientes; un proyecto ISO necesita lo contrario para su evidencia y hoy no puede pedirlo. | `internal/assets/skills/_shared/persistence-contract.md:69`; REQ-28.3 en `openspec/specs/sdd-independent-review/spec.md:112-121` |
| La preparación del cambio exige siempre una delta de spec; un bug que restaura un requisito existente no tiene ruta. | `internal/sddstatus/status.go` (`coreReady`) |
| ODD no integra en la spec viva y no hay vía prevista para trazar un bug contra el requisito que incumple (D4). | Auditoría de visión, G6.5 y D4 |

### 1.2 Por qué ahora

Un cliente real necesita evidencia ISO de los proyectos que se desarrollan con Axiom, y la auditoría de visión (hito P6) dejó pendientes D4 y D6 (Engram `architecture/sdd-carril-versatil/coordinacion-p6`). La segunda opinión mostró que esa evidencia se puede añadir sin retirar nada de lo que hoy funciona. Las demás mejoras que se habían sumado al cambio no aportan evidencia ISO y retrasaban la entrega: se aplazan (§2.2, §3.2).

### 1.3 Resultado esperado

- Un proyecto sin declaración ISO se comporta como hoy, salvo las dos correcciones de §3.1 (piezas 1 y 3) y el perfil de bug sin delta, que es opcional.
- Un proyecto declarado ISO obtiene, sin cambiar de ritmo, de compuertas ni de modo de archivado: al menos un escenario por requisito, IDs de requisito estables en los documentos afectados, cobertura `Cubre:` de cada requisito, una matriz de verify por escenario con evidencia automática contrastable con el árbol y evidencia manual justificada, y un archivado que exige esa matriz válida y con éxito.
- La cadena REQ → tarea → evidencia → commit de integración se reconstruye a partir de los artefactos y de `main` con la rama `-archive`, sea cual sea la estrategia de fusión.
- Los cambios activos en el formato actual siguen avanzando sin migración.

## 2. Decisiones que la propuesta recoge

### 2.1 Cambio de rumbo

La primera propuesta **sustituía** el SDD actual. El usuario la reorientó el 2026-11-05 (`decisiones-convivencia`) a partir de la segunda opinión (`segunda-opinion`), que verificó tres problemas:

1. **Mezclaba ritmo y número de personas.** El preflight Pace ya está implementado: `kickoff seal --from-session-pace` (`internal/kickoff/args.go:190-259`) convierte `interactive` en `checkpointed` y `auto` en `continuous`. La propuesta reutilizaba ese eje para contar personas, con lo que quien trabaja solo perdía las paradas de `spec`, `design` y `tasks`.
2. **Invertía la doctrina de verify sin declararlo.** «Missing or failed verification never gates archive» (`persistence-contract.md:69`) pasaba a «el archivado exige la matriz válida», también en `continuous` y en proyectos sin necesidad ISO.
3. **Retiraba demasiado de golpe.** Desaparecían las compuertas intermedias y la de `integration`, el soporte de cambios activos en formato antiguo y la revisabilidad.

El 2026-11-11 el usuario recorta además el alcance a la evidencia ISO y a las dos correcciones (cabecera). **Se abandona:** el modo «una persona / varias» por estilo sellado; las aprobaciones «Listo» y «Hecho» (Q7 descartada); la retirada de `role-apply:<rol>` e `integration`; la sustitución sin convivencia.

### 2.2 Decisiones vigentes

Se citan por su topic en Engram (proyecto `axiom`). La columna «Estado» indica cómo queda cada decisión.

| ID | Decisión | Estado | Fuente |
|---|---|---|---|
| C1 | La spec viva es la fuente de verdad de los requisitos. Hueco opcional de metadatos externos (referencia externa y REQ afectados) para futuros plugins de Azure DevOps, Jira o Redmine. | Se mantiene | `decisiones` |
| C2 | ~~Una persona = `continuous` sin compuertas.~~ | **Abandonada** | `decisiones-convivencia` |
| C3 | ~~Solo dos aprobaciones, «Listo» y «Hecho».~~ Se conserva únicamente que un rol `deferred` (u `optional`) no bloquee `verify` (pieza 1). | **Abandonada** salvo la corrección del QA diferido | `decisiones-convivencia` |
| C4 | Los criterios de aceptación viven como escenarios de la delta; en ISO, al menos un escenario por requisito en `archive-compose` (Q8). ~~`story.md` como variante de `proposal.md`.~~ | Se mantiene la regla de escenarios; **`story.md` aplazado** (2026-11-11) | `decisiones-funcional`, `decisiones-convivencia` |
| C5 | ~~`design.md` proporcional con mínimo, disparadores y escalado al arquitecto.~~ | **Aplazada** (2026-11-11): no aporta evidencia ISO | `decisiones-design` |
| C6 | `Cubre: REQ-x` en `tasks.md`/`tasks.<rol>.md` y cobertura conjunta validada por Go **en modo ISO**. ~~`design.<rol>.md` y plan propio en `tasks.<rol>.md`.~~ | Se mantiene `Cubre:`; **design por rol y plan propio aplazados** | `decisiones-roles`, `decisiones-convivencia` |
| C7 | Matriz de verify con una fila por escenario: REQ, escenario, tipo (unitario, integración, e2e o manual), referencia, resultado y commit (informativo). En filas automáticas, la referencia resuelve a una prueba existente en el árbol y la fila registra el comando ejecutado y su código de salida; en filas manuales, quién, cuándo y por qué no se automatiza. La matriz es una sección más del `verify-report.md` de #249, sin sustituir «Cumplimiento técnico». **Solo en modo ISO**; fuera de ISO `verify-report.md` no cambia. | Modificada (evidencia contrastable, integración en el informe de #249) | `decisiones-verify`, `decisiones-trazabilidad`, decisión del usuario 2026-11-11 |
| C8 | Trazabilidad independiente de squash, merge o rebase: delta + `integration.yaml`, `Cubre:`, matriz y commit de integración derivado con `git log --first-parent <main>`. Funciona con la rama `-archive`, que sigue siendo el único modo de archivado. Los trailers intermedios son informativos y nunca se validan. **Modo ISO.** | Modificada (con rama `-archive`) | `decisiones-trazabilidad` |
| C9 | ~~Archivado en el mismo PR como opción por proyecto en monorepo.~~ | **Aplazada** (2026-11-11): candidata a un cambio aparte posterior | `decisiones-archivado` |
| C10 | En un proyecto declarado ISO con baseline declarada, todo documento de la baseline afectado por una delta debe tener perfil con `identity: id` (la baseline implícita `openspec/specs/` cuenta como `id`); se comprueba en el estado del cambio, una vez escrita la delta y antes de avanzar a design. `scenarios: omit` se admite. Sin ISO vale cualquier perfil. | Se mantiene | `decisiones-baseline` |
| C11 | Perfil de bug **sin delta**: el informe (observado, esperado, referencia externa) va en `proposal.md`; `Cubre:` apunta al REQ existente que el bug incumple (en ISO, Go comprueba que existe); una fila de evidencia por escenario afectado. Un bug con el requisito mal especificado o ausente va como un cambio normal, con delta. Disponible en todo proyecto. | Reducida (2026-11-11): solo la ruta sin delta | `decisiones-bugs` |
| C12 | ~~Destino del archivado tras una frontera para D5.~~ D5 sigue fuera, en su propio cambio, que diseñará su propia frontera. | **Aplazada** (2026-11-11): abstracción prematura | `decisiones-d5` |
| C13 | ~~Sustituye al SDD actual.~~ **Extiende** el SDD actual; todo convive; sin migración obligatoria. | **Sustituida** | `decisiones-convivencia` |
| C14 | Lo que toque la prosa del orquestador se implementa el último de su cadena. La cadena ya está fusionada (§7), así que la condición queda satisfecha. | **Cumplida** | `orden-ejecucion` |
| C15 | En modo ISO, la matriz de verify válida y con éxito bloquea el archivado, con independencia del ritmo. Es una excepción **declarada** a la doctrina de `persistence-contract.md:69` y a REQ-28.3, que siguen vigentes para proyectos no ISO y para los hallazgos de revisión. | Se mantiene | `decisiones-convivencia` |
| C16 | La corrección del QA `deferred` y el arranque con carpeta son entregables independientes del modo ISO. | Se mantiene | `decisiones-convivencia` |

## 3. Alcance (Scope)

### 3.1 Dentro de alcance (In Scope)

La numeración de piezas se conserva para no romper las referencias de los artefactos anteriores; las piezas 9 a 11 se han cortado.

**Correcciones del SDD actual (independientes del modo ISO)**

1. **QA diferido no bloquea `verify` (C3 residual, C16).** La condición de desbloqueo de `verify` solo exige el `role-apply` aprobado de los roles con política `blocking`; los `deferred` y `optional` no bloquean y se informan, igual que ya hace la barrera (`barrier.go:124`). Modifica REQ-21.15. Coordinación con #249: REQ-28.4 de `sdd-independent-review` dice que el verify global «se evalúa tras todos los `role-apply` aprobados (REQ-21.15)» y que «sigue evaluándose tras aprobar todos los `role-apply`» (`openspec/specs/sdd-independent-review/spec.md:148`); la delta de REQ-28.4 sustituye esa frase por una remisión a la condición de REQ-21.15 («cuando REQ-21.15 habilita el verify global»), sin repetirla, para que no vuelvan a divergir.
3. **Arranque con carpeta (C16) e instantánea de `sdd.iso` (P5 reducida).** `axiom sdd kickoff seal` deja preparada la carpeta del cambio en lugar de exigir que exista (mecanismo en design; hoy lo impide la prueba de `sdd_kickoff_test.go:239-253`, que habrá que invertir) y copia `sdd.iso` del proyecto en `kickoff.yaml` como instantánea: el valor sigue siendo del proyecto y las reglas ISO de ese cambio se leen de su instantánea.

**Modo ISO opcional por proyecto**

2. **Al menos un escenario por requisito (C4).** En un cambio ISO, `archive-compose` rechaza, con motivo tipado que nombra el requisito, un requisito ADDED o MODIFIED sin escenarios; `status` lo adelanta. Fuera de ISO no cambia nada (Q8).
4. **Declaración ISO e `identity: id` (C10).** Nuevo bloque de primer nivel `sdd:` en `axiom.yaml` con una única clave, `iso`, lista cerrada y validación estricta (`unknownKeys`, `internal/workspace/declarations.go:578`, al estilo de `workflowKeys` en `internal/workspace/workflow.go:37`); `workflow:` y `baseline:` no se tocan. Comprobación de `identity: id` en los documentos de baseline afectados, con motivo de bloqueo accionable. Un cambio sin instantánea cuenta como no ISO.
5. **Cobertura `Cubre:` (C6).** Analizador de `Cubre: REQ-x` en `tasks.md`/`tasks.<rol>.md` y validación conjunta: cada REQ de la delta, al menos una tarea (o el REQ existente en un bug sin delta).
6. **Matriz de verify con evidencia contrastable y bloqueo del archivado (C7, C15).** Esquema tabular de la matriz como sección de `verify-report.md` en ISO, junto a «Cumplimiento técnico» y sin escribir líneas `verdict:` (la barrera lee la primera, `report-format.md:11`). Go valida:
   - que todo escenario de la delta (o del REQ existente, en un bug sin delta) tenga fila y que ninguna fila apunte a un escenario inexistente;
   - en filas automáticas (unitario, integración, e2e): que la referencia **resuelva a una prueba existente en el árbol** (forma general `ruta/fichero:símbolo`, con el fichero presente y el símbolo definido en él; en Go basta `TestX` si está definido en un `*_test.go`) y que la fila registre el **comando ejecutado y su código de salida**;
   - en filas manuales: quién, cuándo y por qué no se automatiza;
   - que todo resultado sea de éxito.

   El archivado ISO exige la matriz válida y con éxito. Que la prueba pase sigue siendo **declarado**, pero queda contrastable: la prueba existe y el comando que la ejecuta está registrado para repetirlo. En ISO, una matriz no válida cuenta como comprobación obligatoria fallida, de modo que el verify no es «limpio» en el sentido de #249 (REQ-28.6). La doctrina declara la excepción ISO en `persistence-contract.md` y en REQ-28.3: los hallazgos de revisión siguen siendo informativos; solo la validez de la matriz bloquea.
7. **Trazabilidad (C8).** Derivación con `git log --first-parent <rama de integración>` de dos commits, sin estado nuevo: el de integración del código (el que trae `openspec/changes/<cambio>/` con sus `tasks.md` y `verify-report.md`, contrastable con la evidencia `pr_merged` de la compuerta `integration`, `internal/kickoff/integration.go:9`) y el del archivado (el que trae `openspec/changes/archive/<fecha>-<cambio>/` desde la rama `-archive`). Reutiliza `internal/specbaseline/trace.go` e `internal/specbaseline/ids.go`.

**Perfil de bug sin delta**

8. **Bug sin delta (C11, reducida).** Un cambio declarado como bug puede omitir la delta si `Cubre:` apunta a un REQ existente de la baseline; `coreReady` deja de exigir la delta solo en ese caso y sigue aceptando el formato actual. En ISO, Go comprueba que el REQ existe y la matriz exige una fila por escenario de ese REQ. Un bug con requisito mal especificado o ausente va como cambio normal, con delta. Necesario porque en ISO los bugs van por el SDD (D4).

**Transversal**

12. **D4 en la doctrina.** Una frase en el bloque ODD: ODD nunca integra en la spec viva; en proyectos ISO los bugs van por el SDD con el perfil de bug.
13. **Doctrina, specs, goldens y manual.** Skills `sdd-*` (sobre todo `sdd-verify/references/report-format.md`, `sdd-tasks` y `sdd-archive`), `_shared/sdd-status-contract.md`, `_shared/persistence-contract.md` (excepción ISO), `_shared/sdd-orchestrator-sections.md` donde haga falta; deltas de §4; regeneración de `testdata/golden/` y de las copias versionadas (`AGENTS.md`, `GEMINI.md`, `.codex/AGENTS.md`); `docs/manual-de-inicio.md` y `axiom.example.yaml` con la declaración ISO.

### 3.2 Fuera de alcance (Out of Scope)

- **`story.md` como variante de `proposal.md`** (C4). No aporta evidencia ISO: el informe de un bug cabe en `proposal.md`.
- **Pieza 9 entera** (C5, C6 parcial): design proporcional con mínimo y disparadores, escalado al arquitecto, `design.<rol>.md`, clave `design_triggers` y plan propio en `tasks.<rol>.md`. No aporta evidencia ISO y añadía validación de prosa con riesgo de ruido. Q9 y Q10 dejan de aplicar.
- **Pieza 10 y P6** (C9): archivado en el mismo PR, `axiom sdd publish --archive` en la rama del cambio, excepción a la rama `-archive` y a `change_not_integrated`, y capa first-parent de `axiom branch done`. La rama `-archive` sigue siendo el único modo. Q11 y Q11b dejan de aplicar. Candidato a un cambio aparte posterior.
- **Pieza 11** (C12): frontera del destino del archivado para D5. Abstracción prematura con una sola implementación; la diseñará D5.
- **Q7** («Listo»/«Hecho»): descartada.
- **Claves `archive` y `design_triggers` del bloque `sdd:`**: el bloque queda solo con `iso`.
- **D5**: formatos de spec viva distintos de Markdown por dominio (KVP y otros). Cambio aparte.
- **Plugins externos** (Azure DevOps, Jira, Redmine): solo se reserva el hueco de metadatos.
- **Cualquier retirada** de fases, compuertas, artefactos o del ritmo Pace; cualquier migración de cambios activos o archivados.
- **Multirepo**: sin cambios en su archivado, salvo lo que comparta (matriz, cobertura, perfil de bug).
- **Ejecutar las pruebas desde Go para confirmar el resultado de la matriz**: queda declarado y contrastable, no reejecutado.
- Push, PR o merge desde Axiom (invariante del motor de ramas).
- Intake del repositorio de specs, `axiom init` con `baseline:` o `technical_context:` y demás huecos de P6 (G6.1 a G6.4, G6.6, G6.7).
- Editar `openspec/INDEX.md` a mano (lo regenera `axiom archive sync`).

### 3.3 Qué no cambia

El preflight Pace y el estilo sellado como eje de ritmo; las compuertas `spec`, `design`, `tasks`, `role-apply:<rol>` e `integration` y su registro en `gates.yaml`; las paradas de tres salidas (#218), que desde #235 publican los documentos con `axiom sdd publish`; el archivado en la rama `-archive` con su PR documental y `axiom branch done`; la doctrina «verification never gates archive» en proyectos no ISO; las revisiones independientes de #249 (verify por rol, juicio, cobertura), que siguen siendo informativas; la barrera de `multi-role-fan-out`; los cambios activos y archivados en el formato actual; RDD y su separación de SDD; el carril ODD y `odd/tasks/`, salvo la mención de D4; el bloque `workflow:` y su carril; la persistencia `hybrid`; la traza de integración (`integration.yaml`) y la asignación de IDs, que se reutilizan; el contexto técnico mantenido y `context-impact.md`.

## 4. Capacidades (Capabilities)

> Contrato con `sdd-spec`. Baseline: `openspec/specs/` (este repositorio no declara `baseline:`), que ya incluye #235 (archivado en #236) y `sdd-revision-independiente` (archivado en #251); los MODIFIED se redactan contra ese texto. Los requisitos nuevos usan la serie **REQ-29.x**; la numeración exacta la fija la spec. Las deltas que hay hoy en `specs/` del cambio corresponden a la versión anterior y usan REQ-28.x propios: se renumeran y se recortan como se indica.

### 4.1 Nuevas capacidades (New Capabilities)

- `sdd-iso-evidence`: el modo ISO completo. Bloque `sdd:` con la clave `iso` y su proyección aditiva en el status (absorbe lo que estaba en `spec-baseline-declaration`); `identity: id` en los documentos afectados; cobertura `Cubre:`; matriz de verify con evidencia contrastable dentro de `verify-report.md`; bloqueo del archivado por la matriz (excepción declarada); trazabilidad por `--first-parent` con la rama `-archive`; y la comprobación ISO de que el REQ de un bug sin delta existe. Se conserva la delta actual, renumerada (REQ-28.1 a REQ-28.6 propios → REQ-29.x) y ampliada con la evidencia contrastable.

`sdd-artifact-profiles` **desaparece**: con `story.md` y el design cortados solo le quedaba el perfil de bug sin delta, que es una regla de preparación del cambio (`coreReady`) y encaja en `axiom-sdd-cli-integration` (parte general) y en `sdd-iso-evidence` (comprobación ISO).

### 4.2 Capacidades modificadas (Modified Capabilities)

- `sdd-role-closure-handoff`: REQ-21.15 — `verify` se desbloquea con los `role-apply` de los roles `blocking`; `deferred` y `optional` no bloquean y se informan. La spec revisa si el aviso de cierre de REQ-21.13 necesita ajuste (Q13).
- `sdd-independent-review`: REQ-28.4 — la frase sobre el verify global remite a la condición de REQ-21.15 en lugar de repetir «tras todos los `role-apply` aprobados»; la matriz ISO es una sección adicional del mismo informe, sin líneas `verdict:`. REQ-28.3 — excepción declarada: en un cambio ISO, la validez de la matriz bloquea el archivado; los hallazgos de revisión siguen sin bloquear. Si la spec lo considera necesario, REQ-28.6 aclara que en ISO una matriz no válida impide un verify limpio.
- `sdd-lane-kickoff`: `kickoff seal` prepara la carpeta del cambio; instantánea de `sdd.iso` en `kickoff.yaml` (se conservan las dos deltas actuales, renumeradas, y la de la instantánea se reduce a `iso`).
- `sdd-archive-lifecycle`: en ISO, al menos un escenario por requisito y archivado bloqueado por la matriz. Se conserva solo el requisito de escenarios (renumerado); sobran los MODIFIED de REQ-21.16, REQ-25.39 y REQ-27.14 y el ADDED del archivado en el mismo PR.
- `spec-baseline-integration`: validación de escenarios en la composición (REQ-24.38 y REQ-24.46) y consulta de un REQ existente para el bug sin delta (renumerado). Sobra el MODIFIED de REQ-24.47 (frontera de destino para D5).
- `axiom-sdd-cli-integration`: `coreReady` acepta el bug sin delta sin dejar de aceptar el formato actual, y motivos de bloqueo tipados de ISO, cobertura, escenarios y matriz. Se recortan de la delta actual `story.md` y los motivos de design; sobra el MODIFIED de REQ-25.42 (`wrong_branch` con mismo PR).
- `sdd-orchestrator-assets`: skills `sdd-*` y doctrina con la excepción ISO, el formato de `Cubre:` y de la matriz, y el perfil de bug, en paridad entre runtimes.
- `organic-agent-trigger-rules`: solo la frase de D4 en el bloque ODD.

### 4.3 Capacidades que dejan de modificarse

| Capacidad | Delta actual | Motivo |
|---|---|---|
| `change-branch-lifecycle` | Sobra entera (REQ-25.18, REQ-25.45, REQ-25.16) | Solo existía por el archivado en el mismo PR y la capa first-parent de `branch done` (cortados). La trazabilidad lee `main` con `git log`, sin tocar el motor de ramas. |
| `sdd-block-review-gates` | Sobra entera (REQ-21.9, REQ-26.1) | Los cambios eran los disparadores y el arquitecto en la compuerta `design` y la exención de `integration` con mismo PR (cortados). Las compuertas no cambian. |
| `multi-role-fan-out` | Sobra entera (REQ-1.2, REQ-2.1) | REQ-1.2 era `design.<rol>.md` (cortado). REQ-2.1 ya distingue por política (`openspec/specs/multi-role-fan-out/spec.md:72-74`) y la barrera ya lo implementa (`barrier.go:124`); la corrección de la pieza 1 alinea el estado con ella sin cambiar su requisito. |
| `spec-baseline-declaration` | Sobra entera (bloque `sdd:` y su proyección) | El bloque solo declara `iso`; se especifica en `sdd-iso-evidence`. La frontera de D5 se corta y el bloque `baseline:` no cambia. |
| `sdd-artifact-profiles` | Sobra entera | Ver §4.1. |

## 5. Enfoque (Approach)

1. **Correcciones primero (piezas 1 y 3).** Cada una es una unidad de trabajo con su prueba y su delta: el filtro por `gate_policy` en `verifyDependencyFromRoleGates` (el roster de `Governance` solo lleva nombres, `governance_handoff.go:200-209`; la política está en el kickoff, `governance.go:429`) y la preparación de la carpeta en `kickoff seal`. Van por el SDD porque modifican requisitos de la spec viva y ODD nunca integra en ella (D4).
2. **El modo ISO como una sola llave.** `sdd.iso` en `axiom.yaml`, copiada en `kickoff.yaml` al sellar, activa a la vez las piezas 2 y 4 a 7 y la comprobación ISO de la pieza 8. Sin ella, ningún validador nuevo bloquea.
3. **Validadores en Go, no en prosa.** Escenarios, `Cubre:`, la matriz (incluida la resolución de referencias de prueba contra el árbol) e `identity: id` se analizan en Go y producen motivos de bloqueo tipados en `axiom sdd status` y en `archive-compose`; las skills solo describen el formato.
4. **La matriz se suma al informe de #249.** Es una sección más de `verify-report.md` con su propio esquema; no altera «Cumplimiento técnico», el veredicto que lee la barrera ni el carácter informativo de las revisiones.
5. **Doctrina con excepción explícita.** `persistence-contract.md` y REQ-28.3 mantienen la regla general y declaran la excepción ISO (C15).
6. **Perfil de bug aditivo.** `coreReady` omite la delta solo en un bug sin delta con `Cubre:` a un REQ; el formato actual sigue siendo válido.
7. **Prosa y goldens al final** de la PR; los goldens se regeneran con `-update`, nunca a mano.

## 6. Áreas afectadas (Affected Areas)

| Área | Impacto | Qué cambia |
|---|---|---|
| `internal/sddstatus/governance_handoff.go`, `internal/sddstatus/governance.go` | Modificado | Desbloqueo de `verify` por `gate_policy` (pieza 1) |
| `internal/kickoff/` (`seal.go`, `types.go`) e `internal/cli/sdd_kickoff.go` | Modificado | Arranque con carpeta preparada; instantánea de `sdd.iso` en `kickoff.yaml` (P5 reducida) |
| `internal/sddstatus/openspec_archive_compose.go`, `internal/cli/sdd_archive_compose.go` | Modificado | Regla de escenarios y bloqueo ISO por matriz |
| `internal/sddstatus/` (`status.go`, `status_v2.go`) | Modificado | `coreReady` con bug sin delta; motivos de bloqueo ISO, cobertura, escenarios y matriz; proyección de `sdd.iso` |
| `internal/workspace/declarations.go` y fichero nuevo del bloque `sdd:` | Modificado y nuevo | Bloque `sdd:` con la clave `iso`; `identity: id` en ISO |
| `internal/specbaseline/` (`trace.go`, `ids.go`) | Modificado (reutilizado) | Traza por `--first-parent` con la rama `-archive`; REQ existente en bug sin delta |
| Paquete o ficheros nuevos para `Cubre:`, la matriz y la resolución de referencias de prueba (ubicación en design) | Nuevo | Analizadores y validadores |
| `internal/multirole/barrier.go` | Sin cambio de comportamiento (verificar) | La matriz no escribe `verdict:`; la barrera sigue leyendo la primera línea `verdict:` |
| `internal/assets/skills/sdd-*/` (sobre todo `sdd-verify/references/report-format.md`), `_shared/sdd-status-contract.md`, `_shared/persistence-contract.md`, `_shared/sdd-orchestrator-sections.md` | Modificado | Formato de `Cubre:` y de la matriz, perfil de bug, excepción ISO |
| `internal/components/agentguidance/routing.go` (bloque ODD) | Modificado | Frase de D4 |
| `testdata/golden/`, `AGENTS.md`, `GEMINI.md`, `.codex/AGENTS.md`, `docs/manual-de-inicio.md`, `axiom.example.yaml` | Regenerado o modificado | Copias versionadas, manual y ejemplo con la declaración ISO |
| `openspec/changes/carril-sdd-versatil/specs/` | Rehecho | Se borran las cinco deltas de §4.3; se recortan y renumeran las demás; se añade `sdd-independent-review` |
| `openspec/changes/carril-sdd-versatil/design.md` y `tasks.md` | Rehecho | Corresponden a la versión anterior (design con disparadores, mismo PR y P6) |

Ya no se tocan: `internal/branch/` (`verdict.go`, `archive_refusal.go`, `engine/done.go`, `engine/publication.go`), `internal/sddpublish/` ni `internal/docpaths/`.

## 7. Dependencias y orden de ejecución

- **Cadena de prosa del orquestador (C14): fusionada.** (1) arreglos T9 + adaptadores T1 (#229, #227) → (2) revisión T6 (#233, fusionada el 2026-11-10, `odd/tasks/upstream-revision-nativa.md:291-301`) → (3) guía T1 (#238) → (4) guía T2 (#239) y T3 (#240) (`odd/tasks/guia-orquestador-upstream.md:12-15`). La prosa de este cambio puede aplicarse ya; sigue yendo al final de la PR para regenerar los goldens una sola vez.
- **#235 y #249/#251 archivados.** Los MODIFIED parten de `openspec/specs` actual. `sdd-independent-review` pasa a ser capacidad modificada (§4.2).
- **#250 (`modelos-por-fase`)** está en la base; no comparte capacidades con este cambio.
- **Solapes con P6:** `internal/workspace/declarations.go` (declaración ISO frente a intake de `baseline:`) y el bloque ODD de `routing.go` (D4 frente al localizador de G6.6). Quien llegue segundo rebasa.
- **Cambio siguiente:** D5 (formatos KVP) diseña su propia frontera del destino del archivado. El archivado en el mismo PR, si se retoma, es otro cambio.
- Reutiliza `internal/specbaseline/trace.go`, `internal/specbaseline/ids.go`, `kickoff.RoleTaskFiles` y la evidencia `pr_merged` de `internal/kickoff/integration.go`.
- Restricciones de trabajo: sin compilar binarios sueltos (solo `go test` acotado por paquete), commits Conventional sin atribución de IA, RDD fuera del SDD.

## 8. Riesgos (Risks)

| Riesgo | Prob. | Mitigación |
|---|---|---|
| La excepción ISO choca con REQ-28.3 de #249 («ninguna revisión bloquea el archivado»), que convive en el mismo `verify-report.md`. | Media | Delta explícita de REQ-28.3: solo la validez de la matriz bloquea, y solo en ISO; los hallazgos siguen informativos. |
| La matriz sigue declarando el resultado: Go contrasta que la prueba existe y que el comando está registrado, no que pase. | Media | Aceptado; el comando y su código de salida permiten repetir la prueba en auditoría. Reejecutar queda fuera de alcance. |
| La resolución de referencias de prueba es dependiente del lenguaje y da falsos rechazos en proyectos no Go. | Media | Forma general `ruta/fichero:símbolo` (fichero presente y símbolo definido en él); el atajo `TestX` solo para Go (Q14). |
| La trazabilidad con la rama `-archive` deriva dos commits (código y archivado) y puede elegir mal el de código con rebase (varios commits en `main` tocan la carpeta). | Media | El de código se contrasta con la evidencia `pr_merged` de la compuerta `integration`; prueba con squash, merge y rebase. |
| Regeneración de goldens y paridad entre runtimes. | Media | Prosa al final, goldens regenerados una vez, pines de paridad en los tests de assets. |
| La regla de escenarios rechaza el archivado de cambios activos con requisitos sin escenarios. | Baja | Solo en ISO (Q8) y solo para cambios sellados con `iso` (instantánea); motivo accionable. |
| La corrección de la pieza 1 desbloquea `verify` antes de que cierre el último rol activo y el aviso de REQ-21.13 queda desfasado. | Baja | La spec lo resuelve (Q13). |
| Solape con P6 en `declarations.go` y en el bloque ODD de `routing.go`. | Media | Rebasar sobre lo que se fusione antes. |
| Tamaño por encima de 400 líneas. | Alta | Una sola PR con `size:exception` (§9); piezas como unidades de trabajo con commit propio. |

Riesgo global: **medio-bajo** (antes medio). Baja porque desaparecen el motor de ramas, el archivado en el mismo PR y la validación de design, y la cadena de prosa ya está fusionada; lo que queda es aditivo, opt-in salvo las dos correcciones, y acotado a `sddstatus`, `kickoff`, `workspace` y `specbaseline`. Se mantiene el riesgo de tamaño y el de coherencia con #249.

## 9. Entrega (Review Workload Forecast)

- Estrategia: **una sola PR con `size:exception`** (preferencia permanente del usuario: nunca cadena). Cada pieza de §3.1 es una unidad de trabajo con su commit, sus pruebas y su delta, de modo que el diff se revisa por commits.
- Previsión no medida, orientativa: del orden de **1.200 a 1.800 líneas autoradas** (Go, pruebas, skills, specs y doctrina), sin contar goldens.

```
Decision needed before apply: No
Chained PRs recommended: No
400-line budget risk: High
```

## 10. Plan de reversión (Rollback Plan)

- **Por configuración.** Un proyecto deja de estar en modo ISO retirando `sdd.iso` de `axiom.yaml`; los cambios ya sellados con `iso` conservan su instantánea y la siguen exigiendo hasta archivarse (o se terminan antes de revertir).
- **Por código.** `git revert` del PR fusionado, o quedarse en la versión publicada anterior (`go install` de la etiqueta previa en Windows; canal habitual en el resto). No hay migraciones de datos.
- **Por doctrina.** `axiom sync` con el binario revertido reproyecta las skills anteriores.
- **Por datos.** Los cambios activos y archivados en el formato actual siguen siendo válidos con cualquier versión. Un cambio activo que use el perfil de bug sin delta no es completo para el binario anterior: habría que añadirle una delta o terminarlo antes de revertir. Un binario anterior ignora el bloque `sdd:` (claves de primer nivel desconocidas, `internal/workspace/loader.go:53-55`) y la clave nueva de `kickoff.yaml` según su decodificación (a confirmar en design).
- **Señal para revertir.** Un proyecto no ISO bloqueado por un validador nuevo; un `verify` que siga bloqueado por un rol `deferred`; un archivado ISO que pase sin matriz válida; una referencia de prueba existente rechazada.

## 11. Criterios de éxito (Success Criteria)

- [ ] **QA diferido.** Con un rol `blocking` aprobado y un rol `deferred` sin aprobar, `verify` queda desbloqueado y el rol diferido se informa; con un `blocking` sin aprobar sigue bloqueado.
- [ ] **Arranque.** `axiom sdd kickoff seal` funciona sin crear la carpeta a mano.
- [ ] **Sin ISO, sin cambios.** Un proyecto sin la declaración ISO recorre `continuous` y `checkpointed` con las mismas compuertas y paradas que hoy, y archiva por la rama `-archive` con un `verify-report.md` libre o ausente.
- [ ] **Escenarios.** En ISO, `archive-compose` rechaza, con motivo tipado, un requisito ADDED o MODIFIED sin escenarios.
- [ ] **ISO e identidad.** Con la declaración ISO, una delta que afecta a un documento sin `identity: id` bloquea el cambio con motivo accionable.
- [ ] **Cobertura.** En ISO, `status` bloquea si algún REQ de la delta no aparece en ningún `Cubre:`.
- [ ] **Matriz.** En ISO, el archivado se bloquea si algún escenario no tiene fila, alguna fila apunta a un escenario inexistente, no tiene resultado o lo tiene fallido; si una fila automática apunta a una prueba que no existe en el árbol o no registra comando y código de salida; o si una fila manual carece de quién, cuándo y por qué.
- [ ] **Coherencia con #249.** La matriz convive con «Cumplimiento técnico» en el mismo informe; la barrera sigue leyendo el mismo veredicto; un hallazgo de revisión sigue sin bloquear el archivado, también en ISO.
- [ ] **Trazabilidad.** Para un cambio ISO archivado por la rama `-archive`, la cadena REQ → `Cubre:` → fila de la matriz → commit de integración del código → commit de archivado se reconstruye con `git log --first-parent` igual con squash, merge y rebase (prueba con los tres).
- [ ] **Bug sin delta.** Un bug sin delta con `Cubre:` a un REQ inexistente bloquea en ISO; con un REQ existente llega al archivado; fuera de ISO queda preparado sin delta.
- [ ] **Instantánea (P5).** `kickoff seal` copia `sdd.iso` en `kickoff.yaml`; activar `iso` a mitad no exige la matriz a un cambio ya sellado y desactivarlo no deja de exigirla; un cambio sin instantánea cuenta como no ISO; `status` avisa con una nota no bloqueante de la discrepancia.
- [ ] **Doctrina.** «Verification never gates archive» sigue vigente y declara la excepción ISO; REQ-28.3 la declara también; el bloque ODD expresa D4; los goldens están regenerados y en paridad.
- [ ] **CI verde.** `go test` de los paquetes tocados y `go vet ./...` pasan; los ratchets de refusals y deadcode no empeoran.

## 12. Resolución de D4, D5 y D6 (auditoría de visión, P6)

| Id | Resolución |
|---|---|
| D4 | **Resuelta.** ODD nunca integra en la spec viva. En proyectos ISO, los bugs van por el SDD: con el perfil de bug sin delta cuando restauran un requisito existente, y como cambio normal con delta cuando estaba mal especificado o faltaba. |
| D5 | **Fuera de alcance.** Cambio aparte (formatos KVP), que diseñará su propia frontera del destino del archivado. |
| D6 | **Resuelta vía el modo ISO opcional.** Un proyecto declarado ISO deja las evidencias que pide el atributo de proceso PA 2.2 (gestión de productos de trabajo) de la familia ISO/IEC 330xx: productos de trabajo definidos y validados (propuesta, delta con al menos un escenario por requisito, design, tasks con `Cubre:`, matriz de verify con evidencia contrastable), revisiones registradas (las compuertas existentes en `checkpointed` y las revisiones de #249), trazabilidad requisito → tarea → verificación → commit (C8) e IDs de requisito estables (C10). La calidad de la especificación de requisitos se apoya en ISO/IEC/IEEE 29148. Los proyectos no ISO no quedan sujetos a estas evidencias. |

La hoja de ruta de la auditoría puede marcar D4 y D6 como resueltas (D6 para proyectos que declaren ISO) y D5 como asignada a su propio cambio.

## 13. Preguntas abiertas

Solo huecos reales. Las que bloquean, bloquean únicamente la spec del requisito que nombran; el resto de la spec puede avanzar.

| ID | Pregunta | Bloquea | Quién la cierra |
|---|---|---|---|
| Q1 | ¿Cuándo se publica el ID definitivo de un requisito ADDED hacia la herramienta externa? Hasta el archivado el ID es provisional (`internal/specbaseline/ids.go`). | Nada en este cambio | Usuario |
| Q3b | Tipo y valores de `sdd.iso` (booleano u otro) y forma de declarar que un cambio es un bug sin delta. | Design | Design |
| Q6 | Si se quiere declarar el modelo de referencia de procesos y el nivel de capacidad objetivo (las evidencias se orientan a PA 2.2, nivel 2). | Nada | Usuario (opcional) |
| Q13 | Con la pieza 1, `verify` puede abrirse antes de que cierre el último rol activo. ¿El aviso de REQ-21.13 («queda abierta la verificación global») se mantiene al cerrar el último rol activo, se adelanta al cierre del último rol `blocking`, o se ajusta su texto? | Solo la delta de REQ-21.13/REQ-21.15 | Usuario (la spec propone) |
| Q14 | Formas de referencia de prueba que Go acepta en filas automáticas: ¿basta la forma general `ruta/fichero:símbolo` para cualquier lenguaje, más el atajo `TestX` en Go, o el cliente necesita otras (por ejemplo, identificadores de casos de una herramienta e2e)? | Solo el requisito de la matriz | Usuario |

### 13.1 Preguntas resueltas o que dejan de aplicar

| ID | Resolución |
|---|---|
| Q2 | **Superada.** El estilo sellado sigue siendo solo el eje de ritmo y no cuenta personas. |
| Q3 | **Bloque nuevo `sdd:` con la clave `iso`.** Lista cerrada de claves y validación estricta tipo `unknownKeys`. No se toca `workflow:` ni REQ-2.1 de `workspace-topology`. Un binario anterior ignora las claves de primer nivel desconocidas (`loader.go:53-55`). `archive` y `design_triggers` salen con el recorte. |
| Q7 | **Descartada** (2026-11-11). |
| Q8 | **Solo en ISO.** La regla de al menos un escenario por requisito se aplica únicamente a cambios sellados ISO. |
| Q9, Q10 | **Dejan de aplicar**: el design proporcional y por rol sale del alcance. |
| Q11, Q11b | **Dejan de aplicar**: el archivado en el mismo PR sale del alcance; la rama `-archive` es el único modo. |
| P6 | **Deja de aplicar** con Q11. |
| Q12 | **PR única** con `size:exception` (preferencia permanente del usuario). |
| Baseline implícita | **Cuenta como `identity: id` en ISO.** Solo se exige `identity: id` en baselines declaradas (`baseline:`); en la implícita `openspec/specs/` el identificador de un requisito es el `REQ-<n>.<m>` de su nombre (**supuesto**). |
| P5 | **Instantánea de `sdd.iso` al sellar** (reducida). `axiom sdd kickoff seal` copia `sdd.iso` en `kickoff.yaml` con la semántica de sellado único (precedente: `docs_publication`, REQ-27.4). El valor sigue siendo del proyecto. Un cambio sin instantánea cuenta como no ISO. Cambiar `sdd:` solo afecta a los cambios sellados después; `status` avisa con una nota no bloqueante. El retrosellado congela `iso: false` sin leer `axiom.yaml` (REQ-21.4; resuelto por el usuario). |

### 13.2 Supuestos visibles para la compuerta `spec`

| Supuesto | Dónde consta (orientativo) |
|---|---|
| En ISO, una fila de la matriz que apunta a un escenario inexistente invalida la matriz. | Requisito de la matriz (REQ-29.x) |
| En multirrol, la matriz vive en el `verify-report.md` consolidado de REQ-21.15; con el rol único `fullstack` es el mismo informe que el del rol (REQ-28.4). | Requisito de la matriz; delta de REQ-28.4 |
| En ISO, una matriz no válida cuenta como comprobación obligatoria fallida y el verify no es limpio (REQ-28.6). | Delta de `sdd-independent-review` |
| En la baseline implícita, el identificador de un requisito es el `REQ-<n>.<m>` de su nombre. | Requisitos de identidad, cobertura y bug sin delta |
| En un proyecto ISO, el perfil de bug del SDD prevalece sobre un `workflow.lane: odd` declarado cuando el trabajo es corregir un bug. | Requisito de D4 en `organic-agent-trigger-rules` |
| `archive-compose` determina el cambio (por `--change` o por la ruta de la delta) para leer su instantánea; sin cambio determinable no aplica la regla de escenarios. | Requisito de escenarios en `sdd-archive-lifecycle` |
| La referencia de una fila automática se resuelve contra el árbol de trabajo en el momento de validar (verify en la rama del cambio, archivado en la rama `-archive`). | Requisito de la matriz |

Numeración: los requisitos nuevos usan la serie **REQ-29.x**. Las specs de `main` llegan hasta REQ-28 (`sdd-revision-independiente`, REQ-28.1 a REQ-28.14, archivado en #251); las referencias a REQ-28.x de esta propuesta que no son de `sdd-independent-review` (o de los cambios de `sdd-preflight-configuration`, `sdd-lane-kickoff` y `sdd-block-review-gates` que ese cambio introdujo) eran de la versión anterior y quedan sustituidas por REQ-29.x.
