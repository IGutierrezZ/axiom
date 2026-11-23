# Diseño: Aviso por commit, parada opcional por tarea y validación implícita al pedir la PR (C-02-d)

## Enfoque técnico

El cambio tiene dos mitades que no se tocan entre sí:

1. **Contrato en Go (solo SDD `checkpointed`).** El modo de revisión por commit se persiste como un campo opcional `commit_review` del registro `approved` que cierra la puerta `tasks` en `gates.yaml`, siguiendo el precedente exacto de `handed_off` (REQ-33.2, REQ-33.3). `axiom sdd gate record` lo escribe con la bandera `--commit-review`, que se analiza en la capa CLI sin tocar `internal/kickoff/args.go`. `kickoff.EvaluateGates` lo lleva a `GateState` desde el mismo registro que cierra la puerta, y `axiom sdd status` lo proyecta en la vista `gateV2` de la puerta `tasks` aprobada, con `notify` como valor efectivo cuando el registro no lo lleva (REQ-33.4). La pregunta tipada de la puerta, el digest y el kickoff no cambian.
2. **Doctrina (todos los runtimes y los dos carriles).** Una sección nueva de fuente única, «Commit Review», en `internal/assets/skills/_shared/sdd-orchestrator-sections.md`, referenciada por marcador desde los doce orquestadores, fija el aviso por commit, los tres modos, la interacción propia de la puerta `tasks`, la composición de la invocación, el avance tarea a tarea, la pausa (que no es una puerta), el commit del modo 3, el registro de los commits con cambios pedidos, `fixup!`/commit correctivo y la recomendación de merge sin squash (REQ-33.8, REQ-33.9). `sdd-apply` recibe el parámetro `commit_review` y un contrato de retorno por tarea (REQ-33.10). En ODD, el paso 6 de `routing.go` gana solo la frase del aviso y una viñeta de referencia con las reglas comunes, sin vocabulario de paradas ni del modo (REQ-33.12). Las dos copias de `work-unit-commits` reciben `fixup!`, autosquash diferido, commit correctivo y merge sin squash (REQ-33.11).

El aviso por tarea se consigue lanzando `sdd-apply` con **una tarea por lanzamiento en todos los modos**: es el único mecanismo portable por el que el aviso llega al chat cuando cada tarea termina, mientras la siguiente ya está en marcha (decisión D6).

## Decisiones de arquitectura

### Decisión D1: el modo vive en el registro `approved` de la puerta `tasks`

**Elección**: campo `CommitReview CommitReviewMode` con etiqueta `yaml:"commit_review,omitempty"` en `kickoff.GateRecord`, escrito en la misma llamada de `gate record` que aprueba la puerta `tasks` (decisión cerrada del explore, opción a).
**Alternativas consideradas**: reescribir `kickoff.yaml` (rompe el sellado único, REQ-25.36); un subcomando que actualice el kickoff (mismo problema y un verbo nuevo); una línea en `tasks.md` (cambia el digest que juzga la propia puerta); solo sesión (se pierde en la salida 2 y al reanudar).
**Justificación**: `gates.yaml` es append-only, el digest es del artefacto y no del registro, y `omitempty` deja los registros anteriores idénticos; un binario anterior ignora la clave desconocida y ve la puerta `approved` (comportamiento ya verificado para `handed_off`).

### Decisión D2: `--commit-review` se analiza en la capa CLI, sin tocar `internal/kickoff/args.go`

**Elección**: `runSDDGateRecord` separa primero la bandera de los argumentos con una función nueva `splitCommitReviewFlag(args) (kickoff.CommitReviewMode, []string, error)` en `internal/cli/sdd_gate_commit_review.go`, y pasa el resto sin cambios a `kickoff.ParseGateRecordArgs`. La función rechaza la bandera repetida, sin valor o con un valor fuera del vocabulario (incluida la cadena vacía y las variantes de mayúsculas, REQ-33.1).
**Alternativas consideradas**: añadir el caso a `ParseGateRecordArgs` y el campo a `GateRecordArgs` (lo natural, pero `args.go` queda fuera por decisión cerrada: lo comparte C-03-a y su conflicto de rebase es seguro); un `flag.FlagSet` en la CLI (cambiaría el análisis de todas las banderas).
**Justificación**: la separación previa no es ambigua, porque `requireFlagValue` ya rechaza cualquier valor que empiece por `-`: el token `--commit-review` nunca puede ser el valor de otra bandera. Si alguien escribe `--reason --commit-review stop`, la separación se lleva la bandera y `ParseGateRecordArgs` falla con «`--reason` requiere un valor», que es el error que habría dado igualmente. El vocabulario vive en `internal/kickoff` (lo necesita `GateRecord.Validate`); la CLI solo lo consulta.

### Decisión D3: validación en tres capas, con la comprobación de «puerta aún pendiente» bajo el cerrojo

