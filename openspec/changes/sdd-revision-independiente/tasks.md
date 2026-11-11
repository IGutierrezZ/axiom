# Tareas: Revisión independiente en SDD (sdd-revision-independiente)

> Fase `tasks` · rama `inc/sdd-revision-independiente` · kickoff `continuous`, rol único `fullstack` (`blocking`) · plan único `tasks.md`.
> Entradas: `proposal.md`, `design.md` (DD-01 a DD-10, corregido tras la validación) y las deltas `specs/*/spec.md` (REQ-28.1 a REQ-28.14, REQ-21.4).
> TDD desactivado (`strict_tdd: false`): las pruebas van junto al cambio. Las pruebas de la matriz de amenazas aplicable (selección de repositorio git, DD-10) se escriben antes de la prosa que fijan y se observan en rojo antes de pasar a verde.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ≈ 1.360–1.550 con las specs delta (≈ 750 ya escritas por la fase `spec`); ≈ 610–800 líneas autoradas de implementación (sin goldens) |
| 400-line budget risk | High |
| Chained PRs recommended | No (el usuario no quiere cadena de PRs: una sola PR con `size:exception`) |
| Suggested split | Una sola PR con `size:exception`; cinco unidades de trabajo, cada una cerrada con su commit Conventional Commit |
| Delivery strategy | exception-ok (`single-pr` del pre-vuelo con `size:exception` ya aceptado por el mantenedor) |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

`size:exception` ya está aceptado por el usuario para este cambio; no hay decisión pendiente antes de `sdd-apply`. Los goldens regenerados no cuentan en el presupuesto de líneas autoradas, pero sí se incluyen en la PR.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| U1 | Regla de sellado `continuous` ⇒ roster `[fullstack]` en `internal/kickoff` y `internal/cli`, con sus pruebas (REQ-28.12, REQ-28.13, REQ-21.4) | PR única | `go test ./internal/kickoff/... ./internal/cli/...` | Pruebas en proceso de `RunSDDKickoff seal` (sin binarios sueltos: Cortex XDR los borra) | `internal/kickoff/*`, `internal/cli/sdd_kickoff*.go` y `internal/cli/sdd_gate_handoff_test.go`; revertible sin tocar prosa |
| U2 | Verify por rol con cumplimiento técnico y estado «limpio»: referencia nueva, `report-format.md`, `SKILL.md` de `sdd-verify`, convención y primer tramo del pin (REQ-28.4, REQ-28.5, REQ-28.8 parcial, REQ-28.11) | PR única | `go test ./internal/assets/ ./internal/components/sdd/` | N/A: solo prosa de skills y su prueba de pin; sin superficie de ejecución | Ficheros de `internal/assets/skills/sdd-verify/**`, la entrada de `openspec-convention.md` y `independent_review_boundary_test.go` (tramo verify) |
| U3 | Juicio sobre objetivo SDD por rol y revisión de cobertura: sección `SDD Role Target`, asset `sdd-coverage-review.md`, reglas de `sdd-spec`/`sdd-design` y segundo tramo del pin (REQ-28.6 a REQ-28.9, REQ-28.11) | PR única | `go test ./internal/assets/ ./internal/components/sdd/` | N/A: solo prosa y pins | Ficheros de `internal/assets/skills/judgment-day/**`, `_shared/sdd-coverage-review.md`, reglas de `sdd-spec`/`sdd-design` y el tramo del pin |
| U4 | Sección compartida `SDD Independent Review`, proyección a los doce orquestadores, ediciones de puertas y grafo, registros, caso del hook y goldens (REQ-28.1, REQ-28.2, REQ-28.3, REQ-28.10, REQ-28.14, REQ-28.11) | PR única | `go test ./internal/assets/ ./internal/components/sdd/ ./internal/cli/... && go test ./internal/components/ -run TestGolden` | Los goldens regenerados con `-update` y revisados por diff son la prueba de proyección en los doce runtimes | Sección compartida, doce plantillas, registros, caso del hook y `testdata/golden/*.golden` |
| U5 | Cierre: pin completo de vocabulario y paridad, documentación del cambio de comportamiento y verificación global (REQ-28.11) | PR única | `go test ./internal/assets/ ./internal/kickoff/...` | `go test ./...` y `go vet ./...` (`rules.verify` de `openspec/config.yaml`) | Pin completo, documentación y, si hubo ajustes, los goldens |

Restricciones transversales (aplican a toda tarea):

