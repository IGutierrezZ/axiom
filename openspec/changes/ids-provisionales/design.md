# Design: IDs provisionales, número asignado al integrar y declaración de impacto en `status` (C-03-b)

## Enfoque técnico

El cambio introduce una segunda clase de token de identificador, el **provisional**, con forma propia en el puerto `internal/reqid`, y un **plan único de asignación** que convierte provisionales en definitivos antes de escribir nada. El plan vive en `internal/livespec`, que pasa a ser el motor de integración de los dos carriles: ODD (`axiom archive integrate --odd`) y SDD (verbo nuevo `axiom sdd archive-integrate`). Los dos comparten el plan, la clasificación de lo ya integrado, la sustitución por token completo, la composición nativa de `openspecdoc` y un escritor atómico de conjuntos de ficheros generalizado desde el `Apply` actual.

```
reqid (hoja)          Provisional, IDMap.Substitute, Allocator, SeriesOf, Scan con máscara
   ▲
openspecdoc (hoja)    UnknownRequirementIDError, ProvisionalIDError, ParseChangeSpec, Declared
   ▲
isoevidence           RequirementKeys (válido|provisional), RewriteCubre, RewriteMatrix
   ▲
livespec              Trace, PlanIDs, classify compartido, PlanIntegration (ODD), PlanChange (SDD), writeSet
   ▲
cli                   archive integrate --odd [--check]; sdd archive-integrate [--check]; archive-check
sddstatus             decorateRequirements → status.requirements (usa solo openspecdoc + reqid)
```

Fuentes: `proposal.md` y las seis deltas de `specs/` (REQ-47.1 a REQ-47.10; MODIFIED REQ-21.17, REQ-30.2, REQ-30.3, REQ-30.4, REQ-30.22, REQ-29.3, REQ-29.4, REQ-24.58). La spec y el design están alineados tras la revisión de cobertura (sección final).

## Revalidación contra `main` (`814b3ce5`)

La rama `inc/ids-provisionales` está rebasada sobre `814b3ce5`. Entre `3c55c493` (base del primer design) y `814b3ce5` se fusionaron C-08-d (seis agentes), C-08-e (GGA y telemetría), C-08-i (CI por jobs), retirar-rdd, retirar-gentle-ai, skills-sin-avisos y los textos de estilo de ejecución. Comprobado:

| Objeto | Resultado |
|---|---|
| Specs vivas de las seis capacidades del cambio | Sin cambios (`git diff 3c55c493..814b3ce5 -- openspec/specs/<cap>` vacío). Último cambio de cada una (`ae8cfc09`, `efb9ec2b`) anterior a `3c55c493`. Los ocho MODIFIED conservan todos los escenarios vivos |
| Serie REQ-47 | Solo aparece en `openspec/changes/ids-provisionales/` (`rg "REQ-47" openspec/specs openspec/changes` sin otros resultados); la serie máxima viva es 46 |
| `internal/{reqid,openspecdoc,isoevidence,livespec,sddstatus,kickoff}`, `internal/cli/sdd_archive_*`, `archive_integrate.go`, `sdd_trace.go`, `cmd/axiom/main.go`, skills `sdd-archive`, `sdd-spec`, `_shared/sdd-status-contract.md`, `_shared/sdd-iso-convention.md` | Sin cambios |
| `internal/components/agentguidance` | `routing.go` sin cambios (paso 7 en `routing.go:77`); `routing_test.go` solo cambia `supportedAgentCount` 17 → 11; `inject.go` pierde código de agentes retirados, sin efecto aquí |
| `internal/workspace/docs_pin_test.go` | Sin cambios; fija `docs/manual-de-inicio.md` (no el paso 7) |
| Goldens | 63 ficheros (eran 89); ninguno contiene `Spec viva` ni `archive integrate --odd` |
| Citas de código | `synthetic.go:9-12`, `registry.go:105-112` y `:144`, `livingdoc/indexer.go:140-145`, `adoptcheck/check.go:150-154`, `cli/sdd_trace.go:360-366`, `sddstatus/iso.go:126-131`, `status.go:546`, `kickoff/machine.go:252`, `governance.go:434`, `openspecdoc/compose.go:315` y `:357` siguen vigentes |

Hallazgos nuevos de la revalidación, incorporados al design: (a) `routing_test.go:805` exige que `axiom archive integrate --odd <feature>` aparezca **una sola vez** en el renderizado; (b) varias pruebas de `internal/assets` fijan el texto actual de `sdd-archive` (bucle `archive-compose`, bloque de copia de capacidad nueva); (c) `docs/manual-de-inicio.md:657` y `:662` dicen que la integración ODD «no modifica el documento ODD»; (d) `axiom sdd integrate` es una lápida (`cmd/axiom/main.go:2242`) y el verbo nuevo no la reutiliza; (e) la guarda A del verbo es `guardChangeWriter(..., branch.ContextArchive)` (`internal/cli/sdd_writer_guard.go:35`), no `guardComposeLooseDelta`.

## Decisiones aceptadas por la coordinación del plan

| # | Decisión | Motivo |
|---|---|---|
| A1 | Un provisional puede llevar una **etiqueta de serie nueva** solo con patrones que tengan `<n>`. Con patrones con `<DOM>` el dominio va fijado (un dominio nuevo se escribe literal: `prov:FAC:1`). | El formato está delegado en `docs/plan/backlog-2026-11/README.md:370`; un código de dominio no se puede inventar; un número de serie sí. |
| A2 | Un **ADDED con ID definitivo sigue siendo válido**, con la comprobación de colisión de siempre (`CollisionError`, REQ-30.4; REQ-29.3 en ISO). | D-03-4 exige declarar lo que se crea, no prohíbe el definitivo. D-03-8 lo necesita para C-03-c, y el archive de este cambio usa REQ-47 con el binario actual. |
| A3 | El equivalente ODD de `status` es `axiom archive integrate --odd <feature> --check`: JSON, solo lectura y el mismo `requirements: {modifies, creates}` que `axiom sdd status --json`. | Evita un status ODD completo y reutiliza el plan real. |

## Decisiones de arquitectura

### D1. Formato literal del provisional: `prov:<clave>:<ordinal>`

**Elección.**

```
provisional := "prov:" clave ":" ordinal
clave       := "" | componente ("-" componente)*
componente  := dominio   [A-Z][A-Z0-9]*      (valor de <DOM>)
             | serie     [0-9]+              (valor de <n>, serie fijada)
             | etiqueta  [a-z][a-z0-9]*      (serie nueva; solo en la posición de <n>)
ordinal     := [1-9][0-9]*
forma laxa  := "prov:" [A-Za-z0-9-]* ":" [0-9]+
```

`Pattern.ParseProvisional` exige que los componentes coincidan en número, orden y clase con los marcadores `<DOM>` y `<n>` de la plantilla, y que cada valor fijado sea un valor válido del marcador (un dominio que la plantilla no admite es `provisional_malformed`). Ejemplos: por defecto `prov:30:1` (serie 30) y `prov:a:1` (serie nueva «a»); `RF-<DOM>-<NN>` `prov:USU:1`; `RF-<DOM>-<n>.<m>` `prov:USU-3:1` o `prov:USU-b:1`; serie única `RF-<m>` `prov::1`.

