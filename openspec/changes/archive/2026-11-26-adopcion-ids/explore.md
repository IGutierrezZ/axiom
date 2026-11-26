# Exploración: `adopcion-ids` (C-03-c)

> Cambio SDD sobre `axiom archive adopt scan|check`. Rutas relativas al worktree
> `C:\repos\axiom-wt\adopcion-ids\axiom`. Solo lectura; nada se ha modificado fuera de este fichero.

## 1. Objetivo y fuente

D-03-7/D-03-8/A-03-1: la adopción conserva los IDs funcionales del cliente (`RF-001`…`RF-120`), declara su patrón en
`axiom.yaml` y un requisito nuevo integrado después continúa la serie (`RF-121`). Un origen sin IDs recibe IDs con el patrón
por defecto. `adopt check` valida contra el patrón declarado. Ficha: `docs/plan/backlog-2026-11/03-openspec-unica.md:131-155`.

**Aviso: los números de línea de la ficha están desfasados.** C-03-a ya movió `check.go` a `reqid` (no hay `REQ-n.m` literal
ni `legacy` regex); las líneas reales están en la sección 3.

## 2. Estado actual

### `adopt scan` (`internal/adoption/scan.go`, `internal/cli/archive_adopt.go`)
- `foreignIDPattern` (`scan.go:34`): `\b(?:RF|RNF|RN|REQ|US|HU|UC|CU|FR|NFR)[-_](?:[A-Z]{2,5}[-_])?\d{1,5}\b`. Prefijo cerrado, separador
  `-` o `_`, dominio opcional de 2-5 letras, 1-5 cifras.
- `foreignIDs` (`scan.go:200-216`) descarta los `REQ-n.m` (los seguidos de `.<dígito>`) y devuelve los IDs **ordenados
  lexicográficamente y sin repetir por unidad**. No guarda el orden de aparición ni las repeticiones entre unidades.
- `Units` (`scan.go:223-258`) asigna `Unit.ForeignIDs` por sección H1-H3 (o por el fichero entero sin encabezados).
- Salida: `Inventory{Schema, Sources[], ContextCandidates[], Warnings[]}` (`inventory.go:34-63`); esquema
  `axiom.openspec-adoption-inventory/v1` (`inventory.go:22`). Texto: `adoptScanText` (`archive_adopt.go:198-242`) imprime los IDs
  entre corchetes por unidad. JSON: `adoptScanReport` = Inventory + `written` (`archive_adopt.go:127-130`).
- **No propone patrón.** Solo propone `technical_context`, y el texto dice «axiom.yaml no se modifica» (`archive_adopt.go:221`).
  Ni `scan` ni `check` escriben jamás en `axiom.yaml`. `scan --change` solo escribe `adoption-inventory.yaml` (guarda A,
  `archive_adopt.go:149`); `check` no escribe nada.
- `scan` tampoco lee `axiom.yaml` (`locateAdopt`, `archive_adopt.go:113-123`, no exige ni `axiom.yaml` ni `openspec/`).
- Consumidores del inventario: `adoption.Summarize` (`inventory.go:82`) usado por el hub (`internal/hub/detector.go:158`,
  `internal/hub/init.go:336`) y el dashboard. `Summarize` no incluye `ForeignIDs`; un campo nuevo del inventario no los rompe.

### `adopt check` (`internal/adoption/adoptcheck/check.go`)
- `legacy = reqid.Default()` (`check.go:24-26`), con comentario «hasta que C-03-c lo lea de la declaración (REQ-30.31)».
- Formato (`check.go:151-155`): `legacy.InTitle(r.Name) == ""` → fallo `format` con `(REQ-n.m)`. **Rechaza provisionales**
  (`InTitle` solo acepta definitivos válidos).
- Unicidad (`check.go:165-198`): por ID en el cambio y contra `livespec.LoadRegistry(PlanningHome, {ExcludeChange})`.
- Serie única (`check.go:168-232`): `legacy.Series(id)`; si hay >1 serie, fallo `series` («usa una única serie `REQ-n.*` nueva»);
  y si la serie ya existe en `openspec/specs/`, otro fallo. **Con `RF-<DOM>-<NNN>` varias series son legítimas (una por
  dominio)**; la regla actual no tiene sentido con IDs heredados.
