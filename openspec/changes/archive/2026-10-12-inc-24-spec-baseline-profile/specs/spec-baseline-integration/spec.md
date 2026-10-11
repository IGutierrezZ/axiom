# Especificación de Requerimientos: Integración de los Cambios en la Baseline del Proyecto (INC-24)

> **Incremento:** `inc-24-spec-baseline-profile` · **Rebanada:** C (integración) · **Tipo:** capacidad nueva
> **Idioma:** Español (castellano peninsular)

## Propósito

Definir cómo integra archive un cambio en la baseline declarada del proyecto: la composición determinista sobre documentos direccionables por requisito (patrón de encabezado, identidad por ID, plantilla de render, duplicados, frontera de bloque, fidelidad de bytes), la integración de documentos de prosa mediante edición dirigida con verificación nativa y traza por entrada, las precondiciones de la integración y la lectura de la baseline por las fases de planificación.

El mecanismo de edición dirigida con verificación nativa y traza (REQ-24.50 a REQ-24.52) se especifica aquí una sola vez; la integración del delta de contexto técnico lo reutiliza (`sdd-archive-lifecycle`, REQ-24.55).

**Resultado esperado:** archive integra el cambio en los documentos declarados sin alterar un solo byte ajeno a los bloques editados, rechaza toda operación ambigua antes de escribir y deja una traza verificable por entrada.

## Términos

| Término | Definición |
|---|---|
| Bloque de requisito | Desde la línea de encabezado de un requisito hasta, sin incluirla, la siguiente línea de encabezado ATX de nivel menor o igual que el del requisito, sea requisito o no, fuera de bloques de código. Los encabezados de nivel mayor (por ejemplo h4 bajo un requisito h3) pertenecen al bloque. |
| Cola de secciones contiguas | Secciones no-requisito del mismo nivel que siguen directamente a un bloque, hasta el siguiente requisito de ese nivel o el siguiente encabezado de nivel menor. No pertenecen al bloque. |
| Familia de IDs | IDs que comparten el prefijo no numérico (por ejemplo `RF-`). |
| ID provisional | ID del encabezado de un requisito ADDED de las delta specs del cambio con identidad `id`. Archive lo sustituye por el ID definitivo en todo el texto que integra (REQ-24.59). |
| Referencia a un ID provisional | Aparición exacta del ID provisional como token completo en el texto a integrar: ni precedida ni seguida por letras, dígitos, `_` ni `-`, con distinción de mayúsculas y minúsculas. Todo token igual a un ID provisional del cambio es una referencia a él. |
| Texto integrado | Texto que archive escribe en los documentos del proyecto a partir del cambio: encabezados y cuerpos de los bloques de requisito, la nota de motivo que `--supersede` integra en un bloque conservado, las secciones del cambio en documentos de prosa y el texto de las entradas del delta de contexto. No incluye el texto preexistente de los documentos ni los ficheros del propio cambio. |
| Entrada | Cada requisito de la delta spec de un dominio de prosa o cada entrada del delta de contexto. |
| Traza de integración | Registro por entrada de lo que archive integró, en la carpeta del cambio y en el informe de archive. |
| Comprobación previa | Operación de solo lectura (`axiom sdd integrate check`) que aplica a las delta specs del cambio las validaciones de contenido que archive aplica antes de escribir (REQ-24.46), sin escribir nada y sin las precondiciones de git (REQ-24.49) ni la autorización de rutas externas (REQ-24.37). |

## Ejemplo de traza (no normativo)

| Identificador | Documento | Ancla | Acción | Estado |
|---|---|---|---|---|
| RF-9001 → RF-008 | 01_Requisitos_Funcionales.md | RF-008 | added | integrated |
| inc-99-ejemplo | 03_Modelo_Datos.md | inc-99-ejemplo | modified | integrated |
| inc-99-ejemplo | 04_Flujos_Negocio.md | inc-99-ejemplo | added | pending |

## Requerimientos de un vistazo

| ID | Requerimiento | Garantía principal |
|---|---|---|
| REQ-24.38 | Composición nativa intacta sin perfil | Byte a byte como hasta ahora |
| REQ-24.39 | Reconocimiento de requisitos e identidad | Por patrón, nivel e ID o título |
| REQ-24.40 | Rechazo de IDs duplicados | Operación ambigua rechazada, resto operable |
| REQ-24.41 | Frontera de bloque por nivel | Subsecciones dentro; contiguas fuera, con aviso |
| REQ-24.42 | Formato del delta, render y frontera del cuerpo | Delta nativo, encabezado por plantilla, cuerpo íntegro sin encabezados que cierren el bloque, con detección temprana |
| REQ-24.43 | ADDED con ubicación explícita | ID provisional, definitivo asignado en archive |
| REQ-24.44 | MODIFIED, REMOVED y RENAMED por ID | Cambiar el título es MODIFIED; no se renumera |
| REQ-24.45 | Fidelidad de bytes | LF, CRLF y BOM preservados |
| REQ-24.46 | Validación completa antes de escribir | Todo o nada |
| REQ-24.47 | `axiom sdd archive-compose` con `--cwd` y `--document` | Documento y perfil desde `axiom.yaml` |
| REQ-24.48 | Dominio no declarado | Rechazado; nunca crea `openspec/specs/<dominio>` |
| REQ-24.49 | Precondiciones de la integración | Git y árbol limpio; registros del proyecto intactos |
| REQ-24.50 | Integración dirigida de documentos de prosa | Sección del cambio por edición dirigida |
| REQ-24.51 | Verificación nativa | Presencia, anclas y alcance por diff |
| REQ-24.52 | Traza de integración | Por entrada, en el cambio y en el informe |
| REQ-24.53 | Modo solo registro | Entradas pendientes sin editar el documento |
| REQ-24.54 | Lectura de la baseline por las fases | Localizador del status, sin rutas fijas; `sdd-spec` ejecuta la comprobación previa |
| REQ-24.59 | Sustitución de los IDs provisionales | Todo el texto integrado usa los IDs definitivos; correspondencia única y en la traza |
| REQ-24.61 | Escenarios `include` u `omit` | El bloque del proyecto no impone el formato de escenarios si el documento lo declara |

## No cubre esta capacidad

- Actualizar `REGISTRO_INCREMENTOS.md` ni `REGISTRO_BUGS.md` del proyecto.
- Evaluar la corrección semántica de una edición en prosa: la verificación es determinista y solo comprueba presencia y alcance.
- Renumerar requisitos existentes.
- Normalizar los IDs duplicados de un documento: se exige previamente en el repositorio del proyecto.
- Elegir el actor que ejecuta la edición dirigida (decisión de diseño).

---

## 1. Capacidad: `spec-baseline-integration`

La composición generaliza la nativa de OpenSpec sin sustituirla: con perfil de requisito aplica identidad por ID y plantilla de render; sin perfil, conserva el comportamiento actual byte a byte. Los documentos de prosa se integran con edición dirigida y verificación nativa.

