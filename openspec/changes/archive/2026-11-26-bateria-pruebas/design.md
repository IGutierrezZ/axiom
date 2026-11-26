# Diseño: batería de pruebas de aceptación (`pruebas.md`) generada en sdd-tasks

Cambio `bateria-pruebas` (C-06-c). Entradas: `proposal.md`, `exploration.md` y las deltas de `specs/` (RF-TOP-19, RF-TOP-20, RF-KOF-17 a RF-KOF-23, RF-GAT-31, RF-GAT-32 y RF-WEB-17). Todas las rutas son relativas a la raíz del repositorio.

## Enfoque técnico

Se sigue el enfoque 1 de la exploración y de la propuesta: superficie mínima en lo compartido, sin sellar nada en `kickoff.yaml` y sin tocar el texto del orquestador.

El diseño tiene un núcleo puro y tres adaptadores finos:

1. **Núcleo puro `internal/acceptancetests`.** Parsea la tabla de `pruebas.md` (RF-KOF-18 y RF-KOF-19), cruza la batería con los requisitos ADDED y MODIFIED de la delta (RF-KOF-20) y reescribe los provisionales de `Cubre` (RF-KOF-23). No lee ficheros ni importa nada de Axiom, salvo `internal/reqid` e `internal/isoevidence`.
2. **Adaptador único `sddstatus.AcceptanceTestsFor`.** Lee el estado vivo de `axiom.yaml`, el kickoff (roles sellados), la delta del cambio y `pruebas.md`, y llama a `acceptancetests.Check`. Lo usan tres consumidores, como `ExplainerFreshness` en C-06-b: la proyección `acceptanceTests` del status, el verbo `axiom sdd tests check` y el DTO del detalle del dashboard. Así la cobertura se calcula en un único sitio y toda función exportada nueva es alcanzable desde `cmd/axiom`.
3. **Digest de la puerta `tasks`.** Una única función, `sddstatus.TasksGateArtifactPaths`, resuelve el conjunto de ficheros que juzga la puerta. La usan las dos rutas espejo: `cli/sdd_gate_digest.go` y `sddstatus/governance.go`. La máquina de puertas (`internal/kickoff/machine.go`, D-08) no se toca.

La declaración vive en un bloque de primer nivel `acceptance_tests:` de `axiom.yaml`, con el patrón de `delivery:` e `integrations:`. La regla de topología se aplica en `workspace.Validate` y, con la misma función, en la proyección del status, para que una declaración que `Validate` rechaza nunca active la doctrina.

## Decisiones de arquitectura

### D1. Bloque `acceptance_tests` de primer nivel con decodificador estricto y regla de topología compartida

**Elección.**
- Tipo `AcceptanceTestsConfig { Enabled bool; Environments []string }` y decodificador `decodeAcceptanceTests` en un fichero nuevo, `internal/workspace/acceptance_tests.go`. Sigue el patrón de `delivery.go`: claves cerradas `acceptanceTestsKeys = {"enabled","environments"}` con `unknownKeys`, rechazo de claves duplicadas, `enabled` booleano estricto (solo el escalar con etiqueta `!!bool`), `environments` como lista de cadenas no vacías y, con `enabled: true`, una lista presente, no vacía y sin duplicados (comparación exacta). Los errores se acumulan con su línea.
- Se registra en `Declarations` (`declarations.go:33-50`) con el par `AcceptanceTests *AcceptanceTestsConfig` y `AcceptanceTestsErr error`, se lee en `readDeclarationBlocks` (`declarations.go:99-142`) con `mappingValue(root, "acceptance_tests")` y se suma en `joinLoadErrors` (`loader.go:43`). Así `ParseConfig`, `LoadConfig` y `axiom workspace validate` son estrictos (RF-TOP-19).
- `WorkspaceConfig` (`types.go`) gana `AcceptanceTests *AcceptanceTestsConfig yaml:"-"`, que `parseLayered` (`loader.go:72`) rellena solo cuando el bloque es válido, como `Delivery`.
- Regla de topología en una única función no exportada, `acceptanceTestsTopologyError(cfg *AcceptanceTestsConfig, topology TopologyType) error`. Devuelve error cuando `Enabled` es verdadero y la topología no es `TopologyMultirepo`. La usan dos sitios:
  - `workspace.Validate`, en el paso 3b (`validator.go:122-138`), justo después de la regla de roles de RF-TOP-04 y con la misma redacción de migración;
  - el accesor exportado `func (d Declarations) EffectiveAcceptanceTests(topology TopologyType) (AcceptanceTestsConfig, error)`. Con `ParseErr`, `AcceptanceTestsErr` o la regla de topología violada devuelve un error que envuelve la causa con `%w` y nunca asume un valor (patrón de `ExplainerHTML()`, `declarations.go:73`). Sin bloque devuelve el valor cero (no activo).
- `invalidDeclarationErrors` (`sddstatus/workspace_declarations.go:206`) **no** se toca: el bloque nunca bloquea el archivado (RF-TOP-20).
- `hub/init.go` `declares()` (`:65-75`) añade `d.AcceptanceTests != nil || d.AcceptanceTestsErr != nil`.

