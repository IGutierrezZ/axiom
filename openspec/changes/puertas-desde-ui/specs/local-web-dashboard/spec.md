# Delta para local-web-dashboard

## ADDED Requirements

### Requirement: Consulta de la Parada Actual del Cambio desde el Detalle del Incremento (prov:WEB:1)

El dashboard DEBE ofrecer una ruta de solo lectura que devuelva la parada que proyecta el status nativo (`nextRecommended == await-gate` y su `gateQuestion`) para un cambio `checkpointed`: puerta, criterios, evidencia, las opciones con sus etiquetas, efectos y tokens tal como las proyecta la `GateQuestion` (fuente única, sin recalcular `nextRecommended` ni reordenar), el digest visto y `publication{remote, branch, prTarget, invocation, state}`. DEBE presentar una sola parada a la vez, incluidas las `role-apply:<rol>` de cada rol, sin imponer un orden propio. Cuando no hay parada proyectada, DEBE responder «sin parada».

#### Scenario: Parada `spec` pendiente

- **DADO** un cambio `checkpointed` cuyo status proyecta `await-gate` con la puerta `spec`
- **CUANDO** el detalle del incremento consulta la parada
- **ENTONCES** la respuesta contiene la puerta `spec`, su digest, sus tres opciones en el orden de la `GateQuestion` y `publication.remote` y `publication.branch`

#### Scenario: Una parada a la vez en multirepo

- **DADO** un cambio multirepo con las puertas `role-apply:backend` y `role-apply:frontend` y el status proyectando solo `role-apply:backend`
- **CUANDO** se consulta la parada
- **ENTONCES** la respuesta describe únicamente `role-apply:backend`

#### Scenario: Sin parada proyectada

- **DADO** un cambio sin `gateQuestion` en el status
- **CUANDO** se consulta la parada
- **ENTONCES** la respuesta indica «sin parada» y no contiene opciones

### Requirement: Ruta POST de Registro de Puerta con las Protecciones de las Rutas Mutantes (prov:WEB:2)

El dashboard DEBE exponer `POST /api/increments/gate` para registrar la decisión de la parada con la misma lógica que `axiom sdd gate record` (prov:GAT:1). La ruta DEBE aplicar el mismo preflight que las rutas de marcas: método distinto de `POST` responde 405; `Host` u `Origin` no locales, 403; `Content-Type` distinto de `application/json`, 415; cuerpo superior a 16 KiB o con campos desconocidos, rechazo sin efecto. DEBE exigir `confirm` verdadero (422 antes de cualquier efecto), serializar con un cerrojo (`busy` si está tomado) y, bajo ese cerrojo, comprobar que la puerta y el digest que la persona vio siguen siendo los pendientes: si la puerta ya se decidió o el digest cambió, DEBE responder 409 tipado sin escribir. Iterar DEBE exigir motivo no vacío y, sin él, rechazar sin escribir.

#### Scenario: Aprobación válida

- **DADO** una parada `design` pendiente con el digest `D` visto en pantalla
- **CUANDO** llega `POST /api/increments/gate` con la puerta `design`, el digest `D`, decisión `approved` y `confirm: true`
- **ENTONCES** se añade un registro `approved` a `gates.yaml` y la respuesta lo describe

#### Scenario: Iterar sin motivo

- **DADO** una parada pendiente
- **CUANDO** llega una petición con decisión `rejected` y motivo vacío
- **ENTONCES** se rechaza y `gates.yaml` no cambia

#### Scenario: Puerta ya decidida por otro canal

- **DADO** una parada vista en pantalla que mientras tanto se aprobó con la CLI
- **CUANDO** llega la petición de registro con la puerta y el digest vistos
- **ENTONCES** la respuesta es 409 tipado y `gates.yaml` no se modifica

#### Scenario: Digest cambiado

- **DADO** una parada cuyo artefacto cambió tras mostrarse
- **CUANDO** llega la petición con el digest antiguo
- **ENTONCES** la respuesta es 409 tipado y no se escribe nada

#### Scenario: Sin confirmación

