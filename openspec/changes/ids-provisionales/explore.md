# Exploración: ids-provisionales (C-03-b)

Fuente: `docs/plan/backlog-2026-11/03-openspec-unica.md:94-129`. Rutas relativas a la raíz del worktree. Limitación: la herramienta de la fase no ejecutaba `git`, así que la lógica de `specbaseline/ids.go` se recupera del `design.md` archivado de inc-24, no del `show` del commit.

## 1. Puerto `internal/reqid` (C-03-a)

- API (`reqid.go`): `Pattern` (valor cero = `REQ-<n>.<m>`), `Default`, `Parse`, `Valid`, `InTitle`, `Scan`, `TrailingCandidate`, `Series`, `Number`, `Format(series, n)` (relleno de `<NN>`), `Template/Compact/SeriesCompact`. `resolve.go:38` `Resolve` (empareja por ID y cae a título), `synthetic.go:13` `Synthetic`.
- Declaración: `sdd.id_pattern` en `axiom.yaml` (`internal/workspace/sdd.go:22` `sddKeys = {"iso","id_pattern"}`; `declarations.go:57` `Declarations.IDPattern()`).
- Gramática (`template.go:15,53-203`): literal inicial en mayúsculas + separador; marcadores `<DOM>`, `<n>`, `<m>`, `<NN>..<NNNNNN>`; el literal solo admite mayúsculas, cifras, `.`, `_`, `-`. Por tanto ningún ID válido contiene minúsculas (`synthetic.go:9-12` lo usa como invariante).
- Serie: `Series` opaca (`series.go:7`, campos `prefix,suffix,key` privados); `Key()` une `<DOM>` y `<n>` con `-`. NO hay constructor de serie desde texto.
- Siguiente libre: no está en el puerto; está en `livespec/registry.go:144` `Registry.NextFree(series)` (máximo+1) y `plan.go:320` `nextFree` (salta los del documento). `registry.go:140-143` dice «la asignación de una serie vacía no es de este paquete (C-03-b)». Una serie sin ningún ID previo no se puede formar hoy.
- No contempla provisionales. Specs vivas: `openspec/specs/requirement-identifiers/spec.md` (REQ-44.x); archivo `openspec/changes/archive/2026-11-24-puerto-reqid/`.

## 2. Lógica borrada (`specbaseline/ids.go`, vía `design.md` de inc-24 §2.11, líneas 464-507)

- Familia = prefijo previo a la última serie de dígitos; máximo calculado sobre el documento original antes de aplicar operaciones; definitivos máximo+1.. en orden de delta; relleno a cero hasta el ancho del mayor.
- Provisional = número en rango alto (`RF-9001`, forma `9·10^d+1`); eso es CONFUNDIBLE con un definitivo, por eso se descarta como formato.
- `IDMap` (`PlanIDs`, `Substitute` por token completo: exacto, sensible a mayúsculas, no precedido ni seguido de letra/dígito/`_`/`-`; `CountProvisional`). Trazabilidad: `id_map: [{provisional, definitive, domain}]` en `integration.yaml` (design.md:1395-1413).
- Reutilizable: la idea de plan único previo a escribir con detección de duplicados (mismo provisional dos veces, provisional = definitivo de otro) y la sustitución por token completo. No reutilizable: familias por prefijo y rango alto; ahora manda `reqid`.

## 3. Integración ODD (`internal/livespec`)

