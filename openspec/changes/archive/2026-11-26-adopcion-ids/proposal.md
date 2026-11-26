# Propuesta: la adopción conserva los IDs del cliente y genera la spec viva con IDs funcionales (C-03-c)

## Intención

Cumplir D-03-7 y D-03-8 con la regla A-03-1: al migrar documentación previa a OpenSpec con `axiom archive adopt`, la spec viva conserva los identificadores funcionales del cliente (`RF-001`…`RF-120`), el proyecto declara en `axiom.yaml` el patrón heredado y los requisitos nuevos continúan la serie (`RF-121`). Un origen sin IDs recibe IDs con el patrón por defecto.

Hoy no es posible:

- `adopt check` sigue fijado al patrón por defecto: `legacy = reqid.Default()` (`internal/adoption/adoptcheck/check.go:24-26`), excepción temporal declarada en REQ-44.4 y REQ-30.31 «hasta C-03-c». Un título `(RF-001)` falla la regla `format`.
- La regla `series` (`check.go:168-232`) exige una única serie `REQ-n.*` nueva, y REQ-30.12 relega el ID del cliente a la línea `Origen:`: lo contrario de D-03-8.
- `adopt check` rechaza los provisionales de C-03-b (`InTitle` solo acepta definitivos, `check.go:151-155`), así que un origen sin IDs obliga a elegir `n` a mano.
- Nada compara el ID original de `Origen: <ruta>#<ancla> (<ID original>)` con el ID del título (`check.go:265-331`).
- `adopt scan` detecta IDs ajenos (`internal/adoption/scan.go:34`, `:200-216`) pero no propone ningún patrón; la persona tiene que deducirlo.

Por qué ahora: C-03-a (`2026-11-24-puerto-reqid`) y C-03-b (`2026-11-26-ids-provisionales`) están archivados. A-03-2 prohíbe adoptar proyectos reales hasta este cambio, porque hoy obligaría a una segunda migración.

## Alcance

### Dentro del alcance

1. **Propuesta de patrón en `adopt scan`.** `scan` infiere, de los `ForeignIDs` de todas las unidades, un patrón candidato con su evidencia (recuento por prefijo, separador, dominio y ancho) y sus conflictos. Se muestra en el texto junto a los candidatos a `technical_context` y se añade al inventario JSON como campo aditivo `proposed_id_pattern` (`omitempty`, esquema `axiom.openspec-adoption-inventory/v1` sin subir versión). La plantilla propuesta se valida con `reqid.Parse` y con `Valid` sobre todos los IDs. `scan` **no escribe** `axiom.yaml` ni ofrece `--write-pattern`.
2. **Mezclas de esquemas: variante (c′).** Se mantiene el patrón único `sdd.id_pattern` de C-03-a.
   - Los anchos mixtos del mismo esquema (`RF-1` y `RF-001`) se absorben con el marcador `<m>` en el patrón propuesto.
   - Cualquier otra mezcla (varios prefijos, con y sin dominio, separadores distintos, esquemas incompatibles) no se admite: `scan` la informa como conflicto y `check` la rechaza. Normalizarla es tarea del humano; Axiom nunca reescribe los IDs del cliente.
3. **`adopt check` valida contra el patrón declarado** (`Declarations.IDPattern()`, como el resto de verbos vía `internal/cli/idpattern.go`):
   - formato: el título lleva un ID definitivo válido del patrón o un provisional válido de C-03-b;
   - unicidad por (serie, número entero), no por cadena: `RF-1` y `RF-001` son el mismo ID y cuentan como duplicado, dentro del cambio y contra `openspec/specs/` y los cambios activos (registro de `livespec`);
   - cuando el ID se conserva, el `<ID original>` de `Origen:` coincide con el ID del título (regla nueva; nombre en el design);
   - regla nueva `pattern_undeclared`: el inventario trae IDs ajenos y el proyecto no declara patrón, o declara el defecto y esos IDs no lo cumplen. El motivo cita las líneas exactas de `sdd:` / `id_pattern:` que hay que copiar en `axiom.yaml`;
   - se retira la regla `series` («serie nueva única»).
