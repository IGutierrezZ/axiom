# Delta para sdd-block-review-gates

> Parte del estado vigente: el resto de requisitos NO cambia. Esta delta añade la emisión de eventos de integración al aprobar compuertas (C-07-b, capacidad `integration-plugins`). Es solo ADDED. El contrato `axiom.sdd-gate-record/v1` y su salida NO cambian.

## ADDED Requirements

### Requirement: Aprobar `spec`, `design` o `role-apply` Emite su Evento de Integración (REQ-43.22)

`axiom sdd gate record` con `--decision approved` DEBE despachar un evento de integración (REQ-43.6) DESPUÉS de persistir la decisión en `gates.yaml` y antes de cualquier código que pueda devolver error: `spec` emite `gate.spec.approved`, `design` emite `gate.design.approved` y `role-apply:<rol>` emite `role.apply.approved` con el rol extraído del sufijo. El evento DEBE llevar el cambio, el actor registrado (REQ-42.1) y la clave de idempotencia de REQ-43.7. La emisión NUNCA DEBE hacer fallar el verbo ni cambiar su código de salida ni su salida (REQ-43.10). Una decisión `rejected` NO DEBE emitir nada.

(Placement: after REQ-28.10)

#### Scenario: Aprobar `spec`
- **DADO** el doble configurado y un cambio con kickoff sellado
- **CUANDO** se ejecuta `gate record --gate spec --decision approved`
- **ENTONCES** `gates.yaml` contiene la aprobación
- **Y** el doble recibe `gate.spec.approved` con el actor registrado y su clave de idempotencia

#### Scenario: Aprobar `design`
- **DADO** el doble configurado
- **CUANDO** se aprueba `design`
- **ENTONCES** el doble recibe `gate.design.approved`

#### Scenario: Aprobar `role-apply` de un rol
- **DADO** el doble configurado y el rol `backend` sellado
- **CUANDO** se ejecuta `gate record --gate role-apply:backend --decision approved`
- **ENTONCES** el doble recibe `role.apply.approved` con el rol `backend`

#### Scenario: Rechazar no emite
- **DADO** el doble configurado
- **CUANDO** se ejecuta `gate record --gate spec --decision rejected --reason x`
- **ENTONCES** el doble no recibe ningún evento y no se crea ningún fichero de integración

#### Scenario: El fallo de la integración no rompe la aprobación
- **DADO** el doble configurado para fallar
- **CUANDO** se aprueba `spec`
- **ENTONCES** el verbo termina con código 0 y la salida de siempre
- **Y** la aprobación consta en `gates.yaml` y la operación queda pendiente

#### Scenario: Reaprobar un artefacto cambiado
- **DADO** `spec` aprobada y ya entregada, y después un artefacto modificado
- **CUANDO** se aprueba `spec` de nuevo
- **ENTONCES** el doble recibe un evento con una clave de idempotencia distinta de la primera

### Requirement: `tasks` e `integration` no Emiten y el Contrato `gate-record/v1` no Cambia (REQ-43.23)

La aprobación de las compuertas `tasks` e `integration` NO DEBE despachar ningún evento. El contrato `axiom.sdd-gate-record/v1` (campos de la salida JSON y de texto) NO DEBE cambiar por la integración: los avisos van únicamente a stderr.

(Placement: after REQ-43.22)

#### Scenario: `tasks` no emite
- **DADO** el doble configurado
- **CUANDO** se aprueba `tasks`
- **ENTONCES** el doble no recibe ningún evento

#### Scenario: `integration` no emite
- **DADO** el doble configurado
- **CUANDO** se aprueba `integration`
- **ENTONCES** el doble no recibe ningún evento

#### Scenario: Salida JSON idéntica con y sin integración
- **DADO** la misma aprobación ejecutada con y sin plugin configurado
- **CUANDO** se ejecuta con `--json`
- **ENTONCES** el JSON de stdout es idéntico, con el mismo `schema` `axiom.sdd-gate-record/v1`
