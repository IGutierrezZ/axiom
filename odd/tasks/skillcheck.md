# skillcheck — validador de skills (C-04-d)

## Objetivo

Una única fuente de reglas para validar skills (`SKILL.md`) que se aplica en el build (test de assets), al traer una skill (puertas en `axiom skill approve`, el minero y `agentbuilder`) y bajo demanda (`axiom doctor` y `axiom skill check [--json]`).

## Problema y porqué

Solo las skills embebidas se validan, y solo en `TestSkillFrontmatterIsLintClean`. Las minadas, las aprobadas desde el buzón, las generadas por `agentbuilder` y las copiadas a mano no las revisa nadie; el registro las tolera en silencio (descarta reservadas, resuelve colisiones sin aviso) y `axiom doctor` no mira skills. Origen: `docs/plan/backlog-2026-11/04-consumo-agentes.md` (C-04-d); carril ODD por P9 (apartado 6.2 del README).

## Alcance autorizado

- Paquete nuevo `internal/skillcheck/`.
- `internal/assets/skills_frontmatter_test.go` delega en `skillcheck`.
- Puertas: `Manager.Approve` y la escritura del minero en el buzón (`internal/autoskill`), `agentbuilder.Install`.
- Comprobación `project:skills` en `axiom doctor` (`internal/doctor`, `internal/cli/doctor*.go`).
- Subcomando `axiom skill check [--json] [--path]` (`cmd/axiom/main.go`, solape blando con C-01-d y C-02-c).
- Lectura de `internal/skillregistry` y, si hace falta, helpers exportados de solo lectura (frescura del registro, resolución de la copia elegida) sin cambiar su comportamiento.
- Documentación: `docs/usage.md` (tabla de comprobaciones de `doctor`), `docs/manual-de-inicio.md` y `docs/skill-registry.md` si listan los subcomandos.

## Fuera de alcance

Instalar skills en Pi; que `approve` instale la skill en las carpetas de los runtimes; retirar `skills/branch-pr/`; límite de tamaño total de una skill; endurecer la carga del registro (sigue tolerante: `skillcheck` informa y solo bloquea en las puertas de entrada).

## Decisiones de diseño (de esta sesión, a validar en la parada de plan)

1. **Reglas y severidad por perfil.** Una sola tabla de reglas con dos perfiles:
   - **Estructurales** (error en ambos perfiles): frontmatter al inicio (`---`, tolerando los comentarios `<!-- section:model-… -->` iniciales) y cerrado; líneas de primer nivel parseables; `name` presente e igual al directorio; `description` presente y no vacía; nombre no reservado (`_shared`, `skill-registry`, prefijo `sdd-`, salvo en el perfil canónico de assets, que los contiene).
   - **Canónicas** (error en el perfil `Canonical`, aviso en `Portable`): `description` entrecomillada, de una línea, con `Trigger:` y dentro del presupuesto (160 runas; 205 para `systemic-issue-triage`); solo las claves `name`, `description`, `license`, `metadata`, `version`, `user-invocable`, `disable-model-invocation`. `metadata:` con hijos indentados (formato de C-04-a: `origin`/`trigger` bajo `metadata:`) es válido.
   - **Referencias** (aviso): enlaces Markdown relativos y rutas `references/…`/`assets/…`/`scripts/…` que no existen dentro de la carpeta de la skill.
   - Perfil `Canonical`: assets embebidos, minero y skills con `origin: mined`. Perfil `Portable`: skills de terceros (buzón con origen `midudev`, que llegan verificadas por SHA y no se pueden reescribir), `agentbuilder` (su frontmatter lo compone el código) y las carpetas de proyecto y usuario en `doctor` y `skill check`.
