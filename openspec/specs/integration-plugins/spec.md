# Especificación de integration-plugins

## Purpose

Marco de plugins de integración (C-07-b): un punto de extensión hacia trackers externos que no acopla el núcleo de Axiom a ninguno. Fija el puerto `Plugin`, el manifiesto serializable, los catálogos cerrados de eventos y capacidades, el registro cerrado en compilación, el despacho duradero con reintento (outbox local), el fichero versionado de vínculos por cambio, el almacén seguro de secretos, los verbos `axiom integrations` y las garantías de coste cero y de no fuga de secretos. El bloque `integrations:` de `axiom.yaml` y los enganches en `gate record`, `branch ensure --role` y `sdd publish --archive` se especifican en las capacidades `workspace-topology`, `sdd-block-review-gates`, `change-branch-lifecycle` y `sdd-docs-publication`. Ningún adaptador real (Azure DevOps, C-07-c) forma parte de este cambio.

## Requirements

### Requirement: Catálogo Cerrado de Eventos y Puerto `Plugin` (RF-INT-01)

El paquete `internal/integrations` DEBE definir el puerto `Plugin` y el tipo `Event`. El catálogo de eventos DEBE ser cerrado y contener exactamente `change.start`, `gate.spec.approved`, `gate.design.approved`, `role.start`, `role.apply.approved` y `archive`. `change.start` pertenece al catálogo, pero ningún verbo de este cambio lo emite (lo dispara C-07-d). Todo `Event` DEBE llevar el nombre del evento, el cambio, la clave de idempotencia (RF-INT-07) y, cuando proceda, el rol y el actor. Un nombre de evento fuera del catálogo NO DEBE poder construirse ni despacharse: se rechaza con error tipado.

#### Scenario: El catálogo contiene exactamente los seis eventos
- **DADO** el paquete `internal/integrations`
- **CUANDO** se enumera el catálogo de eventos
- **ENTONCES** contiene `change.start`, `gate.spec.approved`, `gate.design.approved`, `role.start`, `role.apply.approved` y `archive`, y ninguno más

#### Scenario: Evento fuera del catálogo
- **DADO** un despachador con un plugin configurado
- **CUANDO** se intenta despachar un evento llamado `gate.tasks.approved`
- **ENTONCES** el despacho se rechaza con un error tipado que nombra los eventos admitidos
- **Y** no se crea ningún fichero ni se invoca al plugin

### Requirement: Catálogo Cerrado de Capacidades (RF-INT-02)

El catálogo de capacidades DEBE ser cerrado y contener exactamente `items.read`, `items.transition`, `items.create`, `relations`, `assignment`, `estimation`, `time`, `tags`, `attachments`, `vcs_links` y `discovery`. El manifiesto de un plugin SOLO PUEDE declarar capacidades de ese catálogo.

#### Scenario: Capacidad declarada válida
- **DADO** un manifiesto que declara `items.read` y `vcs_links`
- **CUANDO** se valida
- **ENTONCES** la validación termina sin error

#### Scenario: Capacidad desconocida
- **DADO** un manifiesto que declara la capacidad `sprints`
- **CUANDO** se registra el plugin
- **ENTONCES** el registro se rechaza con un error tipado que nombra `sprints` y las capacidades admitidas

### Requirement: Manifiesto Serializable, Versionado y en un Paquete Hoja (RF-INT-03)

Cada plugin DEBE exponer un manifiesto serializable en JSON con esquema versionado (`axiom.integration-manifest/v1`; el nombre definitivo lo fija el diseño). El manifiesto DEBE declarar: identificador, capacidades, eventos que escucha, esquema de configuración, nombres de los secretos que necesita y cómo se resuelve la identidad (`identity`). El manifiesto vive en `internal/integrations/manifest`, un paquete hoja que NO DEBE importar `workspace` (ni, de forma transitiva, `internal/integrations`). La validación DEBE rechazar: capacidades o eventos fuera de los catálogos cerrados, un identificador que no cumple `^[a-z][a-z0-9]*(-[a-z0-9]+)*$` o supera 32 caracteres, y toda clave de configuración que parezca un secreto. El rechazo es **por nombre de clave**: el campo de configuración no tiene ningún atributo de secreto; se rechazan las claves cuyo segmento (separado por `_`) sea `token`, `secret`, `secrets`, `password`, `passwd`, `pwd`, `pat`, `credential`, `credentials` o `apikey`, las que contengan el par `api_key`, las que coincidan con un nombre de la lista de secretos y la clave reservada `enabled`. Los secretos se declaran solo por nombre en la lista de secretos. También DEBE rechazarse un catálogo con identificadores repetidos o con dos pares (plugin, secreto) que den la misma variable de entorno (RF-INT-13).

#### Scenario: Ida y vuelta JSON estable
- **DADO** un manifiesto válido
- **CUANDO** se serializa a JSON y se vuelve a leer
- **ENTONCES** el resultado es idéntico al original y el JSON lleva el campo de esquema versionado

#### Scenario: Evento escuchado desconocido
- **DADO** un manifiesto cuyos eventos escuchados incluyen `change.close`
- **CUANDO** se valida
- **ENTONCES** se rechaza con un error tipado que nombra `change.close`

#### Scenario: Clave de configuración con nombre de secreto
- **DADO** un manifiesto cuyo esquema de configuración declara la clave `api_token`
- **CUANDO** se valida
- **ENTONCES** se rechaza con un error tipado que indica que los secretos se declaran en la lista de secretos y no como configuración

#### Scenario: Clave de configuración que coincide con un secreto declarado
- **DADO** un manifiesto con el secreto `pat_org` y una clave de configuración `pat_org`
- **CUANDO** se valida
- **ENTONCES** se rechaza con un error tipado

#### Scenario: Clave reservada `enabled`
- **DADO** un manifiesto cuyo esquema de configuración declara la clave `enabled`
- **CUANDO** se valida
- **ENTONCES** se rechaza con un error tipado que indica que la clave está reservada

#### Scenario: Identificador inválido
- **DADO** un manifiesto con identificador `Azure DevOps`
- **CUANDO** se valida
- **ENTONCES** se rechaza con un error tipado que nombra el patrón admitido

#### Scenario: El paquete hoja no depende de `workspace`
- **DADO** el paquete `internal/integrations/manifest`
- **CUANDO** se comprueban sus importaciones, directas y transitivas
- **ENTONCES** no incluyen `internal/workspace` ni `internal/integrations`

### Requirement: Registro Cerrado en Compilación (RF-INT-04)

`NewRegistry(plugins...)` DEBE construir un registro de plugins y rechazar con error tipado un identificador duplicado, un manifiesto inválido, un plugin cuyo identificador no está en el catálogo de manifiestos y un plugin cuyo manifiesto difiere de la entrada del catálogo (el código del plugin y el manifiesto publicado no pueden divergir). `DefaultRegistry()` DEBE devolver un registro que contiene exactamente el plugin `azure-devops` (el primer adaptador real, C-07-c), y el catálogo de manifiestos por defecto DEBE contener exactamente `azure-devops`. El catálogo de manifiestos que usa `workspace` para validar `integrations.<plugin>` DEBE derivarse de la misma lista cerrada y DEBE ofrecer una costura de paquete que permita a las pruebas registrar un plugin de prueba. La lista de plugins compilados DEBE ensamblarse sin que `internal/integrations` importe el paquete del adaptador (sin ciclo de importaciones), y una prueba de importaciones DEBE protegerlo. No existe carga dinámica de código.
(Previously: `DefaultRegistry()` devolvía un registro vacío y el catálogo por defecto no admitía ningún identificador; el primer adaptador real era futuro)

#### Scenario: Identificador duplicado
- **DADO** dos plugins con el mismo identificador
- **CUANDO** se llama a `NewRegistry` con ambos
- **ENTONCES** devuelve un error tipado de duplicado y ningún registro

#### Scenario: Plugin ausente del catálogo o distinto de él
- **DADO** un plugin válido cuyo manifiesto no está en el catálogo, o está con contenido distinto
- **CUANDO** se llama a `NewRegistry`
- **ENTONCES** devuelve un error tipado y ningún registro

#### Scenario: Registro por defecto con `azure-devops`
- **CUANDO** se llama a `DefaultRegistry()`
- **ENTONCES** el registro contiene exactamente el plugin `azure-devops` y ningún otro
- **Y** el catálogo de manifiestos por defecto admite exactamente el identificador `azure-devops` bajo `integrations:`

#### Scenario: Identificador fuera del catálogo por defecto
- **DADO** un `axiom.yaml` con `integrations.jira`
- **CUANDO** se carga el workspace con el catálogo por defecto
- **ENTONCES** la carga falla con un error tipado que nombra `azure-devops` como único plugin admitido

#### Scenario: Costura de prueba
- **DADO** una prueba que registra el doble en memoria mediante la costura del catálogo
- **CUANDO** se valida `integrations.<doble>` en un `axiom.yaml`
- **ENTONCES** el bloque se acepta
- **Y** al terminar la prueba el catálogo vuelve a su estado por defecto, que contiene `azure-devops`

#### Scenario: Sin ciclo de importaciones
- **DADO** los paquetes `internal/integrations` e `internal/integrations/azuredevops`
- **CUANDO** se comprueban las importaciones de `internal/integrations`, directas y transitivas
- **ENTONCES** no incluyen `internal/integrations/azuredevops`

### Requirement: Coste Cero sin Integración Configurada (RF-INT-05)

Cuando `axiom.yaml` no declara el bloque `integrations`, lo declara vacío (`integrations: {}`), o ningún plugin activo (`enabled` distinto de `false`) escucha el evento, el despachador DEBE volver sin leer ni escribir ningún fichero (ni `integrations.yaml`, ni `.integrations-outbox.yaml`, ni el cerrojo asociado, ni el directorio `integrations` del git common dir), sin resolver la identidad git para el evento, sin construir el evento, sin tocar el almacén de secretos y sin invocar a ningún plugin, y NO DEBE emitir ningún aviso. La salida (stdout, stderr y código) de los verbos que despachan eventos DEBE ser idéntica a la de antes de este cambio y los goldens existentes NO DEBEN cambiar.

#### Scenario: Sin bloque `integrations`
- **DADO** un workspace cuyo `axiom.yaml` no tiene bloque `integrations`
- **CUANDO** se ejecuta `axiom sdd gate record --decision approved` sobre `spec`
- **ENTONCES** la carpeta del cambio no contiene `integrations.yaml` ni `.integrations-outbox.yaml`
- **Y** stdout, stderr y código de salida son los de antes del cambio

#### Scenario: Plugin desactivado
- **DADO** el doble configurado con `enabled: false`
- **CUANDO** se despacha cualquier evento
- **ENTONCES** no se crea ningún fichero, el doble no recibe nada y no se emite aviso

#### Scenario: Bloque declarado sin plugins
- **DADO** un `axiom.yaml` con `integrations: {}`
- **CUANDO** se despacha cualquier evento
- **ENTONCES** no se crea ningún fichero y no se emite aviso

#### Scenario: El almacén de secretos no se consulta
- **DADO** un workspace sin plugin configurado y un almacén de secretos sustituible que falla la prueba si se invoca
- **CUANDO** se ejecutan los verbos que despachan eventos
- **ENTONCES** el almacén no se invoca

### Requirement: Despacho Duradero con Intento Inmediato Acotado (RF-INT-06)

Con al menos un plugin configurado cuyo manifiesto escucha el evento, el despachador DEBE, en este orden: (1) encolar la operación de forma duradera en `.integrations-outbox.yaml` con su clave de idempotencia, (2) intentar la entrega inmediata al plugin con un tiempo máximo acotado (el valor lo fija el diseño), (3) si la entrega tiene éxito, persistir en `integrations.yaml` los vínculos que devuelva el plugin (salvo para `archive`, que nunca los persiste, RF-INT-21) y retirar la operación de la outbox, y (4) si la entrega falla, agota el tiempo o devuelve un error, dejar la operación pendiente con el número de intentos y el último error (sin secretos) para `axiom integrations sync`. Un plugin cuyo manifiesto no escucha el evento NO DEBE recibirlo ni generar operación. El plugin pide el secreto al almacén en el momento de entregar, y solo los de su manifiesto; el despachador NO DEBE leer secretos ni incluirlos en la operación. Un error que el plugin marca como permanente, o alcanzar el máximo de intentos, abandona la operación (RF-INT-08); cualquier otro error, incluido un pánico del plugin, es transitorio y no hace fallar el proceso. El plazo del intento inmediato se comparte entre los plugins del evento y un plugin que no respeta el plazo no bloquea el verbo más allá de él.

#### Scenario: Entrega inmediata con éxito
- **DADO** el doble en memoria configurado, que devuelve el vínculo `{item: "42"}` para el evento
- **CUANDO** se despacha `gate.spec.approved` del cambio `x`
- **ENTONCES** el doble recibe el evento exactamente una vez
- **Y** `integrations.yaml` de `x` contiene el vínculo devuelto
- **Y** no queda ninguna operación pendiente (la outbox se borra al quedar vacía, RF-INT-08)

