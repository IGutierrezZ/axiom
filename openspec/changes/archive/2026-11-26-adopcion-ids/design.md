# Design: la adopción conserva los IDs del cliente y valida contra el patrón declarado (C-03-c)

## Enfoque técnico

El cambio cierra la excepción temporal de C-03-a en la adopción. `adopt scan` propone un patrón y `adopt check` mide con el patrón efectivo del proyecto, igual que el resto de verbos. No se toca el contrato de C-03-a (un único `sdd.id_pattern`) ni el motor de C-03-b (`axiom sdd archive-integrate`), que ya conserva los definitivos y asigna los provisionales. Para el criterio `RF-001…RF-120` → `RF-121` solo hacía falta que `adopt check` dejara pasar el cambio.

```
reqid (hoja, sin cambios)     Parse, Valid, InTitle, Series, Number, ProvisionalInTitle, ParseProvisional
   ▲
adoption                      IdentityIDs, ProposePattern (pattern.go, nuevo); Inventory.ProposedIDPattern
   ▲            ▲
livespec        │             Registry.LookupIdentity (clave serie + número)
   ▲            │
adoptcheck ─────┘             Check(loc, change, pattern): formato, unicidad, origin_id,
   ▲                          pattern_undeclared, foreign_id_conflict; fuera `series`
cli                           archive adopt scan (bloque del patrón, patrón declarado), check (patrón
                              efectivo con projectIDPattern), ayuda
```

Fuentes: `proposal.md`, las tres deltas de `specs/` (MODIFIED REQ-30.9 a REQ-30.13, REQ-44.4 y REQ-30.31), `explore.md` y `coverage-review.spec.md` (nueve huecos, que se resuelven en la sección final). Decisiones cerradas por la coordinación: variante (c′), `scan` no escribe `axiom.yaml`, `check` mide con `Declarations.IDPattern()`, se retira la serie única y el archivado se hace con `archive-integrate`.

## Comprobaciones sobre el código (base `b2201ce1`)

| Objeto | Estado comprobado |
|---|---|
| Patrón fijo en `adopt check` | `legacy = reqid.Default()` en `internal/adoption/adoptcheck/check.go:24-26`; formato con `legacy.InTitle` en `:151-155`, que rechaza los provisionales |
| Registro sin patrón | `livespec.LoadRegistry(..., RegistryOptions{ExcludeChange})` en `check.go:183`, sin `Pattern`, así que indexa con el patrón por defecto. Con `RF-<NNN>`, un `RF-001` de `openspec/specs/` no aparece en el registro |
| Unicidad por cadena | `all.Lookup(id)` en `check.go:195`; `Registry.byID map[string][]IDRef` en `internal/livespec/registry.go:55-59,130-135` |
| Regla de serie | `check.go:200-232` (`RuleSeries`, `check.go:32`) |
| `Origen:` | `adoption.ParseOrigins` (`internal/adoption/discards.go:111-155`) ya extrae `Origin.OriginalID`; nada lo compara con el título (`check.go:265-331`) |
| Identificadores ajenos | `foreignIDPattern` (`internal/adoption/scan.go:34`), `foreignIDs` ordenados y sin repetir por unidad (`scan.go:197-216`), por sección H1-H3 en `Units` (`scan.go:223-258`). Los orígenes binarios son una unidad sin IDs (`scan.go:133-137`) |
| Inventario | `Inventory` (`internal/adoption/inventory.go:34-39`); `ReadInventory` con yaml.v3 no estricto (`:112-126`), que ignora los campos desconocidos; `Summarize` no lee los IDs (`:82-94`) |
| Obsolescencia | `inventory_stale` solo compara el SHA-256 de cada origen (`check.go:368-385`); `axiom.yaml` no interviene |
| Patrón efectivo y error accionable | `Declarations.IDPattern()` (`internal/workspace/declarations.go:56-67`); `projectIDPattern`/`idPatternRefusal` (`internal/cli/idpattern.go:17-23,45-51`) |
| Serie y número | `Pattern.Series` (`internal/reqid/reqid.go:143-180`) quita el número y normaliza `<n>`, así que `RF-1` y `RF-001` con `RF-<m>` comparten `Series`. `Pattern.Number` (`reqid.go:183-191`) da el entero. `Series` es comparable (`series.go:7`). `SeriesOf` solo existe en `reqid/export_test.go:12` y no hace falta |
| Gramática del número | `<m>` = `[0-9]+`; `<N×k>` = `[0-9]{k}\|[1-9][0-9]{k,}` (`template.go:236-243`); k entre 2 y 6 (`template.go:100`). No existe un marcador de ancho 1, y `<n>` es la serie, no el número (`template.go:89-94`): una plantilla sin `<m>`/`<N…>` falla con «falta el marcador de número» (`template.go:122-124`) |
| Provisionales | `ProvisionalInTitle` (`provisional.go:90`), `ParseProvisional` valida la clave contra los marcadores `<DOM>`/`<n>` del patrón y rechaza las etiquetas sin `<n>` (`provisional.go:155-217`, en particular `:197` y `:213`); `ProvisionalText` cita la forma sin repetir el literal fuera de `reqid` (`provisional.go:27`) |
| Asignación al archivar | `livespec/ids.go:496-641` (`assign`): ocupa primero los definitivos y asigna los provisionales con `alloc.Next`; la colisión de un ADDED definitivo se busca por cadena (`ids.go:427`) |
| Pruebas que fijan lo viejo | `adoptcheck/check_pattern_test.go:23,38`; `internal/cli/archive_idpattern_test.go:136`; `internal/assets/assets_test.go:3544-3570` (pins «(REQ-n.m)», «one series `REQ-n.*`…», «never the identity…», «`axiom sdd archive-compose --canonical --delta`») |
| Goldens | Ningún fichero de `testdata/golden/` contiene «adopt» (`rg -l adopt testdata/golden` sin resultados); la referencia se instala byte a byte (`internal/components/sdd/inject_test.go:8148`) |
| Contexto técnico | `axiom workspace context --json` devuelve `technicalContext: null`, así que no hay `context-impact.md` |

