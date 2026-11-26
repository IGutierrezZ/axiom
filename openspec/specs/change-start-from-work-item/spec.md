# Delta para Arranque de un Cambio desde un Work Item (change-start-from-work-item)

> **Cambio:** `arranque-desde-wi` (C-07-d). **Idioma:** castellano de España; identificadores, comandos y claves en inglés.
> Capacidad nueva, dominio `RF-ARR`. Identificadores asignados al archivar `arranque-desde-wi` (2026-11-26).
> «Plugin activo» significa que `axiom.yaml` declara `integrations.azure-devops` con `enabled` distinto de `false` y que el plugin se resuelve en el registro (patrón `resolvePortTarget`). «WI» es el work item del tracker. Ninguna prueba ni escenario de esta capacidad depende de Azure DevOps real: todos se ejecutan contra el servidor local `adotest`.
> Códigos de salida del verbo: 0 (éxito, «no aplica» y éxito con avisos), 1 (rechazo tipado), 2 (uso incorrecto).
> Catálogo cerrado de códigos de rechazo: `usage_invalid` (salida 2), `work_item_required`, `work_item_reference_invalid`, `work_item_not_found`, `work_item_type_unmapped`, `work_item_closed`, `work_item_linked`, `tracker_unavailable`, `staging_conflict`, `staging_not_found`, `change_not_prepared`, `change_start_busy`, `git_repository_required`, `write_failed` y `port_unsupported` (salida 1 salvo `usage_invalid`).
> Fuera de alcance, sin delta: `odd-living-document` RF-ODO-01 está retirado (su estructura pasó a convención de agente, y `external_ref` en la cabecera ODD lo cubre RF-RUT-22) y `dashboard-sdd-orchestration` RF-DSO-04 no fija una lista cerrada de subcomandos en su escenario de ayuda; ninguno de los dos requiere delta.

## ADDED Requirements

### Requirement: Verbo `axiom change start` en Dos Fases con Salida Versionada (RF-ARR-01)

La CLI MUST ofrecer el subcomando `axiom change start`, delegado desde `axiom change`, con las banderas `--from <id|url>`, `--adopt`, `--discard`, `--change <nombre>`, `--lane sdd|odd`, `--cwd` y `--json`. Los modos `--from`, `--adopt` y `--discard` MUST ser mutuamente excluyentes; combinarlos MUST rechazarse como uso incorrecto con el código tipado `usage_invalid`, salida 2 y sin ninguna operación de red ni escritura. `--adopt` y `--discard` MUST exigir `--change`, y `--change` MUST NOT aceptarse con `--from`. `--lane` MUST valer `sdd` por defecto y solo MUST aceptarse con `--adopt`: `--lane` junto a `--from` o `--discard`, o con un valor distinto de `sdd` y `odd`, MUST rechazarse con `usage_invalid` y salida 2. Las banderas desconocidas, los argumentos posicionales y un `--change` que no sea un componente de nombre de cambio válido MUST rechazarse también con `usage_invalid`. Estas comprobaciones de uso MUST preceder a la de aplicabilidad. Si, con plugin activo, `--from` no vacío, `--adopt` o `--discard` se ejecutan con un `--cwd` que no pertenece a un repositorio git (no se resuelve un common dir), el verbo MUST rechazar con `git_repository_required` (salida 1), sin red y sin escrituras, y la continuación MUST nombrar `axiom change start … --cwd <repositorio>`; la consulta sin argumentos y `--from` vacío MUST NOT exigir git. Con `--json` la salida MUST ser un único documento con el campo `schema: axiom.change-start/v1`, la fase ejecutada, el resultado (o el rechazo tipado con su código, mensaje y continuación) y la lista de avisos. El verbo MUST NOT ejecutar `axiom branch ensure` ni `axiom sdd kickoff seal`, MUST NOT crear `proposal.md` ni el documento ODD y MUST NOT ejecutar `git commit`, `git push` ni ninguna operación de git remota.

