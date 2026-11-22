# Delta para organic-agent-trigger-rules

Series de requisitos: REQ-33.12 (continúa la serie de este cambio). No modifica REQ-27.17 ni REQ-32.4: sus pines siguen vigentes.

## ADDED Requirements

### Requirement: La Doctrina ODD Avisa por Commit en el Modo 1 Fijo, sin Preguntas ni Paradas (REQ-33.12)

La doctrina ODD generada (paso 6 del protocolo) DEBE indicar que, al cerrar cada tarea con su commit de unidad de trabajo, el orquestador avisa en el chat con el sha, la tarea y la orden `git show <sha>`, y continúa sin esperar. ODD DEBE funcionar siempre en el comportamiento `notify`: la doctrina ODD NO DEBE preguntar el modo de revisión por commit, NO DEBE ofrecerlo ni aceptarlo a petición en el chat, y NO DEBE contener ninguno de los tokens `stop` o `stop-before-commit` como modo ni vocabulario de paradas, de modo que los pines de REQ-27.17 sigan en verde. La doctrina ODD DEBE incluir además las reglas comunes de REQ-33.9: pedir la PR da por validados todos los commits pendientes y nada la bloquea; solo se registran, en `odd/tasks/<feature>.md`, los commits sobre los que la persona pidió cambios; los cambios sobre un commit ya superado usan `git commit --fixup=<sha>` con autosquash no interactivo diferido antes de abrir la PR si la rama no está publicada, o un commit correctivo nuevo si ya lo está, sin reescribir nunca historia publicada; y se recomienda fusionar sin squash. La doctrina ODD NO DEBE mencionar RDD ni recibos.

#### Scenario: El paso 6 contiene el aviso por commit

- **DADO** la doctrina ODD generada
- **CUANDO** se lee el paso 6
- **ENTONCES** pide avisar con sha, tarea y `git show <sha>` al cerrar cada tarea y continuar sin esperar

#### Scenario: ODD no contiene vocabulario de paradas ni del modo

- **DADO** la doctrina ODD generada
- **CUANDO** se busca `stop-before-commit`, una pregunta del modo o las tres salidas
- **ENTONCES** no aparece ninguno
- **Y** las pruebas de REQ-27.17 (`routing_test.go`, `routing_stops_test.go`) siguen en verde sin cambios

#### Scenario: Una petición de parada en el chat no cambia el modo

- **DADO** una feature ODD en curso
- **CUANDO** la persona pide en el chat que se pare tras cada tarea
- **ENTONCES** la doctrina no define ningún mecanismo de parada por tarea y el comportamiento sigue siendo avisar y continuar

#### Scenario: El registro de commits con cambios pedidos va al documento ODD

- **DADO** la doctrina ODD generada
- **CUANDO** se lee la regla de registro
- **ENTONCES** indica `odd/tasks/<feature>.md` como destino y limita el registro a los commits con cambios pedidos

#### Scenario: La doctrina ODD declara validación implícita y merge sin squash

- **DADO** la doctrina ODD generada
- **CUANDO** se leen las reglas de la PR y de la fusión
- **ENTONCES** declaran que pedir la PR valida todos los commits pendientes, que nada la bloquea y que se recomienda merge commit sin squash
