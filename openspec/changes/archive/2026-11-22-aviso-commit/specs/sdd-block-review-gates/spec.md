# Delta para sdd-block-review-gates

Series de requisitos: este delta usa REQ-33.1 a REQ-33.7 (la serie 33 está libre en `openspec/specs/`). No modifica REQ-21.10, REQ-26.2 ni REQ-26.3: sus invariantes se verifican como escenarios de no regresión de los requisitos nuevos.

## ADDED Requirements

### Requirement: Vocabulario Cerrado del Modo de Revisión por Commit (REQ-33.1)

El modo de revisión por commit DEBE ser un vocabulario cerrado de tres tokens canónicos, que no se traducen: `notify` (modo 1: avisar tras cada tarea y continuar con commit automático; es el modo por defecto), `stop` (modo 2: parar tras cada tarea con commit automático) y `stop-before-commit` (modo 3: parar tras cada tarea sin hacer el commit hasta que la persona apruebe). Cualquier otro valor, incluida la cadena vacía y las variantes de mayúsculas, DEBE rechazarse. La ausencia del modo equivale a `notify`.

#### Scenario: Los tres tokens se aceptan

- **DADO** una puerta `tasks` pendiente de un cambio `checkpointed`
- **CUANDO** se registra la aprobación con cada uno de `notify`, `stop` y `stop-before-commit`
- **ENTONCES** cada registro se acepta y conserva el token tal cual

#### Scenario: Un valor fuera del vocabulario se rechaza

- **DADO** una puerta `tasks` pendiente de un cambio `checkpointed`
- **CUANDO** se registra la aprobación con el modo `Stop`, `pause` o la cadena vacía
- **ENTONCES** la orden se rechaza indicando el vocabulario válido
- **Y** no se escribe en `gates.yaml`

### Requirement: `gate record` Persiste el Modo Solo en la Aprobación de la Puerta `tasks` de un Cambio `checkpointed` (REQ-33.2)

`axiom sdd gate record` DEBE aceptar la bandera `--commit-review <modo>` únicamente con `--gate tasks` y `--decision approved` (con o sin `--hand-off`), en un cambio cuyo estilo sellado es `checkpointed`. Cuando se acepta, el registro `approved` DEBE persistir el modo en un campo opcional `commit_review` del registro de `gates.yaml`, escrito en la misma llamada que aprueba. El campo NO DEBE escribirse cuando la bandera está ausente. La orden DEBE rechazarse sin escribir en `gates.yaml` cuando la bandera se combina con otra puerta (`spec`, `design`, `role-apply:<rol>`, `integration`), con `--decision rejected`, con un cambio `continuous`, o con una puerta `tasks` que ya está `approved` (el primer registro `approved` cierra la puerta). La salida JSON de la orden DEBE incluir el modo cuando se persiste. El kickoff sellado y el digest de las puertas NO DEBEN cambiar.

#### Scenario: La aprobación con modo escribe el campo

- **DADO** un cambio `checkpointed` con la puerta `tasks` pendiente
- **CUANDO** se ejecuta `axiom sdd gate record --gate tasks --decision approved --commit-review stop`
- **ENTONCES** `gates.yaml` contiene un único registro `approved` de `tasks` con `commit_review: stop`
- **Y** la puerta resulta `approved`

#### Scenario: El modo viaja junto con la cesión en un único registro

- **DADO** un cambio `checkpointed` con la puerta `tasks` pendiente
- **CUANDO** se ejecuta `gate record --gate tasks --decision approved --hand-off --commit-review stop-before-commit`
- **ENTONCES** `gates.yaml` contiene un único registro `approved` con la marca de cesión y con `commit_review: stop-before-commit`

#### Scenario: Sin la bandera no se escribe el campo

- **DADO** un cambio `checkpointed` con la puerta `tasks` pendiente
- **CUANDO** se ejecuta `gate record --gate tasks --decision approved` sin `--commit-review`
- **ENTONCES** el registro resultante no contiene `commit_review`

#### Scenario: Otra puerta rechaza la bandera

- **DADO** un cambio `checkpointed` con la puerta `design` pendiente
- **CUANDO** se ejecuta `gate record --gate design --decision approved --commit-review notify`
- **ENTONCES** la orden se rechaza con un error que indica que el modo solo se admite en la puerta `tasks`
- **Y** no se escribe en `gates.yaml`

#### Scenario: Un rechazo no admite el modo