- `Origen:` (`check.go:265-331`, `adoption.ParseOrigins` en `discards.go:22-28`): `Origin{Path, Anchor, OriginalID, Line}`. **Nada
  compara `OriginalID` con el ID del título**; hoy solo se valida ruta y ancla. El «original coincide con el de la spec»
  de la ficha es nuevo.
- Reglas (`check.go:30-38`): `format, uniqueness, series, scenarios, origin, coverage, discard_reason, inventory_missing,
  inventory_stale`. Se acumulan todos los fallos.
- `loc.Declarations` ya existe en `workspace.Location` y `IDPattern()` (`internal/workspace/declarations.go:52-67`) lo expone; el
  resto de verbos lo usan vía `projectIDPattern` (`internal/cli/idpattern.go:12-23`). `check` es el único que no.

### Pruebas que fijan el comportamiento actual (a invertir o ampliar)
- `check_pattern_test.go:23` `TestCheck_ConUnPatronDeclaradoSigueAplicandoElDefecto` y `:38` `TestCheck_LosTextosDelDefectoNoCambian`:
  **fijan el defecto con patrón declarado; C-03-c las invierte** (hay que reescribir su semántica, no solo añadir).
- `check_test.go`: `TestCheck_RequisitoSinIdentificadorEnElTitulo :145`, `…IdentificadorDuplicadoEnElCambio :161`,
  `…IdentificadorQueYaEstaEnLasSpecs :168`, `…ColisionConOtroCambioActivo :177`, `…MezclaDeSeries :186`,
  `…SerieYaExistenteEnLasSpecs :193`, `…UnaSolaSerieNuevaNoFalla :202`, `…ElOrigenSeConservaTalCualEnLaDelta :333`.
- `scan_test.go`: `TestScan_UnidadesYIdentificadoresAjenos :121`, `…IgnoraIdentificadoresDeOpenSpec :171`,
  `…EsDeterministaYNoEscribe :236`, `TestSummarize :309`.
- `cli/archive_adopt_test.go`: `…ScanWithChangeWritesOnlyTheInventory :159`, `…CheckValidChangePasses :268`,
  `…CheckReportsEveryFailureAndExitsNonZero :291`, `…Help :380`; `cli/archive_idpattern_test.go:136`
  `TestArchiveAdoptHelpIsUnchangedByTheDeclaredPattern` (**fija que la ayuda NO cambia con el patrón; se invierte**).
- Skill: `internal/assets/assets_test.go:3544` `TestSDDSpecAdoptionReferenceDescribesTheWholeFlow`, `:3572`, `:3593`;
  `internal/components/sdd/inject_test.go:8148`. Hub/dashboard: `hub/detector_test.go:95,106`, `dashboard/*adoption_test.go`.

## 3. Puerto de IDs (C-03-a) — piezas reutilizables

- **Declaración:** `sdd.id_pattern` en `axiom.yaml` (`internal/workspace/sdd.go:22,57-64,80-93`; bloque `sdd:` con claves cerradas
  `iso` e `id_pattern`). **Un único escalar por proyecto**: no admite lista ni alternativas. Sin clave rige `REQ-<n>.<m>`.
- **Gramática** (`internal/reqid/template.go:11-125`): literal inicial obligatorio `[A-Z][A-Z0-9]*[-._]`; resto de literales con
  `A-Z 0-9 . _ -`; ≤64 caracteres. Marcadores: `<DOM>` (`[A-Z][A-Z0-9]*`, una vez), `<n>` (serie numérica, una vez), un único número:
  `<m>` (sin relleno) o `<NN>`…`<NNNNNN>` (ancho k).
- **Ancho** (`template.go:236-243`): `<NNN>` acepta `[0-9]{3}|[1-9][0-9]{3,}`: `RF-001` y `RF-1000` válidos; **`RF-1` no**. `<m>` acepta
  `[0-9]+`: `RF-1` y `RF-001` válidos a la vez (pero `Format` no rellena).
