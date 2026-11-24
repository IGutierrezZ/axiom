# Revisión de cobertura · design · verify-conjunto

Informe informativo (`skills/_shared/sdd-coverage-review.md`): no bloquea ninguna fase, compuerta ni archive, y no edita los artefactos. Unión deduplicada de los huecos de dos revisores ciegos (A y B), sin exigir acuerdo.

## Ronda 1 — 2026-11-23

Revisores: A (`jd-judge-a`) y B (`jd-judge-b`), ambos con resultado utilizable. Entradas: `proposal.md`, los tres deltas de `specs/` y `design.md`.

| # | Tipo | Artefacto · ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | uncovered_requirement | spec · `change-branch-lifecycle` REQ-25.51 | Falta MODIFIED de REQ-25.1 (exactamente `ensure`, `status`, `done`, `guard`) y de REQ-25.2: el verbo `scope` contradice la spec viva y sus pines de ayuda y docs (P1 del design). | A, B |
| 2 | uncovered_requirement | spec · `change-branch-lifecycle` REQ-25.53 | El dominio cerrado de motivos no recoge lo que el design necesita: `fetch_required` para base o integración ausentes, `repository_unavailable`, `scope_not_applicable`; el escenario «sin commits propios → `empty`» no distingue modo frente al *fast-forward* resuelto por reflog (P2, P3). | A |
| 3 | uncovered_requirement | design · D12 / `joint-verification.md` | Degradación de `sdd-verify` sin rol cuando `scope` no aplica (sin instantánea, kickoff sin `git:`, carril ODD, rechazo): REQ-28.4 exige comportarse como antes y declararlo en el informe; el design no lo recoge en la referencia ni en `report-format.md` ni en pines. | A, B |
| 4 | uncovered_requirement | design · D12 | REQ-28.4: el alcance solo delimita lo inspeccionado; la suite completa de REQ-21.15 no se acota. No aparece en el design ni tiene pin. | A |
| 5 | edge_case | design · D12 / D13 | El verify global sin rol (REQ-21.15) se habilita antes de las fusiones: no se define si escribe la sección «Verificación conjunta» (y se declararía «hecha» con precedencia máxima) ni qué hace una petición explícita con ramas pendientes o repos `unverifiable` (¿parcial o rechazo?). | A, B |
| 6 | edge_case | design · D13 (`continuous`) | REQ-28.42: en `continuous` el resumen final indica que el verify de `fullstack` cubre el cambio y usa el mismo alcance; D13 dice «nada nuevo» y no define cómo consume `scope` un verify con rol `fullstack`, qué alcance prevalece con la rama sin fusionar, ni el texto y su pin. | A, B |
| 7 | edge_case | design · D3 | Clon superficial (*shallow*), ramas de rol bifurcadas de la rama del cambio e historia reescrita tras la fusión: riesgo de punto de partida erróneo presentado como `ok`. | A |
| 8 | edge_case | design · D11 frente a REQ-25.54 | Repositorio `empty` con rama de rol ausente o no contenida: podría dar `pending_merge` y ocultar la oferta; precedencia entre `nothing_to_verify` y pendientes sin fijar. | A |
| 9 | error | design · D4 / P4 · D13 | Rama del cambio o de rol borrada tras fusionarse (borrado automático de la PR o `axiom branch done`): `fetch_required` con un fetch inútil, sin motivo tipado distinto; el cambio queda no ofrecible de forma permanente. | A, B |
| 10 | edge_case | design · D6 / D11 | Roles `deferred` u `optional` sin rama creada cuentan como `fetch_required` pendiente para siempre; no se distingue rol sin rama esperada de rama borrada. | B |
| 11 | edge_case | design · D12 / D13 frente a REQ-43.32 | D13 habla de «ronda desactualizada» (tips distintos), pero REQ-43.32 declara «hecha» con precedencia máxima sin matiz; sin formato ni escenario ni prueba. | B |
| 12 | error | design · D7 / D10 frente a REQ-43.32 | Fallo de la propia orden (`git_failed`, `config_invalid`, salida distinta de cero) al presentar `integration`: el mapeo a uno de los cinco estados solo está en prosa de D13; la spec define «no verificable» solo por repositorio `unverifiable`. | B |

Sin autocorrección: en `continuous` los huecos pasan al resumen final y una nueva ejecución de `sdd-spec` o `sdd-design` la pide la persona.

## Ronda 2 — 2026-11-24

