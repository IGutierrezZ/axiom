# Delta para Dashboard Web Local (local-web-dashboard)

> **Cambio:** `ajustes-proyecto` (C-06-e). **Idioma:** castellano de España; identificadores, claves y tokens en inglés.
> RF-WEB-02 a RF-WEB-06 y RF-WEB-08, RF-WEB-09 y RF-WEB-11 a RF-WEB-17 no cambian de texto. `multi-project-hub` no cambia de requisito: el dashboard reutiliza `LookupAutofetch` y `SetAutofetch` (RF-HUB-07, RF-HUB-13).

## MODIFIED Requirements

### Requirement: Endpoint REST de Información del Workspace (RF-WEB-01)

El servidor DEBE responder a la ruta `GET /api/workspace` retornando una estructura JSON con los metadatos del espacio de trabajo declarados en `axiom.yaml`: nombre, topología (`monorepo-embedded`, `multirepo`), repositorio canónico de especificaciones, lista de roles con sus rutas y tecnologías, la política de ramas efectiva (`integration_branch` y `branch_prefix`, con los valores por defecto aplicados cuando `axiom.yaml` no declara `git:`), las entradas de `repositories:` y el estado general de validación de la topología. `/api/workspace` es de solo lectura: ninguna ruta del servidor DEBE modificar `axiom.yaml`, salvo `PUT /api/settings` (RF-WEB-19), que es la única vía de escritura de ese fichero (RF-WEB-10).
(Previously: la respuesta era de solo lectura y ninguna ruta del servidor DEBÍA modificar `axiom.yaml`, sin la excepción de `PUT /api/settings`)

#### Scenario: Retorno exitoso de topología y roles
- **DADO** un espacio de trabajo con un archivo `axiom.yaml` válido
- **CUANDO** un cliente realiza una petición HTTP `GET /api/workspace`
- **ENTONCES** la respuesta tiene código de estado `200 OK`
- **Y** el cuerpo contiene un JSON con `name`, `topology`, `specs_repository` y la lista de roles declarados

#### Scenario: Reporte de error cuando el archivo de configuración es inválido
- **DADO** un espacio de trabajo donde `axiom.yaml` no existe o contiene sintaxis corrupta
- **CUANDO** se invoca `GET /api/workspace`
- **ENTONCES** la respuesta tiene código `500 Internal Server Error`
- **Y** el cuerpo JSON detalla el motivo del fallo de lectura o validación

#### Scenario: La respuesta incluye la política de ramas efectiva
- **DADO** un `axiom.yaml` con `git.integration_branch: develop` y sin `git.branch_prefix`
- **CUANDO** se invoca `GET /api/workspace`
- **ENTONCES** el JSON informa de `integration_branch` igual a `develop` y `branch_prefix` igual a `feat`

#### Scenario: La respuesta incluye los ajustes por repositorio
- **DADO** un `axiom.yaml` con `repositories: [{path: repo-backend, integration_branch: release}]`
- **CUANDO** se invoca `GET /api/workspace`
- **ENTONCES** el JSON incluye la entrada de `repo-backend` con su rama de integración `release`

#### Scenario: `/api/workspace` sigue siendo de solo lectura
- **DADO** el servidor en ejecución
- **CUANDO** un cliente envía `POST`, `PUT`, `PATCH` o `DELETE` a `/api/workspace`
- **ENTONCES** el servidor responde `405 Method Not Allowed`
- **Y** las escrituras de `axiom.yaml` solo existen mediante `PUT /api/settings` (RF-WEB-19)

---

### Requirement: Estructura de Navegación y Vistas en la Web UI (RF-WEB-07)

La interfaz web en `index.html` y `app.js` DEBE estructurarse en una experiencia de usuario fluida y reactiva en español que incluya:
1. Encabezado institucional con identidad de Axiom, nombre del workspace y badge de topología.
2. Navegación por pestañas (*tabs*):
   - **Tablero de Incrementos:** vista Kanban o lista con tarjetas interactivas agrupadas por fase SDD y porcentaje de tareas.
   - **Monitor Multi-Rol:** vista de detalle del cambio seleccionado con el estado de cada rol (`blocking` vs `deferred`), tareas y estado de la barrera.
   - **Visor de Handoffs:** renderizado claro de relevos con sus metadatos y acordeón de secciones.
   - **Topología y Salud:** inspección de repositorios asociados y repositorio de especificaciones, sin mostrar la política de ramas (que se consulta y edita en «Ajustes del proyecto»); la tarjeta de Topología de Ecosistema y el selector de proyecto enlazan a esa pantalla.
   - **Ajustes del proyecto:** pantalla de consulta y edición de las claves de `axiom.yaml` (RF-WEB-18), con entrada propia en la navegación.
   - **Catálogo de Skills:** visor de skills disponibles en el proyecto.

