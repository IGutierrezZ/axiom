# Delta para sdd-role-closure-handoff

> **Supuesto visible para la compuerta `spec` (Q13):** con REQ-21.15 modificado, la fase `verify` global puede abrirse antes de que cierre el último rol activo (un rol `deferred` u `optional` puede seguir pendiente). La resolución que propone esta delta para el aviso de REQ-21.13 es la recomendada por defecto: el aviso pasa a emitirse al aprobarse el `role-apply` del último rol `blocking` pendiente, que es el que habilita la verificación global, e informa de los roles `deferred` y `optional` que sigan pendientes; el cierre posterior de un rol no bloqueante no repite el aviso. Con todos los roles `blocking`, y con el rol único `fullstack`, el aviso no cambia.

## MODIFIED Requirements

### Requirement: Aviso Formal de Conclusión del Último Rol Activo (REQ-21.13)

Al aprobarse la compuerta de `apply` (REQ-21.11) del último rol pendiente entre los roles con política `blocking` del cambio —incluido el caso de un único rol `fullstack`, y tanto con la salida 2 como con la salida 3 de REQ-26.1— el orquestador DEBE notificar formalmente al usuario que ha concluido la implementación del último rol bloqueante y que queda abierta la verificación global de la solución (REQ-21.15). Si en ese momento algún rol con política `deferred` u `optional` sigue con su `role-apply` sin aprobar, el aviso DEBE nombrarlo (rol, política y estado de su `role-apply`) e indicar que no bloquea la verificación global. El cierre posterior de un rol `deferred` u `optional` NO DEBE repetir el aviso. El aviso NO DEBE anunciar la generación de ningún relevo y NO DEBE escribir ningún fichero `handoff.md`. Con todos los roles `blocking`, el último rol pendiente entre los bloqueantes es el último rol activo del cambio y el aviso es el mismo que antes de esta modificación.
(Previously: el aviso anunciaba que se generaba el relevo de integración para la verificación global)
(Previously: el aviso se emitía al aprobarse el `role-apply` del último rol pendiente entre todos los roles activos, sin distinguir su política; con REQ-21.15 modificado, la verificación global se abre al cerrar el último rol `blocking`, de modo que el aviso se adelanta a ese momento)

#### Scenario: Último de varios roles activa el aviso sin relevo

- **DADO** un cambio con los roles `core`, `web` y `qa`, todos con política `blocking`, donde `core` y `web` ya están concluidos y `qa` es el único rol pendiente
- **CUANDO** se aprueba la compuerta de `apply` del rol `qa`
- **ENTONCES** el orquestador emite el aviso formal de conclusión del último rol
- **Y** no escribe ningún `handoff.md` en la carpeta del cambio

#### Scenario: Rol único fullstack satisface la condición de último rol de inmediato

- **DADO** un cambio con el rol único `fullstack`
- **CUANDO** se aprueba la compuerta de `apply` de ese único rol
- **ENTONCES** esa misma aprobación satisface la condición de "último rol activo"
- **Y** el orquestador emite el aviso formal inmediatamente, sin esperar ningún otro rol y sin escribir `handoff.md`

#### Scenario: Compuerta rechazada no dispara el cierre aunque sea el único rol pendiente

- **DADO** un cambio en el que el único rol `blocking` pendiente tiene su compuerta de `apply` rechazada (REQ-21.12)
- **CUANDO** se registra el rechazo
- **ENTONCES** el orquestador no emite el aviso de último rol
- **Y** espera a que esa compuerta se apruebe explícitamente antes de reevaluar la condición de cierre

#### Scenario: Aprobar el último rol con la salida 2 emite el aviso y cede verify

- **DADO** un cambio con el rol único `fullstack` y su trabajo confirmado en la rama
- **CUANDO** se aprueba la compuerta `role-apply:fullstack` con la salida 2
- **ENTONCES** se emite el aviso de último rol
- **Y** la fase `verify` queda cedida a otra persona (REQ-26.7)
- **Y** no se escribe ningún `handoff.md`

