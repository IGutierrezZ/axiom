# Propuesta: Evidencias ISO y correcciones del SDD (carril-sdd-versatil)

> Fase `propose` (remediación) · rama `inc/carril-sdd-versatil` · 2026-11-05 · sin `explore.md`: la entrada son las decisiones de producto cerradas con el usuario (Engram, `architecture/sdd-carril-versatil/*`), la segunda opinión (`architecture/sdd-carril-versatil/segunda-opinion`) y los huecos de código verificados.
> **Enmienda 2026-11-12 (D4/D5).** Aprobada por el usuario. D4: ODD integra en la spec viva y los proyectos ISO solo usan SDD (redacción vigente tras la tercera enmienda); este cambio remite todo el trabajo de un proyecto ISO al SDD y valida que `sdd.iso: true` no conviva con `workflow.lane: odd`, y la integración desde ODD (verbo y revisión de impacto en ODD) la diseña P6. D5: OpenSpec es el único formato de spec viva y la migración de proyectos a OpenSpec va en P6; no hay otros formatos de destino.
> **Actualización 2026-11-12 (segunda enmienda).** Tres enmiendas cerradas con el usuario, sin reabrir D4 ni D5. (1) **Fuera lo que solo sirve con una `baseline:` declarada:** ningún proyecto real la declara y P6 (`spec-viva-openspec`) retirará el bloque y su maquinaria cuando este cambio se fusione. Se retiran C10 (`identity: id`), el último escenario de REQ-29.9, la resolución por perfil de REQ-29.14 (`spec-baseline-integration`) y el MODIFIED de REQ-24.46. REQ-29.3 se reduce a «el nombre de todo requisito de la delta lleva un `REQ-<n>.<m>` válido, único y sin ambigüedad». **Ningún ID de la serie 29 queda retirado ni renumerado.** (2) **Perfil de bug con tres rutas:** `sdd.iso` es booleano (Q3b resuelta); el bug se declara con una sección fija `## Informe de bug` (`Observado:` y `Esperado:` obligatorias, `Referencia:` opcional) y hay una tercera ruta, el bug **sin impacto en la spec** (`## Impacto en la spec` con la línea literal `Sin impacto en la spec` y `Motivo:`), aceptada en todo proyecto y, en ISO, con el motivo como evidencia (REQ-29.15, REQ-29.16 y el nuevo **REQ-29.19**). (3) **Retrosellado coherente:** `kickoff seal --infer` no escribe `iso` (serialización `omitempty`) y el cambio cuenta como no ISO, igual que uno sin instantánea (REQ-29.13).
> **Actualización 2026-11-12 (tercera enmienda).** Decisión cerrada con el usuario: **D4 pasa a ser «ODD integra en la spec viva; los proyectos ISO solo usan SDD»**. En un proyecto ISO todo el trabajo (no solo los bugs) va por el carril SDD, y los bugs van siempre por el SDD con el perfil de bug y sus tres rutas. Nivel de cumplimiento: **doctrina más validación**. (1) **Doctrina (REQ-29.18, `organic-agent-trigger-rules`):** el bloque ODD dice que, con ISO proyectado por el status, todo el trabajo se tramita por el SDD sin preguntar el carril y sin crear `odd/tasks/<feature>.md`, y que un bug usa el perfil de bug. El supuesto visible «el perfil de bug prevalece sobre `workflow.lane: odd`» desaparece. (2) **Validación (REQ-29.1, `sdd-iso-evidence`):** `sdd.iso: true` junto a `workflow.lane: odd` es una declaración contradictoria, error accionable al cargar y en `axiom workspace validate` (salida 1) y fallo cerrado al sellar; no cambia el bloque `workflow:` ni sus valores (`odd`, `sdd`, `ask`), y no hace falta MODIFIED en `workspace-topology`. **Ningún ID de la serie 29 cambia.** La integración de ODD en la spec viva sigue en P6.
> Kickoff sellado: `checkpointed`, `handoff_policy: none`, rol único `fullstack` (`blocking`).
> Pre-vuelo de la sesión (Engram `sdd/carril-sdd-versatil/preflight`): ritmo automático, artefactos `hybrid`, entrega `single-pr` con `size:exception` aprobada por el mantenedor.
> Actualización 2026-11-10: la rama está al día con `origin/main` tras #235 (`carril-proyecto-publicacion-docs`). Se incorporan Q3 (bloque `sdd:`), Q11b y la baseline implícita.
> Actualización 2026-11-10 (2): #235 archivado en `openspec/specs` (#236). Se añaden P5 (instantánea en `kickoff.yaml`) y P6 (capa first-parent de `axiom branch done`).
> **Actualización 2026-11-11 (recorte).** El usuario recorta el cambio a lo que importa: **evidencia ISO para un cliente real** y dos correcciones del SDD actual (QA diferido y arranque con carpeta). Se cortan `story.md`, el design proporcional y por rol (pieza 9), el archivado en el mismo PR (pieza 10 y P6), la frontera del destino del archivado (pieza 11, innecesaria: OpenSpec es el único formato) y Q7. P5 se reduce a `sdd.iso`; el perfil de bug se reduce a la ruta sin delta. La matriz de verify pasa a exigir **evidencia contrastable** y se integra en el `verify-report.md` que existe tras #249. La rama está en `origin/main` `1b632f03`, con `sdd-revision-independiente` archivado (#251): sus requisitos ocupan REQ-28.1 a REQ-28.14, así que los requisitos nuevos de este cambio pasan a la serie **REQ-29.x**.
> Esta versión **extiende** el SDD actual, no lo sustituye (rechazo de la compuerta `spec` del 2026-11-05, Engram `architecture/sdd-carril-versatil/decisiones-convivencia`). El slug se conserva.

## 0. Resumen

El SDD actual funciona y se conserva entero: el preflight Pace como eje de ritmo (`continuous`/`checkpointed`), las compuertas `spec`, `design`, `tasks`, `role-apply:<rol>` e `integration`, las paradas de tres salidas, el archivado en la rama `<prefijo>/<cambio>-archive`, la doctrina «Missing or failed verification never gates archive» para proyectos no ISO, las revisiones independientes informativas de #249 y los cambios activos en su formato actual. Lo que le falta es la evidencia que pide un auditor ISO para un cliente real: requisitos con identificador estable (`REQ-<n>.<m>`), cada escenario verificado con evidencia contrastable y trazabilidad requisito → tarea → verificación → commit de `main`.

Este cambio **extiende** el SDD con un **modo ISO opcional por proyecto**, declarado en un bloque nuevo de primer nivel `sdd:` de `axiom.yaml` con una sola clave, `iso`. El modo activa: al menos un escenario por requisito, un `REQ-<n>.<m>` válido en el nombre de cada requisito de la delta, `Cubre: REQ-x` en las tareas con cobertura validada por Go, una matriz de verify por escenario cuya evidencia automática Go puede contrastar con el árbol, un `verify` que sí bloquea el archivado y la trazabilidad por `git log --first-parent`, compatible con la rama `-archive`. Añade el perfil de bug, declarado con `## Informe de bug` y con tres rutas, necesario porque en un proyecto ISO todo el trabajo va por el SDD y los bugs, con este perfil (D4): **sin delta con `Cubre:`** a un requisito existente, **sin impacto en la spec** (marca `Sin impacto en la spec` con su `Motivo:`, sin delta ni `Cubre:`) y **con delta** como cambio normal. Corrige de paso dos fricciones del SDD actual: el QA `deferred` que bloquea `verify` y el `kickoff seal` que no crea la carpeta del cambio. Resuelve D4 y D6 (esta última vía el modo ISO) y deja D5 resuelta: OpenSpec es el único formato de spec viva y la migración de proyectos a OpenSpec va en P6.

