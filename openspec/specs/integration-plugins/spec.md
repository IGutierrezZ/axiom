# Especificación de integration-plugins

## Purpose

Marco de plugins de integración (C-07-b): un punto de extensión hacia trackers externos que no acopla el núcleo de Axiom a ninguno. Fija el puerto `Plugin`, el manifiesto serializable, los catálogos cerrados de eventos y capacidades, el registro cerrado en compilación, el despacho duradero con reintento (outbox local), el fichero versionado de vínculos por cambio, el almacén seguro de secretos, los verbos `axiom integrations` y las garantías de coste cero y de no fuga de secretos. El bloque `integrations:` de `axiom.yaml` y los enganches en `gate record`, `branch ensure --role` y `sdd publish --archive` se especifican en las capacidades `workspace-topology`, `sdd-block-review-gates`, `change-branch-lifecycle` y `sdd-docs-publication`. Ningún adaptador real (Azure DevOps, C-07-c) forma parte de este cambio.

## Requirements

### Requirement: Catálogo Cerrado de Eventos y Puerto `Plugin` (REQ-43.1)

El paquete `internal/integrations` DEBE definir el puerto `Plugin` y el tipo `Event`. El catálogo de eventos DEBE ser cerrado y contener exactamente `change.start`, `gate.spec.approved`, `gate.design.approved`, `role.start`, `role.apply.approved` y `archive`. `change.start` pertenece al catálogo, pero ningún verbo de este cambio lo emite (lo dispara C-07-d). Todo `Event` DEBE llevar el nombre del evento, el cambio, la clave de idempotencia (REQ-43.7) y, cuando proceda, el rol y el actor. Un nombre de evento fuera del catálogo NO DEBE poder construirse ni despacharse: se rechaza con error tipado.

#### Scenario: El catálogo contiene exactamente los seis eventos
- **DADO** el paquete `internal/integrations`
- **CUANDO** se enumera el catálogo de eventos
- **ENTONCES** contiene `change.start`, `gate.spec.approved`, `gate.design.approved`, `role.start`, `role.apply.approved` y `archive`, y ninguno más

#### Scenario: Evento fuera del catálogo
- **DADO** un despachador con un plugin configurado
- **CUANDO** se intenta despachar un evento llamado `gate.tasks.approved`
- **ENTONCES** el despacho se rechaza con un error tipado que nombra los eventos admitidos
- **Y** no se crea ningún fichero ni se invoca al plugin

### Requirement: Catálogo Cerrado de Capacidades (REQ-43.2)

El catálogo de capacidades DEBE ser cerrado y contener exactamente `items.read`, `items.transition`, `items.create`, `relations`, `assignment`, `estimation`, `time`, `tags`, `attachments`, `vcs_links` y `discovery`. El manifiesto de un plugin SOLO PUEDE declarar capacidades de ese catálogo.

#### Scenario: Capacidad declarada válida
- **DADO** un manifiesto que declara `items.read` y `vcs_links`
- **CUANDO** se valida
- **ENTONCES** la validación termina sin error

#### Scenario: Capacidad desconocida
- **DADO** un manifiesto que declara la capacidad `sprints`
- **CUANDO** se registra el plugin
- **ENTONCES** el registro se rechaza con un error tipado que nombra `sprints` y las capacidades admitidas

### Requirement: Manifiesto Serializable, Versionado y en un Paquete Hoja (REQ-43.3)

