# Delta para Dashboard Web Local (local-web-dashboard)

> **Cambio:** `marcas-pruebas` (C-06-d). **Idioma:** castellano de España; identificadores, claves y tokens en inglés.
> RF-WEB-01 a RF-WEB-16 y RF-WEB-18 a RF-WEB-21 no cambian de texto. Se añaden dos rutas POST (marca y publicación de marcas).

## ADDED Requirements

### Requirement: Pantalla de Marcas por Caso y Entorno con Caducidad en la Pestaña «Pruebas» (RF-WEB-22)

Cuando las marcas están disponibles (RF-WEB-25), la pestaña «Pruebas» DEBE mostrar una tabla de casos con una columna por entorno de `acceptance_tests.environments`. Cada celda DEBE mostrar la última marca vigente del par caso-entorno (resultado y nota), o su ausencia. Una marca cuyo `plan_digest` difiere del digest actual de `pruebas.md` (RF-KOF-25) DEBE mostrarse como caducada y NO DEBE contar como vigente. La pestaña DEBE ofrecer en cada celda aplicable el control para marcar `ok`, `ko`, `blocked` o `n/a`, con un campo opcional `commit` (vacío por defecto) y un campo opcional `note`. DEBE listar las entradas pendientes de publicar. Todo texto de marcas, notas y `pruebas.md` DEBE insertarse como texto, nunca como HTML. Las celdas de un entorno no aplicable al caso NO DEBEN ofrecer el control de marca.

#### Scenario: Celda con marca vigente
- **DADO** un cambio con `pruebas.md` y una marca `ok` del caso `P-01` en `pre` con el `plan_digest` actual
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** la celda de `P-01` en `pre` muestra `ok` y su nota, sin indicador de caducidad

#### Scenario: Marca caducada tras cambiar `pruebas.md`
- **DADO** una marca de `P-01` en `pre` y un `pruebas.md` modificado después
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** la marca aparece como caducada y no cuenta como vigente

#### Scenario: Entorno no aplicable al caso
- **DADO** un caso cuyos entornos aplicables no incluyen `pro`
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** la celda de ese caso en `pro` no ofrece control de marca

#### Scenario: Contenido hostil en una nota
- **DADO** una marca cuya nota contiene `<script>`
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** la nota aparece literal y no se interpreta como HTML

#### Scenario: Entradas pendientes visibles
- **DADO** dos marcas escritas en local y aún no publicadas
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** se listan ambas como pendientes de publicar

### Requirement: Ruta de Marca con Validación en Servidor y Escritura Local de Solo Añadir (RF-WEB-23)

El dashboard DEBE exponer una ruta POST de marca, protegida con `isLocalRequest` y con cuerpo JSON, que añada una entrada a `pruebas-estado.yaml` del cambio (formato en RF-KOF-25). La ruta DEBE validar en el servidor, sin escribir nada si falla: que el cambio exista, no esté archivado y tenga `pruebas.md`; que `case` exista en `pruebas.md`; que `environment` pertenezca a `acceptance_tests.environments` y a los entornos aplicables del caso; que `result` pertenezca a `ok|ko|blocked|n/a`; y que `note` no supere la longitud máxima. DEBE rellenar `by` con la identidad git (`user.name` y `user.email`) del repositorio de specs, `at` con la fecha y hora, y `plan_digest` con el digest actual de `pruebas.md`; `commit` es el valor opcional que da la persona y queda vacío por defecto, y NUNCA se infiere del repositorio de código. La escritura DEBE ser atómica y con exclusión mutua por cambio, y el fichero DEBE ser de solo añadir: la ruta NO DEBE editar ni borrar entradas existentes y una corrección es una entrada nueva. La ruta NO DEBE ejecutar ninguna operación git de escritura ni de red. Cada rechazo DEBE tener código tipado, mensaje en castellano y continuación.

