# Delta para `local-web-dashboard`

## ADDED Requirements

### Requirement: Cabeceras de Seguridad Generales con CSP Fija, sin CORS y sin Respuesta Genérica a `OPTIONS` (prov:WEB:1)

Toda respuesta del servidor del dashboard DEBE llevar una `Content-Security-Policy` general con exactamente estas directivas: `default-src 'self'`, `script-src 'self'`, `img-src 'self' data:`, `font-src 'self'`, `connect-src 'self'`, `object-src 'none'`, `base-uri 'none'`, `form-action 'self'` y `frame-ancestors 'none'`, más una directiva `style-src` (cuyo valor fija el diseño). La política NO DEBE incluir `'unsafe-inline'` ni `'unsafe-eval'` en `script-src`, ni ningún origen externo. El script de tema de `index.html` DEBE cargarse desde un fichero propio embebido, de forma síncrona en `<head>`, y `index.html` NO DEBE contener ningún `<script>` con código en línea.

La ruta `/api/explainer` DEBE conservar su CSP propia (RF-WEB-15), que prevalece sobre la general y NO DEBE duplicarse en la cabecera. Ninguna respuesta DEBE llevar cabeceras `Access-Control-*`. El servidor NO DEBE responder `200` de forma genérica a `OPTIONS`: una petición `OPTIONS` DEBE responderse sin cabeceras `Access-Control-*`.

#### Scenario: Toda respuesta lleva la CSP general
- **DADO** el servidor en ejecución
- **CUANDO** un cliente consulta `GET /`, un asset estático y una ruta `GET /api/...` cualquiera
- **ENTONCES** cada respuesta lleva una `Content-Security-Policy` con las nueve directivas fijas indicadas
- **Y** `script-src` vale exactamente `'self'`

#### Scenario: Ninguna respuesta lleva cabeceras CORS
- **DADO** el servidor en ejecución
- **CUANDO** un cliente envía una petición con `Origin` ajeno a una ruta `GET` y a una ruta que muta
- **ENTONCES** ninguna respuesta contiene cabecera alguna que empiece por `Access-Control-`

#### Scenario: `OPTIONS` no responde con CORS ni con un 200 genérico
- **DADO** el servidor en ejecución
- **CUANDO** un cliente envía `OPTIONS` a una ruta de la API
- **ENTONCES** la respuesta no contiene cabeceras `Access-Control-*`
- **Y** el servidor no la trata como una petición aceptada para el origen que la envía

#### Scenario: El script de tema ya no está en línea
- **DADO** el `index.html` embebido
- **CUANDO** se inspecciona su contenido
- **ENTONCES** no contiene ningún `<script>` con cuerpo en línea
- **Y** el script de tema se carga con `src` propio, de forma síncrona, antes de la hoja de estilos de tokens

#### Scenario: `/api/explainer` conserva su CSP propia
- **DADO** una petición a `/api/explainer` que sirve un resumen
- **CUANDO** se leen las cabeceras de la respuesta
- **ENTONCES** lleva la CSP propia de RF-WEB-15 una sola vez
- **Y** no se le añade la CSP general

### Requirement: `Host` y `Origin` Locales en las Rutas que Mutan y en Todas las Rutas `/api/integrations*` (prov:WEB:2)

Toda ruta que muta estado (cualquier método distinto de `GET`, `HEAD` y `OPTIONS`) y toda ruta `/api/integrations*`, también con `GET`, DEBEN exigir que la cabecera `Host` designe un destino local (bucle local o el `Host` del propio servidor) y que, si la petición lleva `Origin`, este sea local. Una petición con `Host` u `Origin` ajeno DEBE rechazarse con `403` antes de leer el cuerpo, de resolver secretos o de escribir nada. Esta comprobación se suma a la de petición local de RF-WEB-10.

#### Scenario: `Origin` ajeno a una ruta que muta se rechaza
- **DADO** el servidor en ejecución
- **CUANDO** un cliente envía `PUT /api/settings` con `Origin: https://sitio-ajeno.invalid`
- **ENTONCES** el servidor responde `403`
- **Y** `axiom.yaml` no se modifica

#### Scenario: `Host` ajeno a una ruta que muta se rechaza
- **DADO** el servidor en ejecución
- **CUANDO** un cliente envía una petición que muta con una cabecera `Host` que no es local
- **ENTONCES** el servidor responde `403` antes de leer el cuerpo

#### Scenario: `Host` ajeno a `GET /api/integrations` se rechaza
- **DADO** un plugin de integraciones activo con un PAT guardado
- **CUANDO** un cliente envía `GET /api/integrations` con `Host` ajeno
- **ENTONCES** el servidor responde `403`
- **Y** la respuesta no contiene datos de integración alguno

#### Scenario: `Origin` ajeno a `GET /api/integrations/...` se rechaza
- **DADO** el servidor en ejecución
- **CUANDO** un cliente envía `GET` a cualquier ruta `/api/integrations*` con `Origin` ajeno
- **ENTONCES** el servidor responde `403`

#### Scenario: Una petición local sin `Origin` se atiende
- **DADO** una petición con `Host` local y sin cabecera `Origin`
- **CUANDO** el cliente consulta `GET /api/integrations`
- **ENTONCES** el servidor la atiende con normalidad

### Requirement: `GET /api/integrations` Devuelve Manifiesto, Valores, Estado del Secreto e Identidad sin Revelar Nunca un Secreto (prov:WEB:3)

`GET /api/integrations` DEBE devolver, para cada plugin de integración disponible, su manifiesto (campos de configuración, tipos de work item, eventos, estados y jornada), los valores actuales de `integrations.<plugin>` leídos de `axiom.yaml`, el estado del secreto y la identidad resuelta persistida. El estado del secreto DEBE limitarse a su procedencia (`source`), con uno de los valores `env`, `keyring`, `missing` o `keyring_unavailable`, y NUNCA incluir el valor del secreto, un fragmento ni su longitud. El descubrimiento de proyectos, tipos y estados DEBE servirse bajo demanda y no en la consulta general.

La respuesta NO DEBE contener el PAT ni ningún valor derivado de él, tampoco en los mensajes de error. Cuando no hay proyecto activo o `axiom.yaml` no existe, DEBE responder `404` con código tipado; para métodos distintos de `GET` DEBE responder `405`.

#### Scenario: El estado del secreto es solo la procedencia
- **DADO** un PAT guardado en el almacén del sistema
- **CUANDO** el cliente consulta `GET /api/integrations`
- **ENTONCES** el plugin informa `source: keyring`
- **Y** el cuerpo no contiene el valor del PAT ni ningún fragmento

#### Scenario: Una variable de entorno que tapa el almacén se indica
- **DADO** la variable de entorno del PAT definida y un PAT guardado en el almacén
- **CUANDO** el cliente consulta `GET /api/integrations`
- **ENTONCES** `source` vale `env`