### Requirement: Composición Nativa Intacta sin Perfil (REQ-24.38)

Cuando el documento de la baseline no declara perfil de requisito (baseline OpenSpec por defecto), la composición y `axiom sdd archive-compose` DEBEN comportarse byte a byte como antes de esta capacidad: identidad por el texto exacto del nombre del requisito, orden de aplicación RENAMED, MODIFIED, REMOVED y ADDED, rechazo sin escribir ante un delta no aplicable y opciones `--supersede` y `--superseded-requirements`. Las pruebas nativas existentes de composición y de la CLI DEBEN pasar sin modificar sus aserciones.

#### Scenario: La salida nativa no cambia
- **DADO** una especificación canónica y un delta con ADDED, MODIFIED, REMOVED y RENAMED
- **CUANDO** se ejecuta `axiom sdd archive-compose --canonical <ruta> --delta <ruta>`
- **ENTONCES** la salida es idéntica byte a byte a la que producía la composición antes de esta capacidad
- **Y** las pruebas existentes de composición y de la CLI pasan sin modificarse

#### Scenario: Rechazo nativo sin escribir
- **DADO** un delta con un MODIFIED sobre un nombre de requisito inexistente
- **CUANDO** se ejecuta la composición con `--output <ruta>`
- **ENTONCES** finaliza con código distinto de cero nombrando la sección y el requisito
- **Y** no se escribe el fichero de salida

#### Scenario: `--supersede` conserva su comportamiento
- **DADO** un delta con un REMOVED y la opción `--supersede`
- **CUANDO** se compone
- **ENTONCES** el requisito se conserva marcado como superado, con su nota de motivo, como hasta ahora

---

### Requirement: Reconocimiento de Requisitos e Identidad según el Perfil (REQ-24.39)

Con perfil de requisito, la composición DEBE reconocer como requisitos las líneas de encabezado que cumplen `heading` y cuyo nivel coincide con `level`, fuera de bloques de código. La identidad de un requisito es su `id` (identidad `id`) o el texto de su `title` (identidad `text`). El nombre del requisito en el delta (el texto tras `### Requirement:`) DEBE ser el encabezado del proyecto sin los signos `#` iniciales ni el espacio que los sigue, de modo que los `#` del nivel, un espacio y ese nombre cumplan `heading`; si no lo cumple, la operación se rechaza nombrando el requisito y el patrón.

#### Scenario: Se reconocen solo los encabezados de requisito
- **DADO** un documento con `### RF-100: Alta`, una sección `### Notas` y un bloque de código con la línea `### RF-101: Falso`
- **CUANDO** se analiza con el perfil de `RF`
- **ENTONCES** el único requisito reconocido es `RF-100`

#### Scenario: Operación sobre el bloque correcto por ID
- **DADO** un documento con `RF-100` y `RF-101` y un delta MODIFIED para `RF-100: Alta de cliente`
- **CUANDO** se compone
- **ENTONCES** solo se sustituye el bloque de `RF-100`

#### Scenario: Nombre del delta incompatible con el patrón
- **DADO** un delta con el requisito `Reintento de pago` (sin ID)
- **CUANDO** se compone con el perfil de `RF`
- **ENTONCES** se rechaza nombrando el requisito y el patrón `heading`
- **Y** no se escribe ningún documento

#### Scenario: Identidad por texto
- **DADO** un perfil con `identity: text` y un delta MODIFIED cuyo título coincide con el de un requisito
- **CUANDO** se compone
- **ENTONCES** se sustituye el bloque con ese título exacto

#### Scenario: El nivel declarado filtra los encabezados
- **DADO** un perfil con `heading: '^#+ (?P<id>RF-\d+): (?P<title>.+)$'` y `level: 3`, y un documento con `### RF-100: A` y `#### RF-101: B`
- **CUANDO** se analiza
- **ENTONCES** solo `RF-100` es requisito y `#### RF-101: B` pertenece al bloque de `RF-100`

---

### Requirement: Rechazo de Operaciones sobre IDs Duplicados (REQ-24.40)

Si el documento contiene más de un encabezado de requisito con el mismo identificador (`id`, o el título con identidad `text`), toda operación MODIFIED, REMOVED o RENAMED sobre ese identificador y todo ADDED con ubicación relativa a él DEBE rechazarse nombrando el identificador y todas sus apariciones (línea y título), sin escribir el documento, e indicando como continuación la normalización previa de los IDs en el repositorio del proyecto. Las operaciones sobre identificadores no ambiguos DEBEN seguir siendo posibles aunque el documento contenga otros duplicados.

#### Scenario: Operación sobre un ID duplicado
- **DADO** un documento con dos encabezados `RF-254` de títulos distintos
- **CUANDO** se compone un MODIFIED para `RF-254`
- **ENTONCES** se rechaza nombrando `RF-254` y las dos apariciones con su línea y su título
- **Y** el documento conserva sus bytes originales

#### Scenario: ADDED relativo a un ID duplicado
- **DADO** el mismo documento y un ADDED con `(Placement: after RF-254)`
- **CUANDO** se compone
- **ENTONCES** se rechaza nombrando `RF-254` y sus apariciones

#### Scenario: Un ID único sigue siendo operable
- **DADO** un documento con `RF-254` duplicado y un `RF-300` único
- **CUANDO** se compone un MODIFIED para `RF-300`
- **ENTONCES** se aplica y el resto del documento conserva sus bytes

---

### Requirement: Frontera de Bloque por Nivel y Secciones No Requisito Contiguas (REQ-24.41)

El bloque de un requisito DEBE delimitarse según el término «Bloque de requisito». MODIFIED y REMOVED DEBEN actuar solo sobre el bloque y NO DEBEN sustituir ni eliminar las secciones de su cola de secciones contiguas, que se conservan byte a byte. La composición DEBE emitir, por cada operación MODIFIED o REMOVED sobre un requisito seguido de secciones contiguas, un aviso no bloqueante (en la salida de error y en la traza) que nombre cada una (encabezado y línea), para que el autor compruebe si son subsecciones del requisito o secciones independientes.

#### Scenario: Las subsecciones de nivel mayor pertenecen al bloque
- **DADO** un `RF-100` h3 con subsecciones h4 y un MODIFIED para `RF-100`
- **CUANDO** se compone
- **ENTONCES** el bloque sustituido incluye las subsecciones h4 anteriores
- **Y** el delta aporta el nuevo contenido completo del bloque

#### Scenario: Una sección contigua del mismo nivel se conserva con aviso
- **DADO** `RF-100` seguido de la sección no-requisito `### Notas de implantación` y después de `RF-101`
- **CUANDO** se compone un MODIFIED para `RF-100`
- **ENTONCES** `### Notas de implantación` conserva sus bytes
- **Y** se emite un aviso que nombra `### Notas de implantación` y su línea