#### Scenario: Marca válida añade una entrada
- **DADO** un cambio SDD multirepo con `pruebas.md` con el caso `P-01` aplicable a `pre`
- **CUANDO** se envía la marca `P-01`, `pre`, `ok`
- **ENTONCES** `pruebas-estado.yaml` recibe una entrada con `by` (identidad git), `at`, `commit` vacío y el `plan_digest` actual
- **Y** el observador de git no registra ninguna operación

#### Scenario: Caso inexistente
- **DADO** un `pruebas.md` sin el caso `P-99`
- **CUANDO** se envía una marca para `P-99`
- **ENTONCES** se rechaza con un código tipado de caso inexistente y su continuación
- **Y** el fichero no cambia

#### Scenario: Entorno no válido
- **DADO** `acceptance_tests.environments` con `local` y `pre`
- **CUANDO** se envía una marca en `staging`, o en un entorno no aplicable al caso
- **ENTONCES** se rechaza con un código tipado de entorno no válido
- **Y** el fichero no cambia

#### Scenario: Resultado fuera de dominio
- **DADO** una marca con `result` igual a `pass`
- **CUANDO** se envía
- **ENTONCES** se rechaza con un código tipado de resultado fuera de dominio
- **Y** el fichero no cambia

#### Scenario: Nota demasiado larga
- **DADO** una `note` que supera la longitud máxima
- **CUANDO** se envía la marca
- **ENTONCES** se rechaza con un código tipado de nota demasiado larga y el fichero no cambia

#### Scenario: Solo añadir
- **DADO** una marca `ko` ya escrita para `P-01` en `pre`
- **CUANDO** se envía una nueva marca `ok` para el mismo par
- **ENTONCES** el fichero conserva la entrada `ko` y añade la entrada `ok` después
- **Y** la celda muestra `ok` como marca vigente

#### Scenario: Petición no local
- **DADO** una petición con `Host` u `Origin` que no son del propio dashboard
- **CUANDO** se envía a la ruta de marca
- **ENTONCES** se rechaza y no se escribe nada

#### Scenario: Escrituras concurrentes del mismo cambio
- **DADO** dos marcas enviadas a la vez para el mismo cambio
- **CUANDO** el servidor las procesa
- **ENTONCES** `pruebas-estado.yaml` contiene ambas entradas íntegras y bien formadas

### Requirement: Ruta de Publicación de Marcas con Confirmación Explícita y Resultado Tipado (RF-WEB-24)

El dashboard DEBE exponer una ruta POST «Publicar marcas», protegida con `isLocalRequest` y con cuerpo JSON, que exija el campo `confirm` verdadero en el cuerpo. Sin `confirm`, DEBE rechazarse con un código tipado y NO DEBE ejecutar ninguna operación git. Antes del clic, la UI DEBE mostrar el remoto, la rama de integración del repositorio de specs y las entradas pendientes. Con `confirm`, el dashboard DEBE delegar la publicación en un publicador de marcas definido por una interfaz propia del dashboard e inyectado desde fuera; el dashboard NO DEBE importar `sddpublish` ni ejecutar git por su cuenta (RF-PUB-07). La respuesta DEBE mostrar el resultado (commit, rama y remoto) o el rechazo tipado con su etiqueta en castellano y su continuación, y DEBE avisar de los demás documentos sucios del repositorio de specs que no se confirmaron (RF-PUB-13). Sin entradas pendientes, la respuesta DEBE indicar que no hay nada que publicar y no hacer commit. Nunca DEBE haber push sin el clic de confirmación.

#### Scenario: Publicar sin confirm no ejecuta git
- **DADO** entradas pendientes y un observador de git
- **CUANDO** se envía la publicación sin `confirm`
- **ENTONCES** se rechaza con un código tipado de confirmación ausente
- **Y** el observador no registra ninguna operación git

