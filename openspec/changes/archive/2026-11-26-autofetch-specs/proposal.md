# Propuesta: Autofetch del repositorio de specs — avisar y avanzar (C-05-d)

## Intención

Hoy nadie ve, sin pedirlo, lo que otras personas han subido a OpenSpec o a la rama de su cambio: la regla vigente («Never pass `--fetch` without the explicit authorization of the user») y RF-BRA-10 («ninguna orden hace red sin opción explícita») obligan a pedir autorización en cada `fetch`. El resultado es una rama de integración local desfasada, cesiones (`axiom sdd publish`) que la siguiente persona no detecta y la tentación de hacer `git fetch` sin autorización.

El cambio añade un **interruptor personal de autofetch**, opt-in y apagado por defecto, guardado en el hub (`~/.axiom/workspaces.json`, RF-HUB-07) y nunca en `axiom.yaml`. Encenderlo es la autorización explícita y persistente del `fetch`. Con el interruptor encendido, Axiom trae el remoto, avanza la rama de integración del repositorio de specs solo con fast-forward estricto y avisa únicamente de lo relevante para el cambio en curso. Con el interruptor apagado, el comportamiento es idéntico al actual: ninguna orden hace red sin `--fetch`.

Éxito: la persona que activa el interruptor ve al iniciar sesión, al preparar ramas y en cada parada los commits ajenos que afectan a su cambio, y su integración local está al día sin que Axiom mueva nunca una rama de cambio ni haga push.

## Alcance

### Dentro del alcance

- **Preferencia personal `autofetch`** por proyecto en el registro del hub (`WorkspaceRecord`), booleana, apagada por defecto y ausente en el JSON mientras está apagada (esquema `1.0` intacto). Proyecto no registrado o hub ilegible ⇒ apagado, con aviso; nunca red por defecto.
- **Comando propio** `axiom autofetch enable|disable|status|run [--cwd] [--agent] [--json]`, fuera de la familia `axiom branch` para no ampliar sus cinco verbos (RF-BRA-01). `enable`/`disable` editan la preferencia; `status` la muestra; `run` ejecuta el ciclo descrito abajo.
- **Ciclo `run`** (solo con el interruptor encendido):
  - `git fetch origin` del repositorio de specs (en `monorepo-embedded`, el propio repositorio; en `multirepo`, el repositorio de specs in situ), sin `--prune`, con timeout de unos 8 s, `GIT_TERMINAL_PROMPT=0` y un intervalo mínimo entre fetches.
  - Avance de la **rama de integración** local con `merge --ff-only`, solo si el árbol está limpio y no hay divergencia, en el checkout donde esa rama está desplegada. Si no se puede (árbol sucio, divergencia, rama no desplegada), solo avisa con la orden exacta que la persona puede ejecutar.
  - `main_branch` nunca se avanza; la rama de un cambio y los worktrees de feature nunca se mueven: ahí solo se avisa.
  - **Aviso por relevancia**: commits entrantes en `origin/<integration_branch>` que tocan `openspec/changes/<cambio actual>/`, las specs afectadas por el cambio u `odd/tasks/<feature>.md`, y commits nuevos en `origin/<rama del cambio>`. Lo demás no se avisa.
  - Un remoto inalcanzable o un timeout no bloquea: deja un aviso y termina con éxito.
- **Fetch implícito antes de `axiom branch ensure`**: con el interruptor encendido, `ensure` se comporta como con `--fetch` en todos los repositorios del plan, incluidos los de código, para crear las ramas sobre lo último de `origin`; en los repositorios de código no se avanza ninguna rama local.
- **Disparadores**:
  - Hook `SessionStart` nuevo que ejecuta `axiom autofetch run --agent <runtime>`, sin bloquear y fail-open (código de salida siempre 0), en Claude Code (hoy sin `SessionStart`) y Codex (que ya usa `SessionStart`); instalación idempotente por `axiom sync` y retirada por la desinstalación. Inerte fuera de proyectos Axiom y con el interruptor apagado.
  - Antes de `axiom branch ensure` (fetch implícito, punto anterior).
  - Antes de presentar cada parada: la doctrina del orquestador ejecuta `axiom autofetch run` (respeta el intervalo mínimo) y relaya el aviso.