#### Scenario: Sin PAT disponible
- **DADO** ni variable de entorno ni PAT guardado
- **CUANDO** el cliente consulta `GET /api/integrations`
- **ENTONCES** `source` vale `missing`

#### Scenario: Almacén no disponible
- **DADO** un sistema donde el almacén de secretos no está disponible
- **CUANDO** el cliente consulta `GET /api/integrations`
- **ENTONCES** `source` vale `keyring_unavailable`

#### Scenario: Centinela de secreto no aparece en la respuesta
- **DADO** un PAT centinela guardado y una identidad persistida
- **CUANDO** el cliente consulta `GET /api/integrations`
- **ENTONCES** el cuerpo no contiene el centinela
- **Y** incluye la identidad resuelta persistida

#### Scenario: Sin proyecto activo
- **DADO** que no hay proyecto activo o que su `axiom.yaml` no existe
- **CUANDO** el cliente consulta `GET /api/integrations`
- **ENTONCES** responde `404` con código tipado

#### Scenario: Métodos no admitidos
- **DADO** el servidor en ejecución
- **CUANDO** el cliente envía `POST`, `PUT` o `DELETE` a `/api/integrations`
- **ENTONCES** responde `405`

### Requirement: Guardar y Eliminar el PAT por Endpoints Propios que Responden Solo `source`, con Guía de Variable de Entorno cuando No Hay Almacén (prov:WEB:4)

El dashboard DEBE ofrecer un endpoint propio para guardar el PAT (`POST`, solo JSON, cuerpo acotado, petición local con `Host` y `Origin` locales) y otro para eliminarlo. El PAT DEBE guardarse únicamente en el almacén de secretos del sistema y las dos operaciones DEBEN responder solo con la procedencia (`source`) resultante, nunca con el valor. El PAT NO DEBE escribirse en `axiom.yaml`, en ningún otro fichero versionado, en logs ni en ninguna respuesta HTTP, y NO DEBE ser clave del catálogo de ajustes.

Cuando el sistema no tiene un almacén escribible (por ejemplo, Linux sin servicio de claves), el guardado DEBE fallar con un error tipado que no simule éxito y que guíe a definir la variable de entorno derivada del manifiesto del plugin (para Azure DevOps, `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`); la interfaz DEBE mostrar esa guía. Si una variable de entorno tapa el valor del almacén, la respuesta DEBE reflejarlo con `source: env`. Un cuerpo que no sea JSON, que supere el límite o que no incluya el valor DEBE rechazarse con `400` en castellano sin tocar el almacén.

#### Scenario: Guardar el PAT responde solo `source`
- **DADO** una petición local con un PAT centinela válido
- **CUANDO** el cliente envía el `POST` de guardado
- **ENTONCES** responde `200` con `source: keyring`
- **Y** el cuerpo no contiene el centinela

#### Scenario: El PAT no llega a `axiom.yaml`
- **DADO** un PAT guardado desde el dashboard
- **CUANDO** se inspeccionan `axiom.yaml` y los ficheros versionados del repositorio
- **ENTONCES** ninguno contiene el PAT
- **Y** `axiom.yaml` solo puede contener la procedencia declarada (`source`), nunca el valor

#### Scenario: Linux sin almacén escribible guía a la variable de entorno
- **DADO** un sistema donde el almacén no admite escritura (`ErrNoWritableStore`)
- **CUANDO** el cliente envía el guardado del PAT
- **ENTONCES** responde con un error tipado en castellano que nombra la variable de entorno derivada del manifiesto
- **Y** no se guarda nada ni se simula un guardado, y la interfaz muestra esa guía

#### Scenario: Eliminar el PAT
- **DADO** un PAT guardado en el almacén
- **CUANDO** el cliente envía la eliminación
- **ENTONCES** responde con la procedencia resultante (`missing` o `env`) sin ningún valor

#### Scenario: Petición con `Origin` ajeno no guarda
- **DADO** una petición de guardado con `Origin` ajeno
- **CUANDO** el servidor la procesa
- **ENTONCES** responde `403` y el almacén no cambia

#### Scenario: Cuerpo inválido o vacío
- **DADO** un cuerpo no JSON, vacío o por encima del límite
- **CUANDO** el cliente envía el guardado
- **ENTONCES** responde `400` en castellano y el almacén no cambia

#### Scenario: El PAT no es clave editable
- **DADO** el catálogo de ajustes
- **CUANDO** se busca cualquier clave de PAT o secreto
- **ENTONCES** no existe, y una escritura por `PUT /api/settings` que la nombre se rechaza como clave desconocida

### Requirement: «Probar Conexión» Resuelve y Persiste la Identidad con la Misma Semántica que `axiom integrations connect` (prov:WEB:5)

El dashboard DEBE ofrecer una ruta para probar la conexión de un plugin con la misma semántica que `axiom integrations connect`: resolver la identidad con el secreto disponible, mostrarla y persistirla en la identidad local no versionada del usuario (`~/.axiom/integration-identities.json`), nunca en `axiom.yaml`. La ruta DEBE exigir petición local con `Host` y `Origin` locales y DEBE acotar su duración con un plazo propio inferior a la espera de escritura del servidor.

Cuando la resolución produce varios candidatos (`IdentitySelectionError`), la respuesta DEBE devolverlos y la interfaz DEBE permitir elegir uno y reenviar la elección, que se persiste. Cuando falla la autenticación, el plazo vence o no hay PAT, la respuesta DEBE ser un error tipado en castellano sin revelar el secreto.

#### Scenario: Conexión correcta muestra y persiste la identidad
- **DADO** un PAT válido y un doble del servicio de work items que resuelve una identidad
- **CUANDO** el cliente prueba la conexión
- **ENTONCES** la respuesta contiene la identidad resuelta
- **Y** la identidad queda persistida en el almacén local de identidades y no en `axiom.yaml`

#### Scenario: Varios candidatos permiten elegir
- **DADO** un doble que devuelve varias identidades candidatas
- **CUANDO** el cliente prueba la conexión
- **ENTONCES** la respuesta lista los candidatos
- **Y** al reenviar la elección de uno, esa identidad queda persistida

#### Scenario: Sin PAT disponible
- **DADO** ningún PAT disponible (`source: missing`)
- **CUANDO** el cliente prueba la conexión
- **ENTONCES** responde con un error tipado en castellano que indica guardar el PAT
- **Y** no se llama al servicio externo

#### Scenario: Plazo agotado
- **DADO** un doble que no responde dentro del plazo de la ruta
- **CUANDO** el cliente prueba la conexión
- **ENTONCES** responde con un error tipado de plazo agotado en castellano antes de que venza la espera de escritura del servidor

#### Scenario: Petición con `Host` ajeno
- **DADO** una petición con `Host` ajeno
- **CUANDO** el cliente prueba la conexión
- **ENTONCES** responde `403` sin resolver ningún secreto

