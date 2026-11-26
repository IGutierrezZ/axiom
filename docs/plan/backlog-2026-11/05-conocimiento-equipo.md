# 05 · Conocimiento compartido entre personas

Rutas verificadas contra `main` en `5f6b05b4`. Detalle del análisis en Engram, con los topic_key `analisis-backlog/conocimiento-equipo`, `analisis-backlog/conocimiento-equipo/publicacion-paradas` y `analisis-backlog/conocimiento-equipo/autofetch`.

## Origen

Temas del backlog original:

1. **Engram compartido.** Texto del backlog: «Además de la documentación de OpenSpec u ODD, tener compartida la memoria de Engram con el resto de roles del equipo; seguro que mucha información les vendría bien: cómo se ha quedado algo, alguna decisión... Creo que Engram ya dispone de alguna utilidad para compartir memoria por git.»
2. **Autofetch del repo de specs.** Texto del backlog: «Hacer fetch del repo de specs habitualmente para ver que no hay memoria nueva de Engram o modificaciones en OpenSpec de la tarea en la que estás trabajando, etc.»

Al rechazar el tema 1, el usuario abrió un objetivo nuevo: que la documentación quede siempre subida en cada parada aprobada. Ese objetivo da lugar a C-05-b y C-05-c.

## Cambios resultantes

### C-05-a · Limpieza de restos del «Engram compartido»

- **Objetivo.** El repo versiona memoria de Engram heredada de upstream, y varias guías invitan a compartirla o a hacer fetch sin autorización. Ambas cosas contradicen lo decidido. Resultado esperado: no queda memoria de terceros versionada, no hay documentación del flujo de equipo con `engram sync` y ninguna guía del repo ordena un fetch sin autorización.
- **Decisiones:**
  - (usuario) «Sí, lote 1», respondiendo a «¿Meto esta limpieza en el lote 1 de la consolidación?». La limpieza va en el lote 1 y no espera a C-05-d.
  - (análisis, presentado al usuario sin objeción) Borrar `.engram/chunks/` (5 ficheros `.jsonl.gz`) y `.engram/manifest.json`, y añadirlos a `.gitignore` para que un `engram sync` no los vuelva a meter.
  - (análisis) Conservar `.engram/config.json` (`{"project_name": "axiom", "project": "axiom"}`). Engram lo usa para detectar el proyecto: `mem_current_project` devuelve `project_source: config`.
  - (análisis) No reescribir el historial. Los prompts siguen en los 7 commits que tocan `.engram/`. Reescribir `main` exigiría un push forzado y rompería clones y PR abiertas. Solo se reescribiría si el contenido resultara sensible, y no se ha abierto.
  - (análisis) Quitar el flujo de equipo de `docs/engram.md` en el mismo commit, porque responde a la misma decisión.
  - (análisis) `AGENTS.md:17` pasa a «fetch solo con autorización de la persona». C-05-d lo reescribirá otra vez para nombrar su interruptor.
  - (análisis, hallazgo nuevo al verificar en `5f6b05b4`, no tratado con el usuario) `GEMINI.md:48` repite la misma orden de fetch pasivo y se alinea igual. `docs/intended-usage.md` y `docs/codebase/sync-and-cloud.md` también documentan el reparto por `engram sync` y se alinean en el mismo commit.
- **Fuera de alcance:**
  - Reescribir el historial de git.
  - Cambiar `internal/assets/skills/_shared/persistence-contract.md`. Su línea `:13` («Local only, not shareable») pasa a ser correcta. Su defecto (`:9`, «if Engram is available → `engram`. Otherwise → `none`») queda desfasado con el hybrid fijo de C-01-b, pero C-01-b no lista ese fichero: está anotado en `revision-05.md` para que lo recoja C-01-b. Este cambio no lo toca.
  - Tocar `governance.shared_memory` (`axiom.yaml:31`, `internal/workspace/types.go:46`).
- **Carril propuesto: ODD.** Es una limpieza pequeña y entendida, sin contrato de producto nuevo ni delta de spec.
- **Tamaño: S.**
- **Depende de / bloquea a:**
  - No depende de nada.
  - Precede a C-05-d, que reescribe `AGENTS.md:17` y `GEMINI.md:48`.