Revisores: A (`jd-judge-a`) y B (`jd-judge-b`), ambos con resultado utilizable. Entradas: `proposal.md`, los tres deltas de `specs/` y `design.md` tras su segunda ejecución (D1–D15).

| # | Tipo | Artefacto · ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | uncovered_requirement | design · D12 / D13 frente a REQ-43.32 | El design evalúa solo la última ronda; la spec declara «hecha» si existe alguna ronda vigente. Una ronda posterior desactualizada (petición explícita parcial) ocultaría una vigente anterior; no se dice qué ronda prevalece. | A |
| 2 | edge_case | design · D7 frente a REQ-25.55 | `axiom sdd tips record` escribe `branch-tips.yaml` después del último commit del rol y lo deja sin commit hasta `axiom sdd publish`: árbol sucio frente a `dirty_worktree` de `ensure`/`done` en `in_place`, y el registro no llega a otra máquina si nadie publica (`continuous` sin `docsPublication: auto`). | A, B |
| 3 | uncovered_requirement | design · D8 / P5 frente a REQ-25.54 | Rama del cambio fusionada por squash y borrada (localizada por `message`) con una rama de rol también borrada: la contención sale `unknown` y la oferta queda en `pending_merge` permanente, flujo habitual en ADO. | A, B |
| 4 | edge_case | design · D13 (vigencia) / D6 | La identidad del `tip` cambia con la localización: una ronda hecha con la rama presente queda desactualizada para siempre si después la rama se borra y se localiza por mensaje (el `tip` pasa a ser el commit de squash). | B |
| 5 | uncovered_requirement | spec · REQ-25.55 | El verbo `axiom sdd tips record` (banderas, rechazos, contrato `axiom.sdd-branch-tips-record/v1`, guarda A, cerrojo) no tiene requisito ni escenario en ninguna spec (P1 del design). | B |
| 6 | edge_case | design · D3 / P2 frente a REQ-25.51/25.53 | Los avisos `already_in_base`, `tree_layer_unsupported` y `branch_tips_unreadable` que emite el design no figuran en el dominio de avisos de la spec; modo `branch` con PR a la rama principal ya fusionada queda sin resolución tipada. | B |
| 7 | edge_case | design · D13 (estado «ofrecida y declinada») | Estado solo de sesión: al retomar el cambio en otra sesión o máquina vuelve a «ofrecida»; no se define qué ocurre si se declina y después se fusionan más ramas. | B |

Sin autocorrección: en `continuous` los huecos pasan al resumen final y una nueva ejecución de `sdd-design` la pide la persona.

## Ronda 3 — 2026-11-24

Revisión final única sobre la versión simplificada (decisión del usuario del 2026-11-24): `proposal.md` alineada, los tres deltas reescritos y `design.md` (D1–D13). Revisores: A (`jd-judge-a`) y B (`jd-judge-b`), ambos con resultado utilizable. Lo declarado «Fuera de alcance» no se trata como hueco.

| # | Tipo | Artefacto · ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | uncovered_requirement | spec · REQ-25.53 / design · P1 | Repositorio de la instantánea ausente en esta máquina: se clasifica `branch_missing` con `details.path`, pero el mensaje obligatorio («la rama ya no está o falta un fetch autorizado») y la continuación `fetch` no le encajan; sin prueba prevista. | A, B |
| 2 | edge_case | design · D11 / D12 frente a REQ-43.32 | El orquestador busca rondas en «el informe consolidado» sin ruta fija en `multirepo` (D11 sí fija la carpeta del cambio del planning home); tampoco se dice qué estado se declara si el informe no existe o no se lee. | A, B |
| 3 | edge_case | design · D5 frente a REQ-25.52 | La condición del aviso `foreign_commits_possible` es imprecisa (puede no saltar con develop fusionada en la rama o saltar tras un avance rápido sin ajenos) y no hay prueba que la fije. | A |
| 4 | uncovered_requirement | design · D7 frente a REQ-25.53 | `ok` exige `files` no vacío: una rama con commits propios sin ficheros listados (vacíos o solo de fusión) saldría `empty`, que la spec reserva a «sin commits propios» o «solo `openspec/changes/<cambio>/`». | B |
| 5 | uncovered_requirement | spec · REQ-25.51 / REQ-25.2 | El token nuevo `scope_not_applicable` no se añade a la tabla de códigos de la capacidad ni tiene etiqueta en castellano fijada. | B |

Sin autocorrección: en `continuous` los huecos pasan al resumen final y una nueva ejecución de `sdd-spec` o `sdd-design` la pide la persona.