### Requirement: «Descubrir» Rellena los Desplegables con el Descubrimiento del Plugin (prov:WEB:6)

El dashboard DEBE ofrecer una ruta de descubrimiento que, bajo demanda y con el secreto disponible, devuelva las opciones que el plugin descubre (por ejemplo proyectos, tipos de work item y estados) para rellenar los desplegables de la pantalla de integraciones. La ruta DEBE exigir `Host` y `Origin` locales, acotar su duración con un plazo propio y devolver errores tipados en castellano sin revelar el secreto. Descubrir NO DEBE escribir `axiom.yaml`: la configuración solo se escribe con `PUT /api/settings`.

#### Scenario: Descubrimiento correcto
- **DADO** un PAT válido y un doble de descubrimiento con proyectos, tipos y estados
- **CUANDO** el cliente pide descubrir
- **ENTONCES** la respuesta contiene esas opciones
- **Y** `axiom.yaml` no se modifica

#### Scenario: Sin PAT disponible
- **DADO** ningún PAT disponible
- **CUANDO** el cliente pide descubrir
- **ENTONCES** responde con un error tipado en castellano que indica guardar el PAT

#### Scenario: Plazo agotado
- **DADO** un doble que no responde en plazo
- **CUANDO** el cliente pide descubrir
- **ENTONCES** responde con un error tipado de plazo agotado y los desplegables conservan su valor anterior

### Requirement: Pantalla «Integraciones» Generada desde el Manifiesto del Plugin, Genérica por Plugin (prov:WEB:7)

El dashboard DEBE ofrecer una pantalla «Integraciones» con entrada en la navegación y ruta por *hash*, generada desde el manifiesto que devuelve `GET /api/integrations` sin conocer ningún plugin concreto. Por cada plugin DEBE ofrecer: activación (`integrations[<plugin>].enabled`), conexión (guardar y eliminar el PAT, mostrar `source`, «Probar conexión» e identidad), y los campos del manifiesto (tipos, eventos, estados, tags y jornada) con un control acorde a su tipo, obligatoriedad, valores permitidos, formato y límites. Los valores se guardan por `PUT /api/settings` en un lote validado por el decodificador estricto (RF-WEB-19). Todo texto del manifiesto, de la configuración y del servicio externo DEBE insertarse como texto, nunca como HTML.

La pantalla NO DEBE duplicar los controles de `branch_template`: DEBE enlazar a «Ajustes del proyecto». Cuando el almacén no es escribible, DEBE mostrar la guía de la variable de entorno y no ofrecer un guardado simulado. Cuando `source` es `env`, DEBE indicar que la variable tapa el almacén.

#### Scenario: La pantalla se genera desde el manifiesto
- **DADO** un manifiesto con un campo de enumeración, uno obligatorio y uno numérico con mínimo y máximo
- **CUANDO** se abre «Integraciones»
- **ENTONCES** el campo de enumeración se pinta como selector con exactamente sus valores, el obligatorio se marca como tal y el numérico respeta su mínimo y máximo
- **Y** la pantalla no contiene literales del plugin que no vengan del manifiesto

#### Scenario: Activar y configurar solo desde la UI
- **DADO** un proyecto sin bloque `integrations`
- **CUANDO** la persona activa el plugin, completa los campos obligatorios y guarda
- **ENTONCES** `axiom.yaml` contiene `integrations.<plugin>` con esos valores y vuelve a cargarse como `COMPLIANT` con el decodificador estricto

#### Scenario: Un valor inválido no pierde lo tecleado
- **DADO** un valor rechazado por el validador
- **CUANDO** la persona guarda
- **ENTONCES** la pantalla muestra el error en castellano y conserva lo tecleado

#### Scenario: Entrada de navegación y ruta propia
- **DADO** la interfaz cargada
- **CUANDO** el usuario selecciona «Integraciones» o abre su ruta por *hash*
- **ENTONCES** se muestra la pantalla sin recargar y la entrada aparece marcada como activa

#### Scenario: Las plantillas de rama no se duplican
- **DADO** la pantalla abierta
- **CUANDO** se recorren sus controles
- **ENTONCES** no existe ningún control de `branch_template`
- **Y** hay un enlace a «Ajustes del proyecto»

#### Scenario: Linux sin almacén muestra la guía y no simula guardado
- **DADO** una respuesta del servidor que indica almacén no escribible
- **CUANDO** la persona intenta guardar el PAT
- **ENTONCES** la pantalla muestra la variable de entorno derivada del manifiesto
- **Y** no muestra el PAT como guardado

#### Scenario: Variable de entorno que tapa el almacén
- **DADO** `source: env`
- **CUANDO** se abre la conexión
- **ENTONCES** la pantalla indica que la variable de entorno tiene prioridad sobre el almacén

#### Scenario: Texto hostil del manifiesto o del servicio externo se muestra como texto
- **DADO** una etiqueta o un nombre descubierto con `<script>` en su valor
- **CUANDO** se abre la pantalla
- **ENTONCES** el valor aparece literal y no se interpreta como HTML

### Requirement: Pestaña «Integraciones» en el Detalle del Cambio con WI, Tasks por Rol, Outbox y Actores (prov:WEB:8)

El detalle del cambio DEBE incluir una pestaña «Integraciones», también para cambios archivados (`Folder`), que muestre: el work item enlazado (identificador, título si existe y enlace); por cada rol, su Task enlazada con la estimación y las imputaciones de tiempo registradas y el tiempo pendiente de imputar; las operaciones pendientes de la outbox del cambio (de solo lectura); y los actores por fase (quién selló el kickoff, quién decidió cada puerta y quién imputó cada tiempo). La pestaña NO DEBE ejecutar ni reintentar operaciones de la outbox ni modificar vínculos. Sin vínculos ni plugin activo, DEBE mostrar un estado vacío explícito. Los datos NO DEBEN contener secretos y todo texto DEBE insertarse como texto.

#### Scenario: Cambio con WI y Tasks por rol
- **DADO** un cambio enlazado a un work item con dos roles que tienen Task, estimación y dos imputaciones
- **CUANDO** se abre la pestaña «Integraciones»
- **ENTONCES** se muestra el WI enlazado y, por rol, su Task con su estimación y sus imputaciones
- **Y** se muestra el tiempo pendiente de imputar

#### Scenario: Outbox pendiente
- **DADO** un cambio con operaciones pendientes en la outbox
- **CUANDO** se abre la pestaña
- **ENTONCES** se listan esas operaciones sin ofrecer ninguna acción sobre ellas

#### Scenario: Actores por fase
- **DADO** un kickoff sellado por una persona y puertas decididas por otras
- **CUANDO** se abre la pestaña
- **ENTONCES** se muestra quién selló, quién decidió cada puerta y quién imputó cada tiempo

