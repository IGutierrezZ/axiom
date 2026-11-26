# Delta para Adaptador de Azure DevOps (azure-devops-adapter)

> **Cambio:** `eventos-ado` (C-07-e). **Idioma:** castellano de España; identificadores, comandos y claves en inglés.
> El plugin `azure-devops` pasa de solo lectura a escribir en el ciclo de vida: Task hija por rol, estimación, horas, descripción, etiquetas de IDs funcionales, estado y asignación.
>
> **Supuestos de diseño no verificados contra Azure DevOps real.** Esta spec los fija como comportamiento del adaptador y se comprueban solo con el doble `adotest`; ninguno es un hecho confirmado de la API: (a) que el `POST` de creación acepte la relación padre `System.LinkTypes.Hierarchy-Reverse` en el mismo cuerpo, con la alternativa documentada de `POST` seguido de `PATCH` de relación; (b) los nombres de los campos de horas `Microsoft.VSTS.Scheduling.OriginalEstimate`, `Microsoft.VSTS.Scheduling.RemainingWork` y `Microsoft.VSTS.Scheduling.CompletedWork` y su presencia en el tipo Task del proceso; (c) qué HTML conserva `System.Description`; (d) que exista el estado `Tech Review` (no es estándar en Agile ni en Scrum); (e) el comportamiento de la asignación por `System.AssignedTo` con la identidad guardada por `connect`.
>
> **Nota sobre el propósito de la capacidad.** El párrafo `## Purpose` de la spec viva dice que no emite eventos del ciclo de vida ni crea work items. Una delta no puede modificar ese párrafo: debe reescribirse al integrar este cambio (tarea de integración de la spec viva), y el requisito RF-ADO-14 modificado aquí es la fuente normativa.

## ADDED Requirements

### Requirement: Creación de una Task Hija con Marca Determinista y sin Duplicados (RF-ADO-20)

El adaptador DEBE poder crear una Task hija del work item (WI) de origen con `POST .../_apis/wit/workitems/$Task` y `Content-Type: application/json-patch+json`. El cuerpo DEBE llevar el título, la etiqueta marca `axiom-task:<cambio>:<rol>`, la relación `System.LinkTypes.Hierarchy-Reverse` hacia el WI y, cuando hay estimación, los campos `OriginalEstimate` y `RemainingWork` con el mismo valor. La marca NO DEBE usar el prefijo `axiom:` (para no activar el rechazo de etiquetas `axiom:*` ajenas de RF-ADO-19). El `POST` NO DEBE reintentarse automáticamente (RF-ADO-04). El control de duplicados DEBE hacerse solo por la marca exacta y NUNCA por heurísticas de título. Para decidir si crear, el adaptador DEBE leer las Tasks hijas del WI y buscar la marca exacta en sus etiquetas: con una coincidencia la devuelve como existente sin crear nada; con varias coincidencias DEBE devolver un error tipado `task_ambiguous`; solo con cero coincidencias crea. Si el proceso del proyecto no tiene alguno de los campos de horas necesarios, DEBE devolver un error tipado que nombra el campo y NO DEBE escribir nada (sin Task creada a medias). Si el `POST` con relación padre es rechazado, el adaptador DEBE seguir el camino alternativo descrito en los supuestos (creación seguida de `PATCH` de relación con `test /rev`) y, si la Task queda sin padre, la siguiente búsqueda DEBE poder recuperarla por la marca.

#### Scenario: Creación con marca, relación y horas
- **DADO** un WI 100 sin Tasks hijas y el rol `fullstack` con estimación de 6 horas
- **CUANDO** el adaptador crea la Task del cambio `x` y el rol `fullstack`
- **ENTONCES** el servidor recibe un `POST` con la etiqueta `axiom-task:x:fullstack`, la relación `Hierarchy-Reverse` hacia el WI 100 y `OriginalEstimate` y `RemainingWork` iguales a 6
- **Y** la respuesta devuelve el identificador y la URL de la Task creada

