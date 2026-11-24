## Exploración: puerto único de identificadores de requisito (C-03-a, `puerto-reqid`)

### Estado actual

**Mapa de `REQ-` en código de producción (no `_test.go`)**

| Fichero:línea | Clase | Qué hace |
|---|---|---|
| `internal/openspecdoc/parse.go:33-42` | lógica | `requirementIDInTitle = \((REQ-\d+\.\d+)\)`; `RequirementID` devuelve el último `(REQ-n.m)` del título. Rellena `Requirement.ID` en `ParseDelta` (:56) y `ParseCanonical` (:98). Paquete hoja (solo stdlib). |
| `internal/openspecdoc/compose.go:310-316,333,348,366` | lógica (sin regex) | `indexOf(name)` empareja por `s.Name == name`, el título completo, `(REQ-x)` incluido. Renombrar el título rompe la identidad. `sddstatus/openspec_archive_compose.go:12` es solo un alias de tipo. |
| `internal/isoevidence/reqid.go:13-14` | lógica | `reqCandidate = \bREQ-[0-9A-Za-z.]*[0-9A-Za-z]` (token que empieza por `REQ-`, sin puntuación final) frente a `reqValid = ^REQ-\d+\.\d+$`. `RequirementIDs(name)` devuelve `(valid, malformed)` sin repetidos y en orden. |
| `reqid.go:55-58,100,159` y `cubre.go:44,159,224` | mensaje/comentario | Continuaciones con `REQ-<n>.<m>`; las pruebas fijan ese texto (`reqid_test.go`, `iso_stages_test.go` x10, `model_test.go`). |
| `internal/isoevidence/cubre.go:21-41` | lógica indirecta | `ParseCubre` devuelve los tokens tal cual (también los mal formados); `NewBaselineIndex` (:51-58), `CheckCoverage`, `ScenarioKey` (:233) usan `RequirementIDs`. |
| `internal/livespec/registry.go:59,122-128,140-170` | lógica | `reqIDParts = ^REQ-(\d+)\.(\d+)$`; `maxFor map[int]int`; `NextFree(series int)` formatea `REQ-%d.%d`; `SeriesRefs(int)`; `SeriesOf(id) (int,bool)`. |
| `internal/livespec/plan.go:257-308` | lógica + mensaje | `checkIdentifiers` (`added_without_id`, `CollisionError`); `nextFree` reformatea `REQ-%d.%d` (:305). |
| `internal/livingdoc/indexer.go:17,136` | lógica (laxa) | `\((REQ-[^\)]+)\)` acepta cualquier cosa tras `REQ-`; ID sintético `REQ-<dominio>-<n>` si falta. Solo informativo. |
| `internal/adoption/adoptcheck/check.go:148,172,204` | mensaje + reutiliza | `RuleFormat` exige `(REQ-n.m)`; `livespec.SeriesOf` (:172) y regla «serie única» (:194-204) con `map[int]bool`. La reescritura es de C-03-c; aquí solo debe seguir compilando. |
| `internal/cli/archive_integrate.go:106`, `archive_adopt.go:54` | mensaje | Ayuda con `(REQ-n.m)`. |
| `internal/cli/sdd_trace.go:357-375` | consumidor | Usa `isoevidence.RequirementIDs` (3 sitios); ya existe `sdd trace`. |
| `internal/adoption/scan.go:34,199` | fuera de alcance (C-03-c) | `foreignIDPattern` para IDs ajenos (`RF-120`); descarta `REQ-n.m` con sufijo `.<dígito>`. |
| `internal/sddstatus/iso.go:111-164` | consumidor | Construye `delta`/`baseline` con `openspecdoc` y llama `isoevidence.CheckRequirementIDs`. |
| El resto de coincidencias de `rg 'REQ-'` | comentario | Citas de REQ-21/25/27/29/30 en `//` y constantes de doc (branch, workspace, agentguidance...). No son lógica de ID. |

