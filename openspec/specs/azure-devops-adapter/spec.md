# Especificación de azure-devops-adapter

## Purpose

Primer adaptador real del marco de integraciones (C-07-c): el plugin `azure-devops`, su manifiesto, un cliente REST 7.1 único y robusto (autenticación por PAT, timeouts, límites de tasa, errores tipados y saneados), la lectura de work items, la escritura protegida por `/rev`, el descubrimiento del proceso del proyecto, la identidad de la persona en Azure DevOps y los verbos `axiom integrations connect` y `discover`. No emite ningún evento del ciclo de vida hacia Azure DevOps (C-07-d y C-07-e), no crea work items, no ofrece interfaz (C-07-h), no usa Entra ID y no soporta Azure DevOps Server on-premises. Todo se verifica con un servidor `httptest`: ninguna prueba llama a Azure DevOps real.

## Requirements

### Requirement: Manifiesto y Alta del Plugin `azure-devops` (RF-ADO-01)

El paquete `internal/integrations/azuredevops` DEBE ofrecer el plugin `azure-devops`. Su manifiesto DEBE declarar el identificador `azure-devops`, las capacidades `items.read`, `discovery`, `tags` y `attachments`, el evento `change.start` como único evento escuchado, las claves de configuración `organization`, `project`, `team` e `item_keys`, el secreto `pat`, la identidad `{source: plugin, secret: pat}` y el alias de referencia `ado`, que es el prefijo de la `external_ref` (`ado:<id>`) y que NO DEBE estar escrito como constante en el verbo de arranque. El manifiesto DEBE superar la validación de RF-INT-03 y ser idéntico a la entrada del catálogo cerrado (RF-INT-04). La variable de entorno del secreto DEBE ser `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`. El manifiesto NO DEBE declarar capacidades que este cambio no entrega.

#### Scenario: El manifiesto es válido e idéntico al del catálogo
- **DADO** el plugin `azure-devops` y el catálogo de manifiestos por defecto
- **CUANDO** se valida el manifiesto y se construye un registro con el plugin
- **ENTONCES** la validación termina sin error y el registro se construye
- **Y** el manifiesto declara exactamente las capacidades `items.read`, `discovery`, `tags` y `attachments`

#### Scenario: Variable de entorno del secreto
- **DADO** el manifiesto de `azure-devops`
- **CUANDO** se consulta la variable de entorno del secreto `pat`
- **ENTONCES** es `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`

#### Scenario: Ninguna capacidad de escritura del ciclo de vida
- **DADO** el manifiesto de `azure-devops`
- **CUANDO** se enumeran sus capacidades
- **ENTONCES** no incluye `items.transition`, `items.create`, `relations`, `assignment`, `estimation`, `time` ni `vcs_links`

#### Scenario: Alias de referencia y clave `item_keys` en el manifiesto
- **DADO** el manifiesto de `azure-devops`
- **CUANDO** se consultan su alias de referencia y sus claves de configuración
- **ENTONCES** el alias es `ado` y las claves incluyen `item_keys`
- **Y** el verbo de arranque obtiene el alias del manifiesto y no de una constante propia

### Requirement: Configuración `integrations.azure-devops` y Secreto `pat` (RF-ADO-02)

El bloque `integrations.azure-devops` de `axiom.yaml` DEBE aceptar las claves `organization`, `project`, `team` e `item_keys` (y la reservada `enabled`) y NO DEBE contener secretos ni datos personales. `item_keys` DEBE ser una lista de cadenas `Tipo=Prefijo`, donde `Tipo` es el nombre del tipo de work item tal y como lo devuelve el tracker y `Prefijo` es un token ASCII alfanumérico no vacío y corto, apto como inicio de un nombre de rama; un elemento malformado, un `Tipo` o un `Prefijo` duplicado o un `Prefijo` no válido DEBE ser un error de carga que nombra la clave y el elemento. Sin `item_keys`, DEBE aplicarse el valor por defecto documentado en el diseño y en `axiom.example.yaml`. Una clave desconocida DEBE ser un error de carga. El PAT DEBE obtenerse únicamente del almacén de secretos del marco de integraciones (keyring en Windows y macOS, solo entorno en Linux) en el momento de cada llamada, nunca de `axiom.yaml`. La URL base DEBE construirse como `https://dev.azure.com/{organization}` desde `organization` y NO DEBE ser configurable.

#### Scenario: Bloque válido
- **DADO** un `axiom.yaml` con `integrations.azure-devops` que declara `organization`, `project` y `team`
- **CUANDO** se carga el workspace
- **ENTONCES** la carga termina sin error

#### Scenario: Clave desconocida
- **DADO** un `axiom.yaml` con `integrations.azure-devops.base_url`
- **CUANDO** se carga el workspace
- **ENTONCES** la carga falla con un error que nombra la clave `base_url`

#### Scenario: Clave que parece un secreto
- **DADO** un `axiom.yaml` con `integrations.azure-devops.pat`
- **CUANDO** se carga el workspace
- **ENTONCES** la carga falla con un error tipado

#### Scenario: El PAT se resuelve en cada llamada
- **DADO** una fuente de secretos sustituible que cuenta las consultas y el cliente configurado con ella
- **CUANDO** el cliente realiza dos peticiones
- **ENTONCES** la fuente se consulta en cada una de ellas

