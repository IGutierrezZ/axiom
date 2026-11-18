# Delta para change-branch-lifecycle

> Parte del estado vigente: el resto de requisitos NO cambia. Esta delta añade la promoción de ramas en dos etapas, configurable por repositorio y por cambio (C-02-a): `git.main_branch` declara la rama principal, y el modo de promoción (`branch` o `release`) decide desde qué rama sale la rama del cambio y contra qué rama se confirma su fusión en `done`. Sin `main_branch` declarado, o con `main_branch` igual a la rama de integración, todo se comporta exactamente como hoy.
>
> Nota de integración (archive): las tablas de «Términos» y «Códigos tipados» de la spec viva no son requisitos y no las reescribe el archive por sí solo. Al integrar esta delta DEBEN actualizarse así: añadir en «Términos» las filas «Rama principal» (`git.main_branch`) y «Modo de promoción» (`branch` | `release` | `ask`), y en «Códigos tipados» ampliar `decision_required` (el modo de trabajo o el modo de promoción es `ask` y no hay respuesta) y `plan_conflict` (un modo de trabajo o de promoción distinto del materializado). La fila «Rama de integración» de «Términos» no cambia de semántica.

## ADDED Requirements

### Requirement: Modo de Promoción por Repositorio y por Cambio (REQ-25.46)

Cuando un repositorio tiene dos etapas (`git.main_branch` declarada y distinta de la rama de integración resuelta de ese repositorio), cada repositorio con rama de cambio DEBE seguir, para cada cambio, uno de dos modos de promoción:

- **`branch`**: la rama del cambio sale de la rama principal, entra en la rama de integración sin borrarse y después pasa a la rama principal, momento en que `done` la borra (REQ-25.49).
- **`release`**: la rama del cambio sale de la rama de integración, entra en ella y `done` la borra. El paso de la integración a la rama principal es la release, ajena al cambio.

El modo DEBE resolverse por repositorio, con esta precedencia de mayor a menor: (1) el modo registrado en el registro local del cambio (REQ-25.14); (2) el modo copiado en la instantánea del `kickoff.yaml` (REQ-25.36 de `sdd-lane-kickoff`); (3) la respuesta de la invocación, `--promotion` (REQ-25.47); (4) `promotion` de la entrada del repositorio en `repositories:`; (5) `promotion` del bloque `git:`; (6) el defecto `branch`. El valor `ask` en (4) o en (5) significa que la persona decide al crear la rama: si ningún origen de mayor precedencia fija el modo, `ensure` DEBE devolver `decision_required` (REQ-25.47).

Un repositorio sin dos etapas, es decir, sin `git.main_branch` o con `git.main_branch` igual a su rama de integración, NO tiene modo de promoción: `promotion` NO tiene efecto, no se pregunta, no se registra ni se copia, y el repositorio se comporta como hasta ahora. El repositorio de specs de un `multirepo` NO tiene modo de promoción porque no recibe ramas de cambio (REQ-25.5).

`ensure` DEBE persistir, en el registro local, el modo resuelto de cada repositorio con dos etapas y su origen (el token que indica de qué eslabón de la precedencia procede: respuesta de la persona, indicador de la invocación, ajuste del repositorio, ajuste de `git:`, defecto, instantánea o `legacy`). El registro DEBE ser la fuente operativa del modo en todos los carriles, incluido ODD, que no tiene instantánea. Si el registro no tiene el modo y la instantánea sí, `ensure` DEBE respetar el de la instantánea y registrarlo. El modo registrado NO DEBE perderse en una ejecución posterior de `ensure` ni de `--archive`.

El modo NO DEBE aplicarse con carácter retroactivo (origen `legacy`): (a) un registro que contiene la rama del cambio del repositorio y no contiene modo, o (b) una instantánea sellada sin `main_branch` (REQ-25.36 de `sdd-lane-kickoff`) cuya rama del cambio ya existe en local, DEBEN resolverse como `release` con el origen `legacy`, es decir, como de una sola etapa a efectos de `done` (el destino es la rama de integración); `ensure` DEBE registrarlo y NO DEBE moverse ninguna rama. Un repositorio cuya rama del cambio aún no existe toma el modo que asigne la política vigente.

#### Scenario: Sin `main_branch` nada cambia
- **DADO** un `axiom.yaml` con `git.integration_branch: develop`, sin `git.main_branch` y con `git.promotion: ask`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** la rama `feat/x` sale de `origin/develop` como hasta ahora
- **Y** no se devuelve `decision_required` por el modo de promoción
- **Y** el registro local no contiene modo de promoción para el repositorio

#### Scenario: `main_branch` igual a la integración equivale a una sola etapa
- **DADO** un `axiom.yaml` con `git.integration_branch: main`, `git.main_branch: main` y `git.promotion: release`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** la rama sale de `origin/main`
- **Y** el repositorio no tiene modo de promoción registrado

#### Scenario: Con `main_branch` y sin `promotion` el modo es `branch`
- **DADO** un `axiom.yaml` con `git.main_branch: master`, `git.integration_branch: develop` y ninguna clave `promotion`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el modo resuelto del repositorio es `branch`
- **Y** el registro local lo guarda con el origen `default`

#### Scenario: El registro prevalece sobre la instantánea
- **DADO** un cambio con el modo `release` registrado y la instantánea del `kickoff.yaml` con el modo `branch`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el modo resuelto es `release`

#### Scenario: La instantánea prevalece sobre `--promotion` y sobre la política
- **DADO** un cambio sin modo registrado, con la instantánea del `kickoff.yaml` con el modo `branch`, y un `axiom.yaml` con `git.promotion: release`
- **CUANDO** se ejecuta `ensure --change x` sin `--promotion`
- **ENTONCES** el modo resuelto es `branch`
- **Y** el registro lo guarda con el origen de instantánea

#### Scenario: `--promotion` prevalece sobre `repositories[].promotion` y `git.promotion`
- **DADO** un `axiom.yaml` con `git.promotion: release` y, para `repo-backend`, `promotion: release`, y un cambio sin modo registrado ni instantánea
- **CUANDO** se ejecuta `ensure --change x --promotion branch`
- **ENTONCES** el modo resuelto de `repo-backend` es `branch`
- **Y** el registro lo guarda con el origen `flag`

#### Scenario: `repositories[].promotion` prevalece sobre `git.promotion`
- **DADO** un `axiom.yaml` con `git.promotion: branch` y, para `repo-backend`, `promotion: release`
- **CUANDO** se resuelve el modo de `repo-backend` y de `repo-frontend` sin registro, instantánea ni `--promotion`
- **ENTONCES** `repo-backend` resuelve `release` y `repo-frontend` resuelve `branch`

#### Scenario: `git.promotion` prevalece sobre el defecto
- **DADO** un `axiom.yaml` con `git.main_branch: master` y `git.promotion: release`
- **CUANDO** se resuelve el modo sin registro, instantánea, `--promotion` ni entrada de repositorio
- **ENTONCES** el modo resuelto es `release`

#### Scenario: El modo registrado sobrevive a una ejecución posterior
- **DADO** un cambio con el modo `release` registrado y la política posterior cambiada a `git.promotion: branch`
- **CUANDO** se ejecuta de nuevo `ensure --change x`
- **ENTONCES** el modo sigue siendo `release`
- **Y** el JSON es idéntico al de la ejecución anterior (REQ-25.7)

#### Scenario: Con el registro borrado se respeta la instantánea
- **DADO** un cambio sellado con la instantánea con el modo `branch` y su registro local borrado
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el modo resuelto es `branch` sin preguntar
- **Y** el registro se reconstruye con ese modo

#### Scenario: Con el registro borrado y sin instantánea `ask` vuelve a preguntar
- **DADO** un carril ODD con `git.promotion: ask`, la rama del cambio ya creada y el registro local borrado
- **CUANDO** se ejecuta `ensure --lane odd --change x`
- **ENTONCES** se devuelve `decision_required` por el modo de promoción
- **Y** la rama existente no se mueve

#### Scenario: Un registro con la rama y sin modo se resuelve como `release` legacy
- **DADO** un cambio registrado antes de esta capacidad, con la rama del cambio en el registro y sin modo de promoción, y un `axiom.yaml` que ahora declara `git.main_branch: master` y `git.integration_branch: develop`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** la rama existente se reutiliza sin moverla (REQ-25.7)
- **Y** el modo resuelto es `release` con el origen `legacy` y se registra
- **Y** a efectos de `done` el destino es la rama de integración