#### Scenario: Publicar con confirm muestra el resultado
- **DADO** entradas pendientes y un repositorio de specs con remoto
- **CUANDO** se envía la publicación con `confirm`
- **ENTONCES** la respuesta incluye el commit creado, la rama y el remoto

#### Scenario: Datos visibles antes del clic
- **DADO** entradas pendientes
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** se muestran el remoto, la rama de integración y las entradas pendientes antes de ofrecer «Publicar marcas»

#### Scenario: Rechazo tipado visible
- **DADO** una publicación que termina con un rechazo tipado (por ejemplo, `push_non_fast_forward` tras el reintento)
- **CUANDO** el dashboard responde
- **ENTONCES** muestra el código, la etiqueta en castellano y la continuación
- **Y** las entradas siguen en el fichero local

#### Scenario: Sin entradas pendientes
- **DADO** que todas las marcas ya están publicadas
- **CUANDO** se envía la publicación con `confirm`
- **ENTONCES** la respuesta indica que no hay nada que publicar y no se crea ningún commit

#### Scenario: El dashboard no importa el publicador
- **DADO** el código fuente del producto
- **CUANDO** se ejecuta la prueba de aislamiento de RF-PUB-07
- **ENTONCES** el paquete del dashboard no importa `sddpublish` y la prueba sigue en verde

### Requirement: Marcas Solo en Multirepo SDD con Estado «No Disponible» Explícito (RF-WEB-25)

Las marcas y sus dos rutas DEBEN estar disponibles únicamente para un cambio SDD activo con `pruebas.md` en un proyecto `multirepo` con `acceptance_tests` activo. En `monorepo-embedded`, la pestaña y las dos rutas DEBEN responder un estado «no disponible» explícito con código tipado y motivo en castellano, NO un 404 ni un error genérico. Para un cambio ODD, un cambio archivado, un cambio sin `pruebas.md` o un proyecto sin `acceptance_tests` activo, DEBEN responder igualmente «no disponible» con su motivo y NO DEBEN escribir nada. Las marcas NO DEBEN usar ninguna clave nueva de `axiom.yaml`: reutilizan `acceptance_tests.environments`.

#### Scenario: Monorepo responde no disponible
- **DADO** un proyecto `monorepo-embedded` con `pruebas.md`
- **CUANDO** se abre la pestaña «Pruebas» y se envía una marca
- **ENTONCES** ambas responden «no disponible» con motivo de topología
- **Y** no se escribe `pruebas-estado.yaml`

#### Scenario: Cambio archivado
- **DADO** un cambio archivado
- **CUANDO** se envía una marca
- **ENTONCES** se responde «no disponible» por cambio archivado y no se escribe nada

#### Scenario: Sin `pruebas.md`
- **DADO** un cambio multirepo sin `pruebas.md`
- **CUANDO** se envía una marca
- **ENTONCES** se responde «no disponible» por falta de `pruebas.md` y no se escribe nada

#### Scenario: Función `acceptance_tests` no activa
- **DADO** un proyecto multirepo sin `acceptance_tests` activo
- **CUANDO** se envía una marca
- **ENTONCES** se responde «no disponible» por función no activa

### Requirement: Rutas de Marcas Clasificadas en el Canario de Secretos (RF-WEB-26)

Las dos rutas POST nuevas DEBEN clasificarse en el canario de secretos como excluidas con el motivo «mutación (POST)», de modo que la prueba que exige clasificar toda ruta de `server.go` siga en verde. Ningún DTO de marcas DEBE contener secretos, y las pruebas de pines de pestañas y de assets DEBEN seguir en verde.

#### Scenario: Rutas clasificadas
- **DADO** las rutas registradas en `server.go`
- **CUANDO** se ejecuta la prueba de clasificación del canario
- **ENTONCES** las rutas de marca y de publicación figuran como excluidas con el motivo «mutación (POST)» y la prueba pasa

#### Scenario: Una ruta nueva sin clasificar falla
- **DADO** una ruta añadida sin clasificar
- **CUANDO** se ejecuta la prueba de clasificación
- **ENTONCES** la prueba falla