- **Doctrina y documentación**: la regla pasa a «Never pass `--fetch` without the explicit authorization of the user, unless the user enabled autofetch» o equivalente, en `routing.go`, `sdd-orchestrator-sections.md`, `sdd-apply/SKILL.md`, `AGENTS.md`, `GEMINI.md` y `docs/manual-de-inicio.md`; regeneración de los 10 goldens afectados y actualización de las pruebas que fijan la frase.
- **Deltas de spec** de las capacidades listadas abajo.

### Fuera del alcance

- `pull`, `merge` o `rebase` de ramas de cambio; cualquier movimiento de worktrees de feature.
- `push`, `prune` y cualquier escritura remota.
- Avance de ramas locales en repositorios de código (solo fetch antes de `ensure`).
- Cualquier memoria de Engram (C-05-a retiró `.engram/manifest.json`).
- Autorización por `axiom.yaml`: es una preferencia personal del hub.
- La pantalla de UI del interruptor: la aporta C-06-e (P16); este cambio deja la preferencia y el comando listos para que la consuma.
- Sección `remote` en `axiom branch status --json`: `status` sigue siendo solo local (RF-BRA-16); el aviso sale de `axiom autofetch run|status --json`.
- Hooks `SessionStart` en runtimes sin canal demostrado (Gemini, Cursor, Antigravity, Kiro, Pi, OpenCode por esta vía): allí el disparo es solo doctrinal.
- El fetch de `axiom branch done` (ver «Preguntas abiertas»).

## Capacidades

### Capacidades nuevas

- Ninguna.

### Capacidades modificadas

- `change-branch-lifecycle`:
  - MODIFIED RF-BRA-10: el interruptor encendido cuenta como opción explícita; `ensure` hace el fetch implícito y `fetched` refleja el fetch real; con el interruptor apagado, sin cambios.
  - MODIFIED RF-BRA-01: la familia `axiom branch` puede hacer red con `--fetch` **o** con el interruptor encendido; se mantienen los cinco subcomandos.
  - MODIFIED tabla «Requerimientos de un vistazo» y «No cubre»; RF-BRA-13 solo si cambia la sugerencia de `--fetch` de `change_not_integrated`.
  - ADDED RF-BRA-39 «Autofetch del repositorio de specs con el interruptor personal»: fetch con timeout e intervalo, avance ff estricto de `integration_branch`, prohibiciones (`main_branch`, ramas de cambio, repositorios de código, push, prune), degradación a aviso con la orden exacta, relevancia del aviso y remoto inalcanzable sin bloqueo.
  - ADDED (o dentro de RF-BRA-39, a decidir en spec) la instalación del hook `SessionStart` fail-open e inerte por `axiom sync`, análoga a RF-BRA-23.
- `multi-project-hub`:
  - MODIFIED RF-HUB-07: nueva preferencia personal `autofetch` por proyecto, apagada por defecto, nunca en `axiom.yaml`; persistencia, valor por defecto, re-registro y desregistro; un binario anterior la ignora.
  - MODIFIED RF-HUB-08 o ADDED RF-HUB-13: cómo se edita la preferencia (`axiom autofetch enable|disable`) y que no se pregunta al registrar (pendiente de confirmar).
- `sdd-block-review-gates` (condicional): solo si se reescribe `internal/assets/generic/stop-exits.md` (RF de la salida «tras `git fetch` o `axiom branch ensure --fetch` con su autorización») o si su spec fija el contenido previo a presentar una parada. La recomendación es no tocar `stop-exits.md`, porque encender el interruptor ya es autorización y la frase sigue siendo cierta.

