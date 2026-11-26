# Especificación de requirement-identifiers

## Purpose

Puerto único de identificadores de requisito (C-03-a): el proyecto declara en el bloque `sdd:` de `axiom.yaml` el patrón de sus identificadores (`RF-USU-01`, `REQ-30.4`...), y un único punto del producto decide qué es un identificador válido, cuál es su serie y su número, y cuál es el siguiente libre de una serie. Los consumidores de la forma del identificador (compositor de OpenSpec, evidencia ISO, spec viva, documentación viva y mensajes de la CLI) usan ese punto y ninguno conserva una expresión regular propia; la única excepción deliberada y temporal es `adoptcheck` con `archive adopt` (REQ-44.4). Sin declaración rige un valor por defecto que admite exactamente los `REQ-<n>.<m>` vigentes, de modo que un proyecto que no declara nada no observa ningún cambio. Esta capacidad define la forma provisional de los identificadores, la asignación del siguiente libre al integrar y la traza de identificadores (REQ-47.1–47.6); no conserva los identificadores de un cliente al adoptar (C-03-c) ni migra las specs de Axiom (C-03-e).

## Términos

| Término | Significado |
|---|---|
| Patrón | Plantilla de segmentos declarada por el proyecto: literales y marcadores. Marcadores: `<DOM>` (código de dominio de la serie), `<n>` (número de la serie), `<m>` (número de secuencia sin relleno) y `<NN>`…`<NNNNNN>` (número de secuencia con relleno a 2..6 cifras). Su gramática es la de REQ-44.3. |
| Identificador válido | Texto que casa por completo con el patrón efectivo del proyecto. |
| Literal inicial | Primer literal del patrón, con la forma `[A-Z][A-Z0-9]*[-._]` (`REQ-`, `RF-`, `HU-`). Es el ancla de los candidatos. |
| Candidato | Token que empieza, en límite de palabra, por el literal inicial del patrón y continúa con letras, cifras y los caracteres no alfanuméricos de los demás literales del patrón, terminando en una letra o una cifra. Sirve para distinguir «mal formado» de «sin identificador». |
| Mal formado | Candidato que no es un identificador válido. |
| Serie | Todo lo que el identificador tiene salvo el número de secuencia: `30` en `REQ-30.4` (marcador `<n>`), `USU` en `RF-USU-01` (marcador `<DOM>`); su clave es la unión de los valores de `<DOM>` y `<n>` con `-`. Sin `<DOM>` ni `<n>` (`RF-<m>`), la serie es única y su clave es vacía. |
| Número | El valor entero del segmento de número de secuencia del identificador. |
| Patrón efectivo | El declarado en `axiom.yaml` o, sin declaración, el valor por defecto `REQ-<n>.<m>`. |
| Capacidad | Carpeta de `openspec/specs/` (`usuarios`). No tiene relación con `<DOM>`: la serie de un identificador sale solo del propio identificador. |
| Notación | Forma de nombrar el patrón en los mensajes: legible (la plantilla tal cual, `RF-<DOM>-<NN>`), compacta (sin ángulos, `RF-DOM-NN`) y compacta de serie (`REQ-n.*`). |

## Requirements

### Requirement: Declaración del Patrón de Identificador en el Bloque `sdd:` con Decodificador Estricto y Lectura en Vivo (REQ-44.1)

`axiom.yaml` PUEDE declarar el patrón de los identificadores de requisito del proyecto en el bloque de primer nivel `sdd:`, con la clave escalar de texto `id_pattern` (`sdd.id_pattern`), junto a `iso` y sin alterarla. La declaración es válida en toda topología y no depende de `sdd.iso`: un proyecto que no declara ISO también puede declararla. El bloque `sdd:` sigue decodificándose de forma estricta por lista cerrada de claves (REQ-29.1 de `sdd-iso-evidence`): una clave desconocida, un valor de `id_pattern` que no sea un escalar de texto (lista, mapa, nulo o cadena vacía) y un patrón que no cumpla la gramática de REQ-44.3 DEBEN reportarse como error accionable, tanto al cargar la configuración como en `axiom workspace validate`, con código de salida `1`. El mensaje, en español, DEBE nombrar la clave y la línea, el valor recibido, el motivo de la regla incumplida, un ejemplo de patrón válido y la salida para volver al patrón por defecto (retirar la clave), con la forma `sdd.id_pattern (línea N): la plantilla "<valor>" no es válida: <motivo>; un ejemplo válido es "RF-<DOM>-<NN>"; para el patrón por defecto (REQ-<n>.<m>) retira la clave`. Un patrón inválido NUNCA DEBE degradarse en silencio al valor por defecto ni ignorarse: la declaración inválida falla cerrada, también al sellar (`axiom sdd kickoff seal`) y en `axiom sdd status`.

El patrón se lee **en vivo** de `axiom.yaml` en cada uso y NUNCA de `kickoff.yaml`: el esquema de `kickoff.yaml` no cambia y el sellado no copia el patrón. `sdd.iso` sigue congelándose en la instantánea del cambio (REQ-29.13 de `sdd-lane-kickoff`) porque decide el proceso del cambio; el patrón define el vocabulario de la spec viva compartida y debe coincidir con el de la canónica contra la que se archiva. Un cambio en curso cuyo proyecto cambia de patrón se revalida contra el patrón vigente. Un `axiom.yaml` sin la clave se valida exactamente como antes de este cambio. El patrón se toma del `axiom.yaml` del workspace del comando.

#### Scenario: Un patrón válido se acepta

- **DADO** un `axiom.yaml` con `sdd: {id_pattern: "RF-<DOM>-<NN>"}`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** no se reporta ningún error por el bloque `sdd:`
- **Y** el patrón efectivo del proyecto es `RF-<DOM>-<NN>`

#### Scenario: El patrón es independiente de la declaración ISO

- **DADO** un `axiom.yaml` con `sdd: {id_pattern: "RF-<DOM>-<NN>"}` y sin `iso`, y otro con `sdd: {iso: true, id_pattern: "RF-<DOM>-<NN>"}`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** ninguno de los dos reporta error por la combinación de claves

#### Scenario: Un patrón inválido es un error accionable y no se ignora

- **DADO** un `axiom.yaml` con `sdd: {id_pattern: "RF-<DOM>-"}` (sin número)
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta en español la clave `sdd.id_pattern` con su línea, el valor recibido y el motivo de la regla incumplida
- **Y** el mensaje incluye un ejemplo de patrón válido y la indicación de retirar la clave para usar el patrón por defecto
- **Y** el código de salida es `1`

#### Scenario: El patrón inválido se reporta también al cargar la configuración

- **DADO** el mismo `axiom.yaml` con el patrón inválido
- **CUANDO** se carga la configuración (por ejemplo, con `axiom sdd status`)
- **ENTONCES** la carga reporta el mismo error accionable
- **Y** no se asume el patrón por defecto

#### Scenario: Un patrón inválido falla cerrado al sellar

- **DADO** un `axiom.yaml` con `sdd: {id_pattern: "<m><m>"}`
- **CUANDO** se ejecuta `axiom sdd kickoff seal` sobre un cambio de ese proyecto
- **ENTONCES** el sellado se rechaza con el error del patrón
- **Y** NO se escribe `kickoff.yaml`

#### Scenario: Un valor que no es texto se rechaza

- **DADO** un `axiom.yaml` con `sdd: {id_pattern: [uno, dos]}`, con `sdd: {id_pattern: {a: b}}`, con `sdd: {id_pattern: }` (nulo) o con `sdd: {id_pattern: ""}`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta que `sdd.id_pattern` debe ser un texto no vacío
- **Y** el código de salida es `1`

#### Scenario: Una clave desconocida sigue rechazándose

- **DADO** un `axiom.yaml` con `sdd: {id_patern: "RF-<DOM>-<NN>"}` (clave mal escrita)
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta en español la clave desconocida y las admitidas, entre las que figura `id_pattern`
- **Y** el código de salida es `1`

#### Scenario: Sin declaración la validación no cambia

- **DADO** un `axiom.yaml` sin la clave del patrón
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** el resultado es idéntico al anterior a este cambio

#### Scenario: El patrón se lee en vivo y el sellado no lo copia

- **DADO** un proyecto con `sdd: {id_pattern: "RF-<DOM>-<NN>"}`
- **CUANDO** se sella un cambio con `axiom sdd kickoff seal`
- **ENTONCES** el `kickoff.yaml` resultante no contiene el patrón ni ningún campo nuevo respecto de antes de este cambio

#### Scenario: Un cambio en curso se revalida contra el patrón vigente

- **DADO** un cambio ISO ya sellado cuya delta nombra `Alta de usuario (RF-USU-01)` con el patrón `RF-<DOM>-<NN>`
- **CUANDO** el proyecto cambia `axiom.yaml` a `sdd: {iso: true, id_pattern: "HU-<NNN>"}` y se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el nombre se evalúa con el patrón `HU-<NNN>`, no con el vigente al sellar
- **Y** se proyecta el motivo tipado de identificador nombrando el patrón `HU-<NNN>`

---

### Requirement: Patrón por Defecto que Admite Exactamente los `REQ-<n>.<m>` Vigentes (REQ-44.2)

Cuando el proyecto no declara patrón, el patrón efectivo DEBE ser el valor por defecto `REQ-<n>.<m>`, que admite exactamente los identificadores que admite hoy el producto (`REQ-` seguido de un entero decimal sin signo, un punto y otro entero decimal sin signo, sin exigir canonicidad: `REQ-30.01` sigue siendo válido) y no admite ningún otro. Su notación legible, usada en mensajes y continuaciones, DEBE ser exactamente `REQ-<n>.<m>`, su notación compacta `REQ-n.m` y su notación compacta de serie `REQ-n.*`. Sin declaración, todo mensaje, continuación y texto fijado hoy por las pruebas DEBE permanecer idéntico byte a byte, y ninguna entrada que hoy se acepta o se rechaza cambia de resultado, con una única excepción deliberada: el rechazo ante un bloque `sdd:` inválido de REQ-44.9, que alcanza a comandos que hoy no leen el bloque. Las rutas que no disponen de un workspace (por ejemplo, la composición con `--canonical` y `--delta` sin contexto de proyecto) DEBEN usar el valor por defecto.

