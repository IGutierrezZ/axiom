# Reporte de Archivado Formal: preflight-topologia

## Resumen de Cierre
- Fecha de Archivado: 2026-11-18
- Estado de Verificación: Cumple según `verify-report.md` en el momento de la verificación (0 CRITICAL, 1 WARNING, 3 SUGGESTION). La WARNING 1 queda resuelta por este archivado: los requisitos modificados ya no conservan escenarios antiguos de monorepo con roles especializados. No se repitió la verificación.
- Destino en Archivo: `openspec/changes/archive/2026-11-18-preflight-topologia/`
- Implementación al cierre: 46 de 46 tareas marcadas `[x]` en el `tasks.md` archivado; 0 pendientes. Lectura directa del fichero, con prioridad sobre cualquier snapshot.
- Integración: PR #272, fusionada en `main` con el merge commit `bd36e5c2`. Correcciones de CI posteriores a la verificación, ya fusionadas, según el orquestador: `6c527419` (fixtures de `workspace validate` con rol `fullstack`) y `9e50c150` (journeys 3336 y 3500). Corrección de spec `06cb141e` (REQ-25.7: en multirepo no se crea rama en el repositorio de specs antes del sellado).
- Modalidad de sellado: `continuous` con el rol `fullstack` (`kickoff.yaml`). Sin puertas de bloque (`gates: []`).

## Especificaciones Históricas Superadas (ODD-4.2)
Ningún requisito se elimina con REMOVED: los 8 deltas no contienen esa sección. El incremento no es puramente aditivo: modifica 17 requisitos y renombra uno. Los escenarios antiguos de monorepo con roles especializados se trasladan a `multirepo` o al rol `fullstack`; cada requisito modificado conserva una nota `(Previously: ...)` con el comportamiento anterior.

| Dominio | Requisito Previo | Estado | Razón de la Poda / Superación | Nueva Referencia |
|---------|------------------|--------|-------------------------------|------------------|
| sdd-preflight-configuration | La Preferencia de Revisión Independiente se Recoge en una Interacción Propia Tras el Pre-Vuelo, sin Alterar sus Tres Grupos Canónicos (REQ-28.14) | [RENOMBRADO] | El pre-vuelo ya no tiene tres grupos fijos: el conjunto de grupos depende de la topología. El identificador REQ-28.14 se conserva. | `sdd-preflight-configuration/spec.md`, requisito «La Preferencia de Revisión Independiente se Recoge en una Interacción Propia Tras el Pre-Vuelo, sin Reutilizar sus Marcadores ni sus Grupos (REQ-28.14)» |

## Integración de Deltas en las Specs Vivas
Composición con `axiom sdd archive-compose --cwd <worktree> --canonical openspec/specs/<dominio>/spec.md --delta openspec/changes/preflight-topologia/specs/<dominio>/spec.md --output openspec/specs/<dominio>/spec.md.compose-tmp`, seguida de `mv` solo con código de salida 0. Los 8 dominios existían como specs vivas. Ningún `--supersede` (no hay REMOVED).

| Dominio | Acción | ADDED | MODIFIED | REMOVED | RENAMED | Resultado |
|---------|--------|-------|----------|---------|---------|-----------|
| axiom-sdd-cli-integration | Actualizada | 0 | 1 | 0 | 0 | rc 0 |
| change-branch-lifecycle | Actualizada | 0 | 5 | 0 | 0 | rc 0 |
| multi-project-hub | Actualizada | 1 | 2 | 0 | 0 | rc 0 |
| project-technical-context | Actualizada | 0 | 1 | 0 | 0 | rc 0 |
| sdd-archive-lifecycle | Actualizada | 0 | 1 | 0 | 0 | rc 0 |
| sdd-lane-kickoff | Actualizada | 0 | 1 | 0 | 0 | rc 0 |
| sdd-preflight-configuration | Actualizada | 8 | 5 | 0 | 1 | rc 0 |
| workspace-topology | Actualizada | 1 | 1 | 0 | 0 | rc 0 |

No quedó ningún fichero `spec.md.compose-tmp`. Los conteos proceden del recuento de encabezados `### Requirement:` de cada delta.

Catálogo maestro: `axiom archive sync --path <worktree>` con código 0. Resultado: 71 especificaciones, 563 requisitos activos y 1725 escenarios BDD. `openspec/INDEX.md` regenerado; contiene REQ-28.15 a REQ-28.22, REQ-2.6 y REQ-3.3.