## Decisiones de arquitectura

### D1. Identificador propio de una unidad (`IdentityIDs`)

**Elección.** Una unidad del inventario tiene como identificadores propios los IDs ajenos de su **título** (`foreignIDs(u.Title)`). Si el título no tiene ninguno, cuentan los de su sección (`u.ForeignIDs`). Con un identificador, la unidad lo trae. Con varios, la unidad es ambigua y eso es un conflicto (D3, D6). Con ninguno, la unidad no trae ID. La propuesta de `scan`, `pattern_undeclared` y la regla `origin_id` se calculan solo con los identificadores propios. Las referencias cruzadas del cuerpo («depende de RF-003») no cuentan cuando el título ya identifica la unidad.

**Alternativas.** (a) Todos los `ForeignIDs` de la unidad: cualquier documento real con referencias cruzadas marcaría casi todas las unidades como ambiguas, y un `RNF` citado desde un `RF` daría un falso conflicto de prefijos. (b) Solo el título: un origen con el ID en el cuerpo (`**ID:** RF-001`) perdería sus IDs sin avisar.

**Motivo.** El título es donde un documento de requisitos pone la identidad, y el cuerpo es la alternativa cuando el título no la lleva. Se calcula a partir de `Title` y `ForeignIDs`, que ya están en el inventario, así que funciona con inventarios escritos antes de este cambio y no cambia el esquema. REQ-30.9 habla de «los identificadores ajenos de todas las unidades»: este design lo interpreta como los identificadores propios de cada unidad (riesgo R1).

### D2. Inferencia de la plantilla (`adoption.ProposePattern`)

**Elección.** Es una función pura en `internal/adoption/pattern.go`, determinista, sobre las unidades en el orden del inventario. Cada identificador propio se descompone con una expresión anclada de forma (no de prefijos): `^([A-Z]+)([-_])(?:([A-Z]{2,5})([-_]))?([0-9]{1,5})$`, que cubre todo lo que produce `foreignIDPattern`.

1. **Esquema.** Se calcula el conjunto de prefijos, de separadores (el primero y, con dominio, el segundo) y de presencia de dominio. Si alguno tiene más de un valor, se registra un conflicto de esquema y no se propone plantilla (D3).
2. **Literal.** El literal es `<prefijo><sep1>` y, si todos los IDs llevan dominio, se añade `<DOM><sep2>`. Los dominios pueden variar: cada uno es una serie.
3. **Marcador de número.** Sea `k` el ancho mínimo de los números. Se usa `<N×k>` (k entre 2 y 6) si se cumplen las dos condiciones: (i) algún número de ancho `k` empieza por `0`, lo que prueba el relleno, y (ii) todos los demás tienen ancho `k` o son más anchos y no empiezan por `0` (el desbordamiento que `reqid` ya admite). En cualquier otro caso se usa `<m>`.
4. **Validación (regla 1 de REQ-30.9).** `reqid.Parse(plantilla)` debe aceptar la plantilla y `Valid(id)` debe aceptar todos los IDs. Si no, hay conflicto `template_invalid` y no se propone plantilla. Es una defensa: con los pasos 1 a 3 no debería ocurrir.

| IDs propios | Plantilla | Motivo |
|---|---|---|
| `RF-001`…`RF-120` | `RF-<NNN>` | ancho 3 con ceros |
| `RF-01`…`RF-45` | `RF-<NN>` | ancho 2 con ceros |
| `RF-01`, `RF-100` | `RF-<NN>` | `100` es el desbordamiento de `<NN>` (`[1-9][0-9]{2,}`) |
| `RF-1`…`RF-9` | `RF-<m>` | no hay marcador de ancho 1; `<n>` no es un número |
| `RF-10`…`RF-99` (sin ceros) | `RF-<m>` | no hay prueba de relleno |
| `RF-1`, `RF-2`, `RF-010` | `RF-<m>` | anchos mixtos del mismo esquema |
| `RF-001`, `RF-01` | `RF-<m>` | `001` empieza por `0` y es más ancho que el mínimo |
| un único `RF-001` / `RF-7` | `RF-<NNN>` / `RF-<m>` | mismas reglas |
| `RF-USU-01`, `RF-FAC-02` | `RF-<DOM>-<NN>` | dominio en todos |
| `RF-USU-1`, `RF-FAC-010` | `RF-<DOM>-<m>` | dominio con anchos mixtos |
| `RF_001` | `RF_<NNN>` | el separador `_` es válido en el literal inicial |
| `REQ-12`, `REQ-13` | `REQ-<m>` | un `REQ-n` ajeno sin punto; `foreignIDs` ya excluye `REQ-n.m` (`scan.go:204-206`) |
| `RF-001`, `RNF-001` | — | conflicto `mixed_prefixes` |
| `RF-001`, `RF-USU-01` | — | conflicto `mixed_domain` |
| `RF-001`, `RF_002` | — | conflicto `mixed_separators` |

**Alternativas.** (a) `<N×k>` siempre que el ancho sea único: con `RF-10…RF-99` propondría `<NN>` sin pruebas de relleno. (b) El prefijo y el separador más frecuentes, con un aviso: contradice (c′), donde una mezcla no se admite.

**Motivo.** Cada caso se traduce a la gramática real de `reqid`. Con `<NNN>`, el siguiente de `RF-120` es `RF-121`, y el de `RF-999` es `RF-1000`, que `Format` no trunca y `Valid` acepta (`reqid.go:195-201`, `template.go:243`).

### D3. Forma de la propuesta y conflictos

