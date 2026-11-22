# Revisión de cobertura — spec y design de `aviso-commit`

Revisión informativa (`skills/_shared/sdd-coverage-review.md`): no bloquea fases, puertas ni archive, y no modifica los artefactos. El cambio está sellado `continuous`: los huecos van al resumen final y solo la persona puede pedir otra pasada de `sdd-spec` o `sdd-design`.

## Ronda 1 — 2026-11-22

- Entradas: `proposal.md`, los tres deltas de `specs/` y `design.md`.
- Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), ciegos y de solo lectura. Los dos devolvieron un resultado válido.
- Nota del orquestador: el hueco 1 contradecía la spec y el design. El gatekeeper lo corrigió acotando el escenario de REQ-33.10 a «la puerta sigue `approved`, no se reabre y conserva su modo». Evidencia: `resolveGateStatus` (`internal/kickoff/machine.go:250-254`) no mira el digest cuando la puerta ya está aprobada.
- Unión de huecos, sin duplicados (12):

| # | Tipo | Ubicación | Hueco | Revisor |
|---|---|---|---|---|
| 1 | requisito sin cubrir | D8 / REQ-33.10 | D8 escribe en el fichero de tareas, que es lo que digiere la puerta `tasks`. Con eso el escenario «su digest no cambia» no se cumplía. Además, D8 cita REQ-33.10 como fuente de la terminalidad, cuando esa regla está en REQ-26.3 y REQ-33.6. | A, B |
| 2 | caso límite | D9 | El objetivo de un `fixup!`, o el propio `fixup!`, se publica entre su creación y la petición de PR: no hay regla de respaldo ni aviso. | A, B |
| 3 | caso límite | D9 | Las precondiciones del autosquash no están cubiertas: árbol o índice sucio (un `pending_commit` del modo 3, un stash aparcado, ficheros de seguimiento sin confirmar) y merges de `main` en el rango, que `rebase -i` aplana sin `--rebase-merges`. Solo se prevé el conflicto. | A, B |
| 4 | caso límite | D9 / flujo de PR | Si la persona abre la PR sin pasar por el orquestador (script o plataforma), no se hace autosquash ni se avisa de los `fixup!` pendientes. Lo mismo ocurre al reanudar o tras un traspaso sin contexto de los avisos. | A |
| 5 | error | D7 | En el modo 3 no se cubren renombrados, borrados ni ficheros nuevos al confirmar por rutas explícitas, ni un índice que divergió tras la revisión. Tampoco el estado de la tarea con el índice vacío, un `git commit` que falla tras aprobar (hook, identidad, firma) ni un `pending_commit` huérfano de una sesión perdida. | A, B |
| 6 | caso límite | D2 | Falta fijar el comportamiento de `--commit-review=<valor>` y de los prefijos abreviados, y comprobar que el mensaje de error nombra el vocabulario. | B |
| 7 | caso de uso | D6, D7, D9 | En multirepo con varios roles no se define en qué repositorio se avisa, se confirma y se hace el autosquash, de dónde sale `<base>` o `prTarget` en cada repositorio, ni qué rol pausa cuando hay varios en paralelo. | A, B |
| 8 | caso límite | ODD / REQ-33.12 | Ninguna frase ni test dice que una petición de parada en el chat no cambia el modo; solo se comprueba que falta el vocabulario. | B |
| 9 | caso límite | D6 | Una tarea por lanzamiento también en `continuous` y `notify` se aparta del lote que mantenía la propuesta, y no hay criterio de coste. Faltan los fallos parciales (tarea sin commit, `pending_commit` en `notify`, hook que falla) y la reanudación sin avisos duplicados. | A, B |
| 10 | caso de uso | D6 | Los commits fuera del `sdd-apply` normal (remediación tras rechazar `role-apply`, arreglos del fix agent de judgment-day) no tienen definidos el aviso, el respeto del modo 3 ni su registro en D8. | A |
| 11 | caso límite | D4 | El modo solo se proyecta en `status --json`: no se define qué muestran la salida de texto de `status` ni `gate show`. | A |
| 12 | caso límite | D5 | No se cubre la aprobación de `tasks` hecha sin pasar por el orquestador. Tampoco la pregunta del modo sin respuesta válida, en la que hay que elegir entre bloquear el registro y registrar sin bandera, contra el principio de no asumir un valor por defecto. Ni una invocación que ya lleva la bandera en una reanudación, que daría una bandera doble. | A |
