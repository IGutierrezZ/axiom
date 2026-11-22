# 06 · UI para que el humano no se salte la revisión (human in the loop)

Rutas verificadas contra `main` en `5f6b05b4`. Las decisiones se citan literales y con fecha. «Usuario» significa que la cerró el usuario. «Análisis» significa que es la propuesta de esta sesión y el usuario no la contradijo.

## Origen

Hay tres temas del backlog original, todos de UI, para que la persona no se salte la revisión:

1. **Configuraciones editables desde la UI**: que la UI (hub o UI de Axiom) tenga las configuraciones de usuario y de proyecto para tocarlas directamente.
2. **Mapa mental HTML de spec y design**: «En spec y en design, que se devuelva un HTML con un mapa mental que se entienda mucho mejor, menos técnico, para que lo entienda cualquiera, y que se pueda ver desde la UI. La mayoría se queja de que los md son un tocho que no se entiende».
3. **Batería de pruebas**: «Un documento con el listado de pruebas a realizar, como una batería que cubra bien todas las posibilidades en el menor número de pruebas […] Si ese listado puede verse en la UI y se pueden marcar fácilmente las que se han hecho, sería brutal».

Durante el análisis apareció además un bug previo. El detalle de un cambio en el dashboard no muestra la spec real de los cambios SDD ni las tareas por rol (C-06-a).

## Cambios resultantes

### C-06-a · El detalle del cambio muestra `specs/<dominio>/spec.md` y `tasks.<rol>.md`

- **Objetivo.** Hoy el detalle de un cambio solo lee `spec.md` y `tasks.md` planos de la raíz del cambio. sdd-spec escribe en `specs/<capacidad>/spec.md` y, con varios roles sellados, sdd-tasks escribe `tasks.<rol>.md`. Por eso la UI no enseña la spec real de ningún cambio SDD actual ni las tareas por rol. El resultado esperado es que el detalle muestre todas las specs de dominio del cambio, una por dominio, y todas las tareas por rol.
- **Decisiones.**
  - (Análisis, 2026-11-11, recogido en la consolidación como «bug de la UI») Se arregla como bug, antes que los demás cambios de esta sección. Va en el lote 1 del orden consolidado.
  - (Análisis) Reutilizar la misma regla de descubrimiento que el digest de la puerta `spec`: todos los ficheros `spec.md` bajo `specs/**` (`internal/cli/sdd_gate_digest.go:117`) y la de `tasks` (`internal/cli/sdd_gate_digest.go:65`). Así el detalle y la puerta ven los mismos ficheros.
- **Fuera de alcance.** Renderizar HTML, cambiar el renderizador de markdown o añadir pestañas nuevas que no sean esos artefactos.
- **Carril propuesto.** ODD. Es un bug acotado, sin decisión de producto.
- **Tamaño.** S.
- **Depende de / bloquea a.** No depende de nada. Bloquea a C-06-b y C-06-c, que añaden pestañas al mismo detalle.
- **Ficheros y paquetes que toca.**
  - `internal/dashboard/service.go:544` (`HasSpec` solo mira `spec.md` plano) y `:555` (`readFileString` de `spec.md` plano).
  - La lista del listado ya contempla `specs/` (`internal/dashboard/service.go:316-319`); el detalle no.
  - `internal/dashboard/types.go`: el DTO de detalle tiene que pasar a una lista de specs y de tareas por rol.
  - `internal/dashboard/assets/js/screens/detalle.js:18-25`: la lista fija `ARTIFACTS` y su render en `:148`.
  - Tests: `internal/dashboard/dashboard_test.go`, `server_test.go` y `listing_test.go`.
- **Criterio de terminado.**
  - Un cambio con `specs/a/spec.md` y `specs/b/spec.md` muestra ambas en el detalle.
  - Un cambio multirrol con `tasks.core.md` y `tasks.qa.md` muestra las dos.
  - Un cambio heredado con `spec.md` y `tasks.md` planos se sigue viendo.
  - Hay tests en `internal/dashboard` que cubren los tres casos.
- **Riesgos y avisos.**
  - La sección 03 (OpenSpec única, P6 ya fusionada) y el futuro cambio de IDs funcionales RF pueden tocar la disposición de las specs. La regla tiene que salir de un único punto compartido con el digest, no duplicarse.

### C-06-b · HTML explicativo de spec y design (`resumen-spec.html` y `resumen-design.html`), visible en la UI