**Elección.** `Inventory.ProposedIDPattern *PatternProposal` (`yaml`/`json`: `proposed_id_pattern,omitempty`) es nil cuando no hay identificadores propios. Los conflictos tienen un código estable:

| Código | Clase | ¿Impide la plantilla? |
|---|---|---|
| `mixed_prefixes` | esquema | sí |
| `mixed_separators` | esquema | sí |
| `mixed_domain` | esquema | sí |
| `template_invalid` | esquema | sí |
| `ambiguous_unit_id` | unidad: varios IDs propios en una unidad | no |
| `duplicate_foreign_id` | unidad: el mismo ID propio en dos unidades o en dos orígenes. Con plantilla, se compara por (serie, número) del patrón propuesto (`RF-1` y `RF-001` con `<m>` se repiten); sin plantilla, por cadena | no |

La evidencia es la lista `{dimension, value, count}` con las dimensiones en orden fijo (`prefix`, `separator`, `domain`, `width`). Los valores van en orden natural y `domain` vale `""` cuando no hay dominio. Los conflictos van primero los de esquema y después los de unidad, en el orden del inventario.

**Motivo.** Un conflicto de unidad no hace que el esquema sea incoherente, así que la plantilla se sigue proponiendo y la persona ve los dos datos. La regla 3 de REQ-30.9 solo prohíbe la plantilla cuando el esquema se mezcla. El campo es aditivo y `omitempty`, no entra en `Summarize` ni en el hash de `inventory_stale`, y los lectores v1 lo ignoran.

### D4. `adopt check` mide con el patrón efectivo

**Elección.** La firma pasa a ser `adoptcheck.Check(loc workspace.Location, change string, pattern reqid.Pattern) (CheckReport, error)`. La CLI obtiene el patrón con `projectIDPattern("archive adopt check", loc)` antes de llamar a `Check`, así que un patrón inaccesible o inválido da exactamente el error de `idpattern.go:45-51`. Se retiran `legacy` y `RuleSeries`. `LoadRegistry` recibe `Pattern: pattern`. `CheckReport` añade `IDPattern string json:"id_pattern"` e `IDPatternDeclared bool json:"id_pattern_declared"`; esta marca vale `loc.Declarations.SDD != nil && loc.Declarations.SDD.IDPattern != nil`. El texto del informe abre con «Patrón de identificadores: `<plantilla>` (declarado en axiom.yaml | por defecto)».

**Alternativas.** `Check` llamaría a `loc.Declarations.IDPattern()` y envolvería el error por su cuenta. Se descarta porque duplicaría el mensaje accionable y la CLI ya lo envuelve con su continuación (`archive_adopt.go:253-256`).

**Formato (regla a).** Para cada ADDED:

1. Si `pattern.InTitle(name) != ""`, el título lleva un definitivo.
2. Si no, si `reqid.ProvisionalInTitle(name)` devuelve un token, `pattern.ParseProvisional(token)` debe aceptarlo. Así se exige que la clave del provisional sea compatible con el patrón efectivo: con `RF-<NNN>` solo vale `prov::<ordinal>`; con `RF-<DOM>-<NN>`, `prov:<DOM>:<ordinal>`; con el patrón por defecto, `prov:<n>:<ordinal>` o `prov:<etiqueta>:<ordinal>`. Si `ParseProvisional` lo rechaza, falla `format` con el `Reason` y el `Code` de `reqid.ProvisionalError`.
3. Si no hay ninguno de los dos, falla `format` nombrando el requisito, el último `(…)` del título si existe (como «identificador `REQ-40.1`»), la plantilla efectiva (`pattern.Compact()`) y la forma del provisional (`reqid.ProvisionalText("<clave>", "<ordinal>")`).
4. Un título con un definitivo válido y un provisional a la vez falla `format` por ambiguo, igual que `provisional_ambiguous` en C-03-b.

**Unicidad (regla b).** La clave de identidad es `idKey{reqid.Series; int}`, que se obtiene con `pattern.Series` y `pattern.Number`. Dentro del cambio se agrupa por clave: dos requisitos con la misma clave fallan `uniqueness` nombrando ambos y la clave («serie `RF-*`, número 1»). Contra el registro se usa `reg.LookupIdentity(id)` (D5). Dos provisionales con el mismo token también fallan `uniqueness`.

**Serie.** Se retira la regla. Un cambio puede usar cualquier serie, prefijo o dominio que admita el patrón.

### D5. Registro de IDs ocupados por clave de identidad

**Elección.** Se añade `func (r *Registry) LookupIdentity(id string) []IDRef` en `internal/livespec/registry.go`. Devuelve las apariciones cuyo identificador tiene la misma `Series` y el mismo `Number` que `id` según `r.pattern`, en orden estable (ruta y después ID). Se apoya en un índice `byKey map[identityKey][]IDRef` que se llena en `add` (`registry.go:120-127`). `Lookup` no cambia.

**Alternativas.** (a) Hacer la comparación en `adoptcheck` recorriendo un volcado del registro: duplica el índice y expone el interior de `Registry`. (b) Exportar `SeriesOf`: no hace falta, porque la clave se construye con `Series` y `Number` de producción. (c) Cambiar también la colisión de `archive-integrate` (`ids.go:427`) a clave de identidad: cambia C-03-b fuera del alcance (riesgo R3).

**Motivo.** Un único registro, el de C-03-b, para specs vivas y cambios activos, sin duplicar lógica. El ámbito de «nunca reutilizados» es el registro actual, como dice la proposal.

### D6. Conservación del ID del cliente: regla `origin_id`

**Elección.** Es una regla nueva, `RuleOriginID = "origin_id"`. Se evalúa por requisito con el inventario ya leído. El inventario se lee una sola vez al inicio de `Check` (`c.inv`, `c.invErr`) y se indexa por `<ruta>#<ancla>`. Para cada ADDED:

1. **Esperados.** `E` es la unión de los `IdentityIDs` de las unidades que citan sus `Origen:` y de los `OriginalID` no vacíos de esos `Origen:`.
2. **`Origen:` contra la unidad.** Si un `Origen:` trae `OriginalID` y su unidad tiene identificador propio, deben coincidir por clave. Si no coinciden, falla `origin_id` nombrando el `Origen:`, el ID escrito y el de la unidad.
3. **Provisional con ID esperado.** Si el título lleva un provisional y `E` no está vacío, falla `origin_id`: «conserva el identificador del cliente `RF-001`».
4. **Definitivo sin ID esperado.** Si el título lleva un definitivo, `E` está vacío y todas las unidades citadas están en el inventario, falla `origin_id`: «la unidad no trae identificador; usa un provisional». Así no se inventan IDs.
5. **Definitivo que no coincide.** Si el título lleva un definitivo y `E` no está vacío, el título debe coincidir por clave con algún elemento de `E`. Si no, falla `origin_id` nombrando el ID del título y todos los de `E`.

La comparación usa la clave (serie, número) del patrón efectivo cuando los dos IDs son válidos. Si alguno no lo es, compara la cadena exacta (ese ID ya lo cubre `pattern_undeclared` o `format`). Sin inventario legible, solo se aplican los pasos 2, 3 y 5 con los `OriginalID`, y se omite el 4 (ya falla `inventory_missing`).

**Alternativas.** Comparar solo con `Origen:`, como dice literalmente la regla (f): si `Origen:` omite el ID, el del cliente se pierde sin aviso (hueco 1).

**Motivo.** Cumple la regla (f) de REQ-30.11 (con varios `Origen:`, basta uno) y además usa el inventario como referencia. Los escenarios de la spec dan el mismo resultado.

### D7. `pattern_undeclared` y `foreign_id_conflict`

**Elección.** Hay dos reglas nuevas que se calculan con las unidades del inventario guardado. Quedan fuera las unidades listadas en `## Descartado` de `adoption.md`, tengan motivo o no; la falta de motivo ya la informa `discard_reason`. Con ese subconjunto, `check` llama a `adoption.ProposePattern`.

- **`RulePatternUndeclared = "pattern_undeclared"`.** Es un único fallo de proyecto (sujeto `sdd.id_pattern`) cuando algún identificador propio no es `Valid` para el patrón efectivo. El motivo nombra el patrón efectivo y su origen (declarado o por defecto), el número de IDs no reconocidos con hasta tres ejemplos, y:
  - con plantilla propuesta: «copia en axiom.yaml las líneas `sdd:` e `  id_pattern: "RF-<NNN>"` (si ya existe el bloque `sdd:`, añade solo `id_pattern`) en la rama del cambio y vuelve a ejecutar `adopt check`»;
  - con conflicto de esquema: los conflictos con su recuento y «normaliza a mano los identificadores del origen (Axiom no los reescribe), vuelve a generar el inventario con `<scanCommand>` y declara el patrón que proponga».
- **`RuleForeignIDConflict = "foreign_id_conflict"`.** Hay un fallo por cada conflicto de unidad (`ambiguous_unit_id`, `duplicate_foreign_id`), con la unidad o las unidades como sujeto, y la continuación de normalizar el origen o descartar la unidad.

Un mismo ID ajeno nunca aparece en dos motivos de proyecto. Los conflictos de esquema solo van dentro de `pattern_undeclared` y los de unidad solo en `foreign_id_conflict`. Un título fuera del patrón sigue fallando `format` en su requisito, porque lo exige el escenario «Una mezcla que no cubre el patrón declarado se rechaza» (riesgo R2). Orden de las reglas en el informe: `format`, `uniqueness`, `scenarios`, `origin`, `origin_id`, `coverage`, `discard_reason`, `inventory_missing`/`inventory_stale`, `pattern_undeclared`, `foreign_id_conflict`.

**Motivo.** La regla se deriva de los `ForeignIDs` del inventario, así que funciona con inventarios v1 sin `proposed_id_pattern` y cierra la circularidad: `check` y `archive-integrate` miden con el mismo `sdd.id_pattern`. Recalcular la propuesta sin los descartes evita que un glosario descartado con `RNF-001` fuerce un conflicto.

### D8. `adopt scan`: bloque del patrón propuesto y del declarado

**Elección.** `Scan` asigna `inv.ProposedIDPattern = ProposePattern(inv.Sources)` al final, sin leer `axiom.yaml`: el inventario sigue dependiendo solo del árbol. La CLI (`runArchiveAdoptScan`) lee además la declaración con `loc.Declarations`. Esto no afecta al inventario escrito: solo al informe.

- `adoptScanReport` añade `DeclaredIDPattern *adoptDeclaredPattern json:"declared_id_pattern,omitempty"` con `{template, matches_proposal, uncovered[]}`, presente solo si `axiom.yaml` declara `sdd.id_pattern`. También añade `DeclaredIDPatternError string json:"declared_id_pattern_error,omitempty"` cuando `IDPattern()` falla. En ese caso `scan` no se niega, porque su inventario no depende del patrón, pero informa el error y la continuación `axiom workspace validate`, sin caer en silencio al defecto.
- Texto, tras los candidatos a `technical_context`:

```
Patrón de identificadores propuesto (sdd.id_pattern; axiom.yaml no se modifica):
  plantilla: RF-<NNN>
  evidencia: 120 identificador(es) · prefijo RF ×120 · separador «-» ×120 · sin dominio ×120 · ancho 3 ×120
  para declararlo, la persona copia en axiom.yaml:
    sdd:
      id_pattern: "RF-<NNN>"
  conflictos: (ninguno | - [código] detalle)
Patrón declarado en axiom.yaml: RF-<NNN> (coincide con la propuesta | difiere de la propuesta | no reconoce N identificador(es): …)
```

