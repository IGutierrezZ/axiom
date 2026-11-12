# Exploración: spec-viva-openspec (P6, O6 · G6.1–G6.7)

> Fase `explore` · rama `inc/spec-viva-openspec` · base `8a704c9e` · 2026-11-12. Solo lectura. Decisiones vinculantes del usuario: D4 (bugs e incrementos ODD integran en la spec viva, con «sin impacto» explícito), D5 (OpenSpec, `openspec/specs/`, es el único formato; todo proyecto migra). D6 va en `carril-sdd-versatil` (spec aprobada, sin implementar).

## 1. Estado actual por hueco (verificado en `main`)

| Hueco | Estado hoy | Efecto de D4/D5 |
|---|---|---|
| G6.1 Sin intake | Sigue abierto. No existe verbo ni flujo de análisis. `axiom archive coldstart` solo promueve un cambio ya archivado a `specs/` (`cmd/axiom/main.go:394-412`, `internal/livingdoc/service.go:187`). El manual describe la adopción como manual (`docs/manual-de-inicio.md:487-494`). | **Se reformula**: pasa a «migración a OpenSpec». |
| G6.2 Detector | Sigue abierto. `detectExistingSDD` solo mira `openspec/` y `.openspec/` (`internal/hub/detector.go:177-190`). No busca ADR, `docs/`, ni specs ajenas. `TechDetection` solo lleva `HasExistingSDD`/`SpecsRepository` (`internal/hub/types.go:68-71`). | **Se reformula**: el detector debe proponer *orígenes* a migrar. |
| G6.3 `init` sin bloques | Sigue abierto. `buildAxiomYamlWithRoles` solo escribe `workspace`, `roles`, `governance` (`internal/hub/init.go:371-438`). `axiom.example.yaml` no contiene `baseline`, `technical_context`, `workflow`, `git` ni `repositories`. | `baseline:` **desaparece**; queda `technical_context:` (y documentar `workflow:`/`sdd:` en el ejemplo). |
| G6.4 Más formatos | Abierto solo como carpeta por requisito u otros formatos (`declarations.go:49-75`). | **Moot** por D5: no hay otros destinos. |
| G6.5 ODD no integra | Exclusión explícita en `openspec/specs/project-technical-context/spec.md:68`. ODD no escribe nunca en `openspec/specs` (`routing.go:68`: «keep it exclusively in `odd/tasks/<feature-name>.md`»). | D4 lo invierte: hay que diseñar la integración. |
| G6.6 Localizador de contexto | Parcial. Paso 7 (`internal/components/agentguidance/routing.go:77`, REQ-24.58) dice «actualiza el contexto mantenido» sin ruta ni verificación. Pero `axiom sdd status --json` ya proyecta `technicalContext.maintained.{entry,paths}` con rutas absolutas **con o sin cambio** (`internal/sddstatus/workspace_declarations.go:32-35,78-100`). | Sigue vigente (el contexto técnico no es baseline ajena). |
| G6.7 Integración dirigida por modelo | Abierto: `integration: directed` y contexto mantenido los edita el modelo; `integrate verify` detecta después (`manual-de-inicio.md:496-498`). | `directed`/`register` **moot** por D5; el contexto mantenido sigue dependiendo del modelo. |

Hallazgo colateral: `docs/manual-de-inicio.md:510-514` cita `axiom odd create|status|promote`, pero el grupo `axiom odd` se retiró (INC-20, D1; `openspec/specs/odd-cli-commands/spec.md:13`) y `cmd/axiom/main.go` no tiene `case "odd"`. Deriva documental a corregir de paso.

## 2. Maquinaria de baseline ajena: inventario y consumidores

