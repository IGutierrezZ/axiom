# Diseño: Paradas con tres salidas en ODD y SDD (paradas-tres-salidas)

> Fase `design` · rama `inc/paradas-tres-salidas` · 2026-11-02.
> Entradas autoritativas: `proposal.md` (aprobada el 2026-11-02; D1–D7 y D-08 intactos), `explore.md` y las seis delta specs (`sdd-block-review-gates`, `organic-agent-trigger-rules`, `sdd-lane-kickoff`, `sdd-preflight-configuration`, `sdd-role-closure-handoff`, `change-branch-lifecycle`): REQ-26.1 a REQ-26.11 nuevos, REQ-21.x y REQ-25.x modificados, REQ-21.14 retirado.
> Kickoff sellado: `continuous`, rol único `fullstack`. Este cambio no tiene puertas propias.
> Idioma: castellano. Identificadores de código, rutas, flags y claves YAML/JSON en inglés.
> Las decisiones de este documento se numeran **DD-01…DD-14** para no confundirlas con las de producto (**D1…D7**) ni con la regla de reapertura **D-08**.
> Este documento fija el CÓMO a nivel de arquitectura. No contiene tareas.
> Contexto técnico mantenido: `axiom.yaml` no declara `technical_context` (ni `maintained` ni `reference`), así que **no hay `context-impact.md`** (paso 2b de la skill: sin contexto mantenido no se crea delta).

## 0. Resumen

El cambio se construye sobre siete piezas, todas sobre código que ya existe:

1. **Ledger ampliado sin vocabulario nuevo.** `GateRecord` gana un booleano opcional `handed_off` (`omitempty`). Solo lo escribe la salida 2. Las salidas 1 y 3 son los registros `rejected` y `approved` de siempre, byte a byte.
2. **Un sobre, N opciones declaradas por quien lo consume.** `consentenvelope.Core.ValidateCompleteness` pasa a recibir la lista exacta de respuestas (variádica). Los tres llamadores actuales siguen pasando dos tokens, de modo que el sobre de autoridad de edición y el de RDD siguen exigiendo exactamente dos opciones sin tocar su código. La pregunta de puerta declara tres (o dos, con almacén sin ficheros, o las dos de siempre en `integration`).
3. **`gate record --hand-off`.** Comprueba, en este orden: guarda de rama, kickoff sellado `checkpointed`, puerta distinta de `integration`, almacén con ficheros, puerta aún no aprobada, artefacto presente, y que la carpeta del cambio (y, en `role-apply`, el árbol de cada repositorio del rol) está confirmada. Solo entonces escribe `gates.yaml`. Después devuelve las órdenes de commit de `gates.yaml`, push y los datos de la PR, calculados por el motor de ramas sin ninguna operación remota.
4. **`status` muestra la cesión** como dato (`handedOff`, `handedOffTo`) y como nota informativa. La ruta no cambia.
5. **`verify` multirrol sin `handoff.md`:** se sustituye `verifyDependencyFromHandoff` por una regla sobre `kickoff.LastRoleClosed` que solo estrecha `ready` a `blocked`. Se retiran `writeIntegrationHandoff`, `kickoff.IntegrationHandoff` y `closure.go`.
6. **`handoff_policy` deprecado:** se acepta y se ignora, no se exige ni se escribe, y `Validate` admite el valor vacío.
7. **Fusión intermedia y texto único.**
   - El motor de ramas lee `gates.yaml` y `kickoff.yaml` del árbol de la integración para distinguir una fusión intermedia de un cambio integrado.
   - El texto de las tres salidas vive en un asset neutro, `internal/assets/generic/stop-exits.md`. Lo proyectan la doctrina ODD (`routing.go`) y la sección compartida de puertas SDD, a través de un marcador que resuelve el compositor de orquestadores.

### 0.1 Decisiones

| ID | Decisión | Alternativa descartada |
|---|---|---|
| DD-01 | Campo `GateRecord.HandedOff bool` (`yaml:"handed_off,omitempty"`), flag `--hand-off` y token `hand_off` | Enumerado `next: handoff`; campo de texto libre |
| DD-02 | Orden de `gate record --hand-off`: comprobar todo → escribir con una comprobación de nuevo bajo el cerrojo → devolver órdenes | Escribir primero y avisar después |
| DD-03 | Comprobación de «confirmado»: toda la carpeta del cambio salvo `gates.yaml` y los ficheros de estado con punto; en `role-apply`, además, el árbol entero de cada repositorio del rol | Solo los ficheros cuyo digest se registra |
| DD-04 | `ValidateCompleteness(answers ...string)`: cada sobre declara su aridad exacta | Literal `3`; un campo `Arity` en `Core`; un segundo validador |
| DD-05 | Pregunta de puerta: `[rejected, hand_off, approved]`; con almacén `engram`/`none`, `[rejected, approved]` más `handOffUnavailable`; `integration` sin cambios | Reordenar también `integration`; ocultar la salida 2 sin decirlo |
| DD-06 | Cesión en `status`: `gates[].handedOff` y `gates[].handedOffTo` (`omitempty`) más una nota | Campo nuevo de nivel raíz; cambiar la ruta |
| DD-07 | Entrega de la salida 2 calculada por `engine.HandOff` sobre el mismo camino de lectura que `Status` | Que la CLI lance `git` por su cuenta; invocar `axiom branch status` como subproceso |
| DD-08 | Sin orden de plataforma (`gh`, `glab`): push exacto y datos `head`/`base` de la PR | Detectar la plataforma por la URL del remoto |
| DD-09 | Fusión intermedia = `gates.yaml` integrado con algún `approved` + `handed_off` y no todos los `role-apply` del roster integrado aprobados | Marcador en un fichero aparte; «cualquier puerta pendiente» |
| DD-10 | `verifyDependencyFromRoleGates`: solo estrecha `ready` a `blocked` | Bloquear siempre que falte un rol (cambiaría la salida de `status` en `spec`/`design`) |
| DD-11 | `handoff_policy` deprecado: flag aceptado y validado, nunca escrito; `Validate` acepta `""` | Eliminar el campo (rompe orquestadores ya sincronizados) |
| DD-12 | Fuente única `generic/stop-exits.md` + marcador `{{AXIOM_STOP_EXITS}}` en la sección compartida de puertas | Sección en `sdd-orchestrator-sections.md` leída por `agentguidance`; copiar el texto en doce plantillas |
| DD-13 | Parada ODD: línea `Plan stop:` en el documento de feature; commit solo de `odd/tasks/<feature>.md` | Ledger ODD; parada a petición |
| DD-14 | Cambios nuevos en ficheros nuevos y ediciones mínimas en los ficheros que también toca la rama r2 | Refactorizar `governance.go` y `status.go` de paso |

## 1. Enfoque técnico

Las tres salidas se apoyan en el modelo de puertas que ya existe, sin crear otro:

| Salida | Registro | Código que cambia |
|---|---|---|
| 1. Iterar | `rejected` con `--reason` (sin cambios; REQ-21.12) | Ninguno; solo el orden de la opción en el sobre |
| 2. Aprobar, subir y parar | `approved` + `handed_off: true` | `kickoff` (campo y lectura), `cli` (comprobaciones y entrega), `branch/engine` (entrega y fusión intermedia), `sddstatus` (sobre y cesión) |
| 3. Aprobar y seguir | `approved` (sin cambios) | Ninguno |

La aprobación sigue siendo terminal (`internal/kickoff/machine.go:198-203`: el primer `approved` cierra la puerta) y la reapertura por digest no cambia (`machine.go:204-211`). Axiom no ejecuta `push`, `fetch`, PR ni commit (`internal/branch/ports.go:7-9`: el puerto Git no tiene método que publique; `internal/branch/gitcli/ops.go:1-10`): la salida 2 solo calcula y presenta.

La retirada del `handoff.md` del flujo SDD reutiliza `kickoff.LastRoleClosed` (`machine.go:152-178`), que ya usan `sdd_gate.go:242-264` y `governance.go:574-583`. `internal/handoff`, `axiom handoff`, el dashboard y la TUI no se tocan (D5).

## 2. Decisiones de arquitectura

### 2.1 DD-01 — Nombre y forma de la cesión en el ledger, el flag y el token

**Elección.**

- `internal/kickoff/types.go`, `GateRecord`: campo nuevo `HandedOff bool \`yaml:"handed_off,omitempty"\``, situado tras `Verified`. Significa «esta aprobación cede la fase siguiente a otra persona».
- `GateState` (`types.go:169-175`): campo nuevo `HandedOff bool`, copiado del registro `approved` que cierra la puerta.
- `axiom sdd gate record`: flag booleano `--hand-off` (sin valor), en `kickoff.ParseGateRecordArgs` (`internal/kickoff/args.go:352-470`) y en `GateRecordArgs.HandOff bool`.
- Token de respuesta de la salida 2 en el sobre: `hand_off` (constante `gateAnswerHandOff` junto a `gateAnswerApproved`/`gateAnswerRejected` en `internal/sddstatus/governance.go:42-43`).
- Validación de forma en `ParseGateRecordArgs` (pura, antes de tocar disco):
  - `--hand-off` sin `--decision approved` se rechaza con el mensaje «`--hand-off` solo acompaña a `--decision approved`».
  - `--hand-off` con `--gate integration` se rechaza como no disponible para esa puerta (REQ-26.1, segundo escenario).
  - `--hand-off` con `--evidence-kind` se rechaza: la evidencia de fusión es solo de `integration`.
  - `--reason` es opcional con `--hand-off` (REQ-26.3).

**Alternativas descartadas.**

- Un enumerado `next: handoff` (sugerido como ejemplo en la propuesta). Un enumerado de un solo valor no añade información, y un valor desconocido en un binario futuro se leería como texto libre, sin validar.
- Un texto libre con la persona destinataria. Invita a escribir datos personales en un fichero versionado y no hace falta para el comportamiento.

