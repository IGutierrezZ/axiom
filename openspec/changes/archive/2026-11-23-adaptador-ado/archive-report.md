# Reporte de Archivado Formal: adaptador-ado

## Resumen de Cierre
- Fecha de Archivado: 2026-11-23
- Estado de Verificación: verify del rol `fullstack` limpio en el momento de verificación (`verify-report.md`, sección «Cumplimiento técnico», `**Estado:** limpio`). Judgment-day: APPROVED. Correcciones posteriores al verify documentadas en «Hechos del estado final» (JD-1 en `a9289f8f`, fallo de CI en `743c5f04`).
- Estado de Implementación: 46 de 46 tareas de `tasks.md` marcadas como cerradas; 0 abiertas. Se observa en el fichero archivado.
- Integración: gate `integration` aprobado con evidencia `pr_merged` (PR #291 fusionada en `main` como `b1ba7246`).
- Destino en Archivo: `openspec/changes/archive/2026-11-23-adaptador-ado/`
- Topología: `monorepo-embedded`. Almacén: `hybrid` (ficheros en `openspec/` y reporte también en Engram). Contexto técnico mantenido: no declarado (`technicalContext: null`); no se ejecutaron `axiom archive context prepare/verify`.
- ISO: no. `axiom sdd archive-check` salió con código 0 sin comprobaciones adicionales.

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. Ningún delta contiene secciones `REMOVED` ni `RENAMED`. Los requisitos `MODIFIED` reemplazan el texto vigente de su capacidad sin marcarlo como `[SUPERSEDED]`; el texto anterior queda sustituido por la versión compuesta.

## Especificaciones Sincronizadas
| Dominio | Acción | Detalle |
|---------|--------|---------|
| `azure-devops-adapter` | Creada | Capacidad nueva copiada mecánicamente desde el delta. Contiene REQ-45.1 a REQ-45.17. |
| `integration-plugins` | Actualizada (`archive-compose`) | `ADDED` REQ-45.17 («Puertos Opcionales de Identidad y Descubrimiento»); `MODIFIED` REQ-43.4 y REQ-43.18. |
| `workspace-topology` | Actualizada (`archive-compose`) | `MODIFIED` REQ-43.21 («Plantilla y Manual Documentan `integrations:`»). |
| `openspec/INDEX.md` | Regenerado (`axiom archive sync`) | 74 especificaciones consolidadas, 641 requisitos activos, 2166 escenarios BDD. |

Nota de la spec: el diseño (`design.md`) indica el scope `vso.project` para las iteraciones. La spec viva, que prevalece, y la implementación usan `vso.work`. No se modifica el diseño archivado.

## Artefactos Consolidados
- `proposal.md`
- `specs/azure-devops-adapter/spec.md`, `specs/integration-plugins/spec.md`, `specs/workspace-topology/spec.md`
- `design.md`
- `tasks.md`
- `verify-report.md` (instantánea en el momento de la verificación)
- `exploration.md`, `kickoff.yaml`, `gates.yaml`, `.gates.lock`
- `coverage-review.spec.md`, `coverage-review.design.md`

## Hechos del estado final
Según el orden de prevalencia del skill, la fuente de mayor rango es la del propio `tasks.md` y el prompt de lanzamiento. Los puntos siguientes describen el estado al cierre:

- Tareas: todas cerradas (`tasks.md`, 46 casillas marcadas, 0 abiertas). La tarea P.4 se marcó en este worktree: `REQ-45` está libre en `main` `b1ba7246`.
- Verify: `verify-report.md` (fullstack) indica `**Estado:** limpio`. Ese informe es una instantánea del momento en que se escribió y no recoge los cambios posteriores.
- JD-1 (un `Retry-After` desorbitado podía desbordar `time.Duration`): detectado tras el verify-report y corregido en `a9289f8f`, con pruebas RED/GREEN. El valor ahora se satura.
- Fallo de CI posterior: el ratchet de rechazos (`TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign`) marcaba cinco centinelas en `internal/cli/integrations_ports.go`. Corregido en `743c5f04`: uno nombra `axiom workspace validate` y cuatro llevan `refusal:by-design`.
- CI de la PR #291: verde (8 pasadas, 5 omitidas, 0 fallos) antes de la fusión en `b1ba7246`.

## Hallazgos abiertos (informativos, no bloquean el archivado)
- JD-2: dos `connect` concurrentes pueden perder una entrada del fichero de identidad.
- JD-3: `connectionData` podría exigir `7.1-preview`.
- Sin verificar contra Azure DevOps real: `connectionData` con PAT, formato del rechazo de `test /rev` y campo `values` frente a `value` en las iteraciones.

## Trazabilidad
- Observation IDs leídos: ninguno. En esta fase las fuentes fueron los ficheros locales indicados por `axiom sdd status` (`artifactPaths`), no observaciones de Engram.
- Verificación mecánica del archivado: véase el `diff -r` registrado en el resultado de la fase (fichero de origen contra destino, sin diferencias).
- Estado de Git: los cambios quedan en el árbol de trabajo sin commit. El commit lo realiza el orquestador. `tasks.md` tiene la modificación de P.4 sin confirmar antes del archivado.
