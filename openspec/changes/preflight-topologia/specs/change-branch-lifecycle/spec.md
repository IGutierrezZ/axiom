# Delta para change-branch-lifecycle

> Parte del estado vigente: el resto de requisitos NO cambia. Esta delta adapta los requisitos de ramas y worktrees a la regla de `workspace-topology` (REQ-2.2): en `monorepo-embedded` el único rol es `fullstack`, así que no existen ramas de rol; los escenarios con roles especializados pasan a `multirepo` o a `fullstack`. La tabla del glosario «Roles del cambio sobre un repositorio» sigue siendo exacta (el rol `fullstack` en `monorepo-embedded` cuenta como un único rol) y no cambia.

## MODIFIED Requirements

### Requirement: Modelo de Ramas por Número de Roles que Tocan Cada Repositorio (REQ-25.4)

El modelo de ramas de cada repositorio con rama de cambio DEBE depender de cuántos roles del cambio lo tocan (roles sellados en el `kickoff.yaml` del cambio intersecados con los roles de `axiom.yaml` cuyos repositorios resuelven a esa raíz git):

- **Un rol** (lo habitual, también `fullstack`): solo existe la rama `<prefijo>/<cambio>`; el rol trabaja en ella y su PR se dirige a la rama de integración.
- **Dos o más roles** (solo posible en `multirepo`: en `monorepo-embedded` el único rol es `fullstack`, REQ-2.2 de `workspace-topology`, y por tanto nunca hay rama de rol): `<prefijo>/<cambio>` es la rama de seguimiento y cada rol saca `<prefijo>/<cambio>-<rol>` desde ella. Hay un PR de cada rol contra la rama de seguimiento y un PR final a la rama de integración que lleva el cambio (código y artefactos), sin el archive. Se RECOMIENDA el merge commit, sin imponerlo.
- **Rama de archivado** (solo en repositorios con rama de cambio): una vez fusionado en la rama de integración el PR del cambio, el archive se hace en `<prefijo>/<cambio>-archive`, que sale de la rama de integración (REQ-25.45) y se entrega en un PR documental aparte contra la rama de integración. NO existe en `multirepo`.

El modelo DEBE evaluarse por repositorio: en un mismo cambio un repositorio puede tener rama de rol y otro no. Un rol cuyo nombre sea `archive` colisionaría con la rama de archivado y DEBE rechazarse con `invalid_name` al planificar las ramas de rol (supuesto S-15). `ensure` NO DEBE publicar la rama de seguimiento: DEBE avisar de que la persona la sube al remoto antes del primer PR de rol. La rama destino propuesta para cada PR (rol contra seguimiento, cambio contra integración, archivado contra integración) DEBE poder obtenerse del plan (`status`, REQ-25.15) para que `sdd-apply` la proponga en lugar de preguntarla.
(Previously: el PR final a la rama de integración llevaba el archive y no existía rama de archivado; los escenarios de ramas de rol, de la colisión con `archive` y del destino de los PR se ejecutaban sobre un monorepo con los roles `core` y `qa`, y ahora se ejecutan sobre `multirepo` o sobre el rol `fullstack`)

#### Scenario: Un rol, rama única
- **DADO** un monorepo con el rol único `fullstack` sellado en el kickoff del cambio `x`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** existe solo la rama `feat/x`
- **Y** no existe ninguna rama `feat/x-<rol>`

#### Scenario: Dos roles sobre el mismo repositorio crean seguimiento y ramas de rol
- **DADO** un multirepo con los roles `core` y `qa` sellados en el kickoff del cambio `x`, ambos con repositorios que resuelven a la misma raíz git de código `repo-backend`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** en `repo-backend` existe la rama de seguimiento `feat/x` desde la base
- **Y** existen `feat/x-core` y `feat/x-qa`, ambas creadas desde `feat/x`
- **Y** la salida avisa de que la rama de seguimiento debe publicarla la persona antes del primer PR de rol
- **Y** no se ejecuta ningún `git push`

#### Scenario: En multirepo cada repositorio de código con un rol usa la rama del cambio
- **DADO** un multirepo con el rol `backend` sobre `repo-backend` y el rol `frontend` sobre `repo-frontend`, ambos sellados en el kickoff del cambio `x`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** `repo-backend` y `repo-frontend` reciben cada uno la rama `feat/x` desde su rama de integración
- **Y** ninguno recibe rama de rol ni rama de archivado