#### Scenario: Una instantánea sin `main_branch` y con la rama ya creada se resuelve como `release` legacy
- **DADO** un cambio sellado antes de esta capacidad (instantánea sin `main_branch`), sin registro y con la rama del cambio ya existente en local, y un `axiom.yaml` que ahora declara dos etapas
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el modo resuelto es `release` con el origen `legacy`
- **Y** `kickoff.yaml` no se modifica y la rama no se mueve

#### Scenario: Una instantánea sin `main_branch` y sin la rama creada toma la política vigente
- **DADO** un cambio sellado sin `main_branch` en la instantánea, sin registro y sin la rama del cambio en local, con `git.promotion: branch`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el modo resuelto es `branch` con el origen `git` y la rama sale de la rama principal

#### Scenario: En multirepo cada repositorio de código resuelve su propio modo
- **DADO** un multirepo con `git.main_branch: master`, `git.integration_branch: develop`, `git.promotion: branch` y, para `repo-frontend`, `promotion: release`
- **CUANDO** se ejecuta `ensure --change x` con los roles de ambos repositorios sellados
- **ENTONCES** `repo-backend` resuelve `branch` y `repo-frontend` resuelve `release`
- **Y** el registro local guarda un modo por repositorio

#### Scenario: El repositorio de specs de un multirepo no tiene modo
- **DADO** un multirepo con `git.main_branch: master` y `git.promotion: ask`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el repositorio de specs se trabaja in situ en su rama de integración sin modo de promoción
- **Y** no se pregunta por él

---

### Requirement: `ensure`: Base Efectiva, `ask` y `--promotion` (REQ-25.47)

La base de la rama del cambio de cada repositorio con dos etapas DEBE ser la **base efectiva** del modo resuelto: la rama principal en `branch` y la rama de integración en `release`. La resolución de la base sigue REQ-25.10 (`origin/<base efectiva>` si existe en local y, si no, la rama local), y un `--base` explícito (REQ-25.12) la sustituye. Si no existe la base efectiva, ni local ni como referencia remota local, `ensure` DEBE rechazar con `base_not_found` nombrando esa rama. Un repositorio sin dos etapas conserva como base la rama de integración. La rama de archivado (REQ-25.45) y las ramas de rol (REQ-25.4) NO cambian de base: la de archivado sale de la rama de integración y las de rol, de su rama de seguimiento.

`ensure` DEBE aceptar el indicador `--promotion <modo>` y `--promotion <repositorio>=<modo>`, este último repetible, donde `<modo>` es `branch` o `release`. La forma sin repositorio se aplica a todos los repositorios del plan con dos etapas; la forma con repositorio, solo al nombrado. `ask` NO es una respuesta válida de `--promotion`. Un indicador presente pero vacío o un `<modo>` inválido DEBEN rechazarse como uso inválido de la orden sin crear nada. Un `--promotion <repositorio>=<modo>` cuyo `<repositorio>` no nombre ningún repositorio del plan DEBE rechazarse con el código tipado `invalid_name`, sin crear nada, nombrando el repositorio pedido y la lista de nombres válidos del plan. Un `--promotion` que alcance un repositorio sin dos etapas NO tiene efecto y DEBE avisar sin bloquear. Dos indicadores con valores distintos para el mismo repositorio DEBEN rechazarse como uso inválido.

Cuando el modo de un repositorio con dos etapas resuelva a `ask` sin respuesta, `ensure` DEBE devolver el resultado tipado `decision_required`, sin crear nada ni interactuar, con el mismo esquema y mecanismo que el de REQ-25.8: una pregunta, las respuestas `branch` y `release`, cada una con su etiqueta y su efecto en castellano y la invocación exacta que repite la orden con la respuesta. La pregunta DEBE ser de **un repositorio por llamada**, en el orden del plan, nombrando el repositorio en el detalle del resultado; con varios repositorios pendientes las preguntas son secuenciales. Cada invocación propuesta DEBE conservar las respuestas ya dadas en la orden (`--mode`, `--prefix`, `--branch`, `--base`, `--repo` y los `--promotion` previos). Cuando también falta el modo de trabajo (REQ-25.8), esa pregunta va primero y las de promoción después. La respuesta DEBE registrarse por cambio y repositorio en el registro local (REQ-25.14) y las ejecuciones posteriores del cambio, incluida la de `--archive`, DEBEN reutilizarla sin volver a preguntar.

Un modo pedido (por `--promotion`) distinto del registrado, o distinto del copiado en la instantánea cuando no hay registro, DEBE rechazarse con `plan_conflict`, sin crear nada y sin rebasar ni mover la rama: la rama ya sale de una base concreta. Un modo pedido igual al registrado es un no-op. Con la rama ya creada y sin registro ni instantánea, el modo pedido se acepta como declaración, se registra y NO mueve la rama.

#### Scenario: En modo `branch` la rama sale de la rama principal
- **DADO** un repositorio con `git.main_branch: master`, `git.integration_branch: develop`, `git.promotion: branch` y las referencias locales `origin/master` y `origin/develop`
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** la rama `feat/x` se crea desde `origin/master`
- **Y** `base.ref` es `origin/master`
- **Y** el destino propuesto del PR de `feat/x` es `develop`

#### Scenario: En modo `release` la rama sale de la integración
- **DADO** el mismo repositorio con `git.promotion: release`
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** la rama `feat/x` se crea desde `origin/develop`
- **Y** `base.ref` es `origin/develop`

#### Scenario: Sin referencia remota la base efectiva es la rama principal local
- **DADO** un repositorio sin remoto, con `master` y `develop` locales y modo `branch`
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** `base.ref` es `master` y `base.source` indica la rama local

#### Scenario: Base efectiva inexistente
- **DADO** un repositorio con `develop` pero sin `master` ni `origin/master`, y modo `branch`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `base_not_found` nombrando `master`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: `--base` prevalece sobre la base efectiva
- **DADO** el repositorio en modo `branch` y la invocación `ensure --change x --base origin/develop`
- **CUANDO** se ejecuta
- **ENTONCES** la rama se crea desde `origin/develop`

#### Scenario: `ask` devuelve `decision_required` con `branch` y `release`
- **DADO** un repositorio con `git.main_branch: master`, `git.integration_branch: develop`, `git.promotion: ask` y sin respuesta registrada
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** el estado es `decision_required` y el código de salida es distinto de cero
- **Y** las respuestas ofrecidas son `branch` y `release`, cada una con etiqueta, efecto e invocación
- **Y** el detalle nombra el repositorio
- **Y** no se crea ninguna rama, worktree ni registro

#### Scenario: Repetir con `--promotion` crea la rama en la base correcta
- **DADO** el resultado `decision_required` del escenario anterior
- **CUANDO** se ejecuta `ensure --change x --promotion release`
- **ENTONCES** la rama `feat/x` se crea desde `origin/develop`
- **Y** el modo `release` queda registrado con el origen de respuesta
- **Y** una ejecución posterior de `ensure --change x` sin `--promotion` no vuelve a preguntar

#### Scenario: `--promotion branch` crea la rama desde la rama principal
- **DADO** el mismo repositorio con `ask` sin respuesta
- **CUANDO** se ejecuta `ensure --change x --promotion branch`
- **ENTONCES** la rama `feat/x` se crea desde `origin/master`

#### Scenario: Un modo distinto del registrado se rechaza con `plan_conflict`
- **DADO** un cambio con la rama ya creada y el modo `branch` registrado
- **CUANDO** se ejecuta `ensure --change x --promotion release`
- **ENTONCES** se rechaza con `plan_conflict`
- **Y** la rama no se mueve y el registro no cambia

#### Scenario: El mismo modo que el registrado es un no-op
- **DADO** un cambio con la rama creada y el modo `branch` registrado
- **CUANDO** se ejecuta `ensure --change x --promotion branch`
- **ENTONCES** el resultado es idéntico al de la ejecución anterior (REQ-25.7)

#### Scenario: Un modo distinto del de la instantánea se rechaza
- **DADO** un cambio sin modo registrado, con la instantánea con el modo `branch`
- **CUANDO** se ejecuta `ensure --change x --promotion release`
- **ENTONCES** se rechaza con `plan_conflict`

#### Scenario: `ask` no es una respuesta válida
- **DADO** un repositorio con dos etapas
- **CUANDO** se ejecuta `ensure --change x --promotion ask`
- **ENTONCES** se rechaza como uso inválido de la orden
- **Y** no se crea ninguna rama ni worktree

#### Scenario: Un `--promotion` vacío se rechaza
- **DADO** un repositorio con dos etapas
- **CUANDO** se ejecuta `ensure --change x --promotion ""`
- **ENTONCES** se rechaza como uso inválido de la orden y no se crea nada

