# Diseño: Migración de las specs propias de Axiom de `REQ-n.m` a `RF-<DOM>-<NN>` (C-03-e)

> **Cambio:** `migrar-ids-rf`. **Idioma:** castellano; identificadores de código en inglés.
> **Entradas:** `proposal.md`, deltas `specs/requirement-trace/spec.md` (prov:TRC:1 → `RF-TRC-16`, prov:TRC:2 → `RF-TRC-17`, MODIFIED `RF-TRC-02` y `RF-TRC-04`) y `specs/requirement-identifiers/spec.md` (prov:RID:1 → `RF-RID-16`), Engram `sdd/migrar-ids-rf/explore` y `sdd/migrar-ids-rf/dominios`.
> **Contexto técnico mantenido:** `axiom workspace context` devuelve `technicalContext: null`; no hay delta de contexto ni sección de impacto.

## Enfoque técnico

El cambio tiene dos naturalezas y el diseño las separa físicamente:

1. **Comportamiento permanente de producto** (alcanzable desde `./cmd/axiom`, por tanto compatible con el trinquete de código muerto): un paquete nuevo `internal/legacyid` que carga y valida `openspec/id-map.yaml` (RF-TRC-16/17), una extensión de `internal/reqtrace` con resolución de IDs heredados acotada por capacidad (RF-TRC-02/04) y su cableado en `internal/cli/trace.go`. Más la corrección de `openspecdoc.ParseDelta` (vallas).
2. **Herramienta de un solo uso** (generación del mapa y reescritura): vive **solo en código de prueba**, en un paquete nuevo sin ficheros de producción, `internal/idmigration`, y se ejecuta en proceso con `go test` y una variable de entorno explícita. No hay binario, no hay verbo nuevo y no hay código de producción sin llamadas. El mismo paquete aloja las pruebas permanentes del criterio de terminado (residuos y coherencia mapa/specs), que corren siempre en la CI.

La reescritura es **mecánica y reproducible**: dado el árbol anterior a la migración y las dos entradas curadas (`residues.yaml` y `contexts.yaml`), ejecutar la herramienta produce exactamente el commit de reescritura. Ante la duda la herramienta **conserva y obliga a declarar**; nunca reescribe hacia una capacidad adivinada.

### Medidas que fijan el diseño (árbol `f596a985`, fuera de `openspec/changes/archive/`)

| Medida | Valor | Consecuencia |
|---|---|---|
| Ficheros versionados | 2047 | La prueba de residuos recorre `git ls-files`, no el disco |
| Apariciones de `REQ-\d+\.\d+` | 6633 (incluye `docs/archive`, `docs/audits` y la carpeta del cambio) | Volumen de la reescritura |
| Títulos `### Requirement:` | 513: 462 con `(REQ-n.m)` (4 no finales: `REQ-21.14`, `REQ-20.12`, `REQ-14.4`, `REQ-2.3`, seguidos de `[SUPERSEDED / DEPRECADO]` o `— RETIRADO`), 1 con `(REQ-27.2b)`, 2 con dos IDs `(REQ-10.1 & REQ-10.2)` y `(REQ-10.3 & REQ-10.4)`, 48 sin ID | Regla de títulos (D6) y `old` como lista (D2) |
| Tokens distintos en títulos | 426 | Claves del mapa |
| Tokens ambiguos (en títulos de ≥2 capacidades) | 12: `REQ-1.1`–`1.4`, `2.1`–`2.3`, `3.1`–`3.3`, `4.1`, `4.2` (cada uno con su propio conjunto de capacidades, no siempre las siete) | El generador calcula la ambigüedad; no se codifica a mano |
| Ambiguos fuera de specs, pruebas e historia | 39: Go 12 (todos comentarios: `internal/dashboard/*`, `internal/hub/init.go`, `internal/workspace/*`, `internal/multirole/roster.go`, `internal/reqtrace/classify.go:104`), ODD 24 (casi todos títulos dentro de `### Capacidad:`), docs 2, JS 1 | Tabla de contexto por ruta y token pequeña (D3) |
| Tokens en literales de cadena de Go de producción | 21 en 10 ficheros; uno retirado: `REQ-24.62` en `internal/techcontext/plan.go:137` | Arreglo manual previo (D13) |
| Tramos de código en línea con `REQ-n.m` en specs | 248 en 8 specs, todos ejemplos del patrón por defecto (`Cobro (REQ-30.1)` en `pagos`, `(REQ-31.1)` en `facturacion`, `REQ-40.1`…); 0 en bloques vallados | En specs el código en línea se conserva (D4) |
| Tokens que no son título de ningún requisito vivo | ~115 distintos (retirados: familia `REQ-24.38`–`REQ-24.62` de la baseline, `REQ-15.2`, `REQ-12.3`…; ejemplos: `REQ-30.99`, `REQ-31.1`, `REQ-999.1`…) | Nunca se inventa un ID (D13) |
| Continuaciones sin prefijo | ~8 fuera de historia (`REQ-21.5/21.6`, `REQ-21.8-21.12`, `REQ-24.26 y 24.27`, `REQ-30.9 a 30.14`…) | Regla de rangos (D5) |

## Decisiones de arquitectura

### D1. Forma de la herramienta: paquete solo de pruebas con ejecución por variable de entorno

**Elección:** `internal/idmigration/` contiene únicamente ficheros `_test.go` (`package idmigration_test`), como ya hacen `internal/components` y `e2e/organicruntime`. La herramienta se ejecuta con:

```
AXIOM_IDMIGRATION=map    go test ./internal/idmigration -run TestMigrateRepository -count=1
AXIOM_IDMIGRATION=plan   go test ./internal/idmigration -run TestMigrateRepository -count=1 -v
AXIOM_IDMIGRATION=write  go test ./internal/idmigration -run TestMigrateRepository -count=1
```

- `map`: genera `openspec/id-map.yaml` desde las specs actuales. Falla si el fichero ya existe (se regenera borrándolo de forma explícita) o si algún título ya tiene un ID que no es `REQ-n.m`.
- `plan`: calcula toda la reescritura en memoria sin escribir y publica con `t.Log` un informe por fichero (sustituciones, conservados por residuo, no mapeados, ambiguos sin resolver) y una propuesta de entradas de `residues.yaml` para lo que queda. Falla si queda algo sin resolver.
- `write`: las mismas comprobaciones que `plan` y, si todo resuelve, escribe. Sin la variable, `TestMigrateRepository` hace `t.Skip`.