**Alternativas descartadas.**
- Una clave dentro de `sdd:`. `sdd:` tiene claves cerradas y semántica ISO (`workspace/sdd.go:35`), y el plan fija un bloque de primer nivel.
- Rechazar en la carga cuando la topología no es multirepo. `readDeclarationBlocks` no conoce la topología, y RF-TOP-04 fija el precedente de validar en `Validate` para que `Locate` y el archivado toleren el error.
- Aplicar la topología solo en `Validate`. El status proyectaría `active: true` en un monorepo y la skill generaría `pruebas.md` en una topología que la propuesta excluye.

**Motivo.** Es el mismo esquema de los bloques ya existentes, con un canal de error por bloque (D-03). Una sola función para la regla de topología evita que `Validate` y el status discrepen.

### D2. Paquete puro `internal/acceptancetests` sobre los ayudantes exportados de `isoevidence`

**Elección.**
- Paquete nuevo `internal/acceptancetests`, con un `imports_test.go` que fija que solo importa `internal/reqid` e `internal/isoevidence`, además de la biblioteca estándar y nunca `os/exec`. Copia el patrón de `isoevidence/imports_test.go`.
- API mínima en la sección «Interfaces»: `FileName`, `Parse`, `Check` y `RewriteCubre`, con sus tipos.
- Reutilización de `isoevidence` sin duplicar:
  - Se exportan por renombrado, en su sitio, los ayudantes de líneas y tablas: `textLine` pasa a `TextLine`, `splitLines` a `SplitLines` (`fence.go:8,16`), `isTableLine` a `IsTableLine`, `isSeparatorLine` a `IsSeparatorLine`, `splitCells` a `SplitCells` (`matrix.go:192,198,217`) y `rawCellSpan` a `RawCellSpan` (`matrix.go:523`). Son 28 usos internos en 5 ficheros, un cambio mecánico sin cambio de comportamiento.
  - Se usan tal cual `RequirementKeys` (`reqid.go:16`), `Scenarios` (`scenarios.go:12`), `Requirement` y `SectionAdded`/`SectionModified` (`model.go`).
  - **No** se usa `AffectedFromDelta` para el recuento. Deduplica escenarios repetidos (`cubre.go:252-262`) y omite en silencio los requisitos sin un ID único, y RF-KOF-20 exige informar de ambos (`requisito_sin_id` y `escenario_ambiguo`). `Check` recorre la delta con la **misma regla de selección** que `AffectedFromDelta`: ADDED y MODIFIED; en ADDED cuentan los válidos más los provisionales; exactamente un ID. Un test de equivalencia fija que, sin requisitos sin ID ni escenarios repetidos, el conjunto de escenarios de `Check` es igual a `AffectedFromDelta`.
- La cabecera de `pruebas.md` **no** reutiliza `ParseMatrix`. Su regla es otra: nombres exactos y sensibles a mayúsculas, y error ante una columna extra o duplicada (RF-KOF-18), mientras que la matriz admite columnas extra y no distingue mayúsculas.
- Los códigos de hallazgo son un tipo propio, `acceptancetests.Code`, con los quince valores de RF-KOF-20 en castellano. No se amplía `isoevidence.Code` ni `AllCodes()`, que pertenecen a ISO.

**Alternativas descartadas.**
- Duplicar los ayudantes en `acceptancetests`: dos lectores de tabla que divergen con el tiempo.
- Un paquete `internal/mdtable` nuevo: mueve código de un paquete hoja estable y hay que actualizar su `imports_test.go`, por un beneficio que el renombrado ya da.
- Envoltorios exportados sobre las funciones privadas: dos nombres para lo mismo.

**Motivo.** Un solo lector de tablas markdown (`\|`, vallas, celdas `-`/`—`) para la matriz ISO y para la batería. El renombrado no añade código muerto: todo lo exportado lo usa `acceptancetests`, que es alcanzable desde el binario.

### D3. Gramática y semántica de `Check` (cierre de la spec)

**Elección.** La gramática es exactamente la de RF-KOF-18 y RF-KOF-19. Estos son los detalles de implementación que la spec deja abiertos:

- **Tabla.** Es la primera línea de tabla fuera de vallas del fichero (`IsTableLine` sobre `SplitLines`); el resto del texto se ignora. Sin tabla, o con una cabecera con una columna ausente, extra o duplicada, el resultado es `tabla_invalida` y no se evalúan filas. El cuerpo empieza tras la línea separadora y termina en la primera línea que no es de tabla. Si falta la separadora, `tabla_invalida`. Una fila con un número de celdas distinto del de la cabecera da `tabla_invalida` con la línea de esa fila; las demás filas se siguen evaluando.
- **Listas en celda.** `Cubre` y `Entornos` se separan por `;` y cada elemento se recorta. `Cubre` se divide por el primer `#`: lo anterior es el ID y lo posterior, el nombre del escenario, recortado.
- **ID de entrada.** Se reconoce con `pattern.Valid(id)` o como provisional que el patrón admite (`reqid.ScanProvisional` cubre el token entero y `pattern.ParseProvisional` no falla). Cualquier otra forma, o un ID que no está en la delta seleccionada, da `id_desconocido`. Esto incluye los REMOVED y RENAMED, que no entran en la base.
- **Dominios.**
  - `ID` cumple `^P-[0-9]{2,}$`.
  - `Tipo` toma un valor de {`manual`, `automatizable`} y `Prioridad`, uno de {`alta`, `media`, `baja`}. Ambos con comparación exacta.
  - `Rol` es un elemento exacto de los roles sellados en `kickoff.yaml`.
  - `Entornos` es un subconjunto no vacío y sin repetidos de `environments`, con comparación exacta.
  - `Pasos` y `Resultado esperado` no pueden estar vacíos (`celda_invalida`).
