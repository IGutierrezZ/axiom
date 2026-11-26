# Delta para Compuertas de Revisión por Bloque SDD (sdd-block-review-gates)

> **Cambio:** `html-explicativo` (C-06-b). **Idioma:** castellano de España; identificadores, códigos y tokens en inglés.
> RF-GAT-01 a RF-GAT-26 no cambian de texto: el HTML explicativo es un complemento informativo; lo que aprueba cada compuerta sigue siendo el md.

## ADDED Requirements

### Requirement: Contrato del HTML Explicativo que Generan `sdd-spec` y `sdd-design` cuando el Kickoff Sella `yes` (RF-GAT-27)

Cuando el kickoff del cambio sella `explainer_html: yes`, la fase `sdd-spec` DEBE escribir, después de los md de spec, el fichero `resumen-spec.html`, y la fase `sdd-design` DEBE escribir, después de `design.md`, el fichero `resumen-design.html`, ambos en la raíz de la carpeta del cambio (uno por cambio, no uno por dominio). Cada HTML DEBE:

- estar escrito en castellano, para público no técnico, e incluir qué cambia, para quién, los flujos antes y después, diagramas SVG en línea y un glosario;
- ser autocontenido: HTML, CSS y SVG en línea, sin ninguna etiqueta `<script>` y sin ningún recurso externo (ni URL `http://` o `https://` en `src`, `href` o `url(...)`, ni hojas de estilo, fuentes, imágenes o marcos externos);
- llevar `<meta name="axiom-source-digest" content="...">` con el mismo digest que `axiom` calcula para los md de origen de esa fase (los de spec para `resumen-spec.html`, `design.md` para `resumen-design.html`), de modo que su frescura sea verificable (RF-GAT-30).

Cuando el kickoff sella `no` o no tiene el campo, las fases NO DEBEN escribir estos HTML y su comportamiento NO DEBE cambiar. El HTML es exclusivo de SDD: ninguna fase de ODD, ni `sdd-propose`, `sdd-tasks` o `sdd-verify`, lo genera. La doctrina de las skills (y de sus copias por agente) DEBE exigir estas restricciones y una comprobación automática DEBE detectar si la doctrina deja de exigirlas.

#### Scenario: Con `yes`, `sdd-spec` escribe `resumen-spec.html` tras el md

- **DADO** un cambio con `explainer_html: yes` y la fase `sdd-spec` en curso
- **CUANDO** la fase termina
- **ENTONCES** existe `resumen-spec.html` en la raíz de la carpeta del cambio, además de los md de spec
- **Y** contiene `<meta name="axiom-source-digest">`

#### Scenario: Con `yes`, `sdd-design` escribe `resumen-design.html`

- **DADO** un cambio con `explainer_html: yes` y la fase `sdd-design` en curso
- **CUANDO** la fase termina
- **ENTONCES** existe `resumen-design.html` en la raíz de la carpeta del cambio, además de `design.md`

#### Scenario: El HTML es autocontenido

- **DADO** un `resumen-spec.html` o `resumen-design.html` generado
- **CUANDO** se inspecciona su contenido
- **ENTONCES** no contiene ninguna etiqueta `<script>`
- **Y** no referencia ninguna URL externa ni recurso externo
- **Y** se abre sin red con doble clic

#### Scenario: Con `no` o sin campo no se genera

- **DADO** un cambio sellado con `explainer_html: no`, o sin el campo
- **CUANDO** terminan `sdd-spec` y `sdd-design`
- **ENTONCES** no existe ningún `resumen-*.html` en la carpeta del cambio
- **Y** las fases producen los mismos md que antes de este cambio

#### Scenario: Las skills exigen las restricciones

- **DADO** las skills `sdd-spec` y `sdd-design` y sus copias por agente
- **CUANDO** se ejecuta la comprobación automática de la doctrina
- **ENTONCES** cada una exige la autocontención (sin `<script>` ni recursos externos), el meta `axiom-source-digest`, el castellano y la cita de IDs
- **Y** la comprobación falla si alguna exigencia desaparece

---

### Requirement: Contenido Fiel del HTML: Cita de IDs, Nada Fuera del md y Aviso de que se Aprueba el md (RF-GAT-28)

Cada bloque del HTML DEBE citar el ID del requisito (en `resumen-spec.html`) o de la decisión (en `resumen-design.html`) del md del que sale. El HTML NO DEBE afirmar nada que no esté en el md de origen. El HTML DEBE decir expresamente, de forma visible, que lo que se aprueba en la puerta es el md y no el HTML. Cuando una puerta se rechaza y la fase regenera el md (RF-GAT-06), DEBE regenerar también el HTML correspondiente, con el digest del md regenerado.

#### Scenario: Cada bloque cita su ID

- **DADO** un `resumen-spec.html` generado para un spec con dos requisitos con ID válido según el patrón del proyecto
- **CUANDO** se inspecciona el HTML
- **ENTONCES** cada bloque explicativo cita el ID del requisito del que procede
- **Y** todos los IDs citados existen en el md de origen

#### Scenario: Aviso de que se aprueba el md

- **DADO** un HTML explicativo generado
- **CUANDO** se inspecciona su texto visible
- **ENTONCES** declara que lo que se aprueba es el md y que el HTML es una ayuda de lectura

