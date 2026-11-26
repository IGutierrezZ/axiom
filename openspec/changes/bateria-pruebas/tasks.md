# Tareas: batería de pruebas de aceptación (`pruebas.md`) generada en sdd-tasks

Cambio `bateria-pruebas` (C-06-c). Rol único `fullstack`, estilo `continuous`, `delivery_strategy: single-pr` (un commit de unidad de trabajo por tarea). Las rutas son relativas a la raíz del repositorio, salvo indicación. La spec manda sobre el design donde difieran.

## Delivery Forecast

| Field | Value |
|-------|-------|
| Estimated size | XL |
| Estimated changed lines | 3.800-5.200 (altas y bajas, sin ficheros generados; los tests son aproximadamente la mitad) |

### Work Units

| Unit | Goal | Commit | Focused test command | Runtime harness | Rollback boundary |
|------|------|--------|----------------------|-----------------|-------------------|
| T1 | Bloque `acceptance_tests` en `workspace` | `feat(workspace): declarar el bloque acceptance_tests solo en multirepo` | `go test ./internal/workspace ./internal/hub -run 'AcceptanceTests\|Validate\|ExampleYAML\|DocsContract\|Declares' -count=1` | `ParseConfig` sobre fixtures multirepo (`workspacetest`) en los tests | Ficheros de `internal/workspace` del bloque, `hub/init.go`, plantilla y docs |
| T2 | Exportar los ayudantes de `isoevidence` | `refactor(isoevidence): exportar los ayudantes de lectura de tablas` | `go test ./internal/isoevidence ./internal/livespec -count=1` | N/A: renombrado mecánico sin cambio de comportamiento | Revertir el renombrado |
| T3 | Núcleo `internal/acceptancetests` | `feat(acceptancetests): parsear y cruzar pruebas.md con las specs delta` | `go test ./internal/acceptancetests -count=1` | Tests sobre `pruebas.md` y delta en memoria | Paquete nuevo completo |
| T4 | Digest de la puerta `tasks` compartido | `feat(sddstatus): calcular una sola vez el conjunto de ficheros del digest tasks` | `go test ./internal/sddstatus ./internal/cli -run 'TasksGate\|Governance\|GateArtifactDigest' -count=1` | Test de paridad `gate record` frente a status sobre el mismo fixture | `TasksGateArtifactPaths` y sus dos llamadores |
| T5 | `AcceptanceTestsFor` y proyección `acceptanceTests` | `feat(sddstatus): proyectar acceptanceTests de forma aditiva` | `go test ./internal/sddstatus -run 'AcceptanceTests' -count=1` | Status v2 sobre fixture multirepo con y sin `pruebas.md` | Fichero nuevo y campo aditivo del status |
| T6 | Verbo `axiom sdd tests check` | `feat(cli): añadir axiom sdd tests check` | `go test ./internal/cli -run 'SDDTests' -count=1` y `go test ./cmd/axiom -run 'SDDHelp\|SDDVerbRouting\|SDDTests' -count=1` | `RunSDDTests` en proceso (texto y JSON) | `sdd_tests.go` y el registro en `cmd/axiom` |
| T7 | Archivado reescribe `Cubre` de `pruebas.md` | `feat(livespec): reescribir los provisionales de pruebas.md al archivar` | `go test ./internal/livespec -run 'AcceptanceTests\|ChangeArtifacts\|PlanChange' -count=1` | `PlanChange` sobre un cambio temporal con `pruebas.md` | Rama `kindAcceptanceTests` |
| T8 | Dashboard: DTO y pestaña «Pruebas» | `feat(dashboard): mostrar la batería de pruebas en el detalle del incremento` | `go test ./internal/dashboard -run 'IncrementDetail\|AcceptanceTests\|Tabs\|Router\|Canary' -count=1` | Test del servicio con hub propio, sin `t.Parallel` | DTO, JS de la pestaña y pines |
| T9 | Doctrina `sdd-tasks`, compartido, agentes, pines y golden | `feat(assets): doctrina de la batería de pruebas de aceptación en sdd-tasks` | `go test ./internal/assets -run AcceptanceTestsDoctrine -count=1` y `go test ./internal/components -run TestGolden -count=1` | Pines sobre los ficheros embebidos | Compartido, skill, agentes y un golden |
| T10 | Cierre: ratchet de código muerto y canario | sin commit propio salvo que corrija algo (`fix(...)`) | `scripts/deadcode-ratchet.sh`; `go test ./internal/dashboard -run Canary -count=1` | N/A | N/A |