Cada plugin DEBE exponer un manifiesto serializable en JSON con esquema versionado (`axiom.integration-manifest/v1`; el nombre definitivo lo fija el diseño). El manifiesto DEBE declarar: identificador, capacidades, eventos que escucha, esquema de configuración, nombres de los secretos que necesita y cómo se resuelve la identidad (`identity`). El manifiesto vive en `internal/integrations/manifest`, un paquete hoja que NO DEBE importar `workspace` (ni, de forma transitiva, `internal/integrations`). La validación DEBE rechazar: capacidades o eventos fuera de los catálogos cerrados, un identificador que no cumple `^[a-z][a-z0-9]*(-[a-z0-9]+)*$` o supera 32 caracteres, y toda clave de configuración que parezca un secreto. El rechazo es **por nombre de clave**: el campo de configuración no tiene ningún atributo de secreto; se rechazan las claves cuyo segmento (separado por `_`) sea `token`, `secret`, `secrets`, `password`, `passwd`, `pwd`, `pat`, `credential`, `credentials` o `apikey`, las que contengan el par `api_key`, las que coincidan con un nombre de la lista de secretos y la clave reservada `enabled`. Los secretos se declaran solo por nombre en la lista de secretos. También DEBE rechazarse un catálogo con identificadores repetidos o con dos pares (plugin, secreto) que den la misma variable de entorno (REQ-43.13).

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

### Requirement: Registro Cerrado en Compilación (REQ-43.4)

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

### Requirement: Coste Cero sin Integración Configurada (REQ-43.5)

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

### Requirement: Despacho Duradero con Intento Inmediato Acotado (REQ-43.6)

Con al menos un plugin configurado cuyo manifiesto escucha el evento, el despachador DEBE, en este orden: (1) encolar la operación de forma duradera en `.integrations-outbox.yaml` con su clave de idempotencia, (2) intentar la entrega inmediata al plugin con un tiempo máximo acotado (el valor lo fija el diseño), (3) si la entrega tiene éxito, persistir en `integrations.yaml` los vínculos que devuelva el plugin (salvo para `archive`, que nunca los persiste, REQ-43.30) y retirar la operación de la outbox, y (4) si la entrega falla, agota el tiempo o devuelve un error, dejar la operación pendiente con el número de intentos y el último error (sin secretos) para `axiom integrations sync`. Un plugin cuyo manifiesto no escucha el evento NO DEBE recibirlo ni generar operación. El plugin pide el secreto al almacén en el momento de entregar, y solo los de su manifiesto; el despachador NO DEBE leer secretos ni incluirlos en la operación. Un error que el plugin marca como permanente, o alcanzar el máximo de intentos, abandona la operación (REQ-43.8); cualquier otro error, incluido un pánico del plugin, es transitorio y no hace fallar el proceso. El plazo del intento inmediato se comparte entre los plugins del evento y un plugin que no respeta el plazo no bloquea el verbo más allá de él.

#### Scenario: Entrega inmediata con éxito
- **DADO** el doble en memoria configurado, que devuelve el vínculo `{item: "42"}` para el evento
- **CUANDO** se despacha `gate.spec.approved` del cambio `x`
- **ENTONCES** el doble recibe el evento exactamente una vez
- **Y** `integrations.yaml` de `x` contiene el vínculo devuelto
- **Y** no queda ninguna operación pendiente (la outbox se borra al quedar vacía, REQ-43.8)

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

### Requirement: Clave de Idempotencia Estable por Operación (REQ-43.7)

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

### Requirement: Outbox Local Persistente con Reintento (REQ-43.8)

Las operaciones pendientes DEBEN persistirse en `.integrations-outbox.yaml` en la raíz de la carpeta del cambio, con esquema versionado (`axiom.integrations-outbox/v1`). El fichero es estado local: su nombre empieza por punto, `docpaths.LocalState` lo excluye de la publicación y `axiom sdd publish` NUNCA lo confirma. Cada operación DEBE registrar el evento, la clave, los intentos y el último error, y NUNCA un secreto. La escritura DEBE hacerse con cerrojo, fichero temporal en la misma carpeta y reemplazo atómico, bajo la guarda de contexto que corresponda, de modo que una escritura interrumpida no deje el fichero a medias. El cerrojo NO DEBE vivir en la carpeta del cambio sino en el directorio común de git (compartido por todos los worktrees), para que una cola vacía deje el árbol limpio. El fichero DEBE borrarse cuando quede sin operaciones. Política de reintento: el intento inmediato tiene un plazo total de 10 s por despacho, `sync` 30 s por operación y las operaciones pasan a `abandoned` tras 10 intentos o con un error permanente. Este requisito NO modifica REQ-25.9 ni REQ-25.17: mientras la outbox tenga operaciones pendientes o abandonadas, que `done` o `ensure` en `in_place` la traten como árbol sucio es aceptable y protege la cola.

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