## 1. Propósito (Intent)

### 1.1 Problema

| Hueco | Evidencia |
|---|---|
| Un rol con `gate_policy: deferred` u `optional` bloquea `verify` igual que uno `blocking`: la condición exige aprobado el `role-apply` de **todos** los roles del roster, sin mirar la política. La barrera de `multi-role-fan-out` ya distingue por política, así que estado y barrera discrepan. | `internal/sddstatus/governance_handoff.go:263-285`; `internal/multirole/barrier.go:124`; REQ-21.15 en `openspec/specs/sdd-role-closure-handoff/spec.md:68-70`; REQ-2.1 en `openspec/specs/multi-role-fan-out/spec.md:69-74` |
| `axiom sdd kickoff seal` rechaza un cambio cuya carpeta no existe y no la crea (fricción observada al arrancar este mismo cambio). | `internal/cli/sdd_kickoff_test.go:239-253` (la prueba fija el rechazo) |
| La composición del archivado no valida escenarios: un requisito puede integrarse sin ningún criterio de aceptación verificable. | `internal/sddstatus/openspec_archive_compose.go` |
| No hay forma de declarar un proyecto como ISO en `axiom.yaml`. | `internal/workspace/declarations.go` (decodificación estricta de bloques de primer nivel) |
| `verify-report.md` no tiene cadena escenario → evidencia: la barrera solo lee `verdict:` y la sección «Cumplimiento técnico» de #249 contrasta reglas, no escenarios. | `internal/multirole/barrier.go:12` y `:110-119`; `internal/assets/skills/sdd-verify/references/report-format.md:11` |
| Las tareas no declaran qué requisito cubren; nada comprueba que cada requisito de la delta tenga al menos una tarea. | Ausencia de `Cubre:` en `internal/` |
| La doctrina declara que la verificación nunca bloquea el archivado, y #249 lo repite para las revisiones independientes; un proyecto ISO necesita lo contrario para su evidencia y hoy no puede pedirlo. | `internal/assets/skills/_shared/persistence-contract.md:69`; REQ-28.3 en `openspec/specs/sdd-independent-review/spec.md:112-121` |
| La preparación del cambio exige siempre una delta de spec; un bug que restaura un requisito existente, o que no toca ningún requisito, no tiene ruta. | `internal/sddstatus/status.go` (`coreReady`) |
| No hay vía prevista para trazar un bug de un proyecto ISO contra el requisito que incumple, ni nada que impida que un proyecto ISO declare el carril `odd` (D4: los proyectos ISO solo usan SDD); la integración desde ODD la diseña P6. | Auditoría de visión, G6.5 y D4; `internal/workspace/workflow.go` (`workflow.lane` admite `odd`, `sdd` y `ask`) |

### 1.2 Por qué ahora

Un cliente real necesita evidencia ISO de los proyectos que se desarrollan con Axiom, y la auditoría de visión (hito P6) dejó pendientes D4 y D6 (Engram `architecture/sdd-carril-versatil/coordinacion-p6`). La segunda opinión mostró que esa evidencia se puede añadir sin retirar nada de lo que hoy funciona. Las demás mejoras que se habían sumado al cambio no aportan evidencia ISO y retrasaban la entrega: se aplazan (§2.2, §3.2).

### 1.3 Resultado esperado

- Un proyecto sin declaración ISO se comporta como hoy, salvo las dos correcciones de §3.1 (piezas 1 y 3) y el perfil de bug, que es opcional.
- Un proyecto declarado ISO obtiene, sin cambiar de ritmo, de compuertas ni de modo de archivado: al menos un escenario por requisito, un identificador `REQ-<n>.<m>` estable en el nombre de cada requisito de la delta, cobertura `Cubre:` de cada requisito, una matriz de verify por escenario con evidencia automática contrastable con el árbol y evidencia manual justificada, y un archivado que exige esa matriz válida y con éxito.
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
| C8 | Trazabilidad independiente de squash, merge o rebase: delta, `Cubre:`, matriz y commit de integración derivado con `git log --first-parent <main>`. Funciona con la rama `-archive`, que sigue siendo el único modo de archivado. Los trailers intermedios son informativos y nunca se validan. **Modo ISO.** | Modificada (con rama `-archive`) | `decisiones-trazabilidad` |
| C9 | ~~Archivado en el mismo PR como opción por proyecto en monorepo.~~ | **Aplazada** (2026-11-11): candidata a un cambio aparte posterior | `decisiones-archivado` |
| C10 | ~~`identity: id` en los documentos de una baseline declarada.~~ Ningún proyecto declara `baseline:` y P6 retira el bloque. En ISO solo cuenta la baseline implícita `openspec/specs/`, cuyo identificador es el `REQ-<n>.<m>` del nombre del requisito: se comprueba en el estado del cambio, una vez escrita la delta y antes de avanzar a design, que todo requisito de la delta lo lleva válido (REQ-29.3, reducido). | **Retirada** la parte de `identity: id` (2026-11-12); se conserva la comprobación del `REQ-<n>.<m>` | `decisiones-baseline`, decisión del usuario 2026-11-12 |
| C11 | Perfil de bug declarado con la sección fija `## Informe de bug` de `proposal.md` (`Observado:` y `Esperado:` obligatorias, `Referencia:` opcional; ver C17). Tres rutas: **sin delta con `Cubre:`** al REQ existente que el bug incumple (en ISO, Go comprueba que existe y una fila de evidencia por escenario afectado); **sin impacto en la spec** (C18); y **con delta** como cambio normal cuando el requisito estaba mal especificado o ausente. Disponible en todo proyecto. | Ampliada (2026-11-12): tercera ruta y formato fijado | `decisiones-bugs` |
| C12 | ~~Destino del archivado tras una frontera para D5.~~ Innecesaria: OpenSpec es el único formato (D5). | **Cortada** (2026-11-11; confirmada 2026-11-12) | `decisiones-d5` |
| C13 | ~~Sustituye al SDD actual.~~ **Extiende** el SDD actual; todo convive; sin migración obligatoria. | **Sustituida** | `decisiones-convivencia` |
| C14 | Lo que toque la prosa del orquestador se implementa el último de su cadena. La cadena ya está fusionada (§7), así que la condición queda satisfecha. | **Cumplida** | `orden-ejecucion` |
| C15 | En modo ISO, la matriz de verify válida y con éxito bloquea el archivado, con independencia del ritmo. Es una excepción **declarada** a la doctrina de `persistence-contract.md:69` y a REQ-28.3, que siguen vigentes para proyectos no ISO y para los hallazgos de revisión. | Se mantiene | `decisiones-convivencia` |
| C16 | La corrección del QA `deferred` y el arranque con carpeta son entregables independientes del modo ISO. | Se mantiene | `decisiones-convivencia` |
| C17 | `sdd.iso` es **booleano** (`iso: true\|false`). Formato de la declaración de bug: sección `## Informe de bug` con las líneas `Observado:` y `Esperado:` (obligatorias) y `Referencia:` (opcional); la propia sección es la declaración, sin front matter. Resuelve Q3b. | Nueva (2026-11-12) | Decisión del usuario 2026-11-12 |
| C18 | **Bug sin impacto en la spec**: tercera ruta de bug, que no toca ningún requisito. Se declara con `## Impacto en la spec` cuyo contenido es la línea literal `Sin impacto en la spec` y una línea `Motivo: <una línea>`, junto a `## Informe de bug`, sin delta y sin `Cubre:`. Texto exacto, porque el carril ODD (P6) usará la misma sección y la misma línea. `coreReady` lo acepta en todo proyecto, sea o no ISO; el archivado no integra nada. En ISO basta el motivo: la matriz es vacía y válida, y no se crea ningún tipo de fila nuevo (REQ-29.19). | Nueva (2026-11-12) | Decisión del usuario 2026-11-12 |
| C19 | El retrosellado (`kickoff seal --infer`) **no escribe `iso`**: el booleano se serializa con `omitempty`, así que un cambio retrosellado no lleva la clave y cuenta como no ISO, igual que uno sin instantánea. | Nueva (2026-11-12); sustituye a «congela `iso: false`» | Decisión del usuario 2026-11-12 |