- **Ficheros:**
  - `.engram/chunks/1ad74530.jsonl.gz`, `aaf7a13f`, `be4ddc87`, `d21f9e56` y `fd2ecf8b` (borrar).
  - `.engram/manifest.json` (borrar).
  - `.engram/config.json` (conservar).
  - `.gitignore:53-55`: hoy el comentario dice «project sync chunks are intentionally versioned» e ignora solo `engram.db` y `cloud.json`. Hay que cambiar el comentario y añadir `chunks/` y `manifest.json`.
  - `docs/engram.md:50-62` (sección «Team Sharing»; revisar también la línea 25).
  - `docs/intended-usage.md:30-31,172`.
  - `docs/codebase/sync-and-cloud.md:12,39,52`.
  - `AGENTS.md:17` y `GEMINI.md:48`.
- **Criterio de terminado:**
  - `git ls-files .engram` devuelve solo `.engram/config.json`.
  - `rg -n "engram sync" docs/` no presenta `engram sync` como flujo de equipo.
  - `rg -n "pasivamente .git fetch" AGENTS.md GEMINI.md` no devuelve nada.
  - `mem_current_project` desde el repo sigue devolviendo `project: axiom` con `project_source: config`.
  - `go test ./internal/workspace/...` pasa. Es por si algún test de contrato de docs (`docs_contract_test.go`) cita estas frases.
- **Riesgos y avisos:**
  - Si alguien con el plugin de Claude de Engram tiene importados esos chunks, su base local los conserva. No pasa nada.
  - El hook SessionStart del plugin de Engram solo importa si existe `.engram/manifest.json`, así que deja de importar al borrarlo. Es lo deseado.

### C-05-b · Publicación en `continuous` con `docs_publication: auto` también tras spec y design

- **Objetivo.** En SDD sin paradas, con la publicación autorizada, los documentos solo se suben al cerrar `tasks` y al archivar. Mientras tanto, spec y design no llegan al equipo. Resultado esperado: con `docs_publication: auto`, se publica también al terminar `spec` y al terminar `design`.
- **Decisiones:**
  - (usuario) «no me convence lo que me cuentas del engram, como axiom es siempre both y tenemos las informacion en openspec, tendra que ser suficiente, tendremos que ayudar mucho en axiom a que la documentacion este siempre subida a cada cambio que se confirme.»
  - (usuario) «cada cambio que se confirme» significa: «Cada parada aprobada».
  - (usuario) «En SDD sin paradas con docs_publication: auto, si no cuesta mucho, meteria subir tambien tras spec y design.»
  - (análisis) La autorización sellada sigue siendo la misma: publicar solo documentos (RF-KOF-10). Los kickoffs ya sellados con `auto` heredan los momentos nuevos sin volver a preguntar.
  - (análisis) Sigue sin haber paradas en `continuous`, y una negativa de `publish` sigue siendo informativa (RF-PUB-08).
- **Fuera de alcance:**
  - `checkpointed`, que ya publica en cada aprobación (RF-GAT-15, `openspec/specs/sdd-block-review-gates/spec.md:234-235`).
  - ODD. Decisión del usuario: «ODD no pasa nada, esta enfocado a monorepos. no hace falta subirlos automaticamente.»
  - Los kickoffs `continuous` sin `docs_publication`.
- **Carril propuesto: ODD con sección `## Spec viva`.** Es un cambio acotado y entendido. El delta toca RF-PUB-08 y RF-KOF-10, y ODD puede integrarlo con `axiom archive integrate --odd`. Si la sesión padre prefiere tramitar por SDD todo cambio de contrato de publicación, también vale.
- **Tamaño: S.**
- **Depende de / bloquea a:**
  - No depende funcionalmente de nada. Su valor presupone la persistencia siempre hybrid de la sección 01: con `engram` o `none` no se publica nada.
  - Comparte `sdd-orchestrator-sections.md:88` con C-05-c y con el texto del pre-vuelo que cambia la sección 01: conviene ordenarlos o fusionarlos.
