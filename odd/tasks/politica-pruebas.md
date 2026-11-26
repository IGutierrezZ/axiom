# ODD: Política de pruebas: en local, solo lo tocado (politica-pruebas)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/politica-pruebas.md`. Espejo Engram: `odd/politica-pruebas/tasks`.
> Rama: `inc/politica-pruebas` · worktree `C:\repos\axiom-wt\politica-pruebas\axiom`. Cambio C-08-j del plan maestro (`docs/plan/backlog-2026-11/08-adelgazamiento.md`), ola A5.

## Objetivo

Que ni las instrucciones que genera Axiom ni las de este repositorio pidan baterías completas en local: al iterar y al cerrar una tarea se prueban los paquetes tocados, y la suite completa es de la CI.

## Problema y porqué

La regla de proporcionalidad pedía «all applicable full checks when a task closes», y el «Delegated Verification Gate» activaba el verificador independiente casi siempre por la regla «si no está claro, el tier más alto». Cada sesión pagaba baterías completas y verificadores que la CI de la PR ya cubre. Decisión del usuario (2026-11-26): aplicar la política de pruebas del análisis de agilidad.

## Decisiones

- Regla de proporcionalidad (`delegation.go`): comprobaciones de los paquetes tocados al iterar y al cerrar la tarea; la suite completa solo en la CI.
- «Delegated Verification Gate»: el verificador independiente queda para cambios de seguridad, ejecutables u operativos y de modo; sale «unclear» de ese tier y la regla «cuando no está claro, el tier más alto». Se mantienen la autoverificación del escritor, el sesgo de modelo pequeño y la comprobación del padre.
- Repositorio: `AGENTS.md:21` define «verificar en PASS» como los tests de los paquetes tocados; `AGENTS.md:25` pide paquetes tocados con `-run`, nunca `internal/cli` entero; `openspec/config.yaml` cambia `go test ./...` por los paquetes tocados y quita las skills `gentleman-*`.
- `.claude/CLAUDE.md` no está versionado (`.gitignore:36`), contra lo que dice el plan: es la salida local de `axiom sync` y recoge la regla nueva al sincronizar tras el merge. No se edita en la PR.
- Sin `## Spec viva`: ninguna spec de `openspec/specs/` fija la regla de proporcionalidad ni los tiers de verificación (búsqueda de «Proportionality», «risk tier», «independent verifier», «Delegated Verification»).
- Fuera de alcance: CI (C-08-k), `docs/testing/windows-tests.md` (describe hechos medidos, no la regla) y las skills `sdd-verify` y `work-unit-commits`, que ya piden comprobaciones focalizadas.

## Alcance autorizado

`internal/components/agentguidance/delegation.go` y sus tests, `internal/assets/skills/_shared/sdd-orchestrator-sections.md` (sección «Delegated Verification Gate») y sus tests, `testdata/golden/*` regenerados, `AGENTS.md` y `openspec/config.yaml`.

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y no hay elección del usuario. Comprobaciones: `go test` de los paquetes tocados con `-run`; la suite entera, en la CI.

## Entrega

`single-pr`: una PR con un commit por tarea. Etiqueta `type:feature`. Previsión: unas 80 líneas de autor más goldens.

## Tareas

- [ ] T1 · Regla de proporcionalidad en `delegation.go`, su test y goldens.
- [ ] T2 · Tiers del «Delegated Verification Gate» en la plantilla compartida, su test y goldens.
- [ ] T3 · `AGENTS.md` y `openspec/config.yaml` con la política de pruebas del repositorio.

Ruta de todas las tareas: inline (unidades de menos de ~100 líneas; sin unidades independientes que justifiquen escritores en paralelo).

## Criterio de terminado

- `rg 'all applicable full checks'` sin resultados fuera de `docs/plan/`.
- Goldens regenerados y revisados.
- Tests de `agentguidance`, `sdd` y de los paquetes de goldens en verde (con `-run`).

## Progreso

- Worktree creado desde `origin/main` (`3193c716`).

## Siguiente paso

T1.
