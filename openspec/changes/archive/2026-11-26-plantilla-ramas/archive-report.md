# Reporte de Archivado Formal: plantilla-ramas

## Resumen de Cierre
- Fecha de Archivado: 2026-11-26
- Estado de Verificación: verify del rol `fullstack` limpio al cierre de la fase apply (0 CRITICAL, 0 WARNING, 1 SUGGESTION) según `verify-report.md`, con la corrección posterior de CI descrita abajo.
- Destino en Archivo: `openspec/changes/archive/2026-11-26-plantilla-ramas/`
- Código: PR #371 fusionada con merge commit `aed635e72` (rama `inc/plantilla-ramas`).
- Tareas: 10/10 completadas según `tasks.md` (1.1, 2.1-2.3, 3.1-3.3, 4.1-4.2, 5.1). Sin tareas pendientes.

## Estado Final (fuente: instrucciones de cierre del orquestador, con prioridad sobre snapshots)
- El `verify-report.md` se escribió antes de la corrección de CI. Tras ese verify, la CI falló en `TestImportsDirection` (`internal/branch` no podía importar `internal/integrations`). Se corrigió en el commit `cc631004d` (`fix(branch): el motor lee los vínculos a través de changestart`), moviendo la lectura de vínculos a `changestart.LinkedIDs`. La CI quedó en verde y la PR se fusionó. Ambos hechos son verificables en el grafo de commits del repositorio.
- La sugerencia del verify (IDs `prov:` citados en comentarios de Go) se mantiene sin cambios por ser la convención vigente del repositorio (p. ej. código de C-07-e).
- Revisión de cobertura: 12 huecos en la spec y 11 en el diseño (`coverage-review.spec.md`, `coverage-review.design.md`). Se cerraron con una corrección acotada de spec y otra de diseño, registradas en el `## Registro de remediación` de la delta `change-branch-lifecycle`.

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. Incremento sin `REMOVED`; no se usó `--supersede` ni `--superseded-requirements`. Los doce requisitos de la delta `MODIFIED` reemplazan contenido vigente y conservan su identificador (RF-BRA-03, RF-BRA-04, RF-BRA-14, RF-BRA-15, RF-BRA-16, RF-BRA-19, RF-BRA-33, RF-KOF-06, RF-TOP-03, RF-TOP-09, RF-TOP-10, RF-TOP-21).

## Identificadores Asignados
Fuente: `assignments[]` de `axiom sdd archive-integrate --json` (exit 0). Traza en `ids.yaml`.

| Capacidad | Requisito | Provisional | Definitivo | Estado |
|-----------|-----------|-------------|------------|--------|
| change-branch-lifecycle | Plantilla de Nombre de Rama: Motor y Variables | prov:BRA:1 | RF-BRA-42 | assigned |
| change-branch-lifecycle | Origen de los Valores de las Variables de la Plantilla | prov:BRA:2 | RF-BRA-43 | assigned |
| change-branch-lifecycle | Plantilla Efectiva por Rol, Repositorio y Proyecto, y Precedencia de Nombres | prov:BRA:3 | RF-BRA-44 | assigned |
| change-branch-lifecycle | Rechazos Tipados de la Resolución de Plantillas | prov:BRA:4 | RF-BRA-45 | assigned |
| change-branch-lifecycle | Diferido de las Ramas con `{task_ref}` a la Entrada del Rol | prov:BRA:5 | RF-BRA-46 | assigned |
| sdd-lane-kickoff | La Instantánea Congela la Plantilla, el Tipo y la Referencia de Work Item | prov:KOF:1 | RF-KOF-26 | assigned |
| workspace-topology | Validación de `branch_template` al Cargar `axiom.yaml` | prov:TOP:1 | RF-TOP-24 | assigned |

Duplicados: ninguno (`duplicates[]` vacío).