#### Scenario: REMOVED conserva las secciones contiguas
- **DADO** el mismo documento
- **CUANDO** se compone un REMOVED para `RF-100`
- **ENTONCES** se elimina solo el bloque de `RF-100`, se conserva `### Notas de implantación` y se emite el aviso

#### Scenario: La frontera es un encabezado de nivel menor
- **DADO** el último requisito h3 de una sección seguido de un encabezado h2
- **CUANDO** se compone un MODIFIED para ese requisito
- **ENTONCES** el bloque termina justo antes del encabezado h2

#### Scenario: Un encabezado dentro de un bloque de código no corta el bloque
- **DADO** un requisito cuyo cuerpo contiene un bloque de código con una línea `## Ejemplo`
- **CUANDO** se compone un MODIFIED para él
- **ENTONCES** el bloque sustituido incluye el bloque de código completo

---

### Requirement: Formato del Delta, Render por Plantilla y Cuerpo Íntegro sin Encabezados que Cierren el Bloque (REQ-24.42)

El delta de un dominio con perfil DEBE conservar el formato nativo: secciones `## ADDED Requirements`, `## MODIFIED Requirements`, `## REMOVED Requirements` y `## RENAMED Requirements`, un bloque por requisito encabezado con `### Requirement: <nombre>` y, en REMOVED, la nota `(Reason: ...)`. Al integrar, el encabezado del bloque DEBE emitirse con la plantilla `render` sustituyendo `{id}` y `{title}`, y el cuerpo del requisito del delta (descripción y escenarios) DEBE pasar íntegro al bloque, tras el encabezado, salvo los escenarios cuando el documento declara `scenarios: omit` (REQ-24.61). Las líneas de nota `(Placement: ...)`, `(Previously: ...)`, `(Reason: ...)` y `(Migration: ...)` son metadatos del delta y NO DEBEN integrarse en el documento.

El cuerpo que se integra de un requisito ADDED o MODIFIED NO DEBE contener, fuera de bloques de código con vallas, ningún encabezado ATX de nivel menor o igual que el `level` del perfil: cerraría el bloque del requisito (REQ-24.41) y dejaría el resto del cuerpo fuera de él, de modo que un MODIFIED posterior no lo sustituiría. La regla se aplica al texto que se integra: con `scenarios: omit` (REQ-24.61) no cuentan los escenarios omitidos. La integración NO DEBE rebajar ni reescribir esos encabezados: la infracción se detecta, sin escribir nada, en tres puntos:

1. Al validar la declaración: con `scenarios: include`, un `level` mayor que 3 se rechaza porque los encabezados de escenario son de nivel 4 (REQ-24.60).
2. En las delta specs del cambio, mediante la comprobación previa de solo lectura (`axiom sdd integrate check`). La comprobación DEBE poder ejecutarse durante la fase de especificación y antes de la compuerta `spec`, sin exigir las precondiciones de git ni la autorización de rutas externas, y DEBE finalizar con código distinto de cero si detecta la infracción.
3. En el archivado, como última guarda: archive DEBE detenerse antes de escribir ningún documento (REQ-24.46) y el cambio permanece en `openspec/changes/`.

Las detecciones de los puntos 2 y 3 DEBEN nombrar el dominio, el requisito, el encabezado infractor con su línea y la continuación (rebajar el encabezado a un nivel mayor que `level` o convertirlo en texto).

#### Scenario: Encabezado renderizado y cuerpo íntegro
- **DADO** un MODIFIED con el nombre `RF-120: Alta de cliente con validación` y un cuerpo de tres párrafos
- **CUANDO** se compone con `render: '### {id}: {title}'`
- **ENTONCES** el bloque resultante comienza con `### RF-120: Alta de cliente con validación`
- **Y** contiene los tres párrafos del cuerpo sin alteración

#### Scenario: Los escenarios del delta se integran como subsecciones del bloque
- **DADO** un documento sin `scenarios` declarado (valor por defecto `include`) y un cuerpo con subsecciones de escenario de nivel h4
- **CUANDO** se integra en un perfil de nivel 3
- **ENTONCES** las subsecciones de escenario quedan dentro del bloque del requisito

#### Scenario: Las notas del delta no se integran
- **DADO** un bloque con `(Placement: after RF-002)` y `(Previously: se añadió el reintento)`
- **CUANDO** se integra
- **ENTONCES** el documento no contiene esas líneas de nota

#### Scenario: Un encabezado del cuerpo de nivel menor o igual que el del requisito se detecta en la comprobación previa
- **DADO** un documento con perfil de nivel 3 y una delta spec con un MODIFIED para `RF-120` cuyo cuerpo contiene, fuera de bloques de código, una línea `### Notas de implantación`
- **CUANDO** se ejecuta `axiom sdd integrate check --cwd <maestra> --change inc-99-ejemplo`
- **ENTONCES** el comando finaliza con código distinto de cero nombrando el dominio, el requisito `RF-120`, el encabezado `### Notas de implantación`, su línea y la continuación
- **Y** no se escribe ningún fichero

#### Scenario: La comprobación previa se puede ejecutar durante la fase de especificación
- **DADO** un cambio en fase de especificación, con las delta specs sin confirmar en git, sin la compuerta `spec` registrada y sin autorizaciones por cambio
- **CUANDO** se ejecuta `axiom sdd integrate check --cwd <maestra> --change inc-99-ejemplo`
- **ENTONCES** la comprobación se ejecuta sin exigir las precondiciones de git ni la autorización de rutas externas
- **Y** no escribe ningún fichero, tampoco en la carpeta del cambio

#### Scenario: El archivado repite la comprobación como última guarda
- **DADO** el mismo MODIFIED con `### Notas de implantación` en el cuerpo, que no se detectó antes
- **CUANDO** se solicita el archivado
- **ENTONCES** archive se detiene antes de escribir ningún documento, nombrando el dominio, el requisito, el encabezado y su línea
- **Y** el cambio permanece en `openspec/changes/`

#### Scenario: Los encabezados de nivel mayor y los de bloques de código no son infracción
- **DADO** un perfil de nivel 3 y un MODIFIED cuyo cuerpo contiene escenarios `#### Scenario:` y un bloque de código con una línea `## Ejemplo`
- **CUANDO** se ejecuta la comprobación previa
- **ENTONCES** no se reporta ninguna infracción

#### Scenario: Con `omit` no cuentan los escenarios omitidos
- **DADO** un documento con perfil de nivel 4 y `scenarios: omit`, y un ADDED cuya descripción no contiene encabezados y que incluye dos escenarios `#### Scenario:`
- **CUANDO** se ejecuta la comprobación previa
- **ENTONCES** no se reporta ninguna infracción, porque los escenarios omitidos no se integran

---

### Requirement: Requisitos ADDED con Ubicación Explícita e ID Definitivo Asignado en Archive (REQ-24.43)