Restricciones de pruebas comunes a todas las tareas: solo pruebas localizadas por paquete tocado con `go test <pkg> -run <patrón> -count=1`; NUNCA `go test ./...` ni `internal/cli` entero; `freePort` y el hub compartido no admiten `t.Parallel`; goldens con `go test ./internal/components -run TestGolden -update` y, para el golden del agente, el test que lo pinea; `go vet <paquetes tocados>`; no compilar binarios sueltos (`go build -o`), solo `go build ./...` o tests. Cada tarea cierra con su commit Conventional (sin atribución de IA ni `Co-Authored-By`) y con los tests y la documentación del comportamiento dentro del mismo commit.

Orden: T1, T2 independientes entre sí; T3 tras T2; T4 tras T3; T5 tras T1, T3 y T4; T6 tras T5; T7 tras T3 (puede ir en paralelo a T5, T6 y T8); T8 tras T5; T9 tras T6; T10 al final. Con un único escritor se ejecutan en orden T1 a T10.

## Phase 1: Fundamentos

- [x] T1 Bloque `acceptance_tests` estricto en `workspace`, con `hub`, plantilla y manual.
  - Crear `internal/workspace/acceptance_tests.go`: `AcceptanceTestsConfig{Enabled bool; Environments []string}`, `decodeAcceptanceTests` (claves cerradas `enabled` y `environments` con `unknownKeys`, rechazo de claves duplicadas, `enabled` solo escalar `!!bool`, `environments` lista de cadenas no vacías, sin duplicados exactos y sin `;`; con `enabled: true` la lista debe existir y no estar vacía; con `enabled: false` no exige entornos). Un bloque nulo o que no es un mapeo es un error tipado (RF-TOP-19). Errores acumulados con línea.
  - En el mismo fichero, `acceptanceTestsTopologyError(cfg, topology)` (error si `Enabled` y la topología no es `TopologyMultirepo`) y `Declarations.EffectiveAcceptanceTests(topology) (AcceptanceTestsConfig, error)`: con `ParseErr`, `AcceptanceTestsErr` o regla de topología violada devuelve error envuelto con `%w` y nunca asume un valor; sin bloque devuelve el valor cero.
  - Modificar `internal/workspace/declarations.go` (par `AcceptanceTests`/`AcceptanceTestsErr` y lectura con `mappingValue(root, "acceptance_tests")` en `readDeclarationBlocks`), `internal/workspace/types.go` (`WorkspaceConfig.AcceptanceTests yaml:"-"`), `internal/workspace/loader.go` (`joinLoadErrors`, `parseLayered` proyecta solo con bloque válido, comentario de claves) y `internal/workspace/validator.go` (paso 3b: `enabled: true` fuera de multirepo, misma redacción de migración que RF-TOP-04).
  - Modificar `internal/hub/init.go` `declares()` (`d.AcceptanceTests != nil || d.AcceptanceTestsErr != nil`) para que `init --force` conserve el bloque.
  - Documentar: `axiom.example.yaml` (bloque comentado), `docs/manual-de-inicio.md` y `docs/openspec-config.md` (bloque y verbo `sdd tests check`).
  - NO tocar `invalidDeclarationErrors` (`internal/sddstatus/workspace_declarations.go`): el bloque nunca bloquea el archivado.
  - Tests: crear `internal/workspace/acceptance_tests_test.go` (claves desconocidas, duplicadas, `enabled` no booleano, bloque nulo, bloque no mapeo, `;` en entorno, entornos vacíos o duplicados, `enabled:false` sin entornos, accesor sin asumir valor); casos de topología en `internal/workspace/validator_test.go` (monorepo rechaza `enabled: true`, admite `enabled: false` y ausencia; multirepo válido con fixtures de `internal/workspace/workspacetest/fixtures.go`); `example_yaml_test.go` y `docs_contract_test.go`; test de `internal/hub` para `declares()` y `init --force`; test de `Locate` con bloque inválido que expone el error sin fallar (RF-TOP-20).
  - Ficheros: `internal/workspace/{acceptance_tests,acceptance_tests_test,declarations,types,loader,validator,validator_test,example_yaml_test,docs_contract_test}.go`, `internal/hub/init.go` (+ test), `axiom.example.yaml`, `docs/manual-de-inicio.md`, `docs/openspec-config.md`.
  - Cubre: RF-TOP-19, RF-TOP-20 (escenarios `Locate` con bloque inválido, `init --force` conserva el bloque, plantilla y manual).
  - Verificación: `go test ./internal/workspace ./internal/hub -run 'AcceptanceTests|Validate|ExampleYAML|DocsContract|Declares|Locate' -count=1`; `go vet ./internal/workspace ./internal/hub`.
  - Commit: `feat(workspace): declarar el bloque acceptance_tests solo en multirepo`.