- **DADO** una petición válida con `confirm` ausente o falso
- **CUANDO** se procesa
- **ENTONCES** la respuesta es 422 y no hay ningún efecto

#### Scenario: Preflight local

- **DADO** peticiones con método `GET`, `Origin` ajeno, `Content-Type: text/plain` o campo desconocido
- **CUANDO** se envían a la ruta
- **ENTONCES** se responde 405, 403, 415 y rechazo respectivamente, sin escribir

#### Scenario: Cerrojo tomado

- **DADO** un registro en curso que mantiene el cerrojo
- **CUANDO** llega una segunda petición
- **ENTONCES** recibe `busy` sin escribir

### Requirement: Ruta POST de Publicación de Documentos del Cambio con Aviso Remoto Confirmado (prov:WEB:3)

El dashboard DEBE exponer `POST /api/increments/gate/publish`, con el mismo preflight, `confirm` obligatorio y cerrojo que prov:WEB:2, que publique con la lógica de `axiom sdd publish` solo los documentos del cambio y solo la rama del cambio a `origin`, con `--branch` atado a la rama que la UI mostró, sin fetch, sin forzar y sin `Only` ni `Replay`. DEBE rechazar con 409 tipado si la rama pedida no es la proyectada. Con almacén `engram` o `none` DEBE respetar `publishUnavailable` y no publicar nada.

#### Scenario: Publicación correcta

- **DADO** una aprobación registrada y `publication.state: pending`
- **CUANDO** llega la petición con la rama mostrada y `confirm: true`
- **ENTONCES** se confirman solo los documentos del cambio y se sube solo esa rama sin forzar, y la respuesta es tipada

#### Scenario: Rama distinta de la mostrada

- **DADO** una petición cuya rama no coincide con la proyectada
- **CUANDO** se procesa
- **ENTONCES** se responde 409 tipado y no se confirma ni sube nada

#### Scenario: Almacén engram

- **DADO** un cambio con almacén `engram` y `publishUnavailable`
- **CUANDO** llega la petición de publicación
- **ENTONCES** no se publica nada y la respuesta nombra el código `publishUnavailable`

### Requirement: Presentación de las Salidas de la Parada: Tres Salidas y Dos en `integration` (prov:WEB:4)

El detalle del incremento DEBE mostrar las salidas que proyecta la `GateQuestion`, en su orden. Para `spec`, `design`, `tasks` y `role-apply:<rol>`: Iterar (rechazo con motivo obligatorio), Aprobar, publicar y parar (aprobación con cesión más publicación) y Aprobar, publicar y continuar (aprobación más publicación). Si la `GateQuestion` no ofrece la salida 2 (almacén `engram` o `none`), la UI NO DEBE ofrecerla ni la publicación. Para `integration` DEBE ofrecer solo dos salidas: aprobar con la evidencia de merge que exige la CLI (validada con las reglas de la CLI) o rechazar con motivo; NO DEBE publicar. Sin `gateQuestion`, o en un cambio `continuous`, la UI NO DEBE ofrecer ninguna decisión.

#### Scenario: Tres salidas

- **DADO** una parada `tasks` con almacén `openspec`
- **CUANDO** se abre el detalle
- **ENTONCES** se muestran las tres salidas con las etiquetas y efectos de la `GateQuestion`

#### Scenario: Almacén engram

- **DADO** una parada con almacén `engram`
- **CUANDO** se abre el detalle
- **ENTONCES** solo se ofrecen Iterar y Aprobar y continuar, sin hand-off

#### Scenario: `integration` con evidencia

- **DADO** la parada `integration`
- **CUANDO** la persona aprueba sin evidencia de merge válida
- **ENTONCES** el registro se rechaza con la misma negativa que la CLI y no se escribe

#### Scenario: `integration` no publica

- **DADO** la parada `integration` aprobada con evidencia válida
- **CUANDO** se registra
- **ENTONCES** no se ofrece ni se ejecuta publicación

#### Scenario: Cambio `continuous`

- **DADO** un cambio `continuous`
- **CUANDO** se abre el detalle
- **ENTONCES** no aparece ningún control de decisión

