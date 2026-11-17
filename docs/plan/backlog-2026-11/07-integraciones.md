# 07 · Integraciones con terceros (Azure DevOps primero)

## Origen

Tema 7 del backlog original, tal como lo planteó el usuario: «Incluir en los proyectos plugins de integración de terceros. Ahora nos enfocamos en Azure, pero lo dejamos preparado para cualquiera. Si un proyecto declara integración con Azure, hay que pedir a cada usuario un token para conectarse, o vemos qué necesitamos según la funcionalidad. Así podríamos traer la información de un work item para arrancar un incremento o un bug, o, si no existe, crearlo desde la spec generada, y usar el ID de la herramienta externa para unificar IDs.»

El usuario lo amplió después:
- Cruzarlo con el SDD anterior que está en producción en KVP25: `Kvp.Sdd/tools/sdd-launcher`, una extensión de VS Code que da de alta work items, estima, imputa horas y gestiona relaciones y tags. No se copia: se mejora, más ágil y como plugin.
- Toda acción sobre Azure DevOps es una parametrización de proyecto.
- Cada persona declara quién es.
- Plantilla de nombre de rama configurable por proyecto y por rol.
- Toda la configuración se edita desde la UI.

Análisis completo, con evidencia y fuentes: `side-session-notes/.../07-integraciones-terceros.md` (v1, v2 y decisiones posteriores). En Engram: `analisis-backlog/integraciones-terceros` y sus subtemas `/arranque`, `/decisiones-v3`, `/ramas`, `/design-y-resumen`, `/odd`, `/horas` y `/continuous`.

### Estado de partida (main 5f6b05b4, verificado)

- **Trackers.** No hay nada de trackers en el binario: ni puerto, ni bloque en `axiom.yaml`, ni HTTP saliente a un tracker. Las llamadas salientes son GET a GitHub y de salud (`internal/update/github.go`, `internal/autoskill/client.go`, `internal/components/engram/download.go`).
- **Bloques de `axiom.yaml`.** Se decodifican uno a uno en `readDeclarationBlocks` (`internal/workspace/declarations.go:59-96`), con lista cerrada de claves a través de `unknownKeys` (`declarations.go:276`). Las claves de git son `integration_branch` y `branch_prefix` (`internal/workspace/gitpolicy.go:34,55`).
- **Secretos.** No hay almacén de secretos. El único token es el de GitHub, que se lee de variables de entorno o de `gh auth token` y nunca se persiste (`internal/update/github.go:51`). La configuración personal no versionada vive en `~/.axiom/` (`internal/hub/manager.go:27-33`).
- **Identidad de persona.** No existe. El actor de una compuerta vale `cli` por defecto (`internal/cli/sdd_gate.go:100-103`; `--actor` es texto libre, `internal/kickoff/args.go:450`) y el kickoff guarda `sealed_by: "cli"` (`internal/kickoff/args.go:324`; campos en `internal/kickoff/types.go:146,191`).
- **Nombres de rama.** Son fijos:
  - Rama del cambio: `<prefijo>/<cambio>` (`internal/branch/planner.go:380`).
  - Rama de rol: `<rama del cambio>-<rol>` (`planner.go:402`), solo con 2 o más roles en el mismo repo.
  - La instantánea del kickoff prevalece y congela los nombres (`planner.go:384-388`, `internal/branch/planner_snapshot.go:28`; `GitSnapshotBranch` en `internal/kickoff/gitsnapshot.go:57`).
  - La única excepción es `--branch`, y es por invocación.
