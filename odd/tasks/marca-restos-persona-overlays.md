# ODD: Restos de la marca del upstream en personas, eliminador heredado y overlays

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/marca-restos-persona-overlays.md`.
> Espejo de recuperación en Engram: topic `odd/marca-restos-persona-overlays/tasks`, proyecto `axiom`.
> **Estado: CERRADO el 2026-10-11.** V1, V2 y V3 se fusionaron en `main` (#130, #131 y #132). Este documento se cierra con el PR de V4.
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

**Respuestas de la exploración (2026-10-11, `main` `902b4487`):**
- F14: además de la línea de identidad, cada persona repite «Gentle AI» en la respuesta a «quién eres», en «Your name / identity» y en la frase de la memoria (gentleman 83, 105, 108 y 111; neutral 80, 90, 92 y 94). El único test que lo fija es `TestHermesPersonaAssetsContainIdentitySection` (`persona/inject_test.go:3102`). No hay goldens de Hermes. `docs/agents.md:304` documentaba la identidad heredada y los marcadores `gentle-ai:persona`, aunque hoy se escribe `axiom:persona`.
- F15: **no es preventivo, es un bug real.** `<!-- axiom:` es el prefijo canónico de escritura (`filemerge/section.go:107`) y lo usan persona, engram y skills-index, entre otros. Si al bloque bare solo le siguen secciones `axiom:`, el corte de `inject.go:3190` las borra hasta el final del fichero. El mismo defecto aparece en la rama con marcador (`:3159`), en `hasLegacyBareOrchestrator` (`:3118`) y en la guarda de `:3373`. No hay ningún test con un marcador `axiom:` después del bloque. Quedan fuera `removePiManagedSections` (limpieza heredada deliberada) y `communitytool/codegraph_guidance.go:131`.
- F16: el briefing del validador emite solo `axiom review inspect-candidate` (`reviewerprovider/contract.go:76`), y ningún código que no sea de test emite `gentle-ai review inspect-candidate`. Aun así, la regla se mantuvo a propósito: `opencode_v2_test.go:106`, caso «validator legacy command still allowed». El dialecto heredado del contrato (`cli/review_dialect.go:104-109`) sigue emitiendo `gentle-ai review …` para los clientes v1. También la fijan `sdd/inject_test.go:3712-3713`, `sdd/commands_test.go:53-54` y `testdata/golden/sdd-opencode-multi-settings.golden:156`. El validador no se renderiza en Kilocode (`inject.go:2445`), así que quitar la regla no cambia el hash de Kilocode.

## 5. Decisiones del usuario

- D0: carril ODD (heredado).
- D1 (2026-10-11): F16, **se quita la regla `gentle-ai review inspect-candidate`**. El caso «validator legacy command still allowed» se invierte: el comando heredado deja de estar permitido. Motivo: ningún código emite ese comando y el binario `gentle-ai` ya no existe.
- Alcance de V2 ampliado, dentro del criterio de aceptación de la sección 7: además del corte de `:3190`, cubre la rama con marcador (`:3159`), `hasLegacyBareOrchestrator` (`:3118`) y la guarda de `:3373`, porque tienen el mismo defecto.

## 6. Tareas (commit → PR → merge)

- [x] **V1 · F14 · Personas de Hermes:** cambiar la identidad a «Axiom running on Hermes Agent» en `gentleman` y `neutral`, y actualizar tests y goldens. Hace falta `axiom sync`.
  - Ruta: inline (4 ficheros mecánicos y ya entendidos). Commit `0fac8138` en `fix/hermes-persona-axiom` (worktree `axiom-wt/hermes-persona-axiom`). +18/−14.
  - Cambios: las 4 menciones por persona, el test cubre las tres personas y rechaza «Gentle AI», y `docs/agents.md:304` pasa a los marcadores `axiom:persona` y la identidad única.
  - Checks: `go build ./...` OK; `go test ./internal/components/persona/ ./internal/assets/` OK; `go test ./internal/components/ -run Golden` OK; `rg "Gentle AI running|identity: Gentle"` (fuera de `odd/`) sin resultados.
  - Riesgo: `medium` (`executable_change` por `docs/agents.md`). Con RDD desactivado basta la verificación del escritor. PR [#130](https://github.com/IGutierrezZ/axiom/pull/130) (`type:bug`), con CI en verde y fusionado por squash como `6211632d`.
- [x] **V2 · F15 · Eliminador heredado:** reconocer `<!-- axiom:` además de `<!-- gentle-ai:` como fin del bloque heredado, con tests de ficheros que mezclan los dos prefijos (antes, después y sin marcador). Sin cambiar los marcadores que se escriben.
  - Ruta: escritor delegado (sonnet). Commit `890a1801` en `fix/strip-legacy-orchestrator-axiom-markers` (worktree `axiom-wt/strip-legacy-axiom-markers`). `inject.go` +42/−6, `inject_test.go` +211.
  - Cambios: helpers `firstManagedMarkerIndex` y `sddOrchestratorMarkerIndex`, que devuelven el marcador más temprano de cualquiera de los dos prefijos. Se usan en `hasLegacyBareOrchestrator`, en las dos ramas de `stripBareOrchestratorForFilePrompt` y en la guarda de `injectMarkdownSections`. No cambia ningún marcador que se escribe hoy.
  - Pruebas: 8 tests nuevos. Antes del arreglo fallaban 6: la sección `axiom:engram-protocol` se perdía, la variante axiom antes que gentle-ai, las dos unitarias en su variante axiom, `hasLegacyBareOrchestrator` y la guarda.
  - Checks: `go build ./...` OK; `go vet` OK; `go test ./internal/components/sdd/` completo OK (174 s); Golden OK. Comprobación repetida por el padre (tests dirigidos) OK.
  - Riesgo: `medium` (`executable_change` en `inject.go`); RDD desactivado, escritor sonnet. PR [#131](https://github.com/IGutierrezZ/axiom/pull/131) (`type:bug`), con CI en verde y fusionado por squash como `e56ce5f2`.
- [x] **V3 · F16 · Regla de permiso sobrante:** quitar la regla `gentle-ai review inspect-candidate …` de los dos overlays y de `inject.go:1428`, salvo que la exploración demuestre que hace falta. Actualizar goldens y el hash de Kilocode si cambia. Hace falta `axiom sync`.
  - Ruta: escritor delegado (sonnet), según D1. Commit `9001581f` en `fix/opencode-validator-drop-gentle-ai-rule` (worktree `axiom-wt/opencode-validator-rule`). 7 ficheros, +5/−11.
  - Cambios: la regla se quita de los dos overlays, de `inject.go:1428`, de `inject_test.go:3713`, de `commands_test.go:54` y del golden `sdd-opencode-multi-settings.golden:156`. El caso de `opencode_v2_test.go:106` pasa a «validator legacy gentle-ai command no longer allowed» y espera `deny`. El hash de Kilocode no cambia.
  - Fuera, a propósito: `scripts/crosslane/opencode.go:357` (arnés que acepta las dos grafías en el prompt hijo; no es un permiso) y `bounded_review_contract_test.go:591` (solo lista la cadena como prohibida).
  - Checks: `go build ./...` OK; `go vet` OK; paquete sdd completo OK (150 s); Golden OK; `internal/assets` OK. Comprobación repetida por el padre (OpenCodeV2, Commands y Kilocode) OK.
  - Riesgo: `medium` (`configuration_change` en un overlay). PR [#132](https://github.com/IGutierrezZ/axiom/pull/132) (`type:bug`), con CI en verde y fusionado por squash como `b4bfd489`.
- [x] **V4 · Cierre:** actualizar este documento y abrir el PR `type:docs`.
  - Rama `docs/odd-marca-restos` (worktree `axiom-wt/odd-marca-restos-docs`) sobre `main` `b4bfd489`.

Previsión: unas 60-120 líneas en total. Real: unas 300 líneas de código y tests (V2 aporta 253, casi todas de tests), en tres PRs independientes contra `main`, todos por debajo de 400 líneas.

## 7. Criterios de aceptación

- Ningún asset instalado por Axiom presenta al agente como «Gentle AI», salvo las excepciones aceptadas (D10).
- El eliminador heredado limpia el bloque antiguo sin tocar secciones gestionadas con cualquiera de los dos prefijos.
- Los overlays de OpenCode solo autorizan comandos del binario `axiom`, salvo que se decida lo contrario.
- Build, tests dirigidos y goldens en verde; el CI de cada PR en verde antes de fusionar.

## 8. Progreso

- 2026-10-11: documento creado al cerrar `marca-seguimientos`, a petición del usuario, para lanzarlo desde otra sesión. Las ubicaciones de la sección 4 se comprobaron sobre `main` `60e75cb2`.
- 2026-10-11: lanzado. Se ha conciliado el estado (`main` = `origin/main` = `902b4487`, sin PRs abiertos). La exploración la hizo un subagente sonnet de solo lectura (disparador de 4+ ficheros); las respuestas están en la sección 4.
- Worktrees de trabajo: `axiom-wt/odd-marca-restos-docs` (`docs/odd-marca-restos`, este documento), `axiom-wt/hermes-persona-axiom` (V1), `axiom-wt/strip-legacy-axiom-markers` (`fix/strip-legacy-orchestrator-axiom-markers`, V2) y `axiom-wt/opencode-validator-rule` (V3). Ruta de V2: escritor delegado (disparador de escritura en `sdd/inject.go` + `inject_test.go`).
- 2026-10-11: D1 decidido (quitar la regla). V3 se encargó a un escritor delegado. Los PRs #130, #131 y #132 se abrieron con autorización del usuario.
- 2026-10-11: con «revisa si ya está verde y continúa», los tres PRs estaban `CLEAN` (checks requeridos en verde; Darwin, Windows y E2E omitidos) y se fusionaron por squash en orden.
  - #131 y #132 tocan los mismos ficheros. Antes de fusionar #132 se comprobó en local con `git merge-tree` que no había conflictos.
  - Verificación del `main` combinado (`b4bfd489`): `go build ./...` OK; `go test ./internal/components/sdd/` completo OK (84 s); `go test ./internal/components/persona/ ./internal/components/ -run 'Hermes|Golden'` OK.
  - Worktrees y ramas de V1, V2 y V3 borrados en local y en remoto.
- Seguimiento opcional, no abierto: `scripts/crosslane/opencode.go:357` todavía acepta `gentle-ai review inspect-candidate` en el prompt hijo del arnés. Es inocuo y no es un permiso.