### Requirement: `integrations.yaml` Versionado con Vínculos e Imputaciones (REQ-43.9)

La carpeta del cambio PUEDE contener `integrations.yaml`, con esquema versionado (`axiom.integrations/v1`), versionado y publicado por `axiom sdd publish` como cualquier documento de la raíz del cambio. SOLO DEBE contener vínculos (cambio, rol u hito con elemento externo) e imputaciones; NUNCA operaciones pendientes, intentos, errores ni secretos. Un vínculo se identifica por (plugin, tipo, id, rol) y una imputación por (plugin, origen, elemento): repetir la fusión no cambia el fichero. Un enlace cuya URL lleve credenciales (`userinfo`) o no sea http(s) DEBE rechazarse como error permanente de la operación, para que ninguna credencial llegue a disco versionado. La escritura DEBE seguir el mismo patrón que REQ-43.8. El fichero DEBE crearse únicamente cuando un plugin devuelve vínculos o imputaciones; sin ellos NO DEBE existir.

#### Scenario: Los vínculos devueltos se persisten
- **DADO** un plugin que devuelve el vínculo `{role: backend, item: "42"}`
- **CUANDO** la entrega tiene éxito
- **ENTONCES** `integrations.yaml` contiene ese vínculo y el campo de esquema `axiom.integrations/v1`

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

### Requirement: La Emisión Nunca Hace Fallar el Verbo de Origen (REQ-43.10)

Cuando un verbo determinista despacha un evento (REQ-43.22, REQ-43.25 y REQ-43.26), cualquier fallo de la integración (entrega, escritura de la outbox o de `integrations.yaml`, secreto no disponible) DEBE avisarse por stderr mediante un `io.Writer` sustituible, SIN cambiar el código de salida ni el contenido de stdout del verbo. Antes de despachar, el adaptador DEBE comprobar explícitamente `Declarations.IntegrationsErr` (porque `Locate` no rechaza declaraciones inválidas): si hay error, DEBE avisar por stderr y NO despachar.

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

### Requirement: `axiom integrations sync` Drena la Outbox (REQ-43.11)

`axiom integrations sync [--change <nombre>]` (con `--cwd`) DEBE reintentar, en orden de encolado, todas las operaciones pendientes de la outbox y retirar las entregadas, persistiendo los vínculos devueltos en la carpeta ordinaria (REQ-43.9). DEBE cubrir también las carpetas archivadas (REQ-43.29). Con `--change` trata la carpeta ordinaria y la carpeta archivada más reciente del cambio, si existen; sin `--change` recorre las carpetas ordinarias y las archivadas que tengan cola. `--retry-abandoned` reactiva las operaciones abandonadas (con los intentos a 0) y `--discard <clave>` (repetible) retira una operación concreta. DEBE terminar con código 0 solo si no queda ninguna operación pendiente o abandonada, y con código distinto de cero (y un resumen por stderr, sin secretos) si queda alguna. Sin ninguna cola existente (con o sin plugin configurado), DEBE terminar con código 0 sin crear ningún fichero. Con una cola existente cuyo plugin ya no está configurado o activo, DEBE dejar esas operaciones pendientes con el motivo `plugin_not_configured`, sin contar un intento, y terminar con código 1. Un bloque `integrations` inválido DEBE ser un error (código 1), no un aviso, porque `sync` es un verbo explícito de integración. Un fallo en una operación NO DEBE impedir reintentar las siguientes. La salida JSON es `axiom.integrations-sync/v1` con, por cambio, las operaciones entregadas, pendientes, abandonadas y descartadas (cada una con clave, plugin, intentos, último error saneado y motivo) y el total restante.

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

### Requirement: `axiom integrations emit` Dispara un Evento del Catálogo (REQ-43.12)