#### Scenario: Una Task con la marca se reutiliza
- **DADO** un WI con una Task hija que lleva la etiqueta `axiom-task:x:fullstack`
- **CUANDO** el adaptador busca o crea la Task del rol `fullstack`
- **ENTONCES** devuelve esa Task como existente
- **Y** el servidor no recibe ningún `POST` de creación

#### Scenario: Respuesta perdida tras la creación
- **DADO** un servidor que aplica el `POST` pero corta la conexión antes de responder
- **CUANDO** el adaptador falla por el error de red y se invoca de nuevo
- **ENTONCES** el primer intento no se reintentó automáticamente
- **Y** el segundo encuentra la Task por la marca y no crea una segunda

#### Scenario: Marcas duplicadas
- **DADO** un WI con dos Tasks hijas que llevan la misma marca `axiom-task:x:fullstack`
- **CUANDO** el adaptador busca la Task del rol
- **ENTONCES** devuelve el error tipado `task_ambiguous`
- **Y** el servidor no recibe ninguna escritura

#### Scenario: El título no decide el duplicado
- **DADO** un WI con una Task hija cuyo título coincide con el que se usaría, pero sin la marca
- **CUANDO** el adaptador busca o crea la Task del rol
- **ENTONCES** crea una Task nueva con la marca

#### Scenario: Falta un campo de horas en el proceso
- **DADO** un proceso cuyo tipo Task no tiene el campo `OriginalEstimate`
- **CUANDO** el adaptador intenta crear la Task con estimación
- **ENTONCES** devuelve un error tipado que nombra el campo ausente
- **Y** el servidor no recibe ningún `POST`

### Requirement: Validación de la Task Vinculada a Mano (RF-ADO-21)

El adaptador DEBE ofrecer la lectura de una Task por identificador con su tipo, su padre, su estado, su `rev` y sus horas (`OriginalEstimate`, `RemainingWork`, `CompletedWork`). Para la validación de un vínculo manual DEBE comprobar, por este orden y con un error tipado distinto para cada fallo, que el elemento existe (`task_not_found`), que su tipo es Task (`not_a_task`) y que cuelga del WI del cambio mediante `System.LinkTypes.Hierarchy-Reverse` (`not_child_of_work_item`). La validación NO DEBE modificar nada en Azure DevOps. Ninguno de los errores DEBE incluir el PAT (RF-ADO-15).

#### Scenario: Task válida
- **DADO** una Task 205 de tipo Task cuyo padre es el WI 100
- **CUANDO** se valida contra el WI 100
- **ENTONCES** la validación termina sin error y devuelve la lectura con su `rev` y sus horas

#### Scenario: Identificador inexistente
- **DADO** un servidor que responde 404 para el elemento 999
- **CUANDO** se valida el 999
- **ENTONCES** devuelve `task_not_found`

#### Scenario: El elemento no es una Task
- **DADO** el elemento 300 de tipo `Bug` hijo del WI 100
- **CUANDO** se valida contra el WI 100
- **ENTONCES** devuelve `not_a_task`

#### Scenario: La Task cuelga de otro WI
- **DADO** una Task 205 cuyo padre es el WI 101
- **CUANDO** se valida contra el WI 100
- **ENTONCES** devuelve `not_child_of_work_item`

#### Scenario: Task sin padre
- **DADO** una Task 205 sin relación `Hierarchy-Reverse`
- **CUANDO** se valida contra el WI 100
- **ENTONCES** devuelve `not_child_of_work_item`

### Requirement: Lectura y Escritura de Horas con `test /rev` (RF-ADO-22)