`sdd-orchestrator-assets` no cambia: RF-ORQ-26 solo prohíbe el `SessionStart` de `axiom review stop-hook`. `sdd-docs-publication`, `sdd-independent-review` y `requirement-trace` tampoco: `publish`, `scope` y la verificación conjunta siguen sin hacer fetch.

## Enfoque

Enfoque híbrido recomendado por la exploración (opción C):

1. **Git**: primitiva de escritura `FastForward` (`merge --ff-only`) en `internal/branch/gitcli`, ejecutada con `gitcommon.RunWithTimeout`; el `Fetch` existente se reutiliza con timeout. `merge --ff-only` no está en los literales vigilados por `sddpublish/isolation_test.go` (`push`, `commit`) ni es operación remota para `gitcommon/architecture_test.go`.
2. **Motor**: método `Autofetch` en `internal/branch/engine` que reutiliza `Policy`, roles del planificador (`repositoryRole`: planning home = repositorio de specs), `DirtyPaths`, `IsAncestor`, `AheadBehind`, `RevList` y `CommitFiles` para el avance y la relevancia. `ensure` combina `r.Fetch || prefs.autofetch`.
3. **Preferencia**: campo `Autofetch` en `hub.WorkspaceRecord` con `LookupAutofetch`/`SetAutofetch` en `hub/prefs.go`; el puerto `branch.Prefs` gana un método para leerla. Estado del intervalo mínimo fuera del hub (mtime de `FETCH_HEAD` o fichero propio bajo el git common dir; lo decide design).
4. **CLI**: verbo de primer nivel `axiom autofetch` en `cmd/axiom/main.go` + `internal/cli`, con salida `additionalContext` para el hook, siguiendo el patrón de `axiom branch guard`.
5. **Hooks**: entradas `SessionStart` en `internal/components/sdd/inject.go` con el patrón `ensureBranchGuardHookEntries` (identidad por orden exacta, idempotente, escritura atómica) y retirada en `internal/components/uninstall/service.go`.
6. **Doctrina**: reescritura coordinada de la regla en todos los textos y goldens en un mismo commit, para evitar deriva.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/branch/gitcli/ops.go` | Modificado | Primitiva `FastForward`; `Fetch` con timeout |
| `internal/branch/engine/` (`engine.go`, `ensure.go`, `prefs.go`, nuevo `autofetch.go`) | Modificado/Nuevo | Ciclo de autofetch, fetch implícito en `ensure`, lectura de la preferencia |
| `internal/branch/ports.go`, `internal/branch/result.go` | Modificado | Puerto `Prefs` ampliado; resultado del autofetch |
| `internal/hub/types.go`, `internal/hub/prefs.go` | Modificado | Campo y acceso a la preferencia `autofetch` |
| `internal/cli/` (nuevo `autofetch.go`), `cmd/axiom/main.go` | Nuevo/Modificado | Verbo `axiom autofetch` |
| `internal/components/sdd/inject.go` | Modificado | Hook `SessionStart` en Claude Code y Codex |
| `internal/components/uninstall/service.go` | Modificado | Retirada del hook nuevo |
| `internal/components/agentguidance/routing.go` | Modificado | Regla de fetch |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md`, `internal/assets/skills/sdd-apply/SKILL.md` | Modificado | Regla de fetch y autofetch antes de cada parada |
| `testdata/golden/*.golden` (10) | Regenerado | Frase de la regla |
| `AGENTS.md`, `GEMINI.md`, `docs/manual-de-inicio.md` | Modificado | Política local y manual |
| `openspec/changes/autofetch-specs/specs/` | Nuevo | Deltas de `change-branch-lifecycle` y `multi-project-hub` (y `sdd-block-review-gates` si procede) |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| `merge --ff-only` en el checkout principal coincide con `scripts/axiom-worktree.ps1 done` u otra escritura (`index.lock`) | Media | Avance idempotente; ante bloqueo o error, degradar a aviso sin reintentar |
| Rama de integración no desplegada en ningún worktree o en uno distinto del principal | Media | Ejecutar en el worktree que la tiene; si no está desplegada, solo avisar |
| Fetch colgado por credenciales | Baja | `GIT_TERMINAL_PROMPT=0` ya fijado en `gitcommon`; timeout de ~8 s; no se toca el entorno SSH del usuario |
| Hook `SessionStart` rompe el arranque con un `axiom` antiguo o fallido | Media | Orden que termina siempre en 0; inerte fuera de proyectos Axiom y con el interruptor apagado |
| Deriva entre los textos de la regla y los 10 goldens | Media | Un único commit con todos los textos, goldens y pruebas |
| Fetch implícito que rechaza `ensure` con red caída | Media | Pendiente de decisión (ver «Preguntas abiertas») |
| Avisos obsoletos por no hacer `--prune` | Baja | Aceptado por el backlog |

