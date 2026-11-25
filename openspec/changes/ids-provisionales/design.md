# Design: IDs provisionales, número asignado al integrar y declaración de impacto en `status` (C-03-b)

## Enfoque técnico

El cambio introduce una segunda clase de token de identificador, el **provisional**, con forma propia en el puerto `internal/reqid`, y un **plan único de asignación** que convierte provisionales en definitivos antes de escribir nada. El plan vive en `internal/livespec`, que pasa a ser el motor de integración de los dos carriles: ODD (`axiom archive integrate --odd`) y SDD (verbo nuevo `axiom sdd archive-integrate`). Los dos comparten el plan, la sustitución por token completo, la composición nativa de `openspecdoc` y un escritor atómico de conjuntos de ficheros generalizado desde el `Apply` actual.

```
reqid (hoja)          Provisional, IDMap.Substitute, SeriesOf, NextNumber, Scan con máscara
   ▲
openspecdoc (hoja)    UnknownRequirementIDError, ProvisionalIDError, ParseChangeSpec, Declared
   ▲
isoevidence           RequirementKeys (válido|provisional), RewriteCubre, RewriteMatrix
   ▲
livespec              Trace, PlanIDs, PlanIntegration (ODD), PlanChange (SDD), writeSet atómico
   ▲
cli                   archive integrate --odd [--check]; sdd archive-integrate [--check]; archive-check
sddstatus             decorateRequirements → status.requirements (usa solo openspecdoc + reqid)
```

Fuentes: `proposal.md` y las seis deltas de `specs/` (REQ-47.1 a REQ-47.9; MODIFIED REQ-21.17, REQ-30.2, REQ-30.3, REQ-30.4, REQ-30.22, REQ-29.3, REQ-29.4, REQ-24.58). Las divergencias con la spec están en la sección «Alineación con la spec».

## Decisiones aceptadas por la coordinación del plan

Las tres siguientes llegan ya aceptadas; el design las adopta y las desarrolla.

| # | Decisión | Motivo |
|---|---|---|
| A1 | Un provisional puede llevar una **etiqueta de serie nueva** solo con patrones que tengan `<n>`. Con patrones con `<DOM>` el dominio va fijado (un dominio nuevo se escribe literal: `prov:FAC:1`). | El formato está delegado en `docs/plan/backlog-2026-11/README.md:370`; el «dominio ya fijado» de `03-openspec-unica.md:100` sale del análisis, no del usuario. Un código de dominio no se puede inventar; un número de serie sí. |
| A2 | Un **ADDED con ID definitivo sigue siendo válido**, con la comprobación de colisión de siempre (`CollisionError`, REQ-30.4; REQ-29.3 en ISO). | D-03-4 exige declarar lo que se crea, no prohíbe el definitivo. D-03-8 lo necesita para C-03-c, y el archive de este cambio usa REQ-47 con el binario actual. |
| A3 | El equivalente ODD de `status` (03:107) es `axiom archive integrate --odd <feature> --check`: JSON, solo lectura y el mismo `requirements: {modifies, creates}` que `axiom sdd status --json`. | Evita un status ODD completo (fuera de alcance) y reutiliza el plan real, de modo que lo previsto y lo integrado no divergen. |

## Decisiones de arquitectura

### D1. Formato literal del provisional: `prov:<clave>:<ordinal>`

**Elección.** Gramática independiente del patrón:

```
provisional := "prov:" clave ":" ordinal
clave       := "" | componente ("-" componente)*
componente  := dominio   [A-Z][A-Z0-9]*      (valor de <DOM>)
             | serie     [0-9]+              (valor de <n>, serie fijada)
             | etiqueta  [a-z][a-z0-9]*      (serie nueva; solo en la posición de <n>)
ordinal     := [1-9][0-9]*
```

La validación contra el patrón (`Pattern.ParseProvisional`) exige que los componentes coincidan en número, orden y clase con los marcadores `<DOM>` y `<n>` de la plantilla. Ejemplos: patrón por defecto `prov:30:1` (serie 30) y `prov:a:1` (serie nueva «a»); `RF-<DOM>-<NN>` `prov:USU:1`; `RF-<DOM>-<n>.<m>` `prov:USU-3:1` o `prov:USU-b:1`; serie única `RF-<m>` `prov::1`.

- **Ubicación en el encabezado:** el mismo sitio que el definitivo, último paréntesis del título: `### Requirement: Cobro con IVA (prov:30:1)`. `openspecdoc` no cambia su parser: `Requirement.Name` conserva el texto y `reqid.ProvisionalInTitle(name)` extrae el último `(prov:…)`.
- **Token en texto:** `reqid.ScanProvisional(text)` reconoce `prov:[A-Za-z0-9-]*:[0-9]+` con la regla de token completo (ni precedido ni seguido de letra, cifra, `_` o `-`); un token que encaja con la forma laxa pero no con la gramática es «provisional mal formado» (error tipado, nunca se ignora).
- **El ordinal no es el número asignado.** Solo distingue provisionales de la misma serie dentro del cambio. El número sale del orden de aparición (REQ-47.2). La guía lo dice explícitamente.
- **Unicidad:** el token completo es único dentro del cambio o del documento ODD (REQ-47.1.3).