Sin plantilla, el bloque sustituye las líneas de copia por «sin plantilla: normaliza a mano los identificadores del origen; Axiom no los reescribe». No hay ninguna opción `--write-pattern`.

**Motivo.** Cubre el re-escaneo tras declarar (hueco 9) sin que el inventario dependa de `axiom.yaml`. Por eso editar `axiom.yaml` no cambia ni el inventario ni `inventory_stale` (hueco 8).

### D9. Ayuda y textos de `archive adopt`

**Elección.** La línea de `check` de la ayuda (`archive_adopt.go:55-57`) pasa a: «solo ADDED; en cada título, un identificador del patrón declarado (sdd.id_pattern; sin declaración, (%s)) o un provisional (%s); únicos por serie y número, con el ID del cliente conservado (`origin_id`) y el patrón declarado (`pattern_undeclared`); escenarios; Origen: que resuelven; cobertura del inventario…». Se construye con `reqid.Default().Compact()` y `reqid.ProvisionalText("<clave>", "<ordinal>")` para respetar `TestNoRequirementIDFormatOutsideReqid`. La línea de `scan` añade «y propone el patrón de identificadores (sdd.id_pattern), sin escribir axiom.yaml». Se quitan el comentario «hasta C-03-c» y el de `check.go:24-25`.

### D10. Guías y documentación

**Elección.**

- `internal/assets/skills/sdd-spec/references/openspec-adoption.md`:
  - «Who checks what» sin «one new series».
  - Nuevo paso 1b «Declare the identifier pattern»: revisar `proposed_id_pattern` y sus conflictos; la persona declara `sdd.id_pattern` en `axiom.yaml` en la rama del cambio antes de `check`; el escáner no lo escribe; las mezclas se normalizan a mano.
  - Título del paso 3: el ID del cliente conservado, que coincide con el `<original ID>` de `Origen:`, o, si la unidad no trae ID, un provisional de la forma `prov:<series>:<ordinal>` cuyo número se asigna al archivar; los provisionales se numeran por orden de aparición.
  - Reglas del paso 5: `format`, `uniqueness`, `scenarios`, `origin`, `origin_id`, `coverage`, `discard_reason`, `inventory_missing`, `inventory_stale`, `pattern_undeclared`, `foreign_id_conflict`.
  - Paso 7: archivar con `axiom sdd archive-integrate`, sin ordenar `archive-compose --canonical --delta` para la adopción.
  - Límites: el límite de prefijos de `foreignIDPattern` (con otro prefijo, declarar a mano).
- `sdd-spec/SKILL.md:81` y `sdd-propose/SKILL.md:196` no contradicen lo nuevo y no se tocan. `sdd-archive/SKILL.md:135` y `_shared/sdd-iso-convention.md:18` tampoco. Se comprueba con `rg -n "new series|serie nueva|C-03-c" internal/assets/skills`.
- `docs/manual-de-inicio.md:587-605`: paso 1 con la propuesta del patrón, nuevo paso de declaración, paso 2 con IDs conservados o provisionales (sin «El identificador ajeno nunca es la identidad»), paso 3 con las reglas nuevas, paso 5 con `axiom sdd archive-integrate`. Se conservan las cadenas fijadas por `internal/workspace/docs_pin_test.go:17-18`.

## Flujo de datos

```
árbol ──► adoption.Scan ──► Units ──► IdentityIDs ──► ProposePattern ──► inventory.proposed_id_pattern
                                                                          │ (scan --change: adoption-inventory.yaml)
axiom.yaml ──► Declarations.IDPattern ──► CLI scan: bloque «declarado» (solo informe)
                     │
                     └──► CLI check: projectIDPattern ──► adoptcheck.Check(loc, change, pattern)
                                                              │
     deltas ADDED ──► formato (InTitle | ParseProvisional) ───┤
     LoadRegistry(Pattern) ──► LookupIdentity ──► unicidad ───┤
     inventario − Descartado ──► ProposePattern ──────────────┼──► pattern_undeclared / foreign_id_conflict
     Origen: + unidades del inventario ──► origin_id ─────────┘
                                                              ▼
                                         CheckReport{id_pattern, failures[]}
archivado: axiom sdd archive-integrate (C-03-b, sin cambios): ocupa RF-001…RF-120; prov::1 → RF-121
```

## Cambios de ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/adoption/pattern.go` | Crear | `IdentityIDs`, `ProposePattern`, `PatternProposal`, `PatternEvidence`, `PatternConflict`, códigos |
| `internal/adoption/pattern_test.go` | Crear | Tabla de D2, conflictos de D3, determinismo, validación con `reqid` |
| `internal/adoption/inventory.go` | Modificar | Campo `ProposedIDPattern` (`omitempty`) |
| `internal/adoption/scan.go` | Modificar | `Scan` rellena la propuesta |
| `internal/adoption/scan_test.go` | Modificar | Propuesta en el inventario, determinismo con el campo, ausencia sin IDs, lector v1 |
| `internal/livespec/registry.go`, `registry_test.go` | Modificar | `LookupIdentity` e índice `byKey` |
| `internal/adoption/adoptcheck/check.go` | Modificar | D4, D6, D7; fuera `legacy`, `RuleSeries` y `checkIdentifiers` de serie |
| `internal/adoption/adoptcheck/check_test.go`, `helpers_test.go` | Modificar | Fixture base con `RF-<NNN>`, `(RF-001)`/`(RF-002)`; se reescriben `MezclaDeSeries`, `SerieYaExistenteEnLasSpecs` y `UnaSolaSerieNuevaNoFalla` (`:186-202`) como «ya no hay serie» |
| `internal/adoption/adoptcheck/check_pattern_test.go` | Reescribir | Invierte `:23` y `:38`: con el patrón declarado mide con él; patrón inválido sin pasar por el defecto |
| `internal/adoption/adoptcheck/check_ids_test.go` | Crear | `origin_id`, `pattern_undeclared`, `foreign_id_conflict`, provisionales compatibles e incompatibles |
| `internal/cli/archive_adopt.go` | Modificar | D8, D9; `projectIDPattern` en `check` |
| `internal/cli/archive_adopt_test.go` | Modificar | Texto y JSON del bloque del patrón; `check` con el patrón declarado y con el patrón inválido; ayuda |
| `internal/cli/archive_idpattern_test.go` | Modificar | `:136` se invierte: la ayuda no contiene «de una serie nueva» y cita `sdd.id_pattern` |
| `internal/cli/archive_adopt_ids_test.go` | Crear | Prueba de integración del criterio (U5) |
| `internal/assets/skills/sdd-spec/references/openspec-adoption.md` | Modificar | D10 |
| `internal/assets/assets_test.go` | Modificar | Pins de `:3544-3570` al flujo nuevo y pins negativos («one new series», «C-03-c», «never the identity», `archive-compose --canonical --delta` en el paso de archivado) |
| `docs/manual-de-inicio.md` | Modificar | Sección de adopción (D10) |
| `testdata/golden/*` | Verificar | No contienen la guía; se regeneran con `go test ./internal/components -run TestGolden -update` solo si `TestGolden` falla |