#### Scenario: Secreto ausente
- **DADO** una fuente de secretos que no contiene `pat`
- **CUANDO** el cliente intenta realizar una petición
- **ENTONCES** devuelve un error tipado que nombra el secreto `pat` y la variable `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`
- **Y** no se envía ninguna petición HTTP

#### Scenario: `item_keys` válido
- **DADO** un `axiom.yaml` con `item_keys: ["User Story=US", "Bug=BUG"]`
- **CUANDO** se carga el workspace
- **ENTONCES** la carga termina sin error y el tipo `User Story` tiene el prefijo `US`

#### Scenario: `item_keys` malformado
- **DADO** un `axiom.yaml` con `item_keys: ["User Story"]`
- **CUANDO** se carga el workspace
- **ENTONCES** la carga falla con un error que nombra `item_keys` y el elemento `User Story`

#### Scenario: Prefijo o tipo duplicado
- **DADO** un `axiom.yaml` con `item_keys: ["User Story=US", "Bug=US"]` o con dos entradas del mismo tipo
- **CUANDO** se carga el workspace
- **ENTONCES** la carga falla con un error que nombra la clave y el elemento duplicado

#### Scenario: Prefijo no apto para una rama
- **DADO** un `axiom.yaml` con `item_keys: ["Bug=B/U..G"]`
- **CUANDO** se carga el workspace
- **ENTONCES** la carga falla con un error que nombra el prefijo

### Requirement: Cliente REST con Autenticación, Versión y Timeout (RF-ADO-03)

El cliente REST DEBE tener un único punto de salida HTTP que concentre autenticación, timeout, espera, reintentos y saneado de errores. Toda petición DEBE llevar `api-version=7.1`, autenticación `Basic` con `:{PAT}` en base64, y un timeout propio por petición. El cliente DEBE respetar la cancelación del `ctx` en cualquier espera, incluida la de reintentos. La URL base y el `*http.Client` DEBEN poder inyectarse para las pruebas.

#### Scenario: Versión y autenticación en toda petición
- **DADO** un servidor `httptest` que registra las peticiones y un PAT conocido
- **CUANDO** el cliente realiza una lectura
- **ENTONCES** la petición lleva el parámetro `api-version=7.1`
- **Y** lleva la cabecera `Authorization: Basic` con el base64 de `:{PAT}`

#### Scenario: Timeout por petición
- **DADO** un servidor que no responde dentro del timeout configurado
- **CUANDO** el cliente realiza una lectura
- **ENTONCES** la petición termina con un error de red tipado tras vencer el timeout
- **Y** el error no contiene el PAT

#### Scenario: Cancelación durante una espera
- **DADO** un servidor que responde 429 con `Retry-After` de 30 segundos y un reloj sustituible
- **CUANDO** se cancela el `ctx` durante la espera
- **ENTONCES** el cliente devuelve el error de cancelación sin enviar más peticiones

#### Scenario: Organización con caracteres reservados
- **DADO** una organización cuyo nombre contiene `/` o `?`
- **CUANDO** se construye la URL base
- **ENTONCES** el nombre se escapa y la petición no sale de `dev.azure.com/{organization}`

### Requirement: Límites de Tasa y Reintentos Acotados (RF-ADO-04)

El cliente DEBE respetar `Retry-After` también en una respuesta HTTP 200, esperando antes de la siguiente petición. DEBE reintentar las respuestas 429 y 503, y los errores de red transitorios, con un presupuesto de intentos acotado y retroceso exponencial con tope; cuando la respuesta trae `Retry-After`, la espera NO DEBE ser menor que ese valor. DEBE leer y exponer las cabeceras `X-RateLimit-Delay`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` y `X-RateLimit-Limit`, y NO DEBE usar `X-RateLimit-Resource` en ningún cálculo. Agotado el presupuesto, DEBE devolver un error tipado de límite de tasa que indica el número de intentos. Las esperas DEBEN usar un reloj sustituible para que las pruebas no duerman.

#### Scenario: Reintento tras 429
- **DADO** un servidor que responde 429 con `Retry-After: 2` a la primera petición y 200 a la segunda, y un reloj sustituible
- **CUANDO** el cliente realiza una lectura
- **ENTONCES** el cliente espera al menos 2 segundos en el reloj sustituible
- **Y** la lectura termina con éxito tras dos peticiones

#### Scenario: `Retry-After` en una respuesta 200
- **DADO** un servidor que responde 200 con `Retry-After: 3` a la primera lectura
- **CUANDO** el cliente realiza dos lecturas seguidas
- **ENTONCES** la segunda petición no se envía hasta que el reloj sustituible ha avanzado al menos 3 segundos

#### Scenario: Reintento tras 503 con retroceso
- **DADO** un servidor que responde 503 dos veces y luego 200
- **CUANDO** el cliente realiza una lectura
- **ENTONCES** la lectura termina con éxito tras tres peticiones
- **Y** las esperas entre intentos crecen de forma exponencial sin superar el tope

#### Scenario: Presupuesto agotado
- **DADO** un servidor que responde siempre 429
- **CUANDO** el cliente realiza una lectura
- **ENTONCES** devuelve un error tipado de límite de tasa tras el máximo de intentos y no más
- **Y** el error indica el número de intentos realizados

#### Scenario: Cabeceras de tasa expuestas
- **DADO** un servidor que responde 200 con `X-RateLimit-Delay`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` y `X-RateLimit-Limit`
- **CUANDO** el cliente realiza una lectura
- **ENTONCES** el cliente expone los cuatro valores leídos
- **Y** una cabecera `X-RateLimit-Resource` presente no altera ninguna espera

