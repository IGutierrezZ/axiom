# Tareas: Paradas con tres salidas en ODD y SDD (paradas-tres-salidas)

> Fase `tasks` · rama `inc/paradas-tres-salidas` · 2026-11-03.
> Entradas: `proposal.md`, `design.md` (DD-01 a DD-14, §9 matriz de amenazas, §10 compatibilidad, §11 riesgos) y las seis deltas de `specs/*/spec.md` (REQ-26.1 a REQ-26.11 nuevos; REQ-21.x y REQ-25.x modificados; REQ-21.14 retirado).
> Kickoff sellado: `continuous`, rol único `fullstack` → un solo `tasks.md`. Sin puertas propias.
> Idioma: castellano. Identificadores de código, rutas, flags y claves en inglés.

## Reglas de ejecución (válidas para todas las tareas)

- **TDD desactivado** (`strict_tdd: false`): cada tarea incluye sus pruebas y comprobaciones funcionales en la misma unidad de trabajo; no se exige RED previo. Los casos de la matriz de amenazas (design §9) que aplican se listan como casos de prueba dentro de T-03, T-04, T-06 y T-07.
- **Verificación solo con `go test` acotado por paquete y en proceso.** PROHIBIDO compilar o ejecutar binarios sueltos (`go build -o`, `axiom.exe`, `go run` a ejecutable temporal): el EDR corporativo los borra. Las pruebas de CLI usan los puntos de entrada en proceso (`RunSDDGate`, `RunSDDKickoff`). Las pruebas con git usan repositorios temporales y `gitcommon.Run`, como las del motor.
- **DD-14 (coordinación con `fix/bloqueos-prueba-multirepo-r2`):** el código nuevo va en ficheros nuevos y las ediciones en `governance.go`, `status.go`, `sdd_kickoff.go` y `args.go` son mínimas y puntuales, sin mover funciones. Los goldens se regeneran **solo** en T-23 y nunca se concilian a mano.
- **Compatibilidad (design §10):** no se tocan las fixtures que pasan `--handoff-policy` (`e2e/organicruntime/sdd_consent_wiring_e2e_test.go` (read-only), `cmd/axiom/handoff_branch_guard_test.go` (read-only), `bench/journeys_sdd.go` (read-only), `bench/journeys_issue_2891.go` (read-only)): son la evidencia de que un orquestador antiguo sigue sellando.
- **Fuera de alcance:** `internal/handoff`, `axiom handoff`, el dashboard y la TUI de relevos, `openspec/INDEX.md`.
- **Cada unidad de trabajo cierra con un commit Conventional** (sin atribución de IA), con sus pruebas y documentación junto al comportamiento. La heurística de unas 400 líneas por tarea es orientativa, no un límite.
- **Orden y compilación:** cada unidad deja el árbol compilando y con sus pruebas en verde; no hay dependencias hacia unidades posteriores.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 3.800 a 5.200 (central ≈ 4.600), autoradas (adiciones + eliminaciones, pruebas incluidas); sin contar goldens ni `.codex/AGENTS.md` regenerados |
| 400-line budget risk | High |
| Chained PRs recommended | No (una sola PR por decisión del usuario) |
| Suggested split | PR única con 11 commits de unidad de trabajo, revisables commit a commit |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

La estimación de `explore.md` §4 (1.500 a 2.500 líneas autoradas) no incluía las pruebas ni las eliminaciones de la unidad 6; la previsión de esta tabla sí. La PR lleva la etiqueta `size:exception`. Al ser una sola PR, la carga de revisión se reparte por commit: ninguna unidad supera ≈700 líneas y la mayor parte de las de la unidad 6 son eliminaciones.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| WU-1 (≈300 l) | Marca de cesión en el ledger y primitivas de dominio | PR única, commit 1 | `go test ./internal/kickoff/ -count=1` | N/A: dominio puro; prueba con struct congelado del esquema anterior | `internal/kickoff/{types,machine,ledger,handed_off}.go` y sus pruebas |
| WU-2 (≈600 l) | Entrega de la salida 2 por topología (`engine.HandOff`) | PR única, commit 2 | `go test ./internal/branch/... -count=1` | Repositorios git temporales con observador de `gitcommon.Run` (sin remotos) | `internal/branch/{types,labels,handoff}.go`, `internal/branch/engine/handoff*.go` |
| WU-3 (≈650 l) | `axiom sdd gate record --hand-off` | PR única, commit 3 | `go test ./internal/cli/ ./internal/kickoff/ -run 'Gate' -count=1` | `RunSDDGate` en proceso sobre repositorios temporales | `internal/cli/sdd_gate_handoff.go`, ediciones puntuales de `sdd_gate.go` y `args.go` |
| WU-4 (≈320 l) | Sobre de N opciones y pregunta tipada de tres salidas | PR única, commit 4 | `go test ./internal/consentenvelope/ ./internal/sddstatus/ -count=1` | `axiom status` proyectado en proceso (`RunSDDStatus` o el punto de entrada existente) | `envelope.go` y la pregunta en `governance.go` |
| WU-5 (≈230 l) | Cesión visible en `axiom sdd status` | PR única, commit 5 | `go test ./internal/sddstatus/ -count=1` | JSON v2 y texto de `status` en proceso | `governance_handoff.go`, `status_v2.go`, línea de notas en `status.go` |
| WU-6 (≈700 l, mayoría eliminaciones) | `verify` multirrol por `role-apply` y retirada del `handoff.md` de integración | PR única, commit 6 | `go test ./internal/kickoff/ ./internal/sddstatus/ ./internal/cli/ ./internal/branch/engine/ -count=1` | `RunSDDGate` en proceso: aprobar el último rol no escribe `handoff.md` | Restaurar `closure.go`, `writeIntegrationHandoff` y `verifyDependencyFromHandoff` con `git revert` del commit |
| WU-7 (≈520 l) | Fusión intermedia en el motor de ramas | PR única, commit 7 | `go test ./internal/branch/... ./internal/dashboard/ -count=1` | Repositorios temporales con referencia de integración local, sin red | `FileAt`, `intermediate.go`, cambios en `pending.go` y `archive.go` |
| WU-8 (≈280 l) | `handoff_policy` deprecado | PR única, commit 8 | `go test ./internal/kickoff/ ./internal/cli/ ./internal/sddstatus/ -count=1` | `RunSDDKickoff` en proceso con y sin `--handoff-policy` | `types.go`, `schema.go`, `args.go`, `infer.go`, `sdd_kickoff.go`, `status_v2.go` |
| WU-9 (≈430 l) | Texto único de las tres salidas y parada ODD | PR única, commit 9 | `go test ./internal/components/agentguidance/ -count=1` | `RenderRouting` en proceso para todos los adaptadores | `stop-exits.md`, `stop_exits.go`, `routing.go` |
| WU-10 (≈400 l) | Proyección a los orquestadores SDD, skills y contratos | PR única, commit 10 | `go test ./internal/assets/ ./internal/components/sdd/ -count=1` | Composición de las doce plantillas y de Pi en proceso | `sdd-orchestrator-sections.md`, `orchestrator.go`, skills y contratos |
| WU-11 (≈220 l + goldens) | Documentación, goldens y verificación final | PR única, commit 11 | `go test ./internal/components/ ./internal/workspace/ -count=1` y `go vet ./...` | Goldens regenerados con `-update` y vueltos a pasar sin él | Copias versionadas (`testdata/golden/`, `.codex/AGENTS.md`, `AGENTS.md`, `GEMINI.md`, manual) |