#### Scenario: El defecto admite `REQ-n.m`

- **DADO** un proyecto sin declaración de patrón
- **CUANDO** se valida cada uno de `REQ-1.1`, `REQ-30.4`, `REQ-123.45` y `REQ-30.01`
- **ENTONCES** los cuatro son identificadores válidos

#### Scenario: El defecto rechaza lo que hoy rechaza

- **DADO** un proyecto sin declaración de patrón
- **CUANDO** se valida cada uno de `REQ-30`, `REQ-x.1`, `REQ-30.`, `REQ-30.4.1`, `REQ--1.2` y `req-30.4`
- **ENTONCES** ninguno es un identificador válido

#### Scenario: El texto legible del defecto no cambia los mensajes

- **DADO** un proyecto sin declaración de patrón y un requisito ADDED llamado `Cobro con IVA` en un cambio ISO
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el motivo tipado de identificador y su continuación son idénticos byte a byte a los de antes de este cambio, con el patrón nombrado como `REQ-<n>.<m>`

#### Scenario: Sin workspace rige el defecto

- **DADO** una composición con `axiom sdd archive-compose --canonical <ruta> --delta <ruta>` sin contexto de proyecto ni de cambio
- **CUANDO** el título de un requisito contiene `(REQ-30.4)`
- **ENTONCES** se reconoce `REQ-30.4` como su identificador, con el patrón por defecto

#### Scenario: El defecto coincide con las expresiones vigentes sobre las specs actuales

- **DADO** el patrón por defecto y los títulos de todos los requisitos de `openspec/specs/*/spec.md`, junto con los casos límite `REQ-30`, `REQ-30.1,`, `REQ-30.1.`, `REQ-x.1`, `REQ-030.01` y `(REQ-1.1) (REQ-2.2)`
- **CUANDO** se clasifica cada texto con el puerto
- **ENTONCES** la validez, el candidato y el identificador del título coinciden con los de las expresiones regulares que el producto aplicaba antes de este cambio

---

### Requirement: Gramática del Patrón, Serie, Número y Formato del Siguiente Identificador (REQ-44.3)

El patrón DEBE ser una plantilla de literales y marcadores que cumpla TODAS estas reglas; cada incumplimiento DEBE rechazarse como error de carga con su motivo (REQ-44.1):

1. No vacía y de 64 caracteres como máximo.
2. Empieza por un literal inicial de la forma `[A-Z][A-Z0-9]*[-._]` (una letra mayúscula, mayúsculas o cifras y un separador final).
3. Los literales solo contienen `[A-Z0-9._-]`: quedan fuera las minúsculas, los espacios, los paréntesis, la coma y los caracteres `<` y `>`.
4. Exactamente un marcador de número de secuencia (`<m>` o `<NN>`…`<NNNNNN>`, con relleno de 2 a 6 cifras); como máximo un `<DOM>` y un `<n>`. `<N>` con una sola `N` no existe (es `<m>`).
5. Dos marcadores nunca van seguidos, y un literal pegado a un marcador no empieza ni termina con un carácter del alfabeto de ese marcador (cifras para `<n>`, `<m>` y `<NN…>`; `[A-Z0-9]` para `<DOM>`), de modo que el reparto del identificador en segmentos es único.
6. Un marcador desconocido, un `<` sin cerrar o un `>` suelto son errores que nombran el texto.

Marcadores: `<DOM>` (serie, código de dominio) encaja con `[A-Z][A-Z0-9]*`; `<n>` (serie, número) con `[0-9]+`; `<m>` con `[0-9]+` y se formatea sin relleno; `<NN>`…`<NNNNNN>` (k = 2 a 6) encaja con la forma canónica de exactamente `k` cifras, o más de `k` cifras sin cero inicial, y se formatea rellenando con ceros hasta `k` cifras. Un identificador es válido cuando casa por completo con el patrón; el número se interpreta como entero y un número que no cabe en un entero NO es un identificador válido. Con `<NN…>` la forma es canónica, de modo que dos escrituras del mismo número no pueden ser ambas válidas; con `<n>` y `<m>` se conserva la permisividad vigente (`REQ-30.01`). Las minúsculas están excluidas de todo identificador válido de cualquier patrón declarable (regla 3): es la propiedad que garantiza REQ-44.6.

La serie de un identificador es todo lo que no es el número de secuencia y su clave la unión de los valores de `<DOM>` y `<n>` con `-`; en un patrón sin `<DOM>` ni `<n>` la serie es única. El patrón DEBE ser invertible: dada una serie y un número, el sistema formatea el identificador rellenando con ceros hasta la anchura declarada y sin truncar nunca un número que la supere. El siguiente identificador libre de una serie es el de número igual al máximo de esa serie en todo el registro de identificadores consultado más uno, con el criterio que rige hoy para `REQ-<n>.<m>`, formateado con el relleno del patrón. Las notaciones de mensaje son tres: legible (la plantilla), compacta y compacta de serie. El patrón declarado es el único origen de estas reglas: ningún consumidor las redefine.

#### Scenario: Serie y número con el patrón por defecto

- **DADO** el patrón por defecto
- **CUANDO** se descompone `REQ-30.4`
- **ENTONCES** la serie tiene la clave `30`, se muestra como `REQ-30.*` y el número es `4`

#### Scenario: Serie y número con un patrón de dominio

- **DADO** el patrón `RF-<DOM>-<NN>`
- **CUANDO** se descompone `RF-USU-01`
- **ENTONCES** la serie tiene la clave `USU`, se muestra como `RF-USU-*` y el número es `1`

#### Scenario: Patrón con serie única

- **DADO** el patrón `RF-<m>`
- **CUANDO** se descomponen `RF-7` y `RF-121`
- **ENTONCES** ambos pertenecen a la misma serie única, de clave vacía, y sus números son `7` y `121`

#### Scenario: Patrones válidos

- **DADO** cada uno de los patrones `REQ-<n>.<m>`, `RF-<DOM>-<NN>`, `RF-<m>` y `HU-<NNN>`
- **CUANDO** se declaran como `sdd.id_pattern`
- **ENTONCES** los cuatro se aceptan
- **Y** sus notaciones legibles son la propia plantilla y las compactas son `REQ-n.m`, `RF-DOM-NN`, `RF-m` y `HU-NNN`

#### Scenario: La anchura es canónica

- **DADO** el patrón `RF-<DOM>-<NN>`
- **CUANDO** se valida cada uno de `RF-USU-01`, `RF-USU-99`, `RF-USU-100`, `RF-USU-123`, `RF-USU-001` y `RF-USU-1`
- **ENTONCES** `RF-USU-01`, `RF-USU-99`, `RF-USU-100` y `RF-USU-123` son identificadores válidos
- **Y** `RF-USU-001` (cero inicial de más) y `RF-USU-1` (menos de dos cifras) no lo son

#### Scenario: Un número que desborda no es válido

- **DADO** el patrón `RF-<DOM>-<NN>`
- **CUANDO** se valida `RF-USU-99999999999999999999`
- **ENTONCES** no es un identificador válido

#### Scenario: El siguiente libre rellena hasta la anchura

- **DADO** el patrón `RF-<DOM>-<NN>` y un registro con `RF-USU-01` y `RF-USU-08`
- **CUANDO** se pide el siguiente identificador libre de la serie `USU`
- **ENTONCES** el resultado es `RF-USU-09`
- **Y** con `RF-USU-99` como máximo, el resultado es `RF-USU-100`, sin truncar

#### Scenario: El siguiente libre del defecto no cambia

- **DADO** el patrón por defecto y un registro cuyo máximo en la serie `30` es `REQ-30.4`
- **CUANDO** se pide el siguiente identificador libre de la serie `30`
- **ENTONCES** el resultado es `REQ-30.5`

#### Scenario: Los huecos no se rellenan

- **DADO** el patrón `RF-<DOM>-<NN>` y un registro con `RF-USU-01` y `RF-USU-05`
- **CUANDO** se pide el siguiente libre de la serie `USU`
- **ENTONCES** el resultado es `RF-USU-06`

#### Scenario: La serie de otro dominio no cuenta

- **DADO** el patrón `RF-<DOM>-<NN>` y un registro con `RF-USU-05` y `RF-FAC-02`
- **CUANDO** se pide el siguiente libre de la serie `FAC`
- **ENTONCES** el resultado es `RF-FAC-03`

#### Scenario: `<DOM>` no depende de la capacidad

- **DADO** el patrón `RF-<DOM>-<NN>` y un requisito `Factura (RF-FAC-01)` en la capacidad `usuarios`
- **CUANDO** se valida y se descompone `RF-FAC-01`
- **ENTONCES** es un identificador válido de la serie `FAC`, sin relación con la capacidad `usuarios`

#### Scenario: Ida y vuelta de serie, número y formato

- **DADO** el patrón `RF-<DOM>-<NN>` y `HU-<NNN>`, y un identificador válido de cada uno
- **CUANDO** se obtienen su serie y su número y se vuelve a formatear
- **ENTONCES** se recupera el identificador original

#### Scenario: Patrones mal construidos

- **DADO** cada uno de los patrones `""`, `RF-` (sin número), `RF-<m>-<m>` (dos números), `RF-<XYZ>-<m>` (marcador desconocido), `RF-<N>` (una sola `N`), `RF-<NNNNNNN>` (más de seis), `RF-<DOM-<m>` (sin cerrar), `RF-<m>>` (`>` suelto), `RF-<n>-<n>-<m>` y `RF-<DOM>-<DOM>-<m>` (marcador de serie repetido), `<DOM>-<m>` y `RF<DOM>-<m>` (sin literal inicial), `RF-<DOM><NN>` (marcadores seguidos), `RF-<m>0` y `RF-A<DOM>-<m>` (literal pegado con carácter del alfabeto del marcador), `rf-<m>` y `RF <m>` y `RF-(<m>)` y `RF-<m>,` (carácter fuera del alfabeto de literales), y un patrón de 65 caracteres
- **CUANDO** se declara como `sdd.id_pattern`
- **ENTONCES** cada uno se rechaza como error accionable que nombra el texto y la regla incumplida