#### Scenario: Error de red transitorio
- **DADO** un servidor que cierra la conexión en la primera petición y responde 200 en la segunda
- **CUANDO** el cliente realiza una lectura
- **ENTONCES** la lectura termina con éxito tras dos peticiones

### Requirement: Errores Tipados y Saneados con Scope Probable (RF-ADO-05)

Todo fallo del cliente DEBE ser un error tipado, nunca silenciado. El cuerpo de error de Azure DevOps DEBE incluirse acotado en tamaño y saneado; el error NO DEBE contener cabeceras ni la petición. Una respuesta 401, una 203 con página de inicio de sesión y una 403 DEBEN producir un error de autenticación que nombra el scope probable de la operación (`vso.work_write` para work items, `vso.project` para proyectos, equipos y miembros, `vso.work` para iteraciones, `vso.code` para repositorios) y recuerda que los PAT globales se retiran. Los códigos 4xx distintos de los anteriores no se reintentan.

#### Scenario: 401 con scope probable
- **DADO** un servidor que responde 401 a una lectura de work item
- **CUANDO** el cliente realiza la lectura
- **ENTONCES** devuelve un error de autenticación que nombra `vso.work_write`
- **Y** no se reintenta la petición

#### Scenario: 203 con página de inicio de sesión
- **DADO** un servidor que responde 203 con una página HTML de inicio de sesión
- **CUANDO** el cliente realiza una lectura de miembros del equipo
- **ENTONCES** devuelve un error de autenticación que nombra `vso.project`
- **Y** no intenta interpretar la página como JSON

#### Scenario: 403 en repositorios
- **DADO** un servidor que responde 403 a la lista de repositorios
- **CUANDO** el cliente la solicita
- **ENTONCES** devuelve un error de autenticación que nombra `vso.code`

#### Scenario: Cuerpo de error acotado
- **DADO** un servidor que responde 500 con un cuerpo de varios megabytes
- **CUANDO** el cliente realiza una lectura
- **ENTONCES** el error incluye como máximo el tamaño acotado del cuerpo
- **Y** el error no contiene cabeceras ni la petición

#### Scenario: Cuerpo de error que contiene el PAT
- **DADO** un servidor que responde 400 con un cuerpo que repite el PAT y su forma base64
- **CUANDO** el cliente realiza una lectura
- **ENTONCES** el error devuelto no contiene el PAT ni su base64

#### Scenario: JSON inválido en una respuesta 200
- **DADO** un servidor que responde 200 con un cuerpo que no es JSON
- **CUANDO** el cliente realiza una lectura
- **ENTONCES** devuelve un error tipado de respuesta inválida y no un valor vacío

### Requirement: Lectura de Work Items y Lotes (RF-ADO-06)

El cliente DEBE leer un work item con `GET .../_apis/wit/workitems/{id}?$expand=relations` y devolver su identificador, revisión (`rev`), campos y relaciones. DEBE leer varios con `POST .../_apis/wit/workitemsbatch`, troceando la lista en lotes del máximo que admite la API y conservando el orden de los identificadores pedidos. Un work item inexistente DEBE producir un error tipado de no encontrado.

#### Scenario: Lectura con relaciones
- **DADO** un servidor con el work item 42 en la revisión 7 con una relación
- **CUANDO** el cliente lo lee
- **ENTONCES** la petición lleva `$expand=relations`
- **Y** el resultado trae `rev` 7, sus campos y la relación

#### Scenario: Lote troceado
- **DADO** una lista de identificadores superior al máximo de un lote y un servidor que registra cada `POST`
- **CUANDO** el cliente los lee en lote
- **ENTONCES** el servidor recibe varios `POST`, cada uno con no más del máximo de identificadores
- **Y** el resultado conserva el orden de los identificadores pedidos

#### Scenario: Work item inexistente
- **DADO** un servidor que responde 404 al work item 999
- **CUANDO** el cliente lo lee
- **ENTONCES** devuelve un error tipado de no encontrado que nombra el identificador 999
- **Y** no se reintenta

#### Scenario: Lote vacío
- **DADO** una lista de identificadores vacía
- **CUANDO** el cliente la lee en lote
- **ENTONCES** devuelve un resultado vacío sin enviar ninguna petición

### Requirement: Escritura con Concurrencia Optimista `test /rev` (RF-ADO-07)

Toda modificación de un work item DEBE enviarse con `PATCH .../_apis/wit/workitems/{id}` y `Content-Type: application/json-patch+json`, y su primera operación DEBE ser siempre `{"op":"test","path":"/rev","value":N}` con la revisión leída. Cualquier rechazo de esa prueba (HTTP 412, o 400/409 con el error de test de Azure DevOps) DEBE devolverse como `ErrRevConflict`: el cliente NO DEBE reintentarlo, NO DEBE releer para reaplicar y NO DEBE modificar el work item. Una petición de modificación sin la operación `test /rev` NO DEBE poder construirse. Un `PATCH` aplicado cuya respuesta se pierde y se reintenta falla por `/rev` y se informa como conflicto, nunca se aplica dos veces.

