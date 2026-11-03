# Delta para Ciclo de Vida de Ramas y Worktrees por Cambio (paradas-tres-salidas)

> **Cambio:** `paradas-tres-salidas` · **Tipo:** delta sobre la capacidad viva `change-branch-lifecycle`
> **Idioma:** Español (castellano peninsular); identificadores en inglés.

Cambios de este delta:

- REQ-26.11 (nuevo): define la **fusión intermedia** (un cambio cuya carpeta se ha fusionado en la rama de integración por la salida 2 de una parada, con trabajo aún por hacer) y fija su comportamiento observable.
- REQ-25.20 (enmienda): una fusión intermedia NO figura como cambio pendiente de archivar.
- REQ-25.45 (enmienda): `ensure --archive` NO acepta una fusión intermedia como base que «contiene el cambio fusionado» y rechaza con `change_not_integrated`.
- El criterio exacto con el que se reconoce la fusión intermedia (qué leer del árbol de la integración) lo fija `sdd-design`; los escenarios de REQ-26.11 fijan el comportamiento que debe cumplir.

Notas no normativas (un delta no puede editar estas partes de la spec viva; deben actualizarse al archivar o en apply):

- El término «Cambio pendiente de archivar» del glosario («ya fusionado y aún sin archivar») y la fila del índice de requisitos de REQ-25.20 deben reflejar la exclusión de las fusiones intermedias.
- El supuesto **S-16** («una base "contiene el cambio fusionado" cuando su árbol incluye `changes/<cambio>/` del planning home») queda enmendado por REQ-25.45 y REQ-26.11: además de incluir la carpeta, el cambio no debe ser una fusión intermedia.
- Supuesto **A-7:** un cambio con todos los `role-apply` aprobados deja de ser fusión intermedia aunque la última puerta lleve marca de cesión (la fase `verify` cedida no hace intermedia la fusión del trabajo ya implementado).
- Supuesto **A-8:** un cambio fusionado sin ninguna marca de cesión en su registro de puertas (incluidos los cambios preexistentes y los sellados en modalidad continua) se comporta exactamente como hasta ahora.

## ADDED Requirements

### Requirement: Fusión Intermedia de un Cambio no es Integración ni Pendiente de Archivar (REQ-26.11)

Una **fusión intermedia** es el estado de un cambio cuya carpeta `changes/<cambio>/` está presente en el árbol de la rama de integración porque se fusionó una PR de la salida 2 de una parada (REQ-26.5 de `sdd-block-review-gates`) y el trabajo del cambio aún no está completo. El criterio observable es: el registro de puertas integrado (`gates.yaml`) contiene al menos una compuerta aprobada con la marca de cesión y NO están aprobadas todas las compuertas `role-apply:<rol>` del roster sellado (supuesto A-7). Un cambio en fusión intermedia:

- NO DEBE figurar entre los cambios pendientes de archivar de `axiom branch status` ni del dashboard (REQ-25.20), ni ofrecer la continuación de archivado;
- NO DEBE ser aceptado por `ensure --archive` como base que contiene el cambio fusionado (REQ-25.45), que rechaza con `change_not_integrated`;
- NO DEBE bloquear ninguna orden ni guarda, y la evaluación sigue haciéndose sobre la referencia de integración local de REQ-25.10, sin cambiar de rama y sin operaciones remotas;
- PUEDE mostrarse como «en curso (fusión intermedia)», sin código de fallo y sin continuación de archivado.

Si el registro de puertas o el kickoff integrados no pueden leerse, el cambio DEBE tratarse como hasta ahora (supuesto A-8), sin error. Un cambio fusionado sin ninguna marca de cesión NO es una fusión intermedia.

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

## MODIFIED Requirements

### Requirement: Cambios Pendientes de Archivar Visibles en `status` y en el Dashboard (REQ-25.20)

Los cambios pendientes de archivar de un repositorio con rama de cambio, es decir, los cambios activos presentes en el árbol de la rama de integración (directorios bajo `changes/` del planning home distintos de `archive/` y de los sumideros `e2e-cumulative` y `cumulative-<rol>`, excluidos por nombre) que no están en fusión intermedia (REQ-26.11), DEBEN mostrarse en `axiom branch status` (REQ-25.15) y en el dashboard (REQ-25.34 de `local-web-dashboard`). Es información, no una comprobación: NO existe ningún código de fallo por su presencia, NO hay integración con la CI y NO DEBE bloquear ninguna orden ni guarda. La evaluación DEBE hacerse sobre la referencia de integración local de REQ-25.10 (`origin/<integración>` si existe en local y, si no, la rama local), sin cambiar de rama y sin operaciones remotas; si esa referencia no puede resolverse, la lista se omite con un aviso y sin error. Solo los directorios cuentan como cambios; los ficheros sueltos de `changes/` no lo son. En el repositorio de specs de un multirepo no aplica: allí los cambios activos viven en la rama de integración de forma normal. Cada cambio pendiente DEBE indicar la continuación para archivarlo (`axiom branch ensure --change <c> --archive`, REQ-25.45).
(Previously: todo cambio activo presente en el árbol de la rama de integración contaba como pendiente de archivar; ahora se excluye el que está en fusión intermedia (REQ-26.11))