Cada requisito ADDED de un dominio con identidad `id` DEBE declarar su ubicación con una nota `(Placement: after <ID>)` (inserción inmediatamente tras el bloque del requisito `<ID>` y su cola de secciones contiguas) o `(Placement: section "<título>")` (inserción al final de la sección cuyo encabezado tiene ese título y un nivel inmediatamente superior al de los requisitos; el título DEBE ser único entre los encabezados de ese nivel del documento). La ubicación NO DEBE deducirse del orden de los IDs. Un ADDED sin ubicación, con un `<ID>` inexistente o ambiguo, o con un título de sección ausente o repetido, DEBE rechazarse. El ID de un requisito ADDED en el delta es siempre provisional: DEBE cumplir `heading` y NO DEBE coincidir con un ID existente del documento. Archive DEBE sustituirlo por el ID definitivo: el siguiente número libre posterior al mayor existente en su familia de IDs, con el relleno de ceros del ID de mayor número de la familia; los ADDED de un mismo delta reciben IDs consecutivos en el orden del delta. El mapeo del ID provisional al definitivo DEBE registrarse en la traza (REQ-24.52). Si la traza del cambio ya registra el mapeo de un ADDED, la composición DEBE rechazarlo como ya integrado, de modo que reaplicar un delta no duplique requisitos.

#### Scenario: ADDED tras un requisito existente con IDs no ordenados
- **DADO** un documento con `RF-001`, `RF-002`, `RF-007` y `RF-003` en ese orden y un ADDED `RF-9001` con `(Placement: after RF-002)`
- **CUANDO** archive integra el cambio
- **ENTONCES** el nuevo requisito se inserta justo tras el bloque de `RF-002`
- **Y** recibe el ID definitivo `RF-008`
- **Y** la traza registra el mapeo `RF-9001` a `RF-008`

#### Scenario: La cola de secciones contiguas no se separa de su requisito
- **DADO** `RF-002` seguido de la sección no-requisito `### Notas` y después de `RF-007`, y un ADDED con `(Placement: after RF-002)`
- **CUANDO** se compone
- **ENTONCES** el orden resultante es `RF-002`, `### Notas`, el nuevo requisito y `RF-007`

#### Scenario: ADDED sin ubicación
- **DADO** un ADDED sin nota `(Placement: ...)`
- **CUANDO** se compone
- **ENTONCES** se rechaza indicando que la ubicación debe ser explícita
- **Y** no se escribe el documento

#### Scenario: Ubicación al final de una sección
- **DADO** un documento con una única sección h2 «Pagos» y un ADDED con `(Placement: section "Pagos")`
- **CUANDO** se compone
- **ENTONCES** el nuevo requisito se inserta al final de la sección «Pagos», antes del siguiente encabezado h2

#### Scenario: Título de sección repetido o inexistente
- **DADO** un ADDED con `(Placement: section "Notas")` y un documento con dos secciones h2 «Notas», o con ninguna
- **CUANDO** se compone
- **ENTONCES** se rechaza nombrando el título y el número de apariciones

#### Scenario: ID provisional que colisiona
- **DADO** un ADDED cuyo ID provisional `RF-002` ya existe en el documento
- **CUANDO** se compone
- **ENTONCES** se rechaza indicando que el ID provisional no debe coincidir con un ID existente

#### Scenario: Varios ADDED reciben IDs consecutivos
- **DADO** un delta con dos ADDED y el mayor ID existente `RF-007`
- **CUANDO** archive integra el cambio
- **ENTONCES** reciben `RF-008` y `RF-009` en el orden del delta

#### Scenario: Reaplicar el delta no duplica
- **DADO** un cambio cuya traza ya registra `RF-9001` a `RF-008`
- **CUANDO** se vuelve a componer el mismo delta
- **ENTONCES** el ADDED se rechaza como ya integrado
- **Y** el documento no cambia

---

### Requirement: MODIFIED, REMOVED y RENAMED con Identidad por ID (REQ-24.44)

Con identidad `id`: MODIFIED DEBE sustituir el bloque del requisito cuyo `id` coincide y PUEDE cambiar su título, pero NO su `id`; un MODIFIED cuyo `id` no existe DEBE rechazarse. REMOVED DEBE eliminar el bloque o, con `--supersede`, conservarlo con el título renderizado seguido de `[SUPERSEDED / DEPRECADO]` y la nota de motivo como en la composición nativa. RENAMED DEBE rechazarse indicando que cambiar el título es un MODIFIED y que renumerar no está soportado. Con identidad `text`, las cuatro operaciones DEBEN comportarse como la composición nativa tomando el título como nombre.

#### Scenario: MODIFIED con cambio de título
- **DADO** `RF-120: Alta de cliente` y un MODIFIED `RF-120: Alta de cliente con validación`
- **CUANDO** se compone
- **ENTONCES** el bloque pasa a titularse `### RF-120: Alta de cliente con validación` con el `id` sin cambios

#### Scenario: MODIFIED sobre un ID inexistente
- **DADO** un MODIFIED para `RF-999` que no existe
- **CUANDO** se compone
- **ENTONCES** se rechaza nombrando la sección y el requisito `RF-999`
- **Y** no se escribe el documento

#### Scenario: REMOVED elimina el bloque
- **DADO** un REMOVED para `RF-120` con `(Reason: ...)`
- **CUANDO** se compone
- **ENTONCES** el documento deja de contener el bloque de `RF-120`

#### Scenario: REMOVED con `--supersede`
- **DADO** el mismo REMOVED con `--supersede`
- **CUANDO** se compone
- **ENTONCES** el bloque se conserva con el título seguido de `[SUPERSEDED / DEPRECADO]` y la nota de motivo

#### Scenario: RENAMED con identidad por ID
- **DADO** un RENAMED con identidad `id`
- **CUANDO** se compone
- **ENTONCES** se rechaza indicando que se use MODIFIED para cambiar el título y que renumerar no está soportado

#### Scenario: RENAMED con identidad por texto
- **DADO** un perfil con `identity: text` y un RENAMED con `(Reason: ...)`
- **CUANDO** se compone
- **ENTONCES** se aplica como en la composición nativa

---

### Requirement: Fidelidad de Bytes al Componer (REQ-24.45)

La composición DEBE preservar byte a byte todo lo que no pertenece a un bloque editado: finales de línea (LF y CRLF), marca de orden de bytes (BOM) UTF-8, contenido entre bloques y final del fichero. Los bloques insertados o sustituidos DEBEN emitirse con el final de línea del documento (el de la línea de encabezado que sustituyen o, en ADDED, el del encabezado del bloque de referencia) y NUNCA DEBEN introducir un BOM; si el fichero tiene BOM, se conserva. Si el documento no termina en salto de línea y el último bloque no se edita, el final del fichero se conserva.

