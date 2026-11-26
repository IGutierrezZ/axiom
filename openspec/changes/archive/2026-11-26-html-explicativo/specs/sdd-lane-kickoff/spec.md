# Delta para Kickoff del Carril SDD (sdd-lane-kickoff)

> **Cambio:** `html-explicativo` (C-06-b). **Idioma:** castellano de España; identificadores, códigos y tokens en inglés.
> Los requisitos RF-KOF-01 a RF-KOF-12 no cambian de texto: la clave `explainer_html` se añade junto a ellos sin alterar el sellado existente.

## ADDED Requirements

### Requirement: Declaración de Proyecto `sdd.explainer_html` con Valor por Defecto `ask` (RF-KOF-13)

`axiom.yaml` PUEDE declarar, dentro del bloque `sdd:`, la clave `explainer_html` con uno de tres valores: `ask`, `always` o `never`. El valor por defecto, cuando la clave o el bloque `sdd:` están ausentes, DEBE ser `ask`. El valor DEBE validarse con el decodificador estricto del bloque `sdd:` (RF-ISO-01 de `sdd-iso-evidence`): un valor no admitido DEBE fallar en `axiom workspace validate`, al cargar la configuración y al sellar, con un mensaje en español que nombre `sdd.explainer_html`, el valor y los admitidos. `axiom.example.yaml` DEBE documentar la clave y sus tres valores. La declaración es por proyecto y exclusiva de SDD: no afecta al carril ODD.

#### Scenario: Sin declaración el valor vigente es `ask`

- **DADO** un `axiom.yaml` sin el bloque `sdd:` o con `sdd:` sin la clave `explainer_html`
- **CUANDO** se resuelve el valor vigente del proyecto
- **ENTONCES** el valor es `ask`

#### Scenario: Valor inválido en `axiom.yaml` falla en validación y al sellar

- **DADO** un `axiom.yaml` con `sdd: {explainer_html: siempre}`
- **CUANDO** se ejecuta `axiom workspace validate` y, por separado, `axiom sdd kickoff seal` sobre un cambio de ese proyecto
- **ENTONCES** ambos reportan en español la clave `sdd.explainer_html`, el valor y los admitidos
- **Y** el código de salida es distinto de cero en ambos
- **Y** el sellado no escribe `kickoff.yaml` ni degrada el valor a `ask`

#### Scenario: La plantilla de ejemplo documenta la clave

- **DADO** `axiom.example.yaml`
- **CUANDO** se valida con `axiom workspace validate`
- **ENTONCES** es válido
- **Y** documenta `explainer_html` con `ask`, `always` y `never`

---

### Requirement: `kickoff.yaml` Registra el Campo Sellado `explainer_html` con Valores `yes` o `no` (RF-KOF-14)

`kickoff.yaml` PUEDE registrar el campo `explainer_html` con uno de dos valores: `yes` o `no`. El campo DEBE omitirse cuando no está sellado (`omitempty`) y su ausencia DEBE equivaler a `no`. El campo DEBE escribirse una sola vez, por `axiom sdd kickoff seal`, y DEBE ser independiente del estilo de ejecución (`continuous` o `checkpointed`), del rol y de `docs_publication`. Un valor distinto de `yes` o `no` en un `kickoff.yaml` DEBE rechazarse como inválido. Un cambio sellado antes de este cambio, sin el campo, NO DEBE cambiar de comportamiento. Un binario anterior ignora el campo al leer `kickoff.yaml`.

#### Scenario: Sellado con `yes` queda registrado