- [x] T2 Exportar los ayudantes de líneas y tablas de `isoevidence` (renombrado mecánico).
  - Renombrar en su sitio: `textLine` a `TextLine`, `splitLines` a `SplitLines` (`fence.go`), `isTableLine` a `IsTableLine`, `isSeparatorLine` a `IsSeparatorLine`, `splitCells` a `SplitCells`, `rawCellSpan` a `RawCellSpan` (`matrix.go`) y actualizar los usos internos (`bug.go`, `cubre.go`, `scenarios.go`, `matrix.go`, `fence.go`). Sin cambio de comportamiento.
  - Tests: los existentes de `isoevidence` y `livespec` siguen en verde; añadir comentario de doc a cada símbolo exportado.
  - Ficheros: `internal/isoevidence/{fence,matrix,bug,cubre,scenarios}.go` y sus tests si usan los nombres antiguos.
  - Cubre: RF-KOF-18, RF-KOF-19 (base de lectura de tablas compartida, D2).
  - Verificación: `go test ./internal/isoevidence ./internal/livespec -count=1`; `go vet ./internal/isoevidence`; `go build ./...`.
  - Commit: `refactor(isoevidence): exportar los ayudantes de lectura de tablas`.

## Phase 2: Núcleo

- [x] T3 Núcleo puro `internal/acceptancetests` (`Parse`, `Check`, `RewriteCubre`).
  - Crear `internal/acceptancetests/{doc,model,parse,check,rewrite}.go` con la API del design (`FileName`, `Code` con los 15 códigos de RF-KOF-20, `CubreEntry`, `Case`, `Battery`, `Finding`, `Coverage`, `Input`, `Report`, `Report.OK()`, `Parse`, `Check`, `RewriteCubre`, `MissingFinding`).
  - Gramática (RF-KOF-18, RF-KOF-19, D3): tabla única fuera de vallas; sin tabla, cabecera con columna ausente, extra, duplicada o con otra capitalización, o sin separadora: `tabla_invalida`; una segunda tabla fuera de vallas produce `tabla_invalida`; una tabla sin filas de datos da cero casos y los escenarios quedan sin cubrir; fila con número de celdas distinto: `tabla_invalida` con su línea y el resto de filas se evalúa. `Cubre` y `Entornos` se separan por `;`; `Cubre` se divide por el primer `#`. `ID#` con escenario vacío es `caso_sin_cubre` (entrada vacía). Una cita repetida en la misma celda o entre casos no es hallazgo y cuenta una sola vez en la cobertura. Dominios de `ID` (`^P-[0-9]{2,}$`), `Tipo`, `Prioridad`, `Rol` (roles sellados), `Entornos` (subconjunto no vacío y sin repetidos), `Pasos` y `Resultado esperado` no vacíos.
  - ID de entrada: `pattern.Valid(id)` o provisional admitido (`reqid.ScanProvisional` y `pattern.ParseProvisional`); cualquier otro, o ausente de la delta seleccionada (incluidos REMOVED y RENAMED), da `id_desconocido`.
  - Recuento (D3): recorre la delta con la misma selección que `AffectedFromDelta` (ADDED con válidos más provisionales, MODIFIED, exactamente un ID) pero informa de `requisito_sin_id`, `id_duplicado_en_delta` y `escenario_ambiguo` (nombre repetido o con `;`); los hallazgos de delta llevan `File: specs/<dominio>/spec.md` y `Line: 0`; orden estable por (fichero, línea, código, asunto). `pruebas_ausente` no lo emite `Check`.
  - Entorno: `Input.Environments` vacío o no declarado no debe usarse aquí para inventar un dominio; el adaptador (T5) decide si valida `Entornos` (ver T5). `Check` recibe un indicador explícito para omitir la validación de `Entornos` del dominio (campo `SkipEnvironments bool` en `Input`).
  - `RewriteCubre(text, m reqid.IDMap)`: misma selección de tabla que `Parse`, `RawCellSpan` por celda de `Cubre`, divide por `;` y aplica `m.Substitute` solo al tramo anterior al primer `#`; nombre de escenario, resto de columnas, texto fuera de la tabla y finales de línea intactos; idempotente; sin mapa o sin tabla devuelve el texto igual.
  - Tests (`internal/acceptancetests/*_test.go`): uno por código de hallazgo (los 15), todos los escenarios de RF-KOF-18, RF-KOF-19 y RF-KOF-20 (orden de columnas, `\|`, comas en el escenario, provisionales con `id_pattern`, `ID#` vacío, cita repetida, segunda tabla, tabla sin filas), equivalencia con `isoevidence.AffectedFromDelta` en el caso sin requisitos sin ID ni escenarios repetidos, determinismo, `RewriteCubre` byte a byte e idempotente; `imports_test.go` que solo admite `internal/reqid`, `internal/isoevidence`, biblioteca estándar y nunca `os/exec`.
  - `Report.OK()` debe tener llamante alcanzable desde `cmd/axiom` (lo usa el verbo de T6 para decidir el código de salida); si T6 no lo usa, eliminarlo.
  - Cubre: RF-KOF-18, RF-KOF-19, RF-KOF-20, RF-KOF-23 (función de reescritura).
  - Verificación: `go test ./internal/acceptancetests -count=1`; `go vet ./internal/acceptancetests`.
  - Commit: `feat(acceptancetests): parsear y cruzar pruebas.md con las specs delta`.