#### Scenario: La ayuda de `axiom change` lista `start`
- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom change --help`
- **ENTONCES** la ayuda lista `start` con sus modos `--from`, `--adopt` y `--discard`
- **Y** `axiom --help` menciona el verbo

#### Scenario: Modos incompatibles se rechazan como uso incorrecto
- **DADO** un proyecto con el plugin activo y un observador de red
- **CUANDO** se ejecuta `axiom change start --from 2525 --adopt --change x`
- **ENTONCES** el verbo termina con código 2
- **Y** el observador no registra ninguna petición y no se escribe ningún fichero

#### Scenario: `--adopt` y `--discard` exigen `--change`
- **DADO** un proyecto con el plugin activo
- **CUANDO** se ejecuta `axiom change start --adopt` o `axiom change start --discard` sin `--change`
- **ENTONCES** el verbo termina con código 2 y el mensaje nombra `--change`

#### Scenario: `--lane` con `--from` o `--discard` es uso incorrecto
- **DADO** un proyecto con el plugin activo y un observador de red
- **CUANDO** se ejecuta `axiom change start --from 2525 --lane odd` o `axiom change start --discard --change x --lane sdd`
- **ENTONCES** el rechazo tiene el código `usage_invalid`, la salida termina con código 2 y la continuación nombra `axiom change start --help`
- **Y** el observador no registra ninguna petición y no se escribe ningún fichero

#### Scenario: `--cwd` fuera de un repositorio git
- **DADO** un directorio con `axiom.yaml` y el plugin activo, pero sin repositorio git
- **CUANDO** se ejecuta `axiom change start --from 2525 --json`
- **ENTONCES** el rechazo tiene el código `git_repository_required` y la salida termina con código 1
- **Y** no se hace ninguna petición HTTP ni se escribe ningún fichero

#### Scenario: La salida `--json` es versionada
- **DADO** una fase 1 correcta con `--json`
- **CUANDO** se lee la salida
- **ENTONCES** contiene `schema: axiom.change-start/v1`, el nombre derivado, el perfil y la `external_ref`
- **Y** un rechazo con `--json` contiene el código tipado, el mensaje y la continuación con el mismo esquema

#### Scenario: El verbo no ejecuta el resto del arranque
- **DADO** un observador instalado en el único punto del motor que ejecuta git y un proyecto con el plugin activo
- **CUANDO** se ejecutan la fase 1 y la fase 2
- **ENTONCES** el observador no registra `git commit`, `git push`, `git fetch`, `git pull` ni `git ls-remote`
- **Y** no se crea ninguna rama, worktree, `kickoff.yaml`, `proposal.md` ni documento ODD

### Requirement: Sin Plugin Activo el Verbo Responde «No Aplica» sin Red (RF-ARR-02)

Si no hay plugin activo (sin bloque `integrations.azure-devops`, `enabled: false`, `axiom.yaml` ausente o inválido, o plugin no resoluble), `axiom change start` MUST informar de que no aplica, MUST terminar con código 0, MUST NOT realizar ninguna operación de red y MUST NOT escribir ningún fichero ni crear staging, en cualquiera de sus modos. Con `--json` la respuesta MUST llevar `applicable: false`. Esta respuesta es la que el orquestador usa para saber si el WI es obligatorio. Cuando la causa sea un `axiom.yaml` inválido (incluido un bloque `integrations.azure-devops` con configuración incompleta o malformada, como `organization` o `project` ausentes) o un plugin activo que no se resuelve en el registro, la respuesta MUST incluir un aviso, porque la exigencia del WI queda desactivada; en el caso del `axiom.yaml` inválido el aviso MUST nombrar `axiom workspace validate`. Un `axiom.yaml` ausente o un plugin desactivado explícitamente no llevan aviso. Un plugin activo y resoluble que no ofrece el puerto de work items MUST rechazar con `port_unsupported` (salida 1) y no responder «no aplica».

#### Scenario: Proyecto sin integraciones
- **DADO** un proyecto sin bloque `integrations` en `axiom.yaml`
- **CUANDO** se ejecuta `axiom change start --from 2525 --json`
- **ENTONCES** la respuesta indica `applicable: false` y el verbo termina con código 0
- **Y** no se hace ninguna petición HTTP ni se escribe ningún fichero

#### Scenario: Plugin desactivado explícitamente
- **DADO** un `axiom.yaml` con `integrations.azure-devops.enabled: false`
- **CUANDO** se ejecuta `axiom change start` en cualquier modo
- **ENTONCES** la respuesta es «no aplica» con código 0 y sin red

#### Scenario: `axiom.yaml` ausente
- **DADO** un directorio sin `axiom.yaml`
- **CUANDO** se ejecuta `axiom change start --adopt --change x`
- **ENTONCES** la respuesta es «no aplica» con código 0 y no se escribe nada

#### Scenario: `axiom.yaml` inválido
- **DADO** un `axiom.yaml` que no supera la validación del espacio de trabajo
- **CUANDO** se ejecuta `axiom change start --json`
- **ENTONCES** la respuesta es «no aplica» con código 0, sin red ni escrituras
- **Y** `warnings` contiene un aviso que nombra `axiom workspace validate`

#### Scenario: Plugin activo con configuración incompleta
- **DADO** un bloque `integrations.azure-devops` activo al que le falta `organization`
- **CUANDO** se ejecuta `axiom change start --from 2525`
- **ENTONCES** la respuesta es «no aplica» con código 0, sin red ni escrituras y con un aviso que nombra `axiom workspace validate`

### Requirement: Con Plugin Activo y sin WI el Verbo se Niega y Pide el WI (RF-ARR-03)

Con el plugin activo, `axiom change start` sin `--from`, `--adopt` ni `--discard` MUST rechazarse con el código tipado `work_item_required` y código de salida 1, sin red ni escrituras. El mensaje MUST pedir el identificador numérico o la URL del work item y la continuación MUST nombrar `axiom change start --from <id|url>`. Crear el WI MUST NOT ser una opción del verbo.

#### Scenario: Falta el WI
- **DADO** un proyecto con el plugin activo
- **CUANDO** se ejecuta `axiom change start --json`
- **ENTONCES** el rechazo tiene el código `work_item_required` y la salida termina con código 1
- **Y** la continuación nombra `axiom change start --from <id|url>`
- **Y** no se hace ninguna petición HTTP ni se escribe ningún fichero

#### Scenario: `--from` vacío
- **DADO** un proyecto con el plugin activo
- **CUANDO** se ejecuta `axiom change start --from ""`
- **ENTONCES** el resultado es el mismo rechazo `work_item_required`

### Requirement: Resolución del WI por ID o por URL del Host y la Organización Configurados (RF-ARR-04)

`--from` MUST aceptar un identificador numérico positivo o una URL `https` cuyo host y organización coincidan con los configurados en `integrations.azure-devops`, y cuyo último segmento de ruta de work item sea un identificador numérico. El identificador MUST cumplir `^[1-9][0-9]{0,9}$` y ser como máximo 2147483647; `0`, los ceros a la izquierda, los signos y los decimales MUST rechazarse. Las formas de URL aceptadas MUST ser `/{org}[/{project}]/_workitems/edit/{id}`, `/{org}[/{project}]/_apis/wit/workitems/{id}` y cualquier ruta bajo `/{org}/` cuya consulta lleve exactamente un parámetro `workitem={id}` (tableros, backlogs y sprints); el fragmento y la barra final MUST ignorarse. Si la ruta nombra un proyecto, MUST coincidir con el `project` configurado (sin distinguir mayúsculas); `{org}.visualstudio.com` MUST NOT aceptarse. Una URL de otro host u otra organización, una URL con `userinfo`, un esquema distinto de `https` o un valor que no sea ni número ni URL MUST rechazarse con el código `work_item_reference_invalid`, sin ninguna petición de red, y su continuación MUST nombrar `axiom change start --from <id|url>`. La URL rechazada MUST NOT enviarse al tracker ni escribirse.

#### Scenario: ID numérico
- **DADO** un plugin activo y un WI 2525 simulado
- **CUANDO** se ejecuta `axiom change start --from 2525`
- **ENTONCES** se solicita el WI 2525 al tracker simulado

#### Scenario: URL de la organización configurada
- **DADO** la organización `acme` configurada y la URL `https://dev.azure.com/acme/Proyecto/_workitems/edit/2525`
- **CUANDO** se ejecuta `axiom change start --from <url>`
- **ENTONCES** se solicita el WI 2525

#### Scenario: URL de otra organización
- **DADO** la organización `acme` configurada
- **CUANDO** se ejecuta `axiom change start --from https://dev.azure.com/otra/P/_workitems/edit/9`
- **ENTONCES** se rechaza con `work_item_reference_invalid`, con código 1 y sin ninguna petición de red

#### Scenario: URL de tablero con `workitem=`
- **DADO** la organización `acme` y el proyecto `Proyecto` configurados
- **CUANDO** se ejecuta `axiom change start --from "https://dev.azure.com/acme/Proyecto/_boards/board/t/Equipo/Stories?workitem=2525"`
- **ENTONCES** se solicita el WI 2525

#### Scenario: URL de otro proyecto de la misma organización
- **DADO** el proyecto `Proyecto` configurado
- **CUANDO** se ejecuta `axiom change start --from https://dev.azure.com/acme/Otro/_workitems/edit/9`
- **ENTONCES** se rechaza con `work_item_reference_invalid` y sin ninguna petición de red

#### Scenario: Host `visualstudio.com` no aceptado
- **DADO** la organización `acme` configurada
- **CUANDO** se ejecuta `axiom change start --from https://acme.visualstudio.com/P/_workitems/edit/9`
- **ENTONCES** se rechaza con `work_item_reference_invalid` y sin ninguna petición de red

#### Scenario: Valor que no es un WI
- **DADO** un plugin activo
- **CUANDO** se ejecuta `axiom change start --from abc` o `--from 0` o `--from http://dev.azure.com/acme/P/_workitems/edit/9`
- **ENTONCES** cada invocación se rechaza con `work_item_reference_invalid` y sin red

### Requirement: La Fase 1 Solo Escribe en Staging bajo el Common Dir de Git (RF-ARR-05)

La fase 1 (`--from`) MUST leer el WI una sola vez, validarlo, importarlo y dejar el resultado en staging bajo el common dir de git, en `<common>/axiom/change-start/v1/<cambio>/` (el WI normalizado en JSON y la carpeta `context/`). MUST NOT escribir en el árbol de trabajo del repositorio ni en el WI del tracker, MUST NOT crear ficheros fuera del common dir y MUST devolver el nombre derivado, el perfil y la `external_ref` (`<alias>:<id>`, con el alias de referencia del manifiesto). La escritura del staging MUST ser atómica: un fallo a mitad MUST NOT dejar un staging parcial que la fase 2 pueda adoptar, y un fallo de escritura MUST rechazarse con `write_failed` (salida 1), con continuación «reintenta `axiom change start …`». La detección de un staging existente MUST hacerse por identidad del WI (plugin e id guardados en su `work-item.json`) y no solo por nombre: tras derivar el nombre, MUST buscarse un staging válido de ese nombre y, si no lo hay, de cualquier otro nombre con el mismo WI (por ejemplo, porque el título cambió entre dos fases 1). Si ya existe un staging del mismo WI, con el mismo nombre o con otro, la fase 1 MUST informar de ello (estado `already_staged`, aviso con la ruta y el nombre de ese staging) sin descargar nada, sin pisarlo, sin crear otro y con código 0; el nombre devuelto MUST ser el del staging existente. Si el staging existente de ese mismo nombre corresponde a otro WI, MUST rechazar con `staging_conflict` y la continuación MUST nombrar `axiom change start --discard --change <cambio>`. Un directorio de staging sin `work-item.json` legible MUST tratarse como inexistente y sustituirse, porque no lo dejó una fase 1 completa. Si no hay staging pero el cambio del mismo nombre ya adoptó el mismo WI (vínculo local o `context/work-item.json` del mismo plugin e id), la fase 1 MUST responder `already_adopted` con código 0, sin descargar nada ni escribir nada, y, si el `backlink` no es `done`, con un aviso que nombre `axiom change start --adopt --change <cambio>` (con `--lane odd` en ODD).