Las funciones de producción que necesita la herramienta (`openspecdoc.ParseCanonical`, `reqid.NewIDMap`, `reqid.Pattern.Format`, `livingdoc.NewService(...).Sync`, `legacyid.Load`) ya son alcanzables desde `./cmd/axiom`; la herramienta no añade ninguna función de producción.

**Alternativas consideradas:**
- Verbo oculto `axiom dev migrate-ids`: sería alcanzable y no rompería el trinquete, pero deja superficie de CLI permanente sin requisito, y ejecutarlo exige compilar el binario (bloqueado por el EDR) o invocarlo en proceso desde una prueba, que es esta misma solución con más código de producción.
- `go run ./cmd/...` o un `main` auxiliar: compila y ejecuta un binario temporal; incompatible con Cortex XDR.
- Script `perl`/`sed`: sin límites de token (`REQ-1.1` frente a `REQ-1.10`), sin contexto de capacidad, sin `go/scanner` para distinguir comentarios de literales y no reutiliza `reqid.IDMap`.
- Flag de prueba `-migrate`: un flag propio solo existe en su paquete; `go test ./... -migrate` falla en los demás. La variable de entorno no tiene ese problema.

**Justificación:** el trinquete (`scripts/deadcode-ratchet.sh`) mide la alcanzabilidad desde `./cmd/axiom`; el código de prueba no cuenta. La herramienta queda versionada (auditable y reaplicable tras rebasar) sin coste de producto.

### D2. Mapa: `openspec/id-map.yaml`, esquema `axiom.id-map/v1`, entradas planas y orden estable

**Elección:** ruta `<planning home>/id-map.yaml`, que en Axiom es `openspec/id-map.yaml` (RF-TRC-16). Formato:

```yaml
# Mapa de identificadores heredados de Axiom (C-03-e). Lo genera internal/idmigration; no se edita a mano.
schema: axiom.id-map/v1
entries:
  - capability: antigravity-support
    title: "Antigravity Adapter Registration"
    new: RF-AGV-01
  - capability: autoskills
    old: [REQ-1.1]
    new: RF-ASK-01
  - capability: persona-behavior-contract
    old: [REQ-10.1, REQ-10.2]
    new: RF-PER-08
```

- Una entrada por título `### Requirement:` vigente (513). `old` es una secuencia en estilo flujo: vacía/ausente en los 48 requisitos sin ID (llevan `title`, el título original sin ID), con un elemento en 462 + 1 (`REQ-27.2b`) y con dos en los dos títulos de `persona-behavior-contract` que citan dos IDs.
- Orden: capacidad por orden de bytes del nombre de carpeta; dentro de ella, orden de aparición (que coincide con el número nuevo). Numeración con `reqid.Pattern.Format(serie, n)` del patrón `RF-<DOM>-<NN>` y la tabla aprobada de 49 códigos (`internal/idmigration/domains_test.go`).
- Serialización determinista: emisor propio sobre `yaml.v3` (escalares con `yaml.Marshal`, sin plegado de líneas), cabecera fija, LF y salto final. La prueba de ida y vuelta exige `legacyid.Load(Generate(x))` sin error.
- Lectura estricta en `legacyid.Load` (`KnownFields`), con los invariantes de RF-TRC-16 y el error tipado de RF-TRC-17 (D8).

**Alternativas consideradas:** anidar por capacidad (más compacto, pero la spec pide que cada entrada lleve su capacidad y el diff de una entrada queda menos legible); JSON (sin comentario de cabecera); CSV (el título puede contener comas y comillas); embeber el mapa en Go (rompe el contrato: `axiom trace` debe leer el mapa del proyecto, no uno compilado).

**Justificación:** el par (capacidad, ID antiguo) es la clave natural de la ambigüedad medida; `old` como lista cubre sin casos especiales los dos títulos con dos IDs (la spec dice «el ID antiguo o su ausencia»; la lista de cero o más elementos es su forma general, véase Preguntas abiertas).

### D3. Resolución de un ID antiguo ambiguo: cadena de contexto y fallo sin escribir

**Elección:** para cada aparición de un token `T`, `cands(T)` son las capacidades con `T` en el mapa.

1. `|cands| = 1` → esa capacidad.
2. **Título** `### Requirement:` en una spec viva → la capacidad de la carpeta; en una delta dentro de `## Spec viva` de un documento ODD → la de su `### Capacidad:`. Contexto estricto: si (capacidad, `T`) no está en el mapa, el token no se reescribe y se informa.
3. La **misma línea** nombra exactamente una capacidad de `cands(T)` (entre comillas invertidas o como palabra completa) → esa.
4. **Fichero**: en una spec, su propia capacidad si está en `cands(T)`; en un documento ODD, la única capacidad de `cands(T)` entre sus `### Capacidad:`.
5. **Tabla de contexto** curada `internal/idmigration/contexts.yaml`: entradas `{path, token, capability, reason}`, con `path` exacto o patrón glob; solo se aplica si `capability ∈ cands(T)`. El motivo cita la evidencia (p. ej. la delta archivada que tocó ese ID en esa capacidad).
6. Si nada resuelve → **fallo** de `plan`/`write` sin escribir ningún fichero, con la lista `ruta:línea token [candidatas]`.

**Alternativas consideradas:** resolver por la capacidad «más probable» (riesgo alto de reescritura equivocada); declarar todos los ambiguos fuera de specs como residuo (oculta citas propias en código de producción, que la spec prohíbe conservar en mensajes y desaconseja en comentarios).

**Justificación:** fuera de specs y pruebas hay solo 39 apariciones ambiguas; las 24 de ODD resuelven por la regla 2 o 4; las 12 de Go son comentarios con un patrón de ruta claro (`internal/dashboard` → `local-web-dashboard` o `multi-project-hub`, `internal/hub` → `multi-project-hub`, `internal/workspace` → `workspace-topology`, `internal/multirole` → `multi-role-fan-out`) que se fijan con unas 10 entradas verificadas contra el archivo. Algunas citas de `internal/dashboard` pueden ser numeración local de un cambio antiguo: la tabla obliga a decidirlo con evidencia o a declararlas residuo `legacy-citation`.

### D4. Reglas de reescritura por tipo de fichero

Regla base: **se reescribe toda aparición de un token del mapa salvo que la cubra una entrada de `residues.yaml` o una regla automática de conservación de esta tabla**. La sustitución usa `reqid.IDMap.Substitute` (token completo, el más largo primero, una sola pasada) sobre cada tramo reescribible, con un `IDMap` por línea cuando la línea tiene ambiguos resueltos por contexto y el `IDMap` global de los no ambiguos en el resto.