#### Scenario: Cambio archivado
- **DADO** un cambio archivado con vínculos
- **CUANDO** se abre su detalle
- **ENTONCES** la pestaña muestra los mismos datos leídos de su carpeta archivada

#### Scenario: Cambio sin integración
- **DADO** un cambio sin vínculos
- **CUANDO** se abre la pestaña
- **ENTONCES** se muestra un estado vacío que lo explica y no se muestra ningún error

#### Scenario: Ningún secreto en los datos de la pestaña
- **DADO** un PAT centinela guardado
- **CUANDO** se consulta el detalle del cambio
- **ENTONCES** la respuesta no contiene el centinela

### Requirement: Alta de Incremento con Work Item de Origen: Campo ID/URL con Plugin Activo, Mismas Fases que `change start` y Fallos con Salidas Explícitas (prov:WEB:9)

Con un plugin de work items activo, el alta de incremento del dashboard DEBE mostrar un campo de identificador o URL del work item, y el alta DEBE ejecutar las mismas fases que `axiom change start` (fase 1, preparación de la rama con el nombre derivado, creación guardada de la carpeta del cambio y fase 2), sin reimplementar su lógica. Como el alta manual, el alta con work item NO DEBE sellar `kickoff.yaml`: el estilo de ejecución es una decisión de gobierno que el dashboard no pregunta, y el kickoff se sella después al retomar el cambio. Sin plugin activo, el alta DEBE comportarse exactamente como antes y NO DEBE mostrar el campo.

Un rechazo de la fase 1 NO DEBE dejar rama, carpeta del cambio ni `kickoff.yaml`, y DEBE mostrarse con su mensaje y su continuación. Un fallo posterior a la fase 1 DEBE conservar el *staging* y DEBE ofrecer dos acciones: reintentar con el mismo nombre derivado sin repetir la fase 1, o descartar. El descarte NO DEBE ejecutarse nunca de forma automática: solo por una acción explícita de la persona. Una advertencia de la fase 2 con éxito DEBE mostrarse y no detener el flujo.

#### Scenario: Alta con WI crea el cambio con el nombre derivado
- **DADO** un plugin activo y un doble que resuelve el WI
- **CUANDO** la persona da su identificador en el alta
- **ENTONCES** se crea el cambio con el nombre derivado de la fase 1
- **Y** el cambio queda enlazado al WI

#### Scenario: Alta con URL del WI
- **DADO** un plugin activo
- **CUANDO** la persona pega la URL del WI
- **ENTONCES** se obtiene el mismo cambio que con su identificador

#### Scenario: Sin plugin activo el alta no cambia
- **DADO** un proyecto sin plugin de work items activo
- **CUANDO** se abre el alta de incremento
- **ENTONCES** no aparece el campo de work item
- **Y** el alta funciona como antes

#### Scenario: Rechazo de la fase 1 no deja rastro
- **DADO** un WI que la fase 1 rechaza
- **CUANDO** se intenta el alta
- **ENTONCES** se muestra su mensaje y su continuación
- **Y** no existe rama, carpeta del cambio ni `kickoff.yaml`

#### Scenario: Fallo posterior conserva el staging y ofrece reintentar o descartar
- **DADO** un fallo de la preparación de la rama o de la fase 2 tras una fase 1 correcta
- **CUANDO** se muestra el resultado
- **ENTONCES** el *staging* se conserva
- **Y** se ofrecen «Reintentar» (mismo nombre derivado, sin repetir la fase 1) y «Descartar»

#### Scenario: El descarte solo se ejecuta por acción explícita
- **DADO** un fallo posterior a la fase 1
- **CUANDO** la persona no pulsa «Descartar»
- **ENTONCES** el *staging* no se descarta en ningún momento

#### Scenario: Reintentar no repite la fase 1
- **DADO** un fallo posterior a la fase 1
- **CUANDO** la persona pulsa «Reintentar»
- **ENTONCES** se reanuda con el nombre derivado sin volver a ejecutar la fase 1

#### Scenario: Advertencia de la fase 2
- **DADO** una fase 2 que termina con advertencia y éxito
- **CUANDO** se muestra el resultado
- **ENTONCES** el cambio se da por creado, se muestra la advertencia con su reintento y el flujo no se detiene

#### Scenario: Petición con `Origin` ajeno no crea nada
- **DADO** un alta con `Origin` ajeno
- **CUANDO** el servidor la procesa
- **ENTONCES** responde `403` sin ejecutar ninguna fase

### Requirement: Enlace «Abrir en Azure DevOps» con URL Validada (prov:WEB:10)

Cuando un work item enlazado tiene `Link.URL`, el dashboard DEBE mostrar un enlace «Abrir en Azure DevOps» a esa URL. El enlace DEBE mostrarse solo si la URL es absoluta con esquema `http` o `https`, DEBE llevar `target="_blank"` y `rel="noopener noreferrer"`, y NO DEBE mostrarse (ni como texto activo) para cualquier otro esquema (por ejemplo `javascript:` o `data:`) ni para una URL no analizable. La URL se inserta como atributo del DOM, nunca como HTML.

#### Scenario: URL https válida
- **DADO** un WI con `Link.URL` `https` válida
- **CUANDO** se muestra el WI
- **ENTONCES** aparece «Abrir en Azure DevOps» con `target="_blank"` y `rel="noopener noreferrer"`

#### Scenario: Esquema no permitido
- **DADO** un WI cuya `Link.URL` empieza por `javascript:`
- **CUANDO** se muestra el WI
- **ENTONCES** no se pinta ningún enlace activo

#### Scenario: Sin URL
- **DADO** un WI sin `Link.URL`
- **CUANDO** se muestra el WI
- **ENTONCES** no aparece el enlace

### Requirement: Rutas de Integraciones Clasificadas en el Canario de Secretos (prov:WEB:11)

Toda ruta nueva de integraciones y de arranque desde un work item DEBE clasificarse en el canario de secretos, como cubierta (`coveredRoutes`) cuando devuelve datos o como excluida con motivo (`excludedRoutes`) cuando solo muta, de modo que la prueba que exige clasificar toda ruta de `server.go` siga en verde. El canario DEBE ejercitar con un PAT centinela las rutas cubiertas (incluida `GET /api/integrations`) y comprobar que ninguna respuesta lo contiene. Las pruebas de pines de pestañas, de assets y de cadenas del JS DEBEN seguir en verde. Esta exigencia complementa a RF-WEB-26, que no cambia.

#### Scenario: Las rutas nuevas están clasificadas
- **DADO** las rutas registradas en `server.go`
- **CUANDO** se ejecuta la prueba de clasificación del canario
- **ENTONCES** cada ruta de integraciones y de arranque desde WI figura como cubierta o excluida con motivo y la prueba pasa

#### Scenario: Una ruta de integraciones sin clasificar falla
- **DADO** una ruta `/api/integrations...` añadida sin clasificar
- **CUANDO** se ejecuta la prueba de clasificación
- **ENTONCES** la prueba falla

