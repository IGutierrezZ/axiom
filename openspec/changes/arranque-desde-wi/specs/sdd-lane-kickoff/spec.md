# Delta para Arranque del Carril SDD (sdd-lane-kickoff)

> **Cambio:** `arranque-desde-wi` (C-07-d). **Idioma:** castellano de España; identificadores, comandos y claves en inglés.
> Se añade un requisito en lugar de modificar RF-KOF-05: el orden `ensure` y `seal` de RF-KOF-05 no cambia; este requisito antepone la fase 1 de `axiom change start` y pospone `--adopt` cuando hay plugin activo. Sin plugin activo, nada cambia.

## ADDED Requirements

### Requirement: Con Plugin Activo el WI y la Fase 1 Preceden a `ensure` y a `seal` (prov:74:24)

Con el plugin de work items activo (la respuesta de `axiom change start` no es «no aplica»), el orquestador MUST, tras elegir el carril SDD y antes de nombrar el cambio, ejecutar `axiom change start` para saber que el WI es obligatorio y, si la persona no lo ha dado, MUST pedirlo y no avanzar. Con el WI, MUST ejecutar la fase 1 (`axiom change start --from <id|url>`) antes de `axiom branch ensure` y de `axiom sdd kickoff seal`, MUST usar como nombre del cambio el nombre derivado que devuelve la fase 1 para `ensure --change`, `seal --change` y todo comando posterior, y MUST ejecutar la fase 2 (`axiom change start --adopt --change <cambio>`) después de `seal` y antes de crear `proposal.md` y de lanzar `sdd-explore`, que recibe `context/work-item.md` como contexto. Un rechazo de la fase 1 MUST detener el arranque sin crear rama, worktree, carpeta ni `kickoff.yaml`, y el orquestador MUST relayar su mensaje y su continuación a la persona. Si la persona abandona el flujo tras la fase 1, el orquestador MUST ofrecer `axiom change start --discard --change <cambio>`. Si `axiom branch ensure` falla tras una fase 1 correcta (un resultado `decision_required` que se resuelve con su procedimiento propio, o un error) o si `axiom sdd kickoff seal` falla, el orquestador MUST conservar el staging, MUST NOT ejecutar la fase 2, MUST relayar el mensaje a la persona y MUST ofrecer reintentar con el mismo nombre derivado (sin repetir la fase 1) o abandonar con `axiom change start --discard --change <cambio>`. Un rechazo de `axiom change start --adopt` (`staging_not_found`, `staging_conflict`, `change_not_prepared`, `change_start_busy` o `write_failed`) MUST relayarse con su mensaje y su continuación, MUST detener el arranque antes de crear `proposal.md` y de lanzar `sdd-explore`, y el orquestador MUST NOT ejecutar la continuación por su cuenta si implica una decisión de la persona (como `--discard`); un aviso de la fase 2 (por ejemplo, el enlace de vuelta pendiente) no detiene el arranque y MUST presentarse con su continuación de reintento. Este orden MUST aplicarse también a un proyecto ISO y al perfil de bug. Sin plugin activo, el flujo de RF-KOF-05 MUST ser el actual, sin ninguna llamada a `axiom change start` más allá de la consulta que responde «no aplica».

#### Scenario: Plugin activo y sin WI
- **DADO** un proyecto con el plugin activo y una petición de cambio SDD sin WI
- **CUANDO** el orquestador va a nombrar el cambio
- **ENTONCES** pide el identificador o la URL del WI y no ejecuta `ensure` ni `seal`

#### Scenario: Orden de las fases
- **DADO** un WI válido y el plugin activo
- **CUANDO** el orquestador arranca el cambio
- **ENTONCES** la fase 1 se ejecuta antes de `axiom branch ensure`, `ensure` antes de `seal` y la fase 2 después de `seal`
- **Y** `ensure` y `seal` usan el nombre devuelto por la fase 1

#### Scenario: Rechazo de la fase 1 detiene el arranque
- **DADO** un WI cerrado
- **CUANDO** la fase 1 se rechaza con `work_item_closed`
- **ENTONCES** no se ejecuta `ensure` ni `seal`, no se crea rama, worktree ni carpeta del cambio
- **Y** se presenta a la persona el mensaje y la continuación del rechazo

#### Scenario: Abandono tras la fase 1
- **DADO** una fase 1 correcta y una persona que decide no continuar
- **CUANDO** el orquestador cierra el arranque
- **ENTONCES** ofrece `axiom change start --discard --change <cambio>`

#### Scenario: Falla `ensure` o `seal` tras la fase 1
- **DADO** una fase 1 correcta y un `axiom branch ensure` (o un `axiom sdd kickoff seal`) que falla
- **CUANDO** el orquestador gestiona el fallo
- **ENTONCES** no ejecuta la fase 2, conserva el staging y ofrece reintentar con el mismo nombre derivado o `axiom change start --discard --change <cambio>`

#### Scenario: Rechazo de `--adopt`
- **DADO** una fase 2 que se rechaza con `change_not_prepared`
- **CUANDO** el orquestador la recibe
- **ENTONCES** relaya el mensaje y la continuación `axiom branch ensure --change <cambio> --json`, no crea `proposal.md` y no lanza `sdd-explore`

#### Scenario: Aviso de enlace de vuelta pendiente
- **DADO** una fase 2 que termina con código 0 y el aviso de enlace de vuelta pendiente
- **CUANDO** el orquestador la recibe
- **ENTONCES** presenta el aviso con su reintento y continúa con `proposal.md` y `sdd-explore`

#### Scenario: El explore recibe el contexto
- **DADO** una fase 2 completada
- **CUANDO** el orquestador lanza `sdd-explore`
- **ENTONCES** le indica `context/work-item.md` como dato no fiable de un tercero

#### Scenario: Proyecto ISO y bug
- **DADO** un proyecto ISO con el plugin activo y un WI de tipo Bug
- **CUANDO** el orquestador arranca el cambio
- **ENTONCES** sigue el mismo orden y el perfil resultante es el de bug

#### Scenario: Sin plugin activo no cambia nada
- **DADO** un proyecto sin plugin activo
- **CUANDO** el orquestador arranca el cambio
- **ENTONCES** el flujo es el de RF-KOF-05 y no ejecuta ninguna fase de `axiom change start`