(Previously: la navegación no incluía «Ajustes del proyecto» y la vista de topología mostraba la política de ramas.)

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

#### Scenario: Topología no muestra la política de ramas
- **DADO** un proyecto que declara `git.integration_branch: develop`
- **CUANDO** el usuario abre Ecosistema (tarjeta de Topología) o el selector de proyecto
- **ENTONCES** ninguna de esas vistas pinta las filas de política de ramas
- **Y** cada una ofrece un enlace a «Ajustes del proyecto»

---

### Requirement: Las Claves de `axiom.yaml` se Editan Solo por `/api/settings`; `/api/workspace` Sigue de Solo Lectura y Toda Escritura Exige Petición Local (RF-WEB-10)

El dashboard NO DEBE editar `axiom.yaml` por ninguna ruta que no sea `PUT /api/settings` (RF-WEB-19), que aplica el catálogo y el escritor de `workspace-topology` (RF-TOP-21 a RF-TOP-23) con validación del documento completo y revisión esperada. `GET /api/workspace` DEBE seguir siendo de solo lectura: `POST`, `PUT`, `PATCH` y `DELETE` sobre `/api/workspace` DEBEN responder `405`. Las claves de solo lectura del catálogo (incluidas `workspace.topology` y `workspace.specs_repository`) NO DEBEN poder editarse desde el dashboard.

Las preferencias personales editables desde el dashboard son las del hub: modo de trabajo y raíz de worktrees (RF-HUB-07), mediante `worktree-prefs`, y el autofetch del proyecto (RF-HUB-13), mediante `/api/settings/personal` (RF-WEB-20), siempre sobre el proyecto activo y con la validación de RF-HUB-07. Editar las preferencias personales NO DEBE modificar ningún fichero del proyecto. Toda ruta del servidor que escribe, sea `PUT /api/settings`, `PUT /api/settings/personal` o `PUT /api/projects/active/worktree-prefs`, DEBE exigir una petición local (`isLocalRequest`) y rechazar cualquier otra con `403` antes de leer el cuerpo o escribir.

(Previously: el dashboard no ofrecía ningún medio de editar `axiom.yaml`, `git:` y `repositories:` eran de solo lectura, ninguna ruta escribía `axiom.yaml`, el autofetch no era editable y el `PUT` de `worktree-prefs` no exigía petición local.)

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

## ADDED Requirements

### Requirement: Pantalla «Ajustes del Proyecto» Generada desde el Catálogo, con Reglas por Topología e ISO y sin Controles de RDD (RF-WEB-18)

El dashboard DEBE ofrecer una pantalla «Ajustes del proyecto» con ruta por *hash* y entrada en la navegación. La pantalla DEBE generarse desde el catálogo que devuelve `GET /api/settings` (RF-WEB-19) sin conocer las claves: cada entrada se pinta con su etiqueta, descripción, tipo, valores permitidos y efecto descrito, y un control acorde (selector para vocabularios cerrados, casilla para booleanos, número, texto o lista). Todo texto del catálogo y del `axiom.yaml` DEBE insertarse como texto, nunca como HTML.

La pantalla DEBE mostrar de forma visible el aviso «Editar solo afecta a cambios nuevos» y comunicar que los cambios ya sellados conservan su instantánea de `kickoff.yaml`; ninguna acción de la pantalla DEBE modificar un `kickoff.yaml` existente. Al guardar, DEBE enviar la revisión leída, mostrar el error del validador (castellano) sin perder lo tecleado cuando se rechaza, y pedir recargar cuando la revisión está obsoleta. Retirar un valor DEBE enviarse como `null`. La pantalla DEBE enviar todos los cambios pendientes en una sola petición (lote atómico, RF-WEB-19). Junto a la ruta del fichero DEBE indicar que lo guardado queda sin confirmar en git, en el árbol de trabajo del repositorio que contiene `axiom.yaml` (en `multirepo`, el de specs), y que confirmarlo es decisión de la persona. Cuando el servidor informa `writable: false` (documento con anclas, alias, `<<` o varios documentos YAML), DEBE deshabilitar los controles de edición y mostrar el motivo.