#### Scenario: Escritura con `/rev` vigente
- **DADO** un work item en la revisión 7 y un servidor que valida `test /rev`
- **CUANDO** el cliente modifica el título con `rev` 7
- **ENTONCES** el `PATCH` lleva el tipo `application/json-patch+json` y su primera operación es `test /rev` con valor 7
- **Y** el work item queda en la revisión 8 con el título nuevo

#### Scenario: `/rev` obsoleto
- **DADO** un work item que el servidor ya tiene en la revisión 9
- **CUANDO** el cliente intenta modificarlo con `rev` 7
- **ENTONCES** devuelve `ErrRevConflict`
- **Y** el servidor no registra ninguna modificación
- **Y** el cliente envía una sola petición

#### Scenario: Variantes del código de rechazo
- **DADO** servidores que rechazan la operación `test` con 412, con 400 y con 409 con el error de test
- **CUANDO** el cliente intenta la modificación en cada caso
- **ENTONCES** en los tres casos devuelve `ErrRevConflict` sin reintentar

#### Scenario: Un 400 ajeno al test no es conflicto
- **DADO** un servidor que responde 400 por un campo inexistente
- **CUANDO** el cliente intenta la modificación
- **ENTONCES** devuelve un error de petición inválida distinto de `ErrRevConflict`

#### Scenario: Reintento tras respuesta perdida
- **DADO** un servidor que aplica el `PATCH` pero corta la conexión antes de responder
- **CUANDO** el cliente reintenta por el error de red transitorio
- **ENTONCES** el segundo envío falla por `/rev` y el cliente devuelve `ErrRevConflict`
- **Y** el servidor aplicó la modificación una sola vez

#### Scenario: Sin operación de test no hay petición
- **DADO** una modificación construida sin revisión
- **CUANDO** se intenta construir la petición
- **ENTONCES** se rechaza con un error tipado y no se envía nada

### Requirement: Repositorios Git con Caché por Proceso (RF-ADO-08)

El cliente DEBE listar los repositorios git del proyecto con `_apis/git/repositories`, devolviendo nombre y GUID, y DEBE cachear el resultado durante la vida del proceso: una segunda consulta del mismo proyecto NO DEBE enviar petición. Un fallo NO DEBE cachearse.

#### Scenario: Listado de repositorios
- **DADO** un servidor con dos repositorios en el proyecto
- **CUANDO** el cliente los lista
- **ENTONCES** devuelve los dos con su nombre y su GUID

#### Scenario: Segunda consulta desde la caché
- **DADO** un cliente que ya listó los repositorios del proyecto
- **CUANDO** los lista de nuevo
- **ENTONCES** el servidor no recibe ninguna petición adicional

#### Scenario: Un fallo no se cachea
- **DADO** un servidor que responde 500 a la primera consulta y 200 a la segunda
- **CUANDO** el cliente lista los repositorios dos veces
- **ENTONCES** la primera devuelve error y la segunda devuelve el listado

### Requirement: Descubrimiento del Proceso del Proyecto (RF-ADO-09)

El descubrimiento DEBE devolver, con un esquema versionado: los tipos de work item del proyecto; para cada tipo, sus estados con la categoría tal como la devuelve la API (`Proposed`, `InProgress`, `Resolved`, `Completed`, `Removed`, sin normalizar); para cada tipo, sus campos; las iteraciones del equipo (a partir de `values`, con fechas y `timeFrame`); y los miembros del equipo con su `id` y `displayName`, obtenidos con paginación `$top/$skip` hasta agotarlos. Las sugerencias de asignación DEBEN salir de la lista de miembros; el código NO DEBE contener direcciones de correo.

#### Scenario: Descubrimiento completo
- **DADO** un servidor con dos tipos de work item, estados, campos, dos iteraciones y tres miembros
- **CUANDO** se ejecuta el descubrimiento
- **ENTONCES** el resultado contiene los dos tipos, los estados y campos de cada uno, las dos iteraciones con fechas y `timeFrame`, y los tres miembros con `id` y `displayName`
- **Y** el resultado lleva el campo de esquema versionado

#### Scenario: Categoría del estado sin normalizar
- **DADO** un tipo cuyos estados tienen las categorías `InProgress` y `Removed`
- **CUANDO** se ejecuta el descubrimiento
- **ENTONCES** las categorías se devuelven exactamente como las envía la API

#### Scenario: Miembros paginados
- **DADO** un servidor con más miembros que el tamaño de página
- **CUANDO** se ejecuta el descubrimiento
- **ENTONCES** el cliente pide páginas con `$top` y `$skip` hasta recibir una página incompleta
- **Y** el resultado contiene todos los miembros sin repetidos

#### Scenario: Iteraciones vacías
- **DADO** un equipo sin iteraciones
- **CUANDO** se ejecuta el descubrimiento
- **ENTONCES** el resultado trae una lista de iteraciones vacía y no un error

#### Scenario: Fallo parcial
- **DADO** un servidor que responde 403 a la lista de miembros y 200 al resto
- **CUANDO** se ejecuta el descubrimiento
- **ENTONCES** el descubrimiento falla con un error de autenticación que nombra `vso.project`
- **Y** no devuelve un resultado parcial como si fuera completo

