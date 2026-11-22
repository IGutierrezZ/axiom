# Delta para sdd-orchestrator-assets

Series de requisitos: REQ-33.8 a REQ-33.11 (continúan la serie de `sdd-block-review-gates` de este mismo cambio). Este delta describe comportamiento observable de la doctrina y de las skills; el mecanismo lo fija design.

## ADDED Requirements

### Requirement: Sección «Commit Review» de Fuente Única Proyectada a Todos los Runtimes (REQ-33.8)

La doctrina SDD DEBE incluir una sección «Commit Review» con una única fuente en `_shared/sdd-orchestrator-sections.md`, referenciada solo por su marcador desde el orquestador de cada runtime soportado y desde el flujo del orquestador de Claude, de modo que el texto proyectado sea idéntico en todos. La sección DEBE registrarse en la lista de secciones compartidas, de modo que la prueba de fuente única y la de ausencia de deriva la cubran. La sección «SDD Change Kickoff and Block Gates» DEBE enlazar la interacción de la puerta `tasks` de REQ-33.5. La sección NO DEBE mencionar RDD ni recibos.

#### Scenario: Todos los orquestadores proyectan la sección

- **DADO** los orquestadores generados de todos los runtimes
- **CUANDO** se comprueba la presencia de la sección «Commit Review»
- **ENTONCES** cada uno contiene el mismo texto, expandido desde la fuente única, y ninguno tiene una copia propia

#### Scenario: Un marcador sin sección registrada falla

- **DADO** un orquestador que referencia el marcador de «Commit Review»
- **CUANDO** la sección no figura en la lista de secciones compartidas
- **ENTONCES** la prueba de fuente única falla

#### Scenario: Los goldens reflejan la sección

- **DADO** los goldens de los orquestadores
- **CUANDO** se regeneran con la herramienta de actualización y se ejecutan las pruebas
- **ENTONCES** pasan sin ediciones manuales

### Requirement: Contenido Común de la Doctrina «Commit Review» (REQ-33.9)

La sección DEBE establecer, como mínimo: (a) al cerrar cada tarea con commit, el orquestador avisa en el chat con el sha, la tarea y la orden `git show <sha>`; (b) los tres modos de REQ-33.1 y su efecto, aclarando que en `stop` y `stop-before-commit` la pausa por tarea no es una puerta, no usa las tres salidas y no escribe en `gates.yaml`; (c) en `notify` el trabajo continúa sin parar; (d) pedir la PR da por validados todos los commits pendientes, no existe aprobación commit a commit y nada bloquea la PR; (e) solo se registran los commits sobre los que la persona pidió cambios; (f) para cambios sobre un commit ya superado, si la rama no está publicada (ningún remoto la contiene) se usa `git commit --fixup=<sha>` y un autosquash no interactivo diferido hasta antes de abrir la PR, y si ya está publicada se usa un commit correctivo nuevo, y nunca se reescribe historia publicada; (g) para intercalar un `fixup!` se termina o se aparca primero la tarea en curso; (h) se recomienda fusionar con merge commit, sin squash, para que los commits revisados lleguen a la rama destino, y Axiom no fusiona. Los modos 2 y 3 DEBEN describir además el avance tarea a tarea y, en el modo 3, que el commit solo se hace tras la aprobación de la persona. El comportamiento por tarea DEBE cumplirse aunque la fase de implementación devuelva varias tareas juntas: el aviso es uno por commit y se emite por el orquestador al recuperar el control.

#### Scenario: La sección contiene el aviso y la orden de revisión

- **DADO** la sección proyectada
- **CUANDO** se lee la regla del aviso
- **ENTONCES** pide sha, tarea y `git show <sha>` por cada commit de tarea

#### Scenario: La sección describe los tres modos sin convertirlos en puerta

- **DADO** la sección proyectada
- **CUANDO** se leen los modos `stop` y `stop-before-commit`
- **ENTONCES** declaran que la pausa no es una puerta y no escribe en `gates.yaml`

#### Scenario: La sección declara la validación implícita

