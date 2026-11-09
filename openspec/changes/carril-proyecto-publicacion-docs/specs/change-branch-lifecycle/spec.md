# Delta para Ciclo de Vida de las Ramas de un Cambio (carril-proyecto-publicacion-docs)

> **Cambio:** `carril-proyecto-publicacion-docs` · **Tipo:** delta sobre la capacidad viva `change-branch-lifecycle`.
> **Idioma:** castellano de España; identificadores, tokens y códigos en inglés.

Cambios de este delta:

- REQ-27.15 (nuevo): `ensure` adopta `refs/remotes/origin/<rama>` cuando la rama del cambio (o de rol, o de archivado) no existe en local y esa referencia existe, sin fetch implícito; el JSON lo indica. Quien continúa un relevo ya no necesita una PR fusionada.
- REQ-25.1: el invariante «ningún subcomando publica» queda acotado a `axiom branch`; el verbo `axiom sdd publish` vive fuera de la familia y usa un puerto propio (REQ-27.11 de `sdd-docs-publication`).
- REQ-25.7: «crea solo lo que falta» contempla la adopción de la rama publicada.
- REQ-25.10: la rama adoptada se indica en `branches[].createdFrom` y `createdFromRef`; `base.source` conserva su semántica (origen de la base de integración); sin fetch implícito.
- REQ-26.11: una rama publicada no es una rama integrada; la fusión intermedia sigue definida por la rama de integración local.
- REQ-25.8 y el resto de REQ-25.x no cambian.

## ADDED Requirements

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

#### Scenario: Adopta también la rama de rol y la de archivado

- **DADO** referencias remotas locales para `inc/x-core` y `inc/x-archive` sin ramas locales
- **CUANDO** se ejecutan `ensure --change x` y `ensure --change x --archive`
- **ENTONCES** cada rama local se crea desde su referencia remota respectiva
- **Y** la entrada de cada una en `branches[]` indica `createdFrom: remote` con su `createdFromRef`

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

## MODIFIED Requirements

### Requirement: Familia de Comandos `axiom branch` (REQ-25.1)

La CLI DEBE exponer la familia de primer nivel `axiom branch` con exactamente los subcomandos `ensure`, `status`, `done` y `guard`. La familia DEBE ser neutral respecto al carril: `ensure`, `status` y `done` DEBEN aceptar `--change <nombre>` y `--lane sdd|odd` (por defecto `sdd`) y NO DEBEN depender de que exista un cambio SDD. Ningún subcomando de `axiom branch` DEBE ejecutar `git push` ni `git commit`, crear un PR ni realizar operaciones remotas salvo la solicitada por `--fetch` (REQ-25.10). La familia NO DEBE exponer subcomandos `pr`, `push` ni ningún comprobador para la CI. Toda ejecución de git del motor DEBE pasar por un único punto, de modo que la ausencia de operaciones remotas sea verificable con un observador instalado en ese punto y con una prueba de arquitectura que cubra el código de `internal/branch/**` y de `internal/gitcommon`. La operación de subir documentación del cambio es el verbo `axiom sdd publish`, que NO pertenece a esta familia, usa un puerto de publicación propio al que `axiom branch` no puede llegar y no altera este invariante (REQ-27.11 de `sdd-docs-publication`).
(Previously: «Ningún subcomando» sin acotar a la familia `axiom branch`, sin mencionar `git commit` ni el verbo de publicación)

#### Scenario: La ayuda lista los cuatro verbos
- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom branch --help`
- **ENTONCES** la salida lista `ensure`, `status`, `done` y `guard`
- **Y** no lista `pr`, `push` ni ningún comprobador para la CI

#### Scenario: Un subcomando no listado no existe
- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom branch` seguido de `push`, de `pr` o del verbo retirado `check`
- **ENTONCES** el comando se rechaza como no reconocido con código de salida distinto de cero

#### Scenario: Ningún subcomando publica ni abre PR
- **DADO** un repositorio con un remoto configurado y un observador instalado en el único punto del motor que ejecuta git
- **CUANDO** se ejecutan `ensure`, `status` y `done` sin `--fetch`
- **ENTONCES** el observador no registra ninguna invocación de `git push`, `git commit`, `git fetch`, `git pull` ni `git ls-remote`

#### Scenario: Una prueba de arquitectura fija el único punto de ejecución de git
- **DADO** el código fuente de los paquetes bajo `internal/branch/**` y de `internal/gitcommon`
- **CUANDO** se ejecuta la prueba de arquitectura
- **ENTONCES** ningún paquete lanza procesos ni abre conexiones de red por otra vía que no sea ese único punto de ejecución de git
- **Y** la prueba falla si se añade una ejecución de git fuera de ese punto

#### Scenario: El carril ODD no exige un cambio SDD
- **DADO** un proyecto sin ningún cambio bajo `openspec/changes/`
- **CUANDO** se ejecuta `axiom branch ensure --lane odd --change mi-feature`
- **ENTONCES** la orden se ejecuta sin requerir `kickoff.yaml` ni `proposal.md`

