## Exploración: identidad git como actor de los artefactos versionados (C-07-a)

### Estado actual

- **Compuerta (`gates.yaml`).** `runSDDGateRecord` fija el actor en `internal/cli/sdd_gate.go:100-103`: `actor := parsed.Actor; if actor == "" { actor = "cli" }`. `--actor` se parsea como texto libre en `internal/kickoff/args.go:496-502` (`GateRecordArgs.Actor`, `args.go:399`). El valor viaja en `GateRecord.Actor` (`internal/kickoff/types.go:191`, tag `yaml:"actor"`) y `kickoff.AppendGate` lo serializa con `yaml.Marshal` (`ledger.go:120`).
- **Validación del actor: ninguna.** `GateRecord.Validate` (`schema.go:194-208`) solo comprueba gate, decisión y `evidence_kind`. Ni longitud, ni caracteres, ni vacío. `Nombre <email>` con espacios y `<>` se admite; yaml.v3 lo emite como escalar plano (o entrecomillado si hace falta) y se relee sin pérdida. Un binario anterior lo lee igual (es un `string`).
- **Consumidores del actor.** Solo `gate show` lo imprime (`sdd_gate.go:240`). Ningún código no-test compara actores (ni `handed_off`, ni el dashboard, ni `sddstatus`). Los demás `--actor` del repo son de otros verbos (`sdd attempt grant`, `review *`, `techcontext/plan.go`) con ledgers no versionados: fuera de alcance.
- **Kickoff (`kickoff.yaml`).** `SealArgs.ToKickoff` fija `SealedBy: "cli"` en `internal/kickoff/args.go:375` (el plan cita `:324`, ya desplazado). No existe `--actor` ni `--sealed-by` en `seal`. `Kickoff.SealedBy` (`types.go:146`) no tiene validación de valor; la única semántica especial es `SealedByInferred = "inferred"` (`types.go:80`), que escribe `InferKickoff` (`infer.go:90`) y que `ValidateForSeal` exime de la regla de continuous (`schema.go:149-155`). `seal --infer` no pasa por `ToKickoff`: queda intacto por construcción.
- **Quién llama a `ToKickoff`.** `sdd_kickoff.go:83` y varios tests/e2e (`e2e/organicruntime/sdd_consent_wiring_e2e_test.go:77`, `kickoff/handoff_policy_deprecation_test.go:22`, `cli/sdd_kickoff_branch_test.go:80`, `cli/sdd_writer_guard_test.go:78`, `kickoff/args_test.go:528`). Es un método puro: conviene mantenerlo puro.
- **Salida.** `sealed_by` se muestra en el resumen de texto (`sdd_kickoff.go:204-205`) y en `--json` (`sealedBy`, `sdd_kickoff.go:237,261`); también lo proyecta `sddstatus/status_v2.go:133,190`. El JSON de `gate record` (`sdd_gate_handoff.go:195-207`) no lleva `actor` ni campo `warnings`.
- **Utilidad de identidad ya existente.** `reviewAuditActor` (`internal/cli/review_audit_actor.go:26-41`) deriva `Name <email>` de `git config --get user.name/user.email`, pero: es no exportada y de la fachada de review; ejecuta `exec.CommandContext("git")` directo (no `gitcommon.Run`); su cadena de reserva (solo email, `GIT_AUTHOR_EMAIL`, literal `gentle-ai-self-recovery@localhost`) es propia de los vínculos de recuperación de review, que son LF-delimitados y ligados por hash. No sirve tal cual: C-07-a necesita conservar `cli` y avisar.
- **Ejecutor de git.** `internal/gitcommon/exec.go`: `Run(ctx, dir, args...)` es el único punto de ejecución (directorio absoluto y existente, entorno saneado, `LC_ALL=C`, timeout 30 s, observador de pruebas). `gitcommon` es hoja (no importa ningún paquete interno). `sddpublish/gitport.go:161-173` (`IdentityConfigured`) usa `git var GIT_AUTHOR_IDENT`/`GIT_COMMITTER_IDENT`, solo para decidir si se puede commitear. Prueba de arquitectura `gitcommon/architecture_test.go:106-153` fija quién puede lanzar procesos (lista cerrada de paquetes: `branch/**`, `sddpublish`, `docpaths`, `techcontext`, `livespec`, `adoption`); un paquete nuevo no está en la lista pero la convención es usar `gitcommon.Run`.
- **Avisos.** El patrón vigente son variables de paquete `io.Writer = os.Stderr` con prefijo `Aviso: ` (`sealBranchWarnings` en `cli/sdd_kickoff_branch.go:32,55`; `writerBranchWarnings` en `cli/sdd_writer_guard.go:31,44`) que las pruebas sustituyen (`captureSealWarnings`, `captureWriterWarnings`). No hay campo `warnings` en el JSON de estos comandos: el aviso va solo a stderr y no ensucia el stdout JSON.
- **Specs vivas.** `rg` en `openspec/specs`: ninguna fija el valor `cli`. `sealed_by` solo aparece como `inferred` (`sdd-lane-kickoff/spec.md:96,129`; `sdd-preflight-configuration/spec.md:180`). Confirma que el requisito es ADDED y que no hay MODIFIED. `openspec/changes/identidad-git-actor/` solo contiene `kickoff.yaml` (`continuous`, `fullstack`, `sealed_by: cli`).
- **Formato y numeración de spec.** Spec viva en castellano: cabecera `# Especificación Viva: ...`, `## Propósito`, `## N. Capacidad: \`nombre\``, `### Requirement: Título (REQ-N.M)`, escenarios `#### Scenario:` con **DADO/CUANDO/ENTONCES/Y** (ejemplo: `openspec/specs/workspace-topology/spec.md`). No hay `INDEX.md` en `openspec/specs/`. Los prefijos de REQ en uso llegan hasta **REQ-41.x** (`openspec-adoption/spec.md:171`); no existe ningún REQ-42+ ni en `openspec/` ni en `docs/plan`. El siguiente prefijo libre es **REQ-42**. El plan y las olas hermanas (C-02-b, C-05-d, C-06-x...) pueden reclamar prefijos a la vez: riesgo de colisión al archivar (ver Riesgos).

