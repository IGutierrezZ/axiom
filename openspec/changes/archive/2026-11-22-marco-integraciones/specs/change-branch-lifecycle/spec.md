# Delta para change-branch-lifecycle

> Parte del estado vigente: el resto de requisitos NO cambia, incluidos REQ-25.9 y REQ-25.17 (suciedad del árbol). Esta delta añade la bandera opcional `--role` a `axiom branch ensure` y la emisión de `role.start` (C-07-b, capacidad `integration-plugins`). Es solo ADDED. El motor (`internal/branch/engine`) NO cambia de semántica ni conoce el rol: la validación y la emisión viven en el adaptador CLI. Contrato provisional que heredarán C-07-e y C-07-f.

## ADDED Requirements

### Requirement: `ensure --role` Validado contra el Roster Sellado (REQ-43.24)

`axiom branch ensure` DEBE aceptar la bandera opcional `--role <id>`. Solo es válida en el carril `sdd`, es incompatible con `--archive` y DEBE referirse a un rol presente en el kickoff sellado del cambio. En caso contrario DEBE rechazar con un error tipado y código distinto de cero ANTES de mutar nada (ni ramas, ni worktrees, ni registro), con los códigos `plan_conflict` (con `--archive`), `role_not_applicable` (fuera del carril `sdd`), `kickoff_not_sealed` (sin kickoff sellado) y `role_not_in_roster` (rol fuera del roster, con el detalle de los roles sellados); un `--role` en blanco es error de uso (salida 2). Sin `--role`, el comportamiento y el JSON de `ensure` NO DEBEN cambiar. Con `--role`, el plan, las ramas creadas y el JSON del motor NO DEBEN diferir de los de la misma invocación sin la bandera.

(Placement: after REQ-27.15)

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

### Requirement: `ensure --role` Listo Emite `role.start` Idempotente (REQ-43.25)

Tras un `ensure --role <id>` con estado `ready`, el adaptador CLI DEBE despachar el evento `role.start` (REQ-43.6) con la clave de idempotencia exacta `role.start/<cambio>/<rol>` (REQ-43.7). NO DEBE emitir con un estado distinto de `ready`, sin `--role`, ni desde el dashboard (que llama al motor directamente). Un `ensure --role` repetido NO DEBE duplicar la operación mientras está pendiente (deduplicación por plugin y clave); tras una entrega con éxito la operación sale de la cola, así que la repetición vuelve a despachar la misma clave y la idempotencia recae en el plugin (REQ-43.7). La emisión nunca hace fallar el verbo (REQ-43.10).

(Placement: after REQ-43.24)

#### Scenario: Primera entrada al rol
- **DADO** el doble configurado y el rol `backend` sellado
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** el doble recibe `role.start` con la clave `role.start/x/backend`

#### Scenario: `ensure` repetido
- **DADO** `role.start/x/backend` ya entregado
- **CUANDO** se repite `ensure --change x --role backend`
- **ENTONCES** `ensure` no cambia nada (idempotente)
- **Y** el evento se vuelve a despachar con la misma clave `role.start/x/backend`, y el plugin, que es idempotente por clave (entrega al menos una vez, REQ-43.7), no duplica el efecto

#### Scenario: Operación pendiente no se duplica
- **DADO** `role.start/x/backend` pendiente en la outbox por un fallo
- **CUANDO** se repite `ensure --change x --role backend`
- **ENTONCES** la outbox sigue teniendo una sola operación con esa clave

#### Scenario: Sin `--role` no se emite
- **DADO** el doble configurado
- **CUANDO** se ejecuta `ensure --change x` sin `--role`
- **ENTONCES** el doble no recibe ningún evento

#### Scenario: Estado distinto de `ready`
- **DADO** un `ensure --role` que termina en `decision_required`
- **CUANDO** el verbo termina
- **ENTONCES** no se emite ningún evento

#### Scenario: El dashboard no emite
- **DADO** el doble configurado
- **CUANDO** el dashboard da de alta un incremento llamando al motor
- **ENTONCES** el doble no recibe ningún evento
