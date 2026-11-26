# Revisión de cobertura del design · autofetch-specs

Revisión informativa (`skills/_shared/sdd-coverage-review.md`): no bloquea ninguna fase y no edita los artefactos. Unión deduplicada de dos revisores ciegos (`jd-judge-a`, `jd-judge-b`), sin exigir acuerdo.

## Ronda 1 — 2026-11-26

Entradas: `proposal.md`, `specs/change-branch-lifecycle/spec.md`, `specs/multi-project-hub/spec.md` y `design.md`. Los dos revisores devolvieron resultado utilizable.

| # | Tipo | Ubicación | Hueco | Base | Revisores |
|---|---|---|---|---|---|
| 1 | uncovered_requirement | design.md#D-10, R-6 | RF-BRA-40 pide que un `axiom sync` posterior que ya no deba instalar el hook lo retire; el design solo cubre la desinstalación y no hay tarea ni prueba para `sync`. | RF-BRA-40: «una ejecución posterior de `axiom sync` que ya no deba instalarlo, y la desinstalación, MUST retirarlo». | A, B |
| 2 | uncovered_requirement | design.md#D-04, R-3 | El fetch implícito de `ensure` excluye el repositorio de specs in situ de un multirepo, contra el literal «en todos los repositorios del plan». | RF-BRA-10. | A, B |
| 3 | error | design.md#D-09, D-10 | Presupuesto de `run` de 12 s y timeout del hook de 15 s frente a «el arranque de la sesión no espera más que el tiempo máximo del fetch» (~8 s). | RF-BRA-40, escenario «El hook termina siempre en 0 y no bloquea». | A, B |
| 4 | edge_case | design.md#D-08, D-09 | Los avisos de no-avance (`integration_dirty`, `integration_not_deployed`, `integration_diverged`) salen aunque no haya nada relevante; con el checkout principal sucio habitual saldrían en cada sesión. No se concilia con «Sin nada relevante, `run` MUST NOT emitir ningún aviso». | RF-BRA-39 paso 3; RF-BRA-40. | A |
| 5 | use_case | design.md#D-07, R-4 | Sin rama de cambio en `HEAD` y con varios cambios activos, o al recoger un cambio cedido sin registro ni rama local, no se resuelve cambio actual y no se avisa de lo relevante, que es el valor central. Con rama remota y sin local, el aviso (b) se omite. | Propuesta, Intención: «cesiones (`axiom sdd publish`) que la siguiente persona no detecta». | A, B |
| 6 | uncovered_requirement | design.md#D-11, R-2 | Otras frases de la doctrina siguen prohibiendo al agente ejecutar `fetch` (`sdd-orchestrator-sections.md:108`, `:288`, `generic/stop-exits.md`); la contradicción se resuelve solo con una excepción y la delta de `sdd-block-review-gates` queda sin decidir. | RF-BRA-40; propuesta: «`sdd-block-review-gates` (condicional)». | A |
| 7 | edge_case | design.md#D-04 | Fetch implícito de `ensure` sin definir para `--base <ref>`, repositorio sin `origin`, coincidencia con `decision_required`, y coste de 30 s por repositorio con red caída. | RF-BRA-10; RF-BRA-12. | A |
| 8 | edge_case | design.md#D-03, D-09 | Identidad del proyecto desde un worktree de feature fuera de la carpeta maestra, puntero `.axiom-workspace` en multirepo, y normalización de rutas (`EqualFold`) igual que `Register` para `LookupAutofetch`/`SetAutofetch`. | RF-HUB-13; RF-HUB-07; RF-BRA-40. | A, B |
| 9 | edge_case | design.md#D-02, D-08 | Integración desplegada en un worktree distinto, desacoplado, bloqueado o en mitad de un merge/rebase/cherry-pick; la orden sugerida `fetch . <origin>:<int>` no comprueba árbol; rama local `<int>` inexistente con `origin/<int>` presente. | RF-BRA-39 paso 2 y prohibiciones. | A, B |
| 10 | uncovered_requirement | design.md#R-1 | `ensure` consulta el interruptor pero no emite el aviso de proyecto no registrado que pide RF-HUB-07 («con aviso en las órdenes que lo consultan»). | RF-HUB-07. | B |
| 11 | uncovered_requirement | design.md#D-05 | Con `main_branch` igual a `integration_branch` (monorepo de una etapa) `run` avanza la rama principal, contra el literal «MUST NOT avanzar `main_branch` en cualquier circunstancia» (la decisión P14 del usuario sí ordena avanzar la integración del repo de specs). | RF-BRA-39, Prohibiciones. | B |
| 12 | edge_case | design.md#D-09 | Sin definir qué emite el hook con proyecto no registrado o hub ilegible: inerte (RF-BRA-40) frente a «con aviso» (RF-HUB-07). | RF-BRA-40; RF-HUB-07. | B |

Tratamiento (`continuous`): la doctrina no relanza fases por huecos de cobertura. Los huecos 3, 10, 11 y 12 son además contradicciones internas entre spec y design o deriva de la spec respecto de la decisión P14. El gatekeeper los trata como fallo de compuerta y relanza una vez `sdd-spec` y `sdd-design` con corrección acotada. Los demás huecos se trasladan a `sdd-tasks` o quedan para el resumen final.