| Destino | Se reescribe | Se conserva automáticamente | Notas |
|---|---|---|---|
| `openspec/specs/*/spec.md` | Títulos (D6), H1, prosa, líneas `Cubre:` | Tramos de código en línea y bloques vallados (ejemplos del patrón por defecto; verificado: 248 tramos, todos ejemplos) | Contexto de fichero = capacidad propia |
| `openspec/INDEX.md` | No se reescribe: se **regenera** (D10) | — | Sus restos (nombres de escenario con ejemplos) se declaran residuo `md-code` |
| `openspec/id-map.yaml`, `openspec/changes/archive/**`, `openspec/changes/migrar-ids-rf/**` | Nunca | Todo | La carpeta del cambio es residuo declarado (D12) |
| Go de producción (`*.go` sin `_test`) | Comentarios y literales de cadena | Nada | Un token no mapeado en un literal es fallo (nunca residuo: RF-RID-16). Tras escribir, `go/format.Source` en proceso; si falla, aborta |
| Go de pruebas (`*_test.go`) | Comentarios cuyo token no aparece en un literal de cadena **no fijado** de ningún fichero de prueba del mismo paquete; literales **fijados** | Literales de fixtures y comentarios que los describen | Un literal está **fijado** (pin de un mensaje de producción) si su token se reescribe en esta ejecución en algún fichero no de prueba bajo el directorio del paquete (literal de Go o asset embebido, p. ej. `internal/assets/skills/**`). Tokenización con `go/scanner` (`ScanComments`) |
| `testdata/**` | Nunca | Todo | Fixtures; los goldens de `internal/components/testdata/golden/**` se regeneran con `-update` (D10) |
| `odd/tasks/*.md` | Todo fuera de vallas, incluidas `Cubre:` y prosa; dentro de vallas ` ```openspec ` de `## Spec viva`, la regla de specs (títulos y prosa; código en línea conservado) con la capacidad de su `### Capacidad:` | Vallas que no son `openspec`; vallas `openspec` de una capacidad que ya no existe (p. ej. `knowledge-query-engine`, `semantic-code`): se conservan enteras | Ningún documento tiene hoy `## Identificadores asignados`; si lo tuviera, sus `definitive` y `previous` se reescriben con la capacidad de cada entrada |
| `docs/**` vivo, `internal/assets/**/*.md`, `skills/**`, `axiom.yaml`, `axiom.example.yaml`, `.github/workflows/*.yml`, `.goreleaser.yaml`, JS/CSS del dashboard | Todo | Nada automático: los ejemplos (`docs/manual-de-inicio.md:416`, `axiom.example.yaml:94`, `internal/assets/skills/_shared/sdd-iso-convention.md:18`) y las citas deliberadas (`axiom trace REQ-24.3` del plan) se declaran en `residues.yaml` | `axiom.yaml`: además, la herramienta inserta el bloque `sdd:` (D10) |
| `docs/archive/**`, `docs/audits/**` | Nunca | Todo | Residuo `dated-history` |

**Alternativas consideradas:** clasificar a mano los ~600 ficheros (no reproducible); reescribir todo y revisar el diff (los fixtures del patrón por defecto se romperían en silencio o con fallos opacos); regla «código en línea = ejemplo» en todos los Markdown (falsa en ODD: `odd/tasks/inc-22-updater-and-skills.md:50` cita `` `REQ-22.1` `` como requisito propio).

**Justificación:** cada regla es comprobable por máquina y está respaldada por una medida; donde no hay regla segura, decide un humano en `residues.yaml` o `contexts.yaml`, que quedan versionados y revisables.

### D5. Rangos y continuaciones sin prefijo

**Elección:** un paso previo detecta, tras un token `REQ-a.b`, la secuencia `sep a.c` con `sep ∈ {"/", ", ", " y ", " a ", " al ", "-", "–"}` y la misma serie `a`, y la expande a tokens completos.

- **Lista** (`/`, `, `, ` y `): cada elemento se sustituye por su ID nuevo, conservando el separador (`REQ-21.5/21.6` → `RF-PRE-01/RF-PRE-02`).
- **Rango** (` a `, ` al `, `-`, `–`): ambos extremos deben resolver en la misma capacidad, y los IDs antiguos del intervalo presentes en el mapa de esa capacidad deben dar una serie contigua de números nuevos; entonces se escribe `NUEVO-X a NUEVO-Y` (o `NUEVO-X–NUEVO-Y` cuando el separador era `-` o `–`: el guion es ambiguo con los guiones del ID RF). Si no se cumple, fallo con la ubicación.
- Los tres H1 con rango (`change-actor-identity` `REQ-42.1 a REQ-42.9` → `RF-ACT-01 a RF-ACT-09`; `odd-living-spec-integration` `REQ-30.1 a REQ-30.8` → `RF-OLS-01 a RF-OLS-08`; `openspec-adoption` `REQ-30.9 a REQ-30.14` → `RF-ADP-01 a RF-ADP-06`) cumplen la contigüidad con las medidas actuales.

**Alternativas consideradas:** suprimir los rangos de los H1 (pierde información y no resuelve los rangos de la prosa y del código); expandir todo rango a lista (ilegible en `REQ-30.16 a REQ-30.18`).

**Justificación:** `reqid.IDMap.Substitute` no sustituye `REQ-21.8` en `REQ-21.8-21.12` porque `-` es carácter de token (`isTokenRune`); sin este paso esos casos quedarían sin migrar en silencio.

### D6. Reescritura de títulos

**Elección:** para el requisito de la entrada (capacidad, posición):

- Si el título tiene un grupo entre paréntesis formado solo por IDs antiguos y separadores (`&`, `,`, ` y `), el primero se sustituye por `(NUEVO)` y los demás se eliminan, en su posición: `(REQ-21.14) [SUPERSEDED / DEPRECADO]` → `(RF-RCL-02) [SUPERSEDED / DEPRECADO]`; `(REQ-10.1 & REQ-10.2)` → `(RF-PER-08)`; `(REQ-27.2b)` → `(RF-CLI-09)`.
- Si no tiene ninguno (48), se añade ` (NUEVO)` al final.
- Si ya tiene un ID válido bajo `RF-<DOM>-<NN>` igual al del mapa, no se toca (idempotencia); si tiene otro distinto, fallo.

