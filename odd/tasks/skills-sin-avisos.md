# skills-sin-avisos

Rama: `fix/skills-sin-avisos`. Origen: el aviso `project:skills` de `axiom doctor` en el checkout principal. Es el seguimiento que dejó abierto `odd/tasks/skillcheck.md`.

## Objetivo

Que `axiom skill check` no informe de errores ni de avisos causados por las skills versionadas en el repositorio. Así `project:skills` del doctor deja de avisar por ellas.

## Problema y porqué

Sobre `main` (`13d2832f`), `axiom skill check` informa de 1 error y 8 avisos.

Hallazgos del repositorio:

- `skills/branch-pr/SKILL.md` declara `name: axiom-branch-pr`, que no coincide con su carpeta (`name-mismatch`, error).
- Cuatro skills tienen una `description` de más de 160 runas (`description-budget`): `issue-root-resolution` (205), `knowledge-crawl` (265), `knowledge-query` (297) y `systemic-issue-triage` (205). De esta última hay dos copias idénticas: `skills/` e `internal/assets/skills/`.
- `internal/assets/skills/sdd-propose/SKILL.md` cita `references/openspec-adoption.md` como si fuera propia (`reference-missing`). En realidad es de `sdd-spec`.

Hallazgos del entorno, fuera del repositorio:

- La copia instalada `.claude/skills/sdd-propose` repite el aviso de la referencia.
- `.claude/skills/judgment-day` choca con `internal/assets/skills/judgment-day` (`collision`). El binario instalado (`v3.7.1-0.20261123045640-d580368d71ee`) es anterior a `7b17ef1c`, que retiró RDD el 2026-11-25.
- Ambos se resuelven al reinstalar `axiom` desde `main` y repetir `axiom install --scope workspace --agent claude-code`.

## Alcance y restricciones

- Solo skills versionadas, el índice de `AGENTS.md` y las pruebas que fijan sus textos.
- Al renombrar se conserva el nombre `axiom-branch-pr`, porque `branch-pr` es el nombre retirado de la skill de upstream (`internal/components/skills/retired.go`).
- Las descripciones recortadas conservan `Trigger:` y los disparadores principales, sin cambiar de idioma.
- Fuera de alcance:
  - reinstalar el binario o las copias de `.claude/`, que depende de la persona;
  - los documentos históricos (planes, ledgers, `odd/tasks` y `openspec/changes/archive`) que citan `skills/branch-pr/`.
- TDD: no configurado (`axiom.yaml` no lo declara). Comprobaciones funcionales con `go test` por paquetes, sin binarios sueltos.
- Entrega: `single-pr`, un commit por tarea.

## Tareas

- [x] T1 — Renombrar `skills/branch-pr/` a `skills/axiom-branch-pr/` y actualizar su fila en el índice de `AGENTS.md`. Ruta: inline (cambio mecánico, menos de 100 líneas).
- [x] T2 — Recortar a 160 runas o menos las `description` de `issue-root-resolution`, `knowledge-crawl`, `knowledge-query` y las dos copias de `systemic-issue-triage`, y regenerar sus filas en `AGENTS.md`. Ruta: inline.
- [ ] T3 — Hacer que `sdd-propose` cite la guía de adopción como `sdd-spec/references/openspec-adoption.md` y ajustar `TestSDDSpecAndProposePointToTheAdoptionReference`. Ruta: inline.

## Criterios de aceptación

- Tras las tres tareas, `axiom skill check --path <worktree>` no informa de ningún `error` ni `warning` en ficheros versionados.
- `go test ./internal/skillregistry/ ./internal/assets/ ./internal/skillcheck/ ./internal/catalog/` pasa, incluido `TestRepoAgentsIndexMatchesVersionedSkills`.

## Progreso y evidencias

- T1:
  - `git mv skills/branch-pr skills/axiom-branch-pr` y fila de `AGENTS.md` con la ruta nueva.
  - `go test ./internal/skillregistry/ -run TestRepoAgentsIndex -count=1`: `ok` (12,1 s).

- T2:
  - Las descripciones quedan en 155, 157, 159 y 156 runas. Las filas de `AGENTS.md` se cambian con el mismo texto.
  - `go test ./internal/skillregistry/ ./internal/skillcheck/ ./internal/catalog/ -count=1`: `ok`.
  - `./internal/assets/` falla por T1: `issue_creation_authority_test.go:77,119` leía `skills/branch-pr/SKILL.md`. Se corrige con un fixup de T1.

## Siguiente paso

Fixup de T1 y T3.