## MODIFIED Requirements

### Requirement: Pestaña «Pruebas» en el Detalle del Incremento, con Marcas por Entorno solo en Multirepo (RF-WEB-17)

`GET /api/increments/<n>` DEBE incluir en el detalle de un incremento SDD un campo de la batería de pruebas con el contenido de `pruebas.md` y su cobertura, calculada con el mismo `acceptancetests.Check` que usan `axiom sdd tests check` y `axiom sdd status` (RF-KOF-20). El campo DEBE incluir los casos de la tabla, las cifras de escenarios M/M, requisitos M/M y número de casos, y los hallazgos. Sin `pruebas.md`, el campo DEBE indicar su ausencia y, si la función está activa, el aviso de RF-KOF-22. La lectura NO DEBE añadir ninguna ruta, y el endpoint de ficheros de RF-WEB-15 NO DEBE ampliar su lista blanca. Las únicas rutas añadidas son las dos rutas POST de marcas (RF-WEB-23 y RF-WEB-24).
(Previously: la pestaña era de solo lectura y el requisito prohibía añadir cualquier ruta)

El detalle DEBE incluir una pestaña «Pruebas»: muestra la tabla de casos con sus columnas, la cobertura y los hallazgos; no ofrece edición de `pruebas.md` ni ejecución. Las marcas por entorno (RF-WEB-22) solo se ofrecen cuando están disponibles (RF-WEB-25); en los demás casos la pestaña sigue siendo de solo lectura. Todo texto proveniente de `pruebas.md` DEBE insertarse como texto, nunca como HTML. Con la función no activa y sin `pruebas.md`, la pestaña DEBE indicar que las pruebas de aceptación no están activadas para el proyecto. Los hallazgos nunca bloquean ninguna otra acción del detalle. Las pruebas de pines de pestañas y de assets, y el canario de secretos, DEBEN seguir en verde.

#### Scenario: El detalle devuelve la batería y su cobertura
- **DADO** un incremento con `pruebas.md` válido y su cobertura completa
- **CUANDO** se consulta `GET /api/increments/<n>`
- **ENTONCES** el detalle incluye los casos, los escenarios M/M, los requisitos M/M, el número de casos y cero hallazgos

#### Scenario: La cobertura coincide con la del verbo
- **DADO** un incremento con `pruebas.md` con hallazgos
- **CUANDO** se consulta el detalle y se ejecuta `axiom sdd tests check --json`
- **ENTONCES** cifras y hallazgos son iguales

#### Scenario: La pestaña es de solo lectura donde las marcas no están disponibles
- **DADO** un incremento con `pruebas.md` en un proyecto `monorepo-embedded`
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** se muestran la tabla, la cobertura y los hallazgos
- **Y** no hay controles de edición, de ejecución ni de marca

#### Scenario: Contenido hostil se muestra como texto
- **DADO** un `pruebas.md` con `<script>` en una celda
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** el texto aparece literal y no se interpreta como HTML

#### Scenario: Función no activada
- **DADO** un proyecto sin el bloque `acceptance_tests` y un incremento sin `pruebas.md`
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** indica que las pruebas de aceptación no están activadas

#### Scenario: Activa y sin fichero
- **DADO** la función activa y un incremento tras tasks sin `pruebas.md`
- **CUANDO** se abre la pestaña «Pruebas»
- **ENTONCES** indica la ausencia y muestra el aviso de RF-KOF-22
- **Y** no bloquea ninguna otra pestaña

#### Scenario: La lectura no añade rutas y las de marcas son solo POST
- **DADO** el servidor del dashboard
- **CUANDO** se enumeran sus rutas
- **ENTONCES** la lectura de la batería no añade ninguna ruta GET, la lista blanca de RF-WEB-15 no cambia y las únicas rutas añadidas son las dos POST de marcas