#### Scenario: Entrega fallida deja la operación pendiente
- **DADO** el doble configurado para fallar
- **CUANDO** se despacha `gate.design.approved`
- **ENTONCES** `.integrations-outbox.yaml` contiene una operación con su clave, `attempts: 1` y el último error
- **Y** `integrations.yaml` no cambia

#### Scenario: Plugin que no escucha el evento
- **DADO** un plugin cuyo manifiesto escucha solo `archive`
- **CUANDO** se despacha `gate.spec.approved`
- **ENTONCES** el plugin no recibe nada y no se crea ninguna operación

#### Scenario: Error permanente abandona la operación
- **DADO** un plugin que devuelve un error marcado como permanente
- **CUANDO** se despacha un evento
- **ENTONCES** la operación queda en la cola con estado `abandoned` tras el primer intento

#### Scenario: Pánico del plugin
- **DADO** un plugin que entra en pánico al entregar
- **CUANDO** se despacha un evento
- **ENTONCES** el verbo de origen termina con normalidad y la operación queda pendiente con el error como transitorio

#### Scenario: Entrega que excede el tiempo acotado
- **DADO** un plugin que no responde dentro del tiempo máximo
- **CUANDO** se despacha un evento
- **ENTONCES** el despacho vuelve tras el tiempo acotado
- **Y** la operación queda pendiente con el error de tiempo agotado

### Requirement: Clave de Idempotencia Estable por Operación (RF-INT-07)

Cada operación DEBE llevar una clave de idempotencia estable, incluida en el `Event` entregado al plugin. Para `gate.spec.approved`, `gate.design.approved`, `role.apply.approved` y `archive` la clave DEBE tener el formato `<evento>/<cambio>[/<rol>]/<huella>`, donde la huella se deriva de la aprobación registrada en el ledger (digest u ordinal; la define el diseño) de modo que reaprobar un artefacto cambiado produce una clave nueva y reintentar la misma aprobación no. Para `role.start` la clave DEBE ser exactamente `role.start/<cambio>/<rol>`. Para `archive` la clave DEBE ser `archive/<cambio>/<carpeta archivada>`, donde la carpeta archivada es `<fecha>-<cambio>`. La operación encolada se identifica por el par (plugin, clave): encolar una operación cuyo par ya está pendiente en la outbox NO DEBE crear una operación duplicada. La entrega es **al menos una vez** con clave estable: tras una entrega con éxito la operación sale de la cola y no queda registro local de claves entregadas, de modo que una caída entre la entrega y el retiro, o una repetición del verbo tras la entrega, PUEDE reentregar la misma clave. El contrato del puerto exige por ello que el plugin sea idempotente por clave.

#### Scenario: Reintento de la misma aprobación
- **DADO** una operación pendiente con la clave `gate.spec.approved/x/<huella>`
- **CUANDO** se vuelve a despachar el mismo evento con la misma aprobación
- **ENTONCES** la outbox sigue teniendo una sola operación con esa clave

#### Scenario: Reaprobar un artefacto cambiado
- **DADO** una operación ya entregada para la aprobación de `spec` con una huella H1
- **CUANDO** el artefacto cambia y se vuelve a aprobar `spec`, con huella H2
- **ENTONCES** se genera una operación nueva con la clave de huella H2

#### Scenario: Clave de `role.start`
- **DADO** el cambio `x` y el rol `backend`
- **CUANDO** se construye el evento `role.start`
- **ENTONCES** su clave de idempotencia es exactamente `role.start/x/backend`

#### Scenario: Clave de `archive`
- **DADO** el cambio `x` archivado en `openspec/changes/archive/2026-11-19-x/`
- **CUANDO** se construye el evento `archive`
- **ENTONCES** su clave de idempotencia es exactamente `archive/x/2026-11-19-x`

#### Scenario: Entrega al menos una vez
- **DADO** una operación entregada con éxito cuyo retiro de la cola no llegó a escribirse por una caída
- **CUANDO** se ejecuta `integrations sync`
- **ENTONCES** el plugin recibe de nuevo la misma clave y la operación se retira al entregarse

### Requirement: Outbox Local Persistente con Reintento (RF-INT-08)

Las operaciones pendientes DEBEN persistirse en `.integrations-outbox.yaml` en la raíz de la carpeta del cambio, con esquema versionado (`axiom.integrations-outbox/v1`). El fichero es estado local: su nombre empieza por punto, `docpaths.LocalState` lo excluye de la publicación y `axiom sdd publish` NUNCA lo confirma. Cada operación DEBE registrar el evento, la clave, los intentos y el último error, y NUNCA un secreto. La escritura DEBE hacerse con cerrojo, fichero temporal en la misma carpeta y reemplazo atómico, bajo la guarda de contexto que corresponda, de modo que una escritura interrumpida no deje el fichero a medias. El cerrojo NO DEBE vivir en la carpeta del cambio sino en el directorio común de git (compartido por todos los worktrees), para que una cola vacía deje el árbol limpio. El fichero DEBE borrarse cuando quede sin operaciones. Política de reintento: el intento inmediato tiene un plazo total de 10 s por despacho, `sync` 30 s por operación y las operaciones pasan a `abandoned` tras 10 intentos o con un error permanente. Este requisito NO modifica RF-BRA-09 ni RF-BRA-18: mientras la outbox tenga operaciones pendientes o abandonadas, que `done` o `ensure` en `in_place` la traten como árbol sucio es aceptable y protege la cola.

#### Scenario: El fichero se borra al vaciarse
- **DADO** una outbox con una sola operación pendiente
- **CUANDO** `axiom integrations sync` la entrega con éxito
- **ENTONCES** `.integrations-outbox.yaml` ya no existe

#### Scenario: La outbox no se publica
- **DADO** una carpeta de cambio con `.integrations-outbox.yaml` pendiente
- **CUANDO** se ejecuta `axiom sdd publish --change x`
- **ENTONCES** el commit de publicación no incluye `.integrations-outbox.yaml`

#### Scenario: Escritura atómica
- **DADO** una outbox existente
- **CUANDO** una escritura se interrumpe después de crear el temporal y antes del reemplazo
- **ENTONCES** `.integrations-outbox.yaml` conserva el contenido anterior íntegro

#### Scenario: Operaciones concurrentes
- **DADO** dos despachos simultáneos sobre el mismo cambio con claves distintas
- **CUANDO** ambos encolan
- **ENTONCES** la outbox contiene ambas operaciones y es YAML válido

#### Scenario: Operación abandonada tras el máximo de intentos
- **DADO** una operación que alcanza el máximo de intentos que fija el diseño
- **CUANDO** `sync` la reintenta de nuevo
- **ENTONCES** queda marcada como abandonada con su último error, sin reintentarse automáticamente, y `sync` sigue terminando con código distinto de cero mientras exista

#### Scenario: El cerrojo no ensucia el árbol
- **DADO** un despacho que entrega con éxito y deja la cola vacía
- **CUANDO** se ejecuta `git status` en la carpeta del cambio
- **ENTONCES** no aparece ningún fichero de cerrojo ni de cola sin seguimiento

### Requirement: `integrations.yaml` Versionado con Vínculos e Imputaciones (RF-INT-09)

La carpeta del cambio PUEDE contener `integrations.yaml`, con esquema versionado (`axiom.integrations/v2`; los binarios nuevos leen también `axiom.integrations/v1`), versionado y publicado por `axiom sdd publish` como cualquier documento de la raíz del cambio. SOLO DEBE contener vínculos (cambio, rol u hito con elemento externo) e imputaciones; NUNCA operaciones pendientes, intentos, errores ni secretos. Todo fichero que un binario nuevo escribe DEBE llevar `axiom.integrations/v2`; un fichero `v1` solo se reescribe como `v2` cuando una fusión lo modifica, y una fusión que no cambia nada NO DEBE reescribir el fichero. Un esquema desconocido DEBE rechazarse con un error claro que nombra el esquema leído, no con un error de campo desconocido. Un vínculo se identifica por (plugin, tipo, id, rol) y una imputación por (plugin, origen, elemento): repetir la fusión no cambia el fichero. El vínculo `kind: task` DEBE llevar además `role`, `source` (`tasks.create` o `tasks.bind`), `estimate_hours` (número positivo, la estimación confirmada, opcional solo cuando se vinculó a mano sin estimación) y `started_at` (instante RFC 3339 de la primera entrada del rol, que una vez registrado NO DEBE sobrescribirse). Fusionar un vínculo de la misma identidad con otro `estimate_hours` DEBE actualizar el valor. La imputación DEBE llevar `plugin`, `role`, `item_id`, `minutes` (entero positivo), `date`, `actor` (la identidad git de la persona), `source` y `recorded_at`; el `source` de una imputación de cierre de rol tiene la forma `role.close/<rol>/<n>` y es único por cierre, de modo que un rol rechazado y aprobado de nuevo puede imputar dos veces sin chocar con la identidad `(plugin, origen, elemento)`. Un enlace cuya URL lleve credenciales (`userinfo`) o no sea http(s) DEBE rechazarse como error permanente de la operación, para que ninguna credencial llegue a disco versionado. La escritura DEBE seguir el mismo patrón que RF-INT-08. El fichero DEBE crearse únicamente cuando un plugin devuelve vínculos o imputaciones, cuando `axiom change start --adopt` vincula el work item del cambio o cuando los verbos `tasks` y `time` registran un vínculo de Task o una imputación; sin ellos NO DEBE existir. El vínculo del work item del cambio DEBE tener `kind: work_item`, `plugin`, `id`, `url` y `source: change.start`, se escribe por una función de vinculación exportada distinta del despacho de entregas, que reutiliza la fusión de vínculos bajo el mismo cerrojo que las entregas; el carril ODD NO escribe `integrations.yaml`.

(Previously: esquema `axiom.integrations/v1`, sin vínculo `kind: task` con estimación ni inicio, imputación sin `actor` y fichero creado solo por entregas o por `change start --adopt`)

#### Scenario: Los vínculos devueltos se persisten
- **DADO** un plugin que devuelve el vínculo `{role: backend, item: "42"}`
- **CUANDO** la entrega tiene éxito
- **ENTONCES** `integrations.yaml` contiene ese vínculo y el campo de esquema `axiom.integrations/v2`

#### Scenario: Persistir el mismo vínculo dos veces
- **DADO** un vínculo ya registrado
- **CUANDO** el plugin lo devuelve de nuevo
- **ENTONCES** `integrations.yaml` no cambia (sin duplicado)

#### Scenario: URL con credenciales
- **DADO** un plugin que devuelve un vínculo con la URL `https://user:pw@tracker.example/1`
- **CUANDO** se intenta fusionar
- **ENTONCES** la operación se abandona como error permanente y `integrations.yaml` no se escribe

#### Scenario: Sin vínculos no hay fichero
- **DADO** un plugin que entrega con éxito sin devolver vínculos
- **CUANDO** se despacha un evento
- **ENTONCES** `integrations.yaml` no se crea

#### Scenario: Ningún estado de entrega en el fichero versionado
- **DADO** una entrega fallida con operación pendiente
- **CUANDO** se inspecciona `integrations.yaml`
- **ENTONCES** no contiene claves de intentos, errores ni operaciones

#### Scenario: El vínculo del work item se escribe con `source: change.start`
- **DADO** un cambio sin `integrations.yaml` y una adopción de `axiom change start`
- **CUANDO** se vincula el work item 2525 del plugin `azure-devops`
- **ENTONCES** `integrations.yaml` se crea con el esquema `axiom.integrations/v2` y un vínculo `kind: work_item`, `plugin: azure-devops`, `id: "2525"`, la URL y `source: change.start`

#### Scenario: Vincular dos veces el mismo work item
- **DADO** el vínculo del work item ya registrado
- **CUANDO** se vincula de nuevo
- **ENTONCES** `integrations.yaml` no cambia

#### Scenario: El vínculo convive con los de las entregas
- **DADO** un `integrations.yaml` con vínculos de una entrega
- **CUANDO** se vincula el work item
- **ENTONCES** se conservan los vínculos previos y se añade el del work item bajo el mismo cerrojo

#### Scenario: URL del work item con credenciales
- **DADO** un work item cuya URL lleva `userinfo`
- **CUANDO** se intenta vincular
- **ENTONCES** la vinculación se rechaza y `integrations.yaml` no se escribe

#### Scenario: Lectura de un fichero `v1`
- **DADO** un `integrations.yaml` con `axiom.integrations/v1` escrito por un binario anterior
- **CUANDO** un binario nuevo lo lee
- **ENTONCES** lo acepta y conserva sus vínculos e imputaciones
- **Y** no lo reescribe mientras la fusión no cambie nada