### Requirement: Resolución de la Identidad de la Persona (RF-ADO-10)

Al conectar, el adaptador DEBE intentar `GET _apis/connectionData` y, si responde con un `authenticatedUser` utilizable (con identificador y nombre visible), esa es la identidad y su origen es `connection_data`. Si la petición falla o no trae un usuario utilizable, la identidad DEBE resolverse entre los miembros del equipo mediante el usuario que la persona indique de forma explícita (`--user`), con origen `member_selection`. Con `--user` indicado, el adaptador NO DEBE consultar `connectionData`. El usuario indicado debe coincidir con el `id` de un miembro; si no coincide, error tipado que nombra los candidatos. NO DEBE haber pregunta interactiva. El contrato de `connectionData` con un PAT real queda marcado como no verificado: solo se prueba con `httptest`.

#### Scenario: Identidad por `connectionData`
- **DADO** un servidor cuyo `connectionData` devuelve un `authenticatedUser` con identificador y nombre
- **CUANDO** se resuelve la identidad sin `--user`
- **ENTONCES** la identidad es la del `authenticatedUser` con origen `connection_data`
- **Y** el servidor no recibe la consulta de miembros

#### Scenario: `connectionData` falla
- **DADO** un servidor cuyo `connectionData` responde 401 y que tiene miembros
- **CUANDO** se resuelve la identidad sin `--user`
- **ENTONCES** la resolución termina con un error tipado que lista los miembros candidatos con su `id` y `displayName` y pide repetir con `--user`
- **Y** no se solicita ninguna entrada interactiva

#### Scenario: `connectionData` sin usuario utilizable
- **DADO** un servidor cuyo `connectionData` responde 200 con `authenticatedUser` vacío
- **CUANDO** se resuelve la identidad sin `--user`
- **ENTONCES** se comporta como en el caso de fallo: lista de miembros y petición de `--user`

#### Scenario: Elección explícita con `--user`
- **DADO** un servidor con el miembro `abc-123` y el usuario `abc-123` indicado
- **CUANDO** se resuelve la identidad con `--user abc-123`
- **ENTONCES** la identidad es la de ese miembro con origen `member_selection`
- **Y** el servidor no recibe ninguna consulta a `connectionData`

#### Scenario: `--user` desconocido
- **DADO** un servidor cuyos miembros no incluyen `zzz`
- **CUANDO** se resuelve la identidad con `--user zzz`
- **ENTONCES** devuelve un error tipado que nombra `zzz` y los candidatos
- **Y** no se guarda ninguna identidad

### Requirement: Fichero de Identidad por Persona Indexado por Organización (RF-ADO-11)

La identidad resuelta DEBE guardarse únicamente en `~/.axiom/` de la persona (respetando `AXIOM_STATE_DIR` mediante `system.AxiomDir`), en un fichero propio junto a `workspaces.json` que NO DEBE versionarse, con esquema versionado, escritura atómica y modo 0600, indexado por organización (una entrada por organización de Azure DevOps). Cada entrada DEBE llevar el `id` de Azure DevOps, el `displayName`, el origen (`connection_data` o `member_selection`) y la fecha. El fichero NO DEBE contener el PAT ni ningún secreto y NO DEBE alterar el esquema de `workspaces.json`. Guardar la identidad de una organización NO DEBE alterar las entradas de las demás.

#### Scenario: Primera identidad guardada
- **DADO** un `AXIOM_STATE_DIR` temporal sin fichero de identidad
- **CUANDO** se guarda la identidad de la organización `contoso`
- **ENTONCES** el fichero existe en el directorio de estado con una entrada `contoso` con `id`, `displayName`, origen y fecha
- **Y** en sistemas con permisos POSIX el modo es 0600

#### Scenario: Varias organizaciones
- **DADO** un fichero con la entrada de `contoso`
- **CUANDO** se guarda la identidad de `fabrikam`
- **ENTONCES** el fichero contiene ambas entradas y la de `contoso` no cambia

#### Scenario: Reconexión sustituye la entrada
- **DADO** un fichero con la entrada de `contoso` con origen `member_selection`
- **CUANDO** se vuelve a conectar y se resuelve por `connectionData`
- **ENTONCES** la entrada de `contoso` pasa a `connection_data` con la fecha nueva

#### Scenario: Escritura atómica
- **DADO** una escritura que falla a mitad
- **CUANDO** se inspecciona el directorio de estado
- **ENTONCES** el fichero anterior permanece íntegro y no queda ningún temporal

#### Scenario: Sin secretos ni cambios en `workspaces.json`
- **CUANDO** se guarda una identidad
- **ENTONCES** el contenido del fichero no incluye el PAT ni su base64
- **Y** `workspaces.json` no cambia

#### Scenario: Fichero corrupto
- **DADO** un fichero de identidad con JSON inválido
- **CUANDO** se intenta guardar una identidad
- **ENTONCES** se devuelve un error tipado que nombra el fichero y no se sobrescribe el contenido ilegible

### Requirement: Verbo `axiom integrations connect` (RF-ADO-12)