## 3. Alcance (Scope)

### 3.1 Dentro de alcance (In Scope)

La numeración de piezas se conserva para no romper las referencias de los artefactos anteriores; las piezas 9 a 11 se han cortado.

**Correcciones del SDD actual (independientes del modo ISO)**

1. **QA diferido no bloquea `verify` (C3 residual, C16).** La condición de desbloqueo de `verify` solo exige el `role-apply` aprobado de los roles con política `blocking`; los `deferred` y `optional` no bloquean y se informan, igual que ya hace la barrera (`barrier.go:124`). Modifica REQ-21.15. Coordinación con #249: REQ-28.4 de `sdd-independent-review` dice que el verify global «se evalúa tras todos los `role-apply` aprobados (REQ-21.15)» y que «sigue evaluándose tras aprobar todos los `role-apply`» (`openspec/specs/sdd-independent-review/spec.md:148`); la delta de REQ-28.4 sustituye esa frase por una remisión a la condición de REQ-21.15 («cuando REQ-21.15 habilita el verify global»), sin repetirla, para que no vuelvan a divergir.
3. **Arranque con carpeta (C16) e instantánea de `sdd.iso` (P5 reducida).** `axiom sdd kickoff seal` deja preparada la carpeta del cambio en lugar de exigir que exista (mecanismo en design; hoy lo impide la prueba de `sdd_kickoff_test.go:239-253`, que habrá que invertir) y copia `sdd.iso` del proyecto en `kickoff.yaml` como instantánea: el valor sigue siendo del proyecto y las reglas ISO de ese cambio se leen de su instantánea. El booleano se serializa con `omitempty`: solo un cambio ISO lleva la clave `iso`, y el retrosellado (`seal --infer`) no la escribe, de modo que cuenta como no ISO (C19).

**Modo ISO opcional por proyecto**

2. **Al menos un escenario por requisito (C4).** En un cambio ISO, `archive-compose` rechaza, con motivo tipado que nombra el requisito, un requisito ADDED o MODIFIED sin escenarios; `status` lo adelanta. Fuera de ISO no cambia nada (Q8).
4. **Declaración ISO e identificador del requisito (C10, C17).** Nuevo bloque de primer nivel `sdd:` en `axiom.yaml` con una única clave, `iso` (booleano), lista cerrada y validación estricta (`unknownKeys`, `internal/workspace/declarations.go:578`, al estilo de `workflowKeys` en `internal/workspace/workflow.go:37`); `workflow:` no se toca. Comprobación, con motivo de bloqueo accionable, de que el nombre de todo requisito de la delta lleva un `REQ-<n>.<m>` válido, sin ambigüedad ni repetición (REQ-29.3); en la baseline implícita `openspec/specs/` el identificador es ese, no hay perfil ni `identity: id` que comprobar. Un cambio sin instantánea cuenta como no ISO.
5. **Cobertura `Cubre:` (C6).** Analizador de `Cubre: REQ-x` en `tasks.md`/`tasks.<rol>.md` y validación conjunta: cada REQ de la delta, al menos una tarea (o el REQ existente en un bug sin delta).
6. **Matriz de verify con evidencia contrastable y bloqueo del archivado (C7, C15).** Esquema tabular de la matriz como sección de `verify-report.md` en ISO, junto a «Cumplimiento técnico» y sin escribir líneas `verdict:` (la barrera lee la primera, `report-format.md:11`). Go valida:
   - que todo escenario de la delta (o del REQ existente, en un bug sin delta) tenga fila y que ninguna fila apunte a un escenario inexistente;
   - en filas automáticas (unitario, integración, e2e): que la referencia **resuelva a una prueba existente en el árbol** (forma general `ruta/fichero:símbolo`, con el fichero presente y el símbolo definido en él; en Go basta `TestX` si está definido en un `*_test.go`) y que la fila registre el **comando ejecutado y su código de salida**;
   - en filas manuales: quién, cuándo y por qué no se automatiza;
   - que todo resultado sea de éxito.

   El archivado ISO exige la matriz válida y con éxito. Que la prueba pase sigue siendo **declarado**, pero queda contrastable: la prueba existe y el comando que la ejecuta está registrado para repetirlo. En ISO, una matriz no válida cuenta como comprobación obligatoria fallida, de modo que el verify no es «limpio» en el sentido de #249 (REQ-28.6). La doctrina declara la excepción ISO en `persistence-contract.md` y en REQ-28.3: los hallazgos de revisión siguen siendo informativos; solo la validez de la matriz bloquea.
7. **Trazabilidad (C8).** Derivación con `git log --first-parent <rama de integración>` de dos commits, sin estado nuevo: el de integración del código (el que trae `openspec/changes/<cambio>/` con sus `tasks.md` y `verify-report.md`, contrastable con la evidencia `pr_merged` de la compuerta `integration`, `internal/kickoff/integration.go:9`) y el del archivado (el que trae `openspec/changes/archive/<fecha>-<cambio>/` desde la rama `-archive`). La traza de integración de IDs (`integration.yaml`, ID provisional y definitivo) no interviene: en la baseline implícita el identificador lo elige el autor; el design decide qué de `internal/specbaseline/` se reutiliza, si algo.

**Perfil de bug (tres rutas)**

8. **Perfil de bug (C11, C17, C18).** Un cambio se declara bug con la sección `## Informe de bug` de `proposal.md` (`Observado:`, `Esperado:`, `Referencia:` opcional). Rutas: (a) **sin delta con `Cubre:`** a un REQ existente de la baseline: `coreReady` deja de exigir la delta solo en ese caso; en ISO, Go comprueba que el REQ existe y la matriz exige una fila por escenario de ese REQ; (b) **sin impacto en la spec**: `## Impacto en la spec` con la línea literal `Sin impacto en la spec` y `Motivo:`, sin delta y sin `Cubre:`; `coreReady` lo acepta en todo proyecto, el archivado no integra nada y, en ISO, el motivo es la evidencia (matriz vacía válida, sin tipo de fila nuevo; REQ-29.19); (c) **con delta**, como cambio normal, cuando el requisito estaba mal especificado o ausente. `coreReady` sigue aceptando el formato actual. Necesario porque en ISO todo el trabajo va por el SDD y los bugs usan siempre este perfil (D4).