El adaptador DEBE leer `OriginalEstimate`, `RemainingWork` y `CompletedWork` de una Task y escribirlos con `PATCH` bajo RF-ADO-07 (primera operación `test /rev`). Un campo ausente en la Task se lee como cero. La imputación de `h` horas DEBE hacerse en una sola modificación que fije `CompletedWork` en la suma del valor leído más `h` y `RemainingWork` en el máximo entre cero y el valor leído menos `h`. Un rechazo de `test /rev` DEBE devolverse como `ErrRevConflict` junto con los valores releídos del servidor, sin reaplicar la suma ni reintentar. Si el proceso no tiene alguno de los campos, DEBE devolver un error tipado que lo nombra, sin escritura parcial.

#### Scenario: Imputación con `rev` vigente
- **DADO** una Task en la revisión 4 con `CompletedWork` 3, `RemainingWork` 5 y `OriginalEstimate` 8
- **CUANDO** el adaptador imputa 2 horas con `rev` 4
- **ENTONCES** el `PATCH` empieza por `test /rev` con valor 4 y deja `CompletedWork` en 5 y `RemainingWork` en 3

#### Scenario: Lo que resta no baja de cero
- **DADO** una Task con `RemainingWork` 1
- **CUANDO** el adaptador imputa 4 horas
- **ENTONCES** `RemainingWork` queda en 0 y `CompletedWork` aumenta en 4

#### Scenario: Horas imputadas a mano en Azure DevOps entre la lectura y la escritura
- **DADO** una Task leída en la revisión 4 y que el servidor ya tiene en la revisión 5 por una imputación manual
- **CUANDO** el adaptador escribe con `rev` 4
- **ENTONCES** devuelve `ErrRevConflict` con los valores releídos
- **Y** el servidor no registra ninguna modificación
- **Y** el adaptador no reintenta ni reaplica la suma

#### Scenario: Campo ausente en el valor
- **DADO** una Task sin `CompletedWork` en su lectura pero con el campo en el proceso
- **CUANDO** el adaptador imputa 2 horas
- **ENTONCES** `CompletedWork` queda en 2

#### Scenario: Campo ausente en el proceso
- **DADO** un proceso sin el campo `CompletedWork`
- **CUANDO** el adaptador intenta imputar
- **ENTONCES** devuelve un error tipado que nombra el campo y el servidor no recibe ninguna escritura

### Requirement: Sustitución Idempotente de `System.Description` (RF-ADO-23)

El adaptador DEBE sustituir `System.Description` completo por el HTML recibido mediante `PATCH` bajo RF-ADO-07. Si el valor actual es idéntico byte a byte al HTML recibido, y no hay etiquetas que cambiar, NO DEBE enviar ninguna petición de escritura. El adaptador NO DEBE modificar `System.Title` ni ningún otro campo del WI de origen. El texto original queda en el historial de revisiones del WI.

#### Scenario: Sustitución de la descripción
- **DADO** un WI con una descripción distinta del resumen
- **CUANDO** el adaptador sustituye la descripción por el HTML `<p>x</p>`
- **ENTONCES** el `PATCH` empieza por `test /rev` y fija `System.Description` en ese HTML

#### Scenario: Sin cambios no hay escritura
- **DADO** un WI cuya descripción ya es exactamente el HTML recibido
- **CUANDO** el adaptador la sustituye de nuevo
- **ENTONCES** el servidor no recibe ninguna petición de escritura

#### Scenario: Solo se toca la descripción
- **DADO** un WI con título, estado y asignado
- **CUANDO** el adaptador sustituye la descripción
- **ENTONCES** el `PATCH` no contiene operaciones sobre `System.Title`, `System.State` ni `System.AssignedTo`

### Requirement: Etiquetas de IDs Funcionales como Conjunto Idempotente (RF-ADO-24)

