# Propuesta: IDs provisionales, número asignado al integrar y declaración de impacto en `status` (C-03-b)

## Intención

Cumplir D-03-4 con la regla A-03-3 («la declaración es la delta»): todo incremento SDD u ODD debe dejar reflejado qué identificadores existentes toca y cuáles crea, aunque el número de los nuevos no se conozca hasta integrar.

Hoy no es posible:

- Un ADDED exige un identificador definitivo elegido a mano (`added_without_id`, `internal/livespec/plan.go:286-292`; REQ-30.4). Dos ramas concurrentes eligen el mismo número y alguien renumera a mano: `carril-sdd-versatil` pasó de REQ-28 a REQ-29 y su `archive-report.md` aún cita REQ-28.x, sin mapa que una las dos numeraciones.
- Un MODIFIED o REMOVED con un identificador que no existe solo falla con el mensaje genérico de `UnappliedDeltaError` («no canonical requirement named»). No hay un error propio para «ID inexistente».
- Nada proyecta la lista de IDs que un cambio modifica o crea. La sección 07 (C-07-e, P12) la necesita para enlazar el work item antes de integrar.
- La lógica de asignación (`specbaseline/ids.go`) se borró sin portar en `1166efca` (nota 3 de «Correcciones a la consolidación»). Hay que rehacerla sobre el puerto `internal/reqid` de C-03-a.

Por qué ahora: es la ola 7 del plan. C-03-a ya está fusionado y archivado (`2026-11-24-puerto-reqid`). Bloquea a C-03-c, C-03-d y C-03-e, y C-07-e depende de él en duro desde P12.

## Alcance

### Dentro del alcance

1. **Forma provisional del identificador** en el puerto `internal/reqid`, con estos invariantes (el formato exacto lo fija el design):
   - Inconfundible con un identificador válido bajo cualquier patrón declarable (`sdd.id_pattern`): nunca es `Valid` y nunca la devuelve `InTitle`/`Scan` como ID válido. La exploración recomienda apoyarse en el invariante de que ningún ID válido contiene minúsculas (`reqid/synthetic.go:9-12`; p. ej. `prov:<serie>:<k>`).
   - Lleva fijada la serie o el dominio cuando se conoce (`RF-USU-…`, o la serie `n` de `REQ-<n>.<m>`). Puede expresar «serie nueva» con una etiqueta local del cambio, de modo que varios ADDED compartan una serie que aún no tiene número. Con `REQ-<n>.<m>` el `n` de una capacidad nueva tampoco se conoce hasta integrar.
   - Único dentro del cambio o del documento ODD, no entre cambios.
   - Nunca llega a `openspec/specs/`.
2. **Asignación del siguiente número libre al integrar**, para el patrón efectivo. Incluye abrir una serie vacía, que hoy no se puede formar (`livespec/registry.go:140-144`). La asignación es determinista y se hace en un plan único para el cambio entero, todas las capacidades a la vez, antes de escribir nada.
3. **Reescritura atómica** del provisional al definitivo, sustituyendo el token completo (idea de `IDMap.Substitute`), en:
   - la spec viva;
   - las líneas `Cubre:` de los ficheros de tareas;
   - el documento ODD (su valla `openspec` y sus líneas `Cubre:`);
   - la delta del cambio antes de archivarla.

   Todo o nada, como `UnappliedDeltaError` y `livespec.Apply` hoy.
4. **Reglas de la delta con IDs mezclados:**
   - ADDED con provisional. Un ADDED con ID definitivo se sigue aceptando con la comprobación de colisión actual (`CollisionError`).
   - MODIFIED, REMOVED y el nombre antiguo de RENAMED con un ID que existe en `openspec/specs/`. Si el ID no existe, error tipado nuevo («identificador inexistente»).
   - Un provisional en MODIFIED, REMOVED o RENAMED es un error tipado.
   - Un provisional citado en `Cubre:` o en el texto, sin un ADDED que lo declare, es un error tipado.
   - Un provisional duplicado en el cambio es un error tipado.
5. **Idempotencia y reintegración.** Si dos ramas asignan el mismo número, la segunda en fusionarse vuelve a ejecutar la integración tras el rebase:
   - La integración reconoce lo que ya integró aunque el título haya cambiado: usa el mapa guardado y la igualdad del cuerpo módulo la sustitución.
   - Reasigna los definitivos que otro cambio ocupó con un cuerpo distinto.
   - Detecta IDs duplicados entre capacidades de la spec viva, para que la colisión no pase en silencio cuando git no produce conflicto (por ejemplo, la misma serie nueva en dos capacidades distintas).