- **Objetivo.** Los md de spec y design son «un tocho» para quien no es técnico, y eso favorece que el humano apruebe sin leer. El objetivo es que las fases `sdd-spec` y `sdd-design` entreguen, además del md, un HTML explicativo con gráficos y diagramas que entienda cualquiera, visible desde la UI y publicado con el resto de la documentación.
- **Decisiones.**
  - (Usuario, 2026-11-13) «en mi cabeza era una peticion mas en la skill de spec o de design para que no solo entreguen el md sino tambien un html mucho mas entendible por un usuario con graficos, diagramas o lo que sea necesario para explicar bien que se va a realizar. y esto se pueda ver desde la ui.»
    - Por eso cambió la recomendación del análisis: el HTML lo **genera el modelo dentro de la propia fase** y no es un derivado determinista.
  - (Usuario, 2026-11-13) «los html también es exclusivo de sdd, no es para odd».
  - (Usuario, 2026-11-13) Preguntar una sola vez **al iniciar el cambio**, no antes de cada fase.
    - El usuario lo planteó así: «nose si deberia preguntarse antes de lanzar spec o design si se queire hacer el html para que en pequeños cambios no realizar esfuerzos extra».
    - Se le propuso preguntarlo una vez al iniciar el cambio, con un valor por defecto de proyecto, y respondió: «Sí, al iniciar».
  - (Análisis, aceptado con esa respuesta) Clave de proyecto `explainer_html: ask|always|never`, por defecto `ask`.
    - Con `ask`, la pregunta se hace al iniciar el cambio y la respuesta se sella en `kickoff.yaml`. Vale para spec y design.
    - Con `always` o `never` no se pregunta.
  - (Análisis) Ficheros: `resumen-spec.html` y `resumen-design.html` en la raíz de la carpeta del cambio. Hay uno por cambio, no uno por dominio.
  - (Análisis) Contenido y forma:
    - En castellano y para público no técnico: qué cambia, para quién, flujos antes y después, diagramas SVG en línea y glosario.
    - Cada bloque cita el ID del requisito o de la decisión del md de origen. No añade nada que no esté en el md.
    - Dice expresamente que lo que se aprueba es el md.
  - (Análisis) Salvaguardas:
    1. **Autocontenido**: HTML, CSS y SVG en línea, sin JavaScript ni recursos externos. Funciona sin red y se abre con doble clic.
    2. **Frescura**: lleva `<meta name="axiom-source-digest">` con el hash de los md de los que sale. La UI y `axiom sdd status` lo marcan «desactualizado» si el md cambia. Si se rechaza una puerta, la fase regenera el md y el HTML.
    3. **Fuera del digest de la puerta**: la puerta aprueba el md, que es el contrato. Hoy el digest ya filtra por nombre exacto, así que un `*.html` no lo altera ni reabre la puerta.
    4. **Aislamiento**: la UI lo muestra en un `iframe` con `sandbox` (sin `allow-same-origin` ni `allow-scripts`). La respuesta lleva la cabecera `Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; img-src data:`.
       - Motivo: las rutas mutantes del dashboard solo exigen mismo origen (`isLocalRequest`). Un HTML del modelo servido desde el mismo origen podría llamarlas.
  - (Análisis, por la decisión de la sección 05) Al estar en la carpeta del cambio, el HTML se publica con `axiom sdd publish` en cada parada aprobada. En continuous con `docs_publication: auto` también se publica tras spec y tras design.
- **Fuera de alcance.**
  - ODD.
  - Generar HTML para proposal, tasks o verify.
  - Diagramas con librerías JS (mermaid, markmap).
  - Editar el HTML desde la UI.
  - Que el HTML entre en el digest de una puerta.
- **Carril propuesto.** SDD. Cambia el contrato de kickoff (un campo sellado nuevo), las skills de dos fases, el esquema de `axiom.yaml`, el estado de `axiom sdd status` y la UI. Afecta a las specs vivas `sdd-lane-kickoff`, `sdd-preflight-configuration`, `sdd-block-review-gates` y `local-web-dashboard`.
- **Tamaño.** M (unas 400-600 líneas, sin contar los goldens).
- **Depende de / bloquea a.**
  - Depende de C-06-a, porque la pestaña va en el mismo detalle.
  - Depende del cambio de la sección 01 que hace que el hook del pre-vuelo valide por etiqueta y marcador y no por número. Hoy exige exactamente 3 preguntas (`internal/cli/sdd_preflight_hook.go:363-364`) y no admite la pregunta nueva.
  - Conviene que vaya después del cambio de publicación de la sección 05 (`docs_publication: auto` tras spec y design), aunque funciona sin él.
  - (Revisión 2026-11-17) Depende de cómo se resuelva P10:
    - Con `explainer_html: ask`, la pregunta se hace al iniciar el cambio y tiene que tener sitio en el pre-vuelo que dejen C-01-b y C-02-c.
    - Según la sección 01, el pre-vuelo nunca queda vacío mientras C-02-c conserve la estrategia de PR.
    - Si P10 lo suprimiera en multirepo, la pregunta necesitaría una interacción propia que el hook admita.
    - Con `always` o `never` no hay pregunta.
  - No bloquea a nadie.