`reqid.Pattern.InTitle` toma el último `(ID)` válido en cualquier posición, así que el sufijo `[SUPERSEDED / DEPRECADO]` o `— RETIRADO` no impide leer el ID.

### D7. Lista de residuos: `internal/idmigration/residues.yaml`, granularidad por ruta, ámbito y tokens

**Elección:**

```yaml
schema: axiom.id-residues/v1
entries:
  - paths: ["internal/livespec/*_test.go"]
    scope: go-string
    category: default-pattern-fixture
    reason: "Fixtures que ejercitan el patrón por defecto REQ-<n>.<m>."
  - paths: ["docs/manual-de-inicio.md"]
    scope: all
    tokens: [REQ-30.4]
    category: default-pattern-example
    reason: "Ejemplo del patrón por defecto para otros proyectos."
  - paths: ["openspec/changes/migrar-ids-rf/**", "openspec/changes/archive/*-migrar-ids-rf/**"]
    scope: all
    category: legacy-citation
    reason: "Documentos de este cambio: citan los IDs antiguos como tales."
```

- `scope` ∈ {`all`, `go-comment`, `go-string`, `md-code`} (`md-code` = código en línea y bloques vallados).
- `category` ∈ {`default-pattern-fixture`, `default-pattern-definition`, `default-pattern-example`, `legacy-citation`, `dated-history`}: el conjunto cerrado de RF-RID-16 (fixture del patrón por defecto; definición o ejemplo del patrón por defecto; ejemplo dirigido a otros proyectos; cita deliberada de un ID antiguo; historia fechada fuera de OpenSpec).
- `tokens` es obligatorio salvo en: `default-pattern-fixture` con `go-string` en `*_test.go` o `scope: all` bajo `**/testdata/**`; `default-pattern-example` con `md-code` en `openspec/specs/**/spec.md` u `openspec/INDEX.md`; `dated-history` bajo `docs/archive/**` o `docs/audits/**`; y la entrada de la carpeta de este cambio.
- Validación estructural (falla la prueba): categoría o ámbito fuera del conjunto, motivo vacío, `tokens` ausente donde es obligatorio, y **cualquier entrada que alcance literales de Go de producción** (`go-string` o `all` sobre un `.go` sin `_test`): un mensaje de producción nunca es residuo.

**Prueba permanente `TestRepositoryLegacyIDResidues`** (siempre activa): recorre `git -C <raíz> ls-files -z`, busca `REQ-\d+\.\d+` (la expresión del criterio del plan) y clasifica cada aparición por ámbito (`go/scanner` para Go; tramos de código para Markdown).

- **Obligación:** toda aparición fuera de `openspec/changes/archive/**` y de `openspec/id-map.yaml` debe estar cubierta por una entrada; si no, falla con `ruta:línea token`.
- **Caducidad:** toda entrada, y cada token de su lista, debe casar con al menos una aparición en **cualquier** fichero versionado, incluido el archivo. Así la entrada de la carpeta de este cambio sigue viva tras archivarse en `openspec/changes/archive/<fecha>-migrar-ids-rf/` y el PR documental del archivado no rompe la prueba.

**Alternativas consideradas:** granularidad por línea (frágil ante cualquier edición); por fichero sin tokens (un fichero mixto oculta citas propias, hueco 9 de la revisión de cobertura); ubicar la lista en `openspec/` (es un artefacto del repositorio de Axiom, no del contrato de producto; junto a su prueba se revisa mejor y ninguna herramienta de producto la leerá por error).

**Justificación:** el par (ámbito, tokens) acota las exenciones sin depender de números de línea, y las excepciones sin `tokens` quedan restringidas por estructura a los casos medidos (fixtures, ejemplos en specs e historia).

### D8. `axiom trace` con IDs heredados: interfaz en `reqtrace`, implementación en `legacyid`, cableado en la CLI

**Elección:**

- `reqtrace` sigue sin importar `os` ni YAML (`imports_test.go`). Define la interfaz `LegacyResolver` (abajo); `legacyid.Map` la implementa de forma estructural, sin importar `reqtrace`.
- `RunTrace` (`internal/cli/trace.go`): tras `projectIDPattern`, **antes** de `ParseID`, ejecuta `legacyid.Load(os.DirFS(loc.PlanningHome), legacyid.FileName, pattern)`. Ausente → `nil, nil` (sin alias, sin aviso). Inválido → `*legacyid.MapError` (fichero, código estable y continuación) y retorno sin llamar a `traceSDDSource`, `traceODDSource` ni `Collect`: no se ejecuta git (RF-TRC-17).
- `ParseID(raw, p, legacy)`: válido → `Query{ID}`; provisional → como hoy; si no, `legacy != nil && legacy.Legacy(raw)` → `Query{ID: raw, Legacy: true}`; si no, `*IDError` (RF-TRC-02). La comparación es exacta, sin normalizar ceros ni mayúsculas, como hoy.
- `resolveEquivalents`: `acceptableAlias` acepta también un alias que el resolutor conoce (`legacy.Legacy(id)` o ID nuevo del mapa). Un ID antiguo resuelve a todos sus IDs nuevos y un ID nuevo a sus antiguos (RF-TRC-04, reglas 1 y 2). Un ID antiguo consultado con más de un ID nuevo añade el aviso `legacy_ambiguous`, simétrico de `provisional_ambiguous`. Las entradas de traza (`ids.yaml`) cuyo `definitive` o `previous` es heredado se traducen con su `capability` antes del cierre.
- **Coincidencia acotada por capacidad** (RF-TRC-04, regla 3): `matchSet` deja de incluir IDs heredados (salvo el propio ID consultado). Los tokens heredados se buscan aparte (`reqid.Default().Scan`, válidos y mal formados, filtrados por `legacy.Legacy`) y un token `T` casa en un contexto de capacidades `ctx` si `{legacy.InCapability(c, T) : c ∈ ctx}` corta el cierre. Contexto: en una delta archivada, la carpeta `specs/<c>/`; en `Cubre:` de un cambio archivado, las capacidades de su `specs/`; en un documento ODD, sus `### Capacidad:`. Sin contexto útil: si `T` tiene una sola capacidad, esa; si `T` es literalmente el ID consultado, casa; si no, no casa y se emite `legacy_ambiguous` con la ruta y las candidatas.