---

### Requirement: Un Único Punto de Parseo, Validación, Formato y Serie que Usan los Consumidores (REQ-44.4)

El parseo del identificador de un título de requisito, la detección de candidatos, la validación, la extracción de serie y número, el formato del siguiente identificador, las notaciones del patrón y el identificador sintético DEBEN residir en un único paquete hoja (`internal/reqid`, solo biblioteca estándar) que no depende de ningún otro paquete de Axiom. `openspecdoc` e `isoevidence` solo pueden importar ese paquete entre los internos. El identificador de un título es el último identificador válido entre paréntesis (`(ID)`); el campo `ID` precalculado de los requisitos de `openspecdoc` desaparece, de modo que ningún consumidor lee un identificador calculado con un patrón por defecto oculto.

Los consumidores `openspecdoc`, `isoevidence`, `sddstatus`, `livespec`, `livingdoc` y la CLI (`archive integrate`, `sdd archive-compose`, `sdd archive-check`, `sdd trace`) DEBEN obtener el patrón efectivo del workspace cuando lo tienen y recibirlo por parámetro u opción cuando no, con el valor por defecto si no se pasa. Ningún consumidor DEBE conservar una expresión regular, una constante ni un `Sprintf` propios del formato del identificador. **Excepción deliberada y temporal (A-03-2, REQ-30.31):** `internal/adoption/adoptcheck` y `axiom archive adopt` siguen usando el patrón por defecto `REQ-<n>.<m>` hasta C-03-c, sea cual sea el patrón declarado por el proyecto; sus reglas (`RuleFormat`, `RuleUniqueness`, `RuleSeries`), textos y notaciones no cambian y NO leen `sdd.id_pattern`.

**Criterio de terminado ejecutable** (decisión del 2026-11-23): `rg -n 'REQ-[^0-9]' --type go internal cmd -g '!*_test.go'` solo coincide en `internal/reqid` y en comentarios, y la prueba de arquitectura `TestNoRequirementIDFormatOutsideReqid` falla si un literal de texto de un `.go` de producción de `internal/` o `cmd/` contiene `REQ-` seguido de algo que no sea una cifra fuera de `internal/reqid` (los comentarios quedan fuera). Las citas de requisitos de Axiom con cifras (`REQ-21.18`) en mensajes y la plantilla de borrador de `internal/knowledge/sweep.go` quedan fuera de este criterio y para C-03-e.

#### Scenario: Ningún consumidor define el formato

- **DADO** el árbol de código de producción tras este cambio
- **CUANDO** se ejecuta `rg -n 'REQ-[^0-9]' --type go internal cmd -g '!*_test.go'`
- **ENTONCES** solo hay coincidencias en `internal/reqid` y en comentarios

#### Scenario: La prueba de arquitectura vigila los literales de texto

- **DADO** un literal de texto de producción fuera de `internal/reqid` que contiene `REQ-` seguido de una letra, un `<` o un guion
- **CUANDO** se ejecuta `TestNoRequirementIDFormatOutsideReqid`
- **ENTONCES** la prueba falla nombrando el fichero
- **Y** un literal con `REQ-` seguido de una cifra, o un comentario, no la hace fallar

#### Scenario: Todos los consumidores coinciden ante la misma entrada

- **DADO** un proyecto con el patrón `RF-<DOM>-<NN>` y un título `Alta de usuario (RF-USU-01)`
- **CUANDO** lo consultan `openspecdoc`, `isoevidence` y `livespec`
- **ENTONCES** los tres obtienen el mismo identificador `RF-USU-01`, la misma serie `USU` y el mismo número `1`

#### Scenario: El puerto y los consumidores de la hoja respetan las importaciones

- **DADO** los paquetes `internal/reqid`, `internal/openspecdoc` e `internal/isoevidence`
- **CUANDO** se listan sus importaciones
- **ENTONCES** `internal/reqid` solo importa biblioteca estándar
- **Y** `internal/openspecdoc` e `internal/isoevidence` solo importan `internal/reqid` entre los paquetes de Axiom

#### Scenario: Sin patrón pasado, un consumidor usa el defecto

- **DADO** un consumidor invocado sin patrón (por ejemplo, `openspecdoc` sin opciones)
- **CUANDO** procesa un título con `(REQ-30.4)`
- **ENTONCES** reconoce `REQ-30.4`, con el patrón por defecto

#### Scenario: `adoptcheck` no cambia sus reglas con el defecto

- **DADO** una adopción con el patrón por defecto
- **CUANDO** se ejecuta `axiom archive adopt check`
- **ENTONCES** las violaciones de formato, unicidad y serie se reportan exactamente como antes de este cambio

#### Scenario: `adoptcheck` y `archive adopt` siguen en el defecto con un patrón declarado

- **DADO** un proyecto con `sdd: {id_pattern: "RF-<DOM>-<NN>"}`
- **CUANDO** se ejecuta `axiom archive adopt check` o `axiom archive adopt`
- **ENTONCES** se aplican las reglas y los textos con el patrón por defecto `REQ-<n>.<m>`, igual que sin declaración
- **Y** no se lee `sdd.id_pattern` hasta C-03-c

---

### Requirement: Candidato frente a Válido con Errores Tipados y Continuaciones Conservadas (REQ-44.5)

El sistema DEBE distinguir entre un texto candidato y un identificador válido. El candidato empieza, en límite de palabra, por el literal inicial del patrón efectivo y sigue con letras, cifras y los caracteres no alfanuméricos de los literales posteriores del patrón, terminando en una letra o una cifra; para el patrón por defecto es exactamente `\bREQ-[0-9A-Za-z.]*[0-9A-Za-z]`. Un candidato que no casa por completo con el patrón efectivo DEBE reportarse como candidato mal formado, con un error tipado que nombre el texto encontrado, el patrón efectivo en su texto legible y la continuación (corregirlo según el patrón). La búsqueda devuelve los candidatos sin repetidos y en orden de aparición, separados en válidos y mal formados. Los demás casos (nombre sin identificador, nombre con más de un identificador distinto y repetición de un identificador entre requisitos) conservan su error tipado y su continuación, formulados con el patrón efectivo (REQ-29.3 de `sdd-iso-evidence`). La puntuación final que no forma parte del identificador (una coma, un punto o un paréntesis de cierre tras `REQ-30.1`) NO DEBE contarse como parte del candidato. Con el patrón por defecto, las clasificaciones, los textos y las continuaciones coinciden con los de antes de este cambio.

#### Scenario: Un candidato mal formado se distingue de la ausencia

- **DADO** el patrón por defecto y un nombre `Cobro con IVA (REQ-30)`
- **CUANDO** se clasifica el nombre
- **ENTONCES** se reporta un candidato mal formado `REQ-30`, no «sin identificador»
- **Y** el error nombra `REQ-30`, el patrón `REQ-<n>.<m>` y la continuación

#### Scenario: Un candidato mal formado con un patrón declarado

- **DADO** el patrón `RF-<DOM>-<NN>` y un nombre `Alta de usuario (RF-USU-1)`
- **CUANDO** se clasifica el nombre
- **ENTONCES** se reporta un candidato mal formado `RF-USU-1`
- **Y** el error nombra el patrón `RF-<DOM>-<NN>`

#### Scenario: Prefijos, extensiones y minúsculas

- **DADO** el patrón `RF-<DOM>-<NN>` y los nombres `Alta (RF-USU-01-bis)`, `Alta (rf-usu-01)` y `Alta (XRF-USU-01)`
- **CUANDO** se clasifica cada nombre
- **ENTONCES** `RF-USU-01-bis` se reporta como candidato mal formado
- **Y** `rf-usu-01` y `XRF-USU-01` no son candidatos: el nombre se reporta como sin identificador

#### Scenario: Extensiones del defecto

- **DADO** el patrón por defecto y los nombres `Cobro (REQ-30.4.1)`, `Cobro (REQ-30.1abc)` y `Cobro (XREQ-30.1)`
- **CUANDO** se clasifica cada nombre
- **ENTONCES** `REQ-30.4.1` y `REQ-30.1abc` se reportan como candidatos mal formados
- **Y** `XREQ-30.1` no es un candidato

#### Scenario: Con un patrón declarado el defecto deja de ser válido

- **DADO** el patrón `RF-<DOM>-<NN>` y un nombre `Cobro con IVA (REQ-30.1)`
- **CUANDO** se clasifica el nombre
- **ENTONCES** el nombre no contiene ningún identificador válido ni ningún candidato: se reporta como sin identificador

#### Scenario: Un nombre sin identificador

- **DADO** cualquier patrón efectivo y un nombre `Cobro con IVA`
- **CUANDO** se clasifica el nombre
- **ENTONCES** se reporta ausencia de identificador, con la continuación de añadirlo según el patrón

#### Scenario: Dos identificadores distintos son ambiguos

- **DADO** el patrón `RF-<DOM>-<NN>` y un nombre `Cobro y devolución (RF-FAC-01, RF-FAC-02)`
- **CUANDO** se clasifica el nombre
- **ENTONCES** se reporta ambigüedad nombrando los dos identificadores

#### Scenario: El mismo identificador repetido en un título cuenta una vez

- **DADO** el patrón `RF-<DOM>-<NN>` y un nombre `Alta (RF-USU-01) y su alta (RF-USU-01)`
- **CUANDO** se clasifica el nombre
- **ENTONCES** el nombre contiene un único identificador `RF-USU-01` y no es ambiguo

#### Scenario: La puntuación final no forma parte del candidato

- **DADO** un texto `Cubre: REQ-30.1,` con el patrón por defecto
- **CUANDO** se extrae el candidato
- **ENTONCES** el candidato es `REQ-30.1`, sin la coma

#### Scenario: Con el defecto los textos no cambian