Orden de dependencias: WU-1 → WU-2 → WU-3 (usa 1 y 2) → WU-4 (invoca el flag de WU-3) → WU-5 (usa 1) → WU-6 (después de WU-3 para no editar `sdd_gate.go` dos veces en paralelo) → WU-7 (usa `IntermediateMerge` de WU-1) → WU-8 (después de WU-6: su único lector con efecto ya no existe) → WU-9 → WU-10 (usa `StopExits()` de WU-9) → WU-11 (siempre última).

---

## WU-1 · Marca de cesión en el ledger y primitivas de dominio

Commit: `feat(kickoff): marcar la cesión de la fase siguiente en el registro de puertas`

- [x] T-01 Añadir la marca de cesión al registro y al estado de la puerta (≈120 l)
  - Ficheros: `internal/kickoff/types.go`, `internal/kickoff/machine.go`, `internal/kickoff/machine_test.go` (o el fichero de pruebas de puertas existente del paquete).
  - Cambio: `GateRecord.HandedOff bool` con etiqueta `yaml:"handed_off,omitempty"` tras `Verified`; `GateState.HandedOff bool`; `resolveGateStatus` pasa a devolver `(status, reason string, reopened, handedOff bool)` con la marca del **primer** registro `approved`; `EvaluateGates` la copia a `GateState`; se ajusta el comentario de `machine.go:157` (la arista con `internal/handoff` desaparece en T-12).
  - Cubre: REQ-26.3 (vocabulario sin cambios; un rechazo posterior no reabre), REQ-26.7 (la marca es la del registro que cierra la puerta) · DD-01, DD-06.
  - Aceptación: (1) un registro `approved` sin marca serializa **byte a byte** igual que antes (`omitempty`); (2) un `gates.yaml` con `handed_off: true` se decodifica con un struct `legacyGateRecord` congelado (el anterior al cambio) sin error y la puerta resulta `approved`; (3) una decisión distinta de `approved|rejected` se sigue rechazando; (4) `approved` con marca seguido de `approved` sin marca conserva la marca y al revés no la añade; (5) un `rejected` posterior a un `approved` con marca deja la puerta `approved` con la marca.
  - Comprobación: `go test ./internal/kickoff/ -count=1`

- [x] T-02 Primitivas `Approval`, `HandedOffTo`, `IntermediateMerge` y escritura condicionada bajo cerrojo (≈180 l)
  - Ficheros: `internal/kickoff/handed_off.go` (nuevo), `internal/kickoff/handed_off_test.go` (nuevo), `internal/kickoff/ledger.go`, `internal/kickoff/ledger_test.go`.
  - Cambio: `GateLedger.Approval(gate GateKey) (GateRecord, bool)` (primer `approved`); `HandedOffTo(gate GateState, roster []multirole.RoleAssignment, states []GateState) string` (fijas → `gate.Blocks`; en `role-apply:<rol>` → `verify` si `LastRoleClosed(...)` y, si no, `apply:<r1>,<r2>` con los roles sin `role-apply` aprobado, en orden alfabético); `IntermediateMerge(k Kickoff, l GateLedger) bool` (algún `approved` con `HandedOff` y no todos los roles del roster con `RoleApplyGate` aprobado; roster vacío ⇒ verdadero en (b)); `AppendGateChecked(changeRoot string, rec GateRecord, check func(GateLedger) error) error` que ejecuta `check` bajo el cerrojo (`.gates.lock`) antes de anexar, y `AppendGate` delega con `check == nil`.
  - Cubre: REQ-26.3 (puerta ya aprobada, comprobada también bajo el cerrojo; A-1), REQ-26.7, REQ-26.11 (A-7), REQ-21.13 (la condición de último rol se apoya en `LastRoleClosed`) · DD-02 paso 10, DD-06, DD-09.
  - Aceptación: (1) `HandedOffTo` devuelve `design`, `tasks` y `apply` para las puertas fijas; `verify` para el último rol; `apply:web` cuando falta `web` (orden alfabético con varios); (2) `IntermediateMerge`: solo `spec` cedida ⇒ verdadero; todos los `role-apply` aprobados con `design` cedida ⇒ falso (A-7); sin marca ⇒ falso (A-8); roster vacío con marca ⇒ verdadero; (3) con dos goroutines concurrentes (salida 2 y salida 3) sobre una puerta pendiente, solo se anexa un `approved` y la segunda recibe el error del `check`; (4) `AppendGate` conserva su comportamiento previo.
  - Comprobación: `go test ./internal/kickoff/ -count=1`

---

## WU-2 · Entrega de la salida 2 por topología (`engine.HandOff`)

Commit: `feat(branch): calcular la entrega de la salida 2 por topología sin operaciones remotas`

- [ ] T-03 Tipos, rechazo y constructores de órdenes de entrega (≈250 l)
  - Ficheros: `internal/branch/types.go`, `internal/branch/labels.go`, `internal/branch/handoff.go` (nuevo), `internal/branch/handoff_test.go` (nuevo).
  - Cambio: `CodeUncommittedArtifacts Code = "uncommitted_artifacts"` con etiqueta «artefactos sin confirmar»; `HandOffRequest`, `HandOffPR`, `HandOffDelivery`, `HandOffPlan` (design §4.3); constructor `UncommittedArtifacts(change, repo, paths)` que devuelve `*Refusal` con las rutas y la continuación de commit; constructores de los pasos exactos (`git -C <gitRoot> add -- <rel>/gates.yaml`, `git -C <gitRoot> commit -m "chore(sdd): aprobar <puerta> de <cambio> y ceder <handedOffTo>" -- <rel>/gates.yaml`, `git -C <gitRoot> push -u origin <rama>`; in situ `git -C <gitRoot> push origin <integración>`) con rutas citadas por `pathquote.Quote`; el texto «la fase siguiente solo empieza cuando la PR esté fusionada en `<prTarget>`…».
  - Cubre: REQ-26.4, REQ-26.5 · DD-07, DD-08 · matriz de amenazas «Selección de repositorio git», «Estado del commit», «Estado del push», «Órdenes de PR».
  - Aceptación (casos de prueba de design §9): (1) ruta con espacios citada; (2) cada orden usa `-C <gitRoot>` absoluto, nunca el cwd; (3) el commit usa pathspec explícito y nunca `-a`; (4) el push nombra la rama concreta, nunca `HEAD`; (5) ningún paso contiene `&&`, `gh ` ni `glab `; (6) `pr.head` y `pr.base` en campos separados; (7) el rechazo lleva continuación ejecutable (ratchet de rechazos).
  - Comprobación: `go test ./internal/branch/ -count=1`