- **Recuento.**
  - Un escenario se cubre solo con `ID#nombre`.
  - Un requisito con ID se cubre con su ID suelto o con cualquiera de sus escenarios citados.
  - El total de requisitos incluye los que no tienen ID (`requisito_sin_id`), que nunca cuentan como cubiertos.
  - El total de escenarios cuenta los escenarios de los requisitos con ID; cada nombre repetido cuenta una vez y produce `escenario_ambiguo`.
  - Un escenario cuyo nombre contiene `;` produce `escenario_ambiguo`, porque no se puede citar.
  - Un ID presente en dos requisitos de la delta produce `id_duplicado_en_delta`, y ninguno de los dos entra en la base: no hay cita que los distinga.
- **Ubicación y orden.**
  - Los hallazgos de fila llevan `File: "pruebas.md"` y la línea de la fila.
  - Los de la delta (`requisito_sin_id`, `id_duplicado_en_delta`, `escenario_ambiguo`, `requisito_sin_cubrir` y `escenario_sin_cubrir`) llevan `File: "specs/<dominio>/spec.md"` y `Line: 0`.
  - Se ordenan de forma estable por (fichero, línea, código, asunto). `pruebas_ausente` no lo emite `Check`: lo emite el adaptador (D5).

**Motivo.** El parseo y el recuento son deterministas y cierran los casos ambiguos con un hallazgo, nunca en silencio. La delta no tiene números de línea, porque el lector nativo `openspecdoc` no los conserva, así que el fichero identifica el origen.

### D4. Digest de la puerta `tasks`: una sola función de conjunto de ficheros

**Elección.**
- Nueva función exportada en `internal/sddstatus/acceptance_tests.go`:

  ```go
  func TasksGateArtifactPaths(changeRoot string, roles []kickoff.KickoffRole) []string
  ```

  Devuelve los ficheros de tareas esperados: `tasks.md`, o cada `TasksFile` cuando `kickoff.RoleTaskFiles(roles)` no está vacío. Les añade `filepath.Join(changeRoot, acceptancetests.FileName)` **solo si** `hasContent` (`status.go:1631`) lo da por presente y no vacío. El predicado de inclusión está dentro de la función para que las dos rutas decidan igual sobre un `pruebas.md` vacío o con solo espacios: queda fuera en ambas y no anula el digest de las tareas.
- `cli/sdd_gate_digest.go`: `tasksArtifactPathsForDigest` (`:65-76`) pasa a delegar en `sddstatus.TasksGateArtifactPaths(changeRoot, rolesOf(sealed))`, donde `sealed == nil` da `roles == nil` y, por tanto, `tasks.md`. `digestIfAllPresent` no cambia.
- `sddstatus/governance.go` `governanceArtifactInputs` (`:472-507`): el bloque `tasksPaths` (`:481-484`) se sustituye por `TasksGateArtifactPaths(changeRoot, roles)`; `setGovernanceArtifactDigest` ya aplica la regla de «todos presentes». El digest por rol (`tasks.<rol>`) y el de `role-apply:<rol>` **no** incluyen `pruebas.md`.
- `kickoff.ArtifactDigest` (`kickoff/digest.go:23`) ordena las rutas, así que el resultado no depende del orden de la lista.
- `kickoff/machine.go` (`resolveGateStatus`, `:252-266`) no se toca. Una puerta `rejected` se reabre al cambiar el digest; una `approved` sigue terminal (D-08), y lo fija `TestEvaluateGatesApprovedGateNeverInvalidatedByDigestChange`.

**Alternativas descartadas.**
- Añadir `pruebas.md` a mano en las dos rutas, como pide la letra del plan. Es justo el riesgo de divergencia que documenta `sdd_gate_digest.go:12-30`.
- Poner la función en `kickoff`. Tendría que conocer el nombre del artefacto de `acceptancetests` y el predicado `hasContent` de `sddstatus`, y reproduciría la diferencia de criterio con `fileHasContent` del CLI.
- Añadir `pruebas.md` a `ArtifactPaths` del status. Cambiaría el JSON de `artifactPaths` en todos los status, y la spec exige que ninguna clave existente cambie.

**Motivo.** Sin `pruebas.md`, la lista es idéntica a la actual y el digest de los cambios en curso no cambia (RF-GAT-31). Un test de paridad compara el digest de `gate record` con el del status sobre el mismo fixture.

### D5. Adaptador `sddstatus.AcceptanceTestsFor` y proyección `acceptanceTests`

**Elección.**
- `internal/sddstatus/acceptance_tests.go` define `AcceptanceTestsFor(location workspace.Location, changeRoot string) AcceptanceTestsResult`, con estos pasos:
  1. Calcula la topología con `location.GoverningTopology()` y obtiene `settings` con `location.Declarations.EffectiveAcceptanceTests(topology)`. Con error, rellena `DeclarationError` con la continuación `corrígela con axiom workspace validate --path <master>` y `Active` queda falso.
  2. Sin `changeRoot`, devuelve solo la declaración.
  3. Lee `pruebas.md`. Si falta: con la función activa y las tareas presentes (`TasksGateArtifactPaths` sin `pruebas.md`, con todos los ficheros con contenido), fija `Missing = true`, que da el hallazgo y el aviso `pruebas_ausente`; en otro caso, nada.
  4. Si existe: carga los roles con `kickoff.Load(changeRoot)`, el patrón con `location.Declarations.IDPattern()` y la delta con `readDeltaRequirements(changeRoot, paths.Specs)` (`iso.go:122`), y llama a `acceptancetests.Check`. Un kickoff ilegible, un patrón inaccesible o la ausencia de delta van a `Error`, nunca a un informe parcial.