- **DADO** un cambio `checkpointed` con la puerta `tasks` pendiente
- **CUANDO** se ejecuta `gate record --gate tasks --decision rejected --reason "x" --commit-review notify`
- **ENTONCES** la orden se rechaza sin escribir en `gates.yaml`

#### Scenario: Un cambio `continuous` rechaza la bandera

- **DADO** un cambio sellado `continuous`
- **CUANDO** se ejecuta `gate record --gate tasks --decision approved --commit-review stop`
- **ENTONCES** la orden se rechaza indicando que el modo solo existe en cambios `checkpointed`
- **Y** no se escribe en `gates.yaml`

#### Scenario: Una puerta `tasks` ya aprobada rechaza un segundo registro con modo

- **DADO** una puerta `tasks` ya `approved` sin modo
- **CUANDO** se ejecuta `gate record --gate tasks --decision approved --commit-review stop`
- **ENTONCES** la orden se rechaza indicando que la puerta ya está resuelta
- **Y** no se escribe en `gates.yaml` y el modo efectivo sigue siendo `notify`

#### Scenario: El kickoff y el digest no cambian

- **DADO** una puerta `tasks` con su digest calculado
- **CUANDO** se aprueba con `--commit-review stop`
- **ENTONCES** `kickoff.yaml` no se modifica
- **Y** el digest de la puerta es idéntico al que habría con la aprobación sin modo

### Requirement: Compatibilidad Hacia Atrás del Campo `commit_review` (REQ-33.3)

El campo `commit_review` DEBE ser opcional en la lectura y en la escritura. Un `gates.yaml` anterior a este cambio, sin el campo, DEBE leerse sin error y su modo efectivo DEBE ser `notify`. Un binario anterior a este cambio DEBE poder leer un `gates.yaml` con el campo y tratar la puerta como `approved` normal, sin error. Un `commit_review` presente en un registro con un valor fuera del vocabulario DEBE rechazarse al leer el esquema vigente, como cualquier valor inválido del registro. El vocabulario de decisiones del registro sigue siendo `approved|rejected`.

#### Scenario: Un ledger antiguo se lee como `notify`

- **DADO** un `gates.yaml` con la puerta `tasks` aprobada y sin `commit_review`
- **CUANDO** se calcula el estado de las puertas
- **ENTONCES** el modo efectivo de la puerta `tasks` es `notify`

#### Scenario: Un decodificador anterior ignora el campo

- **DADO** un `gates.yaml` con un registro `approved` de `tasks` que lleva `commit_review: stop`
- **CUANDO** lo decodifica el esquema previo a este cambio (decodificador no estricto)
- **ENTONCES** la decodificación termina sin error
- **Y** la puerta resulta `approved`

#### Scenario: Un valor inválido persistido se rechaza al leer

- **DADO** un `gates.yaml` con `commit_review: always`
- **CUANDO** se lee con el esquema vigente
- **ENTONCES** se rechaza como un valor inválido del registro

### Requirement: `axiom sdd status` Proyecta el Modo de Revisión por Commit (REQ-33.4)

`axiom sdd status` DEBE proyectar en la vista de la puerta `tasks` el modo del registro `approved` que la cierra, en un campo `commitReview` junto al resto de los campos de la puerta, cuando el cambio es `checkpointed` y la puerta está `approved`; con un registro sin el campo DEBE proyectar `notify`. Mientras la puerta no esté `approved`, en las demás puertas y en un cambio `continuous` el campo NO DEBE aparecer, y quien consume el estado DEBE tratar su ausencia como `notify`. El modo DEBE sobrevivir a la salida 2 (`--hand-off`): quien recoge `apply` lo lee de `status`. La proyección es informativa: NO DEBE bloquear ni cambiar `nextRecommended` ni las dependencias. El orquestador DEBE pasar el modo proyectado a `sdd-apply`.

#### Scenario: Se proyecta el modo registrado

- **DADO** un cambio `checkpointed` con la puerta `tasks` aprobada con `commit_review: stop`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** la puerta `tasks` contiene `commitReview: "stop"`

#### Scenario: Sin campo se proyecta `notify`

- **DADO** un cambio `checkpointed` con la puerta `tasks` aprobada sin `commit_review`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** la puerta `tasks` contiene `commitReview: "notify"`

#### Scenario: Sobrevive al traspaso

- **DADO** una puerta `tasks` aprobada con `--hand-off --commit-review stop-before-commit`
- **CUANDO** otra persona ejecuta `axiom sdd status --json`
- **ENTONCES** la puerta proyecta a la vez la cesión y `commitReview: "stop-before-commit"`

