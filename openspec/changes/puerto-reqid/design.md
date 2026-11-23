# Diseño: puerto único de identificadores de requisito y patrón declarado por proyecto (C-03-a)

> Fase `design` del cambio `puerto-reqid`. Rutas relativas a la raíz del worktree. Castellano de España; identificadores de código, claves, códigos y tokens en inglés. Entradas: `proposal.md`, `explore.md`, `docs/plan/backlog-2026-11/03-openspec-unica.md` (C-03-a) y el código de `main` en `a8cb13b4`. Decisiones cerradas que no se reabren: D-03-3, D-03-8 y A-03-1. La fase `spec` corre en paralelo: este documento no numera requisitos nuevos (familia REQ-44 propuesta) y en §11 dice qué contratos debe recoger la spec.

## 1. Enfoque técnico

Un paquete hoja nuevo, `internal/reqid` (solo biblioteca estándar), es el único lugar donde existe la forma de un identificador de requisito: compila una **plantilla de segmentos** declarada en `sdd.id_pattern` de `axiom.yaml` y ofrece validación, extracción del título, detección laxa de candidatos, serie, número, formato del siguiente ID, notaciones para mensajes y el ID sintético de `livingdoc`. Ningún otro paquete conserva una expresión regular ni un `fmt.Sprintf` con la forma del ID.

- **Valor por defecto byte a byte.** Sin declaración rige la plantilla `REQ-<n>.<m>`, que compila exactamente a las expresiones vigentes (`^REQ-\d+\.\d+$`, `\((REQ-\d+\.\d+)\)`, `\bREQ-[0-9A-Za-z.]*[0-9A-Za-z]`, `\((REQ-[^)]+)\)`). Su notación es la propia plantilla (`REQ-<n>.<m>`) y su notación compacta, `REQ-n.m`, de modo que todos los mensajes, continuaciones y pines actuales siguen iguales sin ningún caso especial.
- **El ID deja de ser un campo precalculado.** `openspecdoc.Requirement.ID` y `openspecdoc.RequirementID` desaparecen: el ID es siempre `pattern.InTitle(name)`. Así ningún consumidor puede leer un ID calculado con un patrón por defecto oculto.
- **El patrón se lee en vivo** de `axiom.yaml` (`workspace.Declarations.IDPattern()`), nunca de `kickoff.yaml`. Un patrón inválido es error de `SDDErr`: falla la carga estricta y `axiom workspace validate`, y los consumidores que lo necesitan se niegan con continuación; ninguno cae en silencio al defecto, salvo `livingdoc`, que es informativo y avisa.
- **El compositor resuelve la identidad por ID** cuando el nombre de la delta lleva un ID válido, con vuelta al título exacto ante ambigüedad, y por título en el resto. Toda composición hoy aceptada produce la misma salida byte a byte; lo único que pasa de rechazo a aceptación es el caso objetivo (MODIFIED, REMOVED o el nombre antiguo de un RENAMED cuyo título cambió pero conserva el ID).

## 2. Decisiones de arquitectura

### D-01. Clave `sdd.id_pattern`: plantilla de segmentos

**Elección.** Escalar de texto `sdd.id_pattern` en el bloque `sdd:` (inglés y plano, como `iso`). Su valor es una plantilla de literales y marcadores:

| Marcador | Papel | Encaja con | `Format` |
|---|---|---|---|
| `<DOM>` | serie: código de dominio | `[A-Z][A-Z0-9]*` | el valor de la serie |
| `<n>` | serie: número | `[0-9]+` | el valor de la serie |
| `<m>` | número de secuencia sin relleno | `[0-9]+` | `strconv.Itoa(n)` |
| `<NN>` … `<NNNNNN>` (k = 2..6) | número de secuencia con relleno a k cifras | forma canónica: exactamente k cifras, o más de k sin cero inicial (`[0-9]{k}\|[1-9][0-9]{k,}`) | `%0kd` |

Gramática (todas las reglas son errores de carga con motivo):

1. No vacía y de 64 caracteres como máximo.
2. Empieza por un **literal inicial** de la forma `[A-Z][A-Z0-9]*[-._]` (una letra mayúscula, mayúsculas o cifras y un separador final). Es el ancla de la detección de candidatos y de la lectura laxa.
3. Los literales solo contienen `[A-Z0-9._-]`. Quedan fuera las minúsculas, los espacios, los paréntesis, la coma y `<`/`>`.
4. Exactamente un marcador de número (`<m>` o `<N…>`); como máximo un `<DOM>` y un `<n>`. `<N>` con una sola N no existe (es `<m>`).
5. Dos marcadores nunca van seguidos, y un literal pegado a un marcador no empieza ni termina con un carácter del alfabeto de ese marcador (cifras para `<n>`, `<m>` y `<N…>`; `[A-Z0-9]` para `<DOM>`). Garantiza que el reparto en segmentos es único.
6. Un `<…>` desconocido, un `<` sin cerrar o un `>` suelto son errores que nombran el texto.

Ejemplos: `REQ-<n>.<m>` (defecto, `REQ-30.4`), `RF-<DOM>-<NN>` (`RF-USU-01`, D-03-3), `RF-<m>` (serie heredada `RF-121`, D-03-8 y A-03-1), `HU-<NNN>` (`HU-007`).