`axiom integrations connect --plugin <id> [--user <id>] [--cwd <ruta>] [--json]` DEBE probar la conexión con el PAT del almacén de secretos, resolver la identidad (RF-ADO-10) y guardarla (RF-ADO-11). La salida DEBE nombrar la organización, el identificador y el nombre visible, y el origen; con `--json` DEBE ser un documento con esquema versionado. Sin `connectionData` utilizable y sin `--user`, el verbo DEBE listar los miembros del equipo y terminar con código distinto de cero pidiendo repetir con `--user`, sin pregunta interactiva. Un `--plugin` desconocido, un bloque `integrations.<plugin>` ausente, un plugin con `enabled: false` o un error de configuración DEBEN ser un error tipado del verbo, no un aviso. Si el plugin no ofrece el puerto de identidad, el error DEBE ser el de RF-INT-23.

#### Scenario: Conexión por `connectionData`
- **DADO** un workspace con `integrations.azure-devops` configurado, el PAT en el entorno y un servidor `httptest` que devuelve `authenticatedUser`
- **CUANDO** se ejecuta `axiom integrations connect --plugin azure-devops --json`
- **ENTONCES** termina con código 0 y la salida lleva el esquema versionado, la organización, el `id`, el `displayName` y el origen `connection_data`
- **Y** el fichero de identidad contiene la entrada de la organización

#### Scenario: Sin identidad utilizable y sin `--user`
- **DADO** un servidor cuyo `connectionData` responde 401
- **CUANDO** se ejecuta `axiom integrations connect --plugin azure-devops`
- **ENTONCES** termina con código distinto de cero y la salida lista los miembros con `id` y `displayName` y pide repetir con `--user`
- **Y** no se guarda ninguna identidad

#### Scenario: Repetición con `--user`
- **DADO** el caso anterior
- **CUANDO** se ejecuta `axiom integrations connect --plugin azure-devops --user <id>` con el `id` de un miembro
- **ENTONCES** termina con código 0 y guarda la identidad con origen `member_selection`

#### Scenario: Bloque ausente
- **DADO** un workspace sin `integrations.azure-devops`
- **CUANDO** se ejecuta `axiom integrations connect --plugin azure-devops`
- **ENTONCES** termina con código distinto de cero y un error que indica que el plugin no está configurado

#### Scenario: Plugin desactivado
- **DADO** un workspace con `integrations.azure-devops.enabled: false`
- **CUANDO** se ejecuta `axiom integrations connect --plugin azure-devops`
- **ENTONCES** termina con código distinto de cero y un error tipado que indica que el plugin está desactivado

#### Scenario: Plugin desconocido
- **CUANDO** se ejecuta `axiom integrations connect --plugin jira`
- **ENTONCES** termina con código distinto de cero y un error que nombra los plugins del catálogo

#### Scenario: Ayuda del verbo
- **CUANDO** se ejecuta `axiom integrations --help`
- **ENTONCES** la lista cerrada de subcomandos incluye `connect` y `discover`

### Requirement: Verbo `axiom integrations discover` (RF-ADO-13)

`axiom integrations discover --plugin <id> [--cwd <ruta>] [--json]` DEBE imprimir el descubrimiento de RF-ADO-09 del plugin indicado. Con `--json` DEBE emitir un documento con esquema versionado que contiene tipos, estados con categoría, campos por tipo, iteraciones y miembros. Sin `--json`, una salida legible equivalente. El verbo NO DEBE escribir ningún fichero. Los errores de configuración y de autorización son errores tipados del verbo. Si el plugin no ofrece el puerto de descubrimiento, el error DEBE ser el de RF-INT-23.

#### Scenario: Descubrimiento en JSON
- **DADO** un workspace con `integrations.azure-devops` configurado, el PAT en el entorno y un servidor `httptest` completo
- **CUANDO** se ejecuta `axiom integrations discover --plugin azure-devops --json`
- **ENTONCES** termina con código 0 y la salida es un documento JSON con esquema versionado, tipos, estados con categoría, campos por tipo, iteraciones y miembros

#### Scenario: Salida legible
- **CUANDO** se ejecuta `axiom integrations discover --plugin azure-devops` sin `--json`
- **ENTONCES** la salida nombra los tipos, los estados con su categoría, las iteraciones y los miembros

#### Scenario: Sin escritura de ficheros
- **CUANDO** se ejecuta `axiom integrations discover --plugin azure-devops`
- **ENTONCES** no se crea ni se modifica ningún fichero del workspace ni de `~/.axiom/`

#### Scenario: Fallo de autorización
- **DADO** un servidor que responde 403 a los miembros
- **CUANDO** se ejecuta `axiom integrations discover --plugin azure-devops`
- **ENTONCES** termina con código distinto de cero y un error que nombra `vso.project`

### Requirement: `Deliver` sin Efectos del Ciclo de Vida (RF-ADO-14)

Este cambio NO DEBE emitir ningún evento del ciclo de vida hacia Azure DevOps. El manifiesto declara solo `change.start`, que ningún verbo emite en este cambio. Si `Deliver` se invocara, DEBE devolver un error permanente tipado sin realizar ninguna petición HTTP ni escribir ficheros. Con el bloque `integrations.azure-devops` configurado y activo, los verbos que despachan eventos (`gate record`, `branch ensure --role`, `sdd publish --archive`) DEBEN producir la misma salida y no crear ficheros, porque ningún plugin activo escucha sus eventos (RF-INT-05).