**Elección**:
1. **Forma** (`validateCommitReviewShape`, CLI, antes de tocar el disco): solo con `--gate tasks`, solo con `--decision approved` y nunca con `--evidence-kind`. Cada rechazo nombra la orden que sigue siendo válida, como hace `validateHandOffShape`.
2. **Estado** (`requireCommitReviewStops`, CLI, tras `kickoff.Load`): el cambio debe estar sellado `checkpointed`; en `continuous` o sin sellar se rechaza nombrando la aprobación sin la bandera.
3. **Ledger** (bajo el cerrojo de `AppendGateChecked`): la puerta `tasks` no debe tener ya un registro `approved`. Se generaliza la comprobación que hoy solo hace la salida 2 (`handOffPrepared.notApprovedCheck`) a una función `gateStillPendingCheck(parsed, workspaceRoot)` que se usa cuando hay `--hand-off`, `--commit-review` o ambas. La comprobación previa sin cerrojo se repite también para dar el error antes de calcular nada más.
4. **Esquema** (`GateRecord.Validate`, en escritura y en lectura): un `commit_review` no vacío debe pertenecer al vocabulario e ir en un registro `tasks` + `approved`. Así un valor inválido persistido se rechaza al leer (REQ-33.3), igual que un `evidence_kind` desconocido.
**Alternativas consideradas**: dejar que una segunda aprobación con modo se anexe (como hoy ocurre con la salida 3 sin cesión): quedaría oculta tras la primera y el modo sería invisible, lo que contradice REQ-33.2 y REQ-33.6.
**Justificación**: mismo patrón y mismo orden que la cesión (DD-02): toda negativa se produce antes de escribir y la carrera entre dos registradores deja un único registro `approved`.

### Decisión D4: proyección en `GateState` y `gateV2`

**Elección**: `resolveGateStatus` pasa a devolver el registro que cierra la puerta (`closing GateRecord`) en lugar de dos valores sueltos de cesión; `EvaluateGates` copia de él `HandedOff`, `HandedOffTo` y el nuevo `CommitReview`. En `projectGovernanceV2`, `gateV2.CommitReview` (`json:"commitReview,omitempty"`) se rellena solo cuando la clave es `tasks` y el estado es `approved`, con `kickoff.EffectiveCommitReview(gate.CommitReview)`, que devuelve `notify` cuando el registro no lo lleva.
**Alternativas consideradas**: proyectar solo lo registrado (ausente sin campo): contradice REQ-33.4, que exige `notify` explícito en la puerta aprobada; un campo efectivo de nivel superior en `governance`: la spec lo sitúa junto a los campos de la puerta.
**Justificación**: un `continuous` no tiene puertas (`EvaluateGates` devuelve `nil`), así que el campo no puede aparecer ahí; una puerta pendiente u otra puerta tampoco lo proyectan. La ruta (`nextRecommended`) y las dependencias no leen el campo.

### Decisión D5: composición de la invocación por regla doctrinal acotada, sin tocar la pregunta tipada

**Elección**: la doctrina indica que, tras la respuesta a la interacción del modo, se ejecuta la invocación de la salida elegida (2 o 3) de la pregunta tipada **con ` --commit-review <token>` añadido al final**, después de rellenar `<your-reason>`, sin alterar ningún otro token. `ParseGateRecordArgs` no depende del orden de las banderas y la separación de D2 tampoco.
**Alternativas consideradas**:
- Un marcador `--commit-review <commit-review>` dentro de las invocaciones de aprobación de la puerta `tasks`, como `<your-reason>`: rechazada, porque el escenario de REQ-33.5 exige que ninguna de las tres opciones mencione el modo y que conserven su texto (REQ-26.2).
- Una invocación por modo: seis opciones de aprobación; rompe REQ-26.2.
- Un bloque `commitReview` dentro de `gateQuestion`, o una segunda pregunta tipada en `status`: no la exige la spec, añade un sobre con su validación y su contrato, y C-06-f aún no está diseñado. Si la interfaz es código Go del mismo módulo puede leer `kickoff.CommitReviewModes()`; si necesita una proyección, la añadirá su propio cambio sobre este contrato (bandera, campo y `commitReview` de `gateV2`).
**Justificación**: es la opción mínima que cumple el comportamiento observable de REQ-33.5 («el resto de los tokens no se altera») y mantiene intactos los tests de la pregunta tipada. La regla de «ejecutar sin alterar» las invocaciones del proveedor ya admite rellenar marcadores; la doctrina declara esta única excepción de forma explícita y acotada a la puerta `tasks`.

### Decisión D6: `sdd-apply` tarea a tarea en todos los modos