**Conceptos.**

- **Válido**: encaja entero con la regex compilada (`^…$`).
- **Candidato**: token que empieza, en límite de palabra, por el literal inicial y sigue con `[0-9A-Za-z<extra>]*[0-9A-Za-z]`, donde `<extra>` son los caracteres no alfanuméricos de los literales posteriores al inicial. Para el defecto es exactamente `\bREQ-[0-9A-Za-z.]*[0-9A-Za-z]`. Un candidato que no es válido es **mal formado**.
- **Serie**: el ID sin su número: `prefix` (texto anterior al número), `suffix` (posterior) y `key` (valores de `<DOM>` y `<n>` unidos por `-`). `REQ-30.4` → serie `REQ-30.*`, clave `30`; `RF-USU-01` → `RF-USU-*`, clave `USU`; `RF-121` → `RF-*`, clave vacía (serie única).
- **Número**: el valor entero del segmento de número (`strconv.Atoi`; desbordamiento = no válido).
- **Identidad**: la cadena exacta. Con `<N…>` la forma es canónica, así que `RF-USU-01` y `RF-USU-001` no pueden ser ambos válidos; con `<m>` y `<n>` se conserva la permisividad actual (`REQ-30.01` sigue siendo válido, como hoy).

**Notaciones para mensajes.** `Template()` es la plantilla tal cual (`REQ-<n>.<m>`, `RF-<DOM>-<NN>`); `Compact()` le quita los ángulos (`REQ-n.m`, `RF-DOM-NN`); `SeriesCompact()` sustituye el número por `*` (`REQ-n.*`). Son las tres notaciones que ya usan hoy los mensajes, así que el defecto no cambia ni un byte.

**Error de carga.** `reqid.Parse` devuelve `*reqid.TemplateError{Template, Reason}`; `decodeSDD` lo envuelve: `sdd.id_pattern (línea N): la plantilla "RF-<DOM><NN>" no es válida: <motivo>; un ejemplo válido es "RF-<DOM>-<NN>"; para el patrón por defecto (<Default().Template()>) retira la clave`. Un nodo que no es escalar de texto (`!!str`), el nulo o la cadena vacía son errores propios de `sdd.id_pattern`; la clave repetida y la desconocida siguen las reglas actuales del bloque.