**Alternativas consideradas:** ampliar `AliasResolver.Aliases(id)` con la capacidad (cambia la firma de un punto de extensión ya especificado y probado); poner la carga del mapa dentro de `Collect` leyendo `in.SDD.Files` (ocurre después de `checkRef`, que ya ejecuta git, y RF-TRC-17 exige fallar sin git); que `legacyid` importe `reqtrace` (acopla el modelo del mapa al de la traza).

**Justificación:** conserva el contrato de solo lectura de `reqtrace`, hace explícita la dependencia en `Input` y resuelve los falsos positivos de los ambiguos donde se producen, en la clasificación.

### D9. Corrección de `ParseDelta`: en `parseOpenSpecDelta`, compartida con `Compose`

**Elección:** `parseOpenSpecDelta` calcula `fencedRanges(text)` una vez y descarta las cabeceras `## (ADDED|…) Requirements` y `### Requirement:` que empiezan dentro de una valla (la segunda pasa a `parseDeltaRequirementBlocks` con el desplazamiento de la sección). `ParseDelta` y `Compose` usan la misma función, así que el arreglo cubre ambos.

**Alternativas consideradas:** filtrar solo en `ParseDelta` (validación y composición divergirían: `ParseDelta` vería un requisito y `Compose` dos).

**Justificación:** la propuesta dice que `Compose` ya respeta las vallas, pero eso solo es cierto para la spec canónica (`parseSpecDocument`); para la delta, `Compose` usa el mismo `parseOpenSpecDelta` que `ParseDelta` (`internal/openspecdoc/compose.go:362`). La prueba RED cubre los dos consumidores. `livespec` (`odddoc.go:317`, `plan.go:214`, `registry.go:117`) llama a `ParseDelta` con el contenido de la valla `openspec` ya extraído, así que solo le afectan vallas anidadas, que pasan a ignorarse como corresponde.

### D10. Declaración del patrón, `INDEX.md` y goldens

**Elección:**

- La herramienta inserta en `axiom.yaml`, tras `workflow:`, el bloque `sdd:` con `id_pattern: "RF-<DOM>-<NN>"` y su comentario. Si ya existe con ese valor, no lo toca; con otro valor, falla.
- `INDEX.md` se regenera en proceso con `livingdoc.NewService(raíz, nil, nil).Sync(ctx)` (lee el patrón del `axiom.yaml` ya actualizado). Para ser idempotente, solo se escribe si el contenido difiere del actual ignorando la línea `> **Última Sincronización:**`.
- Los 10 goldens (`internal/components/testdata/golden/*`, derivados de `_shared/sdd-orchestrator-sections.md`) se regeneran después con `go test ./internal/components -run TestGolden -count=1 -update` y se comprueba una segunda ejecución sin `-update`.

**Alternativas consideradas:** `axiom archive sync` del `PATH` (es el binario de `main`, sin las correcciones de la rama; además, ejecutar binarios es lo que se evita); que la herramienta regenere goldens (están en otro paquete y su mecanismo oficial es `-update`).

### D11. Pruebas sobre el corpus real con el patrón RF

**Elección:**

- `internal/openspecdoc/compose_corpus_test.go`: se añade `TestComposeCorpusIdentityModifiedRF` con `reqid.Parse("RF-<DOM>-<NN>")`, que además exige `p.InTitle(nombre) != ""` en todo requisito y `reqid.Default().InTitle(nombre) == ""` (ningún `REQ-n.m` sobrevive en un título). La variante por defecto se conserva (ahora ejercita la coincidencia por nombre).
- `internal/reqid/legacy_equivalence_test.go`: la equivalencia con la gramática heredada se mantiene y su corpus pasa a ser los títulos de `openspec/changes/archive/*/specs/*/spec.md` (cientos de títulos `REQ-n.m` que no cambian); se añade `TestRFCorpusTitlesHaveIDs` sobre `openspec/specs`.
- `internal/idmigration`: `TestIDMapMatchesLivingSpecs` (siempre): carga el mapa real con `legacyid.Load`; cada ID nuevo presente en la spec viva está en su capacidad y en el orden del mapa; ninguna entrada se ha perdido por edición manual del fichero (el número de entradas es 513 mientras no exista una versión nueva del esquema).
- `internal/reqtrace`: `TestTraceRealMapResolvesLegacyID` lee `../../openspec/id-map.yaml` y el patrón de `../../axiom.yaml` y comprueba que `REQ-24.3` se acepta y resuelve a `RF-PLH-03`, y que `REQ-1.1` resuelve a siete IDs nuevos con `legacy_ambiguous`. Es la verificación en proceso de «`axiom trace REQ-24.3` responde con el ID nuevo» sin depender de la historia git de la CI (clon superficial).

### D12. La delta de este cambio y su carpeta

La delta ya usa los IDs RF (decisión del orquestador): `MODIFIED` con `RF-TRC-02` y `RF-TRC-04`, `ADDED` con provisionales que el archivado asigna (`RF-TRC-16`, `RF-TRC-17`, `RF-RID-16`). La carpeta `openspec/changes/migrar-ids-rf/` no se reescribe y se declara residuo `legacy-citation` con dos rutas (activa y archivada), de modo que la caducidad de D7 no salta al archivar. Antes del archivado, `TestIDMapMatchesLivingSpecs` garantiza que `RF-TRC-02` y `RF-TRC-04` existen en `requirement-trace` con los títulos que la delta modifica.

### D13. Tokens sin equivalente vivo: nunca se inventa un ID

- En un **literal de Go de producción**, un token no mapeado bloquea `plan`/`write`: se corrige a mano antes de la reescritura, citando el requisito vivo que lo sustituyó. Caso único medido: `internal/techcontext/plan.go:137` (`REQ-24.62`, de la baseline retirada; su sucesor vivo es «El Contexto Técnico Mantenido Exige un Almacén de Artefactos de Ficheros», `REQ-30.19` de `project-technical-context`, que tras migrar se cita con su ID nuevo). La prueba que fija ese mensaje se ajusta en el mismo commit.
- En cualquier otro lugar, se conserva y se declara: retirados → `legacy-citation` con motivo «requisito retirado sin equivalente vivo»; ejemplos → `default-pattern-example`. `plan` propone las entradas.

## Flujo de datos

