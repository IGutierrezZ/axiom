# Delta para change-actor-identity

> Capacidad nueva: no existe spec viva previa. Todo lo que sigue es `ADDED`. Ninguna otra capacidad cambia: `sealed_by: inferred` de `sdd-lane-kickoff` y `sdd-preflight-configuration` se conserva tal cual (REQ-42.4). Todas las pruebas de estos requisitos aíslan git (`GIT_CONFIG_GLOBAL` apuntando a un fichero temporal y `GIT_CONFIG_NOSYSTEM=1`) y crean, cuando lo necesitan, un repositorio temporal con `git init` y `git config`: ningún requisito depende de la configuración git de la máquina.

## ADDED Requirements

### Requirement: Identidad Git como Actor por Defecto de `gate record` (REQ-42.1)

`axiom sdd gate record`, cuando se invoca sin `--actor` y existe identidad git completa (`user.name` y `user.email` no vacíos tras el saneado de REQ-42.7), DEBE registrar en el campo `actor` de `gates.yaml` la identidad con el formato `Nombre <email>`. El campo conserva su tipo `string`. El texto de ayuda de `--actor` DEBE indicar que, si se omite, se usa la identidad de git.

#### Scenario: Identidad completa registrada como actor
- **DADO** un workspace cuya configuración git define `user.name = Ana Pérez` y `user.email = ana@example.com`
- **CUANDO** se ejecuta `axiom sdd gate record` con compuerta y decisión válidas y sin `--actor`
- **ENTONCES** el registro añadido a `gates.yaml` tiene `actor: Ana Pérez <ana@example.com>`
- **Y** el comando termina con código de éxito
- **Y** no se emite ningún aviso por stderr

#### Scenario: La ayuda de `--actor` describe el valor por defecto
- **DADO** el verbo `axiom sdd gate record`
- **CUANDO** se muestra su ayuda
- **ENTONCES** el texto de `--actor` menciona la identidad de git como valor por defecto

### Requirement: `--actor` Explícito Prevalece sin Consultar Git (REQ-42.2)

Cuando `gate record` recibe `--actor` con un valor no vacío, el sistema DEBE registrar ese valor tal cual en `actor`, NO DEBE consultar la configuración de git y NO DEBE emitir el aviso de REQ-42.5, tenga o no el workspace identidad git.

#### Scenario: Actor explícito con identidad git presente
- **DADO** un workspace con identidad git completa
- **CUANDO** se ejecuta `gate record ... --actor maintainer`
- **ENTONCES** `gates.yaml` registra `actor: maintainer`
- **Y** no se emite aviso

#### Scenario: Actor explícito sin identidad git
- **DADO** un workspace sin `user.name` ni `user.email`
- **CUANDO** se ejecuta `gate record ... --actor release-manager`
- **ENTONCES** `gates.yaml` registra `actor: release-manager`
- **Y** no se emite aviso
- **Y** la resolución de identidad no se invoca (comprobable con un resolvedor sustituible que falla la prueba si se llama)

### Requirement: Sellador del Kickoff con la Identidad Git (REQ-42.3)

`axiom sdd kickoff seal`, sin `--infer` y con identidad git completa, DEBE registrar en `sealed_by` de `kickoff.yaml` la misma identidad con formato `Nombre <email>`. `seal` NO DEBE admitir `--actor` ni `--sealed-by`. El tipo de `Kickoff.SealedBy` no cambia.