**Núcleo a retirar** (líneas no vacías, medidas con búsqueda):
- `internal/specbaseline/*.go` ≈ 5.575 de código (`plan.go` 1.717, `compose.go` 839, `verify.go` 810, `trace.go` 416, `gitview.go` 330, `ids.go` 300, `document.go` 273, `anchors.go` 239, `delta.go` 184, `profile.go` 84) y ≈ 8.200 de pruebas, más `testdata/reference/` (muestra de un cliente real: `01_Requisitos_Funcionales.md`, IDs `RF-`).
- `internal/cli/sdd_integrate.go` (417) y pruebas (≈ 990); `sdd_archive_compose.go` modos `--document`/`--in-place` (de 467; el modo `--canonical --delta` se conserva) y pruebas (≈ 850).
- `internal/workspace/declarations.go` (563; `BaselineConfig`, `BaselineDocument`, `RequirementProfile`, `:14-75`), `baseline_guard.go` (125, `GuardSpecsWrite`, `Ownership`), sus pruebas (≈ 760); `loader.go`, `validator.go`, `types.go`.
- `internal/sddstatus/workspace_declarations.go` (348; `projectBaseline`, `integratePrepareLine`, `declarationCauses`), `status.go`, `status_v2.go`; `internal/livingdoc/service.go` (`externalBaselineOf`, catálogo externo, `:79-91,115-127`); `internal/hub/init.go` (`BaselineProject`, `:189-205`); `internal/sddpublish/publish.go:345` (parámetro `baseline` de `docpaths.New`); `internal/knowledge/specs_root.go`; `internal/dashboard` (`archive_baseline_test.go`).
- Prosa: 16 ficheros de `internal/assets/skills` con 69 menciones (`sdd-archive` 9 + `references/project-baseline-integration.md`, `sdd-apply` 6, `sdd-spec` 6, `_shared/project-baseline-convention.md` 13, `sdd-status-contract`, etc.), `docs/manual-de-inicio.md:452-498`, goldens.
- Specs vivas: `spec-baseline-declaration` (15 requisitos) y `spec-baseline-integration` (19) pasan a REMOVED; tocan también `sdd-archive-lifecycle` (REQ-24.55–24.57), `project-technical-context` y `workspace-topology`.

**Hay que reubicar, no borrar** (el contexto técnico mantenido no es baseline ajena y sigue vigente):
- `contextdelta.go` (344: `ParseContextDelta`, `ContextScope.CheckDocument`), el bloque de contexto de `plan.go` (`:746`, `:1158-1214`, `:1266`, `:1312`: avisos y bloqueos de `context-impact.md`) y la parte de contexto de `verify.go` (`:283`, `:650`, REQ-24.57 «Sin impacto» contradicho por la edición) + `gitview.go` (vista git que usa la verificación). Hoy solo cuelgan de `sdd integrate prepare|verify`. Hay que decidir si esos verbos se reducen a «solo contexto» o si el chequeo pasa a `sdd status`/`archive`.
- `ids.go` (asignación provisional→definitiva) **no se puede portar tal cual**: opera sobre `PlannedDocument` y `RequirementProfile` (`ids.go:197,243`), exige familia de IDs ya sembrada (`seedContinuation`, `:285`). Para OpenSpec nativo habría que reescribirlo (≈ 150–250 líneas), y solo hace falta si se decide que Go asigna IDs (ver §4.4).

