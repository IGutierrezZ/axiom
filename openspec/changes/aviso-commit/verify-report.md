# Informe de verificación: aviso-commit (C-02-d)

Rol único `fullstack`. Alcance: `git diff 28084a56..HEAD` (7 commits, d5389cb5..cdac5184).

## Cumplimiento técnico

### Build y vet

- `go vet ./internal/kickoff/ ./internal/cli/ ./internal/sddstatus/ ./internal/components/... ./internal/assets/`: sin avisos (exit 0).

### Tests (sin compilar binarios de `axiom`)

| Orden | Resultado |
|-------|-----------|
| `go test ./internal/kickoff/ ./internal/sddstatus/ ./internal/assets/ ./internal/components/sdd/ ./internal/components/agentguidance/` | ok en los cinco paquetes (sddstatus 52 s; el resto en caché) |
| `go test ./internal/components/ -run TestGolden` | ok (en caché) |
| `go test ./internal/cli/ -run "Gate\|CommitReview\|Kickoff\|SDDStatus" -timeout 15m` | ok (resultado en caché de la ejecución de apply; no hay cambios de código posteriores) |

No apareció ningún fallo de entorno conocido en los paquetes ejecutados.

### Cobertura de requisitos por escenario

| Requisito | Evidencia en la implementación | Estado |
|-----------|-------------------------------|--------|
| REQ-33.1 vocabulario cerrado | `CommitReviewMode`, `CommitReviewModes()`, `validCommitReviewModes`; `splitCommitReviewFlag` rechaza `Stop`, `pause`, vacío, sin valor y repetida; `GateRecord.Validate` rechaza al leer | Cumple |
| REQ-33.2 `gate record` | `sdd_gate.go` y `sdd_gate_commit_review.go`: forma (solo `tasks`, `approved`, sin `--evidence-kind`), estado (`checkpointed`), comprobación bajo cerrojo (`gateStillPendingCheck`), campo ausente sin bandera (`omitempty`), JSON `commitReview`, con `--hand-off` en un único registro | Cumple |
| REQ-33.3 compatibilidad | `omitempty`; ledger sin campo da modo efectivo `notify`; pruebas en `internal/kickoff/commit_review_test.go` (decodificador no estricto, valor inválido rechazado) | Cumple |
| REQ-33.4 proyección | `status_v2.go`: `commitReview` solo con clave `tasks` y estado `approved`, vía `EffectiveCommitReview`; `resolveGateStatus` toma el primer `approved`; pruebas en `status_commit_review_test.go`; contrato `sdd-status-contract.md` actualizado | Cumple |
| REQ-33.5 interacción propia | Sección `Commit Review` («Asking the mode») y frase de enlace en Block Gates; pregunta tipada sin modificar | Cumple |
| REQ-33.6 inmutable | El primer `approved` cierra la puerta; un segundo registro con modo se rechaza; un rechazo posterior no cambia el modo | Cumple |
| REQ-33.7 ayuda | `renderSDDGateHelp` menciona la bandera, los tres tokens y la restricción | Cumple |
| REQ-33.8 fuente única | Sección en `sdd-orchestrator-sections.md`, marcador en los 12 orquestadores, goldens regenerados; pruebas de fuente única y deriva en verde | Cumple |
| REQ-33.9 contenido (a)-(h) | Presentes: aviso con sha, tarea y `git show`, tres modos, pausa que no es puerta, la PR valida, `fixup`/autosquash/commit correctivo, merge sin squash | Cumple |
| REQ-33.10 `sdd-apply` | Parámetros `commit_review` y `fixup_of`, `pending_commit`, retorno `commits`, línea en `## Commits con cambios pedidos`, en `model-capable` y `model-small` | Cumple |
| REQ-33.11 `work-unit-commits` | Sección «Review and Fixups» en `internal/assets/skills/` y `skills/`; prueba de paridad en verde | Cumple |
| REQ-33.12 ODD | `routing.go`: frase de aviso al final del paso 6 y viñeta «Commits and the pull request»; sin `stop`, `--commit-review` ni modos | Cumple |

### Restricciones cerradas

- `internal/kickoff/args.go` e `internal/cli/sdd_kickoff.go`: sin cambios en el diff.
- `routing_stops_test.go` y REQ-27.17: sin cambios (en `routing_test.go` solo hay adiciones).
- Pregunta tipada de la puerta `tasks`: sin cambios; conserva tres opciones y no menciona el modo.
- Kickoff y digest: no cambian (campo `omitempty`; pruebas de `sdd_gate_commit_review_test.go` comparan `gates.yaml`, digest y `kickoff.yaml`).
- RDD y recibos: ningún texto añadido de código o doctrina los menciona (sin coincidencias en las líneas añadidas).
- C-02-b y C-03-a: sin rastro en el diff.

### Estilo y conformidad con el design

- Cumple D2 (separación previa de la bandera sin tocar `args.go`), D3 (comprobaciones de forma y de estado con mensajes que nombran la orden válida), D4 (proyección informativa), D5 acotada (única excepción: añadir ` --commit-review <token>`), y D6, D7, D8 y D9 en doctrina y skills.
- Sin `commit -a`, `--force` ni `git push` en la doctrina nueva; todas las órdenes git con `git -C <worktree>`.

## Tareas

Las 21 tareas de `tasks.md` (1.1 a 6.2) figuran `[x]` y las 6 unidades tienen su commit. Sin tareas pendientes.

## Hallazgos

### CRITICAL

Ninguno.

### WARNING

Ninguno.

### SUGGESTION

1. `review_ledger_contract_test.go`: el hash fijado del orquestador de OpenCode que compone Kilocode cambió (`2e1fbb87...` a `a3a32474...`). Está comentado como deliberado y coincide con la sección nueva; conviene confirmarlo en la revisión de la PR.
2. La ejecución de `internal/cli` salió de la caché de Go; es válida (no hay cambios posteriores), pero no se midió el tiempo real.

## Contraste con los informes de cobertura (informativo)

`coverage-review.spec.md` (14 huecos):

- Resueltos: 1 (bandera sin valor o repetida, rechazada y probada), 12 (el JSON lleva `commitReview` y se prueba su ausencia sin modo); en parte 2 (un segundo `approved` se rechaza y el modo sigue siendo el del primero) y 3 (`status` toma el primer `approved`; un valor inválido se rechaza al leer el esquema).
- Siguen abiertos: 4, 5, 6, 7, 8, 9, 10, 11, 13 y 14 (casos límite de la doctrina de commits: reanudación, publicación parcial, fallo de autosquash, tarea con varios commits, fallo de commit, respuesta fuera de dominio, kickoff ilegible, aviso tras traspaso, ODD tras autosquash).

`coverage-review.design.md` (12 huecos):

- Resueltos: 1 (acotado por el gatekeeper, D8 y REQ-33.6) y 6 (`--commit-review=<valor>` no se reconoce como bandera y el error nombra el vocabulario).
- Siguen abiertos: 2, 3, 4, 5, 7, 8, 9, 10, 11 y 12 (publicación intermedia del objetivo de un `fixup!`, precondiciones del autosquash, PR fuera del orquestador, modo 3 con renombrados o hooks fallidos, multirepo con varios roles, petición de parada en el chat, coste de una tarea por lanzamiento, commits de remediación, salida de texto de `status` y `gate show`, aprobación sin orquestador).

## Veredicto

Implementación conforme a la spec y al design, con vet y tests en verde y las restricciones cerradas verificadas. Recuento: 0 CRITICAL, 0 WARNING, 2 SUGGESTION.

**Estado:** limpio