- [ ] T-04 `Engine.HandOff`: comprobación de «confirmado» y entregas por repositorio (≈350 l)
  - Ficheros: `internal/branch/engine/handoff.go` (nuevo), `internal/branch/engine/handoff_test.go` (nuevo).
  - Cambio: `func (e *Engine) HandOff(ctx, req branch.HandOffRequest) (branch.HandOffPlan, *branch.Refusal)` sobre `gather` + `describe`, sin cerrojo, sin escribir el ledger y sin `Fetch`. Resuelve el plan del cambio (carril SDD; un rechazo de plan se devuelve tal cual), localiza el repositorio del planning home (el cuyo `GitRoot` contiene `ChangeRoot`), aplica DD-03 con `DirtyPaths` y compone una `HandOffDelivery` por repositorio (monorepo con rama de cambio o de rol → `PRTarget`; multirepo, repositorio de specs in situ → sin PR; `role-apply` en otros repositorios del rol).
  - DD-03: en `spec`, `design` y `tasks`, nada sucio bajo `<planning home>/changes/<cambio>/`, con exención de `gates.yaml` y de las entradas de la raíz cuyo nombre empieza por punto (`.gates.lock`, `.axiom-instance` y su variante heredada, `.gates-*.yaml`, `.kickoff-*.yaml`); una carpeta entera sin seguimiento aparece como una sola entrada con «/» final y se rechaza. En `role-apply:<rol>`, además, el árbol de trabajo entero de cada repositorio cuyo `Roles` contiene el rol.
  - Cubre: REQ-26.4 (incluida la desviación declarada R-3), REQ-26.5 (A-4) · DD-03, DD-07, DD-08 · matriz «Estado del commit» y «Estado del push».
  - Aceptación (casos de prueba): (1) preparado sin confirmar ⇒ rechazo; (2) sin seguimiento ⇒ rechazo; (3) carpeta entera sin seguimiento ⇒ rechazo; (4) solo `gates.yaml` y `.gates.lock` sucios ⇒ pasa; (5) `role-apply` con trabajo del rol sin confirmar en el repositorio ⇒ rechazo; (6) monorepo con rama de cambio ⇒ `prTarget` = integración; con rama de rol ⇒ `prTarget` = rama de seguimiento, sin afirmar integración; (7) multirepo, repositorio de specs ⇒ entrega `InPlace` sin PR; (8) cwd del proceso distinto del worktree ⇒ el `-C` es el del worktree; (9) un observador instalado en `gitcommon.Run` no registra `push`, `fetch`, `pull`, `ls-remote` ni `commit`; (10) todo resultado lleva `MergeRequired: true`.
  - Comprobación: `go test ./internal/branch/... -count=1`

---

## WU-3 · `axiom sdd gate record --hand-off`

Commit: `feat(sdd): salida 2 de la parada con gate record --hand-off`

- [ ] T-05 Flag `--hand-off` y su validación de forma (≈120 l)
  - Ficheros: `internal/kickoff/args.go` (edición puntual), `internal/kickoff/args_test.go`.
  - Cambio: `GateRecordArgs.HandOff bool` y parseo de `--hand-off` (sin valor) en `ParseGateRecordArgs`. Rechazos puros antes de tocar disco: sin `--decision approved` («`--hand-off` solo acompaña a `--decision approved`»); con `--gate integration` (no disponible para esa puerta); con `--evidence-kind`. `--reason` sigue siendo opcional con `--hand-off`.
  - Cubre: REQ-26.1 (escenario de `integration`), REQ-26.3 (motivo opcional) · DD-01.
  - Aceptación: cada rechazo devuelve un error de uso con continuación; `--hand-off` con `--decision approved --gate spec` parsea con `HandOff == true`; `--decision rejected --reason x` y `--decision approved` siguen parseando con `HandOff == false`.
  - Comprobación: `go test ./internal/kickoff/ -run 'GateRecord' -count=1`

- [ ] T-06 Comprobaciones de la salida 2 en fichero nuevo (≈230 l)
  - Ficheros: `internal/cli/sdd_gate_handoff.go` (nuevo), `internal/cli/sdd_gate_handoff_test.go` (nuevo).
  - Cambio: funciones que `runSDDGateRecord` llamará en dos puntos y que implementan los pasos 4, 6, 7 y 8 de DD-02: kickoff sellado y `execution_style: checkpointed` («el cambio no tiene paradas», con continuación `--decision approved` sin `--hand-off`); almacén declarado (`workspace.DeclaredArtifactStore`; con `engram` o `none`, rechazo que nombra el almacén y ofrece las salidas 1 y 3; sin declaración cuenta como `openspec`); digest no vacío (`gateArtifactDigest`; un artefacto inexistente o vacío se rechaza con la continuación «produce el artefacto»); puerta ya aprobada (`LoadGates` + `Approval(gate)` ⇒ «la puerta ya está resuelta», continuación `axiom sdd status`); y la llamada a `engine.Default().HandOff` con la exención de `gates.yaml`.
  - Cubre: REQ-26.1 (continuo e `integration`), REQ-26.3 (A-1), REQ-26.4, REQ-26.6 · DD-02, DD-03 · REQ-26.4 «último escenario» (`wrong_branch` primero se comprueba en T-07).
  - Aceptación: cada rechazo (continuo, `engram`, `none`, digest vacío, ya aprobada, artefactos sin confirmar) devuelve error distinto de cero con continuación ejecutable y deja `gates.yaml` byte a byte intacto; `uncommitted_artifacts` nombra las rutas pendientes y la continuación de commit.
  - Comprobación: `go test ./internal/cli/ -run 'Gate' -count=1`

