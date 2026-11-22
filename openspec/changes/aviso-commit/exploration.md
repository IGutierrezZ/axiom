## Exploración: aviso-commit (C-02-d) — aviso por commit, parada opcional por tarea, validación al pedir la PR

### Estado actual

- **ODD no tiene paradas, y una spec viva lo exige.** `internal/components/agentguidance/routing.go:64-67` (comentario) y REQ-27.17 de `openspec/specs/organic-agent-trigger-rules/spec.md:427` prohíben en la doctrina ODD generada «Plan stop», «Stops and exits» y cualquier parada; el paso 0 (`routing.go:68`) añade «do not ask any further governance question». `routing_test.go:810` prohíbe literalmente `three exits`, `Plan stop`, `plan stop`. La sección 02 cita `routing.go:66-67` y sigue vigente en `main`/worktree. **No existe «parada de plan» en ODD**: la primera redacción de P2 la citaba; el usuario la corrigió el 2026-11-20 (ver ODD) y no se crea.
- Paso 6 de ODD (`routing.go:76`): «Every task closes with at least one work-unit commit … Conventional Commit … push, PR creation and merge remain the user's decisions». Paso 7 (`routing.go:77`): cierre. La frase de RDD del 77 ya la retiró C-01-c. Los tests de esos pasos: `routing_test.go:186-190`, `:285`, `:354-382`, `:753-810`, `:928` (REQ-32.4: single-pr).
- **SDD**: la puerta `tasks` es una parada de tres salidas solo en `checkpointed` (REQ-21.10, REQ-26.1/26.2). La pregunta tipada (`gateQuestion`, `internal/sddstatus/governance_handoff.go:138-156`) valida **exactamente tres opciones** con tokens `rejected`/salida 2/`approved`: no admite una cuarta dimensión sin romper REQ-26.2. En `continuous` no hay paradas (`kickoff.yaml` de este cambio: `continuous`, `fullstack`) → modo 1 sin preguntar.
- **Fuente única de la doctrina SDD (no son copias)**: `internal/assets/skills/_shared/sdd-orchestrator-sections.md` con secciones delimitadas `<!-- sdd-orchestrator-section:NOMBRE:start/end -->` (p. ej. «SDD Change Kickoff and Block Gates» `:74-92`, «Delivery Strategy» `:261-267`). Cada `internal/assets/<agente>/sdd-orchestrator.md` (11 ficheros reales + `claude/sdd-orchestrator-workflow.md`; `claude/sdd-orchestrator.md` es bootstrap) la referencia con `{{GENTLE_AI_SDD_SECTION:Delivery Strategy}}` (p. ej. `generic/sdd-orchestrator.md:92`, `cursor/…:166`, `claude/sdd-orchestrator-workflow.md:100`). `orchestrator_shared_sections_test.go:103-135` (`sharedOrchestratorSectionNames`) registra cada sección y `TestSharedOrchestratorSectionsHaveOneSource` impide cuerpos inline. Hay precedente de preferencia solo de sesión: «Preference» de `SDD Independent Review` y la estrategia de entrega ("never write it to `kickoff.yaml`, `gates.yaml`, or `tasks.md`", `sections.md:264`).
- **Kickoff inmutable**: `internal/kickoff/seal.go:51-60` (`Seal` publica una sola vez, sin sobrescribir; D-01/D-02), REQ-25.36 y REQ-27.4 (`docs_publication` es el precedente de campo sellado, `omitempty`, `types.go:128`, `schema.go:31,99-107`, `args.go:229-239,260,278,384`, proyectado en `sddstatus/status_v2.go:136,188`). `sdd_kickoff.go:36` es solo el texto de uso.
- **Gate record**: `kickoff.GateRecord` (`types.go:183-207`) ya lleva datos adicionales `omitempty` (`handed_off`, `handed_off_to`, evidencia) escritos por banderas de `gate record` (`args.go:450,505-535`, `cli/sdd_gate.go:119-131`). Su digest (`ArtifactDigest`) es del **artefacto** (`sdd_gate_digest.go:40-57`), no del registro: añadir un campo al registro **no** altera ningún digest ni reabre puertas. La proyección de lo aprobado es `machine.go:245-253` (primer registro `approved`) → `GateState` (`types.go:237`) → `gateV2` en `status_v2.go:139-175`.
- **sdd-apply** (`internal/assets/skills/sdd-apply/SKILL.md:82-91`): Step 2a dice «commit every work unit … one work-unit commit per task»; el cierre (`:249-257`) «offer the user a formal commit». El sub-agente trabaja por lote y no puede avisar al usuario a mitad: el aviso por tarea y la parada solo pueden darlos el orquestador (tras volver el lote) o un apply lanzado tarea a tarea. **Implicación**: los modos 2 y 3 exigen que apply devuelva el control tras cada tarea (parámetro `commit_review` en el lanzamiento, como `delivery_strategy`).
- **work-unit-commits**: dos copias, `internal/assets/skills/work-unit-commits/SKILL.md` y `skills/work-unit-commits/SKILL.md`; hoy en paridad respecto a single-pr (`internal/assets/skill_copies_parity_test.go` lo vigila) → se editan juntas.