### Requirement: Modo de Revisión por Commit al Aprobar `tasks` (prov:WEB:5)

Al aprobar la puerta `tasks` en un cambio `checkpointed`, la UI DEBE ofrecer una interacción propia con el modo de revisión por commit (`notify` por defecto, `stop`, `stop-before-commit`) y DEBE registrarlo en la misma llamada de aprobación, como `--commit-review` en la CLI, con su validación. NO DEBE ofrecerlo al iterar ni en otras puertas.

#### Scenario: Valor por defecto

- **DADO** la aprobación de `tasks` sin elegir modo
- **CUANDO** se registra
- **ENTONCES** el registro lleva `notify`

#### Scenario: Modo inválido

- **DADO** una petición con un modo fuera de los tres
- **CUANDO** se procesa
- **ENTONCES** se rechaza como la CLI y no se escribe

#### Scenario: Otras puertas y rechazo

- **DADO** la parada `design` o un rechazo de `tasks`
- **CUANDO** se muestra el formulario
- **ENTONCES** no aparece la elección de modo

### Requirement: Un Único Clic de Confirmación con Aviso de Operación Remota Encadena Registrar y Publicar (prov:WEB:6)

En las salidas 2 y 3, la UI DEBE mostrar un único diálogo de confirmación que avise de que publicar es una operación remota (commit de los documentos del cambio y push de la rama del cambio a `origin`, sin forzar) nombrando el remoto y la rama, y solo tras el clic de la persona DEBE encadenar el registro y la publicación. NO DEBE publicar sin ese clic ni ejecutar `fetch`, push de código, PR ni merge.

#### Scenario: Confirmación

- **DADO** la salida 3 elegida
- **CUANDO** se abre el diálogo
- **ENTONCES** muestra remoto y rama y no se ha enviado ninguna petición

#### Scenario: Cancelar

- **DADO** el diálogo abierto
- **CUANDO** la persona cancela
- **ENTONCES** no se registra ni se publica nada

### Requirement: Fallo de Publicación como «Aprobada sin Publicar» con Reintento (prov:WEB:7)

Un fallo de publicación NO DEBE deshacer la aprobación registrada. La UI DEBE mostrar «aprobada sin publicar» con el fallo tipado y su continuación o `pushCommand` (por ejemplo `code_commits_unpublished`), y DEBE ofrecer un botón de reintento mientras el status proyecte `publication.state: pending`, que repite solo la publicación, sin volver a registrar.

#### Scenario: Fallo tras aprobar

- **DADO** una aprobación registrada cuya publicación falla
- **CUANDO** se muestra el resultado
- **ENTONCES** `gates.yaml` conserva la aprobación y la UI indica «aprobada sin publicar» con el código y la continuación

#### Scenario: Reintento

- **DADO** `publication.state: pending`
- **CUANDO** la persona pulsa reintentar y confirma
- **ENTONCES** solo se publica, sin nuevo registro en `gates.yaml`

#### Scenario: Ya publicada

- **DADO** `publication.state` distinto de `pending`
- **CUANDO** se muestra la puerta
- **ENTONCES** no hay botón de reintento

### Requirement: Rutas de Puertas Clasificadas en el Canario de Secretos y sin Estilos ni Scripts en Línea (prov:WEB:8)

Las rutas nuevas de puertas (consulta, registro y publicación) DEBEN registrarse en el canario de secretos, y sus respuestas NO DEBEN contener secretos. Los assets nuevos NO DEBEN usar estilos ni scripts en línea (CSP `style-src 'self'`). El escritor nuevo DEBE constar en el inventario de escritores.

#### Scenario: Canario

- **DADO** el canario de secretos con un secreto sembrado
- **CUANDO** se ejercitan las rutas de puertas
- **ENTONCES** ninguna respuesta lo contiene y las rutas figuran clasificadas

#### Scenario: CSP

- **DADO** los assets de la pantalla de puertas
- **CUANDO** se ejecuta la prueba de CSP
- **ENTONCES** no hay `style=` ni `<script>` en línea