#### Scenario: Fase 1 sobre un WI ya adoptado por el mismo cambio
- **DADO** un cambio `US-2525-alta-de-usuarios` que ya adoptó el WI 2525 y sin staging
- **CUANDO** se repite `axiom change start --from 2525 --json`
- **ENTONCES** el estado es `already_adopted`, el código 0 y no se hace ninguna descarga ni escritura
- **Y** si `backlink` es `pending`, un aviso nombra `axiom change start --adopt --change US-2525-alta-de-usuarios`

#### Scenario: Fase 1 correcta no toca el repositorio ni el WI
- **DADO** un WI simulado válido y un repositorio con el árbol limpio
- **CUANDO** se ejecuta `axiom change start --from 2525 --json`
- **ENTONCES** `git status` sigue limpio y no existe ningún fichero nuevo fuera de `<common>/axiom/change-start/v1/`
- **Y** el servidor simulado no recibe ningún `PATCH`
- **Y** la salida contiene el nombre derivado, el perfil y la `external_ref`

#### Scenario: El tracker se lee una sola vez
- **DADO** un servidor simulado que cuenta las lecturas del WI
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** el WI se lee una sola vez y el descubrimiento del proceso se consulta como mucho una vez

#### Scenario: Staging existente del mismo WI
- **DADO** un staging del cambio `US-2525-alta-de-usuarios` creado por una fase 1 anterior del mismo WI
- **CUANDO** se repite `axiom change start --from 2525`
- **ENTONCES** el verbo informa del staging existente, no lo modifica y termina con código 0

#### Scenario: Staging del mismo WI con otro nombre por un título cambiado
- **DADO** un staging `US-2525-alta-de-usuarios` del WI 2525 y un WI 2525 cuyo título pasó a «Alta de usuarios con SSO»
- **CUANDO** se repite `axiom change start --from 2525 --json`
- **ENTONCES** el estado es `already_staged`, el nombre devuelto es `US-2525-alta-de-usuarios` y el aviso nombra su ruta
- **Y** no se crea un segundo staging ni se descarga ningún adjunto

#### Scenario: Staging sin `work-item.json` legible
- **DADO** un directorio de staging del cambio sin `work-item.json` legible
- **CUANDO** se ejecuta la fase 1 del mismo cambio
- **ENTONCES** se sustituye por un staging íntegro y la fase 1 termina con código 0

#### Scenario: Staging existente de otro WI
- **DADO** un staging con el mismo nombre de cambio pero asociado a otro WI
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** se rechaza con `staging_conflict` y la continuación nombra `axiom change start --discard --change <cambio>`

#### Scenario: Fallo durante la escritura del staging
- **DADO** un fallo simulado de escritura a mitad de la importación
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** no queda ningún staging que `--adopt` pueda adoptar y el verbo se rechaza con `write_failed` (salida 1)

### Requirement: Rechazos Tipados de la Fase 1 con Continuación Nombrada (RF-ARR-06)

La fase 1 MUST validar, en este orden y con una sola lectura del tracker: (1) el WI existe; (2) su tipo está mapeado en `item_keys`; (3) no está cerrado, es decir, su estado no pertenece a la categoría `Completed` ni `Removed` obtenida del descubrimiento (la categoría, nunca el nombre del estado); (4) no está enlazado a otro cambio abierto (requisito siguiente). Cada rechazo MUST tener código tipado, terminar con código 1, no dejar staging ni escribir nada, y su continuación MUST nombrar un comando `axiom …` o llevar la marca `refusal:by-design`, de modo que el trinquete de rechazos de la CLI no registre entradas nuevas. Los códigos MUST ser `work_item_not_found`, `work_item_type_unmapped`, `work_item_closed` y `tracker_unavailable`. Un fallo de red, un 5xx, un 401/403, un 429 agotado o un agotamiento de tiempo del tracker en la fase 1 MUST rechazarse con `tracker_unavailable`, cuyo mensaje MUST ser accionable y cuya continuación MUST ser «reintenta `axiom change start --from <id|url>`»; el verbo MUST ser síncrono, sin outbox ni reintentos propios más allá de los del cliente. Lo mismo MUST aplicarse cuando falla la consulta de descubrimiento de la categoría de los estados con el WI ya leído: no se acepta el WI sin conocer su categoría por un fallo del tracker, se rechaza con `tracker_unavailable` y no queda staging. Con el plugin activo y configuración completa, pero sin PAT disponible en el almacén de secretos, el rechazo MUST ser `tracker_unavailable` sin ninguna petición de red, y la continuación MUST nombrar `axiom integrations secret set --plugin azure-devops --name pat` además del reintento. (Un estado del WI que no figure entre los estados de su tipo no es un fallo: el WI se acepta, la categoría queda vacía y el resultado incluye un aviso.) Ningún mensaje MUST contener el PAT.

#### Scenario: WI inexistente
- **DADO** un servidor simulado que responde 404 para el WI 9999
- **CUANDO** se ejecuta `axiom change start --from 9999`
- **ENTONCES** se rechaza con `work_item_not_found`, código 1, sin staging
- **Y** la continuación nombra `axiom change start --from <id|url>`

#### Scenario: Tipo sin mapear
- **DADO** un WI de tipo `Epic` y `item_keys` que solo mapea `User Story=US` y `Bug=BUG`
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** se rechaza con `work_item_type_unmapped`
- **Y** el mensaje nombra el tipo `Epic` y la clave `integrations.azure-devops.item_keys`

#### Scenario: WI cerrado por categoría Completed
- **DADO** un WI cuyo estado `Resolved` pertenece a la categoría `Completed` en el descubrimiento
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** se rechaza con `work_item_closed`

#### Scenario: WI eliminado por categoría Removed
- **DADO** un WI cuyo estado pertenece a la categoría `Removed`
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** se rechaza con `work_item_closed`

#### Scenario: El cierre se decide por categoría y no por nombre
- **DADO** un proceso donde el estado `Closed` pertenece a la categoría `InProgress` y el estado `Hecho` pertenece a `Completed`
- **CUANDO** se arrancan dos WI, uno en cada estado
- **ENTONCES** el primero se acepta y el segundo se rechaza con `work_item_closed`

#### Scenario: El tracker no responde
- **DADO** un servidor simulado que devuelve 503 de forma persistente o agota el tiempo
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** se rechaza con `tracker_unavailable` y código 1
- **Y** la continuación es «reintenta `axiom change start --from <id|url>`»
- **Y** no queda staging ni ningún fichero escrito

#### Scenario: PAT ausente con el plugin activo
- **DADO** un plugin activo con configuración completa y un almacén de secretos sin PAT
- **CUANDO** se ejecuta `axiom change start --from 2525 --json`
- **ENTONCES** se rechaza con `tracker_unavailable`, código 1, sin ninguna petición de red ni staging
- **Y** la continuación nombra `axiom integrations secret set --plugin azure-devops --name pat`