#### Scenario: Dos roles que comparten un repositorio de código en multirepo
- **DADO** un multirepo con los roles `api` y `jobs` sobre el mismo `repo-backend` y el rol `frontend` sobre `repo-frontend`
- **CUANDO** se ejecuta `ensure --change x` con los tres roles sellados
- **ENTONCES** `repo-backend` recibe `feat/x`, `feat/x-api` y `feat/x-jobs`
- **Y** `repo-frontend` recibe solo `feat/x`

#### Scenario: Un rol sellado que no toca ningún repositorio declarado no crea ramas
- **DADO** un rol sellado en el kickoff que no figura en `axiom.yaml` y que no es `fullstack`
- **CUANDO** se planifica el cambio
- **ENTONCES** ese rol no contribuye a ningún repositorio ni a ninguna rama
- **Y** la salida incluye un aviso que nombra el rol

#### Scenario: Un rol llamado `archive` colisiona con la rama de archivado
- **DADO** un multirepo con los roles `core` y `archive` sellados sobre el mismo repositorio de código
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `invalid_name` nombrando el rol `archive` y la colisión con el nombre reservado `feat/x-archive`
- **Y** no se crea ninguna rama de rol

#### Scenario: La rama de archivado no pertenece al modelo de roles
- **DADO** un monorepo con el rol único `fullstack` y las ramas `feat/x` y `feat/x-archive`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el destino propuesto del PR de `feat/x` es la rama de integración
- **Y** el de la rama de archivado `feat/x-archive` es la rama de integración
- **Y** el plan no contiene ninguna rama de rol


---

### Requirement: `ensure` Planifica de Forma Determinista y es Idempotente (REQ-25.7)

`axiom branch ensure` DEBE producir un plan determinista a partir de la topología, `git:`, `repositories:`, los roles del kickoff y las preferencias personales, y ejecutarlo con git. Una segunda ejecución con los mismos argumentos y el mismo estado DEBE no cambiar nada (ni ramas, ni worktrees, ni registro) y devolver el mismo JSON, salvo `branches[].createdFrom` y `branches[].createdFromRef` (REQ-27.15), que se emiten solo en la ejecución que crea la rama y no en las siguientes. El comando DEBE crear solo lo que falta: si la rama ya existe, DEBE reutilizarla sin moverla ni reiniciarla; si no existe en local pero existe su referencia remota local, DEBE adoptarla (REQ-27.15); si ya existe su worktree registrado, DEBE devolver su ruta. Si la rama ya está desplegada en un worktree que el registro local no conoce, DEBE rechazar con `branch_checked_out_elsewhere` nombrando la ruta. Cuando todavía no hay kickoff sellado (el cambio se nombra antes del sellado), `ensure` DEBE planificar solo para el repositorio del planning home la rama `<prefijo>/<cambio>`, que es idéntica en ambos modelos de REQ-25.4, y una ejecución posterior, ya con el kickoff sellado, DEBE completar de forma incremental las ramas de rol y los repositorios de código que correspondan sin alterar lo ya creado (supuesto S-1). `ensure --change <c> --archive` (REQ-25.45) es un plan aparte con las mismas garantías de determinismo e idempotencia: no altera las ramas del cambio ya creadas.
(Previously: los escenarios de ejecución previa y posterior al sellado y de `--archive` suponían un monorepo con los roles `core` y `qa` y la rama `feat/x-core`; ahora el de roles se ejecuta sobre `multirepo` y los de monorepo sobre el rol `fullstack`)


#### Scenario: Segunda ejecución sin cambios
- **DADO** un cambio ya preparado por `ensure`
- **CUANDO** se ejecuta de nuevo `ensure --change x --json` con los mismos argumentos
- **ENTONCES** el JSON es idéntico al de la primera ejecución, salvo `createdFrom` y `createdFromRef`, que solo aparecen en la ejecución que creó la rama
- **Y** `git worktree list` y `git branch --list` no han cambiado

#### Scenario: La rama existente se reutiliza sin moverla
- **DADO** una rama `feat/x` ya existente con commits propios y sin worktree
- **CUANDO** se ejecuta `ensure --change x` en modo `worktree`
- **ENTONCES** se crea el worktree sobre `feat/x`
- **Y** el commit al que apunta `feat/x` no cambia