**Elección**: el orquestador lanza `sdd-apply` con **una tarea por lanzamiento** (la unidad que cierra con un commit de unidad de trabajo) en `notify`, `stop` y `stop-before-commit`, también en `continuous`. Al volver cada lanzamiento emite el aviso (sha, tarea, `git show <sha>`) y, en `notify`, lanza la siguiente tarea en el mismo turno sin esperar, de modo que la persona lee el aviso mientras el modelo avanza. El contrato de `sdd-apply` sigue admitiendo varias tareas en `notify` (REQ-33.10): si un resultado trae varios commits, el orquestador emite un aviso por sha, en orden de tareas (REQ-33.9).
**Alternativas consideradas**:
- Mantener el lote en `notify` y avisar al volver: los avisos llegan agrupados al final del lote, que es justo lo que el usuario no quiere («mientras el modelo sigue con la siguiente task»).
- Que `sdd-apply` avise por sí mismo: imposible; el texto de un subagente solo llega al padre cuando termina.
- Subagentes en segundo plano con sondeo: no existe en todos los runtimes y complica la recuperación.
**Justificación y coste**: cada lanzamiento vuelve a cargar la skill y los artefactos, así que el gasto de los subagentes crece con el número de tareas. Se mitiga en dos puntos: (1) en un lanzamiento de una sola tarea, `sdd-apply` lee completos `tasks` y `apply-progress`, y de spec y design primero los requisitos y decisiones que cita la tarea, ampliando la lectura cuando la tarea no cita nada o surge una duda; (2) el retorno por tarea es un bloque compacto (tarea, sha, asunto, ficheros, comprobaciones), para que el contexto del orquestador crezca poco por tarea. Donde no hay subagentes (Windsurf) `apply` corre en línea y el aviso sigue a cada commit sin coste extra. Esto cambia la redacción del punto 6 de la propuesta («en `notify` mantiene el lote») sin contradecir la spec.

### Decisión D7: modo 3, `apply` deja la tarea preparada y el orquestador hace el commit

**Elección**: en `stop-before-commit`, `sdd-apply` implementa la tarea, marca su casilla, prepara exactamente las rutas que habría confirmado (`git -C <worktree> add -- <paths>`), no hace el commit y devuelve `pending_commit` con la tarea, el mensaje propuesto y las rutas; en `apply-progress` deja la tarea como «commit pendiente de aprobación» con ese mensaje, para poder reanudar. El orquestador presenta `git -C <worktree> diff --cached --stat` y la orden `git -C <worktree> diff --cached` para el diff completo. Si la persona aprueba, el orquestador ejecuta `git -C <worktree> add -- <paths>` (recoge también lo que la persona haya editado en esas rutas) y `git -C <worktree> commit -m "<message>"`, sin `-a`, y emite el aviso con el sha. Si pide cambios, relanza `sdd-apply` sobre la misma tarea con la petición y vuelve a presentar la pausa.
**Alternativas consideradas**: relanzar `sdd-apply` solo para confirmar (un lanzamiento completo por tarea para una operación mecánica de git); dejar el trabajo sin preparar (`git diff` no enseña los ficheros nuevos); volcar el diff completo en el chat (coste de contexto sin límite).
**Justificación**: `git add` y `git commit` son operaciones de estado que el orquestador ya puede hacer en línea, el mensaje y las rutas los decide quien implementó, y el aviso sale con el sha justo después de aprobar. Las rutas modificadas fuera de la lista no se confirman: se informan a la persona.

### Decisión D8: registro de los commits con cambios pedidos

**Elección**: solo se registran los commits que ya existen y sobre los que la persona pidió cambios. En SDD, una línea por petición en la sección `## Commits con cambios pedidos` del fichero de tareas del rol (`tasks_file` del kickoff: `tasks.md` con `fullstack`), escrita por `sdd-apply` en el lanzamiento de corrección. En ODD, la misma sección en `odd/tasks/<feature>.md`, escrita por quien hace la corrección. Formato de línea, nunca una casilla: `- <sha-corto> (tarea <id>): <petición en una línea> → fixup` o `→ commit correctivo`. Las iteraciones del modo 3 antes del commit forman parte de la pausa y no se registran.
**Alternativas consideradas**: `apply-progress` (interno, en Engram o fichero según el almacén, invisible en la PR); un fichero nuevo (otro artefacto que publicar y archivar).
**Justificación**: el fichero de tareas del rol es lo que juzga la puerta `role-apply:<rol>` y ya alberga el `## Registro de remediación`, así que quien revisa la puerta ve las correcciones. La puerta `tasks` aprobada es terminal y no se reabre aunque cambie el digest (REQ-33.10). Una línea escrita antes de presentar `role-apply` entra en el digest que se juzga; una escrita tras un rechazo de `role-apply` la devuelve a `pending`, que es justo la remediación. Las líneas no empiezan por `- [`, así que no alteran el progreso de tareas.

### Decisión D9: `fixup!`, intercalado y «rama publicada»

