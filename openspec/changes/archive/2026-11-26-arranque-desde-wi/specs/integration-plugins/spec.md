# Delta para Marco de Integraciones (integration-plugins)

> **Cambio:** `arranque-desde-wi` (C-07-d). **Idioma:** castellano de España; identificadores, comandos y claves en inglés.
> El puerto de work items sigue el patrón de RF-INT-23: aserción de tipo sobre el plugin, sin cambiar la interfaz `Plugin`.

## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: `integrations.yaml` Versionado con Vínculos e Imputaciones (RF-INT-09)

La carpeta del cambio PUEDE contener `integrations.yaml`, con esquema versionado (`axiom.integrations/v1`), versionado y publicado por `axiom sdd publish` como cualquier documento de la raíz del cambio. SOLO DEBE contener vínculos (cambio, rol u hito con elemento externo) e imputaciones; NUNCA operaciones pendientes, intentos, errores ni secretos. Un vínculo se identifica por (plugin, tipo, id, rol) y una imputación por (plugin, origen, elemento): repetir la fusión no cambia el fichero. Un enlace cuya URL lleve credenciales (`userinfo`) o no sea http(s) DEBE rechazarse como error permanente de la operación, para que ninguna credencial llegue a disco versionado. La escritura DEBE seguir el mismo patrón que RF-INT-08. El fichero DEBE crearse únicamente cuando un plugin devuelve vínculos o imputaciones, o cuando `axiom change start --adopt` vincula el work item del cambio; sin ellos NO DEBE existir. El vínculo del work item del cambio DEBE tener `kind: work_item`, `plugin`, `id`, `url` y `source: change.start`, se escribe por una función de vinculación exportada distinta del despacho de entregas, que reutiliza la fusión de vínculos bajo el mismo cerrojo que las entregas, y NO cambia el esquema `axiom.integrations/v1`; el carril ODD NO escribe `integrations.yaml`.

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

#### Scenario: El vínculo del work item se escribe con `source: change.start`
- **DADO** un cambio sin `integrations.yaml` y una adopción de `axiom change start`
- **CUANDO** se vincula el work item 2525 del plugin `azure-devops`
- **ENTONCES** `integrations.yaml` se crea con el esquema `axiom.integrations/v1` y un vínculo `kind: work_item`, `plugin: azure-devops`, `id: "2525"`, la URL y `source: change.start`

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