**Qué reutiliza `carril-sdd-versatil` de esta maquinaria**
- Proposal §3.1 pieza 7 y §7: «Reutiliza `trace.go` e `ids.go`»; REQ-29.9 enlaza ID provisional→definitivo «por `integration.yaml`»; REQ-29.3 / C10 exigen `identity: id` en `baseline:` declarada; REQ-29.14 consulta un REQ por ID en documentos con perfil.
- **Verificado: `integration.yaml` solo existe con baseline declarada.** `--in-place` rechaza sin `baseline:` (`internal/specbaseline/plan.go:1534`) y la verificación solo trata el caso por defecto aparte (`verify.go:580-594`). Con la baseline implícita `openspec/specs/` (la de Axiom y de cualquier proyecto OpenSpec) no hay `integration.yaml` ni asignación de IDs: los IDs son `(REQ-n.m)` en el nombre del requisito, elegidos por el autor (serie = nº de incremento; `REQ-27`, `REQ-28`, y `REQ-29` reservado a carril) y el compositor empareja por nombre exacto (`internal/sddstatus/openspec_archive_compose.go:33,293`). El único consumo de IDs en Go para OpenSpec es el indexador (`internal/livingdoc/indexer.go:17,136`, regex `\((REQ-[^\)]+)\)`).
- Consecuencia: las ramas de carril que dependen de baseline declarada (C10, escenario final de REQ-29.9, resolución por perfil de REQ-29.14) serán **código muerto** si D5 retira `baseline:`. Mantener solo la rama implícita (`REQ-<n>.<m>` del nombre) es suficiente para carril.
- Uso real de `baseline:` hoy: el `axiom.yaml` de este repo solo declara `workflow:` (`axiom.yaml:17`); ningún fixture del repo salvo `specbaseline/testdata`. **No se puede comprobar qué proyectos externos (clientes) la usan**: el fixture sugiere al menos un caso. Es un hueco para el usuario (ver §9).

## 3. Migración/intake a OpenSpec (G6.1, G6.2, G6.3)

Piezas disponibles: detector del hub (`hub/detector.go`), alta del dashboard (`/api/projects/init`, `internal/dashboard/assets/js/projects.js:611`, `hub.Initializer.Init`), skill `sdd-onboard`, `livingdoc.ColdStart` (prior art de adopción a partir de un cambio), `parseOpenSpecDelta` (formato delta, `openspec_archive_compose.go:181`).

Reparto de responsabilidades:

| Lo verifica Go (determinista) | Necesita modelo o humano |
|---|---|
| Inventario del origen (ficheros, encabezados, IDs ajenos por patrón). | Interpretar el origen y redactar requisitos con escenarios. |
| Formato OpenSpec del cambio de adopción (solo `## ADDED`, `### Requirement:`, `#### Scenario:`). | Repartir capacidades (dominios) y nombrarlas. |
| Unicidad de `REQ-n.m`, ≥ 1 escenario por requisito (regla ya prevista en carril, REQ-29.10, aquí aplicada siempre a la adopción). | Resolver ambigüedades y contradicciones del origen. |
| Matriz de cobertura origen→requisito: cada unidad del origen aparece en algún `Origen:` o en una lista `Descartado: <motivo>`. | Decidir si «descartado» es aceptable; aprobar la compuerta `spec`. |
| Que `Origen:` cite rutas y anclas que existen. | Juzgar la fidelidad semántica (Go no la puede probar). |

Enfoques:

| Enfoque | Pros | Contras | Esfuerzo |
|---|---|---|---|
| A. Skill + verificación Go sobre un **cambio SDD de adopción** (`ADDED` con `Origen:`), reutilizando el carril SDD, sus compuertas y `archive-compose`. | Reutiliza todo el ciclo (gates, archivado, rama `-archive`); trazabilidad y revisión humana ya existentes; ISO-compatible. | Un cambio SDD completo por migración puede pesar; `Origen:` es un campo nuevo que `archive-compose` debe ignorar o preservar. | Medio (700–1.000) |
| B. Verbo `axiom spec adopt` que escribe directo en `openspec/specs/` sin cambio. | Rápido. | Salta compuertas y aprobación humana; rompe «la spec viva solo cambia por un cambio revisado». | Medio |
| C. Importadores por formato (Word, Markdown con IDs…). | Más automático. | Va contra D5 (no otros formatos) y escala mal; el origen es heterogéneo. | Alto |