- **Ficheros y paquetes que toca.**
  - Skills:
    - [COMPARTIDO] `internal/assets/skills/sdd-spec/SKILL.md`, `internal/assets/skills/sdd-design/SKILL.md`.
    - [COMPARTIDO] Copias por agente en `internal/assets/{claude,cursor,kiro,kimi}/agents/sdd-{spec,design}.md`.
    - [COMPARTIDO] Goldens en `testdata/golden/sdd-{claude,kiro}-agent-sdd-{spec,design}.golden` y el resto de goldens de agentes que las incluyan.
  - Kickoff:
    - `internal/kickoff/types.go`: `FlowConfig` en `:111`, `Kickoff` en `:142`, junto a `DocsPublication` en `:127`.
    - `internal/kickoff/schema.go` y `internal/kickoff/args.go`: el flag de `seal`.
    - `internal/cli/sdd_kickoff.go`.
  - Pre-vuelo:
    - [COMPARTIDO] `internal/components/sdd/session_preflight.go`: el texto de la pregunta.
    - [COMPARTIDO] `internal/cli/sdd_preflight_hook.go:325-413`: validador de prefijos, marcadores y número de preguntas.
  - Esquema:
    - [COMPARTIDO] Esquema de `axiom.yaml`: decodificador estricto en `internal/workspace/declarations.go`, registrado como `git` y `workflow` (`:78-81`), con `unknownKeys` en `:276`.
    - [COMPARTIDO] `axiom.example.yaml`.
  - Estado: `internal/sddstatus/status.go`, para proyectar la frescura del HTML.
  - Dashboard:
    - `internal/dashboard/server.go`: un endpoint nuevo que sirve solo `resumen-{spec,design}.html` de un cambio, con CSP. Cabeceras en `withHeaders:215` y MIME en `init:31`.
    - `internal/dashboard/service.go` y `internal/dashboard/assets/js/screens/detalle.js`: pestaña «Explicación» con el `iframe` en sandbox y el aviso de desactualizado.
  - Orquestador: [COMPARTIDO] `internal/components/sdd/orchestrator.go`, si la parada de spec o design debe mencionar el HTML.
- **Criterio de terminado.**
  - Con `explainer_html: ask`, la pregunta aparece una sola vez al iniciar el cambio y su respuesta queda en `kickoff.yaml`.
  - Con `always` o `never` no se pregunta.
  - En ODD nunca aparece.
  - Con la respuesta afirmativa, sdd-spec y sdd-design escriben su HTML. Los dos llevan `axiom-source-digest` y no contienen `<script>` ni URLs externas (comprobación automática).
  - Si se modifica el md, `axiom sdd status` y la UI marcan el HTML como desactualizado.
  - El digest de las puertas `spec` y `design` no cambia al crear o modificar el HTML (test).
  - La UI sirve el HTML con CSP y `sandbox`. Un test comprueba las cabeceras y que el endpoint rechaza cualquier otro fichero o ruta con `..`.
- **Riesgos y avisos.**
  - Coste en tokens en las dos fases.
  - Riesgo de que la explicación simplifique de más o diga algo que no está en el md. Lo mitigan la cita de IDs y la revisión junto al md.
  - Choque con la sección 01 si su cambio del pre-vuelo no está fusionado.
  - C-07-h exige, como requisito previo de seguridad, CSP y retirar `Access-Control-Allow-Origin: *` (`internal/dashboard/server.go:217`). C-06-b añade CSP solo a la ruta del HTML. Los dos tocan `withHeaders` (`:215`): el segundo en fusionarse rebasa sin duplicar cabeceras.
  - El idioma sigue el contrato de artefactos: castellano.

### C-06-c · Batería de pruebas de aceptación (`pruebas.md`) generada en sdd-tasks