## Interfaces

```go
// internal/adoption (pattern.go)
const (
    ConflictMixedPrefixes      = "mixed_prefixes"
    ConflictMixedSeparators    = "mixed_separators"
    ConflictMixedDomain        = "mixed_domain"
    ConflictTemplateInvalid    = "template_invalid"
    ConflictAmbiguousUnitID    = "ambiguous_unit_id"
    ConflictDuplicateForeignID = "duplicate_foreign_id"
)
type PatternProposal struct {
    Template  string            `yaml:"template,omitempty" json:"template,omitempty"`
    IDs       int               `yaml:"ids" json:"ids"`
    Evidence  []PatternEvidence `yaml:"evidence" json:"evidence"`
    Conflicts []PatternConflict `yaml:"conflicts,omitempty" json:"conflicts,omitempty"`
}
type PatternEvidence struct {
    Dimension string `yaml:"dimension" json:"dimension"` // prefix|separator|domain|width
    Value     string `yaml:"value" json:"value"`
    Count     int    `yaml:"count" json:"count"`
}
type PatternConflict struct {
    Code   string   `yaml:"code" json:"code"`
    Detail string   `yaml:"detail" json:"detail"`
    Units  []string `yaml:"units,omitempty" json:"units,omitempty"` // <ruta>#<ancla>
    IDs    []string `yaml:"ids,omitempty" json:"ids,omitempty"`
}
func IdentityIDs(u Unit) []string
func ProposePattern(sources []Source) *PatternProposal // nil sin identificadores propios
func (p *PatternProposal) SchemeConflicts() []PatternConflict
func (p *PatternProposal) UnitConflicts() []PatternConflict

// Inventory
ProposedIDPattern *PatternProposal `yaml:"proposed_id_pattern,omitempty" json:"proposed_id_pattern,omitempty"`

// internal/livespec
func (r *Registry) LookupIdentity(id string) []IDRef

// internal/adoption/adoptcheck
const (
    RuleOriginID          = "origin_id"
    RulePatternUndeclared = "pattern_undeclared"
    RuleForeignIDConflict = "foreign_id_conflict"
) // se retira RuleSeries
type CheckReport struct {
    Change            string    `json:"change"`
    IDPattern         string    `json:"id_pattern"`
    IDPatternDeclared bool      `json:"id_pattern_declared"`
    Passed            bool      `json:"passed"`
    Failures          []Failure `json:"failures"`
}
func Check(loc workspace.Location, change string, pattern reqid.Pattern) (CheckReport, error)

// internal/cli (archive_adopt.go)
type adoptDeclaredPattern struct {
    Template        string   `json:"template"`
    MatchesProposal bool     `json:"matches_proposal"`
    Uncovered       []string `json:"uncovered,omitempty"`
}
```

Retirar `RuleSeries` cambia la API de un paquete interno. Solo lo usan `check.go` y sus pruebas (`rg -n RuleSeries internal`).

## Estrategia de pruebas

TDD por unidad: hay que observar el RED antes de escribir la producción. Comandos acotados: `go test ./internal/adoption/...`, `go test ./internal/livespec -run 'Registry|Lookup'`, `go test ./internal/reqid` (sin cambios; solo como comprobación), `go test ./internal/cli -run 'ArchiveAdopt|AdoptIDs|IDPattern'`, `go test ./internal/assets -run 'SDDSpecAdoption|SDDSpecAndPropose'`, `go test ./internal/components/sdd -run Adoption`, `go test ./internal/components -run TestGolden`, `go test ./internal/workspace -run DocsPin` y `go vet` de los paquetes tocados. Nunca se ejecuta la batería completa ni se compilan binarios sueltos de `axiom`.