- No se compila ningún binario suelto (`go build -o`, `go install`): solo `go test` acotado por paquete y, al cierre, `go test ./...` y `go vet ./...`.
- `.codex/AGENTS.md` queda sin cambios, deliberadamente (DD-06): copia heredada que el binario actual ya no genera.
- `internal/kickoff` no usa el vocabulario que `internal/kickoff/rdd_boundary_test.go` prohíbe ni en código, ni en comentarios, ni en mensajes, ni en las pruebas nuevas (DD-01); ese fichero no se edita.
- No se tocan `internal/kickoff/seal.go`, `internal/kickoff/machine.go`, `internal/cli/sdd_preflight_hook.go`, `internal/components/sdd/session_preflight.go`, los plugins de OpenCode, `internal/sddstatus/**` ni `internal/multirole/barrier.go`.
- Los goldens solo se regeneran con `go test ./internal/components/ -run TestGolden -update`, nunca a mano; se revisa el diff y se repite sin `-update`.
- Artefactos y prosa en español neutro; identificadores de código en inglés.
- Cada unidad cierra con un commit Conventional Commit con código, pruebas y documentación juntos (skill `work-unit-commits`), sin atribución de IA ni `Co-Authored-By`.

## Fase 1: Regla de sellado en Go (infraestructura) — Unidad U1

Cierra con el commit `feat(kickoff): exigir el rol fullstack al sellar continuous`.

- [ ] 1.1 Añadir la constante `SealedByInferred = "inferred"` en `internal/kickoff/types.go` (DD-03). Verificación: `go test ./internal/kickoff/...` compila.
- [ ] 1.2 Escribir las pruebas en `internal/kickoff/schema_test.go` antes del código (tabla de `ValidateForSeal`, DD-01/DD-02, REQ-28.12): `continuous`+`[fullstack]` válido; `continuous`+`[core,web]`, `+[backend]`, `+[fullstack,backend]` y `+[fullstack,fullstack]` inválidos con `errors.Is(err, ErrContinuousRequiresFullstack)` y mensaje que contiene `--execution-style checkpointed` y `--role`; `checkpointed`+`[core,web]` válido; `SealedBy: SealedByInferred` con `continuous`+`[core]` válido; `ParseKickoff` lee un YAML `continuous`+`[core,web]` (lectura tolerante, D7). Observar RED con `go test ./internal/kickoff/ -run 'ValidateForSeal|ParseKickoff'` (no compila hasta 1.3).
- [ ] 1.3 Implementar en `internal/kickoff/schema.go` `ErrContinuousRequiresFullstack`, `checkContinuousRoster(style ExecutionStyle, roles []string, fromSessionPace bool) error` y `(Kickoff) ValidateForSeal() error` (`Validate()` + regla, exento si `SealedBy == SealedByInferred`); `Validate()` no cambia. Mensaje exacto del design §5 y prefijo `--from-session-pace auto deriva continuous, que solo admite el rol único fullstack` cuando el estilo se derivó. Verificación: las pruebas de 1.2 pasan a verde.
- [ ] 1.4 Escribir en `internal/kickoff/args_test.go` los casos de `ParseSealArgs` (REQ-28.12, REQ-28.13) y adaptar `TestParseSealArgsRoleFlagPolicyParsing` a `--execution-style checkpointed`: `continuous` sin roles ⇒ `fullstack`; `--from-session-pace auto` + `--role core` rechaza sin derivar `checkpointed`; `interactive` + roles ⇒ `checkpointed`; `checkpointed` explícito sobre `auto` + roles sella; `continuous` explícito sobre `auto` + roles rechaza; rechazo previo a resolver `--change` y a escribir. Observar RED.
- [ ] 1.5 Aplicar `checkContinuousRoster` en `internal/kickoff/args.go` (`ParseSealArgs`, rama sin `--infer`, sobre el estilo efectivo y el roster ya completado con el `fullstack` por defecto). Verificación: las pruebas de 1.4 pasan a verde.
- [ ] 1.6 Usar `SealedByInferred` en `internal/kickoff/infer.go` y añadir al comentario de `InferKickoff` el párrafo que cita P2 y REQ-21.4, sin cambio de comportamiento; ampliar `internal/kickoff/infer_test.go` con el pin de P2: inferido `continuous` + roles detectados, `sealed_by: inferred`, `ValidateForSeal` sin error. Verificación: `go test ./internal/kickoff/...` verde, incluido `rdd_boundary_test.go`.
- [ ] 1.7 Escribir en `internal/cli/sdd_kickoff_test.go` los casos de integración (REQ-28.12, REQ-21.4, DD-01): `seal continuous --role core --role web` falla sin crear `kickoff.yaml`; un `seal` repetido con `continuous` y roles sobre un kickoff ya sellado así se rechaza y deja el fichero intacto; `seal --infer` con roles detectados sigue sellando `continuous`; la ayuda incluye «continuous solo con el rol fullstack». Observar RED.
- [ ] 1.8 En `internal/cli/sdd_kickoff.go`, llamar a `ValidateForSeal` justo antes de `kickoff.Seal` sobre el `Kickoff` ensamblado y añadir la línea de ayuda. Verificación: las pruebas de 1.7 pasan a verde.
- [ ] 1.9 Adaptar las pruebas existentes que sellaban `continuous` con roles: `internal/cli/sdd_kickoff_branch_test.go` (sellados de `:152`, `:238`, `:348` pasan a `checkpointed`) y `internal/cli/sdd_gate_handoff_test.go` (`continuousArgs`, `:23`, sin roles especializados). No se tocan `internal/branch/engine/snapshot_test.go`, `internal/branch/engine/status_test.go` ni `internal/cli/branch_test.go` (usan `Seal` directo, DD-01).
- [ ] 1.10 Cierre de U1: `go test ./internal/kickoff/... ./internal/cli/... ./internal/branch/...` verde; commit `feat(kickoff): exigir el rol fullstack al sellar continuous`.