- `plan.go:138` `PlanIntegration`: `CheckImpact` → `classify` (`:198`, ADDED por título exacto, MODIFIED/REMOVED por `openspecdoc.Resolve`) → `checkIdentifiers` (`:271`) → `compose` (`:335`) → `checkContext`. Todo en memoria; `apply.go:40` `Apply` escribe todo o nada (temporales + rename + restauración + `INDEX.md`).
- `checkIdentifiers` (`:286-292`): ADDED sin ID → `added_without_id`; ID ya existente → `CollisionError{NextFree}` (`:83`).
- Idempotencia: un ADDED cuyo título ya está en la canónica con el mismo cuerpo es `already` (`plan.go:234-241`). Con provisionales, el título cambia en cada reintento; hace falta un criterio de «ya integrado» (cuerpo normalizado igual bajo otro ID, o mapa guardado).
- `archive_integrate.go:29,117` y `:113`: hoy NO modifica el documento ODD («no modifica el documento ODD»). El flujo `RunArchiveIntegrate` (`:34-106`) es el sitio de la reescritura del documento; `Apply` debe ampliarse a incluir `odd/tasks/<feature>.md` en el conjunto atómico. El `Report` (`axiom.odd-integration/v1`) es donde irá el mapa.
- `odddoc.go`: `CapabilityDelta.Text` es el contenido de la valla `openspec`; reescribir el ID en el documento implica reescribir esa valla y las líneas `Cubre:` fuera de vallas.

## 4. Archivado SDD

- El archivado es dirigido por la skill (`sdd-archive/SKILL.md:143-193`): por cada dominio ejecuta `axiom sdd archive-compose --canonical --delta --output` + `mv`, después `axiom archive sync` y mueve la carpeta (`:250-320`, con `diff -r` obligatorio contra una instantánea previa). NO hay verbo Go que archive atómicamente todo el cambio.
- `internal/cli/sdd_archive_compose.go:131-148` carga el patrón, llama a `openspecdoc.Compose` y escribe por dominio. `openspecdoc/compose.go:315` `Compose`: orden RENAMED→MODIFIED→REMOVED→ADDED; `UnappliedDeltaError` (`:21`) rechaza sin componer parcial. Un MODIFIED/REMOVED cuyo ID no existe ya falla con `UnappliedDeltaError` («no canonical requirement named», `:357`), pero sin tipo propio para «ID inexistente»; y un MODIFIED sin ID cae a título sin error (`resolve.go:41-47`).
- Una capacidad nueva (sin canónica) se COPIA mecánicamente (`SKILL.md:197-237`): no pasa por Go. Con ADDED provisionales esa copia dejaría provisionales en la spec viva: hay que enrutarla por Go también.
- Asignación entre dominios: archive-compose procesa un dominio por invocación; la asignación debe ser determinista para todo el cambio (un plan previo que cubra todas las capacidades, como `PlanIDs`).
- Traza: lo natural es un fichero en la carpeta del cambio antes del `mv` (p. ej. `ids.yaml`/tabla en `archive-report.md`). `SKILL.md:354` exige que `tasks.md` archivado conserve sus bytes: la reescritura de `Cubre:` debe hacerse ANTES de la instantánea previa al `mv`.

## 5. `internal/openspecdoc`

- `parse.go:8-15` `Requirement{Name,Body,Line}`; el ID vive en el título (`### Requirement: Título (ID)`, `reqid.InTitle`). No existe reescritura de ID; hay `renameRequirementHeading` (`compose.go:235`, privado) y `annotateRequirementSuperseded`. Hace falta una operación nueva «sustituir ID en título y en cuerpos/`Cubre:`».
- `render.go:9` `RenderNewSpec` (capacidad nueva) renderiza los ADDED tal cual.

## 6. `isoevidence/cubre.go`

- `cubreLine` regex (`:12`), `ParseCubre` (`:27-43`) devuelve `CubreLine{File,Line,IDs}` pero NO guarda la posición de cada token ni el texto: reescribir exige una función nueva que opere sobre el texto (tokens separados por coma/espacio/tab, fuera de vallas). `CheckCoverage` (`:109`) usa `RequirementIDs`/`reqid.Valid`: un provisional (inválido por diseño) aparecería como «desconocido» (`CodeCubreUnknown`); hay que enseñarle a aceptar provisionales presentes en la delta.

## 7. Status