#### Scenario: Round-trip en LF, CRLF y CRLF con BOM
- **DADO** un documento fixture con encabezados `### RF-NNN: título`, subsecciones `###` no-requisito y variantes LF, CRLF y CRLF con BOM
- **CUANDO** se compone un delta con ADDED, MODIFIED y REMOVED
- **ENTONCES** la salida es idéntica byte a byte a la original salvo en los bloques editados, en las tres variantes

#### Scenario: Los bloques editados usan el final de línea del documento
- **DADO** un documento CRLF y un delta en LF
- **CUANDO** se compone
- **ENTONCES** los bloques insertados o sustituidos usan CRLF y el fichero no contiene saltos LF sueltos

#### Scenario: El BOM se conserva y no se añade
- **DADO** un documento con BOM y otro sin BOM
- **CUANDO** se componen sendos deltas
- **ENTONCES** el primero conserva su BOM y el segundo no gana BOM

#### Scenario: Final de fichero sin salto de línea
- **DADO** un documento cuyo último bloque no termina en salto de línea y un delta que no lo edita
- **CUANDO** se compone
- **ENTONCES** el final del fichero conserva sus bytes

---

### Requirement: Validación Completa antes de Escribir (REQ-24.46)

Todas las operaciones de todos los documentos con perfil de requisito del cambio DEBEN validarse y componerse en memoria antes de escribir ningún documento de la integración (de composición, dirigida o de contexto): o se aplican todas las composiciones o ninguna. Un rechazo NO DEBE dejar la baseline parcialmente modificada y DEBE nombrar la sección, el requisito y el motivo. La integración dirigida y la del contexto solo comienzan cuando todas las composiciones han tenido éxito.

#### Scenario: Un dominio falla y ninguno se modifica
- **DADO** un cambio con deltas para dos dominios, el segundo con un MODIFIED sobre un ID inexistente
- **CUANDO** se ejecuta el archivado
- **ENTONCES** se rechaza nombrando el dominio, la sección y el requisito
- **Y** ningún documento de la baseline cambia

#### Scenario: Todas las operaciones válidas se aplican
- **DADO** un cambio con deltas válidos para dos dominios
- **CUANDO** se ejecuta el archivado
- **ENTONCES** ambos documentos se sustituyen por su composición

#### Scenario: La integración dirigida no empieza si una composición falla
- **DADO** un cambio con una composición inválida y un dominio de prosa con `integration: directed`
- **CUANDO** se ejecuta el archivado
- **ENTONCES** se rechaza nombrando la composición inválida
- **Y** no se edita ningún documento, tampoco el de prosa

---

### Requirement: Resolución de Documento y Perfil en `axiom sdd archive-compose` (REQ-24.47)

`axiom sdd archive-compose` DEBE aceptar `--cwd <dir>` y `--document <dominio>`: resuelve con el primitivo de la capacidad `planning-home-resolution` el `workspaceRoot` y, con la declaración de `axiom.yaml`, la ruta del documento canónico y su perfil. Sin `baseline:` declarada, `--document <dominio>` DEBE resolver `<planning home>/specs/<dominio>/spec.md` y aplicar la composición nativa. `--document` NO DEBE usarse junto con `--canonical`. Con `--cwd`, las rutas relativas de `--delta` y `--output` DEBEN resolverse contra `workspaceRoot`. `--canonical`, `--delta`, `--output`, `--supersede` y `--superseded-requirements` conservan su significado. La skill `sdd-archive` DEBE invocar el comando sin plantillar rutas en shell.

#### Scenario: Documento y perfil desde la declaración
- **DADO** una baseline con el dominio `requisitos-funcionales` con perfil de requisito
- **CUANDO** se ejecuta `axiom sdd archive-compose --cwd <maestra> --document requisitos-funcionales --delta <ruta>`
- **ENTONCES** compone sobre el documento declarado con su perfil

#### Scenario: Sin baseline declarada se usa la ruta OpenSpec
- **DADO** un `axiom.yaml` sin `baseline:`
- **CUANDO** se ejecuta `axiom sdd archive-compose --cwd <raíz> --document workspace-topology --delta <ruta>`
- **ENTONCES** compone sobre `<planning home>/specs/workspace-topology/spec.md` con la composición nativa

#### Scenario: `--document` junto con `--canonical`
- **DADO** ambas opciones en la misma invocación
- **CUANDO** se ejecuta
- **ENTONCES** se rechaza indicando que son excluyentes
- **Y** no se escribe nada

#### Scenario: Dominio de prosa
- **DADO** un dominio declarado sin perfil de requisito
- **CUANDO** se ejecuta `--document` sobre él
- **ENTONCES** se rechaza indicando que el dominio es de prosa y que se integra por edición dirigida o solo registro

#### Scenario: La skill no plantilla rutas
- **DADO** el texto de la skill `sdd-archive`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** invoca `archive-compose` con `--cwd` y `--document` y no compone rutas `openspec/specs/{domain}` en shell para la baseline del proyecto

---

### Requirement: Un Dominio No Declarado se Rechaza (REQ-24.48)

Con baseline del proyecto declarada, un dominio de delta spec que no figura en `baseline.documents` NO DEBE integrarse ni promoverse: archive y `archive-compose` DEBEN rechazarlo nombrando el dominio y la continuación (declararlo en `axiom.yaml` o retirar la delta), y NO DEBEN crear `openspec/specs/<dominio>`.

#### Scenario: Dominio no declarado
- **DADO** una delta en `specs/nuevo-dominio/spec.md` y una baseline que no declara `nuevo-dominio`
- **CUANDO** se ejecuta el archivado
- **ENTONCES** se rechaza nombrando el dominio y la continuación
- **Y** no existe `openspec/specs/nuevo-dominio`

#### Scenario: Un dominio no declarado impide todo el archivado
- **DADO** un cambio con un dominio declarado y otro no declarado
- **CUANDO** se ejecuta el archivado
- **ENTONCES** se rechaza antes de modificar ningún documento (REQ-24.46)

---

### Requirement: Precondiciones de la Integración (REQ-24.49)

Antes de modificar ningún documento de la baseline ni ruta mantenida, archive DEBE comprobar que (a) cada repositorio que contiene un documento afectado es un repositorio git y (b) los documentos afectados no tienen cambios sin confirmar ni están sin seguimiento. Si alguna comprobación falla, archive DEBE detenerse sin modificar nada, nombrando los ficheros afectados y la continuación (confirmar, guardar o descartar los cambios, o versionar el repositorio), y el cambio permanece en `openspec/changes/`. Los conflictos con ediciones concurrentes se resuelven en git y no en Axiom. Archive NO DEBE actualizar `REGISTRO_INCREMENTOS.md`, `REGISTRO_BUGS.md` ni ningún documento no declarado. Los documentos que archive crearía (REQ-24.55) quedan excluidos de (b).

#### Scenario: Documento afectado con cambios sin confirmar
- **DADO** un documento de la baseline afectado por el cambio con modificaciones sin confirmar
- **CUANDO** se ejecuta el archivado
- **ENTONCES** archive se detiene nombrando el fichero y la continuación
- **Y** ningún documento se modifica