#### Scenario: Esquema desconocido
- **DADO** un `integrations.yaml` con `axiom.integrations/v9`
- **CUANDO** se lee
- **ENTONCES** se rechaza con un error que nombra el esquema `axiom.integrations/v9`

#### Scenario: Vínculo de Task con estimación e inicio
- **DADO** un vínculo `kind: task` del rol `fullstack` con `estimate_hours: 6`
- **CUANDO** el rol entra por primera vez y se registra `started_at`
- **ENTONCES** el vínculo conserva `estimate_hours: 6` y guarda `started_at`
- **Y** un registro posterior de `started_at` no lo sobrescribe

#### Scenario: Actualizar la estimación de un vínculo
- **DADO** un vínculo `kind: task` con `estimate_hours: 6`
- **CUANDO** se fusiona el mismo vínculo con `estimate_hours: 4`
- **ENTONCES** el vínculo queda con `estimate_hours: 4` y no se duplica

#### Scenario: Dos imputaciones del mismo rol no chocan
- **DADO** las imputaciones con `source: role.close/fullstack/1` y `source: role.close/fullstack/2` sobre la misma Task
- **CUANDO** se fusionan
- **ENTONCES** ambas se conservan, porque su identidad `(plugin, origen, elemento)` difiere

#### Scenario: Imputación con actor
- **DADO** una imputación registrada por `time record`
- **CUANDO** se lee `integrations.yaml`
- **ENTONCES** la imputación contiene `actor`, `minutes`, `date`, `role`, `item_id` y `source`

### Requirement: La Emisión Nunca Hace Fallar el Verbo de Origen (RF-INT-10)

Cuando un verbo determinista despacha un evento (RF-GAT-17, RF-BRA-34 y RF-PUB-09), cualquier fallo de la integración (entrega, escritura de la outbox o de `integrations.yaml`, secreto no disponible) DEBE avisarse por stderr mediante un `io.Writer` sustituible, SIN cambiar el código de salida ni el contenido de stdout del verbo. Antes de despachar, el adaptador DEBE comprobar explícitamente `Declarations.IntegrationsErr` (porque `Locate` no rechaza declaraciones inválidas): si hay error, DEBE avisar por stderr y NO despachar.

#### Scenario: Fallo del plugin no rompe el verbo
- **DADO** el doble configurado para fallar
- **CUANDO** se ejecuta `axiom sdd gate record --decision approved` sobre `design`
- **ENTONCES** el verbo termina con código 0 y stdout igual al de antes del cambio
- **Y** stderr contiene un aviso que nombra el plugin y la operación pendiente
- **Y** la aprobación consta en `gates.yaml`

#### Scenario: Bloque `integrations` inválido
- **DADO** un `axiom.yaml` con una clave desconocida en `integrations.<plugin>`
- **CUANDO** se ejecuta un verbo que despacha un evento
- **ENTONCES** el verbo termina con su código habitual
- **Y** stderr avisa del bloque inválido
- **Y** no se invoca a ningún plugin ni se crea ningún fichero

#### Scenario: Fallo de escritura de la outbox
- **DADO** una carpeta de cambio cuya outbox no se puede escribir
- **CUANDO** se despacha un evento
- **ENTONCES** el verbo de origen termina con éxito y stderr avisa del fallo

### Requirement: `axiom integrations sync` Drena la Outbox (RF-INT-11)

`axiom integrations sync [--change <nombre>]` (con `--cwd`) DEBE reintentar, en orden de encolado, todas las operaciones pendientes de la outbox y retirar las entregadas, persistiendo los vínculos devueltos en la carpeta ordinaria (RF-INT-09). DEBE cubrir también las carpetas archivadas (RF-INT-20). Con `--change` trata la carpeta ordinaria y la carpeta archivada más reciente del cambio, si existen; sin `--change` recorre las carpetas ordinarias y las archivadas que tengan cola. `--retry-abandoned` reactiva las operaciones abandonadas (con los intentos a 0) y `--discard <clave>` (repetible) retira una operación concreta. DEBE terminar con código 0 solo si no queda ninguna operación pendiente o abandonada, y con código distinto de cero (y un resumen por stderr, sin secretos) si queda alguna. Sin ninguna cola existente (con o sin plugin configurado), DEBE terminar con código 0 sin crear ningún fichero. Con una cola existente cuyo plugin ya no está configurado o activo, DEBE dejar esas operaciones pendientes con el motivo `plugin_not_configured`, sin contar un intento, y terminar con código 1. Un bloque `integrations` inválido DEBE ser un error (código 1), no un aviso, porque `sync` es un verbo explícito de integración. Un fallo en una operación NO DEBE impedir reintentar las siguientes. La salida JSON es `axiom.integrations-sync/v1` con, por cambio, las operaciones entregadas, pendientes, abandonadas y descartadas (cada una con clave, plugin, intentos, último error saneado y motivo) y el total restante.

#### Scenario: Reintento con éxito
- **DADO** dos operaciones pendientes y el doble ya sin fallar
- **CUANDO** se ejecuta `axiom integrations sync --change x`
- **ENTONCES** el doble recibe ambas, en orden, con sus claves
- **Y** `.integrations-outbox.yaml` se borra
- **Y** el código de salida es 0

#### Scenario: Queda algo pendiente
- **DADO** dos operaciones pendientes y el doble que falla la primera y acepta la segunda
- **CUANDO** se ejecuta `sync`
- **ENTONCES** la segunda se entrega y se retira, la primera sigue pendiente con un intento más
- **Y** el código de salida es distinto de cero

#### Scenario: Descartar una operación
- **DADO** una operación abandonada con la clave K
- **CUANDO** se ejecuta `sync --discard K`
- **ENTONCES** la operación se retira de la cola sin entregarse

#### Scenario: Sin integración y sin cola no hay efecto
- **DADO** un workspace sin bloque `integrations` y sin ninguna cola
- **CUANDO** se ejecuta `axiom integrations sync --change x`
- **ENTONCES** termina con código 0 y no crea ningún fichero

#### Scenario: Cola existente sin plugin configurado
- **DADO** una operación pendiente cuyo plugin ya no figura en `axiom.yaml`
- **CUANDO** se ejecuta `sync`
- **ENTONCES** la operación sigue pendiente con el motivo `plugin_not_configured` y los mismos intentos
- **Y** el código de salida es 1

#### Scenario: Bloque inválido en `sync`
- **DADO** un `axiom.yaml` con una clave desconocida en `integrations.<plugin>`
- **CUANDO** se ejecuta `sync`
- **ENTONCES** termina con error y código 1 sin entregar nada

### Requirement: `axiom integrations emit` Dispara un Evento del Catálogo (RF-INT-12)

`axiom integrations emit --change <nombre> --event <evento>` (con `--cwd`; `--role <id>` para los eventos de rol) DEBE despachar un evento del catálogo con las reglas de RF-INT-05 a RF-INT-08, como respaldo explícito de los verbos deterministas. DEBE rechazar con error tipado y código distinto de cero: `change.start` (lo dispara C-07-d), un evento fuera del catálogo, un evento de rol sin `--role` o con un rol ausente del kickoff sellado, y un cambio inexistente. `emit` NO DEBE fabricar aprobaciones: los eventos de compuerta (`gate.spec.approved`, `gate.design.approved`, `role.apply.approved`) exigen que la última decisión de esa compuerta en `gates.yaml` sea `approved`, y toman de ese registro el digest, la huella y, si no se da `--actor`, el actor. `role.apply.approved` y `role.start` exigen `--role` del roster sellado. `emit` termina con código 0 si se entregó todo o si ningún plugin activo escucha el evento (con un mensaje que lo dice), y con código 1 si algo queda pendiente o abandonado; un bloque `integrations` inválido es un error (código 1). La salida JSON es `axiom.integrations-emit/v1` con la clave, las operaciones entregadas, pendientes y abandonadas y un indicador de que no había nada que hacer. `emit --event archive` DEBE exigir que exista la carpeta archivada del cambio y operar sobre ella (RF-INT-20). A diferencia de RF-INT-10, un fallo de entrega en `emit` deja la operación pendiente y el verbo termina con código distinto de cero (la persona lo invoca expresamente).

#### Scenario: Emisión manual con éxito
- **DADO** el doble configurado
- **CUANDO** se ejecuta `axiom integrations emit --change x --event gate.spec.approved`
- **ENTONCES** el doble recibe el evento y el código de salida es 0

#### Scenario: `change.start` rechazado
- **CUANDO** se ejecuta `axiom integrations emit --change x --event change.start`
- **ENTONCES** termina con código distinto de cero y un error tipado que indica que lo emite C-07-d
- **Y** no se crea ningún fichero

#### Scenario: `emit` no fabrica aprobaciones
- **DADO** que la última decisión de `spec` en `gates.yaml` es `rejected`, o que no hay ninguna
- **CUANDO** se ejecuta `emit --change x --event gate.spec.approved`
- **ENTONCES** se rechaza con error tipado y código distinto de cero
- **Y** el doble no recibe nada y no se crea ningún fichero

#### Scenario: `emit` toma digest, huella y actor del ledger
- **DADO** `spec` aprobada en `gates.yaml` con un digest D y actor A
- **CUANDO** se ejecuta `emit --event gate.spec.approved` sin `--actor`
- **ENTONCES** el evento entregado lleva el digest D, su huella y el actor A, y la misma clave que la emisión de `gate record`

#### Scenario: Ningún plugin escucha el evento
- **DADO** un plugin cuyo manifiesto escucha solo `archive`
- **CUANDO** se ejecuta `emit --event gate.spec.approved`
- **ENTONCES** termina con código 0 y un mensaje que indica que no había nada que hacer

#### Scenario: Evento de rol sin rol
- **CUANDO** se ejecuta `axiom integrations emit --change x --event role.start` sin `--role`
- **ENTONCES** termina con código distinto de cero y un error tipado, sin crear ningún fichero

#### Scenario: Rol ausente del kickoff
- **DADO** un kickoff sellado con el rol `backend`
- **CUANDO** se ejecuta `emit --event role.start --role frontend`
- **ENTONCES** se rechaza con error tipado y no se crea ningún fichero

#### Scenario: Fallo de entrega en `emit`
- **DADO** el doble configurado para fallar
- **CUANDO** se ejecuta `emit --event archive`
- **ENTONCES** la operación queda pendiente en la outbox
- **Y** el verbo termina con código distinto de cero

### Requirement: Almacén de Secretos sin Caída Silenciosa a Texto Plano (RF-INT-13)

El paquete `internal/integrations/secrets` DEBE definir el puerto `SecretStore` con un adaptador de entorno, un adaptador de keyring del sistema (`github.com/zalando/go-keyring`: Windows Credential Manager y llavero de macOS) y una cadena de resolución entorno → keyring. El adaptador de keyring se compila SOLO para Windows y macOS (`//go:build windows || darwin`): en Linux y en cualquier otro sistema no hay Secret Service y el único almacén es la variable de entorno. El adaptador de entorno lee `AXIOM_INTEGRATION_<PLUGIN>_<CLAVE>`, con el identificador del plugin en mayúsculas y `-` sustituido por `_` y el nombre del secreto en mayúsculas (`azure-devops` y `pat` dan `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`); una variable vacía cuenta como no definida y el catálogo rechaza dos pares que den la misma variable. El keyring usa el servicio `axiom` y la cuenta `<plugin>/<secreto>`. Un secreto ausente DEBE dar un error tipado que nombre `axiom integrations secret set` y la variable. NUNCA DEBE existir un almacén en texto plano ni una caída a él: si el keyring no está disponible y el entorno no aporta valor, DEBE devolverse un error tipado que nombre la variable de entorno alternativa. Los errores NUNCA DEBEN envolver el valor del secreto. Los secretos solo DEBEN declararse por nombre en el manifiesto (RF-INT-03).

#### Scenario: El entorno prevalece
- **DADO** la variable de entorno definida y un valor distinto en el keyring
- **CUANDO** se resuelve el secreto
- **ENTONCES** se obtiene el valor del entorno y la fuente es `env`

#### Scenario: Resolución desde el keyring
- **DADO** un keyring simulado con el secreto y sin variable de entorno
- **CUANDO** se resuelve
- **ENTONCES** se obtiene el valor y la fuente es `keyring`

#### Scenario: Linux solo tiene el entorno
- **DADO** un sistema que no es Windows ni macOS
- **CUANDO** se resuelve un secreto sin variable de entorno
- **ENTONCES** el error es el de secreto ausente que nombra la variable y la fuente es `missing`, y nunca se consulta un servicio de secretos ni se crea un fichero

#### Scenario: Keyring no disponible
- **DADO** un keyring no disponible y sin variable de entorno
- **CUANDO** se resuelve el secreto
- **ENTONCES** se devuelve un error tipado que nombra la variable de entorno `AXIOM_INTEGRATION_<PLUGIN>_<CLAVE>` correspondiente
- **Y** no se crea ningún fichero con el valor ni se usa ningún almacén alternativo