**Por qué.** `yaml.v3` con `omitempty` omite el `false`: los registros de las salidas 1 y 3 son idénticos a los de hoy (REQ-26.1, escenario «Aprobar y seguir no cede nada»). `ParseGateLedger` usa `yaml.Unmarshal` sin `KnownFields` (`internal/kickoff/schema.go:108-117`), así que un binario anterior ignora la clave y lee `approved` (REQ-26.3). El vocabulario cerrado `approved|rejected` (`schema.go:48-51`, `:143-145`) no cambia.

### 2.2 DD-02 — Orden de `gate record --hand-off` (REQ-26.3 a REQ-26.6)

**Elección.** `runSDDGateRecord` (`internal/cli/sdd_gate.go:56-151`) conserva su recorrido. Las comprobaciones de la salida 2 viven en un fichero nuevo, `internal/cli/sdd_gate_handoff.go`, que el recorrido llama en dos puntos. El orden es:

| # | Paso | Si falla | Escribe |
|---|---|---|---|
| 1 | `ParseGateRecordArgs` (forma, DD-01) | error de uso | nada |
| 2 | `resolveGovernanceChangeRoot` (`sdd_governance.go:34-54`) | rechazo actual | nada |
| 3 | Guarda A `guardChangeWriter` (`sdd_gate.go:69-75`): `wrong_branch` antes que cualquier otra comprobación (REQ-26.4, último escenario) | `*branch.RefusalError` actual | nada |
| 4 | `kickoff.Load`. Con `--hand-off`: kickoff sellado y `execution_style: checkpointed` | «el cambio no tiene paradas» con continuación `--decision approved` sin `--hand-off` (REQ-26.1) | nada |
| 5 | Pertenencia al roster en `role-apply` (`sdd_gate.go:87-96`) | rechazo actual | nada |
| 6 | Almacén: `workspace.DeclaredArtifactStore(loc.WorkspaceRoot)`. Con `engram` o `none`, rechazo que nombra el almacén y ofrece las salidas 1 y 3 (REQ-26.6). Sin declaración cuenta como `openspec`: la carpeta del cambio existe en disco, que es la condición del paso 2 | rechazo con continuación | nada |
| 7 | Digest (`gateArtifactDigest`, `sdd_gate_digest.go`). Con `--hand-off`, un digest vacío (artefacto inexistente o vacío) se rechaza: no se cede una fase cuyo artefacto no existe | rechazo con continuación «produce el artefacto» | nada |
| 8 | Puerta ya aprobada: `kickoff.LoadGates` + `ledger.Approval(gate)` (DD-01, §4.1). Si ya hay un `approved`, rechazo «la puerta ya está resuelta» (A-1) | rechazo con continuación `axiom sdd status` | nada |
| 9 | Artefactos confirmados y datos de entrega: `engine.Default().HandOff(...)` (DD-03, DD-07) | `uncommitted_artifacts` con las rutas y la continuación de commit | nada |
| 10 | `kickoff.AppendGateChecked(changeRoot, record, notApproved(gate))`: vuelve a comprobar el paso 8 **bajo el cerrojo** del ledger (`ledger.go:77-126`) | mismo rechazo del paso 8 | nada |
| 11 | Salida: registro, cesión (`handedOffTo`), órdenes de entrega y aviso de fusión previa; en `role-apply`, aviso de último rol si procede | — | `gates.yaml` (ya escrito) |

Sin `--hand-off` el recorrido es el de hoy, salvo dos cosas: desaparece la escritura de `handoff.md` (DD-10) y `--json` deja de ignorarse (§4.4).

**Alternativas descartadas.**

- Escribir y avisar después. Deja un `approved` con cesión sobre artefactos que quien recibe no podrá obtener, y contradice «no escribe nada» (REQ-26.4).
- Comprobar el paso 8 solo fuera del cerrojo. Dos sesiones concurrentes (salida 2 y salida 3) podrían escribir ambas, y la cesión quedaría invisible tras el primer `approved`.

**Por qué.** Todas las comprobaciones son lecturas y van antes de `AppendGate`, el único escritor de `gates.yaml` (`ledger.go:73-76`), siguiendo el patrón que ya aplica la evidencia de `integration` (`sdd_gate.go:114-126`: «verification runs, and can refuse, BEFORE kickoff.AppendGate»). Cada rechazo lleva continuación ejecutable, como exige el ratchet de rechazos.

### 2.3 DD-03 — Qué significa «confirmado» (REQ-26.4, A-2)

**Elección.**

- **Puertas `spec`, `design` y `tasks`.** En el repositorio que contiene la carpeta del cambio, `DirtyPaths` (`gitcli/ops.go:140-171`: modificados, preparados y sin seguimiento) filtrado al prefijo `<planning home relativo>/changes/<cambio>/` no debe devolver nada. Quedan exentos:
  - `gates.yaml`, que la orden modifica y se confirma después (REQ-26.4);
  - las entradas de la raíz de la carpeta cuyo nombre empieza por punto. Son estado local que nunca se versiona: `.gates.lock` (`ledger.go:18`), `.axiom-instance` (`internal/sddstatus/edit_authority_consent.go:30`) y su variante heredada, y los temporales `.gates-*.yaml`/`.kickoff-*.yaml`. Ningún cambio archivado los tiene versionados (`git ls-files openspec/changes/archive`).

  Una carpeta del cambio entera sin seguimiento aparece como una sola entrada con «/» final que cubre el prefijo y se rechaza.
- **Puerta `role-apply:<rol>`.** Lo anterior más el **árbol de trabajo entero** de cada repositorio del plan cuyo `Roles` contiene el rol (`branch.RepositoryResult.Roles`, `internal/branch/result.go:173-185`), con las mismas exenciones en el repositorio del planning home. En `monorepo-embedded` es el propio repositorio; en `multirepo`, los repositorios de código del rol.

**Alternativas descartadas.**

- **Solo los ficheros cuyo digest se registra** (lectura literal de REQ-26.4). Quien recibe necesita además `kickoff.yaml` (sin él, su `status` no tiene gobierno y `gates.yaml` queda huérfano), `proposal.md` y, desde `design`, las specs. Ceder `design` con una edición sin confirmar en `proposal.md` haría llegar una propuesta antigua a la otra persona.
- **En `role-apply`, solo los ficheros que tocan las tareas del rol.** Axiom no puede atribuir un fichero sucio a un rol concreto. El worktree es por cambio (REQ-25.8), así que el árbol entero es el criterio honesto.

**Por qué.** Es la condición mínima para que «subir» funcione: lo que no está en un commit no viaja en la PR. **Desviación declarada:** este criterio es más estricto que la letra de REQ-26.4 («los artefactos que juzga la puerta»), aunque cumple todos sus escenarios. Ver §11, riesgo R-3.

### 2.4 DD-04 — `ValidateCompleteness` para 2 o 3 opciones

**Elección.** En `internal/consentenvelope/envelope.go:57-78`, la firma pasa a `func (core Core) ValidateCompleteness(answers ...string) error`:

- exige `len(answers) >= 2` y respuestas distintas entre sí (error de construcción en caso contrario);
- exige `len(core.Choices) == len(answers)` y `core.Choices[i].Answer == answers[i]` para todo `i`;
- el resto (titular, motivo, valor, evidencia no nil, etiquetas, efectos, invocaciones y vía alternativa) no cambia.

Los tres llamadores actuales compilan sin tocarse y siguen exigiendo exactamente dos opciones:

| Llamador | Respuestas | Cambio |
|---|---|---|
| `internal/sddstatus/consent_contract.go:119` (autoridad de edición) | `granted`, `declined` | ninguno |
| `internal/cli/review_consent_contract.go:347` (RDD) | `granted`, `declined` | ninguno |
| `internal/sddstatus/governance.go:128` (puerta) | según DD-05 | sí |

La comprobación propia de cada consumidor se queda en el consumidor. La pregunta de puerta añade dos reglas a su `Validate` (`governance.go:107-141`):

- todas las invocaciones son distintas entre sí y empiezan por `axiom sdd gate record ` (se conserva el prefijo compartido);
- solo la opción `hand_off` contiene ` --hand-off`.

**Alternativas descartadas.**

- Un literal `3`. Rompe la autoridad de edición.
- Un campo `Arity` en `Core`. Cada consumidor podría declararlo mal, y no sustituye a nombrar los tokens.
- Un segundo validador `ValidateCompletenessN`. Duplica la lógica de completitud.

**Por qué.** La aridad queda en la propia llamada de cada sobre: el de autoridad de edición no puede aceptar tres opciones salvo que alguien edite su línea, y su prueba de no regresión lo fija. La prueba `envelope_test.go` («three choices» con `("granted", "declined")`) sigue rechazando sin cambiar su expectativa. El paquete actualiza su comentario de cabecera: «exactly two token choices» pasa a «the exact token choices each consumer declares».

### 2.5 DD-05 — La pregunta tipada de puerta (`newGovernanceGateQuestion`, REQ-26.2 y REQ-26.6, A-3)

**Elección.**

- `SDDGovernanceGateResult` (`governance.go:74-101`) gana el campo `HandOffUnavailable string \`json:"handOffUnavailable,omitempty"\``. Vacío significa que la salida 2 está disponible; con valor, es el almacén declarado que la impide (`engram` o `none`).
- `newGovernanceGateQuestion` recibe un parámetro nuevo, `store ArtifactStore`. Su único llamador, `status.go:754`, le pasa el almacén efectivo: el declarado (`workspace.DeclaredArtifactStore`) o, sin declaración, `status.ArtifactStore`. Es la misma regla que `effectiveArtifactStore` (`workspace_declarations.go:192-205`).