- **API** (`reqid.go`): `Parse`, `Default`, `Valid`, `InTitle` (último `(ID)` válido), `Scan` (válidos + mal formados),
  `TrailingCandidate`, `Series(id) (Series, bool)` (`reqid.go:143`), `Number`, `Format(Series, n)` (`reqid.go:195`, rellena a `numWidth`, no trunca).
  `Series.Key()/String()/Compare` en `series.go`.
- **`SeriesOf` no es API de producción**: solo existe en `internal/reqid/export_test.go:12` (reconstruye serie desde la clave).
  La función de producción es `Pattern.Series(id)`.
- **Asignación** (`allocate.go`): `NewAllocator(p)`, `Occupy(id)` (máximo por serie), `Next(Series)` = máx+1 con el relleno
  de la plantilla, `NewSeries(pv)` para etiquetas. Las series no usan estado en disco.
- **Provisionales** (`provisional.go`): `prov:<clave>:<ordinal>`; clave = componentes `<DOM>`/`<n>` en orden de plantilla; en un patrón
  sin `<DOM>` ni `<n>` (como `RF-<NNN>`) la clave es vacía: `prov::1` (`provisional.go:169-170,194-217`). Etiqueta de serie nueva solo con `<n>`.
  `ProvisionalInTitle` (`:90`), `ParseProvisional` (`:155`).
- **Integración** (`internal/livespec/ids.go:496-641` `assign`): ocupa primero los definitivos de los ADDED (`:525-531`) y luego asigna los
  provisionales con `alloc.Next(series)` (`:571-581`). `reg.alloc` se pobla desde `openspec/specs/` + otros cambios activos
  (`livespec/registry.go:58-68,126`). Traza provisional→definitivo en `reqid.IDMap` (`idmap.go`).
- **Casos pedidos** (por lectura del código; ninguno ejecutado):
  - `RF-001..RF-120` con `sdd.id_pattern: RF-<NNN>`: tras integrar (ocupa 001..120), un ADDED `(prov::1)` recibe `RF-121` (`Format` con ancho 3).
    Con `RF-<m>` también `RF-121`. Con 1..9 solo adoptados y `<NNN>`, el siguiente sería `RF-010`.
  - Con dominio: `RF-USU-001` → `RF-<DOM>-<NNN>`; la serie es el dominio; `prov:USU:1` → `RF-USU-<máx+1>`.
  - `Series` no distingue `RF-1` de `RF-001` con `<m>`: ambos son número 1 de la serie `RF-*`.
- `reqid` es hoja (solo stdlib, REQ-44.4; `imports_test.go`, `arch_test.go`): `adoption` puede importarlo (ya lo hace `adoptcheck`), `reqid` no puede importar `adoption`.
  Un inferidor de patrón desde IDs textuales puede vivir en `adoption`, apoyándose en `reqid.Parse/Valid`.

## 4. Requisitos vivos afectados

`openspec/specs/openspec-adoption/spec.md` (REQ-30.9 a REQ-30.14; fichero nuevo «spec completa», no delta):
- **REQ-30.9** (`:46`): scan; añade la propuesta de patrón junto a los candidatos `technical_context`. Escenario «Candidatos a technical_context» (`:80`) es modelo.
- **REQ-30.10** (`:87-104`): exige `(REQ-n.m)` en el título; pasa a «ID válido del patrón declarado (conservado) o provisional».
- **REQ-30.11** (`:108-155`): (a) formato y (b) unicidad citan `REQ-n.m`; añadir «ID original = ID del título cuando se conserva».
- **REQ-30.12** (`:159-184`): «serie `REQ-n.*` única y nueva» y «`RF-xxx` solo en `Origen:`» son exactamente lo que D-03-7 revierte. Escenarios `:165-184` se reescriben (incl. «El identificador funcional no es la identidad»).
- **REQ-30.13** (`:188-202`) y **REQ-30.14** (`:206`): sin cambio de fondo (archivado conserva `Origen:`); revisar el texto.
- Sección «No cubre» (`:38`): «Identificadores funcionales (`RF-xxx`) como identidad de requisito» → contradice C-03-c; se retira.
- También citan `adopt` (revisar, probablemente sin cambio): `sdd-archive-lifecycle/spec.md:672`, `multi-project-hub/spec.md:114,316,339`,
  `change-branch-lifecycle/spec.md:1128-1129`, `requirement-identifiers` (REQ-30.31, REQ-44.x: cita que `adopt check` aplica el defecto «hasta C-03-c»; buscar `REQ-30.31` y ajustarlo).