- **Objetivo.** Los equipos necesitan un listado de pruebas para probar en local, desarrollo o preproducción que cubra todas las posibilidades con el menor número de pruebas, revisable antes de implementar y visible en la UI.
- **Decisiones.**
  - (Usuario, 2026-11-13) «las pruebas lo realizaría solo en multirepo».
  - (Usuario, 2026-11-13) «lanzar el fichero de pruebas podria ser un parametro de proyecto para crearlo en los proytectos que se quiera y solo en sdd».
  - (Usuario, 2026-11-13) «los entornos a guardar las pruebas sera por proyecto que igual va de la mano con la configuracion de git de entnos que se dan de alta. aunque en realidad se dan rams de alta no entornos, por lo que esto sera una configuracion exclusiva de pruebas que defina los entornos a probar.»
  - (Análisis) Clave de proyecto: bloque `acceptance_tests: { enabled: bool, environments: [ ... ] }` en `axiom.yaml`.
    - No se llama `testing:` porque `openspec/config.yaml:45` ya usa `testing:` para el runner automático (`strict_tdd`, `runner`, `layers`).
    - La validación rechaza `enabled: true` cuando la topología no es multirepo.
  - (Análisis) Momento de generación: en **sdd-tasks, después de aprobar el design**.
    - En checkpointed se revisa en la parada `tasks`, junto a los `tasks.<rol>.md`.
    - En continuous se publica sin parada.
    - No se genera en la PR de cada rol, porque saldría del código ya escrito, con sesgo de confirmación, y llegaría tarde para preparar datos y entornos.
    - Cada caso lleva su `rol`, así que la vista por PR de rol es un filtro.
    - En `sdd-apply` de cada rol solo se refina (rutas o URL concretas), sin añadir alcance.
  - (Análisis) `pruebas.md` entra en el digest de la puerta `tasks` para que su aprobación cubra el plan de pruebas. Si se edita después de aprobado, la puerta se reabre como cualquier artefacto de tareas.
  - (Análisis) Minimización, por capas:
    1. Trazado obligatorio: cada requisito y cada escenario Given/When/Then de las specs del cambio queda cubierto por al menos un caso.
    2. Particiones de equivalencia y valores límite sobre las entradas que declara el design.
    3. Combinaciones (pairwise) solo con tres o más parámetros independientes con varios valores.
    4. Fusión de los casos con las mismas precondiciones.
  - (Análisis) La v1 no lleva generador pairwise en Go. El modelo aplica la técnica y Go solo **comprueba la cobertura** (todo requisito y escenario referenciado) e informa de ella: escenarios cubiertos M/M y número de casos. Pairwise determinista, más adelante si hace falta.
  - (Análisis) Trazado al ID del requisito con un patrón agnóstico: hoy `REQ-n.m` y, tras el cambio RF de la sección 03, `RF-<dom>-<n>` o el patrón declarado por el proyecto.
  - (Análisis) Formato de `pruebas.md`, en castellano:
    - Una tabla con ID estable `P-NN`.
    - Columnas: cubre (IDs de requisito y escenario), rol, tipo (manual o automatizable), prioridad, entornos aplicables (subconjunto de `acceptance_tests.environments`), precondiciones, pasos y resultado esperado.
    - Parseable de forma determinista.
- **Fuera de alcance.**
  - ODD.
  - Monorepo.
  - Las marcas de ejecución (C-06-d).
  - Sincronizar con Azure DevOps Test Plans u otro gestor de pruebas: la sección 07 no lo cubre y no se ha analizado.
  - Generar código de test.
- **Carril propuesto.** SDD. Añade una clave al esquema y un artefacto nuevo al contrato de la fase tasks y al digest de una puerta. Afecta a las specs vivas `sdd-block-review-gates`, `sdd-lane-kickoff` y `workspace-topology`.
- **Tamaño.** M.
- **Depende de / bloquea a.**
  - Depende de C-06-a (el detalle muestra `pruebas.md`).
  - Depende del cambio de la sección 01 que deja los roles solo en multirepo.
  - Conviene que vaya después del cambio de IDs funcionales RF de la sección 03, para no trazar dos veces. Si va antes, el parser debe aceptar ya ambos patrones.
  - Bloquea a C-06-d.
- **Ficheros y paquetes que toca.**
  - Skills:
    - [COMPARTIDO] `internal/assets/skills/sdd-tasks/SKILL.md`: «Step 3: Write tasks.md» en `:62` y el roster multirrol en `:74`.
    - [COMPARTIDO] `internal/assets/{claude,cursor,kiro,kimi}/agents/sdd-tasks.md`.
    - [COMPARTIDO] `testdata/golden/sdd-{claude,kiro}-agent-sdd-tasks.golden`.
  - Digest de la puerta `tasks`: `internal/cli/sdd_gate_digest.go:50,65` y su espejo en `internal/sddstatus/governance.go` y `internal/sddstatus/status.go`.
  - Esquema: [COMPARTIDO] decodificador nuevo `acceptance_tests` en `internal/workspace/declarations.go`, con validación por topología; [COMPARTIDO] `axiom.example.yaml`.
  - Comprobación de cobertura: paquete nuevo (p. ej. `internal/acceptancetests`) más su comando (`axiom sdd tests check --change <c>` o similar) y su proyección en `axiom sdd status`.
  - UI: pestaña de solo lectura en `internal/dashboard/assets/js/screens/detalle.js` y en `internal/dashboard/service.go`.