```
               (una vez, commits d1-d3)                              (permanente, cada consulta)
openspec/specs/*/spec.md ──► generador (test) ──► openspec/id-map.yaml ──► legacyid.Load ──► *legacyid.Map
        │                         ▲ domains_test.go                           ▲                 │ (LegacyResolver)
        │                                                                     │                 ▼
        ▼                                                       cli.RunTrace: pattern ─► ParseID(raw,p,m)
 reescritor (test) ◄── residues.yaml + contexts.yaml                                         │
   │  reqid.IDMap por línea, go/scanner, go/format                                          ▼
   ├─► specs, Go, ODD, docs, assets, axiom.yaml (+ bloque sdd:)              reqtrace.Collect(Input{Aliases: m, Legacy: m})
   ├─► livingdoc.Service.Sync ─► openspec/INDEX.md                             resolveEquivalents ─► cierre (+legacy_ambiguous)
   └─► (aparte) go test ./internal/components -update ─► goldens               classify* ─► coincidencia acotada por capacidad

 TestRepositoryLegacyIDResidues: git ls-files ─► REQ-\d+\.\d+ ─► ámbito ─► residues.yaml (obligación + caducidad)
```

Secuencia de `TestMigrateRepository` en modo `write`:

```
cargar mapa (falla si falta) ─► comprobar/insertar sdd: en axiom.yaml (en memoria)
  ─► listar ficheros (git ls-files) ─► por fichero: regla D4 ─► títulos D6 ─► continuaciones D5 ─► ambiguos D3
  ─► ¿algún ambiguo sin resolver, rango no contiguo, literal de producción no mapeado, título con ID distinto?
        sí ─► informe + fallo, sin escribir nada
        no ─► escribir ficheros cambiados (Go por go/format) ─► Sync de INDEX (si cambia algo más que la fecha)
  ─► segunda pasada en memoria: debe producir cero cambios (idempotencia)
```

## Cambios de ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/openspecdoc/compose.go` | Modificar | `parseOpenSpecDelta` y `parseDeltaRequirementBlocks` ignoran cabeceras dentro de vallas (D9) |
| `internal/openspecdoc/parse_test.go`, `compose_test.go` | Modificar | Pruebas RED de vallas en la delta para `ParseDelta` y `Compose` |
| `internal/openspecdoc/compose_corpus_test.go` | Modificar | Variante RF (D11) |
| `internal/legacyid/legacyid.go` | Crear | `FileName`, `Schema`, `Load`, `Map` y sus métodos, `MapError` y códigos |
| `internal/legacyid/legacyid_test.go`, `imports_test.go` | Crear | Carga, invariantes, errores, resolutor; sin `os/exec` ni `gitcommon` en producción |
| `internal/reqtrace/alias.go`, `id.go`, `classify.go`, `collect.go`, `model.go` | Modificar | `LegacyResolver`, `Input.Legacy`, `Query.Legacy`, `ParseID` con resolutor, `acceptableAlias`, `matchSet`, coincidencia heredada por capacidad, `WarnLegacyAmbiguous` |
| `internal/reqtrace/*_test.go` | Modificar/Crear | Fixtures de mapa en memoria; prueba con el mapa real (D11) |
| `internal/cli/trace.go` | Modificar | Carga del mapa antes de `ParseID`; error tipado; `Aliases` y `Legacy` en `Input` |
| `internal/cli/trace_flow_test.go` (o `trace_legacy_test.go`) | Modificar/Crear | Repositorio git temporal con `id-map.yaml`: ID antiguo aceptado, ambiguo, fuera del mapa, mapa inválido sin git |
| `internal/idmigration/*_test.go` | Crear | Generador, tabla de dominios, reescritor, residuos, `TestMigrateRepository` (con variable), `TestRepositoryLegacyIDResidues` y `TestIDMapMatchesLivingSpecs` (siempre) |
| `internal/idmigration/testdata/**` | Crear | Árbol de repositorio en miniatura para las pruebas del generador y del reescritor |
| `internal/idmigration/residues.yaml`, `contexts.yaml` | Crear | Lista de residuos (D7) y tabla de contexto (D3) |
| `internal/reqid/legacy_equivalence_test.go` | Modificar | Corpus del archivo y prueba RF sobre la spec viva (D11) |
| `internal/techcontext/plan.go` (+ su prueba) | Modificar | Mensaje con `REQ-24.62` retirado → requisito vivo (D13) |
| `openspec/id-map.yaml` | Crear (generado) | Mapa de 513 entradas |
| `axiom.yaml` | Modificar (generado) | Bloque `sdd:` y comentarios migrados |
| `openspec/specs/**/spec.md` (49), `openspec/INDEX.md` | Modificar (generado) | Reescritura y regeneración |
| `internal/**`, `cmd/axiom/**`, `e2e/**` (Go), `internal/assets/skills/**` (5 md), `internal/dashboard/assets/**` (JS/CSS), `odd/tasks/*.md`, `docs/**` vivo, `skills/**`, `axiom.example.yaml`, `.github/workflows/ci.yml`, `.goreleaser.yaml` | Modificar (generado) | Reescritura mecánica |
| `internal/components/testdata/golden/*` (10) | Regenerar | `-update` |
| `docs/openspec-config.md`, `docs/manual-de-inicio.md` | Modificar (a mano) | `sdd.id_pattern` de Axiom, `openspec/id-map.yaml` y consulta de IDs antiguos con `axiom trace` |

## Interfaces / contratos

```go
// package legacyid — producción, alcanzable desde RunTrace.

const (
	FileName = "id-map.yaml"     // relativo al planning home
	Schema   = "axiom.id-map/v1"
)

// Códigos estables de MapError (RF-TRC-17).
const (
	CodeUnreadable    = "id_map_unreadable"
	CodeMalformed     = "id_map_malformed"      // YAML inválido, campo desconocido o vacío
	CodeSchemaUnknown = "id_map_schema_unknown"
	CodeInvariant     = "id_map_invariant"      // ID nuevo repetido o no válido, antiguo repetido en su capacidad, entrada sin old ni title, capacidad vacía
)

type MapError struct {
	Path   string // ruta mostrada del fichero
	Code   string
	Detail string // motivo concreto (entrada e invariante)
	Err    error
}
// Error() nombra el fichero, el código, el motivo y la continuación: corregir el mapa o restaurarlo
// desde git (`git checkout -- openspec/id-map.yaml`).

// Load lee name de fsys. Fichero ausente: (nil, nil). Cualquier otro fallo: *MapError.
func Load(fsys fs.FS, name string, p reqid.Pattern) (*Map, error)

type Map struct{ /* índices: (cap, old) -> new; old -> caps; new -> olds */ }

func (m *Map) Aliases(id string) []string                // old -> todos sus nuevos; nuevo -> sus antiguos; ordenados
func (m *Map) Legacy(id string) bool                     // id es clave antigua en alguna capacidad
func (m *Map) Known(id string) bool                      // Legacy(id) o id es un ID nuevo del mapa
func (m *Map) InCapability(capability, old string) string // "" si el par no existe
func (m *Map) Capabilities(old string) []string          // ordenadas
```