#### Scenario: El verbo de publicación no pertenece a la familia
- **DADO** el binario `axiom`
- **CUANDO** se comparan `axiom branch --help` y `axiom sdd --help`
- **ENTONCES** `publish` figura solo en la ayuda de `axiom sdd`
- **Y** el puerto de publicación no es importado por `internal/branch/**`

---

### Requirement: `ensure` Planifica de Forma Determinista y es Idempotente (REQ-25.7)

`axiom branch ensure` DEBE producir un plan determinista a partir de la topología, `git:`, `repositories:`, los roles del kickoff y las preferencias personales, y ejecutarlo con git. Una segunda ejecución con los mismos argumentos y el mismo estado DEBE no cambiar nada (ni ramas, ni worktrees, ni registro) y devolver el mismo JSON. El comando DEBE crear solo lo que falta: si la rama ya existe, DEBE reutilizarla sin moverla ni reiniciarla; si no existe en local pero existe su referencia remota local, DEBE adoptarla (REQ-27.15); si ya existe su worktree registrado, DEBE devolver su ruta. Si la rama ya está desplegada en un worktree que el registro local no conoce, DEBE rechazar con `branch_checked_out_elsewhere` nombrando la ruta. Cuando todavía no hay kickoff sellado (el cambio se nombra antes del sellado), `ensure` DEBE planificar solo para el repositorio del planning home la rama `<prefijo>/<cambio>`, que es idéntica en ambos modelos de REQ-25.4, y una ejecución posterior, ya con el kickoff sellado, DEBE completar de forma incremental las ramas de rol y los repositorios de código que correspondan sin alterar lo ya creado (supuesto S-1). `ensure --change <c> --archive` (REQ-25.45) es un plan aparte con las mismas garantías de determinismo e idempotencia: no altera las ramas del cambio ya creadas.
(Previously: sin la adopción de la rama publicada cuando no existe en local pero sí su referencia remota local)

#### Scenario: Segunda ejecución sin cambios
- **DADO** un cambio ya preparado por `ensure`
- **CUANDO** se ejecuta de nuevo `ensure --change x --json` con los mismos argumentos
- **ENTONCES** el JSON es idéntico al de la primera ejecución
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
- **DADO** un cambio nombrado sin `kickoff.yaml`, en un monorepo
- **CUANDO** se ejecuta `ensure --change x` antes del sellado y de nuevo después de sellar dos roles `core` y `qa`
- **ENTONCES** la primera ejecución crea solo `feat/x`
- **Y** la segunda conserva `feat/x` y crea `feat/x-core` y `feat/x-qa` desde ella

#### Scenario: Se rechaza un nombre de cambio que no es válido
- **DADO** un nombre de cambio con caracteres inválidos como componente de rama git
- **CUANDO** se ejecuta `ensure --change "a b:c"`
- **ENTONCES** se rechaza con `invalid_name`
- **Y** no se crea nada

#### Scenario: `--archive` no altera las ramas del cambio
- **DADO** un cambio con las ramas `feat/x` y `feat/x-core` ya creadas y su PR fusionado
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se crea `feat/x-archive` y el commit al que apunta cada una de las ramas anteriores no cambia

#### Scenario: La rama publicada se adopta en lugar de crearse desde la base
- **DADO** una rama `feat/x` inexistente en local y `refs/remotes/origin/feat/x` presente
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** `feat/x` se crea desde `origin/feat/x`
- **Y** una segunda ejecución con los mismos argumentos devuelve el mismo JSON sin cambiar nada

---

### Requirement: Base sin Red y `--fetch` Explícito (REQ-25.10)

`ensure` NO DEBE realizar ninguna operación remota por defecto. La base de cada rama DEBE ser `origin/<integración>` si esa referencia existe en local y, si no, la rama de integración local; si la propia rama del cambio no existe en local pero existe su referencia remota local, la rama se adopta desde ella (REQ-27.15). El JSON DEBE devolver, para cada repositorio, `base.ref`, `base.sha`, `base.source` (que indica si la base de integración procede de la referencia remota local o de la rama local; la rama del cambio adoptada desde el remoto se indica aparte, en `branches[].createdFrom: remote` y `createdFromRef`, REQ-27.15) y `fetched: false`. Solo con `--fetch` el comando PUEDE ejecutar un `fetch` previo, y entonces `fetched` DEBE ser `true`. Si `--fetch` falla, `ensure` DEBE terminar con `fetch_failed` sin crear nada (supuesto S-8). Si no existe la rama de integración ni local ni como referencia remota local, DEBE rechazar con `base_not_found`, nombrando la rama esperada. `--base <ref>` sustituye la base (REQ-25.12). El `--fetch` sigue exigiendo la autorización de la persona.
(Previously: la rama del cambio nunca se adoptaba desde el remoto y el JSON no indicaba el origen de cada rama; `base.source` no cambia)

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

---

### Requirement: Fusión Intermedia de un Cambio no es Integración ni Pendiente de Archivar (REQ-26.11)