#### Scenario: El canario no encuentra el centinela
- **DADO** un PAT centinela guardado
- **CUANDO** el canario consulta las rutas cubiertas de integraciones
- **ENTONCES** ninguna respuesta contiene el centinela

## MODIFIED Requirements

### Requirement: Estructura de Navegación y Vistas en la Web UI (RF-WEB-07)

La interfaz web en `index.html` y `app.js` DEBE estructurarse en una experiencia de usuario fluida y reactiva en español que incluya:
1. Encabezado institucional con identidad de Axiom, nombre del workspace y badge de topología.
2. Navegación por pestañas (*tabs*):
   - **Tablero de Incrementos:** vista Kanban o lista con tarjetas interactivas agrupadas por fase SDD y porcentaje de tareas.
   - **Monitor Multi-Rol:** vista de detalle del cambio seleccionado con el estado de cada rol (`blocking` vs `deferred`), tareas y estado de la barrera.
   - **Visor de Handoffs:** renderizado claro de relevos con sus metadatos y acordeón de secciones.
   - **Topología y Salud:** inspección de repositorios asociados y repositorio de especificaciones, sin mostrar la política de ramas (que se consulta y edita en «Ajustes del proyecto»); la tarjeta de Topología de Ecosistema y el selector de proyecto enlazan a esa pantalla.
   - **Ajustes del proyecto:** pantalla de consulta y edición de las claves de `axiom.yaml` (RF-WEB-18), con entrada propia en la navegación.
   - **Integraciones:** pantalla de activación y configuración de los plugins de integración, generada desde su manifiesto (prov:WEB:7), con entrada propia en la navegación.
   - **Catálogo de Skills:** visor de skills disponibles en el proyecto.
3. El detalle del cambio incluye una pestaña «Integraciones» con el work item enlazado, las Tasks por rol, la outbox y los actores (prov:WEB:8).

(Previously: la navegación no incluía «Integraciones» ni el detalle del cambio tenía pestaña de integraciones.)

#### Scenario: Navegación interactiva entre vistas del dashboard
- **DADO** la interfaz web cargada en el navegador
- **CUANDO** el usuario selecciona una pestaña en la barra de navegación
- **ENTONCES** la vista correspondiente se activa dinámicamente sin recargar la página completa
- **Y** los datos se obtienen de los endpoints REST correspondientes

#### Scenario: «Ajustes del proyecto» tiene entrada de navegación y ruta propia
- **DADO** la interfaz cargada
- **CUANDO** el usuario selecciona «Ajustes del proyecto» o abre la ruta por *hash* de esa pantalla
- **ENTONCES** se muestra la pantalla de ajustes sin recargar la página
- **Y** la entrada de navegación aparece marcada como activa

#### Scenario: «Integraciones» tiene entrada de navegación y ruta propia
- **DADO** la interfaz cargada
- **CUANDO** el usuario selecciona «Integraciones» o abre la ruta por *hash* de esa pantalla
- **ENTONCES** se muestra la pantalla de integraciones sin recargar la página
- **Y** la entrada de navegación aparece marcada como activa

#### Scenario: El detalle del cambio ofrece la pestaña «Integraciones»
- **DADO** el detalle de un cambio abierto
- **CUANDO** se recorren sus pestañas
- **ENTONCES** existe la pestaña «Integraciones»

#### Scenario: Topología no muestra la política de ramas
- **DADO** un proyecto que declara `git.integration_branch: develop`
- **CUANDO** el usuario abre Ecosistema (tarjeta de Topología) o el selector de proyecto
- **ENTONCES** ninguna de esas vistas pinta las filas de política de ramas
- **Y** cada una ofrece un enlace a «Ajustes del proyecto»

### Requirement: Las Claves de `axiom.yaml` se Editan Solo por `/api/settings`; `/api/workspace` Sigue de Solo Lectura y Toda Escritura Exige Petición Local (RF-WEB-10)

El dashboard NO DEBE editar `axiom.yaml` por ninguna ruta que no sea `PUT /api/settings` (RF-WEB-19), que aplica el catálogo y el escritor de `workspace-topology` (RF-TOP-21 a RF-TOP-23) con validación del documento completo y revisión esperada. Esto incluye el bloque `integrations.<plugin>`: se edita solo por esa ruta. El PAT se guarda y se elimina por sus endpoints propios (prov:WEB:4) en el almacén de secretos del sistema y NUNCA toca `axiom.yaml`. `GET /api/workspace` DEBE seguir siendo de solo lectura: `POST`, `PUT`, `PATCH` y `DELETE` sobre `/api/workspace` DEBEN responder `405`. Las claves de solo lectura del catálogo (incluidas `workspace.topology` y `workspace.specs_repository`) NO DEBEN poder editarse desde el dashboard.

Las preferencias personales editables desde el dashboard son las del hub: modo de trabajo y raíz de worktrees (RF-HUB-07), mediante `worktree-prefs`, y el autofetch del proyecto (RF-HUB-13), mediante `/api/settings/personal` (RF-WEB-20), siempre sobre el proyecto activo y con la validación de RF-HUB-07. Editar las preferencias personales NO DEBE modificar ningún fichero del proyecto. Toda ruta del servidor que escribe, sea `PUT /api/settings`, `PUT /api/settings/personal`, `PUT /api/projects/active/worktree-prefs`, el guardado o la eliminación del PAT, la conexión o el alta con work item, DEBE exigir una petición local (`isLocalRequest`) con `Host` y `Origin` locales (prov:WEB:2) y rechazar cualquier otra con `403` antes de leer el cuerpo o escribir.

(Previously: `integrations.*` no era editable desde el dashboard, el PAT no tenía endpoint y la exigencia de petición local no incluía `Host` ni las rutas de integraciones.)

#### Scenario: `git:` se edita desde Ajustes, no desde Topología
- **DADO** la SPA con un proyecto que declara `git.integration_branch: develop`
- **CUANDO** el usuario abre «Ajustes del proyecto»
- **ENTONCES** `develop` se muestra en un control editable de `git.integration_branch`
- **Y** la vista de topología no ofrece ningún control de edición de `git:` ni `repositories:`

#### Scenario: `/api/workspace` no escribe `axiom.yaml`
- **DADO** el servidor en ejecución
- **CUANDO** un cliente envía `POST`, `PUT`, `PATCH` o `DELETE` a `/api/workspace`
- **ENTONCES** el servidor responde `405 Method Not Allowed`
- **Y** `axiom.yaml` no se modifica

#### Scenario: Edición de las preferencias personales
- **DADO** un proyecto registrado y activo
- **CUANDO** el usuario cambia el modo a `ask` desde el dashboard
- **ENTONCES** la preferencia se persiste en `~/.axiom/workspaces.json`
- **Y** `axiom.yaml` y el resto de ficheros del proyecto no se modifican

