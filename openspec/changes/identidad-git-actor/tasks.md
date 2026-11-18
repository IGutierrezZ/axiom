# Tareas: identidad git como actor de los artefactos versionados (identidad-git-actor, C-07-a)

> Fase `tasks` · rama `inc/identidad-git-actor` · worktree `C:\repos\axiom-wt\identidad-git-actor\axiom` · kickoff `continuous`, rol único `fullstack` (`tasks_file: tasks.md`). Entradas: `proposal.md`, `design.md` (autoridad técnica: D1 a D9) y `specs/change-actor-identity/spec.md` (REQ-42.1 a REQ-42.9). Sin `technical_context` en `axiom.yaml`: no hay `context-impact.md`.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ≈ 600–750 de autor (pruebas incluidas; sin goldens ni ficheros generados) |
| 400-line budget risk | High |
| Chained PRs recommended | No (estrategia `exception-ok`: una sola PR con `size:exception`, preferencia del usuario) |
| Suggested split | Una sola PR; cuatro commits convencionales (U1 a U3 y cierre solo si hay ajustes), revertibles por separado |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

Nota: `exception-ok` ya recoge la aceptación de `size:exception`, así que no hay decisión pendiente antes de `sdd-apply`. Nada de PR encadenadas. El diseño ya anticipa 550-700 líneas (Preguntas abiertas) frente a las 150-250 de la propuesta.

### Previsión por unidad (líneas de autor, adiciones más borrados)

| Unidad | Líneas | Nota |
|---|---|---|
| U1 | ≈ 330 | `gitidentity` (≈ 90 producción, ≈ 220 pruebas) y ampliación de la prueba de arquitectura |
| U2 | ≈ 80 | Constante `FallbackActor`, comentarios y pruebas de `kickoff` |
| U3 | ≈ 300 | Adaptador CLI (≈ 70 producción), `TestMain` y pruebas de integración y extremo a extremo |
| U4 | ≈ 0 | Verificación y cierre |

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| U1 | Paquete hoja `internal/gitidentity` (saneado, formato, `Resolve`) y su inclusión en la prueba de arquitectura de `gitcommon` | PR única, commit 1 | `go test ./internal/gitidentity/...` y `go test ./internal/gitcommon/ -run Architecture` | `go test` en proceso con repositorios git temporales y entorno aislado (`GIT_CONFIG_GLOBAL`, `GIT_CONFIG_NOSYSTEM=1`); sin binario suelto | Revertir el commit: el paquete y su línea en la prueba de arquitectura; nada más depende de él hasta U3 |
| U2 | `kickoff.FallbackActor`, `ToKickoff` sin literal, comentarios de tipo y pruebas de registros anteriores y round-trip | PR única, commit 2 | `go test ./internal/kickoff/... -run "Gate\|Seal\|Kickoff\|Ledger\|Schema\|ToKickoff"` | `go test` con `AppendGate`/`LoadGates` y `Seal`/`Load` sobre directorios temporales | Revertir el commit (U1 queda intacta; U3 usa la constante, revertir U3 antes) |
| U3 | Adaptador CLI: `sdd_actor_identity.go`, inyección en `gate record` y `kickoff seal`, ayuda, `TestMain` determinista y pruebas | PR única, commit 3 | `go test ./internal/cli/ -run "ActorIdentity\|SDDGate\|KickoffSeal\|Seal"` | `go test` con workspaces temporales reales (`workspacetest.Embedded`, `workspacetest.Canonical`) y dos pruebas de extremo a extremo con git real aislado | Revertir el commit: vuelven el literal `cli` y la ayuda anterior; U1 y U2 quedan inertes |
| U4 | Verificación acotada y cierre | PR única, commit 4 (solo si hay ajustes) | Batería de cierre (ver Convenciones) | `go vet` y `go test` por paquete, sin binarios sueltos | Sin código propio |

## Convenciones de verificación (valen para todas las unidades)