- **DADO** un cambio sin sellar en un proyecto con `explainer_html: ask`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --execution-style continuous --explainer-html yes`
- **ENTONCES** `kickoff.yaml` contiene `explainer_html: yes`

#### Scenario: Ausente equivale a `no`

- **DADO** un cambio sellado antes de este cambio, cuyo `kickoff.yaml` no tiene `explainer_html`
- **CUANDO** se consulta el valor sellado
- **ENTONCES** el valor efectivo es `no`
- **Y** el resto del kickoff se lee exactamente igual que antes

#### Scenario: Es independiente del estilo de ejecución

- **DADO** dos cambios sellados con `--explainer-html yes`, uno `continuous` y otro `checkpointed`
- **CUANDO** se leen sus `kickoff.yaml`
- **ENTONCES** ambos registran `explainer_html: yes`
- **Y** el estilo, los roles y `docs_publication` de cada uno no cambian por ello

#### Scenario: Valor inválido en `kickoff.yaml`

- **DADO** un `kickoff.yaml` con `explainer_html: quizás`
- **CUANDO** se lee el kickoff
- **ENTONCES** se rechaza como inválido, nombrando el campo y los admitidos (`yes`, `no`)

---

### Requirement: La Bandera `--explainer-html` y la Resolución contra el Valor del Proyecto (RF-KOF-15)

`axiom sdd kickoff seal` DEBE aceptar la bandera `--explainer-html yes|no`. El valor sellado DEBE resolverse así:

- Con `sdd.explainer_html: always`, se sella `yes` sin preguntar; con `never`, se sella `no` sin preguntar.
- Una bandera cuyo valor contradiga `always` (`--explainer-html no`) o `never` (`--explainer-html yes`) DEBE rechazarse con un error que nombre ambos valores, sin escribir nada. Una bandera coherente con `always` o `never` se acepta y no cambia el resultado.
- Con `ask` y la bandera presente, se sella el valor de la bandera.
- Con `ask` y sin bandera, se sella `no` y la salida DEBE incluir un aviso de que no se ha preguntado y de cómo activar el HTML explicativo; el sellado termina con éxito.
- Un valor de la bandera distinto de `yes` o `no` DEBE rechazarse, nombrando los admitidos, sin escribir nada.
- `--infer` (retrosellado de un cambio preexistente) DEBE rechazar la bandera `--explainer-html`, no DEBE escribir el campo ni heredarlo del proyecto, y DEBE dejarlo ausente (equivale a `no`).

La salida de `seal`, en texto y en JSON, DEBE informar del valor sellado. La ayuda del subcomando DEBE documentar la bandera.

#### Scenario: `always` sella `yes` sin bandera

- **DADO** un proyecto con `sdd.explainer_html: always`
- **CUANDO** se ejecuta `axiom sdd kickoff seal` sin `--explainer-html`
- **ENTONCES** `kickoff.yaml` contiene `explainer_html: yes`
- **Y** no se emite ningún aviso

#### Scenario: `never` sella `no` sin bandera

- **DADO** un proyecto con `sdd.explainer_html: never`
- **CUANDO** se ejecuta `axiom sdd kickoff seal` sin `--explainer-html`
- **ENTONCES** `kickoff.yaml` contiene `explainer_html: no`

#### Scenario: Una bandera contradictoria con `always` o `never` se rechaza sin escribir

- **DADO** un proyecto con `sdd.explainer_html: never`, y otro con `always`
- **CUANDO** se ejecuta `seal --explainer-html yes` en el primero y `seal --explainer-html no` en el segundo
- **ENTONCES** ambos se rechazan con un error que nombra el valor del proyecto y el de la bandera
- **Y** no se escribe `kickoff.yaml` ni se crea la carpeta del cambio

#### Scenario: Una bandera coherente con `always` se acepta

- **DADO** un proyecto con `sdd.explainer_html: always`
- **CUANDO** se ejecuta `seal --explainer-html yes`
- **ENTONCES** el sellado tiene éxito con `explainer_html: yes`

#### Scenario: Con `ask` la bandera decide

- **DADO** un proyecto con `sdd.explainer_html: ask`
- **CUANDO** se ejecuta `seal --explainer-html no` y, en otro cambio, `seal --explainer-html yes`
- **ENTONCES** se sellan `no` y `yes` respectivamente, sin aviso

#### Scenario: `ask` sin bandera sella `no` con aviso

- **DADO** un proyecto con `sdd.explainer_html: ask` (o sin declararlo)
- **CUANDO** se ejecuta `axiom sdd kickoff seal` sin `--explainer-html`
- **ENTONCES** `kickoff.yaml` contiene `explainer_html: no`
- **Y** la salida incluye un aviso de que no se ha preguntado
- **Y** el sellado termina con código de salida `0`

#### Scenario: Valor inválido en la bandera

- **DADO** cualquier proyecto válido
- **CUANDO** se ejecuta `seal --explainer-html quizás`
- **ENTONCES** se rechaza nombrando los valores admitidos `yes` y `no`
- **Y** no se escribe nada

#### Scenario: `--infer` rechaza la bandera y no escribe el campo

- **DADO** un cambio preexistente con artefactos y sin `kickoff.yaml`, en un proyecto con `sdd.explainer_html: always`
- **CUANDO** se ejecuta `seal --infer --explainer-html yes`
- **ENTONCES** se rechaza porque `--infer` no admite la bandera
- **Y** al ejecutar `seal --infer` sin bandera, el `kickoff.yaml` resultante no contiene `explainer_html`

#### Scenario: Cambio sin carpeta

- **DADO** un cambio nombrado cuya carpeta no existe
- **CUANDO** se ejecuta `seal --explainer-html yes` con un valor válido
- **ENTONCES** el sellado crea la carpeta (RF-KOF-11) y registra `explainer_html: yes`

#### Scenario: La salida informa del valor sellado

- **DADO** un sellado correcto con `--json`
- **CUANDO** se lee la salida
- **ENTONCES** incluye el valor sellado de `explainer_html`

---

### Requirement: El Status Proyecta el Valor de Proyecto y el Valor Sellado de `explainer_html` (RF-KOF-16)

`axiom sdd status --json` DEBE proyectar de forma aditiva: (a) el valor vigente del proyecto (`ask`, `always` o `never`), con o sin cambio seleccionado, para que el orquestador sepa si debe preguntar antes de sellar; y (b) el valor sellado del cambio consultado dentro de `governance.kickoff` (`yes` o `no`), para que las fases sepan si deben generar el HTML. Un cambio sin el campo sellado se proyecta como `no` o sin la clave, de modo que un consumidor anterior no observa diferencia. Con un `sdd:` inválido el status DEBE proyectar la declaración como inválida en lugar de asumir `ask`. Ninguna clave existente del status DEBE cambiar de nombre, tipo ni valor, y la proyección NO DEBE alterar `nextRecommended`, las dependencias ni los motivos de bloqueo. El contrato publicado (`sdd-status-contract.md`) DEBE documentar ambas proyecciones.

#### Scenario: Valor de proyecto sin cambio seleccionado

- **DADO** un `axiom.yaml` con `sdd: {explainer_html: always}` y ningún cambio seleccionado
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `changeName` es `null`
- **Y** el status proyecta el valor de proyecto `always`

#### Scenario: Valor sellado en `governance.kickoff`

- **DADO** un cambio sellado con `explainer_html: yes`
- **CUANDO** se ejecuta `axiom sdd status --change <x> --json`
- **ENTONCES** `governance.kickoff` proyecta el valor sellado `yes`

#### Scenario: Cambio sellado antes de este cambio

- **DADO** un cambio cuyo `kickoff.yaml` no tiene `explainer_html`
- **CUANDO** se consulta el status
- **ENTONCES** el valor sellado se proyecta como `no` o se omite
- **Y** ninguna otra clave del status cambia

#### Scenario: Declaración inválida

- **DADO** un `axiom.yaml` con `sdd: {explainer_html: quizás}`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el status proyecta la declaración como inválida con el error de `sdd.explainer_html`
- **Y** no asume `ask`

#### Scenario: La proyección no altera el encadenado

- **DADO** el mismo cambio con `explainer_html: yes` y con `explainer_html: no`
- **CUANDO** se consulta el status
- **ENTONCES** `nextRecommended`, las dependencias y los motivos de bloqueo son idénticos