4. **Orígenes sin ID:** cada requisito lleva un provisional de C-03-b y su número se asigna al archivar con `axiom sdd archive-integrate`. `check` acepta provisionales.
5. **Ayuda y texto de `adopt`** (`internal/cli/archive_adopt.go:47-65`, `:198-242`): fuera el «hasta C-03-c»; se cita el patrón declarado y se muestra el bloque del patrón propuesto con sus conflictos.
6. **Guía y documentación:**
   - referencia `internal/assets/skills/sdd-spec/references/openspec-adoption.md` (pasos de scan, declaración del patrón, títulos con ID conservado o provisional, `Origen:`, check);
   - menciones en otras skills que citen la serie `REQ-n.m` de adopción (`sdd-spec/SKILL.md`, `sdd-propose/SKILL.md`, `sdd-archive/SKILL.md`), solo si contradicen lo nuevo;
   - `docs/manual-de-inicio.md:587-605`;
   - goldens de `internal/components`.
7. **Pruebas que fijan el comportamiento viejo**, reescritas en su semántica (no solo ampliadas): `adoptcheck/check_pattern_test.go`, `check_test.go` (series), `internal/cli/archive_idpattern_test.go:136`, `internal/assets/assets_test.go:3544-3606`.
8. **Prueba de integración del criterio central:** adoptar `RF-001`…`RF-120`, declarar `RF-<NNN>`, archivar e integrar después un ADDED provisional, que recibe `RF-121`.

### Fuera del alcance

- Validar normas ISO publicadas (29148) o usar REQ-29.x como criterio de migración (D-03-7).
- Un patrón con alternativas o un patrón por prefijo (opciones (a)/(b) de la exploración): cambiarían el contrato de C-03-a.
- Normalización automática de IDs del cliente y que `adopt` escriba en `axiom.yaml`.
- Ampliar la lista cerrada de prefijos de `foreignIDPattern` (`scan.go:34`).
- `axiom trace <ID>` (C-03-d) y la migración de las specs propias de Axiom (C-03-e).
- Mostrar el patrón propuesto en el hub o en el dashboard (`adoption.Summarize` no cambia).
- «Nunca reutilizados» frente a IDs retirados o archivados (REMOVED, `superseded`): el ámbito es el registro actual (specs vivas y cambios activos).

## Capacidades

### Nuevas capacidades

Ninguna.

### Capacidades modificadas

| Capacidad | Cambio |
|---|---|
| `openspec-adoption` | MODIFIED REQ-30.9 (`scan` propone el patrón con evidencia y conflictos, sin escribir `axiom.yaml`); REQ-30.10 (título con ID definitivo del patrón declarado o provisional); REQ-30.11 (formato y unicidad según el patrón, unicidad por serie y número, coincidencia del ID original, `pattern_undeclared`); REQ-30.12 reescrito (conservación de los IDs del cliente, mezclas no admitidas, orígenes sin ID con provisionales; desaparecen «serie nueva única» y «`RF-xxx` solo en `Origen:`»); REQ-30.13 (archivado con `axiom sdd archive-integrate`, porque `archive-compose` rechaza los provisionales). Se retira de «No cubre» la línea «Identificadores funcionales como identidad». REQ-30.14 solo se revisa. |
| `requirement-identifiers` | MODIFIED REQ-44.4: se elimina la excepción temporal de `adoptcheck` y `archive adopt`, y los escenarios «`adoptcheck` sigue en el defecto con un patrón declarado» se invierten. |
| `sdd-orchestrator-assets` | MODIFIED REQ-30.31: la guía de adopción describe la propuesta y la declaración del patrón, los IDs conservados o provisionales y `pattern_undeclared`, en lugar de «un `(REQ-n.m)` de una serie nueva». |

