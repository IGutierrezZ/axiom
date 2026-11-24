# Revisión de cobertura · spec · verify-conjunto

Informe informativo (`skills/_shared/sdd-coverage-review.md`): no bloquea ninguna fase, compuerta ni archive, y no edita la spec. Unión deduplicada de los huecos de dos revisores ciegos (A y B), sin exigir acuerdo.

## Ronda 1 — 2026-11-23

Revisores: A (`jd-judge-a`) y B (`jd-judge-b`), ambos con resultado utilizable. Entradas: `proposal.md` y los tres deltas de `specs/`.

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | edge_case | `specs/change-branch-lifecycle/spec.md` · REQ-25.52 | Modo `release` con `develop` fusionado dentro de la rama del cambio antes de fusionarla en `develop`: el merge-base se desplaza y puede dejar fuera commits propios o mezclar ajenos. El tratamiento de commits ajenos solo se define para `branch`. | B |
| 2 | edge_case | `specs/change-branch-lifecycle/spec.md` · REQ-25.53 | Rama presente en local y como `origin/<rama>` con SHA distintos: sin regla de precedencia ni aviso de desfase; la referencia y el SHA solo se declaran cuando la rama existe únicamente como remota. | A, B |
| 3 | error | `specs/sdd-block-review-gates/spec.md` · REQ-43.32 | Si `axiom branch scope` falla al presentar `integration` (salida distinta de cero, sin instantánea, git ausente, contrato desconocido), ninguno de los cinco estados aplica y «exactamente uno» queda sin definir; tampoco se dice que el fallo no bloquee. | A, B |
| 4 | edge_case | `specs/sdd-block-review-gates/spec.md` · REQ-43.32 | «Hecha» se decide por la existencia de la sección: un verify global sin rol previo a las fusiones, o anterior a commits nuevos, se declararía «hecha» con precedencia máxima. Sin noción de ronda obsoleta. | A, B |
| 5 | uncovered_requirement | `specs/sdd-independent-review/spec.md` · REQ-28.4 | Alcance del verify sin rol lanzado por REQ-21.15 antes de fusionar (`merged=false`) y del verify de `fullstack`, que es a la vez rol y cambio: ¿diff del rol o alcance de `branch scope`? Rotulación frente a la ronda posterior en `integration`. | A, B |
| 6 | edge_case | `specs/sdd-independent-review/spec.md` · REQ-28.41 / REQ-28.42 | Mezcla de repositorios `ok`, `empty` y `unverifiable`: ambigüedad entre «no ofrecerla como completa» y ofrecerla parcial; veredicto de la sección parcial; estado que cambia entre la presentación y la ejecución; contenido de la sección cuando una petición explícita llega con todo `empty`/`exempt`. | A, B |
| 7 | error | `specs/change-branch-lifecycle/spec.md` · REQ-25.53 | Falta motivo `unverifiable` para la rama de integración o la base del modo ausentes (ni local ni descargada), base inválida en la instantánea o fallo de git. | A |
| 8 | edge_case | `specs/change-branch-lifecycle/spec.md` · REQ-25.54 | Contención de rama de rol en la rama del cambio: sin regla para squash (no ascendiente) ni para rama de rol borrada tras fusionarse (saldría `fetch_required` aunque esté fusionada). | A |
| 9 | edge_case | `specs/change-branch-lifecycle/spec.md` · REQ-25.53 | Rama del cambio borrada (local y remota) tras fusionarse en la integración —p. ej. `done` en `release` borra al fusionar en develop (C-02-a)—: saldría `fetch_required` con un fetch que no recupera nada. | A |
| 10 | error | `specs/change-branch-lifecycle/spec.md` · REQ-25.51 | Códigos de salida sin especificar (resultado con huecos frente a fallo del comando) y errores de entrada: cambio inexistente, kickoff ilegible, instantánea corrupta. | A |
| 11 | edge_case | `specs/change-branch-lifecycle/spec.md` · REQ-25.51 | Topología `monorepo-embedded` con instantánea sellada (entrada de código, exclusión de `openspec/changes/<cambio>/`, repositorio de specs `exempt` o no) y cambio sin repositorios de código. | A |

Sin autocorrección: en `continuous` los huecos pasan al resumen final y una nueva ejecución de `sdd-spec` la pide la persona.

## Ronda 2 — 2026-11-24

Revisores: A (`jd-judge-a`) y B (`jd-judge-b`), ambos con resultado utilizable. Entradas: `proposal.md` y los tres deltas de `specs/` tras la segunda ejecución de `sdd-spec` (decisión «SHA y luego mensaje», REQ-25.1/25.2 MODIFIED, REQ-25.55–25.57).

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | edge_case | `specs/sdd-independent-review/spec.md` · REQ-28.41 / `specs/sdd-block-review-gates/spec.md` · REQ-43.32 | Una ronda ejecutada con un repositorio `unverifiable` permanente (p. ej. `merge_not_located`) nunca tiene fusión completa `sí`, así que nunca es vigente ni «hecha» aunque la persona la acepte; choca con la nota de precedencia de REQ-43.32 («hecha aunque después aparezcan huecos»). | A, B |
| 2 | error | `specs/sdd-block-review-gates/spec.md` · REQ-43.32 | Con ronda previa, si `axiom branch scope` falla o el repositorio no tiene `tip` actual, la vigencia no se puede evaluar; la precedencia pone «hecha» antes que «no verificable» y no hay estado ni escenario para ese caso. | A, B |
| 3 | edge_case | `specs/change-branch-lifecycle/spec.md` · REQ-25.54 | Contención de rol cuando la rama del rol es la propia rama del cambio (`fullstack`, rama única): no se dice que sea trivialmente cierta ni cómo se reporta; riesgo de `role_not_contained` espurio. | A |
| 4 | edge_case | `specs/change-branch-lifecycle/spec.md` · REQ-25.51 | `scope` sobre un cambio ya archivado se rechazaría como `kickoff_not_sealed` (código engañoso); falta además la salida legible sin `--json`. | A |
| 5 | uncovered_requirement | `specs/change-branch-lifecycle/spec.md` · REQ-25.55 | El registro del SHA es una escritura nueva que la propuesta no recoge («sin estado de gobierno nuevo», rollback aditivo); falta actor en multirrol, fallo del registro, cierre sin acceso a la rama, roles cerrados antes de la capacidad (darían `fetch_required` con rama borrada) y worktree ya eliminado. | A, B |
| 6 | edge_case | `specs/sdd-independent-review/spec.md` · REQ-28.41 / REQ-28.42 | Verificación conjunta pedida explícitamente en `continuous` (sin compuerta `integration`): si se escribe la sección y cómo se distingue del verify de `fullstack`, que es el mismo informe; carpeta del informe cuando el `verify_file` vive en el repositorio de specs de un `multirepo`. | A, B |
| 7 | edge_case | `specs/change-branch-lifecycle/spec.md` · REQ-25.52 | Modo `branch` con la PR a la rama principal ya fusionada (verify tardío o reanudación): la rama es ascendiente de `main_branch` y sale `empty`/`nothing_to_verify` engañoso; sin motivo tipado «ya integrada en la base». | B |
| 8 | edge_case | `specs/change-branch-lifecycle/spec.md` · REQ-25.53 | Solo se resuelve `origin/<rama>`: no cubre remotos con otro nombre ni varios remotos; el `fetch` propuesto podría apuntar al remoto equivocado. | B |

Sin autocorrección: en `continuous` los huecos pasan al resumen final y una nueva ejecución de `sdd-spec` la pide la persona.
