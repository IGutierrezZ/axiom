# Propuesta: Paradas con tres salidas en ODD y SDD (paradas-tres-salidas)

> Fase `propose` · rama `inc/paradas-tres-salidas` · 2026-11-02 · entrada: `explore.md` (autoritativo).
> Kickoff sellado: `continuous`, `handoff_policy: none`, rol único `fullstack` (`blocking`). Este cambio no tiene puertas propias.
> Decisión de producto de origen: Engram #1811 (`product/paradas-tres-salidas`): las tres salidas sustituyen a `handoff_policy` y al `handoff.md` intermedio, que no se implementarán.

## 0. Resumen

Hoy una parada de SDD solo admite «aprobar» o «rechazar», y la forma prevista de ceder el trabajo a otra persona es un `handoff.md` (intermedio, controlado por `handoff_policy`, y otro final de integración que desbloquea `verify`). ODD no tiene ninguna parada. Este cambio unifica el modelo: en cada parada el humano elige entre **tres salidas** —(1) seguir iterando sobre el artefacto actual, (2) aprobarlo, subirlo a la rama de integración y parar, de modo que la fase siguiente la hace otra persona, (3) aprobarlo y seguir uno mismo con la fase siguiente—, retira el `handoff.md` del flujo SDD y deja `handoff_policy` como campo deprecado que se acepta y se ignora. Ejemplos: dejo la spec lista y aprobada y el design lo hace otra persona; dejo el design aprobado y las tasks de un rol las hace otra persona.

## 1. Propósito (Intent)

### 1.1 Problema

| Hueco | Evidencia |
|---|---|
| La puerta de SDD solo ofrece dos salidas; no hay forma tipada de «parar aquí y que otra persona siga». | `internal/sddstatus/governance.go:235-282` (`newGovernanceGateQuestion`); `internal/consentenvelope/envelope.go:61` (`len(core.Choices) != 2`) |
| El relevo entre personas depende de `handoff.md`, un artefacto que nadie revisa como tal: el que de verdad se lee es la spec y el plan. | `explore.md` §1.1 y §1.2; `internal/assets/skills/sdd-spec/SKILL.md:242-250`, `sdd-design/SKILL.md:183-193` |
| `handoff_policy` es obligatorio al sellar y su único efecto es decidir si `verify` mira `handoff.md`. | `internal/kickoff/args.go:257-259`; `internal/sddstatus/governance.go:514` |
| `verify` multirol se desbloquea con un `handoff.md` `ready`, cuando la condición real ya existe: todos los `role-apply` aprobados. | `internal/sddstatus/governance.go:491-546`; `internal/kickoff/machine.go:152-178` (`LastRoleClosed`) |
| ODD no tiene ninguna parada: no hay momento previsto para que otra persona recoja el plan antes de que empiece el código. | `internal/components/agentguidance/routing.go:54` («do not ask any further governance question»); REQ-21.2 |

### 1.2 Por qué ahora