- `decorateAcceptanceTests(status *Status, location workspace.Location)` se engancha en `Resolve` justo después de `decorateExplainer` (`status.go:559`). Proyecta `Status.AcceptanceTests *AcceptanceTestsProjection` con `json:"acceptanceTests,omitempty"` y la copia en `statusV2` (`status_v2.go:58` y `:270`).
- **Omisión.** La clave se omite salvo que `axiom.yaml` declare el bloque (válido o no), sea ilegible o el cambio seleccionado, no archivado, tenga `pruebas.md`. Un status sin la función es idéntico byte a byte al actual. El contrato documenta que **la ausencia equivale a `active: false`**; ver la pregunta abierta 1.
- El aviso va **dentro** de la proyección (`warning`) y no en `Notes` ni en `blockedReasons`. No toca `nextRecommended`, `dependencies`, `gates` ni los digests (RF-KOF-22).

**Alternativas descartadas.**
- Proyectar siempre `{active:false}`: cambia el JSON de todos los status y sus fixtures de serialización.
- Poner el aviso en `Notes`: los consumidores lo leen como diagnóstico global, y la spec lo pide en `acceptanceTests`.

**Motivo.** Es el patrón de `decorateExplainer` (`explainer.go:123`): proyección aditiva, un solo cálculo compartido con el dashboard y ningún efecto en el enrutado.

### D6. Verbo `axiom sdd tests check`

**Elección.**
- `internal/cli/sdd_tests.go` contiene:
  - `RunSDDTests(args, stdout)`, un subdespachador como `RunSDDGate` (`cli/sdd_gate.go:28`) con un único subcomando, `check`, y su ayuda;
  - `runSDDTestsCheck`, que copia el análisis de banderas de `RunSDDArchiveCheck` (`sdd_archive_check.go:37-71`: `--cwd`, `--change`, `--json`, `gitref.ValidBranchComponent`), resuelve con `resolveGovernanceChangeRoot` (`sdd_governance.go:34`) y llama a `sddstatus.AcceptanceTestsFor`.
- Sin specs delta (`len(paths.Specs) == 0`) es un error de uso (RF-KOF-21).
- Salida JSON con el esquema `axiom.sdd-tests-check/v1` y estos campos: `schema`, `change`, `active`, `environments`, `present`, `cases`, `scenarios {covered,total}`, `requirements {covered,total}`, `findings [{code,file,line,message}]` y `error?`. `file` es un campo adicional al mínimo de la spec, necesario para el orden de D3.
- Códigos de salida. El despachador de `cmd/axiom` traduce todo error a 1, así que hay dos:

  | Situación | Salida |
  |---|---|
  | Sin hallazgos (informe limpio) | 0 |
  | Función no activa y sin `pruebas.md` | 0, con el mensaje «las pruebas de aceptación no están activadas» |
  | Función activa, antes de `tasks` y sin `pruebas.md` | 0, con «todavía no hay batería» |
  | Algún hallazgo, incluido `pruebas_ausente` | 1, tras imprimir el informe completo (texto o JSON) |
  | Bloque inválido, regla de topología, kickoff ilegible, patrón inaccesible, sin delta o cambio inexistente | 1, con la continuación |
- Registro:
  - `cmd/axiom/main.go` (`runSDD`, `:1986-2050`): una línea de ayuda (`tests            Comprueba la batería de pruebas de aceptación de un cambio (check)`), el `case "tests": err = cli.RunSDDTests(subArgs, stdout)` y la lista de opciones del mensaje de error.
  - `cmd/axiom/main_test.go`: `tests` en `expectedSubcmds` (`:52`) y un test de enrutado de `tests check` sin `--change` que espera `requiere --change`, con el patrón de `TestSDDVerbRouting` (`:67`).

**Motivo.** Es la misma forma que `archive-check`: solo lectura, el informe se imprime antes del error y la continuación es explícita.

### D7. Archivado: tercer tipo de fichero en `livespec`

**Elección.**
- `internal/livespec/change.go` añade `kindAcceptanceTests` a `changeFileKind` (`:59-63`).
- `changeArtifacts` (`:207-247`) clasifica `name == acceptancetests.FileName` y extrae las citas de los provisionales de cada entrada de `Cubre`, en la parte del ID, con la línea de la fila de `acceptancetests.Parse`.
- En la reescritura (`:176-187`), ese tipo llama a `acceptancetests.RewriteCubre(text, idPlan.Map)`, que hace esto:
  1. localiza la columna `Cubre` con la misma selección de tabla que `Parse`;
  2. obtiene el intervalo crudo de la celda de cada fila con `isoevidence.RawCellSpan`;
  3. divide la celda por `;` y, en cada entrada, aplica `m.Substitute` **solo** al tramo anterior al primer `#`.

  El nombre del escenario, el resto de columnas, el texto fuera de la tabla y los finales de línea quedan intactos. Sin mapa o sin tabla devuelve el texto igual, y con un texto ya reescrito no cambia nada (idempotente).