#### Scenario: Falla la consulta de categorías con el WI ya leído
- **DADO** un servidor simulado que devuelve el WI pero responde 503 a la consulta de estados de su tipo
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** se rechaza con `tracker_unavailable` y código 1, sin staging

#### Scenario: Estado ausente entre los estados de su tipo
- **DADO** un WI cuyo estado no figura en la lista de estados de su tipo
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** el WI se acepta, la categoría queda vacía y `warnings` contiene un aviso de que no se pudo determinar si está cerrado

#### Scenario: Ningún rechazo filtra el PAT
- **DADO** un PAT conocido y cada uno de los rechazos anteriores
- **CUANDO** se inspecciona la salida de texto y la de `--json`
- **ENTONCES** ninguna contiene el valor del PAT

#### Scenario: Cada rechazo cumple el trinquete de continuaciones
- **DADO** el catálogo de rechazos del verbo
- **CUANDO** se ejecuta la prueba del trinquete de rechazos de la CLI
- **ENTONCES** todos los rechazos nombran una continuación `axiom …` o llevan `refusal:by-design`, sin entradas nuevas en la lista base

### Requirement: Un WI Enlazado a Otro Cambio Abierto se Rechaza (RF-ARR-07)

La fase 1 MUST rechazar con el código `work_item_linked` un WI que ya esté enlazado a otro cambio abierto, comprobando (a) las etiquetas `axiom:<otro>` de `System.Tags` del WI (verdad entre máquinas) y (b) los vínculos locales de los cambios abiertos del registro local de ramas (`<common>/axiom/branch/v1/…`): los vínculos `kind: work_item` de su `integrations.yaml` en el carril SDD y el `context/work-item.json` adoptado de su carpeta `odd/context/<cambio>/` en el carril ODD (que no escribe `integrations.yaml`). Si ambas fuentes discrepan, MUST mandar el WI. La misma etiqueta del mismo cambio MUST NOT ser conflicto, y un cambio archivado localmente MUST contar como cerrado. Una etiqueta `axiom:<otro>` de la que no hay ningún rastro local (ni registro ni archivo ni documento ODD) MUST contar como cambio abierto, porque este equipo no puede distinguir un cambio abierto en otra máquina de uno cerrado allí y aún no traído; el mensaje MUST decir que el cambio no consta en este equipo y la continuación MUST ofrecer retirar la etiqueta en el tracker o actualizar el checkout principal y repetir `axiom change start --from <id|url>`. El mensaje MUST nombrar la etiqueta o el cambio que reclama el WI y la continuación MUST nombrar la salida (retirar la etiqueta en el tracker o cerrar el otro cambio con `axiom branch done --change <otro>`).

#### Scenario: Etiqueta de otro cambio en el WI
- **DADO** un WI con la etiqueta `axiom:otro-cambio` y `otro-cambio` abierto en el registro local
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** se rechaza con `work_item_linked`
- **Y** el mensaje nombra `axiom:otro-cambio` y la continuación nombra la salida

#### Scenario: Vínculo local a otro cambio sin etiqueta en el WI
- **DADO** un WI sin etiquetas `axiom:*` y el `integrations.yaml` de un cambio abierto del registro local con el vínculo `work_item` de ese WI
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** se rechaza con `work_item_linked` nombrando el cambio local

#### Scenario: WI ya adoptado en un cambio ODD abierto
- **DADO** un WI sin etiquetas `axiom:*` y un cambio ODD abierto del registro local cuyo `odd/context/<cambio>/work-item.json` corresponde a ese WI
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** se rechaza con `work_item_linked` nombrando el cambio local

#### Scenario: Etiqueta de un cambio sin rastro local
- **DADO** un WI con la etiqueta `axiom:remoto` y ningún rastro local de `remoto`
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** se rechaza con `work_item_linked`, el mensaje dice que el cambio no consta en este equipo y la continuación ofrece retirar la etiqueta o actualizar el checkout principal

#### Scenario: Etiqueta de un cambio ya archivado
- **DADO** un WI con la etiqueta `axiom:viejo` y el cambio `viejo` archivado localmente
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** el WI se acepta

#### Scenario: La misma etiqueta del mismo cambio no es conflicto
- **DADO** un WI con la etiqueta `axiom:US-2525-alta-de-usuarios` y el cambio derivado con ese mismo nombre
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** el WI se acepta

#### Scenario: Registro local ilegible
- **DADO** un registro local de ramas ilegible y un WI sin etiquetas `axiom:*`
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** el WI se acepta y el resultado incluye un aviso de que el escaneo local no fue posible

### Requirement: Derivación Determinista del Nombre del Cambio (RF-ARR-08)

El nombre del cambio MUST derivarse como `<prefijo>-<id>-<slug>`, con el prefijo que `item_keys` asigna al tipo del WI, el identificador numérico del WI y un slug del título en ASCII minúsculas, con las tildes plegadas (`á`→`a`, `ñ`→`n`), con cualquier carácter fuera de `[a-z0-9]` convertido en guion simple, sin guiones al inicio ni al final y truncado para que la rama y el worktree sigan siendo viables en Windows. La derivación MUST ser determinista (mismo WI y mismo título, mismo nombre), el nombre MUST ser válido como componente de rama (`branch/naming.go` y `gitref.ValidBranchComponent`) y, si el slug queda vacío, el nombre MUST ser `<prefijo>-<id>`. Dos WI distintos MUST NOT producir el mismo nombre, porque el identificador forma parte de él.

#### Scenario: Nombre canónico
- **DADO** un WI 2525 de tipo `User Story` con `User Story=US` y título «Alta de usuarios»
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** el nombre derivado es `US-2525-alta-de-usuarios`

#### Scenario: Tildes, signos y mayúsculas
- **DADO** un WI 7 de tipo `Bug` con `Bug=BUG` y título «Fallo en la  pestaña “Señalización” (¡urgente!)»
- **CUANDO** se deriva el nombre
- **ENTONCES** el slug contiene solo `[a-z0-9-]`, sin guiones dobles ni extremos con guion, empezando por `fallo-en-la-pestana-senalizacion`

#### Scenario: Título largo truncado
- **DADO** un título de 300 caracteres
- **CUANDO** se deriva el nombre
- **ENTONCES** la longitud del nombre no supera el límite fijado por el diseño para mantener viables la rama y el worktree en Windows
- **Y** el nombre sigue acabando sin guion y es válido como componente de rama

#### Scenario: Título sin caracteres aprovechables
- **DADO** un título compuesto solo por símbolos
- **CUANDO** se deriva el nombre
- **ENTONCES** el nombre es `<prefijo>-<id>`

#### Scenario: Título con secuencias peligrosas para git
- **DADO** un título con `..`, `~`, `^`, `:` o `@{`
- **CUANDO** se deriva el nombre
- **ENTONCES** el resultado supera `gitref.ValidBranchComponent`

### Requirement: Perfil del Cambio según el Tipo del WI (RF-ARR-09)

El resultado de la fase 1 MUST declarar el perfil: `bug` solo si el tipo del WI, ya mapeado en `item_keys`, es literalmente `Bug` (sin distinguir mayúsculas ni espacios extremos), e `incremento` para cualquier otro tipo mapeado (otros nombres de tipo de defecto, como `Defect`, producen `incremento`, y esto se documenta). El verbo MUST NOT inventar los campos `Observado:` ni `Esperado:` del informe de bug: los pasos de reproducción y los criterios del WI se importan como contexto y es la skill quien redacta el informe y pregunta lo que falte.

#### Scenario: User Story produce incremento
- **DADO** un WI válido de tipo `User Story` mapeado
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** el perfil es `incremento`

#### Scenario: Bug produce perfil de bug
- **DADO** un WI válido de tipo `Bug` mapeado
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** el perfil es `bug`
- **Y** ningún fichero escrito contiene líneas `Observado:` ni `Esperado:`