- `sddstatus/status.go:165-236` `Status`; las proyecciones aditivas siguen el patrón `ISO *ISOProjection` + `decorateISO` (`iso.go:402,533`) y `decorateWorkspaceDeclarations` (`workspace_declarations.go:31`). `requirements` sería un campo `omitempty` nuevo derivado de las deltas del cambio (ADDED→creates, MODIFIED/REMOVED/RENAMED→modifies). Contrato: `_shared/sdd-status-contract.md` (sección `iso` en `:96-142`, tabla de causas `:152`).
- No existe status ODD. Equivalente mínimo: el `Report` de integración ODD y/o una opción de solo lectura para `archive integrate --odd` (hoy no hay `--check`). La spec de status ODD es decisión del design.

## 8. Guías

- `_shared/sdd-iso-convention.md:18-19,24-25`: cada nombre de requisito lleva exactamente un ID válido; el ADDED es único y ausente de la baseline; `Cubre:` cita IDs. Habría que añadir el provisional como forma válida del ADDED y de `Cubre:`.
- `sdd-spec/SKILL.md:257` repite la exigencia del ID válido en ISO. `sdd-archive/SKILL.md`: pasos 2-3 arriba. `openspec-convention.md`: no menciona IDs (sin coincidencias de `id_pattern`).
- `agentguidance/routing.go`: el paso 7 de ODD (`## Spec viva`, `axiom archive integrate --odd`) está en este fichero; `REQ-<n>.<m>` ya no aparece literal (solo comentarios).
- `skills/_shared/sdd-orchestrator-sections.md` y `sdd-status-contract.md` citan identificadores.

## 9. Goldens (`testdata/golden/`)

Los que contienen `Spec viva` o `REQ-` (reglas de orquestador): `sdd-{windsurf-global-rules,vscode-instructions,opencode-multi-settings,kiro-instructions,gemini-geminimd,cursor-rules,codex-agentsmd,codex-agentsmd-powerful,codex-agentsmd-lowcost,claude-claudemd,antigravity-rulesmd}.golden` y `combined-{windsurf-global-rules,claude-claudemd}.golden` (13). El plan habla de 89 goldens; los 13 son los que fijan el texto del paso 7; el resto se regenera si cambia cualquier texto compartido. Se regeneran con el flujo habitual de goldens.

## 10. Caso real `carril-sdd-versatil`

Renumeró REQ-28→REQ-29 a mano al fusionar (plan, línea 96). En el archivo: `archive-report.md:11` aún cita REQ-28.3/28.4/28.6, mientras la spec viva vigente usa REQ-29 (REQ-29.x en `sdd-iso-evidence`), prueba de que la traza vieja y la viva divergen y de que no hubo mapa. Con provisionales ese renumerado manual desaparece.

## Opciones de design

### (a) Formato del ID provisional

Invariante: debe contener al menos una minúscula (ningún ID válido la tiene, `reqid/synthetic.go:9`); así es inconfundible bajo cualquier patrón. Debe llevar el dominio (la serie) y un ordinal por documento.

| Opción | Ejemplo | Pros | Contras |
|---|---|---|---|
| A. `prov:<serie>:<k>` con `<serie>` = `Series.Key()` | `prov:USU:1`, default `prov:30:1`, serie única `prov::1` | Mecánico; sin espacios (compatible con `Cubre:` y `InTitle`); `Key()` ya existe | Hay que añadir `SeriesFromKey` a `reqid`; `prov::1` feo |
| B. `new-<serie compacta>-<k>` | `new-RF-USU-1`, `new-REQ-30-1` | Legible, parece su futuro ID | Ambiguo al separar serie y ordinal; más parseo |
| C. Mayúsculas tipo `RF-USU-N1` | — | Bonito | Descartada: puede ser válido en patrones como `RF-<DOM>-N<m>` |

Recomendación: A. Con el patrón por defecto `REQ-<n>.<m>` el «dominio» es la serie `n`: el autor elige `prov:30:1` para continuar REQ-30.*; para una serie nueva, se admite una serie inexistente (primer número 1). Esto exige el puerto `reqid` con `Provisional{SeriesKey, Ordinal}`, `ParseProvisional`, `SeriesFromKey` (CAPA hoja nueva, API pública pequeña).