#### Scenario: Rama desplegada en un worktree desconocido
- **DADO** una rama `feat/x` desplegada en un worktree que el registro local no conoce
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `branch_checked_out_elsewhere` nombrando la ruta del worktree existente
- **Y** no se crea ningún otro worktree

#### Scenario: Ejecución previa al sellado y posterior al sellado
- **DADO** un cambio nombrado sin `kickoff.yaml`, en un multirepo con los roles `core` y `qa` sobre el mismo repositorio de código `repo-backend`
- **CUANDO** se ejecuta `ensure --change x` antes del sellado y de nuevo después de sellar los dos roles
- **ENTONCES** la primera ejecución crea solo `feat/x` en el repositorio del planning home
- **Y** la segunda conserva lo ya creado y crea en `repo-backend` `feat/x`, `feat/x-core` y `feat/x-qa` (las de rol desde `feat/x`)

#### Scenario: En monorepo la ejecución posterior al sellado no añade ramas
- **DADO** un cambio nombrado sin `kickoff.yaml`, en un monorepo
- **CUANDO** se ejecuta `ensure --change x` antes del sellado y de nuevo después de sellar el rol `fullstack`
- **ENTONCES** ambas ejecuciones dejan solo la rama `feat/x`
- **Y** la segunda no cambia nada

#### Scenario: Se rechaza un nombre de cambio que no es válido
- **DADO** un nombre de cambio con caracteres inválidos como componente de rama git
- **CUANDO** se ejecuta `ensure --change "a b:c"`
- **ENTONCES** se rechaza con `invalid_name`
- **Y** no se crea nada

#### Scenario: `--archive` no altera las ramas del cambio
- **DADO** un monorepo con la rama `feat/x` ya creada y su PR fusionado
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se crea `feat/x-archive` y el commit al que apunta `feat/x` no cambia

#### Scenario: La rama publicada se adopta en lugar de crearse desde la base
- **DADO** una rama `feat/x` inexistente en local y `refs/remotes/origin/feat/x` presente
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** `feat/x` se crea desde `origin/feat/x`
- **Y** una segunda ejecución con los mismos argumentos devuelve el mismo JSON sin cambiar nada


---

### Requirement: Rutas de Worktree y Límite de Longitud (REQ-25.11)

La ruta del worktree de cada rama DEBE ser `<raíz de worktrees>/<cambio>[-<rol>|-archive]/<repo>`, donde el sufijo `-<rol>` solo aparece si existe rama de rol para ese repositorio (REQ-25.4), el sufijo `-archive` solo aparece en el worktree de la rama de archivado (REQ-25.45) y `<repo>` es el último segmento de la ruta (`path`) del repositorio, tal como la declara `repositories:` o el rol (supuesto S-4); para el repositorio cuya ruta es `.`, `<repo>` es el nombre del directorio de su raíz git, y si dos repositorios del plan tuvieran el mismo último segmento, `ensure` DEBE rechazar con `invalid_name` nombrando ambos, antes de crear nada (supuesto S-17). La raíz de worktrees DEBE ser la preferencia personal del hub o, sin ella, `<padre de la carpeta maestra>/<nombre de la carpeta maestra>-wt`, fuera de cualquier repositorio. El planificador DEBE validar, antes de crear nada, que el nombre del cambio es válido a la vez como componente de rama git y como nombre de directorio en Windows, y que la longitud de cada ruta de worktree no supera el límite admitido: el de la plataforma menos un margen de 40 caracteres, es decir, 220 caracteres en Windows sin rutas largas habilitadas (260 menos 40). Si no se cumple, DEBE rechazar con `invalid_name` o `path_too_long`, nombrando la ruta, su longitud y el máximo, sin crear ramas ni worktrees.
(Previously: `<repo>` era el nombre declarado en `repositories:` o el del directorio de la raíz git, la ruta no contemplaba la rama de archivado y el límite solo exigía «margen» sin cifra; el escenario del sufijo de rol se ejecutaba sobre un monorepo con los roles `core` y `qa` y ahora sobre `multirepo`)

#### Scenario: Ruta sin sufijo de rol con una sola rama
- **DADO** un monorepo con un rol y modo `worktree`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el worktree está en `<raíz de worktrees>/x/<repo>`
- **Y** la ruta no contiene sufijo de rol