- Runner: `go test` en proceso, por paquete, desde `C:\repos\axiom-wt\identidad-git-actor\axiom`. En `internal/cli` SIEMPRE filtrado con `-run` (el paquete entero es lento en Windows). NUNCA compilar binarios sueltos de `axiom` (`go build -o`, `go install`): Cortex XDR borra `axiom.exe`.
- TDD estricto desactivado (`openspec/config.yaml`: `strict_tdd: false`): las pruebas se escriben junto con el código; se observa el rojo de las de comportamiento nuevo cuando sea práctico (marcadas RED en las tareas).
- Toda prueba que ejecute git real aísla el entorno: `HOME`/`USERPROFILE` temporales, `GIT_CONFIG_GLOBAL` a un fichero temporal (vacío o con identidad), `GIT_CONFIG_NOSYSTEM=1` y, para el directorio no repositorio, `GIT_CEILING_DIRECTORIES` al padre del temporal; se omite con `-short` y sin git, como `isolateIntegrateGit` (`internal/cli/gittest_helpers_test.go:16-34`).
- Comandos de cierre de cada unidad: `gofmt -l` de los paquetes tocados (salida vacía); `go vet` de los paquetes tocados; `go test ./internal/cli -run TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` si la unidad toca rechazos; `bash scripts/deadcode-ratchet.sh` si añade o retira funciones.
- Cada unidad cierra con un commit Conventional en castellano, sin `Co-Authored-By` ni atribución de IA, con sus pruebas junto al comportamiento. Anotar la identidad del commit en la tarea de cierre.
- Una ruta citada con `(read-only)` solo se lee.
- Heurística de ~400 líneas por tarea: solo orientativa; no es tope ni motivo de dividir artificialmente.

---

## U1 · Paquete `internal/gitidentity` y prueba de arquitectura

Satisface REQ-42.6, REQ-42.7 y la base de REQ-42.1/42.3/42.5. Diseño: D1, D2, D3, D9.
Commit sugerido: `feat(gitidentity): resolver la identidad git saneada para actor y sellador`

Amenazas aplicables de la matriz de diseño: selección de repositorio git (directorio relativo, `GIT_DIR` heredada) y salida del subproceso hacia artefactos y terminal (controles, UTF-8 no válido, `<`/`>`, longitud, git ausente). Las filas N/A se omiten.

