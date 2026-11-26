# Revisión de cobertura del diseño — plantilla-ramas

Revisión informativa: no bloquea fases, compuertas ni archivado, y no cambia ningún digest.

## Ronda 1 — 2026-11-26

Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y de solo lectura. Entradas: `proposal.md`, los tres deltas de `specs/` y `design.md`; ambos contrastaron con el código. Los dos dieron un resultado utilizable. El informe es la unión deduplicada.

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | uncovered_requirement | D-10 | Sin registro, la guarda no renderiza la plantilla de la instantánea con los vínculos versionados: una rama correcta con `{task_ref}` (`feature/TASK-1235`) saldría `wrong_branch` (RF-BRA-19, «La rama de plantilla se reconoce sin registro»). Tampoco se dice cómo se obtienen las ramas de rol de una entrada pendiente. | A, B |
| 2 | uncovered_requirement | D-10 / D-07 | No se define el veredicto `not_applicable` con detalle y continuación `ensure --role <rol>` de un repositorio cuya única rama está pendiente, ni cómo se distingue de una rama inexistente, ni el caso de un repositorio con un rol ya dentro y otro pendiente. | A, B |
| 3 | uncovered_requirement | D-04 / ficheros | Siguen sin tratar las composiciones `prefijo/cambio` y `-archive` de `planner_archive.go:150-160`, `engine/archive_plan.go:70-72`, `engine/autofetch_relevance.go:161` y `verdict.go:296`; tampoco se decide cómo interactúa `--prefix` con una plantilla al archivar. | A |
| 4 | edge_case | D-13 / D-11 | Con `integrations.yaml` ilegible y plugin activo solo se falla cerrado para `{ref}` y `{task_ref}`; con `{slug}` o `{kind}` se calcularía un nombre con valores incompletos. | A |
| 5 | edge_case / error | D-09 | La recuperación sin registro solo cubre pendientes con `{task_ref}`. Faltan: plantillas sin `{task_ref}` (ODD o sin instantánea) con la plantilla cambiada después; `integrations.yaml` ilegible; Task desvinculada; dos candidatos de roles distintos que coinciden; un `BranchExists` que falla por error y no por ausencia. | A, B |
| 6 | edge_case | D-02 | La regla de la rama con `openspec/` en multirepo toma la entrada del repositorio de specs «si existe», cuando la spec dice «si existe y declara `branch_template`»; una entrada que solo fija `branch_prefix` dejaría pasar un `git:` con `{task_ref}`. Falta cómo se identifica esa entrada al cargar. | A, B |
| 7 | use_case | Preguntas abiertas / D-13 | Multirepo con el rol único `fullstack` y `{task_ref}` heredado: no queda definido quién invoca `ensure --role fullstack` ni dónde se vincula su Task. | A |
| 8 | edge_case | D-05 | Mezcla de instantánea sin nombre y registro sin tratar en las pruebas, y el orden nuevo `--branch` > instantánea cambia el D-10 actual sin escenario de spec que lo pida. | A |
| 9 | edge_case | D-13 | Contradicción con prov:KOF:1: un cambio sellado con plantilla pero sin `template_values` debe dar `{kind}` = `feature`, y el diseño lo sacaría del perfil adoptado (`fix`). | B |
| 10 | edge_case | D-06 / D-13 | `{kind}` sin work item (cada clon lo resolvería por su cuenta) y propagación del rechazo `branch_template_unresolved` desde `kickoff seal` sin escribir `kickoff.yaml`. | B |
| 11 | uncovered_requirement | D-04 / D-05 / D-11 | No se dice dónde se detecta la colisión `invalid_name` ni si cuenta las ramas pendientes; dos roles con el mismo `{task_ref}` solo chocarían al entrar el segundo. | B |