### Áreas afectadas

- `internal/cli/sdd_gate.go:100-103` — sustituir el literal `cli` por la resolución (solo si `parsed.Actor == ""`); texto de ayuda `:48` (`--actor <id>`): indicar el valor por defecto.
- `internal/cli/sdd_kickoff.go:83` — tras `parsed.ToKickoff(...)`, y solo en la rama sin `--infer`, asignar `sealing.SealedBy` (igual que ya se asigna `sealing.Change`, `:99`).
- Nuevo `internal/gitidentity` (recomendado) o función en `gitcommon` — lectura y formateo de la identidad.
- `internal/kickoff/types.go:146,191` — solo comentario del campo (el tipo no cambia, como dice el plan). `args.go:375` — documentar que `cli` es el valor de reserva y que la identidad la completa el adaptador CLI.
- Pruebas: `internal/cli/sdd_gate_test.go`, `sdd_kickoff_test.go`, `sdd_kickoff_branch_test.go`; nuevas del paquete de identidad; `internal/gitcommon/architecture_test.go` si se añade el paquete a la lista escaneada.
- `openspec/specs/change-actor-identity/spec.md` (delta ADDED, capacidad nueva).
- No tocan: guías de agente (`internal/assets`, `agentguidance`: ningún `--actor` de gate), goldens (`testdata/golden`: los 15 ficheros con `actor`/`sealed_by` son reglas/instrucciones de agentes, ninguno fija el valor `cli` de los comandos; se confirma ejecutando `go test ./...` en apply), `sealed_by: inferred`, `integrations.yaml`.

### Pruebas existentes y aislamiento de git