- **Criterio de terminado.**
  - Con `acceptance_tests.enabled: true` en un multirepo, sdd-tasks escribe `pruebas.md` con IDs `P-NN`.
  - La comprobación de cobertura falla si algún requisito o escenario de las specs del cambio no está referenciado e informa de M/M.
  - En monorepo, `enabled: true` se rechaza al validar `axiom.yaml`.
  - Con `enabled: false` o sin bloque no se genera nada.
  - En ODD nunca se genera.
  - Editar `pruebas.md` tras aprobar `tasks` reabre la puerta (test).
- **Riesgos y avisos.**
  - Más tokens en sdd-tasks.
  - La calidad de la minimización depende del modelo en la v1.
  - El patrón de IDs depende de la sección 03.
  - Coordinar el texto de la parada `tasks` del orquestador con los cambios de paradas de la sección 02 (vocabulario de stops y exits).

### C-06-d · Marcas de ejecución por entorno en la UI, publicadas con push a la rama de integración del repo de specs

- **Objetivo.** Que los equipos marquen en la UI qué pruebas de `pruebas.md` han hecho en cada entorno y con qué resultado, y que esas marcas sean compartidas por todo el equipo.
- **Decisiones.**
  - (Usuario, 2026-11-13) «las marcas se subes push directo a integración porque es repo de spec».
  - (Usuario, 2026-11-13) Solo multirepo y solo SDD, heredado de C-06-c.
  - (Usuario, en la sección 05, 2026-11-13) Engram compartido descartado: «OpenSpec/ODD en git es la fuente compartida suficiente». Por eso las marcas van en git y no en el hub local por máquina (`~/.axiom/`), que no se comparte.
  - (Análisis) Fichero `pruebas-estado.yaml` en la carpeta del cambio, separado de `pruebas.md`: el plan queda congelado por la puerta y las marcas son estado de ejecución.
    - Se escribe solo añadiendo entradas.
    - Cada entrada: `case` (P-NN), `environment` (uno de `acceptance_tests.environments`), `result` (`ok|ko|blocked|n/a`), `by` (identidad git), `at` (fecha y hora), `commit` (commit o build probado), `plan_digest` (hash de `pruebas.md`) y `note` (texto o enlace de evidencia).
  - (Análisis) Si cambia el hash de `pruebas.md`, las marcas con un `plan_digest` antiguo se muestran como **caducadas**.
  - (Análisis) Pantalla en la UI: la lista de casos con una columna por entorno; cada celda se marca `ok`, `ko` o `bloqueada` con nota.
    - Botón «Publicar marcas»: hace commit de `pruebas-estado.yaml` y push directo a la rama de integración del repo de specs, por el mismo camino que `axiom sdd publish`.
    - Pulsar el botón es la autorización explícita del push.
  - (Análisis) Quién marca: las personas marcan dev, pre y el resto de entornos. Un agente solo puede registrar evidencia automatizada en `local`.
  - (Análisis) La puerta `integration`, que ya exige evidencia de despliegue a pre o prod antes de archivar (`internal/assets/skills/sdd-archive/SKILL.md:76`), muestra el resumen de marcas como **evidencia informativa, no bloqueante**. Ejemplo: «pre: 5/5 ok».
  - (Análisis) Las marcas no bloquean el archivado. Es coherente con verify, que tampoco lo bloquea. El `archive-report` incluye el resumen.
- **Fuera de alcance.**
  - Monorepo y ODD.
  - Sincronizar marcas con Azure DevOps Test Plans u otro gestor.
  - Bloquear el archivado por marcas.
  - Que la UI publique en ramas de cambio o en worktrees.
- **Carril propuesto.** SDD. Hay un artefacto de estado nuevo, una escritura remota desde la UI y un cambio en la parada `integration`. Afecta a `local-web-dashboard`, `sdd-block-review-gates` y a la spec viva de publicación.
- **Tamaño.** M.
- **Depende de / bloquea a.**
  - Depende de C-06-c.
  - Depende de la publicación de documentación de la sección 05 (`axiom sdd publish`, ya existe en `internal/sddpublish`) y de la rama de integración del multirepo de la sección 02.
  - No bloquea a nadie.