- [ ] 1.1 RED (saneado y formato, puras): crear `internal/gitidentity/gitidentity_test.go` con `TestSanitize` (recorte; CR, LF, TAB, ESC, DEL y C1 → ausente; UTF-8 no válido → ausente; retirada de `<`/`>` con `Ana <admin> Pérez` → `Ana admin Pérez`; parte solo de espacios → ausente; truncado a 120 runes con 500 caracteres sin cortar una rune ni dejar espacio final) y `TestIdentityActorAndMissing` (`Actor()` con exactamente un `<` y un `>`; actor ≤ 256 runes con nombre y email de 500; `Complete()`; `Missing()` en orden fijo `user.name`, `user.email` para ninguna, solo nombre y solo email; el actor nunca coincide con `cli` ni `inferred`). Cubre REQ-42.7 (recorte, controles, UTF-8 no válido, `<`/`>`, tope) y REQ-42.5 (parte vacía = ausente). Verificación: `go test ./internal/gitidentity/ -run 'Sanitize|IdentityActorAndMissing'` en rojo (no compila).
- [ ] 1.2 GREEN: crear `internal/gitidentity/gitidentity.go` con `KeyName`, `KeyEmail`, `MaxPartRunes`, `Identity`, `New`, `Sanitize` (pasos 1 a 5 de D3, en ese orden), `Complete`, `Actor` (`Name + " <" + Email + ">"`) y `Missing`. Solo biblioteca estándar. Verificación: `go test ./internal/gitidentity/ -run 'Sanitize|IdentityActorAndMissing'`.
- [ ] 1.3 RED (resolución con git real): añadir a `internal/gitidentity/gitidentity_test.go` un ayudante de entorno aislado y `TestResolveLocalConfig` (identidad local con global vacía), `TestResolveLocalOverridesGlobal` (local B prevalece sobre global A), `TestResolveNonRepositoryUsesGlobal` (directorio temporal sin `.git`, `GIT_CEILING_DIRECTORIES`, identidad global completa), `TestResolveIgnoresGitAuthorEnv` (`GIT_AUTHOR_*`/`GIT_COMMITTER_*` definidas y configuración vacía → ausente), `TestResolveIgnoresRedirectingGitDir` (`GIT_DIR` apuntando a otro repositorio con otra identidad no redirige), `TestResolveWithoutGitIsAbsent` (`t.Setenv("PATH", t.TempDir())`, sin pánico) y `TestResolveRelativeDirIsAbsent` (directorio relativo → ausente). Se omiten con `-short` y sin git. Cubre REQ-42.6 y las amenazas «selección de repositorio» y «salida del subproceso». Verificación: `go test ./internal/gitidentity/ -run 'TestResolve'` en rojo (no compila).
- [ ] 1.4 GREEN: añadir `Resolve(ctx, dir)` en `internal/gitidentity/gitidentity.go`: dos llamadas `gitcommon.RunWithTimeout(ctx, dir, resolveTimeout, "config", "--get", key)` (`resolveTimeout` = 5 s), cualquier error, código distinto de cero o salida vacía → clave ausente, resultado por `New(name, email)`; nunca falla ni entra en pánico. Verificación: `go test ./internal/gitidentity/...` en verde.
- [ ] 1.5 Prueba de arquitectura: en `internal/gitcommon/architecture_test.go` (`TestArchitectureOnlyExecGoLaunchesProcesses`, `:106-153`) añadir `collectSources(t, filepath.Join("..", "gitidentity"), "internal/gitidentity")` con un comentario que cite la capacidad `change-actor-identity` y `"internal/gitidentity/gitidentity.go"` a la lista de etiquetas que el recorrido debe ver. Verificación: `go test ./internal/gitcommon/ -run Architecture` en verde.
- [ ] 1.6 Cierre de U1: `gofmt -l ./internal/gitidentity ./internal/gitcommon`, `go vet ./internal/gitidentity/... ./internal/gitcommon/...`, `go test ./internal/gitidentity/... ./internal/gitcommon/...`, `bash scripts/deadcode-ratchet.sh` (añade funciones exportadas aún sin llamador de producción: si el ratchet lo rechaza, dejar el commit de U1 sin ratchet y comprobarlo en el cierre de U3) y commit `feat(gitidentity): resolver la identidad git saneada para actor y sellador`. Anotar el commit.

---

## U2 · Dominio `kickoff`: constante de reserva y compatibilidad

Satisface REQ-42.8 y la parte de dominio de REQ-42.4. Depende de nada (independiente de U1). Diseño: D4.
Commit sugerido: `feat(kickoff): constante FallbackActor y compatibilidad de registros de actor`

- [ ] 2.1 RED: en `internal/kickoff/schema_test.go` (o `ledger_test.go`, donde estén los helpers de carga) añadir `TestLegacyActorRecordsStillLoadAndValidate` (`gates.yaml` con actores `cli`, `maintainer`, `release-manager` y `kickoff.yaml` con `sealed_by: cli` e `inferred` cargan y validan igual), `TestNewActorRecordCoexistsWithLegacy` (un registro `cli` intacto más uno nuevo `Nombre <email>` tras `AppendGate`), `TestIdentityFormatActorValidates` (`Ana Pérez <ana@example.com>` pasa la validación), `TestActorIdentityYAMLRoundTrip` (`Íñigo: Gutiérrez # Zufia <inigo@example.com>` por `AppendGate`/`LoadGates` y `Seal`/`Load`, sin pérdida ni truncado), `TestValidateForSealKeepsInferredExemption` (el resultado con `sealed_by: inferred` no cambia) y `TestToKickoffSealedByIsFallbackActor` (`ToKickoff` devuelve `kickoff.FallbackActor` = `cli`). Cubre REQ-42.8, REQ-42.4 (exención), REQ-42.7 (round-trip). Verificación: `go test ./internal/kickoff/ -run 'Legacy|FallbackActor|IdentityFormat|ActorIdentityYAML|InferredExemption'` en rojo (no compila por la constante).
- [ ] 2.2 GREEN: en `internal/kickoff/types.go` añadir la constante `FallbackActor = "cli"` junto a `SealedByInferred` (`:80`) y documentar en `Kickoff.SealedBy` (`:146`) y `GateRecord.Actor` (`:191`) el valor por defecto (identidad git `Nombre <email>`), la reserva `cli`, `inferred` en el retro-sellado y que el valor no se valida; en `internal/kickoff/args.go` (`:375`) hacer que `ToKickoff` use `SealedBy: FallbackActor` y documentar que el adaptador CLI la sustituye por la identidad git (sigue pura, mismo resultado byte a byte). Ningún tipo cambia. Verificación: `go test ./internal/kickoff/...` en verde.
- [ ] 2.3 Cierre de U2: `gofmt -l ./internal/kickoff`, `go vet ./internal/kickoff/...`, `go test ./internal/kickoff/... -run "Gate|Seal|Kickoff|Ledger|Schema|ToKickoff"` y commit `feat(kickoff): constante FallbackActor y compatibilidad de registros de actor`. Anotar el commit.