**Semánticas a conservar**
1. isoevidence: *candidato* (laxo, por prefijo) frente a *válido* (exacto) y errores tipados: `Finding{Code: CodeRequirementID, Stage: StageSpecs, Subject, Detail, Continuation}`; tolerancia D17 (`tolerated`, :165); RENAMED que no cambia ID; ID repetido en la delta.
2. livingdoc: lectura laxa con ID sintético; solo para el índice (no debe validar).
3. livespec: serie = `n`, `NextFree` = máximo `m` + 1; `CollisionError{ID, Where, NextFree}`; `DocError` con `Problem.Code` `added_without_id`.
4. adoptcheck: reglas `RuleFormat/RuleUniqueness/RuleSeries`.

**Esquema `axiom.yaml`**
- `internal/workspace/sdd.go:11-17`: `SDDConfig{ISO bool}`, `sddKeys = {"iso"}`, `decodeSDD` estricto (`unknownKeys`, duplicados, `sddBool` exige `!!bool`). Se decodifica en `declarations.go:89` → `Declarations.SDD`/`SDDErr` (:40). `axiom.example.yaml:79-92` documenta `sdd.iso`; lo comprueban `example_yaml_test.go`, `example_config_test.go`, `docs_contract_test.go`.
- `workspace` solo importa `gitcommon`, `gitref`, `pathidentity`, `pathquote`, `integrations/manifest`: no importa ningún consumidor. `openspecdoc` no importa nada interno; `isoevidence` no importa `workspace`; `livespec` importa `openspecdoc`, `isoevidence`, `workspace`, `livingdoc`.
- Llegada del patrón: `livespec` (`PlanIntegration(loc,…)`), `adoptcheck` (`c.loc`), `cli/*` (`loc.Declarations.SDD`) y `sddstatus` (`declarations.SDD`, `iso.go:428`) ya tienen el workspace. `openspecdoc`, `isoevidence` y `livingdoc.Indexer` (`NewIndexer()` sin argumentos) NO: hay que inyectar el patrón.

**Guías y goldens (evidencia corregida frente al plan)**
- `routing.go:68,77` NO contienen `REQ-<n>.<m>` en este árbol (solo `REQ-27.x` en comentarios y prosa de carril). `openspec-convention.md` tampoco (solo REQ-21.x de otro contexto). El texto real está en: `sdd-iso-convention.md:18`, `sdd-spec/SKILL.md:257`, `sdd-spec/references/openspec-adoption.md:15`, `sdd-status-contract.md:153` (`iso_requirement_id`), `docs/manual-de-inicio.md:547,563,625,630`, `axiom.example.yaml:85`.
- `testdata/golden/`: `rg -l 'REQ-<n>\.<m>|REQ-n\.m'` = 0 ficheros; 13 contienen `REQ-` (CLAUDE.md/AGENTS.md generados, todo `REQ-2x`). No hace falta regenerar goldens por este cambio salvo que se toque `routing.go`. Pines de prueba: `assets_test.go`, `sdd_iso_pins_test.go`, `agentguidance/routing*_test.go`, `workspace/docs_*_test.go`.

**Specs vivas afectadas** (`openspec/specs/`)
- `sdd-iso-evidence`: REQ-29.3 (`spec.md:37,244-260`; el Glosario :17 también fija `REQ-<n>.<m>`), y por arrastre REQ-29.4/29.5/29.14 (cobertura/`Cubre:`).
- `odd-living-spec-integration`: REQ-30.4 (`spec.md:30,146-174`).
- `openspec-adoption`: REQ-30.11/30.12 (`:89,:110`) solo si se tocan sus mensajes; la reescritura es C-03-c.
- Compositor: requisito de composición por nombre (REQ-30.22 a 30.24; capacidad a confirmar en el propose con `rg` sobre `openspec/specs`), y `axiom-sdd-cli-integration` (`:445`, escenario con `REQ-<n>.<m>`).
- `sdd-orchestrator-assets:524-529` (cita la serie `REQ-n.m`) si cambian guías.