`axiom integrations emit --change <nombre> --event <evento>` (con `--cwd`; `--role <id>` para los eventos de rol) DEBE despachar un evento del catálogo con las reglas de REQ-43.5 a REQ-43.8, como respaldo explícito de los verbos deterministas. DEBE rechazar con error tipado y código distinto de cero: `change.start` (lo dispara C-07-d), un evento fuera del catálogo, un evento de rol sin `--role` o con un rol ausente del kickoff sellado, y un cambio inexistente. `emit` NO DEBE fabricar aprobaciones: los eventos de compuerta (`gate.spec.approved`, `gate.design.approved`, `role.apply.approved`) exigen que la última decisión de esa compuerta en `gates.yaml` sea `approved`, y toman de ese registro el digest, la huella y, si no se da `--actor`, el actor. `role.apply.approved` y `role.start` exigen `--role` del roster sellado. `emit` termina con código 0 si se entregó todo o si ningún plugin activo escucha el evento (con un mensaje que lo dice), y con código 1 si algo queda pendiente o abandonado; un bloque `integrations` inválido es un error (código 1). La salida JSON es `axiom.integrations-emit/v1` con la clave, las operaciones entregadas, pendientes y abandonadas y un indicador de que no había nada que hacer. `emit --event archive` DEBE exigir que exista la carpeta archivada del cambio y operar sobre ella (REQ-43.29). A diferencia de REQ-43.10, un fallo de entrega en `emit` deja la operación pendiente y el verbo termina con código distinto de cero (la persona lo invoca expresamente).

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

### Requirement: Almacén de Secretos sin Caída Silenciosa a Texto Plano (REQ-43.13)

El paquete `internal/integrations/secrets` DEBE definir el puerto `SecretStore` con un adaptador de entorno, un adaptador de keyring del sistema (`github.com/zalando/go-keyring`: Windows Credential Manager y llavero de macOS) y una cadena de resolución entorno → keyring. El adaptador de keyring se compila SOLO para Windows y macOS (`//go:build windows || darwin`): en Linux y en cualquier otro sistema no hay Secret Service y el único almacén es la variable de entorno. El adaptador de entorno lee `AXIOM_INTEGRATION_<PLUGIN>_<CLAVE>`, con el identificador del plugin en mayúsculas y `-` sustituido por `_` y el nombre del secreto en mayúsculas (`azure-devops` y `pat` dan `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`); una variable vacía cuenta como no definida y el catálogo rechaza dos pares que den la misma variable. El keyring usa el servicio `axiom` y la cuenta `<plugin>/<secreto>`. Un secreto ausente DEBE dar un error tipado que nombre `axiom integrations secret set` y la variable. NUNCA DEBE existir un almacén en texto plano ni una caída a él: si el keyring no está disponible y el entorno no aporta valor, DEBE devolverse un error tipado que nombre la variable de entorno alternativa. Los errores NUNCA DEBEN envolver el valor del secreto. Los secretos solo DEBEN declararse por nombre en el manifiesto (REQ-43.3).

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

### Requirement: Tipo `Secret` Redactado (REQ-43.14)

El tipo `Secret` DEBE redactar su valor en `String`, `GoString`, `MarshalJSON` y `MarshalYAML` (y en cualquier formateo con `fmt`, incluidos `%v`, `%+v` y `%#v`). El valor solo DEBE obtenerse mediante un método explícito que usa el plugin en el momento de entregar.

#### Scenario: Formateo redactado
- **DADO** un `Secret` con valor distintivo
- **CUANDO** se formatea con `%s`, `%v`, `%+v` y `%#v` y se serializa a JSON y a YAML
- **ENTONCES** ninguna salida contiene el valor y todas contienen el marcador de redacción

#### Scenario: Error que incluye el secreto
- **DADO** un plugin cuyo error de entrega se construye a partir de un contexto que contiene un `Secret`
- **CUANDO** el error llega a la outbox y a stderr
- **ENTONCES** no contiene el valor del secreto

### Requirement: `axiom integrations secret set|status|delete` (REQ-43.15)

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

### Requirement: Ningún Secreto Llega a Disco Versionado, a la Salida ni a un GET (REQ-43.16)