## Phase 3: Integración

- [x] T4 Digest de la puerta `tasks` con cálculo único en las dos rutas.
  - Crear `internal/sddstatus/acceptance_tests.go` con `TasksGateArtifactPaths(changeRoot string, roles []kickoff.KickoffRole) []string`: `tasks.md`, o cada `TasksFile` si `kickoff.RoleTaskFiles(roles)` no está vacío, más `filepath.Join(changeRoot, acceptancetests.FileName)` solo si `hasContent` (`status.go`) lo da por presente y no vacío.
  - `internal/cli/sdd_gate_digest.go`: `tasksArtifactPathsForDigest` delega en `sddstatus.TasksGateArtifactPaths`, con `roles` tomados de `sealed.Config.Roles` y `nil` si `sealed == nil` (no usar el pseudocódigo `rolesOf`).
  - `internal/sddstatus/governance.go`: `governanceArtifactInputs` sustituye el bloque `tasksPaths` por la nueva función. El digest por rol y el de `role-apply:<rol>` no incluyen `pruebas.md`. No tocar `internal/kickoff/machine.go`.
  - Tests: sin `pruebas.md` la lista y el digest son idénticos a los actuales; un `pruebas.md` vacío o solo espacios queda fuera en ambas rutas; editar `pruebas.md` cambia el digest; paridad entre `gate record` y el status sobre el mismo fixture; una puerta `tasks` rechazada pasa a `pending` al editar solo `pruebas.md`; una aprobada sigue `approved` (`TestEvaluateGatesApprovedGateNeverInvalidatedByDigestChange` sigue en verde).
  - Ficheros: `internal/sddstatus/{acceptance_tests,governance}.go` (+ test), `internal/cli/sdd_gate_digest.go`, `internal/cli/sdd_gate_artifact_digest_test.go`.
  - Cubre: RF-GAT-31.
  - Verificación: `go test ./internal/sddstatus -run 'TasksGate|Governance' -count=1`; `go test ./internal/cli -run 'GateArtifactDigest' -count=1`; `go test ./internal/kickoff -run 'EvaluateGates' -count=1`; `go vet ./internal/sddstatus ./internal/cli`.
  - Commit: `feat(sddstatus): calcular una sola vez el conjunto de ficheros del digest tasks`.