**Motivo.** Así no quedan citas huérfanas tras integrar el cambio (RF-KOF-23). Se reproduce la garantía byte a byte de `RewriteMatrix` (`matrix.go:550`) y se protege el nombre del escenario, que `Substitute` sobre la celda entera podría alterar.

### D8. Dashboard: campo del detalle y pestaña «Pruebas», sin ruta nueva

**Elección.**
- `internal/dashboard/types.go`: `IncrementDetailDTO` (`:85-104`) gana `AcceptanceTests IncrementAcceptanceTestsDTO json:"acceptance_tests"`, que nunca es nulo, como `Explainer`. Tipos nuevos:
  - `IncrementAcceptanceTestsDTO`, con `active`, `environments`, `error`, `present`, `warning`, `cases`, `scenarios`, `requirements`, `case_count` y `findings`;
  - `AcceptanceCaseDTO`, con las nueve columnas y la línea;
  - `AcceptanceFindingDTO`, con `code`, `file`, `line` y `message`.
- `internal/dashboard/service.go`: `GetIncrementDetail` (`:530`) llama a un `s.incrementAcceptanceTests(path)` nuevo. Este localiza con `workspace.Locate(s.getRootPath())`, como `incrementExplainer` (`:1749`), y mapea `sddstatus.AcceptanceTestsFor`. Si `Locate` falla, el DTO lleva `error` y el detalle se sirve igual.
- Front, todo de solo lectura:
  - `assets/index.html`: pestaña `data-tab="pruebas"` con el rótulo «Pruebas», detrás de «Explicación» y siempre visible, y su panel `data-panel="pruebas"`;
  - `assets/js/core/router.js`: `'pruebas'` en `INCREMENT_TABS` (`:13`);
  - `assets/js/screens/detalle.js`: rama `tab === 'pruebas'` en `renderTabContent` (`:205`);
  - módulo nuevo `assets/js/screens/detalle-pruebas.js` con `renderAcceptanceTests(panel, { detail })`. Pinta el resumen de cobertura (escenarios M/M, requisitos M/M y casos), el aviso, los hallazgos y una tabla de casos construida a partir de `cases`, con cada celda pasada por `escapeHtml` (`core/api.js`). No usa `renderMarkdown`, para que todo texto de `pruebas.md` se inserte como texto (RF-WEB-17).
  - Estados vacíos con `stateEmpty`/`stateError` (`core/components.js`): «Las pruebas de aceptación no están activadas para este proyecto», «Todavía no hay batería de pruebas» y el aviso de ausencia.
- **Sin ruta nueva.** `/api/increments/` ya está clasificada en `secrets_canary_test.go`, y no se tocan `excludedRoutes` ni la lista blanca de ficheros de RF-WEB-15.

**Motivo.** Se reutilizan el DTO de C-06-a y el cálculo compartido de C-06-b. Una tabla construida con datos ya parseados y escapados elimina el riesgo de inyección y no depende de que `markdown.js` soporte tablas con `\|`.

### D9. Doctrina: skill `sdd-tasks`, compartido nuevo y agentes delgados

**Elección.**
- Nuevo `internal/assets/skills/_shared/sdd-acceptance-tests.md` con la doctrina:
  - **Condición.** Solo en SDD, solo con `acceptanceTests.active: true` en el status, y nunca en ODD.
  - **Formato.** La tabla de RF-KOF-18 y RF-KOF-19, con un ejemplo.
  - **Minimización.** Las cuatro capas en su orden.
  - **Comprobación.** Ejecutar `axiom sdd tests check --cwd <workspaceRoot> --change <c>` y corregir hasta que no haya hallazgos, o dejar constancia de los pendientes.
  - **Presentación.** Devolver `pruebas.md` en `artifacts` y el resultado del check en el resumen, para que la parada `tasks` lo presente junto a los ficheros de tareas (RF-GAT-32).
  - **Ausencia.** Con la función activa y sin fichero, declararlo.
  - **Refinado de `sdd-apply`.** Solo puede refinar rutas o URL de `Pasos`.
- `internal/assets/skills/sdd-tasks/SKILL.md`: un paso nuevo tras «Step 3: Write tasks.md» (`:62`), «Step 3b: Acceptance test battery», que remite al compartido con `../_shared/sdd-acceptance-tests.md`, como hacen `sdd-spec` y `sdd-design` con `sdd-explainer-html.md`.
- Agentes delgados: una sección `## Acceptance Tests` de 2 o 3 líneas en `internal/assets/claude/agents/sdd-tasks.md`, `internal/assets/cursor/agents/sdd-tasks.md` y `internal/assets/copilot/agents/sdd-tasks.agent.md`, con el precedente de `## Explainer HTML` en los agentes `sdd-spec` y `sdd-design`.
- Test de pines nuevo, `internal/assets/acceptance_tests_doctrine_test.go`, modelado en `explainer_html_doctrine_test.go`. Fija:
  - en el compartido: la condición SDD con `acceptanceTests` activo, que ODD nunca genera, las cuatro capas en orden, `axiom sdd tests check`, la regla de refinado de `sdd-apply` y las nueve columnas;
  - en la skill: el enlace al compartido;
  - en los tres agentes: la sección.
