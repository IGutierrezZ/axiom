# Delta para spec-baseline-integration

## ADDED Requirements

### Requirement: Consulta de un Requisito Existente en la Baseline para el Bug sin Delta (REQ-29.14)

El motor DEBE poder resolver si un identificador de requisito existe en la baseline declarada y, si existe, devolver sus escenarios. En un documento con perfil de requisito la resolución se hace por el ID (REQ-24.39); en la baseline implícita `openspec/specs/`, que cuenta como identidad `id` en un cambio ISO (REQ-29.3 de `sdd-iso-evidence`), la resolución se hace por el identificador `REQ-<n>.<m>` que contiene el nombre del requisito, y un requisito cuyo nombre no contiene ese identificador no puede citarse (supuesto visible). La consulta es de solo lectura y NO DEBE modificar ningún documento. La usan el perfil de bug sin delta (REQ-29.5 de `sdd-iso-evidence`) para comprobar, en un cambio ISO, que el `Cubre:` apunta a un requisito existente y para conocer los escenarios que la matriz de verify debe cubrir (REQ-29.6). Un identificador que no existe se informa como tal, sin error de ejecución, y un requisito existente sin escenarios se informa como existente con una lista de escenarios vacía.

#### Scenario: Un ID existente se resuelve con sus escenarios

- **DADO** una baseline con perfil de requisito que contiene `RF-120` con dos escenarios
- **CUANDO** se consulta `RF-120`
- **ENTONCES** se informa de que existe y se devuelven sus dos escenarios

#### Scenario: Un ID inexistente se informa sin error

- **DADO** la misma baseline y el identificador `RF-999`
- **CUANDO** se consulta `RF-999`
- **ENTONCES** se informa de que no existe
- **Y** la consulta no falla ni modifica ningún documento

#### Scenario: En la baseline implícita se resuelve por el identificador del nombre

- **DADO** la baseline implícita con el requisito `Subcomando axiom sdd status (REQ-13.1)`
- **CUANDO** se consulta `REQ-13.1`
- **ENTONCES** se informa de que existe y se devuelven sus escenarios

#### Scenario: Un requisito existente sin escenarios se informa como tal

- **DADO** un requisito existente sin ningún `#### Scenario:`
- **CUANDO** se consulta su identificador
- **ENTONCES** se informa de que existe con una lista de escenarios vacía

#### Scenario: La consulta no escribe

- **DADO** cualquier consulta
- **CUANDO** termina
- **ENTONCES** el árbol de trabajo no presenta ningún cambio

## MODIFIED Requirements

### Requirement: Composición Nativa Intacta sin Perfil (REQ-24.38)

Cuando el documento de la baseline no declara perfil de requisito (baseline OpenSpec por defecto), la composición y `axiom sdd archive-compose` DEBEN comportarse byte a byte como antes de esta capacidad: identidad por el texto exacto del nombre del requisito, orden de aplicación RENAMED, MODIFIED, REMOVED y ADDED, rechazo sin escribir ante un delta no aplicable y opciones `--supersede` y `--superseded-requirements`. Las pruebas nativas existentes de composición y de la CLI DEBEN pasar sin modificar sus aserciones. Única excepción: en un cambio ISO (REQ-29.1 de `sdd-iso-evidence`), la composición DEBE rechazar sin escribir un requisito ADDED o MODIFIED sin escenarios (REQ-29.10 de `sdd-archive-lifecycle`); toda composición aceptada sigue produciendo la misma salida byte a byte.
(Previously: la composición nativa no tenía ninguna excepción; en ISO añade el rechazo por falta de escenarios sin alterar la salida de las composiciones aceptadas)

#### Scenario: La salida nativa no cambia

- **DADO** una especificación canónica y un delta con ADDED, MODIFIED, REMOVED y RENAMED
- **CUANDO** se ejecuta `axiom sdd archive-compose --canonical <ruta> --delta <ruta>`
- **ENTONCES** la salida es idéntica byte a byte a la que producía la composición antes de esta capacidad
- **Y** las pruebas existentes de composición y de la CLI pasan sin modificarse

#### Scenario: Rechazo nativo sin escribir

- **DADO** un delta con un MODIFIED sobre un nombre de requisito inexistente
- **CUANDO** se ejecuta la composición con `--output <ruta>`
- **ENTONCES** finaliza con código distinto de cero nombrando la sección y el requisito
- **Y** no se escribe el fichero de salida

#### Scenario: En ISO un ADDED sin escenarios se rechaza y los aceptados no cambian

- **DADO** un cambio ISO y un delta con un ADDED sin escenarios
- **CUANDO** se ejecuta `axiom sdd archive-compose --cwd <raíz> --document <dominio> --delta <ruta>`
- **ENTONCES** finaliza con código distinto de cero nombrando la sección y el requisito
- **Y** un delta con escenarios produce la misma salida byte a byte que antes de esta capacidad

#### Scenario: `--supersede` conserva su comportamiento

- **DADO** un delta con un REMOVED y la opción `--supersede`
- **CUANDO** se compone
- **ENTONCES** el requisito se conserva marcado como superado, con su nota de motivo, como hasta ahora

### Requirement: Validación Completa antes de Escribir (REQ-24.46)

Todas las operaciones de todos los documentos con perfil de requisito del cambio DEBEN validarse y componerse en memoria antes de escribir ningún documento de la integración (de composición, dirigida o de contexto): o se aplican todas las composiciones o ninguna. Un rechazo NO DEBE dejar la baseline parcialmente modificada y DEBE nombrar la sección, el requisito y el motivo. La integración dirigida y la del contexto solo comienzan cuando todas las composiciones han tenido éxito. En un cambio ISO (REQ-29.1 de `sdd-iso-evidence`), esa validación previa incluye la regla de al menos un escenario por requisito ADDED o MODIFIED (REQ-29.10 de `sdd-archive-lifecycle`).
(Previously: la validación previa cubría solo la aplicabilidad de las operaciones; en ISO incluye también la regla de escenarios)

#### Scenario: Un dominio falla y ninguno se modifica

- **DADO** un cambio con deltas para dos dominios, el segundo con un MODIFIED sobre un ID inexistente
- **CUANDO** se ejecuta el archivado
- **ENTONCES** se rechaza nombrando el dominio, la sección y el requisito
- **Y** ningún documento de la baseline cambia

#### Scenario: Todas las operaciones válidas se aplican

- **DADO** un cambio con deltas válidos para dos dominios
- **CUANDO** se ejecuta el archivado
- **ENTONCES** ambos documentos se sustituyen por su composición

#### Scenario: La integración dirigida no empieza si una composición falla

- **DADO** un cambio con una composición inválida y un dominio de prosa con `integration: directed`
- **CUANDO** se ejecuta el archivado
- **ENTONCES** se rechaza nombrando la composición inválida
- **Y** no se edita ningún documento, tampoco el de prosa

#### Scenario: En ISO un requisito sin escenarios en un dominio impide modificar los demás

- **DADO** un cambio ISO con deltas para dos dominios, el segundo con un ADDED sin escenarios
- **CUANDO** se ejecuta el archivado
- **ENTONCES** se rechaza nombrando el dominio, la sección y el requisito
- **Y** ningún documento de la baseline cambia, tampoco el del primer dominio
