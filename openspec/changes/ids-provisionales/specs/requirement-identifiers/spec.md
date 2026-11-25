# Delta para requirement-identifiers

Este delta añade a la capacidad los identificadores provisionales (C-03-b): su forma, la asignación del siguiente número libre al integrar (incluida la apertura de una serie), el plan único de asignación con sus errores tipados, la sustitución por token completo, la reescritura atómica, la detección de duplicados entre capacidades y la traza.

Notación de los escenarios: el formato literal del identificador provisional lo fija el diseño. Aquí un provisional se escribe con los marcadores neutros `«prov-a»`, `«prov-b»`, `«prov-c»`; son notación de esta spec y NO el formato real. Cuando un escenario necesita fijar la serie, lo dice con palabras («un provisional con la serie `30` fijada», «con la serie `USU` fijada», «con una etiqueta de serie nueva»). Las propiedades exigidas son las del requisito REQ-47.1.

Nota sobre el texto vivo: la sección `## Purpose` de la spec viva afirma que esta capacidad «no asigna identificadores provisionales ni definitivos (C-03-b)». Un delta no puede modificar esa sección; queda obsoleta al archivar y debe corregirse a mano en la fase de archivo (véase el informe de la fase). REQ-44.3 no se modifica: su texto no excluye abrir una serie vacía (el máximo de una serie sin identificadores es cero), de modo que REQ-47.2 lo completa sin contradecirlo.

## ADDED Requirements

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

---

### Requirement: Asignación del Siguiente Número Libre al Integrar, Incluida la Apertura de una Serie (REQ-47.2)

Al integrar (archivado SDD, REQ-47.8, o `axiom archive integrate --odd`), el sistema DEBE sustituir cada provisional por un identificador definitivo válido según el patrón efectivo del proyecto. El número es el siguiente libre de la serie, calculado por el patrón (REQ-44.3) como el máximo de la serie en la spec viva de la rama más uno, formateado con el relleno del patrón y sin truncar. La asignación DEBE abrir una serie vacía: cuando la serie no tiene ningún identificador en la spec viva, el primer número es `1` (con el patrón por defecto, `REQ-<n>.1`; con `RF-<DOM>-<NN>`, `RF-<DOM>-01`).

Cuando el provisional lleva una etiqueta de serie nueva (patrón con `<n>`), el sistema DEBE asignarle el siguiente número de serie libre, igual al máximo de los números de serie presentes en la spec viva más uno, y todos los ADDED con la misma etiqueta reciben esa misma serie, con números de secuencia consecutivos desde `1`. Distintas etiquetas de serie nueva reciben series distintas.

La asignación DEBE ser determinista: las mismas entradas (las deltas del cambio y la spec viva) producen siempre el mismo mapa de provisional a definitivo. Los números de una misma serie se asignan en el orden de aparición de los ADDED en el cambio, con un orden fijo y reproducible de las capacidades. El rango de números NO DEBE reservar ni usar valores altos «de cortesía» que puedan confundirse con un identificador válido real (el provisional ya es inconfundible por REQ-47.1).

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

---

### Requirement: Plan Único de Asignación sobre Todas las Capacidades, Validado antes de Escribir (REQ-47.3)

La asignación DEBE calcularse en un único plan para el cambio entero (todas las capacidades del cambio SDD, o todas las capacidades del documento ODD), antes de escribir ningún fichero, de modo que dos capacidades del mismo cambio nunca reciban el mismo número y el resultado no dependa de componer por dominio. El plan DEBE validar las reglas de la delta con identificadores mezclados y rechazar sin escribir, con un error tipado que nombre la capacidad, la sección, el requisito y el identificador:

1. ADDED: admite un provisional o un identificador definitivo. Un ADDED con identificador definitivo DEBE seguir aceptándose con la comprobación de colisión vigente (REQ-30.4 de `odd-living-spec-integration`, REQ-29.3 de `sdd-iso-evidence`); el provisional NO es obligatorio.
2. MODIFIED, REMOVED y el nombre antiguo de un RENAMED: DEBEN citar un identificador definitivo que exista en `openspec/specs/`. Si no existe, el error tipado es el de identificador inexistente. El nombre nuevo de un RENAMED conserva el identificador del antiguo.
3. Un provisional en MODIFIED, REMOVED o RENAMED es un error tipado.
4. Un provisional citado en una línea `Cubre:` o en el texto del cambio (cuerpos de requisito, ficheros de tareas, documento ODD) sin un ADDED que lo declare en el mismo cambio es un error tipado.
5. Un provisional declarado dos veces en el cambio es un error tipado (REQ-47.1).
6. Una etiqueta de serie nueva con un patrón sin `<n>` (REQ-47.1) o cualquier asignación que no pueda formar un identificador válido (por ejemplo, un número que desborda) es un error tipado.

El plan es una función de las deltas y de la spec viva: no escribe nada y su resultado es el que consumen la integración, el archivado y la comprobación de solo lectura (REQ-47.7).

#### Scenario: Dos capacidades del cambio no reciben el mismo número

- **DADO** el patrón por defecto y un cambio con dos capacidades existentes, cada una con un ADDED cuyo provisional lleva la serie `30` fijada, y `REQ-30.4` como máximo
- **CUANDO** se planifica la asignación
- **ENTONCES** los dos requisitos reciben `REQ-30.5` y `REQ-30.6`, sin repetirse

#### Scenario: Un MODIFIED con un identificador inexistente se rechaza

- **DADO** un cambio con un MODIFIED `Cobro con IVA (REQ-30.99)` y una spec viva sin `REQ-30.99`
- **CUANDO** se planifica la asignación
- **ENTONCES** se rechaza con el error tipado de identificador inexistente, que nombra la capacidad, la sección MODIFIED y `REQ-30.99`
- **Y** no se escribe nada

#### Scenario: Un REMOVED y el nombre antiguo de un RENAMED con un identificador inexistente se rechazan

- **DADO** un REMOVED `Cobro (REQ-30.99)` y, en otro cambio, un RENAMED `Cobro (REQ-30.99) → Cobro con IVA (REQ-30.99)`, sin `REQ-30.99` en la spec viva
- **CUANDO** se planifica la asignación de cada uno
- **ENTONCES** ambos se rechazan con el error tipado de identificador inexistente
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

1. La spec viva compuesta (títulos y cuerpos de los requisitos añadidos).
2. Las líneas `Cubre:` de los ficheros de tareas del cambio, conservando la posición del token en la línea y el resto del texto byte a byte.
3. El documento ODD: la valla `openspec` de `## Spec viva` y sus líneas `Cubre:`, sin tocar el resto del documento.
4. La delta del cambio, antes de archivarla.

Todas las escrituras DEBEN formar un único conjunto atómico: o se aplican todas o ninguna, y un fallo en cualquier punto NO DEBE dejar la spec viva, las líneas `Cubre:`, el documento ODD o las deltas a medias; las escrituras se hacen con ficheros temporales y renombrado, con restauración ante un fallo, igual que la integración ODD vigente. Un cambio sin provisionales NO DEBE modificar ni un byte de sus tareas, de su delta ni de su documento ODD.

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
- **ENTONCES** los ficheros de tareas, la delta y el documento ODD conservan sus bytes

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

Toda integración que asigne identificadores DEBE dejar el mapa de provisional a definitivo, con la capacidad de cada requisito, como traza legible: (a) en el cambio SDD archivado, dentro de `openspec/changes/archive/<fecha>-<cambio>/`; (b) en el documento ODD integrado; y (c) en el informe `axiom.odd-integration/v1` de la integración ODD y en el informe del paso de archivado SDD (REQ-47.8). El mapa DEBE incluir, por cada entrada, el provisional, el identificador definitivo y la capacidad, y DEBE poder leerse después de archivar sin reconstruirlo de otro modo. El mapa guardado es además el dato que usa la reintegración idempotente (REQ-30.3) y `axiom sdd archive-check` para reconocer una asignación ya hecha. Una integración sin provisionales NO DEBE escribir un mapa vacío. NO se reescriben las trazas de cambios archivados antes de este cambio.

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