**Transversal**

12. **D4: doctrina y validación (REQ-29.18, REQ-29.1).** «ODD integra en la spec viva; los proyectos ISO solo usan SDD». (a) *Doctrina:* una frase en el bloque ODD: cuando el status proyecta un proyecto ISO, todo el trabajo se tramita por el SDD, sin preguntar el carril y sin crear `odd/tasks/<feature>.md`, y un bug lo hace con el perfil de bug y sus tres rutas; texto idéntico para todos los proyectos y adaptadores, sin paradas. (b) *Validación:* un `axiom.yaml` con `sdd.iso: true` y `workflow.lane: odd` es una declaración contradictoria y se rechaza como error accionable en español, con salida 1, al cargar la configuración y en `axiom workspace validate`, y falla cerrada al sellar; el mensaje nombra `sdd.iso` y `workflow.lane` y la corrección (retirar `workflow.lane: odd` o retirar `iso`). Es una validación cruzada nueva: no cambia el bloque `workflow:` ni sus valores. Son válidos `iso: true` con `lane: sdd`, con `lane: ask` o sin `lane`, e `iso: false` con `lane: odd`. La integración de ODD en la spec viva (verbo de integración de ODD y revisión de impacto en ODD) la diseña P6 y queda fuera de este cambio.
13. **Doctrina, specs, goldens y manual.** Skills `sdd-*` (sobre todo `sdd-verify/references/report-format.md`, `sdd-tasks` y `sdd-archive`), `_shared/sdd-status-contract.md`, `_shared/persistence-contract.md` (excepción ISO), `_shared/sdd-orchestrator-sections.md` donde haga falta; deltas de §4; regeneración de `testdata/golden/` y de las copias versionadas (`AGENTS.md`, `GEMINI.md`, `.codex/AGENTS.md`); `docs/manual-de-inicio.md` y `axiom.example.yaml` con la declaración ISO.

### 3.2 Fuera de alcance (Out of Scope)

- **`story.md` como variante de `proposal.md`** (C4). No aporta evidencia ISO: el informe de un bug cabe en `proposal.md`.
- **Pieza 9 entera** (C5, C6 parcial): design proporcional con mínimo y disparadores, escalado al arquitecto, `design.<rol>.md`, clave `design_triggers` y plan propio en `tasks.<rol>.md`. No aporta evidencia ISO y añadía validación de prosa con riesgo de ruido. Q9 y Q10 dejan de aplicar.
- **Pieza 10 y P6** (C9): archivado en el mismo PR, `axiom sdd publish --archive` en la rama del cambio, excepción a la rama `-archive` y a `change_not_integrated`, y capa first-parent de `axiom branch done`. La rama `-archive` sigue siendo el único modo. Q11 y Q11b dejan de aplicar. Candidato a un cambio aparte posterior.
- **Pieza 11** (C12): frontera del destino del archivado. Innecesaria: OpenSpec es el único formato de spec viva (D5).
- **Q7** («Listo»/«Hecho»): descartada.
- **Claves `archive` y `design_triggers` del bloque `sdd:`**: el bloque queda solo con `iso`.
- **Migración a OpenSpec e integración desde ODD (P6)**: D5 está resuelta (OpenSpec es el único formato de spec viva); la migración de proyectos a OpenSpec, el verbo de integración de ODD y la revisión de impacto de los bugs en ODD los diseña P6.
- **Plugins externos** (Azure DevOps, Jira, Redmine): solo se reserva el hueco de metadatos.
- **Cualquier retirada** de fases, compuertas, artefactos o del ritmo Pace; cualquier migración de cambios activos o archivados.
- **Reglas ISO sobre una baseline declarada (`baseline:`)**: ningún proyecto real la declara y P6 (`spec-viva-openspec`) retira el bloque y su maquinaria. Este cambio solo especifica la baseline implícita `openspec/specs/`.
- **Multirepo**: sin cambios en su archivado, salvo lo que comparta (matriz, cobertura, perfil de bug).
- **Ejecutar las pruebas desde Go para confirmar el resultado de la matriz**: queda declarado y contrastable, no reejecutado.
- Push, PR o merge desde Axiom (invariante del motor de ramas).
- Intake del repositorio de specs, `axiom init` con `baseline:` o `technical_context:` y demás huecos de P6 (G6.1 a G6.4, G6.6, G6.7).
- Editar `openspec/INDEX.md` a mano (lo regenera `axiom archive sync`).

### 3.3 Qué no cambia