- **DADO** el patrón por defecto y los casos de identificador ausente, ambiguo, repetido y mal formado que fijan hoy las pruebas
- **CUANDO** se genera cada error tipado y su continuación
- **ENTONCES** los textos son idénticos byte a byte a los de antes de este cambio

---

### Requirement: El Identificador Sintético de la Documentación Viva Nunca se Confunde con un Identificador Válido (REQ-44.6)

Cuando un requisito indexado por la documentación viva (`internal/livingdoc`) no lleva identificador válido en su encabezado, el índice DEBE asignarle un identificador sintético que NO sea un identificador válido de ningún patrón declarable (REQ-44.3). El sintético es `REQ-<capacidad>-<ordinal>` cuando el nombre de la capacidad contiene al menos una letra minúscula ASCII (todos los casos reales: los sintéticos de `openspec/INDEX.md` no cambian) y `req-<capacidad>-<ordinal>` en otro caso. Como todo identificador válido carece de minúsculas (REQ-44.3, regla 3) y todo sintético contiene al menos una, ambos espacios son disjuntos por construcción, sin depender del patrón del proyecto. La lectura del identificador del encabezado DEBE hacerse por el puerto con el patrón efectivo, y la lectura laxa del identificador final `(<literal inicial>…)` de un título no valida, como hoy. Un requisito con identificador válido conserva ese identificador en el índice. Con el patrón por defecto y las specs actuales, los identificadores del índice no cambian. `livingdoc` obtiene el patrón del workspace; sin workspace usa el defecto. Si el patrón no es accesible (`SDDErr` o `ParseErr`), `livingdoc`, que es informativo, usa el defecto y añade un aviso a los avisos del informe de sincronización en lugar de fallar.

#### Scenario: Un encabezado con identificador válido lo conserva

- **DADO** un requisito `### Requirement: Cobro con IVA (REQ-30.1)` en la capacidad `facturacion` y el patrón por defecto
- **CUANDO** se indexa
- **ENTONCES** el identificador del índice es `REQ-30.1`

#### Scenario: Un encabezado con identificador del patrón declarado lo conserva

- **DADO** el patrón `RF-<DOM>-<NN>` y un requisito `### Requirement: Alta de usuario (RF-USU-01)` en la capacidad `usuarios`
- **CUANDO** se indexa
- **ENTONCES** el identificador del índice es `RF-USU-01`

#### Scenario: Un encabezado sin identificador recibe uno sintético no válido

- **DADO** un requisito `### Requirement: Cobro con IVA` en la capacidad `facturacion`, con el patrón por defecto
- **CUANDO** se indexa
- **ENTONCES** el identificador sintético es `REQ-facturacion-<ordinal>`
- **Y** no es válido según el patrón por defecto

#### Scenario: El sintético no es válido con un patrón declarado

- **DADO** el patrón `RF-<DOM>-<NN>` y un requisito sin identificador en la capacidad `usuarios`
- **CUANDO** se indexa
- **ENTONCES** el identificador sintético no es válido según `RF-<DOM>-<NN>`
- **Y** no es igual a ningún identificador válido existente del proyecto

#### Scenario: Una capacidad sin minúsculas recibe un sintético con prefijo en minúsculas

- **DADO** un requisito sin identificador en una capacidad llamada `USU`
- **CUANDO** se indexa
- **ENTONCES** el identificador sintético empieza por `req-` y no es válido según ningún patrón declarable

#### Scenario: El sintético es estable entre ejecuciones

- **DADO** el mismo documento indexado dos veces
- **CUANDO** se comparan los identificadores sintéticos
- **ENTONCES** son idénticos

#### Scenario: Un identificador de otro patrón en el encabezado recibe un sintético

- **DADO** el patrón `RF-<DOM>-<NN>` y un requisito `### Requirement: Cobro (REQ-30.1)` en la capacidad `facturacion`
- **CUANDO** se indexa
- **ENTONCES** el identificador del índice es un sintético y no `REQ-30.1`

#### Scenario: El sintético no se confunde en la comprobación de identificadores

- **DADO** un índice con un identificador sintético
- **CUANDO** `isoevidence` o `livespec` consultan su validez
- **ENTONCES** ninguno lo considera un identificador válido ni lo cuenta en una serie

#### Scenario: Un patrón inaccesible solo avisa en `livingdoc`

- **DADO** un `axiom.yaml` con un bloque `sdd:` inválido
- **CUANDO** se sincroniza la documentación viva
- **ENTONCES** se indexa con el patrón por defecto
- **Y** el informe de sincronización lleva un aviso que nombra el error del patrón
- **Y** la sincronización no falla por ese motivo

---

### Requirement: Un Proyecto con Patrón `RF-<DOM>-<NN>` Integra una Delta ODD y Archiva un Cambio SDD sin Errores (REQ-44.7)

Un proyecto cuyo `axiom.yaml` declara el patrón `RF-<DOM>-<NN>` DEBE poder, sin ningún error atribuible al formato del identificador: (a) integrar una delta ODD con `axiom archive integrate --odd <feature>`, con la comprobación de colisión (REQ-30.4 de `odd-living-spec-integration`) formulada y calculada por la serie del patrón, y (b) archivar un cambio SDD cuya delta se compone con `axiom sdd archive-compose` (REQ-30.22 de `sdd-archive-lifecycle`), incluido el archivado interrumpido y reanudado y la reintegración idempotente. En un cambio ISO de ese proyecto, el identificador del nombre de los requisitos (REQ-29.3 de `sdd-iso-evidence`), el de `Cubre:` (REQ-29.4) y el de la consulta de un requisito existente (REQ-29.14 de `spec-baseline-integration`) DEBEN ser los del patrón declarado. Mientras las specs vivas conserven `REQ-<n>.<m>` (hasta C-03-e), esos requisitos no son identificadores válidos del proyecto: el sistema los trata como requisitos sin identificador y los empareja por título exacto. La verificación de este requisito se hace con `go test` sobre un proyecto temporal, sin compilar ni ejecutar un binario suelto.

#### Scenario: Integración ODD con el patrón declarado

- **DADO** un proyecto temporal con `sdd: {id_pattern: "RF-<DOM>-<NN>"}`, una capacidad `usuarios` con `RF-USU-01` y una delta ODD con un ADDED `Alta masiva (RF-USU-02)`
- **CUANDO** se ejecuta `axiom archive integrate --odd <feature>`
- **ENTONCES** la integración termina con código `0`
- **Y** `openspec/specs/usuarios/spec.md` contiene `Alta masiva (RF-USU-02)`

#### Scenario: La colisión propone el siguiente libre de la serie

- **DADO** el mismo proyecto y un ADDED `Otra alta (RF-USU-01)`, identificador que ya existe
- **CUANDO** se ejecuta `axiom archive integrate --odd <feature>`
- **ENTONCES** se rechaza nombrando `RF-USU-01`, la capacidad `usuarios` y el siguiente libre `RF-USU-02`
- **Y** no se escribe nada

#### Scenario: Un identificador con el formato por defecto se rechaza en ese proyecto

- **DADO** el mismo proyecto y un ADDED `Alta masiva (REQ-31.1)`
- **CUANDO** se ejecuta `axiom archive integrate --odd <feature>`
- **ENTONCES** se rechaza nombrando el requisito y el patrón `RF-<DOM>-<NN>`
- **Y** no se escribe nada

#### Scenario: Archivado SDD con el patrón declarado

- **DADO** un cambio SDD de ese proyecto con una delta con un ADDED `Baja de usuario (RF-USU-03)` y un MODIFIED de `Alta de usuario (RF-USU-01)`
- **CUANDO** se compone con `axiom sdd archive-compose` y se archiva el cambio
- **ENTONCES** el archivado termina sin errores
- **Y** la spec viva contiene ambos requisitos con sus identificadores

#### Scenario: Archivado SDD interrumpido y reanudado con el patrón declarado

- **DADO** el mismo cambio SDD cuyo archivado se interrumpió después de componer la delta y antes de mover la carpeta del cambio
- **CUANDO** se consulta `axiom sdd archive-check` y se reanuda el archivado
- **ENTONCES** el dominio figura como ya compuesto y el archivado se completa sin duplicar ni perder requisitos

#### Scenario: Cambio ISO con el patrón declarado

- **DADO** un cambio ISO de ese proyecto con una delta cuyos requisitos llevan `RF-USU-02` y `RF-USU-03` y tareas con `Cubre: RF-USU-02, RF-USU-03`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta ningún motivo de identificador ni de cobertura

#### Scenario: La consulta del bug usa el patrón declarado

- **DADO** el mismo proyecto y un bug con `Cubre: RF-USU-01`, requisito existente en la baseline
- **CUANDO** se consulta el requisito en la baseline
- **ENTONCES** se informa de que existe y se devuelven sus escenarios

#### Scenario: Estado transitorio con specs vivas en `REQ-<n>.<m>`

- **DADO** el mismo proyecto y una canónica con `### Requirement: Cobro (REQ-30.1)` en `facturacion`
- **CUANDO** una delta SDD compone un MODIFIED con el título exacto `Cobro (REQ-30.1)`
- **ENTONCES** el requisito se empareja por título exacto y la composición se acepta
- **Y** la consulta de `REQ-30.1` en la baseline no devuelve ese requisito

---

### Requirement: Un Patrón Inaccesible Falla Cerrado en los Consumidores que lo Necesitan (REQ-44.9)

Cuando el bloque `sdd:` de `axiom.yaml` no se puede leer o es inválido (error de sintaxis de la configuración o error del bloque `sdd:`, también el de `iso`), el patrón de identificador es inaccesible y NINGÚN consumidor DEBE evaluar con un patrón que el proyecto no declaró. Fallo cerrado deliberado, sin escribir: `axiom archive integrate` y `axiom sdd archive-compose` se niegan, con la continuación `axiom workspace validate --path <proyecto>`; `axiom sdd trace` y `axiom sdd archive-check` terminan con error y la misma continuación. En un cambio ISO, la etapa `specs` lleva el hallazgo `iso_declaration_invalid` en lugar de las reglas de identificador, la etapa `tasks` no se evalúa y la proyección de la matriz se omite (REQ-29.3 y REQ-29.16). No se crea ningún código de motivo nuevo. `livingdoc`, que es informativo, es la excepción y solo avisa (REQ-44.6). `axiom sdd archive-compose` sin workspace (`--canonical` y `--delta` sin contexto de proyecto) no lee ninguna declaración y usa el patrón por defecto. Esta negativa es la única excepción deliberada a que ninguna entrada hoy aceptada cambie de resultado (REQ-44.2).