- [ ] T5 `AcceptanceTestsFor` y proyección aditiva `acceptanceTests` en el status.
  - En `internal/sddstatus/acceptance_tests.go`: tipos `AcceptanceTestsResult` y `AcceptanceTestsProjection`, y `AcceptanceTestsFor(location workspace.Location, changeRoot string)`. Topología con `location.GoverningTopology()` (`internal/workspace/locate.go:64`, devuelve `(TopologyType, bool)`); con `false`, declaración no evaluable y no activa. `settings` con `EffectiveAcceptanceTests`; con error, `DeclarationError` con la continuación `corrígela con axiom workspace validate --path <master>` y `Active` falso. Roles: `sealed.Config.Roles` de `kickoff.Load(changeRoot)`, `nil` si `sealed == nil`; patrón con `location.Declarations.IDPattern()`; delta con `readDeltaRequirements(changeRoot, paths.Specs)`; fallos de kickoff, patrón o delta van a `Error`, nunca a un informe parcial.
  - Si falta `pruebas.md`: con la función activa y las tareas presentes, `Missing = true` (aviso y hallazgo `pruebas_ausente` vía `acceptancetests.MissingFinding`); en otro caso, nada.
  - Con la función no activa (o bloque inválido) y `pruebas.md` presente: se ejecuta `Check` igualmente pero con `SkipEnvironments` (NO valida `Entornos` contra el dominio porque no hay dominio declarado); el resto de comprobaciones sí (RF-KOF-21, D6).
  - `decorateAcceptanceTests(status *Status, location workspace.Location)` enganchado en `Resolve` tras `decorateExplainer` (`internal/sddstatus/status.go`); campo `Status.AcceptanceTests *AcceptanceTestsProjection json:"acceptanceTests,omitempty"` y copia en `internal/sddstatus/status_v2.go`. Clave omitida salvo que `axiom.yaml` declare el bloque (válido o no), sea ilegible o el cambio seleccionado no archivado tenga `pruebas.md`; la ausencia equivale a `active: false`. El aviso va dentro de la proyección (`warning`), nunca en `Notes` ni `blockedReasons`, y no altera `nextRecommended`, `dependencies`, `gates` ni digests.
  - Tests (`internal/sddstatus/acceptance_tests_test.go`, fixtures multirepo de `workspacetest`): omisión sin bloque con JSON idéntico byte a byte; proyección activa con batería; `enabled: false`; bloque inválido y regla de topología (con `error`); aviso por ausencia sin cambiar `nextRecommended` ni dependencias; función no activa con `pruebas.md` presente no emite `entorno_invalido` pero sí los demás hallazgos; kickoff ilegible va a `Error`; un `acceptance_tests` inválido no bloquea el archivado (RF-TOP-20, test en `sddstatus`); `TasksGateArtifactPaths` sin `pruebas.md` igual a hoy.
  - Documentar el contrato: párrafo `acceptanceTests` junto al de `explainer` en `internal/assets/skills/_shared/sdd-status-contract.md`.
  - Ficheros: `internal/sddstatus/{acceptance_tests,status,status_v2}.go`, `internal/sddstatus/acceptance_tests_test.go`, `internal/assets/skills/_shared/sdd-status-contract.md`.
  - Cubre: RF-KOF-22, RF-KOF-21 (cálculo compartido y caso no activo con fichero), RF-TOP-20 (archivado no bloqueado).
  - Verificación: `go test ./internal/sddstatus -run 'AcceptanceTests|TasksGate|Governance' -count=1`; `go test ./internal/assets -run 'StatusContract' -count=1`; `go vet ./internal/sddstatus`.
  - Commit: `feat(sddstatus): proyectar acceptanceTests de forma aditiva`.

