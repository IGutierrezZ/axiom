# ODD: Seguimientos posteriores a la retirada de los restos del upstream

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/seguimientos-post-restos-upstream.md`.
> Espejo de recuperación en Engram: topic `odd/seguimientos-post-restos-upstream/tasks`, proyecto `axiom`.
> Rama del documento: `docs/odd-seguimientos-post-restos-upstream`, subida a `origin`, con su worktree en `C:\repos\axiom-wt\odd-post-docs`. Este documento viaja **solo** en el PR de cierre.
> Abierto el 2026-10-08 desde `origin/main` `21b9808a`. Recoge los seguimientos de la sección 9 del ODD `seguimientos-restos-upstream` (cerrado con el PR #110).
> **Estado:** en ejecución desde el 2026-10-08 (segunda sesión). Decisiones S1, S2 y S6 tomadas.

---

## 0. Cómo retomar este ODD en una sesión nueva

1. Recupera el contexto:
   - `mem_context`;
   - `mem_search "odd/seguimientos-post-restos-upstream"` en el proyecto `axiom`;
   - `mem_get_observation` sobre la observación de ese topic;
   - **este fichero**, que es la fuente autoritativa: `C:\repos\axiom-wt\odd-post-docs\odd\tasks\` u `origin/docs/odd-seguimientos-post-restos-upstream`.
   - Como referencia de convenciones y lecciones, lee también `odd/tasks/seguimientos-restos-upstream.md` en `origin/main`, sobre todo las secciones 3, 4 y 9.
2. Comprueba el estado real:
   - `git -C C:\repos\axiom fetch --prune origin`;
   - `git worktree list`;
   - `gh pr list --state open`.
3. Concilia este documento con la realidad. Comprueba con `Grep` que cada punto de la sección 5 sigue existiendo y corrige las líneas si se han movido.
4. Plantea las decisiones pendientes de la sección 2, de una en una. Después continúa con la primera tarea sin marcar de la sección 6.

## 1. Objetivo

Cerrar los seguimientos que dejó el ODD anterior:
- un defecto de seguridad del arnés de `bench` en Windows;
- la versión que informa un binario instalado con `go install`;
- dos tests intermitentes bajo carga;
- los avisos que la TUI aún pierde en un flujo;
- un caso conservador de la retirada de prompts heredados;
- dos comentarios de marca obsoletos.

## 2. Decisiones del usuario

| Decisión | Valor |
|---|---|
| Carril | ODD (2026-10-08) |
| Entrega | PRs independientes contra `main`; el documento va en un PR de cierre |
| `size:exception` | Solo con aprobación explícita en cada PR y si no hay un corte cohesivo |
| S1: cómo arreglar `bench` en Windows | **(b)** (2026-10-08): los journeys con shims POSIX se marcan `unsupported` en Windows y el PATH del sandbox no incluye directorios del usuario. Su cobertura queda en el CI de Linux |
| S2: de dónde sale la versión de un binario instalado con `go install` | **(a)** (2026-10-08): primero el valor de ldflags, después `debug.ReadBuildInfo().Main.Version` si no es `(devel)` ni está vacío, y la constante como último recurso |
| S6: `PRD.md:1188` (tabla histórica del upstream) | **Se deja histórica** (2026-10-08): el PRD es entero el borrador del upstream (24 menciones a `gentle-ai`). Solo se corrigen los comentarios del ratchet |

## 3. Convenciones

Son las mismas de la sección 3 del ODD `seguimientos-restos-upstream`:
- **Idioma:** castellano peninsular en respuestas y artefactos; identificadores y comentarios Go en inglés.
- **Commits:** Conventional Commits en castellano, escritos a un fichero y usados con `git commit -F`, sin atribución de IA. El cuerpo del PR lleva la línea final `🤖 Generated with [Claude Code](https://claude.com/claude-code)`.
- **PR:** plantilla `.github/PULL_REQUEST_TEMPLATE.md` y una sola etiqueta `type:*`. La casilla de responsabilidad queda sin marcar.
- **Merge:** squash. No se consulta el CI en bucle: el usuario escribe «revisa si ya está verde y continúa». Tras cada merge se borran el worktree y las ramas.
- **Worktrees:** `C:\repos\axiom-wt\<nombre>`, siempre con **ruta absoluta** y desde `origin/main`. El checkout principal `C:\repos\axiom` es de otra sesión (inc-24) y no se toca.
- **ODD:** TDD desactivado (`openspec/config.yaml:16`) y runner `go test`. RDD desactivado.
  - El riesgo se mide con `axiom review assess --cwd . --base-ref origin/main --committed-only --json`; con `high` hace falta un verificador independiente de solo lectura.
  - Un writer cada vez, unos 60 minutos, tests por nombre exacto, como máximo 2 procesos y `sdd` en solitario. Todas las llamadas a `Agent` llevan `model: sonnet`.
- **Lecciones del ODD anterior:**
  - Si cambia la superficie de comandos, hay que pasar `gentle-ai-bench run` antes del PR.
  - Tras cada tanda de merges, verificar el `main` combinado.
  - Ante un fallo de CI que no se reproduce en local, poner una limpieza de diagnóstico.
  - El ratchet de código muerto ya analiza con `GOOS=linux` (#104).

## 4. Reglas de seguridad

Son las mismas de la sección 4 del ODD anterior:
1. Nunca matar procesos por nombre de imagen.
2. Nunca ejecutar `install`, `setup`, `sync`, `upgrade` ni `doctor` contra el HOME o el PATH reales.
3. Nunca ejecutar `go test ./internal/cli/` completo en local.
4. Para Linux: compilación cruzada y ejecución en una sola invocación de WSL. WSL no tiene Go instalado.
5. Nada de perl con rutas Windows en los reemplazos.
6. **Nuevo:** no ejecutar `gentle-ai-bench run` en Windows sin resolver antes S1, porque puede llamar a agentes reales del PATH.

## 5. Inventario (comprobado en `origin/main` `21b9808a`)

| ID | Punto | Estado y líneas | Clase | Tamaño | Riesgo |
|---|---|---|---|---|---|
| S1 | `bench` en Windows llama a agentes reales | Los journeys crean shims POSIX (`#!/bin/sh`) que Windows no ejecuta, por ejemplo `bench/journeys_provider_capture.go:48` (el directorio `provider-bin` está en `:43`), `journeys_issue_3043.go:20`, `journeys_wave3.go:59`, `journeys_issue3748.go:57`, `journeys_edge.go:338` y `record.go:94`. Conciliado el 2026-10-08: los dos últimos no estaban en el inventario de apertura. Si el shim no se ejecuta, el PATH del sandbox resuelve el `claude` real: en `j105` respondió «Not logged in». **Opciones:** (a) shims portables (`.cmd` o un binario Go de `bench`, como el `commandShim` de `bench/main.go:262`); (b) marcar `unsupported` en Windows los journeys con shims POSIX y garantizar que el PATH del sandbox nunca incluye directorios del usuario. | defecto (aislamiento) | M | medio-alto |
| S2 | Versión de un binario instalado con `go install` | `cmd/axiom/main.go:62` fija `var version = "v3.5.1"`. GoReleaser inyecta `-X main.version` (`.goreleaser.yaml:26-29`), pero `go install …/v3/cmd/axiom@vX` no lo hace, así que ese binario informa de la constante. Si no se sube en cada release, el aviso de actualización no se apaga. Desde #107 este camino vuelve a estar activo. **Opciones:** (a) si `version` no se inyectó, usar `debug.ReadBuildInfo().Main.Version`, que `go install @vX` sí rellena; (b) una comprobación de release que exija que la constante coincida con el tag. | defecto | S | medio |
| S3 | Tests intermitentes bajo carga | `internal/update/cooldown_concurrency_test.go:101` (`TestCheckAllWithCooldown_ContendedPersistenceRejectsReviewModeDisable`) y `internal/cli/review_next_transition_scope_recovery_test.go:35` (`TestRejectedTargetedValidatorCaptureRoutesEscalatedRecovery`). Fallan con `operation_timeout` cuando hay varios `go test` en paralelo y pasan en solitario. | deuda de tests | S-M | bajo |
| S4 | La TUI pierde los avisos del `sync` interno de la desinstalación con *clean install* | `internal/tui/model.go`: `SyncFunc` devuelve `SyncOutcome{Files, Warnings}` (`:472-475`), pero el flujo de `UninstallDoneMsg` (`:353-354`) solo usa los ficheros. Requiere añadir un campo a `UninstallDoneMsg` y un parámetro a `RenderUninstallResult`. | defecto | S-M | bajo |
| S5 | Un prompt heredado creado solo con SDD se conserva siempre | La prueba de propiedad de `internal/components/persona/legacy_prompt.go` solo reconoce el frontmatter del instalador de persona. Un `gentle-ai.md` o `gentle-ai.instructions.md` creado solo con el componente SDD lleva `instructionsFrontmatter` o `steeringFrontmatter` (`internal/components/sdd/inject.go:3287,3293`), así que se conserva con un aviso. Es conservador, pero deja el duplicado. Hay que reconocer esos frontmatter exactos sin relajar la lista de IDs de sección. | defecto | S | medio (borra ficheros del usuario) |
| S6 | Comentarios de marca obsoletos | `scripts/deadcode-ratchet.sh:33,39` hablan de `./cmd/gentle-ai` como si existiera. `PRD.md:1188` conserva `go install …/gentle-ai/cmd/gentle-ai@latest` en una tabla histórica del upstream (decisión pendiente). | marca | S | bajo |

## 6. Tareas (propuesta; se confirma tras las decisiones de la sección 2)

> Estado: `[ ]` pendiente · `[~]` en curso o con PR abierto · `[x]` fusionado. Ruta: delegada, con un writer por tarea en su worktree, salvo lo que sea mecánico y de un solo fichero.

- [ ] **P1 · `bench` aislado en Windows (S1, opción b).** Primero, un test que falle si el PATH del sandbox resuelve un ejecutable fuera de él. Después, se cierra el PATH y los journeys con shims POSIX se marcan `unsupported` en Windows. Riesgo previsto `high` (aislamiento y procesos), así que lleva verificador.
- [~] **P2 · Versión con `go install` (S2).** PR #111 (rama `fix/version-go-install`, commit `34492be0`). Ruta delegada: un writer `sonnet` en el worktree `p2-version`. Riesgo `medium` (`executable_change`, 95 líneas), así que basta la autoverificación del writer y una comprobación puntual del orquestador (`TestResolveVersion` y `TestVersionDefault` en verde). También se actualiza REQ-22.8 de la especificación viva `axiom-updater-resilience`. Opción (a) con test: la constante solo como último recurso, y `ReadBuildInfo` cuando no hay ldflags. Comprobar que `axiom version` y la comparación de versiones del actualizador siguen igual con GoReleaser.
- [ ] **P3 · Migración: avisos de desinstalación y prompts heredados solo con SDD (S4 + S5).** Comparten el ámbito de migración de instalaciones del ODD anterior (U3). Riesgo previsto `high` (borra ficheros del usuario), así que lleva verificador y repite el ataque de propiedad del ODD anterior.
- [ ] **P4 · Tests intermitentes y comentarios (S3 + S6).** Presupuestos o esperas por condición en los dos tests, sin ocultar fallos reales, y los comentarios del ratchet. `PRD.md` no se toca.
- [ ] **Cierre:** PR con este documento.

Previsión: unas 300-600 líneas en 4 PRs, todos por debajo de 400 líneas salvo que P1 crezca.

## 7. Criterios de aceptación

- `gentle-ai-bench run` en Windows no ejecuta ningún binario fuera del sandbox: o los journeys corren con shims portables o se declaran `unsupported` de forma explícita.
- Un binario instalado con `go install …@vX` informa de `vX` sin depender de la constante.
- El flujo de desinstalación con *clean install* de la TUI muestra los avisos de su `sync`.
- `sync` retira un prompt heredado gestionado creado solo con SDD, y sigue conservando con aviso cualquier fichero que el usuario haya editado.
- Los dos tests intermitentes pasan con varios `go test` en paralelo, o queda documentado por qué no.
- Ningún comentario vigente describe `cmd/gentle-ai` como si existiera.
- CI en verde en todos los PRs.

## 8. Progreso

- **2026-10-08, P2:** se abre el PR #111. `go test ./cmd/axiom/`, `go vet`, `gofmtcheck` y `deadcode-ratchet` en verde en local. `internal/update` ya tolera pseudo-versiones y `+dirty`, así que no se toca. Falta el CI.
- **2026-10-08, reanudación:** se concilia con `origin/main` `21b9808a`: no hay PRs abiertos y las líneas de S2 a S6 no han cambiado. En S1 aparecen tres shims ejecutables más, ya añadidos al inventario. El usuario decide S1 = (b), S2 = (a) y S6 = histórico. No fija orden, así que se empieza por P2 y después se sigue el de la sección 6.
- **2026-10-08, apertura:** se abre el documento desde `origin/main` `21b9808a`, en el worktree `odd-post-docs` y la rama `docs/odd-seguimientos-post-restos-upstream`. Los seis puntos se han comprobado con `rg` sobre `main`. Matiz de S2: GoReleaser ya inyecta `main.version`, así que el problema solo afecta a `go install`. Por decisión del usuario, la ejecución va en otra sesión.

## 9. Siguiente paso

En la sesión nueva:
1. Recuperar el contexto (sección 0).
2. Plantear, de una en una, las decisiones S1, S2 y S6 de la sección 2.
3. Empezar por P2, que es la tarea más pequeña y de mayor impacto para los usuarios de `go install`, salvo que el usuario prefiera otro orden.