- **Ficheros y paquetes que toca.**
  - Publicación: `internal/sddpublish/publish.go`. Hoy publica lo que está sucio bajo `changes/<c>` (`:106`, `:162`), así que `pruebas-estado.yaml` entra en su alcance. Falta el disparo desde la UI y el destino «rama de integración del repo de specs» después de la fusión.
  - Dashboard:
    - `internal/dashboard/server.go`: endpoints de marca y de publicación, protegidos con `isLocalRequest` (`:1050`).
    - `internal/dashboard/service.go`.
    - Una pantalla nueva en `internal/dashboard/assets/js/screens/`.
  - Estado: `internal/sddstatus/status.go`, para el resumen de marcas y la caducidad.
  - Archivo:
    - [COMPARTIDO] `internal/assets/skills/sdd-archive/SKILL.md:76,96`: mostrar el resumen en la puerta `integration` y en el `archive-report`.
    - [COMPARTIDO] Goldens de sdd-archive.
- **Criterio de terminado.**
  - Desde la UI se marca un caso en un entorno y se añade una entrada a `pruebas-estado.yaml` con identidad git, commit y `plan_digest`.
  - «Publicar marcas» hace commit y push a la rama de integración del repo de specs y muestra el resultado o el rechazo tipado de `sddpublish`.
  - Al cambiar `pruebas.md`, las marcas antiguas aparecen caducadas.
  - `axiom sdd status` y la puerta `integration` muestran el resumen por entorno.
  - El archivado no se bloquea por marcas.
  - En monorepo la pantalla no está disponible.
- **Riesgos y avisos.**
  - El push desde la UI es una operación remota y debe quedar a la vista; nunca debe ejecutarse sin el clic.
  - Puede haber conflictos si dos personas publican a la vez. Escribir solo añadiendo los mitiga, pero hace falta reintento con *rebase*, o un rechazo claro.
  - Las marcas se ponen tras la fusión, cuando la carpeta del cambio vive en la rama de integración del repo de specs. Hay que comprobar que la UI lee de esa rama y no de un worktree de cambio ya retirado.

### C-06-e · Pantalla «Ajustes del proyecto» basada en un catálogo de claves

- **Objetivo.** Hoy `axiom.yaml` solo se crea en el alta. `internal/hub/init.go:390` lo genera concatenando strings, y no existe ningún endpoint que lo edite después. La UI solo edita preferencias del hub (`worktree_mode` y `worktree_root`). El resultado esperado es que la persona cambie desde la UI las claves de proyecto que gobiernan su flujo, sin abrir el YAML y sin perder comentarios.
- **Decisiones.**
  - (Usuario, 2026-11-13) «la configuracion el rdd lo he sacado de axiom por lo que no hara falta». No hay ningún control de RDD: ni global, ni por proyecto, ni por clon.
  - (Usuario, en la sección 01, 2026-11-11) Roles solo en multirepo; el monorepo siempre usa `fullstack`; persistencia siempre hybrid; el multirepo sella siempre checkpointed.
    - Por eso no se editan `roles` ni `gate_policy` en monorepo, ni `artifact_store`.
    - En multirepo se edita `gate_policy`, incluido `optional`, que el alta actual nunca emite (`internal/hub/init.go`, bloque de roles).
  - (Usuario, en la sección 03) Se retira `baseline:` de `axiom.yaml`, y un proyecto ISO no usa ODD. Por eso `workflow.lane` no admite `odd` en un proyecto ISO.
  - (Análisis) La pantalla se construye desde un **catálogo de claves**: cada clave declara etiqueta, descripción, tipo, valores permitidos, si es editable y su efecto. Así, las claves nuevas de otras secciones se registran en el catálogo sin rehacer formularios:
    - sección 02: `git.main_branch` y `git.promotion`, también por repositorio (C-02-a); `delivery.split_hint_lines` solo si P5 la confirma como clave (C-02-c);
    - sección 03: el patrón de ID y el bloque `sdd:` (C-03-a);
    - esta sección: `explainer_html` (C-06-b) y `acceptance_tests` (C-06-c).
  - (Revisión 2026-11-17) Quedan fuera del catálogo dos claves que la primera versión daba por claves de `axiom.yaml` y no lo son:
    - `commit_review`: la parada por tarea es un campo de la decisión de la puerta `tasks` en SDD (C-02-d); ODD va siempre en modo 1, no una clave de proyecto.
    - `docs_publication`: es un campo de `kickoff.yaml` (`internal/kickoff/types.go:122`, `schema.go:99-105`), no de `axiom.yaml`.
  - (Usuario, en la sección 07, C-07-h) «todas las configuraciones que hablamos a nivel de proyecto o lo que sea tienen que estar en la ui».
    - Corrige la v2 de esta sección: `integrations.<plugin>` **se edita** desde la UI, no queda en solo lectura.
    - Esa edición la hace C-07-h con su propia pantalla «Integraciones», generada desde el manifiesto del plugin y escrita con el catálogo y el escritor `yaml.Node` de este cambio. C-06-e no edita `integrations.*`.
    - Nunca guarda secretos en `axiom.yaml`: el PAT va al almacén del sistema (C-07-h).
  - (Análisis, por la decisión de la sección 02) La instantánea del kickoff manda sobre `axiom.yaml` en los cambios sellados. La pantalla avisa de que **editar solo afecta a cambios nuevos**. `git.promotion` no es editable para cambios ya creados.
  - (Análisis) El escritor YAML hace un *round-trip* con `yaml.Node`: conserva comentarios, orden y bloques ajenos. Valida con los decodificadores estrictos existentes (`decodeGit` en `internal/workspace/gitpolicy.go:86`, `decodeWorkflow` en `internal/workspace/workflow.go:55` y `workspace.Validate`). Las escrituras pasan por `isLocalRequest`.
  - (Análisis, **pendiente de contrastar** con la frase del usuario en C-07-h) Lo estructural o de cambio raro se muestra en solo lectura con «abrir en el editor»: `technical_context`, `workspace.root`, `skill_roots` y `governance.*`. Si «tienen que estar en la ui» significa editables, estas claves también pasan a editarse. La decide el usuario (ver `revision-06.md`).
  - (Análisis) El autofetch es un ajuste personal (sección 05), no del proyecto, y no entra en esta pantalla.
  - (Análisis) Va **al final** de la sección, después de que lleguen las claves nuevas. Las claves que lleguen más tarde las registra en el catálogo el cambio que las introduce, como hace C-07-h con `integrations.<plugin>`.