- **Encabezado:** último paréntesis del título, como el definitivo: `### Requirement: Cobro con IVA (prov:30:1)`. `openspecdoc` no cambia su parser; `reqid.ProvisionalInTitle(name)` extrae el provisional. Un título con un provisional y cualquier otro identificador (definitivo o provisional distinto) es `provisional_ambiguous` (D7).
- **Token en texto:** `reqid.ScanProvisional(text)` reconoce la forma laxa con la regla de token completo (ni precedido ni seguido de letra, cifra, `_` o `-`); un token laxo que no cumple la gramática es `provisional_malformed`, nunca se ignora.
- **El ordinal no es el número asignado.** Solo distingue provisionales de la misma serie dentro del cambio; el número sale del orden de aparición (REQ-47.2).
- **Unicidad:** el token completo es único dentro del cambio o del documento ODD.
- **Sin escape (hueco design#13).** Un texto validado que quiera mostrar la forma usa ángulos (`prov:<serie>:<ordinal>`), que no encajan con la forma laxa. Los textos no validados (D6) pueden citar tokens reales.

**Alternativas descartadas.**

| Alternativa | Motivo del rechazo |
|---|---|
| `REQ-30.new1`, `RF-USU-n1` | Empieza por el literal inicial: `Scan` lo devolvería como candidato mal formado y `TrailingCandidate` lo tomaría como ID. Viola REQ-47.1.1. |
| `RF-USU-N1` | Válido bajo plantillas como `RF-<DOM>-N<m>`. |
| Rango alto `RF-9001` | Confundible con un definitivo real. |
| `new-<serie>-<k>` | Ambiguo con claves que contienen `-`. |

**Motivo.** `prov` empieza en minúscula: nunca es `Valid` (invariante de `synthetic.go:9-12`), nunca encaja con `InTitle` ni con `TrailingCandidate`. Caso límite: con `RF-<DOM>-<n>.<m>` y el dominio `RF`, la clave `RF-3` contiene `\bRF-`; por eso **`Scan` enmascara los tokens provisionales antes de buscar candidatos** (REQ-47.1.1). Los dos puntos y la ausencia de espacios lo hacen compatible con `Cubre:` y con la sustitución por token. `rg "prov:[A-Za-z0-9-]*:[0-9]+"` sobre `openspec/` e `internal/` fuera de este cambio no devuelve nada: ninguna composición aceptada hoy contiene la forma (REQ-30.23).

### D2. Ampliación del puerto `internal/reqid`

Solo biblioteca estándar (se conserva `arch_test.go`):

- `Provisional` con `String()`, `Label() (string, bool)`; `ParseProvisionalToken(s)`, `ProvisionalInTitle(title)`, `ScanProvisional(text) []Span`.
- `(p Pattern) ParseProvisional(token) (Provisional, error)` → `*ProvisionalError{Token, Reason}`.
- `(p Pattern) SeriesOf(key string) (Series, bool)`.
- `Allocator`: `Occupy(id)`, `Next(series) (string, error)`, `NewSeries(pv) (Series, error)` (siguiente `<n>` libre entre las series que comparten los demás componentes; nunca una serie ocupada ni fijada por el propio cambio). `Registry.NextFree` (`livespec/registry.go:144`) delega en él; una serie vacía da `1` con el relleno de la plantilla. Desbordamiento o formato no `Valid` → `*AllocationError`.
- `IDMap` + `Assignment{Provisional, Definitive string}`; `Substitute(text)` por token completo, más largo primero, una sola pasada; admite claves definitivas (D→D') para la reasignación de D5.
- `Scan` enmascara los provisionales.
- `arch_test.go`: el literal `prov:` solo aparece en código de producción de `internal/reqid`.

**Alternativa.** `IDMap` en `openspecdoc` o `livespec`: rechazado; la regla de token es parte de la forma del identificador y la usan `isoevidence` y `livespec` sin depender entre sí.

### D3. Plan único, validación previa y escritura atómica

`livespec.PlanIDs(IDPlanInput) (IDPlan, error)` recibe todas las capacidades del cambio o del documento a la vez y devuelve un `reqid.IDMap` más las entradas de traza, sin escribir. El orquestador del carril (`PlanIntegration` u `PlanChange`) sustituye, compone en memoria, comprueba duplicados y escribe todo con un único `writeSet`.

1. **Orden.** SDD: capacidades por nombre de carpeta; ODD: orden del documento. Dentro de cada capacidad, orden de aparición de los ADDED. Una entrada de la traza conserva su definitivo aunque cambie el orden (renombrar una carpeta o añadir una capacidad que va antes no renumera lo ya asignado; hueco design#8).
2. **Validación previa** (sin escribir; el primer error rechaza todo; misma secuencia con y sin `--check`):
   a. traza legible, de este patrón y coherente (D12);
   b. tokens provisionales de todos los textos validados (D6): gramática, compatibilidad con el patrón, etiqueta con patrón sin `<n>`;
   c. nombres: `provisional_ambiguous`, `provisional_outside_added`, `provisional_duplicate`;
   d. citas: `provisional_undeclared`;
   e. clasificación contra la canónica con la función `classify` compartida (`livespec/plan.go:198`, hoy solo ODD): capacidad sin canónica con MODIFIED/REMOVED/RENAMED → `*MissingCapabilityError` (`capability_missing`, hueco design#2); ADDED presente con otro cuerpo → `*DivergentRequirementError`; MODIFIED o RENAMED antiguo **pendiente** cuyo nombre lleva un ID válido que `Resolve` no encuentra → `unknown_requirement_id` con la capacidad (hueco design#10). REMOVED que ya no se resuelve y RENAMED ya aplicado son `already`, no error (REQ-47.3 regla 3). Un nombre sin ID que no casa sigue pendiente y lo rechaza la composición con el motivo vigente (REQ-47.3 regla 2, hueco design#11);
   f. colisiones de los ADDED definitivos (`checkIdentifiers`, REQ-30.4; en SDD ISO ya lo bloquea el status);
   g. SDD: nombres de `--superseded-requirements` (D4).
3. **Ocupación.** `Allocator` con: el registro (`openspec/specs/` + deltas de cambios activos, `ExcludeChange` = el propio cambio SDD), **más** los ADDED definitivos del propio cambio o documento, **más** los definitivos de la traza. Un registro ilegible conserva el error actual de `LoadRegistry` (`internal/cli/archive_integrate.go:87-90`), que rechaza sin escribir. Definitivos repetidos dentro del registro no alteran el máximo de la serie y no bloquean aquí (los duplicados se tratan en D5). Dos integraciones simultáneas en dos árboles de trabajo no se coordinan: las detecta REQ-47.5 tras el rebase (hueco design#16).
4. **Asignación.** Series fijadas: `Next(series)` en orden. Etiquetas: en orden de primera aparición, `NewSeries` y números consecutivos desde 1.
5. **Traza previa** (D5): entradas `kept`, `reassigned` o rechazo por cuerpo editado.

**Escritura.** `Integration.Apply` se generaliza a `writeSet(files []fileWrite, opts)`: temporal `.<base>.axiom-tmp` junto al destino, renombrado de todos, restauración en orden inverso ante un fallo y retirada de carpetas creadas. Si la restauración falla se conserva `rollbackError` (`livespec/apply.go:165`), que nombra cada ruta no restaurada y la continuación `git checkout -- <ruta>` (hueco design#9). El nombre del temporal es determinista: un huérfano de un proceso matado se sobrescribe y se consume en la siguiente ejecución real que escribe ese destino; `--check` no lo toca. Los ficheros con punto son estado local para `docpaths`. La regeneración de `openspec/INDEX.md` es opcional: ODD la conserva («solo si ya existe»); SDD no la hace, porque la skill ejecuta `axiom archive sync`.

**Alternativas.** Planificar por dominio: dos capacidades recibirían el mismo número. Escribir con `git`: el motor no ejecuta procesos y el patrón temporal+rename ya existe.

### D4. Verbo SDD: `axiom sdd archive-integrate`

```
axiom sdd archive-integrate --cwd <dir> --change <cambio>
    [--supersede] [--superseded-requirements "<a>, <cap>/<b>"] [--check] [--json]
```

- Esquema `axiom.sdd-archive-integrate/v1`: `schema`, `change`, `check`, `capabilities[] {name, path, outcome: created|composed|resumed|already_integrated, skipped[]?}`, `requirements {modifies[], creates[]}`, `assignments[] {provisional, definitive, capability, requirement, state: assigned|kept|reassigned, previous?}`, `duplicates[] {id, capabilities[], blocking}`, `written[]`, `nothingToIntegrate`, y en error `error {code, capability?, section?, requirement?, token?, file?, line?, continuation}`.
- Guardas, en este orden: `baseline:` residual (`loc.GuardSpecsWrite` / `BaselineRetiredErr`), patrón accesible (`projectIDPattern`, REQ-44.9), negativa ISO (`sddstatus.ISOArchiveRefusal`), guarda A `guardChangeWriter(changeRoot, change, "sdd archive-integrate", branch.ContextArchive)` (REQ-25.19: el verbo escribe en la carpeta del cambio). `--check` omite solo la guarda A.
- **Composición por capacidad** (huecos design#2 y #14):
  - sin canónica → la validación previa ya rechazó MODIFIED/REMOVED/RENAMED; se copia el texto de la delta con la sustitución aplicada (`created`; sin provisionales, byte a byte igual al `cp` de hoy);
  - con canónica → `openspecdoc.Compose(canónica, deltaSustituida, opts)` con la delta entera (`composed`, paridad byte a byte con `archive-compose`). Si `Compose` rechaza, se consulta la clasificación: nada pendiente → `already_integrated`; algo pendiente → `Compose(canónica, reducedDelta(pendiente))` (`resumed`, con `skipped[]`). Un error real (MODIFIED que no se resuelve, ADDED con otro cuerpo) sigue pendiente o ya fue rechazado y falla cerrado. Sustituye al antiguo error `partially_integrated`, que no tenía continuación.
- **Superación** (hueco design#3): `--supersede` vale para los REMOVED de todas las capacidades. Cada nombre de `--superseded-requirements` admite el prefijo `<capacidad>/` (se separa en el primer `/` solo si el prefijo es una capacidad del cambio); sin prefijo debe casar por título exacto en la canónica de exactamente una capacidad. Cero → `superseded_requirement_unknown`; varias → `superseded_requirement_ambiguous` con la continuación del prefijo. `archive-compose` conserva su comportamiento (ignora un nombre inexistente) por REQ-30.23.
- **Reescrituras**, solo con mapa no vacío: deltas `specs/<cap>/spec.md` enteras, líneas `Cubre:` de `tasks.md` y `tasks.<rol>.md`, columna `Requisito` de la matriz de `verify-report*.md` (D8) y `ids.yaml`.
- Delta plana `specs/spec.md` → `flat_delta_unsupported`.
- `--check` sobre un cambio archivado (`changes/archive/*-<cambio>/ids.yaml`): cada definitivo de la traza aparece una sola vez en la spec viva y en su capacidad; si no, `duplicate_assigned_id` con la continuación de D5.
- Texto de ayuda: `cmd/axiom/main.go:171` y `:2211` y la lista de `main_test.go:52`. No reutiliza la lápida `sdd integrate`.

**Skill `sdd-archive`.** El bucle por dominio (`archive-compose` + `mv`) y el bloque de copia de capacidad nueva se sustituyen por **una** invocación de `axiom sdd archive-integrate ... --json` tras `axiom sdd archive-check` (y, con contexto mantenido, entre `axiom archive context prepare` y las ediciones de contexto, en el orden de `SKILL.md:97`); después `axiom archive sync`; la instantánea y el `mv` del Step 3 siguen igual y después. `archive-report.md` incluye la tabla de `assignments`. La skill manda `--check` tras cada rebase de la rama de archivo. `archive-compose` queda como vista previa y composición manual.

**`archive-check`** gana `assignments` (de `ids.yaml`) y `pendingProvisionals`. Tras el verbo la delta lleva definitivos y `alreadyComposed` (D17) funciona sin cambios.

**Alternativas.** Modo `axiom archive integrate --change <c>`: las guardas, la negativa ISO, la reescritura de artefactos SDD y la traza en fichero son del carril SDD. Ampliar `archive-compose` a varias capacidades: rompería su contrato de un dominio y su salida byte a byte.

### D4b. `--check` de ODD (A3, hueco design#7)

`axiom archive integrate --odd <feature> --check [--json]` recorre `RunArchiveIntegrate` (`internal/cli/archive_integrate.go:34`) completo salvo `guardODDWriter` y `plan.Apply()`: configuración, negativa ISO, patrón, documento, secciones, impacto, plan único, clasificación, colisiones, composición en memoria, duplicados y `checkContext` (REQ-30.6), en el mismo orden. `INDEX.md` no participa: es un derivado que no rechaza. El informe `axiom.odd-integration/v1` añade, solo con `--check`, `check: true`, `requirements` y `assignments` con estado; sobre un documento ya integrado devuelve `nothingToIntegrate: true`, código `0` y todas las entradas `kept`.

### D5. Asignación en la rama y reintegración idempotente

Se asigna en la rama: ODD en la rama de la feature, SDD en la rama de archivo. La traza es la fuente de identidad de lo ya asignado. Para cada ADDED (C = capacidad de la entrada):

| Título del ADDED | Traza | Estado de D en la spec viva | Resultado |
|---|---|---|---|
| Provisional P | sin entrada | — | `assigned`: siguiente libre |
| P o definitivo D | P→D | D libre en todo el registro | `kept`: reutiliza D |
| P o D | P→D | requisito con D en C, cuerpo igual módulo sustitución, y D en ningún otro sitio | `kept`, `already_integrated` |
| P o D | P→D | requisito con D en C cuyo título es el `requirement` guardado en la traza y cuyo cuerpo difiere | **rechazo** `*DivergentRequirementError` (REQ-30.3): el autor editó lo integrado; continuación: restaurar `openspec/specs/<C>/spec.md` desde la base y reintegrar (la traza conserva D) |
| P o D | P→D | D ocupado por un requisito que no es nuestro (otra capacidad, o título y cuerpo distintos) | `reassigned`: D→D' sobre documento, delta, `Cubre:`, matriz y nuestro bloque en la spec viva si estaba |
| Definitivo D | sin entrada | — | elegido por el autor: colisión de siempre (A2) |

- **«Nuestro» bloque** (hueco design#1): en C, con D, y con cuerpo igual módulo sustitución **o** con el título que la traza guardó. Así un cambio de título del autor con el mismo cuerpo se reconoce (REQ-30.3) y una edición del cuerpo sin cambiar el título se rechaza, en lugar de duplicarse. Si un cambio ajeno usa por azar el mismo título y otro cuerpo, se rechaza (falla cerrado).
- **Serie nueva en colisión:** se reasigna la etiqueta entera a una serie nueva, con números consecutivos desde 1, en todas las capacidades; con serie fijada solo la entrada.
- **Duplicados (REQ-47.5):** tras componer en memoria, se cuentan las apariciones de cada ID que escribe la integración en todas las capacidades resultantes, superados incluidos. Más de una → `duplicate_assigned_id`. Los históricos ajenos se informan con `blocking: false` (la spec viva tiene 12, de `REQ-1.1` a `REQ-3.3`).
- **ODD tras rebase:** con conflicto git en `openspec/specs/`, la guía ordena conservar la base y reintegrar; sin conflicto, la reejecución detecta la ocupación y reasigna, con `previous`.
- **SDD tras rebase:** el cambio archivado es inmutable (REQ-21.18). La rama de archivo se descarta (`axiom branch done --force`) y se rearchiva desde `main`, donde la carpeta del cambio aún tiene los provisionales. La detección la da `--check` sobre el cambio archivado (D4).

**Alternativas descartadas.** Asignar tras la fusión en `main`: exige un paso inexistente y deja provisionales en la spec viva. Guardar solo el mapa sin reescribir el documento ODD: contradice REQ-30.2. Reasignar también el bloque propio editado: deja dos bloques con el mismo texto de origen y contradice REQ-30.3.

### D6. Alcance de validación y de reescritura

| Texto | ¿Se valida que todo provisional esté declarado? | ¿Se reescribe? |
|---|---|---|
| Delta del cambio, **fichero entero** (títulos, cuerpos y preámbulo, que una capacidad nueva copia) | Sí | Sí |
| Ficheros de tareas, líneas `Cubre:` | Sí | Sí, solo el token |
| Ficheros de tareas, resto | Sí | No (la traza explica la mención) |
| `verify-report*.md`, columna `Requisito` de la matriz | Sí | Sí (D8) |
| `verify-report*.md`, resto | No | No |
| Documento ODD, valla `openspec` de `## Spec viva` y `Cubre:` fuera de vallas | Sí | Sí |
| Documento ODD, resto salvo la sección de traza | Sí | No |
| Documento ODD, `## Identificadores asignados` | No | No (la escribe la traza) |
| `proposal.md`, `design.md`, `explore.md`, otros informes | No | No |

«Declarado» = un ADDED del cambio **o** una entrada de la traza. En `Compose`, `ProvisionalIDError` se emite ante cualquier token laxo en el **nombre o el cuerpo** de un requisito de cualquier sección (hueco design#12); el preámbulo de la delta no llega a la spec viva compuesta y `Compose` no lo mira.

### D7. Errores tipados

Un tipo en `livespec`, `*IDPlanError{Code, Capability, Section, Requirement, Token, File, Line, Continuation}`, con códigos estables en el JSON de `--check`. Los errores tipados ya existentes se conservan y se proyectan con código:

| Código | Caso | Origen |
|---|---|---|
| `provisional_malformed` | token laxo que no cumple la gramática o los componentes del patrón (incluida una serie fijada no válida) | `IDPlanError` |
| `provisional_new_series_unsupported` | etiqueta con patrón sin `<n>` | `IDPlanError` |
| `provisional_outside_added` | provisional en MODIFIED, REMOVED o RENAMED | `IDPlanError` |
| `provisional_duplicate` | mismo provisional en dos ADDED | `IDPlanError` |
| `provisional_ambiguous` | nombre con un provisional y otro identificador, en cualquier carril (hueco design#15) | `IDPlanError` |
| `provisional_undeclared` | citado (D6) sin ADDED ni traza | `IDPlanError` |
| `unknown_requirement_id` | MODIFIED o RENAMED antiguo pendiente con ID válido inexistente; envuelve `openspecdoc.UnknownRequirementIDError` y añade la capacidad | `IDPlanError` |
| `capability_missing` | capacidad sin canónica con MODIFIED/REMOVED/RENAMED | `*MissingCapabilityError` (existente, ahora también SDD) |
| `requirement_diverged` | ADDED presente con otro cuerpo, incluido el bloque propio editado (D5) | `*DivergentRequirementError` (existente) |
| `id_allocation_failed` | desbordamiento o definitivo no `Valid` | `IDPlanError` |
| `id_trace_invalid` | traza ilegible, esquema desconocido, sección ODD duplicada o definitivo no válido | `IDPlanError` |
| `id_trace_pattern_mismatch` | `pattern` de la traza distinto del efectivo | `IDPlanError` |
| `id_trace_stale` | entrada que no casa con exactamente un ADDED de su capacidad (por P o por D) | `IDPlanError` |
| `duplicate_assigned_id` | REQ-47.5 | `IDPlanError` |
| `superseded_requirement_unknown`, `superseded_requirement_ambiguous` | D4 | `IDPlanError` |
| `flat_delta_unsupported` | D4 | `IDPlanError` |

En `openspecdoc`: `*UnknownRequirementIDError{Section, Requirement, ID}` sale de `Compose` cuando el nombre lleva un ID válido y `Resolve` da `MatchNone` (`compose.go:357`); `Unwrap()` devuelve el `*UnappliedDeltaError` equivalente para los `errors.As` actuales. El paquete no conoce la capacidad: la añade `livespec` (validación previa) o la CLI de `archive-compose` (que conoce el dominio por la carpeta). `*ProvisionalIDError{Section, Requirement, Token}` sale de `Compose` ante cualquier provisional en nombres o cuerpos; `archive-compose` lo envuelve con la continuación a `axiom sdd archive-integrate`. Un MODIFIED sin ID se resuelve por título (REQ-30.22) y conserva el rechazo actual si no casa; un nombre con dos definitivos y ningún provisional conserva la regla actual (último ID; ISO lo bloquea por REQ-29.3). `added_without_id` (ODD) se conserva para un ADDED sin definitivo ni provisional, con un texto que sugiere el provisional.

### D8. ISO: provisionales en `isoevidence` y matriz

`isoevidence.RequirementKeys(name, p) (valid, provisional, malformed []string)` sustituye a `RequirementIDs` (`reqid.go:14`) en `CheckRequirementIDs`, `deltaIDs`, `CheckCoverage` y `AffectedFromDelta` (`cubre.go:90`, `:109`, `:228`), y en sus llamadores `sddstatus/iso.go` y `cli/sdd_trace.go`. Reglas nuevas de REQ-29.3 con el código existente `iso_requirement_id`; un provisional no declarado en `Cubre:` usa `iso_cubre_unknown` con detalle propio. La matriz admite el provisional de un ADDED (la clave de escenario coincide con `AffectedFromDelta`) y se reescribe al archivar con `RewriteMatrix` (REQ-47.4 destino 5, REQ-29.3), porque tras reescribir la delta un `archive-check` reanudado compararía definitivos con provisionales y bloquearía.

### D9. Proyección `requirements` en `status`

`internal/sddstatus/requirements.go`: `decorateRequirements(status, location)`, llamado tras `decorateISO` (`status.go:546`), con `Requirements *RequirementsProjection json:"requirements,omitempty"` en `Status` y en `status_v2.go`. Deriva de `openspecdoc.Declared(capability, delta, pattern)`, compartida con los dos `--check`.

```json
"requirements": {
  "modifies": [{"capability": "pagos", "id": "REQ-30.1", "operation": "MODIFIED", "requirement": "Cobro con IVA (REQ-30.1)"},
               {"capability": "pagos", "operation": "MODIFIED", "requirement": "Cobro"}],
  "creates":  [{"capability": "pagos", "id": "prov:30:1", "provisional": true, "operation": "ADDED", "requirement": "Cobro (prov:30:1)"}]
}
```

Alineado con REQ-47.9 (hueco design#4 y spec#5):

- `capability`, `operation` y `requirement` siempre; `id` solo si el nombre lleva un ID válido o un provisional; `provisional: true` en ese caso. Un nombre sin ID no tiene `id` que proyectar: se resuelve por título exacto, y un título igual al de la canónica tampoco lleva ID. No se lee la canónica.
- RENAMED usa el nombre antiguo. Una capacidad nueva escrita como spec completa cuenta como ADDED (`ParseChangeSpec`).
- Se omite en los tres casos de REQ-47.9: sin deltas legibles en ficheros (incluido almacén `engram`/`none`), cambio archivado (`status.Archived != nil`) y patrón inaccesible. No toca `nextRecommended`, `dependencies` ni motivos.
- Contrato: sección nueva «Requirements» en `_shared/sdd-status-contract.md`.

### D10. Digest de puertas

`kickoff/machine.go:252` (`resolveGateStatus`): una aprobación es terminal con independencia del digest. El archivado solo se alcanza con las puertas aprobadas, así que reescribir tareas, deltas o `verify-report.md` no devuelve ninguna a `pending`; en `continuous` no se evalúan (`governance.go:434`). `edit_authority.go` mira rutas, que la reescritura no cambia. Prueba de regresión G1.

### D11. Lectores de deltas activas y del documento ODD

| Lector | Efecto del provisional o de la traza | Tratamiento |
|---|---|---|
| `livespec/registry.go:105-112` (colisiones con cambios activos) | `InTitle` → "": no se indexa ni colisiona | Prueba (REQ-47.1) |
| `livingdoc/indexer.go:140-145` | Solo lee `openspec/specs/`; `(prov:…)` no es `TrailingCandidate` | Prueba (REQ-47.1) |
| `livingdoc/synthesizer.go` | Flujo heredado | Sin cambio; documentado |
| `dashboard` (`odd_livespec_test.go`) | La traza no tiene casillas | Prueba (REQ-47.1) |
| `livespec/odddoc.go:184` (`ParseFeatureDoc`) | Solo interpreta `## Spec viva` e `## Impacto en la spec`; un `## ` nuevo cierra la sección anterior y la valla `yaml` queda fuera de las capacidades | Prueba de validación sin cambios con la traza (REQ-47.6); traza duplicada → `id_trace_invalid` |
| `adoption/adoptcheck/check.go:150-154` | Un provisional cuenta como «sin identificador» | Fuera de alcance (C-03-c); documentado |
| `cli/sdd_trace.go:364`, `sddstatus/iso.go:129` | D8 | `RequirementKeys` |

### D12. Traza

`livespec.Trace`, en YAML (`gopkg.in/yaml.v3`, ya en `go.mod`):

```yaml
schema: axiom.requirement-ids/v1
pattern: REQ-<n>.<m>
assignments:
  - provisional: prov:30:1
    definitive: REQ-30.7
    capability: odd-living-spec-integration
    requirement: Título del requisito (REQ-30.7)
    previous: [REQ-30.5]   # solo si se reasignó
```

- SDD: `openspec/changes/<cambio>/ids.yaml`. ODD: sección `## Identificadores asignados` al final del documento con un único bloque vallado `yaml`. Sin asignaciones no se escribe.
- **Robustez** (hueco design#6): YAML ilegible, `schema` desconocido, sección duplicada o definitivo no `Valid` → `id_trace_invalid`; `pattern` distinto del efectivo → `id_trace_pattern_mismatch`; entrada sin exactamente un ADDED de su capacidad que lleve su P o su D → `id_trace_stale`. Continuación común: restaurar la traza con git o rehacer la integración desde la base. Nunca se regenera en silencio.
- **Traza borrada** en un rebase: ODD degrada a definitivos elegidos por el autor (A2): lo propio con el mismo cuerpo se reconoce por `classify` y un número ocupado por otro da la colisión de siempre con el siguiente libre; SDD no la necesita porque se rearchiva desde `main` (D5).
- Informes: `assignments` en `axiom.odd-integration/v1` y `axiom.sdd-archive-integrate/v1`; tabla en `archive-report.md`.

## Flujo de datos

```
deltas / doc ODD ──► ParseChangeSpec ──► validación previa ──► PlanIDs ◄── Registry (specs + activos − propio)
                         (traza, tokens, nombres, citas,          │  ◄── definitivos propios + traza previa
                          classify, colisiones, superados)        ▼
                                                               IDMap ──► Substitute ──► Compose / copia / reanudación
                                                                  │                              │
                                                                  ├──► RewriteCubre / RewriteMatrix / doc ODD
                                                                  └──► Trace ─────────────┐      │
                                                                                          ▼      ▼
                                               duplicados (REQ-47.5) ◄── vista en memoria de la spec viva
                                                          │
                                         contexto (solo ODD, REQ-30.6)
                                                          │
                                      --check ──► informe JSON (sin escribir)
                                      real    ──► writeSet atómico ──► informe
```

## Cambios de ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/reqid/provisional.go` | Crear | Gramática, forma laxa, `ProvisionalInTitle`, `ScanProvisional`, `ProvisionalError` |
| `internal/reqid/idmap.go` | Crear | `IDMap`, `Substitute`, `Count` |
| `internal/reqid/allocate.go` | Crear | `SeriesOf`, `Allocator`, `NewSeries`, `AllocationError` |
| `internal/reqid/reqid.go` | Modificar | `Scan` enmascara provisionales |
| `internal/reqid/arch_test.go`, `fuzz_test.go` | Modificar | `prov:` solo en `reqid`; propiedad «provisional ⇒ no válido» |
| `internal/openspecdoc/compose.go` | Modificar | `UnknownRequirementIDError`, `ProvisionalIDError` en nombres y cuerpos |
| `internal/openspecdoc/declared.go` | Crear | `ParseChangeSpec`, `Declared`, `DeclaredRequirement` |
| `internal/isoevidence/reqid.go`, `cubre.go`, `matrix.go` | Modificar | `RequirementKeys`, reglas de D8, `RewriteCubre`, `RewriteMatrix` |
| `internal/livespec/ids.go` | Crear | `PlanIDs`, validación previa, `IDPlanError`, reasignación, duplicados |
| `internal/livespec/trace.go` | Crear | `Trace` YAML, lectura, validación y escritura (fichero y sección ODD) |
| `internal/livespec/apply.go` | Modificar | `writeSet` generalizado; `Apply` lo usa; `rollbackError` se conserva |
| `internal/livespec/plan.go` | Modificar | `classify` y `reducedDelta` compartidos; provisionales en ODD |
| `internal/livespec/change.go` | Crear | `PlanChange` (SDD): composición, reanudación, superados |
| `internal/livespec/odddoc.go` | Modificar | Reescritura del documento y sección de traza |
| `internal/livespec/registry.go` | Modificar | `NextFree` delega en `reqid.Allocator` |
| `internal/livespec/report.go` | Modificar | `check`, `assignments`, `requirements`, `duplicates` |
| `internal/cli/archive_integrate.go` | Modificar | `--check` (D4b), ayuda, texto |
| `internal/cli/sdd_archive_integrate.go` | Crear | Verbo nuevo |
| `internal/cli/sdd_archive_compose.go` | Modificar | Continuación del `ProvisionalIDError`; capacidad en `UnknownRequirementIDError` |
| `internal/cli/sdd_archive_check.go` | Modificar | `assignments`, `pendingProvisionals` |
| `internal/cli/sdd_trace.go` | Modificar | `RequirementKeys` |
| `cmd/axiom/main.go`, `cmd/axiom/main_test.go` | Modificar | Registro y ayuda de `archive-integrate`; lista de subcomandos |
| `internal/sddstatus/requirements.go` | Crear | `decorateRequirements` |
| `internal/sddstatus/status.go`, `status_v2.go`, `iso.go` | Modificar | Campo `Requirements`; `RequirementKeys` |
| `internal/assets/skills/sdd-archive/SKILL.md` | Modificar | Paso único Go; tabla del mapa; `--check` tras rebase; sin copia `cp` de capacidad nueva (REQ-47.10) |
| `internal/assets/skills/sdd-archive/references/technical-context-integration.md` | Verificar | Su pin («Run the default Step 2 blocks…») sigue siendo cierto; solo se ajusta si cambia el texto |
| `internal/assets/skills/sdd-spec/SKILL.md` | Modificar | Guía del provisional (línea de ISO, `SKILL.md:257`, y sección de ADDED) |
| `internal/assets/skills/sdd-spec/references/openspec-adoption.md` | Modificar | Paso 7: archiva el verbo, con la composición de `archive-compose --canonical --delta` (se conserva la cadena fijada en `assets_test.go:3686`) |
| `internal/assets/skills/_shared/sdd-iso-convention.md` | Modificar | Provisional en ADDED, `Cubre:` y matriz; `:88` nombra también el verbo |
| `internal/assets/skills/_shared/sdd-status-contract.md` | Modificar | Sección `Requirements` |
| `internal/assets/assets_test.go`, `internal/assets/sdd_iso_pins_test.go` | Modificar | Pins de `sdd-archive` (`TestSDDArchiveFollowsTheArchiveBranchFlow`, `TestSDDArchiveStoreSpecificFilesystemContract`, `TestSDDArchiveTechnicalContextBranch`, `TestSDDArchiveRequiresArchiveCheck`) al flujo nuevo, y pins nuevos de REQ-47.10 |
| `internal/components/agentguidance/routing.go`, `routing_test.go` | Modificar | Paso 7 ODD (REQ-24.58 regla 3), un solo párrafo y el comando una sola vez (`routing_test.go:805`) |
| `docs/manual-de-inicio.md` (+ `internal/workspace/docs_pin_test.go`) | Modificar | `:657` y `:662`: ADDED con provisional, reescritura del documento, `--check`; `:431`: verbo de archivado. Se conservan las cadenas fijadas |
| `testdata/golden/*` | Verificar | 0 de 63 contienen el paso 7 o las guías; se regeneran solo si una prueba de goldens falla |

## Interfaces

```go
// internal/reqid
func ProvisionalInTitle(title string) (token string, ok bool)
func ScanProvisional(text string) []Span           // Span{Start, End int; Token string; Lax bool}
func (p Pattern) ParseProvisional(token string) (Provisional, error)
func (p Pattern) SeriesOf(key string) (Series, bool)
func NewAllocator(p Pattern) *Allocator
func (a *Allocator) Occupy(id string)
func (a *Allocator) Next(s Series) (string, error)
func (a *Allocator) NewSeries(pv Provisional) (Series, error)
func NewIDMap(pairs []Assignment) IDMap
func (m IDMap) Substitute(text string) string

// internal/openspecdoc
type UnknownRequirementIDError struct{ Section, Requirement, ID string } // Unwrap() → *UnappliedDeltaError
type ProvisionalIDError struct{ Section, Requirement, Token string }
func ParseChangeSpec(text string) (d Delta, fullSpec bool)
type DeclaredRequirement struct {
    Capability  string `json:"capability"`
    ID          string `json:"id,omitempty"`
    Provisional bool   `json:"provisional,omitempty"`
    Operation   string `json:"operation"`
    Requirement string `json:"requirement"`
}
type Declaration struct {
    Modifies []DeclaredRequirement `json:"modifies"`
    Creates  []DeclaredRequirement `json:"creates"`
}
func Declared(capability string, d Delta, p reqid.Pattern) Declaration

// internal/isoevidence
func RequirementKeys(name string, p reqid.Pattern) (valid, provisional, malformed []string)
func RewriteCubre(text string, m reqid.IDMap) string
func RewriteMatrix(text string, m reqid.IDMap) string

// internal/livespec
type IDPlanInput struct {
    Pattern      reqid.Pattern
    Capabilities []CapabilitySource // nombre, delta, texto, canónica actual
    Registry     *Registry          // sin el propio cambio
    Trace        Trace
    Citations    []Citation         // tokens de Cubre, tareas, matriz, documento (D6)
}
type IDPlan struct{ Map reqid.IDMap; Entries []TraceEntry }
func PlanIDs(in IDPlanInput) (IDPlan, error)
type ChangeOptions struct{ Supersede bool; Superseded []string; Check bool }
func PlanChange(loc workspace.Location, change string, opts ChangeOptions, reg *Registry) (ChangeIntegration, error)
func (c ChangeIntegration) Apply() ([]string, error)
```

## Estrategia de pruebas

TDD por paquete: RED observado antes de cada producción. Comandos localizados: `go test ./internal/reqid/...`, `./internal/openspecdoc/...`, `./internal/isoevidence/...`, `./internal/livespec/...`, `./internal/sddstatus/... -run '<Requirements|Gate|ISO>'`, `go test ./internal/cli -run '<ArchiveIntegrate|SDDArchive>'`, `go test ./internal/assets -run 'SDDArchive|SDDSpec|ISO|StatusContract'`, `go test ./internal/components/agentguidance -run Routing`, `go test ./internal/workspace -run DocsPin`, `go test ./cmd/axiom -run Help` y `go vet` por paquete. Nunca la batería completa ni `go build -o`.

| Capa | Qué | Cómo |
|---|---|---|
| reqid | Provisional nunca `Valid`, `InTitle` ni candidato de `Scan` (`REQ-<n>.<m>`, `RF-<DOM>-<NN>`, `RF-<m>`, `HU-<NNN>`, `RF-<DOM>-<n>.<m>` con dominio `RF`); gramática y forma laxa; dominio no válido; etiqueta con `<DOM>`; `Next` en serie vacía; `NewSeries` evita series propias; desbordamiento; `Substitute` por token | Tablas; `fuzz_test.go` |
| reqid | `NextFree` igual que antes | `registry_test.go`, `registry_pattern_test.go` sin cambios |
| openspecdoc | Corpus byte a byte; `UnknownRequirementIDError` en MODIFIED, REMOVED y RENAMED antiguo con `errors.As` a `UnappliedDeltaError`; `ProvisionalIDError` en título y en cuerpo; `Declared` (sin ID, RENAMED, spec completa) | Unitarias |
| isoevidence | REQ-29.3 nueva; REQ-29.4 con provisional; matriz con provisional; `RewriteCubre`/`RewriteMatrix` conservan bytes | Unitarias |
| livespec plan | Determinismo; dos capacidades consecutivas; definitivos propios y de cambios activos ocupados; serie nueva no recae en una fijada; cada código de D7; MODIFIED sin ID por título; REMOVED ya aplicado; capacidad nueva con MODIFIED; nombre ambiguo | Unitarias |
| livespec D5/D12 | `kept`, `reassigned` individual y por etiqueta, bloque propio editado → `requirement_diverged`; traza ilegible, de otro patrón, obsoleta y duplicada; traza borrada (degradación A2) | Unitarias |
| livespec escritura | Fallo inyectado en cada escritura de `writeSet` sin temporales; fallo de restauración nombrado; temporal huérfano sustituido | Costuras `writeTempFile`/`renameFile` |
| cli ODD | `--check` no escribe (hash del árbol), mismo motivo que la real (contexto incluido), `INDEX.md` intacto; JSON con `requirements` y estados; documento ya integrado → `kept`, `nothingToIntegrate`, código 0; reintegración tras rebase simulado | `t.TempDir()` |
| cli SDD | Capacidad existente y nueva, serie nueva con los dos patrones; sin provisionales igual byte a byte a `archive-compose` + `cp`, sin `ids.yaml`; capacidad parcialmente integrada → `resumed`; capacidad nueva con MODIFIED; superados sin prefijo, con prefijo, inexistentes y ambiguos; fallo inyectado; reejecución idempotente; `archive-check` con `assignments`; `--check` en cambio archivado con duplicado; guarda A en rama ajena | En proceso |
| G1 puertas | `checkpointed` con puertas aprobadas: tras el verbo siguen `approved` y `nextRecommended` no cambia | `sddstatus` + `kickoff` |
| sddstatus | `requirements` ordenado; sin `id` para nombres sin ID; omitido sin deltas, con `engram`, archivado y patrón inaccesible (byte a byte); `nextRecommended` y `dependencies` iguales | Fixtures existentes |
| Lectores | D11 | Una prueba por lector |
| Guías | Paso 7 (`routing_test.go`), manual (`docs_pin_test.go`), pins de `sdd-archive`, `sdd-spec`, `sdd-iso-convention` y del contrato (REQ-47.10, REQ-47.9) | `internal/assets`, `agentguidance`, `workspace` |

## Matriz de amenazas

| Límite | Aplicabilidad | Respuesta | Pruebas RED |
|---|---|---|---|
| Rutas tipo documentación | N/A: no se clasifica ni se ejecuta ningún fichero | — | — |
| Selección de repositorio git | Aplicable: el verbo SDD resuelve el workspace con `--cwd` (`workspace.Locate`) y aplica la guarda A (`guardChangeWriter`, contexto `archive`) | Escribe solo bajo el `workspaceRoot` resuelto; fuera de la rama de archivo rechaza sin escribir (salvo `--check`) | Proceso en otro directorio con `--cwd` absoluto: solo cambian ficheros del workspace; rama que no es de archivo: rechazo y árbol intacto |
| Estado de commit | N/A: el motor no hace commit | — | — |
| Estado de push | N/A | — | — |
| Comandos de PR | N/A | — | — |

## Unidades de trabajo (un commit por unidad)

| U | Unidad | Paquetes | Depende de |
|---|---|---|---|
| U1 | Puerto: provisional y forma laxa, `IDMap`, `Allocator`, máscara de `Scan`, regla de arquitectura; `NextFree` delega | `reqid`, `livespec/registry.go` | — |
| U2 | Errores tipados de `Compose` (nombres y cuerpos) y `Declared`/`ParseChangeSpec`; continuación en `archive-compose` | `openspecdoc`, `cli/sdd_archive_compose.go` | U1 |
| U3 | ISO con provisionales y reescrituras `Cubre:`/matriz | `isoevidence`, `sddstatus/iso.go`, `cli/sdd_trace.go` | U1 |
| U4 | Validación previa, plan único, `classify` compartido, traza robusta, `writeSet`, duplicados | `livespec` | U1–U3 |
| U5 | ODD: reescritura del documento, reintegración (D5), `--check` (D4b), informe | `livespec`, `cli/archive_integrate.go` | U4 |
| U6 | SDD: `PlanChange` (reanudación, superados, capacidad nueva), verbo, `archive-check`, ayuda, prueba G1 | `livespec`, `cli`, `cmd/axiom` | U4 |
| U7 | `status.requirements` y contrato | `sddstatus`, `_shared/sdd-status-contract.md`, pin del contrato | U2 |
| U8 | Guías y documentación: `sdd-archive`, `sdd-spec` (+ adopción), `sdd-iso-convention`, paso 7 ODD, manual de inicio, pins de `internal/assets`, `routing_test.go`, `docs_pin_test.go`; comprobación de goldens | skills, `agentguidance`, `workspace`, `docs`, `internal/assets` | U5–U7 |

## Alineación con la spec

- **Ocupación:** REQ-47.2 enumera ahora los cuatro orígenes que cuenta D3 paso 3.
- **Matriz de verify:** REQ-47.4 (destino 5), REQ-29.3, REQ-21.17 y REQ-47.8 (punto 3) incluyen la reescritura de D8.
- **`requirements`:** REQ-47.9 y D9 coinciden en campos, en la ausencia de `id` y en los tres casos de omisión.
- **MODIFIED sin ID y REMOVED ya aplicado:** REQ-47.3 (reglas 2 y 3) y REQ-30.4 coinciden con D3 paso 2.e y con REQ-30.22.
- **Reintegración SDD:** se rehace el archivo (D5), no se reasigna en sitio (REQ-21.18); REQ-47.8 lo recoge.
- **REQ-44.3:** no se modifica; la nota de `## Purpose` de `requirement-identifiers` queda para la fase de archivo, como dice la delta.

## Migración y despliegue

No hay migración de datos. Las ramas abiertas con ADDED definitivos siguen igual (A2). El archive de este cambio usa REQ-47 con el binario anterior (proposal). La reversión deja la spec viva válida: nunca contiene provisionales.

## Preguntas abiertas

- [ ] Ninguna bloquea.

## Resolución de la revisión de cobertura

Origen: `spec#N` = `coverage-review.spec.md`, hueco N; `design#N` = `coverage-review.design.md`, hueco N. Decisión: **Añadida** (regla o escenario nuevos), **Fuera de alcance** (con motivo) o **Descartada** (infundada, con evidencia).

| Origen | Decisión | Dónde queda reflejado |
|---|---|---|
| spec#1 | Añadida: requisito de las guías y su pin; los goldens no contienen esos textos (0 de 63) | REQ-47.10 (`requirement-identifiers`); U8; Cambios de ficheros |
| spec#2 | Añadida: los definitivos propios, los de cambios activos y los de la traza cuentan como ocupados; una serie nueva no recae en una fijada | REQ-47.2 (texto y tres escenarios); D3 paso 3; D2 `NewSeries` |
| spec#3 | Añadida: la matriz cita el provisional hasta archivar y se reescribe | REQ-47.4 destino 5 y escenario; REQ-29.3 (texto y escenario); REQ-21.17; REQ-47.8 punto 3; D8 |
| spec#4 | Añadida: alcance de validación y de reescritura fuera de los destinos | REQ-47.4 (párrafo de alcance y dos escenarios); REQ-47.3 regla 6; D6 |
| spec#5 | Añadida: elemento sin `id`, con `requirement` siempre | REQ-47.9 (texto y escenario); D9 |
| spec#6 | Añadida: la segunda rama de archivado rebasada detecta el duplicado y se rehace desde la rama de integración | REQ-47.8 (párrafo y escenario); D4 `--check` en archivado; D5 |
| spec#7 | Añadida: `--check` sobre documento ya integrado (código 0, `kept`) y estados `assigned`/`kept`/`reassigned` | REQ-47.7 (párrafo y dos escenarios); D4b |
| spec#8 | Añadida: una serie nueva en colisión se reasigna entera | REQ-30.3 (texto y escenario); D5 |
| spec#9 | Añadida: patrón inaccesible en todas las órdenes y serie fijada no válida | REQ-47.3 reglas 9 y 11 y dos escenarios; D1; D7 `provisional_malformed` |
| spec#10 | Añadida: no regresión de los lectores | REQ-47.1 (escenario); D11 |
| spec#11 | Añadida: contenido del informe del paso y reanudación de una capacidad parcialmente integrada; la interrupción dentro del paso no existe porque `writeSet` es atómico | REQ-47.8 (informe, reanudación y escenarios); D4 (`resumed`) |
| design#1 | Añadida: el bloque propio editado se rechaza (`requirement_diverged`), no se reasigna | REQ-30.3 (texto y escenario); D5 (fila y «nuestro» bloque) |
| design#2 | Añadida: capacidad nueva con MODIFIED/REMOVED/RENAMED se rechaza (`capability_missing`, como ODD hoy en `livespec/plan.go:209-216`) | REQ-47.3 regla 4; REQ-47.8 punto 2; escenarios en ambos; D3 paso 2.e; D4 |
| design#3 | Añadida: `--supersede` para todas las capacidades; nombres con prefijo `<capacidad>/` y errores de nombre inexistente o ambiguo | REQ-47.8 (párrafo y escenario); D4 |
| design#4 | Añadida: spec y design coinciden en los tres casos de omisión y en `id` ausente | REQ-47.9; D9 |
| design#5 | Añadida: la sección de traza no altera la validación (`ParseFeatureDoc` solo interpreta dos secciones, `livespec/odddoc.go:184-230`); duplicada → `id_trace_invalid` | REQ-47.6 (texto y escenario); D11; D12 |
| design#6 | Añadida: traza ilegible, de otro patrón, obsoleta o con definitivo no válido → error tipado; traza borrada degrada a A2 | REQ-47.3 regla 10 y escenario; D7; D12 |
| design#7 | Añadida: `--check` ODD ejecuta la misma secuencia, contexto incluido, sin guarda A ni `INDEX.md` | REQ-47.7 (párrafo y escenario); D4b |
| design#8 | Añadida: una entrada de la traza conserva su definitivo aunque cambie el orden; sin traza nada está asignado todavía | REQ-47.2 (determinismo); D3 paso 1 |
| design#9 | Añadida: fallo de restauración nombrado (`rollbackError`, `livespec/apply.go:165-170`) y temporal huérfano sustituido | REQ-47.4 (texto y escenario); D3 Escritura |
| design#10 | Añadida: `unknown_requirement_id` con capacidad, en la validación previa, visible en `--check` | REQ-47.3 regla 2; D3 paso 2.e; D7 |
| design#11 | Añadida: MODIFIED sin ID por título fuera de ISO, con el rechazo vigente; ISO lo bloquea por REQ-29.3 | REQ-47.3 regla 2 y escenario; REQ-30.4; D7 |
| design#12 | Añadida: `Compose` rechaza un provisional en el cuerpo de cualquier requisito | REQ-30.22 (texto y escenario); D6; D7 |
| design#13 | Fuera de alcance: no hay mecanismo de escape; el texto validado usa un marcador con ángulos, que no encaja con la forma laxa, y los textos de diseño no se validan. Ninguna spec ni fuente actual contiene la forma (`rg` sin resultados) | REQ-47.4 (párrafo de alcance); D1; D6 |
| design#14 | Añadida: `partially_integrated` se sustituye por la reanudación (`resumed`), con la clasificación de ODD (`livespec/plan.go:198`, `:356`) | REQ-47.8 (texto y escenario); D4; D7 |
| design#15 | Añadida: nombre con un provisional y otro identificador → `provisional_ambiguous` en todos los carriles; dos definitivos sin provisional conservan la regla actual | REQ-47.3 regla 8 y escenario; D1; D7 |
| design#16 | Añadida (deltas ilegibles: error actual de `LoadRegistry`, `internal/cli/archive_integrate.go:87-90`; definitivos repetidos no alteran el máximo); Fuera de alcance (integraciones simultáneas en dos árboles: git es el punto de coordinación y REQ-47.5 las detecta tras el rebase) | REQ-47.3 (último párrafo); D3 paso 3 |