**Sugerencias del verify de P6 (estado real hoy)**
1. Marcador: YA no está duplicado. `livespec/odddoc.go:40` usa `impactMarkerLine = isoevidence.NoImpactMark`. No hay prueba de paridad entre carriles. Sí sigue duplicado el literal `"Motivo:"` (`odddoc.go:41` `impactReasonPrefix` frente a `bug.go:70` literal).
2. `Motivo:`: YA convergió. `odddoc.go:349-353` toma la primera línea `Motivo:` aunque esté vacía (`reasonSeen`), igual que `isoevidence.labelValue` (`bug.go:140-148`); lo cubre `impact_test.go:34`. Diferencias residuales sin efecto: ODD recorta con `TrimSpace`, ISO con `TrimLeft` + `TrimSpace` del valor (equivalente); ambos ignoran vallas. Falta fijarlo con una prueba de paridad.
   Cierre mínimo: exportar `isoevidence.NoImpactReasonLabel`, usarlo en las dos y añadir un test tabular en `livespec` (puede importar `isoevidence`) que pase las mismas fixtures a `ParseBugDeclaration` y a `parseImpact` y compare marcador y motivo.

### Opciones para el patrón declarado

Todas: paquete hoja `internal/reqid` (solo stdlib), tipo `Pattern` inmutable con `Parse(text)`, `Extract(title)`, `Candidates(text)` (laxo), `Valid(id)`, `Series(id)`, `Number(id)`, `Format(series, n)`, `Display()` (texto para mensajes). `Default()` equivale a `REQ-<n>.<m>`.

| Opción | Clave y forma | Serie | Pros | Contras | Esfuerzo |
|---|---|---|---|---|---|
| A. Regex declarada | `sdd.id_pattern: '^RF-(?P<series>[A-Z]{3})-(?P<n>\d{2,})$'` | grupo nombrado `series` (opcional) | Máxima flexibilidad (admite `RF-1`/`RF-001`) | Hay que validar grupos y deducir `Format` desde una regex (no invertible); candidato laxo no derivable; fácil equivocarse; mensajes ilegibles | Media-alta |
| **B. Plantilla con segmentos (recomendada)** | `sdd.id_pattern: "RF-<DOM>-<NN>"`; defecto `REQ-<S>.<N>`; literales + tokens `<DOM>` (código de mayúsculas), `<S>` (serie numérica), `<N>`/`<NN>`/`<NNN>` (número de secuencia; anchura mínima por nº de letras), exactamente un `<N...>`, al final del patrón o antes de un literal | todo salvo `<N>`: `<S>`→`30`, `<DOM>`→`USU`, ausente→serie única `""` (`RF-<N>`, serie del cliente `RF-121`) | Invertible (`Format`), compila a regex internamente, candidato laxo = literal inicial + `[0-9A-Za-z.\-]*[0-9A-Za-z]`; mensajes claros (`Display()`); un solo escalar string en el decodificador estricto; cubre A-03-1 y el defecto heredado | No expresa esquemas mixtos (queda como riesgo de C-03-c); anchura fija mínima | Media |
| C. Mapa estructurado | `sdd.requirement_id: {prefix: RF, domain: true, width: 2}` | `domain` booleano | Estricto, sin mini-lenguaje | Más claves, menos expresivo (`REQ-n.m` exige `separators`); más superficie de esquema compartido | Media |

Nombre de clave recomendado: `sdd.id_pattern` (plano, como `iso`, inglés como el resto de claves; `SDDConfig.IDPattern string` ya validada con `reqid.Parse` en `decodeSDD` y con error accionable; ausente = `Default()`). Un patrón inválido es error de `SDDErr`, nunca silencioso.