El adaptador DEBE sincronizar los IDs funcionales de la spec en `System.Tags` con una etiqueta por ID: el prefijo configurado (`req_tag_prefix`, por defecto `req:`) seguido del ID. Sincronizar DEBE: añadir las etiquetas con ese prefijo que falten, retirar las que tengan ese prefijo y ya no correspondan, y conservar sin tocar las demás, incluidas `axiom:<cambio>` y las marcas `axiom-task:*`. Todo DEBE ir en una sola modificación con `test /rev`, y la misma modificación PUEDE llevar la descripción. Una etiqueta DEBE tener como máximo 400 caracteres y no contener `,` ni `;`. Si el número de IDs supera el tope configurado (`req_tag_max`) o si el total de etiquetas del WI resultante superara 100, el adaptador NO DEBE enviar ninguna etiqueta de ID (ni siquiera parcialmente) y DEBE devolver el aviso tipado `req_tags_over_limit`, que no impide actualizar la descripción. Si el conjunto resultante es idéntico al actual, NO DEBE enviar escritura. La comparación de etiquetas es sensible a mayúsculas.

#### Scenario: Primera sincronización
- **DADO** un WI con la etiqueta `axiom:x` y los IDs funcionales `A` y `B`
- **CUANDO** el adaptador sincroniza con el prefijo por defecto
- **ENTONCES** las etiquetas finales son `axiom:x`, `req:A` y `req:B`

#### Scenario: Se retiran las obsoletas y se conservan las ajenas
- **DADO** un WI con las etiquetas `axiom:x`, `urgente`, `req:A` y `req:OLD`
- **CUANDO** el adaptador sincroniza con los IDs `A` y `B`
- **ENTONCES** las etiquetas finales son `axiom:x`, `urgente`, `req:A` y `req:B`

#### Scenario: Repetir no escribe
- **DADO** un WI cuyas etiquetas de ID ya coinciden con el conjunto
- **CUANDO** el adaptador sincroniza de nuevo con el mismo conjunto
- **ENTONCES** el servidor no recibe ninguna petición de escritura

#### Scenario: Se supera el tope configurado
- **DADO** `req_tag_max` igual a 2 y tres IDs funcionales
- **CUANDO** el adaptador sincroniza
- **ENTONCES** no se envía ninguna etiqueta de ID y devuelve el aviso `req_tags_over_limit`
- **Y** la descripción se actualiza igualmente

#### Scenario: Se superaría el límite de 100 etiquetas del WI
- **DADO** un WI con 98 etiquetas y 5 IDs nuevos con un tope configurado mayor
- **CUANDO** el adaptador sincroniza
- **ENTONCES** no se envía ninguna etiqueta de ID y devuelve `req_tags_over_limit`

#### Scenario: Prefijo configurado
- **DADO** `req_tag_prefix` igual a `rf-`
- **CUANDO** el adaptador sincroniza el ID `A`
- **ENTONCES** la etiqueta es `rf-A` y no se retiran las etiquetas con el prefijo `req:` ajenas a la configuración

#### Scenario: Conflicto de `rev` al sincronizar
- **DADO** un servidor que rechaza el `test /rev` por una modificación concurrente
- **CUANDO** el adaptador sincroniza
- **ENTONCES** relee el WI y reintenta como máximo 3 veces, igual que el enlace de vuelta de RF-ADO-19
- **Y** si el conflicto persiste devuelve `ErrRevConflict`

### Requirement: Estado y Asignación de la Task por Evento (RF-ADO-25)

`Deliver` DEBE aplicar sobre la Task vinculada al rol del evento, de forma idempotente: en `role.start`, pasarla al estado configurado `task_state_start` (por defecto `Active`) y asignarla a la identidad de Azure DevOps de la persona guardada por `connect` (RF-ADO-10, RF-ADO-11); en `role.apply.approved`, pasarla al estado `task_state_apply_approved` (por defecto `Tech Review`). Si la Task ya está en el estado destino NO DEBE escribir. La asignación NO DEBE hacerse con una identidad distinta de la de la persona que originó el evento: si el actor del evento no coincide con la persona de la identidad guardada en la máquina que entrega, o no hay identidad guardada, DEBE omitir la asignación, aplicar igualmente el estado y devolver un aviso tipado que nombra `axiom integrations connect`. Un estado que no existe en el proceso del proyecto DEBE terminar como error permanente tipado que lista los estados válidos de `discovery` y NO DEBE modificar la Task. Si el rol del evento no tiene Task vinculada en la instantánea de vínculos, DEBE terminar como error permanente tipado `task_not_bound` sin modificar nada. Todas las escrituras DEBEN cumplir RF-ADO-07.

