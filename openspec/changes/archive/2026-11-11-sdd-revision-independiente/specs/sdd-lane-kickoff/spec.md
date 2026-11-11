# Delta para `sdd-lane-kickoff`

## MODIFIED Requirements

### Requirement: Idempotencia del Kickoff ante un Cambio con Artefactos o Configuración Ya Sellada (REQ-21.4)

El orquestador NO DEBE volver a formular la pregunta de selección de carril (REQ-21.1) ni el cuestionario de pre-vuelo de SDD (capacidad `sdd-preflight-configuration`) para un cambio que ya tiene al menos un artefacto persistido (`proposal.md`, un `spec.md` bajo `specs/`, `design.md` o `tasks.md`) o cuya configuración de kickoff ya está sellada. En ese caso, el orquestador DEBE continuar respetando el carril y la configuración ya selladas en el primer kickoff, sin interrogar de nuevo al usuario. El retro-sello (`axiom sdd kickoff seal --infer`, `sealed_by: inferred`) está **exento** de la regla «`continuous` solo con el rol `fullstack`» (REQ-28.12 de `sdd-preflight-configuration`): conserva `continuous` con los roles que la detección de roles ya vigente infiere, aunque no sean `fullstack`, y esa regla solo se aplica a los sellados explícitos.

(Previamente: esta especificación indicaba que, al retro-sellar un cambio preexistente sin kickoff sellado, el rol por defecto congelado era el rol único `fullstack`. Se corrige tras la pregunta abierta O-1 planteada en `sdd-design` y resuelta explícitamente por el usuario con evidencia de código —`internal/multirole/detector.go:35-55` cae siempre a `core`, nunca a `fullstack`; `internal/multirole/detector.go:58-64` rechaza cualquier rol ausente de `axiom.yaml`; y `axiom.yaml:7-28` no declara `fullstack`—: el retro-sello DEBE congelar el rol que la detección de roles ya vigente infiere para ese cambio, nunca introducir la identidad `fullstack`.)
(Previously: la configuración sellada incluía la política de relevos y el retro-sello inferido nombraba «sin `handoff.md` intermedios»; la política de relevos ya no forma parte de la configuración sellada)
(Previously: no decía nada sobre la regla «`continuous` solo con `fullstack`»; ahora declara expresamente que el retro-sello queda exento de ella y conserva `continuous` con los roles detectados)

#### Scenario: Reanudación de un cambio con kickoff ya sellado

- **DADO** un cambio SDD activo con `proposal.md` existente y su configuración de kickoff ya sellada (carril `sdd` y modalidad definida)
- **CUANDO** se reanuda el trabajo sobre ese cambio en una sesión posterior
- **ENTONCES** el orquestador no repite la pregunta de selección de carril ni el cuestionario de pre-vuelo
- **Y** continúa el flujo respetando la modalidad ya sellada

#### Scenario: Reanudación de un cambio sellado con la política de relevos antigua

- **DADO** un cambio SDD activo cuyo `kickoff.yaml` conserva `handoff_policy: per_checkpoint` o `none` de antes de este cambio
- **CUANDO** se reanuda el trabajo sobre ese cambio
- **ENTONCES** el orquestador no pregunta nada sobre la política de relevos
- **Y** el valor antiguo no altera el flujo ni se vuelve a escribir

#### Scenario: Cambio preexistente con artefactos pero sin kickoff sellado infiere un valor por defecto sin bloquear

- **DADO** un cambio SDD activo con artefactos existentes (por ejemplo `proposal.md` y `design.md`) pero sin ninguna configuración de kickoff sellada previamente
- **CUANDO** el orquestador retoma ese cambio
- **ENTONCES** infiere y sella la configuración por defecto documentada (modalidad continua y el rol que la detección de roles ya vigente infiere para ese cambio —hoy, `core`, cuando existe un `design.md` sin roles explícitos—) sin bloquear el trabajo ni interrogar retroactivamente al usuario
- **Y** el retro-sello no introduce la identidad `fullstack` ni ningún rol nuevo: es conservador por definición y se limita a congelar el rol y el comportamiento que la detección de roles ya aplica hoy para ese cambio, sin redefinirlo
- **Y** el retro-sello no infiere ninguna política de relevos: escribe `handoff_policy: none` solo como valor de compatibilidad (REQ-26.9 de `sdd-preflight-configuration`)
- **Y** dicha configuración inferida queda registrada para no volver a inferirse en la siguiente reanudación

#### Scenario: El retro-sello conserva continuous con roles distintos de fullstack

- **DADO** un cambio preexistente sin kickoff cuya detección de roles infiere `core`
- **CUANDO** se ejecuta el retro-sello
- **ENTONCES** el sellado se completa con `execution_style: continuous`, `sealed_by: inferred` y el rol `core`
- **Y** no se rechaza por la regla de REQ-28.12 ni se infiere `checkpointed`
- **Y** no se abre ninguna puerta `pending` para artefactos ya escritos

#### Scenario: El retro-sello de varios roles detectados también conserva continuous

- **DADO** un cambio preexistente sin kickoff cuyo `design.md` declara los roles `core` y `web`
- **CUANDO** se ejecuta el retro-sello
- **ENTONCES** se sella `continuous` con ambos roles detectados
- **Y** `axiom sdd status` lo trata como un cambio sin paradas, igual que antes de este cambio

#### Scenario: La regla de sellado sigue aplicándose al sellado explícito del mismo cambio

- **DADO** un cambio con artefactos pero sin kickoff, que el usuario sella de forma explícita (sin `--infer`) en `continuous` con los roles `core` y `web`
- **CUANDO** se ejecuta el sellado
- **ENTONCES** se rechaza conforme a REQ-28.12
- **Y** la exención del retro-sello no se extiende a esa ruta

#### Scenario: Un retro-sello sin ningún rol detectable conserva su comportamiento

- **DADO** un cambio preexistente cuya detección de roles no puede determinar ningún rol y devuelve un error
- **CUANDO** se ejecuta el retro-sello
- **ENTONCES** se comporta exactamente como antes de este cambio, sin tratarlo como un caso de la regla de REQ-28.12

#### Scenario: El retro-sello no altera qué `tasks.<rol>.md` consulta la barrera

- **DADO** un cambio SDD preexistente sin kickoff sellado, cuya barrera de sincronización ya consulta `tasks.core.md` porque la detección de roles vigente infiere el rol `core` para su `design.md`
- **CUANDO** el orquestador retro-sella la configuración de kickoff de ese cambio conforme al escenario anterior
- **ENTONCES** la barrera de sincronización sigue consultando exactamente `tasks.core.md`, el mismo fichero que consultaba antes del sellado
- **Y** el retro-sello no crea, renombra ni redirige ningún fichero `tasks.<rol>.md`, ni introduce `tasks.fullstack.md` como fuente de verdad nueva para ese cambio