#### Scenario: En multirepo las preguntas son secuenciales y conservan las respuestas
- **DADO** un multirepo con `git.promotion: ask`, los repositorios de código `repo-backend` y `repo-frontend` con roles sellados, el modo de trabajo del hub en `ask` y ninguna respuesta
- **CUANDO** se ejecuta `ensure --change x` y se responde cada pregunta con la invocación propuesta
- **ENTONCES** primero se pregunta por el modo de trabajo y después por `repo-backend` y por `repo-frontend`, una pregunta por llamada
- **Y** la invocación propuesta en cada pregunta repite las respuestas ya dadas
- **Y** tras la última respuesta se crean las ramas de ambos repositorios, cada una en la base de su modo

#### Scenario: `--promotion <repositorio>=<modo>` fija modos distintos por repositorio
- **DADO** un multirepo con dos repositorios de código con dos etapas
- **CUANDO** se ejecuta `ensure --change x --promotion repo-backend=branch --promotion repo-frontend=release`
- **ENTONCES** `repo-backend` sale de su rama principal y `repo-frontend` de su rama de integración

#### Scenario: Un repositorio inexistente en el plan se rechaza
- **DADO** un multirepo sin el repositorio `repo-otro`
- **CUANDO** se ejecuta `ensure --change x --promotion repo-otro=branch`
- **ENTONCES** se rechaza con `invalid_name` nombrando `repo-otro` y listando los nombres válidos del plan
- **Y** no se crea ninguna rama ni worktree

#### Scenario: `--promotion` sobre un repositorio de una sola etapa avisa y no actúa
- **DADO** un repositorio sin `git.main_branch`
- **CUANDO** se ejecuta `ensure --change x --promotion release`
- **ENTONCES** la rama sale de la integración como hasta ahora
- **Y** el resultado avisa de que `--promotion` no tiene efecto, sin bloquear

#### Scenario: La rama de archivado y las de rol no cambian de base
- **DADO** un repositorio en modo `branch` con la rama `feat/x` fusionada en la integración
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** `feat/x-archive` sale de la rama de integración, no de la principal

---

### Requirement: `status`: Siguiente PR hacia la Rama Principal en Modo `branch` (REQ-25.48)

En un repositorio con dos etapas y modo `branch`, `axiom branch status` DEBE informar de la **siguiente PR** de la rama del cambio. Mientras la rama del cambio no esté fusionada en la rama de integración, el destino propuesto del PR (`prTarget`) DEBE ser la rama de integración. Cuando git, solo con referencias locales y sin ninguna operación remota, confirme que la rama del cambio está fusionada en la rama de integración, por ascendencia o por árbol equivalente (REQ-25.16, capas 1 y 2), `status` DEBE presentar la siguiente PR con la rama del cambio como cabeza y la rama principal como base, y el destino propuesto pasa a ser la rama principal. Si la comprobación no concluye (por ejemplo, la versión de git no admite la capa de árbol equivalente y la rama no es ascendiente), `status` NO DEBE fingir certeza: DEBE mantener la rama de integración como destino y avisar de que no pudo confirmar la fusión.

Con modo `release` o con una sola etapa, el destino propuesto es siempre la rama de integración y no hay siguiente PR. `status` DEBE exponer, además, de forma aditiva por repositorio, el modo de promoción resuelto y la rama principal; los campos nuevos NO DEBEN alterar los ya existentes. Si el modo del repositorio es `ask` sin respuesta (sin registro ni instantánea), `status` NO DEBE calcular la etapa: DEBE avisar de que el modo está pendiente de decisión y de que `ensure` lo preguntará, sin fallar y sin inventar un modo.

#### Scenario: Antes de fusionar en la integración el destino es la integración
- **DADO** un repositorio con `main_branch: master`, `integration_branch: develop`, modo `branch` y la rama `feat/x` sin fusionar en `develop`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el destino propuesto del PR de `feat/x` es `develop`
- **Y** no hay siguiente PR
- **Y** el JSON informa del modo `branch` y de la rama principal `master`

#### Scenario: Tras fusionar con merge commit en la integración aparece la siguiente PR
- **DADO** el mismo repositorio con `feat/x` fusionada en `develop` con un merge commit
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** la siguiente PR tiene como cabeza `feat/x` y como base `master`
- **Y** el destino propuesto es `master`
- **Y** no consta ninguna operación remota

#### Scenario: Tras fusionar con squash se detecta por árbol equivalente
- **DADO** `feat/x` integrada en `develop` mediante un squash y `develop` con commits posteriores
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** la siguiente PR tiene como base `master`

#### Scenario: Si la comprobación no concluye se mantiene la integración con un aviso
- **DADO** un git sin soporte de `git merge-tree --write-tree` y `feat/x` integrada en `develop` con un squash
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el destino propuesto sigue siendo `develop`
- **Y** el resultado avisa de que no se pudo confirmar la fusión en la integración
- **Y** el código de salida es `0`

#### Scenario: En modo `release` no hay siguiente PR
- **DADO** un repositorio con dos etapas, modo `release` y `feat/x` fusionada en `develop`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el destino propuesto es `develop` y no hay siguiente PR

#### Scenario: Sin `main_branch` el resultado no cambia
- **DADO** un repositorio sin `git.main_branch` y `feat/x` fusionada en la integración
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el destino propuesto es la integración, no hay siguiente PR y los campos existentes del JSON no cambian

#### Scenario: Modo `ask` sin respuesta se informa como pendiente
- **DADO** un cambio con la rama creada, `git.promotion: ask`, sin registro ni instantánea
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON no incluye siguiente PR ni etapa
- **Y** el resultado avisa de que el modo está pendiente y de que `ensure` lo preguntará
- **Y** el código de salida es `0`

---

### Requirement: `done`: Destino de la Confirmación de Fusión según el Modo de Promoción (REQ-25.49)

`done` DEBE confirmar la fusión de la rama del cambio contra el destino que corresponde a su modo: en un repositorio con dos etapas y modo `branch`, la **rama principal**; en modo `release` o con una sola etapa, la **rama de integración**, como hasta ahora. Las ramas de rol se siguen confirmando contra su rama de seguimiento y la rama de archivado contra la rama de integración, en cualquier modo (REQ-25.45).

En modo `branch`, la confirmación contra la rama principal DEBE usar solo las capas de ascendencia (`ancestry`) y de árbol equivalente (`equivalent_tree`) de REQ-25.16, o `--force`. La evidencia `pr_merged` NO acredita la fusión en la rama principal, porque el `gates.yaml` identifica el PR del cambio y no su destino: en modo `branch`, la capa `pr_merged` NO se usa. En la rama de integración y en modo `release`, las capas siguen como en REQ-25.16.

El rechazo `merge_not_confirmed` DEBE nombrar el destino real contra el que se evaluó. En modo `branch`, una rama fusionada solo en la rama de integración NO DEBE borrarse. La tolerancia a squash se mantiene: una rama integrada con squash en la rama principal se confirma por `equivalent_tree` o, si esa capa no concluye, con `--force`. Con el modo `ask` pendiente (sin registro ni instantánea), `done` DEBE confirmar contra la rama principal, el destino más exigente, porque borrar una rama es irreversible; `--force` sigue siendo la salida. El repositorio de specs de un `multirepo` se omite porque no tiene ramas de cambio (REQ-25.5). El resto del comportamiento de `done` (REQ-25.17) no cambia.

#### Scenario: En modo `branch` una rama fusionada solo en la integración no se borra
- **DADO** un repositorio con `main_branch: master`, `integration_branch: develop`, modo `branch` y `feat/x` fusionada en `develop` pero no en `master`
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `merge_not_confirmed` nombrando `master` como destino
- **Y** el worktree y la rama local `feat/x` permanecen intactos

#### Scenario: En modo `branch` una fusión en la rama principal con merge commit se confirma por ascendencia
- **DADO** `feat/x` fusionada en `master` con un merge commit
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama se confirma por el token `ancestry`
- **Y** se eliminan el worktree y la rama local

#### Scenario: En modo `branch` un squash en la rama principal se confirma por árbol equivalente
- **DADO** `feat/x` integrada en `develop` y después en `master` mediante squashes, y `master` con commits posteriores
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama se confirma por el token `equivalent_tree`
- **Y** se eliminan el worktree y la rama local

#### Scenario: En modo `branch` un squash que no concluye exige `--force`
- **DADO** un git sin soporte de `git merge-tree --write-tree` y `feat/x` integrada en `master` con un squash
- **CUANDO** se ejecuta `axiom branch done --change x` y después `axiom branch done --change x --force`
- **ENTONCES** la primera se rechaza con `merge_not_confirmed` nombrando `master`
- **Y** la segunda elimina la rama con la capa `forced` e informa de la punta (SHA)