#### Scenario: Inicio del rol
- **DADO** la Task 205 vinculada al rol `fullstack` en estado `New` y la identidad de la persona guardada
- **CUANDO** se entrega `role.start` para `fullstack` con el actor de esa persona
- **ENTONCES** la Task queda en `Active` y asignada a la identidad guardada

#### Scenario: Entrega repetida
- **DADO** la Task ya en `Active` y asignada a esa identidad
- **CUANDO** se entrega de nuevo `role.start` con la misma clave
- **ENTONCES** el servidor no recibe ninguna escritura

#### Scenario: Apply aprobado
- **DADO** la Task en `Active` y `task_state_apply_approved` por defecto
- **CUANDO** se entrega `role.apply.approved`
- **ENTONCES** la Task queda en `Tech Review`

#### Scenario: Estado inexistente en el proceso
- **DADO** un proceso cuyos estados de Task son `New`, `Active` y `Closed`
- **CUANDO** se entrega `role.apply.approved` con el estado por defecto `Tech Review`
- **ENTONCES** la entrega termina con un error permanente tipado que lista `New`, `Active` y `Closed`
- **Y** la Task no se modifica

#### Scenario: Evento originado por otra persona
- **DADO** un `role.start` cuyo actor no coincide con la persona de la identidad guardada en la máquina que entrega
- **CUANDO** se entrega
- **ENTONCES** la Task pasa a `Active` pero no se asigna
- **Y** se emite un aviso tipado que nombra `axiom integrations connect`

#### Scenario: Sin identidad guardada
- **DADO** que no hay identidad guardada para la organización
- **CUANDO** se entrega `role.start`
- **ENTONCES** la Task pasa a `Active` sin asignar y se emite el aviso tipado

#### Scenario: Rol sin Task vinculada
- **DADO** una instantánea de vínculos sin `kind: task` para el rol
- **CUANDO** se entrega `role.start`
- **ENTONCES** la entrega termina con un error permanente `task_not_bound` y no se envía ninguna escritura

### Requirement: Transición del Work Item al Archivar (RF-ADO-26)

`Deliver` DEBE aplicar en el evento `archive` la transición del WI de origen (el vínculo `kind: work_item` de la instantánea) al estado configurado `work_item_state_archive`. Si la clave no está configurada, NO DEBE hacer ninguna transición y la entrega termina con éxito. Si el WI ya está en el estado destino NO DEBE escribir. Un estado inexistente en el proceso termina como error permanente tipado que lista los estados válidos. La entrega NO DEBE devolver vínculos ni imputaciones (RF-INT-21).

#### Scenario: Transición configurada
- **DADO** `work_item_state_archive` igual a `Closed` y el WI 100 en `Active`
- **CUANDO** se entrega el evento `archive`
- **ENTONCES** el WI queda en `Closed`

#### Scenario: Sin estado configurado
- **DADO** un bloque sin `work_item_state_archive`
- **CUANDO** se entrega el evento `archive`
- **ENTONCES** la entrega termina con éxito y el servidor no recibe ninguna escritura

#### Scenario: WI ya en el estado destino
- **DADO** el WI 100 ya en `Closed`
- **CUANDO** se entrega el evento `archive`
- **ENTONCES** el servidor no recibe ninguna escritura

#### Scenario: La entrega no devuelve vínculos
- **DADO** una entrega `archive` con éxito
- **CUANDO** termina
- **ENTONCES** no devuelve vínculos ni imputaciones

### Requirement: Las Escrituras Nuevas Son Herméticas y No Filtran el PAT (RF-ADO-27)

