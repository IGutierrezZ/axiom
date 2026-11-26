# Reporte de Archivado Formal: autofetch-specs

## Resumen de Cierre
- Fecha de Archivado: 2026-11-26
- Estado de Verificación: verify-report de la fase de verificación en estado «limpio» (0 CRITICAL, 0 WARNING, 3 SUGGESTION) en el momento de su redacción; hallazgos posteriores corregidos en `68bdcdd7`, según el estado final proporcionado por el orquestador.
- Implementación: fusionada en `main` mediante la PR https://github.com/IGutierrezZ/axiom/pull/353 con merge commit `4df28b04` (sin squash). CI en verde tras la corrección.
- Destino en Archivo: `openspec/changes/archive/2026-11-26-autofetch-specs/`
- Modo de almacenamiento: `openspec` (sin mirror Engram de artefactos; este reporte se persiste también en Engram como `sdd/autofetch-specs/archive-report`).

## Estado Final de Implementación (ODD-4, jerarquía de fuentes)
Fuentes, por rango: (1) `tasks.md` persistido; (2) hechos explícitos del orquestador; (3) `verify-report.md` y `apply-progress`, instantáneas intermedias.

- Tareas en `tasks.md`: 10 de 10 marcadas como completadas, 0 pendientes (fuente de rango 1).
- Commits por tarea: T1 `66776395`, T2 `dd82958e`, T3 `869dc142`, T4 `81af3089`, T5 `d28adf84`, T6 `08f351ca`, T7 `e56a87bd`, T8 `ba164c0d`, T9 `94d0ba9d`, T10 `7224026f`. Merge de `main` `c82fd3cf` (trae C-06-b). Verify `537271ce`.
- Hallazgos del verify (instantánea de `537271ce`, en el momento de la verificación): SUGGESTION 1 (prueba directa de `WaitDelay` en `internal/gitcommon`), SUGGESTION 2 (`engine.SetAutofetchRunBudget` exportada solo para pruebas) y SUGGESTION 3 (cota real de 10 s y contrato `SessionStart` de Codex no verificados de forma manual).
- Estado final de la SUGGESTION 2: resuelta en `68bdcdd7`, que retira `engine.SetAutofetchRunBudget`; la prueba acota el presupuesto con el contexto de la CLI. Comprobado en el árbol: en `HEAD` ya no existe referencia de código a ese símbolo; solo aparece en los documentos históricos del cambio.
- Estado final de la SUGGESTION 1 y de la SUGGESTION 3: no consta que se hayan cerrado; se registran como pendientes informativos (ver «Trabajo no cerrado»).
- Ese mismo commit `68bdcdd7` corrige la CI: los errores de uso de `axiom autofetch` nombran la orden de continuación (trinquete de rechazos), y la prueba del sello es portable a Linux.

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. Incremento aditivo en lo sustancial: las deltas MODIFIED (RF-BRA-01, RF-BRA-10, RF-BRA-13, RF-HUB-07) reescriben requisitos vivos sin superar ninguno mediante `--supersede`.

## Identificadores Asignados
Ninguno. Sin identificadores provisionales (`assignments[]` vacío en el JSON de `axiom sdd archive-integrate`).

## Integración de Especificaciones Vivas
Invocación: `axiom sdd archive-integrate --cwd "C:\repos\axiom-wt\autofetch-specs-archive\axiom" --change autofetch-specs --json` (precedida de la ejecución `--check`, exit 0, sin escrituras).
Exit 0. Resultado por capacidad: `change-branch-lifecycle` «composed»; `multi-project-hub` «composed».

| Capacidad | Operación | Requisito | Identificador |
|-----------|-----------|-----------|---------------|
| change-branch-lifecycle | MODIFIED | Familia de Comandos `axiom branch` (RF-BRA-01) | RF-BRA-01 |
| change-branch-lifecycle | MODIFIED | Base sin Red y `--fetch` Explícito (RF-BRA-10) | RF-BRA-10 |
| change-branch-lifecycle | MODIFIED | Rama de Archivado y `ensure --archive` (RF-BRA-13) | RF-BRA-13 |
| change-branch-lifecycle | ADDED | Autofetch del Repositorio de Specs con el Interruptor Personal (RF-BRA-39) | RF-BRA-39 |
| change-branch-lifecycle | ADDED | Disparo del Autofetch por Hook de Sesión y por la Doctrina (RF-BRA-40) | RF-BRA-40 |
| multi-project-hub | MODIFIED | Preferencias Personales de Worktree por Proyecto en el Hub (RF-HUB-07) | RF-HUB-07 |
| multi-project-hub | ADDED | Comando `axiom autofetch` para Activar, Desactivar y Consultar la Preferencia (RF-HUB-13) | RF-HUB-13 |

Índice maestro: `axiom archive sync --path` (exit 0): 49 especificaciones, 530 requisitos activos, 2436 escenarios; `openspec/INDEX.md` regenerado.

## Trabajo No Cerrado y Pendientes Informativos
- Tablas de cabecera de `change-branch-lifecycle` (sección «Requerimientos de un vistazo», filas RF-BRA-01, RF-BRA-10 y RF-BRA-13, y línea «Operaciones remotas por defecto» de «No cubre esta capacidad»): la nota de cabecera de la delta pide actualizarlas al integrar, pero el verbo `archive-integrate` no las reescribe y la regla de archivo reserva la escritura de `openspec/specs/` a ese verbo. No se editaron a mano. Queda pendiente de decisión (ver «Riesgos»).
- Sugerencia 1 del verify: prueba directa de `WaitDelay` en `internal/gitcommon`. No consta cerrada.
- Sugerencia 3 del verify: validación manual del contrato `SessionStart` de Codex (R-5) y medición de la cota real de 10 s. No consta cerrada.
- Límite de relevancia con varios cambios activos; `--fetch` explícito no trae el repositorio de specs in situ de un multirepo; `ensure` con red caída puede tardar unos 8·N s. Pendientes informativos del kickoff.
- Huecos de cobertura no absorbidos: listados en `tasks.md`, sección «Huecos de cobertura no absorbidos». Revisiones de cobertura en `coverage-review.spec.md` (10 huecos) y `coverage-review.design.md` (12 huecos).

## Artefactos Consolidados
- `proposal.md`: presente
- `specs/change-branch-lifecycle/spec.md`, `specs/multi-project-hub/spec.md`: presentes
- `design.md`: presente
- `tasks.md`: presente, 10 completadas, 0 pendientes
- `verify-report.md`: presente (instantánea, ver «Estado Final»)
- `coverage-review.spec.md`, `coverage-review.design.md`, `explore.md`, `kickoff.yaml`: presentes

## Verificación de Archivado
- Archive check: `axiom sdd archive-check --cwd ... --change autofetch-specs` exit 0, `ok: true`, sin hallazgos (cambio no ISO).
- Precondición de integración: el estado nativo proyecta `archive: ready`; `axiom sdd gate show` no registra decisiones de compuerta. No se registró ninguna decisión nueva en esta fase.
- Lectura de `diff -r` tras el movimiento (fuente instantánea previa vs. destino): salida vacía, exit 0.

## Riesgos
- Especificaciones vivas: la integración de la delta añadió los requisitos y modificó los MODIFIED; las tablas de cabecera de `change-branch-lifecycle` siguen sin actualizar (ver «Trabajo No Cerrado»).
- Los cambios de archivo (`openspec/INDEX.md`, `openspec/specs/...`, movimiento del cambio) están sin confirmar en el worktree de archivado; el commit y la publicación los realiza el orquestador.