- [ ] T-07 Integrar la salida 2 en `runSDDGateRecord`, texto de entrega y `--json` (≈300 l)
  - Ficheros: `internal/cli/sdd_gate.go` (ediciones mínimas en los puntos de llamada y en la salida), `internal/cli/sdd_gate_handoff.go`, `internal/cli/sdd_gate_test.go`, `internal/cli/sdd_writer_guard_test.go`.
  - Cambio: orden de DD-02 (forma → raíz del cambio → guarda A `wrong_branch` antes que cualquier otra → kickoff → pertenencia al roster → almacén → digest → ya aprobada → `engine.HandOff` → `kickoff.AppendGateChecked(..., notApproved(gate))` bajo cerrojo → salida). Salida de texto de la salida 2 en castellano: «Compuerta "<g>" registrada (approved, fase siguiente cedida: <handedOffTo>) para "<c>".», los pasos numerados por repositorio, la advertencia de fusión previa y «Axiom no ejecuta ninguna de estas órdenes». `--json` (hoy aceptado e ignorado) emite `axiom.sdd-gate-record/v1` en **todas** las salidas (design §4.4); `delivery`, `handedOff` y `handedOffTo` solo en la salida 2; `lastRoleClosed` solo en `role-apply`. Los rechazos siguen siendo errores de la orden. Con `--hand-off` también se emite el aviso de último rol (REQ-21.13).
  - Cubre: REQ-26.3, REQ-26.4 (`wrong_branch` antes de comprobar artefactos), REQ-26.5, REQ-26.6, REQ-21.13 (cuarto escenario) · DD-02, DD-07, DD-08.
  - Aceptación: (1) orden de comprobaciones con aserción de `gates.yaml` intacto en cada rechazo; (2) `wrong_branch` se devuelve antes que el rechazo por sucio; (3) salida 2 en monorepo devuelve rama, `prTarget`, `add`/`commit` de `gates.yaml`, `push -u origin <rama>` y `pr.head`/`pr.base`, y advierte de la fusión; (4) multirepo en el repositorio de specs devuelve commit y push de la integración y no nombra rama de cambio ni PR; (5) un observador sobre `gitcommon.Run` no registra `push`, `fetch`, `pull`, `ls-remote` ni `commit`; (6) las salidas 1 y 3 escriben registros sin `handed_off`; (7) JSON `axiom.sdd-gate-record/v1` válido en las tres salidas; (8) dos invocaciones concurrentes de salida 2 y salida 3 dejan un único `approved`.
  - Comprobación: `go test ./internal/cli/ -run 'Gate|WriterGuard' -count=1`

---

## WU-4 · Sobre de N opciones y pregunta tipada de tres salidas

Commit: `feat(sddstatus): pregunta de puerta con tres salidas sobre gate record`

- [ ] T-08 `ValidateCompleteness` con la lista exacta de respuestas (≈110 l)
  - Ficheros: `internal/consentenvelope/envelope.go`, `internal/consentenvelope/envelope_test.go`.
  - Cambio: `func (core Core) ValidateCompleteness(answers ...string) error`: exige al menos dos respuestas distintas entre sí (error de construcción en caso contrario), `len(core.Choices) == len(answers)` y `core.Choices[i].Answer == answers[i]`; el resto de comprobaciones no cambia. Se actualiza el comentario de cabecera («exactly two token choices» → «the exact token choices each consumer declares»). Los tres llamadores (`internal/sddstatus/consent_contract.go`, `internal/cli/review_consent_contract.go`, `internal/sddstatus/governance.go`) compilan sin tocarse y siguen pasando dos tokens.
  - Cubre: REQ-26.2 (el sobre de autoridad de edición sigue exigiendo exactamente dos) · DD-04.
  - Aceptación: casos válidos con tres respuestas; longitud u orden distintos, duplicados y menos de dos se rechazan; el caso existente «three choices» con `("granted", "declined")` sigue rechazando sin cambiar su expectativa.
  - Comprobación: `go test ./internal/consentenvelope/ -count=1` y `go test ./internal/sddstatus/ ./internal/cli/ -run 'Consent|Envelope' -count=1`

- [ ] T-09 Pregunta de puerta con tres opciones, o dos con `handOffUnavailable` (≈210 l)
  - Ficheros: `internal/sddstatus/governance.go` (la pregunta y su `Validate` en el sitio, sin mover funciones), `internal/sddstatus/governance_handoff.go` (nuevo; almacén efectivo de la pregunta), `internal/sddstatus/status.go` (tres líneas: pasar el almacén en `:754`), `internal/sddstatus/governance_test.go`, `internal/sddstatus/governance_continuous_test.go`, `internal/sddstatus/status_v2_test.go`.
  - Cambio: constante `gateAnswerHandOff = "hand_off"`; `SDDGovernanceGateResult.HandOffUnavailable string` (`json:"handOffUnavailable,omitempty"`); `newGovernanceGateQuestion` recibe `store ArtifactStore` (declarado o, sin declaración, el efectivo de `status`). Opciones: `spec`/`design`/`tasks`/`role-apply:<rol>` con `openspec` o `hybrid` ⇒ `[rejected, hand_off, approved]` con las invocaciones `--decision rejected --reason <your-reason>`, `--decision approved --hand-off --reason <your-reason>`, `--decision approved --reason <your-reason>`; con `none` (único caso en que se construye con dos opciones) ⇒ `[rejected, approved]` más `handOffUnavailable`; `integration` ⇒ `[approved, rejected]` como hoy. `Validate` deriva la lista esperada, exige invocaciones distintas y con prefijo `axiom sdd gate record `, y que solo `hand_off` contenga ` --hand-off`.
  - Cubre: REQ-26.1, REQ-26.2, REQ-26.6 (A-3 corregido) · DD-05.
  - Aceptación: (1) tres opciones en orden `rejected`, `hand_off`, `approved` con `--cwd`, `--change` y `--gate` compartidos; (2) dos opciones con `handOffUnavailable` para `none` y texto que declara la salida 2 no disponible; (3) `integration` sin cambios; (4) dos opciones sin `handOffUnavailable` fuera de `integration` ⇒ rechazada; (5) token o invocación duplicados ⇒ rechazada; (6) invocación con otro verbo ⇒ rechazada; (7) el sobre de autoridad de edición (`axiom.sdd-integration.consent/v1`) con tres opciones se sigue rechazando; (8) el contrato de *Lossless Blocking Prompt* se cumple con tres opciones.
  - Comprobación: `go test ./internal/sddstatus/ -count=1` y `go test ./internal/assets/ -run 'BlockingPrompt' -count=1`

---

## WU-5 · Cesión visible en `axiom sdd status`

Commit: `feat(sddstatus): mostrar en status la cesión de la fase siguiente`