2. **Puertas.** `approve` valida la carpeta del buzón antes de copiar y rechaza con los errores; el minero valida su `SKILL.md` antes de depositarlo; `agentbuilder.Install` valida antes de escribir y no escribe nada si falla.
3. **`agentbuilder` no genera frontmatter hoy** (`prompt.go`: «starting with `# {Title}`»): sin cambio, la puerta rechazaría todo agente generado. Se compone el frontmatter en código a partir de `Name`, `Description` y `Trigger` (como hizo C-04-a con el minero), con la descripción en una línea y `Trigger:`.
4. **Comprobaciones de conjunto** (solo `doctor` y `skill check`, nunca bloquean): sombras y colisiones (mismo nombre en varias carpetas; aviso si el contenido difiere, informativo si es copia idéntica; indica qué copia elige el registro); alcance por carpeta (qué runtimes cargan la carpeta de cada skill, razonando por carpeta porque varias se comparten, p. ej. `.agents/skills`); Pi solo emite el aviso informativo «las skills llegan por gentle-pi, fuera de Axiom»; frescura del registro (huella de la caché `.atl/.skill-registry.cache.json` frente a la calculada).
5. **`doctor`.** Comprobación `project:skills` (ámbito proyecto, solo con raíz): `warn` si hay errores o avisos, `pass` si no; nunca `fail` (el registro las tolera).
6. **`axiom skill check [--json] [--path <dir>]`.** Informe por skill (ruta, nombre, perfil, hallazgos con severidad y regla) más los hallazgos de conjunto; código de salida distinto de cero si hay algún error.

## Tareas

- [x] **T1** — Paquete `internal/skillcheck`: parser de frontmatter, reglas con severidad por perfil, comprobación de referencias, API `CheckFile`/`CheckDir`; tests de tabla. `TestSkillFrontmatterIsLintClean` delega en él (perfil `Canonical`, presupuesto especial conservado). Ruta: delegada (writer; 2+ ficheros no triviales).
- [x] **T2** — Puertas: `Manager.Approve` (perfil según `origin` de `metadata.json`), minero antes de depositar, `agentbuilder.Install` con frontmatter compuesto en código. Tests de rechazo con mensaje. Ruta: delegada (writer).
- [x] **T3** — Comprobaciones de conjunto en `skillcheck` (escaneo de carpetas de proyecto y usuario de `skillregistry`, sombras, alcance por carpeta inyectado desde los adaptadores, aviso de Pi, frescura) y comprobación `project:skills` de `doctor`. Ruta: delegada (writer).
- [x] **T4** — `axiom skill check [--json] [--path]` en `cmd/axiom`, ayuda del grupo `skill`, tests en proceso; documentación (`docs/usage.md`, `docs/manual-de-inicio.md`, `docs/skill-registry.md`). Ruta: delegada (writer).

## Criterio de aceptación

- `TestSkillFrontmatterIsLintClean` delega en `skillcheck` y sigue en verde para todas las skills embebidas.
- `axiom skill approve` rechaza una skill del buzón con frontmatter inválido y explica qué falló.
- `axiom doctor` informa de frontmatter roto, referencias rotas, colisiones o sombras y registro desactualizado; en Pi solo avisa.
- `axiom skill check --json` devuelve un informe por skill y sale con código distinto de cero si hay errores.
- `go test ./internal/skillcheck/... ./internal/autoskill/... ./internal/agentbuilder/... ./internal/assets/... ./internal/doctor/...`, `go test ./cmd/axiom/ -run Skill` y `go test ./internal/cli/ -run Doctor` en verde.

## Comprobaciones

- TDD: desactivado (sin configuración de proyecto ni de sesión); comprobaciones funcionales ordinarias con `go test` por paquetes.
- Sin binarios sueltos de `axiom` (el EDR los borra): todo en proceso con `go test`.
- RDD: según el interruptor del usuario (`axiom review mode status`).

## Entrega

- Rama `inc/skillcheck`, worktree `C:\repos\axiom-wt\skillcheck\axiom`.
- Estrategia: `exception-ok` (una sola PR; `size:exception` si pasa de 400 líneas; previsión ~900 líneas con tests). Etiqueta `type:feat`.
- Push, PR y fusión los decide el usuario.

## Progreso

Plan stop: approved 2026-11-18; implementation continues

- T1: hecho; checks: go test ./internal/skillcheck/... ./internal/assets/ -run Skill ./internal/autoskill/...: ok
- T2: hecho; checks: go test ./internal/autoskill/... ./internal/agentbuilder/... ./internal/tui/ -run AgentBuilder ./cmd/axiom/ -run Skill: ok
- T3: hecho; checks: go test ./internal/skillcheck/... ./internal/skillregistry/... ./internal/doctor/... ./internal/cli/ -run Doctor|ProjectSkills|SkillFolderRuntimes: ok
- T4: hecho; checks: go test ./cmd/axiom/ -run Skill: ok; reejecutados T1-T3 + go vet + gofmt -l limpios

## Siguiente paso

T1.