Toda escritura nueva del adaptador (creación, lectura de hijos y de Task, horas, descripción, etiquetas, estado, asignación) DEBE pasar por el cliente único (RF-ADO-03 a RF-ADO-05) y verificarse con el servidor `httptest` `adotest` sin tocar Azure DevOps real. El PAT NO DEBE aparecer en ninguna salida ni error de las rutas nuevas (RF-ADO-15), y las rutas `/api` nuevas DEBEN estar cubiertas por el canario de secretos del adaptador.

#### Scenario: Ninguna petición sale de los servidores locales
- **DADO** la batería de pruebas de las escrituras nuevas
- **CUANDO** se ejecuta con `adotest`
- **ENTONCES** todas las peticiones llegan a un host local de prueba y ninguna a un host real

#### Scenario: El PAT no aparece en los errores de las escrituras
- **DADO** un servidor que responde error 401 y un PAT conocido
- **CUANDO** falla cada operación de escritura nueva
- **ENTONCES** el valor del PAT no aparece en ningún error, mensaje ni salida

## MODIFIED Requirements

### Requirement: Manifiesto y Alta del Plugin `azure-devops` (RF-ADO-01)

El paquete `internal/integrations/azuredevops` DEBE ofrecer el plugin `azure-devops`. Su manifiesto DEBE declarar el identificador `azure-devops`, las capacidades `items.read`, `discovery`, `tags`, `attachments`, `items.create`, `relations`, `estimation`, `time`, `items.transition` y `assignment`, los eventos `change.start`, `role.start`, `role.apply.approved` y `archive` como únicos eventos escuchados, las claves de configuración `organization`, `project`, `team`, `item_keys` y las de RF-ADO-02, el secreto `pat`, la identidad `{source: plugin, secret: pat}` y el alias de referencia `ado`, que es el prefijo de la `external_ref` (`ado:<id>`) y que NO DEBE estar escrito como constante en el verbo de arranque. El manifiesto DEBE superar la validación de RF-INT-03 y ser idéntico a la entrada del catálogo cerrado (RF-INT-04). La variable de entorno del secreto DEBE ser `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`. El manifiesto NO DEBE declarar capacidades que este cambio no entrega, en particular `vcs_links`, ni escuchar los eventos de compuerta `gate.spec.approved` y `gate.design.approved`.

(Previously: solo las capacidades `items.read`, `discovery`, `tags` y `attachments`; `change.start` era el único evento escuchado y se declaraba que no tenía capacidades de escritura del ciclo de vida)

#### Scenario: El manifiesto es válido e idéntico al del catálogo
- **DADO** el plugin `azure-devops` y el catálogo de manifiestos por defecto
- **CUANDO** se valida el manifiesto y se construye un registro con el plugin
- **ENTONCES** la validación termina sin error y el registro se construye
- **Y** el manifiesto declara exactamente las capacidades `items.read`, `discovery`, `tags`, `attachments`, `items.create`, `relations`, `estimation`, `time`, `items.transition` y `assignment`

#### Scenario: Variable de entorno del secreto
- **DADO** el manifiesto de `azure-devops`
- **CUANDO** se consulta la variable de entorno del secreto `pat`
- **ENTONCES** es `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`

#### Scenario: Capacidades de escritura del ciclo de vida
- **DADO** el manifiesto de `azure-devops`
- **CUANDO** se enumeran sus capacidades y sus eventos
- **ENTONCES** incluye `items.create`, `relations`, `estimation`, `time`, `items.transition` y `assignment`
- **Y** no incluye `vcs_links`
- **Y** los eventos escuchados son exactamente `change.start`, `role.start`, `role.apply.approved` y `archive`

#### Scenario: Alias de referencia y clave `item_keys` en el manifiesto
- **DADO** el manifiesto de `azure-devops`
- **CUANDO** se consultan su alias de referencia y sus claves de configuración
- **ENTONCES** el alias es `ado` y las claves incluyen `item_keys`
- **Y** el verbo de arranque obtiene el alias del manifiesto y no de una constante propia

