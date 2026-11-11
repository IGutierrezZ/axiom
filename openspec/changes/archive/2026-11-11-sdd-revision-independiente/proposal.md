# Propuesta: Revisión independiente en SDD (sdd-revision-independiente)

> Fase `propose` · rama `inc/sdd-revision-independiente` · 2026-11-11 · sin `explore.md` (entrada: decisiones del usuario transmitidas por el orquestador, Engram #2108 y #2114).
> Kickoff sellado: `continuous`, rol único `fullstack` (`blocking`). Este cambio no tiene puertas propias.
> Entrega: `single-pr`; si supera 400 líneas autoradas, `size:exception`.

## 0. Resumen

Hoy nada en SDD comprueba de forma independiente que lo producido cumple la intención y los estándares técnicos del proyecto. Este cambio añade cuatro capacidades, todas **informativas** (no bloquean el archivado) y todas **ajenas a RDD**:

1. **Verify con cumplimiento técnico, por rol.** Tras el apply de cada rol, el orquestador ofrece lanzar `sdd-verify`, que gana una sección de cumplimiento técnico: skills, contexto técnico y convenciones del proyecto contrastados con el diff del rol.
2. **Juicio opcional tras un verify limpio.** El orquestador ofrece `judgment-day` sobre el diff del rol. El usuario elige: verify, verify y juicio, o ninguno.
3. **`continuous` solo con el rol `fullstack`.** `axiom sdd kickoff seal` rechaza `continuous` con cualquier otro rol; los kickoffs ya sellados se siguen leyendo.
4. **Revisión de cobertura opcional tras spec y tras design.** Revisores de contexto limpio y solo lectura buscan casos de uso, casos límite y escenarios de error que falten; hallazgos por unión, sin autocorrección.

Hay **dos decisiones de producto abiertas** (§2.2) que condicionan parte de la spec: cómo se comportan las ofertas en `continuous` y qué hace el retrosellado.

## 1. Propósito (Intent)

### 1.1 Problema

| Hueco | Evidencia |
|---|---|
| `sdd-verify` solo se ejecuta si alguien lo pide y no audita estilos, patrones ni skills del proyecto: su alcance es funcional (requisitos, tests, build). | `internal/assets/skills/sdd-verify/SKILL.md:19` («Run when the orchestrator explicitly requests verification»), `:64-66`; `references/report-format.md:5-11` (sin sección de cumplimiento técnico) |
| Ningún punto del flujo ofrece verificación: el grafo la marca como diagnóstico opcional y el orquestador no la propone tras el apply. | `internal/assets/skills/_shared/sdd-orchestrator-sections.md:36-43` |
| Spec y design se autoevalúan: la tabla de cobertura (caminos felices, casos límite, errores) la rellena el propio autor de la spec. | `internal/assets/skills/sdd-spec/SKILL.md:234-237` |
| El único control externo en modo automático es de contrato, no de contenido: el validador de `sdd-design` y `sdd-apply` «no inspecciona el diff» y busca deriva respecto a las entradas, no huecos de cobertura. | `sdd-orchestrator-sections.md:256-265` |
| `judgment-day` existe, pero solo se activa por petición explícita del usuario y no sabe tomar como objetivo el trabajo de un rol SDD. | `internal/assets/skills/judgment-day/SKILL.md:12`, `:39` |
| `continuous` se admite con cualquier roster, aunque los roles existen para separar responsabilidades con puertas humanas que `continuous` no tiene. | `internal/kickoff/schema.go:83-119` (`Validate` no relaciona estilo y roles); `internal/kickoff/seal.go:64`; REQ-21.7 (`openspec/specs/sdd-block-review-gates/spec.md:15-29`) |
| El retrosellado escribe `continuous` con los roles detectados, y la detección devuelve `core` cuando `design.md` no declara roles: casi todo retrosellado con `design.md` produce hoy un roster distinto de `fullstack` en `continuous`. | `internal/kickoff/infer.go:20-26`, `:83-92`; `internal/multirole/detector.go:34-55`; REQ-21.4 (`openspec/specs/sdd-lane-kickoff/spec.md:119`) |

### 1.2 Por qué ahora

El cambio en curso `carril-sdd-versatil` (modo ISO) se rebasará sobre este y necesita un verify que ya audite el cumplimiento técnico. Además, la incoherencia `continuous` con roles crece con cada cambio multirrol que se sella.

### 1.3 Resultado esperado

- Al terminar el apply de cada rol, el usuario puede pedir un verify que informe del cumplimiento funcional **y** técnico del trabajo de ese rol.
- Si el verify sale limpio, puede pedir además un juicio con dos jueces ciegos; los arreglos se limitan a severos confirmados por ambos.
- Al terminar spec y design, puede pedir una revisión de cobertura que señale huecos sin tocar el artefacto.
- En `checkpointed`, la puerta correspondiente presenta esos resultados junto al artefacto.
- `axiom sdd kickoff seal` rechaza `continuous` con roles distintos de `fullstack`, y ningún kickoff existente ni archivado deja de leerse.

## 2. Decisiones

### 2.1 Decisiones que la aprobación ratifica

Recomendaciones de esta fase. Aprobar la propuesta las fija; cada una indica la alternativa descartada.

| ID | Decisión | Alternativa descartada |
|---|---|---|
| D1 | **Las ofertas son del orquestador.** Verify, juicio y cobertura se ofrecen con prosa de los assets del orquestador (`sdd-orchestrator-sections.md`, proyectada a todos los runtimes) y de las skills. No se añaden estados, campos de ledger ni puertas en Go. | Modelarlas en la máquina de puertas (`internal/kickoff/machine.go`): convertiría una oferta informativa en un bloqueo y acercaría SDD a la frontera de RDD. |
| D2 | **Frontera con RDD intacta.** Ninguna oferta usa vocabulario ni verbos de RDD (`lineage`, `candidate`, `acknowledge`, `burn`, `axiom review …`), no emite recibos ni pide consentimiento RDD. `judgment-day` es una skill independiente (`judgment-day/SKILL.md:24`: no emite recibo). Las guardas AST de `internal/sddstatus/review_offer_absence_guard_test.go` (`OfferReviewAfterVerify`, `ReviewCore`) no cambian, y una prueba de pin comprueba que el texto nuevo no nombra RDD. | Reutilizar la oferta post-verify de RDD, retirada en `openspec/specs/rdd-post-verify-review-offer/spec.md`. |
| D3 | **Verify no bloquea.** Se conserva la doctrina de `persistence-contract.md:69` y `sdd-verify/SKILL.md:35`: verify, juicio y cobertura informan; la decisión es humana. Un hallazgo técnico no altera `nextRecommended` ni bloquea el archivado. | Bloquear por hallazgos: pertenece a `carril-sdd-versatil` (modo ISO). |
| D4 | **Ámbito y fichero del verify por rol.** El verify de un rol inspecciona el diff de ese rol y escribe en el `verify_file` que el kickoff ya asigna (`verify-report.md` para `fullstack`, `verify-report.<rol>.md` para el resto; `internal/kickoff/types.go:218-223`). La sección **Cumplimiento técnico** contrasta el diff con: (a) las skills de `skill_resolution` del apply del rol; (b) las skills del registro cuyos disparadores casan con los ficheros tocados; (c) el `entry` de `technicalContext` cuando la proyección lo declara; (d) las reglas y el contexto de `openspec/config.yaml`. Cada hallazgo cita la regla y la ubicación en el diff. El verify global multirrol (REQ-21.15, `internal/sddstatus/governance_handoff.go:248-266`) no cambia. | Un único verify al final del cambio: pierde el momento en que el rol aún se puede corregir dentro de su puerta. |
| D5 | **Juicio sobre un objetivo SDD por rol.** `judgment-day` acepta como objetivo el diff de un rol SDD; que el usuario acepte la oferta del orquestador cuenta como la petición explícita que exige su contrato de activación. Se conservan sus reglas: dos jueces ciegos en paralelo, corroboración por acuerdo, arreglo solo de severos confirmados por ambos, como mucho dos rondas, veredicto `APPROVED` o `ESCALATED`. El actor de arreglos solo edita dentro de las raíces de edición autorizadas del cambio. Si el juicio arregla código, su verificación final pasa a ser el informe vigente del rol, y los hallazgos del verify anterior se conservan como contexto histórico (`report-format.md:9`). | Juicio sin verify previo, o sobre todo el cambio: el usuario lo quiere por rol y solo tras un verify limpio. |
| D6 | **Cobertura por unión, sin autocorrección.** Uno o dos revisores de contexto limpio y solo lectura (el número lo fija el design) leen la propuesta y el artefacto (spec, o spec y design) y devuelven casos de uso, casos límite y escenarios de error que falten. Se informa la **unión** de hallazgos, deduplicados. El informe se guarda como artefacto informativo de la fase (nombre en design), fuera de los digests de las puertas. Las correcciones las hace `sdd-spec` o `sdd-design` por la salida «iterar» de la puerta (en `checkpointed`) o por una nueva ejecución de la fase. | Intersección, como en `judgment-day`: aquí el objetivo es no perder huecos. Autocorrección: mezclaría revisor y autor. |
| D7 | **Separar la validación de sellado de la de lectura.** La regla «`continuous` solo con `fullstack`» se comprueba al sellar (`Seal` y `ParseSealArgs`, mensaje con la salida: `--execution-style checkpointed` o quitar los roles). `Validate()`, que usa `ParseKickoff` (`schema.go:69-78`), no la comprueba, así que los kickoffs existentes y archivados se siguen leyendo y conservan su comportamiento. Se interpreta la regla literalmente: el roster debe ser exactamente `[fullstack]`; un único rol especializado (p. ej. solo `backend`) también se rechaza en `continuous`. | Meterla en `Validate()`: rompería `ParseKickoff` para cualquier cambio ya sellado que la viole. |
| D8 | **`--from-session-pace auto` con roles.** Hoy deriva `continuous` (`args.go:266-277`). Con roles distintos de `fullstack`, el sellado se rechaza con el mismo mensaje; nunca se deriva `checkpointed` en silencio, porque REQ-21.5 prohíbe asumir la modalidad. | Derivar `checkpointed` automáticamente: cambia la decisión del usuario sin decírselo. |

### 2.2 Decisiones de producto (resueltas por el usuario el 2026-11-11)

**Resolución:** P1 = **(c)**, preferencia en el pre-vuelo. P2 = **(b)**, el retrosellado queda exento y conserva `continuous`. Las tablas se conservan como registro de las alternativas descartadas.

**P1. Ofertas en `continuous`.** REQ-21.7 dice que en `continuous` el orquestador no se detiene. Ofrecer verify, juicio o cobertura es preguntar, y preguntar es detenerse. Tras D7, `continuous` siempre es `fullstack`, que es el caso más frecuente (incluido este cambio).

| Opción | Efecto | Coste |
|---|---|---|
| (a) Solo en `checkpointed`. | En `continuous` no se ofrece nada; verify sigue disponible a petición. | Pierde la revisión independiente justo en el caso más común. |
| (b) Ofrecer también en `continuous`. | Pregunta en cada punto; se enmienda REQ-21.7 para distinguir «oferta» de «parada». | Rompe la automatización que el usuario eligió con `continuous`. |
| (c) Preferencia en el pre-vuelo. | Una pregunta más en el pre-vuelo (verify / verify y juicio / ninguno; cobertura sí/no) que el orquestador aplica sin preguntar en `continuous` y usa como respuesta por defecto en `checkpointed`. Prosa del orquestador, cacheada en sesión como la estrategia de entrega; sin campo nuevo en `kickoff.yaml`. | Una pregunta más al empezar; la preferencia no sobrevive a otra sesión salvo que se persista (decisión del design). |

Decisión del usuario: **(c)**, porque respeta «el usuario elige» y «`continuous` no se detiene» a la vez.

**P2. Retrosellado de un cambio con roles distintos de `fullstack`.** Hoy `InferKickoff` sella `continuous` con los roles detectados, que son `core` cuando `design.md` no declara roles (`detector.go:34-55`). REQ-21.4 exige retrosellar «sin bloquear el trabajo ni interrogar retroactivamente al usuario».

| Opción | Efecto | Coste |
|---|---|---|
| (a) Inferir `checkpointed`. | Cumple la regla. | Abre puertas `pending` para spec, design y tasks ya escritos (`machine.go:75-77`): el cambio en curso se detiene hasta aprobarlas. Contradice REQ-21.4. |
| (b) Eximir el retrosellado (`sealed_by: inferred`) y conservar `continuous`. | No cambia ningún cambio preexistente; la regla se aplica a todo sellado explícito. | Algunos kickoffs inferidos seguirán incumpliendo la regla. |
| (c) No inferir y exigir sellado explícito. | Cumple la regla. | Interroga retroactivamente; contradice REQ-21.4. |

Decisión del usuario: **(b)**. El retrosellado es un mecanismo de compatibilidad para cambios anteriores a INC-21, y (a) bloquearía cambios en curso sin aviso.

## 3. Alcance (Scope)

### 3.1 Dentro de alcance (In Scope)

1. **`sdd-verify` con cumplimiento técnico por rol** (D4): `SKILL.md` (activación por oferta aceptada o petición, ámbito por rol, sección nueva, fuentes de reglas) y `references/report-format.md` (sección «Cumplimiento técnico»). Conserva D3 y la regla de no mutar.
2. **`judgment-day` con objetivo SDD por rol** (D5): `SKILL.md` y `references/prompts-and-formats.md` (identidad de objetivo por rol, raíces de edición, informe vigente tras un arreglo).
3. **Revisión de cobertura** (D6): instrucciones de los revisores en un asset compartido (ubicación en design), sin identidad de agente nueva salvo que el design la justifique; mención en `sdd-spec/SKILL.md` y `sdd-design/SKILL.md` de que la tabla de cobertura propia no sustituye a la revisión.
4. **Ofertas del orquestador** (D1, D2, P1): sección nueva o ampliada en `internal/assets/skills/_shared/sdd-orchestrator-sections.md` (verify tras cada apply de rol, juicio tras un verify limpio, cobertura tras spec y tras design, y presentación de resultados en las puertas `spec`, `design` y `role-apply:<rol>` en `checkpointed`), proyectada a todos los runtimes; ajuste del grafo de dependencias (`:36-43`).
5. **Regla `continuous` solo con `fullstack`** (D7, D8, P2): validación de sellado en `internal/kickoff/` (`seal.go`, `args.go`, `schema.go`), ajuste de `infer.go` según P2, mensajes de `internal/cli/sdd_kickoff.go`, y el texto del orquestador que presenta la opción `continuous` (`sdd-orchestrator-sections.md:80`).
6. **Convención de ficheros**: `_shared/openspec-convention.md` registra los informes de cobertura y, si el design lo fija, el del juicio.
7. **Pruebas**: sellado (`continuous` + `fullstack` sella; `continuous` + otro rol se rechaza; `--from-session-pace auto` + roles se rechaza; un `kickoff.yaml` existente que incumple la regla se lee con `ParseKickoff`), retrosellado según P2, pin de texto sin vocabulario RDD y guardas AST sin cambios.
8. **Specs** de §4 y **copias versionadas** regeneradas (`testdata/golden/`, `AGENTS.md`, `.codex/AGENTS.md`, `GEMINI.md` y demás que proyecte el orquestador).

### 3.2 Fuera de alcance (Out of Scope)

- Modo ISO, matriz de verify, `Cubre:` y bloqueo por evidencias (`carril-sdd-versatil`).
- Que verify, juicio o cobertura bloqueen el archivado o una puerta.
- Cualquier cambio en RDD, en `rdd-post-verify-review-offer` o en las guardas AST de ausencia de oferta.
- Estados, campos de ledger o puertas nuevas en Go para las ofertas.
- Archivado en la misma PR.
- Migrar o reescribir kickoffs ya sellados o archivados.

### 3.3 Qué no cambia

El vocabulario y la lectura de `gates.yaml`; REQ-21.15 (verify global multirrol tras todos los `role-apply` aprobados); la regla de `verify` de no mutar código ni tareas; `--docs-publication auto` (solo con `continuous`, que ahora implica `fullstack`); la frontera INC-18 entre SDD y RDD.

## 4. Capacidades (Capabilities)

> Contrato con `sdd-spec`. Baseline: `openspec/specs/` (el repositorio no declara `baseline:`).

### 4.1 Nuevas capacidades (New Capabilities)

- `sdd-independent-review`: ofertas del orquestador (verify por rol, juicio tras un verify limpio, cobertura tras spec y tras design), elección del usuario, sección de cumplimiento técnico del verify y sus fuentes, juicio sobre objetivo SDD por rol, cobertura por unión sin autocorrección, presentación de resultados, carácter informativo y frontera con RDD. El comportamiento en `continuous` depende de P1.

### 4.2 Capacidades modificadas (Modified Capabilities)

- `sdd-preflight-configuration`: requisito nuevo «`continuous` solo con el rol `fullstack`», comprobado al sellar y no al leer (D7); `--from-session-pace auto` con roles se rechaza (D8). Si P1 = (c), requisito de la pregunta de pre-vuelo sobre revisión independiente.
- `sdd-lane-kickoff`: REQ-21.4, el retrosellado según P2.
- `sdd-block-review-gates`: REQ-21.8 y REQ-21.9 presentan el informe de cobertura cuando existe; REQ-21.11 presenta el verify y el juicio del rol cuando existen. REQ-21.7 solo cambia si P1 = (b).
- `multi-role-fan-out`: solo si la spec fija que `verify-report.<rol>.md` incluye la sección de cumplimiento técnico; la fase de spec lo confirma sin alterar la semántica del veredicto que lee la barrera.
- `sdd-orchestrator-assets`: paridad del texto de las ofertas entre runtimes; la fase de spec confirma si basta con un requisito en `sdd-independent-review`.

`rdd-post-verify-review-offer` **no se modifica**.

## 5. Enfoque (Approach)

- **Prosa antes que máquina.** Las ofertas y la presentación de resultados viven en los assets del orquestador y en las skills (D1). Go solo cambia en `internal/kickoff/` y en los mensajes de la CLI.
- **Dos validaciones del kickoff.** Se añade una comprobación de sellado (nombre en design, p. ej. `ValidateForSeal`) que llama a `Validate()` y añade la regla de estilo y roles. `Seal` y el análisis de argumentos la usan; `ParseKickoff` sigue con `Validate()`.
- **Reutilizar lo que existe.** Ficheros de verify por rol del kickoff, agentes `jd-judge-a`, `jd-judge-b` y `jd-fix-agent` de `judgment-day`, y raíces de edición autorizadas del cambio. No se crean identidades de agente nuevas salvo justificación del design.
- **Cobertura como informe aparte.** El informe de cobertura no forma parte del artefacto revisado, así que no altera su digest ni reabre una puerta. Solo cambia el artefacto cuando su autor lo corrige.
- **Doctrina en un punto.** El texto de las ofertas se define una vez en `sdd-orchestrator-sections.md` y se proyecta; los goldens se regeneran al final con `-update`, nunca a mano.
- **Skills según la guía de estilo.** Los cambios en `SKILL.md` respetan `docs/skill-style-guide.md` (contrato de activación, reglas, compuertas de decisión y contrato de salida) y conservan la intención de las reglas actuales.

## 6. Áreas afectadas (Affected Areas)

| Área | Impacto | Qué cambia |
|---|---|---|
| `internal/kickoff/` (`schema.go`, `seal.go`, `args.go`, `infer.go` y sus tests) | Modificado | Validación de sellado `continuous` ⇒ `[fullstack]`; `--from-session-pace auto` con roles; retrosellado según P2; lectura tolerante |
| `internal/cli/sdd_kickoff.go` | Modificado | Mensaje de rechazo con la salida y ayuda |
| `internal/assets/skills/sdd-verify/` (`SKILL.md`, `references/report-format.md`) | Modificado | Activación por oferta, ámbito por rol, sección «Cumplimiento técnico» |
| `internal/assets/skills/judgment-day/` (`SKILL.md`, `references/prompts-and-formats.md`) | Modificado | Objetivo SDD por rol, raíces de edición, informe vigente |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modificado | Ofertas, presentación en puertas, grafo, texto de la opción `continuous` |
| `internal/assets/skills/_shared/` (asset de cobertura, `openspec-convention.md`) | Nuevo y modificado | Instrucciones de los revisores de cobertura; ficheros nuevos del cambio |
| `internal/assets/skills/sdd-spec/SKILL.md`, `sdd-design/SKILL.md` | Modificado | La cobertura propia no sustituye a la revisión |
| `internal/assets/` (tests de pin) e `internal/sddstatus/review_offer_absence_guard_test.go` | Modificado o sin cambios | Pin de texto sin RDD; la guarda AST se conserva intacta |
| `testdata/golden/`, `AGENTS.md`, `.codex/AGENTS.md`, `GEMINI.md` | Regenerado | Copias proyectadas |
| `openspec/specs/` (capacidades de §4) | Nuevo y modificado | Deltas |

## 7. Riesgos (Risks)

| Riesgo | Prob. | Mitigación |
|---|---|---|
| P1 y P2 sin resolver frenan la spec de `sdd-independent-review` (comportamiento en `continuous`) y de REQ-21.4. | Alta | Se devuelven al orquestador con recomendación; la spec del resto avanza en paralelo. |
| La oferta de juicio tras verify se parece a la oferta de RDD retirada y se lee como su reintroducción. | Media | D2: sin vocabulario ni verbos RDD, pin de texto, guardas AST intactas y declaración explícita en la spec. |
| El actor de arreglos del juicio edita fuera de lo autorizado. | Media | D5: limitado a las raíces de edición autorizadas del cambio; los arreglos quedan en el progreso del rol. |
| «La verificación final sustituye al verify previo» contradice «conservar los hallazgos históricos» (`persistence-contract.md:69`). | Media | D5: el informe vigente se reescribe y los hallazgos anteriores pasan a contexto histórico. |
| Con P2 = (b), quedan kickoffs inferidos que incumplen la regla. | Baja | Se leen igual que hoy; si el design lo considera útil, `status` puede emitir una nota no bloqueante. |
| La regla rechaza un rol especializado único en `continuous`, algo que alguien podía usar a propósito. | Baja | D7 lo declara y el mensaje de rechazo da la salida. |
| Un revisor de cobertura con contexto limpio repite falsos positivos o hallazgos ya cubiertos. | Media | Unión deduplicada, informe informativo, el autor decide en la puerta. |
| Coste: dos jueces y hasta dos rondas por rol, más uno o dos revisores por fase. | Media | Todo es opcional y lo elige el usuario; el orquestador avisa del coste antes de lanzar el juicio (contrato de previsión de trabajo largo). |
| Pines o goldens existentes que fijan «Run when the orchestrator explicitly requests verification» o el texto del grafo. | Media | Regenerar goldens con `-update` y ajustar los pines en el mismo cambio. |
| Rebase de `carril-sdd-versatil` sobre este cambio (tocan `sdd-verify` y `report-format.md`). | Alta | Este cambio no toca el bloqueo ni la matriz; el conflicto se limita a prosa. Los goldens se regeneran después del rebase. |
| Tamaño por encima de 400 líneas. | Alta | `single-pr` con `size:exception` (pre-vuelo). |

Riesgo global: **medio**. No hay migración de datos y la lectura de kickoffs no cambia; el riesgo se concentra en la frontera con RDD y en las dos decisiones abiertas.

## 8. Entrega (Review Workload Forecast)

Estimación (no medida), sin goldens: Go de `internal/kickoff` y CLI con tests, 250–350; prosa de skills y del orquestador, 250–350; specs delta y capacidad nueva, 400–550; pines, 50–100. Total: **950–1.350 líneas autoradas**.

```
Decision needed before apply: Yes
Chained PRs recommended: No
400-line budget risk: High
```

La decisión pendiente antes del apply es P1 y P2 (§2.2), no la estrategia de entrega, que ya es `single-pr` con `size:exception`.

## 9. Plan de reversión (Rollback Plan)

- **Por código.** `git revert` de los commits del cambio (o de la PR fusionada). No hay migraciones.
- **Por datos.** No se escribe ningún campo nuevo en `kickoff.yaml` ni en `gates.yaml`. Un kickoff sellado con el binario nuevo cumple también las reglas del anterior. Los informes nuevos (cobertura, cumplimiento técnico) son Markdown informativo que el binario anterior ignora.
- **Por doctrina.** `axiom sync` reproyecta las instrucciones del binario revertido y desaparecen las ofertas.
- **Señal para revertir.** Un `kickoff.yaml` existente que deja de leerse; un sellado `continuous` + `fullstack` rechazado; una guarda AST de ausencia de oferta que falla; un texto proyectado que nombra un verbo RDD; un verify o un juicio que bloquea el archivado.

## 10. Dependencias (Dependencies)

- Resolución de P1 y P2 por el usuario (§2.2).
- `carril-sdd-versatil` (en otra rama, aún no en `main`) se rebasará sobre este cambio.
- Ya existentes: skill `judgment-day` y sus agentes, `verify_file` por rol en el kickoff, `technicalContext` en la proyección del estado, registro de skills.
- Restricciones de trabajo: TDD desactivado (`strict_tdd: false`), RDD desactivado, sin compilar binarios sueltos (solo `go test` acotado por paquete), commits Conventional sin atribución de IA.

## 11. Criterios de éxito (Success Criteria)

- [ ] **Sellado.** `axiom sdd kickoff seal --execution-style continuous` sella con el rol `fullstack` y se rechaza con cualquier otro rol (incluido uno solo especializado), con un mensaje que nombra la salida; `--from-session-pace auto` con roles también se rechaza.
- [ ] **Lectura tolerante.** Un `kickoff.yaml` existente con `continuous` y roles distintos de `fullstack` se lee con `ParseKickoff` sin error y conserva su comportamiento (prueba).
- [ ] **Retrosellado.** `InferKickoff` se comporta según P2 y su prueba lo fija.
- [ ] **Verify técnico.** El informe de verify de un rol contiene la sección «Cumplimiento técnico» con hallazgos que citan regla (skill, contexto técnico o `config.yaml`) y ubicación; se escribe en el `verify_file` del rol; no altera `nextRecommended` ni bloquea el archivado.
- [ ] **Ofertas.** La doctrina proyectada ofrece verify tras el apply de cada rol, juicio solo tras un verify limpio y cobertura tras spec y tras design, con las opciones verify / verify y juicio / ninguno; en `checkpointed` los resultados se presentan en la puerta correspondiente; en `continuous` se comporta según P1.
- [ ] **Juicio.** `judgment-day` acepta un objetivo SDD por rol, arregla solo severos confirmados por ambos jueces dentro de las raíces autorizadas, en como mucho dos rondas, y su verificación final pasa a ser el informe vigente conservando los hallazgos anteriores como historial.
- [ ] **Cobertura.** El informe de cobertura une y deduplica los hallazgos, no modifica el artefacto revisado ni su digest, y la doctrina enruta las correcciones por «iterar» o por una nueva ejecución de la fase.
- [ ] **Frontera RDD.** `review_offer_absence_guard_test.go` pasa sin cambios y un pin comprueba que el texto nuevo no contiene verbos ni vocabulario de RDD.
- [ ] **Paridad y goldens.** Goldens y copias versionadas regenerados; el texto de las ofertas es idéntico en todos los runtimes.
- [ ] **CI verde.** `go test` de los paquetes tocados y `go vet ./...` pasan; los ratchets no empeoran.