El preflight Pace y el estilo sellado como eje de ritmo; las compuertas `spec`, `design`, `tasks`, `role-apply:<rol>` e `integration` y su registro en `gates.yaml`; las paradas de tres salidas (#218), que desde #235 publican los documentos con `axiom sdd publish`; el archivado en la rama `-archive` con su PR documental y `axiom branch done`; la doctrina «verification never gates archive» en proyectos no ISO; las revisiones independientes de #249 (verify por rol, juicio, cobertura), que siguen siendo informativas; la barrera de `multi-role-fan-out`; los cambios activos y archivados en el formato actual; RDD y su separación de SDD; el carril ODD y `odd/tasks/` para los proyectos no ISO, salvo la remisión de todo el trabajo ISO al SDD; el bloque `workflow:`, sus claves, sus valores (`odd`, `sdd`, `ask`) y su carril por defecto (la validación cruzada con `sdd.iso` no los toca); la persistencia `hybrid`; la traza de integración (`integration.yaml`) y la asignación de IDs, que se reutilizan; el contexto técnico mantenido y `context-impact.md`.

## 4. Capacidades (Capabilities)

> Contrato con `sdd-spec`. Baseline: `openspec/specs/`, la única spec viva (D5), que ya incluye #235 (archivado en #236) y `sdd-revision-independiente` (archivado en #251); los MODIFIED se redactan contra ese texto. Los requisitos nuevos usan la serie **REQ-29.x**; la numeración exacta la fija la spec. Las deltas que hay hoy en `specs/` del cambio corresponden a la versión anterior y usan REQ-28.x propios: se renumeran y se recortan como se indica.

### 4.1 Nuevas capacidades (New Capabilities)

- `sdd-iso-evidence`: el modo ISO completo. Bloque `sdd:` con la clave `iso` y su proyección aditiva en el status (absorbe lo que estaba en `spec-baseline-declaration`); identificador `REQ-<n>.<m>` en el nombre de los requisitos de la delta (REQ-29.3, reducido); cobertura `Cubre:`; matriz de verify con evidencia contrastable dentro de `verify-report.md`; bloqueo del archivado por la matriz (excepción declarada); trazabilidad por `--first-parent` con la rama `-archive`; la comprobación ISO de que el REQ de un bug con requisito existente existe; y la regla ISO del bug sin impacto en la spec (REQ-29.19, nuevo). Se conserva la delta actual, renumerada (REQ-28.1 a REQ-28.6 propios → REQ-29.x) y ampliada con la evidencia contrastable.

`sdd-artifact-profiles` **desaparece**: con `story.md` y el design cortados solo le quedaba el perfil de bug sin delta, que es una regla de preparación del cambio (`coreReady`) y encaja en `axiom-sdd-cli-integration` (parte general) y en `sdd-iso-evidence` (comprobación ISO).

### 4.2 Capacidades modificadas (Modified Capabilities)

- `sdd-role-closure-handoff`: REQ-21.15 — `verify` se desbloquea con los `role-apply` de los roles `blocking`; `deferred` y `optional` no bloquean y se informan. La spec revisa si el aviso de cierre de REQ-21.13 necesita ajuste (Q13).
- `sdd-independent-review`: REQ-28.4 — la frase sobre el verify global remite a la condición de REQ-21.15 en lugar de repetir «tras todos los `role-apply` aprobados»; la matriz ISO es una sección adicional del mismo informe, sin líneas `verdict:`. REQ-28.3 — excepción declarada: en un cambio ISO, la validez de la matriz bloquea el archivado; los hallazgos de revisión siguen sin bloquear. Si la spec lo considera necesario, REQ-28.6 aclara que en ISO una matriz no válida impide un verify limpio.
- `sdd-lane-kickoff`: `kickoff seal` prepara la carpeta del cambio; instantánea de `sdd.iso` en `kickoff.yaml` (se conservan las dos deltas actuales, renumeradas, y la de la instantánea se reduce a `iso`). El booleano se serializa con `omitempty` y el retrosellado no escribe `iso` (REQ-29.13).
- `sdd-archive-lifecycle`: en ISO, al menos un escenario por requisito y archivado bloqueado por la matriz. Se conserva solo el requisito de escenarios (renumerado); sobran los MODIFIED de REQ-21.16, REQ-25.39 y REQ-27.14 y el ADDED del archivado en el mismo PR.
- `spec-baseline-integration`: validación de escenarios en la composición (REQ-24.38; el texto de este MODIFIED conserva literal el del requisito vigente, que habla de documentos sin perfil) y consulta de un REQ existente por el `REQ-<n>.<m>` del nombre para el bug con requisito existente (REQ-29.14, sin resolución por perfil). Sobran los MODIFIED de REQ-24.46 (solo afectaba a documentos con perfil de una baseline declarada) y de REQ-24.47 (frontera de destino, innecesaria: OpenSpec es el único formato).
- `axiom-sdd-cli-integration`: `coreReady` acepta el bug sin delta (con requisito existente o sin impacto en la spec, declarados con `## Informe de bug` y, en el segundo caso, `## Impacto en la spec`) sin dejar de aceptar el formato actual, y motivos de bloqueo tipados de ISO, cobertura, escenarios, matriz y declaración contradictoria. Se recortan de la delta actual `story.md` y los motivos de design; sobra el MODIFIED de REQ-25.42 (`wrong_branch` con mismo PR).
- `sdd-orchestrator-assets`: skills `sdd-*` y doctrina con la excepción ISO, el formato de `Cubre:` y de la matriz, y el perfil de bug, en paridad entre runtimes.
- `organic-agent-trigger-rules`: solo la remisión de todo el trabajo de un proyecto ISO al SDD en el bloque ODD, con los bugs por el perfil de bug y sus tres rutas (REQ-29.18). La validación cruzada de `sdd.iso` con `workflow.lane: odd` vive en `sdd-iso-evidence` (REQ-29.1); `workspace-topology` (REQ-27.1) no necesita MODIFIED porque no cambia el bloque `workflow:` ni sus valores.

### 4.3 Capacidades que dejan de modificarse

| Capacidad | Delta actual | Motivo |
|---|---|---|
| `change-branch-lifecycle` | Sobra entera (REQ-25.18, REQ-25.45, REQ-25.16) | Solo existía por el archivado en el mismo PR y la capa first-parent de `branch done` (cortados). La trazabilidad lee `main` con `git log`, sin tocar el motor de ramas. |
| `sdd-block-review-gates` | Sobra entera (REQ-21.9, REQ-26.1) | Los cambios eran los disparadores y el arquitecto en la compuerta `design` y la exención de `integration` con mismo PR (cortados). Las compuertas no cambian. |
| `multi-role-fan-out` | Sobra entera (REQ-1.2, REQ-2.1) | REQ-1.2 era `design.<rol>.md` (cortado). REQ-2.1 ya distingue por política (`openspec/specs/multi-role-fan-out/spec.md:72-74`) y la barrera ya lo implementa (`barrier.go:124`); la corrección de la pieza 1 alinea el estado con ella sin cambiar su requisito. |
| `spec-baseline-declaration` | Sobra entera (bloque `sdd:` y su proyección) | El bloque solo declara `iso`; se especifica en `sdd-iso-evidence`. La frontera del destino del archivado se corta (OpenSpec es el único formato) y el bloque `baseline:` no se toca aquí (P6 lo retira). |
| `sdd-artifact-profiles` | Sobra entera | Ver §4.1. |

## 5. Enfoque (Approach)

1. **Correcciones primero (piezas 1 y 3).** Cada una es una unidad de trabajo con su prueba y su delta: el filtro por `gate_policy` en `verifyDependencyFromRoleGates` (el roster de `Governance` solo lleva nombres, `governance_handoff.go:200-209`; la política está en el kickoff, `governance.go:429`) y la preparación de la carpeta en `kickoff seal`. Van por el SDD porque modifican requisitos de la spec viva y este cambio es SDD.
2. **El modo ISO como una sola llave.** `sdd.iso` en `axiom.yaml`, copiada en `kickoff.yaml` al sellar, activa a la vez las piezas 2 y 4 a 7 y la comprobación ISO de la pieza 8. Sin ella, ningún validador nuevo bloquea.
3. **Validadores en Go, no en prosa.** Escenarios, `Cubre:`, la matriz (incluida la resolución de referencias de prueba contra el árbol), el `REQ-<n>.<m>` del nombre y las marcas del perfil de bug se analizan en Go y producen motivos de bloqueo tipados en `axiom sdd status` y en `archive-compose`; las skills solo describen el formato.
4. **La matriz se suma al informe de #249.** Es una sección más de `verify-report.md` con su propio esquema; no altera «Cumplimiento técnico», el veredicto que lee la barrera ni el carácter informativo de las revisiones.
5. **Doctrina con excepción explícita.** `persistence-contract.md` y REQ-28.3 mantienen la regla general y declaran la excepción ISO (C15).
6. **Perfil de bug aditivo.** `coreReady` omite la delta solo en un bug declarado con `## Informe de bug`: con `Cubre:` a un REQ existente o con la marca `Sin impacto en la spec`; el formato actual sigue siendo válido.
7. **Prosa y goldens al final** de la PR; los goldens se regeneran con `-update`, nunca a mano.

## 6. Áreas afectadas (Affected Areas)

| Área | Impacto | Qué cambia |
|---|---|---|
| `internal/sddstatus/governance_handoff.go`, `internal/sddstatus/governance.go` | Modificado | Desbloqueo de `verify` por `gate_policy` (pieza 1) |
| `internal/kickoff/` (`seal.go`, `types.go`) e `internal/cli/sdd_kickoff.go` | Modificado | Arranque con carpeta preparada; instantánea de `sdd.iso` en `kickoff.yaml` (P5 reducida) |
| `internal/sddstatus/openspec_archive_compose.go`, `internal/cli/sdd_archive_compose.go` | Modificado | Regla de escenarios y bloqueo ISO por matriz |
| `internal/sddstatus/` (`status.go`, `status_v2.go`) | Modificado | `coreReady` con bug sin delta; motivos de bloqueo ISO, cobertura, escenarios y matriz; proyección de `sdd.iso` |
| `internal/workspace/declarations.go` y fichero nuevo del bloque `sdd:` | Modificado y nuevo | Bloque `sdd:` con la clave `iso` (booleano) y validación cruzada: `sdd.iso: true` con `workflow.lane: odd` es una declaración contradictoria, al cargar y en `axiom workspace validate` (`workflow.go` no cambia) |
| `internal/specbaseline/` y la derivación de la traza | Modificado o nuevo (lo fija design) | Traza por `--first-parent` con la rama `-archive`; REQ existente por el `REQ-<n>.<m>` del nombre en el bug con requisito existente. La traza de integración de IDs (`trace.go`, `ids.go`) ya no interviene |
| Paquete o ficheros nuevos para `Cubre:`, la matriz y la resolución de referencias de prueba (ubicación en design) | Nuevo | Analizadores y validadores |
| `internal/multirole/barrier.go` | Sin cambio de comportamiento (verificar) | La matriz no escribe `verdict:`; la barrera sigue leyendo la primera línea `verdict:` |
| `internal/assets/skills/sdd-*/` (sobre todo `sdd-verify/references/report-format.md`), `_shared/sdd-status-contract.md`, `_shared/persistence-contract.md`, `_shared/sdd-orchestrator-sections.md` | Modificado | Formato de `Cubre:` y de la matriz, perfil de bug, excepción ISO |
| `internal/components/agentguidance/routing.go` (bloque ODD) | Modificado | Frase de D4: todo el trabajo de un proyecto ISO va por el SDD y los bugs con el perfil de bug |
| `testdata/golden/`, `AGENTS.md`, `GEMINI.md`, `.codex/AGENTS.md`, `docs/manual-de-inicio.md`, `axiom.example.yaml` | Regenerado o modificado | Copias versionadas, manual y ejemplo con la declaración ISO |
| `openspec/changes/carril-sdd-versatil/specs/` | Rehecho | Se borran las cinco deltas de §4.3; se recortan y renumeran las demás; se añade `sdd-independent-review` |
| `openspec/changes/carril-sdd-versatil/design.md` y `tasks.md` | Rehecho | Corresponden a la versión anterior (design con disparadores, mismo PR y P6) |

Ya no se tocan: `internal/branch/` (`verdict.go`, `archive_refusal.go`, `engine/done.go`, `engine/publication.go`), `internal/sddpublish/` ni `internal/docpaths/`.

## 7. Dependencias y orden de ejecución

- **Cadena de prosa del orquestador (C14): fusionada.** (1) arreglos T9 + adaptadores T1 (#229, #227) → (2) revisión T6 (#233, fusionada el 2026-11-10, `odd/tasks/upstream-revision-nativa.md:291-301`) → (3) guía T1 (#238) → (4) guía T2 (#239) y T3 (#240) (`odd/tasks/guia-orquestador-upstream.md:12-15`). La prosa de este cambio puede aplicarse ya; sigue yendo al final de la PR para regenerar los goldens una sola vez.
- **#235 y #249/#251 archivados.** Los MODIFIED parten de `openspec/specs` actual. `sdd-independent-review` pasa a ser capacidad modificada (§4.2).
- **#250 (`modelos-por-fase`)** está en la base; no comparte capacidades con este cambio.
- **Solapes con P6:** `internal/workspace/declarations.go` (declaración ISO y su validación cruzada con `workflow.lane` frente a la retirada del bloque `baseline:` por P6) y el bloque ODD de `routing.go` (D4 frente al localizador de G6.6). Quien llegue segundo rebasa.
- **Cambio siguiente:** P6 diseña la migración de proyectos a OpenSpec (D5) y la integración desde ODD en la spec viva (D4). El archivado en el mismo PR, si se retoma, es otro cambio.
- Reutiliza `kickoff.RoleTaskFiles` y la evidencia `pr_merged` de `internal/kickoff/integration.go`. Qué de `internal/specbaseline/` sirve a la traza lo decide design: la asignación de IDs provisional y definitivo no aplica a la baseline implícita.
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
| La validación cruzada rechaza un `axiom.yaml` que declare `iso: true` y `workflow.lane: odd`, y un proyecto con esa combinación dejaría de cargar. | Baja | El bloque `sdd:` es nuevo, así que ningún proyecto existente declara `iso`; el mensaje nombra las dos claves y la corrección. Un binario anterior ignora `sdd:` y no valida la combinación. `workflow.lane: ask` con ISO es válido y la doctrina no formula la pregunta de carril. |
| Un proyecto con `baseline:` declarada no recibiría comprobaciones de identificador ISO propias de su baseline. | Baja | El usuario ha confirmado que ningún proyecto real la declara y P6 retira el bloque; este cambio solo cubre `openspec/specs/`. |
| El texto de la marca `Sin impacto en la spec` es contrato con el carril ODD (P6); un cambio de redacción rompe a quien lo reutilice. | Baja | Texto exacto fijado en REQ-29.15 y repetido literal en la guía (REQ-29.17) y en la doctrina ODD (REQ-29.18), con pines de texto. |
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
- **Por datos.** Los cambios activos y archivados en el formato actual siguen siendo válidos con cualquier versión. Un cambio activo que use el perfil de bug sin delta (con requisito existente o sin impacto en la spec) no es completo para el binario anterior: habría que añadirle una delta o terminarlo antes de revertir. Un binario anterior ignora el bloque `sdd:` (claves de primer nivel desconocidas, `internal/workspace/loader.go:53-55`) y la clave nueva de `kickoff.yaml` según su decodificación (a confirmar en design).
- **Señal para revertir.** Un proyecto no ISO bloqueado por un validador nuevo; un `verify` que siga bloqueado por un rol `deferred`; un archivado ISO que pase sin matriz válida; una referencia de prueba existente rechazada.

## 11. Criterios de éxito (Success Criteria)

- [ ] **QA diferido.** Con un rol `blocking` aprobado y un rol `deferred` sin aprobar, `verify` queda desbloqueado y el rol diferido se informa; con un `blocking` sin aprobar sigue bloqueado.
- [ ] **Arranque.** `axiom sdd kickoff seal` funciona sin crear la carpeta a mano.
- [ ] **Sin ISO, sin cambios.** Un proyecto sin la declaración ISO recorre `continuous` y `checkpointed` con las mismas compuertas y paradas que hoy, y archiva por la rama `-archive` con un `verify-report.md` libre o ausente.
- [ ] **Escenarios.** En ISO, `archive-compose` rechaza, con motivo tipado, un requisito ADDED o MODIFIED sin escenarios.
- [ ] **ISO e identificador.** Con la declaración ISO, una delta con un requisito cuyo nombre no lleva un `REQ-<n>.<m>` válido, único y sin ambigüedad bloquea el cambio antes de design con motivo accionable; sin ISO no bloquea.
- [ ] **`sdd.iso` booleano.** `axiom.yaml` admite `sdd: {iso: true|false}`; cualquier otro valor o clave falla cerrado con mensaje en español.
- [ ] **Cobertura.** En ISO, `status` bloquea si algún REQ de la delta no aparece en ningún `Cubre:`.
- [ ] **Matriz.** En ISO, el archivado se bloquea si algún escenario no tiene fila, alguna fila apunta a un escenario inexistente, no tiene resultado o lo tiene fallido; si una fila automática apunta a una prueba que no existe en el árbol o no registra comando y código de salida; o si una fila manual carece de quién, cuándo y por qué.
- [ ] **Coherencia con #249.** La matriz convive con «Cumplimiento técnico» en el mismo informe; la barrera sigue leyendo el mismo veredicto; un hallazgo de revisión sigue sin bloquear el archivado, también en ISO.
- [ ] **Trazabilidad.** Para un cambio ISO archivado por la rama `-archive`, la cadena REQ → `Cubre:` → fila de la matriz → commit de integración del código → commit de archivado se reconstruye con `git log --first-parent` igual con squash, merge y rebase (prueba con los tres).
- [ ] **Informe de bug.** `## Informe de bug` con `Observado:` y `Esperado:` no vacías declara el bug (`Referencia:` opcional); un informe incompleto no prepara el cambio y el estado nombra la línea que falta.
- [ ] **Bug con requisito existente.** Un bug sin delta con `Cubre:` a un REQ inexistente bloquea en ISO; con un REQ existente llega al archivado; fuera de ISO queda preparado sin delta.
- [ ] **Bug sin impacto en la spec.** Con `## Informe de bug` y `## Impacto en la spec` (línea literal `Sin impacto en la spec` y `Motivo:` no vacío), sin delta y sin `Cubre:`, el artefacto de specs se cumple en todo proyecto, sea o no ISO; en ISO la matriz ausente o vacía es válida y el archivado no integra nada. Sin `Motivo:` (o vacío), con la sección sin `## Informe de bug`, o con otro texto en la línea, no queda preparado; junto a una delta se evalúa como cambio con delta; con `Cubre:` bloquea en ISO con motivo tipado y no bloquea fuera de ISO.
- [ ] **Instantánea (P5).** `kickoff seal` copia `sdd.iso` en `kickoff.yaml`; activar `iso` a mitad no exige la matriz a un cambio ya sellado y desactivarlo no deja de exigirla; un cambio sin instantánea cuenta como no ISO; `status` avisa con una nota no bloqueante de la discrepancia.
- [ ] **Retrosellado.** `kickoff seal --infer` no escribe la clave `iso` (serialización `omitempty`) ni lee `sdd:`; el cambio cuenta como no ISO y sigue exento de las guardas de rama.
- [ ] **Doctrina.** «Verification never gates archive» sigue vigente y declara la excepción ISO; REQ-28.3 la declara también; el bloque ODD expresa D4 (con ISO proyectado, todo el trabajo va por el SDD sin preguntar el carril ni crear `odd/tasks/<feature>.md`, y los bugs con el perfil de bug y sus tres rutas); los goldens están regenerados y en paridad.
- [ ] **Validación cruzada ISO y carril.** `sdd.iso: true` con `workflow.lane: odd` se reporta como error accionable en español, con las dos claves y la corrección, al cargar la configuración y en `axiom workspace validate` (salida 1) y rechaza el sellado sin escribir `kickoff.yaml`; `iso: true` con `lane: sdd`, `lane: ask` o sin `lane`, e `iso: false` con `lane: odd`, son válidos; el bloque `workflow:` y sus valores no cambian.
- [ ] **CI verde.** `go test` de los paquetes tocados y `go vet ./...` pasan; los ratchets de refusals y deadcode no empeoran.

## 12. Resolución de D4, D5 y D6 (auditoría de visión, P6)

| Id | Resolución |
|---|---|
| D4 | **Resuelta** (redacción de la tercera enmienda, 2026-11-12): **ODD integra en la spec viva; los proyectos ISO solo usan SDD.** En un proyecto ISO todo el trabajo va por el SDD (la doctrina lo remite y `axiom.yaml` rechaza `sdd.iso: true` con `workflow.lane: odd`), y los bugs, siempre con el perfil de bug (sin delta con requisito existente, sin impacto en la spec, o con delta como cambio normal). La integración desde ODD (verbo y revisión de impacto en ODD) queda para P6. |
| D5 | **Resuelta.** OpenSpec es el único formato de spec viva; todo proyecto que llega a Axiom migra a OpenSpec y no hay otros formatos de destino. La migración va en P6. |
| D6 | **Resuelta vía el modo ISO opcional.** Un proyecto declarado ISO deja las evidencias que pide el atributo de proceso PA 2.2 (gestión de productos de trabajo) de la familia ISO/IEC 330xx: productos de trabajo definidos y validados (propuesta, delta con al menos un escenario por requisito, design, tasks con `Cubre:`, matriz de verify con evidencia contrastable), revisiones registradas (las compuertas existentes en `checkpointed` y las revisiones de #249), trazabilidad requisito → tarea → verificación → commit (C8) e identificadores de requisito estables (`REQ-<n>.<m>`, REQ-29.3). La calidad de la especificación de requisitos se apoya en ISO/IEC/IEEE 29148. Los proyectos no ISO no quedan sujetos a estas evidencias. |

La hoja de ruta de la auditoría puede marcar D4 y D6 como resueltas (D6 para proyectos que declaren ISO) y D5 como resuelta (la migración de proyectos a OpenSpec va en P6).

## 13. Preguntas abiertas

Solo huecos reales. Las que bloquean, bloquean únicamente la spec del requisito que nombran; el resto de la spec puede avanzar.

| ID | Pregunta | Bloquea | Quién la cierra |
|---|---|---|---|
| Q1 | ¿Cuándo se publica el ID definitivo de un requisito ADDED hacia la herramienta externa? Hasta el archivado el ID es provisional (`internal/specbaseline/ids.go`). | Nada en este cambio | Usuario |
| Q6 | Si se quiere declarar el modelo de referencia de procesos y el nivel de capacidad objetivo (las evidencias se orientan a PA 2.2, nivel 2). | Nada | Usuario (opcional) |
| Q13 | Con la pieza 1, `verify` puede abrirse antes de que cierre el último rol activo. ¿El aviso de REQ-21.13 («queda abierta la verificación global») se mantiene al cerrar el último rol activo, se adelanta al cierre del último rol `blocking`, o se ajusta su texto? | Solo la delta de REQ-21.13/REQ-21.15 | Usuario (la spec propone) |
| Q14 | Formas de referencia de prueba que Go acepta en filas automáticas: ¿basta la forma general `ruta/fichero:símbolo` para cualquier lenguaje, más el atajo `TestX` en Go, o el cliente necesita otras (por ejemplo, identificadores de casos de una herramienta e2e)? | Solo el requisito de la matriz | Usuario |

### 13.1 Preguntas resueltas o que dejan de aplicar

| ID | Resolución |
|---|---|
| Q2 | **Superada.** El estilo sellado sigue siendo solo el eje de ritmo y no cuenta personas. |
| Q3b | **Resuelta** (2026-11-12). `sdd.iso` es booleano. Un bug se declara con la sección fija `## Informe de bug` (`Observado:` y `Esperado:` obligatorias, `Referencia:` opcional) en `proposal.md`, sin front matter; REQ-29.15. Ver C17. |
| Q3 | **Bloque nuevo `sdd:` con la clave `iso`.** Lista cerrada de claves y validación estricta tipo `unknownKeys`. No se toca `workflow:` ni REQ-2.1 de `workspace-topology`. Un binario anterior ignora las claves de primer nivel desconocidas (`loader.go:53-55`). `archive` y `design_triggers` salen con el recorte. |
| Q7 | **Descartada** (2026-11-11). |
| Q8 | **Solo en ISO.** La regla de al menos un escenario por requisito se aplica únicamente a cambios sellados ISO. |
| Q9, Q10 | **Dejan de aplicar**: el design proporcional y por rol sale del alcance. |
| Q11, Q11b | **Dejan de aplicar**: el archivado en el mismo PR sale del alcance; la rama `-archive` es el único modo. |
| P6 | **Deja de aplicar** con Q11. |
| Q12 | **PR única** con `size:exception` (preferencia permanente del usuario). |
| Baseline implícita | **Es la única baseline que este cambio especifica** (2026-11-12). El identificador de un requisito es el `REQ-<n>.<m>` de su nombre; no hay perfil ni `identity: id` que comprobar, porque ningún proyecto declara `baseline:` y P6 retira el bloque. Antes: «cuenta como `identity: id`» (C10, retirada). |
| P5 | **Instantánea de `sdd.iso` al sellar** (reducida). `axiom sdd kickoff seal` copia `sdd.iso` en `kickoff.yaml` con la semántica de sellado único (precedente: `docs_publication`, REQ-27.4). El valor sigue siendo del proyecto. Un cambio sin instantánea cuenta como no ISO. Cambiar `sdd:` solo afecta a los cambios sellados después; `status` avisa con una nota no bloqueante. El retrosellado no escribe `iso` (serialización `omitempty`) ni lee `axiom.yaml` (REQ-21.4; C19), así que el cambio cuenta como no ISO (resuelto por el usuario, 2026-11-12; antes «congela `iso: false`»). |

### 13.2 Supuestos visibles para la compuerta `spec`

| Supuesto | Dónde consta (orientativo) |
|---|---|
| En ISO, una fila de la matriz que apunta a un escenario inexistente invalida la matriz. | Requisito de la matriz (REQ-29.x) |
| En multirrol, la matriz vive en el `verify-report.md` consolidado de REQ-21.15; con el rol único `fullstack` es el mismo informe que el del rol (REQ-28.4). | Requisito de la matriz; delta de REQ-28.4 |
| En ISO, una matriz no válida cuenta como comprobación obligatoria fallida y el verify no es limpio (REQ-28.6). | Delta de `sdd-independent-review` |
| En la baseline implícita, el identificador de un requisito es el `REQ-<n>.<m>` de su nombre. En ISO, el identificador debe ser único (un ADDED no repite uno existente ni el de otro ADDED, salvo el propio requisito ya compuesto por un archivado interrumpido: mismo dominio, mismo nombre y mismo bloque byte a byte, H6), no ambiguo (un solo identificador distinto por nombre) y estable en un RENAMED (el nombre antiguo, si lo lleva, y el nuevo coinciden; un RENAMED puede añadirlo a un requisito que no lo tenía). | REQ-29.3, REQ-29.4 y REQ-29.14 |
| Un `## Informe de bug` incompleto (falta `Observado:` o `Esperado:`, o valor vacío) no cuenta como declaración; un cambio sin delta con una declaración incompleta no queda preparado y el estado nombra la línea que falta. Cada línea del informe lleva su valor en la misma línea. | REQ-29.15 |
| La marca `Sin impacto en la spec` se compara exacta (sensible a mayúsculas, sin texto añadido, ignorando solo el espacio en blanco inicial y final). Sin `Motivo:` o con motivo vacío, o sin `## Informe de bug`, no hay declaración válida. | REQ-29.15 |
| **Casos límite del bug sin impacto.** (1) Marca junto a una delta: se evalúa como cambio con delta y la sección se ignora, sin aviso. (2) Marca junto a un `Cubre:`: declaración contradictoria; **bloquea con motivo tipado solo en ISO** (la comprobación se hace cuando existen los ficheros de tareas, porque el `Cubre:` vive ahí) y no bloquea fuera de ISO, donde `Cubre:` es informativo (REQ-29.4). (3) `## Impacto en la spec` sin `## Informe de bug`: no queda preparado, con motivo que nombra el informe ausente. | REQ-29.15, REQ-29.19 |
| En ISO, un bug sin impacto en la spec no necesita matriz: ausente o presente sin filas es válida y de éxito, y una fila cualquiera la invalida (no hay escenarios afectados). | REQ-29.19, REQ-29.6, REQ-29.11 |
| Sin la asignación de IDs provisional y definitivo, la traza (REQ-29.9) no reutiliza `integration.yaml`; el design decide qué reutiliza de `internal/specbaseline/`. | REQ-29.9 |
| `workflow.lane: ask` con `sdd.iso: true` es una combinación válida; la doctrina de REQ-29.18 no formula entonces la pregunta de carril. Solo `workflow.lane: odd` con `iso: true` es contradictoria (antes: «el perfil de bug prevalece sobre `workflow.lane: odd`», supuesto retirado porque la combinación se rechaza). | REQ-29.1 en `sdd-iso-evidence`; REQ-29.18 en `organic-agent-trigger-rules` |
| `archive-compose` determina el cambio (por `--change` o por la ruta de la delta) para leer su instantánea; sin cambio determinable no aplica la regla de escenarios. | Requisito de escenarios en `sdd-archive-lifecycle` |
| La referencia de una fila automática se resuelve contra el árbol de trabajo en el momento de validar (verify en la rama del cambio, archivado en la rama `-archive`). | Requisito de la matriz |

Numeración: los requisitos nuevos usan la serie **REQ-29.x**. Las specs de `main` llegan hasta REQ-28 (`sdd-revision-independiente`, REQ-28.1 a REQ-28.14, archivado en #251); las referencias a REQ-28.x de esta propuesta que no son de `sdd-independent-review` (o de los cambios de `sdd-preflight-configuration`, `sdd-lane-kickoff` y `sdd-block-review-gates` que ese cambio introdujo) eran de la versión anterior y quedan sustituidas por REQ-29.x.

**IDs de la serie 29 tras la segunda enmienda (2026-11-12).** Sin renumerar: REQ-29.1 a REQ-29.18 se conservan; REQ-29.3 se reduce (identificador `REQ-<n>.<m>` en el nombre, sin `identity: id`); REQ-29.9 pierde su último escenario; REQ-29.14 pierde la resolución por perfil; REQ-29.15, REQ-29.16, REQ-29.5, REQ-29.6, REQ-29.11, REQ-29.13, REQ-29.17 y REQ-29.18 se amplían con la tercera ruta de bug o el retrosellado. **Nuevo: REQ-29.19** (bug sin impacto en la spec en ISO). **Retirados: ninguno entero.** Sale del cambio, además, el MODIFIED de REQ-24.46 de `spec-baseline-integration`. **Tercera enmienda (2026-11-12):** sin renumerar; REQ-29.1 gana la validación cruzada `sdd.iso`/`workflow.lane`, REQ-29.2 proyecta también la declaración contradictoria como inválida y REQ-29.18 pasa a remitir todo el trabajo ISO al SDD (cambia su título).