#### Scenario: `Deliver` devuelve error permanente
- **DADO** el plugin `azure-devops` y un servidor `httptest` que falla la prueba si recibe alguna petición
- **CUANDO** se invoca `Deliver` con un evento `change.start`
- **ENTONCES** devuelve un error permanente tipado
- **Y** el servidor no recibe ninguna petición

#### Scenario: Verbos de despacho sin cambios
- **DADO** un workspace con `integrations.azure-devops` configurado y activo
- **CUANDO** se ejecuta `axiom sdd gate record --decision approved` sobre `spec`
- **ENTONCES** stdout, stderr y código de salida coinciden con los de un workspace sin el bloque
- **Y** la carpeta del cambio no contiene `integrations.yaml` ni `.integrations-outbox.yaml`

### Requirement: El PAT No Aparece en Ninguna Salida (RF-ADO-15)

El PAT y su forma codificada en la cabecera `Authorization: Basic` (base64 de `:{PAT}`) NO DEBEN aparecer en stdout, stderr, errores, logs, el fichero de identidad, `axiom.yaml` ni la salida de `discover`. El cliente NO DEBE volcar cabeceras ni peticiones en errores o logs, y DEBE redactar además el valor codificado. Una prueba canario DEBE usar un PAT reconocible y buscar él y su base64 en todos esos destinos.

#### Scenario: Canario en el flujo completo
- **DADO** un PAT canario reconocible en el entorno, un servidor `httptest` que acepta solo esa credencial y un workspace temporal
- **CUANDO** se ejecutan `connect` (con y sin `--user`) y `discover`, incluidos los caminos de error 401, 403, 429 y 500
- **ENTONCES** ni el canario ni su base64 aparecen en stdout, stderr, los errores devueltos, el fichero de identidad, `axiom.yaml` ni la salida de `discover`

#### Scenario: Eco del servidor
- **DADO** un servidor que devuelve en el cuerpo de error la cabecera `Authorization` recibida
- **CUANDO** el cliente falla con ese error
- **ENTONCES** el error devuelto redacta tanto el PAT como su base64

#### Scenario: El PAT sale del almacén
- **DADO** un PAT presente solo en el almacén de secretos o en `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`
- **CUANDO** el cliente realiza una petición
- **ENTONCES** la cabecera de autenticación se construye con ese valor y ningún otro origen es consultado

### Requirement: Pruebas Herméticas sin Azure DevOps Real (RF-ADO-16)

Ninguna prueba de este cambio DEBE llamar a Azure DevOps real ni usar credenciales reales: todas usan un servidor `httptest` local, un reloj y esperas sustituibles y un almacén de secretos sustituible. Las pruebas NO DEBEN compilar binarios sueltos y se ejecutan con `go test`. El servidor de pruebas `adotest` DEBE cubrir, además de la lectura y la escritura de work items, las relaciones (padre y Related, y la operación `add /relations/-`), el campo `System.Tags`, la revisión para `test /rev` y los adjuntos binarios (incluidos los servidos sin `Content-Length`, los que superan un límite y las redirecciones), y las pruebas que lo usan DEBEN fallar ante cualquier petición a un host no local.

#### Scenario: Servidor local inyectado
- **DADO** el cliente configurado con la URL base de un `httptest.Server` y su `*http.Client`
- **CUANDO** se ejecuta la batería de pruebas del paquete
- **ENTONCES** toda petición HTTP va al servidor local y ninguna a `dev.azure.com`

#### Scenario: Reintentos sin dormir
- **DADO** un reloj sustituible
- **CUANDO** se ejecutan las pruebas de reintento
- **ENTONCES** ninguna prueba espera tiempo real superior a una fracción de segundo

#### Scenario: `adotest` simula relaciones, etiquetas y adjuntos
- **DADO** el servidor `adotest` con un WI con padre, Related, etiquetas y adjuntos binarios
- **CUANDO** el adaptador lee el WI, descarga un adjunto y añade una etiqueta
- **ENTONCES** `adotest` sirve las relaciones, aplica `System.Tags` con control de revisión y entrega el binario sin recurrir a un servicio real

#### Scenario: Un host no local hace fallar la prueba
- **DADO** una prueba con un transporte que registra el destino de cada petición
- **CUANDO** el código bajo prueba intenta una petición a un host distinto del servidor local
- **ENTONCES** la prueba falla
### Requirement: Lectura Tipada del Work Item con Relaciones, Etiquetas y Adjuntos (RF-ADO-17)

El adaptador MUST implementar el puerto de work items traduciendo la respuesta del tracker (`GetWorkItem` con relaciones) al tipo neutral del marco: tipo (`System.WorkItemType`), título, estado (`System.State`) con su categoría tomada del descubrimiento del proceso del proyecto, URL, descripción (`System.Description`), pasos de reproducción y criterios de aceptación (campos del proceso, cuando existan), etiquetas (`System.Tags`, separadas por `;` y recortadas), padre (relación `Hierarchy-Reverse`), relaciones Related y adjuntos (`AttachedFile`, con nombre y tamaño declarado). Una relación de tipo desconocido MUST ignorarse sin error. Un campo ausente MUST producir un valor vacío sin error. Los identificadores de relaciones MUST derivarse únicamente de URL del propio host y la organización configurados. La categoría del estado MUST obtenerse con una sola consulta de descubrimiento por invocación del verbo.