**Elección**:
- **Publicado**: un commit está publicado cuando alguna rama remota de seguimiento lo contiene (`git -C <worktree> branch -r --contains <sha>` imprime algo). Nunca se hace `fetch` para decidirlo.
- **No publicado**: `git -C <worktree> commit --fixup=<sha>`. El autosquash se difiere hasta que la persona pide la PR: `git -C <worktree> -c sequence.editor=true rebase -i --autosquash <base>`, con `<base>` igual a la rama remota de seguimiento de la rama del cambio cuando existe (los commits de documentos que publicó `axiom sdd publish` quedan por debajo) o, si no existe, `git merge-base HEAD <prTarget>` (`prTarget` de `axiom branch status --json`). El `-i` no abre editor: el editor de la secuencia es `true`. Solo se ejecuta si ningún objetivo de `fixup!` está publicado. Ante un conflicto, `git rebase --abort`, los `fixup!` se quedan como están y se informa; nada bloquea la PR. Tras el autosquash se listan los sha nuevos por tarea (`git log --oneline <base>..HEAD`), porque los avisos anteriores quedan obsoletos.
- **Publicado**: commit correctivo nuevo que cita el sha corregido; nunca se reescribe historia publicada ni se fuerza un push.
- **Intercalado**: primero se termina la tarea en curso. En SDD no se interrumpe un `sdd-apply` en marcha: la corrección es su propio lanzamiento, con `fixup_of`, entre dos tareas; en el modo 3, la tarea pendiente se aprueba o se itera antes. En ODD, si la tarea en curso no puede cerrarse antes, se aparca con `git stash push --include-untracked -m "wip <tarea>"`, se hace la corrección y se recupera con `git stash pop`.
**Alternativas consideradas**: `commit --amend` para el último commit (no sirve para commits anteriores y borra la huella de la revisión); autosquash inmediato (cambia sha de avisos ya emitidos y de la tarea en curso); decidir «publicado» con `git ls-remote` o `fetch` (operación remota sin autorización).
**Justificación**: un solo mecanismo para todos los casos, diferido al único momento en que reescribir es seguro, y con la frontera de lo publicado calculada solo con referencias locales.

### Decisión D10: forma de la doctrina

**Elección**: sección nueva `Commit Review` en la fuente única, registrada en `sharedOrchestratorSectionNames`, con encabezado `### Commit Review` y su marcador justo después de `### Chain Strategy` en los doce orquestadores (11 `internal/assets/<agente>/sdd-orchestrator.md` y `internal/assets/claude/sdd-orchestrator-workflow.md`). «SDD Change Kickoff and Block Gates» gana una sola frase que enlaza la interacción de la puerta `tasks` con la sección nueva (sin nombrar `gateQuestion`, que el test de esa sección prohíbe). En ODD, una frase al final del paso 6 y una viñeta nueva en la sección de referencia «Organic Driven Development», tras «Delivery follows work units».
**Alternativas consideradas**: repartir las reglas entre secciones existentes (Delivery Strategy, Block Gates): mezcla dos contratos y complica la paridad; copiar el texto en cada orquestador: lo prohíbe `TestSharedOrchestratorSectionsHaveOneSource`.
**Justificación**: una fuente, un marcador por runtime, y la prueba de ausencia de deriva cubre la sección nueva sin trabajo adicional.

### Decisión D11: tokens y vocabulario

**Elección**: los tokens de la spec, `notify`, `stop` y `stop-before-commit` (REQ-33.1). La doctrina llama **pausa** a la espera por tarea de los modos 2 y 3 y declara que no es una parada de puerta: no ofrece las tres salidas, no escribe en `gates.yaml` y no cambia ningún digest.
**Alternativas consideradas**: renombrar a `pause`/`pause-before-commit` para no chocar con «stop» de «Stops and exits»: la spec ya fijó los tokens y la aclaración doctrinal basta.
**Justificación**: alinear con la spec y neutralizar la ambigüedad con una frase explícita.

## Flujo de datos

Puerta `tasks` en `checkpointed`:

    status (await-gate) ──→ gateQuestion (3 salidas, sin cambios)
                                  │ la persona elige salida 2 o 3
                                  ▼
                 interacción propia del modo (notify | stop | stop-before-commit)
                                  │
                                  ▼
     invocación elegida + " --commit-review <token>"  ──→ gate record
                                  │  splitCommitReviewFlag → ParseGateRecordArgs
                                  │  forma → estado (checkpointed) → cerrojo (pendiente)
                                  ▼
                 gates.yaml: registro approved {commit_review: <token>}
                                  │
                                  ▼
     EvaluateGates → GateState.CommitReview → gateV2.commitReview (tasks approved)

Implementación (todos los modos):

    status.governance.gates[tasks].commitReview  (ausente → notify)
                │
                ▼
    orquestador ──lanza (1 tarea, commit_review)──→ sdd-apply
         ▲                                             │
         │   notify/stop: {tarea, sha}                 │
         │   stop-before-commit: pending_commit        │
         └─────────────────────────────────────────────┘
         │
         ├─ notify: aviso → siguiente lanzamiento sin esperar
         ├─ stop: aviso → pausa → seguir | cambios (lanzamiento fixup_of)
         └─ stop-before-commit: diff --cached --stat → pausa
                 ├─ aprueba: git add -- <paths>; git commit -m → aviso
                 └─ cambios: relanza la misma tarea

Petición de PR:

    ¿hay fixup! sin publicar? ──sí──→ rebase -i --autosquash <base> (editor true)
                │                          └─ conflicto → --abort, se informa
                ▼
    todos los commits quedan validados; se recomienda merge sin squash

