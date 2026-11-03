# Delta para Reglas de Disparo de Agentes Orgánicos (paradas-tres-salidas)

> **Cambio:** `paradas-tres-salidas` · **Tipo:** delta sobre la capacidad viva `organic-agent-trigger-rules`
> **Idioma:** Español (castellano peninsular); identificadores en inglés.

Cambios de este delta:

- REQ-26.10 (nuevo): el protocolo ODD proyectado incluye una única parada por defecto tras crear el documento de feature y antes de la primera escritura de código, con las tres salidas de `sdd-block-review-gates` (REQ-26.1). La salida 2 deja el plan aprobado, confirmado en la rama y con la PR indicada, y la implementación la hace otra persona.
- REQ-25.40: se matiza el escenario «El resto del protocolo no cambia» para que no contradiga la parada (la sustitución de la regla de rama que fija ese requisito no cambia).
- El requisito «Organic Driven Development is the orchestrator's mandatory default protocol» no cambia: no cita ninguna regla de «sin más preguntas». La frase «do not ask any further governance question» vive en el paso 0 de la doctrina generada y en REQ-21.2 de `sdd-lane-kickoff`, que se enmienda en este mismo cambio.

## ADDED Requirements

### Requirement: El Protocolo ODD Incluye una Parada Única con Tres Salidas tras Crear el Documento de Feature (REQ-26.10)

Las instrucciones de enrutamiento gestionadas que proyectan el protocolo ODD DEBEN incluir **una única parada por defecto**, situada tras crear el documento de feature `odd/tasks/<feature>.md` (el plan) y antes de la primera escritura de código. La parada DEBE ofrecer las mismas tres salidas, con la misma redacción, que las paradas de SDD (REQ-26.1 y REQ-26.8 de `sdd-block-review-gates`):

1. **Iterar:** ajustar el plan en el propio documento de feature (fichero y espejo) y volver a presentar la parada. No hay límite de iteraciones.
2. **Aprobar, subir y parar:** el plan queda aprobado; la cesión se anota en el propio documento de feature (ODD no tiene registro de compuertas; forma en design); `odd/tasks/<feature>.md` se confirma (commit) en la rama que preparó `ensure`; el orquestador lee `axiom branch status --json` y presenta la rama, el `prTarget`, la orden de push exacta y los datos de la PR; y se detiene sin escribir código. La implementación la hace otra persona, que recoge el plan una vez fusionada la PR.
3. **Aprobar y seguir:** el plan queda aprobado y el orquestador continúa con la implementación tarea a tarea.

La parada: (a) solo existe cuando el trabajo es sustancial y se crea el documento de feature; el trabajo pequeño y comprendido, que no crea artefactos duraderos, NO DEBE tener parada; (b) NO es una pregunta de gobernanza y NO contraviene REQ-21.2 ni la regla del paso 0 contra las preguntas de gobernanza, que la doctrina DEBE citar como excepción documentada; (c) NO se repite al reanudar una feature cuyo plan ya está aprobado; (d) NO activa SDD ni crea artefactos SDD; (e) NO genera `handoff.md`. En `monorepo-embedded` la salida 2 significa PR fusionada en el destino indicado; en `multirepo`, donde el documento ODD vive en el repositorio de specs en su rama de integración, es commit en esa rama y push a cargo del humano, sin PR. La doctrina NO DEBE ordenar ni ejecutar `push`, `fetch`, apertura de PR ni merge: son del humano. Quien recoge la cesión DEBE partir del fichero `odd/tasks/<feature>.md` y la doctrina de reanudación DEBE tolerar la ausencia del espejo de Engram (que es local a cada persona), empezar por la primera tarea sin completar y NO revisar de nuevo el plan aprobado. La proyección DEBE seguir siendo determinista: el mismo texto en cada renderizado.

#### Scenario: La doctrina contiene una única parada tras el plan y antes del primer código

- **DADO** un adaptador registrado con un manifiesto de capacidades válido
- **CUANDO** se renderizan sus instrucciones de enrutamiento gestionadas
- **ENTONCES** el protocolo ODD contiene una única parada, después de crear el documento de feature y antes de la primera escritura de código
- **Y** la parada ofrece las tres salidas: iterar, aprobar-subir-y-parar y aprobar-y-seguir

#### Scenario: El texto de las tres salidas coincide con el de SDD

- **DADO** la doctrina ODD generada y las plantillas de orquestador SDD
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** el texto de las tres salidas es el mismo en ambas, procedente de la misma fuente (REQ-26.8)

#### Scenario: La salida 2 aprueba el plan, lo confirma, presenta la PR y deja la implementación a otra persona

- **DADO** una feature ODD en un monorepo con el documento `odd/tasks/mi-feature.md` recién creado en la rama `inc/mi-feature`
- **CUANDO** el usuario elige aprobar, subir y parar (salida 2)
- **ENTONCES** el documento anota la cesión del plan, queda confirmado en `inc/mi-feature` y el orquestador presenta la rama, el `prTarget`, la orden de push y los datos de la PR
- **Y** no se ha escrito código ni se ha creado ningún commit de implementación
- **Y** el orquestador se detiene

#### Scenario: La salida 3 continúa con la implementación

- **DADO** el plan recién creado y la parada presentada
- **CUANDO** el usuario elige aprobar y seguir (salida 3)
- **ENTONCES** el orquestador continúa con la primera tarea del plan
- **Y** el documento no anota ninguna cesión

#### Scenario: La salida 1 vuelve a presentar la parada tras ajustar el plan

- **DADO** el plan recién creado y la parada presentada
- **CUANDO** el usuario elige iterar y pide cambios
- **ENTONCES** el orquestador ajusta `odd/tasks/<feature>.md` y su espejo
- **Y** vuelve a presentar la parada con sus tres salidas antes de escribir código