- [ ] T6 Verbo `axiom sdd tests check` y su registro en `cmd/axiom`.
  - Crear `internal/cli/sdd_tests.go`: `RunSDDTests(args, stdout)` (subdespachador con `check` y ayuda, como `RunSDDGate`) y `runSDDTestsCheck` (banderas `--cwd`, `--change`, `--json` como `RunSDDArchiveCheck`, `gitref.ValidBranchComponent`, `resolveGovernanceChangeRoot`, `sddstatus.AcceptanceTestsFor`). Sin specs delta es error de uso. JSON `axiom.sdd-tests-check/v1` con `schema`, `change`, `active`, `environments`, `present`, `cases`, `scenarios`, `requirements`, `findings [{code,file,line,message}]` y `error?`. El verbo decide el código de salida con `Report.OK()`.
  - Códigos de salida: sin hallazgos 0; función no activa y sin `pruebas.md` 0 con «las pruebas de aceptación no están activadas»; función activa, antes de `tasks` y sin `pruebas.md` 0 con «todavía no hay batería»; algún hallazgo (incluido `pruebas_ausente`) 1 tras imprimir el informe completo; bloque inválido, topología, kickoff ilegible, patrón inaccesible, sin delta o cambio inexistente 1 con la continuación; función no activa (o bloque inválido) con `pruebas.md` presente: se ejecuta, NO valida `Entornos` contra el dominio y el resto sí (sale 1 solo si hay hallazgos distintos de `entorno_invalido`).
  - `cmd/axiom/main.go` (`runSDD`): línea de ayuda `tests            Comprueba la batería de pruebas de aceptación de un cambio (check)`, `case "tests": err = cli.RunSDDTests(subArgs, stdout)` y lista de opciones del mensaje de error. `cmd/axiom/main_test.go`: `tests` en `expectedSubcmds` y test de enrutado de `tests check` sin `--change` que espera `requiere --change`.
  - Toda función exportada nueva debe ser alcanzable desde `./cmd/axiom` (`scripts/deadcode-ratchet.sh`).
  - Tests (`internal/cli/sdd_tests_test.go`): texto y JSON, cada fila de la tabla de salida (incluida la nueva fila de función no activa con fichero), hallazgos con salida 1 e informe impreso antes del error, ausencia de escritura (solo lectura).
  - Ficheros: `internal/cli/sdd_tests.go` (+ test), `cmd/axiom/main.go`, `cmd/axiom/main_test.go`.
  - Cubre: RF-KOF-21.
  - Verificación: `go test ./internal/cli -run 'SDDTests' -count=1`; `go test ./cmd/axiom -run 'SDDHelp|SDDVerbRouting|SDDTests' -count=1`; `go vet ./internal/cli ./cmd/axiom`; `go build ./...`.
  - Commit: `feat(cli): añadir axiom sdd tests check`.

- [ ] T7 Archivado: `livespec` reescribe los provisionales de `Cubre` en `pruebas.md`.
  - `internal/livespec/change.go`: añadir `kindAcceptanceTests` a `changeFileKind`; `changeArtifacts` clasifica `name == acceptancetests.FileName`, y extrae como `Citation` los provisionales de la parte ID de cada entrada de `Cubre` (con la línea de la fila de `acceptancetests.Parse`); en la reescritura (rama `idPlan.Map.Count() > 0`) ese tipo llama a `acceptancetests.RewriteCubre`.
  - Comportamiento con tabla inválida o provisional sin correspondencia: igual que en los otros dos tipos (`kindTasks` y `kindVerify`). Las citas se entregan a `PlanIDs` (`internal/livespec/ids.go`), que rechaza un provisional citado y no declarado, como ya hace `TestPlanChangeRejectsAnUndeclaredCitationInTheTasks`; una tabla inválida no produce citas y el fichero queda sin reescribir, sin error propio (los tipos existentes tampoco validan el resto del fichero, RF-RID-12). Describir esto en el comentario de `changeArtifacts` y fijarlo con tests.
  - Tests (`internal/livespec/change_test.go` o nuevo): provisionales reescritos en `Cubre` (con `ID#nombre` y con ID suelto); el resto del fichero (nombre de escenario, otras columnas, texto fuera de tabla, finales de línea) byte a byte; idempotencia y ausencia de `pruebas.md`; IDs definitivos intactos; provisional sin correspondencia rechazado como en tasks; tabla inválida no falla ni reescribe.
  - Ficheros: `internal/livespec/change.go`, `internal/livespec/change_test.go`.
  - Cubre: RF-KOF-23.
  - Verificación: `go test ./internal/livespec -run 'AcceptanceTests|ChangeArtifacts|PlanChange' -count=1`; `go vet ./internal/livespec`.
  - Commit: `feat(livespec): reescribir los provisionales de pruebas.md al archivar`.