#### Scenario: Secreto inexistente
- **DADO** un keyring disponible sin el secreto y sin variable de entorno
- **CUANDO** se resuelve
- **ENTONCES** se devuelve un error tipado de secreto ausente que no incluye ningún valor

### Requirement: Tipo `Secret` Redactado (RF-INT-14)

El tipo `Secret` DEBE redactar su valor en `String`, `GoString`, `MarshalJSON` y `MarshalYAML` (y en cualquier formateo con `fmt`, incluidos `%v`, `%+v` y `%#v`). El valor solo DEBE obtenerse mediante un método explícito que usa el plugin en el momento de entregar.

#### Scenario: Formateo redactado
- **DADO** un `Secret` con valor distintivo
- **CUANDO** se formatea con `%s`, `%v`, `%+v` y `%#v` y se serializa a JSON y a YAML
- **ENTONCES** ninguna salida contiene el valor y todas contienen el marcador de redacción

#### Scenario: Error que incluye el secreto
- **DADO** un plugin cuyo error de entrega se construye a partir de un contexto que contiene un `Secret`
- **CUANDO** el error llega a la outbox y a stderr
- **ENTONCES** no contiene el valor del secreto

### Requirement: `axiom integrations secret set|status|delete` (RF-INT-15)

`axiom integrations secret set|status|delete --plugin <id> --name <secreto>` DEBE gestionar los secretos en el keyring del sistema (Windows y macOS). En Linux y en cualquier otro sistema `set` y `delete` DEBEN negarse con un error tipado que explique que el único almacén es la variable de entorno `AXIOM_INTEGRATION_<PLUGIN>_<SECRETO>` y NUNCA escribir el valor en un fichero; `status` informa `env` o `missing`. `set` DEBE leer el valor únicamente por stdin (máximo 4 KiB, quitando un único salto de línea final y rechazando el valor vacío) y NUNCA aceptarlo como argumento ni como bandera; si stdin es un terminal DEBE negarse y sugerir canalizar el valor, porque no hay prompt sin eco en este cambio. `status` (con `--name` opcional) DEBE informar, por cada secreto del manifiesto, de su `source`, con exactamente uno de los valores `env`, `keyring`, `missing` o `keyring_unavailable`, y NUNCA el valor ni su longitud, tampoco con `--json` (`axiom.integrations-secret-status/v1`). `delete` elimina el secreto del keyring, es idempotente (sin secreto informa y sale con 0) y recuerda que una variable de entorno no se borra desde Axiom. Un `--plugin` o `--name` que el manifiesto no declare DEBE rechazarse con error tipado. Ninguna salida, error ni registro DEBE contener el valor.

#### Scenario: `set` por stdin
- **DADO** el doble con un secreto declarado
- **CUANDO** se ejecuta `secret set --plugin <doble> --name token` con el valor por stdin
- **ENTONCES** el valor queda en el keyring simulado
- **Y** stdout y stderr no contienen el valor

#### Scenario: El valor por argumento se rechaza
- **CUANDO** se ejecuta `secret set --plugin <doble> --name token --value abc`
- **ENTONCES** el verbo rechaza `--value` como opción desconocida y no escribe nada en el keyring

#### Scenario: `set` en Linux se niega
- **DADO** un sistema que no es Windows ni macOS
- **CUANDO** se ejecuta `secret set --plugin <doble> --name token` con el valor por stdin
- **ENTONCES** el verbo sale con 1 con un error tipado que nombra la variable `AXIOM_INTEGRATION_<PLUGIN>_<SECRETO>`
- **Y** no se escribe ningún fichero ni se imprime el valor

#### Scenario: `status` sin revelar el valor
- **DADO** un secreto definido por entorno
- **CUANDO** se ejecuta `secret status --plugin <doble> --name token --json`
- **ENTONCES** la salida indica `source: env` para ese secreto
- **Y** no contiene el valor ni su longitud

#### Scenario: Los cuatro valores de `source`
- **DADO** cuatro secretos del manifiesto: uno por entorno, uno en el keyring, uno inexistente y uno con el keyring no disponible
- **CUANDO** se ejecuta `secret status --plugin <doble> --json`
- **ENTONCES** sus `source` son `env`, `keyring`, `missing` y `keyring_unavailable`

#### Scenario: `set` desde un terminal
- **DADO** stdin conectado a un terminal
- **CUANDO** se ejecuta `secret set --plugin <doble> --name token`
- **ENTONCES** se niega con un mensaje que sugiere canalizar el valor y no escribe nada en el keyring

#### Scenario: `set` con valor vacío o demasiado grande
- **DADO** un valor vacío, o de más de 4 KiB, por stdin
- **CUANDO** se ejecuta `secret set`
- **ENTONCES** se rechaza sin escribir nada en el keyring

#### Scenario: `delete` idempotente
- **DADO** un secreto inexistente
- **CUANDO** se ejecuta `secret delete`
- **ENTONCES** termina con código 0 e informa de que no había secreto
- **Y** recuerda que una variable de entorno no se borra desde Axiom

#### Scenario: Secreto no declarado
- **CUANDO** se ejecuta `secret set --plugin <doble> --name otro`
- **ENTONCES** se rechaza con error tipado que lista los secretos declarados por el manifiesto

### Requirement: Ningún Secreto Llega a Disco Versionado, a la Salida ni a un GET (RF-INT-16)

Un valor de secreto NO DEBE aparecer, tras ningún flujo del marco, en `axiom.yaml`, en ningún fichero de la carpeta del cambio (incluidos `integrations.yaml` y `.integrations-outbox.yaml`), en stdout ni en stderr de ningún verbo, ni en la respuesta de ningún endpoint GET del dashboard. Una prueba canario DEBE verificarlo: con un valor distintivo en el almacén simulado, ejecuta los flujos de despacho, fallo, reintento, `secret set|status|delete` y consulta los GET del dashboard, y comprueba la ausencia del valor en todos esos destinos.

#### Scenario: Canario tras un flujo completo
- **DADO** un valor distintivo en el almacén de secretos simulado y el doble que lo lee al entregar
- **CUANDO** se recorren `gate record`, una entrega fallida, `sync` y `secret status`
- **ENTONCES** el valor no aparece en `axiom.yaml`, en la carpeta del cambio, en stdout ni en stderr

#### Scenario: Canario en los GET del dashboard
- **DADO** la misma configuración
- **CUANDO** se consultan los endpoints GET del dashboard
- **ENTONCES** ninguna respuesta contiene el valor

### Requirement: Doble de Prueba en Memoria y Criterio de Terminado (RF-INT-17)

El paquete `internal/integrations/integrationstest` DEBE ofrecer un plugin en memoria que registra los eventos recibidos (con su clave), devuelve vínculos configurables y puede fallar a voluntad (siempre, N veces o por evento). Debe poder inyectarse en el catálogo de manifiestos y en el registro mediante las costuras de prueba. Una prueba de extremo a extremo DEBE demostrar el criterio de terminado del backlog: un plugin de prueba registrado y configurado en `axiom.yaml` recibe los eventos de `gate record` (spec, design y role-apply), de `branch ensure --role`, de `sdd publish --archive` y de `integrations emit`; persiste vínculos en `integrations.yaml`; y reintenta desde la outbox con `sync`.

#### Scenario: Recorrido de todos los verbos
- **DADO** un workspace con `integrations.<doble>` configurado y el doble registrado
- **CUANDO** se ejecutan `gate record --decision approved` para `spec`, `design` y `role-apply:<rol>`, `branch ensure --role <rol>`, `sdd publish --archive` y `integrations emit`
- **ENTONCES** el doble ha recibido, con su clave de idempotencia, `gate.spec.approved`, `gate.design.approved`, `role.apply.approved`, `role.start`, `archive` y el evento emitido
- **Y** `integrations.yaml` contiene los vínculos que devolvió (salvo los de `archive`, RF-INT-21)

#### Scenario: Reintento tras fallo
- **DADO** el doble configurado para fallar una vez
- **CUANDO** falla la entrega de un evento y después se ejecuta `integrations sync`
- **ENTONCES** la operación se entrega en el reintento, se retira de la outbox y `sync` termina con código 0

### Requirement: Alta del Verbo `integrations` en la CLI y la Ayuda (RF-INT-18)

La CLI DEBE exponer la familia de primer nivel `axiom integrations` con exactamente los subcomandos `sync`, `emit`, `secret` (este con `set`, `status` y `delete`), `connect`, `discover`, `tasks` (con `create` y `bind`), `summary` (con `push`) y `time` (con `propose` y `record`). La ayuda global y la ayuda de cada subcomando DEBEN describirlos y las pruebas de ayuda DEBEN cubrirlos. Todo subcomando acepta `--cwd`; los que operan sobre un cambio, `--change`.

(Previously: la familia tenía exactamente los subcomandos `sync`, `emit`, `secret`, `connect` y `discover`; `tasks`, `summary` y `time` los añade este cambio)

#### Scenario: Ayuda global
- **CUANDO** se ejecuta `axiom --help`
- **ENTONCES** el texto lista `integrations` con `sync`, `emit`, `secret`, `connect`, `discover`, `tasks`, `summary` y `time`

#### Scenario: Ayuda de los verbos nuevos
- **CUANDO** se ejecuta `axiom integrations tasks --help`, `axiom integrations summary --help` y `axiom integrations time --help`
- **ENTONCES** cada ayuda describe sus subcomandos y sus banderas `--change`, `--cwd` y las propias de cada verbo

#### Scenario: Subcomando desconocido
- **CUANDO** se ejecuta `axiom integrations inspect`
- **ENTONCES** termina con código distinto de cero y muestra la ayuda de la familia, que lista `sync`, `emit`, `secret`, `connect`, `discover`, `tasks`, `summary` y `time`
### Requirement: `archive` se Encola de Forma Duradera en la Carpeta Archivada (RF-INT-19)

El evento `archive` DEBE seguir el mismo despacho duradero que el resto de eventos (RF-INT-06), pero la operación DEBE encolarse en `.integrations-outbox.yaml` en la raíz de la carpeta archivada del cambio (`openspec/changes/archive/<fecha>-<cambio>/`), porque `sdd publish --archive` se ejecuta después de mover el incremento. La cola archivada es estado local (RF-PUB-01 modificado): NO se versiona ni se publica. La escritura de la cola archivada y de sus temporales DEBE pasar por una guarda que autorice, bajo `openspec/changes/archive/`, solo el nombre de la cola y sus temporales en la raíz exacta de una carpeta archivada (RF-ARC-03 modificado). Sin repositorio git, el despacho de `archive` NO DEBE encolar: avisa por stderr e indica la orden `axiom integrations emit --event archive`.

#### Scenario: La operación se encola en la carpeta archivada
- **DADO** el doble configurado para fallar y un cambio `x` archivado en `openspec/changes/archive/2026-11-19-x/`
- **CUANDO** se despacha `archive`
- **ENTONCES** `.integrations-outbox.yaml` existe en la raíz de `2026-11-19-x/` con la clave `archive/x/2026-11-19-x`
- **Y** no se crea ningún fichero en `openspec/changes/x/`

#### Scenario: Cola archivada vacía se borra
- **DADO** una cola archivada con una sola operación
- **CUANDO** la entrega tiene éxito
- **ENTONCES** `.integrations-outbox.yaml` de la carpeta archivada ya no existe

#### Scenario: Otro nombre bajo `archive/` se rechaza
- **DADO** una carpeta archivada
- **CUANDO** el marco intenta escribir en ella `gates.yaml`, `.otro` o un fichero anidado
- **ENTONCES** la escritura se rechaza con el mensaje de RF-ARC-03 y no se modifica nada

#### Scenario: Sin git no se encola
- **DADO** un workspace que no está en un repositorio git y un cambio archivado
- **CUANDO** se despacha `archive`
- **ENTONCES** no se crea ninguna cola y stderr avisa e indica `axiom integrations emit --event archive`

### Requirement: `sync` y `emit --event archive` Cubren las Carpetas Archivadas (RF-INT-20)

`axiom integrations sync` DEBE tratar también las colas de las carpetas archivadas, con la misma política de reintento que las ordinarias (RF-INT-08), aplicando a cada carpeta la guarda de su contexto (ordinario o archivado). `axiom integrations emit --change <x> --event archive` DEBE exigir que exista la carpeta archivada del cambio, rechazar con error tipado y sin crear ficheros si no existe, y encolar y entregar sobre ella como RF-INT-19. Con `--change` y una carpeta ordinaria y una archivada, `sync` DEBE tratar ambas; en la carpeta archivada, `sync` NO DEBE fusionar vínculos (RF-INT-21).