```go
// package reqtrace

// LegacyResolver añade a AliasResolver la resolución por capacidad de los IDs heredados (RF-TRC-04).
type LegacyResolver interface {
	AliasResolver
	Legacy(id string) bool
	Known(id string) bool
	InCapability(capability, old string) string
	Capabilities(old string) []string
}

type Query struct {
	ID          string
	Provisional bool
	Legacy      bool // ID heredado aceptado por el mapa (RF-TRC-02)
}

func ParseID(raw string, p reqid.Pattern, legacy LegacyResolver) (Query, error) // legacy puede ser nil

type Input struct {
	// ... campos actuales ...
	Legacy LegacyResolver // nil: sin IDs heredados (comportamiento actual)
}

const WarnLegacyAmbiguous = "legacy_ambiguous"
```

Cableado en la CLI (sin interfaz con puntero nulo):

```go
m, err := legacyid.Load(os.DirFS(loc.PlanningHome), legacyid.FileName, pattern)
if err != nil {
	return fmt.Errorf("trace: %w", err) // antes de cualquier git
}
var legacy reqtrace.LegacyResolver
var aliases reqtrace.AliasResolver = reqtrace.NoAliases{}
if m != nil {
	legacy, aliases = m, m
}
query, err := reqtrace.ParseID(id, pattern, legacy)
// ...
reqtrace.Collect(ctx, reqtrace.Input{Query: query, Pattern: pattern, Ref: ref, Aliases: aliases, Legacy: legacy, SDD: sdd, ODD: odd})
```

Entradas curadas de la herramienta:

```yaml
# internal/idmigration/contexts.yaml
schema: axiom.id-contexts/v1
entries:
  - path: internal/hub/init.go
    token: REQ-2.2
    capability: multi-project-hub
    reason: "El comentario nombra multi-project-hub en la línea 33."
```

## Estrategia de pruebas

TDD no está activo en el proyecto (`strict_tdd: false`); la propuesta exige una prueba RED observada para `ParseDelta`, y se pide también RED para las reglas nuevas de trace. Pruebas siempre localizadas (nunca la suite completa en local; en `internal/cli`, solo con `-run`).

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unidad | `ParseDelta`/`Compose` ignoran `### Requirement:` y `## ADDED Requirements` dentro de vallas | RED primero en `./internal/openspecdoc`; luego `go test ./internal/openspecdoc ./internal/livespec -count=1` |
| Unidad | `legacyid.Load`: ausente, ilegible, YAML inválido, campo desconocido, esquema desconocido, ID nuevo repetido o no válido para el patrón, antiguo repetido en una capacidad (y permitido en dos), entrada sin `old` ni `title`; resolutor (`Aliases` en los dos sentidos, `InCapability`, `Capabilities`) | Tabla en `./internal/legacyid` con `fstest.MapFS` |
| Unidad | `ParseID` (antiguo del mapa aceptado, fuera del mapa rechazado, sin mapa rechazado); cierre con ambiguo y `legacy_ambiguous`; coincidencia acotada (delta de `autoskills` frente a `workspace-topology` con `REQ-1.1`); `Cubre:` con y sin contexto; `ids.yaml` con `definitive` heredado | `./internal/reqtrace` con fuentes `fstest.MapFS` e historia simulada, como las pruebas existentes |
| Integración | `axiom trace` con un `id-map.yaml` en un repositorio git temporal: ID antiguo, ambiguo, fuera del mapa, mapa inválido (falla sin ejecutar git), sin mapa (sin aviso) | `go test ./internal/cli -run 'TestTrace' -count=1` |
| Integración | Mapa real: `REQ-24.3` → `RF-PLH-03`; `REQ-1.1` → siete IDs | `go test ./internal/reqtrace -run TestTraceRealMap -count=1` |
| Unidad | Generador: doble generación idéntica byte a byte, numeración por aparición, títulos no finales, `(A & B)`, `REQ-27.2b`, ambigüedad calculada, ida y vuelta con `legacyid.Load` | Árbol en `internal/idmigration/testdata` |
| Unidad | Reescritor: token completo (`REQ-1.1` no toca `REQ-1.10`), títulos D6, continuaciones y rangos D5 (contiguo y no contiguo), contexto D3 (cada eslabón y el fallo sin escribir), Go (comentario, literal fixture, literal fijado, `go/format`), ODD (vallas `openspec`, capacidad retirada), Markdown de specs (código en línea conservado), idempotencia (segunda pasada sin cambios) | Tablas con `t.TempDir()` |
| Unidad | Residuos: cita no declarada → fallo con `ruta:línea`; entrada o token caducado → fallo; residuo declarado aceptado; validación estructural (categoría, motivo, `tokens`, prohibición sobre literales de producción); caducidad que cuenta el archivo | Árbol en `testdata` |
| Repositorio | `TestRepositoryLegacyIDResidues`, `TestIDMapMatchesLivingSpecs`, variantes RF del corpus | `go test ./internal/idmigration ./internal/openspecdoc ./internal/reqid -count=1` |
| Golden | 10 goldens tras `-update`, estables sin `-update` | `go test ./internal/components -run TestGolden -count=1` |
| Paquetes reescritos | Comentarios, literales y pines | `go test` y `go vet` de cada paquete tocado por la reescritura, por lotes; la suite entera, en la CI |

## Matriz de amenazas

La herramienta de un solo uso (solo pruebas) ejecuta `git ls-files` y la prueba de residuos también; `axiom trace` lee un fichero nuevo pero no añade subprocesos. Ninguna otra frontera cambia.