6. **Traza legible tras archivar.** El mapa provisional→definitivo, con capacidad, queda en el cambio archivado (SDD) o en el documento ODD, y en el informe `axiom.odd-integration/v1`.
7. **Archivado SDD dirigido por Go.** Un paso nuevo (verbo o modo; el nombre lo fija el design) cubre todo el cambio:
   - planifica las asignaciones;
   - compone todas las capacidades, también las nuevas, que hoy la skill copia con `cp` (`sdd-archive/SKILL.md:197-237`) y dejarían provisionales en la spec viva;
   - reescribe tareas y deltas;
   - escribe la traza.

   Todo ello antes de la instantánea y del `mv` de la skill. `archive-compose` por dominio conserva la salida byte a byte de toda composición aceptada hasta hoy y rechaza con error tipado un ADDED provisional, remitiendo al paso nuevo. `axiom sdd archive-check` (archivado interrumpido) reconoce la asignación ya hecha.
8. **ODD:** `axiom archive integrate --odd <feature>`:
   - asigna y reescribe el documento ODD dentro del mismo conjunto atómico de `Apply`. Esto enmienda el «no modifica el documento ODD» de REQ-30.2;
   - añade una opción de solo lectura (`--check`, con salida JSON en el informe) que proyecta `requirements` y el mapa previsto sin escribir nada. Es el equivalente mínimo del `status` para ODD; no se crea un status ODD completo.
9. **`axiom sdd status --json`:**
   - proyecta un campo aditivo `omitempty` `requirements: {modifies: [...], creates: [...]}`, derivado de las deltas del cambio (ADDED → `creates`; MODIFIED, REMOVED y RENAMED → `modifies`);
   - se documenta en `internal/assets/skills/_shared/sdd-status-contract.md`;
   - incluye pruebas de regresión de ausencia del campo cuando no hay deltas.
10. **ISO** (`internal/isoevidence`):
    - REQ-29.3 admite el provisional como forma válida del ADDED;
    - `Cubre:` acepta los provisionales de la propia delta (hoy darían `CodeCubreUnknown`) y se reescribe al archivar.
11. **Guías y goldens:**
    - `sdd-spec` y `sdd-archive`;
    - `_shared/sdd-iso-convention.md` y `sdd-status-contract.md`;
    - el paso 7 de ODD en `internal/components/agentguidance/routing.go:77`;
    - los goldens que fijan ese texto (13 identificados en la exploración; el resto se regenera si cambia texto compartido).

### Fuera del alcance

- Envío de IDs a terceros (sección 07) y `external_ref`.
- Adopción con IDs del cliente (C-03-c). Normalizar IDs de cliente no uniformes es tarea del humano.
- `axiom trace <ID>` (C-03-d), que leerá la traza que deja este cambio.
- Migración de las specs de Axiom a IDs funcionales (C-03-e).
- Renumerar requisitos existentes o reescribir trazas antiguas (por ejemplo, el `archive-report.md` de `carril-sdd-versatil`).
- Limpiar el glosario residual de `spec-baseline-integration` (términos «ID provisional» de INC-24 sin requisitos vivos).
- Un status ODD completo.
- Exigir el provisional: el ADDED con definitivo sigue siendo válido.

## Capacidades

### Nuevas capacidades

Ninguna. Los requisitos nuevos se añaden a capacidades existentes.

### Capacidades modificadas

| Capacidad | Cambio |
|---|---|
| `requirement-identifiers` | ADDED: forma provisional y sus invariantes; asignación del siguiente libre, incluida la serie nueva; plan único y sustitución por token completo; detección de duplicados entre capacidades. Posible MODIFIED de REQ-44.3 («solo parte de una serie existente»). |
| `odd-living-spec-integration` | MODIFIED REQ-30.2 (reescribe el documento ODD de forma atómica; traza). MODIFIED REQ-30.3 (idempotencia y reintegración con provisionales). MODIFIED REQ-30.4 (ADDED con provisional o definitivo; ID inexistente y provisional fuera de ADDED como errores tipados). ADDED: `--check` de solo lectura con `requirements`. |
| `sdd-archive-lifecycle` | MODIFIED REQ-21.17 (asignación, reescritura y traza antes de transferir). MODIFIED REQ-30.22 (error tipado de ID inexistente; rechazo del provisional en `archive-compose`). ADDED: paso Go de archivado de todo el cambio, capacidades nuevas incluidas. |
| `sdd-iso-evidence` | MODIFIED REQ-29.3 (ADDED provisional válido) y REQ-29.4 (`Cubre:` con provisionales de la delta y su reescritura). |
| `axiom-sdd-cli-integration` | ADDED: proyección `requirements.modifies`/`requirements.creates` en `axiom sdd status --json`. |
| `organic-agent-trigger-rules` | MODIFIED REQ-24.58, regla 3 (Spec viva): el ADDED lleva un provisional y `axiom archive integrate --odd` asigna el número y reescribe el documento. |