#### Scenario: En modo `branch` la evidencia `pr_merged` no confirma la fusión en la rama principal
- **DADO** `feat/x` fusionada solo en `develop`, sin confirmación por `equivalent_tree` en `master` y un `gates.yaml` con evidencia `pr_merged` del PR del cambio
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `merge_not_confirmed` nombrando `master`

#### Scenario: En modo `release` la rama se borra al fusionarla en la integración
- **DADO** un repositorio con `main_branch: master`, `integration_branch: develop`, modo `release` y `feat/x` fusionada en `develop`
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama se confirma contra `develop` y se eliminan el worktree y la rama local

#### Scenario: Sin `main_branch` el destino sigue siendo la integración
- **DADO** un repositorio sin `git.main_branch` y `feat/x` fusionada en la integración
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama se confirma y se elimina como hasta ahora, incluida la capa `pr_merged`

#### Scenario: En multirepo `done` omite el repositorio de specs y evalúa cada repositorio con su modo
- **DADO** un multirepo con `repo-backend` en modo `branch` fusionado en su rama principal y `repo-frontend` en modo `release` fusionado en su integración
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** el repositorio de specs se omite sin rama ni worktree que eliminar
- **Y** se eliminan las ramas de `repo-backend` y de `repo-frontend`

#### Scenario: Con `ask` pendiente `done` confirma contra la rama principal
- **DADO** un repositorio con `main_branch: master`, `integration_branch: develop`, `git.promotion: ask`, sin registro ni instantánea, y `feat/x` fusionada solo en `develop`
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `merge_not_confirmed` nombrando `master`
- **Y** la rama se conserva
- **Y** con `--force` se elimina con la capa `forced` e informa de la punta (SHA)

#### Scenario: Las ramas de rol y de archivado conservan su destino
- **DADO** un repositorio en modo `branch` con `feat/x-core` fusionada en `feat/x` y `feat/x-archive` fusionada en la integración
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** `feat/x-core` se confirma contra `feat/x` y `feat/x-archive` contra la rama de integración

---

### Requirement: Contratos de Integración y Publicación que No Cambian con el Modo (REQ-25.50)

El modo de promoción NO DEBE alterar la semántica de la rama de integración ni los destinos de otros verbos, que siguen refiriéndose a la **primera etapa**:

- `IntegrationBranch` conserva su significado: la rama a la que se dirige el PR de la rama del cambio mientras no esté fusionada en ella (REQ-25.3).
- El destino de publicación de `axiom sdd publish` (`PublicationTarget`) y el del relevo (`HandOff`) siguen siendo los del plan estático, es decir, la rama de integración, y NO adoptan la siguiente PR de `status` (REQ-25.48).
- La rama de archivado (REQ-25.45) sale de la rama de integración y `ensure --archive` solo espera a que el cambio esté fusionado en ella, nunca en la rama principal.
- Los repositorios de specs de `multirepo` y las ramas de rol no cambian.

#### Scenario: `IntegrationBranch` no cambia de significado
- **DADO** un repositorio con `main_branch: master`, `integration_branch: develop` y modo `branch`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** `integrationBranch` del repositorio es `develop`

#### Scenario: La publicación y el relevo siguen apuntando a la integración
- **DADO** un repositorio en modo `branch` con `feat/x` ya fusionada en `develop`, de modo que `status` presenta la siguiente PR hacia `master`
- **CUANDO** se calculan el destino de `axiom sdd publish` y el del relevo
- **ENTONCES** ambos siguen siendo los de la primera etapa, calculados con la rama de integración

#### Scenario: El archivado sigue esperando solo a la integración
- **DADO** un repositorio en modo `branch` con el cambio `x` fusionado en `develop` y aún no en `master`
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se crea `feat/x-archive` desde `develop` sin devolver `change_not_integrated`

#### Scenario: El archivado se rechaza si el cambio no está en la integración aunque esté en la rama principal
- **DADO** un repositorio en modo `branch` cuya base `develop` no contiene `openspec/changes/x/`
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se rechaza con `change_not_integrated` nombrando `develop`

## MODIFIED Requirements

### Requirement: Resolución de la Política de Ramas (REQ-25.3)

La política de cada repositorio DEBE resolverse con esta precedencia, de mayor a menor: (1) las excepciones por cambio de `ensure` (`--prefix`, `--branch`, `--base`, REQ-25.12); (2) la entrada del repositorio en `repositories:` de `axiom.yaml`, si existe; (3) el bloque `git:` de `axiom.yaml`; (4) los valores por defecto: rama de integración `main` y prefijo `feat`. Sin bloque `git:`, `integration_branch` DEBE valer `main` y `branch_prefix` DEBE valer `feat`; sin `repositories:`, ningún repositorio tiene ajustes propios y todos heredan los valores del workspace. La política incorpora además la rama principal (`git.main_branch`, solo en `git:` y sin valor por repositorio ni por defecto: su ausencia significa una sola etapa) y el modo de promoción por defecto (`promotion`, con herencia `repositories:` > `git:`; el valor vacío hereda); el modo efectivo de cada repositorio y cambio se resuelve en REQ-25.46. Los nombres de rama resultantes DEBEN ser nombres de referencia git válidos: un prefijo o nombre inválido indicado en la invocación (`--prefix`, `--branch`, `--change`) DEBE rechazarse con `invalid_name`, y un valor inválido o una configuración ilegible en `axiom.yaml` o en el hub personal con `config_invalid`, en ambos casos sin crear nada.
(Previously: la política no incluía la rama principal ni el modo de promoción)

#### Scenario: Sin configuración se aplican los valores por defecto
- **DADO** un `axiom.yaml` sin bloque `git:` ni `repositories:`
- **CUANDO** se planifica el cambio `x`
- **ENTONCES** la rama de integración es `main` y la rama del cambio es `feat/x`

#### Scenario: El bloque `git:` sustituye los valores por defecto
- **DADO** un `axiom.yaml` con `git.integration_branch: develop` y `git.branch_prefix: inc`
- **CUANDO** se planifica el cambio `x`
- **ENTONCES** la rama de integración es `develop` y la rama del cambio es `inc/x`

#### Scenario: Un repositorio con ajustes propios prevalece sobre `git:`
- **DADO** un `axiom.yaml` con `git.integration_branch: main` y una entrada de `repositories:` para `repo-backend` con rama de integración `release`
- **CUANDO** se planifica un cambio que toca `repo-backend` y `repo-frontend`
- **ENTONCES** la base de `repo-backend` es `release` y la de `repo-frontend` es `main`

#### Scenario: La excepción por cambio prevalece sobre todo lo demás
- **DADO** la política anterior y la invocación `ensure --change x --prefix fix`
- **CUANDO** se planifica el cambio
- **ENTONCES** la rama del cambio es `fix/x` en todos los repositorios del plan

#### Scenario: Un prefijo inválido de la invocación se rechaza sin efectos
- **DADO** un `ensure` con `--prefix "mi prefijo//"`
- **CUANDO** se ejecuta
- **ENTONCES** se rechaza con el código tipado `invalid_name`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: Un prefijo inválido de `axiom.yaml` se rechaza como configuración
- **DADO** un `axiom.yaml` con `git.branch_prefix: "inc//"`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con el código tipado `config_invalid`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: La política hereda la rama principal y el modo de promoción
- **DADO** un `axiom.yaml` con `git.main_branch: master`, `git.promotion: branch` y una entrada de `repositories:` para `repo-backend` con `promotion: release`
- **CUANDO** se resuelve la política de `repo-backend` y de `repo-frontend`
- **ENTONCES** ambos tienen la rama principal `master`
- **Y** el modo por defecto de `repo-backend` es `release` y el de `repo-frontend` es `branch`

#### Scenario: Sin `main_branch` la política no inventa una rama principal
- **DADO** un `axiom.yaml` con `git.integration_branch: develop` y sin `git.main_branch`
- **CUANDO** se resuelve la política
- **ENTONCES** la rama principal queda vacía y el repositorio es de una sola etapa

---

### Requirement: `ensure`: Modo de Trabajo `worktree`, `in_place` y `ask` (REQ-25.8)

`ensure` DEBE leer el modo de trabajo del hub personal (`~/.axiom/workspaces.json`) y NUNCA de `axiom.yaml`. Sin preferencias guardadas para el proyecto, estén o no registrado, el modo DEBE ser `worktree`.