#### Scenario: Traducción de un work item completo
- **DADO** un servidor `adotest` con un WI con tipo, estado, etiquetas `a; b`, un padre, una relación Related y dos adjuntos
- **CUANDO** se lee por el puerto
- **ENTONCES** el resultado neutral contiene cada dato, las etiquetas `a` y `b` y los dos adjuntos con nombre y tamaño

#### Scenario: Categoría del estado desde el descubrimiento
- **DADO** un estado `Resolved` que el descubrimiento asigna a `Completed`
- **CUANDO** se lee el WI
- **ENTONCES** la categoría del resultado es `Completed`

#### Scenario: Relación desconocida y campos ausentes
- **DADO** un WI con una relación de un tipo no soportado y sin pasos de reproducción
- **CUANDO** se lee por el puerto
- **ENTONCES** la relación se ignora y los pasos quedan vacíos, sin error

#### Scenario: Relación con URL de otro host
- **DADO** una relación cuya URL apunta a otro host
- **CUANDO** se lee por el puerto
- **ENTONCES** no se solicita ninguna URL ajena con el PAT

### Requirement: Descarga Binaria Acotada con el PAT Restringido (RF-ADO-18)

El adaptador MUST descargar adjuntos con una petición binaria propia que reutilice el punto de salida HTTP del cliente (autenticación, timeout, esperas, reintentos y saneado de errores), sin redirecciones, con `api-version=7.1` cuando proceda y con las comprobaciones previas de esquema `https`, ausencia de `userinfo`, host configurado y prefijo `/{organización}/` antes de construir la petición con credenciales. MUST leer el cuerpo con un lector limitado por el límite que le pasa el llamador (10 MiB por fichero; el presupuesto de 50 MiB por WI lo aplica el llamador) y MUST abortar al superarlo, sin conservar datos parciales. Una URL que no cumpla las comprobaciones MUST devolver un error tipado sin petición de red. Una respuesta no 2xx MUST producir un error tipado y saneado, sin el PAT ni cabeceras de autenticación.

#### Scenario: Descarga correcta
- **DADO** un adjunto bajo la organización configurada en `adotest`
- **CUANDO** se descarga con un límite de 1 MiB y el contenido ocupa 100 KiB
- **ENTONCES** se devuelven los bytes completos y la petición lleva la autenticación `Basic` del PAT

#### Scenario: Superar el límite aborta
- **DADO** un adjunto servido sin `Content-Length` que supera el límite
- **CUANDO** se descarga
- **ENTONCES** se devuelve un error tipado de tamaño, no se devuelven datos parciales y la lectura se corta al límite

#### Scenario: Host ajeno
- **DADO** una URL de adjunto de un host distinto del configurado
- **CUANDO** se descarga
- **ENTONCES** se devuelve un error tipado y el servidor local no recibe ninguna petición

#### Scenario: Redirección no seguida
- **DADO** un adjunto cuyo servidor responde 302 hacia otro host
- **CUANDO** se descarga
- **ENTONCES** la redirección no se sigue y el error tipado no contiene el PAT

#### Scenario: Error del servidor saneado
- **DADO** un servidor que responde 500 con un cuerpo que incluye el PAT
- **CUANDO** se descarga
- **ENTONCES** el error devuelto no contiene el PAT

### Requirement: Enlace de Vuelta por Etiqueta Idempotente sobre `test /rev` (RF-ADO-19)

El adaptador MUST añadir la etiqueta `axiom:<cambio>` a `System.Tags` del WI con `UpdateWorkItem` protegido por `test /rev`, releyendo antes el WI para conservar el resto de etiquetas (separadas por `;`) y sin duplicar la etiqueta si ya figura. Si el WI ya contiene la etiqueta, MUST NOT enviar ninguna escritura. Si la revisión cambió entre la lectura y la escritura (conflicto de `test /rev`), MUST releer y reintentar un número acotado de veces y, si persiste, devolver un error tipado de conflicto. MUST NOT escribir relaciones ni Hyperlinks (C-07-f) ni tocar otros campos.

#### Scenario: Se añade la etiqueta conservando las demás
- **DADO** un WI con la etiqueta `a` y la revisión 4
- **CUANDO** se enlaza el cambio `x`
- **ENTONCES** se envía un único `PATCH` con `test /rev` igual a 4 y `System.Tags` igual a `a; axiom:x`

#### Scenario: Etiqueta ya presente
- **DADO** un WI que ya contiene `axiom:x`
- **CUANDO** se enlaza el cambio `x`
- **ENTONCES** no se envía ninguna escritura

#### Scenario: Conflicto de revisión
- **DADO** un servidor que rechaza el primer `PATCH` por revisión obsoleta y acepta el segundo tras una relectura
- **CUANDO** se enlaza el cambio `x`
- **ENTONCES** el resultado es correcto con una sola etiqueta y sin pérdida de las etiquetas de la otra persona

#### Scenario: Conflicto persistente
- **DADO** un servidor que rechaza siempre por revisión obsoleta
- **CUANDO** se enlaza el cambio
- **ENTONCES** se devuelve un error tipado de conflicto tras el número acotado de reintentos

#### Scenario: Sin relaciones ni Hyperlinks
- **DADO** un enlace de vuelta correcto
- **CUANDO** se inspeccionan las operaciones enviadas
- **ENTONCES** ninguna es `add /relations/-` ni modifica un campo distinto de `System.Tags`