Las claves con `editable: false` (`technical_context`, `workspace.root`, `skill_roots`, `governance.*`, `workspace.topology`, `workspace.specs_repository`) DEBEN mostrarse sin control de edición, con su valor y la acción «Abrir en el editor»: un enlace `vscode://file/<ruta absoluta de axiom.yaml>` etiquetado «Abrir en el editor», más un botón «Copiar ruta» que copia esa ruta absoluta. El servidor NO DEBE lanzar ningún proceso para abrir el editor.

La pantalla DEBE incluir un bloque «Preferencias personales» con el autofetch (RF-WEB-20) y `worktree_mode` y `worktree_root` (estos dos sobre `worktree-prefs`, RF-WEB-10), identificado como personal y no versionado. En `monorepo-embedded` la pantalla NO DEBE mostrar `roles`, `gate_policy`, `repositories` ni `acceptance_tests`; en `multirepo` DEBE mostrar `repositories[]`, `acceptance_tests` y `roles[].gate_policy` con `optional` entre sus opciones. En un proyecto ISO, `workflow.lane` NO DEBE ofrecer `odd`. La pantalla NO DEBE incluir ningún control de RDD (global, por proyecto o por clon), ni `integrations.*`, ni `commit_review`, ni `docs_publication`.

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

#### Scenario: Ningún control de RDD ni de integraciones
- **DADO** la pantalla abierta en cualquier topología
- **CUANDO** se recorren todos sus controles
- **ENTONCES** no existe ningún control de RDD, de `integrations.*`, de `commit_review` ni de `docs_publication`

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

---

### Requirement: Endpoint `GET|PUT /api/settings` con Catálogo, Revisión, Escritura Validada y Errores Tipados en Castellano (RF-WEB-19)

`GET /api/settings` DEBE devolver el catálogo (RF-TOP-21) ya filtrado por la topología y el carácter ISO del proyecto activo, con el valor actual de cada clave, la revisión de `axiom.yaml` (RF-TOP-23) y la ruta absoluta del fichero (la de `Locate().ConfigPath`). La respuesta NO DEBE contener `integrations` ni valor alguno de credenciales o secretos.

`GET /api/settings` DEBE informar además si el documento es escribible (`writable`) y, si no lo es, el motivo (`writeBlockedReason`), y los errores que el documento ya tiene en disco (`documentErrors`); los valores se leen del documento tal como está escrito, de modo que un `axiom.yaml` inválido escrito a mano se muestra con su valor real.

`PUT /api/settings` DEBE recibir un cuerpo `{"revision": "<sha256>", "changes": {"<clave>": <valor>|null, ...}}` con la revisión esperada y un parche de claves del catálogo (con `null` para retirar) y aplicarlo con el escritor (RF-TOP-22). Las claves por elemento se direccionan en el parche como `repositories[<path>].<campo>` y `roles[<id>].<campo>`. Todos los cambios del parche se aplican y validan juntos, como un lote atómico de todo o nada: si cualquiera es rechazado, no se escribe ninguno. DEBE exigir petición local y comprobarla antes de leer el cuerpo (RF-WEB-10). Un fichero con varios documentos YAML, anclas, alias o `<<` DEBE rechazarse, porque reescribirlo perdería o alteraría contenido.

Todo error DEBE llevar un código tipado y mensaje en castellano, con el cuerpo `{"code", "error", "violations"?}`, según esta correspondencia:

| Código | HTTP | Cuándo |
|---|---|---|
| `invalid_request` | `400` | cuerpo mal formado, revisión esperada ausente o `changes` vacío |
| `config_missing` | `404` | no hay proyecto activo o `axiom.yaml` no existe |
| `revision_mismatch` | `409` | revisión esperada obsoleta o cambio externo antes del reemplazo |
| `unsupported_document` | `422` | anclas, alias, `<<`, varios documentos YAML o clave repetida |
| `unknown_key` | `422` | clave que no está en el catálogo (incluidas `integrations.*`) |
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
- **Y** no incluye `integrations`

#### Scenario: GET no filtra secretos
- **DADO** un `axiom.yaml` con `integrations` que contiene una credencial centinela
- **CUANDO** el cliente consulta `GET /api/settings`
- **ENTONCES** la respuesta no contiene la credencial ni el bloque `integrations`
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
- **DADO** una petición con `workspace.topology` o `integrations.<clave>`
- **CUANDO** el cliente envía el `PUT`
- **ENTONCES** responde `422` con el código de clave no editable
- **Y** el fichero no cambia

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

---

### Requirement: Endpoint `GET|PUT /api/settings/personal` para el Autofetch del Proyecto Activo (RF-WEB-20)