### (b) Asignar en la rama vs tras la fusión

| | En la rama (al integrar) | Tras la fusión (en main) |
|---|---|---|
| Colisión | Dos PR pueden asignar el mismo número; la segunda debe reintegrar | Ninguna: el máximo se lee de main |
| Traza en la PR | El documento ODD y la spec salen con IDs definitivos; revisables | La PR lleva provisionales; la spec viva los tendría hasta un paso posterior |
| Coste | Ninguno nuevo | Un paso extra tras fusionar (commit automatizado en main o PR de seguimiento) que hoy no existe en monorepo-embedded |
| Cumple `INDEX.md`/ISO | Sí | La spec viva transitoria con provisionales rompería `Valid` |

Recomendación: asignar en la rama, pero con reintegración idempotente y detectada. Razones: (1) el SDD ya archiva en su propia rama de archivo, tras fusionar el cambio, lo que equivale a «tras la fusión» para SDD; (2) ODD integra en la PR de la feature, y el modelo ODD no tiene paso posterior a la fusión; (3) la colisión se resuelve porque `LoadRegistry` calcula el máximo contra la spec viva de la rama; en un conflicto git en `openspec/specs/<cap>/spec.md` la segunda PR rebasa y vuelve a integrar. Para que sea reintegrable hace falta que el documento ODD conserve los provisionales hasta que la PR fusione, o que el mapa guardado permita revertirlos. Alternativa a evaluar en design: no reescribir el documento ODD y guardar solo el mapa; descartada si se mantiene el requisito del plan de reescribir el documento.

### (c) IDs mezclados en una delta

Reglas propuestas: ADDED = provisional obligatorio (un ID definitivo en ADDED sigue admitido por compatibilidad con specs ya integradas y con las ya existentes en ramas abiertas, con la comprobación de colisión actual `CollisionError`); MODIFIED/REMOVED/RENAMED(antiguo) = ID existente obligatorio con error tipado `UnknownRequirementIDError` si no existe, y rechazo de un provisional en esas secciones; RENAMED(nuevo) conserva el ID del antiguo. `Cubre:` admite provisionales de la propia delta y definitivos. Un provisional sin ADDED que lo declare es error. Provisional duplicado dentro del cambio, o igual a un definitivo existente, rechaza antes de escribir (REQ-24.59 históricos (c)/(d)).

## Estimación por paquete (L en total)

| Paquete | Trabajo | Tamaño |
|---|---|---|
| `internal/reqid` | `Provisional`, parse/format, `SeriesFromKey`, `NextNumber` (mover `NextFree`) | S |
| `internal/openspecdoc` | reescritura de ID en título/cuerpo, error tipado de ID inexistente, plan de asignación | M |
| `internal/livespec` | aceptar provisional, plan de asignación, `Apply` con documento ODD, `Report` con mapa, idempotencia | M-L |
| `internal/isoevidence` | `Cubre:` acepta/reescribe provisionales | S |
| `internal/sddstatus` + contrato | proyección `requirements`, tests de regresión de ausencia | M |
| `internal/cli` | nuevo verbo/modo de archivado SDD que asigne y escriba traza antes del `mv`; `archive-compose` | M |
| Skills/guías/goldens | `sdd-spec`, `sdd-archive`, `sdd-iso-convention`, status-contract, orquestador, 13 goldens | M |

## Riesgos

- Digest de gates: reescribir `tasks.md`/delta en el archivo antes del `mv` cambia el digest que juzgan las puertas; hay que confirmar que tras archivar no se reevalúan (no verificado aquí).
- La copia mecánica de capacidades nuevas en `sdd-archive` (sin canónica) debe pasar a Go.
- `isoevidence.CheckCoverage` hoy marcaría un provisional como desconocido.
- `Series` es opaca: construirla desde texto exige tocar el puerto.
- Sin `git` en la fase: la reconstrucción de `ids.go` viene del diseño archivado, no del código.