- [ ] T-10 `handedOff` y `handedOffTo` en el JSON v2 y nota informativa (≈230 l)
  - Ficheros: `internal/sddstatus/governance_handoff.go`, `internal/sddstatus/status_v2.go`, `internal/sddstatus/status.go` (una línea tras `loadGovernance`), `internal/sddstatus/status_v2_test.go`, `internal/sddstatus/governance_handoff_test.go` (nuevo).
  - Cambio: `gateV2.HandedOff bool` (`json:"handedOff,omitempty"`) y `gateV2.HandedOffTo string` (`json:"handedOffTo,omitempty"`) rellenados con `kickoff.HandedOffTo`; `handedOffNotes(governance)` añade a `status.Notes` (bloque «Informational only») una nota por puerta con cesión: «La compuerta "spec" se aprobó cediendo `design` a otra persona: quien recoge el cambio empieza esa fase sin revisar la anterior.». La ruta (`resolveNextRecommended`) no cambia.
  - Cubre: REQ-26.7 · DD-06.
  - Aceptación: (1) `spec` cedida con `design` sin empezar ⇒ `handedOff: true`, `handedOffTo: design` y `nextRecommended` igual que tras una aprobación normal; (2) último `role-apply` cedido ⇒ `handedOffTo: verify`; (3) `core` cedido con `web` pendiente ⇒ `apply:web`; (4) un cambio con todas las puertas aprobadas con la salida 3 produce una salida idéntica a la anterior (comparación contra un golden de status existente, sin regenerarlo); (5) la marca es la del primer `approved`; (6) la cesión no bloquea ninguna fase ni orden.
  - Comprobación: `go test ./internal/sddstatus/ -count=1`

---

## WU-6 · `verify` multirrol por `role-apply` y retirada del `handoff.md` de integración

Commit: `refactor(sdd): habilitar verify por los role-apply aprobados y retirar el handoff.md de integración`

- [ ] T-11 `verifyDependencyFromRoleGates` sustituye a `verifyDependencyFromHandoff` (≈260 l)
  - Ficheros: `internal/sddstatus/governance_handoff.go`, `internal/sddstatus/governance.go` (solo el borrado de `verifyDependencyFromHandoff` y del import de `internal/handoff`), `internal/sddstatus/status.go` (la llamada de `:730`), `internal/sddstatus/handoff_dependency_test.go` (eliminar), `internal/sddstatus/role_gates_dependency_test.go` (nuevo).
  - Cambio: `func verifyDependencyFromRoleGates(governance *Governance, baseline DependencyState) (DependencyState, string)` devuelve `baseline, ""` si no hay gobierno con puertas, si el roster tiene uno o ningún rol, si `baseline != DependencyReady` o si el último rol está cerrado; en otro caso `DependencyBlocked` con un motivo que nombra los roles sin `role-apply` aprobado. Solo estrecha `ready` a `blocked` (DD-10); no cambia `blockedReasons` en `spec`, `design` ni `tasks`.
  - Cubre: REQ-21.15, REQ-26.9 (`per_checkpoint` sin efecto) · DD-10.
  - Aceptación: (1) `core` aprobado y `web` pendiente ⇒ `verify` bloqueado con motivo que nombra `web`; (2) ambos aprobados y sin `handoff.md` ⇒ `verify` desbloqueado y siguiente fase proyectada; (3) un `handoff.md` `blocked` o `needs_clarification` presente no altera el resultado; (4) un `role-apply` rechazado mantiene el bloqueo; (5) rol único o modalidad continua ⇒ igual que antes; (6) `per_checkpoint` y `none` producen el mismo resultado.
  - Comprobación: `go test ./internal/sddstatus/ -count=1`

- [ ] T-12 Retirar `writeIntegrationHandoff`, `kickoff.IntegrationHandoff` y `closure.go` (≈440 l, casi todo eliminaciones)
  - Ficheros: `internal/cli/sdd_gate.go` (eliminar `writeIntegrationHandoff` y `buildIntegrationArtifactInventory`, el import de `internal/handoff`; `lastRoleJustClosed` devuelve `(bool, error)`; se conserva `fileHasContent`; aviso nuevo), `internal/kickoff/closure.go` (eliminar), `internal/kickoff/closure_test.go` (eliminar), `internal/kickoff/rdd_boundary_test.go` (sin la arista `kickoff → handoff`), `internal/kickoff/machine.go` (comentario), `internal/branch/engine/writer_inventory_test.go` (quitar la entrada `writeIntegrationHandoff` y su motivo; conservar `{"internal/handoff", "WriteFile"}`), `internal/cli/sdd_gate_test.go`.
  - Cambio: el aviso conserva `lastRoleNoticeMarker` y pasa a decir «Aviso: se ha concluido la implementacion del ultimo rol activo del cambio %q; queda abierta la verificacion global de la solucion (REQ-21.13, REQ-21.15).»; se emite con la salida 2 y con la salida 3.
  - Cubre: REQ-21.13, REQ-21.14 (retirado), REQ-21.15 · DD-10.
  - Aceptación: (1) aprobar el último rol, único (`fullstack`) o de varios, emite el aviso y **no** escribe `handoff.md`; (2) un rechazo del único rol pendiente no emite el aviso; (3) `go vet` y el ratchet de deadcode no detectan código muerto tras la retirada; (4) el inventario de escritores y la frontera de importaciones de `kickoff` siguen en verde.
  - Comprobación: `go test ./internal/kickoff/ ./internal/sddstatus/ ./internal/cli/ ./internal/branch/engine/ -count=1`

---

## WU-7 · Fusión intermedia en el motor de ramas

Commit: `feat(branch): no tratar una fusión intermedia de planificación como cambio integrado`

- [ ] T-13 Lectura de ficheros de un árbol de git: `Git.FileAt` (≈170 l)
  - Ficheros: `internal/branch/ports.go`, `internal/branch/gitcli/ops.go`, `internal/branch/gitcli/ops_test.go`, `internal/branch/engine/engine.go` (`gitReader`), y los dobles de prueba de `branch.Git` que ya existan en los paquetes `branch` y `engine`.
  - Cambio: `FileAt(ctx, dir, rev, relPath string) (content []byte, exists bool, err error)` implementado con `ls-tree -z --full-tree <rev> -- <relPath>` seguido de `cat-file blob <objeto>`; valida argumentos con `plainArgument` y `pathspecArgument` como `TreeDirs`; `cat-file` se añade a la lista cerrada de lecturas del comentario de paquete (`gitcli/ops.go`).
  - Cubre: REQ-26.11 (evaluación sobre la referencia local, sin red) · DD-09.
  - Aceptación: fichero existente devuelve contenido; ausente devuelve `exists == false` sin error; revisión o ruta con argumentos no válidos se rechazan; el observador no registra operaciones remotas.
  - Comprobación: `go test ./internal/branch/... -count=1`