## Preguntas abiertas (decisiones de producto)

1. ¿El fetch implícito con el interruptor encendido se aplica también a `ensure --archive` y a `done --fetch`? Recomendación: sí a `ensure --archive` (crea una rama), no a `done`.
2. Con el interruptor encendido, si el fetch implícito de `ensure` falla: ¿rechazo `fetch_failed` (como con `--fetch`) o aviso y continuar con la base local? Recomendación: aviso y continuar; el `--fetch` explícito conserva el rechazo.
3. ¿Se pregunta por el interruptor al registrar el proyecto (RF-HUB-08)? Recomendación: no; se activa solo con `axiom autofetch enable` (y, más adelante, desde la UI de C-06-e).

## Plan de reversión

- **Inmediata, sin código**: `axiom autofetch disable` devuelve cada proyecto al comportamiento actual (ninguna red sin `--fetch`).
- **Del cambio**: revertir el commit de fusión de la PR. El campo `autofetch` es `omitempty` y el esquema del hub sigue en `1.0`, así que un binario anterior lo ignora sin migración. Tras revertir, ejecutar `axiom sync` para retirar el hook `SessionStart`; si se instaló con una versión que ya no lo conoce, se borra a mano la entrada `axiom autofetch run` de `settings.json` (Claude Code) y `hooks.json` (Codex). Las deltas de spec se revierten con la PR; no hay datos que migrar.

## Dependencias

- C-05-a (integrado): `AGENTS.md:17` y `GEMINI.md:48` ya dicen «solo con autorización».
- C-01-c (integrado): RDD retirado; no se reutiliza su `SessionStart`.
- C-02-a (integrado): `main_branch` y `promotion`; la rama que se avanza es `integration_branch`.
- Bloquea a C-06-e, que mostrará y editará el interruptor en la UI.

## Criterios de éxito

- [ ] Con el interruptor apagado, ninguna orden hace red: las pruebas actuales de RF-BRA-10 (`ensure_test.go`, `ensure_fetch_test.go`, `done_test.go`, `archive_test.go`, `status_archive_test.go`, `ensure_adopt_test.go`) siguen pasando sin cambios.
- [ ] Con el interruptor encendido, `axiom autofetch run` hace fetch y avisa solo de los commits relevantes definidos.
- [ ] Con el árbol limpio y sin divergencia, la rama de integración local avanza por fast-forward; con árbol sucio o divergencia, no se mueve y el aviso incluye la orden exacta.
- [ ] Ninguna prueba observa movimiento de una rama de cambio, de `main_branch`, de una rama local de un repositorio de código, ni `push`/`prune`.
- [ ] Un remoto inalcanzable no bloquea el arranque de sesión: el hook termina en 0 y deja un aviso.
- [ ] `ensure` con el interruptor encendido devuelve `fetched: true` y crea las ramas sobre lo último de `origin`.
- [ ] La preferencia vive en el hub, apagada por defecto, y nunca se lee de `axiom.yaml`.
- [ ] La regla de doctrina menciona el interruptor en todos los textos y los 10 goldens están regenerados.