- **Fuera de alcance.**
  - RDD.
  - Preferencias del instalador en `state.json` (persona, preset, modelos por fase).
  - `openspec/config.yaml`.
  - Editar secretos o credenciales.
  - Migrar `axiom.yaml` entre topologías.
- **Carril propuesto.** SDD. Hay un escritor nuevo de un fichero de configuración versionado, con validación cruzada y efecto sobre el kickoff. Afecta a `local-web-dashboard` y `workspace-topology`.
- **Tamaño.** L.
- **Depende de / bloquea a.**
  - Depende de los cambios que introducen las claves que cataloga:
    - C-02-a (`git.main_branch`, `git.promotion`);
    - C-02-c (`delivery.split_hint_lines`, si P5 la confirma);
    - C-03-a (patrón de ID y `sdd:`);
    - C-06-b y C-06-c.
  - Depende de C-01-c (quita el control de RDD del dashboard) y de C-01-b (roles solo en multirepo).
  - **No depende de C-05-b ni de C-07-b**: `docs_publication` no es clave de `axiom.yaml`, e `integrations.<plugin>` lo edita C-07-h.
  - Bloquea a C-07-h, que usa su catálogo y su escritor.
- **Ficheros y paquetes que toca.**
  - Esquema y escritor:
    - [COMPARTIDO] `internal/workspace/`: `types.go:19-48`, `declarations.go:75-81,276`, `gitpolicy.go:86,111,207`, `workflow.go:33,55`.
    - Escritor nuevo con `yaml.Node`.
    - [COMPARTIDO] `axiom.example.yaml`.
  - Dashboard: endpoints en `internal/dashboard/server.go` (registro de rutas en `:74-115`), servicio en `internal/dashboard/service.go` y pantalla nueva en `internal/dashboard/assets/js/screens/`.
  - Lectura actual a sustituir o ampliar: `GET /api/workspace` y `internal/dashboard/assets/js/screens/ecosystem.js` y `internal/dashboard/assets/js/core/branches.js`.
- **Criterio de terminado.**
  - Cambiar `workflow.lane` desde la UI escribe `axiom.yaml` conservando comentarios y orden (test con golden de un `axiom.yaml` comentado).
  - Un valor inválido se rechaza con el error del validador y no escribe nada.
  - `odd` en un proyecto ISO se rechaza.
  - En monorepo no aparecen roles ni `gate_policy`.
  - La pantalla indica «solo cambios nuevos» y un cambio ya sellado conserva su instantánea.
  - No aparece ningún control de RDD.
- **Riesgos y avisos.**
  - El *round-trip* de `yaml.Node` con comentarios tiene casos límite (anclas, comentarios de cabecera y de pie).
  - Riesgo de pisar a mano un fichero que otra persona esté editando a la vez: conviene comparar la revisión antes de escribir.
  - El catálogo debe ser la única fuente; no duplicar las listas de claves cerradas de los decodificadores.

### C-06-f · Registrar decisiones de puerta desde la UI (propuesta del análisis, sin confirmar)