- `worktree`: DEBE crear la rama desde la base y un worktree en la raíz de worktrees (REQ-25.11). NUNCA DEBE modificar el HEAD, el índice ni el árbol de trabajo del checkout principal.
- `in_place`: DEBE crear (o seleccionar) la rama y hacer checkout de ella en el propio repositorio, sujeto a REQ-25.9.
- `ask`: DEBE devolver el resultado tipado `decision_required` sin crear nada, sin leer la entrada estándar y sin interactuar; quien invoca (skill o dashboard) presenta la pregunta y repite la orden con la respuesta (`--mode worktree|in_place`, supuesto S-2). La respuesta DEBE registrarse por cambio en el registro local (REQ-25.14) y las ejecuciones posteriores para ese mismo cambio, incluida la de `--archive`, DEBEN reutilizarla sin volver a preguntar. Una respuesta registrada NO modifica la preferencia personal del hub.

Un `--mode` distinto del ya materializado para el cambio DEBE rechazarse con `plan_conflict`. `decision_required` y `plan_conflict` NO son exclusivos del modo de trabajo: el modo de promoción (REQ-25.46 y REQ-25.47) usa el mismo mecanismo y el mismo esquema de pregunta, con una pregunta distinta por repositorio, y cuando faltan ambas respuestas la del modo de trabajo se pregunta primero.
(Previously: `decision_required` y `plan_conflict` se describían solo para el modo de trabajo)

#### Scenario: Sin preferencias el modo es `worktree`
- **DADO** un proyecto sin preferencias guardadas en el hub
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se crea la rama y un worktree en la raíz de worktrees por defecto

#### Scenario: El checkout principal no se toca en modo `worktree`
- **DADO** el checkout principal desplegado en `main` con un fichero modificado sin confirmar
- **CUANDO** se ejecuta `ensure --change x` en modo `worktree`
- **ENTONCES** el HEAD, el índice y el árbol de trabajo del checkout principal son idénticos a los de antes
- **Y** el checkout principal sigue en `main`

#### Scenario: Modo `in_place` hace checkout en el propio repositorio
- **DADO** un proyecto en modo `in_place` con el árbol limpio
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el repositorio queda desplegado en `feat/x`
- **Y** no se crea ningún worktree

#### Scenario: Modo `ask` devuelve `decision_required` sin crear nada
- **DADO** un proyecto en modo `ask` y sin respuesta registrada para el cambio
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** el estado es `decision_required` y el código de salida es distinto de cero
- **Y** no se crea ninguna rama, worktree ni registro
- **Y** el comando no espera entrada alguna

#### Scenario: La respuesta a `ask` se registra por cambio
- **DADO** un proyecto en modo `ask`
- **CUANDO** se ejecuta `ensure --change x --mode worktree` y después `ensure --change x` sin `--mode`
- **ENTONCES** la segunda ejecución no devuelve `decision_required` y reutiliza `worktree`
- **Y** para el cambio `y`, sin respuesta registrada, `ensure --change y` devuelve `decision_required`
- **Y** la preferencia personal del hub sigue siendo `ask`

#### Scenario: La rama de archivado reutiliza la respuesta registrada
- **DADO** un proyecto en modo `ask` y el cambio `x` con la respuesta `worktree` registrada y su PR fusionado
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** no devuelve `decision_required` y crea el worktree de la rama de archivado

#### Scenario: Un modo distinto del materializado se rechaza
- **DADO** un cambio materializado como `worktree`
- **CUANDO** se ejecuta `ensure --change x --mode in_place`
- **ENTONCES** se rechaza con `plan_conflict` y no se modifica nada

#### Scenario: Con ambas decisiones pendientes se pregunta primero el modo de trabajo
- **DADO** un proyecto con el modo de trabajo `ask` en el hub y `git.promotion: ask` con dos etapas, sin respuestas
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** el `decision_required` pregunta por el modo de trabajo y ofrece `worktree` e `in_place`
- **Y** tras responderlo, la siguiente ejecución pregunta por el modo de promoción con `branch` y `release`

---

### Requirement: Base sin Red y `--fetch` Explícito (REQ-25.10)

`ensure` NO DEBE realizar ninguna operación remota por defecto. La base de cada rama DEBE ser `origin/<base efectiva>` si esa referencia existe en local y, si no, la rama local de la base efectiva, donde la base efectiva es la rama de integración (en los repositorios de una sola etapa y en modo `release`) o la rama principal (en los repositorios con dos etapas en modo `branch`, REQ-25.47); si la propia rama del cambio no existe en local pero existe su referencia remota local, la rama se adopta desde ella (REQ-27.15). El JSON DEBE devolver, para cada repositorio, `base.ref`, `base.sha`, `base.source` (que indica si la base procede de la referencia remota local o de la rama local; la rama del cambio adoptada desde el remoto se indica aparte, en `branches[].createdFrom: remote` y `createdFromRef`, REQ-27.15) y `fetched: false`. Solo con `--fetch` el comando PUEDE ejecutar un `fetch` previo, y entonces `fetched` DEBE ser `true`. Si `--fetch` falla, `ensure` DEBE terminar con `fetch_failed` sin crear nada (supuesto S-8). Si no existe la base efectiva ni local ni como referencia remota local, DEBE rechazar con `base_not_found`, nombrando la rama esperada. `--base <ref>` sustituye la base (REQ-25.12). El `--fetch` sigue exigiendo la autorización de la persona.
(Previously: la base era siempre la rama de integración; ahora es la base efectiva del modo de promoción, que sin `main_branch` es la rama de integración)

#### Scenario: Sin `--fetch` no hay operaciones remotas
- **DADO** un repositorio con remoto y un observador instalado en el único punto del motor que ejecuta git
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el observador no registra ninguna invocación remota (`fetch`, `pull`, `push`, `ls-remote`)
- **Y** el JSON devuelve `fetched: false`

#### Scenario: La base es la referencia remota local si existe
- **DADO** un repositorio con `origin/main` y `main` locales en commits distintos
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** `base.ref` es `origin/main`, `base.sha` es el de `origin/main` y `base.source` indica la referencia remota local

#### Scenario: Sin referencia remota la base es la rama local
- **DADO** un repositorio sin remoto
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** `base.ref` es `main`, `base.sha` es el de `main` y `base.source` indica la rama local
- **Y** `fetched` es `false`

#### Scenario: `--fetch` es explícito
- **DADO** un repositorio con remoto
- **CUANDO** se ejecuta `ensure --change x --fetch`
- **ENTONCES** se ejecuta un `fetch` antes de resolver la base y el JSON devuelve `fetched: true`

#### Scenario: `--fetch` fallido no deja efectos
- **DADO** un remoto inalcanzable
- **CUANDO** se ejecuta `ensure --change x --fetch`
- **ENTONCES** se rechaza con `fetch_failed`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: Rama de integración inexistente
- **DADO** un `axiom.yaml` con `git.integration_branch: develop` y un repositorio sin `develop` ni `origin/develop`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `base_not_found` nombrando `develop`

#### Scenario: La rama del cambio publicada se indica con `createdFrom: remote`
- **DADO** un repositorio con `refs/remotes/origin/inc/x` y sin rama local `inc/x`
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** la entrada de `inc/x` en `branches[]` tiene `createdFrom: remote` y `createdFromRef: refs/remotes/origin/inc/x`
- **Y** `base.source` y `base.ref` siguen describiendo la base de integración
- **Y** `fetched` es `false`

#### Scenario: En modo `branch` la base es la referencia remota local de la rama principal
- **DADO** un repositorio con `main_branch: master`, `integration_branch: develop`, modo `branch`, y `origin/master` y `origin/develop` presentes en local
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** `base.ref` es `origin/master`, `base.sha` es el de `origin/master` y `base.source` indica la referencia remota local
- **Y** `fetched` es `false`

---

### Requirement: Rama de Archivado y `ensure --archive` (REQ-25.45)

En los repositorios con rama de cambio (`monorepo-embedded`), el archivado de un cambio DEBE hacerse en una rama propia, la rama de archivado `<prefijo>/<cambio>-archive`, y entregarse en un PR documental aparte, **después** de fusionar el PR del cambio (REQ-25.39). `axiom branch ensure --change <c> --archive` DEBE crearla en el repositorio del planning home, saliendo de la rama de integración con la base de REQ-25.10 (`origin/<integración>` si existe en local y, si no, la rama de integración local), y aplicando el modo de trabajo personal de REQ-25.8 (en modo `worktree`, un worktree en `<raíz de worktrees>/<cambio>-archive/<repo>`; en `in_place`, el checkout de la rama en el propio repositorio, sujeto a REQ-25.9). `--archive` admite `--prefix` (REQ-25.12). La rama de archivado sale siempre de la rama de integración y espera solo a que el cambio esté fusionado en ella, en cualquier modo de promoción: NUNCA espera a la fusión en la rama principal ni usa la base efectiva de REQ-25.47.