#### Scenario: El aviso se emite al cerrar el último rol blocking aunque un rol deferred siga pendiente

- **DADO** un cambio con el rol `core` (`blocking`) pendiente y el rol `e2e` (`deferred`) con su `role-apply` sin aprobar
- **CUANDO** se aprueba la compuerta de `apply` del rol `core`
- **ENTONCES** el orquestador emite el aviso formal de conclusión del último rol bloqueante y de verificación global abierta
- **Y** el aviso nombra a `e2e` con su política `deferred` y su `role-apply` pendiente, e indica que no bloquea la verificación global

#### Scenario: Un rol optional pendiente se informa en el aviso

- **DADO** un cambio con el rol `core` (`blocking`) pendiente y el rol `docs` (`optional`) sin ninguna decisión registrada
- **CUANDO** se aprueba la compuerta de `apply` del rol `core`
- **ENTONCES** el aviso nombra a `docs` como rol no bloqueante pendiente

#### Scenario: Aprobar un rol blocking cuando otro rol blocking sigue pendiente no emite el aviso

- **DADO** un cambio con los roles `core` y `web` (ambos `blocking`) y `e2e` (`deferred`), con `web` sin aprobar
- **CUANDO** se aprueba la compuerta de `apply` del rol `core`
- **ENTONCES** el orquestador no emite el aviso de último rol

#### Scenario: El cierre posterior de un rol deferred no repite el aviso

- **DADO** el cambio del escenario anterior con `core` y `web` aprobados, el aviso ya emitido y `e2e` (`deferred`) pendiente
- **CUANDO** se aprueba después la compuerta de `apply` del rol `e2e`
- **ENTONCES** el orquestador no emite un segundo aviso de verificación global abierta
- **Y** no escribe ningún `handoff.md`

---

### Requirement: Ejecución de la Fase `verify` Global tras Aprobarse Todos los `role-apply` (REQ-21.15)

En un cambio con más de un rol y con compuertas (modalidad "con paradas"), la fase `verify` global DEBE habilitarse cuando están aprobadas las compuertas `role-apply:<rol>` de todos los roles del roster sellado con política `blocking`, y NO DEBE habilitarse mientras falte alguna de ellas. Los roles con política `deferred` u `optional` NO DEBEN bloquear `verify`, tengan su `role-apply` pendiente, aprobado o rechazado; la dependencia `verify` de `axiom sdd status` DEBE informar de ellos (rol, política y estado de su `role-apply`) sin bloquear. La partición entre roles bloqueantes y no bloqueantes es la misma que usa la barrera de sincronización (REQ-2.1 de `multi-role-fan-out`). No se exige ni se consulta ningún `handoff.md`: un `handoff.md` presente, ausente, `ready`, `blocked` o `needs_clarification` NO DEBE alterar esa condición. Una vez habilitada, la fase `verify` DEBE ejecutar la suite completa de pruebas de la solución (integración, regresión y compuertas de seguridad aplicables) considerando el trabajo consolidado de todos los roles, y DEBE emitir un informe consolidado `verify-report.md`, distinto de los informes individuales `verify-report.<rol>.md` de cada rol. La dependencia `verify` de `axiom sdd status` DEBE reflejar esa condición. Un cambio con un único rol, o en modalidad "continua", NO DEBE verse afectado por este requisito.
(Previously: «Ejecución de la Fase `verify` Global tras el Handoff»: la fase `verify` global exigía un `handoff.md` de integración en `status: ready` y no se iniciaba con `blocked` o `needs_clarification`)
(Previously: la condición exigía aprobado el `role-apply` de todos los roles del roster, sin distinguir su política; un rol `deferred` u `optional` bloqueaba `verify` igual que uno `blocking`)

#### Scenario: Verify global emite un informe consolidado distinto de los informes por rol