#### Scenario: Cambios sin confirmar en documentos no afectados
- **DADO** un documento de la baseline que el cambio no toca con modificaciones sin confirmar
- **CUANDO** se ejecuta el archivado
- **ENTONCES** archive no se detiene por ese documento

#### Scenario: Repositorio que no es git
- **DADO** un documento afectado dentro de un directorio que no es un repositorio git
- **CUANDO** se ejecuta el archivado
- **ENTONCES** archive se detiene indicando que la verificación por diff requiere un repositorio git y cómo versionarlo

#### Scenario: Los registros del proyecto no se tocan
- **DADO** un proyecto con `REGISTRO_INCREMENTOS.md` y `REGISTRO_BUGS.md` no declarados
- **CUANDO** archive integra un cambio
- **ENTONCES** el diff no contiene cambios en esos ficheros

---

### Requirement: Integración Dirigida de Documentos de Prosa (REQ-24.50)

Para un dominio de prosa con `integration: directed`, archive DEBE integrar la delta spec del dominio mediante una edición dirigida por agente: por cada requisito de la delta spec (ADDED, MODIFIED, REMOVED o RENAMED), el agente DEBE crear o editar, en el documento declarado, la sección del cambio, titulada con el identificador del cambio, y la entrada correspondiente DEBE quedar en la traza (REQ-24.52). El agente DEBE editar únicamente los documentos declarados para la integración (REQ-24.51) y DEBE reutilizar la sección del cambio si ya existe, de modo que una integración repetida no duplique secciones.

#### Scenario: Delta de un dominio de prosa
- **DADO** un dominio `modelo-de-datos` con `integration: directed` y una delta con dos requisitos
- **CUANDO** archive integra el cambio `inc-99-ejemplo`
- **ENTONCES** `03_Modelo_Datos.md` contiene una sección titulada con `inc-99-ejemplo`
- **Y** la traza contiene una entrada por cada uno de los dos requisitos

#### Scenario: Integración repetida
- **DADO** un documento que ya contiene la sección del cambio
- **CUANDO** se repite la integración
- **ENTONCES** se edita la sección existente y el documento contiene exactamente una sección del cambio

---

### Requirement: Verificación Nativa de la Integración Dirigida (REQ-24.51)

Tras la edición, una verificación nativa y determinista (sin modelo) DEBE comprobar: (a) presencia: cada entrada del cambio (requisito de un dominio de prosa dirigido o entrada del delta de contexto) figura en la traza; (b) anclas: cada ancla citada existe tras la edición exactamente una vez, salvo en la acción `remove`, donde ya no existe; (c) alcance: el diff de cada repositorio afectado solo toca los documentos de los dominios del cambio con integración `compose` o `directed` y los documentos nombrados en las entradas del delta de contexto, salvo los ficheros de la propia carpeta del cambio, y ninguna ruta del modo `reference` ni documento de un dominio con integración `register`; con la baseline OpenSpec por defecto, los documentos de los dominios del cambio son `<planning home>/specs/<dominio>/spec.md` y el conjunto permitido incluye además `<planning home>/INDEX.md`, que `axiom archive sync` regenera en el flujo de archive por defecto (REQ-21.17); (d) la traza existe y es analizable; y (e) sustitución de IDs: el texto que la integración añade a los documentos no conserva ningún ID provisional del cambio distinto de su ID definitivo (REQ-24.59). Si alguna comprobación falla, la verificación DEBE finalizar con código distinto de cero nombrando la entrada y la comprobación, y archive DEBE detenerse sin mover el cambio a `openspec/changes/archive/`. La verificación NO evalúa la corrección semántica de la edición.

#### Scenario: Verificación superada
- **DADO** una edición dirigida con todas las entradas en la traza, anclas únicas y un diff limitado a los documentos declarados
- **CUANDO** se ejecuta la verificación nativa
- **ENTONCES** finaliza con código `0`

#### Scenario: Entrada ausente de la traza
- **DADO** un requisito de la delta sin entrada en la traza
- **CUANDO** se ejecuta la verificación
- **ENTONCES** falla nombrando el requisito y la comprobación de presencia

#### Scenario: Ancla inexistente o repetida
- **DADO** una entrada cuya ancla no existe tras la edición, o existe dos veces
- **CUANDO** se ejecuta la verificación
- **ENTONCES** falla nombrando la entrada y el ancla

#### Scenario: El diff toca un documento no declarado
- **DADO** una edición que modifica `REGISTRO_INCREMENTOS.md`
- **CUANDO** se ejecuta la verificación
- **ENTONCES** falla nombrando el fichero y la comprobación de alcance

#### Scenario: Con la baseline por defecto el alcance incluye `INDEX.md`
- **DADO** una baseline OpenSpec por defecto y un archivado cuyo diff contiene `<planning home>/specs/<dominio>/spec.md` de un dominio del cambio y `<planning home>/INDEX.md` regenerado por `axiom archive sync`
- **CUANDO** se ejecuta la verificación nativa
- **ENTONCES** la comprobación de alcance se supera

#### Scenario: El diff toca una ruta de referencia
- **DADO** una edición que modifica un fichero bajo `reference.paths`
- **CUANDO** se ejecuta la verificación
- **ENTONCES** falla nombrando el fichero

#### Scenario: Un ID provisional sin sustituir en el texto añadido
- **DADO** una edición dirigida cuyo texto añadido contiene `RF-9001`, ID provisional del cambio con ID definitivo `RF-008`
- **CUANDO** se ejecuta la verificación nativa
- **ENTONCES** falla nombrando `RF-9001`, su ID definitivo y el fichero
- **Y** archive se detiene sin mover el cambio a `openspec/changes/archive/`

#### Scenario: Tras un fallo el cambio permanece activo
- **DADO** una verificación fallida
- **CUANDO** archive se detiene
- **ENTONCES** el cambio permanece en `openspec/changes/` y no existe su carpeta bajo `openspec/changes/archive/`

---

### Requirement: Traza de Integración por Entrada (REQ-24.52)

Archive DEBE registrar una traza por entrada, en un fichero dentro de la carpeta del cambio (que se archiva con él; el nombre lo fija el diseño) y en una sección del informe de archive. Cada entrada DEBE incluir el identificador, el documento, el ancla, la acción (`added`, `modified`, `removed` o `created`) y el estado (`integrated` o `pending`). Para los requisitos ADDED con perfil, la traza DEBE incluir el mapeo del ID provisional al definitivo; para «Sin impacto» del delta de contexto, la decisión y su motivo. La traza cubre todo lo que archive integra: composición, edición dirigida y contexto mantenido.