- [ ] T8 Dashboard: campo `acceptance_tests` del detalle y pestaña «Pruebas» de solo lectura.
  - `internal/dashboard/types.go`: `IncrementDetailDTO.AcceptanceTests *IncrementAcceptanceTestsDTO json:"acceptance_tests,omitempty"` con `IncrementAcceptanceTestsDTO` (`active`, `environments`, `error`, `present`, `warning`, `cases`, `scenarios`, `requirements`, `case_count`, `findings`), `AcceptanceCaseDTO` (nueve columnas y línea) y `AcceptanceFindingDTO`. El campo se omite (nil) para incrementos ODD, sin carpeta de cambio o archivados (mismo criterio que la omisión del status, T5); la pestaña lo trata como no activo.
  - `internal/dashboard/service.go`: `GetIncrementDetail` llama a `s.incrementAcceptanceTests(path)` (localiza con `workspace.Locate(s.getRootPath())` y mapea `sddstatus.AcceptanceTestsFor`); si `Locate` falla el DTO lleva `error` y el detalle se sirve igual.
  - Front de solo lectura: `internal/dashboard/assets/index.html` (pestaña `data-tab="pruebas"` rótulo «Pruebas» tras «Explicación», panel `data-panel="pruebas"`), `assets/js/core/router.js` (`'pruebas'` en `INCREMENT_TABS`), `assets/js/screens/detalle.js` (rama `tab === 'pruebas'`), módulo nuevo `assets/js/screens/detalle-pruebas.js` con `renderAcceptanceTests(panel, { detail })`: resumen de cobertura, aviso, hallazgos y tabla de casos con cada celda por `escapeHtml`, sin `renderMarkdown`; estados vacíos con `stateEmpty`/`stateError`.
  - Sin ruta nueva: no tocar `excludedRoutes` ni la lista blanca de RF-WEB-15.
  - Tests: DTO con casos y cobertura igual a la del verbo; campo nil para ODD, sin carpeta de cambio y archivado (dos casos); `error` cuando `Locate` falla con el detalle servido; contenido hostil (`<script>`, `|`) escapado y mostrado como texto; función no activada y activa sin fichero; pines de pestaña, rótulo, orden y ausencia de controles de escritura en `assets_test.go` y `workflow_assets_test.go`; sin ruta nueva. Los tests con `freePort` o hub compartido no usan `t.Parallel`.
  - Ficheros: `internal/dashboard/{types,service,service_test,assets_test,workflow_assets_test}.go`, `internal/dashboard/assets/index.html`, `internal/dashboard/assets/js/core/router.js`, `internal/dashboard/assets/js/screens/detalle.js`, `internal/dashboard/assets/js/screens/detalle-pruebas.js`.
  - Cubre: RF-WEB-17.
  - Verificación: `go test ./internal/dashboard -run 'IncrementDetail|AcceptanceTests|Tabs|Router|Canary' -count=1`; `go vet ./internal/dashboard`.
  - Commit: `feat(dashboard): mostrar la batería de pruebas en el detalle del incremento`.

