# Especificación de requirement-identifiers

## Purpose

Puerto único de identificadores de requisito (C-03-a): el proyecto declara en el bloque `sdd:` de `axiom.yaml` el patrón de sus identificadores (`RF-USU-01`, `REQ-30.4`...), y un único punto del producto decide qué es un identificador válido, cuál es su serie y su número, y cuál es el siguiente libre de una serie. Los consumidores de la forma del identificador (compositor de OpenSpec, evidencia ISO, spec viva, documentación viva y mensajes de la CLI) usan ese punto y ninguno conserva una expresión regular propia; la única excepción deliberada y temporal es `adoptcheck` con `archive adopt` (REQ-44.4). Sin declaración rige un valor por defecto que admite exactamente los `REQ-<n>.<m>` vigentes, de modo que un proyecto que no declara nada no observa ningún cambio. Esta capacidad no asigna identificadores provisionales ni definitivos (C-03-b), no conserva los identificadores de un cliente al adoptar (C-03-c), no traza identificadores (C-03-d) ni migra las specs de Axiom (C-03-e).

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