#### Scenario: No aparece donde no aplica

- **DADO** una puerta `tasks` pendiente, o la puerta `design` aprobada, o un cambio `continuous`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** ninguna de esas vistas contiene `commitReview`

#### Scenario: La proyección no cambia la ruta

- **DADO** la misma puerta `tasks` aprobada con y sin modo
- **CUANDO** se compara `nextRecommended` y las dependencias de ambos estados
- **ENTONCES** son idénticos

### Requirement: El Modo se Pregunta en una Interacción Propia de la Puerta `tasks` y Solo en `checkpointed` (REQ-33.5)

En un cambio `checkpointed`, cuando la persona elige en la puerta `tasks` una salida de aprobación (la 2 o la 3), la doctrina DEBE indicar al orquestador que formule el modo de revisión por commit en una interacción propia, separada de la pregunta tipada de la puerta y sujeta al contrato de bloqueo sin pérdida, con exactamente los tres tokens de REQ-33.1 (con descripción de su efecto traducida a la lengua de la conversación; los tokens no se traducen) y `notify` como valor por defecto declarado. Después DEBE registrar la aprobación una sola vez, con el modo, en la misma llamada de `gate record` que aprueba. Con la salida 1 (iterar) NO DEBE preguntarse el modo. En un cambio `continuous` NO DEBE preguntarse ni escribirse el modo. En ODD NO DEBE preguntarse. La pregunta tipada de la puerta DEBE seguir con exactamente tres opciones y el mismo texto (REQ-26.2) y NO DEBE incluir ninguna opción del modo. Cómo se compone la invocación de la salida elegida con `--commit-review` es decisión de design; el comportamiento observable es que el resto de los tokens de la invocación no se altera.

#### Scenario: La salida de aprobación dispara la pregunta del modo

- **DADO** un cambio `checkpointed` en la puerta `tasks`
- **CUANDO** la persona elige la salida 2 o la salida 3
- **ENTONCES** el orquestador presenta una pregunta aparte con los tres modos
- **Y** registra una sola aprobación que incluye el modo elegido

#### Scenario: Iterar no pregunta

- **DADO** un cambio `checkpointed` en la puerta `tasks`
- **CUANDO** la persona elige la salida 1 (iterar)
- **ENTONCES** no se pregunta el modo y no se escribe nada en `gates.yaml`

#### Scenario: `continuous` no pregunta

- **DADO** un cambio sellado `continuous`
- **CUANDO** concluye la fase `tasks`
- **ENTONCES** no se pregunta el modo y el comportamiento es `notify`

#### Scenario: La pregunta tipada conserva tres opciones

- **DADO** la proyección de la pregunta tipada de la puerta `tasks`
- **CUANDO** se inspeccionan sus opciones
- **ENTONCES** son exactamente tres, las mismas de REQ-26.2, y ninguna menciona el modo de revisión por commit

#### Scenario: El texto de la doctrina fija la interacción

- **DADO** la sección «Commit Review» proyectada a un runtime
- **CUANDO** se lee la regla de la puerta `tasks`
- **ENTONCES** declara que el modo se pregunta tras una salida de aprobación, en interacción propia, una sola vez y con la misma llamada de registro, y que no se pregunta en `continuous` ni con la salida 1

### Requirement: El Modo es Inmutable tras la Aprobación de `tasks` (REQ-33.6)

Una vez aprobada la puerta `tasks`, el modo NO DEBE poder cambiarse por `gate record`: la aprobación es terminal (REQ-26.3) y un registro posterior se rechaza o no cambia la puerta. Cambiar el modo en medio de `apply` queda fuera de este requisito.

#### Scenario: Un rechazo posterior no cambia el modo

- **DADO** una puerta `tasks` aprobada con `commit_review: stop`
- **CUANDO** se registra después `--decision rejected`
- **ENTONCES** la puerta sigue `approved` y `status` sigue proyectando `commitReview: "stop"`

### Requirement: La Ayuda de `gate record` Documenta la Bandera (REQ-33.7)

La ayuda de `axiom sdd gate record` DEBE describir `--commit-review`, su vocabulario, que solo se admite con `--gate tasks --decision approved` en un cambio `checkpointed` y que su ausencia equivale a `notify`.

#### Scenario: La ayuda menciona la bandera

- **DADO** el comando `axiom sdd gate record --help`
- **CUANDO** se lee su salida
- **ENTONCES** contiene `--commit-review`, los tres tokens y la restricción a la puerta `tasks`