### Requirement: Importación del WI como Contexto en Markdown (RF-ARR-10)

La importación MUST producir `context/work-item.md` con el título, el tipo, el estado, la URL, la descripción, los pasos de reproducción y los criterios de aceptación, las relaciones padre y Related (identificador, tipo y título) y la lista de adjuntos e imágenes incrustadas con su resultado (descargado u omitido con motivo). Los campos HTML MUST convertirse a Markdown con el subconjunto de párrafos, saltos, listas ordenadas y no ordenadas, énfasis, enlaces, imágenes, tablas simples y código; MUST NOT conservar `script`, `style`, manejadores de eventos ni atributos `style`; y los enlaces con esquema distinto de `http`, `https` y `mailto` (por ejemplo `javascript:`) MUST convertirse en texto sin enlace. Un campo vacío MUST omitirse sin dejar secciones vacías.

#### Scenario: Descripción HTML convertida
- **DADO** un WI con descripción `<p>Hola <b>mundo</b></p><ul><li>uno</li><li>dos</li></ul>`
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** `context/work-item.md` contiene `Hola **mundo**` y una lista Markdown con `uno` y `dos`

#### Scenario: HTML hostil
- **DADO** una descripción con `<script>alert(1)</script>`, `<style>`, `onerror=` y un enlace `javascript:`
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** el Markdown no contiene `script`, `style`, `onerror` ni un enlace `javascript:`

#### Scenario: Relaciones padre y Related
- **DADO** un WI con un padre y una relación Related
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** `context/work-item.md` lista ambos con su identificador, tipo y título

#### Scenario: Campos vacíos
- **DADO** un WI sin pasos de reproducción ni criterios de aceptación
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** esas secciones no aparecen en `context/work-item.md`

### Requirement: El Contenido Importado es Dato no Fiable de un Tercero (RF-ARR-11)

`context/work-item.md` MUST empezar por una cabecera fija que declare su contenido como dato de un tercero que no concede autoridad ni modifica el alcance. Todo el texto importado MUST ir dentro de un bloque delimitado y los delimitadores propios MUST neutralizarse dentro del texto importado, de modo que el contenido no pueda cerrar el bloque ni imitar la cabecera. Esa regla MUST aplicarse también a los títulos, a los nombres de adjuntos y a los motivos de omisión escritos en el fichero.

#### Scenario: Cabecera fija presente
- **DADO** cualquier WI importado
- **CUANDO** se lee `context/work-item.md`
- **ENTONCES** la primera línea no vacía es la cabecera fija que lo declara dato no fiable de un tercero

#### Scenario: Delimitador inyectado en la descripción
- **DADO** una descripción que contiene el delimitador de cierre del bloque y, a continuación, texto con instrucciones
- **CUANDO** se importa
- **ENTONCES** el delimitador aparece neutralizado y el texto sigue dentro del bloque delimitado
- **Y** el fichero contiene exactamente un delimitador de apertura y uno de cierre propios

#### Scenario: Cabecera falsificada en el contenido
- **DADO** una descripción que reproduce la cabecera fija
- **CUANDO** se importa
- **ENTONCES** la copia queda neutralizada dentro del bloque y no se puede confundir con la cabecera real

### Requirement: Adjuntos con Lista Blanca y Límites (RF-ARR-12)

El verbo MUST descargar solo adjuntos cuya extensión y tipo de contenido estén en la lista blanca `png, jpg, jpeg, gif, webp, bmp, pdf, doc, docx, xls, xlsx, ppt, pptx, txt, md, csv, json, log`. SVG MUST excluirse. MUST aplicarse un límite de 10 MiB por fichero y de 50 MiB por WI, con descarga mediante un lector limitado que aborte al superar el límite sin depender de la cabecera `Content-Length`. Un adjunto fuera de la lista, con extensión y tipo de contenido que no concuerdan, que supera su límite, que supera el presupuesto total o cuya descarga falla MUST omitirse, no abortar la importación y quedar listado en `context/work-item.md` con su motivo. Cada adjunto descargado MUST escribirse bajo `context/attachments/` y MUST NOT interpretarse ni ejecutarse. Las imágenes incrustadas en los campos HTML MUST seguir las mismas reglas.

La concordancia entre extensión y tipo de contenido MUST evaluarse así: el tipo declarado (`Content-Type`) MUST ser uno de los propios de la extensión o el genérico `application/octet-stream`, y el contenido MUST concordar con la extensión por su firma (los primeros bytes: imagen, PDF, contenedor OOXML, contenedor OLE o texto plano); como Azure DevOps sirve la mayoría de los adjuntos como `application/octet-stream`, ese tipo genérico MUST aceptarse siempre que la firma concuerde con la extensión. Un adjunto servido sin cabecera `Content-Type` MUST tratarse como `application/octet-stream`. `text/html`, `application/xhtml+xml` e `image/svg+xml` MUST rechazarse con cualquier extensión, y una firma que no concuerda con la extensión MUST omitir el adjunto con el motivo de tipo de contenido no permitido.

Para las imágenes incrustadas, MUST aplicarse además: (a) las imágenes `data:` MUST omitirse sin petición; (b) las rutas relativas y las URL con esquema distinto de `https` (`file:`, `ftp:`, `http:`) MUST omitirse sin petición, porque no cumplen la validación de URL del requisito siguiente; (c) una misma URL que aparezca varias veces (en uno o varios campos, o también como adjunto) MUST descargarse una sola vez y consumir una sola vez el presupuesto, y todas sus apariciones MUST apuntar al mismo fichero local; (d) en `context/work-item.md` cada imagen incrustada MUST reescribirse a la ruta local descargada (`![texto](attachments/<fichero>)`) o, si se omite, a `[imagen omitida: <motivo>]` sin conservar la URL original. El motivo de cada omisión MUST quedar listado y saneado en `context/work-item.md`.

#### Scenario: Adjunto permitido
- **DADO** un WI con una captura `pantalla.png` de 200 KiB
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** el fichero existe bajo `context/attachments/` y figura como descargado en `context/work-item.md`

#### Scenario: Extensión fuera de la lista
- **DADO** un adjunto `herramienta.exe`
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** no se descarga y figura omitido con el motivo «tipo no permitido»

#### Scenario: SVG excluido
- **DADO** un adjunto `logo.svg` con tipo de contenido `image/svg+xml`
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** no se descarga y figura omitido con su motivo

#### Scenario: Extensión y tipo de contenido discordantes
- **DADO** un adjunto `foto.png` que el servidor sirve como `text/html`
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** se omite con el motivo de tipo de contenido no permitido

#### Scenario: Adjunto servido como `application/octet-stream`
- **DADO** un adjunto `pantalla.png` servido como `application/octet-stream` cuya firma es la de un PNG
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** se descarga y figura como descargado

#### Scenario: Adjunto genérico con firma discordante
- **DADO** un adjunto `foto.png` servido como `application/octet-stream` cuyo contenido es HTML
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** se omite con el motivo de tipo de contenido no permitido y no queda fichero

#### Scenario: Adjunto sin `Content-Type`
- **DADO** un adjunto `informe.pdf` servido sin cabecera `Content-Type` y con firma de PDF
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** se descarga como si fuera `application/octet-stream`

#### Scenario: Imágenes incrustadas que no se descargan
- **DADO** una descripción con `<img src="data:image/png;base64,AAAA">`, `<img src="img/a.png">`, `<img src="file:///c:/x.png">` y `<img src="ftp://dev.azure.com/acme/x.png">`
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** ninguna genera petición y las cuatro figuran omitidas, cada una con su motivo
- **Y** `context/work-item.md` las muestra como `[imagen omitida: <motivo>]` sin la URL original