- **Ficheros:**
  - `openspec/specs/sdd-docs-publication/spec.md:286-288` (RF-PUB-08, «exactamente en dos momentos»; escenarios a partir de `:290`).
  - `openspec/specs/sdd-lane-kickoff/spec.md:358,364` (RF-KOF-10, texto de la opción del pre-vuelo «al cerrar `tasks` y al archivar»).
  - [COMPARTIDO] `internal/assets/skills/_shared/sdd-orchestrator-sections.md:88` («When to publish…»).
  - [COMPARTIDO] El texto del pre-vuelo en `internal/components/sdd/session_preflight.go`, que también cambia la sección 01.
  - [COMPARTIDO] Los goldens de `testdata/golden/` que incluyen esa sección.
  - `internal/kickoff/types.go:119-130` (`DocsPublicationAuto`). El comentario de la línea 129 dice «al cerrar `tasks` y tras archivar» y hay que actualizarlo.
- **Criterio de terminado:**
  - RF-PUB-08 nombra cuatro momentos: spec, design, tasks y archivo.
  - La sección del orquestador ordena `axiom sdd publish --cwd <root> --change <name> --json` al terminar `spec` y `design` con `docsPublication: auto`.
  - La opción del pre-vuelo dice «sube los documentos a `origin/<rama>` al cerrar spec, design y tasks y al archivar».
  - Goldens regenerados y `go test ./internal/assets/... ./internal/components/... ./internal/kickoff/...` en verde.
- **Riesgos y avisos:**
  - Hay más pushes por cambio. La doctrina ya fija que es sin forzar y solo de rutas de documentación.
  - Tras C-01-b, un multirepo sella siempre `checkpointed`, así que `docs_publication: auto` solo se da en monorepo. El cambio sigue haciendo falta, pero sus pruebas se plantean sobre monorepo.
  - Hay que confirmar que `PublicationTarget` resuelve la rama del cambio antes de `tasks`, es decir, que `axiom branch ensure` ya la preparó en spec. Verificarlo con un test.

### C-05-c · Estado «aprobada sin publicar» en `axiom sdd status`

- **Objetivo.** Hoy `axiom sdd gate record` registra la aprobación y devuelve la orden de `axiom sdd publish`, pero no la ejecuta (RF-GAT-11, `internal/cli/sdd_gate_handoff.go:165-190`). La ejecuta el orquestador por instrucción. Si se le olvida, se corta la sesión, se compacta el contexto o falla el push, la aprobación queda registrada pero los documentos no llegan a `origin`, y nadie lo detecta. Resultado esperado: el status lo detecta y lo proyecta con la orden exacta, y el orquestador lo resuelve antes de la fase siguiente o al reanudar.
- **Decisiones:**
  - (usuario) «3, ok creo que lo entiendo y me parece bien lo del estado, si puedes explicarmelo mejor, pero seguramente hagamos lo que dices.» Se le dio la explicación ampliada, sin nuevas objeciones. Queda aprobado salvo cambio en el diseño.
  - (análisis) Condición de `publication: pending`. Primero, existe una aprobación registrada de una compuerta distinta de `integration`, o un momento `auto` cumplido en `continuous` (C-05-b). Además, se cumple al menos una de estas dos cosas:
    - las rutas de documentación del cambio (la lista cerrada de RF-PUB-01 en `sdd-docs-publication`) tienen cambios sin confirmar;
    - la rama del cambio va por delante de `origin/<rama>`.
  - (análisis) Se mide sin red, con las referencias locales. El ahead/behind ya existe en `internal/branch/engine/status.go:301` (`AheadBehind`). Tras un push propio la referencia de seguimiento está al día, así que el cálculo es fiable sin fetch.
  - (análisis) La proyección incluye la invocación exacta, la misma que `publication.invocation` de la pregunta de compuerta.
  - (análisis) La doctrina del orquestador ordena: si el status proyecta `publication: pending`, ejecutar esa invocación una vez antes de lanzar la fase siguiente, y también al reanudar. Una negativa sigue siendo informativa.
- **Fuera de alcance:**
  - Que `gate record` ejecute git, porque RF-GAT-11 se mantiene.
  - ODD.
  - Hooks que bloqueen el cierre de turno, porque el hook Stop de RDD se retira (sección 01).