---

## U3 · Adaptador CLI: `gate record` y `kickoff seal`

Satisface REQ-42.1, REQ-42.2, REQ-42.3, REQ-42.4, REQ-42.5, REQ-42.6 (multirepo), REQ-42.9. Depende de U1 y U2. Diseño: D5, D6, D7, D8.
Commit sugerido: `feat(cli): gate record y kickoff seal registran la identidad git como actor`

Amenazas aplicables: selección de repositorio (git se ejecuta desde `loc.WorkspaceRoot`, nunca `-C` ni el directorio del proceso) y salida del subproceso hacia artefactos (aviso solo por stderr, nunca en stdout ni JSON).

- [ ] 3.1 TestMain determinista: en `internal/cli/protocol_probe_test.go` (`TestMain`, `:52`) fijar `resolveActorIdentity` a un resolvedor que devuelve `gitidentity.New("Prueba Axiom", "prueba@ejemplo.invalid")` (D8) y añadir en el comentario de la función la razón (sin él, las pruebas existentes lanzarían git y dependerían de la máquina). Depende de 3.3 para compilar: se hace junto con 3.3 y se verifica con 3.7. Cubre D8.
- [ ] 3.2 RED (adaptador): crear `internal/cli/sdd_actor_identity_test.go` con los ayudantes `stubActorIdentity`, `forbidActorIdentity`, `useRealActorIdentity` y `captureActorIdentityWarnings` (guardar/restaurar con `t.Cleanup`, como `captureSealWarnings` en `sdd_kickoff_branch_test.go:34-41`) y los casos de `gate record`: `TestGateRecordUsesGitIdentityAsActor` (identidad completa → `actor: Ana Pérez <ana@example.com>`, éxito, cero avisos), `TestGateRecordExplicitActorSkipsResolution` (con `forbidActorIdentity`, con y sin identidad; REQ-42.2), `TestGateRecordWithoutIdentityFallsBackToCli` (tabla: ninguna, solo nombre, solo email → `actor: cli`, exactamente una línea que empieza por `Aviso: `, claves nombradas según `Missing()`, menciona `--actor`; texto exacto de D5), `TestGateRecordRejectedEmitsNoIdentityWarning` (rechazo de evidencia o `--hand-off` sin artefactos confirmados: sin registro y sin aviso; D6), `TestGateRecordJSONShapeUnchanged` (con y sin identidad el mismo conjunto de claves, sin `actor` ni `warnings`, stdout JSON válido, aviso fuera de stdout; REQ-42.9) y `TestGateRecordHelpMentionsGitIdentity` (la ayuda de `record` menciona la identidad git; REQ-42.1). Cubre REQ-42.1, REQ-42.2, REQ-42.5, REQ-42.9. Verificación: `go test ./internal/cli/ -run 'ActorIdentity|GateRecord'` en rojo (no compila).
- [ ] 3.3 GREEN (adaptador y `gate record`): crear `internal/cli/sdd_actor_identity.go` con `resolveActorIdentity = gitidentity.Resolve`, `actorIdentityWarnings io.Writer = os.Stderr`, `actorTarget` (`gateActorTarget`, `sealActorTarget`), `defaultActor(workspaceRoot, target)` (con `context.Background()`; identidad completa → actor y aviso vacío; si no → `kickoff.FallbackActor` y el aviso con `pathquote.Quote(workspaceRoot)`, `<faltan>` y `<órdenes>` según D5) y `emitActorIdentityWarning` (`Fprintln` si no está vacío). En `internal/cli/sdd_gate.go` sustituir el literal de `runSDDGateRecord` (`:100-103`) por el bloque de D6 (resolución tras guarda de rama, carga de kickoff y roster), emitir el aviso solo tras `AppendGate`/`AppendGateChecked` sin error (`:144-151`) y añadir bajo la línea de ayuda de `record` (`:48`) la línea de `--actor` de D6. Verificación: `go test ./internal/cli/ -run 'ActorIdentity|GateRecord'` en verde para las pruebas de 3.2.
- [ ] 3.4 RED (seal): añadir a `internal/cli/sdd_actor_identity_test.go` `TestKickoffSealUsesGitIdentityAsSealedBy` (`sealed_by: Ana Pérez <ana@example.com>`, éxito, sin aviso), `TestKickoffSealWithoutIdentityFallsBackToCli` (ninguna, solo nombre, solo email → `sealed_by: cli`, una línea `Aviso: `, texto de seal de D5 sin `--actor`), `TestKickoffSealInferKeepsInferredAndSkipsGit` (con `forbidActorIdentity`, con y sin identidad → `sealed_by: inferred`, sin aviso; REQ-42.4), `TestKickoffSealExistingKickoffEmitsNoWarning` (re-seal con kickoff ya sellado, `wrote == false`, sin aviso; D7), `TestKickoffSealRejectsActorFlags` (`--actor x` y `--sealed-by x` rechazados como `bandera desconocida` sin escribir `kickoff.yaml`; REQ-42.3), `TestGateAndSealShareIdentity` (`sealed_by` y `actor` idénticos carácter a carácter) y `TestBranchWarningsStayEmptyWithoutIdentity` (`sealBranchWarnings` y `writerBranchWarnings` vacíos en un entorno sin identidad; REQ-42.5). Cubre REQ-42.3, REQ-42.4, REQ-42.5. Verificación: `go test ./internal/cli/ -run 'ActorIdentity|KickoffSeal'` en rojo.
- [ ] 3.5 GREEN (seal): en `internal/cli/sdd_kickoff.go` (`runSDDKickoffSeal`), en la rama sin `--infer` y tras `applySealBranchPolicy` (`:90`), asignar `sealing.SealedBy, sealWarning = defaultActor(loc.WorkspaceRoot, sealActorTarget)` y, tras `sealKickoffFile` (`:113`), `if wrote { emitActorIdentityWarning(sealWarning) }`; la rama `--infer` no se toca (D7). Verificación: `go test ./internal/cli/ -run 'ActorIdentity|KickoffSeal'`.
- [ ] 3.6 Multirepo y extremo a extremo: añadir `TestActorIdentityMultirepoUsesSpecsRoot` (`workspacetest.Canonical`: el resolvedor recibe la raíz del repositorio de specs y el actor es el de ese repositorio, en gate y en seal; REQ-42.6) y dos pruebas con `useRealActorIdentity` y git aislado, omitidas con `-short`: `TestKickoffSealRealGitIdentity` (seal en `embeddedSealProject`, `sdd_kickoff_branch_test.go:45`, con `user.name`/`user.email` locales) y `TestGateRecordRealGitIdentityNonRepository` (gate en un workspace no repositorio con identidad en el `GIT_CONFIG_GLOBAL` aislado). Cubre REQ-42.6 (multirepo y local/global). Verificación: `go test ./internal/cli/ -run 'ActorIdentityMultirepo|RealGitIdentity'`.
- [ ] 3.7 Cierre de U3: `gofmt -l ./internal/cli`, `go vet ./internal/cli/...`, `go test ./internal/cli/ -run "ActorIdentity|SDDGate|KickoffSeal|Seal"` (con el `TestMain` determinista de 3.1, las pruebas existentes de seal y gate no dependen de la máquina y no afirman avisos nuevos), `go test ./internal/cli -run TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign`, `bash scripts/deadcode-ratchet.sh` y commit `feat(cli): gate record y kickoff seal registran la identidad git como actor`. Anotar el commit.

