# Propuesta: Spec viva única en OpenSpec (spec-viva-openspec)

> Fase `propose` · rama `inc/spec-viva-openspec` · base `8a704c9e` · 2026-11-12. Entrada: `explore.md` (Engram `sdd/spec-viva-openspec/explore`), auditoría de visión (`docs/audits/2026-11-03-vision-producto.md`, O6, D4, D5, D6 y fila P6) y la enmienda D4/D5 de `carril-sdd-versatil` (#255).
> Kickoff sellado: `continuous`, rol único `fullstack` (`blocking`). Pre-vuelo: ritmo automático, artefactos `hybrid`, entrega `single-pr` con `size:exception` (preferencia permanente del usuario: una PR, un commit por unidad de trabajo, nunca cadena).
> Numeración: los requisitos nuevos usan la serie **REQ-30.x**. Verificado: `openspec/specs/` llega a REQ-28 y el único cambio activo con serie propia, `carril-sdd-versatil`, reserva REQ-29; no hay ninguna aparición de REQ-30 o superior en el repositorio fuera de `testdata/`.

## 1. Propósito (Intent)

### 1.1 Problema

O6 pide que Axiom analice el repositorio de specs que trae un proyecto y que los agentes integren en su spec viva al archivar, también desde ODD. Hoy:

| Hueco | Estado verificado |
|---|---|
| G6.1 y G6.2: sin intake ni detección | No hay verbo de análisis; `detectExistingSDD` solo reconoce `openspec/` y `.openspec/` (`internal/hub/detector.go:177-190`). La adopción es manual (`docs/manual-de-inicio.md:487-494`). |
| G6.3: `init` y ejemplo mudos | `axiom.example.yaml` solo documenta `workspace`, `roles` y `governance`; no documenta `technical_context:`. |
| G6.4 y G6.7: formatos ajenos | Existe una maquinaria completa de baseline ajena (`baseline:`, `internal/specbaseline`, `axiom sdd integrate`, `archive-compose --document/--in-place`), ≈ 5.600 líneas de código y ≈ 8.200 de pruebas, que ningún proyecto real usa (decisión del usuario; el único caso es el fixture `internal/specbaseline/testdata/reference`). Con D5 sobra. |
| G6.5: ODD no integra | Exclusión explícita en `openspec/specs/project-technical-context/spec.md:68`; la doctrina ODD prohíbe escribir fuera de `odd/tasks/<feature>.md` (`internal/components/agentguidance/routing.go:68`). |
| G6.6: localizador del contexto | El paso 7 de ODD (REQ-24.58) dice «actualiza el contexto mantenido» sin ruta ni comprobación, aunque `axiom sdd status --json` ya proyecta `technicalContext.maintained.{entry,paths}` con o sin cambio (`internal/sddstatus/workspace_declarations.go:32-35,78-100`). |
| Deriva documental | `docs/manual-de-inicio.md:510-514` cita `axiom odd create|status|promote`, retirados en INC-20 (`openspec/specs/odd-cli-commands/spec.md`). |

### 1.2 Por qué ahora

D4 y D5 están cerradas por el usuario y P4 (alta de proyecto en la UI) está hecho (#243), así que el informe de adopción ya tiene dónde mostrarse. `carril-sdd-versatil` está aprobado pero sin implementar: es el momento de que no construya ramas sobre `baseline:` que este cambio retira.

### 1.3 Resultado esperado

- `openspec/specs/` es el **único** formato de spec viva. Un `axiom.yaml` con `baseline:` falla cerrado con un error tipado y una vía de migración; nunca se ignora en silencio.
- Un proyecto con documentación de requisitos previa la migra a OpenSpec con un **cambio SDD de adopción** que Go verifica (formato, IDs únicos, escenarios, cobertura origen → requisito) y que un humano aprueba.
- Un incremento o bug ODD integra su delta OpenSpec en `openspec/specs/` al cerrar, con un verbo idempotente y sin compuertas; todo bug ODD declara su impacto en la spec, aunque sea «Sin impacto en la spec» con motivo.
- El contexto técnico mantenido sigue funcionando igual en SDD, ahora en un paquete propio, y ODD recibe un localizador y una comprobación estructural.

## 2. Decisiones que la propuesta recoge

| ID | Decisión (usuario) | Efecto en el cambio |
|---|---|---|
| D4 | Bugs e incrementos ODD integran en la spec viva; todo bug se revisa con «Sin impacto en la spec» + motivo explícito cuando no la cambia. | Secciones `## Spec viva` y `## Impacto en la spec` en el documento ODD; verbo `axiom archive integrate --odd`. |
| D5 | OpenSpec es el único formato; los proyectos que llegan sin OpenSpec migran. | Retirada completa de la baseline ajena; verbo `axiom archive adopt`. |
| R1 | Ningún proyecto real declara `baseline:`. Se retira **entera** la maquinaria; un `baseline:` residual se rechaza con error tipado y guía de migración. El código de contexto técnico que vive ahí se **reubica**. La retirada es la **última** unidad y espera a que `carril-sdd-versatil` se fusione. | U1 reubica; U6 retira. |
| R2 | IDs como hoy: el autor elige `REQ-n.m`. Migración: una serie nueva por proyecto migrado. ODD: el siguiente número libre, y Go comprueba que no colisiona. El ID original de un requisito migrado se conserva en `Origen:`. Los IDs funcionales (`RF-xxx`) y los `REQ-x.y` propios de Axiom quedan fuera. | Sin servicio de asignación de IDs (la U2 de la exploración desaparece). |
| R3 | Los verbos van en el grupo existente `axiom archive`. | `axiom archive integrate --odd <feature>` y `axiom archive adopt`. |
| R4 | Las dos secciones ODD van embebidas en `odd/tasks/<feature>.md`, no en un fichero lateral (un lateral rompe el listado y el guardián ODD, `internal/branch/engine/guard_classify.go:52`). | Formato de sección en §5.2. |
| R5 | Segunda enmienda de `carril-sdd-versatil` (solicitada a la sesión de su design): quitar C10, el último escenario de REQ-29.9 y la resolución por perfil de REQ-29.14; añadir la salida «Sin impacto en la spec» a su perfil de bug SDD. | Dependencia y punto de coordinación (§8), no alcance de este cambio. |
| R6 | G6.6: la guía ODD apunta a `technical_context.maintained` vía la proyección de `axiom sdd status --json`, con verificación estructural. | Decisión de esta propuesta: **solo estructural** (§5.4). |
| R7 | G6.3: `axiom.example.yaml` documenta `technical_context:`; el bloque ISO `sdd:` solo si `carril-sdd-versatil` ya se ha fusionado; `baseline:` no se documenta. | U5 y U6. |

## 3. Alcance (Scope)

### 3.1 Dentro de alcance

1. **Contexto técnico en paquete propio.** Mover `ContextDelta`, `ContextScope`, la vista git y las comprobaciones de `context-impact.md` (hoy en `internal/specbaseline/contextdelta.go`, `plan.go:746,1158-1214,1266,1312`, `verify.go:283,650` y `gitview.go`) a un paquete propio, con la edición dirigida, la verificación nativa y la traza **restringidas al contexto**. Exponerlo en el grupo `axiom archive` (nombre propuesto: `axiom archive context check|prepare|verify|show`, pregunta abierta P1). Sin cambio de conducta para el contexto mantenido.
2. **Integración ODD en la spec viva.** Sección `## Spec viva` con la delta OpenSpec del incremento; verbo `axiom archive integrate --odd <feature>` sobre `sddstatus.ComposeOpenSpecCanonicalSpecWithOptions`; idempotente tras un rebase; sin compuertas; comprobación Go de que ningún `REQ-n.m` ADDED colisiona con `openspec/specs/` ni con las deltas de los cambios activos.
3. **Revisión de impacto de los bugs ODD.** Sección `## Impacto en la spec` obligatoria en los bugs, con «Sin impacto en la spec: <motivo>» o remisión a `## Spec viva`; `integrate --odd` la exige y rechaza un bug sin ella.
4. **Localizador del contexto en ODD (G6.6).** El paso 7 de la doctrina ODD manda leer `technicalContext.maintained.entry|paths` de `axiom sdd status --json`; `integrate --odd` informa de si la rama tocó las rutas mantenidas y **rechaza** que toque rutas `reference` (REQ-24.30).
5. **Migración a OpenSpec (G6.1, G6.2, G6.3).** `axiom archive adopt scan` (inventario de solo lectura de orígenes candidatos y sus unidades: ficheros, encabezados, IDs ajenos por patrón, candidatos a `technical_context`) y `axiom archive adopt check --change <c>` (verificación de un cambio SDD de adopción). Detector del hub extendido para proponer orígenes; informe en el dashboard tras el alta de P4 con la instrucción para lanzar la adopción desde el agente. Skill de adopción.
6. **Documentación.** `axiom.example.yaml` con `technical_context:`; manual §8–9 reescrito (migración, integración ODD, contexto); corrección de la deriva `axiom odd`.
7. **Retirada de la baseline ajena** (última, tras la fusión de `carril-sdd-versatil`): bloque `baseline:` con rechazo tipado y guía de migración, `internal/specbaseline` (tras reubicar lo que siga en uso), `axiom sdd integrate`, modos `--document`/`--in-place` de `axiom sdd archive-compose` (se conserva `--canonical --delta`), `BaselineConfig`, `baseline_guard.go`, proyecciones de `baseline` en el status, catálogo externo de livingdoc, `BaselineProject` del hub, parámetro `baseline` de `docpaths.New`, ramas de prosa en skills y goldens; specs `spec-baseline-declaration` y `spec-baseline-integration` a REMOVED con reubicación de los requisitos que sobreviven (§4.3). Incluye la negativa de `integrate --odd` en proyectos ISO (P2), que necesita el bloque `sdd:` de carril.

### 3.2 Fuera de alcance

- IDs funcionales (`RF-xxx`) como identidad de requisito: cambio aparte. El indexador sigue exigiendo `REQ-` (`internal/livingdoc/indexer.go:17`).
- Migrar los `REQ-x.y` propios de Axiom.
- Asignación de IDs por Go (provisional → definitivo) en OpenSpec.
- Importadores por formato (Word, hojas de cálculo) y cualquier formato de destino distinto de OpenSpec.
- Que Go juzgue la fidelidad semántica de una migración o el contenido de una edición del contexto (G6.7 persiste: lo cubre la compuerta humana o la revisión).
- Enganche de Go que fuerce el cierre ODD (`axiom branch done` no lee el documento ODD); la ejecución de `integrate --odd` en el cierre es doctrina.
- Que el dashboard ejecute modelos: solo muestra el informe y la instrucción.
- La segunda enmienda de `carril-sdd-versatil` y cualquier pieza de su alcance.
- `axiom init` escribiendo `technical_context:` por defecto: solo se propone en el informe de `adopt scan`.
- Push, PR o merge desde Axiom.

## 4. Capacidades (Capabilities)

> Contrato con `sdd-spec`. Baseline: `openspec/specs/` (este repositorio no declara `baseline:`). Requisitos nuevos en la serie REQ-30.x.

### 4.1 Nuevas capacidades

- `odd-living-spec-integration`: secciones `## Spec viva` y `## Impacto en la spec` del documento ODD (formato, analizador); `axiom archive integrate --odd <feature>` (composición sobre `openspec/specs/`, idempotencia, comprobación de colisión de IDs, motivos tipados, sin compuertas); obligación del impacto en los bugs; comprobación estructural del contexto (informe de rutas mantenidas, rechazo de rutas `reference`); negativa en proyectos ISO (P2).
- `openspec-adoption`: `axiom archive adopt scan` y `adopt check`; formato del cambio de adopción (solo `## ADDED Requirements`, `Origen:` por requisito con ruta, ancla e ID original, lista `Descartado: <motivo>` para unidades del origen que no se migran); verificaciones Go (formato, unicidad de `REQ-n.m` en el cambio y frente a `openspec/specs/`, al menos un escenario por requisito, `Origen:` que resuelve a ruta y ancla existentes, cobertura completa del inventario); la compuerta `spec` del proyecto que migra como aprobación humana.

### 4.2 Capacidades modificadas

- `project-technical-context`: absorbe, restringidos al contexto, la edición dirigida, la verificación nativa y la traza (hoy REQ-24.50 a REQ-24.52 de `spec-baseline-integration`) y la regla de almacén de ficheros (parte de contexto de REQ-24.62); REQ-24.26 deja de citar `baseline.documents`; la exclusión de ODD (línea 68) se sustituye por la remisión a `odd-living-spec-integration`; superficie CLI del contexto en `axiom archive`.
- `sdd-archive-lifecycle`: REQ-24.55 deja de usar IDs provisionales (escenario `RF-9001` → `RF-008` fuera) y remite al mecanismo reubicado; REQ-24.56 pierde las causas de baseline (c, d y la parte de baseline de f y g) y gana la causa `baseline:` retirada; recibe las reglas de composición OpenSpec que sobreviven (REQ-24.38, REQ-24.45 y REQ-24.46 en lo que aplique a `--canonical --delta`; la spec decide).
- `organic-agent-trigger-rules`: REQ-24.58 amplía el paso de cierre con el localizador (`axiom sdd status --json`) y la llamada a `axiom archive integrate --odd`; la regla «keep it exclusively in `odd/tasks/<feature-name>.md`» admite la integración en `openspec/specs/`; determinismo del renderizado intacto.
- `workspace-topology`: rechazo tipado de un bloque `baseline:` residual en el cargador, `axiom workspace validate`, el status y el archivado, con guía de migración (`axiom archive adopt`).
- `multi-project-hub`: el detector propone orígenes de migración y candidatos a `technical_context`; el alta de P4 muestra el informe de `adopt scan` y la instrucción de adopción.
- `living-documentation`: solo si la spec comprueba que sus requisitos enumeran los subcomandos de `axiom archive` (ayuda del grupo); el catálogo deja de informar de baseline externa.
- `sdd-orchestrator-assets`: skills `sdd-*` sin ramas de baseline; `_shared/project-baseline-convention.md` reducido a contexto técnico (lo citan las instrucciones del orquestador, secciones 5 y 6, así que cualquier renombrado regenera goldens); skill o referencia de adopción; paridad entre runtimes.

### 4.3 Capacidades retiradas

| Capacidad | Destino de sus requisitos |
|---|---|
| `spec-baseline-declaration` (15) | REMOVED entera. REQ-24.62 se reubica en `project-technical-context` (solo contexto); REQ-24.17/24.18 los sustituye el rechazo tipado de `workspace-topology`. |
| `spec-baseline-integration` (19) | REMOVED entera. REQ-24.50–24.52 → `project-technical-context`; REQ-24.38, 24.45 y 24.46 (lo aplicable a OpenSpec) → `sdd-archive-lifecycle`; REQ-24.59 desaparece sin sucesor. |

## 5. Enfoque (Approach)

### 5.1 Reubicar antes de borrar

El contexto técnico mantenido no es baseline ajena y sigue vigente. U1 lo extrae a un paquete propio con su CLI en `axiom archive`; mientras `axiom sdd integrate` exista (hasta U6), delega la parte de contexto en ese paquete, de modo que las skills cambian de verbo en U6 sin ventana rota. Así la retirada final es un borrado limpio.

### 5.2 Integración ODD (enfoque embebido)

- `## Spec viva` contiene un bloque vallado por capacidad bajo `### Capacidad: <dominio>` con la delta OpenSpec (los H2 de la delta, `## ADDED Requirements`, no pueden ir sueltos porque cerrarían la sección; el compositor ya ignora vallas, `openspec_archive_compose.go:91`). El listado ODD solo cuenta casillas (`internal/dashboard/odd_listing.go:30-31`), así que no se ve afectado.
- `axiom archive integrate --odd <feature>` lee `odd/tasks/<feature>.md` de la rama actual, valida las secciones, comprueba colisiones de `REQ-n.m` ADDED (con `openspec/specs/` y con las deltas de `openspec/changes/*/specs/` activas; el error nombra el siguiente número libre), compone con `ComposeOpenSpecCanonicalSpecWithOptions`, escribe `openspec/specs/<capacidad>/spec.md` y regenera el índice si procede (design). El guardián de escritura no bloquea `openspec/specs/` en una rama ODD (`guard_classify_test.go:46`; `guardComposeDelta` solo actúa sobre `openspec/changes/<c>/`).
- **Idempotencia**: si la spec canónica ya contiene cada requisito de la delta con el mismo cuerpo (ADDED/MODIFIED) o ya no contiene los REMOVED, el verbo termina sin cambios y con éxito; un cuerpo divergente es error tipado. Tras un rebase sobre un cambio SDD concurrente, la composición falla cerrada si el nombre exacto ya no casa (`UnappliedDeltaError`) y el autor corrige la delta.
- Corre en el paso 7, en el commit de la unidad de trabajo que cierra la feature, sin paradas (REQ-27.17).

### 5.3 Impacto de los bugs

`## Impacto en la spec` admite exactamente dos formas: «Sin impacto en la spec: <motivo>» (motivo no vacío) o «Ver `## Spec viva`» con la delta presente. `integrate --odd` exige la sección cuando el documento es un bug (cómo se marca: pregunta P3) y comprueba su forma. Misma semántica que la salida «Sin impacto en la spec» que la segunda enmienda pide para el perfil de bug SDD de carril, para que ambos carriles hablen igual.

### 5.4 Localizador del contexto: solo estructural

Decisión: **verificación estructural, sin verificación de contenido.** El paso 7 manda leer `technicalContext.maintained.entry|paths` de `axiom sdd status --json` y abrir `entry`. `integrate --odd` calcula los ficheros cambiados de la rama frente a su base: **rechaza** cambios en rutas `reference` y **informa** (no bloquea) de si tocó rutas `maintained`. No se exige un marcador «sin impacto en el contexto» (añadiría una tercera sección; pregunta P4). Juzgar si la edición es correcta sigue siendo del modelo y de la revisión.

### 5.5 Migración a OpenSpec (enfoque A de la exploración)

Un cambio SDD normal de adopción, reutilizando kickoff, compuertas, `archive-compose --canonical --delta` y la rama `-archive`:

| Go (determinista) | Modelo y humano |
|---|---|
| `adopt scan`: inventario de orígenes y unidades; candidatos a `technical_context`. | Interpretar el origen, redactar requisitos con escenarios. |
| `adopt check`: solo `ADDED`; `### Requirement:` con `(REQ-n.m)` único; ≥ 1 escenario; `Origen:` resoluble; cada unidad del inventario en algún `Origen:` o en `Descartado:`. | Repartir y nombrar capacidades; resolver ambigüedades; aprobar la compuerta `spec`. |

El inventario que `scan` deja en la carpeta del cambio es la referencia de cobertura (nombre y formato en design). `Origen:` es texto del cuerpo del requisito: la composición lo conserva sin tratamiento especial. El detector del hub reutiliza `scan` sin escribir nada; el dashboard muestra su informe tras el alta.

### 5.6 Retirada con rechazo tipado

Un `baseline:` residual no se ignora: si se ignorase, `archive` escribiría una segunda spec viva en `openspec/specs/` (hoy lo impide `GuardSpecsWrite`). Se convierte en una causa tipada (p. ej. `baseline_retired`, nombre en design) en el cargador, `axiom workspace validate`, el status y el archivado, con la continuación «retira `baseline:` y migra con `axiom archive adopt`». Antes de borrar `internal/specbaseline`, U6 reubica lo que `carril-sdd-versatil` haya dejado en uso allí (su traza `--first-parent`, `trace.go`).

## 6. Áreas afectadas

| Área | Impacto | Qué cambia |
|---|---|---|
| `internal/specbaseline/` (`contextdelta.go`, partes de `plan.go`, `verify.go`, `gitview.go`) | Movido (U1) y eliminado (U6) | Contexto a paquete nuevo; resto del paquete y `testdata/reference` borrados |
| Paquete nuevo de contexto técnico (nombre en design, p. ej. `internal/techcontext`) | Nuevo | Delta de contexto, alcance, vista git, edición dirigida, verificación y traza |
| Paquete nuevo de spec ODD y de adopción (ubicación en design; candidatos `internal/livingdoc` o paquetes propios) | Nuevo | Analizador de secciones ODD, colisión de IDs, idempotencia; inventario y `adopt check` |
| `cmd/axiom/main.go`, `internal/cli/` (nuevos `archive_integrate.go`, `archive_adopt.go`, `archive_context.go`; `sdd_integrate.go` y modos de `sdd_archive_compose.go` eliminados) | Nuevo, modificado y eliminado | Verbos del grupo `axiom archive`; retirada de `sdd integrate` |
| `internal/sddstatus/` (`openspec_archive_compose.go` reutilizado; `workspace_declarations.go`, `status.go`, `status_v2.go`) | Modificado | Sin `projectBaseline` ni causas de baseline; causa `baseline:` retirada |
| `internal/workspace/` (`declarations.go`, `baseline_guard.go`, `loader.go`, `validator.go`, `types.go`) | Modificado y eliminado | Sin `BaselineConfig`; rechazo tipado |
| `internal/livingdoc/service.go`, `internal/knowledge/specs_root.go`, `internal/sddpublish/publish.go`, `internal/docpaths/` | Modificado | Sin catálogo externo ni parámetro `baseline` |
| `internal/hub/` (`detector.go`, `types.go`, `init.go`) | Modificado | Orígenes de migración y candidatos a contexto; sin `BaselineProject` |
| `internal/dashboard/` (alta de proyecto, `assets/js/projects.js`, `archive_baseline_test.go`) | Modificado | Informe de `adopt scan` tras el alta |
| `internal/components/agentguidance/routing.go` | Modificado | Paso 7: localizador, `integrate --odd`, excepción de `openspec/specs/` |
| `internal/assets/skills/` (`sdd-archive`, `sdd-apply`, `sdd-spec`, `sdd-propose`, `_shared/project-baseline-convention.md`, `_shared/sdd-status-contract.md`, etc.; skill o referencia de adopción) | Modificado y nuevo | Sin baseline; contexto con verbos nuevos; adopción |
| `testdata/golden/`, `AGENTS.md`, `GEMINI.md`, `.codex/AGENTS.md` | Regenerado | Con `-update`, nunca a mano |
| `docs/manual-de-inicio.md`, `axiom.example.yaml` | Modificado | Manual §8–9, deriva `axiom odd`, `technical_context:` |
| `openspec/changes/spec-viva-openspec/specs/` | Nuevo | Deltas de §4 |

## 7. Unidades de trabajo y orden

Una sola PR con `size:exception`; un commit Conventional por unidad, con sus pruebas, su delta y su doctrina, revertible por separado.

| Unidad | Contenido | Depende de |
|---|---|---|
| U1 | Reubicar el contexto técnico y su CLI en `axiom archive`; `sdd integrate` delega en él. | — |
| U2 | Integración ODD: sección `## Spec viva`, `axiom archive integrate --odd`, colisión de IDs, idempotencia; doctrina del paso 7. | — |
| U3 | `## Impacto en la spec` para bugs y comprobación estructural del contexto (G6.6). | U1 (vista git y alcance), U2 |
| U4 | Adopción: `adopt scan` y `adopt check`, detector del hub, informe en el dashboard tras el alta, skill de adopción. | — |
| U5 | Documentación: manual §8–9, deriva `axiom odd`, `axiom.example.yaml` con `technical_context:`. | U1–U4 |
| U6 | Alineación con carril y retirada: negativa ISO de `integrate --odd` (P2), `sdd:` en el ejemplo, rechazo tipado de `baseline:`, borrado de la maquinaria, specs REMOVED con reubicaciones, skills y goldens. | **Fusión de `carril-sdd-versatil`** con su segunda enmienda; U1–U5 |

U6 no empieza hasta que `carril-sdd-versatil` esté en `main`: hoy su propuesta reutiliza `internal/specbaseline/trace.go` e `ids.go` (§3.1 pieza 7) y, sin la enmienda, construiría C10 y las ramas por perfil sobre `baseline:`. Si la PR de este cambio está lista antes, U1–U5 pueden esperar en la rama o fusionarse sin U6 solo con decisión expresa del usuario (la preferencia es una PR).

## 8. Dependencias y coordinación

- **`carril-sdd-versatil`**: (a) segunda enmienda (R5) antes de su implementación; (b) su fusión antes de U6; (c) solapes en `internal/workspace/declarations.go`, `internal/sddstatus/`, el bloque ODD de `routing.go` (su REQ-29.18) y goldens: quien llega segundo rebasa. Si carril deja la traza en `internal/specbaseline/trace.go`, U6 la reubica.
- **P4** (#243, hecho): el informe de adopción se cuelga del alta existente (`/api/projects/init`, `hub.Initializer.Init`).
- Restricciones de trabajo: sin compilar binarios sueltos (solo `go test` acotado por paquete), commits Conventional sin atribución de IA, RDD fuera del SDD.

## 9. Riesgos

| Riesgo | Prob. | Mitigación |
|---|---|---|
| Retirar `baseline:` en silencio haría que `archive` escribiera una segunda spec viva. | Media | Rechazo tipado en cargador, validate, status y archivado; prueba con un `axiom.yaml` que lo declara. |
| Un proyecto externo no conocido declara `baseline:`. | Baja (usuario: ninguno) | El rechazo nombra la migración con `axiom archive adopt`; el binario anterior sigue disponible. |
| Diff dominado por el borrado (≈ −17.000) mezclado con conducta nueva. | Alta | Retirada en el último commit; U1 mueve sin cambiar conducta; revisión por commits. |
| Colisión de `REQ-n.m` entre ramas ODD y SDD concurrentes. | Media | Comprobación Go contra `openspec/specs/` y deltas activas; tras rebase, el segundo renumera y repite. |
| Renumerar un ID rompe MODIFIED/REMOVED ya redactados (el compositor empareja por nombre exacto). | Media | Error tipado `UnappliedDeltaError`; el autor corrige la delta. |
| El cierre ODD no ejecuta `integrate --odd` (solo doctrina). | Media | Paso 7 explícito y determinista; el verbo es idempotente y puede repetirse antes de la PR. |
| Go no prueba la fidelidad semántica de una migración. | Media | Cobertura completa del inventario + compuerta `spec` humana; `Descartado:` exige motivo. |
| Romper el contexto técnico al reubicarlo. | Baja | U1 conserva las pruebas existentes de contexto y las mueve con el código. |
| Regeneración de goldens y paridad entre runtimes. | Media | Prosa y goldens una vez por unidad que toca doctrina; pines de paridad. |
| Carril tarda en fusionarse y bloquea U6. | Media | U1–U5 no dependen de carril; la decisión de entregar sin U6 es del usuario. |

## 10. Entrega (Review Workload Forecast)

Previsión no medida, orientativa (líneas autoradas, sin goldens):

| Unidad | Añadidas | Borradas |
|---|---|---|
| U1 (en gran parte movido) | 900–1.300 | 700–1.000 |
| U2 | 600–900 | 50 |
| U3 | 250–400 | — |
| U4 | 1.000–1.500 | 50 |
| U5 | 150–300 | 100 |
| U6 | 200–400 | ≈ 16.000–17.500 |
| **Total** | **≈ 3.100–4.800** | **≈ 17.000–18.700** |

```
Decision needed before apply: No
Chained PRs recommended: No
400-line budget risk: High
```

Estrategia: una PR con `size:exception`, un commit por unidad, U6 al final.

## 11. Plan de reversión

- **Por unidad**: cada unidad es un commit independiente; `git revert` del commit de U6 restaura la maquinaria de baseline sin tocar U1–U5 (U1 deja `sdd integrate` delegando en el paquete reubicado).
- **Por PR**: `git revert` del merge, o quedarse en la release anterior (`go install` de la etiqueta previa en Windows). No hay migración de datos de Axiom.
- **Por datos de proyecto**: las integraciones ODD y las adopciones son commits normales en `openspec/specs/` del proyecto, revertibles con git. Las secciones nuevas del documento ODD son texto que un binario anterior ignora.
- **Por doctrina**: `axiom sync` con el binario revertido reproyecta las skills y la guía anteriores.
- **Señales para revertir**: un archivado SDD con contexto mantenido que deja de integrar o de detenerse como hoy; `integrate --odd` que reaplica una delta ya integrada; un `axiom.yaml` sin `baseline:` rechazado.

## 12. Criterios de éxito

- [ ] **Contexto intacto.** Un cambio SDD con `technical_context.maintained` integra, se detiene sin delta y respeta «Sin impacto» igual que hoy, con el paquete reubicado y el verbo de `axiom archive`.
- [ ] **ODD integra.** `axiom archive integrate --odd <feature>` aplica la delta de `## Spec viva` a `openspec/specs/`; repetido tras un rebase no cambia nada y termina con éxito.
- [ ] **Colisión.** Un `REQ-n.m` ADDED que ya existe en `openspec/specs/` o en una delta activa se rechaza con motivo tipado y el siguiente número libre.
- [ ] **Bugs.** Un bug ODD sin `## Impacto en la spec`, o con «Sin impacto en la spec» sin motivo, se rechaza; con motivo o con delta, pasa.
- [ ] **Contexto en ODD.** El paso 7 renderizado cita `technicalContext.maintained` de `axiom sdd status --json`; `integrate --odd` rechaza cambios en rutas `reference` e informa de las `maintained`.
- [ ] **Adopción.** `adopt scan` lista orígenes sin escribir; `adopt check` rechaza un cambio de adopción con un MODIFIED, un ID duplicado, un requisito sin escenario, un `Origen:` que no resuelve o una unidad del inventario sin cubrir ni descartar; un cambio válido archiva con `archive-compose --canonical --delta` y conserva `Origen:`.
- [ ] **Hub y dashboard.** El detector propone orígenes y candidatos a `technical_context`; el alta muestra el informe y la instrucción de adopción.
- [ ] **Retirada.** No quedan `internal/specbaseline`, `axiom sdd integrate` ni `--document/--in-place`; un `axiom.yaml` con `baseline:` falla con el error tipado y la guía de migración en validate, status y archivado.
- [ ] **Docs.** `axiom.example.yaml` documenta `technical_context:` y no `baseline:`; el manual no cita `axiom odd`.
- [ ] **CI verde.** `go test` de los paquetes tocados y `go vet ./...` pasan; goldens regenerados y en paridad; los ratchets de refusals y deadcode no empeoran.

## 13. Preguntas abiertas

No bloquean la spec salvo donde se indica; cada una lleva un valor por defecto propuesto que la spec puede asumir como supuesto visible.

| ID | Pregunta | Defecto propuesto | Bloquea |
|---|---|---|---|
| P1 | Nombre de la superficie CLI del contexto técnico tras retirar `axiom sdd integrate`. | `axiom archive context check|prepare|verify|show` | Solo el requisito de la CLI de contexto |
| P2 | ¿Se prohíbe `integrate --odd` en proyectos ISO? | Sí: negativa tipada que remite al SDD (bugs por el perfil de bug, incrementos como cambio), porque ODD no deja `Cubre:` ni matriz. Se implementa en U6, cuando existe `sdd.iso`. | Solo ese requisito |
| P3 | ¿Cómo sabe Go que un documento ODD es un bug? | Línea de cabecera `Tipo: bug`. Alternativa: exigir `## Spec viva` o `## Impacto en la spec` a **toda** feature que cierra, sin clasificar. | El requisito de impacto de bugs |
| P4 | ¿El contexto en ODD exige un marcador explícito «sin impacto en el contexto»? | No: solo informe de rutas `maintained` y rechazo de rutas `reference` (sin tercera sección). | Nada |
| P5 | Alcance de la comprobación de colisión de IDs. | `openspec/specs/` y deltas de los cambios activos en `openspec/changes/` (sin `archive/`). | Nada |
| P6 | En `multirepo`, ¿`adopt` y `integrate --odd` operan sobre el repositorio de specs (planning home)? | Sí, sobre `workspaceRoot`; el documento ODD de un multirepo ya vive en el repositorio de specs. | Nada |