#### Scenario: Traza en la carpeta del cambio y en el informe
- **DADO** un archivado que integra un requisito con perfil, una sección de prosa y una entrada de contexto
- **CUANDO** termina la integración
- **ENTONCES** la carpeta del cambio contiene la traza con tres entradas
- **Y** el informe de archive contiene una sección con las mismas tres entradas

#### Scenario: Mapeo de IDs provisionales
- **DADO** un ADDED `RF-9001` integrado como `RF-008`
- **CUANDO** se inspecciona la traza
- **ENTONCES** la entrada registra `RF-9001` a `RF-008`, el documento, el ancla `RF-008`, la acción `added` y el estado `integrated`

#### Scenario: Decisión «Sin impacto»
- **DADO** un delta de contexto con «Sin impacto» y su motivo
- **CUANDO** archive lo registra
- **ENTONCES** la traza y el informe contienen la decisión y el motivo

---

### Requirement: Modo Solo Registro para Documentos de Prosa (REQ-24.53)

Para un dominio de prosa con `integration: register`, archive NO DEBE editar el documento: DEBE registrar cada entrada con estado `pending` (documento, ancla propuesta y acción pendiente) para que el flujo del proyecto la integre. El diff de archive NO DEBE incluir ese documento.

#### Scenario: Documento en modo solo registro
- **DADO** un dominio `flujos-de-negocio` con `integration: register` y una delta con un requisito
- **CUANDO** archive integra el cambio
- **ENTONCES** `04_Flujos_Negocio.md` conserva sus bytes
- **Y** la traza contiene una entrada con estado `pending`, el documento, el ancla propuesta y la acción pendiente

#### Scenario: Mezcla de modos en un mismo cambio
- **DADO** un cambio con un dominio `directed` y otro `register`
- **CUANDO** archive integra el cambio
- **ENTONCES** edita solo el documento `directed` y registra como `pending` las entradas del documento `register`

---

### Requirement: Lectura de la Baseline por las Fases de Planificación (REQ-24.54)

Las skills `sdd-explore`, `sdd-propose` y `sdd-spec` (y, como DEBERÍA, `sdd-onboard`) y la convención compartida DEBEN obtener la baseline del campo `baseline` del status cuando existe, leyendo los documentos declarados de los dominios afectados, y NO DEBEN contener rutas `openspec/specs/` fijas fuera de la rama de baseline por defecto. Sin `baseline`, leen `<planning home>/specs/<dominio>/spec.md`. Con baseline del proyecto con perfil de requisito, `sdd-spec` DEBE redactar los requisitos MODIFIED y REMOVED con los IDs existentes leídos de la baseline y los ADDED con un ID provisional y la nota `(Placement: ...)`. Con esa misma baseline, `sdd-spec` DEBE además ejecutar la comprobación previa de las delta specs (`axiom sdd integrate check`, REQ-24.42) antes de dar por terminada la fase y corregir las infracciones que reporte.

#### Scenario: Las skills toman la baseline del localizador
- **DADO** los textos de `sdd-explore`, `sdd-propose` y `sdd-spec`
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** indican leer los documentos del campo `baseline` del status cuando existe
- **Y** no contienen rutas `openspec/specs/` fijas fuera de la rama por defecto

#### Scenario: Sin baseline del proyecto se mantiene la lectura actual
- **DADO** un status sin `baseline`
- **CUANDO** `sdd-spec` lee la especificación viva de un dominio
- **ENTONCES** lee `<planning home>/specs/<dominio>/spec.md`

#### Scenario: `sdd-spec` redacta el delta en el formato del proyecto
- **DADO** una baseline del proyecto con perfil `RF` y un cambio que modifica `RF-120` y añade un requisito
- **CUANDO** `sdd-spec` redacta el delta
- **ENTONCES** el MODIFIED usa el nombre `RF-120: <título>` con el ID existente
- **Y** el ADDED usa un ID provisional que cumple el patrón y la nota `(Placement: after <ID>)`

#### Scenario: `sdd-spec` ejecuta la comprobación previa
- **DADO** el texto de la skill `sdd-spec`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica ejecutar `axiom sdd integrate check` con una baseline del proyecto con perfil de requisito, antes de dar por terminada la fase, y corregir las infracciones que reporte

---

### Requirement: Sustitución de los IDs Provisionales por los Definitivos en Todo el Texto Integrado (REQ-24.59)

Archive DEBE sustituir cada ID provisional del cambio por su ID definitivo (REQ-24.43) en todo el texto integrado: los encabezados y los cuerpos de los bloques de requisito, incluidas las referencias de un requisito a otro (también entre dos requisitos nuevos del mismo cambio) y la nota de motivo que `--supersede` integra en un bloque conservado; las secciones del cambio integradas en documentos de prosa; y el texto de las entradas del delta de contexto integrado en las rutas mantenidas. Los IDs provisionales del cambio son los de los encabezados de los requisitos ADDED de sus delta specs con identidad `id`, y todo token igual a uno de ellos en el texto integrado es una referencia a él.

La correspondencia de IDs provisionales a definitivos DEBE calcularse una sola vez, de forma determinista, como parte de la validación previa a cualquier escritura (REQ-24.46), registrarse en la traza (REQ-24.52) para todos los IDs provisionales del cambio y estar disponible para la edición dirigida y la edición del contexto antes de que escriban. El texto que se escribe en los documentos DEBE contener los IDs definitivos, y la verificación nativa (REQ-24.51) DEBE rechazar el texto añadido que conserve un ID provisional distinto de su ID definitivo.

La sustitución NO DEBE modificar ninguna aparición de un ID en el texto preexistente de los documentos del proyecto ni en los ficheros del propio cambio, que se archivan sin alteración.

Archive DEBE detenerse sin escribir nada, nombrando el ID provisional y la continuación (renombrarlo en las delta specs del cambio o normalizar antes el documento del proyecto), cuando un ID provisional no sea resoluble o sea ambiguo: (a) no termina en un sufijo numérico; (b) su familia de IDs no tiene ningún ID existente en el documento destino; (c) lo declaran dos o más requisitos ADDED del cambio, en el mismo documento o en documentos distintos; o (d) coincide con el ID definitivo asignado a otro requisito del cambio.

#### Scenario: Referencia cruzada entre dos requisitos nuevos
- **DADO** un documento cuyo mayor ID es `RF-007` y un cambio con dos ADDED, `RF-9001` y `RF-9002`, donde el cuerpo de `RF-9002` dice «depende de RF-9001»
- **CUANDO** archive integra el cambio
- **ENTONCES** `RF-9001` pasa a `RF-008` y `RF-9002` pasa a `RF-009`
- **Y** el cuerpo del bloque de `RF-009` dice «depende de RF-008»
- **Y** el documento no contiene `RF-9001` ni `RF-9002`
- **Y** la traza registra ambas correspondencias