**Recomendado A**, con el intake repartido en tres capas: (1) CLI de análisis de solo lectura (nuevo verbo, p. ej. `axiom sdd adopt scan` / `check`, nombre abierto) que lista orígenes candidatos y verifica un cambio de adopción; (2) skill de adopción (variante de `sdd-onboard`/`sdd-spec`); (3) dashboard: tras el alta (P4) mostrar el informe de `scan` y el comando/instrucción para lanzar la adopción desde el agente (el dashboard no ejecuta modelos). G6.2 se cierra extendiendo `detectExistingSDD` con candidatos (`docs/`, `adr/`, `specs/`, `requisitos*`), sin escribir nada. G6.3: `init` propone `technical_context:` solo como sugerencia en el informe (no escribe por defecto) y `axiom.example.yaml` documenta los bloques.

## 4. ODD integra en la spec viva (G6.5, D4)

**4.1 Dónde vive la delta.** Opciones: (a) sección `## Spec viva` dentro de `odd/tasks/<feature>.md`; (b) fichero lateral. El fichero lateral en `odd/tasks/` es mala idea: el guardián y el listado tratan **todo** `odd/tasks/*.md` como una feature (`internal/branch/engine/guard_classify.go:52`, `odd_worktrees.go:43,154`, `dashboard/odd_listing.go`); uno en `openspec/changes/` crearía un artefacto SDD que el protocolo prohíbe (`routing.go:68`). **Recomendado (a)**. Detalle de formato: la delta usa encabezados H2 (`## ADDED Requirements`), que chocarían con los límites de sección; conviene un bloque vallado por capacidad bajo `### Capacidad: <dominio>` (el parser ya ignora vallas, `fencedRanges`, `openspec_archive_compose.go:91`). El listado ODD solo cuenta casillas (`odd_listing.go:30-31`), así que no se ve afectado.

**4.2 Verbo.** El grupo `axiom odd` se retiró por D1 («la creación de un documento vivo ODD pasa a ser un acto de agente con herramientas de fichero»); reintroducirlo contradice esa decisión. Alternativas de nombre: `axiom sdd integrate` (ya existe, se retira/reduce), `axiom spec integrate --odd <feature>` (grupo nuevo) o un modo de `sdd archive-compose`. Hueco de producto (§9). Núcleo: la primitiva pura `sddstatus.ComposeOpenSpecCanonicalSpecWithOptions` (`:293`). El guardián de escritura **no bloquea** `openspec/specs/` en una rama ODD (`guard_classify_test.go:46` «openspec sin changes es otro fichero»); `guardComposeDelta` solo actúa sobre deltas bajo `openspec/changes/<c>/` (`sdd_writer_guard.go:58-81`), así que una delta tomada del documento ODD y escrita con `--output` no choca.

**4.3 Cuándo corre.** En el paso 7 (cierre), sobre la rama de la feature, en el mismo commit de unidad de trabajo, sin compuertas ni matriz (consistente con «ODD no tiene paradas», REQ-27.17). Debe ser **idempotente**: tras un rebase la delta ya integrada no se reaplica (hoy `AlreadyIntegrated` existe solo en el plan de baseline ajena). Propuesta: registrar en el documento `Integrado: <capacidad>` + huella de la composición, o detectar «canónico ya contiene el requisito».