## 5. Guía, manual y goldens

- Skill: `internal/assets/skills/sdd-spec/references/openspec-adoption.md` (flujo; línea 7 «one new series», `:15` título `(REQ-n.m)` «serie nueva»,
  `:17` Origen, `:27` «never migrate REQ-x.y»; pasos 1 y 3-5 cambian) y menciones en `sdd-spec/SKILL.md:81`, `sdd-propose/SKILL.md:196`,
  `sdd-archive/SKILL.md:135`, `_shared/sdd-iso-convention.md:18`. Tests de contenido: `assets_test.go:3544-3606`.
- Goldens: el asset se instala por runtime (`components/sdd/inject_test.go:8148`). Al tocar `references/openspec-adoption.md` hay que regenerar
  los goldens de `internal/components` (`TestGolden`, en `go test ./internal/components/... -run TestGolden`; la regeneración se hace con la bandera de actualización del repo, comprobar su nombre al implementar).
  **[COMPARTIDO]** con otros cambios que toquen assets.
- Manual: `docs/manual-de-inicio.md:587-605` (pasos 1-3 y párrafo «El identificador ajeno nunca es la identidad»).
- Ayuda de `adopt` (`archive_adopt.go:47-65`): hoy cita `(REQ-n.m)` por defecto «hasta C-03-c» (`:55-56`).

## 6. IDs de cliente no uniformes (punto 7)

El patrón declarable es **uno y cerrado** (un escalar con un literal inicial); `reqid` no tiene alternativas ni lista. Casos:

| Caso | `RF-<NNN>` | `RF-<m>` | Observación |
|---|---|---|---|
| `RF-1` y `RF-001` | `RF-1` inválido | ambos válidos, mismo número | `Format` sin relleno; `RF-1` y `RF-001` colisionan como número pero no como cadena |
| `RF` y `RNF` | solo uno | solo uno | otro literal inicial = otro patrón |
| con y sin dominio (`RF-001`, `RF-USU-002`) | no | no | distinto número de marcadores |
| `RF_001` vs `RF-001` | no | no | separadores distintos |

Opciones (sin decidir):
- **(a) Un patrón por prefijo/tipo.** `sdd.id_pattern` pasaría a lista/mapa. Pros: respeta el origen tal cual; nada de normalizar. Contras: **cambia el
  contrato de C-03-a** (`workspace.SDDConfig.IDPattern *reqid.Pattern`, `reqid` ofrece un solo patrón; `livespec`, `isoevidence`, `sddstatus`,
  `livingdoc`, `archive-*` lo asumen único: 9+ llamadas a `IDPattern()`), y un requisito nuevo debe decir de qué tipo es. Tamaño > M.
- **(b) Un patrón con alternativas** (`RF-<NNN>|RNF-<NNN>`). Contras: nueva gramática en `reqid`, ambigüedad de `Series`/`Format`
  (¿qué alternativa recibe el siguiente número?), y toca la hoja `reqid`. Mismo problema de contrato que (a) y más superficie.
- **(c) Rechazar y pedir normalizar.** `scan` detecta la mezcla y la nombra (`mixed_id_schemes`); `adopt check` falla si un ID no cumple el patrón declarado;
  el humano decide en el origen. Pros: cero cambio de contrato, coherente con «la normalización nunca la hace Axiom, se pide al humano»; encaja en M. Contras: fricción en proyectos reales con `RF` y `RNF`; el humano debe elegir si
  unificar `RF`/`RNF` bajo un único prefijo o quedarse con una familia.