- [ ] T-14 Criterio de fusión intermedia y exclusión de `PendingArchive` (≈200 l)
  - Ficheros: `internal/branch/engine/intermediate.go` (nuevo), `internal/branch/engine/pending.go`, `internal/branch/engine/pending_test.go`.
  - Cambio: `func (e *Engine) intermediateMerge(ctx, root, sha, changesRel, change string) (bool, string)`: lee `<changesRel>/<change>/gates.yaml` (ausente o ilegible ⇒ `false`), si ninguna marca `handed_off` ⇒ `false` sin leer más, lee `kickoff.yaml` (ausente o ilegible ⇒ `false`) y aplica `kickoff.IntermediateMerge`; un error de git (no de contenido) ⇒ `false` y un aviso añadido a los existentes. `PendingArchive` salta los cambios intermedios; dashboard y TUI lo heredan sin cambios.
  - Cubre: REQ-26.11, REQ-25.20 (A-7, A-8) · DD-09.
  - Aceptación (escenarios de REQ-26.11): (1) `x` con solo `spec` cedida ⇒ no figura y código de salida 0; (2) `design` cedida con `core`/`web` sin aprobar ⇒ intermedio y sin la continuación de archivado; (3) `role-apply:core` cedido con `web` pendiente ⇒ intermedio; (4) todos los `role-apply` aprobados ⇒ figura como pendiente con `axiom branch ensure --change x --archive`; (5) `y` sin marca o sin `gates.yaml` ⇒ figura como antes; (6) `gates.yaml` o `kickoff.yaml` ilegibles ⇒ como antes, sin error; (7) la evaluación no usa la red; (8) la fusión intermedia no bloquea `axiom branch status`, `axiom sdd status` ni un escritor guardado de otro cambio.
  - Comprobación: `go test ./internal/branch/engine/ ./internal/dashboard/ -count=1`

- [ ] T-15 `ensure --archive` rechaza una fusión intermedia (≈150 l)
  - Ficheros: `internal/branch/archive_refusal.go`, `internal/branch/engine/archive.go`, `internal/branch/engine/archive_test.go`.
  - Cambio: `branch.ChangeIntermediateMerge(change, master, prefix, repo, base, changePath)` con código `change_not_integrated`, detalle `intermediate: "true"`, mensaje que lo dice, sin sugerencia de `--fetch` y continuación «termina el cambio (todos los `role-apply` aprobados), fusiona su PR y repite `axiom branch ensure --change <c> --archive`»; `checkIntegrated` lo devuelve cuando la carpeta existe y el cambio es intermedio. En el repositorio de specs de un `multirepo` sigue rechazando con `archive_branch_not_applicable`.
  - Cubre: REQ-26.11, REQ-25.45 (S-16 enmendado) · DD-09.
  - Aceptación: `--archive` sobre un cambio intermedio rechaza con `change_not_integrated`, indica que es intermedio y no sugiere `--fetch`; sobre uno sin marca o con carpeta ausente se comporta como antes (con la sugerencia de `--fetch` cuando falta); quien recoge la cesión con `axiom branch ensure --change x` obtiene la rama desde la base fusionada sin operaciones remotas.
  - Comprobación: `go test ./internal/branch/... ./internal/tui/ -run 'Archive|Pending|Intermediate' -count=1`

---

## WU-8 · `handoff_policy` deprecado

Commit: `refactor(kickoff): deprecar handoff_policy; aceptar el flag e ignorarlo`

- [ ] T-16 Paquete `kickoff`: campo opcional, flag ignorado y valor siempre `none` (≈150 l)
  - Ficheros: `internal/kickoff/types.go`, `internal/kickoff/schema.go`, `internal/kickoff/args.go` (trozos puntuales), `internal/kickoff/infer.go`, y sus pruebas (`schema_test.go`, `args_test.go`, `infer_test.go`).
  - Cambio: `HandoffPolicy` con comentario `Deprecated:` y etiqueta `yaml:"handoff_policy,omitempty"` (se conservan `HandoffNone` y `HandoffPerCheckpoint`); `Validate` acepta vacío, `none` y `per_checkpoint` y rechaza el resto; `--handoff-policy` se sigue parseando y validando (valor desconocido rechazado, A-6) pero `SealArgs.HandoffPolicy` queda `Deprecated:` y sin uso; se elimina «kickoff seal requiere --handoff-policy»; `ToKickoff` y el retro-sello (`seal --infer`) fijan siempre `HandoffNone`.
  - Cubre: REQ-26.9, REQ-21.4 (retro-sello sin política), REQ-21.5 · DD-11.
  - Aceptación: (1) sellar sin `--handoff-policy` escribe `handoff_policy: none`; (2) con `--handoff-policy per_checkpoint` sella y escribe `none`; (3) `--handoff-policy otra` se rechaza y no escribe `kickoff.yaml`; (4) `Validate` acepta `""`, `none`, `per_checkpoint` y rechaza `otra`; (5) un kickoff antiguo con `per_checkpoint` se lee y valida sin error; (6) `seal --infer` escribe `none` sin inferir política.
  - Comprobación: `go test ./internal/kickoff/ -count=1`

- [ ] T-17 CLI y JSON sin la política en el texto; mensaje de sellado sin el flag (≈130 l)
  - Ficheros: `internal/cli/sdd_kickoff.go` (ayuda de `:36`, texto de `:131-132`, JSON de `:160`), `internal/sddstatus/status_v2.go` (`json:"handoffPolicy,omitempty"`), `internal/cli/sdd_gate.go` (mensaje de `:89` sin `--handoff-policy none`), `internal/cli/sdd_kickoff_test.go`, `internal/sddstatus/status_v2_test.go`.
  - Cubre: REQ-26.9 (texto, JSON y mensajes de error) · DD-11.
  - Aceptación: (1) la ayuda de `kickoff seal` y los mensajes de error no nombran `--handoff-policy`; (2) el texto de `kickoff seal` y de `kickoff show` no menciona la política de relevos; (3) `kickoff show --json`, `kickoff seal --json` y `axiom sdd status --json` no emiten `handoffPolicy` cuando el campo está vacío y sí el valor leído cuando existe; (4) un kickoff antiguo con `per_checkpoint` no altera ninguna proyección de `status`.
  - Comprobación: `go test ./internal/cli/ ./internal/sddstatus/ -run 'Kickoff|Status|Gate' -count=1`

---

## WU-9 · Texto único de las tres salidas y parada ODD

Commit: `feat(agentguidance): texto único de las tres salidas y parada única de ODD tras el plan`

- [ ] T-18 Asset `stop-exits.md` y lector `StopExits()` (≈170 l)
  - Ficheros: `internal/assets/generic/stop-exits.md` (nuevo), `internal/components/agentguidance/stop_exits.go` (nuevo, con la constante `stopExitsAsset = "generic/stop-exits.md"` y el precedente de `remote_authorization.go`), `internal/components/agentguidance/stop_exits_test.go` (nuevo).
  - Cambio: texto en inglés que empieza por el párrafo en negrita **Stops and exits.** y define: qué es una parada; las tres salidas en orden, con iterar sin límite; «subir» = PR **fusionada** en `prTarget` antes de empezar la fase siguiente, con el motivo de `ensure`; en el repositorio de specs de un `multirepo`, commit en la integración y push humano sin PR; `push`, `fetch`, PR y merge son del humano y el agente nunca los ejecuta; con almacén `engram` o `none` la salida 2 no está disponible; quien recoge empieza la fase siguiente sin revisar la anterior. No nombra `gateQuestion`, `gateCriteria` ni `gateEvidence`.
  - Cubre: REQ-26.8, REQ-26.10 · DD-12.
  - Aceptación: pines de las tres salidas y su orden, del significado de «subir» (PR fusionada), del caso `multirepo`, de que push/PR/merge son del humano, de la no disponibilidad con `engram`/`none`; ausencia de los tres nombres prohibidos; salida determinista (mismo texto en cada llamada).
  - Comprobación: `go test ./internal/components/agentguidance/ -count=1`