#### Scenario: `archive integrate` se niega con un bloque `sdd:` inválido

- **DADO** un `axiom.yaml` con `sdd: {iso: quizás}` y una delta ODD válida
- **CUANDO** se ejecuta `axiom archive integrate --odd <feature>`
- **ENTONCES** finaliza con código distinto de cero nombrando el error del bloque `sdd:` y la continuación `axiom workspace validate --path <proyecto>`
- **Y** no se escribe nada

#### Scenario: `archive-compose` se niega con un patrón inválido

- **DADO** un `axiom.yaml` con `sdd: {id_pattern: "RF-"}` y una delta de un cambio de ese proyecto
- **CUANDO** se ejecuta `axiom sdd archive-compose` en el contexto de ese proyecto
- **ENTONCES** finaliza con código distinto de cero con el error del patrón y la continuación de `workspace validate`
- **Y** no se escribe ningún fichero

#### Scenario: `archive-compose` sin workspace usa el defecto

- **DADO** una composición con `--canonical` y `--delta` fuera de cualquier workspace
- **CUANDO** se ejecuta
- **ENTONCES** se compone con el patrón por defecto, sin consultar ninguna declaración

#### Scenario: `sdd trace` y `archive-check` terminan con error

- **DADO** un `axiom.yaml` con un `sdd.id_pattern` inválido
- **CUANDO** se ejecuta `axiom sdd trace` o `axiom sdd archive-check`
- **ENTONCES** terminan con código distinto de cero con el error del patrón y la continuación de `axiom workspace validate`

#### Scenario: Un cambio ISO con el patrón inaccesible

- **DADO** un cambio ISO sellado y un `axiom.yaml` con `sdd.id_pattern` inválido
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** la etapa `specs` proyecta el hallazgo `iso_declaration_invalid`, no el de identificador
- **Y** la etapa `tasks` no se evalúa y no se proyecta `iso.matrix`
### Requirement: Forma del Identificador Provisional, Inconfundible con un Identificador Válido, Única en el Cambio y Ausente de la Spec Viva (REQ-47.1)

Un requisito ADDED de un cambio SDD o de un documento ODD PUEDE llevar en su título un identificador provisional en lugar de un identificador definitivo. El identificador provisional DEBE cumplir TODOS estos invariantes, con independencia del patrón efectivo del proyecto (`sdd.id_pattern`, REQ-44.1):

1. Es inconfundible con un identificador válido bajo cualquier patrón declarable: NUNCA es un identificador válido (REQ-44.3), y las operaciones de búsqueda de identificadores en un texto (`InTitle`, `Scan`) NUNCA lo devuelven como un identificador válido. Se apoya en la propiedad de REQ-44.3 de que ningún identificador válido contiene minúsculas: el provisional contiene al menos una minúscula. NUNCA puede ser, además, un candidato mal formado del patrón (REQ-44.5): no es una errata de un identificador, es otra clase de token.
2. Lleva fijada la serie cuando se conoce (por ejemplo, la serie `30` de `REQ-<n>.<m>` o el dominio `USU` de `RF-<DOM>-<NN>`) y, cuando la serie aún no existe, puede llevar en su lugar una etiqueta de serie nueva propia del cambio, de modo que varios ADDED compartan una serie que todavía no tiene número. La etiqueta de serie nueva solo es admisible con un patrón que tenga el marcador `<n>`; con un patrón con `<DOM>` el dominio DEBE ir fijado, porque no se puede inventar un código de dominio; con un patrón de serie única no hay serie que elegir.
3. Es único dentro del cambio SDD (todas sus capacidades) o del documento ODD, y NO lo es entre cambios: dos cambios o documentos distintos pueden usar el mismo provisional sin colisión.
4. NUNCA llega a `openspec/specs/`: ninguna composición ni integración que acabe en la spec viva puede dejar un provisional en ella.

El formato exacto del identificador provisional lo fija el diseño; esta spec solo exige los invariantes.

#### Scenario: Un provisional no es un identificador válido con ningún patrón

- **DADO** cada uno de los patrones `REQ-<n>.<m>`, `RF-<DOM>-<NN>`, `RF-<m>` y `HU-<NNN>`, y un identificador provisional con una serie fijada que sea admisible en ese patrón
- **CUANDO** se valida el provisional y se buscan identificadores en un título que lo contiene
- **ENTONCES** la validación lo rechaza como identificador válido con los cuatro patrones
- **Y** la búsqueda en el título no devuelve ningún identificador válido
- **Y** no se clasifica como candidato mal formado del patrón

#### Scenario: El provisional contiene una minúscula

- **DADO** cualquier identificador provisional que el sistema produzca o acepte
- **CUANDO** se inspeccionan sus caracteres
- **ENTONCES** contiene al menos una letra minúscula, propiedad que ningún identificador válido de ningún patrón declarable posee

#### Scenario: Un provisional con la serie fijada la conserva

- **DADO** el patrón por defecto y un ADDED con un provisional con la serie `30` fijada
- **CUANDO** se analiza el identificador
- **ENTONCES** la serie reconocida es `30`
- **Y** con el patrón `RF-<DOM>-<NN>` y un provisional con el dominio `USU` fijado, la serie reconocida es `USU`

#### Scenario: Varios ADDED comparten una serie nueva

- **DADO** el patrón por defecto y dos ADDED de la misma capacidad nueva cuyos provisionales llevan la misma etiqueta de serie nueva
- **CUANDO** se analiza el cambio
- **ENTONCES** ambos pertenecen a la misma serie nueva, aún sin número
- **Y** son provisionales distintos entre sí

#### Scenario: Una etiqueta de serie nueva no se admite con un patrón con `<DOM>`

- **DADO** el patrón `RF-<DOM>-<NN>` y un ADDED cuyo provisional lleva una etiqueta de serie nueva en lugar de un dominio
- **CUANDO** se planifica la asignación
- **ENTONCES** se rechaza con un error tipado que nombra el requisito, el patrón `RF-<DOM>-<NN>` y la continuación (fijar el dominio en el provisional)
- **Y** no se escribe nada

#### Scenario: Un provisional repetido en el cambio es un error

- **DADO** un cambio con dos ADDED, en la misma capacidad o en dos capacidades, que declaran el mismo provisional `«prov-a»`
- **CUANDO** se planifica la asignación
- **ENTONCES** se rechaza con un error tipado de provisional duplicado que nombra `«prov-a»` y los dos requisitos
- **Y** no se escribe nada

#### Scenario: El mismo provisional en dos cambios no colisiona

- **DADO** dos cambios activos distintos, cada uno con un ADDED que declara `«prov-a»`
- **CUANDO** se planifica la asignación de cada uno
- **ENTONCES** ninguno se rechaza por el provisional del otro

#### Scenario: Ningún provisional llega a la spec viva

- **DADO** un cambio con ADDED provisionales en una capacidad existente y en una capacidad nueva
- **CUANDO** el cambio se integra o se archiva con éxito
- **ENTONCES** ningún fichero de `openspec/specs/` contiene un provisional
- **Y** cada requisito añadido lleva un identificador definitivo válido según el patrón efectivo

#### Scenario: Los lectores de deltas activas no toman el provisional como identificador

- **DADO** un cambio activo cuya delta tiene un ADDED con un provisional, y otro cambio que integra un ADDED con un identificador definitivo de la misma serie
- **CUANDO** se comprueban las colisiones de la integración (REQ-30.4 de `odd-living-spec-integration`), se indexa la documentación viva y se cuentan las tareas de un documento ODD en el panel
- **ENTONCES** el provisional no se registra como identificador ni colisiona con ningún definitivo
- **Y** la lectura laxa del indexador de la documentación viva no lo toma como identificador
- **Y** el recuento de tareas de un documento ODD no cambia por la sección del mapa de asignaciones (REQ-47.6)

---

### Requirement: Asignación del Siguiente Número Libre al Integrar, Incluida la Apertura de una Serie (REQ-47.2)

Al integrar (archivado SDD, REQ-47.8, o `axiom archive integrate --odd`), el sistema DEBE sustituir cada provisional por un identificador definitivo válido según el patrón efectivo del proyecto. El número es el siguiente libre de la serie, calculado por el patrón (REQ-44.3) como el máximo de la serie entre los identificadores ocupados más uno, formateado con el relleno del patrón y sin truncar. Los identificadores ocupados son la unión de:

1. los de la spec viva de la rama (`openspec/specs/`), superados incluidos;
2. los definitivos que declaran las deltas de los cambios activos (`openspec/changes/<cambio>/specs/`, sin `archive/` y sin el propio cambio), el mismo alcance que la comprobación de colisión de REQ-30.4 de `odd-living-spec-integration`;
3. los ADDED con identificador definitivo del propio cambio SDD o del propio documento ODD;
4. los definitivos que el mapa guardado del propio cambio ya asignó (REQ-47.6).

Así un número asignado nunca coincide con un definitivo del propio cambio ni con el de otro cambio activo. La asignación DEBE abrir una serie vacía: cuando la serie no tiene ningún identificador ocupado, el primer número es `1` (con el patrón por defecto, `REQ-<n>.1`; con `RF-<DOM>-<NN>`, `RF-<DOM>-01`).

Cuando el provisional lleva una etiqueta de serie nueva (patrón con `<n>`), el sistema DEBE asignarle el siguiente número de serie libre, igual al máximo de los números de serie ocupados más uno, y todos los ADDED con la misma etiqueta reciben esa misma serie, con números de secuencia consecutivos desde `1`. Distintas etiquetas de serie nueva reciben series distintas, y una serie nueva nunca coincide con una serie que fija un definitivo o un provisional del propio cambio.