**IDs de los requisitos de este cambio:** la delta usa IDs definitivos con la convención vigente, en la serie nueva **REQ-47**. Está confirmada como libre: la última usada en `openspec/` es REQ-46, y no hay ninguna coincidencia de `REQ-4[7-9].x` en `openspec/`, `odd/` ni `docs/`. No se usa el mecanismo provisional porque no estará instalado al archivar este cambio.

## Enfoque

1. **Puerto (`internal/reqid`, hoja):** tipo `Provisional` (serie conocida o etiqueta de serie nueva, más un ordinal), con parseo y formato. Además, construir una serie desde texto (`Series` es opaca hoy) y calcular el siguiente número, que pasa de `livespec.Registry.NextFree` al puerto. Se conserva la prueba de arquitectura de hoja (`arch_test.go`).
2. **Plan de asignación** (en `openspecdoc` o `livespec`; la ubicación la fija el design):
   - Entrada: todas las deltas del cambio y la spec viva.
   - Salida: un `IDMap` determinista (orden de delta, relleno del patrón), validado antes de escribir. Valida duplicados, provisional igual a un definitivo, provisional fuera de ADDED e ID inexistente.
   - Referencia histórica: `ids.go` en `1166efca^`, descartando su rango alto `RF-9001` porque es confundible.
3. **Reescritura:** operación nueva de sustitución por token completo en títulos y cuerpos (`openspecdoc`), en `Cubre:` (función nueva en `isoevidence`, que conserva la posición del token) y en el documento ODD (`livespec/odddoc.go`). Las escrituras se agrupan en un único conjunto atómico: temporales, `rename` y restauración, como `livespec.Apply`.
4. **Asignación en la rama:**
   - ODD integra en la PR de la feature.
   - SDD integra en su rama de archivo, tras fusionar el código.
   - La colisión entre ramas se resuelve con reintegración idempotente, gracias al mapa guardado y a la detección de duplicados.
   - Se descarta asignar tras la fusión en `main`: exigiría un paso posterior que hoy no existe y dejaría provisionales en la spec viva.
5. **SDD:** el paso Go de archivado sustituye en la skill `sdd-archive` el bucle por dominio `archive-compose` + `mv` y la copia `cp` de las capacidades nuevas. La skill conserva la instantánea, el `diff -r` y el `mv` de la carpeta, que ahora ocurren después de la reescritura, para que `tasks.md` archivado conserve sus bytes respecto a la instantánea (`SKILL.md:354`).
6. **Status:** decorador aditivo con el patrón de `decorateISO` y `decorateWorkspaceDeclarations`. Su contrato va en `sdd-status-contract.md`. ODD: `--check` sobre el `Report` existente.

## Áreas afectadas

| Área | Impacto | Descripción |
|---|---|---|
| `internal/reqid` | Modificado | Provisional, serie desde texto, siguiente número |
| `internal/openspecdoc` (`compose.go`, `resolve.go`, `render.go`, `parse.go`) | Modificado | Error tipado de ID inexistente, rechazo del provisional, sustitución de ID, plan |
| `internal/livespec` (`plan.go`, `registry.go`, `apply.go`, `odddoc.go`, `report.go`) | Modificado | Aceptar y asignar el provisional, documento ODD en el conjunto atómico, mapa en el informe, idempotencia, duplicados |
| `internal/isoevidence/cubre.go`, `reqid.go` | Modificado | `Cubre:` con provisionales y reescritura; REQ-29.3 |
| `internal/sddstatus` (`status.go`, `openspec_archive_compose.go`, decorador nuevo) | Modificado | Proyección `requirements`; composición |
| `internal/cli` (`archive_integrate.go`, `sdd_archive_compose.go`, `sdd_archive_check.go`, verbo nuevo de archivado) | Modificado/Nuevo | `--check`, archivado del cambio entero, rechazo del provisional en `archive-compose` |
| `internal/assets/skills/sdd-spec/SKILL.md`, `sdd-archive/SKILL.md`, `_shared/sdd-iso-convention.md`, `_shared/sdd-status-contract.md` | Modificado | Guía del provisional, nuevo paso de archivado, contrato de `requirements` |
| `internal/components/agentguidance/routing.go` (paso 7) | Modificado | Texto del paso 7 de ODD |
| `testdata/golden/*` | Modificado | Regeneración (13 que fijan el paso 7; más si cambia texto compartido) |

## Riesgos