- **DADO** un cambio con varios roles, todos con `role-apply` aprobado y verificados individualmente
- **CUANDO** se ejecuta la fase `verify` global
- **ENTONCES** se genera `verify-report.md` consolidado, distinto de los `verify-report.<rol>.md` de cada rol
- **Y** el informe refleja la suite completa de pruebas de la solución, no solo la de un rol

#### Scenario: Verify multirrol bloqueado mientras falte algún role-apply aprobado

- **DADO** un cambio con los roles `core` y `web`, ambos con política `blocking`, donde solo `role-apply:core` está aprobado
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** la dependencia `verify` aparece bloqueada
- **Y** no se enruta hacia `verify`

#### Scenario: Verify multirrol se desbloquea con todos los role-apply aprobados sin handoff.md

- **DADO** un cambio con los roles `core` y `web`, ambos `blocking`, con ambos `role-apply` aprobados y sin `handoff.md` en la carpeta del cambio
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** la dependencia `verify` no está bloqueada
- **Y** `verify` es la fase siguiente proyectada

#### Scenario: Un handoff.md bloqueado ya no impide verify

- **DADO** un cambio multirrol con todos los `role-apply` aprobados y un `handoff.md` con `status: blocked` o `needs_clarification`
- **CUANDO** se evalúa si procede iniciar la fase `verify` global
- **ENTONCES** el estado del `handoff.md` no se consulta y `verify` no queda bloqueado por él

#### Scenario: Un role-apply rechazado mantiene verify bloqueado

- **DADO** un cambio multirrol con todos los `role-apply` aprobados salvo el de un rol `blocking` cuya última decisión es `rejected`
- **CUANDO** se evalúa la dependencia `verify`
- **ENTONCES** está bloqueada mientras ese rol no esté aprobado

#### Scenario: Un rol deferred sin aprobar no bloquea verify y se informa

- **DADO** un cambio con el rol `core` (`blocking`) con `role-apply` aprobado y el rol `e2e` (`deferred`) con `role-apply` sin aprobar
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** la dependencia `verify` no está bloqueada y `verify` es la fase siguiente proyectada
- **Y** la dependencia `verify` informa de `e2e` con su política `deferred` y su `role-apply` pendiente

#### Scenario: Un rol optional sin aprobar no bloquea verify

- **DADO** un cambio con el rol `core` (`blocking`) con `role-apply` aprobado y el rol `docs` (`optional`) sin ninguna decisión registrada
- **CUANDO** se evalúa la dependencia `verify`
- **ENTONCES** no está bloqueada y se informa de `docs` como rol no bloqueante pendiente

#### Scenario: Un role-apply rechazado de un rol deferred no bloquea verify

- **DADO** un cambio con `core` (`blocking`) aprobado y `e2e` (`deferred`) cuya última decisión de `role-apply` es `rejected`
- **CUANDO** se evalúa la dependencia `verify`
- **ENTONCES** no está bloqueada por `e2e`
- **Y** se informa del rechazo de `e2e`

#### Scenario: Un rol blocking pendiente bloquea aunque los no bloqueantes estén aprobados

- **DADO** un cambio con `core` (`blocking`) sin `role-apply` aprobado y `e2e` (`deferred`) aprobado
- **CUANDO** se evalúa la dependencia `verify`
- **ENTONCES** está bloqueada y el motivo nombra a `core`

#### Scenario: La barrera y verify coinciden en qué roles bloquean

- **DADO** un cambio con `core` (`blocking`) aprobado y verificado, y `e2e` (`deferred`) con tareas pendientes
- **CUANDO** se evalúan la barrera de sincronización y la dependencia `verify`
- **ENTONCES** ninguna de las dos queda bloqueada por `e2e`
- **Y** ambas informan de `e2e` como rol no bloqueante pendiente

#### Scenario: Un cambio de un solo rol o continuo no cambia

- **DADO** un cambio con el rol único `fullstack`, o un cambio sellado con modalidad "continua"
- **CUANDO** se evalúa la dependencia `verify`
- **ENTONCES** el resultado es el mismo que antes de este cambio, sin consultar `handoff.md` ni la antigua política de relevos