- **Carril propuesto: SDD.** Es un campo nuevo del contrato JSON de `axiom sdd status`, toca varios paquetes (sddstatus, branch engine y doctrina) y necesita requisito y escenarios nuevos.
- **Tamaño: M** (~250-400 líneas con tests).
- **Depende de / bloquea a:**
  - Conviene después de C-05-b, para incluir los momentos `auto` nuevos.
  - Toca la misma doctrina que C-05-b.
  - Debe convivir con la sección 02, en concreto con C-02-a: `git.main_branch` + `promotion: branch|release|ask` y relevos. Comprobar que el destino de publicación de `PublicationTarget` no cambia de semántica. Los documentos se publican en la rama del cambio, no en `main_branch`.
- **Ficheros:**
  - `internal/sddstatus/status_v2.go:134-136` (proyección de `governance.kickoff`; el campo nuevo va junto a la proyección de compuertas).
  - `internal/cli/sdd_gate_handoff.go:146-190` (`gatePublishJSON`, `resolveGatePublish`; reutilizar el cálculo del destino).
  - `internal/branch/engine/status.go:301` (ahead/behind).
  - `internal/sddpublish/` (la lista de rutas de documentación; solo lectura).
  - [COMPARTIDO] `internal/assets/skills/_shared/sdd-orchestrator-sections.md:82,88` y los goldens de `testdata/golden/`.
  - `openspec/specs/sdd-docs-publication/spec.md` (requisito nuevo).
- **Criterio de terminado:**
  - Con una compuerta aprobada y documentos sin confirmar, el status proyecta `publication: pending` con la invocación.
  - Tras `axiom sdd publish`, deja de proyectarlo.
  - Con la rama por delante de `origin/<rama>`, lo proyecta.
  - Con `engram` o `none`, o sin aprobaciones, no lo proyecta.
  - Tests del status y goldens en verde.
- **Riesgos y avisos:**
  - Si la referencia remota de seguimiento no existe (primera publicación), se debe tratar como pendiente, no como error.
  - En multirepo, la rama de publicación del repo de specs es la de integración, in situ (`handoff.go`, `publication.go`). El cálculo del ahead se hace contra esa rama.
  - El vocabulario no puede chocar con el de RDD (`internal/kickoff/rdd_boundary_test.go` prohíbe términos RDD en kickoff).

### C-05-d · Autofetch del repo de specs: avisar y avanzar

- **Objetivo.** Que cada persona vea, sin pedirlo, lo que otros han subido a OpenSpec o a la rama de su tarea, y tenga al día la rama de integración local. Todo ello sin violar la regla vigente: «Never pass `--fetch` without the explicit authorization of the user».
- **Decisiones:**
  - (usuario) Ante «¿El autofetch solo avisa, o también avanza el repo de specs cuando está limpio?»: «Avisar y avanzar».
  - (análisis, presentado y no discutido) Es un interruptor del usuario, opt-in y apagado por defecto. Activarlo es la autorización explícita y persistente que exige la regla de operaciones remotas.
  - (análisis) El interruptor tiene dos estados, apagado y encendido. Encendido equivale a fetch, aviso y avance. No hay nivel «solo avisar», porque el usuario eligió avanzar.
  - (análisis) Avance: solo la rama de integración local, con fast-forward estricto (`merge --ff-only`), y solo si el árbol está limpio y no hay divergencia. Aplica al repo de specs de un multirepo trabajado in situ y al checkout principal de un monorepo que está en la integración.
  - (análisis) Nunca avanza la rama de un cambio ni un worktree de feature. Ahí solo avisa, sin rebase ni merge.
  - (análisis) Si el árbol está sucio o hay divergencia, se degrada a solo aviso con la orden exacta. En multirepo es frecuente, porque los documentos se escriben directo en la integración.
  - (análisis) Qué se trae: solo `git fetch origin` del repo de specs. En `monorepo-embedded` es el propio repo. No se hace prune ni se tocan los repos de código.
  - (análisis) Aviso por relevancia:
    - commits entrantes en `origin/<integration_branch>` que tocan `openspec/changes/<cambio actual>/`, las specs afectadas por el cambio u `odd/tasks/<feature>.md`;
    - commits nuevos en `origin/<rama del cambio>`, por ejemplo tras una cesión con `axiom sdd publish`.
    - Lo demás no se avisa.
  - (análisis) Disparo:
    - al iniciar sesión, sin bloquear, con un timeout de unos 8 s y `GIT_TERMINAL_PROMPT=0`, y con un intervalo mínimo entre fetches;
    - antes de `axiom branch ensure`, donde equivale a `--fetch` si está encendido;
    - antes de presentar cada parada.
  - (análisis) Engram queda fuera de la relevancia, porque C-05-a retira `.engram/manifest.json`.