| Riesgo | Prob. | Mitigación |
|---|---|---|
| Digest de puertas: reescribir `tasks.md` y la delta al archivar cambia el digest que juzgan `role-apply`/`tasks`. Un archivado interrumpido podría reevaluar puertas como `pending` y bloquear la reejecución. | Media | El design verifica si el estado reevalúa puertas antes del archivado. Si lo hace, la reejecución reconoce la sustitución por el mapa guardado, o la reescritura ocurre después de la última lectura del precondicionante. Se cubre con una prueba de archivado interrumpido. |
| Duplicado silencioso entre ramas sin conflicto git (la misma serie nueva en dos capacidades distintas) | Media | Detección de duplicados entre capacidades en la integración y en `--check`. La guía ordena reintegrar tras cada rebase. |
| Contrato «no modifica el documento ODD» (REQ-30.2) y prueba «el documento ODD no se modifica» | Alta (seguro) | MODIFIED explícito de REQ-30.2. `--check` conserva la garantía de solo lectura. |
| Bytes de `archive-compose` (REQ-30.23) | Baja | Solo cambian los rechazos (error tipado nuevo). El corpus `compose_corpus_test.go` sigue byte a byte. |
| Goldens (13 o más) y ficheros compartidos (`routing.go`, skills `sdd-archive`/`sdd-spec`, `sddstatus`) con otros cambios del plan | Media | Ola 7 sin solapes duros (README §6). Rebase y regeneración de goldens al final. |
| Consumidores que leen deltas activas (indexador `livingdoc`, dashboard, colisión con cambios activos de REQ-30.4) ante un provisional | Media | El design inventaría los lectores. El provisional no es `Valid`, así que no colisiona ni se indexa como ID. Se añaden pruebas de no regresión. |
| Ramas abiertas con ADDED definitivo | Baja | Siguen admitidos con la comprobación de colisión actual. |
| Sin `git` en la exploración: `ids.go` se reconstruyó desde el design archivado de inc-24 | Baja | El design o el apply consultan `git show 1166efca^:internal/specbaseline/ids.go` como referencia. |

## Plan de reversión

- Revertir la PR de código. Los provisionales nunca llegan a `openspec/specs/`, así que la spec viva queda con IDs definitivos normales y válidos para el código anterior. Las trazas archivadas son texto inerte.
- Efecto residual: una rama abierta con ADDED provisionales fallaría con `added_without_id` (ODD) o con el bloqueo de REQ-29.3 (ISO). Se corrige escribiendo a mano el definitivo, como hoy.
- La PR del archive de este cambio va aparte. Si se revierte el código antes de archivar, el archive no se fusiona, y la delta (con IDs REQ-47 definitivos) se reescribe o se descarta.

## Dependencias

- C-03-a fusionado y archivado (`openspec/changes/archive/2026-11-24-puerto-reqid/`).
- Referencia histórica: `git show 1166efca^:internal/specbaseline/ids.go` e `ids_test.go`.
- Sin dependencias externas.

## Criterios de éxito

- [ ] Dos ramas concurrentes con un ADDED provisional en la misma capacidad se integran una tras otra sin renumerar a mano y reciben números distintos (ODD y SDD).
- [ ] Un MODIFIED con un ID inexistente se rechaza con un error tipado (también REMOVED y el nombre antiguo de RENAMED), sin escribir nada.
- [ ] `axiom sdd status --json` muestra `requirements.modifies` y `requirements.creates`. El campo se omite cuando el cambio no tiene deltas.
- [ ] La traza provisional→definitivo se puede leer después de archivar (cambio SDD archivado y documento ODD integrado).
- [ ] Ningún provisional llega a `openspec/specs/`. Una capacidad nueva con ADDED provisionales se archiva con IDs definitivos, incluida la apertura de una serie nueva con `REQ-<n>.<m>` y con `RF-<DOM>-<NN>`.
- [ ] Un fallo en cualquier punto de la reescritura no deja cambios a medias en la spec viva, en `Cubre:`, en el documento ODD ni en las deltas.
- [ ] `axiom archive integrate --odd <feature> --check` no escribe nada y proyecta `requirements` y el mapa previsto.
- [ ] Toda composición de `archive-compose` aceptada antes del cambio produce la misma salida byte a byte.
- [ ] `go test ./...` en verde, salvo los fallos ambientales ya conocidos en la base, y goldens regenerados.

## Estimación de tamaño

**L.** Desglose de la exploración: `reqid` S, `openspecdoc` M, `livespec` M-L, `isoevidence` S, `sddstatus` y contrato M, `cli` M, skills/guías/goldens M.

Entrega única (`single-pr`): una PR de código con un commit por tarea y, después, la PR del archive aparte (REQ-25.39).