#### Scenario: Referencia dentro de una sección de un documento de prosa
- **DADO** un dominio de prosa `modelo-de-datos` con `integration: directed`, cuya delta spec cita `RF-9001`, y `RF-9001` ID provisional de un ADDED de `requisitos-funcionales` con ID definitivo `RF-008`
- **CUANDO** archive integra el cambio
- **ENTONCES** la sección del cambio en `03_Modelo_Datos.md` contiene `RF-008` y no contiene `RF-9001`
- **Y** la verificación nativa supera la comprobación de sustitución de IDs

#### Scenario: Referencia dentro del delta de contexto
- **DADO** un delta de contexto con una entrada `modify` cuyo motivo dice «implementa RF-9001», y `RF-9001` con ID definitivo `RF-008`
- **CUANDO** archive integra el cambio
- **ENTONCES** el texto integrado en la sección ancla del documento mantenido cita `RF-008` y no `RF-9001`
- **Y** el delta de contexto de la carpeta del cambio conserva `RF-9001` sin alteración

#### Scenario: Se sustituye también la nota de motivo de un bloque superado
- **DADO** un REMOVED con `--supersede` cuya nota `(Reason: ...)` dice «superado por RF-9001», y `RF-9001` con ID definitivo `RF-008`
- **CUANDO** archive integra el cambio
- **ENTONCES** la nota de motivo integrada en el bloque conservado dice «superado por RF-008»

#### Scenario: Las apariciones ajenas al cambio no se tocan
- **DADO** un documento cuya sección preexistente menciona `RF-9001` con otro sentido y un cambio cuyo ADDED tiene el ID provisional `RF-9001`
- **CUANDO** archive integra el cambio
- **ENTONCES** la sección preexistente conserva sus bytes
- **Y** solo el texto integrado usa el ID definitivo
- **Y** los ficheros del cambio archivado conservan `RF-9001`

#### Scenario: Un ID provisional se sustituye solo como token completo
- **DADO** un ADDED `RF-9001` con ID definitivo `RF-008` y un texto integrado que contiene `RF-9001`, `RF-90012` y `ARF-9001`
- **CUANDO** archive integra el cambio
- **ENTONCES** solo la primera aparición pasa a `RF-008` y las otras dos conservan su texto

#### Scenario: Un ID provisional sin sufijo numérico detiene el archivado
- **DADO** un perfil cuyo `heading` admite `RF-NUEVO` y un ADDED con ese ID provisional
- **CUANDO** se ejecuta el archivado
- **ENTONCES** archive se detiene nombrando `RF-NUEVO` y la continuación de renombrarlo
- **Y** no se escribe ningún documento y el cambio permanece en `openspec/changes/`

#### Scenario: Una familia de IDs sin ningún ID existente detiene el archivado
- **DADO** un perfil cuyo `heading` admite las familias `RF-` y `NFR-`, un documento sin ningún ID `NFR-` y un ADDED con el ID provisional `NFR-9001`
- **CUANDO** se ejecuta el archivado
- **ENTONCES** archive se detiene nombrando `NFR-9001` y la continuación
- **Y** no se escribe ningún documento

#### Scenario: Un ID provisional declarado dos veces es ambiguo
- **DADO** dos ADDED del cambio con el mismo ID provisional `RF-9001`, en el mismo documento o en documentos distintos
- **CUANDO** se ejecuta el archivado
- **ENTONCES** archive se detiene nombrando `RF-9001` y los dos requisitos que lo declaran
- **Y** no se escribe ningún documento

#### Scenario: Un ID provisional que coincide con el definitivo de otro requisito
- **DADO** un documento cuyo mayor ID es `RF-511` y dos ADDED, en este orden, con los IDs provisionales `RF-513` y `RF-512`
- **CUANDO** se ejecuta el archivado
- **ENTONCES** archive se detiene nombrando ambos IDs porque la sustitución sería ambigua
- **Y** no se escribe ningún documento

---

### Requirement: Escenarios de la Delta Spec en el Bloque del Proyecto: `include` y `omit` (REQ-24.61)

Para un documento con perfil de requisito, al integrar un requisito ADDED o MODIFIED, `scenarios` (REQ-24.60) determina qué ocurre con los escenarios de la delta spec. Con `include` (valor por defecto), el cuerpo del requisito pasa íntegro al bloque, escenarios incluidos (REQ-24.42). Con `omit`, archive integra el encabezado renderizado y la descripción del requisito sin sus escenarios: un escenario es el fragmento que comienza en una línea `#### Scenario:` y termina antes del siguiente encabezado de nivel menor o igual que 4, o al final del cuerpo, fuera de bloques de código, y la descripción es el resto del cuerpo. Con `omit`, los escenarios permanecen en la delta spec del cambio archivado, que no se modifica, y la traza (REQ-24.52) DEBE registrar, por cada entrada cuyo requisito contenía escenarios, que se omitieron. REMOVED no se ve afectado. La baseline OpenSpec por defecto (sin `baseline:`) no declara `scenarios` y SIEMPRE incluye los escenarios.

#### Scenario: Por defecto los escenarios se integran
- **DADO** un documento con perfil de requisito sin `scenarios` declarado y un MODIFIED con descripción y dos escenarios
- **CUANDO** archive integra el cambio
- **ENTONCES** el bloque del proyecto contiene la descripción y los dos escenarios

#### Scenario: Con `omit` solo se integra la descripción
- **DADO** un documento con `scenarios: omit` y un ADDED con descripción y tres escenarios
- **CUANDO** archive integra el cambio
- **ENTONCES** el bloque del proyecto contiene el encabezado renderizado y la descripción
- **Y** no contiene ninguna línea `#### Scenario:` ni el contenido de los tres escenarios

#### Scenario: Los escenarios omitidos permanecen en el cambio archivado
- **DADO** la misma integración
- **CUANDO** archive transfiere el cambio a `openspec/changes/archive/`
- **ENTONCES** la delta spec archivada conserva sus tres escenarios byte a byte

#### Scenario: La traza registra la omisión
- **DADO** la misma integración y un segundo requisito del delta sin escenarios
- **CUANDO** se inspecciona la traza
- **ENTONCES** la entrada del requisito con escenarios registra que sus escenarios se omitieron
- **Y** la entrada del requisito sin escenarios no registra omisión alguna

#### Scenario: Un encabezado de nivel 4 que no es un escenario se conserva
- **DADO** un documento con perfil de nivel 3 y `scenarios: omit`, y un cuerpo con la descripción, un escenario y, a continuación, un encabezado `#### Notas de implantación` con su texto
- **CUANDO** archive integra el cambio
- **ENTONCES** el escenario se omite
- **Y** `#### Notas de implantación` y su texto se integran en el bloque

#### Scenario: La baseline OpenSpec por defecto no se ve afectada
- **DADO** un `axiom.yaml` sin `baseline:` y un cambio con un MODIFIED con descripción y escenarios
- **CUANDO** archive integra el cambio en `openspec/specs/`
- **ENTONCES** la especificación viva incorpora el requisito con sus escenarios, como hasta ahora