Si la base resuelta no contiene ya el cambio fusionado, es decir, si su árbol no incluye la carpeta `changes/<cambio>/` del planning home, o si la incluye pero el cambio está en fusión intermedia (REQ-26.11), `ensure --archive` DEBE rechazar con `change_not_integrated`, sin crear nada, nombrando la base evaluada y sugiriendo `--fetch` cuando la carpeta falta; cuando el rechazo se debe a una fusión intermedia, el mensaje DEBE indicarlo y NO DEBE sugerir `--fetch` como remedio, porque la base ya contiene la versión integrada. NO hay `fetch` implícito: solo con `--fetch` explícito se actualiza la referencia remota local antes de evaluar (REQ-25.10). `ensure --archive` es idempotente: reutiliza la rama y el worktree existentes sin moverlos, y NO publica la rama. En `multirepo` NO existe rama de archivado: `--archive` DEBE rechazarse con `archive_branch_not_applicable` porque el archive se hace in situ en la rama de integración del repositorio de specs (REQ-25.5); tampoco existe en el carril ODD (`--lane odd`), que no tiene archive. `done` también cubre la rama de archivado (REQ-25.16 y REQ-25.17) y `status` la incluye en el plan (REQ-25.15).
(Previously: no decía expresamente que, con un repositorio de dos etapas, la rama de archivado sale de la integración y solo espera a la fusión en ella)

#### Scenario: Crea la rama de archivado desde la integración con el cambio fusionado
- **DADO** un monorepo en modo `worktree`, el cambio `x` fusionado en `origin/main` (referencia local) y la política por defecto
- **CUANDO** se ejecuta `ensure --change x --archive --json`
- **ENTONCES** existe la rama `feat/x-archive` creada desde `origin/main`
- **Y** existe su worktree en `<raíz de worktrees>/x-archive/<repo>`
- **Y** `base.ref` es `origin/main` y `fetched` es `false`

#### Scenario: La base local sin el cambio fusionado se rechaza y sugiere `--fetch`
- **DADO** un monorepo cuya rama de integración local y cuya referencia remota local no contienen `openspec/changes/x/`
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se rechaza con `change_not_integrated` nombrando la base evaluada
- **Y** el mensaje sugiere `--fetch`
- **Y** no se crea ninguna rama ni worktree
- **Y** no consta ninguna operación remota

#### Scenario: Una fusión intermedia se rechaza con `change_not_integrated`
- **DADO** un monorepo cuya base contiene `openspec/changes/x/` en fusión intermedia (por ejemplo, solo `spec` y `design` aprobadas y cedidas)
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se rechaza con `change_not_integrated` nombrando la base evaluada
- **Y** el mensaje indica que la versión integrada es una fusión intermedia y no sugiere `--fetch` como remedio
- **Y** no se crea ninguna rama ni worktree
- **Y** no consta ninguna operación remota

#### Scenario: Con todos los role-apply aprobados e integrados, `--archive` procede
- **DADO** un monorepo cuya base contiene `openspec/changes/x/` con todos los `role-apply` aprobados, aunque alguna puerta anterior lleve marca de cesión
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se crea la rama de archivado como en el primer escenario

#### Scenario: `--fetch` explícito permite alcanzar el cambio fusionado
- **DADO** el mismo monorepo, con el cambio fusionado ya en el remoto pero no en las referencias locales
- **CUANDO** se ejecuta `ensure --change x --archive --fetch`
- **ENTONCES** se ejecuta un `fetch` antes de evaluar la base y el JSON devuelve `fetched: true`
- **Y** se crea la rama `feat/x-archive`