- **DADO** la sección proyectada
- **CUANDO** se lee la regla de la PR
- **ENTONCES** dice que pedir la PR valida todos los commits pendientes y que nada bloquea la PR

#### Scenario: La sección distingue publicada de no publicada

- **DADO** la sección proyectada
- **CUANDO** se leen las reglas de `fixup!`
- **ENTONCES** indican autosquash diferido antes de la PR si la rama no está publicada, commit correctivo si lo está, y la prohibición de reescribir historia publicada

#### Scenario: La sección recomienda merge sin squash

- **DADO** la sección proyectada
- **CUANDO** se lee la recomendación de fusión
- **ENTONCES** recomienda merge commit sin squash y declara que Axiom no fusiona

#### Scenario: Un lote devuelto de golpe sigue produciendo un aviso por commit

- **DADO** un lote de `apply` que devuelve tres commits de tarea en modo `notify`
- **CUANDO** el orquestador recupera el control
- **ENTONCES** emite tres avisos, uno por sha, cada uno con su tarea y su `git show <sha>`

### Requirement: Contrato `commit_review` de `sdd-apply` (REQ-33.10)

La skill `sdd-apply` DEBE documentar el parámetro de lanzamiento `commit_review` con el vocabulario de REQ-33.1, tomado de la proyección de `status` (REQ-33.4), con `notify` por defecto cuando falta. En `notify` DEBE devolver, junto a cada tarea completada, el sha de su commit. En `stop` y `stop-before-commit` DEBE devolver el control al orquestador tras cada tarea. En `stop-before-commit` NO DEBE hacer el commit de la tarea hasta recibir la aprobación de la persona, dejando el trabajo de esa tarea sin confirmar y dejando constancia del estado de la tarea para poder reanudar. La skill DEBE indicar dónde se registran, en SDD, los commits con cambios pedidos (REQ-33.9 e), en un artefacto de seguimiento del rol cuya ubicación fija design, sin reabrir la puerta `tasks`.

#### Scenario: `notify` devuelve un sha por tarea

- **DADO** `apply` lanzado con `commit_review: notify` o sin el parámetro
- **CUANDO** completa varias tareas
- **ENTONCES** el resultado lista cada tarea con el sha de su commit

#### Scenario: `stop` devuelve el control tras cada tarea

- **DADO** `apply` lanzado con `commit_review: stop`
- **CUANDO** completa una tarea con su commit
- **ENTONCES** devuelve el control con ese sha y no empieza la tarea siguiente

#### Scenario: `stop-before-commit` no confirma antes de la aprobación

- **DADO** `apply` lanzado con `commit_review: stop-before-commit`
- **CUANDO** termina el trabajo de una tarea
- **ENTONCES** devuelve el control con el cambio sin confirmar y sin sha
- **Y** el commit se hace solo tras la aprobación de la persona

#### Scenario: Registrar cambios pedidos no reabre `tasks`

- **DADO** una puerta `tasks` aprobada
- **CUANDO** se registra un commit con cambios pedidos
- **ENTONCES** la puerta `tasks` sigue `approved`, no se reabre y conserva su `commit_review`, aunque cambie el contenido del fichero de tareas

### Requirement: La Skill `work-unit-commits` Incluye `fixup!`, Autosquash y Merge sin Squash, en Paridad (REQ-33.11)

Las dos copias de la skill `work-unit-commits` (la embebida en `internal/assets/skills/` y la del repositorio en `skills/`) DEBEN recibir las reglas de REQ-33.9 (f) y (h) con el mismo texto, y la paridad entre ambas DEBE seguir vigilada por la prueba de paridad de copias.

#### Scenario: Las copias mantienen la paridad

- **DADO** las dos copias de `work-unit-commits`
- **CUANDO** se ejecuta la prueba de paridad
- **ENTONCES** pasa y ambas contienen las reglas de `fixup!`, autosquash diferido, commit correctivo y merge sin squash

#### Scenario: Divergir entre copias falla

- **DADO** una copia con las reglas y otra sin ellas
- **CUANDO** se ejecuta la prueba de paridad
- **ENTONCES** falla