#### Scenario: Imagen repetida y reescritura a la ruta local
- **DADO** una descripción que incrusta dos veces la misma imagen permitida de la organización configurada
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** el servidor simulado recibe una sola petición de descarga y el presupuesto se consume una sola vez
- **Y** ambas apariciones en `context/work-item.md` son `![…](attachments/<fichero>)` con el mismo fichero

#### Scenario: Fichero por encima de 10 MiB
- **DADO** un adjunto permitido que ocupa 10 MiB más un byte, servido sin `Content-Length`
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** la descarga se aborta al superar el límite, no queda fichero parcial y figura omitido con el motivo de tamaño

#### Scenario: Total por encima de 50 MiB
- **DADO** seis adjuntos permitidos de 9 MiB cada uno
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** se descargan los que caben en 50 MiB y los restantes figuran omitidos con el motivo de presupuesto total agotado

#### Scenario: Un adjunto fallido no aborta la importación
- **DADO** dos adjuntos permitidos, uno que responde 500 y otro correcto
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** el segundo se descarga, el primero figura omitido con su motivo y el verbo termina con código 0

### Requirement: El PAT Solo se Envía al Host y la Organización Configurados y los Nombres se Sanean (RF-ARR-13)

Antes de construir cualquier petición con credencial para un adjunto o una imagen, el verbo MUST validar que la URL es `https`, que no lleva `userinfo`, que su host es el configurado y que su ruta comienza por `/{organización}/`. Una URL que no cumpla MUST NOT descargarse, MUST NOT recibir la cabecera `Authorization` y MUST listarse como omitida con su motivo; esto MUST aplicarse también a las URL de `<img src>` y de los enlaces de relación. Las redirecciones MUST seguir desactivadas. Los nombres de fichero MUST sanearse: sin separadores de ruta, sin `..`, sin nombres de dispositivo de Windows (`CON`, `PRN`, `AUX`, `NUL`, `COM1`–`COM9`, `LPT1`–`LPT9`, con cualquier extensión), sin caracteres no permitidos en Windows, sin espacios ni puntos finales y con una longitud acotada; una colisión de nombres MUST resolverse de forma determinista sin sobrescribir.

#### Scenario: Imagen incrustada en un host ajeno
- **DADO** una descripción con `<img src="https://evil.example/x.png">` y un servidor que registra cada petición
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** ningún servidor distinto del simulado recibe petición alguna ni la cabecera `Authorization`
- **Y** la imagen figura omitida con el motivo de host no permitido

#### Scenario: Adjunto en la misma organización
- **DADO** una URL de adjunto bajo el host y la organización configurados
- **CUANDO** se descarga
- **ENTONCES** la petición lleva la autenticación del PAT

#### Scenario: Otra organización en el mismo host
- **DADO** una URL de adjunto en `https://dev.azure.com/otra/...`
- **CUANDO** se procesa
- **ENTONCES** se omite sin enviar el PAT

#### Scenario: Esquema `http` o `userinfo`
- **DADO** una URL `http://dev.azure.com/acme/...` y otra `https://user:pw@dev.azure.com/acme/...`
- **CUANDO** se procesan
- **ENTONCES** ambas se omiten sin petición y ninguna aparece con sus credenciales en el fichero

#### Scenario: Redirección a un host ajeno
- **DADO** un adjunto cuyo servidor responde con una redirección a otro host
- **CUANDO** se descarga
- **ENTONCES** la redirección no se sigue, el adjunto se omite con su motivo y el host de destino no recibe petición

#### Scenario: Nombres peligrosos
- **DADO** adjuntos llamados `..\..\evil.txt`, `CON.txt`, `a/b.png` y un nombre de 400 caracteres
- **CUANDO** se descargan
- **ENTONCES** todos se escriben bajo `context/` con nombres saneados que no contienen separadores, `..` ni nombres de dispositivo y que no superan la longitud acotada

#### Scenario: Colisión de nombres
- **DADO** dos adjuntos que tras el saneado tienen el mismo nombre
- **CUANDO** se descargan
- **ENTONCES** ambos se conservan con nombres distintos y deterministas

### Requirement: Fase 2 `--adopt` Mueve el Contexto, Vincula el WI y es Idempotente (RF-ARR-14)

`axiom change start --adopt --change <cambio> [--lane sdd|odd]` MUST comprobar, antes de escribir nada, que el cambio está preparado y que existe qué adoptar, y MUST ejecutar la adopción bajo el cerrojo del cambio en este orden fijo: (1) comprobar la preparación; (2) copiar y verificar `context/` desde el staging a `openspec/changes/<cambio>/context/` (carril `sdd`) o a `odd/context/<cambio>/` (carril `odd`): se copia siempre a un temporal del mismo árbol y volumen que el destino, se verifica (rutas, tamaño y SHA-256) y se publica con un único renombrado, de modo que el resultado es idéntico en uno o dos volúmenes; el staging queda intacto hasta el paso 6; (3) en el carril `sdd`, escribir el vínculo `kind: work_item` en `integrations.yaml` (RF-INT-09) con `plugin`, `id`, `url` y `source: change.start` (en el carril `odd` MUST NOT crear `integrations.yaml`); (4) asegurar las entradas de `.git/info/exclude`; (5) registrar en `context/work-item.json` del destino el carril y el enlace de vuelta como pendiente; (6) retirar el staging; (7) añadir el enlace de vuelta (requisito siguiente) y registrar su resultado (`done`, `pending` o `conflict`) en `context/work-item.json`. La implementación MAY ejecutar el paso 4 antes del paso 2 sin cambiar el orden observable del resto; si no lo hace, la única ventana en la que `context/` puede ser visible para git es la muerte del proceso entre los pasos 2 y 4, que ningún fallo controlado deja abierta y que el siguiente `--adopt` cierra. Si el retirado del staging (paso 6) falla, MUST avisar con `staging_not_retired` (nombrando `axiom change start --adopt --change <cambio>`), terminar con código 0 y continuar con el paso 7; el reintento retira el staging.

**Cambio preparado.** El cambio está preparado cuando existe su registro del carril en `<common>/axiom/branch/v1/<carril>/<cambio>.json` y, en el carril `sdd`, existe `openspec/changes/<cambio>/` bajo el planning home; además, el HEAD del checkout de `--cwd`, leído de ficheros sin ejecutar git, MUST ser la rama del cambio (o de uno de sus roles) registrada para ese repositorio, salvo cuando el repositorio es el de especificaciones en el sitio de un multirepo, que solo exige registro y carpeta. Si el HEAD no puede leerse sin git, basta con registro y carpeta y se avisa. En el carril `odd`, el cambio está preparado con el registro y el HEAD: `ensure --lane odd` no crea `odd/context/<cambio>/` ni exige ninguna carpeta previa, y la carpeta de destino la crea la propia adopción. Si algo falta, o si `--lane` no coincide con el carril del registro del cambio (por ejemplo `--lane sdd` por defecto sobre un cambio que solo tiene registro `odd`), MUST rechazar con `change_not_prepared`, sin tocar el staging, con la continuación `axiom branch ensure --change <cambio> --json` (con `--lane odd` en ODD) y, si el registro tiene worktree, el mensaje MUST nombrar `axiom change start --adopt --change <cambio> --cwd <worktree>`.

**Origen de la adopción e identidad del WI.** Con un staging válido del cambio, la identidad del WI sale de su `work-item.json`; si no hay staging, `--adopt` MUST adoptar o reintentar desde el `context/work-item.json` ya presente en el destino (en ambos carriles), y solo si tampoco existe MUST rechazar con `staging_not_found` y continuación `axiom change start --from <id|url>`. El staging MUST conservarse hasta que el contexto esté movido, el vínculo escrito (SDD), las entradas de exclusión aseguradas y `work-item.json` actualizado (pasos 2 a 5); solo entonces se retira. La identidad del WI para completar el enlace de pendiente MUST salir del `work-item.json` del destino y nunca de un staging que ya no existe.