**4.4 IDs y conflictos.** Los IDs son `REQ-<incremento>.<n>` elegidos por el autor; hay colisión real entre ramas concurrentes (carril tuvo que renumerar de REQ-28 a REQ-29 tras #251, `proposal.md:9`). Opciones: serie propia por feature (`REQ-O<n>.m`), asignación definitiva en Go al integrar (el siguiente libre de la serie, reescribiendo referencias), o IDs sin serie. Conflicto con un cambio SDD concurrente: la composición falla cerrada si el nombre exacto ya no casa (`UnappliedDeltaError`), y el segundo en llegar rebasa y repite. Qué pasa con el prefijo funcional (`RF-USU-01`, Engram #2160) es **decisión abierta**: hoy el indexador exige `REQ-` (`indexer.go:17`).

**4.5 Revisión obligatoria de impacto en bugs (D4).** Registro propuesto en el documento ODD: sección `## Impacto en la spec` con una de dos formas, `Sin impacto: <motivo>` (mismo patrón que `context-impact.md`, reutiliza el analizador reubicado) o una delta (4.1). Comprobación Go: un verbo `check` que exige que la sección exista, esté bien formada y, si hay delta, que componga. Hoy ODD no tiene ningún enganche en Go al cierre (`axiom branch done` no lee el documento ODD): la comprobación sería nueva o solo doctrina.
Alineación con carril: su perfil de bug SDD exige `Cubre:` a un REQ existente para omitir la delta y `coreReady` requiere hoy `artifacts["specs"] == ArtifactDone` (`internal/sddstatus/status.go:703,861`). **Falta una vía «sin impacto» en SDD para un bug sin requisito relacionado** (el error nunca llegó a la spec): carril la cubre solo en ISO con `Cubre:`. Hay que decidir si carril acepta `Sin impacto: <motivo>` como alternativa a `Cubre:`.

## 5. Localizador del contexto técnico (G6.6)

El dato ya existe en `axiom sdd status --json` (`technicalContext.maintained.entry|paths`, rutas absolutas). Propuesta: (1) el paso 7 de `routing.go` manda leer ese campo y abrir `entry`; (2) la sección `## Contexto técnico` del documento ODD declara `Sin impacto: <motivo>` o `Entrada: <ruta>` (formato de `contextdelta.go`); (3) verificación Go opcional reutilizando `ContextScope.CheckDocument` y la vista git para confirmar que las rutas citadas cambian en el commit de la unidad de trabajo. Go solo verifica estructura y que se editó la ruta; si la edición es correcta sigue siendo del modelo (G6.7 persiste).

## 6. Solapes con `carril-sdd-versatil` y secuenciación

| Zona | Carril | P6 |
|---|---|---|
| `internal/workspace/declarations.go` | añade bloque `sdd:` con `iso` | retira `baseline:` |
| `internal/specbaseline/` (`trace.go`, `ids.go`) | reutiliza/modifica | borra el paquete |
| `internal/sddstatus` (`status.go`, `workspace_declarations.go`) | C10, bug sin delta, matriz | quita `projectBaseline`, causas de baseline |
| `routing.go` bloque ODD | REQ-29.18 (frase de D4) | integración ODD + localizador |
| Skills `sdd-*`, `_shared/*`, goldens | formato `Cubre:`, matriz | quita ramas de baseline, añade adopción |

El solape es grande, así que la retirada debe ir **después** de que carril se fusione (para no romperlo ni rebasar un borrado masivo). Pero carril construiría código que P6 borra. Secuenciación recomendada: (0) enmendar la spec de carril (barato, aún sin implementar) para quitar C10, el escenario `integration.yaml` de REQ-29.9 y la resolución por perfil de REQ-29.14, y para admitir `Sin impacto` en bugs (§4.5); (1) fusionar carril; (2) P6. Las piezas de P6 que no tocan lo anterior (adopción, ODD integrate) podrían ir antes, pero comparten `routing.go` y goldens: quien llegue segundo rebasa.

## 7. Tamaño y cortes (una PR con `size:exception`, un commit por unidad de trabajo)

| Unidad | Contenido | Autorado aprox. |
|---|---|---|
| U1 | Reubicar contexto técnico a paquete propio (`ContextDelta`, `ContextScope`, gitview, chequeos de `context-impact.md`) y decidir verbos | 600–900 (en gran parte movido) |
| U2 | Servicio de IDs OpenSpec (solo si Go asigna IDs) | 200–400 |
| U3 | Integración ODD: sección, verbo, idempotencia, doctrina `routing.go`, goldens | 600–900 |
| U4 | Revisión de impacto en bugs + localizador de contexto (G6.5/G6.6) | 300–500 |
| U5 | Adopción: detector (G6.2), verbo de análisis/verificación, skill, paso en dashboard, `axiom.example.yaml` (G6.3) | 1.000–1.600 |
| U6 | Retirada de `baseline:` con rechazo explícito y mensaje de migración, specs REMOVED, skills, manual, goldens | +300 añadido / ≈ −17.000 borrado |
| U7 | Documentación (manual §8–9, corregir `axiom odd` obsoleto) | 150–300 |

Previsión de líneas añadidas: ≈ 3.200–5.000; la retirada domina el diff por borrado. Riesgo de revisión: **alto** (la retirada mezcla borrado masivo y conducta nueva); mitigación: commits independientes y revertibles, retirada en el último commit.

## 8. Riesgos

- Retirar `baseline:` en silencio haría que `archive` escribiera una **segunda baseline** en `openspec/specs/` (hoy lo impide `GuardSpecsWrite`, `baseline_guard.go`): la retirada debe **rechazar** con error tipado y vía de migración (`axiom workspace validate`), no ignorar el bloque.
- Proyectos externos con `baseline:` (el fixture `testdata/reference` apunta a un cliente real): sin migración quedan bloqueados al archivar.
- Código de carril que depende de baseline declarada se escribe y se borra (§6).
- Colisión de IDs `REQ-n.m` entre ramas concurrentes y entre ODD y SDD (§4.4); idempotencia de la integración ODD tras rebase.
- ODD en proyectos ISO: carril remite los bugs al SDD, pero no dice nada de incrementos ODD no-bug (Engram #2160, pregunta 1).
- Go no verifica fidelidad semántica de la migración ni la corrección de la edición de contexto: el riesgo residual lo cubre la compuerta humana.
- `ComposeOpenSpecCanonicalSpecWithOptions` empareja por nombre exacto con el sufijo `(REQ-x.y)`: renumerar un ID rompe MODIFIED/REMOVED ya redactados.

## 9. Huecos de producto abiertos (no se inventan respuestas)

1. ¿Algún proyecto real (cliente) declara hoy `baseline:`? Decide si la retirada necesita una herramienta de migración y no solo un rechazo.
2. ¿Go asigna los IDs definitivos de los requisitos (provisional→definitivo) en OpenSpec, o los elige el autor como hoy? ¿Qué esquema de serie para ODD y para requisitos migrados?
3. ¿Los requisitos migrados conservan el ID funcional del origen (`RF-120`) y dónde? Hoy el indexador exige `REQ-` (`indexer.go:17`); `Origen:` lo conservaría como trazabilidad.
4. Nombre y grupo del verbo de integración ODD, dado que `axiom odd` se retiró por D1.
5. ¿Se prohíbe la integración desde ODD en proyectos ISO (todo por SDD) o se permite con delta?
6. ¿Se enmienda `carril-sdd-versatil` antes de implementarlo (quitar la rama de baseline declarada y aceptar `Sin impacto` en bugs) o se acepta que P6 borre ese código después?
7. ¿La migración de los `REQ-x.y` propios de Axiom entra en este cambio? (Engram #2160, pregunta 3.)
8. Alcance del contexto mantenido en ODD: ¿solo estructura (sección + ruta cambiada) o también contenido?

## Recomendación

Cambio único `spec-viva-openspec` con las unidades U1–U7, **después** de enmendar y fusionar carril. Enfoque A para la migración (cambio SDD de adopción con `Origen:` + verificación Go de formato, IDs, escenarios y cobertura), sección `## Spec viva` + `## Impacto en la spec` embebidas en el documento ODD para D4, y reubicación previa del contexto técnico antes de borrar `specbaseline`. Preparado para propuesta: **sí, una vez resueltos los huecos 1, 2, 4 y 6** (bloquean el diseño; el resto puede avanzar).