#### Scenario: Sellado con identidad completa
- **DADO** un workspace con `user.name = Ana Pérez` y `user.email = ana@example.com`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x --execution-style continuous` sin `--infer`
- **ENTONCES** `kickoff.yaml` contiene `sealed_by: Ana Pérez <ana@example.com>`
- **Y** el comando termina con éxito y sin aviso

#### Scenario: Gate y sellado usan la misma identidad
- **DADO** un workspace con identidad git completa
- **CUANDO** se sella un cambio y después se registra una compuerta sin `--actor`
- **ENTONCES** `sealed_by` y `actor` son idénticos carácter a carácter

#### Scenario: `seal` no ofrece forma de indicar el sellador
- **DADO** el verbo `axiom sdd kickoff seal`
- **CUANDO** se invoca con `--actor x` o `--sealed-by x`
- **ENTONCES** el verbo lo rechaza como opción desconocida y no escribe `kickoff.yaml`

### Requirement: El Retro-sellado Conserva `inferred` (REQ-42.4)

`axiom sdd kickoff seal --infer` DEBE seguir escribiendo `sealed_by: inferred`, con independencia de la identidad git, y NO DEBE consultar git ni emitir el aviso de REQ-42.5. La exención de `inferred` en la validación de sellado (`ValidateForSeal`) NO DEBE cambiar.

#### Scenario: `--infer` con identidad git completa
- **DADO** un cambio con artefactos y sin `kickoff.yaml`, en un workspace con identidad git completa
- **CUANDO** se ejecuta `kickoff seal --infer`
- **ENTONCES** `kickoff.yaml` contiene `sealed_by: inferred`
- **Y** no contiene la identidad git

#### Scenario: `--infer` sin identidad git
- **DADO** el mismo cambio en un workspace sin identidad git
- **CUANDO** se ejecuta `kickoff seal --infer`
- **ENTONCES** `sealed_by` es `inferred`
- **Y** no se emite aviso

#### Scenario: La exención de `inferred` sigue vigente
- **DADO** un kickoff con `sealed_by: inferred` que `ValidateForSeal` aceptaba antes de este cambio
- **CUANDO** se valida con `ValidateForSeal`
- **ENTONCES** el resultado es el mismo que antes

### Requirement: Reserva `cli` con Aviso por stderr sin Bloqueo (REQ-42.5)

Cuando se necesita una identidad por defecto (`gate record` sin `--actor`, `kickoff seal` sin `--infer`) y no existe identidad git completa, es decir, falta `user.name`, falta `user.email` o faltan ambos, el sistema DEBE registrar el literal `cli`, DEBE emitir un único aviso por stderr con el prefijo `Aviso: ` que indique qué clave falta y cómo resolverlo (configurar git o, en `gate record`, indicar `--actor`) y NO DEBE bloquear: el comando termina con su código de salida de éxito y con su salida estándar normal. El aviso DEBE escribirse en un `io.Writer` propio, sustituible en pruebas, distinto de los avisos de rama de `seal` y de los escritores; NO DEBE aparecer en la salida estándar ni en el JSON.

#### Scenario: Sin identidad alguna
- **DADO** un workspace sin `user.name` ni `user.email`
- **CUANDO** se ejecuta `gate record` sin `--actor`
- **ENTONCES** `gates.yaml` registra `actor: cli`
- **Y** el writer de aviso recibe exactamente una línea que empieza por `Aviso: ` y nombra `user.name` y `user.email` como ausentes
- **Y** el comando termina con éxito

#### Scenario: Solo nombre
- **DADO** un workspace con `user.name` y sin `user.email`
- **CUANDO** se ejecuta `gate record` sin `--actor`
- **ENTONCES** se registra `actor: cli` y no el nombre solo
- **Y** el aviso nombra únicamente `user.email` como ausente

#### Scenario: Solo email
- **DADO** un workspace con `user.email` y sin `user.name`
- **CUANDO** se ejecuta `kickoff seal` sin `--infer`
- **ENTONCES** `sealed_by` es `cli`
- **Y** el aviso nombra únicamente `user.name` como ausente

#### Scenario: Un valor que queda vacío tras el saneado cuenta como ausente
- **DADO** `user.name` formado solo por espacios y `user.email` válido
- **CUANDO** se resuelve la identidad
- **ENTONCES** el resultado es «sin identidad completa» con `user.name` como ausente

#### Scenario: El aviso no altera stdout ni el código de salida
- **DADO** un workspace sin identidad git
- **CUANDO** se ejecuta `gate record ... --json` sin `--actor`
- **ENTONCES** stdout contiene el mismo JSON que con identidad completa, salvo los valores de `recorded_at` y resumen, y es JSON válido
- **Y** el código de salida es el de éxito
- **Y** el aviso no aparece en stdout

#### Scenario: Los avisos de rama existentes no cambian
- **DADO** una prueba existente que afirma cero avisos de rama en `seal` en un entorno sin identidad git
- **CUANDO** se ejecuta tras este cambio
- **ENTONCES** `sealBranchWarnings` sigue vacío, porque el aviso de identidad usa su propio writer

### Requirement: Resolución Solo desde la Configuración de Git en la Raíz del Workspace (REQ-42.6)

La identidad DEBE resolverse leyendo únicamente `user.name` y `user.email` de la configuración de git (local, global y condicional `includeIf` según el directorio), ejecutada desde la raíz del workspace donde se escribe el artefacto (`loc.WorkspaceRoot`; en `multirepo`, el repositorio de specs). NO DEBE leer variables de entorno `GIT_AUTHOR_*` ni `GIT_COMMITTER_*` ni usar `git var`. La resolución NUNCA DEBE fallar: git ausente, directorio que no es repositorio, clave vacía o error de ejecución se tratan como «sin identidad» y activan REQ-42.5.

#### Scenario: Configuración local del repositorio
- **DADO** un repositorio temporal con `user.name` y `user.email` locales y una configuración global aislada y vacía
- **CUANDO** se resuelve la identidad desde su raíz
- **ENTONCES** se obtienen los valores locales

#### Scenario: La configuración local prevalece sobre la global
- **DADO** una configuración global con una identidad A y una local con una identidad B
- **CUANDO** se resuelve la identidad
- **ENTONCES** se obtiene B

#### Scenario: Las variables `GIT_AUTHOR_*` se ignoran
- **DADO** un workspace sin identidad en la configuración de git y el entorno con `GIT_AUTHOR_NAME` y `GIT_AUTHOR_EMAIL` definidos
- **CUANDO** se resuelve la identidad
- **ENTONCES** el resultado es «sin identidad» y el comando registra `cli` con aviso

#### Scenario: Directorio que no es repositorio
- **DADO** un directorio temporal sin `.git` y con configuración global aislada que define identidad completa
- **CUANDO** se resuelve la identidad desde él
- **ENTONCES** la resolución no devuelve error y devuelve la identidad de la configuración global

#### Scenario: Git no disponible
- **DADO** un entorno sin el ejecutable `git` en el `PATH`
- **CUANDO** se resuelve la identidad
- **ENTONCES** la resolución no falla ni entra en pánico
- **Y** el resultado es «sin identidad»

#### Scenario: En multirepo se lee desde la raíz del workspace de specs
- **DADO** un `multirepo` cuyo repositorio de specs tiene una identidad local distinta de la del repositorio de código
- **CUANDO** se ejecuta `gate record` sin `--actor`
- **ENTONCES** el actor registrado es la identidad del repositorio de specs

### Requirement: Saneado y Formato de la Identidad (REQ-42.7)

La identidad DEBE formatearse como `Nombre <email>`. Antes de formatear, cada parte (`user.name`, `user.email`) DEBE sanearse en este orden: recorte de espacios en blanco en los extremos; rechazo de los caracteres de control (incluidos CR y LF) y del UTF-8 no válido, que dejan la parte como ausente; retirada de los caracteres `<` y `>`; nuevo recorte; y limitación a 120 caracteres (runes). Una parte que queda vacía tras el saneado cuenta como ausente. El actor resultante, por tanto, nunca supera 256 caracteres. El saneado y el formateo DEBEN ser funciones puras comprobables sin git. El valor de `--actor` explícito NO se sanea (REQ-42.2).

#### Scenario: Recorte de espacios
- **DADO** `user.name = "  Ana Pérez  "` y `user.email = "  ana@example.com "`
- **CUANDO** se formatea la identidad
- **ENTONCES** el resultado es `Ana Pérez <ana@example.com>`

#### Scenario: Caracteres de control
- **DADO** un `user.name` que contiene un salto de línea o un retorno de carro
- **CUANDO** se sanea
- **ENTONCES** la parte queda ausente y la identidad es incompleta (REQ-42.5)
- **Y** el actor registrado nunca contiene CR, LF ni otro carácter de control

#### Scenario: UTF-8 no válido
- **DADO** un `user.name` cuyos bytes no son UTF-8 válido
- **CUANDO** se sanea
- **ENTONCES** la parte queda ausente y la identidad es incompleta (REQ-42.5)
- **Y** ningún artefacto registra un valor binario (`!!binary`)

#### Scenario: Retirada de `<` y `>`
- **DADO** `user.name = "Ana <admin> Pérez"` y `user.email = "<ana@example.com>"`
- **CUANDO** se formatea la identidad
- **ENTONCES** el resultado es `Ana admin Pérez <ana@example.com>`
- **Y** el resultado contiene exactamente un `<` y un `>`, los que delimitan el email

#### Scenario: Tope de longitud
- **DADO** un `user.name` de 500 caracteres y un `user.email` de 500 caracteres
- **CUANDO** se formatea la identidad
- **ENTONCES** cada parte tiene como máximo 120 caracteres
- **Y** el resultado completo tiene como máximo 256 caracteres

#### Scenario: Caracteres especiales y no ASCII hacen round-trip por YAML
- **DADO** `user.name = "Íñigo: Gutiérrez # Zufia"` y `user.email = "inigo@example.com"`
- **CUANDO** se registra una compuerta y se vuelve a leer `gates.yaml`, y se sella y se vuelve a leer `kickoff.yaml`
- **ENTONCES** el `actor` y el `sealed_by` leídos son idénticos a la identidad formateada, sin pérdida ni truncado

### Requirement: Registros Anteriores Siguen Siendo Válidos sin Migración (REQ-42.8)

Los `gates.yaml` y `kickoff.yaml` existentes, con `actor` o `sealed_by` iguales a `cli`, `maintainer`, `release-manager`, `inferred` u otro literal, DEBEN seguir leyéndose y validándose igual, sin migración y sin reescritura. La validación NO DEBE comparar ni restringir el valor del actor. Los comentarios de tipo de `GateRecord.Actor` y `Kickoff.SealedBy` DEBEN documentar el nuevo valor por defecto y la reserva `cli`.

#### Scenario: Lectura de registros anteriores
- **DADO** un `gates.yaml` con registros de actor `cli`, `maintainer` y `release-manager`, y un `kickoff.yaml` con `sealed_by: cli` y otro con `sealed_by: inferred`
- **CUANDO** se cargan y validan con el binario nuevo
- **ENTONCES** todos se cargan y validan sin error y con los mismos valores

#### Scenario: Un registro nuevo convive con los anteriores
- **DADO** un `gates.yaml` con un registro de actor `cli`
- **CUANDO** se añade una compuerta con identidad git completa
- **ENTONCES** el fichero contiene el registro anterior intacto y el nuevo con `Nombre <email>`

#### Scenario: Un actor con formato de identidad no se rechaza
- **DADO** un `gates.yaml` cuyo actor es `Ana Pérez <ana@example.com>`
- **CUANDO** se valida el registro
- **ENTONCES** la validación pasa

### Requirement: La Forma de la Salida JSON No Cambia (REQ-42.9)

Los comandos `gate record --json` y `kickoff seal --json` NO DEBEN añadir, quitar ni renombrar campos en su JSON como consecuencia de este cambio. En particular, NO DEBE aparecer un campo `actor` ni `warnings` nuevos en la salida de `gate record` ni de `seal`; el aviso de identidad viaja solo por stderr.

#### Scenario: El conjunto de claves del JSON de `gate record` es el mismo
- **DADO** una ejecución de `gate record --json` con identidad completa y otra sin identidad
- **CUANDO** se comparan las claves del objeto JSON de ambas salidas con las de antes de este cambio
- **ENTONCES** el conjunto de claves es idéntico en los tres casos

#### Scenario: El conjunto de claves del JSON de `seal` es el mismo
- **DADO** una ejecución de `kickoff seal --json` con identidad completa y otra sin identidad
- **CUANDO** se comparan las claves del objeto JSON
- **ENTONCES** son las mismas que antes de este cambio y no incluyen `actor` ni `warnings`