| Puerta | Almacén | Opciones (orden) | Invocaciones (`axiom sdd gate record --cwd … --change … --gate <g>` +) |
|---|---|---|---|
| `spec`, `design`, `tasks`, `role-apply:<rol>` | `openspec`, `hybrid` | `rejected`, `hand_off`, `approved` | `--decision rejected --reason <your-reason>` · `--decision approved --hand-off --reason <your-reason>` · `--decision approved --reason <your-reason>` |
| ídem | `engram`, `none` | `rejected`, `approved` + `handOffUnavailable` | las dos de iterar y seguir |
| `integration` | cualquiera | `approved`, `rejected` (como hoy) | como hoy |

- Etiquetas y efectos en inglés, como los de hoy: «Iterate on this artifact», «Approve, push and stop», «Approve and continue».
- `Headline`, `Reason` y `Value` describen las tres salidas. Con `HandOffUnavailable`, `Value` declara que «approve, push and stop» no está disponible por el almacén declarado.
- `Validate` deriva la lista esperada de la puerta y de `HandOffUnavailable`:
  - `integration` exige `[approved, rejected]` y `HandOffUnavailable == ""`;
  - el resto exige `[rejected, hand_off, approved]` o, con `HandOffUnavailable` en `{engram, none}`, `[rejected, approved]`.

  Un sobre de puerta de planificación con dos opciones y sin `handOffUnavailable` se rechaza (REQ-26.2, segundo escenario).

**Corrección de A-3.** Con `engram` la pregunta **no llega a construirse**: la ruta de status de Engram pasa `nil` como gobierno siempre (`status.go:847-853`), así que no hay puertas que proyectar. La variante de dos opciones solo aparece con `none` declarado y el cambio en disco, porque el resolutor informa `openspec` aunque se declare `none` (`workspace_declarations.go:192-195`). `gate record --hand-off` sí puede invocarse con `engram` declarado (solo necesita la carpeta en disco) y se rechaza en el paso 6 de DD-02.

**Alternativas descartadas.**

- Reordenar también `integration` a `[rejected, approved]`. Cambia un contrato que D2 deja intacto.
- Omitir la salida 2 sin decirlo. Incumple REQ-26.6, y quien relata el sobre no sabría explicar la ausencia.

**Por qué.** El dominio de respuesta queda formado por los tokens que el propio sobre declara, y el relato sin pérdidas (*Lossless Blocking Prompt*) no depende de prosa fuera del sobre.

### 2.6 DD-06 — Cesión visible en `axiom sdd status` (REQ-26.7)

**Elección.**

- **Dominio.** `resolveGateStatus` (`machine.go:198-212`) devuelve también el `HandedOff` del primer registro `approved`. `EvaluateGates` lo copia a `GateState.HandedOff`. La marca es la del registro que cierra la puerta (REQ-26.7, último escenario). Un `approved` posterior sin marca no la borra, y uno posterior con marca no la añade.
- **Función pura nueva** `kickoff.HandedOffTo(gate GateState, roster []multirole.RoleAssignment, states []GateState) string`:
  - en puertas fijas devuelve `gate.Blocks` (`design`, `tasks`, `apply`);
  - en `role-apply:<rol>` devuelve `verify` si `LastRoleClosed(roster, states)` y, si no, `apply:<r1>,<r2>` con los roles del roster sin `role-apply` aprobado, en orden alfabético.
- **JSON v2** (`internal/sddstatus/status_v2.go:130-136`): `gateV2` gana `HandedOff bool \`json:"handedOff,omitempty"\`` y `HandedOffTo string \`json:"handedOffTo,omitempty"\``.
- **Texto.** Una nota en `status.Notes` (bloque «Informational only», `status.go:1177-1182`) por cada puerta con cesión, en castellano: «La compuerta "spec" se aprobó cediendo `design` a otra persona: quien recoge el cambio empieza esa fase sin revisar la anterior.» La construye `handedOffNotes(governance)` en el fichero nuevo `internal/sddstatus/governance_handoff.go`. `status.go` añade una línea tras `loadGovernance`.

La ruta (`resolveNextRecommended`, `status.go:1723`) no cambia: una puerta `approved` es `approved` con o sin marca.

**Alternativas descartadas.**

- Un campo raíz `handOff`. Duplica `governance` y obliga a elegir una sola puerta.
- Enrutar a `await-handoff`. La cesión no bloquea (REQ-26.7, «La cesión no bloquea»).

**Por qué.** `omitempty` y la ausencia de nota dejan un cambio sin marca byte a byte igual que antes (REQ-26.7, «Sin marca no hay cesión»), y las notas ya son el canal que el contrato declara informativo.

### 2.7 DD-07 — Entrega de la salida 2 por topología: `engine.HandOff` (REQ-26.5, A-4)

**Elección.** Hay un método nuevo de solo lectura, `func (e *Engine) HandOff(ctx context.Context, req HandOffRequest) (branch.HandOffPlan, *branch.Refusal)`, en `internal/branch/engine/handoff.go`, construido sobre el mismo camino que `Status` (`gather` + `describe`, `engine/status.go:39-170`). No toma cerrojo, no escribe el registro y no ejecuta `Fetch`.

1. Resuelve el plan del cambio (carril SDD). Un rechazo del plan (`config_invalid` y similares) se devuelve tal cual, y la CLI no escribe nada.
2. Localiza el repositorio del planning home: el `RepositoryResult` cuyo `GitRoot` contiene `ChangeRoot`.
3. Aplica DD-03 con `DirtyPaths`.
4. Compone una `HandOffDelivery` por repositorio:

| Caso | Rama | Destino | Pasos que se presentan |
|---|---|---|---|
| `monorepo-embedded`, repositorio del planning home | `CurrentBranch` del repositorio (la guarda A ya garantizó que es la del cambio o una de sus ramas de rol) | `PRTarget` de esa `BranchResult`: la integración para la rama de cambio (`planner.go:389-394`) o la rama de seguimiento para una rama de rol (`planner.go:400-413`) | `git -C <gitRoot> add -- <rel>/gates.yaml`; `git -C <gitRoot> commit -m "<msg>" -- <rel>/gates.yaml`; `git -C <gitRoot> push -u origin <rama>`; PR `head=<rama>`, `base=<prTarget>` |
| `multirepo`, repositorio de specs (`RepoSpecsInPlace`, `planner.go:367-369`) | rama de integración | — (sin PR) | `add`/`commit` de `gates.yaml` en la integración; `git -C <gitRoot> push origin <integración>` |
| `role-apply` en otros repositorios del rol | rama de rol del rol si existe en el plan; si no, la de cambio | su `PRTarget` | `git -C <gitRoot> push -u origin <rama>`; PR `head`/`base` (sin `add`/`commit`: DD-03 ya exige el árbol limpio) |

- Las rutas se citan con `pathquote.Quote`.
- El mensaje de commit propuesto es Conventional y en castellano: `chore(sdd): aprobar <puerta> de <cambio> y ceder <handedOffTo>`.
- Todo resultado lleva `MergeRequired: true` y el texto «la fase siguiente solo empieza cuando la PR esté fusionada en `<prTarget>`: `axiom branch ensure` en otra máquina crea la rama desde la base y no recupera trabajo sin fusionar» (`internal/branch/engine/ensure.go:38-66`).

**Confirmación de A-4.** En una rama de rol el destino es la rama de seguimiento, que es lo que informa `status`, y la salida no afirma que eso integre en la rama de integración. Ver R-5 sobre quien recoge en otra máquina.

**Alternativas descartadas.**

- Que la CLI lance `git status` o `git rev-parse` por su cuenta. Saltaría el único punto que ejecuta git (`gitcommon.Run` y su observador, `internal/gitcommon/exec.go:31`), así como la política de nombres y el cálculo de `PRTarget`.
- Ejecutar `axiom branch status --json` como subproceso. Compilaría y lanzaría un binario (prohibido en este entorno) y añade un acoplamiento a la salida textual.

**Por qué.** `PRTarget`, la topología y las ramas ya son del motor (REQ-25.15). Reutilizarlo garantiza que lo que la salida 2 llama `prTarget` es lo mismo que enseña `axiom branch status`.

### 2.8 DD-08 — Orden de PR sin plataforma (A-5)

**Elección.** La salida entrega la orden de push exacta y los datos de la PR (`head`, `base`) en campos separados y en el texto. No emite `gh pr create` ni `glab mr create`.

**Alternativa descartada.** Detectar la plataforma por `git remote get-url`. Es una heurística sobre una URL que puede ser un espejo o SSH con alias, y el producto no abre PR por invariante (`docs/manual-de-inicio.md:230`; `internal/branch/planner.go:201`).

**Por qué.** Lo exacto y verificable es el push y el par `head`/`base`. Abrir la PR es una acción de la persona con su herramienta.

### 2.9 DD-09 — Criterio de fusión intermedia (REQ-26.11, REQ-25.20, REQ-25.45; A-7, A-8)

**Elección.**

- **Dominio** (`internal/kickoff/handed_off.go`, puro): `func IntermediateMerge(k Kickoff, l GateLedger) bool` devuelve `true` si **(a)** algún registro es `approved` y lleva `HandedOff`, y **(b)** no todos los roles de `k.Config.Roles` tienen un `approved` en `RoleApplyGate(rol)`. Con un roster vacío, (b) es cierto, igual que en `LastRoleClosed`, que nunca cierra con roster vacío.
- **Puerto Git**: método nuevo `FileAt(ctx, dir, rev, relPath string) (content []byte, exists bool, err error)` en `branch.Git` (`internal/branch/ports.go:13-31`) y en `gitReader` (`engine/engine.go:40-51`). Lo implementa `gitcli` con `ls-tree -z --full-tree <rev> -- <relPath>`, para saber si existe y obtener el objeto, seguido de `cat-file blob <objeto>`. Se añade `cat-file` a la lista cerrada de lecturas del comentario de paquete (`gitcli/ops.go:1-10`). Valida argumentos con `plainArgument` y `pathspecArgument`, como `TreeDirs` (`ops.go:212-245`).
- **Motor** (`internal/branch/engine/intermediate.go`): `func (e *Engine) intermediateMerge(ctx, root, sha, changesRel, change string) (bool, string)`:
  1. lee `<changesRel>/<change>/gates.yaml`; si no existe o `ParseGateLedger` falla, devuelve `false` (A-8);
  2. si ningún registro lleva `handed_off`, devuelve `false` sin leer más;
  3. lee `kickoff.yaml`; si no existe o `ParseKickoff` falla, devuelve `false` (A-8);
  4. devuelve `kickoff.IntermediateMerge(k, l)`.

  Un error de git (no de contenido) devuelve `false` y un aviso, que se añade a los avisos ya existentes.