#### Scenario: Ruta con sufijo de rol cuando hay rama de rol
- **DADO** un multirepo con los roles `core` y `qa` sobre el mismo repositorio de código
- **CUANDO** se ejecuta `ensure --change x` con ambos sellados
- **ENTONCES** el worktree de la rama de seguimiento está en `<raíz de worktrees>/x/<repo>`
- **Y** los de las ramas de rol están en `<raíz de worktrees>/x-core/<repo>` y `<raíz de worktrees>/x-qa/<repo>`

#### Scenario: Ruta de la rama de archivado
- **DADO** un monorepo en modo `worktree` y el cambio `x` fusionado
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** el worktree de la rama de archivado está en `<raíz de worktrees>/x-archive/<repo>`

#### Scenario: `<repo>` es el último segmento de la ruta declarada
- **DADO** un multirepo con un repositorio declarado con `path: services/repo-backend`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el worktree de ese repositorio está en `<raíz de worktrees>/x/repo-backend`

#### Scenario: Dos repositorios con el mismo último segmento se rechazan
- **DADO** dos repositorios del plan declarados con `path: a/api` y `path: b/api`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `invalid_name` nombrando ambas rutas
- **Y** no se crea ninguna rama ni worktree

#### Scenario: Raíz de worktrees por defecto fuera de cualquier repositorio
- **DADO** una carpeta maestra `<P>/axiom` sin preferencia de raíz en el hub
- **CUANDO** se calcula la raíz de worktrees
- **ENTONCES** es `<P>/axiom-wt`, que no está dentro de ningún repositorio del proyecto

#### Scenario: Ruta demasiado larga en Windows
- **DADO** una raíz de worktrees personalizada y un nombre de cambio cuyo worktree tendría 221 caracteres
- **CUANDO** se ejecuta `ensure --change <nombre largo>`
- **ENTONCES** se rechaza con `path_too_long` nombrando la ruta, su longitud (221) y el máximo (220)
- **Y** no se crea ninguna rama ni worktree

#### Scenario: Una ruta de exactamente 220 caracteres se admite
- **DADO** una raíz de worktrees personalizada y un nombre de cambio cuyo worktree tendría exactamente 220 caracteres
- **CUANDO** se ejecuta `ensure --change <nombre>`
- **ENTONCES** se crea el worktree

#### Scenario: Nombre con carácter inválido en Windows
- **DADO** un nombre de cambio con un carácter no admitido en nombres de directorio de Windows (por ejemplo `:` o `?`)
- **CUANDO** se ejecuta `ensure --change`
- **ENTONCES** se rechaza con `invalid_name` antes de crear nada



---
### Requirement: Veredicto Único de Rama, por Papel del Repositorio (REQ-25.18)

Las guardas A (REQ-25.19), B (`blocked(wrong_branch)` de `axiom-sdd-cli-integration`) y D (`guard`, REQ-25.21) DEBEN consultar un único veredicto: «¿esta rama es la que corresponde a este cambio en este repositorio según su papel?». El veredicto DEBE ser uno de los tokens `correct`, `wrong_branch` o `not_applicable` (REQ-25.2). La consulta se hace en uno de dos contextos: el **ordinario** y el de **archivado** (la fase archive de `axiom sdd status` y el escritor de `axiom sdd archive-compose`). Las reglas son:

- **Repositorio con rama de cambio, contexto ordinario**: es correcta la rama del cambio o, si existen ramas de rol de ese cambio, cualquiera de ellas; cualquier otra rama (incluida la de integración, la de archivado y un HEAD desacoplado) es `wrong_branch`.
- **Repositorio con rama de cambio, contexto de archivado**: es correcta únicamente la rama de archivado `<prefijo>/<cambio>-archive`; cualquier otra (la del cambio, la de integración, una de rol o una ajena) es `wrong_branch`, y su continuación DEBE ser `axiom branch ensure --cwd <ruta> --change <c> --archive`.
- **Repositorio de specs en multirepo**: es correcta únicamente la rama de integración, también en el contexto de archivado (no hay rama de archivado); una rama de rol del cambio o cualquier otra es `wrong_branch`.
- **Repositorio de código en multirepo**: el veredicto solo se evalúa cuando el consultante es apply (el estado de aplicación de `sdd status` o la guarda D sobre una ruta de ese repositorio); en otras fases es `not_applicable`.
- **Exenciones**: es `not_applicable` para un cambio cuyo `kickoff.yaml` no contiene instantánea (REQ-25.37), para los sumideros `e2e-cumulative` y `cumulative-<rol>` por nombre, y cuando la ruta no pertenece a ningún repositorio git o git no está disponible (en esos casos la consulta NO DEBE bloquear y DEBE advertirlo).

