# Delta para requirement-identifiers

Este delta añade a la capacidad los identificadores provisionales (C-03-b): su forma, la asignación del siguiente número libre al integrar (incluida la apertura de una serie), el plan único de asignación con sus errores tipados, la sustitución por token completo, la reescritura atómica, la detección de duplicados entre capacidades, la traza y lo que dicen las guías de las fases (REQ-47.10).

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