### Requirement: Configuración `integrations.azure-devops` y Secreto `pat` (RF-ADO-02)

El bloque `integrations.azure-devops` de `axiom.yaml` DEBE aceptar las claves `organization`, `project`, `team`, `item_keys`, `task_state_start`, `task_state_apply_approved`, `task_state_close`, `work_item_state_archive`, `req_tag_prefix`, `req_tag_max`, `workday_hours` y `workdays` (y la reservada `enabled`) y NO DEBE contener secretos ni datos personales. `item_keys` DEBE ser una lista de cadenas `Tipo=Prefijo`, donde `Tipo` es el nombre del tipo de work item tal y como lo devuelve el tracker y `Prefijo` es un token ASCII alfanumérico no vacío y corto, apto como inicio de un nombre de rama; un elemento malformado, un `Tipo` o un `Prefijo` duplicado o un `Prefijo` no válido DEBE ser un error de carga que nombra la clave y el elemento. Sin `item_keys`, DEBE aplicarse el valor por defecto documentado en el diseño y en `axiom.example.yaml`. Las claves nuevas DEBEN tener estos tipos y valores por defecto, todos documentados en `axiom.example.yaml`: `task_state_start` (cadena no vacía, por defecto `Active`), `task_state_apply_approved` (cadena no vacía, por defecto `Tech Review`), `task_state_close` (cadena no vacía, estado de la Task al cerrar el rol en `continuous`, por defecto el mismo valor que `task_state_apply_approved`), `work_item_state_archive` (cadena no vacía, sin valor por defecto: sin él no hay transición al archivar), `req_tag_prefix` (cadena no vacía de hasta 100 caracteres sin `,` ni `;`, por defecto `req:`, y que NO DEBE empezar por `axiom:` ni por `axiom-task:`), `req_tag_max` (entero entre 1 y 100, con el valor por defecto que fija el diseño y documenta el ejemplo), `workday_hours` (entero entre 1 y 24, por defecto 8) y `workdays` (lista de cadenas no vacía sin duplicados tomadas de `mon`, `tue`, `wed`, `thu`, `fri`, `sat` y `sun`, por defecto de `mon` a `fri`). Un valor fuera de dominio DEBE ser un error de carga que nombra la clave y el valor. La existencia de un estado en el proceso del proyecto NO se valida al cargar, sino al entregar (RF-ADO-14). Una clave desconocida DEBE ser un error de carga. El PAT DEBE obtenerse únicamente del almacén de secretos del marco de integraciones (keyring en Windows y macOS, solo entorno en Linux) en el momento de cada llamada, nunca de `axiom.yaml`. La URL base DEBE construirse como `https://dev.azure.com/{organization}` desde `organization` y NO DEBE ser configurable.

(Previously: el bloque solo aceptaba `organization`, `project`, `team`, `item_keys` y `enabled`)

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

#### Scenario: Valores por defecto de las claves nuevas
- **DADO** un bloque sin ninguna de las claves nuevas
- **CUANDO** se carga el workspace
- **ENTONCES** el estado de inicio es `Active`, el de apply aprobado y el de cierre son `Tech Review`, el prefijo de etiquetas es `req:`, la jornada es de 8 horas y los días laborables son de lunes a viernes
- **Y** no hay estado de archivado configurado

#### Scenario: Claves nuevas válidas
- **DADO** un bloque con `task_state_start: Doing`, `work_item_state_archive: Closed`, `workday_hours: 6` y `workdays: [mon, tue, wed, thu]`
- **CUANDO** se carga el workspace
- **ENTONCES** la carga termina sin error y los valores quedan disponibles para el plugin

#### Scenario: Jornada fuera de dominio
- **DADO** un bloque con `workday_hours: 0` o con `workday_hours: 25`
- **CUANDO** se carga el workspace
- **ENTONCES** la carga falla con un error que nombra la clave `workday_hours` y el valor

