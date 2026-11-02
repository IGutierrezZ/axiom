# Delta para Selección Temprana y Bloqueante de Carril ODD/SDD en el Kickoff (paradas-tres-salidas)

> **Cambio:** `paradas-tres-salidas` · **Tipo:** delta sobre la capacidad viva `sdd-lane-kickoff`
> **Idioma:** Español (castellano peninsular); identificadores en inglés.

Cambios de este delta:

- REQ-21.2: el carril ODD admite una única parada por defecto tras crear el documento de feature y antes de la primera escritura de código (REQ-26.10 de `organic-agent-trigger-rules`). Esa parada NO es una pregunta de gobernanza. Se mantiene que ODD no divide en roles ni genera `handoff.md`.
- REQ-21.4: el retro-sello y la reanudación dejan de mencionar la política de relevos y los `handoff.md` intermedios (`handoff_policy` está deprecado, REQ-26.9 de `sdd-preflight-configuration`).
- REQ-21.1 y REQ-21.3 no cambian. Los REQ-25.35 a REQ-25.38 de esta capacidad no cambian: su escenario «no formula ninguna pregunta adicional de gobernanza» sigue siendo cierto, porque la parada de ODD ocurre más tarde y no es una pregunta de gobernanza.

## MODIFIED Requirements

### Requirement: Carril ODD sin Fricción Adicional (REQ-21.2)

Si el usuario elige el carril ODD, el orquestador NO DEBE formular ninguna pregunta adicional de gobernanza (modalidad de avance, roles, política de relevos), DEBE ejecutar el trabajo "del tirón" sin dividirlo en roles ni generar ningún `handoff.md`, y DEBE mantener el trabajo exclusivamente en `odd/tasks/<feature>.md`. La única interrupción prevista por defecto es la parada única de ODD tras crear el documento de feature y antes de la primera escritura de código, con las tres salidas de REQ-26.1 (REQ-26.10 de `organic-agent-trigger-rules`); esa parada NO es una pregunta de gobernanza y NO contraviene este requisito.
(Previously: no admitía ninguna interrupción del trabajo ODD distinta de las preguntas de producto o de entorno)

#### Scenario: Elección de ODD sin preguntas adicionales

- **DADO** que el usuario elige el carril ODD en respuesta a la pregunta de REQ-21.1
- **CUANDO** el orquestador inicia el trabajo
- **ENTONCES** no formula ninguna pregunta adicional de configuración de gobernanza
- **Y** el trabajo se registra exclusivamente en `odd/tasks/<feature>.md`

#### Scenario: Trabajo ODD multi-área sigue sin roles ni handoff

- **DADO** un trabajo bajo el carril ODD que toca varias áreas o módulos del repositorio
- **CUANDO** se ejecuta el trabajo hasta su cierre
- **ENTONCES** el orquestador no genera ningún artefacto `handoff.md`
- **Y** no introduce una división de roles aunque el trabajo abarque varias áreas

#### Scenario: La parada única de ODD no cuenta como pregunta de gobernanza

- **DADO** que el usuario eligió el carril ODD y el orquestador acaba de crear `odd/tasks/<feature>.md`
- **CUANDO** el orquestador se detiene antes de la primera escritura de código para ofrecer las tres salidas
- **ENTONCES** no ha formulado ninguna pregunta de modalidad, roles ni política de relevos
- **Y** la parada se rige por REQ-26.10 de `organic-agent-trigger-rules`

#### Scenario: La parada de ODD no genera handoff.md

- **DADO** un trabajo ODD cuya parada se resuelve con la salida 2 (aprobar, subir y parar)
- **CUANDO** el orquestador deja el plan para otra persona
- **ENTONCES** no genera ningún `handoff.md`: el relevo es el propio `odd/tasks/<feature>.md` confirmado en la rama

---

### Requirement: Idempotencia del Kickoff ante un Cambio con Artefactos o Configuración Ya Sellada (REQ-21.4)

El orquestador NO DEBE volver a formular la pregunta de selección de carril (REQ-21.1) ni el cuestionario de pre-vuelo de SDD (capacidad `sdd-preflight-configuration`) para un cambio que ya tiene al menos un artefacto persistido (`proposal.md`, un `spec.md` bajo `specs/`, `design.md` o `tasks.md`) o cuya configuración de kickoff ya está sellada. En ese caso, el orquestador DEBE continuar respetando el carril y la configuración ya selladas en el primer kickoff, sin interrogar de nuevo al usuario.

(Previamente: esta especificación indicaba que, al retro-sellar un cambio preexistente sin kickoff sellado, el rol por defecto congelado era el rol único `fullstack`. Se corrige tras la pregunta abierta O-1 planteada en `sdd-design` y resuelta explícitamente por el usuario con evidencia de código —`internal/multirole/detector.go:35-55` cae siempre a `core`, nunca a `fullstack`; `internal/multirole/detector.go:58-64` rechaza cualquier rol ausente de `axiom.yaml`; y `axiom.yaml:7-28` no declara `fullstack`—: el retro-sello DEBE congelar el rol que la detección de roles ya vigente infiere para ese cambio, nunca introducir la identidad `fullstack`.)
(Previously: la configuración sellada incluía la política de relevos y el retro-sello inferido nombraba «sin `handoff.md` intermedios»; la política de relevos ya no forma parte de la configuración sellada)

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
- **Y** el retro-sello no escribe ninguna política de relevos
- **Y** dicha configuración inferida queda registrada para no volver a inferirse en la siguiente reanudación

#### Scenario: El retro-sello no altera qué `tasks.<rol>.md` consulta la barrera

- **DADO** un cambio SDD preexistente sin kickoff sellado, cuya barrera de sincronización ya consulta `tasks.core.md` porque la detección de roles vigente infiere el rol `core` para su `design.md`
- **CUANDO** el orquestador retro-sella la configuración de kickoff de ese cambio conforme al escenario anterior
- **ENTONCES** la barrera de sincronización sigue consultando exactamente `tasks.core.md`, el mismo fichero que consultaba antes del sellado
- **Y** el retro-sello no crea, renombra ni redirige ningún fichero `tasks.<rol>.md`, ni introduce `tasks.fullstack.md` como fuente de verdad nueva para ese cambio