Una **fusión intermedia** es el estado de un cambio cuya carpeta `changes/<cambio>/` está presente en el árbol de la rama de integración porque se fusionó una PR que llevaba una aprobación con cesión de una parada (REQ-26.5 de `sdd-block-review-gates`) y el trabajo del cambio aún no está completo. El criterio observable es: el registro de puertas integrado (`gates.yaml`) contiene al menos una compuerta aprobada con la marca de cesión y NO están aprobadas todas las compuertas `role-apply:<rol>` del roster sellado (supuesto A-7). Una rama del cambio **publicada** en `origin` (REQ-27.7 de `sdd-docs-publication`) que aún no se ha fusionado NO es una fusión intermedia ni una rama integrada: la evaluación sigue haciéndose únicamente sobre la referencia de integración local de REQ-25.10. Un cambio en fusión intermedia:

- NO DEBE figurar entre los cambios pendientes de archivar de `axiom branch status` ni del dashboard (REQ-25.20), ni ofrecer la continuación de archivado;
- NO DEBE ser aceptado por `ensure --archive` como base que contiene el cambio fusionado (REQ-25.45), que rechaza con `change_not_integrated`;
- NO DEBE bloquear ninguna orden ni guarda, y la evaluación sigue haciéndose sobre la referencia de integración local de REQ-25.10, sin cambiar de rama y sin operaciones remotas;
- PUEDE mostrarse como «en curso (fusión intermedia)», sin código de fallo y sin continuación de archivado.

Si el registro de puertas o el kickoff integrados no pueden leerse, el cambio DEBE tratarse como hasta ahora (supuesto A-8), sin error. Un cambio fusionado sin ninguna marca de cesión NO es una fusión intermedia.
(Previously: la fusión intermedia se describía como resultado de fusionar «una PR de la salida 2» y no aclaraba que publicar la rama del cambio no la integra)

#### Scenario: Un cambio con solo spec cedida y fusionada no figura como pendiente de archivar

- **DADO** un monorepo cuya referencia de integración contiene `openspec/changes/x/` con un `gates.yaml` en el que `spec` está aprobada con la marca de cesión y sin ningún `role-apply` aprobado
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** `x` no figura entre los cambios pendientes de archivar
- **Y** el código de salida es `0`

#### Scenario: Un cambio con design cedido y roles sin aprobar sigue siendo intermedio

- **DADO** la referencia de integración con el cambio `x`, con `spec` aprobada con la salida 3, `design` aprobada con la marca de cesión y los roles `core` y `web` sin `role-apply` aprobado
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** `x` no se indica como pendiente de archivar y no se ofrece la continuación `axiom branch ensure --change x --archive`

#### Scenario: Un rol cedido con otro rol pendiente sigue siendo intermedio

- **DADO** la referencia de integración con el cambio `x`, con `role-apply:core` aprobado con la marca de cesión y `role-apply:web` sin aprobar
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** `x` no figura entre los pendientes de archivar

#### Scenario: Al aprobarse todos los role-apply el cambio deja de ser intermedio

- **DADO** la referencia de integración con el cambio `x` cuyo `gates.yaml` conserva la marca de cesión de `design` y donde todos los `role-apply:<rol>` del roster están aprobados
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** `x` figura entre los cambios pendientes de archivar con la continuación `axiom branch ensure --change x --archive`

#### Scenario: Un cambio fusionado sin ninguna marca de cesión se comporta como hasta ahora

- **DADO** la referencia de integración con el cambio `y` cuyo `gates.yaml` no contiene ninguna marca de cesión (o que no tiene `gates.yaml`)
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** `y` figura entre los cambios pendientes de archivar, como antes de este cambio

#### Scenario: Un registro integrado ilegible se trata como hasta ahora

- **DADO** la referencia de integración con el cambio `z` cuyo `gates.yaml` o `kickoff.yaml` no se puede leer
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** el código de salida es `0` y `z` se trata como hasta ahora, sin error

#### Scenario: La fusión intermedia no bloquea nada

- **DADO** un repositorio con un cambio en fusión intermedia
- **CUANDO** se ejecutan `axiom branch status`, `axiom sdd status` y un escritor guardado en la rama correcta de otro cambio
- **ENTONCES** todas terminan con normalidad y ninguna se bloquea por la fusión intermedia

#### Scenario: Quien recoge la cesión crea su rama desde la base fusionada

- **DADO** un cambio en fusión intermedia fusionado en la integración
- **CUANDO** otra persona ejecuta `axiom branch ensure --change x`
- **ENTONCES** `ensure` crea la rama del cambio desde la base y la carpeta `changes/x/` con los artefactos aprobados está disponible
- **Y** no se ejecuta ninguna operación remota

#### Scenario: Una rama publicada sin fusionar no es una fusión intermedia

- **DADO** un cambio `x` cuya rama `inc/x` está publicada en `origin` con una compuerta aprobada con cesión, y cuya carpeta `changes/x/` no está en la referencia de integración local
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** `x` no se clasifica como fusión intermedia ni como pendiente de archivar
- **Y** el código de salida es `0` y no se ejecuta ninguna operación remota