#### Scenario: Una preferencia inválida se rechaza
- **DADO** un proyecto registrado
- **CUANDO** el cliente intenta guardar el modo `siempre`
- **ENTONCES** el servidor responde con un error `400` en español que lista los modos válidos
- **Y** el hub no se modifica

#### Scenario: El `PUT` de `worktree-prefs` exige petición local
- **DADO** un cliente cuya petición no es local
- **CUANDO** envía `PUT /api/projects/active/worktree-prefs` con un cuerpo válido
- **ENTONCES** el servidor responde `403`
- **Y** el hub no se modifica
- **Y** el `GET` del mismo endpoint sigue respondiendo con normalidad

#### Scenario: Toda escritura nueva exige petición local
- **DADO** un cliente cuya petición no es local
- **CUANDO** envía `PUT /api/settings` o `PUT /api/settings/personal`
- **ENTONCES** el servidor responde `403` sin escribir ni `axiom.yaml` ni el hub

#### Scenario: `integrations.<plugin>` se edita solo por `/api/settings`
- **DADO** un plugin de integración disponible y una petición local
- **CUANDO** el usuario guarda `integrations[<plugin>].enabled` desde «Integraciones»
- **ENTONCES** el cambio viaja por `PUT /api/settings` y se escribe en `axiom.yaml`
- **Y** ninguna otra ruta modifica `axiom.yaml`

#### Scenario: El PAT nunca toca `axiom.yaml`
- **DADO** un PAT guardado por su endpoint propio
- **CUANDO** se compara `axiom.yaml` antes y después
- **ENTONCES** el fichero no cambia

#### Scenario: Las rutas de integraciones que escriben exigen petición local con `Host` y `Origin` locales
- **DADO** un cliente con `Origin` o `Host` ajeno
- **CUANDO** envía el guardado del PAT, la conexión o el alta con work item
- **ENTONCES** el servidor responde `403` sin escribir ni llamar al servicio externo

### Requirement: Pantalla «Ajustes del Proyecto» Generada desde el Catálogo, con Reglas por Topología e ISO y sin Controles de RDD (RF-WEB-18)

El dashboard DEBE ofrecer una pantalla «Ajustes del proyecto» con ruta por *hash* y entrada en la navegación. La pantalla DEBE generarse desde el catálogo que devuelve `GET /api/settings` (RF-WEB-19) sin conocer las claves: cada entrada se pinta con su etiqueta, descripción, tipo, valores permitidos y efecto descrito, y un control acorde (selector para vocabularios cerrados, casilla para booleanos, número, texto o lista). Todo texto del catálogo y del `axiom.yaml` DEBE insertarse como texto, nunca como HTML.

La pantalla DEBE mostrar de forma visible el aviso «Editar solo afecta a cambios nuevos» y comunicar que los cambios ya sellados conservan su instantánea de `kickoff.yaml`; ninguna acción de la pantalla DEBE modificar un `kickoff.yaml` existente. Al guardar, DEBE enviar la revisión leída, mostrar el error del validador (castellano) sin perder lo tecleado cuando se rechaza, y pedir recargar cuando la revisión está obsoleta. Retirar un valor DEBE enviarse como `null`. La pantalla DEBE enviar todos los cambios pendientes en una sola petición (lote atómico, RF-WEB-19). Junto a la ruta del fichero DEBE indicar que lo guardado queda sin confirmar en git, en el árbol de trabajo del repositorio que contiene `axiom.yaml` (en `multirepo`, el de specs), y que confirmarlo es decisión de la persona. Cuando el servidor informa `writable: false` (documento con anclas, alias, `<<` o varios documentos YAML), DEBE deshabilitar los controles de edición y mostrar el motivo.

Las claves con `editable: false` (`technical_context`, `workspace.root`, `skill_roots`, `governance.*`, `workspace.topology`, `workspace.specs_repository`) DEBEN mostrarse sin control de edición, con su valor y la acción «Abrir en el editor»: un enlace `vscode://file/<ruta absoluta de axiom.yaml>` etiquetado «Abrir en el editor», más un botón «Copiar ruta» que copia esa ruta absoluta. El servidor NO DEBE lanzar ningún proceso para abrir el editor.

La pantalla DEBE incluir un bloque «Preferencias personales» con el autofetch (RF-WEB-20) y `worktree_mode` y `worktree_root` (estos dos sobre `worktree-prefs`, RF-WEB-10), identificado como personal y no versionado. En `monorepo-embedded` la pantalla NO DEBE mostrar `roles`, `gate_policy`, `repositories` ni `acceptance_tests`; en `multirepo` DEBE mostrar `repositories[]`, `acceptance_tests` y `roles[].gate_policy` con `optional` entre sus opciones. En un proyecto ISO, `workflow.lane` NO DEBE ofrecer `odd`. La pantalla NO DEBE incluir ningún control de RDD (global, por proyecto o por clon), ni `commit_review`, ni `docs_publication`. Tampoco DEBE pintar el bloque `integrations`: DEBE enlazar a «Integraciones» (prov:WEB:7), que es donde se consulta y edita.

(Previously: la pantalla excluía `integrations.*` sin ofrecer enlace a ningún sitio donde editarlo.)

#### Scenario: La pantalla se genera desde el catálogo
- **DADO** un catálogo con una entrada de vocabulario cerrado
- **CUANDO** se abre «Ajustes del proyecto»
- **ENTONCES** la entrada se pinta como selector con exactamente los valores que devolvió el servidor
- **Y** se muestran su descripción y su efecto

#### Scenario: Aviso de que solo afecta a cambios nuevos
- **DADO** un proyecto con un cambio ya sellado
- **CUANDO** se abre «Ajustes del proyecto»
- **ENTONCES** se muestra «Editar solo afecta a cambios nuevos»
- **Y** tras guardar `workflow.lane`, el `kickoff.yaml` del cambio sellado no cambia

#### Scenario: Cambiar el carril conserva comentarios y orden
- **DADO** un `axiom.yaml` comentado con `workflow.lane: ask`
- **CUANDO** el usuario elige `sdd` y guarda
- **ENTONCES** el servidor escribe el fichero solo con esa línea cambiada
- **Y** la pantalla muestra la revisión nueva y el valor guardado

#### Scenario: Valor inválido o revisión obsoleta no pierde lo tecleado
- **DADO** un valor rechazado por el validador o una revisión obsoleta
- **CUANDO** el usuario guarda
- **ENTONCES** la pantalla muestra el error en castellano y conserva el valor tecleado
- **Y** en el caso de la revisión obsoleta ofrece recargar

#### Scenario: Claves de solo lectura con «Abrir en el editor» y «Copiar ruta»
- **DADO** la pantalla abierta
- **CUANDO** se inspecciona `workspace.topology`, `technical_context` o `governance.*`
- **ENTONCES** se muestran sin control de edición
- **Y** aparece un enlace «Abrir en el editor» con destino `vscode://file/<ruta absoluta de axiom.yaml>` y un botón «Copiar ruta»
- **Y** el servidor no lanza ningún proceso