El usuario ha decidido (Engram #1811) no implementar el `handoff.md` intermedio. Mientras `handoff_policy` siga siendo obligatorio y los skills sigan ofreciendo «generar `handoff.md` o continuar», el producto pide una decisión que ya no tiene efecto útil y no ofrece la que sí se necesita: dejar el trabajo listo en la rama de integración para otra persona.

### 1.3 Resultado esperado

- Toda parada (puertas SDD en `checkpointed` y la parada única de ODD) presenta las mismas tres salidas, con la misma redacción en todos los runtimes.
- La salida 2 registra `approved` en el ledger con una marca opcional de que la fase siguiente queda cedida a otra persona; quien recoge el relevo empieza esa fase siguiente, no revisa la anterior. Un binario anterior lee el registro como un `approved` normal.
- `axiom sdd status` muestra que la fase siguiente está cedida.
- `verify` multirol se desbloquea cuando todos los `role-apply` están aprobados, sin `handoff.md`.
- Sellar un kickoff ya no exige `--handoff-policy`, y los orquestadores sincronizados con versiones anteriores siguen sellando sin error.

## 2. Decisiones que la aprobación ratifica

Recomendación de la orquestación a partir de `explore.md` §3. Aprobar esta propuesta las fija; cada una indica la alternativa descartada.

| ID | Decisión | Alternativa descartada |
|---|---|---|
| D1 | **Modelo de salidas.** Salida 1 = `rejected` con motivo (iterar; sin cambios respecto a hoy: la remediación la reabre por digest). Salida 2 = **`approved` y parar**: se registra `approved` en `gates.yaml` más un campo opcional que marca que la fase siguiente queda para otra persona (nombre en design, p. ej. `next: handoff`); el orquestador para y devuelve al humano el destino de la PR (`prTarget`) y la orden. Salida 3 = `approved` y seguir. **Compatibilidad:** `ParseGateLedger` usa `yaml.Unmarshal` no estricto (`internal/kickoff/schema.go:110`), así que un binario anterior ignora el campo opcional y lee la decisión como `approved`; el vocabulario `approved|rejected` no cambia. Las tres opciones del sobre `gateQuestion` invocan `axiom sdd gate record` (la salida 2 con un flag adicional, nombre en design), de modo que la comprobación de prefijo compartido se conserva. `consentenvelope.Core.ValidateCompleteness` se generaliza para admitir 3 opciones sin cambiar el sobre de autoridad de edición, que sigue exigiendo exactamente 2 (prueba de no regresión). | (a) Dejar la puerta `pending` para que decida quien recibe: el usuario quiere aprobar lo que entrega, y quien recibe empieza la fase siguiente, no revisa la anterior. (b) Una tercera decisión `handed_off` en el ledger: `ParseGateLedger` rechaza decisiones desconocidas (`schema.go:143-145`) y un `axiom` anterior no podría leer `gates.yaml`. |
| D2 | **Dónde hay paradas.** SDD `checkpointed`: las puertas actuales `spec`, `design`, `tasks` y `role-apply:<rol>`, con las tres salidas. En `role-apply:<rol>`, la salida 2 = rol aprobado y otra persona sigue con lo siguiente (otro rol o `verify`). La puerta `integration` conserva sus dos salidas (bloquea `archive` y su aprobación exige evidencia de fusión; no es una parada de planificación). SDD `continuous`: sin paradas, como hoy (REQ-21.7). No se añade puerta de `proposal` ni de `explore`. ODD: **una única parada por defecto** tras crear el documento de feature (el plan) y antes de la primera escritura de código, con las mismas tres salidas; su salida 2 = plan aprobado, commit de `odd/tasks/<feature>.md` en la rama y PR, y la implementación la hace otra persona. Se enmiendan REQ-21.2 y los pines de `routing_test.go` que dicen «no further governance question»: la parada no es una pregunta de gobernanza. | Paradas también en `continuous`, tras `proposal`/`explore` o al cierre de cada unidad de trabajo ODD: más fricción y solapamiento con `delivery_strategy`. |
| D3 | **«Subir a main».** Significa PR **fusionada** en la rama de integración, no solo empujada, porque `axiom branch ensure` en otra máquina crea la rama desde la base y no recupera artefactos sin fusionar (`internal/branch/engine/ensure.go:38-66`). Axiom nunca hace push ni PR: la salida 2 aprueba la puerta, deja los artefactos commiteados en la rama del cambio y devuelve al humano el destino (`prTarget` de `axiom branch status --json`) y la orden de PR; push, PR y merge son del humano. En `multirepo`, el repositorio de specs se trabaja in situ en la rama de integración: la salida 2 es commit allí y push humano, sin PR. Con almacén `engram` o `none` la salida 2 no está disponible (no hay ficheros que subir) y la parada lo dice así. | Bastaría con empujar la rama: quien recoge el relevo no la obtendría con `ensure`. |
| D4 | **Fusiones intermedias.** Se enmiendan REQ-25.20 y S-16 de `change-branch-lifecycle` para que una fusión intermedia de artefactos de planificación no haga que el cambio parezca integrado ni «pendiente de archivar», y para que `ensure --archive` no lo acepte como base fusionada. El criterio concreto (marcador, artefacto o estado de puertas) lo fija el diseño. | Aceptar que un cambio a medias aparezca como pendiente de archivar: confunde `status`, el dashboard y `ensure --archive`. |
| D5 | **`handoff.md` final.** Deja de ser condición de `verify` multirol: `verify` se desbloquea cuando todos los `role-apply` están aprobados (`kickoff.LastRoleClosed`). `internal/cli/sdd_gate.go` deja de escribirlo (se retiran `writeIntegrationHandoff` y `kickoff.IntegrationHandoff`). Se conservan `internal/handoff`, `axiom handoff`, el dashboard y la TUI como utilidad general fuera del flujo SDD. Se retira REQ-21.14, se reescribe REQ-21.15 y se conserva el aviso de REQ-21.13 sin generar fichero. | Retirar también `axiom handoff`, dashboard y TUI: ensancha el cambio sin que la decisión lo pida. |
| D6 | **`handoff_policy`.** Campo deprecado: el flag `--handoff-policy` se acepta y se ignora (no rompe orquestadores ya sincronizados), deja de ser obligatorio y los kickoffs nuevos escriben siempre `handoff_policy: none` para que los binarios anteriores los validen (enmienda tras R-1 del design); los kickoffs existentes con `none` o `per_checkpoint` se leen sin error; `Validate` acepta vacío. Se elimina su único lector con efecto (`governance.go:514`) y se actualizan REQ-21.4 y REQ-21.5 y las instrucciones y skills que lo piden, incluida la pregunta de dos salidas de `sdd-spec` y `sdd-design` («`handoff.md` o continuar»). La retirada completa del campo queda para un cambio posterior. | Eliminarlo del todo ya: el flag es obligatorio hoy y un binario que lo rechazara como bandera desconocida (`args.go:229-230`) rompería el sellado hasta resincronizar. Reutilizarlo como «salida por defecto»: mezcla conceptos. |
| D7 | **Entrega.** Una sola PR con `size:exception` (pre-vuelo del usuario: `single-pr` más excepción). Se espera superar las 400 líneas por plantillas, specs y goldens. | PRs encadenadas: el usuario ya eligió una sola PR. |

## 3. Alcance (Scope)

### 3.1 Dentro de alcance (In Scope)

1. **Sobre de tres opciones.** `newGovernanceGateQuestion` y su `Validate` ofrecen iterar (`rejected`), aprobar y parar (`approved` + flag) y aprobar y seguir (`approved`), las tres sobre `axiom sdd gate record`; `consentenvelope.Core.ValidateCompleteness` admite 3 opciones conservando la comprobación de prefijo; prueba de no regresión del sobre de autoridad de edición (2 opciones) y del contrato de *Lossless Blocking Prompt*.
2. **Salida 2 = `axiom sdd gate record --decision approved` más un flag** (nombre en design): registra `approved` con el campo opcional de fase siguiente cedida; antes de registrar comprueba que los artefactos de la puerta están confirmados en la rama del cambio (si no, rechaza con continuación y no escribe nada); después emite la rama, el `prTarget` y la orden de PR exacta (o, en el repositorio de specs de un `multirepo`, el push sobre la integración). No ejecuta git remoto. Con almacén `engram` o `none` el flag se rechaza: la salida no está disponible.
3. **Ledger y estado.** Campo opcional en `GateRecord` (con `omitempty`), escrito solo por la salida 2. `axiom sdd status` muestra que la fase siguiente está cedida (en el JSON de `governance` y en el texto); la forma exacta la fija el design.
4. **Retirada del `handoff.md` del flujo SDD.** `sdd_gate.go` deja de escribirlo; `verifyDependencyFromHandoff` se sustituye por la condición `LastRoleClosed` en rosters multirrol con puertas; se retiran `internal/kickoff/closure.go`, su test y `internal/sddstatus/handoff_dependency_test.go`, que se reescribe sobre la condición nueva. Se conserva el aviso de último rol.
5. **`handoff_policy` deprecado** en `internal/kickoff` (`types.go` con `omitempty`, `schema.go`, `args.go`, `infer.go`), `internal/cli/sdd_kickoff.go` (ayuda, texto y JSON), `internal/sddstatus/status_v2.go` (JSON con `omitempty`) y el mensaje de `sdd_gate.go:89`.
6. **Parada ODD** en `internal/components/agentguidance/routing.go`: tras el paso 5, antes de la primera escritura de código, con las tres salidas; la salida 2 = plan aprobado, commit de `odd/tasks/<feature>.md` en la rama ODD, presentación del `prTarget` y de la orden de PR, y la implementación la hace otra persona. ODD no tiene ledger: la cesión queda en el propio documento de feature (forma en design). La reanudación de ODD por otra persona parte del fichero y tolera la ausencia del espejo de Engram.
7. **Texto de las tres salidas, una sola fuente** para ODD y SDD, referenciado desde `internal/assets/skills/_shared/sdd-orchestrator-sections.md` (sección de puertas) y proyectado a los orquestadores de todos los runtimes; retirada de `--handoff-policy` de ese texto, de `sdd-status-contract.md` y de `openspec-convention.md`.
8. **Skills.** Retirada de la «Pregunta de Transición» de `sdd-spec/SKILL.md` y `sdd-design/SKILL.md`; ajustes en los skills que citan `handoff.md` como requisito de `verify`.
9. **Fusiones intermedias** (D4): ajuste del cálculo de «pendiente de archivar» y del criterio de base fusionada de `ensure --archive` en el motor de ramas.
10. **Specs** de §4, **copias versionadas** (`testdata/golden/`, `.codex/AGENTS.md`, `AGENTS.md`, `GEMINI.md`) y **manual** (`docs/manual-de-inicio.md`).

### 3.2 Fuera de alcance (Out of Scope)

- Push, PR o merge desde Axiom (invariante: `internal/branch/planner.go:201`).
- Retirar `internal/handoff`, `axiom handoff`, el dashboard o la TUI de relevos (D5), y cambiar la spec `structured-handoffs`.
- Eliminar el campo `handoff_policy` del esquema (D6: cambio posterior).
- Paradas en `continuous`, tras `proposal` o `explore`, o al cierre de cada unidad de trabajo ODD (D2).
- Cambiar la puerta `integration` (D2) o el ritmo de sesión `interactive` del orquestador (`sdd-orchestrator-workflow.md:68-73`).
- Una tercera decisión en el vocabulario del ledger (D1).
- Editar `openspec/INDEX.md` a mano (lo regenera `axiom archive sync`).

### 3.3 Qué no cambia

El vocabulario del ledger (`approved|rejected`) y su lectura por binarios anteriores (el campo opcional nuevo lo ignoran); la reapertura por digest de un rechazo y el carácter terminal de una aprobación (`internal/kickoff/machine.go:198-212`); REQ-21.7 (`continuous` sin paradas); RDD; el motor `axiom branch` salvo D4; el esquema de `handoff.md`.

## 4. Capacidades (Capabilities)

> Contrato con `sdd-spec`. Baseline: `openspec/specs/` (este repositorio no declara `baseline:`). Los requisitos nuevos usan la siguiente serie libre (no existe ninguna REQ-26.x en `openspec/`); la spec la fija.

### 4.1 Nuevas capacidades (New Capabilities)

- Ninguna. Las tres salidas se especifican dentro de `sdd-block-review-gates` (SDD) y `organic-agent-trigger-rules` (ODD).

### 4.2 Capacidades modificadas (Modified Capabilities)

- `sdd-block-review-gates`: REQ-21.8 a 21.11 pasan de «aprobar o rechazar» a tres salidas; REQ-21.12 queda como la salida 1; requisito nuevo con la semántica de la salida 2 (`approved` con marca opcional de fase siguiente cedida, compatible con binarios anteriores, comprobación de artefactos confirmados, `prTarget` y orden de PR, «fusionada en integración», `multirepo`, no disponible con `engram`/`none`, cesión visible en `axiom sdd status`; en `role-apply:<rol>` cede el siguiente rol o `verify`); REQ-21.7 sin cambio; la puerta `integration` conserva dos salidas.
- `sdd-role-closure-handoff`: se retira REQ-21.14; se reescribe REQ-21.15 («`verify` global tras todos los `role-apply` aprobados»); REQ-21.13 conserva el aviso sin anunciar ni generar relevo.
- `sdd-preflight-configuration`: REQ-21.5 y propósito sellan solo la modalidad y los roles; `--handoff-policy` deprecado, aceptado e ignorado.
- `sdd-lane-kickoff`: REQ-21.2 admite la parada única de ODD, que no es una pregunta de gobernanza; REQ-21.4 retira las menciones a la política de relevos y a «sin `handoff.md` intermedios» en el retro-sello.
- `change-branch-lifecycle`: REQ-25.20 y S-16 distinguen una fusión intermedia de planificación de un cambio integrado (D4).
- `organic-agent-trigger-rules`: requisito nuevo con la parada ODD tras el plan y sus tres salidas; ajuste del requisito del protocolo ODD por defecto donde cite «sin más preguntas».
- `sdd-orchestrator-assets`: solo si la spec fija el texto proyectado de las puertas; la fase de spec lo confirma.

## 5. Enfoque (Approach)

- **Ampliar el ledger sin romperlo.** Salidas 1 y 3 son los registros que ya existen; la salida 2 es un `approved` con un campo opcional (`omitempty`). Los binarios anteriores lo ignoran porque el decodificador no es estricto (`internal/kickoff/schema.go:110`). No hay migración.
- **Un sobre, N opciones.** Generalizar la validación del núcleo del sobre por número de opciones y no por un literal `2`; cada sobre declara su aridad. El de autoridad de edición sigue en 2. Las tres opciones de la puerta comparten `axiom sdd gate record`, así que el prefijo común se sigue validando.
- **Salida 2 sin acción remota.** `gate record` con el flag lee `axiom branch status` y el estado del árbol, registra la aprobación y devuelve órdenes para el humano. Coherente con el contrato de autorización remota.
- **Condición de `verify` ya existente.** `LastRoleClosed` sustituye a la lectura de `handoff.md`; no hace falta lógica nueva.
- **Deprecación tolerante.** `handoff_policy` con `omitempty`, `Validate` que acepta vacío y los valores antiguos, y flag aceptado sin efecto.
- **Doctrina en un punto.** El texto de las tres salidas se define una vez y se proyecta; los goldens se regeneran al final con `-update`.

## 6. Áreas afectadas (Affected Areas)

| Área | Impacto | Qué cambia |
|---|---|---|
| `internal/consentenvelope/envelope.go` | Modificado | Validación de 2 o 3 opciones |
| `internal/sddstatus/` (`governance.go`, `status.go`, `status_v2.go`) | Modificado | Sobre de tres opciones; cesión de la fase siguiente visible en status; `verify` por `LastRoleClosed`; sin lectura de `handoff_policy`; JSON con `omitempty` |
| `internal/kickoff/` (`types.go`, `schema.go`, `args.go`, `infer.go`, `closure.go`) | Modificado y retirado | Campo opcional en `GateRecord` y flag de `gate record`; `handoff_policy` deprecado; retirada de `IntegrationHandoff` |
| `internal/cli/` (`sdd_gate.go`, `sdd_kickoff.go`) | Modificado | Salida 2 en `gate record` (comprobación de artefactos confirmados, `prTarget`, orden de PR); sin escritura de `handoff.md`; aviso conservado; ayuda y salida sin la política |
| `internal/branch/` (cálculo de pendiente de archivar y base de `--archive`) | Modificado | D4 |
| `internal/components/agentguidance/routing.go` y `routing_test.go` | Modificado | Parada ODD y sus pines |
| `internal/assets/skills/_shared/` (`sdd-orchestrator-sections.md`, `sdd-status-contract.md`, `openspec-convention.md`), `sdd-spec/SKILL.md`, `sdd-design/SKILL.md`, `internal/assets/assets_test.go` | Modificado | Tres salidas; sin `--handoff-policy` ni Pregunta de Transición |
| `internal/branch/engine/writer_inventory_test.go` | Modificado | Inventario de escritores sin el de `handoff.md` de integración |
| `testdata/golden/`, `.codex/AGENTS.md`, `AGENTS.md`, `GEMINI.md`, `docs/manual-de-inicio.md` | Regenerado o modificado | Copias versionadas y manual |
| `openspec/specs/` (capacidades de §4) | Modificado | Deltas |

## 7. Riesgos (Risks)

| Riesgo | Prob. | Mitigación |
|---|---|---|
| Solapamiento con `fix/bloqueos-prueba-multirepo-r2`, que tocará `gateQuestion`, `kickoff seal`, la plantilla de proposal y `apply-progress`/`verify-report` por rol (`governance.go`, `status.go`, `sdd_kickoff.go`, `args.go`, textos compartidos y goldens). Hoy no tiene commits. | Alta | Si se fusiona antes, este cambio se rebasa sobre ella; los goldens se regeneran al final, nunca se concilian a mano. |
| Generalizar `ValidateCompleteness` debilita el sobre de autoridad de edición o el contrato de *Lossless Blocking Prompt*. | Media | Aridad declarada por sobre; prueba de no regresión de ese sobre y de `internal/assets/blocking_prompt_contract_test.go`. |
| La aprobación es terminal (`internal/kickoff/machine.go:198-212`, D-08): si quien recibe discrepa del artefacto aprobado, un rechazo posterior no reabre la puerta y la remediación por digest solo reabre rechazos, no aprobaciones. | Media | Es el comportamiento actual de cualquier `approved` y la salida 2 no lo empeora; quien recibe corrige el artefacto dentro de su fase. Si se quiere una vía formal para reabrir, la decide el usuario (pregunta pendiente, no incluida en este alcance). |
| Un binario anterior no muestra la cesión: lee el registro como `approved` normal y su `status` propone seguir con la fase siguiente. | Baja | Degradación esperada y sin pérdida de datos; el campo se conserva en `gates.yaml` (solo anexar). |
| Las tres opciones con prefijo común deben distinguirse sin ambigüedad (dos de ellas son `--decision approved`). | Baja | Prueba del sobre con las tres opciones; el flag de la salida 2 es la diferencia. |
| Desfase binario/instrucciones: orquestadores sincronizados antes siguen pasando `--handoff-policy`. | Alta | D6: flag aceptado e ignorado. |
| Una fusión intermedia deja el cambio como «pendiente de archivar» o hace que `ensure --archive` lo acepte. | Alta sin D4 | D4, con pruebas en el motor de ramas. |
| Quien recoge el relevo trabaja sobre una rama sin fusionar. | Media | D3: la salida 2 exige PR fusionada y lo dice; `gate record` nombra el `prTarget`. |
| Almacén `engram`/`none`: la salida 2 no tiene qué subir. | Baja | La parada la marca como no disponible. |
| Reanudación ODD por otra persona sin espejo de Engram. | Media | La doctrina parte del fichero y tolera la ausencia del espejo. |
| Choque con REQ-21.2 y con `routing_test.go:473-508`. | Alta | Enmienda explícita de REQ-21.2 y de los pines en el mismo cambio. |
| Ruta de Pi para el texto de las puertas sin confirmar (`explore.md` §1.5). | Baja | Se confirma en design. |
| Tamaño muy por encima de 400 líneas. | Alta | D7: una sola PR con `size:exception`. |

Riesgo global: **medio**. No hay migración de datos y el vocabulario del ledger no cambia (solo gana un campo opcional); el riesgo se concentra en el solapamiento con la rama r2 y en el sobre compartido.

## 8. Entrega (Review Workload Forecast)

- Estimación de `explore.md` §4 (no medida): 1.500 a 2.500 líneas autoradas, con más deleciones que adiciones, sin contar goldens.
- Estrategia sellada en el pre-vuelo de la sesión: `single-pr` con `size:exception`.

```
Decision needed before apply: No
Chained PRs recommended: No
400-line budget risk: High
```

## 9. Plan de reversión (Rollback Plan)

- **Por código.** `git revert` de los commits del cambio (o de la PR fusionada). No hay migraciones.
- **Por datos.** `gates.yaml` no cambia de vocabulario: lo escrito con el binario nuevo lo lee el anterior, que ignora el campo opcional de cesión y trata esas puertas como `approved`; no hace falta limpiarlo. Los kickoffs nuevos llevan `handoff_policy: none`, que el binario anterior acepta. Los `handoff.md` ya generados se conservan y siguen siendo legibles por `axiom handoff`.
- **Por doctrina.** `axiom sync` reproyecta las instrucciones del binario revertido; el flag `--handoff-policy` vuelve a ser obligatorio, y las instrucciones revertidas ya lo pasan.
- **Señal para revertir.** Un sobre de autoridad de edición rechazado por la validación generalizada; un `verify` multirrol bloqueado con todos los `role-apply` aprobados, o desbloqueado sin ellos; un `axiom sdd kickoff seal` que falle con instrucciones antiguas; un `gates.yaml` con el campo de cesión que un binario anterior no consiga leer.

## 10. Dependencias (Dependencies)

- Coordinación con `fix/bloqueos-prueba-multirepo-r2` (§7): rebasar sobre ella si se fusiona antes.
- `axiom branch status --json` (`prTarget`) y `kickoff.LastRoleClosed`, ya existentes.
- Restricciones de trabajo: TDD desactivado (`strict_tdd: false`), RDD desactivado, sin compilar binarios sueltos (solo `go test` acotado por paquete), commits Conventional sin atribución de IA.

## 11. Criterios de éxito (Success Criteria)

- [ ] **Tres opciones.** En `checkpointed`, la `gateQuestion` de `spec`, `design`, `tasks` y `role-apply:<rol>` tiene exactamente tres opciones en este orden de salidas: iterar (`rejected`, exige motivo), aprobar y parar (`approved` + flag), aprobar y seguir (`approved`), las tres sobre `axiom sdd gate record`; la de `integration` sigue con dos.
- [ ] **Salida 2 aprueba y cede.** Tras la salida 2, `gates.yaml` contiene un registro `approved` con el campo opcional de cesión; la puerta queda `approved` y `axiom sdd status` indica que la fase siguiente (o, en `role-apply:<rol>`, el siguiente rol o `verify`) queda cedida a otra persona.
- [ ] **Compatibilidad del ledger.** Un `gates.yaml` con el campo opcional se decodifica con el `ParseGateLedger` actual sin error y la puerta resulta `approved` (prueba con el esquema previo al cambio).
- [ ] **Salida 2 informativa.** `gate record` con el flag devuelve rama, `prTarget` y orden de PR (o push in situ en el repositorio de specs de un `multirepo`); si los artefactos no están confirmados rechaza con continuación sin escribir en `gates.yaml`; no ejecuta git remoto (prueba); con almacén `engram` o `none` rechaza el flag como no disponible.
- [ ] **Salida 1 sin cambios.** Un rechazo con motivo se comporta como hoy: `status` enruta a la fase dueña y la remediación reabre la puerta por digest.
- [ ] **Sobre de autoridad intacto.** Las pruebas del sobre de autoridad de edición y del contrato de *Lossless Blocking Prompt* pasan sin cambios en sus expectativas; un sobre de autoridad con tres opciones se sigue rechazando.
- [ ] **`verify` sin `handoff.md`.** En un roster multirrol con puertas, `verify` está bloqueado mientras falte algún `role-apply` aprobado y se desbloquea cuando todos lo están, sin que exista `handoff.md`; aprobar el último rol emite el aviso y no escribe `handoff.md`.
- [ ] **`handoff_policy` tolerante.** `axiom sdd kickoff seal` sin `--handoff-policy` sella; con `--handoff-policy none|per_checkpoint` también, sin efecto; el `kickoff.yaml` nuevo lleva siempre `handoff_policy: none` (compatibilidad con binarios anteriores, R-1 del design); un `kickoff.yaml` existente con el campo se lee y valida sin error.
- [ ] **ODD.** La doctrina generada contiene una única parada tras crear el documento de feature y antes de la primera escritura de código, con las tres salidas, y su salida 2 aprueba el plan, lo confirma en la rama, presenta la PR y deja la implementación a otra persona; `routing_test.go` fija el texto nuevo y ya no exige «no further governance question» en términos que la contradigan.
- [ ] **Skills.** Ningún asset del producto ofrece «generar `handoff.md` o continuar» ni nombra `--handoff-policy` como obligatorio (prueba de pin).
- [ ] **Fusión intermedia.** Un cambio con solo artefactos de planificación fusionados no aparece como pendiente de archivar y `ensure --archive` sigue rechazando con `change_not_integrated` (o el código que fije el diseño).
- [ ] **Paridad y goldens.** Los goldens y `.codex/AGENTS.md` están regenerados y el texto de las tres salidas es idéntico en todos los runtimes.
- [ ] **CI verde.** `go test` de los paquetes tocados y `go vet ./...` pasan; los ratchets de refusals y deadcode no empeoran.
