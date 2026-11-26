# Propuesta: Migración de las specs propias de Axiom de `REQ-n.m` a IDs funcionales `RF-<DOM>-<NN>`

## Intención

Cumplir D-03-6 (C-03-e del plan `docs/plan/backlog-2026-11/03-openspec-unica.md`): las 49 capacidades vivas de Axiom pasan del esquema numérico `REQ-n.m` al esquema funcional `RF-<DOM>-<NN>` que el propio producto ya ofrece a los proyectos (C-03-a, A-03-1), sin reescribir la historia de `openspec/changes/archive/`.

Hoy Axiom no usa el esquema que recomienda. Además, el esquema heredado tiene IDs ambiguos: `REQ-1.1` a `REQ-4.2` se repiten en siete capacidades (`autoskills`, `living-documentation`, `local-web-dashboard`, `multi-project-hub`, `multi-role-fan-out`, `structured-handoffs`, `workspace-topology`), de modo que una cita sin contexto no identifica un único requisito. La migración ha de hacerse ahora: C-03-a, C-03-b y C-03-d ya están fusionados, no hay otro cambio SDD abierto en `openspec/changes/` y la sección 07 del plan enlazará requisitos de Axiom con terceros.

Éxito significa: todas las specs, el código, las pruebas, los goldens, los documentos ODD y la documentación viva citan los IDs nuevos; un mapa versionado y determinista resuelve los IDs antiguos; y `axiom trace REQ-24.3` responde con el ID nuevo.

## Alcance

### Dentro del alcance

1. **Declaración del patrón.** `axiom.yaml` declara `sdd.id_pattern: RF-<DOM>-<NN>`. El patrón por defecto `REQ-<n>.<m>` no cambia (REQ-44.2 de `requirement-identifiers` lo fija para proyectos sin declaración).
2. **Mapa versionado y determinista** `REQ-n.m → RF-<DOM>-<NN>`, indexado por **(capacidad, ID antiguo)**, generado en Go a partir de la tabla de 49 códigos de dominio aprobada (Engram `sdd/migrar-ids-rf/dominios`). Numeración `01`, `02`… por orden de aparición del requisito en su spec, **incluidos los 48 requisitos que hoy no tienen ID** (que reciben ID nuevo sin entrada antigua). Regenerarlo sobre las mismas specs produce un mapa idéntico byte a byte.
3. **Herramienta de reescritura mecánica** a partir del mapa, idempotente (sobre un fichero ya migrado no cambia nada) para poder reaplicarla al rebasar trabajo en vuelo. Sustituye por token completo (reutilizando `reqid.IDMap`/`Assignment`). Resuelve cada ID ambiguo por su contexto (la capacidad de la spec que lo contiene o la capacidad que nombra el entorno) y **falla de forma explícita**, sin escribir nada, ante un ID ambiguo sin contexto resoluble.
4. **Ejecución en proceso.** La herramienta y la generación del mapa se ejecutan desde `go test` (prueba o punto de entrada invocado en proceso); no se compila ni se ejecuta ningún binario suelto (EDR Cortex) y el `axiom` del `PATH` es el de `main`. La forma concreta la decide `sdd-design`, respetando el trinquete de código muerto de la CI (código de producción sin llamadas falla).
5. **Destinos de la reescritura** (fuera de `openspec/changes/archive/`):
   - `openspec/specs/**/spec.md`: los 465 títulos con ID, los 48 títulos sin ID, la prosa, las líneas `Cubre:` y los rangos de los títulos H1 («REQ-42.1 a REQ-42.9» en tres specs). La migración de títulos y de `Cubre:` debe ser **completa y verificable**: bajo el patrón RF, un título que conserve `REQ-n.m` queda sin ID en silencio (sale del índice, de `NextFree` y de la evidencia ISO).
   - `openspec/INDEX.md`, regenerado con la lógica de `axiom archive sync` (`internal/livingdoc/indexer.go`).
   - Código Go de producción: comentarios y los ~31 literales de mensajes que citan IDs propios (p. ej. `internal/branch/engine/verdict.go`, `internal/cli/sdd_gate.go`, `internal/cli/sdd_kickoff.go`, `internal/cli/sdd_integrate_retired.go`, `internal/techcontext/verify.go`, `internal/techcontext/plan.go`, `internal/sddstatus/worktree_changes.go`, `internal/sddstatus/governance_handoff.go`, `internal/kickoff/args.go`, `internal/kickoff/archived.go`), más `cmd/axiom/**` y `e2e/**`.
   - Pruebas Go: comentarios y pines de IDs propios (p. ej. `internal/kickoff/archived_test.go` con REQ-21.18, `internal/kickoff/apply_topology_test.go`, `internal/kickoff/machine_test.go`, `internal/sddstatus/declarations_*_test.go`, `internal/assets/sdd_iso_pins_test.go`, `internal/assets/assets_test.go`).
   - Assets Markdown (5 ficheros de `internal/assets/skills/**`) y los 10 goldens que derivan de `_shared/sdd-orchestrator-sections.md`, regenerados con `-update` (`internal/components/golden_test.go`).
   - `odd/tasks/*.md` (29 ficheros), `axiom.yaml`, `axiom.example.yaml`, `.github/workflows/ci.yml`, `.goreleaser.yaml` y los ficheros JS/CSS del dashboard que citan IDs.
   - `docs/**` vivo: `docs/manual-de-inicio.md`, `docs/openspec-config.md`, `docs/upstream-absorption-ledger.md` y `docs/plan/**`.