**IDs de los requisitos de este cambio:** casi todo son MODIFIED de IDs existentes. Si hace falta algún ADDED, lleva un provisional de C-03-b, ya instalado; el número se asigna al archivar.

## Enfoque

1. **Inferencia en `internal/adoption`** (fichero nuevo, p. ej. `pattern.go`). Es una función determinista sobre el inventario y se apoya en `reqid.Parse`/`Valid`; `reqid` sigue siendo hoja.
   - Prefijo y separador → literal inicial.
   - `<DOM>` solo si todos los IDs llevan dominio.
   - `<NN…>` si todos tienen el mismo ancho con ceros iniciales; `<m>` si los anchos se mezclan.
   - Cualquier otra heterogeneidad → `conflicts[]` y ninguna plantilla cerrada.
   - El campo no entra en el hash de `inventory_stale` (`check.go:368-385`).
2. **`check` lee el patrón efectivo** de `loc.Declarations`. Un patrón inaccesible da el mismo error que en `idpattern.go`.
   - Formato: `InTitle` o `ProvisionalInTitle` + `ParseProvisional`.
   - Unicidad: clave (serie, número) con `Pattern.Series` y `Number` contra el registro de `livespec`.
   - `pattern_undeclared` se deriva de los `ForeignIDs` del inventario guardado, así que funciona también con inventarios escritos antes de este cambio.
3. **La declaración del patrón la hace la persona** (o el agente siguiendo la guía) en `axiom.yaml`, dentro de la rama del cambio de adopción, antes de `adopt check` y de archivar. `pattern_undeclared` cierra la circularidad: `check` y `archive-integrate` miden con el mismo patrón.
4. **Archivado:** el cambio de adopción se archiva con `axiom sdd archive-integrate` (C-03-b), que ocupa los definitivos conservados y asigna los provisionales con `alloc.Next`.
5. **TDD:** la propiedad central (RF-001…RF-120 → RF-121) va primero, como prueba de integración contra `livespec`; después, las reglas de `check` y la inferencia de `scan`.

## Áreas afectadas

| Área | Impacto | Descripción |
|---|---|---|
| `internal/adoption/scan.go`, `inventory.go`, nuevo `pattern.go` | Modificado/Nuevo | Inferencia del patrón propuesto; campo aditivo del inventario |
| `internal/adoption/discards.go` | Modificado | `Origin.OriginalID` para la comparación con el título |
| `internal/adoption/adoptcheck/check.go` | Modificado | Patrón declarado, provisionales, unicidad por (serie, número), ID original, `pattern_undeclared`; fuera `series` |
| `internal/cli/archive_adopt.go` | Modificado | Ayuda, texto del patrón propuesto, paso del patrón a `check` |
| `internal/assets/skills/sdd-spec/references/openspec-adoption.md` y menciones en otras skills | Modificado | Flujo de adopción con IDs conservados |
| `testdata/golden/*` (`internal/components`) | Modificado | Regeneración por el texto compartido |
| `docs/manual-de-inicio.md` | Modificado | Sección de adopción |
| Pruebas: `adoptcheck/*_test.go`, `adoption/scan_test.go`, `cli/archive_adopt_test.go`, `cli/archive_idpattern_test.go`, `assets/assets_test.go`, `components/sdd/inject_test.go` | Modificado | Se invierte la semántica vieja y se cubre la nueva |

## Riesgos