- **Fuera de alcance:**
  - pull, merge o rebase de ramas de cambio;
  - push;
  - fetch de los repos de código;
  - prune;
  - cualquier memoria de Engram;
  - autorización por `axiom.yaml`. Es una preferencia personal: RF-HUB-07 establece que las preferencias salen del hub y nunca de `axiom.yaml`.
- **Carril propuesto: SDD.** Cambia un contrato vivo. RF-BRA-10 (`openspec/specs/change-branch-lifecycle/spec.md:95,116`) dice «ninguna orden hace `fetch`, `pull`, `push` ni consulta remota sin una opción explícita». Hay que reformularlo para que la preferencia persistente cuente como opción explícita. Además, el cambio añade una escritura de git nueva (`merge --ff-only`) al motor de ramas y toca hooks de varios adaptadores.
- **Tamaño: M.** En el análisis se estimó S-M, ~300-500 líneas. El delta de RF-BRA-10 y los hooks por adaptador lo empujan a M.
- **Depende de / bloquea a:**
  - Después de C-05-a, que deja `AGENTS.md:17` y `GEMINI.md:48` en «solo con autorización»; aquí se reescriben para nombrar el interruptor.
  - Sección 01 (RDD apagado y retirado): no apoyarse en el SessionStart de RDD (`internal/cli/review_stop_hook.go:44-63`) ni en el código de `axiom review mode` (`internal/cli/review_mode.go`, `internal/reviewtransaction/rdd_mode.go`), porque se retiran. Se copia el patrón de interruptor del usuario, no el código.
  - Sección 02, C-02-a (`git.main_branch` + `promotion: branch|release|ask`): la rama que se avanza es `integration_branch`. `main_branch` nunca se avanza: solo se avisa (análisis). En las olas del README, C-02-a va antes que este cambio.
  - Sección 03 (OpenSpec única): la relevancia lee el layout de `openspec/specs/` vigente.
- **Ficheros:**
  - `internal/branch/gitcli/ops.go:449` (`Fetch`, hoy la única operación remota) y la nueva escritura ff.
  - `internal/branch/engine/engine.go:64-73` (interfaz `gitWriter`).
  - `internal/branch/engine/ensure.go` (fetch implícito con el interruptor encendido).
  - `internal/branch/engine/status.go:301` (ahead/behind; sección `remote` nueva en `axiom branch status --json`).
  - Preferencia personal en el hub (`~/.axiom/workspaces.json`): `internal/branch/engine/prefs.go:18-28` (`hubPrefs`) e `internal/branch/ports.go:48-50` (`Prefs`), con un método nuevo.
  - CLI del interruptor (nombre propuesto: `axiom branch autofetch enable|disable|status`, que la sesión de diseño decida) en `internal/cli/branch.go`.
  - Hooks SessionStart: hoy `internal/components/sdd/inject.go` los instala para Codex y Kiro. Para Claude y el resto hay que verificar qué hook queda tras retirar el de RDD.
  - [COMPARTIDO] La regla de fetch en `internal/components/agentguidance/routing.go:144`, `internal/assets/skills/_shared/sdd-orchestrator-sections.md:78` e `internal/assets/skills/sdd-apply/SKILL.md:122`, más 13 goldens en `testdata/golden/` (`combined-claude-claudemd.golden`, `sdd-claude-claudemd.golden`, `sdd-codex-agentsmd-powerful.golden`…), y los tests que la fijan (`internal/assets/assets_test.go`, `routing_test.go`, `internal/workspace/docs_contract_test.go`).
  - `AGENTS.md:17,20` y `GEMINI.md:48`.
  - `docs/manual-de-inicio.md:226,236,243`, que describen `--fetch` «solo con tu autorización».
  - `internal/workspace/docs_contract_test.go`, que fija frases de la política local de `AGENTS.md` y del manual.
  - `openspec/specs/change-branch-lifecycle/spec.md` (RF-BRA-10).
  - `openspec/specs/sdd-block-review-gates/spec.md:410` («`axiom branch ensure --fetch` con su autorización»), si cambia la redacción.