**Alternativas descartadas.**

| Alternativa | Motivo del rechazo |
|---|---|
| `REQ-30.new1`, `RF-USU-n1` (forma del patrón con minúscula) | Empieza por el literal inicial: `Scan` lo devolvería como candidato mal formado y `TrailingCandidate` (livingdoc) lo tomaría como ID. Viola REQ-47.1.1. |
| `RF-USU-N1` (mayúsculas) | Válido bajo plantillas como `RF-<DOM>-N<m>`: confundible. |
| Rango alto `RF-9001` (ids.go de `1166efca^`) | Confundible con un definitivo real; la proposal lo descarta. |
| `new-<serie>-<k>` | Ambiguo al separar serie y ordinal con claves que contienen `-`. |

**Motivo.** `prov` empieza en minúscula: nunca es `Valid` (invariante de `synthetic.go:9-12` y de la gramática de `template.go`), nunca encaja con `InTitle` (`\((<id>)\)` exige el literal inicial tras el paréntesis) ni con `TrailingCandidate`. Queda un caso límite: con `RF-<DOM>-<n>.<m>` y el dominio `RF`, la clave `RF-3` contiene `\bRF-`. Por eso **`Scan` enmascara los tokens provisionales antes de buscar candidatos**: el puerto es dueño de las dos formas y garantiza que un provisional nunca es candidato mal formado (REQ-47.1.1). Los dos puntos y la ausencia de espacios lo hacen compatible con `Cubre:` (separadores coma, espacio y tabulador) y con la sustitución por token.

### D2. Ampliación del puerto `internal/reqid`

**Elección.** API nueva, solo biblioteca estándar (se conserva `arch_test.go`):

