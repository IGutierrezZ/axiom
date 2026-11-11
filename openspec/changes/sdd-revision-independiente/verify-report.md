verdict: PASS WITH WARNINGS

# Informe de verify: sdd-revision-independiente (rol `fullstack`)

## Alcance

- Cambio `sdd-revision-independiente`, rol único `fullstack` (kickoff `continuous`), rama `inc/sdd-revision-independiente`, worktree `C:\repos\axiom-wt\sdd-revision-independiente\axiom`.
- Diff del rol: `git merge-base main HEAD` (7ce5a60c) frente al árbol de trabajo, excluida la carpeta del cambio. Sin ficheros sin seguimiento (`git status --short` vacío).
- Commits: 32faf4b3 (plan), 99d439d2 U1, 65047e2f U2, 74c12aa8 U3, ad2b2ae0 U4, 35832ea5 U5.
- Artefactos leídos: `proposal.md`, `specs/*/spec.md`, `tasks.md`, `apply-progress.md` (el `design.md` por referencia). TDD desactivado (`strict_tdd: false`).
- Contrastado con REQ-28.1 a REQ-28.14 y REQ-21.4.

## Progreso observado

`tasks.md`: 36 tareas marcadas `[x]`, 0 sin marcar. No se reescribe ningún estado.

## Comprobaciones ejecutadas

| Comando | Resultado |
|---|---|
| `go vet ./...` | exit 0, sin salida |
| `go test ./internal/assets/ ./internal/kickoff/ ./internal/sddstatus/ ./internal/multirole/ ./internal/components/sdd/ -count=1` | exit 0: assets ok (18,0 s), kickoff ok (1,1 s), sddstatus ok (60,9 s), multirole ok (0,6 s), components/sdd ok (67,0 s) |
| `go test ./internal/cli/ -run` con el patrón de la instrucción (Kickoff, HandOff, SealBranch, GuardProject, TestSDDPreflightHook) `-count=1` | exit 0: ok (42,4 s) |

No ejecutadas (por instrucción): `go test ./internal/cli` y `./internal/reviewtransaction` sin `-run` (más de 60 min en esta máquina) y `go test ./...` completo. No se compiló ningún binario suelto.

## Hallazgos funcionales

### CRITICAL

Ninguno.

### WARNING

- **W1. REQ-28.1, 28.2, 28.3, 28.6, 28.7, 28.8, 28.9, 28.10 y 28.14 son prosa de orquestador y de skills: no hay prueba de comportamiento en runtime.** Su cobertura es estática: pins de texto en `internal/assets/independent_review_boundary_test.go` (`TestIndependentReviewBoundaryVerifyPassages` :98, `...JudgmentAndCoveragePassages` :162, `...OrchestratorPassages` :232, `...TemplateParity` :285, `...PinDetectsMutations` :326, `...IsNotAWholeFileSweep` :356), el registro de la sección (`internal/assets/orchestrator_drift_ratchet_test.go:46`, `internal/components/sdd/orchestrator_shared_sections_test.go:129`) y los goldens de proyección. No se ha observado que un orquestador real lance verify, juicio o cobertura en el momento indicado. El propio `apply-progress.md` lo declara.
- **W2. Pruebas ajenas no ejecutadas ni reproducidas aquí.** `internal/cli` y `internal/reviewtransaction` completos, y `go test ./...`. `apply-progress.md` afirma 91 paquetes en verde salvo los fallos ambientales; esta verificación no lo reprodujo.

### SUGGESTION

- **S1.** La interacción de preferencia (REQ-28.14) queda cubierta en el hook solo por `TestSDDPreflightHookIgnoresIndependentReviewPreferenceInteraction` (`internal/cli/sdd_preflight_hook_test.go`); la validación de la respuesta de texto `After apply: <label>; Coverage: <label>` es texto del orquestador sin prueba de ejecución. Aceptable por diseño (el hook no cambia).

## Cobertura de requisitos (evidencia)

