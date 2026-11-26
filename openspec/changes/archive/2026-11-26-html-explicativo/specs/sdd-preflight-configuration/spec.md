# Delta para Configuración del Pre-Vuelo SDD (sdd-preflight-configuration)

> **Cambio:** `html-explicativo` (C-06-b). **Idioma:** castellano de España; identificadores, códigos y tokens en inglés.
> RF-PRE-01 a RF-PRE-15 no cambian de texto: la pregunta nueva es una interacción propia, no un grupo del pre-vuelo, como la preferencia de RF-PRE-06.

## ADDED Requirements

### Requirement: La Pregunta del HTML Explicativo es una Interacción Propia de Marcador `Axiom SDD kickoff explainer:`, Una Vez por Cambio y Solo con `ask` (RF-PRE-16)

Cuando el valor de proyecto `sdd.explainer_html` es `ask` (RF-KOF y su proyección en el status, capacidad `sdd-lane-kickoff`), el orquestador DEBE formular una única pregunta de selección única, con las opciones `yes` y `no` (etiquetas y descripciones traducidas al idioma de la conversación; los tokens canónicos no se traducen), una sola vez al iniciar el cambio: después de `axiom branch ensure` y antes de `axiom sdd kickoff seal`. La respuesta DEBE pasarse al sellado con `--explainer-html` y quedar en `kickoff.yaml`.

La pregunta DEBE ser una interacción propia, nunca un grupo del pre-vuelo: su marcador DEBE ser `Axiom SDD kickoff explainer:`, distinto del prefijo `Axiom SDD preflight ` de los grupos y distinto de los marcadores de la preferencia de revisión (RF-PRE-06). Por lo tanto el hook de validación del pre-vuelo y el plugin de OpenCode (RF-PRE-13) DEBEN ignorarla: ni la cuentan como grupo ni la rechazan, y no altera el número ni el significado de los grupos exigidos por topología (RF-PRE-07, RF-PRE-09, RF-PRE-10).

La pregunta NO DEBE formularse: con `always` o `never` (se sella `yes` o `no` sin preguntar); en el carril ODD ni en ninguna conversación que no esté iniciando un cambio SDD; ni de nuevo para un cambio ya sellado, con o sin el campo, ni al reanudarlo. Es independiente de la topología y del estilo de ejecución.

#### Scenario: Con `ask` se pregunta una sola vez al iniciar el cambio

- **DADO** un proyecto con `sdd.explainer_html: ask` y un cambio SDD recién nombrado, con `axiom branch ensure` ya ejecutado
- **CUANDO** el orquestador prepara el sellado
- **ENTONCES** formula una única pregunta con marcador `Axiom SDD kickoff explainer:` y opciones `yes` y `no`
- **Y** la formula antes de `axiom sdd kickoff seal`
- **Y** tras la respuesta `seal` se ejecuta con `--explainer-html` y `kickoff.yaml` contiene el valor elegido

#### Scenario: Con `always` o `never` no se pregunta

- **DADO** un proyecto con `sdd.explainer_html: always` o `never`
- **CUANDO** se inicia un cambio SDD
- **ENTONCES** no se formula la pregunta
- **Y** el sellado resuelve `yes` o `no` respectivamente

#### Scenario: En ODD nunca aparece

- **DADO** trabajo tramitado por el carril ODD, con cualquier valor de `sdd.explainer_html`
- **CUANDO** se ejecuta el protocolo ODD completo
- **ENTONCES** no se formula la pregunta
- **Y** no se escribe ningún `kickoff.yaml`

#### Scenario: Un cambio ya sellado no se vuelve a preguntar

- **DADO** un cambio ya sellado, con `explainer_html` o sin él, en un proyecto con `ask`
- **CUANDO** el orquestador reanuda el cambio
- **ENTONCES** no se formula la pregunta
- **Y** el valor sellado no cambia

#### Scenario: El hook del pre-vuelo ignora la pregunta

- **DADO** una invocación de la herramienta de preguntas con un único elemento cuyo texto de pregunta empieza por `Axiom SDD kickoff explainer:`
- **CUANDO** el hook de validación del pre-vuelo la evalúa
- **ENTONCES** la ignora sin denegarla ni contarla como grupo
- **Y** el resultado es el mismo que para la pregunta de preferencia de revisión (RF-PRE-06)

#### Scenario: El plugin de OpenCode también la ignora

- **DADO** la misma pregunta en el plugin de OpenCode
- **CUANDO** el plugin evalúa la invocación
- **ENTONCES** la ignora sin tratarla como grupo del pre-vuelo

#### Scenario: No cambia los grupos del pre-vuelo

- **DADO** cualquier topología con o sin pregunta del HTML explicativo
- **CUANDO** se validan los grupos del pre-vuelo
- **ENTONCES** los grupos exigidos y su validación son idénticos a los de antes de este cambio
- **Y** la pregunta del HTML explicativo nunca se presenta como cuarto grupo
