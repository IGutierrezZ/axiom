# Delta para spec-baseline-integration

## ADDED Requirements

### Requirement: Consulta de un Requisito Existente en la Baseline para el Bug sin Delta (REQ-29.14)

El motor DEBE poder resolver si un identificador de requisito existe en la baseline (`openspec/specs/`, la spec viva del proyecto) y, si existe, devolver sus escenarios. La resolución se hace por el identificador `REQ-<n>.<m>` que contiene el nombre del requisito (REQ-29.3 de `sdd-iso-evidence`), y un requisito cuyo nombre no contiene ese identificador no puede citarse (supuesto visible). La consulta es de solo lectura y NO DEBE modificar ningún documento. La usa el bug con requisito existente (REQ-29.5 de `sdd-iso-evidence`) para comprobar, en un cambio ISO, que el `Cubre:` apunta a un requisito existente y para conocer los escenarios que la matriz de verify debe cubrir (REQ-29.6). Un identificador que no existe se informa como tal, sin error de ejecución, y un requisito existente sin escenarios se informa como existente con una lista de escenarios vacía.

#### Scenario: Un identificador existente se resuelve con sus escenarios

- **DADO** la baseline con el requisito `Subcomando axiom sdd status (REQ-13.1)` con dos escenarios
- **CUANDO** se consulta `REQ-13.1`
- **ENTONCES** se informa de que existe y se devuelven sus dos escenarios

#### Scenario: Un identificador inexistente se informa sin error

- **DADO** la misma baseline y el identificador `REQ-13.99`
- **CUANDO** se consulta `REQ-13.99`
- **ENTONCES** se informa de que no existe
- **Y** la consulta no falla ni modifica ningún documento

#### Scenario: Un requisito sin identificador en su nombre no puede citarse

- **DADO** la baseline con un requisito cuyo nombre no contiene `REQ-<n>.<m>`
- **CUANDO** se consulta cualquier identificador
- **ENTONCES** ese requisito no se devuelve para ningún identificador

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