| REQ | Estado | Evidencia |
|---|---|---|
| 28.12 `continuous` solo con `[fullstack]` | Cumple | `internal/kickoff/schema.go:125` (`ErrContinuousRequiresFullstack`), `:131` (`checkContinuousRoster`, compara identidad, no política de compuerta), `:150` (`ValidateForSeal`); `internal/kickoff/args.go:284-295` (rechazo previo a resolver `--change`); `internal/cli/sdd_kickoff.go:78` (defensa en profundidad antes de `kickoff.Seal`) y línea de ayuda. Pruebas: `TestKickoffValidateForSealContinuousRoster`, `TestParseSealArgsContinuousRosterRule`, `TestRunSDDKickoffSealContinuousRosterRule` (sin `kickoff.yaml`, `seal` repetido intacto, ayuda), `TestParseKickoffReadsContinuousWithRoles` (lectura tolerante). El mensaje nombra `--execution-style checkpointed` y quitar `--role`. |
| 28.13 `--from-session-pace auto` | Cumple | `args.go:292` (`derivedFromPace := sessionPaceGiven && !executionStyleExplicit`; solo cambia el prefijo del mensaje, no deriva `checkpointed`); `TestParseSealArgsContinuousRosterRule`, `...SkipsInfer`. |
| 21.4 retro-sello exento | Cumple | `internal/kickoff/types.go:80` (`SealedByInferred`), `infer.go` (uso y comentario P2), `schema.go:153` (exención); `TestInferKickoffContinuousWithDetectedRolesPassesValidateForSeal`; `seal --infer` en `TestRunSDDKickoffSealContinuousRosterRule`. |
| 28.4 verify por rol y sección técnica | Cumple (estático) | `internal/assets/skills/sdd-verify/references/technical-compliance.md` §1 (diff con `git -C "<worktree>"`, `merge-base <comparedTo> HEAD`, `ls-files --others`), `report-format.md` (Cumplimiento técnico, Estado, prohibición de `verdict:` en la sección), `SKILL.md` (regla dura y fila de compuerta). El verify global y su barrera no cambian: `internal/multirole` ok sin tocar `barrier.go`. |
| 28.5 fuentes y forma de hallazgos | Cumple (estático) | `technical-compliance.md` §2 (fuentes a-d, ausentes como `no aplicable`, nunca modificar `technicalContext.reference`) y §3 (forma `severidad · fuente · regla · path:línea`, observación sin regla). |
| 28.6 juicio solo tras verify limpio | Cumple (estático) | `internal/assets/skills/_shared/sdd-orchestrator-sections.md`, bloques «Three points» y «Judgment»; `technical-compliance.md` §4 (limpio = sin CRITICAL y comprobaciones obligatorias superadas; parcial no limpio); petición explícita prevalece (A-2); línea de coste. |
| 28.7 objetivo SDD y arreglos acotados | Cumple (estático) | `internal/assets/skills/judgment-day/references/prompts-and-formats.md` `## SDD Role Target` (identidad inmutable con sha256, `allowedEditRoots` ∩ worktree, nunca `technicalContext.reference`, hallazgo fuera de raíces impide `APPROVED`, confirmación por estilo); `judgment-day/SKILL.md` (activación y fila de decisión). |
| 28.8 informe vigente tras corrección | Cumple (estático) | `SDD Role Target` «Final verification» y `report-format.md` (contexto histórico). |
| 28.9 cobertura por unión | Cumple (estático) | `internal/assets/skills/_shared/sdd-coverage-review.md` (dos revisores ciegos `jd-judge-a/b`, unión deduplicada, revisor sin resultado declarado, informe sin huecos declarado, rondas fechadas, informes fuera del digest de puertas, sin autocorrección); reglas en `sdd-spec/SKILL.md` y `sdd-design/SKILL.md`; el fichero se instala por derivación de `SharedSkillFileNames` (`internal/assets/shared_skill_files.go:22`). |
| 28.10 paradas presentan resultados | Cumple (estático) | `sdd-orchestrator-sections.md:86` (frase en la sección de puertas) y bloque «Results» (:116); resultados ausentes o no limpios se presentan como tales; sin cambio de digest ni de `gates.yaml`. |
| 28.11 frontera con RDD | Cumple | Pins de vocabulario en los tramos de `independent_review_boundary_test.go` (solo pasajes nuevos, no barrido de ficheros: `...IsNotAWholeFileSweep`); guardas de `internal/sddstatus` ok sin modificación. |
| 28.1 a 28.3 ofertas, estilo sellado, no bloqueo | Cumple (estático) | Sección `SDD Independent Review` (:94 y siguientes): tres puntos y solo ellos, `none`/`no` sin lanzamiento propio con petición explícita atendida, `continuous` sin paradas, `checkpointed` conserva la parada, preferencia solo en caché de sesión (no en `kickoff.yaml` ni `gates.yaml`), sesión sin preferencia, diff vacío sin juicio, verify por rol como excepción al enrutado por dependencias, runtimes sin subagentes. |
| 28.14 interacción de preferencia | Cumple (estático) | Bloque «Preference»: dos preguntas `after_apply` (`verify`, `verify-judgment`, `none`) y `coverage` (`yes`, `no`), tokens no traducibles, sintaxis `After apply: <label>; Coverage: <label>`, sin valor por defecto, sin prefijo del pre-vuelo, descripción de `verify-judgment` con correcciones acotadas en `continuous`. Hook sin cambios (prueba citada en S1). |