## Fase 2: Verify por rol con cumplimiento técnico (implementación, parte 1) — Unidad U2

Cierra con el commit `feat(sdd-verify): añadir cumplimiento técnico y estado limpio al verify por rol`. Depende de U1 solo por orden de entrega; no hay dependencia técnica.

- [ ] 2.1 Crear `internal/assets/independent_review_boundary_test.go` con el tramo del verify (RED de la fila «Selección de repositorio git» y de REQ-28.11, DD-10): extracción de pasajes nuevos por sus límites, constantes de frases nuevas por fichero y pin de vocabulario `(?i)\b(lineage|candidates?|acknowledg\w*|burn(s|ed)?|receipts?|rdd)\b` más `axiom review`, `sdd-attempt` y `review-integration`; comprobación de que `references/technical-compliance.md` y la sección «Cumplimiento técnico»/«Estado» de `report-format.md` nombran `git -C`, `worktree`, `comparedTo`, `merge-base`, `ls-files --others`, `allowedEditRoots` (cuando aplique) y no usan `...HEAD` como ámbito. Observar RED con `go test ./internal/assets/ -run IndependentReview` (los ficheros aún no existen).
- [ ] 2.2 Crear `internal/assets/skills/sdd-verify/references/technical-compliance.md` (DD-07): resolución del diff del rol con `axiom branch status --cwd "<workspaceRoot>" --change <c> --json` (rama `kind: role` o `kind: change` para `fullstack`; `worktree` y `comparedTo` del JSON), `base = git -C "<worktree>" merge-base <comparedTo> HEAD`, `git -C "<worktree>" diff <base>` más `git -C "<worktree>" ls-files --others --exclude-standard`, exclusión de la carpeta del cambio, caída a `apply-progress` con limitación declarada; fuentes (a)–(d) de REQ-28.5; forma del hallazgo `severidad · fuente · regla · path:línea[-fin] · evidencia`; observaciones sin regla; fuentes ausentes como «no aplicable»; comprobaciones no ejecutadas; clave de Engram por rol (`sdd/{change}/verify-report` para `fullstack`, `sdd/{change}/verify-report/{rol}` para el resto); regla de `Estado` de DD-07b; diff vacío ⇒ `no limpio — diff del rol vacío` sin juicio. Sin vocabulario del otro mecanismo de revisión.
- [ ] 2.3 Modificar `internal/assets/skills/sdd-verify/references/report-format.md`: secciones «Cumplimiento técnico» y «Estado» (línea final `**Estado:** limpio` | `**Estado:** no limpio — <motivo>`), prohibición de líneas que empiecen por `verdict:` en la sección técnica (barrera de `internal/multirole/barrier.go`, solo lectura) y contexto histórico tras correcciones del juicio (REQ-28.8).
- [ ] 2.4 Modificar `internal/assets/skills/sdd-verify/SKILL.md`: descripción y activación ampliadas (petición, oferta aceptada o preferencia), una regla dura de ámbito por rol que remite a la referencia y una fila de compuerta de decisión; conservar todas las frases fijadas por `assets_test.go` y `internal/components/sdd/review_foundations_test.go`, y la descripción ≤160 caracteres (`skills_frontmatter_test.go`); respetar el presupuesto de cuerpo de `docs/skill-style-guide.md` (solo remisiones, sin duplicar el contenido de la referencia).
- [ ] 2.5 Modificar `internal/assets/skills/_shared/openspec-convention.md`: nombres `verify-report.<rol>.md` y las claves de Engram por rol de la tabla de informes del design §5 (la parte de `coverage-review` y `review-ledger` se añade en la unidad U3).
- [ ] 2.6 Verificar U2: `go test ./internal/assets/ ./internal/components/sdd/` verde, con el pin de 2.1 en verde; commit `feat(sdd-verify): añadir cumplimiento técnico y estado limpio al verify por rol`.