- Ninguna prueba fija `actor == "cli"` ni `sealed_by == "cli"` como resultado. Fijan `Actor: "maintainer"` (kickoff `ledger_test`, `machine_test`, `handed_off_test`) o `--actor maintainer` (`cli/sdd_gate_test.go:36,50`; `:426` usa `release-manager`). `sealed_by: cli` solo aparece como entrada de fixtures (`kickoff/schema_test.go:17,55,310-315`, `kickoff/seal_test.go:20`, `cli/sdd_kickoff_test.go:253`): no se rompen.
- Pruebas que sí pueden romperse por el aviso: las que afirman cero avisos sobre `sealBranchWarnings` (`cli/sdd_kickoff_branch_test.go:220-233` y `cli/sdd_kickoff_test.go:592-604`). Por eso el aviso de identidad debe ir a **su propio** `io.Writer` y no reutilizar `sealBranchWarnings`/`writerBranchWarnings`. Sin identidad git (típico en CI) el aviso saldría en medio de esos tests.
- `TestMain` de `internal/cli` (`protocol_probe_test.go:52`) no aísla git. `gitcommon/main_test.go:17-76` y `branch/gitcli/ops_test.go:25` sí lo hacen (HOME/USERPROFILE a un temporal, `GIT_CONFIG_GLOBAL` a un `gitconfig` propio, `GIT_CONFIG_NOSYSTEM=1`, `GIT_TEMPLATE_DIR` vacío, identidad de autor por entorno). Patrones por prueba: `t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))` + `GIT_CONFIG_NOSYSTEM=1` (`cmd/axiom/change_create_branch_test.go:28-30`). `newGovernanceWorkspace` (`cli/sdd_kickoff_test.go:26-32`) crea un `t.TempDir()` que **no** es repo git: las pruebas nuevas de resolución deben hacer `git init` en un temporal (ya es habitual: `runGit(t, root, "config", "user.name", ...)`, p. ej. `sddpublish/gitport_test.go:337-338`, `sddstatus/edit_authority_test.go:31-32`) o inyectar una costura.
- En un temporal fuera de git, `git config user.name` aún lee el gitconfig global del desarrollador: hay que fijar `GIT_CONFIG_GLOBAL` (a un fichero vacío o con identidad) y `GIT_CONFIG_NOSYSTEM=1` en cada prueba que dependa de la ausencia o del valor.

### Enfoques

| Enfoque | Pros | Contras | Esfuerzo |
|---|---|---|---|
| **A. Paquete pequeño `internal/gitidentity`** (hoja; importa solo `gitcommon`): `Resolve(ctx, dir) Identity`, `Identity.Actor()` y saneado puro. El CLI lo llama en gate y seal | Reutilizable por C-07-e/C-07-h sin duplicar; saneado puro y testeable sin git; sin ciclos (`cli → gitidentity → gitcommon`); respeta que solo `gitcommon.Run` lanza git | Un paquete nuevo con poca superficie; conviene añadirlo a la lista de la prueba de arquitectura | Bajo |
| B. `gitcommon.ConfigValue(ctx, dir, key)` + formateo en `cli` | Sin paquete nuevo; lectura de config encaja con `TopLevel`/`CommonDir` | Mezcla en `gitcommon` (ejecución) la política de identidad si se pone el formateo allí; formateo en `cli` no reutilizable por otros paquetes | Bajo |
| C. Reutilizar/refactorizar `reviewAuditActor` | Cero lectura nueva | Semántica distinta (cadena de reserva y literal propios, vínculos hasheados LF-delimitados de RDD); `exec` directo; riesgo de tocar contratos de review por un cambio S | Medio, riesgo ajeno |
| D. Resolver dentro de `internal/kickoff` | Todo junto | `kickoff` no ejecuta git hoy; rompe la pureza del dominio y `ToKickoff` (usado por e2e) | Medio |

### Recomendación

Enfoque **A**. Diseño propuesto:

1. `internal/gitidentity`: lee con `gitcommon.Run(ctx, dir, "config", "--get", "user.name")` y `user.email` desde un directorio absoluto y existente (para gate, `loc.WorkspaceRoot`; para seal, también `loc.WorkspaceRoot`, porque `changeRoot` puede no existir aún). Resuelve configuración **de git** (local + global + `includeIf` según el directorio), no variables `GIT_AUTHOR_*` ni `git var` (que puede inventar `usuario@host`). Nunca devuelve error: git ausente, directorio no repositorio o clave vacía producen "sin identidad".
2. Formato: `"<user.name> <<user.email>>"` si ambos existen. Saneado: recorte de espacios, rechazo de caracteres de control (CR/LF) y retirada de `<`/`>` en el nombre (git hace lo mismo con su "crud"), con tope de longitud (256, como los actores de `reviewtransaction`).
3. CLI: variable `actorIdentityWarnings io.Writer = os.Stderr` propia; aviso en castellano, p. ej. `Aviso: git no tiene user.name y user.email configurados; se registra el actor "cli". Configúralos con git config o indica --actor.` Se emite solo cuando se usa la reserva `cli` en `gate record` sin `--actor`, y en `seal` sin `--infer`. `--actor` explícito no consulta git ni avisa. `seal --infer` no se toca.
4. `ToKickoff` sigue puro con `SealedBy: "cli"` como valor de reserva; el adaptador lo sobrescribe tras llamarlo.
5. Spec: capacidad nueva `change-actor-identity`, requisito ADDED (REQ-42.1 como propuesta), escenarios: identidad completa en gate; en seal; `--actor` prevalece; sin identidad conserva `cli` y avisa sin bloquear; `inferred` intacto; registros previos con `cli` siguen válidos; identidad con caracteres no seguros.
6. Estimación: S (≈150-250 líneas con pruebas), una PR (`single-pr`).

### Decisiones abiertas para propose/spec (no del usuario salvo que cambien el alcance)

1. **Identidad parcial.** Solo `user.email` o solo `user.name`: propuesta, tratarla como "sin identidad" (conserva `cli` + aviso que dice qué falta). Alternativa: registrar lo que haya. El plan solo define `Nombre <email>`.
2. **Fuente.** Solo configuración de git (propuesta), o también `GIT_AUTHOR_NAME/EMAIL`. La primera es determinista y coherente con "lo que tengan configurado".
3. **`seal` sin `--actor`.** El plan no añade `--actor` a `seal`; propuesta: no añadirlo (fuera de alcance).
4. **Campo `actor` en el JSON de `gate record`.** No requerido por el criterio de terminado; propuesta: no añadirlo.

### Riesgos

- **Colisión de REQ.** Otras hijas de la ola pueden elegir también REQ-42; al integrar la spec viva habrá que renumerar. Mitigación: confirmar el siguiente prefijo libre en `main` antes de archivar.
- **Pruebas no herméticas.** Sin aislamiento de `GIT_CONFIG_GLOBAL`, una prueba nueva pasa en la máquina del desarrollador y falla en CI (sin identidad) o al revés. Mitigación: aislar siempre el entorno en las pruebas de identidad y no reutilizar los writers de aviso existentes.
- **Aviso en la salida de otras pruebas** (CI sin identidad): inocuo si va a su propio writer; verificar que ninguna prueba compara stderr global.
- **Ruta de resolución en multirepo.** La identidad de git depende del repositorio (`includeIf`, config local). Se resuelve desde `loc.WorkspaceRoot` (planning home), que es donde se escribe el artefacto; documentarlo en el requisito.
- **Privacidad.** El email entra en artefactos versionados; ya es público en los commits (plan, riesgos de C-07-a). Mantenerlo anotado en el requisito.
- **Caracteres especiales en nombre.** `:`, `#`, comillas o `<>` en `user.name`: yaml.v3 escapa y se relee, pero hay que cubrirlo con una prueba (nombre con acentos y `#`).
- **Windows.** `gitcommon.Run` ya fija `LC_ALL=C` y salida UTF-8; los nombres con acentos pasan sin cambios; comprobar el round-trip con un nombre no ASCII.

### Listo para propuesta

Sí. Alcance S, una PR, sin decisiones de producto pendientes (las cuatro abiertas tienen propuesta por defecto). Capacidad nueva `change-actor-identity`, carril SDD continuo, rol `fullstack`.
