# Delta para Ciclo de Vida de Ramas del Cambio (change-branch-lifecycle)

> **Cambio:** `eventos-ado` (C-07-e). **Idioma:** castellano de España; identificadores, comandos y claves en inglés.
> La barrera de P8 vive en la validación de `ensure --role` (en el adaptador CLI, antes del motor), no en el motor: `internal/branch/engine` sigue sin conocer el rol ni las integraciones. RF-BRA-34 no cambia: `role.start` se sigue emitiendo con la clave `role.start/<cambio>/<rol>` tras un `ensure --role` en estado `ready`.

## ADDED Requirements

### Requirement: `ensure --role` Registra el Inicio del Rol la Primera Vez que Queda Listo (prov:BRA:1)

Cuando el cambio tiene el vínculo `kind: task` del rol en `integrations.yaml` y un `ensure --role <id>` termina con estado `ready`, el adaptador CLI DEBE registrar `started_at` (instante actual, RFC 3339 en UTC) en ese vínculo SOLO si aún no lo tiene; un `started_at` ya registrado NO DEBE sobrescribirse, de modo que el reloj de la imputación parte de la primera entrada del rol. El registro DEBE hacerse con la fusión de vínculos bajo el mismo cerrojo que las demás escrituras de `integrations.yaml` (RF-INT-09). NO DEBE registrarse con un estado distinto de `ready`, sin `--role`, sin vínculo de Task del rol ni desde el dashboard. Un fallo al escribir `started_at` NO DEBE hacer fallar el verbo: avisa con un aviso tipado y `time propose` lo rechazará después con `role_not_started` (RF-INT-10). Sin plugin activo con el puerto de Tasks, la salida y los ficheros de `ensure --role` NO DEBEN cambiar.

#### Scenario: Primera entrada al rol
- **DADO** el vínculo `kind: task` del rol `fullstack` sin `started_at`
- **CUANDO** se ejecuta `axiom branch ensure --change x --role fullstack` y termina en `ready`
- **ENTONCES** el vínculo guarda `started_at` con el instante actual

#### Scenario: Reentrada al rol
- **DADO** un vínculo con `started_at` igual a `2026-11-23T09:00:00Z`
- **CUANDO** se repite `ensure --change x --role fullstack`
- **ENTONCES** `started_at` sigue siendo `2026-11-23T09:00:00Z`

#### Scenario: Estado distinto de `ready`
- **DADO** un `ensure --role` que termina en `decision_required`
- **CUANDO** el verbo termina
- **ENTONCES** el vínculo no recibe `started_at`

#### Scenario: Sin `--role`
- **DADO** un vínculo `kind: task` sin `started_at`
- **CUANDO** se ejecuta `ensure --change x` sin `--role`
- **ENTONCES** `integrations.yaml` no cambia

#### Scenario: El dashboard no registra el inicio
- **DADO** un vínculo `kind: task` sin `started_at`
- **CUANDO** el dashboard da de alta un incremento llamando al motor
- **ENTONCES** `integrations.yaml` no cambia

#### Scenario: Fallo al escribir el inicio
- **DADO** un `integrations.yaml` que no se puede escribir
- **CUANDO** se ejecuta `ensure --change x --role fullstack`
- **ENTONCES** el verbo termina con el resultado de `ensure` y un aviso tipado
- **Y** `time propose` posterior rechaza con `role_not_started`

#### Scenario: Sin plugin activo
- **DADO** un workspace sin plugin activo con el puerto de Tasks
- **CUANDO** se ejecuta `ensure --change x --role fullstack`
- **ENTONCES** la salida y los ficheros coinciden con los de antes de este cambio

## MODIFIED Requirements

### Requirement: `ensure --role` Validado contra el Roster Sellado (RF-BRA-33)