#### Scenario: Multirepo no tiene rama de archivado
- **DADO** un multirepo con el repositorio de specs en su rama de integración
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se rechaza con `archive_branch_not_applicable`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: El carril ODD no tiene rama de archivado
- **DADO** un monorepo
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature --archive`
- **ENTONCES** se rechaza con `archive_branch_not_applicable`

#### Scenario: Modo `in_place` hace checkout de la rama de archivado
- **DADO** un proyecto en modo `in_place` con el árbol limpio y el cambio `x` fusionado en la base
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** el repositorio queda desplegado en `feat/x-archive`
- **Y** no se crea ningún worktree

#### Scenario: Modo `in_place` con árbol sucio se rechaza
- **DADO** un proyecto en modo `in_place` con un fichero modificado sin confirmar y el cambio `x` fusionado en la base
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se rechaza con `dirty_worktree` sin cambiar de rama

#### Scenario: Idempotencia y ausencia de publicación
- **DADO** la rama de archivado ya creada por `ensure --archive`
- **CUANDO** se repite `ensure --change x --archive`
- **ENTONCES** el JSON es idéntico, el commit de la rama no cambia y no se ejecuta ningún `git push`

#### Scenario: Con dos etapas en modo `branch` el archivado espera solo a la integración
- **DADO** un repositorio con `main_branch: master`, `integration_branch: develop`, modo `branch` y el cambio `x` fusionado en `develop` pero no en `master`
- **CUANDO** se ejecuta `ensure --change x --archive --json`
- **ENTONCES** `feat/x-archive` se crea desde `origin/develop` y `base.ref` es `origin/develop`
- **Y** no se devuelve `change_not_integrated`

---

### Requirement: Registro Local en el Git Common Dir (REQ-25.14)

`ensure` DEBE mantener un registro local por cambio, fuera de cualquier árbol versionado, en el git common dir del repositorio del planning home, obtenido con `git rev-parse --git-common-dir` (nunca asumiendo `.git`). El registro DEBE contener las rutas absolutas de todos los repositorios, ramas (incluida la de archivado) y worktrees del cambio, la respuesta a `ask` del modo de trabajo (REQ-25.8) y, por cada repositorio con dos etapas, el modo de promoción resuelto y su origen (REQ-25.46). Los campos del modo de promoción DEBEN ser opcionales: un registro sin ellos sigue siendo válido y un binario anterior los ignora. El registro es la fuente operativa del modo de promoción en todos los carriles. Debe ser accesible con el mismo contenido desde el checkout principal y desde cualquier worktree enlazado del mismo repositorio. Si el registro se borra, el repositorio NO DEBE verse afectado y `ensure` DEBE reconstruirlo de forma idempotente a partir del estado de git (ramas y worktrees existentes) y, para el modo de promoción, de la instantánea del `kickoff.yaml` si la hay (REQ-25.46), mientras `status` DEBE informar que el cambio no tiene registro. La coexistencia con el estado local actual de RDD (`<git common dir>/gentle-ai/`) se resuelve en `sdd-design` (Q2) y NO DEBE alterar los linajes RDD existentes.
(Previously: el registro no guardaba el modo de promoción ni su origen y la reconstrucción no consultaba la instantánea)

#### Scenario: El registro se resuelve con el git common dir
- **DADO** un repositorio cuyo directorio git no se llama `.git` (git dir enlazado)
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el registro se crea bajo el directorio devuelto por `git rev-parse --git-common-dir`

#### Scenario: El registro es visible desde un worktree enlazado
- **DADO** un cambio preparado por `ensure` en modo `worktree`
- **CUANDO** se ejecuta `axiom branch status --change x --cwd <ruta del worktree>`
- **ENTONCES** el resultado incluye las rutas absolutas de todos los repositorios y worktrees del cambio registrados desde el checkout principal

#### Scenario: Borrar el registro no daña el repositorio
- **DADO** un cambio preparado y su registro borrado
- **CUANDO** se ejecuta `ensure --change x` de nuevo
- **ENTONCES** las ramas y worktrees existentes se reutilizan sin modificarlos
- **Y** el registro se reconstruye con las mismas rutas absolutas

#### Scenario: Registro ausente informado por `status`
- **DADO** un cambio sin registro local
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON indica que el cambio no tiene registro y no falla

#### Scenario: El registro guarda el modo de promoción y su origen por repositorio
- **DADO** un multirepo con dos repositorios de código con dos etapas, uno resuelto por respuesta y otro por la política
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el registro contiene, por cada repositorio, su modo y su origen

#### Scenario: Un registro sin modo de promoción sigue siendo válido
- **DADO** un registro escrito antes de esta capacidad, sin los campos del modo
- **CUANDO** se ejecutan `status` y `ensure --change x`
- **ENTONCES** ambas órdenes lo leen sin error y no lo consideran corrupto

#### Scenario: `ensure` conserva el modo registrado al reescribir el registro
- **DADO** un cambio con el modo `branch` registrado para `repo-backend`
- **CUANDO** se ejecuta `ensure --change x --archive` o de nuevo `ensure --change x`
- **ENTONCES** el modo de `repo-backend` sigue siendo `branch` en el registro

---

### Requirement: `status`: Estado de Ramas y Worktrees del Cambio (REQ-25.15)

`axiom branch status` DEBE informar, para cada repositorio del cambio, su papel (repositorio con rama de cambio, repositorio de specs in situ o repositorio de código), la rama esperada y la rama actual, la ruta absoluta del worktree si existe, si el árbol tiene cambios sin confirmar, el modo de trabajo materializado, la base, y el número de commits que la rama lleva de atraso o de adelanto respecto de su base calculado solo con referencias locales, y la rama destino propuesta para el PR (REQ-25.4). En un repositorio con dos etapas, la base es la base efectiva de su modo de promoción (REQ-25.47) y DEBE informar además de ese modo, de la rama principal y, en modo `branch` con la rama del cambio ya fusionada en la integración, de la siguiente PR con destino la rama principal (REQ-25.48). Cuando existe la rama de archivado, DEBE incluirla con los mismos datos. `--change` es opcional: sin él, `status` lista los cambios registrados y los cambios pendientes de archivar (REQ-25.20); con él, además informa de si el cambio solicitado está pendiente de archivar. NUNCA DEBE realizar operaciones remotas. DEBE poder ejecutarse desde el checkout principal, desde cualquier worktree del cambio y con `--cwd`. Un cambio sin ramas creadas DEBE informarse como no iniciado (`not_started`), no como error.
(Previously: `status` no informaba del modo de promoción ni de la siguiente PR, y la base era siempre la de integración)

#### Scenario: Estado de un cambio con rama y worktree
- **DADO** un cambio preparado por `ensure` en modo `worktree`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON incluye, por repositorio, la rama esperada, la rama actual, la ruta absoluta del worktree y si el árbol está limpio

#### Scenario: Atraso calculado solo con referencias locales
- **DADO** una rama `feat/x` con 3 commits menos que `main` y sin acceso a la red
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON informa de 3 commits de atraso respecto de la base
- **Y** no se ejecuta ninguna operación remota

#### Scenario: Cambio no iniciado
- **DADO** un nombre de cambio sin rama ni worktree
- **CUANDO** se ejecuta `axiom branch status --change nuevo --json`
- **ENTONCES** el estado es el token `not_started` con la etiqueta `no iniciado` y el código de salida es `0`

#### Scenario: Rama destino propuesta para el PR
- **DADO** un cambio con las ramas de rol `feat/x-core` y `feat/x-qa` y seguimiento `feat/x`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el destino propuesto del PR de cada rama de rol es `feat/x` y el de `feat/x` es la rama de integración

#### Scenario: La rama de archivado figura en el plan
- **DADO** un cambio con la rama de archivado `feat/x-archive` y su worktree
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON incluye la rama `feat/x-archive` con la ruta absoluta de su worktree
- **Y** su destino propuesto de PR es la rama de integración

#### Scenario: `status` sin `--change` lista registrados y pendientes
- **DADO** un proyecto con el cambio `x` registrado y el cambio `y` fusionado y pendiente de archivar
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** el JSON lista `x` entre los cambios registrados y `y` entre los pendientes de archivar

#### Scenario: La base y el modo de un repositorio con dos etapas figuran en el estado
- **DADO** un cambio en modo `branch` con `main_branch: master` y `integration_branch: develop`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** la base del repositorio es `master`
- **Y** el JSON informa del modo `branch` y de la rama principal `master`, además de los campos ya existentes

---

### Requirement: `done`: Detección de Fusión por Capas (REQ-25.16)

`axiom branch done` DEBE determinar, para cada rama del cambio, si ya está fusionada en su destino (la rama de rol en la de seguimiento; la rama del cambio en la de integración, o en la rama principal cuando el repositorio tiene dos etapas y modo `branch`, REQ-25.49; la rama de archivado en la de integración) con estas capas, de la primera a la última, deteniéndose en la primera que confirme:

1. **Ascendencia** (`ancestry`): la rama es ancestro del destino.
2. **Árbol equivalente** (`equivalent_tree`): la fusión simulada de la rama en el destino (`git merge-tree --write-tree`) produce el árbol del destino. Detecta los merges por squash o rebase aunque el destino haya avanzado después. La capa DEBE omitirse sin error si la versión de git no admite la orden (Q1, a verificar en `sdd-design`).
3. **Evidencia `pr_merged`** (`pr_merged`): el `gates.yaml` del cambio contiene una evidencia `pr_merged` ya verificada. Esta capa NO aplica a la rama de archivado: la evidencia del cambio acredita el PR del cambio y no el PR documental de archivado, de modo que la rama de archivado solo se confirma por las capas 1 y 2. Tampoco aplica cuando el destino es la rama principal en modo `branch`: la evidencia no identifica el destino del PR del cambio y por tanto no acredita la fusión en la rama principal (REQ-25.49).

Si ninguna capa confirma la fusión, la rama NO se considera fusionada y `done` DEBE exigir `--force` (REQ-25.17). El JSON DEBE indicar, por rama, qué capa la confirmó con los tokens `ancestry`, `equivalent_tree`, `pr_merged` o `forced`, cada uno con su etiqueta en castellano. La detección DEBE usar solo referencias locales; solo con `--fetch` PUEDE actualizarse el remoto antes de evaluar.
(Previously: el destino de la rama del cambio era siempre la rama de integración y la capa `pr_merged` se aplicaba a todas las ramas salvo la de archivado)

#### Scenario: Fusión real detectada por ascendencia
- **DADO** una rama de rol fusionada con merge commit en la de seguimiento
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama se confirma por el token `ancestry`

#### Scenario: Merge por squash detectado por la capa `equivalent_tree`
- **DADO** la rama `feat/x` integrada en `main` mediante un squash y `main` con commits posteriores
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama se confirma por el token `equivalent_tree`
- **Y** el JSON identifica esa capa

#### Scenario: Sin soporte de `merge-tree` la capa se omite
- **DADO** un git que no admite `git merge-tree --write-tree` y una rama integrada con squash
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la capa 2 se omite sin error
- **Y** si no hay evidencia `pr_merged`, `done` exige `--force`

#### Scenario: Evidencia `pr_merged` confirma la fusión
- **DADO** una rama integrada con squash, sin soporte de `merge-tree`, y un `gates.yaml` con evidencia `pr_merged` verificada
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama se confirma por el token `pr_merged`

#### Scenario: La rama de archivado se confirma por `equivalent_tree` tras un squash
- **DADO** la rama `feat/x-archive` integrada en la rama de integración mediante un squash
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama de archivado se confirma por el token `equivalent_tree`

#### Scenario: La evidencia `pr_merged` no confirma la rama de archivado
- **DADO** la rama `feat/x-archive` sin fusionar, sin confirmación por `equivalent_tree` y un `gates.yaml` con evidencia `pr_merged` del PR del cambio
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama de archivado se rechaza con `merge_not_confirmed`

#### Scenario: Nada confirma la fusión
- **DADO** una rama con commits propios sin fusionar, sin confirmación por `equivalent_tree` y sin evidencia
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `merge_not_confirmed` para esa rama y se exige `--force`
- **Y** no se elimina su worktree ni su rama local

#### Scenario: La detección no usa la red
- **DADO** un observador instalado en el único punto del motor que ejecuta git
- **CUANDO** se ejecuta `axiom branch done --change x` sin `--fetch`
- **ENTONCES** el observador no registra ninguna operación remota

#### Scenario: En modo `branch` la capa `pr_merged` no confirma la fusión en la rama principal
- **DADO** un repositorio con `main_branch: master`, modo `branch`, `feat/x` sin fusionar en `master` y un `gates.yaml` con evidencia `pr_merged`
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `merge_not_confirmed` nombrando `master`

#### Scenario: En modo `branch` las capas 1 y 2 se evalúan contra la rama principal
- **DADO** un repositorio con `main_branch: master`, `integration_branch: develop`, modo `branch` y `feat/x` fusionada en `master` con un squash
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama se confirma por el token `equivalent_tree` evaluado contra `master`

---

### Requirement: `done`: Seguridad y Eliminación de Worktree y Rama Local (REQ-25.17)

Para cada rama cuya fusión esté confirmada por la detección por capas de REQ-25.16, `done` DEBE eliminar su worktree (`git worktree remove`), después eliminar la rama local (también cuando git no la considere fusionada, como ocurre tras un squash, porque la fusión ya está confirmada por las capas) y actualizar el registro local. Si la eliminación del worktree falla, NO DEBE eliminarse la rama. `done` DEBE rechazar la operación sobre una rama cuya fusión no esté confirmada contra el destino que corresponde a su modo (`merge_not_confirmed`, REQ-25.49, nombrando ese destino) y sobre un worktree con cambios sin confirmar (`dirty_worktree`), salvo `--force`. Con `--force`, DEBE saltarse la exigencia de fusión confirmada y la de árbol limpio, y DEBE informar en el JSON de los cambios sin confirmar descartados y, si la rama no estaba fusionada, de la punta (SHA) que tenía la rama eliminada para poder recuperarla. `done` NUNCA DEBE tocar ramas remotas ni ejecutar operaciones remotas (REQ-25.1). Si queda abierto un linaje RDD asociado al worktree, `done` DEBE emitir un aviso que nombre el linaje y la acción (`axiom review abandon`), sin impedir la operación. Cada rama del cambio, incluida la de archivado, DEBE procesarse por separado, sin selector de rol en esta versión: las confirmadas se eliminan y las no confirmadas se conservan y se reportan, y el código de salida DEBE ser distinto de cero si alguna rama no pudo procesarse.

En modo `in_place` no hay worktree que eliminar: `done` DEBE devolver el checkout a la rama de integración (también en modo de promoción `branch`) y, después, eliminar la rama local, y solo si el árbol está limpio. En `in_place`, `--force` solo salta la exigencia de fusión confirmada: NUNCA salta la de árbol limpio, de modo que con cambios sin confirmar `done` rechaza con `dirty_worktree` sin cambiar de rama ni eliminar nada.
(Previously: el rechazo por fusión no confirmada no nombraba el destino ni distinguía el modo de promoción)

#### Scenario: Eliminación tras una fusión confirmada
- **DADO** una rama `feat/x` fusionada, su worktree con el árbol limpio y una rama remota `origin/feat/x`
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** el worktree se retira del disco y de `git worktree list`
- **Y** la rama local `feat/x` se elimina
- **Y** la rama remota `origin/feat/x` permanece intacta y no consta ninguna operación remota

#### Scenario: Una rama integrada con squash se elimina tras confirmarse por `equivalent_tree`
- **DADO** `feat/x` integrada en `main` mediante un squash, confirmada por la capa `equivalent_tree` (REQ-25.16)
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se eliminan su worktree y la rama local aunque git no la considere fusionada

#### Scenario: La rama de archivado se elimina como las demás
- **DADO** la rama `feat/x-archive` fusionada en la rama de integración y su worktree con el árbol limpio
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se eliminan el worktree y la rama local `feat/x-archive`
- **Y** no se toca ninguna rama remota

#### Scenario: Árbol sucio impide la eliminación
- **DADO** una rama fusionada cuyo worktree tiene un fichero modificado sin confirmar
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `dirty_worktree`
- **Y** el worktree y la rama local permanecen intactos

#### Scenario: Rama sin fusionar impide la eliminación
- **DADO** una rama con commits propios sin fusionar y sin evidencia, con su worktree limpio
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `merge_not_confirmed`
- **Y** el worktree y la rama local permanecen intactos

#### Scenario: `--force` elimina con fusión no confirmada y avisa de lo descartado
- **DADO** una rama sin fusionar y su worktree con cambios sin confirmar
- **CUANDO** se ejecuta `axiom branch done --change x --force`
- **ENTONCES** se eliminan el worktree y la rama local
- **Y** el JSON informa de que se descartaron cambios sin confirmar, indica la punta (SHA) de la rama eliminada y la capa de confirmación es el token `forced`
- **Y** no se toca ninguna rama remota

#### Scenario: Linaje RDD abierto genera un aviso
- **DADO** un worktree con un linaje RDD abierto
- **CUANDO** se ejecuta `axiom branch done --change x` sobre una rama fusionada
- **ENTONCES** la salida incluye un aviso que nombra el linaje y `axiom review abandon`
- **Y** la eliminación se realiza

#### Scenario: Ramas de rol y de seguimiento se procesan por separado
- **DADO** `feat/x-core` fusionada en `feat/x` y `feat/x` aún sin fusionar en `main`
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** el worktree y la rama local `feat/x-core` se eliminan
- **Y** el worktree y la rama local `feat/x` se conservan con `merge_not_confirmed`
- **Y** el código de salida es distinto de cero

#### Scenario: `in_place` con árbol limpio vuelve a la integración y elimina la rama local
- **DADO** un cambio materializado en modo `in_place`, desplegado en `feat/x`, fusionado y con el árbol limpio
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** el repositorio queda desplegado en la rama de integración
- **Y** la rama local `feat/x` se elimina
- **Y** no se toca ninguna rama remota

#### Scenario: `in_place` con árbol sucio se rechaza incluso con `--force`
- **DADO** un cambio en modo `in_place`, desplegado en `feat/x`, con un fichero modificado sin confirmar
- **CUANDO** se ejecuta `axiom branch done --change x` y después `axiom branch done --change x --force`
- **ENTONCES** ambas ejecuciones se rechazan con `dirty_worktree`
- **Y** el repositorio sigue desplegado en `feat/x` con la rama y el fichero intactos

#### Scenario: `in_place` sin fusión confirmada se rechaza
- **DADO** un cambio en modo `in_place`, con el árbol limpio y la rama `feat/x` sin fusionar
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `merge_not_confirmed` sin cambiar de rama

#### Scenario: El rechazo en modo `branch` nombra la rama principal
- **DADO** un repositorio con `main_branch: master`, modo `branch` y `feat/x` fusionada solo en la integración
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** el rechazo `merge_not_confirmed` nombra `master` como destino y no `develop`
- **Y** el worktree y la rama local permanecen intactos

#### Scenario: `in_place` en modo `branch` vuelve a la integración al borrar
- **DADO** un cambio en `in_place`, modo `branch`, con `feat/x` fusionada en `master` y el árbol limpio
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** el repositorio queda desplegado en la rama de integración y se elimina la rama local `feat/x`

---

### Requirement: Dogfooding en este Repositorio (REQ-25.24)

El repositorio de Axiom DEBE declarar en su `axiom.yaml` el bloque `git:` con `integration_branch: main` y `branch_prefix: inc`, y DEBE superar `axiom workspace validate`. `CLAUDE.md`, `AGENTS.md` y `GEMINI.md` DEBEN remitir a `axiom branch` para crear ramas y worktrees y NO DEBEN decir que el worktree se decide en apply. El destino de `scripts/axiom-worktree.ps1` y `.sh` y de `scripts/claude-hooks/guard-primary-checkout.ps1` (envoltorio fino o retirada) se resuelve en `sdd-design` (Q6). Los scripts `scripts/axiom-worktree.ps1` y `.sh` NO DEBEN fijar `main` a mano en la subida de la rama principal local tras `done`: DEBEN tomar la rama de integración resuelta del `repositories[].integrationBranch` del `axiom branch status --json` y, si no hay un JSON legible, caer a `main`; con la configuración actual de este repositorio `new`, `pr` y `done` DEBEN seguir funcionando igual. El producto no añade ninguna exención para cambios preexistentes: un cambio activo ya fusionado, como `inc-23-knowledge-profile` mientras no se archive, aparece como cambio pendiente de archivar (REQ-25.20) sin bloquear nada.
(Previously: los scripts de dogfooding fijaban `main` a mano tras `done`)

#### Scenario: `axiom.yaml` de este repositorio declara `git:`
- **DADO** el `axiom.yaml` del repositorio de Axiom
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** reporta `COMPLIANT`
- **Y** el bloque `git:` declara `integration_branch: main` y `branch_prefix: inc`

#### Scenario: La documentación remite al verbo
- **DADO** los textos de `CLAUDE.md`, `AGENTS.md` y `GEMINI.md`
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** cada uno remite a `axiom branch ensure`
- **Y** ninguno afirma que el worktree se crea en apply

#### Scenario: Un cambio preexistente ya fusionado figura como pendiente de archivar
- **DADO** el cambio `inc-23-knowledge-profile` presente en la referencia de integración de este repositorio sin archivar
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** figura entre los cambios pendientes de archivar y el código de salida es `0`

#### Scenario: `done` usa la integración resuelta del `status`
- **DADO** un `status --json` con `repositories[0].integrationBranch: develop` y el checkout principal en `develop`
- **CUANDO** se ejecuta `scripts/axiom-worktree.ps1 done <slug>` o `scripts/axiom-worktree.sh done <slug>`
- **ENTONCES** tras `axiom branch done` el script avanza `develop` con `--ff-only` y no `main`

#### Scenario: Sin JSON legible `done` cae a `main`
- **DADO** un `status --json` ausente o ilegible
- **CUANDO** se ejecuta `done <slug>` en cualquiera de los dos scripts
- **ENTONCES** el script usa `main` como hasta ahora y no falla por ese motivo

#### Scenario: Con la configuración actual los scripts siguen igual
- **DADO** este repositorio con `integration_branch: main` y sin `main_branch`
- **CUANDO** se ejecutan `new`, `pr` y `done` de ambos scripts
- **ENTONCES** su comportamiento y su salida no cambian