### Pregunta de diseño central: dónde se persiste `commit_review` en SDD

| Opción | Compat. digest | Compat. REQ vivos | C-06-f | Contradice P2/README |
|---|---|---|---|---|
| **(a) `gate record --gate tasks --decision approved --commit-review <modo>` → campo `commit_review` `omitempty` en el `GateRecord` de `gates.yaml`** | Sí: el digest es del artefacto, no del registro; sin reabrir puertas. | Sí: mismo patrón que `HandedOff` (REQ-26.3 ya exige compatibilidad hacia atrás con `omitempty`); no toca REQ-25.36/27.4. `gates.yaml` es append-only (REQ-21.12): el modo queda como evidencia. Hay que añadir requisito nuevo en `sdd-block-review-gates` y proyectarlo en `status` (`gateV2`). | La UI llama al mismo verbo con la misma bandera: contrato estable. | No contradice P2 (se pregunta en la parada de `tasks`). Contradice la **redacción** de README apdo. 7 (`:365`) y 06:246 («bandera del kickoff»), no su intención. |
| (a') Misma bandera, pero reescribiendo `kickoff.yaml` | Neutro | **Rompe** inmutabilidad (`seal.go`, D-01/D-02) y REQ-25.36 | Mismo | Cumple la letra del README, contradice P2 en espíritu y la spec |
| (b) Subcomando que actualiza el kickoff tras el sellado | Neutro | Igual que (a'): rompe sellado único; verbo nuevo | Verbo nuevo | — |
| (c) Línea en `tasks.md` | **Cambia el digest de `tasks`** (lo hace pasar de `approved` a `pending` o exige registrar tras cambiarlo; orden circular: la pregunta se hace en la propia puerta) | `tasks.md` lo escribe `sdd-tasks`, el modo lo decide la persona | Frágil (parsear markdown) | Contradice la regla «nunca escribir en tasks.md» (`sections.md:264`) |
| (d) Solo doctrina/sesión | Sí | Sí (precedente: Preference, estrategia de entrega) | **No**: la UI no tiene dónde dejarlo; se pierde al reanudar y en la salida 2 (`--hand-off`), donde otra persona recoge `apply` | No contradice P2, pero pierde la decisión en el traspaso |

**Hallazgo de contradicción real**: README apdo. 7 y 06:246 dicen «bandera del kickoff en SDD»; P2 (apdo. 6.2) dice que se pregunta en la puerta `tasks`, y el kickoff es de sellado único e inmutable. Ambas no pueden cumplirse a la letra. También la viñeta superada de la sección 02 (`:199`, «bandera nueva en `axiom sdd kickoff seal`») ya está superada por P2.

### Recomendación (punto 4)

**Opción (a)** con estas precisiones, no decididas por mí:
1. La pregunta es una **interacción propia** de dos tomas (como «Preference» de `SDD Independent Review`), nunca una cuarta/novena opción de la pregunta tipada de tres salidas (REQ-26.2).
2. Se registra solo con `--decision approved` en `--gate tasks` (rechazo: la pregunta no aplica); vocabulario cerrado de tres tokens (`notify`, `stop-commit`, `stop-nocommit`, o equivalentes fijados en design) + ausencia = `notify`. En `continuous` nunca se pregunta ni se escribe.
3. El orquestador lo lee de `status` (campo nuevo del gate) y lo pasa a `sdd-apply` junto a `delivery_strategy`; con salida 2 sobrevive al traspaso.
4. Corregir la redacción del README apdo. 7 y 06:246 («campo del registro de la puerta `tasks` en SDD»).
- Esfuerzo: Medio. Riesgo bajo por precedente exacto (`HandedOff`).

### ODD (resuelto por el usuario, 2026-11-20)

La primera redacción de P2 pedía preguntar el modo en una «parada de plan en ODD» y el modo 3 en el paso 6, lo que contradecía REQ-27.17. El usuario lo corrigió: **ODD va siempre en modo 1**, sin pregunta ni paradas, tampoco a petición en el chat; quien quiera parar tras cada tarea usa SDD `checkpointed`. Consecuencias: REQ-27.17, el pin de `routing_test.go:810`, el paso 0 y `sdd-block-review-gates/spec.md:237` no cambian; el documento `odd/tasks/<feature>.md` no guarda ningún modo; el modo 3 solo es comportamiento nuevo en `sdd-apply`. Lo único que puede cambiar en ODD es el aviso del modo 1 (sha, tarea y `git show <sha>` al cerrar cada tarea) en el paso 6, sin parar.

### Áreas afectadas

- `internal/components/agentguidance/routing.go:76-77` (+ `routing_test.go`) — pasos 6 y 7; comentario `:64-67`.
- `internal/assets/skills/_shared/sdd-orchestrator-sections.md` — nueva sección «Commit Review» (aviso con sha + tarea + `git show <sha>`; modos; pedir la PR valida todo; solo se registran commits con cambios pedidos; `fixup!`+autosquash si no publicada, commit correctivo si publicada; recomendar merge sin squash, P1) y ajustes en «SDD Change Kickoff and Block Gates».
- 11 `internal/assets/<agente>/sdd-orchestrator.md` + `claude/sdd-orchestrator-workflow.md` — solo el marcador `{{GENTLE_AI_SDD_SECTION:Commit Review}}` (+ registrar el nombre en `orchestrator_shared_sections_test.go:103`).
- `internal/assets/skills/sdd-apply/SKILL.md:82-91,249-257` — parámetro `commit_review`, avance tarea a tarea y modo 3 (sin commit hasta la aprobación).
- `internal/assets/skills/work-unit-commits/SKILL.md` y `skills/work-unit-commits/SKILL.md` (paridad vigilada).
- Go (opción a): `internal/kickoff/types.go`, `schema.go`, `args.go` (bandera de `gate record`), `machine.go:245-253`, `internal/cli/sdd_gate.go:48,119-131` (ayuda, registro, JSON), `internal/sddstatus/status_v2.go:139-175` y `governance*.go`; tests asociados (`args_test.go`, `schema_test.go`, `sdd_gate_test.go`, `status_v2_test.go`).
- **No tocar**: `internal/kickoff/args.go` `SealArgs` ni `internal/cli/sdd_kickoff.go` (no hay bandera de `seal`). `sdd_gate_digest.go` no cambia (verificado).
- No adelantar C-02-b ni C-03-a (comparten `routing.go`, sections, orquestadores, `args.go`).

### Specs vivas afectadas

- `sdd-block-review-gates`: REQ-21.10 (puerta `tasks`: segunda interacción), REQ-26.2 (la pregunta tipada se queda en tres opciones), REQ-26.3 (nuevo campo opcional compatible hacia atrás), REQ-26.8 (texto único de salidas, exclusivo de SDD), REQ-21.7 (en `continuous` no se pregunta), línea 237 («ODD sin paradas»). Requisito nuevo (siguiente número libre; el máximo actual es REQ-32.x).
- `sdd-lane-kickoff`: REQ-25.36 y REQ-27.4 solo como referencia de inmutabilidad; la opción (a) **no** modifica esta spec (la sección 02 y el README la listan como afectada: queda desactualizado salvo que se opte por (a')/(b)).
- `sdd-orchestrator-assets`: REQ-32.3 (entrega single-pr; hermano natural) y paridad entre runtimes; requisito nuevo para la sección «Commit Review» y su paridad.
- `organic-agent-trigger-rules`: REQ-27.17 sin cambios (ODD sigue sin paradas, ver ODD); REQ-32.4 (entrega ODD) solo si el aviso del modo 1 entra en el paso 6; REQ-29.18 sin cambios.
- `sdd-docs-publication` (REQ-27.8/27.12): coherencia con «pedir la PR» y publicación; no debería cambiar.

### Goldens y tests

- Regeneración: `go test ./internal/components/ -run <Test> -update` (`internal/components/golden_test.go:995-1012`, bandera `-update`); sin compilar binarios.
- Candidatos (los que ya contienen la sección de entrega, 43 apariciones en 13 ficheros): `testdata/golden/sdd-{antigravity-rulesmd,codex-agentsmd,codex-agentsmd-lowcost,codex-agentsmd-powerful,cursor-rules,gemini-geminimd,kiro-instructions,vscode-instructions,windsurf-global-rules}.golden`, `combined-windsurf-global-rules.golden`, `sdd-opencode-{cmd-sdd-apply,cmd-sdd-init,multi-settings}.golden`; además `sdd-claude-agent-sdd-apply.golden`, `sdd-kiro-agent-sdd-apply.golden` y `combined-claude-claudemd.golden` según lo que renderice el cambio en `routing.go`.
- Tests a tocar: `routing_test.go` (paso 6, solo el aviso del modo 1; el pin de REQ-27.17 se conserva), `orchestrator_shared_sections_test.go`, `orchestrator_no_drift_test.go`, `assets/skill_copies_parity_test.go`, `assets_test.go`, tests Go del registro/status. Comprobar `internal/assets/blocking_prompt_contract_test.go` (contrato de prompts bloqueantes) por la segunda interacción.

### Estimación

- Ficheros: ~14 Go (producción + tests) + ~17 markdown de doctrina/skills + ~14 goldens + 4 specs delta ≈ 45-50.
- Líneas autoradas (sin goldens): ~250-350 Go (con tests) + ~120-180 doctrina/skills + ~200 delta de specs ≈ 600-750. Tamaño real M-alto; la entrega es una sola PR (`single-pr`).

### Riesgos

- README apdo. 7/06:246 («bandera del kickoff») choca con P2 y con el kickoff de sellado único: corregir la redacción o el cambio quedará incoherente con el backlog.
- `sdd-apply` trabaja por lote: sin cambio de contrato (devolver tras cada tarea) los modos 2 y 3 no son implementables.
- `fixup!`: el autosquash cambia los sha posteriores; la doctrina debe diferirlo hasta antes de la PR, terminar el commit de la tarea en curso (o `stash`) antes de intercalar `git commit --fixup`, usar `GIT_SEQUENCE_EDITOR=true` (sin editor interactivo) y decidir «publicada» con la rama remota que contiene el sha.
- Posible pregunta abierta: en `continuous` ¿puede la persona pedir paradas en el chat? La sección superada lo permitía; P2 no lo prohíbe. En ODD está resuelto: no.
- Solapamiento duro con C-02-b/C-03-a en `routing.go`, sections, orquestadores: rebasar y no adelantar.

### Preparado para propuesta

Sí, con una decisión a confirmar por la persona antes de `propose`: la opción (a) para SDD. ODD frente a REQ-27.17 quedó resuelto el 2026-11-20: ODD siempre en modo 1, sin cambiar REQ-27.17.