- Golden: solo cambia `testdata/golden/sdd-claude-agent-sdd-tasks.golden`, regenerado con `go test ./internal/components -run TestGolden -update`.
- No se tocan `_shared/sdd-orchestrator-sections.md`, sus 10 goldens, la skill `sdd-apply` ni las copias instaladas en `.claude/skills/`.
- Los ficheros de `_shared/` se embeben con `//go:embed all:skills` (`internal/assets/assets.go:5`): no hay manifiesto que registrar.
- `internal/assets/skills/_shared/sdd-status-contract.md`: un párrafo `acceptanceTests` junto al de `explainer` (`:192-200`).

**Motivo.** La parada `tasks` presenta `pruebas.md` porque así lo dicen la skill y el status, sin editar el texto del orquestador. Es la decisión 3 de la propuesta: así se evitan los 10 goldens y el ratchet de deriva.

## Flujo de datos

```
axiom.yaml ─(readDeclarationBlocks)→ Declarations.AcceptanceTests / AcceptanceTestsErr
     │                                         │
     │                       EffectiveAcceptanceTests(topology) ◄── Validate 3b (misma regla)
     ▼                                         ▼
kickoff.yaml (roles) ──┐        sddstatus.AcceptanceTestsFor(location, changeRoot)
specs/**/spec.md ──────┼──────────────► acceptancetests.Check(text, Input) ─► Report
pruebas.md ────────────┘                                                     │
                                          ┌──────────────────┬───────────────┤
                                          ▼                  ▼               ▼
                              status.acceptanceTests   sdd tests check   IncrementDetailDTO
                                 (decorate, aditivo)    (exit 0/1)       .acceptance_tests ─► detalle-pruebas.js

tasks*.md + pruebas.md? ─► sddstatus.TasksGateArtifactPaths ─┬─► cli gate record (digestIfAllPresent)
                                                            └─► governanceArtifactInputs (status)
                                                                      └─► kickoff.EvaluateGates (sin cambios, D-08)

archive-integrate ─► livespec.changeArtifacts ─► kindAcceptanceTests ─► acceptancetests.RewriteCubre
```

## Cambios de ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/workspace/acceptance_tests.go` | Crear | `AcceptanceTestsConfig`, `decodeAcceptanceTests`, `acceptanceTestsTopologyError`, `Declarations.EffectiveAcceptanceTests` |
| `internal/workspace/acceptance_tests_test.go` | Crear | Decodificador estricto, accesor y regla de topología |
| `internal/workspace/declarations.go` | Modificar | Par bloque/error y lectura en `readDeclarationBlocks` |
| `internal/workspace/types.go` | Modificar | `WorkspaceConfig.AcceptanceTests yaml:"-"` |
| `internal/workspace/loader.go` | Modificar | `joinLoadErrors` y `parseLayered` proyectan el bloque; comentario de claves |
| `internal/workspace/validator.go` | Modificar | Paso 3b: `enabled: true` fuera de multirepo |
| `internal/workspace/validator_test.go`, `example_yaml_test.go`, `docs_contract_test.go` | Modificar | Casos de topología y contrato de plantilla y manual |
| `internal/hub/init.go` (+ test) | Modificar | `declares()` conoce el bloque |
| `axiom.example.yaml`, `docs/manual-de-inicio.md`, `docs/openspec-config.md` | Modificar | Bloque comentado, documentación y verbo `sdd tests check` |
| `internal/isoevidence/fence.go`, `matrix.go`, `bug.go`, `cubre.go`, `scenarios.go` | Modificar | Exportar `TextLine`, `SplitLines`, `IsTableLine`, `IsSeparatorLine`, `SplitCells` y `RawCellSpan` (renombrado mecánico) |
| `internal/acceptancetests/{doc,model,parse,check,rewrite}.go` | Crear | Núcleo puro |
| `internal/acceptancetests/*_test.go`, `imports_test.go` | Crear | Unitarias, una por hallazgo, determinismo, equivalencia con `AffectedFromDelta`, reescritura y arquitectura de imports |
| `internal/sddstatus/acceptance_tests.go` (+ test) | Crear | `TasksGateArtifactPaths`, `AcceptanceTestsFor`, `decorateAcceptanceTests` y tipos de proyección |
| `internal/sddstatus/governance.go` | Modificar | `governanceArtifactInputs` usa `TasksGateArtifactPaths` |
| `internal/sddstatus/status.go`, `status_v2.go` | Modificar | Campo `AcceptanceTests`, enganche en `Resolve` y copia en v2 |
| `internal/cli/sdd_gate_digest.go` | Modificar | `tasksArtifactPathsForDigest` delega en `sddstatus.TasksGateArtifactPaths` |
| `internal/cli/sdd_gate_artifact_digest_test.go` | Modificar | Paridad, digest invariable sin `pruebas.md`, reapertura de la rechazada y aprobada terminal |
| `internal/cli/sdd_tests.go` (+ `sdd_tests_test.go`) | Crear | Verbo `sdd tests check` |
| `cmd/axiom/main.go`, `cmd/axiom/main_test.go` | Modificar | Registro, ayuda y enrutado |
| `internal/livespec/change.go` (+ test) | Modificar | `kindAcceptanceTests`, citas y reescritura |
| `internal/dashboard/types.go`, `service.go` (+ `service_test.go`) | Modificar | DTO y cálculo compartido |
| `internal/dashboard/assets/index.html`, `js/core/router.js`, `js/screens/detalle.js` | Modificar | Pestaña «Pruebas» |
| `internal/dashboard/assets/js/screens/detalle-pruebas.js` | Crear | Render de solo lectura con `escapeHtml` |
| `internal/dashboard/assets_test.go`, `workflow_assets_test.go` | Modificar | Pines de pestaña, rótulo, orden y ausencia de controles de escritura |
| `internal/assets/skills/_shared/sdd-acceptance-tests.md` | Crear | Doctrina común |
| `internal/assets/skills/sdd-tasks/SKILL.md` | Modificar | Paso 3b |
| `internal/assets/skills/_shared/sdd-status-contract.md` | Modificar | Contrato de `acceptanceTests` |
| `internal/assets/{claude,cursor}/agents/sdd-tasks.md`, `internal/assets/copilot/agents/sdd-tasks.agent.md` | Modificar | Sección `## Acceptance Tests` |
| `internal/assets/acceptance_tests_doctrine_test.go` | Crear | Test de pines |
| `testdata/golden/sdd-claude-agent-sdd-tasks.golden` | Regenerar | Con `-update` |