#### Scenario: Un cambio fusionado y sin archivar figura como pendiente
- **DADO** un repositorio con rama de cambio cuya referencia de integración local contiene `openspec/changes/inc-99-ejemplo/`
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** el JSON lista `inc-99-ejemplo` entre los cambios pendientes de archivar con la continuación `axiom branch ensure --change inc-99-ejemplo --archive`
- **Y** el código de salida es `0`

#### Scenario: Solo `archive/` no genera pendientes
- **DADO** la referencia de integración con `openspec/changes/` que contiene únicamente `archive/`
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** la lista de pendientes de archivar está vacía

#### Scenario: Los sumideros exentos no son pendientes
- **DADO** la referencia de integración con `openspec/changes/` que contiene `archive/`, `e2e-cumulative` y `cumulative-core`
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** la lista de pendientes de archivar está vacía

#### Scenario: Un fichero suelto no cuenta como cambio
- **DADO** la referencia de integración con `openspec/changes/` que contiene `archive/` y un fichero `.gitkeep`
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** la lista de pendientes de archivar está vacía

#### Scenario: Los pendientes no producen código de fallo ni bloquean
- **DADO** un repositorio con dos cambios pendientes de archivar
- **CUANDO** se ejecutan `axiom branch status`, `axiom sdd status` y un escritor guardado en la rama correcta de otro cambio
- **ENTONCES** todas terminan con normalidad y ninguna se bloquea por la presencia de los pendientes

#### Scenario: `status --change` indica que el cambio solicitado está pendiente
- **DADO** el cambio `x` fusionado y presente en la referencia de integración
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON indica que `x` está pendiente de archivar y la continuación `axiom branch ensure --change x --archive`

#### Scenario: El repositorio de specs de un multirepo no tiene pendientes
- **DADO** un multirepo con el repositorio de specs en `main` y un cambio activo en `openspec/changes/`
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** la lista de pendientes de archivar está vacía

#### Scenario: Sin referencia de integración la lista se omite con aviso
- **DADO** un repositorio sin la rama de integración ni local ni como referencia remota local
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** el código de salida es `0` y la lista de pendientes se omite con un aviso

#### Scenario: La evaluación no usa la red
- **DADO** un observador instalado en el único punto del motor que ejecuta git
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** el observador no registra ninguna operación remota

#### Scenario: Una fusión intermedia no figura como pendiente
- **DADO** la referencia de integración que contiene `openspec/changes/x/` en fusión intermedia (REQ-26.11) y `openspec/changes/y/` fusionado sin marca de cesión
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** la lista de pendientes de archivar contiene `y` y no contiene `x`
- **Y** el código de salida es `0`

---

### Requirement: Rama de Archivado y `ensure --archive` (REQ-25.45)

En los repositorios con rama de cambio (`monorepo-embedded`), el archivado de un cambio DEBE hacerse en una rama propia, la rama de archivado `<prefijo>/<cambio>-archive`, y entregarse en un PR documental aparte, **después** de fusionar el PR del cambio (REQ-25.39). `axiom branch ensure --change <c> --archive` DEBE crearla en el repositorio del planning home, saliendo de la rama de integración con la base de REQ-25.10 (`origin/<integración>` si existe en local y, si no, la rama de integración local), y aplicando el modo de trabajo personal de REQ-25.8 (en modo `worktree`, un worktree en `<raíz de worktrees>/<cambio>-archive/<repo>`; en `in_place`, el checkout de la rama en el propio repositorio, sujeto a REQ-25.9). `--archive` admite `--prefix` (REQ-25.12).

Si la base resuelta no contiene ya el cambio fusionado, es decir, si su árbol no incluye la carpeta `changes/<cambio>/` del planning home, o si la incluye pero el cambio está en fusión intermedia (REQ-26.11), `ensure --archive` DEBE rechazar con `change_not_integrated`, sin crear nada, nombrando la base evaluada y sugiriendo `--fetch` cuando la carpeta falta; cuando el rechazo se debe a una fusión intermedia, el mensaje DEBE indicarlo y NO DEBE sugerir `--fetch` como remedio, porque la base ya contiene la versión integrada. NO hay `fetch` implícito: solo con `--fetch` explícito se actualiza la referencia remota local antes de evaluar (REQ-25.10). `ensure --archive` es idempotente: reutiliza la rama y el worktree existentes sin moverlos, y NO publica la rama. En `multirepo` NO existe rama de archivado: `--archive` DEBE rechazarse con `archive_branch_not_applicable` porque el archive se hace in situ en la rama de integración del repositorio de specs (REQ-25.5); tampoco existe en el carril ODD (`--lane odd`), que no tiene archive. `done` también cubre la rama de archivado (REQ-25.16 y REQ-25.17) y `status` la incluye en el plan (REQ-25.15).
(Previously: bastaba con que el árbol de la base incluyera `changes/<cambio>/` para considerar el cambio fusionado, aunque fuera una fusión intermedia de planificación)

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