---

## U4 · Verificación y cierre

Satisface todos los requisitos (verificación transversal). Depende de U1 a U3.

- [ ] 4.1 Batería acotada de cierre, sin binarios sueltos, desde `C:\repos\axiom-wt\identidad-git-actor\axiom`: `go test ./internal/gitidentity/...`; `go test ./internal/kickoff/... -run "Gate|Seal|Kickoff|Ledger|Schema|ToKickoff"`; `go test ./internal/cli/ -run "ActorIdentity|SDDGate|KickoffSeal|Seal"`; `go test ./internal/gitcommon/ -run Architecture`; `go vet ./internal/gitidentity/... ./internal/kickoff/... ./internal/cli/...`. Antes de cerrar `apply`: `go test ./internal/gitidentity/... ./internal/kickoff/... ./internal/cli/... ./internal/gitcommon/...` (el de `internal/cli` completo puede ser lento: si no cabe, registrarlo como pendiente para `verify`, que ejecuta `go test ./...` y `go vet ./...`).
- [ ] 4.2 Revisar `git -C C:\repos\axiom-wt\identidad-git-actor\axiom log --oneline origin/main..HEAD`: un commit convencional en castellano por unidad, sin atribución de IA; comprobar que no se tocó `internal/kickoff/schema.go`, `internal/kickoff/infer.go`, `internal/cli/review_audit_actor.go`, guías de agente, goldens ni specs vivas (`git -C ... diff --stat origin/main..HEAD`); anotar los commits de U1 a U3 y dejar listos los datos de la PR única con `size:exception` (el empuje y la apertura de la PR son del usuario). Aviso para archive: confirmar en `main` que el prefijo REQ-42 no colisiona con otras hijas de la ola.