6. **Residuos declarados con criterio comprobable.** El criterio del plan («`rg 'REQ-[0-9]+\.[0-9]+'` fuera de `archive/` y del mapa no encuentra nada») se interpreta como «ninguna cita a un requisito propio de Axiom en forma `REQ-n.m`». Las apariciones legítimas que se conservan se declaran en una **lista de exclusión versionada**, cada una con su categoría y motivo, y una prueba Go recorre el repositorio y falla ante cualquier `REQ-n.m` que no esté en `openspec/changes/archive/`, en el mapa o en esa lista. Categorías previstas:
   - Fixtures de pruebas que ejercitan el patrón por defecto (`reqid`, `livespec`, `isoevidence`, `reqtrace`, `openspecdoc`, `sddstatus`, `cli`…).
   - La spec `requirement-identifiers` donde define y ejemplifica el patrón por defecto.
   - Ejemplos del patrón por defecto dirigidos a otros proyectos (p. ej. en el manual o en `axiom.example.yaml`, si su clasificación lo confirma).
   - Citas que nombran deliberadamente un ID antiguo como tal (p. ej. «`axiom trace REQ-24.3` resuelve al ID nuevo» en `docs/plan/backlog-2026-11/03-openspec-unica.md`).
   - Historia fuera de OpenSpec: `docs/archive/**` y `docs/audits/**` (instantáneas fechadas).
   - La carpeta de este cambio, `openspec/changes/migrar-ids-rf/`, que pasa a `archive/` al archivar.
   La granularidad (fichero, línea o literal) la fija `sdd-design`; los pines de mensajes de producción no son fixtures y se reescriben.
7. **`axiom trace` resuelve IDs antiguos** (`internal/reqtrace`, cableado en `internal/cli/trace.go`): acepta en la consulta un ID que no cumple el patrón efectivo cuando está en el mapa, y añade sus equivalentes a través del punto de extensión de alias. Un ID antiguo ambiguo resuelve a todos sus IDs nuevos. La consulta por un ID nuevo incluye su ID antiguo para encontrar los incrementos archivados que lo citan, **acotando por capacidad** la coincidencia de los antiguos ambiguos.
8. **Pruebas que leen las specs reales** (`internal/openspecdoc/compose_corpus_test.go`, `internal/reqid/legacy_equivalence_test.go`): hoy usan el patrón por defecto; tras la migración se añade una variante con el patrón RF para que sigan comprobando IDs y no caigan a coincidencia por título.
9. **Corrección de `openspecdoc.ParseDelta`** (`internal/openspecdoc/parse.go`) como tarea propia, con prueba RED en `./internal/openspecdoc` y comprobación en `./internal/livespec`: hoy toma un `### Requirement:` dentro de un bloque vallado del cuerpo como requisito de la delta, mientras que `Compose` sí respeta las vallas (`fencedRanges`).

### Fuera del alcance

- Reescribir `openspec/changes/archive/**`: es historia; el mapa resuelve sus IDs.
- Reescribir `docs/archive/**` y `docs/audits/**`: son instantáneas fechadas; se declaran como residuo.
- Cambiar el patrón por defecto `REQ-<n>.<m>` o la gramática de `reqid`.
- Reescribir los fixtures de pruebas que ejercitan el patrón por defecto.
- Reescribir memorias de Engram, ramas remotas o PR abiertas de terceros.
- Encadenar PR o pedir `size:exception`: una sola PR, un commit por tarea (P13 del README del plan, revisión 10).

## Capacidades

### Capacidades nuevas

Ninguna.

### Capacidades modificadas