## Fase 3: Juicio por rol y revisión de cobertura (implementación, parte 2) — Unidad U3

Cierra con el commit `feat(judgment-day): admitir el diff de un rol SDD y la revisión de cobertura`. Depende de U2 (el juicio lee `**Estado:**` del informe vigente).

- [ ] 3.1 Ampliar `internal/assets/independent_review_boundary_test.go` con el tramo del juicio y la cobertura (RED, DD-10): extraer `## SDD Role Target` hasta el siguiente `## `; constantes de la frase de activación y la fila de decisión nuevas de `judgment-day/SKILL.md` y de las reglas nuevas de `sdd-spec` y `sdd-design`; pin de vocabulario sobre `_shared/sdd-coverage-review.md`; comprobar que `SDD Role Target` nombra `git -C`, `worktree`, `merge-base`, `ls-files --others` y `allowedEditRoots`, y que no usa `...HEAD` como ámbito; mantener intacto `judgment_day_standalone_test.go` (frase «issues no receipt»). Observar RED.
- [ ] 3.2 Modificar `internal/assets/skills/judgment-day/references/prompts-and-formats.md`: añadir `## SDD Role Target` (DD-08): activación por oferta o preferencia, solo con `**Estado:** limpio` y petición explícita prevaleciente (A-2), línea de coste, objetivo inmutable `{rol, repositorios, merge-base, SHA de HEAD, sha256}` con recálculo antes de fusionar, arreglos de `jd-fix-agent` en `allowedEditRoots` ∩ `worktree` del rol y nunca bajo `technicalContext.reference`, hallazgo fuera de raíces ⇒ pendiente humano y sin `APPROVED`, confirmación de la primera ronda por estilo (A-3), registro en `review-ledger[.<rol>].md` y `sdd/{change}/review-ledger[/{rol}]` (A-4), correcciones en `### Correcciones del juicio — <rol>` dentro del `apply-progress` único y nunca en `tasks*.md`, y verificación final con histórico.
- [ ] 3.3 Modificar `internal/assets/skills/judgment-day/SKILL.md`: una frase de activación para el objetivo SDD por rol y una fila de compuerta de decisión que remiten a la referencia; conservar «issues no receipt» y el resto de frases fijadas.
- [ ] 3.4 Crear `internal/assets/skills/_shared/sdd-coverage-review.md` (DD-09): mandato de los dos revisores ciegos con las identidades `jd-judge-a` y `jd-judge-b`, declaración de que revisan artefactos de planificación y de que no aplican el presupuesto de barridos, la puerta de precisión ni la persistencia de ledger de su definición, salida JSON `{"gaps":[…],"evidence":[…]}`, entradas tras `spec` y tras `design`, regla de unión y deduplicación por el padre, revisor sin resultado declarado, informes `coverage-review.spec.md` y `coverage-review.design.md` (Engram `sdd/{change}/coverage-review/{spec,design}`) con secciones `## Ronda N — AAAA-MM-DD`, informe sin huecos declarado y sin autocorrección. Verificar que el fichero se instala por `SharedSkillFileNames()` (comprobar la lista en `internal/assets` y, si no lo incluye por convención, añadirlo allí).
- [ ] 3.5 Completar `internal/assets/skills/_shared/openspec-convention.md` con `coverage-review.spec.md`, `coverage-review.design.md`, `review-ledger[.<rol>].md` y sus claves de Engram.
- [ ] 3.6 Añadir una regla en `internal/assets/skills/sdd-spec/SKILL.md` y otra en `internal/assets/skills/sdd-design/SKILL.md`: la tabla de cobertura propia no sustituye a la revisión de cobertura; mantener sus pruebas de frontmatter y longitud.
- [ ] 3.7 Verificar U3: `go test ./internal/assets/ ./internal/components/sdd/` verde, con los pins de 3.1 en verde y `judgment_day_standalone_test.go` intacto; commit `feat(judgment-day): admitir el diff de un rol SDD y la revisión de cobertura`.