#### Scenario: `sync` reintenta la cola archivada
- **DADO** una operación `archive` pendiente en la carpeta archivada de `x`
- **CUANDO** se ejecuta `axiom integrations sync --change x` con el doble ya sin fallar
- **ENTONCES** el doble recibe `archive/x/<carpeta>`, la operación se retira y la cola archivada se borra
- **Y** el código de salida es 0

#### Scenario: `sync` sin `--change` recorre las archivadas
- **DADO** colas pendientes en una carpeta ordinaria y en una archivada de otro cambio
- **CUANDO** se ejecuta `axiom integrations sync` sin `--change`
- **ENTONCES** ambas se tratan

#### Scenario: `emit --event archive` sin carpeta archivada
- **DADO** un cambio `x` que no está archivado
- **CUANDO** se ejecuta `emit --change x --event archive`
- **ENTONCES** se rechaza con error tipado, con código distinto de cero, sin crear ningún fichero

#### Scenario: `emit --event archive` con carpeta archivada
- **DADO** el doble configurado y `x` archivado
- **CUANDO** se ejecuta `emit --change x --event archive`
- **ENTONCES** el doble recibe `archive/x/<carpeta archivada>`

### Requirement: El Evento `archive` Nunca Fusiona Vínculos en `integrations.yaml` (RF-INT-21)

Ni el despacho inmediato ni `sync` ni `emit` DEBEN persistir en `integrations.yaml` los vínculos o imputaciones que el plugin devuelva al entregar `archive`: se descartan con un aviso por stderr. `integrations.yaml` de la carpeta archivada solo se lee, para dar los vínculos existentes al plugin (`Delivery.Links`). Para los demás eventos, RF-INT-09 no cambia.

#### Scenario: Vínculos devueltos por `archive` se descartan
- **DADO** un doble que devuelve un vínculo al entregar `archive`
- **CUANDO** se despacha, o se reintenta con `sync`
- **ENTONCES** `integrations.yaml` archivado no cambia
- **Y** stderr avisa del descarte

#### Scenario: El plugin recibe los vínculos existentes
- **DADO** un `integrations.yaml` archivado con un vínculo
- **CUANDO** se entrega `archive`
- **ENTONCES** el plugin recibe ese vínculo en `Delivery.Links`

### Requirement: `done` en la Rama de Archivado se Niega con Cola Pendiente (RF-INT-22)

Mientras la cola de la carpeta archivada tenga operaciones pendientes o abandonadas, `axiom branch done` sobre la rama de archivado DEBE negarse por la regla de árbol sucio vigente (RF-BRA-18), que NO cambia: la cola protege las operaciones evitando que se elimine el worktree con trabajo sin entregar. Con la cola vacía (y por tanto borrada) `done` NO DEBE negarse por ella.

#### Scenario: `done` se niega con cola archivada pendiente
- **DADO** una cola `archive` pendiente en la carpeta archivada del worktree de archivado
- **CUANDO** se ejecuta `axiom branch done` sobre la rama de archivado
- **ENTONCES** se rechaza por árbol sucio y no se elimina el worktree

#### Scenario: `done` tras vaciar la cola
- **DADO** la misma cola, entregada con `axiom integrations sync`
- **CUANDO** se ejecuta `axiom branch done`
- **ENTONCES** no se rechaza por la cola
### Requirement: Puertos Opcionales de Identidad y Descubrimiento (RF-INT-23)

El paquete `internal/integrations` DEBE definir dos puertos opcionales que un plugin implementa solo si los ofrece: la resolución de la identidad de la persona y el descubrimiento del proceso del tracker. Se DEBEN detectar por aserción de tipo sobre el `Plugin` registrado. El puerto `Plugin` (`Manifest`, `Deliver`) NO DEBE cambiar. Los verbos genéricos `axiom integrations connect` y `axiom integrations discover` DEBEN despachar por estos puertos. Cuando el plugin indicado con `--plugin` no ofrece el puerto correspondiente, el verbo DEBE terminar con un error tipado que nombra el plugin y el puerto que falta, sin efectos secundarios.

#### Scenario: Plugin que ofrece ambos puertos
- **DADO** un plugin de prueba que implementa los puertos de identidad y de descubrimiento
- **CUANDO** el verbo `connect` y el verbo `discover` lo invocan
- **ENTONCES** cada verbo detecta su puerto por aserción de tipo y lo invoca

#### Scenario: Plugin sin puerto de identidad
- **DADO** un plugin de prueba que solo implementa `Plugin`
- **CUANDO** se ejecuta `axiom integrations connect --plugin <doble>`
- **ENTONCES** termina con código distinto de cero y un error tipado que nombra el plugin y el puerto de identidad
- **Y** no se crea ningún fichero

#### Scenario: Plugin sin puerto de descubrimiento
- **DADO** un plugin de prueba que solo implementa `Plugin`
- **CUANDO** se ejecuta `axiom integrations discover --plugin <doble>`
- **ENTONCES** termina con código distinto de cero y un error tipado que nombra el plugin y el puerto de descubrimiento

#### Scenario: El puerto `Plugin` no cambia
- **DADO** un plugin que implementa solo `Manifest` y `Deliver`
- **CUANDO** se registra con `NewRegistry`
- **ENTONCES** el registro lo acepta sin exigir ningún puerto opcional
### Requirement: Puerto Opcional de Work Items (RF-INT-24)

El marco MUST ofrecer un puerto opcional de work items, detectado por aserción de tipo sobre el plugin (mismo patrón que `IdentityResolver` y `Discoverer`), sin añadir métodos a `Plugin` ni alterar los plugins que no lo implementan. El puerto MUST exponer, con tipos neutrales propios del marco (sin tipos del tracker): (a) la lectura de un work item (identificador, tipo, título, descripción, pasos de reproducción y criterios de aceptación en HTML o texto original, estado y su categoría, URL, etiquetas, padre, relaciones Related y adjuntos con nombre, URL y tamaño declarado); (b) la descarga acotada de un adjunto o imagen por URL; y (c) el enlace de vuelta idempotente por etiqueta. La invocación del puerto MUST pasar por el mismo punto que los demás puertos (`invokePort`): MUST recuperar los pánicos del plugin como error tipado y MUST sanear de secretos todo error devuelto. Si el plugin no implementa el puerto, la invocación MUST devolver un error tipado `port_unsupported` que nombra el plugin y el puerto, sin red. El puerto MUST estar conectado al verbo `axiom change start` alcanzable desde `cmd/axiom`.

#### Scenario: Plugin que ofrece el puerto
- **DADO** un plugin de prueba que implementa el puerto de work items
- **CUANDO** el marco resuelve el puerto por aserción de tipo
- **ENTONCES** la lectura, la descarga y el enlace de vuelta se despachan a ese plugin

#### Scenario: Plugin sin el puerto
- **DADO** un plugin que no implementa el puerto de work items
- **CUANDO** se invoca el puerto
- **ENTONCES** se devuelve el error tipado `port_unsupported` nombrando el plugin y el puerto
- **Y** no se hace ninguna operación de red

#### Scenario: Pánico del plugin
- **DADO** un plugin cuyo método de lectura entra en pánico
- **CUANDO** se invoca el puerto
- **ENTONCES** el pánico se convierte en un error tipado y el proceso no termina

#### Scenario: Los errores no llevan secretos
- **DADO** un plugin cuyo error contiene el valor del secreto `pat`
- **CUANDO** se devuelve el error por el puerto
- **ENTONCES** el valor del secreto no aparece en el mensaje devuelto

#### Scenario: La interfaz `Plugin` no cambia
- **DADO** un plugin escrito antes de este cambio que implementa solo `Plugin`
- **CUANDO** se compila y se registra
- **ENTONCES** compila sin cambios y se registra sin error

### Requirement: Puerto Opcional de Tasks por Rol (RF-INT-25)

El marco DEBE ofrecer un puerto opcional de Tasks por rol, detectado por aserción de tipo sobre el plugin (mismo patrón que RF-INT-23 y RF-INT-24), sin añadir métodos a `Plugin` ni alterar los plugins que no lo implementan. El puerto DEBE exponer, con tipos neutrales propios del marco: (a) la lectura de las Tasks hijas de un work item; (b) la creación de una Task hija con marca, estimación y relación al padre; (c) la lectura de una Task con su tipo, su padre, su estado, su `rev` y sus horas; (d) la actualización de horas con control optimista de revisión; (e) el cambio de estado y la asignación; (f) la sustitución de la descripción; y (g) la sustitución de un conjunto de etiquetas con prefijo. La invocación DEBE pasar por `invokePort`: recupera los pánicos del plugin como error tipado y sanea de secretos todo error devuelto. Si el plugin no implementa el puerto, DEBE devolver el error tipado `port_unsupported` que nombra el plugin y el puerto, sin red. El puerto DEBE estar conectado a los verbos de los requisitos siguientes, alcanzables desde `cmd/axiom`.

#### Scenario: Plugin que ofrece el puerto
- **DADO** un plugin de prueba que implementa el puerto de Tasks
- **CUANDO** el marco resuelve el puerto por aserción de tipo
- **ENTONCES** la creación, la lectura, las horas, el estado, la descripción y las etiquetas se despachan a ese plugin

#### Scenario: Plugin sin el puerto
- **DADO** un plugin que no implementa el puerto de Tasks
- **CUANDO** se invoca el puerto
- **ENTONCES** se devuelve `port_unsupported` nombrando el plugin y el puerto
- **Y** no se hace ninguna operación de red

#### Scenario: Pánico del plugin
- **DADO** un plugin cuyo método de creación entra en pánico
- **CUANDO** se invoca el puerto
- **ENTONCES** el pánico se convierte en un error tipado y el proceso no termina

#### Scenario: Los errores no llevan secretos
- **DADO** un plugin cuyo error contiene el valor del secreto `pat`
- **CUANDO** se devuelve el error por el puerto
- **ENTONCES** el valor del secreto no aparece en el mensaje devuelto

#### Scenario: La interfaz `Plugin` no cambia
- **DADO** un plugin escrito antes de este cambio que implementa solo `Plugin`
- **CUANDO** se compila y se registra
- **ENTONCES** compila sin cambios y se registra sin error

### Requirement: Verbo `axiom integrations tasks create` (RF-INT-26)

`axiom integrations tasks create --change <cambio> [--estimate <rol>=<horas> ...]` DEBE crear o reutilizar la Task hija del WI de origen de cada rol del roster sellado del cambio. Es un verbo síncrono, fuera de la outbox. Para cada rol DEBE: (1) reutilizar el vínculo `kind: task` ya registrado en `integrations.yaml` sin crear nada (solo toca el tracker para actualizar una estimación confirmada distinta, como se describe al final); (2) si no hay vínculo, buscar por la marca `axiom-task:<cambio>:<rol>` entre las Tasks hijas del WI y vincular la coincidente, y con varias coincidencias negarse con `task_ambiguous` nombrando `axiom integrations tasks bind --change <cambio> --role <rol>=<id>`; (3) solo con cero coincidencias crear la Task con la marca, la relación al WI y la estimación, y registrar el vínculo en cuanto recibe la respuesta, con `source: tasks.create`. La estimación de cada rol DEBE ser la de `--estimate` si se da, y si no la de la sección de `design.md` (requisito de estimación por rol); sin ninguna de las dos para un rol DEBE negarse para ese rol con `estimate_missing`. Un valor de estimación no numérico, cero o negativo DEBE rechazarse como error de uso. El vínculo DEBE guardar `estimate_hours`. Un rol cuya creación falla NO DEBE impedir procesar los demás roles; el verbo DEBE terminar con código distinto de cero y la salida DEBE listar, por rol, su resultado y, para cada fallo, el rechazo tipado y su resolución nombrada, que DEBE incluir `axiom integrations tasks create --change <cambio>` (reintento) y `axiom integrations tasks bind --change <cambio> --role <rol>=<id>`. El fallo NO DEBE alterar el registro de la compuerta `design` (la aprobación ya registrada permanece) ni ningún otro fichero del cambio. Repetir el verbo con todos los vínculos presentes y, para cada rol, la misma estimación confirmada que guarda su vínculo en `estimate_hours` NO DEBE hacer ninguna escritura ni en el tracker ni en `integrations.yaml`. Si un rol ya tiene vínculo y su estimación confirmada (la de `--estimate` o, si falta, la de `design.md`) difiere de su `estimate_hours`, el verbo DEBE leer la Task y, en una sola modificación con `test /rev`, fijar `OriginalEstimate` en la nueva estimación y `RemainingWork` en el máximo entre cero y la nueva estimación menos el `CompletedWork` leído. Solo tras esa escritura con éxito DEBE actualizar `estimate_hours` en el vínculo. Un conflicto de `rev` DEBE rechazar ese rol con `rev_conflict`, sin reaplicar ni actualizar el vínculo, con la resolución `axiom integrations tasks create --change <cambio>`.