## Interfaces y contratos

```go
// package acceptancetests
const FileName = "pruebas.md"

type Code string // pruebas_ausente, tabla_invalida, celda_invalida, id_caso_invalido,
// id_caso_duplicado, caso_sin_cubre, entorno_invalido, rol_invalido, id_desconocido,
// escenario_desconocido, requisito_sin_id, id_duplicado_en_delta, escenario_ambiguo,
// requisito_sin_cubrir, escenario_sin_cubrir

type CubreEntry struct{ ID, Scenario string } // Scenario vacío = cita del requisito
type Case struct {
	Line                                                        int
	ID, Role, Type, Priority, Preconditions, Steps, Expected    string
	Cubre        []CubreEntry
	CubreRaw     string
	Environments []string
}
type Battery struct {
	Cases  []Case
	Issues []Finding // forma: tabla_invalida y celdas con número de celdas distinto
}
type Finding struct {
	Code          Code
	File          string
	Line          int
	Subject, Message string
}
type Coverage struct {
	Covered int `json:"covered"`
	Total   int `json:"total"`
}
type Input struct {
	Delta        []isoevidence.Requirement // ADDED/MODIFIED/REMOVED/RENAMED; Check filtra
	Pattern      reqid.Pattern
	Environments []string
	Roles        []string
}
type Report struct {
	Battery      Battery
	Cases        int
	Scenarios    Coverage
	Requirements Coverage
	Findings     []Finding // ordenados (fichero, línea, código, asunto)
}

func (r Report) OK() bool
func Parse(text string) Battery
func Check(text string, in Input) Report
func RewriteCubre(text string, m reqid.IDMap) string
func MissingFinding() Finding // pruebas_ausente; la usa AcceptanceTestsFor
```

```go
// package workspace
type AcceptanceTestsConfig struct {
	Enabled      bool
	Environments []string
}

func (d Declarations) EffectiveAcceptanceTests(topology TopologyType) (AcceptanceTestsConfig, error)
```

```go
// package sddstatus
func TasksGateArtifactPaths(changeRoot string, roles []kickoff.KickoffRole) []string

type AcceptanceTestsResult struct {
	Active           bool
	Environments     []string
	DeclarationError string // bloque inválido o topología, con su continuación
	Present          bool
	Missing          bool   // activa + tareas presentes + sin pruebas.md
	Path             string
	Error            string // kickoff, patrón o delta ilegibles
	Report           *acceptancetests.Report
}

func AcceptanceTestsFor(location workspace.Location, changeRoot string) AcceptanceTestsResult
```

JSON del status (clave aditiva y omitida en los casos de D5):

```yaml
acceptanceTests:
  active: true | false
  environments: [dev, pre]        # solo con active
  error: "<motivo>; corrígela con `axiom workspace validate --path <master>`"  # bloque o topología inválidos
  warning: "pruebas_ausente: ..." # activa, tasks terminado y sin pruebas.md
  battery:                        # solo si existe pruebas.md en el cambio seleccionado no archivado
    path: <ruta absoluta>
    cases: 4
    scenarios: { covered: 5, total: 5 }
    requirements: { covered: 2, total: 2 }
    findings: 0
    error: "<fallo de lectura>"   # opcional
```

## Estrategia de pruebas