| Riesgo | Prob. | Mitigación |
|---|---|---|
| Pruebas verdes que siguen fijando el comportamiento viejo (`check_pattern_test.go`, `archive_idpattern_test.go:136`, `assets_test.go:3544+`) | Alta | Las tareas nombran esas pruebas para reescribirlas; el verify comprueba que no queda ningún escenario de la excepción. |
| Patrón declarado en `axiom.yaml` (fuera de la carpeta del cambio) olvidado o distinto del que usa el archivado | Media | `pattern_undeclared` con la línea exacta a copiar; la guía ordena declararlo antes de `check`. |
| Fricción con clientes que mezclan `RF` y `RNF` | Media | Conflicto nombrado en `scan` con su evidencia; la normalización es humana (decisión tomada). Un patrón con alternativas queda como evolución futura del puerto. |
| `foreignIDPattern` cerrado deja la propuesta vacía con prefijos desconocidos (`SYS-`, `FUN-`) | Media | Se documenta en la guía; ampliar la lista queda fuera. La persona puede declarar el patrón a mano y `check` lo valida igual. |
| Casos límite de la coincidencia del ID original: un requisito con varios `Origen:` con IDs, o un provisional cuyo origen trae ID | Media | La spec fija la regla. Recomendación: el ID del título debe coincidir con alguno de los `<ID original>`, y un provisional solo se admite si ningún `Origen:` trae ID. Queda abierto para la spec (ver riesgos del resultado). |
| Numeración de provisionales según el orden de la delta (si se reordena, cambian los números) | Baja | Comportamiento determinista de C-03-b; se documenta en la guía. |
| Goldens y skills compartidos con otros cambios de `internal/assets/**` | Media | Rebase y regeneración de goldens al final. |
| Proyecto ISO adoptado: `sdd-iso-convention.md:18` exige un ID válido del patrón | Baja | `pattern_undeclared` obliga a declarar el patrón antes del primer requisito. |

## Plan de reversión

- Revertir la PR de código. `adopt check` vuelve al defecto `REQ-<n>.<m>` y a la regla `series`, y `scan` deja de emitir `proposed_id_pattern` (campo `omitempty`, los lectores v1 no lo exigen).
- Efecto residual: un cambio de adopción en curso con IDs del cliente fallaría `format` con el código viejo. Como A-03-2 prohíbe adoptar antes de este cambio, no hay adopciones reales que romper.
- Las specs vivas ya archivadas con IDs del cliente siguen siendo válidas para el resto de verbos mientras `axiom.yaml` mantenga su `sdd.id_pattern` (contrato de C-03-a, que no se toca).
- La PR del archive va aparte (REQ-25.39). Si se revierte el código antes de archivar, la delta se descarta.

## Dependencias

- C-03-a (`openspec/changes/archive/2026-11-24-puerto-reqid/`) y C-03-b (`openspec/changes/archive/2026-11-26-ids-provisionales/`), los dos archivados: patrón declarado, provisionales, `axiom sdd archive-integrate` y el rechazo de provisionales en `archive-compose` (`internal/cli/sdd_archive_compose.go:181-188`).
- Sin dependencias externas.

## Criterios de éxito

- [ ] Origen con `RF-001`…`RF-120` y `sdd.id_pattern: RF-<NNN>` declarado: la spec viva archivada conserva esos IDs y un requisito nuevo integrado después con un provisional recibe `RF-121`.
- [ ] Origen sin IDs: los requisitos llevan provisionales y al archivar reciben IDs del patrón por defecto.
- [ ] `adopt check` falla con IDs duplicados (también `RF-1` frente a `RF-001` con `<m>`), fuera del patrón declarado, con ID original distinto del título y con `pattern_undeclared`. Reporta todos los fallos.
- [ ] `adopt scan` propone `RF-<NNN>` para `RF-001…RF-120`, `RF-<m>` para anchos mixtos y un conflicto nombrado para `RF`+`RNF`. No escribe nada sin `--change` y nunca escribe `axiom.yaml`. Es determinista.
- [ ] Ninguna prueba ni guía mantiene la excepción «hasta C-03-c»; REQ-44.4 y REQ-30.31 quedan reescritos.
- [ ] Pruebas de los paquetes tocados en verde y goldens regenerados; la batería completa la pasa la CI.

## Estimación de tamaño

**M.** Entrega única (`single-pr`): una PR de código con un commit por tarea y, después, la PR del archive aparte.