La asignación DEBE ser determinista: las mismas entradas (las deltas del cambio o el documento ODD, la spec viva, las deltas de los cambios activos y el mapa guardado) producen siempre el mismo mapa de provisional a definitivo. Los números de una misma serie se asignan en el orden de aparición de los ADDED en el cambio, con un orden fijo y reproducible de las capacidades. Una entrada del mapa guardado conserva su definitivo aunque cambie ese orden (REQ-30.3 de `odd-living-spec-integration`). El rango de números NO DEBE reservar ni usar valores altos «de cortesía» que puedan confundirse con un identificador válido real (el provisional ya es inconfundible por REQ-47.1).

#### Scenario: Se asigna el siguiente libre de una serie existente

- **DADO** el patrón por defecto, una spec viva con `REQ-30.1` a `REQ-30.4` y un ADDED con un provisional con la serie `30` fijada
- **CUANDO** se integra el cambio
- **ENTONCES** el requisito recibe `REQ-30.5`

#### Scenario: Se abre una serie vacía con un patrón de dominio

- **DADO** el patrón `RF-<DOM>-<NN>`, una spec viva sin ningún identificador de la serie `FAC` y un ADDED con un provisional con el dominio `FAC` fijado
- **CUANDO** se integra el cambio
- **ENTONCES** el requisito recibe `RF-FAC-01`

#### Scenario: Se abre una serie nueva con el patrón por defecto

- **DADO** el patrón por defecto, una spec viva cuyo mayor número de serie es `46` y una capacidad nueva con dos ADDED cuyos provisionales comparten una etiqueta de serie nueva
- **CUANDO** se integra el cambio
- **ENTONCES** los dos requisitos reciben `REQ-47.1` y `REQ-47.2`, en el orden de aparición

#### Scenario: Dos etiquetas de serie nueva reciben series distintas

- **DADO** el patrón por defecto, una spec viva cuyo mayor número de serie es `46` y dos ADDED con etiquetas de serie nueva distintas
- **CUANDO** se integra el cambio
- **ENTONCES** uno recibe `REQ-47.1` y el otro `REQ-48.1`, de forma determinista según el orden del cambio

#### Scenario: Varios ADDED de una serie reciben números consecutivos

- **DADO** el patrón `RF-<DOM>-<NN>`, `RF-USU-08` como máximo de la serie `USU` y tres ADDED con provisionales con el dominio `USU` fijado
- **CUANDO** se integra el cambio
- **ENTONCES** reciben `RF-USU-09`, `RF-USU-10` y `RF-USU-11`, en el orden de aparición

#### Scenario: La asignación es determinista

- **DADO** las mismas deltas y la misma spec viva
- **CUANDO** se planifica la asignación dos veces
- **ENTONCES** el mapa de provisional a definitivo es idéntico en ambas

#### Scenario: Patrón de serie única

- **DADO** el patrón `RF-<m>`, `RF-7` como máximo y un ADDED con un provisional
- **CUANDO** se integra el cambio
- **ENTONCES** el requisito recibe `RF-8`

#### Scenario: Un definitivo del propio cambio cuenta como ocupado

- **DADO** el patrón por defecto, una spec viva con `REQ-30.1` a `REQ-30.4` y un cambio con un ADDED `(REQ-30.5)` y otro ADDED con un provisional con la serie `30` fijada
- **CUANDO** se integra el cambio
- **ENTONCES** el ADDED provisional recibe `REQ-30.6`
- **Y** ningún identificador se repite en la spec viva

#### Scenario: Una serie nueva no recae en una serie fijada por el propio cambio

- **DADO** el patrón por defecto, una spec viva cuyo mayor número de serie es `46` y un cambio con un ADDED `(REQ-47.1)` y otro ADDED con una etiqueta de serie nueva
- **CUANDO** se integra el cambio
- **ENTONCES** el ADDED con la etiqueta recibe `REQ-48.1`

#### Scenario: Un definitivo de otro cambio activo cuenta como ocupado

- **DADO** el patrón por defecto, una spec viva con `REQ-30.1` a `REQ-30.4`, otro cambio activo cuya delta añade `(REQ-30.5)` y un ADDED con un provisional con la serie `30` fijada
- **CUANDO** se integra el cambio
- **ENTONCES** el requisito recibe `REQ-30.6`

---

### Requirement: Plan Único de Asignación sobre Todas las Capacidades, Validado antes de Escribir (REQ-47.3)

La asignación DEBE calcularse en un único plan para el cambio entero (todas las capacidades del cambio SDD, o todas las capacidades del documento ODD), antes de escribir ningún fichero, de modo que dos capacidades del mismo cambio nunca reciban el mismo número y el resultado no dependa de componer por dominio. El plan DEBE validar las reglas de la delta con identificadores mezclados y rechazar sin escribir, con un error tipado que nombre la capacidad, la sección, el requisito y el identificador:

1. ADDED: admite un provisional o un identificador definitivo. Un ADDED con identificador definitivo DEBE seguir aceptándose con la comprobación de colisión vigente (REQ-30.4 de `odd-living-spec-integration`, REQ-29.3 de `sdd-iso-evidence`); el provisional NO es obligatorio.
2. MODIFIED y el nombre antiguo de un RENAMED, pendientes de integrar, cuyo nombre lleva un identificador válido según el patrón: DEBEN citar un identificador definitivo que exista, sin superar, en la especificación canónica de su capacidad. Si no existe, el error tipado es el de identificador inexistente. Un nombre sin identificador (solo admisible fuera de ISO; en ISO lo bloquea REQ-29.3) se resuelve por el título exacto, como fija REQ-30.22 de `sdd-archive-lifecycle`, y si no casa se rechaza con el motivo de composición vigente, no con el de identificador inexistente. El nombre nuevo de un RENAMED conserva el identificador del antiguo.
3. Lo ya integrado no está pendiente y no se rechaza (REQ-30.3 de `odd-living-spec-integration`): un REMOVED cuyo requisito ya no se resuelve en la canónica, un RENAMED cuyo nombre nuevo ya está y cuyo nombre antiguo ya no, y un MODIFIED cuyo requisito resuelto ya tiene el mismo cuerpo. `axiom sdd archive-compose`, que compone un único dominio sin clasificar lo ya integrado, sigue rechazando un REMOVED que no se resuelve con el error tipado de identificador inexistente (REQ-30.22).
4. Una capacidad sin especificación canónica cuya delta trae MODIFIED, REMOVED o RENAMED es un error tipado de capacidad inexistente, que nombra la capacidad, la sección y el requisito.
5. Un provisional en MODIFIED, REMOVED o RENAMED es un error tipado.
6. Un provisional citado sin un ADDED que lo declare en el mismo cambio es un error tipado, en los textos que REQ-47.4 somete a validación (la delta entera, los ficheros de tareas y el documento ODD).
7. Un provisional declarado dos veces en el cambio es un error tipado (REQ-47.1).
8. Un nombre de requisito con un provisional y cualquier otro identificador (otro provisional distinto o un identificador definitivo) es ambiguo y es un error tipado, también fuera de ISO. Un nombre con varios identificadores definitivos y ningún provisional conserva la regla vigente (REQ-30.22 toma el último; en ISO lo bloquea REQ-29.3).
9. Un token con la forma de un provisional que no cumple su gramática o cuyos componentes no casan con el patrón (por ejemplo, una serie fijada que no es una serie válida del patrón), una etiqueta de serie nueva con un patrón sin `<n>` (REQ-47.1) o cualquier asignación que no pueda formar un identificador válido (por ejemplo, un número que desborda) es un error tipado.
10. Un mapa guardado (REQ-47.6) ilegible, escrito con un patrón distinto del efectivo, o con una entrada que no casa con exactamente un ADDED de su capacidad, es un error tipado que nombra el fichero o la sección y la continuación (restaurar el mapa con git o rehacer la integración desde la base de la rama). El plan nunca lo ignora ni lo regenera en silencio.
11. Con el patrón inaccesible (bloque `sdd:` inválido o configuración ilegible) el plan no se evalúa con un patrón no declarado: la orden se niega sin escribir con la continuación `axiom workspace validate` (REQ-44.9).

El plan es una función de las deltas, de la spec viva, de las deltas de los cambios activos y del mapa guardado: no escribe nada y su resultado es el que consumen la integración, el archivado y las comprobaciones de solo lectura (REQ-47.7 y la opción de solo lectura de REQ-47.8), que DEBEN aplicar exactamente estas validaciones. Dos integraciones simultáneas en dos árboles de trabajo no se coordinan entre sí: su choque lo detecta REQ-47.5 tras el rebase.

#### Scenario: Dos capacidades del cambio no reciben el mismo número

- **DADO** el patrón por defecto y un cambio con dos capacidades existentes, cada una con un ADDED cuyo provisional lleva la serie `30` fijada, y `REQ-30.4` como máximo
- **CUANDO** se planifica la asignación
- **ENTONCES** los dos requisitos reciben `REQ-30.5` y `REQ-30.6`, sin repetirse

#### Scenario: Un MODIFIED con un identificador inexistente se rechaza

- **DADO** un cambio con un MODIFIED `Cobro con IVA (REQ-30.99)` y una spec viva sin `REQ-30.99`
- **CUANDO** se planifica la asignación
- **ENTONCES** se rechaza con el error tipado de identificador inexistente, que nombra la capacidad, la sección MODIFIED y `REQ-30.99`
- **Y** no se escribe nada

#### Scenario: El nombre antiguo de un RENAMED con un identificador inexistente se rechaza

- **DADO** un RENAMED `Cobro (REQ-30.99) → Cobro con IVA (REQ-30.99)` y una spec viva sin ningún requisito con `REQ-30.99` ni con el nombre nuevo
- **CUANDO** se planifica la asignación
- **ENTONCES** se rechaza con el error tipado de identificador inexistente, que nombra la capacidad, la sección RENAMED y `REQ-30.99`
- **Y** no se escribe nada

#### Scenario: Un REMOVED que ya no se resuelve es lo ya integrado