`GET /api/settings/personal` DEBE devolver el autofetch personal del proyecto activo mediante `hub.LookupAutofetch`, y `PUT /api/settings/personal` DEBE guardarlo mediante `hub.SetAutofetch`, con un valor booleano. El PUT DEBE exigir petición local (RF-WEB-10). El valor DEBE persistirse en `~/.axiom/workspaces.json` y NO DEBE modificar ningún fichero del proyecto. Códigos: `200` en lectura y escritura correctas; `400` para cuerpo mal formado o valor no booleano; `403` para petición no local; `404` si el proyecto activo no está registrado en el hub (`hub.ErrProjectNotRegistered`) o no hay proyecto activo; `405` para otros métodos. Los errores DEBEN llevar mensaje en castellano. Las rutas DEBEN estar registradas en el canario de secretos.

Este requisito (P16) sustituye la línea del plan de la propuesta anterior según la cual «el autofetch… no entra en esta pantalla»: el autofetch del proyecto SÍ se edita desde «Ajustes del proyecto», en el bloque «Preferencias personales» (RF-WEB-18). `multi-project-hub` no cambia: `LookupAutofetch` y `SetAutofetch` (RF-HUB-13) se reutilizan tal cual.

#### Scenario: Lectura del autofetch
- **DADO** un proyecto registrado con el autofetch activado
- **CUANDO** el cliente consulta `GET /api/settings/personal`
- **ENTONCES** responde `200` con el autofetch en `true`

#### Scenario: Escritura del autofetch persiste en el hub
- **DADO** un proyecto registrado y activo y una petición local
- **CUANDO** el cliente envía `PUT /api/settings/personal` con el autofetch en `true`
- **ENTONCES** responde `200`
- **Y** `~/.axiom/workspaces.json` refleja el valor
- **Y** `axiom.yaml` y el resto de ficheros del proyecto no se modifican

#### Scenario: Proyecto no registrado devuelve 404
- **DADO** un proyecto activo que no está registrado en el hub
- **CUANDO** el cliente consulta o escribe `/api/settings/personal`
- **ENTONCES** responde `404` con mensaje en castellano
- **Y** el hub no se modifica

#### Scenario: Valor no booleano devuelve 400
- **DADO** un proyecto registrado
- **CUANDO** el cliente envía `PUT /api/settings/personal` con `"si"`
- **ENTONCES** responde `400` y el hub no se modifica

#### Scenario: Petición no local devuelve 403
- **DADO** una petición que no es local
- **CUANDO** el cliente envía `PUT /api/settings/personal`
- **ENTONCES** responde `403` y el hub no se modifica

---

### Requirement: Ecosistema Deja de Mostrar la Política de Ramas y, Junto con el Selector de Proyecto, Enlaza a «Ajustes del Proyecto» (RF-WEB-21)

La pantalla Ecosistema NO DEBE pintar la política de ramas (`git:` y `repositories[]`) de forma propia: DEBE retirar el uso de las filas de política de ramas (`branchPolicyRows`) y de `repoPolicyHtml`, y su tarjeta de Topología DEBE enlazar a «Ajustes del proyecto», que es el único lugar donde se consulta y edita. El selector de proyecto (`projects.js`, sin política de ramas) DEBE ofrecer un enlace a «Ajustes del proyecto». `GET /api/workspace` NO DEBE perder campos de los que dependan otras pantallas (RF-WEB-01). Las pruebas de cadenas esperadas del JS y de assets DEBEN actualizarse a este comportamiento y seguir en verde. El *service worker* no cambia.

#### Scenario: Ecosistema enlaza a Ajustes
- **DADO** un proyecto con `git.integration_branch: develop`
- **CUANDO** se abre Ecosistema
- **ENTONCES** no se pintan las filas de política de ramas
- **Y** hay un enlace a «Ajustes del proyecto»

#### Scenario: El selector de proyecto enlaza a Ajustes
- **DADO** el selector de proyecto con un proyecto activo
- **CUANDO** se abre
- **ENTONCES** ofrece un enlace a «Ajustes del proyecto»
- **Y** no pinta la política de ramas

#### Scenario: Las pruebas de assets reflejan la retirada
- **DADO** los assets embebidos
- **CUANDO** se ejecutan las pruebas de cadenas del JS
- **ENTONCES** ningún asset de Ecosistema referencia las filas de política de ramas ni `repoPolicyHtml`
- **Y** `GET /api/workspace` sigue exponiendo la política de ramas y `repositories`