Los nombres esperados DEBEN obtenerse de la instantánea del `kickoff.yaml` cuando existe (REQ-25.36) y, si el cambio aún no tiene kickoff (se está creando), de la política de `axiom.yaml` (REQ-25.3); el nombre de la rama de archivado se toma del registro local si `ensure --archive` recibió `--prefix` (REQ-25.12). Un `wrong_branch` DEBE incluir la rama actual, la esperada, el repositorio y la continuación exacta: la orden `axiom branch ensure` con `--cwd`, `--change` y, si procede, `--lane` o `--archive`. El veredicto NO consulta si la rama de integración contiene cambios activos: con el archive en un PR documental aparte es normal que los contenga temporalmente (REQ-25.20).
(Previously: lo consultaban cuatro capas, incluida una de CI; el veredicto no distinguía el contexto de archivado y los valores eran etiquetas en castellano)

#### Scenario: Rama del cambio en un monorepo es correcta
- **DADO** un cambio con instantánea y el repositorio del planning home desplegado en `feat/x`
- **CUANDO** se consulta el veredicto en el contexto ordinario
- **ENTONCES** es `correct`

#### Scenario: La rama de integración es incorrecta para un cambio en monorepo
- **DADO** un cambio con instantánea y el repositorio del planning home desplegado en `main`
- **CUANDO** se consulta el veredicto en el contexto ordinario
- **ENTONCES** es `wrong_branch` con rama actual `main` y rama esperada `feat/x`
- **Y** la continuación es la orden exacta de `axiom branch ensure --cwd <ruta> --change x`

#### Scenario: Una rama de rol es correcta cuando hay ramas de rol
- **DADO** un multirepo con los roles `core` y `qa` sobre el mismo repositorio de código y ese repositorio desplegado en `feat/x-qa`
- **CUANDO** se consulta el veredicto en el contexto ordinario
- **ENTONCES** es `correct`

#### Scenario: La rama de archivado es incorrecta en el contexto ordinario
- **DADO** un cambio con instantánea y el repositorio desplegado en `feat/x-archive`
- **CUANDO** se consulta el veredicto en el contexto ordinario
- **ENTONCES** es `wrong_branch` con rama esperada `feat/x`

#### Scenario: En el contexto de archivado solo la rama de archivado es correcta
- **DADO** un cambio con instantánea y el repositorio desplegado en `feat/x-archive`
- **CUANDO** se consulta el veredicto en el contexto de archivado
- **ENTONCES** es `correct`

#### Scenario: En el contexto de archivado la rama del cambio, la de integración y una ajena son incorrectas
- **DADO** un monorepo con instantánea y el repositorio desplegado, por separado, en `feat/x`, en `main` y en una rama ajena `feat/otra`
- **CUANDO** se consulta el veredicto en el contexto de archivado
- **ENTONCES** es `wrong_branch` en los tres casos con rama esperada `feat/x-archive`
- **Y** la continuación es `axiom branch ensure --cwd <ruta> --change x --archive`

#### Scenario: Un HEAD desacoplado es incorrecto
- **DADO** un cambio con instantánea y el repositorio con HEAD desacoplado
- **CUANDO** se consulta el veredicto
- **ENTONCES** es `wrong_branch` y el detalle indica que el HEAD está desacoplado

#### Scenario: El repositorio de specs de un multirepo exige su rama de integración
- **DADO** un multirepo con un cambio con instantánea y el repositorio de specs desplegado en `feat/x-core`
- **CUANDO** se consulta el veredicto
- **ENTONCES** es `wrong_branch` con rama esperada `main`

#### Scenario: El repositorio de código de un multirepo solo se evalúa en apply
- **DADO** un multirepo con `repo-backend` desplegado en `main` y un cambio con instantánea
- **CUANDO** se consulta el veredicto desde la fase `spec`
- **ENTONCES** es `not_applicable`
- **Y** desde la fase de aplicación de una tarea que toca `repo-backend` es `wrong_branch`

#### Scenario: Un cambio preexistente sin instantánea queda exento
- **DADO** un cambio con `kickoff.yaml` sin instantánea y el repositorio desplegado en `main`
- **CUANDO** se consulta el veredicto
- **ENTONCES** es `not_applicable`