## Fase 4: Sección compartida del orquestador y proyección (integración) — Unidad U4

Cierra con el commit `feat(sdd): proyectar la sección SDD Independent Review a los orquestadores`. Depende de U2 y U3 (la sección remite a sus referencias y activaciones).

- [ ] 4.1 Ampliar `internal/assets/independent_review_boundary_test.go` con el tramo del orquestador (RED, DD-10): extraer `SDD Independent Review` por sus marcadores `sdd-orchestrator-section`; pin de vocabulario; la sección no contiene `Axiom SDD preflight` ni `preflight 4/`; declara la frase de resultados ausentes de REQ-28.10 (una revisión no ejecutada, sin respuesta o no limpia se presenta como tal y nunca como limpia); paridad: toda plantilla de `internal/assets/{antigravity,claude,codex,cursor,gemini,generic,hermes,kimi,kiro,opencode,qwen,windsurf}/sdd-orchestrator.md` que proyecta las puertas proyecta una vez la sección nueva; constantes de las frases nuevas de las puertas, del `Dependency Graph` y de `claude/sdd-orchestrator-workflow.md`. Observar RED.
- [ ] 4.2 Añadir en `internal/assets/skills/_shared/sdd-orchestrator-sections.md` la sección `<!-- sdd-orchestrator-section:SDD Independent Review:start/end -->` (DD-04 a DD-09, DD-11, REQ-28.1, REQ-28.2, REQ-28.3, REQ-28.10, REQ-28.14): interacción propia de dos preguntas tras el pre-vuelo con `after_apply` (`verify`, `verify-judgment`, `none`) y `coverage` (`yes`, `no`), etiquetas localizables, descripciones de efecto y coste (la de `verify-judgment` dice que en `continuous` incluye las correcciones acotadas), respaldo en texto `After apply: <label>; Coverage: <label>`, sin valor por defecto, caché de sesión sin persistencia, aplicación por estilo sellado y sesión sin preferencia, los tres puntos, verify por rol como excepción explícita al enrutado por dependencias (DD-07c), juicio, cobertura sin autocorrección, resultados en paradas y resumen final en `continuous`, y runtimes sin subagentes (DD-11). Sin marcadores de pregunta del pre-vuelo.
- [ ] 4.3 En el mismo fichero, editar el texto de `SDD Change Kickoff and Block Gates` (`continuous` válido solo con el rol único `fullstack`; con roles especializados se sella `checkpointed` y el comando rechaza lo contrario) y la frase de verify del `Dependency Graph` que remite a `SDD Independent Review`.
- [ ] 4.4 Modificar `internal/assets/claude/sdd-orchestrator-workflow.md` (copia propia del grafo): misma remisión a `SDD Independent Review`.
- [ ] 4.5 Añadir en cada una de las doce plantillas `internal/assets/{antigravity,claude,codex,cursor,gemini,generic,hermes,kimi,kiro,opencode,qwen,windsurf}/sdd-orchestrator.md`, inmediatamente después del marcador `{{GENTLE_AI_SDD_SECTION:SDD Change Kickoff and Block Gates}}`, su encabezado `SDD Independent Review` (con el nivel que use el de las puertas) y el marcador `{{GENTLE_AI_SDD_SECTION:SDD Independent Review}}`.
- [ ] 4.6 Registrar la sección: añadir `SDD Independent Review` a `sharedOrchestratorSectionNames` en `internal/components/sdd/orchestrator_shared_sections_test.go` y la entrada `{"SDD Independent Review", 1}` en `internal/assets/orchestrator_drift_ratchet_test.go`.
- [ ] 4.7 Añadir en `internal/cli/sdd_preflight_hook_test.go` el caso del design §6: transcript con el pre-vuelo canónico seguido de la interacción de preferencia ⇒ `allow` con el mismo bloque de 6 líneas; el hook (`internal/cli/sdd_preflight_hook.go`, read-only) no cambia.
- [ ] 4.8 Regenerar los goldens: `go test ./internal/components/ -run TestGolden -update`; revisar el diff de `testdata/golden/` (solo aparece la sección nueva y las ediciones de 4.3 en `sdd-*-claudemd`, `sdd-codex-agentsmd*`, `sdd-gemini-geminimd`, `sdd-cursor-rules`, `sdd-kiro-instructions`, `sdd-opencode-multi-settings`, `sdd-vscode-instructions`, `sdd-windsurf-global-rules`, `sdd-antigravity-rulesmd` y `combined-*`) y repetir sin `-update` hasta verde. No se edita ningún golden a mano.
- [ ] 4.9 Verificar U4: `go test ./internal/assets/ ./internal/components/... ./internal/cli/...` y `go test ./internal/components/sdd/ -run Preflight` verdes; `go test ./internal/sddstatus/ -run ReviewOfferAbsence` y `go test ./internal/multirole/` pasan sin tocar su código; confirmar con `git status` que `.codex/AGENTS.md` no aparece modificado; commit `feat(sdd): proyectar la sección SDD Independent Review a los orquestadores`.