#### Scenario: El trabajo pequeño no tiene parada

- **DADO** una petición pequeña y comprendida bajo ODD que no crea documento de feature
- **CUANDO** el orquestador la ejecuta
- **ENTONCES** no presenta la parada y el trabajo procede como antes

#### Scenario: Reanudar una feature con el plan ya aprobado no repite la parada

- **DADO** una feature cuyo documento anota el plan aprobado, con la cesión o sin ella
- **CUANDO** el orquestador la reanuda
- **ENTONCES** no vuelve a presentar la parada
- **Y** continúa por la primera tarea sin completar

#### Scenario: Otra persona recoge el plan desde el fichero sin espejo de Engram

- **DADO** una feature cedida con la salida 2, fusionada en la rama de integración, y una máquina sin espejo de Engram para esa feature
- **CUANDO** la otra persona reanuda la feature
- **ENTONCES** la doctrina de reanudación parte del fichero `odd/tasks/<feature>.md` y no falla por la ausencia del espejo
- **Y** no revisa de nuevo el plan aprobado y empieza por la primera tarea sin completar

#### Scenario: Multirepo confirma en la integración y no hay PR

- **DADO** un multirepo con el documento ODD en el repositorio de specs, en su rama de integración
- **CUANDO** el usuario elige la salida 2
- **ENTONCES** el orquestador confirma el documento en esa rama y presenta el push a cargo del humano
- **Y** no presenta ninguna rama de feature ni PR en el repositorio de specs

#### Scenario: Solo se confirma el documento de feature

- **DADO** un árbol con el documento de feature y otros cambios sin confirmar ajenos al plan
- **CUANDO** el usuario elige la salida 2
- **ENTONCES** el commit contiene solo `odd/tasks/<feature>.md`

#### Scenario: La doctrina no ordena operaciones remotas

- **DADO** la doctrina ODD generada
- **CUANDO** se inspecciona (pin de texto) la parada
- **ENTONCES** no ordena ejecutar `push`, `fetch`, abrir la PR ni fusionar: los presenta como decisiones del humano

#### Scenario: La doctrina cita la parada como excepción a las preguntas de gobernanza

- **DADO** la doctrina ODD generada
- **CUANDO** se inspecciona (pin de texto) el paso 0
- **ENTONCES** mantiene que elegir ODD no añade preguntas de gobernanza
- **Y** indica que la parada única tras el plan no es una pregunta de gobernanza

#### Scenario: El renderizado sigue siendo determinista

- **DADO** una sincronización de adaptador
- **CUANDO** se renderizan repetidamente las instrucciones de enrutamiento gestionadas
- **ENTONCES** el texto es idéntico en cada renderizado

## MODIFIED Requirements

### Requirement: El Protocolo ODD Prepara la Rama con `axiom branch ensure` (REQ-25.40)

Las instrucciones de enrutamiento gestionadas que proyectan el protocolo ODD DEBEN sustituir la regla «branch first when on the default branch» por la ejecución de `axiom branch ensure --lane odd --change <feature>` antes de la primera escritura del documento vivo `odd/tasks/<feature>.md` y de cualquier escritura de código, y DEBEN indicar que las órdenes posteriores usan como `--cwd` las rutas absolutas que devuelve su JSON. El cierre de cada tarea mantiene el commit por unidad de trabajo, ahora en la rama que `ensure` preparó. La regla NO DEBE alterar los demás pasos del protocolo ODD ni el carril SDD, NO DEBE exigir interacción con el usuario salvo el resultado `decision_required` del modo `ask`, y la proyección DEBE seguir siendo determinista: el mismo texto en cada renderizado. Esta regla no activa SDD ni crea artefactos SDD.
(Previously: el escenario «El resto del protocolo no cambia» no tenía en cuenta la parada única tras el plan que añade REQ-26.10)

#### Scenario: El paso de implementación invoca `ensure`
- **DADO** un adaptador registrado con un manifiesto de capacidades válido
- **CUANDO** se renderizan sus instrucciones de enrutamiento gestionadas
- **ENTONCES** el protocolo ODD indica ejecutar `axiom branch ensure --lane odd --change <feature>` antes de la primera escritura
- **Y** el texto ya no contiene «branch first when on the default branch»

#### Scenario: Las rutas del resultado se usan como `--cwd`
- **DADO** las instrucciones de enrutamiento renderizadas
- **CUANDO** se inspecciona (pin de texto) el paso de implementación
- **ENTONCES** indica usar las rutas absolutas del JSON de `ensure` como `--cwd` de las órdenes posteriores

#### Scenario: El resto del protocolo no cambia
- **DADO** las instrucciones renderizadas antes y después de la regla de rama de este requisito
- **CUANDO** se comparan los demás pasos del protocolo ODD (autorizar, explorar, resolver incertidumbre, clasificar, registrar, implementar, cerrar) y el carril SDD
- **ENTONCES** son idénticos salvo la sustitución de la regla de rama y la parada única tras el plan de REQ-26.10

#### Scenario: El renderizado sigue siendo determinista
- **DADO** una sincronización de adaptador
- **CUANDO** se renderizan repetidamente las instrucciones de enrutamiento gestionadas
- **ENTONCES** el texto es idéntico en cada renderizado

#### Scenario: El modo `ask` no añade una pregunta de gobernanza
- **DADO** las instrucciones renderizadas
- **CUANDO** se inspecciona (pin de texto) la regla
- **ENTONCES** la única interacción que prevé es presentar el resultado `decision_required` y repetir `ensure` con la respuesta