## Cambios de ficheros

| Fichero | Acción | Descripción |
|---------|--------|-------------|
| `internal/kickoff/types.go` | Modificar | Tipo `CommitReviewMode`, constantes, `CommitReviewModes()`, `EffectiveCommitReview()`; campo `CommitReview` en `GateRecord` (`commit_review,omitempty`) y en `GateState`. |
| `internal/kickoff/schema.go` | Modificar | `validCommitReviewModes`; `GateRecord.Validate` rechaza un valor fuera del vocabulario o en un registro que no sea `tasks` + `approved`. |
| `internal/kickoff/machine.go` | Modificar | `resolveGateStatus` devuelve el registro que cierra la puerta; `EvaluateGates` copia `CommitReview`. |
| `internal/kickoff/schema_test.go`, `machine_test.go` (y `ledger_test.go` si procede) | Modificar | Validación del campo, lectura de ledgers antiguos y nuevos, decodificador no estricto anterior, rechazo posterior que no cambia el modo. |
| `internal/cli/sdd_gate_commit_review.go` | Crear | `splitCommitReviewFlag`, `validateCommitReviewShape`, `requireCommitReviewStops`. |
| `internal/cli/sdd_gate.go` | Modificar | Llama a la separación antes de `ParseGateRecordArgs`; comprobaciones de estado; `record.CommitReview`; comprobación de puerta pendiente bajo el cerrojo cuando hay cesión o modo; ayuda con la bandera (REQ-33.7). |
| `internal/cli/sdd_gate_handoff.go` | Modificar | `gateStillPendingCheck` generaliza `notApprovedCheck`; `gateRecordJSON.CommitReview` (`commitReview,omitempty`); cabecera de texto con el modo; firmas de `writeGateRecordJSON` y `writeGateRecordHeader` reciben el modo. |
| `internal/cli/sdd_gate_commit_review_test.go` | Crear | Casos de REQ-33.1, REQ-33.2 y REQ-33.7 de extremo a extremo sobre `RunSDDGate`. |
| `internal/sddstatus/status_v2.go` | Modificar | `gateV2.CommitReview` solo en `tasks` `approved`, con `notify` efectivo. |
| `internal/sddstatus/status_v2_test.go` (o `governance_handoff_test.go`) | Modificar | Escenarios de REQ-33.4. |
| `internal/assets/skills/_shared/sdd-status-contract.md` | Modificar | `commitReview?` en la forma de `gates` y su significado. |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modificar | Sección nueva `Commit Review`; una frase en «SDD Change Kickoff and Block Gates». |
| `internal/assets/{antigravity,codex,cursor,gemini,generic,hermes,kimi,kiro,opencode,qwen,windsurf}/sdd-orchestrator.md`, `internal/assets/claude/sdd-orchestrator-workflow.md` | Modificar | `### Commit Review` + `{{GENTLE_AI_SDD_SECTION:Commit Review}}` tras `### Chain Strategy`. |
| `internal/components/sdd/orchestrator_shared_sections_test.go` | Modificar | Registrar `"Commit Review"` en `sharedOrchestratorSectionNames`. |
| `internal/assets/assets_test.go` | Modificar | Pines del contenido de la sección (a)-(h) de REQ-33.9, de la frase de Block Gates y de la matriz de amenazas. |
| `internal/assets/skills/sdd-apply/SKILL.md` | Modificar | Ambas secciones (`model-capable` y `model-small`): parámetros `commit_review` y `fixup_of`, paso nuevo de revisión por commit, retorno por tarea, registro de cambios pedidos, ajuste del cierre (commit ya hecho por tarea; merge sin squash). |
| `internal/assets/skills/work-unit-commits/SKILL.md`, `skills/work-unit-commits/SKILL.md` | Modificar | Sección «Review and Fixups» idéntica en ambas copias. |
| `internal/assets/skill_copies_parity_test.go` | Sin cambios | Ya vigila la paridad de `work-unit-commits`; basta con ejecutarlo. |
| `internal/components/agentguidance/routing.go` | Modificar | Frase del aviso al final del paso 6; viñeta «Commits and the pull request» en la referencia ODD. |
| `internal/components/agentguidance/routing_test.go` | Modificar | Pines de REQ-33.12 (aviso en el paso 6, reglas comunes, ausencia de tokens del modo); el pin de REQ-27.17 y `routing_stops_test.go` no se tocan. |
| `testdata/golden/*.golden` | Regenerar | Orquestadores, `sdd-apply` y documentos combinados que proyectan routing; con `-update`. |

No cambian: `internal/kickoff/args.go`, `internal/cli/sdd_kickoff.go`, `internal/cli/sdd_gate_digest.go`, `internal/sddstatus/governance.go` y `governance_handoff.go` (pregunta tipada), `gate show`, las envolturas de agente de `sdd-apply` (`internal/assets/*/agents/sdd-apply*`), que remiten a la skill.

## Interfaces y contratos

### Go: vocabulario y registro (`internal/kickoff/types.go`)