## Fase 5: Pins finales, documentación y verificación global — Unidad U5

Cierra con el commit `docs(sdd): documentar la revisión independiente y fijar su frontera`. Depende de U1 a U4.

- [ ] 5.1 Completar `internal/assets/independent_review_boundary_test.go` con los casos que falten tras las unidades previas (REQ-28.11): que no existan barridos de ficheros completos, que los pasajes pineados sean solo los nuevos o modificados, y que las constantes de frases coincidan con el contenido de cada fichero; observar que una frase retirada hace fallar la prueba (RED por mutación temporal en memoria, sin modificar ficheros) y que con el contenido real pasa a verde.
- [ ] 5.2 Localizar con `rg` la documentación del repositorio que describa `axiom sdd kickoff seal`, el estilo `continuous` o las revisiones tras apply (carpeta `docs/` y README relevantes) y actualizarla en el sitio: `continuous` solo con el rol único `fullstack`, el rechazo del `seal` repetido con la regla nueva (DD-01, con `axiom sdd kickoff show` para consultar), y la existencia de la interacción de preferencia, el verify por rol, el juicio y la cobertura. Si no hay documentación que las nombre, añadir una nota breve donde viva la doctrina de puertas; no se editan documentos declarados como `technical_context.reference`.
- [ ] 5.3 Verificación global (`rules.verify`): `go test ./...` y `go vet ./...`, sin compilar binarios. Comparar la implementación con los escenarios de `specs/*/spec.md` (REQ-28.1 a REQ-28.14, REQ-21.4) y anotar cualquier discrepancia en el progreso de apply.
- [ ] 5.4 Si 5.3 obliga a regenerar goldens, repetir el procedimiento de 4.8; commit `docs(sdd): documentar la revisión independiente y fijar su frontera`.

## Orden de ejecución y paralelismo

- Secuencial por dependencia: U1 → U2 → U3 → U4 → U5. Las tres unidades de prosa comparten `independent_review_boundary_test.go` y `openspec-convention.md`, de modo que no se ejecutan en paralelo.
- Paralelizable en teoría (sin ganancia por el escritor único y las ediciones compartidas): U1 es independiente de U2 y U3 a nivel de ficheros; se deja secuencial para mantener un solo escritor y commits limpios.
- Dentro de una unidad, las pruebas (RED) preceden a la implementación; 4.5 (doce plantillas) es mecánica y puede hacerse en un solo paso por el escritor.

## Trazabilidad requisito → tarea

| Requisito | Tareas |
|-----------|--------|
| REQ-28.12, REQ-28.13 | 1.2–1.5, 1.7–1.9 |
| REQ-21.4 | 1.6, 1.7 |
| REQ-28.1, REQ-28.2, REQ-28.3 | 4.1, 4.2 |
| REQ-28.4, REQ-28.5 | 2.1–2.4 |
| REQ-28.6, REQ-28.7, REQ-28.8 | 2.3, 3.1–3.3 |
| REQ-28.9 | 3.4–3.6 |
| REQ-28.10 | 4.2, 4.1 |
| REQ-28.11 | 2.1, 3.1, 4.1, 5.1 |
| REQ-28.14 | 4.2, 4.7 |

## Riesgos y notas para apply

- Las pruebas de cuatro de los paquetes tocados fijan frases de `sdd-verify` (`assets_test.go`, `review_foundations_test.go`) y la longitud de la descripción; cualquier reescritura de `SKILL.md` debe conservarlas.
- El cambio de comportamiento del `seal` repetido (1.7) es deliberado y debe quedar en la documentación (5.2).
- Los goldens generan un diff grande que no cuenta en el presupuesto autorado pero sí en la PR (`size:exception`).

## Registro de remediación

(vacío)