---

## Matriz requisito → tarea

| Requisito | Tareas |
|---|---|
| REQ-42.1 (identidad como actor de `gate record`; ayuda) | 3.2, 3.3 |
| REQ-42.2 (`--actor` prevalece sin consultar git) | 3.2, 3.3 |
| REQ-42.3 (sellador con la identidad; `seal` sin `--actor`/`--sealed-by`) | 3.4, 3.5, 3.6 |
| REQ-42.4 (`--infer` conserva `inferred`; exención de `ValidateForSeal`) | 2.1, 3.4, 3.5 |
| REQ-42.5 (reserva `cli` con aviso por stderr sin bloqueo) | 1.1, 3.2, 3.3, 3.4, 3.5 |
| REQ-42.6 (resolución solo desde la configuración git en la raíz del workspace) | 1.3, 1.4, 3.6 |
| REQ-42.7 (saneado, formato y round-trip YAML) | 1.1, 1.2, 2.1 |
| REQ-42.8 (registros anteriores válidos sin migración; comentarios de tipo) | 2.1, 2.2 |
| REQ-42.9 (forma del JSON sin cambios) | 3.2, 3.3 |

| Decisión | Tareas |
|---|---|
| D1 (paquete hoja sobre `gitcommon`) | 1.3, 1.4 |
| D2 (API sin error) | 1.1, 1.2 |
| D3 (saneado) | 1.1, 1.2 |
| D4 (`FallbackActor`) | 2.1, 2.2 |
| D5 (costura y aviso) | 3.2, 3.3 |
| D6 (`gate record`) | 3.2, 3.3 |
| D7 (`kickoff seal`) | 3.4, 3.5 |
| D8 (`TestMain` determinista) | 3.1 |
| D9 (prueba de arquitectura) | 1.5 |

Dependencias entre unidades: U1 y U2 son independientes entre sí; U3 depende de ambas (usa `gitidentity.Resolve` y `kickoff.FallbackActor`); U4 cierra tras U3. Orden de commits: U1, U2, U3, U4. Dentro de U3, 3.1 y 3.3 deben aterrizar juntos para que `internal/cli` compile (el `TestMain` referencia la costura nueva).
