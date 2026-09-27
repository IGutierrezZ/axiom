# ODD: Mitigación Inmediata del Consumo Masivo de Minutos de CI (GitHub Actions)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/ci-cost-reduction.md`.  
> Espejo de recuperación en Engram: topic `odd/ci-cost-reduction/tasks`, proyecto `axiom`.

## Objetivo

Auditar y erradicar de forma inmediata el sobreconsumo masivo de minutos de cómputo en GitHub Actions (que generó >$130 en metered usage en septiembre), desactivando disparadores cron automáticos sin supervisión humana y restringiendo runners de alto coste (`macos-latest` y matriz de 10 shards de `windows-latest`) a ejecuciones manuales bajo demanda (`workflow_dispatch`).

1. **Eliminación de Crons Automáticos:**
   - Desactivar o eliminar disparadores de tipo `schedule` en `.github/workflows/ci.yml` (03:00 UTC) y `.github/workflows/windows-full-suite.yml` (03:20 UTC).
2. **Restricción de Runners de Alto Coste:**
   - En `windows-full-suite.yml`: eliminar disparador automático `push` en `main`; conservar únicamente `workflow_dispatch` manual.
   - En `ci.yml`:
     - Condicionar `darwin-runtime` (`macos-latest`, multiplicador 10x) a `workflow_dispatch`.
     - Condicionar `windows-runtime` (`windows-latest`, multiplicador 2x) a `workflow_dispatch`.
     - Restringir la matriz de `organic-runtime-e2e` a `[ubuntu-latest]` para evitar runners Windows en cada PR/push.
3. **Auditoría de Workflows Heredados y Propuesta de Reducción:**
   - Identificar dependencias obsoletas heredadas de Gentle-AI (`discord-notifications.yml`, jobs de docker multinúcleo en `e2e-tests`, contratos OpenCode 2.0.4).
   - Elaborar propuesta de arquitectura CI mínima y eficiente para Axiom.

---

## Tareas

- [x] **T1 · Desactivar crons y triggers automáticos en `windows-full-suite.yml`**
  - Eliminado bloque `schedule` (cron 03:20 UTC) y `push: branches: [main]`.
  - Workflow dejado exclusivamente bajo `workflow_dispatch`.
- [x] **T2 · Eliminar cron y restringir runners caros en `ci.yml`**
  - Eliminado bloque `schedule` (cron 03:00 UTC).
  - Restringido `darwin-runtime` (`macos-latest`) a `workflow_dispatch`.
  - Restringido `windows-runtime` (`windows-latest`) a `workflow_dispatch`.
  - Reducida matriz de `organic-runtime-e2e` a `os: [ubuntu-latest]`.
- [x] **T3 · Desactivar notificaciones heredadas de Gentle-AI (`discord-notifications.yml`)**
  - Restringido a `workflow_dispatch` para evitar fallos por webhooks inexistentes (`DISCORD_GENTLE_AI_*`).
- [x] **T4 · Verificación de Sintaxis y Comprobación de Workflows**
  - Verificado que ningún workflow conserva disparadores `schedule` activos.
  - Verificado que los runners caros (`macos-latest` y `windows-latest`) no se disparan en eventos `push` o `pull_request`.
- [x] **T5 · Elaboración de la Propuesta de Arquitectura CI Mínima**
  - Desarrollada auditoría de deuda heredada de Gentle-AI y propuesta de arquitectura CI optimizada para Axiom.
- [ ] **T6 · Commit y Apertura de PR según Gobernanza ODD**
  - Crear commit semántico y generar PR hacia `main`.

---

## Verificación Ejecutable

- Ausencia total de eventos `schedule` en `.github/workflows/`: confirmado con 0 coincidencias activas.
- Runners `macos-latest`: restringido a `workflow_dispatch` (ahorro ~100 min facturables por PR/push).
- Runners `windows-latest`: restringidos a `workflow_dispatch` (ahorro ~600 min facturables diarios por cron y ~30 min por PR/push).
- Aislamiento en worktree `C:\repos\axiom-wt\ci-cost-reduction` sobre rama `fix/ci-cost-reduction`.
