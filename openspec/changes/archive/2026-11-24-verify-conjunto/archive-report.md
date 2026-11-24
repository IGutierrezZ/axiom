# Reporte de Archivado Formal: verify-conjunto

## Resumen de Cierre
- Fecha de Archivado: 2026-11-24
- Estado del cambio al cierre: archivado. Implementación en `main` mediante la PR #296 (commit de fusión `fc4585c6`). Trazabilidad de la PR de código: `fc4585c6`.
- Estado de verificación: `verify-report.md` registra **Estado: limpio**, sin hallazgos CRITICAL ni WARNING, y **JUDGMENT: APPROVED**. Es una foto del momento de la verificación (ver nota de Fuente de verdad más abajo).
- Destino en Archivo: `openspec/changes/archive/2026-11-24-verify-conjunto/`
- Topología: `monorepo-embedded`. Rama de archivo: `inc/verify-conjunto-archive`. Almacén de artefactos: `hybrid`.

## Especificaciones Históricas Superadas (ODD-4.2)
| Dominio | Requisito Previo | Estado | Razón de la Poda / Superación | Nueva Referencia |
|---------|------------------|--------|-------------------------------|------------------|
| change-branch-lifecycle | REQ-25.1 (Familia de Comandos `axiom branch`) | Reemplazado en sitio (MODIFIED) | Ampliación de la familia de comandos con `scope`. Mismo identificador. | `change-branch-lifecycle/spec.md#familia-de-comandos-axiom-branch-req-251` |
| change-branch-lifecycle | REQ-25.2 (Contrato de Salida Común con Rutas Absolutas) | Reemplazado en sitio (MODIFIED) | Contrato de salida extendido al verbo `scope`. Mismo identificador. | `change-branch-lifecycle/spec.md#contrato-de-salida-comun-con-rutas-absolutas-req-252` |
| sdd-independent-review | REQ-28.1 (Tres puntos del flujo) | Renombrado (RENAMED) y reemplazado | Pasa de tres a cuatro puntos del flujo por la verificación conjunta. Mismo identificador. | `sdd-independent-review/spec.md#el-orquestador-ofrece-la-revision-independiente-en-cuatro-puntos-del-flujo-req-281` |
| sdd-independent-review | REQ-28.4 (Verify por rol) | Reemplazado en sitio (MODIFIED) | Ajuste del verify por rol con sección de cumplimiento técnico. Mismo identificador. | `sdd-independent-review/spec.md#verify-por-rol-con-seccion-de-cumplimiento-tecnico-req-284` |

No se usó la marca `[SUPERSEDED / DEPRECADO]`: ningún requisito fue retirado sin sustituto. Los cambios son reemplazos en sitio del mismo identificador más adiciones.

## Sincronización de Especificaciones Vivas
| Dominio | Acción | Detalle |
|---------|--------|---------|
| change-branch-lifecycle | Actualizado | 4 añadidos (REQ-25.51 a REQ-25.54), 2 modificados (REQ-25.1, REQ-25.2). Requisitos: 34 → 38. |
| sdd-independent-review | Actualizado | 2 añadidos (REQ-28.41, REQ-28.42), 1 modificado (REQ-28.4), 1 renombrado (REQ-28.1). Requisitos: 10 → 12. |
| sdd-block-review-gates | Actualizado | 1 añadido (REQ-43.32). Requisitos: 25 → 26. |

Composición con `axiom sdd archive-compose` (exit 0 en los tres dominios). Catálogo maestro regenerado con `axiom archive sync` (75 especificaciones, 657 requisitos activos, 2322 escenarios).

## Artefactos Consolidados
- `proposal.md`: presente
- `design.md`: presente
- `tasks.md`: presente, 21 de 21 tareas marcadas como completadas, 0 pendientes en el fichero
- `verify-report.md`: presente (snapshot de verificación)
- `exploration.md`: presente
- `coverage-review.spec.md`, `coverage-review.design.md`: presentes
- `kickoff.yaml`: presente
- `specs/`: tres deltas (`change-branch-lifecycle`, `sdd-independent-review`, `sdd-block-review-gates`), integrados en las specs vivas

## Fuente de verdad y discrepancias
- **Tareas**: el fichero `tasks.md` tiene 21 casillas, todas marcadas. El lanzamiento indicaba 22 tareas. Se registra el recuento observado (21/21); no se ha modificado ninguna casilla.
- **Verificación**: `verify-report.md` describe el estado en el momento de la verificación. Declara limitaciones: no se ejecutó la suite completa de `internal/cli` ni `go test ./...` (cuelgue y fallos preexistentes conocidos), y no se ejercitó el verbo contra un repositorio real. Esas limitaciones se conservan tal cual.
- **Pendiente de confirmación (no bloqueante)**: `verify-report.md` sugiere confirmar en un caso real que `planningHome` equivale a la carpeta `openspec/` en `monorepo-embedded`. No se ha comprobado en esta fase.
- **Cobertura de escenarios**: REQ-28.1, REQ-28.42 y REQ-43.32 se verifican mediante pines de texto en skills y doctrina, no mediante ejecución real de un agente (indicado en `verify-report.md`).

## Comprobaciones de archivado
- `axiom sdd archive-check --change verify-conjunto`: exit 0, cambio no ISO.
- `axiom sdd status verify-conjunto --json --instructions`: `dependencies.archive: ready`, `nextRecommended: archive`, sin `blockedReasons`.
- Sin `technicalContext.maintained` proyectado: no aplica la integración de contexto mantenido.
- `go test ./internal/assets -run 'TestSDDArchiveFinalStateAuthorityContract$' -count=1`: ok. Valida el contrato de la skill de archivado, no el contenido de las specs vivas.

## Observaciones de Engram
- No se leyeron observaciones de Engram en esta fase; los artefactos se leyeron desde el sistema de ficheros (`openspec`, store `hybrid`). Lista de IDs de observación leídos: ninguna.
