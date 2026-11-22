# 02 · Entrega Git: rama previa configurable y una sola PR sin umbral de 400

> Análisis de la sesión 2 (2026-11-11), con decisiones cerradas con el usuario entre el 2026-11-11 y el 2026-11-12. Rutas verificadas contra `main` en `5f6b05b4`. Detalle en Engram: `analisis-backlog/entrega-git`, `/rdd`, `/v2`, `/verify-conjunto`, `/ci-size-check` y `/validacion-commits`.

## Origen

Temas del backlog original:

1. **Rama previa configurable por proyecto.** «Muchos equipos trabajan con una rama previa a main o master (por ejemplo develop). Se tiene que configurar por proyecto. Al terminar un desarrollo, primero se solicitará subir a esa rama sin borrar el branch; cuando esté validado se lanzará a main, master o la rama que tengan como principal. Y eso para cada rol.»
2. **Quitar el umbral de 400 líneas.** «Si quiero un control de "es mucho cambio, ¿quieres dividirlo?", vale, pero no definir lo de las 400. […] Lo que se divide son los commits: lo divido en 5 tareas, cada tarea hace un commit, aviso "ya está el commit, revisa", y así se validan los commits uno a uno, con parada por commit si lo desea, sin validar una PR enorme. La PR mejor unificada, salvo que el usuario pida otra cosa.»

Verificación de la premisa del tema 2. La afirmación de que «gentle-ai ha cambiado esa filosofía y por defecto será single PR» **no se confirma**:

- gentle-ai v4.4.0 (2026-11-07) y su `main` (2026-11-11) mantienen `ask-on-risk` con unas 400 líneas por defecto y un check de CI de 400.
- El commit `06ad2776f` (2026-10-14) refuerza las rebanadas de unas 400 líneas.
- Gentle Shell añade `single-pr` (#1600) solo como tercera opción, «least recommended».

El cambio se decide por preferencia propia del usuario, no por imitar a upstream.

## Cambios resultantes

### C-02-a · Promoción configurable por repositorio y por cambio (`branch` | `release` | `ask`)

- **Objetivo.** Hoy ya se puede poner `integration_branch: develop`. Las ramas salen de develop y la PR apunta a develop (`internal/branch/planner.go:392`, `internal/branch/engine/status.go:170`). Pero no existe el segundo salto a la rama principal, y `axiom branch done` borra la rama en cuanto está fusionada en la integración (`internal/branch/engine/done.go:441` `mergeEvidence`, `:394` `DeleteBranch`). Con develop configurado, la borraría al llegar a develop, justo lo contrario de lo pedido. El resultado esperado es que cada repositorio y cada cambio sigan uno de los dos flujos reales que usan los equipos, y que Axiom acompañe también la PR a la rama principal.
- **Decisiones.**
  - (Usuario, caso real kvp25.) «Por defecto main recibe cada feature pero hay casos que recibe develop completo». En kvp25, «los front siempre sacan rama de master, suben a develop sin borrar rama y luego suben a master la rama y la borran». En «los back, dependiendo si el cambio viene con migraciones de Entity Framework, sacan la rama de master o develop (si tiene cambios la sacan de develop); entonces, dependiendo de esto, mantienen la rama y la suben a master o muere en develop y se integra luego develop a master junto con las migraciones».
  - (Análisis, sobre esos hechos.) La promoción no se fija por proyecto: se resuelve **por repositorio y por cambio**. Modelo de `axiom.yaml`:
    ```yaml
    git:
      main_branch: master          # nuevo; si no se declara, una sola etapa (comportamiento actual)
      integration_branch: develop  # significado actual: rama de integración
      promotion: branch            # branch | release | ask (defecto del repositorio)
    repositories:
      - path: back
        promotion: ask             # se decide al crear la rama de cada cambio
    ```
  - (Análisis.) Modo `branch`: base = `main_branch`. Primera PR a la integración, conservando la rama. Segunda PR de la misma rama a `main_branch`, y entonces se borra.
  - (Análisis.) Modo `release`: base = `integration_branch`. PR a la integración y se borra la rama. El paso a la rama principal es la release develop→master, fuera del cambio. Es el comportamiento actual de `done`.
  - (Análisis.) Modo `ask`: decide la persona en `axiom branch ensure`, reutilizando su `decision_required` (`internal/branch/engine/prefs.go:123`, `internal/branch/result.go:45`), que hoy pregunta worktree o in place.
    - La base depende del modo, así que tiene que decidirse **antes** de crear la rama.
    - El modo resuelto se guarda por repositorio en la instantánea del kickoff (`internal/kickoff/gitsnapshot.go:24` `GitSnapshot`, `:44` `GitSnapshotRepo`) o en el registro local.
    - Cambiarlo después obligaría a un rebase, así que se rechaza.
  - (Análisis.) El criterio (migraciones de EF u otros) no se codifica en Axiom. Como mucho, el orquestador sugiere el modo a partir del plan; decide la persona.
  - (Usuario, a la pregunta de por qué Axiom no ayuda con la PR a master.) «Lo de que una vez ayudes con la PR a develop, luego ya axiom no ayuda con la PR a master, ¿por qué?». Respuesta del análisis: no es una decisión de diseño, el concepto no existe. En modo `branch`, tras la fusión en develop, `axiom branch status --json` presenta la **siguiente PR** (head = rama del cambio, base = `main_branch`). Axiom sigue sin abrir PRs ni hacer push ni merge.
  - (Análisis.) `done`: en modo `branch` borra la rama tras confirmar la fusión en `main_branch`; en `release`, tras confirmar la fusión en la integración. Reutiliza `ConfirmMerge` (`internal/branch/merge.go:28`) cambiando el destino.
  - (Análisis.) Multirepo: cada repositorio de código resuelve su modo. El front puede ir en `branch` y el back en `release` dentro del mismo cambio. El repositorio de specs queda **exento** y sigue trabajándose in situ en su integración (`planner.go:367-369`, `internal/branch/engine/ensure_repos.go:214-222`, `internal/branch/engine/publication.go:80-84`, `internal/branch/engine/handoff.go:104-108`, `done.go:189` lo omite).
  - (Análisis.) Roles: la cadena empieza en la rama del cambio. Las ramas de rol siguen apuntando a la rama de seguimiento (`planner.go:395-412`).
  - (Análisis.) Se descarta la lista de N etapas: ver *Descartado*.
- **Fuera de alcance.**
  - Abrir PRs, push o merge: siguen siendo de la persona.
  - Detectar automáticamente migraciones u otros criterios.
  - Migración retroactiva de cambios ya sellados: sin el campo, se tratan como una sola etapa.
  - La verificación conjunta (C-02-b).
- **Carril propuesto.** SDD: cambia la spec viva `openspec/specs/change-branch-lifecycle/` y el esquema de `axiom.yaml`.
- **Tamaño.** L (estimación: 1000-1400 líneas junto con C-02-b; C-02-a sola, unas 700-1000).
- **Depende de / bloquea a.**
  - No depende de ningún cambio.
  - Bloquea a C-02-b y a C-06-d.
  - Es dependencia inferida o blanda de C-05-c, C-07-d y C-07-f (README, apartado 3).
  - Se cruza con C-02-d en la estrategia de merge (ver *Riesgos* y P1).
  - C-06-e (ajustes del proyecto) tendrá que catalogar `main_branch` y `promotion`.
- **Ficheros y paquetes que toca.**
  - `internal/workspace/gitpolicy.go:20` (`DefaultIntegrationBranch`), `:34-37` (`GitConfig`), `:47-51` (`RepositoryPolicy`), `:55-56` (`gitKeys`, `repositoryKeys`, lista cerrada de claves). **[COMPARTIDO: esquema de axiom.yaml]**
  - `internal/branch/policy.go` (`ResolvePolicy`, `For`); `internal/branch/planner.go:392` (`PRTarget`), `:395-412`; `internal/branch/engine/status.go:170` (`baseCandidates`); `internal/branch/engine/done.go:441` (`mergeEvidence`), `:394`; `internal/branch/engine/prefs.go:123` (`decision_required`); `internal/branch/result.go` (estado de promoción en el resultado).
  - `internal/kickoff/gitsnapshot.go:24-60`.
  - `axiom.example.yaml` (bloque `git:`), `docs/manual-de-inicio.md` (ramas y worktrees; bloques `git:` y `repositories:`).
  - `openspec/specs/change-branch-lifecycle/`.
  - Tests: `internal/workspace/gitpolicy_test.go`, `internal/branch/*_test.go`, `internal/branch/engine/*_test.go` (en particular los de multirepo, done y publicación), `docs_contract_test.go` y `dogfooding_branch_status_test.go` si fijan claves o textos.
  - Dogfooding: `scripts/axiom-worktree.ps1:251-254` y `scripts/axiom-worktree.sh:223-224` fijan `main` a mano (`fetch origin main`, `merge --ff-only origin/main`). Ajustarlos a la rama de integración resuelta.
- **Criterio de terminado.**
  - Sin `main_branch` declarado, todos los tests de rama actuales pasan sin cambios.
  - Con `main_branch: master`, `integration_branch: develop` y `promotion: branch`: `ensure` crea la rama desde `origin/master`; `status --json` da `prTarget: develop`, y tras la fusión en develop da la siguiente PR con base `master`; `done` no borra la rama hasta que está fusionada en master.
  - Con `promotion: release`: la base es develop y `done` borra al fusionar en develop.
  - Con `promotion: ask`: `ensure` devuelve `decision_required` con las dos respuestas, y repetir con la respuesta crea la rama en la base correcta.
  - En multirepo, el repositorio de specs sigue in situ y `done` lo omite.
  - `go test ./...` en verde.
- **Riesgos y avisos.**
  - En modo `branch` la misma rama entra dos veces (develop y master). Si se fusiona con squash, la rama no queda como ascendiente de master. `done` dependería entonces de la capa de árbol equivalente (`merge-tree`, git ≥ 2.38), que puede dar falsos negativos, con `--force` como salida. Ver C-02-d.
  - Axiom no fusiona: la estrategia de merge la elige la persona o su plataforma. `done` debe funcionar **en los dos casos** (ascendencia con merge commit; árbol equivalente o `--force` con squash), decida lo que decida el usuario en P1. P1 solo cambia lo que recomienda la doctrina (C-02-d), no el diseño de este cambio.
  - El modo de promoción resuelto tiene que guardarse también para ODD, que no tiene kickoff. En ese carril solo cabe el registro local.
  - No cambiar la semántica de `IntegrationBranch`: `PublicationTarget` y `HandOff` comparan contra ese campo.
  - Hay una memoria del repo según la cual `axiom-worktree.ps1 done` falló tras una fusión squash. El propio repositorio usa squash en algunos merges: tenerlo en cuenta en las pruebas de dogfooding.

### C-02-b · Verificación conjunta del cambio antes de la PR a la rama principal

- **Objetivo.** Cada rol y cada repositorio se verifica por separado. La PR a master es el punto natural para verificar el cambio completo, es decir, lo que aportan todos los roles juntos.
- **Decisiones.**
  - (Usuario.) «Sería buen momento para indicarle que indique la rama del otro rol (u otros) para lanzar un verify en conjunto de todo el cambio y no solo lo de su rol».
  - (Análisis.) No hace falta que la persona indique las ramas de los otros roles. Axiom ya las conoce por el plan y por la instantánea del kickoff. Solo hay que traerlas de origin si se trabajaron en otra máquina, con un fetch autorizado explícitamente; nunca `--fetch` sin autorización.
  - (Análisis.) Pasos en SDD:
    1. Comprobar que cada rama de rol o de repositorio está fusionada en la integración. La evidencia `pr_merged` de la compuerta `integration` ya comprueba que un commit sea ascendiente de la base (`internal/kickoff/integration.go:44` `VerifyIntegrationEvidence`); la base por defecto la da `internal/cli/sdd_gate_base_ref.go:14-18` (`PlanningIntegration`, `internal/branch/engine/integration.go:19`). Hay que hacerlo por cada repositorio.
    2. Lanzar `sdd-verify` **sin rol**, sobre el cambio completo. La skill ya solo restringe al diff de un rol cuando se le pasa uno (`internal/assets/skills/sdd-verify/SKILL.md:41,62`).
    3. Usar el resultado como evidencia de la compuerta `integration` antes de presentar la PR a la rama principal.
  - (Análisis.) En modo `release`, la verificación conjunta se hace al fusionar en develop.
  - (Análisis.) En ODD no hay roles: sería un verify normal y opcional.
  - (Usuario, 2026-11-11, a la pregunta «¿el verify conjunto revisa solo lo que aporta el cambio o el estado de develop ya integrado?»): **«Solo el cambio»**. El alcance es la unión, por repositorio, del diff de la rama del cambio frente a su merge-base con la base del modo: `main_branch` en `branch` e `integration_branch` en `release`. En multirrol dentro de un mismo repositorio, la rama de seguimiento ya contiene las de rol. No se verifica el estado integrado de develop: las interferencias con otras features ya integradas quedan para el CI de develop, como tradeoff aceptado.
  - (Sección 01, decisión del usuario.) No usa RDD ni `axiom review`.
- **Fuera de alcance.**
  - Verificar develop integrado.
  - Abrir la PR a master.
  - Ejecutar el CI del cliente.
- **Carril propuesto.** SDD: cambia el contrato de la compuerta `integration` (`openspec/specs/sdd-block-review-gates/`) y de la skill `sdd-verify`.
- **Tamaño.** M.
- **Depende de / bloquea a.**
  - Depende de C-02-a, que aporta el modo y la base por repositorio.
  - Comparte la presentación de la puerta `integration` con C-06-c y C-06-d (evidencia informativa de la batería `pruebas.md`), y la prosa de los orquestadores con C-02-d.
- **Ficheros y paquetes que toca.**
  - `internal/kickoff/integration.go:44`; `internal/kickoff/args.go:351-385` (compuerta y tipos de evidencia).
  - `internal/cli/sdd_gate_base_ref.go:14-18`; `internal/branch/engine/integration.go:19`.
  - `internal/assets/skills/sdd-verify/SKILL.md` **[COMPARTIDO]**.
  - Descripción de la compuerta `integration` en `internal/assets/skills/_shared/sdd-orchestrator-sections.md` y en los 12 `internal/assets/<agente>/sdd-orchestrator.md` **[COMPARTIDO]**.
  - Goldens en `testdata/golden/` **[COMPARTIDO]**.
  - `openspec/specs/sdd-block-review-gates/`.
- **Criterio de terminado.**
  - En un cambio multirrol con dos repositorios, la compuerta `integration` no se aprueba hasta que cada rama tenga `pr_merged` verificado en su integración y exista un informe de `sdd-verify` sin rol.
  - El diff verificado es exactamente la unión de los diffs de las ramas del cambio frente a su merge-base con la base del modo. Un test lo fija con un commit ajeno en develop que no entra.
- **Riesgos y avisos.**
  - Con hand-off entre máquinas, las ramas de otros roles solo están en origin: sin fetch autorizado, el verify conjunto no puede hacerse y debe decirlo, sin degradar en silencio.

### C-02-c · Una sola PR por defecto y retirada del umbral de 400 líneas

- **Objetivo.** Retirar el número 400 de la doctrina, del pre-vuelo y de las skills. La entrega por defecto pasa a ser una sola PR, y la división se hace en commits por tarea (C-02-d), no en PRs.
- **Decisiones.**
  - (Usuario.) «La PR mejor unificada, salvo que el usuario pida otra cosa».
  - (Usuario.) «Lo que comentaba de revisiones por commit es justamente lo que es para mí las rebanadas, más que un conteo de líneas de 400 […] cada task que has creado genera un commit […] es como revisar 5 PRs de 400 líneas pero revisas 5 commits de 400 líneas, es lo mismo, y en realidad no de 400 líneas sino una tarea completa a revisar tenga 200 o 600 líneas, es una tarea indivisible».
  - (Usuario.) «Si quiero un control de "es mucho cambio, ¿quieres dividirlo?", vale, pero no definir lo de las 400».
  - (Análisis.) Vocabulario: `single-pr` (por defecto) y `chained`, solo a petición explícita, que conserva `stacked-to-main` y `feature-branch-chain`. `ask-on-risk`, `auto-chain` y `exception-ok` dejan de ofrecerse, pero se aceptan como alias al leer cambios ya sellados.
  - (Análisis, consecuencia directa de la decisión del usuario «La PR mejor unificada, salvo que el usuario pida otra cosa».) **La pregunta de estrategia de PR sale del pre-vuelo.** La entrega es `single-pr` sin preguntar, y `chained` solo cuando la persona lo pide en el chat. El pre-vuelo no ofrece un menú cuya respuesta ya está decidida. Si el pre-vuelo queda sin preguntas en multirepo, cómo lo trata el hook es diseño de C-01-b (sección 01), no de este cambio. Resuelve P10 en lo que toca a la sección 02.
  - (Análisis.) Sin número en la doctrina. La señal de «cambio grande» es cualitativa: la lista de tareas que la persona ya ve. Aviso opcional: `delivery.split_hint_lines` en `axiom.yaml`, desactivado por defecto; si se fija, avisa una vez y nunca bloquea. *Pendiente de confirmar en la propuesta SDD: el usuario aceptó «un control», pero no se ha cerrado si va como clave de `axiom.yaml`.*
  - (Usuario, vía sección 01.) RDD queda apagado de fábrica y sin posibilidad de activarlo, porque se va a retirar. La revisión por commit no se apoya en RDD ni en `axiom review assess`.
  - (Análisis.) Las dos referencias de RDD al 400 se van con RDD (sección 01) y **este cambio no las toca**, para no editar dos veces los mismos pasajes:
    - `internal/components/agentguidance/routing.go:129`, con la rebanada de PR de RDD medium acotada a unas 400 líneas. **Aviso de la revisión cruzada:** C-01-c solo lista `routing.go:131` y `:158`, así que `:129` hoy no tiene dueño. Ver `revision-02.md`. Si al llegar a C-02-c sigue en `main`, C-02-c lo reduce a su primera frase, la que no es de RDD: «Run applicable functional checks per task, not a review cycle per TODO checkbox».
    - `internal/reviewtransaction/risk.go:22` `LargeChangeLines`, con la composición 4R. Lo retira C-01-e, que va al final del plan (ola 16).
  - (Usuario.) Se elimina el check de tamaño del CI del propio repo: ver C-02-e.
- **Fuera de alcance.**
  - `risk.go:22` (C-01-e) y la parte de RDD de `routing.go:129`, que debería retirar C-01-c (ver *Decisiones* y `revision-02.md`).
  - Retirar la capacidad de encadenar PRs: se mantiene a petición.
  - El aviso y la parada por commit (C-02-d).
- **Carril propuesto.** SDD: cambia la spec viva `openspec/specs/sdd-orchestrator-assets/`, que exige la estrategia y la cadena en los 12 orquestadores, y hay unos 75-80 ficheros afectados.
- **Tamaño.** L (unos 40 ficheros a mano, unos 15 goldens regenerados y unos 18 tests).
- **Depende de / bloquea a.**
  - El grupo «PR strategy» del pre-vuelo vive en los mismos ficheros que C-01-a (hook por etiqueta) y C-01-b (pre-vuelo por topología): `internal/components/sdd/session_preflight.go` e `internal/cli/sdd_preflight_hook.go`. **Va después de los dos, nunca en paralelo.**
  - Conviene que vaya después de C-01-c (RDD apagado), para no reescribir doctrina que C-01-c retira.
  - Bloquea a C-02-d. C-02-e es más laxa: se implanta con C-02-c o justo después.
  - C-06-b (HTML explicativo) añade una pregunta al mismo pre-vuelo y va después (inferida en el README).
- **Ficheros y paquetes que toca.**
  - **Doctrina ODD** (fuente única, se genera por agente): `internal/components/agentguidance/routing.go:122` (unas 400 líneas por tarea como heurística) y `:130` («Delivery follows work units… ask-on-risk… about 400»). **[COMPARTIDO]**
  - **Pre-vuelo SDD:**
    - `internal/components/sdd/session_preflight.go:16` («PR strategy: Ask me, Single PR, or Auto»);
    - `internal/cli/sdd_preflight_hook.go:401` (mapa por índice), `:408` («Review policy: 400 changed lines»), `:445-450` (validación);
    - `internal/assets/opencode/plugins/sdd-task-result-artifacts.ts` y `internal/assets/opencode/plugins-v2/sdd-task-result-artifacts.ts`.
  - **Composición:** `internal/components/sdd/orchestrator.go:151` (fallback de Delivery Strategy y Chain Strategy); `internal/components/sdd/inject.go:1518,1535` (fragmentos heredados «D1 400 lines» y «D1 400 líneas»).
  - **Assets del orquestador [COMPARTIDO]:**
    - `internal/assets/skills/_shared/sdd-orchestrator-sections.md:315-317` (Delivery Strategy), `:319-328` (Chain Strategy) y `:459-474` (Review Workload Guard), además de la mención de `:300`;
    - `internal/assets/skills/_shared/sdd-phase-common.md:109` (E. Review Workload Guard);
    - `internal/assets/claude/sdd-orchestrator-workflow.md:103,114,143`;
    - los 12 `internal/assets/<agente>/sdd-orchestrator.md`.
  - **Skills [COMPARTIDO]:**
    - `internal/assets/skills/chained-pr/SKILL.md` y `references/chaining-details.md`;
    - `work-unit-commits` en **dos copias que divergen**: `internal/assets/skills/work-unit-commits/SKILL.md` frente a `skills/work-unit-commits/SKILL.md`. Reconciliarlas antes de editar;
    - `internal/assets/skills/sdd-tasks/SKILL.md:86,149` (Review Workload Forecast);
    - `internal/assets/skills/sdd-apply/SKILL.md:333` (STOP `workload-decision-required`);
    - `sdd-archive`, `skills/branch-pr/SKILL.md`, `rdd-defect-workflow` (si sobrevive a la sección 01), `issue-creation` y `comment-writer`.
  - **Spec y docs:** `openspec/specs/sdd-orchestrator-assets/spec.md`; `docs/usage.md`, `docs/components.md`, `internal/app/help.go` y `AGENTS.md` (descripción de `chained-pr`).
  - **Tests:**
    - `internal/components/sdd/delivery_strategy_vocabulary_test.go`, que hay que reescribir;
    - `inject_test.go`, `renderer_invariants_test.go`, `session_preflight_test.go`, `orchestrator_composition_test.go` y `planning_detail_test.go`;
    - `internal/assets/assets_test.go` y los tests de skills afectadas;
    - `internal/cli/sdd_preflight_hook_test.go` e `internal/sddtaskresult/preflight_question_harness_test.go`;
    - goldens en `testdata/golden/` **[COMPARTIDO]**.
- **Criterio de terminado.**
  - `rg -n "400" internal/assets internal/components internal/cli skills` no devuelve ninguna referencia al umbral de entrega. Como residuo de RDD solo puede quedar `internal/reviewtransaction/risk.go:22`, mientras C-01-e (misma ola) no se haya fusionado.
  - El pre-vuelo ya no pregunta la estrategia de PR. La doctrina dice `single-pr` por defecto y `chained` solo a petición de la persona.
  - `sdd-apply` ya no se para por carga de revisión.
  - Un cambio sellado con `ask-on-risk` sigue leyéndose.
  - Goldens regenerados y `go test ./...` en verde.
- **Riesgos y avisos.**
  - En `main` el hook del pre-vuelo exige exactamente 3 preguntas (`internal/cli/sdd_preflight_hook.go:363`) y mapea los tokens por posición (`:401`). C-01-a y C-01-b lo cambian a etiquetas y a un número variable de preguntas. Este cambio parte de ese hook ya reescrito y solo retira el grupo de estrategia.
  - Las copias instaladas de las skills en `.claude/`, `.agents/`, `.kiro/` y `.windsurf/` se regeneran con `axiom sync`.

### C-02-d · Aviso por commit, parada opcional por tarea y validación implícita al pedir la PR

- **Objetivo.** Que la persona revise commit a commit mientras el modelo avanza, sin revisar una PR enorme al final y sin ceremonia de aprobación.
- **Decisiones.**
  - (Usuario.) Cada tarea genera un commit, y el usuario, «según está ese commit, podría ir validándolo mientras el modelo sigue con la siguiente task».
  - (Usuario, 2026-11-12, literal.) «Lo de parar en cada tarea que lo decida el usuario; en el chat se avisará cada vez que se termine una tarea con el commit, pero si no quiere paradas el modelo continuará; si al final solicita el PR a develop se da por validados todos los commits».
  - (Análisis, derivado de lo anterior.) Dos modos: `notify` (por defecto) y `stop`.
    - En los dos, al terminar cada tarea, el orquestador avisa en el chat con el sha y la tarea, y la orden `git show <sha>`.
    - En `notify` sigue con la siguiente tarea; en `stop` espera.
  - (Usuario.) Pedir la PR a develop equivale a validar todos los commits pendientes. No hay aprobación commit a commit ni nada bloquea la PR.
  - (Usuario.) Solo se registran los commits en los que se han pedido cambios explícitamente.
  - (Análisis.) Cambios pedidos sobre un commit ya superado:
    - si la rama aún no se ha publicado, commit `fixup!` sobre ese sha y autosquash antes de abrir la PR;
    - si ya se publicó, un commit correctivo nuevo.
    Nunca se reescribe historia publicada.
  - (Análisis, **no confirmado por el usuario**.) Fusionar sin squash (merge commit), para que los commits revisados lleguen a la rama destino y para que `done` confirme por ascendencia en el modo `branch` de C-02-a.
  - (Usuario, 2026-11-17; P2 resuelta, **sustituye a la viñeta siguiente**.) El modo se pregunta en la parada previa a las tareas (puerta `tasks` en SDD `checkpointed`), no en el pre-vuelo ni en `kickoff seal`. Se pregunta si parar al terminar cada tarea y si el commit es automático. Tres modos: (1) avisar y seguir con commit automático, por defecto; (2) parar tras cada tarea con commit automático; (3) parar tras cada tarea sin commit: la persona revisa el diff, aprueba y entonces se hace el commit. «Sin commit» implica parar. En `continuous` no se pregunta: modo 1.
  - (Usuario, 2026-11-20; corrige P2.) ODD va siempre en modo 1: sin pregunta ni paradas, tampoco a petición en el chat. Quien quiera parar tras cada tarea usa SDD `checkpointed`. La «parada de plan en ODD» que citaba P2 ya no existía (la retiró REQ-27.17 el 2026-11-10); REQ-27.17 no cambia y el modo 3 solo es comportamiento nuevo en `sdd-apply`.
  - (Usuario, 2026-11-17; P1 resuelta.) La doctrina recomienda fusionar sin squash (merge commit).
  - (Análisis, **superada por P2**.) Sin pregunta nueva en el pre-vuelo: el modo es `notify` por defecto y la persona pide paradas en cualquier momento. El motivo ya no es que el hook exija exactamente 3 preguntas (C-01-b lo hace variable), sino no ampliar el pre-vuelo con una decisión que el usuario quiere tomar sobre la marcha («que lo decida el usuario»). En SDD se refleja en el kickoff (bandera nueva en `axiom sdd kickoff seal`, ortogonal a `--execution-style`); en ODD, en el documento de feature.
  - (Sección 01.) Sin RDD.
- **Fuera de alcance.**
  - Revisión automática de commits.
  - Bloquear la PR por commits pendientes.
  - La interfaz de usuario para validar commits (en su caso, sección 06).
- **Carril propuesto.** SDD: registra el modo en la decisión de la puerta `tasks` (`gates.yaml`; el kickoff no admite cambios tras el sellado, REQ-25.36) y cambia la doctrina del orquestador. ODD no gana paradas: sigue sin parada propia (`internal/components/agentguidance/routing.go:66-67`, REQ-27.17) y va siempre en modo 1.
- **Tamaño.** M.
- **Depende de / bloquea a.**
  - Depende de C-02-c (comparte doctrina y skills).
  - Bloquea a C-06-f (puertas desde la UI).
  - Se cruza con C-02-a por la estrategia de merge (P1).
  - Comparte prosa de los orquestadores con C-02-b.
- **Ficheros y paquetes que toca.**
  - `internal/components/agentguidance/routing.go:76` (paso 6, «Implement task by task») y `:77` (paso 7, «Close»). **[COMPARTIDO]**
  - `internal/assets/skills/_shared/sdd-orchestrator-sections.md` y los 12 `internal/assets/<agente>/sdd-orchestrator.md` **[COMPARTIDO]**.
  - `internal/assets/skills/sdd-apply/SKILL.md` y `internal/assets/skills/work-unit-commits/SKILL.md` **[COMPARTIDO]**.
  - `internal/kickoff/types.go` (`GateRecord`), `internal/cli/sdd_gate*.go` (bandera de `gate record`) e `internal/sddstatus/status_v2.go` (proyección); ni `internal/kickoff/args.go` ni `sdd_kickoff.go` cambian.
  - `openspec/specs/sdd-block-review-gates/` y `openspec/specs/sdd-orchestrator-assets/`.
  - Goldens **[COMPARTIDO]**.
- **Criterio de terminado.**
  - La doctrina generada indica avisar con el sha tras cada commit de tarea, continuar en `notify`, esperar en `stop`, y dar por validados todos los commits al pedir la PR.
  - `gate record --gate tasks` acepta y persiste el modo; sin registro, modo 1 (avisar y seguir).
  - Tests de la doctrina y del kickoff en verde, y goldens regenerados.
- **Riesgos y avisos.**
  - Un mensaje del usuario con cambios sobre una tarea anterior llega mientras el modelo trabaja en la siguiente. La doctrina debe decir cómo intercalar el `fixup!` sin perder la tarea en curso.

### C-02-e · Eliminar el check de tamaño de PR del CI del repositorio Axiom

- **Objetivo.** El job de 400 líneas del CI propio choca con el modelo de una sola PR.
- **Decisiones.**
  - (Usuario, 2026-11-11, a la pregunta «¿qué hacemos con el check de 400 líneas del CI del repo Axiom?»): **«Eliminarlo»**.
  - (Análisis.) La etiqueta `size:exception` deja de usarse.
  - (Análisis.) Al implantarlo hay que actualizar la memoria local «Una PR con size:exception» (`entrega-una-pr-size-exception.md`) y la preferencia de Engram `preference/delivery-strategy`, que quedan obsoletas en la parte de `size:exception`. La preferencia de una PR y un commit por tarea sigue vigente.
- **Fuera de alcance.** El resto de jobs de `pr-check.yml`, como `check-type-label` desde `:54`.
- **Carril propuesto.** ODD: es política del repositorio, no del producto, y son 2 ficheros.
- **Tamaño.** S.
- **Depende de / bloquea a.** Se implanta con C-02-c o justo después. Si fuera antes, las PRs pequeñas seguirían pasando igualmente, así que no bloquea nada.
- **Ficheros y paquetes que toca.**
  - `.github/workflows/pr-check.yml:17-53` (job `check-pr-size`, `const limit = 400` en `:25`). **[COMPARTIDO: .github/workflows]**
  - `.github/PULL_REQUEST_TEMPLATE.md:85,95`.
- **Criterio de terminado.**
  - `rg -n "check-pr-size|size:exception|400" .github/` no devuelve nada del umbral.
  - Una PR de más de 400 líneas sin etiqueta pasa el CI.
- **Riesgos y avisos.** Ninguno técnico.

## Descartado

- **`git.promotes_to` escalar por proyecto** (primera propuesta del análisis). Descartado por los hechos que aportó el usuario: la base de la rama depende del modo y el modo cambia por repositorio y por cambio (kvp25). Lo sustituye `main_branch` más `promotion: branch|release|ask` (C-02-a).
- **Lista de N etapas** (`promotion: [develop, staging, main]`). Ningún caso real la pide. Exigiría decodificar listas, porque las claves de `git:` hoy son solo cadenas (`gitpolicy.go:55-56`), y estados intermedios.
- **Solo documentar `integration_branch: develop`.** No resuelve el borrado prematuro de `done` ni la segunda PR.
- **Verificar el estado integrado de develop** en el verify conjunto: el usuario eligió «Solo el cambio».
- **Mantener el 400 como aviso no bloqueante en la doctrina.** El usuario no quiere el número.
- **Cambiar solo el defecto a `single-pr`.** Dejaría activos el Review Workload Guard y el STOP de `sdd-apply`, contradiciendo el modelo.
- **Apoyar la revisión por commit en RDD o en `axiom review assess`.** RDD queda apagado y se retira (sección 01).
- **Aprobación explícita commit a commit y PR bloqueada hasta validar todos** (primera versión del modelo). El usuario decidió que pedir la PR valida todos los commits.
- **Dejar el check de CI como aviso.** El usuario eligió eliminarlo.
- **Seguir a gentle-ai como argumento.** La premisa no se confirmó.

## Correcciones a la consolidación

Engram `analisis-backlog/consolidacion` (revisión del 2026-11-13), bloque «Entrega Git»:

1. Dice «Rama previa con `git.promotes_to` y done en dos fases». **Obsoleto.** El modelo cerrado es `git.main_branch` más `promotion: branch | release | ask`, por repositorio y con decisión por cambio en `axiom branch ensure` (C-02-a). En `release`, `done` borra al fusionar en develop, como hoy; en `branch`, al fusionar en `main_branch`.
2. Falta la **verificación conjunta del cambio antes de la PR a la rama principal**: alcance «solo el cambio», decidido por el usuario (C-02-b).
3. Falta la **eliminación del check de 400 líneas del CI del repo** y de las casillas de la plantilla de PR, decidida por el usuario (C-02-e).
4. Falta el aviso de que el cambio del pre-vuelo de C-02-c comparte ficheros con el cambio A de la sección 01 (pre-vuelo por etiqueta) y debe ir con él o justo después. El orden propuesto en la consolidación (entrega Git en el paso 3) es compatible si C-02-c se apoya en el pre-vuelo ya reescrito.
5. Lo demás («una sola PR por defecto, sin el umbral de 400; tras cada tarea se avisa con el commit en el chat; la parada por tarea es opcional; pedir la PR a develop valida todos los commits; solo se registran los commits con cambios pedidos; no depende de RDD») coincide con lo decidido.
