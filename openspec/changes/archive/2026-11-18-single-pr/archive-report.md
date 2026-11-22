# Informe de archivado: single-pr (C-02-c)

Fecha de archivado: 2026-11-18.
Carril: SDD. Almacén de artefactos: `hybrid`. Estrategia de entrega: `single-pr`.

## Resumen

El cambio `single-pr` queda archivado. Su código ya estaba fusionado en `main` mediante la PR #280 (merge `5849fe17`). En este archivado se integraron sus deltas en las especificaciones vivas, se movió la carpeta del cambio a `openspec/changes/archive/2026-11-18-single-pr/` y se escribió este informe. No se hizo ningún push, fetch ni commit.

## Especificaciones integradas

Cada delta se compuso con `axiom sdd archive-compose` (`--canonical` la spec viva, `--delta` la spec del cambio). El resultado se escribió primero en un directorio temporal, se comprobó y después se copió sobre la spec viva.

| Capacidad | Requisitos vivos antes | Requisitos vivos después | Cambios integrados |
|---|---|---|---|
| `sdd-preflight-configuration` | 14 | 15 | MODIFIED REQ-28.14 a REQ-28.18 y REQ-28.20 a REQ-28.22; ADDED REQ-28.23 |
| `sdd-orchestrator-assets` | 20 | 21 | 7 MODIFIED de la cadena de entrega; ADDED REQ-32.3 |
| `organic-agent-trigger-rules` | 19 | 20 | ADDED REQ-32.4 |
| `workspace-topology` | 13 | 15 | MODIFIED REQ-2.1; ADDED REQ-32.1 y REQ-32.2 |

Comprobaciones posteriores a la integración:
- Los 21 bloques de requisito de los deltas aparecen literalmente en las specs vivas resultantes.
- No hay requisitos duplicados ni marcadores de delta (`## ADDED`, `## MODIFIED`) en las specs vivas.
- No hubo conflictos al integrar los deltas.

## Commits del cambio

Rango fusionado en `main` (PR #280, merge `5849fe17`):

- `cc55c9e8` docs(openspec): planificar single-pr (C-02-c)
- `607c927b` feat(workspace): bloque delivery con split_hint_lines en axiom.yaml
- `020de4a2` refactor(sdd): pre-vuelo sin estrategia de PR y bloque de cinco líneas
- `38087c55` refactor(odd): entrega single-pr por defecto en la doctrina ODD
- `b111935b` refactor(sdd): entrega single-pr en orquestadores y fases sin guard de carga
- `8e1cfdc0` docs(skills): chained-pr a petición y work-unit-commits sin presupuesto
- `bf991897` test(assets): guarda contra umbrales de entrega residuales
- `7d62546b` docs(openspec): informe de verificación de single-pr

## Pull request

PR #280, `inc/single-pr` hacia `main`, fusionada con el commit `5849fe17`.

## Verificación

Resultado del informe de verificación: PASS con observaciones, 0 CRITICAL, 0 WARNING y 3 SUGGESTION. Las 17 tareas de `tasks.md` estaban completadas.

Sugerencias registradas (no bloqueantes):
1. `internal/cli/sdd_preflight_hook.go:432` y las tablas TS conservan `ask-on-risk` y `auto-chain` como vocabulario heredado tolerado (REQ-28.16). Conviene un comentario junto a la tabla Go que lo explique.
2. `skills/branch-pr/SKILL.md` e `issue-creation` siguen citando `size:exception`. Es coherente mientras C-02-e no retire el check de CI, y debe revisarse en ese cambio.
3. Los tests de `bench/` y `go test ./internal/cli/` completo no se ejecutaron. La ruta del pre-vuelo sí está cubierta por `-run SDDPreflight`.

## Residuo de `400` aceptado fuera de alcance

El barrido `rg -n "\b400\b" internal/assets internal/components internal/cli skills` solo muestra residuo permitido: umbral de barrido de judgment-day (`jd-judge-a` y `jd-judge-b`), lentes `review-*` de RDD, `rdd-defect-workflow`, literales de migración en `internal/components/sdd/inject.go`, `internal/reviewtransaction/risk.go`, comentarios y aserciones negativas en tests, y `.github/workflows/pr-check.yml:25` (`limit = 400`), que se mantiene hasta el cambio C-02-e. El guard `TestDeliveryThresholdResidue` fija ese residuo por ruta y contenido. Queda aceptado fuera del alcance de este cambio.

## Comandos ejecutados y resultados

- `axiom sdd archive-check --cwd <wt> --change single-pr --json`: `ok: true`, `iso: false`, sin hallazgos. Un cambio no ISO sale sin error.
- `axiom sdd archive-compose` (fusión de cada delta sobre su spec viva): código de salida 0 en las cuatro capacidades.
- `axiom sdd archive-check --cwd <wt> --change 2026-11-18-single-pr --json` tras el archivado: error `ya esta archivado` (REQ-21.18), lo que confirma que el cambio se reconoce en su nueva ubicación.
- `diff -r` entre el árbol de `HEAD` (`openspec/changes/single-pr`, extraído con `git archive`) y `openspec/changes/archive/2026-11-18-single-pr`: sin diferencias (código de salida 0).
- `diff -r` entre cada spec compuesta en el directorio temporal y su spec viva tras la copia: sin diferencias en las cuatro capacidades.
- `go test -count=1 ./internal/workspace/ -run "Docs|Spec|OpenSpec"`: ok.
- `go test -count=1 ./internal/livespec/`: ok.
- `go test -count=1 ./internal/livingdoc/`: ok.

## Estado de Git

- `git mv` del cambio (10 ficheros renombrados) queda en el índice.
- Las cuatro specs vivas quedan modificadas sin preparar.
- Este informe queda sin preparar.

Pendiente para el responsable: revisar y confirmar el commit del archivado.

## Trazabilidad

No se proporcionaron identificadores de observaciones de Engram en el contexto de este archivado. El informe se guarda también en Engram con el tema `sdd/single-pr/archive-report`.