```go
// CommitReviewMode is the per-commit review mode chosen at the tasks gate of a
// checkpointed change (REQ-33.1). The empty value means "not recorded" and is
// read as CommitReviewNotify.
type CommitReviewMode string

const (
	CommitReviewNotify           CommitReviewMode = "notify"
	CommitReviewStop             CommitReviewMode = "stop"
	CommitReviewStopBeforeCommit CommitReviewMode = "stop-before-commit"
)

// CommitReviewModes returns the closed vocabulary in its canonical order.
func CommitReviewModes() []CommitReviewMode

// EffectiveCommitReview maps the empty value to CommitReviewNotify.
func EffectiveCommitReview(m CommitReviewMode) CommitReviewMode

type GateRecord struct {
	// ...campos actuales...
	HandedOffTo string `yaml:"handed_off_to,omitempty"`
	// CommitReview is the mode recorded with the approval that closes the tasks
	// gate (REQ-33.2). Only `gate record --commit-review` writes it; omitempty
	// keeps every other record byte-identical, and an older binary ignores it.
	CommitReview CommitReviewMode `yaml:"commit_review,omitempty"`
	Actor        string           `yaml:"actor"`
	RecordedAt   time.Time        `yaml:"recorded_at"`
}

type GateState struct {
	// ...campos actuales...
	CommitReview CommitReviewMode // del registro approved que cierra la puerta
}
```

`machine.go`:

```go
func resolveGateStatus(digest string, records []GateRecord) (status, reason string, reopened bool, closing GateRecord)
```

### Go: CLI (`internal/cli/sdd_gate_commit_review.go`)

```go
// splitCommitReviewFlag removes `--commit-review <mode>` from args and returns
// the mode ("" when absent) and the remaining arguments unchanged and in order.
func splitCommitReviewFlag(args []string) (kickoff.CommitReviewMode, []string, error)

// validateCommitReviewShape: only --gate tasks, only --decision approved,
// never with --evidence-kind. Runs before anything touches the disk.
func validateCommitReviewShape(parsed kickoff.GateRecordArgs, mode kickoff.CommitReviewMode) error

// requireCommitReviewStops: the change must be sealed checkpointed.
func requireCommitReviewStops(parsed kickoff.GateRecordArgs, sealed *kickoff.Kickoff, workspaceRoot string) error
```

`internal/cli/sdd_gate_handoff.go`:

```go
// gateStillPendingCheck is the check run under the ledger lock when the record
// carries --hand-off or --commit-review: the gate has no approved record yet.
func gateStillPendingCheck(parsed kickoff.GateRecordArgs, workspaceRoot string) func(kickoff.GateLedger) error

type gateRecordJSON struct {
	// ...campos actuales...
	CommitReview string `json:"commitReview,omitempty"`
}
```

Ayuda de `gate record` (REQ-33.7): la línea de uso añade `[--commit-review notify|stop|stop-before-commit]` y una línea explicativa: solo con `--gate tasks --decision approved` en un cambio `checkpointed`; sin la bandera equivale a `notify`.

Mensajes de negativa (castellano, como el resto de la CLI), cada uno con la orden que sigue valiendo: otra puerta, rechazo, evidencia, `continuous`/sin sellar, puerta ya resuelta, valor fuera del vocabulario (lista `notify, stop, stop-before-commit`), bandera repetida o sin valor.

### Go: estado (`internal/sddstatus/status_v2.go`)

```go
type gateV2 struct {
	// ...campos actuales...
	// CommitReview is the effective per-commit review mode of the approved tasks
	// gate (REQ-33.4): the recorded mode, or notify when the record has none.
	// Absent on any other gate, on a pending tasks gate and in continuous.
	CommitReview string `json:"commitReview,omitempty"`
}
```

### Contrato de `sdd-apply`

Parámetros de lanzamiento nuevos:

- `commit_review`: `notify | stop | stop-before-commit`; `notify` cuando falta.
- `fixup_of` (opcional): `{ sha, task, request }`, una petición de cambios sobre un commit ya hecho. Ese lanzamiento no implementa tareas nuevas.

Retorno por tarea (bloque compacto, en `model-capable` dentro del resumen y en `model-small` como campos del JSON):

```json
{
  "commits": [{ "task": "2.1", "sha": "abc1234", "subject": "feat(kickoff): ..." }],
  "pending_commit": { "task": "2.2", "message": "feat(cli): ...", "paths": ["internal/cli/x.go"], "fixup_of": null }
}
```

- `notify`: puede traer varias entradas en `commits`; `pending_commit` es `null`.
- `stop`: una sola entrada en `commits` y devuelve el control.
- `stop-before-commit`: `commits` vacío y `pending_commit` con las rutas ya preparadas (`git add`) y sin confirmar; `apply-progress` marca la tarea «commit pendiente de aprobación».
- `fixup_of`: decide publicado/no publicado, hace `git commit --fixup=<sha>` o un commit correctivo (en `stop-before-commit`, lo deja preparado y lo devuelve en `pending_commit` con `fixup_of`), y añade la línea de D8 al fichero de tareas del rol.