#### Scenario: Los cambios activos en la rama de integración no alteran el veredicto
- **DADO** un cambio `y` fusionado y presente en el árbol de la rama de integración, y un cambio `x` con instantánea en `feat/x`
- **CUANDO** se consulta el veredicto de `x` desplegado en `feat/x`
- **ENTONCES** es `correct`

#### Scenario: Fuera de un repositorio git no se bloquea
- **DADO** un directorio de proyecto que no pertenece a ningún repositorio git
- **CUANDO** se consulta el veredicto
- **ENTONCES** no se bloquea y se emite un aviso


---

### Requirement: `ensure` Adopta la Rama Publicada si Existe como Referencia Remota Local (REQ-27.15)

Si la rama de un cambio (rama del cambio, rama de rol o rama de archivado) NO existe como rama local y SÍ existe la referencia local `refs/remotes/origin/<rama>`, `axiom branch ensure` DEBE crear la rama local desde esa referencia, con `origin/<rama>` como rama de seguimiento (upstream), en lugar de crearla desde la base de integración, y DEBE materializar su worktree o hacer su checkout según el modo de trabajo (REQ-25.8). El JSON DEBE indicarlo en la entrada de esa rama de `branches[]` con `createdFrom: remote` y `createdFromRef: refs/remotes/origin/<rama>` (tokens estables en inglés, con su etiqueta legible en castellano); `base.ref`, `base.sha` y `base.source` NO cambian de semántica y siguen describiendo la base de integración. La adopción solo usa lecturas locales: NO DEBE ejecutar `fetch`, `pull` ni `ls-remote` por defecto; solo con `--fetch` explícito y autorizado (REQ-25.10) se ejecuta el `fetch` previo, y entonces la adopción usa la referencia ya actualizada. Si la rama local ya existe, se reutiliza sin moverla (REQ-25.7), aunque exista la referencia remota. Si no existe la referencia remota, `ensure` DEBE comportarse como hasta ahora (crear desde la base) y el resultado DEBE sugerir `--fetch` (con la salvedad de que requiere autorización de la persona). La adopción NO DEBE alterar el kickoff sellado ni modificar la referencia remota. En `monorepo-embedded`, una cesión publicada (REQ-26.5 de `sdd-block-review-gates`) ya NO exige una PR fusionada para que otra persona continúe: basta `git fetch` (o `ensure --fetch`) y `ensure`.

#### Scenario: Adopta la rama publicada sin rama local

- **DADO** un monorepo cuyo remoto `origin` tiene `inc/x` con documentos del cambio, la persona ha hecho `git fetch` y no existe la rama local `inc/x`
- **CUANDO** ejecuta `axiom branch ensure --change x --json`
- **ENTONCES** la rama local `inc/x` apunta al SHA de `origin/inc/x`
- **Y** su upstream es `origin/inc/x`
- **Y** la entrada de `inc/x` en `branches[]` indica `createdFrom: remote` y `createdFromRef: refs/remotes/origin/inc/x`
- **Y** `base.source` sigue describiendo la base de integración
- **Y** los artefactos publicados están disponibles en el worktree

#### Scenario: Sin referencia remota se comporta como hasta ahora

- **DADO** un repositorio sin `refs/remotes/origin/inc/x` ni rama local `inc/x`
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** `inc/x` se crea desde la base de integración
- **Y** el resultado sugiere `--fetch` indicando que requiere autorización de la persona

#### Scenario: Ningún fetch implícito

- **DADO** un observador instalado en el único punto del producto que ejecuta git
- **CUANDO** se ejecuta `ensure --change x` sin `--fetch`, con y sin referencia remota de la rama
- **ENTONCES** el observador no registra `fetch`, `pull`, `push` ni `ls-remote`
- **Y** el JSON devuelve `fetched: false`

#### Scenario: Con `--fetch` se adopta la referencia actualizada

- **DADO** un remoto con `inc/x` que la persona aún no ha traído y la orden `ensure --change x --fetch` autorizada
- **CUANDO** se ejecuta
- **ENTONCES** se ejecuta el `fetch`, `fetched` es `true` y la rama local se crea desde `origin/inc/x`
- **Y** la entrada de `inc/x` en `branches[]` indica `createdFrom: remote`

#### Scenario: Una rama local existente no se mueve

- **DADO** una rama local `inc/x` con commits propios y una referencia `origin/inc/x` distinta
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** la rama local no cambia de commit
- **Y** la entrada de `inc/x` en `branches[]` no indica `createdFrom: remote`