`axiom branch ensure` DEBE aceptar la bandera opcional `--role <id>`. Solo es válida en el carril `sdd`, es incompatible con `--archive` y DEBE referirse a un rol presente en el kickoff sellado del cambio. En caso contrario DEBE rechazar con un error tipado y código distinto de cero ANTES de mutar nada (ni ramas, ni worktrees, ni registro), con los códigos `plan_conflict` (con `--archive`), `role_not_applicable` (fuera del carril `sdd`), `kickoff_not_sealed` (sin kickoff sellado) y `role_not_in_roster` (rol fuera del roster, con el detalle de los roles sellados); un `--role` en blanco es error de uso (salida 2). Además, cuando el cambio tiene un vínculo `kind: work_item` en `integrations.yaml` y hay activo un plugin con la capacidad `items.create` (RF-INT-02), DEBE rechazar con el código tipado `task_not_bound` si el rol no tiene vínculo `kind: task`, también ANTES de mutar nada y tras las comprobaciones anteriores; la resolución DEBE nombrar `axiom integrations tasks create --change <cambio>` y `axiom integrations tasks bind --change <cambio> --role <rol>=<id>`. Un cambio sin vínculo `kind: work_item`, o sin plugin activo con `items.create`, NO DEBE recibir `task_not_bound`. Sin `--role`, el comportamiento y el JSON de `ensure` NO DEBEN cambiar. Con `--role`, el plan, las ramas creadas y el JSON del motor NO DEBEN diferir de los de la misma invocación sin la bandera.

(Placement: after RF-BRA-27)

(Previously: sin la comprobación `task_not_bound`; el rechazo del rol solo dependía del carril, de `--archive` y del roster sellado)

#### Scenario: Rol sellado
- **DADO** un kickoff sellado con el rol `backend`
- **CUANDO** se ejecuta `axiom branch ensure --change x --role backend`
- **ENTONCES** `ensure` termina con estado `ready` y el mismo JSON que sin `--role`

#### Scenario: Rol ausente del kickoff
- **DADO** un kickoff sellado con el rol `backend`
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** se rechaza con un error tipado que nombra los roles sellados
- **Y** no se crea ninguna rama, worktree ni registro

#### Scenario: Incompatible con `--archive`
- **CUANDO** se ejecuta `ensure --change x --archive --role backend`
- **ENTONCES** se rechaza con un error tipado y no se muta nada

#### Scenario: Fuera del carril `sdd`
- **CUANDO** se ejecuta `ensure --lane odd --change f --role backend`
- **ENTONCES** se rechaza con un error tipado y no se muta nada

#### Scenario: Sin kickoff sellado
- **DADO** un cambio sin `kickoff.yaml`
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** se rechaza con un error tipado que indica que el cambio no tiene roles sellados y no se muta nada

#### Scenario: El motor no conoce el rol
- **DADO** la prueba de arquitectura de `internal/branch/engine`
- **CUANDO** se comprueba `EnsureRequest`
- **ENTONCES** no contiene ningún campo de rol

#### Scenario: Rol sin Task vinculada con tracker activo
- **DADO** un cambio con vínculo `kind: work_item`, un plugin activo con `items.create` y el rol `backend` sin vínculo `kind: task`
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** se rechaza con `task_not_bound` y código distinto de cero
- **Y** la resolución nombra `axiom integrations tasks create --change x` y `axiom integrations tasks bind --change x --role backend=<id>`
- **Y** no se crea ninguna rama, worktree ni registro y el doble no recibe ningún evento

#### Scenario: Con la Task vinculada la barrera se levanta
- **DADO** el rol `backend` con vínculo `kind: task`
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** `ensure` termina con estado `ready` y el mismo JSON que sin `--role`

#### Scenario: El rechazo de roster precede a la barrera
- **DADO** un rol fuera del roster sellado en un cambio con tracker activo
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** el código del rechazo es `role_not_in_roster`, no `task_not_bound`

#### Scenario: Cambio sin vínculo de WI
- **DADO** un cambio sin vínculo `kind: work_item` y un plugin activo
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** no se rechaza por `task_not_bound`

#### Scenario: Sin plugin activo con `items.create`
- **DADO** un cambio con vínculo `kind: work_item` y sin plugin activo (por ejemplo con `integrations.azure-devops.enabled: false`)
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** la salida y el JSON coinciden con los de antes de este cambio y no se rechaza por `task_not_bound`