- **ID del work item como nombre.** `AB#1234` no es un nombre de cambio válido, porque `#` está rechazado (`internal/gitref/gitref.go:115`). `US-2525-alta-usuarios` sí lo es.
- **`external_ref`.** No existe en `main`. El archivado de `carril-sdd-versatil` (#257) recortó `story.md` y solo dejó la intención («se reserva el hueco»).
- **Perfil de bug.** Va en `proposal.md`, con `## Informe de bug` y `## Impacto en la spec` (`internal/isoevidence/bug.go:8-11`).
- **Inyección de MCP por agente.** Ya existe para Context7 (`internal/components/mcp/context7.go:39-89`, `internal/catalog/components.go:15`).
- **Dashboard.** Solo escucha en `127.0.0.1` (`internal/dashboard/server.go:146`). Las mutaciones se protegen con `isLocalRequest` (`server.go:1045`). Las respuestas llevan `Access-Control-Allow-Origin: *` (`server.go:217`) y no hay CSP.

## Cambios resultantes

### C-07-a · Identidad git como actor de los artefactos versionados

- **Objetivo.** Axiom no sabe quién hace cada cosa: compuertas y kickoff registran `cli`. Resultado esperado: cada fichero versionado que dice quién hizo qué registra la identidad git configurada. Así se puede saber quién está con qué y quién hizo qué en cada fase de un incremento o bug.
- **Decisiones:**
  - (Usuario) «lo que tengan configurado entiendo que será su identidad de git». En los ficheros versionados que registran quién hizo qué se escribe la identidad git configurada (`user.name <user.email>`): el `actor` de `gates.yaml`, el `sealed_by` de `kickoff.yaml` y, cuando exista, `integrations.yaml` (quién inició o cerró un rol y quién imputó horas).
  - (Usuario) La identidad de Azure DevOps vive solo en `~/.axiom/` de cada persona (no versionada). Se resuelve al conectar el plugin y sirve para hablar con ADO (asignar a la persona). Esa parte la implementa C-07-c.
  - (Análisis) `--actor` explícito sigue prevaleciendo. Si git no tiene identidad configurada, se conserva el valor actual (`cli`) y se avisa; nunca se bloquea.
- **Fuera de alcance:** vincular con ADO (C-07-c) y la vista «quién está con qué» en la UI (C-07-h).
- **Carril propuesto: SDD.** Añade comportamiento nuevo (el actor por defecto). Ninguna spec viva fija hoy el valor `cli`: `rg` sobre `openspec/specs` no lo encuentra, y `sealed_by` solo aparece en `sdd-lane-kickoff` y `sdd-preflight-configuration` como `inferred`. Por eso el requisito es ADDED. Propuesta del análisis: una capacidad nueva (p. ej. `change-actor-identity`) en lugar de `sdd-block-review-gates`, para no crear un solape duro con C-06-c, que va en la misma ola.
- **Tamaño: S.**
- **Depende de / bloquea a:** no depende de nada. Bloquea a C-07-e (que registra quién imputó) y a C-07-h (vista por persona). Es independiente del plugin y puede ir en cualquier momento.
- **Ficheros y paquetes que toca:**
  - `internal/cli/sdd_gate.go:100-103`.
  - `internal/kickoff/args.go:324` y su lectura de identidad.
  - `internal/kickoff/types.go:146,191` (solo la documentación del campo; el tipo no cambia).
  - Pruebas de `internal/kickoff` y `internal/cli`.
  - **No toca guías de agente.** Corregido en la revisión cruzada: ninguna guía de `internal/assets` ni de `internal/components/agentguidance` cita `--actor`; solo aparece en código Go.
- **Criterio de terminado:**
  - Con `git config user.name/user.email` definidos, `axiom sdd gate record` sin `--actor` escribe `actor: "Nombre <email>"` y `kickoff seal` escribe `sealed_by` con esa identidad.
  - Sin identidad git, el valor sigue siendo `cli` y hay un aviso.
  - Los registros anteriores siguen siendo válidos.
- **Riesgos y avisos:**
  - `sealed_by: inferred` del retro-sellado conserva su significado especial (`internal/kickoff/schema.go:149-154`): no se sustituye.
  - El email entra en artefactos versionados. Ya es público en los commits, así que no añade exposición.

### C-07-b · Marco de plugins de integración

- **Objetivo.** Dejar Axiom preparado para cualquier tracker (Azure DevOps primero; Jira y Redmine después) sin acoplar el núcleo a ninguno. Resultado esperado: el núcleo ofrece un puerto con capacidades, un bloque de configuración estricto, almacenamiento seguro de secretos, un fichero máquina por cambio con los vínculos y una cola de salida idempotente. Los eventos del ciclo de vida salen de los verbos deterministas.
- **Decisiones:**
  - (Usuario) Todo lo que hoy hace el launcher contra ADO debe ser **parametrización de proyecto**: «al elegir un plugin externo, en este caso azure devops, se despliegue la configuración de azure devops para seleccionar cómo quiere que se comporte en cada parte».
  - (Usuario) «no quiero copiar lo que tengo en el launcher, quiero darle una vuelta, mejorarlo, que sea más ágil y tiene que funcionar como un plugin».
  - (Análisis) Un plugin es un **adaptador Go compilado** más un **manifiesto declarativo**. El manifiesto declara:
    - el identificador del plugin;
    - sus capacidades: `items.read`, `items.transition`, `items.create` (solo para las Tasks de rol), `relations`, `assignment`, `estimation`, `time`, `tags`, `attachments`, `vcs_links` y `discovery`;
    - el esquema de configuración, que alimenta la UI (C-07-h);
    - los eventos que escucha;
    - cómo se resuelve la identidad.
  - (Análisis) El contrato es serializable en JSON, para poder añadir más adelante plugins fuera de proceso (stdio) sin tocar el núcleo. **No** hay carga dinámica de código.
  - (Análisis) La configuración versionada va en un bloque `integrations.<plugin>` de `axiom.yaml`, estricto como el resto de bloques: sin secretos ni datos personales. Los nombres de tipos, estados y campos se eligen a partir de `discovery`, no se teclean. Así se evita el hardcodeo del launcher (21 nombres de campo, estados y equipos en el código).
  - (Análisis) Eventos, siempre a partir de verbos deterministas que ya existen:
    - `change.start` (nuevo verbo, C-07-d);
    - `gate.spec.approved` y `gate.design.approved`;
    - `role.start` (el `ensure` del rol);
    - `role.apply.approved`;
    - `archive`.
    - En `continuous`, las mismas fases se señalan al terminar (sin compuerta). Ver C-07-e.
  - (Análisis) Los vínculos del cambio van en un fichero máquina del cambio, `integrations.yaml`: el WI, la Task de cada rol, las imputaciones y la cola pendiente. `kickoff.yaml` no sirve porque es de escritura única. Se prohíben las heurísticas de título, que el launcher sí usa (`roleTaskResolutionService`, búsqueda por puntuación).
  - (Análisis) Cola de salida (outbox) idempotente por cambio, con reintento y `axiom integrations sync`. Una compuerta aprobada nunca falla porque el tracker no responda. Excepciones síncronas, porque algo posterior depende de ellas: el arranque (C-07-d) y la creación de Tasks (C-07-e).
  - (Usuario, en la v1) El token es de cada usuario y nunca va al repo ni a configuración versionada. (Análisis) Se guarda en el almacén del sistema (Windows Credential Manager, Keychain o Secret Service, p. ej. con `github.com/zalando/go-keyring`), con una variable de entorno como alternativa. La caída a texto plano nunca es silenciosa.
- **Fuera de alcance:** el adaptador concreto (C-07-c) y la UI (C-07-h).
- **Carril propuesto: SDD.** Esquema nuevo de `axiom.yaml`, fichero máquina nuevo y verbos nuevos, con spec viva nueva.
- **Tamaño: L** (unas 1.500-2.000 líneas con pruebas).
- **Depende de / bloquea a:** depende de C-07-a de forma blanda, solo para el registro de quién hizo qué. Bloquea a C-07-c, C-07-d, C-07-e, C-07-g y C-07-h, y a C-06-e, que registra en su catálogo las claves de `integrations.<plugin>`.
- **Ficheros y paquetes que toca:**
  - Paquete nuevo `internal/integrations` (puerto, manifiesto, registro, outbox y secretos), siguiendo el patrón de puertos de `internal/branch/ports.go` y el registro cerrado de `internal/agents/registry.go`.
  - Bloque `integrations` en `internal/workspace/declarations.go:59-96` [COMPARTIDO: esquema de `axiom.yaml`].
  - `axiom.example.yaml` [COMPARTIDO].
  - `cmd/axiom/main.go` (verbo `integrations`).
  - `go.mod` (dependencia de keyring).
  - Disparo de eventos desde `internal/cli/sdd_gate.go` (`runSDDGateRecord`, `:53`) y desde `internal/branch/engine/ensure.go:79`.
- **Criterio de terminado:**
  - Un plugin de prueba (doble en memoria) registrado y configurado en `axiom.yaml` recibe los eventos de cada verbo, persiste los vínculos en `integrations.yaml` y reintenta desde la outbox.
  - Una clave desconocida en `integrations.*` es un error de carga.
  - Ningún secreto llega a disco versionado ni a la salida de un GET.
- **Riesgos y avisos:**
  - El token puede filtrarse al contexto del agente o a los logs si se pasa por argumentos: hay que redactarlo siempre.
  - La outbox duplica operaciones si no son idempotentes: cada operación necesita una clave estable.

### C-07-c · Adaptador Azure DevOps: cliente REST, conexión, identidad y descubrimiento

- **Objetivo.** El primer adaptador real: un cliente REST robusto, la conexión con el PAT de cada persona, la identidad de ADO resuelta y el descubrimiento del proceso del proyecto.
- **Decisiones:**
  - (Usuario) «diría que tenemos mcp con entra y el api rest de azure con pat». Lo determinista va por **REST con PAT** desde el binario; el MCP con Entra es para el agente (C-07-g).
  - (Usuario) «será necesario que cada persona defina quién es al meter el plugin para poder enlazar con la persona en azure devops, así podemos asignarlos en las task». La identidad de ADO se resuelve al conectar y se guarda solo en `~/.axiom/` de la persona.
  - (Análisis) Un solo cliente:
    - timeouts;
    - respeto de `Retry-After` y `X-RateLimit-*` (ADO puede limitar con HTTP 200);
    - lotes (`workitemsbatch`);
    - caché de repos;
    - concurrencia optimista con `{"op":"test","path":"/rev"}` de JSON Patch.
    - Corrige los fallos del launcher: transporte triplicado, sin reintentos ni timeouts, `catch {}` silenciosos e imputación con read-modify-write sin control.
  - (Análisis) `discovery` devuelve tipos de work item, estados por tipo (con su categoría estable: Proposed, In Progress, Resolved, Completed, Removed), campos por tipo, iteraciones y miembros del equipo. La asignación usa sugerencias de miembros, nunca emails en el código.
  - (Análisis) API REST 7.1:
    - `GET .../_apis/wit/workitems/{id}?$expand=relations`;
    - `POST .../workitems/${type}` y `PATCH .../workitems/{id}`, con `application/json-patch+json`;
    - enlace a rama con `ArtifactLink` y `vstfs:///Git/Ref/...`, formato que el launcher usa en producción (`adoWorkflowService.ts:1144-1185`).
- **Fuera de alcance:**
  - Entra ID en el binario: el binario usa PAT y Entra queda para el MCP.
  - Azure DevOps Server on-premises.
  - Sincronizar despliegues (builds y pipelines) y el informe de uso de Copilot. El launcher los tiene; si se quieren, irían como plugin aparte.
- **Carril propuesto: SDD.**
- **Tamaño: M** (unas 800-1.100 líneas con pruebas).
- **Depende de / bloquea a:** depende de C-07-b. Bloquea a C-07-d, C-07-e y C-07-h (pantalla de conexión).
- **Ficheros y paquetes que toca:** paquete nuevo `internal/integrations/azuredevops`, y la identidad por persona en `~/.axiom/`, junto a `workspaces.json` (`internal/hub/manager.go:27-33`).
- **Criterio de terminado:**
  - Con un servidor HTTP de prueba:
    - conecta con PAT y resuelve la identidad;
    - devuelve el `discovery` completo;
    - reintenta ante un 429 o un `Retry-After`;
    - rechaza una escritura con `/rev` obsoleto sin pisar datos.
  - El PAT queda en el almacén del sistema y no aparece en ningún log.
- **Riesgos y avisos:**
  - **No verificado:** que `_apis/connectionData` (`authenticatedUser`) responda con un PAT. Microsoft indica que la API de perfiles exige Entra. Hay que comprobarlo al empezar. Si falla, la persona elige su usuario de la lista de miembros del equipo.
  - Microsoft retira los PAT globales (la documentación da dos fechas: 2026-12-01 y 2027-01-27). Hay que usar PAT de una sola organización con `vso.work_write`.

### C-07-d · Arranque de un cambio desde un work item existente

- **Objetivo.** Con el plugin activo, todo cambio o bug, SDD u ODD, nace de un work item que ya existe en Azure DevOps. Axiom lo importa como contexto, lo enlaza y unifica los IDs.
- **Decisiones:**
  - (Usuario) «seria mejor obligar a que tenga que tener el workitem del cambio o el bug creado antes de empezar en axiom … crealo en azure y luego di quiero un nuevo cambio de … que su workitem es XXXX, y axiom leerá el workitem por si hay más información además de la proporcionada, imágenes para descargar, enlazará todo, y empezará el explore para preguntar las dudas». Axiom **no crea** el WI del cambio ni del bug.
  - (Usuario) «si al empezar un cambio no ha indicado el workitem que se lo pregunte».
  - (Usuario) «si tiene el plugin, siempre obligatorio», en SDD y en ODD.
  - (Usuario) «poner en la ui un formulario de creación de workitem me parece redundante cuando puede abrir una pestaña al lado de azure y hacerlo allí». La UI solo ofrece un campo para pegar el ID o la URL y un enlace «Abrir en Azure DevOps».
  - (Usuario) «el contenido del work item sí que habría que tener cuidado con instrucciones inyectadas; datos personales: en realidad todo proyecto que trabaja con IA ya tiene firmado lo de datos personales, aceptado por el cliente».
  - (Usuario) «limitemos a tamaños y tipos comunes que se suelen utilizar y no es necesario versionar». Los adjuntos y las imágenes se descargan a `context/` del cambio y no se versionan.
  - (Usuario) «si cambia el contenido después de arrancar, culpa del que lo cambia; se coge el contenido, si lo hay, al arrancar como contexto extra». No hay re-importación.
  - (Análisis) Verbo determinista nuevo, común a SDD y ODD (p. ej. `axiom change start --from <id|url>`). Va **antes** de `axiom branch ensure` y del kickoff, y hace esto:
    1. Valida el WI: existe, su tipo está mapeado, no está cerrado y no está enlazado a otro cambio abierto. Lo comprueba con los `integrations.yaml` y con un enlace de vuelta en el WI.
    2. Importa título, descripción y pasos de reproducción (HTML → Markdown), criterios de aceptación, relaciones (padre y Related), y adjuntos e imágenes incrustadas.
    3. Deriva el nombre del cambio (`US-2525-alta-usuarios`, a partir de `keys` del bloque `integrations`).
    4. Escribe `external_ref: ado:2525`, el formato tipado que propuso la sesión 3.
    5. Añade al WI un Hyperlink o un tag de vuelta.
    6. Lanza el explore con ese contexto.
  - (Análisis) **El tipo del WI decide el perfil**: User Story → incremento; Bug → perfil de bug en `proposal.md` (`## Informe de bug`, `internal/isoevidence/bug.go:8-11`).
  - (Análisis) Es síncrono, sin outbox: sin ID no hay nombre de cambio ni rama. Si ADO no responde, el verbo se niega con la continuación «reintenta».
  - (Usuario, al cerrar ODD) En ODD el WI obligatorio se importa igual al arrancar. La rama usa `{ref}` y los eventos (estado, enlace de rama o PR) actúan sobre el propio WI.
- **Fuera de alcance:**
  - Crear el WI del cambio o del bug desde Axiom o desde la UI.
  - Re-importar o sincronizar el WI.
  - Sincronización bidireccional.
- **Carril propuesto: SDD.** Verbo nuevo y regla nueva de arranque en ambos carriles.
- **Tamaño: M** (unas 600-900 líneas con pruebas).
- **Depende de / bloquea a:** depende de C-07-b y C-07-c. Bloquea a C-07-e y C-07-f (`{ref}`).
- **Ficheros y paquetes que toca:**
  - `cmd/axiom/main.go` (verbo nuevo).
  - `internal/integrations`.
  - Validación del nombre con `internal/branch/naming.go:27` y `internal/gitref/gitref.go:103-143`.
  - Plantilla o skill de `proposal.md` y del documento ODD, para `external_ref` [COMPARTIDO]: `internal/assets/skills/sdd-propose/SKILL.md`, `internal/assets/skills/sdd-explore/SKILL.md` y `internal/assets/skills/_shared/sdd-iso-convention.md`.
  - Guía del orquestador, para que pida el WI si falta [COMPARTIDO]: `internal/components/agentguidance/routing.go` y `internal/assets/skills/_shared/sdd-orchestrator-sections.md`.
  - Goldens [COMPARTIDO]: `testdata/golden/`.
- **Criterio de terminado:**
  - Con el plugin activo y sin WI, el orquestador lo pide y el verbo se niega.
  - Con un WI válido:
    - se crea `context/` con la descripción en Markdown y los adjuntos permitidos;
    - el nombre del cambio sale del ID;
    - `external_ref` queda escrito y el WI recibe el enlace de vuelta;
    - un Bug produce el perfil de bug.
  - Un WI cerrado, de un tipo sin mapear o ya enlazado a otro cambio produce un rechazo tipado.
- **Riesgos y avisos:**
  - El contenido importado no es fiable: hay que marcarlo como dato y no como instrucción en lo que se pasa al agente.
  - Adjuntos: lista blanca de tipos y tamaño máximo (análisis: imágenes comunes, PDF, Office y texto, unos 10 MB por fichero; ajustable).
  - `context/` no se versiona: hay que añadirlo a `.gitignore` en el patrón del cambio, o guardarlo fuera del repo.

### C-07-e · Eventos del ciclo de vida: Tasks por rol, estimación, resumen de la spec e imputación

- **Objetivo.** Automatizar lo que el launcher hace a mano en cada fase, con confirmación humana donde el usuario la pidió y sin paradas en `continuous`.
- **Decisiones:**
  - (Usuario) «lo de que axiom crea la task al aprobar las tareas, en realidad es al aprobar el design, que en ese momento ya se conocen los roles que participan y pueden crearse las task en azure, ya que se crea una única task por rol de cada workitem aunque luego por debajo ese rol tenga muchas task a realizar. No mezclar task de axiom sdd con task de azure devops».
  - (Usuario) «si falla crear la rama que es al aceptar el design, en ese momento no se crean las ramas ya que no han entrado los roles aún, de hecho es un punto de parada. Si no consigue crear las task, la aprobación del design deberá indicarlo y ofrecer reintentar o que la cree a mano e indicar los IDs de las task de cada rol».
    - Salidas: reintentar (p. ej. `axiom integrations tasks create --change <c>`) o vincular a mano (p. ej. `axiom integrations tasks bind --change <c> --role back=1234 --role front=1235`).
    - El binario valida que cada ID existe, es de tipo Task y cuelga del WI del cambio.
  - (Análisis, propuesto y no rebatido) La aprobación del design se registra igualmente. Lo que se bloquea es la entrada de cada rol: su `ensure` se niega mientras su Task no esté vinculada en `integrations.yaml`.
  - (Usuario) Estimación: «al crearlas, el llm debería proponer una estimación según el contenido a desarrollar por ese rol, pero deberá confirmar el usuario que cierra el design las horas estimadas de cada task». Se escribe `OriginalEstimate` y, al inicio, `RemainingWork` = estimación. Las horas confirmadas viajan en la misma invocación (p. ej. `--estimate back=16 --estimate front=8`); el reintento y el vínculo manual reutilizan esa entrada.
  - (Usuario) «El resumen va a la descripción del work item reemplazando lo que tuviera.»
    - Cuándo: al validar la spec (compuerta `spec` aprobada, o fin de la fase spec en `continuous`).
    - Formato: «en formato bonito un resumen de la spec» (análisis: HTML).
    - Es idempotente: si la spec se rechaza y se vuelve a aprobar, se reemplaza otra vez.
    - El original queda en `context/` y en el historial de revisiones del WI.
  - (Usuario) Imputación **solo al cerrar el rol**: «al cerrar el rol se le podría proponer también unas horas, que pueden ser las que ha tardado el rol desde que ha iniciado las tareas hasta que las ha finalizado, indicándole las horas estimadas que lleva la task y las horas acumuladas imputadas que lleva esa task».
    - El binario lee de ADO la estimación y las horas ya imputadas en `CompletedWork`, también las que alguien metió a mano en Azure.
    - Tras la confirmación, suma a `CompletedWork` y ajusta `RemainingWork` con `test /rev`.
    - Registra en `integrations.yaml` quién imputó (identidad git, C-07-a), cuánto y cuándo.
  - (Análisis) La propuesta de horas recorta el tiempo de reloj a una jornada configurable por proyecto (p. ej. 8 h, de lunes a viernes). Siempre es una sugerencia y manda la confirmación.
  - (Usuario) `continuous`: «si da a continuous se le indica que acepta la estimación que calcule el llm sin parada y luego al final en el resumen se le indica lo que se ha puesto. Lo mismo para la imputación».
    - Sin paradas.
    - El resumen final del cambio informa de la estimación por Task, de las horas imputadas, de quién y de cuándo.
    - El disparador determinista es el fin de la fase design y el fin del apply de cada rol.
  - (Usuario) ODD: **sin Task en Azure**. Como no hay Task ni rol, Axiom no estima ni imputa.
  - (Análisis) Eventos de estado y asignación por defecto, todos configurables:
    - `role.start`: Task a `Active` y asignada a la persona (identidad de ADO de C-07-c).
    - `role.apply.approved`: Task a `Tech Review`, o al estado que configure el proyecto.
    - `archive`: transición del WI.
- **Fuera de alcance:**
  - Verbo libre de imputación (`axiom time log`): el usuario eligió «al cerrar el rol».
  - Crear Tasks en ODD.
  - Comentarios en el WI.
- **Carril propuesto: SDD.**
- **Tamaño: L** (unas 1.000-1.400 líneas con pruebas).
- **Depende de / bloquea a:**
  - Depende de C-07-a, C-07-b, C-07-c y C-07-d.
  - Bloquea a C-07-f, porque `{task_ref}` exige la Task.
  - Si se reutiliza el resumen HTML explicativo de la sección 06 (`resumen-spec.html`), depende de ese cambio (blanda).
  - C-03-b es una dependencia blanda: el resumen reproduce la spec con los IDs que tenga. Que esos IDs funcionales «se envíen también» al WI está pendiente del usuario (P12 del README; `revision-03.md`).
  - La regla «la aprobación del design se registra igualmente y se bloquea el `ensure` del rol» está pendiente de confirmar (P8).
- **Ficheros y paquetes que toca:**
  - `internal/cli/sdd_gate.go:53`, para disparar eventos tras registrar.
  - `internal/branch/engine/ensure.go:79`, para negarse sin Task vinculada.
  - `internal/integrations`.
  - Skills [COMPARTIDO]: `internal/assets/skills/sdd-design/SKILL.md` (propuesta de estimación por rol), `internal/assets/skills/sdd-apply/SKILL.md` (propuesta de horas al cerrar el rol) y `internal/assets/skills/sdd-archive/SKILL.md` (informe final en `continuous`).
  - Orquestador [COMPARTIDO]: `internal/assets/skills/_shared/sdd-orchestrator-sections.md` y `internal/components/agentguidance/routing.go`.
  - Goldens [COMPARTIDO].
- **Criterio de terminado** (con el doble de ADO):
  - Aprobar el design crea una Task por rol con la estimación confirmada.
  - Un fallo deja la aprobación registrada, ofrece las dos salidas y bloquea el `ensure` del rol hasta que haya vínculo.
  - Aprobar la spec reemplaza la descripción.
  - Cerrar el rol muestra la estimación y las horas acumuladas, y escribe la imputación sin pisar cambios concurrentes.
  - En `continuous` no hay paradas y el resumen final lista lo escrito.
  - En ODD no se crea ninguna Task.
- **Riesgos y avisos:**
  - **No verificado:** qué HTML acepta `System.Description`. Es probable que filtre SVG y CSS externos, así que hará falta una variante sencilla del resumen.
  - **No verificado:** si los campos de horas existen en User Story o Bug según el proceso. No afecta, porque las horas van solo a las Tasks y ODD no imputa.

### C-07-f · Plantillas de nombre de rama y ramas de código diferidas a la entrada del rol

- **Objetivo.** Que cada proyecto, repo y rol defina cómo se llama su rama y que, con el plugin, se pueda usar la referencia externa. Por ejemplo, la convención de KVP es `feature/<taskId>` en back y `feature/TASK-<taskId>` en front (`Kvp.Sdd/tools/sdd-launcher/src/services/roleBranchService.ts:215-285`).
- **Decisiones:**
  - (Usuario) «el tema de ramas tendrá que ser configurable cómo se genera el nombre de las ramas en cada proyecto y en cada rol, y si usa un plugin debería poder utilizar el external ref como parte de la rama».
  - (Análisis) `git.branch_template` por proyecto, con sobrescritura por repositorio (`repositories:`) y por rol. Variables: `{prefix}`, `{kind}` (feature|fix), `{change}`, `{role}`, `{slug}`, `{ref}` (WI del cambio) y `{task_ref}` (Task del rol). El resultado se valida con `gitref`. Sin plantilla, el comportamiento actual no cambia.
  - (Usuario) Opción **B**: «Tras aprobar el design». Las ramas de código se crean cuando cada rol entra en el apply, no al arrancar. La instantánea del kickoff guarda la **plantilla** de cada rama de código, y el **nombre** se resuelve y se congela en el `ensure` del rol, cuando ya existe su Task.
  - (Usuario, precisión) Al aprobar el design **no** se crean ramas («no han entrado los roles aún»); solo Tasks (C-07-e).
  - (Análisis) Límite: la rama que lleva los artefactos de `openspec/` existe desde el arranque y no puede usar `{task_ref}`, así que usa `{ref}`.
    - Esa rama es la del cambio en un monorepo y la del repo de specs en un multirepo.
    - En monorepo `fullstack` (sección 01) hay una sola rama y se nombra con `{ref}`.
    - `{task_ref}` aplica a los repos de código de un multirepo y a las ramas de rol.
  - (Análisis) Cruza con la sección 02 (entrega Git), sin conflicto: la promoción `branch|release|ask` decide la **base** y la plantilla decide el **nombre**.
- **Fuera de alcance:** la base y la promoción de la rama, que son de la sección 02.
- **Carril propuesto: SDD.** Modifica `change-branch-lifecycle` y el congelado D-10 de inc-25.
- **Tamaño: M** (unas 500-800 líneas con pruebas).
- **Depende de / bloquea a:**
  - Depende de C-07-d (`{ref}`) y de C-07-e (`{task_ref}`), aunque la parte sin plugin (plantilla con `{prefix}`, `{kind}`, `{change}` y `{role}`) puede ir antes.
  - Conviene que vaya tras la sección 02, que toca el mismo bloque `git:` y la instantánea del kickoff.
- **Ficheros y paquetes que toca:**
  - `internal/workspace/gitpolicy.go:34,55` [COMPARTIDO: esquema de `axiom.yaml`].
  - `internal/branch/planner.go:380-410`.
  - `internal/branch/planner_snapshot.go:14-45`.
  - `internal/kickoff/gitsnapshot.go:24,57` (la plantilla, además del nombre).
  - `internal/branch/naming.go`.
  - `internal/branch/engine/ensure.go:79`.
  - `axiom.example.yaml` [COMPARTIDO].
  - Pruebas de `internal/branch/...`.
- **Criterio de terminado:**
  - Con `branch_template` por repo y rol, el `ensure` del rol produce `feature/TASK-1235` cuando su Task es 1235.
  - El nombre queda congelado en la instantánea tras el primer `ensure`.
  - La rama del cambio con `openspec/` usa `{ref}`.
  - Sin plantilla, los nombres son idénticos a los de hoy.
  - Una plantilla que produce un nombre inválido se rechaza al cargar o en el `ensure`, con un rechazo tipado.
- **Riesgos y avisos:**
  - Los cambios ya sellados no tienen plantilla en su instantánea: hay que tratarlos con el comportamiento actual, sin migración.
  - El `ensure` de un rol cuya Task no existe se niega (C-07-e).

### C-07-g · MCP de Azure DevOps para el agente y frontera agente/binario

- **Objetivo.** El agente consulta Azure DevOps (contexto, duplicados, redacción) por MCP con Entra, y nunca escribe en los eventos del ciclo de vida, que son del binario.
- **Decisiones:**
  - (Usuario) «tenemos mcp con entra», «aquí verás que usamos mcp y rest de azure devops para cosas deterministas».
  - (Análisis) Cuando el plugin está activo, Axiom inyecta el servidor MCP oficial de Azure DevOps en cada agente, con el mismo patrón que Context7.
    - El servidor local (`npx`, Node 20 o superior) admite PAT o Entra.
    - El servidor remoto `https://mcp.dev.azure.com/{org}` admite solo Entra. Claude Code y Cursor necesitan un registro de app Entra propio.
  - (Análisis) Regla del orquestador: el agente no escribe en ADO en los eventos del ciclo de vida. Esto resuelve la contradicción de KVP: `Kvp.Sdd/CLAUDE.md:98-102` prohíbe HTTP mientras el launcher usa REST.
- **Fuera de alcance:** autenticación Entra en el binario.
- **Carril propuesto: ODD.** Es un componente más de inyección y texto de guía, sin spec de comportamiento nueva del binario. Si el equipo prefiere fijarlo en la spec de assets, puede ir por SDD.
- **Tamaño: S.**
- **Depende de / bloquea a:** depende de C-07-b (saber si el plugin está activo). No bloquea a nada.
- **Ficheros y paquetes que toca:**
  - `internal/components/mcp/` (nuevo junto a `context7.go`).
  - `internal/catalog/components.go:15`.
  - Guía [COMPARTIDO]: `internal/components/agentguidance/routing.go` y `internal/assets/skills/_shared/sdd-orchestrator-sections.md`.
  - Goldens [COMPARTIDO]: `testdata/golden/`.
- **Criterio de terminado:**
  - Con el plugin activo, cada agente soportado tiene la entrada MCP de ADO y la guía contiene la regla.
  - Sin plugin, nada cambia.
- **Riesgos y avisos:** la versión del paquete MCP debe ir fijada, como la de Context7 (`versions.Context7MCP`).

### C-07-h · UI de integraciones

- **Objetivo.** Que toda la configuración de integración se gestione desde la UI.
- **Decisiones:**
  - (Usuario) «todas las configuraciones que hablamos a nivel de proyecto o lo que sea tienen que estar en la ui». Esto **corrige** la v2 de la sección 06, que dejaba `integrations.tracker` en solo lectura.
  - (Análisis) Pantalla «Integraciones» generada desde el manifiesto del plugin. Al activarlo se despliegan sus secciones: conexión, tipos, eventos, estados, campos, tags, plantillas de rama y jornada para la propuesta de horas.
    - Los desplegables se rellenan con `discovery` (C-07-c).
    - Se escribe con el escritor `yaml.Node` y el catálogo de claves de la sección 06.
  - (Análisis) «Conectar»: el PAT se envía por POST a un endpoint local protegido por `isLocalRequest` (`internal/dashboard/server.go:1045`), se guarda en el almacén del sistema y **nunca se devuelve**. «Probar conexión» muestra la identidad resuelta.
  - (Usuario) No hay formulario de alta de WI: solo un campo de ID o URL al arrancar y un enlace «Abrir en Azure DevOps» (ver C-07-d).
  - (Análisis) Vista por cambio: WI enlazado, Task por rol con su estimación e imputaciones, operaciones pendientes de la outbox y quién hizo cada fase (C-07-a).
- **Fuera de alcance:** formulario de alta de work items.
- **Carril propuesto: SDD** (spec `local-web-dashboard`).
- **Tamaño: L** (unas 1.000-1.400 líneas con pruebas).
- **Depende de / bloquea a:**
  - Depende de C-07-b y C-07-c.
  - Depende del catálogo de claves y del escritor `yaml.Node` de la pantalla «Ajustes del proyecto» de la sección 06 (C-06-e).
  - Depende de C-07-d: el campo para pegar el ID o la URL del WI al arrancar invoca su verbo.
  - Depende de C-07-e: la vista por cambio muestra las Tasks, las imputaciones y la outbox que genera ese cambio.
  - Depende de C-07-a: la vista de quién hizo cada fase.
  - Depende de C-07-f de forma blanda: la pantalla incluye las plantillas de rama. Si C-07-h se fusiona antes, las claves de `branch_template` se registran en el catálogo cuando llegue C-07-f.
  - **Requisito previo de seguridad, dentro del alcance de este cambio** (primer commit): añadir CSP y quitar `Access-Control-Allow-Origin: *` de las respuestas que puedan llevar datos sensibles (`internal/dashboard/server.go:217`) antes de manejar tokens.
- **Ficheros y paquetes que toca:**
  - `internal/dashboard/server.go` (rutas nuevas, CSP).
  - `internal/dashboard/service.go`.
  - `internal/dashboard/assets/js/screens/` (pantalla nueva y vista por cambio en `detalle.js`) [COMPARTIDO con la sección 06].
- **Criterio de terminado:**
  - Se puede activar y configurar Azure DevOps solo desde la UI, y el resultado valida con el decodificador estricto.
  - El PAT no aparece en ninguna respuesta HTTP ni en `axiom.yaml`.
  - La vista del cambio muestra los vínculos, la outbox y los actores.
  - Una petición con `Origin` ajeno a una ruta que muta se rechaza.
- **Riesgos y avisos:** la UI es una superficie nueva para el secreto. Hay que probar expresamente CSRF y DNS rebinding sobre el endpoint de conexión.

## Descartado

- **Crear el WI del cambio o del bug desde Axiom** (al arrancar o al aprobar la spec). Lo decidió primero el análisis y después el usuario: se crea en Azure y Axiom lo importa. La creación de Axiom se limita a las Tasks por rol.
- **Formulario de alta de WI en la UI** (usuario): es redundante con la pestaña de Azure.
- **Re-importación o `refresh` del WI** (usuario): «culpa del que lo cambia».
- **Sincronización bidireccional** (análisis): exige resolver conflictos, webhooks o sondeo, y un servicio que Axiom no tiene. Los eventos van en un solo sentido.
- **Versionar adjuntos** (usuario).
- **Task de Azure en ODD** (usuario), y con ella la estimación y la imputación en ODD.
- **Verbo libre de imputación** (usuario): se imputa solo al cerrar el rol.
- **Paradas de estimación e imputación en `continuous`** (usuario, que corrigió su propia respuesta anterior): sin paradas, se informa en el resumen final.
- **Fase 1 «solo MCP»** (análisis v1, sustituida): lo determinista va por REST con PAT desde el binario.
- **Plugins con carga dinámica de código** (`~/.axiom/plugins/`, el antiguo Lote B de 0025 del producto TS) (análisis): choca con el registro cerrado en compilación y abre ejecución de código ajeno. Se deja el contrato serializable para plugins fuera de proceso en el futuro.
- **Heurísticas de título para encontrar la Task de un rol** (análisis): los vínculos se persisten.
- **El ID externo con `#` en el nombre** (`AB#1234`) (análisis): `gitref` lo rechaza. Se usa `US-2525-...` y `external_ref: ado:2525`.
- **Sincronizar despliegues e informe de uso de Copilot** (análisis): fuera de alcance; si se quieren, irían como plugin aparte. El usuario no lo ha decidido expresamente.

## Correcciones a la consolidación

La entrada de Engram `analisis-backlog/consolidacion` (#2167, actualizada el 2026-11-13) es anterior a la v2 de este análisis y **ya no coincide**. Dice: «Integraciones. Fase 1 con el MCP oficial de Azure, más integrations.tracker, external_ref y change start --from; fase 2 con el puerto Tracker REST; Entra ID antes que PAT.» Lo decidido después es:

- Lo determinista va por **REST con PAT** desde el binario desde el primer cambio. El MCP con Entra es para el agente (C-07-g). No hay «fase 1 solo MCP» ni «Entra antes que PAT» en el binario.
- No hay un `integrations.tracker` genérico de solo lectura en la UI: hay `integrations.<plugin>` con configuración por eventos, editable desde la UI (C-07-b y C-07-h).
- El WI es **obligatorio** con el plugin y **se crea en Azure**. Axiom solo lo importa al arrancar (C-07-d).
- Hay alcance nuevo que la consolidación no recoge:
  - identidad git como actor (C-07-a);
  - Tasks por rol y estimación al aprobar el design (C-07-e);
  - resumen de la spec que reemplaza la descripción del WI;
  - imputación al cerrar el rol;
  - plantillas de rama y ramas de código diferidas (C-07-f).
- El tamaño pasa de «fases» a 8 cambios: unas 5.500-7.500 líneas en total.
- Orden: la consolidación dice «integraciones (necesita RF y external_ref)».
  - `external_ref` no lo aporta el cambio RF: lo introduce C-07-d.
  - El cambio RF de la sección 03 no es requisito duro. Solo hace falta si se quiere citar los IDs funcionales en el resumen que se sube al WI.
  - C-07-a no depende de nada y puede adelantarse.