Todo se prueba localizado por paquete y con `-run`; la batería completa queda para la CI. Los fixtures multirepo salen de `workspace/workspacetest/fixtures.go:29`, porque este repositorio es `monorepo-embedded` y no puede activar la función.

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unitaria `workspace` | Decodificador estricto (claves, duplicadas, `enabled` no booleano, entornos vacíos o duplicados, `enabled:false` sin entornos), accesor sin asumir valor y `Validate` en monorepo frente a multirepo | `go test ./internal/workspace -run 'AcceptanceTests\|Validate\|ExampleYAML\|DocsContract'` |
| Unitaria `isoevidence` | El renombrado no cambia el comportamiento | `go test ./internal/isoevidence` (paquete pequeño) |
| Unitaria `acceptancetests` | Un test por código de hallazgo de RF-KOF-20, gramática de RF-KOF-18 y RF-KOF-19 (orden de columnas, `\|`, comas en el escenario, provisionales y `id_pattern`), equivalencia con `AffectedFromDelta`, determinismo, `RewriteCubre` byte a byte e idempotente, e imports | `go test ./internal/acceptancetests` |
| Integración `sddstatus` | Omisión sin bloque (JSON idéntico), proyección activa, inválida y de topología, aviso sin cambio de `nextRecommended` ni dependencias, y `TasksGateArtifactPaths` sin `pruebas.md` igual a hoy | `go test ./internal/sddstatus -run 'AcceptanceTests\|TasksGate\|Governance'` |
| Integración `cli` | Paridad del digest entre `gate record` y el status, digest distinto al editar `pruebas.md`, rechazada que pasa a `pending` y aprobada que sigue `approved`, y `tests check` en texto y JSON con los códigos de salida | `go test ./internal/cli -run 'GateArtifactDigest\|SDDTests'` |
| Integración `cmd/axiom` | Ayuda y enrutado | `go test ./cmd/axiom -run 'SDDHelp\|SDDVerbRouting\|SDDTests'` |
| Integración `livespec` | Reescritura de provisionales en `Cubre`, sin `pruebas.md`, idempotencia e IDs definitivos intactos | `go test ./internal/livespec -run 'AcceptanceTests\|ChangeArtifacts'` |
| Integración `dashboard` | DTO con casos y cobertura igual a la del verbo, contenido hostil como texto, pines de pestaña, sin ruta nueva y canario en verde | `go test ./internal/dashboard -run 'IncrementDetail\|AcceptanceTests\|Tabs\|Router\|SecretsCanary'`. Los tests con `freePort` o el hub compartido no usan `t.Parallel` |
| Doctrina y golden | Pines del compartido, la skill y los agentes; golden de claude | `go test ./internal/assets -run AcceptanceTestsDoctrine`; `go test ./internal/components -run TestGolden -update`, y después sin `-update` |
| Ratchets | Código muerto y deriva del orquestador | Los ejecuta la CI (`scripts/deadcode-ratchet.sh` y `TestOrchestratorSectionDriftDoesNotGrow`). En local, solo si una unidad toca exportaciones nuevas, y sin compilar binarios sueltos |

## Orden sugerido de unidades de trabajo (para sdd-tasks)

1. **U1. Bloque `acceptance_tests` en `workspace`**, con `hub/init.go`, la plantilla y los documentos. Es independiente del resto.
2. **U2. Exportación de los ayudantes de `isoevidence`** (renombrado mecánico, sin cambio de comportamiento).
3. **U3. Núcleo `internal/acceptancetests`**: `Parse`, `Check` y `RewriteCubre`, con tests. Depende de U2.
4. **U4. Digest compartido**: `TasksGateArtifactPaths`, refactor de las dos rutas y tests de paridad y de reapertura. Depende de U3 por `FileName`.
5. **U5. `AcceptanceTestsFor` y proyección `acceptanceTests`** en el status y su contrato. Depende de U1, U3 y U4.
6. **U6. Verbo `sdd tests check`** y su registro en `cmd/axiom`. Depende de U5.
7. **U7. Archivado en `livespec`**. Depende de U3.
8. **U8. Dashboard: DTO y pestaña «Pruebas»**. Depende de U5.
9. **U9. Doctrina**: compartido, skill, agentes, pines y golden. Depende de U6, porque cita el verbo.

Hay un commit por unidad. U7 puede ir en paralelo a U5, U6 y U8.

## Matriz de amenazas

N/A. El cambio no altera el enrutado, no ejecuta órdenes de shell ni subprocesos, no automatiza VCS ni PR y no clasifica ficheros ejecutables. El verbo nuevo es de solo lectura, y la única escritura es la reescritura de `pruebas.md` dentro del `archive-integrate` existente. El único límite de confianza es la inserción en el DOM de texto de `pruebas.md` en el dashboard. Se cubre con `escapeHtml` en cada celda y con un test de contenido hostil (RF-WEB-17), sin filas de la matriz.

## Migración y despliegue

No hay migración. Sin el bloque, o con `enabled: false`, nada cambia: no hay proyección, el digest es el mismo y no hay aviso. Los cambios en curso sin `pruebas.md` conservan su digest de `tasks`. Un binario anterior ignora el bloque de primer nivel. Para revertir, basta revertir la PR, como describe la propuesta.

## Preguntas abiertas

- [ ] **1. Proyección con la función no activa.** RF-KOF-22 dice «DEBE proyectar la función como no activa». El diseño omite la clave sin bloque (la ausencia equivale a `active: false`, documentado en el contrato) y proyecta `active: false` cuando el bloque declara `enabled: false`. Así un status sin la función queda idéntico byte a byte, como hizo C-06-b con `explainer`. Si la spec exige la clave siempre presente, hay que regenerar los fixtures de serialización del status. Recomendación: precisar en la spec «ausente o `active: false`».
- [ ] **2. JSON del verbo.** El diseño añade `file` a cada hallazgo y `schema`, `change` y `error?` al objeto de RF-KOF-21. Son claves adicionales; conviene que la spec las admita o diga «al menos».
- [ ] **3. Nombres de entorno con `;`.** RF-TOP-19 solo exige cadenas no vacías. Un entorno con `;` no se puede citar en `Entornos`. Opciones: rechazarlo en el decodificador (recomendado, una línea más en RF-TOP-19) o dejarlo como hallazgo implícito de `entorno_invalido`.