- **Consumidores.**
  - `PendingArchive` (`engine/pending.go:44-89`) salta los cambios intermedios. El dashboard (`internal/dashboard/service.go:235`) y la TUI (`internal/tui/model.go:6015`) lo heredan sin cambios. No se añade la etiqueta opcional «en curso (fusión intermedia)» (REQ-26.11 usa PUEDE).
  - `checkIntegrated` (`engine/archive.go:54-89`): si la carpeta existe y el cambio es intermedio, devuelve el rechazo nuevo `branch.ChangeIntermediateMerge(...)` (`internal/branch/archive_refusal.go`). Lleva código `change_not_integrated`, detalle `intermediate: "true"`, un mensaje que lo dice, ninguna sugerencia de `--fetch` y la continuación «termina el cambio (todos los `role-apply` aprobados), fusiona su PR y repite `axiom branch ensure --change <c> --archive`».
  - El repositorio de specs de un `multirepo` sigue fuera (`pending.go:45-47`) y `--archive` sigue rechazando allí con `archive_branch_not_applicable`.

**Confirmaciones.**

- A-7: el criterio (b) deja de ser cierto en cuanto todos los `role-apply` están aprobados, aunque la última puerta lleve marca.
- A-8: un cambio sin ninguna marca, sin `gates.yaml` o con `gates.yaml`/`kickoff.yaml` ilegibles se comporta como hasta ahora. En modalidad continua no hay puertas, así que nunca hay marca.

**Alternativas descartadas.**

- Un marcador aparte (`.intermediate` o similar). Sería un artefacto más que versionar, y se olvidaría quitarlo al fusionar la PR final.
- «Alguna puerta pendiente». Un cambio continuo, o uno integrado con `integration` pendiente, dejaría de figurar como pendiente de archivar, y eso es justo lo que REQ-25.20 quiere enseñar.

**Por qué.** El criterio solo usa datos que la propia salida 2 deja integrados (`gates.yaml` y `kickoff.yaml`), se calcula sobre la referencia local sin red (REQ-25.10) y no inventa estado nuevo.

### 2.10 DD-10 — `verify` multirrol por `LastRoleClosed`; retirada del `handoff.md` de integración (REQ-21.13, REQ-21.15, REQ-21.14 retirado)

**Elección.**

- `verifyDependencyFromHandoff` (`governance.go:491-546`) se elimina. En su lugar, en `governance_handoff.go`:

  ```go
  func verifyDependencyFromRoleGates(governance *Governance, baseline DependencyState) (DependencyState, string)
  ```

  Devuelve `baseline, ""` si `!governanceGated(governance)`, si el roster tiene un rol o ninguno, si `baseline != DependencyReady` o si `lastRoleClosedForGovernance(governance)`. En cualquier otro caso devuelve `DependencyBlocked` y un motivo que nombra los roles sin `role-apply` aprobado (REQ-21.15). Se elimina el import de `internal/handoff` en `governance.go`.
- `status.go:730`: la llamada pasa a `verifyDependencyFromRoleGates(governance, dependencies.Verify)`.
- `sdd_gate.go`:
  - se eliminan `writeIntegrationHandoff` (`:266-287`) y `buildIntegrationArtifactInventory` (`:289-310`), y el import de `internal/handoff`;
  - `lastRoleJustClosed` devuelve solo `(bool, error)`;
  - se conserva `fileHasContent`, que usa `sdd_gate_digest.go`;
  - el aviso conserva `lastRoleNoticeMarker` (`:25`) y pasa a decir «Aviso: se ha concluido la implementacion del ultimo rol activo del cambio %q; queda abierta la verificacion global de la solucion (REQ-21.13, REQ-21.15).». Se emite también con `--hand-off` (REQ-21.13, cuarto escenario).
- Se eliminan `internal/kickoff/closure.go` y `closure_test.go`. Se reescribe `internal/sddstatus/handoff_dependency_test.go` como `role_gates_dependency_test.go`. En `internal/branch/engine/writer_inventory_test.go:113-114` se quita la entrada `writeIntegrationHandoff` y el motivo de `runSDDGateRecord` deja de citar el relevo de integración. `{"internal/handoff", "WriteFile"}` (`:64`) se conserva, porque `axiom handoff create` y el dashboard siguen escribiendo.
- `internal/kickoff/rdd_boundary_test.go:177` deja de permitir el import de `internal/handoff`: `kickoff` ya no lo necesita. Se ajusta también el comentario de `machine.go:157`.

**Alternativa descartada.** Bloquear `verify` siempre que falte un rol. Añadiría un motivo bloqueante a la salida de `status` de cualquier cambio multirrol en `spec`, `design` o `tasks`, que hoy no lo tiene, y cambiaría `blockedReasons`. Solo estrechar `ready` a `blocked` sigue el patrón de `archiveDependencyFromGovernance` (`governance.go:559-572`).

**Por qué.** Un `role-apply` solo se abre con cero tareas pendientes (`machine.go:113-116`), así que con todos los roles aprobados la base ya es `ready` y no hace falta subirla. El antiguo ascenso a `ready` por un `handoff.md` `ready` desaparece con su razón de ser. `per_checkpoint` deja de tener efecto (REQ-26.9).

### 2.11 DD-11 — Deprecación de `handoff_policy` (REQ-21.5, REQ-26.9; A-6)

**Elección.**

| Punto | Cambio |
|---|---|
| `kickoff/types.go:26-28,105` | `HandoffPolicy` con comentario `Deprecated:` y etiqueta `yaml:"handoff_policy,omitempty"`. Se conservan `HandoffNone` y `HandoffPerCheckpoint`: las usan fixtures de prueba y sirven para validar el flag |
| `kickoff/schema.go:89-91` | `if k.Config.HandoffPolicy != "" && !validHandoffPolicies[...]`: acepta vacío, `none` y `per_checkpoint`; rechaza el resto |
| `kickoff/args.go:197-207` | `--handoff-policy` sigue parseándose y validándose. Un valor desconocido se rechaza (**A-6 confirmado**: `args.go:203-205` ya lo hace). `SealArgs.HandoffPolicy` se conserva con `Deprecated:` y no se usa |
| `kickoff/args.go:257-259` | se elimina el «requiere --handoff-policy» |
| `kickoff/args.go:296` | `ToKickoff` deja de copiar el campo |
| `kickoff/infer.go:22,89` | el retro-sello deja de escribirlo y se actualiza el comentario |
| `cli/sdd_kickoff.go:36` | la ayuda deja de nombrar el flag |
| `cli/sdd_kickoff.go:131-132` | el texto solo añade `handoff_policy=<v> (obsoleto)` cuando el kickoff lo trae |
| `cli/sdd_kickoff.go:160` | `json:"handoffPolicy,omitempty"` |
| `sddstatus/status_v2.go:125` | `json:"handoffPolicy,omitempty"` |
| `cli/sdd_gate.go:89` | el ejemplo de sellado pierde `--handoff-policy none` |
| `sddstatus/governance.go:514` | el único lector con efecto desaparece con DD-10 |

Las fixtures que pasan `--handoff-policy` (`e2e/organicruntime/sdd_consent_wiring_e2e_test.go:72`, `cmd/axiom/handoff_branch_guard_test.go:55`, `bench/journeys_sdd.go:151`, `bench/journeys_issue_2891.go:36` y los `*_test.go` de `kickoff`, `cli`, `sddstatus` y `branch/engine`) **no se tocan**: son la evidencia viva de que un orquestador antiguo sigue sellando.

**Alternativas descartadas.**

- Eliminar el campo y el flag ya. `args.go:229-230` rechazaría el flag como desconocido y rompería el sellado de los orquestadores ya sincronizados (D6).
- Seguir escribiendo `handoff_policy: none`. Contradice REQ-26.9 («Un kickoff.yaml nuevo NO DEBE contener la clave»). Ver R-1.

### 2.12 DD-12 — Texto único de las tres salidas y su proyección (REQ-26.8, REQ-26.10)

**Elección.**

- **Fuente única**: `internal/assets/generic/stop-exits.md`, en inglés como el resto de la doctrina, sin encabezado propio. Empieza por un párrafo en negrita, **Stops and exits.**, y define:
  - qué es una parada;
  - las tres salidas en orden, con iterar sin límite;
  - «subir» = PR **fusionada** en `prTarget` antes de que nadie empiece la fase siguiente, con el motivo de `ensure`;
  - en el repositorio de specs de un `multirepo`, commit en la integración y push humano sin PR;
  - que `push`, `fetch`, PR y merge son del humano y nunca los ejecuta el agente;
  - que con almacén `engram` o `none` la salida 2 no está disponible;
  - que quien recoge empieza la fase siguiente sin revisar la anterior.

  No nombra `gateQuestion`, `gateCriteria` ni `gateEvidence` (`assets_test.go:2457-2461` lo prohíbe en la sección de puertas).