**Alternativas descartadas.** (A) Regex declarada: no es invertible (no hay `Format`), el candidato laxo no se deriva, los mensajes serían ilegibles y abre la puerta a regex que admiten espacios o comas, que romperían `Cubre:`. (C) Mapa estructurado (`prefix`, `domain`, `width`): más claves en un esquema compartido y no expresa el defecto `REQ-<n>.<m>` sin un campo de separadores. (B') Plantilla con marcadores `<S>`/`<N>` y una notación «de lectura» especial para el defecto: obliga a un caso especial en `Template()`; con `<n>`/`<m>` el defecto se describe a sí mismo y coincide con la prosa vigente de specs y guías.

### D-02. Paquete hoja `internal/reqid` con `Pattern` de valor cero = defecto

**Elección.** `reqid.Pattern` es un valor inmutable y comparable (`struct{ c *compiled }`); el valor cero se comporta exactamente como `Default()`. Así los campos nuevos en estructuras de opciones (`openspecdoc.ComposeOptions.IDPattern`, `livespec.RegistryOptions.Pattern`) no rompen a ningún llamador y el defecto es explícito en la documentación del tipo. `reqid` importa solo `fmt`, `regexp`, `strconv`, `strings` y `errors`; una prueba de importaciones lo fija.

**Alternativas descartadas.** Interfaz definida por cada consumidor (`isoevidence.IDScheme`): una sola implementación, más indirección y la misma dependencia en las pruebas. Constructor obligatorio sin valor cero útil: forzaría a tocar los 27 usos de `Compose` y los 19 de `LoadRegistry` en pruebas sin ningún beneficio.

**Excepción de hoja.** `openspecdoc` e `isoevidence` dejan de ser «sin importaciones internas» y pasan a «solo `internal/reqid`». `isoevidence/imports_test.go` admite exactamente esa ruta; se añade la misma prueba a `openspecdoc`.

### D-03. Llegada del patrón a cada consumidor

`workspace.Declarations.IDPattern() (reqid.Pattern, error)`: con `ParseErr` o `SDDErr` devuelve un error que los envuelve (`%w`); sin bloque `sdd:` o sin la clave, `reqid.Default()`; con la clave, el patrón ya validado al decodificar.

| Consumidor | Cómo recibe el patrón | Con el patrón inaccesible |
|---|---|---|
| `openspecdoc` | `ComposeOptions.IDPattern` (cero = defecto) y parámetro de `Resolve` | — (no lee `axiom.yaml`) |
| `isoevidence` | parámetro final `p reqid.Pattern` en seis funciones (§5.3) | — |
| `sddstatus` (ISO) | `location.Declarations.IDPattern()`, guardado en `isoInputs` | ver D-07 |
| `livespec` | `RegistryOptions.Pattern`; `Registry.Pattern()` lo da a `PlanIntegration` | — |
| `cli/archive_integrate.go` | `loc.Declarations.IDPattern()` → `RegistryOptions.Pattern` | se niega sin escribir, con `axiom workspace validate --path …` |
| `cli/sdd_archive_compose.go` | workspace de `--cwd` o, sin él, el que contiene la delta (como `refuseISOCompose`); sin workspace, defecto | se niega sin escribir |
| `cli/sdd_archive_check.go`, `cli/sdd_trace.go` | `loc.Declarations.IDPattern()` | error con la continuación de `validate` |
| `livingdoc.Service` | resuelve `loc` y usa `Indexer.WithPattern(p)` | usa el defecto y añade un aviso a `SyncReport.Warnings` (es informativo) |
| `adoption/adoptcheck` | **siempre `reqid.Default()`** | — |

`adoptcheck` queda anclado al defecto a propósito: REQ-30.31 y A-03-2 mantienen `adopt check` en `REQ-n.m` hasta C-03-c. Usar el patrón del proyecto cambiaría sus reglas en un proyecto que declare otro patrón, y eso es alcance de C-03-c.

**Alternativa descartada.** Variantes `ParseDeltaWith`/opciones variádicas en `openspecdoc` que rellenen `Requirement.ID`: dejan `ParseDelta(text)` calculando IDs con un defecto oculto, y un consumidor olvidado validaría con el patrón equivocado sin que nada lo detecte.

### D-04. Patrón leído en vivo, sin instantánea en `kickoff.yaml`

**Elección.** El patrón se lee de `axiom.yaml` en cada uso. `kickoff.yaml` no cambia de esquema.

**Motivo.** `sdd.iso` se congela (REQ-29.13) porque decide el **proceso** de un cambio; el patrón define el **vocabulario de la spec viva compartida**. Las deltas de un cambio se componen contra `openspec/specs/` vigente, y `livespec`, `adopt check` y `sdd-archive-compose` no tienen kickoff. Con instantánea, un cambio en curso validaría y compondría con un patrón distinto del de la canónica contra la que se archiva. Cambiar el patrón de un proyecto es una migración de datos (C-03-e) que reescribe la canónica; tras ella, los cambios en curso deben revalidarse contra el patrón nuevo, que es justo lo que hace la lectura en vivo (fallo cerrado con motivos tipados, no corrupción).

**Alternativa descartada.** Instantánea `sdd.id_pattern` en `kickoff.yaml`: amplía `axiom.sdd-kickoff/v1`, solo cubre el carril SDD y produce la divergencia descrita.

### D-05. Compositor por ID (`openspecdoc`)

**Elección.** Una única función de resolución, `openspecdoc.Resolve`, la usan el compositor, la clasificación idempotente de `livespec` y `alreadyComposed` de `archive-check`, para que los tres coincidan siempre. *Ajuste del 2026-11-23:* su núcleo vive en `reqid.Pattern.Resolve` (con `Match`, `SupersededSuffix` e `IsSuperseded`) y `openspecdoc.Resolve` delega en él, para que `isoevidence` resuelva el «existente» de un MODIFIED/REMOVED y la tolerancia D17 igual que el compositor sin importar `openspecdoc` (regla de hoja de D-02):

1. `id := p.InTitle(name)`. Si `id == ""`, se busca el **título exacto** (comportamiento actual; primera coincidencia).
2. Si hay `id`, se buscan los requisitos de la canónica **no superados** (nombre sin el sufijo ` [SUPERSEDED / DEPRECADO]`) cuyo `p.InTitle` es `id`:
   - exactamente uno → ese;
   - ninguno → no encontrado (mismo motivo de hoy, `no canonical requirement named %q`);
   - dos o más → si exactamente uno tiene el título exacto, ese (es lo que hoy elegiría el título); si no, `MatchAmbiguous` con el motivo `identifier %s is carried by %d canonical requirements and none is named %q`.

Aplicación por operación:

| Operación | Resolución |
|---|---|
| MODIFIED | `Resolve`; además el segmento toma el nombre de la delta (`segments[i].Name = r.Name`), para que las operaciones siguientes de la misma delta lo vean |
| REMOVED | `Resolve`; con `SupersedeRemoved`, la anotación usa el nombre del **segmento** de la canónica (hoy es idéntico al de la delta) |
| RENAMED, nombre antiguo | `Resolve` (sigue exigiendo `Antiguo → Nuevo` y la nota `(Reason: …)`) |
| RENAMED, colisión del nombre nuevo | título exacto (sin cambios) |
| ADDED, existencia previa | título exacto (sin cambios) |
| `ExplicitSuperseded` | título exacto (sin cambios) |

**Invariante de no regresión.** Si hoy una composición se acepta, cada nombre de la delta encontró su título exacto `S`; `S` lleva el mismo ID que la delta; o es el único no superado con ese ID (mismo segmento), o hay varios y gana el título exacto (mismo segmento). Por tanto la salida es la misma byte a byte. Lo que hoy se rechaza y se sigue rechazando puede cambiar de motivo solo en el caso ambiguo.

**ID repetido en la canónica.** No se rechaza la canónica entera: se resuelve por título exacto y solo es error si tampoco hay título exacto. La unicidad de un ADDED la siguen imponiendo REQ-29.3 (ISO) y el registro de REQ-30.4 (ODD), antes de componer.

**Requisitos superados.** Conservan su `(ID)` en el título (lo comprobé en `sdd-role-closure-handoff/spec.md:78` y `workspace-topology/spec.md:159`), pero quedan fuera de la resolución por ID: su identificador está retirado y un MODIFIED no debe reabrirlos. Siguen en el registro de `livespec` (el ID no se reutiliza).

**Encaje con D17 (archivado interrumpido).**

- `isoevidence.tolerated` (ADDED con el mismo nombre y bloque) no cambia: los ADDED siguen por título.
- `alreadyComposed` (`cli/sdd_archive_check.go`) sigue por nombre para ADDED, MODIFIED y RENAMED, porque tras componer la canónica lleva el bloque de la delta con su encabezado. **REMOVED pasa a `Resolve`**: hoy `!present` por nombre; con un REMOVED por ID y título distinto, el nombre no está antes de componer y el dominio se daría por compuesto sin borrar nada. Con `Resolve`, «compuesto» es «no se resuelve».
- `livespec.classify` pasa a `Resolve` para MODIFIED (pendiente salvo que el resuelto tenga el mismo cuerpo normalizado) y REMOVED (pendiente si se resuelve). Tras integrar, el MODIFIED queda con el nombre nuevo y la segunda ejecución da `already`: la idempotencia de REQ-30.2 se conserva.

**Alternativas descartadas.** Rechazar todo ID repetido en la canónica: convertiría en rechazo composiciones hoy aceptadas. Resolver también ADDED y la colisión del RENAMED por ID: añadiría rechazos nuevos a composiciones aceptadas (un ADDED con un ID que ya está bajo otro título), contra REQ-30.22; esa comprobación vive aguas arriba.

### D-06. ID sintético de `livingdoc` fuera del espacio válido

**Elección.** `reqid.Synthetic(domain string, ordinal int) string` devuelve `REQ-<domain>-<ordinal>` cuando `domain` contiene al menos una letra minúscula ASCII (todos los casos reales: las 194 entradas sintéticas de `openspec/INDEX.md` quedan idénticas) y `req-<domain>-<ordinal>` en otro caso.

**Garantía.** Por D-01 (reglas 2 y 3) ningún ID válido de ningún patrón declarable contiene minúsculas; todo sintético contiene al menos una. La separación de ambos espacios es por construcción, sin depender del patrón del proyecto. Una prueba de fuzz lo comprueba contra un conjunto de plantillas.

La lectura laxa (`Pattern.TrailingCandidate`) generaliza `\((REQ-[^)]+)\)\s*$` a `\(<literal inicial>[^)]+\)\s*$`: sigue sin validar, como hoy.

**Alternativa descartada.** Un carácter marcador en todos los sintéticos (`auth#1`): reescribe las 194 líneas de `INDEX.md` en la siguiente sincronización sin ningún beneficio para los dominios reales.

### D-07. ISO con el patrón inaccesible

Con `SDDErr` o `ParseErr` el proyecto ya proyecta `iso.project: invalid` y el archivado ya lleva `iso_declaration_invalid`. Para no evaluar con un patrón que el proyecto no declaró: la etapa `specs` de un cambio ISO lleva ese mismo hallazgo (código `iso_declaration_invalid`, etapa `specs`) en lugar de las reglas de identificador; `tasks` no se evalúa; `evaluateISOArchive` no calcula escenarios afectados ni matriz (la proyección `iso.matrix` se omite). Ningún código nuevo de motivo. `sdd trace` y `archive-check` terminan con error y la continuación de `axiom workspace validate`.

### D-08. Constante `Motivo:` compartida y prueba de paridad

`isoevidence.NoImpactReasonLabel = "Motivo:"` (exportada junto a `NoImpactMark`). La usan `isoevidence.ParseBugDeclaration` (hoy el literal de `bug.go:70` y los textos de `Missing`) y `livespec` (`impactReasonPrefix = isoevidence.NoImpactReasonLabel`). Prueba tabular `TestImpactParityWithISO` en `livespec`: para cada cuerpo de sección construye `## Informe de bug\nObservado: x\nEsperado: y\n## Impacto en la spec\n<cuerpo>` para `ParseBugDeclaration` y un documento ODD con `Tipo: bug` y la misma sección para `ParseFeatureDoc`, y compara: marcador presente (ISO: `Missing` sin la línea de `NoImpactMark`; ODD: `Defect != ImpactMarkerMissing`), validez y motivo (`Motivo` frente a `Impact.Reason`). Casos: completo; sin marcador; sin `Motivo:`; `Motivo:` vacío seguido de otro con texto (cuenta el primero); sangría con espacios y tabulador; espacios finales; marcador y `Motivo:` dentro de una valla; marcador con texto adicional.

### D-09. Guías y documentación

Se editan, conservando todos los literales que fijan las pruebas (`Cubre: REQ-x[, REQ-y ...]`, tokens de la tabla de motivos):

| Fichero | Cambio |
|---|---|
| `internal/assets/skills/_shared/sdd-iso-convention.md:18` | «exactamente un identificador válido según el patrón del proyecto (`sdd.id_pattern` de `axiom.yaml`; sin él, `REQ-<n>.<m>`)»; el mal formado es «un token que empieza como el patrón y no encaja con él, p. ej. `REQ-30`» |
| `internal/assets/skills/sdd-spec/SKILL.md:257` | mismo criterio |
| `internal/assets/skills/_shared/sdd-status-contract.md:153` | fila `iso_requirement_id`: «add or fix an identifier valid for the project's `sdd.id_pattern` (default `REQ-<n>.<m>`)» |
| `internal/assets/skills/sdd-archive/SKILL.md:151` | el compositor empareja por identificador cuando el nombre de la delta lleva uno válido, y por nombre si no |
| `docs/manual-de-inicio.md:408,547,625,630` | regla ISO, numeración común y ADDED de ODD según `sdd.id_pattern`; subapartado nuevo «Patrón de identificadores de requisito (`sdd.id_pattern`)» dentro de «Modo ISO y Perfil de Bug (`sdd:`)», sin renombrar ese encabezado (lo fija `example_config_test.go:89-90`) |
| `axiom.example.yaml:79-92` | cabecera «Bloque SDD: modo ISO y patrón de identificadores (opcional)», documentación de la gramática y `#   id_pattern: "RF-<DOM>-<NN>"` dentro del bloque comentado |

**No se tocan** (REQ-30.31 y C-03-c): `sdd-spec/references/openspec-adoption.md`, `docs/manual-de-inicio.md:563` y la ayuda de reglas de `adopt check`, cuyo texto se genera ahora con `reqid.Default()` y queda idéntico.

**`internal/components/agentguidance/routing.go`: este diseño no necesita tocarlo.** En `main` no contiene `REQ-<n>.<m>` ni `REQ-n.m` (solo citas `REQ-2x.y` en comentarios y prosa de carril), y la regla del ID no forma parte del texto de enrutado. Tampoco cambia ningún golden de `testdata/golden/` (ningún fichero fija ese texto y las skills no se materializan en goldens). Así no hay solape con la PR #289 (C-02-d).

**Tablas fuera de bloques de requisito** (`sdd-iso-evidence/spec.md:17,37`, `odd-living-spec-integration/spec.md:30`): no se editan. Una delta no puede tocarlas y, con el defecto `REQ-<n>.<m>`, siguen siendo ciertas. Las reescribe C-03-e al cambiar el defecto.

## 3. Flujo de datos

```
axiom.yaml ── decodeSDD ── reqid.Parse ──► SDDConfig.IDPattern (o SDDErr)
                                               │
                      Declarations.IDPattern() ▼ (Default si no hay clave)
   ┌───────────────┬───────────────┬────────────┴───┬────────────────┬──────────────┐
   ▼               ▼               ▼                ▼                ▼              ▼
archive integrate  sdd-archive-   sddstatus (ISO)   sdd trace /      livingdoc      (adoptcheck:
  RegistryOptions  compose        isoInputs.pattern archive-check    Service        reqid.Default)
   │ .Pattern       │ ComposeOpts    │                  │              │
   ▼                ▼ .IDPattern     ▼                  ▼              ▼
livespec.Registry  openspecdoc.   isoevidence.(…, p)  isoevidence/   Indexer.WithPattern
 (InTitle, Series,  Compose ──► Resolve ◄── livespec.classify,  openspecdoc.Resolve
  Format)                         archive-check.alreadyComposed
```

## 4. Cambios de ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/reqid/reqid.go`, `template.go`, `series.go`, `synthetic.go` | Crear | `Pattern`, `Parse`, `Default`, compilación, notaciones, `Series`, `Synthetic`, `TemplateError` |
| `internal/reqid/*_test.go` | Crear | tablas, equivalencia con las regex heredadas, fuzz, importaciones |
| `internal/workspace/sdd.go` | Modificar | `sddKeys = {"iso", "id_pattern"}`, `sddPattern`, `SDDConfig.IDPattern`, mensaje del nodo no mapeo |
| `internal/workspace/declarations.go` | Modificar | `func (d Declarations) IDPattern() (reqid.Pattern, error)` |
| `internal/openspecdoc/parse.go` | Modificar | fuera `Requirement.ID`, `RequirementID` y `requirementIDInTitle` |
| `internal/openspecdoc/compose.go`, `resolve.go` (nuevo) | Modificar/Crear | `ComposeOptions.IDPattern`, `Resolve`, `Match`, `SupersededSuffix`, `IsSuperseded` |
| `internal/isoevidence/reqid.go`, `cubre.go`, `bug.go`, `imports_test.go` | Modificar | sin regex; `p` en seis funciones; continuaciones con `p.Template()`; `NoImpactReasonLabel`; admite `internal/reqid` |
| `internal/sddstatus/iso.go`, `status.go` | Modificar | patrón en `isoInputs`; D-07 |
| `internal/livespec/registry.go`, `plan.go`, `odddoc.go` | Modificar | `RegistryOptions.Pattern`, `Registry.Pattern`, series `reqid.Series`, fuera `SeriesOf`; `classify` con `Resolve`; mensajes con `Compact()`; constante `Motivo:` |
| `internal/livespec/impact_parity_test.go` | Crear | D-08 |
| `internal/livingdoc/indexer.go`, `service.go` | Modificar | sin regex de ID; `WithPattern`; `reqid.Synthetic`; aviso con `SDDErr` |
| `internal/adoption/adoptcheck/check.go` | Modificar (mínimo) | `legacy := reqid.Default()`; `InTitle`, `Series`, `Key`, `Compare`; textos idénticos |
| `internal/cli/archive_integrate.go`, `archive_adopt.go`, `sdd_archive_compose.go`, `sdd_archive_check.go`, `sdd_trace.go` | Modificar | patrón del workspace; ayudas sin literal de forma |
| `internal/cli/reqid_rf_e2e_test.go` | Crear | proyecto `RF-<DOM>-<NN>` (§7) |
| `internal/reqid/arch_test.go` | Crear | guarda de §8 |
| guías, manual y `axiom.example.yaml` (D-09) y sus pruebas (`example_yaml_test.go`, `example_config_test.go`, `sdd_test.go`) | Modificar | documentación de la clave |

## 5. Interfaces y contratos

### 5.1 `internal/reqid`

```go
package reqid

// Pattern es una plantilla de identificador compilada e inmutable. El valor cero
// se comporta exactamente como Default().
type Pattern struct{ c *compiled }

func Default() Pattern                       // plantilla "REQ-<n>.<m>"
func Parse(template string) (Pattern, error) // *TemplateError

func (p Pattern) Template() string      // "REQ-<n>.<m>", "RF-<DOM>-<NN>"
func (p Pattern) Compact() string       // "REQ-n.m", "RF-DOM-NN"
func (p Pattern) SeriesCompact() string // "REQ-n.*", "RF-DOM-*"

func (p Pattern) Valid(id string) bool
// InTitle devuelve el último "(ID)" válido del título, o "".
func (p Pattern) InTitle(title string) string
// Scan devuelve los candidatos del texto, válidos y mal formados, sin repetidos
// y en orden de aparición.
func (p Pattern) Scan(text string) (valid, malformed []string)
// TrailingCandidate es la lectura laxa de livingdoc: "(<literal inicial>…)" al
// final del título, sin validar.
func (p Pattern) TrailingCandidate(title string) (rest, id string, ok bool)

func (p Pattern) Series(id string) (Series, bool)
func (p Pattern) Number(id string) (int, bool)
// Format compone el ID de número n (n >= 0) en una serie obtenida de este patrón.
func (p Pattern) Format(s Series, n int) string

// Series es comparable y sirve de clave de mapa.
type Series struct{ prefix, suffix, key string }
func (s Series) Key() string           // "30", "USU", ""
func (s Series) String() string        // "REQ-30.*"
func (s Series) Compare(o Series) int  // orden natural: tramos de cifras por valor

// Synthetic es el ID informativo de livingdoc; nunca es válido para ningún patrón.
func Synthetic(domain string, ordinal int) string

type TemplateError struct{ Template, Reason string }
func (e *TemplateError) Error() string // `la plantilla "<t>" no es válida: <motivo>`
```

### 5.2 `workspace` y `openspecdoc`

```go
type SDDConfig struct {
	ISO       bool
	IDPattern *reqid.Pattern // nil: no declarada
}
func (d Declarations) IDPattern() (reqid.Pattern, error)

type ComposeOptions struct {
	SupersedeRemoved   bool
	ExplicitSuperseded []string
	IDPattern          reqid.Pattern // valor cero = defecto
}
type Match int
const (MatchNone Match = iota; MatchFound; MatchAmbiguous)
const SupersededSuffix = " [SUPERSEDED / DEPRECADO]"
func IsSuperseded(name string) bool
// Resolve devuelve el índice, en names (canónica en orden), del requisito al que
// se refiere el nombre de una delta (REQ-30.22).
func Resolve(names []string, name string, p reqid.Pattern) (int, Match)
```

`Requirement` pierde `ID`; `ParseDelta`, `ParseCanonical`, `Compose` y `RenderNewSpec` conservan su firma.

### 5.3 `isoevidence`, `livespec` y `livingdoc`

```go
func RequirementIDs(name string, p reqid.Pattern) (valid, malformed []string)
func CheckRequirementIDs(delta, baseline []Requirement, p reqid.Pattern) []Finding
func NewBaselineIndex(baseline []Requirement, p reqid.Pattern) BaselineIndex
func CheckCoverage(delta []Requirement, index BaselineIndex, cubre []CubreLine, p reqid.Pattern) []Finding
func CheckExistingRequirementBug(index BaselineIndex, cubre []CubreLine, p reqid.Pattern) []Finding
func AffectedFromDelta(delta []Requirement, p reqid.Pattern) []ScenarioKey
const NoImpactReasonLabel = "Motivo:"

type RegistryOptions struct { ExcludeChange string; SpecsOnly bool; Pattern reqid.Pattern }
func (r *Registry) Pattern() reqid.Pattern
func (r *Registry) NextFree(s reqid.Series) string
func (r *Registry) SeriesRefs(s reqid.Series) []IDRef
// SeriesOf (paquete) desaparece: r.Pattern().Series(id).

func (idx *Indexer) WithPattern(p reqid.Pattern) *Indexer // copia
```

Los `Finding` conservan `Code`, `Stage`, `Subject`, `Detail` y `Continuation`; solo la notación del patrón entra en el texto, idéntico con el defecto. `CollisionError{ID, Where, NextFree}`, `DocError` con `added_without_id` y `DivergentRequirementError{ID}` no cambian de forma (`ID` sale de `InTitle`).

## 6. Estrategia de pruebas

| Capa | Qué | Cómo |
|---|---|---|
| Unidad `reqid` | `Parse`: cada regla de D-01 con su motivo; plantillas válidas | tabla `t.Run` |
| Unidad `reqid` | `Valid`, `InTitle`, `Scan`, `Series`, `Number`, `Format`, `Compact`, `SeriesCompact`, `Compare` para `REQ-<n>.<m>`, `RF-<DOM>-<NN>`, `RF-<m>`, `HU-<NNN>`; ida y vuelta `Format(Series(id), Number(id)) == id` en las formas canónicas | tabla |
| Equivalencia del defecto | las regex heredadas (copiadas en la prueba) frente a `Default()` sobre cada título de `openspec/specs/*/spec.md` y una batería de bordes (`REQ-30`, `REQ-30.1,`, `REQ-30.1.`, `REQ-x.1`, `REQ-030.01`, `(REQ-1.1) (REQ-2.2)`) | tabla y corpus |
| Fuzz | `Parse` no entra en pánico; `Synthetic` nunca es válido para un conjunto de plantillas | `FuzzParse`, `FuzzSynthetic` (semillas en el corpus, se ejecutan con `go test`) |
| Unidad `workspace` | `id_pattern` ausente, válido, inválido, no texto, vacío, repetido, con `iso`; `IDPattern()` defecto/declarado/error; carga estricta y `validate`; plantilla del ejemplo | tablas en `sdd_test.go`, `example_*_test.go` |
| Unidad `openspecdoc` | MODIFIED con título cambiado y mismo ID; REMOVED por ID con título obsoleto (con y sin `SupersedeRemoved`); RENAMED con nombre antiguo por ID; ID repetido con y sin título exacto; superados excluidos; ADDED y colisión de RENAMED por título; títulos sin ID; las mismas tablas con `RF-<DOM>-<NN>` | tabla |
| No regresión de composición | para cada spec viva y cada requisito, MODIFIED identidad → salida igual a la entrada; los 27 usos actuales de `Compose` en pruebas, sin cambios de expectativa | corpus |
| `isoevidence` / `sddstatus` | pruebas actuales con `reqid.Default()` y textos sin cambios; con `RF-<DOM>-<NN>`: válido, mal formado (`RF-USU-1`), continuación con `RF-<DOM>-<NN>`; patrón inaccesible → D-07 | tabla |
| `livespec` | registro con `RF-<DOM>-<NN>`: `NextFree` por serie con relleno (`RF-USU-03`), colisión con el siguiente libre saltando los usados en el documento, `classify` de REMOVED y MODIFIED por ID e idempotencia; paridad de D-08 | tabla con `t.TempDir()` |
| `livingdoc` | lectura laxa con RF; sintético idéntico al actual para dominios con minúsculas; aviso con `SDDErr` | tabla |
| `adoptcheck` | pruebas actuales sin cambios (textos `(REQ-n.m)`, `REQ-n.*`, `serie 40`) | existentes |
| Extremo a extremo (en proceso) | proyecto temporal con `sdd: {id_pattern: "RF-<DOM>-<NN>"}`: (1) `archive integrate --odd` con un ADDED `RF-USU-01` en una capacidad nueva; (2) un segundo documento que reutiliza `RF-USU-01` se rechaza proponiendo `RF-USU-02`; (3) `sdd-archive-compose --cwd` de un MODIFIED que cambia el título y conserva `RF-USU-01`; (4) `archive-check` de ese dominio da `alreadyComposed` falso antes y verdadero después; (5) `sdd trace` encadena `RF-USU-01` | llamadas a los ejecutores de `internal/cli` con `t.TempDir()` y repositorio git temporal, como `archive_integrate_test.go`; nunca un binario compilado |

Comprobación final: `go test ./...` (fallo conocido de `internal/components/filemerge` en Windows) y `go vet ./...`.

## 7. Matriz de amenazas

N/A: no hay enrutado, órdenes de shell, subprocesos, automatización de VCS/PR, clasificación de ejecutables ni integración de procesos. La plantilla declarada no es una regex: se compila con literales escapados (`regexp.QuoteMeta`) y clases fijas, y RE2 es lineal, así que no hay riesgo de ReDoS ni de inyección de patrón.

## 8. Criterio de terminado ejecutable

El criterio literal del plan (`rg 'REQ-' --type go internal cmd -g '!*_test.go'` solo en el puerto, el defecto y comentarios) **no se puede cumplir sin salir del alcance**: en `main` hay unas 25 citas de requisitos de Axiom en mensajes (`(REQ-21.18)`, `(REQ-24.36)`, …) y la plantilla de borrador `### Requirement: … (REQ-01)` de `internal/knowledge/sweep.go:266`, que no son lógica de formato. Decisión del usuario (§10, 2026-11-23): el criterio verificable es «ninguna forma de ID fuera de `internal/reqid`», es decir, `rg -n 'REQ-[^0-9]' --type go internal cmd -g '!*_test.go'` solo coincide en `internal/reqid` y en comentarios. Hoy coincide exactamente en las 19 líneas del alcance (adoptcheck, cli, openspecdoc, livespec, livingdoc, isoevidence). Una prueba de arquitectura, `TestNoRequirementIDFormatOutsideReqid`, recorre con `go/scanner` los literales de texto de los `.go` de producción de `internal/` y `cmd/` y falla si alguno contiene `REQ-` seguido de algo que no sea una cifra fuera de `internal/reqid`; los comentarios quedan fuera por construcción.

## 9. Migración y despliegue

Sin migración de datos: las specs de Axiom siguen en `REQ-n.m` y el defecto las admite. `openspec/INDEX.md` no cambia. Reversión: retirar `sdd.id_pattern` de `axiom.yaml` antes de volver a un binario anterior, porque `sdd:` es de lista cerrada (ningún proyecto la declara hoy).

**Plan de commits (una PR, `single-pr`; cada commit compila y pasa sus pruebas):**

1. `internal/reqid` con pruebas, fuzz y guarda de importaciones.
2. `sdd.id_pattern` en `workspace`, `IDPattern()`, `axiom.example.yaml` y sus pruebas.
3. `openspecdoc`: fuera `Requirement.ID`, `Resolve` y compositor por ID con su corpus de no regresión; los usos de `.ID` en `livespec` y `adoptcheck` pasan a `reqid.Default().InTitle` (mismo comportamiento).
4. `isoevidence`, `sddstatus` (D-07), `archive-check` y `sdd trace`.
5. `livespec` con el patrón del registro, `classify` con `Resolve`, `adoptcheck` con `reqid.Default()` y ayudas de la CLI.
6. `livingdoc`.
7. `NoImpactReasonLabel` y prueba de paridad.
8. Guías, manual y skill de archivado.
9. Prueba extremo a extremo `RF-<DOM>-<NN>` y prueba de arquitectura de §8.

## 10. Preguntas abiertas

- [x] **Resuelta por el usuario (2026-11-23):** interpretación del criterio de terminado (§8). Se acepta `REQ-[^0-9]` más la prueba de arquitectura `TestNoRequirementIDFormatOutsideReqid`; las citas `REQ-<cifras>` en mensajes y el borrador de `knowledge/sweep.go` quedan para C-03-e.

## 11. Contratos que la spec debe recoger

- REQ-44 (capacidad `requirement-identifiers`): clave `sdd.id_pattern`, gramática de D-01 (marcadores, literal inicial, alfabeto, reglas de adyacencia), defecto `REQ-<n>.<m>`, válido/candidato/mal formado, serie y siguiente libre, forma canónica de `<N…>`, error de carga y de `validate`, lectura en vivo, y que el sintético de `livingdoc` nunca es un ID válido.
- REQ-30.22: resolución de D-05 (ID con vuelta al título exacto, superados fuera, ADDED y colisión de RENAMED por título, misma salida para toda composición hoy aceptada).
- REQ-29.3/29.16: la etapa `specs` con el patrón inaccesible lleva `iso_declaration_invalid` (D-07).
- REQ-30.4: el siguiente libre usa la serie y el relleno del patrón.
- Confirmar en `main`, antes de archivar, que REQ-44 sigue libre (C-02-b y otras olas).

## 12. Riesgos de diseño

| Riesgo | Mitigación |
|---|---|
| Unos 25 sitios de prueba cambian de firma (`isoevidence`, `NextFree`, `.ID`) | Mecánico con `reqid.Default()`; el corpus de equivalencia prueba que el defecto no cambia |
| `archive integrate` y `sdd-archive-compose` se niegan ahora con cualquier `SDDErr`, aunque el error sea de `iso` | Fallo cerrado deliberado: sin bloque válido no se conoce el patrón; el mensaje lleva la continuación de `validate` |
| La regla del literal inicial con separador deja fuera IDs de cliente como `RF120` | Suficiente para D-03-3 y D-03-8 (`RF-120`); C-03-c puede relajarla junto a los esquemas mixtos |
| `adoptcheck` sigue en `REQ-n.m` aunque el proyecto declare otro patrón | Es lo que pide REQ-30.31/A-03-2 hasta C-03-c; C-03-c debe pasar a `IDPattern()` antes de que C-03-e cambie el defecto |
| Resolución por ID de REMOVED y del nombre antiguo de RENAMED: un título obsoleto en la delta actúa sobre el requisito del mismo ID | Es la identidad por ID que pide el cambio; `archive-check` y `livespec.classify` usan el mismo `Resolve` |
| Superficies compartidas (`axiom.yaml`, guías, `livespec`, `openspecdoc`, `isoevidence`) con C-03-b, C-06-e y C-06-c | Cambios acotados por commit; C-03-b rebasa sobre este; C-06-e registra `sdd.id_pattern` en su catálogo |