#### Scenario: Bloque «Preferencias personales»
- **DADO** un proyecto registrado y activo
- **CUANDO** se abre la pantalla
- **ENTONCES** el bloque «Preferencias personales» muestra el autofetch, `worktree_mode` y `worktree_root`
- **Y** indica que no se guardan en `axiom.yaml`

#### Scenario: Monorepo sin claves de multirepo
- **DADO** un proyecto `monorepo-embedded`
- **CUANDO** se abre la pantalla
- **ENTONCES** no aparecen `roles`, `gate_policy`, `repositories` ni `acceptance_tests`

#### Scenario: Multirepo con `gate_policy` incluido `optional`
- **DADO** un proyecto `multirepo`
- **CUANDO** se abre la pantalla
- **ENTONCES** aparecen `repositories[]`, `acceptance_tests` y `roles[].gate_policy`
- **Y** `gate_policy` ofrece `blocking`, `deferred` y `optional`

#### Scenario: ISO no ofrece `odd`
- **DADO** un proyecto con `sdd.iso: true`
- **CUANDO** se abre la pantalla
- **ENTONCES** `workflow.lane` no ofrece `odd`
- **Y** si un cliente envía `odd`, el servidor lo rechaza (RF-WEB-19)

#### Scenario: Ningún control de RDD ni de integraciones, con enlace a «Integraciones»
- **DADO** la pantalla abierta en cualquier topología
- **CUANDO** se recorren todos sus controles
- **ENTONCES** no existe ningún control de RDD, de `commit_review` ni de `docs_publication`
- **Y** no se pinta ningún control de `integrations.*`, sino un enlace a «Integraciones»

#### Scenario: Lo guardado queda sin confirmar en git
- **DADO** la pantalla abierta con la ruta de `axiom.yaml`
- **CUANDO** se inspecciona junto a esa ruta
- **ENTONCES** se indica que los cambios guardados quedan sin confirmar en git y que confirmarlos es decisión de la persona
- **Y** guardar no ejecuta ninguna operación de git

#### Scenario: Documento no editable deshabilita los controles
- **DADO** un `axiom.yaml` con anclas, alias, `<<` o varios documentos YAML
- **CUANDO** se abre la pantalla
- **ENTONCES** los controles de edición están deshabilitados y se muestra el motivo informado por el servidor

#### Scenario: Texto hostil se muestra como texto
- **DADO** un `workspace.name` con `<script>` en su valor
- **CUANDO** se abre la pantalla
- **ENTONCES** el valor aparece literal y no se interpreta como HTML

### Requirement: Endpoint `GET|PUT /api/settings` con Catálogo, Revisión, Escritura Validada y Errores Tipados en Castellano (RF-WEB-19)

`GET /api/settings` DEBE devolver el catálogo (RF-TOP-21) ya filtrado por la topología y el carácter ISO del proyecto activo, con el valor actual de cada clave, la revisión de `axiom.yaml` (RF-TOP-23) y la ruta absoluta del fichero (la de `Locate().ConfigPath`). La respuesta incluye `integrations`: sus entradas (`integrations[<plugin>].<clave>` y `integrations[<plugin>].enabled`) con su valor leído de `axiom.yaml`, derivadas del manifiesto del plugin (RF-TOP-21). La respuesta NO DEBE contener valor alguno de credenciales o secretos: el PAT no es clave del catálogo y los secretos solo se informan por su procedencia en `GET /api/integrations` (prov:WEB:3).

(Previously: la respuesta no contenía `integrations` y `integrations.*` era `unknown_key`.)

`GET /api/settings` DEBE informar además si el documento es escribible (`writable`) y, si no lo es, el motivo (`writeBlockedReason`), y los errores que el documento ya tiene en disco (`documentErrors`); los valores se leen del documento tal como está escrito, de modo que un `axiom.yaml` inválido escrito a mano se muestra con su valor real.

`PUT /api/settings` DEBE recibir un cuerpo `{"revision": "<sha256>", "changes": {"<clave>": <valor>|null, ...}}` con la revisión esperada y un parche de claves del catálogo (con `null` para retirar) y aplicarlo con el escritor (RF-TOP-22). Las claves por elemento se direccionan en el parche como `repositories[<path>].<campo>` y `roles[<id>].<campo>`. Todos los cambios del parche se aplican y validan juntos, como un lote atómico de todo o nada: si cualquiera es rechazado, no se escribe ninguno. DEBE exigir petición local y comprobarla antes de leer el cuerpo (RF-WEB-10). Un fichero con varios documentos YAML, anclas, alias o `<<` DEBE rechazarse, porque reescribirlo perdería o alteraría contenido.

Todo error DEBE llevar un código tipado y mensaje en castellano, con el cuerpo `{"code", "error", "violations"?}`, según esta correspondencia:

| Código | HTTP | Cuándo |
|---|---|---|
| `invalid_request` | `400` | cuerpo mal formado, revisión esperada ausente o `changes` vacío |
| `config_missing` | `404` | no hay proyecto activo o `axiom.yaml` no existe |
| `revision_mismatch` | `409` | revisión esperada obsoleta o cambio externo antes del reemplazo |
| `unsupported_document` | `422` | anclas, alias, `<<`, varios documentos YAML o clave repetida |
| `unknown_key` | `422` | clave que no está en el catálogo (incluida una clave de `integrations` que el manifiesto del plugin no declara, un plugin desconocido o una clave de PAT o de secreto) |
| `key_not_editable` | `422` | clave de solo lectura |
| `key_not_applicable` | `422` | clave de multirepo en monorepo, o rol o repositorio inexistente |
| `invalid_value` | `422` | tipo incorrecto, valor fuera del vocabulario o `null` en una clave obligatoria |
| `validation_failed` | `422` | el documento resultante no supera la validación del motor, incluida la contradicción ISO; incluye `violations` con el error del validador |
| `write_failed` | `500` | fallo de E/S al escribir el fichero |

Además: `200` con el catálogo y la revisión nuevos, `changed` y `warnings` si se escribe (o si el parche no cambia nada); `403` para petición no local; `405` para métodos distintos de `GET` y `PUT`. Los errores que `axiom.yaml` ya tenía en disco antes del parche NO DEBEN impedir guardar: se devuelven como `warnings` (RF-TOP-22). Un rechazo NO DEBE modificar `axiom.yaml`; el mensaje de `write_failed` DEBE indicar si el reemplazo llegó a producirse. Las rutas nuevas DEBEN registrarse en el canario de secretos (`coveredRoutes` o `excludedRoutes`) y este DEBE seguir en verde.