Proyección a los doce runtimes: 12 plantillas `internal/assets/*/sdd-orchestrator.md` (+4 líneas cada una), `internal/assets/claude/sdd-orchestrator-workflow.md` y 13 goldens regenerados; paridad por `TestIndependentReviewBoundaryTemplateParity`. `.codex/AGENTS.md` sin cambios, conforme a DD-06.

## Cumplimiento técnico

Fuentes consultadas:

- (a) Skills de `skill_resolution` del apply (según `apply-progress.md`): contrastadas con commits Conventional Commit por unidad, sin atribución, y pruebas junto al código.
- (b) Skills del registro cuyos disparadores casan con los ficheros tocados (Go en `internal/kickoff` e `internal/cli`, prosa de skills): sin hallazgos.
- (c) `technicalContext`: no aplicable (`axiom.yaml` no declara `technical_context`).
- (d) `openspec/config.yaml`: `rules.apply` (patrones de `internal/`, pruebas) y `rules.verify` (`go test ./...`, `go vet ./...`, comparar con cada escenario).

Hallazgos de cumplimiento: ninguno con regla citable incumplida.

- Comprobación `go test ./...` de `rules.verify`: no ejecutada completa por esta verificación (limitación W2); se ejecutaron las acotadas de la instrucción y `go vet ./...` completo.
- Observación sin regla asociada: `apply-progress.md` documenta que se actualizó el hash de Kilocode en `TestKilocodeReviewSettingsMatchCurrentMainBaseline` con un comentario que lo declara deliberado, que es la convención del propio test.

## Avisos de contexto

No aplican: no hay `technicalContext.maintained`. Además, el diff no toca `go.mod`, manifiestos de dependencias, `Makefile`, `Dockerfile` ni `.github/workflows/*`.

## Limitaciones

- Fallos ajenos, preexistentes en `main` o de Windows (no ejecutados aquí, solo citados): `internal/app` `TestTuiInstallOnThenSyncPreservesAndRefreshesOpenCodeActivation`, `internal/backup` `TestSnapshotterSkipsUnixSockets`, `internal/components/filemerge` `TestWriteFileAtomicFollowsDirectoryJunction*`.
- `internal/cli` completo y `internal/reviewtransaction` no ejecutados (más de 60 min).
- Fuera del alcance de esta verificación: el comportamiento en runtime del orquestador (W1).

## Resumen

Implementación completa (36/36 tareas) y conforme a REQ-28.1 a REQ-28.14 y REQ-21.4 según el código y las pruebas ejecutadas: `go vet` limpio y todas las pruebas permitidas en verde. 0 CRITICAL, 2 WARNING (cobertura de prosa solo estática; suite completa no reproducida), 1 SUGGESTION. Siguiente paso recomendado: `sdd-archive`.

**Estado:** limpio