### Inyección y dependencias
- `workspace` importa `reqid` (hoja) para validar al decodificar; consumidores obtienen `reqid.Pattern` con un helper `workspace.Declarations.IDPattern()` (devuelve `Default()` si `SDD == nil`). Sin ciclos: `reqid` solo stdlib.
- API sin ruptura: `openspecdoc.ParseDelta/ParseCanonical(text, opts ...Option)` con `WithPattern(p)`; por defecto `reqid.Default()`. `isoevidence.CheckRequirementIDs(delta, baseline, p)` y `NewBaselineIndex(baseline, p)` reciben el patrón (cambio de firma, ~6 llamadores incl. `sddstatus/iso.go:164` y `cli/sdd_trace.go`). `livespec.LoadRegistry(home, RegistryOptions{Pattern})`, `Registry.maxFor map[string]int`, `NextFree(series string)`, `SeriesOf(id) (string,bool)` (impacta a `adoptcheck/check.go:172,194-204` y sus pruebas, solo tipos). `livingdoc.NewIndexer()` mantiene firma y gana `NewIndexerWithPattern`; su ID sintético pasa a `reqid.Synthetic(domain,n)`, claramente distinto de un ID válido.
- Compositor: `Compose` empareja por ID cuando ambos títulos lo tienen (mismo ID ⇒ mismo requisito aunque cambie el título; ID repetido en canónico ⇒ `UnappliedDeltaError`), y por título si no (cumple C-03-a). El RENAMED sigue exigiendo `Antiguo → Nuevo`.
- Criterio de cierre del plan: `rg 'REQ-' --type go internal cmd -g '!*_test.go'` solo en `internal/reqid`, el valor por defecto y comentarios; hay que reescribir también los textos de continuación (usar `Display()`, con el defecto renderizando exactamente `REQ-<n>.<m>` para no romper pines).

### Recomendación
Opción B, clave `sdd.id_pattern`, defecto `REQ-<S>.<N>` (admite todos los `REQ-n.m` vigentes hasta C-03-e), un único paquete `internal/reqid`, inyección por parámetro explícito (opciones en `openspecdoc`, argumento en `isoevidence`/`livespec`) y compositor por ID. Cerrar las dos sugerencias de P6 con la constante compartida y la prueba de paridad. Reescribir REQ-29.3 a «identificador válido según el patrón declarado» y ajustar guías con `Display()`.

### Riesgos
- Cambios de firma en `isoevidence` y `livespec` tocan muchas pruebas (19 coincidencias de `REQ-<n>.<m>` en tests, más fixtures `REQ-n.m`); hay que mantener el defecto byte a byte.
- Compositor por ID: cuidar la tolerancia D17 (archivado interrumpido) y los MODIFIED cuyo título cambió; definir el comportamiento con IDs duplicados en la canónica (hoy se detecta por nombre).
- `livingdoc` sintético `REQ-<dom>-<n>` puede coincidir con un patrón declarado `REQ-<DOM>-<NN>`: separarlo del espacio de IDs válidos.
- Esquemas mixtos de cliente (`RF-1` y `RF-001`) no cabrán en B; es decisión de C-03-c (normalizar o admitir alternativas). No adelantar.
- Esquema de `axiom.yaml` y guías son superficies [COMPARTIDAS]: coordinar con otras olas; actualizar `axiom.example.yaml` y las pruebas de documentación del ejemplo.
- Evidencia del plan desactualizada: `routing.go:68,77` y `openspec-convention.md` no contienen `REQ-<n>.<m>`; goldens: 0 ficheros afectados por el texto. Afinar la tarea en el propose.
- Sin ciclo de import previsto si `reqid` se mantiene hoja; `workspace` ya no importa consumidores.

### Listo para propose
Sí. Pregunta de producto pendiente para el propose (no bloqueante): confirmar la clave `sdd.id_pattern` y la plantilla con tokens `<DOM>/<S>/<N>` (opción B).
