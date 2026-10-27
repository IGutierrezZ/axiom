# ODD: el ratchet de deadcode debe funcionar con Go 1.27

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/deadcode-ratchet-go127.md`, en el repositorio `axiom`. Worktree `C:\repos\axiom-wt\deadcode-ratchet-go127`, rama `fix/deadcode-ratchet-go127`.
> Espejo de recuperación en Engram: topic `odd/deadcode-ratchet-go127/tasks`, proyecto `axiom`.
> **Estado: En curso (2026-10-27).**
> Origen: seguimiento fuera de alcance del ODD `semantic-detector-codegraph` (`odd/tasks/semantic-detector-codegraph.md`) y del ODD `pruebas-aislamiento-host`.

## Objetivo

Que `bash scripts/deadcode-ratchet.sh` funcione en local con Go 1.27 sin dejar de funcionar en la CI (Go 1.25.10) y sin cambiar la línea base `.deadcode-baseline.txt`.

## Problema y porqué

- El script instala `golang.org/x/tools/cmd/deadcode@v0.30.0` (`scripts/deadcode-ratchet.sh:42`) en un directorio temporal y lo ejecuta con `GOOS=linux GOARCH=amd64`.
- Con Go 1.27.0 en local, el análisis entra en pánico dentro de `golang.org/x/tools@v0.30.0/go/ssa/builder.go:958`: `panic: unexpected expr: *ast.KeyValueExpr`. La v0.30.0 no reconoce la sintaxis que aparece en la biblioteca estándar de Go 1.27.
- La CI no lo ve: `actions/setup-go` lee `go-version-file: go.mod` (`go 1.25.10`, sin línea `toolchain`). El último `ci.yml` terminado en `main` (`8c4f59fa`) pasó.
- Consecuencia: la guarda no se puede verificar en local, y cualquier ODD que la liste como comprobación la deja como «no disponible».

## Evidencia (exploración, 2026-10-27)

- `main` `c8e7dbf2`, Go 1.27.0: `bash scripts/deadcode-ratchet.sh` → pánico reproducido.
- `main` `c8e7dbf2`, `GOTOOLCHAIN=go1.25.10`: `no new unreachable functions`. La causa es la combinación herramienta antigua + Go 1.27, no el código del repositorio.
- Directiva `go` de `golang.org/x/tools` (proxy): v0.43.0–v0.49.0 → `go 1.25.0`; v0.50.0–v0.52.0 → `go 1.26.0`. Con `GOTOOLCHAIN=go1.25.10`, `go install ...deadcode@v0.52.0` falla (`requires go >= 1.26.0`).
- `deadcode@v0.49.0`, análisis de `./cmd/axiom` con `GOOS=linux GOARCH=amd64`, normalizado como en el script:
  - Go 1.27.0: 248 entradas, 0 añadidas y 0 retiradas frente a la línea base.
  - Go 1.25.10: 248 entradas, 0 añadidas y 0 retiradas frente a la línea base.
- `deadcode@v0.52.0` con Go 1.27.0: también 0/0, pero no instala con el Go de `go.mod`.
- Cortex no bloqueó el `deadcode.exe` instalado por el script en un directorio temporal.

## Decisión

Fijar `deadcode@v0.49.0`: es la versión más reciente cuyo `go.mod` admite el Go que fija este repositorio (1.25), y funciona con Go 1.27. Así la CI no depende de que `GOTOOLCHAIN=auto` descargue otro toolchain. La línea base no cambia.

## Alcance autorizado

- `scripts/deadcode-ratchet.sh`: la versión fijada y un comentario breve que explique la elección.
- Este documento.
- **Fuera de alcance:** la línea base, el objetivo del análisis, la plataforma fijada, `go.mod` y los flujos de CI.

## Restricciones

- Commits en castellano, Conventional Commits, sin `Co-Authored-By` ni atribución de IA.
- Worktree dedicado; `C:\repos\axiom` siempre en `main` y sin escrituras.
- Veto del EDR: no compilar ni ejecutar binarios propios (`go build -o`, `go run ./cmd/axiom`).
- No pasar `go test ./...` completo ni e2e en local.
- Push, PR y merge (squash) solo con permiso del usuario; el repositorio no admite auto-merge.

## Modo de trabajo

- **TDD:** desactivado. Fuente: `openspec/config.yaml` (`strict_tdd: false`). Runner: `go test`. Comprobaciones funcionales: el propio ratchet.
- **RDD:** `axiom review mode status` → `off (decided by default)`. Sin evaluación de riesgo ni revisión nativa.
- **Entrega:** `ask-on-risk`. Previsión: menos de 60 líneas de autoría, un solo PR.

## Tareas

- [x] **T1 · Explorar y reproducir** — Ruta: en línea (1 script, `go.mod`, `ci.yml`). Evidencia arriba. Sin commit.
- [x] **T2 · Subir `deadcode` a v0.49.0** — Ruta: en línea (un fichero mecánico, ya entendido).
  - Cambiar `tool=` en `scripts/deadcode-ratchet.sh` y documentar por qué v0.49.0.
  - Comprobaciones: `bash scripts/deadcode-ratchet.sh` con Go 1.27.0 y con `GOTOOLCHAIN=go1.25.10`; `git diff --exit-code .deadcode-baseline.txt`; `go test ./cmd/axiom -run TestDeadcodeRatchetTargetsCanonicalBinary -count=1` (lee el script).
  - **Resultado (2026-10-27):** Go 1.27.0 → `no new unreachable functions`; `GOTOOLCHAIN=go1.25.10` → `no new unreachable functions`; `.deadcode-baseline.txt` sin cambios; `go test ./cmd/axiom -run 'TestCanonicalBinary|Deadcode' -count=1` → `ok`. RDD apagado, sin evaluación de riesgo. Commit de la unidad en la rama `fix/deadcode-ratchet-go127`.
- [ ] **T3 · Entregar** — push y PR con permiso del usuario; CI en verde.

## Criterios de aceptación

- `bash scripts/deadcode-ratchet.sh` → `no new unreachable functions` con Go 1.27.0 y con Go 1.25.10.
- `.deadcode-baseline.txt` sin cambios.
- CI en verde en el PR.

## Progreso

- 2026-10-27: T1 y T2 hechas. Siguiente: T3 (push y PR con permiso del usuario).