- **DADO** un REMOVED `Cobro (REQ-30.9)` y una spec viva sin ningún requisito vivo con `REQ-30.9`
- **CUANDO** se planifica la asignación de la integración o del paso de archivado del cambio
- **ENTONCES** el REMOVED se clasifica como ya aplicado y no se rechaza
- **Y** `axiom sdd archive-compose` con esa misma delta sigue rechazándolo con el error tipado de identificador inexistente

#### Scenario: Un MODIFIED sin identificador se resuelve por título fuera de ISO

- **DADO** un cambio no ISO, una canónica con `### Requirement: Cobro` y una delta con un MODIFIED `### Requirement: Cobro`
- **CUANDO** se planifica la asignación
- **ENTONCES** no se rechaza por falta de identificador
- **Y** un MODIFIED `### Requirement: Cobro con IVA`, que no casa con ningún título, se rechaza con el motivo de composición vigente que nombra la sección MODIFIED y el requisito, no con el de identificador inexistente

#### Scenario: Una capacidad nueva con MODIFIED, REMOVED o RENAMED se rechaza

- **DADO** un cambio con una capacidad `pagos` sin `openspec/specs/pagos/spec.md` cuya delta trae un MODIFIED `Cobro (REQ-30.1)`
- **CUANDO** se planifica la asignación
- **ENTONCES** se rechaza con el error tipado de capacidad inexistente, que nombra `pagos`, la sección MODIFIED y el requisito
- **Y** no se escribe nada, tampoco la capacidad nueva

#### Scenario: Un nombre con un provisional y otro identificador es ambiguo

- **DADO** un cambio no ISO con un ADDED `Cobro con IVA («prov-a», REQ-30.1)` y otro con `Devolución («prov-a», «prov-b»)`
- **CUANDO** se planifica la asignación
- **ENTONCES** se rechaza con un error tipado de nombre ambiguo que nombra el requisito y los dos identificadores
- **Y** no se escribe nada

#### Scenario: Una serie fijada que el patrón no admite se rechaza

- **DADO** el patrón `RF-<DOM>-<NN>` y un ADDED con un provisional cuya serie fijada no es un dominio válido del patrón
- **CUANDO** se planifica la asignación
- **ENTONCES** se rechaza con el error tipado de provisional mal formado, que nombra el requisito, el token y el patrón
- **Y** no se escribe nada

#### Scenario: Un mapa guardado ilegible o de otro patrón se rechaza

- **DADO** un cambio con provisionales cuyo mapa guardado (REQ-47.6) no se puede leer, o fue escrito con un patrón distinto del efectivo
- **CUANDO** se planifica la asignación
- **ENTONCES** se rechaza con un error tipado que nombra el mapa y la continuación
- **Y** no se escribe nada ni se regenera el mapa

#### Scenario: Con el patrón inaccesible el plan no se evalúa

- **DADO** un proyecto cuyo `axiom.yaml` tiene un bloque `sdd:` inválido y un cambio o un documento ODD con provisionales
- **CUANDO** se ejecuta la integración, el paso de archivado del cambio o cualquiera de sus comprobaciones de solo lectura
- **ENTONCES** finaliza con código distinto de cero con la continuación `axiom workspace validate`
- **Y** no se escribe nada

#### Scenario: Un MODIFIED con un identificador existente se acepta

- **DADO** un MODIFIED `Cobro con IVA (REQ-30.1)` y una spec viva con `Cobro (REQ-30.1)`
- **CUANDO** se planifica la asignación
- **ENTONCES** no se rechaza por el identificador y no se asigna ningún número a ese requisito

#### Scenario: Un provisional en MODIFIED, REMOVED o RENAMED se rechaza

- **DADO** un cambio con un MODIFIED cuyo nombre lleva `«prov-a»`
- **CUANDO** se planifica la asignación
- **ENTONCES** se rechaza con un error tipado que nombra la sección MODIFIED y `«prov-a»`
- **Y** lo mismo ocurre con un provisional en REMOVED o en cualquiera de los dos nombres de un RENAMED

#### Scenario: Un provisional citado sin ADDED que lo declare se rechaza

- **DADO** un cambio cuyo `tasks.md` contiene `Cubre: «prov-b»` y ningún ADDED declara `«prov-b»`
- **CUANDO** se planifica la asignación
- **ENTONCES** se rechaza con un error tipado que nombra `«prov-b»` y el fichero de tareas
- **Y** lo mismo ocurre con un provisional citado solo en el cuerpo de un requisito o en el texto del documento ODD

#### Scenario: Un ADDED con un identificador definitivo se sigue aceptando

- **DADO** un cambio con un ADDED `(REQ-31.1)` y una spec viva sin `REQ-31.1`
- **CUANDO** se planifica la asignación
- **ENTONCES** el ADDED se acepta con su identificador, sin asignación
- **Y** con `REQ-31.1` ya existente en la spec viva se rechaza con la colisión de REQ-30.4

#### Scenario: El plan no escribe nada

- **DADO** cualquier cambio, válido o inválido
- **CUANDO** se planifica la asignación
- **ENTONCES** ningún fichero del árbol cambia

---

### Requirement: Sustitución por Token Completo y Reescritura Atómica del Provisional en Todos sus Destinos (REQ-47.4)

La sustitución de un provisional por su identificador definitivo DEBE hacerse por token completo: coincidencia exacta y sensible a mayúsculas, sin estar precedida ni seguida de una letra, una cifra, `_` o `-`, de modo que un provisional que es prefijo de otro nunca lo corrompe ni toca un texto que solo se le parece. La sustitución DEBE aplicarse a todas las apariciones del token en estos destinos:

1. La spec viva compuesta (títulos y cuerpos de los requisitos añadidos y, en una capacidad nueva, el texto entero que se escribe).
2. Las líneas `Cubre:` de los ficheros de tareas del cambio (`tasks.md` y `tasks.<rol>.md`), conservando la posición del token en la línea y el resto del texto byte a byte.
3. El documento ODD: la valla `openspec` de `## Spec viva` y sus líneas `Cubre:`, sin tocar el resto del documento.
4. La delta del cambio, entera, antes de archivarla.
5. La columna del requisito de las filas de la matriz de verificación (REQ-29.6 de `sdd-iso-evidence`) de los informes `verify-report*.md` del cambio, conservando el resto del informe byte a byte, de modo que un archivado reanudado no compare definitivos de la delta con provisionales de la matriz.

Alcance de la validación y de la reescritura fuera de esos destinos: un provisional citado en el resto del texto de los ficheros de tareas o del documento ODD se valida (REQ-47.3, regla 6) pero NO se reescribe, porque el mapa guardado (REQ-47.6) explica la mención; la sección del documento ODD que guarda el mapa no se valida ni se reescribe. `proposal.md`, `design.md`, `explore.md` y los demás informes del cambio ni se validan ni se reescriben, de modo que la prosa de diseño puede citar provisionales de ejemplo. «Declarado» significa declarado por un ADDED del cambio o por una entrada del mapa guardado. No existe un mecanismo de escape: un texto validado que necesita mostrar la forma del provisional usa un marcador que no es un token (por ejemplo, con la serie y el ordinal entre ángulos).

Todas las escrituras DEBEN formar un único conjunto atómico: o se aplican todas o ninguna, y un fallo en cualquier punto NO DEBE dejar la spec viva, las líneas `Cubre:`, el documento ODD, las deltas, los informes de verify o el mapa a medias; las escrituras se hacen con ficheros temporales y renombrado, con restauración ante un fallo, igual que la integración ODD vigente. Si la propia restauración falla, la orden DEBE terminar con código distinto de cero nombrando cada fichero que no pudo restaurar y la continuación (revisarlo con `git status` y recuperarlo con `git checkout -- <ruta>` antes de repetir). Un temporal huérfano de una ejecución interrumpida por el sistema se sustituye y se retira en la siguiente ejecución que escribe ese destino. Un cambio sin provisionales NO DEBE modificar ni un byte de sus tareas, de su delta, de sus informes ni de su documento ODD.

#### Scenario: La sustitución respeta los límites del token

- **DADO** un texto con `«prov-a»`, `«prov-ab»` y la palabra `x«prov-a»` y un mapa que sustituye `«prov-a»` por `REQ-31.1`
- **CUANDO** se aplica la sustitución
- **ENTONCES** solo el token completo `«prov-a»` cambia
- **Y** `«prov-ab»` y el texto pegado a una letra quedan intactos

#### Scenario: La sustitución es sensible a mayúsculas

- **DADO** un texto con `«prov-a»` y `«PROV-A»`
- **CUANDO** se aplica la sustitución de `«prov-a»`
- **ENTONCES** solo se sustituye la forma exacta

#### Scenario: El `Cubre:` conserva su posición

- **DADO** una línea `Cubre: REQ-29.1, «prov-a», REQ-29.2` en un fichero de tareas
- **CUANDO** se asigna `«prov-a»` a `REQ-31.1`
- **ENTONCES** la línea queda `Cubre: REQ-29.1, REQ-31.1, REQ-29.2`
- **Y** el resto del fichero es idéntico byte a byte

#### Scenario: Un fallo en la reescritura no deja nada a medias

- **DADO** un cambio con provisionales y un fallo inyectado al escribir el último destino del conjunto
- **CUANDO** se ejecuta la integración o el archivado
- **ENTONCES** la spec viva, las líneas `Cubre:`, el documento ODD y las deltas conservan sus bytes originales
- **Y** no queda ningún fichero temporal en el árbol

#### Scenario: Una asignación inválida no escribe ningún destino

- **DADO** un cambio con un provisional válido y, en otra capacidad, un MODIFIED con un identificador inexistente
- **CUANDO** se ejecuta la integración o el archivado
- **ENTONCES** finaliza con el error tipado y ningún destino cambia, tampoco la capacidad con el provisional válido

#### Scenario: Sin provisionales no se reescribe nada

- **DADO** un cambio cuyos ADDED llevan identificadores definitivos y sin ningún provisional
- **CUANDO** se integra o se archiva
- **ENTONCES** los ficheros de tareas, la delta, los informes de verify y el documento ODD conservan sus bytes