- **Lector único**: `agentguidance.StopExits() string` en un fichero nuevo, `internal/components/agentguidance/stop_exits.go`, con la constante `stopExitsAsset = "generic/stop-exits.md"`. Sigue el precedente de `remote_authorization.go:13-16`, que lee `generic/remote-authorization-contract.md`.
- **ODD** (`routing.go`): sección nueva `### Stops and exits` con `StopExits()`, tras *Branches and worktrees* y antes de *Receipt-driven development*. Ver DD-13 para el resto de la doctrina ODD.
- **SDD**: en `internal/assets/skills/_shared/sdd-orchestrator-sections.md`, sección *SDD Change Kickoff and Block Gates*:
  - **segundo párrafo**: desaparece `--handoff-policy none|per_checkpoint` del ejemplo de `kickoff seal`;
  - **tercer párrafo**: cada puerta salvo `integration` es una parada con las tres salidas, registradas con `axiom sdd gate record`: iterar = `--decision rejected --reason`; aprobar, subir y parar = `--decision approved --hand-off`, que rechaza sin escribir si la carpeta no está confirmada, registra y devuelve el commit de `gates.yaml`, el push y los datos de la PR para presentarlos y parar; aprobar y seguir = `--decision approved`. `integration` conserva aprobar con evidencia o rechazar. Se conservan literalmente «re-present the SAME gate afterward, never skip ahead to the next one» y el resto de pines;
  - **párrafo nuevo con una sola línea**, `{{AXIOM_STOP_EXITS}}`, al final de ese tercer párrafo y antes del párrafo del contexto mantenido.
- **Compositor**: `substituteSharedOrchestratorSections` (`internal/components/sdd/orchestrator.go:65-75`) sustituye, **después** de la pasada de secciones, el marcador `{{AXIOM_STOP_EXITS}}` por `agentguidance.StopExits()`. `sdd` ya importa `agentguidance` (`internal/components/sdd/inject.go:15`). Sirve a los dos caminos que componen secciones (`orchestrator.go:79-97` e `inject.go:1055`). El comentario «one pass» se amplía: el marcador de salidas es un token distinto y se resuelve en un segundo paso explícito; no es anidamiento. Un marcador sin resolver hace fallar las pruebas existentes (`orchestrator_shared_sections_test.go:246` comprueba que no queda `{{GENTLE_AI_SDD_SECTION:`; se añade la misma comprobación para `{{AXIOM_STOP_EXITS}}`).
- **Runtimes.** Las doce plantillas (claude, codex, opencode, que también sirve a kilocode, gemini, antigravity, windsurf, cursor, kimi, qwen, kiro, hermes y generic) ya incluyen la sección por su marcador. **Pi** usa `generic/sdd-orchestrator.md` (`sddOrchestratorAsset`, rama `default`, `orchestrator.go:317-347`), así que recibe el texto por la misma vía; la ruta de Pi pendiente de `explore.md` §1.5 queda confirmada. La doctrina ODD llega a todos por `InjectRouting` (`agentguidance/inject.go:92-110`).
- **Skills.**
  - `sdd-spec/SKILL.md:240-250` y `sdd-design/SKILL.md:183-193`: se retira la «Pregunta de Transición» y queda «Listo para la fase siguiente; en `checkpointed`, el orquestador presenta la parada de la puerta con sus tres salidas».
  - `sdd-spec/SKILL.md:269` y `sdd-design/SKILL.md:208`: «Antes de ceder el testigo en handoff.md…» pasa a «Antes de la salida 2 de la parada, los artefactos quedan confirmados en la rama del cambio; publicarla es decisión de la persona».
  - `sdd-status-contract.md:127-135` (kickoff sin `handoffPolicy`; gates con `handedOff?`/`handedOffTo?`; `gateQuestion` con tres opciones o dos más `handOffUnavailable`; `verify` multirrol por `role-apply`) y `openspec-convention.md:26,47`: la fila de `handoff.md` pasa a «legado: lo escribían versiones anteriores; hoy solo `axiom handoff create`».
- **Copias versionadas.**
  - Se regeneran al final: los goldens con `-update` (`internal/components/golden_test.go:32`) y `.codex/AGENTS.md`.
  - Se editan a mano: `AGENTS.md:16`, `GEMINI.md:47` y `docs/manual-de-inicio.md:230`, sobre las paradas, y `:515-566`, que pasa a describir `handoff.md` como utilidad general fuera del flujo SDD.

**Alternativas descartadas.**