#### Scenario: Tras un rechazo se regeneran md y HTML

- **DADO** un cambio con `explainer_html: yes` cuya puerta `spec` o `design` se rechaza
- **CUANDO** la fase remedia y regenera el md
- **ENTONCES** regenera también el HTML correspondiente
- **Y** su `axiom-source-digest` coincide con el digest del md regenerado

---

### Requirement: El HTML Queda Fuera del Digest de las Puertas (RF-GAT-29)

Crear, modificar o borrar un `resumen-spec.html` o un `resumen-design.html` NO DEBE cambiar el digest que juzgan las puertas `spec` y `design`, NO DEBE reabrir una puerta ya aprobada ni cambiar su estado, y NO DEBE entrar en el conjunto de artefactos que una puerta aprueba. La puerta aprueba el md, que es el contrato.

#### Scenario: Crear el HTML no cambia el digest

- **DADO** un cambio con las puertas `spec` y `design` en cualquier estado y sin HTML
- **CUANDO** se crea `resumen-spec.html` y `resumen-design.html` y se recalcula el digest de cada puerta
- **ENTONCES** los dos digests son idénticos a los de antes

#### Scenario: Modificar el HTML no reabre una puerta aprobada

- **DADO** las puertas `spec` y `design` aprobadas
- **CUANDO** se modifica el contenido de `resumen-spec.html`
- **ENTONCES** ambas puertas siguen aprobadas
- **Y** el digest de cada una no cambia

#### Scenario: Modificar el md sí cambia el digest

- **DADO** una puerta `spec` aprobada
- **CUANDO** se modifica un md de spec
- **ENTONCES** el digest de la puerta cambia, como antes de este cambio

---

### Requirement: Frescura Informativa `fresh`, `stale` o `missing` del HTML en `axiom sdd status` (RF-GAT-30)

Cuando el kickoff del cambio sella `explainer_html: yes`, `axiom sdd status --json` DEBE proyectar, para `spec` y para `design`, un estado de frescura de su HTML con tres valores, calculado de solo lectura sobre los md de origen que ya descubre el status (sin una regla paralela de descubrimiento de specs) y con independencia del estilo de ejecución, también en `continuous`:

- `missing`: el HTML de ese artefacto no existe.
- `stale`: el HTML existe pero su `axiom-source-digest` no coincide con el digest actual de los md de origen, o no tiene la etiqueta `<meta name="axiom-source-digest">`, o esta no tiene un valor legible.
- `fresh`: el HTML existe y su `axiom-source-digest` coincide con el digest actual de los md de origen.

Con `explainer_html: no` o sin el campo no DEBE proyectarse ningún estado de frescura. Un artefacto sin md de origen todavía no proyecta estado. La frescura es solo informativa: NO DEBE cambiar `nextRecommended`, las dependencias, los motivos de bloqueo ni el digest de ninguna puerta, ni bloquear una fase, una puerta o el archivado (tampoco en un cambio ISO). Con el almacén de artefactos `engram` o `none` no se proyecta. El contrato publicado (`sdd-status-contract.md`) DEBE documentar la proyección.

#### Scenario: HTML al día es `fresh`

- **DADO** un cambio con `explainer_html: yes`, `spec.md` y un `resumen-spec.html` cuyo `axiom-source-digest` coincide con el digest actual
- **CUANDO** se ejecuta `axiom sdd status --change <x> --json`
- **ENTONCES** la frescura de `spec` es `fresh`

#### Scenario: Modificar el md marca el HTML como `stale`

- **DADO** el escenario anterior
- **CUANDO** se modifica un md de spec sin regenerar el HTML
- **ENTONCES** la frescura de `spec` es `stale`
- **Y** `nextRecommended` y las dependencias son idénticos a los de antes de modificarlo

#### Scenario: Sin HTML es `missing`

- **DADO** un cambio con `explainer_html: yes` y `design.md` sin `resumen-design.html`
- **CUANDO** se consulta el status
- **ENTONCES** la frescura de `design` es `missing`
- **Y** `nextRecommended` no cambia

#### Scenario: HTML sin meta de digest es `stale`

- **DADO** un `resumen-spec.html` que no contiene `<meta name="axiom-source-digest">`, o cuyo `content` está vacío
- **CUANDO** se consulta el status
- **ENTONCES** la frescura de `spec` es `stale`

#### Scenario: Se proyecta también en `continuous`

- **DADO** un cambio `continuous` con `explainer_html: yes` y un HTML desactualizado
- **CUANDO** se consulta el status
- **ENTONCES** la frescura es `stale`
- **Y** no se detiene ninguna fase por ello

#### Scenario: Con `no` o sin campo no se proyecta

- **DADO** un cambio con `explainer_html: no` o sin el campo, aunque exista un `resumen-spec.html`
- **CUANDO** se consulta el status
- **ENTONCES** no se proyecta ningún estado de frescura

#### Scenario: Un cambio ISO no se bloquea por la frescura

- **DADO** un cambio ISO con `explainer_html: yes` y un HTML `stale` o `missing`
- **CUANDO** se consulta el status
- **ENTONCES** ningún motivo de bloqueo cita la frescura
- **Y** el archivado no se condiciona a ella