#### Scenario: La matriz de verificación se reescribe al archivar

- **DADO** un cambio ISO con un ADDED `Cobro con IVA («prov-a»)` y un `verify-report.md` cuya matriz tiene una fila de un escenario de ese requisito que cita `«prov-a»`
- **CUANDO** el paso de archivado del cambio asigna `REQ-31.2` a `«prov-a»`
- **ENTONCES** la fila de la matriz cita `REQ-31.2`
- **Y** el resto del informe conserva sus bytes
- **Y** una reanudación del archivado no bloquea por la matriz

#### Scenario: La prosa de las tareas se valida pero no se reescribe

- **DADO** un `tasks.md` cuya descripción de una tarea menciona `«prov-a»` fuera de una línea `Cubre:` y un ADDED que declara `«prov-a»`
- **CUANDO** se integra el cambio
- **ENTONCES** la mención en prosa conserva `«prov-a»`
- **Y** el mapa guardado permite saber qué definitivo recibió

#### Scenario: La prosa de diseño no se valida

- **DADO** un `design.md` que cita como ejemplo un provisional que ningún ADDED declara
- **CUANDO** se planifica la asignación
- **ENTONCES** no se rechaza por esa cita
- **Y** `design.md` conserva sus bytes

#### Scenario: Un fallo al restaurar se nombra

- **DADO** un cambio con provisionales, un fallo inyectado al renombrar el último destino y un fallo al restaurar uno de los ya renombrados
- **CUANDO** se ejecuta la integración o el archivado
- **ENTONCES** finaliza con código distinto de cero nombrando el fichero que no pudo restaurar y la continuación con `git checkout -- <ruta>`

---

### Requirement: Detección de Identificadores Duplicados entre Capacidades de la Spec Viva (REQ-47.5)

La integración (archivado SDD y `axiom archive integrate --odd`) y la comprobación de solo lectura (REQ-47.7) DEBEN detectar un identificador definitivo usado por requisitos de capacidades distintas de la spec viva, de modo que una colisión entre ramas concurrentes que git no marca como conflicto (por ejemplo, la misma serie nueva abierta en dos capacidades distintas) no pase en silencio. La detección DEBE bloquear, sin escribir, cuando el duplicado afecta a un identificador que la propia integración escribe (asignado, reasignado o ADDED con identificador definitivo), con un error tipado que nombre el identificador, las dos capacidades y la continuación (reintegrar tras el rebase, REQ-30.3). Un duplicado previo que no involucra ningún identificador que escriba la integración NO DEBE bloquearla y se informa como dato, sin cambiar el código de salida.

Supuesto de esta spec: la comprobación se limita a los duplicados que la integración introduce o toca, para que un duplicado histórico no relacionado no impida integrar; el diseño puede ampliarlo a un informe completo mientras no bloquee por duplicados ajenos.

#### Scenario: La misma serie nueva en dos capacidades se detecta

- **DADO** dos ramas que abrieron `REQ-47.1` cada una en una capacidad distinta, fusionadas en `main` sin conflicto de git
- **CUANDO** se ejecuta la integración o la comprobación de solo lectura en la segunda rama tras el rebase
- **ENTONCES** se rechaza con un error tipado que nombra `REQ-47.1` y las dos capacidades
- **Y** no se escribe nada

#### Scenario: Un duplicado ajeno a la integración no la bloquea

- **DADO** una spec viva con un identificador histórico repetido en dos capacidades, que la integración en curso no escribe
- **CUANDO** se ejecuta la integración
- **ENTONCES** termina con éxito
- **Y** informa del duplicado histórico sin cambiar el código de salida

#### Scenario: Un ADDED con identificador definitivo duplicado entre capacidades se rechaza

- **DADO** un ADDED `(REQ-31.1)` en una capacidad y un `REQ-31.1` ya existente en otra capacidad de la spec viva
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza con la colisión nombrando `REQ-31.1` y la capacidad donde ya existe

---

### Requirement: Traza Legible del Mapa de Provisional a Definitivo tras Archivar (REQ-47.6)

Toda integración que asigne identificadores DEBE dejar el mapa de provisional a definitivo, con la capacidad de cada requisito, como traza legible: (a) en el cambio SDD archivado, dentro de `openspec/changes/archive/<fecha>-<cambio>/`; (b) en el documento ODD integrado; y (c) en el informe `axiom.odd-integration/v1` de la integración ODD y en el informe del paso de archivado SDD (REQ-47.8). El mapa DEBE incluir, por cada entrada, el provisional, el identificador definitivo y la capacidad, y DEBE poder leerse después de archivar sin reconstruirlo de otro modo. El mapa guardado es además el dato que usa la reintegración idempotente (REQ-30.3) y `axiom sdd archive-check` para reconocer una asignación ya hecha. Una integración sin provisionales NO DEBE escribir un mapa vacío. NO se reescriben las trazas de cambios archivados antes de este cambio. En el documento ODD el mapa ocupa una única sección propia, distinta de `## Spec viva` y de `## Impacto en la spec`, de modo que no altera la validación de esas secciones (REQ-30.1 y REQ-30.5 de `odd-living-spec-integration`) ni la lectura de sus vallas `openspec`; un documento con dos secciones del mapa se rechaza con un error tipado. Un mapa ilegible o incoherente se rige por la regla 10 de REQ-47.3.

#### Scenario: La traza de un cambio SDD es legible tras archivar

- **DADO** un cambio SDD con un ADDED provisional en la capacidad `facturacion` que se archiva con éxito
- **CUANDO** se lee la carpeta archivada del cambio
- **ENTONCES** contiene una entrada con el provisional, el identificador definitivo asignado y la capacidad `facturacion`

#### Scenario: La traza ODD consta en el documento y en el informe

- **DADO** un documento ODD con un ADDED provisional integrado con éxito
- **CUANDO** se lee el documento ODD y el informe `axiom.odd-integration/v1`
- **ENTONCES** ambos contienen la entrada con el provisional, el definitivo y la capacidad

#### Scenario: Sin provisionales no hay traza

- **DADO** una integración cuyos ADDED llevan identificadores definitivos
- **CUANDO** termina
- **ENTONCES** no se escribe ningún mapa y el informe no lleva entradas de asignación

#### Scenario: Las trazas anteriores no se reescriben

- **DADO** el cambio archivado `carril-sdd-versatil`, cuyo `archive-report.md` cita `REQ-28.x`
- **CUANDO** se ejecuta la integración o el archivado de cualquier cambio
- **ENTONCES** ese informe conserva sus bytes

#### Scenario: La sección del mapa no altera la validación del documento ODD

- **DADO** un documento ODD ya integrado, con su sección del mapa al final, una `## Spec viva` válida y una `## Impacto en la spec` ausente
- **CUANDO** se vuelve a validar el documento
- **ENTONCES** las secciones `## Spec viva` y `## Impacto en la spec` se validan como antes de añadir el mapa
- **Y** las vallas `openspec` de cada capacidad se leen igual

---

### Requirement: Las Guías de las Fases Describen el Identificador Provisional y el Paso de Archivado (REQ-47.10)

Las guías que Axiom distribuye DEBEN describir el identificador provisional y su asignación de forma coherente con REQ-47.1 a REQ-47.8, y su contenido DEBE estar fijado por pruebas de texto:

1. La guía de la fase `sdd-spec` DEBE indicar que un requisito ADDED lleva un identificador provisional con el formato literal que fija el diseño, que un identificador definitivo sigue admitiéndose con la comprobación de colisión, que el provisional solo es válido en ADDED, que el ordinal del provisional no es el número que se asignará, y que MODIFIED, REMOVED y el nombre antiguo de un RENAMED citan el identificador definitivo existente.
2. La guía de la fase `sdd-archive` DEBE sustituir la composición por dominio y la copia de las capacidades nuevas por una única invocación del paso de archivado del cambio (REQ-47.8), ejecutada después de `axiom sdd archive-check` y antes de la instantánea y del movimiento de la carpeta; DEBE incluir en el informe de archivado la tabla del mapa de provisional a definitivo; y DEBE ordenar la comprobación de solo lectura del paso tras cada rebase de la rama de archivado y antes de pedir la fusión. NO DEBE conservar la copia de una capacidad nueva con la shell.
3. La convención ISO compartida (`_shared/sdd-iso-convention.md`) DEBE indicar que un ADDED puede llevar un provisional, que `Cubre:` y la matriz de verificación lo citan hasta archivar, y que el paso de archivado los reescribe al definitivo.
4. El contrato de status (`_shared/sdd-status-contract.md`) se rige por REQ-47.9.

Las instrucciones de enrutamiento se rigen por REQ-24.58 de `organic-agent-trigger-rules`. Los renderizados de referencia (goldens) que contengan alguno de estos textos DEBEN regenerarse en el mismo cambio y seguir siendo deterministas.

#### Scenario: La guía de `sdd-spec` describe el provisional

- **DADO** la guía distribuida de la fase `sdd-spec`
- **CUANDO** se inspecciona su texto (pin de texto)
- **ENTONCES** indica el formato literal del provisional, que solo vale en ADDED, que un definitivo sigue admitiéndose y que el ordinal no es el número asignado

#### Scenario: La guía de `sdd-archive` usa el paso de archivado del cambio

- **DADO** la guía distribuida de la fase `sdd-archive`
- **CUANDO** se inspecciona su texto (pin de texto)
- **ENTONCES** invoca el paso de archivado del cambio una sola vez, después de `axiom sdd archive-check` y antes de la instantánea y del movimiento
- **Y** no contiene la copia con la shell de una capacidad nueva a `openspec/specs/`
- **Y** ordena la comprobación de solo lectura tras cada rebase de la rama de archivado e incluye el mapa en el informe

#### Scenario: La convención ISO admite el provisional hasta archivar

- **DADO** la convención ISO compartida
- **CUANDO** se inspecciona su texto (pin de texto)
- **ENTONCES** indica que el ADDED puede llevar un provisional, que `Cubre:` y la matriz lo citan, y que el archivado los reescribe