- Poner el texto como sección en `sdd-orchestrator-sections.md` y leerlo desde `agentguidance`. El enrutamiento es incondicional y no debe depender de un asset de SDD (comentario de paquete, `routing.go:1-5`).
- Añadir un marcador nuevo a las doce plantillas. Son doce ediciones y doce sitios donde olvidarlo.
- Copiar el texto. Es la deriva que `orchestrator.go:14-19` (#3817) ya sufrió.

**Por qué.** Un único fichero y un único lector garantizan la igualdad byte a byte que exige REQ-26.8, y un pin puede comprobar `strings.Contains(rendered, StopExits())` en cada runtime y en la doctrina ODD.

### 2.13 DD-13 — Parada ODD en la doctrina generada (REQ-26.10, REQ-21.2, REQ-25.40)

**Elección.** Cambios en `internal/components/agentguidance/routing.go`:

- **Paso 0** (`:54`): se conserva literalmente «do not ask any further governance question» (pin `routing_test.go:496`) y se añade al final «The single plan stop after step 5 (see *Stops and exits*) is not a governance question.».
- **Paso 5** (`:61`): se añade «Then present the plan stop: the single ODD stop, after the feature document exists and before the first code write, with the three exits of *Stops and exits*.». No nombra `axiom branch`, porque el pin de `routing_test.go:364` prohíbe que lo haga cualquier paso distinto del 6.
- **Línea de reanudación** (`:64`): se añade «When the Engram mirror does not exist (for example, the feature was handed off to you), start from the task file alone, never fail for the missing mirror, never review an approved plan again, and continue with the first unfinished task.».
- **Viñeta nueva** en *Organic Driven Development* que concreta la parada:
  - solo existe con documento de feature: el trabajo pequeño no tiene parada;
  - no es pregunta de gobernanza, no crea artefactos SDD ni `handoff.md` y no se repite al reanudar si el documento ya la registra;
  - el documento registra la decisión en una línea `Plan stop: approved <YYYY-MM-DD>; implementation continues` o `Plan stop: approved <YYYY-MM-DD>; implementation handed off`;
  - iterar ajusta fichero y espejo y vuelve a presentar la parada;
  - la salida 2 confirma **solo** `odd/tasks/<feature-name>.md` (`git -C <path> add -- odd/tasks/<feature-name>.md` y `git -C <path> commit -m "<msg>" -- odd/tasks/<feature-name>.md`), lee `axiom branch status --lane odd --change <feature-name> --json` (el flag `--lane` existe, `internal/cli/branch.go:282`), presenta la rama, el `prTarget`, el push y los datos de la PR, y se detiene sin escribir código;
  - en `multirepo`, commit en la rama de integración del repositorio de specs y push humano, sin PR.
- **Pines** (`routing_test.go`): se conservan los de los pasos 0 y 6 y se añaden pruebas de la sección *Stops and exits* (igual a `StopExits()`), del paso 5, de la línea `Plan stop:`, de la tolerancia al espejo ausente, de la ausencia de órdenes remotas y del determinismo.

**Alternativa descartada.** Un ledger ODD o un `gate record --lane odd`. ODD no tiene CLI (`openspec/specs/odd-cli-commands`, REMOVED), y el documento de feature ya es el registro autoritativo (D2).

**Por qué.** La parada queda como una excepción documentada en el mismo texto que la regla «sin preguntas de gobernanza», y se puede comprobar con un pin.

### 2.14 DD-14 — Coordinación con `fix/bloqueos-prueba-multirepo-r2`

**Estado comprobado (solo lectura, 2026-11-02).** `git log --oneline main..fix/bloqueos-prueba-multirepo-r2` no devuelve nada: la rama apunta al mismo commit que `main` (`512ee80f`) y su worktree `C:\repos\axiom-wt\bloqueos-prueba-multirepo-r2\axiom` no tiene cambios sin confirmar. **Hoy no hay solapamiento real.**

**Elección.** Para reducir la superficie de conflicto futura:

- **Código nuevo en ficheros nuevos:**
  - `internal/cli/sdd_gate_handoff.go`;
  - `internal/sddstatus/governance_handoff.go` (con `verifyDependencyFromRoleGates`, `handedOffNotes` y el cálculo del almacén efectivo para la pregunta);
  - `internal/kickoff/handed_off.go`;
  - `internal/branch/handoff.go`;
  - `internal/branch/engine/handoff.go` y `intermediate.go`;
  - `internal/components/agentguidance/stop_exits.go`;
  - `internal/assets/generic/stop-exits.md`.
- **Ediciones mínimas** en los ficheros que la r2 anuncia tocar:
  - `status.go`: tres líneas (la llamada de `verify`, el almacén de la pregunta y las notas);
  - `governance.go`: la pregunta y su `Validate` en el sitio, y el borrado de `verifyDependencyFromHandoff`, sin mover funciones;
  - `sdd_kickoff.go` y `args.go`: trozos puntuales del flag.
- En la sección compartida, el texto nuevo va en el tercer párrafo y en la línea del marcador, sin reescribir los demás párrafos.
- Los goldens y `.codex/AGENTS.md` se regeneran **siempre** en la última tarea y nunca se concilian a mano.
- Si la r2 se fusiona antes, este cambio se rebasa sobre ella. Los conflictos previsibles son el borrado de `verifyDependencyFromHandoff` (si la r2 lo edita) y el cuerpo de `newGovernanceGateQuestion`; ambos se resuelven a favor de la forma de este diseño.

## 3. Flujo de datos

### 3.1 Salida 2 en SDD (`gate record --hand-off`)

```
orquestador ──(opción hand_off de la pregunta tipada)──▶ axiom sdd gate record … --decision approved --hand-off
   │
   ├─ ParseGateRecordArgs ── forma (DD-01)
   ├─ resolveGovernanceChangeRoot ── carpeta del cambio
   ├─ guardChangeWriter ───────────▶ engine.RequireChangeBranch ── wrong_branch? ✗
   ├─ kickoff.Load ── sellado y checkpointed? ✗
   ├─ DeclaredArtifactStore ── engram/none? ✗
   ├─ gateArtifactDigest ── vacío? ✗
   ├─ LoadGates + Approval(gate) ── ya aprobada? ✗
   ├─ engine.HandOff ──▶ gather/describe (solo lecturas) ──▶ gitcommon.Run
   │        │                  (rev-parse, worktree list, status, rev-list)
   │        ├─ DirtyPaths filtrado (DD-03) ── sucio? ✗ uncommitted_artifacts
   │        └─ HandOffPlan{deliveries por repositorio}
   ├─ AppendGateChecked(record{approved, handed_off}, notApproved) ──▶ gates.yaml
   └─ salida: texto o JSON (§4.4) ── commit de gates.yaml, push y PR head/base; «fusionar antes de seguir»
orquestador ── presenta y PARA (no inicia la fase siguiente)
```

### 3.2 Lectura posterior

```
axiom sdd status ── loadGovernance ── EvaluateGates ── GateState{approved, HandedOff}
     ├─ governance.gates[].handedOff/handedOffTo (JSON v2)
     ├─ notes: «… cediendo <fase> a otra persona …»
     └─ nextRecommended: igual que tras un approved normal

axiom branch status / dashboard / TUI ── PendingArchive ── TreeDirs(changes/)
     └─ por carpeta: FileAt(gates.yaml) → ¿alguna marca? → FileAt(kickoff.yaml) → IntermediateMerge → se omite
axiom branch ensure --archive ── checkIntegrated ── carpeta presente → IntermediateMerge? → change_not_integrated (intermediate)
```

### 3.3 Parada ODD

```
paso 5: odd/tasks/<f>.md creado ──▶ parada del plan (tres salidas)
   1 iterar ─▶ ajustar fichero y espejo ─▶ volver a la parada
   2 subir   ─▶ «Plan stop: approved …; implementation handed off» ─▶ commit solo del documento
              ─▶ axiom branch status --lane odd --change <f> --json ─▶ presentar rama, prTarget, push y PR ─▶ PARAR
   3 seguir  ─▶ «Plan stop: approved …; implementation continues» ─▶ paso 6
```

## 4. Interfaces y contratos

### 4.1 `internal/kickoff`

```go
// types.go
type GateRecord struct {
	// … campos actuales sin cambios …
	Verified bool `yaml:"verified,omitempty"`
	// HandedOff marca que esta aprobación cede la fase siguiente a otra persona
	// (salida 2). Solo la escribe `gate record --hand-off`; omitempty deja los
	// registros de las salidas 1 y 3 idénticos a los de antes.
	HandedOff  bool      `yaml:"handed_off,omitempty"`
	Actor      string    `yaml:"actor"`
	RecordedAt time.Time `yaml:"recorded_at"`
}

type GateState struct {
	Key       GateKey
	Status    string
	Reason    string
	Blocks    string
	Reopened  bool
	HandedOff bool // del registro approved que cierra la puerta
}

// Deprecated: handoff_policy ya no tiene efecto (REQ-26.9); se lee y no se escribe.
HandoffPolicy HandoffPolicy `yaml:"handoff_policy,omitempty"`

// handed_off.go
func (l GateLedger) Approval(gate GateKey) (GateRecord, bool)           // primer approved de la puerta
func HandedOffTo(gate GateState, roster []multirole.RoleAssignment, states []GateState) string
func IntermediateMerge(k Kickoff, l GateLedger) bool

// ledger.go
func AppendGateChecked(changeRoot string, rec GateRecord, check func(GateLedger) error) error
func AppendGate(changeRoot string, rec GateRecord) error // = AppendGateChecked(changeRoot, rec, nil)

// args.go
type GateRecordArgs struct { /* … */ HandOff bool }
```

`resolveGateStatus` devuelve `(status, reason string, reopened, handedOff bool)`, un cambio interno del paquete.

### 4.2 `internal/consentenvelope`

```go
func (core Core) ValidateCompleteness(answers ...string) error
```

### 4.3 `internal/branch` y `internal/branch/engine`

```go
// ports.go (Git) y engine.gitReader
FileAt(ctx context.Context, dir, rev, relPath string) (content []byte, exists bool, err error)

// types.go
CodeUncommittedArtifacts Code = "uncommitted_artifacts" // etiqueta: "artefactos sin confirmar"

// handoff.go
type HandOffRequest struct {
	CWD        string   // workspaceRoot resuelto por la CLI
	Change     string
	ChangeRoot string   // carpeta del cambio (absoluta)
	Role       string   // "" en spec/design/tasks; el rol en role-apply
	Exempt     []string // rutas relativas a ChangeRoot exentas: "gates.yaml" (+ dotfiles de la raíz, regla fija)
}

type HandOffPR struct {
	Head string `json:"head"`
	Base string `json:"base"`
}

type HandOffDelivery struct {
	Repository string     `json:"repository"`
	GitRoot    string     `json:"gitRoot"`
	Branch     string     `json:"branch"`
	PRTarget   string     `json:"prTarget,omitempty"` // vacío en el repositorio de specs in situ
	InPlace    bool       `json:"inPlace,omitempty"`
	Steps      []string   `json:"steps"` // órdenes exactas, en orden, para la persona
	PR         *HandOffPR `json:"pr,omitempty"`
}

type HandOffPlan struct {
	Topology      workspace.TopologyType `json:"topology"`
	MergeRequired bool                   `json:"mergeRequired"`
	Deliveries    []HandOffDelivery      `json:"deliveries"`
}

func UncommittedArtifacts(change string, repo RepoPlan, paths []string) *Refusal
func ChangeIntermediateMerge(change, master, prefix string, repo RepoPlan, base, changePath string) *Refusal

// engine/handoff.go
func (e *Engine) HandOff(ctx context.Context, req branch.HandOffRequest) (branch.HandOffPlan, *branch.Refusal)
```

### 4.4 Salida de `axiom sdd gate record`

- **Texto, salida 2** (castellano): la línea «Compuerta "<g>" registrada (approved, fase siguiente cedida: <handedOffTo>) para "<c>".», los pasos numerados por repositorio (commit de `gates.yaml`, push y PR `<head>` → `<base>`, o push de la integración in situ), la advertencia de fusión previa y la aclaración «Axiom no ejecuta ninguna de estas órdenes».
- **`--json`**, que hoy se acepta y se ignora (`sdd_gate.go` no lee `parsed.JSON`). Pasa a emitir el esquema `axiom.sdd-gate-record/v1` en **todas** las salidas:

```json
{
  "schema": "axiom.sdd-gate-record/v1",
  "change": "x",
  "gate": "spec",
  "decision": "approved",
  "handedOff": true,
  "handedOffTo": "design",
  "lastRoleClosed": false,
  "delivery": {
    "topology": "monorepo-embedded",
    "mergeRequired": true,
    "deliveries": [
      {
        "repository": "axiom",
        "gitRoot": "C:\\repos\\axiom-wt\\x\\axiom",
        "branch": "inc/x",
        "prTarget": "main",
        "steps": [
          "git -C \"C:\\repos\\axiom-wt\\x\\axiom\" add -- openspec/changes/x/gates.yaml",
          "git -C \"C:\\repos\\axiom-wt\\x\\axiom\" commit -m \"chore(sdd): aprobar spec de x y ceder design\" -- openspec/changes/x/gates.yaml",
          "git -C \"C:\\repos\\axiom-wt\\x\\axiom\" push -u origin inc/x"
        ],
        "pr": { "head": "inc/x", "base": "main" }
      }
    ]
  }
}
```

`delivery`, `handedOff` y `handedOffTo` se omiten fuera de la salida 2. `lastRoleClosed` solo aparece en `role-apply`. En el JSON, los rechazos siguen siendo errores de la orden: código distinto de cero y mensaje con continuación, como hoy.

### 4.5 `axiom sdd status` (JSON v2)

```yaml
governance:
  kickoff: { schema, flowMode, executionStyle, handoffPolicy?, sealedAt, sealedBy }   # handoffPolicy solo si el kickoff antiguo lo trae
  gates: [{ key, status, blocks, reason?, reopened?, handedOff?, handedOffTo? }]
gateQuestion:   # solo con nextRecommended = await-gate
  choices: [rejected, hand_off, approved]          # o [rejected, approved] + handOffUnavailable: none
  handOffUnavailable?: engram|none
notes: ["La compuerta \"spec\" se aprobó cediendo `design` a otra persona: …"]
```

## 5. Mapa de ficheros

| Fichero | Acción | Qué cambia |
|---|---|---|
| `internal/consentenvelope/envelope.go` | Modificar | `ValidateCompleteness(answers ...string)`; comentario del paquete |
| `internal/consentenvelope/envelope_test.go` | Modificar | Casos válidos con 3; longitud o orden distintos; respuestas duplicadas; menos de 2 |
| `internal/kickoff/types.go` | Modificar | `GateRecord.HandedOff`, `GateState.HandedOff`, `handoff_policy` `omitempty` y deprecado |
| `internal/kickoff/schema.go` | Modificar | `Validate` acepta `handoff_policy` vacío |
| `internal/kickoff/args.go` | Modificar | `--handoff-policy` opcional e ignorado; `--hand-off` y su validación de forma |
| `internal/kickoff/infer.go` | Modificar | El retro-sello no escribe la política |
| `internal/kickoff/machine.go` | Modificar | `resolveGateStatus` devuelve la marca; `EvaluateGates` la copia; comentario `:157` |
| `internal/kickoff/ledger.go` | Modificar | `AppendGateChecked`; `AppendGate` delega |
| `internal/kickoff/handed_off.go` | Crear | `Approval`, `HandedOffTo`, `IntermediateMerge` |
| `internal/kickoff/closure.go`, `closure_test.go` | Eliminar | REQ-21.14 retirado |
| `internal/kickoff/rdd_boundary_test.go` | Modificar | Sin la arista `kickoff → handoff` |
| `internal/kickoff/*_test.go` | Crear y modificar | Compatibilidad del ledger con el esquema anterior (struct congelado), marca, intermedia, args y validación |
| `internal/cli/sdd_gate.go` | Modificar | Flujo de la salida 2 (puntos de llamada), `--json`, ayuda, mensaje de `:89`, sin `handoff.md`, aviso nuevo |
| `internal/cli/sdd_gate_handoff.go` | Crear | Comprobaciones 4, 6, 7 y 8, llamada a `engine.HandOff` y render texto/JSON |
| `internal/cli/sdd_kickoff.go` | Modificar | Ayuda, texto y JSON sin la política |
| `internal/cli/sdd_gate_test.go`, `sdd_writer_guard_test.go`, `sdd_kickoff_test.go` | Modificar | Sin `handoff.md` (`:211-213`); salida 2: orden de comprobaciones, observador sin remotos ni commit, almacén, continuo, `integration`, ya aprobada, sucio, monorepo/multirepo y JSON |
| `internal/sddstatus/governance.go` | Modificar | Pregunta de 2/3 opciones con `handOffUnavailable`; `Validate`; sin `verifyDependencyFromHandoff` ni el import de `handoff` |
| `internal/sddstatus/governance_handoff.go` | Crear | `verifyDependencyFromRoleGates`, `handedOffNotes` y almacén efectivo de la pregunta |
| `internal/sddstatus/status.go` | Modificar | `:730` (llamada nueva), `:754` (almacén) y notas de cesión |
| `internal/sddstatus/status_v2.go` | Modificar | `handoffPolicy` `omitempty`; `gateV2.handedOff`/`handedOffTo` |
| `internal/sddstatus/handoff_dependency_test.go` | Eliminar | Sustituido |
| `internal/sddstatus/role_gates_dependency_test.go` | Crear | REQ-21.15: bloqueado/desbloqueado sin `handoff.md`; `handoff.md` `blocked` ignorado; un rol o continuo sin cambios; `per_checkpoint` sin efecto |
| `internal/sddstatus/governance_test.go`, `governance_continuous_test.go`, `status_v2_test.go` | Modificar | Fixture del sobre de 3; casos de 2 con `handOffUnavailable`; `integration` sin cambios; autoridad de edición con 3 rechazada |
| `internal/branch/ports.go` | Modificar | `Git.FileAt` |
| `internal/branch/gitcli/ops.go`, `ops_test.go` | Modificar | `FileAt` (`ls-tree` + `cat-file blob`); lista cerrada de lecturas |
| `internal/branch/types.go`, `labels.go` | Modificar | `CodeUncommittedArtifacts` y su etiqueta |
| `internal/branch/handoff.go` | Crear | Tipos de entrega, constructores de órdenes y rechazo `UncommittedArtifacts` |
| `internal/branch/archive_refusal.go` | Modificar | `ChangeIntermediateMerge` |
| `internal/branch/engine/engine.go` | Modificar | `gitReader.FileAt` |
| `internal/branch/engine/handoff.go`, `handoff_test.go` | Crear | `Engine.HandOff` |
| `internal/branch/engine/intermediate.go` | Crear | Lectura del árbol de integración y criterio DD-09 |
| `internal/branch/engine/pending.go`, `pending_test.go` | Modificar | Excluye las fusiones intermedias |
| `internal/branch/engine/archive.go`, `archive_test.go` | Modificar | `checkIntegrated` con rechazo intermedio |
| `internal/branch/engine/writer_inventory_test.go` | Modificar | Sin la entrada `writeIntegrationHandoff` |
| `internal/assets/generic/stop-exits.md` | Crear | Texto único de las tres salidas |
| `internal/components/agentguidance/stop_exits.go` | Crear | `StopExits()` |
| `internal/components/agentguidance/routing.go`, `routing_test.go` | Modificar | Paso 0, paso 5, reanudación, viñeta de la parada y sección *Stops and exits*; pines |
| `internal/components/sdd/orchestrator.go`, `orchestrator_shared_sections_test.go` | Modificar | Sustitución de `{{AXIOM_STOP_EXITS}}`; pin de igualdad en los doce orquestadores |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modificar | Sin `--handoff-policy`; tres salidas; marcador |
| `internal/assets/skills/_shared/sdd-status-contract.md`, `openspec-convention.md` | Modificar | Contrato y convención (DD-12) |
| `internal/assets/skills/sdd-spec/SKILL.md`, `sdd-design/SKILL.md` | Modificar | Sin la «Pregunta de Transición»; remisión a la parada |
| `internal/assets/assets_test.go` | Modificar | Pines `:2422-2470` y `:3798-3840` ampliados; pin «ningún asset ofrece generar `handoff.md` o continuar» y «ningún asset prescribe `--handoff-policy`» |
| `testdata/golden/*.golden`, `.codex/AGENTS.md` | Regenerar | Última tarea, con `-update` |
| `AGENTS.md`, `GEMINI.md`, `docs/manual-de-inicio.md` | Modificar | Paradas y relevo; `handoff.md` como utilidad general |

Fuera del mapa: `internal/handoff`, `cmd/axiom` (`axiom handoff`), `internal/dashboard`, `internal/tui` (heredan DD-09 sin cambios), `e2e/`, `bench/` y `openspec/INDEX.md`.

## 6. Conformidad con las specs

| Requisito | Dónde se cumple |
|---|---|
| REQ-21.8 a REQ-21.11 (tres salidas por puerta) | DD-05 (sobre), DD-12 (texto de la sección de puertas) |
| REQ-21.12 (iterar sin cambios) | `rejected` + digest (`machine.go:204-211`) sin tocar; texto DD-12 |
| REQ-26.1 (modelo; `integration` con dos; continuo sin paradas) | DD-02 pasos 1 y 4; DD-05 |
| REQ-26.2 (pregunta de 3 sobre `gate record`; autoridad de edición en 2) | DD-04, DD-05 |
| REQ-26.3 (`approved` + marca; compatibilidad; ya aprobada) | DD-01, DD-02 pasos 8 y 10; prueba con el struct congelado |
| REQ-26.4 (confirmado antes de escribir; `wrong_branch` primero) | DD-02 pasos 3 y 9, DD-03 (más estricto, R-3) |
| REQ-26.5 (destino y órdenes sin remoto; topologías; JSON) | DD-07, DD-08, §4.4; observador `gitcommon.Observe` |
| REQ-26.6 (almacén) | DD-02 paso 6, DD-05 (corrección de A-3) |
| REQ-26.7 (cesión visible, no bloquea) | DD-06 |
| REQ-26.8 (texto único; sin «Pregunta de Transición» ni flag obligatorio) | DD-12 |
| REQ-26.9 (`handoff_policy` deprecado) | DD-11 |
| REQ-21.5 (sellado solo de la modalidad) | DD-11, DD-12 |
| REQ-21.2, REQ-21.4 (ODD; retro-sello) | DD-13, DD-11 (`infer.go`) |
| REQ-26.10, REQ-25.40 (parada ODD) | DD-13 |
| REQ-21.13, REQ-21.15; REQ-21.14 retirado | DD-10 |
| REQ-26.11, REQ-25.20, REQ-25.45 (fusión intermedia) | DD-09 |

## 7. Corrección de los supuestos de las specs

| Supuesto | Veredicto | Evidencia y decisión |
|---|---|---|
| A-1 (salida 2 sobre puerta ya aprobada se rechaza) | **Confirmado y reforzado** | `resolveGateStatus` devuelve el primer `approved` (`machine.go:199-203`): una marca posterior sería invisible. Se comprueba antes de escribir y otra vez bajo el cerrojo (`AppendGateChecked`, DD-02) |
| A-2 (`role-apply` exige el trabajo del rol confirmado) | **Confirmado y concretado** | Árbol entero de cada repositorio del rol (`RepositoryResult.Roles`) más la carpeta del cambio, con exenciones de `gates.yaml` y ficheros de estado con punto (DD-03) |
| A-3 (`engram`/`none`: pregunta de dos opciones) | **Corregido** | Con `engram` no hay pregunta: la ruta de Engram nunca carga gobierno (`status.go:847-853`). La variante de dos opciones aparece solo con `none` declarado y el cambio en disco (`workspace_declarations.go:192-205`), y la marca `handOffUnavailable` la hace validable. La CLI rechaza `--hand-off` con ambos (DD-02 paso 6, DD-05) |
| A-4 (destino de la PR de un rol = el `prTarget` de `status`) | **Confirmado** | Rama de rol → rama de seguimiento (`planner.go:400-413`); la salida no afirma integración (DD-07). Riesgo R-5 |
| A-5 (orden de plataforma) | **Decidido: no** | Push exacto y `head`/`base`; sin `gh`/`glab` (DD-08) |
| A-6 (valor desconocido de `--handoff-policy` se rechaza) | **Confirmado** | `args.go:203-205` se conserva (DD-11) |
| A-7 (todos los `role-apply` aprobados ⇒ no intermedia) | **Confirmado** | Criterio (b) de DD-09 |
| A-8 (sin marca o ilegible ⇒ como hasta ahora) | **Confirmado** | DD-09 pasos 1 a 3. Un error de git (no de contenido) también se trata como hasta ahora, con un aviso |

## 8. Estrategia de pruebas

TDD desactivado (`strict_tdd: false`). Solo `go test` acotado por paquete, en proceso. No se compilan binarios sueltos (Cortex los borra; las pruebas de CLI usan los puntos de entrada `RunSDDGate` y `RunSDDKickoff` en proceso). Las pruebas que necesitan git usan repositorios temporales con `gitcommon.Run`, como las del motor.

| Capa | Qué | Cómo |
|---|---|---|
| `consentenvelope` | Aridad declarada; autoridad de edición con 3 rechazada | `go test ./internal/consentenvelope/` |
| `kickoff` | Marca `omitempty` (salidas 1/3 byte a byte); compatibilidad con un `legacyGateRecord` congelado (el struct anterior) decodificando `handed_off: true` sin error y la puerta `approved`; primer `approved` manda; `HandedOffTo`; `IntermediateMerge` (A-7, A-8, roster vacío); `Validate` con `""`, `none`, `per_checkpoint` y `otra`; flags | `go test ./internal/kickoff/` |
| `sddstatus` | Sobre 3/2/`integration`; validación de duplicados y prefijo; cesión en JSON y notas; sin marca = salida idéntica (comparación con un golden de status existente); `verify` multirrol por roles; `per_checkpoint` sin efecto | `go test ./internal/sddstatus/` |
| `branch`, `engine`, `gitcli` | `FileAt`; `HandOff` (monorepo con rama de cambio y de rol, multirepo in situ, sucio/limpio/sin seguimiento/preparado, exenciones); `PendingArchive` con `x` intermedio y `y` normal; `--archive` intermedio sin `--fetch`; observador sin operación remota | `go test ./internal/branch/...` |
| `cli` | Orden de comprobaciones (DD-02) con aserción de `gates.yaml` intacto en cada rechazo; observador sin `push`/`fetch`/`pull`/`ls-remote`/`commit`; JSON `axiom.sdd-gate-record/v1`; aviso de último rol sin `handoff.md`; kickoff sin y con `--handoff-policy` | `go test ./internal/cli/` |
| Doctrina y assets | `StopExits()` contenido literal en `RenderRouting` de todos los agentes y en los doce orquestadores compuestos (Pi incluido); paso 0 y 5; sin «Pregunta de Transición»; sin `--handoff-policy` prescrito; marcador resuelto | `go test ./internal/components/agentguidance/ ./internal/components/sdd/ ./internal/assets/` |
| Goldens | Regeneración final | `go test ./internal/components/ -run Golden -update`, después sin `-update`; `go vet ./...` |

Ratchets: cada rechazo nuevo de `gate record` y del motor lleva continuación ejecutable o una anotación `refusal:by-design` justificada. La retirada de `closure.go` y de `writeIntegrationHandoff` no deja código muerto (ratchet de deadcode).

## 9. Matriz de amenazas

El cambio presenta órdenes de VCS y PR y añade lecturas git en proceso, así que la matriz aplica.

| Frontera | Aplicabilidad | Respuesta de diseño | Pruebas RED previstas |
|---|---|---|---|
| Rutas que parecen documentación | N/A: no se clasifica ni ejecuta ningún fichero; `stop-exits.md` es texto embebido | — | — |
| Selección de repositorio git | **Aplica** | Toda orden presentada usa `git -C <gitRoot>` absoluto, resuelto por el motor (no el cwd del proceso) y citado con `pathquote.Quote`; el repositorio es el que contiene `ChangeRoot`. Las lecturas pasan por `gitcommon.Run` con `dir` del repositorio | (1) Ruta con espacios citada; (2) cwd del proceso distinto del worktree: el `-C` es el del worktree; (3) en multirepo, cada entrega con su propio `gitRoot` |
| Estado del commit | **Aplica** | «Confirmado» = sin entradas en `status --porcelain` (modificado, preparado o sin seguimiento) bajo el prefijo, con exenciones fijas. Axiom nunca hace commit; las órdenes presentadas usan pathspec explícito (`add -- <gates.yaml>`, `commit -m … -- <gates.yaml>`), nunca `-a` | (1) Preparado sin confirmar ⇒ rechazo; (2) sin seguimiento ⇒ rechazo; (3) carpeta entera sin seguimiento ⇒ rechazo; (4) solo `gates.yaml`/`.gates.lock` sucios ⇒ pasa; (5) observador sin `commit` |
| Estado del push | **Aplica** | Push presentado con refspec explícita (`push -u origin <rama>`; in situ, `push origin <integración>`); nunca se ejecuta (el puerto no tiene push) | (1) La orden nombra la rama actual, no `HEAD`; (2) rama de rol ⇒ `prTarget` = seguimiento; (3) observador sin `push`/`fetch`/`pull`/`ls-remote` |
| Órdenes de PR | **Aplica (solo datos)** | Sin orden de plataforma; `head`/`base` explícitos y en campos separados; sin prefijos de entorno ni órdenes compuestas con `&&` (pasos separados, compatibles con PowerShell 5.1) | (1) La salida no contiene `gh `/`glab `; (2) `pr.head`/`pr.base` exactos; (3) ningún paso contiene `&&` |

## 10. Compatibilidad, migración y reversión

### 10.1 Matriz de compatibilidad

| Escribe → Lee | Binario anterior | Binario nuevo |
|---|---|---|
| `gates.yaml` con `handed_off: true` | Decodifica sin error (no estricto, `schema.go:110`); puerta `approved`; no muestra la cesión (degradación esperada) | Muestra la cesión |
| `gates.yaml` de las salidas 1 y 3 | Idéntico a hoy | Idéntico |
| `kickoff.yaml` nuevo **sin** `handoff_policy` | **Falla** `Validate` («handoff_policy desconocido ""», `schema.go:89-91`): `status` y `gate` de ese cambio dan error (R-1) | Válido |
| `kickoff.yaml` antiguo con `none`/`per_checkpoint` | Como hoy | Válido; sin efecto; JSON y texto lo muestran como obsoleto |
| Orquestador antiguo pasa `--handoff-policy` | — | Aceptado e ignorado |
| Orquestador nuevo pasa `--hand-off` a un binario antiguo | «bandera desconocida» (`args.go:444-446`): falla cerrada, sin escribir | — |
| `handoff.md` ya generados | `verify` multirrol los consulta | Se ignoran en `verify`; legibles con `axiom handoff` |
| Integración con un cambio intermedio | Figura como pendiente de archivar; `--archive` lo acepta | Excluido; `--archive` rechaza |

### 10.2 Migración

No hay migración de datos. Los kickoffs y ledgers existentes se leen sin cambios. `axiom sync` reproyecta la doctrina y los orquestadores.

### 10.3 Reversión

- **Código**: `git revert` de la PR.
- **Datos**: los `gates.yaml` con `handed_off` siguen siendo legibles por el binario revertido. A los `kickoff.yaml` sellados sin la clave basta con añadirles `handoff_policy: none`; `seal` revertido vuelve a escribirla.
- **Doctrina**: `axiom sync` con el binario revertido.
- **Señales para revertir** (`proposal.md` §9): sobre de autoridad de edición rechazado por la validación nueva; `verify` multirrol bloqueado con todos los roles aprobados o desbloqueado sin ellos; `kickoff seal` que falle con instrucciones antiguas; un `gates.yaml` con la marca que un binario anterior no lea.

## 11. Riesgos

| ID | Riesgo | Prob. | Mitigación |
|---|---|---|---|
| R-1 | Equipos con versiones mezcladas: un `kickoff.yaml` nuevo, sin `handoff_policy`, no lo valida un binario anterior. Afecta justo al caso de uso (quien recoge con otra versión) | Media | Lo impone REQ-26.9. Mitigación documental en el manual («actualiza Axiom antes de recoger un relevo»); alternativa para la compuerta de producto: escribir `handoff_policy: none` durante una versión de transición (requiere enmendar REQ-26.9). **Decisión del usuario** |
| R-2 | La r2 se fusiona antes y edita `verifyDependencyFromHandoff`/`newGovernanceGateQuestion` | Media | DD-14; rebase; goldens regenerados al final |
| R-3 | DD-03 es más estricto que la letra de REQ-26.4 (toda la carpeta, no solo lo juzgado) | Baja | Cumple todos los escenarios; motivo explícito; si se prefiere la letra, el filtro se reduce a las rutas del digest sin tocar el resto del diseño |
| R-4 | Generalizar `ValidateCompleteness` debilita otro sobre | Baja | Llamadores sin cambios; pruebas de no regresión de autoridad de edición, RDD y `blocking_prompt_contract_test.go` |
| R-5 | Multirrol en monorepo: la PR de un rol va a la rama de seguimiento; quien recoge en otra máquina ejecuta `ensure`, que crea la rama de seguimiento desde la base, no desde `origin/<seguimiento>`, y no ve el trabajo del rol | Media | La salida no afirma integración (A-4) y el texto lo advierte; limitación previa del motor, fuera de alcance. Punto abierto 2 |
| R-6 | `PendingArchive` hace hasta dos lecturas git más por cambio activo | Baja | Se lee `kickoff.yaml` solo si `gates.yaml` tiene alguna marca; el número de cambios activos es pequeño |
| R-7 | Tamaño muy por encima de 400 líneas | Alta | D7: una sola PR con `size:exception` |

## 12. Puntos abiertos (no bloquean)

1. R-1: ¿se acepta la incompatibilidad de `kickoff.yaml` con binarios anteriores tal como fija REQ-26.9, o se enmienda para una versión de transición que siga escribiendo `handoff_policy: none`?
2. R-5: la recogida en otra máquina de un cambio multirrol con rama de seguimiento queda como limitación conocida. ¿Hace falta un cambio posterior en `ensure` para partir de `origin/<seguimiento>`?
3. R-3: ¿se acepta el criterio estricto de DD-03 o se prefiere la lectura literal de REQ-26.4?
