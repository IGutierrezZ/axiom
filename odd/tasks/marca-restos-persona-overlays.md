# ODD: Restos de la marca del upstream en personas, eliminador heredado y overlays

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/marca-restos-persona-overlays.md`.
> Espejo de recuperación en Engram: topic `odd/marca-restos-persona-overlays/tasks`, proyecto `axiom`.
> **Estado: creado el 2026-10-11 y PENDIENTE DE LANZAR en otra sesión.** Todavía no hay rama de trabajo ni código cambiado.
> Origen: puntos F14, F15 y F16 del ODD `marca-seguimientos` (`odd/tasks/marca-seguimientos.md`, sección 4.2), que se dejaron fuera a propósito.

---

## 0. Cómo lanzarlo o retomarlo

1. Recupera el contexto: `mem_context`; `mem_search "odd/marca-restos-persona-overlays"` en el proyecto `axiom`; `mem_get_observation` sobre la observación de ese topic. Lee **este fichero**, que es la fuente autoritativa (en `main` cuando se fusione el PR de cierre de `marca-seguimientos`, #129; antes, en la rama `docs/odd-marca-seguimientos`).
2. Concilia con la realidad: `git -C C:\repos\axiom fetch --prune origin`, `git -C C:\repos\axiom worktree list` y `gh pr list --repo IGutierrezZ/axiom --state open`. Comprueba que las líneas citadas en la sección 4 siguen siendo válidas en `origin/main`.
3. Carril ya elegido: **ODD** (heredado de `marca-seguimientos`, D0). No vuelvas a preguntar el carril.
4. Empieza por la sección 6 (primera tarea sin marcar) y respeta las secciones 2, 3 y 7.

## 1. Objetivo

Retirar los últimos restos de la marca del upstream que quedaron fuera de `marca-seguimientos` porque tocan comportamiento y no solo texto: la identidad de las personas de Hermes, el eliminador heredado del orquestador y una regla de permiso sobrante en los overlays de OpenCode. Hay que mantener siempre la compatibilidad con las instalaciones existentes.

## 2. Alcance y restricciones

- **Dentro:** F14, F15 y F16 (sección 4).
- **Fuera:** IDs de contrato, marcadores `<!-- gentle-ai:… -->` que se escriben hoy, rutas, variables de entorno, orígenes de backup y detección heredada, salvo lo que exija F15 de forma explícita. Los títulos de schema publicados con «Gentle AI» quedan como excepción aceptada (D10 de `marca-seguimientos`).
- `contracts/review-integration/v1` está congelado (FREEZE.md).
- Se conserva la atribución veraz (CONTRIBUTORS.md).
- Un cambio en `internal/assets/**` cambia `assets.ManagedDigest()`: hace falta `axiom sync`, y hay que avisarlo en el PR. También cambia el hash fijado de Kilocode en `internal/components/sdd/review_ledger_contract_test.go` cuando el asset llega a Kilocode.
- Push, PR y merge solo con autorización explícita del usuario en cada caso.

## 3. Convenciones (heredadas de `marca-seguimientos`, sección 8.2)

- TDD desactivado (`openspec/config.yaml strict_tdd: false`); runner `go test -run`. Nunca `go test ./internal/cli/` completo; `internal/components/sdd` con `-timeout 40m`.
- RDD desactivado (configuración global). Riesgo por PR con `axiom review assess --cwd . --base-ref origin/main --committed-only --json`; con `high`, verificador independiente de solo lectura.
- Un PR por unidad contra `main`, con la plantilla en castellano, una sola etiqueta `type:*` y `size:exception` solo si la aprueba el usuario. Commits en Conventional Commits, en castellano y sin atribución. Merge por squash cuando el usuario diga «revisa si ya está verde y continúa»; nada de auto-merge ni de consultar el CI en bucle.
- Goldens: `go test ./internal/components/ -run Golden -update`.
- Subagentes con `model: sonnet`.
- Fallos de entorno conocidos en Windows: symlinks sin privilegio y `unlinkat … in use`.

## 4. Inventario (comprobado sobre `main` `60e75cb2` el 2026-10-11)

| ID | Qué | Dónde | Lo fija | Riesgo |
|---|---|---|---|---|
| F14 | Las personas de Hermes `gentleman` y `neutral` se presentan como «You are **Gentle AI running on Hermes Agent**». La persona `axiom` ya dice «Axiom running on Hermes Agent». | `internal/assets/hermes/persona-gentleman.md:105`, `internal/assets/hermes/persona-neutral.md:90` | `internal/components/persona/inject_test.go` (~3100) y goldens de Hermes, si los hay | bajo |
| F15 | `stripBareOrchestratorForFilePrompt` termina el bloque heredado en el siguiente `<!-- gentle-ai:` y no reconoce `<!-- axiom:`. Si algún marcador gestionado empieza por `<!-- axiom:`, el bloque heredado podría comerse esa sección hasta el final del fichero. | `internal/components/sdd/inject.go:3190` (función desde ~3158) | tests de `inject_test.go` del eliminador heredado | medio |
| F16 | Queda una regla de permiso `gentle-ai review inspect-candidate --purpose targeted-validation *` junto a la de `axiom`. No rompe nada, pero el binario `gentle-ai` ya no existe. | `internal/assets/opencode/sdd-overlay-single.json:224`, `internal/assets/opencode/sdd-overlay-multi.json:224`, `internal/components/sdd/inject.go:1428` | goldens de OpenCode y tests de overlays | bajo |

**Preguntas abiertas para la exploración inicial:**
- F15: ¿existe hoy algún marcador gestionado `<!-- axiom:` (Axiom escribe `<!-- gentle-ai:` + `sectionID` en `inject.go:3265`)? Si no existe, F15 es preventivo y conviene reconocer los dos prefijos sin cambiar los que se escriben.
- F16: ¿la regla `gentle-ai` sigue haciendo falta para una ventana de compatibilidad, por ejemplo para transacciones de revisión en curso emitidas por un binario antiguo? Si es así, documentarlo y decidir con el usuario.

## 5. Decisiones del usuario

- D0: carril ODD (heredado).
- (pendientes) La decisión de F16 sobre la compatibilidad, si la exploración la plantea.

## 6. Tareas (commit → PR → merge)

- [ ] **V1 · F14 · Personas de Hermes:** cambiar la identidad a «Axiom running on Hermes Agent» en `gentleman` y `neutral`, y actualizar tests y goldens. Hace falta `axiom sync`.
- [ ] **V2 · F15 · Eliminador heredado:** reconocer `<!-- axiom:` además de `<!-- gentle-ai:` como fin del bloque heredado, con tests de ficheros que mezclan los dos prefijos (antes, después y sin marcador). Sin cambiar los marcadores que se escriben.
- [ ] **V3 · F16 · Regla de permiso sobrante:** quitar la regla `gentle-ai review inspect-candidate …` de los dos overlays y de `inject.go:1428`, salvo que la exploración demuestre que hace falta. Actualizar goldens y el hash de Kilocode si cambia. Hace falta `axiom sync`.
- [ ] **V4 · Cierre:** actualizar este documento y abrir el PR `type:docs`.

Previsión: unas 60-120 líneas en total. Se espera un PR por unidad, independientes contra `main`.

## 7. Criterios de aceptación

- Ningún asset instalado por Axiom presenta al agente como «Gentle AI», salvo las excepciones aceptadas (D10).
- El eliminador heredado limpia el bloque antiguo sin tocar secciones gestionadas con cualquiera de los dos prefijos.
- Los overlays de OpenCode solo autorizan comandos del binario `axiom`, salvo que se decida lo contrario.
- Build, tests dirigidos y goldens en verde; el CI de cada PR en verde antes de fusionar.

## 8. Progreso

- 2026-10-11: documento creado al cerrar `marca-seguimientos`, a petición del usuario, para lanzarlo desde otra sesión. Las ubicaciones de la sección 4 se comprobaron sobre `main` `60e75cb2`.