#### Scenario: Creación de una Task por rol
- **DADO** un cambio con el WI 100 vinculado, un roster sellado con el rol `fullstack` y un `design.md` con la estimación de 6 horas para `fullstack`
- **CUANDO** se ejecuta `axiom integrations tasks create --change x`
- **ENTONCES** el doble recibe la creación de una Task con la marca `axiom-task:x:fullstack` y las horas 6 en `OriginalEstimate` y `RemainingWork`
- **Y** `integrations.yaml` contiene un vínculo `kind: task` con `role: fullstack`, `estimate_hours: 6` y `source: tasks.create`
- **Y** el verbo termina con código 0

#### Scenario: `--estimate` prevalece sobre `design.md`
- **DADO** `design.md` con 6 horas para `fullstack`
- **CUANDO** se ejecuta `tasks create --change x --estimate fullstack=4`
- **ENTONCES** la Task se crea con 4 horas y el vínculo guarda `estimate_hours: 4`

#### Scenario: Repetir tras respuesta perdida no duplica
- **DADO** una creación cuyo `POST` se aplicó pero cuya respuesta se perdió y sin vínculo registrado
- **CUANDO** se repite `tasks create --change x`
- **ENTONCES** el verbo encuentra la Task por la marca y la vincula
- **Y** el doble no recibe ningún `POST` de creación más

#### Scenario: Vínculo ya registrado
- **DADO** un vínculo `kind: task` del rol `fullstack` con `estimate_hours: 6` en `integrations.yaml` y un `design.md` con 6 horas para `fullstack`
- **CUANDO** se repite `tasks create --change x`
- **ENTONCES** no se hace ninguna petición de creación ni escritura y `integrations.yaml` no cambia
- **Y** el verbo termina con código 0

#### Scenario: Estimación confirmada distinta con el vínculo ya registrado
- **DADO** un vínculo `kind: task` del rol `fullstack` con `estimate_hours: 6`, la Task en la revisión 7 con `CompletedWork` 2 y un `design.md` iterado con 10 horas para `fullstack`
- **CUANDO** se ejecuta `tasks create --change x`
- **ENTONCES** el doble recibe un único `PATCH` con `test /rev` 7, `OriginalEstimate` 10 y `RemainingWork` 8
- **Y** el vínculo queda con `estimate_hours: 10` y no se crea ninguna Task

#### Scenario: Conflicto de `rev` al actualizar la estimación
- **DADO** un vínculo con `estimate_hours: 6` y una Task modificada a mano entre la lectura y la escritura
- **CUANDO** se ejecuta `tasks create --change x --estimate fullstack=10`
- **ENTONCES** el rol se rechaza con `rev_conflict`, cuya resolución nombra `axiom integrations tasks create --change x`
- **Y** el vínculo conserva `estimate_hours: 6` y el doble no registra ninguna modificación

#### Scenario: Marcas duplicadas
- **DADO** dos Tasks hijas con la misma marca del rol
- **CUANDO** se ejecuta `tasks create --change x`
- **ENTONCES** el rol queda con el rechazo `task_ambiguous` cuya resolución nombra `axiom integrations tasks bind`
- **Y** no se crea ninguna Task para ese rol y el código de salida es distinto de cero

#### Scenario: Sin estimación para un rol
- **DADO** un `design.md` sin sección de estimación y sin `--estimate`
- **CUANDO** se ejecuta `tasks create --change x`
- **ENTONCES** se rechaza el rol con `estimate_missing` cuya resolución nombra `--estimate`
- **Y** el doble no recibe ninguna creación

#### Scenario: Fallo de creación con la aprobación registrada
- **DADO** la compuerta `design` aprobada en `gates.yaml` y un doble que falla la creación
- **CUANDO** se ejecuta `tasks create --change x`
- **ENTONCES** el verbo termina con código distinto de cero y la salida nombra `axiom integrations tasks create --change x` y `axiom integrations tasks bind --change x --role fullstack=<id>`
- **Y** `gates.yaml` conserva la decisión `approved` sin cambios

#### Scenario: Un fallo no impide los demás roles
- **DADO** un roster sellado con dos roles y un doble que falla solo la creación del primero
- **CUANDO** se ejecuta `tasks create --change x`
- **ENTONCES** el segundo rol queda con su vínculo y el primero con su rechazo tipado
- **Y** el código de salida es distinto de cero

#### Scenario: Estimación inválida
- **DADO** `--estimate fullstack=0` o `--estimate fullstack=abc`
- **CUANDO** se ejecuta `tasks create --change x`
- **ENTONCES** termina con error de uso y no se hace ninguna petición ni escritura

### Requirement: Verbo `axiom integrations tasks bind` (RF-INT-27)

`axiom integrations tasks bind --change <cambio> --role <rol>=<id> [--estimate <rol>=<horas>] [--replace]` DEBE vincular a mano una Task existente al rol. DEBE rechazar un rol ausente del roster sellado con `role_not_in_roster`, y validar con el adaptador, antes de escribir, que el elemento existe (`task_not_found`), que es de tipo Task (`not_a_task`) y que cuelga del WI del cambio (`not_child_of_work_item`); un fallo de validación NO DEBE escribir `integrations.yaml` ni el tracker. Con éxito DEBE escribir el vínculo `kind: task` con `source: tasks.bind` y su `estimate_hours` (de `--estimate` o, si falta, de la sección de `design.md`; sin ninguna, el vínculo se escribe sin `estimate_hours` y la salida avisa de ello). Con una estimación y una Task sin `OriginalEstimate`, DEBE escribir `OriginalEstimate` y `RemainingWork` con esa estimación; si la Task ya tenía `OriginalEstimate`, NO DEBE tocar las horas del tracker. Vincular de nuevo la misma Task al mismo rol NO DEBE cambiar el fichero. Vincular otra Task a un rol que ya tiene vínculo sin `--replace` DEBE rechazarse con `task_already_bound`, sin escribir nada; su resolución nombra `axiom integrations tasks bind --change <cambio> --role <rol>=<id> --replace`. Con `--replace`, el verbo DEBE revalidar la Task nueva con las mismas comprobaciones y los mismos rechazos (`task_not_found`, `not_a_task`, `not_child_of_work_item`) y, solo si pasa, sustituir bajo el cerrojo del cambio el vínculo `kind: task` de ese rol por el de la Task nueva con `source: tasks.bind`. La sustitución conserva el `started_at` del vínculo anterior y todas las imputaciones ya registradas, que mantienen su `item_id` original. Conserva también su `estimate_hours` cuando no hay ni `--estimate` ni sección de `design.md`. Las horas de la Task nueva siguen la regla de estimación de este requisito. Un fallo de validación con `--replace` NO DEBE modificar el vínculo existente. `--replace` sin vínculo previo del rol equivale a un `bind` normal, y `--replace` con la misma Task ya vinculada NO DEBE cambiar el fichero.

#### Scenario: Vínculo manual válido
- **DADO** una Task 205 de tipo Task hija del WI 100 del cambio `x`
- **CUANDO** se ejecuta `tasks bind --change x --role fullstack=205`
- **ENTONCES** `integrations.yaml` contiene el vínculo `kind: task`, `id: "205"`, `role: fullstack` y `source: tasks.bind`

#### Scenario: Identificador inexistente
- **DADO** un identificador que el servidor responde como inexistente
- **CUANDO** se ejecuta `tasks bind --change x --role fullstack=999`
- **ENTONCES** se rechaza con `task_not_found` y `integrations.yaml` no se escribe

#### Scenario: El elemento no es Task
- **DADO** el elemento 300 de tipo `Bug`
- **CUANDO** se ejecuta `tasks bind --change x --role fullstack=300`
- **ENTONCES** se rechaza con `not_a_task` y no se escribe nada

#### Scenario: La Task cuelga de otro WI
- **DADO** una Task cuyo padre no es el WI del cambio
- **CUANDO** se ejecuta `tasks bind --change x --role fullstack=205`
- **ENTONCES** se rechaza con `not_child_of_work_item` y no se escribe nada

#### Scenario: Rol fuera del roster
- **DADO** un kickoff sellado con el rol `fullstack`
- **CUANDO** se ejecuta `tasks bind --change x --role backend=205`
- **ENTONCES** se rechaza con `role_not_in_roster` y no se hace ninguna petición

#### Scenario: Estimación sobre una Task sin horas
- **DADO** una Task 205 sin `OriginalEstimate`
- **CUANDO** se ejecuta `tasks bind --change x --role fullstack=205 --estimate fullstack=5`
- **ENTONCES** el doble recibe `OriginalEstimate` y `RemainingWork` iguales a 5 con `test /rev`
- **Y** el vínculo guarda `estimate_hours: 5`

#### Scenario: Estimación sobre una Task con horas
- **DADO** una Task 205 con `OriginalEstimate` igual a 8
- **CUANDO** se ejecuta `tasks bind --change x --role fullstack=205 --estimate fullstack=5`
- **ENTONCES** el doble no recibe ninguna escritura de horas
- **Y** el vínculo guarda `estimate_hours: 5`

#### Scenario: Otra Task en un rol ya vinculado
- **DADO** el rol `fullstack` vinculado a la Task 205
- **CUANDO** se ejecuta `tasks bind --change x --role fullstack=206`
- **ENTONCES** se rechaza con `task_already_bound` y `integrations.yaml` no cambia
- **Y** la resolución nombra `axiom integrations tasks bind --change x --role fullstack=206 --replace`

#### Scenario: Sustituir el vínculo de un rol
- **DADO** el rol `fullstack` vinculado a la Task 205 con `started_at` igual a `2026-11-23T09:00:00Z` y una imputación `role.close/fullstack/1` sobre el `item_id` 205, y una Task 206 de tipo Task hija del WI del cambio
- **CUANDO** se ejecuta `tasks bind --change x --role fullstack=206 --replace`
- **ENTONCES** el vínculo `kind: task` del rol pasa a `id: "206"` con `source: tasks.bind` y `started_at` igual a `2026-11-23T09:00:00Z`
- **Y** no queda ningún vínculo `kind: task` del rol con `id: "205"`
- **Y** la imputación `role.close/fullstack/1` sigue registrada con `item_id: "205"`

#### Scenario: Sustitución con una Task que no valida
- **DADO** el rol `fullstack` vinculado a la Task 205 y una Task 206 cuyo padre es otro WI
- **CUANDO** se ejecuta `tasks bind --change x --role fullstack=206 --replace`
- **ENTONCES** se rechaza con `not_child_of_work_item` y `integrations.yaml` no cambia

#### Scenario: Vincular dos veces la misma Task
- **DADO** el rol `fullstack` vinculado a la Task 205
- **CUANDO** se repite `tasks bind --change x --role fullstack=205`
- **ENTONCES** `integrations.yaml` no cambia

### Requirement: Verbo `axiom integrations summary push` (RF-INT-28)

`axiom integrations summary push --change <cambio>` DEBE sustituir `System.Description` del WI de origen por el resumen HTML de la spec del cambio (renderizador del requisito siguiente) y sincronizar las etiquetas de IDs funcionales (etiquetas con el prefijo configurado, una por ID) en una sola escritura, sin tocar `integrations.yaml`. Los IDs funcionales DEBEN extraerse de los títulos `### Requirement:` de las deltas del cambio: con provisionales antes de archivar y definitivos tras archivar. Sobre la carpeta archivada del cambio DEBE leer los IDs definitivos de la propia carpeta (sus deltas reescritas y su `ids.yaml`) y NO DEBE emitir ningún identificador provisional; si queda alguno sin resolver DEBE rechazar con `provisional_ids_unresolved`. Si el HTML y las etiquetas no cambian respecto del tracker, DEBE terminar con código 0 sin ninguna escritura. Un tope de etiquetas superado DEBE terminar con código 0, con la descripción actualizada y el aviso tipado `req_tags_over_limit` en la salida. Un cambio sin ninguna delta de spec DEBE rechazarse con `artifact_missing`. NO DEBE modificar ningún otro campo del WI.

#### Scenario: Resumen antes de archivar
- **DADO** un cambio con una delta que añade un requisito con el ID provisional `P1` y un WI 100 con descripción original
- **CUANDO** se ejecuta `summary push --change x`
- **ENTONCES** la descripción del WI pasa a ser el HTML del resumen y las etiquetas incluyen `req:P1`
- **Y** `integrations.yaml` no cambia

#### Scenario: Resumen sobre la carpeta archivada
- **DADO** el cambio archivado con `ids.yaml` que asigna el definitivo `D1` al provisional `P1`
- **CUANDO** se ejecuta `summary push --change x`
- **ENTONCES** la descripción contiene `D1` y ningún provisional
- **Y** las etiquetas pasan de `req:P1` a `req:D1`, con las ajenas conservadas

#### Scenario: Repetición sin cambios
- **DADO** una ejecución anterior de `summary push` con la misma spec
- **CUANDO** se repite
- **ENTONCES** el doble no recibe ninguna escritura y el verbo termina con código 0