- [ ] T-19 Parada ODD en la doctrina generada (≈260 l)
  - Ficheros: `internal/components/agentguidance/routing.go`, `internal/components/agentguidance/routing_test.go`.
  - Cambio: sección nueva `### Stops and exits` con `StopExits()` tras *Branches and worktrees* y antes de *Receipt-driven development*; paso 0 conserva literalmente «do not ask any further governance question» y añade que la parada única tras el paso 5 no es una pregunta de gobernanza; paso 5 añade la parada del plan sin nombrar `axiom branch` (el pin de `routing_test.go:364` lo reserva al paso 6); línea de reanudación que parte del fichero y tolera la ausencia del espejo de Engram, no revisa el plan aprobado y sigue por la primera tarea sin completar; viñeta nueva en *Organic Driven Development* con la parada (solo con documento de feature; no es pregunta de gobernanza ni crea artefactos SDD ni `handoff.md`; no se repite al reanudar; línea `Plan stop: approved <YYYY-MM-DD>; implementation continues` o `...; implementation handed off`; iterar ajusta fichero y espejo y vuelve a presentar; salida 2 confirma **solo** `odd/tasks/<feature-name>.md` con `git -C <path> add -- ...` y `git -C <path> commit -m "<msg>" -- ...`, lee `axiom branch status --lane odd --change <feature-name> --json`, presenta rama, `prTarget`, push y datos de la PR y se detiene sin escribir código; en `multirepo`, commit en la integración y push humano, sin PR).
  - Cubre: REQ-26.10, REQ-21.2, REQ-25.40 (escenario «El resto del protocolo no cambia») · DD-13.
  - Aceptación: se conservan los pines de los pasos 0 y 6; nuevos pines de la sección (igual a `StopExits()`), del paso 5, de la línea `Plan stop:`, de la tolerancia al espejo ausente, de la ausencia de órdenes remotas ejecutables (`push`, `fetch`, PR, merge presentados como decisión humana) y del determinismo en todos los adaptadores.
  - Comprobación: `go test ./internal/components/agentguidance/ -count=1`

---

## WU-10 · Proyección a los orquestadores SDD, skills y contratos

Commit: `feat(sdd): proyectar las tres salidas a los orquestadores y retirar la pregunta de transición`

- [ ] T-20 Sección de puertas con las tres salidas y marcador `{{AXIOM_STOP_EXITS}}` (≈180 l)
  - Ficheros: `internal/assets/skills/_shared/sdd-orchestrator-sections.md` (sección *SDD Change Kickoff and Block Gates*: segundo párrafo sin `--handoff-policy none|per_checkpoint`; tercer párrafo con las tres salidas registradas con `axiom sdd gate record`; línea nueva con el marcador antes del párrafo del contexto mantenido), `internal/components/sdd/orchestrator.go` (`substituteSharedOrchestratorSections`: segundo paso explícito que sustituye el marcador por `agentguidance.StopExits()`; el comentario «one pass» se amplía), `internal/components/sdd/orchestrator_shared_sections_test.go`.
  - Cambio: tercer párrafo con iterar = `--decision rejected --reason`; aprobar, subir y parar = `--decision approved --hand-off` (rechaza sin escribir si no está confirmado, registra y devuelve commit de `gates.yaml`, push y datos de la PR para presentarlos y parar); aprobar y seguir = `--decision approved`; `integration` conserva aprobar con evidencia o rechazar; se conservan literalmente «re-present the SAME gate afterward, never skip ahead to the next one» y los demás pines.
  - Cubre: REQ-26.8, REQ-21.5 · DD-12.
  - Aceptación: en los doce orquestadores compuestos (claude, codex, opencode/kilocode, gemini, antigravity, windsurf, cursor, kimi, qwen, kiro, hermes y generic) y en Pi, `strings.Contains(rendered, StopExits())` es verdadero; no queda sin resolver ni `{{GENTLE_AI_SDD_SECTION:` ni `{{AXIOM_STOP_EXITS}}`; el texto indica que `axiom sdd gate record` es el único modo de registrar la decisión (ver hueco G-4); ningún orquestador nombra `--handoff-policy`.
  - Comprobación: `go test ./internal/components/sdd/ -count=1`

- [ ] T-21 Skills, contrato de status, convención y pines de assets (≈220 l)
  - Ficheros: `internal/assets/skills/sdd-spec/SKILL.md` (`:240-250` Pregunta de Transición → «Listo para la fase siguiente; en `checkpointed`, el orquestador presenta la parada de la puerta con sus tres salidas»; `:269` → «Antes de la salida 2 de la parada, los artefactos quedan confirmados en la rama del cambio; publicarla es decisión de la persona»), `internal/assets/skills/sdd-design/SKILL.md` (`:183-193` y `:208`, igual), `internal/assets/skills/_shared/sdd-status-contract.md` (`:127-135`: kickoff sin `handoffPolicy`; gates con `handedOff?`/`handedOffTo?`; `gateQuestion` con tres opciones o dos más `handOffUnavailable`; `verify` multirrol por `role-apply`), `internal/assets/skills/_shared/openspec-convention.md` (`:26,47`: `handoff.md` pasa a «legado: lo escribían versiones anteriores; hoy solo `axiom handoff create`»), `internal/assets/assets_test.go` (pines de `:2422-2470` y `:3798-3840` ampliados; pin «ningún asset ofrece generar `handoff.md` o continuar» y «ningún asset prescribe `--handoff-policy`»).
  - Cubre: REQ-26.8 (escenarios de la Pregunta de Transición y de «subir»), REQ-26.9 (ningún asset nombra el flag como obligatorio), REQ-21.15 · DD-12.
  - Aceptación: `sdd-spec` y `sdd-design` remiten a las tres salidas; ningún asset contiene la Pregunta de Transición ni prescribe `--handoff-policy`; los pines existentes de la sección de puertas (incluida la prohibición de `gateQuestion`/`gateCriteria`/`gateEvidence` en esa sección) siguen en verde; el contrato de *Lossless Blocking Prompt* (`internal/assets/blocking_prompt_contract_test.go`) pasa sin cambiar sus expectativas.
  - Comprobación: `go test ./internal/assets/ ./internal/components/sdd/ -count=1`

