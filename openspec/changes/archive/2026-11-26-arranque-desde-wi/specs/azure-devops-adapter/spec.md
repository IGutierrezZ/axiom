# Delta para Adaptador de Azure DevOps (azure-devops-adapter)

> **Cambio:** `arranque-desde-wi` (C-07-d). **Idioma:** castellano de España; identificadores, comandos y claves en inglés.
> El adaptador sigue sin emitir eventos del ciclo de vida (C-07-e) ni crear work items. `change.start` continúa declarado como evento escuchado; este cambio no lo emite. El valor por defecto de `item_keys` lo fija el diseño; su contrato mínimo (formato, validación y rechazo del tipo sin mapear) se especifica aquí.
> La capacidad `tags` cubre la etiqueta de vuelta `axiom:<cambio>` y `attachments` la descarga de adjuntos; `relations` sigue sin declararse porque no se escriben relaciones.

## ADDED Requirements

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

## MODIFIED Requirements

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