**Reintento, interrupción y destino existente.** La operación MUST ser reintentable e idempotente: un `--adopt` interrumpido a mitad (con el staging todavía presente) MUST poder repetirse y completar los pasos que falten sin duplicar el vínculo, el contexto ni la etiqueta. Si el destino `context/` ya existe y su `work-item.json` corresponde al mismo plugin e id, el movimiento MUST darse por hecho y el staging retirarse sin sobrescribir el destino; si corresponde a otro WI, o si `context/` existe sin un `work-item.json` legible, MUST rechazar con `staging_conflict` sin fusionar ni sobrescribir nada y con la continuación `axiom change start --discard --change <cambio>`. Sin staging y con una adopción completa (`backlink: done`), un segundo `--adopt` MUST responder éxito con estado `already_adopted`, código 0, sin ninguna petición de red ni cambios en el árbol; sin staging y con el enlace de vuelta `pending` o `conflict`, MUST repetir solo los pasos idempotentes 3 y 4 y el enlace de vuelta. Un fallo de E/S al mover `context/` o al vincular MUST rechazarse con `write_failed`, con el estado anterior intacto y la continuación «reintenta `axiom change start --adopt …`». MUST NOT escribir en una rama protegida distinta de la del cambio ni ejecutar git.

#### Scenario: Adopción en SDD
- **DADO** un staging de `US-2525-alta-de-usuarios`, la carpeta `openspec/changes/US-2525-alta-de-usuarios/` creada por `ensure` y `seal`
- **CUANDO** se ejecuta `axiom change start --adopt --change US-2525-alta-de-usuarios`
- **ENTONCES** `openspec/changes/US-2525-alta-de-usuarios/context/work-item.md` existe y el staging deja de existir
- **Y** `integrations.yaml` contiene `kind: work_item` con `plugin: azure-devops`, `id: "2525"`, la URL y `source: change.start`

#### Scenario: Adopción en ODD
- **DADO** un staging y la rama del carril ODD preparada
- **CUANDO** se ejecuta `axiom change start --adopt --change <feature> --lane odd`
- **ENTONCES** `odd/context/<feature>/work-item.md` existe
- **Y** no se crea `integrations.yaml` ni ningún fichero bajo `openspec/changes/`

#### Scenario: Segundo `--adopt` idempotente
- **DADO** una adopción completada
- **CUANDO** se repite `axiom change start --adopt --change <cambio>`
- **ENTONCES** el resultado es éxito con estado `already_adopted`, código 0, sin ninguna petición de red ni cambios: `integrations.yaml` mantiene un único vínculo del WI y el WI conserva una sola etiqueta `axiom:<cambio>`

#### Scenario: Segundo `--adopt` en ODD
- **DADO** una adopción ODD completada, sin `integrations.yaml` y con el staging ya retirado
- **CUANDO** se repite `axiom change start --adopt --change <feature> --lane odd`
- **ENTONCES** la identidad del WI sale de `odd/context/<feature>/work-item.json` y el resultado es `already_adopted` con código 0 y sin red

#### Scenario: Reintento tras una adopción interrumpida
- **DADO** un `--adopt` interrumpido tras mover `context/` y antes de retirar el staging
- **CUANDO** se repite `--adopt`
- **ENTONCES** el movimiento se da por hecho al coincidir plugin e id, el vínculo y la exclusión se completan sin duplicados y el staging se retira

#### Scenario: Destino `context/` de otro WI
- **DADO** un staging del WI 2525 y un `context/` ya existente en el destino cuyo `work-item.json` es del WI 3000
- **CUANDO** se ejecuta `--adopt`
- **ENTONCES** se rechaza con `staging_conflict`, el destino y el staging permanecen intactos y la continuación nombra `axiom change start --discard --change <cambio>`

#### Scenario: Fallo de escritura al adoptar
- **DADO** un fallo simulado de E/S al mover `context/` o al escribir el vínculo
- **CUANDO** se ejecuta `--adopt`
- **ENTONCES** se rechaza con `write_failed`, el staging permanece intacto y un `--adopt` posterior completa la adopción

#### Scenario: Sin staging ni contexto adoptado
- **DADO** que no existe staging para el cambio ni `context/work-item.json` en su destino
- **CUANDO** se ejecuta `--adopt`
- **ENTONCES** se rechaza con `staging_not_found` y la continuación nombra `axiom change start --from <id|url>`

#### Scenario: Cambio sin preparar
- **DADO** un staging y ningún registro ni carpeta del cambio en el repositorio
- **CUANDO** se ejecuta `--adopt`
- **ENTONCES** se rechaza con `change_not_prepared` y la continuación nombra `axiom branch ensure --change <cambio> --json`
- **Y** el staging permanece intacto

#### Scenario: Cambio ODD sin preparar o `--lane` incongruente
- **DADO** un staging y un cambio que solo tiene registro del carril `odd`
- **CUANDO** se ejecuta `--adopt` sin `--lane` (carril `sdd` por defecto), o con `--lane odd` sobre un checkout cuyo HEAD no es la rama del cambio
- **ENTONCES** se rechaza con `change_not_prepared`, el staging permanece intacto y la continuación incluye `--lane odd`

#### Scenario: HEAD de otra rama
- **DADO** un staging, el registro del cambio y la carpeta del cambio, pero un `--cwd` cuyo HEAD es `main`
- **CUANDO** se ejecuta `--adopt`
- **ENTONCES** se rechaza con `change_not_prepared` y no se escribe nada

#### Scenario: Staging y destino en volúmenes distintos
- **DADO** un staging y un destino en volúmenes distintos
- **CUANDO** se ejecuta `--adopt`
- **ENTONCES** el contexto se copia y se verifica siempre, con el mismo resultado final que en un solo volumen, y el staging permanece intacto hasta el paso 6

#### Scenario: Fallo al retirar el staging
- **DADO** un fallo simulado al renombrar el staging en el paso 6
- **CUANDO** se ejecuta `--adopt`
- **ENTONCES** el verbo termina con código 0 y un aviso `staging_not_retired` que nombra `axiom change start --adopt --change <cambio>`
- **Y** un `--adopt` posterior retira el staging

#### Scenario: Vínculo concurrente con una entrega
- **DADO** una escritura de `integrations.yaml` por una entrega mientras `--adopt` vincula el WI
- **CUANDO** ambas terminan
- **ENTONCES** el fichero contiene ambos vínculos sin pérdida ni duplicado

### Requirement: Enlace de Vuelta por Etiqueta; un Fallo del Tracker Avisa sin Fallar (RF-ARR-15)

En la fase 2, tras mover el contexto y vincular, el verbo MUST releer el WI y MUST añadir a `System.Tags` la etiqueta `axiom:<cambio>` solo si no figura ya, conservando las demás etiquetas (separadas por `;`) y sin duplicar. Si el tracker no responde o rechaza la escritura, el verbo MUST NOT fallar: MUST terminar con código 0, MUST avisar (aviso accionable que nombra `axiom change start --adopt --change <cambio>` como reintento) y MUST dejar el contexto y el vínculo ya aplicados; un `--adopt` posterior MUST completar solo el enlace pendiente. Si al releer el WI consta una etiqueta `axiom:<otro>` de otro cambio distinto, el verbo MUST NOT añadir la suya y MUST avisar del conflicto, salvo que `<otro>` sea un cambio archivado localmente: igual que en la fase 1 (RF-ARR-07), esa etiqueta MUST tolerarse también en la fase 2 y la etiqueta propia MUST añadirse conservando la ajena, para que un WI aceptado en la fase 1 no quede sin enlazar. El Hyperlink de vuelta al cambio queda fuera de este requisito.