#### Scenario: Día laborable desconocido o repetido
- **DADO** un bloque con `workdays: [mon, funday]` o con `workdays: [mon, mon]`
- **CUANDO** se carga el workspace
- **ENTONCES** la carga falla con un error que nombra la clave `workdays` y el elemento

#### Scenario: Prefijo de etiquetas en conflicto con las marcas de Axiom
- **DADO** un bloque con `req_tag_prefix: "axiom:"`
- **CUANDO** se carga el workspace
- **ENTONCES** la carga falla con un error que nombra la clave `req_tag_prefix`

#### Scenario: Tope de etiquetas fuera de dominio
- **DADO** un bloque con `req_tag_max: 101`
- **CUANDO** se carga el workspace
- **ENTONCES** la carga falla con un error que nombra la clave `req_tag_max` y el valor

### Requirement: `Deliver` Aplica los Eventos de Rol y de Archivo (RF-ADO-14)

`Deliver` DEBE actuar sobre los eventos `role.start`, `role.apply.approved` y `archive` con las reglas de los requisitos de estado, asignación y transición del adaptador, y NO DEBE actuar sobre ningún otro. Para `change.start` DEBE devolver un error permanente tipado sin realizar ninguna petición HTTP ni escribir ficheros (lo emite `axiom change start`, no `Deliver`). Un evento fuera de esa lista DEBE devolver un error permanente tipado sin red. Con el bloque `integrations.azure-devops` configurado y activo, los verbos que despachan eventos de compuerta (`gate record`) DEBEN producir la misma salida y no crear ficheros, porque ningún plugin activo escucha `gate.spec.approved` ni `gate.design.approved` (RF-INT-05). Un fallo de `Deliver` en un evento de rol o de archivo NO DEBE hacer fallar el verbo de origen (RF-INT-10): queda como operación pendiente o abandonada tipada visible en `axiom integrations sync`. Sin el bloque configurado y activo, la salida de `branch ensure --role` y la de `gate record` NO DEBEN cambiar.

(Previously: `Deliver` devolvía siempre un error permanente sin efectos y ningún evento del ciclo de vida llegaba a Azure DevOps)

#### Scenario: `Deliver` de `change.start` devuelve error permanente
- **DADO** el plugin `azure-devops` y un servidor `httptest` que falla la prueba si recibe alguna petición
- **CUANDO** se invoca `Deliver` con un evento `change.start`
- **ENTONCES** devuelve un error permanente tipado
- **Y** el servidor no recibe ninguna petición

#### Scenario: Eventos de rol y de archivo actúan
- **DADO** el plugin con una Task vinculada al rol `fullstack` y el WI 100 vinculado
- **CUANDO** se invoca `Deliver` con `role.start` y después con `role.apply.approved`
- **ENTONCES** el servidor recibe las escrituras de estado y asignación descritas en sus requisitos y no otras

#### Scenario: Evento fuera de la lista
- **DADO** el plugin y un servidor que falla la prueba si recibe alguna petición
- **CUANDO** se invoca `Deliver` con `gate.spec.approved`
- **ENTONCES** devuelve un error permanente tipado y el servidor no recibe ninguna petición

#### Scenario: Verbos de compuerta sin cambios
- **DADO** un workspace con `integrations.azure-devops` configurado y activo
- **CUANDO** se ejecuta `axiom sdd gate record --decision approved` sobre `spec`
- **ENTONCES** stdout, stderr y código de salida coinciden con los de un workspace sin el bloque
- **Y** la carpeta del cambio no contiene `integrations.yaml` ni `.integrations-outbox.yaml`

#### Scenario: Un fallo de entrega no rompe el verbo de origen
- **DADO** un workspace con el bloque activo y un servidor que falla en el estado pedido
- **CUANDO** se ejecuta `axiom branch ensure --change x --role fullstack`
- **ENTONCES** el verbo termina con el resultado de `ensure` sin error de entrega
- **Y** `axiom integrations sync` muestra la operación pendiente o abandonada con su motivo tipado