## Comprobación de Monorepo con Roles tras la Integración
Búsqueda de `monorepo` junto con `core`, `qa`, `web`, `backend`, `frontend`, `feat/x-core` o `feat/x-qa` en las 8 specs vivas. Los resultados restantes son intencionados:
- Escenarios de validación que rechazan monorepo con roles especializados: `workspace-topology` (líneas 133 a 145), `multi-project-hub` (línea 149) y `sdd-preflight-configuration` (línea 635).
- Escenario de retro-sello, exento de la regla de topología por el propio delta: `sdd-preflight-configuration` (línea 658).
- Notas `(Previously: ...)`, que describen el comportamiento anterior.

No queda ningún escenario de flujo normal de monorepo con roles especializados en los requisitos modificados.

## Artefactos Consolidados
- `proposal.md`, `exploration.md`, `design.md`, `kickoff.yaml`: presentes y movidos.
- `specs/`: 8 deltas presentes y movidos.
- `tasks.md`: presente; 46 completadas y 0 pendientes.
- `verify-report.md`: presente; movido sin cambios.
- `apply-progress.md`: presente; movido sin cambios.
- `archive-report.md`: este fichero, añadido tras el movimiento.

## Verificación Mecánica (diff -r)
Instantánea previa al movimiento en el directorio temporal de la sesión, `cp -R` del cambio, después `git mv` a la carpeta de archivado. Readback:

```
diff -r "<scratchpad>/sdd-archive-snapshot" "openspec/changes/archive/2026-11-18-preflight-topologia"
```

Salida verbatim: vacía, código 0. El origen `openspec/changes/preflight-topologia` ya no existe.

## Comprobaciones Ejecutadas
- `axiom sdd status preflight-topologia --cwd <worktree> --json --instructions` (antes del movimiento): código 0; `nextRecommended: archive`; `blockedReasons: []`; `dependencies.archive: ready`; `taskProgress` 46 de 46.
- `axiom sdd status preflight-topologia --cwd <worktree> --json` (después del movimiento): código 0, pero no concluyente. Al no estar el cambio en `openspec/changes/`, el status resuelve el nombre contra Engram (`artifactStore: engram`) y reporta `archive: blocked` con el motivo «tasks.md has no markdown task checkboxes». Ese lookup no lee la carpeta archivada, donde el `tasks.md` tiene 46 `[x]` y ningún `[ ]`.
- `go test ./internal/livespec/... ./internal/openspecdoc/... ./internal/isoevidence/...`: código 0; los tres paquetes `ok`.
- `go test ./internal/workspace/ -run "Docs|Contract|Pin" -count=1`: código 0; paquete `ok`.
- Hashes SHA-256 de los artefactos clave tras el archivado:
  - `tasks.md`: 61ddd10f0ef2217ead3bdba554557572e2557a023e593b889ba89d6db03ee47c
  - `verify-report.md`: 4a6f640957473c9e5d9f52a16890dce115b1b902a1a9404013048278feac2b7a
  - `openspec/INDEX.md`: 1941e58002c06b6d2cc82a44aca78ab4da6ed1cf4e784f8272351a8077ebadc6

## Hallazgos y Advertencias No Resueltos
Se registran sin corregirlos, según la regla de no reescribir informes históricos ni casillas:
- SUGGESTION 1 de `verify-report`: divergencia aceptada entre el lector de topología del plugin y el binario cuando falta el repositorio de specs de un multirepo.
- SUGGESTION 2 de `verify-report`: `sddPreflightDerivedBlock()` no recibe la topología; inocuo con cero grupos.
- SUGGESTION 3 de `verify-report`: `tasks.md` cita `screens/projects.js`; el fichero real es `internal/dashboard/assets/js/projects.js`. Pendiente de corregir si se quiere trazabilidad exacta.
- Limitaciones de `verify-report`: la primera ejecución en segundo plano de las pruebas del hook no produjo salida y se repitió en primer plano; no se ejecutaron paquetes completos de `internal/cli` ni `internal/components`; `bun` no está instalado.
- Lectura post-movimiento del status por nombre (ver arriba): no es evidencia del estado del cambio archivado.

## Trazabilidad
- Fuentes: ficheros del worktree de archivado. No se leyó ningún artefacto por ID de observación de Engram.
- Escritura en Engram (almacén hybrid): este reporte, con `topic_key` `sdd/preflight-topologia/archive-report`, proyecto `axiom`, `capture_prompt: false`.
- Puerta de integración: no se registró ninguna. El status nativo proyectó `dependencies.archive: ready` y el sellado es `continuous` sin puertas.