- **Objetivo.** Hoy la UI solo lee `gates.yaml`, para los roles pendientes (`internal/dashboard/service.go:403`, `kickoff.LoadGates`). No se puede aprobar ni rechazar una puerta desde la UI. Si el humano revisa el HTML (C-06-b) o la batería (C-06-c) en la UI, lo natural es decidir allí mismo, con motivo obligatorio al rechazar.
- **Decisiones.**
  - (Análisis, 2026-11-11) Propuesto como complemento del tema 1 («aporta más contra saltarse la revisión»).
  - **El usuario no lo ha confirmado ni descartado.** La sesión padre debe preguntárselo antes de planificarlo.
  - (Análisis) Si se hace, la escritura pasa por `kickoff.AppendGateChecked` o por `axiom sdd gate record`, con las tres salidas de las paradas: iterar, aprobar, publicar y parar, y aprobar y continuar. La puerta `integration` conserva sus dos salidas.
- **Fuera de alcance.** Cambiar la semántica de las puertas.
- **Carril propuesto.** SDD.
- **Tamaño.** M.
- **Depende de / bloquea a.** Depende de C-06-a y de los cambios de paradas de la sección 02.
- **Ficheros y paquetes que toca.**
  - `internal/cli/sdd_gate.go:70` (`guardChangeWriter`), `internal/kickoff/ledger.go:91` (`AppendGateChecked`) y `internal/kickoff/machine.go:250`.
  - Dashboard: `server.go`, `service.go` y `assets/js/screens/detalle.js`.
- **Criterio de terminado.** Por definir cuando el usuario lo confirme.
- **Riesgos y avisos.** «Aprobar, publicar y parar» implica commit y push; el mismo aviso de operación remota que C-06-d.

## Descartado

- **Mapa mental derivado de forma determinista en Go** a partir de la estructura del md: árbol plegable en SVG servido como JSON, con líneas «> En claro:» dentro del md.
  - Era la primera recomendación del análisis.
  - Se descartó como primera opción tras la aclaración del usuario («una peticion mas en la skill de spec o de design»): solo pinta la estructura y no explica.
  - Queda como posible respaldo si el HTML del modelo resultara poco fiable.
- **HTML del modelo servido sin aislamiento o con JS.** Desde el mismo origen podría llamar a las rutas mutantes del dashboard.
- **Mermaid o markmap embebidos.** Habría que vendorizar unos 2 MB y unos 300 KB en `internal/dashboard/assets` (`//go:embed assets/*`, `internal/dashboard/assets.go:8`). El dashboard funciona sin red y sin librerías.
- **Preguntar por el HTML antes de cada fase** (usuario, 2026-11-13: «Sí, al iniciar»). Serían dos interrupciones para la misma decisión.
- **Batería en monorepo y en ODD** (usuario, 2026-11-13).
- **Generar la batería en la PR de cada rol.** Sesgo de confirmación y demasiado tarde para preparar entornos.
- **Generador pairwise determinista en Go en la v1.** Más adelante, si hace falta.
- **Marcas en el hub local por máquina o en Engram compartido.** No se comparten entre el equipo, y el Engram compartido se descartó en la sección 05.
- **Marcas por PR nueva en monorepo.** Ya no aplica: la batería es solo de multirepo.
- **Entornos derivados de las ramas de `git.*`** (usuario, 2026-11-13: «en realidad se dan rams de alta no entornos»).
- **Bloque `testing:` en `axiom.yaml`.** Choca con `testing:` de `openspec/config.yaml:45`.
- **Editar «todas» las claves desde la UI** (análisis). Coste alto y riesgo de romper comentarios para claves que casi no cambian. **En revisión:** el usuario dijo en C-07-h «todas las configuraciones que hablamos a nivel de proyecto o lo que sea tienen que estar en la ui». Si eso significa editables, este descarte cae. Pendiente del usuario (ver `revision-06.md`).
- **`integrations.tracker` en solo lectura** (v2 de este análisis). Lo corrigió el usuario en C-07-h: se edita desde la UI.
- **Controles de RDD en la UI** (usuario, 2026-11-13: RDD sacado de Axiom).

## Correcciones a la consolidación

La entrada de Engram `analisis-backlog/consolidacion` se actualizó en esta sesión el 2026-11-13 con la línea de UI revisada. Quedan dos matices:

- La primera versión decía «Configuración acotada más aprobar puertas desde la UI». La versión actual ya no menciona las puertas.
  - Aprobar puertas desde la UI no está decidido por el usuario: queda como C-06-f, pendiente de confirmar.
- La consolidación dice «Bug: el detalle no lee specs/<dom>/spec.md ni tasks.<rol>.md (internal/dashboard/service.go:539)».
  - En `main` `5f6b05b4` las líneas son `internal/dashboard/service.go:544` y `:555`.
  - Su lote es el 1, no el 6.