- `type Provisional struct{ raw string; comps []component; ordinal int }` con `String()`, `Label() (string, bool)`.
- `ParseProvisionalToken(s) (string, bool)` (forma, sin patrón), `ProvisionalInTitle(title) (string, bool)`, `ScanProvisional(text) []Span` (posición, longitud, token).
- `(p Pattern) ParseProvisional(token) (Provisional, error)` → `*ProvisionalError{Token, Reason}` (componentes que no casan, etiqueta con `<DOM>`, etc.).
- `(p Pattern) SeriesOf(key string) (Series, bool)`: construye una serie desde los valores de `<DOM>` y `<n>`; resuelve la opacidad de `Series` sin exponer sus campos.
- `(p Pattern) NewSeries(fixed Series-sin-n, used []Series) Series`: siguiente `<n>` libre (máximo de las series que comparten los demás componentes, más uno).
- `type Allocator` (ocupación por serie): `Occupy(id)`, `Next(series) (string, error)`. Sustituye a `Registry.NextFree` (`livespec/registry.go:144`), que delega en él; una serie vacía da `1` con el relleno de la plantilla (`REQ-47.1`, `RF-FAC-01`). Desbordamiento o un formato que no sea `Valid` → `*AllocationError`.
- `type IDMap` + `Assignment{Provisional, Definitive string}`; `Substitute(text) string` por token completo, más largo primero, una sola pasada; `Count(text)`. Se reconstruye desde la idea de `IDMap` de `1166efca^`, sin familias por prefijo ni rango alto. El mapa admite claves definitivas (D→D') para la reasignación de D5.
- `Scan` enmascara los provisionales (D1).
- `arch_test.go` gana una regla: el literal `prov:` solo aparece en `internal/reqid` (código de producción), igual que la forma `REQ-`.

**Alternativas.** Poner `IDMap` en `openspecdoc` o `livespec`: rechazado, porque la regla de token es parte de la forma del identificador y la usan también `isoevidence` y `livespec` sin depender entre sí.

### D3. Plan único y escritura atómica

**Elección.** `livespec.PlanIDs(IDPlanInput) (IDPlan, error)` recibe todas las capacidades del cambio o del documento a la vez y devuelve un `reqid.IDMap` más las entradas de traza, sin escribir. Después se sustituye el texto de cada delta, se compone en memoria y se escribe todo con un único `writeSet`.

Algoritmo determinista:

1. **Orden.** SDD: capacidades por nombre de carpeta (`specs/<cap>/`). ODD: orden del documento. Dentro de cada capacidad, orden de aparición de los ADDED.
2. **Validación previa** (sin escribir; primer error rechaza todo): provisional mal formado o incompatible con el patrón; provisional duplicado; provisional en MODIFIED, REMOVED o en cualquiera de los nombres de RENAMED; provisional citado sin ADDED que lo declare (D6 define el alcance); etiqueta con patrón sin `<n>`.
3. **Ocupación.** `Allocator` cargado con: el registro (`openspec/specs/` + deltas de cambios activos, con `ExcludeChange` = el propio cambio SDD) **más los ADDED definitivos del propio cambio** (hueco 2 de la revisión de cobertura) **más** las entradas `kept` de la traza. Así una serie nueva nunca recae en una serie fijada por un definitivo del cambio y un siguiente libre nunca pisa un definitivo propio.
4. **Asignación.** Series fijadas: `Next(series)` en orden. Etiquetas nuevas: en orden de primera aparición, `NewSeries` y números consecutivos desde 1; etiquetas distintas → series distintas.
5. **Traza previa** (D5): entradas ya asignadas se conservan (`kept`) o se reasignan (`reassigned`).

**Escritura.** `Integration.Apply` se generaliza a `writeSet(files []fileWrite, opts)`: temporal por fichero con nombre `.<base>.axiom-tmp` junto al destino (los ficheros con punto son estado local para `docpaths`, nunca documentación publicada), renombrado de todos, restauración en orden inverso ante cualquier fallo y retirada de carpetas creadas. La regeneración de `openspec/INDEX.md` es una opción: ODD la conserva («solo si ya existe», criterio actual de REQ-1.2); SDD no la hace, porque la skill sigue ejecutando `axiom archive sync` (REQ-21.17 (4)).

**Alternativas.** Planificar por dominio (como `archive-compose`): rechazado, dos capacidades recibirían el mismo número (REQ-47.3). Escribir con `git` (stash/commit): rechazado, el motor no ejecuta procesos y el patrón temporal+rename ya existe y está probado.

### D4. Verbo SDD: `axiom sdd archive-integrate`

**Elección.**

```
axiom sdd archive-integrate --cwd <dir> --change <cambio>
    [--supersede] [--superseded-requirements "<a>, <b>"] [--check] [--json]
```

- Esquema JSON `axiom.sdd-archive-integrate/v1`: `schema`, `change`, `check`, `capabilities[] {name, path, outcome: created|composed|already_integrated}`, `requirements {modifies[], creates[]}`, `assignments[] {provisional, definitive, capability, requirement, state: assigned|kept|reassigned, previous?}`, `duplicates[] {id, capabilities[], blocking}`, `written[]`, `nothingToIntegrate`.
- Guardas en el orden de `archive-compose`: `baseline:` residual (`GuardSpecsWrite`), patrón accesible (`projectIDPattern`, REQ-44.9), guarda A de rama de archivo (`guardComposeLooseDelta`, REQ-25.19), negativa ISO (`sddstatus.ISOArchiveRefusal`). `--check` no aplica la guarda A (no escribe) y sí el resto.
- Por capacidad: canónica existente → `openspecdoc.Compose(canónica, deltaSustituida, opts)` con la delta **entera** (misma llamada que `archive-compose`: paridad byte a byte); canónica inexistente → **copia del texto de la delta con la sustitución aplicada** (sin provisionales, byte a byte igual al `cp` de hoy). Si todos los requisitos de un dominio ya están integrados (clasificación de `livespec`, la misma que ODD) → `already_integrated`; si solo algunos → error tipado `partially_integrated` con la lista (caso heredado de un archivado por dominio interrumpido; hueco 11).
- Reescrituras del cambio, solo cuando el mapa no está vacío: deltas `specs/<cap>/spec.md` (títulos y cuerpos), líneas `Cubre:` de `tasks.md` y `tasks.<rol>.md`, columna `Requisito` de la matriz de `verify-report*.md` (D8), y la traza `ids.yaml`.
- Delta plana `specs/spec.md` → error tipado `flat_delta_unsupported` (continuación: moverla a `specs/<capacidad>/spec.md`).
- `--check` sobre un cambio ya archivado (`changes/archive/*-<cambio>/ids.yaml`): verifica que cada definitivo de la traza aparece una sola vez en la spec viva y en su capacidad; si no, `duplicate_assigned_id` (D5).

Cambios en la skill `sdd-archive`: el bucle por dominio (`archive-compose` + `mv`) y el bloque `cp` de capacidad nueva se sustituyen por **una** invocación de `axiom sdd archive-integrate ... --json` tras `axiom sdd archive-check`; después `axiom archive sync`; la instantánea y el `mv` (Step 3) se quedan igual y ocurren después, de modo que el `diff -r` cubre `ids.yaml` y las reescrituras. El informe `archive-report.md` incluye la tabla de `assignments` del JSON. `archive-compose` sigue documentado como herramienta de vista previa y composición manual.

`axiom sdd archive-check` gana dos campos aditivos: `assignments` (leído de `ids.yaml`) y `pendingProvisionals` (tokens que quedan en las deltas). Tras el verbo, la delta ya lleva definitivos, así que `alreadyComposed` (D17) funciona sin cambios.

**Alternativas.** Modo `axiom archive integrate --change <c>` junto a `--odd`: rechazado; las guardas, la negativa ISO, la reescritura de artefactos SDD y la traza en fichero son propias del carril SDD, y los verbos SDD del archivado ya viven en `axiom sdd` (`archive-compose`, `archive-check`). Ampliar `archive-compose` a varias capacidades: rechazado, rompería su contrato de un dominio por invocación y su salida byte a byte.

### D5. Asignación en la rama y reintegración idempotente

**Elección.** Se asigna en la rama: ODD en la rama de la feature, SDD en la rama de archivo. La traza es la fuente de identidad de lo ya asignado.

Para cada ADDED, con la traza previa:

| Título del ADDED | Traza | Estado en la spec viva | Resultado |
|---|---|---|---|
| Provisional P | sin entrada | — | `assigned`: siguiente libre |
| Provisional P | P→D | D libre | `kept`: reutiliza D |
| P o definitivo D | P→D | D en la misma capacidad con cuerpo igual módulo sustitución | `kept`, `already_integrated` |
| P o definitivo D | P→D | D ocupado por otro requisito (otra capacidad, u otro cuerpo) que no es el nuestro | `reassigned`: siguiente libre D'; mapa D→D' sobre documento, delta, `Cubre:`, y nuestro propio bloque en la spec viva si ya estaba |
| Definitivo D | sin entrada | — | elegido por el autor: colisión de siempre (A2) |

- **«Nuestro» bloque** = el requisito de la capacidad de la entrada cuyo cuerpo normalizado (`NormalizeBody`) es igual al del ADDED con la sustitución aplicada. Cualquier otra aparición es ajena y gana: la rama que integra después cede.
- **Serie nueva en colisión (hueco 8):** si una entrada de una etiqueta se reasigna porque su serie la ocupó otro cambio, se reasigna **la etiqueta entera** a una serie nueva, con números consecutivos desde 1, en todas las capacidades. Con serie fijada se reasigna solo la entrada afectada.
- **Duplicados (REQ-47.5):** tras componer en memoria, se cuentan las apariciones de cada ID que escribe la integración (asignados, reasignados, conservados y definitivos propios) en todas las capacidades resultantes, superados incluidos. Más de una → `duplicate_assigned_id` (bloquea). Los duplicados históricos que no tocan esos IDs se informan en `duplicates` con `blocking: false` (la spec viva de Axiom ya tiene 12, de `REQ-1.1` a `REQ-3.3`).
- **ODD tras rebase:** con conflicto git en `openspec/specs/`, la guía ordena quedarse con la versión de la base y volver a ejecutar la integración; sin conflicto, la reejecución detecta la ocupación y reasigna. La traza del documento lleva `previous` en las reasignadas.
- **SDD tras rebase (hueco 6):** el cambio archivado es inmutable (REQ-21.18), así que no se reasigna en sitio. La rama de archivo se descarta (`axiom branch done --force`, flujo ya documentado en la skill) y se vuelve a archivar desde `main`, donde la carpeta del cambio aún tiene los provisionales. La detección sin conflicto git la da `axiom sdd archive-integrate --check --change <c>` sobre el cambio archivado (D4); la skill ordena ejecutarlo tras cada rebase de la rama de archivo y antes de pedir la fusión.

**Alternativa descartada.** Asignar tras la fusión en `main`: exige un paso posterior inexistente y deja provisionales en la spec viva (proposal). No reescribir el documento ODD y guardar solo el mapa: contradice REQ-30.2.

### D6. Alcance de validación y de reescritura (hueco 4)

| Texto | ¿Se valida que todo provisional esté declarado? | ¿Se reescribe? |
|---|---|---|
| Delta del cambio (títulos y cuerpos) | Sí | Sí, entera |
| Ficheros de tareas, líneas `Cubre:` | Sí | Sí, solo el token, resto byte a byte |
| Ficheros de tareas, resto del texto | Sí | No (la traza explica la mención) |
| `verify-report*.md`, columna `Requisito` de la matriz | Sí | Sí (D8) |
| Documento ODD, valla `openspec` de `## Spec viva` y `Cubre:` fuera de vallas | Sí | Sí |
| Documento ODD, resto (salvo la sección de traza) | Sí | No |
| `proposal.md`, `design.md`, `explore.md`, informes | No | No |

«Declarado» = un ADDED del cambio **o** una entrada de la traza (así la reejecución tras reescribir no rechaza las menciones en prosa). La prosa de diseño queda fuera para que los ejemplos de un design no bloqueen ni se reescriban.

### D7. Errores tipados

Un único tipo en `livespec`, `*IDPlanError{Code, Capability, Section, Requirement, Token, File, Line, Continuation}`, con códigos estables (también en el JSON de `--check`):

| Código | Caso |
|---|---|
| `provisional_malformed` | token `prov:` que no cumple la gramática o los componentes del patrón |
| `provisional_new_series_unsupported` | etiqueta con patrón sin `<n>` (A1) |
| `provisional_outside_added` | provisional en MODIFIED, REMOVED o RENAMED |
| `provisional_duplicate` | mismo provisional en dos ADDED del cambio |
| `provisional_undeclared` | citado (D6) sin ADDED ni traza |
| `id_allocation_failed` | desbordamiento o definitivo no `Valid` |
| `duplicate_assigned_id` | REQ-47.5 |
| `partially_integrated`, `flat_delta_unsupported` | D4 |

En `openspecdoc`: `*UnknownRequirementIDError{Section, Requirement, ID}` sale de `Compose` cuando el nombre lleva un ID válido y `Resolve` da `MatchNone` (`compose.go:~300`); su `Unwrap()` devuelve el `*UnappliedDeltaError` equivalente, de modo que `errors.As` de los llamadores actuales sigue funcionando. `*ProvisionalIDError{Section, Requirement, Token}` sale de `Compose` ante cualquier provisional; `archive-compose` lo envuelve con la continuación a `axiom sdd archive-integrate`. Un MODIFIED sin ID sigue resolviéndose por título (REQ-30.22) y conserva el rechazo actual si no casa; en `modifies` aparece sin `id` (D9). «Provisional igual a un definitivo» es imposible por construcción (D1) y no tiene código. `added_without_id` (ODD) se conserva para un ADDED sin definitivo ni provisional, con el texto que sugiere el provisional.

### D8. ISO: provisionales en `isoevidence` y matriz (hueco 3)

`isoevidence.RequirementKeys(name, p) (valid, provisional, malformed []string)` sustituye a `RequirementIDs` en `CheckRequirementIDs`, `deltaIDs`, `CheckCoverage` y `AffectedFromDelta`: el «identificador» de un requisito de la delta es el válido o el provisional. Reglas nuevas de REQ-29.3 con el código existente `iso_requirement_id`; un provisional no declarado en `Cubre:` usa el código existente `iso_cubre_unknown` con detalle propio (sin tokens nuevos en la tabla del contrato). La matriz de `verify-report.md` admite el provisional de un ADDED (la clave de escenario coincide con `AffectedFromDelta`) y se reescribe al archivar con `RewriteMatrix`, porque tras la reescritura de la delta un `archive-check` reanudado compararía definitivos de la delta con provisionales de la matriz y bloquearía.

### D9. Proyección `requirements` en `status`

`internal/sddstatus/requirements.go`: `decorateRequirements(status, location)`, llamado tras `decorateISO` (`status.go:546`), con `Requirements *RequirementsProjection json:"requirements,omitempty"` en `Status` y proyectado sin cambios en `status_v2.go` (como `ISO`). Deriva de `openspecdoc.Declared(capability, delta, pattern)`, función pura compartida con `--check` ODD y SDD.

```json
"requirements": {
  "modifies": [{"capability": "pagos", "id": "REQ-30.1", "operation": "MODIFIED", "requirement": "Cobro con IVA (REQ-30.1)"}],
  "creates":  [{"capability": "pagos", "id": "prov:30:1", "provisional": true, "operation": "ADDED", "requirement": "Cobro (prov:30:1)"}]
}
```

- `id` se omite si el nombre no lleva ninguno (MODIFIED o REMOVED por título fuera de ISO, RENAMED cuyo nombre antiguo no lo tiene, ADDED sin ID); `requirement` siempre está (hueco 5). RENAMED usa el nombre antiguo.
- Capacidad nueva escrita como spec completa (sin secciones de delta): `openspecdoc.ParseChangeSpec` trata sus requisitos como ADDED.
- Se omite sin deltas, con almacén `engram`/`none`, con cambio archivado y con el patrón inaccesible (REQ-44.9: no se evalúa con un patrón no declarado). No toca `nextRecommended`, `dependencies` ni motivos.
- Contrato: sección nueva «Requirements» en `_shared/sdd-status-contract.md`.

### D10. Digest de puertas

Comprobado en `internal/kickoff/machine.go:252` (`resolveGateStatus`): **una aprobación es terminal**, con independencia del digest actual. El archivado solo se alcanza con las puertas aprobadas, así que reescribir `tasks.md`, las deltas o `verify-report.md` no devuelve ninguna puerta a `pending`; en `continuous` no se evalúan puertas (`governance.go:434`). La autoridad de edición (`edit_authority.go`) mira rutas, que la reescritura de `Cubre:` no toca. No hace falta reordenar nada; se cubre con una prueba de regresión (Estrategia de pruebas, G1).

### D11. Lectores de deltas activas (hueco 10)

| Lector | Efecto del provisional | Tratamiento |
|---|---|---|
| `livespec/registry.go:109` (colisión con cambios activos) | `InTitle` → "": no se indexa ni colisiona | Prueba de no regresión |
| `livingdoc/indexer.go:143` | Solo lee `openspec/specs/`, que nunca tiene provisionales | Prueba: `(prov:…)` no es `TrailingCandidate` |
| `livingdoc/synthesizer.go` | Lee el `spec.md` de un cambio (flujo heredado) | Sin cambio; documentado |
| `dashboard` (`odd_livespec_test.go`) | Cuenta casillas; la traza no tiene | Prueba: el recuento no cambia con `## Identificadores asignados` |
| `adoption/adoptcheck/check.go:152` | Un provisional cuenta como «sin identificador» | Fuera de alcance (C-03-c); se documenta |
| `cli/sdd_trace.go:364` | Ignora el provisional hasta archivar | Pasa a `RequirementKeys` para no perder la fila |
| `sddstatus/iso.go:129` | D8 | D8 |

### D12. Traza

Formato único, `livespec.Trace`, codificado en YAML (`gopkg.in/yaml.v3`, ya en `go.mod`):

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

- SDD: `openspec/changes/<cambio>/ids.yaml` (sin punto: es documentación, se publica y se archiva con la carpeta). Sin asignaciones no se escribe.
- ODD: sección `## Identificadores asignados` al final del documento con un único bloque vallado `yaml` con el mismo contenido. Se excluye de la sustitución y de la validación de provisionales no declarados.
- Informes: `assignments` en `axiom.odd-integration/v1` y en `axiom.sdd-archive-integrate/v1`; tabla en `archive-report.md`.

## Flujo de datos

```
deltas / doc ODD ──► ParseChangeSpec ──► PlanIDs ◄── Registry (specs + activos − propio)
                                           │  ◄── definitivos propios + traza previa
                                           ▼
                                        IDMap ──► Substitute(deltas) ──► classify ──► Compose / copia
                                           │                                              │
                                           ├──► RewriteCubre / RewriteMatrix / doc ODD    │
                                           └──► Trace ─────────────────────────┐          │
                                                                               ▼          ▼
                                             duplicados (REQ-47.5) ◄── vista en memoria de la spec viva
                                                                               │
                                                         --check ──► informe JSON (sin escribir)
                                                         real    ──► writeSet atómico ──► informe
```

## Cambios de ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/reqid/provisional.go` | Crear | Gramática, parseo, `ProvisionalInTitle`, `ScanProvisional`, `ProvisionalError` |
| `internal/reqid/idmap.go` | Crear | `IDMap`, `Substitute`, `Count`, regla de token |
| `internal/reqid/allocate.go` | Crear | `SeriesOf`, `NewSeries`, `Allocator`, `AllocationError` |
| `internal/reqid/reqid.go` | Modificar | `Scan` enmascara provisionales |
| `internal/reqid/arch_test.go` | Modificar | `prov:` solo en `reqid` |
| `internal/openspecdoc/compose.go` | Modificar | `UnknownRequirementIDError`, `ProvisionalIDError` |
| `internal/openspecdoc/declared.go` | Crear | `ParseChangeSpec`, `Declared`, `DeclaredRequirement` |
| `internal/isoevidence/reqid.go`, `cubre.go`, `matrix.go`, `scenarios.go` | Modificar | `RequirementKeys`, reglas de D8, `RewriteCubre`, `RewriteMatrix` |
| `internal/livespec/ids.go` | Crear | `PlanIDs`, `IDPlanError`, reasignación, duplicados |
| `internal/livespec/trace.go` | Crear | `Trace` YAML, lectura y escritura (fichero y sección ODD) |
| `internal/livespec/apply.go` | Modificar | `writeSet` generalizado; `Apply` lo usa |
| `internal/livespec/plan.go` | Modificar | Provisionales en ODD; `checkIdentifiers` sobre el plan |
| `internal/livespec/change.go` | Crear | `PlanChange` (SDD) |
| `internal/livespec/odddoc.go` | Modificar | Reescritura del documento y sección de traza |
| `internal/livespec/registry.go` | Modificar | `NextFree` delega en `reqid.Allocator` |
| `internal/livespec/report.go` | Modificar | `assignments`, `requirements` (solo `--check`), `duplicates`, `check` |
| `internal/cli/archive_integrate.go` | Modificar | `--check`, ayuda, texto |
| `internal/cli/sdd_archive_integrate.go` | Crear | Verbo nuevo |
| `internal/cli/sdd_archive_compose.go` | Modificar | Continuación del `ProvisionalIDError` |
| `internal/cli/sdd_archive_check.go` | Modificar | `assignments`, `pendingProvisionals` |
| `internal/cli/sdd_trace.go` | Modificar | `RequirementKeys` |
| `cmd/axiom/main.go` | Modificar | Registro de `archive-integrate` en `runSDD` |
| `internal/sddstatus/requirements.go` | Crear | `decorateRequirements` |
| `internal/sddstatus/status.go`, `status_v2.go` | Modificar | Campo `Requirements` |
| `internal/assets/skills/sdd-archive/SKILL.md` | Modificar | Paso único Go; traza en el informe; `--check` tras rebase |
| `internal/assets/skills/sdd-spec/SKILL.md` | Modificar | Guía del provisional (L257 y sección de ADDED) |
| `internal/assets/skills/_shared/sdd-iso-convention.md` | Modificar | Provisional en ADDED, `Cubre:` y matriz |
| `internal/assets/skills/_shared/sdd-status-contract.md` | Modificar | Sección `requirements` |
| `internal/components/agentguidance/routing.go` (+ `routing_test.go`, `internal/workspace/docs_pin_test.go`) | Modificar | Paso 7 ODD (REQ-24.58, regla 3) |
| `testdata/golden/*` | Verificar | En esta rama ningún golden contiene el paso 7 (`grep` de `Spec viva` y de `archive integrate --odd`: 0 de 89); la cifra de 13 de la exploración está obsoleta. Se regeneran solo si una prueba de goldens falla |

## Interfaces

```go
// internal/reqid
func ProvisionalInTitle(title string) (token string, ok bool)
func ScanProvisional(text string) []Span           // Span{Start, End int; Token string}
func (p Pattern) ParseProvisional(token string) (Provisional, error)
func (p Pattern) SeriesOf(key string) (Series, bool)
type Allocator struct{ /* pattern, max por serie, ocupados */ }
func NewAllocator(p Pattern) *Allocator
func (a *Allocator) Occupy(id string)
func (a *Allocator) Next(s Series) (string, error)
func (a *Allocator) NewSeries(pv Provisional) (Series, error)
type IDMap struct{ /* … */ }
func NewIDMap(pairs []Assignment) IDMap
func (m IDMap) Substitute(text string) string

// internal/openspecdoc
type UnknownRequirementIDError struct{ Section, Requirement, ID string }
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
    Citations    []Citation         // tokens de Cubre, tareas, matriz, documento
}
type IDPlan struct{ Map reqid.IDMap; Entries []TraceEntry }
func PlanIDs(in IDPlanInput) (IDPlan, error)
func PlanChange(loc workspace.Location, change string, opts ChangeOptions, reg *Registry) (ChangeIntegration, error)
func (c ChangeIntegration) Apply() ([]string, error)
```

## Estrategia de pruebas

TDD por paquete: RED observado antes de cada producción. Comandos: `go test ./internal/reqid/...`, `./internal/openspecdoc/...`, `./internal/isoevidence/...`, `./internal/livespec/...`, `./internal/sddstatus/... -run '<Requirements|Gate|ISO>'`, `go test ./internal/cli -run '<ArchiveIntegrate|SDDArchive>'` (el paquete `cli` completo no termina en esta máquina) y `go vet` por paquete. Nunca `go build -o`: los arneses van en proceso.

| Capa | Qué | Cómo |
|---|---|---|
| reqid | Un provisional nunca es `Valid`, `InTitle` ni candidato de `Scan` con `REQ-<n>.<m>`, `RF-<DOM>-<NN>`, `RF-<m>`, `HU-<NNN>` y `RF-<DOM>-<n>.<m>` con dominio `RF`; gramática; etiqueta con `<DOM>` rechazada; `Next` en serie vacía (`REQ-47.1`, `RF-FAC-01`); `NewSeries`; desbordamiento; `Substitute` por token (prefijo, pegado a letra, mayúsculas) | Tablas; extensión de `fuzz_test.go` con la propiedad «provisional ⇒ no válido» |
| reqid | `NextFree` del registro igual que antes | `registry_test.go` y `registry_pattern_test.go` sin cambios |
| openspecdoc | Corpus `compose_corpus_test.go` byte a byte; `UnknownRequirementIDError` en MODIFIED, REMOVED y RENAMED antiguo, y `errors.As` a `UnappliedDeltaError`; `ProvisionalIDError`; `Declared` (sin ID, RENAMED, spec completa) | Unitarias |
| isoevidence | REQ-29.3 (tabla nueva), REQ-29.4 con provisional, matriz con provisional; `RewriteCubre` conserva bytes y posición | Unitarias |
| livespec | Plan determinista; dos capacidades consecutivas; definitivos propios ocupados (hueco 2); serie nueva entre capacidades; errores de D7; reasignación individual y por etiqueta; duplicado bloqueante e histórico informativo; fallo inyectado en cada escritura de `writeSet` sin temporales restantes; documento ODD sin provisionales intacto | `writeTempFile`/`renameFile` como costuras |
| cli ODD | `--check` no escribe (hash del árbol antes y después), JSON con `requirements` y `assignments`, salida 0/≠0; reintegración tras rebase simulado (dos documentos en serie); documento ya integrado → `kept`, `nothingToIntegrate` | `t.TempDir()` con workspace de prueba |
| cli SDD | Capacidad existente y nueva, serie nueva con los dos patrones; cambio sin provisionales: spec viva igual byte a byte a `archive-compose` + `cp`, sin `ids.yaml`; fallo inyectado; reejecución idempotente; `archive-check` con `assignments`; `--check` en cambio archivado con duplicado | En proceso |
| G1 puertas | Cambio `checkpointed` con `spec`, `design`, `tasks` y `role-apply:fullstack` aprobados: tras el verbo (que cambia el digest de specs y tareas) las puertas siguen `approved` y `nextRecommended` no cambia | `sddstatus` + `kickoff` |
| sddstatus | `requirements` presente y ordenado; ausente sin deltas (byte a byte igual al fixture previo); `nextRecommended` y `dependencies` iguales con y sin el campo | Fixtures existentes |
| Lectores | D11 | Una prueba por lector |
| Guías | Paso 7 (`routing_test.go`, `docs_pin_test.go`); pruebas de contenido de skills si existen | `go test ./internal/components/agentguidance/... ./internal/workspace/...` |

## Matriz de amenazas

| Límite | Aplicabilidad | Respuesta | Pruebas RED |
|---|---|---|---|
| Rutas tipo documentación | N/A: no se clasifica ni se ejecuta ningún fichero | — | — |
| Selección de repositorio git | Aplicable: el verbo SDD resuelve el workspace con `--cwd` (`workspace.Locate`) y aplica la guarda A existente (`engine.RequireChangeBranch`) | Escribe solo bajo el `workspaceRoot` resuelto; ruta relativa a `--cwd`; fuera de la rama de archivo rechaza sin escribir (salvo `--check`) | Ejecutar con el proceso en otro directorio y `--cwd` absoluto: solo cambian ficheros del workspace; en una rama que no es de archivo: rechazo y árbol intacto |
| Estado de commit | N/A: el motor no hace commit (la skill tampoco) | — | — |
| Estado de push | N/A | — | — |
| Comandos de PR | N/A | — | — |

## Unidades de trabajo (un commit por unidad)

| U | Unidad | Paquetes | Depende de |
|---|---|---|---|
| U1 | Puerto: provisional, `IDMap`, `Allocator`, máscara de `Scan`, regla de arquitectura; `NextFree` delega | `reqid`, `livespec/registry.go` | — |
| U2 | Errores tipados y `Declared`/`ParseChangeSpec` | `openspecdoc`, `cli/sdd_archive_compose.go` | U1 |
| U3 | ISO con provisionales y reescrituras `Cubre:`/matriz | `isoevidence`, `cli/sdd_trace.go` | U1 |
| U4 | Plan único, traza, `writeSet`, duplicados | `livespec` | U1–U3 |
| U5 | ODD: reescritura del documento, reintegración, `--check`, informe | `livespec`, `cli/archive_integrate.go` | U4 |
| U6 | SDD: `PlanChange`, verbo `archive-integrate`, `archive-check`, prueba G1 | `livespec`, `cli`, `cmd/axiom` | U4 |
| U7 | `status.requirements` y contrato | `sddstatus`, `_shared/sdd-status-contract.md` | U2 |
| U8 | Guías: `sdd-archive`, `sdd-spec`, `sdd-iso-convention`, paso 7 ODD y sus pins; comprobación de goldens | skills, `agentguidance`, `workspace` | U5–U7 |

## Alineación con la spec

- **Ocupación:** REQ-47.2 habla del máximo «en la spec viva». El design cuenta además los definitivos de los cambios activos (mismo alcance que la colisión de REQ-30.4) y los ADDED definitivos del propio cambio (instrucción de la coordinación). Con una spec viva sin cambios activos que declaren la serie, el resultado es idéntico al de los escenarios.
- **Matriz de verify:** D8 reescribe la columna `Requisito` de `verify-report*.md`, que REQ-47.4 no enumera entre los destinos. Es necesario para que un archivado ISO reanudado no se bloquee; conviene añadirlo a REQ-29.6 o a REQ-47.4 en una iteración de la spec.
- **Reintegración SDD:** se resuelve rehaciendo el archivo (D5), no reasignando en sitio, por la inmutabilidad de REQ-21.18.
- **REQ-44.3:** no se modifica (la spec lo justifica); la nota de `## Purpose` de `requirement-identifiers` queda para la fase de archivo, como dice la delta.

## Migración y despliegue

No hay migración de datos. Las ramas abiertas con ADDED definitivos siguen igual (A2). El archive de este cambio usa REQ-47 con el binario anterior (proposal). La reversión deja la spec viva válida: nunca contiene provisionales.

## Preguntas abiertas

- [ ] Ninguna bloquea. Para la spec: incorporar la reescritura de la matriz (Alineación) y precisar «spec viva» frente a «registro» en REQ-47.2.