#### Scenario: GET devuelve catálogo, valores, revisión y ruta
- **DADO** un proyecto activo con `axiom.yaml`
- **CUANDO** el cliente consulta `GET /api/settings`
- **ENTONCES** la respuesta incluye las entradas del catálogo con su valor actual, la revisión y la ruta absoluta del fichero
- **Y** incluye las entradas de `integrations[<plugin>].<clave>` con su valor actual

#### Scenario: GET no filtra secretos
- **DADO** un `axiom.yaml` con `integrations` y una credencial centinela en el entorno y en el almacén
- **CUANDO** el cliente consulta `GET /api/settings`
- **ENTONCES** la respuesta no contiene la credencial
- **Y** el canario de secretos pasa con la ruta registrada

#### Scenario: PUT válido escribe y devuelve la revisión nueva
- **DADO** la revisión R leída y una petición local
- **CUANDO** el cliente envía `PUT /api/settings` con `workflow.lane: sdd` y la revisión R
- **ENTONCES** responde `200` con la revisión nueva
- **Y** `axiom.yaml` conserva sus comentarios y orden

#### Scenario: Revisión obsoleta devuelve 409
- **DADO** una edición externa de `axiom.yaml` tras la lectura
- **CUANDO** el cliente envía el `PUT` con la revisión anterior
- **ENTONCES** responde `409` con código tipado y mensaje en castellano
- **Y** el fichero conserva la edición externa

#### Scenario: Valor inválido devuelve 422 con el error del validador
- **DADO** una petición local con `git.promotion: siempre`
- **CUANDO** el cliente envía el `PUT`
- **ENTONCES** responde `422` con el error del validador en castellano
- **Y** la revisión del fichero no cambia

#### Scenario: `odd` en ISO se rechaza
- **DADO** un proyecto con `sdd.iso: true`
- **CUANDO** el cliente envía `workflow.lane: odd`
- **ENTONCES** responde `422` con la contradicción ISO
- **Y** el fichero no cambia

#### Scenario: Clave no editable devuelve 422
- **DADO** una petición con `workspace.topology`
- **CUANDO** el cliente envía el `PUT`
- **ENTONCES** responde `422` con el código de clave no editable
- **Y** el fichero no cambia

#### Scenario: Una clave de integración del manifiesto se escribe
- **DADO** el plugin `azuredevops` con un campo `organization` declarado en su manifiesto y una petición local
- **CUANDO** el cliente envía `PUT /api/settings` con `integrations[azuredevops].organization`
- **ENTONCES** responde `200` y `axiom.yaml` contiene el valor en `integrations.azuredevops`
- **Y** el documento vuelve a cargarse como `COMPLIANT` con el decodificador estricto

#### Scenario: Una clave de integración no declarada por el manifiesto devuelve 422
- **DADO** una petición con `integrations[azuredevops].no_existe` o con un plugin desconocido
- **CUANDO** el cliente envía el `PUT`
- **ENTONCES** responde `422` con `unknown_key` y el fichero no cambia

#### Scenario: Un valor fuera del manifiesto devuelve 422
- **DADO** un campo de enumeración, un número con mínimo y máximo o un formato del manifiesto
- **CUANDO** el cliente envía un valor que el manifiesto no admite
- **ENTONCES** responde `422` con `invalid_value` nombrando la clave y el fichero no cambia

#### Scenario: El PAT no es clave del catálogo
- **DADO** una petición con una clave de PAT o de secreto de integración
- **CUANDO** el cliente envía el `PUT`
- **ENTONCES** responde `422` con `unknown_key` y el fichero no cambia

#### Scenario: Cuerpo mal formado o sin revisión devuelve 400
- **DADO** un cuerpo que no es JSON válido o que no incluye la revisión esperada
- **CUANDO** el cliente envía el `PUT`
- **ENTONCES** responde `400` con mensaje en castellano
- **Y** el fichero no cambia

#### Scenario: Petición no local devuelve 403
- **DADO** una petición que no es local
- **CUANDO** el cliente envía `PUT /api/settings`
- **ENTONCES** responde `403`
- **Y** el fichero no cambia

#### Scenario: Sin proyecto activo o sin fichero devuelve 404
- **DADO** que no hay proyecto activo o que su `axiom.yaml` no existe
- **CUANDO** el cliente consulta `GET /api/settings` o envía `PUT /api/settings`
- **ENTONCES** responde `404` con código tipado

#### Scenario: Métodos no admitidos devuelven 405
- **DADO** el servidor en ejecución
- **CUANDO** el cliente envía `POST`, `PATCH` o `DELETE` a `/api/settings`
- **ENTONCES** responde `405`

#### Scenario: Multirepo escribe el fichero del repositorio de specs
- **DADO** un `multirepo` activo
- **CUANDO** el cliente envía un `PUT` válido
- **ENTONCES** se escribe el `axiom.yaml` del repositorio de specs y `GET` informa esa ruta

#### Scenario: El lote es atómico, todo o nada
- **DADO** un parche con un cambio válido y otro inválido
- **CUANDO** el cliente envía el `PUT`
- **ENTONCES** responde `422` y `axiom.yaml` no cambia, tampoco por el cambio válido

#### Scenario: Un lote puede pasar de `odd` a ISO de una vez
- **DADO** un proyecto con `workflow.lane: odd` y sin ISO
- **CUANDO** el cliente envía en un mismo lote `sdd.iso: true` y `workflow.lane: sdd`
- **ENTONCES** responde `200` y el documento resultante es válido

#### Scenario: Las claves por elemento se direccionan por ruta o por identificador
- **DADO** un `multirepo` con el repositorio `repo-a` y el rol `web`
- **CUANDO** el cliente envía `repositories[repo-a].promotion` y `roles[web].gate_policy`
- **ENTONCES** se escriben en la entrada de `repo-a` y en el rol `web`

#### Scenario: Varios documentos YAML se rechazan
- **DADO** un `axiom.yaml` con dos documentos YAML
- **CUANDO** el cliente envía un `PUT` válido
- **ENTONCES** responde `422` con código `unsupported_document`
- **Y** el fichero conserva ambos documentos

#### Scenario: La petición no local se rechaza antes de leer el cuerpo
- **DADO** una petición no local con un cuerpo mal formado
- **CUANDO** el cliente envía el `PUT`
- **ENTONCES** responde `403` y no `400`

#### Scenario: Un error previo del documento no impide guardar
- **DADO** un `axiom.yaml` que ya tiene un error del motor que el parche no toca
- **CUANDO** el cliente envía un `PUT` válido
- **ENTONCES** responde `200` con ese error en `warnings`
- **Y** el parche se escribe

#### Scenario: Fallo de escritura devuelve 500
- **DADO** una petición válida cuyo reemplazo del fichero falla
- **CUANDO** el cliente envía el `PUT`
- **ENTONCES** responde `500` con código `write_failed` y mensaje en castellano que dice si el reemplazo llegó a producirse
- **Y** `axiom.yaml` conserva su contenido anterior y no queda ningún temporal