Un valor de secreto NO DEBE aparecer, tras ningún flujo del marco, en `axiom.yaml`, en ningún fichero de la carpeta del cambio (incluidos `integrations.yaml` y `.integrations-outbox.yaml`), en stdout ni en stderr de ningún verbo, ni en la respuesta de ningún endpoint GET del dashboard. Una prueba canario DEBE verificarlo: con un valor distintivo en el almacén simulado, ejecuta los flujos de despacho, fallo, reintento, `secret set|status|delete` y consulta los GET del dashboard, y comprueba la ausencia del valor en todos esos destinos.

#### Scenario: Canario tras un flujo completo
- **DADO** un valor distintivo en el almacén de secretos simulado y el doble que lo lee al entregar
- **CUANDO** se recorren `gate record`, una entrega fallida, `sync` y `secret status`
- **ENTONCES** el valor no aparece en `axiom.yaml`, en la carpeta del cambio, en stdout ni en stderr

#### Scenario: Canario en los GET del dashboard
- **DADO** la misma configuración
- **CUANDO** se consultan los endpoints GET del dashboard
- **ENTONCES** ninguna respuesta contiene el valor

### Requirement: Doble de Prueba en Memoria y Criterio de Terminado (REQ-43.17)

El paquete `internal/integrations/integrationstest` DEBE ofrecer un plugin en memoria que registra los eventos recibidos (con su clave), devuelve vínculos configurables y puede fallar a voluntad (siempre, N veces o por evento). Debe poder inyectarse en el catálogo de manifiestos y en el registro mediante las costuras de prueba. Una prueba de extremo a extremo DEBE demostrar el criterio de terminado del backlog: un plugin de prueba registrado y configurado en `axiom.yaml` recibe los eventos de `gate record` (spec, design y role-apply), de `branch ensure --role`, de `sdd publish --archive` y de `integrations emit`; persiste vínculos en `integrations.yaml`; y reintenta desde la outbox con `sync`.

#### Scenario: Recorrido de todos los verbos
- **DADO** un workspace con `integrations.<doble>` configurado y el doble registrado
- **CUANDO** se ejecutan `gate record --decision approved` para `spec`, `design` y `role-apply:<rol>`, `branch ensure --role <rol>`, `sdd publish --archive` y `integrations emit`
- **ENTONCES** el doble ha recibido, con su clave de idempotencia, `gate.spec.approved`, `gate.design.approved`, `role.apply.approved`, `role.start`, `archive` y el evento emitido
- **Y** `integrations.yaml` contiene los vínculos que devolvió (salvo los de `archive`, REQ-43.30)

#### Scenario: Reintento tras fallo
- **DADO** el doble configurado para fallar una vez
- **CUANDO** falla la entrega de un evento y después se ejecuta `integrations sync`
- **ENTONCES** la operación se entrega en el reintento, se retira de la outbox y `sync` termina con código 0

### Requirement: Alta del Verbo `integrations` en la CLI y la Ayuda (REQ-43.18)

La CLI DEBE exponer la familia de primer nivel `axiom integrations` con exactamente los subcomandos `sync`, `emit`, `secret` (este con `set`, `status` y `delete`), `connect` y `discover`. La ayuda global y la ayuda de cada subcomando DEBEN describirlos y las pruebas de ayuda DEBEN cubrirlos. Todo subcomando acepta `--cwd`; los que operan sobre un cambio, `--change`.

(Previously: la familia tenía exactamente los subcomandos `sync`, `emit` y `secret`; `connect` y `discover` los añaden REQ-45.12 y REQ-45.13)

#### Scenario: Ayuda global
- **CUANDO** se ejecuta `axiom --help`
- **ENTONCES** el texto lista `integrations` con `sync`, `emit`, `secret`, `connect` y `discover`

#### Scenario: Subcomando desconocido
- **CUANDO** se ejecuta `axiom integrations inspect`
- **ENTONCES** termina con código distinto de cero y muestra la ayuda de la familia, que lista `sync`, `emit`, `secret`, `connect` y `discover`
### Requirement: `archive` se Encola de Forma Duradera en la Carpeta Archivada (REQ-43.28)