- `requirement-trace`:
  - REQ-48.2 (validación del ID contra el patrón): un ID que no cumple el patrón efectivo pero figura en el mapa de IDs heredados se acepta y se resuelve; el resto conserva el error tipado. El escenario «`axiom trace REQ-1.1` falla con el error tipado bajo el patrón RF» se reescribe en consecuencia.
  - REQ-48.4 (punto de extensión de alias): deja de ser solo un punto preparado; el mapa de IDs heredados es una implementación cableada del resolutor, con resolución por capacidad y con un ambiguo que resuelve a varios IDs.
  - Probablemente un requisito ADDED que fija el contrato del mapa (ubicación, esquema versionado, clave por capacidad e ID antiguo, determinismo) como fuente de equivalencias de la traza. `sdd-spec` decide si va aquí o en `requirement-identifiers`; si va en esta última, se añade como capacidad modificada.

La migración de los títulos de las 49 specs no es un cambio de comportamiento: no genera deltas por capacidad, la aplica la herramienta.

## Enfoque

1. **Mapa primero.** Un generador Go recorre `openspec/specs/*/spec.md` en orden, asigna `RF-<DOM>-<NN>` por orden de aparición con la tabla aprobada y emite el mapa versionado con clave (capacidad, ID antiguo). Sus pruebas fijan el determinismo, la cobertura de los 513 requisitos y la ausencia de colisiones.
2. **Reescritura por destino.** La herramienta aplica el mapa por categorías (specs, Go, assets, ODD, docs), resuelve cada ambiguo por contexto y aborta sin escribir ante uno sin contexto. Tras ella se regeneran `INDEX.md` y los goldens.
3. **Trace.** El resolutor de alias del mapa se inyecta en `internal/cli/trace.go`; `ParseID` y `acceptableAlias` admiten un ID heredado cuando el mapa lo conoce.
4. **Comprobación.** La prueba de residuos garantiza el criterio de terminado; las variantes RF de las pruebas sobre el corpus real garantizan que ningún título quedó sin ID.
5. **Orden de commits (uno por tarea).** Corrección de `ParseDelta`; mapa y generador; resolución en trace; reescritura masiva y regeneraciones; prueba de residuos y variantes RF. `sdd-design` fija el orden exacto y si la reescritura masiva va en un commit propio, como pide el plan.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `axiom.yaml` | Modificado | Bloque `sdd:` con `id_pattern: RF-<DOM>-<NN>`; comentarios |
| `openspec/specs/**` (49 capacidades) | Modificado | Títulos, prosa, `Cubre:`, rangos H1 |
| `openspec/INDEX.md` | Regenerado | Con la lógica de `axiom archive sync` |
| Mapa de IDs heredados (ruta a fijar en design) | Nuevo | Mapa versionado por (capacidad, ID antiguo) |
| Lista de residuos declarados (ruta a fijar en design) | Nuevo | Exclusiones con categoría y motivo |
| `internal/reqtrace` | Modificado | Resolutor de alias del mapa; aceptación de IDs heredados |
| `internal/cli/trace.go` | Modificado | Cableado del resolutor |
| `internal/reqid` | Modificado/Nuevo | Generador del mapa y reescritura (o paquete nuevo, según design); variante RF de `legacy_equivalence_test.go` |
| `internal/openspecdoc` | Modificado | Corrección de `ParseDelta` con vallas; variante RF de `compose_corpus_test.go` |
| `internal/livespec` | Comprobado | Sin regresión tras la corrección de `ParseDelta` |
| `internal/branch/engine`, `internal/cli`, `internal/techcontext`, `internal/sddstatus`, `internal/kickoff` | Modificado | Literales de mensajes y pines de pruebas |
| `internal/assets/**`, `internal/components` + `testdata/golden/**` | Modificado/Regenerado | 5 assets Markdown y 10 goldens |
| `internal/**` (comentarios), `cmd/axiom/**`, `e2e/**` | Modificado | Citas en comentarios y pruebas |
| `odd/tasks/*.md` | Modificado | 29 documentos ODD |
| `docs/**` (vivo) | Modificado | Manual, configuración, ledger de absorción y plan |
| `axiom.example.yaml`, `.github/workflows/ci.yml`, `.goreleaser.yaml`, dashboard JS/CSS | Modificado | Comentarios o literales que citan IDs propios |