#### Scenario: Tope de etiquetas superado
- **DADO** más IDs funcionales que el tope configurado
- **CUANDO** se ejecuta `summary push --change x`
- **ENTONCES** la descripción se actualiza, no se envía ninguna etiqueta de ID
- **Y** la salida incluye el aviso `req_tags_over_limit` y el código de salida es 0

#### Scenario: Provisional sin resolver en la carpeta archivada
- **DADO** una carpeta archivada cuyas deltas aún contienen un provisional sin entrada en `ids.yaml`
- **CUANDO** se ejecuta `summary push --change x`
- **ENTONCES** se rechaza con `provisional_ids_unresolved` y no se escribe nada

#### Scenario: Sin spec
- **DADO** un cambio sin ninguna delta de spec
- **CUANDO** se ejecuta `summary push --change x`
- **ENTONCES** se rechaza con `artifact_missing` y no se escribe nada

#### Scenario: Solo la descripción y las etiquetas
- **DADO** un WI con título, estado y asignado
- **CUANDO** se ejecuta `summary push --change x`
- **ENTONCES** el título, el estado y el asignado no cambian

### Requirement: Renderizador Determinista del Resumen de la Spec (RF-INT-29)

El resumen de la spec DEBE generarse con un renderizador en un paquete hoja propio, sin red y sin reutilizar `resumen-spec.html`, a partir de las deltas del cambio. Por cada capacidad, en orden alfabético, DEBE emitir un `h2` con su nombre y, por cada requisito en el orden del fichero, un `h3` con su título (que incluye su ID) y un `p` con su primer párrafo normativo, con el marcado en línea `` `código` `` convertido a `code` y `**texto**` a `strong`. El HTML SOLO DEBE emplear las etiquetas `h2`, `h3`, `p`, `ul`, `li`, `code` y `strong`, sin ningún atributo, y todo el texto DEBE ir escapado (`&`, `<`, `>`, `"` y `'`): nunca `style`, SVG ni scripts. A igual entrada DEBE producir la misma salida byte a byte. Una delta cuyo texto contenga marcado HTML DEBE emitirlo escapado como texto.

#### Scenario: Estructura del resumen
- **DADO** una capacidad `alfa` con dos requisitos y una capacidad `beta` con uno
- **CUANDO** se renderiza el resumen
- **ENTONCES** el HTML contiene `h2` de `alfa` y de `beta` en ese orden, un `h3` por requisito en el orden del fichero y un `p` con el primer párrafo de cada uno

#### Scenario: Solo etiquetas de la lista blanca y sin atributos
- **DADO** una delta cualquiera
- **CUANDO** se renderiza el resumen
- **ENTONCES** todas las etiquetas del HTML pertenecen a `h2`, `h3`, `p`, `ul`, `li`, `code` y `strong`
- **Y** ninguna etiqueta tiene atributos y no aparecen `style`, `svg` ni `script`

#### Scenario: Escape del texto
- **DADO** un requisito cuyo texto contiene `<script>alert(1)</script>` y `a & b`
- **CUANDO** se renderiza el resumen
- **ENTONCES** el HTML contiene `&lt;script&gt;` y `a &amp; b` y ninguna etiqueta `script`

#### Scenario: Determinismo
- **DADO** la misma entrada
- **CUANDO** se renderiza dos veces
- **ENTONCES** las dos salidas son idénticas byte a byte

#### Scenario: Marcado en línea
- **DADO** un párrafo con `` `ensure` `` y `**DEBE**`
- **CUANDO** se renderiza el resumen
- **ENTONCES** el HTML contiene `<code>ensure</code>` y `<strong>DEBE</strong>`

### Requirement: Estimación por Rol en `design.md` (RF-INT-30)

El diseño del cambio DEBE llevar la sección fija `## Estimación por rol` con una línea `- <rol>: <horas> h` por cada rol del roster sellado, con horas decimales positivas. Los verbos `tasks create` y `tasks bind` DEBEN leer esa sección cuando no se pasa `--estimate`. Una línea malformada, un valor no positivo, un rol repetido o un rol fuera del roster DEBE rechazarse con `estimate_invalid` que nombra la línea; la ausencia de la sección NO es un error por sí sola (solo lo es la falta de estimación del rol concreto, `estimate_missing`). El contenido de la sección forma parte del digest de la compuerta `design`.

#### Scenario: Lectura de la sección
- **DADO** un `design.md` con `- fullstack: 6 h` bajo `## Estimación por rol`
- **CUANDO** `tasks create` lee la estimación del rol `fullstack`
- **ENTONCES** obtiene 6 horas

#### Scenario: Horas decimales
- **DADO** una línea `- fullstack: 2.5 h`
- **CUANDO** se lee la estimación
- **ENTONCES** obtiene 2,5 horas

#### Scenario: Línea malformada
- **DADO** una línea `- fullstack: mucho`
- **CUANDO** se lee la estimación
- **ENTONCES** se rechaza con `estimate_invalid` que nombra la línea

#### Scenario: Rol repetido o fuera del roster
- **DADO** dos líneas del mismo rol o una línea de un rol que no está en el roster sellado
- **CUANDO** se lee la estimación
- **ENTONCES** se rechaza con `estimate_invalid`

#### Scenario: Cambiar la sección cambia el digest
- **DADO** la compuerta `design` aprobada
- **CUANDO** se edita una hora de la sección `## Estimación por rol`
- **ENTONCES** el digest de `design.md` cambia y la compuerta vuelve a `pending`

### Requirement: Verbo `axiom integrations time propose` (RF-INT-31)

`axiom integrations time propose --change <cambio> --role <rol>` DEBE ser de solo lectura: no escribe en el tracker ni en `integrations.yaml`. DEBE exigir el vínculo `kind: task` del rol con `started_at` registrado (si falta la Task, `task_not_bound`; si falta `started_at`, `role_not_started` cuya resolución nombra `axiom branch ensure --change <cambio> --role <rol>`). DEBE mostrar: el reloj desde `started_at` hasta ahora, recortado a la jornada; la estimación del vínculo; y `OriginalEstimate`, `RemainingWork` y `CompletedWork` leídos del tracker en ese momento, que incluyen las horas metidas a mano en Azure DevOps. El recorte DEBE sumar, por cada día natural del intervalo en la zona horaria local, como máximo las horas de la jornada configurada (`workday_hours`) y solo si el día pertenece a `workdays`; los demás días no suman. El redondeo y la zona horaria los fija el diseño y se documentan. La propuesta NO DEBE ser vinculante: `time record` toma las horas de su propio `--hours`.

#### Scenario: Propuesta recortada a la jornada
- **DADO** un rol iniciado el lunes a las 09:00 y consultado el martes a las 12:00 con jornada de 8 horas de lunes a viernes
- **CUANDO** se ejecuta `time propose --change x --role fullstack`
- **ENTONCES** la propuesta es de 11 horas (8 del lunes y 3 del martes)

#### Scenario: Un día entero no pasa de la jornada
- **DADO** un rol iniciado a las 00:00 y consultado a las 23:59 del mismo lunes con jornada de 8 horas
- **CUANDO** se ejecuta `time propose`
- **ENTONCES** la propuesta es de 8 horas

#### Scenario: Fin de semana sin suma
- **DADO** un rol iniciado el sábado y consultado el domingo con días laborables de lunes a viernes
- **CUANDO** se ejecuta `time propose`
- **ENTONCES** la propuesta es de 0 horas

#### Scenario: Muestra los valores del tracker
- **DADO** una Task con `OriginalEstimate` 8, `RemainingWork` 5 y `CompletedWork` 3 (parte metida a mano)
- **CUANDO** se ejecuta `time propose`
- **ENTONCES** la salida muestra la estimación del vínculo y los tres valores leídos del tracker
- **Y** el doble no recibe ninguna escritura y `integrations.yaml` no cambia

#### Scenario: Rol sin Task vinculada
- **DADO** un rol sin vínculo `kind: task`
- **CUANDO** se ejecuta `time propose`
- **ENTONCES** se rechaza con `task_not_bound` cuya resolución nombra `axiom integrations tasks create`

#### Scenario: Rol sin inicio registrado
- **DADO** un vínculo `kind: task` sin `started_at`
- **CUANDO** se ejecuta `time propose`
- **ENTONCES** se rechaza con `role_not_started` cuya resolución nombra `axiom branch ensure --change x --role fullstack`

### Requirement: Verbo `axiom integrations time record` (RF-INT-32)

`axiom integrations time record --change <cambio> --role <rol> (--hours <horas> | --resolve-pending applied|discarded | --close-only)` DEBE imputar las horas a la Task del rol: leer su `rev`, sumar las horas a `CompletedWork`, fijar `RemainingWork` en el máximo entre cero y su valor menos las horas, y escribir con `test /rev` (RF-ADO-07 y la imputación del adaptador). Las horas DEBEN ser un número positivo; cero, negativo o no numérico es un error de uso sin ninguna petición. Solo tras una escritura con éxito DEBE registrar en `integrations.yaml` una imputación con `actor` (la identidad git de la persona, la de `change-actor-identity`), `minutes` (las horas por 60 redondeadas al minuto), `date` (la fecha de la imputación), `role`, `item_id` y un `source` único por cierre con la forma `role.close/<rol>/<n>`, con `<n>` correlativo desde 1 entre las imputaciones del rol. Con un conflicto de `rev` DEBE terminar con código distinto de cero y `rev_conflict`, mostrar los valores releídos, NO DEBE reaplicar la escritura ni registrar la imputación, y la resolución nombra `axiom integrations time propose` para volver a decidir las horas. Un fallo del tracker (red, 5xx, campo ausente) NO DEBE registrar la imputación. En `checkpointed`, `time record` NO DEBE tocar el estado de la Task (lo aplica el evento `role.apply.approved`); en `continuous` DEBE, tras la imputación, pasar la Task al estado `task_state_close` con la misma operación idempotente del adaptador, y un fallo de esa transición NO DEBE deshacer la imputación ni su registro y se informa como aviso tipado. En `continuous`, la misma transición de cierre DEBE aplicarse también:
- tras registrar una imputación recuperada (`time_record_recovered`);
- tras `--resolve-pending applied`;
- con `--close-only`, que pasa la Task a `task_state_close` sin imputar horas, para un cierre sin horas que imputar.

`--close-only` es incompatible con `--hours` y con `--resolve-pending`, y en `checkpointed` es un error de uso. Un rol rechazado y aprobado de nuevo PUEDE imputar otra vez: cada cierre usa su propio `source`.

Para no perder ni duplicar una imputación cuyo `PATCH` se aplicó pero cuyo registro local falló, `time record` DEBE escribir, antes del `PATCH`, una intención en el diario local `.time-pending.yaml` de la raíz de la carpeta del cambio. Es estado local que nunca se publica ni se versiona. La intención lleva el rol, el `item_id`, las horas, los minutos, el actor, la fecha, el `source`, la revisión leída y los valores esperados de `CompletedWork` y `RemainingWork`. Si la intención no se puede escribir, el verbo DEBE rechazar con `write_failed` sin tocar el tracker. La intención DEBE borrarse solo cuando consta que la escritura no se aplicó: un rechazo de `test /rev` en el primer intento o una respuesta 4xx distinta del conflicto. También se borra tras una escritura con éxito y el registro de la imputación. Ante un resultado incierto la intención DEBE conservarse: fallo de transporte sin respuesta, tiempo agotado, 5xx, o conflicto tras un reintento no verificado.

`time propose` es de solo lectura también con una intención pendiente: no la reconcilia ni la borra, y avisa `time_pending` con la resolución `axiom integrations time record --change <cambio> --role <rol>`. Al empezar `time record` de un rol con una intención pendiente, el verbo DEBE releer la Task de la intención (su `item_id`, aunque el vínculo del rol haya cambiado) y reconciliarla antes de nada:

- Con la revisión igual a la leída, la intención se borra y el verbo sigue.
- Con una revisión posterior y los dos valores iguales a los esperados, registra la imputación de la intención, la borra y avisa `time_record_recovered`. En esa ejecución, `time record` NO DEBE aplicar sus `--hours`.
- En cualquier otro caso, DEBE rechazar con `time_pending_ambiguous`, mostrando los valores releídos y sin escribir nada. Su resolución nombra `axiom integrations time record --change <cambio> --role <rol> --resolve-pending applied|discarded`.

`--resolve-pending applied` registra la imputación de la intención sin tocar el tracker y la borra; `--resolve-pending discarded` la borra sin registrar nada. `--resolve-pending` es incompatible con `--hours` (error de uso) y, sin intención pendiente del rol, es un error de uso.

