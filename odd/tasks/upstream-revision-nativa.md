# ODD: Revisión nativa — absorción upstream (corrección, recuperación, plataforma, riesgo y esquemas versionados)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/upstream-revision-nativa.md`.
> Espejo de recuperación en Engram: topic `odd/upstream-revision-nativa/tasks`, proyecto `axiom`.
> Rama: `inc/upstream-revision-nativa` (worktree `C:\repos\axiom-wt\upstream-revision-nativa\axiom`).

## Estado

**Alcance validado** por el usuario el 2026-11-04, con D-RN1 aplazada, D-RN2 como deuda aceptada y D-RN3 descartada. **Implementación en curso** desde el 2026-11-05, autorizada por el usuario («me gustaria que arranquen las que pueden empezar», relayed por la sesión coordinadora) para T1–T17. T18–T20 siguen bloqueadas por D2. Commit por tarea en esta rama; push y PR los decide el usuario. La estrategia de cadena de PR (`stacked-to-main` o `feature-branch-chain`) se pregunta antes del primer PR: los commits por tarea son lineales y admiten cualquiera de las dos. RDD apagado (`axiom review mode status`: off por defecto); cada tarea se verifica con `axiom review assess` según su nivel.

## Objetivo

Absorber de upstream gentle-ai (revisado hasta `origin/main` 3ea08cdf, posterior a v4.3.1) los arreglos del motor de revisión nativa (`internal/reviewtransaction/`, `internal/cli/review_*`, `internal/reviewerprovider/`). La absorción es selectiva y conserva las divergencias de Axiom:

- RDD opcional y apagado por defecto.
- SDD y OpenSpec vivos.
- Sin telemetría.
- Marca propia (`axiom`, `axiom.*`) y módulo `/v3`.
- **D6:** un esquema ya publicado en `contracts/` no se modifica nunca (`docs/upstream-absorption-ledger.md:123`, fila `71a47477`). Los desfases que upstream corrigió editando esquemas se resuelven publicando **versiones nuevas** de esquema.

## Problema