## Requisitos Modificados (MODIFIED)
| Capacidad | Identificador | Requisito |
|-----------|---------------|-----------|
| change-branch-lifecycle | RF-BRA-03, RF-BRA-04, RF-BRA-14, RF-BRA-15, RF-BRA-16, RF-BRA-19, RF-BRA-33 | Política de ramas, modelo por número de roles, carril ODD, registro local, `status`, veredicto por papel, `ensure --role` |
| sdd-lane-kickoff | RF-KOF-06 | Instantánea de política Git en `kickoff.yaml` |
| workspace-topology | RF-TOP-03, RF-TOP-09, RF-TOP-10, RF-TOP-21 | Esquema de `axiom.yaml`, bloque `git:`, `repositories:`, catálogo de claves |

## Integración de Especificaciones
- Comando: `axiom sdd archive-integrate --cwd C:\repos\axiom-wt\plantilla-ramas-archive\axiom --change plantilla-ramas --json` (exit 0; precedido de `--check` con exit 0).
- Capacidades escritas: `openspec/specs/change-branch-lifecycle/spec.md`, `openspec/specs/sdd-lane-kickoff/spec.md`, `openspec/specs/workspace-topology/spec.md` (todas `composed`).
- Corrección posterior: el integrador copió a la spec viva el bloque `## Registro de remediación` (registro del cambio, no contenido de spec). Se eliminó de `openspec/specs/change-branch-lifecycle/spec.md` tras verificar sus límites. Se conserva íntegro en la delta archivada (`specs/change-branch-lifecycle/spec.md`), que es el registro histórico.
- `## Purpose` de las specs vivas tocadas: revisado; sigue describiendo el alcance de cada capacidad. No requirió cambio.
- Contexto técnico mantenido: no proyectado por el estado nativo; no aplica.

## Índice Maestro
`openspec/INDEX.md` regenerado con `axiom archive sync` (exit 0): 50 dominios, 619 requisitos, 3099 escenarios BDD.

## Artefactos Consolidados
Movidos mecánicamente con `git mv` (readback `diff -r` vacío, ver abajo):
- `proposal.md`, `design.md`, `explore.md`, `tasks.md`, `verify-report.md`, `apply-progress.md`
- `specs/` (3 deltas), `kickoff.yaml`, `coverage-review.spec.md`, `coverage-review.design.md`
- `resumen-spec.html`, `resumen-design.html`
- `ids.yaml` (traza de identificadores, generado por `archive-integrate`)

Sin `integrations.yaml`: no se compone sección de seguimiento en el tracker. Sin clave `acceptanceTests.marks` en el estado: no se compone sección de marcas de pruebas.

## Trazabilidad de Observaciones de Engram (copias espejo, no leídas en esta fase)
El estado nativo proyectó el almacén `openspec`; las observaciones siguientes existen como espejo y no se usaron como fuente de contenido:
- `sdd/plantilla-ramas/explore` (#2742), `proposal` (#2745), `design` (#2751), `spec` (#2749), `tasks` (#2756), `apply-progress` (#2758), `verify-report` (#2763), `coverage-review/spec` (#2750), `coverage-review/design` (#2753).

## Verificación del Archivado
- Pre-check: `axiom sdd archive-check --cwd <worktree> --change plantilla-ramas --json` exit 0, sin hallazgos.
- Comprobación `diff -r` snapshot previo al movimiento vs. destino: vacío (ver resultado de la fase).
- `axiom archive list`: exit 0.
- `go test ./internal/idmigration/ -run Residue`: ok.
- `go test ./internal/livespec/ -run 'Index|Registry'`: ok.

## Pendientes Informativos
- Publicación de la rama de archivado y PR documental hacia `integration`: la realiza la persona (no se hizo push, fetch ni PR).
- Limpieza del worktree de archivo con `axiom branch done` tras el merge de la PR documental: la realiza la persona.
- Sugerencia de verify (IDs `prov:` en comentarios Go): sin acción, por convención vigente del repositorio.
- Las tareas de `tasks.md` están todas marcadas; no hay tareas abiertas al cierre.