- [ ] T9 Doctrina de `sdd-tasks`: compartido, skill, agentes, pines y golden.
  - Crear `internal/assets/skills/_shared/sdd-acceptance-tests.md`: condición (solo SDD con `acceptanceTests.active: true`, nunca ODD), formato de la tabla con ejemplo y nueve columnas, minimización en cuatro capas en su orden, comprobación con `axiom sdd tests check --cwd <workspaceRoot> --change <c>`, regla de refinado de `sdd-apply` (solo rutas o URL de `Pasos`), y presentación (RF-GAT-32): la skill devuelve `pruebas.md` en `artifacts` con el resultado de `axiom sdd tests check` en el resumen, y declara su ausencia cuando la función está activa y no lo escribió.
  - `internal/assets/skills/sdd-tasks/SKILL.md`: «Step 3b: Acceptance test battery» que remite a `../_shared/sdd-acceptance-tests.md`.
  - Secciones `## Acceptance Tests` (2 o 3 líneas) en `internal/assets/claude/agents/sdd-tasks.md`, `internal/assets/cursor/agents/sdd-tasks.md` y `internal/assets/copilot/agents/sdd-tasks.agent.md`.
  - Test de pines `internal/assets/acceptance_tests_doctrine_test.go` (modelo `explainer_html_doctrine_test.go`): compartido (condición SDD activa, ODD nunca, cuatro capas en orden, verbo, refinado de `sdd-apply`, nueve columnas, devolución de `pruebas.md` en `artifacts` con el resultado del check y declaración de ausencia), skill (enlace) y los tres agentes (sección).
  - Regenerar `testdata/golden/sdd-claude-agent-sdd-tasks.golden` con `go test ./internal/components -run TestGolden -update`, y comprobar sin `-update`.
  - NO tocar `internal/assets/skills/_shared/sdd-orchestrator-sections.md` (C-05-d edita la doctrina en paralelo), sus 10 goldens, la skill `sdd-apply` ni las copias instaladas en `.claude/skills/`.
  - Riesgo pendiente (dejar anotado, no resolver aquí): la frase de la parada `tasks` del orquestador que presenta `pruebas.md` junto a los ficheros de tareas no se añade porque exige editar `sdd-orchestrator-sections.md`; RF-GAT-32 se cumple vía la skill y el status. Reiterar o reanudar la parada en otra sesión queda sin pin propio.
  - Ficheros: `internal/assets/skills/_shared/sdd-acceptance-tests.md`, `internal/assets/skills/sdd-tasks/SKILL.md`, `internal/assets/claude/agents/sdd-tasks.md`, `internal/assets/cursor/agents/sdd-tasks.md`, `internal/assets/copilot/agents/sdd-tasks.agent.md`, `internal/assets/acceptance_tests_doctrine_test.go`, `testdata/golden/sdd-claude-agent-sdd-tasks.golden`.
  - Cubre: RF-KOF-17, RF-GAT-32.
  - Verificación: `go test ./internal/assets -run 'AcceptanceTestsDoctrine' -count=1`; `go test ./internal/components -run TestGolden -update` y después `go test ./internal/components -run TestGolden -count=1`; `go vet ./internal/assets`.
  - Commit: `feat(assets): doctrina de la batería de pruebas de aceptación en sdd-tasks`.

## Phase 4: Cierre

- [ ] T10 Ratchet de código muerto y canario de secretos de `/api`.
  - Ejecutar `scripts/deadcode-ratchet.sh` si es ejecutable en Git Bash; si no, revisar a mano que toda función exportada nueva (`acceptancetests.*`, `workspace.EffectiveAcceptanceTests`, `sddstatus.TasksGateArtifactPaths`, `sddstatus.AcceptanceTestsFor`, `cli.RunSDDTests`, `Report.OK`, `MissingFinding`, ayudantes de `isoevidence` exportados) es alcanzable desde `./cmd/axiom`. Eliminar lo inalcanzable.
  - Ejecutar `go test ./internal/dashboard -run Canary -count=1`.
  - Cubre: RF-WEB-17 (sin ruta nueva ni fuga de secretos), higiene de RF-KOF-21.
  - Verificación: salida del ratchet o la revisión manual anotada; canario en verde; `go build ./...`.
  - Commit: solo si hay corrección: `fix(acceptancetests): eliminar código inalcanzable desde cmd/axiom`.

## Riesgos pendientes

- La frase de la parada `tasks` del orquestador (RF-GAT-32) no se edita en este cambio; si C-05-d no la absorbe, la presentación de `pruebas.md` en la parada depende solo de la skill y del status.
- Huecos 4 y 9 de `coverage-review.design.md` resueltos en T7 y T8 por spec vigente; el 3 permanece como riesgo (arriba).
- Cuello de botella: T3 es camino crítico (T4, T5, T6, T7 dependen de él); T5 bloquea T6 y T8.