- **Variante (c′), recomendada:** `<m>` absorbe la mezcla de anchos (`RF-1`/`RF-001`) sin rechazar, pero `check` detecta duplicados por **(serie, número)**, no por cadena; mezcla de prefijos, de dominio o de separador → propuesta con aviso y
  conflicto en la salida de `scan`, y fallo `format` en `check`.

## 7. Cómo proponer el patrón (punto 8)

- `adopt` no escribe en `axiom.yaml` hoy y no debería empezar (política de la skill: «Never write `technical_context:` into `axiom.yaml` on your own»; `scan` solo lee).
- Propuesta: `scan` calcula, de las `ForeignIDs` de todas las unidades, un `proposed_id_pattern` con `template`, `evidence` (recuento por prefijo/ancho/dominio),
  `conflicts[]` y `confidence`; lo imprime junto a los candidatos de contexto (texto) y lo añade al JSON del inventario como campo aditivo (`omitempty`, esquema `v1`
  compatible; no entra en `Summarize`). Heurística: prefijo y separador más frecuentes → literal inicial; dominio presente en **todos** los IDs → `<DOM>`;
  todas las cifras con el mismo ancho y ceros iniciales → `<NNN>`; anchos distintos → `<m>`; validar la plantilla con `reqid.Parse` y que `Valid` acepte todos los IDs; si no → conflicto.
  El campo no debe ser parte del hash de obsolescencia: `inventory_stale` solo compara el SHA-256 de los orígenes (`check.go:368-385`).
- **Circularidad a resolver en el design:** `archive-integrate`/`livespec` leen `sdd.id_pattern` de `axiom.yaml` en la rama del cambio. La persona (o el agente con su permiso) debe declararlo **antes** de `adopt check`
  y de archivar; `axiom.yaml` está fuera de la carpeta del cambio (autoridad de edición/`allowedEditRoots`). Opciones: (i) la skill manda al agente a pedir la declaración y editarla en la misma rama; (ii) `check` falla con una regla nueva
  `pattern_undeclared` (hay IDs conservados que no encajan en el patrón vigente) que cita la línea exacta `sdd:\n  id_pattern: "<plantilla>"` a copiar. Se recomienda (ii) + (i).
- Proyecto sin IDs de cliente: la propuesta es vacía y rige el defecto; los requisitos nuevos usan provisionales `prov:<etiqueta>:<k>` de una serie nueva (C-03-b) en vez de elegir `n` a mano.

## 8. Puntos de cambio (resumen)

| Pieza | Ruta:línea | Cambio |
|---|---|---|
| Inferencia de patrón | `internal/adoption/scan.go` (nuevo fichero p. ej. `pattern.go`) | `ProposePattern(inv) PatternProposal` usando `ForeignIDs`; determinista |
| Orden de ocurrencia / IDs por unidad | `scan.go:200-216` | hoy ordenados y deduplicados por unidad: basta para inferir; no cambiar |
| Inventario | `inventory.go:34-39` | campo aditivo `ProposedIDPattern` (`omitempty`), esquema `v1` |
| Texto de scan | `archive_adopt.go:198-242` | bloque «Patrón de identificadores propuesto» y `conflicts` |
| Ayuda | `archive_adopt.go:54-58` | sin «hasta C-03-c»; cita el patrón declarado |
| check: patrón | `check.go:24-26` | `legacy` → `loc.Declarations.IDPattern()`; error de patrón inaccesible como en `idpattern.go:50` |
| check: formato | `check.go:151-155` | aceptar definitivo del patrón **o** provisional válido (`ProvisionalInTitle`+`ParseProvisional`) |
| check: unicidad | `check.go:165-198` | por (serie, número) y «nunca reutilizados» (decidir fuente: specs + activos; histórico/archivado en el design) |
| check: series | `check.go:168-232` | quitar «serie única nueva» para patrones heredados; mantener la nueva-serie solo para el defecto con IDs numéricos definitivos (o retirarla en favor de provisionales) |
| check: `Origen:` | `check.go:265-331` | si el ID se conserva, `OriginalID == ID` del título; nueva regla `origin_id` |
| Regla nueva | `check.go:30-38` | p. ej. `pattern_undeclared`; documentar en la skill |
| Skill, manual, spec | ver secciones 4-5 | reescritura + goldens |

