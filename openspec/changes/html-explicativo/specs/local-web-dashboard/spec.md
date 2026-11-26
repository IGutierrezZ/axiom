# Delta para Dashboard Web Local (local-web-dashboard)

> **Cambio:** `html-explicativo` (C-06-b). **Idioma:** castellano de España; identificadores, códigos y tokens en inglés.
> RF-WEB-01 a RF-WEB-14 no cambian de texto. Las cabeceras globales y la política de orígenes del servidor no se tocan en este cambio.

## ADDED Requirements

### Requirement: Endpoint de Solo Lectura que Sirve Únicamente `resumen-spec.html` y `resumen-design.html` (prov:WEB:1)

El dashboard DEBE exponer un endpoint de solo lectura (método GET) que sirva el contenido de `resumen-spec.html` o de `resumen-design.html` de un cambio, y ningún otro fichero. El nombre del fichero DEBE validarse contra una lista blanca de exactamente esos dos nombres. El nombre del cambio DEBE rechazarse (error de cliente) cuando contenga `..`, un separador de ruta (`/` o `\`), su forma codificada (por ejemplo `%2e%2e`, `%2f`, `%5c`) o un byte nulo, sin llegar a leer del disco. Un fichero fuera de la lista blanca DEBE rechazarse sin leer del disco. Si el cambio no existe o el fichero no existe, la respuesta DEBE ser 404. Un método distinto de GET DEBE rechazarse.

La respuesta correcta DEBE llevar, fijadas por el propio endpoint:

- `Content-Type: text/html; charset=utf-8`;
- `Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; img-src data:`, con ese valor exacto;
- `X-Content-Type-Options: nosniff`.

El endpoint NO DEBE modificar ningún fichero ni estado, y NO DEBE ampliar la lista de ficheros servibles.

#### Scenario: Sirve `resumen-spec.html` con las cabeceras de seguridad

- **DADO** un cambio con `resumen-spec.html`
- **CUANDO** se pide ese fichero al endpoint
- **ENTONCES** la respuesta es 200 con el contenido del fichero
- **Y** `Content-Security-Policy` vale exactamente `default-src 'none'; style-src 'unsafe-inline'; img-src data:`
- **Y** `X-Content-Type-Options` vale `nosniff`
- **Y** `Content-Type` es `text/html; charset=utf-8`

#### Scenario: Sirve `resumen-design.html`

- **DADO** un cambio con `resumen-design.html`
- **CUANDO** se pide ese fichero al endpoint
- **ENTONCES** la respuesta es 200 con el contenido del fichero y las mismas cabeceras

#### Scenario: Cualquier otro fichero se rechaza

- **DADO** un cambio con `spec.md`, `kickoff.yaml` y `proposal.md`
- **CUANDO** se pide al endpoint `spec.md`, `kickoff.yaml`, `proposal.md` o `resumen-tasks.html`
- **ENTONCES** todas las peticiones se rechazan con un error de cliente
- **Y** ninguna devuelve contenido del fichero

#### Scenario: Nombre de cambio con `..` o separadores

- **DADO** un directorio hermano fuera de `openspec/changes/` con un `resumen-spec.html`
- **CUANDO** se pide con el cambio `..`, `../x`, `x/..`, `%2e%2e`, `a%2fb` o `a\b`
- **ENTONCES** todas se rechazan con un error de cliente
- **Y** no se devuelve el contenido de ningún fichero fuera de la carpeta del cambio

#### Scenario: Fichero ausente

- **DADO** un cambio sin `resumen-design.html`
- **CUANDO** se pide ese fichero al endpoint
- **ENTONCES** la respuesta es 404

#### Scenario: Cambio inexistente

- **DADO** un nombre de cambio válido sin carpeta
- **CUANDO** se pide `resumen-spec.html`
- **ENTONCES** la respuesta es 404

#### Scenario: Método distinto de GET

- **DADO** el endpoint
- **CUANDO** se envía POST, PUT o DELETE
- **ENTONCES** la petición se rechaza
- **Y** no se modifica ningún fichero

---

### Requirement: Pestaña «Explicación» en el Detalle del Incremento con `iframe` en Sandbox y Aviso de Desactualizado (prov:WEB:2)

El detalle de un incremento SDD DEBE incluir una pestaña «Explicación» que muestre `resumen-spec.html` y `resumen-design.html` del cambio, cada uno en un `iframe` cuyo atributo `sandbox` NO DEBE contener nunca `allow-same-origin` ni `allow-scripts` (puede estar vacío). El `iframe` DEBE cargar el contenido desde el endpoint de prov:WEB:1, nunca incrustarlo en el documento. El detalle DEBE proyectar la frescura de cada HTML con el mismo cálculo que `axiom sdd status` (`fresh`, `stale`, `missing`): con `stale` la pestaña DEBE mostrar un aviso visible de que el resumen está desactualizado respecto del md; con `missing` DEBE indicar que no hay explicación generada, sin mostrar un `iframe` roto. La pestaña NO DEBE ofrecer edición del HTML ni ninguna acción de escritura, y NO DEBE mostrar nada de HTML cuando el cambio no lo tiene sellado (`explainer_html` ausente o `no`) más allá de indicar que la explicación no está activada para el cambio. La frescura nunca bloquea ninguna otra acción del detalle.

#### Scenario: HTML al día se muestra en un `iframe` en sandbox

- **DADO** un incremento con `explainer_html: yes` y un `resumen-spec.html` `fresh`
- **CUANDO** se abre la pestaña «Explicación»
- **ENTONCES** se muestra un `iframe` con `sandbox` que no contiene `allow-same-origin` ni `allow-scripts`
- **Y** su origen apunta al endpoint de prov:WEB:1
- **Y** no se muestra aviso de desactualizado

#### Scenario: HTML desactualizado muestra el aviso

- **DADO** un incremento cuyo `resumen-spec.html` está `stale`
- **CUANDO** se abre la pestaña «Explicación»
- **ENTONCES** se muestra un aviso visible de que el resumen está desactualizado respecto del md
- **Y** el `iframe` sigue mostrando el HTML existente

#### Scenario: HTML ausente

- **DADO** un incremento con `explainer_html: yes` sin `resumen-design.html`
- **CUANDO** se abre la pestaña «Explicación»
- **ENTONCES** la sección de design indica que no hay explicación generada
- **Y** no se muestra un `iframe` para ese fichero

#### Scenario: Cambio sin explicación activada

- **DADO** un incremento con `explainer_html: no` o sin el campo
- **CUANDO** se abre la pestaña «Explicación»
- **ENTONCES** indica que la explicación no está activada para el cambio
- **Y** no se carga ningún `iframe`

#### Scenario: La frescura del dashboard coincide con la del status

- **DADO** un incremento con HTML `stale` según `axiom sdd status`
- **CUANDO** se consulta el detalle del incremento por la API del dashboard
- **ENTONCES** la frescura proyectada es la misma
- **Y** la consulta no modifica ningún fichero

#### Scenario: Sin acciones de escritura

- **DADO** la pestaña «Explicación»
- **CUANDO** se inspeccionan sus controles
- **ENTONCES** no ofrece editar, guardar ni regenerar el HTML