### Borrador de la sección `Commit Review` (doctrina en inglés)

El texto definitivo lo escribe `sdd-apply`; este borrador fija su contenido y orden para los pines de los tests:

1. **Notice.** After each task commit, one line with the short sha, the task ID and `git show <sha>`; one notice per commit, also when a result reports several commits, in task order.
2. **Mode.** The three canonical tokens, never translated, and their effect; read from `commitReview` of the `tasks` gate in `governance.gates` of `axiom sdd status`; absent means `notify`; passed as `commit_review` to every `sdd-apply` launch.
3. **Asking the mode.** Only in `checkpointed`, at the `tasks` gate, after an approval exit; its own interaction, never an option of the gate question, which keeps its three exits; the three tokens with translated descriptions and `notify` declared as the default; exact single match or present again; then run the chosen exit's invocation with ` --commit-review <token>` appended at its end and every other token unchanged, once; iterating asks nothing; `continuous` never asks or records it.
4. **Task by task.** One task per `sdd-apply` launch in every mode; `notify` relaunches at once; `stop` and `stop-before-commit` wait.
5. **Pause.** Not a gate: no exits, nothing in `gates.yaml`, no digest change; reply handling for `stop` and for `stop-before-commit` (D7), and resumption of a pending commit.
6. **Pull request.** Asking for it validates every commit (and the pending diff in `stop-before-commit`, committed first); no commit-by-commit approval; nothing blocks it; recommend a merge commit without squash; Axiom never merges.
7. **Changes on a commit already passed.** Record only those (D8); finish the task in progress first; published test without fetch; `fixup!` with deferred autosquash or a corrective commit (D9); never rewrite published history and never force a push.
8. **No subagents.** Where `sdd-apply` runs inline, the notice follows each task commit directly.

La frase de «SDD Change Kickoff and Block Gates», tras la de las salidas: *"At the `tasks` gate, an approval exit is followed by the commit review question and recorded with `--commit-review`, as `Commit Review` describes."*

### ODD (`routing.go`)

- Paso 6, al final, tras «...under ordinary repository policy.»: *"After each work-unit commit, tell the person in one line the short sha, the task ID and `git show <sha>`, then continue with the next task without waiting."*
- Viñeta nueva tras «Delivery follows work units»: *"Commits and the pull request. Asking for the pull request validates every commit of the feature: there is no commit-by-commit approval and nothing blocks it. Record in a `## Commits con cambios pedidos` section of `odd/tasks/<feature>.md` only the commits the person asked to change, one line each, never a checkbox. To change a commit already made, first finish the task in progress, or park it with `git stash push --include-untracked` and restore it afterwards; when no remote-tracking branch contains the commit (`git branch -r --contains <sha>`, never fetch to decide), use `git commit --fixup=<sha>` and squash the fixups with `git -c sequence.editor=true rebase -i --autosquash <base>` only when the pull request is requested; once it is published, add a new corrective commit and never rewrite published history. Recommend merging with a merge commit, without squash."*
- Ninguna mención de `stop`, `stop-before-commit`, `--commit-review`, «commit review mode», «three exits» ni «plan stop».

### `work-unit-commits` (ambas copias)

Sección nueva «Review and Fixups» con: aviso por commit; solo se registran los commits con cambios pedidos; `fixup!` + autosquash diferido hasta pedir la PR si no está publicado, commit correctivo si lo está, nunca reescribir historia publicada ni forzar el push; terminar o aparcar la tarea en curso antes de intercalar; recomendar merge commit sin squash.

## Estrategia de pruebas

TDD estricto no está declarado para este cambio; las pruebas se escriben con cada unidad y se ejecutan con `go test` acotado (sin compilar binarios).