| Capa | Qué | Cómo |
|---|---|---|
| adoption/pattern | Cada fila de la tabla de D2; los conflictos de D3 con su recuento; `IdentityIDs` (título, cuerpo, varios, ninguno, fichero entero); determinismo; plantilla válida y `Valid` sobre todos | Tablas en `pattern_test.go` |
| adoption/scan | Propuesta `RF-<NNN>` para 120 unidades; `RF-<m>`; conflicto `RF`/`RNF`; sin IDs no hay campo; dos `Scan --json` idénticos; un inventario con el campo leído por un struct v1 | `scan_test.go` |
| livespec | `LookupIdentity` encuentra `RF-1` desde `RF-001` con `RF-<m>` y no con `RF-<NNN>`; las referencias de cambios activos cuentan; `ExcludeChange` respetado | `registry_test.go` |
| adoptcheck | Todos los escenarios de REQ-30.11 y REQ-30.12; provisional incompatible (`prov:40:1` con `RF-<NNN>`); etiqueta con `RF-<NNN>` (`provisional_new_series_unsupported`); título ambiguo; `origin_id` en los pasos 2 a 5; descartes fuera de `pattern_undeclared`; con conflicto, el motivo cita el conflicto y no las líneas; inventario sin `proposed_id_pattern`; `check` no escribe (hash del árbol, `axiom.yaml` incluido) | `check_test.go`, `check_pattern_test.go`, `check_ids_test.go` |
| cli | Patrón inválido: el mismo error de `idPatternRefusal`; JSON `id_pattern`; bloque del patrón en el texto de `scan`; `declared_id_pattern` coincide o difiere; `scan --change` deja `axiom.yaml` byte a byte igual; ayuda sin «serie nueva» | `archive_adopt_test.go`, `archive_idpattern_test.go` |
| Integración (criterio) | U5 | `archive_adopt_ids_test.go` |
| Guías | Pins de REQ-30.31 (positivos y negativos); paridad entre runtimes; manual | `assets_test.go`, `inject_test.go:8148`, `docs_pin_test.go` |

**Prueba de integración del criterio (U5), en proceso.** Usa el andamiaje de `sddIntegrateProject` (`internal/cli/sdd_archive_integrate_test.go:29-36`): `guardProject`, el cambio sellado y git de prueba.

1. Escribir `docs/requisitos.md` con `# Requisitos` y 120 secciones `## RF-%03d …`, y añadir `sdd: {id_pattern: "RF-<NNN>"}` a `axiom.yaml`.
2. `RunArchiveAdopt scan --change` → `proposed_id_pattern.template == "RF-<NNN>"`, sin conflictos y `axiom.yaml` sin cambios.
3. Escribir una delta con 120 ADDED `(RF-%03d)`, cada uno con su `Origen: docs/requisitos.md#<ancla> (RF-%03d)` y un escenario, y `adoption.md` con el título descartado.
4. `RunArchiveAdopt check` → código 0 y `id_pattern == "RF-<NNN>"`.
5. Cambiar a la rama de archivado y ejecutar `RunSDDArchiveIntegrate`: la spec viva contiene `(RF-001)`…`(RF-120)` y las 120 líneas `Origen:` byte a byte.
6. Integrar después un ADDED `(prov::1)` de la misma capacidad. El resultado es `RF-121`. Si el verbo exige un segundo cambio sellado y el andamiaje no lo ofrece, este paso usa `livespec.PlanChange` + `Apply` con `LoadRegistry(Pattern)`, que es el mismo motor.

Caso negativo: el mismo árbol sin `id_pattern` → `check` falla `pattern_undeclared` citando `id_pattern: "RF-<NNN>"`. Segundo caso: un origen sin IDs, sin `id_pattern` y con títulos `(prov:40:1)` y `(prov:40:2)` → `check` pasa y el archivado da `REQ-40.1` y `REQ-40.2`, sin provisionales en la spec viva.

## Matriz de amenazas

| Límite | Aplicabilidad | Respuesta |
|---|---|---|
| Rutas tipo documentación / clasificación de ejecutables | N/A: no se clasifica ni se ejecuta ningún fichero; `scan` sigue leyendo solo `.md/.markdown/.txt/.docx/.xlsx/.pdf` | — |
| Selección de repositorio git | N/A: `locateAdopt` y la guarda A de `scan --change` no cambian (`archive_adopt.go:113-123,149`); `check` sigue sin escribir | — |
| Subprocesos y shell | N/A: no hay comandos ni procesos nuevos | — |
| Estado de commit y de push | N/A | — |
| Comandos de PR | N/A | — |

## Unidades de implementación (un commit por unidad)

| U | Unidad | Paquetes y ficheros | Depende de |
|---|---|---|---|
| U1 | Inferencia del patrón y campo del inventario: `IdentityIDs`, `ProposePattern`, conflictos, `Scan` lo rellena | `internal/adoption` (`pattern.go`, `inventory.go`, `scan.go` y sus pruebas) | — |
| U2 | `check` mide con el patrón efectivo: firma, `projectIDPattern` en la CLI, registro con `Pattern`, formato con definitivo o provisional compatible, unicidad por clave con `LookupIdentity`, retirada de `series`, informe con `id_pattern`. Se reescriben `check_pattern_test.go` y las pruebas de serie | `internal/livespec/registry.go`, `adoptcheck`, `internal/cli/archive_adopt.go` (`check`) | — |
| U3 | Conservación del ID: `origin_id`, `pattern_undeclared` (sin descartes) y `foreign_id_conflict` | `adoptcheck` (`check.go`, `check_ids_test.go`) | U1, U2 |
| U4 | Textos de la CLI: bloque del patrón en `scan`, patrón declarado frente al propuesto, ayuda; se invierte `archive_idpattern_test.go:136` | `internal/cli` (`archive_adopt.go`, pruebas) | U1, U2 |
| U5 | Prueba de integración del criterio y del origen sin IDs | `internal/cli/archive_adopt_ids_test.go` | U1–U4 |
| U6 | Guía, manual y pins: `openspec-adoption.md`, `assets_test.go`, `docs/manual-de-inicio.md`; comprobación de goldens y de la paridad entre runtimes | `internal/assets`, `docs`, `internal/components` | U3, U4 |

U1 y U2 no dependen entre sí. Por tamaño, encajan con un escritor en línea cada una.

## Notas para la fase de archivo

`axiom sdd archive-integrate` compone solo los bloques de requisitos. Hay que aplicar a mano, en la rama de archivado, tras componer y antes de `axiom sdd archive-check`, y dejarlo anotado en `archive-report.md` como se hizo con el `## Purpose` de C-03-b:

1. `openspec/specs/openspec-adoption/spec.md:38` («No cubre»): sustituir la línea por el texto que fija la cabecera de la delta de `openspec-adoption`.
2. `openspec/specs/openspec-adoption/spec.md:30` (tabla «de un vistazo», fila REQ-30.12): sustituir el resumen por el de la cabecera de la delta. Revisar que las filas de REQ-30.9, REQ-30.10, REQ-30.11 y REQ-30.13 no contradigan el texto nuevo.
3. `openspec/specs/requirement-identifiers/spec.md:5` (`## Purpose`): quitar «la única excepción deliberada y temporal es `adoptcheck` con `archive adopt` (REQ-44.4)» y «no conserva los identificadores de un cliente al adoptar (C-03-c)». La delta de `requirement-identifiers` no lo anota: es una corrección del texto vivo que el archivo debe aplicar.

## Migración y despliegue

No hay migración de datos. Los inventarios v1 existentes se siguen leyendo, y `pattern_undeclared` y `origin_id` se derivan de sus `Title` y `ForeignIDs`. A-03-2 prohibía adoptar antes de este cambio, así que no hay ninguna adopción real que romper. La reversión deja `adopt check` en el patrón por defecto (efecto residual descrito en la proposal).

## Riesgos

- **R1. Identificador propio (D1).** REQ-30.9 dice «de todas las unidades»; el design usa los identificadores propios (título o, si no tiene, cuerpo). Si una unidad no tiene ID en el título y su cuerpo cita un único ID que no es el suyo, ese ID se tomaría como su identidad. Se puede mitigar descartando la unidad o normalizando el origen. El verify debe confirmar que ningún escenario de la spec cambia de resultado.
- **R2. Dos motivos por el mismo ID.** El escenario «Una mezcla que no cubre el patrón declarado se rechaza» exige `format` en el requisito `RNF-001`, y en ese mismo caso `pattern_undeclared` también se dispara. D7 limita a uno el motivo de proyecto por ID, pero el fallo del título se mantiene porque lo pide la spec.
- **R3. Colisión por clave en `archive-integrate`.** `livespec/ids.go:427` compara por cadena. Con `RF-<m>`, un ADDED definitivo `RF-001` frente a un `RF-1` vivo pasaría al archivar aunque `adopt check` lo rechace. Queda fuera de alcance; se propone como seguimiento en C-03-d o en un cambio de `livespec`.
- **R4. Reglas nuevas más allá del texto literal de la spec.** `origin_id` paso 4 (definitivo sin ID → provisional), `foreign_id_conflict` y la exclusión de los descartes vienen de la coordinación (huecos 1, 2 y 5) y no tienen escenario propio en la delta. Las pruebas de U3 las fijan; si el verify pide escenarios en la spec, se añaden al archivar como MODIFIED de REQ-30.11, no se reabre la spec en `continuous`.
- **R5. Lista cerrada de prefijos** (`scan.go:34`): con `SYS-`/`FUN-` no hay propuesta ni `pattern_undeclared`, y además `origin_id` paso 4 exigiría provisionales para unidades que en realidad traen ID. La guía lo documenta (declarar a mano y escribir `(<ID original>)` en `Origen:`, que entra en `E`).
- **R6. Pruebas que fijan lo viejo** (`check_pattern_test.go`, `archive_idpattern_test.go:136`, `assets_test.go:3544+`): U2, U4 y U6 las reescriben en su semántica. El verify comprueba `rg -n "hasta C-03-c|serie nueva|RuleSeries" internal docs`.

## Preguntas abiertas

- [ ] Ninguna bloquea.

## Resolución de la revisión de cobertura (spec, ronda 1)

| Hueco | Decisión | Dónde |
|---|---|---|
| 1 | Añadida como regla de implementación: la conservación se valida contra los `IdentityIDs` de las unidades del inventario además de `Origen:`; un provisional o un definitivo distinto con ID esperado se rechaza; un definitivo sin ID esperado se rechaza | D6; U3; R4 |
| 2 | Añadida: `ambiguous_unit_id` y `duplicate_foreign_id` como conflicto en `scan` (sin impedir la plantilla) y rechazo `foreign_id_conflict` en `check` | D1, D3, D7; R4 |
| 3 | Añadida: tabla completa de inferencia sobre la gramática real (`<m>` para el ancho 1 y sin pruebas de relleno; `<N×k>` con desbordamiento; `<DOM>`) | D2 |
| 4 | Añadida: con conflicto, `pattern_undeclared` cita los conflictos y remite a normalizar, sin líneas; un motivo de proyecto por ID; `format` se mantiene en el título | D7; R2 |
| 5 | Añadida: `check` recalcula la propuesta sin las unidades de `## Descartado`; `scan` no conoce los descartes y su texto no los presupone | D7 |
| 6 | Añadida: los provisionales deben ser compatibles con el patrón efectivo (`ParseProvisional`); sintaxis por patrón (`prov::<k>` con `RF-<NNN>`, `prov:<DOM>:<k>` con dominio, `prov:<n>:<k>` o etiqueta con el defecto); orden: definitivos ocupados primero y provisionales por orden de aparición (C-03-b, `ids.go:496-641`); el desbordamiento sigue a `reqid` (`RF-999` → `RF-1000`) | D2, D4; U5 |
| 7 | Añadida: igual que el 6 (compatibilidad y etiqueta sin `<n>` → `provisional_new_series_unsupported` como fallo `format`) | D4 |
| 8 | Añadida: la clave se calcula con el patrón vigente en cada `check`; el inventario guarda cadenas; `axiom.yaml` no interviene en `inventory_stale`; un provisional cuyo `Origen:` trae un ID de otro patrón se rechaza por `origin_id` paso 3 | D6, D8 |
| 9 | Añadida: `scan` muestra el patrón propuesto y el declarado, y avisa si difieren o si el declarado no reconoce los IDs, solo en el informe | D8 |
