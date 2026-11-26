# Reporte de Archivado Formal: arranque-desde-wi (C-07-d)

## Resumen de Cierre
- Fecha de Archivado: 2026-11-26
- Estado de Verificación: verify limpio (`sdd/arranque-desde-wi/verify-report`, observación #2716, «Estado: limpio», 0 CRITICAL, 0 WARNING, 3 SUGGESTION). Snapshot de verificación: commit 8a77b6310 (según el orquestador, verify limpio en ese commit).
- Estado de Implementación: T0..T9 completas (77/77 casillas en `tasks.md`). T9 cerrada en el commit 78b1f48c1 (`chore(change-start)`: evidencia de los trinquetes de cierre). Nota: el `verify-report` (instantánea de 15:25) afirma «T9 sin commit propio»; prevalece el estado final indicado por el orquestador, que cita 78b1f48c1.
- Integración (compuerta `integration`): aprobada con evidencia `pr_merged`, PR #359, merge 64fd72a0331468dcd62c0612c2dc2db7d3c2e917, verificado contra el grafo de commits local (`git merge-base --is-ancestor` sobre HEAD). Registro en `gates.yaml`.
- CI de PR #359: en verde según el orquestador (no comprobado en esta fase con `gh`).
- Destino en Archivo: `openspec/changes/archive/2026-11-26-arranque-desde-wi/`

## Especificaciones Históricas Superadas (ODD-4.2)
Ninguno. Incremento puramente aditivo en la trazabilidad de requisitos: solo `MODIFIED` (RF-ADO-01, RF-ADO-02, RF-ADO-16, RF-INT-09) y `ADDED`. Sin `REMOVED` ni `--supersede`.

## Identificadores Asignados
| Capacidad | Requisito | Provisional | Definitivo | Estado |
|-----------|-----------|-------------|------------|--------|
| azure-devops-adapter | Lectura Tipada del Work Item con Relaciones, Etiquetas y Adjuntos | prov:ADO:21 | RF-ADO-17 | assigned |
| azure-devops-adapter | Descarga Binaria Acotada con el PAT Restringido | prov:ADO:22 | RF-ADO-18 | assigned |
| azure-devops-adapter | Enlace de Vuelta por Etiqueta Idempotente sobre `test /rev` | prov:ADO:23 | RF-ADO-19 | assigned |
| change-start-from-work-item | Verbo `axiom change start` en Dos Fases con Salida Versionada | prov:ARR:1 | RF-ARR-01 | assigned |
| change-start-from-work-item | Sin Plugin Activo el Verbo Responde «No Aplica» sin Red | prov:ARR:2 | RF-ARR-02 | assigned |
| change-start-from-work-item | Con Plugin Activo y sin WI el Verbo se Niega y Pide el WI | prov:ARR:3 | RF-ARR-03 | assigned |
| change-start-from-work-item | Resolución del WI por ID o por URL del Host y la Organización Configurados | prov:ARR:4 | RF-ARR-04 | assigned |
| change-start-from-work-item | La Fase 1 Solo Escribe en Staging bajo el Common Dir de Git | prov:ARR:5 | RF-ARR-05 | assigned |
| change-start-from-work-item | Rechazos Tipados de la Fase 1 con Continuación Nombrada | prov:ARR:6 | RF-ARR-06 | assigned |
| change-start-from-work-item | Un WI Enlazado a Otro Cambio Abierto se Rechaza | prov:ARR:7 | RF-ARR-07 | assigned |
| change-start-from-work-item | Derivación Determinista del Nombre del Cambio | prov:ARR:8 | RF-ARR-08 | assigned |
| change-start-from-work-item | Perfil del Cambio según el Tipo del WI | prov:ARR:9 | RF-ARR-09 | assigned |
| change-start-from-work-item | Importación del WI como Contexto en Markdown | prov:ARR:10 | RF-ARR-10 | assigned |
| change-start-from-work-item | El Contenido Importado es Dato no Fiable de un Tercero | prov:ARR:11 | RF-ARR-11 | assigned |
| change-start-from-work-item | Adjuntos con Lista Blanca y Límites | prov:ARR:12 | RF-ARR-12 | assigned |
| change-start-from-work-item | El PAT Solo se Envía al Host y la Organización Configurados y los Nombres se Sanean | prov:ARR:13 | RF-ARR-13 | assigned |
| change-start-from-work-item | Fase 2 `--adopt` Mueve el Contexto, Vincula el WI y es Idempotente | prov:ARR:14 | RF-ARR-14 | assigned |
| change-start-from-work-item | Enlace de Vuelta por Etiqueta; un Fallo del Tracker Avisa sin Fallar | prov:ARR:15 | RF-ARR-15 | assigned |
| change-start-from-work-item | `--discard` Retira un Staging Abandonado | prov:ARR:16 | RF-ARR-16 | assigned |
| change-start-from-work-item | Un Cerrojo Local por Cambio Evita Arranques Simultáneos | prov:ARR:17 | RF-ARR-17 | assigned |
| change-start-from-work-item | `context/` no se Versiona | prov:ARR:18 | RF-ARR-18 | assigned |
| change-start-from-work-item | Hermeticidad de las Pruebas del Arranque | prov:ARR:19 | RF-ARR-19 | assigned |
| integration-plugins | Puerto Opcional de Work Items | prov:INT:20 | RF-INT-24 | assigned |
| organic-agent-trigger-rules | La Doctrina ODD Exige el WI con Plugin Activo y Rodea `ensure` con las Dos Fases | prov:RUT:25 | RF-RUT-22 | assigned |
| sdd-lane-kickoff | Con Plugin Activo el WI y la Fase 1 Preceden a `ensure` y a `seal` | prov:KOF:24 | RF-KOF-24 | assigned |
| sdd-orchestrator-assets | Paridad entre Runtimes de la Regla del WI Obligatorio y del Contenido Importado como Dato no Fiable | prov:ORQ:26 | RF-ORQ-27 | assigned |
| sdd-orchestrator-assets | `sdd-propose` Copia `external_ref` y `Referencia:` desde el Vínculo del WI | prov:ORQ:27 | RF-ORQ-28 | assigned |

Los provisionales ya estaban corregidos al formato `prov:<DOM>:N` antes del archivado (corrección del orquestador, en el árbol del worktree de archivo). `axiom sdd archive-check` devolvió `ok: true` sin hallazgos.

## Integración de Especificaciones Vivas
- Verbo: `axiom sdd archive-integrate --cwd <raíz> --change arranque-desde-wi --json` (sin `--check`), código de salida 0.
- Capacidades: `azure-devops-adapter` (composed), `change-start-from-work-item` (created), `integration-plugins` (composed), `organic-agent-trigger-rules` (composed), `sdd-lane-kickoff` (composed), `sdd-orchestrator-assets` (composed).
- Escritos: `openspec/specs/**` de las seis capacidades, `tasks.md` e `ids.yaml` del cambio.
- Catálogo maestro: `axiom archive sync --path <raíz>` (exit 0): 50 especificaciones, 569 requisitos activos, 2674 escenarios; `openspec/INDEX.md` regenerado.
- Tablas resumen («Requerimientos de un vistazo», «No cubre»): las capacidades tocadas no las tienen; no se editó nada a mano.
- Contexto técnico mantenido: no proyectado por el estado nativo; no aplica.

## Hallazgos y Cobertura (informativos, no bloquean el archivo)
- `coverage-review.spec.md` (ronda 1, 2026-11-26): 10 huecos, dos revisores coinciden en 5. Clave: contradicción en prov:74:14 (staging retirado frente a `--adopt` idempotente y reintento del enlace de vuelta). Estado de su resolución: no verificado en esta fase; el hallazgo queda registrado como informativo.
- `coverage-review.design.md` (ronda 1, 2026-11-26): 17 huecos, dos revisores coinciden en 7. Principales: doctrina D-21 sin fallos de `ensure`/`seal`, rechazos de `--adopt` ni bug ODD; ReadItem consulta estados antes de validar tipo; 404 genérico frente a `not_found`.
- Sugerencias del verify (`verify-report.md`, observación #2716): (1) el registro de commits de `tasks.md` no menciona que T9 no generó commit propio (ver nota de T9 arriba); (2) las pruebas de paridad de T8 viven en `internal/components/sdd/workitem_rule_test.go` y `internal/components/agentguidance/routing_workitem_test.go`, no en el fichero que citaba la tarea; (3) `FuzzConvert` y `FuzzDeriveName` solo se han ejecutado con su corpus de semillas; no hay evidencia de fuzzing extendido.
- Verificación no ejecutada en esta fase: `scripts/deadcode-ratchet.sh` (según el verify, lo ejecutó el orquestador sin funciones inalcanzables nuevas).
- RF-GAT-32 (pendiente de C-06-c) incorporado en T8 (tarea 8.6).

## Artefactos Consolidados
- `proposal.md` (Engram #2702)
- `specs/` (6 capacidades; Engram #2703 «spec»)
- `design.md` (Engram #2705)
- `tasks.md` (Engram #2707; 77/77 casillas completas en el estado final)
- `verify-report.md` (Engram #2716)
- `explore.md` (Engram #2701), `coverage-review.spec.md` (#2704), `coverage-review.design.md` (#2706), `apply-progress` (#2708, instantánea previa a T9)
- `kickoff.yaml`, `gates.yaml`, `ids.yaml`

Observaciones Engram leídas o referenciadas para trazabilidad: #2701, #2702, #2703, #2704, #2705, #2706, #2707, #2708, #2716. Observación previa #2717 (archivado bloqueado por `provisional_malformed`) sobrescrita por este reporte.

## Comprobaciones
- `axiom sdd archive-check`: exit 0, `ok: true`.
- `axiom sdd archive-integrate --check`: exit 0.
- `axiom sdd archive-integrate`: exit 0, `written` con 14 rutas.
- Movimiento de carpeta: `git mv` + `diff -r` contra snapshot previo: salida vacía (aprobado).