#### Scenario: Etiqueta añadida una sola vez
- **DADO** un WI con las etiquetas `a; b`
- **CUANDO** se ejecuta `--adopt` dos veces
- **ENTONCES** las etiquetas finales son `a; b; axiom:<cambio>` sin duplicados y se realiza una sola escritura efectiva

#### Scenario: El tracker no responde en la fase 2
- **DADO** un servidor simulado que falla al escribir el WI
- **CUANDO** se ejecuta `--adopt`
- **ENTONCES** el verbo termina con código 0 y el contexto y el vínculo quedan escritos
- **Y** el aviso nombra `axiom change start --adopt --change <cambio>`

#### Scenario: Reintento tras el fallo
- **DADO** una adopción con enlace pendiente y el tracker ya disponible
- **CUANDO** se repite `--adopt`
- **ENTONCES** se añade la etiqueta y no se duplica nada más

#### Scenario: Etiqueta de otro cambio aparecida entre fases
- **DADO** un WI que entre la fase 1 y la fase 2 recibió la etiqueta `axiom:otro`
- **CUANDO** se ejecuta `--adopt`
- **ENTONCES** no se escribe la etiqueta propia y el resultado incluye un aviso del conflicto

#### Scenario: Etiqueta de un cambio archivado en la fase 2
- **DADO** un WI con la etiqueta `axiom:viejo` y el cambio `viejo` archivado localmente, aceptado en la fase 1
- **CUANDO** se ejecuta `--adopt`
- **ENTONCES** se añade la etiqueta `axiom:<cambio>` conservando `axiom:viejo` y no hay aviso de conflicto

#### Scenario: Escritura protegida contra carreras
- **DADO** un WI modificado por otra persona entre la relectura y la escritura
- **CUANDO** se ejecuta `--adopt`
- **ENTONCES** la escritura protegida por revisión no sobrescribe el cambio ajeno y el verbo avisa o reintenta sin perder etiquetas

### Requirement: `--discard` Retira un Staging Abandonado (RF-ARR-16)

`axiom change start --discard --change <cambio>` MUST borrar únicamente `<common>/axiom/change-start/v1/<cambio>/` sin red y sin tocar el repositorio ni el WI, y MUST terminar con código 0. Si el staging no existe MUST ser un éxito sin efecto con aviso. `--change` MUST validarse como componente de nombre de cambio, y un valor con separadores o `..` MUST rechazarse como uso incorrecto sin borrar nada.

#### Scenario: Descarte de un staging existente
- **DADO** un staging del cambio `x`
- **CUANDO** se ejecuta `axiom change start --discard --change x`
- **ENTONCES** el staging desaparece y no se hace ninguna petición ni se modifica el repositorio

#### Scenario: Descarte sin staging
- **DADO** que no existe staging de `x`
- **CUANDO** se ejecuta `--discard --change x`
- **ENTONCES** el verbo termina con código 0 y un aviso de que no había nada que retirar

#### Scenario: Nombre con salida del directorio
- **DADO** un valor `--change ../../algo`
- **CUANDO** se ejecuta `--discard`
- **ENTONCES** se rechaza como uso incorrecto con código 2 y no se borra nada

### Requirement: Un Cerrojo Local por Cambio Evita Arranques Simultáneos (RF-ARR-17)

Las fases 1 y 2 y `--discard` MUST tomar un cerrojo local por cambio en el common dir. La fase 1 MUST tomar además, antes de leer el WI, un cerrojo por identidad del WI (`item-<plugin>-<id>`), con orden fijo WI, cambio, integraciones; la fase 2 y el descarte MUST NOT pedir el del WI. Dos invocaciones simultáneas sobre el mismo cambio MUST serializarse o rechazar la segunda con el código `change_start_busy` y continuación «reintenta `axiom change start …`»; en ningún caso MUST producir un staging corrupto ni un vínculo duplicado. Un cerrojo huérfano de un proceso que ya no existe MUST poder recuperarse sin intervención manual.

#### Scenario: Dos fases 1 simultáneas
- **DADO** dos invocaciones concurrentes de la fase 1 para el mismo WI
- **CUANDO** ambas terminan
- **ENTONCES** existe un único staging íntegro y la segunda informó del staging existente o fue rechazada con `change_start_busy`

#### Scenario: Dos fases 1 del mismo WI con títulos distintos
- **DADO** dos invocaciones concurrentes de la fase 1 para el mismo WI que derivarían nombres distintos por un título cambiado
- **CUANDO** ambas terminan
- **ENTONCES** existe un único staging y la segunda informó de él (`already_staged`) por el cerrojo del WI

#### Scenario: Cerrojo huérfano
- **DADO** un cerrojo de un proceso inexistente
- **CUANDO** se ejecuta la fase 1
- **ENTONCES** el cerrojo se recupera y la fase 1 prosigue

### Requirement: `context/` no se Versiona (RF-ARR-18)

El contenido de `context/` MUST quedar fuera del control de versiones. En este repositorio, `.gitignore` MUST contener las reglas `openspec/changes/*/context/`, `openspec/changes/archive/*/context/` y `odd/context/`. En un proyecto de usuario, `--adopt` MUST añadir, de forma idempotente y sin tocar el `.gitignore` versionado, las entradas equivalentes a `.git/info/exclude` del repositorio (resuelto con el common dir), conservando las líneas ajenas y sin duplicarlas. Tras `--adopt`, `git status` MUST NOT listar nada bajo `context/`. La presencia de `context/` MUST NOT impedir `axiom sdd status`, la publicación ni el archivado del cambio.

#### Scenario: Este repositorio ignora `context/`
- **DADO** este repositorio y un fichero bajo `openspec/changes/x/context/`
- **CUANDO** se ejecuta `git status --porcelain`
- **ENTONCES** el fichero no aparece

#### Scenario: Proyecto de usuario
- **DADO** un proyecto de usuario sin esas reglas y una adopción completada
- **CUANDO** se ejecuta `git status --porcelain`
- **ENTONCES** `context/` no aparece
- **Y** `.git/info/exclude` contiene las entradas una sola vez y `.gitignore` no ha cambiado

#### Scenario: Entradas ajenas conservadas
- **DADO** un `.git/info/exclude` con líneas del usuario
- **CUANDO** se ejecuta `--adopt` dos veces
- **ENTONCES** las líneas del usuario permanecen y las entradas de `context/` aparecen una sola vez

#### Scenario: Un cambio con `context/` se archiva y se publica
- **DADO** un cambio con `context/` presente
- **CUANDO** se ejecutan `axiom sdd status`, `axiom sdd publish` y el archivado
- **ENTONCES** ninguno falla por la existencia de `context/` y `context/` no se commitea

### Requirement: Hermeticidad de las Pruebas del Arranque (RF-ARR-19)

Ninguna prueba del verbo, del puerto de work items ni del adaptador MUST llamar a Azure DevOps real ni usar credenciales reales: todas MUST usar el servidor local `adotest`, un almacén de secretos sustituible y `go test` en proceso, sin compilar binarios sueltos. Toda petición saliente de esas pruebas MUST ir a un host local, y una petición a cualquier host no local MUST hacer fallar la prueba.

#### Scenario: Toda petición va al servidor local
- **DADO** la batería del verbo con el servidor `adotest` y un transporte que falla ante hosts no locales
- **CUANDO** se ejecutan las fases 1 y 2 y los rechazos
- **ENTONCES** ninguna petición sale hacia un host no local

#### Scenario: Sin binarios sueltos
- **DADO** las pruebas del verbo
- **CUANDO** se inspecciona su ejecución
- **ENTONCES** se ejecutan con `go test` en proceso y no invocan `go build` ni un `axiom.exe` aparte