| Frontera | Casos adversos mínimos | Aplicabilidad | Respuesta de diseño | Pruebas RED previstas |
|---|---|---|---|---|
| Rutas con aspecto de documentación | `requirements.txt`, Markdown ejecutable, `README.sh` | N/A: no se clasifica ni se ejecuta ningún fichero; la herramienta solo lee y reescribe texto | — | — |
| Selección del repositorio git | `git -C`, ruta relativa, ruta absoluta | Aplicable (solo código de prueba) | La raíz se calcula una vez como ruta absoluta desde el directorio del paquete (`filepath.Abs("../..")`) y se comprueba con `git -C <raíz> rev-parse --show-toplevel`; toda orden usa `git -C <raíz>` y es de solo lectura (`ls-files`); nunca depende del directorio del proceso; las rutas escritas son siempre `<raíz>/<ruta de ls-files>` tras rechazar `..` y rutas absolutas | Prueba de la resolución de raíz que rechaza una raíz cuyo `show-toplevel` no coincide; prueba de que una ruta con `..` de la lista no se escribe |
| Estado del commit | índice preparado, `commit -a`, índice vacío | N/A: la herramienta no hace commits; los commits los hace la persona o `sdd-apply` | — | — |
| Estado del push | rama con seguimiento, primer push, refspec | N/A: sin push | — | — |
| Órdenes de PR | `--head`, prefijo de entorno | N/A: sin automatización de PR | — | — |

## Migración / despliegue

### Orden de las unidades de trabajo (un commit por tarea, una sola PR)

| # | Commit | Contenido | Reversión aislada |
|---|---|---|---|
| a | `fix(openspecdoc): ignorar requisitos dentro de vallas en la delta` | D9 con RED | Sí, independiente |
| b | `feat(trace): resolver identificadores heredados con un mapa versionado` | `internal/legacyid`, `reqtrace`, cableado CLI, pruebas con fixtures. Sin mapa real: comportamiento idéntico al actual | Sí, si no se ha aplicado e (su prueba del mapa real depende de b) |
| c | `test(idmigration): herramienta de migración de identificadores` | Paquete solo de pruebas: generador, reescritor, modelo y validación de residuos con `testdata`; `TestMigrateRepository` con variable | Sí (solo pruebas) |
| d1 | `chore(specs): generar el mapa de identificadores heredados` | `AXIOM_IDMIGRATION=map`: solo `openspec/id-map.yaml` | Sí con d2/d3/e revertidos |
| d2 | `chore(idmigration): residuos, contextos y mensajes retirados` | `residues.yaml`, `contexts.yaml` y el arreglo a mano de `techcontext/plan.go:137` y su prueba (D13); hasta que `plan` pase sin pendientes | Con d3 |
| d3 | `refactor(specs): migrar los identificadores a RF-<DOM>-<NN>` | Salida exacta de `AXIOM_IDMIGRATION=write` (bloque `sdd:`, specs, INDEX, Go, ODD, docs, assets) más los goldens con `-update`. Sin ediciones a mano | Sí; es el commit que se revierte si falla la migración, conservando a, b y c |
| e | `test(specs): residuos y corpus con el patrón RF` | `TestRepositoryLegacyIDResidues`, `TestIDMapMatchesLivingSpecs`, variantes RF, prueba del mapa real en `reqtrace` | Sí |
| f | `docs: patrón RF de Axiom y consulta de IDs antiguos` | `docs/openspec-config.md`, `docs/manual-de-inicio.md` | Sí |

Revisión de d3: se reproduce con `git checkout <d2>`, `AXIOM_IDMIGRATION=write go test ./internal/idmigration -run TestMigrateRepository -count=1`, `go test ./internal/components -run TestGolden -count=1 -update` y `git diff <d3>`, que solo puede diferir en la línea de fecha de `INDEX.md`.

### Reaplicación tras rebasar

El mapa no es «publicado» (invariante 5 de RF-TRC-16) hasta fusionar la PR. Si `main` avanza con trabajo que añade o cita `REQ-n.m`:

1. Rebasar descartando d1 y d3 (generados): `git rebase -i` con `drop` (o en no interactivo, rehacer la rama con `git cherry-pick` de a, b, c, d2, e, f sobre `main`).
2. Resolver conflictos solo en commits escritos a mano (d2 suele necesitar entradas nuevas).
3. `AXIOM_IDMIGRATION=map …`, commit d1; `AXIOM_IDMIGRATION=plan …` hasta que no quede nada pendiente (ampliando d2); `AXIOM_IDMIGRATION=write …` y `-update` de goldens, commit d3.
4. `go test ./internal/idmigration ./internal/openspecdoc ./internal/reqid ./internal/reqtrace -count=1`.

Sobre un árbol ya migrado, `write` no cambia nada (D6, títulos con el mismo ID; D10, INDEX solo por fecha); la idempotencia la comprueba la segunda pasada en memoria.

### Tras fusionar y al archivar

- El archivado (`axiom sdd archive` en `inc/migrar-ids-rf-archive`) compone la delta con las specs ya migradas: `RF-TRC-02`/`RF-TRC-04` casan por ID y los provisionales reciben `RF-TRC-16`, `RF-TRC-17` y `RF-RID-16`.
- La entrada de residuos de la carpeta del cambio sigue viva tras moverse a `archive/` (D7, caducidad sobre todo el repositorio).
- Reversión después de fusionar: revertir el merge (recomendado merge sin squash) o solo d3 + d1 + e; b sin mapa se comporta como hoy.

## Preguntas abiertas

- [ ] **No bloquea.** Las citas de requisitos retirados sin equivalente vivo (`REQ-24.38`–`REQ-24.62`, `REQ-15.2`…) y las deltas ODD de capacidades retiradas se declaran con `legacy-citation` («cita deliberada de un ID antiguo como tal»), porque el conjunto cerrado de RF-RID-16 no tiene una categoría propia. Si se prefiere una categoría `retired` explícita, debe añadirla `sdd-spec`.
- [ ] **No bloquea.** El aviso `legacy_ambiguous` amplía los códigos de advertencia de `axiom.trace/v1` sin escenario en la spec (huecos 6 y 8 de la revisión de cobertura); el diseño lo adopta por simetría con `provisional_ambiguous`.
- [ ] **No bloquea.** `old` es una lista para cubrir los dos títulos de `persona-behavior-contract` con dos IDs; la spec habla de «el ID antiguo». Se interpreta como cero o más IDs antiguos por requisito.
- [ ] **Fuera de alcance, seguimiento recomendado.** El visor de specs del dashboard (`internal/dashboard/assets/js/core/spec.js:25`, `REQ_ID = /…\(((?:REQ|REQUISITO)[^)]*)\)\s*$/i`) solo separa del título IDs que empiezan por `REQ`; tras migrar, el dashboard de Axiom mostrará `(RF-…)` como parte del título. Ya afecta a cualquier proyecto RF desde C-03-a; conviene un ODD propio.