## 9. Riesgos

1. **Contrato del patrón único (C-03-a)**: (a) y (b) de la sección 6 lo rompen; elegir (c)/(c′) evita ampliar la tarea más allá de M.
2. **Circularidad `axiom.yaml`**: declarar el patrón fuera de la carpeta del cambio; riesgo de que `check` y `archive-integrate` midan con patrones distintos si se olvida (mitigar con `pattern_undeclared`).
3. **Sentido inverso de pruebas existentes** (`check_pattern_test.go`, `archive_idpattern_test.go:136`, `assets_test.go:3544+`): riesgo de dejar pruebas verdes que cubran el comportamiento viejo; hay que reescribirlas.
4. **IDs provisionales y asignación en la rama** (riesgo de C-03-b, `ids.go`): dos adopciones concurrentes no son realistas, pero la serie `RF-<NNN>` compartida con un cambio activo que añada requisitos sí puede colisionar; la reintegración ya existe.
5. **Orden determinista de los números nuevos**: los provisionales reciben `RF-121…` por ordinal; si el humano pega orígenes sin ID en orden distinto, la numeración cambia; documentar.
6. **`foreignIDPattern` cerrado** (`scan.go:34`): prefijos fuera de la lista (`REQ` sí, `SYS`, `FUN`, `R`) no se detectan; la propuesta puede quedar vacía injustamente. Decidir si se amplía la lista o se acepta lo conocido. Colisión con `REQ-n.m` ajeno: si el cliente usa `REQ-12` (sin punto) el inferidor propondría `REQ-<m>`, que difiere del defecto de Axiom (`REQ-<n>.<m>`) pero no es conflictivo.
7. **Goldens y skill compartidos** (`[COMPARTIDO]`): coordinar con otros cambios que modifican `internal/assets/**`.
8. **ISO**: `sdd-iso-convention.md:18` exige un ID válido del patrón por requisito; un proyecto ISO adoptado debe declarar el patrón antes del primer requisito.
9. Fuera de alcance: validar normas ISO publicadas (D-03-7).

## 10. Preguntas abiertas (para propose/design)

1. ¿(c) o (c′) para mezclas de esquemas? (recomendado: (c′); (a)/(b) quedan como evolución futura del puerto, no de este cambio.)
2. ¿`sdd.id_pattern` lo escribe `adopt` con una opción explícita (`--write-pattern`) o solo se informa? Recomendado: solo informa (coherente con `technical_context`), más la regla `pattern_undeclared`.
3. ¿Se mantiene la regla `series` para el defecto o se sustituye por provisionales con etiqueta? Recomendado: provisionales; retirar la elección manual de `n`.
4. ¿«Nunca se reutilizan» incluye IDs retirados/archivados (REMOVED, `superseded`)? Hoy el registro cubre specs vivas + cambios activos; fuera, no.
5. ¿Se amplía `foreignIDPattern`? (propuesta: no en este cambio.)

## 11. Recomendación

Carril SDD, tamaño M, un solo cambio:
1. `adopt scan` calcula y muestra un `proposed_id_pattern` (aditivo, `omitempty`, v1) con evidencia y conflictos; no escribe `axiom.yaml`.
2. `adopt check` lee `loc.Declarations.IDPattern()`; formato acepta definitivo del patrón o provisional; unicidad por (serie, número);
   `Origen:` con ID conservado exige igualdad; fuera la serie única; nueva regla `pattern_undeclared`.
3. Mezclas: variante (c′) (`<m>` para anchos mixtos; otras mezclas → conflicto y fallo `format`, el humano normaliza).
4. Sin IDs de origen: provisionales de C-03-b, sin elegir `n` a mano.
5. Delta de `openspec-adoption` (REQ-30.9 a 30.13 y «No cubre»), skill `openspec-adoption.md`, manual y goldens.
6. TDD: la propiedad central es el criterio de la ficha: adoptar `RF-001..RF-120`, declarar `RF-<NNN>` e integrar un nuevo requisito da `RF-121` (prueba de integración con `livespec`).
