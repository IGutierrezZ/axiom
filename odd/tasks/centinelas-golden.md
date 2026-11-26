# ODD: Centinelas de prosa: borrar los que fijan redacción (C-08-h)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/centinelas-golden.md`.
> Espejo de recuperación en Engram: topic `odd/centinelas-golden/tasks`, proyecto `axiom`.
> Fuente: `docs/plan/backlog-2026-11/08-adelgazamiento.md`, entrada C-08-h (revisión 10), y apartado 6.2 del README del plan. El slug `centinelas-golden` se conserva aunque el enfoque ya no es pasar a goldens.

## Objetivo

Que cambiar la redacción de una skill, de una instrucción o de un asset no obligue a tocar tests. Se borran los centinelas que fijan redacción y las lápidas de ausencia; se conservan solo los contratos que lee el código.

## Problema y porqué

El 86 % de los commits que solo tocaban prosa de `internal/assets` tuvo que tocar tests (diagnóstico de agilidad, revisión 10). Un golden traslada la fricción a `-update`, no la quita: el usuario decidió borrar (2026-11-26).

## Alcance

Se **borran**:

- aserciones de ausencia de frases retiradas;
- `Contains` sobre redacción libre que ningún código consume;
- la misma frase comprobada en varios paquetes (como mucho, una);
- tests de funcionalidades retiradas que no protegen una migración;
- lápidas `*Retired`, `*Removed`, `*Absent`, `*NoLonger`, `*Legacy`, `*NotContain`, salvo las de migración.

Se **conservan**:

- tokens y marcadores que parsea el código (`{{AXIOM_*}}`, `axiom.sdd-integration.consent/v1`, `granted`/`declined`, comandos y flags, claves YAML);
- invariantes de seguridad (permisos, nada de push sin autorización);
- paridad de plantillas entre runtimes;
- tolerancia a estado antiguo y migraciones (por ejemplo, `state/retired_rdd_roles_test.go`, limpieza en `uninstall`);
- goldens que ya renderizan el asset completo.

Fuera de alcance: tests de comportamiento que además comprueban texto (solo se recortan sus aserciones de prosa redundantes); tests de `internal/knowledge`, `internal/semantic` y `dashboard/knowledge_assets_test.go` (C-08-l); tests del modo ISO; paralelizar tests (C-08-g).

## Restricciones

- Carril ODD, una sola PR, commit por tarea, Conventional Commits.
- Pruebas localizadas (paquetes tocados y `-run`); la batería completa la pasa la CI.
- Cada token retirado se justifica en el commit (por grupo cuando comparten motivo).
- TDD: desactivado (sin configuración en `axiom.yaml`; cambio de borrado). Comprobaciones funcionales: `go vet` y `go test` de los paquetes tocados.

## Tareas

Ruta: delegada en paralelo (disparador de escritor por paralelismo: cuatro superficies disjuntas, cada una de varios minutos). Los escritores no hacen commit; el padre revisa el diff y hace el commit de cada tarea.

- [x] **T1 · `internal/assets`**: centinelas de redacción y lápidas de todos sus `*_test.go`, salvo ISO (`sdd_iso_pins_test.go`) y guardas de CI (`workflow_shards_test.go`). Commit `fc923349`.
- [x] **T2 · `internal/components/sdd`**: tests de contrato y vocabulario, y aserciones de prosa redundantes de `inject_test.go`. Commit `94e23083`.
- [x] **T3 · `internal/components/agentguidance` e `internal/workspace`** (`docs_contract_test.go` y lápidas del paquete). Commit `3d751d1f`.
- [x] **T4 · Lápidas del resto del repositorio** (`cli`, `persona`, `filemerge`, `sddstatus`, `state`, `uninstall`, `engram` y demás), salvo migraciones y lo excluido. Commit `84e46ffe`.
- [x] **T5 · Prueba de la fricción** (no se commitea) y cierre: métricas antes/después.

## Criterios de aceptación

- Líneas de test de `internal/assets` y de `components/sdd` antes y después, registradas aquí.
- Cada token retirado con su justificación en el commit.
- Goldens que queden, regenerados con `-update` y revisados (si alguno cambia).
- Tests de los paquetes tocados en verde.
- Reescribir una frase no contractual de `internal/assets/skills/sdd-apply/SKILL.md` hace fallar como mucho goldens regenerables.

## Métricas

| Medida | Antes (`7464c5a7`) | Después |
|---|---|---|
| Líneas de test de `internal/assets` | 8.644 | 4.149 |
| Líneas de test de `internal/components/sdd` | 17.315 | 15.073 |
| Líneas de test de `agentguidance` + `workspace/docs_contract_test.go` | 2.405 + 668 | 1.370 + 39 |
| Funciones lápida (`*Retired`…`*NotContain`) | 314 | 286 |
| Líneas de `*_test.go` del repositorio (sin `bench/`) | 245.031 | 235.824 |

Balance: 63 ficheros, +277 / −9.408 líneas. Queda por debajo de la estimación del plan (10.000-15.000) porque las lápidas restantes se revisaron una a una y ejercitan código vigente de migración, lectores tolerantes, limpieza de `uninstall`/`sync` o rechazos con contrato, que el plan manda conservar. En `components/sdd` se conservan los tests de comportamiento de `Inject` (solo se recortó su prosa redundante).

## Progreso y evidencias

- Ruta: cuatro escritores delegados en paralelo (superficies disjuntas); el padre revisó cada informe, hizo una segunda pasada de T4 revisando las lápidas una a una y commiteó por tarea con la justificación de cada token en el cuerpo del commit.
- `go vet` y `go test -count=1` en verde de `internal/assets`, `internal/components/sdd`, `internal/components`, `internal/components/agentguidance`, `internal/app`, `internal/sddstatus`, `internal/branch/registry`, `internal/dashboard` e `internal/workspace`; `internal/cli` con `-run` de los tests tocados y compilación del paquete.
- Goldens: ninguno cambió; no hizo falta `-update`.
- Workflows y `.refusal-ratchet-baseline.txt`: ningún nombre borrado aparece en ellos.
- Prueba de la fricción (no commiteada): renombrar el encabezado `#### Step 2c: Branch and Worktree (`axiom branch`)` de `internal/assets/skills/sdd-apply/SKILL.md`, que en `7464c5a7` fijaba `assets_test.go`, y reescribir la frase «Delivery only chooses the target branch…» deja en verde `internal/assets`, `internal/components/sdd`, `internal/components` y `agentguidance`. Revertido.
- Fallo previo, fuera de alcance: `TestScriptsAxiomWorktreePowerShellDrivesAxiomBranch` (`internal/workspace`, solo Windows con PowerShell) falla igual en `main` (`7464c5a7`): el test llama a `pr` sin `-Label` y el envoltorio lo exige.

## Siguiente paso

PR única con `scripts/axiom-worktree.ps1 pr centinelas-golden -Lane odd -Label chore`.
