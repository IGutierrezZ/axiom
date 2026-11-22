# Revisión de cobertura — spec de `aviso-commit`

Revisión informativa (`skills/_shared/sdd-coverage-review.md`): no bloquea fases, puertas ni archive, y no modifica la spec. El cambio está sellado `continuous`: los huecos van al resumen final y solo la persona puede pedir otra pasada de `sdd-spec`.

## Ronda 1 — 2026-11-22

- Entradas: `proposal.md` y los deltas `specs/sdd-block-review-gates/spec.md`, `specs/sdd-orchestrator-assets/spec.md` y `specs/organic-agent-trigger-rules/spec.md`.
- Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y de solo lectura. Los dos devolvieron un resultado válido.
- Unión de huecos, sin duplicados (14):

| # | Tipo | Ubicación | Hueco | Revisor |
|---|---|---|---|---|
| 1 | caso límite | REQ-33.2 | `--commit-review` sin valor, o repetida con valores distintos en la misma orden (error de análisis de argumentos); hoy solo se cubre la cadena vacía. | A |
| 2 | caso límite | REQ-33.6 | Un segundo registro `approved` (con o sin modo) sobre una puerta `tasks` ya aprobada con modo; qué ocurre con `commit_review` si la puerta vuelve a `pending` porque cambia el digest de tareas. | A |
| 3 | caso límite | REQ-33.4 | Proyección con varios registros de `tasks` (p. ej. `rejected` y luego `approved`) y con un `commit_review` inválido en el ledger, por ejemplo por edición manual: ¿`status` falla o se degrada a `notify`? | A |
| 4 | caso de uso | REQ-33.10 y REQ-33.9 (d) | En modo 3, la persona pide cambios en vez de aprobar; reanudación tras interrumpir la sesión con una tarea sin confirmar; pedir la PR con trabajo sin confirmar (¿qué cuenta como commits pendientes?). | A, B |
| 5 | caso límite | REQ-33.9 (f) | Criterio de «publicada» ambiguo: rama publicada en parte (sha objetivo dentro o fuera del remoto), sin remoto configurado, sin red o con la rama remota ya borrada. | A, B |
| 6 | error | REQ-33.9 (f) | El autosquash diferido falla por conflicto o deja la rebase a medias; `fixup!` sobre un sha que ya no existe; los sha avisados dejan de valer tras el autosquash y no se exige informarlo. | A, B |
| 7 | caso límite | REQ-33.9 (a) y REQ-33.12 | Tarea sin commit propio, tarea con varios commits, varias tareas en un mismo commit, o commit hecho por un worker delegado: qué sha se avisa. | A, B |
| 8 | caso límite | REQ-33.9 (a) y (g) | ¿Se avisa el propio commit `fixup!` o correctivo? En `stop` y `stop-before-commit`, ¿provoca una pausa propia o cuenta como tarea? | B |
| 9 | error | REQ-33.9 (a) y REQ-33.10 | El commit de una tarea falla (hook de pre-commit, nada que confirmar, commit vacío), también tras aprobar en modo 3. | B |
| 10 | caso límite | REQ-33.5 | Respuesta fuera del dominio, abandono o arrepentimiento en la pregunta del modo; no se exige que la aprobación espere a tener el modo (si no, cerraría la puerta con `notify` sin elegirlo); cambio `checkpointed` aprobado con un binario anterior y sin modo. | A, B |
| 11 | caso límite | REQ-33.2 | Kickoff ausente o ilegible al validar la bandera; en un cambio multirol, un único modo rige para todos los roles, pero la spec no lo dice. | B |
| 12 | requisito sin cubrir | REQ-33.2 | Ningún escenario verifica el JSON de salida de `gate record`: nombre de la clave y ausencia cuando no se persiste. | A |
| 13 | requisito sin cubrir | REQ-33.10 | Tras un traspaso (salida 2) no se exige avisar a quien recoge `apply` del modo heredado que eligió otra persona. | A |
| 14 | caso límite | REQ-33.12 | En ODD, cómo se registran en `odd/tasks/<feature>.md` los commits con cambios pedidos cuando el autosquash ya hizo desaparecer el sha original. | B |