- **Criterio de terminado:**
  - Con el interruptor apagado, ninguna orden hace red (los tests actuales de RF-BRA-10 siguen pasando).
  - Con el interruptor encendido:
    - el SessionStart hace fetch y avisa solo de lo relevante;
    - la integración limpia avanza;
    - con el árbol sucio o divergencia, solo avisa con la orden;
    - una rama de cambio nunca se mueve.
  - Un remoto inalcanzable no bloquea el arranque y deja un aviso.
  - La regla de la doctrina pasa a «Never pass `--fetch` without the explicit authorization of the user, unless the user enabled autofetch» o una equivalente. Goldens regenerados.
- **Riesgos y avisos:**
  - El test de aislamiento `internal/sddpublish/isolation_test.go:238-261` vigila que `internal/branch/**` no ejecute `push` ni `commit`. `merge --ff-only` no está en esa lista, pero hay que revisar si existe otra guarda de arquitectura que limite las escrituras del motor (`ops_write_test.go:195`, `TestFetchBringsTheRemoteCommitsLaunchingOnlyFetchOrigin`).
  - Las credenciales en un fetch en segundo plano pueden colgar el proceso: usar `GIT_TERMINAL_PROMPT=0` y timeout.
  - El avance del checkout principal en monorepo coincide con la política local de este repo («solo recibe fast-forwards», `CLAUDE.md`). No debe interferir con `scripts/axiom-worktree.ps1 done`, que también avanza `main`.
  - Aviso del análisis: durante esta sesión se ejecutó por descuido un `git fetch` en `C:\repos\axiom` sin autorización (no movió `origin/main`). Es la misma tentación que este cambio debe cerrar con el interruptor.

## Descartado

- **Compartir la memoria de Engram entre personas** (`engram sync` a `.engram/` en git, o `engram cloud`). Lo decidió el usuario: «no me convence lo que me cuentas del engram, como axiom es siempre both y tenemos las informacion en openspec, tendra que ser suficiente». Motivos que se le presentaron:
  - `engram sync` no filtra por scope (lo `personal` viaja) ni por tipo o `topic_key` (https://raw.githubusercontent.com/Gentleman-Programming/engram/main/docs/TEAM-USAGE.md, «enrolled project means full project export»);
  - solo redacta `<private>…</private>`;
  - con persistencia both, compartir los espejos `sdd/*` y `odd/*` crearía dos fuentes de verdad.
- Se descartan con ello las vías A (sync en bruto), B (envoltorio `axiom memory publish|pull`), C (filtro aguas arriba en Engram) y D (registro de decisiones aparte en OpenSpec).
- **Publicación automática en ODD.** Lo decidió el usuario: «ODD no pasa nada, esta enfocado a monorepos. no hace falta subirlos automaticamente».
- **Un nivel «solo avisar» del autofetch.** El usuario eligió «Avisar y avanzar».
- **Reescribir el historial para eliminar los prompts heredados.** Lo descartó el análisis por el coste: push forzado sobre `main`. Se puede reconsiderar si el contenido resultara sensible.

## Correcciones a la consolidación

Referencia: Engram `analisis-backlog/consolidacion`, revisión del 2026-11-13.

- La limpieza (C-05-a) amplía su alcance respecto a lo consolidado, por hallazgos en `5f6b05b4`. Además de `.engram/`, `docs/engram.md` y `AGENTS.md:17`, incluye `GEMINI.md:48` (la misma orden de fetch pasivo), `docs/intended-usage.md:30-31,172` y `docs/codebase/sync-and-cloud.md:12,39,52`.
- Las líneas citadas se han movido: `docs/engram.md` es ahora `:50-62` (no `:52-62`) y el bloque de Engram de `.gitignore` está en `:53-55`.
- El autofetch figura como S-M sin más. Hay que anotar que contradice RF-BRA-10 («sin `fetch` implícito», `change-branch-lifecycle/spec.md:95,116`): exige delta de spec y carril SDD, y se estima en M.
- El autofetch no puede reutilizar el hook SessionStart ni el código de `axiom review mode`, que se retiran con RDD (sección 01). Solo se copia el patrón.
- Hay que precisar el carril de cada cambio: C-05-a por ODD (lote 1), C-05-b por ODD con `## Spec viva` (o SDD si la sesión padre lo prefiere), C-05-c por SDD y C-05-d por SDD.