El evento `archive` DEBE seguir el mismo despacho duradero que el resto de eventos (REQ-43.6), pero la operación DEBE encolarse en `.integrations-outbox.yaml` en la raíz de la carpeta archivada del cambio (`openspec/changes/archive/<fecha>-<cambio>/`), porque `sdd publish --archive` se ejecuta después de mover el incremento. La cola archivada es estado local (REQ-27.5 modificado): NO se versiona ni se publica. La escritura de la cola archivada y de sus temporales DEBE pasar por una guarda que autorice, bajo `openspec/changes/archive/`, solo el nombre de la cola y sus temporales en la raíz exacta de una carpeta archivada (REQ-21.18 modificado). Sin repositorio git, el despacho de `archive` NO DEBE encolar: avisa por stderr e indica la orden `axiom integrations emit --event archive`.

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
- **ENTONCES** la escritura se rechaza con el mensaje de REQ-21.18 y no se modifica nada

#### Scenario: Sin git no se encola
- **DADO** un workspace que no está en un repositorio git y un cambio archivado
- **CUANDO** se despacha `archive`
- **ENTONCES** no se crea ninguna cola y stderr avisa e indica `axiom integrations emit --event archive`

### Requirement: `sync` y `emit --event archive` Cubren las Carpetas Archivadas (REQ-43.29)

`axiom integrations sync` DEBE tratar también las colas de las carpetas archivadas, con la misma política de reintento que las ordinarias (REQ-43.8), aplicando a cada carpeta la guarda de su contexto (ordinario o archivado). `axiom integrations emit --change <x> --event archive` DEBE exigir que exista la carpeta archivada del cambio, rechazar con error tipado y sin crear ficheros si no existe, y encolar y entregar sobre ella como REQ-43.28. Con `--change` y una carpeta ordinaria y una archivada, `sync` DEBE tratar ambas; en la carpeta archivada, `sync` NO DEBE fusionar vínculos (REQ-43.30).

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

### Requirement: El Evento `archive` Nunca Fusiona Vínculos en `integrations.yaml` (REQ-43.30)

Ni el despacho inmediato ni `sync` ni `emit` DEBEN persistir en `integrations.yaml` los vínculos o imputaciones que el plugin devuelva al entregar `archive`: se descartan con un aviso por stderr. `integrations.yaml` de la carpeta archivada solo se lee, para dar los vínculos existentes al plugin (`Delivery.Links`). Para los demás eventos, REQ-43.9 no cambia.

#### Scenario: Vínculos devueltos por `archive` se descartan
- **DADO** un doble que devuelve un vínculo al entregar `archive`
- **CUANDO** se despacha, o se reintenta con `sync`
- **ENTONCES** `integrations.yaml` archivado no cambia
- **Y** stderr avisa del descarte

#### Scenario: El plugin recibe los vínculos existentes
- **DADO** un `integrations.yaml` archivado con un vínculo
- **CUANDO** se entrega `archive`
- **ENTONCES** el plugin recibe ese vínculo en `Delivery.Links`

### Requirement: `done` en la Rama de Archivado se Niega con Cola Pendiente (REQ-43.31)

Mientras la cola de la carpeta archivada tenga operaciones pendientes o abandonadas, `axiom branch done` sobre la rama de archivado DEBE negarse por la regla de árbol sucio vigente (REQ-25.17), que NO cambia: la cola protege las operaciones evitando que se elimine el worktree con trabajo sin entregar. Con la cola vacía (y por tanto borrada) `done` NO DEBE negarse por ella.

#### Scenario: `done` se niega con cola archivada pendiente
- **DADO** una cola `archive` pendiente en la carpeta archivada del worktree de archivado
- **CUANDO** se ejecuta `axiom branch done` sobre la rama de archivado
- **ENTONCES** se rechaza por árbol sucio y no se elimina el worktree

#### Scenario: `done` tras vaciar la cola
- **DADO** la misma cola, entregada con `axiom integrations sync`
- **CUANDO** se ejecuta `axiom branch done`
- **ENTONCES** no se rechaza por la cola
### Requirement: Puertos Opcionales de Identidad y Descubrimiento (REQ-45.17)

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