- **Fallo real hoy.** `internal/reviewtransaction/risk.go:74` emite `dangerous_sink` (entró con `sync-upstream-phase2-rdd`), pero `validateReviewStartRiskReasons` (`internal/cli/review_start_contract.go:354-399`) no tiene ese caso y cae en el `default` de la línea 395: un START sobre un candidato con ese motivo falla en su propio `Validate()`. Los esquemas `contracts/review-integration/v2/schemas/start-v4.schema.json:115-120`, `v2/schemas/start.schema.json:107-111` y `v1/schemas/start-v2.schema.json:196-211` tampoco admiten el motivo.
- **Desfases de contrato.** Axiom emite `targeted_validator_evidence` en el cierre del último evento (`internal/cli/review_last_event_closure.go:30,181-199`) sin que `last-event-closure.schema.json` lo declare, y emite `gentle-ai.review-acknowledged/v1` (`review_last_event_closure.go:54,95-98`) sin esquema publicado.
- **Defectos del ciclo de corrección y recuperación** que upstream ya corrigió: hallazgos `inconclusive` rechazados en el plan de corrección, eco de `request_hash` ausente en el refuter, recover y resume en rangos base-diff/committed, base del rango committed, recuperación desde STATUS y linaje escalado no terminal.
- **Defectos de plataforma** (Windows, SMB/UNC, `\\?\`, montajes sin modos POSIX) y una serie de arreglos pequeños del motor.

## Por qué

- El fallo de `dangerous_sink` bloquea la revisión de cualquier candidato con un sumidero peligroso: es la prioridad.
- Los demás defectos están verificados en el código de Axiom (`ruta:línea` en las tablas) y tienen arreglo probado en upstream.

## Configuración resuelta

- **TDD:** desactivado (`openspec/config.yaml:16`, `strict_tdd: false`). Se ejecutan las comprobaciones funcionales ordinarias, no ninguna.
- **Ejecutor de pruebas:** `go test` en proceso, por paquete. Nunca binarios sueltos (el antivirus borra `axiom.exe`); los tests de CLI compilados de upstream se pasan a ejecución en proceso.
- **Estrategia de entrega:** `ask-on-risk` (por defecto). La previsión supera con mucho las 400 líneas, así que antes del primer commit de implementación se preguntará una vez la estrategia de cadena (`stacked-to-main` o `feature-branch-chain`).
- **Ruta por tarea:** cada rebanada que toque 2 o más ficheros no triviales va por un único writer delegado; la ruta y su disparador se anotan en la tarea al ejecutarla.

## Veredictos verificados

Detalle completo por candidato en las notas de la sesión de formalización (lotes A, B y C). Las líneas son estimaciones de Go y tests para Axiom.

### Corrección, refuter y recuperación

| PR upstream | Commit | Ruta:línea en Axiom | Estado | Veredicto | Líneas |
|---|---|---|---|---|---|
| #5896 plan de corrección acepta `inconclusive` | `0806325b` | `internal/reviewtransaction/correction_plan_request.go:71` (incoherente con `compact.go:1208`) | Defecto | ADOPTAR | ~160 |
| #5865 refuter repite `request_hash` | `1f9e4ce0` | `internal/reviewerprovider/contract.go:134` | Defecto (el prompt no pide el eco) | ADOPTAR | ~15 |
| #5870 recover de predecesor base-diff | `133254d8` | `internal/cli/review_facade.go:1661`, `:1692-1700` | Defecto | ADOPTAR | ~80 |
| #5881 resume de corrección en rango committed | `fbbaf409` | `internal/cli/review_facade.go:958-968` | Defecto | ADOPTAR (tras #5870) | ~220 |
| #5635 base committed desde el upstream de la rama | `9b66648b` | `internal/reviewtransaction/target_status.go:241-299` | Defecto | ADOPTAR | ~210 |
| #4963 linaje escalado terminal | `c2ed12a5` | `internal/reviewtransaction/target_status.go:399-402`, `:445-470`; `internal/cli/review_facade.go:1096-1111` | Defecto | ADOPTAR | ~195 |
| #5146 recuperación desde STATUS (recover) | `f15c855d` | `internal/cli/review_facade.go` ~1730-1745 | Falta la función | ADAPTAR (mensajes `axiom`) | ~240 |
| #5151 recuperación desde STATUS (status) | `b31cc497` | `internal/cli/review_next_transition.go:1156-1170`, `:1228-1234`; `internal/cli/review_status_contract.go:275` | A medias: RECOVER ya autoderiva (`review_facade.go:1604`, `:1747`) | ADAPTAR: fixture **nuevo** de recover con 4 `required_inputs`; los fixtures v1 existentes están fijados por hash (`internal/cli/review_provider_artifact_contract_test.go:30`) y no se editan | ~340 |
| #5872 selector en untracked | `bee71745` | `internal/cli/review_intended_untracked.go:146-147`; `internal/cli/review_status_contract.go:1503-1509` | Defecto | **APLAZADO (D-RN1)** | ~135 + esquema |

### Plataforma y arreglos pequeños

| PR upstream | Commit | Ruta:línea en Axiom | Estado | Veredicto | Líneas |
|---|---|---|---|---|---|
| #5765 citas `host:port` | `332aaeb8` | `internal/reviewtransaction/artifact_admission.go:987` | Defecto (no es de remotos Git: afecta a las citas de revisores; upstream solo cubre IPv4) | ADOPTAR, con test propio que cubra `localhost:8000` y `dominio:443` | ~40 |
| #5096 worktrees muertos | `f1a9b1c2` | `internal/reviewtransaction/repository_locator.go:455` | Defecto | ADOPTAR | ~55 |
| #5145 códigos rctx2 | `0de9d0b4` | `repository_locator.go:126`, `:401`; `internal/cli/review_incident.go:155` | Falta la función | ADOPTAR (los `code` no están enumerados en esquema) | ~250 |
| #4835 causas tipadas en la captura de OpenCode | `a120e5c2` | `internal/cli/review_opencode_transport.go:435`, `:443`; `review_provider_roles.go:624` | Defecto | ADOPTAR; acortar el texto fijo: con nombres de Axiom mide 261 caracteres y `failure.message` admite 240 | ~110 |
| #5408 lockfiles | `70d06568` | `frozen_candidate_context.go:431`; `internal/cli/review_lens_context.go:700`; `review_provider_roles.go:359`; `reviewerprovider/contract.go:72` | Defecto | ADOPTAR | ~165 |
| #4772 validar correcciones contra HEAD | `2472ce05` | `internal/reviewtransaction/targeted_validation_request.go:82`; `internal/cli/review_provider_roles.go:764` | Defecto | ADOPTAR; verificar antes que `append([]string{}, ...)` no altera el hash de peticiones de validación ya persistidas | ~145 |
| 37b3145b + 0bc4fdbc validador rechazado terminal | `37b3145b` | `internal/reviewtransaction/authority_disposition_plan.go:73`, `:392`; `internal/cli/review_repair.go:347` | Falta la función | ADAPTAR: reescribir sin los selectores históricos de #2995 (#4899, 1.928 líneas), que Axiom no tiene; test de CLI en proceso | 320-360 |
| #5322 reconocer delta pasivo tras el acknowledgement | `3317f4cc` | `internal/cli/review_facade.go:1095`; `review_next_transition.go:223`; `review_status_contract.go:824` | Falta la función | ADAPTAR: sin la parte de `review_assess.go` (Axiom no tiene `review_due`) | ~320 |
| #5654 continuación del delta pasivo | `3604b5eb` | `internal/cli/review_facade.go:1526` (devuelve `bool`) | Falta la función | ADAPTAR (tras #5322) | ~330 |
| #5138 montajes sin modos POSIX | `717087b1` | `internal/reviewtransaction/rar_path_safety_unix.go:104`; `internal/cli/review_mode.go:274` | Defecto | ADOPTAR (`PrivateModeIneffectiveError`) | 330-460 |
| #4670 RDD en SMB/UNC | `7550c26f` | `internal/reviewtransaction/rdd_mode.go:301`, `:338`, `:800`, `:872`; `rar_path_safety_windows.go:803`; `internal/cli/review_stop_hook.go:248` | Defecto | ADAPTAR: el atajo upstream (`globalMode == off`) no cubre el estado por defecto de Axiom; la regla es «si el global no es `on`, se ignora el fallo del override» | 550-600 |
| #5533 rutas `\\?\` | `89f15238` | `rar_path_safety_windows.go:803`; `internal/cli/review_mode.go:97` | Defecto | ADAPTAR: el discriminador `reviewModeStatusResolved` no sirve porque en Axiom el estado por defecto resuelto es idéntico al de fallo cerrado; hace falta un indicador explícito | ~200 |

**Efecto en documentación:** #5322, #5654 y #4772 tocan `review-ledger-contract.md` y `docs/review-integration.md`; hay que regenerar los golden de `testdata/golden/` y mover los techos de `internal/components/sdd/review_ledger_contract_test.go` en la misma rebanada.

### Riesgo, assess y esquemas

| PR upstream | Commit | Ruta:línea en Axiom | Estado | Veredicto | Líneas |
|---|---|---|---|---|---|
| #5345 (parte) `dangerous_sink` en START | `c13d4b75` | `internal/cli/review_start_contract.go:354-399` | **Fallo real** | ADOPTAR con `start-v5` | ver T1 |
| #5345 (parte) clasificador más estrecho y reglas de severidad compartidas | `c13d4b75` | `internal/reviewtransaction/risk.go:264` | No aplicado | ADOPTAR | ~520 |
| #5345 (parte) request-context, escalación de agente y lentes | `c13d4b75` | — | — | DESCARTAR en este bloque (~2.800 líneas, otro alcance) | — |
| #5345 consentimiento | — | `internal/cli/review_consent_contract.go:161-180` | `risk_evidence` son strings libres | Sin esquema nuevo | — |
| #4272 cierre con evidencia del validador | `e433f191` | `internal/cli/review_last_event_closure.go:30,181-199`; esquema `last-event-closure.schema.json:6` | Desfase | ADOPTAR con `last-event-closure-v2` | ~305 |
| #5395 esquema `review-acknowledged` | `9edd84b0` | `internal/cli/review_last_event_closure.go:54,95-98` | Falta el esquema | ADOPTAR (fichero nuevo) | ~45 |
| #4822 `review assess` falla cerrado | `27d8cf5f` | `internal/cli/review_assess.go:110-205` (hoy no escribe nada en stdout al fallar) | Falta el sobre | ADOPTAR solo la mitad de assess; `assess.schema.json` ya valida el sobre, compatible con D6. La mitad del esquema de fallo v2 es código muerto en upstream (`contract == V2` nunca se cumple en esas ramas): no se porta | ~120 |
| #5351 verificar por nivel de riesgo | `4c69c0ef` | `internal/assets/skills/_shared/sdd-orchestrator-sections.md:46-69` (RDD on en :53, off en :54) | Prosa | ADAPTAR: se adopta verificar por nivel con RDD encendido o apagado; **no** se adopta «medium nunca dispara revisión»: se conserva la revisión diferida del trozo de PR (`internal/components/agentguidance/routing.go:123`) | ~30 + golden |
| #5522/#5250 «Risk line» y verificador solo por ítem de riesgo nombrado (remitido por la sesión de la guía del orquestador) | — | — | — | DESCARTAR: contradice verificar por nivel de riesgo (decisión del usuario sobre #5351) | — |
| #5677 `rctx2` en START v1 | `b6778c22` | esquemas `v1/start-v2.schema.json:116`, `v1/start.schema.json:108`, `v1/status.schema.json:107`, `v1/transition-execution.schema.json:31`; emisión `review_facade.go:2298-2307` | Desfase solo en v1 (v2 ya acepta `rctx[12]`) | **DEUDA ACEPTADA (D-RN2)** | ~850 si se corrige |
| #5459 timings de fase | `2caac62d` | flag nuevo de STATUS en `review_facade.go` | No existe | **DESCARTADO (D-RN3)** | ~490 |
| 874719ae `review_due` en assess | — | — | — | DESCARTADO (D6, ya en el ledger) | — |

**Clasificación de las 32 diferencias de esquema con upstream:** 17 son solo marca o ids; 15 son semánticas (#5677: 4 de v1; #5345/#5348: 5 de capacidades y START; #4272: 1; 874719ae: 1; #5300 `rctx3` de OpenCode, que no aplica: 3; el resto, divergencia propia de Axiom). Upstream tiene además `review-acknowledged.schema.json`, que Axiom no tiene. `contracts/telemetry/**` se ignora.

## Esquemas nuevos versionados

Ningún esquema existente se modifica. No hay `go:embed`: los esquemas los cargan los tests (`compileWholePublishedReviewSchema`, `internal/cli/review_published_schema_conformance_test.go:15-51`). La numeración de versiones es global entre `v1/` y `v2/`.

| Fichero nuevo | Deriva de | Cambia | Motivo | Go y anuncio |
|---|---|---|---|---|
| `contracts/review-integration/v2/schemas/start-v5.schema.json` | `start-v4` | Añade `dangerous_sink` como `code` y `signal` de riesgo | #5345, fallo real | Constante `ReviewIntegrationStartSchemaV5`; alias `ReviewIntegrationStartSchema` (`review_start_contract.go:20`); comparaciones con V4 en `:192` y `:259`; caso nuevo en `validateReviewStartRiskReasons`; el contrato v1 no emite el motivo contra su esquema viejo |
| `contracts/review-integration/v2/schemas/capabilities-v2.7.schema.json` | `capabilities-v2.6` | `start/v4` → `start/v5` en la lista cerrada; protocolo 2.7 | Anunciar `start-v5` | `internal/cli/review_capabilities.go:46-47`, `:324-346`; tests `review_capabilities_test.go:119-171`, `dual_contract_integration_test.go:257`, `review_dialect_test.go:362` |
| `contracts/review-integration/v2/schemas/last-event-closure-v2.schema.json` | `last-event-closure` | Declara `targeted_validator_evidence` y sus definiciones | #4272 | Emitir v2 solo cuando haya evidencia del validador; el resto de cierres sigue en v1 |
| `contracts/review-integration/v2/schemas/review-acknowledged.schema.json` | upstream | Publica el sobre ya emitido (id con el dialecto que Axiom emite hoy) | #5395 | Test de conformidad y alta en la lista de `review_provider_artifact_contract_test.go:508-539` |
| `contracts/review-integration/v2/schemas/status-v10.schema.json` | `status-v9` | `intended_untracked_submission` con 5 o 6 tokens y `substitution_location` en {4, 5} | #5872 | **No se publica: D-RN1 aplazada**; constante en `internal/reviewtransaction/contract.go` y negociación en capacidades |

Descartado: `assess-v2.schema.json` (`review_due`, descartado por D6).

## Decisiones (validadas el 2026-11-04)

- **D-RN1 · #5872 selector en untracked: aplazada** fuera de este bloque. Cambia `collect.inputs[0].submission` en targets base-diff y overlay: `argument_tokens` pasa de 5 a 6 y `substitution_location` de 4 a 5. Rompe `status-v6.schema.json:65-83`, heredado por v7, v8 y v9. Upstream no actualizó su propio esquema y el PR depende de un decodificador del host (gentle-shell#2657) que Axiom no tiene. Por tanto, `status-v10` no se publica en este bloque.
- **D-RN2 · `rctx2` en el contrato v1 (#5677): deuda aceptada.** Los esquemas v1 (`v1/start-v2.schema.json:116`, `v1/start.schema.json:108`, `v1/status.schema.json:107`, `v1/transition-execution.schema.json:31`) siguen exigiendo `rctx1_` mientras el runtime emite `rctx2_` (`review_facade.go:2298-2307`); v2 ya acepta `rctx[12]`. Se documenta aquí y no en el ledger, que no se edita. Se reabre si aparece un consumidor real del contrato v1; corregirlo exigiría tres esquemas v1 nuevos (START, STATUS, `transition-execution`).
- **D-RN3 · #5459 timings de fase: descartada** en este bloque. No toca esquemas (stderr y campo `cause`); su valor era diagnosticar retrasos del antivirus.

## Alcance propuesto (tareas)

Cada tarea es una entrega de unas 400 líneas como máximo (heurística orientativa) con sus tests y documentación, y cierra con al menos un commit convencional.

### Bloque 1 · Contratos y riesgo (prioritario)

- [x] **T1 · `dangerous_sink` en START con `start-v5`** (~350). Esquema `start-v5`, constante V5, caso en `validateReviewStartRiskReasons`, selección por contrato y test de conformidad. Aceptación: un START con motivo `dangerous_sink` valida contra `start-v5` y no se emite contra esquemas viejos. Ruta: delegada (writer; 2+ ficheros no triviales).
- [ ] **T2 · `capabilities-v2.7`** (~300). Esquema nuevo, protocolo 2.7, anuncio de `start/v5` y tests de capacidades. Depende de T1.
- [ ] **T3 · Clasificador de riesgo más estrecho (#5345, parte)** (~260). `risk.go:264` y tests.
- [ ] **T4 · Reglas de severidad compartidas (#5345, parte)** (~260). Depende de T3.
- [ ] **T5 · Cierre y acknowledgement versionados (#4272, #5395)** (~350). `last-event-closure-v2`, `review-acknowledged` y tests de conformidad.
- [ ] **T6 · `review assess` falla cerrado y verificación por nivel (#4822, #5351 parcial)** (~150 + golden). Aceptación: assess escribe un sobre válido contra `assess.schema.json` al fallar; la prosa conserva la revisión diferida de medium.

### Bloque 2 · Corrección y recuperación

- [ ] **T7 · Refuter y plan de corrección (#5865, #5896)** (~175).
- [ ] **T8 · Recover y resume en rangos base-diff/committed (#5870, #5881)** (~300).
- [ ] **T9 · Linaje escalado terminal y base committed (#4963, #5635)** (~405).
- [ ] **T10 · Recuperación desde STATUS: recover (#5146)** (~240).
- [ ] **T11 · Recuperación desde STATUS: status y fixture nuevo (#5151)** (~340). Depende de T10.

### Bloque 3 · Arreglos pequeños del motor

- [ ] **T12 · Citas `host:port`, worktrees muertos y causas de OpenCode (#5765, #5096, #4835)** (~205).
- [ ] **T13 · Códigos rctx2 (#5145)** (~250). Tras T12 (#5096).
- [ ] **T14 · Lockfiles y validación contra HEAD (#5408, #4772)** (~310).
- [ ] **T15 · Validador rechazado terminal sin #2995 (37b3145b, 0bc4fdbc)** (~340).
- [ ] **T16 · Delta pasivo tras el acknowledgement (#5322)** (~320 + golden).
- [ ] **T17 · Continuación del delta pasivo (#5654)** (~330). Depende de T16.

### Bloque 4 · Plataforma

- [ ] **T18 · Montajes sin modos POSIX (#5138)** (~400).
- [ ] **T19 · RDD en SMB/UNC (#4670)** (~575). Supera la heurística porque el arreglo toca a la vez la resolución del modo, la seguridad de rutas de Windows y el hook de parada; partirlo dejaría un estado intermedio incoherente. Depende de T18.
- [ ] **T20 · Rutas `\\?\` (#5533)** (~200). Depende de T19.

**Dependencia con la hoja de ruta de producto (D2).** T18, T19 y T20 tocan la resolución del modo RDD (`internal/reviewtransaction/rdd_mode.go`, `internal/cli/review_mode.go`). La decisión D2 de `docs/audits/2026-11-03-vision-producto.md` (rama `inc/auditoria-vision-producto`) decide si RDD pasa a configuración por proyecto en `axiom.yaml` o sigue siendo un interruptor de máquina, y bloquea P4. Si D2 cambia el origen del modo, la regla de T19 («si el global no es `on`, se ignora el fallo del override») y el indicador explícito de T20 deben revisarse. El Bloque 4 se ejecuta después de resolver D2 o con esa revisión anotada en la tarea.

**Sin solape con distribución.** Comprobado el 2026-11-04 con `git diff origin/main...inc/distribucion-un-comando --stat`: esa rama toca `scripts/install.*`, `internal/update/**`, `cmd/axiom/main.go` y documentación de uso, y ninguna tarea de este documento edita esos ficheros.

**Orden frente a los otros bloques (2026-11-05).** Sale del cruce de ficheros que hizo la sesión coordinadora:

- **Pueden ir en paralelo ya:** T1–T5 y T7–T15. No comparten ficheros con otros bloques mientras no toquen la prosa compartida de instrucciones.
- **T6 es el eslabón 2 de una cadena en serie.** T6 edita «Delegated Verification Gate» en `internal/assets/skills/_shared/sdd-orchestrator-sections.md:46-69`. La cadena es:
  1. arreglos T9 y adaptadores T1;
  2. revisión nativa T6;
  3. guía del orquestador T1, que mueve esas secciones como bloques opacos;
  4. guía del orquestador T2 y T3;
  5. carril SDD versátil.

  T6 entra después del eslabón 1 y antes de guía T1, o los dos chocarían en la misma región. Cada eslabón regenera unos 13 goldens de orquestador, entre ellos `sdd-claude-claudemd`, `combined-claude-claudemd` y `sdd-opencode-multi-settings`.
- **T14, T16 y T17:** guía T1 también puede mover los techos de tamaño de `internal/components/sdd/review_ledger_contract_test.go`. Al rebasar, se regeneran o se ajustan.
- **T18–T20:** esperan a D2, que también bloquea P4.

**Previsión total:** unas 6.000 líneas en 20 tareas (sin D-RN1, D-RN2 ni D-RN3).

## Criterios de aceptación

- Ningún fichero existente de `contracts/` cambia: `git diff --stat origin/main -- contracts/` solo muestra ficheros nuevos.
- `docs/upstream-absorption-ledger.md` no cambia.
- Cada tarea: `go test` en proceso de los paquetes tocados en verde, `go vet` de esos paquetes y los golden regenerados cuando aplique.
- Al cierre: `go test ./internal/reviewtransaction/... ./internal/cli/... ./internal/reviewerprovider/... ./internal/components/...` en verde.

## Fuera de alcance

- Request-context, escalación de agente y lentes de #5345.
- `review_due` (874719ae, 71a47477, 972446f1), por D6.
- `rctx3` de OpenCode (#5300).
- Módulo `/v4`, telemetría y RDD por defecto.

## Progreso

- 2026-11-04: verificados los candidatos en tres lotes (corrección/recuperación, plataforma/arreglos pequeños, esquemas/riesgo). Comprobado en persona el fallo de `dangerous_sink` (`review_start_contract.go:395`) y el patrón `rctx1_` de `v1/start-v2.schema.json:116`. Carril elegido por el usuario: ODD.
- 2026-11-04: comprobado que no hay solape con `inc/distribucion-un-comando` y anotada la dependencia del Bloque 4 con D2 (commit `dfd389f8`).
- 2026-11-04: el usuario valida el alcance (T1–T20) y las recomendaciones: D-RN1 aplazada, D-RN2 deuda aceptada, D-RN3 descartada.
- 2026-11-05: T1 hecha. Esquema nuevo `start-v5` (fichero añadido; deriva de `start-v4` con `dangerous_sink` como `code` y `signal`), constante `ReviewIntegrationStartSchemaV5` como alias vivo, caso nuevo en `validateReviewStartRiskReasons` (solo admitido con start/v5; v3, v4 y v2 lo rechazan), frases de consentimiento EN/ES y contratos `axiom|gentle-ai.review-integration.start/v5`. Decisión sobre el contrato v1: su esquema `start/v2` está congelado, así que un START v1 sobre un candidato con ese motivo se rechaza antes del consentimiento y de crear autoridad (`invalid_request`, nombra el motivo y el contrato v2); no se degrada el motivo a otro inexacto. `capabilities/v2.6` sigue anunciando `start/v4` hasta T2 (su lista es cerrada). Comprobaciones: `go vet ./internal/cli/ ./internal/reviewtransaction/` OK; `go test ./internal/cli/ -run 'Start|Capabilit|Dialect|DualContract|PublishedSchema|Schema'` OK (2086 s). Fallos previos al cambio y ajenos: `TestOpaqueRepositoryContext*` esperan el texto inglés del SO («The system cannot find the file specified.») y esta máquina lo devuelve en castellano.

## Siguiente paso

Cuando el usuario pida implementar, preguntar la estrategia de cadena de PR (`stacked-to-main` o `feature-branch-chain`), porque la previsión de unas 6.000 líneas supera el presupuesto de entrega, y empezar por T1. El Bloque 4 espera a D2.