#### Scenario: Imputación correcta
- **DADO** una Task en la revisión 4 con `CompletedWork` 3 y `RemainingWork` 5 y la persona `Ana <ana@x.test>` como actor git
- **CUANDO** se ejecuta `time record --change x --role fullstack --hours 2`
- **ENTONCES** el doble recibe un `PATCH` con `test /rev` 4, `CompletedWork` 5 y `RemainingWork` 3
- **Y** `integrations.yaml` contiene una imputación con actor `Ana <ana@x.test>`, 120 minutos, la fecha de hoy, `role: fullstack`, el `item_id` de la Task y `source: role.close/fullstack/1`

#### Scenario: Conflicto de `rev`
- **DADO** una Task modificada a mano entre la lectura y la escritura
- **CUANDO** se ejecuta `time record --change x --role fullstack --hours 2`
- **ENTONCES** termina con código distinto de cero y `rev_conflict`, con los valores releídos en la salida
- **Y** el doble no registra ninguna modificación y `integrations.yaml` no recibe ninguna imputación

#### Scenario: Segundo cierre del mismo rol
- **DADO** una imputación `role.close/fullstack/1` ya registrada y el rol rechazado y aprobado de nuevo
- **CUANDO** se ejecuta otro `time record` del rol
- **ENTONCES** se registra una imputación con `source: role.close/fullstack/2` y no choca con la anterior

#### Scenario: Horas inválidas
- **DADO** `--hours 0`, `--hours -1` o `--hours mucho`
- **CUANDO** se ejecuta `time record`
- **ENTONCES** termina con error de uso, sin ninguna petición y sin tocar `integrations.yaml`

#### Scenario: Fallo del tracker
- **DADO** un doble que responde 500 a la escritura
- **CUANDO** se ejecuta `time record`
- **ENTONCES** termina con código distinto de cero y no se registra ninguna imputación

#### Scenario: `checkpointed` no toca el estado
- **DADO** un cambio sellado `checkpointed` con la compuerta `role-apply:fullstack` aprobada
- **CUANDO** se ejecuta `time record`
- **ENTONCES** el doble no recibe ningún cambio de estado de la Task

#### Scenario: `continuous` aplica el estado de cierre
- **DADO** un cambio sellado `continuous` y `task_state_close` igual a `Tech Review`
- **CUANDO** se ejecuta `time record` con éxito
- **ENTONCES** la Task queda en `Tech Review`

#### Scenario: Fallo de la transición en `continuous`
- **DADO** un cambio `continuous` y un estado de cierre inexistente en el proceso
- **CUANDO** se ejecuta `time record`
- **ENTONCES** la imputación queda escrita y registrada
- **Y** la salida incluye un aviso tipado con los estados válidos y el código de salida es 0

#### Scenario: `PATCH` aplicado y registro local fallido
- **DADO** un `time record --hours 2` sobre una Task en la revisión 4 con `CompletedWork` 3 y `RemainingWork` 5, cuyo `PATCH` se aplica pero cuya escritura de `integrations.yaml` falla
- **CUANDO** se ejecuta después `time record --change x --role fullstack --hours 2` y la Task está en la revisión 5 con `CompletedWork` 5 y `RemainingWork` 3
- **ENTONCES** se registra la imputación de la intención con su `source` y se avisa `time_record_recovered`
- **Y** el doble no recibe ningún `PATCH` más y `.time-pending.yaml` ya no tiene la intención del rol

#### Scenario: Intención no aplicada
- **DADO** una intención pendiente con la revisión leída 4 y la Task todavía en la revisión 4
- **CUANDO** se ejecuta `time record --change x --role fullstack --hours 2`
- **ENTONCES** la intención se borra y la imputación sigue el flujo normal desde la revisión 4

#### Scenario: Intención ambigua
- **DADO** una intención pendiente con la revisión leída 4 y valores esperados `CompletedWork` 5 y `RemainingWork` 3, y la Task en la revisión 6 con `CompletedWork` 7
- **CUANDO** se ejecuta `time record` del rol
- **ENTONCES** se rechaza con `time_pending_ambiguous`, con los valores releídos y la resolución `axiom integrations time record --change x --role fullstack --resolve-pending applied|discarded`
- **Y** no se escribe nada ni en el tracker ni en `integrations.yaml`

#### Scenario: Resolver la intención a mano
- **DADO** una intención pendiente ambigua del rol `fullstack`
- **CUANDO** se ejecuta `time record --change x --role fullstack --resolve-pending applied`
- **ENTONCES** se registra la imputación de la intención sin ninguna petición al tracker y la intención se borra
- **Y** con `--resolve-pending discarded` la intención se borra sin registrar ninguna imputación

#### Scenario: La intención es estado local
- **DADO** un `.time-pending.yaml` en la carpeta del cambio
- **CUANDO** se publican los documentos del cambio con `axiom sdd publish`
- **ENTONCES** el fichero no se confirma ni se publica

#### Scenario: `time propose` no reconcilia
- **DADO** una intención pendiente del rol `fullstack`
- **CUANDO** se ejecuta `time propose --change x --role fullstack`
- **ENTONCES** la salida incluye el aviso `time_pending` con la resolución `axiom integrations time record --change x --role fullstack`
- **Y** `.time-pending.yaml` e `integrations.yaml` no cambian

#### Scenario: Fallo de transporte sin respuesta
- **DADO** un `PATCH` de horas cuya conexión se corta sin respuesta en el intento y en el reintento
- **CUANDO** termina `time record`
- **ENTONCES** termina con código distinto de cero, no registra ninguna imputación y conserva la intención
- **Y** el siguiente `time record` del rol la reconcilia releyendo la Task

#### Scenario: La reconciliación usa la Task de la intención
- **DADO** una intención pendiente sobre el `item_id` 205 y el vínculo del rol ya sustituido por la Task 206
- **CUANDO** se ejecuta `time record --change x --role fullstack --hours 2`
- **ENTONCES** la reconciliación relee la Task 205, no la 206

#### Scenario: Cierre sin horas en `continuous`
- **DADO** un cambio `continuous` y `task_state_close` igual a `Tech Review`
- **CUANDO** se ejecuta `time record --change x --role fullstack --close-only`
- **ENTONCES** la Task queda en `Tech Review` y no se registra ninguna imputación
- **Y** en un cambio `checkpointed` la misma invocación es un error de uso sin ninguna petición

### Requirement: Precondiciones por Estilo, Carril ODD y Plugin Activo de los Verbos Nuevos (RF-INT-33)

Los verbos `tasks create`, `tasks bind`, `summary push`, `time propose` y `time record` DEBEN comprobar, antes de cualquier petición o escritura y por este orden: (1) que el cambio es del carril SDD; el nombre de un cambio del carril ODD (con `odd/tasks/<nombre>.md` y sin carpeta SDD) DEBE rechazarse con `odd_lane_not_supported` y ningún camino de ODD crea Tasks, horas ni escribe `integrations.yaml`; (2) que hay un plugin activo con el puerto de Tasks, y sin él DEBEN responder `not_applicable`, con código 0, sin red y sin crear ficheros; (3) que el cambio tiene un vínculo `kind: work_item`, y sin él DEBEN rechazar con `work_item_not_linked` (sin Task que colgar de un WI inexistente, y sin crear el WI); (4) la precondición del estilo sellado. En `checkpointed`, `summary push` exige la compuerta `spec` con su última decisión `approved`, `tasks create` la `design` y `time propose` y `time record` la `role-apply:<rol>`, y sin ella DEBEN rechazar con `gate_not_approved` que nombra la compuerta y la resolución `axiom sdd gate show --change <cambio>`. En `continuous`, que no tiene compuertas, basta con que el artefacto exista: las deltas para `summary push` y `design.md` para `tasks create`, y sin él DEBEN rechazar con `artifact_missing`. Para `time propose` y `time record`, el artefacto es el vínculo `kind: task` del rol, y su ausencia se rechaza en ambos estilos con `task_not_bound` (requisito de `time propose`), nunca con `artifact_missing`. Los verbos NO DEBEN fabricar aprobaciones ni registrar compuertas. Sobre la carpeta archivada, `summary push` NO exige compuerta alguna. Toda continuación de un rechazo DEBE nombrar un comando `axiom` o declarar explícitamente que es de diseño sin resolución.

#### Scenario: Nombre del carril ODD
- **DADO** una característica ODD `f` con `odd/tasks/f.md` y sin carpeta SDD
- **CUANDO** se ejecuta `axiom integrations tasks create --change f`
- **ENTONCES** se rechaza con `odd_lane_not_supported` y código distinto de cero
- **Y** no se hace ninguna petición ni se crea ningún fichero

#### Scenario: ODD no escribe `integrations.yaml`
- **DADO** el carril ODD con un plugin activo
- **CUANDO** se ejecuta cualquiera de los cinco verbos con el nombre de la característica
- **ENTONCES** `integrations.yaml` no existe ni se modifica

#### Scenario: Sin plugin activo
- **DADO** un workspace sin plugin activo
- **CUANDO** se ejecuta cualquiera de los cinco verbos sobre un cambio SDD
- **ENTONCES** responde `not_applicable` con código 0
- **Y** no hace red y no crea ficheros

#### Scenario: Cambio sin vínculo de WI
- **DADO** un cambio SDD con plugin activo y sin vínculo `kind: work_item`
- **CUANDO** se ejecuta `tasks create --change x`
- **ENTONCES** se rechaza con `work_item_not_linked` y no se hace ninguna petición

#### Scenario: `checkpointed` exige la compuerta
- **DADO** un cambio `checkpointed` con la compuerta `design` en `pending`
- **CUANDO** se ejecuta `tasks create --change x`
- **ENTONCES** se rechaza con `gate_not_approved` que nombra `design` y `axiom sdd gate show --change x`
- **Y** no se hace ninguna petición ni se registra ninguna compuerta

#### Scenario: `checkpointed` con la compuerta rechazada
- **DADO** la última decisión de `role-apply:fullstack` en `rejected`
- **CUANDO** se ejecuta `time record --change x --role fullstack --hours 2`
- **ENTONCES** se rechaza con `gate_not_approved` y no se imputa nada

#### Scenario: `continuous` basta con el artefacto
- **DADO** un cambio `continuous` con `design.md` y sin compuertas
- **CUANDO** se ejecuta `tasks create --change x`
- **ENTONCES** el verbo procede sin exigir ninguna compuerta ni crear ninguna

#### Scenario: `continuous` sin artefacto
- **DADO** un cambio `continuous` sin `design.md`
- **CUANDO** se ejecuta `tasks create --change x`
- **ENTONCES** se rechaza con `artifact_missing` y no se hace ninguna petición

#### Scenario: `continuous` sin vínculo de Task del rol
- **DADO** un cambio `continuous` sin vínculo `kind: task` del rol `fullstack`
- **CUANDO** se ejecuta `time propose --change x --role fullstack` o `time record --change x --role fullstack --hours 2`
- **ENTONCES** se rechaza con `task_not_bound`, no con `artifact_missing`, y no se hace ninguna petición

#### Scenario: `summary push` sobre la carpeta archivada
- **DADO** un cambio archivado de estilo `checkpointed`
- **CUANDO** se ejecuta `summary push --change x`
- **ENTONCES** el verbo no exige ninguna compuerta y procede

### Requirement: Salida JSON Versionada y Sin Secretos de los Verbos Nuevos (RF-INT-34)

Los verbos `tasks create`, `tasks bind`, `summary push`, `time propose` y `time record` DEBEN aceptar `--cwd` y `--change` y ofrecer salida legible y salida `--json` con un esquema versionado propio por familia (`axiom.integrations-tasks/v1`, `axiom.integrations-summary/v1` y `axiom.integrations-time/v1`). La salida JSON DEBE incluir el estado (`done`, `partial`, `refused` o `not_applicable`), el detalle por rol cuando aplique, los avisos tipados y, para cada rechazo, su código y su resolución. Ni la salida, ni los errores, ni `integrations.yaml` DEBEN contener el PAT ni ninguna credencial (RF-INT-16): los canarios de secretos del marco y del adaptador DEBEN cubrir los verbos nuevos. Todo rechazo nuevo DEBE nombrar en su continuación un comando `axiom` concreto o declararse de diseño sin resolución.

#### Scenario: Esquema versionado
- **DADO** una ejecución con `--json` de cada familia de verbos
- **CUANDO** termina
- **ENTONCES** la salida lleva el campo de esquema `axiom.integrations-tasks/v1`, `axiom.integrations-summary/v1` o `axiom.integrations-time/v1` según la familia

#### Scenario: Rechazo con resolución nombrada
- **DADO** cualquier rechazo tipado de los verbos nuevos
- **CUANDO** se muestra
- **ENTONCES** incluye el código y una continuación que nombra un comando `axiom`

#### Scenario: Ningún secreto en la salida
- **DADO** un PAT conocido y fallos de autenticación del tracker
- **CUANDO** se ejecutan los cinco verbos con y sin `--json`
- **ENTONCES** el valor del PAT no aparece en stdout, en stderr ni en `integrations.yaml`