---

## WU-11 · Documentación, goldens y verificación final

Commit: `docs: describir las paradas con tres salidas y regenerar las copias versionadas`

- [ ] T-22 Documentación versionada a mano (≈120 l)
  - Ficheros: `AGENTS.md` (`:16`), `GEMINI.md` (`:47`), `docs/manual-de-inicio.md` (`:230` sobre las paradas y `:515-566`, que pasa a describir `handoff.md` como utilidad general fuera del flujo SDD).
  - Cubre: REQ-26.8, REQ-21.14 (retirado; migración) · DD-12.
  - Aceptación: los tres documentos describen las tres salidas y la regla de «PR fusionada» sin contradecir la doctrina generada; el manual no presenta `handoff.md` como paso del flujo SDD; los pines de `internal/workspace/docs_contract_test.go` siguen en verde.
  - Comprobación: `go test ./internal/workspace/ -count=1`

- [ ] T-23 Regenerar goldens y `.codex/AGENTS.md` (autoría ≈40 l; el resto es generado y se excluye del recuento)
  - Ficheros: `testdata/golden/*.golden` (regenerados con `-update`), `.codex/AGENTS.md` (edición manual acotada).
  - Cambio: ejecutar `go test ./internal/components/ -run Golden -update -count=1` y, después, sin `-update`. Los goldens nunca se concilian a mano. `.codex/AGENTS.md` se edita a mano solo en los párrafos afectados (paradas, compuertas, `handoff_policy`, `handoff.md`), copiando literalmente el texto que proyectan los assets, como en el precedente `9f4bd977`; no se ejecuta `axiom sync` porque está prohibido ejecutar binarios (G-1, resuelto).
  - Cubre: REQ-26.8 (paridad: texto de las tres salidas idéntico en todos los runtimes) · DD-12, DD-14.
  - Aceptación: los goldens pasan sin `-update`; `.codex/AGENTS.md` contiene «### Stops and exits» y los cinco textos de ramas que pinta `internal/workspace/docs_contract_test.go`; el texto de las tres salidas es idéntico al de `StopExits()`.
  - Comprobación: `go test ./internal/components/ -run Golden -count=1` y `go test ./internal/workspace/ -count=1`

- [ ] T-24 Verificación final del cambio (≈60 l de ajustes si aparecen)
  - Ficheros: ninguno de partida; cualquier ajuste mínimo que surja de las pruebas.
  - Cambio: `go vet ./...`; `go test` de todos los paquetes tocados (`./internal/kickoff/ ./internal/consentenvelope/ ./internal/sddstatus/ ./internal/branch/... ./internal/cli/ ./internal/components/... ./internal/assets/ ./internal/workspace/ ./internal/dashboard/`); ratchets de rechazos (cada rechazo nuevo con continuación ejecutable o anotación `refusal:by-design` justificada) y de deadcode (la retirada de `closure.go` y `writeIntegrationHandoff` no deja código muerto).
  - Cubre: criterios de éxito de `proposal.md` §11 («CI verde», «Paridad y goldens»).
  - Aceptación: `go vet ./...` y todas las pruebas anteriores en verde; ningún binario compilado ni ejecutado; `gates.yaml`/`kickoff.yaml` de este cambio sin tocar.
  - Comprobación: `go vet ./...` y `go test ./internal/kickoff/ ./internal/consentenvelope/ ./internal/sddstatus/ ./internal/branch/... ./internal/cli/ ./internal/components/... ./internal/assets/ ./internal/workspace/ ./internal/dashboard/ -count=1`

---

## Trazabilidad requisito → tarea

| Requisito | Tareas |
|---|---|
| REQ-21.2, REQ-25.40 (parada ODD en la doctrina) | T-19 |
| REQ-21.4, REQ-21.5, REQ-26.9 | T-16, T-17, T-20, T-21 |
| REQ-21.8 a REQ-21.12 (tres salidas por puerta) | T-07, T-09, T-20, T-21 |
| REQ-21.13 (aviso sin relevo) | T-07, T-12 |
| REQ-21.14 (retirado), REQ-21.15 | T-11, T-12, T-22 |
| REQ-26.1 | T-05, T-06, T-09 |
| REQ-26.2 | T-08, T-09 |
| REQ-26.3 | T-01, T-02, T-05, T-06, T-07 |
| REQ-26.4 | T-03, T-04, T-06, T-07 |
| REQ-26.5 | T-03, T-04, T-07 |
| REQ-26.6 | T-06, T-09 |
| REQ-26.7 | T-01, T-02, T-10 |
| REQ-26.8 | T-18, T-20, T-21, T-22, T-23 |
| REQ-26.10 | T-18, T-19 |
| REQ-26.11, REQ-25.20, REQ-25.45 | T-02, T-13, T-14, T-15 |

## Matriz de amenazas (design §9) → tareas

| Frontera | Tareas con casos de prueba |
|---|---|
| Selección de repositorio git | T-03 (citado y `-C` absoluto), T-04 (cwd distinto, `gitRoot` por repositorio en multirepo) |
| Estado del commit | T-03 (pathspec explícito, nunca `-a`), T-04 (preparado, sin seguimiento, carpeta entera, exenciones), T-07 (observador sin `commit`) |
| Estado del push | T-03 (rama concreta), T-04 (rama de rol ⇒ seguimiento), T-07 (observador sin `push`/`fetch`/`pull`/`ls-remote`) |
| Órdenes de PR | T-03 (sin `gh`/`glab`/`&&`; `head`/`base` separados), T-07 (JSON) |

## Riesgos de planificación

- **Cuello de botella en `sdd_gate.go`:** lo editan T-07, T-12 y T-17 de forma secuencial; el orden WU-3 → WU-6 → WU-8 evita conflictos internos. Un único escritor, sin paralelismo entre estas tres.
- **Solapamiento con `fix/bloqueos-prueba-multirepo-r2`** (hoy sin commits propios): ediciones mínimas en `governance.go`, `status.go`, `sdd_kickoff.go` y `args.go`; si se fusiona antes, rebase y conflictos resueltos a favor de este diseño (DD-14).
- **WU-6 y WU-9/WU-10 son las de mayor carga de revisión** (≈700 y ≈830 líneas entre las dos de texto); la unidad 6 es sobre todo eliminación.
- **Paralelismo posible** (si hubiera más de un escritor en worktrees aislados, hoy no autorizado): WU-5 y WU-7 son independientes entre sí tras WU-1; WU-9 es independiente de WU-1 a WU-8. En la práctica se ejecutan en el orden indicado.
- **Verificación de las fixtures que pasan `--handoff-policy`** (`e2e/`, `cmd/axiom/`, `bench/`): no se ejecutan si compilan o lanzan binarios; la compatibilidad queda cubierta por las pruebas en proceso de T-16 y T-17.