| Capa | Qué se prueba | Cómo |
|------|---------------|------|
| Unidad (`kickoff`) | Vocabulario cerrado; `GateRecord.Validate` acepta los tres tokens en `tasks`+`approved` y rechaza `Stop`, `pause`, `always`, el campo en otra puerta y en un rechazo; `EffectiveCommitReview`; `EvaluateGates` toma el modo del primer `approved` y un rechazo posterior no lo cambia (REQ-33.6); ledger antiguo sin campo → modo vacío/efectivo `notify`; un `struct` sin el campo (decodificador anterior, no estricto) decodifica un ledger con `commit_review` y ve `approved` (REQ-33.3). | Tablas en `schema_test.go` y `machine_test.go` (skill `axiom-go-table-tests`). |
| Unidad (CLI) | `splitCommitReviewFlag`: ausente, presente en cualquier posición, repetida, sin valor, valor inválido, resto intacto y en orden. | Tabla en `sdd_gate_commit_review_test.go`. |
| Integración (CLI) | `RunSDDGate record` sobre un cambio temporal: escribe `commit_review` con cada token; viaja con `--hand-off` en un único registro; sin bandera no escribe el campo; rechaza sin escribir con `design`, `rejected`, `--evidence-kind`, `continuous`, sin sellar y con `tasks` ya aprobada; el digest y `kickoff.yaml` no cambian (bytes antes/después); el JSON lleva `commitReview`; la ayuda menciona la bandera, los tres tokens y la restricción. | Fixtures de `sdd_gate_test.go`/`sdd_gate_handoff_test.go`; comprobación de `gates.yaml` byte a byte en los rechazos. |
| Integración (`sddstatus`) | `gateV2.commitReview`: `stop` registrado; `notify` sin campo; con cesión, ambos campos; ausente en `tasks` pendiente, en `design` aprobada y en `continuous`; `nextRecommended` y dependencias idénticos con y sin modo; la pregunta tipada de `tasks` sigue con tres opciones y sin mención del modo. | `status_v2_test.go` con fixtures de ledger. |
| Doctrina | La sección `Commit Review` existe una vez en la fuente, la referencian los doce orquestadores, no hay copia en línea, no menciona RDD ni recibos; contiene (a)-(h) de REQ-33.9, la regla de la puerta `tasks` (REQ-33.5) y la composición ` --commit-review <token>`; usa `git -C <worktree>`, no contiene `commit -a`, `--force` ni `git push`, y dice «never fetch»; Block Gates enlaza la sección. | `orchestrator_shared_sections_test.go`, `orchestrator_no_drift_test.go`, pines en `assets_test.go`. |
| Doctrina ODD | Paso 6 con el aviso idéntico en todos los adaptadores; viñeta con las reglas comunes y `odd/tasks/<feature>.md`; ausencia de `` `stop` ``, `stop-before-commit`, `--commit-review` y «commit review»; `routing_stops_test.go` y el pin de REQ-27.17 en verde sin tocarlos. | `routing_test.go`. |
| Skills | `sdd-apply` documenta `commit_review`, `fixup_of`, el retorno por tarea y el registro en el fichero de tareas del rol, en ambas secciones; `work-unit-commits` en paridad. | Pines en `assets_test.go`; `skill_copies_parity_test.go`. |
| Goldens | Regeneración y verificación. | `go test ./internal/components/ -run TestGolden -update`, después `go test ./internal/components/ -run TestGolden` sin `-update`; revisar el diff de `testdata/golden/`. |
| Cierre | Todo el repositorio. | `go test ./...` y `go vet ./...`. |

## Matriz de amenazas

El código Go no ejecuta git; la frontera aplicable es la automatización de git que la doctrina ordena al orquestador y a `sdd-apply`. Las pruebas son pines de doctrina.

| Frontera | Casos adversarios mínimos | Aplicabilidad | Respuesta de diseño | Pruebas RED previstas |
|---|---|---|---|---|
| Rutas de tipo documentación | `requirements.txt`, `CMakeLists.txt`, Markdown ejecutable, `README.sh` | N/A: el cambio no clasifica ficheros ni cambia qué se ejecuta. | — | — |
| Selección del repositorio git | `git -C`, rutas relativas, absolutas | Aplicable | Toda orden git de la doctrina usa `git -C <worktree>` con la ruta absoluta de `axiom branch ensure`; nunca se cambia el directorio de la sesión. | Pin: cada orden git de la sección `Commit Review` empieza por `git -C <worktree>`. |
| Estado del commit | preparado, `commit -a`, índice vacío | Aplicable | Modo 3: `git add -- <paths>` explícito y `git commit -m` sin `-a`; las rutas ajenas a la lista no se confirman y se informan; con índice vacío no hay commit y se informa. | Pin: la sección contiene `add -- <paths>` y no contiene `commit -a`. |
| Estado del push | rama de seguimiento, primer push, refspec explícito | Aplicable (como negativa) | La doctrina no hace push ni `fetch`; «publicado» se decide con `git branch -r --contains` sobre referencias locales; nunca se reescribe historia publicada ni se fuerza. La publicación de documentos sigue siendo `axiom sdd publish`. | Pin: la sección contiene «never fetch» y no contiene `--force` ni `git push`. |
| Órdenes de PR | `--head` explícito, prefijo de entorno, órdenes compuestas | N/A: no se añade ninguna orden de PR; la PR sigue siendo de la persona y la orden existente de `sdd-apply` no cambia. | — | — |

## Migración y despliegue

No hay migración. El campo es opcional y aditivo: los `gates.yaml` existentes se leen igual y su modo efectivo es `notify`; un binario anterior ignora `commit_review`. La reversión es revertir el merge commit, regenerar los goldens y ejecutar `axiom sync`.

## Preguntas abiertas

- [ ] Producto, no bloqueante: con la salida 2, quien aprueba elige el modo para quien recoge `apply`; queda persistido y visible en `status`, y cambiarlo después queda fuera de alcance (REQ-33.6).
- [ ] Coste de D6 en cambios con muchas tareas: si el gasto de los lanzamientos por tarea resulta excesivo, la alternativa con menos coste es volver al lote en `notify` con avisos agrupados al volver, que la spec también admite; se valorará con datos de uso, sin cambiar el contrato.