#### Scenario: Adopta también la rama de rol en multirepo

- **DADO** un multirepo con dos roles sobre el mismo repositorio de código y la referencia remota local `inc/x-core` sin rama local
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** la rama local `inc/x-core` se crea desde su referencia remota
- **Y** su entrada en `branches[]` indica `createdFrom: remote` con su `createdFromRef`

#### Scenario: Adopta también la rama de archivado en monorepo

- **DADO** un monorepo y la referencia remota local `inc/x-archive` sin rama local
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** la rama local `inc/x-archive` se crea desde su referencia remota
- **Y** su entrada en `branches[]` indica `createdFrom: remote` con su `createdFromRef`

#### Scenario: En modo `in_place` se adopta y se hace checkout

- **DADO** un proyecto en modo `in_place`, árbol limpio y `origin/inc/x` presente sin rama local
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se crea `inc/x` desde `origin/inc/x` y se hace su checkout en el propio repositorio

#### Scenario: Idempotencia tras adoptar

- **DADO** un cambio ya adoptado
- **CUANDO** se ejecuta de nuevo `ensure --change x --json`
- **ENTONCES** el JSON es idéntico y no cambia ninguna rama ni worktree

#### Scenario: Quien continúa un relevo no necesita una PR fusionada

- **DADO** una compuerta `spec` aprobada con la salida 2 y publicada en `origin/inc/x`, sin PR fusionada
- **CUANDO** otra persona ejecuta `git fetch` y `axiom branch ensure --change x`
- **ENTONCES** obtiene la rama `inc/x` con los documentos aprobados
- **Y** `axiom sdd status` proyecta la fase siguiente cedida

## Supuestos confirmados en la compuerta de spec (2026-10-15)

Los supuestos son decisiones mínimas tomadas para que los requerimientos sean comprobables. El usuario los confirmó en la compuerta de spec del 2026-10-15; S-6 se corrigió según D3. Q1, Q2 y Q6 de la propuesta pertenecen a `sdd-design` y se citan como dependencias.

> **Nota de reapertura (2026-10-17):** la spec se reabrió por D16 a D19. D15 queda retirada y S-12 se resuelve ahora por D16. S-4 se actualiza por la lectura G-5 y S-5 queda sin objeto por D17. Los supuestos S-15 a S-18 se añaden en la reapertura.