Dimensión medida en la exploración: 639 ficheros y 5235 coincidencias fuera de `archive/`.

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| IDs ambiguos (`REQ-1.1`…`REQ-4.2` en 7 capacidades) reescritos hacia la capacidad equivocada | Alta | Clave (capacidad, ID antiguo); resolución por contexto; fallo explícito sin escribir ante un ambiguo sin contexto |
| Un título que conserve `REQ-n.m` queda sin ID en silencio bajo el patrón RF | Media | Variantes RF de las pruebas sobre el corpus real y prueba de residuos |
| La delta de este cambio se compone al archivar, cuando las specs ya están migradas: sus títulos `MODIFIED` deben citar los IDs vigentes entonces (p. ej. REQ-48.2 → `RF-TRC-02`, REQ-48.4 → `RF-TRC-04`), y bajo el patrón por defecto previo a la migración esos títulos no se reconocen | Alta | `sdd-spec`/`sdd-design` eligen: escribir la delta ya con los IDs nuevos deterministas (y un provisional para el ADDED, REQ-47.1) o que la herramienta reescriba también la delta activa del cambio. Debe validarse antes de archivar |
| `axiom trace` por un ID nuevo arrastra incrementos archivados de otra capacidad que usaban el mismo ID antiguo ambiguo | Media | Coincidencia de antiguos acotada por capacidad de la delta archivada |
| Código de producción de la herramienta sin llamadas rompe el trinquete de código muerto | Media | Ejecutarla desde `go test` o como código alcanzable; lo decide `sdd-design` |
| Imposibilidad de compilar binarios (Cortex) y `axiom` del `PATH` desactualizado | Alta | Todo en proceso: mapa, reescritura, `INDEX.md` y la comprobación de `axiom trace REQ-24.3` |
| Lista de residuos demasiado amplia que oculte citas propias | Media | Categorías cerradas con motivo por entrada; los pines de mensajes de producción nunca son residuo |
| Conflictos con trabajo en vuelo que añada o cite `REQ-n.m` (ramas ODD o PR abiertas) | Alta | Ventana sin otros cambios SDD abiertos; herramienta idempotente reaplicable tras rebasar |
| Reescribir documentos ODD cerrados altera la fuente que lee `axiom trace` (REQ-48.7) | Baja | Los IDs nuevos quedan coherentes con las specs; la traza encuentra los antiguos por el mapa |
| Rangos de títulos H1 («REQ-42.1 a REQ-42.9») sin equivalente directo | Baja | `sdd-design` decide la forma (rango RF o supresión) |
| Volumen (639 ficheros) dificulta la revisión de la PR | Alta | Reescritura mecánica en commits propios y separados de la lógica; la revisión se centra en el mapa, la herramienta y la lista de residuos |

## Plan de reversión

Antes de fusionar: descartar la rama `inc/migrar-ids-rf` y su worktree; `main` no cambia. Después de fusionar: revertir el commit de merge de la PR (se recomienda merge sin squash). La reversión devuelve las specs a `REQ-n.m`, retira la declaración `sdd.id_pattern` de `axiom.yaml` y el mapa; el patrón por defecto vuelve a aplicar sin pérdida, porque `openspec/changes/archive/` no se tocó. Si solo falla una parte, se puede revertir de forma aislada el commit de la reescritura masiva conservando la corrección de `ParseDelta` y la resolución de alias, que son retrocompatibles (sin mapa, el resolutor por defecto se comporta como hoy). Tras revertir, regenerar `INDEX.md` y los goldens.

## Dependencias

- C-03-a (patrón por proyecto), C-03-b (provisionales y asignación) y C-03-d (`axiom trace` con punto de extensión de alias), ya en `main`.
- Tabla de 49 códigos de dominio aprobada por el usuario (Engram `sdd/migrar-ids-rf/dominios`).
- Ventana sin otros cambios SDD abiertos en `openspec/changes/` (comprobado: solo existe este cambio).

## Criterios de éxito

- [ ] El mapa versionado cubre los 513 requisitos de las 49 capacidades, con clave (capacidad, ID antiguo), sin colisiones, y su regeneración es idéntica byte a byte.
- [ ] `axiom.yaml` declara `sdd.id_pattern: RF-<DOM>-<NN>` y todos los títulos `### Requirement:` de `openspec/specs/` tienen un ID válido bajo ese patrón (comprobado por la variante RF de las pruebas del corpus real).
- [ ] Ninguna cita a un requisito propio de Axiom en forma `REQ-n.m` fuera de `openspec/changes/archive/`, del mapa y de la lista de residuos declarados; lo garantiza una prueba Go.
- [ ] `axiom trace REQ-24.3` resuelve al ID nuevo, verificado en proceso; un ID antiguo ambiguo resuelve a todos sus IDs nuevos.
- [ ] `openspec/INDEX.md` regenerado sin diferencias inesperadas (solo cambios de ID y de los 48 títulos que reciben ID).
- [ ] Goldens regenerados con `-update` y estables en una segunda ejecución.
- [ ] `ParseDelta` ignora `### Requirement:` dentro de bloques vallados (prueba RED observada antes del arreglo; `./internal/livespec` en verde).
- [ ] Pruebas localizadas en verde en los paquetes tocados; la suite completa la ejecuta la CI.
- [ ] Una sola PR, un commit por tarea.