| ID | Supuesto | Estado | Dónde afecta |
|---|---|---|---|
| S-1 | La propuesta no indica cómo conocer los roles cuando `ensure` se ejecuta al nombrar el cambio, antes de `kickoff seal`. `ensure` es incremental: antes del sellado planifica solo `<prefijo>/<cambio>` en el repositorio del planning home (idéntica en ambos modelos) y tras el sellado completa ramas de rol y repositorios de código. | Confirmado | REQ-25.7 |
| S-2 | La respuesta a `decision_required` de `ask` se da con la opción `--mode worktree\|in_place` de `ensure`. | Confirmado | REQ-25.8 |
| S-3 | `fullstack` no existe en `axiom.yaml`. Cuenta como un único rol que toca todos los repositorios declarados por algún rol y, en monorepo, el del planning home. | Confirmado | Términos, REQ-25.4 |
| S-4 | `<repo>` es el último segmento de la ruta (`path`) del repositorio (G-5), y `<workspace>-wt` usa el nombre de la carpeta maestra. | Confirmado (actualizado por G-5) | REQ-25.11 |
| S-5 | Retirado por D17: existía para fijar el mecanismo con el que el comprobador de CI evaluaba un árbol desacoplado, y el comprobador se ha eliminado. | Sin objeto (D17) | — |
| S-6 | Resuelto según D3: `done` elimina el worktree y la rama local una vez confirmada la fusión por capas (REQ-25.16); se niega si la rama no está fusionada o si el árbol está sucio, salvo `--force`; en `in_place` devuelve el checkout a la rama de integración y elimina la rama local solo con el árbol limpio (`--force` no salta esa exigencia); nunca toca ramas remotas; procesa cada rama por separado, incluida la de archivado, sin selector de rol en esta versión. | Confirmado (corregido, D3) | REQ-25.17 |
| S-7 | Si el cambio ya tiene `kickoff.yaml` pero sin instantánea queda exento; si aún no tiene `kickoff.yaml` se aplica la política de `axiom.yaml`. | Confirmado | REQ-25.18 |
| S-8 | `--fetch` fallido termina con `fetch_failed` sin crear nada. | Confirmado | REQ-25.10 |
| S-9 | `blocked(wrong_branch)` se evalúa antes de `blocked(edit_authority_missing)`. | Confirmado | `axiom-sdd-cli-integration` |
| S-10 | La pregunta de `ask` del carril ODD es una preferencia de entorno personal, no una pregunta de gobernanza, y no contraviene REQ-21.2. | Confirmado | `sdd-lane-kickoff` |
| S-11 | Si el directorio de partida es un worktree enlazado sin marcador ascendente, el planning home se resuelve desde el checkout principal del mismo clon. | Confirmado | `planning-home-resolution` |
| S-12 | Resuelto por D16 (sustituye a la decisión anterior): en monorepo el archive se hace en un PR documental aparte, en la rama de archivado `<prefijo>/<cambio>-archive` (D18), una vez fusionado el PR del cambio. REQ-21.16 vuelve a su forma original con dos cambios (rama de integración configurada y sin `monorepo-decoupled`) y la precondición incluye la evidencia `pr_merged` del PR del cambio. Archivar en otra rama se rechaza con `blocked(wrong_branch)` y la continuación `ensure --archive`. `multirepo` no cambia. | Confirmado (D16) | `sdd-archive-lifecycle` |
| S-13 | `repositories:` identifica cada entrada por `path` y admite `integration_branch` y `branch_prefix` como ajustes propios por repositorio. | Confirmado (incluye `branch_prefix` por repositorio) | `workspace-topology` |
| S-14 | La TUI se cubre como escenario de paridad en `local-web-dashboard`; `tui-ui-parity` no figura en las capacidades modificadas de la propuesta. | Confirmado | `local-web-dashboard` |
| S-15 | Un rol llamado `archive` colisionaría con la rama de archivado `<prefijo>/<cambio>-archive` y se rechaza con `invalid_name` al planificar las ramas de rol. | Propuesto en la reapertura (2026-10-17) | REQ-25.4 |
| S-16 | Con `--archive` solo se admite `--prefix`; `--branch` y `--base` se rechazan con `plan_conflict`; sin `--prefix` la rama de archivado deriva de la rama del cambio registrada como `<rama del cambio>-archive`. Una base «contiene el cambio fusionado» cuando su árbol incluye `changes/<cambio>/` del planning home y el cambio no está en fusión intermedia (REQ-26.11). | Propuesto en la reapertura (2026-10-17); enmendado por REQ-25.45 y REQ-26.11 (paradas-tres-salidas) | REQ-25.12, REQ-25.45 |
| S-17 | Para el repositorio cuya ruta es `.`, `<repo>` es el nombre del directorio de su raíz git; dos repositorios del plan con el mismo último segmento se rechazan con `invalid_name`. | Propuesto en la reapertura (2026-10-17) | REQ-25.11 |
| S-18 | D19: el bloqueo de documentos de Axiom (carpeta del cambio y `odd/tasks/`) rige en todo proyecto Axiom aunque no haya cambio registrado, para cubrir el carril ODD antes de `ensure`; el aviso en otros ficheros solo rige en repositorios con un cambio activo registrado; el repositorio de specs de un multirepo queda exento del aviso pero no del bloqueo de documentos. | Propuesto en la reapertura (2026-10-17) | REQ-25.21 |
| A-7 | Un cambio con todos los `role-apply` aprobados deja de ser fusión intermedia aunque la última puerta lleve marca de cesión (la fase `verify` cedida no hace intermedia la fusión del trabajo ya implementado). | Confirmado (paradas-tres-salidas) | REQ-26.11 |
| A-8 | Un cambio fusionado sin ninguna marca de cesión en su registro de puertas (incluidos los cambios preexistentes y los sellados en modalidad continua) se comporta exactamente como hasta ahora. | Confirmado (paradas-tres-salidas) | REQ-26.11 |

### Dependencias de diseño

| ID | Dependencia | Dónde afecta |
|---|---|---|
| Q1 | Versión mínima de git para `git merge-tree --write-tree` (2.38). | REQ-25.16 (design) |
| Q2 | Coexistencia del registro local con `<git common dir>/gentle-ai/`. | REQ-25.14 (design) |
| Q6 | Destino de los scripts locales y de `guard-primary-checkout.ps1`. | REQ-25.24 (design) |


